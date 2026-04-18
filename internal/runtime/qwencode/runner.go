package qwencode

import (
	"bufio"
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
	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultbinding"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultextractor"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

var (
	ErrRunnerUnavailable = errors.New("qwen-code runner is unavailable")
)

type HeadlessRunner struct {
	Command string
	Args    []string
}

type retryManifestAssessment = artifactquality.ManifestAssessment

type promptRetryMode int

const (
	promptRetryNone promptRetryMode = iota
	promptRetryParse
	promptRetryArtifact
)

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

	result, parseStage, parseErr, runErr := runQwenCommand(ctx, task, command, args)
	if runErr != nil {
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
		if len(r.Args) == 0 {
			return maybeRepairCollectArtifacts(ctx, task, taskPayload, command, result)
		}
		return result, nil
	}
	finalStdout := result.Stdout
	finalStderr := result.Stderr

	// Live qwen output can occasionally contain malformed tokens. Retry once with
	// an explicitly stricter prompt before classifying as parse failure.
	if len(r.Args) == 0 {
		retryArgs := buildRetryQwenArgs(task, buildPrompt(taskPayload, true))
		retryResult, retryParseStage, retryParseErr, retryRunErr := runQwenCommand(ctx, task, command, retryArgs)
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
			return maybeRepairCollectArtifacts(ctx, task, taskPayload, command, retryResult)
		}
		result = retryResult
		parseStage = retryParseStage
		parseErr = retryParseErr
		finalStdout = retryResult.Stdout
		finalStderr = retryResult.Stderr
	}

	parseFailureMessage := buildParseFailureMessage(task, parseStage, parseErr, result)
	return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
		acpruntime.ProviderQwenCode,
		acpruntime.ErrorCodeRunnerParseFailed,
		fmt.Sprintf("headless provider %q returned invalid taskresult: %s", acpruntime.ProviderQwenCode, parseFailureMessage),
		finalStdout,
		finalStderr,
		parseErr,
	)
}

func buildDefaultQwenArgs(task acpruntime.Task, prompt string) []string {
	args := []string{"--output-format", "json", "--chat-recording", "false", "--yolo", "--channel", "CI"}
	for _, dir := range acpruntime.ResolveHeadlessIncludeDirectories(task) {
		args = append(args, "--include-directories", dir)
	}
	args = append(args, "--prompt", prompt)
	return args
}

func buildRetryQwenArgs(task acpruntime.Task, prompt string) []string {
	args := []string{"--output-format", "json", "--chat-recording", "false", "--yolo", "--channel", "CI"}
	for _, dir := range resolveRetryIncludeDirectories(task) {
		args = append(args, "--include-directories", dir)
	}
	args = append(args, "--prompt", prompt)
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

	assessment, err := assessRetryManifestAtWriteRoot(task.WriteRoot)
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

	repairArgs := buildRetryQwenArgs(task, buildPromptWithMode(taskPayload, promptRetryArtifact))
	repaired, repairParseStage, parseErr, runErr := runQwenCommand(ctx, task, command, repairArgs)
	if runErr != nil {
		_ = snapshot.Restore()
		unavailableMessage := buildUnavailableFailureMessage(task, runErr, repaired)
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
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
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q returned invalid collect artifact repair result after %s: %s", acpruntime.ProviderQwenCode, initialProblem, parseFailureMessage),
			repaired.Stdout,
			repaired.Stderr,
			parseErr,
		)
	}

	repairedAssessment, err := assessRetryManifestAtWriteRoot(task.WriteRoot)
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
		acpruntime.ProviderQwenCode,
		acpruntime.ErrorCodeRunnerParseFailed,
		fmt.Sprintf("headless provider %q produced invalid collect artifacts: %s", acpruntime.ProviderQwenCode, strings.TrimSpace(message)),
		stdout,
		stderr,
		cause,
	)
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

