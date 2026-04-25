package qwencode

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/providercommon"
)

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

func emitDiagnostic(task acpruntime.Task, message string, fields map[string]any) {
	if task.OnDiagnostic == nil {
		return
	}
	task.OnDiagnostic(acpruntime.DiagnosticEvent{
		Message: strings.TrimSpace(message),
		Fields:  fields,
	})
}

func emitRetryCompletedDiagnostic(task acpruntime.Task, phase collectStallPhase, recoveryMode string) {
	fields := map[string]any{
		"provider":      string(acpruntime.ProviderQwenCode),
		"shard_id":      strings.TrimSpace(task.ShardID),
		"stall_phase":   strings.TrimSpace(string(phase)),
		"recovery_mode": strings.TrimSpace(recoveryMode),
	}
	emitDiagnostic(task, "retry completed", fields)
}

func emitRetryExhaustedDiagnostic(task acpruntime.Task, diagnostic collectStallDiagnostic, recoveryMode string) {
	fields := diagnostic.fields(task)
	fields["recovery_mode"] = strings.TrimSpace(recoveryMode)
	emitDiagnostic(task, "retry exhausted", fields)
}

func classifyRunFailure(task acpruntime.Task, result acpruntime.Result, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		message, rawOutputRefs := buildFailureMessage(task, "exec", err, result)
		return acpruntime.WrapRunnerErrorWithDiagnostics(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRuntimeTimeout,
			message,
			result.Stdout,
			result.Stderr,
			rawOutputRefs,
			err,
		)
	}
	if errors.Is(err, context.Canceled) {
		message, rawOutputRefs := buildFailureMessage(task, "exec", err, result)
		return acpruntime.WrapRunnerErrorWithDiagnostics(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunCanceled,
			message,
			result.Stdout,
			result.Stderr,
			rawOutputRefs,
			err,
		)
	}
	var stalled collectStallError
	if errors.As(err, &stalled) {
		emitDiagnostic(task, "retry scheduled", stalled.Diagnostic.fields(task))
		return wrapArtifactContractFailure(task, "stall", result, "runtime stalled after artifact writes", err)
	}
	if isProviderUnavailableText(result.Stdout, result.Stderr, err) {
		message, rawOutputRefs := buildFailureMessage(task, "exec", err, result)
		return acpruntime.WrapRunnerErrorWithDiagnostics(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			message,
			result.Stdout,
			result.Stderr,
			rawOutputRefs,
			err,
		)
	}
	message, rawOutputRefs := buildFailureMessage(task, "exec", err, result)
	return acpruntime.WrapRunnerErrorWithDiagnostics(
		acpruntime.ProviderQwenCode,
		acpruntime.ErrorCodeRunnerUnavailable,
		message,
		result.Stdout,
		result.Stderr,
		rawOutputRefs,
		err,
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
		failure = errors.New("invalid runtime artifacts")
	}
	failureMessage, rawOutputRefs := buildFailureMessage(task, stage, failure, result)
	return acpruntime.WrapRunnerErrorWithDiagnostics(
		acpruntime.ProviderQwenCode,
		acpruntime.ErrorCodeRuntimeContract,
		fmt.Sprintf("headless provider %q produced invalid runtime artifacts: %s", acpruntime.ProviderQwenCode, failureMessage),
		result.Stdout,
		result.Stderr,
		rawOutputRefs,
		cause,
	)
}

func wrapArtifactProviderUnavailable(task acpruntime.Task, stage string, result acpruntime.Result, message string, cause error) error {
	failure := cause
	trimmed := strings.TrimSpace(message)
	switch {
	case trimmed != "" && cause != nil:
		failure = fmt.Errorf("%s: %w", trimmed, cause)
	case trimmed != "":
		failure = errors.New(trimmed)
	case failure == nil:
		failure = errors.New("provider unavailable before required artifacts were written")
	}
	failureMessage, rawOutputRefs := buildFailureMessage(task, stage, failure, result)
	return acpruntime.WrapRunnerErrorWithDiagnostics(
		acpruntime.ProviderQwenCode,
		acpruntime.ErrorCodeRunnerUnavailable,
		fmt.Sprintf("headless provider %q became unavailable before required artifacts were written: %s", acpruntime.ProviderQwenCode, failureMessage),
		result.Stdout,
		result.Stderr,
		rawOutputRefs,
		cause,
	)
}

func shouldTreatArtifactFailureAsProviderUnavailable(result acpruntime.Result, err error) bool {
	if !isProviderUnavailableText(result.Stdout, result.Stderr, err) {
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

func buildFailureMessage(task acpruntime.Task, stage string, failure error, result acpruntime.Result) (string, contracts.RuntimeOutputRefs) {
	diagnostics := map[string]any{
		"current_step":      strings.TrimSpace(task.StepID),
		"last_stdout_bytes": len([]byte(result.Stdout)),
		"last_stderr_bytes": len([]byte(result.Stderr)),
	}
	var stalled collectStallError
	if errors.As(failure, &stalled) {
		for key, value := range stalled.Diagnostic.fields(task) {
			diagnostics[key] = value
		}
	}
	return providercommon.BuildFailureMessage(acpruntime.ProviderQwenCode, task, stage, failure, result.Stdout, result.Stderr, diagnostics)
}

func isProviderUnavailableText(stdout string, stderr string, err error) bool {
	text := strings.ToLower(strings.Join([]string{stdout, stderr, errorText(err)}, "\n"))
	markers := []string{
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
	for _, marker := range markers {
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
