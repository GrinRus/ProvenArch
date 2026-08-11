package tasks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestNewRegistryStartsEmptyWhenNoHistoryExists(t *testing.T) {
	registry, err := NewRegistry(workspace.Root{Path: t.TempDir()}, fixedClock())
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	snapshot := registry.Snapshot()
	if snapshot.Version != CurrentVersion || len(snapshot.Tasks) != 0 || len(snapshot.Attempts) != 0 {
		t.Fatalf("unexpected empty history: %+v", snapshot)
	}
	if diagnostics := registry.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("missing history should not produce recovery diagnostics: %v", diagnostics)
	}
}

func TestRegistryRecoversMalformedCurrentFromLastGood(t *testing.T) {
	root := workspace.Root{Path: t.TempDir()}
	valid := readHistoryFixture(t)
	raw, err := MarshalHistory(valid)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := root.WriteFileAtomic(HistoryPath+".last-good", raw); err != nil {
		t.Fatalf("write last-good: %v", err)
	}
	if err := root.WriteFileAtomic(HistoryPath, []byte("{malformed")); err != nil {
		t.Fatalf("write malformed current: %v", err)
	}

	registry, err := NewRegistry(root, fixedClock())
	if err != nil {
		t.Fatalf("recover registry: %v", err)
	}
	if len(registry.Snapshot().Tasks) != 1 {
		t.Fatalf("expected recovered task, got %+v", registry.Snapshot())
	}
	diagnostics := strings.Join(registry.Diagnostics(), " ")
	if !strings.Contains(diagnostics, "last-good") || !strings.Contains(diagnostics, "unavailable") {
		t.Fatalf("expected bounded recovery diagnostics, got %v", diagnostics)
	}
}

func TestNewRegistryFailsWhenCurrentAndLastGoodAreInvalid(t *testing.T) {
	root := workspace.Root{Path: t.TempDir()}
	if err := root.WriteFileAtomic(HistoryPath, []byte("{malformed")); err != nil {
		t.Fatalf("write current: %v", err)
	}
	if err := root.WriteFileAtomic(HistoryPath+".last-good", []byte("{also-malformed")); err != nil {
		t.Fatalf("write last-good: %v", err)
	}
	if _, err := NewRegistry(root, fixedClock()); err == nil || !strings.Contains(err.Error(), "current=") || !strings.Contains(err.Error(), "last_good=") {
		t.Fatalf("expected both-source failure, got %v", err)
	}
}

func TestRegistryPrimaryWriteFailureDoesNotPublishCandidate(t *testing.T) {
	root := workspace.Root{Path: t.TempDir()}
	registry, err := NewRegistry(root, fixedClock())
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	base := registry.Snapshot()
	if err := registry.Update(func(history *History) error {
		history.Diagnostics = []string{"base"}
		return nil
	}); err != nil {
		t.Fatalf("persist base history: %v", err)
	}
	before := registry.Snapshot()
	registry.writeFile = func(_ workspace.Root, path string, _ []byte) error {
		if path == HistoryPath {
			return errors.New("primary write fault")
		}
		return nil
	}
	if err := registry.Update(func(history *History) error {
		history.Diagnostics = []string{"candidate"}
		return nil
	}); err == nil || !strings.Contains(err.Error(), "primary write fault") {
		t.Fatalf("expected primary write error, got %v", err)
	}
	after := registry.Snapshot()
	if !EqualJSON(before, after) || EqualJSON(base, after) {
		t.Fatalf("unexpected in-memory state after failed primary write: before=%+v after=%+v", before, after)
	}
	if stringMustRead(t, filepath.Join(root.Path, HistoryPath)) == "" {
		t.Fatal("expected durable previous current history")
	}
}

func TestRegistryBackupWriteFailurePublishesPrimaryAndDiagnostic(t *testing.T) {
	root := workspace.Root{Path: t.TempDir()}
	registry, err := NewRegistry(root, fixedClock())
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	registry.writeFile = func(root workspace.Root, path string, content []byte) error {
		if path == HistoryPath+".last-good" {
			return errors.New("backup write fault")
		}
		return root.WriteFileAtomic(path, content)
	}
	if err := registry.Update(func(history *History) error {
		history.Diagnostics = []string{"published"}
		return nil
	}); err != nil {
		t.Fatalf("primary write should succeed despite backup fault: %v", err)
	}
	if got := registry.Snapshot().Diagnostics; len(got) != 1 || got[0] != "published" {
		t.Fatalf("candidate was not published: %v", got)
	}
	if diagnostics := strings.Join(registry.Diagnostics(), " "); !strings.Contains(diagnostics, "last-good") {
		t.Fatalf("expected backup diagnostic, got %v", diagnostics)
	}
	if _, err := root.ReadFile(HistoryPath); err != nil {
		t.Fatalf("primary file missing after backup fault: %v", err)
	}
}

func TestRegistryConcurrentReadersAndWriters(t *testing.T) {
	root := workspace.Root{Path: t.TempDir()}
	registry, err := NewRegistry(root, fixedClock())
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	var group sync.WaitGroup
	for index := 0; index < 8; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			if err := registry.Update(func(history *History) error {
				history.Diagnostics = []string{time.Now().UTC().Format(time.RFC3339Nano), string(rune('a' + index))}
				return nil
			}); err != nil {
				t.Errorf("update %d: %v", index, err)
			}
		}(index)
	}
	for index := 0; index < 8; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if snapshot := registry.Snapshot(); snapshot.Version != CurrentVersion {
				t.Errorf("unexpected snapshot version %d", snapshot.Version)
			}
		}()
	}
	group.Wait()
	if err := registry.Snapshot().Validate(); err != nil {
		t.Fatalf("concurrent registry left invalid snapshot: %v", err)
	}
}

func readHistoryFixture(t *testing.T) History {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "tasks", "task-history.json"))
	if err != nil {
		t.Fatalf("read history fixture: %v", err)
	}
	value, err := ParseHistory(raw)
	if err != nil {
		t.Fatalf("parse history fixture: %v", err)
	}
	return value
}

func fixedClock() func() time.Time {
	return func() time.Time { return time.Date(2026, time.August, 11, 10, 5, 0, 0, time.UTC) }
}

func stringMustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}
