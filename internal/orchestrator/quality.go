package orchestrator

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/reports"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type runtimeStepQuality struct {
	StepID           string   `json:"step_id"`
	DomainID         string   `json:"domain_id,omitempty"`
	RuntimeName      string   `json:"runtime_name"`
	RuntimeVersion   string   `json:"runtime_version,omitempty"`
	RepoScopes       []string `json:"repo_scopes,omitempty"`
	ChangesetOps     int      `json:"changeset_ops"`
	EntityUpserts    int      `json:"entity_upserts"`
	EdgeUpserts      int      `json:"edge_upserts"`
	FindingsAdded    int      `json:"findings_added"`
	QuestionsCount   int      `json:"questions_count"`
	CoverageObserved int      `json:"coverage_observed"`
	CoverageMissing  int      `json:"coverage_missing"`
	WarningsCount    int      `json:"warnings_count"`
}

type runQualityTotals struct {
	Steps            int `json:"steps"`
	ChangesetOps     int `json:"changeset_ops"`
	EntityUpserts    int `json:"entity_upserts"`
	EdgeUpserts      int `json:"edge_upserts"`
	FindingsAdded    int `json:"findings_added"`
	QuestionsCount   int `json:"questions_count"`
	CoverageObserved int `json:"coverage_observed"`
	CoverageMissing  int `json:"coverage_missing"`
	WarningsCount    int `json:"warnings_count"`
	SignalScore      int `json:"signal_score"`
}

type runQualitySummary struct {
	Version         int                         `json:"version"`
	RunID           string                      `json:"run_id"`
	Pipeline        string                      `json:"pipeline"`
	Status          RunStatus                   `json:"status"`
	ErrorCode       string                      `json:"error_code,omitempty"`
	Error           string                      `json:"error,omitempty"`
	GeneratedAt     string                      `json:"generated_at"`
	RuntimeVersions []string                    `json:"runtime_versions"`
	RunWarnings     []string                    `json:"run_warnings,omitempty"`
	EvidenceState   reports.ReportRenderContext `json:"evidence_state"`
	Totals          runQualityTotals            `json:"totals"`
	Steps           []runtimeStepQuality        `json:"steps"`
}

func (e *pipelineExecution) writeRunQualitySummary(status RunStatus, errorCode string, errorMessage string) (Artifact, error) {
	versions := normalizeRuntimeVersions(e.runtimeVersions)
	renderContext := e.terminalRenderContext(status)
	runWarnings := append([]string(nil), e.warnings...)
	runWarnings = append(runWarnings, assessLiveReportSurfaceWarnings(e.workspace, renderContext, status)...)
	runWarnings = normalizeOrderedUniqueStrings(runWarnings)

	steps := append([]runtimeStepQuality(nil), e.runtimeStepMetrics...)
	sort.Slice(steps, func(i, j int) bool {
		if steps[i].StepID == steps[j].StepID {
			return steps[i].DomainID < steps[j].DomainID
		}
		return steps[i].StepID < steps[j].StepID
	})

	totals := runQualityTotals{Steps: len(steps)}
	for _, step := range steps {
		totals.ChangesetOps += step.ChangesetOps
		totals.EntityUpserts += step.EntityUpserts
		totals.EdgeUpserts += step.EdgeUpserts
		totals.FindingsAdded += step.FindingsAdded
		totals.QuestionsCount += step.QuestionsCount
		totals.CoverageObserved += step.CoverageObserved
		totals.CoverageMissing += step.CoverageMissing
		totals.WarningsCount += step.WarningsCount
	}
	totals.SignalScore = (totals.EntityUpserts * 2) +
		(totals.EdgeUpserts * 2) +
		(totals.FindingsAdded * 3) +
		totals.QuestionsCount +
		totals.CoverageObserved +
		totals.CoverageMissing

	summary := runQualitySummary{
		Version:         1,
		RunID:           e.runID,
		Pipeline:        string(e.pipeline),
		Status:          status,
		ErrorCode:       strings.TrimSpace(errorCode),
		Error:           strings.TrimSpace(errorMessage),
		GeneratedAt:     e.clock().UTC().Format(time.RFC3339),
		RuntimeVersions: versions,
		RunWarnings:     runWarnings,
		EvidenceState:   renderContext,
		Totals:          totals,
		Steps:           steps,
	}

	content, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return Artifact{}, err
	}
	content = append(content, '\n')
	path := "reports/taskruns/" + e.runID + "-quality.json"
	if err := e.workspace.WriteFile(path, content); err != nil {
		return Artifact{}, err
	}
	e.logInfo(e.stepStatus.CurrentStep, "", "run quality summary persisted", map[string]any{
		"path":         path,
		"signal_score": totals.SignalScore,
	})
	return Artifact{Path: path, Kind: "taskrun", Label: "Run Quality Summary"}, nil
}

