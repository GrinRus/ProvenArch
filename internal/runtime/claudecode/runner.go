package claudecode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultbinding"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultcompat"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultextractor"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

var (
	ErrRunnerUnavailable = errors.New("claude-code runner is unavailable")
	errRunnerStalled     = errors.New("runner stalled due to output inactivity")
)

var findingsIdleSilenceTimeout = 10 * time.Minute

type HeadlessRunner struct {
	Command string
	Args    []string
}

type builtPrompt struct {
	Text string
	Pack promptcontract.PromptPack
}

type promptRetryMode int

const (
	promptRetryNone promptRetryMode = iota
	promptRetryParse
	promptRetryArtifact
)

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
	command := r.commandName()

	taskPayload, err := json.Marshal(task)
	if err != nil {
		return acpruntime.Result{}, fmt.Errorf("marshal runner task: %w", err)
	}
	if len(r.Args) > 0 || !isNativeDirectClaudeCommand(command) {
		return runStdinPassthrough(ctx, command, r.Args, task, taskPayload)
	}

	return runNativeDirectClaude(ctx, command, task, taskPayload)
}

func isNativeDirectClaudeCommand(command string) bool {
	base := strings.ToLower(strings.TrimSpace(filepath.Base(command)))
	return base == "claude" || base == "claude.exe"
}

func runStdinPassthrough(ctx context.Context, command string, args []string, task acpruntime.Task, taskPayload []byte) (acpruntime.Result, error) {
	result, parseStage, parseErr, runErr := runClaudeCommand(ctx, task, command, append([]string(nil), args...), taskPayload)
	if runErr != nil {
		if isRunnerStalledError(runErr) {
			stalledMessage := buildUnavailableFailureMessage(task, runErr, result)
			return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
				acpruntime.ProviderClaudeCode,
				acpruntime.ErrorCodeRunnerStalled,
				fmt.Sprintf("headless provider %q stalled: %s", acpruntime.ProviderClaudeCode, stalledMessage),
				result.Stdout,
				result.Stderr,
				runErr,
			)
		}
		unavailableMessage := buildUnavailableFailureMessage(task, runErr, result)
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: %s", ErrRunnerUnavailable, unavailableMessage),
			result.Stdout,
			result.Stderr,
			runErr,
		)
	}
	if parseErr != nil {
		parseFailureMessage := buildParseFailureMessage(task, parseStage, parseErr, result)
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q returned invalid taskresult: %s", acpruntime.ProviderClaudeCode, parseFailureMessage),
			result.Stdout,
			result.Stderr,
			parseErr,
		)
	}
	return result, nil
}

