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
	default:
		return nil
	}
}

func runtimeDraftArtifactRoot(runID string, stepID string) string {
	stepKey := acpruntime.StepProviderKeyForStepID(stepID)
	if strings.TrimSpace(stepKey) == "" {
		stepKey = "runtime"
	}
	return path.Join("reports", "taskruns", runID, "staging", "drafts", stepKey)
}

func runtimeStepWriteRoot(runID string, stepID string) string {
	stepKey := acpruntime.StepProviderKeyForStepID(stepID)
	if strings.TrimSpace(stepKey) == "" {
		stepKey = "runtime"
	}
	return path.Join("reports", "taskruns", runID, "runtime", stepKey)
}
