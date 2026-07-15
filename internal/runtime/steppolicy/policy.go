package steppolicy

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func PrimaryTaskRepoScope(explicit string, scopes []string) string {
	if value := strings.TrimSpace(explicit); value != "" {
		return value
	}
	for _, scope := range scopes {
		if value := strings.TrimSpace(scope); value != "" {
			return value
		}
	}
	return ""
}

func TopLevelSemanticOutputRule(stepID string) string {
	if isCollectStep(stepID) {
		return `Do NOT emit any top-level semantic payload on stdout; shard-pack-manifest.json is the only semantic result surface for collect steps.`
	}
	if runtimedrafts.IsDraftStep(stepID) {
		return `Do NOT emit semantic payloads on stdout; runtime draft metadata belongs only inside constitution-draft.json / asis-draft-manifest.json / proposals-draft-manifest.json under write_root.`
	}
	return `Do NOT emit semantic payloads on stdout unless the step contract explicitly defines a runtime artifact for them under write_root.`
}

func StepSpecificPolicy(stepID string) string {
	switch strings.TrimSpace(stepID) {
	case "init.step0.constitution":
		return strings.Join([]string{
			`STEP POLICY init.step0.constitution:`,
			`- Do NOT delegate to agent/subagent helpers.`,
			`- Do NOT use todo_write-style planning or long plan narration.`,
			`- Use only existing repo entrypoint hints; do not assume README.md exists when it is not present.`,
			`- After the first useful evidence pass, converge to constitution drafts instead of continuing a broad repo sweep.`,
			`- Same provider turn requirement: after the first constitution draft skeleton exists, the very next filesystem action must do a bounded repository-entrypoint evidence read and fresh-overwrite charter-overview.md before any final answer, status note, or optional extra analysis.`,
			`- Do not answer with "I will read", "I will rewrite", "ready for repair", or similar analysis-only text between the skeleton write and the enriched charter-overview.md overwrite.`,
			`- charter-overview.md must only use workspace/charter/repository-entrypoint evidence. Do not mention later collection steps, later analysis, future pipeline passes, downstream checks, collected shards, validator output, final indexes, citations, proposal artifacts, runtime repair, providers, or taskrun mechanics.`,
			`- Express unknowns as "not confirmed in the current constitution evidence" instead of pointing to later/downstream pipeline stages.`,
			`- constitution-draft.json must use the runtime draft manifest contract exactly; legacy constitution schemas are forbidden.`,
			`- This is a draft-only step; do not invent semantic entities, edges, findings, or questions on stdout.`,
		}, "\n")
	case "init.step1.collect":
		return strings.Join([]string{
			`STEP POLICY init.step1.collect:`,
			`- Do NOT delegate to agent/subagent helpers.`,
			`- Do NOT use todo_write-style planning or long plan narration.`,
			`- Use only existing repo entrypoint hints; do not assume README.md exists when it is not present.`,
			`- After the first useful evidence pass, converge to authored docs plus shard-pack-manifest.json instead of continuing a broad repo sweep.`,
			`- Markdown-only completion is invalid: shard-pack-manifest.json is the required collect result contract for every shard.`,
			`- Treat schemas/*, docs/spec/*, and the enforced prompt contract as the only schema source of truth for shard-pack-manifest.json.`,
			`- Do NOT inspect reports/taskruns, prior shard-pack-manifest.json files, raw logs, or archive docs to infer collect manifest shape.`,
			`- Every semantic provenance.evidence[] item must include non-empty repo and path values that resolve to real repository evidence.`,
			`- Citation-only semantic evidence objects such as {"citation_id":"..."} are forbidden.`,
			`- semantic.questions[*] must use id + text; do NOT omit stable question ids.`,
			`- semantic.findings[*] must use id + severity + title + description + provenance; do NOT use summary as a compatibility alias.`,
			`- If this shard includes evidence from multiple repositories, encode at least one semantic edge, finding, or question that names the cross-repo relationship/gap.`,
		}, "\n")
	case "refresh.step1.collect":
		return strings.Join([]string{
			`STEP POLICY refresh.step1.collect:`,
			`- Do NOT delegate to agent/subagent helpers.`,
			`- Do NOT use todo_write-style planning or long plan narration.`,
			`- Use only existing repo entrypoint hints; do not assume README.md exists when it is not present.`,
			`- After the first useful evidence pass, converge to authored docs plus shard-pack-manifest.json instead of continuing a broad repo sweep.`,
			`- Markdown-only completion is invalid: shard-pack-manifest.json is the required collect result contract for every shard.`,
			`- Treat schemas/*, docs/spec/*, and the enforced prompt contract as the only schema source of truth for shard-pack-manifest.json.`,
			`- Do NOT inspect reports/taskruns, prior shard-pack-manifest.json files, raw logs, or archive docs to infer collect manifest shape.`,
			`- Allowed semantic.entities[*].type values: service, datastore, integration, external.system, team, domain, api, component.`,
			`- Forbidden placeholder entity types: runtime_provider, runtime, metadata.`,
			`- Analyze only repository/workspace artifacts; do NOT perform web search or external browsing.`,
			`- Every provenance.evidence.path must resolve to an existing file in workspace/repo scope.`,
			`- Every provenance.evidence.repo must name the repository that owns the cited path.`,
			`- Do NOT emit synthetic evidence paths such as search_source/*, search_query/*, search_config/*.`,
			`- Citation-only semantic evidence objects such as {"citation_id":"..."} are forbidden.`,
			`- Do NOT introduce unrelated incident domains (for example bidding/tender/power-system topics) unless explicitly present in repository evidence.`,
			`- If evidence is incomplete, capture gap via coverage.missing instead of synthetic placeholder entities.`,
			`- semantic.questions[*] must use id + text; do NOT omit stable question ids.`,
			`- semantic.findings[*] must use id + severity + title + description + provenance; do NOT use summary as a compatibility alias.`,
			`- If refresh evidence spans multiple repositories, encode at least one semantic edge, finding, or question that names the cross-repo relationship/gap.`,
			`- Include at least one question and at least three items in coverage.missing.`,
		}, "\n")
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return strings.Join([]string{
			fmt.Sprintf(`STEP POLICY %s:`, strings.TrimSpace(stepID)),
			`- Do NOT delegate to agent/subagent helpers.`,
			`- Do NOT use todo_write-style planning or long plan narration.`,
			`- Use staged final evidence from read_context_roots; do not treat sibling baseline workspaces, prior draft manifests, or prior reports as template sources.`,
			`- Evidence-first draft requirement: the first filesystem work unit must read bounded current-run staged evidence, write overview.md, summary.md, and architect-summary.md, then write asis-draft-manifest.json last before any final answer, status note, or optional extra analysis.`,
			`- Do not write bootstrap-only as-is markdown as the normal happy path; first-pass drafts must already be evidence-backed or must state explicit evidence-backed insufficiency tied to current-run coverage/questions.`,
			`- Do not answer with "I will read", "I will rewrite", "I have enough evidence", "ready for repair", or similar analysis-only text before the evidence-backed as-is draft write.`,
			`- Markdown writes must be shell-safe: use single-quoted heredocs (<<'EOF') or python3 Path.write_text with literal strings; never place markdown containing backticks inside double-quoted shell strings or unquoted heredocs because shell command substitution can erase path/citation refs.`,
			`- When current-run typed shard-plan/shard-summary evidence is visible, summary.md and architect-summary.md must include exact planned/succeeded/failed/incomplete counts before successful exit.`,
			`- When typed shard status shows failed=0 and incomplete=0, summary.md and architect-summary.md must include an explicit no-shard-coverage-blocker statement that says current-run shard coverage is not a blocker; do not write generic failed/incomplete caveats.`,
			`- Treat schemas/*, docs/spec/*, and the enforced prompt contract as the only manifest source of truth for asis-draft-manifest.json.`,
			`- Keep step_contract exactly "as_is"; null, empty, or alternate values are invalid.`,
			`- The first authored draft set must include asis-draft-manifest.json plus validation-ready overview.md, summary.md, and architect-summary.md under draft_final_root.`,
			`- Do NOT register legacy metadata envelopes or repo_scopes/path_scopes fields in asis-draft-manifest.json.`,
		}, "\n")
	case "init.step3.findings", "refresh.step3.findings":
		return strings.Join([]string{
			fmt.Sprintf(`STEP POLICY %s:`, strings.TrimSpace(stepID)),
			`- Write validator-verdict.json only; do not write shard manifests or semantic snapshots for this step.`,
			`- validator-verdict.json must include version=1, run_id, generated_at, verdict, and checked_paths.`,
			`- validator-verdict.json issues[] must use code/severity/message plus optional path/document_id/citation_id only.`,
			`- Do NOT use legacy issue fields id/title/description/rule_id/related_paths inside issues[].`,
			`- findings[] items must use title + description + provenance; do NOT use summary as a finding alias.`,
			`- For observation provenance, findings[*].provenance.evidence[] must be non-empty and each evidence item must include repo/path.`,
			`- If repo_scopes contains multiple repositories or staged citations span multiple repositories, include at least one semantic edge or a PASS-compatible finding/question that names the cross-repo relationship/gap and links at least two repository scopes.`,
			`- Cross-repo findings must cite concrete repo/path evidence for every related repository; cross-repo questions must use stable related_ids for at least two repository scopes and be backed by repo-specific citations in checked artifacts.`,
			`- If owner mapping remains unresolved in evidence, include at least one finding and at least one question in validator-verdict.json.`,
			`- Owner-gap findings/questions may coexist with verdict PASS when no technical validator issues remain.`,
		}, "\n")
	case "init.step4.proposals", "refresh.step4.proposals":
		return strings.Join([]string{
			fmt.Sprintf(`STEP POLICY %s:`, strings.TrimSpace(stepID)),
			`- Do NOT delegate to agent/subagent helpers.`,
			`- Do NOT use todo_write-style planning or long plan narration.`,
			`- Use validated staged final evidence from read_context_roots; do not treat prior proposal reports or final indexes as manifest templates.`,
			`- Evidence-first draft requirement: the first filesystem work unit must read bounded current-run findings/coverage/index evidence and then write proposals-draft-manifest.json plus proposal.md and changelog.md before any final answer, status note, or optional extra analysis.`,
			`- Do not write bootstrap-only proposal/changelog markdown as the normal happy path; first-pass drafts must already be evidence-backed or must state explicit evidence-backed insufficiency tied to validator findings/coverage.`,
			`- Do not answer with "I will read", "I will rewrite", "I have enough evidence", "ready for repair", or similar analysis-only text before the evidence-backed proposal/changelog draft write.`,
			`- If current-run staged final reports/findings/findings.md contains finding IDs, proposal.md and changelog.md must cite at least one exact current-run finding ID before successful exit.`,
			`- Treat schemas/*, docs/spec/*, and the enforced prompt contract as the only manifest source of truth for proposals-draft-manifest.json.`,
			`- Keep step_contract exactly "proposals"; null, empty, or alternate values are invalid.`,
			`- outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/*.`,
			`- Do NOT register legacy top-level fields: pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps.`,
		}, "\n")
	case acpruntime.StepIDQAAsk:
		return strings.Join([]string{
			`STEP POLICY qa.ask:`,
			`- Answer the user question only from the provided QA context pack.`,
			`- Do NOT inspect source repositories, reports/taskruns history, raw logs, or sibling workspaces.`,
			`- Do NOT mutate canonical workspace artifacts, source repositories, schemas, docs/spec, charter, reports, model, or proposals.`,
			`- Write exactly one semantic answer artifact: qa-answer.json under write_root.`,
			`- Every citation path must be one of the workspace-relative paths listed in context-pack.json documents[].path.`,
			`- If context evidence is insufficient or contradictory, say so in unresolved instead of inventing owners, paths, decisions, or runtime outcomes.`,
		}, "\n")
	default:
		if strings.HasPrefix(strings.TrimSpace(stepID), "refresh.") {
			return `For refresh steps, keep unresolved gaps explicit in artifacts instead of inventing placeholder semantic payloads.`
		}
		return ""
	}
}

func QAFirstActionSection(task acpruntime.Task) string {
	if !acpruntime.IsQAStep(task.StepID) {
		return ""
	}
	return strings.Join([]string{
		`FIRST QA ANSWER COMMAND:`,
		`- First read context-pack.json from the exact context_pack_path below.`,
		`- Then write qa-answer.json to the exact absolute write_root target below before exiting successfully.`,
		`- The JSON object must follow the canonical qa-answer shape exactly.`,
		fmt.Sprintf(`- question = %q`, strings.TrimSpace(task.Question)),
		fmt.Sprintf(`- context_pack_path = %q`, strings.TrimSpace(task.ContextPackPath)),
		fmt.Sprintf(`- qa_answer_path = %q`, filepath.Join(strings.TrimSpace(task.WriteRoot), "qa-answer.json")),
	}, "\n")
}

