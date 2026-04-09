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
		args = []string{"--output-format", "json", "--yolo", "--channel", "CI", buildPrompt(taskPayload)}
	}

	cmd := exec.CommandContext(ctx, command, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errText := strings.TrimSpace(stderr.String())
		if errText == "" {
			errText = err.Error()
		}
		return acpruntime.Result{}, acpruntime.WrapRunnerError(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: %s", ErrRunnerUnavailable, errText),
			err,
		)
	}

	rawTaskResult, err := extractTaskResultJSON(stdout.Bytes())
	if err != nil {
		return acpruntime.Result{}, acpruntime.WrapRunnerError(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q returned invalid taskresult: %v", acpruntime.ProviderQwenCode, err),
			err,
		)
	}
	taskResult, err := contracts.ParseTaskResult(rawTaskResult)
	if err != nil {
		return acpruntime.Result{}, acpruntime.WrapRunnerError(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q returned invalid taskresult: %v", acpruntime.ProviderQwenCode, err),
			err,
		)
	}

	return acpruntime.Result{
		TaskResult: taskResult,
		RawJSON:    rawTaskResult,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}, nil
}

func buildPrompt(taskPayload []byte) string {
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
	refreshHint := ""
	if strings.HasPrefix(task.StepID, "refresh.") {
		refreshHint = `For refresh steps include at least one question object and at least three items in coverage.missing.`
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
%s

Set meta fields exactly:
- meta.task_id = %q
- meta.step_id = %q
- meta.run_id = %q
- meta.runtime.name = %q
- meta.started_at = %q
- meta.workspace = %q
- meta.repo_scopes = %s

Schema-valid template for this task (copy structure and field TYPES, then refine values with available evidence):
%s

Serialized runtime task JSON (context only):
%s`, acpruntime.ProviderQwenCode, refreshHint, task.TaskID, task.StepID, task.RunID, acpruntime.ProviderQwenCode, task.StartedAtUTC.UTC().Format(time.RFC3339), task.Workspace, repoScopesJSON, buildTaskResultTemplateJSON(task), strings.TrimSpace(string(taskPayload))))
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
			RepoScopes: append([]string(nil), task.RepoScopes...),
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

func extractTaskResultJSON(raw []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, errors.New("empty stdout")
	}
	if _, err := contracts.ParseTaskResult(trimmed); err == nil {
		return trimmed, nil
	}

	if parsed, err := parseTaskResultFromJSON(trimmed); err == nil {
		return parsed, nil
	}
	if parsed, err := parseTaskResultFromText(string(trimmed)); err == nil {
		return parsed, nil
	}

	return nil, errors.New("unable to extract valid TaskResult JSON from qwen output")
}

func parseTaskResultFromJSON(raw []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return parseCandidateValue(value)
}

func parseCandidateValue(value any) ([]byte, error) {
	switch typed := value.(type) {
	case string:
		return parseTaskResultFromText(typed)
	case map[string]any:
		raw, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		if _, err := contracts.ParseTaskResult(raw); err == nil {
			return raw, nil
		}

		priorityKeys := []string{"response", "result", "message", "content", "text"}
		seen := map[string]struct{}{}
		for _, key := range priorityKeys {
			seen[key] = struct{}{}
			nested, ok := typed[key]
			if !ok {
				continue
			}
			if parsed, nestedErr := parseCandidateValue(nested); nestedErr == nil {
				return parsed, nil
			}
		}
		for key, nested := range typed {
			if _, ok := seen[key]; ok {
				continue
			}
			if parsed, nestedErr := parseCandidateValue(nested); nestedErr == nil {
				return parsed, nil
			}
		}
	case []any:
		for idx := len(typed) - 1; idx >= 0; idx-- {
			item := typed[idx]
			if parsed, err := parseCandidateValue(item); err == nil {
				return parsed, nil
			}
		}
	}
	return nil, errors.New("unsupported candidate value")
}

func parseTaskResultFromText(text string) ([]byte, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, errors.New("empty text")
	}

	if candidate, ok := stripCodeFence(trimmed); ok {
		trimmed = candidate
	}

	if json.Valid([]byte(trimmed)) {
		if parsed, err := parseTaskResultFromJSON([]byte(trimmed)); err == nil {
			return parsed, nil
		}
	}

	candidates := extractJSONObjects(trimmed)
	if len(candidates) == 0 {
		return nil, errors.New("no json object found in text")
	}
	for _, candidate := range candidates {
		if parsed, err := parseTaskResultFromJSON([]byte(candidate)); err == nil {
			return parsed, nil
		}
	}
	return nil, errors.New("unable to parse taskresult from extracted json objects")
}

func stripCodeFence(input string) (string, bool) {
	if !strings.HasPrefix(input, "```") {
		return "", false
	}
	withoutPrefix := input[3:]
	if newline := strings.IndexByte(withoutPrefix, '\n'); newline >= 0 {
		withoutPrefix = withoutPrefix[newline+1:]
	}
	if tail := strings.LastIndex(withoutPrefix, "```"); tail >= 0 {
		withoutPrefix = withoutPrefix[:tail]
	}
	candidate := strings.TrimSpace(withoutPrefix)
	if candidate == "" {
		return "", false
	}
	return candidate, true
}

func extractJSONObjects(input string) []string {
	candidates := make([]string, 0, 4)
	start := strings.IndexByte(input, '{')
	for start >= 0 {
		depth := 0
		inString := false
		escapeNext := false
		found := false
		for idx := start; idx < len(input); idx++ {
			ch := input[idx]
			if inString {
				if escapeNext {
					escapeNext = false
					continue
				}
				switch ch {
				case '\\':
					escapeNext = true
				case '"':
					inString = false
				}
				continue
			}
			switch ch {
			case '"':
				inString = true
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					candidate := strings.TrimSpace(input[start : idx+1])
					if candidate != "" && json.Valid([]byte(candidate)) {
						candidates = append(candidates, candidate)
					}
					found = true
					break
				}
			}
		}
		if !found {
			break
		}
		next := strings.IndexByte(input[start+1:], '{')
		if next < 0 {
			break
		}
		start = start + 1 + next
	}
	return candidates
}
