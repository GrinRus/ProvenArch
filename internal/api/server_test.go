package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/testutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
}

func TestEmbeddedUIIsServedFromRoot(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if !strings.Contains(strings.ToLower(string(content)), "<!doctype html") {
		t.Fatalf("expected html shell, got %q", string(content))
	}
}

func TestWorkspaceValidateEndpoint(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(httpServer.URL+"/api/workspace/validate", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/workspace/validate: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
}

func TestWorkspaceValidateEndpointReturnsErrorEnvelopeWithDiagnostics(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + filepath.Join(root, "repos", "missing-repo") + `
`
	server := newTestServerFromManifest(t, manifest)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(httpServer.URL+"/api/workspace/validate", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/workspace/validate: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.StatusCode)
	}

	var payload struct {
		OK    bool `json:"ok"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Errors []map[string]any `json:"errors"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode validation payload: %v", err)
	}
	if payload.OK {
		t.Fatalf("expected ok=false, got true")
	}
	if payload.Error.Code != "workspace_validation_failed" {
		t.Fatalf("expected workspace_validation_failed code, got %q", payload.Error.Code)
	}
	if payload.Error.Message == "" {
		t.Fatalf("expected non-empty error message")
	}
	if len(payload.Errors) == 0 {
		t.Fatalf("expected non-empty validation errors diagnostics")
	}
}

