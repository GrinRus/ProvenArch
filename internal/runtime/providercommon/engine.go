package providercommon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

const (
	defaultPostArtifactStallWindow = 20 * time.Second
	defaultPreArtifactStallWindow  = 75 * time.Second
	defaultRetryPreArtifactWindow  = 3 * time.Minute
	defaultCollectRepairWindow     = 3 * time.Minute
	defaultFocusedRepairWindow     = 90 * time.Second
	defaultStallPollInterval       = 2 * time.Second
	defaultStallTerminateGrace     = 2 * time.Second
	defaultPostTerminateDrain      = 500 * time.Millisecond
)

// CommandSpec is the provider-specific process invocation surface. Success is
// still determined by artifact validation, not by stdout/stderr shape.
type CommandSpec struct {
	Provider acpruntime.Provider
	Command  string
	Args     []string
	Stdin    io.Reader
	Dir      string
	// PromptBytes records the provider prompt payload size without requiring
	// diagnostics to inspect or consume stdin readers.
	PromptBytes int
	// IncludeDirs records the read scope the adapter encoded into provider CLI
	// args. The shared engine does not interpret it as a success contract.
	IncludeDirs []string
}

// ProviderAdapter keeps CLI differences at the edge while sharing process
// lifecycle, artifact validation, and failure classification.
type ProviderAdapter interface {
	Provider() acpruntime.Provider
	RuntimeVersion() string
	CommandSpec(acpruntime.Task) (CommandSpec, error)
	ValidateArtifacts(acpruntime.Task) error
	ActivityPolicy(acpruntime.Task) ActivityPolicy
	RecoveryPolicy(acpruntime.Task) RecoveryPolicy
	UnavailableMarkers() []string
}

// CollectManifestRepairAdapter is implemented by provider adapters that can run
// a narrow collect-manifest repair prompt after a provider already wrote
// authored collect documents but missed or malformed shard-pack-manifest.json.
type CollectManifestRepairAdapter interface {
	CollectManifestRepairCommandSpec(acpruntime.Task, error) (CommandSpec, error)
}

// CollectArtifactPairRepairAdapter is implemented by adapters that can run a
// focused collect recovery when a provider produced diagnostics but no collect
// artifacts. The provider still authors both files; ACP only validates them.
type CollectArtifactPairRepairAdapter interface {
	CollectArtifactPairRepairCommandSpec(acpruntime.Task, error) (CommandSpec, error)
}

// ValidatorVerdictRepairAdapter is implemented by adapters that can run a
// verdict-only recovery prompt after step3 misses or malforms
// validator-verdict.json.
type ValidatorVerdictRepairAdapter interface {
	ValidatorVerdictRepairCommandSpec(acpruntime.Task, error) (CommandSpec, error)
}

// DraftArtifactRepairAdapter is implemented by adapters that can run a
// manifest/draft-only recovery prompt for runtime draft steps.
type DraftArtifactRepairAdapter interface {
	DraftArtifactRepairCommandSpec(acpruntime.Task, error) (CommandSpec, error)
}

type ActivityPolicy struct {
	MonitorArtifacts            bool
	MonitorPreArtifact          bool
	PreArtifactStallWindow      time.Duration
	RetryPreArtifactStallWindow time.Duration
	PostArtifactStallWindow     time.Duration
	PartialArtifactStallWindow  time.Duration
	PollInterval                time.Duration
	TerminateGrace              time.Duration
	PostTerminateDrain          time.Duration
}

type RecoveryPolicy struct {
	AcceptValidArtifactsAfterStop               bool
	RepairCollectManifestOnce                   bool
	RepairCollectArtifactPairOnce               bool
	RepairValidatorVerdictOnce                  bool
	RepairDraftArtifactsOnce                    bool
	RetryInvalidOrMissingArtifactsOnce          bool
	RetryZeroOutputPreArtifactStallOnce         bool
	RetryTransientProviderUnavailableRepairOnce bool
	ClassifySilentRetryExhaustionUnavailable    bool
}

func DefaultUnavailableMarkers() []string {
	return []string{
		"permission_error",
		"permission error",
		"usage limit",
		"quota exceeded",
		"quota",
		"insufficient credits",
		"credit balance",
		"rate limit",
		"rate_limit",
		"status code: 403",
		"status code: 429",
		"api error:",
		"api error: 403",
		"api error: 429",
		"premature close",
		"ssl",
		"tls",
		"certificate",
		"network",
		"socket",
		"packet length too long",
		"http2",
	}
}

