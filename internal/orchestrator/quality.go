package orchestrator

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/reports"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

var requiredCanonicalLiveDocuments = []string{
	"reports/as-is/overview.md",
	"reports/coverage/summary.md",
	"reports/coverage/open-questions.md",
	"reports/findings/findings.md",
}

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
	Steps                       int `json:"steps"`
	SemanticEntities            int `json:"semantic_entities"`
	SemanticEdges               int `json:"semantic_edges"`
	FindingsCount               int `json:"findings_count"`
	QuestionsCount              int `json:"questions_count"`
	CoverageObserved            int `json:"coverage_observed"`
	CoverageMissing             int `json:"coverage_missing"`
	WarningsCount               int `json:"warnings_count"`
	SignalScore                 int `json:"signal_score"`
	RepairAttempts              int `json:"repair_attempts"`
	RepairExhausted             int `json:"repair_exhausted"`
	FreshRetries                int `json:"fresh_retries"`
	FocusedRepairs              int `json:"focused_repairs"`
	StallCount                  int `json:"stall_count"`
	PreArtifactStalls           int `json:"pre_artifact_stalls"`
	PostArtifactStalls          int `json:"post_artifact_stalls"`
	ZeroOutputPreArtifactStalls int `json:"zero_output_pre_artifact_stalls"`
	PartialFailureCount         int `json:"partial_failure_count"`
}

type runtimeRecoveryCounters struct {
	RepairAttempts              int
	RepairExhausted             int
	FreshRetries                int
	FocusedRepairs              int
	StallCount                  int
	PreArtifactStalls           int
	PostArtifactStalls          int
	ZeroOutputPreArtifactStalls int
}

type runFailureClassification struct {
	Class           string `json:"class"`
	Subclass        string `json:"subclass,omitempty"`
	ParseStage      string `json:"parse_stage,omitempty"`
	Provider        string `json:"provider,omitempty"`
	TaskID          string `json:"task_id,omitempty"`
	StepID          string `json:"step_id,omitempty"`
	ShardID         string `json:"shard_id,omitempty"`
	FailureArtifact string `json:"failure_artifact,omitempty"`
	RawOutput       string `json:"raw_output,omitempty"`
	ShortCause      string `json:"short_cause,omitempty"`
	Source          string `json:"source,omitempty"`
}

type runQualitySignal struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
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
	Failure         runFailureClassification    `json:"failure"`
	RunWarnings     []string                    `json:"run_warnings,omitempty"`
	QualitySignals  []runQualitySignal          `json:"quality_signals,omitempty"`
	EvidenceState   reports.ReportRenderContext `json:"evidence_state"`
	Totals          runQualityTotals            `json:"totals"`
	Steps           []runtimeStepQuality        `json:"steps"`
}

