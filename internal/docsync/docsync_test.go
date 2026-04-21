package docsync

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestStakeholderMatrixIsCanonicalSource(t *testing.T) {
	t.Parallel()

	content := readDoc(t, "docs/STAKEHOLDER_DOC.md")
	lower := strings.ToLower(content)

	assertContains(t, content, "Canonical Stakeholder Matrix (source of truth)")
	assertContains(t, content, "Runtime policy `fake` default + `headless` opt-in")
	assertContains(t, content, "Q&A capability without public beta API surface")
	assertContains(t, content, "POST /api/qa/ask")

	if !strings.Contains(lower, "follow-up") && !strings.Contains(lower, "post-beta") {
		t.Fatalf("expected follow-up boundary for /api/qa/ask in stakeholder matrix")
	}
}

func TestCoreDocsReferenceCanonicalStakeholderMatrix(t *testing.T) {
	t.Parallel()

	paths := []string{
		"README.md",
		"docs/ARCHITECTURE.md",
		"docs/PLANS.md",
		"docs/spec/PIPELINE_SPEC.md",
		"docs/spec/API_SPEC.md",
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			lower := strings.ToLower(content)
			if !strings.Contains(content, "docs/STAKEHOLDER_DOC.md") {
				t.Fatalf("expected %s to reference docs/STAKEHOLDER_DOC.md", path)
			}
			if !strings.Contains(lower, "canonical stakeholder matrix") {
				t.Fatalf("expected %s to reference canonical stakeholder matrix", path)
			}
		})
	}
}

func TestRuntimeAndQABoundaryConsistentAcrossDocs(t *testing.T) {
	t.Parallel()

	paths := []string{
		"README.md",
		"docs/ARCHITECTURE.md",
		"docs/spec/API_SPEC.md",
		"docs/spec/PIPELINE_SPEC.md",
	}
	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := strings.ToLower(readDoc(t, path))
			for _, required := range []string{"fake", "headless", "opt-in"} {
				if !strings.Contains(content, required) {
					t.Fatalf("expected %s to mention runtime policy token %q", path, required)
				}
			}
			if !strings.Contains(content, "/api/qa/ask") {
				t.Fatalf("expected %s to mention /api/qa/ask boundary", path)
			}
			if !strings.Contains(content, "follow-up") && !strings.Contains(content, "post-beta") {
				t.Fatalf("expected %s to mark /api/qa/ask as follow-up/post-beta", path)
			}
		})
	}
}

func TestPromptLayerTruthConsistentAcrossCoreDocs(t *testing.T) {
	t.Parallel()

	const mergeOrder = "provider header -> artifact-only/filesystem policy -> step-specific policy -> workspace prompt pack -> provider completion footer"
	paths := []string{
		"docs/ARCHITECTURE.md",
		"docs/spec/PIPELINE_SPEC.md",
		"docs/PLANS.md",
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := strings.ToLower(readDoc(t, path))
			if !strings.Contains(content, mergeOrder) {
				t.Fatalf("expected %s to document exact prompt merge order", path)
			}
			if !strings.Contains(content, "content layer") {
				t.Fatalf("expected %s to describe workspace prompt pack as content layer", path)
			}
			if !strings.Contains(content, "cannot be") && !strings.Contains(content, "не может") && !strings.Contains(content, "не могут") {
				t.Fatalf("expected %s to describe enforced runtime policy invariants", path)
			}
		})
	}
}

func TestPlansRemainActiveOnlyAfterCleanupClosure(t *testing.T) {
	t.Parallel()

	content := readDoc(t, "docs/PLANS.md")
	for _, marker := range []string{
		"EP-20260421-flow-audit-followups",
		"EP-20260421-artifact-only-cleanup-followthrough",
	} {
		if strings.Contains(content, marker) {
			t.Fatalf("expected docs/PLANS.md to stay active-only and exclude closed cleanup plan %q", marker)
		}
	}
	assertContains(t, content, "EP-20260420-regres-small-live-triage")
}

func TestCompatibilityInventoryConsistentAcrossDocs(t *testing.T) {
	t.Parallel()

	paths := []string{
		"docs/ARCHITECTURE.md",
		"docs/TESTING_STRATEGY.md",
		"docs/PLANS.md",
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			assertContains(t, content, "collect.documents_path_normalization")
			assertContains(t, content, "drafts.reconcile_existing_canonical_outputs")
		})
	}
}

