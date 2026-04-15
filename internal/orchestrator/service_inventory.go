package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

const (
	serviceInventoryLatestSnapshotPath = "reports/taskruns/service-inventory-latest.json"

	serviceLargeFilesThreshold = 500
	serviceLargeBytesThreshold = 8 * 1024 * 1024
	serviceChunkMaxFiles       = 200
	serviceChunkMaxBytes       = 3 * 1024 * 1024
	serviceChunkMaxCount       = 8

	serviceInventoryArtifactVersion = 1
)

type serviceSourceFile struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

type serviceShardPlan struct {
	RepoScope   string   `json:"repo_scope"`
	ServiceID   string   `json:"service_id"`
	ServiceRoot string   `json:"service_root"`
	ShardID     string   `json:"shard_id"`
	SortKey     string   `json:"sort_key"`
	PathScopes  []string `json:"path_scopes"`
	FileCount   int      `json:"file_count"`
	SourceBytes int64    `json:"source_bytes"`
}

type serviceInventoryService struct {
	RepoScope   string              `json:"repo_scope"`
	ServiceID   string              `json:"service_id"`
	ServiceRoot string              `json:"service_root"`
	FileCount   int                 `json:"file_count"`
	SourceBytes int64               `json:"source_bytes"`
	SourceFiles []serviceSourceFile `json:"source_files,omitempty"`
	Shards      []serviceShardPlan  `json:"shards"`
}

type serviceInventoryRemoved struct {
	RepoScope   string `json:"repo_scope"`
	ServiceID   string `json:"service_id"`
	ServiceRoot string `json:"service_root"`
}

type serviceInventoryPlan struct {
	Version         int                        `json:"version"`
	RunID           string                     `json:"run_id"`
	Pipeline        string                     `json:"pipeline"`
	Mode            string                     `json:"mode"`
	GeneratedAt     string                     `json:"generated_at"`
	Services        []serviceInventoryService  `json:"services"`
	SelectedShards  []serviceShardPlan         `json:"selected_shards"`
	RemovedServices []serviceInventoryRemoved  `json:"removed_services,omitempty"`
	Warnings        []string                   `json:"warnings,omitempty"`
	RepoHeads       []serviceInventoryRepoHead `json:"repo_heads,omitempty"`
}

type serviceInventoryRepoHead struct {
	RepoScope string `json:"repo_scope"`
	Head      string `json:"head,omitempty"`
}

type serviceInventorySnapshot struct {
	Version     int                               `json:"version"`
	GeneratedAt string                            `json:"generated_at"`
	RunID       string                            `json:"run_id"`
	Pipeline    string                            `json:"pipeline"`
	RepoHeads   []serviceInventoryRepoHead        `json:"repo_heads"`
	Services    []serviceInventorySnapshotService `json:"services"`
}

type serviceInventorySnapshotService struct {
	RepoScope   string                          `json:"repo_scope"`
	ServiceID   string                          `json:"service_id"`
	ServiceRoot string                          `json:"service_root"`
	FileCount   int                             `json:"file_count"`
	SourceBytes int64                           `json:"source_bytes"`
	Shards      []serviceInventorySnapshotShard `json:"shards"`
}

type serviceInventorySnapshotShard struct {
	ShardID     string   `json:"shard_id"`
	PathScopes  []string `json:"path_scopes"`
	FileCount   int      `json:"file_count"`
	SourceBytes int64    `json:"source_bytes"`
}

