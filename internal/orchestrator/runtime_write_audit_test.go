package orchestrator

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestRuntimeWriteAuditIgnoresStagedRootWrites(t *testing.T) {
	t.Parallel()

	ws := writeAuditWorkspace(t)
	task := writeAuditTask(ws, nil)
	execution, _ := newWriteAuditExecution(ws)

	before := beginRuntimeWriteAudit(task)
	if err := os.MkdirAll(task.WriteRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "diagnostic.txt"), []byte("staged only"), 0o644); err != nil {
		t.Fatalf("write staged file: %v", err)
	}
	if err := execution.completeRuntimeWriteAudit("init.step1.collect", "", acpruntime.ProviderCodexCode, task, before); err != nil {
		t.Fatalf("did not expect runtime write audit error for staged write: %v", err)
	}

	if hasWarningContaining(execution.warnings, runtimeWriteAuditUnexpectedMutation) {
		t.Fatalf("did not expect runtime write audit warning for staged write: %#v", execution.warnings)
	}
}

func TestRuntimeWriteAuditFailsOnProtectedWorkspaceMutation(t *testing.T) {
	t.Parallel()

	ws := writeAuditWorkspace(t)
	task := writeAuditTask(ws, nil)
	execution, logs := newWriteAuditExecution(ws)

	before := beginRuntimeWriteAudit(task)
	if err := os.WriteFile(filepath.Join(ws.Path, "workspace.yaml"), []byte("version: 1\nrepos: []\n# mutated\n"), 0o644); err != nil {
		t.Fatalf("mutate workspace manifest: %v", err)
	}
	err := execution.completeRuntimeWriteAudit("init.step1.collect", "", acpruntime.ProviderCodexCode, task, before)

	if !isRuntimeContractError(err) {
		t.Fatalf("expected runtime contract error, got %v", err)
	}
	if !hasWarningContaining(execution.warnings, runtimeWriteAuditUnexpectedMutation) {
		t.Fatalf("expected runtime write audit warning, got %#v", execution.warnings)
	}
	if !hasLogWithMessage(logs, runtimeWriteAuditUnexpectedMutation) {
		t.Fatalf("expected runtime write audit log, got %#v", logs)
	}
	restored, err := os.ReadFile(filepath.Join(ws.Path, "workspace.yaml"))
	if err != nil {
		t.Fatalf("read restored workspace manifest: %v", err)
	}
	if string(restored) != "version: 1\nrepos: []\n" {
		t.Fatalf("expected protected workspace mutation to be restored, got %q", restored)
	}
	if !hasLogWithMessage(logs, runtimeWriteAuditRestoredMutation) {
		t.Fatalf("expected restore log, got %#v", logs)
	}
}

func TestRuntimeWriteAuditDoesNotOverwritePostRunConflict(t *testing.T) {
	t.Parallel()

	ws := writeAuditWorkspace(t)
	task := writeAuditTask(ws, nil)
	execution, logs := newWriteAuditExecution(ws)

	before := beginRuntimeWriteAudit(task)
	path := filepath.Join(ws.Path, "workspace.yaml")
	if err := os.WriteFile(path, []byte("provider mutation\n"), 0o644); err != nil {
		t.Fatalf("mutate workspace manifest: %v", err)
	}
	after := snapshotProtectedWorkspaceFiles(ws.Path)
	if err := os.WriteFile(path, []byte("external edit\n"), 0o644); err != nil {
		t.Fatalf("write post-run conflict: %v", err)
	}
	execution.restoreRuntimeWriteAuditMutations("init.step1.collect", "", task, before.protectedFiles, after, []string{"workspace.yaml"})

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read conflicted workspace manifest: %v", err)
	}
	if string(content) != "external edit\n" {
		t.Fatalf("restore must not overwrite post-run conflict, got %q", content)
	}
	if !hasWarningContaining(execution.warnings, runtimeWriteAuditRestoreConflict) {
		t.Fatalf("expected restore conflict warning, got %#v", execution.warnings)
	}
	if !hasLogWithMessage(logs, runtimeWriteAuditRestoreConflict) {
		t.Fatalf("expected restore conflict log, got %#v", logs)
	}
}

func TestRuntimeWriteAuditRestoresProtectedFileMode(t *testing.T) {
	t.Parallel()

	ws := writeAuditWorkspace(t)
	task := writeAuditTask(ws, nil)
	execution, _ := newWriteAuditExecution(ws)
	path := filepath.Join(ws.Path, "workspace.yaml")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("set baseline mode: %v", err)
	}
	before := beginRuntimeWriteAudit(task)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("mutate workspace mode: %v", err)
	}
	if err := execution.completeRuntimeWriteAudit("init.step1.collect", "", acpruntime.ProviderCodexCode, task, before); err == nil {
		t.Fatal("expected protected mode mutation to fail runtime audit")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat restored workspace manifest: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Fatalf("expected mode %04o after restore, got %04o", want, got)
	}
}

