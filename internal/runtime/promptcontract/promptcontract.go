package promptcontract

import (
	"fmt"
	"strings"

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