func (e *pipelineExecution) runStepServiceInventory(ctx context.Context) error {
	plan, err := e.buildServiceInventoryPlan(ctx)
	if err != nil {
		return err
	}
	e.servicePlan = plan
	for _, warning := range plan.Warnings {
		e.addWarning(fmt.Sprintf("%s: %s", e.stepStatus.CurrentStep, warning))
	}
	if err := e.persistServiceInventoryArtifacts(plan); err != nil {
		return err
	}
	e.logInfo(e.stepStatus.CurrentStep, "", "service inventory prepared", map[string]any{
		"services":         len(plan.Services),
		"selected_shards":  len(plan.SelectedShards),
		"mode":             plan.Mode,
		"removed_services": len(plan.RemovedServices),
	})
	if len(plan.RemovedServices) > 0 {
		questions := make([]contractsQuestion, 0, len(plan.RemovedServices))
		removedFindings := make([]contracts.Finding, 0, len(plan.RemovedServices))
		for _, removed := range plan.RemovedServices {
			qid := fmt.Sprintf("q.service.%s.removed", slugutil.Slugify(removed.ServiceID))
			questions = append(questions, contractsQuestion{
				ID:       qid,
				Text:     fmt.Sprintf("Service %q (repo_scope=%q root=%q) existed in previous snapshot but is missing now", removed.ServiceID, removed.RepoScope, removed.ServiceRoot),
				Priority: "medium",
			})
			removedFindings = append(removedFindings, contracts.Finding{
				ID:          fmt.Sprintf("finding.service.%s.removed", slugutil.Slugify(removed.ServiceID)),
				Severity:    "medium",
				Title:       "Removed service detected",
				Description: fmt.Sprintf("Service %q (repo_scope=%q root=%q) existed in previous snapshot but is missing in current inventory", removed.ServiceID, removed.RepoScope, removed.ServiceRoot),
				RuleID:      "rule.service.inventory.removed",
				RelatedIDs:  []string{removed.ServiceID},
				Provenance: contracts.Provenance{
					Kind:       "inference",
					Confidence: 0.72,
					Evidence: []contracts.Evidence{
						{Repo: removed.RepoScope, Path: removed.ServiceRoot},
					},
				},
			})
		}
		e.questions = mergeQuestions(e.questions, toContractsQuestions(questions))
		e.findings = append(e.findings, removedFindings...)
	}
	return nil
}

// tiny local surrogate to avoid importing contracts in this file hot path repeatedly.
type contractsQuestion struct {
	ID       string
	Text     string
	Priority string
}

func toContractsQuestions(items []contractsQuestion) []contracts.Question {
	out := make([]contracts.Question, 0, len(items))
	for _, item := range items {
		out = append(out, contracts.Question{ID: item.ID, Text: item.Text, Priority: item.Priority})
	}
	return out
}

func (e *pipelineExecution) buildServiceInventoryPlan(ctx context.Context) (serviceInventoryPlan, error) {
	mode := string(e.refreshMode)
	if mode == "" {
		mode = string(RefreshModeIncremental)
	}
	plan := serviceInventoryPlan{
		Version:        serviceInventoryArtifactVersion,
		RunID:          e.runID,
		Pipeline:       string(e.pipeline),
		Mode:           mode,
		GeneratedAt:    e.clock().UTC().Format(time.RFC3339),
		Services:       []serviceInventoryService{},
		SelectedShards: []serviceShardPlan{},
		Warnings:       []string{},
		RepoHeads:      []serviceInventoryRepoHead{},
	}

	scopes := normalizeOrderedUniqueStrings(e.selectedRepoScopes)
	if len(scopes) == 0 {
		scopes = collectRepoScopes(e.workspace.Manifest.Repos)
	}
	sort.Strings(scopes)

	serviceIDSeq := map[string]int{}
	for _, scope := range scopes {
		repoPath, err := e.resolveRepoPath(scope)
		if err != nil {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("repo scope %q: %v", scope, err))
			continue
		}
		head := ""
		if e.pipeline == PipelineRefresh {
			head, err = gitRevParseHead(ctx, repoPath)
			if err != nil && e.refreshMode == RefreshModeIncremental {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("repo scope %q: cannot resolve HEAD (%v)", scope, err))
			}
		}
		plan.RepoHeads = append(plan.RepoHeads, serviceInventoryRepoHead{RepoScope: scope, Head: head})

		services, warnings, err := e.discoverServicesForScope(scope, repoPath, serviceIDSeq)
		if err != nil {
			return serviceInventoryPlan{}, err
		}
		plan.Warnings = append(plan.Warnings, warnings...)
		plan.Services = append(plan.Services, services...)
	}

	sort.Slice(plan.Services, func(i, j int) bool {
		if plan.Services[i].RepoScope != plan.Services[j].RepoScope {
			return plan.Services[i].RepoScope < plan.Services[j].RepoScope
		}
		if plan.Services[i].ServiceID != plan.Services[j].ServiceID {
			return plan.Services[i].ServiceID < plan.Services[j].ServiceID
		}
		return plan.Services[i].ServiceRoot < plan.Services[j].ServiceRoot
	})

	if e.pipeline != PipelineRefresh || e.refreshMode == RefreshModeFull {
		plan.SelectedShards = flattenServiceShards(plan.Services)
		return plan, nil
	}

	previous, err := e.loadLatestServiceInventorySnapshot()
	if err != nil {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("incremental refresh fallback to full: %v", err))
		plan.SelectedShards = flattenServiceShards(plan.Services)
		return plan, nil
	}
	selectedShards, removedServices, warnings := e.selectIncrementalShards(ctx, plan.Services, previous)
	plan.SelectedShards = selectedShards
	plan.RemovedServices = removedServices
	plan.Warnings = append(plan.Warnings, warnings...)
	if len(plan.SelectedShards) == 0 {
		plan.Warnings = append(plan.Warnings, "incremental refresh selected zero changed services")
	}
	return plan, nil
}

