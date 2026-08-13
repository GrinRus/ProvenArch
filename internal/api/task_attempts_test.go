package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	producttasks "github.com/GrinRus/ProvenArch/internal/tasks"
)

func TestTaskAttemptAdmissionIsIdempotentAndLinksExactRun(t *testing.T) {
	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	created := createTaskForAttemptTest(t, httpServer.URL, "first")

	first := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"attempt-key","pipeline":"init"}`)
	var firstPayload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(first.Body).Decode(&firstPayload); err != nil {
		first.Body.Close()
		t.Fatalf("decode first admission: %v", err)
	}
	first.Body.Close()
	if first.StatusCode != http.StatusAccepted || firstPayload.Attempt.TaskID != created.TaskID || firstPayload.Attempt.AttemptID == "" || firstPayload.Attempt.RunID == "" {
		t.Fatalf("unexpected first admission: status=%d payload=%+v", first.StatusCode, firstPayload)
	}
	if firstPayload.Attempt.IdempotencyKey != "attempt-key" || firstPayload.Attempt.RequestFingerprint == "" || firstPayload.Attempt.EffectiveRuntime.Provider == "" {
		t.Fatalf("admission snapshot is incomplete: %+v", firstPayload.Attempt)
	}

	second := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"attempt-key","pipeline":"init"}`)
	var secondPayload struct {
		Attempt    producttasks.Attempt `json:"attempt"`
		Idempotent bool                 `json:"idempotent"`
	}
	if err := json.NewDecoder(second.Body).Decode(&secondPayload); err != nil {
		second.Body.Close()
		t.Fatalf("decode duplicate admission: %v", err)
	}
	second.Body.Close()
	if second.StatusCode != http.StatusOK || !secondPayload.Idempotent || secondPayload.Attempt.AttemptID != firstPayload.Attempt.AttemptID || secondPayload.Attempt.RunID != firstPayload.Attempt.RunID {
		t.Fatalf("duplicate admission was not idempotent: status=%d payload=%+v", second.StatusCode, secondPayload)
	}

	conflict := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"attempt-key","pipeline":"refresh"}`)
	conflict.Body.Close()
	if conflict.StatusCode != http.StatusConflict {
		t.Fatalf("expected idempotency conflict, got %d", conflict.StatusCode)
	}
	run, ok := server.getService().GetRun(firstPayload.Attempt.RunID)
	if !ok || run.TaskID != created.TaskID || run.AttemptID != firstPayload.Attempt.AttemptID {
		t.Fatalf("run linkage missing: ok=%v run=%+v", ok, run)
	}
}

func TestTaskAttemptRetryCreatesChildAttempt(t *testing.T) {
	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	created := createTaskForAttemptTest(t, httpServer.URL, "retry")
	first := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"parent-key"}`)
	var firstPayload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(first.Body).Decode(&firstPayload); err != nil {
		first.Body.Close()
		t.Fatalf("decode parent: %v", err)
	}
	first.Body.Close()
	waitForTerminalAttempt(t, server, firstPayload.Attempt.AttemptID)

	retry := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts/"+firstPayload.Attempt.AttemptID+"/retry", `{"idempotency_key":"child-key","reason":"repair"}`)
	var retryPayload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(retry.Body).Decode(&retryPayload); err != nil {
		retry.Body.Close()
		t.Fatalf("decode retry: %v", err)
	}
	retry.Body.Close()
	if retry.StatusCode != http.StatusAccepted || retryPayload.Attempt.AttemptID == firstPayload.Attempt.AttemptID || retryPayload.Attempt.ParentAttemptID == nil || *retryPayload.Attempt.ParentAttemptID != firstPayload.Attempt.AttemptID || retryPayload.Attempt.RetryReason != "repair" {
		t.Fatalf("retry did not create child attempt: status=%d payload=%+v", retry.StatusCode, retryPayload)
	}
}

