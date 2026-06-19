package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--help"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stdout.String(), "acp serve [--workspace <abs-path>]") {
		t.Fatalf("expected help output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "acp doctor") {
		t.Fatalf("expected doctor command in help output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "acp version") {
		t.Fatalf("expected version command in help output, got %q", stdout.String())
	}
}

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"version"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"acp version ",
		"commit: ",
		"built: ",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected version output to contain %q, got %q", want, output)
		}
	}
}

func TestVersionFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--version"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stdout.String(), "acp version ") {
		t.Fatalf("expected version output, got %q", stdout.String())
	}
}

func TestServeHelpReturnsZero(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"serve", "--help"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stderr.String(), "Usage: acp serve [--workspace <abs-path>]") {
		t.Fatalf("expected serve usage in stderr, got %q", stderr.String())
	}
}

func TestInitWorkspaceHelpReturnsZero(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"init-workspace", "--help"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stderr.String(), "Usage: acp init-workspace --workspace <abs-path> ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>)") {
		t.Fatalf("expected init-workspace usage in stderr, got %q", stderr.String())
	}
}

func TestRunSubcommandHelpReturnsZero(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"run", "--help"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stderr.String(), "Usage: acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]") {
		t.Fatalf("expected run usage in stderr, got %q", stderr.String())
	}
}

func TestQAHelpReturnsZero(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"qa", "--help"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stderr.String(), "Usage: acp qa --workspace <abs-path> --question") {
		t.Fatalf("expected qa usage in stderr, got %q", stderr.String())
	}
}

func TestDoctorHelpReturnsZero(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"doctor", "--help"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stderr.String(), "Usage: acp doctor") {
		t.Fatalf("expected doctor usage in stderr, got %q", stderr.String())
	}
}

func TestDoctorReportsUserFixableIssuesAsExitOne(t *testing.T) {
	t.Setenv("ACP_CODEX_CMD", "definitely-missing-acp-doctor-command")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{
		"doctor",
		"--workspace", t.TempDir(),
		"--runtime", "headless",
		"--runtime-provider", "codex-code",
		"--listen", "127.0.0.1:0",
		"--json",
	}, &stdout, &stderr)
	if code != exitCodeDoctorIssues {
		t.Fatalf("expected exit code %d, got %d stderr=%q stdout=%q", exitCodeDoctorIssues, code, stderr.String(), stdout.String())
	}
	var payload struct {
		OK     bool `json:"ok"`
		Checks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode doctor json: %v\n%s", err, stdout.String())
	}
	if payload.OK {
		t.Fatalf("expected ok=false")
	}
	if !hasDoctorCLIResultCheck(payload.Checks, "runtime_provider", "fail") {
		t.Fatalf("expected runtime_provider failure, got %+v", payload.Checks)
	}
}

func TestDoctorTextOutputReportsReadyChecks(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{
		"doctor",
		"--workspace", t.TempDir(),
		"--runtime", "fake",
		"--listen", "127.0.0.1:0",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d stderr=%q stdout=%q", exitCodeOK, code, stderr.String(), stdout.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"ACP doctor",
		"status: ready",
		"[pass] Git",
		"[pass] Embedded UI",
		"[warn] Repository source",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected doctor text output to contain %q, got %q", want, output)
		}
	}
}

func TestDoctorRejectsAmbiguousRepoSources(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{
		"doctor",
		"--repo-path", "/tmp/repo",
		"--repo-git-url", "https://github.com/org/repo.git",
	}, &stdout, &stderr)
	if code != exitCodeInvalidCommand {
		t.Fatalf("expected exit code %d, got %d", exitCodeInvalidCommand, code)
	}
	if !strings.Contains(stderr.String(), "set at most one") {
		t.Fatalf("expected ambiguous source error, got %q", stderr.String())
	}
}

func TestInitWorkspaceRequiresWorkspace(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"init-workspace", "--repo-name", "sample", "--repo-path", "/tmp/repo"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "--workspace is required") {
		t.Fatalf("expected workspace validation error, got %q", stderr.String())
	}
}

func TestInitWorkspaceCreatesManifestAndLayout(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "arch-workspace")
	repoPath := filepath.Join(root, "repos", "sample")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create sample repo path: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"init-workspace",
		"--workspace", workspaceRoot,
		"--repo-name", "sample",
		"--repo-path", repoPath,
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	manifestContent, err := os.ReadFile(filepath.Join(workspaceRoot, "workspace.yaml"))
	if err != nil {
		t.Fatalf("read workspace manifest: %v", err)
	}
	manifestText := string(manifestContent)
	if !strings.Contains(manifestText, "name: sample") {
		t.Fatalf("expected sample repo name in manifest, got %q", manifestText)
	}
	if !strings.Contains(manifestText, "path: "+repoPath) {
		t.Fatalf("expected repo path in manifest, got %q", manifestText)
	}

	requiredDirs := []string{
		"charter/cards/domains",
		"skills",
		"reports/as-is/services",
		"docs/imports",
	}
	for _, rel := range requiredDirs {
		info, statErr := os.Stat(filepath.Join(workspaceRoot, rel))
		if statErr != nil {
			t.Fatalf("expected directory %s to exist: %v", rel, statErr)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be directory", rel)
		}
	}

	if !strings.Contains(stdout.String(), "next commands:") {
		t.Fatalf("expected next commands guidance, got %q", stdout.String())
	}

	if info, err := os.Stat(filepath.Join(workspaceRoot, ".git")); err != nil || !info.IsDir() {
		t.Fatalf("expected workspace git repo to be initialized: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceRoot, "skills", "subagents.yaml")); err != nil {
		t.Fatalf("expected baseline bundle to be seeded: %v", err)
	}
}

func TestInitWorkspaceRejectsExistingManifestWithoutForce(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "sample")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create sample repo path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("version: 1\nrepos:\n  - name: sample\n    path: /tmp/sample\n"), 0o644); err != nil {
		t.Fatalf("seed workspace manifest: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"init-workspace",
		"--workspace", root,
		"--repo-name", "sample",
		"--repo-path", repoPath,
	}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("expected overwrite guidance, got %q", stderr.String())
	}
}

