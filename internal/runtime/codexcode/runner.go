package codexcode

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

var ErrRunnerUnavailable = errors.New("codex-code runner is unavailable")

type HeadlessRunner struct {
	Command string
	Args    []string
}

func (r HeadlessRunner) commandName() string {
	command := strings.TrimSpace(r.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv("ACP_CODEX_CMD"))
	}
	if command == "" {
		command = "codex"
	}
	return command
}

func (r HeadlessRunner) Preflight(_ context.Context) error {
	command := r.commandName()
	if _, err := exec.LookPath(command); err != nil {
		return acpruntime.WrapRunnerError(
			acpruntime.ProviderCodexCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command %q is unavailable: %v", acpruntime.ProviderCodexCode, command, err),
			err,
		)
	}
	return nil
}

func (r HeadlessRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if err := r.Preflight(ctx); err != nil {
		return acpruntime.Result{}, err
	}
	command := r.commandName()
	stdout, stderr, runErr := runCodexCommand(ctx, task, command, r.Args)
	if runErr != nil {
		return acpruntime.Result{}, wrapCommandFailure(task, stdout, stderr, runErr)
	}
	if err := validateRuntimeArtifacts(task); err != nil {
		return acpruntime.Result{}, wrapContractFailure(task, stdout, stderr, err)
	}
	return acpruntime.Result{
		Execution: acpruntime.NewExecution(task, acpruntime.ProviderCodexCode, "headless", "succeeded", time.Now().UTC(), nil),
		Stdout:    stdout,
		Stderr:    stderr,
	}, nil
}

func runCodexCommand(ctx context.Context, task acpruntime.Task, command string, args []string) (string, string, error) {
	commandArgs := append([]string(nil), args...)
	if len(commandArgs) == 0 {
		commandArgs = buildDefaultCodexArgs(task)
	}

	cmd := exec.CommandContext(ctx, command, commandArgs...)
	if workDir := strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)); workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Stdin = strings.NewReader(buildPrompt(task))

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", "", err
	}

	if err := cmd.Start(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", "", ctxErr
		}
		return "", "", err
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	errCh := make(chan error, 2)
	go func() { errCh <- captureCommandStream(stdoutPipe, &stdout, task, acpruntime.OutputStreamStdout) }()
	go func() { errCh <- captureCommandStream(stderrPipe, &stderr, task, acpruntime.OutputStreamStderr) }()
	for i := 0; i < 2; i++ {
		if streamErr := <-errCh; streamErr != nil {
			_ = cmd.Wait()
			return stdout.String(), stderr.String(), streamErr
		}
	}
	if err := cmd.Wait(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return stdout.String(), stderr.String(), ctxErr
		}
		return stdout.String(), stderr.String(), runnerdiag.BuildExecFailure(err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String(), nil
}

func buildDefaultCodexArgs(task acpruntime.Task) []string {
	cwd := strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task))
	args := []string{
		"exec",
		"--json",
		"--color", "never",
		"--skip-git-repo-check",
		"--sandbox", "danger-full-access",
	}
	if cwd != "" {
		args = append(args, "--cd", cwd)
	}
	for _, dir := range acpruntime.ResolveHeadlessIncludeDirectories(task) {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "--ephemeral", "-")
	return args
}

func buildPrompt(task acpruntime.Task) string {
	return promptcontract.ComposeArtifactOnlyPrompt(acpruntime.ProviderCodexCode, task)
}

