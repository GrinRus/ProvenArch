package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	producttasks "github.com/GrinRus/ProvenArch/internal/tasks"
)

func TestAdmittedAttemptPreservesResolvedProviderAndModelProvenance(t *testing.T) {
	t.Setenv(acpruntime.ClaudeModelEnv, "")
	t.Setenv(acpruntime.ClaudeEffortEnv, "")
	t.Setenv(acpruntime.QwenModelEnv, "")
	t.Setenv(acpruntime.CodexModelEnv, "")
	t.Setenv(acpruntime.CodexReasoningEffortEnv, "")
	server := newTestServerFromManifest(t, `version: 1
repos:
  - name: payments-service
    path: /tmp
runtime:
  profile:
    steps:
      step1_collect:
        provider: qwen-code
      step3_findings:
        provider: codex-code
    providers:
      qwen-code:
        model: qwen-workspace
      codex-code:
        model: codex-workspace
        effort: high
`)
	snapshot := server.sessionSnapshot()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	task := producttasks.Task{
		Version: 1, TaskID: "task-rem12-provenance", Title: "provenance", Goal: "preserve provenance",
		Scope:         producttasks.Scope{Repositories: []producttasks.RepositoryScope{{Name: "payments-service", Paths: []string{"."}}}},
		DesiredRunner: producttasks.RunnerPreset{Preset: "default"}, Lifecycle: producttasks.LifecycleOpen, Revision: 1,
		CreatedAt: now, UpdatedAt: now, LastActivityAt: now, Attempts: []producttasks.AttemptSummary{},
		Outcome:     producttasks.Outcome{State: producttasks.Unavailable, UnavailableReason: "no attempt has completed"},
		Publication: producttasks.Publication{State: producttasks.PublicationUnavailable, UnavailableReason: "no publication has been recorded"},
	}
	attempt, err := server.buildAdmittedAttempt(snapshot, task, orchestrator.PipelineInit, orchestrator.RunIntentStart, "key", "", nil, "")
	if err != nil {
		t.Fatalf("build admitted attempt: %v", err)
	}
	if got := attempt.EffectiveRuntime.StepOverrides[acpruntime.StepProviderStep1Collect].Provider; got != string(acpruntime.ProviderQwenCode) {
		t.Fatalf("step1 provider drifted to global fallback: %q", got)
	}
	if got := attempt.EffectiveRuntime.StepOverrides[acpruntime.StepProviderStep3Findings].Provider; got != string(acpruntime.ProviderCodexCode) {
		t.Fatalf("step3 provider drifted to global fallback: %q", got)
	}
	if got := attempt.EffectiveRuntime.ResolutionSources["step."+acpruntime.StepProviderStep1Collect+".provider"]; got != "workspace" {
		t.Fatalf("step1 provider source was not retained: %q", got)
	}
	if got := attempt.EffectiveRuntime.ResolutionSources["provider.claude-code.model"]; got != "provider_default" {
		t.Fatalf("default model source was not retained: %q", got)
	}
	if got := attempt.EffectiveRuntime.StepOverrides[acpruntime.StepProviderStep3Findings].Effort; got != "high" {
		t.Fatalf("step3 effort was not retained: %q", got)
	}

	runtimeSnapshot, err := attemptRuntimeSnapshot(attempt)
	if err != nil {
		t.Fatalf("rebuild runtime snapshot: %v", err)
	}
	if got := runtimeSnapshot.StepProviders[acpruntime.StepProviderStep1Collect]; got != acpruntime.ProviderQwenCode {
		t.Fatalf("snapshot changed step1 provider: %q", got)
	}
	if got := runtimeSnapshot.StepProviderSources[acpruntime.StepProviderStep1Collect]; got != acpruntime.ProviderSourceWorkspace {
		t.Fatalf("snapshot changed step1 provider source: %q", got)
	}
	if got := runtimeSnapshot.ProviderModels[acpruntime.ProviderQwenCode].Model; got != "qwen-workspace" {
		t.Fatalf("snapshot lost qwen model: %q", got)
	}
	if got := runtimeSnapshot.ProviderModelSources[acpruntime.ProviderCodexCode].Effort; got != acpruntime.ProviderModelSourceWorkspace {
		t.Fatalf("snapshot changed codex effort source: %q", got)
	}
	if got := runtimeSnapshot.ProviderModelSources[acpruntime.ProviderClaudeCode].Model; got != acpruntime.ProviderModelSourceDefault {
		t.Fatalf("snapshot changed default model source: %q", got)
	}

	t.Setenv(acpruntime.QwenModelEnv, "qwen-env")
	envAttempt, err := server.buildAdmittedAttempt(snapshot, task, orchestrator.PipelineInit, orchestrator.RunIntentStart, "env-key", "", nil, "")
	if err != nil {
		t.Fatalf("build env-model attempt: %v", err)
	}
	if got := envAttempt.EffectiveRuntime.StepOverrides[acpruntime.StepProviderStep1Collect].Model; got != "qwen-env" {
		t.Fatalf("env model was not admitted: %q", got)
	}
	if got := envAttempt.EffectiveRuntime.ResolutionSources["provider.qwen-code.model"]; got != "env" {
		t.Fatalf("env model source was not retained: %q", got)
	}
	envSnapshot, err := attemptRuntimeSnapshot(envAttempt)
	if err != nil {
		t.Fatalf("rebuild env-model runtime snapshot: %v", err)
	}
	if got := envSnapshot.ProviderModelSources[acpruntime.ProviderQwenCode].Model; got != acpruntime.ProviderModelSourceEnv {
		t.Fatalf("snapshot changed env model source: %q", got)
	}

	task.DesiredRunner = producttasks.RunnerPreset{Preset: "default", Provider: string(acpruntime.ProviderCodexCode), Model: "codex-task", Effort: "high"}
	taskAttempt, err := server.buildAdmittedAttempt(snapshot, task, orchestrator.PipelineInit, orchestrator.RunIntentStart, "task-key", "", nil, "")
	if err != nil {
		t.Fatalf("build task-preset attempt: %v", err)
	}
	if got := taskAttempt.EffectiveRuntime.ResolutionSources["provider.codex-code.model"]; got != "task_preset" {
		t.Fatalf("task model source was not retained: %q", got)
	}
	taskSnapshot, err := attemptRuntimeSnapshot(taskAttempt)
	if err != nil {
		t.Fatalf("rebuild task-preset runtime snapshot: %v", err)
	}
	if got := taskSnapshot.ProviderModels[acpruntime.ProviderCodexCode].Model; got != "codex-task" {
		t.Fatalf("task model was not admitted: %q", got)
	}
	if got := taskSnapshot.ProviderModelSources[acpruntime.ProviderCodexCode].Model; got != acpruntime.ProviderModelSourceTaskPreset {
		t.Fatalf("task model source was relabeled: %q", got)
	}
}

