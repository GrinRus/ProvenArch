package providercommon

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
	"strings"
	"sync"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

const (
	defaultPostArtifactStallWindow = 20 * time.Second
	defaultPreArtifactStallWindow  = 75 * time.Second
	defaultRetryPreArtifactWindow  = 3 * time.Minute
	defaultStallPollInterval       = 2 * time.Second
	defaultStallTerminateGrace     = 2 * time.Second
	defaultPostTerminateDrain      = 500 * time.Millisecond
)

// CommandSpec is the provider-specific process invocation surface. Success is
// still determined by artifact validation, not by stdout/stderr shape.
type CommandSpec struct {
	Command string
	Args    []string
	Stdin   io.Reader
	Dir     string
	// IncludeDirs records the read scope the adapter encoded into provider CLI
	// args. The shared engine does not interpret it as a success contract.
	IncludeDirs []string
}

// ProviderAdapter keeps CLI differences at the edge while sharing process
// lifecycle, artifact validation, and failure classification.
type ProviderAdapter interface {
	Provider() acpruntime.Provider
	RuntimeVersion() string
	CommandSpec(acpruntime.Task) (CommandSpec, error)
	ValidateArtifacts(acpruntime.Task) error
	ActivityPolicy(acpruntime.Task) ActivityPolicy
	RecoveryPolicy(acpruntime.Task) RecoveryPolicy
	UnavailableMarkers() []string
}

type ActivityPolicy struct {
	MonitorArtifacts            bool
	MonitorPreArtifact          bool
	PreArtifactStallWindow      time.Duration
	RetryPreArtifactStallWindow time.Duration
	PostArtifactStallWindow     time.Duration
	PartialArtifactStallWindow  time.Duration
	PollInterval                time.Duration
	TerminateGrace              time.Duration
	PostTerminateDrain          time.Duration
}

type RecoveryPolicy struct {
	AcceptValidArtifactsAfterStop            bool
	RetryInvalidOrMissingArtifactsOnce       bool
	ClassifySilentRetryExhaustionUnavailable bool
}

func DefaultUnavailableMarkers() []string {
	return []string{
		"permission_error",
		"permission error",
		"usage limit",
		"quota exceeded",
		"quota",
		"insufficient credits",
		"credit balance",
		"rate limit",
		"rate_limit",
		"status code: 403",
		"status code: 429",
		"api error: 403",
		"api error: 429",
		"ssl",
		"tls",
		"certificate",
		"network",
		"socket",
		"packet length too long",
		"http2",
	}
}

type StallPhase string

const (
	StallPhasePreArtifact  StallPhase = "pre_artifact"
	StallPhasePostArtifact StallPhase = "post_artifact"
)

var (
	ErrStalledBeforeArtifacts = errors.New("runtime_stalled_before_artifacts")
	ErrStalledAfterArtifacts  = errors.New("runtime_stalled_after_artifacts")
)

type StallDiagnostic struct {
	StallPhase            StallPhase
	ArtifactState         string
	AuthoredFileCount     int
	LastPipeActivity      time.Time
	LastWriteRootMutation time.Time
}

type StallError struct {
	Sentinel   error
	Diagnostic StallDiagnostic
}

func (e StallError) Error() string {
	if e.Sentinel == nil {
		return "runtime_stalled"
	}
	return e.Sentinel.Error()
}

func (e StallError) Is(target error) bool {
	return target != nil && target == e.Sentinel
}

func PreflightCommand(provider acpruntime.Provider, command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return acpruntime.WrapRunnerError(
			provider,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command is empty", provider),
			nil,
		)
	}
	if _, err := exec.LookPath(command); err != nil {
		return acpruntime.WrapRunnerError(
			provider,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command %q is unavailable: %v", provider, command, err),
			err,
		)
	}
	return nil
}

func JSONTaskStdin(task acpruntime.Task) (io.Reader, error) {
	taskPayload, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime task: %w", err)
	}
	return bytes.NewReader(taskPayload), nil
}

