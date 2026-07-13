package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GrinRus/ProvenArch/internal/doctor"
	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	"github.com/GrinRus/ProvenArch/internal/qa"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimeprofile"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type Server struct {
	mu                sync.RWMutex
	workspace         workspace.Root
	workspacePath     string
	workspaceSelected bool
	runtimeSelected   bool
	launcherMode      bool
	service           *orchestrator.Service
	runtimeConfig     ServerRuntimeConfig
	serviceFactory    ServiceFactory
}

type ServerRuntimeConfig struct {
	Mode               string
	Provider           acpruntime.Provider
	ProviderSource     acpruntime.ProviderSource
	ExecutionOverrides acpruntime.ExecutionOverrides
	RunLogsTTL         time.Duration
	RunLogsMaxRuns     int
	Build              BuildInfo
}

type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Built   string `json:"built"`
}

type ServiceFactory func(workspace.Root, ServerRuntimeConfig) *orchestrator.Service

const defaultServerShutdownTimeout = 10 * time.Second

func NewServer(ws workspace.Root, service *orchestrator.Service) *Server {
	return NewServerWithRuntime(ws, service, ServerRuntimeConfig{
		Mode:           acpruntime.RuntimeModeFake,
		Provider:       acpruntime.ProviderClaudeCode,
		ProviderSource: acpruntime.ProviderSourceDefault,
	})
}

func NewServerWithRuntime(ws workspace.Root, service *orchestrator.Service, runtimeConfig ServerRuntimeConfig) *Server {
	if strings.TrimSpace(runtimeConfig.Mode) == "" {
		runtimeConfig.Mode = acpruntime.RuntimeModeFake
	}
	if runtimeConfig.Provider == "" {
		runtimeConfig.Provider = acpruntime.ProviderClaudeCode
	}
	if runtimeConfig.ProviderSource == "" {
		runtimeConfig.ProviderSource = acpruntime.ProviderSourceDefault
	}
	return &Server{
		workspace:         ws,
		workspacePath:     ws.Path,
		workspaceSelected: true,
		runtimeSelected:   true,
		runtimeConfig:     runtimeConfig,
		service:           service,
	}
}

func NewLauncherServer(runtimeConfig ServerRuntimeConfig, factory ServiceFactory) *Server {
	if strings.TrimSpace(runtimeConfig.Mode) == "" {
		runtimeConfig.Mode = acpruntime.RuntimeModeFake
	}
	if runtimeConfig.Provider == "" {
		runtimeConfig.Provider = acpruntime.ProviderClaudeCode
	}
	if runtimeConfig.ProviderSource == "" {
		runtimeConfig.ProviderSource = acpruntime.ProviderSourceDefault
	}
	return &Server{
		launcherMode:   true,
		runtimeConfig:  runtimeConfig,
		serviceFactory: factory,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/system/version", s.handleSystemVersion)
	mux.HandleFunc("/api/onboarding/status", s.handleOnboardingStatus)
	mux.HandleFunc("/api/onboarding/workspace", s.handleOnboardingWorkspace)
	mux.HandleFunc("/api/onboarding/runtime", s.handleOnboardingRuntime)
	mux.HandleFunc("/api/onboarding/path-suggestions", s.handleOnboardingPathSuggestions)
	mux.HandleFunc("/api/onboarding/recent-workspaces/forget", s.handleOnboardingRecentWorkspaceForget)
	mux.HandleFunc("/api/system/info", s.handleSystemInfo)
	mux.HandleFunc("/api/system/doctor", s.handleSystemDoctor)
	mux.HandleFunc("/api/workspace/validate", s.handleWorkspaceValidate)
	mux.HandleFunc("/api/workspace/bundle", s.handleWorkspaceBundle)
	mux.HandleFunc("/api/workspace/manifest", s.handleWorkspaceManifest)
	mux.HandleFunc("/api/runtime/timeouts", s.handleRuntimeTimeouts)
	mux.HandleFunc("/api/runtime/execution", s.handleRuntimeExecution)
	mux.HandleFunc("/api/runtime/permissions", s.handleRuntimePermissions)
	mux.HandleFunc("/api/runtime/profile", s.handleRuntimeProfile)
	mux.HandleFunc("/api/artifacts", s.handleArtifacts)
	mux.HandleFunc("/api/artifacts/write", s.handleArtifactsWrite)
	mux.HandleFunc("/api/git/diff", s.handleGitDiff)
	mux.HandleFunc("/api/git/commit", s.handleGitCommit)
	mux.HandleFunc("/api/git/proposal-branch", s.handleGitProposalBranch)
	mux.HandleFunc("/api/qa/ask", s.handleQAAsk)
	mux.HandleFunc("/api/qa/runs", s.handleQARuns)
	mux.HandleFunc("/api/qa/runs/", s.handleQARuns)
	mux.HandleFunc("/api/pipeline/init", s.handlePipelineInit)
	mux.HandleFunc("/api/pipeline/refresh", s.handlePipelineRefresh)
	mux.HandleFunc("/api/pipeline/runs", s.handlePipelineRuns)
	mux.HandleFunc("/api/pipeline/runs/", s.handlePipelineRuns)

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			if s.shouldBlockAPIRequest(request.URL.Path) {
				writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select or create an ACP workspace before using this API")
				return
			}
			mux.ServeHTTP(writer, request)
			return
		}
		serveEmbeddedUI(writer, request)
	})
}

