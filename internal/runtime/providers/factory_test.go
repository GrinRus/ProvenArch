package providers

import (
	"strings"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/runtime/codexcode"
	"github.com/GrinRus/ProvenArch/internal/runtime/fakeruntime"
	"github.com/GrinRus/ProvenArch/internal/runtime/qwencode"
)

func TestBuildRunnerFakeModeAlwaysUsesNeutralRuntime(t *testing.T) {
	t.Parallel()

	runner, err := BuildRunner(acpruntime.RuntimeModeFake, acpruntime.ProviderQwenCode)
	if err != nil {
		t.Fatalf("build fake runner: %v", err)
	}
	if _, ok := runner.(fakeruntime.Runner); !ok {
		t.Fatalf("expected fakeruntime.Runner, got %T", runner)
	}
}

func TestBuildRunnerHeadlessClaudeCode(t *testing.T) {
	t.Parallel()

	runner, err := BuildRunner(acpruntime.RuntimeModeHeadless, acpruntime.ProviderClaudeCode)
	if err != nil {
		t.Fatalf("build claude-code headless runner: %v", err)
	}
	if _, ok := runner.(claudecode.HeadlessRunner); !ok {
		t.Fatalf("expected claudecode.HeadlessRunner, got %T", runner)
	}
}

func TestBuildRunnerHeadlessDefaultsToClaudeCodeWhenProviderEmpty(t *testing.T) {
	t.Parallel()

	runner, err := BuildRunner(acpruntime.RuntimeModeHeadless, "")
	if err != nil {
		t.Fatalf("build headless runner with empty provider: %v", err)
	}
	if _, ok := runner.(claudecode.HeadlessRunner); !ok {
		t.Fatalf("expected claudecode.HeadlessRunner, got %T", runner)
	}
}

func TestBuildRunnerHeadlessQwenCode(t *testing.T) {
	t.Parallel()

	runner, err := BuildRunner(acpruntime.RuntimeModeHeadless, acpruntime.ProviderQwenCode)
	if err != nil {
		t.Fatalf("build qwen-code headless runner: %v", err)
	}
	if _, ok := runner.(qwencode.HeadlessRunner); !ok {
		t.Fatalf("expected qwencode.HeadlessRunner, got %T", runner)
	}
}

func TestBuildRunnerHeadlessCodexCode(t *testing.T) {
	t.Parallel()

	runner, err := BuildRunner(acpruntime.RuntimeModeHeadless, acpruntime.ProviderCodexCode)
	if err != nil {
		t.Fatalf("build codex-code headless runner: %v", err)
	}
	if _, ok := runner.(codexcode.HeadlessRunner); !ok {
		t.Fatalf("expected codexcode.HeadlessRunner, got %T", runner)
	}
}

func TestBuildRunnerRejectsUnsupportedProviderInHeadlessMode(t *testing.T) {
	t.Parallel()

	_, err := BuildRunner(acpruntime.RuntimeModeHeadless, acpruntime.Provider("bogus"))
	if err == nil {
		t.Fatalf("expected unsupported provider error")
	}
	if !strings.Contains(err.Error(), "unsupported runtime provider") {
		t.Fatalf("unexpected error: %v", err)
	}
}