type StallPhase string

const (
	StallPhasePreArtifact  StallPhase = "pre_artifact"
	StallPhasePostArtifact StallPhase = "post_artifact"
)

var (
	ErrStalledBeforeArtifacts = errors.New("runtime_stalled_before_artifacts")
	ErrStalledAfterArtifacts  = errors.New("runtime_stalled_after_artifacts")
)

type StallDiagnostic struct {
	StallPhase            StallPhase
	ArtifactState         string
	ArtifactObserved      bool
	AuthoredFileCount     int
	StdoutBytes           int
	StderrBytes           int
	LastPipeActivity      time.Time
	LastWriteRootMutation time.Time
}

type StallError struct {
	Sentinel   error
	Diagnostic StallDiagnostic
}

func (e StallError) Error() string {
	if e.Sentinel == nil {
		return "runtime_stalled"
	}
	return e.Sentinel.Error()
}

func (e StallError) Is(target error) bool {
	return target != nil && target == e.Sentinel
}

func PreflightCommand(provider acpruntime.Provider, command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return acpruntime.WrapRunnerError(
			provider,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command is empty", provider),
			nil,
		)
	}
	if _, err := exec.LookPath(command); err != nil {
		return acpruntime.WrapRunnerError(
			provider,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command %q is unavailable: %v", provider, command, err),
			err,
		)
	}
	return nil
}

func JSONTaskStdin(task acpruntime.Task) (io.Reader, error) {
	taskPayload, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime task: %w", err)
	}
	return bytes.NewReader(taskPayload), nil
}

func RunHeadlessProvider(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter) (acpruntime.Result, error) {
	if adapter == nil {
		return acpruntime.Result{}, errors.New("provider adapter is nil")
	}
	result, runErr := runProviderCommand(ctx, task, adapter, normalizeActivityPolicy(adapter.ActivityPolicy(task)))
	if runErr != nil {
		if recovered, recoveredResult, recoveredErr := recoverAfterStall(ctx, task, adapter, result, runErr); recovered {
			if recoveredErr != nil {
				return acpruntime.Result{}, recoveredErr
			}
			recoveredResult.Execution = acpruntime.NewExecution(task, adapter.Provider(), adapter.RuntimeVersion(), "succeeded", time.Now().UTC(), nil)
			return recoveredResult, nil
		}
		return acpruntime.Result{}, classifyCommandFailure(adapter, task, result, runErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		if recovered, recoveredResult, recoveredErr := recoverAfterArtifactValidationFailure(ctx, task, adapter, result, err); recovered {
			if recoveredErr != nil {
				return acpruntime.Result{}, recoveredErr
			}
			recoveredResult.Execution = acpruntime.NewExecution(task, adapter.Provider(), adapter.RuntimeVersion(), "succeeded", time.Now().UTC(), nil)
			return recoveredResult, nil
		}
		return acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "contract", "artifact validation failed", err)
	}
	result.Execution = acpruntime.NewExecution(task, adapter.Provider(), adapter.RuntimeVersion(), "succeeded", time.Now().UTC(), nil)
	return result, nil
}

func normalizeActivityPolicy(policy ActivityPolicy) ActivityPolicy {
	if policy.PreArtifactStallWindow <= 0 {
		policy.PreArtifactStallWindow = defaultPreArtifactStallWindow
	}
	if policy.RetryPreArtifactStallWindow <= 0 {
		policy.RetryPreArtifactStallWindow = defaultRetryPreArtifactWindow
	}
	if policy.PostArtifactStallWindow <= 0 {
		policy.PostArtifactStallWindow = defaultPostArtifactStallWindow
	}
	if policy.PartialArtifactStallWindow <= 0 {
		policy.PartialArtifactStallWindow = policy.PostArtifactStallWindow
	}
	if policy.PollInterval <= 0 {
		policy.PollInterval = defaultStallPollInterval
	}
	if policy.TerminateGrace <= 0 {
		policy.TerminateGrace = defaultStallTerminateGrace
	}
	if policy.PostTerminateDrain <= 0 {
		policy.PostTerminateDrain = defaultPostTerminateDrain
	}
	return policy
}

func MonitorsRuntimeArtifacts(task acpruntime.Task) bool {
	switch acpruntime.StepProviderKeyForStepID(task.StepID) {
	case acpruntime.StepProviderStep1Collect, acpruntime.StepProviderStep3Findings:
		return true
	default:
		return runtimedrafts.IsDraftStep(task.StepID)
	}
}
