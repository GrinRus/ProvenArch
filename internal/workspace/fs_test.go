package workspace

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomicKeepsPreviousContentBeforeRenameFailure(t *testing.T) {
	ws := Root{Path: t.TempDir()}
	rel := "reports/taskruns/state.json"
	if err := ws.WriteFileAtomic(rel, []byte("old\n")); err != nil {
		t.Fatalf("write initial state: %v", err)
	}

	restore := setAtomicWriteFaultHookForTest(func(point atomicWriteFaultPoint) error {
		if point == atomicWriteFaultBeforeRename {
			return errors.New("injected before rename")
		}
		return nil
	})
	defer restore()

	err := ws.WriteFileAtomic(rel, []byte("new\n"))
	if err == nil || !strings.Contains(err.Error(), "injected before rename") {
		t.Fatalf("expected injected rename failure, got %v", err)
	}

	raw, readErr := os.ReadFile(filepath.Join(ws.Path, rel))
	if readErr != nil {
		t.Fatalf("read target after failed write: %v", readErr)
	}
	if string(raw) != "old\n" {
		t.Fatalf("expected previous content after failed atomic write, got %q", string(raw))
	}
	assertNoAtomicTemps(t, filepath.Join(ws.Path, "reports/taskruns"), "state.json")
}

func TestWriteFileAtomicLeavesNoTargetBeforeWriteFailure(t *testing.T) {
	ws := Root{Path: t.TempDir()}
	rel := "reports/taskruns/state.json"

	restore := setAtomicWriteFaultHookForTest(func(point atomicWriteFaultPoint) error {
		if point == atomicWriteFaultBeforeWrite {
			return errors.New("injected before write")
		}
		return nil
	})
	defer restore()

	err := ws.WriteFileAtomic(rel, []byte("new\n"))
	if err == nil || !strings.Contains(err.Error(), "injected before write") {
		t.Fatalf("expected injected write failure, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(ws.Path, rel)); !os.IsNotExist(statErr) {
		t.Fatalf("expected target to be absent after before-write failure, stat err=%v", statErr)
	}
	assertNoAtomicTemps(t, filepath.Join(ws.Path, "reports/taskruns"), "state.json")
}

func TestWriteFileAtomicSurfacesParentSyncFailure(t *testing.T) {
	ws := Root{Path: t.TempDir()}
	rel := "reports/taskruns/state.json"

	restore := setAtomicWriteFaultHookForTest(func(point atomicWriteFaultPoint) error {
		if point == atomicWriteFaultBeforeDirSync {
			return errors.New("injected parent sync failure")
		}
		return nil
	})
	defer restore()

	err := ws.WriteFileAtomic(rel, []byte("new\n"))
	if err == nil || !strings.Contains(err.Error(), "injected parent sync failure") {
		t.Fatalf("expected parent sync failure, got %v", err)
	}
}

func TestWriteFileAtomicWithLastGoodCanRecoverMissingCurrent(t *testing.T) {
	ws := Root{Path: t.TempDir()}
	rel := "reports/taskruns/run-history.json"
	content := []byte("{\"version\":1,\"items\":[]}\n")
	if err := ws.WriteFileAtomicWithLastGood(rel, content); err != nil {
		t.Fatalf("write current and last-good: %v", err)
	}
	if err := os.Remove(filepath.Join(ws.Path, rel)); err != nil {
		t.Fatalf("remove current: %v", err)
	}

	raw, recovered, err := ws.ReadFileWithLastGood(rel)
	if err != nil {
		t.Fatalf("read with last-good fallback: %v", err)
	}
	if !recovered {
		t.Fatal("expected fallback to last-good")
	}
	if string(raw) != string(content) {
		t.Fatalf("expected last-good content %q, got %q", string(content), string(raw))
	}
}

func TestWorkspaceReadWriteRejectSymlinkEscape(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws := Root{Path: t.TempDir()}
	if err := os.Symlink(outside, filepath.Join(ws.Path, "escape")); err != nil {
		t.Fatal(err)
	}

	if _, err := ws.ReadFile("escape/secret.txt"); err == nil {
		t.Fatal("expected escaped read to fail")
	}
	if err := ws.WriteFileAtomic("escape/created.txt", []byte("unsafe\n")); err == nil {
		t.Fatal("expected escaped write to fail")
	}
	if _, err := os.Stat(filepath.Join(outside, "created.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside target changed, stat err=%v", err)
	}
}

func TestWorkspaceReadWriteAllowsRelativeInRootSymlink(t *testing.T) {
	ws := Root{Path: t.TempDir()}
	if err := os.Mkdir(filepath.Join(ws.Path, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(ws.Path, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFileAtomic("alias/value.txt", []byte("inside\n")); err != nil {
		t.Fatalf("write through in-root symlink: %v", err)
	}
	raw, err := ws.ReadFile("alias/value.txt")
	if err != nil {
		t.Fatalf("read through in-root symlink: %v", err)
	}
	if string(raw) != "inside\n" {
		t.Fatalf("unexpected content %q", raw)
	}
}

func TestReadRegularTreeRejectsSymlinkAndReturnsDeterministicCopies(t *testing.T) {
	ws := Root{Path: t.TempDir()}
	if err := ws.WriteFileAtomic("tree/b.txt", []byte("b")); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFileAtomic("tree/a.txt", []byte("a")); err != nil {
		t.Fatal(err)
	}

	files, err := ws.ReadRegularTree("tree")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Path != "a.txt" || files[1].Path != "b.txt" {
		t.Fatalf("unexpected deterministic tree: %#v", files)
	}
	files[0].Content[0] = 'x'
	raw, err := ws.ReadFile("tree/a.txt")
	if err != nil || string(raw) != "a" {
		t.Fatalf("tree result must be an immutable copy: raw=%q err=%v", raw, err)
	}

	if err := os.Symlink("../tree/a.txt", filepath.Join(ws.Path, "tree", "alias.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ReadRegularTree("tree"); !errors.Is(err, ErrPathSymlink) {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func TestWorkspaceReadWriteRejectsDanglingAndAbsoluteSymlink(t *testing.T) {
	ws := Root{Path: t.TempDir()}
	if err := os.Symlink("missing", filepath.Join(ws.Path, "dangling")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(ws.Path, "absolute")); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"dangling/value.txt", "absolute/value.txt"} {
		if err := ws.WriteFileAtomic(rel, []byte("unsafe\n")); err == nil {
			t.Fatalf("expected write through %q to fail", rel)
		}
		if _, err := ws.Resolve(rel); err == nil {
			t.Fatalf("expected resolve through %q to fail", rel)
		}
	}
}

func TestReadFileLimitRejectsOversizedContent(t *testing.T) {
	ws := Root{Path: t.TempDir()}
	if err := ws.WriteFileAtomic("reports/value.txt", []byte("12345")); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ReadFileLimit("reports/value.txt", 4); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}
	raw, err := ws.ReadFileLimit("reports/value.txt", 5)
	if err != nil || string(raw) != "12345" {
		t.Fatalf("expected exact bounded read, raw=%q err=%v", raw, err)
	}
}

func TestWriteDirectoryAtomicExclusiveRollsBackWriteAndRenameFailures(t *testing.T) {
	ws := Root{Path: t.TempDir()}
	files := map[string][]byte{"proposal.md": []byte("# Proposal\n"), "source.json": []byte("{}\n")}
	for _, fault := range []atomicWriteFaultPoint{atomicWriteFaultBeforeWrite, atomicWriteFaultBeforeRename} {
		restore := setAtomicWriteFaultHookForTest(func(point atomicWriteFaultPoint) error {
			if point == fault {
				return errors.New("injected directory publication failure")
			}
			return nil
		})
		err := ws.WriteDirectoryAtomicExclusive("proposals/example", files)
		restore()
		if err == nil {
			t.Fatalf("expected injected %s failure", fault)
		}
		if _, statErr := os.Stat(filepath.Join(ws.Path, "proposals", "example")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("partial destination after %s: %v", fault, statErr)
		}
		matches, globErr := filepath.Glob(filepath.Join(ws.Path, "proposals", "example.tmp-*"))
		if globErr != nil || len(matches) != 0 {
			t.Fatalf("temporary directory leaked after %s: %v err=%v", fault, matches, globErr)
		}
	}
	if err := ws.WriteDirectoryAtomicExclusive("proposals/example", files); err != nil {
		t.Fatalf("publish complete directory: %v", err)
	}
	if err := ws.WriteDirectoryAtomicExclusive("proposals/example", files); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("expected exclusive destination error, got %v", err)
	}
}

func assertNoAtomicTemps(t *testing.T, dir string, base string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "."+base+".tmp-*"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) > 0 {
		t.Fatalf("expected no atomic temp files, got %v", matches)
	}
}
