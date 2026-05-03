package providercommon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

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
	if shouldClassifySilentRetryExhaustionUnavailable(adapter.RecoveryPolicy(task), task, result) && isMissingArtifactFailure(cause) {
		return wrapProviderUnavailable(adapter, task, stage, result, "provider unavailable before required artifacts were written", cause)
	}
	if shouldTreatArtifactFailureAsProviderUnavailable(result, cause, adapter.UnavailableMarkers()) {
		return wrapProviderUnavailable(adapter, task, stage, result, "provider unavailable before required artifacts were written", cause)
	}
	return wrapArtifactContractFailure(adapter, task, stage, result, message, cause)
}

func shouldClassifySilentRetryExhaustionUnavailable(policy RecoveryPolicy, task acpruntime.Task, result acpruntime.Result) bool {
	snapshot := runtimeArtifactSnapshot(task)
	return policy.ClassifySilentRetryExhaustionUnavailable &&
		strings.TrimSpace(result.Stdout) == "" &&
		strings.TrimSpace(result.Stderr) == "" &&
		!snapshot.ArtifactObserved &&
		snapshot.AuthoredFiles == 0
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
	for key, value := range runtimeArtifactSnapshot(task).diagnosticFields() {
		diagnostics[key] = value
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
