package orchestrator

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
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
	Steps                        int `json:"steps"`
	SemanticEntities             int `json:"semantic_entities"`
	SemanticEdges                int `json:"semantic_edges"`
	FindingsCount                int `json:"findings_count"`
	QuestionsCount               int `json:"questions_count"`
	CoverageObserved             int `json:"coverage_observed"`
	CoverageMissing              int `json:"coverage_missing"`
	WarningsCount                int `json:"warnings_count"`
	SignalScore                  int `json:"signal_score"`
	RepairAttempts               int `json:"repair_attempts"`
	RepairExhausted              int `json:"repair_exhausted"`
	FreshRetries                 int `json:"fresh_retries"`
	FocusedRepairs               int `json:"focused_repairs"`
	StallCount                   int `json:"stall_count"`
	PreArtifactStalls            int `json:"pre_artifact_stalls"`
	PostArtifactStalls           int `json:"post_artifact_stalls"`
	ValidArtifactControlledStops int `json:"valid_artifact_controlled_stops"`
	ZeroOutputPreArtifactStalls  int `json:"zero_output_pre_artifact_stalls"`
	PartialFailureCount          int `json:"partial_failure_count"`
}

type runtimeRecoveryCounters struct {
	RepairAttempts               int
	RepairExhausted              int
	FreshRetries                 int
	FocusedRepairs               int
	StallCount                   int
	PreArtifactStalls            int
	PostArtifactStalls           int
	ValidArtifactControlledStops int
	ZeroOutputPreArtifactStalls  int
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
	Version           int                         `json:"version"`
	RunID             string                      `json:"run_id"`
	Pipeline          string                      `json:"pipeline"`
	Status            RunStatus                   `json:"status"`
	ErrorCode         string                      `json:"error_code,omitempty"`
	Error             string                      `json:"error,omitempty"`
	GeneratedAt       string                      `json:"generated_at"`
	RuntimeVersions   []string                    `json:"runtime_versions"`
	Failure           runFailureClassification    `json:"failure"`
	RunWarnings       []string                    `json:"run_warnings,omitempty"`
	QualitySignals    []runQualitySignal          `json:"quality_signals,omitempty"`
	EvidenceState     reports.ReportRenderContext `json:"evidence_state"`
	ArtifactInventory runArtifactInventory        `json:"artifact_inventory"`
	Totals            runQualityTotals            `json:"totals"`
	Steps             []runtimeStepQuality        `json:"steps"`
}

type runArtifactInventory struct {
	RunID             string                        `json:"run_id"`
	FinalIndexPath    string                        `json:"final_index_path,omitempty"`
	FinalIndexPresent bool                          `json:"final_index_present"`
	Semantic          runArtifactSemanticInventory  `json:"semantic"`
	Surfaces          []runArtifactSurfaceInventory `json:"surfaces"`
}

type runArtifactSemanticInventory struct {
	CanonicalDocuments int `json:"canonical_documents"`
	Entities           int `json:"entities"`
	Edges              int `json:"edges"`
	Findings           int `json:"findings"`
	Questions          int `json:"questions"`
	CoverageObserved   int `json:"coverage_observed"`
	CoverageMissing    int `json:"coverage_missing"`
}

type runArtifactSurfaceInventory struct {
	Name     string   `json:"name"`
	Expected bool     `json:"expected"`
	Count    int      `json:"count"`
	Status   string   `json:"status"`
	Paths    []string `json:"paths,omitempty"`
}

type artifactSurfaceDefinition struct {
	name     string
	expected bool
	prefixes []string
}