func runNativeDirectClaude(ctx context.Context, command string, task acpruntime.Task, taskPayload []byte) (acpruntime.Result, error) {
	initialPrompt := buildDirectPromptArtifact(taskPayload, promptRetryNone, false, nil)
	recordPromptArtifacts(task, "initial", initialPrompt, acpruntime.ProviderClaudeCode, acpruntime.ResolveHeadlessIncludeDirectories(task), taskPayload)
	args := buildNativeDirectClaudeArgs(task, initialPrompt.Text)
	result, parseStage, parseErr, runErr := runClaudeCommand(ctx, task, command, args, nil)
	if runErr != nil {
		if isRunnerStalledError(runErr) {
			retryPrompt := buildDirectPromptArtifact(taskPayload, promptRetryParse, false, buildStallRetryHints(runErr))
			recordPromptArtifacts(task, "stall-retry", retryPrompt, acpruntime.ProviderClaudeCode, acpruntime.ResolveHeadlessIncludeDirectories(task), taskPayload)
			retryArgs := buildNativeDirectClaudeArgs(task, retryPrompt.Text)
			retryResult, retryParseStage, retryParseErr, retryRunErr := runClaudeCommand(ctx, task, command, retryArgs, nil)
			if retryRunErr != nil {
				if isRunnerStalledError(retryRunErr) {
					stalledMessage := buildUnavailableFailureMessage(task, retryRunErr, retryResult)
					return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
						acpruntime.ProviderClaudeCode,
						acpruntime.ErrorCodeRunnerStalled,
						fmt.Sprintf("headless provider %q stalled after retry: %s", acpruntime.ProviderClaudeCode, stalledMessage),
						retryResult.Stdout,
						retryResult.Stderr,
						retryRunErr,
					)
				}
				unavailableMessage := buildUnavailableFailureMessage(task, retryRunErr, retryResult)
				return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
					acpruntime.ProviderClaudeCode,
					acpruntime.ErrorCodeRunnerUnavailable,
					fmt.Sprintf("%v: %s", ErrRunnerUnavailable, unavailableMessage),
					retryResult.Stdout,
					retryResult.Stderr,
					retryRunErr,
				)
			}
			if retryParseErr == nil {
				return maybeRepairCollectArtifacts(ctx, task, taskPayload, command, retryResult)
			}
			parseFailureMessage := buildParseFailureMessage(task, retryParseStage, retryParseErr, retryResult)
			return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
				acpruntime.ProviderClaudeCode,
				acpruntime.ErrorCodeRunnerStalled,
				fmt.Sprintf("headless provider %q stalled after retry (invalid taskresult): %s", acpruntime.ProviderClaudeCode, parseFailureMessage),
				retryResult.Stdout,
				retryResult.Stderr,
				retryParseErr,
			)
		}
		unavailableMessage := buildUnavailableFailureMessage(task, runErr, result)
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: %s", ErrRunnerUnavailable, unavailableMessage),
			result.Stdout,
			result.Stderr,
			runErr,
		)
	}
	if parseErr == nil {
		return maybeRepairCollectArtifacts(ctx, task, taskPayload, command, result)
	}

	retryPrompt := buildDirectPromptArtifact(
		taskPayload,
		promptRetryParse,
		parseStage == "extract" && (isEnvelopeResultEmptyError(parseErr) || isEnvelopeResultMalformedError(parseErr)),
		buildParseRepairHints(parseStage, parseErr),
	)
	recordPromptArtifacts(task, "parse-retry", retryPrompt, acpruntime.ProviderClaudeCode, acpruntime.ResolveHeadlessIncludeDirectories(task), taskPayload)
	retryArgs := buildNativeDirectClaudeArgs(task, retryPrompt.Text)
	retryResult, retryParseStage, retryParseErr, retryRunErr := runClaudeCommand(ctx, task, command, retryArgs, nil)
	if retryRunErr != nil {
		unavailableMessage := buildUnavailableFailureMessage(task, retryRunErr, retryResult)
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: %s", ErrRunnerUnavailable, unavailableMessage),
			retryResult.Stdout,
			retryResult.Stderr,
			retryRunErr,
		)
	}
	if retryParseErr == nil {
		return maybeRepairCollectArtifacts(ctx, task, taskPayload, command, retryResult)
	}

	parseFailureMessage := buildParseFailureMessage(task, retryParseStage, retryParseErr, retryResult)
	return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
		acpruntime.ProviderClaudeCode,
		acpruntime.ErrorCodeRunnerParseFailed,
		fmt.Sprintf("headless provider %q returned invalid taskresult: %s", acpruntime.ProviderClaudeCode, parseFailureMessage),
		retryResult.Stdout,
		retryResult.Stderr,
		retryParseErr,
	)
}

func buildNativeDirectClaudeArgs(task acpruntime.Task, prompt string) []string {
	args := []string{"--output-format", "json", "--permission-mode", "bypassPermissions"}
	for _, dir := range acpruntime.ResolveHeadlessIncludeDirectories(task) {
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "-p", prompt)
	return args
}

func maybeRepairCollectArtifacts(
	ctx context.Context,
	task acpruntime.Task,
	taskPayload []byte,
	command string,
	current acpruntime.Result,
) (acpruntime.Result, error) {
	if task.StepID != "init.step1.collect" && task.StepID != "refresh.step1.collect" {
		return current, nil
	}
	if strings.TrimSpace(task.WriteRoot) == "" {
		return current, nil
	}

	ensureErr := artifactquality.EnsureCanonicalCollectManifest(task, current.TaskResult)
	assessment, err := artifactquality.ValidateCollectManifestAtWriteRoot(task.WriteRoot)
	if ensureErr != nil {
		err = ensureErr
	}
	if err == nil && assessment.Rich {
		return current, nil
	}
	initialProblem := artifactquality.DescribeAssessmentProblem(assessment, err)

	snapshot, err := artifactquality.SnapshotWriteRoot(task.WriteRoot)
	if err != nil {
		return acpruntime.Result{}, wrapArtifactContractFailure(
			current.Stdout,
			current.Stderr,
			fmt.Sprintf("collect artifacts require repair (%s), but write_root snapshot failed: %v", initialProblem, err),
			err,
		)
	}
	defer func() {
		_ = snapshot.Cleanup()
	}()

	repairPrompt := buildDirectPromptArtifact(taskPayload, promptRetryArtifact, false, buildArtifactRepairHints(initialProblem))
	recordPromptArtifacts(task, "artifact-repair", repairPrompt, acpruntime.ProviderClaudeCode, acpruntime.ResolveHeadlessIncludeDirectories(task), taskPayload)
	repairArgs := buildNativeDirectClaudeArgs(task, repairPrompt.Text)
	repaired, repairParseStage, parseErr, runErr := runClaudeCommand(ctx, task, command, repairArgs, nil)
	if runErr != nil {
		_ = snapshot.Restore()
		if isRunnerStalledError(runErr) {
			stalledMessage := buildUnavailableFailureMessage(task, runErr, repaired)
			return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
				acpruntime.ProviderClaudeCode,
				acpruntime.ErrorCodeRunnerStalled,
				fmt.Sprintf("headless provider %q stalled during collect artifact repair retry after %s: %s", acpruntime.ProviderClaudeCode, initialProblem, stalledMessage),
				repaired.Stdout,
				repaired.Stderr,
				runErr,
			)
		}
		unavailableMessage := buildUnavailableFailureMessage(task, runErr, repaired)
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: artifact repair retry failed after %s: %s", ErrRunnerUnavailable, initialProblem, unavailableMessage),
			repaired.Stdout,
			repaired.Stderr,
			runErr,
		)
	}
	if parseErr != nil {
		_ = snapshot.Restore()
		parseFailureMessage := buildParseFailureMessage(task, "artifact_repair."+repairParseStage, parseErr, repaired)
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q returned invalid collect artifact repair result after %s: %s", acpruntime.ProviderClaudeCode, initialProblem, parseFailureMessage),
			repaired.Stdout,
			repaired.Stderr,
			parseErr,
		)
	}

	ensureErr = artifactquality.EnsureCanonicalCollectManifest(task, repaired.TaskResult)
	repairedAssessment, err := artifactquality.ValidateCollectManifestAtWriteRoot(task.WriteRoot)
	if ensureErr != nil {
		err = ensureErr
	}
	if err != nil || !repairedAssessment.Rich {
		_ = snapshot.Restore()
		repairedProblem := artifactquality.DescribeAssessmentProblem(repairedAssessment, err)
		return acpruntime.Result{}, wrapArtifactContractFailure(
			repaired.Stdout,
			repaired.Stderr,
			fmt.Sprintf("collect artifacts remained invalid after one repair attempt: initial=%s; repaired=%s", initialProblem, repairedProblem),
			err,
		)
	}
	return repaired, nil
}

