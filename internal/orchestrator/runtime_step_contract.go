package orchestrator

import (
	"path"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

const (
	constitutionDraftManifestFile = runtimedrafts.ConstitutionManifestFile
	asisDraftManifestFile         = runtimedrafts.AsIsManifestFile
	proposalsDraftManifestFile    = runtimedrafts.ProposalsManifestFile
	runtimeExecutionFile          = "runtime-execution.json"
)

func runtimeStepContract(stepID string) string {
	if contract := strings.TrimSpace(runtimedrafts.StepContractForStep(stepID)); contract != "" {
		return contract
	}
	switch acpruntime.StepProviderKeyForStepID(stepID) {
	case acpruntime.StepProviderStep1Collect:
		return "collect"
	case acpruntime.StepProviderStep3Findings:
		return "findings"
	case acpruntime.StepProviderQA:
		return "qa"
	default:
		return "runtime"
	}
}

func runtimeExpectedArtifacts(stepID string) []string {
	switch acpruntime.StepProviderKeyForStepID(stepID) {
	case acpruntime.StepProviderStep0Constitution:
		return []string{constitutionDraftManifestFile}
	case acpruntime.StepProviderStep1Collect:
		return []string{shardPackManifestFile}
	case acpruntime.StepProviderStep2AsIs:
		return []string{asisDraftManifestFile}
	case acpruntime.StepProviderStep3Findings:
		return []string{validatorVerdictFile}
	case acpruntime.StepProviderStep4Proposals:
		return []string{proposalsDraftManifestFile}
	case acpruntime.StepProviderQA:
		return []string{"qa-answer.json"}
	default:
		return nil
	}
}

func runtimeDraftArtifactRoot(runID string, stepID string) string {
	stepKey := acpruntime.StepProviderKeyForStepID(stepID)
	if stepKey == acpruntime.StepProviderQA {
		return runtimeQATaskRoot(runID)
	}
	if strings.TrimSpace(stepKey) == "" {
		stepKey = "runtime"
	}
	return path.Join("reports", "taskruns", runID, "staging", "drafts", stepKey)
}

func runtimeStepWriteRoot(runID string, stepID string) string {
	stepKey := acpruntime.StepProviderKeyForStepID(stepID)
	if stepKey == acpruntime.StepProviderQA {
		return runtimeQATaskRoot(runID)
	}
	if strings.TrimSpace(stepKey) == "" {
		stepKey = "runtime"
	}
	return path.Join("reports", "taskruns", runID, "runtime", stepKey)
}

func runtimeExecutionMetadataPath(runID string, stepID string, shardID string) string {
	switch acpruntime.StepProviderKeyForStepID(stepID) {
	case acpruntime.StepProviderStep1Collect:
		return path.Join(runtimeShardArtifactRoot(runID, shardID), runtimeExecutionFile)
	case acpruntime.StepProviderStep3Findings:
		return path.Join(runtimeValidatorArtifactRoot(runID), runtimeExecutionFile)
	case acpruntime.StepProviderQA:
		return path.Join(runtimeQATaskRoot(runID), runtimeExecutionFile)
	default:
		return path.Join(runtimeStepWriteRoot(runID, stepID), runtimeExecutionFile)
	}
}

func runtimeQATaskRoot(runID string) string {
	return path.Join("reports", "taskruns", runID, "qa")
}

func runtimeExecutionMetadataPathForTask(task acpruntime.Task) string {
	artifactRoot := strings.TrimSpace(task.ArtifactRoot)
	if artifactRoot != "" {
		return path.Join(artifactRoot, runtimeExecutionFile)
	}
	return runtimeExecutionMetadataPath(task.RunID, task.StepID, task.ShardID)
}