func (e *pipelineExecution) writeRunQualitySummary(status RunStatus, errorCode string, errorMessage string, failure runFailureClassification) (Artifact, error) {
	versions := normalizeRuntimeVersions(e.runtimeVersions)
	renderContext := e.terminalRenderContext(status)
	artifactInventory, artifactInventorySignals := assessRunArtifactInventory(e.workspace, e.runID, status, renderContext)
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
	totals.ValidArtifactControlledStops = e.runtimeRecoveryCounters.ValidArtifactControlledStops
	totals.ZeroOutputPreArtifactStalls = e.runtimeRecoveryCounters.ZeroOutputPreArtifactStalls
	totals.PartialFailureCount = len(e.partialFailures)

	runWarnings := append([]string(nil), e.warnings...)
	qualitySignals := artifactQualitySignalsFromWarnings(e.warnings)
	qualitySignals = append(qualitySignals, assessLiveReportSurfaceSignals(e.workspace, renderContext, status, steps, totals)...)
	qualitySignals = append(qualitySignals, artifactInventorySignals...)
	qualitySignals = append(qualitySignals, runtimeRecoveryQualitySignals(e.runtimeRecoveryCounters, len(e.partialFailures))...)
	for _, signal := range qualitySignals {
		runWarnings = append(runWarnings, signal.Message)
	}
	runWarnings = normalizeOrderedUniqueStrings(runWarnings)
	qualitySignals = normalizeRunQualitySignals(qualitySignals)

	summary := runQualitySummary{
		Version:           1,
		RunID:             e.runID,
		Pipeline:          string(e.pipeline),
		Status:            status,
		ErrorCode:         strings.TrimSpace(errorCode),
		Error:             strings.TrimSpace(errorMessage),
		GeneratedAt:       e.clock().UTC().Format(time.RFC3339),
		RuntimeVersions:   versions,
		Failure:           normalizeRunFailureClassification(failure),
		RunWarnings:       runWarnings,
		QualitySignals:    qualitySignals,
		EvidenceState:     renderContext,
		ArtifactInventory: artifactInventory,
		Totals:            totals,
		Steps:             steps,
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
		"valid_artifact_controlled_stops": totals.ValidArtifactControlledStops,
		"zero_output_pre_artifact_stalls": totals.ZeroOutputPreArtifactStalls,
		"partial_failure_count":           totals.PartialFailureCount,
	})
	return Artifact{Path: path, Kind: "taskrun", Label: "Run Quality Summary"}, nil
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
	if isValidArtifactControlledStopDiagnostic(message, fields) {
		e.runtimeRecoveryCounters.ValidArtifactControlledStops++
		return
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
	if isValidArtifactControlledStopDiagnostic(message, fields) {
		return false
	}
	action := diagnosticFieldString(fields, "action")
	recoveryMode := diagnosticFieldString(fields, "recovery_mode")
	validationError := diagnosticFieldString(fields, "validation_error")
	switch message {
	case "retry scheduled":
		return action == "terminate_and_validate"
	case "retry exhausted":
		return true
	case "retry completed":
		return recoveryMode == "fresh_process_artifact_only" && !diagnosticArtifactValid(fields)
	case "focused artifact repair exhausted", "collect manifest repair exhausted":
		return strings.Contains(validationError, "runtime_stalled")
	default:
		return false
	}
}

func isValidArtifactControlledStopDiagnostic(message string, fields map[string]any) bool {
	if message != "provider command finished" {
		return false
	}
	if diagnosticFieldString(fields, "exit_reason") != "stall" {
		return false
	}
	if diagnosticFieldString(fields, "validation_error") != "" {
		return false
	}
	return diagnosticArtifactValid(fields)
}

func diagnosticArtifactValid(fields map[string]any) bool {
	if diagnosticFieldBool(fields, "artifact_valid") {
		return true
	}
	switch diagnosticFieldString(fields, "artifact_state") {
	case "valid", "succeeded", "success":
		return true
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

func assessLiveReportSurfaceSignals(ws workspace.Root, ctx reports.ReportRenderContext, status RunStatus, steps []runtimeStepQuality, totals runQualityTotals) []runQualitySignal {
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
		for _, line := range strings.Split(overviewText, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			nonEmpty++
		}
		if reportHasIncompleteBanner(overviewText) {
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.overview_incomplete_banner",
				Severity: "warning",
				Message:  "artifact_quality: overview report still carries incomplete-analysis banner in a succeeded run",
				Path:     "reports/as-is/overview.md",
			})
		}
		if nonEmpty < 4 || overviewLooksPlaceholder(overviewText) {
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.overview_placeholder",
				Severity: "warning",
				Message:  "artifact_quality: overview report is too sparse or placeholder-like for a succeeded run",
				Path:     "reports/as-is/overview.md",
			})
		}
	}

	criticalGapCategories := []string{}
	if coverageText, ok := readRel("reports/coverage/summary.md"); ok {
		criticalGapCategories = criticalCoverageGapCategories(coverageText)
		if reportHasIncompleteBanner(coverageText) || coverageHasIncompleteFallback(coverageText) {
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.coverage_incomplete",
				Severity: "warning",
				Message:  "artifact_quality: coverage summary still indicates incomplete analysis in a succeeded run",
				Path:     "reports/coverage/summary.md",
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
		if len(criticalGapCategories) > 0 && findingsReportsNoFindings(findingsText) {
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.empty_findings_with_gaps",
				Severity: "warning",
				Message:  "artifact_quality: findings are empty while critical coverage gaps remain",
				Path:     "reports/findings/findings.md",
			})
		}
		signals = append(signals, proposalsFindingsAlignmentSignals(readRel, findingsText)...)
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

	if collectStepsObserved(steps) &&
		totals.CoverageObserved > 0 &&
		totals.SemanticEntities == 0 &&
		totals.SemanticEdges == 0 &&
		totals.FindingsCount == 0 {
		signals = append(signals, runQualitySignal{
			Code:     "artifact_quality.low_semantic_density",
			Severity: "warning",
			Message:  "artifact_quality: successful run observed coverage but produced no semantic entities, edges, or findings",
		})
	}

	if diagramsArePlaceholderOnly(ws, readRel) {
		signals = append(signals, runQualitySignal{
			Code:     "artifact_quality.diagram_placeholder_only",
			Severity: "warning",
			Message:  "artifact_quality: diagrams contain only gap nodes without component or code views",
			Path:     "reports/diagrams",
		})
	}

	return normalizeRunQualitySignals(signals)
}

