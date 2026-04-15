package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type runtimeShardPlan struct {
	SortKey     string
	ShardID     string
	RepoScopes  []string
	PathScopes  []string
	PrimaryRepo string
}

type runtimeShardRunResult struct {
	Plan     runtimeShardPlan
	Prepared runtimePreparedExecution
	Err      error
}

type runtimeShardSummary struct {
	Version       int                        `json:"version"`
	RunID         string                     `json:"run_id"`
	StepID        string                     `json:"step_id"`
	DomainID      string                     `json:"domain_id,omitempty"`
	Strategy      string                     `json:"strategy"`
	MaxParallel   int                        `json:"max_parallel_tasks"`
	FailurePolicy string                     `json:"failure_policy"`
	ShardMode     string                     `json:"shard_discovery_mode"`
	GeneratedAt   string                     `json:"generated_at"`
	Items         []runtimeShardSummaryEntry `json:"items"`
}

type runtimeShardSummaryEntry struct {
	ShardID    string   `json:"shard_id"`
	RepoScopes []string `json:"repo_scopes,omitempty"`
	PathScopes []string `json:"path_scopes,omitempty"`
	Status     string   `json:"status"`
	TaskID     string   `json:"task_id,omitempty"`
	TaskRun    string   `json:"taskrun_path,omitempty"`
	Error      string   `json:"error,omitempty"`
}

type runtimeShardPlanArtifact struct {
	Version       int                            `json:"version"`
	RunID         string                         `json:"run_id"`
	StepID        string                         `json:"step_id"`
	DomainID      string                         `json:"domain_id,omitempty"`
	Strategy      string                         `json:"strategy"`
	MaxParallel   int                            `json:"max_parallel_tasks"`
	FailurePolicy string                         `json:"failure_policy"`
	ShardMode     string                         `json:"shard_discovery_mode"`
	PlannerNotes  []string                       `json:"planner_notes,omitempty"`
	SemanticGraph []runtimeShardPlanGraphEdge    `json:"semantic_graph,omitempty"`
	Items         []runtimeShardPlanArtifactItem `json:"items"`
}

type runtimeShardPlanArtifactItem struct {
	SortKey    string   `json:"sort_key"`
	ShardID    string   `json:"shard_id"`
	RepoScopes []string `json:"repo_scopes,omitempty"`
	PathScopes []string `json:"path_scopes,omitempty"`
}

type runtimeShardPlanGraphEdge struct {
	RepoScope string `json:"repo_scope"`
	FromPath  string `json:"from_path"`
	ToPath    string `json:"to_path"`
	Reason    string `json:"reason"`
}

type heuristicShardDiscoveryResult struct {
	Paths             []string
	FallbackNoMarkers bool
}

var shardModuleMarkerFiles = map[string]struct{}{
	"go.mod":          {},
	"package.json":    {},
	"pyproject.toml":  {},
	"cargo.toml":      {},
	"pom.xml":         {},
	"build.gradle":    {},
	"settings.gradle": {},
	"workspace":       {},
	"module.bazel":    {},
}

var shardSkippedDirs = map[string]struct{}{
	".git":         {},
	".hg":          {},
	".svn":         {},
	"node_modules": {},
	"vendor":       {},
	"dist":         {},
	"build":        {},
	"target":       {},
	"tmp":          {},
	".idea":        {},
	".vscode":      {},
}

var semanticSourceExtensions = map[string]struct{}{
	".go":   {},
	".ts":   {},
	".tsx":  {},
	".js":   {},
	".jsx":  {},
	".py":   {},
	".java": {},
	".kt":   {},
	".rs":   {},
}

