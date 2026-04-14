package runtime

import (
	"os"
	"strconv"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const (
	RuntimeStepTimeoutEnv     = "ACP_RUNTIME_STEP_TIMEOUT_SEC"
	RuntimeHeartbeatEnv       = "ACP_RUNTIME_HEARTBEAT_SEC"
	PipelineTimeoutEnv        = "ACP_PIPELINE_TIMEOUT_SEC"
	PipelineKillGraceEnv      = "ACP_PIPELINE_KILL_GRACE_SEC"
	APIReadyTimeoutEnv        = "ACP_API_READY_TIMEOUT_SEC"
	APIInitTimeoutEnv         = "ACP_API_INIT_TIMEOUT_SEC"
	UIInitPollTimeoutEnv      = "ACP_UI_INIT_POLL_TIMEOUT_SEC"
	UICancelPollTimeoutEnv    = "ACP_UI_CANCEL_POLL_TIMEOUT_SEC"
	ReadyTimeoutDeprecatedEnv = "READY_TIMEOUT_SEC"
	UIInitDeprecatedEnv       = "UI_E2E_INIT_TIMEOUT_SEC"
	UICancelDeprecatedEnv     = "UI_E2E_CANCEL_TIMEOUT_SEC"
)

const (
	DefaultRuntimeStepTimeoutSec  = 1800
	DefaultRuntimeHeartbeatSec    = 30
	DefaultPipelineTimeoutSec     = 2400
	DefaultPipelineKillGraceSec   = 30
	DefaultAPIReadyTimeoutSec     = 60
	DefaultAPIInitTimeoutSec      = 120
	DefaultUIInitPollTimeoutSec   = 900
	DefaultUICancelPollTimeoutSec = 420
)

type TimeoutSource string

const (
	TimeoutSourceDefault       TimeoutSource = "default"
	TimeoutSourceWorkspace     TimeoutSource = "workspace"
	TimeoutSourceEnv           TimeoutSource = "env"
	TimeoutSourceDeprecatedEnv TimeoutSource = "deprecated_env"
)

type TimeoutValues struct {
	StepTimeoutSec         int `json:"step_timeout_sec"`
	HeartbeatSec           int `json:"heartbeat_sec"`
	PipelineTimeoutSec     int `json:"pipeline_timeout_sec"`
	PipelineKillGraceSec   int `json:"pipeline_kill_grace_sec"`
	APIReadyTimeoutSec     int `json:"api_ready_timeout_sec"`
	APIInitTimeoutSec      int `json:"api_init_timeout_sec"`
	UIInitPollTimeoutSec   int `json:"ui_init_poll_timeout_sec"`
	UICancelPollTimeoutSec int `json:"ui_cancel_poll_timeout_sec"`
}

type TimeoutSources struct {
	StepTimeoutSec         TimeoutSource `json:"step_timeout_sec"`
	HeartbeatSec           TimeoutSource `json:"heartbeat_sec"`
	PipelineTimeoutSec     TimeoutSource `json:"pipeline_timeout_sec"`
	PipelineKillGraceSec   TimeoutSource `json:"pipeline_kill_grace_sec"`
	APIReadyTimeoutSec     TimeoutSource `json:"api_ready_timeout_sec"`
	APIInitTimeoutSec      TimeoutSource `json:"api_init_timeout_sec"`
	UIInitPollTimeoutSec   TimeoutSource `json:"ui_init_poll_timeout_sec"`
	UICancelPollTimeoutSec TimeoutSource `json:"ui_cancel_poll_timeout_sec"`
}

type TimeoutResolution struct {
	Persisted workspace.RuntimeTimeoutsConfig `json:"persisted"`
	Effective TimeoutValues                   `json:"effective"`
	Source    TimeoutSources                  `json:"source"`
}

type envLookup func(string) (string, bool)

func ResolveTimeouts(manifest workspace.Manifest) TimeoutResolution {
	return resolveTimeoutsWithLookup(manifest, os.LookupEnv)
}

func DefaultTimeouts() TimeoutValues {
	return TimeoutValues{
		StepTimeoutSec:         DefaultRuntimeStepTimeoutSec,
		HeartbeatSec:           DefaultRuntimeHeartbeatSec,
		PipelineTimeoutSec:     DefaultPipelineTimeoutSec,
		PipelineKillGraceSec:   DefaultPipelineKillGraceSec,
		APIReadyTimeoutSec:     DefaultAPIReadyTimeoutSec,
		APIInitTimeoutSec:      DefaultAPIInitTimeoutSec,
		UIInitPollTimeoutSec:   DefaultUIInitPollTimeoutSec,
		UICancelPollTimeoutSec: DefaultUICancelPollTimeoutSec,
	}
}

func resolveTimeoutsWithLookup(manifest workspace.Manifest, lookup envLookup) TimeoutResolution {
	persisted := workspace.RuntimeTimeoutsConfig{}
	if manifest.Runtime != nil && manifest.Runtime.Profile != nil && manifest.Runtime.Profile.Timeouts != nil {
		persisted = *manifest.Runtime.Profile.Timeouts
	}

	effective := DefaultTimeouts()
	source := TimeoutSources{
		StepTimeoutSec:         TimeoutSourceDefault,
		HeartbeatSec:           TimeoutSourceDefault,
		PipelineTimeoutSec:     TimeoutSourceDefault,
		PipelineKillGraceSec:   TimeoutSourceDefault,
		APIReadyTimeoutSec:     TimeoutSourceDefault,
		APIInitTimeoutSec:      TimeoutSourceDefault,
		UIInitPollTimeoutSec:   TimeoutSourceDefault,
		UICancelPollTimeoutSec: TimeoutSourceDefault,
	}

	effective.StepTimeoutSec, source.StepTimeoutSec = resolveIntTimeout(persisted.StepTimeoutSec, RuntimeStepTimeoutEnv, nil, DefaultRuntimeStepTimeoutSec, lookup)
	effective.HeartbeatSec, source.HeartbeatSec = resolveIntTimeout(persisted.HeartbeatSec, RuntimeHeartbeatEnv, nil, DefaultRuntimeHeartbeatSec, lookup)
	effective.PipelineTimeoutSec, source.PipelineTimeoutSec = resolveIntTimeout(persisted.PipelineTimeoutSec, PipelineTimeoutEnv, nil, DefaultPipelineTimeoutSec, lookup)
	effective.PipelineKillGraceSec, source.PipelineKillGraceSec = resolveIntTimeout(persisted.PipelineKillGraceSec, PipelineKillGraceEnv, nil, DefaultPipelineKillGraceSec, lookup)
	effective.APIReadyTimeoutSec, source.APIReadyTimeoutSec = resolveIntTimeout(persisted.APIReadyTimeoutSec, APIReadyTimeoutEnv, []string{ReadyTimeoutDeprecatedEnv}, DefaultAPIReadyTimeoutSec, lookup)
	effective.APIInitTimeoutSec, source.APIInitTimeoutSec = resolveIntTimeout(persisted.APIInitTimeoutSec, APIInitTimeoutEnv, nil, DefaultAPIInitTimeoutSec, lookup)
	effective.UIInitPollTimeoutSec, source.UIInitPollTimeoutSec = resolveIntTimeout(persisted.UIInitPollTimeoutSec, UIInitPollTimeoutEnv, []string{UIInitDeprecatedEnv}, DefaultUIInitPollTimeoutSec, lookup)
	effective.UICancelPollTimeoutSec, source.UICancelPollTimeoutSec = resolveIntTimeout(persisted.UICancelPollTimeoutSec, UICancelPollTimeoutEnv, []string{UICancelDeprecatedEnv}, DefaultUICancelPollTimeoutSec, lookup)

	return TimeoutResolution{
		Persisted: persisted,
		Effective: effective,
		Source:    source,
	}
}

func resolveIntTimeout(
	persisted *int,
	canonicalEnv string,
	deprecatedEnvs []string,
	defaultValue int,
	lookup envLookup,
) (int, TimeoutSource) {
	if value, ok := readPositiveIntEnv(canonicalEnv, lookup); ok {
		return value, TimeoutSourceEnv
	}
	for _, alias := range deprecatedEnvs {
		if value, ok := readPositiveIntEnv(alias, lookup); ok {
			return value, TimeoutSourceDeprecatedEnv
		}
	}
	if persisted != nil && *persisted > 0 {
		return *persisted, TimeoutSourceWorkspace
	}
	return defaultValue, TimeoutSourceDefault
}

func readPositiveIntEnv(name string, lookup envLookup) (int, bool) {
	if strings.TrimSpace(name) == "" || lookup == nil {
		return 0, false
	}
	raw, ok := lookup(name)
	if !ok {
		return 0, false
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}