func RunHeadlessProvider(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter) (acpruntime.Result, error) {
	if adapter == nil {
		return acpruntime.Result{}, errors.New("provider adapter is nil")
	}
	result, runErr := runProviderCommand(ctx, task, adapter, normalizeActivityPolicy(adapter.ActivityPolicy(task)))
	if runErr != nil {
		if recovered, recoveredResult, recoveredErr := recoverAfterStall(ctx, task, adapter, result, runErr); recovered {
			if recoveredErr != nil {
				return acpruntime.Result{}, recoveredErr
			}
			recoveredResult.Execution = acpruntime.NewExecution(task, adapter.Provider(), adapter.RuntimeVersion(), "succeeded", time.Now().UTC(), nil)
			return recoveredResult, nil
		}
		return acpruntime.Result{}, classifyCommandFailure(adapter, task, result, runErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		if recovered, recoveredResult, recoveredErr := recoverAfterArtifactValidationFailure(ctx, task, adapter, result, err); recovered {
			if recoveredErr != nil {
				return acpruntime.Result{}, recoveredErr
			}
			recoveredResult.Execution = acpruntime.NewExecution(task, adapter.Provider(), adapter.RuntimeVersion(), "succeeded", time.Now().UTC(), nil)
			return recoveredResult, nil
		}
		return acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "contract", "artifact validation failed", err)
	}
	result.Execution = acpruntime.NewExecution(task, adapter.Provider(), adapter.RuntimeVersion(), "succeeded", time.Now().UTC(), nil)
	return result, nil
}

func recoverAfterArtifactValidationFailure(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error) (bool, acpruntime.Result, error) {
	policy := adapter.RecoveryPolicy(task)
	if !policy.RetryInvalidOrMissingArtifactsOnce {
		return false, acpruntime.Result{}, nil
	}

	initialStructuralFailure := isStructuralArtifactContractFailure(validationErr)
	emitArtifactRetryScheduledDiagnostic(task, adapter.Provider(), validationErr)
	retryResult, retryErr := runProviderCommand(ctx, task, adapter, normalizeActivityPolicy(adapter.ActivityPolicy(task)))
	if retryErr != nil {
		var retryStalled StallError
		if errors.As(retryErr, &retryStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitRetryCompletedDiagnostic(task, adapter.Provider(), retryStalled.Diagnostic.StallPhase, "fresh_process_artifact_only")
					return true, retryResult, nil
				}
			}
			emitRetryExhaustedDiagnostic(task, adapter.Provider(), retryStalled.Diagnostic, "fresh_process")
			if initialStructuralFailure {
				return true, acpruntime.Result{}, wrapArtifactContractFailure(adapter, task, "retry", retryResult, "fresh-process retry stalled after an earlier malformed artifact contract", validationErr)
			}
			if policy.ClassifySilentRetryExhaustionUnavailable && strings.TrimSpace(retryResult.Stdout) == "" && strings.TrimSpace(retryResult.Stderr) == "" {
				return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "retry", retryResult, "provider unavailable after fresh-process artifact retry", retryErr)
			}
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "retry", "fresh-process retry stalled before producing valid artifacts", retryErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, retryResult, retryErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		emitArtifactRetryExhaustedDiagnostic(task, adapter.Provider(), err)
		if initialStructuralFailure {
			return true, acpruntime.Result{}, wrapArtifactContractFailure(adapter, task, "retry", retryResult, "artifact validation failed after an earlier malformed artifact contract", validationErr)
		}
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "retry", "artifact validation failed after fresh-process retry", err)
	}
	emitArtifactRetryCompletedDiagnostic(task, adapter.Provider())
	return true, retryResult, nil
}

