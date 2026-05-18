package claudecode

import (
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestClaudeAdapterClassifiesSilentRetryExhaustionAsUnavailable(t *testing.T) {
	t.Parallel()

	policy := (claudeAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "init.step1.collect"})
	if !policy.RetryInvalidOrMissingArtifactsOnce {
		t.Fatalf("expected claude missing-artifact retry policy, got %+v", policy)
	}
	if !policy.ClassifySilentRetryExhaustionUnavailable {
		t.Fatalf("expected claude silent retry exhaustion to use runner_unavailable lane, got %+v", policy)
	}
	if policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("claude collect zero-output pre-artifact fail-fast behavior must remain unchanged, got %+v", policy)
	}

	policy = (claudeAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "init.step3.findings"})
	if !policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("expected claude validator zero-output pre-artifact stall to be retryable, got %+v", policy)
	}

	policy = (claudeAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "refresh.step3.findings"})
	if !policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("expected refresh validator zero-output pre-artifact stall to be retryable, got %+v", policy)
	}

	policy = (claudeAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "init.step2.asis_docs"})
	if policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("claude non-validator zero-output pre-artifact fail-fast behavior must remain unchanged, got %+v", policy)
	}
}

func TestClaudeAdapterUsesExtendedPreArtifactWindowForArtifactSteps(t *testing.T) {
	t.Parallel()

	policy := (claudeAdapter{}).ActivityPolicy(acpruntime.Task{StepID: "init.step1.collect"})
	if !policy.MonitorArtifacts || !policy.MonitorPreArtifact {
		t.Fatalf("expected claude collect artifact monitoring, got %+v", policy)
	}
	if got, want := policy.PreArtifactStallWindow, 180*time.Second; got != want {
		t.Fatalf("expected claude pre-artifact window %s, got %s", want, got)
	}

	policy = (claudeAdapter{}).ActivityPolicy(acpruntime.Task{StepID: "init.step2.asis_docs"})
	if !policy.MonitorArtifacts || !policy.MonitorPreArtifact {
		t.Fatalf("expected claude draft artifact monitoring, got %+v", policy)
	}
	if got, want := policy.PreArtifactStallWindow, 180*time.Second; got != want {
		t.Fatalf("expected claude draft pre-artifact window %s, got %s", want, got)
	}
}
