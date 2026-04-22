package qwencode

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

var (
	errCollectStalledAfterArtifacts  = errors.New("collect_stalled_after_artifacts")
	errCollectStalledBeforeArtifacts = errors.New("collect_stalled_before_artifacts")
	errDraftStalledAfterArtifacts    = errors.New("draft_stalled_after_artifacts")
)

type HeadlessRunner struct {
	Command string
	Args    []string
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

	options := runQwenOptions{
		EnableCollectStallMonitor: len(r.Args) == 0 && isCollectStep(task.StepID),
		EnableDraftStallMonitor:   len(r.Args) == 0 && isDraftStep(task.StepID),
	}
	result, runErr := runQwenCommand(ctx, task, command, r.Args, options)
	if runErr != nil {
		if recovered, recoveredResult, recoveredErr := r.recoverAfterStall(ctx, task, command, options, result, runErr); recovered {
			if recoveredErr != nil {
				return acpruntime.Result{}, recoveredErr
			}
			recoveredResult.Execution = acpruntime.NewExecution(task, acpruntime.ProviderQwenCode, "headless", "succeeded", time.Now().UTC(), nil)
			return recoveredResult, nil
		}
		return acpruntime.Result{}, classifyRunFailure(task, result, runErr)
	}
	if err := repairAndValidateArtifacts(task); err != nil {
		if isProviderUnavailableText(result.Stdout, result.Stderr, err) {
			return acpruntime.Result{}, wrapArtifactProviderUnavailable(task, "contract", result, "provider unavailable before artifact validation completed", err)
		}
		return acpruntime.Result{}, wrapArtifactContractFailure(task, "contract", result, "artifact validation failed", err)
	}
	result.Execution = acpruntime.NewExecution(task, acpruntime.ProviderQwenCode, "headless", "succeeded", time.Now().UTC(), nil)
	return result, nil
}

func (r HeadlessRunner) recoverAfterStall(
	ctx context.Context,
	task acpruntime.Task,
	command string,
	options runQwenOptions,
	result acpruntime.Result,
	runErr error,
) (bool, acpruntime.Result, error) {
	var stalled collectStallError
	if !errors.As(runErr, &stalled) {
		return false, acpruntime.Result{}, nil
	}
	emitDiagnostic(task, "retry scheduled", stalled.Diagnostic.fields(task))

	if stalled.Diagnostic.StallPhase == collectStallPhasePostArtifact {
		if err := repairAndValidateArtifacts(task); err != nil {
			if isProviderUnavailableText(result.Stdout, result.Stderr, err) {
				return true, acpruntime.Result{}, wrapArtifactProviderUnavailable(task, "stall", result, "provider unavailable before artifact-only retry completed", err)
			}
			return true, acpruntime.Result{}, wrapArtifactContractFailure(task, "stall", result, "artifact-only retry after stall failed", err)
		}
		emitRetryCompletedDiagnostic(task, stalled.Diagnostic.StallPhase, "artifact_only")
		return true, result, nil
	}

	retryResult, retryErr := runQwenCommand(ctx, task, command, r.Args, stallRetryOptions(options, stalled.Diagnostic))
	if retryErr != nil {
		var retryStalled collectStallError
		if errors.As(retryErr, &retryStalled) {
			emitRetryExhaustedDiagnostic(task, retryStalled.Diagnostic, "fresh_process")
			if isProviderUnavailableText(retryResult.Stdout, retryResult.Stderr, retryErr) {
				return true, acpruntime.Result{}, wrapArtifactProviderUnavailable(task, "retry", retryResult, "provider unavailable after fresh-process stall retry", retryErr)
			}
			return true, acpruntime.Result{}, wrapArtifactContractFailure(task, "retry", retryResult, "fresh-process retry stalled before producing required artifacts", retryErr)
		}
		return true, acpruntime.Result{}, classifyRunFailure(task, retryResult, retryErr)
	}
	if err := repairAndValidateArtifacts(task); err != nil {
		if isProviderUnavailableText(retryResult.Stdout, retryResult.Stderr, err) {
			return true, acpruntime.Result{}, wrapArtifactProviderUnavailable(task, "retry", retryResult, "provider unavailable before retry artifact validation completed", err)
		}
		return true, acpruntime.Result{}, wrapArtifactContractFailure(task, "retry", retryResult, "artifact validation failed after stall retry", err)
	}
	emitRetryCompletedDiagnostic(task, stalled.Diagnostic.StallPhase, "fresh_process")
	return true, retryResult, nil
}

func stallRetryOptions(options runQwenOptions, diagnostic collectStallDiagnostic) runQwenOptions {
	retryOptions := options
	if diagnostic.StallPhase == collectStallPhasePreArtifact {
		// The second attempt gets a wider but still bounded pre-artifact grace
		// window, while preserving post-artifact stall recovery.
		retryOptions.CollectPreArtifactWindow = collectRetryPreArtifactWindow
	}
	return retryOptions
}
