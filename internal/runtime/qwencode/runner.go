package qwencode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
	"github.com/GrinRus/ProvenArch/internal/runtime/providercommon"
)

type HeadlessRunner struct {
	Command string
	Args    []string
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
	command := r.commandName()
	if _, err := exec.LookPath(command); err != nil {
		return acpruntime.WrapRunnerError(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command %q is unavailable: %v", acpruntime.ProviderQwenCode, command, err),
			err,
		)
	}
	return nil
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
	prompt := buildPrompt(task)
	commandArgs := qwenArgsWithPrompt(append([]string(nil), a.runner.Args...), prompt)
	if len(commandArgs) == 0 {
		commandArgs = buildQwenArgsWithIncludeDirectories(includeDirs, prompt)
	}
	return providercommon.CommandSpec{
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Dir:         strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)),
		IncludeDirs: includeDirs,
	}, nil
}

func (a qwenAdapter) CollectManifestRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	includeDirs := acpruntime.ResolveHeadlessCollectRepairIncludeDirectories(task)
	repairTask := task
	repairTask.ReadContextRoots = append([]string(nil), includeDirs...)
	prompt := promptcontract.ComposeCollectManifestRepairPrompt(acpruntime.ProviderQwenCode, repairTask, validationErr)
	commandArgs := qwenArgsWithPrompt(append([]string(nil), a.runner.Args...), prompt)
	if len(commandArgs) == 0 {
		commandArgs = buildQwenArgsWithIncludeDirectories(includeDirs, prompt)
	}
	return providercommon.CommandSpec{
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Dir:         strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)),
		IncludeDirs: includeDirs,
	}, nil
}

func (a qwenAdapter) CollectArtifactPairRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	includeDirs := acpruntime.ResolveHeadlessCollectRepairIncludeDirectories(task)
	repairTask := task
	repairTask.ReadContextRoots = append([]string(nil), includeDirs...)
	prompt := promptcontract.ComposeCollectArtifactPairRepairPrompt(acpruntime.ProviderQwenCode, repairTask, validationErr)
	commandArgs := qwenArgsWithPrompt(append([]string(nil), a.runner.Args...), prompt)
	if len(commandArgs) == 0 {
		commandArgs = buildQwenArgsWithIncludeDirectories(includeDirs, prompt)
	}
	return providercommon.CommandSpec{
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Dir:         strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)),
		IncludeDirs: includeDirs,
	}, nil
}

func (a qwenAdapter) ValidatorVerdictRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	includeDirs := acpruntime.ResolveHeadlessValidatorRepairIncludeDirectories(task)
	repairTask := task
	repairTask.ReadContextRoots = append([]string(nil), includeDirs...)
	prompt := promptcontract.ComposeValidatorVerdictRepairPrompt(acpruntime.ProviderQwenCode, repairTask, validationErr)
	commandArgs := qwenArgsWithPrompt(append([]string(nil), a.runner.Args...), prompt)
	if len(commandArgs) == 0 {
		commandArgs = buildQwenArgsWithIncludeDirectories(includeDirs, prompt)
	}
	return providercommon.CommandSpec{
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Dir:         strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)),
		IncludeDirs: includeDirs,
	}, nil
}

func (a qwenAdapter) DraftArtifactRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	includeDirs := acpruntime.ResolveHeadlessDraftRepairIncludeDirectories(task)
	repairTask := task
	repairTask.ReadContextRoots = append([]string(nil), includeDirs...)
	prompt := promptcontract.ComposeDraftArtifactRepairPrompt(acpruntime.ProviderQwenCode, repairTask, validationErr)
	commandArgs := qwenArgsWithPrompt(append([]string(nil), a.runner.Args...), prompt)
	if len(commandArgs) == 0 {
		commandArgs = buildQwenArgsWithIncludeDirectories(includeDirs, prompt)
	}
	return providercommon.CommandSpec{
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Dir:         strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)),
		IncludeDirs: includeDirs,
	}, nil
}

func (a qwenAdapter) ValidateArtifacts(task acpruntime.Task) error {
	return providercommon.ValidateRuntimeArtifacts(task, acpruntime.ProviderQwenCode)
}

func (a qwenAdapter) ActivityPolicy(task acpruntime.Task) providercommon.ActivityPolicy {
	if len(a.runner.Args) != 0 {
		return providercommon.ActivityPolicy{}
	}
	monitorArtifacts := providercommon.MonitorsRuntimeArtifacts(task)
	return providercommon.ActivityPolicy{
		MonitorArtifacts:           monitorArtifacts,
		MonitorPreArtifact:         monitorArtifacts,
		PartialArtifactStallWindow: 90 * time.Second,
	}
}

func (a qwenAdapter) RecoveryPolicy(_ acpruntime.Task) providercommon.RecoveryPolicy {
	return providercommon.RecoveryPolicy{
		AcceptValidArtifactsAfterStop:            true,
		RepairCollectManifestOnce:                true,
		RepairCollectArtifactPairOnce:            true,
		RepairValidatorVerdictOnce:               true,
		RepairDraftArtifactsOnce:                 true,
		RetryInvalidOrMissingArtifactsOnce:       true,
		ClassifySilentRetryExhaustionUnavailable: true,
	}
}

func (a qwenAdapter) UnavailableMarkers() []string {
	return providercommon.DefaultUnavailableMarkers()
}

func qwenArgsWithPrompt(args []string, prompt string) []string {
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
		normalized = append(normalized, arg)
	}
	if !promptSet {
		normalized = append(normalized, "-p", prompt)
	}
	return normalized
}

func qwenPromptFlagValueLooksPresent(arg string) bool {
	trimmed := strings.TrimSpace(arg)
	return trimmed == "" || !strings.HasPrefix(trimmed, "-")
}
