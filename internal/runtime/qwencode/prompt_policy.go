package qwencode

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

func buildPrompt(taskPayload []byte, retry bool) string {
	mode := promptRetryNone
	if retry {
		mode = promptRetryParse
	}
	return buildPromptWithMode(taskPayload, mode)
}

func buildPromptWithMode(taskPayload []byte, mode promptRetryMode) string {
	return buildPromptWithModeAndHints(taskPayload, mode, nil)
}

func buildPromptWithModeAndHints(taskPayload []byte, mode promptRetryMode, extraHints []string) string {
	var task acpruntime.Task
	if err := json.Unmarshal(taskPayload, &task); err != nil {
		return strings.TrimSpace(fmt.Sprintf(`You are ACP runtime provider %q.
Return exactly one valid JSON object for ACP TaskResult.
Do not output markdown, code fences, or any explanatory text.
Task payload JSON:
%s`, acpruntime.ProviderQwenCode, strings.TrimSpace(string(taskPayload))))
	}

	repoScopesJSON := "[]"
	if rawRepoScopes, err := json.Marshal(task.RepoScopes); err == nil {
		repoScopesJSON = string(rawRepoScopes)
	}
	primaryRepoScope := primaryTaskRepoScope(task.RepoScope, task.RepoScopes)
	pathScopesJSON := "[]"
	if rawPathScopes, err := json.Marshal(task.PathScopes); err == nil {
		pathScopesJSON = string(rawPathScopes)
	}
	stepPolicy := buildStepSpecificPolicy(task.StepID)
	topLevelCompatibilityRule := buildTopLevelCompatibilityRule(task.StepID)
	repositoryEvidencePolicy := strings.Join([]string{
		`REPOSITORY EVIDENCE RULES:`,
		`- ACP workspace scaffold (workspace.yaml, charter/, model/, reports/) is support context, not the primary source tree.`,
		`- Prefer evidence from repository files under meta.repo_scopes/meta.path_scopes when those files are available.`,
		`- meta.path_scopes may contain directories, files, or a mixed disjoint partition; treat every listed scope as in-bounds evidence for this task.`,
		`- Use ACP-generated workspace artifacts as evidence only for ACP runtime/report state, not as a substitute for repository analysis.`,
	}, "\n")
	docFirstPolicy := buildDocFirstFilesystemPolicy(task)
	strictResultHint := strings.Join([]string{
		`STRICT RESULT JSON MODE:`,
		`- Prefer returning a direct TaskResult JSON object (without envelope wrappers).`,
		`- If using envelope fields like "result", value MUST be a non-empty valid JSON object string.`,
		`- Do NOT emit empty or malformed "result" payload.`,
		`- Do NOT draft, preview, or explain the TaskResult before returning it.`,
		`- Do NOT echo template fragments, markdown examples, or partial JSON.`,
	}, "\n")
	finalResponseDiscipline := strings.Join([]string{
		`FINAL RESPONSE DISCIPLINE:`,
		`- Use tool calls for filesystem reads/writes when needed, but the final assistant message MUST be only the TaskResult JSON object.`,
		`- Do NOT narrate file writes, manifest contents, or planning steps in the final message.`,
		`- After the last tool call, respond immediately with the final JSON object.`,
	}, "\n")
	retryHint := ""
	retryTemplate := ""
	if mode != promptRetryNone {
		retryLines := []string{
			`RETRY MODE: previous output needs one deterministic repair pass.`,
			`Do not include non-ASCII symbols in numbers or timestamps.`,
			`RFC3339 timestamps only (example: 2026-04-09T15:28:49Z).`,
			`Decimals must be compact numeric literals (example: 0.7, not 0. 7).`,
			`COMPACT JSON MODE: keep output concise and deterministic.`,
			`- If envelope form is unavoidable, "result" MUST be a non-empty valid JSON object string.`,
			`- Limit changeset to the minimum actionable operations for this step.`,
			`- Prefer "changeset": [] when the step can reuse already written artifacts and does not strictly require an operation.`,
			`- If changeset is non-empty, use at most 1 operation and at most 3 provenance.evidence items.`,
			`- Keep coverage compact: observed <=2 items, missing <=2 canonical items, notes <=1 short entry.`,
			`- Keep coverage.notes short (<=2 entries).`,
			`- Avoid long prose in summary; keep a single sentence.`,
			`- Unknown changeset[].op values are forbidden; allowed values are exactly: upsert_entity, remove_entity, upsert_edge, remove_edge, add_finding, add_doc_artifact.`,
			`- For op="add_doc_artifact", the payload key MUST be "doc_artifact"; never use "artifact".`,
			`- If a retry only repaired files inside write_root/draft_final_root, prefer "changeset": [] instead of inventing file-write operations.`,
			`- Retry repair is forbidden from inventing synthetic filesystem operations in changeset.`,
			`- Final response MUST start with "{" and end with "}".`,
			`- Do not output markdown fences, bullet lists, plan text, or template walkthroughs.`,
		}
		switch mode {
		case promptRetryParse:
			retryLines[0] = `RETRY MODE: previous output was invalid JSON.`
		case promptRetryArtifact:
			retryLines[0] = `ARTIFACT REPAIR MODE: previous collect output was schema-valid but write_root artifacts look skeletal or generic-only.`
			retryLines = append(retryLines,
				`- Repair artifact fidelity before returning JSON; this retry is not a fresh repository rediscovery pass.`,
				`- If write_root already contains authored docs, return the final TaskResult JSON immediately after minimal write_root inspection.`,
				`- Do NOT copy long citations, topic arrays, or manifest document inventories into TaskResult JSON.`,
				`- Do NOT collapse a multi-document refresh into one generic "cite.runtime-summary" citation.`,
				`- Do NOT overwrite a rich shard-pack-manifest.json with a skeletal reuse-only manifest.`,
				`- Preserve repo-specific citations when repository evidence already exists or can be recovered from repo roots.`,
				`- Keep repo roots available when write_root artifacts lack repo-specific citations or collapse to generic summaries.`,
				`- In shard-pack-manifest.json, compatibility.coverage/questions/entities/edges/findings must all exist, and questions/entities/edges/findings must be arrays rather than booleans or null.`,
				`- Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.`,
			)
		case promptRetryDraftArtifact:
			retryLines[0] = `DRAFT ARTIFACT REPAIR MODE: previous draft-only output was schema-valid but required runtime draft artifacts are invalid.`
			retryLines = append(retryLines,
				`- Repair runtime draft artifacts before returning JSON; this retry is not a fresh repository rediscovery pass.`,
				`- First inspect write_root and draft_final_root, then reuse already written draft files whenever possible.`,
				`- If the draft manifest and referenced draft files are already present, return the final TaskResult JSON immediately with "changeset": [].`,
				`- Draft-only retry must not register draft artifacts through legacy add_doc_artifact ops when the draft manifest already describes the publish surface.`,
			)
		case promptRetryCollectFresh:
			retryLines[0] = `COLLECT STALL RECOVERY MODE: previous collect attempt stalled before any artifact was finalized.`
			retryLines = append(retryLines,
				`- Do one minimal repo sweep only; avoid broad exploratory list_directory/read_file loops before the first authored artifact.`,
				`- Quickly produce authored docs plus shard-pack-manifest.json in write_root, or return the final TaskResult immediately if write_root is already complete.`,
				`- After the first filesystem write in write_root, repository exploration is finished except for minimal JSON/manifest repair.`,
				`- Broad repo sweeps after the first write are forbidden in this retry.`,
			)
		}
		retryLines = append(retryLines, extraHints...)
		retryLines = append(retryLines, buildRetryRecoveryHint(task))
		retryHint = strings.Join(retryLines, "\n")
		retryTemplate = "\nRetry-safe minimal template (preferred when reusing existing write_root artifacts):\n" + buildRetryMinimalTaskResultTemplateJSON(task)
	}

	return strings.TrimSpace(fmt.Sprintf(`You are ACP runtime provider %q.
Return exactly one valid JSON object for ACP TaskResult.
Do not output markdown, code fences, explanations, or any text outside the JSON object.

STRICT CONTRACT (must pass):
- top-level required keys: "meta", "summary", "changeset"
- meta required keys: "task_id", "step_id", "runtime", "started_at"
- meta.runtime required key: "name"
- use snake_case keys exactly as shown.
- DO NOT use top-level fields: task_id, run_id, step_id, status.
- %s
- provenance.kind MUST be one of: observation, inference, assertion.
- provenance.confidence MUST be a NUMBER in range [0,1], never a string.
- provenance.evidence MUST be an ARRAY of objects with repo/path.
- if "questions" is present, it MUST be an array of objects (each object has at least "id" and "text").
- coverage.missing MUST use canonical terms only: owner mappings, ci-cd evidence, delta validation, dependency graph, runtime metrics, api contracts, deployment configs, integration edges, datastore bindings, dependencies.
- question IDs MUST use canonical form without numeric suffixes (example: q.refresh.delta, not q.refresh.delta.1).
- Do not claim workspace is empty/minimal unless provenance evidence includes concrete file paths proving it.
%s
%s
%s
%s
%s
%s

Set meta fields exactly:
- meta.task_id = %q
- meta.step_id = %q
- meta.run_id = %q
- meta.runtime.name = %q
- meta.started_at = %q
- meta.workspace = %q
- meta.shard_id = %q
- meta.repo_scope = %q
- meta.repo_scopes = %s
- meta.path_scopes = %s

Schema-valid template for this task (copy structure and field TYPES, then refine values with available evidence):
%s
%s

Serialized runtime task JSON (context only):
%s`, acpruntime.ProviderQwenCode, topLevelCompatibilityRule, stepPolicy, repositoryEvidencePolicy, docFirstPolicy, strictResultHint, finalResponseDiscipline, retryHint, task.TaskID, task.StepID, task.RunID, acpruntime.ProviderQwenCode, task.StartedAtUTC.UTC().Format(time.RFC3339), task.Workspace, task.ShardID, primaryRepoScope, repoScopesJSON, pathScopesJSON, buildTaskResultTemplateJSON(task), retryTemplate, strings.TrimSpace(string(taskPayload))))
}