func (s *Server) Serve(ctx context.Context, address string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	httpServer := &http.Server{
		Addr:    address,
		Handler: s.Handler(),
	}

	shutdownErrCh := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultServerShutdownTimeout)
		defer cancel()
		shutdownErrCh <- firstError(httpServer.Shutdown(shutdownCtx), s.Shutdown(shutdownCtx))
	}()

	err := httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		if ctx.Err() != nil {
			return <-shutdownErrCh
		}
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	service := s.getService()
	if service == nil {
		return nil
	}
	return service.Shutdown(ctx)
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) getWorkspace() workspace.Root {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workspace
}

func (s *Server) getWorkspacePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workspacePath
}

func (s *Server) hasReadyWorkspace() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workspaceSelected && s.workspace.Path != "" && s.service != nil
}

func (s *Server) getService() *orchestrator.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.service
}

func (s *Server) getRuntimeConfig() ServerRuntimeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtimeConfig
}

func (s *Server) isRuntimeSelected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtimeSelected
}

func (s *Server) setWorkspace(ws workspace.Root) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspace = ws
	s.workspacePath = ws.Path
	s.workspaceSelected = true
}

func (s *Server) setDraftWorkspace(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspace = workspace.Root{}
	s.workspacePath = path
	s.workspaceSelected = true
	s.service = nil
}

func (s *Server) attachWorkspace(ws workspace.Root) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspace = ws
	s.workspacePath = ws.Path
	s.workspaceSelected = true
	if s.serviceFactory != nil {
		s.service = s.serviceFactory(ws, s.runtimeConfig)
	}
}

func (s *Server) setRuntimeConfig(config ServerRuntimeConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(config.Mode) == "" {
		config.Mode = acpruntime.RuntimeModeFake
	}
	if config.Provider == "" {
		config.Provider = acpruntime.ProviderClaudeCode
	}
	if config.ProviderSource == "" {
		config.ProviderSource = acpruntime.ProviderSourceDefault
	}
	if strings.TrimSpace(config.Build.Version) == "" {
		config.Build = s.runtimeConfig.Build
	}
	config.ExecutionOverrides = s.runtimeConfig.ExecutionOverrides
	config.RunLogsTTL = s.runtimeConfig.RunLogsTTL
	config.RunLogsMaxRuns = s.runtimeConfig.RunLogsMaxRuns
	s.runtimeConfig = config
	s.runtimeSelected = true
	if s.workspaceSelected && s.workspace.Path != "" && s.serviceFactory != nil {
		s.service = s.serviceFactory(s.workspace, s.runtimeConfig)
	}
}

func (s *Server) shouldBlockAPIRequest(apiPath string) bool {
	if apiPath == "/api/health" || apiPath == "/api/system/version" || apiPath == "/api/system/info" || strings.HasPrefix(apiPath, "/api/onboarding/") {
		return false
	}
	s.mu.RLock()
	selected := s.workspaceSelected
	ready := selected && s.workspace.Path != "" && s.service != nil
	pathSelected := selected && strings.TrimSpace(s.workspacePath) != ""
	s.mu.RUnlock()
	if ready {
		return false
	}
	if pathSelected && (apiPath == "/api/workspace/manifest" || apiPath == "/api/system/doctor") {
		return false
	}
	return true
}

