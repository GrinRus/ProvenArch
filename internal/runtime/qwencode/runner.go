package qwencode

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/providercommon"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

type HeadlessRunner struct {
	Command string
	Args    []string
}

func (r HeadlessRunner) RuntimeMeta() contracts.RuntimeMeta {
	return contracts.RuntimeMeta{Name: string(acpruntime.ProviderQwenCode), Version: "headless"}
}

func (r HeadlessRunner) commandName() string {
	command := strings.TrimSpace(r.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv("ACP_QWEN_CMD"))
	}
	if command == "" {
		command = "qwen"
	}
	return command
}

func (r HeadlessRunner) Preflight(_ context.Context) error {
	return providercommon.PreflightCommand(acpruntime.ProviderQwenCode, r.commandName())
}

func (r HeadlessRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if err := r.Preflight(ctx); err != nil {
		return acpruntime.Result{}, err
	}
	return providercommon.RunHeadlessProvider(ctx, task, qwenAdapter{runner: r})
}

type qwenAdapter struct {
	runner HeadlessRunner
}

func (a qwenAdapter) Provider() acpruntime.Provider {
	return acpruntime.ProviderQwenCode
}

func (a qwenAdapter) RuntimeVersion() string {
	return "headless"
}

func (a qwenAdapter) CommandSpec(task acpruntime.Task) (providercommon.CommandSpec, error) {
	includeDirs := acpruntime.ResolveHeadlessIncludeDirectories(task)
	return a.commandSpecWithPrompt(task, includeDirs, buildPrompt(task))
}

func (a qwenAdapter) commandSpecWithPrompt(task acpruntime.Task, includeDirs []string, prompt string) (providercommon.CommandSpec, error) {
	commandArgs := qwenArgsWithPrompt(append([]string(nil), a.runner.Args...), prompt, task.RuntimePermissions)
	if len(commandArgs) == 0 {
		commandArgs = buildQwenArgsWithPermissions(includeDirs, prompt, task.RuntimePermissions)
	}
	return providercommon.CommandSpec{
		Provider:    acpruntime.ProviderQwenCode,
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Dir:         strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)),
		PromptBytes: len([]byte(prompt)),
		IncludeDirs: includeDirs,
	}, nil
}

func (a qwenAdapter) CollectManifestRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderQwenCode, providercommon.FocusedRepairCollectManifest, validationErr, a.commandSpecWithPrompt)
}

func (a qwenAdapter) CollectArtifactPairRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderQwenCode, providercommon.FocusedRepairCollectArtifactPair, validationErr, a.commandSpecWithPrompt)
}

func (a qwenAdapter) ValidatorVerdictRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderQwenCode, providercommon.FocusedRepairValidatorVerdict, validationErr, a.commandSpecWithPrompt)
}

func (a qwenAdapter) DraftArtifactRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderQwenCode, providercommon.FocusedRepairDraftArtifacts, validationErr, a.commandSpecWithPrompt)
}

func (a qwenAdapter) ValidateArtifacts(task acpruntime.Task) error {
	return providercommon.ValidateRuntimeArtifacts(task, acpruntime.ProviderQwenCode)
}

func (a qwenAdapter) ActivityPolicy(task acpruntime.Task) providercommon.ActivityPolicy {
	monitorArtifacts := providercommon.MonitorsRuntimeArtifacts(task)
	policy := providercommon.ActivityPolicy{
		MonitorArtifacts:           monitorArtifacts,
		MonitorPreArtifact:         monitorArtifacts,
		PreArtifactStallWindow:     180 * time.Second,
		PartialArtifactStallWindow: 90 * time.Second,
	}
	if runtimedrafts.IsDraftStep(task.StepID) {
		policy.ValidArtifactStopWindow = 2 * time.Minute
	}
	return policy
}

func (a qwenAdapter) RecoveryPolicy(_ acpruntime.Task) providercommon.RecoveryPolicy {
	return providercommon.RecoveryPolicy{
		AcceptValidArtifactsAfterStop:               true,
		RepairCollectManifestOnce:                   true,
		RepairCollectArtifactPairOnce:               true,
		RepairValidatorVerdictOnce:                  true,
		RepairDraftArtifactsOnce:                    true,
		RetryInvalidOrMissingArtifactsOnce:          true,
		RetryZeroOutputPreArtifactStallOnce:         true,
		RetryTransientProviderUnavailableRepairOnce: true,
		ClassifySilentRetryExhaustionUnavailable:    true,
	}
}

func (a qwenAdapter) UnavailableMarkers() []string {
	return providercommon.DefaultUnavailableMarkers()
}

func qwenArgsWithPrompt(args []string, prompt string, permissions acpruntime.PermissionValues) []string {
	if len(args) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(args)+2)
	promptSet := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		trimmed := strings.TrimSpace(arg)
		switch {
		case strings.HasPrefix(trimmed, "--prompt="):
			if !promptSet {
				normalized = append(normalized, "--prompt="+prompt)
				promptSet = true
			}
			continue
		case strings.HasPrefix(trimmed, "-p="):
			if !promptSet {
				normalized = append(normalized, "-p="+prompt)
				promptSet = true
			}
			continue
		}
		switch trimmed {
		case "-p", "--prompt":
			if !promptSet {
				normalized = append(normalized, trimmed, prompt)
				promptSet = true
			}
			if i+1 < len(args) && qwenPromptFlagValueLooksPresent(args[i+1]) {
				i++
			}
			continue
		}
		if strings.TrimSpace(permissions.Mode) == acpruntime.PermissionModeManaged && (trimmed == "--yolo" || strings.HasPrefix(trimmed, "--yolo=")) {
			continue
		}
		normalized = append(normalized, arg)
	}
	if !promptSet {
		normalized = append(normalized, "-p", prompt)
	}
	return ensureQwenActivityArgs(normalized)
}

func qwenPromptFlagValueLooksPresent(arg string) bool {
	trimmed := strings.TrimSpace(arg)
	return trimmed == "" || !strings.HasPrefix(trimmed, "-")
}

func ensureQwenActivityArgs(args []string) []string {
	hasOutputFormat := false
	outputFormat := ""
	hasPartialMessages := false
	for i, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "--output-format" {
			hasOutputFormat = true
			if i+1 < len(args) {
				outputFormat = strings.TrimSpace(args[i+1])
			}
		}
		if strings.HasPrefix(trimmed, "--output-format=") {
			hasOutputFormat = true
			outputFormat = strings.TrimSpace(strings.TrimPrefix(trimmed, "--output-format="))
		}
		if trimmed == "--include-partial-messages" {
			hasPartialMessages = true
		}
	}
	addOutputFormat := !hasOutputFormat
	addPartialMessages := !hasPartialMessages && (!hasOutputFormat || outputFormat == "stream-json")
	if !addOutputFormat && !addPartialMessages {
		return args
	}
	extra := []string{}
	if addOutputFormat {
		extra = append(extra, "--output-format", "stream-json")
	}
	if addPartialMessages {
		extra = append(extra, "--include-partial-messages")
	}
	promptIndex := len(args)
	for i, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "-p" || trimmed == "--prompt" || strings.HasPrefix(trimmed, "-p=") || strings.HasPrefix(trimmed, "--prompt=") {
			promptIndex = i
			break
		}
	}
	normalized := make([]string, 0, len(args)+len(extra))
	normalized = append(normalized, args[:promptIndex]...)
	normalized = append(normalized, extra...)
	normalized = append(normalized, args[promptIndex:]...)
	return normalized
}
