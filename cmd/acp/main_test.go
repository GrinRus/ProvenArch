package main

import (
	"bytes"
	"os"
	"path/filepath"
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
	if !strings.Contains(stderr.String(), "Usage: acp init-workspace --workspace <abs-path> --repo-name <name> (--repo-path <path> | --repo-git-url <url>)") {
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
	if !strings.Contains(stderr.String(), "Usage: acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--non-interactive]") {
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
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
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
	if !strings.Contains(stderr.String(), "set exactly one of --repo-path or --repo-git-url") {
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
	if !strings.Contains(stderr.String(), "set exactly one of --repo-path or --repo-git-url") {
		t.Fatalf("expected mutually-exclusive source validation error, got %q", stderr.String())
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
