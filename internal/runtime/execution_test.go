package runtime

import (
	"testing"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestResolveExecutionDefaults(t *testing.T) {
	t.Parallel()

	resolved := resolveExecutionWithLookup(workspace.Manifest{}, ExecutionOverrides{}, func(string) (string, bool) {
		return "", false
	})
	if resolved.Effective.Strategy != DefaultExecutionStrategy {
		t.Fatalf("expected default strategy %q, got %q", DefaultExecutionStrategy, resolved.Effective.Strategy)
	}
	if resolved.Effective.MaxParallel != DefaultExecutionMaxParallel {
		t.Fatalf("expected default max parallel %d, got %d", DefaultExecutionMaxParallel, resolved.Effective.MaxParallel)
	}
	if resolved.Source.Strategy != ExecutionSourceDefault {
		t.Fatalf("expected default source, got %q", resolved.Source.Strategy)
	}
}

func TestResolveExecutionWorkspaceValues(t *testing.T) {
	t.Parallel()

	maxParallel := 7
	manifest := workspace.Manifest{
		Runtime: &workspace.RuntimeConfig{
			Profile: &workspace.RuntimeProfileConfig{
				Execution: &workspace.RuntimeExecutionConfig{
					Strategy:      ExecutionStrategyParallel,
					MaxParallel:   &maxParallel,
					FailurePolicy: ExecutionFailurePolicyFailFast,
					ShardDiscovery: &workspace.RuntimeShardDiscoveryConfig{
						Mode: ExecutionShardDiscoverySemantic,
					},
				},
			},
		},
	}
	resolved := resolveExecutionWithLookup(manifest, ExecutionOverrides{}, func(string) (string, bool) {
		return "", false
	})
	if resolved.Effective.Strategy != ExecutionStrategyParallel {
		t.Fatalf("expected strategy parallel, got %q", resolved.Effective.Strategy)
	}
	if resolved.Effective.MaxParallel != 7 {
		t.Fatalf("expected max parallel 7, got %d", resolved.Effective.MaxParallel)
	}
	if resolved.Source.Strategy != ExecutionSourceWorkspace {
		t.Fatalf("expected workspace source, got %q", resolved.Source.Strategy)
	}
}

func TestResolveExecutionOverrideBeatsEnvAndWorkspace(t *testing.T) {
	t.Parallel()

	maxParallel := 3
	manifest := workspace.Manifest{
		Runtime: &workspace.RuntimeConfig{
			Profile: &workspace.RuntimeProfileConfig{
				Execution: &workspace.RuntimeExecutionConfig{
					Strategy:      ExecutionStrategySequential,
					MaxParallel:   &maxParallel,
					FailurePolicy: ExecutionFailurePolicyBestEffort,
				},
			},
		},
	}
	overrideStrategy := ExecutionStrategyParallel
	overrideMax := 9
	overridePolicy := ExecutionFailurePolicyFailFast
	resolved := resolveExecutionWithLookup(manifest, ExecutionOverrides{
		Strategy:      &overrideStrategy,
		MaxParallel:   &overrideMax,
		FailurePolicy: &overridePolicy,
	}, func(name string) (string, bool) {
		switch name {
		case ExecutionStrategyEnv:
			return ExecutionStrategySequential, true
		case ExecutionMaxParallelEnv:
			return "4", true
		case ExecutionFailurePolicyEnv:
			return ExecutionFailurePolicyBestEffort, true
		default:
			return "", false
		}
	})
	if resolved.Effective.Strategy != ExecutionStrategyParallel {
		t.Fatalf("expected override strategy parallel, got %q", resolved.Effective.Strategy)
	}
	if resolved.Effective.MaxParallel != 9 {
		t.Fatalf("expected override max 9, got %d", resolved.Effective.MaxParallel)
	}
	if resolved.Effective.FailurePolicy != ExecutionFailurePolicyFailFast {
		t.Fatalf("expected override failure policy fail_fast, got %q", resolved.Effective.FailurePolicy)
	}
	if resolved.Source.Strategy != ExecutionSourceOverride {
		t.Fatalf("expected override source, got %q", resolved.Source.Strategy)
	}
}
