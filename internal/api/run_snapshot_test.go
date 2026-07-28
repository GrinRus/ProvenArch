package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestResolveRunSnapshotReturnsServerOwnedInventory(t *testing.T) {
	ws, indexPath, documentPath, citationPath := writeSnapshotFixture(t, "run-selected")
	artifacts := []orchestrator.Artifact{
		{Path: indexPath},
		{Path: documentPath},
		{Path: citationPath},
	}

	response := resolveRunSnapshot(ws, "run-selected", artifacts)
	if response.Status != runSnapshotAvailable {
		t.Fatalf("status = %q, issues=%+v", response.Status, response.Issues)
	}
	if len(response.Artifacts) != 3 {
		t.Fatalf("artifacts = %+v", response.Artifacts)
	}
	document := response.Artifacts[1]
	if document.Path != "reports/agent-outputs/domains/payments.md" || document.ReadPath != documentPath {
		t.Fatalf("unexpected resolved document: %+v", document)
	}
}

func TestResolveRunSnapshotRejectsForeignRunIdentity(t *testing.T) {
	ws, indexPath, _, _ := writeSnapshotFixture(t, "run-foreign")
	if err := ws.WriteFileAtomic(runFinalIndexPath("run-selected"), mustReadWorkspaceFile(t, ws, indexPath)); err != nil {
		t.Fatal(err)
	}
	response := resolveRunSnapshot(ws, "run-selected", []orchestrator.Artifact{{Path: runFinalIndexPath("run-selected")}})
	if response.Status != runSnapshotError || response.Issues[0].Code != "snapshot_run_mismatch" {
		t.Fatalf("expected run mismatch, got %+v", response)
	}
}

func TestResolveRunSnapshotDistinguishesNotProducedAndPartial(t *testing.T) {
	ws, indexPath, _, citationPath := writeSnapshotFixture(t, "run-selected")
	notProduced := resolveRunSnapshot(ws, "run-selected", nil)
	if notProduced.Status != runSnapshotNotProduced {
		t.Fatalf("not-produced status = %q", notProduced.Status)
	}

	partial := resolveRunSnapshot(ws, "run-selected", []orchestrator.Artifact{
		{Path: indexPath},
		{Path: citationPath},
	})
	if partial.Status != runSnapshotPartial || partial.Issues[0].Code != "snapshot_document_not_in_inventory" {
		t.Fatalf("expected partial missing inventory, got %+v", partial)
	}
}

func TestResolveRunSnapshotRejectsTraversalAndMissingIndexBytes(t *testing.T) {
	ws, indexPath, documentPath, citationPath := writeSnapshotFixture(t, "run-selected")
	raw := mustReadWorkspaceFile(t, ws, indexPath)
	invalid := strings.Replace(string(raw), documentPath, "reports/taskruns/run-selected/staging/final/../foreign.md", 1)
	if err := ws.WriteFileAtomic(indexPath, []byte(invalid)); err != nil {
		t.Fatal(err)
	}
	response := resolveRunSnapshot(ws, "run-selected", []orchestrator.Artifact{
		{Path: indexPath},
		{Path: documentPath},
		{Path: citationPath},
	})
	if response.Status != runSnapshotError {
		t.Fatalf("expected invalid path error, got %+v", response)
	}

	if err := os.Remove(filepath.Join(ws.Path, filepath.FromSlash(indexPath))); err != nil {
		t.Fatal(err)
	}
	response = resolveRunSnapshot(ws, "run-selected", []orchestrator.Artifact{{Path: indexPath}})
	if response.Status != runSnapshotUnavailable {
		t.Fatalf("expected unavailable index, got %+v", response)
	}
}

func writeSnapshotFixture(t *testing.T, runID string) (workspace.Root, string, string, string) {
	t.Helper()
	ws := workspace.Root{Path: t.TempDir()}
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "final-run-index.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.ReplaceAll(string(raw), "run-123", runID))
	indexPath := runFinalIndexPath(runID)
	documentPath := "reports/taskruns/" + runID + "/staging/final/reports/agent-outputs/domains/payments.md"
	citationPath := "reports/taskruns/" + runID + "/staging/final/citation-index.json"
	for rel, content := range map[string][]byte{
		indexPath:    raw,
		documentPath: []byte("# Payments\n"),
		citationPath: []byte("{}\n"),
	} {
		if err := ws.WriteFileAtomic(rel, content); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return ws, indexPath, documentPath, citationPath
}

func mustReadWorkspaceFile(t *testing.T, ws workspace.Root, rel string) []byte {
	t.Helper()
	raw, err := ws.ReadFile(rel)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
