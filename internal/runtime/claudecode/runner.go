package claudecode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

var ErrRunnerUnavailable = errors.New("claude-code runner is unavailable")

type HeadlessRunner struct {
	Command string
	Args    []string
}

func (r HeadlessRunner) commandName() string {
	command := strings.TrimSpace(r.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv("ACP_CLAUDE_CMD"))
	}
	if command == "" {
		command = "claude-code"
	}
	return command
}

func (r HeadlessRunner) Preflight(_ context.Context) error {
	command := r.commandName()
	if _, err := exec.LookPath(command); err != nil {
		return acpruntime.WrapRunnerError(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command %q is unavailable: %v", acpruntime.ProviderClaudeCode, command, err),
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
	stdout, stderr, runErr := runClaudeCommand(ctx, task, command, r.Args)
	if runErr != nil {
		return acpruntime.Result{}, wrapCommandFailure(acpruntime.ProviderClaudeCode, task, stdout, stderr, runErr)
	}
	if err := validateRuntimeArtifacts(task); err != nil {
		return acpruntime.Result{}, wrapContractFailure(acpruntime.ProviderClaudeCode, task, stdout, stderr, err)
	}
	return acpruntime.Result{
		Execution: acpruntime.NewExecution(task, acpruntime.ProviderClaudeCode, "headless", "succeeded", time.Now().UTC(), nil),
		Stdout:    stdout,
		Stderr:    stderr,
	}, nil
}

func runClaudeCommand(ctx context.Context, task acpruntime.Task, command string, args []string) (string, string, error) {
	taskPayload, err := json.Marshal(task)
	if err != nil {
		return "", "", fmt.Errorf("marshal runtime task: %w", err)
	}
	commandArgs := append([]string(nil), args...)
	if len(commandArgs) == 0 {
		commandArgs = buildDefaultClaudeArgs(task, buildPrompt(task))
	}

	cmd := exec.CommandContext(ctx, command, commandArgs...)
	if workspace := strings.TrimSpace(task.Workspace); workspace != "" {
		cmd.Dir = workspace
	} else if writeRoot := strings.TrimSpace(task.WriteRoot); writeRoot != "" {
		cmd.Dir = writeRoot
	}
	cmd.Stdin = bytes.NewReader(taskPayload)

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

func buildDefaultClaudeArgs(task acpruntime.Task, prompt string) []string {
	args := []string{"--output-format", "json", "--permission-mode", "bypassPermissions"}
	for _, dir := range acpruntime.ResolveHeadlessIncludeDirectories(task) {
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "-p", prompt)
	return args
}

func buildPrompt(task acpruntime.Task) string {
	return promptcontract.ComposeArtifactOnlyPrompt(acpruntime.ProviderClaudeCode, task)
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
	if err := artifactquality.RepairCollectManifest(task); err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(task.WriteRoot), shardPackManifestFileName))
	if err != nil {
		return err
	}
	if _, err := contracts.ParseShardPackManifest(raw); err != nil {
		return err
	}
	return nil
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
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(task.WriteRoot), validatorVerdictFileName))
	if err != nil {
		return err
	}
	_, err = contracts.ParseValidatorVerdict(raw)
	return err
}

func wrapCommandFailure(provider acpruntime.Provider, task acpruntime.Task, stdout string, stderr string, cause error) error {
	if errors.Is(cause, context.DeadlineExceeded) {
		message, rawOutputRefs := buildFailureMessage(provider, task, "exec", cause, stdout, stderr)
		return acpruntime.WrapRunnerErrorWithDiagnostics(provider, acpruntime.ErrorCodeRuntimeTimeout, message, stdout, stderr, rawOutputRefs, cause)
	}
	if errors.Is(cause, context.Canceled) {
		message, rawOutputRefs := buildFailureMessage(provider, task, "exec", cause, stdout, stderr)
		return acpruntime.WrapRunnerErrorWithDiagnostics(provider, acpruntime.ErrorCodeRunCanceled, message, stdout, stderr, rawOutputRefs, cause)
	}
	message, rawOutputRefs := buildFailureMessage(provider, task, "exec", cause, stdout, stderr)
	return acpruntime.WrapRunnerErrorWithDiagnostics(provider, acpruntime.ErrorCodeRunnerUnavailable, message, stdout, stderr, rawOutputRefs, cause)
}

func wrapContractFailure(provider acpruntime.Provider, task acpruntime.Task, stdout string, stderr string, cause error) error {
	message, rawOutputRefs := buildFailureMessage(provider, task, "contract", cause, stdout, stderr)
	return acpruntime.WrapRunnerErrorWithDiagnostics(provider, acpruntime.ErrorCodeRuntimeContract, message, stdout, stderr, rawOutputRefs, cause)
}

