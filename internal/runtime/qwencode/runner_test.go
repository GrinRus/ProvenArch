package qwencode

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/testutil"
)

func TestRunClassifiesProviderUnavailableWhenArtifactsAreMissingAfterSuccessfulProcess(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-provider-marker")
	runner := HeadlessRunner{Command: writeQwenScript(t, "#!/usr/bin/env bash\nset -eu\nprintf '%s\\n' 'API Error: 403 permission_error usage limit'\n")}

	_, err := runner.Run(context.Background(), task)
	if err == nil {
		t.Fatal("expected runner error")
	}

	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected runtime runner error, got %T: %v", err, err)
	}
	if got, want := runnerErr.Code, acpruntime.ErrorCodeRunnerUnavailable; got != want {
		t.Fatalf("expected %s, got %s (%v)", want, got, err)
	}
	if runnerErr.Stdout == "" || !containsAll(runnerErr.Stdout, []string{"403", "permission_error"}) {
		t.Fatalf("expected provider-unavailable markers in stdout, got %q", runnerErr.Stdout)
	}
}

func TestRunClassifiesSilentMissingArtifactsAsProviderUnavailable(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-silent-missing")
	runner := HeadlessRunner{Command: writeQwenScript(t, "#!/usr/bin/env bash\nset -eu\n")}

	_, err := runner.Run(context.Background(), task)
	if err == nil {
		t.Fatal("expected runner error")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected runtime runner error, got %T: %v", err, err)
	}
	if got, want := runnerErr.Code, acpruntime.ErrorCodeRunnerUnavailable; got != want {
		t.Fatalf("expected %s, got %s (%v)", want, got, err)
	}
}

func TestRunKeepsMalformedManifestAsRuntimeContractFailure(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-malformed")
	script := "#!/usr/bin/env bash\nset -eu\nmkdir -p " + shellQuote(task.WriteRoot) + "\nprintf '%s\\n' '{\"version\":1,\"run_id\":\"" + task.RunID + "\",\"step_id\":\"init.step0.constitution\",\"step_contract\":\"constitution\",\"agent_role\":\"architect\",\"manifest_version\":2,\"outputs\":[]}' > " + shellQuote(filepath.Join(task.WriteRoot, "constitution-draft.json")) + "\nprintf '%s\\n' 'API Error: 403 permission_error usage limit'\n"
	runner := HeadlessRunner{Command: writeQwenScript(t, script)}

	_, err := runner.Run(context.Background(), task)
	if err == nil {
		t.Fatal("expected runtime contract error")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected runtime runner error, got %T: %v", err, err)
	}
	if got, want := runnerErr.Code, acpruntime.ErrorCodeRuntimeContract; got != want {
		t.Fatalf("expected %s, got %s (%v)", want, got, err)
	}
}

func TestQwenAdapterMonitorsPreArtifactStallsForArtifactSteps(t *testing.T) {
	t.Parallel()

	policy := (qwenAdapter{}).ActivityPolicy(acpruntime.Task{StepID: "init.step1.collect"})
	if !policy.MonitorArtifacts || !policy.MonitorPreArtifact {
		t.Fatalf("expected collect artifact and pre-artifact monitoring, got %+v", policy)
	}
	if got, want := policy.PreArtifactStallWindow, 180*time.Second; got != want {
		t.Fatalf("expected qwen pre-artifact window %s, got %s", want, got)
	}

	policy = (qwenAdapter{}).ActivityPolicy(acpruntime.Task{StepID: "init.step0.constitution"})
	if !policy.MonitorArtifacts || !policy.MonitorPreArtifact {
		t.Fatalf("expected draft artifact and pre-artifact monitoring, got %+v", policy)
	}
	if got, want := policy.PreArtifactStallWindow, 180*time.Second; got != want {
		t.Fatalf("expected qwen draft pre-artifact window %s, got %s", want, got)
	}
}