func TestTaskAttemptRetryRejectsArchivedTask(t *testing.T) {
	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	created := createTaskForAttemptTest(t, httpServer.URL, "archived-retry")
	first := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"parent-key"}`)
	var firstPayload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(first.Body).Decode(&firstPayload); err != nil {
		first.Body.Close()
		t.Fatalf("decode parent: %v", err)
	}
	first.Body.Close()
	waitForTerminalAttempt(t, server, firstPayload.Attempt.AttemptID)

	archive := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/archive", `{"expected_revision":1}`)
	archive.Body.Close()
	if archive.StatusCode != http.StatusOK {
		t.Fatalf("archive task failed: %d", archive.StatusCode)
	}
	retry := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts/"+firstPayload.Attempt.AttemptID+"/retry", `{"idempotency_key":"archived-child"}`)
	retry.Body.Close()
	if retry.StatusCode != http.StatusConflict {
		t.Fatalf("expected archived retry conflict, got %d", retry.StatusCode)
	}
}

func TestMissingRuntimeRunIsReconciledAsUnavailable(t *testing.T) {
	t.Parallel()
	attempt := readAttemptFixture(t)
	attempt.Status = producttasks.AttemptRunning
	attempt.FinishedAt = nil
	attempt.TerminalSummary = nil
	attempt.Outcome = producttasks.Outcome{State: producttasks.Unavailable, UnavailableReason: "attempt has not completed"}
	history := producttasks.History{
		Tasks:    []producttasks.Task{{TaskID: attempt.TaskID, Revision: attempt.TaskRevision, Outcome: attempt.Outcome, Attempts: []producttasks.AttemptSummary{{AttemptID: attempt.AttemptID, RunID: attempt.RunID, TaskRevision: attempt.TaskRevision, Status: attempt.Status, AdmittedAt: attempt.AdmittedAt, UpdatedAt: attempt.AdmittedAt, RetainedEvidence: attempt.RetainedEvidence}}}},
		Attempts: []producttasks.Attempt{attempt},
	}
	if err := markAttemptUnavailableAfterRestart(&history, attempt.TaskID, attempt.AttemptID); err != nil {
		t.Fatalf("reconcile missing run: %v", err)
	}
	if history.Attempts[0].Status != producttasks.AttemptFailed || history.Attempts[0].TerminalSummary == nil || history.Attempts[0].TerminalSummary.ErrorCode != "run_reconciled_after_restart" {
		t.Fatalf("missing run was not terminalized: %+v", history.Attempts[0])
	}
	if history.Tasks[0].Outcome.State != producttasks.Unavailable || history.Tasks[0].Attempts[0].Status != producttasks.AttemptFailed {
		t.Fatalf("task projection was not updated: %+v", history.Tasks[0])
	}
}

func TestTaskAttemptRejectsForeignScopeBeforeAdmission(t *testing.T) {
	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	response := postJSON(t, httpServer.URL+"/api/tasks", `{"title":"foreign","goal":"Goal","scope":{"repositories":[{"name":"not-in-manifest","paths":["."]}]},"desired_runner":{"preset":"default","mode":"fake","provider":"claude-code"}}`)
	var payload struct {
		Task producttasks.Task `json:"task"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		response.Body.Close()
		t.Fatalf("decode foreign task: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected task creation to preserve durable intent, got %d", response.StatusCode)
	}
	attempt := postJSON(t, httpServer.URL+"/api/tasks/"+payload.Task.TaskID+"/attempts", `{"idempotency_key":"foreign-key"}`)
	attempt.Body.Close()
	if attempt.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected foreign scope to be rejected before runtime admission, got %d", attempt.StatusCode)
	}
}

func createTaskForAttemptTest(t *testing.T, baseURL, title string) producttasks.Task {
	t.Helper()
	response := postJSON(t, baseURL+"/api/tasks", `{"title":"`+title+`","goal":"Goal","scope":{"repositories":[{"name":"payments-service","paths":["."]}]},"desired_runner":{"preset":"default","mode":"fake","provider":"claude-code"}}`)
	defer response.Body.Close()
	var payload struct {
		Task producttasks.Task `json:"task"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create task failed: %d", response.StatusCode)
	}
	return payload.Task
}

func waitForTerminalAttempt(t *testing.T, server *Server, attemptID string) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		history := server.taskRegistry.Snapshot()
		for _, attempt := range history.Attempts {
			if attempt.AttemptID == attemptID && (attempt.Status == producttasks.AttemptSucceeded || attempt.Status == producttasks.AttemptFailed || attempt.Status == producttasks.AttemptCanceled) {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("attempt %s did not reach terminal state; coordination=%+v", attemptID, server.getService().Coordination())
}