func overviewLooksPlaceholder(text string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "no ") && strings.Contains(lower, " yet") {
		return true
	}
	placeholderMarkers := []string{
		"provider wrote this draft artifact under the required draft_final_root",
		"draft artifact under the required draft_final_root",
		"placeholder",
		"todo",
	}
	for _, marker := range placeholderMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func criticalCoverageGapCategories(text string) []string {
	categories := map[string]struct{}{}
	scanCoverageMissingBullets(text, func(value string) {
		if category := canonicalCoverageGapCategory(value); category != "" {
			categories[category] = struct{}{}
		}
	})
	if len(categories) == 0 {
		scanCoverageAllBullets(text, func(value string) {
			if category := canonicalCoverageGapCategory(value); category != "" {
				categories[category] = struct{}{}
			}
		})
	}
	ordered := make([]string, 0, len(categories))
	for category := range categories {
		ordered = append(ordered, category)
	}
	sort.Strings(ordered)
	return ordered
}

func scanCoverageMissingBullets(text string, visit func(string)) {
	inMissing := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(lower, "## "))
			inMissing = heading == "missing" || heading == "missing coverage" || heading == "coverage missing"
			continue
		}
		if !inMissing || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		visit(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
	}
}

func scanCoverageAllBullets(text string, visit func(string)) {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			visit(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
		}
	}
}

func canonicalCoverageGapCategory(value string) string {
	normalized := normalizeCoverageGapText(value)
	switch {
	case strings.Contains(normalized, "owner team id") ||
		(strings.Contains(normalized, "owner") && (strings.Contains(normalized, "mapping") || strings.Contains(normalized, "team") || strings.Contains(normalized, "ownership"))):
		return "owner_mapping"
	case strings.Contains(normalized, "runbook") ||
		(strings.Contains(normalized, "operational") && (strings.Contains(normalized, "handoff") || strings.Contains(normalized, "procedure") || strings.Contains(normalized, "playbook"))):
		return "operational_runbook"
	case strings.Contains(normalized, "third party") ||
		(strings.Contains(normalized, "external") && (strings.Contains(normalized, "dependency") || strings.Contains(normalized, "system") || strings.Contains(normalized, "integration"))):
		return "external_dependency"
	case strings.Contains(normalized, "datastore") ||
		strings.Contains(normalized, "database") ||
		strings.Contains(normalized, "storage") ||
		strings.Contains(normalized, "persistence"):
		return "datastore_storage"
	case strings.Contains(normalized, "ci cd") ||
		strings.Contains(normalized, "cicd") ||
		strings.Contains(normalized, "continuous integration") ||
		strings.Contains(normalized, "workflow") ||
		strings.Contains(normalized, "pipeline"):
		return "cicd"
	case strings.Contains(normalized, "api") ||
		strings.Contains(normalized, "interface") ||
		strings.Contains(normalized, "endpoint"):
		return "api_interface"
	default:
		return ""
	}
}

func normalizeCoverageGapText(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		"`", " ",
		"_", " ",
		"-", " ",
		":", " ",
		".", " ",
		",", " ",
		"/", " ",
		"(", " ",
		")", " ",
	)
	return strings.Join(strings.Fields(replacer.Replace(lower)), " ")
}

func findingsReportsNoFindings(text string) bool {
	return strings.Contains(strings.ToLower(text), "no findings reported.")
}