func buildParseFailureMessage(task acpruntime.Task, parseStage string, parseErr error, result acpruntime.Result) string {
	base := strings.TrimSpace(parseErr.Error())
	if base == "" {
		base = "unknown parse error"
	}
	stage := strings.TrimSpace(parseStage)
	if stage == "" {
		stage = "unknown"
	}
	artifacts, err := runnerdiag.WriteParseFailureArtifacts(task, acpruntime.ProviderQwenCode, result.Stdout, result.Stderr)
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
	artifacts, err := runnerdiag.WriteParseFailureArtifacts(task, acpruntime.ProviderQwenCode, result.Stdout, result.Stderr)
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

func runQwenCommand(ctx context.Context, task acpruntime.Task, command string, args []string) (acpruntime.Result, string, error, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	// Keep qwen's project root anchored at the ACP workspace rather than a
	// deep shard staging directory. This avoids provider-local path explosions
	// (for example chat/debug history paths derived from cwd) while preserving
	// explicit write_root instructions inside the prompt.
	if workspace := strings.TrimSpace(task.Workspace); workspace != "" {
		cmd.Dir = workspace
	} else if writeRoot := strings.TrimSpace(task.WriteRoot); writeRoot != "" {
		cmd.Dir = writeRoot
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
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
		captureErr(captureCommandStream(stdoutPipe, &stdout, task, acpruntime.OutputStreamStdout))
	}()
	go func() {
		defer wg.Done()
		captureErr(captureCommandStream(stderrPipe, &stderr, task, acpruntime.OutputStreamStderr))
	}()

	// Drain both output streams before waiting to avoid racy early pipe closes
	// that can truncate stdout/stderr under parallel test/process scheduling.
	wg.Wait()
	waitErr := cmd.Wait()
	if waitErr == nil && streamErr != nil {
		waitErr = streamErr
	}
	if waitErr != nil {
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

	rawTaskResult, err := taskresultextractor.Extract(stdout.Bytes())
	if err != nil {
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, "extract", err, nil
	}
	taskResult, err := contracts.ParseTaskResult(rawTaskResult)
	if err != nil {
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, "schema", err, nil
	}
	if err := taskresultbinding.Validate(task, taskResult, acpruntime.ProviderQwenCode); err != nil {
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, "binding", err, nil
	}
	return acpruntime.Result{
		TaskResult: taskResult,
		RawJSON:    rawTaskResult,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}, "", nil, nil
}

type streamedOutputBudget struct {
	forwardedBytes int
	truncated      bool
}

func captureCommandStream(reader io.Reader, sink *bytes.Buffer, task acpruntime.Task, stream acpruntime.OutputStream) error {
	if sink == nil {
		return errors.New("capture sink is nil")
	}
	bufReader := bufio.NewReader(reader)
	budget := &streamedOutputBudget{}
	for {
		part, err := bufReader.ReadString('\n')
		if len(part) > 0 {
			sink.WriteString(part)
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

func buildPrompt(taskPayload []byte, retry bool) string {
	mode := promptRetryNone
	if retry {
		mode = promptRetryParse
	}
	return buildPromptWithMode(taskPayload, mode)
}

func buildPromptWithMode(taskPayload []byte, mode promptRetryMode) string {
	var task acpruntime.Task
	if err := json.Unmarshal(taskPayload, &task); err != nil {
		return strings.TrimSpace(fmt.Sprintf(`You are ACP runtime provider %q.
Return exactly one valid JSON object for ACP TaskResult.
Do not output markdown, code fences, or any explanatory text.
Task payload JSON:
%s`, acpruntime.ProviderQwenCode, strings.TrimSpace(string(taskPayload))))
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
	stepPolicy := buildStepSpecificPolicy(task.StepID)
	repositoryEvidencePolicy := strings.Join([]string{
		`REPOSITORY EVIDENCE RULES:`,
		`- ACP workspace scaffold (workspace.yaml, charter/, model/, reports/) is support context, not the primary source tree.`,
		`- Prefer evidence from repository files under meta.repo_scopes/meta.path_scopes when those files are available.`,
		`- meta.path_scopes may contain directories, files, or a mixed disjoint partition; treat every listed scope as in-bounds evidence for this task.`,
		`- Use ACP-generated workspace artifacts as evidence only for ACP runtime/report state, not as a substitute for repository analysis.`,
	}, "\n")
	docFirstPolicy := buildDocFirstFilesystemPolicy(task)
	strictResultHint := strings.Join([]string{
		`STRICT RESULT JSON MODE:`,
		`- Prefer returning a direct TaskResult JSON object (without envelope wrappers).`,
		`- If using envelope fields like "result", value MUST be a non-empty valid JSON object string.`,
		`- Do NOT emit empty or malformed "result" payload.`,
		`- Do NOT draft, preview, or explain the TaskResult before returning it.`,
		`- Do NOT echo template fragments, markdown examples, or partial JSON.`,
	}, "\n")
	finalResponseDiscipline := strings.Join([]string{
		`FINAL RESPONSE DISCIPLINE:`,
		`- Use tool calls for filesystem reads/writes when needed, but the final assistant message MUST be only the TaskResult JSON object.`,
		`- Do NOT narrate file writes, manifest contents, or planning steps in the final message.`,
		`- After the last tool call, respond immediately with the final JSON object.`,
	}, "\n")
	retryHint := ""
	retryTemplate := ""
	if mode != promptRetryNone {
		retryLines := []string{
			`RETRY MODE: previous output needs one deterministic repair pass.`,
			`Do not include non-ASCII symbols in numbers or timestamps.`,
			`RFC3339 timestamps only (example: 2026-04-09T15:28:49Z).`,
			`Decimals must be compact numeric literals (example: 0.7, not 0. 7).`,
			`COMPACT JSON MODE: keep output concise and deterministic.`,
			`- If envelope form is unavoidable, "result" MUST be a non-empty valid JSON object string.`,
			`- Limit changeset to the minimum actionable operations for this step.`,
			`- Prefer "changeset": [] when write_root already contains authored docs and the step does not strictly require an operation.`,
			`- If changeset is non-empty, use at most 1 operation and at most 3 provenance.evidence items.`,
			`- Keep coverage compact: observed <=2 items, missing <=2 canonical items, notes <=1 short entry.`,
			`- Keep coverage.notes short (<=2 entries).`,
			`- Avoid long prose in summary; keep a single sentence.`,
			`- Do NOT copy long citations, topic arrays, or manifest document inventories into TaskResult JSON.`,
			`- Do NOT collapse a multi-document refresh into one generic "cite.runtime-summary" citation.`,
			`- Do NOT overwrite a rich shard-pack-manifest.json with a skeletal reuse-only manifest.`,
			`- Preserve repo-specific citations when repository evidence already exists or can be recovered from repo roots.`,
			`- Final response MUST start with "{" and end with "}".`,
			`- Do not output markdown fences, bullet lists, plan text, or template walkthroughs.`,
		}
		switch mode {
		case promptRetryParse:
			retryLines[0] = `RETRY MODE: previous output was invalid JSON.`
		case promptRetryArtifact:
			retryLines[0] = `ARTIFACT REPAIR MODE: previous collect output was schema-valid but write_root artifacts look skeletal or generic-only.`
			retryLines = append(retryLines,
				`- Repair artifact fidelity before returning JSON; this retry is not a fresh repository rediscovery pass.`,
				`- Keep repo roots available when write_root artifacts lack repo-specific citations or collapse to generic summaries.`,
			)
		}
		retryLines = append(retryLines, buildRetryRecoveryHint(task))
		retryHint = strings.Join(retryLines, "\n")
		retryTemplate = "\nRetry-safe minimal template (preferred when reusing existing write_root artifacts):\n" + buildRetryMinimalTaskResultTemplateJSON(task)
	}

	return strings.TrimSpace(fmt.Sprintf(`You are ACP runtime provider %q.
Return exactly one valid JSON object for ACP TaskResult.
Do not output markdown, code fences, explanations, or any text outside the JSON object.

STRICT CONTRACT (must pass):
- top-level required keys: "meta", "summary", "changeset"
- meta required keys: "task_id", "step_id", "runtime", "started_at"
- meta.runtime required key: "name"
- use snake_case keys exactly as shown.
- DO NOT use top-level fields: task_id, run_id, step_id, status.
- provenance.kind MUST be one of: observation, inference, assertion.
- provenance.confidence MUST be a NUMBER in range [0,1], never a string.
- provenance.evidence MUST be an ARRAY of objects with repo/path.
- if "questions" is present, it MUST be an array of objects (each object has at least "id" and "text").
- coverage.missing MUST use canonical terms only: owner mappings, ci-cd evidence, delta validation, dependency graph, runtime metrics, api contracts, deployment configs, integration edges, datastore bindings, dependencies.
- question IDs MUST use canonical form without numeric suffixes (example: q.refresh.delta, not q.refresh.delta.1).
- Do not claim workspace is empty/minimal unless provenance evidence includes concrete file paths proving it.
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
- meta.started_at = %q
- meta.workspace = %q
- meta.shard_id = %q
- meta.repo_scope = %q
- meta.repo_scopes = %s
- meta.path_scopes = %s

Schema-valid template for this task (copy structure and field TYPES, then refine values with available evidence):
%s
%s

Serialized runtime task JSON (context only):
%s`, acpruntime.ProviderQwenCode, stepPolicy, repositoryEvidencePolicy, docFirstPolicy, strictResultHint, finalResponseDiscipline, retryHint, task.TaskID, task.StepID, task.RunID, acpruntime.ProviderQwenCode, task.StartedAtUTC.UTC().Format(time.RFC3339), task.Workspace, task.ShardID, primaryRepoScope, repoScopesJSON, pathScopesJSON, buildTaskResultTemplateJSON(task), retryTemplate, strings.TrimSpace(string(taskPayload))))
}

func buildTaskResultTemplateJSON(task acpruntime.Task) string {
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
			Runtime:    contracts.RuntimeMeta{Name: string(acpruntime.ProviderQwenCode), Version: "0.14.2"},
			StartedAt:  task.StartedAtUTC.UTC().Format(time.RFC3339),
			FinishedAt: task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
			Workspace:  task.Workspace,
			ShardID:    task.ShardID,
			RepoScope:  primaryTaskRepoScope(task.RepoScope, task.RepoScopes),
			RepoScopes: append([]string(nil), task.RepoScopes...),
			PathScopes: append([]string(nil), task.PathScopes...),
		},
		Summary:   "Task completed with contract-compliant output.",
		Changeset: buildTemplateChangeset(task),
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

func buildRetryMinimalTaskResultTemplateJSON(task acpruntime.Task) string {
	template := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:     task.TaskID,
			StepID:     task.StepID,
			RunID:      task.RunID,
			Runtime:    contracts.RuntimeMeta{Name: string(acpruntime.ProviderQwenCode), Version: "0.14.2"},
			StartedAt:  task.StartedAtUTC.UTC().Format(time.RFC3339),
			FinishedAt: task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
			Workspace:  task.Workspace,
			ShardID:    task.ShardID,
			RepoScope:  primaryTaskRepoScope(task.RepoScope, task.RepoScopes),
			RepoScopes: append([]string(nil), task.RepoScopes...),
			PathScopes: append([]string(nil), task.PathScopes...),
		},
		Summary:   "Reused existing shard artifacts.",
		Changeset: []contracts.Operation{},
		Coverage: &contracts.Coverage{
			Observed: []string{"artifacts"},
			Missing:  []string{"owner mappings", "runtime metrics"},
			Notes:    []string{"write_root artifacts reused."},
		},
	}

	raw, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func buildRetryRecoveryHint(task acpruntime.Task) string {
	lines := []string{
		`RETRY RECOVERY MODE: previous attempt may already have written shard artifacts.`,
		`- Retry goal is JSON repair, not fresh repository exploration.`,
		`- First inspect write_root, not repo roots.`,
		`- If authored docs already exist in write_root, reuse them instead of rediscovering the repository.`,
		`- Preserve repo-specific citations when existing artifacts already contain them; do not replace them with one generic runtime summary citation.`,
		`- Do NOT use todo_write, plan-style narration, or repeated broad list_directory sweeps in retry mode.`,
		`- Use at most 3 tool calls in retry mode unless a required artifact is missing from write_root.`,
		`- After optional write_root inspection, respond immediately with the final TaskResult JSON object.`,
	}

	writeRoot := strings.TrimSpace(task.WriteRoot)
	files, total, err := collectRetryWriteRootFiles(writeRoot, 8)
	assessment, assessmentErr := assessRetryManifestAtWriteRoot(writeRoot)
	switch {
	case writeRoot == "":
		lines = append(lines, `- write_root snapshot unavailable because write_root is empty.`)
	case err != nil:
		lines = append(lines, fmt.Sprintf(`- write_root snapshot unavailable: %v.`, err))
	case total == 0:
		lines = append(lines, fmt.Sprintf(`- write_root %q is empty or missing; create only the minimum missing artifacts before returning JSON.`, writeRoot))
	default:
		suffix := ""
		if total > len(files) {
			suffix = fmt.Sprintf(" (+%d more)", total-len(files))
		}
		lines = append(lines, fmt.Sprintf(`- write_root already contains %d file(s): %s%s`, total, strings.Join(files, ", "), suffix))
		if containsString(files, "shard-pack-manifest.json") {
			switch {
			case assessmentErr != nil:
				lines = append(lines, fmt.Sprintf(`- shard-pack-manifest.json is present but could not be assessed (%v); keep repo roots available while repairing JSON.`, assessmentErr))
				lines = append(lines, `- Rewrite shard-pack-manifest.json to the canonical ACP schema (version=1 integer, documents[].citation_ids, citations[].id/document_ids, stable canonical_path).`)
			case assessment.Rich:
				lines = append(lines, `- shard-pack-manifest.json is already present in write_root; read it first and reuse authored docs instead of re-reading the repository.`)
				lines = append(lines, `- When reusing authored docs, prefer "changeset": [] or one minimal operation instead of restating manifest contents.`)
				lines = append(lines, `- Do NOT copy long citations, topic arrays, or manifest document inventories into TaskResult JSON.`)
			default:
				lines = append(lines, `- shard-pack-manifest.json exists in write_root but looks skeletal/reuse-only; keep repo roots in include-directories and repair JSON without collapsing repository evidence.`)
				lines = append(lines, `- Do NOT reduce multi-document refresh evidence to one generic "cite.runtime-summary" citation.`)
				lines = append(lines, `- Preserve or restore repo-specific citations before returning the final TaskResult JSON object.`)
			}
		}
		if shouldConstrainRetryToWriteRoot(task) {
			lines = append(lines, `- Retry include-directories are constrained to write_root because the manifest and authored docs already exist.`)
		}
	}
	return strings.Join(lines, "\n")
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func buildStepSpecificPolicy(stepID string) string {
	switch stepID {
	case "refresh.step1.collect":
		return strings.Join([]string{
			`STEP POLICY refresh.step1.collect:`,
			`- Allowed upsert_entity types: service, datastore, integration, external.system, team, domain, api, component.`,
			`- Forbidden placeholder entity types: runtime_provider, runtime, metadata.`,
			`- Analyze only repository/workspace artifacts; do NOT perform web search or external browsing.`,
			`- Every provenance.evidence.path must resolve to an existing file in workspace/repo scope.`,
			`- Do NOT emit synthetic evidence paths such as search_source/*, search_query/*, search_config/*.`,
			`- Do NOT introduce unrelated incident domains (for example bidding/tender/power-system topics) unless explicitly present in repository evidence.`,
			`- If evidence is incomplete, capture gap via coverage.missing instead of synthetic placeholder entities.`,
			`- Include at least one question and at least three items in coverage.missing.`,
		}, "\n")
	case "refresh.step3.findings":
		return strings.Join([]string{
			`STEP POLICY refresh.step3.findings:`,
			`- If owner mapping is unresolved in evidence/coverage, include at least one add_finding operation.`,
			`- Each finding must include rule_id, related_ids, and provenance.evidence[].`,
			`- For observation provenance, evidence array MUST be non-empty.`,
			`- If meta.repo_scopes has 2+ scopes, include at least one upsert_edge that links entities from different repo_scope values.`,
			`- For upsert_edge use canonical keys only: edge.id, edge.type, edge.from, edge.to.`,
			`- Forbidden edge aliases: edge.kind, edge.source, edge.target.`,
			`- Minimal valid upsert_edge example: {"op":"upsert_edge","edge":{"id":"edge.cross.scope","type":"depends_on","from":"svc.a","to":"svc.b","provenance":{"kind":"inference","confidence":0.6,"evidence":[{"repo":"scope-a","path":"README.md"},{"repo":"scope-b","path":"README.md"}]}}}`,
			`- Include at least one question and at least three items in coverage.missing.`,
		}, "\n")
	default:
		if strings.HasPrefix(stepID, "refresh.") {
			return `For refresh steps include at least one question object and at least three items in coverage.missing.`
		}
		return ""
	}
}

func buildDocFirstFilesystemPolicy(task acpruntime.Task) string {
	readContextRootsJSON := "[]"
	if raw, err := json.Marshal(task.ReadContextRoots); err == nil {
		readContextRootsJSON = string(raw)
	}
	lines := []string{
		`DOCS-FIRST FILESYSTEM CONTRACT:`,
		`- Read only from meta.workspace and meta.path_scopes plus runtime read_context_roots; do not treat workspace root as implicit write target.`,
		`- Write ONLY inside write_root. Never write to workspace.yaml, schemas/*, docs/spec/*, charter/*, or analyzed user repositories.`,
		`- Use tool calls for any file writes, but keep the final assistant response limited to the required TaskResult JSON object.`,
		fmt.Sprintf(`- artifact_root (workspace-relative) = %q`, strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- read_context_roots = %s`, readContextRootsJSON),
		fmt.Sprintf(`- domain_id = %q`, strings.TrimSpace(task.DomainID)),
		fmt.Sprintf(`- agent_role = %q`, strings.TrimSpace(task.AgentRole)),
	}
	switch task.StepID {
	case "init.step1.collect", "refresh.step1.collect":
		lines = append(lines,
			`- Produce runtime-authored documents in write_root and then write shard-pack-manifest.json in write_root.`,
			`- shard-pack-manifest.json must describe every authored document, its canonical stable path, citations, and compatibility snapshot.`,
			`- You may be flexible in document structure, but promotion and rendering depend on manifest citations/topics remaining accurate.`,
		)
		lines = append(lines, artifactquality.CollectManifestContractLines(strings.TrimSpace(task.ArtifactRoot))...)
		lines = append(lines,
			`- Do NOT collapse a multi-document refresh surface to one generic "cite.runtime-summary" citation when repository evidence exists.`,
			`- Preserve repo-specific citations in shard-pack-manifest.json whenever repository files support them.`,
		)
	case "init.step3.findings", "refresh.step3.findings":
		lines = append(lines,
			`- Inspect staged final artifacts under reports/taskruns/<run_id>/staging/final from read_context_roots.`,
			`- Write validator-verdict.json in write_root.`,
			`- Validator may fix only indexes, references, or technical document issues inside write_root; do not rewrite document meaning wholesale.`,
		)
	}
	return strings.Join(lines, "\n")
}

func buildTemplateChangeset(task acpruntime.Task) []contracts.Operation {
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
					Name: humanizeScope(scope) + " Service",
					Attributes: map[string]any{
						"repo_scope": scope,
						"runtime":    acpruntime.ProviderQwenCode,
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

func humanizeScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "Repository"
	}
	parts := strings.FieldsFunc(scope, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})
	for idx, part := range parts {
		if part == "" {
			continue
		}
		parts[idx] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	name := strings.TrimSpace(strings.Join(parts, " "))
	if name == "" {
		return "Repository"
	}
	return name
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
