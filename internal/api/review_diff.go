package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
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
	})
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
	case orchestrator.RunStatusFailed:
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
	Path      string `json:"path"`
	Folder    string `json:"folder"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Binary    bool   `json:"binary"`
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
	files, err := collectWorkspaceGitDiffFiles(request.Context(), ws, pathFilter, folderFilter, runID, stepID, snapshot.Service)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "git_diff_failed", err.Error())
		return
	}
	folders := summarizeGitDiffFolders(files)
	selectedPath := pathFilter
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
		"workspace":     ws.Path,
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

func collectWorkspaceGitDiffFiles(ctx context.Context, ws workspace.Root, pathFilter string, folderFilter string, runID string, stepID string, service *orchestrator.Service) ([]gitDiffFile, error) {
	statusOutput, err := runGitRaw(ctx, ws.Path, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	files := parseGitStatusFiles(statusOutput)
	allowedByRun := artifactPathFilterForRun(runID, stepID, service)
	filtered := make([]gitDiffFile, 0, len(files))
	for _, file := range files {
		if pathFilter != "" && file.Path != pathFilter {
			continue
		}
		if folderFilter != "" && file.Path != folderFilter && !strings.HasPrefix(file.Path, strings.TrimRight(folderFilter, "/")+"/") {
			continue
		}
		if allowedByRun != nil {
			if _, ok := allowedByRun[file.Path]; !ok {
				continue
			}
		}
		enriched := enrichGitDiffFile(ctx, ws, file)
		filtered = append(filtered, enriched)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Path < filtered[j].Path })
	return filtered, nil
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
		if strings.Contains(code, "R") || strings.Contains(code, "C") {
			if index+1 < len(parts) && strings.TrimSpace(parts[index+1]) != "" {
				rel = strings.TrimSpace(parts[index+1])
				index += 1
			}
		}
		if rel == "" {
			continue
		}
		files = append(files, gitDiffFile{
			Path:   filepath.ToSlash(rel),
			Folder: gitDiffFolder(filepath.ToSlash(rel)),
			Status: gitDiffStatusLabel(code),
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

func gitDiffFolder(rel string) string {
	parts := strings.Split(strings.Trim(rel, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "root"
	}
	return parts[0]
}

func enrichGitDiffFile(ctx context.Context, ws workspace.Root, file gitDiffFile) gitDiffFile {
	file.Binary = workspaceFileIsBinary(ws, file.Path)
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
