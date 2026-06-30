package claudecode

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestClaudeCommandNamePrefersEnvOverride(t *testing.T) {
	t.Setenv("ACP_CLAUDE_CMD", "/custom/claude")

	runner := HeadlessRunner{}
	if got, want := runner.commandName(), "/custom/claude"; got != want {
		t.Fatalf("expected env override command %q, got %q", want, got)
	}
}

func TestClaudeCommandNamePrefersClaudeBinaryBeforeLegacyClaudeCode(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "claude"))
	writeExecutable(t, filepath.Join(binDir, "claude-code"))
	t.Setenv("PATH", binDir)

	runner := HeadlessRunner{}
	if got, want := runner.commandName(), "claude"; got != want {
		t.Fatalf("expected default command %q, got %q", want, got)
	}
}

func TestClaudeCommandNameFallsBackToLegacyClaudeCode(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "claude-code"))
	t.Setenv("PATH", binDir)

	runner := HeadlessRunner{}
	if got, want := runner.commandName(), "claude-code"; got != want {
		t.Fatalf("expected legacy fallback command %q, got %q", want, got)
	}
}

func TestClaudeAdapterClassifiesSilentRetryExhaustionAsUnavailable(t *testing.T) {
	t.Parallel()

	policy := (claudeAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "init.step1.collect"})
	if !policy.RetryInvalidOrMissingArtifactsOnce {
		t.Fatalf("expected claude missing-artifact retry policy, got %+v", policy)
	}
	if !policy.ClassifySilentRetryExhaustionUnavailable {
		t.Fatalf("expected claude silent retry exhaustion to use runner_unavailable lane, got %+v", policy)
	}
	if !policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("expected claude collect zero-output pre-artifact stall to be retryable, got %+v", policy)
	}

	policy = (claudeAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "init.step0.constitution"})
	if !policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("expected claude constitution zero-output pre-artifact stall to be retryable, got %+v", policy)
	}

	policy = (claudeAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "init.step3.findings"})
	if !policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("expected claude validator zero-output pre-artifact stall to be retryable, got %+v", policy)
	}

	policy = (claudeAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "refresh.step3.findings"})
	if !policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("expected refresh validator zero-output pre-artifact stall to be retryable, got %+v", policy)
	}

	policy = (claudeAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "init.step4.proposals"})
	if !policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("expected claude proposals zero-output pre-artifact stall to be retryable, got %+v", policy)
	}

	policy = (claudeAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "refresh.step4.proposals"})
	if !policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("expected refresh proposals zero-output pre-artifact stall to be retryable, got %+v", policy)
	}

	policy = (claudeAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "init.step2.asis_docs"})
	if policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("claude as-is zero-output pre-artifact fail-fast behavior must remain unchanged, got %+v", policy)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func TestClaudeAdapterUsesExtendedPreArtifactWindowForArtifactSteps(t *testing.T) {
	t.Parallel()

	policy := (claudeAdapter{}).ActivityPolicy(acpruntime.Task{StepID: "init.step1.collect"})
	if !policy.MonitorArtifacts || !policy.MonitorPreArtifact {
		t.Fatalf("expected claude collect artifact monitoring, got %+v", policy)
	}
	if got, want := policy.PreArtifactStallWindow, 5*time.Minute; got != want {
		t.Fatalf("expected claude pre-artifact window %s, got %s", want, got)
	}
	if got, want := policy.RetryPreArtifactStallWindow, 5*time.Minute; got != want {
		t.Fatalf("expected claude retry pre-artifact window %s, got %s", want, got)
	}
	if got, want := policy.PostArtifactStallWindow, 90*time.Second; got != want {
		t.Fatalf("expected claude collect post-artifact enrichment window %s, got %s", want, got)
	}
	if got, want := policy.PartialArtifactStallWindow, 90*time.Second; got != want {
		t.Fatalf("expected claude collect partial-artifact enrichment window %s, got %s", want, got)
	}

	policy = (claudeAdapter{}).ActivityPolicy(acpruntime.Task{StepID: "init.step2.asis_docs"})
	if !policy.MonitorArtifacts || !policy.MonitorPreArtifact {
		t.Fatalf("expected claude draft artifact monitoring, got %+v", policy)
	}
	if got, want := policy.PreArtifactStallWindow, 180*time.Second; got != want {
		t.Fatalf("expected claude draft pre-artifact window %s, got %s", want, got)
	}
	if got, want := policy.RetryPreArtifactStallWindow, time.Duration(0); got != want {
		t.Fatalf("expected claude draft retry pre-artifact window %s, got %s", want, got)
	}
	if policy.PostArtifactStallWindow != 0 || policy.PartialArtifactStallWindow != 0 {
		t.Fatalf("draft steps must keep shared post-artifact defaults, got %+v", policy)
	}
}
