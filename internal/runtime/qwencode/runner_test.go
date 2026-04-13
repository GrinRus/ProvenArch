package qwencode

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestHeadlessRunnerUnavailableClassifiesAsRunnerUnavailable(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{Command: "definitely-missing-acp-qwen-command"}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
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

func TestHeadlessRunnerUnavailableIncludesStdoutExcerptWhenStderrEmpty(t *testing.T) {
	t.Parallel()

	commandPath := filepath.Join(t.TempDir(), "qwen-unavailable-stdout.sh")
	script := `#!/bin/sh
set -eu
echo "qwen failed due to transient setup issue"
exit 1
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write unavailable stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-unavailable-stdout",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected runner unavailable error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerUnavailable) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "stdout_excerpt=") {
		t.Fatalf("expected stdout excerpt in unavailable error message, got %q", message)
	}
}

func TestHeadlessRunnerUnsupportedPromptFlagReportsCompatibilityGuard(t *testing.T) {
	t.Parallel()

	commandPath := filepath.Join(t.TempDir(), "qwen-unsupported-prompt.sh")
	script := `#!/bin/sh
set -eu
echo "unknown option --prompt" >&2
exit 1
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write unsupported-prompt stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-unsupported-prompt",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected runner unavailable error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerUnavailable) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "compatibility guard") {
		t.Fatalf("expected compatibility guard hint in error message, got %q", message)
	}
	if !strings.Contains(message, "HeadlessRunner.Args") {
		t.Fatalf("expected legacy path hint in error message, got %q", message)
	}
}

func TestHeadlessRunnerInvalidTaskResultClassifiesAsParseFailed(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			"echo '{\"response\":\"not-json\"}'",
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
	if !strings.Contains(message, "parse_stage=") {
		t.Fatalf("expected parse_stage marker in parse-fail message, got %q", message)
	}
	rawMetaFiles, globErr := filepath.Glob(filepath.Join(workspace, "reports", "taskruns", "raw", "*-meta.json"))
	if globErr != nil {
		t.Fatalf("glob raw output meta files: %v", globErr)
	}
	if len(rawMetaFiles) == 0 {
		t.Fatalf("expected parse-fail raw output metadata in %s", filepath.Join(workspace, "reports", "taskruns", "raw"))
	}
}

func TestHeadlessRunnerSchemaInvalidCandidateClassifiesAsSchemaStage(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n' '{"meta":{"task_id":"task-1"}}'`,
		},
	}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
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
	if !strings.Contains(message, "parse_stage=schema") {
		t.Fatalf("expected parse_stage=schema, got %q", message)
	}
}