func TestRuntimeWriteAuditFailsOnRepoMutation(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}

	ws := writeAuditWorkspace(t)
	repoRoot := writeAuditGitRepo(t)
	task := writeAuditTask(ws, []string{repoRoot})
	execution, logs := newWriteAuditExecution(ws)

	before := beginRuntimeWriteAudit(task)
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# changed\n"), 0o644); err != nil {
		t.Fatalf("mutate repo file: %v", err)
	}
	err := execution.completeRuntimeWriteAudit("init.step1.collect", "", acpruntime.ProviderCodexCode, task, before)

	if !isRuntimeContractError(err) {
		t.Fatalf("expected runtime contract error, got %v", err)
	}
	if !hasWarningContaining(execution.warnings, runtimeWriteAuditUnexpectedMutation) {
		t.Fatalf("expected repo mutation warning, got %#v", execution.warnings)
	}
	if !hasLogField(logs, "category", "repo") {
		t.Fatalf("expected repo mutation log category, got %#v", logs)
	}
}

func TestRuntimeWriteAuditAllowsReadOnlyRepoSnapshot(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}

	ws := writeAuditWorkspace(t)
	repoRoot := writeAuditGitRepo(t)
	t.Cleanup(func() {
		chmodTreeForAudit(t, repoRoot, 0o755, 0o644)
	})
	chmodTreeForAudit(t, repoRoot, 0o555, 0o444)
	task := writeAuditTask(ws, []string{repoRoot})
	execution, logs := newWriteAuditExecution(ws)

	before := beginRuntimeWriteAudit(task)
	var snapshot []string
	for _, status := range before.repoStatuses {
		snapshot = status
	}
	if len(snapshot) == 0 {
		t.Fatal("expected read-only repo snapshot")
	}
	if !stringSliceContainsPrefix(snapshot, "readonly:head:") {
		t.Fatalf("expected read-only snapshot, got %#v", snapshot)
	}
	err := execution.completeRuntimeWriteAudit("init.step1.collect", "", acpruntime.ProviderCodexCode, task, before)

	if err != nil {
		t.Fatalf("did not expect runtime write audit error for unchanged read-only repo: %v", err)
	}
	if hasWarningContaining(execution.warnings, runtimeWriteAuditUnexpectedMutation) || hasWarningContaining(execution.warnings, runtimeWriteAuditRepoSkipped) {
		t.Fatalf("did not expect read-only repo audit warnings, got %#v", execution.warnings)
	}
	if hasLogWithMessage(logs, runtimeWriteAuditUnexpectedMutation) || hasLogWithMessage(logs, runtimeWriteAuditRepoSkipped) {
		t.Fatalf("did not expect read-only repo audit logs, got %#v", logs)
	}
}

func TestRuntimeWriteAuditFailsWhenAuditedRepoStatusBecomesUnavailable(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}

	ws := writeAuditWorkspace(t)
	repoRoot := writeAuditGitRepo(t)
	task := writeAuditTask(ws, []string{repoRoot})
	execution, logs := newWriteAuditExecution(ws)

	before := beginRuntimeWriteAudit(task)
	if len(before.repoStatuses) == 0 {
		t.Fatal("expected audited repo status before runtime")
	}
	if err := os.RemoveAll(repoRoot); err != nil {
		t.Fatalf("remove repo root: %v", err)
	}
	err := execution.completeRuntimeWriteAudit("init.step1.collect", "", acpruntime.ProviderCodexCode, task, before)

	if !isRuntimeContractError(err) {
		t.Fatalf("expected runtime contract error, got %v", err)
	}
	if hasWarningContaining(execution.warnings, runtimeWriteAuditUnexpectedMutation) {
		t.Fatalf("unavailable repo status must not be reported as mutation: %#v", execution.warnings)
	}
	if !hasWarningContaining(execution.warnings, runtimeWriteAuditRepoSkipped) {
		t.Fatalf("expected repo skipped warning, got %#v", execution.warnings)
	}
	if !hasLogField(logs, "reason", "status_unavailable_after_runtime") {
		t.Fatalf("expected unavailable-after-runtime reason, got %#v", logs)
	}
	if hasLogField(logs, "changed_paths", "po status unavailable after runtime>") {
		t.Fatalf("unexpected truncated unavailable sentinel in changed_paths: %#v", logs)
	}
}