func TestQARequiresQuestion(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"qa", "--workspace", root}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "--question is required") {
		t.Fatalf("expected qa validation error, got %q", stderr.String())
	}
}

func TestQACommandOutputsAnswer(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"qa", "--workspace", root, "--question", "who owns sample service?"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stdout.String(), "answer:") {
		t.Fatalf("expected answer output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "confidence:") {
		t.Fatalf("expected confidence output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestServeRejectsUnsupportedRuntime(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"serve", "--workspace", root, "--runtime", "bogus", "--dry-run"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "unsupported runtime") {
		t.Fatalf("expected runtime validation error, got %q", stderr.String())
	}
}

func TestServeRejectsUnsupportedRuntimeProvider(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"serve", "--workspace", root, "--runtime-provider", "bogus", "--dry-run"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "unsupported runtime provider") {
		t.Fatalf("expected runtime provider validation error, got %q", stderr.String())
	}
}

func TestServeRejectsInvalidRunLogsRetentionFlags(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"serve", "--workspace", root, "--runtime", "fake", "--dry-run", "--run-logs-ttl-hours", "0"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "--run-logs-ttl-hours must be > 0") {
		t.Fatalf("expected run logs ttl validation error, got %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"serve", "--workspace", root, "--runtime", "fake", "--dry-run", "--run-logs-max-runs", "0"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "--run-logs-max-runs must be > 0") {
		t.Fatalf("expected run logs max-runs validation error, got %q", stderr.String())
	}
}

func TestRunRejectsUnsupportedRuntime(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"run", "--workspace", root, "--pipeline", "init", "--runtime", "bogus", "--non-interactive"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "unsupported runtime") {
		t.Fatalf("expected runtime validation error, got %q", stderr.String())
	}
}

func TestRunRejectsUnsupportedRuntimeProvider(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"run", "--workspace", root, "--pipeline", "init", "--runtime-provider", "bogus", "--non-interactive"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "unsupported runtime provider") {
		t.Fatalf("expected runtime provider validation error, got %q", stderr.String())
	}
}

func TestRunRejectsInvalidRunLogsRetentionFlags(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{
		"run",
		"--workspace", root,
		"--pipeline", "init",
		"--runtime", "fake",
		"--non-interactive",
		"--run-logs-ttl-hours", "0",
	}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "--run-logs-ttl-hours must be > 0") {
		t.Fatalf("expected run logs ttl validation error, got %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{
		"run",
		"--workspace", root,
		"--pipeline", "init",
		"--runtime", "fake",
		"--non-interactive",
		"--run-logs-max-runs", "0",
	}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "--run-logs-max-runs must be > 0") {
		t.Fatalf("expected run logs max-runs validation error, got %q", stderr.String())
	}
}

func TestRunPipelineRejectsInvalidPipeline(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"run", "--workspace", root, "--pipeline", "bogus"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "unsupported pipeline") {
		t.Fatalf("expected pipeline validation error, got %q", stderr.String())
	}
}