func DocFirstFilesystemPolicy(task acpruntime.Task) string {
	readContextRootsJSON := "[]"
	if raw, err := json.Marshal(task.ReadContextRoots); err == nil {
		readContextRootsJSON = string(raw)
	}
	lines := []string{
		`DOCS-FIRST FILESYSTEM CONTRACT:`,
		`- Read only from meta.workspace and meta.path_scopes plus runtime read_context_roots; do not treat workspace root as implicit write target.`,
		`- Write ONLY inside write_root. Never write to workspace.yaml, schemas/*, docs/spec/*, charter/*, or analyzed user repositories.`,
		`- Use tool calls for file writes. Stdout/stderr are diagnostics only and are not a semantic result surface.`,
		fmt.Sprintf(`- task_id = %q`, strings.TrimSpace(task.TaskID)),
		fmt.Sprintf(`- run_id = %q`, strings.TrimSpace(task.RunID)),
		fmt.Sprintf(`- step_id = %q`, strings.TrimSpace(task.StepID)),
		fmt.Sprintf(`- artifact_root (workspace-relative) = %q`, strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- draft_final_root (absolute) = %q`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- read_context_roots = %s`, readContextRootsJSON),
		fmt.Sprintf(`- repo_scopes = %s`, jsonStringList(task.RepoScopes)),
		fmt.Sprintf(`- path_scopes = %s`, jsonStringList(task.PathScopes)),
		fmt.Sprintf(`- domain_id = %q`, strings.TrimSpace(task.DomainID)),
		fmt.Sprintf(`- agent_role = %q`, strings.TrimSpace(task.AgentRole)),
		fmt.Sprintf(`- step_contract = %q`, strings.TrimSpace(task.StepContract)),
		fmt.Sprintf(`- expected_artifacts = %s`, strings.Join(task.ExpectedArtifacts, ", ")),
		`- Every required artifact write/check MUST use the exact absolute write_root or draft_final_root paths above.`,
		`- Relative CWD checks/writes such as test -f validator-verdict.json or test -f overview.md are invalid for runtime artifacts.`,
	}
	if entrypointHints := CollectRepoEntrypointHints(task); len(entrypointHints) > 0 {
		if isCollectStep(task.StepID) {
			lines = append(lines, fmt.Sprintf(`- Existing repo entrypoint hints (read these first for the bounded collect evidence pass): %s`, strings.Join(entrypointHints, ", ")))
		} else {
			lines = append(lines, fmt.Sprintf(`- Existing repo entrypoint hints (read only these first when relevant): %s`, strings.Join(entrypointHints, ", ")))
		}
	} else if isPromptHintedStep(task.StepID) {
		lines = append(lines, `- Repo entrypoint hints are limited to actually existing files; do not assume README.md exists when it is absent.`)
	}
	if scopeHints := CollectPathScopeFileHints(task); len(scopeHints) > 0 && isCollectStep(task.StepID) {
		lines = append(lines, fmt.Sprintf(`- Existing path-scope file candidates (use these concrete files for bounded collect before citing deeper paths): %s`, strings.Join(scopeHints, ", ")))
	}
	switch strings.TrimSpace(task.StepID) {
	case acpruntime.StepIDQAAsk:
		lines = append(lines,
			`- QA mode reads only the generated context pack and writes only qa-answer.json in write_root.`,
			fmt.Sprintf(`- User question = %q`, strings.TrimSpace(task.Question)),
			fmt.Sprintf(`- Context pack path = %q`, strings.TrimSpace(task.ContextPackPath)),
			fmt.Sprintf(`- Absolute QA answer target: %q.`, filepath.Join(strings.TrimSpace(task.WriteRoot), "qa-answer.json")),
			`- qa-answer.json canonical shape:`,
			QAAnswerCanonicalExample(task),
		)
	case "init.step0.constitution":
		lines = append(lines,
			`- Do NOT delegate to agent/subagent helpers and do NOT use todo_write-style planning.`,
			`- Write constitution-draft.json in write_root.`,
			`- Write the referenced draft files exactly at draft_final_root/charter-overview.md and draft_final_root/baseline-subagents.yaml.`,
			`- Do NOT place the draft files under draft_final_root/charter/ or draft_final_root/skills/; those are canonical publish paths, not draft file locations.`,
			`- The first constitution draft artifact set is bootstrap-only; before final exit, replace placeholder scaffold text in charter-overview.md with evidence-backed charter content from the configured repository scope and charter wizard contract when available.`,
			`- Same provider turn requirement: do not wait for focused repair; run a bounded repository-entrypoint enrichment rewrite of charter-overview.md in this normal turn before successful exit.`,
			`- Final charter-overview.md must not mention later collection steps, later analysis, future pipeline passes, downstream checks, collected shards, validator output, final indexes, citations, proposal artifacts, runtime repair, providers, or taskrun mechanics.`,
			`- Express unknowns as current constitution evidence gaps, not as work delegated to later/downstream steps.`,
			`- If constitution-draft.json already describes the publish surface, stop only after confirming charter-overview.md is not an unchanged bootstrap placeholder; baseline-subagents.yaml may remain the baseline bundle.`,
			`- constitution-draft.json must use the exact runtime draft manifest shape shown below; do not emit legacy constitution schemas.`,
			`- outputs[] must map charter-overview.md -> charter/overview.md and baseline-subagents.yaml -> skills/subagents.yaml exactly.`,
			`- Exact constitution-draft.json example (replace IDs/summary only, keep keys/types and output mapping):`,
			ConstitutionDraftManifestExample(task),
			`- Keep the draft deterministic in shape; compiler will normalize/publish canonical files afterwards.`,
		)
	case "init.step1.collect", "refresh.step1.collect":
		lines = append(lines,
			`- Do NOT delegate to agent/subagent helpers and do NOT use todo_write-style planning.`,
			`- The first collect filesystem work unit may contain only two mechanically simple commands: one bounded evidence read/list, then one direct literal write of the authored document plus shard-pack-manifest.json.`,
			`- Cap the bounded evidence read/list to at most 8 representative files and at most the first 6000 bytes from each file; oversized files are truncated or skipped while the work unit continues.`,
			`- Do not run analysis-only narration, status/progress text, todo/planning, broad repository sweeps, or any second read-only preflight before the direct literal write command.`,
			`- Produce runtime-authored documents in write_root and then write shard-pack-manifest.json in write_root.`,
			`- Evidence-write pair requirement: the first work unit writes the suggested overview doc and shard-pack-manifest.json as one focused marker-free evidence-backed artifact pair.`,
			fmt.Sprintf(`- Suggested collect authored doc path for this shard: %q. Prefer exactly this single doc path unless already writing an existing clearer authored doc.`, SuggestedCollectDocumentPath(task)),
			fmt.Sprintf(`- Absolute collect targets for the evidence-backed pair: %q and %q.`, filepath.Join(strings.TrimSpace(task.WriteRoot), filepath.FromSlash(SuggestedCollectDocumentPath(task))), filepath.Join(strings.TrimSpace(task.WriteRoot), "shard-pack-manifest.json")),
			fmt.Sprintf(`- Minimal collect target shape: write %q + "shard-pack-manifest.json" with concrete observed evidence; record remaining uncertainty in coverage/questions instead of continuing open-ended exploration.`, SuggestedCollectDocumentPath(task)),
			`- Do not wait for a complete broad repository sweep before writing shard-pack-manifest.json; the bounded first action is enough when it records observed evidence and remaining gaps honestly.`,
			`- Do not rely on focused collect repair as the expected success path; normal collect must attempt to produce the valid pair itself.`,
			`- Before both target files exist, do not use Ruby, Node, Python, Perl, awk, jq, generated source-code strings, template programs, or nested quote tricks to synthesize markdown/JSON; use direct shell heredoc/printf/tee literal writes authored from the bounded read.`,
			`- If the direct write command fails before both targets exist, immediately retry the same direct literal write pattern with simpler content from observed evidence; do not wait for collect_pair_repair.`,
			`- When writing shard-pack-manifest.json, adapt the task-specific JSON skeleton embedded in the collect evidence-first section; keep exact metadata keys and replace skeleton evidence/content values with facts you actually observed.`,
			`- Do not exit after writing markdown only; every collect shard must finish with a valid shard-pack-manifest.json.`,
			`- The final collect pair must not be seed-only, scaffold-only, or copied unchanged from the skeleton; use evidence-backed content where repository files support it.`,
			`- After the first artifact pair exists, perform a bounded enrichment pass over the assigned repo/path scope; final semantic arrays must contain repo-specific entities/edges/findings/questions or an explicit evidence-backed insufficient-evidence finding/question.`,
			`- The final collect markdown must not describe itself as an initial/temporary artifact, interrupted evidence read, or content that "will be repaired"; if concrete file evidence is unavailable, record a gap without claiming the artifact is pending later replacement.`,
			`- The final collect markdown must not mention bounded reads/passes, guessed paths/files/evidence, expected-missing path checks, recovery attempts, or runtime repair mechanics; unsupported expected files belong only in coverage gaps/questions without citations.`,
			`- shard-pack-manifest.json must describe every authored document, its canonical stable path, citations, and semantic snapshot.`,
			`- In shard-pack-manifest.json, semantic MUST include coverage, questions, entities, edges, and findings.`,
			`- Use only canonical collect vocabulary: semantic.coverage.observed, semantic.questions[*].id + semantic.questions[*].text, semantic.edges[*].type, and object-shaped provenance blocks.`,
			`- Every semantic.questions[] item must include id and text; every semantic.findings[] item must include id, severity, title, and provenance.`,
			`- Do NOT emit semantic payloads on stdout; keep semantic only inside shard-pack-manifest.json.`,
			`- You may be flexible in document structure, but promotion and rendering depend on manifest citations/topics remaining accurate.`,
			`- After writing the evidence-backed pair, avoid broad repository exploration; only minimal manifest/JSON repair needed for the current shard is allowed afterwards.`,
			`- After writing shard-pack-manifest.json, do NOT continue broad list_directory/read_file sweeps across repo roots.`,
			`- Do NOT read reports/taskruns/**, raw runtime logs, or previously generated shard-pack-manifest.json files as schema examples during collect.`,
			`- If authored docs and shard-pack-manifest.json already exist in write_root, stop only after confirming they contain marker-free scoped evidence for this shard and are not placeholder prose.`,
		)
		if rootFileScopes := rootFileShardPathScopes(task.PathScopes); len(rootFileScopes) > 0 {
			lines = append(lines,
				fmt.Sprintf(`- Root-file collect shard detected: path_scopes contains root-level files only: %s.`, strings.Join(rootFileScopes, ", ")),
				`- For this root-file shard, read only the listed root files first; do not recursively sweep top-level directories or unrelated source trees.`,
				`- Produce one concise evidence-backed root overview document in write_root, then write an enriched shard-pack-manifest.json for that document before exiting successfully.`,
			)
		}
		lines = append(lines,
			`TASK-SPECIFIC COLLECT MANIFEST JSON SKELETON: use the JSON embedded in the collect evidence-first section above as a schema/key/type guide; do not copy it unchanged and do not copy a generic manifest example.`,
			`COLLECT MANIFEST CONTRACT CHECKLIST:`,
		)
		lines = append(lines, artifactquality.CollectManifestContractLines(strings.TrimSpace(task.ArtifactRoot))...)
		lines = append(lines,
			`- The task-specific collect manifest JSON skeleton above is normative for field names and value types; replace skeleton evidence/content values with observed repository evidence.`,
		)
		lines = append(lines, artifactquality.ClaimIDContractLines()...)
		lines = append(lines,
			`- Do NOT collapse a multi-document refresh surface to one generic "cite.runtime-summary" citation when repository evidence exists.`,
			`- Preserve repo-specific citations in shard-pack-manifest.json whenever repository files support them.`,
			`- For multi-repo evidence, include at least one semantic edge or a finding/question whose provenance/evidence names the related repositories.`,
		)
	case "init.step3.findings", "refresh.step3.findings":
		lines = append(lines,
			`- Inspect staged final artifacts under reports/taskruns/<run_id>/staging/final from read_context_roots.`,
			`- Write validator-verdict.json in write_root.`,
			fmt.Sprintf(`- Absolute validator verdict target: %q.`, filepath.Join(strings.TrimSpace(task.WriteRoot), "validator-verdict.json")),
			`- Validator may fix only indexes, references, or technical document issues inside write_root; do not rewrite document meaning wholesale.`,
			`- Do NOT shard this step and do NOT emit findings through stdout; validator-verdict.json is the only primary output.`,
			`- Multi-repo release profiles require explicit cross-repo semantic signal: if repo_scopes or staged citations cover multiple repositories, encode at least one edge, finding, or question that relates two repository scopes instead of only listing repositories separately.`,
		)
		lines = append(lines, artifactquality.ValidatorVerdictContractLines()...)
		lines = append(lines,
			`- Canonical validator-verdict fragment below is normative for metadata fields and finding evidence shape; copy keys/types exactly and only change IDs/content.`,
			artifactquality.ValidatorVerdictCanonicalExample(),
			`- Canonical issues[] item example (use only when verdict has a technical validator issue):`,
			artifactquality.ValidatorVerdictIssueCanonicalExample(),
		)
		lines = append(lines, artifactquality.ClaimIDContractLines()...)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines,
			`- Write asis-draft-manifest.json in write_root.`,
			`- Write overview.md, summary.md, and architect-summary.md only under draft_final_root.`,
			`- Use the FIRST AS-IS DRAFT COMMAND above as an evidence-first write contract, not as a bootstrap placeholder command.`,
			`- The first draft artifact set must already be validation-ready: read bounded staged evidence first, then write the manifest and all three markdown targets in the same filesystem work unit.`,
			`- Same provider turn requirement: the normal turn must write all three markdown targets with bounded evidence-backed content before successful exit.`,
			`- Before successful exit, ensure the first-pass content is evidence-backed as-is content or explicit evidence-backed insufficiency tied to coverage/questions.`,
			`- Use staged final evidence from read_context_roots only; do NOT read sibling baseline workspaces or previously published as-is drafts as templates.`,
			`- If asis-draft-manifest.json already describes the publish surface, stop only after confirming referenced draft files are validation-ready and not bootstrap placeholders; do NOT re-register draft artifacts through any legacy metadata op.`,
			`- Compiler may materialize indexes and derived technical artifacts only; canonical narratives come from your drafts.`,
		)
		lines = append(lines, currentRunEvidenceIndexLines(task, currentRunEvidenceAsIs)...)
		lines = append(lines, artifactquality.AsIsDraftManifestContractLines()...)
		lines = append(lines,
			`- Canonical as-is draft fragment below is normative for field names, step_contract, and required outputs; copy keys/types exactly and only change IDs/content.`,
			artifactquality.AsIsDraftManifestCanonicalExample(),
		)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines,
			`- Inspect validated staged artifacts under reports/taskruns/<run_id>/staging/final from read_context_roots.`,
			`- Write proposals-draft-manifest.json in write_root.`,
			`- Draft final docs only under draft_final_root.`,
			`- Allowed canonical targets are proposals/* and reports/changelog/*.`,
			`- Use the FIRST PROPOSALS DRAFT COMMAND above as an evidence-first write contract, not as a bootstrap placeholder command.`,
			`- The first proposals draft artifact set must already be validation-ready: read current-run findings/coverage/index evidence first, then write the manifest, proposal.md, and changelog.md in the same filesystem work unit.`,
			`- Same provider turn requirement: the normal turn must write proposal.md and changelog.md with bounded evidence-backed content before successful exit.`,
			`- Enrichment must read current-run staged final surfaces when visible: reports/taskruns/<run_id>/staging/final/reports/findings/findings.md, reports/taskruns/<run_id>/staging/final/reports/coverage/summary.md, final-run-index.json, citation-index.json, and typed shard summary.`,
			`- Do NOT look for current-run findings at reports/taskruns/<run_id>/reports/findings/findings.md; the current-run promoted proposal evidence lives under staging/final before publish.`,
			"- Do NOT shorten the findings path to staging/final/reports/findings.md; the file is nested at staging/final/reports/findings/findings.md. IDs are usually backticked in lines like - ID: `finding.example`; strip the backticks and copy the exact ID. Never emit synthetic placeholders such as no-current-run-finding-id, no structured current-run finding ID, or finding unavailable.",
			`- If current-run findings are non-empty, proposal/changelog drafts must cite current-run finding IDs and include severity summary, top actionable findings, affected surfaces/paths, recommended operator action, and residual gaps.`,
			`- If current-run findings are non-empty, never write "Finding ID: none", "Finding ID: n/a", or "Finding ID: unavailable" in any actionable finding bullet; use an exact current-run finding ID or omit the actionable bullet when no actionable finding exists.`,
			`- If current-run findings include high or medium severity, proposal.md must include a bullet-only Top Actionable Findings section with one bullet per high/medium finding. Each bullet must keep all required fields on the same bullet line: exact Finding ID, copied Severity value from the finding block, Affected surface/path from Related IDs/Evidence, Recommended operator action using a concrete operator verb, and Residual gap.`,
			"- Do NOT split one finding across multiple bullets; a separate Description bullet after a Finding ID bullet does not satisfy actionability. Do NOT write Severity: unspecified for any finding that has a - Severity: field; copy high/medium/low exactly from findings.md. Example bullet: - Finding ID: `finding.example`; Severity: `medium`; Affected surface/path: `svc.example` / `repo:path`; Recommended operator action: document the owner and escalation path; Residual gap: production evidence remains unconfirmed.",
			`- Do not use markdown tables for actionable findings; tables are rejected because they have repeatedly produced malformed live markdown.`,
			`- Do NOT satisfy high/medium findings with generic inspect/review/decide wording only, and do NOT cite only low-severity findings when high/medium findings are present.`,
			`- Do NOT claim structured findings are absent when current-run findings.md contains finding IDs.`,
			`- Before successful exit, ensure the first-pass content is evidence-backed proposal/changelog content or explicit evidence-backed insufficiency tied to validator findings/coverage.`,
			`- If proposals-draft-manifest.json already describes the publish surface, stop only after confirming referenced draft files are validation-ready and not bootstrap placeholders; do NOT re-register draft artifacts through any legacy metadata op.`,
			`- Promotion remains deterministic; your drafts become publish candidates only after compile/publish gates.`,
		)
		lines = append(lines, currentRunEvidenceIndexLines(task, currentRunEvidenceProposals)...)
		lines = append(lines, artifactquality.ProposalsDraftManifestContractLines()...)
		lines = append(lines,
			`- Canonical proposals draft fragment below is normative for field names, step_contract, and allowed publish surface; copy keys/types exactly and only change IDs/content.`,
			artifactquality.ProposalsDraftManifestCanonicalExample(),
		)
	}
	return strings.Join(lines, "\n")
}

