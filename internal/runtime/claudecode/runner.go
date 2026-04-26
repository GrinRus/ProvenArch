package claudecode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
		command = strings.TrimSpace(os.Getenv("ACP_CLAUDE_CMD"))
	}
	if command == "" {
		command = "claude-code"
	}
	return command
}

func (r HeadlessRunner) Preflight(_ context.Context) error {
	command := r.commandName()
	if _, err := exec.LookPath(command); err != nil {
		return acpruntime.WrapRunnerError(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command %q is unavailable: %v", acpruntime.ProviderClaudeCode, command, err),
			err,
		)
	}
	return nil
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
	commandArgs := append([]string(nil), a.runner.Args...)
	if len(commandArgs) == 0 {
		commandArgs = buildClaudeArgsWithIncludeDirectories(includeDirs, buildPrompt(task))
	}
	stdin, err := providercommon.JSONTaskStdin(task)
	if err != nil {
		return providercommon.CommandSpec{}, err
	}
	return providercommon.CommandSpec{
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Stdin:       stdin,
		Dir:         strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)),
		IncludeDirs: includeDirs,
	}, nil
}

func (a claudeAdapter) CollectManifestRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	includeDirs := acpruntime.ResolveHeadlessCollectRepairIncludeDirectories(task)
	repairTask := task
	repairTask.ReadContextRoots = append([]string(nil), includeDirs...)
	prompt := promptcontract.ComposeCollectManifestRepairPrompt(acpruntime.ProviderClaudeCode, repairTask, validationErr)
	commandArgs := append([]string(nil), a.runner.Args...)
	if len(commandArgs) == 0 {
		commandArgs = buildClaudeArgsWithIncludeDirectories(includeDirs, prompt)
	}
	stdin, err := providercommon.JSONTaskStdin(repairTask)
	if err != nil {
		return providercommon.CommandSpec{}, err
	}
	return providercommon.CommandSpec{
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Stdin:       stdin,
		Dir:         strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)),
		IncludeDirs: includeDirs,
	}, nil
}

func (a claudeAdapter) ValidateArtifacts(task acpruntime.Task) error {
	return providercommon.ValidateRuntimeArtifacts(task, acpruntime.ProviderClaudeCode)
}

func (a claudeAdapter) ActivityPolicy(task acpruntime.Task) providercommon.ActivityPolicy {
	monitorArtifacts := providercommon.MonitorsRuntimeArtifacts(task)
	return providercommon.ActivityPolicy{
		MonitorArtifacts:   monitorArtifacts,
		MonitorPreArtifact: monitorArtifacts,
	}
}

func (a claudeAdapter) RecoveryPolicy(_ acpruntime.Task) providercommon.RecoveryPolicy {
	return providercommon.RecoveryPolicy{
		AcceptValidArtifactsAfterStop: true,
		RepairCollectManifestOnce:     true,
	}
}

func (a claudeAdapter) UnavailableMarkers() []string {
	return providercommon.DefaultUnavailableMarkers()
}

func buildDefaultClaudeArgs(task acpruntime.Task, prompt string) []string {
	return buildClaudeArgsWithIncludeDirectories(acpruntime.ResolveHeadlessIncludeDirectories(task), prompt)
}

func buildClaudeArgsWithIncludeDirectories(includeDirs []string, prompt string) []string {
	args := []string{"--output-format", "json", "--permission-mode", "bypassPermissions"}
	for _, dir := range includeDirs {
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "-p", prompt)
	return args
}

func buildPrompt(task acpruntime.Task) string {
	return promptcontract.ComposeArtifactOnlyPrompt(acpruntime.ProviderClaudeCode, task)
}
