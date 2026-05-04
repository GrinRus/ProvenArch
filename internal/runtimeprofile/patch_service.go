package runtimeprofile

import (
	"errors"
	"fmt"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type RuntimeProfilePatchService struct{}

type PatchError struct {
	Code string
	Err  error
}

func (err PatchError) Error() string {
	if err.Err == nil {
		return err.Code
	}
	return err.Err.Error()
}

type RuntimeExecutionPatch struct {
	Strategy           *string                    `json:"strategy"`
	MaxParallelTasks   *int                       `json:"max_parallel_tasks"`
	FailurePolicy      *string                    `json:"failure_policy"`
	ShardDiscoveryMode *string                    `json:"shard_discovery_mode"`
	Steps              *RuntimeStepProvidersPatch `json:"steps"`
}

func (patch RuntimeExecutionPatch) IsZero() bool {
	return patch.Strategy == nil &&
		patch.MaxParallelTasks == nil &&
		patch.FailurePolicy == nil &&
		patch.ShardDiscoveryMode == nil &&
		(patch.Steps == nil || patch.Steps.IsZero())
}

type RuntimeStepProvidersPatch struct {
	Step0Constitution *string `json:"step0_constitution"`
	Step1Collect      *string `json:"step1_collect"`
	Step2AsIs         *string `json:"step2_as_is"`
	Step3Findings     *string `json:"step3_findings"`
	Step4Proposals    *string `json:"step4_proposals"`
}

func (patch RuntimeStepProvidersPatch) IsZero() bool {
	return patch.Step0Constitution == nil &&
		patch.Step1Collect == nil &&
		patch.Step2AsIs == nil &&
		patch.Step3Findings == nil &&
		patch.Step4Proposals == nil
}

func (RuntimeProfilePatchService) ApplyTimeouts(ws workspace.Root, patch workspace.RuntimeTimeoutsConfig) (workspace.Root, error) {
	if err := ValidateRuntimeTimeoutPatch(patch); err != nil {
		return workspace.Root{}, err
	}
	manifest := ws.Manifest
	ensureRuntimeProfile(&manifest)
	if manifest.Runtime.Profile.Timeouts == nil {
		manifest.Runtime.Profile.Timeouts = &workspace.RuntimeTimeoutsConfig{}
	}
	mergeRuntimeTimeoutPatch(manifest.Runtime.Profile.Timeouts, patch)
	pruneRuntimeProfile(&manifest)
	return writeRuntimeProfileManifest(ws, manifest, "runtime_timeouts")
}

func (RuntimeProfilePatchService) ApplyExecution(ws workspace.Root, patch RuntimeExecutionPatch) (workspace.Root, error) {
	if err := ValidateRuntimeExecutionPatch(patch); err != nil {
		return workspace.Root{}, err
	}
	manifest := ws.Manifest
	ensureRuntimeProfile(&manifest)
	if manifest.Runtime.Profile.Execution == nil {
		manifest.Runtime.Profile.Execution = &workspace.RuntimeExecutionConfig{}
	}
	if manifest.Runtime.Profile.Steps == nil {
		manifest.Runtime.Profile.Steps = &workspace.RuntimeStepsConfig{}
	}
	mergeRuntimeExecutionPatch(manifest.Runtime.Profile.Execution, patch)
	if patch.Steps != nil {
		mergeRuntimeStepProvidersPatch(manifest.Runtime.Profile.Steps, *patch.Steps)
	}
	pruneRuntimeProfile(&manifest)
	return writeRuntimeProfileManifest(ws, manifest, "runtime_execution")
}

func ValidateRuntimeTimeoutPatch(patch workspace.RuntimeTimeoutsConfig) error {
	checks := []struct {
		name  string
		value *int
	}{
		{name: "step_timeout_sec", value: patch.StepTimeoutSec},
		{name: "heartbeat_sec", value: patch.HeartbeatSec},
		{name: "pipeline_timeout_sec", value: patch.PipelineTimeoutSec},
		{name: "pipeline_kill_grace_sec", value: patch.PipelineKillGraceSec},
		{name: "api_ready_timeout_sec", value: patch.APIReadyTimeoutSec},
		{name: "api_init_timeout_sec", value: patch.APIInitTimeoutSec},
		{name: "ui_init_poll_timeout_sec", value: patch.UIInitPollTimeoutSec},
		{name: "ui_cancel_poll_timeout_sec", value: patch.UICancelPollTimeoutSec},
	}
	for _, check := range checks {
		if check.value != nil && *check.value <= 0 {
			return fmt.Errorf("%s must be > 0", check.name)
		}
	}
	return nil
}

func ValidateRuntimeExecutionPatch(patch RuntimeExecutionPatch) error {
	if patch.Strategy != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.Strategy))
		if value != acpruntime.ExecutionStrategySequential && value != acpruntime.ExecutionStrategyParallel {
			return fmt.Errorf("strategy must be one of: %s, %s", acpruntime.ExecutionStrategySequential, acpruntime.ExecutionStrategyParallel)
		}
	}
	if patch.MaxParallelTasks != nil && *patch.MaxParallelTasks <= 0 {
		return errors.New("max_parallel_tasks must be > 0")
	}
	if patch.FailurePolicy != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.FailurePolicy))
		if value != acpruntime.ExecutionFailurePolicyFailFast && value != acpruntime.ExecutionFailurePolicyBestEffort {
			return fmt.Errorf("failure_policy must be one of: %s, %s", acpruntime.ExecutionFailurePolicyFailFast, acpruntime.ExecutionFailurePolicyBestEffort)
		}
	}
	if patch.ShardDiscoveryMode != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.ShardDiscoveryMode))
		if value != acpruntime.ExecutionShardDiscoveryHeuristics && value != acpruntime.ExecutionShardDiscoverySemantic {
			return fmt.Errorf("shard_discovery_mode must be one of: %s, %s", acpruntime.ExecutionShardDiscoveryHeuristics, acpruntime.ExecutionShardDiscoverySemantic)
		}
	}
	if patch.Steps != nil {
		for label, value := range map[string]*string{
			"step0_constitution": patch.Steps.Step0Constitution,
			"step1_collect":      patch.Steps.Step1Collect,
			"step2_as_is":        patch.Steps.Step2AsIs,
			"step3_findings":     patch.Steps.Step3Findings,
			"step4_proposals":    patch.Steps.Step4Proposals,
		} {
			if value == nil {
				continue
			}
			provider := strings.TrimSpace(strings.ToLower(*value))
			if provider != string(acpruntime.ProviderClaudeCode) &&
				provider != string(acpruntime.ProviderQwenCode) &&
				provider != string(acpruntime.ProviderCodexCode) {
				return fmt.Errorf("%s must be one of: %s, %s, %s", label, acpruntime.ProviderClaudeCode, acpruntime.ProviderQwenCode, acpruntime.ProviderCodexCode)
			}
		}
	}
	return nil
}

