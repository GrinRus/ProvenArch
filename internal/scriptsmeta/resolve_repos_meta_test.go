package scriptsmeta_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type declaredRepo struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Path   string `json:"path"`
	GitURL string `json:"git_url"`
	Ref    string `json:"ref"`
}

type reposMeta struct {
	ReposFile         string         `json:"repos_file"`
	TargetReposFile   string         `json:"target_repos_file"`
	ProfileID         string         `json:"profile_id"`
	ProfileSourceKind string         `json:"profile_source_kind"`
	ExpectedRepoCount int            `json:"expected_repo_count"`
	TargetProfile     string         `json:"target_profile"`
	DeclaredRepos     []declaredRepo `json:"declared_repos"`
}

type resolveInput struct {
	reposFile         string
	expectedRepoCount string
	sourceKind        string
	profileID         string
}

func TestResolveReposMetaDetectsPathSourceKind(t *testing.T) {
	requirePythonAndYAML(t)
	scriptPath := resolveReposMetaScriptPath(t)
	tmp := t.TempDir()

	repoPath := filepath.Join(tmp, "repo-a")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo path: %v", err)
	}
	repoPathResolved, err := filepath.EvalSymlinks(repoPath)
	if err != nil {
		t.Fatalf("resolve repo path symlinks: %v", err)
	}
	reposFile := filepath.Join(tmp, "repos.yaml")
	writeTextFile(t, reposFile, "version: 1\nrepos:\n  - name: repo-a\n    path: ./repo-a\n")

	meta, output, err := runResolveReposMeta(scriptPath, resolveInput{
		reposFile:         reposFile,
		expectedRepoCount: "1",
		sourceKind:        "",
		profileID:         "single-path",
	})
	if err != nil {
		t.Fatalf("resolve repos meta failed: %v\n%s", err, output)
	}
	if meta.ProfileSourceKind != "path" {
		t.Fatalf("expected profile_source_kind=path, got %q", meta.ProfileSourceKind)
	}
	if meta.ExpectedRepoCount != 1 {
		t.Fatalf("expected expected_repo_count=1, got %d", meta.ExpectedRepoCount)
	}
	if len(meta.DeclaredRepos) != 1 {
		t.Fatalf("expected 1 declared repo, got %d", len(meta.DeclaredRepos))
	}
	if meta.DeclaredRepos[0].Source != "path" {
		t.Fatalf("expected declared repo source=path, got %q", meta.DeclaredRepos[0].Source)
	}
	if meta.DeclaredRepos[0].Path != repoPathResolved {
		t.Fatalf("expected declared repo path=%q, got %q", repoPathResolved, meta.DeclaredRepos[0].Path)
	}
}

func TestResolveReposMetaDetectsGitURLSourceKind(t *testing.T) {
	requirePythonAndYAML(t)
	scriptPath := resolveReposMetaScriptPath(t)
	tmp := t.TempDir()

	reposFile := filepath.Join(tmp, "repos.yaml")
	writeTextFile(t, reposFile, strings.Join([]string{
		"version: 1",
		"repos:",
		"  - name: repo-a",
		"    git_url: https://example.com/org/repo-a.git",
		"    ref: deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		"  - name: repo-b",
		"    git_url: https://example.com/org/repo-b.git",
		"    ref: cafebabecafebabecafebabecafebabecafebabe",
		"",
	}, "\n"))

	meta, output, err := runResolveReposMeta(scriptPath, resolveInput{
		reposFile:         reposFile,
		expectedRepoCount: "2",
		sourceKind:        "",
		profileID:         "multi-git-url",
	})
	if err != nil {
		t.Fatalf("resolve repos meta failed: %v\n%s", err, output)
	}
	if meta.ProfileSourceKind != "git_url" {
		t.Fatalf("expected profile_source_kind=git_url, got %q", meta.ProfileSourceKind)
	}
	if len(meta.DeclaredRepos) != 2 {
		t.Fatalf("expected 2 declared repos, got %d", len(meta.DeclaredRepos))
	}
	for idx, repo := range meta.DeclaredRepos {
		if repo.Source != "git_url" {
			t.Fatalf("declared repo[%d] expected source=git_url, got %q", idx+1, repo.Source)
		}
		if strings.TrimSpace(repo.Ref) == "" {
			t.Fatalf("declared repo[%d] expected pinned ref, got empty", idx+1)
		}
	}
}