func recoverAfterStall(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, runErr error) (bool, acpruntime.Result, error) {
	var stalled StallError
	if !errors.As(runErr, &stalled) {
		return false, acpruntime.Result{}, nil
	}

	policy := adapter.RecoveryPolicy(task)
	emitDiagnostic(task, "retry scheduled", stalled.Diagnostic.fields(adapter.Provider(), task, "terminate_and_validate"))
	if policy.AcceptValidArtifactsAfterStop {
		if err := adapter.ValidateArtifacts(task); err == nil {
			emitRetryCompletedDiagnostic(task, adapter.Provider(), stalled.Diagnostic.StallPhase, "artifact_only")
			return true, result, nil
		}
	}
	if !policy.RetryInvalidOrMissingArtifactsOnce {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "stall", "runtime stalled before valid artifacts were available", runErr)
	}

	retryPolicy := normalizeActivityPolicy(adapter.ActivityPolicy(task))
	if stalled.Diagnostic.StallPhase == StallPhasePreArtifact && retryPolicy.RetryPreArtifactStallWindow > 0 {
		retryPolicy.PreArtifactStallWindow = retryPolicy.RetryPreArtifactStallWindow
	}
	retryResult, retryErr := runProviderCommand(ctx, task, adapter, retryPolicy)
	if retryErr != nil {
		var retryStalled StallError
		if errors.As(retryErr, &retryStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitRetryCompletedDiagnostic(task, adapter.Provider(), retryStalled.Diagnostic.StallPhase, "fresh_process_artifact_only")
					return true, retryResult, nil
				}
			}
			emitRetryExhaustedDiagnostic(task, adapter.Provider(), retryStalled.Diagnostic, "fresh_process")
			if policy.ClassifySilentRetryExhaustionUnavailable && strings.TrimSpace(retryResult.Stdout) == "" && strings.TrimSpace(retryResult.Stderr) == "" {
				return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "retry", retryResult, "provider unavailable after fresh-process stall retry", retryErr)
			}
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "retry", "fresh-process retry stalled before producing valid artifacts", retryErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, retryResult, retryErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "retry", "artifact validation failed after stall retry", err)
	}
	emitRetryCompletedDiagnostic(task, adapter.Provider(), stalled.Diagnostic.StallPhase, "fresh_process")
	return true, retryResult, nil
}

func runProviderCommand(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, policy ActivityPolicy) (acpruntime.Result, error) {
	spec, err := adapter.CommandSpec(task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	cmd := exec.CommandContext(ctx, strings.TrimSpace(spec.Command), append([]string(nil), spec.Args...)...)
	configureCommandProcessGroup(cmd)
	if dir := strings.TrimSpace(spec.Dir); dir != "" {
		cmd.Dir = dir
	}
	if spec.Stdin != nil {
		cmd.Stdin = spec.Stdin
	}

	stdout := &commandOutputBuffer{}
	stderr := &commandOutputBuffer{}
	stdoutPipe, stdoutWriter, err := os.Pipe()
	if err != nil {
		return acpruntime.Result{}, err
	}
	stderrPipe, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutPipe.Close()
		_ = stdoutWriter.Close()
		return acpruntime.Result{}, err
	}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	if err := cmd.Start(); err != nil {
		_ = stdoutPipe.Close()
		_ = stdoutWriter.Close()
		_ = stderrPipe.Close()
		_ = stderrWriter.Close()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return acpruntime.Result{}, ctxErr
		}
		return acpruntime.Result{}, err
	}
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	defer closeCommandPipe(stdoutPipe)
	defer closeCommandPipe(stderrPipe)

	activityTracker := newCommandActivityTracker(time.Now().UTC())
	monitorCtx, stopMonitor := context.WithCancel(context.Background())
	defer stopMonitor()
	var monitorWG sync.WaitGroup
	stallCh := make(chan StallError, 1)
	if policy.MonitorArtifacts && strings.TrimSpace(task.WriteRoot) != "" {
		monitorWG.Add(1)
		go func() {
			defer monitorWG.Done()
			if stallErr, stalled := monitorArtifactStall(monitorCtx, task, activityTracker, policy); stalled {
				select {
				case stallCh <- stallErr:
				default:
				}
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
		captureErr(captureTrackedCommandStream(&activityTrackingReader{reader: stdoutPipe, tracker: activityTracker}, stdout, task, acpruntime.OutputStreamStdout))
	}()
	go func() {
		defer wg.Done()
		captureErr(captureTrackedCommandStream(&activityTrackingReader{reader: stderrPipe, tracker: activityTracker}, stderr, task, acpruntime.OutputStreamStderr))
	}()
	go func() {
		wg.Wait()
		close(streamDone)
	}()

	var waitErr error
	select {
	case <-streamDone:
		stopMonitor()
		monitorWG.Wait()
		waitErr = cmd.Wait()
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = killCommandProcessTree(cmd.Process)
		}
		waitForCommandStreams(stdoutPipe, stderrPipe, streamDone, policy.PostTerminateDrain)
		stopMonitor()
		monitorWG.Wait()
		waitErr = waitForCommandExit(cmd, policy.PostTerminateDrain)
		if waitErr == nil {
			waitErr = ctx.Err()
		}
	case stallErr := <-stallCh:
		if cmd.Process != nil {
			_ = killCommandProcessTree(cmd.Process)
		}
		waitForCommandStreams(stdoutPipe, stderrPipe, streamDone, policy.PostTerminateDrain)
		stopMonitor()
		monitorWG.Wait()
		_ = waitForCommandExit(cmd, policy.TerminateGrace)
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, stallErr
	}

	if waitErr == nil && streamErr != nil {
		waitErr = streamErr
	}
	result := acpruntime.Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if waitErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return result, ctxErr
		}
		return result, runnerdiag.BuildExecFailure(waitErr, result.Stdout, result.Stderr)
	}
	return result, nil
}