func TestWorkspaceValidateIncludesLayoutReadinessWarnings(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(httpServer.URL+"/api/workspace/validate", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/workspace/validate: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var payload struct {
		OK       bool `json:"ok"`
		Warnings []struct {
			Code string `json:"code"`
		} `json:"warnings"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode validation payload: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected ok=true for partial layout with warnings")
	}
	hasLayoutWarning := false
	for _, warning := range payload.Warnings {
		if warning.Code == "workspace.layout.dir.missing" {
			hasLayoutWarning = true
			break
		}
	}
	if !hasLayoutWarning {
		t.Fatalf("expected workspace.layout.dir.missing warning, got %+v", payload.Warnings)
	}
}

func TestWorkspaceValidateReportsInvalidRepoRef(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for repo ref validation test")
	}

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "payments-service")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	runGitTestCommand(t, repoPath, "init")

	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoPath + `
    ref: definitely-missing-ref
`
	server := newTestServerFromManifest(t, manifest)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(httpServer.URL+"/api/workspace/validate", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/workspace/validate: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode validation payload: %v", err)
	}
	if payload.Error.Code != "workspace_validation_failed" {
		t.Fatalf("expected workspace_validation_failed code, got %q", payload.Error.Code)
	}
	hasInvalidRefError := false
	for _, diagnostic := range payload.Errors {
		if diagnostic.Code == "workspace.repo.ref.invalid" {
			hasInvalidRefError = true
			break
		}
	}
	if !hasInvalidRefError {
		t.Fatalf("expected workspace.repo.ref.invalid in diagnostics, got %+v", payload.Errors)
	}
}

func TestPipelineEndpoints(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(httpServer.URL+"/api/pipeline/init", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/pipeline/init: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", response.StatusCode)
	}

	var started map[string]any
	if err := json.NewDecoder(response.Body).Decode(&started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	runID, _ := started["run_id"].(string)
	if runID == "" {
		t.Fatalf("expected run_id in start response, got %+v", started)
	}

	runStatus := waitForRunTerminalStatus(t, httpServer.URL, runID, 8*time.Second)
	if runStatus.Status != string(orchestrator.RunStatusSucceeded) {
		t.Fatalf("expected async run success, got status=%q error_code=%q", runStatus.Status, runStatus.ErrorCode)
	}
}

func TestPipelineRunLogsEndpointSupportsPagination(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(httpServer.URL+"/api/pipeline/init", "application/json", bytes.NewBufferString(`{"trigger":"ui"}`))
	if err != nil {
		t.Fatalf("POST /api/pipeline/init: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", response.StatusCode)
	}

	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	if strings.TrimSpace(started.RunID) == "" {
		t.Fatalf("expected non-empty run_id")
	}

	runStatus := waitForRunTerminalStatus(t, httpServer.URL, started.RunID, 8*time.Second)
	if runStatus.Status != string(orchestrator.RunStatusSucceeded) {
		t.Fatalf("expected async run success, got status=%q error_code=%q", runStatus.Status, runStatus.ErrorCode)
	}

	logResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + started.RunID + "/logs?cursor=0&limit=2")
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs/<id>/logs: %v", err)
	}
	defer logResp.Body.Close()
	if logResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", logResp.StatusCode)
	}

	var firstPage struct {
		RunID string `json:"run_id"`
		Items []struct {
			Cursor  int    `json:"cursor"`
			Message string `json:"message"`
		} `json:"items"`
		NextCursor int  `json:"next_cursor"`
		EOF        bool `json:"eof"`
	}
	if err := json.NewDecoder(logResp.Body).Decode(&firstPage); err != nil {
		t.Fatalf("decode first logs page: %v", err)
	}
	if firstPage.RunID != started.RunID {
		t.Fatalf("expected run_id %q, got %q", started.RunID, firstPage.RunID)
	}
	if len(firstPage.Items) == 0 {
		t.Fatalf("expected non-empty first logs page")
	}
	if strings.TrimSpace(firstPage.Items[0].Message) == "" {
		t.Fatalf("expected non-empty log message")
	}
	if firstPage.NextCursor <= firstPage.Items[len(firstPage.Items)-1].Cursor {
		t.Fatalf("expected next_cursor to move forward, got next=%d last_cursor=%d", firstPage.NextCursor, firstPage.Items[len(firstPage.Items)-1].Cursor)
	}

	logResp2, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + started.RunID + "/logs?cursor=" + fmt.Sprintf("%d", firstPage.NextCursor) + "&limit=2")
	if err != nil {
		t.Fatalf("GET second logs page: %v", err)
	}
	defer logResp2.Body.Close()
	if logResp2.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for second page, got %d", logResp2.StatusCode)
	}

	var secondPage struct {
		Items []struct {
			Cursor int `json:"cursor"`
		} `json:"items"`
		NextCursor int  `json:"next_cursor"`
		EOF        bool `json:"eof"`
	}
	if err := json.NewDecoder(logResp2.Body).Decode(&secondPage); err != nil {
		t.Fatalf("decode second logs page: %v", err)
	}
	for _, item := range secondPage.Items {
		if item.Cursor < firstPage.NextCursor {
			t.Fatalf("expected cursor >= %d, got %d", firstPage.NextCursor, item.Cursor)
		}
	}
}

func TestPipelineRunLogsEndpointValidatesParams(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	resp, err := http.Get(httpServer.URL + "/api/pipeline/runs/run-unknown/logs?cursor=-1&limit=10")
	if err != nil {
		t.Fatalf("GET invalid cursor logs request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid cursor, got %d", resp.StatusCode)
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode invalid cursor error payload: %v", err)
	}
	if payload.Error.Code != "invalid_cursor" {
		t.Fatalf("expected invalid_cursor code, got %q", payload.Error.Code)
	}

	resp2, err := http.Get(httpServer.URL + "/api/pipeline/runs/run-unknown/logs?cursor=0&limit=0")
	if err != nil {
		t.Fatalf("GET invalid limit logs request: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid limit, got %d", resp2.StatusCode)
	}
	var payload2 struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&payload2); err != nil {
		t.Fatalf("decode invalid limit error payload: %v", err)
	}
	if payload2.Error.Code != "invalid_limit" {
		t.Fatalf("expected invalid_limit code, got %q", payload2.Error.Code)
	}
}

func TestPipelineRunLogsEndpointReturnsRunNotFound(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	resp, err := http.Get(httpServer.URL + "/api/pipeline/runs/run-missing/logs?cursor=0&limit=10")
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs/run-missing/logs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.StatusCode)
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode run-not-found payload: %v", err)
	}
	if payload.Error.Code != "run_not_found" {
		t.Fatalf("expected run_not_found code, got %q", payload.Error.Code)
	}
}

func TestPipelineRunsListEndpointReturnsRecentRuns(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	firstStartResp, err := http.Post(httpServer.URL+"/api/pipeline/init", "application/json", bytes.NewBufferString(`{"trigger":"ui"}`))
	if err != nil {
		t.Fatalf("POST /api/pipeline/init first run: %v", err)
	}
	defer firstStartResp.Body.Close()
	if firstStartResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected first run status 202, got %d", firstStartResp.StatusCode)
	}
	var firstPayload struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(firstStartResp.Body).Decode(&firstPayload); err != nil {
		t.Fatalf("decode first run payload: %v", err)
	}
	if firstPayload.RunID == "" {
		t.Fatalf("expected first run id")
	}
	_ = waitForRunTerminalStatus(t, httpServer.URL, firstPayload.RunID, 8*time.Second)

	secondStartResp, err := http.Post(httpServer.URL+"/api/pipeline/refresh", "application/json", bytes.NewBufferString(`{"trigger":"manual"}`))
	if err != nil {
		t.Fatalf("POST /api/pipeline/refresh second run: %v", err)
	}
	defer secondStartResp.Body.Close()
	if secondStartResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected second run status 202, got %d", secondStartResp.StatusCode)
	}
	var secondPayload struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(secondStartResp.Body).Decode(&secondPayload); err != nil {
		t.Fatalf("decode second run payload: %v", err)
	}
	if secondPayload.RunID == "" {
		t.Fatalf("expected second run id")
	}
	_ = waitForRunTerminalStatus(t, httpServer.URL, secondPayload.RunID, 8*time.Second)

	listResp, err := http.Get(httpServer.URL + "/api/pipeline/runs?limit=1")
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs?limit=1: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listResp.StatusCode)
	}
	var listPayload struct {
		Items []struct {
			RunID      string   `json:"run_id"`
			Pipeline   string   `json:"pipeline"`
			Status     string   `json:"status"`
			StartedAt  string   `json:"started_at"`
			FinishedAt *string  `json:"finished_at"`
			Warnings   []string `json:"warnings"`
			ErrorCode  any      `json:"error_code"`
			Error      any      `json:"error"`
		} `json:"items"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listPayload); err != nil {
		t.Fatalf("decode runs list payload: %v", err)
	}
	if len(listPayload.Items) != 1 {
		t.Fatalf("expected one run from limited list, got %d", len(listPayload.Items))
	}
	if listPayload.Items[0].RunID != secondPayload.RunID {
		t.Fatalf("expected newest run first, got %q want %q", listPayload.Items[0].RunID, secondPayload.RunID)
	}
	if listPayload.Items[0].Pipeline != "refresh" {
		t.Fatalf("expected pipeline refresh, got %q", listPayload.Items[0].Pipeline)
	}
	if listPayload.Items[0].StartedAt == "" || listPayload.Items[0].FinishedAt == nil {
		t.Fatalf("expected started_at and finished_at in list payload, got %+v", listPayload.Items[0])
	}
	if listPayload.Items[0].Status != "succeeded" {
		t.Fatalf("expected status succeeded, got %q", listPayload.Items[0].Status)
	}
}

func TestPipelineRunsListEndpointRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/pipeline/runs?limit=not-a-number")
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs invalid limit: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode invalid limit payload: %v", err)
	}
	if payload.Error.Code != "invalid_limit" {
		t.Fatalf("expected invalid_limit code, got %q", payload.Error.Code)
	}
}

