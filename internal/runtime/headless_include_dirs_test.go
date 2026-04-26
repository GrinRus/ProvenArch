package runtime

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveHeadlessIncludeDirectoriesUsesManifestPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "arch-workspace")
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	for _, dir := range []string{workspace, repoA, repoB} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	manifest := "version: 1\nrepos:\n  - name: repo-a\n    path: " + repoA + "\n  - name: repo-b\n    path: " + repoB + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	got := ResolveHeadlessIncludeDirectories(Task{
		Workspace:  workspace,
		RepoScopes: []string{"repo-b"},
	})

	want := []string{workspace, repoB}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected include dirs:\n got=%v\nwant=%v", got, want)
	}
}

func TestResolveHeadlessIncludeDirectoriesCollectUsesWriteRootAndRepoScopes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "arch-workspace")
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "shards", "repo-b")
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	for _, dir := range []string{workspace, writeRoot, repoA, repoB} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	manifest := "version: 1\nrepos:\n  - name: repo-a\n    path: " + repoA + "\n  - name: repo-b\n    path: " + repoB + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	got := ResolveHeadlessIncludeDirectories(Task{
		StepID:     "init.step1.collect",
		Workspace:  workspace,
		WriteRoot:  writeRoot,
		RepoScopes: []string{"repo-b"},
	})

	want := []string{writeRoot, repoB}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected collect include dirs:\n got=%v\nwant=%v", got, want)
	}
}

func TestResolveHeadlessCollectRepairIncludeDirectoriesExcludesWorkspaceReports(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "arch-workspace")
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "shards", "repo-b")
	stagedFinal := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "final")
	repoB := filepath.Join(root, "repo-b")
	for _, dir := range []string{workspace, writeRoot, stagedFinal, repoB} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	manifest := "version: 1\nrepos:\n  - name: repo-b\n    path: " + repoB + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	got := ResolveHeadlessCollectRepairIncludeDirectories(Task{
		StepID:           "init.step1.collect",
		Workspace:        workspace,
		WriteRoot:        writeRoot,
		RepoScopes:       []string{"repo-b"},
		ReadContextRoots: []string{workspace, stagedFinal, repoB},
	})

	want := []string{writeRoot, repoB}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected repair include dirs:\n got=%v\nwant=%v", got, want)
	}
}

func TestResolveHeadlessIncludeDirectoriesFallsBackToWorkspaceValidate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "arch-workspace")
	resolvedRepo := filepath.Join(root, "resolved-repo")
	for _, dir := range []string{workspace, resolvedRepo} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	manifest := "version: 1\nrepos:\n  - name: repo-a\n    git_url: https://github.com/example/repo-a.git\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	validate := "{\n  \"resolved_repos\": [\n    {\"name\": \"repo-a\", \"path\": \"" + resolvedRepo + "\"},\n    {\"name\": \"repo-b\", \"path\": \"" + filepath.Join(root, "other-repo") + "\"}\n  ]\n}"
	if err := os.WriteFile(filepath.Join(root, "workspace-validate.json"), []byte(validate), 0o644); err != nil {
		t.Fatalf("write workspace-validate: %v", err)
	}

	got := ResolveHeadlessIncludeDirectories(Task{
		Workspace:  workspace,
		RepoScopes: []string{"repo-a"},
	})

	want := []string{workspace, resolvedRepo}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected include dirs:\n got=%v\nwant=%v", got, want)
	}
}

func TestResolveHeadlessIncludeDirectoriesUsesReadContextRootsWithoutRepoFallback(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "arch-workspace")
	stagedFinal := filepath.Join(workspace, "reports", "taskruns", "run-1", "staging", "final")
	validatorRoot := filepath.Join(workspace, "reports", "taskruns", "run-1", "validator")
	repoRoot := filepath.Join(root, "repo-a")
	for _, dir := range []string{workspace, stagedFinal, validatorRoot, repoRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	manifest := "version: 1\nrepos:\n  - name: repo-a\n    path: " + repoRoot + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	got := ResolveHeadlessIncludeDirectories(Task{
		Workspace:        workspace,
		RepoScopes:       []string{"repo-a"},
		ReadContextRoots: []string{workspace, stagedFinal, validatorRoot},
	})

	want := []string{workspace, stagedFinal, validatorRoot}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected include dirs with read_context_roots:\n got=%v\nwant=%v", got, want)
	}
}

func TestResolveHeadlessWorkingDirectoryUsesWriteRootForCollect(t *testing.T) {
	t.Parallel()

	task := Task{
		StepID:    "refresh.step1.collect",
		Workspace: "/tmp/workspace",
		WriteRoot: "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments",
	}

	if got, want := ResolveHeadlessWorkingDirectory(task), task.WriteRoot; got != want {
		t.Fatalf("unexpected working directory: got=%q want=%q", got, want)
	}
}

func TestResolveHeadlessWorkingDirectoryUsesDraftRootForDraftSteps(t *testing.T) {
	t.Parallel()

	task := Task{
		StepID:         "init.step2.asis_docs",
		Workspace:      "/tmp/workspace",
		WriteRoot:      "/tmp/workspace/reports/taskruns/run-1/step2",
		DraftFinalRoot: "/tmp/workspace/reports/taskruns/run-1/staging/final",
	}

	if got, want := ResolveHeadlessWorkingDirectory(task), task.DraftFinalRoot; got != want {
		t.Fatalf("unexpected working directory: got=%q want=%q", got, want)
	}
}

func TestResolveHeadlessWorkingDirectoryUsesWriteRootForValidator(t *testing.T) {
	t.Parallel()

	task := Task{
		StepID:         "refresh.step3.findings",
		Workspace:      "/tmp/workspace",
		WriteRoot:      "/tmp/workspace/reports/taskruns/run-1/validator",
		DraftFinalRoot: "/tmp/workspace/reports/taskruns/run-1/staging/final",
	}

	if got, want := ResolveHeadlessWorkingDirectory(task), task.WriteRoot; got != want {
		t.Fatalf("unexpected working directory: got=%q want=%q", got, want)
	}
}
