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
	statusEvidenceFiles := draftEnrichmentShardStatusEvidenceFiles(task)
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in draft artifact enrichment focused recovery mode.", provider),
		"Immediate draft artifact enrichment action:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		"- Do not answer with a plan, status note, or analysis-only message. Your next action must be a filesystem command that rewrites every referenced markdown draft target.",
		"- Forbidden analysis-only phrases before the rewrite: I have enough evidence; I will rewrite; I will now rewrite; ready to rewrite; the drafts will cite.",
		"- Do not run the earlier heredoc/bootstrap draft command again.",
		"- Do not create or preserve recovery scaffold text as final content.",
		"- First focused work unit: execute one bounded filesystem command that reads the current draft manifest and bounded staged evidence, then rewrites every referenced markdown target in that same command before any optional extended analysis.",
		"- Fresh mutation is required: the harness ignores pre-existing bootstrap files until you rewrite every markdown target in this enrichment command.",
		"- Do not spend the whole run reading evidence without a write; make a marker-free evidence-backed rewrite for every markdown target in the first command, then refine it if time remains.",
		fmt.Sprintf("- Read and keep the existing manifest target in write_root: %q.", manifestTarget),
		fmt.Sprintf("- Rewrite draft content only under draft_final_root: %q.", strings.TrimSpace(task.DraftFinalRoot)),
		"- Allowed write targets are the step draft manifest in write_root and referenced draft files under draft_final_root.",
		"- Prefer not to rewrite the draft manifest during enrichment; if you must touch it, preserve its existing outputs[] entries exactly.",
		"- Do not add, remove, rename, or alias outputs[] fields. The only allowed output object keys are path, canonical_path, kind, and title.",
		"- Never add logical_path, target, output_path, publish_path, or other output aliases; the strict parser treats them as runtime_contract_failed.",
		"- Use only the bounded read_context_roots, the current write_root/draft_final_root files, and staged evidence already available to this provider.",
		fmt.Sprintf(`- bounded_read_context_roots = %q`, strings.Join(task.ReadContextRoots, ", ")),
		"- Do not write shard-pack-manifest.json, validator-verdict.json, raw logs, sibling taskruns, workspace source-of-truth files, or repository files.",
		"- Read the current draft manifest only for contract fields and exact outputs; do not quote or copy its bootstrap summary, schema keys, canonical_path examples, validation errors, or scaffold phrases into final markdown.",
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- draft_final_root (absolute) = %q`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- manifest_file = %q`, manifestFile),
		fmt.Sprintf(`- step_contract = %q`, strings.TrimSpace(task.StepContract)),
		"DRAFT ENRICHMENT TARGET IDENTITY:",
		fmt.Sprintf("- current_run_id = %q", strings.TrimSpace(task.RunID)),
		fmt.Sprintf("- current_step_id = %q", strings.TrimSpace(task.StepID)),
		fmt.Sprintf("- current_domain_id = %q", strings.TrimSpace(task.DomainID)),
		fmt.Sprintf("- current_repo_scope = %q", strings.TrimSpace(task.RepoScope)),
		fmt.Sprintf("- current_repo_scopes = %s", strings.Join(task.RepoScopes, ", ")),
		"- Current target identity comes from repo_scope/repo_scopes/domain_id and the current staged artifacts, not from matrix id, batch id, profile id, workspace path, or run-folder names.",
		"- If current_repo_scopes contains exactly one repo, final markdown must not name sibling matrix targets or other repositories unless an allowed staged artifact, final index, citation index, or shard status file explicitly names that repo as evidence.",
		"- Matrix/profile/batch names such as combined multi-target folder names are harness labels, not architecture evidence.",
		"- Final markdown must not cite taskrun identifiers other than current_run_id. If older workspace artifacts are visible, do not use them as current-run evidence and do not list their run_* paths as reviewed indexes.",
		"DRAFT ENRICHMENT CURRENT-RUN SHARD STATUS EVIDENCE:",
	}
	if len(statusEvidenceFiles) == 0 {
		lines = append(lines,
			"- No current-run typed shard-plan/shard-summary files were visible in the allowed read roots; use observed staging/shards coverage and call unknowns out explicitly.",
		)
	} else {
		lines = append(lines,
			"- Read these exact current-run typed shard-plan/shard-summary files before falling back to staging/shards counts:",
		)
		for _, file := range statusEvidenceFiles {
			lines = append(lines, fmt.Sprintf("- %s", file))
		}
	}
	lines = append(lines,
		"- For typed shard-summary JSON with items[], planned = len(items), succeeded = count of items where status == \"succeeded\", failed = count of items where status == \"failed\"; pending/checkpointed/other statuses are incomplete coverage and must be named separately.",
		"- Do not report planned=unknown or failed=unknown when a readable current-run typed shard-summary items[] list is available.",
		"- When a readable typed shard-summary shows failed=0 and no pending/checkpointed/other statuses, write that there are no failed or incomplete shards. Do not write generic conditional phrases such as \"if present above\", \"any failed or incomplete shards\", \"failed shards require rerun\", or \"failed or incomplete shards remain coverage gaps\".",
		"- Do not infer shard counts from lexical occurrences of words such as failed/error/summary inside markdown or manifests.",
		"DRAFT ENRICHMENT TARGETS:",
	)
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
		"- Keep the manifest contract shape: version=1, run_id, step_id, step_contract, agent_role, optional summary/updated_at, and outputs[].",
		"- Preserve outputs[].path and outputs[].canonical_path exactly as loaded from the current manifest; do not synthesize a new outputs[] map from scratch.",
		"- If you update manifest metadata, update only top-level summary or updated_at and leave outputs[] byte-for-byte equivalent for path/canonical_path/kind/title.",
		"- Every outputs[].path must stay relative to draft_final_root and every referenced draft file must exist before exit.",
		"- Replace bootstrap-only markdown with evidence-backed content that cites concrete repositories, staged artifacts, files, services, modules, findings, or coverage gaps visible in the allowed read roots.",
		"- Final markdown must read as an operator-facing architecture/report/proposal artifact, not as a runtime recovery log. Do not make draft manifests, draft roots, enrichment, recovery, bounded reads, or staged evidence mechanics the subject of the report.",
		"- Do not read every staged shard document. Prefer all shard-pack-manifest.json files plus at most 6 authored markdown docs selected for architectural signal, then cite remaining coverage as summarized from manifests/indexes.",
		"- A no-op rewrite is invalid: every referenced markdown draft must be freshly rewritten with marker-free evidence-backed content, not merely re-saved unchanged.",
		"- Preserve valid non-markdown support bundles when they are already canonical; for constitution, baseline-subagents.yaml may remain the baseline YAML bundle.",
		"- Final content MUST NOT include these scaffold/recovery markers: Provider wrote this draft artifact; Drafted required runtime artifacts for this step; Draft surface initialized for the scoped repository analysis; This draft is grounded in the current step manifest; current draft manifest; manifest target remains; draft_final_root; bounded staged evidence; bounded read; recovery pass; recovery action; enrichment read; enrichment pass; Final content must stay tied to collected shard evidence and validator output; Runtime draft recovery initialized; draft recovery initialized; Treat this as diagnostic evidence until; Use collected shard manifests and validator output as the evidence source before final review; Draft surface initialized; Current run evidence should be reviewed; Runtime proposal surface initialized; bootstrap-only placeholder; placeholder draft content; placeholder draft text; placeholder content; placeholder proposal content; replace placeholder; replaced placeholder; replacing placeholders.",
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
			"- Enrich overview.md, summary.md, and architect-summary.md from collected shard manifests, bounded authored shard docs, final indexes, citations, and staged model evidence.",
			"- STEP2 WRITE-FIRST SEQUENCE: your next filesystem command must read asis-draft-manifest.json, current-run typed shard-plan/shard-summary files listed above when present, all available shard-pack-manifest.json summaries, final-run-index.json and citation-index.json if present, and at most 6 high-signal shard manifests or authored shard docs, then overwrite overview.md, summary.md, and architect-summary.md under draft_final_root before any optional extra analysis.",
			fmt.Sprintf("- Exact required as-is overview overwrite target: %q.", filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "overview.md")),
			fmt.Sprintf("- Exact required coverage summary overwrite target: %q.", filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "summary.md")),
			fmt.Sprintf("- Exact required architect summary overwrite target: %q.", filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "architect-summary.md")),
			"- overview.md must contain: architecture surface summary; concrete repositories, paths, services/modules/integrations or staged artifact references; and explicit coverage gaps.",
			"- summary.md must contain: planned/succeeded/failed shard completeness; evidence density/readability notes; key citations or staged artifact refs; and remaining gaps.",
			"- For shard completeness, derive planned/succeeded/failed from typed shard-plan/shard-summary artifacts when visible, including shard-summary items[].status; otherwise use observed shard directories and shard-pack-manifest.json counts. Never count the words failed/error/summary lexically inside manifests or markdown.",
			"- If planned shard status is not explicitly visible, write planned=unknown, succeeded=<observed shard-pack-manifest.json count>, failed=unknown, and name the missing typed shard-plan/shard-summary surface instead of fabricating failed counts.",
			"- Do not list final-run-index.json or citation-index.json from a different run_id as current-run evidence. Current-run markdown may mention only current_run_id taskrun paths.",
			"- architect-summary.md must contain: decision-ready operator summary with what is complete, what is missing, what the operator should inspect or decide next, and any residual risk.",
			"- Include enough repository/path and staged artifact references for an operator to understand the architecture surface and remaining coverage gaps.",
			"- Include a decision-ready operator summary: what is complete, what is missing, and what the operator should inspect or decide next.",
			"- Include explicit coverage gaps when any planned shard, repo path, citation, or staged evidence surface is partial or missing.",
			"- Do not write phrases like 'the required draft outputs are present', 'manifest target remains', 'bounded staged evidence', 'replace placeholder content', or 'this draft is grounded in the current step manifest'. Translate runtime evidence into architecture facts, coverage gaps, and operator decisions.",
			"- Do not stop after writing only one markdown target; all three step2 markdown targets must be freshly overwritten in this focused call.",
			"- Do not stop after saying you have enough evidence; that sentence is not an artifact and will be treated as a failed no-op enrichment.",
			"- If staged evidence is sparse, write the exact missing staged surface or shard coverage gap instead of keeping bootstrap scaffold.",
			"- Final self-check: overview.md, summary.md, and architect-summary.md were freshly overwritten in this focused call, name concrete staged evidence or repo/path references when available, and contain none of the banned scaffold markers.",
		)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines,
			"- Enrich proposal and changelog drafts from validated staged findings, coverage gaps, questions, citations, and proposal candidates.",
			"- Proposals must be actionable and traceable to staged evidence; do not leave generic validation notes as the only content.",
			"- STEP4 WRITE-FIRST SEQUENCE: read proposals-draft-manifest.json, current-run typed shard-plan/shard-summary files listed above when present, final-run-index.json and citation-index.json if present, validator/finding summaries and at most 6 high-signal shard manifests or authored shard docs, then overwrite proposal.md and changelog.md under draft_final_root before any optional extra analysis.",
			"- Do not treat proposals-draft-manifest.json summary text, canonical_path examples, or bootstrap output metadata as findings/proposals. Use validator/finding/coverage/proposal evidence; if none is visible, record an explicit no-actionable-proposal gap.",
			fmt.Sprintf("- Exact required proposal draft overwrite target: %q.", filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "proposal.md")),
			fmt.Sprintf("- Exact required changelog draft overwrite target: %q.", filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "changelog.md")),
			"- proposal.md must contain: Decision / recommended operator action; Evidence used with repo/path or staged artifact references; Proposed changes or follow-up plan; Risks, gaps, and out-of-scope notes.",
			"- changelog.md must contain: Updated architecture/proposal surfaces; Findings/proposals summary; Evidence index or citation references; Residual coverage gaps.",
			"- Do not report 0 authored markdown shard documents unless you actually globbed staging/shards/**/*.md in the allowed roots and found none; otherwise give the observed count or omit the count.",
			"- Do not ask the operator to re-run or repair non-succeeded shards when the current-run typed shard-summary shows failed=0 and no incomplete statuses; write an explicit no-shard-coverage-blocker statement instead.",
			"- Do not list final-run-index.json, citation-index.json, validator verdicts, or shard summaries from a different run_id as current-run proposal evidence.",
			"- Do not mention placeholder replacement, placeholder proposal content, replaced placeholder content, or recovery mechanics in operator-facing proposal/changelog content.",
			"- Do not write phrases like 'the enrichment read the current draft manifest', 'bounded staged evidence', 'recovery pass', or 'proposal surface only as a follow-up queue' as the primary recommendation. Convert findings into concrete operator decisions and explicitly state no-actionable-proposal only when evidence is absent.",
			"- If staged evidence is sparse, write the gap explicitly with the exact missing staged surface instead of keeping bootstrap scaffold.",
			"- Final self-check: both proposal.md and changelog.md were freshly overwritten in this focused call, name concrete staged evidence or repo/path references when available, and contain none of the banned scaffold markers.",
		)
	}
	if validationErr != nil {
		lines = append(lines, fmt.Sprintf(`- Previous draft artifact validation failure: %s`, compactDraftEnrichmentHint(validationErr.Error())))
	}
	return strings.Join(lines, "\n")
}

