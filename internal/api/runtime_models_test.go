package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRuntimeModelsGetAndPut(t *testing.T) {
	for _, key := range []string{"ACP_CLAUDE_MODEL", "ACP_CLAUDE_EFFORT", "ACP_QWEN_MODEL", "ACP_CODEX_MODEL", "ACP_CODEX_REASONING_EFFORT"} {
		t.Setenv(key, "")
	}
	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/runtime/models")
	if err != nil {
		t.Fatalf("GET /api/runtime/models: %v", err)
	}
	var initial struct {
		OK        bool `json:"ok"`
		Providers map[string]struct {
			Persisted map[string]string `json:"persisted"`
			Effective map[string]string `json:"effective"`
			Source    map[string]string `json:"source"`
		} `json:"providers"`
	}
	if err := json.NewDecoder(response.Body).Decode(&initial); err != nil {
		response.Body.Close()
		t.Fatalf("decode initial model payload: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !initial.OK {
		t.Fatalf("expected successful model payload, status=%d ok=%v", response.StatusCode, initial.OK)
	}
	if initial.Providers["codex-code"].Source["model"] != "provider_default" {
		t.Fatalf("expected provider_default source, got %#v", initial.Providers["codex-code"])
	}

	request, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/runtime/models", strings.NewReader(`{"providers":{"codex-code":{"model":"gpt-5.6-luna","effort":"high"}}}`))
	if err != nil {
		t.Fatalf("create PUT /api/runtime/models: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	putResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("PUT /api/runtime/models: %v", err)
	}
	defer putResponse.Body.Close()
	if putResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected PUT status 200, got %d", putResponse.StatusCode)
	}

	verify, err := http.Get(httpServer.URL + "/api/runtime/models")
	if err != nil {
		t.Fatalf("GET updated /api/runtime/models: %v", err)
	}
	defer verify.Body.Close()
	var updated struct {
		Providers map[string]struct {
			Persisted map[string]string `json:"persisted"`
			Effective map[string]string `json:"effective"`
			Source    map[string]string `json:"source"`
		} `json:"providers"`
	}
	if err := json.NewDecoder(verify.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated model payload: %v", err)
	}
	codex := updated.Providers["codex-code"]
	if codex.Persisted["model"] != "gpt-5.6-luna" || codex.Persisted["effort"] != "high" {
		t.Fatalf("unexpected persisted codex model: %#v", codex)
	}
	if codex.Effective["model"] != "gpt-5.6-luna" || codex.Source["effort"] != "workspace" {
		t.Fatalf("unexpected effective/source codex model: %#v", codex)
	}
}
