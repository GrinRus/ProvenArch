package qwencode

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

func repairAndValidateArtifacts(task acpruntime.Task) error {
	switch {
	case isCollectStep(task.StepID):
		return repairAndValidateCollectArtifacts(task)
	case isDraftStep(task.StepID):
		return repairAndValidateDraftArtifacts(task)
	case isFindingsStep(task.StepID):
		return validateValidatorArtifacts(task)
	default:
		return nil
	}
}

func repairAndValidateCollectArtifacts(task acpruntime.Task) error {
	report, err := artifactquality.RepairCollectManifest(task)
	if err != nil {
		return err
	}
	emitCollectRepairDiagnostic(task, report)
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(task.WriteRoot), "shard-pack-manifest.json"))
	if err != nil {
		return err
	}
	_, err = contracts.ParseShardPackManifest(raw)
	return err
}

func emitCollectRepairDiagnostic(task acpruntime.Task, report artifactquality.RepairReport) {
	if task.OnDiagnostic == nil || len(report.AppliedRuleIDs) == 0 {
		return
	}
	emitDiagnostic(task, "collect compatibility repair applied", map[string]any{
		"provider":         string(acpruntime.ProviderQwenCode),
		"changed":          report.Changed,
		"applied_rule_ids": append([]string(nil), report.AppliedRuleIDs...),
	})
}

func repairAndValidateDraftArtifacts(task acpruntime.Task) error {
	if _, _, err := validateRuntimeDraftArtifactsAtWriteRoot(task); err == nil {
		return nil
	}

	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	if manifestFile == "" {
		return fmt.Errorf("runtime draft manifest is undefined for %s", task.StepID)
	}
	manifest, _, err := runtimedrafts.Load(task.WriteRoot, manifestFile)
	if err != nil {
		return err
	}
	if err := runtimedrafts.ValidateManifestForTask(manifest, task.RunID, task.StepID, task.StepContract); err != nil {
		return err
	}
	if _, err := runtimedrafts.ReconcileOutputsAtDraftRoot(task.DraftFinalRoot, manifest); err != nil {
		return err
	}
	_, _, err = validateRuntimeDraftArtifactsAtWriteRoot(task)
	return err
}

func validateRuntimeDraftArtifactsAtWriteRoot(task acpruntime.Task) (runtimedrafts.Manifest, []byte, error) {
	return runtimedrafts.ValidateRequiredManifest(
		task.WriteRoot,
		task.DraftFinalRoot,
		task.RunID,
		task.StepID,
		task.StepContract,
		task.ExpectedArtifacts,
	)
}

func validateValidatorArtifacts(task acpruntime.Task) error {
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(task.WriteRoot), "validator-verdict.json"))
	if err != nil {
		return err
	}
	_, err = contracts.ParseValidatorVerdict(raw)
	return err
}

func isCollectStep(stepID string) bool {
	switch strings.TrimSpace(stepID) {
	case "init.step1.collect", "refresh.step1.collect":
		return true
	default:
		return false
	}
}

func isFindingsStep(stepID string) bool {
	switch strings.TrimSpace(stepID) {
	case "init.step3.findings", "refresh.step3.findings":
		return true
	default:
		return false
	}
}

func isDraftStep(stepID string) bool {
	return runtimedrafts.IsDraftStep(stepID)
}
