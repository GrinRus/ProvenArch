package workspace

import (
	"errors"
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
