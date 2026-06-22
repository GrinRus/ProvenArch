package promptcontract

import (
	"encoding/json"
	"fmt"
	"os"
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
	if draftEnrichmentValidationMentionsNoActionRetry(validationErr) {
		return composeDraftArtifactEnrichmentNoActionRetryPrompt(provider, task, manifestFile, manifestTarget, outputs, statusEvidenceFiles, validationErr)
	}
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in draft artifact enrichment focused recovery mode.", provider),
		"Immediate draft artifact enrichment action:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		"- Do not answer with a plan, status note, or analysis-only message. Your next action must be a filesystem command that rewrites every referenced markdown draft target.",
		"- Forbidden analysis-only phrases before the rewrite: I have enough evidence; I will rewrite; I will now rewrite; ready to rewrite; the drafts will cite.",
		"- Do not run the earlier heredoc/bootstrap draft command again.",
		"- Do not create or preserve recovery scaffold text as final content.",
		"- First focused work unit: execute one bounded filesystem command that reads the current draft manifest and bounded staged evidence, then rewrites every referenced markdown target in that same command before any optional extended analysis.",
		"- If you use Python for this bounded filesystem command, invoke python3 explicitly. Never invoke python; some trusted live hosts do not provide a python binary.",
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
		"- When a readable typed shard-summary shows failed=0 and no pending/checkpointed/other statuses, write exact current-run counts and an explicit no-shard-coverage-blocker statement such as \"Shard completeness: 16/16 succeeded; no failed, pending, or incomplete shard statuses were observed in the current-run typed shard summary.\" Do not write generic conditional phrases such as \"if present above\", \"any failed or incomplete shards\", \"failed shards require rerun\", or \"failed or incomplete shards remain coverage gaps\".",
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
		"- Final markdown must summarize structured JSON evidence in readable prose or compact bullets. Do not paste raw JSON, Python dict/list reprs, `documents=[{...}]`, `citations=[{...}]`, `{'id': ...}`, or truncated object fragments.",
		"- Final markdown must be syntactically readable: every inline-code/path backtick pair must be balanced on the same non-fence line, and code fences must be closed before exit.",
		"- Do not copy raw authored-shard prose fragments that contain backticks, especially truncated excerpts. Paraphrase signals instead, or write paths without inline-code formatting when truncating or summarizing.",
		"- Do not end prose sentences with stray backticks. If any summary text contains an unmatched backtick, remove the backtick or rewrite the sentence before exit.",
		"- Do not read every staged shard document. Prefer all shard-pack-manifest.json files plus at most 6 authored markdown docs selected for architectural signal, then cite remaining coverage as summarized from manifests/indexes.",
		"- A no-op rewrite is invalid: every referenced markdown draft must be freshly rewritten with marker-free evidence-backed content, not merely re-saved unchanged.",
		"- Preserve valid non-markdown support bundles when they are already canonical; for constitution, baseline-subagents.yaml may remain the baseline YAML bundle.",
		"- Final content MUST NOT include these scaffold/recovery markers: Provider wrote this draft artifact; Drafted required runtime artifacts for this step; Draft surface initialized for the scoped repository analysis; This draft is grounded in the current step manifest; current draft manifest; manifest target remains; draft_final_root; bounded staged evidence; bounded read; recovery pass; recovery action; enrichment read; enrichment pass; Final content must stay tied to collected shard evidence and validator output; Runtime draft recovery initialized; draft recovery initialized; Treat this as diagnostic evidence until; Use collected shard manifests and validator output as the evidence source before final review; Draft surface initialized; Current run evidence should be reviewed; Runtime proposal surface initialized; bootstrap-only placeholder; placeholder draft content; placeholder draft text; placeholder content; placeholder proposal content; replace placeholder; replaced placeholder; replacing placeholders.",
		"- Final action must be: ensure the draft manifest and every referenced draft file exist, then ensure every referenced markdown draft changed and contains no unchanged bootstrap/recovery scaffold text.",
	)
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		charterTarget := filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "charter-overview.md")
		lines = append(lines,
			"- STEP0 CONSTITUTION WRITE-FIRST SEQUENCE: collected shards and validator output do not exist yet for this step. Do not wait for later pipeline evidence.",
			"- Your next filesystem command must read the current constitution-draft.json, current charter-overview.md, and bounded repository entrypoint evidence from read_context_roots, then overwrite charter-overview.md under draft_final_root before any optional extra analysis.",
			fmt.Sprintf("- Exact required constitution overview overwrite target: %q.", charterTarget),
			"- Enrich charter-overview.md with concrete constitution content from read_context_roots, repo scope, and charter wizard contract when available.",
			"- charter-overview.md must contain: target identity; repository evidence used with repo/path references; architecture scope; operating principles or constraints; coverage gaps; and a decision-ready operator summary.",
			"- If repository evidence is sparse, write the exact missing evidence category as a coverage gap; do not keep bootstrap text or mention that collected shards/validator output will arrive later.",
			"- Do not rewrite baseline-subagents.yaml unless it is invalid; it must remain a valid baseline agents bundle.",
			"- Final self-check: charter-overview.md was freshly overwritten in this focused call, includes at least one concrete repo/path evidence reference when available, and contains none of the banned scaffold markers.",
		)
		if candidates := draftEnrichmentStep0EvidenceCandidates(task); len(candidates) > 0 {
			lines = append(lines, "- STEP0 bounded repository evidence candidates:")
			for _, candidate := range candidates {
				lines = append(lines, "- "+candidate)
			}
		}
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
			"- final-run-index.json and citation-index.json are downstream/final staging artifacts and may not exist yet during step2. If they are absent, omit final-index availability from the as-is markdown; do not write that current-run final/citation indexes are missing, not found, or unavailable.",
			"- If final-run-index.json or citation-index.json are present for current_run_id, summarize counts, document titles, citation ids, and repo/path references in concise markdown. Do not paste raw object payloads, `documents=[{...}]`, `citations=[{...}]`, or Python-style dict snippets.",
			"- Do not write broken path bullets such as a lone backtick, partial prose inside backticks, or unbalanced inline-code/path references.",
			"- Do not paste sampled authored-shard snippets as semicolon-separated prose when they contain inline-code markers. Convert sampled evidence into short paraphrased facts and balanced path references.",
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
			"- Do not ask the operator to re-run or repair non-succeeded shards when the current-run typed shard-summary shows failed=0 and no incomplete statuses; write exact planned/succeeded/failed/incomplete counts plus an explicit no-shard-coverage-blocker statement in both proposal.md and changelog.md instead.",
			"- Do not list final-run-index.json, citation-index.json, validator verdicts, or shard summaries from a different run_id as current-run proposal evidence.",
			"- When final-run-index.json or citation-index.json are present for current_run_id, summarize counts, selected document titles, citation ids, and repo/path references. Do not paste raw object payloads, Python-style dict snippets, `{'id': ...}`, or truncated JSON fragments.",
			"- Do not paste sampled shard markdown snippets with inline-code markers into proposal.md or changelog.md. If a sampled signal includes backticks, paraphrase it in plain text or use a fully balanced path reference.",
			"- Do not write stale index availability claims such as `No current-run final-run-index document list was available`; if final-run-index.json is absent, omit that index status, and if it is present, summarize the observed canonical document count.",
			"- Do not write stale zero-count claims such as `final-run-index.json contains 0 observed document entries` unless you have validated a current-run zero-document index; normally, summarize the observed canonical document count or omit the count.",
			"- Do not claim citation detail is limited or unavailable when current-run citation-index.json contains citation entries. State the observed citation count or omit the claim.",
			"- Do not mention placeholder replacement, placeholder proposal content, replaced placeholder content, or recovery mechanics in operator-facing proposal/changelog content.",
			"- Do not write phrases like 'the enrichment read the current draft manifest', 'bounded staged evidence', 'recovery pass', or 'proposal surface only as a follow-up queue' as the primary recommendation. Convert findings into concrete operator decisions and explicitly state no-actionable-proposal only when evidence is absent.",
			"- If staged evidence is sparse, write the gap explicitly with the exact missing staged surface instead of keeping bootstrap scaffold.",
			"- Final self-check: both proposal.md and changelog.md were freshly overwritten in this focused call, name concrete staged evidence or repo/path references when available, and contain none of the banned scaffold markers.",
		)
	}
	if validationErr != nil {
		if draftEnrichmentValidationMentionsMalformedMarkdown(validationErr) {
			lines = append(lines,
				"DRAFT ENRICHMENT MARKDOWN SYNTAX RETRY:",
				"- The previous enrichment attempt failed because at least one referenced markdown file had malformed inline-code or code-fence syntax.",
				"- Rewrite every referenced markdown file again. Preserve the evidence-backed meaning, but remove or balance all inline backticks before exit.",
				"- Prefer plain text service/module names over inline-code when summarizing sampled shard prose. Use inline-code only for short complete paths or identifiers with both opening and closing backticks on the same line.",
				"- Do not copy truncated shard excerpts, raw snippets, or semicolon lists that may carry half-open backticks.",
			)
		}
		lines = append(lines, fmt.Sprintf(`- Previous draft artifact validation failure: %s`, compactDraftEnrichmentHint(validationErr.Error())))
	}
	return strings.Join(lines, "\n")
}

