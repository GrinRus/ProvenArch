package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseEffectiveVerdictExample(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "effective-verdict.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	verdict, err := ParseEffectiveVerdict(raw)
	if err != nil {
		t.Fatalf("parse effective verdict example: %v", err)
	}
	if verdict.Authority != "orchestrator" || verdict.Kind != "effective" || verdict.Verdict != "PASS" {
		t.Fatalf("unexpected effective verdict: %+v", verdict)
	}
}

func TestParseEffectiveVerdictRejectsProviderAuthorityAndPassErrors(t *testing.T) {
	base := EffectiveVerdict{
		Version: 1, Kind: "effective", Authority: "orchestrator", RunID: "run-1",
		GeneratedAt: "2026-04-16T12:00:03Z", ProviderVerdictPath: "reports/taskruns/run-1/validator/validator-verdict.json",
		ProviderVerdictSHA256: "0000000000000000000000000000000000000000000000000000000000000000",
		Verdict:               "PASS", CheckedPaths: []string{"reports/taskruns/run-1/staging/final/final-run-index.json"},
		FixedPaths: []string{}, TechnicalIssues: []ValidatorIssue{}, AdvisoryIssues: []AdvisoryValidatorIssue{},
		Audit: EffectiveAuditSummary{Status: "pass", IssueCodes: []string{}},
	}
	raw, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseEffectiveVerdict(raw); err != nil {
		t.Fatalf("expected valid effective verdict, got %v", err)
	}
	base.Authority = "provider"
	raw, _ = json.Marshal(base)
	if _, err := ParseEffectiveVerdict(raw); err == nil {
		t.Fatal("expected provider authority to be rejected")
	}
	base.Authority = "orchestrator"
	base.TechnicalIssues = []ValidatorIssue{{Code: "audit.bad", Severity: "error", Message: "bad"}}
	raw, _ = json.Marshal(base)
	if _, err := ParseEffectiveVerdict(raw); err == nil {
		t.Fatal("expected PASS with technical error to be rejected")
	}
}

func TestParseEffectiveVerdictPreservesAdvisoryProviderIssue(t *testing.T) {
	verdict := EffectiveVerdict{
		Version: 1, Kind: "effective", Authority: "orchestrator", RunID: "run-1",
		GeneratedAt: "2026-04-16T12:00:03Z", ProviderVerdictPath: "reports/taskruns/run-1/validator/validator-verdict.json",
		ProviderVerdictSHA256: "0000000000000000000000000000000000000000000000000000000000000000",
		Verdict:               "PASS", CheckedPaths: []string{"reports/taskruns/run-1/staging/final/final-run-index.json"},
		FixedPaths: []string{}, TechnicalIssues: []ValidatorIssue{},
		AdvisoryIssues: []AdvisoryValidatorIssue{{Source: "provider", MatchKey: "provider.issue::::", Code: "provider.observation", Severity: "warning", Message: "advisory"}},
		Audit:          EffectiveAuditSummary{Status: "warn", WarningCount: 1, IssueCodes: []string{"audit.proposal.not_actionable"}},
	}
	raw, err := json.Marshal(verdict)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseEffectiveVerdict(raw)
	if err != nil {
		t.Fatalf("parse advisory effective verdict: %v", err)
	}
	if len(parsed.AdvisoryIssues) != 1 || parsed.AdvisoryIssues[0].Severity != "warning" {
		t.Fatalf("unexpected advisory issues: %+v", parsed.AdvisoryIssues)
	}
}
