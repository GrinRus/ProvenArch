package orchestrator

import (
	"encoding/json"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/refreshplan"
)

func TestSelectiveBaselineRequiresFullIdentityAndDigest(t *testing.T) {
	ws := createWorkspace(t)
	baselineRunID := "run-baseline"
	currentRunID := "run-current"
	domainID := "bank"
	plan := runtimeShardPlan{
		ShardID:    "bank",
		RepoScopes: []string{"bank-of-anthos"},
		PathScopes: []string{"."},
	}
	baselineRevision := strings.Repeat("1", 40)
	currentRevision := strings.Repeat("2", 40)
	writeSourceRevisionsFixture(t, ws.WriteFileAtomic, baselineRunID, nil, baselineRevision)
	writeSourceRevisionsFixture(t, ws.WriteFileAtomic, currentRunID, &baselineRevision, currentRevision)
	writeShardFixture(t, ws.WriteFileAtomic, baselineRunID, domainID, plan)

	baselineExecution := &pipelineExecution{runID: baselineRunID, workspace: ws}
	if err := baselineExecution.writeShardBaselineIntegrity("init.step1.collect", domainID, plan, nil); err != nil {
		t.Fatalf("write baseline integrity: %v", err)
	}
	currentExecution := &pipelineExecution{
		runID:     currentRunID,
		workspace: ws,
		refreshExecution: &refreshplan.RefreshExecution{
			SourceRanges: []refreshplan.SourceRange{{
				Repo:             "bank-of-anthos",
				BaselineRevision: &baselineRevision,
				CurrentRevision:  &currentRevision,
			}},
		},
	}
	if err := currentExecution.copyBaselineShard(baselineRunID, "refresh.step1.collect", domainID, plan); err != nil {
		t.Fatalf("copy verified baseline: %v", err)
	}
	manifestRaw, err := ws.ReadFile(path.Join(runtimeShardArtifactRoot(currentRunID, plan.ShardID), shardPackManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := contracts.ParseShardPackManifest(manifestRaw)
	if err != nil || manifest.RunID != currentRunID || manifest.StepID != "refresh.step1.collect" {
		t.Fatalf("preserved manifest identity was not rebound: manifest=%+v err=%v", manifest, err)
	}

	if err := ws.WriteFileAtomic(path.Join(runtimeShardArtifactRoot(baselineRunID, plan.ShardID), "overview.md"), []byte("tampered")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := currentExecution.validateBaselineShard(baselineRunID, "refresh.step1.collect", domainID, plan); err == nil || !strings.Contains(err.Error(), "digest inventory") {
		t.Fatalf("expected digest mismatch, got %v", err)
	}

	conflictingPlan := plan
	conflictingPlan.PathScopes = []string{"src"}
	if _, _, err := currentExecution.validateBaselineShard(baselineRunID, "refresh.step1.collect", domainID, conflictingPlan); err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("expected identity mismatch, got %v", err)
	}
}

func TestBaselineDocumentReadUsesImmutableStagedSnapshot(t *testing.T) {
	ws := createWorkspace(t)
	execution := &pipelineExecution{workspace: ws}
	staged := "reports/taskruns/run-baseline/staging/final/reports/as-is/bank/overview.md"
	if err := ws.WriteFileAtomic(staged, []byte("baseline")); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFileAtomic("reports/as-is/bank/overview.md", []byte("user edit")); err != nil {
		t.Fatal(err)
	}
	raw, err := execution.readBaselineStagedDocument("run-baseline", contracts.FinalRunDocument{StagedPath: staged})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "baseline" {
		t.Fatalf("expected immutable baseline bytes, got %q", raw)
	}
	if _, err := execution.readBaselineStagedDocument("run-baseline", contracts.FinalRunDocument{StagedPath: "reports/as-is/bank/overview.md"}); err == nil {
		t.Fatal("expected non-baseline staged path to be rejected")
	}
}

func writeSourceRevisionsFixture(
	t *testing.T,
	write func(string, []byte) error,
	runID string,
	baselineRevision *string,
	currentRevision string,
) {
	t.Helper()
	revisions := refreshplan.SourceRevisions{
		Version:                  1,
		RunID:                    runID,
		Pipeline:                 "refresh",
		CapturedAt:               time.Now().UTC().Format(time.RFC3339),
		AnalysisInputFingerprint: strings.Repeat("a", 64),
		AnalysisInputsState:      "complete",
		InputIssues:              []string{},
		Repos: []refreshplan.RepoRevision{{
			Name:             "bank-of-anthos",
			SourceKind:       "path",
			Path:             "/tmp/bank",
			ConfiguredRef:    "HEAD",
			CurrentRevision:  &currentRevision,
			BaselineRevision: baselineRevision,
			WorktreeState:    "clean",
			EffectiveInclude: []string{"."},
			EffectiveExclude: []string{},
			Comparison: func() string {
				if baselineRevision == nil {
					return "initial"
				}
				return "ahead"
			}(),
			FallbackReasons: []string{},
		}},
	}
	raw, err := refreshplan.MarshalSourceRevisions(revisions)
	if err != nil {
		t.Fatal(err)
	}
	if err := write(path.Join("reports", "taskruns", runID, "source-revisions.json"), raw); err != nil {
		t.Fatal(err)
	}
}

func writeShardFixture(
	t *testing.T,
	write func(string, []byte) error,
	runID string,
	domainID string,
	plan runtimeShardPlan,
) {
	t.Helper()
	root := runtimeShardArtifactRoot(runID, plan.ShardID)
	manifest := contracts.ShardPackManifest{
		Version:      1,
		RunID:        runID,
		StepID:       "init.step1.collect",
		ShardID:      plan.ShardID,
		DomainID:     domainID,
		AgentRole:    "shard-analyst",
		ArtifactRoot: root,
		RepoScopes:   plan.RepoScopes,
		PathScopes:   plan.PathScopes,
		Documents: []contracts.AuthoredDocument{{
			ID: "doc.bank", Kind: "report", Title: "Bank", Path: "overview.md",
			CanonicalPath: "reports/as-is/bank/overview.md", Topics: []string{"bank"},
			CitationIDs: []string{"cite.bank"},
		}},
		Citations: []contracts.DocumentCitation{{
			ID: "cite.bank", Repo: "bank-of-anthos", Path: "README.md",
			ClaimIDs: []string{"claim.bank"}, DocumentIDs: []string{"doc.bank"},
		}},
		Semantic: contracts.SemanticSnapshot{
			Coverage:  contracts.Coverage{Observed: []string{"README"}, Missing: []string{}},
			Questions: []contracts.Question{},
			Entities:  []contracts.Entity{},
			Edges:     []contracts.Edge{},
			Findings:  []contracts.Finding{},
		},
	}
	execution := contracts.RuntimeExecution{
		Version: 1, TaskID: "task-bank", RunID: runID, StepID: "init.step1.collect",
		ShardID: plan.ShardID, DomainID: domainID, Provider: "fake",
		StartedAt: time.Now().UTC().Format(time.RFC3339), FinishedAt: time.Now().UTC().Format(time.RFC3339),
		RepoScopes: plan.RepoScopes, PathScopes: plan.PathScopes, ArtifactRoot: root,
		Status: "succeeded", RequiredArtifacts: []string{shardPackManifestFile},
	}
	for name, value := range map[string]any{shardPackManifestFile: manifest, runtimeExecutionFile: execution} {
		raw, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := write(path.Join(root, name), append(raw, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	if err := write(path.Join(root, "overview.md"), []byte("# Bank\n")); err != nil {
		t.Fatal(err)
	}
}
