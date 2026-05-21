package steppolicy

import (
	"encoding/json"
	"fmt"
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
			`- Treat schemas/*, docs/spec/*, and the enforced prompt contract as the only manifest source of truth for asis-draft-manifest.json.`,
			`- Keep step_contract exactly "as_is"; null, empty, or alternate values are invalid.`,
			`- The first authored draft set must include asis-draft-manifest.json plus overview.md, summary.md, and architect-summary.md under draft_final_root.`,
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
			`- Treat schemas/*, docs/spec/*, and the enforced prompt contract as the only manifest source of truth for proposals-draft-manifest.json.`,
			`- Keep step_contract exactly "proposals"; null, empty, or alternate values are invalid.`,
			`- outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/*.`,
			`- Do NOT register legacy top-level fields: pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps.`,
		}, "\n")
	default:
		if strings.HasPrefix(strings.TrimSpace(stepID), "refresh.") {
			return `For refresh steps, keep unresolved gaps explicit in artifacts instead of inventing placeholder semantic payloads.`
		}
		return ""
	}
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
			lines = append(lines, fmt.Sprintf(`- Existing repo entrypoint hints (after the first collect artifact pair exists, read only these first when further evidence is needed): %s`, strings.Join(entrypointHints, ", ")))
		} else {
			lines = append(lines, fmt.Sprintf(`- Existing repo entrypoint hints (read only these first when relevant): %s`, strings.Join(entrypointHints, ", ")))
		}
	} else if isPromptHintedStep(task.StepID) {
		lines = append(lines, `- Repo entrypoint hints are limited to actually existing files; do not assume README.md exists when it is absent.`)
	}
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		lines = append(lines,
			`- Do NOT delegate to agent/subagent helpers and do NOT use todo_write-style planning.`,
			`- Write constitution-draft.json in write_root.`,
			`- Write the referenced draft files exactly at draft_final_root/charter-overview.md and draft_final_root/baseline-subagents.yaml.`,
			`- Do NOT place the draft files under draft_final_root/charter/ or draft_final_root/skills/; those are canonical publish paths, not draft file locations.`,
			`- constitution-draft.json must use the exact runtime draft manifest shape shown below; do not emit legacy constitution schemas.`,
			`- outputs[] must map charter-overview.md -> charter/overview.md and baseline-subagents.yaml -> skills/subagents.yaml exactly.`,
			`- Exact constitution-draft.json example (replace IDs/summary only, keep keys/types and output mapping):`,
			ConstitutionDraftManifestExample(task),
			`- Keep the draft deterministic in shape; compiler will normalize/publish canonical files afterwards.`,
		)
	case "init.step1.collect", "refresh.step1.collect":
		lines = append(lines,
			`- Do NOT delegate to agent/subagent helpers and do NOT use todo_write-style planning.`,
			`- Before the first filesystem write inside write_root, keep repository exploration minimal and converge quickly on the first authored doc plus shard-pack-manifest.json.`,
			`- Produce runtime-authored documents in write_root and then write shard-pack-manifest.json in write_root.`,
			`- Early pair-write requirement: write the suggested overview doc and shard-pack-manifest.json as one focused artifact pair before any broad second-pass repository sweep.`,
			fmt.Sprintf(`- Suggested collect authored doc path for this shard: %q. Prefer exactly this single doc path unless already writing an existing clearer authored doc.`, SuggestedCollectDocumentPath(task)),
			fmt.Sprintf(`- Absolute collect targets for the early pair-write: %q and %q.`, filepath.Join(strings.TrimSpace(task.WriteRoot), filepath.FromSlash(SuggestedCollectDocumentPath(task))), filepath.Join(strings.TrimSpace(task.WriteRoot), "shard-pack-manifest.json")),
			fmt.Sprintf(`- Minimal collect target shape: write %q + "shard-pack-manifest.json" early, then record remaining uncertainty in coverage/questions instead of continuing open-ended exploration.`, SuggestedCollectDocumentPath(task)),
			`- Do not wait for a complete broad repository sweep before writing shard-pack-manifest.json; once the first authored doc covers the assigned shard scope, write the manifest and record remaining gaps in semantic.coverage.missing.`,
			`- Immediately after writing the first authored doc, write shard-pack-manifest.json by adapting the task-specific JSON skeleton embedded in the first-action command section; keep exact metadata keys and replace only evidence/content values you actually observed.`,
			`- Do not exit after writing markdown only; every collect shard must finish with a valid shard-pack-manifest.json.`,
			`- shard-pack-manifest.json must describe every authored document, its canonical stable path, citations, and semantic snapshot.`,
			`- In shard-pack-manifest.json, semantic MUST include coverage, questions, entities, edges, and findings.`,
			`- Use only canonical collect vocabulary: semantic.coverage.observed, semantic.questions[*].id + semantic.questions[*].text, semantic.edges[*].type, and object-shaped provenance blocks.`,
			`- Every semantic.questions[] item must include id and text; every semantic.findings[] item must include id, severity, title, and provenance.`,
			`- Do NOT emit semantic payloads on stdout; keep semantic only inside shard-pack-manifest.json.`,
			`- You may be flexible in document structure, but promotion and rendering depend on manifest citations/topics remaining accurate.`,
			`- After the first filesystem write inside write_root, stop broad repository exploration; only minimal manifest/JSON repair is allowed afterwards.`,
			`- After writing shard-pack-manifest.json, do NOT continue broad list_directory/read_file sweeps across repo roots.`,
			`- Do NOT read reports/taskruns/**, raw runtime logs, or previously generated shard-pack-manifest.json files as schema examples during collect.`,
			`- If authored docs and shard-pack-manifest.json already exist in write_root, stop and exit successfully.`,
		)
		if rootFileScopes := rootFileShardPathScopes(task.PathScopes); len(rootFileScopes) > 0 {
			lines = append(lines,
				fmt.Sprintf(`- Root-file collect shard detected: path_scopes contains root-level files only: %s.`, strings.Join(rootFileScopes, ", ")),
				`- For this root-file shard, read only the listed root files first; do not recursively sweep top-level directories or unrelated source trees.`,
				`- Produce one concise root overview document in write_root, then write shard-pack-manifest.json for that document and exit successfully.`,
			)
		}
		lines = append(lines,
			`TASK-SPECIFIC COLLECT MANIFEST JSON SKELETON: use the heredoc JSON embedded in the first-action command section above; do not copy a generic manifest example.`,
			`COLLECT MANIFEST CONTRACT CHECKLIST:`,
		)
		lines = append(lines, artifactquality.CollectManifestContractLines(strings.TrimSpace(task.ArtifactRoot))...)
		lines = append(lines,
			`- The task-specific collect manifest JSON skeleton above is normative for field names and value types; copy that skeleton, not a generic example.`,
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
			`- Use the FIRST AS-IS DRAFT COMMAND skeleton above as the first draft artifact set; do not wait for broad analysis before creating the required files.`,
			`- Use staged final evidence from read_context_roots only; do NOT read sibling baseline workspaces or previously published as-is drafts as templates.`,
			`- If asis-draft-manifest.json already describes the publish surface, stop after artifact validation; do NOT re-register draft artifacts through any legacy metadata op.`,
			`- Compiler may materialize indexes and derived technical artifacts only; canonical narratives come from your drafts.`,
		)
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
			`- If proposals-draft-manifest.json already describes the publish surface, stop after artifact validation; do NOT re-register draft artifacts through any legacy metadata op.`,
			`- Promotion remains deterministic; your drafts become publish candidates only after compile/publish gates.`,
		)
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

