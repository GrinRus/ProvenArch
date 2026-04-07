package orchestrator

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
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
	Version         int                  `json:"version"`
	RunID           string               `json:"run_id"`
	Pipeline        string               `json:"pipeline"`
	Status          RunStatus            `json:"status"`
	ErrorCode       string               `json:"error_code,omitempty"`
	Error           string               `json:"error,omitempty"`
	GeneratedAt     string               `json:"generated_at"`
	RuntimeVersions []string             `json:"runtime_versions"`
	RunWarnings     []string             `json:"run_warnings,omitempty"`
	Totals          runQualityTotals     `json:"totals"`
	Steps           []runtimeStepQuality `json:"steps"`
}

func (e *pipelineExecution) writeRunQualitySummary(status RunStatus, errorCode string, errorMessage string) (Artifact, error) {
	versions := make([]string, 0, len(e.runtimeVersions))
	for version := range e.runtimeVersions {
		versions = append(versions, version)
	}
	sort.Strings(versions)

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
		RunWarnings:     append([]string(nil), e.warnings...),
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