func (s *Server) handleHealth(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSystemVersion(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	writeJSON(writer, http.StatusOK, CurrentBuildInfo())
}

func (s *Server) handleSystemInfo(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	s.mu.RLock()
	info := s.runtimeConfig.Build
	s.mu.RUnlock()
	writeJSON(writer, http.StatusOK, map[string]string{
		"version": firstNonEmpty(info.Version, "dev"),
		"commit":  firstNonEmpty(info.Commit, "none"),
		"built":   firstNonEmpty(info.Built, "unknown"),
	})
}

func (s *Server) handleSystemDoctor(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}

	query := request.URL.Query()
	if strings.TrimSpace(query.Get("repo_path")) != "" && strings.TrimSpace(query.Get("repo_git_url")) != "" {
		writeError(writer, http.StatusBadRequest, "invalid_doctor_request", "set at most one of repo_path or repo_git_url")
		return
	}

	workspacePath := s.getWorkspacePath()
	report, err := doctor.Run(request.Context(), doctor.Options{
		WorkspacePath:       workspacePath,
		RepoPath:            query.Get("repo_path"),
		RepoGitURL:          query.Get("repo_git_url"),
		RuntimeMode:         query.Get("runtime"),
		RuntimeProvider:     query.Get("runtime_provider"),
		CheckPort:           false,
		EmbeddedUIAvailable: EmbeddedUIAvailable(),
	})
	if err != nil {
		writeError(writer, http.StatusBadRequest, "doctor_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, report)
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

func (s *Server) handleWorkspaceBundle(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}

	ws := s.getWorkspace()
	manifest, diagnostics := ws.EffectiveBaselineBundleManifest()
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":        true,
		"workspace": ws.Path,
		"manifest":  manifest,
		"warnings":  diagnostics,
	})
}

func (s *Server) handleWorkspaceManifest(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		workspacePath := s.getWorkspacePath()
		if strings.TrimSpace(workspacePath) == "" {
			writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select or create an ACP workspace before reading workspace.yaml")
			return
		}
		content, err := os.ReadFile(filepath.Join(workspacePath, workspace.ManifestFileName))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				writeJSON(writer, http.StatusOK, map[string]any{"content": ""})
				return
			}
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

		workspacePath := s.getWorkspacePath()
		if strings.TrimSpace(workspacePath) == "" {
			writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select or create an ACP workspace before saving workspace.yaml")
			return
		}
		if err := os.WriteFile(filepath.Join(workspacePath, workspace.ManifestFileName), []byte(payload.Content), 0o644); err != nil {
			writeError(writer, http.StatusInternalServerError, "manifest_write_failed", err.Error())
			return
		}

		reopened, err := workspace.Open(workspacePath)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "manifest_reopen_failed", err.Error())
			return
		}
		if err := reopened.EnsureLayout(); err != nil {
			writeError(writer, http.StatusInternalServerError, "workspace_layout_failed", err.Error())
			return
		}
		if err := reopened.EnsureBaselineBundle(); err != nil {
			writeError(writer, http.StatusInternalServerError, "workspace_baseline_failed", err.Error())
			return
		}
		if s.hasReadyWorkspace() {
			s.setWorkspace(reopened)
		} else {
			s.attachWorkspace(reopened)
			if service := s.getService(); service != nil {
				service.ReconcileStaleRunsAfterRestart()
			}
		}
		writeJSON(writer, http.StatusOK, map[string]any{"ok": true})
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPut)
	}
}

func (s *Server) handleRuntimeTimeouts(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		ws := s.getWorkspace()
		resolved := acpruntime.ResolveTimeouts(ws.Manifest)
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"persisted": resolved.Persisted,
			"effective": resolved.Effective,
			"source":    resolved.Source,
		})
	case http.MethodPut:
		var payload struct {
			Timeouts workspace.RuntimeTimeoutsConfig `json:"timeouts"`
		}
		if err := decodeStrictJSON(request, &payload); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		if payload.Timeouts.IsZero() {
			writeError(writer, http.StatusBadRequest, "runtime_timeouts_empty", "timeouts payload must include at least one field")
			return
		}
		if err := runtimeprofile.ValidateRuntimeTimeoutPatch(payload.Timeouts); err != nil {
			writeError(writer, http.StatusBadRequest, "runtime_timeouts_invalid", err.Error())
			return
		}

		ws := s.getWorkspace()
		reopened, err := (runtimeprofile.RuntimeProfilePatchService{}).ApplyTimeouts(ws, payload.Timeouts)
		if err != nil {
			if typed := (runtimeprofile.PatchError{}); errors.As(err, &typed) {
				writeError(writer, http.StatusInternalServerError, typed.Code, typed.Error())
				return
			}
			writeError(writer, http.StatusInternalServerError, "runtime_timeouts_write_failed", err.Error())
			return
		}
		s.setWorkspace(reopened)
		resolved := acpruntime.ResolveTimeouts(reopened.Manifest)
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"persisted": resolved.Persisted,
			"effective": resolved.Effective,
			"source":    resolved.Source,
		})
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPut)
	}
}