func TestRunPipelineBootstrapSkeleton(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"run", "--workspace", root, "--pipeline", "refresh", "--non-interactive"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stdout.String(), "status: succeeded") {
		t.Fatalf("expected successful run output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime mode: fake") {
		t.Fatalf("expected runtime mode output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider: claude-code") {
		t.Fatalf("expected runtime provider output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider note: ignored in fake mode") {
		t.Fatalf("expected fake runtime provider note, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunReconcilesStaleRunningHistoryBeforeNewRun(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	historyPath := filepath.Join(root, "reports", "taskruns", "run-history.json")
	if err := os.MkdirAll(filepath.Dir(historyPath), 0o755); err != nil {
		t.Fatalf("mkdir run history dir: %v", err)
	}
	historyPayload := map[string]any{
		"version": 1,
		"items": []map[string]any{
			{
				"run_id":       "run_stale_running",
				"pipeline":     "refresh",
				"status":       "running",
				"started_at":   "2026-04-19T12:00:00Z",
				"current_step": "refresh.step1.collect",
			},
		},
	}
	rawHistory, err := json.Marshal(historyPayload)
	if err != nil {
		t.Fatalf("marshal run history: %v", err)
	}
	if err := os.WriteFile(historyPath, rawHistory, 0o644); err != nil {
		t.Fatalf("write run history: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"run", "--workspace", root, "--pipeline", "init", "--runtime", "fake", "--non-interactive"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}

	content, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("read updated run history: %v", err)
	}
	var snapshot struct {
		Items []struct {
			RunID     string `json:"run_id"`
			Status    string `json:"status"`
			ErrorCode string `json:"error_code"`
		} `json:"items"`
	}
	if err := json.Unmarshal(content, &snapshot); err != nil {
		t.Fatalf("decode updated run history: %v", err)
	}
	foundReconciled := false
	foundSucceeded := false
	for _, item := range snapshot.Items {
		if item.RunID == "run_stale_running" {
			foundReconciled = item.Status == "failed" && item.ErrorCode == "run_reconciled_after_restart"
		}
		if strings.HasPrefix(item.RunID, "run_") && item.RunID != "run_stale_running" && item.Status == "succeeded" {
			foundSucceeded = true
		}
	}
	if !foundReconciled {
		t.Fatalf("expected stale running run to be reconciled before new run, history=%s", string(content))
	}
	if !foundSucceeded {
		t.Fatalf("expected new run to succeed after reconciliation, history=%s", string(content))
	}
}

func TestServeBootstrapSkeleton(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"serve", "--workspace", root, "--dry-run"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stdout.String(), "workspace ready") {
		t.Fatalf("expected validation output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "server configured") {
		t.Fatalf("expected dry-run server output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime mode: fake") {
		t.Fatalf("expected runtime mode output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider: claude-code") {
		t.Fatalf("expected runtime provider output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider note: ignored in fake mode") {
		t.Fatalf("expected fake runtime provider note, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestServeDryRunWithoutWorkspaceStartsLauncher(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"serve", "--dry-run"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "launcher ready") {
		t.Fatalf("expected launcher dry-run output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime mode: fake") {
		t.Fatalf("expected runtime mode output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestServeLauncherRejectsAutoInitFlagsWithoutWorkspace(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"serve", "--auto-init", "--repo-name", "sample", "--repo-path", "/tmp/sample", "--dry-run"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "--auto-init and repo flags require --workspace") {
		t.Fatalf("expected auto-init workspace error, got %q", stderr.String())
	}
}

func TestServeDryRunUsesRuntimeProviderFromEnv(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_RUNTIME_PROVIDER", "qwen-code")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"serve", "--workspace", root, "--runtime", "fake", "--dry-run"}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stdout.String(), "runtime provider: qwen-code") {
		t.Fatalf("expected runtime provider from env, got %q", stdout.String())
	}
}

func TestServeWithoutAutoInitFailsWhenWorkspaceMissing(t *testing.T) {
	t.Parallel()

	workspacePath := filepath.Join(t.TempDir(), "missing-workspace")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"serve", "--workspace", workspacePath, "--dry-run"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "stat workspace") {
		t.Fatalf("expected missing workspace error, got %q", stderr.String())
	}
}

func TestServeAutoInitCreatesWorkspaceOnDryRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspacePath := filepath.Join(root, "arch-workspace")
	repoPath := filepath.Join(root, "repos", "sample")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create sample repo path: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"serve",
		"--workspace", workspacePath,
		"--auto-init",
		"--repo-name", "sample",
		"--repo-path", repoPath,
		"--dry-run",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(workspacePath, "workspace.yaml")); err != nil {
		t.Fatalf("expected workspace.yaml after auto-init: %v", err)
	}
	if info, err := os.Stat(filepath.Join(workspacePath, ".git")); err != nil || !info.IsDir() {
		t.Fatalf("expected workspace git repo to be initialized: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, "skills", "subagents.yaml")); err != nil {
		t.Fatalf("expected baseline bundle to be seeded on auto-init: %v", err)
	}
	if !strings.Contains(stdout.String(), "workspace ready") {
		t.Fatalf("expected workspace ready output, got %q", stdout.String())
	}
}

func TestServeAutoInitRequiresSingleRepoSourceWhenWorkspaceMissing(t *testing.T) {
	t.Parallel()

	workspacePath := filepath.Join(t.TempDir(), "arch-workspace")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"serve",
		"--workspace", workspacePath,
		"--auto-init",
		"--repo-name", "sample",
		"--dry-run",
	}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "set exactly one of --repo-path or --repo-git-url") && !strings.Contains(stderr.String(), "set either --repos-file or single-repo flags") {
		t.Fatalf("expected repo source validation error, got %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{
		"serve",
		"--workspace", workspacePath,
		"--auto-init",
		"--repo-name", "sample",
		"--repo-path", "/tmp/a",
		"--repo-git-url", "https://example.invalid/repo.git",
		"--dry-run",
	}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "set exactly one of --repo-path or --repo-git-url") && !strings.Contains(stderr.String(), "set either --repos-file or single-repo flags") {
		t.Fatalf("expected mutually-exclusive source validation error, got %q", stderr.String())
	}
}

func TestInitWorkspaceSupportsReposFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "arch-workspace")
	repoA := filepath.Join(root, "repos", "a")
	repoB := filepath.Join(root, "repos", "b")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatalf("create repoA: %v", err)
	}
	if err := os.MkdirAll(repoB, 0o755); err != nil {
		t.Fatalf("create repoB: %v", err)
	}
	reposFile := filepath.Join(root, "repos.yaml")
	reposContent := `repos:
  - name: repo-a
    path: ` + repoA + `
  - name: repo-b
    path: ` + repoB + `
`
	if err := os.WriteFile(reposFile, []byte(reposContent), 0o644); err != nil {
		t.Fatalf("write repos file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"init-workspace",
		"--workspace", workspaceRoot,
		"--repos-file", reposFile,
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	manifestContent, err := os.ReadFile(filepath.Join(workspaceRoot, "workspace.yaml"))
	if err != nil {
		t.Fatalf("read workspace manifest: %v", err)
	}
	manifestText := string(manifestContent)
	if !strings.Contains(manifestText, "name: repo-a") || !strings.Contains(manifestText, "name: repo-b") {
		t.Fatalf("expected repos from repos-file in manifest, got %q", manifestText)
	}
}

func TestInitWorkspaceReposFilePreservesRuntimeTimeouts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "arch-workspace")
	repoA := filepath.Join(root, "repos", "a")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatalf("create repoA: %v", err)
	}
	reposFile := filepath.Join(root, "repos.yaml")
	reposContent := `repos:
  - name: repo-a
    path: ` + repoA + `
runtime:
  profile:
    timeouts:
      step_timeout_sec: 777
      ui_cancel_poll_timeout_sec: 333
`
	if err := os.WriteFile(reposFile, []byte(reposContent), 0o644); err != nil {
		t.Fatalf("write repos file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"init-workspace",
		"--workspace", workspaceRoot,
		"--repos-file", reposFile,
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}

	manifestContent, err := os.ReadFile(filepath.Join(workspaceRoot, "workspace.yaml"))
	if err != nil {
		t.Fatalf("read workspace manifest: %v", err)
	}
	manifestText := string(manifestContent)
	if !strings.Contains(manifestText, "runtime:") {
		t.Fatalf("expected runtime block in manifest, got %q", manifestText)
	}
	if !strings.Contains(manifestText, "step_timeout_sec: 777") {
		t.Fatalf("expected step_timeout_sec from repos-file, got %q", manifestText)
	}
	if !strings.Contains(manifestText, "ui_cancel_poll_timeout_sec: 333") {
		t.Fatalf("expected ui_cancel_poll_timeout_sec from repos-file, got %q", manifestText)
	}
}

func TestServeAutoInitSupportsReposFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspacePath := filepath.Join(root, "arch-workspace")
	repoA := filepath.Join(root, "repos", "a")
	repoB := filepath.Join(root, "repos", "b")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatalf("create repoA: %v", err)
	}
	if err := os.MkdirAll(repoB, 0o755); err != nil {
		t.Fatalf("create repoB: %v", err)
	}
	reposFile := filepath.Join(root, "repos.yaml")
	reposContent := `repos:
  - name: repo-a
    path: ` + repoA + `
  - name: repo-b
    path: ` + repoB + `
runtime:
  profile:
    timeouts:
      step_timeout_sec: 1440
      heartbeat_sec: 12
`
	if err := os.WriteFile(reposFile, []byte(reposContent), 0o644); err != nil {
		t.Fatalf("write repos file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"serve",
		"--workspace", workspacePath,
		"--auto-init",
		"--repos-file", reposFile,
		"--dry-run",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(workspacePath, "workspace.yaml")); err != nil {
		t.Fatalf("expected workspace.yaml after auto-init repos-file: %v", err)
	}
	manifestContent, err := os.ReadFile(filepath.Join(workspacePath, "workspace.yaml"))
	if err != nil {
		t.Fatalf("read workspace manifest: %v", err)
	}
	manifestText := string(manifestContent)
	if !strings.Contains(manifestText, "step_timeout_sec: 1440") {
		t.Fatalf("expected runtime timeouts from repos-file in auto-init manifest, got %q", manifestText)
	}
	if !strings.Contains(manifestText, "heartbeat_sec: 12") {
		t.Fatalf("expected runtime heartbeat from repos-file in auto-init manifest, got %q", manifestText)
	}
}

func TestRunHeadlessReturnsRunnerUnavailableWhenCommandMissing(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_CLAUDE_CMD", "definitely-missing-acp-headless-command")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"run", "--workspace", root, "--pipeline", "refresh", "--runtime", "headless", "--non-interactive"}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "runner_unavailable") {
		t.Fatalf("expected runner_unavailable diagnostics, got %q", stderr.String())
	}
}

func TestServeHeadlessQwenDryRunDoesNotPreflightMissingCommand(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_QWEN_CMD", "definitely-missing-acp-qwen-command")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"serve",
		"--workspace", root,
		"--runtime", "headless",
		"--runtime-provider", "qwen-code",
		"--dry-run",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for dry-run without preflight, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider: qwen-code") {
		t.Fatalf("expected qwen-code output, got %q", stdout.String())
	}
}