func (e *pipelineExecution) executeRuntimeTasksSharded(
	ctx context.Context,
	stepID string,
	domainID string,
	repoScopes []string,
	taskSuffixPrefix string,
) ([]runtimeTaskExecution, error) {
	plans, plannerWarnings, semanticGraph := e.planRuntimeShards(repoScopes)
	for _, warning := range plannerWarnings {
		message := strings.TrimSpace(warning)
		if message == "" {
			continue
		}
		e.addWarning(fmt.Sprintf("%s: %s", stepID, message))
		e.logWarn(stepID, domainID, "runtime shard planner warning", map[string]any{"warning": message})
	}
	if len(plans) == 0 {
		plans = []runtimeShardPlan{{
			SortKey:     "workspace:.",
			ShardID:     "workspace-root",
			RepoScopes:  append([]string(nil), repoScopes...),
			PathScopes:  []string{"."},
			PrimaryRepo: "workspace",
		}}
	}

	strategy := strings.TrimSpace(e.executionProfile.Strategy)
	if strategy == "" {
		strategy = "sequential"
	}
	maxParallel := e.executionProfile.MaxParallel
	if maxParallel <= 0 {
		maxParallel = 1
	}
	if strategy != "parallel" {
		maxParallel = 1
	}
	failurePolicy := strings.TrimSpace(e.executionProfile.FailurePolicy)
	if failurePolicy == "" {
		failurePolicy = "best_effort"
	}
	bestEffort := failurePolicy == "best_effort"

	if err := e.persistShardPlan(stepID, domainID, plans, plannerWarnings, semanticGraph, strategy, maxParallel, failurePolicy); err != nil {
		return nil, err
	}

	e.logInfo(stepID, domainID, "runtime shard execution prepared", map[string]any{
		"shards":         len(plans),
		"strategy":       strategy,
		"max_parallel":   maxParallel,
		"failure_policy": failurePolicy,
		"shard_mode":     e.executionProfile.ShardMode,
	})

	results := make([]runtimeShardRunResult, len(plans))
	if maxParallel <= 1 || len(plans) <= 1 {
		for idx, plan := range plans {
			taskSuffix := buildShardTaskSuffix(taskSuffixPrefix, plan.ShardID)
			prepared, err := e.runRuntimeTaskNormalized(ctx, stepID, taskSuffix, plan.RepoScopes, plan.PathScopes, domainID, plan.ShardID)
			results[idx] = runtimeShardRunResult{Plan: plan, Prepared: prepared, Err: err}
			if err != nil && !bestEffort {
				return nil, err
			}
		}
	} else {
		runCtx := ctx
		cancel := func() {}
		if !bestEffort {
			runCtx, cancel = context.WithCancel(ctx)
		}
		defer cancel()

		sem := make(chan struct{}, maxParallel)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstErr error

		for idx, plan := range plans {
			wg.Add(1)
			go func(index int, shard runtimeShardPlan) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				taskSuffix := buildShardTaskSuffix(taskSuffixPrefix, shard.ShardID)
				prepared, err := e.runRuntimeTaskNormalized(runCtx, stepID, taskSuffix, shard.RepoScopes, shard.PathScopes, domainID, shard.ShardID)
				mu.Lock()
				results[index] = runtimeShardRunResult{Plan: shard, Prepared: prepared, Err: err}
				if err != nil && !bestEffort && firstErr == nil {
					firstErr = err
					cancel()
				}
				mu.Unlock()
			}(idx, plan)
		}
		wg.Wait()
		if firstErr != nil {
			return nil, firstErr
		}
	}

	executions := make([]runtimeTaskExecution, 0, len(plans))
	summary := make([]runtimeShardSummaryEntry, 0, len(plans))

	singleShard := len(plans) == 1
	for _, result := range results {
		if result.Err != nil {
			summary = append(summary, runtimeShardSummaryEntry{
				ShardID:    result.Plan.ShardID,
				RepoScopes: append([]string(nil), result.Plan.RepoScopes...),
				PathScopes: append([]string(nil), result.Plan.PathScopes...),
				Status:     "failed",
				Error:      strings.TrimSpace(result.Err.Error()),
			})
			if bestEffort {
				e.registerPartialShardFailure(stepID, domainID, result.Plan, result.Err)
				continue
			}
			return nil, result.Err
		}

		execution, err := e.applyRuntimeTaskExecution(stepID, domainID, result.Prepared)
		if err != nil {
			summary = append(summary, runtimeShardSummaryEntry{
				ShardID:    result.Plan.ShardID,
				RepoScopes: append([]string(nil), result.Plan.RepoScopes...),
				PathScopes: append([]string(nil), result.Plan.PathScopes...),
				Status:     "failed",
				TaskID:     result.Prepared.Task.TaskID,
				Error:      strings.TrimSpace(err.Error()),
			})
			if bestEffort {
				e.registerPartialShardFailure(stepID, domainID, result.Plan, err)
				continue
			}
			return nil, err
		}

		taskrunPath := shardTaskrunPath(e.runID, stepID, domainID, result.Plan.ShardID, singleShard)
		taskrunLabel := shardTaskrunLabel(stepID, domainID, result.Plan.ShardID, singleShard)
		if err := e.persistTaskRun(taskrunPath, taskrunLabel, result.Prepared.NormalizedRaw); err != nil {
			return nil, err
		}
		summary = append(summary, runtimeShardSummaryEntry{
			ShardID:    result.Plan.ShardID,
			RepoScopes: append([]string(nil), result.Plan.RepoScopes...),
			PathScopes: append([]string(nil), result.Plan.PathScopes...),
			Status:     "succeeded",
			TaskID:     result.Prepared.Task.TaskID,
			TaskRun:    taskrunPath,
		})
		executions = append(executions, execution)
	}

	if err := e.persistShardSummary(stepID, domainID, summary); err != nil {
		return nil, err
	}
	return executions, nil
}