func TestKeySurfacesDoNotContainStaleMarkers(t *testing.T) {
	t.Parallel()

	staleByPath := map[string][]string{
		"cmd/acp/main.go": {
			"ACP bootstrap CLI",
			"future local service shell",
			"future init/refresh pipeline",
		},
		"README.md": {
			"bootstrap CLI skeleton",
			"step-scoped runtime provider selection для headless режима: `claude-code` (default fallback) и `qwen-code`",
			"direct-only `claude`/`qwen`",
			"`BATCH_PROVIDER_FILTER`: `all` (default) или CSV из `qwen-code,claude-code`",
			"Headless Runtime Provider (claude-code or qwen-code)",
			"runtime provider adapters (`claude-code`, `qwen-code`)",
		},
		"AGENTS.md": {
			"runtime анализа: **headless multi-provider** (`claude-code` default, `qwen-code` optional) + deterministic `fake` baseline",
		},
		"docs/BACKLOG.md": {
			"orchestrator запускает headless runtime adapter для workspace (`claude-code` default, `qwen-code` optional)",
		},
		"cmd/README.md": {
			"Bootstrap slice",
			"runnable skeleton",
		},
		"docs/DOCS_POLICY.md": {
			"v0.x",
		},
		"docs/STAKEHOLDER_DOC.md": {
			"Горизонт по идеям",
			"**В MVP используем step-scoped headless runtime providers**: `claude-code` (default fallback) и `qwen-code`.",
			"Headless multi-provider runtime (`claude-code` + `qwen-code`)",
		},
		"docs/ARCHITECTURE.md": {
			"headless providers: `claude-code` (`internal/runtime/claudecode`) и `qwen-code` (`internal/runtime/qwencode`)",
			"`claude-code` и `qwen-code` используют shared provider-agnostic step-policy/prompt layer",
		},
		"docs/TESTING_STRATEGY.md": {
			"direct-only runtime commands (`claude`, `qwen`)",
			"expected backend totals from catalog: `regres fast=3`, `regres long=2`, `release fast=8`, `release long=8`, `release full=24`",
			"обязательны оба провайдера (`qwen-code`, `claude-code`)",
		},
		"docs/RELEASE_LIVE_E2E_RUNBOOK.md": {
			"BATCH_PROVIDER_FILTER=qwen-code|claude-code",
		},
		"docs/PLANS.md": {
			"provider choice (`claude-code` default, `qwen-code`), persisted runtime execution metadata",
		},
		".agents/skills/e2e-live-gate/SKILL.md": {
			"BATCH_PROVIDER_FILTER=qwen-code|claude-code",
		},
		"internal/reports/README.md": {
			"MVP placeholder",
			"Next milestones",
		},
		"internal/api/ui_dist/README.md": {
			"Здесь будут embed-ассеты UI",
		},
		"internal/orchestrator/README.md": {
			"package orchestrator",
			"MVP baseline implementation.",
		},
		"internal/runtime/claudecode/README.md": {
			"package claudecode",
			"MVP baseline implementation.",
		},
		"internal/model/README.md": {
			"package model",
			"MVP baseline implementation.",
		},
	}

	for path, staleMarkers := range staleByPath {
		path := path
		staleMarkers := staleMarkers
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			for _, marker := range staleMarkers {
				if strings.Contains(content, marker) {
					t.Fatalf("expected %s to avoid stale marker %q", path, marker)
				}
			}
		})
	}
}

func TestCLIDocsSurfaceMatchesHelp(t *testing.T) {
	t.Parallel()

	helpSource := readDoc(t, "cmd/acp/main.go")
	helpTokens := []string{
		"acp init-workspace --workspace <abs-path> ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path ./docs/imports]",
		"acp serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--auto-init ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path ./docs/imports]]",
		"acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]",
		"acp qa --workspace <abs-path> --question \\\"<text>\\\"",
	}
	for _, token := range helpTokens {
		if !strings.Contains(helpSource, token) {
			t.Fatalf("expected cmd/acp/main.go to include help token %q", token)
		}
	}

	requiredByDoc := map[string][]string{
		"README.md": {
			"acp init-workspace --workspace /path/to/arch-workspace --repo-name payments-service --repo-path /path/to/payments-service",
			"acp serve --workspace /path/to/arch-workspace --auto-init --repo-name payments-service --repo-path /path/to/payments-service --runtime fake",
			"acp init-workspace --workspace /path/to/arch-workspace --repos-file /path/to/repos.yaml",
			"acp serve --workspace /path/to/arch-workspace --auto-init --repos-file /path/to/repos.yaml --runtime fake",
			"acp run --workspace /path/to/arch-workspace --pipeline init --runtime fake --non-interactive",
			"acp qa --workspace /path/to/arch-workspace --question \"Who owns payments-service?\"",
		},
		"cmd/acp/README.md": {
			"acp init-workspace --workspace <abs-path> ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>)",
			"acp serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--auto-init ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path ./docs/imports]]",
			"acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]",
			"acp qa --workspace <abs-path> --question",
		},
		"docs/ARCHITECTURE.md": {
			"`init-workspace --workspace <abs-path> ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>)`",
			"`serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--auto-init ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path <path>]]`",
			"run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]",
			"`acp qa`",
		},
		"docs/spec/API_SPEC.md": {
			"acp init-workspace --workspace <abs-path> ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>)",
			"acp serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--auto-init ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path <path>]]",
			"acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]",
		},
	}
	for path, tokens := range requiredByDoc {
		path := path
		tokens := tokens
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			for _, token := range tokens {
				if !strings.Contains(content, token) {
					t.Fatalf("expected %s to include CLI docs token %q", path, token)
				}
			}
		})
	}
}

