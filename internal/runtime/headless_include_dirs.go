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
	if IsCollectStep(task.StepID) {
		return resolveCollectHeadlessIncludeDirectories(task)
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

	for _, path := range task.ReadContextRoots {
		addDir(path)
	}
	if len(dirs) > 0 {
		return dirs
	}

	workspace := strings.TrimSpace(task.Workspace)
	if workspace == "" {
		return nil
	}

	addDir(workspace)
	addResolvedRepoScopeDirectories(addDir, workspace, headlessRepoScopeFilter(task))

	return dirs
}

func resolveCollectHeadlessIncludeDirectories(task Task) []string {
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

	addDir(task.WriteRoot)
	for _, path := range task.ReadContextRoots {
		addDir(path)
	}

	workspace := strings.TrimSpace(task.Workspace)
	if workspace == "" {
		return dirs
	}
	addResolvedRepoScopeDirectories(addDir, workspace, headlessRepoScopeFilter(task))
	return dirs
}

// ResolveHeadlessCollectRepairIncludeDirectories returns the narrow read scope
// for manifest-only collect repair. The repair prompt should see the current
// shard write root plus repository evidence, but not the broader ACP workspace
// where sibling shard manifests and reports/taskruns history can be mistaken
// for schema examples.
func ResolveHeadlessCollectRepairIncludeDirectories(task Task) []string {
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

	addDir(task.WriteRoot)
	workspace := filepath.Clean(strings.TrimSpace(task.Workspace))
	for _, root := range task.ReadContextRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		cleaned := filepath.Clean(root)
		if workspace != "." && cleaned == workspace {
			continue
		}
		if strings.Contains(filepath.ToSlash(cleaned), "/reports/taskruns/") && cleaned != filepath.Clean(strings.TrimSpace(task.WriteRoot)) {
			continue
		}
		addDir(cleaned)
	}
	if workspace != "." {
		addResolvedRepoScopeDirectories(addDir, workspace, headlessRepoScopeFilter(task))
	}
	return dirs
}

func addResolvedRepoScopeDirectories(addDir func(string), workspace string, scopeFilter repoScopeFilter) {
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
