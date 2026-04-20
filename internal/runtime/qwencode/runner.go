package qwencode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultbinding"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultcompat"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultextractor"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

var (
	ErrRunnerUnavailable             = errors.New("qwen-code runner is unavailable")
	errCollectStalledAfterArtifacts  = errors.New("collect_stalled_after_artifacts")
	errCollectStalledBeforeArtifacts = errors.New("collect_stalled_before_artifacts")

	collectPostArtifactStallWindow = 20 * time.Second
	collectPreArtifactStallWindow  = 75 * time.Second
	collectStallPollInterval       = 2 * time.Second
	collectStallTerminateGrace     = 2 * time.Second
	collectStallPostTerminateDrain = 500 * time.Millisecond
)

type HeadlessRunner struct {
	Command string
	Args    []string
}

type retryManifestAssessment = artifactquality.ManifestAssessment

type promptRetryMode int

type collectStallPhase string

type runQwenOptions struct {
	EnableCollectStallMonitor bool
}

type collectStallDiagnostic struct {
	StallPhase            collectStallPhase
	ManifestState         string
	AuthoredFileCount     int
	LastPipeActivity      time.Time
	LastWriteRootMutation time.Time
}

type collectStallError struct {
	Sentinel   error
	Diagnostic collectStallDiagnostic
}

type collectWriteRootSnapshot struct {
	ManifestPresent   bool
	ManifestState     string
	AuthoredFileCount int
	LastMutation      time.Time
}

type collectRepairAttempt struct {
	Attempted                bool
	ManifestStateBeforeRetry string
	Diagnostic               collectStallDiagnostic
}

type retryDiagnosticContext struct {
	LastStallPhase           collectStallPhase
	ManifestStateBeforeRetry string
	AuthoredFileCount        int
	LastPipeActivity         time.Time
	LastWriteRootMutation    time.Time
}

type commandActivityTracker struct {
	mu       sync.Mutex
	lastRead time.Time
}

type activityTrackingReader struct {
	reader  io.Reader
	tracker *commandActivityTracker
}

type commandOutputBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

const (
	promptRetryNone promptRetryMode = iota
	promptRetryParse
	promptRetryArtifact
	promptRetryCollectFresh
)

const (
	collectStallPhasePreArtifact  collectStallPhase = "pre_artifact"
	collectStallPhasePostArtifact collectStallPhase = "post_artifact"
)

func (e collectStallError) Error() string {
	if e.Sentinel == nil {
		return "collect_stalled"
	}
	return e.Sentinel.Error()
}

func (e collectStallError) Is(target error) bool {
	return target != nil && target == e.Sentinel
}

func (d collectStallDiagnostic) fields(task acpruntime.Task) map[string]any {
	fields := map[string]any{
		"provider":            string(acpruntime.ProviderQwenCode),
		"shard_id":            strings.TrimSpace(task.ShardID),
		"stall_phase":         strings.TrimSpace(string(d.StallPhase)),
		"manifest_state":      strings.TrimSpace(d.ManifestState),
		"authored_file_count": d.AuthoredFileCount,
		"action":              "terminate_and_retry",
	}
	if !d.LastPipeActivity.IsZero() {
		fields["last_pipe_activity_at"] = d.LastPipeActivity.UTC().Format(time.RFC3339)
	}
	if !d.LastWriteRootMutation.IsZero() {
		fields["last_write_root_mutation_at"] = d.LastWriteRootMutation.UTC().Format(time.RFC3339)
	}
	return fields
}

func newCommandActivityTracker(start time.Time) *commandActivityTracker {
	return &commandActivityTracker{lastRead: start.UTC()}
}

func (t *commandActivityTracker) Note(at time.Time) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if at.After(t.lastRead) {
		t.lastRead = at.UTC()
	}
}

