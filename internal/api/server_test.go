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
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/runtime/qwencode"
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
	var payload struct {
		OK            bool `json:"ok"`
		ResolvedRepos []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"resolved_repos"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode validate payload: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected ok=true")
	}
	if len(payload.ResolvedRepos) == 0 {
		t.Fatalf("expected resolved_repos in payload")
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

func TestPipelineEndpointsSucceedWithHeadlessClaudeRunnerStub(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, claudecode.HeadlessRunner{
		Command: writeHeadlessRunnerStub(t, "claude-code"),
	})
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

	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&started); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}
	if strings.TrimSpace(started.RunID) == "" {
		t.Fatalf("expected non-empty run_id")
	}

	runStatus := waitForRunTerminalStatus(t, httpServer.URL, started.RunID, 8*time.Second)
	if runStatus.Status != string(orchestrator.RunStatusSucceeded) {
		t.Fatalf("expected succeeded status, got %q", runStatus.Status)
	}
	if runStatus.ErrorCode != "" {
		t.Fatalf("expected empty error_code, got %q", runStatus.ErrorCode)
	}
}

func TestPipelineEndpointsSucceedWithHeadlessQwenRunnerStub(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, qwencode.HeadlessRunner{
		Command: writeHeadlessRunnerStub(t, "qwen-code"),
	})
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

	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&started); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}
	if strings.TrimSpace(started.RunID) == "" {
		t.Fatalf("expected non-empty run_id")
	}

	runStatus := waitForRunTerminalStatus(t, httpServer.URL, started.RunID, 8*time.Second)
	if runStatus.Status != string(orchestrator.RunStatusSucceeded) {
		t.Fatalf("expected succeeded status, got %q", runStatus.Status)
	}
	if runStatus.ErrorCode != "" {
		t.Fatalf("expected empty error_code, got %q", runStatus.ErrorCode)
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

func TestPipelineRunLogsEndpointIncludesRuntimeOutputWireShape(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, streamingRunLogsRunner{})
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

	logResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + started.RunID + "/logs?cursor=0&limit=500")
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs/<id>/logs: %v", err)
	}
	defer logResp.Body.Close()
	if logResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", logResp.StatusCode)
	}

	var payload struct {
		Items []struct {
			Kind   string         `json:"kind"`
			Stream string         `json:"stream,omitempty"`
			Fields map[string]any `json:"fields,omitempty"`
		} `json:"items"`
	}
	if err := json.NewDecoder(logResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode logs payload: %v", err)
	}
	if len(payload.Items) == 0 {
		t.Fatalf("expected non-empty logs payload")
	}

	hasEvent := false
	hasStdoutRaw := false
	hasStderrRaw := false
	hasTruncated := false
	for _, item := range payload.Items {
		if item.Kind == "event" {
			hasEvent = true
		}
		if item.Kind == "runtime_output" && item.Stream == "stdout" {
			hasStdoutRaw = true
			if truncated, ok := item.Fields["output_truncated"].(bool); ok && truncated {
				hasTruncated = true
			}
		}
		if item.Kind == "runtime_output" && item.Stream == "stderr" {
			hasStderrRaw = true
		}
	}
	if !hasEvent {
		t.Fatalf("expected event logs in mixed wire-shape")
	}
	if !hasStdoutRaw || !hasStderrRaw {
		t.Fatalf("expected runtime_output logs for stdout+stderr")
	}
	if !hasTruncated {
		t.Fatalf("expected truncation marker with fields.output_truncated=true")
	}
}