func (s *Server) handleRuntimeExecution(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		ws := s.getWorkspace()
		resolved := s.service.ResolveExecutionProfile(ws.Manifest)
		stepProviders, err := s.service.ResolveStepProviderProfile(ws.Manifest)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "runtime_step_provider_resolution_failed", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"persisted": runtimeExecutionPersistedPayload(resolved.Persisted, stepProviders.Persisted),
			"effective": runtimeExecutionEffectivePayload(resolved.Effective, stepProviders.Effective),
			"source":    runtimeExecutionSourcePayload(resolved.Source, stepProviders.Source),
		})
	case http.MethodPut:
		var payload struct {
			Execution runtimeprofile.RuntimeExecutionPatch `json:"execution"`
		}
		if err := decodeStrictJSON(request, &payload); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		if payload.Execution.IsZero() {
			writeError(writer, http.StatusBadRequest, "runtime_execution_empty", "execution payload must include at least one field")
			return
		}
		if err := runtimeprofile.ValidateRuntimeExecutionPatch(payload.Execution); err != nil {
			writeError(writer, http.StatusBadRequest, "runtime_execution_invalid", err.Error())
			return
		}

		ws := s.getWorkspace()
		reopened, err := (runtimeprofile.RuntimeProfilePatchService{}).ApplyExecution(ws, payload.Execution)
		if err != nil {
			if typed := (runtimeprofile.PatchError{}); errors.As(err, &typed) {
				writeError(writer, http.StatusInternalServerError, typed.Code, typed.Error())
				return
			}
			writeError(writer, http.StatusInternalServerError, "runtime_execution_write_failed", err.Error())
			return
		}
		s.setWorkspace(reopened)
		resolved := s.service.ResolveExecutionProfile(reopened.Manifest)
		stepProviders, err := s.service.ResolveStepProviderProfile(reopened.Manifest)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "runtime_step_provider_resolution_failed", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"persisted": runtimeExecutionPersistedPayload(resolved.Persisted, stepProviders.Persisted),
			"effective": runtimeExecutionEffectivePayload(resolved.Effective, stepProviders.Effective),
			"source":    runtimeExecutionSourcePayload(resolved.Source, stepProviders.Source),
		})
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPut)
	}
}

func (s *Server) handleRuntimePermissions(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		ws := s.getWorkspace()
		resolved := acpruntime.ResolvePermissions(ws.Manifest)
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"persisted": runtimePermissionsPersistedPayload(resolved.Persisted),
			"effective": resolved.Effective,
			"source":    resolved.Source,
		})
	case http.MethodPut:
		var payload struct {
			Permissions runtimeprofile.RuntimePermissionsPatch `json:"permissions"`
		}
		if err := decodeStrictJSON(request, &payload); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		if payload.Permissions.IsZero() {
			writeError(writer, http.StatusBadRequest, "runtime_permissions_empty", "permissions payload must include at least one field")
			return
		}
		if err := runtimeprofile.ValidateRuntimePermissionsPatch(payload.Permissions); err != nil {
			writeError(writer, http.StatusBadRequest, "runtime_permissions_invalid", err.Error())
			return
		}

		ws := s.getWorkspace()
		reopened, err := (runtimeprofile.RuntimeProfilePatchService{}).ApplyPermissions(ws, payload.Permissions)
		if err != nil {
			if typed := (runtimeprofile.PatchError{}); errors.As(err, &typed) {
				writeError(writer, http.StatusInternalServerError, typed.Code, typed.Error())
				return
			}
			writeError(writer, http.StatusInternalServerError, "runtime_permissions_write_failed", err.Error())
			return
		}
		s.setWorkspace(reopened)
		resolved := acpruntime.ResolvePermissions(reopened.Manifest)
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"persisted": runtimePermissionsPersistedPayload(resolved.Persisted),
			"effective": resolved.Effective,
			"source":    resolved.Source,
		})
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPut)
	}
}