func (t *commandActivityTracker) LastRead() time.Time {
	if t == nil {
		return time.Time{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastRead
}

func (r *activityTrackingReader) Read(p []byte) (int, error) {
	if r == nil || r.reader == nil {
		return 0, io.EOF
	}
	n, err := r.reader.Read(p)
	if n > 0 && r.tracker != nil {
		r.tracker.Note(time.Now().UTC())
	}
	return n, err
}

func (b *commandOutputBuffer) WriteString(value string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.WriteString(value)
}

func (b *commandOutputBuffer) BytesCopy() []byte {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}

func (b *commandOutputBuffer) String() string {
	if b == nil {
		return ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (r HeadlessRunner) commandName() string {
	command := strings.TrimSpace(r.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv("ACP_QWEN_CMD"))
	}
	if command == "" {
		command = "qwen"
	}
	return command
}

func (r HeadlessRunner) Preflight(_ context.Context) error {
	command := r.commandName()
	if _, err := exec.LookPath(command); err != nil {
		return acpruntime.WrapRunnerError(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command %q is unavailable: %v", acpruntime.ProviderQwenCode, command, err),
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

	taskPayload, err := json.Marshal(task)
	if err != nil {
		return acpruntime.Result{}, fmt.Errorf("marshal runner task: %w", err)
	}

	args := append([]string(nil), r.Args...)
	if len(args) == 0 {
		args = buildDefaultQwenArgs(task, buildPrompt(taskPayload, false))
	}

	options := runQwenOptions{
		EnableCollectStallMonitor: len(r.Args) == 0 && isCollectStep(task.StepID),
	}
	result, parseStage, parseErr, runErr := runQwenCommand(ctx, task, command, args, options)
	if runErr != nil {
		var stalled collectStallError
		if len(r.Args) == 0 && errors.As(runErr, &stalled) {
			if errors.Is(stalled, errCollectStalledAfterArtifacts) {
				return recoverCollectArtifactsAfterStall(ctx, task, taskPayload, command, result, stalled)
			}
			if errors.Is(stalled, errCollectStalledBeforeArtifacts) {
				return recoverCollectBeforeArtifactsAfterStall(ctx, task, taskPayload, command, result, stalled)
			}
		}
		unavailableMessage := buildUnavailableFailureMessage(task, runErr, result)
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: %s", ErrRunnerUnavailable, unavailableMessage),
			result.Stdout,
			result.Stderr,
			runErr,
		)
	}
	if parseErr == nil {
		if len(r.Args) == 0 {
			repaired, _, repairErr := maybeRepairCollectArtifacts(ctx, task, taskPayload, command, result)
			return repaired, repairErr
		}
		return result, nil
	}
	finalStdout := result.Stdout
	finalStderr := result.Stderr

	// Live qwen output can occasionally contain malformed tokens. Retry once with
	// an explicitly stricter prompt before classifying as parse failure.
	if len(r.Args) == 0 {
		retryArgs := buildRetryQwenArgs(task, buildPromptWithModeAndHints(taskPayload, promptRetryParse, buildParseRepairHints(parseStage, parseErr)))
		retryResult, retryParseStage, retryParseErr, retryRunErr := runQwenCommand(ctx, task, command, retryArgs, runQwenOptions{})
		if retryRunErr != nil {
			unavailableMessage := buildUnavailableFailureMessage(task, retryRunErr, retryResult)
			return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
				acpruntime.ProviderQwenCode,
				acpruntime.ErrorCodeRunnerUnavailable,
				fmt.Sprintf("%v: %s", ErrRunnerUnavailable, unavailableMessage),
				retryResult.Stdout,
				retryResult.Stderr,
				retryRunErr,
			)
		}
		if retryParseErr == nil {
			repaired, _, repairErr := maybeRepairCollectArtifacts(ctx, task, taskPayload, command, retryResult)
			return repaired, repairErr
		}
		result = retryResult
		parseStage = retryParseStage
		parseErr = retryParseErr
		finalStdout = retryResult.Stdout
		finalStderr = retryResult.Stderr
	}

	return acpruntime.Result{}, wrapQwenParseFailure(
		task,
		parseStage,
		parseErr,
		acpruntime.Result{
			TaskResult: result.TaskResult,
			RawJSON:    result.RawJSON,
			Stdout:     finalStdout,
			Stderr:     finalStderr,
		},
		fmt.Sprintf("headless provider %q returned invalid taskresult", acpruntime.ProviderQwenCode),
	)
}

func buildDefaultQwenArgs(task acpruntime.Task, prompt string) []string {
	args := []string{"--output-format", "json", "--chat-recording", "false", "--yolo", "--channel", "CI"}
	for _, dir := range acpruntime.ResolveHeadlessIncludeDirectories(task) {
		args = append(args, "--include-directories", dir)
	}
	args = append(args, "--prompt", prompt)
	return args
}

func buildRetryQwenArgs(task acpruntime.Task, prompt string) []string {
	args := []string{"--output-format", "json", "--chat-recording", "false", "--yolo", "--channel", "CI"}
	for _, dir := range resolveRetryIncludeDirectories(task) {
		args = append(args, "--include-directories", dir)
	}
	args = append(args, "--prompt", prompt)
	return args
}

func maybeRepairCollectArtifacts(
	ctx context.Context,
	task acpruntime.Task,
	taskPayload []byte,
	command string,
	current acpruntime.Result,
) (acpruntime.Result, collectRepairAttempt, error) {
	if task.StepID != "init.step1.collect" && task.StepID != "refresh.step1.collect" {
		return current, collectRepairAttempt{}, nil
	}
	if strings.TrimSpace(task.WriteRoot) == "" {
		return current, collectRepairAttempt{}, nil
	}

	_ = artifactquality.EnsureCanonicalCollectManifest(task, current.TaskResult)
	assessment, err := assessRetryManifestAtWriteRoot(task.WriteRoot)
	if err == nil && assessment.Rich {
		return current, collectRepairAttempt{ManifestStateBeforeRetry: "rich"}, nil
	}
	initialProblem := artifactquality.DescribeAssessmentProblem(assessment, err)
	beforeRepairState := currentCollectManifestState(task.WriteRoot)
	attempt := collectRepairAttempt{
		Attempted:                true,
		ManifestStateBeforeRetry: beforeRepairState,
		Diagnostic:               buildCollectRepairDiagnostic(task.WriteRoot),
	}

	snapshot, err := artifactquality.SnapshotWriteRoot(task.WriteRoot)
	if err != nil {
		return acpruntime.Result{}, attempt, wrapArtifactContractFailure(
			task,
			"artifact_repair.snapshot",
			current,
			fmt.Sprintf("collect artifacts require repair (%s), but write_root snapshot failed", initialProblem),
			err,
		)
	}
	defer func() {
		_ = snapshot.Cleanup()
	}()

	repairArgs := buildRetryQwenArgs(task, buildPromptWithModeAndHints(taskPayload, promptRetryArtifact, buildArtifactRepairHints(initialProblem)))
	repaired, repairParseStage, parseErr, runErr := runQwenCommand(ctx, task, command, repairArgs, runQwenOptions{})
	if runErr != nil {
		_ = snapshot.Restore()
		unavailableMessage := buildUnavailableFailureMessage(task, runErr, repaired)
		return acpruntime.Result{}, attempt, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: artifact repair retry failed after %s: %s", ErrRunnerUnavailable, initialProblem, unavailableMessage),
			repaired.Stdout,
			repaired.Stderr,
			runErr,
		)
	}
	if parseErr != nil {
		_ = snapshot.Restore()
		return acpruntime.Result{}, attempt, wrapQwenParseFailure(
			task,
			"artifact_repair."+repairParseStage,
			parseErr,
			repaired,
			fmt.Sprintf("headless provider %q returned invalid collect artifact repair result after %s", acpruntime.ProviderQwenCode, initialProblem),
		)
	}

	_ = artifactquality.EnsureCanonicalCollectManifest(task, repaired.TaskResult)
	if contractErr := validateCollectManifestContractAtWriteRoot(task.WriteRoot); contractErr != nil {
		_ = snapshot.Restore()
		return acpruntime.Result{}, attempt, wrapArtifactContractFailure(
			task,
			"artifact_repair.contract",
			repaired,
			fmt.Sprintf("collect artifacts remained invalid after one repair attempt: initial=%s; validation=%v", initialProblem, contractErr),
			contractErr,
		)
	}
	repairedAssessment, err := assessRetryManifestAtWriteRoot(task.WriteRoot)
	if err != nil || !repairedAssessment.Rich {
		_ = snapshot.Restore()
		repairedProblem := artifactquality.DescribeAssessmentProblem(repairedAssessment, err)
		return acpruntime.Result{}, attempt, wrapArtifactContractFailure(
			task,
			"artifact_repair.contract",
			repaired,
			fmt.Sprintf("collect artifacts remained invalid after one repair attempt: initial=%s; repaired=%s", initialProblem, repairedProblem),
			err,
		)
	}
	return repaired, attempt, nil
}

func recoverCollectArtifactsAfterStall(
	ctx context.Context,
	task acpruntime.Task,
	taskPayload []byte,
	command string,
	current acpruntime.Result,
	stalled collectStallError,
) (acpruntime.Result, error) {
	initialProblem := errCollectStalledAfterArtifacts.Error()
	emitDiagnostic(task, "runtime task retry scheduled", stalled.Diagnostic.fields(task))
	retryContext := retryDiagnosticContext{
		LastStallPhase:           collectStallPhasePostArtifact,
		ManifestStateBeforeRetry: stalled.Diagnostic.ManifestState,
		AuthoredFileCount:        stalled.Diagnostic.AuthoredFileCount,
		LastPipeActivity:         stalled.Diagnostic.LastPipeActivity,
		LastWriteRootMutation:    stalled.Diagnostic.LastWriteRootMutation,
	}

	snapshot, err := artifactquality.SnapshotWriteRoot(task.WriteRoot)
	if err != nil {
		message := buildFailureMessage(task, "stall_snapshot", fmt.Errorf("%w: snapshot write_root: %v", errCollectStalledAfterArtifacts, err), current)
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", err.Error(), stalled.Diagnostic.ManifestState))
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q collect_stalled_after_artifacts and artifact recovery setup failed: %s", acpruntime.ProviderQwenCode, message),
			current.Stdout,
			current.Stderr,
			err,
		)
	}
	defer func() {
		_ = snapshot.Cleanup()
	}()

	repairArgs := buildRetryQwenArgs(task, buildPromptWithModeAndHints(taskPayload, promptRetryArtifact, buildArtifactRepairHints(initialProblem)))
	repaired, repairParseStage, parseErr, runErr := runQwenCommand(ctx, task, command, repairArgs, runQwenOptions{})
	if runErr != nil {
		_ = snapshot.Restore()
		failureResult := selectFailureResult(repaired, current)
		failureMessage := buildFailureMessage(task, "stall_repair.exec", fmt.Errorf("%w: %v", errCollectStalledAfterArtifacts, runErr), failureResult)
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", runErr.Error(), stalled.Diagnostic.ManifestState))
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q collect_stalled_after_artifacts and repair retry failed: %s", acpruntime.ProviderQwenCode, failureMessage),
			failureResult.Stdout,
			failureResult.Stderr,
			runErr,
		)
	}
	if parseErr != nil {
		_ = snapshot.Restore()
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", parseErr.Error(), stalled.Diagnostic.ManifestState))
		return acpruntime.Result{}, wrapQwenParseFailure(
			task,
			"stall_repair."+strings.TrimSpace(repairParseStage),
			parseErr,
			repaired,
			fmt.Sprintf("headless provider %q collect_stalled_after_artifacts and repair retry returned invalid taskresult", acpruntime.ProviderQwenCode),
		)
	}

	_ = artifactquality.EnsureCanonicalCollectManifest(task, repaired.TaskResult)
	if contractErr := validateCollectManifestContractAtWriteRoot(task.WriteRoot); contractErr != nil {
		_ = snapshot.Restore()
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", contractErr.Error(), currentCollectManifestState(task.WriteRoot)))
		return acpruntime.Result{}, wrapArtifactContractFailure(
			task,
			"stall_repair.contract",
			repaired,
			fmt.Sprintf("collect_stalled_after_artifacts and collect artifacts remained invalid after repair: validation=%v", contractErr),
			errors.Join(errCollectStalledAfterArtifacts, contractErr),
		)
	}
	repairedAssessment, assessErr := assessRetryManifestAtWriteRoot(task.WriteRoot)
	if assessErr != nil || !repairedAssessment.Rich {
		_ = snapshot.Restore()
		repairedProblem := artifactquality.DescribeAssessmentProblem(repairedAssessment, assessErr)
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", repairedProblem, stalled.Diagnostic.ManifestState))
		return acpruntime.Result{}, wrapArtifactContractFailure(
			task,
			"stall_repair.contract",
			repaired,
			fmt.Sprintf("collect_stalled_after_artifacts and collect artifacts remained invalid after repair: repaired=%s", repairedProblem),
			errors.Join(errCollectStalledAfterArtifacts, assessErr),
		)
	}

	emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "succeeded", "", "rich"))
	return repaired, nil
}

func recoverCollectBeforeArtifactsAfterStall(
	ctx context.Context,
	task acpruntime.Task,
	taskPayload []byte,
	command string,
	current acpruntime.Result,
	stalled collectStallError,
) (acpruntime.Result, error) {
	emitDiagnostic(task, "runtime task retry scheduled", stalled.Diagnostic.fields(task))
	retryContext := retryDiagnosticContext{
		LastStallPhase:           stalled.Diagnostic.StallPhase,
		ManifestStateBeforeRetry: stalled.Diagnostic.ManifestState,
		AuthoredFileCount:        stalled.Diagnostic.AuthoredFileCount,
		LastPipeActivity:         stalled.Diagnostic.LastPipeActivity,
		LastWriteRootMutation:    stalled.Diagnostic.LastWriteRootMutation,
	}

	retryArgs := buildRetryQwenArgs(task, buildPromptWithModeAndHints(taskPayload, promptRetryCollectFresh, buildCollectFreshRetryHints(task)))
	retried, retryParseStage, parseErr, runErr := runQwenCommand(ctx, task, command, retryArgs, runQwenOptions{
		EnableCollectStallMonitor: true,
	})
	if runErr != nil {
		var retryStalled collectStallError
		if errors.As(runErr, &retryStalled) {
			retryContext.absorbDiagnostic(retryStalled.Diagnostic)
			if errors.Is(retryStalled, errCollectStalledAfterArtifacts) {
				return recoverCollectArtifactsAfterStall(ctx, task, taskPayload, command, retried, retryStalled)
			}
		}
		failureResult := selectFailureResult(retried, current)
		failureMessage := buildFailureMessage(task, "stall_retry.exec", fmt.Errorf("%w: %v", errCollectStalledBeforeArtifacts, runErr), failureResult)
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", runErr.Error(), currentCollectManifestState(task.WriteRoot)))
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q collect_stalled_before_artifacts and forced retry failed: %s", acpruntime.ProviderQwenCode, failureMessage),
			failureResult.Stdout,
			failureResult.Stderr,
			runErr,
		)
	}
	if parseErr != nil {
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", parseErr.Error(), currentCollectManifestState(task.WriteRoot)))
		return acpruntime.Result{}, wrapQwenParseFailure(
			task,
			"stall_retry."+strings.TrimSpace(retryParseStage),
			parseErr,
			retried,
			fmt.Sprintf("headless provider %q collect_stalled_before_artifacts and forced retry returned invalid taskresult", acpruntime.ProviderQwenCode),
		)
	}

	finalResult, repairAttempt, err := maybeRepairCollectArtifacts(ctx, task, taskPayload, command, retried)
	if repairAttempt.Attempted {
		retryContext.absorbDiagnostic(repairAttempt.Diagnostic)
		retryContext.LastStallPhase = collectStallPhasePostArtifact
		if state := strings.TrimSpace(repairAttempt.ManifestStateBeforeRetry); state != "" {
			retryContext.ManifestStateBeforeRetry = state
		}
	}
	if err != nil {
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", err.Error(), currentCollectManifestState(task.WriteRoot)))
		return acpruntime.Result{}, err
	}

	emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "succeeded", "", currentCollectManifestState(task.WriteRoot)))
	return finalResult, nil
}