func wrapArtifactContractFailure(stdout string, stderr string, message string, cause error) error {
	return acpruntime.WrapRunnerErrorWithOutput(
		acpruntime.ProviderClaudeCode,
		acpruntime.ErrorCodeRunnerParseFailed,
		fmt.Sprintf("headless provider %q produced invalid collect artifacts: %s", acpruntime.ProviderClaudeCode, strings.TrimSpace(message)),
		stdout,
		stderr,
		cause,
	)
}

func isEnvelopeResultEmptyError(err error) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "envelope result is empty")
}

func isEnvelopeResultMalformedError(err error) bool {
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	if !strings.Contains(text, "envelope key \"result\"") {
		return false
	}
	return strings.Contains(text, "string candidate parse failed") ||
		strings.Contains(text, "invalid character") ||
		strings.Contains(text, "unexpected end of json input")
}

func buildParseFailureMessage(task acpruntime.Task, parseStage string, parseErr error, result acpruntime.Result) string {
	base := strings.TrimSpace(parseErr.Error())
	if base == "" {
		base = "unknown parse error"
	}
	stage := strings.TrimSpace(parseStage)
	if stage == "" {
		stage = "unknown"
	}
	artifacts, err := runnerdiag.WriteParseFailureArtifacts(task, acpruntime.ProviderClaudeCode, result.Stdout, result.Stderr)
	if err != nil {
		return fmt.Sprintf("parse_stage=%s %s (raw_output_persist_failed=%v)", stage, base, err)
	}
	return fmt.Sprintf(
		"parse_stage=%s %s (raw_output=%s stdout_bytes=%d stdout_sha256=%s stderr_bytes=%d stderr_sha256=%s)",
		stage,
		base,
		artifacts.RelativeMetadataPath,
		artifacts.Stdout.Bytes,
		artifacts.Stdout.SHA256,
		artifacts.Stderr.Bytes,
		artifacts.Stderr.SHA256,
	)
}

func buildUnavailableFailureMessage(task acpruntime.Task, runErr error, result acpruntime.Result) string {
	base := strings.TrimSpace(runErr.Error())
	if base == "" {
		base = "unknown execution error"
	}
	artifacts, err := runnerdiag.WriteParseFailureArtifacts(task, acpruntime.ProviderClaudeCode, result.Stdout, result.Stderr)
	if err != nil {
		return fmt.Sprintf("parse_stage=exec %s (raw_output_persist_failed=%v)", base, err)
	}
	return fmt.Sprintf(
		"parse_stage=exec %s (raw_output=%s stdout_bytes=%d stdout_sha256=%s stderr_bytes=%d stderr_sha256=%s)",
		base,
		artifacts.RelativeMetadataPath,
		artifacts.Stdout.Bytes,
		artifacts.Stdout.SHA256,
		artifacts.Stderr.Bytes,
		artifacts.Stderr.SHA256,
	)
}