func (s *Server) handleRuntimeProfile(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	ws := s.getWorkspace()
	timeouts := acpruntime.ResolveTimeouts(ws.Manifest)
	execution := s.service.ResolveExecutionProfile(ws.Manifest)
	permissions := acpruntime.ResolvePermissions(ws.Manifest)
	stepProviders, err := s.service.ResolveStepProviderProfile(ws.Manifest)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "runtime_step_provider_resolution_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok": true,
		"timeouts": map[string]any{
			"persisted": timeouts.Persisted,
			"effective": timeouts.Effective,
			"source":    timeouts.Source,
		},
		"execution": map[string]any{
			"persisted": runtimeExecutionPersistedPayload(execution.Persisted, stepProviders.Persisted),
			"effective": runtimeExecutionEffectivePayload(execution.Effective, stepProviders.Effective),
			"source":    runtimeExecutionSourcePayload(execution.Source, stepProviders.Source),
		},
		"permissions": map[string]any{
			"persisted": runtimePermissionsPersistedPayload(permissions.Persisted),
			"effective": permissions.Effective,
			"source":    permissions.Source,
		},
		"step_providers": map[string]any{
			"persisted": runtimeStepProvidersPersistedPayload(stepProviders.Persisted),
			"effective": runtimeStepProvidersEffectivePayload(stepProviders.Effective),
			"source":    runtimeStepProvidersSourcePayload(stepProviders.Source),
		},
	})
}

func runtimeExecutionPersistedPayload(persisted workspace.RuntimeExecutionConfig, steps workspace.RuntimeStepsConfig) map[string]any {
	payload := map[string]any{}
	if value := strings.TrimSpace(persisted.Strategy); value != "" {
		payload["strategy"] = value
	}
	if persisted.MaxParallel != nil {
		payload["max_parallel_tasks"] = *persisted.MaxParallel
	}
	if value := strings.TrimSpace(persisted.FailurePolicy); value != "" {
		payload["failure_policy"] = value
	}
	if persisted.ShardDiscovery != nil {
		if value := strings.TrimSpace(persisted.ShardDiscovery.Mode); value != "" {
			payload["shard_discovery_mode"] = value
		}
	}
	if runtimeSteps := runtimeStepProvidersPersistedPayload(steps); len(runtimeSteps) > 0 {
		payload["steps"] = runtimeSteps
	}
	return payload
}

func runtimeExecutionEffectivePayload(effective acpruntime.ExecutionValues, steps acpruntime.StepProviderValues) map[string]any {
	payload := map[string]any{
		"strategy":             effective.Strategy,
		"max_parallel_tasks":   effective.MaxParallel,
		"failure_policy":       effective.FailurePolicy,
		"shard_discovery_mode": effective.ShardMode,
	}
	if runtimeSteps := runtimeStepProvidersEffectivePayload(steps); len(runtimeSteps) > 0 {
		payload["steps"] = runtimeSteps
	}
	return payload
}

func runtimeExecutionSourcePayload(source acpruntime.ExecutionSources, steps acpruntime.StepProviderSources) map[string]any {
	payload := map[string]any{
		"strategy":             source.Strategy,
		"max_parallel_tasks":   source.MaxParallel,
		"failure_policy":       source.FailurePolicy,
		"shard_discovery_mode": source.ShardMode,
	}
	if runtimeSteps := runtimeStepProvidersSourcePayload(steps); len(runtimeSteps) > 0 {
		payload["steps"] = runtimeSteps
	}
	return payload
}

func runtimePermissionsPersistedPayload(persisted workspace.RuntimePermissionsConfig) map[string]any {
	payload := map[string]any{}
	if value := strings.TrimSpace(persisted.Mode); value != "" {
		payload["mode"] = value
	}
	if value := strings.TrimSpace(persisted.ApprovalChannel); value != "" {
		payload["approval_channel"] = value
	}
	return payload
}

func runtimeStepProvidersPersistedPayload(persisted workspace.RuntimeStepsConfig) map[string]any {
	payload := map[string]any{}
	appendStep := func(key string, step *workspace.RuntimeStepConfig) {
		if step == nil {
			return
		}
		if provider := strings.TrimSpace(step.Provider); provider != "" {
			payload[key] = provider
		}
	}
	appendStep("step0_constitution", persisted.Step0Constitution)
	appendStep("step1_collect", persisted.Step1Collect)
	appendStep("step2_as_is", persisted.Step2AsIs)
	appendStep("step3_findings", persisted.Step3Findings)
	appendStep("step4_proposals", persisted.Step4Proposals)
	appendStep("qa", persisted.QA)
	return payload
}

