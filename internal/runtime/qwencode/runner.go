package qwencode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

var (
	ErrRunnerUnavailable             = errors.New("qwen-code runner is unavailable")
	errCollectStalledAfterArtifacts  = errors.New("collect_stalled_after_artifacts")
	errCollectStalledBeforeArtifacts = errors.New("collect_stalled_before_artifacts")
	errDraftStalledAfterArtifacts    = errors.New("draft_stalled_after_artifacts")

	collectPostArtifactStallWindow = 20 * time.Second
	collectPreArtifactStallWindow  = 75 * time.Second
	collectStallPollInterval       = 2 * time.Second
	collectStallTerminateGrace     = 2 * time.Second
	collectStallPostTerminateDrain = 500 * time.Millisecond
)

type HeadlessRunner struct {
	Command string
	Args    []string
}

type retryManifestAssessment = artifactquality.ManifestAssessment

type promptRetryMode int

type collectStallPhase string

type runQwenOptions struct {
	EnableCollectStallMonitor bool
	EnableDraftStallMonitor   bool
}

type collectStallDiagnostic struct {
	StallPhase            collectStallPhase
	ManifestState         string
	AuthoredFileCount     int
	LastPipeActivity      time.Time
	LastWriteRootMutation time.Time
}

type collectStallError struct {
	Sentinel   error
	Diagnostic collectStallDiagnostic
}

type collectWriteRootSnapshot struct {
	ManifestPresent   bool
	ManifestState     string
	AuthoredFileCount int
	LastMutation      time.Time
}

type collectRepairAttempt struct {
	Attempted                bool
	ManifestStateBeforeRetry string
	Diagnostic               collectStallDiagnostic
}

type retryDiagnosticContext struct {
	LastStallPhase           collectStallPhase
	ManifestStateBeforeRetry string
	AuthoredFileCount        int
	LastPipeActivity         time.Time
	LastWriteRootMutation    time.Time
}

type commandActivityTracker struct {
	mu       sync.Mutex
	lastRead time.Time
}

type activityTrackingReader struct {
	reader  io.Reader
	tracker *commandActivityTracker
}

type commandOutputBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

const (
	promptRetryNone promptRetryMode = iota
	promptRetryParse
	promptRetryArtifact
	promptRetryDraftArtifact
	promptRetryCollectFresh
)

const (
	collectStallPhasePreArtifact  collectStallPhase = "pre_artifact"
	collectStallPhasePostArtifact collectStallPhase = "post_artifact"
)

func (e collectStallError) Error() string {
	if e.Sentinel == nil {
		return "collect_stalled"
	}
	return e.Sentinel.Error()
}

func (e collectStallError) Is(target error) bool {
	return target != nil && target == e.Sentinel
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

	taskPayload, err := json.Marshal(task)
	if err != nil {
		return acpruntime.Result{}, fmt.Errorf("marshal runner task: %w", err)
	}

	args := append([]string(nil), r.Args...)
	if len(args) == 0 {
		args = buildDefaultQwenArgs(task, buildPrompt(taskPayload, false))
	}

	options := runQwenOptions{
		EnableCollectStallMonitor: len(r.Args) == 0 && isCollectStep(task.StepID),
		EnableDraftStallMonitor:   len(r.Args) == 0 && runtimedrafts.IsDraftStep(task.StepID),
	}
	result, parseStage, parseErr, runErr := runQwenCommand(ctx, task, command, args, options)
	if runErr != nil {
		var stalled collectStallError
		if len(r.Args) == 0 && errors.As(runErr, &stalled) {
			if errors.Is(stalled, errCollectStalledAfterArtifacts) {
				return recoverCollectArtifactsAfterStall(ctx, task, taskPayload, command, result, stalled)
			}
			if errors.Is(stalled, errCollectStalledBeforeArtifacts) {
				return recoverCollectBeforeArtifactsAfterStall(ctx, task, taskPayload, command, result, stalled)
			}
			if errors.Is(stalled, errDraftStalledAfterArtifacts) {
				return recoverDraftArtifactsAfterStall(ctx, task, taskPayload, command, result, stalled)
			}
		}
		unavailableMessage := buildUnavailableFailureMessage(task, runErr, result)
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: %s", ErrRunnerUnavailable, unavailableMessage),
			result.Stdout,
			result.Stderr,
			runErr,
		)
	}
	if parseErr == nil {
		return finalizeSuccessfulQwenResult(ctx, task, taskPayload, command, result, len(r.Args) == 0)
	}
	finalStdout := result.Stdout
	finalStderr := result.Stderr

	// Live qwen output can occasionally contain malformed tokens. Retry once with
	// an explicitly stricter prompt before classifying as parse failure.
	if len(r.Args) == 0 {
		retryArgs := buildRetryQwenArgs(task, buildPromptWithModeAndHints(taskPayload, promptRetryParse, buildParseRepairHints(task.StepID, parseStage, parseErr)))
		retryResult, retryParseStage, retryParseErr, retryRunErr := runQwenCommand(ctx, task, command, retryArgs, runQwenOptions{})
		if retryRunErr != nil {
			unavailableMessage := buildUnavailableFailureMessage(task, retryRunErr, retryResult)
			return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
				acpruntime.ProviderQwenCode,
				acpruntime.ErrorCodeRunnerUnavailable,
				fmt.Sprintf("%v: %s", ErrRunnerUnavailable, unavailableMessage),
				retryResult.Stdout,
				retryResult.Stderr,
				retryRunErr,
			)
		}
		if retryParseErr == nil {
			return finalizeSuccessfulQwenResult(ctx, task, taskPayload, command, retryResult, true)
		}
		result = retryResult
		parseStage = retryParseStage
		parseErr = retryParseErr
		finalStdout = retryResult.Stdout
		finalStderr = retryResult.Stderr
	}

	return acpruntime.Result{}, wrapQwenParseFailure(
		task,
		parseStage,
		parseErr,
		acpruntime.Result{
			TaskResult: result.TaskResult,
			RawJSON:    result.RawJSON,
			Stdout:     finalStdout,
			Stderr:     finalStderr,
		},
		fmt.Sprintf("headless provider %q returned invalid taskresult", acpruntime.ProviderQwenCode),
	)
}