func isCollectStep(stepID string) bool {
	return strings.TrimSpace(stepID) == "init.step1.collect" || strings.TrimSpace(stepID) == "refresh.step1.collect"
}

func emitDiagnostic(task acpruntime.Task, message string, fields map[string]any) {
	if task.OnDiagnostic == nil {
		return
	}
	task.OnDiagnostic(acpruntime.DiagnosticEvent{
		Message: strings.TrimSpace(message),
		Fields:  fields,
	})
}

func selectFailureResult(primary acpruntime.Result, fallback acpruntime.Result) acpruntime.Result {
	if strings.TrimSpace(primary.Stdout) != "" || strings.TrimSpace(primary.Stderr) != "" {
		return primary
	}
	return fallback
}

func retryDiagnosticFields(task acpruntime.Task, stalled collectStallError, context retryDiagnosticContext, retryStatus string, errText string, manifestState string) map[string]any {
	fields := stalled.Diagnostic.fields(task)
	lastStallPhase := context.LastStallPhase
	if lastStallPhase == "" {
		lastStallPhase = stalled.Diagnostic.StallPhase
	}
	if lastStallPhase != "" {
		fields["stall_phase"] = strings.TrimSpace(string(lastStallPhase))
		fields["last_stall_phase"] = strings.TrimSpace(string(lastStallPhase))
	}
	if state := strings.TrimSpace(context.ManifestStateBeforeRetry); state != "" {
		fields["manifest_state_before_retry"] = state
	}
	if context.AuthoredFileCount > 0 {
		fields["authored_file_count"] = context.AuthoredFileCount
	}
	if !context.LastPipeActivity.IsZero() {
		fields["last_pipe_activity_at"] = context.LastPipeActivity.UTC().Format(time.RFC3339)
	}
	if !context.LastWriteRootMutation.IsZero() {
		fields["last_write_root_mutation_at"] = context.LastWriteRootMutation.UTC().Format(time.RFC3339)
	}
	if state := strings.TrimSpace(manifestState); state != "" {
		fields["manifest_state"] = state
	}
	if status := strings.TrimSpace(retryStatus); status != "" {
		fields["retry_status"] = status
	}
	if detail := strings.TrimSpace(errText); detail != "" {
		fields["error"] = detail
	}
	return fields
}

func wrapQwenParseFailure(task acpruntime.Task, parseStage string, parseErr error, result acpruntime.Result, contextLabel string) error {
	failureMessage := buildParseFailureMessage(task, parseStage, parseErr, result)
	if taskresultextractor.IsTransportError(parseErr) {
		return acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: %s transport/API failure: %s", ErrRunnerUnavailable, strings.TrimSpace(contextLabel), failureMessage),
			result.Stdout,
			result.Stderr,
			parseErr,
		)
	}
	return acpruntime.WrapRunnerErrorWithOutput(
		acpruntime.ProviderQwenCode,
		acpruntime.ErrorCodeRunnerParseFailed,
		fmt.Sprintf("%s: %s", strings.TrimSpace(contextLabel), failureMessage),
		result.Stdout,
		result.Stderr,
		parseErr,
	)
}

func wrapArtifactContractFailure(task acpruntime.Task, stage string, result acpruntime.Result, message string, cause error) error {
	failure := cause
	trimmed := strings.TrimSpace(message)
	switch {
	case trimmed != "" && cause != nil:
		failure = fmt.Errorf("%s: %w", trimmed, cause)
	case trimmed != "":
		failure = errors.New(trimmed)
	case failure == nil:
		failure = errors.New("invalid collect artifacts")
	}
	failureMessage := buildFailureMessage(task, stage, failure, result)
	return acpruntime.WrapRunnerErrorWithOutput(
		acpruntime.ProviderQwenCode,
		acpruntime.ErrorCodeRunnerParseFailed,
		fmt.Sprintf("headless provider %q produced invalid collect artifacts: %s", acpruntime.ProviderQwenCode, failureMessage),
		result.Stdout,
		result.Stderr,
		cause,
	)
}

func resolveRetryIncludeDirectories(task acpruntime.Task) []string {
	if shouldConstrainRetryToWriteRoot(task) {
		return []string{strings.TrimSpace(task.WriteRoot)}
	}
	return acpruntime.ResolveHeadlessIncludeDirectories(task)
}

func shouldConstrainRetryToWriteRoot(task acpruntime.Task) bool {
	if task.StepID != "init.step1.collect" && task.StepID != "refresh.step1.collect" {
		return false
	}
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return false
	}
	files, total, err := collectRetryWriteRootFiles(writeRoot, 4)
	if err != nil || total < 2 {
		return false
	}
	assessment, assessErr := assessRetryManifestAtWriteRoot(writeRoot)
	if assessErr != nil || !assessment.Rich {
		return false
	}
	for _, rel := range files {
		if rel == "shard-pack-manifest.json" {
			return true
		}
	}
	return false
}

func assessRetryManifestAtWriteRoot(writeRoot string) (retryManifestAssessment, error) {
	return artifactquality.LoadManifestAssessment(writeRoot)
}

func assessRetryManifest(raw []byte) retryManifestAssessment {
	assessment, err := artifactquality.AssessManifestBytes(raw)
	if err != nil {
		return retryManifestAssessment{}
	}
	return assessment
}

func isGenericRuntimeSummaryCitationID(id string) bool {
	return artifactquality.IsGenericRuntimeSummaryCitation(id)
}

func collectRetryWriteRootFiles(writeRoot string, limit int) ([]string, int, error) {
	root := strings.TrimSpace(writeRoot)
	if root == "" {
		return nil, 0, nil
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	if !info.IsDir() {
		return nil, 0, fmt.Errorf("write_root is not a directory: %s", root)
	}

	files := []string{}
	if walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	}); walkErr != nil {
		return nil, 0, walkErr
	}
	sort.Strings(files)

	total := len(files)
	if limit > 0 && len(files) > limit {
		files = append([]string(nil), files[:limit]...)
	}
	return files, total, nil
}

func buildParseFailureMessage(task acpruntime.Task, parseStage string, parseErr error, result acpruntime.Result) string {
	return buildFailureMessage(task, parseStage, parseErr, result)
}

func buildUnavailableFailureMessage(task acpruntime.Task, runErr error, result acpruntime.Result) string {
	return buildFailureMessage(task, "exec", runErr, result)
}

func buildFailureMessage(task acpruntime.Task, stage string, failure error, result acpruntime.Result) string {
	base := "unknown failure"
	if failure != nil {
		base = strings.TrimSpace(failure.Error())
	}
	if base == "" {
		base = "unknown failure"
	}
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = "unknown"
	}
	artifacts, err := runnerdiag.WriteParseFailureArtifacts(task, acpruntime.ProviderQwenCode, result.Stdout, result.Stderr)
	if err != nil {
		return fmt.Sprintf("parse_stage=%s %s (raw_output_persist_failed=%v)", stage, base, err)
	}
	return fmt.Sprintf(
		"parse_stage=%s %s (raw_output=%s stdout_bytes=%d stdout_sha256=%s stderr_bytes=%d stderr_sha256=%s)",
		stage,
		base,
		artifacts.RelativeMetadataPath,
		artifacts.Stdout.Bytes,
		artifacts.Stdout.SHA256,
		artifacts.Stderr.Bytes,
		artifacts.Stderr.SHA256,
	)
}

