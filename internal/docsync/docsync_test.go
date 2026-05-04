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
	assertContains(t, content, "Q&A capability with CLI + public read-only beta API surface")
	assertContains(t, content, "POST /api/qa/ask")

	if !strings.Contains(lower, "public `post /api/qa/ask` | done") {
		t.Fatalf("expected stakeholder matrix to mark /api/qa/ask as done")
	}
	if strings.Contains(lower, "not in current beta api surface") {
		t.Fatalf("expected stakeholder matrix not to mark /api/qa/ask outside beta surface")
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
			if !strings.Contains(content, "read-only") && !strings.Contains(content, "read only") {
				t.Fatalf("expected %s to mark /api/qa/ask as read-only", path)
			}
			if !strings.Contains(content, "headless") {
				t.Fatalf("expected %s to keep non-headless Q&A boundary", path)
			}
			if strings.Contains(content, "/api/qa/ask не входит") || strings.Contains(content, "/api/qa/ask remains follow-up") {
				t.Fatalf("expected %s not to mark /api/qa/ask as outside beta surface", path)
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

func TestBaselineBundleInventoryDocumentsQASkill(t *testing.T) {
	t.Parallel()

	code := readDoc(t, "internal/workspace/baseline.go")
	assertContains(t, code, `"qa",`)
	assertContains(t, code, "skills: [qa]")

	requiredSkillsLineByPath := map[string]string{
		"README.md":               "skills: `service-inventory`, `interface-extraction`, `integration-mapping`, `datastore-mapping`, `cicd-mapping`, `ownership-coverage`, `findings`, `proposals`, `qa`",
		"docs/STAKEHOLDER_DOC.md": "skills: `service-inventory`, `interface-extraction`, `integration-mapping`, `datastore-mapping`, `cicd-mapping`, `ownership-coverage`, `findings`, `proposals`, `qa`",
		"docs/BACKLOG.md":         "baseline skills фиксированы: `service-inventory`, `interface-extraction`, `integration-mapping`, `datastore-mapping`, `cicd-mapping`, `ownership-coverage`, `findings`, `proposals`, `qa`",
	}
	for path, required := range requiredSkillsLineByPath {
		path := path
		required := required
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			assertContains(t, readDoc(t, path), required)
		})
	}

	pipelineBaseline := extractBetween(t, readDoc(t, "docs/spec/PIPELINE_SPEC.md"), "## Baseline Agents/Skills/Prompts (MVP)", "Bundle поставляется")
	assertContains(t, pipelineBaseline, "- skills:")
	assertContains(t, pipelineBaseline, "  - `qa`")
	assertContains(t, pipelineBaseline, "- prompt packs:")
	assertContains(t, pipelineBaseline, "  - `qa`")
}

func TestQABetaBoundaryDocumentsDeterministicService(t *testing.T) {
	t.Parallel()

	requiredByPath := map[string]string{
		"README.md":                  "deterministic workspace-backed read-only service + CLI `acp qa` + public read-only `POST /api/qa/ask`; это не headless runtime agent",
		"docs/ARCHITECTURE.md":       "deterministic workspace-backed read-only service + CLI `acp qa` + `POST /api/qa/ask`; не headless runtime agent",
		"docs/STAKEHOLDER_DOC.md":    "deterministic workspace-backed read-only capability доступна как internal service + CLI `acp qa` + public read-only `POST /api/qa/ask`; это не headless runtime agent",
		"docs/spec/PIPELINE_SPEC.md": "deterministic workspace-backed read-only service + CLI `acp qa` + public read-only `POST /api/qa/ask`",
		"docs/BACKLOG.md":            "deterministic workspace-backed read-only service + CLI `acp qa` + public read-only `POST /api/qa/ask`",
	}
	for path, required := range requiredByPath {
		path := path
		required := required
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			assertContains(t, content, required)
			assertContains(t, content, "POST /api/qa/ask")
			assertContains(t, content, "skills/prompt-packs/qa.md")
			assertNotContains(t, content, "System Analyst Q&A Agent")
		})
	}
}

