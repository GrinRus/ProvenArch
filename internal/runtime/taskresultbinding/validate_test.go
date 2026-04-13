package taskresultbinding

import (
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestValidateSuccess(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		RepoScope:    "payments-service",
		Workspace:    "/tmp/workspace",
		StartedAtUTC: time.Date(2026, 4, 12, 8, 0, 0, 0, time.UTC),
	}
	result := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:    "task-1",
			RunID:     "run-1",
			StepID:    "init.step1.collect",
			RepoScope: "payments-service",
			Runtime:   contracts.RuntimeMeta{Name: "qwen-code"},
			StartedAt: "2026-04-12T08:00:00Z",
		},
		Summary:   "ok",
		Changeset: []contracts.Operation{},
	}

	if err := Validate(task, result, acpruntime.ProviderQwenCode); err != nil {
		t.Fatalf("expected binding validation success, got %v", err)
	}
}

func TestValidateAllowsEmptyRunIDInResult(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID: "task-1",
		RunID:  "run-1",
		StepID: "init.step1.collect",
	}
	result := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:    "task-1",
			StepID:    "init.step1.collect",
			Runtime:   contracts.RuntimeMeta{Name: "claude-code"},
			StartedAt: "2026-04-12T08:00:00Z",
		},
		Summary:   "ok",
		Changeset: []contracts.Operation{},
	}
	if err := Validate(task, result, acpruntime.ProviderClaudeCode); err != nil {
		t.Fatalf("empty meta.run_id should be allowed, got %v", err)
	}
}

func TestValidateReportsMismatches(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:    "task-expected",
		RunID:     "run-expected",
		StepID:    "refresh.step3.findings",
		RepoScope: "users-service",
	}
	result := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:    "task-old",
			RunID:     "run-old",
			StepID:    "init.step1.collect",
			RepoScope: "payments-service",
			Runtime:   contracts.RuntimeMeta{Name: "claude-code"},
			StartedAt: "2026-04-12T08:00:00Z",
		},
		Summary:   "ok",
		Changeset: []contracts.Operation{},
	}

	err := Validate(task, result, acpruntime.ProviderQwenCode)
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	message := err.Error()
	for _, fragment := range []string{
		"meta.task_id",
		"meta.run_id",
		"meta.step_id",
		"meta.repo_scope",
		"meta.runtime.name",
	} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("expected %q in mismatch message: %s", fragment, message)
		}
	}
}
