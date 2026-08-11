package orchestrator

import (
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/artifactaudit"
	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func TestBuildEffectiveVerdictProviderFailCannotVetoCleanSnapshot(t *testing.T) {
	candidate := contracts.ValidatorVerdict{
		Version: 1, RunID: "run-1", GeneratedAt: "2026-08-11T10:00:00Z", Verdict: "FAIL",
		CheckedPaths: []string{"reports/taskruns/run-1/staging/final/final-run-index.json"},
		Issues:       []contracts.ValidatorIssue{{Code: "provider.source_observation", Severity: "error", Message: "source observation"}},
	}
	effective, err := buildEffectiveVerdict("run-1", candidate, []byte(`{"provider":true}`), nil, artifactaudit.Report{Version: 1, RunID: "run-1", Status: artifactaudit.StatusPass, Issues: []artifactaudit.Issue{}, Summary: artifactaudit.Summary{}}, time.Date(2026, 8, 11, 10, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if effective.Verdict != "PASS" {
		t.Fatalf("provider FAIL must not veto clean deterministic snapshot: %+v", effective)
	}
	if len(effective.AdvisoryIssues) != 1 || effective.AdvisoryIssues[0].Severity != "warning" {
		t.Fatalf("provider issue should remain advisory: %+v", effective.AdvisoryIssues)
	}
}

func TestBuildEffectiveVerdictDeterministicErrorWinsProviderPass(t *testing.T) {
	candidate := contracts.ValidatorVerdict{
		Version: 1, RunID: "run-1", GeneratedAt: "2026-08-11T10:00:00Z", Verdict: "PASS",
		CheckedPaths: []string{"reports/taskruns/run-1/staging/final/final-run-index.json"},
	}
	effective, err := buildEffectiveVerdict("run-1", candidate, []byte(`{"provider":true}`), []contracts.ValidatorIssue{{Code: "semantic_id_collision", Severity: "error", Message: "conflict"}}, artifactaudit.Report{Version: 1, RunID: "run-1", Status: artifactaudit.StatusPass, Issues: []artifactaudit.Issue{}, Summary: artifactaudit.Summary{}}, time.Date(2026, 8, 11, 10, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if effective.Verdict != "FAIL" || len(effective.TechnicalIssues) != 1 || effective.TechnicalIssues[0].Code != "semantic_id_collision" {
		t.Fatalf("deterministic error must own effective verdict: %+v", effective)
	}
}