func runQwenCommand(ctx context.Context, task acpruntime.Task, command string, args []string, options runQwenOptions) (acpruntime.Result, string, error, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	// Keep qwen's project root anchored at the ACP workspace rather than a
	// deep shard staging directory. This avoids provider-local path explosions
	// (for example chat/debug history paths derived from cwd) while preserving
	// explicit write_root instructions inside the prompt.
	if workspace := strings.TrimSpace(task.Workspace); workspace != "" {
		cmd.Dir = workspace
	} else if writeRoot := strings.TrimSpace(task.WriteRoot); writeRoot != "" {
		cmd.Dir = writeRoot
	}

	stdout := &commandOutputBuffer{}
	stderr := &commandOutputBuffer{}
	stdoutPipe, stdoutWriter, err := os.Pipe()
	if err != nil {
		return acpruntime.Result{}, "", nil, err
	}
	stderrPipe, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutPipe.Close()
		_ = stdoutWriter.Close()
		return acpruntime.Result{}, "", nil, err
	}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	if err := cmd.Start(); err != nil {
		_ = stdoutPipe.Close()
		_ = stdoutWriter.Close()
		_ = stderrPipe.Close()
		_ = stderrWriter.Close()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return acpruntime.Result{}, "", nil, ctxErr
		}
		return acpruntime.Result{}, "", nil, err
	}
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	defer closeCommandPipe(stdoutPipe)
	defer closeCommandPipe(stderrPipe)

	activityTracker := newCommandActivityTracker(time.Now().UTC())
	monitorCtx, stopMonitor := context.WithCancel(context.Background())
	defer stopMonitor()
	var monitorWG sync.WaitGroup
	stallCh := make(chan collectStallError, 1)
	var stalledErr error
	var stalledMu sync.Mutex
	if options.EnableCollectStallMonitor && isCollectStep(task.StepID) && strings.TrimSpace(task.WriteRoot) != "" && cmd.Process != nil {
		monitorWG.Add(1)
		go func() {
			defer monitorWG.Done()
			stallErr, stalled := monitorCollectStall(monitorCtx, cmd.Process, task, activityTracker)
			if !stalled {
				return
			}
			stalledMu.Lock()
			stalledErr = stallErr
			stalledMu.Unlock()
			select {
			case stallCh <- stallErr:
			default:
			}
		}()
	}

	var streamErr error
	var streamErrMu sync.Mutex
	captureErr := func(captureErr error) {
		if captureErr == nil {
			return
		}
		streamErrMu.Lock()
		defer streamErrMu.Unlock()
		if streamErr == nil {
			streamErr = captureErr
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	streamDone := make(chan struct{})
	go func() {
		defer wg.Done()
		captureErr(captureCommandStream(&activityTrackingReader{reader: stdoutPipe, tracker: activityTracker}, stdout, task, acpruntime.OutputStreamStdout))
	}()
	go func() {
		defer wg.Done()
		captureErr(captureCommandStream(&activityTrackingReader{reader: stderrPipe, tracker: activityTracker}, stderr, task, acpruntime.OutputStreamStderr))
	}()
	go func() {
		wg.Wait()
		close(streamDone)
	}()

	var waitErr error
	stalledTriggered := false
	select {
	case <-streamDone:
		stopMonitor()
		monitorWG.Wait()
		waitErr = cmd.Wait()
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		waitForCommandStreams(stdoutPipe, stderrPipe, streamDone, collectStallPostTerminateDrain)
		stopMonitor()
		monitorWG.Wait()
		waitErr = waitForCommandExit(cmd, collectStallPostTerminateDrain)
		if waitErr == nil {
			waitErr = ctx.Err()
		}
	case <-stallCh:
		waitForCommandStreams(stdoutPipe, stderrPipe, streamDone, collectStallPostTerminateDrain)
		stopMonitor()
		monitorWG.Wait()
		waitErr = waitForCommandExit(cmd, collectStallPostTerminateDrain)
	}

	if waitErr == nil && streamErr != nil {
		waitErr = streamErr
	}
	result := acpruntime.Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	stalledMu.Lock()
	currentStallErr := stalledErr
	stalledMu.Unlock()
	if currentStallErr != nil {
		stalledTriggered = true
	}
	if stalledTriggered {
		parsed, _, parseErr := parseCapturedTaskResult(task, stdout.BytesCopy(), result.Stdout, result.Stderr)
		if parseErr == nil {
			return parsed, "", nil, nil
		}
		return result, "", nil, currentStallErr
	}
	if waitErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return result, "", nil, ctxErr
		}
		return result, "", nil, runnerdiag.BuildExecFailure(waitErr, result.Stdout, result.Stderr)
	}

	parsed, parseStage, parseErr := parseCapturedTaskResult(task, stdout.BytesCopy(), result.Stdout, result.Stderr)
	return parsed, parseStage, parseErr, nil
}

type streamedOutputBudget struct {
	forwardedBytes int
	truncated      bool
}

func parseCapturedTaskResult(task acpruntime.Task, stdoutBytes []byte, stdoutText string, stderrText string) (acpruntime.Result, string, error) {
	rawTaskResult, err := taskresultextractor.Extract(stdoutBytes)
	if err != nil {
		parseStage := "extract"
		if taskresultextractor.IsTransportError(err) {
			parseStage = "transport"
		}
		return acpruntime.Result{
			Stdout: stdoutText,
			Stderr: stderrText,
		}, parseStage, err
	}
	if normalizedRawTaskResult, changed, normalizeErr := taskresultcompat.NormalizeRawTaskResult(task, rawTaskResult); normalizeErr == nil && changed {
		rawTaskResult = normalizedRawTaskResult
	}
	taskResult, err := contracts.ParseTaskResult(rawTaskResult)
	if err != nil {
		return acpruntime.Result{
			Stdout: stdoutText,
			Stderr: stderrText,
		}, "schema", err
	}
	if err := taskresultbinding.Validate(task, taskResult, acpruntime.ProviderQwenCode); err != nil {
		return acpruntime.Result{
			Stdout: stdoutText,
			Stderr: stderrText,
		}, "binding", err
	}
	return acpruntime.Result{
		TaskResult: taskResult,
		RawJSON:    rawTaskResult,
		Stdout:     stdoutText,
		Stderr:     stderrText,
	}, "", nil
}

func monitorCollectStall(
	ctx context.Context,
	process *os.Process,
	task acpruntime.Task,
	activity *commandActivityTracker,
) (collectStallError, bool) {
	startedAt := time.Now().UTC()
	ticker := time.NewTicker(collectStallPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return collectStallError{}, false
		case <-ticker.C:
			snapshot, err := scanCollectWriteRoot(task.WriteRoot)
			if err != nil {
				continue
			}
			lastPipeActivity := activity.LastRead()
			if lastPipeActivity.IsZero() {
				lastPipeActivity = startedAt
			}
			now := time.Now().UTC()
			lastWriteRootMutation := effectiveCollectMutationTime(snapshot.LastMutation, startedAt)

			if snapshot.ManifestPresent && snapshot.AuthoredFileCount > 0 {
				if now.Sub(lastPipeActivity) < collectPostArtifactStallWindow {
					continue
				}
				if now.Sub(lastWriteRootMutation) < collectPostArtifactStallWindow {
					continue
				}

				diagnostic := collectStallDiagnostic{
					StallPhase:            collectStallPhasePostArtifact,
					ManifestState:         strings.TrimSpace(snapshot.ManifestState),
					AuthoredFileCount:     snapshot.AuthoredFileCount,
					LastPipeActivity:      lastPipeActivity.UTC(),
					LastWriteRootMutation: lastWriteRootMutation.UTC(),
				}
				emitDiagnostic(task, "runtime task stalled after artifacts", diagnostic.fields(task))
				terminateProcessWithGrace(process)
				return collectStallError{Sentinel: errCollectStalledAfterArtifacts, Diagnostic: diagnostic}, true
			}

			if snapshot.ManifestPresent || snapshot.AuthoredFileCount > 0 {
				continue
			}
			if now.Sub(lastPipeActivity) < collectPreArtifactStallWindow {
				continue
			}
			if now.Sub(lastWriteRootMutation) < collectPreArtifactStallWindow {
				continue
			}

			diagnostic := collectStallDiagnostic{
				StallPhase:            collectStallPhasePreArtifact,
				ManifestState:         strings.TrimSpace(snapshot.ManifestState),
				AuthoredFileCount:     snapshot.AuthoredFileCount,
				LastPipeActivity:      lastPipeActivity.UTC(),
				LastWriteRootMutation: lastWriteRootMutation.UTC(),
			}
			emitDiagnostic(task, "runtime task stalled before artifacts", diagnostic.fields(task))
			terminateProcessWithGrace(process)
			return collectStallError{Sentinel: errCollectStalledBeforeArtifacts, Diagnostic: diagnostic}, true
		}
	}
}

func effectiveCollectMutationTime(lastMutation time.Time, startedAt time.Time) time.Time {
	if lastMutation.IsZero() || lastMutation.Before(startedAt) {
		return startedAt.UTC()
	}
	return lastMutation.UTC()
}

func currentCollectManifestState(writeRoot string) string {
	snapshot, err := scanCollectWriteRoot(writeRoot)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(snapshot.ManifestState)
}

func scanCollectWriteRoot(writeRoot string) (collectWriteRootSnapshot, error) {
	root := strings.TrimSpace(writeRoot)
	if root == "" {
		return collectWriteRootSnapshot{ManifestState: "missing"}, nil
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return collectWriteRootSnapshot{ManifestState: "missing"}, nil
		}
		return collectWriteRootSnapshot{}, err
	}
	if !info.IsDir() {
		return collectWriteRootSnapshot{}, fmt.Errorf("write_root is not a directory: %s", root)
	}

	snapshot := collectWriteRootSnapshot{ManifestState: "missing"}
	if walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if info, infoErr := d.Info(); infoErr == nil && info.ModTime().After(snapshot.LastMutation) {
			snapshot.LastMutation = info.ModTime().UTC()
		}
		if rel == "shard-pack-manifest.json" {
			snapshot.ManifestPresent = true
			return nil
		}
		snapshot.AuthoredFileCount++
		return nil
	}); walkErr != nil {
		return collectWriteRootSnapshot{}, walkErr
	}

	if snapshot.ManifestPresent {
		assessment, assessErr := assessRetryManifestAtWriteRoot(root)
		if assessErr == nil && assessment.Rich {
			snapshot.ManifestState = "rich"
		} else {
			snapshot.ManifestState = "invalid"
		}
	}
	return snapshot, nil
}

