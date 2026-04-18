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
	ErrRunnerUnavailable = errors.New("claude-code runner is unavailable")
)

type HeadlessRunner struct {
	Command string
	Args    []string
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
	args := buildNativeDirectClaudeArgs(task, buildDirectPrompt(taskPayload, false, false))
	result, parseStage, parseErr, runErr := runClaudeCommand(ctx, task, command, args, nil)
	if runErr != nil {
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
		return maybeRepairCollectArtifacts(ctx, task, taskPayload, command, result), nil
	}

	retryArgs := buildNativeDirectClaudeArgs(
		task,
		buildDirectPrompt(
			taskPayload,
			true,
			parseStage == "extract" && (isEnvelopeResultEmptyError(parseErr) || isEnvelopeResultMalformedError(parseErr)),
		),
	)
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
		return maybeRepairCollectArtifacts(ctx, task, taskPayload, command, retryResult), nil
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
) acpruntime.Result {
	if task.StepID != "init.step1.collect" && task.StepID != "refresh.step1.collect" {
		return current
	}

	assessment, err := artifactquality.LoadManifestAssessment(task.WriteRoot)
	if err != nil || assessment.Rich {
		return current
	}

	snapshot, err := artifactquality.SnapshotWriteRoot(task.WriteRoot)
	if err != nil {
		return current
	}
	defer func() {
		_ = snapshot.Cleanup()
	}()

	repairArgs := buildNativeDirectClaudeArgs(task, buildDirectPromptWithMode(taskPayload, promptRetryArtifact, false))
	repaired, _, parseErr, runErr := runClaudeCommand(ctx, task, command, repairArgs, nil)
	if runErr != nil || parseErr != nil {
		_ = snapshot.Restore()
		return current
	}

	repairedAssessment, err := artifactquality.LoadManifestAssessment(task.WriteRoot)
	if err != nil || !repairedAssessment.Rich {
		_ = snapshot.Restore()
		return current
	}
	return repaired
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

	raw, err := taskresultextractor.Extract(stdout.Bytes())
	if err != nil {
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, "extract", err, nil
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

func buildDirectPrompt(taskPayload []byte, retry bool, requireNonEmptyResult bool) string {
	mode := promptRetryNone
	if retry {
		mode = promptRetryParse
	}
	return buildDirectPromptWithMode(taskPayload, mode, requireNonEmptyResult)
}

func buildDirectPromptWithMode(taskPayload []byte, mode promptRetryMode, requireNonEmptyResult bool) string {
	var task acpruntime.Task
	if err := json.Unmarshal(taskPayload, &task); err != nil {
		return strings.TrimSpace(fmt.Sprintf(`You are ACP runtime provider %q.
Return exactly one valid JSON object for ACP TaskResult.
Do not output markdown, code fences, or any explanatory text.
Task payload JSON:
%s`, acpruntime.ProviderClaudeCode, strings.TrimSpace(string(taskPayload))))
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
	stepPolicy := buildStepSpecificDirectPolicy(task.StepID)
	repositoryEvidencePolicy := strings.Join([]string{
		`REPOSITORY EVIDENCE RULES:`,
		`- ACP workspace scaffold (workspace.yaml, charter/, model/, reports/) is support context, not the primary source tree.`,
		`- Prefer evidence from repository files under meta.repo_scopes/meta.path_scopes when those files are available.`,
		`- meta.path_scopes may contain directories, files, or a mixed disjoint partition; treat every listed scope as in-bounds evidence for this task.`,
		`- Use ACP-generated workspace artifacts as evidence only for ACP runtime/report state, not as a substitute for repository analysis.`,
	}, "\n")
	docFirstPolicy := buildDocFirstFilesystemPolicy(task)
	retryHint := ""
	if mode != promptRetryNone {
		retryLines := []string{
			`RETRY MODE: previous output was invalid JSON.`,
			`Do not include non-ASCII symbols in numbers or timestamps.`,
			`RFC3339 timestamps only (example: 2026-04-09T15:28:49Z).`,
			`Decimals must be compact numeric literals (example: 0.7, not 0. 7).`,
			`Do NOT collapse a multi-document refresh into one generic "cite.runtime-summary" citation.`,
			`Do NOT overwrite a rich shard-pack-manifest.json with a skeletal reuse-only manifest.`,
			`Preserve repo-specific citations when repository evidence already exists or can be recovered from repo roots.`,
			`Return only JSON object, without prose.`,
		}
		if mode == promptRetryArtifact {
			retryLines[0] = `ARTIFACT REPAIR MODE: previous collect output was schema-valid but write_root artifacts look skeletal or generic-only.`
			retryLines = append(retryLines,
				`Repair artifact fidelity before returning JSON; this retry is not a fresh repository rediscovery pass.`,
				`Keep repo roots available while restoring repo-specific citations in shard-pack-manifest.json.`,
			)
		}
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
%s`, acpruntime.ProviderClaudeCode, stepPolicy, repositoryEvidencePolicy, docFirstPolicy, retryHint, nonEmptyResultHint, task.TaskID, task.StepID, task.RunID, acpruntime.ProviderClaudeCode, "claude-cli", task.StartedAtUTC.UTC().Format(time.RFC3339), task.Workspace, task.ShardID, primaryRepoScope, repoScopesJSON, pathScopesJSON, buildDirectTaskResultTemplateJSON(task), strings.TrimSpace(string(taskPayload))))
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

func buildStepSpecificDirectPolicy(stepID string) string {
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
			`- Every provenance.evidence.path must resolve to an existing file in workspace/repo scope.`,
			`- Do NOT emit synthetic evidence paths such as search_source/*, search_query/*, search_config/*.`,
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
