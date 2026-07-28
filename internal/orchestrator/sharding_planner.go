package orchestrator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/pathscope"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func (e *pipelineExecution) planRuntimeShards(repoScopes []string) ([]runtimeShardPlan, []string, []runtimeShardPlanGraphEdge) {
	result := (defaultShardPlanner{}).PlanRuntimeShards(ShardPlanInput{
		Workspace:         e.workspace,
		ResolvedRepoPaths: cloneStringMap(e.resolvedRepoPaths),
		ExecutionProfile:  e.executionProfile,
		RepoScopes:        append([]string(nil), repoScopes...),
	})
	return result.Plans, result.Warnings, result.SemanticGraph
}

func (defaultShardPlanner) PlanRuntimeShards(input ShardPlanInput) ShardPlanResult {
	scopes := normalizeOrderedUniqueStrings(input.RepoScopes)
	if len(scopes) == 0 {
		return ShardPlanResult{Plans: []runtimeShardPlan{{
			SortKey:     "workspace:.",
			ShardID:     "workspace-root",
			RepoScopes:  nil,
			PathScopes:  []string{"."},
			PrimaryRepo: "workspace",
		}}}
	}

	warnings := []string{}
	plans := []runtimeShardPlan{}
	graphEdges := []runtimeShardPlanGraphEdge{}
	seenShardIDs := map[string]int{}

	for _, scope := range scopes {
		paths, pathWarnings := planScopePathsForInput(input, scope)
		warnings = append(warnings, pathWarnings...)
		if len(paths) == 0 {
			paths = []string{"."}
		}

		repoPath := resolveRepoPathForInput(input, scope)
		groups, groupingWarnings := buildStructuralShardGroups(repoPath, paths)
		warnings = append(warnings, groupingWarnings...)
		if len(groups) == 0 {
			groups = make([][]string, 0, len(paths))
			for _, pathScope := range paths {
				groups = append(groups, []string{pathScope})
			}
		}
		mode := strings.TrimSpace(strings.ToLower(input.ExecutionProfile.ShardMode))
		if mode == "" {
			mode = "heuristics"
		}
		if mode == "semantic" {
			semanticWarnings, semanticEdges := discoverSemanticShardGraph(repoPath, paths)
			warnings = append(warnings, semanticWarnings...)
			for _, edge := range semanticEdges {
				graphEdges = append(graphEdges, runtimeShardPlanGraphEdge{
					RepoScope: scope,
					FromPath:  edge.FromPath,
					ToPath:    edge.ToPath,
					Reason:    edge.Reason,
				})
			}
		}
		for idx, group := range groups {
			normalizedGroup := normalizeOrderedUniqueStrings(group)
			if len(normalizedGroup) == 0 {
				normalizedGroup = []string{"."}
			}
			sort.Strings(normalizedGroup)
			baseID := buildShardID(scope, normalizedGroup)
			sequence := seenShardIDs[baseID]
			seenShardIDs[baseID] = sequence + 1
			shardID := baseID
			if sequence > 0 {
				shardID = appendShardIDSequence(baseID, sequence+1)
			}
			sortKey := fmt.Sprintf("%s:%s:%03d", scope, strings.Join(normalizedGroup, "|"), idx)
			plans = append(plans, runtimeShardPlan{
				SortKey:     sortKey,
				ShardID:     shardID,
				RepoScopes:  []string{scope},
				PathScopes:  normalizedGroup,
				PrimaryRepo: scope,
			})
		}
	}

	sort.Slice(plans, func(i, j int) bool {
		if plans[i].SortKey == plans[j].SortKey {
			return plans[i].ShardID < plans[j].ShardID
		}
		return plans[i].SortKey < plans[j].SortKey
	})
	sort.Slice(graphEdges, func(i, j int) bool {
		if graphEdges[i].RepoScope != graphEdges[j].RepoScope {
			return graphEdges[i].RepoScope < graphEdges[j].RepoScope
		}
		if graphEdges[i].FromPath != graphEdges[j].FromPath {
			return graphEdges[i].FromPath < graphEdges[j].FromPath
		}
		if graphEdges[i].ToPath != graphEdges[j].ToPath {
			return graphEdges[i].ToPath < graphEdges[j].ToPath
		}
		return graphEdges[i].Reason < graphEdges[j].Reason
	})
	return ShardPlanResult{
		Plans:         plans,
		Warnings:      warnings,
		SemanticGraph: graphEdges,
	}
}

