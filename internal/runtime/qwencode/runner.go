package qwencode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
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
	commandArgs := append([]string(nil), a.runner.Args...)
	if len(commandArgs) == 0 {
		commandArgs = buildQwenArgsWithIncludeDirectories(includeDirs, buildPrompt(task))
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

func (a qwenAdapter) ValidateArtifacts(task acpruntime.Task) error {
	return repairAndValidateArtifacts(task)
}

func (a qwenAdapter) ActivityPolicy(task acpruntime.Task) providercommon.ActivityPolicy {
	if len(a.runner.Args) != 0 {
		return providercommon.ActivityPolicy{}
	}
	return providercommon.ActivityPolicy{
		MonitorArtifacts:           isCollectStep(task.StepID) || isDraftStep(task.StepID) || isFindingsStep(task.StepID),
		MonitorPreArtifact:         isCollectStep(task.StepID),
		PartialArtifactStallWindow: 90 * time.Second,
	}
}

func (a qwenAdapter) RecoveryPolicy(_ acpruntime.Task) providercommon.RecoveryPolicy {
	return providercommon.RecoveryPolicy{
		AcceptValidArtifactsAfterStop:            true,
		RetryInvalidOrMissingArtifactsOnce:       true,
		ClassifySilentRetryExhaustionUnavailable: true,
	}
}

func (a qwenAdapter) UnavailableMarkers() []string {
	return providercommon.DefaultUnavailableMarkers()
}
