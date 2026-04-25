package providercommon

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
