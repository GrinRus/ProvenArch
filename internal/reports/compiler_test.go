package reports

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func TestCompileAsIsIncompleteAnalysisDoesNotClaimNoServicesFound(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)
	renderCtx := ReportRenderContext{
		Collect: EvidencePhaseState{
			Status:          EvidenceStatusUnusable,
			PlannedShards:   2,
			SucceededShards: 0,
			FailedShards:    2,
		},
		Findings: EvidencePhaseState{
			Status: EvidenceStatusSkipped,
		},
		ReportMode: ReportModeIncomplete,
		Reasons: []string{
			"collect_all_shards_failed",
			"findings_skipped_due_to_unusable_collect",
		},
	}

	if _, err := compiler.CompileAsIs(nil, nil, renderCtx); err != nil {
		t.Fatalf("compile as-is: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "reports/as-is/service-catalog.md"))
	if err != nil {
		t.Fatalf("read service catalog: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "Analysis incomplete.") {
		t.Fatalf("expected incomplete-analysis banner, got:\n%s", text)
	}
	if !strings.Contains(text, "No evidence-backed services were materialized because analysis did not complete.") {
		t.Fatalf("expected incomplete-analysis empty-state, got:\n%s", text)
	}
	if strings.Contains(text, "No services found.") {
		t.Fatalf("did not expect misleading empty-state, got:\n%s", text)
	}
}

func TestCompileAsIsPartialAnalysisMarksPerServiceArtifacts(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)
	renderCtx := ReportRenderContext{
		Collect: EvidencePhaseState{
			Status:          EvidenceStatusPartial,
			PlannedShards:   2,
			SucceededShards: 1,
			FailedShards:    1,
		},
		Findings: EvidencePhaseState{
			Status: EvidenceStatusUsable,
		},
		ReportMode: ReportModeIncomplete,
		Reasons:    []string{"collect_partial_shard_failures"},
	}
	entities := []contracts.Entity{
		{ID: "svc.orders", Name: "Orders", Type: "service"},
	}

	if _, err := compiler.CompileAsIs(entities, nil, renderCtx); err != nil {
		t.Fatalf("compile as-is: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "reports/as-is/services/svc.orders.md"))
	if err != nil {
		t.Fatalf("read per-service as-is report: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "Partial analysis. Some shards failed; downstream content may be incomplete.") {
		t.Fatalf("expected partial-analysis banner, got:\n%s", text)
	}
	if !strings.Contains(text, "- ID: `svc.orders`") {
		t.Fatalf("expected materialized service payload to remain visible, got:\n%s", text)
	}
}

func TestWriteCoverageIncompleteAnalysisUsesExplicitUnknownLanguage(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)
	renderCtx := ReportRenderContext{
		Collect: EvidencePhaseState{
			Status:          EvidenceStatusUnusable,
			PlannedShards:   2,
			SucceededShards: 0,
			FailedShards:    2,
		},
		Findings: EvidencePhaseState{
			Status: EvidenceStatusSkipped,
		},
		ReportMode: ReportModeIncomplete,
	}

	if _, err := compiler.WriteCoverage(nil, nil, renderCtx); err != nil {
		t.Fatalf("write coverage: %v", err)
	}

	summaryContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/summary.md"))
	if err != nil {
		t.Fatalf("read coverage summary: %v", err)
	}
	summaryText := string(summaryContent)
	if !strings.Contains(summaryText, "Unknown due to incomplete analysis.") {
		t.Fatalf("expected explicit missing fallback, got:\n%s", summaryText)
	}
	if !strings.Contains(summaryText, "Unavailable due to incomplete analysis.") {
		t.Fatalf("expected explicit observed fallback, got:\n%s", summaryText)
	}
	questionsContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if err != nil {
		t.Fatalf("read open questions: %v", err)
	}
	questionsText := string(questionsContent)
	if !strings.Contains(questionsText, "Open questions unavailable due to incomplete analysis.") {
		t.Fatalf("expected explicit open-questions fallback, got:\n%s", questionsText)
	}
	if strings.Contains(questionsText, "No open questions.") {
		t.Fatalf("did not expect misleading open-questions empty-state, got:\n%s", questionsText)
	}
}

func TestWriteFindingsIncompleteAnalysisDoesNotClaimNoFindings(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)
	renderCtx := ReportRenderContext{
		Collect: EvidencePhaseState{
			Status:          EvidenceStatusUnusable,
			PlannedShards:   2,
			SucceededShards: 0,
			FailedShards:    2,
		},
		Findings: EvidencePhaseState{
			Status: EvidenceStatusSkipped,
		},
		ReportMode: ReportModeIncomplete,
	}

	if _, err := compiler.WriteFindings(nil, renderCtx); err != nil {
		t.Fatalf("write findings: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "reports/findings/findings.md"))
	if err != nil {
		t.Fatalf("read findings: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "Findings unavailable because analysis did not complete.") {
		t.Fatalf("expected explicit incomplete-analysis findings fallback, got:\n%s", text)
	}
	if strings.Contains(text, "No findings reported.") {
		t.Fatalf("did not expect misleading findings empty-state, got:\n%s", text)
	}
}

func TestCompileProposalsIncompleteAnalysisDoesNotEmitBenignBaseline(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)
	renderCtx := ReportRenderContext{
		Collect: EvidencePhaseState{
			Status:          EvidenceStatusUsable,
			PlannedShards:   2,
			SucceededShards: 2,
			FailedShards:    0,
		},
		Findings: EvidencePhaseState{
			Status:          EvidenceStatusUnusable,
			PlannedShards:   2,
			SucceededShards: 0,
			FailedShards:    2,
		},
		ReportMode: ReportModeIncomplete,
		Reasons:    []string{"findings_all_shards_failed"},
	}

	if _, err := compiler.CompileProposals(nil, renderCtx); err != nil {
		t.Fatalf("compile proposals: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "proposals/proposal-baseline/proposal.md"))
	if err != nil {
		t.Fatalf("read proposal: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "Proposal generation incomplete because no reliable findings set was produced.") {
		t.Fatalf("expected incomplete proposal fallback, got:\n%s", text)
	}
	if strings.Contains(text, "No findings available. Keep monitoring coverage and refresh pipeline.") {
		t.Fatalf("did not expect benign baseline proposal fallback, got:\n%s", text)
	}

	adrContent, err := os.ReadFile(filepath.Join(ws.Path, "proposals/proposal-baseline/ADR.md"))
	if err != nil {
		t.Fatalf("read ADR draft: %v", err)
	}
	if !strings.Contains(string(adrContent), "This draft is triage-only because analysis did not complete.") {
		t.Fatalf("expected triage-only note in ADR draft, got:\n%s", string(adrContent))
	}

	checklistContent, err := os.ReadFile(filepath.Join(ws.Path, "proposals/proposal-baseline/migration-checklist.md"))
	if err != nil {
		t.Fatalf("read migration checklist: %v", err)
	}
	if !strings.Contains(string(checklistContent), "This checklist is triage-only because analysis did not complete.") {
		t.Fatalf("expected triage-only note in migration checklist, got:\n%s", string(checklistContent))
	}
}

func TestWriteArchitectSummaryIncludesIncompleteBanner(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)
	renderCtx := ReportRenderContext{
		Collect: EvidencePhaseState{
			Status:          EvidenceStatusUnusable,
			PlannedShards:   2,
			SucceededShards: 0,
			FailedShards:    2,
		},
		Findings: EvidencePhaseState{
			Status: EvidenceStatusSkipped,
		},
		ReportMode: ReportModeIncomplete,
		Reasons:    []string{"collect_all_shards_failed", "findings_skipped_due_to_unusable_collect"},
	}

	if _, err := compiler.WriteArchitectSummary("# Architect Aggregation Summary\n\n- total findings: 0\n", renderCtx); err != nil {
		t.Fatalf("write architect summary: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "reports/agent-outputs/architect/summary.md"))
	if err != nil {
		t.Fatalf("read architect summary: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "Analysis incomplete.") {
		t.Fatalf("expected incomplete-analysis banner, got:\n%s", text)
	}
	if !strings.Contains(text, "# Architect Aggregation Summary") {
		t.Fatalf("expected architect summary body to remain present, got:\n%s", text)
	}
}
