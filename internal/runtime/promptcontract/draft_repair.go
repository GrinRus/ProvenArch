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
			"- Read the current-run staged final findings markdown before writing final proposal text: reports/taskruns/<run_id>/staging/final/reports/findings/findings.md, or the absolute staging/final findings path listed in read_context_roots. Finding IDs are markdown fields like '- ID: `finding.example`'; copy exact current-run IDs into proposal and changelog when any such line exists.",
			"- Do not look for current-run findings under reports/taskruns/<run_id>/reports/findings/findings.md; before publish, current-run findings and coverage are staged under reports/taskruns/<run_id>/staging/final/reports/.",
			"- Do not shorten the staged findings file to staging/final/reports/findings.md. The exact file is staging/final/reports/findings/findings.md; strip backticks from '- ID: `...`' lines and never emit synthetic placeholders such as no-current-run-finding-id, no structured current-run finding ID, or finding unavailable.",
			"- If findings include Severity: `high` or Severity: `medium`, proposal.md must include a bullet-only Top Actionable Findings section with one bullet per high/medium finding.",
			"- Each actionable finding bullet must keep all required fields on the same bullet line: exact Finding ID, copied Severity value from that finding block, Affected surface/path copied from Related IDs/Evidence, Recommended operator action with a concrete verb such as update, add, document, assign, or remediate, and Residual gap.",
			"- Do not split one finding across multiple bullets; a separate Description bullet after a Finding ID bullet does not satisfy actionability. Do not write Severity: unspecified when findings.md has a - Severity: field; copy high/medium/low exactly from the referenced finding. Example: - Finding ID: `finding.example`; Severity: `medium`; Affected surface/path: `svc.example` / `repo:path`; Recommended operator action: document the owner and escalation path; Residual gap: production evidence remains unconfirmed.",
			"- Do not use markdown tables for actionable findings; write compact bullets instead.",
			"- Do not satisfy high/medium findings with generic inspect/review/decide text only, and do not cite only low-severity findings when high/medium findings are present.",
			"- Never write `No structured finding summary was present` or `No source-level architecture change is approved` when current-run findings.md contains any `- ID:` line.",
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
	if draftEnrichmentValidationMentionsCompactStep2Retry(validationErr) {
		return composeDraftArtifactEnrichmentCompactStep2RetryPrompt(provider, task, manifestFile, manifestTarget, outputs, statusEvidenceFiles, validationErr)
	}
	if draftEnrichmentValidationMentionsCompactStep4Retry(validationErr) {
		return composeDraftArtifactEnrichmentCompactStep4RetryPrompt(provider, task, manifestFile, manifestTarget, outputs, statusEvidenceFiles, validationErr)
	}
	if draftEnrichmentValidationMentionsCommandTextRetry(validationErr) {
		return composeDraftArtifactEnrichmentCommandTextRetryPrompt(provider, task, manifestFile, manifestTarget, outputs, statusEvidenceFiles, validationErr)
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
		"- Prefer not to rewrite the draft manifest during enrichment; if it is structurally valid, leave it byte-for-byte and rewrite only markdown.",
		"- If the manifest must be touched, keep this exact manifest shape: top-level keys version, run_id, step_id, step_contract, agent_role, summary, updated_at, outputs.",
		"- Do not add top-level status, enriched_at, metadata, validation, confidence, source, content_digest, or any provider-invented manifest field.",
		"- Do not add, remove, rename, or alias outputs[] fields. The only allowed output object keys are path, canonical_path, kind, and title.",
		"- Never add outputs[].status, outputs[].content_digest, logical_path, target, output_path, publish_path, or other output aliases; the strict parser treats them as runtime_contract_failed.",
		"- Use only the bounded read_context_roots, the current write_root/draft_final_root files, and staged evidence already available to this provider.",
		fmt.Sprintf(`- bounded_read_context_roots = %q`, strings.Join(task.ReadContextRoots, ", ")),
		"- Do not write non-draft runtime artifacts, raw logs, sibling taskruns, workspace source-of-truth files, or repository files.",
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
	}
	if isStep0DraftEnrichmentStep(task.StepID) {
		lines = append(lines,
			"- Current target identity comes from repo_scope/repo_scopes/domain_id and repository entrypoint evidence, not from matrix id, batch id, profile id, workspace path, run-folder names, staged artifacts, or later pipeline evidence.",
			"- Step0 runs before later pipeline evidence exists. Do not read, mention, count, or summarize downstream evidence surfaces in constitution markdown.",
			"- Final markdown must not cite taskrun identifiers, runtime provider names, generated timestamps, draft manifests, draft roots, recovery mechanics, or future pipeline outputs.",
			"DRAFT ENRICHMENT STEP0 REPOSITORY EVIDENCE:",
			"- Use bounded repository entrypoint evidence from read_context_roots and the charter contract. If repository evidence is sparse, record that exact repository evidence gap.",
		)
	} else {
		lines = append(lines,
			"- Current target identity comes from repo_scope/repo_scopes/domain_id and the current staged artifacts, not from matrix id, batch id, profile id, workspace path, or run-folder names.",
			"- If current_repo_scopes contains exactly one repo, final markdown must not name sibling matrix targets or other repositories unless an allowed staged artifact, final index, citation index, or shard status file explicitly names that repo as evidence.",
			"- Matrix/profile/batch names such as combined multi-target folder names are harness labels, not architecture evidence.",
			"- Final markdown must not cite taskrun identifiers other than current_run_id. If older workspace artifacts are visible, do not use them as current-run evidence and do not list their run_* paths as reviewed indexes.",
			"DRAFT ENRICHMENT CURRENT-RUN SHARD STATUS EVIDENCE:",
		)
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
		)
	}
	lines = append(lines,
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
		"- Replace bootstrap-only markdown with evidence-backed content that cites concrete evidence visible in the allowed read roots.",
		"- Final markdown must read as an operator-facing architecture/report/proposal artifact, not as a runtime recovery log. Do not make draft manifests, draft roots, enrichment, recovery, bounded reads, or staged evidence mechanics the subject of the report.",
		"- Final markdown must summarize structured JSON evidence in readable prose or compact bullets. Do not paste raw JSON, Python dict/list reprs, `documents=[{...}]`, `citations=[{...}]`, `{'id': ...}`, or truncated object fragments.",
		"- Final markdown must be syntactically readable: every inline-code/path backtick pair must be balanced on the same non-fence line, and code fences must be closed before exit.",
		"- Do not copy raw authored-shard prose fragments that contain backticks, especially truncated excerpts. Paraphrase signals instead, or write paths without inline-code formatting when truncating or summarizing.",
		"- Do not append sampled shard first paragraphs after an evidence path. A valid evidence bullet is a path plus a short paraphrased signal; never paste heading/body snippets that may include unbalanced inline-code markers.",
		"- Do not end prose sentences with stray backticks. If any summary text contains an unmatched backtick, remove the backtick or rewrite the sentence before exit.",
		"- Do not perform an unbounded evidence sweep. Prefer compact high-signal evidence reads before the first markdown rewrite.",
		"- A no-op rewrite is invalid: every referenced markdown draft must be freshly rewritten with marker-free evidence-backed content, not merely re-saved unchanged.",
		"- Preserve valid non-markdown support bundles when they are already canonical; for constitution, baseline-subagents.yaml may remain the baseline YAML bundle.",
		"- Final content MUST NOT include these scaffold/recovery markers: Provider wrote this draft artifact; Drafted required runtime artifacts for this step; Draft surface initialized for the scoped repository analysis; This draft is grounded in the current step manifest; current draft manifest; manifest target remains; draft_final_root; bounded staged evidence; bounded evidence read; bounded read roots; bounded read pass; recovery pass; recovery action; enrichment read; enrichment pass; later-pipeline evidence placeholder; Runtime draft recovery initialized; draft recovery initialized; Treat this as diagnostic evidence until; Draft surface initialized; Current run evidence should be reviewed; Runtime proposal surface initialized; bootstrap-only placeholder; placeholder draft content; placeholder draft text; placeholder content; placeholder proposal content; replace placeholder; replaced placeholder; replacing placeholders.",
		"- Final action must be: ensure the draft manifest and every referenced draft file exist, then ensure every referenced markdown draft changed and contains no unchanged bootstrap/recovery scaffold text.",
	)
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		charterTarget := filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "charter-overview.md")
		lines = append(lines,
			"- STEP0 CONSTITUTION WRITE-FIRST SEQUENCE: only repository entrypoint evidence is valid for this step. Do not wait for later pipeline evidence.",
			"- Your next filesystem command must read the current constitution-draft.json, current charter-overview.md, and bounded repository entrypoint evidence from read_context_roots, then overwrite charter-overview.md under draft_final_root before any optional extra analysis.",
			fmt.Sprintf("- Exact required constitution overview overwrite target: %q.", charterTarget),
			"- Enrich charter-overview.md with concrete constitution content from read_context_roots, repo scope, and charter wizard contract when available.",
			"- charter-overview.md must contain: target identity; repository evidence used with repo/path references; architecture scope; operating principles or constraints; coverage gaps; and a decision-ready operator summary.",
			"- If repository evidence is sparse, write the exact missing evidence category as a coverage gap; do not keep bootstrap text or mention that later pipeline evidence will arrive.",
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
			"- overview.md is the canonical Architecture Home and must contain non-empty sections named exactly: System at a glance; Analyzed scope; Domains and ownership; Key flows; Integrations and datastores; Where to start; Safe-change guidance; Evidence gaps and open questions.",
			"- Populate those Architecture Home sections with concrete repositories, paths, services/modules/integrations, or canonical artifact references; do not substitute generic headings such as Architecture Surface, Evidence Used, or Coverage Gaps.",
			"- In overview.md, never publish reports/taskruns/** or taskrun staging paths; staged evidence is an input only and operator navigation must use canonical reports/model/proposals paths or concrete repository paths.",
			"- summary.md must contain: planned/succeeded/failed shard completeness; evidence density/readability notes; key citations or staged artifact refs; and remaining gaps.",
			"- For shard completeness, derive planned/succeeded/failed from typed shard-plan/shard-summary artifacts when visible, including shard-summary items[].status; otherwise use observed shard directories and shard-pack-manifest.json counts. Never count the words failed/error/summary lexically inside manifests or markdown.",
			"- If planned shard status is not explicitly visible, write planned=unknown, succeeded=<observed shard-pack-manifest.json count>, failed=unknown, and name the missing typed shard-plan/shard-summary surface instead of fabricating failed counts.",
			"- Do not list final-run-index.json or citation-index.json from a different run_id as current-run evidence. Current-run markdown may mention only current_run_id taskrun paths.",
			"- final-run-index.json and citation-index.json are downstream/final staging artifacts and may not exist yet during step2. If they are absent, omit final-index availability from the as-is markdown; do not write that current-run final/citation indexes are missing, not observed, not found, or unavailable.",
			"- When reading current-run staging/final/final-run-index.json, count indexed documents from the top-level canonical_documents[] array. Do not use nonexistent documents[] fields, checked_paths[], or validation checked_paths as the document count.",
			"- When reading current-run staging/final/citation-index.json, count citations from the top-level citations[] array.",
			"- If final-run-index.json exists but canonical_documents[] cannot be parsed, write an explicit parse gap or omit the count; never infer that the current run has 0 observed documents from a missing documents[] field.",
			"- If final-run-index.json or citation-index.json are present for current_run_id, summarize counts, document titles, citation ids, and repo/path references in concise markdown. Do not paste raw object payloads, `documents=[{...}]`, `citations=[{...}]`, or Python-style dict snippets.",
			"- Do not write broken path bullets such as a lone backtick, partial prose inside backticks, or unbalanced inline-code/path references.",
			"- Do not paste sampled authored-shard snippets as semicolon-separated prose when they contain inline-code markers. Convert sampled evidence into short paraphrased facts and balanced path references.",
			"- Do not write `Shard pack manifests: none observed`, `no shard manifests observed`, or equivalent empty-shard evidence claims when typed shard-summary items[] or shard-pack-manifest.json files are visible.",
			"- architect-summary.md must contain: decision-ready operator summary with what is complete, what is missing, what the operator should inspect or decide next, and any residual risk.",
			"- Include enough repository/path and canonical artifact references for an operator to understand the architecture surface and remaining coverage gaps.",
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
			"- Also read current-run staged final reports/findings/findings.md and reports/coverage/summary.md when present, under reports/taskruns/<run_id>/staging/final/reports/. If findings.md contains one or more `- ID: finding...` entries, proposal.md and changelog.md must reference at least one current-run finding ID.",
			"- Do not read reports/taskruns/<run_id>/reports/findings/findings.md as the current-run source; that path is not the staged final evidence surface used before publish.",
			"- Do not shorten the staged findings file to staging/final/reports/findings.md. The exact file is staging/final/reports/findings/findings.md; strip backticks from '- ID: `...`' lines and never emit synthetic placeholders such as no-current-run-finding-id, no structured current-run finding ID, or finding unavailable.",
			"- For non-empty current-run findings, proposal.md must include severity summary, top actionable findings, affected surfaces/paths, recommended operator action, and residual gaps; do not write that no structured finding summary was present.",
			"- If findings.md contains any Severity: `high` or Severity: `medium` item, proposal.md must include a bullet-only Top Actionable Findings section with one bullet per high/medium finding.",
			"- Each actionable finding bullet must keep all required fields on the same bullet line: exact Finding ID, copied Severity value from that finding block, Affected surface/path copied from Related IDs/Evidence, Recommended operator action with a concrete verb such as update, add, document, assign, or remediate, and Residual gap.",
			"- Do not split one finding across multiple bullets; a separate Description bullet after a Finding ID bullet does not satisfy actionability. Do not write Severity: unspecified when findings.md has a - Severity: field; copy high/medium/low exactly from the referenced finding. Example: - Finding ID: `finding.example`; Severity: `medium`; Affected surface/path: `svc.example` / `repo:path`; Recommended operator action: document the owner and escalation path; Residual gap: production evidence remains unconfirmed.",
			"- Do not use markdown tables for actionable findings; write one compact bullet per high/medium finding instead.",
			"- Do not satisfy high/medium findings with generic inspect/review/decide text only, and do not cite only low-severity findings when high/medium findings are present.",
			"- Do not treat proposals-draft-manifest.json summary text, canonical_path examples, or bootstrap output metadata as findings/proposals. Use validator/finding/coverage/proposal evidence; if none is visible, record an explicit no-actionable-proposal gap.",
			fmt.Sprintf("- Exact required proposal draft overwrite target: %q.", filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "proposal.md")),
			fmt.Sprintf("- Exact required changelog draft overwrite target: %q.", filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "changelog.md")),
			"- proposal.md must contain: Decision / recommended operator action; Evidence used with repo/path or staged artifact references; Proposed changes or follow-up plan; Risks, gaps, and out-of-scope notes.",
			"- changelog.md must contain: Updated architecture/proposal surfaces; Findings/proposals summary; Evidence index or citation references; Residual coverage gaps.",
			"- Required proposal/changelog sections must not be empty. If structured finding/proposal evidence is absent, put an explicit no-actionable-proposal gap in the Findings/proposals summary and proposed plan instead of leaving the section blank.",
			"- Do not write dangling references such as `prioritize each finding above` or `findings listed above` unless the same proposal/changelog contains substantive findings/proposals above that sentence.",
			"- The proposed changes/follow-up plan must include at least one concrete operator action tied to current-run evidence, or an explicit no-actionable-proposal gap tied to the observed evidence set.",
			"- Do not report 0 authored markdown shard documents unless you actually globbed staging/shards/**/*.md in the allowed roots and found none; otherwise give the observed count or omit the count.",
			"- Do not ask the operator to re-run or repair non-succeeded shards when the current-run typed shard-summary shows failed=0 and no incomplete statuses; write the exact literal shard completeness string `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` plus an explicit no-shard-coverage-blocker statement in both proposal.md and changelog.md instead.",
			"- Do not list final-run-index.json, citation-index.json, validator verdicts, or shard summaries from a different run_id as current-run proposal evidence.",
			"- When reading current-run staging/final/final-run-index.json, count indexed documents from the top-level canonical_documents[] array. Do not use nonexistent documents[] fields, checked_paths[], or validation checked_paths as the document count.",
			"- When reading current-run staging/final/citation-index.json, count citations from the top-level citations[] array.",
			"- If final-run-index.json exists but canonical_documents[] cannot be parsed, write an explicit parse gap or omit the count; never infer that the current run has 0 observed documents from a missing documents[] field.",
			"- When final-run-index.json or citation-index.json are present for current_run_id, summarize counts, selected document titles, citation ids, and repo/path references. Do not paste raw object payloads, Python-style dict snippets, `{'id': ...}`, or truncated JSON fragments.",
			"- If you include final-run-index or citation-index counts, compute them inside the write command from top-level canonical_documents[] and citations[] variables, then write those exact variables into markdown. Approximate, stale, or manually guessed non-zero counts are invalid.",
			"- Do not list JSON metadata-only lines such as `\"version\": 1`, `\"run_id\": ...`, `\"pipeline\": ...`, `\"generated_at\": ...`, or `\"citation_index_path\": ...` as high-signal staged evidence. Use architecture/proposal document titles, citation ids, findings, questions, and repo/path refs instead.",
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
		if draftEnrichmentValidationMentionsManifestShape(validationErr) {
			lines = append(lines,
				"DRAFT ENRICHMENT MANIFEST SHAPE RETRY:",
				"- The previous enrichment produced markdown, but the draft manifest JSON no longer matched the strict runtime draft manifest shape.",
				"- Read the existing draft manifest and markdown targets, then restore the manifest to the allowed key set without weakening or bypassing validation.",
				"- Leave evidence-backed markdown content in place when it is already valid; otherwise rewrite every referenced markdown target again in the same filesystem command.",
				"- Remove unknown manifest fields such as status, content_digest, enriched_at, metadata, validation, confidence, source, logical_path, target, output_path, or publish_path.",
				"- Allowed top-level manifest keys are exactly version, run_id, step_id, step_contract, agent_role, summary, updated_at, and outputs.",
				"- Allowed output object keys are exactly path, canonical_path, kind, and title.",
			)
		}
		if draftEnrichmentValidationMentionsMalformedMarkdown(validationErr) {
			lines = append(lines,
				"DRAFT ENRICHMENT MARKDOWN SYNTAX RETRY:",
				"- The previous enrichment attempt failed because at least one referenced markdown file had malformed inline-code or code-fence syntax.",
				"- Rewrite every referenced markdown file again in one bounded filesystem command. Preserve the evidence-backed meaning, but remove or balance all inline backticks before exit.",
				"- Treat this as a cleanup over already written markdown and current-run typed shard evidence; do not perform another broad evidence sweep before rewriting.",
				"- Prefer plain text service/module names over inline-code when summarizing sampled shard prose. Use inline-code only for short complete paths or identifiers with both opening and closing backticks on the same line.",
				"- Remove sampled shard prose excerpts after evidence paths. Replace them with short paraphrased architecture signals such as service/API/integration coverage, not raw headings or body snippets.",
				"- Do not copy truncated shard excerpts, raw snippets, or semicolon lists that may carry half-open backticks.",
				"- Do not write stale downstream-index availability claims in step2 markdown. If final-run-index.json or citation-index.json is absent during step2, omit that index status entirely.",
				"- Do not write generic shard-gap wording when typed shard-summary shows all shards succeeded; keep exact planned/succeeded/failed/incomplete counts and explicit no-shard-coverage-blocker wording.",
				"- For step4 actionable findings, preserve bullet-only Top Actionable Findings format and do not introduce markdown tables.",
				"- Final self-check: every markdown line has balanced backticks outside fences, no raw sampled snippets remain, no stale downstream-index claim remains, bullet-only actionable findings remain table-free, and exact typed shard completeness is still present.",
			)
		}
		if draftEnrichmentValidationMentionsDownstreamIndexClaim(validationErr) {
			lines = append(lines,
				"DRAFT ENRICHMENT DOWNSTREAM INDEX CLAIM RETRY:",
				"- The previous enrichment wrote stale current-run final/citation index availability text.",
				"- Rewrite every referenced markdown file again and remove any sentence that says current-run final-run-index.json or citation-index.json is missing, not observed, unavailable, not present, not yet present, not yet available, not readable, or has 0 observed documents without a validated zero-document index.",
				"- For step2, final-run-index.json and citation-index.json are downstream artifacts; when absent, omit index availability entirely and focus on shard completeness, evidence density, readable architecture facts, coverage gaps, and operator decisions.",
				"- If a current-run final-run-index.json is present, count only top-level canonical_documents[]. If citation-index.json is present, count only top-level citations[].",
			)
		}
		if draftEnrichmentValidationMentionsShardStatusCleanup(validationErr) {
			focusTarget := draftEnrichmentShardStatusCleanupFocusTarget(validationErr)
			if isProposalsDraftStep(task.StepID) {
				lines = append(lines,
					"DRAFT ENRICHMENT SHARD STATUS CLEANUP RETRY:",
					"- The previous enrichment freshly rewrote markdown, but at least one step4 proposal target still omitted exact current-run proposal shard completeness.",
					fmt.Sprintf("- Rewrite every referenced markdown target again in one filesystem command, with special attention to %s.", focusTarget),
					"- Read the current-run typed shard-plan/shard-summary files listed above when present and compute planned, succeeded, failed, and incomplete counts from items[].status.",
					"- If the typed shard-summary shows all shards succeeded, both proposal.md and changelog.md must contain the exact literal shape planned=<n> succeeded=<n> failed=<n> incomplete=<n> and an explicit no-shard-coverage-blocker statement.",
					"- Preserve exact current-run finding IDs, copied severities, affected surface/path, recommended operator action, residual gap, and bullet-only Top Actionable Findings format.",
					"- Do not use generic conditional phrases such as any failed or incomplete shards, failed shards require rerun, failed or incomplete shards remain coverage gaps, or if present above.",
				)
			} else {
				lines = append(lines,
					"DRAFT ENRICHMENT SHARD STATUS CLEANUP RETRY:",
					"- The previous enrichment freshly rewrote markdown, but at least one step2 target still used generic conditional shard-gap wording instead of exact current-run shard status.",
					fmt.Sprintf("- Rewrite every referenced markdown target again in one filesystem command, with special attention to %s.", focusTarget),
					"- Read the current-run typed shard-plan/shard-summary files listed above when present and compute planned, succeeded, failed, and incomplete counts from items[].status.",
					"- If the typed shard-summary shows all shards succeeded, write exact counts and an explicit no-shard-coverage-blocker statement in overview.md, summary.md, and architect-summary.md.",
					"- Do not use generic conditional phrases such as any failed or incomplete shards, failed shards require rerun, failed or incomplete shards remain coverage gaps, or if present above.",
					"- The operator decision summary must say what is complete now, what residual artifact-quality risks remain, and what the operator should inspect next without suggesting nonexistent shard failures.",
				)
			}
		}
		if draftEnrichmentValidationMentionsMarkerCleanup(validationErr) {
			lines = append(lines,
				"DRAFT ENRICHMENT MARKER CLEANUP RETRY:",
				"- The previous enrichment changed markdown, but final content still included runtime-process, scaffold, or step0-downstream wording.",
				"- Rewrite every referenced markdown target again in one filesystem command; do not only edit the flagged sentence.",
				"- Do not use these phrases or close variants anywhere in final markdown: bounded read root, bounded read roots, bounded evidence, current draft manifest, draft manifest, draft space, manifest target, manifest retains, recovery, enrichment, placeholder, scaffold, runtime provider, taskrun, generated, validator output, later passes.",
				"- If evidence is sparse, say the concrete architecture evidence was not found in the current run inputs or selected repository files; do not explain runtime read limits or recovery mechanics.",
			)
			switch strings.TrimSpace(task.StepID) {
			case "init.step0.constitution":
				lines = append(lines,
					"- For step0, do not mention downstream pipeline surfaces, validation artifacts, proposal/change artifacts, shard/final/citation indexes, baseline-subagents.yaml, draft files, or future/later pipeline passes.",
					"- Rewrite charter-overview.md as a repository constitution: target identity, repo/path evidence, architecture scope, operating constraints, coverage gaps, and immediate operator decision.",
					"- Use plain wording such as \"not inspected in this constitution pass\" for gaps; do not write \"later passes\", \"validator output\", or \"pipeline artifacts\".",
				)
			case "init.step4.proposals", "refresh.step4.proposals":
				lines = append(lines,
					"- For step4, rewrite both proposal.md and changelog.md so they discuss proposal decisions, evidence files, shard completeness, citation/index counts, risks, and gaps without saying \"bounded read roots\" or describing the provider's recovery process.",
					"- If finding summaries are absent, write \"No structured finding summary was present in current-run proposal evidence\" rather than mentioning bounded roots.",
				)
			}
		}
		if draftEnrichmentValidationMentionsWriteSetCleanup(validationErr) {
			lines = append(lines,
				"DRAFT ENRICHMENT WRITE-SET CLEANUP RETRY:",
				"- Keep the step draft manifest in write_root. Do not delete, rename, or rewrite repository files, staged evidence, raw logs, sibling taskruns, or workspace source-of-truth files.",
			)
			if draftEnrichmentValidationMentionsStep0CanonicalPathCleanup(validationErr) {
				lines = append(lines,
					"- The previous step0 enrichment created the canonical publish path draft_final_root/skills/subagents.yaml. That path is forbidden during draft enrichment; the allowed draft bundle remains draft_final_root/baseline-subagents.yaml.",
					"- Your next filesystem action must delete only draft_final_root/skills/subagents.yaml and remove draft_final_root/skills only if it is empty.",
					"- Do not delete, rewrite, rename, or move draft_final_root/baseline-subagents.yaml.",
					"- Do not create or keep draft_final_root/skills/subagents.yaml; outputs[].canonical_path is publish metadata, not a draft write path.",
					"- Final self-check: draft_final_root/baseline-subagents.yaml exists, draft_final_root/skills/subagents.yaml is absent, and charter-overview.md remains the freshly enriched constitution markdown.",
				)
			} else {
				lines = append(lines,
					"- The previous enrichment wrote otherwise valid referenced draft markdown into write_root as misplaced duplicates.",
					"- Your next filesystem action must delete only the misplaced referenced markdown duplicates from write_root and leave draft markdown only under draft_final_root.",
					"- Do not create overview.md, summary.md, architect-summary.md, proposal.md, changelog.md, or charter-overview.md in write_root.",
					"- If you refresh content, write only referenced markdown targets under draft_final_root and preserve strict manifest outputs[].",
					"- Final self-check: write_root contains the step draft manifest but no referenced markdown output files; draft_final_root contains all referenced markdown outputs.",
				)
			}
		}
		if draftEnrichmentValidationMentionsWriteFirstRetry(validationErr) {
			lines = append(lines,
				"DRAFT ENRICHMENT SILENT WRITE-FIRST RETRY:",
				"- The previous enrichment was stopped before any fresh markdown mutation and produced no provider stdout/stderr.",
				"- This is one narrow retry for a silent no-write enrichment, not a generic no-action retry.",
				"- Your next response must execute exactly one bounded filesystem command before any prose or extra analysis.",
				"- The command must read the current draft manifest plus bounded current-run evidence and overwrite every referenced markdown target under draft_final_root before it exits.",
				"- Do not repeat the bootstrap/heredoc draft command, do not re-save unchanged scaffold, and do not print a plan/status note.",
				"- Success still requires provider-authored fresh marker-free markdown for every referenced markdown target; repeated silent/noop/scaffold output remains runtime_contract_failed.",
			)
		}
		lines = append(lines, fmt.Sprintf(`- Previous draft artifact validation failure: %s`, compactDraftEnrichmentHint(validationErr.Error())))
	}
	return strings.Join(lines, "\n")
}

