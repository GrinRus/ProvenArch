package orchestrator

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestRunLogPersistsResolvedGitURLSourceSHA(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for source resolution tests")
	}

	tmp := t.TempDir()
	sourceRepo := filepath.Join(tmp, "source-repo")
	bareRepo := filepath.Join(tmp, "remote.git")
	workspaceRoot := filepath.Join(tmp, "workspace")
	if err := os.MkdirAll(sourceRepo, 0o755); err != nil {
		t.Fatalf("create source repo: %v", err)
	}
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("create workspace root: %v", err)
	}
	runGitForSourceResolution(t, tmp, "init", "--bare", bareRepo)
	runGitForSourceResolution(t, bareRepo, "symbolic-ref", "HEAD", "refs/heads/main")
	runGitForSourceResolution(t, sourceRepo, "init", "-b", "main")
	runGitForSourceResolution(t, sourceRepo, "config", "user.email", "acp@example.local")
	runGitForSourceResolution(t, sourceRepo, "config", "user.name", "ACP")
	runGitForSourceResolution(t, sourceRepo, "remote", "add", "origin", bareRepo)
	resolvedSHA := commitFileForSourceResolution(t, sourceRepo, "README.md", "# source\n", "init")
	runGitForSourceResolution(t, sourceRepo, "push", "-u", "origin", "main")

	if err := os.WriteFile(filepath.Join(workspaceRoot, "workspace.yaml"), []byte(`
version: 1
repos:
  - name: source-repo
    git_url: `+bareRepo+`
`), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}
	ws, err := workspace.Open(workspaceRoot)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	service := NewService(
		WithRunner(sourceResolutionFailingRunner{}),
		WithHistoryWorkspace(ws),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected runner failure after source resolution")
	}
	if info.RunID == "" {
		t.Fatalf("expected run id")
	}

	logs, ok, err := service.GetRunLogs(info.RunID, 0, 100)
	if err != nil {
		t.Fatalf("get run logs: %v", err)
	}
	if !ok {
		t.Fatalf("expected run logs for %s", info.RunID)
	}
	for _, entry := range logs.Items {
		if entry.Message != "runtime execution profile resolved" {
			continue
		}
		repos, ok := entry.Fields["source_repos"].([]any)
		if !ok || len(repos) != 1 {
			t.Fatalf("expected one source_repos entry, got %#v", entry.Fields["source_repos"])
		}
		repo, ok := repos[0].(map[string]any)
		if !ok {
			t.Fatalf("expected source repo object, got %#v", repos[0])
		}
		if got := strings.TrimSpace(stringField(repo, "resolved_sha")); got != resolvedSHA {
			t.Fatalf("expected resolved_sha %s, got %s in fields %#v", resolvedSHA, got, repo)
		}
		return
	}
	t.Fatalf("runtime execution profile log with source_repos not found in %#v", logs.Items)
}

type sourceResolutionFailingRunner struct{}

func (sourceResolutionFailingRunner) Run(context.Context, acpruntime.Task) (acpruntime.Result, error) {
	return acpruntime.Result{}, acpruntime.WrapRunnerError(
		acpruntime.ProviderClaudeCode,
		acpruntime.ErrorCodeRunnerUnavailable,
		"intentional source resolution test failure",
		errors.New("intentional failure"),
	)
}

func stringField(values map[string]any, key string) string {
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func commitFileForSourceResolution(t *testing.T, repoPath string, relPath string, content string, message string) string {
	t.Helper()
	absPath := filepath.Join(repoPath, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("create commit file dir: %v", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write commit file: %v", err)
	}
	runGitForSourceResolution(t, repoPath, "add", relPath)
	runGitForSourceResolution(t, repoPath, "commit", "-m", message)
	return strings.TrimSpace(runGitOutputForSourceResolution(t, repoPath, "rev-parse", "--verify", "HEAD^{commit}"))
}

func runGitForSourceResolution(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
	}
}

func runGitOutputForSourceResolution(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
	}
	return strings.TrimSpace(string(output))
}
