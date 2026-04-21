package orchestrator

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/reports"
)

type runtimeStepQuality struct {
	StepID           string   `json:"step_id"`
	DomainID         string   `json:"domain_id,omitempty"`
	RuntimeName      string   `json:"runtime_name"`
	RuntimeVersion   string   `json:"runtime_version,omitempty"`
	RepoScopes       []string `json:"repo_scopes,omitempty"`
	SemanticEntities int      `json:"semantic_entities"`
	SemanticEdges    int      `json:"semantic_edges"`
	FindingsCount    int      `json:"findings_count"`
	QuestionsCount   int      `json:"questions_count"`
	CoverageObserved int      `json:"coverage_observed"`
	CoverageMissing  int      `json:"coverage_missing"`
	WarningsCount    int      `json:"warnings_count"`
}

type runQualityTotals struct {
	Steps            int `json:"steps"`
	SemanticEntities int `json:"semantic_entities"`
	SemanticEdges    int `json:"semantic_edges"`
	FindingsCount    int `json:"findings_count"`
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

	steps := append([]runtimeStepQuality(nil), e.runtimeStepMetrics...)
	sort.Slice(steps, func(i, j int) bool {
		if steps[i].StepID == steps[j].StepID {
			return steps[i].DomainID < steps[j].DomainID
		}
		return steps[i].StepID < steps[j].StepID
	})

	totals := runQualityTotals{Steps: len(steps)}
	for _, step := range steps {
		totals.SemanticEntities += step.SemanticEntities
		totals.SemanticEdges += step.SemanticEdges
		totals.FindingsCount += step.FindingsCount
		totals.QuestionsCount += step.QuestionsCount
		totals.CoverageObserved += step.CoverageObserved
		totals.CoverageMissing += step.CoverageMissing
		totals.WarningsCount += step.WarningsCount
	}
	totals.SignalScore = (totals.SemanticEntities * 2) +
		(totals.SemanticEdges * 2) +
		(totals.FindingsCount * 3) +
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
		EvidenceState:   e.terminalRenderContext(status),
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