func composeDraftArtifactEnrichmentNoActionRetryPrompt(provider acpruntime.Provider, task acpruntime.Task, manifestFile string, manifestTarget string, outputs []runtimedrafts.Output, statusEvidenceFiles []string, validationErr error) string {
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in draft artifact enrichment no-action retry mode.", provider),
		"DRAFT ENRICHMENT NO-ACTION RETRY:",
		"- The previous focused enrichment did not freshly replace every referenced markdown target with valid evidence-backed content.",
		"- Your next response must be exactly one filesystem command, not a plan, not prose, not a status note.",
		"- The command must use python3, read bounded current-run evidence, and overwrite every markdown target listed below before it exits.",
		"- Do not run or copy the earlier heredoc/bootstrap draft command.",
		"- Do not write deterministic filler, raw JSON dumps, placeholder text, or recovery mechanics as final markdown.",
		fmt.Sprintf("- Manifest target to read/preserve: %q.", manifestTarget),
		fmt.Sprintf("- Draft root for markdown overwrites: %q.", strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- write_root = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- draft_final_root = %q`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- manifest_file = %q`, manifestFile),
		fmt.Sprintf(`- current_run_id = %q`, strings.TrimSpace(task.RunID)),
		fmt.Sprintf(`- current_step_id = %q`, strings.TrimSpace(task.StepID)),
		fmt.Sprintf(`- bounded_read_context_roots = %q`, strings.Join(task.ReadContextRoots, ", ")),
		"- Allowed writes: the existing step draft manifest in write_root and referenced draft files under draft_final_root only.",
		"- If you update the manifest, preserve each outputs[] path/canonical_path/kind/title exactly; never add logical_path, target, output_path, or aliases.",
		"- Required markdown overwrite targets:",
	}
	if len(outputs) == 0 {
		lines = append(lines, "- Load outputs[] from the existing manifest and overwrite every referenced markdown target.")
	} else {
		for _, output := range outputs {
			if strings.ToLower(filepath.Ext(strings.TrimSpace(output.Path))) != ".md" {
				continue
			}
			lines = append(lines, fmt.Sprintf(
				"- %s -> %s (%s)",
				strings.TrimSpace(output.Path),
				strings.TrimSpace(output.CanonicalPath),
				strings.TrimSpace(output.Kind),
			))
		}
	}
	if len(statusEvidenceFiles) > 0 {
		lines = append(lines, "- Read these current-run shard status files if present:")
		for _, file := range statusEvidenceFiles {
			lines = append(lines, "- "+file)
		}
	}
	lines = append(lines,
		"- Also read current-run staging/final final-run-index.json and citation-index.json if present.",
		"- Also read validator/finding/coverage/proposal summaries and up to 6 high-signal staging/shards manifests or authored markdown docs.",
		"- Banned final markdown markers: Runtime draft recovery initialized; Treat this as diagnostic evidence until; Use collected shard manifests; Runtime proposal surface initialized; Current run evidence should be reviewed; placeholder; bootstrap-only; recovery pass; enrichment read; bounded staged evidence; current draft manifest; draft_final_root; replace placeholder; replaced placeholder; replacing placeholders.",
		"- Final self-check inside the command: every markdown target was freshly overwritten, is marker-free, has balanced backticks/fences, and contains operator-facing evidence, gaps, and next decision content.",
	)
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		lines = append(lines,
			"- For step0, overwrite charter-overview.md with target identity, repository evidence, architecture scope, operating principles/constraints, coverage gaps, and a decision-ready operator summary.",
			"- Preserve baseline-subagents.yaml when it is already a valid baseline bundle.",
		)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines,
			"- For step2, overwrite overview.md, summary.md, and architect-summary.md.",
			"- overview.md must summarize architecture surfaces with concrete repo/path or staged-artifact evidence.",
			"- summary.md must state shard completeness from typed shard status when visible plus evidence density/readability and gaps.",
			"- architect-summary.md must state what is complete, what is missing, and what the operator should inspect or decide next.",
		)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines,
			"- For step4, overwrite proposal.md and changelog.md.",
			"- proposal.md must include Decision / recommended operator action, evidence used, proposed changes or follow-up plan, risks/gaps/out-of-scope.",
			"- changelog.md must include updated architecture/proposal surfaces, findings/proposals summary, evidence index/citation refs, and residual coverage gaps.",
			"- If typed shard status shows all shards succeeded, write exact planned/succeeded/failed/incomplete counts and an explicit no-shard-coverage-blocker statement.",
		)
	}
	if validationErr != nil {
		lines = append(lines, fmt.Sprintf(`- Previous draft artifact validation failure: %s`, compactDraftEnrichmentHint(validationErr.Error())))
	}
	return strings.Join(lines, "\n")
}

