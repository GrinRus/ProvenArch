package orchestrator

import (
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestObservedModelIDsExtractsRuntimeTelemetry(t *testing.T) {
	t.Parallel()

	models := observedModelIDs(`{"model":"claude-opus-4-5-20251101"} modelUsage.kimi-for-coding model = "gpt-5.5" "{\"model\":\"qwen3-coder\"}"`)
	if len(models) != 4 {
		t.Fatalf("expected four unique observed models, got %v", models)
	}
	if !containsModel(models, "qwen3-coder") {
		t.Fatalf("expected escaped JSON model telemetry to be extracted, got %v", models)
	}
}

func TestProviderModelMismatchFlagsCrossFamilyTelemetry(t *testing.T) {
	t.Parallel()

	if !providerModelMismatch(acpruntime.ProviderQwenCode, []string{"claude-opus-4-5-20251101"}) {
		t.Fatal("expected qwen provider to flag claude model telemetry")
	}
	if !providerModelMismatch(acpruntime.ProviderClaudeCode, []string{"kimi-for-coding"}) {
		t.Fatal("expected claude provider to flag kimi model telemetry")
	}
	if providerModelMismatch(acpruntime.ProviderClaudeCode, []string{"claude-sonnet-4"}) {
		t.Fatal("expected claude model telemetry to be accepted for claude provider")
	}
}

func containsModel(models []string, want string) bool {
	for _, model := range models {
		if model == want {
			return true
		}
	}
	return false
}