func captureCommandStream(reader io.Reader, sink *bytes.Buffer, task acpruntime.Task, stream acpruntime.OutputStream) error {
	if sink == nil {
		return errors.New("capture sink is nil")
	}
	bufReader := bufio.NewReader(reader)
	for {
		part, err := bufReader.ReadString('\n')
		if len(part) > 0 {
			sink.WriteString(part)
			if task.OnOutput != nil {
				task.OnOutput(acpruntime.OutputChunk{
					Stream: stream,
					Text:   strings.TrimRight(part, "\r\n"),
				})
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func validateRuntimeArtifacts(task acpruntime.Task) error {
	switch acpruntime.StepProviderKeyForStepID(task.StepID) {
	case acpruntime.StepProviderStep1Collect:
		return validateCollectArtifacts(task)
	case acpruntime.StepProviderStep3Findings:
		return validateValidatorArtifacts(task)
	default:
		if runtimedrafts.IsDraftStep(task.StepID) {
			return validateDraftArtifacts(task)
		}
		return nil
	}
}

func validateCollectArtifacts(task acpruntime.Task) error {
	report, err := artifactquality.RepairCollectManifest(task)
	if err != nil {
		return err
	}
	emitCollectRepairDiagnostic(task, report)
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(task.WriteRoot), "shard-pack-manifest.json"))
	if err != nil {
		return err
	}
	if _, err := contracts.ParseShardPackManifest(raw); err != nil {
		return err
	}
	return nil
}

func emitCollectRepairDiagnostic(task acpruntime.Task, report artifactquality.RepairReport) {
	if task.OnDiagnostic == nil || len(report.AppliedRuleIDs) == 0 {
		return
	}
	task.OnDiagnostic(acpruntime.DiagnosticEvent{
		Message: "collect compatibility repair applied",
		Fields: map[string]any{
			"provider":         string(acpruntime.ProviderCodexCode),
			"changed":          report.Changed,
			"applied_rule_ids": append([]string(nil), report.AppliedRuleIDs...),
		},
	})
}

func validateDraftArtifacts(task acpruntime.Task) error {
	if _, _, err := validateRequiredRuntimeDraftArtifacts(task); err == nil {
		return nil
	}
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	if manifestFile == "" {
		return fmt.Errorf("draft manifest file is undefined for %s", task.StepID)
	}
	manifest, _, loadErr := runtimedrafts.Load(task.WriteRoot, manifestFile)
	if loadErr != nil {
		return loadErr
	}
	if err := runtimedrafts.ValidateManifestForTask(manifest, task.RunID, task.StepID, task.StepContract); err != nil {
		return err
	}
	if _, err := runtimedrafts.ReconcileOutputsAtDraftRoot(task.DraftFinalRoot, manifest); err != nil {
		return err
	}
	_, _, err := validateRequiredRuntimeDraftArtifacts(task)
	return err
}

func validateRequiredRuntimeDraftArtifacts(task acpruntime.Task) (runtimedrafts.Manifest, []byte, error) {
	return runtimedrafts.ValidateRequiredManifest(
		task.WriteRoot,
		task.DraftFinalRoot,
		task.RunID,
		task.StepID,
		task.StepContract,
		task.ExpectedArtifacts,
	)
}

func validateValidatorArtifacts(task acpruntime.Task) error {
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(task.WriteRoot), "validator-verdict.json"))
	if err != nil {
		return err
	}
	_, err = contracts.ParseValidatorVerdict(raw)
	return err
}

func wrapCommandFailure(task acpruntime.Task, stdout string, stderr string, cause error) error {
	if errors.Is(cause, context.DeadlineExceeded) {
		message, rawOutputRefs := buildFailureMessage(task, "exec", cause, stdout, stderr)
		return acpruntime.WrapRunnerErrorWithDiagnostics(acpruntime.ProviderCodexCode, acpruntime.ErrorCodeRuntimeTimeout, message, stdout, stderr, rawOutputRefs, cause)
	}
	if errors.Is(cause, context.Canceled) {
		message, rawOutputRefs := buildFailureMessage(task, "exec", cause, stdout, stderr)
		return acpruntime.WrapRunnerErrorWithDiagnostics(acpruntime.ProviderCodexCode, acpruntime.ErrorCodeRunCanceled, message, stdout, stderr, rawOutputRefs, cause)
	}
	message, rawOutputRefs := buildFailureMessage(task, "exec", cause, stdout, stderr)
	return acpruntime.WrapRunnerErrorWithDiagnostics(acpruntime.ProviderCodexCode, acpruntime.ErrorCodeRunnerUnavailable, message, stdout, stderr, rawOutputRefs, cause)
}

func wrapContractFailure(task acpruntime.Task, stdout string, stderr string, cause error) error {
	message, rawOutputRefs := buildFailureMessage(task, "contract", cause, stdout, stderr)
	return acpruntime.WrapRunnerErrorWithDiagnostics(acpruntime.ProviderCodexCode, acpruntime.ErrorCodeRuntimeContract, message, stdout, stderr, rawOutputRefs, cause)
}

func buildFailureMessage(task acpruntime.Task, stage string, cause error, stdout string, stderr string) (string, contracts.RuntimeOutputRefs) {
	base := "unknown failure"
	if cause != nil {
		base = strings.TrimSpace(cause.Error())
	}
	artifacts, err := runnerdiag.WriteFailureArtifacts(task, acpruntime.ProviderCodexCode, stdout, stderr)
	if err != nil {
		return fmt.Sprintf("stage=%s %s (raw_output_persist_failed=%v)", stage, base, err), contracts.RuntimeOutputRefs{}
	}
	return fmt.Sprintf(
			"stage=%s %s (raw_output=%s stdout_bytes=%d stdout_sha256=%s stderr_bytes=%d stderr_sha256=%s)",
			stage,
			base,
			artifacts.RelativeMetadataPath,
			artifacts.Stdout.Bytes,
			artifacts.Stdout.SHA256,
			artifacts.Stderr.Bytes,
			artifacts.Stderr.SHA256,
		), contracts.RuntimeOutputRefs{
			Stdout:   artifacts.Stdout.RelativePath,
			Stderr:   artifacts.Stderr.RelativePath,
			Metadata: artifacts.RelativeMetadataPath,
		}
}
