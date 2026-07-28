package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestArtifactReadRejectsOversizedViewerContent(t *testing.T) {
	server := newTestServer(t)
	if err := server.getWorkspace().WriteFile("reports/oversized.md", []byte(strings.Repeat("x", 2*1024*1024+1))); err != nil {
		t.Fatalf("write oversized artifact: %v", err)
	}
	recorder := httptest.NewRecorder()
	server.handleArtifacts(recorder, httptest.NewRequest(http.MethodGet, "/api/artifacts?path=reports/oversized.md", nil))
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusRequestEntityTooLarge, recorder.Body.String())
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if payload.Error.Code != "artifact_too_large" {
		t.Fatalf("error code = %q, want artifact_too_large", payload.Error.Code)
	}
}