func TestAttemptRuntimeSnapshotFailsClosedOnInvalidPersistedProvider(t *testing.T) {
	attempt := producttasks.Attempt{EffectiveRuntime: producttasks.EffectiveRuntime{Provider: "unknown-provider"}}
	if _, err := attemptRuntimeSnapshot(attempt); err == nil {
		t.Fatal("expected invalid persisted provider to fail closed")
	}
	attempt.EffectiveRuntime.Provider = string(acpruntime.ProviderClaudeCode)
	attempt.EffectiveRuntime.StepOverrides = map[string]producttasks.RunnerPreset{
		acpruntime.StepProviderStep0Constitution: {Provider: string(acpruntime.ProviderClaudeCode)},
	}
	if _, err := attemptRuntimeSnapshot(attempt); err == nil {
		t.Fatal("expected incomplete persisted step identity to fail closed")
	}
}

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
	// The Attempt watcher persists terminal history after the runtime finishes.
	// Wait for that write before TempDir cleanup removes the workspace.
	waitForTerminalAttempt(t, server, retryPayload.Attempt.AttemptID)
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

func TestAttemptWatcherStopsAfterRunCancellation(t *testing.T) {
	server := newTestServerWithRunner(t, cancellableDelayedRunner{delay: 5 * time.Second})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	created := createTaskForAttemptTest(t, httpServer.URL, "watcher-cancel")
	response := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"watcher-cancel-key"}`)
	var payload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		response.Body.Close()
		t.Fatalf("decode admission: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("admission failed: status=%d payload=%+v", response.StatusCode, payload)
	}
	waitForAttemptStatus(t, server, payload.Attempt.AttemptID, producttasks.AttemptRunning)
	if err := server.getService().CancelRun(payload.Attempt.RunID); err != nil {
		t.Fatalf("cancel run: %v", err)
	}
	waitForTerminalAttempt(t, server, payload.Attempt.AttemptID)
	waitForAttemptWatchers(t, server)
}

func TestAttemptWatcherStopsWithServerShutdown(t *testing.T) {
	server := newTestServerWithRunner(t, cancellableDelayedRunner{delay: 5 * time.Second})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	created := createTaskForAttemptTest(t, httpServer.URL, "watcher-shutdown")
	response := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"watcher-shutdown-key"}`)
	var payload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		response.Body.Close()
		t.Fatalf("decode admission: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("admission failed: status=%d payload=%+v", response.StatusCode, payload)
	}
	waitForAttemptStatus(t, server, payload.Attempt.AttemptID, producttasks.AttemptRunning)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown server: %v", err)
	}
	waitForAttemptWatchers(t, server)
	history := server.taskRegistry.Snapshot()
	attempt, ok := findAttempt(history, created.TaskID, payload.Attempt.AttemptID)
	if !ok || attempt.Status != producttasks.AttemptCanceled {
		t.Fatalf("shutdown did not publish canceled Attempt: ok=%v attempt=%+v", ok, attempt)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("repeated shutdown: %v", err)
	}
}