func buildShardID(scope string, pathScopes []string) string {
	parts := append([]string{scope}, pathScopes...)
	joined := strings.Join(parts, "-")
	slug := slugutil.Slugify(joined)
	if strings.TrimSpace(slug) == "" {
		return "shard"
	}
	return boundRuntimeShardID(slug)
}

func appendShardIDSequence(base string, sequence int) string {
	if sequence <= 1 {
		return base
	}
	suffix := fmt.Sprintf("-%d", sequence)
	if len(base)+len(suffix) <= maxRuntimeShardIDLength {
		return base + suffix
	}
	limit := maxRuntimeShardIDLength - len(suffix)
	if limit <= 0 {
		return strings.TrimPrefix(suffix, "-")
	}
	prefix := strings.Trim(base[:limit], "-")
	if prefix == "" {
		prefix = "shard"
	}
	return prefix + suffix
}

func boundRuntimeShardID(slug string) string {
	slug = strings.Trim(strings.TrimSpace(slug), "-")
	if slug == "" {
		return "shard"
	}
	if len(slug) <= maxRuntimeShardIDLength {
		return slug
	}
	sum := sha256.Sum256([]byte(slug))
	hash := hex.EncodeToString(sum[:])[:runtimeShardIDHashLength]
	limit := maxRuntimeShardIDLength - len(hash) - 1
	if limit <= 0 {
		return hash
	}
	prefix := strings.Trim(slug[:limit], "-")
	if prefix == "" {
		prefix = "shard"
	}
	return prefix + "-" + hash
}

func resolveRepoPathForInput(input ShardPlanInput, scope string) string {
	repoPath := strings.TrimSpace(input.ResolvedRepoPaths[scope])
	repo, ok := lookupManifestRepo(input.Workspace.Manifest.Repos, scope)
	if !ok {
		return repoPath
	}
	if repoPath == "" && strings.TrimSpace(repo.Path) != "" {
		repoPath = strings.TrimSpace(repo.Path)
		if !filepath.IsAbs(repoPath) {
			repoPath = filepath.Join(input.Workspace.Path, repoPath)
		}
	}
	return strings.TrimSpace(repoPath)
}

func planScopePathsForInput(input ShardPlanInput, scope string) ([]string, []string) {
	warnings := []string{}
	repo, ok := lookupManifestRepo(input.Workspace.Manifest.Repos, scope)
	if !ok {
		return []string{"."}, []string{fmt.Sprintf("repo scope %q is not present in workspace manifest; fallback shard='.'", scope)}
	}

	repoPath := resolveRepoPathForInput(input, scope)
	if strings.TrimSpace(repoPath) == "" {
		return []string{"."}, []string{fmt.Sprintf("repo scope %q has no local path for shard discovery; fallback shard='.'", scope)}
	}

	discovery, err := discoverHeuristicShardPathsWithMeta(repoPath)
	if err != nil {
		return []string{"."}, []string{fmt.Sprintf("repo scope %q shard discovery failed (%v); fallback shard='.'", scope, err)}
	}
	paths := discovery.Paths
	if discovery.FallbackNoMarkers {
		warnings = append(
			warnings,
			fmt.Sprintf("repo scope %q shard discovery found zero module markers; heuristic fallback shard='.'", scope),
		)
	}
	filtered := applyRepoAnalysisFilters(paths, repo.Analysis)
	if len(filtered) == 0 {
		return []string{"."}, []string{fmt.Sprintf("repo scope %q analysis filters produced zero shards; fallback shard='.'", scope)}
	}
	return filtered, warnings
}