func (e *pipelineExecution) persistServiceInventoryArtifacts(plan serviceInventoryPlan) error {
	planPath := fmt.Sprintf("reports/taskruns/%s-service-inventory-plan.json", e.runID)
	planRaw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal service inventory plan: %w", err)
	}
	planRaw = append(planRaw, '\n')
	if err := e.workspace.WriteFile(planPath, planRaw); err != nil {
		return err
	}
	e.addArtifacts(Artifact{Path: planPath, Kind: "taskrun", Label: "service inventory plan"})

	totalShards := 0
	for _, service := range plan.Services {
		totalShards += len(service.Shards)
	}
	summaryPayload := map[string]any{
		"version":          serviceInventoryArtifactVersion,
		"run_id":           e.runID,
		"pipeline":         string(e.pipeline),
		"mode":             plan.Mode,
		"services_total":   len(plan.Services),
		"shards_total":     totalShards,
		"selected_shards":  len(plan.SelectedShards),
		"removed_services": len(plan.RemovedServices),
		"warnings":         normalizeOrderedUniqueStrings(plan.Warnings),
		"generated_at":     e.clock().UTC().Format(time.RFC3339),
	}
	summaryRaw, err := json.MarshalIndent(summaryPayload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal service inventory summary: %w", err)
	}
	summaryRaw = append(summaryRaw, '\n')
	summaryPath := fmt.Sprintf("reports/taskruns/%s-service-inventory-summary.json", e.runID)
	if err := e.workspace.WriteFile(summaryPath, summaryRaw); err != nil {
		return err
	}
	e.addArtifacts(Artifact{Path: summaryPath, Kind: "taskrun", Label: "service inventory summary"})

	snapshot := serviceInventorySnapshot{
		Version:     serviceInventoryArtifactVersion,
		GeneratedAt: e.clock().UTC().Format(time.RFC3339),
		RunID:       e.runID,
		Pipeline:    string(e.pipeline),
		RepoHeads:   append([]serviceInventoryRepoHead(nil), plan.RepoHeads...),
		Services:    toSnapshotServices(plan.Services),
	}
	snapshotRaw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal service inventory snapshot: %w", err)
	}
	snapshotRaw = append(snapshotRaw, '\n')
	if err := e.workspace.WriteFile(serviceInventoryLatestSnapshotPath, snapshotRaw); err != nil {
		return err
	}
	e.addArtifacts(Artifact{Path: serviceInventoryLatestSnapshotPath, Kind: "taskrun", Label: "service inventory latest"})
	return nil
}

func toSnapshotServices(services []serviceInventoryService) []serviceInventorySnapshotService {
	out := make([]serviceInventorySnapshotService, 0, len(services))
	for _, service := range services {
		shards := make([]serviceInventorySnapshotShard, 0, len(service.Shards))
		for _, shard := range service.Shards {
			shards = append(shards, serviceInventorySnapshotShard{
				ShardID:     shard.ShardID,
				PathScopes:  append([]string(nil), shard.PathScopes...),
				FileCount:   shard.FileCount,
				SourceBytes: shard.SourceBytes,
			})
		}
		out = append(out, serviceInventorySnapshotService{
			RepoScope:   service.RepoScope,
			ServiceID:   service.ServiceID,
			ServiceRoot: service.ServiceRoot,
			FileCount:   service.FileCount,
			SourceBytes: service.SourceBytes,
			Shards:      shards,
		})
	}
	return out
}

func flattenServiceShards(services []serviceInventoryService) []serviceShardPlan {
	out := []serviceShardPlan{}
	for _, service := range services {
		out = append(out, service.Shards...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortKey != out[j].SortKey {
			return out[i].SortKey < out[j].SortKey
		}
		return out[i].ShardID < out[j].ShardID
	})
	return out
}

