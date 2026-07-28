package orchestrator

import (
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/reports"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestAssessLiveReportSurfaceSignalsOverviewPlaceholder(t *testing.T) {
	ws := qualitySignalWorkspace(t, map[string]string{
		"reports/as-is/overview.md": "# As-Is Overview\n\nProvider wrote this draft artifact under the required draft_final_root.\n\n- Services: no services yet\n",
	})

	signals := assessLiveReportSurfaceSignals(ws, reports.DefaultReportRenderContext(), RunStatusSucceeded, usefulQualitySteps(), usefulQualityTotals())

	assertQualitySignal(t, signals, "artifact_quality.overview_placeholder")
}

func TestAssessLiveReportSurfaceSignalsOverviewAllowsSubstantivePlaceholderGuidance(t *testing.T) {
	ws := qualitySignalWorkspace(t, map[string]string{
		"reports/as-is/overview.md": strings.Join([]string{
			"# Architecture Home",
			"",
			"## System at a glance",
			"The payment API serves HTTP requests and persists transactions in PostgreSQL (`payments:services/api/main.go`).",
			"",
			"## Safe-change guidance",
			"Treat the checked-in demo secret as a placeholder that must be replaced before production deployment.",
			"",
			"## Evidence gaps and open questions",
			"Service ownership is not confirmed in the scoped repository evidence.",
			"",
		}, "\n"),
	})

	signals := assessLiveReportSurfaceSignals(ws, reports.DefaultReportRenderContext(), RunStatusSucceeded, usefulQualitySteps(), usefulQualityTotals())
	for _, signal := range signals {
		if signal.Code == "artifact_quality.overview_placeholder" {
			t.Fatalf("substantive safe-change guidance must not be classified as placeholder: %+v", signals)
		}
	}
}

func TestAssessLiveReportSurfaceSignalsOverviewDoesNotJoinNoAndYetAcrossLines(t *testing.T) {
	ws := qualitySignalWorkspace(t, map[string]string{
		"reports/as-is/overview.md": strings.Join([]string{
			"# Architecture Home: Bank of Anthos",
			"",
			"## Integrations and datastores",
			"No explicit message broker is visible in the cited manifests.",
			"",
			"## Evidence gaps and open questions",
			"Runtime behavior and inter-service API contracts are not yet traced from source.",
			"",
		}, "\n"),
	})

	signals := assessLiveReportSurfaceSignals(ws, reports.DefaultReportRenderContext(), RunStatusSucceeded, usefulQualitySteps(), usefulQualityTotals())
	for _, signal := range signals {
		if signal.Code == "artifact_quality.overview_placeholder" {
			t.Fatalf("independent substantive no/not-yet gap lines must not be joined into a placeholder signal: %+v", signals)
		}
	}
}

func TestAssessLiveReportSurfaceSignalsEmptyFindingsWithCriticalGaps(t *testing.T) {
	ws := qualitySignalWorkspace(t, map[string]string{
		"reports/findings/findings.md": "# Findings\n\nNo findings reported.\n",
		"reports/coverage/summary.md": strings.Join([]string{
			"# Coverage Summary",
			"",
			"## Observed",
			"",
			"- service entrypoints",
			"",
			"## Missing",
			"",
			"- owner_team_id mappings for service ownership",
			"- operational runbook for incident response",
			"- external dependency inventory",
			"",
		}, "\n"),
	})

	signals := assessLiveReportSurfaceSignals(ws, reports.DefaultReportRenderContext(), RunStatusSucceeded, usefulQualitySteps(), usefulQualityTotals())

	assertQualitySignal(t, signals, "artifact_quality.empty_findings_with_gaps")
}

func TestAssessLiveReportSurfaceSignalsLowSemanticDensity(t *testing.T) {
	ws := qualitySignalWorkspace(t, nil)
	steps := []runtimeStepQuality{{StepID: "refresh.step1.collect", CoverageObserved: 2}}
	totals := runQualityTotals{Steps: 1, CoverageObserved: 2}

	signals := assessLiveReportSurfaceSignals(ws, reports.DefaultReportRenderContext(), RunStatusSucceeded, steps, totals)

	assertQualitySignal(t, signals, "artifact_quality.low_semantic_density")
}

