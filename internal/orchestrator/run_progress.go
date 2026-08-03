package orchestrator

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type runProgressTracker struct {
	mu        sync.Mutex
	pipeline  Pipeline
	startedAt time.Time
	progress  *RunProgress
	plans     map[string]int
	running   map[string]struct{}
	succeeded map[string]struct{}
	failed    map[string]struct{}
	scopes    map[string]struct{}
}

func newRunProgressTracker(pipeline Pipeline, startedAt time.Time, stepID string) *runProgressTracker {
	tracker := &runProgressTracker{pipeline: pipeline, startedAt: startedAt}
	tracker.beginStep(stepID, nil, startedAt)
	return tracker
}

func (t *runProgressTracker) beginStep(stepID string, scopes []string, now time.Time) *RunProgress {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.progress = runningStepProgress(t.pipeline, t.startedAt, stepID, scopes, now)
	t.plans = map[string]int{}
	t.running = map[string]struct{}{}
	t.succeeded = map[string]struct{}{}
	t.failed = map[string]struct{}{}
	t.scopes = map[string]struct{}{}
	for _, scope := range scopes {
		if scope = strings.TrimSpace(scope); scope != "" {
			t.scopes[scope] = struct{}{}
		}
	}
	t.applyCounts()
	return cloneRunProgress(t.progress)
}

func (t *runProgressTracker) observe(entry RunLogEntry) *RunProgress {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.progress == nil {
		t.progress = runningStepProgress(t.pipeline, t.startedAt, entry.StepID, nil, entry.Timestamp)
	}
	now := entry.Timestamp.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	t.progress.LastActivityAt = now.Format(time.RFC3339)
	if scope := strings.TrimSpace(entry.DomainID); scope != "" {
		t.scopes[scope] = struct{}{}
	}
	message := strings.ToLower(strings.TrimSpace(entry.Message))
	unit := progressUnitID(entry.Fields)
	meaningful := false
	if limit := progressInt(entry.Fields["repair_limit"]); limit > 0 {
		t.progress.RepairLimit = limit
	}
	if deadline := progressString(entry.Fields["stall_deadline_at"]); deadline != "" {
		t.progress.StallDeadlineAt = deadline
	}
	switch message {
	case "runtime shard execution prepared":
		if count := progressInt(entry.Fields["shards"]); count > 0 {
			t.plans[strings.TrimSpace(entry.StepID)+":"+strings.TrimSpace(entry.DomainID)] = count
			meaningful = true
		}
	case "runtime task started":
		if unit != "" {
			t.running[unit] = struct{}{}
			meaningful = true
		}
	case "runtime task completed", "shard replayed from checkpoint":
		if unit != "" {
			delete(t.running, unit)
			delete(t.failed, unit)
			t.succeeded[unit] = struct{}{}
			meaningful = true
		}
	case "runtime task failed", "runtime shard failed (best-effort continues)":
		if unit != "" {
			delete(t.running, unit)
			delete(t.succeeded, unit)
			t.failed[unit] = struct{}{}
			meaningful = true
		}
	}
	if strings.Contains(message, "repair") {
		t.progress.Phase = "repairing"
		if attempt := progressInt(entry.Fields["repair_attempt"]); attempt > t.progress.RepairAttempt {
			t.progress.RepairAttempt = attempt
		} else if strings.Contains(message, "repair scheduled") {
			t.progress.RepairAttempt++
		}
		meaningful = true
	} else if strings.Contains(message, "stall") || progressString(entry.Fields["exit_reason"]) == "stall" {
		t.progress.Phase = "stalled"
		meaningful = true
	} else if strings.Contains(message, "validator") || strings.Contains(message, "validat") || strings.Contains(message, "promoting") {
		t.progress.Phase = "validating"
		meaningful = true
	} else if strings.Contains(message, "artifact") || strings.Contains(message, "collected") || strings.Contains(message, "assembled") || strings.Contains(message, "compiled") {
		t.progress.Phase = "artifact_observed"
		t.progress.ArtifactState = "observed"
		meaningful = true
	}
	if meaningful {
		t.progress.LastProgressAt = now.Format(time.RFC3339)
	}
	t.applyCounts()
	return cloneRunProgress(t.progress)
}

func (t *runProgressTracker) snapshot() *RunProgress {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return cloneRunProgress(t.progress)
}

func (e *pipelineExecution) currentRunProgress() *RunProgress {
	if e != nil && e.progressTracker != nil {
		return e.progressTracker.snapshot()
	}
	if e == nil {
		return nil
	}
	return cloneRunProgress(e.stepStatus.Progress)
}