func (e *pipelineExecution) loadLatestServiceInventorySnapshot() (serviceInventorySnapshot, error) {
	raw, err := e.workspace.ReadFile(serviceInventoryLatestSnapshotPath)
	if err != nil {
		return serviceInventorySnapshot{}, err
	}
	var snapshot serviceInventorySnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return serviceInventorySnapshot{}, err
	}
	if snapshot.Version != serviceInventoryArtifactVersion {
		return serviceInventorySnapshot{}, fmt.Errorf("unsupported snapshot version %d", snapshot.Version)
	}
	return snapshot, nil
}

func (e *pipelineExecution) selectIncrementalShards(
	ctx context.Context,
	services []serviceInventoryService,
	previous serviceInventorySnapshot,
) ([]serviceShardPlan, []serviceInventoryRemoved, []string) {
	warnings := []string{}
	selectedByService := map[string]struct{}{}
	removed := []serviceInventoryRemoved{}

	currentByRepo := map[string][]serviceInventoryService{}
	for _, service := range services {
		currentByRepo[service.RepoScope] = append(currentByRepo[service.RepoScope], service)
	}
	prevByRepo := map[string][]serviceInventorySnapshotService{}
	for _, service := range previous.Services {
		prevByRepo[service.RepoScope] = append(prevByRepo[service.RepoScope], service)
	}
	prevHeadByRepo := map[string]string{}
	for _, head := range previous.RepoHeads {
		prevHeadByRepo[head.RepoScope] = strings.TrimSpace(head.Head)
	}

	for repoScope, currentServices := range currentByRepo {
		repoPath, err := e.resolveRepoPath(repoScope)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("incremental repo %q fallback to full: %v", repoScope, err))
			for _, service := range currentServices {
				selectedByService[service.ServiceID] = struct{}{}
			}
			continue
		}
		prevHead := strings.TrimSpace(prevHeadByRepo[repoScope])
		if prevHead == "" {
			warnings = append(warnings, fmt.Sprintf("incremental repo %q fallback to full: previous HEAD is unavailable", repoScope))
			for _, service := range currentServices {
				selectedByService[service.ServiceID] = struct{}{}
			}
			continue
		}
		changedPaths, err := gitChangedPaths(ctx, repoPath, prevHead)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("incremental repo %q fallback to full: git diff failed (%v)", repoScope, err))
			for _, service := range currentServices {
				selectedByService[service.ServiceID] = struct{}{}
			}
			continue
		}
		changedPaths = normalizeAndSortShardPaths(changedPaths)

		prevSet := map[string]serviceInventorySnapshotService{}
		for _, prevService := range prevByRepo[repoScope] {
			prevSet[prevService.ServiceID] = prevService
		}
		currentSet := map[string]serviceInventoryService{}
		for _, currentService := range currentServices {
			currentSet[currentService.ServiceID] = currentService
		}

		for serviceID, service := range currentSet {
			if _, exists := prevSet[serviceID]; !exists {
				selectedByService[serviceID] = struct{}{}
				continue
			}
			if serviceMatchesChangedPaths(service, changedPaths) {
				selectedByService[serviceID] = struct{}{}
			}
		}
		for serviceID, prevService := range prevSet {
			if _, exists := currentSet[serviceID]; exists {
				continue
			}
			removed = append(removed, serviceInventoryRemoved{
				RepoScope:   repoScope,
				ServiceID:   serviceID,
				ServiceRoot: prevService.ServiceRoot,
			})
		}
	}

	selected := []serviceShardPlan{}
	for _, service := range services {
		if _, ok := selectedByService[service.ServiceID]; !ok {
			continue
		}
		selected = append(selected, service.Shards...)
	}
	sort.Slice(selected, func(i, j int) bool {
		if selected[i].SortKey != selected[j].SortKey {
			return selected[i].SortKey < selected[j].SortKey
		}
		return selected[i].ShardID < selected[j].ShardID
	})
	sort.Slice(removed, func(i, j int) bool {
		if removed[i].RepoScope != removed[j].RepoScope {
			return removed[i].RepoScope < removed[j].RepoScope
		}
		if removed[i].ServiceID != removed[j].ServiceID {
			return removed[i].ServiceID < removed[j].ServiceID
		}
		return removed[i].ServiceRoot < removed[j].ServiceRoot
	})
	warnings = normalizeOrderedUniqueStrings(warnings)
	return selected, removed, warnings
}

func serviceMatchesChangedPaths(service serviceInventoryService, changedPaths []string) bool {
	if len(changedPaths) == 0 {
		return false
	}
	root := normalizeShardPath(service.ServiceRoot)
	for _, changed := range changedPaths {
		candidate := normalizeShardPath(changed)
		if root == "." || root == "" {
			return true
		}
		if candidate == root || strings.HasPrefix(candidate, root+"/") {
			return true
		}
	}
	return false
}