func SuggestedCollectDocumentPath(task acpruntime.Task) string {
	if rootFileScopes := rootFileShardPathScopes(task.PathScopes); len(rootFileScopes) > 0 {
		return "root-overview.md"
	}
	if len(task.PathScopes) == 1 {
		if slug := slugComponent(task.PathScopes[0]); slug != "" {
			return slug + "-overview.md"
		}
	}
	if slug := slugComponent(task.ShardID); slug != "" {
		return slug + "-overview.md"
	}
	return "overview.md"
}

func CollectManifestCanonicalPath(task acpruntime.Task, docPath string) string {
	docPath = filepath.ToSlash(strings.TrimSpace(docPath))
	docSlug := slugComponent(strings.TrimSuffix(docPath, filepath.Ext(docPath)))
	if docSlug == "" {
		docSlug = "overview"
	}
	shardSlug := slugComponent(firstNonEmpty(task.ShardID, task.DomainID, strings.Join(task.PathScopes, "-"), "shard"))
	if shardSlug == "" {
		shardSlug = "shard"
	}
	return fmt.Sprintf("reports/as-is/%s/%s.md", shardSlug, docSlug)
}

func CollectFirstActionSection(task acpruntime.Task) string {
	if !acpruntime.IsCollectStep(task.StepID) {
		return ""
	}
	writeRoot := strings.TrimSpace(task.WriteRoot)
	docTarget := filepath.Join(writeRoot, filepath.FromSlash(SuggestedCollectDocumentPath(task)))
	manifestTarget := filepath.Join(writeRoot, "shard-pack-manifest.json")
	lines := []string{
		"COLLECT EVIDENCE-FIRST ARTIFACT PAIR:",
		"- This collect step must start by writing an evidence-backed artifact pair; do not write a seed-only pair.",
		fmt.Sprintf(`- Exact authored document target: %q.`, docTarget),
		fmt.Sprintf(`- Exact manifest target: %q.`, manifestTarget),
		"FIRST COLLECT BOUNDED WRITE ACTION:",
		"- The next filesystem work unit may contain only two mechanically simple commands: one bounded evidence read/list, then one direct literal write of both exact targets.",
		"- Read only existing repo entrypoint hints and assigned path_scopes needed for this shard; do not inspect reports/taskruns, raw logs, sibling shards, or archive docs.",
		"- Keep reads bounded: root-file shards read only listed root files first; directory shards inspect at most 8 representative entrypoint/build/config/source files.",
		"- Read at most the first 6000 bytes from any file. Truncate or skip oversized files while continuing with other available evidence.",
		"- Assigned path_scopes may be directories for discovery, but manifest citations and semantic provenance paths must resolve to concrete existing files, never directories.",
		"- Before writing shard-pack-manifest.json, prove every citation/provenance repo evidence path with file-level checks such as test -f, rg --files, or portable find ... -type f -print.",
		"- Every citations[].id must be unique; when citing multiple repo files from one authored document, derive each citation id from the shard/document stem plus the repo path slug, not from the authored markdown document id alone.",
		"- If a scoped path is missing or directory-only, record it as coverage.missing or a question instead of using it as citation/provenance evidence.",
		"- Syntax-only checks such as jq empty or python3 -m json.tool are insufficient; the final manifest check must parse JSON and verify semantic.questions[] all have id and text, citations[].id has no duplicates, every citation has non-empty claim_ids and document_ids, every documents[].citation_ids value exists, and every citation/provenance repo path is an existing file.",
		"- Use python3, not python; do not use GNU-only find -printf; do not assign to the zsh-reserved status variable.",
		"- Do not emit analysis-only prose, status/progress narration, todo/planning, broad repo sweeps, or any second read-only preflight before writing the artifact pair.",
		"- Before both targets exist, do not use Ruby, Node, Python, Perl, awk, jq, generated source-code strings, template programs, or nested quote tricks for markdown/JSON assembly.",
		"COLLECT FINAL WRITE REQUIREMENT:",
		"- Write the authored document and shard-pack-manifest.json with direct shell heredoc/printf/tee literal content from the bounded reads in the first work unit.",
		"- If a write command fails before both targets exist, immediately retry with a simpler direct literal write; do not wait for collect_pair_repair.",
		"- If both target files already exist, overwrite them unless they already contain marker-free, non-placeholder, evidence-backed content for this shard.",
		"- Do not exit after writing markdown only; shard-pack-manifest.json is required.",
		"- Final collect markdown must be operator-facing architecture evidence. Do not mention bounded reads/passes, guessed paths/files/evidence, expected-missing path checks, recovery attempts, or later repair as final content.",
		"- Successful final output must remain marker-free and must not collapse to generic owner-mapping-only findings or only repo/shard contains edges when concrete repository files support richer entities, relationships, coverage, or findings.",
		"- Normal collect must not depend on collect_pair_repair as the expected path to success.",
		"COLLECT MANIFEST TASK SKELETON:",
		strings.TrimSpace(CollectManifestTaskSkeleton(task, []string{SuggestedCollectDocumentPath(task)}, nil)),
		"SKELETON USE:",
		"- Use the JSON above as the task-specific schema/key/type guide, not as final content.",
		"- Replace skeleton citations, coverage, questions, entities, edges, findings, titles, and descriptions with facts observed in repository files.",
		"- Copying this skeleton unchanged is invalid and will be rejected as scaffold-only output.",
	}
	if scopeHints := CollectPathScopeFileHints(task); len(scopeHints) > 0 {
		lines = append(lines[:7], append([]string{
			"- Existing path-scope file candidates for this collect shard:",
			"  - " + strings.Join(scopeHints, "\n  - "),
		}, lines[7:]...)...)
	}
	return strings.Join(lines, "\n")
}

func ValidatorVerdictWriteCommand(task acpruntime.Task) string {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	target := filepath.Join(writeRoot, "validator-verdict.json")
	skeleton := ValidatorVerdictTaskSkeleton(task)
	return strings.Join([]string{
		"mkdir -p " + shellSingleQuote(writeRoot),
		"cat > " + shellSingleQuote(target) + " <<'ACP_VALIDATOR_VERDICT_JSON'",
		strings.TrimSpace(skeleton),
		"ACP_VALIDATOR_VERDICT_JSON",
	}, "\n")
}

func ValidatorFirstActionSection(task acpruntime.Task) string {
	if acpruntime.StepProviderKeyForStepID(task.StepID) != acpruntime.StepProviderStep3Findings {
		return ""
	}
	target := filepath.Join(strings.TrimSpace(task.WriteRoot), "validator-verdict.json")
	return strings.Join([]string{
		"VALIDATOR FIRST-ACTION ARTIFACT:",
		"- This validator step must start by writing validator-verdict.json before broad validation instructions.",
		fmt.Sprintf(`- Exact validator verdict target: %q.`, target),
		"FIRST VALIDATOR VERDICT COMMAND:",
		"Run this exact command as the next filesystem action after checking whether the target file already exists; do not inspect repository files, sibling taskruns, or raw logs before this command:",
		"After the skeleton exists, inspect staged final artifacts and replace skeleton findings/questions with evidence-backed validator findings/questions for remaining critical gaps or explicit no-issue rationale before exiting.",
		ValidatorVerdictWriteCommand(task),
	}, "\n")
}