func classifyCommandFailure(adapter ProviderAdapter, task acpruntime.Task, result acpruntime.Result, err error) error {
	message, rawOutputRefs := buildEngineFailureMessage(adapter, task, "exec", err, result)
	if errors.Is(err, context.DeadlineExceeded) {
		return acpruntime.WrapRunnerErrorWithDiagnostics(adapter.Provider(), acpruntime.ErrorCodeRuntimeTimeout, message, result.Stdout, result.Stderr, rawOutputRefs, err)
	}
	if errors.Is(err, context.Canceled) {
		return acpruntime.WrapRunnerErrorWithDiagnostics(adapter.Provider(), acpruntime.ErrorCodeRunCanceled, message, result.Stdout, result.Stderr, rawOutputRefs, err)
	}
	if textHasUnavailableMarker(result.Stdout, result.Stderr, err, adapter.UnavailableMarkers()) {
		return acpruntime.WrapRunnerErrorWithDiagnostics(adapter.Provider(), acpruntime.ErrorCodeRunnerUnavailable, message, result.Stdout, result.Stderr, rawOutputRefs, err)
	}
	return acpruntime.WrapRunnerErrorWithDiagnostics(adapter.Provider(), acpruntime.ErrorCodeRunnerUnavailable, message, result.Stdout, result.Stderr, rawOutputRefs, err)
}

func classifyArtifactFailure(adapter ProviderAdapter, task acpruntime.Task, result acpruntime.Result, stage string, message string, cause error) error {
	if adapter.RecoveryPolicy(task).ClassifySilentRetryExhaustionUnavailable &&
		strings.TrimSpace(result.Stdout) == "" &&
		strings.TrimSpace(result.Stderr) == "" &&
		isMissingArtifactFailure(cause) {
		return wrapProviderUnavailable(adapter, task, stage, result, "provider unavailable before required artifacts were written", cause)
	}
	if shouldTreatArtifactFailureAsProviderUnavailable(result, cause, adapter.UnavailableMarkers()) {
		return wrapProviderUnavailable(adapter, task, stage, result, "provider unavailable before required artifacts were written", cause)
	}
	return wrapArtifactContractFailure(adapter, task, stage, result, message, cause)
}

func isMissingArtifactFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(text, "no such file or directory") ||
		strings.Contains(text, "is unavailable") ||
		strings.Contains(text, "required") && strings.Contains(text, "missing")
}

func isStructuralArtifactContractFailure(err error) bool {
	if err == nil || isMissingArtifactFailure(err) {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case text == "":
		return false
	case strings.Contains(text, "parse runtime draft manifest"):
		return true
	case strings.Contains(text, "runtime draft manifest"):
		return true
	case strings.Contains(text, "shard pack manifest"):
		return true
	case strings.Contains(text, "validator verdict"):
		return true
	case strings.Contains(text, "must point to a file"):
		return true
	default:
		return false
	}
}