func buildDefaultQwenArgs(task acpruntime.Task, prompt string) []string {
	return buildQwenArgsWithIncludeDirectories(acpruntime.ResolveHeadlessIncludeDirectories(task), prompt)
}

func buildRetryQwenArgs(task acpruntime.Task, prompt string) []string {
	return buildQwenArgsWithIncludeDirectories(resolveRetryIncludeDirectories(task), prompt)
}

func buildDraftRepairQwenArgs(task acpruntime.Task, prompt string) []string {
	directories := []string{}
	for _, candidate := range []string{strings.TrimSpace(task.WriteRoot), strings.TrimSpace(task.DraftFinalRoot)} {
		if candidate == "" {
			continue
		}
		directories = append(directories, candidate)
	}
	return buildQwenArgsWithIncludeDirectories(directories, prompt)
}

func buildQwenArgsWithIncludeDirectories(includeDirs []string, prompt string) []string {
	args := []string{"--output-format", "json", "--chat-recording", "false", "--yolo", "--channel", "CI"}
	seen := map[string]struct{}{}
	for _, dir := range includeDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if _, exists := seen[dir]; exists {
			continue
		}
		seen[dir] = struct{}{}
		args = append(args, "--include-directories", dir)
	}
	args = append(args, "--prompt", prompt)
	return args
}

func finalizeSuccessfulQwenResult(
	ctx context.Context,
	task acpruntime.Task,
	taskPayload []byte,
	command string,
	current acpruntime.Result,
	allowRepair bool,
) (acpruntime.Result, error) {
	if isCollectStep(task.StepID) {
		if !allowRepair {
			return current, nil
		}
		repaired, _, repairErr := maybeRepairCollectArtifacts(ctx, task, taskPayload, command, current)
		return repaired, repairErr
	}
	if !runtimedrafts.IsDraftStep(task.StepID) {
		return current, nil
	}
	current, _, _ = reconcileRuntimeDraftOutputsAtWriteRoot(task, current)
	if _, _, err := validateRuntimeDraftArtifactsAtWriteRoot(task); err == nil {
		return current, nil
	} else if !allowRepair {
		return acpruntime.Result{}, wrapRuntimeDraftContractFailure(
			task,
			"draft.contract",
			current,
			fmt.Sprintf("runtime required draft artifacts invalid: %v", err),
			err,
		)
	}
	return maybeRepairRuntimeDraftArtifacts(ctx, task, taskPayload, command, current)
}

func isCollectStep(stepID string) bool {
	return strings.TrimSpace(stepID) == "init.step1.collect" || strings.TrimSpace(stepID) == "refresh.step1.collect"
}

func resolveRetryIncludeDirectories(task acpruntime.Task) []string {
	if shouldConstrainRetryToWriteRoot(task) {
		return []string{strings.TrimSpace(task.WriteRoot)}
	}
	return acpruntime.ResolveHeadlessIncludeDirectories(task)
}

func shouldConstrainRetryToWriteRoot(task acpruntime.Task) bool {
	if task.StepID != "init.step1.collect" && task.StepID != "refresh.step1.collect" {
		return false
	}
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return false
	}
	files, total, err := collectRetryWriteRootFiles(writeRoot, 4)
	if err != nil || total < 2 {
		return false
	}
	assessment, assessErr := assessRetryManifestAtWriteRoot(writeRoot)
	if assessErr != nil || !assessment.Rich {
		return false
	}
	for _, rel := range files {
		if rel == "shard-pack-manifest.json" {
			return true
		}
	}
	return false
}

func assessRetryManifestAtWriteRoot(writeRoot string) (retryManifestAssessment, error) {
	return artifactquality.LoadManifestAssessment(writeRoot)
}

func assessRetryManifest(raw []byte) retryManifestAssessment {
	assessment, err := artifactquality.AssessManifestBytes(raw)
	if err != nil {
		return retryManifestAssessment{}
	}
	return assessment
}

func isGenericRuntimeSummaryCitationID(id string) bool {
	return artifactquality.IsGenericRuntimeSummaryCitation(id)
}

func collectRetryWriteRootFiles(writeRoot string, limit int) ([]string, int, error) {
	root := strings.TrimSpace(writeRoot)
	if root == "" {
		return nil, 0, nil
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	if !info.IsDir() {
		return nil, 0, fmt.Errorf("write_root is not a directory: %s", root)
	}

	files := []string{}
	if walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	}); walkErr != nil {
		return nil, 0, walkErr
	}
	sort.Strings(files)

	total := len(files)
	if limit > 0 && len(files) > limit {
		files = append([]string(nil), files[:limit]...)
	}
	return files, total, nil
}