func ConstitutionFirstActionSection(task acpruntime.Task) string {
	if strings.TrimSpace(task.StepID) != "init.step0.constitution" {
		return ""
	}
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), runtimedrafts.ConstitutionManifestFile)
	return strings.Join([]string{
		"CONSTITUTION FIRST-ACTION DRAFT ARTIFACTS:",
		"- This constitution step must start by writing constitution-draft.json and its referenced draft files before broad workspace analysis.",
		"- The heredoc charter overview is bootstrap-only; replace it with evidence-backed charter content before successful exit.",
		"- Same provider turn requirement: after this skeleton command succeeds, immediately run bounded repo-entrypoint enrichment and fresh-overwrite charter-overview.md before final success.",
		"- Final charter-overview.md must express unknowns as current constitution evidence gaps; do not write later collection steps, later analysis, future pipeline passes, downstream checks, collected shards, validator output, final indexes, citations, proposal artifacts, runtime repair, providers, or taskrun mechanics.",
		fmt.Sprintf(`- Exact constitution draft manifest target: %q.`, manifestTarget),
		fmt.Sprintf(`- Draft files must be written only under draft_final_root: %q.`, strings.TrimSpace(task.DraftFinalRoot)),
		"FIRST CONSTITUTION DRAFT COMMAND:",
		"Run this exact shell command as the next filesystem action; do not manually retype paths, rewrite slash-separated path components, inspect repository files, inspect sibling taskruns, or inspect raw logs before this command:",
		RuntimeDraftFirstActionWriteCommand(task),
	}, "\n")
}

func AsIsFirstActionSection(task acpruntime.Task) string {
	switch strings.TrimSpace(task.StepID) {
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
	default:
		return ""
	}
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), runtimedrafts.AsIsManifestFile)
	lines := []string{
		"AS-IS FIRST-ACTION DRAFT ARTIFACTS:",
		"- This as-is step must start with one bounded evidence-read/write filesystem work unit, not with a bootstrap-only draft.",
		"- The first filesystem work unit must read current-run staged evidence first, write overview.md, summary.md, and architect-summary.md under draft_final_root, then write asis-draft-manifest.json last before returning.",
		"- Do not create, write, touch, or leave asis-draft-manifest.json as a standalone first artifact; a manifest-only first write before markdown is invalid and may be classified as pre-artifact stall.",
		"- Same provider turn requirement: do not send any final answer, status note, or analysis-only prose before those validation-ready markdown files exist.",
		"- Your first response item for this step must be the filesystem command itself; do not send a preliminary assistant/status message before the command.",
		"- overview.md must contain concrete repo/path, citation, or staged artifact refs when evidence exists; summary.md must include exact shard counts when typed shard status is visible; architect-summary.md must include decision-ready operator cues.",
		"- If evidence is sparse, write explicit evidence-backed insufficiency tied to current-run coverage/questions instead of generic placeholder text.",
		fmt.Sprintf(`- Exact as-is draft manifest target: %q.`, manifestTarget),
		fmt.Sprintf(`- Draft files must be written only under draft_final_root: %q.`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- Exact overview target: %q.`, filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "overview.md")),
		fmt.Sprintf(`- Exact coverage summary target: %q.`, filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "summary.md")),
		fmt.Sprintf(`- Exact architect summary target: %q.`, filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "architect-summary.md")),
		"FIRST AS-IS DRAFT COMMAND:",
		"Run one filesystem command as the next action. In that command, perform only a bounded current-run evidence read/list, write all three markdown targets first, then write the manifest last before returning.",
		"Do not emit an assistant message, status sentence, or analysis-only response before this command; the first provider item must be command_execution.",
		"Do not run a separate read-only preflight, broad repo sweep, sibling taskrun inspection, prior-report templating, or analysis-only response before the writes.",
		"AS-IS FIRST-PASS WRITE SEQUENCE:",
		asIsFirstPassWriteSequence(task),
		"Use the manifest JSON below as the shape guide for the command output; copy keys/types exactly, but write operator-facing markdown from observed evidence instead of copying scaffold prose.",
		"AS-IS DRAFT MANIFEST SHAPE GUIDE:",
		strings.TrimSpace(RuntimeDraftManifestShapeGuide(task)),
	}
	lines = append(lines, currentRunEvidenceIndexLines(task, currentRunEvidenceAsIs)...)
	lines = append(lines,
		"AS-IS FIRST-PASS SELF-CHECK:",
		"- asis-draft-manifest.json exists under write_root and uses step_contract=\"as_is\".",
		"- overview.md, summary.md, and architect-summary.md exist under draft_final_root.",
		"- The first write set was not manifest-only: all three markdown targets were created before or in the same command as the final manifest write.",
		"- summary.md and architect-summary.md contain the exact planned=<n> succeeded=<n> failed=<n> incomplete=<n> literal when typed shard status is visible.",
		"- If typed shard status has failed=0 and incomplete=0, summary.md and architect-summary.md include an explicit no-shard-coverage-blocker statement that current-run shard coverage is not a blocker.",
		"- architect-summary.md says what is complete, what is missing, what the operator should inspect or decide next, and residual risk.",
		"- If final-run-index.json or citation-index.json is absent from the current-run staged evidence index above, omit downstream index availability entirely; do not say an index is unavailable, missing, not observed, pending, or will appear later.",
		"- No markdown target says it is a bootstrap, placeholder, draft surface initialized, recovery output, or content that will be replaced later.",
	)
	return strings.Join(lines, "\n")
}

func asIsFirstPassWriteSequence(task acpruntime.Task) string {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	draftRoot := strings.TrimSpace(task.DraftFinalRoot)
	return strings.Join([]string{
		"- Use this exact target setup at the top of that one command:",
		"  write_root=" + shellSingleQuote(writeRoot),
		"  draft_root=" + shellSingleQuote(draftRoot),
		`  mkdir -p "$write_root" "$draft_root"`,
		"- In the same shell process, first perform bounded evidence reads/lists from the current-run staged evidence index below; do not stop after this read phase.",
		"- Markdown writes must preserve literal backticks and paths: use single-quoted heredocs such as <<'EOF' or a python3 - <<'PY' program with Path.write_text literal strings.",
		"- Do not put markdown content that contains backticks inside double-quoted shell strings or unquoted heredocs; shell command substitution can erase refs and produce empty slots.",
		`- Then write evidence-backed markdown to "$draft_root/overview.md", "$draft_root/summary.md", and "$draft_root/architect-summary.md".`,
		`- Then write "$write_root/asis-draft-manifest.json" last using the manifest shape guide below and outputs for exactly those three markdown files.`,
		"- End the same command with these existence checks before exit:",
		`  test -s "$draft_root/overview.md"`,
		`  test -s "$draft_root/summary.md"`,
		`  test -s "$draft_root/architect-summary.md"`,
		`  test -s "$write_root/asis-draft-manifest.json"`,
		"- If any check would fail, fix the missing target in that same command before returning; do not rely on focused repair to create it later.",
		`- Also check the markdown has no empty evidence slots such as "from  and", "checked:  and", "under .", or "Use  and".`,
	}, "\n")
}

func RuntimeDraftFirstActionWriteCommand(task acpruntime.Task) string {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	draftRoot := strings.TrimSpace(task.DraftFinalRoot)
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	skeleton := RuntimeDraftManifestTaskSkeleton(task)
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
			runtimeDraftFirstActionFileTemplate(task, output),
			"ACP_DRAFT_FILE",
			"test -s \""+target+"\"",
		)
	}
	return strings.Join(lines, "\n")
}

func ProposalsFirstActionSection(task acpruntime.Task) string {
	if acpruntime.StepProviderKeyForStepID(task.StepID) != acpruntime.StepProviderStep4Proposals {
		return ""
	}
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), runtimedrafts.ProposalsManifestFile)
	lines := []string{
		"PROPOSALS FIRST-ACTION DRAFT ARTIFACTS:",
		"- This proposals step must start with one bounded evidence-read/write filesystem work unit, not with a bootstrap-only draft.",
		"- The first filesystem work unit must read current-run staged findings/coverage/index evidence first, write proposal.md and changelog.md under draft_final_root, then write proposals-draft-manifest.json last before returning.",
		"- Do not create, write, touch, or leave proposals-draft-manifest.json as a standalone first artifact; a manifest-only first write before proposal/changelog markdown is invalid and may be classified as pre-artifact stall.",
		"- Same provider turn requirement: do not send any final answer, status note, or analysis-only prose before those validation-ready markdown files exist.",
		"- In that first work unit, read current-run staged final findings and coverage under reports/taskruns/<run_id>/staging/final/reports/*, plus final-run-index.json, citation-index.json, and typed shard summary if visible; link non-empty findings to proposal/changelog content by finding ID.",
		"- When typed shard-summary items[] is visible, proposal.md and changelog.md must both contain the exact literal shard completeness shape planned=<n> succeeded=<n> failed=<n> incomplete=<n>; if failed=0 and incomplete=0, both files must also state an explicit no-shard-coverage-blocker.",
		"- The exact findings file is reports/taskruns/<run_id>/staging/final/reports/findings/findings.md, not reports/taskruns/<run_id>/staging/final/reports/findings.md. Copy IDs from backticked - ID: lines and never emit no-current-run-finding-id, no structured current-run finding ID, or finding unavailable.",
		"- proposal.md must contain non-empty sections named Decision / recommended operator action, Evidence used, Proposed changes or follow-up plan, and Risks, gaps, and out-of-scope notes.",
		"- changelog.md must contain non-empty sections named Updated architecture/proposal surfaces, Findings/proposals summary, Evidence index or citation references, and Residual coverage gaps.",
		"- For high/medium findings, include a bullet-only Top Actionable Findings section with one bullet per finding and all required fields on that same bullet line: exact Finding ID, copied Severity value from the finding block, Affected surface/path from Related IDs/Evidence, Recommended operator action, and Residual gap.",
		"- If current-run findings are non-empty, do not write Finding ID: none, Finding ID: n/a, Finding ID: unavailable, or similar placeholder IDs in any proposal/changelog bullet.",
		"- Do not split one finding across multiple bullets; do not write Severity: unspecified when findings.md has a - Severity: field; copy the exact high/medium/low value from that finding.",
		"- Do not use markdown tables for actionable findings.",
		fmt.Sprintf(`- Exact proposals draft manifest target: %q.`, manifestTarget),
		fmt.Sprintf(`- Draft files must be written only under draft_final_root: %q.`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- Exact proposal target: %q.`, filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "proposal.md")),
		fmt.Sprintf(`- Exact changelog target: %q.`, filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "changelog.md")),
		"FIRST PROPOSALS DRAFT COMMAND:",
		"Run one filesystem command as the next action. In that command, perform only a bounded current-run evidence read/list, write proposal.md and changelog.md first with all required sections, then write the manifest last before returning.",
		"Do not run a separate read-only preflight, broad repo sweep, sibling taskrun inspection, prior-proposal templating, or analysis-only response before the writes.",
		"Use the manifest JSON below as the shape guide for the command output; copy keys/types exactly, but write operator-facing proposal/changelog markdown from observed evidence instead of copying scaffold prose.",
		"PROPOSALS DRAFT MANIFEST SHAPE GUIDE:",
		strings.TrimSpace(RuntimeDraftManifestShapeGuide(task)),
	}
	lines = append(lines, currentRunEvidenceIndexLines(task, currentRunEvidenceProposals)...)
	lines = append(lines,
		"PROPOSALS FIRST-PASS SELF-CHECK:",
		"- proposals-draft-manifest.json exists under write_root and uses step_contract=\"proposals\".",
		"- proposal.md and changelog.md exist under draft_final_root.",
		"- The first write set was not manifest-only: proposal.md and changelog.md were created before or in the same command as the final manifest write.",
		"- proposal.md contains Decision / recommended operator action, Evidence used, Proposed changes or follow-up plan, and Risks, gaps, and out-of-scope notes.",
		"- changelog.md contains Updated architecture/proposal surfaces, Findings/proposals summary, Evidence index or citation references, and Residual coverage gaps.",
		"- If typed shard completeness is visible, proposal.md and changelog.md both include the exact planned=<n> succeeded=<n> failed=<n> incomplete=<n> literal and no-shard-coverage-blocker statement when there are no failed or incomplete shards.",
		"- If findings.md has any - ID: lines, both markdown targets cite at least one exact current-run finding ID.",
		"- Every medium/high finding represented in the first-pass proposal uses one same-line bullet containing its exact Finding ID, copied Severity, concrete Affected surface/path, concrete Recommended operator action verb, and Residual gap before the manifest is written.",
		"- If any Top Actionable Findings section is present while findings are non-empty, every Finding ID field uses an exact current-run finding ID, never none/n/a/unavailable.",
		"- No markdown target says structured findings are absent when findings.md is non-empty, and no target uses synthetic finding placeholders.",
		"- No markdown target says it is a bootstrap, placeholder, draft surface initialized, recovery output, or content that will be replaced later.",
	)
	return strings.Join(lines, "\n")
}

type currentRunEvidenceKind string

const (
	currentRunEvidenceAsIs      currentRunEvidenceKind = "as_is"
	currentRunEvidenceProposals currentRunEvidenceKind = "proposals"
)

