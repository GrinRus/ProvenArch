package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/orchestrator"
)

func TestArchitectureComparisonUsesSemanticFindingsAndGaps(t *testing.T) {
	root := t.TempDir()
	writeSemanticSnapshot := func(runID string, semantic contracts.SemanticSnapshot) {
		t.Helper()
		folder := filepath.Join(root, "reports", "taskruns", runID, "promoted-snapshot")
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(map[string]any{"version": 2, "run_id": runID, "files": []any{}, "semantic": semantic})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "architecture-snapshot.json"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	provenance := contracts.Provenance{Kind: "observation", Confidence: .8, Evidence: []contracts.Evidence{}}
	writeSemanticSnapshot("baseline", contracts.SemanticSnapshot{Coverage: contracts.Coverage{Missing: []string{"owner"}}, Questions: []contracts.Question{}, Entities: []contracts.Entity{}, Edges: []contracts.Edge{}, Findings: []contracts.Finding{{ID: "finding.old", Severity: "low", Title: "Old", Provenance: provenance}}})
	writeSemanticSnapshot("current", contracts.SemanticSnapshot{Coverage: contracts.Coverage{Missing: []string{"api contract"}}, Questions: []contracts.Question{}, Entities: []contracts.Entity{}, Edges: []contracts.Edge{}, Findings: []contracts.Finding{{ID: "finding.new", Severity: "high", Title: "New", Provenance: provenance}}})
	comparison := comparePromotedArchitectures(root, "current", "baseline")
	if got := comparison.Categories["findings"]; len(got.Added) != 1 || got.Added[0].ID != "finding.new" || len(got.Removed) != 1 || got.Removed[0].ID != "finding.old" {
		t.Fatalf("unexpected finding comparison: %#v", got)
	}
	if got := comparison.Categories["gaps"]; len(got.Added) != 1 || got.Added[0].Name != "api contract" || len(got.Removed) != 1 || got.Removed[0].Name != "owner" {
		t.Fatalf("unexpected gap comparison: %#v", got)
	}
}

func TestOutcomeWorkflowFixtureMatchesPublicShapes(t *testing.T) {
	for _, path := range []string{filepath.Join("..", "..", "fixtures", "api", "outcome-workflow.json"), filepath.Join("..", "..", "examples", "outcome-workflow.example.json")} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fixture struct {
			Architecture architectureResponse     `json:"architecture"`
			Progress     orchestrator.RunProgress `json:"progress"`
			RetryPlan    retryPlan                `json:"retry_plan"`
		}
		if err := json.Unmarshal(raw, &fixture); err != nil {
			t.Fatal(err)
		}
		if fixture.Architecture.Authority.Mode != evidenceAuthorityPromotedCurrent || fixture.Progress.Phase != "repairing" || fixture.RetryPlan.PlanHash == "" {
			t.Fatalf("outcome workflow example %s is incomplete: %#v", path, fixture)
		}
	}
}

func TestRunReviewContractFixtureMatchesPublicShape(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "api", "run-review-contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture runReviewContract
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode run review fixture: %v", err)
	}
	if fixture.ReviewKind != "refresh" || fixture.SourceRunID == "" || fixture.BaselineRunID == "" || !fixture.SemanticChanges.Available || !fixture.DocumentChanges.Available {
		t.Fatalf("run review fixture is incomplete: %#v", fixture)
	}
	if fixture.Authority.SourceRunID != fixture.SourceRunID || fixture.GeneratedAt == "" || len(fixture.Runtime.Providers) != 1 {
		t.Fatalf("run review fixture lost identity metadata: %#v", fixture)
	}
}

func TestRunReviewContractKeepsInitialRunsAsSummary(t *testing.T) {
	root := t.TempDir()
	writeSnapshot := func(runID string) {
		t.Helper()
		folder := filepath.Join(root, "reports", "taskruns", runID, "promoted-snapshot")
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatal(err)
		}
		raw := []byte(`{"version":2,"run_id":"` + runID + `","files":[],"semantic":{"coverage":{"missing":[]},"questions":[],"entities":[],"edges":[],"findings":[]}}`)
		if err := os.WriteFile(filepath.Join(folder, "architecture-snapshot.json"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeSnapshot("run-initial")
	writeSnapshot("run-previous")
	review := buildRunReviewContract(root, orchestrator.RunInfo{RunID: "run-initial", Pipeline: "init", RuntimeMode: "fake"}, "run-previous", "fake")
	if review.ReviewKind != "initial" || review.BaselineRunID != "" || review.SemanticChanges.Available {
		t.Fatalf("initial review must be a summary without a baseline delta: %#v", review)
	}
}