type markdownFindingsSummary struct {
	IDs            []string
	HighOrMediumID bool
}

func proposalsFindingsAlignmentSignals(readRel func(string) (string, bool), findingsText string) []runQualitySignal {
	findings := summarizeMarkdownFindings(findingsText)
	if len(findings.IDs) == 0 || findingsReportsNoFindings(findingsText) {
		return nil
	}
	proposalText, proposalOK := readRel("proposals/runtime-recommendations.md")
	changelogText, changelogOK := readRel("reports/changelog/runtime-proposals.md")
	if !proposalOK && !changelogOK {
		return nil
	}
	combined := strings.Join([]string{proposalText, changelogText}, "\n")
	signals := []runQualitySignal{}
	if proposalTextDeniesStructuredFindings(combined) || !artifactTextReferencesAnyFindingID(combined, findings.IDs) {
		signals = append(signals, runQualitySignal{
			Code:     "artifact_quality.proposals_findings_disconnected",
			Severity: "warning",
			Message:  "artifact_quality: proposals/changelog are disconnected from non-empty findings",
			Path:     "proposals/runtime-recommendations.md",
		})
	}
	if findings.HighOrMediumID && !proposalTextHasFindingActionability(combined, findings.IDs) {
		signals = append(signals, runQualitySignal{
			Code:     "artifact_quality.proposals_low_actionability",
			Severity: "warning",
			Message:  "artifact_quality: medium/high findings exist but proposals lack finding-linked operator action and affected surface",
			Path:     "proposals/runtime-recommendations.md",
		})
	}
	return signals
}

func summarizeMarkdownFindings(text string) markdownFindingsSummary {
	summary := markdownFindingsSummary{}
	currentID := ""
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, "- id:"):
			currentID = firstMarkdownFieldValue(trimmed[len("- id:"):])
			if currentID != "" {
				summary.IDs = append(summary.IDs, currentID)
			}
		case strings.HasPrefix(lower, "- severity:"):
			severity := strings.ToLower(firstMarkdownFieldValue(trimmed[len("- severity:"):]))
			if currentID != "" && (severity == "high" || severity == "medium") {
				summary.HighOrMediumID = true
			}
		}
	}
	summary.IDs = normalizeOrderedUniqueStrings(summary.IDs)
	return summary
}

func firstMarkdownFieldValue(value string) string {
	value = strings.TrimSpace(value)
	if start := strings.Index(value, "`"); start >= 0 {
		if end := strings.Index(value[start+1:], "`"); end >= 0 {
			return strings.TrimSpace(value[start+1 : start+1+end])
		}
	}
	value = strings.Trim(value, "` \t:;,")
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], "` \t:;,.")
}

