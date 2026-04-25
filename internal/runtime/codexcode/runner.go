package codexcode

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
	"github.com/GrinRus/ProvenArch/internal/runtime/providercommon"
)

var ErrRunnerUnavailable = errors.New("codex-code runner is unavailable")

type HeadlessRunner struct {
	Command string
	Args    []string
}

func (r HeadlessRunner) commandName() string {
	command := strings.TrimSpace(r.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv("ACP_CODEX_CMD"))
	}
	if command == "" {
		command = "codex"
	}
	return command
}

func (r HeadlessRunner) Preflight(_ context.Context) error {
	command := r.commandName()
	if _, err := exec.LookPath(command); err != nil {
		return acpruntime.WrapRunnerError(
			acpruntime.ProviderCodexCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command %q is unavailable: %v", acpruntime.ProviderCodexCode, command, err),
			err,
		)
	}
	return nil
}

func (r HeadlessRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if err := r.Preflight(ctx); err != nil {
		return acpruntime.Result{}, err
	}
	return providercommon.RunHeadlessProvider(ctx, task, codexAdapter{runner: r})
}

type codexAdapter struct {
	runner HeadlessRunner
}

func (a codexAdapter) Provider() acpruntime.Provider {
	return acpruntime.ProviderCodexCode
}

func (a codexAdapter) RuntimeVersion() string {
	return "headless"
}

func (a codexAdapter) CommandSpec(task acpruntime.Task) (providercommon.CommandSpec, error) {
	includeDirs := acpruntime.ResolveHeadlessIncludeDirectories(task)
	cwd := strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task))
	commandArgs := append([]string(nil), a.runner.Args...)
	if len(commandArgs) == 0 {
		commandArgs = buildCodexArgsWithIncludeDirectories(cwd, includeDirs)
	}
	return providercommon.CommandSpec{
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Stdin:       strings.NewReader(buildPrompt(task)),
		Dir:         cwd,
		IncludeDirs: includeDirs,
	}, nil
}

func (a codexAdapter) ValidateArtifacts(task acpruntime.Task) error {
	return providercommon.ValidateRuntimeArtifacts(task, acpruntime.ProviderCodexCode)
}

func (a codexAdapter) ActivityPolicy(_ acpruntime.Task) providercommon.ActivityPolicy {
	return providercommon.ActivityPolicy{
		MonitorArtifacts: true,
	}
}

func (a codexAdapter) RecoveryPolicy(_ acpruntime.Task) providercommon.RecoveryPolicy {
	return providercommon.RecoveryPolicy{
		AcceptValidArtifactsAfterStop: true,
	}
}

func (a codexAdapter) UnavailableMarkers() []string {
	return providercommon.DefaultUnavailableMarkers()
}

func buildDefaultCodexArgs(task acpruntime.Task) []string {
	cwd := strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task))
	return buildCodexArgsWithIncludeDirectories(cwd, acpruntime.ResolveHeadlessIncludeDirectories(task))
}

func buildCodexArgsWithIncludeDirectories(cwd string, includeDirs []string) []string {
	args := []string{
		"exec",
		"--json",
		"--color", "never",
		"--skip-git-repo-check",
		"--sandbox", "danger-full-access",
	}
	if cwd != "" {
		args = append(args, "--cd", cwd)
	}
	for _, dir := range includeDirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "--ephemeral", "-")
	return args
}

func buildPrompt(task acpruntime.Task) string {
	return promptcontract.ComposeArtifactOnlyPrompt(acpruntime.ProviderCodexCode, task)
}
