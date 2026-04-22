package qwencode

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

var (
	errCollectStalledAfterArtifacts  = errors.New("collect_stalled_after_artifacts")
	errCollectStalledBeforeArtifacts = errors.New("collect_stalled_before_artifacts")
	errDraftStalledAfterArtifacts    = errors.New("draft_stalled_after_artifacts")
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
	command := r.commandName()

	options := runQwenOptions{
		EnableCollectStallMonitor: len(r.Args) == 0 && isCollectStep(task.StepID),
		EnableDraftStallMonitor:   len(r.Args) == 0 && isDraftStep(task.StepID),
	}
	result, runErr := runQwenCommand(ctx, task, command, r.Args, options)
	if runErr != nil {
		return acpruntime.Result{}, classifyRunFailure(task, result, runErr)
	}
	if err := repairAndValidateArtifacts(task); err != nil {
		return acpruntime.Result{}, wrapArtifactContractFailure(task, "contract", result, "artifact validation failed", err)
	}
	result.Execution = acpruntime.NewExecution(task, acpruntime.ProviderQwenCode, "headless", "succeeded", time.Now().UTC(), nil)
	return result, nil
}
