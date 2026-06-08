package orchestrator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/reports"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestAssessRefreshArtifactWarningsFlagsBankLikeCollapseFixture(t *testing.T) {
	t.Parallel()

	manifests, finalIndex, citationIndex, verdict := loadRefreshArtifactFixtureSet(
		t,
		filepath.Join("..", "..", "fixtures", "scenarios", "refresh-artifact-quality", "bank-collapse"),
	)
	if verdict.Verdict != "PASS" {
		t.Fatalf("expected canonical PASS validator verdict in frozen bank fixture, got %q", verdict.Verdict)
	}

	warnings := assessRefreshArtifactWarnings(manifests, finalIndex, citationIndex)
	if len(warnings) != 2 {
		t.Fatalf("expected 2 artifact-quality warnings, got %d: %#v", len(warnings), warnings)
	}
	if !containsArtifactWarning(warnings, "only 1 generic runtime-summary citation") {
		t.Fatalf("expected runtime-summary collapse warning, got %#v", warnings)
	}
	if !containsArtifactWarning(warnings, "reuse-only and preserve no repo-specific citations") {
		t.Fatalf("expected reuse-only warning, got %#v", warnings)
	}
}

func TestAssessRefreshArtifactWarningsAllowsOpenstackRichReuseFixture(t *testing.T) {
	t.Parallel()

	manifests, finalIndex, citationIndex, verdict := loadRefreshArtifactFixtureSet(
		t,
		filepath.Join("..", "..", "fixtures", "scenarios", "refresh-artifact-quality", "openstack-rich-reuse"),
	)
	if verdict.Verdict != "PASS" {
		t.Fatalf("expected canonical PASS validator verdict in frozen openstack fixture, got %q", verdict.Verdict)
	}

	warnings := assessRefreshArtifactWarnings(manifests, finalIndex, citationIndex)
	if len(warnings) != 0 {
		t.Fatalf("expected no artifact-quality warnings for acceptable rich reuse fixture, got %#v", warnings)
	}
}

func TestRuntimeDiagnosticCountersSurfaceRepairStallPressure(t *testing.T) {
	t.Parallel()

	execution := &pipelineExecution{}
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "retry scheduled",
		Fields: map[string]any{
			"action":      "terminate_and_validate",
			"stall_phase": "pre_artifact",
		},
	})
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "retry scheduled",
		Fields: map[string]any{
			"action":        "fresh_process_after_stall",
			"recovery_mode": "fresh_process",
		},
	})
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "focused artifact repair scheduled",
		Fields: map[string]any{
			"action": "focused_artifact_repair",
		},
	})
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "zero-output pre-artifact stall classified unavailable",
		Fields: map[string]any{
			"zero_output_pre_artifact_stall": true,
			"stall_phase":                    "pre_artifact",
		},
	})
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "focused artifact repair exhausted",
		Fields: map[string]any{
			"stall_phase":      "post_artifact",
			"validation_error": "validator verdict is invalid",
		},
	})
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "retry exhausted",
		Fields: map[string]any{
			"stall_phase":      "pre_artifact",
			"validation_error": "runtime_stalled_before_artifacts",
		},
	})

	counters := execution.runtimeRecoveryCounters
	if counters.RepairAttempts != 2 || counters.FreshRetries != 1 || counters.FocusedRepairs != 1 {
		t.Fatalf("unexpected repair counters: %+v", counters)
	}
	if counters.RepairExhausted != 2 {
		t.Fatalf("expected exhausted repair counter, got %+v", counters)
	}
	if counters.StallCount != 2 || counters.PreArtifactStalls != 2 || counters.PostArtifactStalls != 0 {
		t.Fatalf("unexpected stall counters: %+v", counters)
	}
	if counters.ZeroOutputPreArtifactStalls != 1 {
		t.Fatalf("expected zero-output pre-artifact counter, got %+v", counters)
	}

	signals := runtimeRecoveryQualitySignals(counters, 1)
	if !hasRunQualitySignal(signals, "runtime_quality.repair_heavy") ||
		!hasRunQualitySignal(signals, "runtime_quality.stall_pressure") ||
		!hasRunQualitySignal(signals, "runtime_quality.zero_output_pre_artifact_stalls") ||
		!hasRunQualitySignal(signals, "runtime_quality.partial_failures") {
		t.Fatalf("expected runtime quality signals, got %#v", signals)
	}
}

