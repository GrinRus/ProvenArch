package steppolicy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
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
			`- Treat schemas/*, docs/spec/*, and the enforced prompt contract as the only schema source of truth for shard-pack-manifest.json.`,
			`- Do NOT inspect reports/taskruns, prior shard-pack-manifest.json files, raw logs, or archive docs to infer collect manifest shape.`,
			`- Every semantic provenance.evidence[] item must include non-empty repo and path values that resolve to real repository evidence.`,
			`- Citation-only semantic evidence objects such as {"citation_id":"..."} are forbidden.`,
			`- semantic.findings[*] must use title + description + provenance; do NOT use summary as a compatibility alias.`,
		}, "\n")
	case "refresh.step1.collect":
		return strings.Join([]string{
			`STEP POLICY refresh.step1.collect:`,
			`- Do NOT delegate to agent/subagent helpers.`,
			`- Do NOT use todo_write-style planning or long plan narration.`,
			`- Use only existing repo entrypoint hints; do not assume README.md exists when it is not present.`,
			`- After the first useful evidence pass, converge to authored docs plus shard-pack-manifest.json instead of continuing a broad repo sweep.`,
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
			`- Do NOT register legacy metadata envelopes or repo_scopes/path_scopes fields in asis-draft-manifest.json.`,
		}, "\n")
	case "init.step3.findings", "refresh.step3.findings":
		return strings.Join([]string{
			fmt.Sprintf(`STEP POLICY %s:`, strings.TrimSpace(stepID)),
			`- Write validator-verdict.json only; do not write shard manifests or semantic snapshots for this step.`,
			`- validator-verdict.json must include version=1, run_id, generated_at, verdict, and checked_paths.`,
			`- findings[] items must use title + description + provenance; do NOT use summary as a finding alias.`,
			`- For observation provenance, findings[*].provenance.evidence[] must be non-empty and each evidence item must include repo/path.`,
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
		fmt.Sprintf(`- artifact_root (workspace-relative) = %q`, strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- draft_final_root (absolute) = %q`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- read_context_roots = %s`, readContextRootsJSON),
		fmt.Sprintf(`- domain_id = %q`, strings.TrimSpace(task.DomainID)),
		fmt.Sprintf(`- agent_role = %q`, strings.TrimSpace(task.AgentRole)),
		fmt.Sprintf(`- step_contract = %q`, strings.TrimSpace(task.StepContract)),
		fmt.Sprintf(`- expected_artifacts = %s`, strings.Join(task.ExpectedArtifacts, ", ")),
	}
	if entrypointHints := CollectRepoEntrypointHints(task); len(entrypointHints) > 0 {
		lines = append(lines, fmt.Sprintf(`- Existing repo entrypoint hints (read only these first when relevant): %s`, strings.Join(entrypointHints, ", ")))
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
			`- shard-pack-manifest.json must describe every authored document, its canonical stable path, citations, and semantic snapshot.`,
			`- In shard-pack-manifest.json, semantic MUST include coverage, questions, entities, edges, and findings.`,
			`- Use only canonical collect vocabulary: semantic.coverage.observed, semantic.questions[*].text, semantic.edges[*].type, and object-shaped provenance blocks.`,
			`- Do NOT emit semantic payloads on stdout; keep semantic only inside shard-pack-manifest.json.`,
			`- You may be flexible in document structure, but promotion and rendering depend on manifest citations/topics remaining accurate.`,
			`- After the first filesystem write inside write_root, stop broad repository exploration; only minimal manifest/JSON repair is allowed afterwards.`,
			`- After writing shard-pack-manifest.json, do NOT continue broad list_directory/read_file sweeps across repo roots.`,
			`- Do NOT read reports/taskruns/**, raw runtime logs, or previously generated shard-pack-manifest.json files as schema examples during collect.`,
			`- If authored docs and shard-pack-manifest.json already exist in write_root, stop and exit successfully.`,
		)
		lines = append(lines, artifactquality.CollectManifestContractLines(strings.TrimSpace(task.ArtifactRoot))...)
		lines = append(lines,
			`- Canonical collect fragment below is normative for field names and value types; do not substitute legacy aliases.`,
			artifactquality.CollectManifestCanonicalExample(),
		)
		lines = append(lines, artifactquality.ClaimIDContractLines()...)
		lines = append(lines,
			`- Do NOT collapse a multi-document refresh surface to one generic "cite.runtime-summary" citation when repository evidence exists.`,
			`- Preserve repo-specific citations in shard-pack-manifest.json whenever repository files support them.`,
		)
	case "init.step3.findings", "refresh.step3.findings":
		lines = append(lines,
			`- Inspect staged final artifacts under reports/taskruns/<run_id>/staging/final from read_context_roots.`,
			`- Write validator-verdict.json in write_root.`,
			`- Validator may fix only indexes, references, or technical document issues inside write_root; do not rewrite document meaning wholesale.`,
			`- Do NOT shard this step and do NOT emit findings through stdout; validator-verdict.json is the only primary output.`,
		)
		lines = append(lines, artifactquality.ValidatorVerdictContractLines()...)
		lines = append(lines,
			`- Canonical validator-verdict fragment below is normative for metadata fields and finding evidence shape; copy keys/types exactly and only change IDs/content.`,
			artifactquality.ValidatorVerdictCanonicalExample(),
		)
		lines = append(lines, artifactquality.ClaimIDContractLines()...)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines,
			`- Write asis-draft-manifest.json in write_root.`,
			`- Draft final docs only under draft_final_root.`,
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
	if strings.Contains(parseErr.Error(), `additionalProperties 'compatibility' not allowed`) {
		lines = append(lines, `- Replace any legacy "compatibility" block with the canonical "semantic" block in shard-pack-manifest.json.`)
		lines = append(lines, "- "+TopLevelSemanticOutputRule(stepID))
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
		`- documents[].path MUST stay relative to artifact_root only; valid example: "iac-overview.md". Invalid examples: "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac/iac-overview.md", "charter/overview.md".`,
		`- Do NOT emit top-level semantic payloads on stdout; keep semantic only inside shard-pack-manifest.json.`,
		`- semantic.entities[*] MUST remain full entity objects with provenance included; do not drop entities[*].provenance during repair.`,
		`- semantic.edges[*] MUST remain objects with canonical keys type/from/to; do not use kind/source/target aliases.`,
		`- semantic.findings[*] MUST remain objects and each finding MUST include title; never replace findings with plain strings or bullet text.`,
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