func TestQAPublicAPIDocsMatchImplementedRoute(t *testing.T) {
	t.Parallel()

	apiSpec := readDoc(t, "docs/spec/API_SPEC.md")
	server := readDoc(t, "internal/api/server.go")
	assertContains(t, server, `mux.HandleFunc("/api/qa/ask", s.handleQAAsk)`)
	assertContains(t, apiSpec, "### POST `/api/qa/ask`")
	assertContains(t, apiSpec, "question_required")
	assertContains(t, apiSpec, "qa_failed")
	assertContains(t, apiSpec, "does not call headless runtime providers, git helpers or pipeline runs")
}

func TestSCMWebhookBoundaryDocumentsExternalCIOnly(t *testing.T) {
	t.Parallel()

	requiredByPath := map[string]string{
		"README.md":                  "ACP не принимает public SCM webhooks сам: native webhook listener / external SCM app integration остаются вне MVP",
		"docs/spec/PIPELINE_SPEC.md": "Webhook принимает CI provider, не ACP: native SCM webhook listener / external SCM app integration остаются вне MVP",
		"docs/BACKLOG.md":            "SCM hooks обрабатываются CI provider; native ACP webhook listener / external SCM app integration остаются вне MVP",
		"docs/ARCHITECTURE.md":       "Native GitHub/GitLab webhook listener, hosted control plane and external SCM app integration остаются вне MVP",
		"docs/spec/API_SPEC.md":      "Native SCM webhook listener/hosted control plane остаются вне MVP",
	}
	for path, required := range requiredByPath {
		path := path
		required := required
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			assertContains(t, readDoc(t, path), required)
		})
	}
}

func TestDocsImportsMetadataIndexContractDocumented(t *testing.T) {
	t.Parallel()

	requiredByPath := map[string]string{
		"README.md":                   "<docs.imports_path>/index.yaml",
		"docs/spec/WORKSPACE_SPEC.md": "Canonical metadata index: `<imports_path>/index.yaml`",
		"docs/spec/PIPELINE_SPEC.md":  "<docs.imports_path>/index.yaml",
		"docs/STAKEHOLDER_DOC.md":     "<docs.imports_path>/index.yaml",
	}
	for path, required := range requiredByPath {
		path := path
		required := required
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			assertContains(t, content, required)
			assertContains(t, content, "warning")
			assertContains(t, content, "id")
			assertContains(t, content, "path")
		})
	}
}

func TestUIBaselineEditorPromptPackHintMatchesStepPolicy(t *testing.T) {
	t.Parallel()

	content := readDoc(t, "ui/src/components/BaselineEditorsPanel.tsx")
	assertContains(t, content, "step0/step1/step3/step4")
	assertContains(t, content, "step2 uses enforced as-is policy without an editable prompt pack")
	assertNotContains(t, content, "collect`/`findings")
}

func TestPromptPackCoverageDocumentsStep2Boundary(t *testing.T) {
	t.Parallel()

	policy := readDoc(t, "internal/runtime/steppolicy/policy.go")
	packFunc := extractBetween(t, policy, "func WorkspacePromptPackPath(stepID string) string {", "func WorkspacePromptPackSection")
	for _, expected := range []string{
		"skills/prompt-packs/constitution.md",
		"skills/prompt-packs/collect-context.md",
		"skills/prompt-packs/findings.md",
		"skills/prompt-packs/proposals.md",
	} {
		assertContains(t, packFunc, expected)
	}
	assertNotContains(t, packFunc, "step2.asis_docs")
	assertNotContains(t, packFunc, "as-is.md")

	required := map[string][]string{
		"README.md": {
			"Editable prompt pack layer",
			"`step2.asis_docs` использует enforced policy only и не имеет отдельного editable `as-is` prompt pack",
		},
		"docs/ARCHITECTURE.md": {
			"Editable prompt pack layer",
			"`step2.asis_docs` остаётся enforced-policy-only и не имеет отдельного editable `as-is` prompt pack",
		},
		"docs/spec/PIPELINE_SPEC.md": {
			"Editable prompt pack layer",
			"`step2.asis_docs` работает через enforced policy only без отдельного editable `as-is` prompt pack",
			"`step2.asis_docs` не имеет отдельного editable workspace prompt pack",
		},
	}
	for path, tokens := range required {
		path := path
		tokens := tokens
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			for _, token := range tokens {
				assertContains(t, content, token)
			}
		})
	}
}

