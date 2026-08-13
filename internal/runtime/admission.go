package runtime

// AdmittedRuntimeSnapshot is the immutable runtime configuration accepted for
// one Task Attempt. It is intentionally independent from workspace Manifest so
// later Settings/env changes cannot rewrite an admitted execution.
type AdmittedRuntimeSnapshot struct {
	Mode                 string               `json:"mode"`
	StepProviders        StepProviderValues   `json:"step_providers"`
	StepProviderSources  StepProviderSources  `json:"step_provider_sources"`
	ProviderModels       ProviderModelValues  `json:"provider_models"`
	ProviderModelSources ProviderModelSources `json:"provider_model_sources"`
	Execution            ExecutionValues      `json:"execution"`
	Permissions          PermissionValues     `json:"permissions"`
	Timeouts             TimeoutValues        `json:"timeouts"`
	RepositoryScopes     []string             `json:"repository_scopes"`
}

func CloneAdmittedRuntimeSnapshot(value *AdmittedRuntimeSnapshot) *AdmittedRuntimeSnapshot {
	if value == nil {
		return nil
	}
	clone := *value
	clone.StepProviders = cloneStepProviderValues(value.StepProviders)
	clone.StepProviderSources = cloneStepProviderSources(value.StepProviderSources)
	clone.ProviderModels = cloneProviderModelValues(value.ProviderModels)
	clone.ProviderModelSources = cloneProviderModelSources(value.ProviderModelSources)
	return &clone
}

func cloneStepProviderValues(value StepProviderValues) StepProviderValues {
	if value == nil {
		return nil
	}
	clone := make(StepProviderValues, len(value))
	for key, provider := range value {
		clone[key] = provider
	}
	return clone
}

func cloneStepProviderSources(value StepProviderSources) StepProviderSources {
	if value == nil {
		return nil
	}
	clone := make(StepProviderSources, len(value))
	for key, source := range value {
		clone[key] = source
	}
	return clone
}

func cloneProviderModelValues(value ProviderModelValues) ProviderModelValues {
	if value == nil {
		return nil
	}
	clone := make(ProviderModelValues, len(value))
	for provider, config := range value {
		clone[provider] = config
	}
	return clone
}

func cloneProviderModelSources(value ProviderModelSources) ProviderModelSources {
	if value == nil {
		return nil
	}
	clone := make(ProviderModelSources, len(value))
	for provider, source := range value {
		clone[provider] = source
	}
	return clone
}