func (e *pipelineExecution) registerPartialShardFailure(stepID string, domainID string, plan runtimeShardPlan, err error) {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "unknown shard error"
	}
	e.partialFailures = append(e.partialFailures, runtimeShardFailure{
		StepID:     stepID,
		DomainID:   domainID,
		ShardID:    plan.ShardID,
		RepoScopes: append([]string(nil), plan.RepoScopes...),
		PathScopes: append([]string(nil), plan.PathScopes...),
		Message:    message,
	})
	e.addWarning(fmt.Sprintf("%s: shard %q failed (%s)", stepID, plan.ShardID, message))
	e.logError(stepID, domainID, "runtime shard failed (best-effort continues)", map[string]any{
		"shard_id":    plan.ShardID,
		"repo_scopes": plan.RepoScopes,
		"path_scopes": plan.PathScopes,
		"error":       message,
	})
}

func (e *pipelineExecution) persistShardPlan(
	stepID string,
	domainID string,
	plans []runtimeShardPlan,
	plannerWarnings []string,
	semanticGraph []runtimeShardPlanGraphEdge,
	strategy string,
	maxParallel int,
	failurePolicy string,
) error {
	normalizedWarnings := make([]string, 0, len(plannerWarnings))
	for _, warning := range plannerWarnings {
		trimmed := strings.TrimSpace(warning)
		if trimmed == "" {
			continue
		}
		normalizedWarnings = append(normalizedWarnings, trimmed)
	}
	normalizedWarnings = normalizeOrderedUniqueStrings(normalizedWarnings)

	items := make([]runtimeShardPlanArtifactItem, 0, len(plans))
	for _, plan := range plans {
		items = append(items, runtimeShardPlanArtifactItem{
			SortKey:    plan.SortKey,
			ShardID:    plan.ShardID,
			RepoScopes: append([]string(nil), plan.RepoScopes...),
			PathScopes: append([]string(nil), plan.PathScopes...),
		})
	}
	semanticEdges := append([]runtimeShardPlanGraphEdge(nil), semanticGraph...)
	sort.Slice(semanticEdges, func(i, j int) bool {
		if semanticEdges[i].RepoScope != semanticEdges[j].RepoScope {
			return semanticEdges[i].RepoScope < semanticEdges[j].RepoScope
		}
		if semanticEdges[i].FromPath != semanticEdges[j].FromPath {
			return semanticEdges[i].FromPath < semanticEdges[j].FromPath
		}
		if semanticEdges[i].ToPath != semanticEdges[j].ToPath {
			return semanticEdges[i].ToPath < semanticEdges[j].ToPath
		}
		return semanticEdges[i].Reason < semanticEdges[j].Reason
	})

	payload := runtimeShardPlanArtifact{
		Version:       1,
		RunID:         e.runID,
		StepID:        stepID,
		DomainID:      strings.TrimSpace(domainID),
		Strategy:      strings.TrimSpace(strategy),
		MaxParallel:   maxParallel,
		FailurePolicy: strings.TrimSpace(failurePolicy),
		ShardMode:     strings.TrimSpace(e.executionProfile.ShardMode),
		PlannerNotes:  normalizedWarnings,
		SemanticGraph: semanticEdges,
		Items:         items,
	}
	if payload.Strategy == "" {
		payload.Strategy = "sequential"
	}
	if payload.MaxParallel <= 0 {
		payload.MaxParallel = 1
	}
	if payload.FailurePolicy == "" {
		payload.FailurePolicy = "best_effort"
	}
	if payload.ShardMode == "" {
		payload.ShardMode = "heuristics"
	}

	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	path := shardPlanPath(e.runID, stepID, domainID)
	if err := e.workspace.WriteFile(path, content); err != nil {
		return err
	}
	e.addArtifacts(Artifact{Path: path, Kind: "taskrun", Label: shardPlanLabel(stepID, domainID)})
	e.logInfo(stepID, domainID, "runtime shard plan persisted", map[string]any{
		"path":   path,
		"shards": len(plans),
	})
	return nil
}

