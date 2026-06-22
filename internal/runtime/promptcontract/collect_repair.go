package promptcontract

import (
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

func ComposeCollectManifestRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, _ error) string {
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
		"- JSON syntax-only checks such as jq empty or python3 -m json.tool are insufficient; the first command must also prove every citation/provenance repo path is an existing file.",
		fmt.Sprintf("- Write exactly one file: %q.", manifestTarget),
		"- Do not rewrite existing authored markdown documents.",
		"- If shard-pack-manifest.json already exists but is invalid, do not inspect or patch it; overwrite it after the evidence pass.",
		"- The manifest top-level object may contain only ACP shard-pack-manifest fields: version, run_id, step_id, shard_id, domain_id, agent_role, artifact_root, repo_scopes, path_scopes, summary, documents, citations, and semantic.",
		"- Do not add top-level claims, claim_map, validation, metadata, compatibility, schema, or any alternate semantic wrapper; claim IDs belong only in citations[].claim_ids.",
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
	}
	if len(evidencePaths) > 0 {
		repairLines = append(repairLines, "Repository evidence candidates available for bounded repair:")
		for _, rel := range evidencePaths {
			repairLines = append(repairLines, fmt.Sprintf("- %s", rel))
		}
	}
	repairLines = append(repairLines, overwriteCollectManifestRepairInstructions()...)
	repairLines = append(repairLines,
		"COLLECT MANIFEST REPAIR CHECKLIST:",
	)
	repairLines = append(repairLines, compactCollectManifestValidationChecklist(strings.TrimSpace(task.ArtifactRoot))...)
	repairLines = append(repairLines,
		`- Repair-mode note: if schemas/* or docs/spec/* are absent from the runtime workspace, do not look for them; use this embedded checklist.`,
		`- The task-specific JSON skeleton above is a schema guide only; copied scaffold semantic is invalid even when the JSON schema passes.`,
	)
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
		"  - write the final provider-authored manifest to " + shellSingleQuote(manifestTarget) + " before returning;",
		"  - run a local `test -s`/JSON parse check plus file-level evidence path checks after the write.",
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
		"- Do not cite repository directories. If a candidate is directory-only or missing, record it in coverage.missing/questions instead of citations/provenance.",
		"- Use python3, not python; do not use GNU-only find -printf; do not assign to the zsh-reserved status variable.",
		"- Do not use a copied skeleton or generic repo/shard wrapper semantic as the final manifest.",
	)
	return strings.Join(lines, "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func ComposeCollectArtifactPairRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error) string {
	docRel := collectArtifactPairRepairDocumentPath(task, validationErr)
	docTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), filepath.FromSlash(docRel))
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), "shard-pack-manifest.json")
	evidencePaths := repairEvidenceCandidates(task)
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
		"- Do not add top-level claims, claim_map, validation, metadata, compatibility, schema, or any alternate semantic wrapper; claim IDs belong only in citations[].claim_ids.",
		"- Do not leave template placeholders in JSON strings. Literal SHARD, <shard>, <claim>, TODO, REPLACE_ME, or quoted suffix fragments such as \"claim.foo.\"SHARD are invalid.",
		"- Use repo/path provenance for every semantic evidence object; stdout claims are diagnostics only.",
		"- Every citation/provenance path must be a concrete existing repository file; directories and missing paths are coverage gaps/questions, not evidence.",
		"- JSON syntax-only checks such as jq empty or python3 -m json.tool are insufficient; include file-level citation/provenance checks before a successful exit.",
		"- Use python3, not python; do not use GNU-only find -printf; do not assign to the zsh-reserved status variable.",
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
		"test -s " + shellSingleQuote(docTarget) + " && test -s " + shellSingleQuote(manifestTarget) + " && ! grep -E 'ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT|Recovery Bootstrap|Recovery Summary|Recovery Evidence Summary|seed-only collect recovery fallback|Additional provider enrichment should replace|Treat this as diagnostic evidence until|first bounded evidence read was attempted|initial artifact records only|will be repaired with concrete|bounded read|bounded pass|guessed path|guessed file|guessed evidence|missing expected evidence|expected path was not present|expected file was not present' " + shellSingleQuote(docTarget) + " " + shellSingleQuote(manifestTarget),
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
	}
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
