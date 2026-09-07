package api

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestParseGitStatusV2FilesPreservesMetadataAndSpecialPaths(t *testing.T) {
	headOID := strings.Repeat("a", 40)
	indexOID := strings.Repeat("b", 40)
	output := strings.Join([]string{
		"1 M. N... 100644 100644 100644 " + headOID + " " + indexOID + " docs/file with spaces.md",
		"2 R. N... 100644 100644 100644 " + headOID + " " + indexOID + " R100 docs/new name.md",
		"docs/old name.md",
		"2 C. N... 100644 100644 100644 " + headOID + " " + indexOID + " C100 docs/copy.md",
		"docs/file with spaces.md",
		"? untracked file.md",
		"u UU N... 100644 100644 100644 100644 " + headOID + " " + indexOID + " " + strings.Repeat("c", 40) + " conflict.md",
	}, "\x00") + "\x00"

	files := parseGitStatusV2Files(output)
	if len(files) != 5 {
		t.Fatalf("expected five status records, got %+v", files)
	}
	modified := files[0]
	if modified.Path != "docs/file with spaces.md" || modified.Status != "modified" || modified.IndexStatus != "M" || modified.WorktreeStatus != "clean" {
		t.Fatalf("unexpected modified record: %+v", modified)
	}
	if modified.OldMode != "100644" || modified.NewMode != "100644" || modified.HeadOID != headOID || modified.IndexOID != indexOID {
		t.Fatalf("expected v2 modes and OIDs, got %+v", modified)
	}
	rename := files[1]
	if rename.Status != "renamed" || rename.Path != "docs/new name.md" || rename.OriginalPath == nil || *rename.OriginalPath != "docs/old name.md" {
		t.Fatalf("expected rename source/path, got %+v", rename)
	}
	copyFile := files[2]
	if copyFile.Status != "copied" || copyFile.Path != "docs/copy.md" || copyFile.OriginalPath == nil || *copyFile.OriginalPath != "docs/file with spaces.md" {
		t.Fatalf("expected copy source/path, got %+v", copyFile)
	}
	untracked := files[3]
	if untracked.Status != "untracked" || untracked.IndexStatus != "untracked" || untracked.WorktreeStatus != "untracked" || untracked.Path != "untracked file.md" {
		t.Fatalf("unexpected untracked record: %+v", untracked)
	}
	unmerged := files[4]
	if unmerged.Status != "changed" || unmerged.WorktreeStatus != "U" || unmerged.OldMode != "100644" || unmerged.HeadOID != headOID {
		t.Fatalf("unexpected unmerged record: %+v", unmerged)
	}
}

func TestParseGitNumstatHandlesBinaryAndPathsWithTabs(t *testing.T) {
	output := "3\t1\tdocs/changed.md\x00-\t-\tassets/file\twith-tab.bin\x00"
	stats := parseGitNumstat(output)
	if stats["docs/changed.md"].Additions != 3 || stats["docs/changed.md"].Deletions != 1 {
		t.Fatalf("unexpected text numstat: %+v", stats["docs/changed.md"])
	}
	if !stats["assets/file\twith-tab.bin"].Binary {
		t.Fatalf("expected binary numstat, got %+v", stats["assets/file\twith-tab.bin"])
	}
}

func TestGitDiffInventoryPreservesRenameAndDestinationNumstat(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for git inventory tests")
	}
	root := t.TempDir()
	ws := workspace.Root{Path: root}
	runGitInventoryTestCommand(t, root, "init")
	runGitInventoryTestCommand(t, root, "config", "user.email", "acp-test@example.test")
	runGitInventoryTestCommand(t, root, "config", "user.name", "ACP Test")
	if err := os.WriteFile(filepath.Join(root, "old.md"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInventoryTestCommand(t, root, "add", "-A")
	runGitInventoryTestCommand(t, root, "commit", "-m", "baseline")
	runGitInventoryTestCommand(t, root, "mv", "old.md", "new.md")
	if err := os.WriteFile(filepath.Join(root, "new.md"), []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := collectWorkspaceGitState(context.Background(), ws)
	if err != nil {
		t.Fatalf("collect inventory: %v", err)
	}
	if len(state.Files) != 1 {
		t.Fatalf("expected one rename record, got %+v", state.Files)
	}
	file := state.Files[0]
	if file.Status != "renamed" || file.Path != "new.md" || file.OriginalPath == nil || *file.OriginalPath != "old.md" {
		t.Fatalf("expected rename inventory record, got %+v", file)
	}
	if file.Additions != 3 || file.Deletions != 0 {
		t.Fatalf("expected destination-side rename line counts 3/0, got additions=%d deletions=%d", file.Additions, file.Deletions)
	}
}

func TestGitWorkspaceInventoryRepresentativeCommandBudget(t *testing.T) {
	ws := prepareGitInventoryWorkspace(t, 275)
	commands := 0
	runner := func(ctx context.Context, repoPath string, args ...string) (string, error) {
		commands++
		return runGitRaw(ctx, repoPath, args...)
	}
	state, err := collectWorkspaceGitStateWithRunner(context.Background(), ws, runner)
	if err != nil {
		t.Fatalf("collect representative inventory: %v", err)
	}
	if len(state.Files) != 275 {
		t.Fatalf("expected 275 changed files, got %d", len(state.Files))
	}
	if commands > 8 {
		t.Fatalf("Git command budget exceeded: got %d commands for 275 files", commands)
	}
}

func BenchmarkGitWorkspaceInventoryRepresentative(b *testing.B) {
	ws := prepareGitInventoryWorkspace(b, 275)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		commands := 0
		runner := func(ctx context.Context, repoPath string, args ...string) (string, error) {
			commands++
			return runGitRaw(ctx, repoPath, args...)
		}
		if _, err := collectWorkspaceGitStateWithRunner(context.Background(), ws, runner); err != nil {
			b.Fatalf("collect representative inventory: %v", err)
		}
		b.ReportMetric(float64(commands), "git-procs")
	}
}

func prepareGitInventoryWorkspace(t testing.TB, count int) workspace.Root {
	t.Helper()
	root := t.TempDir()
	for i := 0; i < count; i++ {
		rel := filepath.Join("reports", "generated", fmt.Sprintf("file-%03d.md", i))
		fullPath := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("create fixture directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(fmt.Sprintf("baseline %03d\n", i)), 0o644); err != nil {
			t.Fatalf("write baseline file: %v", err)
		}
	}
	runGitInventoryTestCommand(t, root, "init")
	runGitInventoryTestCommand(t, root, "config", "user.email", "acp-test@example.test")
	runGitInventoryTestCommand(t, root, "config", "user.name", "ACP Test")
	runGitInventoryTestCommand(t, root, "add", "-A")
	runGitInventoryTestCommand(t, root, "commit", "-m", "baseline")
	for i := 0; i < count; i++ {
		rel := filepath.Join(root, "reports", "generated", fmt.Sprintf("file-%03d.md", i))
		if err := os.WriteFile(rel, []byte(fmt.Sprintf("baseline %03d\nchanged\n", i)), 0o644); err != nil {
			t.Fatalf("write changed file: %v", err)
		}
	}
	return workspace.Root{Path: root}
}

func runGitInventoryTestCommand(t testing.TB, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
}