func (t *runProgressTracker) applyCounts() {
	planned := 0
	for _, count := range t.plans {
		planned += count
	}
	t.progress.PlannedUnits = planned
	t.progress.RunningUnits = len(t.running)
	t.progress.SucceededUnits = len(t.succeeded)
	t.progress.FailedUnits = len(t.failed)
	t.progress.CurrentScopes = t.progress.CurrentScopes[:0]
	for scope := range t.scopes {
		t.progress.CurrentScopes = append(t.progress.CurrentScopes, scope)
	}
	sort.Strings(t.progress.CurrentScopes)
}

func progressUnitID(fields map[string]any) string {
	if fields == nil {
		return ""
	}
	if value := progressString(fields["shard_id"]); value != "" {
		return value
	}
	return progressString(fields["task_id"])
}

func progressString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func progressInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func isPersistedRunProgressEvent(entry RunLogEntry) bool {
	if entry.Kind == RunLogKindRuntimeOutput {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(entry.Message))
	if message == "runtime task heartbeat" || message == "runtime task started" || message == "runtime task completed" || message == "runtime task failed" || message == "runtime shard execution prepared" || message == "runtime shard failed (best-effort continues)" || message == "shard replayed from checkpoint" {
		return true
	}
	return strings.Contains(message, "artifact") || strings.Contains(message, "collected") || strings.Contains(message, "assembled") || strings.Contains(message, "compiled") || strings.Contains(message, "repair") || strings.Contains(message, "stall") || strings.Contains(message, "validator") || strings.Contains(message, "validat") || strings.Contains(message, "promoting")
}

func newRunProgress(pipeline Pipeline, startedAt time.Time, currentStep string) *RunProgress {
	now := startedAt.UTC().Format(time.RFC3339)
	progress := runningStepProgress(pipeline, startedAt, currentStep, nil, startedAt)
	progress.Phase = "provider_working"
	progress.StartedAt = now
	return progress
}

func runningStepProgress(pipeline Pipeline, startedAt time.Time, stepID string, scopes []string, now time.Time) *RunProgress {
	steps := stepIDsForPipeline(pipeline)
	index := indexOfPipelineStep(steps, strings.TrimSpace(stepID))
	completed := index
	if completed < 0 {
		completed = 0
	}
	phase := "provider_working"
	if strings.Contains(stepID, "findings") {
		phase = "validating"
	}
	if strings.Contains(stepID, "proposals") {
		phase = "validating"
	}
	return &RunProgress{
		Phase:          phase,
		CompletedSteps: completed,
		TotalSteps:     len(steps),
		CurrentStep:    strings.TrimSpace(stepID),
		ExpectedResult: expectedResultForStep(stepID),
		CurrentScopes:  append([]string(nil), scopes...),
		StartedAt:      startedAt.UTC().Format(time.RFC3339),
		LastActivityAt: now.UTC().Format(time.RFC3339),
		LastProgressAt: now.UTC().Format(time.RFC3339),
		ArtifactState:  "pending",
	}
}

func terminalRunProgress(value *RunProgress, status RunStatus, finishedAt time.Time) *RunProgress {
	if value == nil {
		value = &RunProgress{StartedAt: finishedAt.UTC().Format(time.RFC3339)}
	} else {
		value = cloneRunProgress(value)
	}
	value.Phase = string(status)
	value.LastActivityAt = finishedAt.UTC().Format(time.RFC3339)
	if status == RunStatusSucceeded {
		value.Phase = "completed"
		value.CompletedSteps = value.TotalSteps
		value.CurrentStep = ""
		value.ExpectedResult = "Validated architecture is ready"
		value.ArtifactState = "validated"
	} else if value.ArtifactState == "" {
		value.ArtifactState = "retained_partial"
	}
	return value
}

func expectedResultForStep(stepID string) string {
	switch {
	case strings.Contains(stepID, "constitution"):
		return "Architecture scope and charter"
	case strings.Contains(stepID, "collect"):
		return "Evidence-backed entities and relationships"
	case strings.Contains(stepID, "asis_docs"):
		return "Architecture Home, model and diagrams"
	case strings.Contains(stepID, "findings"):
		return "Validated findings, questions and coverage"
	case strings.Contains(stepID, "proposals"):
		return "Proposals and promoted current architecture"
	default:
		return "Validated architecture knowledge"
	}
}