func draftEnrichmentValidationMentionsMalformedMarkdown(err error) bool {
	return err != nil && strings.Contains(err.Error(), "malformed markdown inline-code")
}

func draftEnrichmentValidationMentionsNoActionRetry(err error) bool {
	return err != nil && strings.Contains(err.Error(), "draft_artifact_enrichment_no_action_retry")
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

func draftEnrichmentStep0EvidenceCandidates(task acpruntime.Task) []string {
	candidateRelPaths := []string{
		"README.md",
		"AGENTS.md",
		"package.json",
		"pyproject.toml",
		"go.mod",
		"pom.xml",
		"build.gradle",
		"settings.gradle",
		"docker-compose.yml",
		"docker-compose.yaml",
		filepath.Join(".github", "CODEOWNERS"),
	}
	workspaceRoot := filepath.Clean(strings.TrimSpace(task.Workspace))
	out := make([]string, 0, 8)
	seen := map[string]struct{}{}
	for _, root := range task.ReadContextRoots {
		cleanRoot := filepath.Clean(strings.TrimSpace(root))
		if cleanRoot == "." || cleanRoot == "" {
			continue
		}
		if workspaceRoot != "." && workspaceRoot != "" && pathIsUnder(cleanRoot, workspaceRoot) {
			continue
		}
		for _, rel := range candidateRelPaths {
			candidate := filepath.Clean(filepath.Join(cleanRoot, rel))
			if _, ok := seen[candidate]; ok {
				continue
			}
			info, err := os.Stat(candidate)
			if err != nil || info.IsDir() {
				continue
			}
			seen[candidate] = struct{}{}
			out = append(out, candidate)
			if len(out) >= 8 {
				return out
			}
		}
	}
	return out
}

func pathIsUnder(pathValue string, root string) bool {
	if pathValue == "" || root == "" {
		return false
	}
	rel, err := filepath.Rel(root, pathValue)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
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
