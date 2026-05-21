package promptcontract

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func draftArtifactRepairWriteCommand(task acpruntime.Task, manifestTarget string, skeleton string) string {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	draftRoot := strings.TrimSpace(task.DraftFinalRoot)
	lines := []string{
		"mkdir -p " + shellSingleQuote(writeRoot) + " " + shellSingleQuote(draftRoot),
		"cat > " + shellSingleQuote(strings.TrimSpace(manifestTarget)) + " <<'ACP_DRAFT_MANIFEST_JSON'",
		strings.TrimSpace(skeleton),
		"ACP_DRAFT_MANIFEST_JSON",
		"test -s " + shellSingleQuote(strings.TrimSpace(manifestTarget)),
	}
	var manifest runtimedrafts.Manifest
	if err := json.Unmarshal([]byte(skeleton), &manifest); err != nil {
		return strings.Join(lines, "\n")
	}
	for _, output := range manifest.Outputs {
		rel, ok := safeDraftOutputPath(output.Path)
		if !ok {
			continue
		}
		target := filepath.Join(draftRoot, filepath.FromSlash(rel))
		lines = append(lines,
			"mkdir -p "+shellSingleQuote(filepath.Dir(target)),
			"cat > "+shellSingleQuote(target)+" <<'ACP_DRAFT_FILE'",
			draftArtifactRepairFileTemplate(task, output),
			"ACP_DRAFT_FILE",
			"test -s "+shellSingleQuote(target),
		)
	}
	return strings.Join(lines, "\n")
}