func (e *pipelineExecution) writeRunQualitySummary(status RunStatus, errorCode string, errorMessage string, failure runFailureClassification) (Artifact, error) {
	versions := normalizeRuntimeVersions(e.runtimeVersions)
	renderContext := e.terminalRenderContext(status)
	runWarnings := append([]string(nil), e.warnings...)
	qualitySignals := artifactQualitySignalsFromWarnings(e.warnings)
	qualitySignals = append(qualitySignals, assessLiveReportSurfaceSignals(e.workspace, renderContext, status)...)
	qualitySignals = append(qualitySignals, runtimeRecoveryQualitySignals(e.runtimeRecoveryCounters, len(e.partialFailures))...)
	for _, signal := range qualitySignals {
		runWarnings = append(runWarnings, signal.Message)
	}
	runWarnings = normalizeOrderedUniqueStrings(runWarnings)
	qualitySignals = normalizeRunQualitySignals(qualitySignals)

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
	totals.RepairAttempts = e.runtimeRecoveryCounters.RepairAttempts
	totals.RepairExhausted = e.runtimeRecoveryCounters.RepairExhausted
	totals.FreshRetries = e.runtimeRecoveryCounters.FreshRetries
	totals.FocusedRepairs = e.runtimeRecoveryCounters.FocusedRepairs
	totals.StallCount = e.runtimeRecoveryCounters.StallCount
	totals.PreArtifactStalls = e.runtimeRecoveryCounters.PreArtifactStalls
	totals.PostArtifactStalls = e.runtimeRecoveryCounters.PostArtifactStalls
	totals.ZeroOutputPreArtifactStalls = e.runtimeRecoveryCounters.ZeroOutputPreArtifactStalls
	totals.PartialFailureCount = len(e.partialFailures)

	summary := runQualitySummary{
		Version:         1,
		RunID:           e.runID,
		Pipeline:        string(e.pipeline),
		Status:          status,
		ErrorCode:       strings.TrimSpace(errorCode),
		Error:           strings.TrimSpace(errorMessage),
		GeneratedAt:     e.clock().UTC().Format(time.RFC3339),
		RuntimeVersions: versions,
		Failure:         normalizeRunFailureClassification(failure),
		RunWarnings:     runWarnings,
		QualitySignals:  qualitySignals,
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
		"path":                            path,
		"signal_score":                    totals.SignalScore,
		"quality_alerts":                  len(qualitySignals),
		"repair_attempts":                 totals.RepairAttempts,
		"repair_exhausted":                totals.RepairExhausted,
		"fresh_retries":                   totals.FreshRetries,
		"focused_repairs":                 totals.FocusedRepairs,
		"stall_count":                     totals.StallCount,
		"pre_artifact_stalls":             totals.PreArtifactStalls,
		"post_artifact_stalls":            totals.PostArtifactStalls,
		"zero_output_pre_artifact_stalls": totals.ZeroOutputPreArtifactStalls,
		"partial_failure_count":           totals.PartialFailureCount,
	})
	return Artifact{Path: path, Kind: "taskrun", Label: "Run Quality Summary"}, nil
}

func assessLiveReportSurfaceWarnings(ws workspace.Root, ctx reports.ReportRenderContext, status RunStatus) []string {
	signals := assessLiveReportSurfaceSignals(ws, ctx, status)
	warnings := make([]string, 0, len(signals))
	for _, signal := range signals {
		warnings = append(warnings, signal.Message)
	}
	return normalizeOrderedUniqueStrings(warnings)
}

func runtimeRecoveryQualitySignals(counters runtimeRecoveryCounters, partialFailureCount int) []runQualitySignal {
	signals := []runQualitySignal{}
	if counters.RepairAttempts >= 2 {
		signals = append(signals, runQualitySignal{
			Code:     "runtime_quality.repair_heavy",
			Severity: "warning",
			Message:  "runtime_quality: provider recovery was repair-heavy",
		})
	}
	if counters.RepairExhausted > 0 {
		signals = append(signals, runQualitySignal{
			Code:     "runtime_quality.repair_exhausted",
			Severity: "warning",
			Message:  "runtime_quality: provider recovery was exhausted",
		})
	}
	if counters.StallCount > 0 {
		signals = append(signals, runQualitySignal{
			Code:     "runtime_quality.stall_pressure",
			Severity: "warning",
			Message:  "runtime_quality: provider stall pressure was observed",
		})
	}
	if counters.ZeroOutputPreArtifactStalls > 0 {
		signals = append(signals, runQualitySignal{
			Code:     "runtime_quality.zero_output_pre_artifact_stalls",
			Severity: "warning",
			Message:  "runtime_quality: zero-output pre-artifact stalls were observed",
		})
	}
	if partialFailureCount > 0 {
		signals = append(signals, runQualitySignal{
			Code:     "runtime_quality.partial_failures",
			Severity: "warning",
			Message:  "runtime_quality: partial shard failures were recorded",
		})
	}
	return normalizeRunQualitySignals(signals)
}

