package tasks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTaskAndAttemptExamplesRoundTrip(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		filepath.Join("..", "..", "examples", "task.example.json"),
		filepath.Join("..", "..", "examples", "attempt.example.json"),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.HasSuffix(path, "task.example.json") {
			value, parseErr := ParseTask(raw)
			if parseErr != nil {
				t.Fatalf("parse task: %v", parseErr)
			}
			encoded, marshalErr := MarshalTask(value)
			if marshalErr != nil {
				t.Fatalf("marshal task: %v", marshalErr)
			}
			if _, parseErr = ParseTask(encoded); parseErr != nil {
				t.Fatalf("parse task round trip: %v", parseErr)
			}
			continue
		}
		value, parseErr := ParseAttempt(raw)
		if parseErr != nil {
			t.Fatalf("parse attempt: %v", parseErr)
		}
		encoded, marshalErr := MarshalAttempt(value)
		if marshalErr != nil {
			t.Fatalf("marshal attempt: %v", marshalErr)
		}
		if _, parseErr = ParseAttempt(encoded); parseErr != nil {
			t.Fatalf("parse attempt round trip: %v", parseErr)
		}
	}
}

func TestParseHistoryChecksTaskAttemptJoin(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "tasks", "task-history.json"))
	if err != nil {
		t.Fatalf("read history fixture: %v", err)
	}
	history, err := ParseHistory(raw)
	if err != nil {
		t.Fatalf("parse history: %v", err)
	}
	if len(history.Tasks) != 1 || len(history.Attempts) != 1 {
		t.Fatalf("unexpected history cardinality: tasks=%d attempts=%d", len(history.Tasks), len(history.Attempts))
	}
	encoded, err := MarshalHistory(history)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}
	if _, err := ParseHistory(encoded); err != nil {
		t.Fatalf("parse history round trip: %v", err)
	}

	history.Attempts[0].RunID = "run-foreign"
	if err := history.Validate(); err == nil || !strings.Contains(err.Error(), "equivalent task attempt summary") {
		t.Fatalf("expected dangling task/attempt join error, got %v", err)
	}
}

func TestParseRejectsUnknownFieldsAndInvalidIDs(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "task.example.json"))
	if err != nil {
		t.Fatalf("read task fixture: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode task fixture: %v", err)
	}
	payload["future_field"] = true
	unknown, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode unknown task: %v", err)
	}
	if _, err := ParseTask(unknown); err == nil || !strings.Contains(err.Error(), "additionalProperties") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}
	delete(payload, "future_field")
	payload["task_id"] = "legacy"
	invalidID, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode invalid task: %v", err)
	}
	if _, err := ParseTask(invalidID); err == nil || !strings.Contains(err.Error(), "task_id") {
		t.Fatalf("expected invalid id rejection, got %v", err)
	}
	payload["task_id"] = "task_20260811_0001"
	payload["scope"].(map[string]any)["repositories"].([]any)[0].(map[string]any)["paths"] = []any{"../escape"}
	invalidScope, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode invalid scope task: %v", err)
	}
	if _, err := ParseTask(invalidScope); err == nil || !strings.Contains(err.Error(), "workspace-relative") {
		t.Fatalf("expected invalid scope rejection, got %v", err)
	}
	payload["scope"].(map[string]any)["repositories"].([]any)[0].(map[string]any)["paths"] = []any{"."}
	payload["desired_runner"].(map[string]any)["provider"] = "unknown-provider"
	invalidRunner, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode invalid runner task: %v", err)
	}
	if _, err := ParseTask(invalidRunner); err == nil || !strings.Contains(err.Error(), "provider") {
		t.Fatalf("expected invalid runner rejection, got %v", err)
	}
}

func TestAttemptSnapshotCloneIsIndependent(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "attempt.example.json"))
	if err != nil {
		t.Fatalf("read attempt fixture: %v", err)
	}
	original, err := ParseAttempt(raw)
	if err != nil {
		t.Fatalf("parse attempt: %v", err)
	}
	clone := CloneAttempt(original)
	clone.EffectiveRuntime.Timeouts["pipeline_timeout_sec"] = 1
	clone.EffectiveRuntime.ResolutionSources["model"] = "env"
	clone.EffectiveRuntime.Scope.Repositories[0].Paths[0] = "reports"
	clone.IntentSnapshot.Scope.Repositories[0].Paths[0] = "docs"
	if original.EffectiveRuntime.Timeouts["pipeline_timeout_sec"] != 1800 ||
		original.EffectiveRuntime.ResolutionSources["model"] != "workspace" ||
		original.EffectiveRuntime.Scope.Repositories[0].Paths[0] != "." ||
		original.IntentSnapshot.Scope.Repositories[0].Paths[0] != "." {
		t.Fatal("mutating the attempt clone changed the immutable source snapshot")
	}
}

func TestAttemptRejectsSelfParentAndTimestampRegression(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "attempt.example.json"))
	if err != nil {
		t.Fatalf("read attempt fixture: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode attempt fixture: %v", err)
	}
	payload["parent_attempt_id"] = payload["attempt_id"]
	payload["started_at"] = "2026-08-11T09:00:00Z"
	invalid, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode invalid attempt: %v", err)
	}
	if _, err := ParseAttempt(invalid); err == nil || !strings.Contains(err.Error(), "parent_attempt_id") || !strings.Contains(err.Error(), "started_at") {
		t.Fatalf("expected lineage and timestamp rejection, got %v", err)
	}
}
