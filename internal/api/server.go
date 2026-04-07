package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type Server struct {
	mu        sync.RWMutex
	workspace workspace.Root
	service   *orchestrator.Service
}

func NewServer(ws workspace.Root, service *orchestrator.Service) *Server {
	return &Server{
		workspace: ws,
		service:   service,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/workspace/validate", s.handleWorkspaceValidate)
	mux.HandleFunc("/api/workspace/manifest", s.handleWorkspaceManifest)
	mux.HandleFunc("/api/artifacts", s.handleArtifacts)
	mux.HandleFunc("/api/artifacts/write", s.handleArtifactsWrite)
	mux.HandleFunc("/api/git/commit", s.handleGitCommit)
	mux.HandleFunc("/api/git/proposal-branch", s.handleGitProposalBranch)
	mux.HandleFunc("/api/pipeline/init", s.handlePipelineInit)
	mux.HandleFunc("/api/pipeline/refresh", s.handlePipelineRefresh)
	mux.HandleFunc("/api/pipeline/runs", s.handlePipelineRuns)
	mux.HandleFunc("/api/pipeline/runs/", s.handlePipelineRuns)

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			mux.ServeHTTP(writer, request)
			return
		}
		serveEmbeddedUI(writer, request)
	})
}

func (s *Server) Serve(ctx context.Context, address string) error {
	httpServer := &http.Server{
		Addr:    address,
		Handler: s.Handler(),
	}

	go func() {
		<-ctx.Done()
		_ = httpServer.Shutdown(context.Background())
	}()

	err := httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) getWorkspace() workspace.Root {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workspace
}

func (s *Server) setWorkspace(ws workspace.Root) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspace = ws
}

func (s *Server) handleHealth(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleWorkspaceValidate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}

	ws := s.getWorkspace()
	report := ws.Validate(request.Context(), workspace.ValidateOptions{
		ResolveRepos: true,
		FetchGit:     false,
		VerifyRefs:   true,
	})
	if !report.OK {
		writeJSON(writer, http.StatusBadRequest, map[string]any{
			"ok":        false,
			"workspace": ws.Path,
			"error": map[string]string{
				"code":    "workspace_validation_failed",
				"message": "workspace validation failed",
			},
			"errors":         report.Errors,
			"warnings":       report.Warnings,
			"resolved_repos": report.ResolvedRepos,
		})
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":             true,
		"workspace":      ws.Path,
		"warnings":       report.Warnings,
		"resolved_repos": report.ResolvedRepos,
	})
}

func (s *Server) handleWorkspaceManifest(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		ws := s.getWorkspace()
		content, err := ws.ReadFile(workspace.ManifestFileName)
		if err != nil {
			writeError(writer, http.StatusNotFound, "manifest_read_failed", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"content": string(content),
		})
	case http.MethodPut:
		var payload struct {
			Content string `json:"content"`
		}
		if err := decodeStrictJSON(request, &payload); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		if strings.TrimSpace(payload.Content) == "" {
			writeError(writer, http.StatusBadRequest, "manifest_empty", "manifest content is required")
			return
		}
		if _, err := workspace.ParseManifest([]byte(payload.Content)); err != nil {
			writeError(writer, http.StatusBadRequest, "manifest_invalid", err.Error())
			return
		}

		ws := s.getWorkspace()
		if err := ws.WriteFile(workspace.ManifestFileName, []byte(payload.Content)); err != nil {
			writeError(writer, http.StatusInternalServerError, "manifest_write_failed", err.Error())
			return
		}

		reopened, err := workspace.Open(ws.Path)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "manifest_reopen_failed", err.Error())
			return
		}
		s.setWorkspace(reopened)
		writeJSON(writer, http.StatusOK, map[string]any{"ok": true})
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPut)
	}
}

