package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const maxRepositoryEvidenceReadBytes = 2 * 1024 * 1024

type repositoryEvidenceResponse struct {
	Repo    string `json:"repo"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// handleRepositoryEvidence reads a bounded, read-only file from the configured
// source checkout identified by the logical repository scope in an evidence ref.
// It is deliberately separate from /api/artifacts: workspace artifacts and
// repository evidence have different authorities and must not be conflated.
func (s *Server) handleRepositoryEvidence(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	repoName := strings.TrimSpace(request.URL.Query().Get("repo"))
	relPath, err := normalizeRepositoryEvidencePath(request.URL.Query().Get("path"))
	if repoName == "" {
		writeError(writer, http.StatusBadRequest, "repository_required", "repo query parameter is required")
		return
	}
	if err != nil {
		writeError(writer, http.StatusBadRequest, "path_invalid", err.Error())
		return
	}

	root, err := resolveRepositoryEvidenceRoot(request, s.getWorkspace(), repoName)
	if err != nil {
		status := http.StatusNotFound
		code := "repository_not_found"
		if errors.Is(err, errRepositorySourceUnavailable) {
			code = "repository_source_unavailable"
		}
		writeError(writer, status, code, err.Error())
		return
	}
	absolute := filepath.Join(root, filepath.FromSlash(relPath))
	content, err := os.ReadFile(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(writer, http.StatusNotFound, "evidence_not_found", fmt.Sprintf("repository evidence %s:%s was not found", repoName, relPath))
			return
		}
		writeError(writer, http.StatusNotFound, "evidence_unreadable", err.Error())
		return
	}
	if int64(len(content)) > maxRepositoryEvidenceReadBytes {
		writeError(writer, http.StatusRequestEntityTooLarge, "evidence_too_large", "repository evidence exceeds the 2 MiB viewer read budget")
		return
	}
	writeJSON(writer, http.StatusOK, repositoryEvidenceResponse{Repo: repoName, Path: relPath, Content: string(content)})
}

var errRepositorySourceUnavailable = errors.New("configured repository source is not available locally")

func resolveRepositoryEvidenceRoot(request *http.Request, ws workspace.Root, repoName string) (string, error) {
	for _, repo := range ws.Manifest.Repos {
		if strings.TrimSpace(repo.Name) != repoName {
			continue
		}
		if strings.TrimSpace(repo.Path) != "" {
			root := strings.TrimSpace(repo.Path)
			if !filepath.IsAbs(root) {
				root = filepath.Join(ws.Path, root)
			}
			root = filepath.Clean(root)
			info, err := os.Stat(root)
			if err != nil || !info.IsDir() {
				return "", errRepositorySourceUnavailable
			}
			return root, nil
		}
		resolved, _ := ws.ResolveRepoSources(request.Context(), workspace.ResolveOptions{FetchGit: false})
		for _, candidate := range resolved {
			if candidate.Name == repoName && candidate.Path != "" {
				info, statErr := os.Stat(candidate.Path)
				if statErr == nil && info.IsDir() {
					return filepath.Clean(candidate.Path), nil
				}
			}
		}
		return "", errRepositorySourceUnavailable
	}
	return "", fmt.Errorf("repository %q is not configured", repoName)
}

func normalizeRepositoryEvidencePath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return "", errors.New("path query parameter is required")
	}
	if strings.HasPrefix(value, "/") || filepath.IsAbs(value) || strings.Contains(value, "\x00") {
		return "", errors.New("repository evidence path must be relative")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("repository evidence path must stay within the repository")
	}
	return clean, nil
}