func currentRunEvidenceIndexLines(task acpruntime.Task, kind currentRunEvidenceKind) []string {
	runID := strings.TrimSpace(task.RunID)
	if runID == "" {
		runID = "run-1"
	}
	evidence := currentRunEvidenceIndex(task, kind)
	lines := []string{
		"- Current-run staged evidence index: read these exact public artifact files when present inside the first evidence-first write command before writing final markdown.",
	}
	for _, item := range evidence {
		if item.Path == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf(`  - %s: %q`, item.Label, item.Path))
	}
	if completenessLine := currentRunShardCompletenessPromptLine(task); completenessLine != "" {
		lines = append(lines, completenessLine)
	}
	if kind == currentRunEvidenceAsIs {
		lines = append(lines,
			`- As-is enrichment must use the typed shard plan/summary, shard-pack-manifest summaries, final-run-index.json, and citation-index.json when visible.`,
			`- summary.md and architect-summary.md must state exact shard completeness counts in this literal shape when evidence is visible: planned=<n> succeeded=<n> failed=<n> incomplete=<n>.`,
			`- If typed shard completeness shows failed=0 and incomplete=0, summary.md and architect-summary.md must include an explicit no-shard-coverage-blocker statement that says current-run shard coverage is not a blocker and must not include generic failed/incomplete caveats.`,
			`- overview.md and architect-summary.md must cite concrete repo/path or staged citation/index references from the current run, not generic scaffold language.`,
		)
	} else {
		lines = append(lines,
			fmt.Sprintf(`- Exact current-run findings source: reports/taskruns/%s/staging/final/reports/findings/findings.md. Read that file when present and copy at least one exact finding ID into proposal.md and changelog.md when findings are non-empty.`, runID),
			`- proposal.md and changelog.md must both state exact shard completeness in this literal shape when typed shard evidence is visible: planned=<n> succeeded=<n> failed=<n> incomplete=<n>. If failed=0 and incomplete=0, both files must state an explicit no-shard-coverage-blocker.`,
			`- Do not use synthetic finding placeholders; if findings.md has - ID: lines, proposal/changelog linkage must use one of those exact IDs.`,
			`- High/medium findings require one bullet per finding, with Finding ID, Severity, Affected surface/path, Recommended operator action, and Residual gap all on the same bullet line.`,
		)
		if findingPreviewLine := currentRunFindingsPreviewPromptLine(task); findingPreviewLine != "" {
			lines = append(lines, findingPreviewLine)
		}
	}
	return lines
}

type currentRunEvidenceItem struct {
	Label string
	Path  string
}

func currentRunEvidenceIndex(task acpruntime.Task, kind currentRunEvidenceKind) []currentRunEvidenceItem {
	runID := strings.TrimSpace(task.RunID)
	if runID == "" {
		runID = "run-1"
	}
	items := []currentRunEvidenceItem{}
	add := func(label, path string) {
		path = filepath.ToSlash(strings.TrimSpace(path))
		if path == "" {
			return
		}
		for _, existing := range items {
			if existing.Label == label && existing.Path == path {
				return
			}
		}
		items = append(items, currentRunEvidenceItem{Label: label, Path: path})
	}
	for _, rel := range []struct {
		label string
		path  string
	}{
		{"final-run-index", "final-run-index.json"},
		{"citation-index", "citation-index.json"},
		{"coverage-summary", "reports/coverage/summary.md"},
	} {
		if path := firstExistingStagedFinalPath(task, rel.path); path != "" {
			add(rel.label, path)
		}
	}
	if kind == currentRunEvidenceProposals {
		if path := firstExistingStagedFinalPath(task, "reports/findings/findings.md"); path != "" {
			add("findings", path)
		}
	} else {
		if path := firstExistingStagedFinalPath(task, "reports/findings/findings.md"); path != "" {
			add("findings", path)
		}
	}
	for _, path := range existingTaskrunGlob(task, []string{
		runID + "*shard-summary*.json",
		runID + "*shard-plan*.json",
		runID + "*typed*summary*.json",
		runID + "*typed*plan*.json",
	}) {
		add("typed-shard-evidence", path)
	}
	for _, path := range existingShardManifestPaths(task, 6) {
		add("shard-pack-manifest", path)
	}
	if len(items) == 0 {
		add("expected-staged-final-root", fmt.Sprintf("reports/taskruns/%s/staging/final", runID))
		add("expected-typed-shard-summary", fmt.Sprintf("reports/taskruns/%s-*-shard-summary-*.json", runID))
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Label == items[j].Label {
			return items[i].Path < items[j].Path
		}
		return items[i].Label < items[j].Label
	})
	return items
}

func currentRunShardCompletenessPromptLine(task acpruntime.Task) string {
	path, counts, ok := currentRunShardCompleteness(task)
	if !ok {
		return ""
	}
	return fmt.Sprintf(
		`- Current-run typed shard completeness observed from %q: planned=%d succeeded=%d failed=%d incomplete=%d. Copy this exact literal into summary/proposal text when shard status is mentioned.`,
		filepath.ToSlash(path),
		counts.Planned,
		counts.Succeeded,
		counts.Failed,
		counts.Incomplete,
	)
}

type currentRunShardCompletenessCounts struct {
	Planned    int
	Succeeded  int
	Failed     int
	Incomplete int
}

func currentRunShardCompleteness(task acpruntime.Task) (string, currentRunShardCompletenessCounts, bool) {
	runID := strings.TrimSpace(task.RunID)
	if runID == "" {
		runID = "run-1"
	}
	for _, candidate := range existingTaskrunGlob(task, []string{
		runID + "*shard-summary*.json",
		runID + "*typed*summary*.json",
	}) {
		raw, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		var summary struct {
			Items []struct {
				Status string `json:"status"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &summary); err != nil || len(summary.Items) == 0 {
			continue
		}
		counts := currentRunShardCompletenessCounts{Planned: len(summary.Items)}
		for _, item := range summary.Items {
			switch strings.ToLower(strings.TrimSpace(item.Status)) {
			case "succeeded", "success", "passed":
				counts.Succeeded++
			case "failed", "error":
				counts.Failed++
			default:
				counts.Incomplete++
			}
		}
		return candidate, counts, true
	}
	return "", currentRunShardCompletenessCounts{}, false
}

func currentRunFindingsPreviewPromptLine(task acpruntime.Task) string {
	path := firstExistingStagedFinalPath(task, "reports/findings/findings.md")
	if path == "" {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	findings := summarizePromptMarkdownFindings(string(raw), 5)
	if len(findings) == 0 {
		return ""
	}
	return fmt.Sprintf(
		`- Visible current-run finding IDs from %q include %s. Proposal/changelog drafts must copy exact IDs; high/medium IDs require Top Actionable Findings bullets.`,
		filepath.ToSlash(path),
		strings.Join(findings, ", "),
	)
}

func summarizePromptMarkdownFindings(text string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	type finding struct {
		id       string
		severity string
	}
	findings := []finding{}
	currentID := ""
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, "- id:"):
			currentID = promptMarkdownFieldValue(trimmed[len("- id:"):])
			if currentID != "" {
				findings = append(findings, finding{id: currentID})
			}
		case strings.HasPrefix(lower, "- severity:") && currentID != "":
			severity := strings.ToLower(promptMarkdownFieldValue(trimmed[len("- severity:"):]))
			for idx := range findings {
				if findings[idx].id == currentID {
					findings[idx].severity = severity
					break
				}
			}
		}
	}
	preferred := make([]finding, 0, len(findings))
	fallback := make([]finding, 0, len(findings))
	seen := map[string]struct{}{}
	for _, finding := range findings {
		if finding.id == "" {
			continue
		}
		if _, ok := seen[finding.id]; ok {
			continue
		}
		seen[finding.id] = struct{}{}
		if finding.severity == "high" || finding.severity == "medium" {
			preferred = append(preferred, finding)
		} else {
			fallback = append(fallback, finding)
		}
	}
	if len(preferred) == 0 {
		preferred = fallback
	}
	out := []string{}
	for _, finding := range preferred {
		value := finding.id
		if finding.severity != "" {
			value += " severity=" + finding.severity
		}
		out = append(out, value)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func promptMarkdownFieldValue(value string) string {
	value = strings.TrimSpace(value)
	if start := strings.Index(value, "`"); start >= 0 {
		if end := strings.Index(value[start+1:], "`"); end >= 0 {
			return strings.TrimSpace(value[start+1 : start+1+end])
		}
	}
	value = strings.Trim(value, "` \t:;,")
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], "` \t:;,.")
}

func firstExistingStagedFinalPath(task acpruntime.Task, rel string) string {
	for _, candidate := range stagedFinalPathCandidates(task, rel) {
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func stagedFinalPathCandidates(task acpruntime.Task, rel string) []string {
	runID := strings.TrimSpace(task.RunID)
	if runID == "" {
		runID = "run-1"
	}
	rel = filepath.FromSlash(strings.Trim(strings.TrimSpace(rel), "/"))
	candidates := []string{}
	add := func(path string) {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "" || path == "." {
			return
		}
		for _, existing := range candidates {
			if existing == path {
				return
			}
		}
		candidates = append(candidates, path)
	}
	for _, root := range task.ReadContextRoots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "" || root == "." {
			continue
		}
		slash := filepath.ToSlash(root)
		if strings.HasSuffix(slash, "/staging/final") {
			add(filepath.Join(root, rel))
		}
		add(filepath.Join(root, "reports", "taskruns", runID, "staging", "final", rel))
		add(filepath.Join(root, "staging", "final", rel))
		add(filepath.Join(root, rel))
	}
	if workspace := strings.TrimSpace(task.Workspace); workspace != "" {
		add(filepath.Join(workspace, "reports", "taskruns", runID, "staging", "final", rel))
	}
	return candidates
}

func existingTaskrunGlob(task acpruntime.Task, patterns []string) []string {
	roots := taskrunRoots(task)
	matches := []string{}
	seen := map[string]struct{}{}
	for _, root := range roots {
		for _, pattern := range patterns {
			for _, path := range globExisting(filepath.Join(root, pattern)) {
				if _, ok := seen[path]; ok {
					continue
				}
				seen[path] = struct{}{}
				matches = append(matches, path)
			}
		}
	}
	sort.Strings(matches)
	if len(matches) > 8 {
		return matches[:8]
	}
	return matches
}

func existingShardManifestPaths(task acpruntime.Task, limit int) []string {
	runID := strings.TrimSpace(task.RunID)
	if runID == "" {
		runID = "run-1"
	}
	roots := []string{}
	for _, root := range task.ReadContextRoots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "" || root == "." {
			continue
		}
		slash := filepath.ToSlash(root)
		if strings.HasSuffix(slash, "/staging/final") {
			roots = append(roots, filepath.Join(filepath.Dir(root), "shards"))
		}
		roots = append(roots,
			filepath.Join(root, "reports", "taskruns", runID, "staging", "shards"),
			filepath.Join(root, "staging", "shards"),
		)
	}
	if workspace := strings.TrimSpace(task.Workspace); workspace != "" {
		roots = append(roots, filepath.Join(workspace, "reports", "taskruns", runID, "staging", "shards"))
	}
	seen := map[string]struct{}{}
	paths := []string{}
	for _, root := range roots {
		if !dirExists(root) {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d == nil || d.IsDir() || d.Name() != "shard-pack-manifest.json" {
				return nil
			}
			path = filepath.ToSlash(path)
			if _, ok := seen[path]; ok {
				return nil
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
			if limit > 0 && len(paths) >= limit {
				return filepath.SkipAll
			}
			return nil
		})
		if limit > 0 && len(paths) >= limit {
			break
		}
	}
	sort.Strings(paths)
	return paths
}

func taskrunRoots(task acpruntime.Task) []string {
	runID := strings.TrimSpace(task.RunID)
	if runID == "" {
		runID = "run-1"
	}
	roots := []string{}
	add := func(path string) {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "" || path == "." {
			return
		}
		for _, existing := range roots {
			if existing == path {
				return
			}
		}
		roots = append(roots, path)
	}
	for _, root := range task.ReadContextRoots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "" || root == "." {
			continue
		}
		slash := filepath.ToSlash(root)
		marker := "/reports/taskruns/" + runID + "/"
		if idx := strings.Index(slash, marker); idx >= 0 {
			add(root[:idx+len("/reports/taskruns")])
		}
		if strings.HasSuffix(slash, "/reports/taskruns") {
			add(root)
		}
		add(filepath.Join(root, "reports", "taskruns"))
	}
	if workspace := strings.TrimSpace(task.Workspace); workspace != "" {
		add(filepath.Join(workspace, "reports", "taskruns"))
	}
	return roots
}

func globExisting(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, path := range matches {
		if fileExists(path) {
			out = append(out, filepath.ToSlash(path))
		}
	}
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func safeDraftOutputPath(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if filepath.IsAbs(trimmed) || strings.HasPrefix(filepath.ToSlash(trimmed), "/") {
		return "", false
	}
	rel := filepath.ToSlash(trimmed)
	rel = strings.TrimPrefix(rel, "./")
	rel = strings.Trim(rel, "/")
	if rel == "" || rel == "." || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
		return "", false
	}
	return rel, true
}