func lookupManifestRepo(repos []workspace.RepoSource, name string) (workspace.RepoSource, bool) {
	target := strings.TrimSpace(name)
	for _, repo := range repos {
		if strings.TrimSpace(repo.Name) == target {
			return repo, true
		}
	}
	return workspace.RepoSource{}, false
}

func discoverHeuristicShardPathsWithMeta(repoPath string) (heuristicShardDiscoveryResult, error) {
	root := strings.TrimSpace(repoPath)
	if root == "" {
		return heuristicShardDiscoveryResult{Paths: []string{"."}}, nil
	}
	info, err := os.Stat(root)
	if err != nil {
		return heuristicShardDiscoveryResult{}, err
	}
	if !info.IsDir() {
		return heuristicShardDiscoveryResult{}, fmt.Errorf("repo path %q is not a directory", root)
	}

	markerRoots, err := discoverShardModuleMarkerRoots(root)
	if err != nil {
		return heuristicShardDiscoveryResult{}, err
	}
	if len(markerRoots) == 0 {
		return heuristicShardDiscoveryResult{
			Paths:             []string{"."},
			FallbackNoMarkers: true,
		}, nil
	}
	if hasOnlyRootModuleMarker(markerRoots) {
		coverageRoots, err := discoverRootMarkerCoverageRoots(root)
		if err != nil {
			return heuristicShardDiscoveryResult{}, err
		}
		return heuristicShardDiscoveryResult{Paths: coverageRoots}, nil
	}
	coverageRoots, err := buildStructuralCoverageRoots(root, markerRoots)
	if err != nil {
		return heuristicShardDiscoveryResult{}, err
	}
	return heuristicShardDiscoveryResult{Paths: coverageRoots}, nil
}

func hasOnlyRootModuleMarker(markerRoots []string) bool {
	normalized := normalizeAndSortShardPaths(markerRoots)
	return len(normalized) == 1 && normalized[0] == "."
}

func discoverRootMarkerCoverageRoots(repoPath string) ([]string, error) {
	root := strings.TrimSpace(repoPath)
	if root == "" {
		return []string{"."}, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	roots := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		if entry.IsDir() {
			lowerName := strings.ToLower(name)
			if _, skip := shardSkippedDirs[lowerName]; skip {
				continue
			}
			if strings.HasPrefix(lowerName, ".") {
				continue
			}
		}
		roots = append(roots, normalizeShardPath(name))
	}
	if len(roots) == 0 {
		return []string{"."}, nil
	}
	return normalizeAndSortShardPaths(roots), nil
}

func discoverShardModuleMarkerRoots(repoPath string) ([]string, error) {
	root := strings.TrimSpace(repoPath)
	if root == "" {
		return nil, nil
	}
	markerRoots := map[string]struct{}{}
	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := strings.ToLower(strings.TrimSpace(entry.Name()))
			if _, skip := shardSkippedDirs[name]; skip {
				if current != root {
					return filepath.SkipDir
				}
			}
			if strings.HasPrefix(name, ".") && current != root {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(strings.TrimSpace(entry.Name()))
		if _, ok := shardModuleMarkerFiles[name]; !ok {
			return nil
		}
		rel, relErr := filepath.Rel(root, filepath.Dir(current))
		if relErr != nil {
			return nil
		}
		markerRoots[normalizeShardPath(rel)] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}
	leafMarkers := make([]string, 0, len(markerRoots))
	for candidate := range markerRoots {
		leafMarkers = append(leafMarkers, candidate)
	}
	return pruneParentShardPaths(leafMarkers), nil
}

func buildStructuralCoverageRoots(repoPath string, leafMarkers []string) ([]string, error) {
	normalizedMarkers := normalizeAndSortShardPaths(leafMarkers)
	if len(normalizedMarkers) == 0 {
		return []string{"."}, nil
	}

	leafSet := map[string]struct{}{}
	descendantSet := map[string]struct{}{}
	for _, marker := range normalizedMarkers {
		leafSet[marker] = struct{}{}
		current := marker
		for {
			descendantSet[current] = struct{}{}
			if current == "." {
				break
			}
			current = shardParentPath(current)
		}
	}

	coverageRoots := []string{}
	if err := appendCoverageRoots(repoPath, ".", leafSet, descendantSet, &coverageRoots); err != nil {
		return nil, err
	}
	return normalizeAndSortShardPaths(coverageRoots), nil
}

