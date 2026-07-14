package claudecode

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
	"github.com/GrinRus/ProvenArch/internal/runtime/providercommon"
)

type HeadlessRunner struct {
	Command string
	Args    []string
}

func (r HeadlessRunner) RuntimeMeta() contracts.RuntimeMeta {
	return contracts.RuntimeMeta{Name: string(acpruntime.ProviderClaudeCode), Version: "headless"}
}

func (r HeadlessRunner) commandName() string {
	command := strings.TrimSpace(r.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv("ACP_CLAUDE_CMD"))
	}
	if command == "" {
		if _, err := exec.LookPath("claude"); err == nil {
			return "claude"
		}
		command = "claude-code"
	}
	return command
}

func (r HeadlessRunner) Preflight(_ context.Context) error {
	return providercommon.PreflightCommand(acpruntime.ProviderClaudeCode, r.commandName())
}

func (r HeadlessRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if err := r.Preflight(ctx); err != nil {
		return acpruntime.Result{}, err
	}
	return providercommon.RunHeadlessProvider(ctx, task, claudeAdapter{runner: r})
}

type claudeAdapter struct {
	runner HeadlessRunner
}

func (a claudeAdapter) Provider() acpruntime.Provider {
	return acpruntime.ProviderClaudeCode
}

func (a claudeAdapter) RuntimeVersion() string {
	return "headless"
}

func (a claudeAdapter) CommandSpec(task acpruntime.Task) (providercommon.CommandSpec, error) {
	includeDirs := acpruntime.ResolveHeadlessIncludeDirectories(task)
	return a.commandSpecWithPrompt(task, includeDirs, buildPrompt(task))
}

func (a claudeAdapter) commandSpecWithPrompt(task acpruntime.Task, includeDirs []string, prompt string) (providercommon.CommandSpec, error) {
	commandArgs := append([]string(nil), a.runner.Args...)
	if len(commandArgs) == 0 {
		commandArgs = buildClaudeArgsWithPermissions(includeDirs, prompt, task.RuntimePermissions)
	} else if strings.TrimSpace(task.RuntimePermissions.Mode) == acpruntime.PermissionModeManaged {
		commandArgs = stripClaudeBypassPermissionArgs(commandArgs)
	}
	stdin, err := providercommon.JSONTaskStdin(task)
	if err != nil {
		return providercommon.CommandSpec{}, err
	}
	return providercommon.CommandSpec{
		Provider:    acpruntime.ProviderClaudeCode,
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Stdin:       stdin,
		Dir:         strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)),
		PromptBytes: len([]byte(prompt)),
		IncludeDirs: includeDirs,
	}, nil
}

func stripClaudeBypassPermissionArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		trimmed := strings.TrimSpace(args[i])
		if trimmed == "--dangerously-skip-permissions" {
			continue
		}
		if strings.HasPrefix(trimmed, "--permission-mode=") && strings.TrimSpace(strings.TrimPrefix(trimmed, "--permission-mode=")) == "bypassPermissions" {
			continue
		}
		if trimmed == "--permission-mode" && i+1 < len(args) && strings.TrimSpace(args[i+1]) == "bypassPermissions" {
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}

func (a claudeAdapter) CollectManifestRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderClaudeCode, providercommon.FocusedRepairCollectManifest, validationErr, a.commandSpecWithPrompt)
}

func (a claudeAdapter) CollectArtifactPairRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderClaudeCode, providercommon.FocusedRepairCollectArtifactPair, validationErr, a.commandSpecWithPrompt)
}

func (a claudeAdapter) ValidatorVerdictRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderClaudeCode, providercommon.FocusedRepairValidatorVerdict, validationErr, a.commandSpecWithPrompt)
}

func (a claudeAdapter) DraftArtifactRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderClaudeCode, providercommon.FocusedRepairDraftArtifacts, validationErr, a.commandSpecWithPrompt)
}

func (a claudeAdapter) DraftArtifactEnrichmentCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderClaudeCode, providercommon.FocusedRepairDraftEnrichment, validationErr, a.commandSpecWithPrompt)
}

func (a claudeAdapter) ValidateArtifacts(task acpruntime.Task) error {
	return providercommon.ValidateRuntimeArtifacts(task, acpruntime.ProviderClaudeCode)
}

func (a claudeAdapter) ActivityPolicy(task acpruntime.Task) providercommon.ActivityPolicy {
	monitorArtifacts := providercommon.MonitorsRuntimeArtifacts(task)
	preArtifactWindow := 180 * time.Second
	retryPreArtifactWindow := time.Duration(0)
	if acpruntime.StepProviderKeyForStepID(task.StepID) == acpruntime.StepProviderStep1Collect {
		preArtifactWindow = 5 * time.Minute
		retryPreArtifactWindow = 5 * time.Minute
	}
	return providercommon.WithCollectArtifactEnrichmentWindow(task, providercommon.ActivityPolicy{
		MonitorArtifacts:            monitorArtifacts,
		MonitorPreArtifact:          monitorArtifacts,
		PreArtifactStallWindow:      preArtifactWindow,
		RetryPreArtifactStallWindow: retryPreArtifactWindow,
	})
}

func (a claudeAdapter) RecoveryPolicy(task acpruntime.Task) providercommon.RecoveryPolicy {
	policy := providercommon.RecoveryPolicy{
		AcceptValidArtifactsAfterStop:            true,
		RepairCollectManifestOnce:                true,
		RepairCollectArtifactPairOnce:            true,
		RepairValidatorVerdictOnce:               true,
		RepairDraftArtifactsOnce:                 true,
		RepairDraftArtifactEnrichmentOnce:        true,
		RetryInvalidOrMissingArtifactsOnce:       true,
		ClassifySilentRetryExhaustionUnavailable: true,
	}
	switch acpruntime.StepProviderKeyForStepID(task.StepID) {
	case acpruntime.StepProviderStep0Constitution, acpruntime.StepProviderStep1Collect, acpruntime.StepProviderStep3Findings, acpruntime.StepProviderStep4Proposals:
		policy.RetryZeroOutputPreArtifactStallOnce = true
	}
	return policy
}

func (a claudeAdapter) UnavailableMarkers() []string {
	return providercommon.DefaultUnavailableMarkers()
}

func buildClaudeArgsWithIncludeDirectories(includeDirs []string, prompt string) []string {
	return buildClaudeArgsWithPermissions(includeDirs, prompt, acpruntime.DefaultPermissions())
}

func buildClaudeArgsWithPermissions(includeDirs []string, prompt string, permissions acpruntime.PermissionValues) []string {
	args := []string{"--output-format", "json"}
	if strings.TrimSpace(permissions.Mode) != acpruntime.PermissionModeManaged {
		args = append(args, "--permission-mode", "bypassPermissions")
	}
	for _, dir := range includeDirs {
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "-p", prompt)
	return args
}

func buildPrompt(task acpruntime.Task) string {
	return promptcontract.ComposeArtifactOnlyPrompt(acpruntime.ProviderClaudeCode, task)
}
