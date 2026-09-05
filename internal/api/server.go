package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/GrinRus/ProvenArch/internal/artifactaudit"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/doctor"
	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	"github.com/GrinRus/ProvenArch/internal/proposaldraft"
	"github.com/GrinRus/ProvenArch/internal/qa"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimeprofile"
	producttasks "github.com/GrinRus/ProvenArch/internal/tasks"
	"github.com/GrinRus/ProvenArch/internal/workspace"
	"github.com/GrinRus/ProvenArch/internal/workspacehealth"
)

type Server struct {
	mu                 sync.RWMutex
	admissionMu        sync.Mutex
	attemptWatchCtx    context.Context
	attemptWatchCancel context.CancelFunc
	attemptWatchWG     sync.WaitGroup
	workspace          workspace.Root
	workspacePath      string
	workspaceSelected  bool
	runtimeSelected    bool
	launcherMode       bool
	consoleEntered     bool
	service            *orchestrator.Service
	taskRegistry       *producttasks.Registry
	taskRegistryErr    error
	runtimeConfig      ServerRuntimeConfig
	serviceFactory     ServiceFactory
	generation         uint64
	admissionHook      func(string)
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

const (
	defaultServerShutdownTimeout = 10 * time.Second
	attemptWatcherShutdownGrace  = 500 * time.Millisecond
)

var errSessionMutationConflict = errors.New("workspace or runtime cannot change while a run is active or queued")
var errRuntimeSwitchRequiresRestart = errors.New("runtime switch requires server restart after entering Console")

type serverSessionSnapshot struct {
	Workspace         workspace.Root
	WorkspacePath     string
	WorkspaceSelected bool
	RuntimeSelected   bool
	LauncherMode      bool
	ConsoleEntered    bool
	Service           *orchestrator.Service
	RuntimeConfig     ServerRuntimeConfig
	Generation        uint64
}

func (snapshot serverSessionSnapshot) ready() bool {
	return snapshot.WorkspaceSelected && snapshot.Workspace.Path != "" && snapshot.Service != nil
}

func NewServer(ws workspace.Root, service *orchestrator.Service) *Server {
	return NewServerWithRuntime(ws, service, ServerRuntimeConfig{
		Mode:           acpruntime.RuntimeModeFake,
		Provider:       acpruntime.ProviderClaudeCode,
		ProviderSource: acpruntime.ProviderSourceDefault,
	})
}

func NewServerWithRuntime(ws workspace.Root, service *orchestrator.Service, runtimeConfig ServerRuntimeConfig) *Server {
	runtimeConfig = normalizeServerRuntimeConfig(runtimeConfig, ServerRuntimeConfig{})
	attemptWatchCtx, attemptWatchCancel := context.WithCancel(context.Background())
	if service != nil {
		service.SetRuntimeMode(runtimeConfig.Mode)
	}
	taskRegistry, taskRegistryErr := producttasks.NewRegistry(ws, time.Now)
	server := &Server{
		attemptWatchCtx:    attemptWatchCtx,
		attemptWatchCancel: attemptWatchCancel,
		workspace:          ws,
		workspacePath:      ws.Path,
		workspaceSelected:  true,
		runtimeSelected:    true,
		runtimeConfig:      runtimeConfig,
		service:            service,
		taskRegistry:       taskRegistry,
		taskRegistryErr:    taskRegistryErr,
		consoleEntered:     true,
		generation:         1,
	}
	server.reconcileTaskAttemptsAfterRestart(service, taskRegistry)
	return server
}

func NewLauncherServer(runtimeConfig ServerRuntimeConfig, factory ServiceFactory) *Server {
	runtimeConfig = normalizeServerRuntimeConfig(runtimeConfig, ServerRuntimeConfig{})
	attemptWatchCtx, attemptWatchCancel := context.WithCancel(context.Background())
	return &Server{
		attemptWatchCtx:    attemptWatchCtx,
		attemptWatchCancel: attemptWatchCancel,
		launcherMode:       true,
		runtimeConfig:      runtimeConfig,
		serviceFactory:     factory,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/system/version", s.handleSystemVersion)
	mux.HandleFunc("/api/onboarding/status", s.handleOnboardingStatus)
	mux.HandleFunc("/api/onboarding/workspace", s.handleOnboardingWorkspace)
	mux.HandleFunc("/api/onboarding/runtime", s.handleOnboardingRuntime)
	mux.HandleFunc("/api/onboarding/enter-console", s.handleOnboardingEnterConsole)
	mux.HandleFunc("/api/onboarding/path-suggestions", s.handleOnboardingPathSuggestions)
	mux.HandleFunc("/api/onboarding/recent-workspaces/forget", s.handleOnboardingRecentWorkspaceForget)
	mux.HandleFunc("/api/system/info", s.handleSystemInfo)
	mux.HandleFunc("/api/system/doctor", s.handleSystemDoctor)
	mux.HandleFunc("/api/workspace/health", s.handleWorkspaceHealth)
	mux.HandleFunc("/api/workspace/validate", s.handleWorkspaceValidate)
	mux.HandleFunc("/api/workspace/bundle", s.handleWorkspaceBundle)
	mux.HandleFunc("/api/workspace/manifest", s.handleWorkspaceManifest)
	mux.HandleFunc("/api/runtime/timeouts", s.handleRuntimeTimeouts)
	mux.HandleFunc("/api/runtime/execution", s.handleRuntimeExecution)
	mux.HandleFunc("/api/runtime/models", s.handleRuntimeModels)
	mux.HandleFunc("/api/runtime/permissions", s.handleRuntimePermissions)
	mux.HandleFunc("/api/runtime/profile", s.handleRuntimeProfile)
	mux.HandleFunc("/api/artifacts", s.handleArtifacts)
	mux.HandleFunc("/api/repository-evidence", s.handleRepositoryEvidence)
	mux.HandleFunc("/api/artifacts/write", s.handleArtifactsWrite)
	mux.HandleFunc("/api/knowledge", s.handleKnowledge)
	mux.HandleFunc("/api/architecture", s.handleArchitecture)
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
	mux.HandleFunc("/api/tasks", s.handleTasks)
	mux.HandleFunc("/api/tasks/", s.handleTasks)

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/robots.txt" {
			writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte("User-agent: *\nDisallow:\n"))
			return
		}
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
	if ctx == nil {
		ctx = context.Background()
	}

	// Admission and shutdown share the same lease. This prevents a request from
	// registering a new watcher while Shutdown is canceling and waiting for the
	// existing watcher set.
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	service := s.getService()
	if service == nil {
		if s.attemptWatchCancel != nil {
			s.attemptWatchCancel()
			s.attemptWatchCancel = nil
		}
		s.attemptWatchWG.Wait()
		return nil
	}

	// Let the orchestrator publish cancellation/terminal state first. Watchers
	// remain alive during that transition so the exact Attempt state can be
	// mirrored before their lifecycle context is canceled below.
	serviceErr := service.Shutdown(ctx)

	// Give watchers a bounded grace period to mirror the terminal state just
	// published by the orchestrator. A broken Task-history writer must not make
	// server shutdown wait forever; the lifecycle context remains the fallback.
	watchersDone := make(chan struct{})
	go func() {
		s.attemptWatchWG.Wait()
		close(watchersDone)
	}()
	graceTimer := time.NewTimer(attemptWatcherShutdownGrace)
	defer graceTimer.Stop()
	select {
	case <-watchersDone:
	case <-graceTimer.C:
	}
	if s.attemptWatchCancel != nil {
		s.attemptWatchCancel()
		s.attemptWatchCancel = nil
	}
	s.attemptWatchWG.Wait()
	return serviceErr
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func normalizeServerRuntimeConfig(config ServerRuntimeConfig, previous ServerRuntimeConfig) ServerRuntimeConfig {
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
		config.Build = previous.Build
	}
	if executionOverridesEmpty(config.ExecutionOverrides) {
		config.ExecutionOverrides = previous.ExecutionOverrides
	}
	if config.RunLogsTTL == 0 {
		config.RunLogsTTL = previous.RunLogsTTL
	}
	if config.RunLogsMaxRuns == 0 {
		config.RunLogsMaxRuns = previous.RunLogsMaxRuns
	}
	return config
}

