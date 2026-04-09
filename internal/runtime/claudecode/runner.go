package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

var (
	ErrRunnerUnavailable = errors.New("claude-code runner is unavailable")
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
	command := r.commandName()

	taskPayload, err := json.Marshal(task)
	if err != nil {
		return acpruntime.Result{}, fmt.Errorf("marshal runner task: %w", err)
	}

	cmd := exec.CommandContext(ctx, command, r.Args...)
	cmd.Stdin = bytes.NewReader(taskPayload)

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
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: %s", ErrRunnerUnavailable, errText),
			err,
		)
	}

	raw := bytes.TrimSpace(stdout.Bytes())
	taskResult, err := contracts.ParseTaskResult(raw)
	if err != nil {
		return acpruntime.Result{}, acpruntime.WrapRunnerError(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q returned invalid taskresult: %v", acpruntime.ProviderClaudeCode, err),
			err,
		)
	}

	return acpruntime.Result{
		TaskResult: taskResult,
		RawJSON:    raw,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}, nil
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
				RepoScopes: repoScopes,
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
				RepoScopes: repoScopes,
			},
			Summary:   "Fake findings completed",
			Changeset: makeFindingsChangeset(repoScopes),
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