func TestArtifactOwnershipTaxonomyDocumented(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"README.md", "docs/ARCHITECTURE.md", "docs/spec/PIPELINE_SPEC.md"} {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			for _, token := range []string{"provider-authored", "orchestrator-authored", "compiler-derived"} {
				assertContains(t, content, token)
			}
		})
	}
}

func TestActivePlansHaveOpenGoals(t *testing.T) {
	t.Parallel()

	active := extractAfter(t, readDoc(t, "docs/PLANS.md"), "## Active Plans")
	for _, section := range splitPlanSections(active) {
		lines := strings.Split(section, "\n")
		if len(lines) < 2 {
			continue
		}
		planID := strings.TrimSpace(lines[1])
		goals := extractBetween(t, section, "### Goals (must have)", "\n### ")
		checked := strings.Count(goals, "- [x]")
		open := strings.Count(goals, "- [ ]")
		if checked > 0 && open == 0 {
			t.Fatalf("expected active plan %s to keep at least one open goal or move to docs/archive", planID)
		}
	}
}

func TestPlansRemainActiveOnlyAfterCleanupClosure(t *testing.T) {
	t.Parallel()

	content := readDoc(t, "docs/PLANS.md")
	if strings.Contains(content, "cleanup/refactor closure") {
		t.Fatalf("expected docs/PLANS.md active section to exclude closed cleanup closure notes")
	}
	for _, marker := range []string{
		"EP-20260421-repo-garbage-audit",
		"EP-20260421-repo-cleanup-pr1-pr2",
		"EP-20260421-flow-audit-followups",
		"EP-20260421-artifact-only-cleanup-followthrough",
	} {
		if strings.Contains(content, marker) {
			t.Fatalf("expected docs/PLANS.md to stay active-only and exclude closed cleanup plan %q", marker)
		}
	}
	assertContains(t, content, "EP-20260420-regres-small-live-triage")
}

func TestDocsDoNotAdvertiseActiveCompatibilityInventory(t *testing.T) {
	t.Parallel()

	paths := []string{
		"docs/ARCHITECTURE.md",
		"docs/TESTING_STRATEGY.md",
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			assertNotContains(t, content, "collect.documents_path_normalization")
			assertNotContains(t, content, "drafts.reconcile_existing_canonical_outputs")
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

func TestCLIHelpSurfaceMatchesCommands(t *testing.T) {
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
}

func TestCLIDocsPointToCanonicalSources(t *testing.T) {
	t.Parallel()

	requiredByDoc := map[string][]string{
		"cmd/acp/README.md": {
			"acp --help",
			"acp <command> --help",
			"cmd/acp/main.go",
			"README.md",
			"docs/spec/API_SPEC.md",
			"docs/ARCHITECTURE.md",
		},
		"docs/spec/API_SPEC.md": {
			"HTTP API wire-contract",
			"cmd/acp/main.go",
			"README.md",
			"CLI batch mode",
		},
		"docs/ARCHITECTURE.md": {
			"acp --help",
			"cmd/acp/main.go",
			"behavior boundary",
			"local API+UI service",
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
					t.Fatalf("expected %s to include canonical-source token %q", path, token)
				}
			}
		})
	}

	forbiddenByDoc := map[string][]string{
		"cmd/acp/README.md": {
			"acp serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code]",
			"acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code]",
			"--max-parallel-tasks <n>",
			"--run-logs-ttl-hours",
		},
		"docs/spec/API_SPEC.md": {
			"acp serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--auto-init ... [--docs-imports-path <path>]]",
			"ACP_RUNTIME_STEP_TIMEOUT_SEC",
			"ACP_EXECUTION_STRATEGY",
		},
		"docs/ARCHITECTURE.md": {
			"serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort]",
			"run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--runtime-provider claude-code|qwen-code|codex-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]",
			"ACP_RUNTIME_*",
			"--max-parallel-tasks <n>",
		},
	}
	for path, tokens := range forbiddenByDoc {
		path := path
		tokens := tokens
		t.Run(path+"-forbidden", func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			for _, token := range tokens {
				if strings.Contains(content, token) {
					t.Fatalf("expected %s to avoid duplicated detail token %q", path, token)
				}
			}
		})
	}
}

