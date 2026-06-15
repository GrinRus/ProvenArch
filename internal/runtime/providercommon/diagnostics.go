package providercommon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
)

func WrapCommandFailure(provider acpruntime.Provider, task acpruntime.Task, stdout string, stderr string, cause error) error {
	message, rawOutputRefs := BuildFailureMessage(provider, task, "exec", cause, stdout, stderr, nil)
	if errors.Is(cause, context.DeadlineExceeded) {
		return acpruntime.WrapRunnerErrorWithDiagnostics(provider, acpruntime.ErrorCodeRuntimeTimeout, message, stdout, stderr, rawOutputRefs, cause)
	}
	if errors.Is(cause, context.Canceled) {
		return acpruntime.WrapRunnerErrorWithDiagnostics(provider, acpruntime.ErrorCodeRunCanceled, message, stdout, stderr, rawOutputRefs, cause)
	}
	return acpruntime.WrapRunnerErrorWithDiagnostics(provider, acpruntime.ErrorCodeRunnerUnavailable, message, stdout, stderr, rawOutputRefs, cause)
}

func WrapContractFailure(provider acpruntime.Provider, task acpruntime.Task, stdout string, stderr string, cause error) error {
	message, rawOutputRefs := BuildFailureMessage(provider, task, "contract", cause, stdout, stderr, nil)
	return acpruntime.WrapRunnerErrorWithDiagnostics(provider, acpruntime.ErrorCodeRuntimeContract, message, stdout, stderr, rawOutputRefs, cause)
}

