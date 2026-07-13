package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/runtime/fakeruntime"
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

func TestServeContextCancellationShutsDownService(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, cancellableDelayedRunner{delay: 5 * time.Second})
	service := server.getService()
	ws := server.getWorkspace()
	runID, err := service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace:      ws,
		Pipeline:       orchestrator.PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start async run: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	address := freeTCPAddress(t)
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ctx, address)
	}()
	waitForHTTPHealth(t, "http://"+address+"/api/health", 2*time.Second)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve returned error after context cancellation: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("serve did not stop after context cancellation")
	}

	info, ok := service.GetRun(runID)
	if !ok {
		t.Fatalf("expected run %q", runID)
	}
	if info.Status != orchestrator.RunStatusFailed || info.ErrorCode != "run_canceled" {
		t.Fatalf("expected service shutdown to cancel active run, got status=%s code=%q", info.Status, info.ErrorCode)
	}
}

func freeTCPAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on free TCP address: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close free TCP listener: %v", err)
	}
	return address
}

func waitForHTTPHealth(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		response, err := http.Get(url)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not become healthy at %s before timeout", url)
}

func TestSystemInfoEndpointReturnsBuildMetadataBeforeWorkspaceSelection(t *testing.T) {
	t.Parallel()

	config := testServerRuntimeConfig()
	config.Build = BuildInfo{Version: "0.1.2", Commit: "abc123", Built: "2026-06-02T13:20:26Z"}
	server := NewLauncherServer(config, testLauncherServiceFactory())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/system/info")
	if err != nil {
		t.Fatalf("GET /api/system/info: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	var payload struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Built   string `json:"built"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode system info: %v", err)
	}
	if payload.Version != "0.1.2" || payload.Commit != "abc123" || payload.Built != "2026-06-02T13:20:26Z" {
		t.Fatalf("unexpected build metadata: %+v", payload)
	}
}

func TestLauncherBlocksWorkspaceAPIsUntilWorkspaceSelected(t *testing.T) {
	t.Parallel()

	server := NewLauncherServer(testServerRuntimeConfig(), testLauncherServiceFactory())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(httpServer.URL+"/api/workspace/validate", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST /api/workspace/validate: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusPreconditionRequired {
		t.Fatalf("expected 428 before workspace selection, got %d", response.StatusCode)
	}

	statusResponse, err := http.Get(httpServer.URL + "/api/onboarding/status")
	if err != nil {
		t.Fatalf("GET /api/onboarding/status: %v", err)
	}
	defer statusResponse.Body.Close()
	if statusResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected onboarding status 200, got %d", statusResponse.StatusCode)
	}
	var status map[string]any
	if err := json.NewDecoder(statusResponse.Body).Decode(&status); err != nil {
		t.Fatalf("decode onboarding status: %v", err)
	}
	if status["workspace_selected"].(bool) {
		t.Fatalf("expected no workspace selected in launcher status: %#v", status)
	}

	infoResponse, err := http.Get(httpServer.URL + "/api/system/info")
	if err != nil {
		t.Fatalf("GET /api/system/info: %v", err)
	}
	defer infoResponse.Body.Close()
	if infoResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected system info 200 before workspace selection, got %d", infoResponse.StatusCode)
	}
}

func TestSystemVersionEndpointIsAvailableBeforeWorkspaceSelection(t *testing.T) {
	t.Parallel()

	server := NewLauncherServer(testServerRuntimeConfig(), testLauncherServiceFactory())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/system/version")
	if err != nil {
		t.Fatalf("GET /api/system/version: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status 200 before workspace selection, got %d body=%s", response.StatusCode, string(body))
	}
	var payload struct {
		Version  string `json:"version"`
		Commit   string `json:"commit"`
		Built    string `json:"built"`
		UIBundle string `json:"ui_bundle"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode version payload: %v", err)
	}
	if payload.Version == "" || payload.Commit == "" || payload.Built == "" || payload.UIBundle != "embedded" {
		t.Fatalf("unexpected version payload: %+v", payload)
	}
}

func TestDirectModeOnboardingStatusReflectsRuntimeConfig(t *testing.T) {
	t.Parallel()

	wsServer := newTestServer(t)
	ws := wsServer.getWorkspace()
	server := NewServerWithRuntime(ws, orchestrator.NewService(orchestrator.WithHistoryWorkspace(ws)), ServerRuntimeConfig{
		Mode:           acpruntime.RuntimeModeHeadless,
		Provider:       acpruntime.ProviderQwenCode,
		ProviderSource: acpruntime.ProviderSourceOverride,
	})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/onboarding/status")
	if err != nil {
		t.Fatalf("GET /api/onboarding/status: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected onboarding status 200, got %d", response.StatusCode)
	}
	var status struct {
		CanEnterConsole bool `json:"can_enter_console"`
		Runtime         struct {
			Selected        bool   `json:"selected"`
			Runtime         string `json:"runtime"`
			RuntimeProvider string `json:"runtime_provider"`
			ProviderSource  string `json:"provider_source"`
		} `json:"runtime"`
	}
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatalf("decode onboarding status: %v", err)
	}
	if !status.CanEnterConsole || !status.Runtime.Selected {
		t.Fatalf("expected direct mode ready runtime status, got %+v", status)
	}
	if status.Runtime.Runtime != acpruntime.RuntimeModeHeadless {
		t.Fatalf("expected headless runtime, got %q", status.Runtime.Runtime)
	}
	if status.Runtime.RuntimeProvider != string(acpruntime.ProviderQwenCode) {
		t.Fatalf("expected qwen provider, got %q", status.Runtime.RuntimeProvider)
	}
	if status.Runtime.ProviderSource != string(acpruntime.ProviderSourceOverride) {
		t.Fatalf("expected override provider source, got %q", status.Runtime.ProviderSource)
	}
}

func TestLauncherWorkspaceManifestAndRuntimeSelection(t *testing.T) {
	t.Parallel()

	server := NewLauncherServer(testServerRuntimeConfig(), testLauncherServiceFactory())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	root := t.TempDir()
	workspacePath := filepath.Join(root, "arch-workspace")
	repoPath := filepath.Join(root, "repo")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}

	workspaceResponse := postJSON(t, httpServer.URL+"/api/onboarding/workspace", fmt.Sprintf(`{"path":%q,"create":true}`, workspacePath))
	defer workspaceResponse.Body.Close()
	if workspaceResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected workspace selection 200, got %d", workspaceResponse.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, ".git")); err != nil {
		t.Fatalf("expected draft workspace git init: %v", err)
	}

	manifest := fmt.Sprintf("version: 1\nrepos:\n  - name: sample\n    path: %q\ndocs:\n  imports_path: %q\n", repoPath, "./docs/imports")
	manifestResponse := postJSONWithMethod(t, http.MethodPut, httpServer.URL+"/api/workspace/manifest", fmt.Sprintf(`{"content":%q}`, manifest))
	defer manifestResponse.Body.Close()
	if manifestResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected manifest save 200, got %d", manifestResponse.StatusCode)
	}

	runtimeResponse := postJSON(t, httpServer.URL+"/api/onboarding/runtime", `{"runtime":"fake","runtime_provider":"claude-code"}`)
	if runtimeResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected runtime selection 200, got %d", runtimeResponse.StatusCode)
	}
	defer runtimeResponse.Body.Close()
	var status map[string]any
	if err := json.NewDecoder(runtimeResponse.Body).Decode(&status); err != nil {
		t.Fatalf("decode runtime status: %v", err)
	}
	if !status["can_enter_console"].(bool) {
		t.Fatalf("expected launcher ready after workspace manifest and runtime selection: %#v", status)
	}

	validateResponse := postJSON(t, httpServer.URL+"/api/workspace/validate", "{}")
	defer validateResponse.Body.Close()
	if validateResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected workspace validate 200 after manifest save, got %d", validateResponse.StatusCode)
	}
}

func TestOnboardingWorkspaceSwitchConflictsWithActiveRun(t *testing.T) {
	t.Parallel()

	server := NewLauncherServer(testServerRuntimeConfig(), func(ws workspace.Root, config ServerRuntimeConfig) *orchestrator.Service {
		return orchestrator.NewService(
			orchestrator.WithHistoryWorkspace(ws),
			orchestrator.WithRunner(cancellableDelayedRunner{delay: 5 * time.Second}),
		)
	})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})

	root := t.TempDir()
	workspacePath := filepath.Join(root, "arch-workspace")
	repoPath := filepath.Join(root, "repo")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}

	workspaceResponse := postJSON(t, httpServer.URL+"/api/onboarding/workspace", fmt.Sprintf(`{"path":%q,"create":true}`, workspacePath))
	defer workspaceResponse.Body.Close()
	if workspaceResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected workspace selection 200, got %d", workspaceResponse.StatusCode)
	}
	manifest := fmt.Sprintf("version: 1\nrepos:\n  - name: sample\n    path: %q\n", repoPath)
	manifestResponse := postJSONWithMethod(t, http.MethodPut, httpServer.URL+"/api/workspace/manifest", fmt.Sprintf(`{"content":%q}`, manifest))
	defer manifestResponse.Body.Close()
	if manifestResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected manifest save 200, got %d", manifestResponse.StatusCode)
	}

	service := server.getService()
	runID, err := service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace:      server.getWorkspace(),
		Pipeline:       orchestrator.PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start active run: %v", err)
	}
	if !service.HasInFlightRun() {
		t.Fatalf("expected active run to be visible before workspace switch")
	}

	secondWorkspacePath := filepath.Join(root, "second-arch-workspace")
	switchResponse := postJSON(t, httpServer.URL+"/api/onboarding/workspace", fmt.Sprintf(`{"path":%q,"create":true}`, secondWorkspacePath))
	defer switchResponse.Body.Close()
	if switchResponse.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(switchResponse.Body)
		t.Fatalf("expected workspace switch conflict 409, got %d body=%s", switchResponse.StatusCode, string(body))
	}
	if code := decodeErrorCode(t, switchResponse); code != "workspace_switch_conflict" {
		t.Fatalf("expected workspace_switch_conflict, got %q", code)
	}
	if server.getWorkspace().Path != workspacePath {
		t.Fatalf("expected original workspace to remain selected, got %q", server.getWorkspace().Path)
	}
	if _, ok := server.getService().GetRun(runID); !ok {
		t.Fatalf("expected active run %q to remain on original service", runID)
	}
}

func TestOnboardingRuntimeSwitchConflictsWithActiveRun(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, cancellableDelayedRunner{delay: 5 * time.Second})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})

	service := server.getService()
	if _, err := service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace:      server.getWorkspace(),
		Pipeline:       orchestrator.PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("start active run: %v", err)
	}

	response := postJSON(t, httpServer.URL+"/api/onboarding/runtime", `{"runtime":"headless","runtime_provider":"qwen-code"}`)
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected runtime switch conflict 409, got %d body=%s", response.StatusCode, string(body))
	}
	if code := decodeErrorCode(t, response); code != "runtime_switch_conflict" {
		t.Fatalf("expected runtime_switch_conflict, got %q", code)
	}

	statusResponse, err := http.Get(httpServer.URL + "/api/onboarding/status")
	if err != nil {
		t.Fatalf("GET /api/onboarding/status: %v", err)
	}
	defer statusResponse.Body.Close()
	var status struct {
		Runtime struct {
			Runtime         string `json:"runtime"`
			RuntimeProvider string `json:"runtime_provider"`
		} `json:"runtime"`
	}
	if err := json.NewDecoder(statusResponse.Body).Decode(&status); err != nil {
		t.Fatalf("decode onboarding status: %v", err)
	}
	if status.Runtime.Runtime != acpruntime.RuntimeModeFake || status.Runtime.RuntimeProvider != string(acpruntime.ProviderClaudeCode) {
		t.Fatalf("expected effective runtime to remain fake/claude-code, got %+v", status.Runtime)
	}
}

func TestRuntimeProfileMutationConflictsWithActiveRun(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, cancellableDelayedRunner{delay: 5 * time.Second})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})

	beforeManifest, err := os.ReadFile(filepath.Join(server.getWorkspace().Path, workspace.ManifestFileName))
	if err != nil {
		t.Fatalf("read manifest before mutation: %v", err)
	}
	if _, err := server.getService().StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace:      server.getWorkspace(),
		Pipeline:       orchestrator.PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("start active run: %v", err)
	}

	response := postJSONWithMethod(t, http.MethodPut, httpServer.URL+"/api/runtime/execution", `{"execution":{"strategy":"parallel"}}`)
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected runtime profile conflict 409, got %d body=%s", response.StatusCode, string(body))
	}
	if code := decodeErrorCode(t, response); code != "runtime_profile_conflict" {
		t.Fatalf("expected runtime_profile_conflict, got %q", code)
	}
	afterManifest, err := os.ReadFile(filepath.Join(server.getWorkspace().Path, workspace.ManifestFileName))
	if err != nil {
		t.Fatalf("read manifest after mutation: %v", err)
	}
	if string(afterManifest) != string(beforeManifest) {
		t.Fatalf("expected runtime profile conflict not to mutate manifest\nbefore:\n%s\nafter:\n%s", string(beforeManifest), string(afterManifest))
	}
}

func TestConcurrentPollingAndSessionMutationConflictIsRaceSafe(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, cancellableDelayedRunner{delay: 5 * time.Second})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})

	if _, err := server.getService().StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace:      server.getWorkspace(),
		Pipeline:       orchestrator.PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("start active run: %v", err)
	}

	errCh := make(chan error, 32)
	for i := 0; i < 8; i++ {
		go func() {
			resp, err := http.Get(httpServer.URL + "/api/pipeline/runs")
			if err != nil {
				errCh <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errCh <- fmt.Errorf("GET runs status=%d", resp.StatusCode)
				return
			}
			errCh <- nil
		}()
		go func() {
			resp, err := http.Get(httpServer.URL + "/api/runtime/profile")
			if err != nil {
				errCh <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errCh <- fmt.Errorf("GET runtime profile status=%d", resp.StatusCode)
				return
			}
			errCh <- nil
		}()
		go func() {
			resp, err := http.Post(httpServer.URL+"/api/onboarding/runtime", "application/json", strings.NewReader(`{"runtime":"headless","runtime_provider":"qwen-code"}`))
			if err != nil {
				errCh <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusConflict {
				errCh <- fmt.Errorf("POST runtime status=%d", resp.StatusCode)
				return
			}
			errCh <- nil
		}()
	}
	for i := 0; i < 24; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}
}

func TestNormalizeOnboardingWorkspacePath(t *testing.T) {
	t.Parallel()

	validTemp := filepath.Join(os.TempDir(), "acp-onboarding-test")
	normalized, err := normalizeOnboardingWorkspacePath(validTemp)
	if err != nil {
		t.Fatalf("expected temp workspace path to be accepted: %+v", err)
	}
	if normalized != filepath.Clean(validTemp) {
		t.Fatalf("expected cleaned temp path %q, got %q", filepath.Clean(validTemp), normalized)
	}

	if filepath.IsAbs("/tmp") {
		tmpAlias := filepath.Join("/tmp", "acp-onboarding-test")
		normalized, err := normalizeOnboardingWorkspacePath(tmpAlias)
		if err != nil {
			t.Fatalf("expected /tmp workspace path to be accepted: %+v", err)
		}
		if normalized != filepath.Clean(tmpAlias) {
			t.Fatalf("expected cleaned /tmp path %q, got %q", filepath.Clean(tmpAlias), normalized)
		}
	}

	if home, homeErr := os.UserHomeDir(); homeErr == nil && strings.TrimSpace(home) != "" {
		validHome := filepath.Join(home, "acp-workspaces", "sample")
		normalized, err := normalizeOnboardingWorkspacePath(validHome)
		if err != nil {
			t.Fatalf("expected home workspace path to be accepted: %+v", err)
		}
		if normalized != filepath.Clean(validHome) {
			t.Fatalf("expected cleaned home path %q, got %q", filepath.Clean(validHome), normalized)
		}
	}

	cases := []struct {
		name string
		path string
		code string
	}{
		{name: "empty", path: " ", code: "workspace_path_required"},
		{name: "relative", path: "workspace", code: "workspace_path_not_absolute"},
		{name: "traversal", path: filepath.Join(os.TempDir(), "..", "acp"), code: "workspace_path_traversal"},
		{name: "root", path: string(filepath.Separator), code: "workspace_path_invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := normalizeOnboardingWorkspacePath(tc.path)
			if err == nil {
				t.Fatalf("expected %s to be rejected", tc.path)
			}
			if err.code != tc.code {
				t.Fatalf("expected code %q, got %q", tc.code, err.code)
			}
		})
	}
}

func TestOnboardingWorkspacePathSuggestionsBeforeWorkspaceSelection(t *testing.T) {
	t.Parallel()

	server := NewLauncherServer(testServerRuntimeConfig(), testLauncherServiceFactory())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	query := filepath.Join(os.TempDir(), "acp-suggestion-workspace")
	response, err := http.Get(httpServer.URL + "/api/onboarding/path-suggestions?kind=workspace&query=" + url.QueryEscape(query))
	if err != nil {
		t.Fatalf("GET workspace suggestions: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected workspace suggestions 200, got %d", response.StatusCode)
	}

	var payload struct {
		OK    bool                       `json:"ok"`
		Kind  string                     `json:"kind"`
		Items []onboardingPathSuggestion `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode workspace suggestions: %v", err)
	}
	if !payload.OK || payload.Kind != "workspace" {
		t.Fatalf("unexpected workspace suggestions payload: %+v", payload)
	}
	if !containsPathSuggestion(payload.Items, filepath.Clean(query), "workspace") {
		t.Fatalf("expected query workspace suggestion %q in %s", query, pathSuggestionDebugString(payload.Items))
	}
}

