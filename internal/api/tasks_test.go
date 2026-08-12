package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	producttasks "github.com/GrinRus/ProvenArch/internal/tasks"
)

func TestTaskAPICreateListPatchArchiveAndCursor(t *testing.T) {
	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	create := `{"title":"Task one","goal":"Inspect the payments architecture","scope":{"repositories":[{"name":"payments-service","paths":["."]}]},"desired_runner":{"preset":"default","mode":"fake","provider":"claude-code"}}`
	response := postJSON(t, httpServer.URL+"/api/tasks", create)
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected task create 201, got %d", response.StatusCode)
	}
	var created struct {
		Task producttasks.Task `json:"task"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode created task: %v", err)
	}
	if !producttasks.IsOpaqueID(created.Task.TaskID) || created.Task.Revision != 1 {
		t.Fatalf("unexpected created task: %+v", created.Task)
	}

	listResponse, err := http.Get(httpServer.URL + "/api/tasks?limit=1")
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	var listed struct {
		Items       []producttasks.Task `json:"items"`
		NextCursor  string              `json:"next_cursor"`
		HasMore     bool                `json:"has_more"`
		Diagnostics []string            `json:"diagnostics"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&listed); err != nil {
		listResponse.Body.Close()
		t.Fatalf("decode task list: %v", err)
	}
	listResponse.Body.Close()
	if listResponse.StatusCode != http.StatusOK || len(listed.Items) != 1 || listed.Items[0].TaskID != created.Task.TaskID || listed.HasMore || listed.Diagnostics == nil {
		t.Fatalf("unexpected task list: status=%d payload=%+v", listResponse.StatusCode, listed)
	}

	patchRequest, err := http.NewRequest(http.MethodPatch, httpServer.URL+"/api/tasks/"+created.Task.TaskID, strings.NewReader(`{"expected_revision":1,"title":"Updated task","context":"new context"}`))
	if err != nil {
		t.Fatalf("create task patch request: %v", err)
	}
	patchRequest.Header.Set("Content-Type", "application/json")
	patchResponse, err := http.DefaultClient.Do(patchRequest)
	if err != nil {
		t.Fatalf("patch task: %v", err)
	}
	var patched struct {
		Task producttasks.Task `json:"task"`
	}
	if err := json.NewDecoder(patchResponse.Body).Decode(&patched); err != nil {
		patchResponse.Body.Close()
		t.Fatalf("decode patched task: %v", err)
	}
	patchResponse.Body.Close()
	if patchResponse.StatusCode != http.StatusOK || patched.Task.Revision != 2 || patched.Task.Title != "Updated task" || patched.Task.Context != "new context" {
		t.Fatalf("unexpected patched task: status=%d payload=%+v", patchResponse.StatusCode, patched)
	}

	conflictRequest, err := http.NewRequest(http.MethodPatch, httpServer.URL+"/api/tasks/"+created.Task.TaskID, strings.NewReader(`{"expected_revision":1,"title":"stale"}`))
	if err != nil {
		t.Fatalf("create conflict request: %v", err)
	}
	conflictRequest.Header.Set("Content-Type", "application/json")
	conflictResponse, err := http.DefaultClient.Do(conflictRequest)
	if err != nil {
		t.Fatalf("patch stale task: %v", err)
	}
	conflictResponse.Body.Close()
	if conflictResponse.StatusCode != http.StatusConflict {
		t.Fatalf("expected stale patch 409, got %d", conflictResponse.StatusCode)
	}

	archiveResponse := postJSON(t, httpServer.URL+"/api/tasks/"+created.Task.TaskID+"/archive", `{"expected_revision":2}`)
	archiveResponse.Body.Close()
	if archiveResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected archive 200, got %d", archiveResponse.StatusCode)
	}
	unarchiveResponse := postJSON(t, httpServer.URL+"/api/tasks/"+created.Task.TaskID+"/unarchive", `{"expected_revision":3}`)
	unarchiveResponse.Body.Close()
	if unarchiveResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected unarchive 200, got %d", unarchiveResponse.StatusCode)
	}
}

func TestTaskAPIAttemptReadsRequireExactTaskIdentity(t *testing.T) {
	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response := postJSON(t, httpServer.URL+"/api/tasks", `{"title":"Task","goal":"Goal","scope":{"repositories":[{"name":"payments-service","paths":["."]}]},"desired_runner":{"preset":"default","mode":"fake","provider":"claude-code"}}`)
	var created struct {
		Task producttasks.Task `json:"task"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		response.Body.Close()
		t.Fatalf("decode created task: %v", err)
	}
	response.Body.Close()
	attempt := readAttemptFixture(t)
	attempt.TaskID = created.Task.TaskID
	if err := server.taskRegistry.Update(func(history *producttasks.History) error {
		history.Attempts = append(history.Attempts, attempt)
		history.Tasks[0].Attempts = []producttasks.AttemptSummary{{AttemptID: attempt.AttemptID, RunID: attempt.RunID, TaskRevision: attempt.TaskRevision, Status: attempt.Status, AdmittedAt: attempt.AdmittedAt, UpdatedAt: *attempt.FinishedAt, FinishedAt: attempt.FinishedAt, RetainedEvidence: attempt.RetainedEvidence}}
		history.Tasks[0].Outcome.AttemptID = attempt.AttemptID
		history.Tasks[0].Outcome.RunID = attempt.RunID
		history.Tasks[0].Outcome.SnapshotPath = attempt.Outcome.SnapshotPath
		history.Tasks[0].Outcome.State = producttasks.Available
		history.Tasks[0].Outcome.UnavailableReason = ""
		return nil
	}); err != nil {
		t.Fatalf("persist attempt fixture: %v", err)
	}

	attemptResponse, err := http.Get(httpServer.URL + "/api/tasks/" + created.Task.TaskID + "/attempts/" + attempt.AttemptID)
	if err != nil {
		t.Fatalf("read attempt: %v", err)
	}
	var payload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(attemptResponse.Body).Decode(&payload); err != nil {
		attemptResponse.Body.Close()
		t.Fatalf("decode attempt: %v", err)
	}
	attemptResponse.Body.Close()
	if attemptResponse.StatusCode != http.StatusOK || payload.Attempt.AttemptID != attempt.AttemptID || payload.Attempt.TaskID != created.Task.TaskID {
		t.Fatalf("unexpected attempt response: status=%d payload=%+v", attemptResponse.StatusCode, payload)
	}

	missingResponse, err := http.Get(httpServer.URL + "/api/tasks/other-task/attempts/" + attempt.AttemptID)
	if err != nil {
		t.Fatalf("read attempt through foreign task: %v", err)
	}
	missingResponse.Body.Close()
	if missingResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("expected foreign task attempt read 404, got %d", missingResponse.StatusCode)
	}
}

func readAttemptFixture(t *testing.T) producttasks.Attempt {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "attempt.example.json"))
	if err != nil {
		t.Fatalf("read attempt fixture: %v", err)
	}
	attempt, err := producttasks.ParseAttempt(raw)
	if err != nil {
		t.Fatalf("parse attempt fixture: %v", err)
	}
	return attempt
}