func idleWatchdogTimeout(task acpruntime.Task) time.Duration {
	if strings.TrimSpace(task.StepID) == "init.step3.findings" || strings.TrimSpace(task.StepID) == "refresh.step3.findings" {
		return findingsIdleSilenceTimeout
	}
	return 0
}

func isRunnerStalledError(err error) bool {
	return errors.Is(err, errRunnerStalled)
}

func runClaudeCommand(ctx context.Context, task acpruntime.Task, command string, args []string, stdin []byte) (acpruntime.Result, string, error, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	if writeRoot := strings.TrimSpace(task.WriteRoot); writeRoot != "" {
		cmd.Dir = writeRoot
	} else if workspace := strings.TrimSpace(task.Workspace); workspace != "" {
		cmd.Dir = workspace
	}
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var stalled atomic.Bool
	lastOutputAtUnixNano := atomic.Int64{}
	lastOutputAtUnixNano.Store(time.Now().UnixNano())
	notifyOutput := func() {
		lastOutputAtUnixNano.Store(time.Now().UnixNano())
	}
	idleTimeout := idleWatchdogTimeout(task)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return acpruntime.Result{}, "", nil, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return acpruntime.Result{}, "", nil, err
	}

	if err := cmd.Start(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return acpruntime.Result{}, "", nil, ctxErr
		}
		return acpruntime.Result{}, "", nil, err
	}

	watchdogDone := make(chan struct{})
	if idleTimeout > 0 {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-watchdogDone:
					return
				case <-ticker.C:
					last := time.Unix(0, lastOutputAtUnixNano.Load())
					if time.Since(last) < idleTimeout {
						continue
					}
					stalled.Store(true)
					_ = cmd.Process.Kill()
					return
				}
			}
		}()
	}
	defer close(watchdogDone)

	var streamErr error
	var streamErrMu sync.Mutex
	captureErr := func(captureErr error) {
		if captureErr == nil {
			return
		}
		streamErrMu.Lock()
		defer streamErrMu.Unlock()
		if streamErr == nil {
			streamErr = captureErr
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		captureErr(captureCommandStream(stdoutPipe, &stdout, task, acpruntime.OutputStreamStdout, notifyOutput))
	}()
	go func() {
		defer wg.Done()
		captureErr(captureCommandStream(stderrPipe, &stderr, task, acpruntime.OutputStreamStderr, notifyOutput))
	}()

	// Drain both output streams before waiting to avoid racy early pipe closes
	// that can truncate stdout/stderr under parallel test/process scheduling.
	wg.Wait()
	waitErr := cmd.Wait()
	if waitErr == nil && streamErr != nil {
		waitErr = streamErr
	}
	if waitErr != nil {
		if stalled.Load() {
			stallErr := fmt.Errorf(
				"%w: no stdout/stderr output for %s (task=%s step=%s shard=%s)",
				errRunnerStalled,
				idleTimeout,
				strings.TrimSpace(task.TaskID),
				strings.TrimSpace(task.StepID),
				strings.TrimSpace(task.ShardID),
			)
			return acpruntime.Result{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
			}, "", nil, stallErr
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return acpruntime.Result{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
			}, "", nil, ctxErr
		}
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, "", nil, runnerdiag.BuildExecFailure(waitErr, stdout.String(), stderr.String())
	}

	raw, err := taskresultextractor.Extract(stdout.Bytes())
	if err != nil {
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, "extract", err, nil
	}
	if normalizedRaw, changed, normalizeErr := taskresultcompat.NormalizeRawTaskResult(task, raw); normalizeErr == nil && changed {
		raw = normalizedRaw
	}
	taskResult, err := contracts.ParseTaskResult(raw)
	if err != nil {
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, "schema", err, nil
	}
	if err := taskresultbinding.Validate(task, taskResult, acpruntime.ProviderClaudeCode); err != nil {
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, "binding", err, nil
	}

	return acpruntime.Result{
		TaskResult: taskResult,
		RawJSON:    raw,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}, "", nil, nil
}

type streamedOutputBudget struct {
	forwardedBytes int
	truncated      bool
}

func captureCommandStream(
	reader io.Reader,
	sink *bytes.Buffer,
	task acpruntime.Task,
	stream acpruntime.OutputStream,
	onOutput func(),
) error {
	if sink == nil {
		return errors.New("capture sink is nil")
	}
	bufReader := bufio.NewReader(reader)
	budget := &streamedOutputBudget{}
	for {
		part, err := bufReader.ReadString('\n')
		if len(part) > 0 {
			sink.WriteString(part)
			if onOutput != nil {
				onOutput()
			}
			forwardStreamOutput(task, stream, part, budget)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if isPipeClosedErr(err) {
				return nil
			}
			return err
		}
	}
}

func isPipeClosedErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrClosed) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "file already closed")
}

