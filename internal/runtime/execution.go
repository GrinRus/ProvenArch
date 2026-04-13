package runtime

import (
	"os"
	"strconv"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const (
	ExecutionStrategyEnv      = "ACP_EXECUTION_STRATEGY"
	ExecutionMaxParallelEnv   = "ACP_MAX_PARALLEL_TASKS"
	ExecutionFailurePolicyEnv = "ACP_FAILURE_POLICY"
	ExecutionShardModeEnv     = "ACP_SHARD_DISCOVERY_MODE"
	ExecutionRepoSelectionEnv = "ACP_REPO_SELECTION"
)

const (
	ExecutionStrategySequential = "sequential"
	ExecutionStrategyParallel   = "parallel"

	ExecutionFailurePolicyFailFast   = "fail_fast"
	ExecutionFailurePolicyBestEffort = "best_effort"

	ExecutionShardDiscoveryHeuristics = "heuristics"
	ExecutionShardDiscoverySemantic   = "semantic"

	ExecutionRepoSelectionAll         = workspace.RepoSelectionAll
	ExecutionRepoSelectionBackendOnly = workspace.RepoSelectionBackendOnly
)

const (
	DefaultExecutionStrategy      = ExecutionStrategySequential
	DefaultExecutionMaxParallel   = 1
	DefaultExecutionFailurePolicy = ExecutionFailurePolicyBestEffort
	DefaultExecutionShardMode     = ExecutionShardDiscoveryHeuristics
	DefaultExecutionRepoSelection = ExecutionRepoSelectionAll
)

type ExecutionSource string

const (
	ExecutionSourceDefault   ExecutionSource = "default"
	ExecutionSourceWorkspace ExecutionSource = "workspace"
	ExecutionSourceEnv       ExecutionSource = "env"
	ExecutionSourceOverride  ExecutionSource = "override"
)

type ExecutionValues struct {
	Strategy      string `json:"strategy"`
	MaxParallel   int    `json:"max_parallel_tasks"`
	FailurePolicy string `json:"failure_policy"`
	ShardMode     string `json:"shard_discovery_mode"`
	RepoSelection string `json:"repo_selection"`
}

type ExecutionSources struct {
	Strategy      ExecutionSource `json:"strategy"`
	MaxParallel   ExecutionSource `json:"max_parallel_tasks"`
	FailurePolicy ExecutionSource `json:"failure_policy"`
	ShardMode     ExecutionSource `json:"shard_discovery_mode"`
	RepoSelection ExecutionSource `json:"repo_selection"`
}

type ExecutionResolution struct {
	Persisted workspace.RuntimeExecutionConfig `json:"persisted"`
	Effective ExecutionValues                  `json:"effective"`
	Source    ExecutionSources                 `json:"source"`
}

type ExecutionOverrides struct {
	Strategy      *string
	MaxParallel   *int
	FailurePolicy *string
	ShardMode     *string
	RepoSelection *string
}

func ResolveExecution(manifest workspace.Manifest, overrides ExecutionOverrides) ExecutionResolution {
	return resolveExecutionWithLookup(manifest, overrides, os.LookupEnv)
}

func DefaultExecution() ExecutionValues {
	return ExecutionValues{
		Strategy:      DefaultExecutionStrategy,
		MaxParallel:   DefaultExecutionMaxParallel,
		FailurePolicy: DefaultExecutionFailurePolicy,
		ShardMode:     DefaultExecutionShardMode,
		RepoSelection: DefaultExecutionRepoSelection,
	}
}

func resolveExecutionWithLookup(manifest workspace.Manifest, overrides ExecutionOverrides, lookup envLookup) ExecutionResolution {
	persisted := workspace.RuntimeExecutionConfig{}
	if manifest.Runtime != nil && manifest.Runtime.Profile != nil && manifest.Runtime.Profile.Execution != nil {
		persisted = *manifest.Runtime.Profile.Execution
	}

	effective := DefaultExecution()
	source := ExecutionSources{
		Strategy:      ExecutionSourceDefault,
		MaxParallel:   ExecutionSourceDefault,
		FailurePolicy: ExecutionSourceDefault,
		ShardMode:     ExecutionSourceDefault,
		RepoSelection: ExecutionSourceDefault,
	}

	effective.Strategy, source.Strategy = resolveEnumValue(
		persisted.Strategy,
		overrides.Strategy,
		ExecutionStrategyEnv,
		[]string{ExecutionStrategySequential, ExecutionStrategyParallel},
		DefaultExecutionStrategy,
		lookup,
	)
	effective.MaxParallel, source.MaxParallel = resolvePositiveIntValue(
		persisted.MaxParallel,
		overrides.MaxParallel,
		ExecutionMaxParallelEnv,
		DefaultExecutionMaxParallel,
		lookup,
	)
	effective.FailurePolicy, source.FailurePolicy = resolveEnumValue(
		persisted.FailurePolicy,
		overrides.FailurePolicy,
		ExecutionFailurePolicyEnv,
		[]string{ExecutionFailurePolicyFailFast, ExecutionFailurePolicyBestEffort},
		DefaultExecutionFailurePolicy,
		lookup,
	)
	persistedShardMode := ""
	if persisted.ShardDiscovery != nil {
		persistedShardMode = persisted.ShardDiscovery.Mode
	}
	effective.ShardMode, source.ShardMode = resolveEnumValue(
		persistedShardMode,
		overrides.ShardMode,
		ExecutionShardModeEnv,
		[]string{ExecutionShardDiscoveryHeuristics, ExecutionShardDiscoverySemantic},
		DefaultExecutionShardMode,
		lookup,
	)
	effective.RepoSelection, source.RepoSelection = resolveEnumValue(
		persisted.RepoSelection,
		overrides.RepoSelection,
		ExecutionRepoSelectionEnv,
		[]string{ExecutionRepoSelectionAll, ExecutionRepoSelectionBackendOnly},
		DefaultExecutionRepoSelection,
		lookup,
	)

	return ExecutionResolution{
		Persisted: persisted,
		Effective: effective,
		Source:    source,
	}
}

func resolveEnumValue(
	persisted string,
	override *string,
	envName string,
	allowed []string,
	defaultValue string,
	lookup envLookup,
) (string, ExecutionSource) {
	if override != nil {
		if normalized, ok := normalizeEnumValue(*override, allowed); ok {
			return normalized, ExecutionSourceOverride
		}
	}
	if envRaw, ok := lookup(envName); ok {
		if normalized, valid := normalizeEnumValue(envRaw, allowed); valid {
			return normalized, ExecutionSourceEnv
		}
	}
	if normalized, ok := normalizeEnumValue(persisted, allowed); ok {
		return normalized, ExecutionSourceWorkspace
	}
	return defaultValue, ExecutionSourceDefault
}

func resolvePositiveIntValue(
	persisted *int,
	override *int,
	envName string,
	defaultValue int,
	lookup envLookup,
) (int, ExecutionSource) {
	if override != nil && *override > 0 {
		return *override, ExecutionSourceOverride
	}
	if envRaw, ok := lookup(envName); ok {
		if value, err := strconv.Atoi(strings.TrimSpace(envRaw)); err == nil && value > 0 {
			return value, ExecutionSourceEnv
		}
	}
	if persisted != nil && *persisted > 0 {
		return *persisted, ExecutionSourceWorkspace
	}
	return defaultValue, ExecutionSourceDefault
}

func normalizeEnumValue(value string, allowed []string) (string, bool) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return "", false
	}
	for _, candidate := range allowed {
		if normalized == candidate {
			return normalized, true
		}
	}
	return "", false
}