func CollectEarlyPairWriteCommand(task acpruntime.Task) string {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	docRel := SuggestedCollectDocumentPath(task)
	docTarget := filepath.Join(writeRoot, filepath.FromSlash(docRel))
	manifestTarget := filepath.Join(writeRoot, "shard-pack-manifest.json")
	skeleton := CollectManifestTaskSkeleton(task, []string{docRel}, nil)
	return strings.Join([]string{
		"mkdir -p " + shellSingleQuote(writeRoot),
		"cat > " + shellSingleQuote(docTarget) + " <<'ACP_COLLECT_DOC'",
		collectDocumentInitialTemplate(task, docRel),
		"ACP_COLLECT_DOC",
		"cat > " + shellSingleQuote(manifestTarget) + " <<'ACP_MANIFEST_JSON'",
		strings.TrimSpace(skeleton),
		"ACP_MANIFEST_JSON",
	}, "\n")
}

func CollectFirstActionSection(task acpruntime.Task) string {
	if !acpruntime.IsCollectStep(task.StepID) {
		return ""
	}
	writeRoot := strings.TrimSpace(task.WriteRoot)
	docTarget := filepath.Join(writeRoot, filepath.FromSlash(SuggestedCollectDocumentPath(task)))
	manifestTarget := filepath.Join(writeRoot, "shard-pack-manifest.json")
	return strings.Join([]string{
		"COLLECT FIRST-ACTION ARTIFACT PAIR:",
		"- This collect step must start by writing the task-specific artifact pair before broad repository instructions.",
		fmt.Sprintf(`- Exact authored document target: %q.`, docTarget),
		fmt.Sprintf(`- Exact manifest target: %q.`, manifestTarget),
		"FIRST COLLECT ARTIFACT PAIR COMMAND:",
		"Run this exact command as the next filesystem action after checking whether both target files already exist; do not call read_file, list_directory, grep_search, glob, find, rg, or any repository exploration before this command:",
		"- The embedded skeleton is intentionally valid before additional evidence; do not improve or rewrite it before the first pair exists.",
		CollectEarlyPairWriteCommand(task),
	}, "\n")
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
	return strings.Join([]string{
		"AS-IS FIRST-ACTION DRAFT ARTIFACTS:",
		"- This as-is step must start by writing asis-draft-manifest.json and its referenced draft files before broad as-is assembly.",
		fmt.Sprintf(`- Exact as-is draft manifest target: %q.`, manifestTarget),
		fmt.Sprintf(`- Draft files must be written only under draft_final_root: %q.`, strings.TrimSpace(task.DraftFinalRoot)),
		"FIRST AS-IS DRAFT COMMAND:",
		"Run this exact shell command as the next filesystem action; do not manually retype paths, rewrite slash-separated path components, inspect sibling taskruns, prior reports, or raw logs before this command:",
		RuntimeDraftFirstActionWriteCommand(task),
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
	return strings.Join([]string{
		"PROPOSALS FIRST-ACTION DRAFT ARTIFACTS:",
		"- This proposals step must start by writing proposals-draft-manifest.json and its referenced draft files before broad proposal analysis.",
		fmt.Sprintf(`- Exact proposals draft manifest target: %q.`, manifestTarget),
		fmt.Sprintf(`- Draft files must be written only under draft_final_root: %q.`, strings.TrimSpace(task.DraftFinalRoot)),
		"FIRST PROPOSALS DRAFT COMMAND:",
		"Run this exact shell command as the next filesystem action; do not manually retype paths, rewrite slash-separated path components, inspect repository files, inspect sibling taskruns, or inspect raw logs before this command:",
		RuntimeDraftFirstActionWriteCommand(task),
	}, "\n")
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
	case "proposals/runtime-recommendations.md":
		return strings.Join([]string{
			"# Runtime Recommendations",
			"",
			"## Summary",
			"- Provider wrote the required proposals draft artifact set for this run.",
			"",
			"## Recommendation",
			"- Review collected evidence, validator findings, and coverage gaps before promotion.",
		}, "\n")
	case "reports/changelog/runtime-proposals.md":
		return strings.Join([]string{
			"# Runtime Proposal Changelog",
			"",
			"## Changes",
			"- Provider wrote the required proposal draft surface.",
			"",
			"## Notes",
			"- Promote only after artifact validation succeeds.",
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
			"- Provider wrote this draft artifact under the required draft_final_root.",
		}, "\n")
	}
}

func collectDocumentInitialTemplate(task acpruntime.Task, docRel string) string {
	repo := PrimaryTaskRepoScope(task.RepoScope, task.RepoScopes)
	if repo == "" {
		repo = "repo"
	}
	evidencePath := collectEvidencePath(task, nil)
	if evidencePath == "" {
		evidencePath = "README.md"
	}
	scopeText := strings.Join(nonEmptyList(task.PathScopes), ", ")
	if scopeText == "" {
		scopeText = evidencePath
	}
	titleSlug := slugComponent(strings.TrimSuffix(filepath.Base(docRel), filepath.Ext(docRel)))
	title := titleFromSlug(titleSlug)
	return strings.Join([]string{
		"# " + title,
		"",
		"## Scope",
		"- Repository: " + repo,
		"- Path scopes: " + scopeText,
		"",
		"## Observations",
		"- Repository scope: " + repo + ".",
		"- Primary scoped evidence path: `" + evidencePath + "`.",
		"",
		"## Evidence",
		"- Primary evidence path: `" + evidencePath + "`",
		"",
		"## Follow-up",
		"- Owner mapping evidence not confirmed from the initial scoped evidence path.",
		"",
	}, "\n")
}

func CollectManifestTaskSkeleton(task acpruntime.Task, docPaths []string, evidencePaths []string) string {
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
			CanonicalPath: fmt.Sprintf("reports/as-is/%s/%s.md", shardSlug, docSlug),
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
	questions := collectQuestionsSkeleton(task, idStem, topic)
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
		Semantic: contracts.SemanticSnapshot{
			Coverage: contracts.Coverage{
				Observed: []string{topic},
				Missing:  coverageMissing,
				Notes:    []string{"Collect manifest covers the assigned shard scope with evidence paths listed in citations."},
			},
			Questions: questions,
			Entities:  []contracts.Entity{},
			Edges:     []contracts.Edge{},
			Findings:  []contracts.Finding{},
		},
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
		Findings:     []contracts.Finding{},
		Questions:    []contracts.Question{},
		Issues:       []contracts.ValidatorIssue{},
	}
	raw, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		return `{"version":1,"run_id":"run-1","generated_at":"2026-04-16T12:00:02Z","verdict":"PASS","checked_paths":["reports/taskruns/run-1/staging/final/final-run-index.json"],"fixed_paths":[],"findings":[],"questions":[],"issues":[]}`
	}
	return string(raw)
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
		`- documents[].path MUST stay relative to artifact_root only; valid example: "iac-overview.md". Invalid examples: "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac/iac-overview.md", "charter/overview.md".`,
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
	default:
		return ""
	}
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
