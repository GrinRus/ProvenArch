package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestPersistPromotedArchitectureSnapshotCopiesOnlyArchitectureRoots(t *testing.T) {
	root := t.TempDir()
	ws := workspace.Root{Path: root}
	if err := ws.WriteFile("model/entities/svc.payments.yaml", []byte("id: svc.payments\n")); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("reports/as-is/overview.md", []byte("# Payments\n")); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("reports/taskruns/foreign/raw.log", []byte("must not be copied\n")); err != nil {
		t.Fatal(err)
	}
	artifact, err := persistPromotedArchitectureSnapshot(ws, "run-1", time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if artifact == nil || artifact.Kind != architectureSnapshotKind {
		t.Fatalf("unexpected snapshot artifact: %#v", artifact)
	}
	if _, err := os.Stat(filepath.Join(root, "reports", "taskruns", "run-1", "promoted-snapshot", "model", "entities", "svc.payments.yaml")); err != nil {
		t.Fatalf("promoted model was not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "reports", "taskruns", "run-1", "promoted-snapshot", "reports", "taskruns", "foreign", "raw.log")); !os.IsNotExist(err) {
		t.Fatalf("taskrun history leaked into promoted snapshot: %v", err)
	}
}