func TestAssessRunArtifactInventoryFlagsSparseCurrentRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ws := workspace.Root{Path: root}
	runID := "run-sparse"
	writeWorkspaceText(t, root, "reports/as-is/overview.md", "# As-Is Overview\n\nProvider wrote this draft artifact for the as-is step.\n")
	writeWorkspaceText(t, root, "reports/agent-outputs/architect/summary.md", "# Architect Summary\n\nProvider wrote this draft artifact for the as-is step.\n")
	writeWorkspaceText(t, root, "proposals/runtime-recommendations.md", "# Runtime Recommendations\n\nDrafted required runtime artifacts for this step.\n")
	writeWorkspaceText(t, root, "reports/changelog/runtime-proposals.md", "# Runtime Changelog\n\nDrafted required runtime artifacts for this step.\n")
	writeWorkspaceText(t, root, "reports/coverage/summary.md", "# Coverage\n\n## Missing\n- owner mappings\n- operational runbooks\n")
	writeWorkspaceText(t, root, "reports/coverage/open-questions.md", "# Open Questions\n\n- Who owns this service?\n")
	writeWorkspaceText(t, root, "reports/findings/findings.md", "# Findings\n\nNo findings reported.\n")
	writeWorkspaceText(t, root, "reports/diagrams/c4-context.mmd", "flowchart LR\n  System[\"Workspace System\"]\n  GapExternal[\"Gap: no evidence-backed external systems\"]\n")
	writeWorkspaceText(t, root, "reports/diagrams/c4-container.mmd", "flowchart LR\n  System[\"Workspace System\"]\n  GapServices[\"Gap: no evidence-backed services\"]\n")
	writeWorkspaceJSON(t, root, filepath.Join("reports", "taskruns", runID, "staging", "final", finalRunIndexFile), contracts.FinalRunIndex{
		Version:           1,
		RunID:             runID,
		Pipeline:          "refresh",
		GeneratedAt:       time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		CitationIndexPath: filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "final", citationIndexFile)),
		CanonicalDocuments: []contracts.FinalRunDocument{
			sparseFinalDocument("doc.overview", "report", "Overview", "reports/as-is/overview.md"),
			sparseFinalDocument("doc.coverage", "coverage", "Coverage", "reports/coverage/summary.md"),
			sparseFinalDocument("doc.questions", "questions", "Questions", "reports/coverage/open-questions.md"),
			sparseFinalDocument("doc.findings", "findings", "Findings", "reports/findings/findings.md"),
		},
		Topics: []contracts.TopicIndexEntry{},
		Semantic: contracts.SemanticSnapshot{
			Coverage: contracts.Coverage{
				Observed: []string{"ftgo"},
				Missing:  []string{"owner mappings", "operational runbooks"},
				Notes:    []string{},
			},
			Questions: []contracts.Question{{ID: "q.owner", Text: "Who owns this service?"}},
			Entities:  []contracts.Entity{},
			Edges:     []contracts.Edge{},
			Findings:  []contracts.Finding{},
		},
	})
	writeWorkspaceText(t, root, filepath.Join("reports", "taskruns", runID, "staging", "shards", "ftgo", "shard-pack-manifest.json"), `{
  "documents": [
    {"path": ".qwen/skills/acp-collect-shard-execution/SKILL.md"}
  ]
}`)

	inventory, signals := assessRunArtifactInventory(
		ws,
		runID,
		RunStatusSucceeded,
		reports.ReportRenderContext{ReportMode: reports.ReportModeNormal},
	)
	if !inventory.FinalIndexPresent {
		t.Fatalf("expected final index to be present: %#v", inventory)
	}
	if inventory.Semantic.Entities != 0 || inventory.Semantic.CoverageMissing != 2 {
		t.Fatalf("unexpected semantic inventory: %#v", inventory.Semantic)
	}
	for _, code := range []string{
		"artifact_quality.empty_semantic_model",
		"artifact_quality.model_entities_missing",
		"artifact_quality.placeholder_artifact",
		"artifact_quality.findings_empty_with_coverage_gap",
		"artifact_quality.c4_gap_only",
		"artifact_quality.hidden_provider_document",
	} {
		if !hasRunQualitySignal(signals, code) {
			t.Fatalf("expected signal %s in %#v", code, signals)
		}
	}
}