func buildTaskResultTemplateJSON(task acpruntime.Task) string {
	coverageMissing := []string{"owner mappings", "ci-cd evidence"}
	questions := []contracts.Question{}
	coverageObserved := []string{"services"}
	coverageNotes := []string{"evidence gaps are captured explicitly"}
	if strings.HasPrefix(task.StepID, "refresh.") {
		coverageMissing = append(coverageMissing, "delta validation")
		questions = []contracts.Question{
			{
				ID:       "q.refresh.delta",
				Text:     "What changed since previous run that affects ownership or dependencies?",
				Priority: "high",
			},
		}
	}
	if runtimedrafts.IsDraftStep(task.StepID) {
		coverageObserved = []string{"draft artifacts"}
		coverageMissing = nil
		coverageNotes = []string{"draft artifact coverage is captured by the runtime draft manifest"}
	}

	template := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:     task.TaskID,
			StepID:     task.StepID,
			RunID:      task.RunID,
			Runtime:    contracts.RuntimeMeta{Name: string(acpruntime.ProviderQwenCode), Version: "0.14.2"},
			StartedAt:  task.StartedAtUTC.UTC().Format(time.RFC3339),
			FinishedAt: task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
			Workspace:  task.Workspace,
			ShardID:    task.ShardID,
			RepoScope:  primaryTaskRepoScope(task.RepoScope, task.RepoScopes),
			RepoScopes: append([]string(nil), task.RepoScopes...),
			PathScopes: append([]string(nil), task.PathScopes...),
		},
		Summary:   "Task completed with contract-compliant output.",
		Changeset: buildTemplateChangeset(task),
		Coverage: &contracts.Coverage{
			Observed: coverageObserved,
			Missing:  coverageMissing,
			Notes:    coverageNotes,
		},
		Questions: questions,
		Warnings:  []string{},
	}
	raw, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func buildRetryMinimalTaskResultTemplateJSON(task acpruntime.Task) string {
	summary := "Reused existing shard artifacts."
	coverageObserved := []string{"artifacts"}
	coverageMissing := []string{"owner mappings", "runtime metrics"}
	coverageNotes := []string{"write_root artifacts reused."}
	if runtimedrafts.IsDraftStep(task.StepID) {
		summary = "Reused existing draft artifacts."
		coverageObserved = []string{"draft artifacts"}
		coverageMissing = nil
		coverageNotes = []string{"write_root and draft_final_root artifacts reused."}
	}

	template := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:     task.TaskID,
			StepID:     task.StepID,
			RunID:      task.RunID,
			Runtime:    contracts.RuntimeMeta{Name: string(acpruntime.ProviderQwenCode), Version: "0.14.2"},
			StartedAt:  task.StartedAtUTC.UTC().Format(time.RFC3339),
			FinishedAt: task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
			Workspace:  task.Workspace,
			ShardID:    task.ShardID,
			RepoScope:  primaryTaskRepoScope(task.RepoScope, task.RepoScopes),
			RepoScopes: append([]string(nil), task.RepoScopes...),
			PathScopes: append([]string(nil), task.PathScopes...),
		},
		Summary:   summary,
		Changeset: []contracts.Operation{},
		Coverage: &contracts.Coverage{
			Observed: coverageObserved,
			Missing:  coverageMissing,
			Notes:    coverageNotes,
		},
	}

	raw, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func buildRetryRecoveryHint(task acpruntime.Task) string {
	if runtimedrafts.IsDraftStep(task.StepID) {
		return buildDraftRetryRecoveryHint(task)
	}
	return buildCollectRetryRecoveryHint(task)
}