func terminateProcessWithGrace(process *os.Process) {
	if process == nil {
		return
	}
	if err := process.Signal(syscall.SIGTERM); err != nil && !isProcessDoneErr(err) {
		_ = process.Kill()
		return
	}
	if collectStallTerminateGrace > 0 {
		time.Sleep(collectStallTerminateGrace)
	}
	_ = process.Kill()
}

func isProcessDoneErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrProcessDone) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "already finished") || strings.Contains(lower, "process already finished") || strings.Contains(lower, "no such process")
}

func captureCommandStream(reader io.Reader, sink *commandOutputBuffer, task acpruntime.Task, stream acpruntime.OutputStream) error {
	if sink == nil {
		return errors.New("capture sink is nil")
	}
	bufReader := bufio.NewReader(reader)
	budget := &streamedOutputBudget{}
	for {
		part, err := bufReader.ReadString('\n')
		if len(part) > 0 {
			sink.WriteString(part)
			forwardStreamOutput(task, stream, part, budget)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if isPipeClosedErr(err) {
				return nil
			}
			return err
		}
	}
}

func isPipeClosedErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrClosed) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "file already closed")
}

func forwardStreamOutput(task acpruntime.Task, stream acpruntime.OutputStream, chunk string, budget *streamedOutputBudget) {
	if task.OnOutput == nil || budget == nil {
		return
	}
	normalized := strings.ReplaceAll(chunk, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if budget.truncated {
			continue
		}
		lineBytes := len([]byte(line))
		nextBytes := budget.forwardedBytes + lineBytes
		if nextBytes <= acpruntime.RuntimeOutputStreamHardCapBytes {
			budget.forwardedBytes = nextBytes
			task.OnOutput(acpruntime.OutputChunk{
				Stream: stream,
				Text:   line,
			})
			continue
		}
		remaining := acpruntime.RuntimeOutputStreamHardCapBytes - budget.forwardedBytes
		if remaining > 0 {
			trimmed := line
			if len([]byte(trimmed)) > remaining {
				trimmedBytes := []byte(trimmed)
				if remaining < len(trimmedBytes) {
					trimmed = string(trimmedBytes[:remaining])
				}
			}
			trimmed = strings.TrimSpace(trimmed)
			if trimmed != "" {
				task.OnOutput(acpruntime.OutputChunk{
					Stream: stream,
					Text:   trimmed,
				})
			}
		}
		budget.truncated = true
		task.OnOutput(acpruntime.OutputChunk{
			Stream:    stream,
			Truncated: true,
			Text:      fmt.Sprintf("%s output truncated after %d bytes (internal safeguard)", stream, acpruntime.RuntimeOutputStreamHardCapBytes),
		})
	}
}

func buildPrompt(taskPayload []byte, retry bool) string {
	mode := promptRetryNone
	if retry {
		mode = promptRetryParse
	}
	return buildPromptWithMode(taskPayload, mode)
}

func buildPromptWithMode(taskPayload []byte, mode promptRetryMode) string {
	return buildPromptWithModeAndHints(taskPayload, mode, nil)
}

func buildPromptWithModeAndHints(taskPayload []byte, mode promptRetryMode, extraHints []string) string {
	var task acpruntime.Task
	if err := json.Unmarshal(taskPayload, &task); err != nil {
		return strings.TrimSpace(fmt.Sprintf(`You are ACP runtime provider %q.
Return exactly one valid JSON object for ACP TaskResult.
Do not output markdown, code fences, or any explanatory text.
Task payload JSON:
%s`, acpruntime.ProviderQwenCode, strings.TrimSpace(string(taskPayload))))
	}

	repoScopesJSON := "[]"
	if rawRepoScopes, err := json.Marshal(task.RepoScopes); err == nil {
		repoScopesJSON = string(rawRepoScopes)
	}
	primaryRepoScope := primaryTaskRepoScope(task.RepoScope, task.RepoScopes)
	pathScopesJSON := "[]"
	if rawPathScopes, err := json.Marshal(task.PathScopes); err == nil {
		pathScopesJSON = string(rawPathScopes)
	}
	stepPolicy := buildStepSpecificPolicy(task.StepID)
	repositoryEvidencePolicy := strings.Join([]string{
		`REPOSITORY EVIDENCE RULES:`,
		`- ACP workspace scaffold (workspace.yaml, charter/, model/, reports/) is support context, not the primary source tree.`,
		`- Prefer evidence from repository files under meta.repo_scopes/meta.path_scopes when those files are available.`,
		`- meta.path_scopes may contain directories, files, or a mixed disjoint partition; treat every listed scope as in-bounds evidence for this task.`,
		`- Use ACP-generated workspace artifacts as evidence only for ACP runtime/report state, not as a substitute for repository analysis.`,
	}, "\n")
	docFirstPolicy := buildDocFirstFilesystemPolicy(task)
	strictResultHint := strings.Join([]string{
		`STRICT RESULT JSON MODE:`,
		`- Prefer returning a direct TaskResult JSON object (without envelope wrappers).`,
		`- If using envelope fields like "result", value MUST be a non-empty valid JSON object string.`,
		`- Do NOT emit empty or malformed "result" payload.`,
		`- Do NOT draft, preview, or explain the TaskResult before returning it.`,
		`- Do NOT echo template fragments, markdown examples, or partial JSON.`,
	}, "\n")
	finalResponseDiscipline := strings.Join([]string{
		`FINAL RESPONSE DISCIPLINE:`,
		`- Use tool calls for filesystem reads/writes when needed, but the final assistant message MUST be only the TaskResult JSON object.`,
		`- Do NOT narrate file writes, manifest contents, or planning steps in the final message.`,
		`- After the last tool call, respond immediately with the final JSON object.`,
	}, "\n")
	retryHint := ""
	retryTemplate := ""
	if mode != promptRetryNone {
		retryLines := []string{
			`RETRY MODE: previous output needs one deterministic repair pass.`,
			`Do not include non-ASCII symbols in numbers or timestamps.`,
			`RFC3339 timestamps only (example: 2026-04-09T15:28:49Z).`,
			`Decimals must be compact numeric literals (example: 0.7, not 0. 7).`,
			`COMPACT JSON MODE: keep output concise and deterministic.`,
			`- If envelope form is unavoidable, "result" MUST be a non-empty valid JSON object string.`,
			`- Limit changeset to the minimum actionable operations for this step.`,
			`- Prefer "changeset": [] when write_root already contains authored docs and the step does not strictly require an operation.`,
			`- If changeset is non-empty, use at most 1 operation and at most 3 provenance.evidence items.`,
			`- Keep coverage compact: observed <=2 items, missing <=2 canonical items, notes <=1 short entry.`,
			`- Keep coverage.notes short (<=2 entries).`,
			`- Avoid long prose in summary; keep a single sentence.`,
			`- Do NOT copy long citations, topic arrays, or manifest document inventories into TaskResult JSON.`,
			`- Do NOT collapse a multi-document refresh into one generic "cite.runtime-summary" citation.`,
			`- Do NOT overwrite a rich shard-pack-manifest.json with a skeletal reuse-only manifest.`,
			`- Preserve repo-specific citations when repository evidence already exists or can be recovered from repo roots.`,
			`- Unknown changeset[].op values are forbidden; allowed values are exactly: upsert_entity, remove_entity, upsert_edge, remove_edge, add_finding, add_doc_artifact.`,
			`- For op="add_doc_artifact", the payload key MUST be "doc_artifact"; never use "artifact".`,
			`- If a retry only repaired files inside write_root, prefer "changeset": [] instead of inventing file-write operations.`,
			`- Retry repair is forbidden from inventing synthetic filesystem operations in changeset.`,
			`- If write_root already contains authored docs, return the final TaskResult JSON immediately after minimal write_root inspection.`,
			`- Final response MUST start with "{" and end with "}".`,
			`- Do not output markdown fences, bullet lists, plan text, or template walkthroughs.`,
		}
		switch mode {
		case promptRetryParse:
			retryLines[0] = `RETRY MODE: previous output was invalid JSON.`
		case promptRetryArtifact:
			retryLines[0] = `ARTIFACT REPAIR MODE: previous collect output was schema-valid but write_root artifacts look skeletal or generic-only.`
			retryLines = append(retryLines,
				`- Repair artifact fidelity before returning JSON; this retry is not a fresh repository rediscovery pass.`,
				`- Keep repo roots available when write_root artifacts lack repo-specific citations or collapse to generic summaries.`,
				`- In shard-pack-manifest.json, compatibility.coverage/questions/entities/edges/findings must all exist, and questions/entities/edges/findings must be arrays rather than booleans or null.`,
				`- Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.`,
			)
		case promptRetryCollectFresh:
			retryLines[0] = `COLLECT STALL RECOVERY MODE: previous collect attempt stalled before any artifact was finalized.`
			retryLines = append(retryLines,
				`- Do one minimal repo sweep only; avoid broad exploratory list_directory/read_file loops before the first authored artifact.`,
				`- Quickly produce authored docs plus shard-pack-manifest.json in write_root, or return the final TaskResult immediately if write_root is already complete.`,
				`- After the first filesystem write in write_root, repository exploration is finished except for minimal JSON/manifest repair.`,
				`- Broad repo sweeps after the first write are forbidden in this retry.`,
			)
		}
		retryLines = append(retryLines, extraHints...)
		retryLines = append(retryLines, buildRetryRecoveryHint(task))
		retryHint = strings.Join(retryLines, "\n")
		retryTemplate = "\nRetry-safe minimal template (preferred when reusing existing write_root artifacts):\n" + buildRetryMinimalTaskResultTemplateJSON(task)
	}

	return strings.TrimSpace(fmt.Sprintf(`You are ACP runtime provider %q.
Return exactly one valid JSON object for ACP TaskResult.
Do not output markdown, code fences, explanations, or any text outside the JSON object.

STRICT CONTRACT (must pass):
- top-level required keys: "meta", "summary", "changeset"
- meta required keys: "task_id", "step_id", "runtime", "started_at"
- meta.runtime required key: "name"
- use snake_case keys exactly as shown.
- DO NOT use top-level fields: task_id, run_id, step_id, status.
- Do NOT add a top-level "compatibility" field to TaskResult JSON; shard-pack-manifest.json may contain compatibility, but the response payload must not.
- provenance.kind MUST be one of: observation, inference, assertion.
- provenance.confidence MUST be a NUMBER in range [0,1], never a string.
- provenance.evidence MUST be an ARRAY of objects with repo/path.
- if "questions" is present, it MUST be an array of objects (each object has at least "id" and "text").
- coverage.missing MUST use canonical terms only: owner mappings, ci-cd evidence, delta validation, dependency graph, runtime metrics, api contracts, deployment configs, integration edges, datastore bindings, dependencies.
- question IDs MUST use canonical form without numeric suffixes (example: q.refresh.delta, not q.refresh.delta.1).
- Do not claim workspace is empty/minimal unless provenance evidence includes concrete file paths proving it.
%s
%s
%s
%s
%s
%s

Set meta fields exactly:
- meta.task_id = %q
- meta.step_id = %q
- meta.run_id = %q
- meta.runtime.name = %q
- meta.started_at = %q
- meta.workspace = %q
- meta.shard_id = %q
- meta.repo_scope = %q
- meta.repo_scopes = %s
- meta.path_scopes = %s

Schema-valid template for this task (copy structure and field TYPES, then refine values with available evidence):
%s
%s

Serialized runtime task JSON (context only):
%s`, acpruntime.ProviderQwenCode, stepPolicy, repositoryEvidencePolicy, docFirstPolicy, strictResultHint, finalResponseDiscipline, retryHint, task.TaskID, task.StepID, task.RunID, acpruntime.ProviderQwenCode, task.StartedAtUTC.UTC().Format(time.RFC3339), task.Workspace, task.ShardID, primaryRepoScope, repoScopesJSON, pathScopesJSON, buildTaskResultTemplateJSON(task), retryTemplate, strings.TrimSpace(string(taskPayload))))
}

