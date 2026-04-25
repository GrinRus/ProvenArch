package providercommon

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

const (
	ShardPackManifestFileName = "shard-pack-manifest.json"
	ValidatorVerdictFileName  = "validator-verdict.json"
)

func ValidateRuntimeArtifacts(task acpruntime.Task, provider acpruntime.Provider) error {
	switch acpruntime.StepProviderKeyForStepID(task.StepID) {
	case acpruntime.StepProviderStep1Collect:
		return ValidateCollectArtifacts(task, provider)
	case acpruntime.StepProviderStep3Findings:
		return ValidateValidatorArtifacts(task)
	default:
		if runtimedrafts.IsDraftStep(task.StepID) {
			return ValidateDraftArtifacts(task)
		}
		return nil
	}
}

func ValidateCollectArtifacts(task acpruntime.Task, provider acpruntime.Provider) error {
	report, err := artifactquality.RepairCollectManifest(task)
	if err != nil {
		return err
	}
	EmitCollectRepairDiagnostic(task, report, provider)
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(task.WriteRoot), ShardPackManifestFileName))
	if err != nil {
		return err
	}
	if _, err := contracts.ParseShardPackManifest(raw); err != nil {
		return err
	}
	return nil
}

func EmitCollectRepairDiagnostic(task acpruntime.Task, report artifactquality.RepairReport, provider acpruntime.Provider) {
	if task.OnDiagnostic == nil || len(report.AppliedRuleIDs) == 0 {
		return
	}
	task.OnDiagnostic(acpruntime.DiagnosticEvent{
		Message: "collect compatibility repair applied",
		Fields: map[string]any{
			"provider":         string(provider),
			"changed":          report.Changed,
			"applied_rule_ids": append([]string(nil), report.AppliedRuleIDs...),
		},
	})
}

func ValidateDraftArtifacts(task acpruntime.Task) error {
	if _, _, err := ValidateRequiredRuntimeDraftArtifacts(task); err == nil {
		return nil
	}
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	if manifestFile == "" {
		return fmt.Errorf("draft manifest file is undefined for %s", task.StepID)
	}
	manifest, _, loadErr := runtimedrafts.Load(task.WriteRoot, manifestFile)
	if loadErr != nil {
		return loadErr
	}
	if err := runtimedrafts.ValidateManifestForTask(manifest, task.RunID, task.StepID, task.StepContract); err != nil {
		return err
	}
	if _, err := runtimedrafts.ReconcileOutputsAtDraftRoot(task.DraftFinalRoot, manifest); err != nil {
		return err
	}
	_, _, err := ValidateRequiredRuntimeDraftArtifacts(task)
	return err
}

func ValidateRequiredRuntimeDraftArtifacts(task acpruntime.Task) (runtimedrafts.Manifest, []byte, error) {
	return runtimedrafts.ValidateRequiredManifest(
		task.WriteRoot,
		task.DraftFinalRoot,
		task.RunID,
		task.StepID,
		task.StepContract,
		task.ExpectedArtifacts,
	)
}

func ValidateValidatorArtifacts(task acpruntime.Task) error {
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(task.WriteRoot), ValidatorVerdictFileName))
	if err != nil {
		return err
	}
	_, err = contracts.ParseValidatorVerdict(raw)
	return err
}

func NonEmptyStage(stage string) string {
	stage = strings.TrimSpace(stage)
	if stage == "" {
		return "unknown"
	}
	return stage
}