func executionOverridesEmpty(overrides acpruntime.ExecutionOverrides) bool {
	return overrides.Strategy == nil &&
		overrides.MaxParallel == nil &&
		overrides.FailurePolicy == nil &&
		overrides.ShardMode == nil
}

func (s *Server) sessionSnapshot() serverSessionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return serverSessionSnapshot{
		Workspace:         s.workspace,
		WorkspacePath:     s.workspacePath,
		WorkspaceSelected: s.workspaceSelected,
		RuntimeSelected:   s.runtimeSelected,
		LauncherMode:      s.launcherMode,
		ConsoleEntered:    s.consoleEntered,
		Service:           s.service,
		RuntimeConfig:     s.runtimeConfig,
		Generation:        s.generation,
	}
}

func (s *Server) getWorkspace() workspace.Root {
	return s.sessionSnapshot().Workspace
}

func (s *Server) getWorkspacePath() string {
	return s.sessionSnapshot().WorkspacePath
}

func (s *Server) getService() *orchestrator.Service {
	return s.sessionSnapshot().Service
}

func (s *Server) taskRegistrySnapshot() (*producttasks.Registry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.taskRegistry == nil {
		if s.taskRegistryErr != nil {
			return nil, s.taskRegistryErr
		}
		return nil, errors.New("task history is not available")
	}
	return s.taskRegistry, nil
}

func (s *Server) resetTaskRegistryLocked(ws workspace.Root) {
	registry, err := producttasks.NewRegistry(ws, time.Now)
	s.taskRegistry = registry
	s.taskRegistryErr = err
	if err == nil {
		s.reconcileTaskAttemptsAfterRestart(s.service, registry)
	}
}

func (s *Server) serviceHasInFlightWorkLocked() bool {
	return s.service != nil && s.service.HasInFlightRun()
}

func (s *Server) mutateWorkspaceRoot(mutate func(workspace.Root) (workspace.Root, error)) (workspace.Root, error) {
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.serviceHasInFlightWorkLocked() {
		return workspace.Root{}, errSessionMutationConflict
	}
	reopened, err := mutate(s.workspace)
	if err != nil {
		return workspace.Root{}, err
	}
	s.workspace = reopened
	s.workspacePath = reopened.Path
	s.workspaceSelected = true
	s.resetTaskRegistryLocked(reopened)
	s.generation++
	return reopened, nil
}

func (s *Server) saveWorkspaceManifest(content string) (*orchestrator.Service, error) {
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.serviceHasInFlightWorkLocked() {
		return nil, errSessionMutationConflict
	}
	workspacePath := strings.TrimSpace(s.workspacePath)
	if workspacePath == "" {
		return nil, errors.New("workspace is not selected")
	}
	current := workspace.Root{Path: workspacePath}
	if err := current.WriteFileAtomic(workspace.ManifestFileName, []byte(content)); err != nil {
		return nil, fmt.Errorf("manifest_write_failed: %w", err)
	}
	reopened, err := workspace.Open(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("manifest_reopen_failed: %w", err)
	}
	if err := reopened.EnsureLayout(); err != nil {
		return nil, fmt.Errorf("workspace_layout_failed: %w", err)
	}
	if err := reopened.EnsureBaselineBundle(); err != nil {
		return nil, fmt.Errorf("workspace_baseline_failed: %w", err)
	}
	s.workspace = reopened
	s.workspacePath = reopened.Path
	s.workspaceSelected = true
	s.resetTaskRegistryLocked(reopened)
	var reconcileService *orchestrator.Service
	if s.service == nil && s.serviceFactory != nil {
		s.service = s.serviceFactory(reopened, s.runtimeConfig)
		reconcileService = s.service
	}
	s.generation++
	return reconcileService, nil
}

