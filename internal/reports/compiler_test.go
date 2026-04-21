package reports

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