func TestServeHeadlessCodexDryRunDoesNotPreflightMissingCommand(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_CODEX_CMD", "definitely-missing-acp-codex-command")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"serve",
		"--workspace", root,
		"--runtime", "headless",
		"--runtime-provider", "codex-code",
		"--dry-run",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for dry-run without preflight, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider: codex-code") {
		t.Fatalf("expected codex-code output, got %q", stdout.String())
	}
}

func TestServeHeadlessClaudeCodeWithStubRunnerDryRunSucceeds(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_CLAUDE_CMD", writeStubHeadlessRunner(t, "claude-code"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"serve",
		"--workspace", root,
		"--runtime", "headless",
		"--runtime-provider", "claude-code",
		"--dry-run",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "runtime mode: headless") {
		t.Fatalf("expected runtime mode output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider: claude-code") {
		t.Fatalf("expected runtime provider output, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "ignored in fake mode") {
		t.Fatalf("did not expect fake-mode provider note, got %q", stdout.String())
	}
}

func TestServeHeadlessQwenCodeWithStubRunnerDryRunSucceeds(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_QWEN_CMD", writeStubHeadlessRunner(t, "qwen-code"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"serve",
		"--workspace", root,
		"--runtime", "headless",
		"--runtime-provider", "qwen-code",
		"--dry-run",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "runtime mode: headless") {
		t.Fatalf("expected runtime mode output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider: qwen-code") {
		t.Fatalf("expected runtime provider output, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "ignored in fake mode") {
		t.Fatalf("did not expect fake-mode provider note, got %q", stdout.String())
	}
}

func TestServeHeadlessCodexCodeWithStubRunnerDryRunSucceeds(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_CODEX_CMD", writeStubCodexRunner(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"serve",
		"--workspace", root,
		"--runtime", "headless",
		"--runtime-provider", "codex-code",
		"--dry-run",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "runtime mode: headless") {
		t.Fatalf("expected runtime mode output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider: codex-code") {
		t.Fatalf("expected runtime provider output, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "ignored in fake mode") {
		t.Fatalf("did not expect fake-mode provider note, got %q", stdout.String())
	}
}

func TestRunHeadlessQwenReturnsRunnerUnavailableWhenCommandMissing(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_QWEN_CMD", "definitely-missing-acp-qwen-command")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"run",
		"--workspace", root,
		"--pipeline", "refresh",
		"--runtime", "headless",
		"--runtime-provider", "qwen-code",
		"--non-interactive",
	}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "runner_unavailable") {
		t.Fatalf("expected runner_unavailable diagnostics, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "qwen-code") {
		t.Fatalf("expected qwen-code diagnostics, got %q", stderr.String())
	}
}

func TestRunHeadlessCodexReturnsRunnerUnavailableWhenCommandMissing(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_CODEX_CMD", "definitely-missing-acp-codex-command")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"run",
		"--workspace", root,
		"--pipeline", "refresh",
		"--runtime", "headless",
		"--runtime-provider", "codex-code",
		"--non-interactive",
	}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "runner_unavailable") {
		t.Fatalf("expected runner_unavailable diagnostics, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "codex-code") {
		t.Fatalf("expected codex-code diagnostics, got %q", stderr.String())
	}
}

func TestRunHeadlessClaudeCodeWithStubRunnerSucceeds(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_CLAUDE_CMD", writeStubHeadlessRunner(t, "claude-code"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"run",
		"--workspace", root,
		"--pipeline", "init",
		"--runtime", "headless",
		"--runtime-provider", "claude-code",
		"--non-interactive",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "status: succeeded") {
		t.Fatalf("expected succeeded run, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider: claude-code") {
		t.Fatalf("expected runtime provider output, got %q", stdout.String())
	}
}

func TestRunHeadlessQwenCodeWithStubRunnerSucceeds(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_QWEN_CMD", writeStubHeadlessRunner(t, "qwen-code"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"run",
		"--workspace", root,
		"--pipeline", "init",
		"--runtime", "headless",
		"--runtime-provider", "qwen-code",
		"--non-interactive",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "status: succeeded") {
		t.Fatalf("expected succeeded run, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider: qwen-code") {
		t.Fatalf("expected runtime provider output, got %q", stdout.String())
	}

	runID := outputField(stdout.String(), "run_id")
	if runID == "" {
		t.Fatalf("expected non-empty run_id in output, got %q", stdout.String())
	}
	qualityPath := filepath.Join(root, "reports", "taskruns", runID+"-quality.json")
	content, err := os.ReadFile(qualityPath)
	if err != nil {
		t.Fatalf("read quality summary: %v", err)
	}
	var quality struct {
		RuntimeVersions []string `json:"runtime_versions"`
	}
	if err := json.Unmarshal(content, &quality); err != nil {
		t.Fatalf("decode quality summary: %v", err)
	}
	containsQwen := false
	for _, version := range quality.RuntimeVersions {
		if strings.Contains(strings.ToLower(version), "qwen-code") {
			containsQwen = true
			break
		}
	}
	if !containsQwen {
		t.Fatalf("expected qwen runtime in quality summary, got %+v", quality.RuntimeVersions)
	}
}

func TestRunHeadlessCodexCodeWithStubRunnerSucceeds(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_CODEX_CMD", writeStubCodexRunner(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"run",
		"--workspace", root,
		"--pipeline", "init",
		"--runtime", "headless",
		"--runtime-provider", "codex-code",
		"--non-interactive",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "status: succeeded") {
		t.Fatalf("expected succeeded run, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runtime provider: codex-code") {
		t.Fatalf("expected runtime provider output, got %q", stdout.String())
	}
}

func TestRunHeadlessRuntimeProviderFlagOverridesEnv(t *testing.T) {
	root := writeWorkspace(t)
	t.Setenv("ACP_RUNTIME_PROVIDER", "qwen-code")
	t.Setenv("ACP_CLAUDE_CMD", "missing-claude-provider-command")
	t.Setenv("ACP_QWEN_CMD", "missing-qwen-provider-command")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"run",
		"--workspace", root,
		"--pipeline", "refresh",
		"--runtime", "headless",
		"--runtime-provider", "claude-code",
		"--non-interactive",
	}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "missing-claude-provider-command") {
		t.Fatalf("expected CLI provider to win over env, got %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "missing-qwen-provider-command") {
		t.Fatalf("expected env provider command not to be used, got %q", stderr.String())
	}
}