func (s *Server) setWorkspace(ws workspace.Root) error {
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.serviceHasInFlightWorkLocked() {
		return errSessionMutationConflict
	}
	s.workspace = ws
	s.workspacePath = ws.Path
	s.workspaceSelected = true
	s.resetTaskRegistryLocked(ws)
	s.generation++
	return nil
}

func (s *Server) setDraftWorkspace(path string) error {
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.serviceHasInFlightWorkLocked() {
		return errSessionMutationConflict
	}
	s.workspace = workspace.Root{}
	s.workspacePath = path
	s.workspaceSelected = true
	s.service = nil
	s.taskRegistry = nil
	s.taskRegistryErr = nil
	s.generation++
	return nil
}

func (s *Server) attachWorkspace(ws workspace.Root) (*orchestrator.Service, error) {
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.serviceHasInFlightWorkLocked() {
		return nil, errSessionMutationConflict
	}
	s.workspace = ws
	s.workspacePath = ws.Path
	s.workspaceSelected = true
	if s.serviceFactory != nil {
		s.service = s.serviceFactory(ws, s.runtimeConfig)
	}
	s.resetTaskRegistryLocked(ws)
	s.generation++
	return s.service, nil
}

func (s *Server) setRuntimeConfig(config ServerRuntimeConfig) (*orchestrator.Service, error) {
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.consoleEntered {
		return nil, errRuntimeSwitchRequiresRestart
	}
	if s.serviceHasInFlightWorkLocked() {
		return nil, errSessionMutationConflict
	}
	config = normalizeServerRuntimeConfig(config, s.runtimeConfig)
	s.runtimeConfig = config
	s.runtimeSelected = true
	if s.workspaceSelected && s.workspace.Path != "" && s.serviceFactory != nil {
		s.service = s.serviceFactory(s.workspace, s.runtimeConfig)
	}
	s.generation++
	return s.service, nil
}

func (s *Server) enterConsole() error {
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.workspaceSelected || s.workspace.Path == "" || s.service == nil || !s.runtimeSelected {
		return errors.New("workspace and runtime must be ready before entering Console")
	}
	s.consoleEntered = true
	s.generation++
	return nil
}