func (e *pipelineExecution) persistShardSummary(stepID string, domainID string, items []runtimeShardSummaryEntry) error {
	summary := runtimeShardSummary{
		Version:       1,
		RunID:         e.runID,
		StepID:        stepID,
		DomainID:      strings.TrimSpace(domainID),
		Strategy:      strings.TrimSpace(e.executionProfile.Strategy),
		MaxParallel:   e.executionProfile.MaxParallel,
		FailurePolicy: strings.TrimSpace(e.executionProfile.FailurePolicy),
		ShardMode:     strings.TrimSpace(e.executionProfile.ShardMode),
		GeneratedAt:   e.clock().UTC().Format(time.RFC3339),
		Items:         append([]runtimeShardSummaryEntry(nil), items...),
	}
	if summary.Strategy == "" {
		summary.Strategy = "sequential"
	}
	if summary.MaxParallel <= 0 {
		summary.MaxParallel = 1
	}
	if summary.FailurePolicy == "" {
		summary.FailurePolicy = "best_effort"
	}
	if summary.ShardMode == "" {
		summary.ShardMode = "heuristics"
	}

	content, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	path := shardSummaryPath(e.runID, stepID, domainID)
	if err := e.workspace.WriteFile(path, content); err != nil {
		return err
	}
	e.addArtifacts(Artifact{Path: path, Kind: "taskrun", Label: shardSummaryLabel(stepID, domainID)})
	e.logInfo(stepID, domainID, "runtime shard summary persisted", map[string]any{"path": path, "items": len(items)})
	return nil
}

func shardTaskrunPath(runID string, stepID string, domainID string, shardID string, singleShard bool) string {
	if singleShard {
		return singleShardTaskrunPath(runID, stepID, domainID)
	}
	stepSlug := strings.ReplaceAll(stepID, ".", "-")
	shardSlug := sanitizeDomainArtifactSlug(shardID)
	if strings.TrimSpace(shardSlug) == "" {
		shardSlug = "shard"
	}
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("reports/taskruns/%s-%s-domain-%s-shard-%s.json", runID, stepSlug, sanitizeDomainArtifactSlug(domainID), shardSlug)
	}
	return fmt.Sprintf("reports/taskruns/%s-%s-shard-%s.json", runID, stepSlug, shardSlug)
}

func singleShardTaskrunPath(runID string, stepID string, domainID string) string {
	stepSlug := strings.ReplaceAll(stepID, ".", "-")
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("reports/taskruns/%s-%s-domain-%s.json", runID, stepSlug, sanitizeDomainArtifactSlug(domainID))
	}
	return fmt.Sprintf("reports/taskruns/%s-%s.json", runID, stepSlug)
}

func shardTaskrunLabel(stepID string, domainID string, shardID string, singleShard bool) string {
	if singleShard {
		if strings.TrimSpace(domainID) != "" {
			return fmt.Sprintf("%s.%s", stepID, domainID)
		}
		return stepID
	}
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("%s.%s.%s", stepID, domainID, shardID)
	}
	return fmt.Sprintf("%s.%s", stepID, shardID)
}