func buildTaskResultTemplateJSON(task acpruntime.Task) string {
	coverageMissing := []string{"owner mappings", "ci-cd evidence"}
	questions := []contracts.Question{}
	if strings.HasPrefix(task.StepID, "refresh.") {
		coverageMissing = append(coverageMissing, "delta validation")
		questions = []contracts.Question{
			{
				ID:       "q.refresh.delta",
				Text:     "What changed since previous run that affects ownership or dependencies?",
				Priority: "high",
			},
		}
	}

	template := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:     task.TaskID,
			StepID:     task.StepID,
			RunID:      task.RunID,
			Runtime:    contracts.RuntimeMeta{Name: string(acpruntime.ProviderQwenCode), Version: "0.14.2"},
			StartedAt:  task.StartedAtUTC.UTC().Format(time.RFC3339),
			FinishedAt: task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
			Workspace:  task.Workspace,
			ShardID:    task.ShardID,
			RepoScope:  primaryTaskRepoScope(task.RepoScope, task.RepoScopes),
			RepoScopes: append([]string(nil), task.RepoScopes...),
			PathScopes: append([]string(nil), task.PathScopes...),
		},
		Summary:   "Task completed with contract-compliant output.",
		Changeset: buildTemplateChangeset(task),
		Coverage: &contracts.Coverage{
			Observed: []string{"services"},
			Missing:  coverageMissing,
			Notes:    []string{"evidence gaps are captured explicitly"},
		},
		Questions: questions,
		Warnings:  []string{},
	}
	raw, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func buildRetryMinimalTaskResultTemplateJSON(task acpruntime.Task) string {
	template := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:     task.TaskID,
			StepID:     task.StepID,
			RunID:      task.RunID,
			Runtime:    contracts.RuntimeMeta{Name: string(acpruntime.ProviderQwenCode), Version: "0.14.2"},
			StartedAt:  task.StartedAtUTC.UTC().Format(time.RFC3339),
			FinishedAt: task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
			Workspace:  task.Workspace,
			ShardID:    task.ShardID,
			RepoScope:  primaryTaskRepoScope(task.RepoScope, task.RepoScopes),
			RepoScopes: append([]string(nil), task.RepoScopes...),
			PathScopes: append([]string(nil), task.PathScopes...),
		},
		Summary:   "Reused existing shard artifacts.",
		Changeset: []contracts.Operation{},
		Coverage: &contracts.Coverage{
			Observed: []string{"artifacts"},
			Missing:  []string{"owner mappings", "runtime metrics"},
			Notes:    []string{"write_root artifacts reused."},
		},
	}

	raw, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func buildRetryRecoveryHint(task acpruntime.Task) string {
	lines := []string{
		`RETRY RECOVERY MODE: previous attempt may already have written shard artifacts.`,
		`- Retry goal is JSON repair, not fresh repository exploration.`,
		`- First inspect write_root, not repo roots.`,
		`- If authored docs already exist in write_root, reuse them instead of rediscovering the repository.`,
		`- Once write_root contains authored docs, repository exploration is finished except for minimal JSON/manifest repair.`,
		`- After shard-pack-manifest.json exists, do NOT continue broad list_directory/read_file sweeps across repo roots.`,
		`- Preserve repo-specific citations when existing artifacts already contain them; do not replace them with one generic runtime summary citation.`,
		`- Do NOT use todo_write, plan-style narration, or repeated broad list_directory sweeps in retry mode.`,
		`- Do NOT delegate to agent/subagent helpers in retry mode.`,
		`- Use at most 3 tool calls in retry mode unless a required artifact is missing from write_root.`,
		`- Repair mode forbids inventing file operations in changeset; prefer "changeset": [].`,
		`- Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.`,
		`- After optional write_root inspection, respond immediately with the final TaskResult JSON object.`,
	}

	writeRoot := strings.TrimSpace(task.WriteRoot)
	files, total, err := collectRetryWriteRootFiles(writeRoot, 8)
	assessment, assessmentErr := assessRetryManifestAtWriteRoot(writeRoot)
	switch {
	case writeRoot == "":
		lines = append(lines, `- write_root snapshot unavailable because write_root is empty.`)
	case err != nil:
		lines = append(lines, fmt.Sprintf(`- write_root snapshot unavailable: %v.`, err))
	case total == 0:
		lines = append(lines, fmt.Sprintf(`- write_root %q is empty or missing; create only the minimum missing artifacts before returning JSON.`, writeRoot))
	default:
		suffix := ""
		if total > len(files) {
			suffix = fmt.Sprintf(" (+%d more)", total-len(files))
		}
		lines = append(lines, fmt.Sprintf(`- write_root already contains %d file(s): %s%s`, total, strings.Join(files, ", "), suffix))
		if containsString(files, "shard-pack-manifest.json") {
			switch {
			case assessmentErr != nil:
				lines = append(lines, fmt.Sprintf(`- shard-pack-manifest.json is present but could not be assessed (%v); keep repo roots available while repairing JSON.`, assessmentErr))
				lines = append(lines, `- Rewrite shard-pack-manifest.json to the canonical ACP schema (version=1 integer, documents[].citation_ids, citations[].id/document_ids, stable canonical_path).`)
			case assessment.Rich:
				lines = append(lines, `- shard-pack-manifest.json is already present in write_root; read it first and reuse authored docs instead of re-reading the repository.`)
				lines = append(lines, `- When reusing authored docs, prefer "changeset": [] or one minimal operation instead of restating manifest contents.`)
				lines = append(lines, `- Do NOT copy long citations, topic arrays, or manifest document inventories into TaskResult JSON.`)
			default:
				lines = append(lines, `- shard-pack-manifest.json exists in write_root but looks skeletal/reuse-only; keep repo roots in include-directories and repair JSON without collapsing repository evidence.`)
				lines = append(lines, `- Do NOT reduce multi-document refresh evidence to one generic "cite.runtime-summary" citation.`)
				lines = append(lines, `- Preserve or restore repo-specific citations before returning the final TaskResult JSON object.`)
			}
		}
		if shouldConstrainRetryToWriteRoot(task) {
			lines = append(lines, `- Retry include-directories are constrained to write_root because the manifest and authored docs already exist.`)
		}
	}
	return strings.Join(lines, "\n")
}