func TestHeadlessRunnerRetriesOnceOnParseFailureForDefaultArgs(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-retry-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY MODE"; then
  cat <<'JSON'
{"meta":{"task_id":"task-1","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[{"op":"upsert_entity","entity":{"id":"svc.payments-service","type":"service","name":"Payments Service","attributes":{"repo_scope":"payments-service"},"provenance":{"kind":"observation","confidence":0.7,"evidence":[{"repo":"payments-service","path":"README.md"}]}}}]}
JSON
  exit 0
fi
echo '{"response":"not-json"}'
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write retry command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected retry run to succeed: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Name != "qwen-code" {
		t.Fatalf("unexpected runtime name %q", result.TaskResult.Meta.Runtime.Name)
	}
}

func TestHeadlessRunnerDefaultArgsUsePromptFlagForExecution(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-prompt-flag-stub.sh")
	script := `#!/bin/sh
set -eu
has_prompt=0
prompt_value=""
expect_prompt_value=0
for arg in "$@"; do
  if [ "$expect_prompt_value" -eq 1 ]; then
    prompt_value="$arg"
    expect_prompt_value=0
    continue
  fi
  if [ "$arg" = "--prompt" ]; then
    has_prompt=1
    expect_prompt_value=1
    continue
  fi
done
if [ "$has_prompt" -ne 1 ] || [ -z "$prompt_value" ]; then
  echo "missing --prompt argument" >&2
  exit 1
fi
cat <<'JSON'
{"meta":{"task_id":"task-prompt-flag","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}
JSON
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write prompt-flag command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-prompt-flag",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected default run to succeed with --prompt: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-prompt-flag" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
}

func TestHeadlessRunnerRetryParseFailureUsesRetryOutputInRunnerError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-retry-parse-fail-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY MODE"; then
  echo 'retry invalid payload'
  echo 'retry stderr detail' >&2
  exit 0
fi
echo 'first invalid payload'
echo 'first stderr detail' >&2
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write retry-parse-fail command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected parse-failed error")
	}

	code, _, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("unexpected error code %q", code)
	}

	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError details")
	}
	if !strings.Contains(runnerErr.Stdout, "retry invalid payload") {
		t.Fatalf("expected retry stdout in runner error, got %q", runnerErr.Stdout)
	}
	if strings.Contains(runnerErr.Stdout, "first invalid payload") {
		t.Fatalf("did not expect first-attempt stdout in runner error, got %q", runnerErr.Stdout)
	}
	if !strings.Contains(runnerErr.Stderr, "retry stderr detail") {
		t.Fatalf("expected retry stderr in runner error, got %q", runnerErr.Stderr)
	}
	if strings.Contains(runnerErr.Stderr, "first stderr detail") {
		t.Fatalf("did not expect first-attempt stderr in runner error, got %q", runnerErr.Stderr)
	}
}

func TestHeadlessRunnerPreservesContextCancellation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-cancel-stub.sh")
	script := `#!/bin/sh
set -eu
sleep 10
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write cancel command: %v", err)
	}

	runner := HeadlessRunner{
		Command: commandPath,
		Args:    []string{},
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(150*time.Millisecond, cancel)

	_, err := runner.Run(ctx, acpruntime.Task{
		TaskID:       "task-cancel",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
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

func TestHeadlessRunnerParsesTaskResultFromResponseField(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n' '{"response":"{\"meta\":{\"task_id\":\"task-1\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"qwen-code\",\"version\":\"test\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}'`,
		},
	}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run qwen response parser: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Name != "qwen-code" {
		t.Fatalf("unexpected runtime name %q", result.TaskResult.Meta.Runtime.Name)
	}
}

func TestHeadlessRunnerParsesTaskResultFromJsonArrayResultMessage(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n' '[{"type":"system","subtype":"session_start"},{"type":"result","result":"{\"meta\":{\"task_id\":\"task-1\",\"step_id\":\"init.step3.findings\",\"runtime\":{\"name\":\"qwen-code\",\"version\":\"test\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}]'`,
		},
	}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step3.findings",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run qwen array parser: %v", err)
	}
	if result.TaskResult.Meta.StepID != "init.step3.findings" {
		t.Fatalf("unexpected step id %q", result.TaskResult.Meta.StepID)
	}
}

func TestHeadlessRunnerParsesTaskResultFromQwenAssistantContentEvents(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n' '[{"type":"system","subtype":"init"},{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"ignored"},{"type":"text","text":"{\"meta\":{\"task_id\":\"task-1\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"qwen-code\",\"version\":\"test\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}" }]}}]'`,
		},
	}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run qwen event content parser: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Name != "qwen-code" {
		t.Fatalf("unexpected runtime name %q", result.TaskResult.Meta.Runtime.Name)
	}
}

func TestHeadlessRunnerParsesTaskResultFromJsonObjectsStream(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n%s\n' '{"type":"system","message":"ignored"}' '{"type":"result","result":"{\"meta\":{\"task_id\":\"task-1\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"qwen-code\",\"version\":\"test\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}'`,
		},
	}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run qwen stream parser: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-1" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
}

func TestHeadlessRunnerBindingMismatchClassifiesAsParseFailed(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n' '{"meta":{"task_id":"task-stale","step_id":"init.step1.collect","run_id":"run-stale","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}'`,
		},
	}

	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-expected",
		RunID:        "run-expected",
		StepID:       "init.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected binding parse-failed error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "parse_stage=binding") {
		t.Fatalf("expected parse_stage=binding in parse-fail message, got %q", message)
	}
}

func TestBuildPromptIncludesStepSpecificPolicies(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-refresh",
		RunID:        "run-1",
		StepID:       "refresh.step3.findings",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, false)
	if !strings.Contains(prompt, "STEP POLICY refresh.step3.findings") {
		t.Fatalf("expected step3 policy block in prompt")
	}
	if !strings.Contains(prompt, "include at least one add_finding operation") {
		t.Fatalf("expected required add_finding policy in prompt")
	}
	if !strings.Contains(prompt, "coverage.missing MUST use canonical terms only") {
		t.Fatalf("expected canonical coverage dictionary policy in prompt")
	}
}

func TestBuildPromptRefreshStep1CollectIncludesNoWebSearchPolicy(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-refresh-step1",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, false)
	if !strings.Contains(prompt, "do NOT perform web search or external browsing") {
		t.Fatalf("expected no-web-search rule in step1 collect policy")
	}
	if !strings.Contains(prompt, "Do NOT emit synthetic evidence paths such as search_source/*") {
		t.Fatalf("expected synthetic evidence-path ban in step1 collect policy")
	}
}

func TestBuildDefaultQwenArgsUsesPromptFlagWithIncludeDirectories(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-args",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    "/tmp/workspace",
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	args := buildDefaultQwenArgs(task, "prompt-text")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--include-directories /tmp/workspace") {
		t.Fatalf("expected include-directories in args, got %q", joined)
	}
	if !strings.Contains(joined, "--prompt prompt-text") {
		t.Fatalf("expected --prompt usage in args, got %q", joined)
	}
	if args[len(args)-2] != "--prompt" || args[len(args)-1] != "prompt-text" {
		t.Fatalf("expected prompt to be appended as --prompt <value>, got %#v", args)
	}
}