func TestServeDryRunExecutionUsesEnvOverridesWhenCLIUnset(t *testing.T) {
	root := writeWorkspaceWithExecutionProfile(t, "sequential", 2, "fail_fast")
	t.Setenv("ACP_EXECUTION_STRATEGY", "parallel")
	t.Setenv("ACP_MAX_PARALLEL_TASKS", "5")
	t.Setenv("ACP_FAILURE_POLICY", "best_effort")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"serve",
		"--workspace", root,
		"--runtime", "fake",
		"--dry-run",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "execution strategy: parallel") {
		t.Fatalf("expected env execution strategy override, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "execution max_parallel_tasks: 5") {
		t.Fatalf("expected env max_parallel_tasks override, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "execution failure_policy: best_effort") {
		t.Fatalf("expected env failure_policy override, got %q", stdout.String())
	}
}

func TestServeDryRunExecutionCLIOverridesBeatEnvAndWorkspace(t *testing.T) {
	root := writeWorkspaceWithExecutionProfile(t, "parallel", 2, "fail_fast")
	t.Setenv("ACP_EXECUTION_STRATEGY", "parallel")
	t.Setenv("ACP_MAX_PARALLEL_TASKS", "5")
	t.Setenv("ACP_FAILURE_POLICY", "fail_fast")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"serve",
		"--workspace", root,
		"--runtime", "fake",
		"--execution-strategy", "sequential",
		"--max-parallel-tasks", "7",
		"--failure-policy", "best_effort",
		"--dry-run",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d: stderr=%q", exitCodeOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "execution strategy: sequential") {
		t.Fatalf("expected CLI execution strategy override, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "execution max_parallel_tasks: 7") {
		t.Fatalf("expected CLI max_parallel_tasks override, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "execution failure_policy: best_effort") {
		t.Fatalf("expected CLI failure_policy override, got %q", stdout.String())
	}
}

func TestEnsureWorkspaceGitRepositoryReturnsActionableErrorWhenGitMissing(t *testing.T) {
	t.Setenv("PATH", "")
	err := ensureWorkspaceGitRepository(t.TempDir())
	if err == nil {
		t.Fatalf("expected git required error")
	}
	if !strings.Contains(err.Error(), "workspace.git.init.git_required") {
		t.Fatalf("expected workspace.git.init.git_required code, got %q", err.Error())
	}
}

func TestEnsureWorkspaceGitRepositoryAcceptsGitFileMarker(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: /tmp/nonexistent\n"), 0o644); err != nil {
		t.Fatalf("write .git marker: %v", err)
	}
	if err := ensureWorkspaceGitRepository(root); err != nil {
		t.Fatalf("expected .git file marker to be accepted, got %v", err)
	}
}

func writeWorkspace(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "sample")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create sample repo path: %v", err)
	}
	writeTestRepoReadme(t, repoPath, "# Sample\n")
	manifestPath := filepath.Join(root, "workspace.yaml")
	manifest := "version: 1\nrepos:\n  - name: sample\n    path: " + repoPath + "\n"
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}
	return root
}