func forwardStreamOutput(task acpruntime.Task, stream acpruntime.OutputStream, chunk string, budget *streamedOutputBudget) {
	if task.OnOutput == nil || budget == nil {
		return
	}
	normalized := strings.ReplaceAll(chunk, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if budget.truncated {
			continue
		}
		lineBytes := len([]byte(line))
		nextBytes := budget.forwardedBytes + lineBytes
		if nextBytes <= acpruntime.RuntimeOutputStreamHardCapBytes {
			budget.forwardedBytes = nextBytes
			task.OnOutput(acpruntime.OutputChunk{
				Stream: stream,
				Text:   line,
			})
			continue
		}
		remaining := acpruntime.RuntimeOutputStreamHardCapBytes - budget.forwardedBytes
		if remaining > 0 {
			trimmed := line
			if len([]byte(trimmed)) > remaining {
				trimmedBytes := []byte(trimmed)
				if remaining < len(trimmedBytes) {
					trimmed = string(trimmedBytes[:remaining])
				}
			}
			trimmed = strings.TrimSpace(trimmed)
			if trimmed != "" {
				task.OnOutput(acpruntime.OutputChunk{
					Stream: stream,
					Text:   trimmed,
				})
			}
		}
		budget.truncated = true
		task.OnOutput(acpruntime.OutputChunk{
			Stream:    stream,
			Truncated: true,
			Text:      fmt.Sprintf("%s output truncated after %d bytes (internal safeguard)", stream, acpruntime.RuntimeOutputStreamHardCapBytes),
		})
	}
}

func buildDirectPrompt(taskPayload []byte, retry bool, requireNonEmptyResult bool) string {
	mode := promptRetryNone
	if retry {
		mode = promptRetryParse
	}
	return buildDirectPromptArtifact(taskPayload, mode, requireNonEmptyResult, nil).Text
}

func buildDirectPromptWithMode(taskPayload []byte, mode promptRetryMode, requireNonEmptyResult bool) string {
	return buildDirectPromptArtifact(taskPayload, mode, requireNonEmptyResult, nil).Text
}

func buildDirectPromptWithModeAndHints(taskPayload []byte, mode promptRetryMode, requireNonEmptyResult bool, extraHints []string) string {
	return buildDirectPromptArtifact(taskPayload, mode, requireNonEmptyResult, extraHints).Text
}

func buildDirectPromptArtifact(taskPayload []byte, mode promptRetryMode, requireNonEmptyResult bool, extraHints []string) builtPrompt {
	var task acpruntime.Task
	if err := json.Unmarshal(taskPayload, &task); err != nil {
		return builtPrompt{Text: strings.TrimSpace(fmt.Sprintf(`You are ACP runtime provider %q.
Return exactly one valid JSON object for ACP TaskResult.
Do not output markdown, code fences, or any explanatory text.
Task payload JSON:
%s`, acpruntime.ProviderClaudeCode, strings.TrimSpace(string(taskPayload))))}
	}

	repoScopesJSON := "[]"
	if rawRepoScopes, err := json.Marshal(task.RepoScopes); err == nil {
		repoScopesJSON = string(rawRepoScopes)
	}
	primaryRepoScope := primaryTaskRepoScope(task.RepoScope, task.RepoScopes)
	pathScopesJSON := "[]"
	if rawPathScopes, err := json.Marshal(task.PathScopes); err == nil {
		pathScopesJSON = string(rawPathScopes)
	}
	promptPackSection, pack := promptcontract.AdditivePromptPackSection(task)
	stepPolicy := promptcontract.StepPolicy(task.StepID)
	repositoryEvidencePolicy := promptcontract.RepositoryEvidencePolicy()
	docFirstPolicy := promptcontract.DocFirstFilesystemPolicy(task)
	strictContractLines := strings.Join(promptcontract.SharedTaskResultContractLines(), "\n")
	retryHint := ""
	if mode != promptRetryNone {
		retryLines := []string{
			`RETRY MODE: previous output was invalid JSON.`,
			`Do not include non-ASCII symbols in numbers or timestamps.`,
			`RFC3339 timestamps only (example: 2026-04-09T15:28:49Z).`,
			`Decimals must be compact numeric literals (example: 0.7, not 0. 7).`,
		}
		retryLines = append(retryLines, promptcontract.SharedRetryGuardrailLines()...)
		retryLines = append(retryLines, `Return only JSON object, without prose.`)
		if mode == promptRetryArtifact {
			retryLines[0] = `ARTIFACT REPAIR MODE: previous collect output was schema-valid but write_root artifacts look skeletal or generic-only.`
			retryLines = append(retryLines,
				`Repair artifact fidelity before returning JSON; this retry is not a fresh repository rediscovery pass.`,
				`Keep repo roots available while restoring repo-specific citations in shard-pack-manifest.json.`,
				`Rewrite shard-pack-manifest.json to the canonical ACP schema (version=1 integer, documents[].citation_ids, citations[].id/document_ids, stable canonical_path).`,
				`compatibility.coverage/questions/entities/edges/findings must all exist, and questions/entities/edges/findings must be arrays rather than booleans or null.`,
			)
		}
		retryLines = append(retryLines, extraHints...)
		retryHint = strings.Join(retryLines, "\n")
	}
	nonEmptyResultHint := ""
	if requireNonEmptyResult {
		nonEmptyResultHint = strings.Join([]string{
			`STRICT RESULT JSON MODE:`,
			`- If using envelope fields like "result", value MUST be a non-empty valid JSON object string.`,
			`- Do NOT emit empty or malformed "result" payload.`,
			`- Prefer returning a direct TaskResult JSON object (without envelope wrappers).`,
		}, "\n")
	}

	return builtPrompt{
		Text: strings.TrimSpace(fmt.Sprintf(`You are ACP runtime provider %q.
Return exactly one valid JSON object for ACP TaskResult.
Do not output markdown, code fences, explanations, or any text outside the JSON object.

%s

STRICT CONTRACT (must pass):
%s
%s
%s
%s
%s
%s

Set meta fields exactly:
- meta.task_id = %q
- meta.step_id = %q
- meta.run_id = %q
- meta.runtime.name = %q
- meta.runtime.version = %q
- meta.started_at = %q
- meta.workspace = %q
- meta.shard_id = %q
- meta.repo_scope = %q
- meta.repo_scopes = %s
- meta.path_scopes = %s

Schema-valid template for this task (copy structure and field TYPES, then refine values with available evidence):
%s

Serialized runtime task JSON (context only):
%s`, acpruntime.ProviderClaudeCode, promptPackSection, strictContractLines, stepPolicy, repositoryEvidencePolicy, docFirstPolicy, retryHint, nonEmptyResultHint, task.TaskID, task.StepID, task.RunID, acpruntime.ProviderClaudeCode, "claude-cli", task.StartedAtUTC.UTC().Format(time.RFC3339), task.Workspace, task.ShardID, primaryRepoScope, repoScopesJSON, pathScopesJSON, buildDirectTaskResultTemplateJSON(task), strings.TrimSpace(string(taskPayload)))),
		Pack: pack,
	}
}