func shardSummaryPath(runID string, stepID string, domainID string) string {
	stepSlug := strings.ReplaceAll(stepID, ".", "-")
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("reports/taskruns/%s-%s-shard-summary-%s.json", runID, stepSlug, sanitizeDomainArtifactSlug(domainID))
	}
	return fmt.Sprintf("reports/taskruns/%s-%s-shard-summary.json", runID, stepSlug)
}

func shardSummaryLabel(stepID string, domainID string) string {
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("%s.%s.shards", stepID, domainID)
	}
	return fmt.Sprintf("%s.shards", stepID)
}

func shardPlanPath(runID string, stepID string, domainID string) string {
	stepSlug := strings.ReplaceAll(stepID, ".", "-")
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("reports/taskruns/%s-%s-shard-plan-%s.json", runID, stepSlug, sanitizeDomainArtifactSlug(domainID))
	}
	return fmt.Sprintf("reports/taskruns/%s-%s-shard-plan.json", runID, stepSlug)
}

func shardPlanLabel(stepID string, domainID string) string {
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("%s.%s.shard-plan", stepID, domainID)
	}
	return fmt.Sprintf("%s.shard-plan", stepID)
}

func buildShardTaskSuffix(prefix string, shardID string) string {
	if strings.TrimSpace(prefix) == "" {
		return "shard-" + sanitizeDomainArtifactSlug(shardID)
	}
	return strings.TrimSpace(prefix) + "-shard-" + sanitizeDomainArtifactSlug(shardID)
}

