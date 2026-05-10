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
