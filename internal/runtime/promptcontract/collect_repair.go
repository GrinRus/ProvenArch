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
	sections := []string{
		fmt.Sprintf("You are ACP runtime provider %q in collect manifest repair mode.", provider),
		"FIRST COLLECT MANIFEST REPAIR COMMAND:",
		"Run this exact command as your next filesystem action. Do not analyze, browse, read, diff, patch, or inspect repository files before this write:",
		collectManifestRepairWriteCommand(task.WriteRoot, manifestTarget, skeleton),
		"Immediate repair action:",
		fmt.Sprintf("- Write exactly one file now: %q.", manifestTarget),
		"- Do not begin with broad analysis. The heredoc above is the minimal evidence-backed repair artifact for this shard.",
		"- Do not rewrite existing authored markdown documents.",
		"- If shard-pack-manifest.json already exists but is invalid, do not inspect or patch it; overwrite it from the heredoc command.",
		"- Copy the heredoc JSON during repair and preserve semantic.entities, semantic.edges, semantic.findings, coverage, and evidence repo/path fields.",
		"TASK-SPECIFIC MANIFEST JSON SKELETON:",
		"Use the JSON embedded in the first repair command above as the skeleton and target content.",
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
		`- The task-specific JSON skeleton above is the repair artifact; write it from the heredoc command, preserve its semantic signal, then exit.`,
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
		"- Treat the heredoc JSON as the repair artifact. Preserve embedded evidence-backed semantic entries; validation, not stdout, is the success surface.",
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
