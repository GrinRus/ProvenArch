package artifactquality

import (
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
