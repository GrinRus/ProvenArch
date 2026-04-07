package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRepoSourcesPathMode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "payments-service")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	writeManifestFile(t, root, `
version: 1
repos:
  - name: payments-service
    path: `+repoPath+`
`)

	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	resolved, diagnostics := ws.ResolveRepoSources(context.Background(), ResolveOptions{FetchGit: false, VerifyRefs: true})
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", diagnostics)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected one resolved repo, got %d", len(resolved))
	}
	if resolved[0].Path != repoPath {
		t.Fatalf("unexpected resolved path: %q", resolved[0].Path)
	}
}

func TestResolveRepoSourcesGitURLFetch(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for resolver tests")
	}

	tmp := t.TempDir()
	sourceRepo := filepath.Join(tmp, "source-repo")
	bareRepo := filepath.Join(tmp, "remote.git")
	if err := os.MkdirAll(sourceRepo, 0o755); err != nil {
		t.Fatalf("create source repo: %v", err)
	}

	runGit(t, sourceRepo, "init")
	runGit(t, sourceRepo, "config", "user.email", "acp@example.local")
	runGit(t, sourceRepo, "config", "user.name", "ACP")
	if err := os.WriteFile(filepath.Join(sourceRepo, "README.md"), []byte("# source\n"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	runGit(t, sourceRepo, "add", "README.md")
	runGit(t, sourceRepo, "commit", "-m", "init")
	runGit(t, tmp, "clone", "--bare", sourceRepo, bareRepo)

	workspaceRoot := filepath.Join(tmp, "workspace")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("create workspace root: %v", err)
	}
	writeManifestFile(t, workspaceRoot, `
version: 1
repos:
  - name: source-repo
    git_url: `+bareRepo+`
`)

	ws, err := Open(workspaceRoot)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	resolved, diagnostics := ws.ResolveRepoSources(context.Background(), ResolveOptions{FetchGit: true, VerifyRefs: true})
	for _, diagnostic := range diagnostics {
		if diagnostic.Level == DiagnosticError {
			t.Fatalf("unexpected resolver error diagnostics: %+v", diagnostics)
		}
	}
	if len(resolved) != 1 {
		t.Fatalf("expected one resolved repo, got %d", len(resolved))
	}
	if _, err := os.Stat(filepath.Join(resolved[0].Path, ".git")); err != nil {
		t.Fatalf("expected cloned git repository in cache: %v", err)
	}
}

func TestResolveRepoSourcesPathRefFallbackToOriginAndHeadMismatchWarning(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for resolver tests")
	}

	tmp := t.TempDir()
	sourceRepo := filepath.Join(tmp, "source-repo")
	bareRepo := filepath.Join(tmp, "remote.git")
	localRepo := filepath.Join(tmp, "local-repo")
	if err := os.MkdirAll(sourceRepo, 0o755); err != nil {
		t.Fatalf("create source repo: %v", err)
	}

	runGit(t, sourceRepo, "init", "-b", "main")
	runGit(t, sourceRepo, "config", "user.email", "acp@example.local")
	runGit(t, sourceRepo, "config", "user.name", "ACP")
	if err := os.WriteFile(filepath.Join(sourceRepo, "README.md"), []byte("# source\n"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	runGit(t, sourceRepo, "add", "README.md")
	runGit(t, sourceRepo, "commit", "-m", "init")
	runGit(t, tmp, "clone", "--bare", sourceRepo, bareRepo)
	runGit(t, tmp, "clone", bareRepo, localRepo)
	runGit(t, localRepo, "config", "user.email", "acp@example.local")
	runGit(t, localRepo, "config", "user.name", "ACP")
	runGit(t, localRepo, "checkout", "-b", "work")
	if err := os.WriteFile(filepath.Join(localRepo, "work.txt"), []byte("work\n"), 0o644); err != nil {
		t.Fatalf("write local work file: %v", err)
	}
	runGit(t, localRepo, "add", "work.txt")
	runGit(t, localRepo, "commit", "-m", "work")
	runGit(t, localRepo, "branch", "-D", "main")

	workspaceRoot := filepath.Join(tmp, "workspace")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("create workspace root: %v", err)
	}
	writeManifestFile(t, workspaceRoot, `
version: 1
repos:
  - name: source-repo
    path: `+localRepo+`
    ref: main
`)

	ws, err := Open(workspaceRoot)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	resolved, diagnostics := ws.ResolveRepoSources(context.Background(), ResolveOptions{FetchGit: false, VerifyRefs: true})
	if len(resolved) != 1 {
		t.Fatalf("expected one resolved repo, got %d", len(resolved))
	}

	hasFallbackWarning := false
	hasHeadMismatchWarning := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Level == DiagnosticError {
			t.Fatalf("expected warnings only, got %+v", diagnostics)
		}
		if diagnostic.Code == "workspace.repo.ref.resolved_via_remote" {
			hasFallbackWarning = true
		}
		if diagnostic.Code == "workspace.repo.ref.head_mismatch" {
			hasHeadMismatchWarning = true
		}
	}
	if !hasFallbackWarning {
		t.Fatalf("expected workspace.repo.ref.resolved_via_remote warning, got %+v", diagnostics)
	}
	if !hasHeadMismatchWarning {
		t.Fatalf("expected workspace.repo.ref.head_mismatch warning, got %+v", diagnostics)
	}
}