func assessLiveReportSurfaceWarnings(ws workspace.Root, ctx reports.ReportRenderContext, status RunStatus) []string {
	if status != RunStatusSucceeded || ctx.ReportMode != reports.ReportModeNormal {
		return nil
	}

	warnings := []string{}
	readRel := func(rel string) (string, bool) {
		abs, err := ws.Resolve(rel)
		if err != nil {
			return "", false
		}
		raw, err := os.ReadFile(abs)
		if err != nil {
			return "", false
		}
		return string(raw), true
	}

	for _, rel := range requiredCanonicalLiveDocuments {
		if _, ok := readRel(rel); !ok {
			warnings = append(warnings, "artifact_quality: canonical live surface is missing required document "+rel)
		}
	}

	if overviewText, ok := readRel("reports/as-is/overview.md"); ok {
		nonEmpty := 0
		placeholder := false
		for _, line := range strings.Split(overviewText, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			nonEmpty++
			lower := strings.ToLower(trimmed)
			if strings.Contains(lower, "no ") && strings.Contains(lower, " yet") {
				placeholder = true
			}
		}
		if reportHasIncompleteBanner(overviewText) {
			warnings = append(warnings, "artifact_quality: overview report still carries incomplete-analysis banner in a succeeded run")
		}
		if nonEmpty < 4 || placeholder {
			warnings = append(
				warnings,
				"artifact_quality: overview report is too sparse or placeholder-like for a succeeded run",
			)
		}
	}

	if findingsText, ok := readRel("reports/findings/findings.md"); ok {
		if reportHasIncompleteBanner(findingsText) || findingsHasIncompleteFallback(findingsText) {
			warnings = append(warnings, "artifact_quality: findings report still indicates incomplete analysis in a succeeded run")
		}
	}

	if coverageText, ok := readRel("reports/coverage/summary.md"); ok {
		if reportHasIncompleteBanner(coverageText) || coverageHasIncompleteFallback(coverageText) {
			warnings = append(warnings, "artifact_quality: coverage summary still indicates incomplete analysis in a succeeded run")
		}
	}

	if questionsText, ok := readRel("reports/coverage/open-questions.md"); ok {
		if reportHasIncompleteBanner(questionsText) || openQuestionsHasIncompleteFallback(questionsText) {
			warnings = append(warnings, "artifact_quality: open questions report still indicates incomplete analysis in a succeeded run")
		}
	}

	return normalizeOrderedUniqueStrings(warnings)
}

func reportHasIncompleteBanner(text string) bool {
	return strings.Contains(text, "Analysis incomplete.") ||
		strings.Contains(text, "Partial analysis. Some shards failed; downstream content may be incomplete.")
}

func findingsHasIncompleteFallback(text string) bool {
	return strings.Contains(text, "Findings unavailable because analysis did not complete.") ||
		strings.Contains(text, "Findings may be incomplete because some shards failed.")
}

func coverageHasIncompleteFallback(text string) bool {
	return strings.Contains(text, "Unavailable due to incomplete analysis.") ||
		strings.Contains(text, "Unknown due to incomplete analysis.") ||
		strings.Contains(text, "May be incomplete because some shards failed.") ||
		strings.Contains(text, "Analysis incomplete. See banner above.")
}

func openQuestionsHasIncompleteFallback(text string) bool {
	return strings.Contains(text, "Open questions unavailable due to incomplete analysis.") ||
		strings.Contains(text, "Open questions may be incomplete because some shards failed.")
}

func normalizeRuntimeVersions(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	bare := map[string]struct{}{}
	versioned := map[string]map[string]struct{}{}

	for value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		name, version, hasSep := strings.Cut(trimmed, "@")
		name = strings.TrimSpace(name)
		version = strings.TrimSpace(version)
		if name == "" {
			continue
		}
		if !hasSep || version == "" {
			bare[name] = struct{}{}
			continue
		}
		if _, exists := versioned[name]; !exists {
			versioned[name] = map[string]struct{}{}
		}
		versioned[name][version] = struct{}{}
	}

	normalized := make([]string, 0, len(values))
	for name, versionSet := range versioned {
		for version := range versionSet {
			normalized = append(normalized, name+"@"+version)
		}
	}
	for name := range bare {
		if _, exists := versioned[name]; exists {
			continue
		}
		normalized = append(normalized, name)
	}
	sort.Strings(normalized)
	return normalized
}