func proposalTextDeniesStructuredFindings(text string) bool {
	lower := strings.ToLower(text)
	markers := []string{
		"no structured finding summary was present",
		"no structured findings were present",
		"no structured finding summary",
		"no source-level architecture change is approved",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func artifactTextReferencesAnyFindingID(text string, findingIDs []string) bool {
	lower := strings.ToLower(text)
	for _, id := range findingIDs {
		id = strings.ToLower(strings.TrimSpace(id))
		if id != "" && strings.Contains(lower, id) {
			return true
		}
	}
	return false
}

func proposalTextHasFindingActionability(text string, findingIDs []string) bool {
	lower := strings.ToLower(text)
	if !artifactTextReferencesAnyFindingID(text, findingIDs) {
		return false
	}
	actionMarkers := []string{
		"recommended operator action",
		"recommended action",
		"operator action",
		"follow-up",
		"follow up",
		"remediate",
		"replace",
		"add ",
		"update ",
		"document ",
		"assign ",
	}
	surfaceMarkers := []string{
		"affected surface",
		"affected path",
		"service",
		"component",
		"datastore",
		"runbook",
		"src/",
		"services/",
		"internal/",
		"cmd/",
		".go",
		".yaml",
		".yml",
		".ts",
		".tsx",
	}
	return textContainsAny(lower, actionMarkers) && textContainsAny(lower, surfaceMarkers)
}

func textContainsAny(text string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func collectStepsObserved(steps []runtimeStepQuality) bool {
	for _, step := range steps {
		stepID := strings.ToLower(strings.TrimSpace(step.StepID))
		if strings.Contains(stepID, ".step1.collect") {
			return true
		}
	}
	return false
}

func diagramsArePlaceholderOnly(ws workspace.Root, readRel func(string) (string, bool)) bool {
	contextText, contextOK := readRel("reports/diagrams/c4-context.mmd")
	containerText, containerOK := readRel("reports/diagrams/c4-container.mmd")
	if !contextOK || !containerOK {
		return false
	}
	if hasDiagramFiles(ws, "reports/diagrams/components") || hasDiagramFiles(ws, "reports/diagrams/code") {
		return false
	}
	return mermaidDiagramIsGapOnly(contextText) && mermaidDiagramIsGapOnly(containerText)
}

func hasDiagramFiles(ws workspace.Root, rel string) bool {
	abs, err := ws.Resolve(rel)
	if err != nil {
		return false
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".mmd") {
			return true
		}
	}
	return false
}

func mermaidDiagramIsGapOnly(text string) bool {
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "gap:") {
		return false
	}
	evidenceMarkers := []string{
		"evidence-backed services:",
		"service:",
		"datastore:",
		"external:",
		"actor:",
	}
	for _, marker := range evidenceMarkers {
		if strings.Contains(lower, marker) {
			return false
		}
	}
	return true
}

func assessRunArtifactInventory(
	ws workspace.Root,
	runID string,
	status RunStatus,
	ctx reports.ReportRenderContext,
) (runArtifactInventory, []runQualitySignal) {
	runID = strings.TrimSpace(runID)
	inventory := runArtifactInventory{
		RunID:          runID,
		FinalIndexPath: filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "final", finalRunIndexFile)),
	}
	finalIndex, finalIndexOK := loadRunFinalIndex(ws, runID)
	inventory.FinalIndexPresent = finalIndexOK
	if finalIndexOK {
		inventory.Semantic = runArtifactSemanticInventory{
			CanonicalDocuments: len(finalIndex.CanonicalDocuments),
			Entities:           len(finalIndex.Semantic.Entities),
			Edges:              len(finalIndex.Semantic.Edges),
			Findings:           len(finalIndex.Semantic.Findings),
			Questions:          len(finalIndex.Semantic.Questions),
			CoverageObserved:   len(finalIndex.Semantic.Coverage.Observed),
			CoverageMissing:    len(finalIndex.Semantic.Coverage.Missing),
		}
	}
	inventory.Surfaces = collectRunArtifactSurfaceInventory(ws, runID)

	if status != RunStatusSucceeded || ctx.ReportMode != reports.ReportModeNormal {
		return inventory, nil
	}

	signals := []runQualitySignal{}
	nontrivial := isNontrivialArtifactInventory(inventory)
	if !finalIndexOK {
		signals = append(signals, runQualitySignal{
			Code:     "artifact_quality.final_index_missing",
			Severity: "warning",
			Message:  "artifact_quality: current run artifact inventory is missing final-run-index.json",
			Path:     inventory.FinalIndexPath,
		})
	}
	if nontrivial && finalIndexOK && inventory.Semantic.Entities == 0 {
		signals = append(signals, runQualitySignal{
			Code:     "artifact_quality.empty_semantic_model",
			Severity: "warning",
			Message:  "artifact_quality: current run produced reviewable documents but final semantic model has zero entities",
			Path:     inventory.FinalIndexPath,
		})
	}
	if nontrivial && finalIndexOK && inventory.Semantic.Entities > 1 && inventory.Semantic.Edges == 0 {
		signals = append(signals, runQualitySignal{
			Code:     "artifact_quality.empty_semantic_edges",
			Severity: "warning",
			Message:  "artifact_quality: current run produced multiple semantic entities but no semantic edges",
			Path:     inventory.FinalIndexPath,
		})
	}
	if nontrivial && finalIndexOK {
		signals = append(signals, semanticScaffoldSignals(finalIndex, inventory)...)
	}
	if nontrivial && surfaceCount(inventory, "model_entities") == 0 {
		signals = append(signals, runQualitySignal{
			Code:     "artifact_quality.model_entities_missing",
			Severity: "warning",
			Message:  "artifact_quality: current run has no promoted model/entities artifacts",
			Path:     "model/entities",
		})
	}
	signals = append(signals, placeholderArtifactSignals(ws)...)
	signals = append(signals, findingsCoverageGapSignals(ws, finalIndex, finalIndexOK)...)
	signals = append(signals, gapOnlyDiagramSignals(ws, inventory, nontrivial)...)
	signals = append(signals, scaffoldDiagramSignals(ws, inventory, nontrivial)...)
	signals = append(signals, hiddenProviderDocumentSignals(ws, runID)...)
	return inventory, normalizeRunQualitySignals(signals)
}

