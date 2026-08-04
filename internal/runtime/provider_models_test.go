package runtime

import (
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestResolveProviderModelsUsesProviderDefaultWhenUnset(t *testing.T) {
	resolved, err := resolveProviderModelsWithLookup(workspace.Manifest{}, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("resolve provider models: %v", err)
	}
	for _, provider := range SupportedProviders() {
		if got := resolved.Effective[provider]; got.Model != "" || got.Effort != "" {
			t.Fatalf("expected native default values for %s, got %+v", provider, got)
		}
		if got := resolved.Source[provider]; got.Model != ProviderModelSourceDefault || got.Effort != ProviderModelSourceDefault {
			t.Fatalf("expected provider_default sources for %s, got %+v", provider, got)
		}
	}
}

func TestResolveProviderModelsEnvOverridesWorkspacePerField(t *testing.T) {
	manifest := workspace.Manifest{Runtime: &workspace.RuntimeConfig{Profile: &workspace.RuntimeProfileConfig{
		Providers: &workspace.RuntimeProvidersConfig{
			string(ProviderCodexCode): &workspace.RuntimeProviderConfig{Model: "workspace-model", Effort: "medium"},
		},
	}}}
	env := map[string]string{CodexModelEnv: "env-model"}
	resolved, err := resolveProviderModelsWithLookup(manifest, func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("resolve provider models: %v", err)
	}
	got := resolved.Effective[ProviderCodexCode]
	if got.Model != "env-model" || got.Effort != "medium" {
		t.Fatalf("expected independent env/workspace resolution, got %+v", got)
	}
	source := resolved.Source[ProviderCodexCode]
	if source.Model != ProviderModelSourceEnv || source.Effort != ProviderModelSourceWorkspace {
		t.Fatalf("unexpected sources: %+v", source)
	}
}

func TestResolveProviderModelsRejectsUnsupportedQwenEffort(t *testing.T) {
	manifest := workspace.Manifest{Runtime: &workspace.RuntimeConfig{Profile: &workspace.RuntimeProfileConfig{
		Providers: &workspace.RuntimeProvidersConfig{
			string(ProviderQwenCode): &workspace.RuntimeProviderConfig{Effort: "high"},
		},
	}}}
	_, err := resolveProviderModelsWithLookup(manifest, func(string) (string, bool) { return "", false })
	if err == nil || !strings.Contains(err.Error(), "does not support effort") {
		t.Fatalf("expected unsupported effort error, got %v", err)
	}
}

func TestValidateProviderModelConfigRejectsControlCharacters(t *testing.T) {
	err := ValidateProviderModelConfig(ProviderCodexCode, ProviderModelConfig{Model: "model\nname"})
	if err == nil || !strings.Contains(err.Error(), "control characters") {
		t.Fatalf("expected control character error, got %v", err)
	}
}