func TestResolveRepoSourcesPathInvalidRefIncludesTriedCandidates(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for resolver tests")
	}

	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	runGit(t, repoPath, "init", "-b", "master")
	runGit(t, repoPath, "config", "user.email", "acp@example.local")
	runGit(t, repoPath, "config", "user.name", "ACP")
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# repo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGit(t, repoPath, "add", "README.md")
	runGit(t, repoPath, "commit", "-m", "init")

	workspaceRoot := filepath.Join(tmp, "workspace")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("create workspace root: %v", err)
	}
	writeManifestFile(t, workspaceRoot, `
version: 1
repos:
  - name: sample
    path: `+repoPath+`
    ref: main
`)

	ws, err := Open(workspaceRoot)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	_, diagnostics := ws.ResolveRepoSources(context.Background(), ResolveOptions{FetchGit: false, VerifyRefs: true})
	if len(diagnostics) == 0 {
		t.Fatalf("expected diagnostics for invalid ref")
	}
	hasInvalidRef := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != "workspace.repo.ref.invalid" {
			continue
		}
		hasInvalidRef = true
		if !strings.Contains(diagnostic.Message, "tried:") {
			t.Fatalf("expected tried candidates in message, got %q", diagnostic.Message)
		}
	}
	if !hasInvalidRef {
		t.Fatalf("expected workspace.repo.ref.invalid, got %+v", diagnostics)
	}
}

func TestResolveRepoSourcesPathOriginRefDoesNotDuplicateOriginPrefix(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for resolver tests")
	}

	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	runGit(t, repoPath, "init", "-b", "master")
	runGit(t, repoPath, "config", "user.email", "acp@example.local")
	runGit(t, repoPath, "config", "user.name", "ACP")
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# repo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGit(t, repoPath, "add", "README.md")
	runGit(t, repoPath, "commit", "-m", "init")

	workspaceRoot := filepath.Join(tmp, "workspace")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("create workspace root: %v", err)
	}
	writeManifestFile(t, workspaceRoot, `
version: 1
repos:
  - name: sample
    path: `+repoPath+`
    ref: origin/main
`)

	ws, err := Open(workspaceRoot)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	_, diagnostics := ws.ResolveRepoSources(context.Background(), ResolveOptions{FetchGit: false, VerifyRefs: true})
	hasInvalidRef := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != "workspace.repo.ref.invalid" {
			continue
		}
		hasInvalidRef = true
		if strings.Contains(diagnostic.Message, "origin/origin") {
			t.Fatalf("expected tried candidates without duplicated origin prefix, got %q", diagnostic.Message)
		}
	}
	if !hasInvalidRef {
		t.Fatalf("expected workspace.repo.ref.invalid, got %+v", diagnostics)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
	}
}