func (e *pipelineExecution) discoverServicesForScope(scope string, repoPath string, idSeq map[string]int) ([]serviceInventoryService, []string, error) {
	repo, _ := lookupManifestRepo(e.workspace.Manifest.Repos, scope)
	discovery, err := discoverHeuristicShardPathsWithMeta(repoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("discover service roots for %q: %w", scope, err)
	}
	roots := discovery.Paths
	warnings := []string{}
	if discovery.FallbackNoMarkers {
		warnings = append(
			warnings,
			fmt.Sprintf("repo scope %q service inventory discovered zero module markers; fallback service_root='.'", scope),
		)
	}
	roots = applyRepoAnalysisFilters(roots, repo.Analysis)
	if len(roots) == 0 {
		roots = []string{"."}
		warnings = append(
			warnings,
			fmt.Sprintf("repo scope %q analysis filters produced zero service roots; fallback service_root='.'", scope),
		)
	}
	roots = pruneParentShardPaths(roots)
	roots = normalizeAndSortShardPaths(roots)

	services := make([]serviceInventoryService, 0, len(roots))
	for _, root := range roots {
		sourceFiles, err := collectServiceSourceFiles(repoPath, root)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("service root %q in repo %q: source scan failed (%v)", root, scope, err))
		}
		totalBytes := int64(0)
		for _, file := range sourceFiles {
			totalBytes += file.Bytes
		}
		baseID := buildServiceID(scope, root)
		seq := idSeq[baseID]
		idSeq[baseID] = seq + 1
		serviceID := baseID
		if seq > 0 {
			serviceID = fmt.Sprintf("%s-%d", baseID, seq+1)
		}
		shards := chunkService(scope, serviceID, root, sourceFiles, totalBytes)
		services = append(services, serviceInventoryService{
			RepoScope:   scope,
			ServiceID:   serviceID,
			ServiceRoot: root,
			FileCount:   len(sourceFiles),
			SourceBytes: totalBytes,
			SourceFiles: sourceFiles,
			Shards:      shards,
		})
	}

	sort.Slice(services, func(i, j int) bool {
		if services[i].ServiceID != services[j].ServiceID {
			return services[i].ServiceID < services[j].ServiceID
		}
		return services[i].ServiceRoot < services[j].ServiceRoot
	})
	return services, normalizeOrderedUniqueStrings(warnings), nil
}