func buildDirectTaskResultTemplateJSON(task acpruntime.Task) string {
	coverageMissing := []string{"owner mappings", "ci-cd evidence"}
	questions := []contracts.Question{}
	if strings.HasPrefix(task.StepID, "refresh.") {
		coverageMissing = append(coverageMissing, "delta validation")
		questions = []contracts.Question{
			{
				ID:       "q.refresh.delta",
				Text:     "What changed since previous run that affects ownership or dependencies?",
				Priority: "high",
			},
		}
	}

	template := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:     task.TaskID,
			StepID:     task.StepID,
			RunID:      task.RunID,
			Runtime:    contracts.RuntimeMeta{Name: string(acpruntime.ProviderClaudeCode), Version: "claude-cli"},
			StartedAt:  task.StartedAtUTC.UTC().Format(time.RFC3339),
			FinishedAt: task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
			Workspace:  task.Workspace,
			ShardID:    task.ShardID,
			RepoScope:  primaryTaskRepoScope(task.RepoScope, task.RepoScopes),
			RepoScopes: append([]string(nil), task.RepoScopes...),
			PathScopes: append([]string(nil), task.PathScopes...),
		},
		Summary:   "Task completed with contract-compliant output.",
		Changeset: buildDirectTemplateChangeset(task),
		Coverage: &contracts.Coverage{
			Observed: []string{"services"},
			Missing:  coverageMissing,
			Notes:    []string{"evidence gaps are captured explicitly"},
		},
		Questions: questions,
		Warnings:  []string{},
	}
	raw, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func buildParseRepairHints(parseStage string, parseErr error) []string {
	return promptcontract.ParseRepairHints(parseStage, parseErr)
}

func buildStallRetryHints(stallErr error) []string {
	lines := []string{
		`Previous attempt stalled due to long stdout/stderr silence; respond quickly with one final TaskResult JSON object.`,
		`Skip exploratory chatter and long tool narration; only minimal reads needed for deterministic completion are allowed.`,
		`Do NOT return event arrays, transcript envelopes, or tool recap prose.`,
	}
	if detail := compactRetryHint(errorString(stallErr)); detail != "" {
		lines = append(lines, "Stall detail: "+detail)
	}
	return lines
}

func buildArtifactRepairHints(initialProblem string) []string {
	return promptcontract.ArtifactRepairHints(initialProblem)
}

