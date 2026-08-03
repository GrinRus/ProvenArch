package orchestrator

import (
	"testing"
	"time"
)

func TestRunProgressUsesOnlyDeterministicPipelineSteps(t *testing.T) {
	started := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	progress := runningStepProgress(PipelineInit, started, "init.step2.asis_docs", []string{"payments"}, started.Add(time.Minute))
	if progress.CompletedSteps != 2 || progress.TotalSteps != 5 || progress.Phase != "provider_working" {
		t.Fatalf("unexpected progress: %#v", progress)
	}
	if progress.ExpectedResult != "Architecture Home, model and diagrams" || len(progress.CurrentScopes) != 1 {
		t.Fatalf("unexpected progress detail: %#v", progress)
	}
	terminal := terminalRunProgress(progress, RunStatusSucceeded, started.Add(2*time.Minute))
	if terminal.CompletedSteps != terminal.TotalSteps || terminal.ArtifactState != "validated" || terminal.Phase != "completed" || terminal.CurrentStep != "" {
		t.Fatalf("unexpected terminal progress: %#v", terminal)
	}
	if terminal.ElapsedMS != int64((2*time.Minute)/time.Millisecond) {
		t.Fatalf("terminal elapsed time = %d", terminal.ElapsedMS)
	}
}

func TestRunProgressSeparatesProviderActivityFromUsefulProgress(t *testing.T) {
	started := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	tracker := newRunProgressTracker(PipelineInit, started, "init.step1.collect")
	before := tracker.snapshot()
	activity := tracker.observe(RunLogEntry{Timestamp: started.Add(time.Minute), Kind: RunLogKindRuntimeOutput, StepID: "init.step1.collect", Message: "provider stdout"})
	if activity.LastActivityAt == before.LastActivityAt {
		t.Fatalf("provider output did not update activity: %#v", activity)
	}
	if activity.LastProgressAt != before.LastProgressAt {
		t.Fatalf("stdout must not count as useful progress: before=%s after=%s", before.LastProgressAt, activity.LastProgressAt)
	}
	prepared := tracker.observe(RunLogEntry{Timestamp: started.Add(2 * time.Minute), StepID: "init.step1.collect", DomainID: "payments", Message: "runtime shard execution prepared", Fields: map[string]any{"shards": 2}})
	if prepared.PlannedUnits != 2 || prepared.LastProgressAt == activity.LastProgressAt {
		t.Fatalf("structured shard plan was not counted as useful progress: %#v", prepared)
	}
	tracker.observe(RunLogEntry{Timestamp: started.Add(3 * time.Minute), StepID: "init.step1.collect", Message: "runtime task started", Fields: map[string]any{"shard_id": "payments-a"}})
	completed := tracker.observe(RunLogEntry{Timestamp: started.Add(4 * time.Minute), StepID: "init.step1.collect", Message: "runtime task completed", Fields: map[string]any{"shard_id": "payments-a"}})
	if completed.RunningUnits != 0 || completed.SucceededUnits != 1 || completed.PlannedUnits != 2 {
		t.Fatalf("unexpected unit counters: %#v", completed)
	}
}

func TestRunProgressTracksRepairAndStallPhases(t *testing.T) {
	started := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	tracker := newRunProgressTracker(PipelineRefresh, started, "refresh.step1.collect")
	repair := tracker.observe(RunLogEntry{Timestamp: started.Add(time.Minute), StepID: "refresh.step1.collect", Message: "focused artifact repair scheduled", Fields: map[string]any{"repair_limit": 2, "stall_deadline_at": "2026-08-03T10:05:00Z"}})
	if repair.Phase != "repairing" || repair.RepairAttempt != 1 || repair.RepairLimit != 2 || repair.StallDeadlineAt == "" {
		t.Fatalf("unexpected repair progress: %#v", repair)
	}
	stalled := tracker.observe(RunLogEntry{Timestamp: started.Add(2 * time.Minute), StepID: "refresh.step1.collect", Message: "runtime stalled before artifacts"})
	if stalled.Phase != "stalled" || stalled.RepairAttempt != 1 {
		t.Fatalf("unexpected stalled progress: %#v", stalled)
	}
}