func wrapArtifactContractFailure(adapter ProviderAdapter, task acpruntime.Task, stage string, result acpruntime.Result, message string, cause error) error {
	failure := failureWithMessage(message, cause, "invalid runtime artifacts")
	failureMessage, rawOutputRefs := buildEngineFailureMessage(adapter, task, stage, failure, result)
	return acpruntime.WrapRunnerErrorWithDiagnostics(
		adapter.Provider(),
		acpruntime.ErrorCodeRuntimeContract,
		fmt.Sprintf("headless provider %q produced invalid runtime artifacts: %s", adapter.Provider(), failureMessage),
		result.Stdout,
		result.Stderr,
		rawOutputRefs,
		cause,
	)
}

func wrapProviderUnavailable(adapter ProviderAdapter, task acpruntime.Task, stage string, result acpruntime.Result, message string, cause error) error {
	failure := failureWithMessage(message, cause, "provider unavailable before required artifacts were written")
	failureMessage, rawOutputRefs := buildEngineFailureMessage(adapter, task, stage, failure, result)
	return acpruntime.WrapRunnerErrorWithDiagnostics(
		adapter.Provider(),
		acpruntime.ErrorCodeRunnerUnavailable,
		fmt.Sprintf("headless provider %q became unavailable before required artifacts were written: %s", adapter.Provider(), failureMessage),
		result.Stdout,
		result.Stderr,
		rawOutputRefs,
		cause,
	)
}

func failureWithMessage(message string, cause error, fallback string) error {
	trimmed := strings.TrimSpace(message)
	switch {
	case trimmed != "" && cause != nil:
		return fmt.Errorf("%s: %w", trimmed, cause)
	case trimmed != "":
		return errors.New(trimmed)
	case cause != nil:
		return cause
	default:
		return errors.New(fallback)
	}
}

func buildEngineFailureMessage(adapter ProviderAdapter, task acpruntime.Task, stage string, failure error, result acpruntime.Result) (string, contracts.RuntimeOutputRefs) {
	diagnostics := map[string]any{
		"current_step":      strings.TrimSpace(task.StepID),
		"last_stdout_bytes": len([]byte(result.Stdout)),
		"last_stderr_bytes": len([]byte(result.Stderr)),
	}
	var stalled StallError
	if errors.As(failure, &stalled) {
		for key, value := range stalled.Diagnostic.fields(adapter.Provider(), task, "") {
			diagnostics[key] = value
		}
	}
	return BuildFailureMessage(adapter.Provider(), task, stage, failure, result.Stdout, result.Stderr, diagnostics)
}

func shouldTreatArtifactFailureAsProviderUnavailable(result acpruntime.Result, err error, markers []string) bool {
	if !textHasUnavailableMarker(result.Stdout, result.Stderr, err, markers) {
		return false
	}
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}

	text := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case text == "":
		return true
	case strings.Contains(text, "parse runtime draft manifest"):
		return false
	case strings.Contains(text, "runtime draft manifest step_id must equal"):
		return false
	case strings.Contains(text, "runtime draft manifest step_contract must equal"):
		return false
	case strings.Contains(text, "runtime draft manifest run_id must equal"):
		return false
	case strings.Contains(text, "runtime draft manifest agent_role must not be empty"):
		return false
	case strings.Contains(text, "runtime draft manifest outputs are invalid"):
		return false
	case strings.Contains(text, "shard pack manifest"):
		return false
	case strings.Contains(text, "validator verdict"):
		return false
	case strings.Contains(text, "is unavailable"):
		return true
	case strings.Contains(text, "no such file or directory"):
		return true
	case strings.Contains(text, "must point to a file"):
		return false
	default:
		return false
	}
}

