package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestRetryPlanRejectsUnknownRun(t *testing.T) {
	server := newTestServer(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs/missing/retry-plan", strings.NewReader(`{}`))
	server.handlePipelineRetryPlan(recorder, request, "missing")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if !strings.Contains(recorder.Body.String(), "run_not_found") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestRetryClosureAndScopeNormalization(t *testing.T) {
	steps := pipelineSteps("init")
	if len(steps) != 5 || steps[2] != "init.step2.asis_docs" {
		t.Fatalf("unexpected init closure: %#v", steps)
	}
	scopes := normalizeRetryScopes([]string{"payments", " users ", "payments", ""})
	if strings.Join(scopes, ",") != "payments,users" {
		t.Fatalf("unexpected scopes: %#v", scopes)
	}
}

func TestRetryClosuresCoverEveryPipelineStep(t *testing.T) {
	steps := pipelineSteps("init")
	for index, requested := range steps {
		execute := steps[index:]
		if execute[0] != requested || execute[len(execute)-1] != "init.step4.proposals" {
			t.Fatalf("closure for %s = %#v", requested, execute)
		}
	}
	refresh := pipelineSteps("refresh")
	if strings.Join(refresh[2:], ",") != "refresh.step3.findings,refresh.step4.proposals" {
		t.Fatalf("unexpected refresh findings closure: %#v", refresh[2:])
	}
}

func TestRetryAcceptsEveryTerminalParentStatus(t *testing.T) {
	for _, status := range []orchestrator.RunStatus{orchestrator.RunStatusSucceeded, orchestrator.RunStatusFailed, orchestrator.RunStatusCanceled} {
		if !retryParentTerminal(status) {
			t.Fatalf("terminal status %q was rejected", status)
		}
	}
	for _, status := range []orchestrator.RunStatus{orchestrator.RunStatusQueued, orchestrator.RunStatusRunning} {
		if retryParentTerminal(status) {
			t.Fatalf("non-terminal status %q was accepted", status)
		}
	}
}

func TestRetryPlanHashChangesWithParentArtifactsAndSourceInput(t *testing.T) {
	server := newTestServer(t)
	snapshot := server.sessionSnapshot()
	artifactPath := "reports/taskruns/parent/staging/shards/payments/manifest.json"
	if err := snapshot.Workspace.WriteFile(artifactPath, []byte("validated-parent-input")); err != nil {
		t.Fatal(err)
	}
	parent := orchestrator.RunInfo{RunID: "parent", Pipeline: "init", Status: orchestrator.RunStatusFailed, CurrentStep: "init.step1.collect"}
	plan := retryPlan{ParentRunID: parent.RunID, Pipeline: parent.Pipeline, RequestedStep: parent.CurrentStep, EffectiveStartStep: parent.CurrentStep, ExecuteSteps: pipelineSteps("init")[1:]}
	artifacts := []orchestrator.Artifact{{Path: artifactPath, Kind: "taskrun"}}
	initial := retryPlanHash(snapshot, parent, artifacts, plan)
	if err := os.WriteFile(filepath.Join(snapshot.Workspace.Path, filepath.FromSlash(artifactPath)), []byte("mutated-parent-input"), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed := retryPlanHash(snapshot, parent, artifacts, plan); changed == initial {
		t.Fatal("parent artifact drift did not invalidate retry plan hash")
	}
	stagingOnly := filepath.Join(snapshot.Workspace.Path, "reports", "taskruns", "parent", "staging", "unlisted.md")
	if err := os.MkdirAll(filepath.Dir(stagingOnly), 0o755); err != nil {
		t.Fatal(err)
	}
	beforeStaging := retryPlanHash(snapshot, parent, artifacts, plan)
	if err := os.WriteFile(stagingOnly, []byte("new staged input"), 0o644); err != nil {
		t.Fatal(err)
	}
	if retryPlanHash(snapshot, parent, artifacts, plan) == beforeStaging {
		t.Fatal("unlisted parent staging drift did not invalidate retry plan hash")
	}
	sourceChanged := snapshot
	sourceChanged.Workspace.Manifest.Repos = append([]workspace.RepoSource(nil), snapshot.Workspace.Manifest.Repos...)
	sourceChanged.Workspace.Manifest.Repos[0].Ref = "changed-source-ref"
	if retrySourceFingerprint(sourceChanged) == retrySourceFingerprint(snapshot) {
		t.Fatal("source input drift did not change retry fingerprint")
	}
}

func TestRetryEndpointRejectsPlanAfterParentStagingDrifts(t *testing.T) {
	server := newTestServer(t)
	snapshot := server.sessionSnapshot()
	runID, err := snapshot.Service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace: snapshot.Workspace, Pipeline: orchestrator.PipelineInit, NonInteractive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(8 * time.Second)
	for {
		parent, ok := snapshot.Service.GetRun(runID)
		if !ok {
			t.Fatalf("parent run %q disappeared", runID)
		}
		if retryParentTerminal(parent.Status) {
			if parent.Status != orchestrator.RunStatusSucceeded {
				t.Fatalf("parent run status = %s, want succeeded: %s", parent.Status, parent.Error)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("parent run %q did not finish", runID)
		}
		time.Sleep(10 * time.Millisecond)
	}

	planRecorder := httptest.NewRecorder()
	planRequest := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs/"+runID+"/retry-plan", strings.NewReader(`{"step_id":"init.step4.proposals"}`))
	server.handlePipelineRetryPlan(planRecorder, planRequest, runID)
	if planRecorder.Code != http.StatusOK {
		t.Fatalf("retry-plan status = %d: %s", planRecorder.Code, planRecorder.Body.String())
	}
	var plan retryPlan
	if err := json.NewDecoder(planRecorder.Body).Decode(&plan); err != nil {
		t.Fatal(err)
	}
	if plan.PlanHash == "" {
		t.Fatal("retry plan hash is empty")
	}

	driftPath := filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "changed-after-plan.txt"))
	if err := snapshot.Workspace.WriteFile(driftPath, []byte("parent staging changed after planning\n")); err != nil {
		t.Fatal(err)
	}
	retryRaw, err := json.Marshal(retryRequest{StepID: plan.RequestedStep, PlanHash: plan.PlanHash})
	if err != nil {
		t.Fatal(err)
	}
	retryRecorder := httptest.NewRecorder()
	retryRequestHTTP := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs/"+runID+"/retry", strings.NewReader(string(retryRaw)))
	server.handlePipelineRetry(retryRecorder, retryRequestHTTP, runID)
	if retryRecorder.Code != http.StatusConflict || !strings.Contains(retryRecorder.Body.String(), "retry_plan_stale") {
		t.Fatalf("stale retry status = %d, body=%s", retryRecorder.Code, retryRecorder.Body.String())
	}
}