func (e *pipelineExecution) recordRuntimeDiagnosticCounters(event acpruntime.DiagnosticEvent) {
	message := strings.ToLower(strings.TrimSpace(event.Message))
	fields := event.Fields
	recoveryMode := diagnosticFieldString(fields, "recovery_mode")
	action := diagnosticFieldString(fields, "action")

	if strings.Contains(message, "repair scheduled") {
		e.runtimeRecoveryCounters.RepairAttempts++
		if strings.Contains(message, "focused artifact repair") ||
			strings.Contains(action, "repair") ||
			strings.Contains(recoveryMode, "repair") {
			e.runtimeRecoveryCounters.FocusedRepairs++
		}
	}
	if message == "retry scheduled" && (recoveryMode == "fresh_process" || action == "fresh_process_after_invalid_artifacts") {
		e.runtimeRecoveryCounters.RepairAttempts++
		e.runtimeRecoveryCounters.FreshRetries++
	}
	if strings.Contains(message, "exhausted") {
		e.runtimeRecoveryCounters.RepairExhausted++
	}
	if diagnosticFieldBool(fields, "zero_output_pre_artifact_stall") {
		e.runtimeRecoveryCounters.ZeroOutputPreArtifactStalls++
	}

	phase := diagnosticFieldString(fields, "stall_phase")
	if phase == "" || !isActualRuntimeStallDiagnostic(message, fields) {
		return
	}
	switch phase {
	case "pre_artifact":
		e.runtimeRecoveryCounters.StallCount++
		e.runtimeRecoveryCounters.PreArtifactStalls++
	case "post_artifact":
		e.runtimeRecoveryCounters.StallCount++
		e.runtimeRecoveryCounters.PostArtifactStalls++
	}
}

func isActualRuntimeStallDiagnostic(message string, fields map[string]any) bool {
	action := diagnosticFieldString(fields, "action")
	recoveryMode := diagnosticFieldString(fields, "recovery_mode")
	validationError := diagnosticFieldString(fields, "validation_error")
	switch message {
	case "retry scheduled":
		return action == "terminate_and_validate"
	case "retry exhausted":
		return true
	case "retry completed":
		return recoveryMode == "fresh_process_artifact_only"
	case "focused artifact repair completed", "collect manifest repair completed":
		return true
	case "focused artifact repair exhausted", "collect manifest repair exhausted":
		return strings.Contains(validationError, "runtime_stalled")
	default:
		return false
	}
}

func diagnosticFieldString(fields map[string]any, key string) string {
	if len(fields) == 0 {
		return ""
	}
	value, ok := fields[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.ToLower(strings.TrimSpace(typed))
	default:
		return strings.ToLower(strings.TrimSpace(jsonScalarString(typed)))
	}
}

func diagnosticFieldBool(fields map[string]any, key string) bool {
	if len(fields) == 0 {
		return false
	}
	value, ok := fields[key]
	if !ok || value == nil {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true") || strings.TrimSpace(typed) == "1"
	default:
		return false
	}
}

func jsonScalarString(value any) string {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return strings.Trim(string(raw), `"`)
	}
}

func assessLiveReportSurfaceSignals(ws workspace.Root, ctx reports.ReportRenderContext, status RunStatus) []runQualitySignal {
	if status != RunStatusSucceeded || ctx.ReportMode != reports.ReportModeNormal {
		return nil
	}

	signals := []runQualitySignal{}
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
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.canonical_live_surface_missing",
				Severity: "warning",
				Message:  "artifact_quality: canonical live surface is missing required document " + rel,
				Path:     rel,
			})
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
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.overview_incomplete_banner",
				Severity: "warning",
				Message:  "artifact_quality: overview report still carries incomplete-analysis banner in a succeeded run",
				Path:     "reports/as-is/overview.md",
			})
		}
		if nonEmpty < 4 || placeholder {
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.overview_placeholder",
				Severity: "warning",
				Message:  "artifact_quality: overview report is too sparse or placeholder-like for a succeeded run",
				Path:     "reports/as-is/overview.md",
			})
		}
	}

	if findingsText, ok := readRel("reports/findings/findings.md"); ok {
		if reportHasIncompleteBanner(findingsText) || findingsHasIncompleteFallback(findingsText) {
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.findings_incomplete",
				Severity: "warning",
				Message:  "artifact_quality: findings report still indicates incomplete analysis in a succeeded run",
				Path:     "reports/findings/findings.md",
			})
		}
	}

	if coverageText, ok := readRel("reports/coverage/summary.md"); ok {
		if reportHasIncompleteBanner(coverageText) || coverageHasIncompleteFallback(coverageText) {
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.coverage_incomplete",
				Severity: "warning",
				Message:  "artifact_quality: coverage summary still indicates incomplete analysis in a succeeded run",
				Path:     "reports/coverage/summary.md",
			})
		}
	}

	if questionsText, ok := readRel("reports/coverage/open-questions.md"); ok {
		if reportHasIncompleteBanner(questionsText) || openQuestionsHasIncompleteFallback(questionsText) {
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.open_questions_incomplete",
				Severity: "warning",
				Message:  "artifact_quality: open questions report still indicates incomplete analysis in a succeeded run",
				Path:     "reports/coverage/open-questions.md",
			})
		}
	}

	return normalizeRunQualitySignals(signals)
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

