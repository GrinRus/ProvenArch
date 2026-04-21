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
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
)

func TestFakeRunnerCollectStep(t *testing.T) {
	t.Parallel()

	runner := FakeRunner{}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
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
		Workspace:    t.TempDir(),
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

	commandPath := filepath.Join(t.TempDir(), "claude-unavailable-stdout.sh")
	script := `#!/bin/sh
set -eu
echo "claude failed before writing stderr"
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
	if !strings.Contains(message, "parse_stage=exec") || !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("expected raw-output diagnostics in unavailable error message, got %q", message)
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected runner error details")
	}
	if runnerErr.Failure.FailureClass != "runner_unavailable" {
		t.Fatalf("expected failure class runner_unavailable, got %+v", runnerErr.Failure)
	}
	if runnerErr.Failure.FailureArtifactPath == "" || runnerErr.Failure.RawOutputPath == "" {
		t.Fatalf("expected structured failure artifact refs, got %+v", runnerErr.Failure)
	}
}

func TestHeadlessRunnerTimeoutClassifiesAsRuntimeTimeout(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			"sleep 5",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := runner.Run(ctx, acpruntime.Task{
		TaskID:       "task-timeout",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected runtime timeout error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRuntimeTimeout) {
		t.Fatalf("expected runtime_timeout code, got %q (%v)", code, err)
	}
	if !strings.Contains(message, "timed out") {
		t.Fatalf("expected timeout message, got %q", message)
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected runner error details")
	}
	if runnerErr.Failure.FailureClass != "runtime_timeout" {
		t.Fatalf("expected runtime_timeout failure class, got %+v", runnerErr.Failure)
	}
	if runnerErr.Failure.FailureArtifactPath == "" || runnerErr.Failure.RawOutputPath == "" {
		t.Fatalf("expected structured timeout artifact refs, got %+v", runnerErr.Failure)
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

func TestHeadlessRunnerStdinPassthroughWhenArgsConfigured(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			fmt.Sprintf("cat >/dev/null; echo '%s'", validTaskResultJSON("task-passthrough", "init.step1.collect", "claude-code", "passthrough-test")),
		},
	}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-passthrough",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run stdin passthrough: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Name != "claude-code" {
		t.Fatalf("unexpected runtime name %q", result.TaskResult.Meta.Runtime.Name)
	}
}

func TestHeadlessRunnerUsesStdinPassthroughForNonClaudeCommandWithEmptyArgs(t *testing.T) {
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
{"meta":{"task_id":"task-passthrough-empty-args","step_id":"init.step1.collect","runtime":{"name":"claude-code","version":"passthrough"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}
JSON
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write custom command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-passthrough-empty-args",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected stdin passthrough success: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Version != "passthrough" {
		t.Fatalf("expected passthrough runtime version, got %q", result.TaskResult.Meta.Runtime.Version)
	}
}

func TestBuildNativeDirectClaudeArgsAddsWorkspaceAndRepoDirectories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "payments-service")
	for _, dir := range []string{workspace, repoPath} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: payments-service\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	task := acpruntime.Task{
		TaskID:       "task-direct-args",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}

	args := buildNativeDirectClaudeArgs(task, "prompt-text")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--add-dir "+workspace) {
		t.Fatalf("expected workspace add-dir in args, got %q", joined)
	}
	if !strings.Contains(joined, "--add-dir "+repoPath) {
		t.Fatalf("expected repo add-dir in args, got %q", joined)
	}
	if !strings.Contains(joined, "-p prompt-text") {
		t.Fatalf("expected prompt flag in args, got %q", joined)
	}
}

func TestHeadlessRunnerNativeDirectClaudeParsesEnvelopeResult(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "claude")
	script := `#!/bin/sh
set -eu
cat <<'JSON'
{"type":"result","result":"{\"meta\":{\"task_id\":\"task-native-success\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"claude-code\",\"version\":\"claude-cli\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}
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
		Workspace:    t.TempDir(),
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
{"result":"{\"meta\":{\"task_id\":\"task-native-retry\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"claude-code\",\"version\":\"claude-cli\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}
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
		Workspace:    t.TempDir(),
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

func TestHeadlessRunnerNativeDirectClaudeRetriesOnEmptyEnvelopeResult(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "claude")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
	if echo "$last_arg" | grep -q "STRICT RESULT JSON MODE"; then
  cat <<'JSON'
{"result":"{\"meta\":{\"task_id\":\"task-native-empty-retry\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"claude-code\",\"version\":\"claude-cli\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}
JSON
  exit 0
fi
cat <<'JSON'
{"type":"result","result":""}
JSON
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude empty-result retry command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-native-empty-retry",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected empty-envelope retry run to succeed: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-native-empty-retry" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
}

func TestHeadlessRunnerNativeDirectClaudeRetriesOnMalformedEnvelopeResult(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "claude")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "STRICT RESULT JSON MODE"; then
  cat <<'JSON'
{"meta":{"task_id":"task-native-malformed-retry","step_id":"init.step1.collect","runtime":{"name":"claude-code","version":"claude-cli"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}
JSON
  exit 0
fi
cat <<'JSON'
{"result":"{\"meta\":{\"task_id\":\"task-native-malformed-retry\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"claude-code\",\"version\":\"claude-cli\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"broken\",\"changeset\":[]"}
JSON
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude malformed-result retry command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-native-malformed-retry",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected malformed-envelope retry run to succeed: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-native-malformed-retry" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
}

func TestHeadlessRunnerNativeDirectClaudeCapturesInitialAndRetryPromptArtifactsWithWorkspacePromptPack(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(filepath.Join(workspace, "skills", "prompt-packs"), 0o755); err != nil {
		t.Fatalf("mkdir prompt-pack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "skills", "prompt-packs", "collect-context.md"), []byte("Custom claude collect-context.\n"), 0o644); err != nil {
		t.Fatalf("write custom prompt pack: %v", err)
	}

	commandPath := filepath.Join(tempDir, "claude")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY MODE"; then
  cat <<'JSON'
{"result":"{\"meta\":{\"task_id\":\"task-claude-prompt-artifacts\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"claude-code\",\"version\":\"claude-cli\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}
JSON
  exit 0
fi
echo "This is not JSON"
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude prompt-artifacts command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	if _, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-claude-prompt-artifacts",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("expected claude run with retry prompt artifacts to succeed: %v", err)
	}

	metaFiles, err := filepath.Glob(filepath.Join(workspace, "reports", "taskruns", "raw", "*-prompt-meta.json"))
	if err != nil {
		t.Fatalf("glob prompt artifact metadata: %v", err)
	}
	if len(metaFiles) != 2 {
		t.Fatalf("expected initial + retry prompt metadata files, got %d", len(metaFiles))
	}
	foundPackContent := false
	foundRetryAttempt := false
	for _, metaPath := range metaFiles {
		rawMeta, readErr := os.ReadFile(metaPath)
		if readErr != nil {
			t.Fatalf("read prompt metadata %q: %v", metaPath, readErr)
		}
		meta := map[string]any{}
		if err := json.Unmarshal(rawMeta, &meta); err != nil {
			t.Fatalf("parse prompt metadata %q: %v", metaPath, err)
		}
		if strings.TrimSpace(meta["attempt"].(string)) == "parse-retry" {
			foundRetryAttempt = true
		}
		promptBlock, ok := meta["prompt"].(map[string]any)
		if !ok {
			t.Fatalf("metadata %q missing prompt block", metaPath)
		}
		promptRel := strings.TrimSpace(promptBlock["relative_path"].(string))
		promptRaw, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(promptRel)))
		if err != nil {
			t.Fatalf("read prompt file %q: %v", promptRel, err)
		}
		if strings.Contains(string(promptRaw), "Custom claude collect-context.") {
			foundPackContent = true
		}
	}
	if !foundRetryAttempt {
		t.Fatalf("expected parse-retry prompt artifact metadata")
	}
	if !foundPackContent {
		t.Fatalf("expected custom workspace prompt pack content in captured claude prompt artifacts")
	}
}

func TestHeadlessRunnerNativeDirectClaudeCapturesFindingsPromptPackInPromptArtifacts(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	workspace := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(filepath.Join(workspace, "skills", "prompt-packs"), 0o755); err != nil {
		t.Fatalf("mkdir prompt-pack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "skills", "prompt-packs", "findings.md"), []byte("Custom claude findings pack.\n"), 0o644); err != nil {
		t.Fatalf("write custom findings prompt pack: %v", err)
	}

	commandPath := filepath.Join(tempDir, "claude")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY MODE"; then
  cat <<'JSON'
{"result":"{\"meta\":{\"task_id\":\"task-claude-findings-prompt-artifacts\",\"step_id\":\"refresh.step3.findings\",\"runtime\":{\"name\":\"claude-code\",\"version\":\"claude-cli\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}
JSON
  exit 0
fi
echo "This is not JSON"
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude findings prompt-artifacts command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	if _, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-claude-findings-prompt-artifacts",
		RunID:        "run-1",
		StepID:       "refresh.step3.findings",
		Workspace:    workspace,
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("expected claude findings run with retry prompt artifacts to succeed: %v", err)
	}

	metaFiles, err := filepath.Glob(filepath.Join(workspace, "reports", "taskruns", "raw", "*-prompt-meta.json"))
	if err != nil {
		t.Fatalf("glob prompt artifact metadata: %v", err)
	}
	if len(metaFiles) != 2 {
		t.Fatalf("expected initial + retry prompt metadata files, got %d", len(metaFiles))
	}
	foundPackContent := false
	foundRetryAttempt := false
	for _, metaPath := range metaFiles {
		rawMeta, readErr := os.ReadFile(metaPath)
		if readErr != nil {
			t.Fatalf("read prompt metadata %q: %v", metaPath, readErr)
		}
		meta := map[string]any{}
		if err := json.Unmarshal(rawMeta, &meta); err != nil {
			t.Fatalf("parse prompt metadata %q: %v", metaPath, err)
		}
		if strings.TrimSpace(meta["attempt"].(string)) == "parse-retry" {
			foundRetryAttempt = true
		}
		promptBlock, ok := meta["prompt"].(map[string]any)
		if !ok {
			t.Fatalf("metadata %q missing prompt block", metaPath)
		}
		promptRel := strings.TrimSpace(promptBlock["relative_path"].(string))
		promptRaw, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(promptRel)))
		if err != nil {
			t.Fatalf("read prompt file %q: %v", promptRel, err)
		}
		if strings.Contains(string(promptRaw), "Custom claude findings pack.") {
			foundPackContent = true
		}
	}
	if !foundRetryAttempt {
		t.Fatalf("expected parse-retry prompt artifact metadata")
	}
	if !foundPackContent {
		t.Fatalf("expected custom findings prompt pack content in captured claude prompt artifacts")
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

func TestHeadlessRunnerBindingMismatchClassifiesAsParseFailed(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`cat >/dev/null; echo '{"meta":{"task_id":"task-stale","step_id":"init.step1.collect","run_id":"run-stale","runtime":{"name":"claude-code","version":"stale-meta"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}'`,
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
		t.Fatalf("expected parse_stage=binding in error message, got %q", message)
	}
}

func TestHeadlessRunnerRejectsDeprecatedEdgeAliasesOnSchemaValidation(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`cat >/dev/null; echo '{"meta":{"task_id":"task-edge-repair","step_id":"refresh.step3.findings","run_id":"run-1","runtime":{"name":"claude-code","version":"claude-cli"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[{"op":"upsert_edge","edge":{"id":"edge.a.b","kind":"depends_on","source":"svc.a","target":"svc.b","provenance":{"kind":"inference","confidence":0.7,"evidence":[{"repo":"repo-a","path":"README.md"}]}}}]}'`,
		},
	}

	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-edge-repair",
		RunID:        "run-1",
		StepID:       "refresh.step3.findings",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"repo-a", "repo-b"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected parse-failed error for deprecated edge aliases")
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

func TestBuildDirectPromptIncludesStepSpecificPolicies(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-refresh",
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

	prompt := buildDirectPrompt(raw, false, false)
	if !strings.Contains(prompt, "STEP POLICY refresh.step1.collect") {
		t.Fatalf("expected step1 policy block in prompt")
	}
	if !strings.Contains(prompt, "Forbidden placeholder entity types: runtime_provider, runtime, metadata.") {
		t.Fatalf("expected forbidden runtime placeholder policy in prompt")
	}
	if !strings.Contains(prompt, "do NOT perform web search or external browsing") {
		t.Fatalf("expected explicit web-search prohibition in step1 policy")
	}
	if !strings.Contains(prompt, "Do NOT emit synthetic evidence paths such as search_source/*, search_query/*, search_config/*.") {
		t.Fatalf("expected explicit synthetic evidence-path prohibition in step1 policy")
	}
	if !strings.Contains(prompt, "question IDs MUST use canonical form without numeric suffixes") {
		t.Fatalf("expected canonical question-id policy in prompt")
	}
}

func TestBuildDirectPromptRefreshStep3IncludesCanonicalEdgeContract(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-refresh-step3",
		RunID:        "run-1",
		StepID:       "refresh.step3.findings",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service", "users-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildDirectPrompt(raw, false, false)
	if !strings.Contains(prompt, "For upsert_edge use canonical keys only: edge.id, edge.type, edge.from, edge.to.") {
		t.Fatalf("expected canonical upsert_edge key policy in prompt")
	}
	if !strings.Contains(prompt, "Forbidden edge aliases: edge.kind, edge.source, edge.target.") {
		t.Fatalf("expected forbidden edge alias policy in prompt")
	}
}

func TestBuildDirectPromptCollectIncludesArtifactQualityGuardrails(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-refresh-collect-guardrails",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"openstack"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildDirectPrompt(raw, false, false)
	if !strings.Contains(prompt, `Do NOT collapse a multi-document refresh surface to one generic "cite.runtime-summary" citation when repository evidence exists.`) {
		t.Fatalf("expected collect artifact-quality guardrail in prompt")
	}
	if !strings.Contains(prompt, "Preserve repo-specific citations in shard-pack-manifest.json whenever repository files support them.") {
		t.Fatalf("expected repo-specific citation guardrail in prompt")
	}
}

func TestBuildDirectPromptRetryIncludesArtifactQualityGuardrails(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-refresh-retry-guardrails",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"openstack"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildDirectPrompt(raw, true, true)
	if !strings.Contains(prompt, `Do NOT collapse a multi-document refresh into one generic "cite.runtime-summary" citation.`) {
		t.Fatalf("expected retry runtime-summary collapse ban in prompt")
	}
	if !strings.Contains(prompt, "Do NOT overwrite a rich shard-pack-manifest.json with a skeletal reuse-only manifest.") {
		t.Fatalf("expected retry rich-manifest preservation rule in prompt")
	}
	if !strings.Contains(prompt, "Preserve repo-specific citations when repository evidence already exists or can be recovered from repo roots.") {
		t.Fatalf("expected retry repo-specific citation rule in prompt")
	}
}

func TestBuildDirectPromptArtifactRepairModeIncludesArtifactFidelityGuardrails(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-refresh-artifact-repair",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"openstack"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildDirectPromptWithMode(raw, promptRetryArtifact, false)
	if !strings.Contains(prompt, "ARTIFACT REPAIR MODE") {
		t.Fatalf("expected artifact repair mode banner in prompt")
	}
	if !strings.Contains(prompt, "Repair artifact fidelity before returning JSON; this retry is not a fresh repository rediscovery pass.") {
		t.Fatalf("expected artifact fidelity repair guidance in prompt")
	}
	if !strings.Contains(prompt, `Do NOT use documents[].citations; only documents[].citation_ids is allowed.`) {
		t.Fatalf("expected canonical manifest field guardrail in artifact repair prompt")
	}
}

func TestHeadlessRunnerRepairsSkeletalCollectArtifactsAfterSchemaValidRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "openstack")
	writeRoot := filepath.Join(root, "write-root")
	for _, dir := range []string{workspace, repoPath, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: openstack\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	stateFile := filepath.Join(root, "claude-repair-count.txt")
	commandPath := filepath.Join(root, "claude")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
mkdir -p %q
if [ "$count" -eq 1 ]; then
  cat <<'JSON' > %q/shard-pack-manifest.json
%s
JSON
  cat <<'EOF' > %q/shard-analysis.md
# Reused
EOF
  cat <<'JSON'
{"meta":{"task_id":"task-claude-artifact-repair","step_id":"refresh.step1.collect","runtime":{"name":"claude-code","version":"claude-cli"},"started_at":"2026-04-03T12:00:00Z"},"summary":"initial collect","changeset":[]}
JSON
  exit 0
fi
cat <<'JSON' > %q/shard-pack-manifest.json
%s
JSON
cat <<'EOF' > %q/architecture-overview.md
# Architecture
EOF
cat <<'JSON'
{"meta":{"task_id":"task-claude-artifact-repair","step_id":"refresh.step1.collect","runtime":{"name":"claude-code","version":"claude-cli"},"started_at":"2026-04-03T12:00:00Z"},"summary":"artifact repair succeeded","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, writeRoot, validRetrySkeletalManifestJSONForClaude(), writeRoot, writeRoot, validRetryRichManifestJSONForClaude(), writeRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude repair command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-claude-artifact-repair",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"openstack"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected claude artifact repair retry to succeed: %v", err)
	}
	if result.TaskResult.Summary != "artifact repair succeeded" {
		t.Fatalf("expected repaired result summary, got %q", result.TaskResult.Summary)
	}
	raw, err := os.ReadFile(filepath.Join(writeRoot, "shard-pack-manifest.json"))
	if err != nil {
		t.Fatalf("read repaired manifest: %v", err)
	}
	if !strings.Contains(string(raw), "cite.openstack.readme") {
		t.Fatalf("expected rich repo-specific citation after repair, got %q", string(raw))
	}
}

func TestHeadlessRunnerCanonicalizesRichManifestMissingMetadataWithoutRepairRetry(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "openstack")
	writeRoot := filepath.Join(root, "write-root")
	for _, dir := range []string{workspace, repoPath, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: openstack\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	stateFile := filepath.Join(root, "claude-metadata-normalize-count.txt")
	commandPath := filepath.Join(root, "claude")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
mkdir -p %q
cat <<'JSON' > %q/shard-pack-manifest.json
{
  "artifact_root": "reports/taskruns/run-1/staging/shards/openstack-api",
  "version": 1,
  "documents": [
    {
      "id": "doc.architecture",
      "kind": "analysis",
      "title": "OpenStack Architecture",
      "path": "architecture-overview.md",
      "canonical_path": "reports/as-is/openstack/architecture-overview.md",
      "topics": ["api", "architecture"],
      "citation_ids": ["cite.openstack.readme", "cite.openstack.api"]
    }
  ],
  "citations": [
    {
      "id": "cite.openstack.readme",
      "repo": "openstack",
      "path": "README.md",
      "claim_ids": ["claim.openstack.services"],
      "document_ids": ["doc.architecture"]
    },
    {
      "id": "cite.openstack.api",
      "repo": "openstack",
      "path": "api/openapi.yaml",
      "claim_ids": ["claim.openstack.api"],
      "document_ids": ["doc.architecture"]
    }
  ]
}
JSON
cat <<'EOF' > %q/architecture-overview.md
# Architecture
EOF
cat <<'JSON'
{"result":"{\"meta\":{\"task_id\":\"task-claude-manifest-metadata-normalize\",\"step_id\":\"refresh.step1.collect\",\"runtime\":{\"name\":\"claude-code\",\"version\":\"claude-cli\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"metadata normalized\",\"changeset\":[],\"coverage\":{\"observed\":[\"api\"],\"missing\":[\"owner mappings\"],\"notes\":[\"repo evidence preserved\"]}}"}
JSON
`, stateFile, stateFile, stateFile, writeRoot, writeRoot, writeRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude metadata normalize command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-claude-manifest-metadata-normalize",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		ShardID:      "openstack-api",
		DomainID:     "openstack",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/openstack-api",
		RepoScopes:   []string{"openstack"},
		PathScopes:   []string{"api"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected claude metadata canonicalization to succeed without repair retry: %v", err)
	}
	if result.TaskResult.Summary != "metadata normalized" {
		t.Fatalf("unexpected result summary %q", result.TaskResult.Summary)
	}
	stateRaw, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	if count := strings.TrimSpace(string(stateRaw)); count != "1" {
		t.Fatalf("expected exactly 1 runner invocation, got %q", count)
	}
	raw, err := os.ReadFile(filepath.Join(writeRoot, "shard-pack-manifest.json"))
	if err != nil {
		t.Fatalf("read canonicalized manifest: %v", err)
	}
	text := string(raw)
	for _, expected := range []string{`"run_id": "run-1"`, `"step_id": "refresh.step1.collect"`, `"shard_id": "openstack-api"`, `"agent_role": "shard-analyst"`, `"compatibility": {`} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected canonicalized manifest to contain %s, got %q", expected, text)
		}
	}
}

func TestHeadlessRunnerRejectsUnreadableCollectManifestAfterArtifactRepair(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "openstack")
	writeRoot := filepath.Join(root, "write-root")
	for _, dir := range []string{workspace, repoPath, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: openstack\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	stateFile := filepath.Join(root, "claude-unreadable-manifest-count.txt")
	commandPath := filepath.Join(root, "claude")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
mkdir -p %q
cat <<'JSON' > %q/shard-pack-manifest.json
%s
JSON
cat <<'JSON'
{"meta":{"task_id":"task-claude-unreadable-manifest","step_id":"refresh.step1.collect","runtime":{"name":"claude-code","version":"claude-cli"},"started_at":"2026-04-03T12:00:00Z"},"summary":"retry still unreadable","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, writeRoot, validRetryRichManifestJSONForClaude())
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude unreadable manifest command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-claude-unreadable-manifest",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"openstack"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected unreadable collect manifest to fail after one repair attempt")
	}
	code, _, ok := acpruntime.ClassifyError(err)
	if !ok || code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("expected runner_parse_failed classification, got code=%q err=%v", code, err)
	}
	stateRaw, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	if count := strings.TrimSpace(string(stateRaw)); count != "2" {
		t.Fatalf("expected exactly 2 runner invocations for artifact repair retry, got %q", count)
	}
}

func TestBuildDirectPromptIncludesCanonicalManifestSchemaGuardrails(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-claude-manifest-schema-guardrails",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    t.TempDir(),
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/openstack-api",
		RepoScopes:   []string{"openstack"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildDirectPrompt(raw, false, false)
	if !strings.Contains(prompt, `Do NOT use documents[].citations; only documents[].citation_ids is allowed.`) {
		t.Fatalf("expected canonical manifest field guardrail in prompt")
	}
	if !strings.Contains(prompt, `citations[].claim_ids MUST be globally unique across the assembled staged final set`) {
		t.Fatalf("expected global claim-id uniqueness guardrail in prompt")
	}
	if !strings.Contains(prompt, `compatibility MUST include coverage, questions, entities, edges, and findings`) {
		t.Fatalf("expected compatibility completeness guardrail in prompt")
	}
	if !strings.Contains(prompt, `documents[].path MUST be write_root/artifact_root-relative`) {
		t.Fatalf("expected documents[].path artifact_root-relative guardrail in prompt")
	}
	if !strings.Contains(prompt, `Do NOT use reports/taskruns/... staging paths as canonical_path.`) {
		t.Fatalf("expected canonical_path staging-path ban in prompt")
	}
}

func TestBuildDirectPromptRetryIncludesSchemaFailureHintsForInvalidChangesetOp(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-claude-invalid-op-retry",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		WriteRoot:    filepath.Join(t.TempDir(), "write-root"),
		RepoScopes:   []string{"course-discovery"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	parseErr := errors.New(`taskresult is invalid: taskresult.schema.json validation failed: jsonschema: '/changeset/0/op' does not validate`)
	prompt := buildDirectPromptWithModeAndHints(raw, promptRetryParse, false, buildParseRepairHints("schema", parseErr))
	if !strings.Contains(prompt, "Previous schema validation failure") {
		t.Fatalf("expected schema failure hint in claude retry prompt")
	}
	if !strings.Contains(prompt, "Unknown changeset[].op values are forbidden") {
		t.Fatalf("expected allowed-op whitelist in claude retry prompt")
	}
	if !strings.Contains(prompt, `For op="add_doc_artifact", the payload key MUST be "doc_artifact"; never use "artifact".`) {
		t.Fatalf("expected doc_artifact payload guardrail in claude retry prompt")
	}
	if !strings.Contains(prompt, "Do NOT use ad-hoc ops such as upsert_file") {
		t.Fatalf("expected upsert_file ban in claude retry prompt")
	}
}

func TestBuildDirectPromptRetryIncludesSharedPromptContractGuardrails(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-claude-shared-guardrails",
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

	prompt := buildDirectPromptWithMode(raw, promptRetryParse, false)
	for _, line := range promptcontract.SharedTaskResultContractLines() {
		if !strings.Contains(prompt, line) {
			t.Fatalf("expected shared taskresult guardrail in claude prompt: %q", line)
		}
	}
	for _, line := range promptcontract.SharedRetryGuardrailLines() {
		if !strings.Contains(prompt, line) {
			t.Fatalf("expected shared retry guardrail in claude prompt: %q", line)
		}
	}
}

func TestHeadlessRunnerReturnsRunnerStalledForSilentFindingsTask(t *testing.T) {
	previousTimeout := findingsIdleSilenceTimeout
	findingsIdleSilenceTimeout = 100 * time.Millisecond
	t.Cleanup(func() {
		findingsIdleSilenceTimeout = previousTimeout
	})

	runner := HeadlessRunner{
		Command: "sh",
		Args:    []string{"-c", "sleep 2"},
	}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-claude-stalled",
		RunID:        "run-1",
		StepID:       "refresh.step3.findings",
		Workspace:    t.TempDir(),
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected stalled runner error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classified runner error, got %v", err)
	}
	if code != string(acpruntime.ErrorCodeRunnerStalled) {
		t.Fatalf("expected runner_stalled code, got %q (%v)", code, err)
	}
	if !strings.Contains(message, "stalled") {
		t.Fatalf("expected stalled message, got %q", message)
	}
}

func TestHeadlessRunnerReturnsRunnerStalledWhenStallRetryReturnsInvalidTaskResult(t *testing.T) {
	previousTimeout := findingsIdleSilenceTimeout
	findingsIdleSilenceTimeout = 500 * time.Millisecond
	t.Cleanup(func() {
		findingsIdleSilenceTimeout = previousTimeout
	})

	root := t.TempDir()
	stateFile := filepath.Join(root, "claude-stall-retry-count.txt")
	commandPath := filepath.Join(root, "claude")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
if [ "$count" -eq 1 ]; then
  sleep 2
  exit 0
fi
printf '%%s\n' '{"meta":{"task_id":"bad-only"}}'
`, stateFile, stateFile, stateFile)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude stall-retry command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-claude-stalled-invalid-retry",
		RunID:        "run-1",
		StepID:       "refresh.step3.findings",
		Workspace:    t.TempDir(),
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected stalled runner error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classified runner error, got %v", err)
	}
	if code != string(acpruntime.ErrorCodeRunnerStalled) {
		t.Fatalf("expected runner_stalled code, got %q (%v)", code, err)
	}
	if !strings.Contains(message, "stalled after retry") {
		t.Fatalf("expected stall-retry message, got %q", message)
	}
	if !strings.Contains(message, "parse_stage=exec") && !strings.Contains(message, "invalid taskresult") {
		t.Fatalf("expected retry diagnostics in stall message, got %q", message)
	}
}

