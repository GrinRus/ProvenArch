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

func TopLevelCompatibilityRule(stepID string) string {
	if isCollectStep(stepID) {
		return `Do NOT add a top-level "compatibility" field to TaskResult JSON; shard-pack-manifest.json may contain compatibility, but the response payload must not.`
	}
	if runtimedrafts.IsDraftStep(stepID) {
		return `Do NOT add a top-level "compatibility" field to TaskResult JSON; runtime draft metadata belongs only inside constitution-draft.json / asis-draft-manifest.json / proposals-draft-manifest.json files under write_root.`
	}
	return `Do NOT add a top-level "compatibility" field to TaskResult JSON unless the step contract explicitly allows it in a runtime file under write_root.`
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
			`- Default TaskResult template for this step is changeset: []; do not invent synthetic upsert_entity operations for draft-only output.`,
		}, "\n")
	case "init.step1.collect":
		return strings.Join([]string{
			`STEP POLICY init.step1.collect:`,
			`- Do NOT delegate to agent/subagent helpers.`,
			`- Do NOT use todo_write-style planning or long plan narration.`,
			`- Use only existing repo entrypoint hints; do not assume README.md exists when it is not present.`,
			`- After the first useful evidence pass, converge to authored docs plus shard-pack-manifest.json instead of continuing a broad repo sweep.`,
		}, "\n")
	case "refresh.step1.collect":
		return strings.Join([]string{
			`STEP POLICY refresh.step1.collect:`,
			`- Do NOT delegate to agent/subagent helpers.`,
			`- Do NOT use todo_write-style planning or long plan narration.`,
			`- Use only existing repo entrypoint hints; do not assume README.md exists when it is not present.`,
			`- After the first useful evidence pass, converge to authored docs plus shard-pack-manifest.json instead of continuing a broad repo sweep.`,
			`- Allowed upsert_entity types: service, datastore, integration, external.system, team, domain, api, component.`,
			`- Forbidden placeholder entity types: runtime_provider, runtime, metadata.`,
			`- Analyze only repository/workspace artifacts; do NOT perform web search or external browsing.`,
			`- Every provenance.evidence.path must resolve to an existing file in workspace/repo scope.`,
			`- Do NOT emit synthetic evidence paths such as search_source/*, search_query/*, search_config/*.`,
			`- Do NOT introduce unrelated incident domains (for example bidding/tender/power-system topics) unless explicitly present in repository evidence.`,
			`- If evidence is incomplete, capture gap via coverage.missing instead of synthetic placeholder entities.`,
			`- Include at least one question and at least three items in coverage.missing.`,
		}, "\n")
	case "refresh.step3.findings":
		return strings.Join([]string{
			`STEP POLICY refresh.step3.findings:`,
			`- If owner mapping is unresolved in evidence/coverage, include at least one add_finding operation.`,
			`- Each finding must include rule_id, related_ids, and provenance.evidence[].`,
			`- For observation provenance, evidence array MUST be non-empty.`,
			`- If meta.repo_scopes has 2+ scopes, include at least one upsert_edge that links entities from different repo_scope values.`,
			`- For upsert_edge use canonical keys only: edge.id, edge.type, edge.from, edge.to.`,
			`- Forbidden edge aliases: edge.kind, edge.source, edge.target.`,
			`- Minimal valid upsert_edge example: {"op":"upsert_edge","edge":{"id":"edge.cross.scope","type":"depends_on","from":"svc.a","to":"svc.b","provenance":{"kind":"inference","confidence":0.6,"evidence":[{"repo":"scope-a","path":"README.md"},{"repo":"scope-b","path":"README.md"}]}}}`,
			`- Include at least one question and at least three items in coverage.missing.`,
		}, "\n")
	default:
		if strings.HasPrefix(strings.TrimSpace(stepID), "refresh.") {
			return `For refresh steps include at least one question object and at least three items in coverage.missing.`
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
		`- Use tool calls for any file writes, but keep the final assistant response limited to the required TaskResult JSON object.`,
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
			`- The default final TaskResult for this step should keep "changeset": [].`,
			`- Exact constitution-draft.json example (replace IDs/summary only, keep keys/types and output mapping):`,
			ConstitutionDraftManifestExample(task),
			`- Keep the draft deterministic in shape; compiler will normalize/publish canonical files afterwards.`,
		)
	case "init.step1.collect", "refresh.step1.collect":
		lines = append(lines,
			`- Do NOT delegate to agent/subagent helpers and do NOT use todo_write-style planning.`,
			`- Before the first filesystem write inside write_root, keep repository exploration minimal and converge quickly on the first authored doc plus shard-pack-manifest.json.`,
			`- Produce runtime-authored documents in write_root and then write shard-pack-manifest.json in write_root.`,
			`- shard-pack-manifest.json must describe every authored document, its canonical stable path, citations, and compatibility snapshot.`,
			`- In shard-pack-manifest.json, compatibility MUST include coverage, questions, entities, edges, and findings.`,
			`- Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.`,
			`- You may be flexible in document structure, but promotion and rendering depend on manifest citations/topics remaining accurate.`,
			`- After the first filesystem write inside write_root, stop broad repository exploration; only minimal manifest/JSON repair is allowed afterwards.`,
			`- After writing shard-pack-manifest.json, do NOT continue broad list_directory/read_file sweeps across repo roots.`,
			`- If authored docs already exist in write_root, respond immediately with the final TaskResult JSON object.`,
		)
		lines = append(lines, artifactquality.CollectManifestContractLines(strings.TrimSpace(task.ArtifactRoot))...)
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
		)
		lines = append(lines, artifactquality.ClaimIDContractLines()...)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines,
			`- Write asis-draft-manifest.json in write_root.`,
			`- Draft final docs only under draft_final_root.`,
			`- Allowed canonical targets are reports/as-is/*, reports/coverage/*, and reports/agent-outputs/*.`,
			`- If asis-draft-manifest.json already describes the publish surface, prefer "changeset": [] and do NOT re-register draft artifacts through legacy add_doc_artifact ops.`,
			`- Compiler will merge these drafts into staged final artifacts and keep canonical layout/indexing deterministic.`,
		)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines,
			`- Inspect validated staged artifacts under reports/taskruns/<run_id>/staging/final from read_context_roots.`,
			`- Write proposals-draft-manifest.json in write_root.`,
			`- Draft final docs only under draft_final_root.`,
			`- Allowed canonical targets are proposals/* and reports/changelog/*.`,
			`- If proposals-draft-manifest.json already describes the publish surface, prefer "changeset": [] and do NOT re-register draft artifacts through legacy add_doc_artifact ops.`,
			`- Promotion remains deterministic; your drafts become publish candidates only after compile/publish gates.`,
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
		lines = append(lines, "- "+TopLevelCompatibilityRule(stepID))
	}
	return append(lines,
		`- Return a direct TaskResult JSON object, not an event-stream transcript or tool wrapper.`,
		`- Do NOT use ad-hoc ops such as upsert_file, write_file, update_file, or todo_write in changeset[].op.`,
		`- For add_doc_artifact, use doc_artifact and never the legacy artifact field.`,
	)
}