func composeDraftArtifactEnrichmentCompactStep2RetryPrompt(provider acpruntime.Provider, task acpruntime.Task, manifestFile string, manifestTarget string, outputs []runtimedrafts.Output, statusEvidenceFiles []string, validationErr error) string {
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in compact step2 draft enrichment retry mode.", provider),
		"DRAFT ENRICHMENT COMPACT STEP2 RETRY:",
		"- This retry is only for init|refresh.step2.asis_docs after a silent write-first enrichment made no fresh markdown mutation.",
		"- Your next response must execute exactly one bounded filesystem command before any prose, plan, or status note.",
		"- The command must read a small current-run evidence set and overwrite every step2 markdown target under draft_final_root before it exits.",
		"- Do not run the earlier heredoc/bootstrap draft command and do not perform an open-ended workspace, repository, or sibling-taskrun sweep.",
		"- Do not return semantic JSON or use stdout as the artifact. Success requires fresh provider-authored markdown mutations on disk.",
		fmt.Sprintf("- Read/preserve the current manifest target: %q.", manifestTarget),
		fmt.Sprintf("- Draft root for markdown overwrites: %q.", strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- write_root = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- draft_final_root = %q`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- manifest_file = %q`, manifestFile),
		fmt.Sprintf(`- current_run_id = %q`, strings.TrimSpace(task.RunID)),
		fmt.Sprintf(`- current_step_id = %q`, strings.TrimSpace(task.StepID)),
		fmt.Sprintf(`- bounded_read_context_roots = %q`, strings.Join(task.ReadContextRoots, ", ")),
		"- Allowed writes: the existing step draft manifest in write_root and referenced draft files under draft_final_root only.",
		"- Prefer not to rewrite the manifest. If touched, keep only top-level version, run_id, step_id, step_contract, agent_role, summary, updated_at, outputs.",
		"- Preserve each outputs[] path/canonical_path/kind/title exactly; never add status, content_digest, logical_path, target, output_path, publish_path, metadata, validation, confidence, or source fields.",
		"- Compact evidence set: current asis-draft-manifest.json, typed shard-plan/shard-summary files listed below, observed staging/shards/*/shard-pack-manifest.json counts, at most 3 authored shard docs or manifests, and current-run final-run-index.json/citation-index.json only if present.",
		"- If final-run-index.json or citation-index.json are absent during step2, omit index availability from markdown. Do not claim those downstream indexes are missing, not observed, unavailable, not found, or zero-document.",
	}
	if len(statusEvidenceFiles) == 0 {
		lines = append(lines,
			"- No current-run typed shard-plan/shard-summary files were visible; count observed shard-pack-manifest.json files and state typed shard status as unknown.",
		)
	} else {
		lines = append(lines, "- Read these exact current-run typed shard-plan/shard-summary files first:")
		for _, file := range statusEvidenceFiles {
			lines = append(lines, "- "+file)
		}
	}
	lines = append(lines,
		"- If typed shard-summary items[] is readable, compute planned=len(items), succeeded=count(status==\"succeeded\"), failed=count(status==\"failed\"), incomplete=count(status not succeeded/failed).",
		"- When typed shard-summary shows all shards succeeded, write this exact class of statement in summary.md and architect-summary.md: \"Shard completeness: 16/16 succeeded; no failed, pending, or incomplete shard statuses were observed in the current-run typed shard summary.\"",
		"- Do not infer shard counts from lexical occurrences of failed/error in markdown or manifests.",
		"- Required markdown overwrite targets:",
	)
	markdownTargets := 0
	for _, output := range outputs {
		if strings.ToLower(filepath.Ext(strings.TrimSpace(output.Path))) != ".md" {
			continue
		}
		markdownTargets++
		lines = append(lines, fmt.Sprintf(
			"- %s -> %s; exact target %q",
			strings.TrimSpace(output.Path),
			strings.TrimSpace(output.CanonicalPath),
			filepath.Join(strings.TrimSpace(task.DraftFinalRoot), filepath.FromSlash(strings.TrimSpace(output.Path))),
		))
	}
	if markdownTargets == 0 {
		for _, target := range []string{"overview.md", "summary.md", "architect-summary.md"} {
			lines = append(lines, fmt.Sprintf("- %s; exact target %q", target, filepath.Join(strings.TrimSpace(task.DraftFinalRoot), target)))
		}
	}
	lines = append(lines,
		"- overview.md is the canonical Architecture Home and must contain non-empty sections named exactly: System at a glance; Analyzed scope; Domains and ownership; Key flows; Integrations and datastores; Where to start; Safe-change guidance; Evidence gaps and open questions.",
		"- Populate those Architecture Home sections with concrete repo/path or canonical artifact references; do not substitute generic headings such as Architecture Surface, Evidence Used, or Coverage Gaps.",
		"- In overview.md, never publish reports/taskruns/** or taskrun staging paths; staged evidence is an input only and operator navigation must use canonical reports/model/proposals paths or concrete repository paths.",
		"- summary.md must summarize shard completeness, evidence density/readability, selected citation or staged artifact refs, and remaining gaps.",
		"- architect-summary.md must give a decision-ready operator summary: what is complete, what is missing, what to inspect or decide next, and residual risk.",
		"- Evidence bullets must be path plus paraphrased signal only. Do not paste the first paragraph or heading body from authored shard markdown after the path.",
		"- Final markdown must be readable operator-facing architecture content, not a runtime process report. Do not paste raw JSON/Python object dumps, documents=[...], citations=[...], or metadata-only keys as evidence.",
		"- Final markdown must not include scaffold or process markers: Runtime draft recovery initialized; Drafted required runtime artifacts for this step; Treat this as diagnostic evidence until; Use collected shard manifests; current draft manifest; manifest target remains; draft_final_root; bounded staged evidence; bounded evidence read; bounded read roots; recovery pass; enrichment read; bootstrap-only placeholder; placeholder draft content; replace placeholder; replacing placeholders.",
		"- Final self-check inside the command: overview.md, summary.md, and architect-summary.md were freshly overwritten, have balanced backticks/fences, include current-run evidence or explicit gaps, and contain none of the banned markers.",
	)
	if validationErr != nil {
		lines = append(lines, fmt.Sprintf(`- Previous draft artifact validation failure: %s`, compactDraftEnrichmentHint(validationErr.Error())))
	}
	return strings.Join(lines, "\n")
}

