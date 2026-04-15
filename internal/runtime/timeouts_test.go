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
			Profile: &workspace.RuntimeProfileConfig{
				Timeouts: &workspace.RuntimeTimeoutsConfig{
					StepTimeoutSec: &step,
					HeartbeatSec:   &heartbeat,
				},
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
			Profile: &workspace.RuntimeProfileConfig{
				Timeouts: &workspace.RuntimeTimeoutsConfig{
					StepTimeoutSec: &step,
				},
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

func TestResolveTimeoutsIgnoresLegacyTimeoutAliases(t *testing.T) {
	t.Parallel()

	apiReady := 55
	uiInit := 333
	uiCancel := 444
	manifest := workspace.Manifest{
		Runtime: &workspace.RuntimeConfig{
			Profile: &workspace.RuntimeProfileConfig{
				Timeouts: &workspace.RuntimeTimeoutsConfig{
					APIReadyTimeoutSec:     &apiReady,
					UIInitPollTimeoutSec:   &uiInit,
					UICancelPollTimeoutSec: &uiCancel,
				},
			},
		},
	}
	resolved := resolveTimeoutsWithLookup(manifest, func(name string) (string, bool) {
		switch name {
		case "READY_TIMEOUT_SEC":
			return "77", true
		case "UI_E2E_INIT_TIMEOUT_SEC":
			return "888", true
		case "UI_E2E_CANCEL_TIMEOUT_SEC":
			return "999", true
		default:
			return "", false
		}
	})
	if resolved.Effective.APIReadyTimeoutSec != apiReady {
		t.Fatalf("expected workspace api ready timeout %d, got %d", apiReady, resolved.Effective.APIReadyTimeoutSec)
	}
	if resolved.Effective.UIInitPollTimeoutSec != uiInit {
		t.Fatalf("expected workspace ui init timeout %d, got %d", uiInit, resolved.Effective.UIInitPollTimeoutSec)
	}
	if resolved.Effective.UICancelPollTimeoutSec != uiCancel {
		t.Fatalf("expected workspace ui cancel timeout %d, got %d", uiCancel, resolved.Effective.UICancelPollTimeoutSec)
	}
	if resolved.Source.APIReadyTimeoutSec != TimeoutSourceWorkspace {
		t.Fatalf("expected workspace source for api ready timeout, got %s", resolved.Source.APIReadyTimeoutSec)
	}
	if resolved.Source.UIInitPollTimeoutSec != TimeoutSourceWorkspace {
		t.Fatalf("expected workspace source for ui init timeout, got %s", resolved.Source.UIInitPollTimeoutSec)
	}
	if resolved.Source.UICancelPollTimeoutSec != TimeoutSourceWorkspace {
		t.Fatalf("expected workspace source for ui cancel timeout, got %s", resolved.Source.UICancelPollTimeoutSec)
	}
}