func CollectArtifactRepairHints(initialProblem string) []string {
	lines := []string{
		`- Rebuild shard-pack-manifest.json to the canonical ACP schema before returning JSON.`,
		`- In shard-pack-manifest.json, compatibility.coverage/questions/entities/edges/findings are all required; questions/entities/edges/findings must be arrays even when empty.`,
		`- documents[].path MUST stay relative to artifact_root only; valid example: "iac-overview.md". Invalid examples: "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac/iac-overview.md", "charter/overview.md".`,
		`- Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.`,
		`- compatibility.entities[*] MUST remain full entity objects with provenance included; do not drop entities[*].provenance during repair.`,
		`- compatibility.edges[*] MUST remain objects with canonical keys type/from/to; do not use kind/source/target aliases.`,
		`- compatibility.findings[*] MUST remain objects and each finding MUST include title; never replace findings with plain strings or bullet text.`,
		`- compatibility.questions/entities/edges/findings must stay object-only arrays; booleans, nulls, and string-valued findings are invalid.`,
		`- Do NOT leave claim_ids empty for cited repository evidence; preserve concrete repo-backed claim ids whenever the evidence supports them.`,
		`- Do NOT describe shard-pack-manifest.json repair via add_doc_artifact; repair the file in write_root and return "changeset": [].`,
		`- Repair mode is JSON-only: do not invent extra repository file reads/writes in changeset after authored docs already exist.`,
		`- Valid compatibility examples: entities[*].provenance={"kind":"observation","confidence":0.7,"evidence":[...]}, edges[*]={"id":"edge.dep","type":"depends_on","from":"svc.a","to":"svc.b","provenance":{...}}, findings[*]={"id":"finding.x","severity":"medium","title":"Missing owner mapping","description":"...","rule_id":"rule.owner.required","related_ids":["svc.a"],"provenance":{...}}.`,
	}
	if detail := compactRetryHint(initialProblem); detail != "" {
		lines = append(lines, fmt.Sprintf(`- Previous artifact contract failure: %s`, detail))
	}
	return lines
}

func DraftArtifactRepairHints(task acpruntime.Task, validationErr error) []string {
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	lines := []string{
		fmt.Sprintf(`- Repair %s to the canonical ACP runtime draft manifest contract before returning JSON.`, manifestFile),
		`- version MUST be the integer 1; string values such as "0.1.0" are invalid.`,
		`- The manifest MUST include run_id, step_id, step_contract, agent_role, and outputs[].`,
		`- outputs[].path MUST stay relative to draft_final_root and outputs[].canonical_path MUST stay workspace-relative.`,
		`- If valid draft files already exist under draft_final_root, reuse them and return "changeset": [].`,
		`- Do NOT describe draft manifest repair via add_doc_artifact when the draft manifest already describes the publish surface.`,
		`- Repair mode is draft-only: do not invent extra repository file reads/writes in changeset after draft files already exist.`,
	}
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		lines = append(lines,
			`- constitution-draft.json exact required outputs[] entries are: {"path":"charter-overview.md","canonical_path":"charter/overview.md","kind":"charter","title":"Constitution"} and {"path":"baseline-subagents.yaml","canonical_path":"skills/subagents.yaml","kind":"bundle","title":"Baseline Subagents"}.`,
			`- Do NOT emit legacy constitution shapes such as schema_version, system_id, services, relations, governance, coverage_notes, or version:"0.1.0".`,
			`- Draft files referenced by constitution-draft.json must exist under draft_final_root before the final TaskResult is returned.`,
		)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines,
			`- asis-draft-manifest.json is the only publish-surface manifest for reports/as-is/*, reports/coverage/*, and reports/agent-outputs/* drafts.`,
			`- Final response should prefer "changeset": [] after draft artifact repair; do not re-register draft artifacts via legacy metadata ops.`,
		)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines,
			`- proposals-draft-manifest.json is the only publish-surface manifest for proposals/* and reports/changelog/* drafts.`,
			`- Final response should prefer "changeset": [] after draft artifact repair; do not re-register draft artifacts via legacy metadata ops.`,
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
	patterns := []string{"README.*", "catalog-info.yaml", "pyproject.toml", "package.json", "docker-compose*", "skaffold.yaml", "Makefile"}
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
