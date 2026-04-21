package promptcontract

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type PromptPack struct {
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	Source       string `json:"source"`
	Warning      string `json:"warning,omitempty"`
	Content      string `json:"content,omitempty"`
}

func PromptPackNameForStep(stepID string) string {
	switch strings.TrimSpace(stepID) {
	case "init.step1.collect", "refresh.step1.collect":
		return "collect-context"
	case "init.step3.findings", "refresh.step3.findings":
		return "findings"
	default:
		return ""
	}
}

func LoadPromptPack(task acpruntime.Task) PromptPack {
	name := PromptPackNameForStep(task.StepID)
	if name == "" {
		return PromptPack{}
	}
	relativePath := fmt.Sprintf("skills/prompt-packs/%s.md", name)
	pack := PromptPack{
		Name:         name,
		RelativePath: relativePath,
		Source:       "seeded-baseline",
	}

	fallback, hasFallback := workspace.BaselinePromptPack(name)
	workspaceRoot := strings.TrimSpace(task.Workspace)
	if workspaceRoot == "" {
		pack.Warning = fmt.Sprintf("%s unavailable because meta.workspace is empty; using seeded baseline prompt pack", relativePath)
		pack.Content = fallback
		if !hasFallback {
			pack.Source = "missing"
			pack.Warning = fmt.Sprintf("%s is unavailable and no seeded baseline exists", relativePath)
		}
		return pack
	}

	ws := workspace.Root{Path: workspaceRoot}
	absPath, err := ws.Resolve(relativePath)
	switch {
	case err == nil:
		raw, readErr := os.ReadFile(absPath)
		if readErr == nil && strings.TrimSpace(string(raw)) != "" {
			pack.Source = "workspace"
			pack.Content = strings.TrimSpace(string(raw))
			return pack
		}
		if readErr != nil {
			pack.Warning = fmt.Sprintf("%s could not be read (%v); using seeded baseline prompt pack", relativePath, readErr)
		} else {
			pack.Warning = fmt.Sprintf("%s is empty; using seeded baseline prompt pack", relativePath)
		}
	default:
		pack.Warning = fmt.Sprintf("%s could not be resolved (%v); using seeded baseline prompt pack", relativePath, err)
	}

	if hasFallback {
		pack.Content = fallback
		return pack
	}
	pack.Source = "missing"
	pack.Warning = fmt.Sprintf("%s is unavailable and no seeded baseline exists", relativePath)
	return pack
}

func AdditivePromptPackSection(task acpruntime.Task) (string, PromptPack) {
	pack := LoadPromptPack(task)
	if strings.TrimSpace(pack.Name) == "" {
		return "", pack
	}
	lines := []string{
		fmt.Sprintf("ADDITIVE WORKSPACE PROMPT PACK (%s, source=%s):", pack.RelativePath, pack.Source),
	}
	if warning := strings.TrimSpace(pack.Warning); warning != "" {
		lines = append(lines, fmt.Sprintf("- Prompt pack warning: %s", warning))
	}
	if content := strings.TrimSpace(pack.Content); content != "" {
		lines = append(lines, content)
	}
	return strings.Join(lines, "\n"), pack
}

func RepositoryEvidencePolicy() string {
	return strings.Join([]string{
		`REPOSITORY EVIDENCE RULES:`,
		`- ACP workspace scaffold (workspace.yaml, charter/, model/, reports/) is support context, not the primary source tree.`,
		`- Prefer evidence from repository files under meta.repo_scopes/meta.path_scopes when those files are available.`,
		`- meta.path_scopes may contain directories, files, or a mixed disjoint partition; treat every listed scope as in-bounds evidence for this task.`,
		`- Use ACP-generated workspace artifacts as evidence only for ACP runtime/report state, not as a substitute for repository analysis.`,
	}, "\n")
}

