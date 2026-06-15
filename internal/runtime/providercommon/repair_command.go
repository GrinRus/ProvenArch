package providercommon

import (
	"fmt"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
)

type FocusedRepairKind string

const (
	FocusedRepairCollectManifest     FocusedRepairKind = "collect_manifest"
	FocusedRepairCollectArtifactPair FocusedRepairKind = "collect_artifact_pair"
	FocusedRepairValidatorVerdict    FocusedRepairKind = "validator_verdict"
	FocusedRepairDraftArtifacts      FocusedRepairKind = "draft_artifacts"
	FocusedRepairDraftEnrichment     FocusedRepairKind = "draft_artifact_enrichment"
)

type FocusedRepairCommandBuilder func(task acpruntime.Task, includeDirs []string, prompt string) (CommandSpec, error)

func BuildFocusedRepairCommandSpec(
	task acpruntime.Task,
	provider acpruntime.Provider,
	kind FocusedRepairKind,
	validationErr error,
	build FocusedRepairCommandBuilder,
) (CommandSpec, error) {
	if build == nil {
		return CommandSpec{}, fmt.Errorf("focused repair command builder is nil")
	}
	includeDirs := focusedRepairIncludeDirectories(task, kind)
	repairTask := task
	repairTask.ReadContextRoots = append([]string(nil), includeDirs...)
	return build(repairTask, includeDirs, focusedRepairPrompt(provider, repairTask, kind, validationErr))
}

func focusedRepairIncludeDirectories(task acpruntime.Task, kind FocusedRepairKind) []string {
	switch kind {
	case FocusedRepairCollectManifest, FocusedRepairCollectArtifactPair:
		return acpruntime.ResolveHeadlessCollectRepairIncludeDirectories(task)
	case FocusedRepairValidatorVerdict:
		return acpruntime.ResolveHeadlessValidatorRepairIncludeDirectories(task)
	case FocusedRepairDraftArtifacts, FocusedRepairDraftEnrichment:
		return acpruntime.ResolveHeadlessDraftRepairIncludeDirectories(task)
	default:
		return acpruntime.ResolveHeadlessIncludeDirectories(task)
	}
}

func focusedRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, kind FocusedRepairKind, validationErr error) string {
	switch kind {
	case FocusedRepairCollectManifest:
		return promptcontract.ComposeCollectManifestRepairPrompt(provider, task, validationErr)
	case FocusedRepairCollectArtifactPair:
		return promptcontract.ComposeCollectArtifactPairRepairPrompt(provider, task, validationErr)
	case FocusedRepairValidatorVerdict:
		return promptcontract.ComposeValidatorVerdictRepairPrompt(provider, task, validationErr)
	case FocusedRepairDraftArtifacts:
		return promptcontract.ComposeDraftArtifactRepairPrompt(provider, task, validationErr)
	case FocusedRepairDraftEnrichment:
		return promptcontract.ComposeDraftArtifactEnrichmentPrompt(provider, task, validationErr)
	default:
		return promptcontract.ComposeArtifactOnlyPrompt(provider, task)
	}
}