func buildCollectRetryRecoveryHint(task acpruntime.Task) string {
	lines := []string{
		`RETRY RECOVERY MODE: previous attempt may already have written shard artifacts.`,
		`- Retry goal is JSON repair, not fresh repository exploration.`,
		`- First inspect write_root, not repo roots.`,
		`- If authored docs already exist in write_root, reuse them instead of rediscovering the repository.`,
		`- Prefer "changeset": [] when write_root already contains authored docs.`,
		`- Once write_root contains authored docs, repository exploration is finished except for minimal JSON/manifest repair.`,
		`- After shard-pack-manifest.json exists, do NOT continue broad list_directory/read_file sweeps across repo roots.`,
		`- Do NOT collapse a multi-document refresh into one generic "cite.runtime-summary" citation.`,
		`- Do NOT reduce multi-document refresh evidence to one generic "cite.runtime-summary" citation.`,
		`- Preserve repo-specific citations when repository evidence already exists or can be recovered from repo roots.`,
		`- Preserve repo-specific citations when existing artifacts already contain them; do not replace them with one generic runtime summary citation.`,
		`- Do NOT use todo_write, plan-style narration, or repeated broad list_directory sweeps in retry mode.`,
		`- Do NOT delegate to agent/subagent helpers in retry mode.`,
		`- Use at most 3 tool calls in retry mode unless a required artifact is missing from write_root.`,
		`- Repair mode forbids inventing file operations in changeset; prefer "changeset": [].`,
		`- Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.`,
		`- After optional write_root inspection, respond immediately with the final TaskResult JSON object.`,
	}

	writeRoot := strings.TrimSpace(task.WriteRoot)
	files, total, err := collectRetryWriteRootFiles(writeRoot, 8)
	assessment, assessmentErr := assessRetryManifestAtWriteRoot(writeRoot)
	switch {
	case writeRoot == "":
		lines = append(lines, `- write_root snapshot unavailable because write_root is empty.`)
	case err != nil:
		lines = append(lines, fmt.Sprintf(`- write_root snapshot unavailable: %v.`, err))
	case total == 0:
		lines = append(lines, fmt.Sprintf(`- write_root %q is empty or missing; create only the minimum missing artifacts before returning JSON.`, writeRoot))
	default:
		suffix := ""
		if total > len(files) {
			suffix = fmt.Sprintf(" (+%d more)", total-len(files))
		}
		lines = append(lines, fmt.Sprintf(`- write_root already contains %d file(s): %s%s`, total, strings.Join(files, ", "), suffix))
		if containsString(files, "shard-pack-manifest.json") {
			switch {
			case assessmentErr != nil:
				lines = append(lines, fmt.Sprintf(`- shard-pack-manifest.json is present but could not be assessed (%v); keep repo roots available while repairing JSON.`, assessmentErr))
				lines = append(lines, `- Rewrite shard-pack-manifest.json to the canonical ACP schema (version=1 integer, documents[].citation_ids, citations[].id/document_ids, stable canonical_path).`)
			case assessment.Rich:
				lines = append(lines, `- shard-pack-manifest.json is already present in write_root; read it first and reuse authored docs instead of re-reading the repository.`)
				lines = append(lines, `- When reusing authored docs, prefer "changeset": [] or one minimal operation instead of restating manifest contents.`)
				lines = append(lines, `- Do NOT copy long citations, topic arrays, or manifest document inventories into TaskResult JSON.`)
			default:
				lines = append(lines, `- shard-pack-manifest.json exists in write_root but looks skeletal/reuse-only; keep repo roots in include-directories and repair JSON without collapsing repository evidence.`)
				lines = append(lines, `- Do NOT reduce multi-document refresh evidence to one generic "cite.runtime-summary" citation.`)
				lines = append(lines, `- Preserve or restore repo-specific citations before returning the final TaskResult JSON object.`)
			}
		}
		if shouldConstrainRetryToWriteRoot(task) {
			lines = append(lines, `- Retry include-directories are constrained to write_root because the manifest and authored docs already exist.`)
		}
	}
	return strings.Join(lines, "\n")
}