func composeDraftArtifactEnrichmentCompactStep4RetryPrompt(provider acpruntime.Provider, task acpruntime.Task, manifestFile string, manifestTarget string, outputs []runtimedrafts.Output, statusEvidenceFiles []string, validationErr error) string {
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in compact step4 draft enrichment retry mode.", provider),
		"DRAFT ENRICHMENT COMPACT STEP4 RETRY:",
		"- This retry is only for init|refresh.step4.proposals after a silent write-first enrichment made no fresh proposal/changelog markdown mutation.",
		"- Your next response must execute exactly one bounded filesystem command before any prose, plan, or status note.",
		"- The command must read a small current-run proposal evidence set and overwrite every step4 markdown target under draft_final_root before it exits.",
		"- Do not run the earlier heredoc/bootstrap draft command and do not perform an open-ended workspace, repository, or sibling-taskrun sweep.",
		"- Do not return semantic JSON or use stdout as the artifact. Success requires fresh provider-authored proposal/changelog markdown mutations on disk.",
		fmt.Sprintf("- Read/preserve the current manifest target: %q.", manifestTarget),
		fmt.Sprintf("- Draft root for markdown overwrites: %q.", strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- write_root = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- draft_final_root = %q`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- manifest_file = %q`, manifestFile),
		fmt.Sprintf(`- current_run_id = %q`, strings.TrimSpace(task.RunID)),
		fmt.Sprintf(`- current_step_id = %q`, strings.TrimSpace(task.StepID)),
		fmt.Sprintf(`- bounded_read_context_roots = %q`, strings.Join(task.ReadContextRoots, ", ")),
		"- Allowed writes: the existing proposals draft manifest in write_root and referenced draft files under draft_final_root only.",
		"- Prefer not to rewrite the manifest. If touched, keep only top-level version, run_id, step_id, step_contract, agent_role, summary, updated_at, outputs.",
		"- Preserve each outputs[] path/canonical_path/kind/title exactly; never add status, content_digest, logical_path, target, output_path, publish_path, metadata, validation, confidence, or source fields.",
		"- Compact evidence set: current proposals-draft-manifest.json, typed shard-plan/shard-summary files listed below, current-run validator/finding/proposal/coverage summaries if present, current-run final-run-index.json/citation-index.json if present, and at most 3 staged shard docs or manifests.",
		"- If current-run staged final reports/findings/findings.md is visible and contains finding IDs, proposal.md and changelog.md must cite at least one current-run finding ID; proposal.md must include severity summary, affected surfaces/paths, recommended operator action, and residual gaps.",
		"- Current-run findings and coverage are under reports/taskruns/<run_id>/staging/final/reports/ before publish; do not treat reports/taskruns/<run_id>/reports/* as current-run evidence.",
		"- Do not shorten the staged findings file to staging/final/reports/findings.md. The exact file is staging/final/reports/findings/findings.md; strip backticks from '- ID: `...`' lines and never emit synthetic placeholders such as no-current-run-finding-id, no structured current-run finding ID, or finding unavailable.",
		"- If findings.md contains Severity: `high` or Severity: `medium`, proposal.md must include a bullet-only Top Actionable Findings section with one bullet per high/medium finding and all required fields on the same bullet line: exact Finding ID, copied Severity value from that finding block, Affected surface/path, Recommended operator action, and Residual gap.",
		"- Do not split one finding across multiple bullets; do not write Severity: unspecified when findings.md has a - Severity: field; copy high/medium/low exactly from the referenced finding.",
		"- Do not use markdown tables for actionable findings.",
		"- Generic inspect/review/decide wording is not enough for high/medium findings; use a concrete operator verb such as update, add, document, assign, or remediate and cite the affected Related IDs/Evidence path.",
		"- Never claim structured findings are absent when current-run findings.md contains finding IDs.",
		"- If proposal/finding evidence is sparse, still overwrite both files with an explicit no-actionable-proposal gap tied to observed current-run evidence instead of waiting for more evidence.",
	}
	if len(statusEvidenceFiles) == 0 {
		lines = append(lines,
			"- No current-run typed shard-plan/shard-summary files were visible; count observed shard-pack-manifest.json files and state typed shard status as unknown.",
		)
	} else {
		lines = append(lines, "- Read these exact current-run typed shard-plan/shard-summary files first:")
		for _, file := range statusEvidenceFiles {
			lines = append(lines, "- "+file)
		}
	}
	lines = append(lines,
		"- If typed shard-summary items[] is readable, compute planned=len(items), succeeded=count(status==\"succeeded\"), failed=count(status==\"failed\"), incomplete=count(status not succeeded/failed).",
		"- When typed shard-summary shows all shards succeeded, write the exact literal shard completeness string `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` and an explicit no-shard-coverage-blocker statement in both proposal.md and changelog.md.",
		"- Do not infer shard counts from lexical occurrences of failed/error in markdown or manifests.",
		"- Required markdown overwrite targets:",
	)
	markdownTargets := 0
	for _, output := range outputs {
		if strings.ToLower(filepath.Ext(strings.TrimSpace(output.Path))) != ".md" {
			continue
		}
		markdownTargets++
		lines = append(lines, fmt.Sprintf(
			"- %s -> %s; exact target %q",
			strings.TrimSpace(output.Path),
			strings.TrimSpace(output.CanonicalPath),
			filepath.Join(strings.TrimSpace(task.DraftFinalRoot), filepath.FromSlash(strings.TrimSpace(output.Path))),
		))
	}
	if markdownTargets == 0 {
		for _, target := range []string{"proposal.md", "changelog.md"} {
			lines = append(lines, fmt.Sprintf("- %s; exact target %q", target, filepath.Join(strings.TrimSpace(task.DraftFinalRoot), target)))
		}
	}
	lines = append(lines,
		"- proposal.md must include Decision / recommended operator action, evidence used, proposed changes or follow-up plan, and risks/gaps/out-of-scope.",
		"- changelog.md must include updated architecture/proposal surfaces, findings/proposals summary, evidence index/citation refs, and residual coverage gaps.",
		"- Required proposal/changelog sections must not be empty. Do not use dangling references like `findings above`; include the findings/proposals inline or state an explicit no-actionable-proposal gap tied to current-run evidence.",
		"- The proposed changes/follow-up plan must include a concrete evidence-backed operator action, unless the proposal explicitly says no actionable proposal evidence was present.",
		"- Final markdown must be readable operator-facing proposal content, not a runtime process report. Do not paste raw JSON/Python object dumps, documents=[...], citations=[...], {'id': ...}, or metadata-only keys as evidence.",
		"- Final markdown must not include scaffold or process markers: Runtime draft recovery initialized; Runtime proposal surface initialized; Treat this as diagnostic evidence until; Use collected shard manifests; current draft manifest; manifest target remains; draft_final_root; bounded staged evidence; bounded evidence read; bounded read roots; recovery pass; enrichment read; bootstrap-only placeholder; placeholder proposal content; replace placeholder; replacing placeholders.",
		"- Final self-check inside the command: proposal.md and changelog.md were freshly overwritten, have balanced backticks/fences, include current-run evidence or explicit gaps, and contain none of the banned markers.",
	)
	if validationErr != nil {
		lines = append(lines, fmt.Sprintf(`- Previous draft artifact validation failure: %s`, compactDraftEnrichmentHint(validationErr.Error())))
	}
	return strings.Join(lines, "\n")
}