func sparseFinalDocument(id string, kind string, title string, canonicalPath string) contracts.FinalRunDocument {
	return contracts.FinalRunDocument{
		ID:            id,
		Kind:          kind,
		Title:         title,
		CanonicalPath: canonicalPath,
		StagedPath:    filepath.ToSlash(filepath.Join("reports", "taskruns", "run-sparse", "staging", "final", canonicalPath)),
		Topics:        []string{},
		CitationIDs:   []string{},
		SourceShards:  []string{"ftgo"},
		Status:        "staged",
	}
}

func loadRefreshArtifactFixtureSet(
	t *testing.T,
	root string,
) ([]contracts.ShardPackManifest, contracts.FinalRunIndex, contracts.CitationIndex, contracts.ValidatorVerdict) {
	t.Helper()

	manifestPaths, err := filepath.Glob(filepath.Join(root, "manifests", "*.json"))
	if err != nil {
		t.Fatalf("glob manifests: %v", err)
	}
	if len(manifestPaths) == 0 {
		t.Fatalf("expected manifests in %s", root)
	}

	manifests := make([]contracts.ShardPackManifest, 0, len(manifestPaths))
	for _, manifestPath := range manifestPaths {
		raw, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			t.Fatalf("read manifest %s: %v", manifestPath, readErr)
		}
		manifest, parseErr := contracts.ParseShardPackManifest(raw)
		if parseErr != nil {
			t.Fatalf("parse manifest %s: %v", manifestPath, parseErr)
		}
		manifests = append(manifests, manifest)
	}

	finalIndexRaw, err := os.ReadFile(filepath.Join(root, "final-run-index.json"))
	if err != nil {
		t.Fatalf("read final-run-index: %v", err)
	}
	finalIndex, err := contracts.ParseFinalRunIndex(finalIndexRaw)
	if err != nil {
		t.Fatalf("parse final-run-index: %v", err)
	}

	citationIndexRaw, err := os.ReadFile(filepath.Join(root, "citation-index.json"))
	if err != nil {
		t.Fatalf("read citation-index: %v", err)
	}
	citationIndex, err := contracts.ParseCitationIndex(citationIndexRaw)
	if err != nil {
		t.Fatalf("parse citation-index: %v", err)
	}

	validatorVerdictRaw, err := os.ReadFile(filepath.Join(root, "validator-verdict.json"))
	if err != nil {
		t.Fatalf("read validator-verdict: %v", err)
	}
	validatorVerdict, err := contracts.ParseValidatorVerdict(validatorVerdictRaw)
	if err != nil {
		t.Fatalf("parse validator-verdict: %v", err)
	}

	return manifests, finalIndex, citationIndex, validatorVerdict
}

func writeWorkspaceText(t *testing.T, root string, rel string, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", rel, err)
	}
}

func writeWorkspaceJSON(t *testing.T, root string, rel string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %q: %v", rel, err)
	}
	writeWorkspaceText(t, root, rel, string(append(raw, '\n')))
}

func hasRunQualitySignal(signals []runQualitySignal, code string) bool {
	for _, signal := range signals {
		if signal.Code == code {
			return true
		}
	}
	return false
}

func containsArtifactWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}