func (s *Server) handleArtifactsWrite(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	path := strings.TrimSpace(payload.Path)
	if path == "" {
		writeError(writer, http.StatusBadRequest, "artifact_path_required", "path is required")
		return
	}
	if !strings.HasPrefix(path, "charter/") && !strings.HasPrefix(path, "skills/") {
		writeError(writer, http.StatusBadRequest, "artifact_path_forbidden", "only charter/* and skills/* are editable through this endpoint")
		return
	}
	ws := s.getWorkspace()
	if err := ws.WriteFile(path, []byte(payload.Content)); err != nil {
		writeError(writer, http.StatusBadRequest, "artifact_write_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGitCommit(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload struct {
		Message string `json:"message"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	message := strings.TrimSpace(payload.Message)
	if message == "" {
		message = "chore: update ACP workspace artifacts"
	}
	ws := s.getWorkspace()
	if _, err := runGit(request.Context(), ws.Path, "add", "-A"); err != nil {
		writeError(writer, http.StatusBadRequest, "git_add_failed", err.Error())
		return
	}
	output, err := runGit(request.Context(), ws.Path, "commit", "-m", message)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "nothing to commit") {
			writeJSON(writer, http.StatusOK, map[string]any{
				"ok":      true,
				"status":  "no_changes",
				"message": "nothing to commit",
			})
			return
		}
		writeError(writer, http.StatusBadRequest, "git_commit_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":     true,
		"status": "committed",
		"output": output,
	})
}

func (s *Server) handleGitProposalBranch(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload struct {
		Name string `json:"name"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	branch := sanitizeBranchName(payload.Name)
	if branch == "" {
		branch = "proposal/" + time.Now().UTC().Format("20060102-150405")
	}

	ws := s.getWorkspace()
	if _, err := runGit(request.Context(), ws.Path, "checkout", "-b", branch); err != nil {
		if _, fallbackErr := runGit(request.Context(), ws.Path, "checkout", branch); fallbackErr != nil {
			writeError(writer, http.StatusBadRequest, "git_branch_failed", err.Error())
			return
		}
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":     true,
		"branch": branch,
	})
}

func (s *Server) handleArtifacts(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}

	relPath := strings.TrimSpace(request.URL.Query().Get("path"))
	if relPath == "" {
		writeError(writer, http.StatusBadRequest, "bad_request", "path query parameter is required")
		return
	}
	ws := s.getWorkspace()
	content, err := ws.ReadFile(relPath)
	if err != nil {
		if errors.Is(err, workspace.ErrPathTraversal) || errors.Is(err, workspace.ErrPathAbsolute) {
			writeError(writer, http.StatusBadRequest, "path_invalid", err.Error())
			return
		}
		writeError(writer, http.StatusNotFound, "artifact_not_found", err.Error())
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(relPath))
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	writer.Header().Set("Content-Type", contentType)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(content)
}

func (s *Server) handlePipelineInit(writer http.ResponseWriter, request *http.Request) {
	s.handlePipelineStart(writer, request, orchestrator.PipelineInit, "ui")
}

func (s *Server) handlePipelineRefresh(writer http.ResponseWriter, request *http.Request) {
	s.handlePipelineStart(writer, request, orchestrator.PipelineRefresh, "manual")
}

type pipelineRequest struct {
	Commit               bool   `json:"commit"`
	CreateProposalBranch bool   `json:"create_proposal_branch"`
	Trigger              string `json:"trigger"`
}

var supportedTriggers = map[string]struct{}{
	"ui":         {},
	"manual":     {},
	"hook":       {},
	"automation": {},
}

func (s *Server) handlePipelineStart(writer http.ResponseWriter, request *http.Request, pipeline orchestrator.Pipeline, defaultTrigger string) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}

	body := pipelineRequest{Trigger: defaultTrigger}
	if request.Body != nil {
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil && !errors.Is(err, context.Canceled) {
			if !errors.Is(err, io.EOF) {
				writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
				return
			}
		}
		var trailing any
		if err := decoder.Decode(&trailing); err == nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		} else if !errors.Is(err, io.EOF) {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
	}
	if strings.TrimSpace(body.Trigger) == "" {
		body.Trigger = defaultTrigger
	}
	if _, ok := supportedTriggers[body.Trigger]; !ok {
		writeError(writer, http.StatusBadRequest, "trigger_unsupported", "trigger must be one of: ui, manual, hook, automation")
		return
	}
	if body.Commit || body.CreateProposalBranch {
		writeError(writer, http.StatusNotImplemented, "not_supported", "commit/create_proposal_branch is not supported in this slice")
		return
	}

	runID, err := s.service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace:      s.getWorkspace(),
		Pipeline:       pipeline,
		NonInteractive: true,
	})
	if err != nil {
		if statusCode, code, message, ok := mapTypedRunnerAPIError(err); ok {
			writeError(writer, statusCode, code, message)
			return
		}
		writeError(writer, http.StatusBadRequest, "run_start_failed", err.Error())
		return
	}

	writeJSON(writer, http.StatusAccepted, map[string]any{
		"run_id": runID,
		"status": "started",
	})
}