func runtimeDraftFirstActionFileTemplate(task acpruntime.Task, output runtimedrafts.Output) string {
	canonicalPath := strings.TrimSpace(output.CanonicalPath)
	title := strings.TrimSpace(output.Title)
	if title == "" {
		title = strings.TrimSpace(output.Path)
	}
	switch canonicalPath {
	case "skills/subagents.yaml":
		return strings.TrimSpace(string(workspace.BaselineSubagentsContent()))
	case "charter/overview.md":
		scopeLines := runtimeDraftScopeLines(task)
		return strings.Join([]string{
			"# Constitution",
			"",
			"## Scope",
			strings.Join(scopeLines, "\n"),
			"",
			"## Charter Draft",
			"- Draft constitution surface initialized for the configured repository scope.",
			"- Replace this bootstrap with target identity, repo/path evidence, decision boundaries, and local-first operating rules before successful exit.",
		}, "\n")
	case "proposals/runtime-recommendations.md":
		return strings.Join([]string{
			"# Runtime Recommendations",
			"",
			"## Summary",
			"- Current run evidence should be reviewed before promotion.",
			"- Owner mappings and unresolved coverage gaps remain the first follow-up surfaces.",
			"",
			"## Recommendation",
			"- Promote only recommendations that cite collected shard manifests, validator findings, or final coverage output.",
		}, "\n")
	case "reports/changelog/runtime-proposals.md":
		return strings.Join([]string{
			"# Runtime Proposal Changelog",
			"",
			"## Changes",
			"- Runtime proposal surface initialized for this analysis run.",
			"- Changes must remain traceable to collected evidence, findings, or coverage gaps before promotion.",
			"",
			"## Notes",
			"- Promote only after artifact validation succeeds.",
		}, "\n")
	default:
		scopeLines := runtimeDraftScopeLines(task)
		return strings.Join([]string{
			"# " + title,
			"",
			"## Scope",
			strings.Join(scopeLines, "\n"),
			"",
			"## Summary",
			"- Draft surface initialized for the scoped repository analysis.",
			"- Final content must stay tied to collected shard evidence and validator output.",
		}, "\n")
	}
}

func runtimeDraftScopeLines(task acpruntime.Task) []string {
	repo := PrimaryTaskRepoScope(task.RepoScope, task.RepoScopes)
	if repo == "" {
		repo = "repo"
	}
	pathScopes := strings.Join(nonEmptyList(task.PathScopes), ", ")
	if pathScopes == "" {
		pathScopes = collectEvidencePath(task, nil)
	}
	if pathScopes == "" {
		pathScopes = "README.md"
	}
	return []string{
		"- Run: " + strings.TrimSpace(task.RunID),
		"- Step: " + strings.TrimSpace(task.StepID),
		"- Repository scope: " + repo,
		"- Path scopes: " + pathScopes,
	}
}

func CollectManifestTaskSkeleton(task acpruntime.Task, docPaths []string, evidencePaths []string) string {
	return collectManifestTaskSkeleton(task, docPaths, evidencePaths)
}

func collectManifestTaskSkeleton(task acpruntime.Task, docPaths []string, evidencePaths []string) string {
	cleanDocPaths := make([]string, 0, len(docPaths))
	seenDocs := map[string]struct{}{}
	for _, docPath := range docPaths {
		docPath = filepath.ToSlash(strings.TrimSpace(docPath))
		if docPath == "" || docPath == "runtime-execution.json" || docPath == "shard-pack-manifest.json" {
			continue
		}
		if _, exists := seenDocs[docPath]; exists {
			continue
		}
		seenDocs[docPath] = struct{}{}
		cleanDocPaths = append(cleanDocPaths, docPath)
	}
	if len(cleanDocPaths) == 0 {
		cleanDocPaths = []string{SuggestedCollectDocumentPath(task)}
	}

	repo := PrimaryTaskRepoScope(task.RepoScope, task.RepoScopes)
	if strings.TrimSpace(repo) == "" {
		repo = "repo"
	}
	evidencePath := collectEvidencePath(task, evidencePaths)
	if evidencePath == "" {
		evidencePath = "README.md"
	}

	shardSlug := slugComponent(firstNonEmpty(task.ShardID, task.DomainID, strings.Join(task.PathScopes, "-"), "shard"))
	idStem := idComponent(shardSlug)
	topic := firstNonEmpty(slugComponent(task.DomainID), shardSlug)
	if topic == "" {
		topic = "architecture"
	}
	runID := firstNonEmpty(task.RunID, "run-1")
	stepID := firstNonEmpty(task.StepID, "init.step1.collect")
	shardID := firstNonEmpty(task.ShardID, shardSlug)
	artifactRoot := firstNonEmpty(task.ArtifactRoot, fmt.Sprintf("reports/taskruns/%s/staging/shards/%s", runID, shardSlug))

	documents := make([]contracts.AuthoredDocument, 0, len(cleanDocPaths))
	citations := make([]contracts.DocumentCitation, 0, len(cleanDocPaths))
	for idx, docPath := range cleanDocPaths {
		docSlug := slugComponent(strings.TrimSuffix(docPath, filepath.Ext(docPath)))
		if docSlug == "" {
			docSlug = fmt.Sprintf("doc-%d", idx+1)
		}
		docID := fmt.Sprintf("doc.%s.%s", idStem, idComponent(docSlug))
		citationID := fmt.Sprintf("cite.%s.%s", idStem, idComponent(docSlug))
		documents = append(documents, contracts.AuthoredDocument{
			ID:            docID,
			Kind:          "report",
			Title:         titleFromSlug(docSlug),
			Path:          docPath,
			CanonicalPath: CollectManifestCanonicalPath(task, docPath),
			Topics:        []string{topic},
			CitationIDs:   []string{citationID},
		})
		citations = append(citations, contracts.DocumentCitation{
			ID:          citationID,
			Repo:        repo,
			Path:        evidencePath,
			ClaimIDs:    []string{fmt.Sprintf("claim.%s.%s", idStem, idComponent(docSlug))},
			DocumentIDs: []string{docID},
		})
	}

	coverageMissing := collectCoverageMissingSkeleton(task)
	semantic := collectSemanticSkeleton(task, repo, evidencePath, shardSlug, idStem, topic, coverageMissing)
	manifest := contracts.ShardPackManifest{
		Version:      1,
		RunID:        runID,
		StepID:       stepID,
		ShardID:      shardID,
		DomainID:     strings.TrimSpace(task.DomainID),
		AgentRole:    firstNonEmpty(task.AgentRole, "shard-analyst"),
		ArtifactRoot: artifactRoot,
		RepoScopes:   nonEmptyList(task.RepoScopes),
		PathScopes:   nonEmptyList(task.PathScopes),
		Documents:    documents,
		Citations:    citations,
		Semantic:     semantic,
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func shellSingleQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func ValidatorVerdictTaskSkeleton(task acpruntime.Task) string {
	runID := firstNonEmpty(task.RunID, "run-1")
	checkedPaths := validatorCheckedPathSkeleton(task)
	findings, questions := validatorCrossRepoSignalSkeleton(task)
	type validatorVerdictSkeleton struct {
		Version      int                        `json:"version"`
		RunID        string                     `json:"run_id"`
		GeneratedAt  string                     `json:"generated_at"`
		Verdict      string                     `json:"verdict"`
		Summary      string                     `json:"summary"`
		CheckedPaths []string                   `json:"checked_paths"`
		FixedPaths   []string                   `json:"fixed_paths"`
		Findings     []contracts.Finding        `json:"findings"`
		Questions    []contracts.Question       `json:"questions"`
		Issues       []contracts.ValidatorIssue `json:"issues"`
	}
	verdict := validatorVerdictSkeleton{
		Version:      1,
		RunID:        runID,
		GeneratedAt:  "2026-04-16T12:00:02Z",
		Verdict:      "PASS",
		Summary:      "No blocking technical validator issues remain after inspecting the staged final artifacts.",
		CheckedPaths: checkedPaths,
		FixedPaths:   []string{},
		Findings:     findings,
		Questions:    questions,
		Issues:       []contracts.ValidatorIssue{},
	}
	if len(findings) > 0 || len(questions) > 0 {
		verdict.Summary = "No blocking technical validator issues remain after inspecting the staged final artifacts. Multi-repo semantic follow-up is recorded in findings/questions until concrete cross-repo edges are confirmed."
	}
	raw, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		return `{"version":1,"run_id":"run-1","generated_at":"2026-04-16T12:00:02Z","verdict":"PASS","checked_paths":["reports/taskruns/run-1/staging/final/final-run-index.json"],"fixed_paths":[],"findings":[],"questions":[],"issues":[]}`
	}
	return string(raw)
}

func validatorCrossRepoSignalSkeleton(task acpruntime.Task) ([]contracts.Finding, []contracts.Question) {
	repos := uniqueRepoScopes(task)
	if len(repos) < 2 {
		return []contracts.Finding{}, []contracts.Question{}
	}
	evidence := make([]contracts.Evidence, 0, len(repos))
	for _, repo := range repos {
		evidence = append(evidence, contracts.Evidence{
			Repo: repo,
			Path: validatorEvidencePathForRepo(task, repo),
		})
	}
	description := fmt.Sprintf(
		"Validator observed multiple repository scopes (%s) and records the required cross-repo semantic follow-up until a concrete integration edge or ownership relationship is promoted from staged evidence.",
		strings.Join(repos, ", "),
	)
	questionText := fmt.Sprintf(
		"Which ownership or integration contract connects %s and %s, and should it be promoted as an explicit semantic edge?",
		repos[0],
		repos[1],
	)
	return []contracts.Finding{
			{
				ID:          "finding.cross_repo.semantic_signal.required",
				Severity:    "medium",
				Title:       "Cross-repo semantic relationship needs explicit validation",
				Description: description,
				RuleID:      "analysis.cross_repo_semantic_signal",
				RelatedIDs:  repos,
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.7,
					Evidence:   evidence,
				},
			},
		}, []contracts.Question{
			{
				ID:         "q.cross_repo.integration_contract",
				Text:       questionText,
				Priority:   "medium",
				RelatedIDs: repos,
			},
		}
}

func uniqueRepoScopes(task acpruntime.Task) []string {
	values := append([]string{}, task.RepoScopes...)
	if strings.TrimSpace(task.RepoScope) != "" {
		values = append([]string{task.RepoScope}, values...)
	}
	result := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := repoScopeMatchKey(trimmed)
		if key == "" {
			key = strings.ToLower(trimmed)
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func validatorEvidencePathForRepo(task acpruntime.Task, repo string) string {
	if path := repoEntrypointEvidencePath(task, repo); path != "" {
		return path
	}
	if path := collectEvidencePath(task, nil); path != "" {
		return path
	}
	return "README.md"
}

func repoEntrypointEvidencePath(task acpruntime.Task, repo string) string {
	repoKey := repoScopeMatchKey(repo)
	if repoKey == "" {
		return ""
	}
	patterns := []string{
		"README.md",
		"README.adoc",
		"README.rst",
		"README",
		"Makefile",
		"catalog-info.yaml",
		"pyproject.toml",
		"package.json",
	}
	for _, root := range task.ReadContextRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		candidates := []string{}
		cleaned := filepath.Clean(root)
		if repoScopeMatchKey(filepath.Base(cleaned)) == repoKey {
			candidates = append(candidates, cleaned)
		}
		for _, child := range []string{repo, filepath.Base(repo)} {
			if strings.TrimSpace(child) == "" {
				continue
			}
			candidates = append(candidates, filepath.Join(cleaned, child))
		}
		for _, candidate := range candidates {
			info, err := os.Stat(candidate)
			if err != nil || !info.IsDir() {
				continue
			}
			for _, pattern := range patterns {
				matches, err := filepath.Glob(filepath.Join(candidate, pattern))
				if err != nil || len(matches) == 0 {
					continue
				}
				sort.Strings(matches)
				return filepath.ToSlash(strings.TrimPrefix(matches[0], strings.TrimRight(candidate, string(filepath.Separator))+string(filepath.Separator)))
			}
		}
	}
	return ""
}

func repoScopeMatchKey(value string) string {
	value = strings.ToLower(filepath.Base(filepath.ToSlash(strings.TrimSpace(value))))
	replacer := strings.NewReplacer("-", "", "_", "", ".", "", " ", "")
	return replacer.Replace(value)
}

func validatorCheckedPathSkeleton(task acpruntime.Task) []string {
	runID := firstNonEmpty(task.RunID, "run-1")
	if root := validatorStagedFinalRoot(task); root != "" {
		return []string{
			filepath.ToSlash(filepath.Join(root, "final-run-index.json")),
			filepath.ToSlash(filepath.Join(root, "citation-index.json")),
		}
	}
	return []string{
		fmt.Sprintf("reports/taskruns/%s/staging/final/final-run-index.json", runID),
		fmt.Sprintf("reports/taskruns/%s/staging/final/citation-index.json", runID),
	}
}

func validatorStagedFinalRoot(task acpruntime.Task) string {
	writeRoot := filepath.Clean(strings.TrimSpace(task.WriteRoot))
	for _, root := range task.ReadContextRoots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		cleaned := filepath.Clean(trimmed)
		if writeRoot != "." && cleaned == writeRoot {
			continue
		}
		slash := filepath.ToSlash(cleaned)
		if strings.HasSuffix(slash, "/staging/final") || strings.Contains(slash, "/staging/final/") {
			return cleaned
		}
	}
	return ""
}