func appendCoverageRoots(repoPath string, rel string, leafSet map[string]struct{}, descendantSet map[string]struct{}, out *[]string) error {
	rel = normalizeShardPath(rel)
	if _, ok := leafSet[rel]; ok {
		*out = append(*out, rel)
		return nil
	}

	abs := repoPath
	if rel != "." {
		abs = filepath.Join(repoPath, filepath.FromSlash(rel))
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		childRel := shardJoin(rel, name)
		if entry.IsDir() {
			lowerName := strings.ToLower(name)
			if _, skip := shardSkippedDirs[lowerName]; skip {
				continue
			}
			if strings.HasPrefix(lowerName, ".") {
				continue
			}
			if _, covered := descendantSet[childRel]; covered {
				if err := appendCoverageRoots(repoPath, childRel, leafSet, descendantSet, out); err != nil {
					return err
				}
				continue
			}
			*out = append(*out, childRel)
			continue
		}
		*out = append(*out, childRel)
	}
	return nil
}

func shardParentPath(rel string) string {
	normalized := normalizeShardPath(rel)
	if normalized == "." {
		return "."
	}
	if idx := strings.LastIndex(normalized, "/"); idx >= 0 {
		return normalized[:idx]
	}
	return "."
}

func shardJoin(base string, child string) string {
	child = strings.TrimSpace(child)
	if child == "" {
		return normalizeShardPath(base)
	}
	if normalizeShardPath(base) == "." {
		return normalizeShardPath(child)
	}
	return normalizeShardPath(path.Join(base, child))
}

func pruneParentShardPaths(paths []string) []string {
	if len(paths) <= 1 {
		return normalizeAndSortShardPaths(paths)
	}
	normalized := normalizeAndSortShardPaths(paths)
	out := make([]string, 0, len(normalized))
	for _, candidate := range normalized {
		hasChild := false
		for _, other := range normalized {
			if candidate == other {
				continue
			}
			if candidate == "." {
				hasChild = true
				break
			}
			if strings.HasPrefix(other, candidate+"/") {
				hasChild = true
				break
			}
		}
		if hasChild {
			continue
		}
		out = append(out, candidate)
	}
	if len(out) == 0 {
		return []string{"."}
	}
	return out
}

