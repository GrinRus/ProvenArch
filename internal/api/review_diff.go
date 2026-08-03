package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type runReviewStepDefinition struct {
	key    string
	suffix string
	label  string
}

var runReviewStepDefinitions = []runReviewStepDefinition{
	{key: acpruntime.StepProviderStep0Constitution, suffix: "step0.constitution", label: "Charter"},
	{key: acpruntime.StepProviderStep1Collect, suffix: "step1.collect", label: "Collect"},
	{key: acpruntime.StepProviderStep2AsIs, suffix: "step2.asis_docs", label: "As-is docs"},
	{key: acpruntime.StepProviderStep3Findings, suffix: "step3.findings", label: "Findings"},
	{key: acpruntime.StepProviderStep4Proposals, suffix: "step4.proposals", label: "Proposals"},
}

type runReviewStepSummary struct {
	StepID        string   `json:"step_id"`
	Key           string   `json:"key"`
	Label         string   `json:"label"`
	State         string   `json:"state"`
	Provider      string   `json:"provider"`
	ArtifactCount int      `json:"artifact_count"`
	ArtifactPaths []string `json:"artifact_paths"`
	TaskrunPaths  []string `json:"taskrun_paths"`
	WarningsCount int      `json:"warnings_count"`
	ErrorsCount   int      `json:"errors_count"`
	LastMessage   string   `json:"last_message"`
}

func (s *Server) handlePipelineRunReviewSummary(writer http.ResponseWriter, runID string) {
	snapshot := s.sessionSnapshot()
	if snapshot.Service == nil {
		writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select or create an ACP workspace before reading run review summary")
		return
	}
	runInfo, ok := snapshot.Service.GetRun(runID)
	if !ok {
		writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
		return
	}
	artifacts, _ := snapshot.Service.GetRunArtifacts(runID)
	logs, err := readAllRunLogs(snapshot.Service, runID)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "run_logs_unavailable", err.Error())
		return
	}
	steps := buildRunReviewSteps(runInfo, artifacts, logs, snapshot.RuntimeConfig.Mode)
	result, recovery := buildRunOutcome(snapshot.Workspace.Path, runInfo, artifacts, steps, logs, previousPromotedRunID(snapshot.Service, runInfo))
	writeJSON(writer, http.StatusOK, map[string]any{
		"run_id":       runInfo.RunID,
		"pipeline":     runInfo.Pipeline,
		"status":       runInfo.Status,
		"started_at":   runInfo.StartedAt.UTC().Format(time.RFC3339),
		"finished_at":  formatOptionalTime(runInfo.FinishedAt),
		"current_step": runInfo.CurrentStep,
		"warnings":     runInfo.Warnings,
		"error_code":   formatOptionalString(runInfo.ErrorCode),
		"error":        formatOptionalString(runInfo.Error),
		"steps":        steps,
		"result":       result,
		"recovery":     recovery,
		"progress":     runInfo.Progress,
		"retry":        runInfo.Retry,
	})
}

type runOutcomeSummary struct {
	State             string            `json:"state"`
	Summary           string            `json:"summary"`
	Produced          map[string]int    `json:"produced"`
	PartialScopes     int               `json:"partial_scopes"`
	FailedScopes      int               `json:"failed_scopes"`
	Promotion         runPromotionState `json:"promotion"`
	RecommendedAction string            `json:"recommended_action"`
	Coverage          runCoverageState  `json:"coverage"`
}

type runCoverageState struct {
	Observed int    `json:"observed"`
	Missing  int    `json:"missing"`
	Status   string `json:"status"`
}

type runPromotionState struct {
	Changed       bool   `json:"changed"`
	CurrentUsable bool   `json:"current_usable"`
	BaselineRunID string `json:"baseline_run_id,omitempty"`
}

type runRecoverySummary struct {
	Category         string   `json:"category"`
	Title            string   `json:"title"`
	Explanation      string   `json:"explanation"`
	Impact           string   `json:"impact"`
	RetainedEvidence string   `json:"retained_evidence"`
	RecommendedFix   string   `json:"recommended_fix"`
	CanRetry         bool     `json:"can_retry"`
	FailedStep       string   `json:"failed_step,omitempty"`
	FailedScopes     []string `json:"failed_scopes,omitempty"`
	TechnicalCode    string   `json:"technical_code,omitempty"`
}

