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

	dirs := newOrderedExistingDirs(4)

	for _, path := range task.ReadContextRoots {
		dirs.add(path)
	}
	if dirs.len() > 0 {
		return dirs.values()
	}

	workspace := strings.TrimSpace(task.Workspace)
	if workspace == "" {
		return nil
	}

	dirs.add(workspace)
	addResolvedRepoScopeDirectories(dirs.add, workspace, headlessRepoScopeFilter(task))

	return dirs.values()
}

func resolveCollectHeadlessIncludeDirectories(task Task) []string {
	dirs := newOrderedExistingDirs(4)

	dirs.add(task.WriteRoot)
	dirs.add(task.DraftFinalRoot)
	for _, path := range task.ReadContextRoots {
		dirs.add(path)
	}

	workspace := strings.TrimSpace(task.Workspace)
	if workspace == "" {
		return dirs.values()
	}
	addResolvedRepoScopeDirectories(dirs.add, workspace, headlessRepoScopeFilter(task))
	return dirs.values()
}

// ResolveHeadlessCollectRepairIncludeDirectories returns the narrow read scope
// for manifest-only collect repair. The repair prompt should see the current
// shard write root plus repository evidence, but not the broader ACP workspace
// where sibling shard manifests and reports/taskruns history can be mistaken
// for schema examples.
func ResolveHeadlessCollectRepairIncludeDirectories(task Task) []string {
	dirs := newOrderedExistingDirs(4)

	dirs.add(task.WriteRoot)
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
		dirs.add(cleaned)
	}
	if workspace != "." {
		addResolvedRepoScopeDirectories(dirs.add, workspace, headlessRepoScopeFilter(task))
	}
	return dirs.values()
}

// ResolveHeadlessValidatorRepairIncludeDirectories returns the narrow read
// scope for verdict-only recovery: current write root plus staged final
// evidence. Repository source roots are intentionally omitted because the
// validator contract should repair/author only from assembled staged artifacts.
func ResolveHeadlessValidatorRepairIncludeDirectories(task Task) []string {
	dirs := newOrderedExistingDirs(4)

	dirs.add(task.WriteRoot)
	for _, path := range task.ReadContextRoots {
		if isStagedFinalRuntimeRoot(path) {
			dirs.add(path)
		}
	}
	return dirs.values()
}

func isStagedFinalRuntimeRoot(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	slash := filepath.ToSlash(filepath.Clean(path))
	return strings.HasSuffix(slash, "/staging/final") || strings.Contains(slash, "/staging/final/")
}

// ResolveHeadlessDraftRepairIncludeDirectories returns the draft recovery
// scope: current write/draft roots plus staged context and repository evidence.
func ResolveHeadlessDraftRepairIncludeDirectories(task Task) []string {
	dirs := newOrderedExistingDirs(6)

	dirs.add(task.WriteRoot)
	dirs.add(task.DraftFinalRoot)
	for _, path := range task.ReadContextRoots {
		dirs.add(path)
	}
	workspace := strings.TrimSpace(task.Workspace)
	if workspace != "" {
		addResolvedRepoScopeDirectories(dirs.add, filepath.Clean(workspace), headlessRepoScopeFilter(task))
	}
	return dirs.values()
}

// ResolveHeadlessDraftEnrichmentIncludeDirectories returns a bounded recovery
// scope for scaffold-only draft enrichment. Unlike first-pass draft repair, it
// must not expose the whole headless workspace or source repo for large
// medium/full runs; providers should enrich from current draft files and the
// current taskrun staging evidence without reading every repository file.
func ResolveHeadlessDraftEnrichmentIncludeDirectories(task Task) []string {
	dirs := newOrderedExistingDirs(6)

	dirs.add(task.WriteRoot)
	dirs.add(task.DraftFinalRoot)

	taskrunRoot := taskRunRootFromRuntimeArtifactPath(task.DraftFinalRoot)
	if taskrunRoot == "" {
		taskrunRoot = taskRunRootFromRuntimeArtifactPath(task.WriteRoot)
	}
	if taskrunRoot != "" {
		dirs.add(filepath.Join(taskrunRoot, "staging", "shards"))
		dirs.add(filepath.Join(taskrunRoot, "staging", "final"))
	}

	for _, root := range task.ReadContextRoots {
		if shouldIncludeDraftEnrichmentReadRoot(root, taskrunRoot) {
			dirs.add(root)
		}
	}

	if strings.TrimSpace(task.StepID) == "init.step0.constitution" {
		workspace := strings.TrimSpace(task.Workspace)
		if workspace != "" {
			addResolvedRepoScopeDirectories(dirs.add, filepath.Clean(workspace), headlessRepoScopeFilter(task))
		}
	}
	return dirs.values()
}

func taskRunRootFromRuntimeArtifactPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	slash := filepath.ToSlash(filepath.Clean(path))
	for _, marker := range []string{"/staging/drafts/", "/staging/shards/", "/staging/final", "/runtime/"} {
		if idx := strings.Index(slash, marker); idx > 0 {
			return filepath.FromSlash(slash[:idx])
		}
	}
	return ""
}

func shouldIncludeDraftEnrichmentReadRoot(root string, taskrunRoot string) bool {
	root = strings.TrimSpace(root)
	if root == "" || taskrunRoot == "" {
		return false
	}
	cleanRoot := filepath.Clean(root)
	cleanTaskrun := filepath.Clean(taskrunRoot)
	rel, err := filepath.Rel(cleanTaskrun, cleanRoot)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false
	}
	slash := filepath.ToSlash(rel)
	return slash == "staging/shards" ||
		strings.HasPrefix(slash, "staging/shards/") ||
		slash == "staging/final" ||
		strings.HasPrefix(slash, "staging/final/") ||
		slash == "staging/drafts" ||
		strings.HasPrefix(slash, "staging/drafts/")
}

type orderedExistingDirs struct {
	valuesList []string
	seen       map[string]struct{}
}

func newOrderedExistingDirs(capacity int) *orderedExistingDirs {
	return &orderedExistingDirs{
		valuesList: make([]string, 0, capacity),
		seen:       map[string]struct{}{},
	}
}

func (dirs *orderedExistingDirs) add(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	cleaned := filepath.Clean(path)
	info, err := os.Stat(cleaned)
	if err != nil || !info.IsDir() {
		return
	}
	if _, ok := dirs.seen[cleaned]; ok {
		return
	}
	dirs.seen[cleaned] = struct{}{}
	dirs.valuesList = append(dirs.valuesList, cleaned)
}

func (dirs *orderedExistingDirs) len() int {
	if dirs == nil {
		return 0
	}
	return len(dirs.valuesList)
}

func (dirs *orderedExistingDirs) values() []string {
	if dirs == nil {
		return nil
	}
	return dirs.valuesList
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