func runtimeStepProvidersEffectivePayload(effective acpruntime.StepProviderValues) map[string]any {
	payload := map[string]any{}
	for key, value := range effective.StringMap() {
		payload[key] = value
	}
	return payload
}

func runtimeStepProvidersSourcePayload(source acpruntime.StepProviderSources) map[string]any {
	payload := map[string]any{}
	for key, value := range source {
		payload[key] = value
	}
	return payload
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
	artifactPath := strings.TrimSpace(payload.Path)
	if artifactPath == "" {
		writeError(writer, http.StatusBadRequest, "artifact_path_required", "path is required")
		return
	}
	artifactPath, ok := normalizeEditableArtifactPath(artifactPath)
	if !ok {
		writeError(writer, http.StatusBadRequest, "artifact_path_forbidden", "only charter/* and skills/* are editable through this endpoint")
		return
	}
	ws := s.getWorkspace()
	if err := ws.WriteFile(artifactPath, []byte(payload.Content)); err != nil {
		writeError(writer, http.StatusBadRequest, "artifact_write_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"ok": true})
}

func normalizeEditableArtifactPath(rawPath string) (string, bool) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" || strings.ContainsRune(rawPath, 0) {
		return "", false
	}
	if filepath.IsAbs(rawPath) {
		return "", false
	}
	normalized := strings.ReplaceAll(rawPath, "\\", "/")
	if path.IsAbs(normalized) || isWindowsAbsolutePath(normalized) {
		return "", false
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return "", false
		}
	}
	clean := path.Clean(normalized)
	if clean == "." || clean == "charter" || clean == "skills" {
		return "", false
	}
	if !strings.HasPrefix(clean, "charter/") && !strings.HasPrefix(clean, "skills/") {
		return "", false
	}
	return clean, true
}

func isWindowsAbsolutePath(pathValue string) bool {
	if len(pathValue) >= 3 && pathValue[1] == ':' && pathValue[2] == '/' {
		drive := pathValue[0]
		return (drive >= 'A' && drive <= 'Z') || (drive >= 'a' && drive <= 'z')
	}
	return strings.HasPrefix(pathValue, "//")
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

func (s *Server) handleQAAsk(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload struct {
		Question string `json:"question"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	question := strings.TrimSpace(payload.Question)
	if question == "" {
		writeError(writer, http.StatusBadRequest, "question_required", "question is required")
		return
	}

	response, err := qa.NewService().Ask(request.Context(), s.getWorkspace(), question)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "qa_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleQARuns(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		s.handleQARunsGet(writer, request)
	case http.MethodPost:
		s.handleQARunsPost(writer, request)
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPost)
	}
}

func (s *Server) handleQARunsPost(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/api/qa/runs" && request.URL.Path != "/api/qa/runs/" {
		writeError(writer, http.StatusNotFound, "endpoint_not_found", "endpoint not found")
		return
	}
	var payload struct {
		Question string `json:"question"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	question := strings.TrimSpace(payload.Question)
	if question == "" {
		writeError(writer, http.StatusBadRequest, "question_required", "question is required")
		return
	}
	if !s.isRuntimeSelected() {
		writeError(writer, http.StatusBadRequest, "runtime_not_selected", "select a runner before starting Q&A")
		return
	}
	service := s.getService()
	if service == nil {
		writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select or create an ACP workspace before starting Q&A")
		return
	}
	runID, err := service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace:      s.getWorkspace(),
		Pipeline:       orchestrator.PipelineQA,
		NonInteractive: true,
		Question:       question,
	})
	if err != nil {
		if statusCode, code, message, ok := mapTypedRunnerAPIError(err); ok {
			writeError(writer, statusCode, code, message)
			return
		}
		writeError(writer, http.StatusBadRequest, "qa_run_start_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusAccepted, map[string]any{
		"run_id": runID,
		"status": string(orchestrator.RunStatusQueued),
	})
}

func (s *Server) handleQARunsGet(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/api/qa/runs" || request.URL.Path == "/api/qa/runs/" {
		s.handleQARunsList(writer, request)
		return
	}
	rest := strings.TrimPrefix(request.URL.Path, "/api/qa/runs/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 1 || strings.TrimSpace(parts[0]) == "" {
		writeError(writer, http.StatusNotFound, "qa_run_not_found", "qa run not found")
		return
	}
	s.handleQARunStatus(writer, strings.TrimSpace(parts[0]))
}

