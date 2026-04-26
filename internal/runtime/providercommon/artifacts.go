package providercommon

import (
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
	return artifactquality.ValidateCollectManifest(task)
}

func ValidateDraftArtifacts(task acpruntime.Task) error {
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
