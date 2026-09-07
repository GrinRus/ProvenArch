package runtime

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const (
	ClaudeModelEnv             = "ACP_CLAUDE_MODEL"
	ClaudeEffortEnv            = "ACP_CLAUDE_EFFORT"
	QwenModelEnv               = "ACP_QWEN_MODEL"
	CodexModelEnv              = "ACP_CODEX_MODEL"
	CodexReasoningEffortEnv    = "ACP_CODEX_REASONING_EFFORT"
	ProviderDefaultModelSource = "provider_default"
)

type ProviderModelSource string

const (
	ProviderModelSourceDefault   ProviderModelSource = ProviderDefaultModelSource
	ProviderModelSourceWorkspace ProviderModelSource = "workspace"
	ProviderModelSourceEnv       ProviderModelSource = "env"
	// ProviderModelSourceTaskPreset is used only by an immutable Task/Attempt
	// snapshot when the Task explicitly overrides the resolved profile.
	ProviderModelSourceTaskPreset ProviderModelSource = "task_preset"
)

type ProviderModelConfig struct {
	Model  string `json:"model,omitempty"`
	Effort string `json:"effort,omitempty"`
}

type ProviderModelFieldSources struct {
	Model  ProviderModelSource `json:"model"`
	Effort ProviderModelSource `json:"effort"`
}

type ProviderModelValues map[Provider]ProviderModelConfig
type ProviderModelSources map[Provider]ProviderModelFieldSources

type ProviderModelResolution struct {
	Persisted workspace.RuntimeProvidersConfig `json:"persisted"`
	Effective ProviderModelValues              `json:"effective"`
	Source    ProviderModelSources             `json:"source"`
}

type providerModelEnvLookup func(string) (string, bool)

func SupportedProviders() []Provider {
	return []Provider{ProviderClaudeCode, ProviderQwenCode, ProviderCodexCode}
}

func ProviderModelEnv(provider Provider) (modelEnv string, effortEnv string) {
	switch provider {
	case ProviderClaudeCode:
		return ClaudeModelEnv, ClaudeEffortEnv
	case ProviderQwenCode:
		return QwenModelEnv, ""
	case ProviderCodexCode:
		return CodexModelEnv, CodexReasoningEffortEnv
	default:
		return "", ""
	}
}

func SupportedEfforts(provider Provider) []string {
	switch provider {
	case ProviderClaudeCode:
		return []string{"low", "medium", "high", "max"}
	case ProviderCodexCode:
		return []string{"none", "low", "medium", "high", "xhigh", "max"}
	default:
		return nil
	}
}

func ResolveProviderModels(manifest workspace.Manifest) (ProviderModelResolution, error) {
	return resolveProviderModelsWithLookup(manifest, os.LookupEnv)
}

func resolveProviderModelsWithLookup(manifest workspace.Manifest, lookup providerModelEnvLookup) (ProviderModelResolution, error) {
	persisted := workspace.RuntimeProvidersConfig{}
	if manifest.Runtime != nil && manifest.Runtime.Profile != nil && manifest.Runtime.Profile.Providers != nil {
		persisted = cloneWorkspaceProviderModels(*manifest.Runtime.Profile.Providers)
	}
	resolution := ProviderModelResolution{
		Persisted: persisted,
		Effective: ProviderModelValues{},
		Source:    ProviderModelSources{},
	}
	for _, provider := range SupportedProviders() {
		configured := workspace.RuntimeProviderConfig{}
		if value := persisted[string(provider)]; value != nil {
			configured = *value
		}
		modelEnv, effortEnv := ProviderModelEnv(provider)
		model, modelSource := resolveProviderModelField(configured.Model, modelEnv, lookup)
		effort, effortSource := resolveProviderModelField(configured.Effort, effortEnv, lookup)
		effort = strings.ToLower(strings.TrimSpace(effort))
		if err := ValidateProviderModelConfig(provider, ProviderModelConfig{Model: model, Effort: effort}); err != nil {
			return ProviderModelResolution{}, err
		}
		resolution.Effective[provider] = ProviderModelConfig{Model: model, Effort: effort}
		resolution.Source[provider] = ProviderModelFieldSources{Model: modelSource, Effort: effortSource}
	}
	return resolution, nil
}

func resolveProviderModelField(persisted string, envName string, lookup providerModelEnvLookup) (string, ProviderModelSource) {
	if envName != "" {
		if value, ok := lookup(envName); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), ProviderModelSourceEnv
		}
	}
	if value := strings.TrimSpace(persisted); value != "" {
		return value, ProviderModelSourceWorkspace
	}
	return "", ProviderModelSourceDefault
}

func ValidateProviderModelConfig(provider Provider, config ProviderModelConfig) error {
	model := strings.TrimSpace(config.Model)
	if len(model) > 256 {
		return fmt.Errorf("runtime.profile.providers.%s.model must be at most 256 characters", provider)
	}
	for _, char := range model {
		if unicode.IsControl(char) {
			return fmt.Errorf("runtime.profile.providers.%s.model must not contain control characters", provider)
		}
	}
	effort := strings.TrimSpace(strings.ToLower(config.Effort))
	if effort == "" {
		return nil
	}
	allowed := SupportedEfforts(provider)
	if len(allowed) == 0 {
		return fmt.Errorf("runtime provider %s does not support effort", provider)
	}
	for _, candidate := range allowed {
		if effort == candidate {
			return nil
		}
	}
	return fmt.Errorf("runtime.profile.providers.%s.effort %q is invalid (allowed: %s)", provider, effort, strings.Join(allowed, ", "))
}

func cloneWorkspaceProviderModels(source workspace.RuntimeProvidersConfig) workspace.RuntimeProvidersConfig {
	clone := workspace.RuntimeProvidersConfig{}
	for provider, config := range source {
		if config == nil {
			continue
		}
		copy := *config
		clone[provider] = &copy
	}
	return clone
}