func TestREADMEFullRunSectionPointsToRunbooks(t *testing.T) {
	t.Parallel()

	content := readDoc(t, "README.md")
	for _, required := range []string{
		"docs/LOCAL_FULL_RUN_AI_ADVENT.md",
		"docs/RELEASE_LIVE_E2E_RUNBOOK.md",
		"TARGET_REPOS_FILE",
		"E2E_MATRIX_FILE",
		"scripts/full-run-ai-advent.sh",
		"scripts/full-run-batch-matrix.sh",
	} {
		assertContains(t, content, required)
	}
	for _, forbidden := range []string{
		"BATCH_PROVIDER_FILTER",
		"UI_E2E_CANCEL_STUB_SLEEP_SEC",
		"profile-status/*.json",
		"parallel-default",
	} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("expected README.md full-run section to stay high-level and avoid %q", forbidden)
		}
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

func TestArtifactFixtureTerminologyIsConsistent(t *testing.T) {
	t.Parallel()

	required := map[string][]string{
		"README.md":                             {"synthetic fixtures and recorded artifacts", "artifact fixtures"},
		"docs/ARCHITECTURE.md":                  {"artifact fixtures"},
		"docs/spec/API_SPEC.md":                 {"artifact fixtures"},
		"docs/TESTING_STRATEGY.md":              {"artifact fixtures", "recorded artifacts"},
		"docs/BACKLOG.md":                       {"fake runner + artifact fixtures", "recorded artifacts"},
		"fixtures/README.md":                    {"recorded artifacts"},
		"internal/runtime/claudecode/README.md": {"artifact fixtures"},
	}
	forbidden := []string{
		"recordedrunner",
		"recorded runner outputs",
		"recorded runner harness",
		"recorded runtime outputs",
		"fake/recorded runner",
	}

	for path, tokens := range required {
		path := path
		tokens := tokens
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			content := strings.ToLower(readDoc(t, path))
			for _, marker := range forbidden {
				if strings.Contains(content, marker) {
					t.Fatalf("expected %s to avoid stale recorded-runner marker %q", path, marker)
				}
			}
			found := false
			for _, token := range tokens {
				if strings.Contains(content, strings.ToLower(token)) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected %s to use artifact-fixture terminology", path)
			}
		})
	}
}

func TestActiveDocsAvoidStaleRecordedRunnerTerminologyRepoWide(t *testing.T) {
	t.Parallel()

	forbidden := []string{
		"recordedrunner",
		"recorded runner outputs",
		"recorded runner harness",
		"recorded runtime outputs",
		"fake/recorded runner",
	}
	roots := []string{
		"README.md",
		"cmd",
		"docs",
		"internal",
	}
	allowPrefixes := []string{
		"docs/archive/",
		"internal/api/ui_dist/",
	}

	for _, rel := range collectRepoMarkdownFiles(t, roots, allowPrefixes) {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			content := strings.ToLower(readDoc(t, rel))
			for _, marker := range forbidden {
				if strings.Contains(content, marker) {
					t.Fatalf("expected active doc %s to avoid stale recorded-runner marker %q", rel, marker)
				}
			}
		})
	}
}