func collectServiceSourceFiles(repoPath string, root string) ([]serviceSourceFile, error) {
	base := filepath.Clean(repoPath)
	target := base
	if normalizeShardPath(root) != "." {
		target = filepath.Join(base, filepath.FromSlash(root))
	}
	info, err := os.Stat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	files := []serviceSourceFile{}
	err = filepath.WalkDir(target, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := strings.ToLower(strings.TrimSpace(entry.Name()))
			if _, skip := shardSkippedDirs[name]; skip && current != target {
				return filepath.SkipDir
			}
			if strings.HasPrefix(name, ".") && current != target {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSourceFileCandidate(entry.Name()) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(base, current)
		if err != nil {
			return nil
		}
		files = append(files, serviceSourceFile{
			Path:  normalizeShardPath(rel),
			Bytes: info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func isSourceFileCandidate(name string) bool {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(name)))
	if ext == "" {
		return false
	}
	if _, ok := semanticSourceExtensions[ext]; ok {
		return true
	}
	switch ext {
	case ".rb", ".php", ".cs", ".swift", ".scala", ".proto", ".sql", ".yaml", ".yml", ".json", ".toml":
		return true
	default:
		return false
	}
}

func buildServiceID(scope string, root string) string {
	rootKey := normalizeShardPath(root)
	if rootKey == "." || rootKey == "" {
		rootKey = "root"
	}
	slug := slugutil.Slugify(scope + "-" + rootKey)
	if slug == "" {
		slug = "service"
	}
	return "svc." + slug
}

func chunkService(scope string, serviceID string, root string, files []serviceSourceFile, totalBytes int64) []serviceShardPlan {
	large := len(files) > serviceLargeFilesThreshold || totalBytes > serviceLargeBytesThreshold
	if !large {
		return []serviceShardPlan{newServiceShard(scope, serviceID, root, serviceID+"-s1", files, 1)}
	}

	chunks := [][]serviceSourceFile{}
	current := []serviceSourceFile{}
	currentBytes := int64(0)
	for idx, file := range files {
		isLastChunk := len(chunks) >= serviceChunkMaxCount-1
		if isLastChunk {
			current = append(current, files[idx:]...)
			currentBytes += sumServiceSourceBytes(files[idx:])
			break
		}
		shouldSplit := len(current) >= serviceChunkMaxFiles
		if !shouldSplit && len(current) > 0 && currentBytes+file.Bytes > serviceChunkMaxBytes {
			shouldSplit = true
		}
		if shouldSplit {
			chunks = append(chunks, current)
			current = []serviceSourceFile{}
			currentBytes = 0
		}
		current = append(current, file)
		currentBytes += file.Bytes
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	if len(chunks) == 0 {
		chunks = append(chunks, nil)
	}

	out := make([]serviceShardPlan, 0, len(chunks))
	for idx, chunk := range chunks {
		shardID := fmt.Sprintf("%s-s%d", serviceID, idx+1)
		out = append(out, newServiceShard(scope, serviceID, root, shardID, chunk, idx+1))
	}
	return out
}

func sumServiceSourceBytes(files []serviceSourceFile) int64 {
	total := int64(0)
	for _, file := range files {
		total += file.Bytes
	}
	return total
}

func newServiceShard(scope string, serviceID string, root string, shardID string, files []serviceSourceFile, order int) serviceShardPlan {
	pathScopes := deriveServicePathScopes(root, files)
	return serviceShardPlan{
		RepoScope:   scope,
		ServiceID:   serviceID,
		ServiceRoot: normalizeShardPath(root),
		ShardID:     shardID,
		SortKey:     fmt.Sprintf("%s:%s:%03d", scope, serviceID, order),
		PathScopes:  pathScopes,
		FileCount:   len(files),
		SourceBytes: sumServiceSourceBytes(files),
	}
}

func deriveServicePathScopes(root string, files []serviceSourceFile) []string {
	root = normalizeShardPath(root)
	if len(files) == 0 {
		return []string{root}
	}
	dirs := map[string]struct{}{}
	for _, file := range files {
		dir := normalizeShardPath(filepath.ToSlash(filepath.Dir(file.Path)))
		if dir == "." || dir == "" {
			dir = root
		}
		dirs[dir] = struct{}{}
	}
	paths := make([]string, 0, len(dirs))
	for dir := range dirs {
		paths = append(paths, dir)
	}
	sort.Strings(paths)
	if len(paths) > 40 {
		if root == "" {
			root = "."
		}
		return []string{root}
	}
	return paths
}

func (e *pipelineExecution) resolveRepoPath(scope string) (string, error) {
	if path := strings.TrimSpace(e.resolvedRepoPaths[scope]); path != "" {
		return path, nil
	}
	repo, ok := lookupManifestRepo(e.workspace.Manifest.Repos, scope)
	if !ok {
		return "", fmt.Errorf("repo scope is not in workspace manifest")
	}
	if strings.TrimSpace(repo.Path) != "" {
		candidate := strings.TrimSpace(repo.Path)
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(e.workspace.Path, candidate)
		}
		return filepath.Clean(candidate), nil
	}
	return "", fmt.Errorf("repo scope has no resolved local path")
}

func gitRevParseHead(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "--verify", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func gitChangedPaths(ctx context.Context, repoPath string, prevHead string) ([]string, error) {
	paths := []string{}
	seen := map[string]struct{}{}
	appendPath := func(raw string) {
		normalized := normalizeShardPath(raw)
		if normalized == "" {
			return
		}
		if _, ok := seen[normalized]; ok {
			return
		}
		seen[normalized] = struct{}{}
		paths = append(paths, normalized)
	}

	diffCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", "--name-only", strings.TrimSpace(prevHead)+"...HEAD")
	diffOut, err := diffCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w: %s", err, strings.TrimSpace(string(diffOut)))
	}
	for _, line := range strings.Split(string(diffOut), "\n") {
		appendPath(line)
	}

	statusCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "status", "--porcelain")
	statusOut, err := statusCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w: %s", err, strings.TrimSpace(string(statusOut)))
	}
	for _, line := range strings.Split(string(statusOut), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(line) < 4 {
			continue
		}
		candidate := strings.TrimSpace(line[3:])
		if strings.Contains(candidate, " -> ") {
			parts := strings.Split(candidate, " -> ")
			for _, part := range parts {
				appendPath(part)
			}
			continue
		}
		appendPath(candidate)
	}
	sort.Strings(paths)
	return paths, nil
}