func TestRunDetailAndArtifactsLoadFromPersistedHistoryAfterRestart(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "payments-service")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoPath + `
`
	ws, err := workspace.Open(writeManifestRootWithRoot(t, root, manifest))
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	serviceWithRuns := orchestrator.NewService(orchestrator.WithHistoryWorkspace(ws))
	runInfo, artifacts, err := serviceWithRuns.Run(context.Background(), orchestrator.RunRequest{
		Workspace:      ws,
		Pipeline:       orchestrator.PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run pipeline for persisted history setup: %v", err)
	}
	if runInfo.RunID == "" {
		t.Fatalf("expected run id from setup run")
	}
	if len(artifacts) == 0 {
		t.Fatalf("expected non-empty artifacts from setup run")
	}

	restartedService := orchestrator.NewService(orchestrator.WithHistoryWorkspace(ws))
	server := NewServer(ws, restartedService)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	runResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + runInfo.RunID)
	if err != nil {
		t.Fatalf("GET run from persisted history: %v", err)
	}
	defer runResp.Body.Close()
	if runResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for persisted run detail, got %d", runResp.StatusCode)
	}

	artifactsResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + runInfo.RunID + "/artifacts")
	if err != nil {
		t.Fatalf("GET run artifacts from persisted history: %v", err)
	}
	defer artifactsResp.Body.Close()
	if artifactsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for persisted run artifacts, got %d", artifactsResp.StatusCode)
	}
	var artifactsPayload struct {
		Artifacts []orchestrator.Artifact `json:"artifacts"`
	}
	if err := json.NewDecoder(artifactsResp.Body).Decode(&artifactsPayload); err != nil {
		t.Fatalf("decode persisted artifacts payload: %v", err)
	}
	if len(artifactsPayload.Artifacts) == 0 {
		t.Fatalf("expected non-empty artifacts from persisted history")
	}
}