func TestGeneratedArtifactsPolicyIsDocumented(t *testing.T) {
	t.Parallel()

	policy := readDoc(t, "docs/BASELINE_POLICY.md")
	assertContains(t, policy, "internal/api/ui_dist/*")
	assertContains(t, policy, "fixtures/scenarios/*/golden/readable/*")
	assertContains(t, policy, "make build")

	uiDist := readDoc(t, "internal/api/ui_dist/README.md")
	assertContains(t, uiDist, "tracked embed-ассеты UI")
	assertContains(t, uiDist, "make build")

	fixtures := readDoc(t, "fixtures/README.md")
	assertContains(t, fixtures, "fixtures/scenarios/*/golden/readable/*")
	assertContains(t, fixtures, "human-readable deterministic export")
}

func TestActiveSurfacesRejectLegacyArtifactOnlyMarkers(t *testing.T) {
	t.Parallel()

	legacyMarkers := []string{
		joinLegacyMarker("Task", "Result"),
		"schemas/" + "taskresult" + ".schema.json",
		"change" + "set",
		"add" + "_doc_" + "artifact",
		"upsert" + "_entity",
		"upsert" + "_edge",
		"add" + "_finding",
	}
	allowPrefixes := []string{
		"docs/archive/",
		"fixtures/legacy-rejection/",
		"internal/runtime/testdata/legacy-rejection/",
		"internal/api/ui_dist/",
	}
	roots := []string{
		"README.md",
		"docs",
		"cmd",
		"internal",
		"ui/src",
		"scripts",
	}

	for _, rel := range collectRepoTextFiles(t, roots, allowPrefixes) {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, rel)
			for _, marker := range legacyMarkers {
				if strings.Contains(content, marker) {
					t.Fatalf("expected active surface %s to avoid legacy marker %q", rel, marker)
				}
			}
		})
	}
}

func TestMultiRepoE2EDocsAreConsistent(t *testing.T) {
	t.Parallel()

	required := map[string][]string{
		"README.md": {
			"TARGET_REPOS_FILE",
			"E2E_MATRIX_FILE",
			"single-path",
			"single-git_url",
			"multi-path",
			"multi-git_url",
			"UI_E2E_EXPECTED_REPO_COUNT",
			"required ci gate",
			"pinned",
		},
		"docs/LOCAL_FULL_RUN_AI_ADVENT.md": {
			"TARGET_REPOS_FILE",
			"E2E_MATRIX_FILE",
			"single-path",
			"single-git_url",
			"multi-path",
			"multi-git_url",
			"analysis:cross-repo-missing",
			"trusted machine",
		},
		"docs/TESTING_STRATEGY.md": {
			"TARGET_REPOS_FILE",
			"E2E_MATRIX_FILE",
			"analysis:cross-repo-missing",
			"single-path",
			"single-git_url",
			"multi-path",
			"multi-git_url",
			"UI_E2E_EXPECTED_REPO_COUNT",
		},
	}

	for path, tokens := range required {
		path := path
		tokens := tokens
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			lower := strings.ToLower(content)
			for _, token := range tokens {
				if strings.HasPrefix(token, "analysis:") {
					assertContains(t, content, token)
					continue
				}
				if !strings.Contains(lower, strings.ToLower(token)) {
					t.Fatalf("expected %s to include token %q", path, token)
				}
			}
		})
	}
}

func assertContains(t *testing.T, content string, needle string) {
	t.Helper()
	if !strings.Contains(content, needle) {
		t.Fatalf("expected document content to include %q", needle)
	}
}

func readDoc(t *testing.T, rel string) string {
	t.Helper()
	path := filepath.Join(repoRoot(t), filepath.FromSlash(rel))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(content)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func collectRepoTextFiles(t *testing.T, roots []string, allowPrefixes []string) []string {
	t.Helper()

	repo := repoRoot(t)
	files := []string{}
	seen := map[string]struct{}{}

	addFile := func(rel string) {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" {
			return
		}
		if strings.Contains(rel, "/__pycache__/") || strings.HasSuffix(rel, ".pyc") {
			return
		}
		for _, prefix := range allowPrefixes {
			if strings.HasPrefix(rel, prefix) {
				return
			}
		}
		if _, exists := seen[rel]; exists {
			return
		}
		seen[rel] = struct{}{}
		files = append(files, rel)
	}

	for _, rel := range roots {
		full := filepath.Join(repo, filepath.FromSlash(rel))
		info, err := os.Stat(full)
		if err != nil {
			t.Fatalf("stat %s: %v", rel, err)
		}
		if !info.IsDir() {
			addFile(rel)
			continue
		}
		if err := filepath.Walk(full, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				relPath, err := filepath.Rel(repo, path)
				if err != nil {
					return err
				}
				relPath = filepath.ToSlash(relPath)
				for _, prefix := range allowPrefixes {
					if strings.HasPrefix(relPath+"/", prefix) {
						return filepath.SkipDir
					}
				}
				return nil
			}
			relPath, err := filepath.Rel(repo, path)
			if err != nil {
				return err
			}
			addFile(relPath)
			return nil
		}); err != nil {
			t.Fatalf("walk %s: %v", rel, err)
		}
	}

	sort.Strings(files)
	return files
}

func joinLegacyMarker(parts ...string) string {
	return strings.Join(parts, "")
}