func artifactQualitySignalsFromWarnings(warnings []string) []runQualitySignal {
	if len(warnings) == 0 {
		return nil
	}
	signals := make([]runQualitySignal, 0, len(warnings))
	for _, warning := range warnings {
		text := strings.TrimSpace(warning)
		if !strings.HasPrefix(text, "artifact_quality:") {
			continue
		}
		signals = append(signals, runQualitySignal{
			Code:     "artifact_quality.runtime_warning",
			Severity: "warning",
			Message:  text,
		})
	}
	return normalizeRunQualitySignals(signals)
}

func normalizeRunQualitySignals(signals []runQualitySignal) []runQualitySignal {
	if len(signals) == 0 {
		return nil
	}
	normalized := make([]runQualitySignal, 0, len(signals))
	seen := map[string]struct{}{}
	for _, signal := range signals {
		signal.Code = strings.TrimSpace(signal.Code)
		signal.Severity = strings.TrimSpace(signal.Severity)
		signal.Message = strings.TrimSpace(signal.Message)
		signal.Path = strings.TrimSpace(signal.Path)
		if signal.Code == "" || signal.Message == "" {
			continue
		}
		if signal.Severity == "" {
			signal.Severity = "warning"
		}
		key := signal.Code + "\x00" + signal.Path + "\x00" + signal.Message
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, signal)
	}
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].Code == normalized[j].Code {
			if normalized[i].Path == normalized[j].Path {
				return normalized[i].Message < normalized[j].Message
			}
			return normalized[i].Path < normalized[j].Path
		}
		return normalized[i].Code < normalized[j].Code
	})
	return normalized
}

func normalizeRunFailureClassification(value runFailureClassification) runFailureClassification {
	value.Class = strings.TrimSpace(value.Class)
	value.Subclass = strings.TrimSpace(value.Subclass)
	value.ParseStage = strings.TrimSpace(value.ParseStage)
	value.Provider = strings.TrimSpace(value.Provider)
	value.TaskID = strings.TrimSpace(value.TaskID)
	value.StepID = strings.TrimSpace(value.StepID)
	value.ShardID = strings.TrimSpace(value.ShardID)
	value.FailureArtifact = strings.TrimSpace(value.FailureArtifact)
	value.RawOutput = strings.TrimSpace(value.RawOutput)
	value.ShortCause = strings.TrimSpace(value.ShortCause)
	value.Source = strings.TrimSpace(value.Source)
	if value.Class == "" {
		value.Class = "none"
	}
	if value.Subclass == "" && value.Class != "none" {
		value.Subclass = "none"
	}
	if value.Source == "" {
		value.Source = "none"
	}
	return value
}

func classifyRunFailureSummary(stepID string, err error) runFailureClassification {
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		return runFailureClassification{
			Class:      strings.TrimSpace(errClassFromMessage(err)),
			StepID:     strings.TrimSpace(stepID),
			ShortCause: strings.TrimSpace(err.Error()),
			Source:     "orchestrator",
		}
	}

	rawOutput := strings.TrimSpace(runnerErr.RawOutputRefs.Metadata)
	if rawOutput == "" {
		rawOutput = strings.TrimSpace(runnerErr.RawOutputRefs.Stdout)
	}
	if rawOutput == "" {
		rawOutput = strings.TrimSpace(runnerErr.RawOutputRefs.Stderr)
	}

	return runFailureClassification{
		Class:      strings.TrimSpace(string(runnerErr.Code)),
		Provider:   strings.TrimSpace(string(runnerErr.Provider)),
		StepID:     strings.TrimSpace(stepID),
		RawOutput:  rawOutput,
		ShortCause: strings.TrimSpace(runnerErr.Error()),
		Source:     "runtime",
	}
}

func errClassFromMessage(err error) string {
	if err == nil {
		return ""
	}
	return "failed"
}