func (s *Server) handleQARunsList(writer http.ResponseWriter, request *http.Request) {
	const (
		defaultLimit = 20
		maxLimit     = 100
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

	runs := s.service.ListRuns(0)
	items := []map[string]any{}
	for _, runInfo := range runs {
		if runInfo.Pipeline != string(orchestrator.PipelineQA) {
			continue
		}
		items = append(items, formatQARunSummaryPayload(runInfo))
		if len(items) >= limit {
			break
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (s *Server) handleQARunStatus(writer http.ResponseWriter, runID string) {
	runInfo, ok := s.service.GetRun(runID)
	if !ok || runInfo.Pipeline != string(orchestrator.PipelineQA) {
		writeError(writer, http.StatusNotFound, "qa_run_not_found", "qa run not found")
		return
	}
	payload, err := s.formatQARunPayload(runInfo)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "qa_answer_unavailable", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, payload)
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

	if !s.isRuntimeSelected() {
		writeError(writer, http.StatusBadRequest, "runtime_not_selected", "select a runner before starting analysis")
		return
	}
	service := s.getService()
	if service == nil {
		writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select or create an ACP workspace before starting analysis")
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
				"code":    "workspace_not_ready",
				"message": "fix workspace, repository and runtime readiness blockers before starting analysis",
			},
			"errors":         report.Errors,
			"warnings":       report.Warnings,
			"resolved_repos": report.ResolvedRepos,
		})
		return
	}

	runID, err := service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace:      ws,
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
	switch request.Method {
	case http.MethodGet:
		s.handlePipelineRunsGet(writer, request)
	case http.MethodPost:
		s.handlePipelineRunsPost(writer, request)
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPost)
	}
}

func (s *Server) handlePipelineRunsGet(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/api/pipeline/runs" || request.URL.Path == "/api/pipeline/runs/" {
		s.handlePipelineRunsList(writer, request)
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
		s.handlePipelineRunStatus(writer, runID)
		return
	}

	if len(parts) == 2 && parts[1] == "artifacts" {
		s.handlePipelineRunArtifacts(writer, runID)
		return
	}

	if len(parts) == 2 && parts[1] == "review-summary" {
		s.handlePipelineRunReviewSummary(writer, runID)
		return
	}

	if len(parts) == 2 && parts[1] == "permissions" {
		s.handlePipelineRunPermissions(writer, runID)
		return
	}

	if len(parts) == 2 && parts[1] == "logs" {
		s.handlePipelineRunLogs(writer, request, runID)
		return
	}

	writeError(writer, http.StatusNotFound, "endpoint_not_found", "endpoint not found")
}

