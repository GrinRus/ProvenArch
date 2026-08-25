package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryEvidenceReadsConfiguredSourceWithoutUsingWorkspaceArtifacts(t *testing.T) {
	server := newTestServer(t)
	repo := server.getWorkspace().Manifest.Repos[0]
	if err := os.MkdirAll(filepath.Join(repo.Path, "src"), 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo.Path, "src", "main.go"), []byte("package payments\n"), 0o644); err != nil {
		t.Fatalf("write source evidence: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/repository-evidence?repo=payments-service&path=src%2Fmain.go", nil)
	server.handleRepositoryEvidence(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response repositoryEvidenceResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Repo != "payments-service" || response.Path != "src/main.go" || response.Content != "package payments\n" {
		t.Fatalf("unexpected repository evidence response: %#v", response)
	}
}

func TestRepositoryEvidenceRejectsMissingOrEscapingPaths(t *testing.T) {
	server := newTestServer(t)
	for _, path := range []string{"", "../workspace.yaml", "/tmp/outside"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/repository-evidence?repo=payments-service&path="+path, nil)
		server.handleRepositoryEvidence(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("path %q status = %d, want %d", path, recorder.Code, http.StatusBadRequest)
		}
	}
}