func (s *Server) shouldBlockAPIRequest(apiPath string) bool {
	if apiPath == "/api/health" || apiPath == "/api/system/version" || apiPath == "/api/system/info" || strings.HasPrefix(apiPath, "/api/onboarding/") {
		return false
	}
	snapshot := s.sessionSnapshot()
	selected := snapshot.WorkspaceSelected
	ready := snapshot.ready()
	pathSelected := selected && strings.TrimSpace(snapshot.WorkspacePath) != ""
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
	info := s.sessionSnapshot().RuntimeConfig.Build
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

func (s *Server) handleWorkspaceHealth(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}

	report, err := workspacehealth.Scan(request.Context(), s.getWorkspace(), workspacehealth.Options{})
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "workspace_health_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, report)
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

		service, err := s.saveWorkspaceManifest(payload.Content)
		if err != nil {
			switch {
			case errors.Is(err, errSessionMutationConflict):
				writeError(writer, http.StatusConflict, "workspace_switch_conflict", err.Error())
			case strings.Contains(err.Error(), "workspace is not selected"):
				writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select or create an ACP workspace before saving workspace.yaml")
			case strings.HasPrefix(err.Error(), "manifest_write_failed:"):
				writeError(writer, http.StatusInternalServerError, "manifest_write_failed", strings.TrimPrefix(err.Error(), "manifest_write_failed: "))
			case strings.HasPrefix(err.Error(), "manifest_reopen_failed:"):
				writeError(writer, http.StatusInternalServerError, "manifest_reopen_failed", strings.TrimPrefix(err.Error(), "manifest_reopen_failed: "))
			case strings.HasPrefix(err.Error(), "workspace_layout_failed:"):
				writeError(writer, http.StatusInternalServerError, "workspace_layout_failed", strings.TrimPrefix(err.Error(), "workspace_layout_failed: "))
			case strings.HasPrefix(err.Error(), "workspace_baseline_failed:"):
				writeError(writer, http.StatusInternalServerError, "workspace_baseline_failed", strings.TrimPrefix(err.Error(), "workspace_baseline_failed: "))
			default:
				writeError(writer, http.StatusInternalServerError, "manifest_save_failed", err.Error())
			}
			return
		}
		if service != nil {
			service.ReconcileStaleRunsAfterRestart()
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

		reopened, err := s.mutateWorkspaceRoot(func(ws workspace.Root) (workspace.Root, error) {
			return (runtimeprofile.RuntimeProfilePatchService{}).ApplyTimeouts(ws, payload.Timeouts)
		})
		if err != nil {
			if errors.Is(err, errSessionMutationConflict) {
				writeError(writer, http.StatusConflict, "runtime_profile_conflict", err.Error())
				return
			}
			if typed := (runtimeprofile.PatchError{}); errors.As(err, &typed) {
				writeError(writer, http.StatusInternalServerError, typed.Code, typed.Error())
				return
			}
			writeError(writer, http.StatusInternalServerError, "runtime_timeouts_write_failed", err.Error())
			return
		}
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
		snapshot := s.sessionSnapshot()
		resolved := snapshot.Service.ResolveExecutionProfile(snapshot.Workspace.Manifest)
		stepProviders, err := snapshot.Service.ResolveStepProviderProfile(snapshot.Workspace.Manifest)
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

		reopened, err := s.mutateWorkspaceRoot(func(ws workspace.Root) (workspace.Root, error) {
			return (runtimeprofile.RuntimeProfilePatchService{}).ApplyExecution(ws, payload.Execution)
		})
		if err != nil {
			if errors.Is(err, errSessionMutationConflict) {
				writeError(writer, http.StatusConflict, "runtime_profile_conflict", err.Error())
				return
			}
			if typed := (runtimeprofile.PatchError{}); errors.As(err, &typed) {
				writeError(writer, http.StatusInternalServerError, typed.Code, typed.Error())
				return
			}
			writeError(writer, http.StatusInternalServerError, "runtime_execution_write_failed", err.Error())
			return
		}
		snapshot := s.sessionSnapshot()
		resolved := snapshot.Service.ResolveExecutionProfile(reopened.Manifest)
		stepProviders, err := snapshot.Service.ResolveStepProviderProfile(reopened.Manifest)
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

func (s *Server) handleRuntimeModels(writer http.ResponseWriter, request *http.Request) {
	snapshot := s.sessionSnapshot()
	resolved, err := snapshot.Service.ResolveProviderModelProfile(snapshot.Workspace.Manifest)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "runtime_provider_models_invalid", err.Error())
		return
	}
	switch request.Method {
	case http.MethodGet:
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"providers": runtimeProviderModelsPayload(resolved),
		})
	case http.MethodPut:
		var payload runtimeprofile.RuntimeProviderModelsPatch
		if err := decodeStrictJSON(request, &payload); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		if payload.IsZero() {
			writeError(writer, http.StatusBadRequest, "runtime_provider_models_empty", "providers payload is required")
			return
		}
		if err := runtimeprofile.ValidateRuntimeProviderModelsPatch(payload); err != nil {
			writeError(writer, http.StatusBadRequest, "runtime_provider_models_invalid", err.Error())
			return
		}
		reopened, err := s.mutateWorkspaceRoot(func(ws workspace.Root) (workspace.Root, error) {
			return (runtimeprofile.RuntimeProfilePatchService{}).ApplyProviderModels(ws, payload)
		})
		if err != nil {
			if errors.Is(err, errSessionMutationConflict) {
				writeError(writer, http.StatusConflict, "runtime_profile_conflict", err.Error())
				return
			}
			if typed := (runtimeprofile.PatchError{}); errors.As(err, &typed) {
				writeError(writer, http.StatusInternalServerError, typed.Code, typed.Error())
				return
			}
			writeError(writer, http.StatusInternalServerError, "runtime_provider_models_write_failed", err.Error())
			return
		}
		resolved, err = snapshot.Service.ResolveProviderModelProfile(reopened.Manifest)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "runtime_provider_models_reopen_failed", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"providers": runtimeProviderModelsPayload(resolved),
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

		reopened, err := s.mutateWorkspaceRoot(func(ws workspace.Root) (workspace.Root, error) {
			return (runtimeprofile.RuntimeProfilePatchService{}).ApplyPermissions(ws, payload.Permissions)
		})
		if err != nil {
			if errors.Is(err, errSessionMutationConflict) {
				writeError(writer, http.StatusConflict, "runtime_profile_conflict", err.Error())
				return
			}
			if typed := (runtimeprofile.PatchError{}); errors.As(err, &typed) {
				writeError(writer, http.StatusInternalServerError, typed.Code, typed.Error())
				return
			}
			writeError(writer, http.StatusInternalServerError, "runtime_permissions_write_failed", err.Error())
			return
		}
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
	snapshot := s.sessionSnapshot()
	timeouts := acpruntime.ResolveTimeouts(snapshot.Workspace.Manifest)
	execution := snapshot.Service.ResolveExecutionProfile(snapshot.Workspace.Manifest)
	permissions := acpruntime.ResolvePermissions(snapshot.Workspace.Manifest)
	stepProviders, err := snapshot.Service.ResolveStepProviderProfile(snapshot.Workspace.Manifest)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "runtime_step_provider_resolution_failed", err.Error())
		return
	}
	providerModels, err := snapshot.Service.ResolveProviderModelProfile(snapshot.Workspace.Manifest)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "runtime_provider_models_resolution_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":               true,
		"runtime_mode":     snapshot.RuntimeConfig.Mode,
		"runtime_provider": snapshot.RuntimeConfig.Provider,
		"provider_source":  snapshot.RuntimeConfig.ProviderSource,
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
		"provider_models": runtimeProviderModelsPayload(providerModels),
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