func StepPolicy(stepID string) string {
	switch stepID {
	case "refresh.step1.collect":
		return strings.Join([]string{
			`STEP POLICY refresh.step1.collect:`,
			`- Allowed upsert_entity types: service, datastore, integration, external.system, team, domain, api, component.`,
			`- Forbidden placeholder entity types: runtime_provider, runtime, metadata.`,
			`- Analyze only repository/workspace artifacts; do NOT perform web search or external browsing.`,
			`- Every provenance.evidence.path must resolve to an existing file in workspace/repo scope.`,
			`- Do NOT emit synthetic evidence paths such as search_source/*, search_query/*, search_config/*.`,
			`- Do NOT introduce unrelated incident domains (for example bidding/tender/power-system topics) unless explicitly present in repository evidence.`,
			`- If evidence is incomplete, capture gap via coverage.missing instead of synthetic placeholder entities.`,
			`- Include at least one question and at least three items in coverage.missing.`,
		}, "\n")
	case "init.step3.findings", "refresh.step3.findings":
		return strings.Join([]string{
			`STEP POLICY step3.findings:`,
			`- If owner mapping is unresolved in evidence/coverage, include at least one add_finding operation.`,
			`- Each finding must include rule_id, related_ids, and provenance.evidence[].`,
			`- For observation provenance, evidence array MUST be non-empty.`,
			`- Prioritize staged final artifacts and validator context roots; do not perform broad repository rediscovery in this step.`,
			`- Do not recursively crawl whole repositories; inspect only files needed to validate/fix staged final artifacts.`,
			`- If meta.repo_scopes has 2+ scopes, include at least one upsert_edge that links entities from different repo_scope values.`,
			`- For upsert_edge use canonical keys only: edge.id, edge.type, edge.from, edge.to.`,
			`- Forbidden edge aliases: edge.kind, edge.source, edge.target.`,
			`- Minimal valid upsert_edge example: {"op":"upsert_edge","edge":{"id":"edge.cross.scope","type":"depends_on","from":"svc.a","to":"svc.b","provenance":{"kind":"inference","confidence":0.6,"evidence":[{"repo":"scope-a","path":"README.md"},{"repo":"scope-b","path":"README.md"}]}}}`,
			`- Every provenance.evidence.path must resolve to an existing file in workspace/repo scope.`,
			`- Do NOT emit synthetic evidence paths such as search_source/*, search_query/*, search_config/*.`,
			`- Include at least one question and at least three items in coverage.missing.`,
		}, "\n")
	default:
		if strings.HasPrefix(stepID, "refresh.") {
			return `For refresh steps include at least one question object and at least three items in coverage.missing.`
		}
		return ""
	}
}

func SharedTaskResultContractLines() []string {
	return []string{
		`- top-level required keys: "meta", "summary", "changeset"`,
		`- meta required keys: "task_id", "step_id", "runtime", "started_at"`,
		`- meta.runtime required key: "name"`,
		`- use snake_case keys exactly as shown.`,
		`- DO NOT use top-level fields: task_id, run_id, step_id, status.`,
		`- provenance.kind MUST be one of: observation, inference, assertion.`,
		`- provenance.confidence MUST be a NUMBER in range [0,1], never a string.`,
		`- provenance.evidence MUST be an ARRAY of objects with repo/path.`,
		`- if "questions" is present, it MUST be an array of objects (each object has at least "id" and "text").`,
		`- coverage.missing MUST use canonical terms only: owner mappings, ci-cd evidence, delta validation, dependency graph, runtime metrics, api contracts, deployment configs, integration edges, datastore bindings, dependencies.`,
		`- question IDs MUST use canonical form without numeric suffixes (example: q.refresh.delta, not q.refresh.delta.1).`,
		`- Do not claim workspace is empty/minimal unless provenance evidence includes concrete file paths proving it.`,
	}
}