func TestResolveReposMetaDetectsMixedSourceKind(t *testing.T) {
	requirePythonAndYAML(t)
	scriptPath := resolveReposMetaScriptPath(t)
	tmp := t.TempDir()

	repoPath := filepath.Join(tmp, "repo-local")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo path: %v", err)
	}
	reposFile := filepath.Join(tmp, "repos.yaml")
	writeTextFile(t, reposFile, strings.Join([]string{
		"version: 1",
		"repos:",
		"  - name: repo-local",
		"    path: ./repo-local",
		"  - name: repo-remote",
		"    git_url: https://example.com/org/repo-remote.git",
		"",
	}, "\n"))

	meta, output, err := runResolveReposMeta(scriptPath, resolveInput{
		reposFile:         reposFile,
		expectedRepoCount: "2",
		sourceKind:        "",
		profileID:         "mixed-profile",
	})
	if err != nil {
		t.Fatalf("resolve repos meta failed: %v\n%s", err, output)
	}
	if meta.ProfileSourceKind != "mixed" {
		t.Fatalf("expected profile_source_kind=mixed, got %q", meta.ProfileSourceKind)
	}
}

func TestResolveReposMetaRejectsGitURLWithoutPinnedRef(t *testing.T) {
	requirePythonAndYAML(t)
	scriptPath := resolveReposMetaScriptPath(t)
	tmp := t.TempDir()

	reposFile := filepath.Join(tmp, "repos.yaml")
	writeTextFile(t, reposFile, "version: 1\nrepos:\n  - name: repo-a\n    git_url: https://example.com/org/repo-a.git\n")

	_, output, err := runResolveReposMeta(scriptPath, resolveInput{
		reposFile:         reposFile,
		expectedRepoCount: "1",
		sourceKind:        "",
		profileID:         "single-git-url",
	})
	if err == nil {
		t.Fatalf("expected resolver to fail for unpinned git_url ref")
	}
	if !strings.Contains(output, "git_url entry must have pinned ref") {
		t.Fatalf("expected pinned-ref failure, got output:\n%s", output)
	}
}

func TestResolveReposMetaRejectsExplicitSourceKindMismatch(t *testing.T) {
	requirePythonAndYAML(t)
	scriptPath := resolveReposMetaScriptPath(t)
	tmp := t.TempDir()

	repoPath := filepath.Join(tmp, "repo-a")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo path: %v", err)
	}
	reposFile := filepath.Join(tmp, "repos.yaml")
	writeTextFile(t, reposFile, "version: 1\nrepos:\n  - name: repo-a\n    path: ./repo-a\n")

	_, output, err := runResolveReposMeta(scriptPath, resolveInput{
		reposFile:         reposFile,
		expectedRepoCount: "1",
		sourceKind:        "git_url",
		profileID:         "single-git-url",
	})
	if err == nil {
		t.Fatalf("expected resolver to fail for source_kind mismatch")
	}
	if !strings.Contains(output, "profile source_kind=git_url but repos[1] uses path") {
		t.Fatalf("expected source_kind mismatch failure, got output:\n%s", output)
	}
}

func resolveReposMetaScriptPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve test file path: runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	scriptPath := filepath.Join(repoRoot, "scripts", "resolve-repos-meta.py")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("resolve script path %q: %v", scriptPath, err)
	}
	return scriptPath
}

func requirePythonAndYAML(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skipf("python3 is unavailable: %v", err)
	}
	check := exec.Command("python3", "-c", "import yaml")
	if output, err := check.CombinedOutput(); err != nil {
		t.Skipf("PyYAML is unavailable: %v (%s)", err, strings.TrimSpace(string(output)))
	}
}

func runResolveReposMeta(scriptPath string, input resolveInput) (reposMeta, string, error) {
	var meta reposMeta
	outPath := filepath.Join(filepath.Dir(input.reposFile), "resolved-meta.json")
	cmd := exec.Command(
		"python3",
		scriptPath,
		"--repos-file", input.reposFile,
		"--expected-repo-count", input.expectedRepoCount,
		"--source-kind", input.sourceKind,
		"--profile-id", input.profileID,
		"--out", outPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return reposMeta{}, string(output), err
	}
	raw, readErr := os.ReadFile(outPath)
	if readErr != nil {
		return reposMeta{}, string(output), readErr
	}
	if unmarshalErr := json.Unmarshal(raw, &meta); unmarshalErr != nil {
		return reposMeta{}, string(output), unmarshalErr
	}
	return meta, string(output), nil
}

func writeTextFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