func collectRunArtifactSurfaceInventory(ws workspace.Root, runID string) []runArtifactSurfaceInventory {
	definitions := []artifactSurfaceDefinition{
		{name: "charter", expected: true, prefixes: []string{"charter/"}},
		{name: "skills", expected: true, prefixes: []string{"skills/"}},
		{name: "as_is", expected: true, prefixes: []string{"reports/as-is/"}},
		{name: "coverage", expected: true, prefixes: []string{"reports/coverage/"}},
		{name: "findings", expected: true, prefixes: []string{"reports/findings/"}},
		{name: "agent_outputs", expected: true, prefixes: []string{"reports/agent-outputs/"}},
		{name: "model_entities", expected: true, prefixes: []string{"model/entities/"}},
		{name: "model_edges", expected: false, prefixes: []string{"model/edges/"}},
		{name: "diagrams", expected: true, prefixes: []string{"reports/diagrams/"}},
		{name: "proposals", expected: true, prefixes: []string{"proposals/"}},
		{name: "changelog", expected: true, prefixes: []string{"reports/changelog/"}},
		{name: "taskrun", expected: true, prefixes: []string{filepath.ToSlash(filepath.Join("reports", "taskruns", strings.TrimSpace(runID))) + "/"}},
	}
	paths := listWorkspaceFiles(ws)
	surfaces := make([]runArtifactSurfaceInventory, 0, len(definitions))
	for _, definition := range definitions {
		matches := []string{}
		for _, rel := range paths {
			for _, prefix := range definition.prefixes {
				if strings.HasPrefix(rel, prefix) {
					matches = append(matches, rel)
					break
				}
			}
		}
		status := "present"
		if len(matches) == 0 {
			status = "missing"
		}
		surfaces = append(surfaces, runArtifactSurfaceInventory{
			Name:     definition.name,
			Expected: definition.expected,
			Count:    len(matches),
			Status:   status,
			Paths:    sampleArtifactPaths(matches, 16),
		})
	}
	return surfaces
}

func listWorkspaceFiles(ws workspace.Root) []string {
	root := strings.TrimSpace(ws.Path)
	if root == "" {
		return nil
	}
	paths := []string{}
	_ = filepath.WalkDir(root, func(absPath string, entry fs.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, absPath)
		if relErr != nil {
			return nil
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(paths)
	return paths
}

func sampleArtifactPaths(paths []string, limit int) []string {
	if len(paths) == 0 || limit <= 0 {
		return nil
	}
	sampled := append([]string(nil), paths...)
	sort.Strings(sampled)
	if len(sampled) > limit {
		sampled = sampled[:limit]
	}
	return sampled
}

func surfaceCount(inventory runArtifactInventory, name string) int {
	for _, surface := range inventory.Surfaces {
		if surface.Name == name {
			return surface.Count
		}
	}
	return 0
}

func isNontrivialArtifactInventory(inventory runArtifactInventory) bool {
	if inventory.Semantic.CanonicalDocuments >= 4 {
		return true
	}
	if inventory.Semantic.CoverageObserved > 0 || inventory.Semantic.CoverageMissing > 0 {
		return true
	}
	if surfaceCount(inventory, "as_is") >= 2 {
		return true
	}
	return false
}

func loadRunFinalIndex(ws workspace.Root, runID string) (contracts.FinalRunIndex, bool) {
	if strings.TrimSpace(runID) == "" {
		return contracts.FinalRunIndex{}, false
	}
	rel := filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "final", finalRunIndexFile))
	abs, err := ws.Resolve(rel)
	if err != nil {
		return contracts.FinalRunIndex{}, false
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return contracts.FinalRunIndex{}, false
	}
	index, err := contracts.ParseFinalRunIndex(raw)
	if err != nil {
		return contracts.FinalRunIndex{}, false
	}
	return index, true
}

func placeholderArtifactSignals(ws workspace.Root) []runQualitySignal {
	paths := []string{
		"charter/overview.md",
		"reports/as-is/overview.md",
		"reports/agent-outputs/architect/summary.md",
		"proposals/runtime-recommendations.md",
		"reports/changelog/runtime-proposals.md",
	}
	signals := []runQualitySignal{}
	for _, rel := range paths {
		text, ok := readWorkspaceText(ws, rel)
		if !ok || !artifactTextPlaceholderLike(text) {
			continue
		}
		signals = append(signals, runQualitySignal{
			Code:     "artifact_quality.placeholder_artifact",
			Severity: "warning",
			Message:  "artifact_quality: promoted analysis artifact is placeholder-like: " + rel,
			Path:     rel,
		})
	}
	return signals
}

