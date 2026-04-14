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
	if !strings.Contains(stdout.String(), "acp serve --workspace <abs-path>") {
		t.Fatalf("expected help output, got %q", stdout.String())
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
	if !strings.Contains(stderr.String(), "Usage: acp serve --workspace <abs-path>") {
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
	if !strings.Contains(stderr.String(), "Usage: acp run --workspace <abs-path> --pipeline init|refresh [--refresh-mode incremental|full] [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]") {
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

func TestRunPipelineRejectsInvalidRefreshMode(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{
		"run",
		"--workspace", root,
		"--pipeline", "refresh",
		"--refresh-mode", "delta",
		"--non-interactive",
	}, &stdout, &stderr)
	if code != exitCodeValidation {
		t.Fatalf("expected exit code %d, got %d", exitCodeValidation, code)
	}
	if !strings.Contains(stderr.String(), "unsupported refresh mode") {
		t.Fatalf("expected refresh-mode validation error, got %q", stderr.String())
	}
}

func TestRunPipelineInitWarnsRefreshModeIgnored(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{
		"run",
		"--workspace", root,
		"--pipeline", "init",
		"--refresh-mode", "full",
		"--non-interactive",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stderr.String(), "warning: --refresh-mode=full is ignored for init pipeline") {
		t.Fatalf("expected init warning about ignored refresh mode, got %q", stderr.String())
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
	if !strings.Contains(stdout.String(), "refresh_mode: incremental") {
		t.Fatalf("expected default refresh mode output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunPipelineRefreshPrintsExplicitRefreshMode(t *testing.T) {
	t.Parallel()

	root := writeWorkspace(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{
		"run",
		"--workspace", root,
		"--pipeline", "refresh",
		"--refresh-mode", "full",
		"--non-interactive",
	}, &stdout, &stderr)
	if code != exitCodeOK {
		t.Fatalf("expected exit code %d, got %d", exitCodeOK, code)
	}
	if !strings.Contains(stdout.String(), "refresh_mode: full") {
		t.Fatalf("expected explicit refresh mode output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
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

func TestServeHeadlessQwenReturnsRunnerUnavailableWhenCommandMissing(t *testing.T) {
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
    match = re.search(r'"%s"\s*:\s*"([^"]+)"' % re.escape(field), prompt)
    return match.group(1).strip() if match else ""

task = {}
if raw:
    try:
        task = json.loads(raw)
    except Exception:
        task = {}

task_id = first_non_empty(task, ["task_id", "TaskID"]) or from_prompt("TaskID") or from_prompt("task_id") or "task"
step_id = first_non_empty(task, ["step_id", "StepID"]) or from_prompt("StepID") or from_prompt("step_id") or "init.step2.service_collect"
run_id = first_non_empty(task, ["run_id", "RunID"]) or from_prompt("RunID") or from_prompt("run_id")
payload = {
    "meta": {
        "task_id": task_id,
        "step_id": step_id,
        "run_id": run_id,
        "runtime": {
            "name": "` + runtimeName + `",
            "version": "stub"
        },
        "started_at": "2026-04-03T12:00:00Z"
    },
    "summary": "stub taskresult",
    "changeset": []
}
sys.stdout.write(json.dumps(payload))
PY
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub headless runner: %v", err)
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
