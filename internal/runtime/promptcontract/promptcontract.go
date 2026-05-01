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

func ComposeArtifactOnlyPrompt(provider acpruntime.Provider, task acpruntime.Task) string {
	sections := []string{
		fmt.Sprintf("You are ACP runtime provider %q.", provider),
	}
	sections = append(sections, SharedSections(task)...)
	return strings.Join(sections, "\n\n")
}

func ComposeCollectManifestRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error) string {
	sections := []string{
		fmt.Sprintf("You are ACP runtime provider %q in collect manifest repair mode.", provider),
		"Artifact-only repair contract:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		"- Do not rewrite existing authored markdown documents.",
		"- Write or replace only write_root/shard-pack-manifest.json.",
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
		"COLLECT MANIFEST REPAIR INSTRUCTIONS:",
		"- Existing authored documents in write_root are the source surface to describe.",
		"- Do not search the filesystem for schemas/*, docs/spec/*, examples, or prior manifests; the schema embedded in this prompt is authoritative for this repair.",
		"- Do not inspect reports/taskruns outside the current write_root and do not read sibling shard manifests, prior manifests, or raw logs as examples.",
		"- Reuse existing repository evidence from read_context_roots and path scopes; do not perform broad new exploration.",
		"- If evidence is sparse, keep the gap explicit in semantic.coverage.missing instead of inventing unsupported entities.",
	}
	authoredDocs := authoredRepairDocuments(task.WriteRoot)
	if len(authoredDocs) > 0 {
		repairLines = append(repairLines, "Existing authored document files in write_root:")
		for _, rel := range authoredDocs {
			repairLines = append(repairLines, fmt.Sprintf("- %s", rel))
		}
	}
	evidencePaths := repairEvidenceCandidates(task)
	if len(evidencePaths) > 0 {
		repairLines = append(repairLines, "Use these repository evidence path candidates before any broader lookup:")
		for _, rel := range evidencePaths {
			repairLines = append(repairLines, fmt.Sprintf("- %s", rel))
		}
	}
	repairLines = append(repairLines, collectManifestRepairScaffold(task, authoredDocs, evidencePaths)...)
	repairLines = append(repairLines, steppolicy.CollectArtifactRepairHints(errorText(validationErr))...)
	repairLines = append(repairLines,
		"COLLECT MANIFEST CANONICAL SHAPE:",
	)
	repairLines = append(repairLines, artifactquality.CollectManifestContractLines(strings.TrimSpace(task.ArtifactRoot))...)
	repairLines = append(repairLines,
		`- Repair-mode note: if schemas/* or docs/spec/* are absent from the runtime workspace, do not look for them; use this embedded canonical fragment and contract text.`,
		`- Canonical fragment below is normative for field names and value types; copy keys/types exactly and only change IDs/content.`,
		artifactquality.CollectManifestCanonicalExample(),
	)
	sections = append(sections, strings.Join(repairLines, "\n"))
	return strings.Join(sections, "\n\n")
}

func collectManifestRepairScaffold(task acpruntime.Task, authoredDocs []string, evidencePaths []string) []string {
	lines := []string{
		"TASK-SPECIFIC MANIFEST SCAFFOLD:",
		"- Start from the canonical fragment below, but copy these exact metadata values:",
		`  - version: 1`,
		fmt.Sprintf(`  - run_id: %q`, strings.TrimSpace(task.RunID)),
		fmt.Sprintf(`  - step_id: %q`, strings.TrimSpace(task.StepID)),
		fmt.Sprintf(`  - shard_id: %q`, strings.TrimSpace(task.ShardID)),
		fmt.Sprintf(`  - domain_id: %q`, strings.TrimSpace(task.DomainID)),
		fmt.Sprintf(`  - agent_role: %q`, strings.TrimSpace(task.AgentRole)),
		fmt.Sprintf(`  - artifact_root: %q`, strings.TrimSpace(task.ArtifactRoot)),
		fmt.Sprintf(`  - repo_scopes: %s`, jsonStringList(task.RepoScopes)),
		fmt.Sprintf(`  - path_scopes: %s`, jsonStringList(task.PathScopes)),
	}
	if len(authoredDocs) > 0 {
		lines = append(lines,
			fmt.Sprintf(`- Create one documents[] item per authored file; documents[].path MUST be exactly one of: %s.`, strings.Join(authoredDocs, ", ")),
			`- documents[].canonical_path must be a stable promoted reports/as-is or reports/agent-outputs path, never a reports/taskruns staging path.`,
		)
	} else {
		lines = append(lines, `- No authored document file was listed; do not invent markdown in repair mode. Write the manifest only if authored docs are visible in write_root.`)
	}
	if len(evidencePaths) > 0 {
		lines = append(lines,
			fmt.Sprintf(`- Build citations[] from real repository evidence; citations[].path should use one of these candidates when applicable: %s.`, strings.Join(evidencePaths, ", ")),
			`- Link documents[].citation_ids to citations[].id and keep every citation document_ids[] pointed back to an authored document id.`,
		)
	} else {
		lines = append(lines, `- If no evidence candidate is listed, use concrete repo evidence from the assigned path scopes before finalizing; do not leave placeholder citations or citation-only semantic evidence.`)
	}
	lines = append(lines,
		`- Keep semantic.coverage/questions/entities/edges/findings present; empty arrays are allowed for questions/entities/edges/findings when evidence is sparse.`,
		`TASK-SPECIFIC MANIFEST JSON SKELETON:`,
		steppolicy.CollectManifestTaskSkeleton(task, authoredDocs, evidencePaths),
		`- Copy the JSON skeleton into write_root/shard-pack-manifest.json, preserve exact task metadata, then adjust document/citation/semantic content to match the authored docs and real evidence.`,
		`- Final repair action: write write_root/shard-pack-manifest.json, verify no other file changed, then exit successfully.`,
	)
	return lines
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
	if acpruntime.IsCollectStep(task.StepID) {
		collectLines := []string{
			"COLLECT MANIFEST CANONICAL SHAPE:",
		}
		collectLines = append(collectLines, artifactquality.CollectManifestContractLines(strings.TrimSpace(task.ArtifactRoot))...)
		collectLines = append(collectLines,
			`- Canonical fragment below is normative for field names and value types; copy keys/types exactly and only change IDs/content.`,
			artifactquality.CollectManifestCanonicalExample(),
		)
		sections = append(sections, strings.Join(collectLines, "\n"))
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