func composeDraftArtifactEnrichmentCommandTextRetryPrompt(provider acpruntime.Provider, task acpruntime.Task, manifestFile string, manifestTarget string, outputs []runtimedrafts.Output, statusEvidenceFiles []string, validationErr error) string {
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in draft artifact enrichment command-text retry mode.", provider),
		"DRAFT ENRICHMENT COMMAND-TEXT RETRY:",
		"- The previous focused enrichment returned shell/Python command text instead of executing a filesystem mutation.",
		"- Your next response must be exactly one filesystem command, not a plan, not prose, not a status note.",
		"- Do not print the command, fenced code, or a Python script as assistant text. The command must actually execute and mutate files before exit.",
		"- A plain-text response containing `python3 - <<'PY'` without filesystem mutation is classified as failed command-text enrichment.",
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
		"- Prefer not to rewrite the manifest; if it is structurally valid, leave it byte-for-byte and only rewrite markdown.",
		"- If you update the manifest, preserve each outputs[] path/canonical_path/kind/title exactly; never add logical_path, target, output_path, status, content_digest, or aliases.",
		"- Allowed top-level manifest keys are version, run_id, step_id, step_contract, agent_role, summary, updated_at, and outputs only; never add status, enriched_at, metadata, validation, confidence, source, or provider-invented fields.",
		"- Allowed output object keys are path, canonical_path, kind, and title only.",
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
	if isStep0DraftEnrichmentStep(task.StepID) {
		lines = append(lines,
			"- For step0, do not read current-run downstream evidence artifacts; those surfaces are invalid for constitution.",
			"- Step0 final markdown must not mention runtime providers, generated timestamps, taskrun/log/report mechanics, draft manifests, draft roots, recovery/enrichment, downstream indexes, downstream checks, downstream reports, or future pipeline outputs.",
			"- Banned final markdown markers: Runtime draft recovery initialized; Draft surface initialized; Treat this as diagnostic evidence until; Runtime proposal surface initialized; Current run evidence should be reviewed; placeholder; bootstrap-only; recovery pass; enrichment read; bounded staged evidence; current draft manifest; draft_final_root; runtime provider; Produced by; Generated; taskrun; replace placeholder; replaced placeholder; replacing placeholders.",
			"- Final self-check inside the command: charter-overview.md was freshly overwritten, is marker-free, has balanced backticks/fences, and contains repository evidence, coverage gaps, and next decision content.",
		)
	} else {
		lines = append(lines,
			"- Also read current-run staging/final final-run-index.json and citation-index.json if present.",
			"- For final-run-index.json, count documents from top-level canonical_documents[] only; never infer 0 observed documents from a missing documents[] field.",
			"- For citation-index.json, count citations from top-level citations[] only.",
			"- Also read validator/finding/coverage/proposal summaries and up to 6 high-signal staging/shards manifests or authored markdown docs.",
			"- Any final-run-index/citation-index counts included in markdown must be computed by the command from parsed JSON variables and must match the current files exactly; otherwise omit counts.",
			"- Do not list metadata-only JSON keys such as `\"version\": 1`, `\"run_id\"`, `\"pipeline\"`, `\"generated_at\"`, or `\"citation_index_path\"` as evidence bullets.",
			"- Banned final markdown markers: Runtime draft recovery initialized; Draft surface initialized; Treat this as diagnostic evidence until; Use collected shard manifests; Runtime proposal surface initialized; Current run evidence should be reviewed; placeholder; bootstrap-only; recovery pass; enrichment read; bounded staged evidence; current draft manifest; draft_final_root; replace placeholder; replaced placeholder; replacing placeholders.",
			"- Final self-check inside the command: every markdown target was freshly overwritten, is marker-free, has balanced backticks/fences, and contains operator-facing evidence, gaps, and next decision content.",
		)
	}
	if draftEnrichmentValidationMentionsCommandTextRetry(validationErr) {
		lines = append(lines,
			"- The previous enrichment printed a shell/Python command as text instead of executing it. This retry is accepted only if the provider runtime observes actual file mutations under draft_final_root.",
		)
	}
	if draftEnrichmentValidationMentionsManifestShape(validationErr) {
		lines = append(lines,
			"DRAFT ENRICHMENT MANIFEST SHAPE RETRY:",
			"- The previous enrichment changed the manifest shape. Your command must restore the strict manifest shape and keep or refresh evidence-backed markdown.",
			"- Remove unknown manifest fields such as status, content_digest, enriched_at, metadata, validation, confidence, source, logical_path, target, output_path, or publish_path.",
		)
	}
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		lines = append(lines,
			fmt.Sprintf("- Exact required constitution overview overwrite target: %q.", filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "charter-overview.md")),
			"- Read bounded repository entrypoint evidence from the allowed read_context_roots before writing the step0 summary; later pipeline evidence is invalid for this step.",
			"- For step0, overwrite charter-overview.md with target identity, repository evidence, architecture scope, operating principles/constraints, coverage gaps, and a decision-ready operator summary.",
			"- Preserve baseline-subagents.yaml when it is already a valid baseline bundle.",
		)
		if candidates := draftEnrichmentStep0EvidenceCandidates(task); len(candidates) > 0 {
			lines = append(lines, "- STEP0 bounded repository evidence candidates:")
			for _, candidate := range candidates {
				lines = append(lines, "- "+candidate)
			}
		}
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines,
			"- For step2, overwrite overview.md, summary.md, and architect-summary.md.",
			"- overview.md is the canonical Architecture Home and must contain non-empty sections named exactly: System at a glance; Analyzed scope; Domains and ownership; Key flows; Integrations and datastores; Where to start; Safe-change guidance; Evidence gaps and open questions.",
			"- Populate those Architecture Home sections with concrete repo/path or canonical artifact references; do not substitute generic headings such as Architecture Surface, Evidence Used, or Coverage Gaps.",
			"- In overview.md, never publish reports/taskruns/** or taskrun staging paths; staged evidence is an input only and operator navigation must use canonical reports/model/proposals paths or concrete repository paths.",
			"- summary.md must state shard completeness from typed shard status when visible plus evidence density/readability and gaps.",
			"- architect-summary.md must state what is complete, what is missing, and what the operator should inspect or decide next.",
			"- If a typed shard-summary JSON with items[] is visible, compute planned=len(items), succeeded=count(status==\"succeeded\"), failed=count(status==\"failed\"), and incomplete=count of pending/checkpointed/other statuses.",
			"- When typed shard-summary shows all shards succeeded, summary.md must include an exact statement such as \"Shard completeness: 16/16 succeeded; no failed, pending, or incomplete shard statuses were observed in the current-run typed shard summary.\"",
			"- Do not dump shard-summary metadata keys such as meta, step_id, domain_id, strategy, max_parallel_tasks, failure_policy, or shard_discovery_mode as evidence bullets.",
			"- Do not claim the staging shard directory contains 0 files or 0 shards when typed shard-summary items[] or shard-pack-manifest.json files are visible.",
			"- Do not write `Shard pack manifests: none observed`, `no shard manifests observed`, or equivalent empty-shard evidence claims when typed shard-summary items[] or shard-pack-manifest.json files are visible.",
		)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines,
			"- For step4, overwrite proposal.md and changelog.md.",
			"- For step4 command-text retry, the command must perform at most one bounded evidence listing/read pass before writing both markdown files; do not run an open-ended repo or staged-artifact sweep before the first mutation.",
			"- Read current-run staged final findings.md, coverage summary, final-run-index.json, citation-index.json, and typed shard status when visible before writing proposal/changelog markdown.",
			"- Current-run findings and coverage are under reports/taskruns/<run_id>/staging/final/reports/ before publish; do not read reports/taskruns/<run_id>/reports/* as current-run evidence.",
			"- The exact findings file is reports/taskruns/<run_id>/staging/final/reports/findings/findings.md, not reports/taskruns/<run_id>/staging/final/reports/findings.md; strip backticks from '- ID: `...`' lines and never emit synthetic placeholders such as no-current-run-finding-id, no structured current-run finding ID, or finding unavailable.",
			"- If current-run findings.md contains finding IDs, proposal.md and changelog.md must cite at least one current-run finding ID and must not claim structured findings are absent.",
			"- If current-run findings.md contains Severity: `high` or Severity: `medium`, proposal.md must include a bullet-only Top Actionable Findings section with one bullet per high/medium finding and all required fields on the same bullet line: exact Finding ID, copied Severity value from that finding block, Affected surface/path, Recommended operator action, and Residual gap.",
			"- Do not split one finding across multiple bullets; do not write Severity: unspecified when findings.md has a - Severity: field; copy high/medium/low exactly from the referenced finding.",
			"- Do not use markdown tables for actionable findings.",
			"- Do not satisfy high/medium findings with generic inspect/review/decide text only; cite Related IDs/Evidence as affected surface/path and use a concrete operator verb such as update, add, document, assign, or remediate.",
			"- If proposal evidence is sparse, still overwrite both files with a decision-ready no-actionable-proposal gap tied to observed current-run evidence instead of waiting for more evidence.",
			"- proposal.md must include Decision / recommended operator action, evidence used, proposed changes or follow-up plan, risks/gaps/out-of-scope.",
			"- changelog.md must include updated architecture/proposal surfaces, findings/proposals summary, evidence index/citation refs, and residual coverage gaps.",
			"- None of those required proposal/changelog sections may be empty; avoid dangling references to findings above unless the same file includes those findings.",
			"- The proposed changes/follow-up plan must contain at least one concrete operator action tied to current-run evidence, or an explicit no-actionable-proposal gap.",
			"- If typed shard status shows all shards succeeded, write the exact literal shard completeness string `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` and an explicit no-shard-coverage-blocker statement in both proposal.md and changelog.md.",
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

