package providercommon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/qa"
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
	case acpruntime.StepProviderQA:
		return ValidateQAArtifacts(task)
	default:
		if runtimedrafts.IsDraftStep(task.StepID) {
			return ValidateDraftArtifacts(task)
		}
		return nil
	}
}

func ValidateCollectArtifacts(task acpruntime.Task, provider acpruntime.Provider) error {
	return artifactquality.ValidateCollectManifestInRoot(task.WriteRoot)
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

func ValidateQAArtifacts(task acpruntime.Task) error {
	answer, err := qa.ValidateAnswerFile(filepath.Join(filepath.Clean(task.WriteRoot), "qa-answer.json"))
	if err != nil {
		return err
	}
	if strings.TrimSpace(answer.RunID) != strings.TrimSpace(task.RunID) {
		return fmt.Errorf("qa answer run_id %q does not match task run_id %q", answer.RunID, task.RunID)
	}
	if strings.TrimSpace(answer.Question) != strings.TrimSpace(task.Question) {
		return fmt.Errorf("qa answer question does not match task question")
	}
	contextPack, err := readQAContextPack(task.ContextPackPath)
	if err != nil {
		return err
	}
	return qa.ValidateAnswerAgainstContext(answer, contextPack)
}

func readQAContextPack(contextPackPath string) (qa.ContextPack, error) {
	contextPackPath = strings.TrimSpace(contextPackPath)
	if contextPackPath == "" {
		return qa.ContextPack{}, fmt.Errorf("qa context pack path is required")
	}
	raw, err := os.ReadFile(filepath.Clean(contextPackPath))
	if err != nil {
		return qa.ContextPack{}, fmt.Errorf("read qa context pack: %w", err)
	}
	return qa.ParseContextPack(raw)
}

func NonEmptyStage(stage string) string {
	stage = strings.TrimSpace(stage)
	if stage == "" {
		return "unknown"
	}
	return stage
}