func textHasUnavailableMarker(stdout string, stderr string, err error, markers []string) bool {
	text := strings.ToLower(strings.Join([]string{stdout, stderr, errorText(err)}, "\n"))
	for _, marker := range markers {
		marker = strings.ToLower(strings.TrimSpace(marker))
		if marker == "" {
			continue
		}
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func normalizeActivityPolicy(policy ActivityPolicy) ActivityPolicy {
	if policy.PreArtifactStallWindow <= 0 {
		policy.PreArtifactStallWindow = defaultPreArtifactStallWindow
	}
	if policy.RetryPreArtifactStallWindow <= 0 {
		policy.RetryPreArtifactStallWindow = defaultRetryPreArtifactWindow
	}
	if policy.PostArtifactStallWindow <= 0 {
		policy.PostArtifactStallWindow = defaultPostArtifactStallWindow
	}
	if policy.PartialArtifactStallWindow <= 0 {
		policy.PartialArtifactStallWindow = policy.PostArtifactStallWindow
	}
	if policy.PollInterval <= 0 {
		policy.PollInterval = defaultStallPollInterval
	}
	if policy.TerminateGrace <= 0 {
		policy.TerminateGrace = defaultStallTerminateGrace
	}
	if policy.PostTerminateDrain <= 0 {
		policy.PostTerminateDrain = defaultPostTerminateDrain
	}
	return policy
}

type artifactSnapshot struct {
	ArtifactObserved bool
	Valid            bool
	State            string
	AuthoredFiles    int
	LastMutation     time.Time
}

func monitorArtifactStall(ctx context.Context, task acpruntime.Task, tracker *commandActivityTracker, policy ActivityPolicy) (StallError, bool) {
	for {
		select {
		case <-ctx.Done():
			return StallError{}, false
		case <-time.After(policy.PollInterval):
		}
		snapshot := runtimeArtifactSnapshot(task)
		lastPipe := tracker.LastRead()
		lastMutation := snapshot.LastMutation
		if snapshot.ArtifactObserved {
			stallWindow := policy.PostArtifactStallWindow
			if !snapshot.Valid {
				stallWindow = policy.PartialArtifactStallWindow
			}
			if !lastPipe.IsZero() && time.Since(lastPipe) < stallWindow {
				continue
			}
			if !lastMutation.IsZero() && time.Since(lastMutation) < stallWindow {
				continue
			}
			return StallError{
				Sentinel: ErrStalledAfterArtifacts,
				Diagnostic: StallDiagnostic{
					StallPhase:            StallPhasePostArtifact,
					ArtifactState:         snapshot.State,
					AuthoredFileCount:     snapshot.AuthoredFiles,
					LastPipeActivity:      lastPipe,
					LastWriteRootMutation: lastMutation,
				},
			}, true
		}
		if !policy.MonitorPreArtifact {
			continue
		}
		if !lastMutation.IsZero() && time.Since(lastMutation) < policy.PreArtifactStallWindow {
			continue
		}
		if !lastPipe.IsZero() && time.Since(lastPipe) < policy.PreArtifactStallWindow {
			continue
		}
		return StallError{
			Sentinel: ErrStalledBeforeArtifacts,
			Diagnostic: StallDiagnostic{
				StallPhase:            StallPhasePreArtifact,
				ArtifactState:         snapshot.State,
				AuthoredFileCount:     snapshot.AuthoredFiles,
				LastPipeActivity:      lastPipe,
				LastWriteRootMutation: lastMutation,
			},
		}, true
	}
}

func runtimeArtifactSnapshot(task acpruntime.Task) artifactSnapshot {
	switch acpruntime.StepProviderKeyForStepID(task.StepID) {
	case acpruntime.StepProviderStep1Collect:
		return collectArtifactSnapshot(task.WriteRoot)
	case acpruntime.StepProviderStep3Findings:
		return validatorArtifactSnapshot(task.WriteRoot)
	default:
		if runtimedrafts.IsDraftStep(task.StepID) {
			return draftArtifactSnapshot(task)
		}
		return artifactSnapshot{}
	}
}

func collectArtifactSnapshot(writeRoot string) artifactSnapshot {
	snapshot := artifactSnapshot{}
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return snapshot
	}
	manifestPath := filepath.Join(filepath.Clean(writeRoot), ShardPackManifestFileName)
	if info, err := os.Stat(manifestPath); err == nil && !info.IsDir() {
		snapshot.ArtifactObserved = true
		snapshot.LastMutation = info.ModTime().UTC()
		snapshot.State = "present"
		if raw, readErr := os.ReadFile(manifestPath); readErr == nil {
			if _, parseErr := contracts.ParseShardPackManifest(raw); parseErr == nil {
				snapshot.Valid = true
				snapshot.State = "valid"
			} else {
				snapshot.State = "invalid"
			}
		}
	}
	entries, err := os.ReadDir(writeRoot)
	if err != nil {
		return snapshot
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == ShardPackManifestFileName {
			continue
		}
		snapshot.AuthoredFiles++
		snapshot.ArtifactObserved = true
		if info, statErr := entry.Info(); statErr == nil {
			modTime := info.ModTime().UTC()
			if modTime.After(snapshot.LastMutation) {
				snapshot.LastMutation = modTime
			}
		}
	}
	if snapshot.State == "" && snapshot.ArtifactObserved {
		snapshot.State = "partial"
	}
	return snapshot
}

func draftArtifactSnapshot(task acpruntime.Task) artifactSnapshot {
	snapshot := artifactSnapshot{}
	if strings.TrimSpace(task.WriteRoot) == "" {
		return snapshot
	}
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	if manifestFile != "" {
		manifestPath := filepath.Join(filepath.Clean(task.WriteRoot), manifestFile)
		if info, err := os.Stat(manifestPath); err == nil && !info.IsDir() {
			snapshot.ArtifactObserved = true
			snapshot.LastMutation = info.ModTime().UTC()
			snapshot.State = "present"
			if _, _, validateErr := ValidateRequiredRuntimeDraftArtifacts(task); validateErr == nil {
				snapshot.Valid = true
				snapshot.State = "valid"
			} else {
				snapshot.State = "invalid"
			}
		}
	}
	snapshot.AuthoredFiles += countFiles(task.WriteRoot, manifestFile)
	snapshot.AuthoredFiles += countFiles(task.DraftFinalRoot, "")
	snapshot.LastMutation = latestMutation(snapshot.LastMutation, latestFileMutation(task.WriteRoot))
	snapshot.LastMutation = latestMutation(snapshot.LastMutation, latestFileMutation(task.DraftFinalRoot))
	if snapshot.AuthoredFiles > 0 {
		snapshot.ArtifactObserved = true
		if snapshot.State == "" {
			snapshot.State = "partial"
		}
	}
	return snapshot
}

func validatorArtifactSnapshot(writeRoot string) artifactSnapshot {
	snapshot := artifactSnapshot{}
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return snapshot
	}
	verdictPath := filepath.Join(filepath.Clean(writeRoot), ValidatorVerdictFileName)
	if info, err := os.Stat(verdictPath); err == nil && !info.IsDir() {
		snapshot.ArtifactObserved = true
		snapshot.LastMutation = info.ModTime().UTC()
		snapshot.State = "present"
		if raw, readErr := os.ReadFile(verdictPath); readErr == nil {
			if _, parseErr := contracts.ParseValidatorVerdict(raw); parseErr == nil {
				snapshot.Valid = true
				snapshot.State = "valid"
			} else {
				snapshot.State = "invalid"
			}
		}
	}
	return snapshot
}