func draftEnrichmentValidationMentionsMarkerCleanup(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "bootstrap-only placeholder draft content") ||
		strings.Contains(text, "mentions downstream or runtime-only evidence in step0 constitution content")
}

func isStep0DraftEnrichmentStep(stepID string) bool {
	return strings.TrimSpace(stepID) == "init.step0.constitution"
}

func draftEnrichmentValidationMentionsManifestShape(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "parse runtime draft manifest: json: unknown field") ||
		(strings.Contains(text, "runtime draft manifest outputs are invalid:") && strings.Contains(text, "unknown field"))
}

func draftEnrichmentValidationMentionsDownstreamIndexClaim(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "claims current-run final/citation indexes are unavailable") ||
		strings.Contains(text, "claims current-run final-run-index has zero observed documents")
}

func draftEnrichmentValidationMentionsCommandTextRetry(err error) bool {
	return err != nil && strings.Contains(err.Error(), "draft_artifact_enrichment_command_text_retry")
}

func draftEnrichmentValidationMentionsCompactStep2Retry(err error) bool {
	return err != nil && strings.Contains(err.Error(), "draft_artifact_enrichment_compact_step2_retry")
}

func draftEnrichmentValidationMentionsCompactStep4Retry(err error) bool {
	return err != nil && strings.Contains(err.Error(), "draft_artifact_enrichment_compact_step4_retry")
}