func TestHeadlessRunnerClassifiesProviderQuotaAsRunnerUnavailable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	writeRoot := filepath.Join(workspace, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write-root: %v", err)
	}

	stateFile := filepath.Join(root, "claude-quota-count.txt")
	commandPath := filepath.Join(root, "claude")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
cat <<'JSON'
[{"type":"assistant","message":{"content":[{"type":"text","text":"[API Error: 403 {\"error\":{\"type\":\"permission_error\",\"message\":\"usage limit reached for this billing cycle\"},\"type\":\"error\"}]"}]}}]
JSON
`, stateFile, stateFile, stateFile)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude quota command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-claude-quota",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected runner_unavailable for provider quota signal")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classified runtime error, got %v", err)
	}
	if code != string(acpruntime.ErrorCodeRunnerUnavailable) {
		t.Fatalf("expected runner_unavailable code, got %q (%v)", code, err)
	}
	if !strings.Contains(message, "quota_or_permission") {
		t.Fatalf("expected quota_or_permission subreason in error message, got %q", message)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "1" {
		t.Fatalf("expected one invocation without parse retry for provider availability error, got %q", count)
	}
}

func TestHeadlessRunnerSalvagesCollectWhenParseFailsButManifestIsReadable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "openstack")
	writeRoot := filepath.Join(root, "write-root")
	for _, dir := range []string{workspace, repoPath, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: openstack\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "architecture-overview.md"), []byte("# Architecture\n"), 0o644); err != nil {
		t.Fatalf("write collect doc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-pack-manifest.json"), []byte(validRetryRichManifestJSONForClaude()), 0o644); err != nil {
		t.Fatalf("write rich manifest: %v", err)
	}

	stateFile := filepath.Join(root, "claude-salvage-count.txt")
	commandPath := filepath.Join(root, "claude")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
cat <<'OUT'
[{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"tool chatter only, no taskresult"}]}}]
OUT
`, stateFile, stateFile, stateFile)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude salvage command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-claude-salvage",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		RepoScope:    "openstack",
		RepoScopes:   []string{"openstack"},
		PathScopes:   []string{"api"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected deterministic salvage from readable collect artifacts, got %v", err)
	}
	if !strings.Contains(result.TaskResult.Summary, "synthesized deterministic TaskResult") {
		t.Fatalf("expected synthesized summary, got %q", result.TaskResult.Summary)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "1" {
		t.Fatalf("expected salvage to avoid parse retry, got invocations=%q", count)
	}
}