func buildRunOutcome(root string, info orchestrator.RunInfo, artifacts []orchestrator.Artifact, steps []runReviewStepSummary, logs []orchestrator.RunLogEntry, previousBaseline string) (runOutcomeSummary, *runRecoverySummary) {
	knowledgeRoot := root
	if info.Status == orchestrator.RunStatusSucceeded && hasArchitectureSnapshot(artifacts) {
		knowledgeRoot = filepath.Join(root, "reports", "taskruns", info.RunID, "promoted-snapshot")
	}
	knowledge := collectKnowledge(knowledgeRoot)
	semantic := loadRunSemanticSnapshot(root, info.RunID)
	if info.Status == orchestrator.RunStatusSucceeded && hasArchitectureSnapshot(artifacts) {
		semantic = loadPromotedSemanticSnapshot(root, info.RunID)
	}
	produced := map[string]int{"entities": len(knowledge.Entities), "edges": len(knowledge.Edges), "diagrams": 0, "findings": 0, "questions": 0, "proposals": 0, "artifacts": len(artifacts)}
	if info.Status != orchestrator.RunStatusSucceeded {
		produced["entities"], produced["edges"] = 0, 0
	}
	for _, artifact := range artifacts {
		path := strings.ToLower(artifact.Path)
		switch {
		case strings.Contains(path, "/diagrams/"):
			produced["diagrams"]++
		case strings.Contains(path, "/findings/"):
			produced["findings"]++
		case strings.Contains(path, "open-questions"):
			produced["questions"]++
		case strings.HasPrefix(path, "proposals/") || strings.Contains(path, "/proposals/"):
			produced["proposals"]++
		}
	}
	if len(semantic.Findings) > 0 {
		produced["findings"] = len(semantic.Findings)
	}
	if len(semantic.Questions) > 0 {
		produced["questions"] = len(semantic.Questions)
	}
	partial, failed := runScopeCounts(logs)
	for _, step := range steps {
		if partial == 0 && step.WarningsCount > 0 {
			partial++
		}
		if failed == 0 && step.State == "failed" {
			failed++
		}
	}
	state := "failed"
	summary := "Analysis did not replace the last validated architecture."
	action := "Review failure"
	promotionChanged := false
	currentUsable := len(knowledge.Entities)+len(knowledge.Edges)+len(knowledge.Artifacts) > 0
	switch info.Status {
	case orchestrator.RunStatusSucceeded:
		state, summary, action = "completed", "Validated architecture knowledge is ready to explore.", "explore_architecture"
		if len(info.Warnings) > 0 || knowledge.Status == "partial" {
			state, summary, action = "completed_with_gaps", "Architecture is usable, with named evidence gaps that need review.", "review_gaps"
		}
		promotionChanged = hasArchitectureSnapshot(artifacts) && (info.RefreshSummary == nil || info.RefreshSummary.Mode != "no_op")
	case orchestrator.RunStatusCanceled:
		state, summary, action = "canceled", "Analysis was canceled; retained evidence and the last validated architecture remain available.", "review_or_retry"
	}
	baseline := previousBaseline
	if info.RefreshSummary != nil && strings.TrimSpace(info.RefreshSummary.BaselineRunID) != "" {
		baseline = info.RefreshSummary.BaselineRunID
	}
	coverageStatus := "available"
	if len(semantic.Coverage.Missing) > 0 {
		coverageStatus = "partial"
	} else if len(semantic.Coverage.Observed) == 0 {
		coverageStatus = "unavailable"
	}
	outcome := runOutcomeSummary{State: state, Summary: summary, Produced: produced, PartialScopes: partial, FailedScopes: failed, Promotion: runPromotionState{Changed: promotionChanged, CurrentUsable: currentUsable, BaselineRunID: baseline}, RecommendedAction: action, Coverage: runCoverageState{Observed: len(semantic.Coverage.Observed), Missing: len(semantic.Coverage.Missing), Status: coverageStatus}}
	if info.Status == orchestrator.RunStatusSucceeded {
		return outcome, nil
	}
	category, title, fix, safeRetry := classifyRecovery(info.ErrorCode)
	failedScopes := failedScopesFromLogs(logs)
	if len(failedScopes) == 0 {
		failedScopes = failedScopesFromError(info.Error)
	}
	recovery := &runRecoverySummary{Category: category, Title: title, Explanation: strings.TrimSpace(info.Error), Impact: "The attempted run was not promoted; current architecture remains on the last validator-approved generation.", RetainedEvidence: "Run-scoped artifacts remain available for audit. The retry planner revalidates every reused input before execution.", RecommendedFix: fix, CanRetry: safeRetry && (info.Status == orchestrator.RunStatusFailed || info.Status == orchestrator.RunStatusCanceled), FailedStep: info.CurrentStep, FailedScopes: failedScopes, TechnicalCode: info.ErrorCode}
	return outcome, recovery
}

func previousPromotedRunID(service *orchestrator.Service, current orchestrator.RunInfo) string {
	if service == nil {
		return ""
	}
	candidates := []orchestrator.RunInfo{}
	for _, run := range service.ListRuns(0) {
		if run.RunID == current.RunID || run.Status != orchestrator.RunStatusSucceeded || !run.StartedAt.Before(current.StartedAt) {
			continue
		}
		artifacts, _ := service.GetRunArtifacts(run.RunID)
		if hasArchitectureSnapshot(artifacts) {
			candidates = append(candidates, run)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].StartedAt.After(candidates[j].StartedAt) })
	if len(candidates) > 0 {
		return candidates[0].RunID
	}
	return ""
}

var failedShardPattern = regexp.MustCompile(`(?i)shard\s+"([^"]+)"`)