func artifactTextPlaceholderLike(text string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "provider wrote this draft artifact") ||
		strings.Contains(lower, "drafted required runtime artifacts") ||
		strings.Contains(lower, "no findings reported.") ||
		strings.Contains(lower, "draft surface initialized for the scoped repository analysis") ||
		strings.Contains(lower, "final content must stay tied to collected shard evidence and validator output") ||
		strings.Contains(lower, "runtime proposal surface initialized") ||
		strings.Contains(lower, "runtime draft recovery initialized") ||
		strings.Contains(lower, "draft recovery initialized") ||
		strings.Contains(lower, "treat this as diagnostic evidence until") {
		return true
	}
	if !artifactTextHasEvidenceMarkers(lower) &&
		(strings.Contains(lower, "changes must remain traceable to collected evidence") ||
			strings.Contains(lower, "promote only after artifact validation succeeds") ||
			strings.Contains(lower, "current run evidence should be reviewed before promotion") ||
			strings.Contains(lower, "owner mappings and unresolved coverage gaps remain the first follow-up surfaces") ||
			strings.Contains(lower, "promote only recommendations that cite collected shard manifests")) {
		return true
	}
	nonEmpty := 0
	noYet := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		nonEmpty++
		lineLower := strings.ToLower(trimmed)
		if strings.Contains(lineLower, "no ") && strings.Contains(lineLower, " yet") {
			noYet = true
		}
	}
	return nonEmpty <= 4 && noYet
}

func artifactTextHasEvidenceMarkers(lower string) bool {
	markers := []string{
		"finding.",
		"question.",
		"reports/findings/",
		"reports/coverage/",
		"validator-verdict.json",
		"evidence traceability",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func findingsCoverageGapSignals(ws workspace.Root, finalIndex contracts.FinalRunIndex, finalIndexOK bool) []runQualitySignal {
	categories := map[string]struct{}{}
	if finalIndexOK {
		for _, value := range finalIndex.Semantic.Coverage.Missing {
			if category := canonicalCoverageGapCategory(value); category != "" {
				categories[category] = struct{}{}
			}
		}
	}
	if coverageText, ok := readWorkspaceText(ws, "reports/coverage/summary.md"); ok {
		for _, category := range criticalCoverageGapCategories(coverageText) {
			categories[category] = struct{}{}
		}
	}
	if len(categories) == 0 {
		return nil
	}
	findingsText, ok := readWorkspaceText(ws, "reports/findings/findings.md")
	if !ok {
		return []runQualitySignal{{
			Code:     "artifact_quality.findings_missing_for_coverage_gap",
			Severity: "warning",
			Message:  "artifact_quality: critical coverage gaps are present but findings report is missing",
			Path:     "reports/findings/findings.md",
		}}
	}
	if finalIndexOK && len(finalIndex.Semantic.Findings) > 0 && !findingsReportsNoFindings(findingsText) {
		return nil
	}
	hasStructuredFinding := strings.Contains(findingsText, "## ") &&
		(strings.Contains(findingsText, "- Severity:") || strings.Contains(strings.ToLower(findingsText), "severity"))
	if hasStructuredFinding && !findingsReportsNoFindings(findingsText) {
		return nil
	}
	return []runQualitySignal{{
		Code:     "artifact_quality.empty_findings_with_gaps",
		Severity: "warning",
		Message:  "artifact_quality: findings are empty while critical coverage gaps remain",
		Path:     "reports/findings/findings.md",
	}}
}

func semanticScaffoldSignals(finalIndex contracts.FinalRunIndex, inventory runArtifactInventory) []runQualitySignal {
	semantic := finalIndex.Semantic
	entities := len(semantic.Entities)
	edges := len(semantic.Edges)
	findings := len(semantic.Findings)
	if inventory.Semantic.CanonicalDocuments < 8 || entities < 4 || edges < 3 || findings == 0 {
		return nil
	}
	containerEdges := 0
	for _, edge := range semantic.Edges {
		edgeType := strings.ToLower(strings.TrimSpace(edge.Type))
		edgeName := strings.ToLower(strings.TrimSpace(edge.Name))
		if edgeType == "contains" || strings.Contains(edgeName, "contains scoped surface") {
			containerEdges++
		}
	}
	if containerEdges != edges {
		return nil
	}
	genericOwnerFindings := 0
	for _, finding := range semantic.Findings {
		title := strings.ToLower(strings.TrimSpace(finding.Title))
		description := strings.ToLower(strings.TrimSpace(finding.Description))
		ruleID := strings.ToLower(strings.TrimSpace(finding.RuleID))
		if title == "owner mapping not confirmed" &&
			strings.Contains(ruleID, "owner.mapping") &&
			strings.Contains(description, "scoped evidence identifies") &&
			strings.Contains(description, "does not confirm an owning team") {
			genericOwnerFindings++
		}
	}
	if genericOwnerFindings*100 < findings*80 {
		return nil
	}
	return []runQualitySignal{{
		Code:     "artifact_quality.semantic_scaffold_only",
		Severity: "warning",
		Message:  "artifact_quality: semantic model is non-empty but only contains scaffold-like repo/shard containment and generic owner-gap findings",
		Path:     inventory.FinalIndexPath,
	}}
}

func gapOnlyDiagramSignals(ws workspace.Root, inventory runArtifactInventory, nontrivial bool) []runQualitySignal {
	if !nontrivial || inventory.Semantic.Entities > 0 {
		return nil
	}
	signals := []runQualitySignal{}
	for _, rel := range []string{"reports/diagrams/c4-context.mmd", "reports/diagrams/c4-container.mmd"} {
		text, ok := readWorkspaceText(ws, rel)
		if !ok {
			continue
		}
		if strings.Contains(text, "Gap:") {
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.c4_gap_only",
				Severity: "warning",
				Message:  "artifact_quality: C4 diagram is diagnostic gap-only because the semantic model has no entities: " + rel,
				Path:     rel,
			})
		}
	}
	return signals
}