func validTaskResultJSON(taskID string, stepID string, runtimeName string, runtimeVersion string) string {
	return fmt.Sprintf(`{"meta":{"task_id":"%s","step_id":"%s","runtime":{"name":"%s","version":"%s"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}`, taskID, stepID, runtimeName, runtimeVersion)
}

func validRetryRichManifestJSONForClaude() string {
	return `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "shard_id": "openstack-api",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/write-root",
  "repo_scopes": ["openstack"],
  "path_scopes": ["api"],
  "summary": "Collected openstack architecture evidence.",
  "documents": [
    {
      "id": "doc.architecture.overview",
      "kind": "report",
      "title": "Architecture Overview",
      "path": "architecture-overview.md",
      "canonical_path": "reports/as-is/architecture-overview.md",
      "topics": ["architecture"],
      "citation_ids": ["cite.openstack.readme"],
      "status": "staged"
    }
  ],
  "citations": [
    {
      "id": "cite.openstack.readme",
      "repo": "openstack",
      "path": "README.md",
      "claim_ids": ["claim.architecture.readme"],
      "document_ids": ["doc.architecture.overview"]
    }
  ],
  "compatibility": {
    "coverage": {
      "observed": ["architecture"],
      "missing": ["owner mappings"],
      "notes": ["repo evidence preserved"]
    },
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`
}

func validRetrySkeletalManifestJSONForClaude() string {
	return `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "shard_id": "openstack-api",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/write-root",
  "repo_scopes": ["openstack"],
  "path_scopes": ["api"],
  "documents": [
    {
      "id": "doc.reused.analysis",
      "kind": "report",
      "title": "Reused Analysis",
      "path": "shard-analysis.md",
      "canonical_path": "reports/as-is/shard-analysis.md",
      "topics": ["api"],
      "citation_ids": ["cite.runtime-summary"]
    }
  ],
  "citations": [
    {
      "id": "cite.runtime-summary",
      "repo": "openstack",
      "path": "README.md",
      "claim_ids": ["claim.runtime.summary"],
      "document_ids": ["doc.reused.analysis"]
    }
  ],
  "summary": "Reused existing shard artifacts.",
  "compatibility": {
    "coverage": {
      "observed": ["api"],
      "missing": ["owner mappings"],
      "notes": ["reused existing artifacts"]
    },
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
