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

func TestPendingCommitPublicationReconcilesAfterLinkageCrash(t *testing.T) {
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
	publicationContext := taskPublicationContext{TaskID: task.TaskID, AttemptID: attempt.AttemptID, RunID: attempt.RunID}
	if err := prepareTaskPublication(server.taskRegistry, ws, publicationContext, "commit", "", "publish", state); err != nil {
		t.Fatalf("prepare publication intent: %v", err)
	}
	if _, err := runGit(context.Background(), ws.Path, "add", "-A"); err != nil {
		t.Fatalf("stage publication: %v", err)
	}
	if _, err := runGit(context.Background(), ws.Path, "commit", "-m", "publish"); err != nil {
		t.Fatalf("create publication commit: %v", err)
	}

	restarted := NewServerWithRuntime(ws, nil, ServerRuntimeConfig{Mode: "fake"})
	history := restarted.taskRegistry.Snapshot()
	if history.Tasks[0].Publication.State != producttasks.PublicationLinked || history.Tasks[0].Publication.Action != "commit" || history.Tasks[0].Publication.Commit == "" {
		t.Fatalf("restart did not recover exact commit publication: %+v", history.Tasks[0].Publication)
	}
	if history.Attempts[0].Publication != history.Tasks[0].Publication {
		t.Fatalf("recovered Task/Attempt linkage diverged: task=%+v attempt=%+v", history.Tasks[0].Publication, history.Attempts[0].Publication)
	}
	journalPath, err := publicationJournalPath(context.Background(), ws)
	if err != nil {
		t.Fatalf("resolve publication journal: %v", err)
	}
	intents, err := readPublicationJournal(journalPath)
	if err != nil {
		t.Fatalf("read publication journal: %v", err)
	}
	if len(intents) != 0 {
		t.Fatalf("resolved publication intent remained durable: %+v", intents)
	}
}

func TestPendingBranchPublicationReconcilesAfterLinkageCrash(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for git mutation API tests")
	}
	server := newTestServer(t)
	ws := initGitWorkspaceForDiffTest(t, server)
	commitWorkspaceForDiffTest(t, ws, "baseline")
	task, attempt := publicationFixtures(t)
	if err := server.taskRegistry.Replace(producttasks.History{Version: producttasks.CurrentVersion, GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Tasks: []producttasks.Task{task}, Attempts: []producttasks.Attempt{attempt}, Diagnostics: []string{}}); err != nil {
		t.Fatalf("seed publication fixtures: %v", err)
	}
	state, err := collectWorkspaceGitState(context.Background(), ws)
	if err != nil {
		t.Fatalf("collect confirmation state: %v", err)
	}
	publicationContext := taskPublicationContext{TaskID: task.TaskID, AttemptID: attempt.AttemptID, RunID: attempt.RunID}
	branch := "proposal/recoverable"
	if err := prepareTaskPublication(server.taskRegistry, ws, publicationContext, "branch", branch, "", state); err != nil {
		t.Fatalf("prepare branch intent: %v", err)
	}
	if _, err := runGit(context.Background(), ws.Path, "checkout", "-b", branch); err != nil {
		t.Fatalf("create proposal branch: %v", err)
	}

	restarted := NewServerWithRuntime(ws, nil, ServerRuntimeConfig{Mode: "fake"})
	history := restarted.taskRegistry.Snapshot()
	if history.Tasks[0].Publication.State != producttasks.PublicationLinked || history.Tasks[0].Publication.Action != "branch" || history.Tasks[0].Publication.Branch != branch {
		t.Fatalf("restart did not recover exact branch publication: %+v", history.Tasks[0].Publication)
	}
	if history.Attempts[0].Publication != history.Tasks[0].Publication {
		t.Fatalf("recovered Task/Attempt branch linkage diverged: task=%+v attempt=%+v", history.Tasks[0].Publication, history.Attempts[0].Publication)
	}
}

func TestPendingCommitPublicationDoesNotLinkAmbiguousCommit(t *testing.T) {
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
	publicationContext := taskPublicationContext{TaskID: task.TaskID, AttemptID: attempt.AttemptID, RunID: attempt.RunID}
	if err := prepareTaskPublication(server.taskRegistry, ws, publicationContext, "commit", "", "publish", state); err != nil {
		t.Fatalf("prepare publication intent: %v", err)
	}
	if _, err := runGit(context.Background(), ws.Path, "add", "-A"); err != nil {
		t.Fatalf("stage unrelated commit: %v", err)
	}
	if _, err := runGit(context.Background(), ws.Path, "commit", "-m", "unrelated"); err != nil {
		t.Fatalf("create unrelated commit: %v", err)
	}

	restarted := NewServerWithRuntime(ws, nil, ServerRuntimeConfig{Mode: "fake"})
	history := restarted.taskRegistry.Snapshot()
	if history.Tasks[0].Publication.State != producttasks.PublicationUnavailable || history.Attempts[0].Publication.State != producttasks.PublicationUnavailable {
		t.Fatalf("ambiguous commit was falsely linked: task=%+v attempt=%+v", history.Tasks[0].Publication, history.Attempts[0].Publication)
	}
	journalPath, err := publicationJournalPath(context.Background(), ws)
	if err != nil {
		t.Fatalf("resolve publication journal: %v", err)
	}
	intents, err := readPublicationJournal(journalPath)
	if err != nil {
		t.Fatalf("read publication journal: %v", err)
	}
	if len(intents) != 1 {
		t.Fatal("ambiguous publication intent was not retained for recovery")
	}
}

func TestGitCommitNoChangesClearsPublicationIntent(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for git mutation API tests")
	}
	server := newTestServer(t)
	ws := initGitWorkspaceForDiffTest(t, server)
	commitWorkspaceForDiffTest(t, ws, "baseline")
	task, attempt := publicationFixtures(t)
	if err := server.taskRegistry.Replace(producttasks.History{Version: producttasks.CurrentVersion, GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Tasks: []producttasks.Task{task}, Attempts: []producttasks.Attempt{attempt}, Diagnostics: []string{}}); err != nil {
		t.Fatalf("seed publication fixtures: %v", err)
	}
	commitWorkspaceForDiffTest(t, ws, "history baseline")
	state, err := collectWorkspaceGitState(context.Background(), ws)
	if err != nil {
		t.Fatalf("collect confirmation state: %v", err)
	}
	body := `{"message":"publish","expected_fingerprint":"` + state.Fingerprint + `","expected_head_oid":"` + state.Identity.HeadOID + `","task_id":"` + task.TaskID + `","attempt_id":"` + attempt.AttemptID + `","run_id":"` + attempt.RunID + `"}`
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	response := postJSON(t, httpServer.URL+"/api/git/commit", body)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected no-op commit 200, got %d", response.StatusCode)
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode no-op response: %v", err)
	}
	if payload.Status != "no_changes" {
		t.Fatalf("expected no_changes status, got %q", payload.Status)
	}
	journalPath, err := publicationJournalPath(context.Background(), ws)
	if err != nil {
		t.Fatalf("resolve publication journal: %v", err)
	}
	intents, err := readPublicationJournal(journalPath)
	if err != nil {
		t.Fatalf("read publication journal: %v", err)
	}
	if len(intents) != 0 {
		t.Fatalf("no-op commit left publication intent: %+v", intents)
	}
	if status, err := runGit(context.Background(), ws.Path, "status", "--porcelain"); err != nil || strings.TrimSpace(status) != "" {
		t.Fatalf("no-op commit left workspace drift: status=%q err=%v", status, err)
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