func scaffoldDiagramSignals(ws workspace.Root, inventory runArtifactInventory, nontrivial bool) []runQualitySignal {
	if !nontrivial || inventory.Semantic.Entities < 4 {
		return nil
	}
	text, ok := readWorkspaceText(ws, "reports/diagrams/c4-context.mmd")
	if !ok {
		return nil
	}
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "gap: no evidence-backed external systems") ||
		!strings.Contains(lower, "gap: no evidence-backed actors") ||
		!strings.Contains(lower, "gap: no evidence-backed relationships") {
		return nil
	}
	if strings.Contains(text, "Service:") || strings.Contains(text, "External:") || strings.Contains(text, "Actor:") {
		return nil
	}
	return []runQualitySignal{{
		Code:     "artifact_quality.c4_context_scaffold_only",
		Severity: "warning",
		Message:  "artifact_quality: C4 context diagram still shows only diagnostic gap nodes despite a non-empty semantic model",
		Path:     "reports/diagrams/c4-context.mmd",
	}}
}

func hiddenProviderDocumentSignals(ws workspace.Root, runID string) []runQualitySignal {
	root := strings.TrimSpace(ws.Path)
	runID = strings.TrimSpace(runID)
	if root == "" || runID == "" {
		return nil
	}
	stagingRoot := filepath.Join(root, "reports", "taskruns", runID, "staging")
	signals := []runQualitySignal{}
	_ = filepath.WalkDir(stagingRoot, func(absPath string, entry fs.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() || entry.Name() != "shard-pack-manifest.json" {
			return nil
		}
		raw, readErr := os.ReadFile(absPath)
		if readErr != nil {
			return nil
		}
		var payload struct {
			Documents []struct {
				Path string `json:"path"`
			} `json:"documents"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil
		}
		relManifest, relErr := filepath.Rel(root, absPath)
		if relErr != nil {
			relManifest = absPath
		}
		for _, document := range payload.Documents {
			component := forbiddenProviderArtifactPathComponent(document.Path)
			if component == "" {
				continue
			}
			rel := filepath.ToSlash(relManifest)
			signals = append(signals, runQualitySignal{
				Code:     "artifact_quality.hidden_provider_document",
				Severity: "warning",
				Message:  "artifact_quality: shard manifest references provider/tool side-effect document path component " + component,
				Path:     rel,
			})
		}
		return nil
	})
	return signals
}

func forbiddenProviderArtifactPathComponent(rawPath string) string {
	normalized := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(rawPath))))
	for _, component := range strings.Split(normalized, "/") {
		trimmed := strings.TrimSpace(strings.ToLower(component))
		if trimmed == "node_modules" || strings.HasPrefix(trimmed, ".") {
			return component
		}
	}
	return ""
}

func readWorkspaceText(ws workspace.Root, rel string) (string, bool) {
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
