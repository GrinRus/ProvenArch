package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	workspacecfg "github.com/GrinRus/ProvenArch/internal/workspace"
)

type headlessWorkspaceValidateReport struct {
	ResolvedRepos []headlessResolvedRepo `json:"resolved_repos"`
}

type headlessResolvedRepo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ResolveHeadlessIncludeDirectories returns the workspace plus any resolved
// repository directories relevant to the current task, so headless providers
// can inspect source evidence instead of only ACP-generated workspace files.
func ResolveHeadlessIncludeDirectories(task Task) []string {
	workspace := strings.TrimSpace(task.Workspace)
	if workspace == "" {
		return nil
	}

	dirs := make([]string, 0, 4)
	seen := map[string]struct{}{}
	addDir := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		cleaned := filepath.Clean(path)
		info, err := os.Stat(cleaned)
		if err != nil || !info.IsDir() {
			return
		}
		if _, ok := seen[cleaned]; ok {
			return
		}
		seen[cleaned] = struct{}{}
		dirs = append(dirs, cleaned)
	}

	addDir(workspace)
	scopeFilter := headlessRepoScopeFilter(task)

	manifestPath := filepath.Join(workspace, "workspace.yaml")
	if manifest, err := workspacecfg.LoadManifest(manifestPath); err == nil {
		for _, repo := range manifest.Repos {
			if !scopeFilter.allows(repo.Name) {
				continue
			}
			addDir(repo.Path)
		}
	}

	validatePath := filepath.Join(filepath.Dir(workspace), "workspace-validate.json")
	if raw, err := os.ReadFile(validatePath); err == nil {
		var report headlessWorkspaceValidateReport
		if json.Unmarshal(raw, &report) == nil {
			for _, repo := range report.ResolvedRepos {
				if !scopeFilter.allows(repo.Name) {
					continue
				}
				addDir(repo.Path)
			}
		}
	}

	return dirs
}

type repoScopeFilter map[string]struct{}

func headlessRepoScopeFilter(task Task) repoScopeFilter {
	filter := repoScopeFilter{}
	for _, scope := range task.RepoScopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		filter[scope] = struct{}{}
	}
	if scope := strings.TrimSpace(task.RepoScope); scope != "" {
		filter[scope] = struct{}{}
	}
	return filter
}

func (f repoScopeFilter) allows(scope string) bool {
	if len(f) == 0 {
		return true
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false
	}
	_, ok := f[scope]
	return ok
}