func TestOnboardingRepoPathSuggestionsClassifyGitDirectories(t *testing.T) {
	t.Parallel()

	repoPath := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0o755); err != nil {
		t.Fatalf("create repo-like directory: %v", err)
	}
	server := NewLauncherServer(testServerRuntimeConfig(), testLauncherServiceFactory())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/onboarding/path-suggestions?kind=repo&query=" + url.QueryEscape(repoPath))
	if err != nil {
		t.Fatalf("GET repo suggestions: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected repo suggestions 200, got %d", response.StatusCode)
	}
	var payload struct {
		Items []onboardingPathSuggestion `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode repo suggestions: %v", err)
	}
	if !containsPathSuggestion(payload.Items, filepath.Clean(repoPath), "git_repo") {
		t.Fatalf("expected git_repo suggestion %q in %s", repoPath, pathSuggestionDebugString(payload.Items))
	}
}

func TestOnboardingPathSuggestionsRejectUnsafeInputs(t *testing.T) {
	t.Parallel()

	server := NewLauncherServer(testServerRuntimeConfig(), testLauncherServiceFactory())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	cases := []string{
		"/api/onboarding/path-suggestions?kind=bad&query=" + url.QueryEscape(filepath.Join(os.TempDir(), "acp")),
		"/api/onboarding/path-suggestions?kind=repo&query=" + url.QueryEscape(filepath.Clean(os.TempDir())+string(filepath.Separator)+".."+string(filepath.Separator)+"escape"),
		"/api/onboarding/path-suggestions?kind=repo&query=" + url.QueryEscape(string(filepath.Separator)),
	}
	for _, pathValue := range cases {
		response, err := http.Get(httpServer.URL + pathValue)
		if err != nil {
			t.Fatalf("GET unsafe suggestion case %q: %v", pathValue, err)
		}
		if response.StatusCode != http.StatusBadRequest {
			response.Body.Close()
			t.Fatalf("expected 400 for %q, got %d", pathValue, response.StatusCode)
		}
		response.Body.Close()
	}
}

func TestOnboardingRepoPathSuggestionsRejectSymlinkEscape(t *testing.T) {
	t.Parallel()

	linkPath := filepath.Join(os.TempDir(), "acp-symlink-escape-"+strings.NewReplacer("/", "-", " ", "-").Replace(t.Name()))
	_ = os.Remove(linkPath)
	if err := os.Symlink(string(filepath.Separator), linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(linkPath)
	})
	if _, ok := normalizeRepoSuggestionPath(linkPath); ok {
		t.Fatalf("expected symlink escape %q to be rejected", linkPath)
	}
}

func containsPathSuggestion(items []onboardingPathSuggestion, pathValue string, kind string) bool {
	for _, item := range items {
		if filepath.Clean(item.Path) == filepath.Clean(pathValue) && item.Kind == kind {
			return true
		}
	}
	return false
}

func pathSuggestionDebugString(items []onboardingPathSuggestion) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s:%s:%s", item.Source, item.Kind, item.Path))
	}
	return strings.Join(parts, ", ")
}

func TestOnboardingRecentWorkspacesRecordListAndForget(t *testing.T) {
	recentsPath := filepath.Join(t.TempDir(), "recent-workspaces.json")
	withOnboardingRecentWorkspacesPath(t, recentsPath)

	server := NewLauncherServer(testServerRuntimeConfig(), testLauncherServiceFactory())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	root := t.TempDir()
	firstWorkspace := filepath.Join(root, "first-workspace")
	secondWorkspace := filepath.Join(root, "second-workspace")

	firstResponse := postJSON(t, httpServer.URL+"/api/onboarding/workspace", fmt.Sprintf(`{"path":%q,"create":true}`, firstWorkspace))
	defer firstResponse.Body.Close()
	if firstResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected first workspace selection 200, got %d", firstResponse.StatusCode)
	}
	secondResponse := postJSON(t, httpServer.URL+"/api/onboarding/workspace", fmt.Sprintf(`{"path":%q,"create":true}`, secondWorkspace))
	defer secondResponse.Body.Close()
	if secondResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected second workspace selection 200, got %d", secondResponse.StatusCode)
	}
	if err := os.RemoveAll(firstWorkspace); err != nil {
		t.Fatalf("remove first workspace: %v", err)
	}

	statusResponse, err := http.Get(httpServer.URL + "/api/onboarding/status")
	if err != nil {
		t.Fatalf("GET onboarding status: %v", err)
	}
	defer statusResponse.Body.Close()
	var status struct {
		RecentWorkspaces []onboardingRecentWorkspace `json:"recent_workspaces"`
	}
	if err := json.NewDecoder(statusResponse.Body).Decode(&status); err != nil {
		t.Fatalf("decode onboarding status: %v", err)
	}
	if len(status.RecentWorkspaces) != 2 {
		t.Fatalf("expected 2 recent workspaces, got %#v", status.RecentWorkspaces)
	}
	if status.RecentWorkspaces[0].Path != secondWorkspace || !status.RecentWorkspaces[0].Exists {
		t.Fatalf("expected newest existing second workspace first, got %#v", status.RecentWorkspaces[0])
	}
	if status.RecentWorkspaces[1].Path != firstWorkspace || status.RecentWorkspaces[1].Exists {
		t.Fatalf("expected missing first workspace second, got %#v", status.RecentWorkspaces[1])
	}

	forgetResponse := postJSON(t, httpServer.URL+"/api/onboarding/recent-workspaces/forget", fmt.Sprintf(`{"path":%q}`, firstWorkspace))
	defer forgetResponse.Body.Close()
	if forgetResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected forget response 200, got %d", forgetResponse.StatusCode)
	}
	var forgetStatus struct {
		RecentWorkspaces []onboardingRecentWorkspace `json:"recent_workspaces"`
	}
	if err := json.NewDecoder(forgetResponse.Body).Decode(&forgetStatus); err != nil {
		t.Fatalf("decode forget status: %v", err)
	}
	if len(forgetStatus.RecentWorkspaces) != 1 || forgetStatus.RecentWorkspaces[0].Path != secondWorkspace {
		t.Fatalf("expected only second workspace after forget, got %#v", forgetStatus.RecentWorkspaces)
	}
}

func TestOnboardingRecentWorkspacesLimitNewestFirst(t *testing.T) {
	recentsPath := filepath.Join(t.TempDir(), "recent-workspaces.json")
	withOnboardingRecentWorkspacesPath(t, recentsPath)

	root := t.TempDir()
	baseTime := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	for index := 0; index < 12; index++ {
		workspacePath := filepath.Join(root, fmt.Sprintf("workspace-%02d", index))
		if err := os.MkdirAll(workspacePath, 0o755); err != nil {
			t.Fatalf("create workspace %d: %v", index, err)
		}
		if err := recordOnboardingRecentWorkspace(workspacePath, baseTime.Add(time.Duration(index)*time.Minute)); err != nil {
			t.Fatalf("record workspace %d: %v", index, err)
		}
	}

	recents := loadOnboardingRecentWorkspaces()
	if len(recents) != onboardingRecentWorkspaceLimit {
		t.Fatalf("expected limit %d, got %d", onboardingRecentWorkspaceLimit, len(recents))
	}
	if want := filepath.Join(root, "workspace-11"); recents[0].Path != want {
		t.Fatalf("expected newest workspace %q first, got %q", want, recents[0].Path)
	}
	if want := filepath.Join(root, "workspace-02"); recents[len(recents)-1].Path != want {
		t.Fatalf("expected oldest retained workspace %q last, got %q", want, recents[len(recents)-1].Path)
	}
}

func TestSystemDoctorEndpointReturnsReadinessChecks(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/system/doctor?runtime=fake&runtime_provider=claude-code")
	if err != nil {
		t.Fatalf("GET /api/system/doctor: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status 200, got %d body=%s", response.StatusCode, string(body))
	}
	var payload struct {
		OK     bool `json:"ok"`
		Checks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode doctor payload: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected doctor ok=true, got %+v", payload.Checks)
	}
	if !hasDoctorCheck(payload.Checks, "embedded_ui", "pass") {
		t.Fatalf("expected embedded_ui pass, got %+v", payload.Checks)
	}
	if !hasDoctorCheck(payload.Checks, "runtime_provider", "pass") {
		t.Fatalf("expected runtime_provider pass, got %+v", payload.Checks)
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

func TestEmbeddedUIServesUnderscoreAssets(t *testing.T) {
	t.Parallel()

	assetName := findSourceUnderscoreJSAsset(t)
	expected, err := os.ReadFile(filepath.Join("ui_dist", "assets", assetName))
	if err != nil {
		t.Fatalf("read source asset: %v", err)
	}

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/assets/" + assetName)
	if err != nil {
		t.Fatalf("GET underscore asset: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	contentType := response.Header.Get("Content-Type")
	if !strings.Contains(contentType, "javascript") {
		t.Fatalf("expected javascript content type, got %q", contentType)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if strings.Contains(strings.ToLower(string(content)), "<!doctype html") {
		t.Fatalf("expected asset body, got html shell")
	}
	if !bytes.Equal(content, expected) {
		t.Fatalf("served asset does not match source asset")
	}
}

func findSourceUnderscoreJSAsset(t *testing.T) string {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join("ui_dist", "assets"))
	if err != nil {
		t.Fatalf("read embedded ui asset directory: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasPrefix(name, "_") && strings.HasSuffix(name, ".js") {
			return name
		}
	}
	t.Skip("embedded UI has no underscore-prefixed JavaScript asset")
	return ""
}

func hasDoctorCheck(checks []struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}, id string, status string) bool {
	for _, check := range checks {
		if check.ID == id && check.Status == status {
			return true
		}
	}
	return false
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

func TestQAAskEndpointReturnsWorkspaceBackedResponseAndDoesNotMutateWorkspace(t *testing.T) {
	t.Parallel()

	server := newQATestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	before := snapshotWorkspaceFiles(t, server.getWorkspace().Path)
	response, err := http.Post(
		httpServer.URL+"/api/qa/ask",
		"application/json",
		bytes.NewBufferString(`{"question":"What does coverage say about owner mappings and architecture notes?"}`),
	)
	if err != nil {
		t.Fatalf("POST /api/qa/ask: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status 200, got %d body=%s", response.StatusCode, string(body))
	}
	var payload struct {
		Answer     string          `json:"answer"`
		Citations  []qaAPICitation `json:"citations"`
		Unresolved []string        `json:"unresolved"`
		Confidence float64         `json:"confidence"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode qa response: %v", err)
	}
	if strings.TrimSpace(payload.Answer) == "" {
		t.Fatalf("expected answer")
	}
	if payload.Confidence <= 0 {
		t.Fatalf("expected positive confidence, got %f", payload.Confidence)
	}
	if len(payload.Citations) == 0 {
		t.Fatalf("expected citations")
	}
	if !hasAPICitationPathPrefix(payload.Citations, "docs/imports/") {
		t.Fatalf("expected docs/imports citation, got %+v", payload.Citations)
	}

	after := snapshotWorkspaceFiles(t, server.getWorkspace().Path)
	assertWorkspaceSnapshotEqual(t, before, after)
}

func TestQAAskEndpointRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		body       string
		statusCode int
		errorCode  string
	}{
		{
			name:       "empty question",
			body:       `{"question":"   "}`,
			statusCode: http.StatusBadRequest,
			errorCode:  "question_required",
		},
		{
			name:       "unknown field",
			body:       `{"question":"owners","extra":true}`,
			statusCode: http.StatusBadRequest,
			errorCode:  "invalid_request_body",
		},
		{
			name:       "malformed json",
			body:       `{"question":`,
			statusCode: http.StatusBadRequest,
			errorCode:  "invalid_request_body",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := newQATestServer(t)
			httpServer := httptest.NewServer(server.Handler())
			defer httpServer.Close()

			response, err := http.Post(httpServer.URL+"/api/qa/ask", "application/json", bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatalf("POST /api/qa/ask: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != tc.statusCode {
				body, _ := io.ReadAll(response.Body)
				t.Fatalf("expected status %d, got %d body=%s", tc.statusCode, response.StatusCode, string(body))
			}
			var payload struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatalf("decode error payload: %v", err)
			}
			if payload.Error.Code != tc.errorCode {
				t.Fatalf("expected error code %q, got %q", tc.errorCode, payload.Error.Code)
			}
		})
	}
}

func TestQARunsEndpointStartsFakeRuntimeRunAndWritesAuditArtifacts(t *testing.T) {
	t.Parallel()

	server := newQATestServer(t)
	ws := server.getWorkspace()
	if err := ws.WriteFile("reports/taskruns/old-run/qa/qa-answer.json", []byte(`{"secret":"old taskrun should not enter context"}`)); err != nil {
		t.Fatalf("write old taskrun fixture: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/api/qa/runs",
		"application/json",
		bytes.NewBufferString(`{"question":"What does coverage say about owner mappings and architecture notes?"}`),
	)
	if err != nil {
		t.Fatalf("POST /api/qa/runs: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status 202, got %d body=%s", response.StatusCode, string(body))
	}
	var started struct {
		RunID  string `json:"run_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&started); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}
	if strings.TrimSpace(started.RunID) == "" {
		t.Fatalf("expected qa run id")
	}
	if started.Status != string(orchestrator.RunStatusQueued) {
		t.Fatalf("expected queued status, got %q", started.Status)
	}

	var detail struct {
		RunID           string          `json:"run_id"`
		Pipeline        string          `json:"pipeline"`
		Status          string          `json:"status"`
		Question        string          `json:"question"`
		CurrentStep     string          `json:"current_step"`
		RuntimeProvider string          `json:"runtime_provider"`
		Provider        string          `json:"provider"`
		Answer          string          `json:"answer"`
		Citations       []qaAPICitation `json:"citations"`
		Confidence      float64         `json:"confidence"`
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(httpServer.URL + "/api/qa/runs/" + started.RunID)
		if err != nil {
			t.Fatalf("GET /api/qa/runs/%s: %v", started.RunID, err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			t.Fatalf("expected status 200, got %d body=%s", resp.StatusCode, string(body))
		}
		if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
			_ = resp.Body.Close()
			t.Fatalf("decode qa run detail: %v", err)
		}
		_ = resp.Body.Close()
		if detail.Status == string(orchestrator.RunStatusSucceeded) || detail.Status == string(orchestrator.RunStatusFailed) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if detail.Status != string(orchestrator.RunStatusSucceeded) {
		t.Fatalf("expected qa run succeeded, got %+v", detail)
	}
	if detail.Pipeline != string(orchestrator.PipelineQA) || detail.CurrentStep != acpruntime.StepIDQAAsk {
		t.Fatalf("expected qa pipeline/current_step, got pipeline=%q step=%q", detail.Pipeline, detail.CurrentStep)
	}
	if detail.Provider != "fake" {
		t.Fatalf("expected fake answer provider, got %q", detail.Provider)
	}
	if detail.RuntimeProvider != string(acpruntime.ProviderClaudeCode) {
		t.Fatalf("expected default qa runtime provider claude-code, got %q", detail.RuntimeProvider)
	}
	if !strings.Contains(detail.Answer, "Fake runtime QA inspected") {
		t.Fatalf("expected fake qa answer, got %q", detail.Answer)
	}
	if len(detail.Citations) == 0 || detail.Confidence <= 0 {
		t.Fatalf("expected citations/confidence, got citations=%+v confidence=%f", detail.Citations, detail.Confidence)
	}

	contextPack, err := ws.ReadFile(filepath.Join("reports", "taskruns", started.RunID, "qa", "context-pack.json"))
	if err != nil {
		t.Fatalf("read context pack: %v", err)
	}
	if strings.Contains(string(contextPack), "old taskrun should not enter context") {
		t.Fatalf("context pack included reports/taskruns evidence:\n%s", string(contextPack))
	}
	if _, err := ws.ReadFile(filepath.Join("reports", "taskruns", started.RunID, "qa", "runtime-execution.json")); err != nil {
		t.Fatalf("expected runtime execution artifact: %v", err)
	}

	listResp, err := http.Get(httpServer.URL + "/api/pipeline/runs?limit=100")
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs: %v", err)
	}
	defer listResp.Body.Close()
	var listPayload struct {
		Items []struct {
			RunID    string `json:"run_id"`
			Pipeline string `json:"pipeline"`
		} `json:"items"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listPayload); err != nil {
		t.Fatalf("decode pipeline run list: %v", err)
	}
	for _, item := range listPayload.Items {
		if item.RunID == started.RunID || item.Pipeline == string(orchestrator.PipelineQA) {
			t.Fatalf("qa run leaked into pipeline run list: %+v", listPayload.Items)
		}
	}
}

func TestQARunsEndpointRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	server := newQATestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(httpServer.URL+"/api/qa/runs", "application/json", bytes.NewBufferString(`{"question":"   "}`))
	if err != nil {
		t.Fatalf("POST /api/qa/runs: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status 400, got %d body=%s", response.StatusCode, string(body))
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Code != "question_required" {
		t.Fatalf("expected question_required, got %q", payload.Error.Code)
	}
}

func TestQARunsEndpointReportsFailedProviderAnswerWithoutParsingPartialArtifact(t *testing.T) {
	t.Parallel()

	server := newQATestServerWithService(t, func(ws workspace.Root) *orchestrator.Service {
		return orchestrator.NewService(
			orchestrator.WithHistoryWorkspace(ws),
			orchestrator.WithRunner(qaInvalidCitationRunner{}),
		)
	})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/api/qa/runs",
		"application/json",
		bytes.NewBufferString(`{"question":"Which overview exists?"}`),
	)
	if err != nil {
		t.Fatalf("POST /api/qa/runs: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status 202, got %d body=%s", response.StatusCode, string(body))
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&started); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}

	var detail struct {
		Status    string  `json:"status"`
		ErrorCode string  `json:"error_code"`
		Error     string  `json:"error"`
		Answer    *string `json:"answer"`
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(httpServer.URL + "/api/qa/runs/" + started.RunID)
		if err != nil {
			t.Fatalf("GET /api/qa/runs/%s: %v", started.RunID, err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			t.Fatalf("expected status 200 while polling failed QA run, got %d body=%s", resp.StatusCode, string(body))
		}
		if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
			_ = resp.Body.Close()
			t.Fatalf("decode qa run detail: %v", err)
		}
		_ = resp.Body.Close()
		if detail.Status == string(orchestrator.RunStatusSucceeded) || detail.Status == string(orchestrator.RunStatusFailed) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if detail.Status != string(orchestrator.RunStatusFailed) {
		t.Fatalf("expected failed QA run, got %+v", detail)
	}
	if detail.ErrorCode != string(acpruntime.ErrorCodeRuntimeContract) {
		t.Fatalf("expected runtime_contract_failed, got %+v", detail)
	}
	if !strings.Contains(detail.Error, "not present in context pack") {
		t.Fatalf("expected context-pack citation error, got %+v", detail)
	}
	if detail.Answer != nil {
		t.Fatalf("failed QA run should not expose partial answer, got %q", *detail.Answer)
	}
}

