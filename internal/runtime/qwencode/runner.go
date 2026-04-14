package qwencode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultbinding"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultextractor"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultrepair"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

var (
	ErrRunnerUnavailable = errors.New("qwen-code runner is unavailable")
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
		if len(r.Args) == 0 && isPromptFlagUnsupported(runErr) {
			runErr = fmt.Errorf(
				"%w (compatibility guard: current qwen command does not support --prompt in default mode; use explicit HeadlessRunner.Args for legacy positional prompt flow)",
				runErr,
			)
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
		return result, nil
	}
	finalStdout := result.Stdout
	finalStderr := result.Stderr

	// Live qwen output can occasionally contain malformed tokens. Retry once with
	// an explicitly stricter prompt before classifying as parse failure.
	if len(r.Args) == 0 {
		retryArgs := buildDefaultQwenArgs(task, buildPrompt(taskPayload, true))
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
			return retryResult, nil
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
	args := []string{"--output-format", "json", "--yolo", "--channel", "CI"}
	workspace := strings.TrimSpace(task.Workspace)
	if workspace != "" {
		args = append(args, "--include-directories", workspace)
	}
	args = append(args, "--prompt", prompt)
	return args
}

func isPromptFlagUnsupported(err error) bool {
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	if !strings.Contains(text, "--prompt") {
		return false
	}
	return strings.Contains(text, "unknown option") ||
		strings.Contains(text, "unknown argument") ||
		strings.Contains(text, "unrecognized option") ||
		strings.Contains(text, "invalid option")
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
	workspace := strings.TrimSpace(task.Workspace)
	if workspace != "" {
		cmd.Dir = workspace
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return acpruntime.Result{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
			}, "", nil, ctxErr
		}
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, "", nil, runnerdiag.BuildExecFailure(err, stdout.String(), stderr.String())
	}

	rawTaskResult, err := taskresultextractor.Extract(stdout.Bytes())
	if err != nil {
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, "extract", err, nil
	}
	rawTaskResult = taskresultrepair.RepairEdgeAliases(rawTaskResult)
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

func buildPrompt(taskPayload []byte, retry bool) string {
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
	retryHint := ""
	if retry {
		retryHint = strings.Join([]string{
			`RETRY MODE: previous output was invalid JSON.`,
			`Do not include non-ASCII symbols in numbers or timestamps.`,
			`RFC3339 timestamps only (example: 2026-04-09T15:28:49Z).`,
			`Decimals must be compact numeric literals (example: 0.7, not 0. 7).`,
			`COMPACT JSON MODE: keep output concise and deterministic.`,
			`- Limit changeset to the minimum actionable operations for this step.`,
			`- Keep coverage.notes short (<=2 entries).`,
			`- Avoid long prose in summary; keep a single sentence.`,
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

Serialized runtime task JSON (context only):
%s`, acpruntime.ProviderQwenCode, stepPolicy, retryHint, task.TaskID, task.StepID, task.RunID, acpruntime.ProviderQwenCode, task.StartedAtUTC.UTC().Format(time.RFC3339), task.Workspace, task.ShardID, primaryRepoScope, repoScopesJSON, pathScopesJSON, buildTaskResultTemplateJSON(task), strings.TrimSpace(string(taskPayload))))
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

func buildStepSpecificPolicy(stepID string) string {
	switch stepID {
	case "refresh.step2.service_collect", "refresh.step1.collect":
		return strings.Join([]string{
			fmt.Sprintf(`STEP POLICY %s:`, stepID),
			`- Allowed upsert_entity types: service, datastore, integration, external.system, team, domain, api, component.`,
			`- Forbidden placeholder entity types: runtime_provider, runtime, metadata.`,
			`- Analyze only repository/workspace artifacts; do NOT perform web search or external browsing.`,
			`- Every provenance.evidence.path must resolve to an existing file in workspace/repo scope.`,
			`- Do NOT emit synthetic evidence paths such as search_source/*, search_query/*, search_config/*.`,
			`- Do NOT introduce unrelated incident domains (for example bidding/tender/power-system topics) unless explicitly present in repository evidence.`,
			`- If evidence is incomplete, capture gap via coverage.missing instead of synthetic placeholder entities.`,
			`- Include at least one question and at least three items in coverage.missing.`,
		}, "\n")
	case "refresh.step4.service_findings", "refresh.step3.findings":
		return strings.Join([]string{
			fmt.Sprintf(`STEP POLICY %s:`, stepID),
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

func buildTemplateChangeset(task acpruntime.Task) []contracts.Operation {
	scopes := append([]string(nil), task.RepoScopes...)
	if len(scopes) == 0 {
		scopes = []string{"repository"}
	}
	changes := make([]contracts.Operation, 0, len(scopes))
	switch task.StepID {
	case "init.step4.service_findings", "refresh.step4.service_findings", "init.step3.findings", "refresh.step3.findings":
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