func buildDraftRetryRecoveryHint(task acpruntime.Task) string {
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	lines := []string{
		`RETRY RECOVERY MODE: previous attempt may already have written draft artifacts.`,
		fmt.Sprintf(`- Retry goal is repairing %s plus referenced draft files, not fresh repository exploration.`, manifestFile),
		`- First inspect write_root and draft_final_root, not repo roots.`,
		`- Reuse already written draft files whenever possible and prefer "changeset": [].`,
		`- Do NOT use todo_write, plan-style narration, or repeated broad list_directory/read_file sweeps in retry mode.`,
		`- Do NOT delegate to agent/subagent helpers in retry mode.`,
		`- Repair mode forbids inventing filesystem operations in changeset; if files already exist, return the final TaskResult JSON immediately.`,
	}

	writeRoot := strings.TrimSpace(task.WriteRoot)
	draftRoot := strings.TrimSpace(task.DraftFinalRoot)
	files, total, err := collectRetryWriteRootFiles(writeRoot, 8)
	switch {
	case writeRoot == "":
		lines = append(lines, `- write_root snapshot unavailable because write_root is empty.`)
	case err != nil:
		lines = append(lines, fmt.Sprintf(`- write_root snapshot unavailable: %v.`, err))
	case total == 0:
		lines = append(lines, fmt.Sprintf(`- write_root %q is empty or missing; create only the missing draft manifest and referenced draft files before returning JSON.`, writeRoot))
	default:
		suffix := ""
		if total > len(files) {
			suffix = fmt.Sprintf(" (+%d more)", total-len(files))
		}
		lines = append(lines, fmt.Sprintf(`- write_root already contains %d file(s): %s%s`, total, strings.Join(files, ", "), suffix))
		if containsString(files, manifestFile) {
			lines = append(lines, fmt.Sprintf(`- %s is already present in write_root; repair that file instead of writing a legacy constitution/report schema.`, manifestFile))
		}
	}
	if draftRoot != "" {
		lines = append(lines, fmt.Sprintf(`- draft_final_root for this retry: %q`, draftRoot))
	}
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		lines = append(lines,
			`- constitution-draft.json must use the exact runtime draft manifest contract (version=1 integer, run_id, step_id, step_contract, agent_role, outputs[]).`,
			`- outputs[] must map "charter-overview.md" -> "charter/overview.md" and "baseline-subagents.yaml" -> "skills/subagents.yaml".`,
			`- The actual draft files must be written at draft_final_root/charter-overview.md and draft_final_root/baseline-subagents.yaml; do NOT nest them under draft_final_root/charter/ or draft_final_root/skills/.`,
			`- Do NOT emit legacy constitution shapes such as schema_version/system_id/services/coverage or version:"0.1.0".`,
		)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines,
			`- asis-draft-manifest.json must remain the only publish-surface manifest for reports/as-is/*, reports/coverage/*, and reports/agent-outputs/* draft outputs.`,
			`- If valid draft artifacts already exist, return "changeset": [] and do NOT register them again via add_doc_artifact.`,
		)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines,
			`- proposals-draft-manifest.json must remain the only publish-surface manifest for proposals/* and reports/changelog/* draft outputs.`,
			`- If valid draft artifacts already exist, return "changeset": [] and do NOT register them again via add_doc_artifact.`,
		)
	}
	return strings.Join(lines, "\n")
}

