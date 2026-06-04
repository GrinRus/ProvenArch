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
	workspacePath, prepareErr := prepareOnboardingWorkspace(payload.Path, payload.Create)
	if prepareErr != nil {
		writeError(writer, prepareErr.status, prepareErr.code, prepareErr.message)
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

	return map[string]any{
		"ok":                 true,
		"launcher_mode":      launcherMode,
		"workspace_selected": workspaceSelected,
		"workspace_ready":    workspaceReady,
		"workspace":          workspacePath,
		"manifest_present":   workspaceReady,
		"runtime": map[string]any{
			"selected":         runtimeSelected,
			"runtime":          runtimeConfig.Mode,
			"runtime_provider": string(runtimeConfig.Provider),
			"provider_source":  string(runtimeConfig.ProviderSource),
		},
		"can_enter_console": workspaceReady && runtimeSelected,
	}
}

func onboardingWorkspaceLayoutDirectories() []string {
	return []string{
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
	}
}

type onboardingPathError struct {
	status  int
	code    string
	message string
}

func prepareOnboardingWorkspace(rawPath string, create bool) (string, *onboardingPathError) {
	workspacePath, pathErr := normalizeOnboardingWorkspacePath(rawPath)
	if pathErr != nil {
		return "", pathErr
	}

	if create {
		// codeql[go/path-injection] Launcher workspace paths are normalized and constrained to user-home or temp roots above.
		if err := os.MkdirAll(workspacePath, 0o755); err != nil {
			return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_create_failed", message: fmt.Sprintf("create workspace directory: %v", err)}
		}
	} else {
		// codeql[go/path-injection] Launcher workspace paths are normalized and constrained to user-home or temp roots above.
		info, err := os.Stat(workspacePath)
		if err != nil {
			return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_open_failed", message: fmt.Sprintf("stat workspace: %v", err)}
		}
		if !info.IsDir() {
			return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_not_directory", message: "workspace path must point to a directory"}
		}
	}

	gitDir := filepath.Join(workspacePath, ".git")
	// codeql[go/path-injection] gitDir is derived from a normalized launcher workspace path constrained to user-home or temp roots.
	_, err := os.Stat(gitDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_prepare_failed", message: fmt.Sprintf("workspace.git.init.stat_failed: %v", err)}
		}
		if _, err := exec.LookPath("git"); err != nil {
			return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_prepare_failed", message: fmt.Sprintf("workspace.git.init.git_required: install git and ensure it is available in PATH: %v", err)}
		}
		cmd := exec.CommandContext(context.Background(), "git", "init")
		cmd.Dir = workspacePath
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_prepare_failed", message: fmt.Sprintf("workspace.git.init.failed: git init failed: %v: %s", err, strings.TrimSpace(string(output)))}
		}
	}

	for _, rel := range onboardingWorkspaceLayoutDirectories() {
		// codeql[go/path-injection] Layout directories are fixed literals joined under the normalized launcher workspace root.
		if err := os.MkdirAll(filepath.Join(workspacePath, rel), 0o755); err != nil {
			return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_prepare_failed", message: fmt.Sprintf("create layout directory %q: %v", rel, err)}
		}
	}
	return workspacePath, nil
}

func normalizeOnboardingWorkspacePath(rawPath string) (string, *onboardingPathError) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_path_required", message: "workspace path is required"}
	}
	if strings.ContainsRune(trimmed, '\x00') {
		return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_path_invalid", message: "workspace path must not contain NUL bytes"}
	}
	if hasParentPathSegment(trimmed) {
		return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_path_traversal", message: "workspace path must not contain '..' path segments"}
	}
	cleaned := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleaned) {
		return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_path_not_absolute", message: "workspace path must be absolute"}
	}
	if isFilesystemRoot(cleaned) {
		return "", &onboardingPathError{status: http.StatusBadRequest, code: "workspace_path_invalid", message: "workspace path must point to a dedicated workspace directory"}
	}
	for _, safeRoot := range onboardingWorkspaceAllowedRoots() {
		safeRoot = filepath.Clean(safeRoot)
		safePrefix := safeRoot + string(filepath.Separator)
		if cleaned != safeRoot && !strings.HasPrefix(cleaned, safePrefix) {
			continue
		}
		rel, err := filepath.Rel(safeRoot, cleaned)
		if err != nil || rel == "." {
			continue
		}
		absPath, err := filepath.Abs(filepath.Join(safeRoot, rel))
		if err != nil {
			continue
		}
		if absPath == safeRoot || strings.HasPrefix(absPath, safePrefix) {
			return absPath, nil
		}
	}
	return "", &onboardingPathError{
		status:  http.StatusBadRequest,
		code:    "workspace_path_outside_allowed_roots",
		message: fmt.Sprintf("workspace path must be under an allowed root: %s", strings.Join(onboardingWorkspaceAllowedRoots(), ", ")),
	}
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
	addRoot := func(root string) {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "" || !filepath.IsAbs(root) {
			return
		}
		for _, existing := range roots {
			if existing == root {
				return
			}
		}
		roots = append(roots, root)
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		addRoot(home)
	}
	if tmp := strings.TrimSpace(os.TempDir()); tmp != "" {
		addRoot(tmp)
	}
	addRoot("/tmp")
	for _, root := range append([]string(nil), roots...) {
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			addRoot(resolved)
		}
	}
	return roots
}

func isFilesystemRoot(pathValue string) bool {
	cleaned := filepath.Clean(pathValue)
	parent := filepath.Dir(cleaned)
	return cleaned == parent
}