func countFiles(root string, except string) int {
	root = strings.TrimSpace(root)
	if root == "" {
		return 0
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || (except != "" && entry.Name() == except) {
			continue
		}
		count++
	}
	return count
}

func latestFileMutation(root string) time.Time {
	root = strings.TrimSpace(root)
	if root == "" {
		return time.Time{}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return time.Time{}
	}
	latest := time.Time{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}
		latest = latestMutation(latest, info.ModTime().UTC())
	}
	return latest
}

func latestMutation(current time.Time, candidate time.Time) time.Time {
	if candidate.After(current) {
		return candidate
	}
	return current
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

func (b *commandOutputBuffer) String() string {
	if b == nil {
		return ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func captureTrackedCommandStream(reader io.Reader, sink *commandOutputBuffer, task acpruntime.Task, stream acpruntime.OutputStream) error {
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
			if errors.Is(err, io.EOF) || isPipeClosedErr(err) {
				return nil
			}
			return err
		}
	}
}

type streamedOutputBudget struct {
	forwardedBytes int
	truncated      bool
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
			task.OnOutput(acpruntime.OutputChunk{Stream: stream, Text: line})
			continue
		}
		budget.truncated = true
		task.OnOutput(acpruntime.OutputChunk{
			Stream:    stream,
			Truncated: true,
			Text:      fmt.Sprintf("%s output truncated after %d bytes (internal safeguard)", stream, acpruntime.RuntimeOutputStreamHardCapBytes),
		})
	}
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