func TestPackageDocsStayAligned(t *testing.T) {
	t.Parallel()

	assertPathMissing(t, "cmd/README.md")
	assertPathMissing(t, "internal/README.md")

	reportsReadme := readDoc(t, "internal/reports/README.md")
	assertContains(t, reportsReadme, "`reports/diagrams/*`")
	assertContains(t, reportsReadme, "render-context")
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
		"fixtures/contract-rejection/",
		"internal/runtime/testdata/contract-rejection/",
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
			"docs/LOCAL_FULL_RUN_AI_ADVENT.md",
			"docs/RELEASE_LIVE_E2E_RUNBOOK.md",
			"TARGET_REPOS_FILE",
			"E2E_MATRIX_FILE",
			"docs/LOCAL_FULL_RUN_AI_ADVENT.md",
			"docs/RELEASE_LIVE_E2E_RUNBOOK.md",
			"trusted-machine",
		},
		"docs/LOCAL_FULL_RUN_AI_ADVENT.md": {
			"docs/RELEASE_LIVE_E2E_RUNBOOK.md",
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
			"docs/LOCAL_FULL_RUN_AI_ADVENT.md",
			"docs/RELEASE_LIVE_E2E_RUNBOOK.md",
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
	forbidden := map[string][]string{
		"README.md": {
			"MATRIX_ID=release-fast-",
			"ACP_APPLY_TIMEOUTS_VIA_API=1",
		},
		"docs/LOCAL_FULL_RUN_AI_ADVENT.md": {
			"MATRIX_ID=release-fast-",
			"`release full` = `24` backend runs total",
		},
		"docs/TESTING_STRATEGY.md": {
			"MATRIX_ID=release-fast-",
			"ACP_APPLY_TIMEOUTS_VIA_API=1",
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

	for path, markers := range forbidden {
		path := path
		markers := markers
		t.Run(path+"/forbidden", func(t *testing.T) {
			t.Parallel()
			content := readDoc(t, path)
			for _, marker := range markers {
				assertNotContains(t, content, marker)
			}
		})
	}
}

func extractBetween(t *testing.T, content string, start string, end string) string {
	t.Helper()
	startIdx := strings.Index(content, start)
	if startIdx < 0 {
		t.Fatalf("expected content to include start marker %q", start)
	}
	afterStart := content[startIdx+len(start):]
	endIdx := strings.Index(afterStart, end)
	if endIdx < 0 {
		t.Fatalf("expected content after %q to include end marker %q", start, end)
	}
	return afterStart[:endIdx]
}

func extractAfter(t *testing.T, content string, marker string) string {
	t.Helper()
	idx := strings.Index(content, marker)
	if idx < 0 {
		t.Fatalf("expected content to include marker %q", marker)
	}
	return content[idx+len(marker):]
}

func splitPlanSections(content string) []string {
	parts := strings.Split(content, "\n### Plan ID\n")
	sections := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		sections = append(sections, "### Plan ID\n"+part)
	}
	return sections
}

func assertContains(t *testing.T, content string, needle string) {
	t.Helper()
	if !strings.Contains(content, needle) {
		t.Fatalf("expected document content to include %q", needle)
	}
}

func assertNotContains(t *testing.T, content string, needle string) {
	t.Helper()
	if strings.Contains(content, needle) {
		t.Fatalf("expected document content to avoid %q", needle)
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

func assertPathMissing(t *testing.T, rel string) {
	t.Helper()
	path := filepath.Join(repoRoot(t), filepath.FromSlash(rel))
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected %s to stay absent", rel)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", rel, err)
	}
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

func collectRepoMarkdownFiles(t *testing.T, roots []string, allowPrefixes []string) []string {
	t.Helper()

	all := collectRepoTextFiles(t, roots, allowPrefixes)
	docs := make([]string, 0, len(all))
	for _, rel := range all {
		if strings.HasSuffix(strings.ToLower(rel), ".md") {
			docs = append(docs, rel)
		}
	}
	return docs
}

func joinLegacyMarker(parts ...string) string {
	return strings.Join(parts, "")
}