func TestPipelineRunLogsEndpointStreamFieldContract(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, streamingRunLogsRunner{})
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
	runStatus := waitForRunTerminalStatus(t, httpServer.URL, started.RunID, 8*time.Second)
	if runStatus.Status != string(orchestrator.RunStatusSucceeded) {
		t.Fatalf("expected async run success, got status=%q error_code=%q", runStatus.Status, runStatus.ErrorCode)
	}

	logResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + started.RunID + "/logs?cursor=0&limit=500")
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs/<id>/logs: %v", err)
	}
	defer logResp.Body.Close()
	if logResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", logResp.StatusCode)
	}

	var payload struct {
		Items []struct {
			Kind   string `json:"kind"`
			Stream string `json:"stream,omitempty"`
		} `json:"items"`
	}
	if err := json.NewDecoder(logResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode logs payload: %v", err)
	}
	if len(payload.Items) == 0 {
		t.Fatalf("expected non-empty logs payload")
	}

	seenEvent := false
	seenRuntime := false
	for _, item := range payload.Items {
		switch item.Kind {
		case "event":
			seenEvent = true
			if strings.TrimSpace(item.Stream) != "" {
				t.Fatalf("expected empty stream for event entries, got %q", item.Stream)
			}
		case "runtime_output":
			seenRuntime = true
			if item.Stream != "stdout" && item.Stream != "stderr" {
				t.Fatalf("expected runtime_output stream stdout|stderr, got %q", item.Stream)
			}
		}
	}
	if !seenEvent {
		t.Fatalf("expected at least one event log entry")
	}
	if !seenRuntime {
		t.Fatalf("expected at least one runtime_output entry")
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

func TestPipelineRunCancelEndpointAcceptsRunningRun(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, cancellableDelayedRunner{delay: 260 * time.Millisecond})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	startResp, err := http.Post(httpServer.URL+"/api/pipeline/refresh", "application/json", bytes.NewBufferString(`{"trigger":"ui"}`))
	if err != nil {
		t.Fatalf("POST /api/pipeline/refresh: %v", err)
	}
	defer startResp.Body.Close()
	if startResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", startResp.StatusCode)
	}

	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(startResp.Body).Decode(&started); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}
	if strings.TrimSpace(started.RunID) == "" {
		t.Fatalf("expected non-empty run id")
	}

	cancelReq, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/pipeline/runs/"+started.RunID+"/cancel", nil)
	if err != nil {
		t.Fatalf("create cancel request: %v", err)
	}
	cancelResp, err := http.DefaultClient.Do(cancelReq)
	if err != nil {
		t.Fatalf("POST /api/pipeline/runs/<id>/cancel: %v", err)
	}
	defer cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", cancelResp.StatusCode)
	}

	var cancelPayload struct {
		RunID  string `json:"run_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(cancelResp.Body).Decode(&cancelPayload); err != nil {
		t.Fatalf("decode cancel payload: %v", err)
	}
	if cancelPayload.RunID != started.RunID {
		t.Fatalf("expected run_id %q, got %q", started.RunID, cancelPayload.RunID)
	}
	if cancelPayload.Status != "cancel_requested" {
		t.Fatalf("expected cancel_requested status, got %q", cancelPayload.Status)
	}

	terminal := waitForRunTerminalStatus(t, httpServer.URL, started.RunID, 8*time.Second)
	if terminal.Status != string(orchestrator.RunStatusFailed) {
		t.Fatalf("expected canceled run to fail, got status=%q", terminal.Status)
	}
	if terminal.ErrorCode != "run_canceled" {
		t.Fatalf("expected error_code run_canceled, got %q", terminal.ErrorCode)
	}
}

func TestPipelineRunCancelEndpointReturnsRunNotCancelableAfterAcceptedCancel(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, cancellableDelayedRunner{delay: 260 * time.Millisecond})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	startResp, err := http.Post(httpServer.URL+"/api/pipeline/refresh", "application/json", bytes.NewBufferString(`{"trigger":"ui"}`))
	if err != nil {
		t.Fatalf("POST /api/pipeline/refresh: %v", err)
	}
	defer startResp.Body.Close()
	if startResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", startResp.StatusCode)
	}

	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(startResp.Body).Decode(&started); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}
	if strings.TrimSpace(started.RunID) == "" {
		t.Fatalf("expected non-empty run id")
	}

	cancelReq, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/pipeline/runs/"+started.RunID+"/cancel", nil)
	if err != nil {
		t.Fatalf("create first cancel request: %v", err)
	}
	cancelResp, err := http.DefaultClient.Do(cancelReq)
	if err != nil {
		t.Fatalf("first POST /api/pipeline/runs/<id>/cancel: %v", err)
	}
	defer cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected first cancel status 202, got %d", cancelResp.StatusCode)
	}

	_ = waitForRunTerminalStatus(t, httpServer.URL, started.RunID, 8*time.Second)

	secondCancelReq, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/pipeline/runs/"+started.RunID+"/cancel", nil)
	if err != nil {
		t.Fatalf("create second cancel request: %v", err)
	}
	secondCancelResp, err := http.DefaultClient.Do(secondCancelReq)
	if err != nil {
		t.Fatalf("second POST /api/pipeline/runs/<id>/cancel: %v", err)
	}
	defer secondCancelResp.Body.Close()
	if secondCancelResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected second cancel status 409, got %d", secondCancelResp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(secondCancelResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode second cancel payload: %v", err)
	}
	if payload.Error.Code != "run_not_cancelable" {
		t.Fatalf("expected run_not_cancelable code on second cancel, got %q", payload.Error.Code)
	}
}

func TestPipelineRunCancelEndpointReturnsRunNotFound(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	cancelReq, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/pipeline/runs/run-missing/cancel", nil)
	if err != nil {
		t.Fatalf("create cancel request: %v", err)
	}
	cancelResp, err := http.DefaultClient.Do(cancelReq)
	if err != nil {
		t.Fatalf("POST /api/pipeline/runs/run-missing/cancel: %v", err)
	}
	defer cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", cancelResp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(cancelResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode run-not-found payload: %v", err)
	}
	if payload.Error.Code != "run_not_found" {
		t.Fatalf("expected run_not_found code, got %q", payload.Error.Code)
	}
}

func TestPipelineRunCancelEndpointReturnsRunNotCancelableForTerminalRun(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	startResp, err := http.Post(httpServer.URL+"/api/pipeline/init", "application/json", bytes.NewBufferString(`{"trigger":"ui"}`))
	if err != nil {
		t.Fatalf("POST /api/pipeline/init: %v", err)
	}
	defer startResp.Body.Close()
	if startResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", startResp.StatusCode)
	}

	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(startResp.Body).Decode(&started); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}
	if strings.TrimSpace(started.RunID) == "" {
		t.Fatalf("expected non-empty run id")
	}
	_ = waitForRunTerminalStatus(t, httpServer.URL, started.RunID, 8*time.Second)

	cancelReq, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/pipeline/runs/"+started.RunID+"/cancel", nil)
	if err != nil {
		t.Fatalf("create cancel request: %v", err)
	}
	cancelResp, err := http.DefaultClient.Do(cancelReq)
	if err != nil {
		t.Fatalf("POST /api/pipeline/runs/<id>/cancel: %v", err)
	}
	defer cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", cancelResp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(cancelResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode run-not-cancelable payload: %v", err)
	}
	if payload.Error.Code != "run_not_cancelable" {
		t.Fatalf("expected run_not_cancelable code, got %q", payload.Error.Code)
	}
}

func TestPipelineRunCancelEndpointRejectsInvalidRequestBody(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, cancellableDelayedRunner{delay: 260 * time.Millisecond})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	startResp, err := http.Post(httpServer.URL+"/api/pipeline/refresh", "application/json", bytes.NewBufferString(`{"trigger":"ui"}`))
	if err != nil {
		t.Fatalf("POST /api/pipeline/refresh: %v", err)
	}
	defer startResp.Body.Close()
	if startResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", startResp.StatusCode)
	}

	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(startResp.Body).Decode(&started); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}
	if strings.TrimSpace(started.RunID) == "" {
		t.Fatalf("expected non-empty run id")
	}

	cancelReq, err := http.NewRequest(
		http.MethodPost,
		httpServer.URL+"/api/pipeline/runs/"+started.RunID+"/cancel",
		bytes.NewBufferString(`{"unexpected":"field"}`),
	)
	if err != nil {
		t.Fatalf("create cancel request: %v", err)
	}
	cancelReq.Header.Set("Content-Type", "application/json")
	cancelResp, err := http.DefaultClient.Do(cancelReq)
	if err != nil {
		t.Fatalf("POST /api/pipeline/runs/<id>/cancel invalid body: %v", err)
	}
	defer cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", cancelResp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(cancelResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode invalid body payload: %v", err)
	}
	if payload.Error.Code != "invalid_request_body" {
		t.Fatalf("expected invalid_request_body code, got %q", payload.Error.Code)
	}

	_ = waitForRunTerminalStatus(t, httpServer.URL, started.RunID, 8*time.Second)
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

func TestRuntimeTimeoutsGetReturnsEffectiveDefaults(t *testing.T) {
	clearRuntimeTimeoutEnvForTest(t)

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/runtime/timeouts")
	if err != nil {
		t.Fatalf("GET /api/runtime/timeouts: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var payload struct {
		OK        bool                            `json:"ok"`
		Persisted workspace.RuntimeTimeoutsConfig `json:"persisted"`
		Effective acpruntime.TimeoutValues        `json:"effective"`
		Source    acpruntime.TimeoutSources       `json:"source"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode runtime timeouts payload: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected ok=true")
	}
	if payload.Effective.StepTimeoutSec != acpruntime.DefaultRuntimeStepTimeoutSec {
		t.Fatalf("expected default step timeout %d, got %d", acpruntime.DefaultRuntimeStepTimeoutSec, payload.Effective.StepTimeoutSec)
	}
	if payload.Source.StepTimeoutSec != acpruntime.TimeoutSourceDefault {
		t.Fatalf("expected default source, got %s", payload.Source.StepTimeoutSec)
	}
}

