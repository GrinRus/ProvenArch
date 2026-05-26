package runtime

import (
	"testing"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestResolveProviderWithSourceCLIOverridesEnv(t *testing.T) {
	t.Parallel()

	provider, source, err := resolveProviderWithLookup("claude-code", func(string) (string, bool) {
		return "qwen-code", true
	})
	if err != nil {
		t.Fatalf("resolve provider with source: %v", err)
	}
	if provider != ProviderClaudeCode {
		t.Fatalf("expected claude-code provider, got %q", provider)
	}
	if source != ProviderSourceOverride {
		t.Fatalf("expected override source, got %q", source)
	}
}

func TestResolveStepProvidersWorkspaceOverridesGlobalFallback(t *testing.T) {
	t.Parallel()

	manifest := workspace.Manifest{
		Runtime: &workspace.RuntimeConfig{
			Profile: &workspace.RuntimeProfileConfig{
				Steps: &workspace.RuntimeStepsConfig{
					Step2AsIs: &workspace.RuntimeStepConfig{Provider: "qwen-code"},
					QA:        &workspace.RuntimeStepConfig{Provider: "codex-code"},
				},
			},
		},
	}
	resolved, err := ResolveStepProviders(manifest, ProviderClaudeCode, ProviderSourceOverride)
	if err != nil {
		t.Fatalf("resolve step providers: %v", err)
	}
	if resolved.Effective.ProviderForStep("init.step2.asis_docs") != ProviderQwenCode {
		t.Fatalf("expected workspace step2 provider qwen-code, got %q", resolved.Effective.ProviderForStep("init.step2.asis_docs"))
	}
	if resolved.Source[StepProviderStep2AsIs] != ProviderSourceWorkspace {
		t.Fatalf("expected workspace source, got %q", resolved.Source[StepProviderStep2AsIs])
	}
	if resolved.Effective.ProviderForStep("init.step3.findings") != ProviderClaudeCode {
		t.Fatalf("expected global fallback provider for step3, got %q", resolved.Effective.ProviderForStep("init.step3.findings"))
	}
	if resolved.Source[StepProviderStep3Findings] != ProviderSourceOverride {
		t.Fatalf("expected override source for step3, got %q", resolved.Source[StepProviderStep3Findings])
	}
	if resolved.Effective.ProviderForStep(StepIDQAAsk) != ProviderCodexCode {
		t.Fatalf("expected workspace qa provider codex-code, got %q", resolved.Effective.ProviderForStep(StepIDQAAsk))
	}
	if resolved.Source[StepProviderQA] != ProviderSourceWorkspace {
		t.Fatalf("expected workspace source for qa, got %q", resolved.Source[StepProviderQA])
	}
}

func TestResolveProviderWithSourceSupportsCodexCode(t *testing.T) {
	t.Parallel()

	provider, source, err := resolveProviderWithLookup("codex-code", func(string) (string, bool) {
		return "qwen-code", true
	})
	if err != nil {
		t.Fatalf("resolve provider with source: %v", err)
	}
	if provider != ProviderCodexCode {
		t.Fatalf("expected codex-code provider, got %q", provider)
	}
	if source != ProviderSourceOverride {
		t.Fatalf("expected override source, got %q", source)
	}
}

func TestStepProviderKeyForStepID(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"init.step0.constitution": StepProviderStep0Constitution,
		"init.step1.collect":      StepProviderStep1Collect,
		"refresh.step1.collect":   StepProviderStep1Collect,
		"init.step2.asis_docs":    StepProviderStep2AsIs,
		"refresh.step3.findings":  StepProviderStep3Findings,
		"init.step4.proposals":    StepProviderStep4Proposals,
		"qa.ask":                  StepProviderQA,
	}
	for stepID, want := range tests {
		if got := StepProviderKeyForStepID(stepID); got != want {
			t.Fatalf("step %q: expected %q, got %q", stepID, want, got)
		}
	}
}
