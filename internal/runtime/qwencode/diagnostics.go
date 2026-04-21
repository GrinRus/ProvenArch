package qwencode

import (
	"errors"
	"fmt"
	"strings"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultextractor"
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

func wrapRuntimeDraftContractFailure(task acpruntime.Task, stage string, result acpruntime.Result, message string, cause error) error {
	failure := cause
	trimmed := strings.TrimSpace(message)
	switch {
	case trimmed != "" && cause != nil:
		failure = fmt.Errorf("%s: %w", trimmed, cause)
	case trimmed != "":
		failure = errors.New(trimmed)
	case failure == nil:
		failure = errors.New("invalid runtime draft artifacts")
	}
	failureMessage := buildFailureMessage(task, stage, failure, result)
	return acpruntime.WrapRunnerErrorWithOutput(
		acpruntime.ProviderQwenCode,
		acpruntime.ErrorCodeRunnerParseFailed,
		fmt.Sprintf("headless provider %q produced invalid runtime draft artifacts: %s", acpruntime.ProviderQwenCode, failureMessage),
		result.Stdout,
		result.Stderr,
		cause,
	)
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