func TestWorkspaceManifestRoundTrip(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	getResp, err := http.Get(httpServer.URL + "/api/workspace/manifest")
	if err != nil {
		t.Fatalf("GET /api/workspace/manifest: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode manifest payload: %v", err)
	}
	if !strings.Contains(payload.Content, "version: 1") {
		t.Fatalf("expected manifest content, got %q", payload.Content)
	}

	updateBody := `{"content":"version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/does-not-need-to-exist-for-contract\n"}`
	putReq, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/workspace/manifest", strings.NewReader(updateBody))
	if err != nil {
		t.Fatalf("create PUT request: %v", err)
	}
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatalf("PUT /api/workspace/manifest: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", putResp.StatusCode)
	}
}

func TestArtifactsWriteEndpoint(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	writeBody := `{"path":"charter/overview.md","content":"# Updated Charter"}`
	response, err := http.Post(httpServer.URL+"/api/artifacts/write", "application/json", strings.NewReader(writeBody))
	if err != nil {
		t.Fatalf("POST /api/artifacts/write: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	readResp, err := http.Get(httpServer.URL + "/api/artifacts?path=charter/overview.md")
	if err != nil {
		t.Fatalf("GET artifact: %v", err)
	}
	defer readResp.Body.Close()
	if readResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", readResp.StatusCode)
	}
	content, err := io.ReadAll(readResp.Body)
	if err != nil {
		t.Fatalf("read artifact body: %v", err)
	}
	if string(content) != "# Updated Charter" {
		t.Fatalf("unexpected artifact content: %q", string(content))
	}
}

func TestArtifactsEndpointRejectsPathTraversal(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/artifacts?path=../outside")
	if err != nil {
		t.Fatalf("GET /api/artifacts path traversal: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for path traversal, got %d", response.StatusCode)
	}
}

func TestPipelineStartRejectsInvalidBody(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/api/pipeline/init",
		"application/json",
		bytes.NewBufferString(`{"trigger":"ui"}{"extra":"payload"}`),
	)
	if err != nil {
		t.Fatalf("POST /api/pipeline/init invalid body: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid body, got %d", response.StatusCode)
	}

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Code != "invalid_request_body" {
		t.Fatalf("expected invalid_request_body code, got %q", payload.Error.Code)
	}
}

func TestPipelineStartRejectsUnsupportedTrigger(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/api/pipeline/refresh",
		"application/json",
		bytes.NewBufferString(`{"trigger":"cron"}`),
	)
	if err != nil {
		t.Fatalf("POST /api/pipeline/refresh unsupported trigger: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.StatusCode)
	}

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Code != "trigger_unsupported" {
		t.Fatalf("expected trigger_unsupported code, got %q", payload.Error.Code)
	}
}

func TestPipelineStartRejectsCommitAndBranchOpsInThisSlice(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/api/pipeline/init",
		"application/json",
		bytes.NewBufferString(`{"trigger":"ui","commit":true}`),
	)
	if err != nil {
		t.Fatalf("POST /api/pipeline/init not-supported flags: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", response.StatusCode)
	}

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Code != "not_supported" {
		t.Fatalf("expected not_supported code, got %q", payload.Error.Code)
	}
}

func TestPipelineStartRejectsCreateProposalBranchInThisSlice(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/api/pipeline/refresh",
		"application/json",
		bytes.NewBufferString(`{"trigger":"manual","create_proposal_branch":true}`),
	)
	if err != nil {
		t.Fatalf("POST /api/pipeline/refresh not-supported branch flag: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", response.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Code != "not_supported" {
		t.Fatalf("expected not_supported code, got %q", payload.Error.Code)
	}
}

func TestPipelineStartReturnsRunnerUnavailableEnvelope(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "payments-service")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoPath + `
`
	ws, err := workspace.Open(writeManifestRoot(t, manifest))
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	service := orchestrator.NewService(orchestrator.WithRunner(unavailableRunner{}))
	server := NewServer(ws, service)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/api/pipeline/init",
		"application/json",
		bytes.NewBufferString(`{"trigger":"ui"}`),
	)
	if err != nil {
		t.Fatalf("POST /api/pipeline/init runner unavailable: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", response.StatusCode)
	}

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Code != "runner_unavailable" {
		t.Fatalf("expected runner_unavailable code, got %q", payload.Error.Code)
	}
	if payload.Error.Message == "" {
		t.Fatalf("expected non-empty runner_unavailable message")
	}
}