func (s *Server) handlePipelineRuns(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}

	if request.URL.Path == "/api/pipeline/runs" || request.URL.Path == "/api/pipeline/runs/" {
		const (
			defaultLimit = 50
			maxLimit     = 500
		)
		limit := defaultLimit
		rawLimit := strings.TrimSpace(request.URL.Query().Get("limit"))
		if rawLimit != "" {
			parsedLimit, err := strconv.Atoi(rawLimit)
			if err != nil || parsedLimit <= 0 {
				writeError(writer, http.StatusBadRequest, "invalid_limit", "limit must be a positive integer")
				return
			}
			if parsedLimit > maxLimit {
				parsedLimit = maxLimit
			}
			limit = parsedLimit
		}

		runs := s.service.ListRuns(limit)
		items := make([]map[string]any, 0, len(runs))
		for _, runInfo := range runs {
			items = append(items, formatRunInfoPayload(runInfo))
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"items": items,
		})
		return
	}

	rest := strings.TrimPrefix(request.URL.Path, "/api/pipeline/runs/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(writer, http.StatusNotFound, "run_id_required", "run id is required")
		return
	}
	runID := parts[0]

	if len(parts) == 1 {
		runInfo, ok := s.service.GetRun(runID)
		if !ok {
			writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
			return
		}
		writeJSON(writer, http.StatusOK, formatRunInfoPayload(runInfo))
		return
	}

	if len(parts) == 2 && parts[1] == "artifacts" {
		artifacts, ok := s.service.GetRunArtifacts(runID)
		if !ok {
			writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"run_id":    runID,
			"artifacts": artifacts,
		})
		return
	}

	if len(parts) == 2 && parts[1] == "logs" {
		cursor := 0
		rawCursor := strings.TrimSpace(request.URL.Query().Get("cursor"))
		if rawCursor != "" {
			parsedCursor, err := strconv.Atoi(rawCursor)
			if err != nil || parsedCursor < 0 {
				writeError(writer, http.StatusBadRequest, "invalid_cursor", "cursor must be a non-negative integer")
				return
			}
			cursor = parsedCursor
		}

		limit := 200
		rawLimit := strings.TrimSpace(request.URL.Query().Get("limit"))
		if rawLimit != "" {
			parsedLimit, err := strconv.Atoi(rawLimit)
			if err != nil || parsedLimit <= 0 {
				writeError(writer, http.StatusBadRequest, "invalid_limit", "limit must be a positive integer")
				return
			}
			if parsedLimit > 500 {
				parsedLimit = 500
			}
			limit = parsedLimit
		}

		page, ok, err := s.service.GetRunLogs(runID, cursor, limit)
		if !ok {
			writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "run_logs_unavailable", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"run_id":      page.RunID,
			"items":       page.Items,
			"next_cursor": page.NextCursor,
			"eof":         page.EOF,
		})
		return
	}

	writeError(writer, http.StatusNotFound, "endpoint_not_found", "endpoint not found")
}

func runGit(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func sanitizeBranchName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, " ", "-")
	allowed := make([]rune, 0, len(name))
	for _, r := range name {
		isAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isAlpha || isDigit || r == '/' || r == '-' || r == '_' {
			allowed = append(allowed, r)
			continue
		}
		allowed = append(allowed, '-')
	}
	branch := strings.Trim(strings.ReplaceAll(string(allowed), "//", "/"), "/-")
	if branch == "" {
		return ""
	}
	if !strings.HasPrefix(branch, "proposal/") {
		return "proposal/" + branch
	}
	return branch
}

func writeMethodNotAllowed(writer http.ResponseWriter, allowedMethod string) {
	writer.Header().Set("Allow", allowedMethod)
	writeError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}

func decodeStrictJSON(request *http.Request, payload any) error {
	if request == nil || request.Body == nil {
		return io.EOF
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(payload); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("unexpected trailing JSON tokens")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func writeError(writer http.ResponseWriter, statusCode int, code string, message string) {
	writeJSON(writer, statusCode, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}

func formatOptionalTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

func formatOptionalString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func formatRunInfoPayload(runInfo orchestrator.RunInfo) map[string]any {
	return map[string]any{
		"run_id":       runInfo.RunID,
		"pipeline":     runInfo.Pipeline,
		"status":       runInfo.Status,
		"started_at":   runInfo.StartedAt.UTC().Format(time.RFC3339),
		"finished_at":  formatOptionalTime(runInfo.FinishedAt),
		"current_step": runInfo.CurrentStep,
		"warnings":     runInfo.Warnings,
		"error_code":   formatOptionalString(runInfo.ErrorCode),
		"error":        formatOptionalString(runInfo.Error),
	}
}

func mapTypedRunnerAPIError(err error) (statusCode int, code string, message string, ok bool) {
	runnerCode, runnerMessage, classified := claudecode.ClassifyError(err)
	if !classified {
		return 0, "", "", false
	}
	switch runnerCode {
	case string(claudecode.ErrorCodeRunnerUnavailable):
		return http.StatusServiceUnavailable, runnerCode, runnerMessage, true
	case string(claudecode.ErrorCodeRunnerParseFailed):
		// Parse failures are surfaced as run-level failures (`error_code`) after async start.
		return 0, "", "", false
	default:
		return http.StatusBadRequest, runnerCode, runnerMessage, true
	}
}
