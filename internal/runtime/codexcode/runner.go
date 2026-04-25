package codexcode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
	"github.com/GrinRus/ProvenArch/internal/runtime/providercommon"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
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
	if err := providercommon.ValidateRuntimeArtifacts(task, acpruntime.ProviderCodexCode); err != nil {
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
	go func() {
		errCh <- providercommon.CaptureCommandStream(stdoutPipe, &stdout, task, acpruntime.OutputStreamStdout)
	}()
	go func() {
		errCh <- providercommon.CaptureCommandStream(stderrPipe, &stderr, task, acpruntime.OutputStreamStderr)
	}()
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

func wrapCommandFailure(task acpruntime.Task, stdout string, stderr string, cause error) error {
	message, rawOutputRefs := providercommon.BuildFailureMessage(
		acpruntime.ProviderCodexCode,
		task,
		"exec",
		cause,
		stdout,
		stderr,
		map[string]any{
			"current_step":      strings.TrimSpace(task.StepID),
			"last_stdout_bytes": len([]byte(stdout)),
			"last_stderr_bytes": len([]byte(stderr)),
		},
	)
	if errors.Is(cause, context.DeadlineExceeded) {
		return acpruntime.WrapRunnerErrorWithDiagnostics(acpruntime.ProviderCodexCode, acpruntime.ErrorCodeRuntimeTimeout, message, stdout, stderr, rawOutputRefs, cause)
	}
	if errors.Is(cause, context.Canceled) {
		return acpruntime.WrapRunnerErrorWithDiagnostics(acpruntime.ProviderCodexCode, acpruntime.ErrorCodeRunCanceled, message, stdout, stderr, rawOutputRefs, cause)
	}
	return acpruntime.WrapRunnerErrorWithDiagnostics(acpruntime.ProviderCodexCode, acpruntime.ErrorCodeRunnerUnavailable, message, stdout, stderr, rawOutputRefs, cause)
}

func wrapContractFailure(task acpruntime.Task, stdout string, stderr string, cause error) error {
	return providercommon.WrapContractFailure(acpruntime.ProviderCodexCode, task, stdout, stderr, cause)
}