func TestRunStatusIncludesRunnerErrorCode(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoPath + `
`
	ws, err := workspace.Open(writeManifestRoot(t, manifest))
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	service := orchestrator.NewService(orchestrator.WithRunner(parseFailureRunner{}))
	server := NewServer(ws, service)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/api/pipeline/init",
		"application/json",
		bytes.NewBufferString(`{"trigger":"ui"}`),
	)
	if err != nil {
		t.Fatalf("POST /api/pipeline/init parse failure runner: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", response.StatusCode)
	}

	var startPayload struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&startPayload); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}
	if startPayload.RunID == "" {
		t.Fatalf("expected non-empty run id")
	}

	runStatus := waitForRunTerminalStatus(t, httpServer.URL, startPayload.RunID, 8*time.Second)
	if runStatus.Status != string(orchestrator.RunStatusFailed) {
		t.Fatalf("expected failed run status, got %q", runStatus.Status)
	}
	if runStatus.ErrorCode != "runner_parse_failed" {
		t.Fatalf("expected runner_parse_failed, got %q", runStatus.ErrorCode)
	}
}

func TestPipelineStartWithParseFailureStillReturnsAccepted(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoPath + `
`
	ws, err := workspace.Open(writeManifestRoot(t, manifest))
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	service := orchestrator.NewService(orchestrator.WithRunner(parseFailureRunner{}))
	server := NewServer(ws, service)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/api/pipeline/init",
		"application/json",
		bytes.NewBufferString(`{"trigger":"ui"}`),
	)
	if err != nil {
		t.Fatalf("POST /api/pipeline/init parse failure runner: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", response.StatusCode)
	}

	var payload struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}
	if payload.RunID == "" {
		t.Fatalf("expected non-empty run id")
	}

	_ = waitForRunTerminalStatus(t, httpServer.URL, payload.RunID, 8*time.Second)
}

func TestMapTypedRunnerAPIErrorDoesNotExposeRunnerParseFailedAtStartTime(t *testing.T) {
	t.Parallel()

	err := claudecode.WrapRunnerError(
		claudecode.ErrorCodeRunnerParseFailed,
		"runner returned invalid taskresult in test",
		errors.New("json decode error"),
	)
	if _, code, _, ok := mapTypedRunnerAPIError(err); ok {
		t.Fatalf("expected runner_parse_failed to bypass start-time mapping, got mapped code %q", code)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "payments-service")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoPath + `
`
	return newTestServerFromManifest(t, manifest)
}

func newTestServerFromManifest(t *testing.T, manifest string) *Server {
	t.Helper()

	root := writeManifestRoot(t, manifest)
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	return NewServer(ws, orchestrator.NewService(orchestrator.WithHistoryWorkspace(ws)))
}

func writeManifestRoot(t *testing.T, manifest string) string {
	t.Helper()

	root := t.TempDir()
	return writeManifestRootWithRoot(t, root, manifest)
}

func writeManifestRootWithRoot(t *testing.T, root string, manifest string) string {
	t.Helper()

	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return root
}

func runGitTestCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
}

type runStatusPayload struct {
	Status    string `json:"status"`
	ErrorCode string `json:"error_code"`
}

func waitForRunTerminalStatus(t *testing.T, serverURL string, runID string, timeout time.Duration) runStatusPayload {
	t.Helper()

	var terminal runStatusPayload
	testutil.WaitFor(t, timeout, testutil.WaitDescription("run %q did not reach terminal status", runID), func() (bool, error) {
		runResp, err := http.Get(serverURL + "/api/pipeline/runs/" + runID)
		if err != nil {
			return false, err
		}
		defer runResp.Body.Close()
		if runResp.StatusCode != http.StatusOK {
			return false, nil
		}
		var payload runStatusPayload
		if err := json.NewDecoder(runResp.Body).Decode(&payload); err != nil {
			return false, err
		}
		if payload.Status == string(orchestrator.RunStatusSucceeded) || payload.Status == string(orchestrator.RunStatusFailed) {
			terminal = payload
			return true, nil
		}
		return false, nil
	})
	return terminal
}

type unavailableRunner struct{}

func (unavailableRunner) Run(context.Context, claudecode.Task) (claudecode.Result, error) {
	return claudecode.Result{}, claudecode.WrapRunnerError(
		claudecode.ErrorCodeRunnerUnavailable,
		"headless runner command is unavailable in test",
		nil,
	)
}

func (unavailableRunner) Preflight(context.Context) error {
	return claudecode.WrapRunnerError(
		claudecode.ErrorCodeRunnerUnavailable,
		"headless runner command is unavailable in test",
		nil,
	)
}

type parseFailureRunner struct{}

func (parseFailureRunner) Run(context.Context, claudecode.Task) (claudecode.Result, error) {
	return claudecode.Result{}, claudecode.WrapRunnerError(
		claudecode.ErrorCodeRunnerParseFailed,
		"runner returned invalid taskresult in test",
		nil,
	)
}

func (parseFailureRunner) Preflight(context.Context) error {
	return nil
}
