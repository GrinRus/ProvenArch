package artifactquality

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func TestValidateValidatorVerdictRejectsContradictoryPassAndProviderFixedPaths(t *testing.T) {
	base := contracts.ValidatorVerdict{
		Verdict:    "PASS",
		FixedPaths: []string{"reports/taskruns/run-1/staging/final/citation-index.json"},
		Issues:     []contracts.ValidatorIssue{{Code: "technical", Severity: "error", Message: "broken"}},
	}
	err := ValidateValidatorVerdict(base, nil, nil, false, false)
	if err == nil || !strings.Contains(err.Error(), "PASS verdict cannot contain error issues") || !strings.Contains(err.Error(), "fixed_paths") {
		t.Fatalf("expected contradictory provider draft to fail, got %v", err)
	}
}

func TestValidateValidatorVerdictRequiresTechnicalFailureForEffectiveVerdict(t *testing.T) {
	verdict := contracts.ValidatorVerdict{Verdict: "FAIL", Findings: []contracts.Finding{{ID: "finding.owner", Title: "Owner gap"}}}
	if err := ValidateValidatorVerdict(verdict, nil, nil, false, true); err != nil {
		t.Fatalf("draft advisory FAIL should remain available before reconciliation: %v", err)
	}
	if err := ValidateValidatorVerdict(verdict, nil, nil, true, true); err == nil {
		t.Fatal("expected effective FAIL without technical issue to fail")
	}
}

func TestValidateValidatorVerdictRejectsDuplicateUnorderedAndDanglingIssues(t *testing.T) {
	finalIndex := &contracts.FinalRunIndex{CanonicalDocuments: []contracts.FinalRunDocument{{ID: "doc.home", CanonicalPath: "reports/as-is/overview.md", StagedPath: "reports/taskruns/run-1/staging/final/reports/as-is/overview.md"}}}
	citations := &contracts.CitationIndex{Citations: []contracts.DocumentCitation{{ID: "cite.home", Path: "README.md"}}}
	verdict := contracts.ValidatorVerdict{
		Verdict: "FAIL",
		Issues: []contracts.ValidatorIssue{
			{Code: "zeta", Severity: "error", Message: "z", DocumentID: "doc.home", CitationID: "cite.home", Path: "README.md"},
			{Code: "alpha", Severity: "error", Message: "a", DocumentID: "doc.missing"},
			{Code: "alpha", Severity: "error", Message: "a", DocumentID: "doc.missing"},
		},
	}
	err := ValidateValidatorVerdict(verdict, finalIndex, citations, true, true)
	for _, marker := range []string{"duplicates issue identity", "deterministic", "document_id \"doc.missing\""} {
		if err == nil || !strings.Contains(err.Error(), marker) {
			t.Fatalf("expected %q in consistency error, got %v", marker, err)
		}
	}
}

func TestValidateValidatorVerdictAcceptsCurrentRunIndexPaths(t *testing.T) {
	finalIndex := &contracts.FinalRunIndex{
		RunID:             "run-1",
		CitationIndexPath: "reports/taskruns/run-1/staging/final/citation-index.json",
	}
	citationIndex := &contracts.CitationIndex{RunID: "run-1"}
	verdict := contracts.ValidatorVerdict{
		RunID: "run-1",
		Issues: []contracts.ValidatorIssue{
			{Code: "citation.missing", Severity: "warning", Message: "citation index drift", Path: "reports/taskruns/run-1/staging/final/citation-index.json"},
			{Code: "index.missing", Severity: "error", Message: "final index drift", Path: "reports/taskruns/run-1/staging/final/final-run-index.json"},
		},
	}
	if err := ValidateValidatorVerdict(verdict, finalIndex, citationIndex, true, false); err != nil {
		t.Fatalf("current-run validator paths should be in inventory: %v", err)
	}
}

func TestNormalizeProviderValidatorPathsConvertsContainedAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	finalPath := filepath.Join(root, "reports", "taskruns", "run-1", "staging", "final", "final-run-index.json")
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(finalPath, []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	verdict := contracts.ValidatorVerdict{
		Verdict:      "PASS",
		CheckedPaths: []string{finalPath},
		Issues:       []contracts.ValidatorIssue{{Code: "index.naming", Severity: "warning", Message: "naming drift", Path: finalPath}},
	}
	if err := NormalizeProviderValidatorPaths(&verdict, root); err != nil {
		t.Fatalf("normalize contained absolute paths: %v", err)
	}
	want := "reports/taskruns/run-1/staging/final/final-run-index.json"
	if verdict.CheckedPaths[0] != want || verdict.Issues[0].Path != want {
		t.Fatalf("normalized paths = %#v / %q, want %q", verdict.CheckedPaths, verdict.Issues[0].Path, want)
	}
	finalIndex := &contracts.FinalRunIndex{CanonicalDocuments: []contracts.FinalRunDocument{{CanonicalPath: "reports/as-is/overview.md", StagedPath: want}}}
	if err := ValidateValidatorVerdict(verdict, finalIndex, nil, false, false); err != nil {
		t.Fatalf("normalized provider verdict should pass inventory validation: %v", err)
	}
}

func TestNormalizeProviderValidatorPathsRejectsForeignAbsolutePaths(t *testing.T) {
	foreignRoot := t.TempDir()
	foreignPath := filepath.Join(foreignRoot, "foreign.json")
	if err := os.WriteFile(foreignPath, []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}
	verdict := contracts.ValidatorVerdict{Issues: []contracts.ValidatorIssue{{Path: foreignPath}}}
	if err := NormalizeProviderValidatorPaths(&verdict, t.TempDir()); err == nil || !strings.Contains(err.Error(), "outside the selected workspace") {
		t.Fatalf("expected foreign absolute path rejection, got %v", err)
	}
}