func compactRetryHint(value string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(normalized) <= 320 {
		return normalized
	}
	return normalized[:317] + "..."
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func buildDirectTemplateChangeset(task acpruntime.Task) []contracts.Operation {
	scopes := append([]string(nil), task.RepoScopes...)
	if len(scopes) == 0 {
		scopes = []string{"repository"}
	}
	changes := make([]contracts.Operation, 0, len(scopes))
	switch task.StepID {
	case "init.step3.findings", "refresh.step3.findings":
		for _, scope := range scopes {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				scope = "repository"
			}
			slug := slugutil.Slugify(scope)
			if slug == "" {
				slug = "repository"
			}
			changes = append(changes, contracts.Operation{
				Op: "add_finding",
				Finding: &contracts.Finding{
					ID:          "finding.missing-owner.svc." + slug,
					Severity:    "medium",
					Title:       "Missing owner mapping",
					Description: "owner_team_id is not confirmed",
					RuleID:      "rule.owner.required",
					RelatedIDs:  []string{"svc." + slug},
					Provenance: contracts.Provenance{
						Kind:       "inference",
						Confidence: 0.66,
						Evidence: []contracts.Evidence{
							{Repo: scope, Path: "service.yaml"},
						},
					},
				},
			})
		}
	default:
		for _, scope := range scopes {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				scope = "repository"
			}
			slug := slugutil.Slugify(scope)
			if slug == "" {
				slug = "repository"
			}
			changes = append(changes, contracts.Operation{
				Op: "upsert_entity",
				Entity: &contracts.Entity{
					ID:   "svc." + slug,
					Type: "service",
					Name: humanizeServiceName(scope),
					Attributes: map[string]any{
						"repo_scope": scope,
						"runtime":    acpruntime.ProviderClaudeCode,
					},
					Provenance: contracts.Provenance{
						Kind:       "observation",
						Confidence: 0.7,
						Evidence: []contracts.Evidence{
							{
								Repo: scope,
								Path: "README.md",
							},
						},
					},
				},
			})
		}
	}
	return changes
}

type FakeRunner struct{}

func (FakeRunner) Run(_ context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	repoScopes := append([]string(nil), task.RepoScopes...)
	sort.Strings(repoScopes)

	switch task.StepID {
	case "init.step1.collect", "refresh.step1.collect":
		result := contracts.TaskResult{
			Meta: contracts.Meta{
				TaskID:     task.TaskID,
				StepID:     task.StepID,
				RunID:      task.RunID,
				Runtime:    contracts.RuntimeMeta{Name: "claude-code", Version: "fake"},
				StartedAt:  task.StartedAtUTC.UTC().Format(time.RFC3339),
				FinishedAt: task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
				Workspace:  task.Workspace,
				ShardID:    task.ShardID,
				RepoScope:  primaryTaskRepoScope(task.RepoScope, repoScopes),
				RepoScopes: repoScopes,
				PathScopes: append([]string(nil), task.PathScopes...),
			},
			Summary:   "Fake collect context completed",
			Changeset: makeCollectChangeset(repoScopes),
			Questions: makeCollectQuestions(repoScopes),
			Coverage: &contracts.Coverage{
				Observed: []string{"services", "entrypoints"},
				Missing:  []string{"owner mappings", "ci-cd evidence"},
				Notes:    []string{"fake runner materialized deterministic baseline output"},
			},
		}
		if err := persistDocsFirstArtifacts(task, result); err != nil {
			return acpruntime.Result{}, err
		}
		return marshalResult(result)
	case "init.step3.findings", "refresh.step3.findings":
		result := contracts.TaskResult{
			Meta: contracts.Meta{
				TaskID:     task.TaskID,
				StepID:     task.StepID,
				RunID:      task.RunID,
				Runtime:    contracts.RuntimeMeta{Name: "claude-code", Version: "fake"},
				StartedAt:  task.StartedAtUTC.UTC().Format(time.RFC3339),
				FinishedAt: task.StartedAtUTC.UTC().Add(1 * time.Second).Format(time.RFC3339),
				Workspace:  task.Workspace,
				ShardID:    task.ShardID,
				RepoScope:  primaryTaskRepoScope(task.RepoScope, repoScopes),
				RepoScopes: repoScopes,
				PathScopes: append([]string(nil), task.PathScopes...),
			},
			Summary:   "Fake findings completed",
			Changeset: makeFindingsChangeset(repoScopes),
		}
		if err := persistDocsFirstArtifacts(task, result); err != nil {
			return acpruntime.Result{}, err
		}
		return marshalResult(result)
	default:
		return acpruntime.Result{}, fmt.Errorf("fake runner does not support step %q", task.StepID)
	}
}

type RecordedRunner struct {
	ByStep map[string]string
}

