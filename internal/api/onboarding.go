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
	workspacePath, pathErr := normalizeOnboardingWorkspacePath(payload.Path)
	if pathErr != nil {
		writeError(writer, http.StatusBadRequest, pathErr.code, pathErr.message)
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

type onboardingPathError struct {
	code    string
	message string
}

func normalizeOnboardingWorkspacePath(rawPath string) (string, *onboardingPathError) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", &onboardingPathError{code: "workspace_path_required", message: "workspace path is required"}
	}
	if strings.ContainsRune(trimmed, '\x00') {
		return "", &onboardingPathError{code: "workspace_path_invalid", message: "workspace path must not contain NUL bytes"}
	}
	if hasParentPathSegment(trimmed) {
		return "", &onboardingPathError{code: "workspace_path_traversal", message: "workspace path must not contain '..' path segments"}
	}
	cleaned := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleaned) {
		return "", &onboardingPathError{code: "workspace_path_not_absolute", message: "workspace path must be absolute"}
	}
	if isFilesystemRoot(cleaned) {
		return "", &onboardingPathError{code: "workspace_path_invalid", message: "workspace path must point to a dedicated workspace directory"}
	}
	if !isUnderAnyRoot(cleaned, onboardingWorkspaceAllowedRoots()) {
		return "", &onboardingPathError{code: "workspace_path_outside_allowed_roots", message: "workspace path must be under the current user home directory or system temp directory"}
	}
	return cleaned, nil
}

func hasParentPathSegment(rawPath string) bool {
	for _, part := range strings.FieldsFunc(rawPath, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if part == ".." {
			return true
		}
	}
	return false
}

func onboardingWorkspaceAllowedRoots() []string {
	roots := []string{}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		roots = append(roots, filepath.Clean(home))
	}
	if tmp := strings.TrimSpace(os.TempDir()); tmp != "" {
		roots = append(roots, filepath.Clean(tmp))
	}
	return roots
}

func isUnderAnyRoot(candidate string, roots []string) bool {
	for _, root := range roots {
		if root == "" {
			continue
		}
		if isPathWithinRoot(candidate, root) {
			return true
		}
	}
	return false
}

func isPathWithinRoot(candidate, root string) bool {
	candidate = filepath.Clean(candidate)
	root = filepath.Clean(root)
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func isFilesystemRoot(pathValue string) bool {
	cleaned := filepath.Clean(pathValue)
	parent := filepath.Dir(cleaned)
	return cleaned == parent
}