func writeWorkspaceWithExecutionProfile(t *testing.T, strategy string, maxParallel int, failurePolicy string) string {
	t.Helper()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "sample")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create sample repo path: %v", err)
	}
	writeTestRepoReadme(t, repoPath, "# Sample\n")
	manifest := strings.Join([]string{
		"version: 1",
		"repos:",
		"  - name: sample",
		"    path: " + repoPath,
		"runtime:",
		"  profile:",
		"    execution:",
		"      strategy: " + strategy,
		"      max_parallel_tasks: " + strconv.Itoa(maxParallel),
		"      failure_policy: " + failurePolicy,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}
	return root
}

func writeStubHeadlessRunner(t *testing.T, runtimeName string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "stub-headless-runner.sh")
	script := `#!/usr/bin/env bash
set -eu
TASK_PAYLOAD="$(cat)"
LAST_ARG=""
for arg in "$@"; do
  LAST_ARG="$arg"
done
TASK_PAYLOAD="$TASK_PAYLOAD" TASK_PROMPT="$LAST_ARG" python3 - <<'PY'
import json
import os
import re
import sys

raw = os.environ.get("TASK_PAYLOAD", "").strip()
prompt = os.environ.get("TASK_PROMPT", "")

def first_non_empty(mapping, keys):
    for key in keys:
        value = mapping.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()
    return ""

def from_prompt(field):
    patterns = [
        r'%s(?:\s+\([^)]+\))?\s*=\s*"([^"]+)"',
        r'"%s"\s*:\s*"([^"]+)"',
    ]
    for pattern in patterns:
        match = re.search(pattern % re.escape(field), prompt)
        if match:
            return match.group(1).strip()
    return ""

def step_id_from_prompt():
    match = re.search(r"STEP POLICY ([A-Za-z0-9._-]+):", prompt)
    if match:
        return match.group(1).strip()
    if "Write constitution-draft.json in write_root." in prompt:
        return "init.step0.constitution"
    if "Write asis-draft-manifest.json in write_root." in prompt:
        return "init.step2.asis_docs"
    if "Write validator-verdict.json in write_root." in prompt:
        return "init.step3.findings"
    if "Write proposals-draft-manifest.json in write_root." in prompt:
        return "init.step4.proposals"
    if "shard-pack-manifest.json" in prompt:
        return "init.step1.collect"
    return ""

def first_non_empty_list(mapping, keys):
    for key in keys:
        value = mapping.get(key)
        if isinstance(value, list) and value:
            return [str(item).strip() for item in value if str(item).strip()]
    return []

def slugify(value):
    return re.sub(r'[^a-z0-9]+', '-', value.lower()).strip('-') or "stub"

def infer_repo_scope_from_shard(shard):
    slug = slugify(shard)
    for suffix in [
        "-readme-md",
        "-makefile",
        "-dockerfile",
        "-package-json",
        "-pom-xml",
        "-build-gradle",
        "-settings-gradle",
    ]:
        if slug.endswith(suffix) and len(slug) > len(suffix):
            return slug[:-len(suffix)]
    return slug

task = {}
if raw:
    try:
        task = json.loads(raw)
    except Exception:
        task = {}

task_id = first_non_empty(task, ["task_id", "TaskID"]) or from_prompt("TaskID") or from_prompt("task_id") or "task"
step_id = first_non_empty(task, ["step_id", "StepID"]) or from_prompt("StepID") or from_prompt("step_id") or step_id_from_prompt() or "init.step1.collect"
run_id = first_non_empty(task, ["run_id", "RunID"]) or from_prompt("RunID") or from_prompt("run_id")
write_root = first_non_empty(task, ["write_root", "WriteRoot"]) or from_prompt("write_root") or from_prompt("WriteRoot")
artifact_root = first_non_empty(task, ["artifact_root", "ArtifactRoot"]) or from_prompt("artifact_root") or from_prompt("ArtifactRoot")
draft_root = first_non_empty(task, ["draft_final_root", "DraftFinalRoot"]) or from_prompt("draft_final_root") or from_prompt("DraftFinalRoot")
step_contract = first_non_empty(task, ["step_contract", "StepContract"]) or from_prompt("step_contract") or from_prompt("StepContract")
agent_role = first_non_empty(task, ["agent_role", "AgentRole"]) or from_prompt("agent_role") or from_prompt("AgentRole") or "architect"
shard_id = first_non_empty(task, ["shard_id", "ShardID"]) or from_prompt("shard_id") or from_prompt("ShardID") or slugify(step_id)
repo_scopes = first_non_empty_list(task, ["repo_scopes", "RepoScopes"])
if not repo_scopes:
    repo_scope = first_non_empty(task, ["repo_scope", "RepoScope"]) or from_prompt("repo_scope") or from_prompt("RepoScope")
    if repo_scope:
        repo_scopes = [repo_scope]
path_scopes = first_non_empty_list(task, ["path_scopes", "PathScopes"])
if not repo_scopes:
    inferred_repo_scope = infer_repo_scope_from_shard(shard_id)
    if inferred_repo_scope:
        repo_scopes = [inferred_repo_scope]

def write_runtime_draft(manifest_name, outputs, default_step_contract="draft"):
    if not write_root or not draft_root:
        return
    os.makedirs(write_root, exist_ok=True)
    os.makedirs(draft_root, exist_ok=True)
    rendered_outputs = []
    for output in outputs:
        draft_name = output["path"]
        with open(os.path.join(draft_root, draft_name), "w", encoding="utf-8") as handle:
            handle.write(output.get("content", "# Stub Draft\n"))
        rendered_outputs.append(
            {
                "path": draft_name,
                "canonical_path": output["canonical_path"],
                "kind": output["kind"],
                "title": output["title"],
            }
        )
    manifest = {
        "version": 1,
        "run_id": run_id or "run-1",
        "step_id": step_id,
        "step_contract": step_contract or default_step_contract,
        "agent_role": agent_role,
        "summary": "stub runtime draft",
        "outputs": rendered_outputs,
    }
    with open(os.path.join(write_root, manifest_name), "w", encoding="utf-8") as handle:
        json.dump(manifest, handle)

if step_id == "init.step0.constitution":
    write_runtime_draft(
        "constitution-draft.json",
        [
            {
                "path": "charter-overview.md",
                "canonical_path": "charter/overview.md",
                "kind": "charter",
                "title": "Stub Constitution",
                "content": "# Stub Constitution\n\n## Scope\n- Stub runner evidence: README.md.\n",
            },
            {
                "path": "baseline-subagents.yaml",
                "canonical_path": "skills/subagents.yaml",
                "kind": "bundle",
                "title": "Baseline Subagents",
                "content": "agents: []\n",
            }
        ],
    )
elif step_id in {"init.step2.asis_docs", "refresh.step2.asis_docs"}:
    write_runtime_draft(
        "asis-draft-manifest.json",
        [
            {
                "path": "overview.md",
                "canonical_path": "reports/as-is/overview.md",
                "kind": "report",
                "title": "Stub As-Is Overview",
            },
            {
                "path": "summary.md",
                "canonical_path": "reports/coverage/summary.md",
                "kind": "report",
                "title": "Stub Coverage Summary",
            },
            {
                "path": "architect-summary.md",
                "canonical_path": "reports/agent-outputs/architect/summary.md",
                "kind": "agent-output",
                "title": "Stub Architect Summary",
            },
        ],
        "as_is",
    )
elif step_id in {"init.step3.findings", "refresh.step3.findings"} and write_root:
    os.makedirs(write_root, exist_ok=True)
    verdict = {
        "version": 1,
        "run_id": run_id or "run-1",
        "generated_at": "2026-04-21T10:00:00Z",
        "verdict": "PASS",
        "summary": "stub validator verdict",
        "checked_paths": ["reports/taskruns/" + (run_id or "run-1") + "/staging/final/final-run-index.json"],
        "fixed_paths": [],
        "findings": [],
        "questions": [],
    }
    with open(os.path.join(write_root, "validator-verdict.json"), "w", encoding="utf-8") as handle:
        json.dump(verdict, handle)
elif step_id in {"init.step4.proposals", "refresh.step4.proposals"}:
    write_runtime_draft(
        "proposals-draft-manifest.json",
        [
            {
                "path": "proposal.md",
                "canonical_path": "proposals/proposal-baseline/proposal.md",
                "kind": "proposal",
                "title": "Stub Proposal",
            }
        ],
    )

if step_id in {"init.step1.collect", "refresh.step1.collect"} and write_root:
    os.makedirs(write_root, exist_ok=True)
    document_name = slugify(shard_id) + ".md"
    document_id = "doc." + slugify(shard_id)
    citation_id = "cite." + slugify(shard_id)
    canonical_path = "reports/agent-outputs/domains/" + document_name
    with open(os.path.join(write_root, document_name), "w", encoding="utf-8") as handle:
        handle.write("# Stub Analysis\n")
    manifest = {
        "version": 1,
        "run_id": run_id or "run-1",
        "step_id": step_id,
        "shard_id": shard_id,
        "agent_role": "shard-analyst",
        "artifact_root": write_root,
        "repo_scopes": repo_scopes,
        "path_scopes": path_scopes,
        "summary": "stub shard pack",
        "documents": [
            {
                "id": document_id,
                "kind": "report",
                "title": "Stub Analysis",
                "path": document_name,
                "canonical_path": canonical_path,
                "topics": ["stub"],
                "citation_ids": [citation_id],
                "status": "staged"
            }
        ],
        "citations": [
            {
                "id": citation_id,
                "repo": repo_scopes[0] if repo_scopes else "stub-repo",
                "path": "README.md",
                "claim_ids": ["claim.stub"],
                "document_ids": [document_id]
            }
        ],
        "semantic": {
            "coverage": {
                "observed": ["stub"],
                "missing": ["owner mappings"],
                "notes": ["stub manifest for integration tests"]
            },
            "questions": [],
            "entities": [],
            "edges": [],
            "findings": []
        }
    }
    with open(os.path.join(write_root, "shard-pack-manifest.json"), "w", encoding="utf-8") as handle:
        json.dump(manifest, handle)
PY
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub headless runner: %v", err)
	}
	return path
}

func writeStubCodexRunner(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "stub-codex-runner.sh")
	script := `#!/usr/bin/env bash
