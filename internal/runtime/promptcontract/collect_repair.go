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

func ComposeCollectManifestRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, _ error) string {
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), "shard-pack-manifest.json")
	authoredDocs := authoredRepairDocuments(task.WriteRoot)
	evidencePaths := repairEvidenceCandidates(task)
	skeleton := steppolicy.CollectManifestTaskSkeleton(task, authoredDocs, evidencePaths)
	firstCommand := collectManifestRepairPreflightCommand(task, authoredDocs, evidencePaths)
	sections := []string{
		fmt.Sprintf("You are ACP runtime provider %q in collect manifest repair mode.", provider),
		"COLLECT MANIFEST EVIDENCE-FIRST REPAIR:",
		"- Repair shard-pack-manifest.json from the existing authored documents and bounded repository evidence; do not start with a placeholder scaffold.",
		"- The first command below is a read-only evidence preflight; it reads existing authored documents in write_root and prints a bounded summary, but it does not write shard-pack-manifest.json.",
		"- After the preflight, you must author shard-pack-manifest.json yourself from the observed markdown and allowed repository evidence; no deterministic helper writes the manifest for you.",
		"- Read only the listed repository evidence candidates if authored docs need support; do not start an open-ended repository sweep.",
		fmt.Sprintf("- Write exactly one file: %q.", manifestTarget),
		"- Do not rewrite existing authored markdown documents.",
		"- If shard-pack-manifest.json already exists but is invalid, do not inspect or patch it; overwrite it after the evidence pass.",
		"FIRST COLLECT MANIFEST REPAIR COMMAND:",
		"- Run this exact command as your next filesystem action. Do not manually retype paths, inspect sibling taskruns, read raw logs, or write any other file before this command.",
		"- The command is read-only: it reads authored markdown already in write_root, verifies there is material to repair from, and prints the exact bounded evidence surface.",
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

func collectManifestRepairPreflightCommand(task acpruntime.Task, authoredDocs []string, evidencePaths []string) string {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		writeRoot = "."
	}
	payload := map[string]any{
		"write_root":     writeRoot,
		"run_id":         firstNonEmpty(task.RunID, "run-1"),
		"step_id":        firstNonEmpty(task.StepID, "init.step1.collect"),
		"shard_id":       firstNonEmpty(task.ShardID, task.DomainID, "shard"),
		"domain_id":      strings.TrimSpace(task.DomainID),
		"agent_role":     firstNonEmpty(task.AgentRole, "shard-analyst"),
		"artifact_root":  strings.TrimSpace(task.ArtifactRoot),
		"repo_scopes":    nonEmptyList(task.RepoScopes),
		"path_scopes":    nonEmptyList(task.PathScopes),
		"authored_docs":  nonEmptyList(authoredDocs),
		"evidence_paths": nonEmptyList(evidencePaths),
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		rawPayload = []byte("{}")
	}
	return strings.Join([]string{
		"python3 - " + shellSingleQuote(string(rawPayload)) + " <<'ACP_COLLECT_MANIFEST_REPAIR_PY'",
		"import json, pathlib, sys",
		"",
		"meta = json.loads(sys.argv[1])",
		"write_root = pathlib.Path(meta.get('write_root') or '.').resolve()",
		"target = write_root / 'shard-pack-manifest.json'",
		"",
		"def clean_rel(value):",
		"    value = str(value or '').replace('\\\\', '/').strip().strip('/')",
		"    if not value or value in ('.', '..') or value.startswith('../') or '/..' in value:",
		"        return ''",
		"    return value",
		"",
		"doc_paths = []",
		"for rel in meta.get('authored_docs') or []:",
		"    rel = clean_rel(rel)",
		"    if rel and rel not in doc_paths and (write_root / rel).is_file():",
		"        doc_paths.append(rel)",
		"if not doc_paths:",
		"    for path in sorted(write_root.rglob('*.md')):",
		"        rel = clean_rel(path.relative_to(write_root).as_posix())",
		"        if rel and path.name not in ('runtime-execution.json', 'shard-pack-manifest.json'):",
		"            doc_paths.append(rel)",
		"",
		"docs = []",
		"for rel in doc_paths:",
		"    text = (write_root / rel).read_text(encoding='utf-8', errors='replace')",
		"    if text.strip():",
		"        docs.append({'path': rel, 'bytes': len(text.encode('utf-8')), 'preview': ' '.join(text.split())[:280]})",
		"if not docs:",
		"    raise SystemExit('collect manifest repair found no authored markdown documents')",
		"",
		"evidence_paths = [clean_rel(p) for p in (meta.get('evidence_paths') or meta.get('path_scopes') or ['README.md'])]",
		"evidence_paths = [p for p in evidence_paths if p] or ['README.md']",
		"summary = {",
		"    'mode': 'collect_manifest_repair_preflight',",
		"    'target': target.as_posix(),",
		"    'writes_manifest': False,",
		"    'authored_doc_count': len(docs),",
		"    'authored_docs': docs[:12],",
		"    'evidence_paths': evidence_paths[:12],",
		"    'repo_scopes': meta.get('repo_scopes') or [],",
		"    'path_scopes': meta.get('path_scopes') or [],",
		"}",
		"print(json.dumps(summary, indent=2, ensure_ascii=False))",
		"ACP_COLLECT_MANIFEST_REPAIR_PY",
		"test -d " + shellSingleQuote(writeRoot),
	}, "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func nonEmptyList(values []string) []string {
	out := []string{}
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func ComposeCollectArtifactPairRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error) string {
	docRel := steppolicy.SuggestedCollectDocumentPath(task)
	docTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), filepath.FromSlash(docRel))
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), "shard-pack-manifest.json")
	evidencePaths := repairEvidenceCandidates(task)
	skeleton := steppolicy.CollectManifestTaskSkeleton(task, []string{docRel}, evidencePaths)
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in collect artifact pair focused recovery mode.", provider),
		"COLLECT PAIR EVIDENCE-FIRST REPAIR:",
		"- This repair is not a bootstrap/fallback writer. Do not create a seed-only or recovery-summary pair.",
		"- First read bounded repository evidence from the exact read_context_roots/path_scopes below, then write final evidence-backed artifacts.",
		"- Write the markdown document first as concise evidence-backed content, then write the manifest that references it; do not add optional enrichment until both exact targets are non-empty and marker-free.",
		"- After writing the markdown document, do not read another repository file; your next filesystem action must write shard-pack-manifest.json, then run the final self-check.",
		fmt.Sprintf("- Write exactly two files after the evidence pass: %q and %q.", docTarget, manifestTarget),
		"- Do not write any file outside the exact write_root collect pair.",
		"- Do not delete existing files, run rm -f, use git rev-parse, rely on cwd discovery, inspect sibling reports/taskruns, or read raw logs.",
		"- If bounded repo evidence cannot be read, exit non-zero without writing fallback scaffold.",
		"FIRST COLLECT PAIR REPAIR WORKFLOW:",
		"- Read only the listed evidence candidates below using the exact absolute read_context_roots; do not open unlisted path_scopes after this bounded pass.",
		"- Do not read lockfiles, generated baselines, test duration indexes, raw logs, reports/taskruns history, or files larger than 96 KiB during collect pair repair.",
		"- Explain concrete components, runtime/config/deploy/test/data surfaces, ownership gaps, and dependencies in the markdown document.",
		"- Build shard-pack-manifest.json from that markdown plus the same repo/path evidence: documents, citations, semantic.coverage, semantic.questions, semantic.entities, semantic.edges, and semantic.findings.",
		"- Use repo/path provenance for every semantic evidence object; stdout claims are diagnostics only.",
		"TASK-SPECIFIC MANIFEST JSON SKELETON:",
		strings.TrimSpace(skeleton),
		"SKELETON USE:",
		"- Use this JSON only as the task-specific schema/key/type guide, not as final content.",
		"- Replace skeleton citations, coverage, questions, entities, edges, findings, titles, and descriptions with facts from allowed repository evidence.",
		"- Copying this skeleton unchanged is invalid and will be rejected as scaffold-only output.",
		"RECOVERY ACCEPTANCE REQUIREMENT:",
		"- Successful recovery output must not contain ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT, Recovery Bootstrap/Recovery Summary scaffold text, Recovery Evidence Summary fallback prose, or the normal collect first-action scaffold prose.",
		"- Successful recovery output must not say seed-only collect recovery fallback, Additional provider enrichment should replace, or Treat this as diagnostic evidence until.",
		"- A manifest with many citations but only repo/shard wrapper entities, only contains edges, or only generic owner-mapping findings is invalid.",
		"- Exit successfully only after both exact targets are complete, marker-free, and evidence-backed.",
		"- A noop, zero-output, unchanged skeleton, or partially-written repair is terminal; exit non-zero instead of claiming success when either exact target is missing or unchanged.",
		"FINAL SELF-CHECK COMMAND:",
		"test -s " + shellSingleQuote(docTarget) + " && test -s " + shellSingleQuote(manifestTarget) + " && ! grep -E 'ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT|Recovery Bootstrap|Recovery Summary|Recovery Evidence Summary|seed-only collect recovery fallback|Additional provider enrichment should replace|Treat this as diagnostic evidence until' " + shellSingleQuote(docTarget) + " " + shellSingleQuote(manifestTarget),
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

