package orchestrator

import (
	"testing"
	"time"
)

func TestRunRetentionPreservesInFlightRecordsAndTaskAttemptEvidence(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	runs := map[string]*runRecord{
		"run-old-terminal": {
			info: RunInfo{
				RunID:     "run-old-terminal",
				Pipeline:  string(PipelineRefresh),
				Status:    RunStatusSucceeded,
				StartedAt: started,
			},
		},
		"run-new-terminal": {
			info: RunInfo{
				RunID:     "run-new-terminal",
				Pipeline:  string(PipelineRefresh),
				Status:    RunStatusFailed,
				StartedAt: started.Add(time.Hour),
			},
		},
		"run-old-queued": {
			info: RunInfo{
				RunID:     "run-old-queued",
				TaskID:    "task-queued",
				AttemptID: "attempt-queued",
				Pipeline:  string(PipelineRefresh),
				Status:    RunStatusQueued,
				StartedAt: started.Add(-2 * time.Hour),
			},
		},
		"run-old-running": {
			info: RunInfo{
				RunID:     "run-old-running",
				TaskID:    "task-running",
				AttemptID: "attempt-running",
				Pipeline:  string(PipelineRefresh),
				Status:    RunStatusRunning,
				StartedAt: started.Add(-3 * time.Hour),
			},
		},
	}

	trimRunRegistry(runs, 1)

	if len(runs) != 3 {
		t.Fatalf("expected two in-flight and one terminal record, got %d: %#v", len(runs), runs)
	}
	for _, runID := range []string{"run-old-queued", "run-old-running", "run-new-terminal"} {
		if _, ok := runs[runID]; !ok {
			t.Fatalf("retention removed required run %q", runID)
		}
	}
	if _, ok := runs["run-old-terminal"]; ok {
		t.Fatal("retention kept the oldest terminal run over the configured terminal budget")
	}

	for _, runID := range []string{"run-old-queued", "run-old-running"} {
		record := runs[runID]
		if record.info.TaskID == "" || record.info.AttemptID == "" {
			t.Fatalf("retained in-flight run %q lost Task/Attempt linkage: %+v", runID, record.info)
		}
	}
}

func TestRunRetentionKeepsAllInFlightRecordsWhenNoTerminalBudgetIsAvailable(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	runs := map[string]*runRecord{
		"run-queued": {
			info: RunInfo{RunID: "run-queued", Status: RunStatusQueued, StartedAt: started},
		},
		"run-running": {
			info: RunInfo{RunID: "run-running", Status: RunStatusRunning, StartedAt: started.Add(time.Hour)},
		},
	}

	trimRunRegistry(runs, 1)

	if len(runs) != 2 {
		t.Fatalf("retention must not evict in-flight records, got %d: %#v", len(runs), runs)
	}
}

func TestRunRetentionPersistsInFlightTaskAttemptLinkage(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(WithHistoryWorkspace(ws))
	service.historyRetention = 1
	started := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	for _, record := range []runRecord{
		{info: RunInfo{RunID: "run-terminal-old", Pipeline: string(PipelineRefresh), Status: RunStatusSucceeded, StartedAt: started}},
		{info: RunInfo{RunID: "run-terminal-new", Pipeline: string(PipelineRefresh), Status: RunStatusFailed, StartedAt: started.Add(time.Hour)}},
		{info: RunInfo{RunID: "run-queued", TaskID: "task-queued", AttemptID: "attempt-queued", Pipeline: string(PipelineRefresh), Status: RunStatusQueued, StartedAt: started.Add(-2 * time.Hour)}},
		{info: RunInfo{RunID: "run-running", TaskID: "task-running", AttemptID: "attempt-running", Pipeline: string(PipelineRefresh), Status: RunStatusRunning, StartedAt: started.Add(-3 * time.Hour)}},
	} {
		if err := service.storeRun(record); err != nil {
			t.Fatalf("store %s: %v", record.info.RunID, err)
		}
	}

	snapshot := readRunHistorySnapshot(t, ws.Path)
	if len(snapshot.Items) != 3 {
		t.Fatalf("expected two in-flight and one terminal history item, got %d: %#v", len(snapshot.Items), snapshot.Items)
	}
	for _, runID := range []string{"run-queued", "run-running", "run-terminal-new"} {
		_, ok := findHistoryItem(snapshot, runID)
		if !ok {
			t.Fatalf("persisted history removed required run %q: %#v", runID, snapshot.Items)
		}
	}
	for runID, want := range map[string][2]string{
		"run-queued":  {"task-queued", "attempt-queued"},
		"run-running": {"task-running", "attempt-running"},
	} {
		item, _ := findHistoryItem(snapshot, runID)
		if item.TaskID != want[0] || item.AttemptID != want[1] {
			t.Fatalf("persisted history lost Task/Attempt linkage for %q: got task=%q attempt=%q", runID, item.TaskID, item.AttemptID)
		}
	}
	if _, ok := findHistoryItem(snapshot, "run-terminal-old"); ok {
		t.Fatal("persisted history kept the oldest terminal run over the configured terminal budget")
	}
}
