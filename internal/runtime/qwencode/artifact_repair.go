package qwencode

import (
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/providercommon"
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
	return providercommon.ValidateCollectArtifacts(task, acpruntime.ProviderQwenCode)
}

func repairAndValidateDraftArtifacts(task acpruntime.Task) error {
	return providercommon.ValidateDraftArtifacts(task)
}

func validateValidatorArtifacts(task acpruntime.Task) error {
	return providercommon.ValidateValidatorArtifacts(task)
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
