package orchestrator

import (
	"path"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

const (
	constitutionDraftManifestFile = "constitution-draft.json"
	asisDraftManifestFile         = "asis-draft-manifest.json"
	proposalsDraftManifestFile    = "proposals-draft-manifest.json"
)

func runtimeStepContract(stepID string) string {
	switch acpruntime.StepProviderKeyForStepID(stepID) {
	case acpruntime.StepProviderStep0Constitution:
		return "constitution"
	case acpruntime.StepProviderStep1Collect:
		return "collect"
	case acpruntime.StepProviderStep2AsIs:
		return "as_is"
	case acpruntime.StepProviderStep3Findings:
		return "findings"
	case acpruntime.StepProviderStep4Proposals:
		return "proposals"
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
