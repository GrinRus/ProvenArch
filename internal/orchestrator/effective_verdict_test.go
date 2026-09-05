package orchestrator

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestBuildEffectiveVerdictKeepsBankSourceObservationsAdvisory(t *testing.T) {
	t.Parallel()

	_, candidate := loadEvidenceAdvisoryFixture(t)
	providerRaw, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	audit := artifactaudit.Report{Version: 1, RunID: candidate.RunID, Status: artifactaudit.StatusPass}
	effective, err := buildEffectiveVerdict(candidate.RunID, candidate, providerRaw, nil, audit, time.Date(2026, 8, 11, 10, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if effective.Verdict != "PASS" || len(effective.TechnicalIssues) != 0 {
		t.Fatalf("source observations must not veto clean deterministic evidence: %+v", effective)
	}
	if len(effective.AdvisoryIssues) != len(candidate.Issues) {
		t.Fatalf("all source observations must remain visible: %+v", effective.AdvisoryIssues)
	}
	for _, original := range candidate.Issues {
		found := false
		for _, advisory := range effective.AdvisoryIssues {
			if advisory.Code != original.Code {
				continue
			}
			found = true
			if advisory.Source != "provider" || advisory.Severity != "warning" || advisory.Path != original.Path || advisory.CitationID != original.CitationID || advisory.Message != original.Message {
				t.Fatalf("source observation lost its evidence identity: original=%+v advisory=%+v", original, advisory)
			}
		}
		if !found {
			t.Fatalf("source observation %q was dropped", original.Code)
		}
	}
	after, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(providerRaw) || candidate.Verdict != "FAIL" {
		t.Fatalf("effective verdict construction must preserve the provider FAIL evidence: %s", after)
	}
}

func TestBuildEffectiveVerdictUsesIssueAuthorityForBankEvidenceBoundaries(t *testing.T) {
	t.Parallel()

	_, fixture := loadEvidenceAdvisoryFixture(t)
	for _, scenario := range []struct {
		name   string
		change func(*contracts.ValidatorIssue)
	}{
		{"staged_target", func(issue *contracts.ValidatorIssue) {
			issue.Path = "reports/taskruns/run_bank_refresh/staging/final/final-run-index.json"
		}},
		{"citation_mismatch", func(issue *contracts.ValidatorIssue) { issue.CitationID = "cite.iac.acm.jwt" }},
		{"missing_path_and_citation", func(issue *contracts.ValidatorIssue) { issue.Path, issue.CitationID = "", "" }},
		{"missing_citation", func(issue *contracts.ValidatorIssue) { issue.CitationID = "" }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			issue := fixture.Issues[0]
			scenario.change(&issue)
			candidate := fixture
			candidate.Issues = []contracts.ValidatorIssue{issue}
			providerRaw, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			for _, authoritative := range []bool{false, true} {
				var deterministic []contracts.ValidatorIssue
				if authoritative {
					deterministic = []contracts.ValidatorIssue{issue}
				}
				audit := artifactaudit.Report{Version: 1, RunID: candidate.RunID, Status: artifactaudit.StatusPass}
				effective, err := buildEffectiveVerdict(candidate.RunID, candidate, providerRaw, deterministic, audit, time.Date(2026, 8, 11, 10, 0, 1, 0, time.UTC))
				if err != nil {
					t.Fatal(err)
				}
				if authoritative {
					if effective.Verdict != "FAIL" || len(effective.TechnicalIssues) != 1 || len(effective.AdvisoryIssues) != 0 {
						t.Fatalf("deterministic evidence error must remain blocking without an advisory duplicate: %+v", effective)
					}
					if effective.TechnicalIssues[0] != issue {
						t.Fatalf("deterministic issue was changed: got %+v want %+v", effective.TechnicalIssues[0], issue)
					}
				} else if effective.Verdict != "PASS" || len(effective.TechnicalIssues) != 0 || len(effective.AdvisoryIssues) != 1 || effective.AdvisoryIssues[0].Severity != "warning" {
					t.Fatalf("provider prose alone must not establish technical authority: %+v", effective)
				}
			}
		})
	}
}

func loadEvidenceAdvisoryFixture(t *testing.T) (contracts.CitationIndex, contracts.ValidatorVerdict) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", "bank_validator_evidence_advisory.json"))
	if err != nil {
		t.Fatalf("read evidence advisory fixture: %v", err)
	}
	var fixture struct {
		CitationIndex json.RawMessage `json:"citation_index"`
		Verdict       json.RawMessage `json:"verdict"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode evidence advisory fixture: %v", err)
	}
	citationIndex, err := contracts.ParseCitationIndex(fixture.CitationIndex)
	if err != nil {
		t.Fatalf("parse fixture citation index: %v", err)
	}
	verdict, err := contracts.ParseValidatorVerdict(fixture.Verdict)
	if err != nil {
		t.Fatalf("parse fixture validator verdict: %v", err)
	}
	return citationIndex, verdict
}