func TestQwenAdapterKeepsArtifactMonitorWithCustomArgs(t *testing.T) {
	t.Parallel()

	policy := (qwenAdapter{runner: HeadlessRunner{Args: []string{"--prompt"}}}).ActivityPolicy(acpruntime.Task{StepID: "init.step1.collect"})
	if !policy.MonitorArtifacts || !policy.MonitorPreArtifact {
		t.Fatalf("custom args must not disable collect artifact monitoring, got %+v", policy)
	}
}

func TestQwenAdapterRetriesZeroOutputPreArtifactStallOnce(t *testing.T) {
	t.Parallel()

	policy := (qwenAdapter{}).RecoveryPolicy(acpruntime.Task{StepID: "init.step1.collect"})
	if !policy.RetryInvalidOrMissingArtifactsOnce {
		t.Fatalf("expected qwen missing-artifact retry policy, got %+v", policy)
	}
	if !policy.RetryZeroOutputPreArtifactStallOnce {
		t.Fatalf("expected qwen zero-output pre-artifact retry policy, got %+v", policy)
	}
	if !policy.ClassifySilentRetryExhaustionUnavailable {
		t.Fatalf("expected qwen exhausted silence to use runner_unavailable lane, got %+v", policy)
	}
}

func TestQwenCommandSpecUsesPromptOnlyWithoutTaskJSONStdin(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-command-spec")
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	task.ReadContextRoots = []string{task.Workspace, repoRoot}
	spec, err := (qwenAdapter{runner: HeadlessRunner{Command: "qwen-test"}}).CommandSpec(task)
	if err != nil {
		t.Fatalf("command spec: %v", err)
	}
	if got, want := spec.Command, "qwen-test"; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
	if spec.Stdin != nil {
		t.Fatalf("qwen default invocation must keep stdin empty; task context belongs in -p prompt")
	}
	args := strings.Join(spec.Args, "\n")
	for _, token := range []string{
		"--chat-recording",
		"--yolo",
		"--output-format",
		"stream-json",
		"--include-partial-messages",
		"-p",
		"Artifact-only contract:",
		task.WriteRoot,
		task.DraftFinalRoot,
		"read_context_roots",
		repoRoot,
	} {
		if !strings.Contains(args, token) {
			t.Fatalf("expected qwen args to contain %q, got %v", token, spec.Args)
		}
	}
}

func TestQwenCommandSpecAppendsPromptForCustomArgsWithoutTaskJSONStdin(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-custom-args")
	spec, err := (qwenAdapter{runner: HeadlessRunner{
		Command: "qwen-test",
		Args:    []string{"--chat-recording", "false", "--yolo"},
	}}).CommandSpec(task)
	if err != nil {
		t.Fatalf("command spec: %v", err)
	}
	if spec.Stdin != nil {
		t.Fatalf("qwen custom invocation must keep stdin empty; task context belongs in -p prompt")
	}
	args := strings.Join(spec.Args, "\n")
	for _, token := range []string{
		"--chat-recording",
		"--yolo",
		"--output-format",
		"stream-json",
		"--include-partial-messages",
		"-p",
		"Artifact-only contract:",
		task.WriteRoot,
		task.DraftFinalRoot,
	} {
		if !strings.Contains(args, token) {
			t.Fatalf("expected qwen custom args to contain %q, got %v", token, spec.Args)
		}
	}
}

func TestQwenCommandSpecRespectsCustomNonStreamOutputFormat(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-custom-output-format")
	spec, err := (qwenAdapter{runner: HeadlessRunner{
		Command: "qwen-test",
		Args:    []string{"--chat-recording", "false", "--output-format", "json"},
	}}).CommandSpec(task)
	if err != nil {
		t.Fatalf("command spec: %v", err)
	}

	args := strings.Join(spec.Args, "\n")
	for _, token := range []string{"--output-format", "json", "-p", "Artifact-only contract:"} {
		if !strings.Contains(args, token) {
			t.Fatalf("expected qwen custom args to contain %q, got %v", token, spec.Args)
		}
	}
	if strings.Contains(args, "stream-json") {
		t.Fatalf("custom output format should not be overwritten, got %v", spec.Args)
	}
	if strings.Contains(args, "--include-partial-messages") {
		t.Fatalf("partial messages should only be injected for stream-json output, got %v", spec.Args)
	}
}

