package promptcontract

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
)

func ComposeCollectManifestRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error) string {
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), "shard-pack-manifest.json")
	authoredDocs := authoredRepairDocuments(task.WriteRoot)
	evidencePaths := repairEvidenceCandidates(task)
	skeleton := steppolicy.CollectManifestTaskSkeleton(task, authoredDocs, evidencePaths)
	firstCommand := collectManifestRepairWriteFirstGuidance(task, authoredDocs, evidencePaths)
	sections := []string{
		fmt.Sprintf("You are ACP runtime provider %q in collect manifest repair mode.", provider),
		"COLLECT MANIFEST EVIDENCE-FIRST REPAIR:",
		"- Repair shard-pack-manifest.json from the existing authored documents and bounded repository evidence; do not start with a placeholder scaffold.",
		"- Do not run a separate read-only preflight and do not print an evidence packet as the only action.",
		"- Do not answer with a plan, status note, or analysis-only message before the write.",
		"- Forbidden analysis-only phrases before the write: I have enough evidence; I will write; I will now write; ready to write; the manifest will cite.",
		"- The first command below is a write-first provider-authored command contract: it must read existing authored documents plus bounded evidence and write shard-pack-manifest.json before returning.",
		"- No deterministic helper writes the manifest for you; you must author shard-pack-manifest.json from the observed markdown and allowed repository evidence.",
		"- Read only the listed repository evidence candidates if authored docs need support; do not start an open-ended repository sweep.",
		"- Repository evidence in citations/provenance must be file-level: path_scopes may guide discovery, but directories or missing paths must become coverage gaps/questions, never citation paths.",
		"- Every citations[].id must be unique; when citing multiple repo files from one authored document, derive each citation id from the shard/document stem plus the repo path slug, not from the authored markdown document id alone.",
		"- JSON syntax-only checks such as jq empty or python3 -m json.tool are insufficient; the first command must parse JSON and verify semantic.questions[] all have id and text, citations[].id has no duplicates, every citation has non-empty claim_ids and document_ids, every documents[].citation_ids value exists, and every citation/provenance repo path is an existing file.",
		fmt.Sprintf("- Write exactly one file: %q.", manifestTarget),
		"- Do not rewrite existing authored markdown documents.",
		"- documents[].canonical_path must be a stable promoted workspace path. Never use write_root, artifact_root, absolute paths, reports/taskruns, staging, raw logs, or runtime metadata paths as canonical_path.",
		"- If shard-pack-manifest.json already exists but is invalid, do not inspect or patch it; overwrite it after the evidence pass.",
		"- The manifest top-level object may contain only ACP shard-pack-manifest fields: version, run_id, step_id, shard_id, domain_id, agent_role, artifact_root, repo_scopes, path_scopes, summary, documents, citations, and semantic.",
		"- Do not add top-level claims, claim_map, validation, metadata, compatibility, schema, or any alternate semantic wrapper; claim IDs belong only in citations[].claim_ids.",
		"- Every citations[] item must include a non-empty claim_ids array; derive stable claim ids from the citation/shard semantic claim, for example claim.<shard>.<surface>.",
		"- Do not leave template placeholders in JSON strings. Literal SHARD, <shard>, <claim>, TODO, REPLACE_ME, or quoted suffix fragments such as \"claim.foo.\"SHARD are invalid.",
		"FIRST COLLECT MANIFEST REPAIR COMMAND:",
		"- Execute one bounded filesystem command as your next action. Do not inspect sibling taskruns, read raw logs, or write any other file before this command.",
		"- The command must read authored markdown already in write_root, optionally read the listed evidence candidates, and write the final shard-pack-manifest.json before it returns.",
		"- The command must not stop after printing evidence or saying what it will write; preflight-only completion is a failed no-op repair.",
		firstCommand,
		"TASK-SPECIFIC MANIFEST JSON SKELETON:",
		strings.TrimSpace(skeleton),
		"SKELETON USE:",
		"- Use this JSON only as the task-specific schema/key/type guide, not as final content.",
		"- Replace skeleton citations, coverage, questions, entities, edges, findings, titles, and descriptions with facts from authored docs and allowed repository evidence.",
		"- Copying this skeleton unchanged is invalid and will be rejected as scaffold-only output.",
		"SEMANTIC EXTRACTION REQUIREMENT:",
		"- Before writing shard-pack-manifest.json, extract semantic signal from the authored documents: named systems, runtimes, services, data stores, build/deploy/test/config surfaces, and material risks or gaps.",
		"- Evidence-rich authored documents require concrete semantic.entities beyond the repo plus shard wrapper.",
		"- Evidence-rich authored documents require concrete semantic.edges beyond repo/shard contains relationships, using relationships such as uses, configures, depends_on, runs, exposes, stores, builds, or deploys when supported by evidence.",
		"- Evidence-rich authored documents require semantic.findings or semantic.questions beyond a generic owner-mapping gap when the authored docs describe real stack, deploy, environment, testing, licensing, or operational concerns.",
		"- A manifest with many citations but only repo/shard entities, only contains edges, and only Owner mapping not confirmed is invalid scaffold-only semantic output.",
		"- Coverage notes must state concrete observed architecture/configuration surfaces; do not use generic notes like \"Collect manifest covers the assigned shard scope\" as proof of completeness.",
		"- Coverage gaps must not use runtime-process wording such as \"not examined in this bounded pass\"; write \"not confirmed in scoped repository evidence\" instead.",
		"Artifact-only repair contract:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		"- Write or replace only write_root/shard-pack-manifest.json.",
		fmt.Sprintf("- Exact allowed write target: %q.", manifestTarget),
		"- Final action must be: write only write_root/shard-pack-manifest.json, then exit successfully.",
		"- Backend validation, not stdout claims, is the success surface.",
		"- Exit with code 0 only after shard-pack-manifest.json is complete, evidence-backed, and not copied from the skeleton.",
		fmt.Sprintf(`- artifact_root (workspace-relative) = %q`, strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- read_context_roots = %q`, strings.Join(task.ReadContextRoots, ", ")),
		fmt.Sprintf(`- domain_id = %q`, strings.TrimSpace(task.DomainID)),
		fmt.Sprintf(`- shard_id = %q`, strings.TrimSpace(task.ShardID)),
		fmt.Sprintf(`- agent_role = %q`, strings.TrimSpace(task.AgentRole)),
		fmt.Sprintf(`- repo_scopes = %q`, strings.Join(task.RepoScopes, ", ")),
		fmt.Sprintf(`- path_scopes = %q`, strings.Join(task.PathScopes, ", ")),
	}
	repairLines := []string{
		"Existing authored documents in write_root must drive manifest repair; read them, do not rewrite them.",
	}
	if len(authoredDocs) > 0 {
		repairLines = append(repairLines, "Existing authored document files in write_root:")
		for _, rel := range authoredDocs {
			repairLines = append(repairLines, fmt.Sprintf("- %s", rel))
		}
		repairLines = append(repairLines, "Stable collect canonical_path mapping for existing authored documents:")
		for _, mapping := range collectCanonicalPathMappingLines(task, authoredDocs) {
			repairLines = append(repairLines, "- "+mapping)
		}
	}
	if len(evidencePaths) > 0 {
		repairLines = append(repairLines, "Repository evidence candidates available for bounded repair:")
		for _, rel := range evidencePaths {
			repairLines = append(repairLines, fmt.Sprintf("- %s", rel))
		}
	}
	repairLines = append(repairLines, overwriteCollectManifestRepairInstructions()...)
	if focus := collectManifestRepairValidationFocus(validationErr); len(focus) > 0 {
		repairLines = append(repairLines, "VALIDATION-SPECIFIC MANIFEST REPAIR FOCUS:")
		repairLines = append(repairLines, focus...)
	}
	repairLines = append(repairLines,
		"CANONICAL SEMANTIC SHAPE:",
	)
	repairLines = append(repairLines, collectManifestCanonicalShapeBlock()...)
	repairLines = append(repairLines,
		"COLLECT MANIFEST REPAIR CHECKLIST:",
	)
	repairLines = append(repairLines, compactCollectManifestValidationChecklist(strings.TrimSpace(task.ArtifactRoot))...)
	repairLines = append(repairLines,
		`- Repair-mode note: if schemas/* or docs/spec/* are absent from the runtime workspace, do not look for them; use this embedded checklist.`,
		`- The task-specific JSON skeleton above is a schema guide only; copied scaffold semantic is invalid even when the JSON schema passes.`,
	)
	if detail := errorText(validationErr); detail != "" {
		repairLines = append(repairLines, fmt.Sprintf("- Previous collect manifest validation failure: %s", detail))
	}
	sections = append(sections, strings.Join(repairLines, "\n"))
	return strings.Join(sections, "\n\n")
}

func collectManifestRepairWriteFirstGuidance(task acpruntime.Task, authoredDocs []string, evidencePaths []string) string {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		writeRoot = "."
	}
	manifestTarget := filepath.Join(writeRoot, "shard-pack-manifest.json")
	lines := []string{
		"- First command contract:",
		"  - read bounded authored markdown under write_root;",
		"  - read only listed repository evidence candidates if needed;",
		"  - verify every manifest citation/provenance repo path with file-level checks such as test -f, rg --files, or portable find ... -type f -print;",
		"  - verify semantic.questions[] all have id and text, citations[].id has no duplicates, every citation has non-empty claim_ids and document_ids, and every documents[].citation_ids value references an existing citation;",
		"  - write the final provider-authored manifest to " + shellSingleQuote(manifestTarget) + " before returning;",
		"  - run a local `test -s`/JSON parse check plus structural and file-level evidence path checks after the write.",
		"- Exact manifest write target: " + shellSingleQuote(manifestTarget),
		"- Authored markdown inputs already present under write_root:",
	}
	if len(authoredDocs) == 0 {
		lines = append(lines, "  - discover non-empty *.md files under write_root with a bounded depth and ignore shard-pack-manifest.json/runtime-execution.json.")
	} else {
		for _, rel := range authoredDocs {
			lines = append(lines, "  - "+rel)
		}
	}
	if len(authoredDocs) > 0 {
		lines = append(lines, "- Stable canonical_path mapping to copy into documents[] exactly:")
		for _, mapping := range collectCanonicalPathMappingLines(task, authoredDocs) {
			lines = append(lines, "  - "+mapping)
		}
	}
	lines = append(lines, "- Bounded repository evidence candidates:")
	if len(evidencePaths) == 0 {
		lines = append(lines, "  - README.md", "  - README.adoc")
	} else {
		for _, rel := range evidencePaths {
			lines = append(lines, "  - "+rel)
		}
	}
	lines = append(lines,
		"- Do not emit only this evidence list. The same first command must write shard-pack-manifest.json.",
		"- Reject any documents[].canonical_path that contains reports/taskruns, /staging/, raw runtime paths, write_root, artifact_root, or an absolute filesystem prefix.",
		"- Do not cite repository directories. If a candidate is directory-only or missing, record it in coverage.missing/questions instead of citations/provenance.",
		"- Every citations[].id must be unique; when citing multiple repository files, include a path slug in each id and update documents[].citation_ids to the exact generated IDs.",
		"- Use python3, not python; do not use GNU-only find -printf; do not assign to the zsh-reserved status variable.",
		"- Do not use a copied skeleton or generic repo/shard wrapper semantic as the final manifest.",
	)
	return strings.Join(lines, "\n")
}

func collectManifestRepairValidationFocus(validationErr error) []string {
	detail := strings.ToLower(errorText(validationErr))
	if strings.TrimSpace(detail) == "" {
		return nil
	}
	lines := []string{}
	if strings.Contains(detail, "/semantic") && strings.Contains(detail, "required: missing properties: 'findings'") {
		lines = append(lines,
			"- Terminal shape focus: semantic must contain the required findings array. Preserve every existing manifest value and add semantic.findings as [] when no evidence-backed findings were observed.",
			"- Do not invent findings to satisfy the shape; run a local JSON check that semantic.findings exists and is an array before exiting 0.",
		)
	}
	if strings.Contains(detail, "semantic/questions") && strings.Contains(detail, "text") {
		lines = append(lines,
			"- Terminal shape focus: rewrite every semantic.questions[] item as an object with both id and text; do not use question, title-only, or id-only question objects.",
			"- Run a local JSON check that fails when any semantic.questions[] item lacks id or text before exiting 0.",
		)
	}
	if strings.Contains(detail, "citations") && strings.Contains(detail, ".id must be unique") {
		lines = append(lines,
			"- Terminal shape focus: regenerate citations[].id values so every citation id is unique.",
			"- If several citations point to different repo files for one authored markdown document, derive IDs from shard/document stem plus repo path slug, for example cite.<shard>.<doc>.<path-slug>.",
			"- Update every documents[].citation_ids array to reference the exact regenerated citation IDs; do not leave stale duplicate IDs behind.",
			"- Run a local duplicate check on citations[].id before exiting 0.",
		)
	}
	if strings.Contains(detail, ".claim_ids is required") || strings.Contains(detail, ".document_ids is required") ||
		(strings.Contains(detail, "citation_ids") && strings.Contains(detail, "reference")) {
		lines = append(lines,
			"- Terminal binding focus: every citations[] item needs non-empty claim_ids and document_ids arrays.",
			"- Every citation document_ids entry must point to an existing documents[].id, and every documents[].citation_ids entry must point to an existing citations[].id.",
			"- Rewrite citation/document bindings together; do not patch only one side of the relationship.",
		)
	}
	return lines
}

func collectCanonicalPathMappingLines(task acpruntime.Task, docPaths []string) []string {
	lines := make([]string, 0, len(docPaths))
	for _, rel := range docPaths {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s -> %s", rel, steppolicy.CollectManifestCanonicalPath(task, rel)))
	}
	return lines
}

func ComposeCollectArtifactPairRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error) string {
	docRel := collectArtifactPairRepairDocumentPath(task, validationErr)
	docTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), filepath.FromSlash(docRel))
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), "shard-pack-manifest.json")
	evidencePaths := repairEvidenceCandidates(task)
	if provider == acpruntime.ProviderQwenCode && strings.Contains(strings.ToLower(errorText(validationErr)), "runtime_stalled_before_artifacts") {
		return composeQwenToolFirstCollectPromptWithTargets(task, validationErr, docRel, docTarget, manifestTarget, evidencePaths)
	}
	if useCompactCollectPairRepairPrompt(validationErr) {
		return composeCompactCollectArtifactPairRepairPrompt(provider, task, validationErr, docRel, docTarget, manifestTarget, evidencePaths)
	}
	skeleton := steppolicy.CollectManifestTaskSkeleton(task, []string{docRel}, evidencePaths)
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in collect artifact pair focused recovery mode.", provider),
		"COLLECT PAIR WRITE-FIRST EVIDENCE REPAIR:",
		"- This repair is not a bootstrap/fallback writer. Do not create a seed-only or recovery-summary pair.",
		"- Do not run a separate read-only preflight and do not print an evidence packet as the only action.",
		"- Do not answer with a plan, status note, or analysis-only message before the writes.",
		"- Forbidden analysis-only phrases before the writes: I have enough evidence; I am now writing; I'm now writing; I will write; I will now write; ready to write; the manifest will cite.",
		"- Your next action must be one bounded filesystem command that reads allowed evidence and writes the markdown document first, then shard-pack-manifest.json, before any optional explanation.",
		"- The first command may read only read_context_roots and listed evidence candidates, at most 8 files and at most the first 6000 bytes from each file; do not run open-ended find/grep/sed/tree sweeps.",
		"- The first command must not contain a hard-coded required phrase list, `missing expected evidence` gate, or exact-substring precondition for planned claims.",
		"- Before writing, the only allowed evidence prechecks are structural: exact targets stay under write_root and at least one allowed evidence file yielded bytes after bounded prefix reads.",
		"- If a candidate is larger than the per-file prefix budget, truncate to the first 6000 bytes or skip that candidate and continue; do not abort the repair with errors such as `read file exceeds size limit` when other evidence is available.",
		"- Do not add any other pre-write `raise SystemExit`/`exit 1` checks for semantic richness, entity counts, edge counts, citation counts, generated text shape, or self-invented validation; backend validation is the semantic gate.",
		"- Keep the first command mechanically simple: no Python f-strings, no `.format(...)` templates, no generated Python source strings, and no nested quote tricks for markdown/JSON assembly.",
		"- If using Python, build markdown from a list of plain strings plus simple concatenation, build JSON as dictionaries/lists, and write it with `json.dumps`; prefer a short script over a clever generator.",
		"- Derive claims from observed snippets after reading. If a planned claim is not present in the snippets, omit it or record a coverage gap; do not abort the whole repair because a guessed phrase is absent.",
		"- The final markdown must be operator-facing architecture evidence, not a recovery/process log; do not mention bounded reads/passes, guessed paths/files/evidence, expected-missing path checks, recovery attempts, or later repair.",
		"- When describing coverage gaps, do not write phrases like \"not examined in this bounded pass\" or \"during the bounded pass\"; use \"not confirmed in scoped repository evidence\" or a concrete missing evidence category instead.",
		"- Write the markdown document first as concise evidence-backed content, then write the manifest that references it; do not emit prose, status, or optional enrichment until both exact targets are non-empty and marker-free.",
		fmt.Sprintf("- Write exactly two files in the first command: %q and %q.", docTarget, manifestTarget),
		"- If the authored document target already exists, rewrite it completely from observed evidence; do not leave stale missing-path claims in place.",
		"- Do not write any file outside the exact write_root collect pair.",
		"- Do not delete existing files, run rm -f, use git rev-parse, rely on cwd discovery, inspect sibling reports/taskruns, or read raw logs.",
		"- If bounded repo evidence cannot be read, exit non-zero without writing fallback scaffold.",
		"FIRST COLLECT PAIR REPAIR WRITE-FIRST COMMAND:",
		"- Execute one filesystem command now. It must both read the bounded evidence and write the final markdown + manifest before returning.",
		"- You may use shell or Python, but any heredoc content must be final provider-authored evidence-backed content, never placeholders or copied skeleton text.",
		"COLLECT PAIR REPAIR EVIDENCE LIMITS:",
		"- Use only the listed task metadata and bounded repository evidence candidates below.",
		"- Do not read lockfiles, generated baselines, test duration indexes, raw logs, reports/taskruns history, or full files larger than 96 KiB during collect pair repair; read a bounded prefix or skip oversized candidates.",
		"- Do not verify claims with a provider-invented exact phrase checklist such as `required = [...]` plus `raise SystemExit('missing expected evidence ...')` before writing.",
		"- Use short observed snippets, package names, service names, config keys, and file paths from the bytes actually read; never require marketing copy or long prose to match exactly.",
		"- Do not abort before writing because your own generated script has no entities/edges/findings; write the best observed manifest and let ACP validation reject it if it is insufficient.",
		"- Explain concrete components, runtime/config/deploy/test/data surfaces, ownership gaps, and dependencies in the markdown document.",
		"- Build shard-pack-manifest.json from that markdown plus the same repo/path evidence: documents, citations, semantic.coverage, semantic.questions, semantic.entities, semantic.edges, and semantic.findings.",
		"- The manifest top-level object may contain only ACP shard-pack-manifest fields: version, run_id, step_id, shard_id, domain_id, agent_role, artifact_root, repo_scopes, path_scopes, summary, documents, citations, and semantic.",
		"- documents[].canonical_path must be the stable promoted path from the task-specific skeleton, never the staging path where the markdown is authored.",
		"- Any canonical_path beginning with reports/taskruns/ or containing /staging/ is invalid even if documents[].path is correct.",
		"- Do not add top-level claims, claim_map, validation, metadata, compatibility, schema, or any alternate semantic wrapper; claim IDs belong only in citations[].claim_ids.",
		"- Do not leave template placeholders in JSON strings. Literal SHARD, <shard>, <claim>, TODO, REPLACE_ME, or quoted suffix fragments such as \"claim.foo.\"SHARD are invalid.",
		"- Use repo/path provenance for every semantic evidence object; stdout claims are diagnostics only.",
		"- Every citation/provenance path must be a concrete existing repository file; directories and missing paths are coverage gaps/questions, not evidence.",
		"- JSON syntax-only checks such as jq empty or python3 -m json.tool are insufficient; include file-level citation/provenance checks before a successful exit.",
		"- Use python3, not python; do not use GNU-only find -printf; do not assign to the zsh-reserved status variable.",
	}
	if focus := collectArtifactPairRepairValidationFocus(validationErr); len(focus) > 0 {
		lines = append(lines, "VALIDATION-SPECIFIC REPAIR FOCUS:")
		lines = append(lines, focus...)
	}
	lines = append(lines,
		"TASK-SPECIFIC MANIFEST JSON SKELETON:",
		strings.TrimSpace(skeleton),
		"SKELETON USE:",
		"- Use this JSON only as the task-specific schema/key/type guide, not as final content.",
		"- Replace skeleton citations, coverage, questions, entities, edges, findings, titles, and descriptions with facts from allowed repository evidence.",
		"- Copying this skeleton unchanged is invalid and will be rejected as scaffold-only output.",
		"RECOVERY ACCEPTANCE REQUIREMENT:",
		"- Successful recovery output must not contain ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT, Recovery Bootstrap/Recovery Summary scaffold text, Recovery Evidence Summary fallback prose, or the normal collect first-action scaffold prose.",
		"- Successful recovery output must not say seed-only collect recovery fallback, Additional provider enrichment should replace, Treat this as diagnostic evidence until, first bounded evidence read was attempted, initial artifact records only, or will be repaired with concrete.",
		"- Successful recovery output must not mention bounded read, bounded pass, guessed path, guessed file, guessed evidence, missing expected evidence, expected path was not present, or expected file was not present as final artifact content.",
		"- A manifest with many citations but only repo/shard wrapper entities, only contains edges, or only generic owner-mapping findings is invalid.",
		"- Exit successfully only after both exact targets are complete, marker-free, and evidence-backed.",
		"- A noop, zero-output, unchanged skeleton, or partially-written repair is terminal; exit non-zero instead of claiming success when either exact target is missing or unchanged.",
		"FINAL SELF-CHECK COMMAND:",
		"test -s "+shellSingleQuote(docTarget)+" && test -s "+shellSingleQuote(manifestTarget)+" && ! grep -E 'ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT|Recovery Bootstrap|Recovery Summary|Recovery Evidence Summary|seed-only collect recovery fallback|Additional provider enrichment should replace|Treat this as diagnostic evidence until|first bounded evidence read was attempted|initial artifact records only|will be repaired with concrete|bounded read|bounded pass|guessed path|guessed file|guessed evidence|missing expected evidence|expected path was not present|expected file was not present' "+shellSingleQuote(docTarget)+" "+shellSingleQuote(manifestTarget),
		"Artifact-only recovery contract:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		"- Final action must be: write the enriched recovery overview document and shard-pack-manifest.json under write_root, then exit successfully.",
		"- Exit with code 0 only after shard-pack-manifest.json validates and referenced markdown is not seed-only/recovery fallback prose.",
		fmt.Sprintf(`- artifact_root (workspace-relative) = %q`, strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- exact authored document target = %q`, docTarget),
		fmt.Sprintf(`- exact manifest target = %q`, manifestTarget),
		fmt.Sprintf(`- read_context_roots = %q`, strings.Join(task.ReadContextRoots, ", ")),
		fmt.Sprintf(`- repo_scopes = %q`, strings.Join(task.RepoScopes, ", ")),
		fmt.Sprintf(`- path_scopes = %q`, strings.Join(task.PathScopes, ", ")),
		"COLLECT PAIR RECOVERY CHECKLIST:",
	)
	if len(evidencePaths) > 0 {
		lines = append(lines, "Bounded repository evidence candidates:")
		for _, rel := range evidencePaths {
			lines = append(lines, "- "+rel)
		}
	}
	lines = append(lines, compactCollectManifestValidationChecklist(strings.TrimSpace(task.ArtifactRoot))...)
	lines = append(lines,
		"- Use the embedded JSON above as the task-specific skeleton. Do not infer schema from prior reports/taskruns artifacts or raw logs.",
		"- This is provider-authored recovery; ACP will only validate artifacts and write the runtime-execution metadata.",
	)
	if detail := errorText(validationErr); detail != "" {
		lines = append(lines, fmt.Sprintf("- Previous collect artifact validation failure: %s", detail))
	}
	return strings.Join(lines, "\n")
}

func composeQwenToolFirstCollectPrompt(task acpruntime.Task, validationErr error) string {
	docRel := steppolicy.SuggestedCollectDocumentPath(task)
	docTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), filepath.FromSlash(docRel))
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), "shard-pack-manifest.json")
	return composeQwenToolFirstCollectPromptWithTargets(task, validationErr, docRel, docTarget, manifestTarget, repairEvidenceCandidates(task))
}

func composeQwenToolFirstCollectPromptWithTargets(task acpruntime.Task, validationErr error, docRel string, docTarget string, manifestTarget string, evidencePaths []string) string {
	lines := []string{
		"You are ACP runtime provider \"qwen-code\".",
		"QWEN COLLECT — READ THEN ATOMIC PAIR WRITE:",
		"- Do not plan, explain, use a todo, or generate a shell/Python/template writer.",
		"- Your next response block must be read_file tool calls for at most 4 listed evidence files and at most 4000 bytes per file.",
		"- After those tool results, compose BOTH final payloads before calling any write tool.",
		"- Your next assistant response must contain exactly two write_file tool calls in the same response block: markdown first and manifest second. Do not wait for the markdown tool result before issuing the manifest write_file call.",
		"- Do not run another read/preflight and do not emit thinking, prose, or a status note between the evidence results and the two write_file calls.",
		fmt.Sprintf("- Exact markdown target: %q.", docTarget),
		fmt.Sprintf("- Exact manifest target: %q.", manifestTarget),
		"- Read only concrete existing files under read_context_roots; do not inspect taskruns, raw logs, sibling shards, or source outside the listed roots.",
		"- Markdown must be at most 3000 characters of concise operator-facing architecture evidence: observed components/config/runtime behavior, concrete file references, and honest gaps. Do not mention repair, bounded reads, providers, or process mechanics.",
		"- Manifest JSON must be at most 6000 characters. Do not model every observed component.",
		"- Use exactly 1 document, 1-3 citations, 1-3 entities, 0-2 edges, exactly 1 finding, and exactly 1 question.",
		"MINIMUM MANIFEST SHAPE:",
		fmt.Sprintf("- Top level: version=1, run_id=%q, step_id=%q, shard_id=%q, domain_id=%q, agent_role=%q, artifact_root=%q, repo_scopes, path_scopes, summary, documents, citations, semantic.", strings.TrimSpace(task.RunID), strings.TrimSpace(task.StepID), strings.TrimSpace(task.ShardID), strings.TrimSpace(task.DomainID), strings.TrimSpace(task.AgentRole), strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf("- Use these exact JSON array values: repo_scopes=%s and path_scopes=%s. Both fields MUST remain arrays even when they contain exactly one value; never encode either field as a string.", jsonStringArrayLiteral(task.RepoScopes), jsonStringArrayLiteral(task.PathScopes)),
		fmt.Sprintf("- documents[0]: id, kind, title, path=%q, canonical_path=%q, topics, citation_ids.", filepath.ToSlash(docRel), steppolicy.CollectManifestCanonicalPath(task, docRel)),
		`- Every citation object must contain all five keys: {"id":"...","repo":"...","path":"...","claim_ids":["..."],"document_ids":[documents[0].id]}. document_ids is mandatory and non-empty; never omit it. Use unique ids and concrete files; every document citation_id exists; every citation document_id equals documents[0].id. No citation provenance or omitted ids.`,
		"- semantic.coverage: observed[], missing[], notes[]. semantic.questions: objects with id and text.",
		"- semantic.entities: objects with id, name, type, provenance. semantic.edges and semantic.findings use the same provenance shape.",
		"- Every provenance.kind must be exactly one of: observation, inference, assertion. Values such as document, report, source, observed, inferred, or asserted are invalid.",
		"- provenance: {kind, numeric confidence, evidence:[{repo,path}]}; every finding also needs id, severity, title, description.",
		"- Do not add alternate/legacy keys. Every evidence path must be an existing repository file, not a directory.",
		"- Before the two write_file calls, inspect every citation object in the composed manifest: if any citation lacks id, repo, path, non-empty claim_ids, or non-empty document_ids, rebuild the manifest payload before writing either file.",
		"- Stop after the same-response pair of write_file calls. Backend validation, not stdout claims, is the only success surface.",
		fmt.Sprintf("- write_root=%q", strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf("- read_context_roots=%q", strings.Join(task.ReadContextRoots, ", ")),
	}
	lines = append(lines, "CANONICAL SEMANTIC OBJECTS:")
	lines = append(lines, collectManifestCanonicalShapeBlock()...)
	if strings.TrimSpace(task.StepID) == "refresh.step1.collect" {
		lines = append(lines,
			"- Refresh collect requires at least one semantic question and at least three concrete coverage.missing items.",
		)
		if intent := boundedQwenCollectIntent(task.RefreshIntentContext, 600); intent != "" {
			lines = append(lines,
				"- Current source files and observed evidence are authoritative; this bounded refresh intent is a secondary hint only:",
				intent,
			)
		}
	}
	if len(evidencePaths) > 0 {
		lines = append(lines, "Repository evidence candidates (choose up to 4):")
		for i, rel := range evidencePaths {
			if i == 4 {
				break
			}
			lines = append(lines, "- "+rel)
		}
	}
	if detail := errorText(validationErr); detail != "" {
		lines = append(lines, "- Retry reason: "+detail)
	}
	return strings.Join(lines, "\n")
}

func jsonStringArrayLiteral(values []string) string {
	if values == nil {
		values = []string{}
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func boundedQwenCollectIntent(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if value == "" || maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "…"
}

func useCompactCollectPairRepairPrompt(validationErr error) bool {
	detail := strings.ToLower(errorText(validationErr))
	return strings.Contains(detail, "runtime_stalled_before_artifacts") ||
		(strings.Contains(detail, "repo evidence path") && strings.Contains(detail, "directory")) ||
		strings.Contains(detail, "process-contaminated") ||
		strings.Contains(detail, "process contaminated")
}

func composeCompactCollectArtifactPairRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error, docRel string, docTarget string, manifestTarget string, evidencePaths []string) string {
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in collect artifact pair focused recovery mode.", provider),
		"COLLECT PAIR WRITE-FIRST EVIDENCE REPAIR:",
		"- Compact live recovery path: write first, keep the command small, and do not spend the window on broad analysis.",
		"- This repair is not a bootstrap/fallback writer. Do not create a seed-only or recovery-summary pair.",
		"- Do not run a separate read-only preflight, print an evidence packet, or answer with a plan/status note before the writes.",
		"- Your next action must be one bounded filesystem command that reads allowed evidence and writes both exact targets before returning.",
		fmt.Sprintf("- Write exactly two files in the first command: %q and %q.", docTarget, manifestTarget),
		"- Write the markdown document first as operator-facing architecture evidence, then write shard-pack-manifest.json that references it.",
		"- If the authored document target already exists, rewrite it completely from observed evidence; do not leave stale missing-path or process wording in place.",
		"- Read at most 8 listed evidence candidates and at most the first 6000 bytes from each file.",
		"- If a claim is not supported by observed snippets, omit it or record a concrete coverage gap; do not abort for self-invented semantic checks.",
		"- Repository evidence in citations/provenance must be concrete existing files. Directories and missing paths are coverage gaps/questions, never evidence paths.",
		"- Do not write outside write_root, delete files, read raw logs, inspect sibling taskruns, or rely on prior reports/taskruns history.",
		"- Use python3 if scripting; keep the command mechanically simple and avoid generated source strings or nested quote tricks.",
		"FIRST COLLECT PAIR REPAIR WRITE-FIRST COMMAND:",
		"- Execute one filesystem command now. It must write the final markdown + manifest before any optional explanation.",
		"COMPACT MANIFEST FIELD CONTRACT:",
		fmt.Sprintf("- version=1, run_id=%q, step_id=%q, shard_id=%q, domain_id=%q, agent_role=%q.", strings.TrimSpace(task.RunID), strings.TrimSpace(task.StepID), strings.TrimSpace(task.ShardID), strings.TrimSpace(task.DomainID), strings.TrimSpace(task.AgentRole)),
		fmt.Sprintf("- artifact_root must be exactly %q.", strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf("- documents[0].path must be %q and must point to the authored markdown file under write_root.", filepath.ToSlash(docRel)),
		fmt.Sprintf("- documents[0].canonical_path must be exactly %q.", steppolicy.CollectManifestCanonicalPath(task, docRel)),
		"- documents[].canonical_path must never contain reports/taskruns, /staging/, write_root, artifact_root, raw runtime paths, or absolute filesystem paths.",
		"- documents[] may contain only id, kind, title, path, canonical_path, topics, citation_ids.",
		"- citations[] may contain only id, repo, path, claim_ids, document_ids; every cited path must be an existing file under the resolved repo root.",
		"- Every citations[] item must include a non-empty claim_ids array; empty claim_ids means the manifest is invalid.",
		"- Every citations[] item must include a non-empty document_ids array that references an id from documents[].id.",
		"- semantic must include coverage, questions, entities, edges, and findings with repo/path provenance; avoid repo/shard wrapper-only semantic output.",
		"- Do not add top-level claims, claim_map, validation, metadata, compatibility, schema, or alternate semantic wrappers.",
		"CANONICAL SEMANTIC SHAPE:",
	}
	lines = append(lines, collectManifestCanonicalShapeBlock()...)
	if focus := collectArtifactPairRepairValidationFocus(validationErr); len(focus) > 0 {
		lines = append(lines, "VALIDATION-SPECIFIC REPAIR FOCUS:")
		lines = append(lines, focus...)
	}
	lines = append(lines,
		"RECOVERY ACCEPTANCE REQUIREMENT:",
		"- Successful recovery output must not contain ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT, Recovery Bootstrap, Recovery Summary, Recovery Evidence Summary, seed-only collect recovery fallback, Additional provider enrichment should replace, Treat this as diagnostic evidence until, first bounded evidence read was attempted, initial artifact records only, or will be repaired with concrete.",
		"- Successful recovery output must not mention bounded read, bounded pass, guessed path, guessed file, guessed evidence, missing expected evidence, expected path was not present, expected file was not present, or recovery attempts as final artifact content.",
		"- A noop, zero-output, unchanged skeleton, generic owner-mapping-only manifest, or partially-written repair is terminal.",
		"FINAL SELF-CHECK COMMAND:",
		"test -s "+shellSingleQuote(docTarget)+" && test -s "+shellSingleQuote(manifestTarget)+" && ! grep -E 'ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT|Recovery Bootstrap|Recovery Summary|Recovery Evidence Summary|seed-only collect recovery fallback|Additional provider enrichment should replace|Treat this as diagnostic evidence until|first bounded evidence read was attempted|initial artifact records only|will be repaired with concrete|bounded read|bounded pass|guessed path|guessed file|guessed evidence|missing expected evidence|expected path was not present|expected file was not present' "+shellSingleQuote(docTarget)+" "+shellSingleQuote(manifestTarget),
		"Artifact-only recovery contract:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		"- Backend validation, not stdout claims, is the success surface.",
		fmt.Sprintf(`- artifact_root (workspace-relative) = %q`, strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- exact authored document target = %q`, docTarget),
		fmt.Sprintf(`- exact manifest target = %q`, manifestTarget),
		fmt.Sprintf(`- read_context_roots = %q`, strings.Join(task.ReadContextRoots, ", ")),
		fmt.Sprintf(`- repo_scopes = %q`, strings.Join(task.RepoScopes, ", ")),
		fmt.Sprintf(`- path_scopes = %q`, strings.Join(task.PathScopes, ", ")),
	)
	if len(evidencePaths) > 0 {
		lines = append(lines, "Bounded repository evidence candidates:")
		for _, rel := range evidencePaths {
			lines = append(lines, "- "+rel)
		}
	}
	lines = append(lines, compactCollectManifestValidationChecklist(strings.TrimSpace(task.ArtifactRoot))...)
	lines = append(lines,
		"- This is provider-authored recovery; ACP will only validate artifacts and write runtime-execution metadata.",
	)
	if detail := errorText(validationErr); detail != "" {
		lines = append(lines, fmt.Sprintf("- Previous collect artifact validation failure: %s", detail))
	}
	return strings.Join(lines, "\n")
}

func collectArtifactPairRepairValidationFocus(validationErr error) []string {
	detail := strings.ToLower(errorText(validationErr))
	if strings.TrimSpace(detail) == "" {
		return nil
	}
	lines := []string{}
	if strings.Contains(detail, "repo evidence path") && strings.Contains(detail, "directory") {
		lines = append(lines,
			"- Immediate repair objective: replace every directory-only citation/provenance repo path with concrete existing file paths before writing the manifest.",
			"- Start from the directory names mentioned by the validation failure, but cite only files discovered under those directories or listed evidence candidates.",
			"- Use file-level checks for every replacement path; if no concrete file supports a claim, remove that claim or record a coverage gap/question.",
			"- The rewritten markdown may mention a directory only as scope context, never as a cited/provenance evidence path.",
		)
	}
	if strings.Contains(detail, "process-contaminated") || strings.Contains(detail, "process contaminated") {
		lines = append(lines,
			"- Immediate repair objective: rewrite the existing process-contaminated markdown target with final operator-facing architecture evidence before writing the manifest.",
			"- Remove runtime/process language from the markdown: bounded reads, bounded passes, guessed paths/files/evidence, expected-missing path checks, recovery mechanics, and repair narration.",
			"- Do not keep the old markdown and only patch shard-pack-manifest.json; both the markdown document and manifest must be freshly rewritten from observed repository evidence.",
			"- The final markdown may describe unsupported evidence only as a concrete coverage gap, using repository/domain language rather than repair-process language.",
		)
	}
	if strings.Contains(detail, "runtime_stalled_before_artifacts") {
		lines = append(lines,
			"- Immediate repair objective: write the markdown document and shard-pack-manifest.json in the first filesystem command before any optional analysis.",
			"- Do not wait for broad repository exploration; use the listed candidates and record missing details as coverage gaps/questions.",
		)
	}
	return lines
}

func collectManifestCanonicalShapeBlock() []string {
	return []string{
		`- coverage.notes shape: "notes": ["observed config/runtime/deploy surface"]; a bare string is invalid.`,
		`- entity object shape: {"id":"svc.example","name":"Example","type":"service","provenance":{"kind":"observation","confidence":0.8,"evidence":[{"repo":"repo-name","path":"README.md"}]}}.`,
		`- entity forbidden fields: kind, repo, path, evidence at the entity top level. Use type and provenance.evidence[].`,
		`- edge object shape: {"id":"edge.example.depends","type":"depends_on","from":"svc.example","to":"store.example","provenance":{"kind":"observation","confidence":0.7,"evidence":[{"repo":"repo-name","path":"README.md"}]}}.`,
		`- edge forbidden fields: relation, source, target. Use type, from, and to.`,
		`- finding object shape: {"id":"finding.example.owner","severity":"medium","title":"Owner not confirmed","description":"Scoped evidence does not identify an owning team.","provenance":{"kind":"observation","confidence":0.6,"evidence":[{"repo":"repo-name","path":"README.md"}]}}.`,
		`- finding forbidden fields: summary, inference, evidence_citation_ids, confidence at the finding top level.`,
		`- provenance object shape is always kind + numeric confidence + evidence[]; provenance as direct {"repo":"...","path":"..."} is invalid.`,
	}
}

func collectArtifactPairRepairDocumentPath(task acpruntime.Task, validationErr error) string {
	staleDocs := authoredDocumentsMentioningMissingRepoEvidencePaths(task.WriteRoot, validationErr)
	for _, rel := range staleDocs {
		if strings.ToLower(filepath.Ext(rel)) == ".md" {
			return rel
		}
	}
	authoredDocs := authoredRepairDocuments(task.WriteRoot)
	for _, rel := range authoredDocs {
		if strings.ToLower(filepath.Ext(rel)) == ".md" {
			return rel
		}
	}
	return steppolicy.SuggestedCollectDocumentPath(task)
}

func authoredDocumentsMentioningMissingRepoEvidencePaths(writeRoot string, err error) []string {
	missingPaths := missingRepoEvidencePathsFromError(err)
	if len(missingPaths) == 0 {
		return nil
	}
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return nil
	}
	cleanRoot := filepath.Clean(writeRoot)
	if _, statErr := os.Stat(cleanRoot); statErr != nil {
		return nil
	}
	missing := map[string]struct{}{}
	for _, path := range missingPaths {
		path = strings.TrimSpace(path)
		if path != "" {
			missing[path] = struct{}{}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	docs := []string{}
	_ = filepath.WalkDir(cleanRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := string(raw)
		for missingPath := range missing {
			if strings.Contains(text, missingPath) {
				rel, relErr := filepath.Rel(cleanRoot, path)
				if relErr == nil && strings.TrimSpace(rel) != "" && rel != "." {
					docs = append(docs, filepath.ToSlash(rel))
				}
				return nil
			}
		}
		return nil
	})
	sort.Strings(docs)
	return docs
}

func missingRepoEvidencePathsFromError(err error) []string {
	if err == nil {
		return nil
	}
	const marker = `repo evidence path "`
	text := err.Error()
	paths := []string{}
	seen := map[string]struct{}{}
	for {
		idx := strings.Index(text, marker)
		if idx < 0 {
			break
		}
		text = text[idx+len(marker):]
		end := strings.Index(text, `"`)
		if end < 0 {
			break
		}
		path := strings.TrimSpace(text[:end])
		if path != "" {
			if _, ok := seen[path]; !ok {
				seen[path] = struct{}{}
				paths = append(paths, path)
			}
		}
		text = text[end+1:]
	}
	return paths
}

func overwriteCollectManifestRepairInstructions() []string {
	return []string{
		"COLLECT MANIFEST REPAIR INSTRUCTIONS:",
		"- Perform a bounded evidence pass over existing authored documents and listed repository evidence candidates inside the same first command that writes shard-pack-manifest.json.",
		"- Do not read, diff, or patch an existing invalid shard-pack-manifest.json; replace it after the evidence pass.",
		"- Do not stop after status prose such as \"I have enough evidence\" or \"I am replacing\"; that is a no-op repair unless shard-pack-manifest.json was already written.",
		"- Do not search the filesystem for schemas/*, docs/spec/*, examples, prior manifests, sibling shards, raw logs, or reports/taskruns history.",
		"- Do not rewrite authored markdown documents and do not create any file other than shard-pack-manifest.json.",
		"- Do not cite directory paths in citations/provenance; resolve them to concrete existing files or move them to coverage.missing/questions.",
		"- Syntax-valid JSON alone is not a valid repair; verify citation/provenance paths are existing files before exiting successfully.",
		"- Treat the embedded JSON as a schema guide only. Build semantic entities, edges, findings, coverage, questions, and citations from the authored docs and allowed repository evidence.",
		"- Do not add top-level claims/claim_map/metadata/validation/compatibility, and do not preserve placeholder tokens such as SHARD, <shard>, <claim>, TODO, or REPLACE_ME in any JSON string.",
		"- Do not collapse semantic output to repo/shard wrappers plus owner mapping; cite concrete components and relationships already described in the authored docs.",
		"- Validation, not stdout, is the success surface; do not claim validation unless the backend accepts the artifact.",
	}
}

func compactCollectManifestValidationChecklist(artifactRoot string) []string {
	return artifactquality.CompactCollectManifestValidationChecklist(artifactRoot)
}

func authoredRepairDocuments(writeRoot string) []string {
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return nil
	}
	cleanRoot := filepath.Clean(writeRoot)
	if _, err := os.Stat(cleanRoot); err != nil {
		return nil
	}
	docs := []string{}
	_ = filepath.WalkDir(cleanRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil {
			return nil
		}
		if path == cleanRoot || entry.IsDir() {
			return nil
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || name == "shard-pack-manifest.json" || name == "runtime-execution.json" {
			return nil
		}
		rel, relErr := filepath.Rel(cleanRoot, path)
		if relErr != nil || strings.TrimSpace(rel) == "" || rel == "." {
			return nil
		}
		docs = append(docs, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(docs)
	return docs
}

func repairEvidenceCandidates(task acpruntime.Task) []string {
	if len(task.PathScopes) == 0 || len(task.ReadContextRoots) == 0 {
		return nil
	}
	workspace := filepath.Clean(strings.TrimSpace(task.Workspace))
	writeRoot := filepath.Clean(strings.TrimSpace(task.WriteRoot))
	seen := map[string]struct{}{}
	candidates := []string{}
	add := func(path string, info os.FileInfo) {
		path = filepath.ToSlash(strings.TrimSpace(path))
		if path == "" {
			return
		}
		if !repairEvidenceCandidateAllowed(path, info) {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		candidates = append(candidates, path)
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
		for _, scope := range task.PathScopes {
			for _, rel := range repairEvidenceCandidatesForScope(cleanRoot, scope) {
				info, err := os.Stat(filepath.Join(cleanRoot, filepath.FromSlash(rel)))
				if err != nil {
					continue
				}
				add(rel, info)
			}
		}
	}
	sortRepairEvidenceCandidates(candidates)
	if len(candidates) > 18 {
		candidates = candidates[:18]
	}
	return candidates
}

func repairEvidenceCandidatesForScope(cleanRoot string, scope string) []string {
	scope = strings.Trim(strings.TrimSpace(scope), string(filepath.Separator))
	scope = strings.Trim(scope, "/")
	if scope == "" {
		return nil
	}
	target := filepath.Join(cleanRoot, filepath.FromSlash(scope))
	info, err := os.Stat(target)
	if err != nil {
		return nil
	}
	if !info.IsDir() {
		if repairEvidenceCandidateAllowed(scope, info) {
			return []string{filepath.ToSlash(scope)}
		}
		return nil
	}
	candidates := []string{}
	_ = filepath.WalkDir(target, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "node_modules" || name == ".venv" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(cleanRoot, path)
		if relErr != nil {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil || !repairEvidenceCandidateAllowed(rel, info) {
			return nil
		}
		candidates = append(candidates, filepath.ToSlash(rel))
		return nil
	})
	sortRepairEvidenceCandidates(candidates)
	if len(candidates) > 6 {
		candidates = candidates[:6]
	}
	return candidates
}

func sortRepairEvidenceCandidates(candidates []string) {
	sort.Slice(candidates, func(i, j int) bool {
		leftRank := repairEvidenceCandidateRank(candidates[i])
		rightRank := repairEvidenceCandidateRank(candidates[j])
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return candidates[i] < candidates[j]
	})
}

func repairEvidenceCandidateRank(path string) int {
	name := strings.ToLower(filepath.Base(path))
	switch name {
	case "readme.md", "readme.adoc", "readme.rst", "readme.txt", "agents.md", "claude.md", "contributing.md":
		return 0
	case "package.json", "pyproject.toml", "go.mod", "pom.xml", "build.gradle", "settings.gradle", "cargo.toml":
		return 1
	case "docker-compose.yml", "docker-compose.yaml", "docker-compose.base.yml", "docker-compose.dev.yml", "dockerfile", "dockerfile.node":
		return 2
	case ".env.example", "posthog.json", "app.json", "workspace.yaml", "pnpm-workspace.yaml":
		return 3
	case "tsconfig.json", "pytest.ini", "tach.toml", "turbo.json":
		return 4
	}
	if strings.HasPrefix(name, "dockerfile") || strings.HasPrefix(name, "docker-compose.") {
		return 5
	}
	if strings.HasSuffix(name, ".md") {
		return 6
	}
	if strings.HasPrefix(name, ".") {
		return 9
	}
	return 7
}

func repairEvidenceCandidateAllowed(path string, info os.FileInfo) bool {
	if info == nil || info.IsDir() {
		return false
	}
	const maxRepairEvidenceBytes = 96 * 1024
	if info.Size() > maxRepairEvidenceBytes {
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