func TestQueuedAttemptFailsClosedAfterServiceRestart(t *testing.T) {
	server := newTestServerWithRunner(t, cancellableDelayedRunner{delay: 5 * time.Second})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	activeTask := createTaskForAttemptTest(t, httpServer.URL, "restart-active")
	activeResponse := postJSON(t, httpServer.URL+"/api/tasks/"+activeTask.TaskID+"/attempts", `{"idempotency_key":"restart-active-key"}`)
	var activePayload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(activeResponse.Body).Decode(&activePayload); err != nil {
		activeResponse.Body.Close()
		t.Fatalf("decode active admission: %v", err)
	}
	activeResponse.Body.Close()
	if activeResponse.StatusCode != http.StatusAccepted {
		t.Fatalf("active admission failed: %d", activeResponse.StatusCode)
	}
	waitForAttemptStatus(t, server, activePayload.Attempt.AttemptID, producttasks.AttemptRunning)

	queuedTask := createTaskForAttemptTest(t, httpServer.URL, "restart-queued")
	queuedResponse := postJSON(t, httpServer.URL+"/api/tasks/"+queuedTask.TaskID+"/attempts", `{"idempotency_key":"restart-queued-key","pipeline":"refresh","intent":"queue"}`)
	var queuedPayload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(queuedResponse.Body).Decode(&queuedPayload); err != nil {
		queuedResponse.Body.Close()
		t.Fatalf("decode queued admission: %v", err)
	}
	queuedResponse.Body.Close()
	if queuedResponse.StatusCode != http.StatusAccepted || queuedPayload.Attempt.Status != producttasks.AttemptQueued {
		t.Fatalf("queued admission failed: status=%d attempt=%+v", queuedResponse.StatusCode, queuedPayload.Attempt)
	}
	beforeIntent := queuedPayload.Attempt.IntentSnapshot
	beforeRuntime := queuedPayload.Attempt.EffectiveRuntime

	restartedService := orchestrator.NewService(
		orchestrator.WithHistoryWorkspace(server.getWorkspace()),
		orchestrator.WithRunner(cancellableDelayedRunner{delay: time.Second}),
	)
	restartedService.ReconcileStaleRunsAfterRestart()
	restartedServer := NewServer(server.getWorkspace(), restartedService)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := restartedServer.Shutdown(ctx); err != nil {
			t.Errorf("shutdown restarted server: %v", err)
		}
	}()

	after, ok := findAttempt(restartedServer.taskRegistry.Snapshot(), queuedTask.TaskID, queuedPayload.Attempt.AttemptID)
	if !ok {
		t.Fatal("queued Attempt disappeared after restart reconciliation")
	}
	if after.Status != producttasks.AttemptFailed || after.TerminalSummary == nil || after.TerminalSummary.ErrorCode != "run_reconciled_after_restart" {
		t.Fatalf("queued Attempt did not fail closed with restart diagnostic: %+v", after)
	}
	if after.TaskID != queuedPayload.Attempt.TaskID || after.AttemptID != queuedPayload.Attempt.AttemptID || after.RunID != queuedPayload.Attempt.RunID {
		t.Fatalf("restart reconciliation changed exact Task/Attempt/run identity: before=%+v after=%+v", queuedPayload.Attempt, after)
	}
	if !reflect.DeepEqual(beforeIntent, after.IntentSnapshot) || !reflect.DeepEqual(beforeRuntime, after.EffectiveRuntime) {
		t.Fatalf("restart reconciliation changed immutable admission context: before=(%+v,%+v) after=(%+v,%+v)", beforeIntent, beforeRuntime, after.IntentSnapshot, after.EffectiveRuntime)
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

func waitForAttemptStatus(t *testing.T, server *Server, attemptID string, want producttasks.AttemptStatus) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, attempt := range server.taskRegistry.Snapshot().Attempts {
			if attempt.AttemptID == attemptID && attempt.Status == want {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("attempt %s did not reach %s before timeout; coordination=%+v", attemptID, want, server.getService().Coordination())
}

func waitForAttemptWatchers(t *testing.T, server *Server) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		server.attemptWatchWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("attempt watcher did not become quiescent")
	}
}
