package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func (s *Server) handleOnboardingStatus(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	writeJSON(writer, http.StatusOK, s.onboardingStatusPayload())
}

func (s *Server) handleOnboardingWorkspace(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload struct {
		Path   string `json:"path"`
		Create bool   `json:"create"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	workspacePath := strings.TrimSpace(payload.Path)
	if workspacePath == "" {
		writeError(writer, http.StatusBadRequest, "workspace_path_required", "workspace path is required")
		return
	}
	if !filepath.IsAbs(workspacePath) {
		writeError(writer, http.StatusBadRequest, "workspace_path_not_absolute", "workspace path must be absolute")
		return
	}

	if payload.Create {
		if err := os.MkdirAll(workspacePath, 0o755); err != nil {
			writeError(writer, http.StatusBadRequest, "workspace_create_failed", fmt.Sprintf("create workspace directory: %v", err))
			return
		}
	} else {
		info, err := os.Stat(workspacePath)
		if err != nil {
			writeError(writer, http.StatusBadRequest, "workspace_open_failed", fmt.Sprintf("stat workspace: %v", err))
			return
		}
		if !info.IsDir() {
			writeError(writer, http.StatusBadRequest, "workspace_not_directory", "workspace path must point to a directory")
			return
		}
	}

	if err := ensureDraftWorkspace(workspacePath); err != nil {
		writeError(writer, http.StatusBadRequest, "workspace_prepare_failed", err.Error())
		return
	}

	ws, err := workspace.Open(workspacePath)
	if err != nil {
		if errors.Is(err, workspace.ErrManifestMissing) {
			s.setDraftWorkspace(workspacePath)
			writeJSON(writer, http.StatusOK, s.onboardingStatusPayload())
			return
		}
		writeError(writer, http.StatusBadRequest, "workspace_open_failed", err.Error())
		return
	}
	if err := ws.EnsureLayout(); err != nil {
		writeError(writer, http.StatusBadRequest, "workspace_layout_failed", err.Error())
		return
	}
	if err := ws.EnsureBaselineBundle(); err != nil {
		writeError(writer, http.StatusBadRequest, "workspace_baseline_failed", err.Error())
		return
	}
	s.attachWorkspace(ws)
	if service := s.getService(); service != nil {
		service.ReconcileStaleRunsAfterRestart()
	}
	writeJSON(writer, http.StatusOK, s.onboardingStatusPayload())
}

func (s *Server) handleOnboardingRuntime(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload struct {
		Runtime         string `json:"runtime"`
		RuntimeProvider string `json:"runtime_provider"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	mode, err := acpruntime.NormalizeMode(payload.Runtime)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "runtime_invalid", err.Error())
		return
	}
	provider, providerSource, err := acpruntime.ResolveProviderWithSource(payload.RuntimeProvider)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "runtime_provider_invalid", err.Error())
		return
	}
	s.setRuntimeConfig(ServerRuntimeConfig{
		Mode:           mode,
		Provider:       provider,
		ProviderSource: providerSource,
	})
	writeJSON(writer, http.StatusOK, s.onboardingStatusPayload())
}

func (s *Server) onboardingStatusPayload() map[string]any {
	s.mu.RLock()
	workspacePath := s.workspacePath
	workspaceSelected := s.workspaceSelected
	workspaceReady := workspaceSelected && s.workspace.Path != "" && s.service != nil
	runtimeConfig := s.runtimeConfig
	runtimeSelected := s.runtimeSelected
	launcherMode := s.launcherMode
	s.mu.RUnlock()

	manifestPresent := false
	if strings.TrimSpace(workspacePath) != "" {
		if info, err := os.Stat(filepath.Join(workspacePath, workspace.ManifestFileName)); err == nil && !info.IsDir() {
			manifestPresent = true
		}
	}

	return map[string]any{
		"ok":                 true,
		"launcher_mode":      launcherMode,
		"workspace_selected": workspaceSelected,
		"workspace_ready":    workspaceReady,
		"workspace":          workspacePath,
		"manifest_present":   manifestPresent,
		"runtime": map[string]any{
			"selected":         runtimeSelected,
			"runtime":          runtimeConfig.Mode,
			"runtime_provider": string(runtimeConfig.Provider),
			"provider_source":  string(runtimeConfig.ProviderSource),
		},
		"can_enter_console": workspaceReady && runtimeSelected,
	}
}

func ensureDraftWorkspace(workspacePath string) error {
	if err := ensureWorkspaceGitRepository(workspacePath); err != nil {
		return err
	}
	for _, rel := range []string{
		"charter/cards/domains",
		"charter/cards/teams",
		"charter/templates",
		"skills",
		"model/entities",
		"model/edges",
		"reports/as-is/services",
		"reports/findings",
		"reports/coverage",
		"reports/taskruns",
		"reports/agent-outputs/domains",
		"reports/agent-outputs/architect",
		"reports/changelog",
		"proposals",
		"docs/imports",
		"docs/rfcs",
		"docs/meetings",
		"docs/decisions",
	} {
		if err := os.MkdirAll(filepath.Join(workspacePath, rel), 0o755); err != nil {
			return fmt.Errorf("create layout directory %q: %w", rel, err)
		}
	}
	return nil
}

func ensureWorkspaceGitRepository(workspacePath string) error {
	gitDir := filepath.Join(workspacePath, ".git")
	_, err := os.Stat(gitDir)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("workspace.git.init.stat_failed: %w", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("workspace.git.init.git_required: install git and ensure it is available in PATH: %w", err)
	}
	cmd := exec.CommandContext(context.Background(), "git", "init")
	cmd.Dir = workspacePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("workspace.git.init.failed: git init failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