func emitRetryCompletedDiagnostic(task acpruntime.Task, provider acpruntime.Provider, phase StallPhase, recoveryMode string) {
	fields := map[string]any{
		"provider":      string(provider),
		"shard_id":      strings.TrimSpace(task.ShardID),
		"stall_phase":   strings.TrimSpace(string(phase)),
		"recovery_mode": strings.TrimSpace(recoveryMode),
	}
	emitDiagnostic(task, "retry completed", fields)
}

func emitRetryExhaustedDiagnostic(task acpruntime.Task, provider acpruntime.Provider, diagnostic StallDiagnostic, recoveryMode string) {
	fields := diagnostic.fields(provider, task, "")
	fields["recovery_mode"] = strings.TrimSpace(recoveryMode)
	emitDiagnostic(task, "retry exhausted", fields)
}

func emitArtifactRetryScheduledDiagnostic(task acpruntime.Task, provider acpruntime.Provider, cause error) {
	fields := map[string]any{
		"provider":      string(provider),
		"shard_id":      strings.TrimSpace(task.ShardID),
		"action":        "fresh_process_after_invalid_artifacts",
		"recovery_mode": "fresh_process",
	}
	if cause != nil {
		fields["validation_error"] = strings.TrimSpace(cause.Error())
	}
	emitDiagnostic(task, "retry scheduled", fields)
}

func emitArtifactRetryCompletedDiagnostic(task acpruntime.Task, provider acpruntime.Provider) {
	fields := map[string]any{
		"provider":      string(provider),
		"shard_id":      strings.TrimSpace(task.ShardID),
		"recovery_mode": "fresh_process",
	}
	emitDiagnostic(task, "retry completed", fields)
}

func emitArtifactRetryExhaustedDiagnostic(task acpruntime.Task, provider acpruntime.Provider, cause error) {
	fields := map[string]any{
		"provider":      string(provider),
		"shard_id":      strings.TrimSpace(task.ShardID),
		"recovery_mode": "fresh_process",
	}
	if cause != nil {
		fields["validation_error"] = strings.TrimSpace(cause.Error())
	}
	emitDiagnostic(task, "retry exhausted", fields)
}

func (d StallDiagnostic) fields(provider acpruntime.Provider, task acpruntime.Task, action string) map[string]any {
	fields := map[string]any{
		"provider":            string(provider),
		"shard_id":            strings.TrimSpace(task.ShardID),
		"stall_phase":         strings.TrimSpace(string(d.StallPhase)),
		"manifest_state":      strings.TrimSpace(d.ArtifactState),
		"authored_file_count": d.AuthoredFileCount,
	}
	if strings.TrimSpace(action) != "" {
		fields["action"] = strings.TrimSpace(action)
	}
	if !d.LastPipeActivity.IsZero() {
		fields["last_pipe_activity_at"] = d.LastPipeActivity.UTC().Format(time.RFC3339)
	}
	if !d.LastWriteRootMutation.IsZero() {
		fields["last_write_root_mutation_at"] = d.LastWriteRootMutation.UTC().Format(time.RFC3339)
	}
	return fields
}

func closeCommandPipe(file *os.File) {
	if file != nil {
		_ = file.Close()
	}
}

func waitForCommandStreams(stdoutPipe *os.File, stderrPipe *os.File, streamDone <-chan struct{}, timeout time.Duration) {
	closeCommandPipe(stdoutPipe)
	closeCommandPipe(stderrPipe)
	select {
	case <-streamDone:
	case <-time.After(timeout):
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
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return nil
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