func overwriteCollectManifestRepairInstructions() []string {
	return []string{
		"COLLECT MANIFEST REPAIR INSTRUCTIONS:",
		"- Perform a bounded evidence pass over existing authored documents and listed repository evidence candidates before writing shard-pack-manifest.json.",
		"- Do not read, diff, or patch an existing invalid shard-pack-manifest.json; replace it after the evidence pass.",
		"- Do not search the filesystem for schemas/*, docs/spec/*, examples, prior manifests, sibling shards, raw logs, or reports/taskruns history.",
		"- Do not rewrite authored markdown documents and do not create any file other than shard-pack-manifest.json.",
		"- Treat the embedded JSON as a schema guide only. Build semantic entities, edges, findings, coverage, questions, and citations from the authored docs and allowed repository evidence.",
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
			scope = strings.Trim(strings.TrimSpace(scope), string(filepath.Separator))
			scope = strings.Trim(scope, "/")
			if scope == "" {
				continue
			}
			target := filepath.Join(cleanRoot, filepath.FromSlash(scope))
			info, err := os.Stat(target)
			if err != nil {
				continue
			}
			if !info.IsDir() {
				add(scope, info)
				continue
			}
			_ = filepath.WalkDir(target, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if len(candidates) >= 24 {
					return filepath.SkipAll
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
				if infoErr != nil {
					return nil
				}
				add(rel, info)
				return nil
			})
		}
	}
	sort.Strings(candidates)
	return candidates
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
