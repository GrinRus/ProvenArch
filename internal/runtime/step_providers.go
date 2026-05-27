package runtime

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type ProviderSource string

const (
	ProviderSourceDefault   ProviderSource = "default"
	ProviderSourceWorkspace ProviderSource = "workspace"
	ProviderSourceEnv       ProviderSource = "env"
	ProviderSourceOverride  ProviderSource = "override"
)

const (
	StepProviderStep0Constitution = "step0_constitution"
	StepProviderStep1Collect      = "step1_collect"
	StepProviderStep2AsIs         = "step2_as_is"
	StepProviderStep3Findings     = "step3_findings"
	StepProviderStep4Proposals    = "step4_proposals"
	StepProviderQA                = "qa"
	StepIDQAAsk                   = "qa.ask"
)

var orderedStepProviderKeys = []string{
	StepProviderStep0Constitution,
	StepProviderStep1Collect,
	StepProviderStep2AsIs,
	StepProviderStep3Findings,
	StepProviderStep4Proposals,
	StepProviderQA,
}

type StepProviderValues map[string]Provider
type StepProviderSources map[string]ProviderSource

type StepProviderResolution struct {
	Persisted workspace.RuntimeStepsConfig `json:"persisted"`
	Effective StepProviderValues           `json:"effective"`
	Source    StepProviderSources          `json:"source"`
}

func ResolveProviderWithSource(cliValue string) (Provider, ProviderSource, error) {
	return resolveProviderWithLookup(cliValue, os.LookupEnv)
}

func resolveProviderWithLookup(cliValue string, lookup envLookup) (Provider, ProviderSource, error) {
	value := strings.TrimSpace(cliValue)
	if value != "" {
		provider, err := ParseProvider(value)
		if err != nil {
			return "", "", err
		}
		return provider, ProviderSourceOverride, nil
	}
	if envValue, ok := lookup(RuntimeProviderEnv); ok {
		provider, err := ParseProvider(envValue)
		if err != nil {
			return "", "", err
		}
		return provider, ProviderSourceEnv, nil
	}
	return ProviderClaudeCode, ProviderSourceDefault, nil
}

func ResolveStepProviders(manifest workspace.Manifest, globalProvider Provider, globalSource ProviderSource) (StepProviderResolution, error) {
	return resolveStepProvidersWithGlobal(manifest, globalProvider, globalSource)
}

func resolveStepProvidersWithGlobal(manifest workspace.Manifest, globalProvider Provider, globalSource ProviderSource) (StepProviderResolution, error) {
	if globalProvider == "" {
		globalProvider = ProviderClaudeCode
	}
	if globalSource == "" {
		globalSource = ProviderSourceDefault
	}

	persisted := workspace.RuntimeStepsConfig{}
	if manifest.Runtime != nil && manifest.Runtime.Profile != nil && manifest.Runtime.Profile.Steps != nil {
		persisted = *manifest.Runtime.Profile.Steps
	}
	effective := StepProviderValues{}
	source := StepProviderSources{}
	for _, key := range orderedStepProviderKeys {
		raw := stepProviderValueForKey(&persisted, key)
		if strings.TrimSpace(raw) != "" {
			provider, err := ParseProvider(raw)
			if err != nil {
				return StepProviderResolution{}, fmt.Errorf("%s.provider: %w", key, err)
			}
			effective[key] = provider
			source[key] = ProviderSourceWorkspace
			continue
		}
		effective[key] = globalProvider
		source[key] = globalSource
	}
	return StepProviderResolution{
		Persisted: persisted,
		Effective: effective,
		Source:    source,
	}, nil
}

func (values StepProviderValues) ProviderForStep(stepID string) Provider {
	key := StepProviderKeyForStepID(stepID)
	if key == "" {
		return ProviderClaudeCode
	}
	if provider, ok := values[key]; ok && provider != "" {
		return provider
	}
	return ProviderClaudeCode
}

func (values StepProviderValues) All() []Provider {
	seen := map[Provider]struct{}{}
	out := make([]Provider, 0, len(values))
	for _, key := range orderedStepProviderKeys {
		provider := values[key]
		if provider == "" {
			continue
		}
		if _, exists := seen[provider]; exists {
			continue
		}
		seen[provider] = struct{}{}
		out = append(out, provider)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (values StepProviderValues) StringMap() map[string]string {
	out := map[string]string{}
	for _, key := range orderedStepProviderKeys {
		if provider := strings.TrimSpace(string(values[key])); provider != "" {
			out[key] = provider
		}
	}
	return out
}

func StepProviderKeyForStepID(stepID string) string {
	switch strings.TrimSpace(stepID) {
	case "init.step0.constitution":
		return StepProviderStep0Constitution
	case "init.step1.collect", "refresh.step1.collect":
		return StepProviderStep1Collect
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return StepProviderStep2AsIs
	case "init.step3.findings", "refresh.step3.findings":
		return StepProviderStep3Findings
	case "init.step4.proposals", "refresh.step4.proposals":
		return StepProviderStep4Proposals
	case StepIDQAAsk:
		return StepProviderQA
	default:
		return ""
	}
}

func stepProviderValueForKey(cfg *workspace.RuntimeStepsConfig, key string) string {
	if cfg == nil {
		return ""
	}
	switch key {
	case StepProviderStep0Constitution:
		if cfg.Step0Constitution != nil {
			return cfg.Step0Constitution.Provider
		}
	case StepProviderStep1Collect:
		if cfg.Step1Collect != nil {
			return cfg.Step1Collect.Provider
		}
	case StepProviderStep2AsIs:
		if cfg.Step2AsIs != nil {
			return cfg.Step2AsIs.Provider
		}
	case StepProviderStep3Findings:
		if cfg.Step3Findings != nil {
			return cfg.Step3Findings.Provider
		}
	case StepProviderStep4Proposals:
		if cfg.Step4Proposals != nil {
			return cfg.Step4Proposals.Provider
		}
	case StepProviderQA:
		if cfg.QA != nil {
			return cfg.QA.Provider
		}
	}
	return ""
}