func TestQwenCommandSpecNormalizesCustomPromptArgsToArtifactPrompt(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-custom-prompt")
	spec, err := (qwenAdapter{runner: HeadlessRunner{
		Command: "qwen-test",
		Args:    []string{"--chat-recording", "false", "--prompt", "custom prompt", "--yolo", "-p=second prompt"},
	}}).CommandSpec(task)
	if err != nil {
		t.Fatalf("command spec: %v", err)
	}
	if spec.Stdin != nil {
		t.Fatalf("qwen custom invocation must keep stdin empty; task context belongs in -p prompt")
	}
	args := strings.Join(spec.Args, "\n")
	for _, forbidden := range []string{"custom prompt", "second prompt"} {
		if strings.Contains(args, forbidden) {
			t.Fatalf("expected custom prompt %q to be replaced by artifact prompt, got %v", forbidden, spec.Args)
		}
	}
	for _, token := range []string{
		"--chat-recording",
		"--yolo",
		"--output-format",
		"stream-json",
		"--include-partial-messages",
		"--prompt",
		"Artifact-only contract:",
		task.WriteRoot,
		task.DraftFinalRoot,
	} {
		if !strings.Contains(args, token) {
			t.Fatalf("expected qwen custom args to contain %q, got %v", token, spec.Args)
		}
	}
	if got := strings.Count(args, "Artifact-only contract:"); got != 1 {
		t.Fatalf("expected exactly one artifact prompt, got %d in %v", got, spec.Args)
	}
}

func TestQwenRepairCommandSpecUsesPromptOnlyWithoutTaskJSONStdin(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "arch-workspace")
	repoRoot := filepath.Join(root, "repo-a")
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "shards", "repo-a")
	for _, dir := range []string{workspace, repoRoot, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte("version: 1\nrepos:\n  - name: repo-a\n    path: "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "overview.md"), []byte("# Repo A\n"), 0o644); err != nil {
		t.Fatalf("write authored doc: %v", err)
	}

	task := acpruntime.Task{
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        workspace,
		WriteRoot:        writeRoot,
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/repo-a",
		ReadContextRoots: []string{workspace, repoRoot},
		RepoScopes:       []string{"repo-a"},
		PathScopes:       []string{"README.md", "pom.xml"},
		ShardID:          "repo-a-root-files",
	}

	spec, err := (qwenAdapter{runner: HeadlessRunner{Command: "qwen-test"}}).CollectManifestRepairCommandSpec(task, os.ErrNotExist)
	if err != nil {
		t.Fatalf("repair command spec: %v", err)
	}
	if spec.Stdin != nil {
		t.Fatalf("qwen repair invocation must keep stdin empty; repair contract belongs in -p prompt")
	}
	if !containsArg(spec.IncludeDirs, writeRoot) || !containsArg(spec.IncludeDirs, repoRoot) {
		t.Fatalf("repair include dirs must keep write root and repo evidence, got %v", spec.IncludeDirs)
	}
	args := strings.Join(spec.Args, "\n")
	for _, token := range []string{
		"collect manifest repair mode",
		"Run the preferred file write command below",
		"TASK-SPECIFIC MANIFEST JSON SKELETON:",
		"<<'ACP_MANIFEST_JSON'",
		"Copy the heredoc JSON exactly during repair",
		"Do not read, diff, or patch an existing invalid shard-pack-manifest.json",
		"Final action must be: write only write_root/shard-pack-manifest.json",
		"overview.md",
		`"path": "overview.md"`,
	} {
		if !strings.Contains(args, token) {
			t.Fatalf("expected qwen repair prompt arg to contain %q, got %v", token, spec.Args)
		}
	}
}