func draftEnrichmentShardStatusEvidenceFiles(task acpruntime.Task) []string {
	workspaceRoot := strings.TrimSpace(task.Workspace)
	runID := strings.TrimSpace(task.RunID)
	if workspaceRoot == "" || runID == "" {
		return nil
	}
	collectStep := draftEnrichmentCollectStepID(task.StepID)
	if collectStep == "" {
		return nil
	}
	stepSlug := strings.ReplaceAll(collectStep, ".", "-")
	taskrunsRoot := filepath.Join(filepath.Clean(workspaceRoot), "reports", "taskruns")
	patterns := []string{
		filepath.Join(taskrunsRoot, fmt.Sprintf("%s-%s-shard-plan*.json", runID, stepSlug)),
		filepath.Join(taskrunsRoot, fmt.Sprintf("%s-%s-shard-summary*.json", runID, stepSlug)),
	}
	files := make([]string, 0, 2)
	seen := map[string]struct{}{}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			cleaned := filepath.Clean(match)
			if _, ok := seen[cleaned]; ok {
				continue
			}
			seen[cleaned] = struct{}{}
			files = append(files, cleaned)
		}
	}
	return files
}

func draftEnrichmentCollectStepID(stepID string) string {
	switch strings.TrimSpace(stepID) {
	case "init.step2.asis_docs", "init.step4.proposals":
		return "init.step1.collect"
	case "refresh.step2.asis_docs", "refresh.step4.proposals":
		return "refresh.step1.collect"
	default:
		return ""
	}
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
