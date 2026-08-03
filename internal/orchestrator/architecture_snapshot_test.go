package orchestrator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
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
	index := contracts.FinalRunIndex{Version: 1, RunID: "run-1", Pipeline: "init", GeneratedAt: "2026-08-03T10:00:00Z", CitationIndexPath: "reports/taskruns/run-1/staging/final/citation-index.json", CanonicalDocuments: []contracts.FinalRunDocument{}, Topics: []contracts.TopicIndexEntry{}, Semantic: contracts.SemanticSnapshot{Coverage: contracts.Coverage{Observed: []string{"payments"}, Missing: []string{"owner"}, Notes: []string{}}, Questions: []contracts.Question{}, Entities: []contracts.Entity{}, Edges: []contracts.Edge{}, Findings: []contracts.Finding{{ID: "finding.owner", Severity: "medium", Title: "Owner unknown", Provenance: contracts.Provenance{Kind: "inference", Confidence: .7, Evidence: []contracts.Evidence{}}}}}}
	indexRaw, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("reports/taskruns/run-1/staging/final/final-run-index.json", indexRaw); err != nil {
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
	manifestRaw, err := os.ReadFile(filepath.Join(root, "reports", "taskruns", "run-1", "promoted-snapshot", "architecture-snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest architectureSnapshotManifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != 2 || len(manifest.Semantic.Findings) != 1 || manifest.Semantic.Coverage.Missing[0] != "owner" {
		t.Fatalf("semantic promoted snapshot is incomplete: %#v", manifest)
	}
}

func TestNoOpSnapshotInheritsBaselineSemanticAuthority(t *testing.T) {
	ws := workspace.Root{Path: t.TempDir()}
	if err := ws.WriteFile("model/entities/svc.payments.yaml", []byte("id: svc.payments\n")); err != nil {
		t.Fatal(err)
	}
	baseline := architectureSnapshotManifest{Version: 2, RunID: "baseline", Files: []architectureSnapshotManifestFile{}, Semantic: contracts.SemanticSnapshot{Coverage: contracts.Coverage{Missing: []string{"owner"}}, Questions: []contracts.Question{}, Entities: []contracts.Entity{}, Edges: []contracts.Edge{}, Findings: []contracts.Finding{{ID: "finding.owner", Severity: "medium", Title: "Owner missing", Provenance: contracts.Provenance{Kind: "inference", Confidence: .7}}}}}
	raw, err := json.Marshal(baseline)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("reports/taskruns/baseline/promoted-snapshot/architecture-snapshot.json", raw); err != nil {
		t.Fatal(err)
	}
	if _, err := persistPromotedArchitectureSnapshotFrom(ws, "noop", "baseline", time.Now()); err != nil {
		t.Fatal(err)
	}
	currentRaw, err := ws.ReadFile("reports/taskruns/noop/promoted-snapshot/architecture-snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var current architectureSnapshotManifest
	if err := json.Unmarshal(currentRaw, &current); err != nil {
		t.Fatal(err)
	}
	if len(current.Semantic.Findings) != 1 || current.Semantic.Findings[0].ID != "finding.owner" {
		t.Fatalf("baseline semantic authority was lost: %#v", current.Semantic)
	}
}