func buildFailureMessage(provider acpruntime.Provider, task acpruntime.Task, stage string, cause error, stdout string, stderr string) (string, contracts.RuntimeOutputRefs) {
	base := "unknown failure"
	if cause != nil {
		base = strings.TrimSpace(cause.Error())
	}
	artifacts, err := runnerdiag.WriteFailureArtifacts(task, provider, stdout, stderr)
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

type FakeRunner struct{}

func (FakeRunner) Run(_ context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	semantic := fakeSemanticSnapshot(task)
	summary := fakeSummary(task)
	var verdict *contracts.ValidatorVerdict
	if acpruntime.StepProviderKeyForStepID(task.StepID) == acpruntime.StepProviderStep3Findings {
		verdict = &contracts.ValidatorVerdict{
			Version:      1,
			RunID:        task.RunID,
			GeneratedAt:  task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
			Verdict:      "PASS",
			Summary:      "Fake validator verdict accepted",
			CheckedPaths: []string{"reports/taskruns/" + task.RunID + "/staging/final"},
			Findings:     append([]contracts.Finding(nil), semantic.Findings...),
			Questions:    append([]contracts.Question(nil), semantic.Questions...),
		}
	}
	if err := PersistRuntimeArtifacts(task, summary, semantic, verdict); err != nil {
		return acpruntime.Result{}, err
	}
	if err := validateRuntimeArtifacts(task); err != nil {
		return acpruntime.Result{}, err
	}
	return acpruntime.Result{
		Execution: acpruntime.NewExecution(task, acpruntime.ProviderClaudeCode, "fake", "succeeded", task.StartedAtUTC.UTC().Add(2*time.Second), nil),
	}, nil
}

func fakeSummary(task acpruntime.Task) string {
	switch acpruntime.StepProviderKeyForStepID(task.StepID) {
	case acpruntime.StepProviderStep0Constitution:
		return "Fake constitution draft completed"
	case acpruntime.StepProviderStep1Collect:
		return "Fake collect shard completed"
	case acpruntime.StepProviderStep2AsIs:
		return "Fake as-is draft completed"
	case acpruntime.StepProviderStep3Findings:
		return "Fake validator findings completed"
	case acpruntime.StepProviderStep4Proposals:
		return "Fake proposals draft completed"
	default:
		return "Fake runtime completed"
	}
}

func fakeSemanticSnapshot(task acpruntime.Task) contracts.SemanticSnapshot {
	repoScopes := append([]string(nil), task.RepoScopes...)
	sort.Strings(repoScopes)
	primaryRepo := ""
	if len(repoScopes) > 0 {
		primaryRepo = repoScopes[0]
	}
	if primaryRepo == "" {
		primaryRepo = "stub-repo"
	}
	serviceID := "svc." + slugutil.Slugify(primaryRepo)
	snapshot := contracts.SemanticSnapshot{
		Coverage: contracts.Coverage{
			Observed: []string{"services", "entrypoints"},
			Missing:  []string{"owner mappings", "ci-cd evidence", "api contracts"},
			Notes:    []string{"fake artifact-only runner output"},
		},
		Questions: []contracts.Question{
			{
				ID:         "q.owner." + serviceID,
				Text:       fmt.Sprintf("Who owns %s?", serviceID),
				Priority:   "high",
				RelatedIDs: []string{serviceID},
			},
		},
		Entities: []contracts.Entity{
			{
				ID:   serviceID,
				Type: "service",
				Name: humanizeServiceName(primaryRepo),
				Attributes: map[string]any{
					"repo_scope": primaryRepo,
					"runtime":    "fake",
				},
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.8,
					Evidence: []contracts.Evidence{
						{Repo: primaryRepo, Path: "README.md"},
					},
				},
			},
		},
		Edges: []contracts.Edge{},
		Findings: []contracts.Finding{
			{
				ID:          "finding.owner." + serviceID,
				Severity:    "medium",
				Title:       "Owner mapping is missing",
				Description: "Fake runner keeps an explicit ownership gap for deterministic validation coverage.",
				RuleID:      "rule.owner.required",
				RelatedIDs:  []string{serviceID},
				Provenance: contracts.Provenance{
					Kind:       "inference",
					Confidence: 0.6,
					Evidence: []contracts.Evidence{
						{Repo: primaryRepo, Path: "README.md"},
					},
				},
			},
		},
	}
	if acpruntime.StepProviderKeyForStepID(task.StepID) == acpruntime.StepProviderStep3Findings {
		snapshot.Coverage.Observed = []string{"validator sweep", "staged findings"}
		snapshot.Coverage.Missing = []string{"manual rollout review"}
	}
	return snapshot
}

func humanizeServiceName(repo string) string {
	if strings.TrimSpace(repo) == "" {
		return "Stub Service"
	}
	parts := strings.FieldsFunc(repo, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

type RecordedRunner struct {
	ByStep map[string]string
}

func (r RecordedRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	// Legacy recorded stdout fixtures with semantic JSON are intentionally
	// unsupported in the artifact-only architecture. Tests that still depend on them should be
	// updated to artifact fixtures or removed.
	if _, ok := r.ByStep[task.StepID]; ok {
		return acpruntime.Result{}, fmt.Errorf("recorded legacy semantic stdout fixtures are unsupported in artifact-only runtime")
	}
	return FakeRunner{}.Run(ctx, task)
}
