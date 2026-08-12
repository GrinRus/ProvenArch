package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	producttasks "github.com/GrinRus/ProvenArch/internal/tasks"
)

func TestGitCommitRecordsExactTaskAttemptPublicationLinkage(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for git mutation API tests")
	}
	server := newTestServer(t)
	ws := initGitWorkspaceForDiffTest(t, server)
	commitWorkspaceForDiffTest(t, ws, "baseline")
	if err := ws.WriteFile("reports/as-is/overview.md", []byte("candidate\n")); err != nil {
		t.Fatal(err)
	}
	task, attempt := publicationFixtures(t)
	if err := server.taskRegistry.Replace(producttasks.History{Version: producttasks.CurrentVersion, GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Tasks: []producttasks.Task{task}, Attempts: []producttasks.Attempt{attempt}, Diagnostics: []string{}}); err != nil {
		t.Fatalf("seed publication fixtures: %v", err)
	}
	state, err := collectWorkspaceGitState(context.Background(), ws)
	if err != nil {
		t.Fatalf("collect confirmation state: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	body := `{"message":"publish","expected_fingerprint":"` + state.Fingerprint + `","expected_head_oid":"` + state.Identity.HeadOID + `","task_id":"` + task.TaskID + `","attempt_id":"` + attempt.AttemptID + `","run_id":"` + attempt.RunID + `"}`
	response := postJSON(t, httpServer.URL+"/api/git/commit", body)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected commit 200, got %d", response.StatusCode)
	}
	var payload struct {
		Publication producttasks.Publication `json:"publication"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode publication response: %v", err)
	}
	if payload.Publication.State != producttasks.PublicationLinked || payload.Publication.AttemptID != attempt.AttemptID || payload.Publication.RunID != attempt.RunID || payload.Publication.Action != "commit" {
		t.Fatalf("unexpected publication response: %+v", payload.Publication)
	}
	if payload.Publication.InventoryFingerprint != state.Fingerprint || payload.Publication.BaseOID != state.Identity.BaseOID || payload.Publication.HeadOID == state.Identity.HeadOID || payload.Publication.Commit != payload.Publication.HeadOID {
		t.Fatalf("publication lost exact Git identity: before=%+v publication=%+v", state, payload.Publication)
	}
	history := server.taskRegistry.Snapshot()
	if history.Tasks[0].Publication != payload.Publication || history.Attempts[0].Publication != payload.Publication {
		t.Fatalf("Task and Attempt publication records diverged: task=%+v attempt=%+v response=%+v", history.Tasks[0].Publication, history.Attempts[0].Publication, payload.Publication)
	}
}

func TestGitCommitRejectsPartialPublicationContextWithoutMutation(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for git mutation API tests")
	}
	server := newTestServer(t)
	ws := initGitWorkspaceForDiffTest(t, server)
	commitWorkspaceForDiffTest(t, ws, "baseline")
	if err := ws.WriteFile("reports/as-is/overview.md", []byte("candidate\n")); err != nil {
		t.Fatal(err)
	}
	state, err := collectWorkspaceGitState(context.Background(), ws)
	if err != nil {
		t.Fatalf("collect confirmation state: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	body := `{"message":"publish","expected_fingerprint":"` + state.Fingerprint + `","expected_head_oid":"` + state.Identity.HeadOID + `","task_id":"task_20260811_0001"}`
	response := postJSON(t, httpServer.URL+"/api/git/commit", body)
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest || decodeErrorCode(t, response) != "publication_context_invalid" {
		t.Fatalf("expected publication_context_invalid 400, got %d", response.StatusCode)
	}
	status, err := runGit(context.Background(), ws.Path, "status", "--porcelain")
	if err != nil || !strings.Contains(status, "reports/") {
		t.Fatalf("partial context mutated workspace: status=%q err=%v", status, err)
	}
}

func publicationFixtures(t *testing.T) (producttasks.Task, producttasks.Attempt) {
	t.Helper()
	taskRaw, err := os.ReadFile(filepath.Join("..", "..", "examples", "task.example.json"))
	if err != nil {
		t.Fatalf("read task fixture: %v", err)
	}
	task, err := producttasks.ParseTask(taskRaw)
	if err != nil {
		t.Fatalf("parse task fixture: %v", err)
	}
	attempt := readAttemptFixture(t)
	return task, attempt
}