func buildCollectFreshRetryHints(task acpruntime.Task) []string {
	lines := []string{
		`- This retry exists because the provider produced no durable collect artifacts in time.`,
		`- Spend the minimum time needed to identify repo-backed evidence for one authored doc and shard-pack-manifest.json.`,
		`- Keep repository exploration minimal; do not resume a broad repo sweep on this retry.`,
		`- Do NOT delegate to agent/subagent helpers and do NOT use todo_write-style planning on this retry.`,
		`- If write_root is still empty, create only the minimum viable authored doc set before returning the final JSON.`,
		`- As soon as the first authored doc and shard-pack-manifest.json exist in write_root, stop repository exploration and return the final TaskResult JSON immediately.`,
	}
	if strings.TrimSpace(task.WriteRoot) != "" {
		lines = append(lines, fmt.Sprintf(`- write_root for this retry: %q`, strings.TrimSpace(task.WriteRoot)))
	}
	return lines
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func buildStepSpecificPolicy(stepID string) string {
	switch stepID {
	case "init.step0.constitution":
		return strings.Join([]string{
			`STEP POLICY init.step0.constitution:`,
			`- Do NOT delegate to agent/subagent helpers.`,
			`- Do NOT use todo_write-style planning or long plan narration.`,
			`- Use only existing repo entrypoint hints; do not assume README.md exists when it is not present.`,
			`- After the first useful evidence pass, converge to constitution drafts instead of continuing a broad repo sweep.`,
		}, "\n")
	case "init.step1.collect":
		return strings.Join([]string{
			`STEP POLICY init.step1.collect:`,
			`- Do NOT delegate to agent/subagent helpers.`,
			`- Do NOT use todo_write-style planning or long plan narration.`,
			`- Use only existing repo entrypoint hints; do not assume README.md exists when it is not present.`,
			`- After the first useful evidence pass, converge to authored docs plus shard-pack-manifest.json instead of continuing a broad repo sweep.`,
		}, "\n")
	case "refresh.step1.collect":
		return strings.Join([]string{
			`STEP POLICY refresh.step1.collect:`,
			`- Do NOT delegate to agent/subagent helpers.`,
			`- Do NOT use todo_write-style planning or long plan narration.`,
			`- Use only existing repo entrypoint hints; do not assume README.md exists when it is not present.`,
			`- After the first useful evidence pass, converge to authored docs plus shard-pack-manifest.json instead of continuing a broad repo sweep.`,
			`- Allowed upsert_entity types: service, datastore, integration, external.system, team, domain, api, component.`,
			`- Forbidden placeholder entity types: runtime_provider, runtime, metadata.`,
			`- Analyze only repository/workspace artifacts; do NOT perform web search or external browsing.`,
			`- Every provenance.evidence.path must resolve to an existing file in workspace/repo scope.`,
			`- Do NOT emit synthetic evidence paths such as search_source/*, search_query/*, search_config/*.`,
			`- Do NOT introduce unrelated incident domains (for example bidding/tender/power-system topics) unless explicitly present in repository evidence.`,
			`- If evidence is incomplete, capture gap via coverage.missing instead of synthetic placeholder entities.`,
			`- Include at least one question and at least three items in coverage.missing.`,
		}, "\n")
	case "refresh.step3.findings":
		return strings.Join([]string{
			`STEP POLICY refresh.step3.findings:`,
			`- If owner mapping is unresolved in evidence/coverage, include at least one add_finding operation.`,
			`- Each finding must include rule_id, related_ids, and provenance.evidence[].`,
			`- For observation provenance, evidence array MUST be non-empty.`,
			`- If meta.repo_scopes has 2+ scopes, include at least one upsert_edge that links entities from different repo_scope values.`,
			`- For upsert_edge use canonical keys only: edge.id, edge.type, edge.from, edge.to.`,
			`- Forbidden edge aliases: edge.kind, edge.source, edge.target.`,
			`- Minimal valid upsert_edge example: {"op":"upsert_edge","edge":{"id":"edge.cross.scope","type":"depends_on","from":"svc.a","to":"svc.b","provenance":{"kind":"inference","confidence":0.6,"evidence":[{"repo":"scope-a","path":"README.md"},{"repo":"scope-b","path":"README.md"}]}}}`,
			`- Include at least one question and at least three items in coverage.missing.`,
		}, "\n")
	default:
		if strings.HasPrefix(stepID, "refresh.") {
			return `For refresh steps include at least one question object and at least three items in coverage.missing.`
		}
		return ""
	}
}

func buildDocFirstFilesystemPolicy(task acpruntime.Task) string {
	readContextRootsJSON := "[]"
	if raw, err := json.Marshal(task.ReadContextRoots); err == nil {
		readContextRootsJSON = string(raw)
	}
	lines := []string{
		`DOCS-FIRST FILESYSTEM CONTRACT:`,
		`- Read only from meta.workspace and meta.path_scopes plus runtime read_context_roots; do not treat workspace root as implicit write target.`,
		`- Write ONLY inside write_root. Never write to workspace.yaml, schemas/*, docs/spec/*, charter/*, or analyzed user repositories.`,
		`- Use tool calls for any file writes, but keep the final assistant response limited to the required TaskResult JSON object.`,
		fmt.Sprintf(`- artifact_root (workspace-relative) = %q`, strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- draft_final_root (absolute) = %q`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- read_context_roots = %s`, readContextRootsJSON),
		fmt.Sprintf(`- domain_id = %q`, strings.TrimSpace(task.DomainID)),
		fmt.Sprintf(`- agent_role = %q`, strings.TrimSpace(task.AgentRole)),
		fmt.Sprintf(`- step_contract = %q`, strings.TrimSpace(task.StepContract)),
		fmt.Sprintf(`- expected_artifacts = %s`, strings.Join(task.ExpectedArtifacts, ", ")),
	}
	if entrypointHints := collectRepoEntrypointHints(task); len(entrypointHints) > 0 {
		lines = append(lines, fmt.Sprintf(`- Existing repo entrypoint hints (read only these first when relevant): %s`, strings.Join(entrypointHints, ", ")))
	} else if task.StepID == "init.step0.constitution" || task.StepID == "init.step1.collect" || task.StepID == "refresh.step1.collect" {
		lines = append(lines, `- Repo entrypoint hints are limited to actually existing files; do not assume README.md exists when it is absent.`)
	}
	switch task.StepID {
	case "init.step0.constitution":
		lines = append(lines,
			`- Do NOT delegate to agent/subagent helpers and do NOT use todo_write-style planning.`,
			`- Write constitution-draft.json in write_root.`,
			`- Draft canonical files only under draft_final_root, targeting charter/overview.md and skills/subagents.yaml.`,
			`- Keep the draft deterministic in shape; compiler will normalize/publish canonical files afterwards.`,
		)
	case "init.step1.collect", "refresh.step1.collect":
		lines = append(lines,
			`- Do NOT delegate to agent/subagent helpers and do NOT use todo_write-style planning.`,
			`- Before the first filesystem write inside write_root, keep repository exploration minimal and converge quickly on the first authored doc plus shard-pack-manifest.json.`,
			`- Produce runtime-authored documents in write_root and then write shard-pack-manifest.json in write_root.`,
			`- shard-pack-manifest.json must describe every authored document, its canonical stable path, citations, and compatibility snapshot.`,
			`- In shard-pack-manifest.json, compatibility MUST include coverage, questions, entities, edges, and findings.`,
			`- Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.`,
			`- You may be flexible in document structure, but promotion and rendering depend on manifest citations/topics remaining accurate.`,
			`- After the first filesystem write inside write_root, stop broad repository exploration; only minimal manifest/JSON repair is allowed afterwards.`,
			`- After writing shard-pack-manifest.json, do NOT continue broad list_directory/read_file sweeps across repo roots.`,
			`- If authored docs already exist in write_root, respond immediately with the final TaskResult JSON object.`,
		)
		lines = append(lines, artifactquality.CollectManifestContractLines(strings.TrimSpace(task.ArtifactRoot))...)
		lines = append(lines, artifactquality.ClaimIDContractLines()...)
		lines = append(lines,
			`- Do NOT collapse a multi-document refresh surface to one generic "cite.runtime-summary" citation when repository evidence exists.`,
			`- Preserve repo-specific citations in shard-pack-manifest.json whenever repository files support them.`,
		)
	case "init.step3.findings", "refresh.step3.findings":
		lines = append(lines,
			`- Inspect staged final artifacts under reports/taskruns/<run_id>/staging/final from read_context_roots.`,
			`- Write validator-verdict.json in write_root.`,
			`- Validator may fix only indexes, references, or technical document issues inside write_root; do not rewrite document meaning wholesale.`,
		)
		lines = append(lines, artifactquality.ClaimIDContractLines()...)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines,
			`- Write asis-draft-manifest.json in write_root.`,
			`- Draft final docs only under draft_final_root.`,
			`- Allowed canonical targets are reports/as-is/*, reports/coverage/*, and reports/agent-outputs/*.`,
			`- Compiler will merge these drafts into staged final artifacts and keep canonical layout/indexing deterministic.`,
		)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines,
			`- Inspect validated staged artifacts under reports/taskruns/<run_id>/staging/final from read_context_roots.`,
			`- Write proposals-draft-manifest.json in write_root.`,
			`- Draft final docs only under draft_final_root.`,
			`- Allowed canonical targets are proposals/* and reports/changelog/*.`,
			`- Promotion remains deterministic; your drafts become publish candidates only after compile/publish gates.`,
		)
	}
	return strings.Join(lines, "\n")
}