func safeDraftOutputPath(raw string) (string, bool) {
	rel := filepath.ToSlash(strings.TrimSpace(raw))
	rel = strings.TrimPrefix(rel, "./")
	if rel == "" || rel == "." || strings.HasPrefix(rel, "/") {
		return "", false
	}
	rel = path.Clean(rel)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

func draftArtifactRepairFileTemplate(task acpruntime.Task, output runtimedrafts.Output) string {
	canonicalPath := strings.TrimSpace(output.CanonicalPath)
	title := strings.TrimSpace(output.Title)
	if title == "" {
		title = strings.TrimSpace(output.Path)
	}
	if canonicalPath == "skills/subagents.yaml" {
		return strings.TrimSpace(string(workspace.BaselineSubagentsContent()))
	}
	if strings.HasSuffix(strings.ToLower(output.Path), ".yaml") || strings.HasSuffix(strings.ToLower(output.Path), ".yml") {
		return strings.Join([]string{
			"version: 1",
			"generated_by: acp-runtime-provider-focused-recovery",
			"run_id: " + strings.TrimSpace(task.RunID),
			"step_id: " + strings.TrimSpace(task.StepID),
			"items:",
			"  - id: baseline",
			"    title: Baseline Runtime Draft",
			"    notes: Provider focused recovery wrote the required draft artifact set.",
		}, "\n")
	}
	switch canonicalPath {
	case "reports/coverage/summary.md":
		return strings.Join([]string{
			"# Coverage Summary",
			"",
			"## Observed",
			"- Collected runtime shard manifests were available before draft recovery.",
			"",
			"## Missing",
			"- Owner mapping details need follow-up review in the promoted reports.",
			"",
			"## Notes",
			"- Provider focused recovery wrote this draft artifact under the required draft_final_root.",
		}, "\n")
	case "reports/agent-outputs/architect/summary.md":
		return strings.Join([]string{
			"# Architect Summary",
			"",
			"## Summary",
			"- Provider focused recovery produced the required as-is draft publish surface.",
			"- Review collected shard manifests before treating this diagnostic output as release evidence.",
			"",
			"## Next Checks",
			"- Validate owner mappings and staged findings in the following runtime steps.",
		}, "\n")
	case "reports/changelog/runtime-proposals.md":
		return strings.Join([]string{
			"# Runtime Proposal Changelog",
			"",
			"## Changes",
			"- Provider focused recovery wrote the required proposal changelog draft.",
			"",
			"## Notes",
			"- Promote only after validator artifacts pass.",
		}, "\n")
	default:
		return strings.Join([]string{
			"# " + title,
			"",
			"## Scope",
			"- Run: " + strings.TrimSpace(task.RunID),
			"- Step: " + strings.TrimSpace(task.StepID),
			"",
			"## Summary",
			"- Provider focused recovery wrote this draft artifact to satisfy the runtime draft contract.",
			"- Use collected shard manifests and validator output as the evidence source for final review.",
		}, "\n")
	}
}

func ComposeDraftArtifactRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error) string {
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), manifestFile)
	skeleton := steppolicy.RuntimeDraftManifestTaskSkeleton(task)
	firstCommandHeading, firstCommandInstruction := draftRepairFirstCommandIntro(task)
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in draft artifact focused recovery mode.", provider),
		"Immediate draft artifact repair action:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		fmt.Sprintf("- Write exactly this manifest target in write_root: %q.", manifestTarget),
		fmt.Sprintf("- Write draft content only under draft_final_root: %q.", strings.TrimSpace(task.DraftFinalRoot)),
		"- Do not begin with broad analysis. Run the preferred file write command below, or perform exactly the same bounded writes.",
		"- If the draft manifest or referenced draft files already exist but are invalid, overwrite them from the heredoc artifacts instead of reading/diffing/patching.",
		"- Copy the heredoc artifacts exactly first. Do not make factual edits before the manifest and draft files validate.",
		"- Do not write shard-pack-manifest.json, validator-verdict.json, raw logs, sibling taskruns, or repository files.",
		"- Final action must be: write the draft manifest plus its referenced draft_final_root files, then exit successfully.",
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- draft_final_root (absolute) = %q`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- manifest_file = %q`, manifestFile),
		fmt.Sprintf(`- step_contract = %q`, strings.TrimSpace(task.StepContract)),
		fmt.Sprintf(`- expected_artifacts = %s`, strings.Join(task.ExpectedArtifacts, ", ")),
		firstCommandHeading,
		firstCommandInstruction,
		draftArtifactRepairWriteCommand(task, manifestTarget, skeleton),
		"DRAFT ARTIFACT REPAIR INSTRUCTIONS:",
	}
	lines = append(lines, steppolicy.DraftArtifactRepairHints(task, validationErr)...)
	lines = append(lines,
		"- Keep the skeleton manifest keys/types; adjust only summary and output metadata needed for real draft files after the first valid artifact set exists.",
		"- Every outputs[].path must be relative to draft_final_root and every referenced draft file must exist before exit.",
		"- Absolute target checks must use write_root/draft_final_root exactly; relative CWD checks are invalid.",
	)
	switch strings.TrimSpace(task.StepID) {
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines, "AS-IS DRAFT MANIFEST CANONICAL SHAPE:")
		lines = append(lines, artifactquality.AsIsDraftManifestContractLines()...)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines, "PROPOSALS DRAFT MANIFEST CANONICAL SHAPE:")
		lines = append(lines, artifactquality.ProposalsDraftManifestContractLines()...)
	}
	return strings.Join(lines, "\n")
}

func draftRepairFirstCommandIntro(task acpruntime.Task) (string, string) {
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		return "FIRST CONSTITUTION DRAFT COMMAND:",
			"Run this exact command as the next filesystem action; it writes constitution-draft.json plus every referenced draft file as the first valid artifact set:"
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return "FIRST AS-IS DRAFT COMMAND:",
			"Run this exact command as the next filesystem action; it writes asis-draft-manifest.json plus overview.md, summary.md, and architect-summary.md as the first valid artifact set:"
	case "init.step4.proposals", "refresh.step4.proposals":
		return "FIRST PROPOSALS DRAFT COMMAND:",
			"Run this exact command as the next filesystem action; it writes proposals-draft-manifest.json plus every referenced draft file as the first valid artifact set:"
	default:
		return "FIRST RUNTIME DRAFT COMMAND:",
			"Run this exact command as the next filesystem action; it writes the runtime draft manifest plus every referenced draft file as the first valid artifact set:"
	}
}
