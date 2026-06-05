package api

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const onboardingPathSuggestionLimit = 40

type onboardingPathSuggestion struct {
	Path   string `json:"path"`
	Label  string `json:"label"`
	Exists bool   `json:"exists"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
}

func (s *Server) handleOnboardingPathSuggestions(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	query := strings.TrimSpace(request.URL.Query().Get("query"))
	kind := strings.TrimSpace(request.URL.Query().Get("kind"))
	if kind != "workspace" && kind != "repo" {
		writeError(writer, http.StatusBadRequest, "path_suggestion_kind_invalid", "path suggestion kind must be workspace or repo")
		return
	}
	if err := validatePathSuggestionQuery(query); err != nil {
		writeError(writer, err.status, err.code, err.message)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":    true,
		"kind":  kind,
		"query": query,
		"items": s.onboardingPathSuggestions(kind, query),
	})
}

func validatePathSuggestionQuery(query string) *onboardingPathError {
	if strings.ContainsRune(query, '\x00') {
		return &onboardingPathError{status: http.StatusBadRequest, code: "path_suggestion_query_invalid", message: "path suggestion query must not contain NUL bytes"}
	}
	if hasParentPathSegment(query) {
		return &onboardingPathError{status: http.StatusBadRequest, code: "path_suggestion_query_traversal", message: "path suggestion query must not contain '..' path segments"}
	}
	if filepath.IsAbs(query) && isFilesystemRoot(filepath.Clean(query)) {
		return &onboardingPathError{status: http.StatusBadRequest, code: "path_suggestion_query_root", message: "path suggestion query must not be a filesystem root"}
	}
	return nil
}

func (s *Server) onboardingPathSuggestions(kind string, query string) []onboardingPathSuggestion {
	builder := pathSuggestionBuilder{
		query:      strings.TrimSpace(query),
		queryLower: strings.ToLower(strings.TrimSpace(query)),
		seen:       map[string]bool{},
	}

	if kind == "workspace" {
		builder.addWorkspaceSuggestions()
		return builder.items
	}

	workspacePath := s.getWorkspacePath()
	if workspacePath != "" {
		builder.addManifestRepoSuggestions(workspacePath)
	}
	builder.addRepoSuggestions()
	return builder.items
}

type pathSuggestionBuilder struct {
	query      string
	queryLower string
	seen       map[string]bool
	items      []onboardingPathSuggestion
}

func (b *pathSuggestionBuilder) addWorkspaceSuggestions() {
	for _, recent := range loadOnboardingRecentWorkspaces() {
		b.addWorkspaceCandidate(recent.Path, "recent")
	}
	if b.query != "" && filepath.IsAbs(b.query) {
		b.addWorkspaceCandidate(b.query, "query")
		b.addDirectoryChildren(filepath.Dir(filepath.Clean(b.query)), "discovered", true)
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		b.addWorkspaceCandidate(filepath.Join(home, "acp-workspaces"), "common")
		b.addWorkspaceCandidate(home, "common")
		b.addDirectoryChildren(filepath.Join(home, "acp-workspaces"), "discovered", true)
	}
	if tempRoot := strings.TrimSpace(os.TempDir()); tempRoot != "" {
		b.addWorkspaceCandidate(tempRoot, "common")
		b.addDirectoryChildren(tempRoot, "discovered", true)
	}
	b.addWorkspaceCandidate("/tmp", "common")
	b.addDirectoryChildren("/tmp", "discovered", true)
}

func (b *pathSuggestionBuilder) addRepoSuggestions() {
	if b.query != "" && filepath.IsAbs(b.query) {
		b.addRepoCandidate(b.query, "query")
		queryPath := filepath.Clean(b.query)
		if info, err := os.Stat(queryPath); err == nil && info.IsDir() {
			b.addDirectoryChildren(queryPath, "discovered", false)
		} else {
			b.addDirectoryChildren(filepath.Dir(queryPath), "discovered", false)
		}
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		for _, rel := range []string{"src", "Projects", "code"} {
			root := filepath.Join(home, rel)
			b.addRepoCandidate(root, "common")
			b.addDirectoryChildren(root, "discovered", false)
		}
	}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		b.addRepoCandidate(cwd, "cwd")
	}
}

func (b *pathSuggestionBuilder) addManifestRepoSuggestions(workspacePath string) {
	manifestPath := filepath.Join(workspacePath, workspace.ManifestFileName)
	manifest, err := workspace.LoadManifest(manifestPath)
	if err != nil {
		return
	}
	for _, repo := range manifest.Repos {
		if strings.TrimSpace(repo.Path) == "" {
			continue
		}
		repoPath := strings.TrimSpace(repo.Path)
		if !filepath.IsAbs(repoPath) {
			repoPath = filepath.Join(workspacePath, repoPath)
		}
		b.addRepoCandidate(repoPath, "manifest")
	}
}

func (b *pathSuggestionBuilder) addDirectoryChildren(root string, source string, workspaceKind bool) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || !filepath.IsAbs(root) || isFilesystemRoot(root) {
		return
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return
	}
	if workspaceKind {
		if _, pathErr := normalizeOnboardingWorkspacePath(filepath.Join(root, "candidate")); pathErr != nil {
			return
		}
	} else if !isRepoSuggestionPathAllowed(root) {
		return
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	sort.SliceStable(entries, func(left, right int) bool {
		return strings.ToLower(entries[left].Name()) < strings.ToLower(entries[right].Name())
	})
	for _, entry := range entries {
		if len(b.items) >= onboardingPathSuggestionLimit {
			return
		}
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		pathValue := filepath.Join(root, name)
		if workspaceKind {
			b.addWorkspaceCandidate(pathValue, source)
		} else {
			b.addRepoCandidate(pathValue, source)
		}
	}
}

func (b *pathSuggestionBuilder) addWorkspaceCandidate(pathValue string, source string) {
	normalized, err := normalizeOnboardingWorkspacePath(pathValue)
	if err != nil {
		return
	}
	b.addCandidate(normalized, "workspace", source)
}

func (b *pathSuggestionBuilder) addRepoCandidate(pathValue string, source string) {
	normalized, ok := normalizeRepoSuggestionPath(pathValue)
	if !ok {
		return
	}
	kind := "directory"
	if isGitRepositoryLike(normalized) {
		kind = "git_repo"
	}
	b.addCandidate(normalized, kind, source)
}

func (b *pathSuggestionBuilder) addCandidate(pathValue string, kind string, source string) {
	pathValue = filepath.Clean(pathValue)
	if len(b.items) >= onboardingPathSuggestionLimit || b.seen[pathValue] {
		return
	}
	label := pathSuggestionLabel(pathValue)
	if b.queryLower != "" && !filepath.IsAbs(b.query) {
		haystack := strings.ToLower(pathValue + " " + label)
		if !strings.Contains(haystack, b.queryLower) {
			return
		}
	}
	if b.queryLower != "" && filepath.IsAbs(b.query) {
		queryPath := strings.ToLower(filepath.Clean(b.query))
		pathLower := strings.ToLower(pathValue)
		if !strings.Contains(pathLower, queryPath) && !strings.Contains(pathLower, b.queryLower) {
			return
		}
	}
	b.seen[pathValue] = true
	info, err := os.Stat(pathValue)
	b.items = append(b.items, onboardingPathSuggestion{
		Path:   pathValue,
		Label:  label,
		Exists: err == nil && info.IsDir(),
		Kind:   kind,
		Source: source,
	})
}

func pathSuggestionLabel(pathValue string) string {
	cleaned := filepath.Clean(pathValue)
	base := filepath.Base(cleaned)
	if base == "." || base == string(filepath.Separator) {
		return cleaned
	}
	return base
}

func normalizeRepoSuggestionPath(pathValue string) (string, bool) {
	trimmed := strings.TrimSpace(pathValue)
	if trimmed == "" || strings.ContainsRune(trimmed, '\x00') || hasParentPathSegment(trimmed) {
		return "", false
	}
	cleaned := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleaned) || isFilesystemRoot(cleaned) {
		return "", false
	}
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return "", false
	}
	if !isRepoSuggestionPathAllowed(absPath) {
		return "", false
	}
	return absPath, true
}

func isRepoSuggestionPathAllowed(pathValue string) bool {
	return isPathUnderAllowedRoots(pathValue, repoSuggestionAllowedRoots(), true)
}

func repoSuggestionAllowedRoots() []string {
	roots := onboardingWorkspaceAllowedRoots()
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		roots = append(roots, cwd)
	}
	return dedupeCleanAbsolutePaths(roots)
}

func dedupeCleanAbsolutePaths(paths []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(paths))
	for _, pathValue := range paths {
		cleaned := filepath.Clean(strings.TrimSpace(pathValue))
		if cleaned == "" || !filepath.IsAbs(cleaned) || seen[cleaned] {
			continue
		}
		seen[cleaned] = true
		result = append(result, cleaned)
	}
	return result
}

func isPathUnderAllowedRoots(pathValue string, roots []string, allowRoot bool) bool {
	pathValue = filepath.Clean(pathValue)
	evaluatedPath, pathEvaluated := evalPathForSuggestion(pathValue)
	for _, root := range roots {
		root = filepath.Clean(root)
		if root == "" || !filepath.IsAbs(root) || isFilesystemRoot(root) {
			continue
		}
		if !pathHasRoot(pathValue, root, allowRoot) {
			continue
		}
		if pathEvaluated {
			evaluatedRoot, rootEvaluated := evalPathForSuggestion(root)
			if rootEvaluated && !pathHasRoot(evaluatedPath, evaluatedRoot, allowRoot) {
				continue
			}
		}
		return true
	}
	return false
}

func evalPathForSuggestion(pathValue string) (string, bool) {
	evaluated, err := filepath.EvalSymlinks(pathValue)
	if err != nil {
		return pathValue, false
	}
	return filepath.Clean(evaluated), true
}

func pathHasRoot(pathValue string, root string, allowRoot bool) bool {
	pathValue = filepath.Clean(pathValue)
	root = filepath.Clean(root)
	if pathValue == root {
		return allowRoot
	}
	prefix := root + string(filepath.Separator)
	return strings.HasPrefix(pathValue, prefix)
}

func isGitRepositoryLike(pathValue string) bool {
	gitPath := filepath.Join(pathValue, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}