func mergeRuntimeTimeoutPatch(dst *workspace.RuntimeTimeoutsConfig, patch workspace.RuntimeTimeoutsConfig) {
	if dst == nil {
		return
	}
	if patch.StepTimeoutSec != nil {
		dst.StepTimeoutSec = patch.StepTimeoutSec
	}
	if patch.HeartbeatSec != nil {
		dst.HeartbeatSec = patch.HeartbeatSec
	}
	if patch.PipelineTimeoutSec != nil {
		dst.PipelineTimeoutSec = patch.PipelineTimeoutSec
	}
	if patch.PipelineKillGraceSec != nil {
		dst.PipelineKillGraceSec = patch.PipelineKillGraceSec
	}
	if patch.APIReadyTimeoutSec != nil {
		dst.APIReadyTimeoutSec = patch.APIReadyTimeoutSec
	}
	if patch.APIInitTimeoutSec != nil {
		dst.APIInitTimeoutSec = patch.APIInitTimeoutSec
	}
	if patch.UIInitPollTimeoutSec != nil {
		dst.UIInitPollTimeoutSec = patch.UIInitPollTimeoutSec
	}
	if patch.UICancelPollTimeoutSec != nil {
		dst.UICancelPollTimeoutSec = patch.UICancelPollTimeoutSec
	}
}