func buildCollectFreshRetryHints(task acpruntime.Task) []string {
	lines := []string{
		`- This retry exists because the provider produced no durable collect artifacts in time.`,
		`- Spend the minimum time needed to identify repo-backed evidence for one authored doc and shard-pack-manifest.json.`,
		`- Keep repository exploration minimal; do not resume a broad repo sweep on this retry.`,
		`- Do NOT delegate to agent/subagent helpers and do NOT use todo_write-style planning on this retry.`,
		`- If write_root is still empty, create only the minimum viable authored doc set before returning the final JSON.`,
		`- As soon as the first authored doc and shard-pack-manifest.json exist in write_root, stop repository exploration and return the final TaskResult JSON immediately.`,
	}
	if strings.TrimSpace(task.WriteRoot) != "" {
		lines = append(lines, fmt.Sprintf(`- write_root for this retry: %q`, strings.TrimSpace(task.WriteRoot)))
	}
	return lines
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func nonEmptyOr(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

func buildTopLevelCompatibilityRule(stepID string) string {
	return steppolicy.TopLevelCompatibilityRule(stepID)
}

func buildStepSpecificPolicy(stepID string) string {
	return steppolicy.StepSpecificPolicy(stepID)
}

func buildDocFirstFilesystemPolicy(task acpruntime.Task) string {
	return steppolicy.DocFirstFilesystemPolicy(task)
}

func buildConstitutionDraftManifestExample(task acpruntime.Task) string {
	return steppolicy.ConstitutionDraftManifestExample(task)
}

func buildParseRepairHints(stepID string, parseStage string, parseErr error) []string {
	return steppolicy.ParseRepairHints(stepID, parseStage, parseErr)
}

func buildArtifactRepairHints(initialProblem string) []string {
	return steppolicy.CollectArtifactRepairHints(initialProblem)
}

func buildDraftArtifactRepairHints(task acpruntime.Task, validationErr error) []string {
	return steppolicy.DraftArtifactRepairHints(task, validationErr)
}

func compactRetryHint(value string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(normalized) <= 320 {
		return normalized
	}
	return normalized[:317] + "..."
}

func buildTemplateChangeset(task acpruntime.Task) []contracts.Operation {
	if runtimedrafts.IsDraftStep(task.StepID) {
		return []contracts.Operation{}
	}
	scopes := append([]string(nil), task.RepoScopes...)
	if len(scopes) == 0 {
		scopes = []string{"repository"}
	}
	changes := make([]contracts.Operation, 0, len(scopes))
	switch task.StepID {
	case "init.step3.findings", "refresh.step3.findings":
		for _, scope := range scopes {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				scope = "repository"
			}
			slug := slugutil.Slugify(scope)
			if slug == "" {
				slug = "repository"
			}
			changes = append(changes, contracts.Operation{
				Op: "add_finding",
				Finding: &contracts.Finding{
					ID:          "finding.missing-owner.svc." + slug,
					Severity:    "medium",
					Title:       "Missing owner mapping",
					Description: "owner_team_id is not confirmed",
					RuleID:      "rule.owner.required",
					RelatedIDs:  []string{"svc." + slug},
					Provenance: contracts.Provenance{
						Kind:       "inference",
						Confidence: 0.66,
					},
				},
			})
		}
	}
	return changes
}

func humanizeScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "Repository"
	}
	parts := strings.FieldsFunc(scope, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})
	for idx, part := range parts {
		if part == "" {
			continue
		}
		parts[idx] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	name := strings.TrimSpace(strings.Join(parts, " "))
	if name == "" {
		return "Repository"
	}
	return name
}

func primaryTaskRepoScope(explicit string, scopes []string) string {
	return steppolicy.PrimaryTaskRepoScope(explicit, scopes)
}

func collectRepoEntrypointHints(task acpruntime.Task) []string {
	return steppolicy.CollectRepoEntrypointHints(task)
}

func formatEntrypointHint(_ string, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	return filepath.ToSlash(target)
}
