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
	if !strings.Contains(stderr.String(), "Usage: acp serve --workspace <abs-path> [--runtime fake|headless]") {
		t.Fatalf("expected serve usage in stderr, got %q", stderr.String())
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
	if !strings.Contains(stdout.String(), "workspace validated") {
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
