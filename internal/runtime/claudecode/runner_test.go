package claudecode

import (
	"testing"

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
