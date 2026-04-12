package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	workspace := t.TempDir()
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
		Workspace:    workspace,
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
	if !strings.Contains(message, "invalid taskresult") || !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("unexpected error message %q", message)
	}
	rawMetaFiles, globErr := filepath.Glob(filepath.Join(workspace, "reports", "taskruns", "raw", "*-meta.json"))
	if globErr != nil {
		t.Fatalf("glob raw output meta files: %v", globErr)
	}
	if len(rawMetaFiles) == 0 {
		t.Fatalf("expected parse-fail raw output metadata in %s", filepath.Join(workspace, "reports", "taskruns", "raw"))
	}
}

func TestHeadlessRunnerLegacyPassthroughWhenArgsConfigured(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			fmt.Sprintf("cat >/dev/null; echo '%s'", validTaskResultJSON("claude-code", "legacy-test")),
		},
	}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-legacy",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    "/tmp/workspace",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run legacy passthrough: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Name != "claude-code" {
		t.Fatalf("unexpected runtime name %q", result.TaskResult.Meta.Runtime.Name)
	}
}

func TestHeadlessRunnerUsesLegacyModeForNonClaudeCommandWithEmptyArgs(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "custom-runner")
	script := `#!/bin/sh
set -eu
input="$(cat)"
if [ -z "$input" ]; then
  echo "missing stdin payload" >&2
  exit 1
fi
cat <<'JSON'
{"meta":{"task_id":"task-1","step_id":"init.step1.collect","runtime":{"name":"claude-code","version":"legacy"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}
JSON
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write custom command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-legacy-empty-args",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    "/tmp/workspace",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected legacy mode success: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Version != "legacy" {
		t.Fatalf("expected legacy runtime version, got %q", result.TaskResult.Meta.Runtime.Version)
	}
}

func TestHeadlessRunnerNativeDirectClaudeParsesEnvelopeResult(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "claude")
	script := `#!/bin/sh
set -eu
cat <<'JSON'
{"type":"result","result":"{\"meta\":{\"task_id\":\"task-1\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"claude-code\",\"version\":\"claude-cli\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}
JSON
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude direct command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-native-success",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    "/tmp/workspace",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run native direct claude: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Name != "claude-code" {
		t.Fatalf("unexpected runtime name %q", result.TaskResult.Meta.Runtime.Name)
	}
}

func TestHeadlessRunnerNativeDirectClaudeRetriesOnParseFailure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "claude")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY MODE"; then
  cat <<'JSON'
{"result":"{\"meta\":{\"task_id\":\"task-1\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"claude-code\",\"version\":\"claude-cli\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}
JSON
  exit 0
fi
echo "This is not JSON"
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude retry command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-native-retry",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    "/tmp/workspace",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected retry run to succeed: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Version != "claude-cli" {
		t.Fatalf("unexpected runtime version %q", result.TaskResult.Meta.Runtime.Version)
	}
}

func TestHeadlessRunnerPreservesContextCancellation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "claude-cancel-stub.sh")
	script := `#!/bin/sh
set -eu
cat >/dev/null
sleep 10
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write cancel command: %v", err)
	}

	runner := HeadlessRunner{
		Command: commandPath,
		Args: []string{
			"-c",
			"cat >/dev/null; sleep 10",
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(150*time.Millisecond, cancel)

	_, err := runner.Run(ctx, acpruntime.Task{
		TaskID:       "task-cancel",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    "/tmp/workspace",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected error to preserve context.Canceled, got %v", err)
	}
}

func TestBuildDirectPromptIncludesStepSpecificPolicies(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-refresh",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    "/tmp/workspace",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildDirectPrompt(raw, false)
	if !strings.Contains(prompt, "STEP POLICY refresh.step1.collect") {
		t.Fatalf("expected step1 policy block in prompt")
	}
	if !strings.Contains(prompt, "Forbidden placeholder entity types: runtime_provider, runtime, metadata.") {
		t.Fatalf("expected forbidden runtime placeholder policy in prompt")
	}
	if !strings.Contains(prompt, "question IDs MUST use canonical form without numeric suffixes") {
		t.Fatalf("expected canonical question-id policy in prompt")
	}
}

func validTaskResultJSON(runtimeName string, runtimeVersion string) string {
	return fmt.Sprintf(`{"meta":{"task_id":"task-1","step_id":"init.step1.collect","runtime":{"name":"%s","version":"%s"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}`, runtimeName, runtimeVersion)
}
