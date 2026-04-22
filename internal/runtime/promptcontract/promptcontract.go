package promptcontract

import (
	"fmt"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
)

func ComposeArtifactOnlyPrompt(provider acpruntime.Provider, task acpruntime.Task) string {
	sections := []string{
		fmt.Sprintf("You are ACP runtime provider %q.", provider),
	}
	sections = append(sections, SharedSections(task)...)
	return strings.Join(sections, "\n\n")
}

func SharedSections(task acpruntime.Task) []string {
	sections := []string{
		"Artifact-only contract:",
		"- Do not return semantic JSON or any other semantic payload on stdout.",
		"- Write only the required step artifacts into write_root and draft_final_root.",
		"- Stdout/stderr are diagnostics only.",
		steppolicy.DocFirstFilesystemPolicy(task),
	}
	if stepPolicy := strings.TrimSpace(steppolicy.StepSpecificPolicy(task.StepID)); stepPolicy != "" {
		sections = append(sections, stepPolicy)
	}
	if acpruntime.IsCollectStep(task.StepID) {
		collectLines := []string{
			"COLLECT MANIFEST CANONICAL SHAPE:",
		}
		collectLines = append(collectLines, artifactquality.CollectManifestContractLines(strings.TrimSpace(task.ArtifactRoot))...)
		collectLines = append(collectLines,
			`- Canonical fragment below is normative for field names and value types; copy keys/types exactly and only change IDs/content.`,
			artifactquality.CollectManifestCanonicalExample(),
		)
		sections = append(sections, strings.Join(collectLines, "\n"))
	}
	switch strings.TrimSpace(task.StepID) {
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		asIsLines := []string{
			"AS-IS DRAFT MANIFEST CANONICAL SHAPE:",
		}
		asIsLines = append(asIsLines, artifactquality.AsIsDraftManifestContractLines()...)
		asIsLines = append(asIsLines,
			`- Canonical fragment below is normative for field names, step_contract, and required outputs; copy keys/types exactly and only change IDs/content.`,
			artifactquality.AsIsDraftManifestCanonicalExample(),
		)
		sections = append(sections, strings.Join(asIsLines, "\n"))
	case "init.step3.findings", "refresh.step3.findings":
		findingsLines := []string{
			"VALIDATOR VERDICT CANONICAL SHAPE:",
		}
		findingsLines = append(findingsLines, artifactquality.ValidatorVerdictContractLines()...)
		findingsLines = append(findingsLines,
			`- Canonical fragment below is normative for validator metadata and finding evidence shape; copy keys/types exactly and only change IDs/content.`,
			artifactquality.ValidatorVerdictCanonicalExample(),
		)
		sections = append(sections, strings.Join(findingsLines, "\n"))
	}
	if pack := strings.TrimSpace(steppolicy.WorkspacePromptPackSection(task)); pack != "" {
		sections = append(sections, pack)
	}
	sections = append(sections,
		"Completion rule:",
		"- Exit with code 0 only after required artifacts are fully written.",
		"- Do not emit legacy operation logs or any wrapper envelopes on stdout.",
	)
	return sections
}