func TestQwenCollectArtifactPairRepairCommandSpecUsesPromptOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "arch-workspace")
	repoRoot := filepath.Join(root, "repo-a")
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "shards", "repo-a-root-files")
	for _, dir := range []string{workspace, repoRoot, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte("version: 1\nrepos:\n  - name: repo-a\n    path: "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	task := acpruntime.Task{
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        workspace,
		WriteRoot:        writeRoot,
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/repo-a-root-files",
		ReadContextRoots: []string{workspace, repoRoot},
		RepoScopes:       []string{"repo-a"},
		PathScopes:       []string{"README.md", "pom.xml"},
		ShardID:          "repo-a-root-files",
	}

	spec, err := (qwenAdapter{runner: HeadlessRunner{Command: "qwen-test"}}).CollectArtifactPairRepairCommandSpec(task, os.ErrNotExist)
	if err != nil {
		t.Fatalf("collect pair repair command spec: %v", err)
	}
	if spec.Stdin != nil {
		t.Fatalf("qwen collect pair repair invocation must keep stdin empty; repair contract belongs in -p prompt")
	}
	if !containsArg(spec.IncludeDirs, writeRoot) || !containsArg(spec.IncludeDirs, repoRoot) {
		t.Fatalf("collect pair repair include dirs must keep write root and repo evidence, got %v", spec.IncludeDirs)
	}
	args := strings.Join(spec.Args, "\n")
	for _, token := range []string{
		"-p",
		"collect artifact pair focused recovery mode",
		"Run the exact shell command below as your next command. Do not inspect repository files first.",
		"COLLECT PAIR WRITE COMMAND:",
		"<<'ACP_COLLECT_DOC'",
		"<<'ACP_MANIFEST_JSON'",
		"root-overview.md",
		"shard-pack-manifest.json",
		`"path": "root-overview.md"`,
		"Previous collect artifact validation failure",
	} {
		if !strings.Contains(args, token) {
			t.Fatalf("expected qwen collect pair repair prompt arg to contain %q, got %v", token, spec.Args)
		}
	}
}

func TestQwenValidatorRepairCommandSpecUsesPromptOnly(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-validator-repair-spec")
	task.StepID = "init.step3.findings"
	task.WriteRoot = filepath.Join(task.Workspace, "reports", "taskruns", task.RunID, "validator")
	task.ReadContextRoots = []string{task.DraftFinalRoot}
	task.ExpectedArtifacts = []string{"validator-verdict.json"}
	if err := os.MkdirAll(task.WriteRoot, 0o755); err != nil {
		t.Fatalf("mkdir validator write root: %v", err)
	}

	spec, err := (qwenAdapter{runner: HeadlessRunner{Command: "qwen-test"}}).ValidatorVerdictRepairCommandSpec(task, os.ErrNotExist)
	if err != nil {
		t.Fatalf("validator repair command spec: %v", err)
	}
	if spec.Stdin != nil {
		t.Fatalf("qwen validator repair invocation must keep stdin empty; repair contract belongs in -p prompt")
	}
	if !containsArg(spec.IncludeDirs, task.WriteRoot) || !containsArg(spec.IncludeDirs, task.DraftFinalRoot) {
		t.Fatalf("validator repair include dirs must keep write root and staged final evidence, got %v", spec.IncludeDirs)
	}
	args := strings.Join(spec.Args, "\n")
	for _, token := range []string{
		"validator verdict focused recovery mode",
		"Immediate validator verdict repair action:",
		"VALIDATOR VERDICT WRITE COMMAND:",
		"<<'ACP_VALIDATOR_VERDICT_JSON'",
		"VALIDATOR VERDICT JSON SKELETON:",
		"issues[] items must use only: code, severity, message, path, document_id, citation_id",
		"Legacy issue fields are forbidden inside issues[]",
		`"issues": []`,
	} {
		if !strings.Contains(args, token) {
			t.Fatalf("expected qwen validator repair prompt arg to contain %q, got %v", token, spec.Args)
		}
	}
}

func TestQwenDraftRepairCommandSpecUsesPromptOnly(t *testing.T) {
	t.Parallel()

	task := newQwenDraftTask(t, "run-draft-repair-spec")
	spec, err := (qwenAdapter{runner: HeadlessRunner{Command: "qwen-test"}}).DraftArtifactRepairCommandSpec(task, os.ErrNotExist)
	if err != nil {
		t.Fatalf("draft repair command spec: %v", err)
	}
	if spec.Stdin != nil {
		t.Fatalf("qwen draft repair invocation must keep stdin empty; repair contract belongs in -p prompt")
	}
	if !containsArg(spec.IncludeDirs, task.WriteRoot) || !containsArg(spec.IncludeDirs, task.DraftFinalRoot) {
		t.Fatalf("draft repair include dirs must keep write/draft roots, got %v", spec.IncludeDirs)
	}
	args := strings.Join(spec.Args, "\n")
	for _, token := range []string{
		"draft artifact focused recovery mode",
		"Immediate draft artifact repair action:",
		"RUNTIME DRAFT MANIFEST JSON SKELETON:",
		"<<'ACP_DRAFT_MANIFEST_JSON'",
		"<<'ACP_DRAFT_FILE'",
		"Copy the heredoc artifacts exactly first",
		"overwrite them from the heredoc artifacts",
		"Write draft content only under draft_final_root",
		"Absolute target checks must use write_root/draft_final_root exactly",
	} {
		if !strings.Contains(args, token) {
			t.Fatalf("expected qwen draft repair prompt arg to contain %q, got %v", token, spec.Args)
		}
	}
}

func containsAll(text string, tokens []string) bool {
	for _, token := range tokens {
		if !strings.Contains(text, token) {
			return false
		}
	}
	return true
}

func newQwenDraftTask(t *testing.T, runID string) acpruntime.Task {
	t.Helper()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", runID, "constitution")
	draftRoot := filepath.Join(workspace, "reports", "taskruns", runID, "staging", "final")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	return acpruntime.Task{
		TaskID:            "task-" + runID,
		RunID:             runID,
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		Workspace:         workspace,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"constitution-draft.json", "charter-overview.md", "baseline-subagents.yaml"},
		StartedAtUTC:      time.Now().UTC(),
	}
}

func writeQwenScript(t *testing.T, script string) string {
	t.Helper()
	return testutil.WriteExecutableScript(t, "qwen-stub.sh", script)
}

func qwenValidDraftScript(task acpruntime.Task, tail string) string {
	return `#!/usr/bin/env bash
set -eu
write_root=` + shellQuote(task.WriteRoot) + `
draft_root=` + shellQuote(task.DraftFinalRoot) + `
run_id=` + shellQuote(task.RunID) + `
mkdir -p "$write_root" "$draft_root"
cat >"$write_root/constitution-draft.json" <<EOF
{
  "version": 1,
  "run_id": "$run_id",
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "summary": "Recovered constitution artifacts.",
  "outputs": [
    {
      "path": "charter-overview.md",
      "canonical_path": "charter/overview.md",
      "kind": "charter",
      "title": "Constitution"
    },
    {
      "path": "baseline-subagents.yaml",
      "canonical_path": "skills/subagents.yaml",
      "kind": "bundle",
      "title": "Baseline Subagents"
    }
  ]
}
EOF
cat >"$draft_root/charter-overview.md" <<'EOF'
# Constitution
EOF
cat >"$draft_root/baseline-subagents.yaml" <<'EOF'
version: 1
EOF
` + tail + "\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
