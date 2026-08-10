package providercommon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

// recoverDraftManifestShapeDeterministically restores only the normative draft
// manifest envelope after a provider writes malformed JSON or unknown fields.
// The authored markdown remains untouched and is still validated by the adapter;
// this recovery must never turn invalid document content into a pass.
func recoverDraftManifestShapeDeterministically(
	task acpruntime.Task,
	adapter ProviderAdapter,
	result acpruntime.Result,
	beforeWriteRoot writeRootFileSnapshot,
	beforeDraftRoot writeRootFileSnapshot,
	validationErr error,
	stage string,
) (bool, acpruntime.Result, error) {
	issues := classifyValidationIssues(validationErr)
	if !issues.HasAny(issueDraftManifestParse, issueDraftUnknownField) {
		return false, acpruntime.Result{}, nil
	}
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	if strings.TrimSpace(manifestFile) == "" || strings.TrimSpace(task.WriteRoot) == "" {
		return false, acpruntime.Result{}, nil
	}
	manifestPath := filepath.Join(filepath.Clean(task.WriteRoot), manifestFile)
	raw := []byte(strings.TrimSpace(steppolicy.RuntimeDraftManifestTaskSkeleton(task)) + "\n")
	if err := writeDraftManifestShapeAtomically(manifestPath, raw); err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "draft_artifact_manifest_shape_recovery", "write normative draft manifest skeleton", err)
	}
	if err := validateDraftArtifactRepairWriteSetForStage(task, beforeWriteRoot, beforeDraftRoot, stage); err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "draft_artifact_manifest_shape_recovery", "manifest shape recovery wrote outside the draft artifact write set", err)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		// The envelope is repaired, but markdown/content or another contract issue
		// still fails closed and must continue through the normal provider repair.
		return false, acpruntime.Result{}, nil
	}
	if result.Diagnostics == nil {
		result.Diagnostics = map[string]any{}
	}
	result.Diagnostics["draft_manifest_shape_recovery"] = map[string]any{
		"recovery_mode":            "draft_artifact_manifest_shape_recovery",
		"provider_authored":        false,
		"manifest_file":            manifestFile,
		"stage":                    stage,
		"operator_review_required": true,
	}
	return true, result, nil
}

func writeDraftManifestShapeAtomically(target string, content []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create draft manifest temp file: %w", err)
	}
	tmpPath := tmp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write draft manifest temp file: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod draft manifest temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync draft manifest temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close draft manifest temp file: %w", err)
	}
	if err := os.Rename(tmpPath, target); err != nil {
		return fmt.Errorf("replace draft manifest: %w", err)
	}
	removeTemp = false
	return nil
}