func buildParseRepairHints(parseStage string, parseErr error) []string {
	if parseErr == nil {
		return nil
	}
	lines := []string{}
	if detail := compactRetryHint(parseErr.Error()); detail != "" {
		stage := strings.TrimSpace(parseStage)
		if stage == "" {
			stage = "unknown"
		}
		lines = append(lines, fmt.Sprintf(`- Previous %s validation failure: %s`, stage, detail))
	}
	if strings.Contains(parseErr.Error(), `additionalProperties 'compatibility' not allowed`) {
		lines = append(lines, `- Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.`)
	}
	return append(lines,
		`- Return a direct TaskResult JSON object, not an event-stream transcript or tool wrapper.`,
		`- Do NOT use ad-hoc ops such as upsert_file, write_file, update_file, or todo_write in changeset[].op.`,
		`- For add_doc_artifact, use doc_artifact and never the legacy artifact field.`,
	)
}

func buildArtifactRepairHints(initialProblem string) []string {
	lines := []string{
		`- Rebuild shard-pack-manifest.json to the canonical ACP schema before returning JSON.`,
		`- In shard-pack-manifest.json, compatibility.coverage/questions/entities/edges/findings are all required; questions/entities/edges/findings must be arrays even when empty.`,
		`- documents[].path MUST stay relative to artifact_root only; valid example: "iac-overview.md". Invalid examples: "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac/iac-overview.md", "charter/overview.md".`,
		`- Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.`,
		`- compatibility.entities[*] MUST remain full entity objects with provenance included; do not drop entities[*].provenance during repair.`,
		`- compatibility.edges[*] MUST remain objects with canonical keys type/from/to; do not use kind/source/target aliases.`,
		`- compatibility.findings[*] MUST remain objects and each finding MUST include title; never replace findings with plain strings or bullet text.`,
		`- compatibility.questions/entities/edges/findings must stay object-only arrays; booleans, nulls, and string-valued findings are invalid.`,
		`- Do NOT leave claim_ids empty for cited repository evidence; preserve concrete repo-backed claim ids whenever the evidence supports them.`,
		`- Do NOT describe shard-pack-manifest.json repair via add_doc_artifact; repair the file in write_root and return "changeset": [].`,
		`- Repair mode is JSON-only: do not invent extra repository file reads/writes in changeset after authored docs already exist.`,
		`- Valid compatibility examples: entities[*].provenance={"kind":"observation","confidence":0.7,"evidence":[...]}, edges[*]={"id":"edge.dep","type":"depends_on","from":"svc.a","to":"svc.b","provenance":{...}}, findings[*]={"id":"finding.x","severity":"medium","title":"Missing owner mapping","description":"...","rule_id":"rule.owner.required","related_ids":["svc.a"],"provenance":{...}}.`,
	}
	if detail := compactRetryHint(initialProblem); detail != "" {
		lines = append(lines, fmt.Sprintf(`- Previous artifact contract failure: %s`, detail))
	}
	return lines
}

func compactRetryHint(value string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(normalized) <= 320 {
		return normalized
	}
	return normalized[:317] + "..."
}

func buildTemplateChangeset(task acpruntime.Task) []contracts.Operation {
	scopes := append([]string(nil), task.RepoScopes...)
	if len(scopes) == 0 {
		scopes = []string{"repository"}
	}
	changes := make([]contracts.Operation, 0, len(scopes))
	switch task.StepID {
	case "init.step3.findings", "refresh.step3.findings":
		for _, scope := range scopes {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				scope = "repository"
			}
			slug := slugutil.Slugify(scope)
			if slug == "" {
				slug = "repository"
			}
			changes = append(changes, contracts.Operation{
				Op: "add_finding",
				Finding: &contracts.Finding{
					ID:          "finding.missing-owner.svc." + slug,
					Severity:    "medium",
					Title:       "Missing owner mapping",
					Description: "owner_team_id is not confirmed",
					RuleID:      "rule.owner.required",
					RelatedIDs:  []string{"svc." + slug},
					Provenance: contracts.Provenance{
						Kind:       "inference",
						Confidence: 0.66,
					},
				},
			})
		}
	default:
		for _, scope := range scopes {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				scope = "repository"
			}
			slug := slugutil.Slugify(scope)
			if slug == "" {
				slug = "repository"
			}
			changes = append(changes, contracts.Operation{
				Op: "upsert_entity",
				Entity: &contracts.Entity{
					ID:   "svc." + slug,
					Type: "service",
					Name: humanizeScope(scope) + " Service",
					Attributes: map[string]any{
						"repo_scope": scope,
						"runtime":    acpruntime.ProviderQwenCode,
					},
					Provenance: contracts.Provenance{
						Kind:       "observation",
						Confidence: 0.7,
						Evidence: []contracts.Evidence{
							{
								Repo: scope,
								Path: "README.md",
							},
						},
					},
				},
			})
		}
	}
	return changes
}

func humanizeScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "Repository"
	}
	parts := strings.FieldsFunc(scope, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})
	for idx, part := range parts {
		if part == "" {
			continue
		}
		parts[idx] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	name := strings.TrimSpace(strings.Join(parts, " "))
	if name == "" {
		return "Repository"
	}
	return name
}

func primaryTaskRepoScope(explicit string, scopes []string) string {
	if value := strings.TrimSpace(explicit); value != "" {
		return value
	}
	for _, scope := range scopes {
		if value := strings.TrimSpace(scope); value != "" {
			return value
		}
	}
	return ""
}

func collectRepoEntrypointHints(task acpruntime.Task) []string {
	if len(task.ReadContextRoots) == 0 {
		return nil
	}
	patterns := []string{"README.*", "catalog-info.yaml", "pyproject.toml", "package.json", "docker-compose*", "skaffold.yaml", "Makefile"}
	hints := []string{}
	seen := map[string]struct{}{}
	for _, root := range task.ReadContextRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		info, err := os.Stat(root)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			continue
		}
		for _, pattern := range patterns {
			matches, err := filepath.Glob(filepath.Join(root, pattern))
			if err != nil {
				continue
			}
			sort.Strings(matches)
			for _, match := range matches {
				matchInfo, err := os.Stat(match)
				if err != nil || matchInfo.IsDir() {
					continue
				}
				display := formatEntrypointHint(task.Workspace, match)
				if display == "" {
					continue
				}
				if _, exists := seen[display]; exists {
					continue
				}
				seen[display] = struct{}{}
				hints = append(hints, display)
				if len(hints) >= 8 {
					return hints
				}
			}
		}
	}
	return hints
}

func formatEntrypointHint(_ string, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	return filepath.ToSlash(target)
}

func validateCollectManifestContractAtWriteRoot(writeRoot string) error {
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return fmt.Errorf("write_root is empty")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(writeRoot), "shard-pack-manifest.json"))
	if err != nil {
		return fmt.Errorf("read shard pack manifest: %w", err)
	}
	if _, err := contracts.ParseShardPackManifest(raw); err != nil {
		return err
	}
	return nil
}

func (c *retryDiagnosticContext) absorbDiagnostic(diagnostic collectStallDiagnostic) {
	if c == nil {
		return
	}
	if diagnostic.StallPhase != "" {
		c.LastStallPhase = diagnostic.StallPhase
	}
	if state := strings.TrimSpace(diagnostic.ManifestState); state != "" {
		c.ManifestStateBeforeRetry = state
	}
	if diagnostic.AuthoredFileCount > 0 {
		c.AuthoredFileCount = diagnostic.AuthoredFileCount
	}
	if !diagnostic.LastPipeActivity.IsZero() {
		c.LastPipeActivity = diagnostic.LastPipeActivity.UTC()
	}
	if !diagnostic.LastWriteRootMutation.IsZero() {
		c.LastWriteRootMutation = diagnostic.LastWriteRootMutation.UTC()
	}
}

func buildCollectRepairDiagnostic(writeRoot string) collectStallDiagnostic {
	snapshot, err := scanCollectWriteRoot(writeRoot)
	if err != nil {
		return collectStallDiagnostic{}
	}
	return collectStallDiagnostic{
		StallPhase:            collectStallPhasePostArtifact,
		ManifestState:         strings.TrimSpace(snapshot.ManifestState),
		AuthoredFileCount:     snapshot.AuthoredFileCount,
		LastWriteRootMutation: snapshot.LastMutation.UTC(),
	}
}

func waitForCommandStreams(stdoutPipe io.ReadCloser, stderrPipe io.ReadCloser, streamDone <-chan struct{}, timeout time.Duration) {
	if timeout <= 0 {
		<-streamDone
		return
	}
	select {
	case <-streamDone:
		return
	case <-time.After(timeout):
		closeCommandPipe(stdoutPipe)
		closeCommandPipe(stderrPipe)
		select {
		case <-streamDone:
		case <-time.After(timeout):
		}
	}
}

func waitForCommandExit(cmd *exec.Cmd, timeout time.Duration) error {
	if cmd == nil {
		return nil
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	if timeout <= 0 {
		return <-done
	}
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return nil
	}
}

func closeCommandPipe(pipe io.Closer) {
	if pipe == nil {
		return
	}
	_ = pipe.Close()
}