func TestRuntimeTimeoutsPutSupportsPartialUpdate(t *testing.T) {
	clearRuntimeTimeoutEnvForTest(t)

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	requestBody := `{"timeouts":{"step_timeout_sec":1200,"ui_cancel_poll_timeout_sec":555}}`
	request, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/runtime/timeouts", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("create PUT /api/runtime/timeouts request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("PUT /api/runtime/timeouts: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	manifestResp, err := http.Get(httpServer.URL + "/api/workspace/manifest")
	if err != nil {
		t.Fatalf("GET /api/workspace/manifest: %v", err)
	}
	defer manifestResp.Body.Close()
	var manifestPayload struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(manifestResp.Body).Decode(&manifestPayload); err != nil {
		t.Fatalf("decode manifest payload: %v", err)
	}
	if !strings.Contains(manifestPayload.Content, "runtime:") || !strings.Contains(manifestPayload.Content, "step_timeout_sec: 1200") {
		t.Fatalf("expected runtime timeout in manifest content, got:\n%s", manifestPayload.Content)
	}
	if !strings.Contains(manifestPayload.Content, "ui_cancel_poll_timeout_sec: 555") {
		t.Fatalf("expected ui cancel timeout in manifest content, got:\n%s", manifestPayload.Content)
	}
}

func TestRuntimeTimeoutsPutRejectsInvalidValues(t *testing.T) {
	clearRuntimeTimeoutEnvForTest(t)

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	requestBody := `{"timeouts":{"step_timeout_sec":0}}`
	request, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/runtime/timeouts", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("create PUT /api/runtime/timeouts request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("PUT /api/runtime/timeouts: %v", err)
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
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Code != "runtime_timeouts_invalid" {
		t.Fatalf("expected runtime_timeouts_invalid code, got %q", payload.Error.Code)
	}
}

func clearRuntimeTimeoutEnvForTest(t *testing.T) {
	t.Helper()
	keys := []string{
		"ACP_RUNTIME_STEP_TIMEOUT_SEC",
		"ACP_RUNTIME_HEARTBEAT_SEC",
		"ACP_PIPELINE_TIMEOUT_SEC",
		"ACP_PIPELINE_KILL_GRACE_SEC",
		"ACP_API_READY_TIMEOUT_SEC",
		"ACP_API_INIT_TIMEOUT_SEC",
		"ACP_UI_INIT_POLL_TIMEOUT_SEC",
		"ACP_UI_CANCEL_POLL_TIMEOUT_SEC",
	}
	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func TestRuntimeExecutionGetReturnsEffectiveDefaults(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/runtime/execution")
	if err != nil {
		t.Fatalf("GET /api/runtime/execution: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var payload struct {
		OK        bool                        `json:"ok"`
		Persisted map[string]any              `json:"persisted"`
		Effective acpruntime.ExecutionValues  `json:"effective"`
		Source    acpruntime.ExecutionSources `json:"source"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode runtime execution payload: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected ok=true")
	}
	if payload.Effective.Strategy != acpruntime.DefaultExecutionStrategy {
		t.Fatalf("expected default strategy %q, got %q", acpruntime.DefaultExecutionStrategy, payload.Effective.Strategy)
	}
	if payload.Effective.MaxParallel != acpruntime.DefaultExecutionMaxParallel {
		t.Fatalf("expected default max_parallel_tasks %d, got %d", acpruntime.DefaultExecutionMaxParallel, payload.Effective.MaxParallel)
	}
	if payload.Source.Strategy != acpruntime.ExecutionSourceDefault {
		t.Fatalf("expected default source, got %s", payload.Source.Strategy)
	}
}

func TestRuntimeExecutionPutSupportsPartialUpdate(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	requestBody := `{"execution":{"strategy":"parallel","max_parallel_tasks":3,"failure_policy":"fail_fast","shard_discovery_mode":"semantic"}}`
	request, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/runtime/execution", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("create PUT /api/runtime/execution request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("PUT /api/runtime/execution: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	manifestResp, err := http.Get(httpServer.URL + "/api/workspace/manifest")
	if err != nil {
		t.Fatalf("GET /api/workspace/manifest: %v", err)
	}
	defer manifestResp.Body.Close()
	var manifestPayload struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(manifestResp.Body).Decode(&manifestPayload); err != nil {
		t.Fatalf("decode manifest payload: %v", err)
	}
	if !strings.Contains(manifestPayload.Content, "runtime:") || !strings.Contains(manifestPayload.Content, "profile:") || !strings.Contains(manifestPayload.Content, "execution:") {
		t.Fatalf("expected runtime.profile.execution in manifest content, got:\n%s", manifestPayload.Content)
	}
	if !strings.Contains(manifestPayload.Content, "strategy: parallel") {
		t.Fatalf("expected execution strategy in manifest content, got:\n%s", manifestPayload.Content)
	}
	if !strings.Contains(manifestPayload.Content, "max_parallel_tasks: 3") {
		t.Fatalf("expected max_parallel_tasks in manifest content, got:\n%s", manifestPayload.Content)
	}
	if !strings.Contains(manifestPayload.Content, "failure_policy: fail_fast") {
		t.Fatalf("expected failure_policy in manifest content, got:\n%s", manifestPayload.Content)
	}
	if !strings.Contains(manifestPayload.Content, "mode: semantic") {
		t.Fatalf("expected shard_discovery.mode in manifest content, got:\n%s", manifestPayload.Content)
	}
}

func TestRuntimeExecutionPutRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	requestBody := `{"execution":{"max_parallel_tasks":0}}`
	request, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/runtime/execution", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("create PUT /api/runtime/execution request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("PUT /api/runtime/execution: %v", err)
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
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Code != "runtime_execution_invalid" {
		t.Fatalf("expected runtime_execution_invalid code, got %q", payload.Error.Code)
	}
}

func TestRuntimeProfileGetIncludesStepProviders(t *testing.T) {
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
runtime:
  profile:
    execution:
      strategy: parallel
      max_parallel_tasks: 2
    steps:
      step2_as_is:
        provider: qwen-code
`
	server := newTestServerFromManifest(t, manifest)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/runtime/profile")
	if err != nil {
		t.Fatalf("GET /api/runtime/profile: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var payload struct {
		OK            bool `json:"ok"`
		StepProviders struct {
			Persisted map[string]string `json:"persisted"`
			Effective map[string]string `json:"effective"`
			Source    map[string]string `json:"source"`
		} `json:"step_providers"`
		Execution struct {
			Effective map[string]any `json:"effective"`
		} `json:"execution"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode runtime profile payload: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected ok=true")
	}
	if payload.StepProviders.Persisted["step2_as_is"] != "qwen-code" {
		t.Fatalf("expected persisted step2_as_is=qwen-code, got %+v", payload.StepProviders.Persisted)
	}
	if payload.StepProviders.Effective["step2_as_is"] != "qwen-code" {
		t.Fatalf("expected effective step2_as_is=qwen-code, got %+v", payload.StepProviders.Effective)
	}
	if payload.StepProviders.Effective["step1_collect"] != "claude-code" {
		t.Fatalf("expected default effective step1_collect=claude-code, got %+v", payload.StepProviders.Effective)
	}
	if payload.StepProviders.Source["step2_as_is"] != "workspace" {
		t.Fatalf("expected workspace source for step2_as_is, got %+v", payload.StepProviders.Source)
	}
	effectiveSteps, ok := payload.Execution.Effective["steps"].(map[string]any)
	if !ok {
		t.Fatalf("expected execution.effective.steps map, got %+v", payload.Execution.Effective)
	}
	if effectiveSteps["step2_as_is"] != "qwen-code" {
		t.Fatalf("expected execution.effective.steps.step2_as_is=qwen-code, got %+v", effectiveSteps)
	}
}

func TestRuntimeExecutionPutSupportsStepProviderUpdate(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	requestBody := `{"execution":{"strategy":"parallel","steps":{"step2_as_is":"qwen-code","step4_proposals":"claude-code"}}}`
	request, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/runtime/execution", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("create PUT /api/runtime/execution request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("PUT /api/runtime/execution: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	manifestResp, err := http.Get(httpServer.URL + "/api/workspace/manifest")
	if err != nil {
		t.Fatalf("GET /api/workspace/manifest: %v", err)
	}
	defer manifestResp.Body.Close()
	var manifestPayload struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(manifestResp.Body).Decode(&manifestPayload); err != nil {
		t.Fatalf("decode manifest payload: %v", err)
	}
	if !strings.Contains(manifestPayload.Content, "steps:") || !strings.Contains(manifestPayload.Content, "step2_as_is:") {
		t.Fatalf("expected runtime.profile.steps in manifest content, got:\n%s", manifestPayload.Content)
	}
	if !strings.Contains(manifestPayload.Content, "provider: qwen-code") {
		t.Fatalf("expected qwen step provider in manifest content, got:\n%s", manifestPayload.Content)
	}
}

func TestRunStatusIncludesEffectiveStepProviders(t *testing.T) {
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
runtime:
  profile:
    steps:
      step2_as_is:
        provider: qwen-code
`
	server := newTestServerFromManifest(t, manifest)
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
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&started); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}
	if started.RunID == "" {
		t.Fatalf("expected run id")
	}

	runStatus := waitForRunTerminalStatus(t, httpServer.URL, started.RunID, 8*time.Second)
	if runStatus.Status != string(orchestrator.RunStatusSucceeded) {
		t.Fatalf("expected succeeded status, got %q", runStatus.Status)
	}

	detailResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + started.RunID)
	if err != nil {
		t.Fatalf("GET run detail: %v", err)
	}
	defer detailResp.Body.Close()
	var detail struct {
		StepProviders map[string]string `json:"step_providers"`
	}
	if err := json.NewDecoder(detailResp.Body).Decode(&detail); err != nil {
		t.Fatalf("decode run detail payload: %v", err)
	}
	if detail.StepProviders["step2_as_is"] != "qwen-code" {
		t.Fatalf("expected run detail step2_as_is=qwen-code, got %+v", detail.StepProviders)
	}
	if detail.StepProviders["step1_collect"] != "claude-code" {
		t.Fatalf("expected default step1_collect=claude-code, got %+v", detail.StepProviders)
	}
}

func TestPipelineRunReflectsSequentialVsParallelExecutionProfileInLogsAndShardArtifacts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		strategy    string
		maxParallel int
	}{
		{name: "sequential", strategy: "sequential", maxParallel: 1},
		{name: "parallel", strategy: "parallel", maxParallel: 3},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			repoPath := filepath.Join(root, "repos", "orders-monolith")
			if err := os.MkdirAll(filepath.Join(repoPath, "services", "api"), 0o755); err != nil {
				t.Fatalf("create api module dir: %v", err)
			}
			if err := os.MkdirAll(filepath.Join(repoPath, "services", "web"), 0o755); err != nil {
				t.Fatalf("create web module dir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(repoPath, "services", "api", "package.json"), []byte("{\"name\":\"api\"}\n"), 0o644); err != nil {
				t.Fatalf("write api package.json: %v", err)
			}
			if err := os.WriteFile(filepath.Join(repoPath, "services", "web", "package.json"), []byte("{\"name\":\"web\"}\n"), 0o644); err != nil {
				t.Fatalf("write web package.json: %v", err)
			}
			if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# orders-monolith\n"), 0o644); err != nil {
				t.Fatalf("write monolith readme: %v", err)
			}

			manifest := fmt.Sprintf(`version: 1
repos:
  - name: orders-monolith
    path: %s
runtime:
  profile:
    execution:
      strategy: %s
      max_parallel_tasks: %d
      failure_policy: best_effort
`, repoPath, testCase.strategy, testCase.maxParallel)
			ws, err := workspace.Open(writeManifestRootWithRoot(t, root, manifest))
			if err != nil {
				t.Fatalf("open workspace: %v", err)
			}
			service := orchestrator.NewService(orchestrator.WithHistoryWorkspace(ws))
			server := NewServer(ws, service)
			httpServer := httptest.NewServer(server.Handler())
			defer httpServer.Close()

			response, err := http.Post(
				httpServer.URL+"/api/pipeline/init",
				"application/json",
				bytes.NewBufferString(`{"trigger":"ui"}`),
			)
			if err != nil {
				t.Fatalf("POST /api/pipeline/init: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusAccepted {
				t.Fatalf("expected status 202, got %d", response.StatusCode)
			}
			var startPayload struct {
				RunID string `json:"run_id"`
			}
			if err := json.NewDecoder(response.Body).Decode(&startPayload); err != nil {
				t.Fatalf("decode run start payload: %v", err)
			}
			if strings.TrimSpace(startPayload.RunID) == "" {
				t.Fatalf("expected non-empty run_id")
			}

			runStatus := waitForRunTerminalStatus(t, httpServer.URL, startPayload.RunID, 8*time.Second)
			if runStatus.Status != string(orchestrator.RunStatusSucceeded) {
				t.Fatalf("expected succeeded status, got %q (%q)", runStatus.Status, runStatus.ErrorCode)
			}

			logsResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + startPayload.RunID + "/logs?cursor=0&limit=500")
			if err != nil {
				t.Fatalf("GET run logs: %v", err)
			}
			defer logsResp.Body.Close()
			if logsResp.StatusCode != http.StatusOK {
				t.Fatalf("expected status 200 for logs, got %d", logsResp.StatusCode)
			}
			var logsPayload struct {
				Items []struct {
					StepID  string         `json:"step_id"`
					Message string         `json:"message"`
					Fields  map[string]any `json:"fields"`
				} `json:"items"`
			}
			if err := json.NewDecoder(logsResp.Body).Decode(&logsPayload); err != nil {
				t.Fatalf("decode run logs payload: %v", err)
			}
			foundStep1Prepared := false
			for _, item := range logsPayload.Items {
				if item.StepID != "init.step1.collect" || item.Message != "runtime shard execution prepared" {
					continue
				}
				foundStep1Prepared = true
				strategyValue, _ := item.Fields["strategy"].(string)
				if strategyValue != testCase.strategy {
					t.Fatalf("expected logs strategy %q, got %q", testCase.strategy, strategyValue)
				}
				maxParallelValue, ok := item.Fields["max_parallel"].(float64)
				if !ok {
					t.Fatalf("expected numeric max_parallel in logs fields, got %#v", item.Fields["max_parallel"])
				}
				if int(maxParallelValue) != testCase.maxParallel {
					t.Fatalf("expected max_parallel=%d in logs, got %v", testCase.maxParallel, maxParallelValue)
				}
				break
			}
			if !foundStep1Prepared {
				t.Fatalf("expected step1 shard execution log entry in run logs")
			}

			artifactsResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + startPayload.RunID + "/artifacts")
			if err != nil {
				t.Fatalf("GET run artifacts: %v", err)
			}
			defer artifactsResp.Body.Close()
			if artifactsResp.StatusCode != http.StatusOK {
				t.Fatalf("expected status 200 for artifacts, got %d", artifactsResp.StatusCode)
			}
			var artifactsPayload struct {
				Artifacts []orchestrator.Artifact `json:"artifacts"`
			}
			if err := json.NewDecoder(artifactsResp.Body).Decode(&artifactsPayload); err != nil {
				t.Fatalf("decode artifacts payload: %v", err)
			}
			summaryPath := ""
			for _, artifact := range artifactsPayload.Artifacts {
				if strings.Contains(artifact.Path, "-init-step1-collect-shard-summary-") {
					summaryPath = artifact.Path
					break
				}
			}
			if summaryPath == "" {
				t.Fatalf("expected step1 shard summary artifact path in run artifacts")
			}
			summaryRaw, err := ws.ReadFile(summaryPath)
			if err != nil {
				t.Fatalf("read shard summary artifact %q: %v", summaryPath, err)
			}
			var summary struct {
				Strategy    string `json:"strategy"`
				MaxParallel int    `json:"max_parallel_tasks"`
				Items       []struct {
					PathScopes []string `json:"path_scopes"`
				} `json:"items"`
			}
			if err := json.Unmarshal(summaryRaw, &summary); err != nil {
				t.Fatalf("decode shard summary artifact %q: %v", summaryPath, err)
			}
			if summary.Strategy != testCase.strategy {
				t.Fatalf("expected shard summary strategy %q, got %q", testCase.strategy, summary.Strategy)
			}
			if summary.MaxParallel != testCase.maxParallel {
				t.Fatalf("expected shard summary max_parallel_tasks=%d, got %d", testCase.maxParallel, summary.MaxParallel)
			}
			if len(summary.Items) < 2 {
				t.Fatalf("expected at least two shard summary items for monolith modules, got %d", len(summary.Items))
			}
		})
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