func TestWorkspaceBundleEndpointReturnsEffectiveManifestAndWarnings(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/workspace/bundle")
	if err != nil {
		t.Fatalf("GET /api/workspace/bundle: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	var payload struct {
		OK       bool `json:"ok"`
		Manifest struct {
			SchemaVersion     int `json:"schema_version"`
			BundleVersion     int `json:"bundle_version"`
			EditableArtifacts []struct {
				Path  string `json:"path"`
				Label string `json:"label"`
			} `json:"editable_artifacts"`
		} `json:"manifest"`
		Warnings []struct {
			Code string `json:"code"`
		} `json:"warnings"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode bundle payload: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected ok=true")
	}
	if payload.Manifest.SchemaVersion == 0 || payload.Manifest.BundleVersion == 0 {
		t.Fatalf("expected non-zero manifest versions, got %+v", payload.Manifest)
	}
	if len(payload.Manifest.EditableArtifacts) == 0 {
		t.Fatalf("expected editable artifacts in bundle manifest")
	}
	foundMissingWarning := false
	for _, warning := range payload.Warnings {
		if warning.Code == "workspace.skills.bundle_manifest.missing" {
			foundMissingWarning = true
			break
		}
	}
	if !foundMissingWarning {
		t.Fatalf("expected missing bundle manifest warning, got %+v", payload.Warnings)
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

func TestPipelineRunReviewSummaryEndpointMapsCanonicalSteps(t *testing.T) {
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
		t.Fatalf("decode start payload: %v", err)
	}
	if strings.TrimSpace(started.RunID) == "" {
		t.Fatalf("expected run id")
	}
	terminal := waitForRunTerminalStatus(t, httpServer.URL, started.RunID, 8*time.Second)
	if terminal.Status != string(orchestrator.RunStatusSucceeded) {
		t.Fatalf("expected succeeded run, got %+v", terminal)
	}

	summaryResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + started.RunID + "/review-summary")
	if err != nil {
		t.Fatalf("GET review-summary: %v", err)
	}
	defer summaryResp.Body.Close()
	if summaryResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(summaryResp.Body)
		t.Fatalf("expected status 200, got %d body=%s", summaryResp.StatusCode, string(body))
	}
	var payload struct {
		RunID       string `json:"run_id"`
		Pipeline    string `json:"pipeline"`
		Status      string `json:"status"`
		CurrentStep string `json:"current_step"`
		Steps       []struct {
			StepID        string   `json:"step_id"`
			Key           string   `json:"key"`
			State         string   `json:"state"`
			Provider      string   `json:"provider"`
			ArtifactCount int      `json:"artifact_count"`
			ArtifactPaths []string `json:"artifact_paths"`
			LastMessage   string   `json:"last_message"`
		} `json:"steps"`
	}
	if err := json.NewDecoder(summaryResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode review summary: %v", err)
	}
	if payload.RunID != started.RunID || payload.Pipeline != "init" || payload.Status != "succeeded" {
		t.Fatalf("unexpected summary identity: %+v", payload)
	}
	if len(payload.Steps) != 5 {
		t.Fatalf("expected five canonical steps, got %d", len(payload.Steps))
	}
	expected := []struct {
		stepID string
		key    string
	}{
		{"init.step0.constitution", acpruntime.StepProviderStep0Constitution},
		{"init.step1.collect", acpruntime.StepProviderStep1Collect},
		{"init.step2.asis_docs", acpruntime.StepProviderStep2AsIs},
		{"init.step3.findings", acpruntime.StepProviderStep3Findings},
		{"init.step4.proposals", acpruntime.StepProviderStep4Proposals},
	}
	for index, want := range expected {
		got := payload.Steps[index]
		if got.StepID != want.stepID || got.Key != want.key || got.State != "done" {
			t.Fatalf("unexpected step %d: got %+v want step=%q key=%q state=done", index, got, want.stepID, want.key)
		}
		if got.Provider != "fake" {
			t.Fatalf("expected fake provider for release fake run presentation, got step %d provider %q", index, got.Provider)
		}
	}
	if payload.Steps[2].ArtifactCount == 0 || !containsString(payload.Steps[2].ArtifactPaths, "reports/as-is/overview.md") {
		t.Fatalf("expected as-is step artifacts to include overview.md, got %+v", payload.Steps[2])
	}
}

func TestBuildRunReviewStepsUsesFakeProviderAndUniqueWarningCounts(t *testing.T) {
	t.Parallel()

	runInfo := orchestrator.RunInfo{
		RunID:         "run-1",
		Pipeline:      "init",
		Status:        orchestrator.RunStatusSucceeded,
		CurrentStep:   "init.step4.proposals",
		StepProviders: map[string]string{acpruntime.StepProviderStep1Collect: string(acpruntime.ProviderClaudeCode)},
		Warnings:      []string{"init.step1.collect: duplicate warning"},
	}
	logs := []orchestrator.RunLogEntry{
		{
			StepID:  "init.step1.collect",
			Level:   orchestrator.RunLogLevelWarning,
			Message: "runtime shard planner warning",
			Fields:  map[string]any{"warning": "duplicate warning"},
		},
		{StepID: "init.step1.collect", Level: orchestrator.RunLogLevelWarning, Message: "unique step warning"},
		{StepID: "init.step1.collect", Level: orchestrator.RunLogLevelError, Message: "step failed once"},
	}

	steps := buildRunReviewSteps(runInfo, nil, logs, acpruntime.RuntimeModeFake)
	collect := steps[1]
	if collect.Provider != "fake" {
		t.Fatalf("expected fake provider label, got %q", collect.Provider)
	}
	if collect.WarningsCount != 1 {
		t.Fatalf("expected only unique step warning count, got %d", collect.WarningsCount)
	}
	if collect.ErrorsCount != 1 {
		t.Fatalf("expected one step error, got %d", collect.ErrorsCount)
	}
}

func TestPipelineRunReviewSummaryEndpointHandlesFailedRun(t *testing.T) {
	t.Parallel()

	server := newTestServerWithRunner(t, parseFailureRunner{})
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
		t.Fatalf("decode start payload: %v", err)
	}
	terminal := waitForRunTerminalStatus(t, httpServer.URL, started.RunID, 8*time.Second)
	if terminal.Status != string(orchestrator.RunStatusFailed) {
		t.Fatalf("expected failed run, got %+v", terminal)
	}

	summaryResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + started.RunID + "/review-summary")
	if err != nil {
		t.Fatalf("GET review-summary failed run: %v", err)
	}
	defer summaryResp.Body.Close()
	if summaryResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(summaryResp.Body)
		t.Fatalf("expected status 200, got %d body=%s", summaryResp.StatusCode, string(body))
	}
	var payload struct {
		Status string `json:"status"`
		Steps  []struct {
			StepID      string `json:"step_id"`
			State       string `json:"state"`
			LastMessage string `json:"last_message"`
		} `json:"steps"`
	}
	if err := json.NewDecoder(summaryResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode failed review summary: %v", err)
	}
	if payload.Status != "failed" || len(payload.Steps) != 5 {
		t.Fatalf("unexpected failed summary: %+v", payload)
	}
	failedIndex := -1
	for index, step := range payload.Steps {
		if step.State == "failed" {
			failedIndex = index
			break
		}
	}
	if failedIndex < 0 {
		t.Fatalf("expected one failed step in summary, got %+v", payload.Steps)
	}
	if strings.TrimSpace(payload.Steps[failedIndex].LastMessage) == "" {
		t.Fatalf("expected failed step last_message to carry recovery context, got %+v", payload.Steps[failedIndex])
	}
}

func TestGitDiffEndpointReturnsWorkspaceFolderAndLineHunks(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for git diff API tests")
	}

	server := newTestServer(t)
	ws := initGitWorkspaceForDiffTest(t, server)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	if err := ws.WriteFile("reports/as-is/overview.md", []byte("old overview\n")); err != nil {
		t.Fatalf("write overview baseline: %v", err)
	}
	if err := ws.WriteFile("reports/coverage/summary.md", []byte("old coverage\n")); err != nil {
		t.Fatalf("write coverage baseline: %v", err)
	}
	commitWorkspaceForDiffTest(t, ws, "baseline")
	if err := ws.WriteFile("reports/as-is/overview.md", []byte("old overview\nnew evidence\n")); err != nil {
		t.Fatalf("modify overview: %v", err)
	}
	if err := os.Remove(filepath.Join(ws.Path, "reports", "coverage", "summary.md")); err != nil {
		t.Fatalf("delete coverage baseline: %v", err)
	}
	if err := ws.WriteFile("proposals/proposal-baseline/proposal.md", []byte("# Proposal\n")); err != nil {
		t.Fatalf("write proposal: %v", err)
	}

	diffResp, err := http.Get(httpServer.URL + "/api/git/diff?path=reports/as-is/overview.md")
	if err != nil {
		t.Fatalf("GET git diff selected file: %v", err)
	}
	defer diffResp.Body.Close()
	if diffResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(diffResp.Body)
		t.Fatalf("expected status 200, got %d body=%s", diffResp.StatusCode, string(body))
	}
	var payload struct {
		Empty        bool `json:"empty"`
		Files        []gitDiffFile
		Folders      []gitDiffFolderSummary
		SelectedFile *gitDiffFile  `json:"selected_file"`
		Hunks        []gitDiffHunk `json:"hunks"`
	}
	if err := json.NewDecoder(diffResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode git diff payload: %v", err)
	}
	if payload.Empty || len(payload.Files) != 1 {
		t.Fatalf("expected only selected workspace file in path-filtered diff, got %+v", payload)
	}
	if payload.SelectedFile == nil || payload.SelectedFile.Path != "reports/as-is/overview.md" || payload.SelectedFile.Status != "modified" {
		t.Fatalf("expected selected modified overview file, got %+v", payload.SelectedFile)
	}
	if !gitDiffHunksContain(payload.Hunks, "add", "new evidence") {
		t.Fatalf("expected added line in overview diff, got %+v", payload.Hunks)
	}
	if !gitDiffFolderPresent(payload.Folders, "reports") {
		t.Fatalf("expected reports folder summary, got %+v", payload.Folders)
	}

	allResp, err := http.Get(httpServer.URL + "/api/git/diff")
	if err != nil {
		t.Fatalf("GET full git diff: %v", err)
	}
	defer allResp.Body.Close()
	if allResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(allResp.Body)
		t.Fatalf("expected full diff status 200, got %d body=%s", allResp.StatusCode, string(body))
	}
	var allPayload struct {
		Files []gitDiffFile `json:"files"`
	}
	if err := json.NewDecoder(allResp.Body).Decode(&allPayload); err != nil {
		t.Fatalf("decode full diff payload: %v", err)
	}
	if !gitDiffFileStatus(allPayload.Files, "reports/coverage/summary.md", "deleted") {
		t.Fatalf("expected deleted coverage file, got %+v", allPayload.Files)
	}

	folderResp, err := http.Get(httpServer.URL + "/api/git/diff?folder=proposals")
	if err != nil {
		t.Fatalf("GET git diff folder: %v", err)
	}
	defer folderResp.Body.Close()
	if folderResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(folderResp.Body)
		t.Fatalf("expected folder status 200, got %d body=%s", folderResp.StatusCode, string(body))
	}
	var folderPayload struct {
		Files []gitDiffFile `json:"files"`
	}
	if err := json.NewDecoder(folderResp.Body).Decode(&folderPayload); err != nil {
		t.Fatalf("decode folder diff payload: %v", err)
	}
	if len(folderPayload.Files) != 1 || folderPayload.Files[0].Path != "proposals/proposal-baseline/proposal.md" || folderPayload.Files[0].Status != "untracked" {
		t.Fatalf("expected only untracked proposal in folder filter, got %+v", folderPayload.Files)
	}
}

func TestGitDiffEndpointHandlesEmptyAndInvalidPath(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for git diff API tests")
	}

	server := newTestServer(t)
	ws := initGitWorkspaceForDiffTest(t, server)
	commitWorkspaceForDiffTest(t, ws, "baseline")
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	emptyResp, err := http.Get(httpServer.URL + "/api/git/diff")
	if err != nil {
		t.Fatalf("GET empty git diff: %v", err)
	}
	defer emptyResp.Body.Close()
	if emptyResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(emptyResp.Body)
		t.Fatalf("expected empty diff 200, got %d body=%s", emptyResp.StatusCode, string(body))
	}
	var emptyPayload struct {
		Empty bool          `json:"empty"`
		Files []gitDiffFile `json:"files"`
	}
	if err := json.NewDecoder(emptyResp.Body).Decode(&emptyPayload); err != nil {
		t.Fatalf("decode empty diff: %v", err)
	}
	if !emptyPayload.Empty || len(emptyPayload.Files) != 0 {
		t.Fatalf("expected valid empty diff, got %+v", emptyPayload)
	}

	invalidResp, err := http.Get(httpServer.URL + "/api/git/diff?path=..%2Fworkspace.yaml")
	if err != nil {
		t.Fatalf("GET invalid path git diff: %v", err)
	}
	defer invalidResp.Body.Close()
	if invalidResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid path status 400, got %d", invalidResp.StatusCode)
	}
	var invalidPayload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(invalidResp.Body).Decode(&invalidPayload); err != nil {
		t.Fatalf("decode invalid path payload: %v", err)
	}
	if invalidPayload.Error.Code != "path_invalid" {
		t.Fatalf("expected path_invalid, got %q", invalidPayload.Error.Code)
	}
}

func TestGitDiffEndpointReportsBinarySelectedFile(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for git diff API tests")
	}

	server := newTestServer(t)
	ws := initGitWorkspaceForDiffTest(t, server)
	commitWorkspaceForDiffTest(t, ws, "baseline")
	if err := ws.WriteFile("reports/as-is/blob.bin", []byte{0, 1, 2, 3}); err != nil {
		t.Fatalf("write binary fixture: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	diffResp, err := http.Get(httpServer.URL + "/api/git/diff?path=reports/as-is/blob.bin")
	if err != nil {
		t.Fatalf("GET binary git diff: %v", err)
	}
	defer diffResp.Body.Close()
	if diffResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(diffResp.Body)
		t.Fatalf("expected binary diff 200, got %d body=%s", diffResp.StatusCode, string(body))
	}
	var payload struct {
		SelectedFile *gitDiffFile  `json:"selected_file"`
		Hunks        []gitDiffHunk `json:"hunks"`
		Message      string        `json:"message"`
	}
	if err := json.NewDecoder(diffResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode binary diff payload: %v", err)
	}
	if payload.SelectedFile == nil || !payload.SelectedFile.Binary {
		t.Fatalf("expected selected file to be marked binary, got %+v", payload.SelectedFile)
	}
	if len(payload.Hunks) != 0 || !strings.Contains(payload.Message, "binary") {
		t.Fatalf("expected binary file message without hunks, got message=%q hunks=%+v", payload.Message, payload.Hunks)
	}
}

func TestRunDetailAndArtifactsLoadFromPersistedHistoryAfterRestart(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "payments-service")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	writeAPITestRepoReadme(t, repoPath, "# Payments Service\n")
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

func TestRuntimeExecutionPutRejectsInvalidStepProvider(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	requestBody := `{"execution":{"steps":{"step3_findings":"unsupported-provider"}}}`
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

func TestRuntimePermissionsGetReturnsEffectiveDefaults(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/api/runtime/permissions")
	if err != nil {
		t.Fatalf("GET /api/runtime/permissions: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var payload struct {
		OK        bool                         `json:"ok"`
		Persisted map[string]any               `json:"persisted"`
		Effective acpruntime.PermissionValues  `json:"effective"`
		Source    acpruntime.PermissionSources `json:"source"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode runtime permissions payload: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected ok=true")
	}
	if payload.Effective.Mode != acpruntime.PermissionModeTrustedFullAccess {
		t.Fatalf("expected default permission mode %q, got %q", acpruntime.PermissionModeTrustedFullAccess, payload.Effective.Mode)
	}
	if payload.Effective.ApprovalChannel != acpruntime.PermissionApprovalFailFast {
		t.Fatalf("expected default approval channel %q, got %q", acpruntime.PermissionApprovalFailFast, payload.Effective.ApprovalChannel)
	}
	if payload.Source.Mode != acpruntime.PermissionSourceDefault {
		t.Fatalf("expected default mode source, got %s", payload.Source.Mode)
	}
}

func TestRuntimePermissionsPutSupportsManagedUpdate(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	requestBody := `{"permissions":{"mode":"managed","approval_channel":"ui"}}`
	request, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/runtime/permissions", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("create PUT /api/runtime/permissions request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("PUT /api/runtime/permissions: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var putPayload struct {
		Effective acpruntime.PermissionValues  `json:"effective"`
		Source    acpruntime.PermissionSources `json:"source"`
	}
	if err := json.NewDecoder(response.Body).Decode(&putPayload); err != nil {
		t.Fatalf("decode PUT runtime permissions payload: %v", err)
	}
	if putPayload.Effective.Mode != acpruntime.PermissionModeManaged || putPayload.Effective.ApprovalChannel != acpruntime.PermissionApprovalUI {
		t.Fatalf("unexpected effective permissions: %+v", putPayload.Effective)
	}
	if putPayload.Source.Mode != acpruntime.PermissionSourceWorkspace || putPayload.Source.ApprovalChannel != acpruntime.PermissionSourceWorkspace {
		t.Fatalf("expected workspace sources, got %+v", putPayload.Source)
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
	if !strings.Contains(manifestPayload.Content, "permissions:") || !strings.Contains(manifestPayload.Content, "mode: managed") {
		t.Fatalf("expected runtime permissions in manifest content, got:\n%s", manifestPayload.Content)
	}
	if !strings.Contains(manifestPayload.Content, "approval_channel: ui") {
		t.Fatalf("expected approval_channel in manifest content, got:\n%s", manifestPayload.Content)
	}
}

func TestRuntimePermissionsPutRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	requestBody := `{"permissions":{"mode":"superuser"}}`
	request, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/runtime/permissions", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("create PUT /api/runtime/permissions request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("PUT /api/runtime/permissions: %v", err)
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
	if payload.Error.Code != "runtime_permissions_invalid" {
		t.Fatalf("expected runtime_permissions_invalid code, got %q", payload.Error.Code)
	}
}

func TestRuntimeProfileGetIncludesStepProviders(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "payments-service")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	writeAPITestRepoReadme(t, repoPath, "# Payments Service\n")
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoPath + `
runtime:
  profile:
    permissions:
      mode: managed
      approval_channel: ui
    execution:
      strategy: parallel
      max_parallel_tasks: 2
    steps:
      step2_as_is:
        provider: qwen-code
      step3_findings:
        provider: codex-code
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
		Permissions struct {
			Persisted map[string]string `json:"persisted"`
			Effective map[string]string `json:"effective"`
			Source    map[string]string `json:"source"`
		} `json:"permissions"`
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
	if payload.StepProviders.Persisted["step3_findings"] != "codex-code" {
		t.Fatalf("expected persisted step3_findings=codex-code, got %+v", payload.StepProviders.Persisted)
	}
	if payload.StepProviders.Effective["step3_findings"] != "codex-code" {
		t.Fatalf("expected effective step3_findings=codex-code, got %+v", payload.StepProviders.Effective)
	}
	if payload.StepProviders.Effective["step1_collect"] != "claude-code" {
		t.Fatalf("expected default effective step1_collect=claude-code, got %+v", payload.StepProviders.Effective)
	}
	if payload.StepProviders.Source["step2_as_is"] != "workspace" {
		t.Fatalf("expected workspace source for step2_as_is, got %+v", payload.StepProviders.Source)
	}
	if payload.Permissions.Persisted["mode"] != "managed" || payload.Permissions.Persisted["approval_channel"] != "ui" {
		t.Fatalf("expected persisted managed permissions, got %+v", payload.Permissions.Persisted)
	}
	if payload.Permissions.Effective["mode"] != "managed" || payload.Permissions.Source["mode"] != "workspace" {
		t.Fatalf("expected effective workspace permissions, got effective=%+v source=%+v", payload.Permissions.Effective, payload.Permissions.Source)
	}
	effectiveSteps, ok := payload.Execution.Effective["steps"].(map[string]any)
	if !ok {
		t.Fatalf("expected execution.effective.steps map, got %+v", payload.Execution.Effective)
	}
	if effectiveSteps["step2_as_is"] != "qwen-code" {
		t.Fatalf("expected execution.effective.steps.step2_as_is=qwen-code, got %+v", effectiveSteps)
	}
	if effectiveSteps["step3_findings"] != "codex-code" {
		t.Fatalf("expected execution.effective.steps.step3_findings=codex-code, got %+v", effectiveSteps)
	}
}

func TestRuntimeExecutionPutSupportsStepProviderUpdate(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	requestBody := `{"execution":{"strategy":"parallel","steps":{"step2_as_is":"qwen-code","step3_findings":"codex-code","step4_proposals":"claude-code"}}}`
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
	if !strings.Contains(manifestPayload.Content, "provider: codex-code") {
		t.Fatalf("expected codex step provider in manifest content, got:\n%s", manifestPayload.Content)
	}
}

func TestRunStatusIncludesEffectiveStepProviders(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "payments-service")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	writeAPITestRepoReadme(t, repoPath, "# Payments Service\n")
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoPath + `
runtime:
  profile:
    steps:
      step2_as_is:
        provider: qwen-code
      step3_findings:
        provider: codex-code
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
	if detail.StepProviders["step3_findings"] != "codex-code" {
		t.Fatalf("expected run detail step3_findings=codex-code, got %+v", detail.StepProviders)
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

	skillBody := `{"path":"skills/prompt-packs/qa.md","content":"qa prompt"}`
	skillResponse, err := http.Post(httpServer.URL+"/api/artifacts/write", "application/json", strings.NewReader(skillBody))
	if err != nil {
		t.Fatalf("POST /api/artifacts/write skill: %v", err)
	}
	defer skillResponse.Body.Close()
	if skillResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected skill write status 200, got %d", skillResponse.StatusCode)
	}
}

