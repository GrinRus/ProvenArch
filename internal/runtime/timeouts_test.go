package runtime

import (
	"testing"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestResolveTimeoutsUsesDefaults(t *testing.T) {
	t.Parallel()

	resolved := resolveTimeoutsWithLookup(workspace.Manifest{}, func(string) (string, bool) {
		return "", false
	})
	if resolved.Effective.StepTimeoutSec != DefaultRuntimeStepTimeoutSec {
		t.Fatalf("expected default step timeout %d, got %d", DefaultRuntimeStepTimeoutSec, resolved.Effective.StepTimeoutSec)
	}
	if resolved.Effective.UIInitPollTimeoutSec != DefaultUIInitPollTimeoutSec {
		t.Fatalf("expected default ui init timeout %d, got %d", DefaultUIInitPollTimeoutSec, resolved.Effective.UIInitPollTimeoutSec)
	}
	if resolved.Source.StepTimeoutSec != TimeoutSourceDefault {
		t.Fatalf("expected default source, got %s", resolved.Source.StepTimeoutSec)
	}
}

func TestResolveTimeoutsUsesWorkspaceValues(t *testing.T) {
	t.Parallel()

	step := 111
	heartbeat := 22
	manifest := workspace.Manifest{
		Runtime: &workspace.RuntimeConfig{
			Timeouts: &workspace.RuntimeTimeoutsConfig{
				StepTimeoutSec: &step,
				HeartbeatSec:   &heartbeat,
			},
		},
	}

	resolved := resolveTimeoutsWithLookup(manifest, func(string) (string, bool) {
		return "", false
	})
	if resolved.Effective.StepTimeoutSec != 111 {
		t.Fatalf("expected workspace step timeout 111, got %d", resolved.Effective.StepTimeoutSec)
	}
	if resolved.Effective.HeartbeatSec != 22 {
		t.Fatalf("expected workspace heartbeat 22, got %d", resolved.Effective.HeartbeatSec)
	}
	if resolved.Source.StepTimeoutSec != TimeoutSourceWorkspace {
		t.Fatalf("expected workspace source, got %s", resolved.Source.StepTimeoutSec)
	}
}

func TestResolveTimeoutsCanonicalEnvOverridesWorkspace(t *testing.T) {
	t.Parallel()

	step := 100
	manifest := workspace.Manifest{
		Runtime: &workspace.RuntimeConfig{
			Timeouts: &workspace.RuntimeTimeoutsConfig{
				StepTimeoutSec: &step,
			},
		},
	}
	resolved := resolveTimeoutsWithLookup(manifest, func(name string) (string, bool) {
		if name == RuntimeStepTimeoutEnv {
			return "222", true
		}
		return "", false
	})
	if resolved.Effective.StepTimeoutSec != 222 {
		t.Fatalf("expected env override value 222, got %d", resolved.Effective.StepTimeoutSec)
	}
	if resolved.Source.StepTimeoutSec != TimeoutSourceEnv {
		t.Fatalf("expected env source, got %s", resolved.Source.StepTimeoutSec)
	}
}

func TestResolveTimeoutsDeprecatedEnvFallback(t *testing.T) {
	t.Parallel()

	resolved := resolveTimeoutsWithLookup(workspace.Manifest{}, func(name string) (string, bool) {
		if name == ReadyTimeoutDeprecatedEnv {
			return "77", true
		}
		return "", false
	})
	if resolved.Effective.APIReadyTimeoutSec != 77 {
		t.Fatalf("expected deprecated env value 77, got %d", resolved.Effective.APIReadyTimeoutSec)
	}
	if resolved.Source.APIReadyTimeoutSec != TimeoutSourceDeprecatedEnv {
		t.Fatalf("expected deprecated env source, got %s", resolved.Source.APIReadyTimeoutSec)
	}
}

func TestResolveTimeoutsCanonicalEnvBeatsDeprecatedAlias(t *testing.T) {
	t.Parallel()

	resolved := resolveTimeoutsWithLookup(workspace.Manifest{}, func(name string) (string, bool) {
		switch name {
		case APIReadyTimeoutEnv:
			return "88", true
		case ReadyTimeoutDeprecatedEnv:
			return "99", true
		default:
			return "", false
		}
	})
	if resolved.Effective.APIReadyTimeoutSec != 88 {
		t.Fatalf("expected canonical env value 88, got %d", resolved.Effective.APIReadyTimeoutSec)
	}
	if resolved.Source.APIReadyTimeoutSec != TimeoutSourceEnv {
		t.Fatalf("expected env source, got %s", resolved.Source.APIReadyTimeoutSec)
	}
}