func normalizeAndSortShardPaths(paths []string) []string {
	set := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, raw := range paths {
		normalized := normalizeShardPath(raw)
		if _, exists := set[normalized]; exists {
			continue
		}
		set[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return []string{"."}
	}
	return out
}

func normalizeShardPath(value string) string {
	normalized := filepath.ToSlash(strings.TrimSpace(value))
	normalized = strings.TrimPrefix(normalized, "./")
	normalized = strings.TrimPrefix(normalized, "/")
	normalized = strings.TrimSuffix(normalized, "/")
	if normalized == "" {
		return "."
	}
	return normalized
}

func applyRepoAnalysisFilters(paths []string, analysis *workspace.RepoAnalysisConfig) []string {
	normalized := normalizeAndSortShardPaths(paths)
	if analysis == nil {
		return normalized
	}
	include := normalizeOrderedUniqueStrings(analysis.Include)
	exclude := normalizeOrderedUniqueStrings(analysis.Exclude)
	filtered := make([]string, 0, len(normalized))
	for _, candidate := range normalized {
		if len(include) > 0 && !matchesAnyShardPattern(candidate, include) {
			continue
		}
		if len(exclude) > 0 && matchesAnyShardPattern(candidate, exclude) {
			continue
		}
		filtered = append(filtered, candidate)
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func matchesAnyShardPattern(candidate string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchShardPattern(candidate, pattern) {
			return true
		}
	}
	return false
}

func matchShardPattern(candidate string, pattern string) bool {
	return pathscope.Match(pattern, candidate)
}

func buildStructuralShardGroups(repoPath string, coverageRoots []string) ([][]string, []string) {
	normalized := normalizeAndSortShardPaths(coverageRoots)
	if len(normalized) == 0 {
		return [][]string{{"."}}, nil
	}
	if len(normalized) <= maxAutoShardsPerRepo || strings.TrimSpace(repoPath) == "" {
		if strings.TrimSpace(repoPath) != "" && len(normalized) <= maxAutoShardsPerRepo {
			if grouped, ok := groupRootFilesWithinCap(repoPath, normalized); ok {
				return grouped, nil
			}
		}
		groups := make([][]string, 0, len(normalized))
		for _, value := range normalized {
			groups = append(groups, []string{value})
		}
		return groups, nil
	}

	rootFiles := make([]string, 0, len(normalized))
	topLevelRoots := map[string][]string{}
	for _, rel := range normalized {
		if rel == "." {
			return [][]string{{"."}}, []string{fmt.Sprintf("structural shard coalescing skipped because repo %q is already covered by root scope", repoPath)}
		}
		abs := filepath.Join(repoPath, filepath.FromSlash(rel))
		info, err := os.Stat(abs)
		if err != nil {
			groups := make([][]string, 0, len(normalized))
			for _, value := range normalized {
				groups = append(groups, []string{value})
			}
			return groups, []string{fmt.Sprintf("structural shard coalescing fallback: stat failed for %q (%v); keeping coverage roots", rel, err)}
		}
		if info.IsDir() {
			key := topLevelSegment(rel)
			topLevelRoots[key] = append(topLevelRoots[key], rel)
			continue
		}
		if !strings.Contains(rel, "/") {
			rootFiles = append(rootFiles, rel)
			continue
		}
		key := topLevelSegment(rel)
		topLevelRoots[key] = append(topLevelRoots[key], rel)
	}

	keys := make([]string, 0, len(topLevelRoots))
	for key := range topLevelRoots {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	sort.Strings(rootFiles)

	groups := make([][]string, 0, len(keys)+1)
	if len(rootFiles) > 0 {
		groups = append(groups, append([]string(nil), rootFiles...))
	}
	for _, key := range keys {
		groups = append(groups, []string{key})
	}

	warnings := []string{
		fmt.Sprintf(
			"structural shard coalescing reduced %d coverage roots to %d shard groups using top-level ancestry",
			len(normalized),
			len(groups),
		),
	}
	if preservedGroups, preservedWarnings := preserveMarkerLeafShardGroups(repoPath, groups, rootFiles, keys, topLevelRoots); len(preservedGroups) > 0 {
		groups = preservedGroups
		warnings = append(warnings, preservedWarnings...)
	} else {
		warnings = append(warnings, preservedWarnings...)
	}
	if len(groups) > maxAutoShardsPerRepo {
		before := len(groups)
		groups = coalesceShardGroupsWithinCap(groups, rootFiles)
		warnings = append(
			warnings,
			fmt.Sprintf(
				"structural shard coalescing merged %d shard groups to %d groups to enforce target cap=%d",
				before,
				len(groups),
				maxAutoShardsPerRepo,
			),
		)
	}
	return groups, warnings
}

func coalesceShardGroupsWithinCap(groups [][]string, rootFiles []string) [][]string {
	if len(groups) <= maxAutoShardsPerRepo {
		return cloneShardGroups(groups)
	}
	result := make([][]string, 0, maxAutoShardsPerRepo)
	start := 0
	if len(rootFiles) > 0 && len(groups) > 0 && sameShardGroup(groups[0], rootFiles) {
		result = append(result, append([]string(nil), groups[0]...))
		start = 1
	}
	available := maxAutoShardsPerRepo - len(result)
	if available <= 0 {
		return result
	}
	remaining := groups[start:]
	if len(remaining) <= available {
		result = append(result, cloneShardGroups(remaining)...)
		return result
	}
	for idx := 0; idx < available; idx++ {
		from := idx * len(remaining) / available
		to := (idx + 1) * len(remaining) / available
		merged := make([]string, 0, to-from)
		for _, group := range remaining[from:to] {
			merged = append(merged, group...)
		}
		result = append(result, normalizeAndSortShardPaths(merged))
	}
	return result
}

func cloneShardGroups(groups [][]string) [][]string {
	cloned := make([][]string, 0, len(groups))
	for _, group := range groups {
		cloned = append(cloned, append([]string(nil), group...))
	}
	return cloned
}

func sameShardGroup(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa := normalizeAndSortShardPaths(a)
	bb := normalizeAndSortShardPaths(b)
	for idx := range aa {
		if aa[idx] != bb[idx] {
			return false
		}
	}
	return true
}

func groupRootFilesWithinCap(repoPath string, normalized []string) ([][]string, bool) {
	rootFiles := make([]string, 0, len(normalized))
	others := make([]string, 0, len(normalized))
	for _, rel := range normalized {
		if rel == "." {
			return nil, false
		}
		abs := filepath.Join(repoPath, filepath.FromSlash(rel))
		info, err := os.Stat(abs)
		if err != nil {
			return nil, false
		}
		if !info.IsDir() && !strings.Contains(rel, "/") {
			rootFiles = append(rootFiles, rel)
			continue
		}
		others = append(others, rel)
	}
	if len(rootFiles) <= 1 {
		return nil, false
	}
	sort.Strings(rootFiles)
	sort.Strings(others)
	groups := make([][]string, 0, len(others)+1)
	groups = append(groups, append([]string(nil), rootFiles...))
	for _, rel := range others {
		groups = append(groups, []string{rel})
	}
	return groups, true
}

func preserveMarkerLeafShardGroups(repoPath string, baseGroups [][]string, rootFiles []string, keys []string, topLevelRoots map[string][]string) ([][]string, []string) {
	markerRoots, err := discoverShardModuleMarkerRoots(repoPath)
	if err != nil {
		return nil, []string{fmt.Sprintf("structural shard marker preservation skipped: marker discovery failed (%v)", err)}
	}
	if len(markerRoots) == 0 {
		return nil, nil
	}
	markerSet := map[string]struct{}{}
	for _, marker := range markerRoots {
		markerSet[normalizeShardPath(marker)] = struct{}{}
	}

	groups := make([][]string, 0, len(baseGroups))
	if len(rootFiles) > 0 {
		groups = append(groups, append([]string(nil), rootFiles...))
	}
	warnings := []string{}
	preserved := 0
	for idx, key := range keys {
		roots := normalizeAndSortShardPaths(topLevelRoots[key])
		markerGroups := make([][]string, 0, len(roots))
		residual := make([]string, 0, len(roots))
		for _, rel := range roots {
			if _, ok := markerSet[rel]; ok {
				markerGroups = append(markerGroups, []string{rel})
				continue
			}
			residual = append(residual, rel)
		}
		if len(markerGroups) == 0 {
			groups = append(groups, []string{key})
			continue
		}

		nextGroupCount := len(groups) + len(markerGroups)
		if len(residual) > 0 {
			nextGroupCount++
		}
		if nextGroupCount+minimumRemainingTopLevelGroups(keys[idx+1:], topLevelRoots) > maxAutoShardsPerRepo {
			groups = append(groups, []string{key})
			warnings = append(warnings, fmt.Sprintf("structural shard marker preservation skipped for %q because it would exceed target cap=%d", key, maxAutoShardsPerRepo))
			continue
		}
		if len(residual) > 0 {
			groups = append(groups, residual)
		}
		groups = append(groups, markerGroups...)
		preserved += len(markerGroups)
	}
	if preserved == 0 {
		return nil, warnings
	}
	warnings = append(warnings, fmt.Sprintf("structural shard coalescing preserved %d module marker leaf shard groups within target cap=%d", preserved, maxAutoShardsPerRepo))
	return groups, warnings
}

func minimumRemainingTopLevelGroups(keys []string, topLevelRoots map[string][]string) int {
	count := 0
	for _, key := range keys {
		if len(topLevelRoots[key]) > 0 {
			count++
		}
	}
	return count
}

func topLevelSegment(rel string) string {
	normalized := normalizeShardPath(rel)
	if normalized == "." {
		return "."
	}
	if idx := strings.Index(normalized, "/"); idx >= 0 {
		return normalized[:idx]
	}
	return normalized
}

func discoverSemanticShardGraph(repoPath string, paths []string) ([]string, []runtimeShardPlanGraphEdge) {
	normalized := normalizeAndSortShardPaths(paths)
	if len(normalized) <= 1 {
		return nil, nil
	}
	if strings.TrimSpace(repoPath) == "" {
		return []string{"semantic shard discovery fallback: repo path unavailable; semantic graph omitted"}, nil
	}

	corpora := make([]string, len(normalized))
	warnings := []string{}
	for idx, rel := range normalized {
		corpus, err := buildSemanticCorpus(repoPath, rel)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("semantic shard discovery: %s corpus failed (%v)", rel, err))
		}
		corpora[idx] = corpus
	}

	graphEdges := make([]runtimeShardPlanGraphEdge, 0, len(normalized))
	for left := 0; left < len(normalized); left++ {
		for right := left + 1; right < len(normalized); right++ {
			if related, reason := semanticRootsRelated(normalized[left], normalized[right], corpora[left], corpora[right]); related {
				graphEdges = append(graphEdges, runtimeShardPlanGraphEdge{
					FromPath: normalized[left],
					ToPath:   normalized[right],
					Reason:   reason,
				})
			}
		}
	}
	sort.Slice(graphEdges, func(i, j int) bool {
		if graphEdges[i].FromPath != graphEdges[j].FromPath {
			return graphEdges[i].FromPath < graphEdges[j].FromPath
		}
		if graphEdges[i].ToPath != graphEdges[j].ToPath {
			return graphEdges[i].ToPath < graphEdges[j].ToPath
		}
		return graphEdges[i].Reason < graphEdges[j].Reason
	})
	return warnings, graphEdges
}

func semanticRootsRelated(left string, right string, leftCorpus string, rightCorpus string) (bool, string) {
	leftTokens := shardSemanticTokens(right)
	rightTokens := shardSemanticTokens(left)
	for _, token := range leftTokens {
		if token != "" && strings.Contains(leftCorpus, token) {
			return true, "left_corpus_contains:" + token
		}
	}
	for _, token := range rightTokens {
		if token != "" && strings.Contains(rightCorpus, token) {
			return true, "right_corpus_contains:" + token
		}
	}
	return false, ""
}

func shardSemanticTokens(rel string) []string {
	normalized := normalizeShardPath(rel)
	if normalized == "." {
		return []string{"./", "../"}
	}
	parts := strings.Split(normalized, "/")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts)+1)
	out = append(out, normalized)
	seen[normalized] = struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 2 {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func buildSemanticCorpus(repoPath string, rel string) (string, error) {
	abs := filepath.Join(repoPath, filepath.FromSlash(rel))
	if rel == "." {
		abs = repoPath
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return readSemanticSourceFile(abs), nil
	}

	parts := make([]string, 0, 32)
	err = filepath.WalkDir(abs, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			name := strings.ToLower(entry.Name())
			if _, skip := shardSkippedDirs[name]; skip && current != abs {
				return filepath.SkipDir
			}
			if strings.HasPrefix(name, ".") && current != abs {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if _, ok := semanticSourceExtensions[ext]; !ok {
			return nil
		}
		file, err := os.Open(current)
		if err != nil {
			return nil
		}
		defer file.Close()
		limited := io.LimitReader(file, 128*1024)
		content, err := io.ReadAll(limited)
		if err != nil {
			return nil
		}
		trimmed := strings.TrimSpace(strings.ToLower(string(content)))
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.Join(parts, "\n"), nil
}

func readSemanticSourceFile(abs string) string {
	ext := strings.ToLower(filepath.Ext(abs))
	if _, ok := semanticSourceExtensions[ext]; !ok {
		return ""
	}
	file, err := os.Open(abs)
	if err != nil {
		return ""
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, 128*1024))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(string(content)))
}