func mergeRuntimeExecutionPatch(dst *workspace.RuntimeExecutionConfig, patch RuntimeExecutionPatch) {
	if dst == nil {
		return
	}
	if patch.Strategy != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.Strategy))
		dst.Strategy = value
	}
	if patch.MaxParallelTasks != nil {
		dst.MaxParallel = patch.MaxParallelTasks
	}
	if patch.FailurePolicy != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.FailurePolicy))
		dst.FailurePolicy = value
	}
	if patch.ShardDiscoveryMode != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.ShardDiscoveryMode))
		if dst.ShardDiscovery == nil {
			dst.ShardDiscovery = &workspace.RuntimeShardDiscoveryConfig{}
		}
		dst.ShardDiscovery.Mode = value
	}
}

func mergeRuntimeStepProvidersPatch(dst *workspace.RuntimeStepsConfig, patch RuntimeStepProvidersPatch) {
	if dst == nil {
		return
	}
	mergeStep := func(target **workspace.RuntimeStepConfig, raw *string) {
		if raw == nil {
			return
		}
		if *target == nil {
			*target = &workspace.RuntimeStepConfig{}
		}
		(*target).Provider = strings.TrimSpace(strings.ToLower(*raw))
		if (*target).IsZero() {
			*target = nil
		}
	}
	mergeStep(&dst.Step0Constitution, patch.Step0Constitution)
	mergeStep(&dst.Step1Collect, patch.Step1Collect)
	mergeStep(&dst.Step2AsIs, patch.Step2AsIs)
	mergeStep(&dst.Step3Findings, patch.Step3Findings)
	mergeStep(&dst.Step4Proposals, patch.Step4Proposals)
}

func ensureRuntimeProfile(manifest *workspace.Manifest) {
	if manifest.Runtime == nil {
		manifest.Runtime = &workspace.RuntimeConfig{}
	}
	if manifest.Runtime.Profile == nil {
		manifest.Runtime.Profile = &workspace.RuntimeProfileConfig{}
	}
}

func pruneRuntimeProfile(manifest *workspace.Manifest) {
	if manifest.Runtime == nil || manifest.Runtime.Profile == nil {
		return
	}
	if manifest.Runtime.Profile.Timeouts != nil && manifest.Runtime.Profile.Timeouts.IsZero() {
		manifest.Runtime.Profile.Timeouts = nil
	}
	if manifest.Runtime.Profile.Execution != nil && manifest.Runtime.Profile.Execution.IsZero() {
		manifest.Runtime.Profile.Execution = nil
	}
	if manifest.Runtime.Profile.Steps != nil && manifest.Runtime.Profile.Steps.IsZero() {
		manifest.Runtime.Profile.Steps = nil
	}
	if manifest.Runtime.Profile.IsZero() {
		manifest.Runtime.Profile = nil
	}
	if manifest.Runtime.IsZero() {
		manifest.Runtime = nil
	}
}

func writeRuntimeProfileManifest(ws workspace.Root, manifest workspace.Manifest, codePrefix string) (workspace.Root, error) {
	rawManifest, err := workspace.RenderManifest(manifest)
	if err != nil {
		return workspace.Root{}, PatchError{Code: codePrefix + "_render_failed", Err: fmt.Errorf("render workspace manifest: %w", err)}
	}
	if err := ws.WriteFile(workspace.ManifestFileName, rawManifest); err != nil {
		return workspace.Root{}, PatchError{Code: codePrefix + "_write_failed", Err: fmt.Errorf("write workspace manifest: %w", err)}
	}
	reopened, err := workspace.Open(ws.Path)
	if err != nil {
		return workspace.Root{}, PatchError{Code: codePrefix + "_reopen_failed", Err: fmt.Errorf("reopen workspace: %w", err)}
	}
	return reopened, nil
}