func SharedRetryGuardrailLines() []string {
	return []string{
		`- Do NOT collapse a multi-document refresh into one generic "cite.runtime-summary" citation.`,
		`- Do NOT overwrite a rich shard-pack-manifest.json with a skeletal reuse-only manifest.`,
		`- Preserve repo-specific citations when repository evidence already exists or can be recovered from repo roots.`,
		`- Unknown changeset[].op values are forbidden; allowed values are exactly: upsert_entity, remove_entity, upsert_edge, remove_edge, add_finding, add_doc_artifact.`,
		`- For op="add_doc_artifact", the payload key MUST be "doc_artifact"; never use "artifact".`,
		`- If a retry only repaired files inside write_root, prefer "changeset": [] instead of inventing file-write operations.`,
		`- Final response MUST start with "{" and end with "}".`,
		`- Do not output markdown fences, bullet lists, plan text, or template walkthroughs.`,
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
		fmt.Sprintf(`- artifact_root (workspace-relative) = %q`, strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- read_context_roots = %s`, readContextRootsJSON),
		fmt.Sprintf(`- domain_id = %q`, strings.TrimSpace(task.DomainID)),
		fmt.Sprintf(`- agent_role = %q`, strings.TrimSpace(task.AgentRole)),
	}
	switch task.StepID {
	case "init.step1.collect", "refresh.step1.collect":
		lines = append(lines,
			`- Produce runtime-authored documents in write_root and then write shard-pack-manifest.json in write_root.`,
			`- shard-pack-manifest.json must describe every authored document, its canonical stable path, citations, and compatibility snapshot.`,
			`- You may be flexible in document structure, but promotion and rendering depend on manifest citations/topics remaining accurate.`,
		)
		lines = append(lines, artifactquality.CollectManifestContractLines(strings.TrimSpace(task.ArtifactRoot))...)
		lines = append(lines, artifactquality.ClaimIDContractLines()...)
		lines = append(lines,
			`- documents[].path in shard-pack-manifest.json MUST stay artifact_root-relative; reports/taskruns/... staging-prefixed paths are forbidden.`,
			`- Do NOT collapse a multi-document refresh surface to one generic "cite.runtime-summary" citation when repository evidence exists.`,
			`- Preserve repo-specific citations in shard-pack-manifest.json whenever repository files support them.`,
		)
	case "init.step3.findings", "refresh.step3.findings":
		lines = append(lines,
			`- Inspect staged final artifacts under reports/taskruns/<run_id>/staging/final from read_context_roots.`,
			`- Use staged/final artifacts as primary evidence; do not treat full repository recrawl as default behavior in validator step.`,
			`- Write validator-verdict.json in write_root.`,
			`- Validator may fix only indexes, references, or technical document issues inside write_root; do not rewrite document meaning wholesale.`,
		)
		lines = append(lines, artifactquality.ClaimIDContractLines()...)
	}
	return strings.Join(lines, "\n")
}

func ParseRepairHints(parseStage string, parseErr error) []string {
	if parseErr == nil {
		return nil
	}
	lines := []string{}
	if detail := compactRetryHint(parseErr.Error()); detail != "" {
		stage := strings.TrimSpace(parseStage)
		if stage == "" {
			stage = "unknown"
		}
		lines = append(lines, fmt.Sprintf(`Previous %s validation failure: %s`, stage, detail))
	}
	return append(lines,
		`RETRY OBJECTIVE: return exactly one minimal TaskResult JSON object that validates for this task.`,
		`Return a direct TaskResult JSON object, not an event-stream transcript or tool wrapper.`,
		`Do NOT include tool logs, event arrays, envelope commentary, or any non-JSON preface/suffix.`,
		`Do NOT use ad-hoc ops such as upsert_file, write_file, update_file, or todo_write in changeset[].op.`,
		`For add_doc_artifact, use doc_artifact and never the legacy artifact field.`,
	)
}

func ArtifactRepairHints(initialProblem string) []string {
	lines := []string{
		`RETRY OBJECTIVE: repair collect artifacts deterministically, then return exactly one minimal TaskResult JSON object.`,
		`Rebuild shard-pack-manifest.json to the canonical ACP schema before returning JSON.`,
		`compatibility.coverage/questions/entities/edges/findings are all required; questions/entities/edges/findings must be arrays even when empty.`,
		`Do NOT leave claim_ids empty for cited repository evidence; preserve concrete repo-backed claim ids whenever the evidence supports them.`,
		`Do NOT describe shard-pack-manifest.json repair via add_doc_artifact; repair the file in write_root and return "changeset": [].`,
	}
	if detail := compactRetryHint(initialProblem); detail != "" {
		lines = append(lines, fmt.Sprintf(`Previous artifact contract failure: %s`, detail))
	}
	return lines
}

func compactRetryHint(value string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(normalized) <= 320 {
		return normalized
	}
	return normalized[:317] + "..."
}