func BuildFailureMessage(
	provider acpruntime.Provider,
	task acpruntime.Task,
	stage string,
	cause error,
	stdout string,
	stderr string,
	diagnostics map[string]any,
) (string, contracts.RuntimeOutputRefs) {
	base := "unknown failure"
	if cause != nil {
		base = strings.TrimSpace(cause.Error())
	}
	if base == "" {
		base = "unknown failure"
	}
	stage = NonEmptyStage(stage)
	artifacts, err := runnerdiag.WriteFailureArtifactsWithMetadata(task, provider, stdout, stderr, diagnostics)
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

func emitZeroOutputPreArtifactStallDiagnostic(task acpruntime.Task, provider acpruntime.Provider, diagnostic StallDiagnostic, recoveryMode string) {
	fields := diagnostic.fields(provider, task, "classify_runner_unavailable")
	fields["recovery_mode"] = strings.TrimSpace(recoveryMode)
	fields["zero_output_pre_artifact_stall"] = true
	emitDiagnostic(task, "zero-output pre-artifact stall classified unavailable", fields)
}

func emitZeroOutputPreArtifactStallRetryDiagnostic(task acpruntime.Task, provider acpruntime.Provider, diagnostic StallDiagnostic) {
	fields := diagnostic.fields(provider, task, "warn_and_retry")
	fields["recovery_mode"] = "fresh_process"
	fields["severity"] = "warning"
	fields["zero_output_pre_artifact_stall"] = true
	emitDiagnostic(task, "zero-output pre-artifact stall will retry", fields)
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

func emitStallRetryScheduledDiagnostic(task acpruntime.Task, provider acpruntime.Provider, diagnostic StallDiagnostic) {
	fields := map[string]any{
		"provider":            string(provider),
		"shard_id":            strings.TrimSpace(task.ShardID),
		"action":              "fresh_process_after_stall",
		"recovery_mode":       "fresh_process",
		"initial_stall_phase": strings.TrimSpace(string(diagnostic.StallPhase)),
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

func emitCollectManifestRepairScheduledDiagnostic(task acpruntime.Task, provider acpruntime.Provider, snapshot artifactSnapshot, cause error) {
	fields := snapshot.diagnosticFields()
	fields["provider"] = string(provider)
	fields["shard_id"] = strings.TrimSpace(task.ShardID)
	fields["action"] = "manifest_only_repair"
	fields["recovery_mode"] = "collect_manifest_repair"
	if cause != nil {
		fields["validation_error"] = strings.TrimSpace(cause.Error())
	}
	emitDiagnostic(task, "collect manifest repair scheduled", fields)
}

func emitCollectManifestRepairCompletedDiagnostic(task acpruntime.Task, provider acpruntime.Provider, phase StallPhase) {
	fields := map[string]any{
		"provider":      string(provider),
		"shard_id":      strings.TrimSpace(task.ShardID),
		"recovery_mode": "collect_manifest_repair",
	}
	if phase != "" {
		fields["stall_phase"] = strings.TrimSpace(string(phase))
	}
	emitDiagnostic(task, "collect manifest repair completed", fields)
}

func emitCollectManifestRepairExhaustedDiagnostic(task acpruntime.Task, provider acpruntime.Provider, diagnostic StallDiagnostic, cause error) {
	fields := diagnostic.fields(provider, task, "")
	fields["recovery_mode"] = "collect_manifest_repair"
	if cause != nil {
		fields["validation_error"] = strings.TrimSpace(cause.Error())
	}
	emitDiagnostic(task, "collect manifest repair exhausted", fields)
}

func emitCollectManifestDeterministicRecoveryCompletedDiagnostic(task acpruntime.Task, provider acpruntime.Provider, report collectManifestRuntimeRecoveryReport) {
	fields := map[string]any{
		"provider":       string(provider),
		"shard_id":       strings.TrimSpace(task.ShardID),
		"recovery_mode":  "collect_manifest_runtime_recovery",
		"document_count": report.DocumentCount,
		"entity_count":   report.EntityCount,
		"edge_count":     report.EdgeCount,
		"evidence_path":  strings.TrimSpace(report.EvidencePath),
	}
	emitDiagnostic(task, "collect manifest runtime recovery completed", fields)
}

func emitCollectManifestDeterministicRecoveryFailedDiagnostic(task acpruntime.Task, provider acpruntime.Provider, cause error) {
	fields := map[string]any{
		"provider":      string(provider),
		"shard_id":      strings.TrimSpace(task.ShardID),
		"recovery_mode": "collect_manifest_runtime_recovery",
	}
	if cause != nil {
		fields["validation_error"] = strings.TrimSpace(cause.Error())
	}
	emitDiagnostic(task, "collect manifest runtime recovery failed", fields)
}

func emitFocusedArtifactRepairScheduledDiagnostic(task acpruntime.Task, provider acpruntime.Provider, mode string, stage string, snapshot artifactSnapshot, cause error) {
	fields := snapshot.diagnosticFields()
	fields["provider"] = string(provider)
	fields["shard_id"] = strings.TrimSpace(task.ShardID)
	fields["step_id"] = strings.TrimSpace(task.StepID)
	fields["action"] = "focused_artifact_repair"
	fields["recovery_mode"] = strings.TrimSpace(mode)
	if strings.TrimSpace(stage) != "" {
		fields["recovery_stage"] = strings.TrimSpace(stage)
	}
	if cause != nil {
		fields["validation_error"] = strings.TrimSpace(cause.Error())
	}
	emitDiagnostic(task, "focused artifact repair scheduled", fields)
}

func emitFocusedArtifactRepairCompletedDiagnostic(task acpruntime.Task, provider acpruntime.Provider, mode string, phase StallPhase) {
	fields := map[string]any{
		"provider":      string(provider),
		"shard_id":      strings.TrimSpace(task.ShardID),
		"step_id":       strings.TrimSpace(task.StepID),
		"recovery_mode": strings.TrimSpace(mode),
	}
	if phase != "" {
		fields["stall_phase"] = strings.TrimSpace(string(phase))
	}
	emitDiagnostic(task, "focused artifact repair completed", fields)
}

func emitFocusedArtifactRepairRetryScheduledDiagnostic(task acpruntime.Task, provider acpruntime.Provider, mode string, cause error) {
	fields := map[string]any{
		"provider":      string(provider),
		"shard_id":      strings.TrimSpace(task.ShardID),
		"step_id":       strings.TrimSpace(task.StepID),
		"action":        "retry_transient_provider_unavailable",
		"recovery_mode": strings.TrimSpace(mode),
		"severity":      "warning",
	}
	if cause != nil {
		fields["validation_error"] = strings.TrimSpace(cause.Error())
	}
	emitDiagnostic(task, "focused artifact repair retry scheduled", fields)
}

func emitFocusedArtifactRepairExhaustedDiagnostic(task acpruntime.Task, provider acpruntime.Provider, mode string, diagnostic StallDiagnostic, cause error) {
	fields := diagnostic.fields(provider, task, "")
	fields["step_id"] = strings.TrimSpace(task.StepID)
	fields["recovery_mode"] = strings.TrimSpace(mode)
	if cause != nil {
		fields["validation_error"] = strings.TrimSpace(cause.Error())
	}
	emitDiagnostic(task, "focused artifact repair exhausted", fields)
}

func emitDraftArtifactRepairSnapshotDiagnostic(task acpruntime.Task, provider acpruntime.Provider, outcome string, snapshot artifactSnapshot) {
	fields := snapshot.diagnosticFields()
	fields["provider"] = string(provider)
	fields["shard_id"] = strings.TrimSpace(task.ShardID)
	fields["step_id"] = strings.TrimSpace(task.StepID)
	fields["recovery_mode"] = "draft_artifact_repair"
	fields["outcome"] = strings.TrimSpace(outcome)
	emitDiagnostic(task, "draft artifact repair snapshot", fields)
}

func emitDraftArtifactEnrichmentSnapshotDiagnostic(task acpruntime.Task, provider acpruntime.Provider, outcome string, snapshot artifactSnapshot) {
	fields := snapshot.diagnosticFields()
	fields["provider"] = string(provider)
	fields["shard_id"] = strings.TrimSpace(task.ShardID)
	fields["step_id"] = strings.TrimSpace(task.StepID)
	fields["recovery_mode"] = "draft_artifact_enrichment"
	fields["outcome"] = strings.TrimSpace(outcome)
	emitDiagnostic(task, "draft artifact enrichment snapshot", fields)
}

func (d StallDiagnostic) fields(provider acpruntime.Provider, task acpruntime.Task, action string) map[string]any {
	fields := map[string]any{
		"provider":            string(provider),
		"shard_id":            strings.TrimSpace(task.ShardID),
		"stall_phase":         strings.TrimSpace(string(d.StallPhase)),
		"manifest_state":      strings.TrimSpace(d.ArtifactState),
		"artifact_observed":   d.ArtifactObserved,
		"authored_file_count": d.AuthoredFileCount,
		"stdout_bytes":        d.StdoutBytes,
		"stderr_bytes":        d.StderrBytes,
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