func failedScopesFromError(message string) []string {
	set := map[string]struct{}{}
	for _, match := range failedShardPattern.FindAllStringSubmatch(message, -1) {
		if len(match) > 1 && strings.TrimSpace(match[1]) != "" {
			set[strings.TrimSpace(match[1])] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for scope := range set {
		result = append(result, scope)
	}
	sort.Strings(result)
	return result
}

func classifyRecovery(code string) (string, string, string, bool) {
	lower := strings.ToLower(strings.TrimSpace(code))
	switch {
	case strings.Contains(lower, "permission"):
		return "permission", "Runtime permission is required", "Review the blocked permission and runner policy, then calculate a retry plan.", true
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "stall"):
		return "timeout", "Analysis stopped making useful progress", "Inspect the last useful artifact progress and provider health, then retry the affected step.", true
	case strings.Contains(lower, "runner") || strings.Contains(lower, "provider"):
		return "provider", "The selected provider could not complete the work", "Verify provider readiness or select another configured provider before retrying.", true
	case strings.Contains(lower, "evidence") || strings.Contains(lower, "citation") || strings.Contains(lower, "artifact_quality"):
		return "evidence", "Collected evidence is incomplete or unusable", "Inspect the named evidence gap, correct repository scope if needed, then calculate a retry plan.", true
	case strings.Contains(lower, "contract") || strings.Contains(lower, "validation"):
		return "contract", "Generated output failed validation", "Inspect the failed artifact contract and retry the affected step after correcting the cause.", true
	case strings.Contains(lower, "workspace") || strings.Contains(lower, "source"):
		return "setup", "Workspace or source configuration is not ready", "Open Setup and resolve the named workspace or repository blocker.", true
	case strings.Contains(lower, "cancel"):
		return "canceled", "Analysis was canceled", "Review retained work and calculate a retry plan when ready.", true
	default:
		return "infrastructure", "Analysis could not complete", "Inspect technical details and logs. Retry is disabled until the failure is classified safely.", false
	}
}

func failedScopesFromLogs(logs []orchestrator.RunLogEntry) []string {
	set := map[string]struct{}{}
	for _, entry := range logs {
		if strings.EqualFold(string(entry.Level), "error") || strings.Contains(strings.ToLower(entry.Message), "failed") {
			if scope := strings.TrimSpace(entry.DomainID); scope != "" {
				set[scope] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(set))
	for scope := range set {
		result = append(result, scope)
	}
	sort.Strings(result)
	return result
}

func runScopeCounts(logs []orchestrator.RunLogEntry) (int, int) {
	partial, failed := map[string]struct{}{}, map[string]struct{}{}
	for _, entry := range logs {
		scope := strings.TrimSpace(entry.DomainID)
		if scope == "" {
			continue
		}
		if strings.EqualFold(string(entry.Level), "warning") || strings.EqualFold(string(entry.Level), "warn") {
			partial[scope] = struct{}{}
		}
		if strings.EqualFold(string(entry.Level), "error") || strings.Contains(strings.ToLower(entry.Message), "failed") {
			failed[scope] = struct{}{}
		}
	}
	return len(partial), len(failed)
}

func readAllRunLogs(service *orchestrator.Service, runID string) ([]orchestrator.RunLogEntry, error) {
	logs := []orchestrator.RunLogEntry{}
	cursor := 0
	for pageCount := 0; pageCount < 50; pageCount += 1 {
		page, ok, err := service.GetRunLogs(runID, cursor, 500)
		if !ok {
			return nil, orchestrator.ErrRunNotFound
		}
		if err != nil {
			return nil, err
		}
		logs = append(logs, page.Items...)
		if page.EOF {
			break
		}
		if page.NextCursor <= cursor {
			break
		}
		cursor = page.NextCursor
	}
	return logs, nil
}

func buildRunReviewSteps(runInfo orchestrator.RunInfo, artifacts []orchestrator.Artifact, logs []orchestrator.RunLogEntry, runtimeMode string) []runReviewStepSummary {
	currentIndex := runReviewStepIndex(runInfo.CurrentStep)
	loggedIndex := -1
	for _, entry := range logs {
		if index := runReviewStepIndex(entry.StepID); index > loggedIndex {
			loggedIndex = index
		}
	}
	activeIndex := currentIndex
	if activeIndex < 0 {
		activeIndex = loggedIndex
	}
	if activeIndex < 0 && (runInfo.Status == orchestrator.RunStatusQueued || runInfo.Status == orchestrator.RunStatusRunning || runInfo.Status == orchestrator.RunStatusFailed) {
		activeIndex = 0
	}

	steps := make([]runReviewStepSummary, 0, len(runReviewStepDefinitions))
	for index, definition := range runReviewStepDefinitions {
		stepID := runInfo.Pipeline + "." + definition.suffix
		stepArtifacts := artifactPathsForReviewStep(definition.key, artifacts)
		stepLogs := logsForReviewStep(definition.key, logs)
		taskrunPaths := taskrunPathsForLogs(stepLogs)
		warnings, errorsCount := countReviewLogLevels(stepLogs, runInfo.Warnings)
		lastMessage := lastReviewLogMessage(stepLogs)
		if lastMessage == "" && index == activeIndex {
			lastMessage = strings.TrimSpace(runInfo.Error)
			if lastMessage == "" {
				lastMessage = strings.TrimSpace(runInfo.ErrorCode)
			}
		}
		steps = append(steps, runReviewStepSummary{
			StepID:        stepID,
			Key:           definition.key,
			Label:         definition.label,
			State:         runReviewStepState(runInfo.Status, index, activeIndex),
			Provider:      runReviewProviderLabel(runInfo.StepProviders[definition.key], runtimeMode),
			ArtifactCount: len(stepArtifacts),
			ArtifactPaths: stepArtifacts,
			TaskrunPaths:  taskrunPaths,
			WarningsCount: warnings,
			ErrorsCount:   errorsCount,
			LastMessage:   lastMessage,
		})
	}
	return steps
}

func runReviewStepState(status orchestrator.RunStatus, index int, activeIndex int) string {
	switch status {
	case orchestrator.RunStatusSucceeded:
		return "done"
	case orchestrator.RunStatusFailed, orchestrator.RunStatusCanceled:
		if index < activeIndex {
			return "done"
		}
		if index == activeIndex {
			return "failed"
		}
		return "pending"
	case orchestrator.RunStatusQueued, orchestrator.RunStatusRunning:
		if index < activeIndex {
			return "done"
		}
		if index == activeIndex {
			return "active"
		}
		return "pending"
	default:
		return "pending"
	}
}

func runReviewStepIndex(stepID string) int {
	stepID = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(stepID)), "_", ".")
	for index, definition := range runReviewStepDefinitions {
		if strings.Contains(stepID, strings.ReplaceAll(definition.suffix, "_", ".")) || strings.Contains(stepID, strings.ReplaceAll(definition.key, "_", ".")) {
			return index
		}
	}
	return -1
}

func reviewStepKeyForPath(relPath string) string {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(relPath)), "\\", "/")
	if normalized == "" {
		return ""
	}
	if strings.Contains(normalized, "step0_constitution") || strings.Contains(normalized, "step0.constitution") {
		return acpruntime.StepProviderStep0Constitution
	}
	if strings.Contains(normalized, "step1_collect") || strings.Contains(normalized, "step1.collect") {
		return acpruntime.StepProviderStep1Collect
	}
	if strings.Contains(normalized, "step2_as_is") || strings.Contains(normalized, "step2.asis") || strings.Contains(normalized, "step2.as-is") {
		return acpruntime.StepProviderStep2AsIs
	}
	if strings.Contains(normalized, "step3_findings") || strings.Contains(normalized, "step3.findings") {
		return acpruntime.StepProviderStep3Findings
	}
	if strings.Contains(normalized, "step4_proposals") || strings.Contains(normalized, "step4.proposals") {
		return acpruntime.StepProviderStep4Proposals
	}
	switch {
	case strings.HasPrefix(normalized, "charter/") || strings.HasPrefix(normalized, "skills/"):
		return acpruntime.StepProviderStep0Constitution
	case strings.HasPrefix(normalized, "docs/imports/") || strings.Contains(normalized, "shard-pack-manifest"):
		return acpruntime.StepProviderStep1Collect
	case strings.HasPrefix(normalized, "reports/as-is/") || strings.HasPrefix(normalized, "reports/diagrams/"):
		return acpruntime.StepProviderStep2AsIs
	case strings.HasPrefix(normalized, "reports/findings/") || strings.HasPrefix(normalized, "reports/coverage/") || strings.HasPrefix(normalized, "model/"):
		return acpruntime.StepProviderStep3Findings
	case strings.HasPrefix(normalized, "proposals/") || strings.HasPrefix(normalized, "reports/changelog/"):
		return acpruntime.StepProviderStep4Proposals
	}
	return ""
}