func TestArtifactsWriteEndpointRejectsForbiddenPaths(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	for _, artifactPath := range []string{
		"charter/../workspace.yaml",
		"skills/../schemas/x",
		"../charter/x",
		"/tmp/x",
		`C:\tmp\x`,
		"charter",
		"skills",
	} {
		artifactPath := artifactPath
		t.Run(artifactPath, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{
				"path":    artifactPath,
				"content": "blocked",
			})
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}
			response, err := http.Post(httpServer.URL+"/api/artifacts/write", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("POST /api/artifacts/write forbidden path: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected status 400 for %q, got %d", artifactPath, response.StatusCode)
			}
			var payload struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload.Error.Code != "artifact_path_forbidden" {
				t.Fatalf("expected artifact_path_forbidden for %q, got %q", artifactPath, payload.Error.Code)
			}
		})
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
		t.Fatalf("POST /api/pipeline/init contract failure runner: %v", err)
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
	if runStatus.ErrorCode != "runtime_contract_failed" {
		t.Fatalf("expected runtime_contract_failed, got %q", runStatus.ErrorCode)
	}
}

func TestRunPermissionsEndpointReturnsPendingRequests(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoPath + `
runtime:
  profile:
    permissions:
      mode: managed
      approval_channel: fail_fast
`
	server := newTestServerFromManifestWithRunner(t, manifest, apiPermissionRequiredRunner{})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/api/pipeline/init",
		"application/json",
		bytes.NewBufferString(`{"trigger":"ui"}`),
	)
	if err != nil {
		t.Fatalf("POST /api/pipeline/init permission runner: %v", err)
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

	runStatus := waitForRunTerminalStatus(t, httpServer.URL, startPayload.RunID, 8*time.Second)
	if runStatus.Status != string(orchestrator.RunStatusFailed) {
		t.Fatalf("expected failed run status, got %q", runStatus.Status)
	}
	if runStatus.ErrorCode != string(acpruntime.ErrorCodePermissionRequired) {
		t.Fatalf("expected runtime_permission_required, got %q", runStatus.ErrorCode)
	}

	detailResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + startPayload.RunID)
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs/<id>: %v", err)
	}
	defer detailResp.Body.Close()
	var detailPayload struct {
		PendingPermissions []acpruntime.PermissionRequest `json:"pending_permissions"`
	}
	if err := json.NewDecoder(detailResp.Body).Decode(&detailPayload); err != nil {
		t.Fatalf("decode run detail payload: %v", err)
	}
	if len(detailPayload.PendingPermissions) != 1 {
		t.Fatalf("expected one pending permission in run status, got %+v", detailPayload.PendingPermissions)
	}

	permissionsResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + startPayload.RunID + "/permissions")
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs/<id>/permissions: %v", err)
	}
	defer permissionsResp.Body.Close()
	if permissionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", permissionsResp.StatusCode)
	}
	var permissionsPayload struct {
		RunID    string                         `json:"run_id"`
		Requests []acpruntime.PermissionRequest `json:"requests"`
	}
	if err := json.NewDecoder(permissionsResp.Body).Decode(&permissionsPayload); err != nil {
		t.Fatalf("decode run permissions payload: %v", err)
	}
	if permissionsPayload.RunID != startPayload.RunID {
		t.Fatalf("expected run_id %q, got %q", startPayload.RunID, permissionsPayload.RunID)
	}
	if len(permissionsPayload.Requests) != 1 || permissionsPayload.Requests[0].RequestID != "perm-api-shell" {
		t.Fatalf("unexpected permission requests: %+v", permissionsPayload.Requests)
	}
	if permissionsPayload.Requests[0].Decision == nil ||
		permissionsPayload.Requests[0].Decision.Decision != acpruntime.PermissionDecisionNeedsUser ||
		permissionsPayload.Requests[0].Decision.RuleID != "ask_unsafe_operation" {
		t.Fatalf("expected pending permission decision metadata, got %+v", permissionsPayload.Requests[0].Decision)
	}
}

