package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	sourceChanged := snapshot
	sourceChanged.Workspace.Manifest.Repos = append([]workspace.RepoSource(nil), snapshot.Workspace.Manifest.Repos...)
	sourceChanged.Workspace.Manifest.Repos[0].Ref = "changed-source-ref"
	if retrySourceFingerprint(sourceChanged) == retrySourceFingerprint(snapshot) {
		t.Fatal("source input drift did not change retry fingerprint")
	}
}