func TestPipelineStartAcceptsAsyncRunAndReportsRunnerUnavailableInRunStatus(t *testing.T) {
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

	runStatus := waitForRunTerminalStatus(t, httpServer.URL, payload.RunID, 8*time.Second)
	if runStatus.Status != string(orchestrator.RunStatusFailed) {
		t.Fatalf("expected failed run status, got %q", runStatus.Status)
	}
	if runStatus.ErrorCode != "runner_unavailable" {
		t.Fatalf("expected runner_unavailable, got %q", runStatus.ErrorCode)
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

	err := acpruntime.WrapRunnerError(
		acpruntime.ProviderClaudeCode,
		acpruntime.ErrorCodeRunnerParseFailed,
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

func newTestServerWithRunner(t *testing.T, runner acpruntime.Runner) *Server {
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
	root = writeManifestRootWithRoot(t, root, manifest)
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	service := orchestrator.NewService(
		orchestrator.WithHistoryWorkspace(ws),
		orchestrator.WithRunner(runner),
	)
	return NewServer(ws, service)
}

func writeHeadlessRunnerStub(t *testing.T, runtimeName string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "headless-runner-stub.sh")
	script := `#!/usr/bin/env bash
set -eu
TASK_PAYLOAD="$(cat)"
LAST_ARG=""
for arg in "$@"; do
  LAST_ARG="$arg"
done
TASK_PAYLOAD="$TASK_PAYLOAD" TASK_PROMPT="$LAST_ARG" python3 - <<'PY'
import json
import os
import re
import sys

raw = os.environ.get("TASK_PAYLOAD", "").strip()
prompt = os.environ.get("TASK_PROMPT", "")

def first_non_empty(mapping, keys):
    for key in keys:
        value = mapping.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()
    return ""

def from_prompt(field):
    match = re.search(r'"%s"\s*:\s*"([^"]+)"' % re.escape(field), prompt)
    return match.group(1).strip() if match else ""

def first_non_empty_list(mapping, keys):
    for key in keys:
        value = mapping.get(key)
        if isinstance(value, list) and value:
            return [str(item).strip() for item in value if str(item).strip()]
    return []

def slugify(value):
    return re.sub(r'[^a-z0-9]+', '-', value.lower()).strip('-') or "stub"

task = {}
if raw:
    try:
        task = json.loads(raw)
    except Exception:
        task = {}

task_id = first_non_empty(task, ["task_id", "TaskID"]) or from_prompt("TaskID") or from_prompt("task_id") or "task"
step_id = first_non_empty(task, ["step_id", "StepID"]) or from_prompt("StepID") or from_prompt("step_id") or "init.step1.collect"
run_id = first_non_empty(task, ["run_id", "RunID"]) or from_prompt("RunID") or from_prompt("run_id")
write_root = first_non_empty(task, ["write_root", "WriteRoot"]) or from_prompt("write_root") or from_prompt("WriteRoot")
artifact_root = first_non_empty(task, ["artifact_root", "ArtifactRoot"]) or from_prompt("artifact_root") or from_prompt("ArtifactRoot")
draft_root = first_non_empty(task, ["draft_final_root", "DraftFinalRoot"]) or from_prompt("draft_final_root") or from_prompt("DraftFinalRoot")
step_contract = first_non_empty(task, ["step_contract", "StepContract"]) or from_prompt("step_contract") or from_prompt("StepContract")
agent_role = first_non_empty(task, ["agent_role", "AgentRole"]) or from_prompt("agent_role") or from_prompt("AgentRole") or "architect"
shard_id = first_non_empty(task, ["shard_id", "ShardID"]) or from_prompt("shard_id") or from_prompt("ShardID") or slugify(step_id)
repo_scopes = first_non_empty_list(task, ["repo_scopes", "RepoScopes"])
if not repo_scopes:
    repo_scope = first_non_empty(task, ["repo_scope", "RepoScope"]) or from_prompt("repo_scope") or from_prompt("RepoScope")
    if repo_scope:
        repo_scopes = [repo_scope]
path_scopes = first_non_empty_list(task, ["path_scopes", "PathScopes"])

def write_runtime_draft(manifest_name, draft_name, canonical_path, kind, title):
    if not write_root or not draft_root:
        return
    os.makedirs(write_root, exist_ok=True)
    os.makedirs(draft_root, exist_ok=True)
    with open(os.path.join(draft_root, draft_name), "w", encoding="utf-8") as handle:
        handle.write("# Stub Draft\n")
    manifest = {
        "version": 1,
        "run_id": run_id or "run-1",
        "step_id": step_id,
        "step_contract": step_contract or "draft",
        "agent_role": agent_role,
        "summary": "stub runtime draft",
        "outputs": [
            {
                "path": draft_name,
                "canonical_path": canonical_path,
                "kind": kind,
                "title": title,
            }
        ],
    }
    with open(os.path.join(write_root, manifest_name), "w", encoding="utf-8") as handle:
        json.dump(manifest, handle)

if step_id == "init.step0.constitution":
    write_runtime_draft("constitution-draft.json", "overview.md", "charter/overview.md", "charter", "Stub Constitution")
elif step_id in {"init.step2.asis_docs", "refresh.step2.asis_docs"}:
    write_runtime_draft("asis-draft-manifest.json", "overview.md", "reports/as-is/overview.md", "report", "Stub As-Is Overview")
elif step_id in {"init.step4.proposals", "refresh.step4.proposals"}:
    write_runtime_draft("proposals-draft-manifest.json", "proposal.md", "proposals/proposal-baseline/proposal.md", "proposal", "Stub Proposal")

if step_id in {"init.step1.collect", "refresh.step1.collect"} and write_root:
    os.makedirs(write_root, exist_ok=True)
    document_name = slugify(shard_id) + ".md"
    document_id = "doc." + slugify(shard_id)
    citation_id = "cite." + slugify(shard_id)
    canonical_path = "reports/agent-outputs/domains/" + document_name
    with open(os.path.join(write_root, document_name), "w", encoding="utf-8") as handle:
        handle.write("# Stub Analysis\n")
    manifest = {
        "version": 1,
        "run_id": run_id or "run-1",
        "step_id": step_id,
        "shard_id": shard_id,
        "agent_role": "shard-analyst",
        "artifact_root": write_root,
        "repo_scopes": repo_scopes,
        "path_scopes": path_scopes,
        "summary": "stub shard pack",
        "documents": [
            {
                "id": document_id,
                "kind": "report",
                "title": "Stub Analysis",
                "path": document_name,
                "canonical_path": canonical_path,
                "topics": ["stub"],
                "citation_ids": [citation_id],
                "status": "staged"
            }
        ],
        "citations": [
            {
                "id": citation_id,
                "repo": repo_scopes[0] if repo_scopes else "stub-repo",
                "path": "README.md",
                "claim_ids": ["claim.stub"],
                "document_ids": [document_id]
            }
        ],
        "compatibility": {
            "coverage": {
                "observed": ["stub"],
                "missing": ["owner mappings"],
                "notes": ["stub manifest for integration tests"]
            },
            "questions": [],
            "entities": [],
            "edges": [],
            "findings": []
        }
    }
    with open(os.path.join(write_root, "shard-pack-manifest.json"), "w", encoding="utf-8") as handle:
        json.dump(manifest, handle)
payload = {
    "meta": {
        "task_id": task_id,
        "step_id": step_id,
        "run_id": run_id,
        "runtime": {
            "name": "` + runtimeName + `",
            "version": "stub"
        },
        "started_at": "2026-04-03T12:00:00Z"
    },
    "summary": "stub taskresult",
    "changeset": []
}
print(json.dumps(payload))
PY
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write runner stub: %v", err)
	}
	return path
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

func (unavailableRunner) Run(context.Context, acpruntime.Task) (acpruntime.Result, error) {
	return acpruntime.Result{}, acpruntime.WrapRunnerError(
		acpruntime.ProviderClaudeCode,
		acpruntime.ErrorCodeRunnerUnavailable,
		"headless runner command is unavailable in test",
		nil,
	)
}

func (unavailableRunner) Preflight(context.Context) error {
	return acpruntime.WrapRunnerError(
		acpruntime.ProviderClaudeCode,
		acpruntime.ErrorCodeRunnerUnavailable,
		"headless runner command is unavailable in test",
		nil,
	)
}

type parseFailureRunner struct{}

func (parseFailureRunner) Run(context.Context, acpruntime.Task) (acpruntime.Result, error) {
	return acpruntime.Result{}, acpruntime.WrapRunnerError(
		acpruntime.ProviderClaudeCode,
		acpruntime.ErrorCodeRunnerParseFailed,
		"runner returned invalid taskresult in test",
		nil,
	)
}

func (parseFailureRunner) Preflight(context.Context) error {
	return nil
}

type cancellableDelayedRunner struct {
	delay time.Duration
}

func (r cancellableDelayedRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	select {
	case <-ctx.Done():
		return acpruntime.Result{}, ctx.Err()
	case <-time.After(r.delay):
	}
	return claudecode.FakeRunner{}.Run(ctx, task)
}

func (cancellableDelayedRunner) Preflight(context.Context) error {
	return nil
}

type streamingRunLogsRunner struct{}

func (streamingRunLogsRunner) Run(_ context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if err := writeSyntheticServerDraftArtifacts(task); err != nil {
		return acpruntime.Result{}, err
	}
	if task.OnOutput != nil {
		task.OnOutput(acpruntime.OutputChunk{
			Stream: acpruntime.OutputStreamStdout,
			Text:   "stream stdout line",
		})
		task.OnOutput(acpruntime.OutputChunk{
			Stream: acpruntime.OutputStreamStderr,
			Text:   "stream stderr line",
		})
		task.OnOutput(acpruntime.OutputChunk{
			Stream:    acpruntime.OutputStreamStdout,
			Truncated: true,
			Text:      "stdout output truncated after cap (synthetic)",
		})
	}

	payload := map[string]any{
		"meta": map[string]any{
			"task_id":    task.TaskID,
			"step_id":    task.StepID,
			"run_id":     task.RunID,
			"runtime":    map[string]any{"name": "streaming-test-runner", "version": "v1"},
			"started_at": task.StartedAtUTC.Format(time.RFC3339),
		},
		"summary":   "synthetic streaming success",
		"changeset": []any{},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return acpruntime.Result{}, err
	}
	return acpruntime.Result{RawJSON: raw}, nil
}

func (streamingRunLogsRunner) Preflight(context.Context) error {
	return nil
}

func writeSyntheticServerDraftArtifacts(task acpruntime.Task) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	draftRoot := strings.TrimSpace(task.DraftFinalRoot)
	if writeRoot == "" || draftRoot == "" {
		return nil
	}

	type draftSpec struct {
		manifest string
		content  string
		outputs  []map[string]any
	}

	var spec draftSpec
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		spec = draftSpec{
			manifest: "constitution-draft.json",
			content:  "# Stub Constitution\n",
			outputs: []map[string]any{
				{
					"path":           "overview.md",
					"canonical_path": "charter/overview.md",
					"kind":           "charter",
					"title":          "Stub Constitution",
				},
			},
		}
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		spec = draftSpec{
			manifest: "asis-draft-manifest.json",
			content:  "# Stub As-Is Overview\n",
			outputs: []map[string]any{
				{
					"path":           "overview.md",
					"canonical_path": "reports/as-is/overview.md",
					"kind":           "report",
					"title":          "Stub As-Is Overview",
				},
			},
		}
	case "init.step4.proposals", "refresh.step4.proposals":
		spec = draftSpec{
			manifest: "proposals-draft-manifest.json",
			content:  "# Stub Proposal\n",
			outputs: []map[string]any{
				{
					"path":           "proposal.md",
					"canonical_path": "proposals/proposal-baseline/proposal.md",
					"kind":           "proposal",
					"title":          "Stub Proposal",
				},
			},
		}
	default:
		return nil
	}

	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		return err
	}
	for _, output := range spec.outputs {
		pathValue, _ := output["path"].(string)
		if err := os.WriteFile(filepath.Join(draftRoot, pathValue), []byte(spec.content), 0o644); err != nil {
			return err
		}
	}
	manifest := map[string]any{
		"version":       1,
		"run_id":        task.RunID,
		"step_id":       task.StepID,
		"step_contract": task.StepContract,
		"agent_role":    task.AgentRole,
		"summary":       "stub runtime draft",
		"outputs":       spec.outputs,
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(writeRoot, spec.manifest), raw, 0o644)
}
