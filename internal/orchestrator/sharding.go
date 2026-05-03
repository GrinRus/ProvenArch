package orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
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
	Plan           runtimeShardPlan
	Prepared       runtimePreparedExecution
	Err            error
	AlreadyApplied bool
	FromCheckpoint bool
}

type runtimeShardExecutionOptions struct {
	Strategy      string
	MaxParallel   int
	FailurePolicy string
	BestEffort    bool
}

type runtimeShardSummary struct {
	Version       int                        `json:"version"`
	Meta          runtimeArtifactMeta        `json:"meta,omitempty"`
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

type runtimeShardSummaryState struct {
	execution   *pipelineExecution
	stepID      string
	domainID    string
	singleShard bool
	entries     []runtimeShardSummaryEntry
	index       map[string]int
	mu          sync.Mutex
}

type runtimeShardPlanArtifact struct {
	Version       int                            `json:"version"`
	Meta          runtimeArtifactMeta            `json:"meta,omitempty"`
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

type runtimeArtifactMeta struct {
	Runtime contracts.RuntimeMeta `json:"runtime"`
}

type heuristicShardDiscoveryResult struct {
	Paths             []string
	FallbackNoMarkers bool
}

const (
	maxAutoShardsPerRepo     = 16
	maxRuntimeShardIDLength  = 96
	runtimeShardIDHashLength = 12
)

func runtimeMetaForRunner(runner acpruntime.Runner) contracts.RuntimeMeta {
	if metadataRunner, ok := runner.(acpruntime.MetadataRunner); ok {
		meta := metadataRunner.RuntimeMeta()
		if strings.TrimSpace(meta.Name) != "" {
			return meta
		}
	}
	return contracts.RuntimeMeta{Name: "unknown"}
}

func (e *pipelineExecution) runtimeMetaForStep(stepID string) contracts.RuntimeMeta {
	if e.runnerResolver != nil {
		provider, runner, err := e.runnerResolver.RunnerForStep(stepID)
		if err == nil {
			meta := runtimeMetaForRunner(runner)
			if strings.TrimSpace(meta.Name) == "" || meta.Name == "unknown" {
				if provider != "" {
					meta.Name = string(provider)
				}
			}
			return meta
		}
	}
	if provider := e.stepProviders.ProviderForStep(stepID); provider != "" {
		return contracts.RuntimeMeta{Name: string(provider)}
	}
	return contracts.RuntimeMeta{Name: "unknown"}
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
) ([]runtimeTaskExecution, runtimeShardOutcome, error) {
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

	options := normalizeRuntimeShardExecutionOptions(e.executionProfile)
	if err := e.persistShardPlan(stepID, domainID, plans, plannerWarnings, semanticGraph, options.Strategy, options.MaxParallel, options.FailurePolicy); err != nil {
		return nil, runtimeShardOutcome{}, err
	}

	singleShard := len(plans) == 1
	summaryState, err := e.loadRuntimeShardSummaryState(stepID, domainID, plans, singleShard)
	if err != nil {
		return nil, runtimeShardOutcome{}, err
	}

	e.logInfo(stepID, domainID, "runtime shard execution prepared", map[string]any{
		"shards":         len(plans),
		"strategy":       options.Strategy,
		"max_parallel":   options.MaxParallel,
		"failure_policy": options.FailurePolicy,
		"shard_mode":     e.executionProfile.ShardMode,
	})

	results, terminalErr := e.scheduleRuntimeShardRuns(ctx, stepID, domainID, plans, summaryState, options, taskSuffixPrefix)
	if terminalErr != nil && !options.BestEffort {
		e.logWarn(stepID, domainID, "runtime shard scheduler stopped after terminal failure", map[string]any{
			"error": strings.TrimSpace(terminalErr.Error()),
		})
	}

	executions := make([]runtimeTaskExecution, 0, len(plans))
	outcome := runtimeShardOutcome{PlannedShards: len(plans)}

	for _, result := range results {
		if result.Err == nil && result.Prepared.Task.TaskID == "" && terminalErr != nil && !options.BestEffort {
			if err := summaryState.markAborted(result.Plan); err != nil {
				return nil, runtimeShardOutcome{}, err
			}
			outcome.FailedShards++
			continue
		}
		if result.Err != nil {
			outcome.FailedShards++
			if options.BestEffort {
				e.registerPartialShardFailure(stepID, domainID, result.Plan, result.Err)
				continue
			}
			if terminalErr == nil {
				terminalErr = result.Err
			}
			continue
		}

		var execution runtimeTaskExecution
		if result.AlreadyApplied && !strings.HasSuffix(stepID, "step3.findings") {
			execution, err = e.replayRuntimeTaskExecution(stepID, domainID, result.Prepared)
		} else {
			execution, err = e.applyRuntimeTaskExecution(stepID, domainID, result.Prepared)
		}
		if err != nil {
			outcome.FailedShards++
			if markErr := summaryState.markFailed(result.Plan, result.Prepared.Task.TaskID, strings.TrimSpace(err.Error())); markErr != nil {
				return nil, runtimeShardOutcome{}, markErr
			}
			if options.BestEffort {
				e.registerPartialShardFailure(stepID, domainID, result.Plan, err)
				continue
			}
			if terminalErr == nil {
				terminalErr = err
			}
			continue
		}

		taskrunPath := shardTaskrunPath(e.runID, stepID, domainID, result.Plan.ShardID, summaryState.singleShard)
		if err := summaryState.markSucceeded(result.Plan, result.Prepared.Task.TaskID, taskrunPath); err != nil {
			return nil, runtimeShardOutcome{}, err
		}
		if result.FromCheckpoint {
			e.logInfo(stepID, domainID, "shard replayed from checkpoint", map[string]any{
				"shard_id":     result.Plan.ShardID,
				"task_id":      result.Prepared.Task.TaskID,
				"taskrun_path": taskrunPath,
			})
		}
		outcome.SucceededShards++
		executions = append(executions, execution)
	}

	if terminalErr != nil {
		return nil, outcome, terminalErr
	}
	return executions, outcome, nil
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

func normalizeRuntimeShardExecutionOptions(profile acpruntime.ExecutionValues) runtimeShardExecutionOptions {
	strategy := strings.TrimSpace(profile.Strategy)
	if strategy == "" {
		strategy = "sequential"
	}
	maxParallel := profile.MaxParallel
	if maxParallel <= 0 {
		maxParallel = 1
	}
	if strategy != "parallel" {
		maxParallel = 1
	}
	failurePolicy := strings.TrimSpace(profile.FailurePolicy)
	if failurePolicy == "" {
		failurePolicy = "best_effort"
	}
	return runtimeShardExecutionOptions{
		Strategy:      strategy,
		MaxParallel:   maxParallel,
		FailurePolicy: failurePolicy,
		BestEffort:    failurePolicy == "best_effort",
	}
}

func (e *pipelineExecution) scheduleRuntimeShardRuns(
	ctx context.Context,
	stepID string,
	domainID string,
	plans []runtimeShardPlan,
	summaryState *runtimeShardSummaryState,
	options runtimeShardExecutionOptions,
	taskSuffixPrefix string,
) ([]runtimeShardRunResult, error) {
	results := make([]runtimeShardRunResult, len(plans))
	for idx, plan := range plans {
		results[idx].Plan = plan
	}
	if len(plans) == 0 {
		return results, nil
	}

	workerCount := options.MaxParallel
	if workerCount <= 0 {
		workerCount = 1
	}
	if workerCount > len(plans) {
		workerCount = len(plans)
	}

	runCtx := ctx
	cancel := func() {}
	if !options.BestEffort {
		runCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	nextIndex := 0

	recordResult := func(index int, result runtimeShardRunResult) {
		mu.Lock()
		defer mu.Unlock()
		results[index] = result
		if result.Err != nil && !options.BestEffort && firstErr == nil {
			firstErr = result.Err
			cancel()
		}
	}
	nextJob := func() (int, runtimeShardPlan, bool) {
		mu.Lock()
		defer mu.Unlock()
		if firstErr != nil && !options.BestEffort {
			return 0, runtimeShardPlan{}, false
		}
		if nextIndex >= len(plans) {
			return 0, runtimeShardPlan{}, false
		}
		index := nextIndex
		nextIndex++
		return index, plans[index], true
	}

	for worker := 0; worker < workerCount; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				index, plan, ok := nextJob()
				if !ok {
					return
				}
				result := e.runRuntimeShard(runCtx, stepID, domainID, plan, summaryState, taskSuffixPrefix)
				recordResult(index, result)
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	return results, firstErr
}

func (e *pipelineExecution) runRuntimeShard(
	ctx context.Context,
	stepID string,
	domainID string,
	plan runtimeShardPlan,
	summaryState *runtimeShardSummaryState,
	taskSuffixPrefix string,
) runtimeShardRunResult {
	entry := summaryState.entry(plan.ShardID)
	if handled, result := e.loadReplayableShardResult(stepID, domainID, plan, entry, summaryState.singleShard); handled {
		if result.Err != nil {
			taskID := strings.TrimSpace(entry.TaskID)
			if markErr := summaryState.markFailed(plan, taskID, strings.TrimSpace(result.Err.Error())); markErr != nil {
				result.Err = markErr
			}
		}
		return result
	}
	if entry.Status == "failed" {
		return runtimeShardRunResult{
			Plan: plan,
			Err:  fmt.Errorf("%s", shardFailureMessage(entry)),
		}
	}

	taskSuffix := buildShardTaskSuffix(taskSuffixPrefix, plan.ShardID)
	prepared, err := e.runRuntimeTaskNormalized(ctx, stepID, taskSuffix, plan.RepoScopes, plan.PathScopes, domainID, plan.ShardID)
	if err == nil {
		taskrunPath := shardTaskrunPath(e.runID, stepID, domainID, plan.ShardID, summaryState.singleShard)
		taskrunLabel := shardTaskrunLabel(stepID, domainID, plan.ShardID, summaryState.singleShard)
		if checkpointErr := summaryState.markCheckpointed(plan, prepared.Task.TaskID, taskrunPath, taskrunLabel, prepared.ExecutionRaw); checkpointErr != nil {
			err = checkpointErr
		}
	}
	if err != nil {
		message := strings.TrimSpace(err.Error())
		if markErr := summaryState.markFailed(plan, prepared.Task.TaskID, message); markErr != nil {
			err = markErr
		}
	}
	return runtimeShardRunResult{Plan: plan, Prepared: prepared, Err: err}
}

func (e *pipelineExecution) loadRuntimeShardSummaryState(
	stepID string,
	domainID string,
	plans []runtimeShardPlan,
	singleShard bool,
) (*runtimeShardSummaryState, error) {
	existingEntries, err := e.loadPersistedShardSummaryEntries(stepID, domainID)
	if err != nil {
		return nil, err
	}
	existingByShard := make(map[string]runtimeShardSummaryEntry, len(existingEntries))
	for _, entry := range existingEntries {
		if strings.TrimSpace(entry.ShardID) == "" {
			continue
		}
		existingByShard[entry.ShardID] = entry
	}

	state := &runtimeShardSummaryState{
		execution:   e,
		stepID:      stepID,
		domainID:    domainID,
		singleShard: singleShard,
		entries:     make([]runtimeShardSummaryEntry, 0, len(plans)),
		index:       make(map[string]int, len(plans)),
	}
	for _, plan := range plans {
		entry := runtimeShardSummaryEntry{
			ShardID:    plan.ShardID,
			RepoScopes: append([]string(nil), plan.RepoScopes...),
			PathScopes: append([]string(nil), plan.PathScopes...),
			Status:     "pending",
		}
		if existing, ok := existingByShard[plan.ShardID]; ok {
			entry.Status = normalizeShardSummaryStatus(existing.Status)
			entry.TaskID = strings.TrimSpace(existing.TaskID)
			entry.TaskRun = strings.TrimSpace(existing.TaskRun)
			entry.Error = strings.TrimSpace(existing.Error)
		}
		taskrunPath := entry.TaskRun
		if taskrunPath == "" {
			taskrunPath = shardTaskrunPath(e.runID, stepID, domainID, plan.ShardID, singleShard)
		}
		if taskrunPath != "" && e.runtimeExecutionExists(taskrunPath) {
			entry.TaskRun = taskrunPath
			if entry.Status == "pending" {
				entry.Status = "checkpointed"
				entry.Error = ""
			}
		}
		state.index[plan.ShardID] = len(state.entries)
		state.entries = append(state.entries, entry)
	}
	if err := state.persist(); err != nil {
		return nil, err
	}
	return state, nil
}

func (e *pipelineExecution) loadPersistedShardSummaryEntries(stepID string, domainID string) ([]runtimeShardSummaryEntry, error) {
	content, err := e.workspace.ReadFile(shardSummaryPath(e.runID, stepID, domainID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var summary runtimeShardSummary
	if err := json.Unmarshal(content, &summary); err != nil {
		return nil, fmt.Errorf("decode persisted shard summary: %w", err)
	}
	return append([]runtimeShardSummaryEntry(nil), summary.Items...), nil
}

func (e *pipelineExecution) runtimeExecutionExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := e.workspace.ReadFile(path)
	return err == nil
}

func normalizeShardSummaryStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "checkpointed", "succeeded", "failed":
		return strings.TrimSpace(status)
	default:
		return "pending"
	}
}

func shardFailureMessage(entry runtimeShardSummaryEntry) string {
	message := strings.TrimSpace(entry.Error)
	if message == "" {
		message = "shard failed in previous attempt"
	}
	return message
}

func (e *pipelineExecution) loadReplayableShardResult(
	stepID string,
	domainID string,
	plan runtimeShardPlan,
	entry runtimeShardSummaryEntry,
	singleShard bool,
) (bool, runtimeShardRunResult) {
	status := normalizeShardSummaryStatus(entry.Status)
	if status != "checkpointed" && status != "succeeded" {
		return false, runtimeShardRunResult{}
	}
	taskrunPath := strings.TrimSpace(entry.TaskRun)
	if taskrunPath == "" {
		taskrunPath = shardTaskrunPath(e.runID, stepID, domainID, plan.ShardID, singleShard)
	}
	raw, err := e.workspace.ReadFile(taskrunPath)
	if err != nil {
		return true, runtimeShardRunResult{
			Plan: plan,
			Err:  fmt.Errorf("load persisted runtime execution %q: %w", taskrunPath, err),
		}
	}
	prepared, err := loadPreparedExecutionFromPersistedRuntimeExecution(raw)
	if err != nil {
		return true, runtimeShardRunResult{
			Plan: plan,
			Err:  fmt.Errorf("decode persisted runtime execution %q: %w", taskrunPath, err),
		}
	}
	prepared.Task.StepID = stepID
	prepared.Task.DomainID = strings.TrimSpace(domainID)
	if strings.TrimSpace(prepared.Task.ShardID) == "" {
		prepared.Task.ShardID = strings.TrimSpace(plan.ShardID)
	}
	if len(prepared.Task.RepoScopes) == 0 {
		prepared.Task.RepoScopes = append([]string(nil), plan.RepoScopes...)
	}
	if len(prepared.Task.PathScopes) == 0 {
		prepared.Task.PathScopes = append([]string(nil), plan.PathScopes...)
	}
	if strings.TrimSpace(prepared.Task.RepoScope) == "" {
		prepared.Task.RepoScope = strings.TrimSpace(plan.PrimaryRepo)
	}
	if strings.TrimSpace(prepared.Task.Workspace) == "" {
		prepared.Task.Workspace = e.workspace.Path
	}
	artifactRoot, writeRoot, draftFinalRoot, readContextRoots, artifactErr := e.runtimeArtifactContext(stepID, prepared.Task.ShardID, prepared.Task.RepoScopes)
	if artifactErr != nil {
		return true, runtimeShardRunResult{
			Plan: plan,
			Err:  fmt.Errorf("prepare replay artifact context: %w", artifactErr),
		}
	}
	prepared.Task.ArtifactRoot = artifactRoot
	prepared.Task.WriteRoot = writeRoot
	prepared.Task.DraftFinalRoot = draftFinalRoot
	prepared.Task.ReadContextRoots = append([]string(nil), readContextRoots...)
	prepared.Task.AgentRole = runtimeAgentRole(stepID)
	prepared.Task.StepContract = runtimeStepContract(stepID)
	prepared.Task.ExpectedArtifacts = append([]string(nil), runtimeExpectedArtifacts(stepID)...)
	e.logInfo(stepID, domainID, "shard loaded from persisted runtime execution", map[string]any{
		"shard_id":     plan.ShardID,
		"task_id":      prepared.Task.TaskID,
		"taskrun_path": taskrunPath,
		"status":       status,
	})
	return true, runtimeShardRunResult{
		Plan:           plan,
		Prepared:       prepared,
		AlreadyApplied: status == "succeeded",
		FromCheckpoint: status == "checkpointed",
	}
}

func (s *runtimeShardSummaryState) entry(shardID string) runtimeShardSummaryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx, ok := s.index[shardID]; ok {
		return s.entries[idx]
	}
	return runtimeShardSummaryEntry{ShardID: shardID, Status: "pending"}
}

func (s *runtimeShardSummaryState) persist() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.persistLocked()
}

func (s *runtimeShardSummaryState) persistLocked() error {
	return s.execution.persistShardSummary(s.stepID, s.domainID, append([]runtimeShardSummaryEntry(nil), s.entries...))
}

func (s *runtimeShardSummaryState) markCheckpointed(
	plan runtimeShardPlan,
	taskID string,
	taskrunPath string,
	taskrunLabel string,
	raw []byte,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.execution.persistRuntimeExecutionArtifact(taskrunPath, taskrunLabel, raw); err != nil {
		return err
	}
	s.updateLocked(plan, "checkpointed", taskID, taskrunPath, "")
	return s.persistLocked()
}

func (s *runtimeShardSummaryState) markSucceeded(plan runtimeShardPlan, taskID string, taskrunPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateLocked(plan, "succeeded", taskID, taskrunPath, "")
	return s.persistLocked()
}

func (s *runtimeShardSummaryState) markFailed(plan runtimeShardPlan, taskID string, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateLocked(plan, "failed", taskID, "", message)
	return s.persistLocked()
}

func (s *runtimeShardSummaryState) markAborted(plan runtimeShardPlan) error {
	return s.markFailed(plan, "", "shard not executed because fail_fast aborted remaining work")
}

func (s *runtimeShardSummaryState) updateLocked(
	plan runtimeShardPlan,
	status string,
	taskID string,
	taskrunPath string,
	message string,
) {
	idx, ok := s.index[plan.ShardID]
	if !ok {
		return
	}
	entry := s.entries[idx]
	entry.RepoScopes = append([]string(nil), plan.RepoScopes...)
	entry.PathScopes = append([]string(nil), plan.PathScopes...)
	entry.Status = normalizeShardSummaryStatus(status)
	if strings.TrimSpace(taskID) != "" {
		entry.TaskID = strings.TrimSpace(taskID)
	}
	if strings.TrimSpace(taskrunPath) != "" {
		entry.TaskRun = strings.TrimSpace(taskrunPath)
	}
	if strings.TrimSpace(message) != "" {
		entry.Error = strings.TrimSpace(message)
	} else {
		entry.Error = ""
	}
	s.entries[idx] = entry
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
		Meta:          runtimeArtifactMeta{Runtime: e.runtimeMetaForStep(stepID)},
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
	normalizedItems, err := normalizeAndValidateShardSummaryItems(items)
	if err != nil {
		return err
	}
	summary := runtimeShardSummary{
		Version:       1,
		Meta:          runtimeArtifactMeta{Runtime: e.runtimeMetaForStep(stepID)},
		RunID:         e.runID,
		StepID:        stepID,
		DomainID:      strings.TrimSpace(domainID),
		Strategy:      strings.TrimSpace(e.executionProfile.Strategy),
		MaxParallel:   e.executionProfile.MaxParallel,
		FailurePolicy: strings.TrimSpace(e.executionProfile.FailurePolicy),
		ShardMode:     strings.TrimSpace(e.executionProfile.ShardMode),
		GeneratedAt:   e.clock().UTC().Format(time.RFC3339),
		Items:         normalizedItems,
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

func normalizeAndValidateShardSummaryItems(items []runtimeShardSummaryEntry) ([]runtimeShardSummaryEntry, error) {
	normalized := make([]runtimeShardSummaryEntry, 0, len(items))
	for _, entry := range items {
		candidate := entry
		candidate.ShardID = strings.TrimSpace(candidate.ShardID)
		candidate.TaskID = strings.TrimSpace(candidate.TaskID)
		candidate.TaskRun = strings.TrimSpace(candidate.TaskRun)
		candidate.Error = strings.TrimSpace(candidate.Error)
		candidate.Status = normalizeShardSummaryStatus(candidate.Status)
		if (candidate.Status == "checkpointed" || candidate.Status == "succeeded") && candidate.TaskRun == "" {
			shardID := candidate.ShardID
			if shardID == "" {
				shardID = "<unknown>"
			}
			return nil, fmt.Errorf("invalid shard summary: shard %q status %q requires taskrun_path", shardID, candidate.Status)
		}
		normalized = append(normalized, candidate)
	}
	return normalized, nil
}

func shardTaskrunPath(runID string, stepID string, domainID string, shardID string, singleShard bool) string {
	_ = domainID
	_ = singleShard
	return runtimeExecutionMetadataPath(runID, stepID, shardID)
}

func singleShardTaskrunPath(runID string, stepID string, domainID string) string {
	_ = domainID
	return runtimeExecutionMetadataPath(runID, stepID, "")
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

		repoPath := e.resolveRepoPath(scope)
		groups, groupingWarnings := buildStructuralShardGroups(repoPath, paths)
		warnings = append(warnings, groupingWarnings...)
		if len(groups) == 0 {
			groups = make([][]string, 0, len(paths))
			for _, pathScope := range paths {
				groups = append(groups, []string{pathScope})
			}
		}
		mode := strings.TrimSpace(strings.ToLower(e.executionProfile.ShardMode))
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
	return plans, warnings, graphEdges
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

func (e *pipelineExecution) resolveRepoPath(scope string) string {
	repoPath := strings.TrimSpace(e.resolvedRepoPaths[scope])
	repo, ok := lookupManifestRepo(e.workspace.Manifest.Repos, scope)
	if !ok {
		return repoPath
	}
	if repoPath == "" && strings.TrimSpace(repo.Path) != "" {
		repoPath = strings.TrimSpace(repo.Path)
		if !filepath.IsAbs(repoPath) {
			repoPath = filepath.Join(e.workspace.Path, repoPath)
		}
	}
	return strings.TrimSpace(repoPath)
}

func (e *pipelineExecution) planScopePaths(scope string) ([]string, []string) {
	warnings := []string{}
	repo, ok := lookupManifestRepo(e.workspace.Manifest.Repos, scope)
	if !ok {
		return []string{"."}, []string{fmt.Sprintf("repo scope %q is not present in workspace manifest; fallback shard='.'", scope)}
	}

	repoPath := e.resolveRepoPath(scope)
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
		warnings = append(
			warnings,
			fmt.Sprintf(
				"structural shard coalescing kept %d shard groups because cross-top-level merges are forbidden (target cap=%d)",
				len(groups),
				maxAutoShardsPerRepo,
			),
		)
	}
	return groups, warnings
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
