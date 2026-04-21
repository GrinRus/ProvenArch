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
	prompt := buildDirectPromptWithModeAndHints(raw, promptRetryParse, false, buildParseRepairHints(task.StepID, "schema", parseErr))
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

func TestBuildDirectPromptCollectUsesExistingEntrypointHintsWithoutSyntheticTemplate(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	repoRoot := filepath.Join(workspaceDir, "service-repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	packageJSONPath := filepath.Join(repoRoot, "package.json")
	if err := os.WriteFile(packageJSONPath, []byte("{\"name\":\"service-repo\"}\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	task := acpruntime.Task{
		TaskID:           "task-claude-init-step1",
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        workspaceDir,
		ReadContextRoots: []string{repoRoot},
		RepoScopes:       []string{"service-repo"},
		StartedAtUTC:     time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildDirectPrompt(raw, false, false)
	if !strings.Contains(prompt, "Do NOT delegate to agent/subagent helpers.") {
		t.Fatalf("expected delegation ban in collect prompt")
	}
	if strings.Contains(prompt, `"op": "upsert_entity"`) {
		t.Fatalf("did not expect synthetic upsert_entity template in collect prompt")
	}
	if !strings.Contains(prompt, filepath.ToSlash(packageJSONPath)) {
		t.Fatalf("expected existing package.json entrypoint hint in prompt")
	}
	if strings.Contains(prompt, filepath.ToSlash(filepath.Join(repoRoot, "README.md"))) {
		t.Fatalf("did not expect non-existent README.md entrypoint hint in prompt")
	}
}

func TestBuildDirectPromptConstitutionIncludesExactDraftContract(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	repoRoot := filepath.Join(workspaceDir, "service-repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	packageJSONPath := filepath.Join(repoRoot, "package.json")
	if err := os.WriteFile(packageJSONPath, []byte("{\"name\":\"service-repo\"}\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	task := acpruntime.Task{
		TaskID:           "task-claude-init-step0",
		RunID:            "run-1",
		StepID:           "init.step0.constitution",
		Workspace:        workspaceDir,
		ReadContextRoots: []string{repoRoot},
		RepoScopes:       []string{"service-repo"},
		StartedAtUTC:     time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildDirectPrompt(raw, false, false)
	if !strings.Contains(prompt, "STEP POLICY init.step0.constitution:") {
		t.Fatalf("expected step0 policy section in prompt")
	}
	if !strings.Contains(prompt, "Write constitution-draft.json in write_root.") {
		t.Fatalf("expected constitution draft manifest instruction in prompt")
	}
	if !strings.Contains(prompt, `"step_contract": "constitution"`) {
		t.Fatalf("expected exact constitution draft manifest example in prompt")
	}
	if !strings.Contains(prompt, `"canonical_path": "charter/overview.md"`) || !strings.Contains(prompt, `"canonical_path": "skills/subagents.yaml"`) {
		t.Fatalf("expected exact constitution outputs mapping in prompt")
	}
	if strings.Contains(prompt, `"op": "upsert_entity"`) {
		t.Fatalf("did not expect synthetic upsert_entity template for draft-only step0")
	}
	if !strings.Contains(prompt, filepath.ToSlash(packageJSONPath)) {
		t.Fatalf("expected existing package.json entrypoint hint in prompt")
	}
	if strings.Contains(prompt, filepath.ToSlash(filepath.Join(repoRoot, "README.md"))) {
		t.Fatalf("did not expect non-existent README.md entrypoint hint in prompt")
	}
}

func TestBuildDirectPromptConstitutionRetryUsesDraftArtifactHints(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	writeRoot := filepath.Join(workspaceDir, "write-root")
	draftRoot := filepath.Join(workspaceDir, "draft-root")
	task := acpruntime.Task{
		TaskID:            "task-claude-step0-repair",
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		Workspace:         workspaceDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		StepContract:      "constitution",
		ExpectedArtifacts: []string{"constitution-draft.json"},
		StartedAtUTC:      time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildDirectPromptWithModeAndHints(
		raw,
		promptRetryDraftArtifact,
		false,
		buildDraftArtifactRepairHints(task, errors.New("runtime draft manifest version must be 1")),
	)
	if !strings.Contains(prompt, "DRAFT ARTIFACT REPAIR MODE") {
		t.Fatalf("expected draft artifact repair banner in prompt")
	}
	if !strings.Contains(prompt, "constitution-draft.json") {
		t.Fatalf("expected constitution draft recovery guidance in prompt")
	}
	if !strings.Contains(prompt, `"step_contract": "constitution"`) {
		t.Fatalf("expected exact constitution manifest example in retry prompt")
	}
	if strings.Contains(prompt, "shard-pack-manifest.json") {
		t.Fatalf("did not expect collect-specific shard wording in step0 draft retry prompt")
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
