package providercommon

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
)

func TestRecoverDraftManifestShapeDeterministicallyRestoresNormativeEnvelope(t *testing.T) {
	t.Parallel()

	task := newProposalsDraftTask(t, "run-manifest-shape-recovery")
	manifestPath := filepath.Join(task.WriteRoot, "proposals-draft-manifest.json")
	if err := os.WriteFile(manifestPath, []byte("{\"version\":1,\"outputs\":"), 0o644); err != nil {
		t.Fatalf("write malformed manifest: %v", err)
	}
	beforeWriteRoot, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		t.Fatalf("snapshot write root: %v", err)
	}
	beforeDraftRoot, err := snapshotWriteRootFiles(task.DraftFinalRoot)
	if err != nil {
		t.Fatalf("snapshot draft root: %v", err)
	}

	adapter := manifestShapeRecoveryTestAdapter{}
	result := acpruntime.Result{}
	recovered, recoveredResult, recoveryErr := recoverDraftManifestShapeDeterministically(
		task,
		adapter,
		result,
		beforeWriteRoot,
		beforeDraftRoot,
		errors.New(`parse runtime draft manifest: invalid character '\\' looking for beginning of object key string`),
		"draft_artifact_enrichment",
	)
	if recoveryErr != nil {
		t.Fatalf("shape recovery failed: %v", recoveryErr)
	}
	if !recovered {
		t.Fatal("expected malformed manifest shape recovery")
	}
	if got, want := string(mustReadFile(t, manifestPath)), steppolicy.RuntimeDraftManifestTaskSkeleton(task)+"\n"; got != want {
		t.Fatalf("manifest was not restored to normative skeleton:\n got %q\nwant %q", got, want)
	}
	if recoveredResult.Diagnostics["draft_manifest_shape_recovery"] == nil {
		t.Fatalf("expected deterministic recovery diagnostic, got %#v", recoveredResult.Diagnostics)
	}
}

func TestRecoverDraftManifestMissingDeterministicallyRestoresNormativeEnvelope(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-manifest-missing-recovery")
	for _, name := range []string{"overview.md", "summary.md", "architect-summary.md"} {
		if err := os.WriteFile(filepath.Join(task.DraftFinalRoot, name), []byte("# Repository overview\n\nProvider-authored draft content.\n"), 0o644); err != nil {
			t.Fatalf("write draft output %s: %v", name, err)
		}
	}
	beforeWriteRoot, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		t.Fatalf("snapshot write root: %v", err)
	}
	beforeDraftRoot, err := snapshotWriteRootFiles(task.DraftFinalRoot)
	if err != nil {
		t.Fatalf("snapshot draft root: %v", err)
	}

	adapter := manifestShapeRecoveryTestAdapter{}
	result := acpruntime.Result{}
	recovered, recoveredResult, recoveryErr := recoverDraftManifestShapeDeterministically(
		task,
		adapter,
		result,
		beforeWriteRoot,
		beforeDraftRoot,
		errors.New("read runtime draft manifest: open asis-draft-manifest.json: no such file or directory"),
		"draft_manifest_missing_after_final_fresh_process",
	)
	if recoveryErr != nil {
		t.Fatalf("missing manifest recovery failed: %v", recoveryErr)
	}
	if !recovered {
		t.Fatal("expected missing manifest recovery")
	}
	manifestPath := filepath.Join(task.WriteRoot, "asis-draft-manifest.json")
	if got, want := string(mustReadFile(t, manifestPath)), steppolicy.RuntimeDraftManifestTaskSkeleton(task)+"\n"; got != want {
		t.Fatalf("manifest was not restored to normative skeleton:\n got %q\nwant %q", got, want)
	}
	recoveryDiagnostic, ok := recoveredResult.Diagnostics["draft_manifest_shape_recovery"].(map[string]any)
	if !ok {
		t.Fatalf("expected deterministic recovery diagnostic, got %#v", recoveredResult.Diagnostics)
	}
	if got, want := recoveryDiagnostic["recovery_mode"], "draft_artifact_manifest_missing_recovery"; got != want {
		t.Fatalf("unexpected recovery mode: got %v want %v", got, want)
	}
}

type manifestShapeRecoveryTestAdapter struct{ testAdapter }

func (manifestShapeRecoveryTestAdapter) ValidateArtifacts(acpruntime.Task) error { return nil }

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
