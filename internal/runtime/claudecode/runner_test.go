package claudecode

import (
	"context"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestFakeRunnerCollectStep(t *testing.T) {
	t.Parallel()

	runner := FakeRunner{}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    "/tmp/workspace",
		RepoScopes:   []string{"payments-service", "users-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run fake collect: %v", err)
	}
	if len(result.TaskResult.Changeset) != 2 {
		t.Fatalf("expected one service per repo scope, got %d", len(result.TaskResult.Changeset))
	}
	if len(result.TaskResult.Questions) != 2 {
		t.Fatalf("expected one question per repo scope, got %d", len(result.TaskResult.Questions))
	}
	firstEntity := result.TaskResult.Changeset[0].Entity
	if firstEntity == nil {
		t.Fatalf("expected first changeset op to contain entity")
	}
	if firstEntity.Name == "Payments Service Service" {
		t.Fatalf("service name should not duplicate suffix: %q", firstEntity.Name)
	}
}

func TestFakeRunnerFindingsStep(t *testing.T) {
	t.Parallel()

	runner := FakeRunner{}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-2",
		RunID:        "run-1",
		StepID:       "init.step3.findings",
		Workspace:    "/tmp/workspace",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run fake findings: %v", err)
	}
	if len(result.TaskResult.Changeset) != 1 {
		t.Fatalf("expected one finding op, got %d", len(result.TaskResult.Changeset))
	}
	if result.TaskResult.Changeset[0].Op != "add_finding" {
		t.Fatalf("expected add_finding op, got %q", result.TaskResult.Changeset[0].Op)
	}
}

func TestHeadlessRunnerUnavailableClassifiesAsRunnerUnavailable(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{Command: "definitely-missing-acp-headless-command"}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    "/tmp/workspace",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected headless runner unavailable error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerUnavailable) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "is unavailable") {
		t.Fatalf("unexpected error message %q", message)
	}
}

func TestHeadlessRunnerInvalidTaskResultClassifiesAsParseFailed(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			"cat >/dev/null; echo '{\"meta\": {\"task_id\": \"bad\"}}'",
		},
	}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-2",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    "/tmp/workspace",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected parse-failed error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "invalid taskresult") {
		t.Fatalf("unexpected error message %q", message)
	}
}