set -eu
TASK_PROMPT="$(cat)"
TASK_PROMPT="$TASK_PROMPT" python3 - <<'PY'
import json
import os
import pathlib
import re

prompt = os.environ.get("TASK_PROMPT", "")

def policy_value(name):
    match = re.search(r'%s(?:\s+\([^)]+\))?\s*=\s*"([^"]+)"' % re.escape(name), prompt)
    return match.group(1).strip() if match else ""

def step_id():
    match = re.search(r"STEP POLICY ([A-Za-z0-9._-]+):", prompt)
    if match:
        return match.group(1).strip()
    if "Write constitution-draft.json in write_root." in prompt:
        return "init.step0.constitution"
    if "Write asis-draft-manifest.json in write_root." in prompt:
        return "init.step2.asis_docs"
    if "Write validator-verdict.json in write_root." in prompt:
        return "refresh.step3.findings"
    if "Write proposals-draft-manifest.json in write_root." in prompt:
        return "init.step4.proposals"
    return "init.step1.collect"

def slugify(value):
    return re.sub(r'[^a-z0-9]+', '-', value.lower()).strip('-') or "stub"

def run_id_from_path(path_value):
    parts = pathlib.Path(path_value).parts
    for idx, part in enumerate(parts):
        if part == "taskruns" and idx + 1 < len(parts):
            return parts[idx + 1]
    return "run-1"

write_root = policy_value("write_root")
draft_root = policy_value("draft_final_root")
artifact_root = policy_value("artifact_root") or write_root
agent_role = policy_value("agent_role") or "architect"
step_contract = policy_value("step_contract") or "draft"
current_step = step_id()
run_id = run_id_from_path(write_root or draft_root or "/tmp/run-1")
repo_scope = policy_value("repo_scope") or policy_value("RepoScope") or "sample"