func RuntimeDraftManifestTaskSkeleton(task acpruntime.Task) string {
	manifest := runtimedrafts.Manifest{
		Version:      1,
		RunID:        firstNonEmpty(task.RunID, "run-1"),
		StepID:       firstNonEmpty(task.StepID, "init.step2.asis_docs"),
		StepContract: firstNonEmpty(task.StepContract, runtimedrafts.StepContractForStep(task.StepID)),
		AgentRole:    firstNonEmpty(task.AgentRole, "architect"),
		Summary:      "Drafted required runtime artifacts for this step.",
		Outputs:      runtimeDraftOutputSkeleton(task),
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return `{"version":1,"run_id":"run-1","step_id":"init.step2.asis_docs","step_contract":"as_is","agent_role":"architect","outputs":[{"path":"overview.md","canonical_path":"reports/as-is/overview.md","kind":"report","title":"System Overview"}]}`
	}
	return string(raw)
}

func RuntimeDraftManifestShapeGuide(task acpruntime.Task) string {
	raw := RuntimeDraftManifestTaskSkeleton(task)
	var manifest runtimedrafts.Manifest
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		return raw
	}
	manifest.Summary = "Evidence-backed manifest for provider-authored runtime artifacts."
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return raw
	}
	return string(out)
}

func runtimeDraftOutputSkeleton(task acpruntime.Task) []runtimedrafts.Output {
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		return []runtimedrafts.Output{
			{Path: "charter-overview.md", CanonicalPath: "charter/overview.md", Kind: "charter", Title: "Constitution"},
			{Path: "baseline-subagents.yaml", CanonicalPath: "skills/subagents.yaml", Kind: "bundle", Title: "Baseline Subagents"},
		}
	case "init.step4.proposals", "refresh.step4.proposals":
		return []runtimedrafts.Output{
			{Path: "proposal.md", CanonicalPath: "proposals/runtime-recommendations.md", Kind: "proposal", Title: "Runtime Recommendations"},
			{Path: "changelog.md", CanonicalPath: "reports/changelog/runtime-proposals.md", Kind: "changelog", Title: "Runtime Proposal Changelog"},
		}
	default:
		return []runtimedrafts.Output{
			{Path: "overview.md", CanonicalPath: "reports/as-is/overview.md", Kind: "report", Title: "System Overview"},
			{Path: "summary.md", CanonicalPath: "reports/coverage/summary.md", Kind: "report", Title: "Coverage Summary"},
			{Path: "architect-summary.md", CanonicalPath: "reports/agent-outputs/architect/summary.md", Kind: "agent-output", Title: "Architect Summary"},
		}
	}
}

func rootFileShardPathScopes(pathScopes []string) []string {
	if len(pathScopes) == 0 {
		return nil
	}
	values := make([]string, 0, len(pathScopes))
	for _, raw := range pathScopes {
		value := filepath.ToSlash(strings.TrimSpace(raw))
		value = strings.TrimPrefix(value, "./")
		value = strings.Trim(value, "/")
		if value == "" || value == "." || strings.Contains(value, "/") {
			return nil
		}
		if !isLikelyRootFileName(value) {
			return nil
		}
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func jsonStringList(values []string) string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	if len(cleaned) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(cleaned)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func isLikelyRootFileName(value string) bool {
	name := strings.TrimSpace(value)
	if name == "" {
		return false
	}
	if isLikelyRootDirectoryName(name) {
		return false
	}
	if strings.Contains(name, ".") {
		return true
	}
	switch strings.ToLower(name) {
	case "authors", "changelog", "copying", "contributors", "dockerfile", "gradlew", "justfile", "license", "makefile", "mvnw", "notice", "procfile", "rakefile", "readme":
		return true
	default:
		return false
	}
}

func isLikelyRootDirectoryName(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ".circleci", ".github", ".gitlab", ".idea", ".vscode":
		return true
	default:
		return false
	}
}

func ConstitutionDraftManifestExample(task acpruntime.Task) string {
	manifest := runtimedrafts.Manifest{
		Version:      1,
		RunID:        task.RunID,
		StepID:       "init.step0.constitution",
		StepContract: "constitution",
		AgentRole:    nonEmptyOr(strings.TrimSpace(task.AgentRole), "architect"),
		Summary:      "Drafted constitution materials for compile/publish.",
		Outputs: []runtimedrafts.Output{
			{
				Path:          "charter-overview.md",
				CanonicalPath: "charter/overview.md",
				Kind:          "charter",
				Title:         "Constitution",
			},
			{
				Path:          "baseline-subagents.yaml",
				CanonicalPath: "skills/subagents.yaml",
				Kind:          "bundle",
				Title:         "Baseline Subagents",
			},
		},
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return `{"version":1,"run_id":"run-1","step_id":"init.step0.constitution","step_contract":"constitution","agent_role":"architect","outputs":[{"path":"charter-overview.md","canonical_path":"charter/overview.md","kind":"charter","title":"Constitution"},{"path":"baseline-subagents.yaml","canonical_path":"skills/subagents.yaml","kind":"bundle","title":"Baseline Subagents"}]}`
	}
	return string(raw)
}

func ParseRepairHints(stepID string, parseStage string, parseErr error) []string {
	if parseErr == nil {
		return nil
	}
	lines := []string{}
	if detail := compactRetryHint(parseErr.Error()); detail != "" {
		stage := strings.TrimSpace(parseStage)
		if stage == "" {
			stage = "unknown"
		}
		lines = append(lines, fmt.Sprintf(`- Previous %s validation failure: %s`, stage, detail))
	}
	return append(lines,
		`- Do NOT return semantic payloads on stdout; write the required artifacts and exit.`,
		`- Do NOT use stdout as a transport for JSON wrappers, tool transcripts, or operation logs.`,
		`- Keep all semantic state inside the required step artifacts under write_root or draft_final_root.`,
	)
}

func CollectArtifactRepairHints(initialProblem string) []string {
	lines := []string{
		`- Rebuild shard-pack-manifest.json to the canonical ACP schema before exiting successfully.`,
		`- In shard-pack-manifest.json, semantic.coverage/questions/entities/edges/findings are all required; questions/entities/edges/findings must be arrays even when empty.`,
		`- semantic.questions[*] must include id and text; do not emit question-only aliases.`,
		`- documents[].path MUST stay relative to artifact_root only; valid example: "component-overview.md". Invalid examples: "reports/taskruns/run-1/staging/shards/example-component/component-overview.md", "charter/overview.md".`,
		`- Do NOT emit top-level semantic payloads on stdout; keep semantic only inside shard-pack-manifest.json.`,
		`- semantic.entities[*] MUST remain full entity objects with provenance included; do not drop entities[*].provenance during repair.`,
		`- semantic.edges[*] MUST remain objects with canonical keys type/from/to; do not use kind/source/target aliases.`,
		`- semantic.findings[*] MUST remain objects and each finding MUST include id, severity, title, and provenance; never replace findings with plain strings or bullet text.`,
		`- semantic.questions/entities/edges/findings must stay object-only arrays; booleans, nulls, and string-valued findings are invalid.`,
		`- Do NOT leave claim_ids empty for cited repository evidence; preserve concrete repo-backed claim ids whenever the evidence supports them.`,
		`- Repair mode is artifact-only: do not invent extra repository file reads/writes after authored docs already exist.`,
		`- Valid semantic examples: entities[*].provenance={"kind":"observation","confidence":0.7,"evidence":[...]}, edges[*]={"id":"edge.dep","type":"depends_on","from":"svc.a","to":"svc.b","provenance":{...}}, findings[*]={"id":"finding.x","severity":"medium","title":"Missing owner mapping","description":"...","rule_id":"rule.owner.required","related_ids":["svc.a"],"provenance":{...}}.`,
	}
	lines = append(lines, artifactquality.CollectManifestLegacyHygieneLines()...)
	if detail := compactRetryHint(initialProblem); detail != "" {
		lines = append(lines, fmt.Sprintf(`- Previous artifact contract failure: %s`, detail))
	}
	return lines
}

func WorkspacePromptPackPath(stepID string) string {
	switch strings.TrimSpace(stepID) {
	case "init.step0.constitution":
		return "skills/prompt-packs/constitution.md"
	case "init.step1.collect", "refresh.step1.collect":
		return "skills/prompt-packs/collect-context.md"
	case "init.step3.findings", "refresh.step3.findings":
		return "skills/prompt-packs/findings.md"
	case "init.step4.proposals", "refresh.step4.proposals":
		return "skills/prompt-packs/proposals.md"
	case acpruntime.StepIDQAAsk:
		return "skills/prompt-packs/qa.md"
	default:
		return ""
	}
}

func QAAnswerCanonicalExample(task acpruntime.Task) string {
	runID := strings.TrimSpace(task.RunID)
	if runID == "" {
		runID = "run_qa"
	}
	question := strings.TrimSpace(task.Question)
	if question == "" {
		question = "What architecture evidence is available?"
	}
	provider := strings.TrimSpace(string(acpruntime.StepProviderQA))
	example := map[string]any{
		"version":      1,
		"run_id":       runID,
		"question":     question,
		"answer":       "Short answer grounded only in context-pack evidence.",
		"citations":    []map[string]string{{"path": "reports/as-is/overview.md", "reason": "supports the stated architecture fact"}},
		"unresolved":   []string{"missing owner evidence for the requested service"},
		"confidence":   0.6,
		"provider":     provider,
		"generated_at": "2026-01-01T00:00:00Z",
	}
	raw, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return `{"version":1}`
	}
	return string(raw)
}

func WorkspacePromptPackSection(task acpruntime.Task) string {
	workspacePath := strings.TrimSpace(task.Workspace)
	packPath := strings.TrimSpace(WorkspacePromptPackPath(task.StepID))
	if workspacePath == "" || packPath == "" {
		return ""
	}

	raw, err := os.ReadFile(filepath.Join(filepath.Clean(workspacePath), filepath.FromSlash(packPath)))
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		return strings.Join([]string{
			`WORKSPACE PROMPT PACK CONTENT LAYER:`,
			fmt.Sprintf(`- Failed to read %q; continue using enforced runtime policy only.`, packPath),
		}, "\n")
	}

	content := strings.TrimSpace(string(raw))
	if content == "" {
		return ""
	}
	return strings.Join([]string{
		`WORKSPACE PROMPT PACK CONTENT LAYER:`,
		fmt.Sprintf(`- Source file: %q`, packPath),
		`- This workspace prompt pack is an editable content layer only. It MUST NOT weaken or override any enforced policy or contract rule above.`,
		`BEGIN WORKSPACE PROMPT PACK`,
		content,
		`END WORKSPACE PROMPT PACK`,
	}, "\n")
}

func DraftArtifactRepairHints(task acpruntime.Task, validationErr error) []string {
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	lines := []string{
		fmt.Sprintf(`- Repair %s to the canonical ACP runtime draft manifest contract before exiting successfully.`, manifestFile),
		`- version MUST be the integer 1; string values such as "0.1.0" are invalid.`,
		`- The manifest MUST include run_id, step_id, step_contract, agent_role, and outputs[].`,
		`- outputs[].path MUST stay relative to draft_final_root and outputs[].canonical_path MUST stay workspace-relative.`,
		`- If valid draft files already exist under draft_final_root, reuse them and exit successfully.`,
		`- Repair mode is draft-only: do not invent extra repository file reads/writes after draft files already exist.`,
	}
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		lines = append(lines,
			`- constitution-draft.json exact required outputs[] entries are: {"path":"charter-overview.md","canonical_path":"charter/overview.md","kind":"charter","title":"Constitution"} and {"path":"baseline-subagents.yaml","canonical_path":"skills/subagents.yaml","kind":"bundle","title":"Baseline Subagents"}.`,
			`- Do NOT emit legacy constitution shapes such as schema_version, system_id, services, relations, governance, coverage_notes, or version:"0.1.0".`,
			`- Draft files referenced by constitution-draft.json must exist under draft_final_root before the runtime process exits successfully.`,
			`- charter-overview.md must be evidence-backed final content, not unchanged first-action or recovery scaffold text; baseline-subagents.yaml may remain the baseline YAML bundle.`,
		)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines,
			`- asis-draft-manifest.json is the only publish-surface manifest for reports/as-is/*, reports/coverage/*, and reports/agent-outputs/* drafts.`,
			`- After draft artifact repair, stop after artifacts validate; do not emit any legacy metadata registration surface.`,
		)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines,
			`- proposals-draft-manifest.json is the only publish-surface manifest for proposals/* and reports/changelog/* drafts.`,
			`- step_contract MUST be exactly "proposals"; version MUST be integer 1.`,
			`- outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/* and MUST be unique.`,
			`- If current-run staged final reports/findings/findings.md contains finding IDs, repaired proposal/changelog markdown MUST cite current-run finding IDs and MUST NOT claim structured findings are absent.`,
			`- Current-run findings/coverage evidence is staged at reports/taskruns/<run_id>/staging/final/reports/* before publish; do NOT treat reports/taskruns/<run_id>/reports/* as the current-run source.`,
			`- If findings include high/medium severity, repaired proposal.md MUST link at least one high/medium finding to affected surface/path and recommended operator action; generic review-only text is insufficient.`,
			`- Do NOT add legacy top-level fields such as pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps.`,
			`- After draft artifact repair, stop after artifacts validate; do not emit any legacy metadata registration surface.`,
		)
	}
	if validationErr != nil {
		lines = append(lines, fmt.Sprintf(`- Previous draft artifact validation failure: %s`, compactRetryHint(validationErr.Error())))
	}
	return lines
}