func TestAssessLiveReportSurfaceSignalsDiagramPlaceholderOnly(t *testing.T) {
	ws := qualitySignalWorkspace(t, map[string]string{
		"reports/diagrams/c4-context.mmd": strings.Join([]string{
			"flowchart LR",
			"  System[\"Workspace System\"]",
			"  GapExternal[\"Gap: no evidence-backed external systems\"]",
			"  System -.-> GapExternal",
			"  GapActors[\"Gap: no evidence-backed actors\"]",
			"  GapActors -.-> System",
			"  GapRelations[\"Gap: no evidence-backed relationships\"]",
			"  System -.-> GapRelations",
			"",
		}, "\n"),
		"reports/diagrams/c4-container.mmd": strings.Join([]string{
			"flowchart LR",
			"  subgraph Workspace[\"Workspace Containers\"]",
			"  end",
			"  GapContainerEdges[\"Gap: no evidence-backed container relations\"]",
			"  Workspace -.-> GapContainerEdges",
			"  GapContainers[\"Gap: no evidence-backed containers\"]",
			"  Workspace -.-> GapContainers",
			"",
		}, "\n"),
	})

	signals := assessLiveReportSurfaceSignals(ws, reports.DefaultReportRenderContext(), RunStatusSucceeded, usefulQualitySteps(), usefulQualityTotals())

	assertQualitySignal(t, signals, "artifact_quality.diagram_placeholder_only")
}

func TestAssessLiveReportSurfaceSignalsProposalsFindingsDisconnected(t *testing.T) {
	ws := qualitySignalWorkspace(t, map[string]string{
		"proposals/runtime-recommendations.md": strings.Join([]string{
			"# Runtime Recommendations",
			"",
			"## Decision / recommended operator action",
			"- No structured finding summary was present, so no source-level architecture change is approved.",
			"",
			"## Evidence used",
			"- `reports/findings/findings.md`",
			"",
			"## Proposed changes or follow-up plan",
			"- Keep the proposal queue unchanged.",
			"",
			"## Risks, gaps, and out-of-scope notes",
			"- Owner mapping remains unresolved.",
			"",
		}, "\n"),
		"reports/changelog/runtime-proposals.md": strings.Join([]string{
			"# Runtime Proposal Changelog",
			"",
			"## Updated architecture/proposal surfaces",
			"- Proposal surface was refreshed.",
			"",
			"## Findings/proposals summary",
			"- No structured finding summary was present.",
			"",
			"## Evidence index or citation references",
			"- `reports/findings/findings.md`",
			"",
			"## Residual coverage gaps",
			"- Owner mapping remains unresolved.",
			"",
		}, "\n"),
	})

	signals := assessLiveReportSurfaceSignals(ws, reports.DefaultReportRenderContext(), RunStatusSucceeded, usefulQualitySteps(), usefulQualityTotals())

	assertQualitySignal(t, signals, "artifact_quality.proposals_findings_disconnected")
}

func TestAssessLiveReportSurfaceSignalsProposalsLowActionability(t *testing.T) {
	ws := qualitySignalWorkspace(t, map[string]string{
		"proposals/runtime-recommendations.md": strings.Join([]string{
			"# Runtime Recommendations",
			"",
			"## Decision / recommended operator action",
			"- Review `finding.owner.missing` during the next architecture review.",
			"",
			"## Evidence used",
			"- `reports/findings/findings.md`",
			"",
			"## Proposed changes or follow-up plan",
			"- Review current-run evidence.",
			"",
			"## Risks, gaps, and out-of-scope notes",
			"- Ownership remains unresolved.",
			"",
		}, "\n"),
		"reports/changelog/runtime-proposals.md": strings.Join([]string{
			"# Runtime Proposal Changelog",
			"",
			"## Updated architecture/proposal surfaces",
			"- Proposal surface was refreshed.",
			"",
			"## Findings/proposals summary",
			"- `finding.owner.missing` remains visible.",
			"",
			"## Evidence index or citation references",
			"- `reports/findings/findings.md`",
			"",
			"## Residual coverage gaps",
			"- Ownership remains unresolved.",
			"",
		}, "\n"),
	})

	signals := assessLiveReportSurfaceSignals(ws, reports.DefaultReportRenderContext(), RunStatusSucceeded, usefulQualitySteps(), usefulQualityTotals())

	assertQualitySignal(t, signals, "artifact_quality.proposals_low_actionability")
}