func artifactPathsForReviewStep(stepKey string, artifacts []orchestrator.Artifact) []string {
	paths := []string{}
	seen := map[string]struct{}{}
	for _, artifact := range artifacts {
		rel := strings.TrimSpace(artifact.Path)
		if rel == "" || reviewStepKeyForPath(rel) != stepKey {
			continue
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	return paths
}

func logsForReviewStep(stepKey string, logs []orchestrator.RunLogEntry) []orchestrator.RunLogEntry {
	entries := []orchestrator.RunLogEntry{}
	for _, entry := range logs {
		if acpruntime.StepProviderKeyForStepID(entry.StepID) == stepKey || reviewStepKeyForPath(entry.TaskrunPath) == stepKey {
			entries = append(entries, entry)
		}
	}
	return entries
}

func taskrunPathsForLogs(logs []orchestrator.RunLogEntry) []string {
	seen := map[string]struct{}{}
	paths := []string{}
	for _, entry := range logs {
		rel := strings.TrimSpace(entry.TaskrunPath)
		if rel == "" {
			continue
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	return paths
}

func countReviewLogLevels(logs []orchestrator.RunLogEntry, runWarnings []string) (warnings int, errorsCount int) {
	runWarningMessages := map[string]struct{}{}
	for _, warning := range runWarnings {
		for _, variant := range reviewWarningMessageVariants(warning) {
			runWarningMessages[variant] = struct{}{}
		}
	}
	for _, entry := range logs {
		switch entry.Level {
		case orchestrator.RunLogLevelWarning:
			if _, exists := runWarningMessages[normalizeReviewWarningMessage(reviewWarningLogMessage(entry))]; exists {
				continue
			}
			warnings += 1
		case orchestrator.RunLogLevelError:
			errorsCount += 1
		}
	}
	return warnings, errorsCount
}

func runReviewProviderLabel(provider string, runtimeMode string) string {
	if strings.TrimSpace(runtimeMode) == acpruntime.RuntimeModeFake {
		return "fake"
	}
	return strings.TrimSpace(provider)
}

func normalizeReviewWarningMessage(message string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(message)), " ")
}

func reviewWarningLogMessage(entry orchestrator.RunLogEntry) string {
	if value, ok := entry.Fields["warning"].(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return entry.Message
}

func reviewWarningMessageVariants(message string) []string {
	normalized := normalizeReviewWarningMessage(message)
	if normalized == "" {
		return nil
	}
	variants := []string{normalized}
	if separator := strings.Index(normalized, ": "); separator >= 0 {
		if suffix := strings.TrimSpace(normalized[separator+2:]); suffix != "" {
			variants = append(variants, suffix)
		}
	}
	return variants
}

func lastReviewLogMessage(logs []orchestrator.RunLogEntry) string {
	for index := len(logs) - 1; index >= 0; index -= 1 {
		if message := strings.TrimSpace(logs[index].Message); message != "" {
			return message
		}
	}
	return ""
}

type gitDiffFile struct {
	Path           string  `json:"path"`
	OriginalPath   *string `json:"original_path"`
	Folder         string  `json:"folder"`
	Status         string  `json:"status"`
	IndexStatus    string  `json:"index_status"`
	WorktreeStatus string  `json:"worktree_status"`
	OldMode        string  `json:"old_mode,omitempty"`
	NewMode        string  `json:"new_mode,omitempty"`
	HeadOID        string  `json:"head_oid,omitempty"`
	IndexOID       string  `json:"index_oid,omitempty"`
	WorktreeSHA256 string  `json:"worktree_sha256,omitempty"`
	Additions      int     `json:"additions"`
	Deletions      int     `json:"deletions"`
	Binary         bool    `json:"binary"`
	Unavailable    bool    `json:"unavailable"`
}

type gitWorkspaceIdentity struct {
	Branch  string `json:"branch"`
	HeadOID string `json:"head_oid"`
	BaseRef string `json:"base_ref"`
	BaseOID string `json:"base_oid"`
}

type gitWorkspaceState struct {
	Identity    gitWorkspaceIdentity
	Files       []gitDiffFile
	Fingerprint string
}

type gitDiffFolderSummary struct {
	Folder    string `json:"folder"`
	Files     int    `json:"files"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type gitDiffLine struct {
	Kind    string `json:"kind"`
	OldLine *int   `json:"old_line,omitempty"`
	NewLine *int   `json:"new_line,omitempty"`
	Content string `json:"content"`
}

type gitDiffHunk struct {
	Header string        `json:"header"`
	Lines  []gitDiffLine `json:"lines"`
}

func (s *Server) handleGitDiff(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	query := request.URL.Query()
	pathFilter, err := normalizeOptionalWorkspacePath(query.Get("path"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "path_invalid", err.Error())
		return
	}
	folderFilter, err := normalizeOptionalWorkspacePath(query.Get("folder"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "folder_invalid", err.Error())
		return
	}
	runID := strings.TrimSpace(query.Get("run_id"))
	stepID := strings.TrimSpace(query.Get("step_id"))

	snapshot := s.sessionSnapshot()
	ws := snapshot.Workspace
	state, err := collectWorkspaceGitState(request.Context(), ws)
	if err != nil {
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok": false, "workspace": ws.Path, "scope": "full_workspace",
			"state": "unknown", "files": []gitDiffFile{}, "folders": []gitDiffFolderSummary{},
			"hunks": []gitDiffHunk{}, "empty": true, "message": err.Error(),
		})
		return
	}
	gitState := deriveGitWorkspaceState(state.Files, snapshot.Service.Coordination(), query.Get("fingerprint"), state.Fingerprint)
	files := state.Files
	folders := summarizeGitDiffFolders(files)
	selectedPath := pathFilter
	if selectedPath == "" && folderFilter != "" {
		for _, file := range files {
			if file.Path == folderFilter || strings.HasPrefix(file.Path, strings.TrimRight(folderFilter, "/")+"/") {
				selectedPath = file.Path
				break
			}
		}
	}
	if selectedPath == "" && runID != "" {
		allowed := artifactPathFilterForRun(runID, stepID, snapshot.Service)
		for _, file := range files {
			if _, ok := allowed[file.Path]; ok {
				selectedPath = file.Path
				break
			}
		}
	}
	if selectedPath == "" && len(files) > 0 {
		selectedPath = files[0].Path
	}
	selectedFile, hunks, selectedMessage, err := selectedGitDiff(request.Context(), ws, selectedPath, files)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "git_diff_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":            true,
		"state":         gitState,
		"workspace":     ws.Path,
		"scope":         "full_workspace",
		"branch":        state.Identity.Branch,
		"head_oid":      formatOptionalString(state.Identity.HeadOID),
		"base_ref":      state.Identity.BaseRef,
		"base_oid":      formatOptionalString(state.Identity.BaseOID),
		"fingerprint":   state.Fingerprint,
		"run_id":        formatOptionalString(runID),
		"step_id":       formatOptionalString(stepID),
		"selected_path": formatOptionalString(selectedPath),
		"selected_file": selectedFile,
		"files":         files,
		"folders":       folders,
		"hunks":         hunks,
		"message":       selectedMessage,
		"empty":         len(files) == 0,
	})
}

func deriveGitWorkspaceState(files []gitDiffFile, coordination orchestrator.RunCoordination, expectedFingerprint string, actualFingerprint string) string {
	if coordination.ActiveRunID != "" || coordination.Pending != nil {
		return "blocked"
	}
	expectedFingerprint = strings.TrimSpace(expectedFingerprint)
	if expectedFingerprint != "" && expectedFingerprint != actualFingerprint {
		return "stale"
	}
	if len(files) > 0 {
		return "dirty"
	}
	return "clean"
}

func normalizeOptionalWorkspacePath(rawPath string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", nil
	}
	if strings.ContainsRune(rawPath, 0) {
		return "", errors.New("path contains NUL")
	}
	if filepath.IsAbs(rawPath) {
		return "", errors.New("absolute paths are not allowed")
	}
	normalized := strings.ReplaceAll(rawPath, "\\", "/")
	if path.IsAbs(normalized) || isWindowsAbsolutePath(normalized) {
		return "", errors.New("absolute paths are not allowed")
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return "", errors.New("path traversal is not allowed")
		}
	}
	clean := path.Clean(normalized)
	if clean == "." {
		return "", nil
	}
	return clean, nil
}

func collectWorkspaceGitState(ctx context.Context, ws workspace.Root) (gitWorkspaceState, error) {
	statusOutput, err := runGitRaw(ctx, ws.Path, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return gitWorkspaceState{}, err
	}
	files := parseGitStatusFiles(statusOutput)
	for index := range files {
		files[index] = enrichGitDiffFile(ctx, ws, files[index])
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Path == files[j].Path {
			return optionalStringValue(files[i].OriginalPath) < optionalStringValue(files[j].OriginalPath)
		}
		return files[i].Path < files[j].Path
	})
	identity, err := collectGitWorkspaceIdentity(ctx, ws.Path)
	if err != nil {
		return gitWorkspaceState{}, err
	}
	fingerprint, err := fingerprintGitWorkspaceState(identity, files)
	if err != nil {
		return gitWorkspaceState{}, err
	}
	return gitWorkspaceState{Identity: identity, Files: files, Fingerprint: fingerprint}, nil
}

func artifactPathFilterForRun(runID string, stepID string, service *orchestrator.Service) map[string]struct{} {
	runID = strings.TrimSpace(runID)
	stepID = strings.TrimSpace(stepID)
	if runID == "" || service == nil {
		return nil
	}
	artifacts, ok := service.GetRunArtifacts(runID)
	if !ok {
		return map[string]struct{}{}
	}
	stepKey := ""
	if stepID != "" {
		stepKey = acpruntime.StepProviderKeyForStepID(stepID)
	}
	allowed := map[string]struct{}{}
	for _, artifact := range artifacts {
		rel := strings.TrimSpace(artifact.Path)
		if rel == "" {
			continue
		}
		if stepKey != "" && reviewStepKeyForPath(rel) != stepKey {
			continue
		}
		allowed[rel] = struct{}{}
	}
	return allowed
}

func parseGitStatusFiles(output string) []gitDiffFile {
	parts := strings.Split(output, "\x00")
	files := []gitDiffFile{}
	for index := 0; index < len(parts); index += 1 {
		item := parts[index]
		if len(item) < 4 {
			continue
		}
		code := item[:2]
		rel := strings.TrimSpace(item[3:])
		var originalPath *string
		if strings.Contains(code, "R") || strings.Contains(code, "C") {
			if index+1 < len(parts) && strings.TrimSpace(parts[index+1]) != "" {
				original := filepath.ToSlash(strings.TrimSpace(parts[index+1]))
				originalPath = &original
				index += 1
			}
		}
		if rel == "" {
			continue
		}
		files = append(files, gitDiffFile{
			Path:           filepath.ToSlash(rel),
			OriginalPath:   originalPath,
			Folder:         gitDiffFolder(filepath.ToSlash(rel)),
			Status:         gitDiffStatusLabel(code),
			IndexStatus:    gitStatusColumn(code, 0),
			WorktreeStatus: gitStatusColumn(code, 1),
		})
	}
	return files
}

func gitDiffStatusLabel(code string) string {
	code = strings.TrimSpace(code)
	switch {
	case code == "??":
		return "untracked"
	case strings.Contains(code, "R"):
		return "renamed"
	case strings.Contains(code, "C"):
		return "copied"
	case strings.Contains(code, "D"):
		return "deleted"
	case strings.Contains(code, "A"):
		return "new"
	case strings.Contains(code, "M"):
		return "modified"
	default:
		return "changed"
	}
}

func gitStatusColumn(code string, index int) string {
	if index < 0 || index >= len(code) || code[index] == ' ' {
		return "clean"
	}
	if code[index] == '?' {
		return "untracked"
	}
	return string(code[index])
}

func gitDiffFolder(rel string) string {
	parts := strings.Split(strings.Trim(rel, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "root"
	}
	return parts[0]
}

func enrichGitDiffFile(ctx context.Context, ws workspace.Root, file gitDiffFile) gitDiffFile {
	file.Binary = workspaceFileIsBinary(ws, file.Path)
	file.OldMode, file.HeadOID = gitHeadEntry(ctx, ws.Path, file.Path)
	file.NewMode, file.IndexOID = gitIndexEntry(ctx, ws.Path, file.Path)
	if content, err := ws.ReadFile(file.Path); err == nil {
		sum := sha256.Sum256(content)
		file.WorktreeSHA256 = hex.EncodeToString(sum[:])
	} else if file.Status != "deleted" {
		file.Unavailable = true
	}
	switch file.Status {
	case "untracked", "new":
		if !file.Binary {
			file.Additions = countWorkspaceTextLines(ws, file.Path)
		}
	case "deleted":
		file.Deletions = countGitHeadTextLines(ctx, ws, file.Path)
	default:
		additions, deletions, binary := gitNumstatForPath(ctx, ws, file.Path)
		file.Additions = additions
		file.Deletions = deletions
		file.Binary = file.Binary || binary
	}
	return file
}

func gitHeadEntry(ctx context.Context, repoPath string, rel string) (string, string) {
	output, err := runGit(ctx, repoPath, "ls-tree", "HEAD", "--", rel)
	if err != nil || strings.TrimSpace(output) == "" {
		return "", ""
	}
	fields := strings.Fields(output)
	if len(fields) < 3 {
		return "", ""
	}
	return fields[0], fields[2]
}

func gitIndexEntry(ctx context.Context, repoPath string, rel string) (string, string) {
	output, err := runGit(ctx, repoPath, "ls-files", "--stage", "--", rel)
	if err != nil || strings.TrimSpace(output) == "" {
		return "", ""
	}
	fields := strings.Fields(output)
	if len(fields) < 2 {
		return "", ""
	}
	return fields[0], fields[1]
}

func collectGitWorkspaceIdentity(ctx context.Context, repoPath string) (gitWorkspaceIdentity, error) {
	identity := gitWorkspaceIdentity{Branch: "DETACHED"}
	if branch, err := runGit(ctx, repoPath, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil && strings.TrimSpace(branch) != "" {
		identity.Branch = strings.TrimSpace(branch)
	}
	if head, err := runGit(ctx, repoPath, "rev-parse", "--verify", "HEAD"); err == nil {
		identity.HeadOID = strings.TrimSpace(head)
	}
	identity.BaseRef = identity.Branch
	identity.BaseOID = identity.HeadOID
	if upstream, err := runGit(ctx, repoPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"); err == nil && strings.TrimSpace(upstream) != "" {
		identity.BaseRef = strings.TrimSpace(upstream)
		if base, baseErr := runGit(ctx, repoPath, "merge-base", "HEAD", identity.BaseRef); baseErr == nil {
			identity.BaseOID = strings.TrimSpace(base)
		}
	}
	return identity, nil
}

func fingerprintGitWorkspaceState(identity gitWorkspaceIdentity, files []gitDiffFile) (string, error) {
	payload := struct {
		Identity gitWorkspaceIdentity `json:"identity"`
		Files    []gitDiffFile        `json:"files"`
	}{Identity: identity, Files: files}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal git fingerprint manifest: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func gitNumstatForPath(ctx context.Context, ws workspace.Root, rel string) (int, int, bool) {
	output, err := runGit(ctx, ws.Path, "diff", "HEAD", "--numstat", "--", rel)
	if err != nil {
		return 0, 0, false
	}
	fields := strings.Fields(output)
	if len(fields) < 2 {
		return 0, 0, false
	}
	if fields[0] == "-" || fields[1] == "-" {
		return 0, 0, true
	}
	additions, _ := strconv.Atoi(fields[0])
	deletions, _ := strconv.Atoi(fields[1])
	return additions, deletions, false
}

func workspaceFileIsBinary(ws workspace.Root, rel string) bool {
	content, err := ws.ReadFile(rel)
	if err != nil {
		return false
	}
	return bytes.IndexByte(content, 0) >= 0 || !utf8.Valid(content)
}

func countWorkspaceTextLines(ws workspace.Root, rel string) int {
	content, err := ws.ReadFile(rel)
	if err != nil || bytes.IndexByte(content, 0) >= 0 || !utf8.Valid(content) {
		return 0
	}
	return countTextLines(string(content))
}

func countGitHeadTextLines(ctx context.Context, ws workspace.Root, rel string) int {
	output, err := runGitRaw(ctx, ws.Path, "show", "HEAD:"+rel)
	if err != nil {
		return 0
	}
	return countTextLines(output)
}

func countTextLines(content string) int {
	if content == "" {
		return 0
	}
	count := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		count += 1
	}
	return count
}

func summarizeGitDiffFolders(files []gitDiffFile) []gitDiffFolderSummary {
	grouped := map[string]gitDiffFolderSummary{}
	for _, file := range files {
		summary := grouped[file.Folder]
		summary.Folder = file.Folder
		summary.Files += 1
		summary.Additions += file.Additions
		summary.Deletions += file.Deletions
		grouped[file.Folder] = summary
	}
	summaries := make([]gitDiffFolderSummary, 0, len(grouped))
	for _, summary := range grouped {
		summaries = append(summaries, summary)
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Folder < summaries[j].Folder })
	return summaries
}

func selectedGitDiff(ctx context.Context, ws workspace.Root, selectedPath string, files []gitDiffFile) (*gitDiffFile, []gitDiffHunk, string, error) {
	if selectedPath == "" {
		return nil, []gitDiffHunk{}, "No changed file selected.", nil
	}
	var selected *gitDiffFile
	for index := range files {
		if files[index].Path == selectedPath {
			selected = &files[index]
			break
		}
	}
	if selected == nil {
		if _, err := ws.ReadFile(selectedPath); err == nil {
			unchanged := gitDiffFile{Path: selectedPath, Folder: gitDiffFolder(selectedPath), Status: "unchanged"}
			return &unchanged, []gitDiffHunk{}, "Selected file has no workspace Git changes.", nil
		}
		return nil, []gitDiffHunk{}, "Selected file is not changed or no longer exists.", nil
	}
	if selected.Binary {
		return selected, []gitDiffHunk{}, "Selected file is binary; line-level diff is unavailable.", nil
	}
	switch selected.Status {
	case "untracked", "new":
		hunks := syntheticWorkspaceFileHunks(ws, selected.Path, "added")
		return selected, hunks, "", nil
	case "deleted":
		hunks := syntheticGitHeadFileHunks(ctx, ws, selected.Path, "deleted")
		return selected, hunks, "", nil
	default:
		output, err := runGitRaw(ctx, ws.Path, "diff", "HEAD", "--no-color", "--unified=3", "--", selected.Path)
		if err != nil {
			return selected, []gitDiffHunk{}, "", err
		}
		return selected, parseUnifiedDiffHunks(output), "", nil
	}
}

func syntheticWorkspaceFileHunks(ws workspace.Root, rel string, mode string) []gitDiffHunk {
	content, err := ws.ReadFile(rel)
	if err != nil {
		return []gitDiffHunk{}
	}
	return syntheticTextHunks(string(content), mode)
}

func syntheticGitHeadFileHunks(ctx context.Context, ws workspace.Root, rel string, mode string) []gitDiffHunk {
	content, err := runGitRaw(ctx, ws.Path, "show", "HEAD:"+rel)
	if err != nil {
		return []gitDiffHunk{}
	}
	return syntheticTextHunks(content, mode)
}

func syntheticTextHunks(content string, mode string) []gitDiffHunk {
	lines := splitDiffContentLines(content)
	hunk := gitDiffHunk{Header: fmt.Sprintf("@@ %s %d lines @@", mode, len(lines))}
	oldLine := 1
	newLine := 1
	for _, line := range lines {
		if mode == "deleted" {
			hunk.Lines = append(hunk.Lines, gitDiffLine{Kind: "delete", OldLine: intPtr(oldLine), Content: line})
			oldLine += 1
		} else {
			hunk.Lines = append(hunk.Lines, gitDiffLine{Kind: "add", NewLine: intPtr(newLine), Content: line})
			newLine += 1
		}
	}
	return []gitDiffHunk{hunk}
}

func splitDiffContentLines(content string) []string {
	if content == "" {
		return []string{}
	}
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func parseUnifiedDiffHunks(diffOutput string) []gitDiffHunk {
	hunks := []gitDiffHunk{}
	var current *gitDiffHunk
	oldLine := 0
	newLine := 0
	for _, line := range strings.Split(diffOutput, "\n") {
		if strings.HasPrefix(line, "@@") {
			header := line
			oldLine, newLine = parseUnifiedHunkStart(line)
			hunks = append(hunks, gitDiffHunk{Header: header})
			current = &hunks[len(hunks)-1]
			continue
		}
		if current == nil {
			continue
		}
		if strings.HasPrefix(line, "\\ No newline") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			current.Lines = append(current.Lines, gitDiffLine{Kind: "add", NewLine: intPtr(newLine), Content: strings.TrimPrefix(line, "+")})
			newLine += 1
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			current.Lines = append(current.Lines, gitDiffLine{Kind: "delete", OldLine: intPtr(oldLine), Content: strings.TrimPrefix(line, "-")})
			oldLine += 1
		default:
			content := line
			if strings.HasPrefix(content, " ") {
				content = content[1:]
			}
			current.Lines = append(current.Lines, gitDiffLine{Kind: "context", OldLine: intPtr(oldLine), NewLine: intPtr(newLine), Content: content})
			oldLine += 1
			newLine += 1
		}
	}
	return hunks
}

func parseUnifiedHunkStart(header string) (int, int) {
	parts := strings.Fields(header)
	if len(parts) < 3 {
		return 1, 1
	}
	return parseUnifiedLineStart(parts[1]), parseUnifiedLineStart(parts[2])
}

func parseUnifiedLineStart(value string) int {
	value = strings.TrimPrefix(value, "-")
	value = strings.TrimPrefix(value, "+")
	value = strings.Split(value, ",")[0]
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 1
	}
	return parsed
}

func intPtr(value int) *int {
	return &value
}

func runGitRaw(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