func draftEnrichmentValidationMentionsWriteFirstRetry(err error) bool {
	return err != nil && strings.Contains(err.Error(), "draft_artifact_enrichment_write_first_retry")
}

func draftEnrichmentValidationMentionsWriteSetCleanup(err error) bool {
	return err != nil && strings.Contains(err.Error(), "draft_artifact_enrichment_write_set_cleanup")
}

func draftEnrichmentValidationMentionsStep0CanonicalPathCleanup(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "forbidden draft_final_root files") &&
		strings.Contains(text, "skills/subagents.yaml")
}

func isProposalsDraftStep(stepID string) bool {
	switch strings.TrimSpace(stepID) {
	case "init.step4.proposals", "refresh.step4.proposals":
		return true
	default:
		return false
	}
}

func draftEnrichmentValidationMentionsShardStatusCleanup(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "draft_artifact_enrichment_shard_status_cleanup") ||
		strings.Contains(text, "generic conditional shard-gap wording") ||
		strings.Contains(text, "uses generic conditional shard-gap wording") ||
		strings.Contains(text, "does not report exact current-run proposal shard completeness") ||
		strings.Contains(text, "does not report exact current-run shard completeness") ||
		strings.Contains(text, "claims staging shard evidence is empty") ||
		strings.Contains(text, "does not include concrete repo/path, citation, or staged artifact evidence references") ||
		strings.Contains(text, "does not include a decision-ready operator summary")
}

func draftEnrichmentShardStatusCleanupFocusTarget(err error) string {
	if err == nil {
		return "architect-summary.md"
	}
	text := err.Error()
	for _, target := range []string{"proposal.md", "changelog.md", "overview.md", "summary.md", "architect-summary.md"} {
		if strings.Contains(text, `path "`+target+`"`) {
			return target
		}
	}
	for _, target := range []string{"summary.md", "architect-summary.md", "overview.md"} {
		if strings.Contains(text, `"`+target+`"`) || strings.Contains(text, target) {
			return target
		}
	}
	return "architect-summary.md"
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