if current_step == "init.step0.constitution":
    os.makedirs(write_root, exist_ok=True)
    os.makedirs(draft_root, exist_ok=True)
    with open(os.path.join(draft_root, "charter-overview.md"), "w", encoding="utf-8") as handle:
        handle.write("# Stub Constitution\n")
    with open(os.path.join(draft_root, "baseline-subagents.yaml"), "w", encoding="utf-8") as handle:
        handle.write("version: 1\n")
    manifest = {
        "version": 1,
        "run_id": run_id,
        "step_id": current_step,
        "step_contract": step_contract,
        "agent_role": agent_role,
        "summary": "stub codex constitution draft",
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
    with open(os.path.join(write_root, "constitution-draft.json"), "w", encoding="utf-8") as handle:
        json.dump(manifest, handle)
elif current_step in {"init.step2.asis_docs", "refresh.step2.asis_docs"}:
    os.makedirs(write_root, exist_ok=True)
    os.makedirs(draft_root, exist_ok=True)
    with open(os.path.join(draft_root, "overview.md"), "w", encoding="utf-8") as handle:
        handle.write("# Stub As-Is\n")
    with open(os.path.join(draft_root, "summary.md"), "w", encoding="utf-8") as handle:
        handle.write("# Stub Coverage Summary\n")
    with open(os.path.join(draft_root, "architect-summary.md"), "w", encoding="utf-8") as handle:
        handle.write("# Stub Architect Summary\n")
    manifest = {
        "version": 1,
        "run_id": run_id,
        "step_id": current_step,
        "step_contract": step_contract or "as_is",
        "agent_role": agent_role,
        "summary": "stub codex as-is draft",
        "outputs": [
            {
                "path": "overview.md",
                "canonical_path": "reports/as-is/overview.md",
                "kind": "report",
                "title": "Stub As-Is Overview"
            },
            {
                "path": "summary.md",
                "canonical_path": "reports/coverage/summary.md",
                "kind": "report",
                "title": "Stub Coverage Summary"
            },
            {
                "path": "architect-summary.md",
                "canonical_path": "reports/agent-outputs/architect/summary.md",
                "kind": "agent-output",
                "title": "Stub Architect Summary"
            }
        ]
    }
    with open(os.path.join(write_root, "asis-draft-manifest.json"), "w", encoding="utf-8") as handle:
        json.dump(manifest, handle)
elif current_step in {"init.step3.findings", "refresh.step3.findings"}:
    os.makedirs(write_root, exist_ok=True)
    verdict = {
        "version": 1,
        "run_id": run_id,
        "generated_at": "2026-04-21T10:00:00Z",
        "verdict": "PASS",
        "summary": "stub codex validator verdict",
        "checked_paths": ["reports/taskruns/" + run_id + "/staging/final/final-run-index.json"],
        "fixed_paths": [],
        "findings": [],
        "questions": []
    }
    with open(os.path.join(write_root, "validator-verdict.json"), "w", encoding="utf-8") as handle:
        json.dump(verdict, handle)
elif current_step in {"init.step4.proposals", "refresh.step4.proposals"}:
    os.makedirs(write_root, exist_ok=True)
    os.makedirs(draft_root, exist_ok=True)
    with open(os.path.join(draft_root, "proposal.md"), "w", encoding="utf-8") as handle:
        handle.write("# Stub Proposal\n")
    manifest = {
        "version": 1,
        "run_id": run_id,
        "step_id": current_step,
        "step_contract": step_contract,
        "agent_role": agent_role,
        "summary": "stub codex proposal draft",
        "outputs": [
            {
                "path": "proposal.md",
                "canonical_path": "proposals/proposal-baseline/proposal.md",
                "kind": "proposal",
                "title": "Stub Proposal"
            }
        ]
    }
    with open(os.path.join(write_root, "proposals-draft-manifest.json"), "w", encoding="utf-8") as handle:
        json.dump(manifest, handle)
else:
    os.makedirs(write_root, exist_ok=True)
    shard_id = slugify(os.path.basename(write_root))
    document_name = shard_id + ".md"
    with open(os.path.join(write_root, document_name), "w", encoding="utf-8") as handle:
        handle.write("# Stub Analysis\n")
    manifest = {
        "version": 1,
        "run_id": run_id,
        "step_id": current_step,
        "shard_id": shard_id,
        "agent_role": "shard-analyst",
        "artifact_root": artifact_root,
        "repo_scopes": [repo_scope] if repo_scope else [],
        "path_scopes": [],
        "summary": "stub codex shard pack",
        "documents": [
            {
                "id": "doc." + shard_id,
                "kind": "report",
                "title": "Stub Analysis",
                "path": document_name,
                "canonical_path": "reports/agent-outputs/domains/" + document_name,
                "topics": ["stub"],
                "citation_ids": ["cite." + shard_id],
                "status": "staged"
            }
        ],
        "citations": [
            {
                "id": "cite." + shard_id,
                "repo": repo_scope or "sample",
                "path": "README.md",
                "claim_ids": ["claim.stub"],
                "document_ids": ["doc." + shard_id]
            }
        ],
        "semantic": {
            "coverage": {
                "observed": ["stub"],
                "missing": ["owner mappings"],
                "notes": ["stub codex manifest"]
            },
            "questions": [],
            "entities": [],
            "edges": [],
            "findings": []
        }
    }
    with open(os.path.join(write_root, "shard-pack-manifest.json"), "w", encoding="utf-8") as handle:
        json.dump(manifest, handle)

print('{"type":"result","status":"ok"}')
PY
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub codex runner: %v", err)
	}
	return path
}

func outputField(output string, key string) string {
	prefix := key + ": "
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func writeTestRepoReadme(t *testing.T, repoPath string, content string) {
	t.Helper()
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path for README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write repo README: %v", err)
	}
}

func hasDoctorCLIResultCheck(checks []struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}, id string, status string) bool {
	for _, check := range checks {
		if check.ID == id && check.Status == status {
			return true
		}
	}
	return false
}