func runtimeProviderModelsPayload(resolved acpruntime.ProviderModelResolution) map[string]any {
	payload := map[string]any{}
	for _, provider := range acpruntime.SupportedProviders() {
		persisted := map[string]any{}
		if config := resolved.Persisted[string(provider)]; config != nil {
			if strings.TrimSpace(config.Model) != "" {
				persisted["model"] = strings.TrimSpace(config.Model)
			}
			if strings.TrimSpace(config.Effort) != "" {
				persisted["effort"] = strings.TrimSpace(config.Effort)
			}
		}
		effective := resolved.Effective[provider]
		effectivePayload := map[string]any{}
		if strings.TrimSpace(effective.Model) != "" {
			effectivePayload["model"] = strings.TrimSpace(effective.Model)
		}
		if strings.TrimSpace(effective.Effort) != "" {
			effectivePayload["effort"] = strings.TrimSpace(effective.Effort)
		}
		sources := resolved.Source[provider]
		payload[string(provider)] = map[string]any{
			"persisted": persisted,
			"effective": effectivePayload,
			"source": map[string]any{
				"model":  sources.Model,
				"effort": sources.Effort,
			},
			"capabilities": map[string]any{
				"model":   true,
				"efforts": acpruntime.SupportedEfforts(provider),
			},
		}
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
		Message             string `json:"message"`
		ExpectedFingerprint string `json:"expected_fingerprint"`
		ExpectedHeadOID     string `json:"expected_head_oid"`
		TaskID              string `json:"task_id,omitempty"`
		AttemptID           string `json:"attempt_id,omitempty"`
		RunID               string `json:"run_id,omitempty"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	message := strings.TrimSpace(payload.Message)
	if message == "" {
		message = "chore: update ACP workspace artifacts"
	}
	if strings.TrimSpace(payload.ExpectedFingerprint) == "" {
		writeError(writer, http.StatusBadRequest, "git_confirmation_required", "expected_fingerprint is required")
		return
	}
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	snapshot := s.sessionSnapshot()
	publicationContext := taskPublicationContext{TaskID: payload.TaskID, AttemptID: payload.AttemptID, RunID: payload.RunID}
	var registry *producttasks.Registry
	if publicationContext.provided() {
		var err error
		registry, err = s.taskRegistrySnapshot()
		if err != nil {
			writeTaskHistoryUnavailable(writer, err)
			return
		}
		if _, _, err := validateTaskPublicationContext(registry.Snapshot(), publicationContext); err != nil {
			if strings.Contains(err.Error(), "was not found") {
				writeError(writer, http.StatusNotFound, "publication_context_not_found", err.Error())
			} else {
				writeError(writer, http.StatusBadRequest, "publication_context_invalid", err.Error())
			}
			return
		}
	}
	if snapshot.Service != nil && snapshot.Service.HasInFlightRun() {
		writeError(writer, http.StatusConflict, "run_active", "Git mutation is blocked while a run is active or queued")
		return
	}
	if s.admissionHook != nil {
		s.admissionHook("git_commit_commit")
	}
	ws := snapshot.Workspace
	state, err := collectWorkspaceGitState(request.Context(), ws)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "git_diff_failed", err.Error())
		return
	}
	if state.Fingerprint != strings.TrimSpace(payload.ExpectedFingerprint) || state.Identity.HeadOID != strings.TrimSpace(payload.ExpectedHeadOID) {
		writeError(writer, http.StatusConflict, "stale_git_confirmation", "workspace Git state changed after confirmation")
		return
	}
	if _, err := runGit(request.Context(), ws.Path, "add", "-A"); err != nil {
		writeError(writer, http.StatusBadRequest, "git_add_failed", err.Error())
		return
	}
	output, err := runGit(request.Context(), ws.Path, "commit", "-m", message)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "nothing to commit") {
			writeJSON(writer, http.StatusOK, map[string]any{
				"ok":          true,
				"status":      "no_changes",
				"message":     "nothing to commit",
				"publication": publicationUnavailable("Git commit did not create a new publication"),
			})
			return
		}
		writeError(writer, http.StatusBadRequest, "git_commit_failed", err.Error())
		return
	}
	publication := publicationUnavailable("no exact Task/Attempt/run context was supplied")
	if publicationContext.provided() {
		after, stateErr := collectWorkspaceGitState(request.Context(), ws)
		if stateErr != nil {
			writeError(writer, http.StatusInternalServerError, "publication_state_unavailable", stateErr.Error())
			return
		}
		publication = buildTaskPublication("commit", publicationContext, state, after)
		if err := recordTaskPublication(registry, publicationContext, publication); err != nil {
			writeError(writer, http.StatusInternalServerError, "publication_linkage_failed", err.Error())
			return
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":          true,
		"status":      "committed",
		"output":      output,
		"publication": publication,
	})
}

func (s *Server) handleGitProposalBranch(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload struct {
		Name                 string `json:"name"`
		ExpectedFingerprint  string `json:"expected_fingerprint"`
		ExpectedSourceBranch string `json:"expected_source_branch"`
		ExpectedBaseRef      string `json:"expected_base_ref"`
		ExpectedBaseOID      string `json:"expected_base_oid"`
		ExpectedHeadOID      string `json:"expected_head_oid"`
		TaskID               string `json:"task_id,omitempty"`
		AttemptID            string `json:"attempt_id,omitempty"`
		RunID                string `json:"run_id,omitempty"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	branch := sanitizeBranchName(payload.Name)
	if branch == "" {
		branch = "proposal/" + time.Now().UTC().Format("20060102-150405")
	}

	if strings.TrimSpace(payload.ExpectedFingerprint) == "" || strings.TrimSpace(payload.ExpectedSourceBranch) == "" || strings.TrimSpace(payload.ExpectedBaseRef) == "" {
		writeError(writer, http.StatusBadRequest, "git_confirmation_required", "expected_fingerprint, expected_source_branch and expected_base_ref are required")
		return
	}
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	snapshot := s.sessionSnapshot()
	publicationContext := taskPublicationContext{TaskID: payload.TaskID, AttemptID: payload.AttemptID, RunID: payload.RunID}
	var registry *producttasks.Registry
	if publicationContext.provided() {
		var err error
		registry, err = s.taskRegistrySnapshot()
		if err != nil {
			writeTaskHistoryUnavailable(writer, err)
			return
		}
		if _, _, err := validateTaskPublicationContext(registry.Snapshot(), publicationContext); err != nil {
			if strings.Contains(err.Error(), "was not found") {
				writeError(writer, http.StatusNotFound, "publication_context_not_found", err.Error())
			} else {
				writeError(writer, http.StatusBadRequest, "publication_context_invalid", err.Error())
			}
			return
		}
	}
	if snapshot.Service != nil && snapshot.Service.HasInFlightRun() {
		writeError(writer, http.StatusConflict, "run_active", "Git mutation is blocked while a run is active or queued")
		return
	}
	if s.admissionHook != nil {
		s.admissionHook("git_branch_commit")
	}
	ws := snapshot.Workspace
	state, err := collectWorkspaceGitState(request.Context(), ws)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "git_diff_failed", err.Error())
		return
	}
	if state.Fingerprint != strings.TrimSpace(payload.ExpectedFingerprint) ||
		state.Identity.Branch != strings.TrimSpace(payload.ExpectedSourceBranch) ||
		state.Identity.BaseRef != strings.TrimSpace(payload.ExpectedBaseRef) ||
		state.Identity.BaseOID != strings.TrimSpace(payload.ExpectedBaseOID) ||
		state.Identity.HeadOID != strings.TrimSpace(payload.ExpectedHeadOID) {
		writeError(writer, http.StatusConflict, "stale_git_confirmation", "workspace Git state changed after confirmation")
		return
	}
	if _, err := runGit(request.Context(), ws.Path, "checkout", "-b", branch); err != nil {
		if _, fallbackErr := runGit(request.Context(), ws.Path, "checkout", branch); fallbackErr != nil {
			writeError(writer, http.StatusBadRequest, "git_branch_failed", err.Error())
			return
		}
	}
	publication := publicationUnavailable("no exact Task/Attempt/run context was supplied")
	if publicationContext.provided() {
		after, stateErr := collectWorkspaceGitState(request.Context(), ws)
		if stateErr != nil {
			writeError(writer, http.StatusInternalServerError, "publication_state_unavailable", stateErr.Error())
			return
		}
		publication = buildTaskPublication("branch", publicationContext, state, after)
		if err := recordTaskPublication(registry, publicationContext, publication); err != nil {
			writeError(writer, http.StatusInternalServerError, "publication_linkage_failed", err.Error())
			return
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":          true,
		"branch":      branch,
		"publication": publication,
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
	const maxArtifactReadBytes = 2 * 1024 * 1024
	content, err := ws.ReadFileLimit(relPath, maxArtifactReadBytes)
	if err != nil {
		if errors.Is(err, workspace.ErrPathTraversal) || errors.Is(err, workspace.ErrPathAbsolute) {
			writeError(writer, http.StatusBadRequest, "path_invalid", err.Error())
			return
		}
		if errors.Is(err, workspace.ErrFileTooLarge) {
			writeError(writer, http.StatusRequestEntityTooLarge, "artifact_too_large", "artifact exceeds the 2 MiB viewer read budget")
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
		if strings.HasSuffix(strings.TrimRight(request.URL.Path, "/"), "/proposal-draft") {
			s.handleQAProposalDraft(writer, request)
		} else {
			s.handleQARunsPost(writer, request)
		}
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPost)
	}
}

func (s *Server) handleQAProposalDraft(writer http.ResponseWriter, request *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(request.URL.Path, "/api/qa/runs/"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || parts[1] != "proposal-draft" {
		writeError(writer, http.StatusNotFound, "qa_run_not_found", "qa run not found")
		return
	}
	var payload struct {
		Title                string `json:"title"`
		ExpectedAnswerDigest string `json:"expected_answer_digest"`
		Slug                 string `json:"slug"`
		OperatorNote         string `json:"operator_note"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	if strings.TrimSpace(payload.Title) == "" {
		writeError(writer, http.StatusBadRequest, "proposal_title_required", "title is required")
		return
	}
	if strings.TrimSpace(payload.ExpectedAnswerDigest) == "" {
		writeError(writer, http.StatusBadRequest, "answer_digest_required", "expected_answer_digest is required")
		return
	}

	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	snapshot := s.sessionSnapshot()
	if snapshot.Service == nil {
		writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select a workspace before creating a proposal draft")
		return
	}
	if snapshot.Service.HasInFlightRun() {
		writeError(writer, http.StatusConflict, "run_active", "proposal mutation is blocked while a run is active or queued")
		return
	}
	runID := strings.TrimSpace(parts[0])
	runInfo, ok := snapshot.Service.GetRun(runID)
	if !ok || runInfo.Pipeline != string(orchestrator.PipelineQA) {
		writeError(writer, http.StatusNotFound, "qa_run_not_found", "qa run not found")
		return
	}
	if runInfo.Status != orchestrator.RunStatusSucceeded {
		writeError(writer, http.StatusConflict, "qa_run_not_succeeded", "proposal draft requires a succeeded QA run")
		return
	}
	qaRoot := path.Join("reports", "taskruns", runID, "qa")
	answerRaw, err := snapshot.Workspace.ReadFile(path.Join(qaRoot, "qa-answer.json"))
	if err != nil {
		writeError(writer, http.StatusConflict, "qa_answer_unavailable", "the immutable QA answer is unavailable")
		return
	}
	contextRaw, err := snapshot.Workspace.ReadFile(path.Join(qaRoot, "context-pack.json"))
	if err != nil {
		writeError(writer, http.StatusConflict, "qa_answer_unavailable", "the immutable QA context is unavailable")
		return
	}
	if s.admissionHook != nil {
		s.admissionHook("qa_proposal_draft_commit")
	}
	current := s.sessionSnapshot()
	if current.Generation != snapshot.Generation || current.Service != snapshot.Service || current.Workspace.Path != snapshot.Workspace.Path {
		writeError(writer, http.StatusConflict, "session_generation_changed", "workspace session changed before proposal creation")
		return
	}
	result, err := proposaldraft.Create(snapshot.Workspace, proposaldraft.Input{
		RunID:                runID,
		Title:                payload.Title,
		Slug:                 payload.Slug,
		OperatorNote:         payload.OperatorNote,
		ExpectedAnswerDigest: payload.ExpectedAnswerDigest,
		AnswerRaw:            answerRaw,
		ContextRaw:           contextRaw,
		CreatedAt:            time.Now().UTC(),
	})
	if err != nil {
		switch {
		case errors.Is(err, proposaldraft.ErrStaleDigest):
			writeError(writer, http.StatusConflict, "qa_answer_stale", err.Error())
		case errors.Is(err, proposaldraft.ErrAlreadyExists):
			writeError(writer, http.StatusConflict, "proposal_already_exists", err.Error())
		case errors.Is(err, proposaldraft.ErrInvalidSlug):
			writeError(writer, http.StatusBadRequest, "proposal_slug_invalid", err.Error())
		case errors.Is(err, proposaldraft.ErrUnresolvedCitation):
			writeError(writer, http.StatusUnprocessableEntity, "qa_citation_unresolved", err.Error())
		default:
			writeError(writer, http.StatusInternalServerError, "proposal_draft_create_failed", err.Error())
		}
		return
	}
	writeJSON(writer, http.StatusCreated, result)
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
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	snapshot := s.sessionSnapshot()
	if !snapshot.RuntimeSelected {
		writeError(writer, http.StatusBadRequest, "runtime_not_selected", "select a runner before starting Q&A")
		return
	}
	if snapshot.Service == nil {
		writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select or create an ACP workspace before starting Q&A")
		return
	}
	if s.admissionHook != nil {
		s.admissionHook("qa_start_commit")
	}
	if current := s.sessionSnapshot(); current.Generation != snapshot.Generation || current.Service != snapshot.Service {
		writeError(writer, http.StatusConflict, "session_generation_changed", "workspace session changed before Q&A admission")
		return
	}
	runID, err := snapshot.Service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace:      snapshot.Workspace,
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

	snapshot := s.sessionSnapshot()
	runs := snapshot.Service.ListRuns(0)
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
	snapshot := s.sessionSnapshot()
	runInfo, ok := snapshot.Service.GetRun(runID)
	if !ok || runInfo.Pipeline != string(orchestrator.PipelineQA) {
		writeError(writer, http.StatusNotFound, "qa_run_not_found", "qa run not found")
		return
	}
	payload, err := s.formatQARunPayload(snapshot.Workspace, runInfo)
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
	Intent               string `json:"intent"`
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
	intent := orchestrator.RunIntent(strings.TrimSpace(body.Intent))
	if intent == "" {
		intent = orchestrator.RunIntentStart
	}
	if intent != orchestrator.RunIntentStart && intent != orchestrator.RunIntentQueue {
		writeError(writer, http.StatusBadRequest, "run_intent_invalid", "intent must be start or queue")
		return
	}
	if intent == orchestrator.RunIntentQueue && pipeline != orchestrator.PipelineRefresh {
		writeError(writer, http.StatusBadRequest, "queue_intent_not_supported", "queue intent is supported only for refresh runs")
		return
	}

	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	snapshot := s.sessionSnapshot()
	if !snapshot.RuntimeSelected {
		writeError(writer, http.StatusBadRequest, "runtime_not_selected", "select a runner before starting analysis")
		return
	}
	if snapshot.Service == nil {
		writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select or create an ACP workspace before starting analysis")
		return
	}
	report := snapshot.Workspace.Validate(request.Context(), workspace.ValidateOptions{
		ResolveRepos: true,
		FetchGit:     false,
		VerifyRefs:   true,
	})
	if !report.OK {
		writeJSON(writer, http.StatusBadRequest, map[string]any{
			"ok":        false,
			"workspace": snapshot.Workspace.Path,
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
	if s.admissionHook != nil {
		s.admissionHook("pipeline_start_commit")
	}
	if current := s.sessionSnapshot(); current.Generation != snapshot.Generation ||
		current.Service != snapshot.Service ||
		current.Workspace.Path != snapshot.Workspace.Path {
		writeError(writer, http.StatusConflict, "session_generation_changed", "workspace session changed before run admission")
		return
	}

	runID, err := snapshot.Service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace:      snapshot.Workspace,
		Pipeline:       pipeline,
		NonInteractive: true,
		Intent:         intent,
	})
	if err != nil {
		if errors.Is(err, orchestrator.ErrRunActive) {
			writeError(writer, http.StatusConflict, "run_active", "another run is already active")
			return
		}
		if errors.Is(err, orchestrator.ErrQueueUnsupported) {
			writeError(writer, http.StatusBadRequest, "queue_intent_not_supported", err.Error())
			return
		}
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

	if len(parts) == 2 && parts[1] == "snapshot" {
		s.handlePipelineRunSnapshot(writer, runID)
		return
	}

	if len(parts) == 2 && parts[1] == "audit" {
		s.handlePipelineRunAudit(writer, runID)
		return
	}

	if len(parts) == 2 && parts[1] == "effective-verdict" {
		s.handlePipelineRunEffectiveVerdict(writer, runID)
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

func (s *Server) handlePipelineRunAudit(writer http.ResponseWriter, runID string) {
	snapshot := s.sessionSnapshot()
	runInfo, ok := snapshot.Service.GetRun(runID)
	if !ok || runInfo.Pipeline == string(orchestrator.PipelineQA) {
		writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
		return
	}
	writeJSON(writer, http.StatusOK, artifactaudit.ScanSelectedRunPublic(snapshot.Workspace, runID))
}

type effectiveVerdictResponse struct {
	Status    string                      `json:"status"`
	Authority string                      `json:"authority"`
	Path      string                      `json:"path"`
	Verdict   *contracts.EffectiveVerdict `json:"verdict,omitempty"`
}

func (s *Server) handlePipelineRunEffectiveVerdict(writer http.ResponseWriter, runID string) {
	snapshot := s.sessionSnapshot()
	runInfo, ok := snapshot.Service.GetRun(runID)
	if !ok || runInfo.Pipeline == string(orchestrator.PipelineQA) {
		writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
		return
	}
	verdictPath := path.Join("reports", "taskruns", runID, "validator", "effective-verdict.json")
	raw, err := snapshot.Workspace.ReadFile(verdictPath)
	if err != nil {
		writeJSON(writer, http.StatusOK, effectiveVerdictResponse{Status: "legacy_unavailable", Authority: "legacy", Path: verdictPath})
		return
	}
	verdict, err := contracts.ParseEffectiveVerdict(raw)
	providerPath := path.Join("reports", "taskruns", runID, "validator", "validator-verdict.json")
	providerRaw, providerErr := snapshot.Workspace.ReadFile(providerPath)
	providerDigest := sha256.Sum256(providerRaw)
	if err != nil || verdict.RunID != runID || verdict.ProviderVerdictPath != providerPath || providerErr != nil || hex.EncodeToString(providerDigest[:]) != verdict.ProviderVerdictSHA256 {
		writeJSON(writer, http.StatusOK, effectiveVerdictResponse{Status: "invalid", Authority: "invalid", Path: verdictPath})
		return
	}
	writeJSON(writer, http.StatusOK, effectiveVerdictResponse{Status: "available", Authority: "effective", Path: verdictPath, Verdict: &verdict})
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

	snapshot := s.sessionSnapshot()
	runs := snapshot.Service.ListRuns(0)
	items := make([]map[string]any, 0, len(runs))
	for _, runInfo := range runs {
		if runInfo.Pipeline == string(orchestrator.PipelineQA) {
			continue
		}
		item := formatRunInfoPayload(runInfo)
		artifacts, _ := snapshot.Service.GetRunArtifacts(runInfo.RunID)
		item["authoritative_index"] = hasAuthoritativeRunIndex(runInfo.RunID, artifacts)
		items = append(items, item)
		if len(items) >= limit {
			break
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"items":               items,
		"coordination":        snapshot.Service.Coordination(),
		"history_diagnostics": snapshot.Service.HistoryDiagnostics(),
	})
}

func hasAuthoritativeRunIndex(runID string, artifacts []orchestrator.Artifact) bool {
	expected := filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "final", "final-run-index.json"))
	for _, artifact := range artifacts {
		if filepath.ToSlash(filepath.Clean(artifact.Path)) == expected {
			return true
		}
	}
	return false
}

func (s *Server) handlePipelineRunStatus(writer http.ResponseWriter, runID string) {
	snapshot := s.sessionSnapshot()
	runInfo, ok := snapshot.Service.GetRun(runID)
	if !ok {
		writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
		return
	}
	writeJSON(writer, http.StatusOK, formatRunInfoPayload(runInfo))
}

func (s *Server) handlePipelineRunArtifacts(writer http.ResponseWriter, runID string) {
	snapshot := s.sessionSnapshot()
	artifacts, ok := snapshot.Service.GetRunArtifacts(runID)
	if !ok {
		writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"run_id":    runID,
		"artifacts": artifacts,
	})
}

func (s *Server) handlePipelineRunSnapshot(writer http.ResponseWriter, runID string) {
	snapshot := s.sessionSnapshot()
	artifacts, ok := snapshot.Service.GetRunArtifacts(runID)
	if !ok {
		writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
		return
	}
	writeJSON(writer, http.StatusOK, resolveRunSnapshot(snapshot.Workspace, runID, artifacts))
}

func (s *Server) handlePipelineRunPermissions(writer http.ResponseWriter, runID string) {
	snapshot := s.sessionSnapshot()
	permissions, ok := snapshot.Service.GetRunPermissions(runID)
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

	snapshot := s.sessionSnapshot()
	page, ok, err := snapshot.Service.GetRunLogs(runID, cursor, limit)
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
	if len(parts) == 2 && parts[0] != "" && parts[1] == "retry-plan" {
		s.handlePipelineRetryPlan(writer, request, strings.TrimSpace(parts[0]))
		return
	}
	if len(parts) == 2 && parts[0] != "" && parts[1] == "retry" {
		s.handlePipelineRetry(writer, request, strings.TrimSpace(parts[0]))
		return
	}
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

	snapshot := s.sessionSnapshot()
	if err := snapshot.Service.CancelRun(runID); err != nil {
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
		"run_id":               runInfo.RunID,
		"task_id":              formatOptionalString(runInfo.TaskID),
		"attempt_id":           formatOptionalString(runInfo.AttemptID),
		"pipeline":             runInfo.Pipeline,
		"status":               runInfo.Status,
		"started_at":           runInfo.StartedAt.UTC().Format(time.RFC3339),
		"finished_at":          formatOptionalTime(runInfo.FinishedAt),
		"question":             formatOptionalString(runInfo.Question),
		"current_step":         runInfo.CurrentStep,
		"runtime_mode":         formatOptionalString(runInfo.RuntimeMode),
		"step_providers":       runInfo.StepProviders,
		"warnings":             runInfo.Warnings,
		"pending_permissions":  formatPermissionRequests(runInfo.PendingPermissions),
		"error_code":           formatOptionalString(runInfo.ErrorCode),
		"error":                formatOptionalString(runInfo.Error),
		"superseded_by_run_id": formatOptionalString(runInfo.SupersededByRunID),
		"refresh_summary":      runInfo.RefreshSummary,
		"progress":             runInfo.Progress,
		"retry":                runInfo.Retry,
	}
}

func formatQARunSummaryPayload(runInfo orchestrator.RunInfo) map[string]any {
	payload := formatRunInfoPayload(runInfo)
	runtimeProvider := strings.TrimSpace(runInfo.StepProviders[acpruntime.StepProviderQA])
	payload["runtime_provider"] = formatOptionalString(runtimeProvider)
	payload["provider"] = formatOptionalString(runtimeProvider)
	return payload
}

func (s *Server) formatQARunPayload(ws workspace.Root, runInfo orchestrator.RunInfo) (map[string]any, error) {
	payload := formatQARunSummaryPayload(runInfo)
	qaRoot := path.Join("reports", "taskruns", runInfo.RunID, "qa")
	payload["answer_authority"] = evidenceAuthority{
		Mode:  evidenceAuthorityQASnapshot,
		RunID: runInfo.RunID,
		Root:  qaRoot,
	}
	payload["audit_authority"] = evidenceAuthority{
		Mode:  evidenceAuthorityQAAudit,
		RunID: runInfo.RunID,
		Root:  qaRoot,
	}
	payload["answer_status"] = "not_produced"
	payload["answer_digest"] = nil
	payload["answer"] = nil
	payload["citations"] = []qa.Citation{}
	payload["unresolved"] = []string{}
	payload["confidence"] = nil
	payload["generated_at"] = nil
	if runInfo.Status != orchestrator.RunStatusSucceeded {
		return payload, nil
	}

	answerRel := path.Join(qaRoot, "qa-answer.json")
	answerRaw, err := ws.ReadFile(answerRel)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read %s: %w", answerRel, err)
		}
		return nil, fmt.Errorf("read %s: %w", answerRel, err)
	}
	answer, err := qa.ParseAnswer(answerRaw)
	if err != nil {
		return nil, err
	}
	contextRel := path.Join(qaRoot, "context-pack.json")
	contextRaw, err := ws.ReadFile(contextRel)
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
	payload["answer_status"] = "available"
	payload["answer_digest"] = proposaldraft.AnswerDigest(answerRaw)
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
