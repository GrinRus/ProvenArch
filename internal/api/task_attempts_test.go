package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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
	if got := runtimeSnapshot.RepositoryPathScopes["payments-service"]; !reflect.DeepEqual(got, []string{"."}) {
		t.Fatalf("snapshot changed admitted repository path scope: %v", got)
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

func TestAttemptRuntimeSnapshotRejectsInvalidRepositoryPathScope(t *testing.T) {
	attempt := producttasks.Attempt{EffectiveRuntime: producttasks.EffectiveRuntime{
		Provider: string(acpruntime.ProviderClaudeCode),
		Scope:    producttasks.Scope{Repositories: []producttasks.RepositoryScope{{Name: "repo", Paths: []string{"../escape"}}}},
	}}
	if _, err := attemptRuntimeSnapshot(attempt); err == nil {
		t.Fatal("expected invalid repository path scope to fail closed")
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

func TestTaskAttemptAdmissionRollbackPreservesTaskProjection(t *testing.T) {
	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	created := createTaskForAttemptTest(t, httpServer.URL, "admission-rollback")
	beforeHistory := server.taskRegistry.Snapshot()
	beforeTask, ok := findTask(beforeHistory, created.TaskID)
	if !ok {
		t.Fatalf("created Task %q is missing before admission", created.TaskID)
	}

	if err := server.getService().Shutdown(context.Background()); err != nil {
		t.Fatalf("close service for admission failure: %v", err)
	}
	response := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"rollback-key"}`)
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected typed admission failure after service close, got %d", response.StatusCode)
	}
	if code := decodeErrorCode(t, response); code != "attempt_start_failed" {
		t.Fatalf("expected attempt_start_failed, got %q", code)
	}

	afterHistory := server.taskRegistry.Snapshot()
	afterTask, ok := findTask(afterHistory, created.TaskID)
	if !ok {
		t.Fatalf("Task %q disappeared after admission rollback", created.TaskID)
	}
	if !reflect.DeepEqual(afterTask, beforeTask) {
		t.Fatalf("admission rollback changed Task projection: before=%+v after=%+v", beforeTask, afterTask)
	}
	for _, attempt := range afterHistory.Attempts {
		if attempt.TaskID == created.TaskID {
			t.Fatalf("failed admission left phantom Attempt: %+v", attempt)
		}
	}
	coordination := server.getService().Coordination()
	if coordination.ActiveRunID != "" || coordination.Pending != nil {
		t.Fatalf("failed admission left service ownership: %+v", coordination)
	}
}

func TestTaskAttemptRerunCreatesChildAttempt(t *testing.T) {
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

	run := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts/"+firstPayload.Attempt.AttemptID+"/rerun", `{"idempotency_key":"child-key","reason":"repair"}`)
	var retryPayload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(run.Body).Decode(&retryPayload); err != nil {
		run.Body.Close()
		t.Fatalf("decode retry: %v", err)
	}
	run.Body.Close()
	if run.StatusCode != http.StatusAccepted || retryPayload.Attempt.AttemptID == firstPayload.Attempt.AttemptID || retryPayload.Attempt.ParentAttemptID == nil || *retryPayload.Attempt.ParentAttemptID != firstPayload.Attempt.AttemptID || retryPayload.Attempt.RetryReason != "repair" {
		t.Fatalf("rerun did not create child attempt: status=%d payload=%+v", run.StatusCode, retryPayload)
	}
	// The Attempt watcher persists terminal history after the runtime finishes.
	// Wait for that write before TempDir cleanup removes the workspace.
	waitForTerminalAttempt(t, server, retryPayload.Attempt.AttemptID)
}

func TestTaskAttemptActionsDistinguishRetryAndRerun(t *testing.T) {
	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	created := createTaskForAttemptTest(t, httpServer.URL, "action-gates")
	first := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"action-parent"}`)
	var parentPayload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(first.Body).Decode(&parentPayload); err != nil {
		first.Body.Close()
		t.Fatalf("decode parent: %v", err)
	}
	first.Body.Close()
	waitForTerminalAttempt(t, server, parentPayload.Attempt.AttemptID)

	wrongAction := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts/"+parentPayload.Attempt.AttemptID+"/retry", `{"idempotency_key":"wrong-action"}`)
	body, _ := io.ReadAll(wrongAction.Body)
	wrongAction.Body.Close()
	if wrongAction.StatusCode != http.StatusConflict || !strings.Contains(string(body), "retry_parent_not_retryable") {
		t.Fatalf("successful parent was accepted by retry: status=%d body=%s", wrongAction.StatusCode, body)
	}

	failureServer := newTestServerWithRunner(t, unavailableRunner{})
	failureHTTP := httptest.NewServer(failureServer.Handler())
	defer failureHTTP.Close()
	failureTask := createTaskForAttemptTest(t, failureHTTP.URL, "retry-gate")
	failed := postJSON(t, failureHTTP.URL+"/api/tasks/"+failureTask.TaskID+"/attempts", `{"idempotency_key":"failed-parent"}`)
	var failedPayload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(failed.Body).Decode(&failedPayload); err != nil {
		failed.Body.Close()
		t.Fatalf("decode failed parent: %v", err)
	}
	failed.Body.Close()
	waitForTerminalAttempt(t, failureServer, failedPayload.Attempt.AttemptID)

	wrongRerun := postJSON(t, failureHTTP.URL+"/api/tasks/"+failureTask.TaskID+"/attempts/"+failedPayload.Attempt.AttemptID+"/rerun", `{"idempotency_key":"wrong-rerun"}`)
	body, _ = io.ReadAll(wrongRerun.Body)
	wrongRerun.Body.Close()
	if wrongRerun.StatusCode != http.StatusConflict || !strings.Contains(string(body), "rerun_parent_not_succeeded") {
		t.Fatalf("failed parent was accepted by rerun: status=%d body=%s", wrongRerun.StatusCode, body)
	}

	retry := postJSON(t, failureHTTP.URL+"/api/tasks/"+failureTask.TaskID+"/attempts/"+failedPayload.Attempt.AttemptID+"/retry", `{"idempotency_key":"valid-retry"}`)
	var retryPayload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(retry.Body).Decode(&retryPayload); err != nil {
		retry.Body.Close()
		t.Fatalf("decode child retry: %v", err)
	}
	retry.Body.Close()
	if retry.StatusCode != http.StatusAccepted || retryPayload.Attempt.ParentAttemptID == nil || *retryPayload.Attempt.ParentAttemptID != failedPayload.Attempt.AttemptID || retryPayload.Attempt.RetryReason != "operator_retry" {
		t.Fatalf("failed parent did not create retry child: status=%d payload=%+v", retry.StatusCode, retryPayload.Attempt)
	}
	waitForTerminalAttempt(t, failureServer, retryPayload.Attempt.AttemptID)
}

func TestTaskAttemptEditDoesNotMutateAdmittedSnapshot(t *testing.T) {
	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	created := createTaskForAttemptTest(t, httpServer.URL, "immutable-edit")
	admission := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"immutable-parent"}`)
	var admitted struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(admission.Body).Decode(&admitted); err != nil {
		admission.Body.Close()
		t.Fatalf("decode admitted attempt: %v", err)
	}
	admission.Body.Close()

	patchRequest, err := http.NewRequest(http.MethodPatch, httpServer.URL+"/api/tasks/"+created.TaskID, strings.NewReader(`{"expected_revision":1,"title":"edited title","goal":"edited goal","context":"edited context"}`))
	if err != nil {
		t.Fatalf("create task edit request: %v", err)
	}
	patchRequest.Header.Set("Content-Type", "application/json")
	patchResponse, err := http.DefaultClient.Do(patchRequest)
	if err != nil {
		t.Fatalf("edit task: %v", err)
	}
	var patched struct {
		Task producttasks.Task `json:"task"`
	}
	if err := json.NewDecoder(patchResponse.Body).Decode(&patched); err != nil {
		patchResponse.Body.Close()
		t.Fatalf("decode edited task: %v", err)
	}
	patchResponse.Body.Close()
	if patchResponse.StatusCode != http.StatusOK || patched.Task.Revision != 2 || patched.Task.Title != "edited title" {
		t.Fatalf("task edit failed: status=%d task=%+v", patchResponse.StatusCode, patched.Task)
	}

	attemptResponse, err := http.Get(httpServer.URL + "/api/tasks/" + created.TaskID + "/attempts/" + admitted.Attempt.AttemptID)
	if err != nil {
		t.Fatalf("read immutable attempt: %v", err)
	}
	var after struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(attemptResponse.Body).Decode(&after); err != nil {
		attemptResponse.Body.Close()
		t.Fatalf("decode immutable attempt: %v", err)
	}
	attemptResponse.Body.Close()
	if attemptResponse.StatusCode != http.StatusOK {
		t.Fatalf("read immutable attempt status=%d", attemptResponse.StatusCode)
	}
	if !reflect.DeepEqual(admitted.Attempt.IntentSnapshot, after.Attempt.IntentSnapshot) || !reflect.DeepEqual(admitted.Attempt.EffectiveRuntime, after.Attempt.EffectiveRuntime) {
		t.Fatalf("Task edit mutated admitted Attempt: before=(%+v,%+v) after=(%+v,%+v)", admitted.Attempt.IntentSnapshot, admitted.Attempt.EffectiveRuntime, after.Attempt.IntentSnapshot, after.Attempt.EffectiveRuntime)
	}
	waitForTerminalAttempt(t, server, admitted.Attempt.AttemptID)
	rerun := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts/"+admitted.Attempt.AttemptID+"/rerun", `{"idempotency_key":"edited-child"}`)
	var child struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(rerun.Body).Decode(&child); err != nil {
		rerun.Body.Close()
		t.Fatalf("decode edited child: %v", err)
	}
	rerun.Body.Close()
	if rerun.StatusCode != http.StatusAccepted || child.Attempt.TaskRevision != 2 || child.Attempt.IntentSnapshot.Title != "edited title" || child.Attempt.IntentSnapshot.Goal != "edited goal" || child.Attempt.ParentAttemptID == nil || *child.Attempt.ParentAttemptID != admitted.Attempt.AttemptID {
		t.Fatalf("edited Task values were not explicitly inherited by child: status=%d child=%+v", rerun.StatusCode, child.Attempt)
	}
	waitForTerminalAttempt(t, server, child.Attempt.AttemptID)
}

func TestTaskAttemptSecondRootRequiresExplicitChildAction(t *testing.T) {
	server := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	created := createTaskForAttemptTest(t, httpServer.URL, "root-once")
	first := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"root-once-parent"}`)
	var payload struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(first.Body).Decode(&payload); err != nil {
		first.Body.Close()
		t.Fatalf("decode first root: %v", err)
	}
	first.Body.Close()
	waitForTerminalAttempt(t, server, payload.Attempt.AttemptID)

	second := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"root-once-second"}`)
	body, _ := io.ReadAll(second.Body)
	second.Body.Close()
	if second.StatusCode != http.StatusConflict || !strings.Contains(string(body), "attempt_action_required") {
		t.Fatalf("second root admission was not rejected: status=%d body=%s", second.StatusCode, body)
	}
}

func TestTaskAttemptChildRejectsEditedForeignScopeBeforePersistence(t *testing.T) {
	server := newTestServerWithRunner(t, unavailableRunner{})
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	created := createTaskForAttemptTest(t, httpServer.URL, "foreign-child-scope")
	first := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts", `{"idempotency_key":"foreign-parent"}`)
	var parent struct {
		Attempt producttasks.Attempt `json:"attempt"`
	}
	if err := json.NewDecoder(first.Body).Decode(&parent); err != nil {
		first.Body.Close()
		t.Fatalf("decode foreign-scope parent: %v", err)
	}
	first.Body.Close()
	waitForTerminalAttempt(t, server, parent.Attempt.AttemptID)

	patchRequest, err := http.NewRequest(http.MethodPatch, httpServer.URL+"/api/tasks/"+created.TaskID, strings.NewReader(`{"expected_revision":1,"scope":{"repositories":[{"name":"not-in-manifest","paths":["."]}]}}`))
	if err != nil {
		t.Fatalf("create foreign scope patch: %v", err)
	}
	patchRequest.Header.Set("Content-Type", "application/json")
	patchResponse, err := http.DefaultClient.Do(patchRequest)
	if err != nil {
		t.Fatalf("patch foreign scope: %v", err)
	}
	patchResponse.Body.Close()
	if patchResponse.StatusCode != http.StatusOK {
		t.Fatalf("patch foreign scope status=%d", patchResponse.StatusCode)
	}

	retry := postJSON(t, httpServer.URL+"/api/tasks/"+created.TaskID+"/attempts/"+parent.Attempt.AttemptID+"/retry", `{"idempotency_key":"foreign-child"}`)
	body, _ := io.ReadAll(retry.Body)
	retry.Body.Close()
	if retry.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "task_scope_invalid") {
		t.Fatalf("foreign child scope was not rejected before persistence: status=%d body=%s", retry.StatusCode, body)
	}
	attempts, err := http.Get(httpServer.URL + "/api/tasks/" + created.TaskID + "/attempts")
	if err != nil {
		t.Fatalf("list child attempts: %v", err)
	}
	defer attempts.Body.Close()
	var listed struct {
		Items []producttasks.Attempt `json:"items"`
	}
	if err := json.NewDecoder(attempts.Body).Decode(&listed); err != nil {
		t.Fatalf("decode child attempt list: %v", err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("foreign child was persisted despite scope rejection: %+v", listed.Items)
	}
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