func TestAssessLiveReportSurfaceSignalsSkipsIncompleteReportMode(t *testing.T) {
	ws := qualitySignalWorkspace(t, map[string]string{
		"reports/as-is/overview.md":            "# As-Is Overview\n\nProvider wrote this draft artifact under the required draft_final_root.\n",
		"proposals/runtime-recommendations.md": "# Runtime Recommendations\n\nNo structured finding summary was present.\n",
	})
	ctx := reports.ReportRenderContext{ReportMode: reports.ReportModeIncomplete}
	steps := []runtimeStepQuality{{StepID: "refresh.step1.collect", CoverageObserved: 2}}
	totals := runQualityTotals{Steps: 1, CoverageObserved: 2}

	signals := assessLiveReportSurfaceSignals(ws, ctx, RunStatusSucceeded, steps, totals)

	if len(signals) != 0 {
		t.Fatalf("expected incomplete report mode to skip artifact-quality signals, got %+v", signals)
	}
}

func qualitySignalWorkspace(t *testing.T, overrides map[string]string) workspace.Root {
	t.Helper()
	ws := workspace.Root{Path: t.TempDir()}
	files := map[string]string{
		"reports/as-is/overview.md": strings.Join([]string{
			"# As-Is Overview",
			"",
			"- Services: payment-api with HTTP entrypoint",
			"- Dependencies (edges): payment-api -> payment database",
			"- External systems: payment gateway webhook",
			"- Datastores: payment database",
			"",
		}, "\n"),
		"reports/findings/findings.md": strings.Join([]string{
			"# Findings",
			"",
			"## Missing owner mapping",
			"",
			"- ID: `finding.owner.missing`",
			"- Severity: `medium`",
			"- Description: payment-api owner mapping is unresolved in repository evidence.",
			"",
		}, "\n"),
		"reports/coverage/summary.md": strings.Join([]string{
			"# Coverage Summary",
			"",
			"## Observed",
			"",
			"- service entrypoints",
			"",
			"## Missing",
			"",
			"- none",
			"",
			"## Notes",
			"",
			"- fixture coverage",
			"",
		}, "\n"),
		"reports/coverage/open-questions.md": "# Open Questions\n\n- `q.owner.payment-api` Who owns payment-api?\n",
		"proposals/runtime-recommendations.md": strings.Join([]string{
			"# Runtime Recommendations",
			"",
			"## Decision / recommended operator action",
			"- Add an owner mapping for `finding.owner.missing` on the payment-api service.",
			"",
			"## Evidence used",
			"- `reports/findings/findings.md`",
			"",
			"## Proposed changes or follow-up plan",
			"- Update services/payment/CODEOWNERS with the owning team for the affected service path.",
			"",
			"## Risks, gaps, and out-of-scope notes",
			"- Runtime SLO ownership remains out of scope.",
			"",
		}, "\n"),
		"reports/changelog/runtime-proposals.md": strings.Join([]string{
			"# Runtime Proposal Changelog",
			"",
			"## Updated architecture/proposal surfaces",
			"- `proposals/runtime-recommendations.md` records the owner mapping action for `finding.owner.missing`.",
			"",
			"## Findings/proposals summary",
			"- `finding.owner.missing` is mapped to a CODEOWNERS follow-up.",
			"",
			"## Evidence index or citation references",
			"- `reports/findings/findings.md`",
			"",
			"## Residual coverage gaps",
			"- Runtime SLO ownership remains out of scope.",
			"",
		}, "\n"),
	}
	for rel, content := range overrides {
		files[rel] = content
	}
	for rel, content := range files {
		if err := ws.WriteFile(rel, []byte(content)); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return ws
}

func usefulQualitySteps() []runtimeStepQuality {
	return []runtimeStepQuality{{
		StepID:           "refresh.step1.collect",
		SemanticEntities: 1,
		SemanticEdges:    1,
		FindingsCount:    1,
		CoverageObserved: 1,
	}}
}

func usefulQualityTotals() runQualityTotals {
	return runQualityTotals{
		Steps:            1,
		SemanticEntities: 1,
		SemanticEdges:    1,
		FindingsCount:    1,
		CoverageObserved: 1,
	}
}

func assertQualitySignal(t *testing.T, signals []runQualitySignal, code string) {
	t.Helper()
	for _, signal := range signals {
		if signal.Code == code {
			return
		}
	}
	t.Fatalf("expected quality signal %q, got %+v", code, signals)
}