func (s *Server) handlePipelineRunsList(writer http.ResponseWriter, request *http.Request) {
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

	runs := s.service.ListRuns(0)
	items := make([]map[string]any, 0, len(runs))
	for _, runInfo := range runs {
		if runInfo.Pipeline == string(orchestrator.PipelineQA) {
			continue
		}
		items = append(items, formatRunInfoPayload(runInfo))
		if len(items) >= limit {
			break
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (s *Server) handlePipelineRunStatus(writer http.ResponseWriter, runID string) {
	runInfo, ok := s.service.GetRun(runID)
	if !ok {
		writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
		return
	}
	writeJSON(writer, http.StatusOK, formatRunInfoPayload(runInfo))
}

func (s *Server) handlePipelineRunArtifacts(writer http.ResponseWriter, runID string) {
	artifacts, ok := s.service.GetRunArtifacts(runID)
	if !ok {
		writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"run_id":    runID,
		"artifacts": artifacts,
	})
}

func (s *Server) handlePipelineRunPermissions(writer http.ResponseWriter, runID string) {
	permissions, ok := s.service.GetRunPermissions(runID)
	if !ok {
		writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"run_id":   runID,
		"requests": formatPermissionRequests(permissions),
	})
}

func (s *Server) handlePipelineRunLogs(writer http.ResponseWriter, request *http.Request, runID string) {
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
}

func (s *Server) handlePipelineRunsPost(writer http.ResponseWriter, request *http.Request) {
	rest := strings.TrimPrefix(request.URL.Path, "/api/pipeline/runs/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "cancel" {
		writeError(writer, http.StatusNotFound, "endpoint_not_found", "endpoint not found")
		return
	}
	runID := strings.TrimSpace(parts[0])

	if request.Body != nil {
		var payload struct{}
		if err := decodeStrictJSON(request, &payload); err != nil && !errors.Is(err, io.EOF) {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
	}

	if err := s.service.CancelRun(runID); err != nil {
		if errors.Is(err, orchestrator.ErrRunNotFound) {
			writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
			return
		}
		if errors.Is(err, orchestrator.ErrRunNotCancelable) {
			writeError(writer, http.StatusConflict, "run_not_cancelable", "run is already terminal")
			return
		}
		writeError(writer, http.StatusInternalServerError, "run_cancel_failed", err.Error())
		return
	}

	writeJSON(writer, http.StatusAccepted, map[string]any{
		"run_id": runID,
		"status": "cancel_requested",
	})
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
		"run_id":              runInfo.RunID,
		"pipeline":            runInfo.Pipeline,
		"status":              runInfo.Status,
		"started_at":          runInfo.StartedAt.UTC().Format(time.RFC3339),
		"finished_at":         formatOptionalTime(runInfo.FinishedAt),
		"question":            formatOptionalString(runInfo.Question),
		"current_step":        runInfo.CurrentStep,
		"step_providers":      runInfo.StepProviders,
		"warnings":            runInfo.Warnings,
		"pending_permissions": formatPermissionRequests(runInfo.PendingPermissions),
		"error_code":          formatOptionalString(runInfo.ErrorCode),
		"error":               formatOptionalString(runInfo.Error),
	}
}

func formatQARunSummaryPayload(runInfo orchestrator.RunInfo) map[string]any {
	payload := formatRunInfoPayload(runInfo)
	runtimeProvider := strings.TrimSpace(runInfo.StepProviders[acpruntime.StepProviderQA])
	payload["runtime_provider"] = formatOptionalString(runtimeProvider)
	payload["provider"] = formatOptionalString(runtimeProvider)
	return payload
}

func (s *Server) formatQARunPayload(runInfo orchestrator.RunInfo) (map[string]any, error) {
	payload := formatQARunSummaryPayload(runInfo)
	payload["answer"] = nil
	payload["citations"] = []qa.Citation{}
	payload["unresolved"] = []string{}
	payload["confidence"] = nil
	payload["generated_at"] = nil
	if runInfo.Status != orchestrator.RunStatusSucceeded {
		return payload, nil
	}

	answerRel := path.Join("reports", "taskruns", runInfo.RunID, "qa", "qa-answer.json")
	answerRaw, err := s.getWorkspace().ReadFile(answerRel)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return payload, nil
		}
		return nil, fmt.Errorf("read %s: %w", answerRel, err)
	}
	answer, err := qa.ParseAnswer(answerRaw)
	if err != nil {
		return nil, err
	}
	contextRel := path.Join("reports", "taskruns", runInfo.RunID, "qa", "context-pack.json")
	contextRaw, err := s.getWorkspace().ReadFile(contextRel)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", contextRel, err)
	}
	contextPack, err := qa.ParseContextPack(contextRaw)
	if err != nil {
		return nil, err
	}
	if err := qa.ValidateAnswerAgainstContext(answer, contextPack); err != nil {
		return nil, err
	}
	payload["answer"] = answer.Answer
	payload["citations"] = answer.Citations
	payload["unresolved"] = answer.Unresolved
	payload["confidence"] = answer.Confidence
	payload["provider"] = answer.Provider
	payload["generated_at"] = answer.GeneratedAt
	return payload, nil
}

func formatPermissionRequests(requests []acpruntime.PermissionRequest) []acpruntime.PermissionRequest {
	if requests == nil {
		return []acpruntime.PermissionRequest{}
	}
	return requests
}

func mapTypedRunnerAPIError(err error) (statusCode int, code string, message string, ok bool) {
	runnerCode, runnerMessage, classified := acpruntime.ClassifyError(err)
	if !classified {
		return 0, "", "", false
	}
	switch runnerCode {
	case string(acpruntime.ErrorCodeRunnerUnavailable):
		return http.StatusServiceUnavailable, runnerCode, runnerMessage, true
	case string(acpruntime.ErrorCodeRuntimeContract), string(acpruntime.ErrorCodeRuntimeTimeout), string(acpruntime.ErrorCodeRunCanceled):
		// Runtime execution failures are surfaced as run-level failures (`error_code`) after async start.
		return 0, "", "", false
	default:
		return http.StatusBadRequest, runnerCode, runnerMessage, true
	}
}