func TestChangedRepoStatusPathsOnlyStripsGitPorcelainPrefix(t *testing.T) {
	t.Parallel()

	paths := changedRepoStatusPaths(nil, []string{
		" M README.md",
		"?? notes.md",
		"<repo status unavailable after runtime>",
	})

	if !stringSliceContains(paths, "README.md") || !stringSliceContains(paths, "notes.md") {
		t.Fatalf("expected git porcelain paths, got %#v", paths)
	}
	if !stringSliceContains(paths, "<repo status unavailable after runtime>") {
		t.Fatalf("expected non-porcelain sentinel to stay intact, got %#v", paths)
	}
	if stringSliceContains(paths, "po status unavailable after runtime>") {
		t.Fatalf("sentinel must not be trimmed like porcelain output, got %#v", paths)
	}
}

func TestRuntimeWriteAuditLogsNonGitRepoSkip(t *testing.T) {
	t.Parallel()

	ws := writeAuditWorkspace(t)
	nonGitRoot := t.TempDir()
	task := writeAuditTask(ws, []string{nonGitRoot})
	execution, logs := newWriteAuditExecution(ws)

	before := beginRuntimeWriteAudit(task)
	if err := execution.completeRuntimeWriteAudit("init.step1.collect", "", acpruntime.ProviderCodexCode, task, before); err != nil {
		t.Fatalf("did not expect runtime write audit error for initial non-git skip: %v", err)
	}

	if !hasWarningContaining(execution.warnings, runtimeWriteAuditRepoSkipped) {
		t.Fatalf("expected repo skipped warning, got %#v", execution.warnings)
	}
	if !hasLogWithMessage(logs, runtimeWriteAuditRepoSkipped) {
		t.Fatalf("expected repo skipped log, got %#v", logs)
	}
}

func writeAuditWorkspace(t *testing.T) workspace.Root {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "charter"), 0o755); err != nil {
		t.Fatalf("mkdir charter: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte("version: 1\nrepos: []\n"), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "charter", "overview.md"), []byte("# Charter\n"), 0o644); err != nil {
		t.Fatalf("write charter overview: %v", err)
	}
	return workspace.Root{Path: root}
}

func writeAuditTask(ws workspace.Root, extraReadRoots []string) acpruntime.Task {
	writeRoot := filepath.Join(ws.Path, "reports", "taskruns", "run-1", "staging", "shards", "shard-1")
	draftRoot := filepath.Join(ws.Path, "reports", "taskruns", "run-1", "drafts", "step1")
	readRoots := append([]string{ws.Path}, extraReadRoots...)
	return acpruntime.Task{
		TaskID:           "task-1",
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        ws.Path,
		WriteRoot:        writeRoot,
		DraftFinalRoot:   draftRoot,
		ReadContextRoots: readRoots,
	}
}

func newWriteAuditExecution(ws workspace.Root) (*pipelineExecution, *[]RunLogEntry) {
	logs := []RunLogEntry{}
	execution := &pipelineExecution{
		workspace: ws,
		pipelineRunProgressState: pipelineRunProgressState{
			onLog: func(entry RunLogEntry) {
				logs = append(logs, entry)
			},
		},
	}
	execution.clock = func() time.Time {
		return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	}
	return execution, &logs
}

func writeAuditGitRepo(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()
	runGitForAudit(t, repoRoot, "init")
	runGitForAudit(t, repoRoot, "config", "user.email", "test@example.invalid")
	runGitForAudit(t, repoRoot, "config", "user.name", "ACP Test")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# repo\n"), 0o644); err != nil {
		t.Fatalf("write repo readme: %v", err)
	}
	runGitForAudit(t, repoRoot, "add", "README.md")
	runGitForAudit(t, repoRoot, "commit", "-m", "initial")
	return repoRoot
}

func runGitForAudit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

func chmodTreeForAudit(t *testing.T, root string, dirMode fs.FileMode, fileMode fs.FileMode) {
	t.Helper()

	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil {
			return nil
		}
		mode := fileMode
		if entry.IsDir() {
			mode = dirMode
		}
		_ = os.Chmod(path, mode)
		return nil
	})
}

func hasLogWithMessage(logs *[]RunLogEntry, message string) bool {
	if logs == nil {
		return false
	}
	for _, entry := range *logs {
		if entry.Message == message {
			return true
		}
	}
	return false
}

func hasLogField(logs *[]RunLogEntry, key string, value string) bool {
	if logs == nil {
		return false
	}
	for _, entry := range *logs {
		if entry.Fields == nil {
			continue
		}
		if got, ok := entry.Fields[key].(string); ok && got == value {
			return true
		}
	}
	return false
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringSliceContainsPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func isRuntimeContractError(err error) bool {
	code, _, ok := acpruntime.ClassifyError(err)
	return ok && code == string(acpruntime.ErrorCodeRuntimeContract)
}