func (e *pipelineExecution) planRuntimeShards(repoScopes []string) ([]runtimeShardPlan, []string, []runtimeShardPlanGraphEdge) {
	scopes := normalizeOrderedUniqueStrings(repoScopes)
	if len(scopes) == 0 {
		return []runtimeShardPlan{{
			SortKey:     "workspace:.",
			ShardID:     "workspace-root",
			RepoScopes:  nil,
			PathScopes:  []string{"."},
			PrimaryRepo: "workspace",
		}}, nil, nil
	}

	warnings := []string{}
	plans := []runtimeShardPlan{}
	graphEdges := []runtimeShardPlanGraphEdge{}
	seenShardIDs := map[string]int{}

	for _, scope := range scopes {
		paths, pathWarnings := e.planScopePaths(scope)
		warnings = append(warnings, pathWarnings...)
		if len(paths) == 0 {
			paths = []string{"."}
		}

		groups := make([][]string, 0, len(paths))
		mode := strings.TrimSpace(strings.ToLower(e.executionProfile.ShardMode))
		if mode == "" {
			mode = "heuristics"
		}
		if mode == "semantic" {
			repoPath := strings.TrimSpace(e.resolvedRepoPaths[scope])
			semanticGroups, semanticWarnings, semanticEdges := discoverSemanticShardGroups(repoPath, paths)
			warnings = append(warnings, semanticWarnings...)
			if len(semanticGroups) > 0 {
				groups = semanticGroups
			}
			for _, edge := range semanticEdges {
				graphEdges = append(graphEdges, runtimeShardPlanGraphEdge{
					RepoScope: scope,
					FromPath:  edge.FromPath,
					ToPath:    edge.ToPath,
					Reason:    edge.Reason,
				})
			}
		}
		if len(groups) == 0 {
			for _, pathScope := range paths {
				groups = append(groups, []string{pathScope})
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
				shardID = fmt.Sprintf("%s-%d", baseID, sequence+1)
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
	return plans, warnings, graphEdges
}

func buildShardID(scope string, pathScopes []string) string {
	parts := append([]string{scope}, pathScopes...)
	joined := strings.Join(parts, "-")
	slug := slugutil.Slugify(joined)
	if strings.TrimSpace(slug) == "" {
		return "shard"
	}
	return slug
}

func (e *pipelineExecution) planScopePaths(scope string) ([]string, []string) {
	warnings := []string{}
	repo, ok := lookupManifestRepo(e.workspace.Manifest.Repos, scope)
	if !ok {
		return []string{"."}, []string{fmt.Sprintf("repo scope %q is not present in workspace manifest; fallback shard='.'", scope)}
	}

	repoPath := strings.TrimSpace(e.resolvedRepoPaths[scope])
	if repoPath == "" && strings.TrimSpace(repo.Path) != "" {
		repoPath = strings.TrimSpace(repo.Path)
		if !filepath.IsAbs(repoPath) {
			repoPath = filepath.Join(e.workspace.Path, repoPath)
		}
	}
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

func discoverHeuristicShardPaths(repoPath string) ([]string, error) {
	result, err := discoverHeuristicShardPathsWithMeta(repoPath)
	if err != nil {
		return nil, err
	}
	return result.Paths, nil
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

	candidates := map[string]struct{}{}
	err = filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
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
		rel = normalizeShardPath(rel)
		candidates[rel] = struct{}{}
		return nil
	})
	if err != nil {
		return heuristicShardDiscoveryResult{}, err
	}
	if len(candidates) == 0 {
		return heuristicShardDiscoveryResult{
			Paths:             []string{"."},
			FallbackNoMarkers: true,
		}, nil
	}
	paths := make([]string, 0, len(candidates))
	for candidate := range candidates {
		paths = append(paths, candidate)
	}
	return heuristicShardDiscoveryResult{Paths: pruneParentShardPaths(paths)}, nil
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
	candidate = normalizeShardPath(candidate)
	pattern = normalizeShardPath(pattern)
	if pattern == "." {
		return candidate == "."
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return candidate == prefix || strings.HasPrefix(candidate, prefix+"/")
	}
	if candidate == pattern {
		return true
	}
	matched, err := path.Match(pattern, candidate)
	if err == nil && matched {
		return true
	}
	if strings.Contains(pattern, "*") {
		return false
	}
	return strings.HasPrefix(candidate, pattern+"/")
}

func discoverSemanticShardGroups(repoPath string, paths []string) ([][]string, []string, []runtimeShardPlanGraphEdge) {
	normalized := normalizeAndSortShardPaths(paths)
	if len(normalized) <= 1 {
		groups := make([][]string, 0, len(normalized))
		for _, value := range normalized {
			groups = append(groups, []string{value})
		}
		return groups, nil, nil
	}
	if strings.TrimSpace(repoPath) == "" {
		groups := make([][]string, 0, len(normalized))
		for _, value := range normalized {
			groups = append(groups, []string{value})
		}
		return groups, []string{"semantic shard discovery fallback: repo path unavailable, using heuristic shard groups"}, nil
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

	parent := make([]int, len(normalized))
	for idx := range parent {
		parent[idx] = idx
	}
	graphEdges := make([]runtimeShardPlanGraphEdge, 0, len(normalized))
	find := func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	union := func(a int, b int) {
		ra := find(a)
		rb := find(b)
		if ra == rb {
			return
		}
		if ra < rb {
			parent[rb] = ra
			return
		}
		parent[ra] = rb
	}

	for left := 0; left < len(normalized); left++ {
		for right := left + 1; right < len(normalized); right++ {
			if related, reason := semanticRootsRelated(normalized[left], normalized[right], corpora[left], corpora[right]); related {
				graphEdges = append(graphEdges, runtimeShardPlanGraphEdge{
					FromPath: normalized[left],
					ToPath:   normalized[right],
					Reason:   reason,
				})
				union(left, right)
			}
		}
	}

	groupsMap := map[int][]string{}
	for idx, rel := range normalized {
		root := find(idx)
		groupsMap[root] = append(groupsMap[root], rel)
	}
	groups := make([][]string, 0, len(groupsMap))
	for _, group := range groupsMap {
		sort.Strings(group)
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool {
		return strings.Join(groups[i], "|") < strings.Join(groups[j], "|")
	})
	sort.Slice(graphEdges, func(i, j int) bool {
		if graphEdges[i].FromPath != graphEdges[j].FromPath {
			return graphEdges[i].FromPath < graphEdges[j].FromPath
		}
		if graphEdges[i].ToPath != graphEdges[j].ToPath {
			return graphEdges[i].ToPath < graphEdges[j].ToPath
		}
		return graphEdges[i].Reason < graphEdges[j].Reason
	})
	return groups, warnings, graphEdges
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
	if info, err := os.Stat(abs); err != nil {
		return "", err
	} else if !info.IsDir() {
		return "", fmt.Errorf("path %q is not a directory", abs)
	}

	parts := make([]string, 0, 32)
	err := filepath.WalkDir(abs, func(current string, entry fs.DirEntry, walkErr error) error {
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
