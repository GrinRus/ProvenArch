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
