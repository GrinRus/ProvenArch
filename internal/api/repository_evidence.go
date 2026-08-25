package api

import (
	"errors"
	"fmt"
	"io"
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
	if err := validateRepositoryEvidencePath(root, relPath); err != nil {
		if errors.Is(err, errRepositoryEvidencePathOutsideRoot) {
			writeError(writer, http.StatusBadRequest, "path_invalid", err.Error())
			return
		}
		if errors.Is(err, errRepositoryEvidencePathNotFound) {
			writeError(writer, http.StatusNotFound, "evidence_not_found", fmt.Sprintf("repository evidence %s:%s was not found", repoName, relPath))
			return
		}
		writeError(writer, http.StatusNotFound, "evidence_unreadable", err.Error())
		return
	}
	content, err := readRepositoryEvidenceFile(root, relPath)
	if err != nil {
		if errors.Is(err, errRepositoryEvidencePathOutsideRoot) {
			writeError(writer, http.StatusBadRequest, "path_invalid", err.Error())
			return
		}
		if errors.Is(err, errRepositoryEvidencePathNotFound) || errors.Is(err, os.ErrNotExist) {
			writeError(writer, http.StatusNotFound, "evidence_not_found", fmt.Sprintf("repository evidence %s:%s was not found", repoName, relPath))
			return
		}
		if errors.Is(err, errRepositoryEvidenceTooLarge) {
			writeError(writer, http.StatusRequestEntityTooLarge, "evidence_too_large", "repository evidence exceeds the 2 MiB viewer read budget")
			return
		}
		writeError(writer, http.StatusNotFound, "evidence_unreadable", err.Error())
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

var errRepositoryEvidencePathOutsideRoot = errors.New("repository evidence path must stay within the repository")
var errRepositoryEvidencePathNotFound = errors.New("repository evidence path was not found")
var errRepositoryEvidenceTooLarge = errors.New("repository evidence exceeds the read budget")

func validateRepositoryEvidencePath(root string, relPath string) error {
	rootAbsolute, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	rootCanonical, err := filepath.EvalSymlinks(rootAbsolute)
	if err != nil {
		return fmt.Errorf("resolve repository root symlinks: %w", err)
	}

	candidate := filepath.Join(rootAbsolute, filepath.FromSlash(relPath))
	candidateAbsolute, err := filepath.Abs(candidate)
	if err != nil {
		return fmt.Errorf("resolve repository evidence path: %w", err)
	}
	if !repositoryEvidencePathWithinRoot(rootAbsolute, candidateAbsolute) {
		return errRepositoryEvidencePathOutsideRoot
	}

	candidateCanonical, err := filepath.EvalSymlinks(candidateAbsolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errRepositoryEvidencePathNotFound
		}
		return fmt.Errorf("resolve repository evidence path symlinks: %w", err)
	}
	if !repositoryEvidencePathWithinRoot(rootCanonical, candidateCanonical) {
		return errRepositoryEvidencePathOutsideRoot
	}
	return nil
}

func readRepositoryEvidenceFile(root string, relPath string) ([]byte, error) {
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open repository root: %w", err)
	}
	defer rootHandle.Close()

	file, err := rootHandle.Open(filepath.FromSlash(relPath))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxRepositoryEvidenceReadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read repository evidence: %w", err)
	}
	if int64(len(content)) > maxRepositoryEvidenceReadBytes {
		return nil, errRepositoryEvidenceTooLarge
	}
	return content, nil
}

func repositoryEvidencePathWithinRoot(root string, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil || relative == "." || relative == ".." {
		return false
	}
	return !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