func TestRunPermissionsEndpointReturnsEmptyArrays(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoPath + `
runtime:
  profile:
    permissions:
      mode: managed
      approval_channel: fail_fast
`
	server := newTestServerFromManifestWithRunner(t, manifest, fakeruntime.Runner{})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/api/pipeline/init",
		"application/json",
		bytes.NewBufferString(`{"trigger":"ui"}`),
	)
	if err != nil {
		t.Fatalf("POST /api/pipeline/init fake runner: %v", err)
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

	runStatus := waitForRunTerminalStatus(t, httpServer.URL, startPayload.RunID, 8*time.Second)
	if runStatus.Status != string(orchestrator.RunStatusSucceeded) {
		t.Fatalf("expected succeeded run status, got %q", runStatus.Status)
	}

	detailResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + startPayload.RunID)
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs/<id>: %v", err)
	}
	defer detailResp.Body.Close()
	var detailPayload map[string]json.RawMessage
	if err := json.NewDecoder(detailResp.Body).Decode(&detailPayload); err != nil {
		t.Fatalf("decode run detail payload: %v", err)
	}
	if got := strings.TrimSpace(string(detailPayload["pending_permissions"])); got != "[]" {
		t.Fatalf("expected pending_permissions to be [], got %s", got)
	}

	listResp, err := http.Get(httpServer.URL + "/api/pipeline/runs?limit=1")
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs: %v", err)
	}
	defer listResp.Body.Close()
	var listPayload struct {
		Items []map[string]json.RawMessage `json:"items"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listPayload); err != nil {
		t.Fatalf("decode run list payload: %v", err)
	}
	if len(listPayload.Items) != 1 {
		t.Fatalf("expected one run list item, got %d", len(listPayload.Items))
	}
	if got := strings.TrimSpace(string(listPayload.Items[0]["pending_permissions"])); got != "[]" {
		t.Fatalf("expected list pending_permissions to be [], got %s", got)
	}

	permissionsResp, err := http.Get(httpServer.URL + "/api/pipeline/runs/" + startPayload.RunID + "/permissions")
	if err != nil {
		t.Fatalf("GET /api/pipeline/runs/<id>/permissions: %v", err)
	}
	defer permissionsResp.Body.Close()
	var permissionsPayload map[string]json.RawMessage
	if err := json.NewDecoder(permissionsResp.Body).Decode(&permissionsPayload); err != nil {
		t.Fatalf("decode run permissions payload: %v", err)
	}
	if got := strings.TrimSpace(string(permissionsPayload["requests"])); got != "[]" {
		t.Fatalf("expected requests to be [], got %s", got)
	}
}

func TestPipelineStartWithRuntimeContractFailureStillReturnsAccepted(t *testing.T) {
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
		t.Fatalf("POST /api/pipeline/init contract failure runner: %v", err)
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

func TestMapTypedRunnerAPIErrorDoesNotExposeRuntimeContractFailedAtStartTime(t *testing.T) {
	t.Parallel()

	err := acpruntime.WrapRunnerError(
		acpruntime.ProviderClaudeCode,
		acpruntime.ErrorCodeRuntimeContract,
		"runner failed required artifact contract in test",
		errors.New("missing shard-pack-manifest.json"),
	)
	if _, code, _, ok := mapTypedRunnerAPIError(err); ok {
		t.Fatalf("expected runtime_contract_failed to bypass start-time mapping, got mapped code %q", code)
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

type qaAPICitation struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

func newQATestServer(t *testing.T) *Server {
	t.Helper()

	return newQATestServerWithService(t, func(ws workspace.Root) *orchestrator.Service {
		return orchestrator.NewService(orchestrator.WithHistoryWorkspace(ws))
	})
}

func newQATestServerWithService(t *testing.T, newService func(workspace.Root) *orchestrator.Service) *Server {
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
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	fixtures := map[string]string{
		"reports/coverage/summary.md":        "Missing: owner mappings",
		"reports/coverage/open-questions.md": "Who owns payments deployment?",
		"reports/as-is/overview.md":          "Services: payments-service",
		"docs/imports/index.yaml":            "- id: architecture-notes\n  path: docs/imports/architecture-notes.md\n",
		"docs/imports/architecture-notes.md": "Architecture notes mention owner mappings and deployment concerns.",
	}
	for path, content := range fixtures {
		if err := ws.WriteFile(path, []byte(content)); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}

	service := newService(ws)
	return NewServer(ws, service)
}

func hasAPICitationPathPrefix(citations []qaAPICitation, prefix string) bool {
	for _, citation := range citations {
		if strings.HasPrefix(citation.Path, prefix) {
			return true
		}
	}
	return false
}

func snapshotWorkspaceFiles(t *testing.T, root string) map[string]string {
	t.Helper()

	files := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(content)
		return nil
	}); err != nil {
		t.Fatalf("snapshot workspace files: %v", err)
	}
	return files
}

func assertWorkspaceSnapshotEqual(t *testing.T, before map[string]string, after map[string]string) {
	t.Helper()

	if len(before) != len(after) {
		t.Fatalf("expected workspace file count to remain %d, got %d", len(before), len(after))
	}
	for path, beforeContent := range before {
		afterContent, ok := after[path]
		if !ok {
			t.Fatalf("expected workspace file %s to remain present", path)
		}
		if afterContent != beforeContent {
			t.Fatalf("expected workspace file %s to remain unchanged", path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			t.Fatalf("expected no new workspace file, got %s", path)
		}
	}
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

func newTestServerFromManifestWithRunner(t *testing.T, manifest string, runner acpruntime.Runner) *Server {
	t.Helper()

	root := writeManifestRoot(t, manifest)
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

func testServerRuntimeConfig() ServerRuntimeConfig {
	return ServerRuntimeConfig{
		Mode:           acpruntime.RuntimeModeFake,
		Provider:       acpruntime.ProviderClaudeCode,
		ProviderSource: acpruntime.ProviderSourceDefault,
		RunLogsTTL:     168 * time.Hour,
		RunLogsMaxRuns: 200,
	}
}

func testLauncherServiceFactory() ServiceFactory {
	return func(ws workspace.Root, _ ServerRuntimeConfig) *orchestrator.Service {
		return orchestrator.NewService(
			orchestrator.WithHistoryWorkspace(ws),
			orchestrator.WithRunner(fakeruntime.Runner{}),
		)
	}
}

func withOnboardingRecentWorkspacesPath(t *testing.T, path string) {
	t.Helper()
	previous := onboardingRecentWorkspacesPath
	onboardingRecentWorkspacesPath = func() (string, error) {
		return path, nil
	}
	t.Cleanup(func() {
		onboardingRecentWorkspacesPath = previous
	})
}

func postJSON(t *testing.T, url string, body string) *http.Response {
	t.Helper()
	return postJSONWithMethod(t, http.MethodPost, url, body)
}

func postJSONWithMethod(t *testing.T, method string, url string, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("create %s request: %v", method, err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return response
}

func decodeErrorCode(t *testing.T, response *http.Response) string {
	t.Helper()
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	return payload.Error.Code
}

func writeHeadlessRunnerStub(t *testing.T, runtimeName string) string {
	t.Helper()

	script := `#!/usr/bin/env bash
set -eu
TASK_PAYLOAD="$(cat)"
LAST_ARG=""
for arg in "$@"; do
  LAST_ARG="$arg"
done
TASK_PAYLOAD="$TASK_PAYLOAD" TASK_PROMPT="$LAST_ARG" python3 - <<'PY'
import glob
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
    patterns = [
        r'%s(?:\s+\([^)]+\))?\s*=\s*"([^"]+)"',
        r'"%s"\s*:\s*"([^"]+)"',
    ]
    for pattern in patterns:
        match = re.search(pattern % re.escape(field), prompt)
        if match:
            return match.group(1).strip()
    return ""

def step_id_from_prompt():
    match = re.search(r"STEP POLICY ([A-Za-z0-9._-]+):", prompt)
    if match:
        return match.group(1).strip()
    if "Write constitution-draft.json in write_root." in prompt:
        return "init.step0.constitution"
    if "Write asis-draft-manifest.json in write_root." in prompt:
        return "init.step2.asis_docs"
    if "Write validator-verdict.json in write_root." in prompt:
        return "init.step3.findings"
    if "Write proposals-draft-manifest.json in write_root." in prompt:
        return "init.step4.proposals"
    if "shard-pack-manifest.json" in prompt:
        return "init.step1.collect"
    return ""

def first_non_empty_list(mapping, keys):
    for key in keys:
        value = mapping.get(key)
        if isinstance(value, list) and value:
            return [str(item).strip() for item in value if str(item).strip()]
    return []

def slugify(value):
    return re.sub(r'[^a-z0-9]+', '-', value.lower()).strip('-') or "stub"

def infer_repo_scope_from_shard(shard):
    slug = slugify(shard)
    for suffix in [
        "-readme-md",
        "-makefile",
        "-dockerfile",
        "-package-json",
        "-pom-xml",
        "-build-gradle",
        "-settings-gradle",
    ]:
        if slug.endswith(suffix) and len(slug) > len(suffix):
            return slug[:-len(suffix)]
    return slug

task = {}
if raw:
    try:
        task = json.loads(raw)
    except Exception:
        task = {}

task_id = first_non_empty(task, ["task_id", "TaskID"]) or from_prompt("TaskID") or from_prompt("task_id") or "task"
step_id = first_non_empty(task, ["step_id", "StepID"]) or from_prompt("StepID") or from_prompt("step_id") or step_id_from_prompt() or "init.step1.collect"
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
if not repo_scopes:
    inferred_repo_scope = infer_repo_scope_from_shard(shard_id)
    if inferred_repo_scope:
        repo_scopes = [inferred_repo_scope]
question = first_non_empty(task, ["question", "Question"]) or from_prompt("question") or from_prompt("Question")
context_pack_path = first_non_empty(task, ["context_pack_path", "ContextPackPath"]) or from_prompt("context_pack_path") or from_prompt("ContextPackPath")

def shard_completeness_line():
    if not draft_root or not run_id:
        return "Shard completeness: planned=unknown succeeded=unknown failed=unknown incomplete=unknown; typed shard summary was not visible to the stub runner."
    taskruns_root = os.path.abspath(os.path.join(draft_root, "..", "..", "..", ".."))
    matches = sorted(glob.glob(os.path.join(taskruns_root, f"{run_id}-*-step1-collect-shard-summary-*.json")))
    for path in matches:
        try:
            with open(path, "r", encoding="utf-8") as handle:
                items = json.load(handle).get("items", [])
        except Exception:
            continue
        if not items:
            continue
        planned = len(items)
        succeeded = sum(1 for item in items if str(item.get("status", "")).strip().lower() == "succeeded")
        failed = sum(1 for item in items if str(item.get("status", "")).strip().lower() == "failed")
        incomplete = planned - succeeded - failed
        if failed == 0 and incomplete == 0:
            return f"Shard completeness: {succeeded}/{planned} succeeded; no failed, pending, or incomplete shard statuses were observed in the current-run typed shard summary."
        return f"Shard completeness: planned={planned} succeeded={succeeded} failed={failed} incomplete={incomplete} from the current-run typed shard summary."
    return "Shard completeness: planned=unknown succeeded=unknown failed=unknown incomplete=unknown; typed shard summary was not visible to the stub runner."

def write_runtime_draft(manifest_name, outputs, default_step_contract="draft"):
    if not write_root or not draft_root:
        return
    os.makedirs(write_root, exist_ok=True)
    os.makedirs(draft_root, exist_ok=True)
    rendered_outputs = []
    for output in outputs:
        draft_name = output["path"]
        with open(os.path.join(draft_root, draft_name), "w", encoding="utf-8") as handle:
            handle.write(output.get("content", "# Stub Draft\n"))
        rendered_outputs.append(
            {
                "path": draft_name,
                "canonical_path": output["canonical_path"],
                "kind": output["kind"],
                "title": output["title"],
            }
        )
    manifest = {
        "version": 1,
        "run_id": run_id or "run-1",
        "step_id": step_id,
        "step_contract": step_contract or default_step_contract,
        "agent_role": agent_role,
        "summary": "stub runtime draft",
        "outputs": rendered_outputs,
    }
    with open(os.path.join(write_root, manifest_name), "w", encoding="utf-8") as handle:
        json.dump(manifest, handle)

if step_id == "init.step0.constitution":
    write_runtime_draft(
        "constitution-draft.json",
        [
            {
                "path": "charter-overview.md",
                "canonical_path": "charter/overview.md",
                "kind": "charter",
                "title": "Stub Constitution",
                "content": "# Stub Constitution\n\n## Scope\n- Stub runner evidence: README.md.\n",
            },
            {
                "path": "baseline-subagents.yaml",
                "canonical_path": "skills/subagents.yaml",
                "kind": "bundle",
                "title": "Baseline Subagents",
                "content": "agents: []\n",
            }
        ],
    )
elif step_id in {"init.step2.asis_docs", "refresh.step2.asis_docs"}:
    write_runtime_draft(
        "asis-draft-manifest.json",
        [
            {
                "path": "overview.md",
                "canonical_path": "reports/as-is/overview.md",
                "kind": "report",
                "title": "Stub As-Is Overview",
                "content": "# Stub As-Is Overview\n\nEvidence references: reports/as-is/overview.md.\n",
            },
            {
                "path": "summary.md",
                "canonical_path": "reports/coverage/summary.md",
                "kind": "report",
                "title": "Stub Coverage Summary",
                "content": "# Stub Coverage Summary\n\n" + shard_completeness_line() + "\n",
            },
            {
                "path": "architect-summary.md",
                "canonical_path": "reports/agent-outputs/architect/summary.md",
                "kind": "agent-output",
                "title": "Stub Architect Summary",
                "content": "# Stub Architect Summary\n\nWhat is complete: " + shard_completeness_line() + "\n\nWhat to inspect next: review generated as-is and coverage artifacts.\n",
            },
        ],
        "as_is",
    )
elif step_id in {"init.step3.findings", "refresh.step3.findings"} and write_root:
    os.makedirs(write_root, exist_ok=True)
    verdict = {
        "version": 1,
        "run_id": run_id or "run-1",
        "generated_at": "2026-04-21T10:00:00Z",
        "verdict": "PASS",
        "summary": "stub validator verdict",
        "checked_paths": ["reports/taskruns/" + (run_id or "run-1") + "/staging/final/final-run-index.json"],
        "fixed_paths": [],
        "findings": [],
        "questions": [],
    }
    with open(os.path.join(write_root, "validator-verdict.json"), "w", encoding="utf-8") as handle:
        json.dump(verdict, handle)
elif step_id in {"init.step4.proposals", "refresh.step4.proposals"}:
    write_runtime_draft(
        "proposals-draft-manifest.json",
        [
            {
                "path": "proposal.md",
                "canonical_path": "proposals/proposal-baseline/proposal.md",
                "kind": "proposal",
                "title": "Stub Proposal",
            }
        ],
    )
elif step_id == "qa.ask" and write_root:
    os.makedirs(write_root, exist_ok=True)
    documents = []
    if context_pack_path:
        try:
            with open(context_pack_path, "r", encoding="utf-8") as handle:
                documents = json.load(handle).get("documents", [])
        except Exception:
            documents = []
    citations = []
    for document in documents[:3]:
        path = str(document.get("path", "")).strip()
        if path:
            citations.append({"path": path, "reason": "selected from QA context pack by headless test stub"})
    if not citations:
        citations = [{"path": "reports/taskruns/" + (run_id or "run-1") + "/qa/context-pack.json", "reason": "context pack was available but no source document was selected"}]
    answer = {
        "version": 1,
        "run_id": run_id or "run-1",
        "question": question or "test question",
        "answer": "Headless test stub answered from the QA context pack.",
        "citations": citations,
        "unresolved": [],
        "confidence": 0.74,
        "provider": "headless-test-stub",
        "generated_at": "2026-04-21T10:00:00Z",
    }
    with open(os.path.join(write_root, "qa-answer.json"), "w", encoding="utf-8") as handle:
        json.dump(answer, handle)

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
        "semantic": {
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
PY
`
	return testutil.WriteExecutableScript(t, "headless-runner-stub.sh", script)
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
	seedExistingManifestRepoReadmes(t, manifest)
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return root
}

func seedExistingManifestRepoReadmes(t *testing.T, manifest string) {
	t.Helper()
	for _, line := range strings.Split(manifest, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "path: ") {
			continue
		}
		repoPath := strings.TrimSpace(strings.TrimPrefix(trimmed, "path: "))
		if repoPath == "" {
			continue
		}
		info, err := os.Stat(repoPath)
		if err != nil || !info.IsDir() {
			continue
		}
		readmePath := filepath.Join(repoPath, "README.md")
		if _, err := os.Stat(readmePath); err == nil {
			continue
		}
		if err := os.WriteFile(readmePath, []byte("# Test Repository\n"), 0o644); err != nil {
			t.Fatalf("write test repo README %s: %v", readmePath, err)
		}
	}
}

func writeAPITestRepoReadme(t *testing.T, repoPath string, content string) {
	t.Helper()
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path for README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write repo README: %v", err)
	}
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

func initGitWorkspaceForDiffTest(t *testing.T, server *Server) workspace.Root {
	t.Helper()
	ws := server.getWorkspace()
	runGitTestCommand(t, ws.Path, "init")
	runGitTestCommand(t, ws.Path, "config", "user.email", "acp-test@example.test")
	runGitTestCommand(t, ws.Path, "config", "user.name", "ACP Test")
	return ws
}

func commitWorkspaceForDiffTest(t *testing.T, ws workspace.Root, message string) {
	t.Helper()
	runGitTestCommand(t, ws.Path, "add", "-A")
	runGitTestCommand(t, ws.Path, "commit", "-m", message)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func gitDiffHunksContain(hunks []gitDiffHunk, kind string, content string) bool {
	for _, hunk := range hunks {
		for _, line := range hunk.Lines {
			if line.Kind == kind && line.Content == content {
				return true
			}
		}
	}
	return false
}

func gitDiffFileStatus(files []gitDiffFile, path string, status string) bool {
	for _, file := range files {
		if file.Path == path && file.Status == status {
			return true
		}
	}
	return false
}

func gitDiffFolderPresent(folders []gitDiffFolderSummary, folder string) bool {
	for _, summary := range folders {
		if summary.Folder == folder {
			return true
		}
	}
	return false
}

type runStatusPayload struct {
	Status    string `json:"status"`
	ErrorCode string `json:"error_code"`
}

func waitForRunTerminalStatus(t *testing.T, serverURL string, runID string, timeout time.Duration) runStatusPayload {
	t.Helper()

	const minAsyncRunTimeout = 30 * time.Second
	if timeout < minAsyncRunTimeout {
		timeout = minAsyncRunTimeout
	}

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
		acpruntime.ErrorCodeRuntimeContract,
		"runner failed required artifact contract in test",
		nil,
	)
}

func (parseFailureRunner) Preflight(context.Context) error {
	return nil
}

type qaInvalidCitationRunner struct{}

func (qaInvalidCitationRunner) Run(_ context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if err := os.MkdirAll(task.WriteRoot, 0o755); err != nil {
		return acpruntime.Result{}, err
	}
	answer := map[string]any{
		"version":      1,
		"run_id":       task.RunID,
		"question":     task.Question,
		"answer":       "This answer cites a taskrun audit artifact instead of context-pack evidence.",
		"citations":    []map[string]string{{"path": "reports/taskruns/" + task.RunID + "/qa/context-pack.json", "reason": "invalid audit citation"}},
		"unresolved":   []string{},
		"confidence":   0.8,
		"provider":     "qa-invalid-citation",
		"generated_at": "2026-05-26T18:30:00Z",
	}
	raw, err := json.MarshalIndent(answer, "", "  ")
	if err != nil {
		return acpruntime.Result{}, err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "qa-answer.json"), raw, 0o644); err != nil {
		return acpruntime.Result{}, err
	}
	return acpruntime.Result{
		Execution: acpruntime.NewExecution(task, acpruntime.ProviderClaudeCode, "test", "succeeded", task.StartedAtUTC.Add(time.Second), nil),
	}, nil
}

func (qaInvalidCitationRunner) Preflight(context.Context) error {
	return nil
}

type apiPermissionRequiredRunner struct{}

func (apiPermissionRequiredRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if task.OnPermissionRequest == nil {
		return acpruntime.Result{}, errors.New("expected runtime permission hook")
	}
	decision := task.OnPermissionRequest(acpruntime.PermissionRequest{
		RequestID:     "perm-api-shell",
		Action:        "shell",
		PathOrCommand: "npm install",
		Reason:        "package install requires review",
	})
	if decision.Approved() {
		return fakeruntime.Runner{}.Run(ctx, task)
	}
	return acpruntime.Result{}, acpruntime.WrapRunnerError(
		acpruntime.Provider("fake"),
		acpruntime.ErrorCodePermissionRequired,
		"runtime permission required",
		errors.New("package install requires review"),
	)
}

func (apiPermissionRequiredRunner) Preflight(context.Context) error {
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
	return fakeruntime.Runner{}.Run(ctx, task)
}

func (cancellableDelayedRunner) Preflight(context.Context) error {
	return nil
}

type streamingRunLogsRunner struct{}

func (streamingRunLogsRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	result, err := fakeruntime.Runner{}.Run(ctx, task)
	if err != nil {
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
	result.Execution.RuntimeVersion = "streaming-test-runner"
	return result, nil
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
	shardCompleteness := syntheticServerShardCompletenessLine(task)

	type draftSpec struct {
		manifest string
		outputs  []map[string]any
	}

	var spec draftSpec
	switch strings.TrimSpace(task.StepID) {
	case "init.step0.constitution":
		spec = draftSpec{
			manifest: "constitution-draft.json",
			outputs: []map[string]any{
				{
					"path":           "charter-overview.md",
					"canonical_path": "charter/overview.md",
					"kind":           "charter",
					"title":          "Stub Constitution",
					"content":        "# Stub Constitution\n\n## Scope\n- Stub runner evidence: `README.md`.\n",
				},
				{
					"path":           "baseline-subagents.yaml",
					"canonical_path": "skills/subagents.yaml",
					"kind":           "bundle",
					"title":          "Baseline Subagents",
					"content":        "agents: []\n",
				},
			},
		}
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		spec = draftSpec{
			manifest: "asis-draft-manifest.json",
			outputs: []map[string]any{
				{
					"path":           "overview.md",
					"canonical_path": "reports/as-is/overview.md",
					"kind":           "report",
					"title":          "Stub As-Is Overview",
					"content":        "# Stub As-Is Overview\n\nEvidence references: reports/as-is/overview.md.\n",
				},
				{
					"path":           "summary.md",
					"canonical_path": "reports/coverage/summary.md",
					"kind":           "report",
					"title":          "Stub Coverage Summary",
					"content":        "# Stub Coverage Summary\n\n" + shardCompleteness + "\n",
				},
				{
					"path":           "architect-summary.md",
					"canonical_path": "reports/agent-outputs/architect/summary.md",
					"kind":           "agent-output",
					"title":          "Stub Architect Summary",
					"content":        "# Stub Architect Summary\n\nWhat is complete: " + shardCompleteness + "\n\nWhat to inspect next: review generated as-is and coverage artifacts.\n",
				},
			},
		}
	case "init.step4.proposals", "refresh.step4.proposals":
		spec = draftSpec{
			manifest: "proposals-draft-manifest.json",
			outputs: []map[string]any{
				{
					"path":           "proposal.md",
					"canonical_path": "proposals/proposal-baseline/proposal.md",
					"kind":           "proposal",
					"title":          "Stub Proposal",
					"content":        "# Stub Proposal\n",
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
		content, _ := output["content"].(string)
		if content == "" {
			content = "# Stub Draft\n"
		}
		if err := os.WriteFile(filepath.Join(draftRoot, pathValue), []byte(content), 0o644); err != nil {
			return err
		}
	}
	stepContract := task.StepContract
	if strings.TrimSpace(stepContract) == "" {
		switch strings.TrimSpace(task.StepID) {
		case "init.step2.asis_docs", "refresh.step2.asis_docs":
			stepContract = "as_is"
		default:
			stepContract = "draft"
		}
	}
	manifest := map[string]any{
		"version":       1,
		"run_id":        task.RunID,
		"step_id":       task.StepID,
		"step_contract": stepContract,
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

func syntheticServerShardCompletenessLine(task acpruntime.Task) string {
	taskrunsRoot := filepath.Clean(filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "..", "..", "..", ".."))
	runID := strings.TrimSpace(task.RunID)
	if runID == "" {
		return "Shard completeness: planned=unknown succeeded=unknown failed=unknown incomplete=unknown; typed shard summary was not visible to the stub runner."
	}
	matches, err := filepath.Glob(filepath.Join(taskrunsRoot, runID+"-*-step1-collect-shard-summary-*.json"))
	if err != nil || len(matches) == 0 {
		return "Shard completeness: planned=unknown succeeded=unknown failed=unknown incomplete=unknown; typed shard summary was not visible to the stub runner."
	}
	sort.Strings(matches)
	for _, match := range matches {
		line, ok := syntheticServerShardCompletenessLineFromFile(match)
		if ok {
			return line
		}
	}
	return "Shard completeness: planned=unknown succeeded=unknown failed=unknown incomplete=unknown; typed shard summary was not visible to the stub runner."
}

func syntheticServerShardCompletenessLineFromFile(filename string) (string, bool) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return "", false
	}
	var summary struct {
		Items []struct {
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &summary); err != nil || len(summary.Items) == 0 {
		return "", false
	}
	planned := len(summary.Items)
	succeeded := 0
	failed := 0
	incomplete := 0
	for _, item := range summary.Items {
		switch strings.ToLower(strings.TrimSpace(item.Status)) {
		case "succeeded":
			succeeded++
		case "failed":
			failed++
		default:
			incomplete++
		}
	}
	if failed == 0 && incomplete == 0 {
		return fmt.Sprintf("Shard completeness: %d/%d succeeded; no failed, pending, or incomplete shard statuses were observed in the current-run typed shard summary.", succeeded, planned), true
	}
	return fmt.Sprintf("Shard completeness: planned=%d succeeded=%d failed=%d incomplete=%d from the current-run typed shard summary.", planned, succeeded, failed, incomplete), true
}
