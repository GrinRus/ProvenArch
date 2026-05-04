package orchestrator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

func (e *pipelineExecution) loadRuntimeShardSummaryState(
	stepID string,
	domainID string,
	plans []runtimeShardPlan,
	singleShard bool,
) (*runtimeShardSummaryState, error) {
	store := defaultShardSummaryStore{execution: e}
	existingEntries, err := store.LoadSummary(stepID, domainID)
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
		store:       store,
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
		if taskrunPath != "" && store.RuntimeExecutionExists(taskrunPath) {
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

func (store defaultShardSummaryStore) LoadSummary(stepID string, domainID string) ([]runtimeShardSummaryEntry, error) {
	if store.execution == nil {
		return nil, fmt.Errorf("shard summary store is not configured")
	}
	content, err := store.execution.workspace.ReadFile(shardSummaryPath(store.execution.runID, stepID, domainID))
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

func (store defaultShardSummaryStore) PersistSummary(stepID string, domainID string, items []runtimeShardSummaryEntry) error {
	if store.execution == nil {
		return fmt.Errorf("shard summary store is not configured")
	}
	return store.execution.persistShardSummary(stepID, domainID, items)
}

func (store defaultShardSummaryStore) RuntimeExecutionExists(path string) bool {
	if store.execution == nil {
		return false
	}
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := store.execution.workspace.ReadFile(path)
	return err == nil
}

func (store defaultShardSummaryStore) PersistRuntimeExecutionArtifact(path string, label string, raw []byte) error {
	if store.execution == nil {
		return fmt.Errorf("shard summary store is not configured")
	}
	return store.execution.persistRuntimeExecutionArtifact(path, label, raw)
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
	if s.store == nil {
		return fmt.Errorf("shard summary store is not configured")
	}
	return s.store.PersistSummary(s.stepID, s.domainID, append([]runtimeShardSummaryEntry(nil), s.entries...))
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
	if s.store == nil {
		return fmt.Errorf("shard summary store is not configured")
	}
	if err := s.store.PersistRuntimeExecutionArtifact(taskrunPath, taskrunLabel, raw); err != nil {
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