func CollectRepoEntrypointHints(task acpruntime.Task) []string {
	if len(task.ReadContextRoots) == 0 {
		return nil
	}
	patterns := []string{
		"README.*",
		".github/CODEOWNERS",
		"CODEOWNERS",
		"OWNERS*",
		"MAINTAINERS*",
		"catalog-info.yaml",
		"pyproject.toml",
		"package.json",
		"docker-compose*",
		"skaffold.yaml",
		"Makefile",
	}
	hints := []string{}
	seen := map[string]struct{}{}
	for _, root := range task.ReadContextRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		for _, pattern := range patterns {
			matches, err := filepath.Glob(filepath.Join(root, pattern))
			if err != nil {
				continue
			}
			sort.Strings(matches)
			for _, match := range matches {
				matchInfo, err := os.Stat(match)
				if err != nil || matchInfo.IsDir() {
					continue
				}
				display := filepath.ToSlash(strings.TrimSpace(match))
				if display == "" {
					continue
				}
				if _, exists := seen[display]; exists {
					continue
				}
				seen[display] = struct{}{}
				hints = append(hints, display)
				if len(hints) >= 8 {
					return hints
				}
			}
		}
	}
	return hints
}

func CollectPathScopeFileHints(task acpruntime.Task) []string {
	if len(task.ReadContextRoots) == 0 || len(task.PathScopes) == 0 {
		return nil
	}
	workspace := filepath.Clean(strings.TrimSpace(task.Workspace))
	writeRoot := filepath.Clean(strings.TrimSpace(task.WriteRoot))
	seen := map[string]struct{}{}
	hints := []string{}
	add := func(path string, info os.FileInfo) {
		path = filepath.ToSlash(strings.TrimSpace(path))
		if path == "" || !collectPathScopeHintAllowed(path, info) {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		hints = append(hints, path)
	}
	for _, root := range task.ReadContextRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		cleanRoot := filepath.Clean(root)
		if (workspace != "." && cleanRoot == workspace) || (writeRoot != "." && cleanRoot == writeRoot) {
			continue
		}
		if strings.Contains(filepath.ToSlash(cleanRoot), "/reports/taskruns/") {
			continue
		}
		info, err := os.Stat(cleanRoot)
		if err != nil || !info.IsDir() {
			continue
		}
		for _, scope := range task.PathScopes {
			for _, rel := range collectPathScopeFileHintsForScope(cleanRoot, scope) {
				info, err := os.Stat(filepath.Join(cleanRoot, filepath.FromSlash(rel)))
				if err != nil {
					continue
				}
				add(rel, info)
			}
		}
	}
	sortCollectPathScopeFileHints(hints)
	if len(hints) > 18 {
		hints = hints[:18]
	}
	return hints
}

func collectPathScopeFileHintsForScope(root string, scope string) []string {
	scope = strings.Trim(strings.TrimSpace(scope), string(filepath.Separator))
	scope = strings.Trim(scope, "/")
	if scope == "" {
		return nil
	}
	target := filepath.Join(root, filepath.FromSlash(scope))
	info, err := os.Stat(target)
	if err != nil {
		return nil
	}
	if !info.IsDir() {
		if collectPathScopeHintAllowed(scope, info) {
			return []string{filepath.ToSlash(scope)}
		}
		return nil
	}
	hints := []string{}
	_ = filepath.WalkDir(target, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", ".venv", "__pycache__":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !collectPathScopeHintAllowed(rel, info) {
			return nil
		}
		hints = append(hints, filepath.ToSlash(rel))
		return nil
	})
	sortCollectPathScopeFileHints(hints)
	if len(hints) > 6 {
		hints = hints[:6]
	}
	return hints
}

func sortCollectPathScopeFileHints(hints []string) {
	sort.Slice(hints, func(i, j int) bool {
		leftRank := collectPathScopeHintRank(hints[i])
		rightRank := collectPathScopeHintRank(hints[j])
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return hints[i] < hints[j]
	})
}

func collectPathScopeHintRank(path string) int {
	name := strings.ToLower(filepath.Base(path))
	switch name {
	case "readme.md", "readme.adoc", "readme.rst", "readme.txt", "contributing.md":
		return 0
	case "package.json", "pyproject.toml", "go.mod", "pom.xml", "build.gradle", "settings.gradle", "cargo.toml":
		return 1
	case "docker-compose.yml", "docker-compose.yaml", "dockerfile", "makefile":
		return 2
	case ".env.example", "workspace.yaml", "skaffold.yaml", "catalog-info.yaml":
		return 3
	}
	if strings.HasPrefix(name, "dockerfile") || strings.HasPrefix(name, "docker-compose.") {
		return 4
	}
	if strings.HasSuffix(name, ".md") {
		return 5
	}
	if strings.HasPrefix(name, ".") {
		return 9
	}
	return 6
}

func collectPathScopeHintAllowed(path string, info os.FileInfo) bool {
	if info == nil || info.IsDir() {
		return false
	}
	const maxCollectPathScopeHintBytes = 96 * 1024
	if info.Size() > maxCollectPathScopeHintBytes {
		return false
	}
	name := strings.ToLower(filepath.Base(path))
	if name == "" {
		return false
	}
	switch name {
	case ".test_durations", "mypy-baseline.txt", "ty-baseline.txt", "pnpm-lock.yaml", "package-lock.json", "yarn.lock", "uv.lock", "go.sum":
		return false
	}
	if strings.HasSuffix(name, ".lock") || strings.HasSuffix(name, "-lock.yaml") || strings.HasSuffix(name, "-lock.json") {
		return false
	}
	if strings.HasPrefix(name, "coverage.") || strings.HasSuffix(name, ".snap") || strings.HasSuffix(name, ".snapshot") {
		return false
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyPath(values []string) string {
	for _, value := range values {
		value = filepath.ToSlash(strings.Trim(strings.TrimSpace(value), "/"))
		if value != "" && value != "." {
			return value
		}
	}
	return ""
}

func collectEvidencePath(task acpruntime.Task, evidencePaths []string) string {
	if rootFileScopes := rootFileShardPathScopes(task.PathScopes); len(rootFileScopes) > 0 {
		if rootFileEvidence := rootFileShardPathScopes(evidencePaths); len(rootFileEvidence) > 0 {
			return preferredRootFileEvidencePath(rootFileEvidence)
		}
		return preferredRootFileEvidencePath(rootFileScopes)
	}
	if value := firstNonEmptyPath(evidencePaths); value != "" {
		return value
	}
	return firstNonEmptyPath(task.PathScopes)
}

func preferredRootFileEvidencePath(pathScopes []string) string {
	preferred := []func(string) bool{
		func(value string) bool {
			lower := strings.ToLower(value)
			return lower == "readme" || strings.HasPrefix(lower, "readme.")
		},
		func(value string) bool {
			return strings.EqualFold(value, "makefile")
		},
		func(value string) bool {
			lower := strings.ToLower(value)
			switch lower {
			case "pom.xml", "package.json", "go.mod", "build.gradle", "build.gradle.kts", "gradlew", "mvnw", "dockerfile", "justfile":
				return true
			default:
				return strings.HasPrefix(lower, "skaffold") || strings.HasPrefix(lower, "docker-compose")
			}
		},
	}
	for _, match := range preferred {
		for _, value := range pathScopes {
			if match(value) {
				return value
			}
		}
	}
	return firstNonEmptyPath(pathScopes)
}

func collectCoverageMissingSkeleton(task acpruntime.Task) []string {
	if strings.TrimSpace(task.StepID) != "refresh.step1.collect" {
		return []string{"owner mapping evidence not confirmed from scoped repository files"}
	}
	return []string{
		"owner mapping evidence not confirmed from scoped repository files",
		"operational runbook evidence not confirmed from scoped repository files",
		"external dependency evidence not confirmed from scoped repository files",
	}
}

func collectQuestionsSkeleton(task acpruntime.Task, idStem string, topic string) []contracts.Question {
	if strings.TrimSpace(task.StepID) != "refresh.step1.collect" {
		return []contracts.Question{}
	}
	stem := idComponent(firstNonEmpty(idStem, topic, "refresh"))
	if stem == "" {
		stem = "refresh"
	}
	subject := strings.TrimSpace(topic)
	if subject == "" {
		subject = "scoped repository surface"
	}
	return []contracts.Question{
		{
			ID:       fmt.Sprintf("question.%s.owner.mapping", stem),
			Text:     fmt.Sprintf("Which team owns the %s surface and its operational escalation path?", subject),
			Priority: "medium",
		},
	}
}

func collectSemanticSkeleton(task acpruntime.Task, repo string, evidencePath string, shardSlug string, idStem string, topic string, coverageMissing []string) contracts.SemanticSnapshot {
	repoStem := idComponent(firstNonEmpty(repo, "repo"))
	shardStem := idComponent(firstNonEmpty(idStem, shardSlug, topic, "shard"))
	repoEntityID := "svc." + repoStem
	shardEntityID := "svc." + shardStem
	if shardEntityID == repoEntityID {
		shardEntityID = shardEntityID + ".surface"
	}
	if strings.TrimSpace(topic) == "" {
		topic = shardSlug
	}
	if strings.TrimSpace(topic) == "" {
		topic = "architecture"
	}
	if strings.TrimSpace(repo) == "" {
		repo = "repo"
	}
	if strings.TrimSpace(evidencePath) == "" {
		evidencePath = "README.md"
	}
	evidence := []contracts.Evidence{{
		Repo: repo,
		Path: evidencePath,
	}}
	questions := collectQuestionsSkeleton(task, shardStem, topic)
	if len(questions) == 0 {
		questions = []contracts.Question{{
			ID:         fmt.Sprintf("question.%s.owner.mapping", shardStem),
			Text:       fmt.Sprintf("Which team owns the %s surface and its operational escalation path?", strings.TrimSpace(topic)),
			Priority:   "medium",
			RelatedIDs: []string{shardEntityID},
		}}
	}
	for idx := range questions {
		if len(questions[idx].RelatedIDs) == 0 {
			questions[idx].RelatedIDs = []string{shardEntityID}
		}
	}
	return contracts.SemanticSnapshot{
		Coverage: contracts.Coverage{
			Observed: []string{topic},
			Missing:  coverageMissing,
			Notes:    []string{"Collect manifest covers the assigned shard scope with evidence paths listed in citations."},
		},
		Questions: questions,
		Entities: []contracts.Entity{
			{
				ID:   repoEntityID,
				Type: "service",
				Name: titleFromSlug(repoStem),
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.55,
					Evidence:   evidence,
				},
			},
			{
				ID:   shardEntityID,
				Type: "service",
				Name: titleFromSlug(firstNonEmpty(shardSlug, topic, "Scoped Surface")),
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.55,
					Evidence:   evidence,
				},
			},
		},
		Edges: []contracts.Edge{{
			ID:   fmt.Sprintf("edge.%s.contained-by-repo", shardStem),
			Type: "contains",
			From: repoEntityID,
			To:   shardEntityID,
			Name: "contains scoped surface",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.45,
				Evidence:   evidence,
			},
		}},
		Findings: []contracts.Finding{{
			ID:          fmt.Sprintf("finding.%s.owner.mapping", shardStem),
			Severity:    "medium",
			Title:       "Owner mapping not confirmed",
			Description: fmt.Sprintf("Scoped evidence identifies the %s surface but does not confirm an owning team or escalation path.", strings.TrimSpace(topic)),
			RuleID:      "rule.owner.mapping.required",
			RelatedIDs:  []string{shardEntityID},
			Provenance: contracts.Provenance{
				Kind:       "inference",
				Confidence: 0.45,
				Evidence:   evidence,
			},
		}},
	}
}

func nonEmptyList(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func slugComponent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if len(result) > 72 {
		result = strings.Trim(result[:72], "-")
	}
	return result
}

func idComponent(value string) string {
	value = strings.ReplaceAll(slugComponent(value), "-", ".")
	if value == "" {
		return "shard"
	}
	return value
}

func titleFromSlug(value string) string {
	parts := strings.Fields(strings.ReplaceAll(slugComponent(value), "-", " "))
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	if len(parts) == 0 {
		return "Shard Overview"
	}
	return strings.Join(parts, " ")
}

func compactRetryHint(value string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(normalized) <= 320 {
		return normalized
	}
	return normalized[:317] + "..."
}

func nonEmptyOr(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

func isCollectStep(stepID string) bool {
	switch strings.TrimSpace(stepID) {
	case "init.step1.collect", "refresh.step1.collect":
		return true
	default:
		return false
	}
}

func isPromptHintedStep(stepID string) bool {
	switch strings.TrimSpace(stepID) {
	case "init.step0.constitution", "init.step1.collect", "refresh.step1.collect":
		return true
	default:
		return false
	}
}
