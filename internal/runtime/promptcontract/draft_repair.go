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
	manifestFile := filepath.Base(strings.TrimSpace(manifestTarget))
	lines := []string{
		"write_root=" + shellSingleQuote(writeRoot),
		"draft_root=" + shellSingleQuote(draftRoot),
		"mkdir -p \"$write_root\" \"$draft_root\"",
		"cat > \"$write_root/" + manifestFile + "\" <<'ACP_DRAFT_MANIFEST_JSON'",
		strings.TrimSpace(skeleton),
		"ACP_DRAFT_MANIFEST_JSON",
		"test -s \"$write_root/" + manifestFile + "\"",
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
		rel = filepath.ToSlash(filepath.FromSlash(rel))
		dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(rel)))
		if dir == "." {
			dir = ""
		}
		target := "$draft_root/" + rel
		mkdirTarget := "$draft_root"
		if dir != "" {
			mkdirTarget = "$draft_root/" + dir
		}
		lines = append(lines,
			"mkdir -p \""+mkdirTarget+"\"",
			"cat > \""+target+"\" <<'ACP_DRAFT_FILE'",
			draftArtifactRepairFileTemplate(task, output),
			"ACP_DRAFT_FILE",
			"test -s \""+target+"\"",
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
			"    notes: Runtime draft recovery initialized the required artifact set for this run.",
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
			"- Review staged shard manifests, validator findings, and owner gaps before promotion.",
		}, "\n")
	case "reports/agent-outputs/architect/summary.md":
		return strings.Join([]string{
			"# Architect Summary",
			"",
			"## Summary",
			"- Runtime draft recovery initialized the required as-is publish surface.",
			"- Treat this as diagnostic evidence until collected shard manifests and validator findings are reviewed.",
			"",
			"## Next Checks",
			"- Validate owner mappings and staged findings in the following runtime steps.",
		}, "\n")
	case "reports/changelog/runtime-proposals.md":
		return strings.Join([]string{
			"# Runtime Proposal Changelog",
			"",
			"## Changes",
			"- Runtime draft recovery initialized the proposal changelog surface.",
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
			"- Runtime draft recovery initialized this artifact for the scoped analysis step.",
			"- Use collected shard manifests and validator output as the evidence source before final review.",
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
		"- Do not begin with broad analysis. Run the exact shell command below as a single command before any other filesystem action.",
		"- If the draft manifest or referenced draft files already exist but are invalid, overwrite them from the heredoc artifacts instead of reading/diffing/patching.",
		"- Copy the heredoc artifacts exactly first. Do not manually retype or transform absolute paths; keep slash-separated path components unchanged.",
		"- Do not claim files are written or verified unless the exact test -s checks in the command pass.",
		"- Do not write shard-pack-manifest.json, validator-verdict.json, raw logs, sibling taskruns, or repository files.",
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
	case "init.step0.constitution":
		lines = append(lines,
			"- The heredoc charter overview is a bootstrap-only repair target, not valid final content.",
			"- Before final exit, replace recovery scaffold text in charter-overview.md with evidence-backed constitution content from read_context_roots, repo scope, and charter wizard contract when available.",
			"- Final action must be: ensure constitution-draft.json and every referenced draft file exist, charter-overview.md has no unchanged bootstrap/recovery scaffold, and baseline-subagents.yaml is a valid baseline bundle.",
		)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines,
			"- The heredoc as-is files are bootstrap-only repair targets, not valid final content.",
			"- Before final exit, replace recovery scaffold text with evidence-backed as-is content from read_context_roots.",
			"- Final action must be: ensure asis-draft-manifest.json and every referenced draft file exist and no referenced draft file contains unchanged bootstrap/recovery scaffold.",
		)
		lines = append(lines, "AS-IS DRAFT MANIFEST CANONICAL SHAPE:")
		lines = append(lines, artifactquality.AsIsDraftManifestContractLines()...)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines,
			"- The heredoc proposal/changelog files are bootstrap-only repair targets, not valid final content.",
			"- Before final exit, replace recovery scaffold text with evidence-backed proposal/changelog content from validated staged artifacts.",
			"- Final action must be: ensure proposals-draft-manifest.json and every referenced draft file exist and no referenced draft file contains unchanged bootstrap/recovery scaffold.",
		)
		lines = append(lines, "PROPOSALS DRAFT MANIFEST CANONICAL SHAPE:")
		lines = append(lines, artifactquality.ProposalsDraftManifestContractLines()...)
	default:
		lines = append(lines, "- Final action must be: write the draft manifest plus its referenced draft_final_root files, then exit successfully.")
	}
	return strings.Join(lines, "\n")
}

func ComposeDraftArtifactEnrichmentPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error) string {
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), manifestFile)
	skeleton := steppolicy.RuntimeDraftManifestTaskSkeleton(task)
	outputs := draftEnrichmentOutputs(skeleton)
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in draft artifact enrichment focused recovery mode.", provider),
		"Immediate draft artifact enrichment action:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		"- Do not run the earlier heredoc/bootstrap draft command again.",
		"- Do not create or preserve recovery scaffold text as final content.",
		"- First action: read the current draft manifest, then read staged shard manifests/authored shard docs/final-run-index/citation-index evidence available under the allowed read roots before rewriting markdown.",
		fmt.Sprintf("- Read and keep the existing manifest target in write_root: %q.", manifestTarget),
		fmt.Sprintf("- Rewrite draft content only under draft_final_root: %q.", strings.TrimSpace(task.DraftFinalRoot)),
		"- Allowed write targets are the step draft manifest in write_root and referenced draft files under draft_final_root.",
		"- Use only read_context_roots, the current write_root/draft_final_root files, staged evidence, and selected repository evidence roots already available to this provider.",
		"- Do not write shard-pack-manifest.json, validator-verdict.json, raw logs, sibling taskruns, workspace source-of-truth files, or repository files.",
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- draft_final_root (absolute) = %q`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- manifest_file = %q`, manifestFile),
		fmt.Sprintf(`- step_contract = %q`, strings.TrimSpace(task.StepContract)),
		"DRAFT ENRICHMENT TARGETS:",
	}
	if len(outputs) == 0 {
		lines = append(lines, "- Read the existing draft manifest outputs[] and enrich every referenced markdown draft file.")
	} else {
		for _, output := range outputs {
			lines = append(lines, fmt.Sprintf(
				"- %s -> %s (%s)",
				strings.TrimSpace(output.Path),
				strings.TrimSpace(output.CanonicalPath),
				strings.TrimSpace(output.Kind),
			))
		}
	}
	lines = append(lines,
		"DRAFT ENRICHMENT RULES:",
		"- Keep the manifest contract shape: version=1, run_id, step_id, step_contract, agent_role, outputs[].",
		"- Every outputs[].path must stay relative to draft_final_root and every referenced draft file must exist before exit.",
		"- Replace bootstrap-only markdown with evidence-backed content that cites concrete repositories, staged artifacts, files, services, modules, findings, or coverage gaps visible in the allowed read roots.",
		"- A no-op rewrite is invalid: every referenced markdown draft must be rewritten with marker-free evidence-backed content, not merely re-saved unchanged.",
		"- Preserve valid non-markdown support bundles when they are already canonical; for constitution, baseline-subagents.yaml may remain the baseline YAML bundle.",
		"- Final content MUST NOT include these scaffold markers: Runtime draft recovery initialized; draft recovery initialized; Treat this as diagnostic evidence until; Use collected shard manifests and validator output as the evidence source before final review; Draft surface initialized; Current run evidence should be reviewed; Runtime proposal surface initialized.",
		"- Final action must be: ensure the draft manifest and every referenced draft file exist, then ensure every referenced markdown draft changed and contains no unchanged bootstrap/recovery scaffold text.",
	)
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		lines = append(lines,
			"- Enrich charter-overview.md with concrete constitution content from read_context_roots, repo scope, and charter wizard contract when available.",
			"- Do not rewrite baseline-subagents.yaml unless it is invalid; it must remain a valid baseline agents bundle.",
		)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines,
			"- Enrich overview.md, summary.md, and architect-summary.md from collected shard manifests, authored shard docs, final indexes, citations, staged model evidence, and allowed repo evidence roots.",
			"- Include enough repository/path and staged artifact references for an operator to understand the architecture surface and remaining coverage gaps.",
			"- Include a decision-ready operator summary: what is complete, what is missing, and what the operator should inspect or decide next.",
			"- Include explicit coverage gaps when any planned shard, repo path, citation, or staged evidence surface is partial or missing.",
		)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines,
			"- Enrich proposal and changelog drafts from validated staged findings, coverage gaps, questions, citations, and proposal candidates.",
			"- Proposals must be actionable and traceable to staged evidence; do not leave generic validation notes as the only content.",
		)
	}
	if validationErr != nil {
		lines = append(lines, fmt.Sprintf(`- Previous draft artifact validation failure: %s`, compactDraftEnrichmentHint(validationErr.Error())))
	}
	return strings.Join(lines, "\n")
}

func compactDraftEnrichmentHint(value string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(normalized) <= 320 {
		return normalized
	}
	return normalized[:317] + "..."
}

func draftEnrichmentOutputs(skeleton string) []runtimedrafts.Output {
	var manifest runtimedrafts.Manifest
	if err := json.Unmarshal([]byte(skeleton), &manifest); err != nil {
		return nil
	}
	return manifest.Outputs
}

func draftRepairFirstCommandIntro(task acpruntime.Task) (string, string) {
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		return "FIRST CONSTITUTION DRAFT COMMAND:",
			"Run this exact shell command as the next filesystem action; it writes constitution-draft.json plus every referenced draft file as the first bootstrap draft set:"
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return "FIRST AS-IS DRAFT COMMAND:",
			"Run this exact shell command as the next filesystem action; it writes asis-draft-manifest.json plus overview.md, summary.md, and architect-summary.md as the first bootstrap draft set:"
	case "init.step4.proposals", "refresh.step4.proposals":
		return "FIRST PROPOSALS DRAFT COMMAND:",
			"Run this exact shell command as the next filesystem action; it writes proposals-draft-manifest.json plus every referenced draft file as the first bootstrap draft set:"
	default:
		return "FIRST RUNTIME DRAFT COMMAND:",
			"Run this exact shell command as the next filesystem action; it writes the runtime draft manifest plus every referenced draft file as the first valid artifact set:"
	}
}
