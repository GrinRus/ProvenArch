package promptcontract

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

func ComposeArtifactOnlyPrompt(provider acpruntime.Provider, task acpruntime.Task) string {
	sections := []string{
		fmt.Sprintf("You are ACP runtime provider %q.", provider),
	}
	sections = append(sections, SharedSections(task)...)
	return strings.Join(sections, "\n\n")
}

func ComposeCollectManifestRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, _ error) string {
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), "shard-pack-manifest.json")
	authoredDocs := authoredRepairDocuments(task.WriteRoot)
	evidencePaths := repairEvidenceCandidates(task)
	skeleton := steppolicy.CollectManifestTaskSkeleton(task, authoredDocs, evidencePaths)
	sections := []string{
		fmt.Sprintf("You are ACP runtime provider %q in collect manifest repair mode.", provider),
		"Immediate repair action:",
		fmt.Sprintf("- Write exactly one file now: %q.", manifestTarget),
		"- Do not begin with broad analysis. Run the preferred file write command below, or perform exactly the same single-file edit.",
		"- Do not rewrite existing authored markdown documents.",
		"- If shard-pack-manifest.json already exists but is invalid, do not inspect or patch it; overwrite it from the heredoc command.",
		"- Copy the heredoc JSON exactly during repair. Do not make factual edits before the file validates.",
		"TASK-SPECIFIC MANIFEST JSON SKELETON:",
		"Use the JSON embedded in this command as the skeleton and target content:",
		collectManifestRepairWriteCommand(task.WriteRoot, manifestTarget, skeleton),
		"Artifact-only repair contract:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		"- Write or replace only write_root/shard-pack-manifest.json.",
		fmt.Sprintf("- Exact allowed write target: %q.", manifestTarget),
		"- Final action must be: write only write_root/shard-pack-manifest.json, then exit successfully.",
		"- Exit with code 0 only after shard-pack-manifest.json validates.",
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
		"Existing authored documents in write_root are already encoded in the task-specific skeleton; do not rewrite them.",
	}
	if len(authoredDocs) > 0 {
		repairLines = append(repairLines, "Existing authored document files in write_root:")
		for _, rel := range authoredDocs {
			repairLines = append(repairLines, fmt.Sprintf("- %s", rel))
		}
	}
	if len(evidencePaths) > 0 {
		repairLines = append(repairLines,
			fmt.Sprintf("- Repository evidence candidates are already encoded in the skeleton (%d path candidates); do not browse for more evidence in repair mode.", len(evidencePaths)),
		)
	}
	repairLines = append(repairLines, overwriteCollectManifestRepairInstructions()...)
	repairLines = append(repairLines,
		"COLLECT MANIFEST REPAIR CHECKLIST:",
	)
	repairLines = append(repairLines, compactCollectManifestValidationChecklist(strings.TrimSpace(task.ArtifactRoot))...)
	repairLines = append(repairLines,
		`- Repair-mode note: if schemas/* or docs/spec/* are absent from the runtime workspace, do not look for them; use this embedded checklist.`,
		`- The task-specific JSON skeleton above is the complete repair artifact; write it exactly from the heredoc command, do not make factual edits in repair mode, then exit.`,
	)
	sections = append(sections, strings.Join(repairLines, "\n"))
	return strings.Join(sections, "\n\n")
}

func ComposeCollectArtifactPairRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error) string {
	docRel := steppolicy.SuggestedCollectDocumentPath(task)
	docTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), filepath.FromSlash(docRel))
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), "shard-pack-manifest.json")
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in collect artifact pair focused recovery mode.", provider),
		"Immediate recovery action:",
		"- Run the exact shell command below as your next command. Do not inspect repository files first.",
		fmt.Sprintf("- Write exactly two files now: %q and %q.", docTarget, manifestTarget),
		"- Do not browse for more evidence before this write. Record uncertainty in semantic.coverage.missing/questions.",
		"- Do not write any file outside the exact write_root collect pair.",
		"COLLECT PAIR WRITE COMMAND:",
		steppolicy.CollectEarlyPairWriteCommand(task),
		"Artifact-only recovery contract:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		"- Final action must be: write the overview document and shard-pack-manifest.json under write_root, then exit successfully.",
		"- Exit with code 0 only after shard-pack-manifest.json validates.",
		fmt.Sprintf(`- artifact_root (workspace-relative) = %q`, strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- exact authored document target = %q`, docTarget),
		fmt.Sprintf(`- exact manifest target = %q`, manifestTarget),
		fmt.Sprintf(`- repo_scopes = %q`, strings.Join(task.RepoScopes, ", ")),
		fmt.Sprintf(`- path_scopes = %q`, strings.Join(task.PathScopes, ", ")),
		"COLLECT PAIR RECOVERY CHECKLIST:",
	}
	lines = append(lines, compactCollectManifestValidationChecklist(strings.TrimSpace(task.ArtifactRoot))...)
	lines = append(lines,
		"- Use the heredoc JSON above as the task-specific skeleton. Do not infer schema from prior reports/taskruns artifacts or raw logs.",
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
		"- Execute the preferred heredoc write command, or perform an equivalent single-file overwrite of shard-pack-manifest.json.",
		"- Do not read, diff, or patch an existing invalid shard-pack-manifest.json; replace it.",
		"- Do not search the filesystem for schemas/*, docs/spec/*, examples, prior manifests, sibling shards, raw logs, or reports/taskruns history.",
		"- Do not rewrite authored markdown documents and do not create any file other than shard-pack-manifest.json.",
		"- Treat the heredoc JSON as the complete repair artifact. Do not perform additional factual edits; validation, not stdout, is the success surface.",
	}
}

func collectManifestRepairWriteCommand(writeRoot string, manifestTarget string, skeleton string) string {
	return strings.Join([]string{
		"mkdir -p " + shellSingleQuote(strings.TrimSpace(writeRoot)),
		"cat > " + shellSingleQuote(strings.TrimSpace(manifestTarget)) + " <<'ACP_MANIFEST_JSON'",
		strings.TrimSpace(skeleton),
		"ACP_MANIFEST_JSON",
	}, "\n")
}

func draftArtifactRepairWriteCommand(task acpruntime.Task, manifestTarget string, skeleton string) string {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	draftRoot := strings.TrimSpace(task.DraftFinalRoot)
	lines := []string{
		"mkdir -p " + shellSingleQuote(writeRoot) + " " + shellSingleQuote(draftRoot),
		"cat > " + shellSingleQuote(strings.TrimSpace(manifestTarget)) + " <<'ACP_DRAFT_MANIFEST_JSON'",
		strings.TrimSpace(skeleton),
		"ACP_DRAFT_MANIFEST_JSON",
		"test -s " + shellSingleQuote(strings.TrimSpace(manifestTarget)),
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
		target := filepath.Join(draftRoot, filepath.FromSlash(rel))
		lines = append(lines,
			"mkdir -p "+shellSingleQuote(filepath.Dir(target)),
			"cat > "+shellSingleQuote(target)+" <<'ACP_DRAFT_FILE'",
			draftArtifactRepairFileTemplate(task, output),
			"ACP_DRAFT_FILE",
			"test -s "+shellSingleQuote(target),
		)
	}
	return strings.Join(lines, "\n")
}

func validatorVerdictRepairWriteCommand(writeRoot string, target string, skeleton string) string {
	return strings.Join([]string{
		"mkdir -p " + shellSingleQuote(strings.TrimSpace(writeRoot)),
		"cat > " + shellSingleQuote(strings.TrimSpace(target)) + " <<'ACP_VALIDATOR_VERDICT_JSON'",
		strings.TrimSpace(skeleton),
		"ACP_VALIDATOR_VERDICT_JSON",
	}, "\n")
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
	if strings.HasSuffix(strings.ToLower(output.Path), ".yaml") || strings.HasSuffix(strings.ToLower(output.Path), ".yml") {
		return strings.Join([]string{
			"version: 1",
			"generated_by: acp-runtime-provider-focused-recovery",
			"run_id: " + strings.TrimSpace(task.RunID),
			"step_id: " + strings.TrimSpace(task.StepID),
			"items:",
			"  - id: baseline",
			"    title: Baseline Runtime Draft",
			"    notes: Provider focused recovery wrote the required draft artifact set.",
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
			"- Provider focused recovery wrote this draft artifact under the required draft_final_root.",
		}, "\n")
	case "reports/agent-outputs/architect/summary.md":
		return strings.Join([]string{
			"# Architect Summary",
			"",
			"## Summary",
			"- Provider focused recovery produced the required as-is draft publish surface.",
			"- Review collected shard manifests before treating this diagnostic output as release evidence.",
			"",
			"## Next Checks",
			"- Validate owner mappings and staged findings in the following runtime steps.",
		}, "\n")
	case "reports/changelog/runtime-proposals.md":
		return strings.Join([]string{
			"# Runtime Proposal Changelog",
			"",
			"## Changes",
			"- Provider focused recovery wrote the required proposal changelog draft.",
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
			"- Provider focused recovery wrote this draft artifact to satisfy the runtime draft contract.",
			"- Use collected shard manifests and validator output as the evidence source for final review.",
		}, "\n")
	}
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func ComposeValidatorVerdictRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error) string {
	target := filepath.Join(strings.TrimSpace(task.WriteRoot), "validator-verdict.json")
	skeleton := steppolicy.ValidatorVerdictTaskSkeleton(task)
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in validator verdict focused recovery mode.", provider),
		"Immediate validator verdict repair action:",
		"- Run the exact shell command below as your next command. Do not inspect repository files first.",
		fmt.Sprintf("- Write exactly one file now: %q.", target),
		"- Do not browse for more evidence before this write. The embedded skeleton is the first valid artifact set.",
		"- If validator-verdict.json already exists but is invalid, overwrite it from the heredoc command.",
		"- Copy the heredoc JSON exactly first. Do not make factual edits before the verdict validates.",
		"VALIDATOR VERDICT WRITE COMMAND:",
		validatorVerdictRepairWriteCommand(task.WriteRoot, target, skeleton),
		"Artifact-only recovery contract:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		"- Do not write shard-pack-manifest.json, draft manifests, markdown reports, raw logs, or sibling taskrun files.",
		"- Final action must be: write only write_root/validator-verdict.json, then exit successfully.",
		"- Exit with code 0 only after validator-verdict.json validates.",
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- validator-verdict.json absolute target = %q`, target),
		fmt.Sprintf(`- read_context_roots = %q`, strings.Join(task.ReadContextRoots, ", ")),
		"VALIDATOR VERDICT JSON SKELETON:",
		skeleton,
		"VALIDATOR VERDICT REPAIR INSTRUCTIONS:",
		"- The heredoc JSON is the complete first repair artifact; write it first, then exit if it validates.",
		"- Do not inspect sibling baseline workspaces, prior reports/taskruns history, or raw provider logs as examples.",
		"- If you do inspect staged final artifacts after writing the first valid artifact set, adjust only checked_paths, verdict, summary, fixed_paths, findings, questions, and issues.",
		"- If the only residual gap is missing owner mapping evidence, keep it in findings/questions and use verdict=PASS when there are no technical validator issues.",
		"- If there is a technical validator issue, use verdict=FAIL and encode it only in canonical issues[] shape.",
		"- issues[] items must use only: code, severity, message, path, document_id, citation_id.",
		"- issues[].severity must be error or warning only.",
		"- Legacy issue fields are forbidden inside issues[]: id, title, description, rule_id, related_paths, related_ids, provenance.",
		"VALIDATOR VERDICT CANONICAL SHAPE:",
	}
	lines = append(lines, artifactquality.ValidatorVerdictContractLines()...)
	lines = append(lines,
		"- Canonical fragment below is normative for validator metadata and finding evidence shape; copy keys/types exactly and only change IDs/content.",
		artifactquality.ValidatorVerdictCanonicalExample(),
		"- Canonical issues[] item example:",
		artifactquality.ValidatorVerdictIssueCanonicalExample(),
	)
	if detail := errorText(validationErr); detail != "" {
		lines = append(lines, fmt.Sprintf("- Previous validator artifact validation failure: %s", detail))
	}
	return strings.Join(lines, "\n")
}

func ComposeDraftArtifactRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error) string {
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	manifestTarget := filepath.Join(strings.TrimSpace(task.WriteRoot), manifestFile)
	skeleton := steppolicy.RuntimeDraftManifestTaskSkeleton(task)
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in draft artifact focused recovery mode.", provider),
		"Immediate draft artifact repair action:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		fmt.Sprintf("- Write exactly this manifest target in write_root: %q.", manifestTarget),
		fmt.Sprintf("- Write draft content only under draft_final_root: %q.", strings.TrimSpace(task.DraftFinalRoot)),
		"- Do not begin with broad analysis. Run the preferred file write command below, or perform exactly the same bounded writes.",
		"- If the draft manifest or referenced draft files already exist but are invalid, overwrite them from the heredoc artifacts instead of reading/diffing/patching.",
		"- Copy the heredoc artifacts exactly first. Do not make factual edits before the manifest and draft files validate.",
		"- Do not write shard-pack-manifest.json, validator-verdict.json, raw logs, sibling taskruns, or repository files.",
		"- Final action must be: write the draft manifest plus its referenced draft_final_root files, then exit successfully.",
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- draft_final_root (absolute) = %q`, strings.TrimSpace(task.DraftFinalRoot)),
		fmt.Sprintf(`- manifest_file = %q`, manifestFile),
		fmt.Sprintf(`- step_contract = %q`, strings.TrimSpace(task.StepContract)),
		fmt.Sprintf(`- expected_artifacts = %s`, strings.Join(task.ExpectedArtifacts, ", ")),
		"RUNTIME DRAFT MANIFEST JSON SKELETON:",
		"Use the JSON and draft files embedded in this command as the first valid artifact set:",
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
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		lines = append(lines, "AS-IS DRAFT MANIFEST CANONICAL SHAPE:")
		lines = append(lines, artifactquality.AsIsDraftManifestContractLines()...)
	case "init.step4.proposals", "refresh.step4.proposals":
		lines = append(lines, "PROPOSALS DRAFT MANIFEST CANONICAL SHAPE:")
		lines = append(lines, artifactquality.ProposalsDraftManifestContractLines()...)
	}
	return strings.Join(lines, "\n")
}

func compactCollectManifestValidationChecklist(artifactRoot string) []string {
	return artifactquality.CompactCollectManifestValidationChecklist(artifactRoot)
}

func SharedSections(task acpruntime.Task) []string {
	sections := []string{
		"Artifact-only contract:",
		"- Do not return semantic JSON or any other semantic payload on stdout.",
		"- Write only the required step artifacts into write_root and draft_final_root.",
		"- Stdout/stderr are diagnostics only.",
		steppolicy.DocFirstFilesystemPolicy(task),
	}
	if stepPolicy := strings.TrimSpace(steppolicy.StepSpecificPolicy(task.StepID)); stepPolicy != "" {
		sections = append(sections, stepPolicy)
	}
	switch strings.TrimSpace(task.StepID) {
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		asIsLines := []string{
			"AS-IS DRAFT MANIFEST CANONICAL SHAPE:",
		}
		asIsLines = append(asIsLines, artifactquality.AsIsDraftManifestContractLines()...)
		asIsLines = append(asIsLines,
			`- Canonical fragment below is normative for field names, step_contract, and required outputs; copy keys/types exactly and only change IDs/content.`,
			artifactquality.AsIsDraftManifestCanonicalExample(),
		)
		sections = append(sections, strings.Join(asIsLines, "\n"))
	case "init.step3.findings", "refresh.step3.findings":
		findingsLines := []string{
			"VALIDATOR VERDICT CANONICAL SHAPE:",
		}
		findingsLines = append(findingsLines, artifactquality.ValidatorVerdictContractLines()...)
		findingsLines = append(findingsLines,
			`- Canonical fragment below is normative for validator metadata and finding evidence shape; copy keys/types exactly and only change IDs/content.`,
			artifactquality.ValidatorVerdictCanonicalExample(),
			`- Canonical issues[] item example:`,
			artifactquality.ValidatorVerdictIssueCanonicalExample(),
		)
		sections = append(sections, strings.Join(findingsLines, "\n"))
	case "init.step4.proposals", "refresh.step4.proposals":
		proposalsLines := []string{
			"PROPOSALS DRAFT MANIFEST CANONICAL SHAPE:",
		}
		proposalsLines = append(proposalsLines, artifactquality.ProposalsDraftManifestContractLines()...)
		proposalsLines = append(proposalsLines,
			`- Canonical fragment below is normative for field names, step_contract, allowed publish surface, and forbidden legacy fields; copy keys/types exactly and only change IDs/content.`,
			artifactquality.ProposalsDraftManifestCanonicalExample(),
		)
		sections = append(sections, strings.Join(proposalsLines, "\n"))
	}
	if pack := strings.TrimSpace(steppolicy.WorkspacePromptPackSection(task)); pack != "" {
		sections = append(sections, pack)
	}
	sections = append(sections,
		"Completion rule:",
		"- Exit with code 0 only after required artifacts are fully written.",
		"- Do not emit legacy operation logs or any wrapper envelopes on stdout.",
	)
	return sections
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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
	add := func(path string) {
		path = filepath.ToSlash(strings.TrimSpace(path))
		if path == "" {
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
				add(scope)
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
				add(rel)
				return nil
			})
		}
	}
	sort.Strings(candidates)
	return candidates
}