func (r RecordedRunner) Run(_ context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	path, ok := r.ByStep[task.StepID]
	if !ok {
		return acpruntime.Result{}, fmt.Errorf("recorded taskresult is missing for step %q", task.StepID)
	}
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return acpruntime.Result{}, fmt.Errorf("read recorded taskresult: %w", err)
	}
	taskResult, err := contracts.ParseTaskResult(content)
	if err != nil {
		return acpruntime.Result{}, fmt.Errorf("parse recorded taskresult: %w", err)
	}
	if err := persistDocsFirstArtifacts(task, taskResult); err != nil {
		return acpruntime.Result{}, err
	}
	return acpruntime.Result{
		TaskResult: taskResult,
		RawJSON:    bytes.TrimSpace(content),
	}, nil
}

func marshalResult(result contracts.TaskResult) (acpruntime.Result, error) {
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return acpruntime.Result{}, fmt.Errorf("marshal fake taskresult: %w", err)
	}
	return acpruntime.Result{
		TaskResult: result,
		RawJSON:    raw,
	}, nil
}

func makeCollectChangeset(repoScopes []string) []contracts.Operation {
	changes := make([]contracts.Operation, 0, len(repoScopes))
	for _, repo := range repoScopes {
		slug := slugutil.Slugify(repo)
		changes = append(changes, contracts.Operation{
			Op: "upsert_entity",
			Entity: &contracts.Entity{
				ID:   "svc." + slug,
				Type: "service",
				Name: humanizeServiceName(repo),
				Attributes: map[string]any{
					"repo_scope": repo,
					"runtime":    "claude-code",
				},
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.7,
					Evidence: []contracts.Evidence{
						{
							Repo: repo,
							Path: "README.md",
						},
					},
				},
			},
		})
	}
	return changes
}

func makeCollectQuestions(repoScopes []string) []contracts.Question {
	questions := make([]contracts.Question, 0, len(repoScopes))
	for _, repo := range repoScopes {
		slug := slugutil.Slugify(repo)
		questions = append(questions, contracts.Question{
			ID:       "q.owner.svc." + slug,
			Text:     fmt.Sprintf("Who owns service derived from repo %q?", repo),
			Priority: "high",
			RelatedIDs: []string{
				"svc." + slug,
			},
		})
	}
	return questions
}

func makeFindingsChangeset(repoScopes []string) []contracts.Operation {
	findings := make([]contracts.Operation, 0, len(repoScopes))
	for _, repo := range repoScopes {
		slug := slugutil.Slugify(repo)
		findings = append(findings, contracts.Operation{
			Op: "add_finding",
			Finding: &contracts.Finding{
				ID:          "finding.missing-owner.svc." + slug,
				Severity:    "medium",
				Title:       "Missing owner mapping",
				Description: fmt.Sprintf("owner_team_id is unknown for service derived from repo %q", repo),
				RuleID:      "rule.owner.required",
				RelatedIDs: []string{
					"svc." + slug,
				},
				Provenance: contracts.Provenance{
					Kind:       "inference",
					Confidence: 0.66,
				},
			},
		})
	}
	return findings
}

func humanizeServiceName(repo string) string {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return "Unknown Service"
	}
	parts := strings.FieldsFunc(repo, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	name := strings.Join(parts, " ")
	if strings.HasSuffix(strings.ToLower(name), " service") {
		return name
	}
	return name + " Service"
}

func primaryTaskRepoScope(explicit string, scopes []string) string {
	if value := strings.TrimSpace(explicit); value != "" {
		return value
	}
	for _, scope := range scopes {
		if value := strings.TrimSpace(scope); value != "" {
			return value
		}
	}
	return ""
}

func recordPromptArtifacts(task acpruntime.Task, attempt string, prompt builtPrompt, provider acpruntime.Provider, includeDirectories []string, taskPayload []byte) {
	if warning := strings.TrimSpace(prompt.Pack.Warning); warning != "" && task.OnOutput != nil {
		task.OnOutput(acpruntime.OutputChunk{
			Stream: acpruntime.OutputStreamStderr,
			Text:   "prompt_pack_warning: " + warning,
		})
	}
	_, err := runnerdiag.WritePromptArtifacts(task, provider, prompt.Text, taskPayload, runnerdiag.PromptArtifactsMetadata{
		Attempt:            attempt,
		IncludeDirectories: append([]string(nil), includeDirectories...),
		PromptPack: runnerdiag.PromptPackMetadata{
			Name:         prompt.Pack.Name,
			RelativePath: prompt.Pack.RelativePath,
			Source:       prompt.Pack.Source,
			Warning:      prompt.Pack.Warning,
		},
	})
	if err != nil && task.OnOutput != nil {
		task.OnOutput(acpruntime.OutputChunk{
			Stream: acpruntime.OutputStreamStderr,
			Text:   fmt.Sprintf("prompt_artifact_warning: %v", err),
		})
	}
}
