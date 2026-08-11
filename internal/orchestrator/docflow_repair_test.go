package orchestrator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestRepairDuplicateClaimIDsInCitationIndexUsesShardSuffixes(t *testing.T) {
	t.Parallel()

	finalIndex, citationIndex := loadDuplicateClaimFixture(t)
	changed := repairDuplicateClaimIDsInCitationIndex(&finalIndex, &citationIndex)
	if !changed {
		t.Fatalf("expected duplicate claim-id repair to modify citation index")
	}

	seen := map[string]struct{}{}
	for _, citation := range citationIndex.Citations {
		for _, claimID := range citation.ClaimIDs {
			if _, exists := seen[claimID]; exists {
				t.Fatalf("expected repaired claim ids to be unique, got duplicate %q in %#v", claimID, citationIndex.Citations)
			}
			seen[claimID] = struct{}{}
		}
	}

	secondCitation := citationIndex.Citations[1]
	if !containsStringValue(secondCitation.ClaimIDs, "claim.balancereader.deployment.bank-of-anthos-docs") {
		t.Fatalf("expected duplicate deployment claim id to be repaired with shard suffix, got %#v", secondCitation.ClaimIDs)
	}
	if !containsStringValue(secondCitation.ClaimIDs, "claim.balancereader.service.bank-of-anthos-docs") {
		t.Fatalf("expected duplicate service claim id to be repaired with shard suffix, got %#v", secondCitation.ClaimIDs)
	}
}

func TestRepairValidatorScopedArtifactsPersistsCitationRepairAndVerdictFixedPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	execution := pipelineExecution{
		runID: "run_bank_duplicate_claim",
		workspace: workspace.Root{
			Path: root,
		},
	}
	finalIndex, citationIndex := loadDuplicateClaimFixture(t)
	execution.finalRunIndex = &finalIndex
	execution.citationIndex = &citationIndex

	verdict := contracts.ValidatorVerdict{
		Version:      1,
		RunID:        "run_bank_duplicate_claim",
		GeneratedAt:  "2026-04-18T16:23:02Z",
		Verdict:      "PASS",
		Summary:      "validator passed",
		CheckedPaths: []string{runtimeCitationIndexPath("run_bank_duplicate_claim")},
	}
	result, err := execution.applyValidatorRepairStage("init.step3.findings", "payments", "task-1", &verdict)
	if err != nil {
		t.Fatalf("repair validator scoped artifacts: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected validator repair stage to report change")
	}
	if result.ResolvedIssues != 0 {
		t.Fatalf("expected no resolved issues for already-passing verdict, got %d", result.ResolvedIssues)
	}
	if !containsStringValue(verdict.FixedPaths, runtimeCitationIndexPath("run_bank_duplicate_claim")) {
		t.Fatalf("expected repaired verdict fixed_paths to include citation index, got %#v", verdict.FixedPaths)
	}
	if !strings.Contains(verdict.Summary, "deterministically repaired duplicate claim_ids") {
		t.Fatalf("expected repair note in verdict summary, got %q", verdict.Summary)
	}

	citationRaw, err := os.ReadFile(filepath.Join(root, runtimeCitationIndexPath("run_bank_duplicate_claim")))
	if err != nil {
		t.Fatalf("read repaired citation index: %v", err)
	}
	repairedCitationIndex, err := contracts.ParseCitationIndex(citationRaw)
	if err != nil {
		t.Fatalf("parse repaired citation index: %v", err)
	}
	if !containsStringValue(repairedCitationIndex.Citations[1].ClaimIDs, "claim.balancereader.deployment.bank-of-anthos-docs") {
		t.Fatalf("expected repaired citation index to persist renamed claim id, got %#v", repairedCitationIndex.Citations[1].ClaimIDs)
	}

	if _, err := os.Stat(filepath.Join(root, runtimeValidatorVerdictPath("run_bank_duplicate_claim"))); !os.IsNotExist(err) {
		t.Fatalf("provider validator verdict must remain untouched, stat err=%v", err)
	}
}

func TestRepairValidatorScopedArtifactsPromotesFailingDuplicateClaimVerdictToPass(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	execution := pipelineExecution{
		runID: "run_bank_duplicate_claim",
		workspace: workspace.Root{
			Path: root,
		},
	}
	finalIndex, citationIndex := loadDuplicateClaimFixture(t)
	execution.finalRunIndex = &finalIndex
	execution.citationIndex = &citationIndex

	verdict := contracts.ValidatorVerdict{
		Version:      1,
		RunID:        "run_bank_duplicate_claim",
		GeneratedAt:  "2026-04-18T16:23:02Z",
		Verdict:      "FAIL",
		Summary:      "duplicate claim ids detected",
		CheckedPaths: []string{runtimeCitationIndexPath("run_bank_duplicate_claim")},
		Issues: []contracts.ValidatorIssue{
			{
				Code:       "duplicate_claim_id",
				Severity:   "error",
				Message:    `duplicate claim id "claim.balancereader.deployment"`,
				CitationID: "cite.bank-of-anthos-docs",
			},
		},
	}

	result, err := execution.applyValidatorRepairStage("init.step3.findings", "payments", "task-1", &verdict)
	if err != nil {
		t.Fatalf("repair validator scoped artifacts: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected validator repair stage to report change")
	}
	if result.ResolvedIssues != 1 {
		t.Fatalf("expected one resolved duplicate-claim issue, got %d", result.ResolvedIssues)
	}
	if verdict.Verdict != "PASS" {
		t.Fatalf("expected repaired duplicate-claim verdict to become PASS, got %q", verdict.Verdict)
	}
	if len(verdict.Issues) != 0 {
		t.Fatalf("expected repaired duplicate-claim issues to be removed, got %#v", verdict.Issues)
	}
}

func TestRepairValidatorScopedArtifactsDoesNotMutateStateWhenWriteFails(t *testing.T) {
	t.Parallel()

	rootFile, err := os.CreateTemp(t.TempDir(), "workspace-root-file-*")
	if err != nil {
		t.Fatalf("create temp root file: %v", err)
	}
	if err := rootFile.Close(); err != nil {
		t.Fatalf("close temp root file: %v", err)
	}

	execution := pipelineExecution{
		runID: "run_bank_duplicate_claim",
		workspace: workspace.Root{
			Path: rootFile.Name(),
		},
	}
	finalIndex, citationIndex := loadDuplicateClaimFixture(t)
	execution.finalRunIndex = &finalIndex
	execution.citationIndex = &citationIndex

	originalCitationRaw, err := json.Marshal(execution.citationIndex)
	if err != nil {
		t.Fatalf("marshal original citation index: %v", err)
	}

	verdict := contracts.ValidatorVerdict{
		Version:      1,
		RunID:        "run_bank_duplicate_claim",
		GeneratedAt:  "2026-04-18T16:23:02Z",
		Verdict:      "FAIL",
		Summary:      "duplicate claim ids detected",
		CheckedPaths: []string{runtimeCitationIndexPath("run_bank_duplicate_claim")},
		Issues: []contracts.ValidatorIssue{
			{
				Code:       "duplicate_claim_id",
				Severity:   "error",
				Message:    `duplicate claim id "claim.balancereader.deployment"`,
				CitationID: "cite.bank-of-anthos-docs",
			},
		},
	}
	originalVerdictRaw, err := json.Marshal(verdict)
	if err != nil {
		t.Fatalf("marshal original verdict: %v", err)
	}

	_, err = execution.applyValidatorRepairStage("init.step3.findings", "payments", "task-1", &verdict)
	if err == nil {
		t.Fatalf("expected validator repair stage to fail when workspace root is not writable")
	}

	citationAfterRaw, err := json.Marshal(execution.citationIndex)
	if err != nil {
		t.Fatalf("marshal citation index after failed repair: %v", err)
	}
	if string(citationAfterRaw) != string(originalCitationRaw) {
		t.Fatalf("expected citation index to stay unchanged after failed repair")
	}

	verdictAfterRaw, err := json.Marshal(verdict)
	if err != nil {
		t.Fatalf("marshal verdict after failed repair: %v", err)
	}
	if string(verdictAfterRaw) != string(originalVerdictRaw) {
		t.Fatalf("expected verdict to stay unchanged after failed repair")
	}
}

func TestReconcileEvidenceAdvisoryOnlyVerdictPinsBankRefreshFailure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	citationIndex, verdict := loadEvidenceAdvisoryFixture(t)
	execution := pipelineExecution{
		runID: verdict.RunID,
		workspace: workspace.Root{
			Path: root,
		},
	}
	execution.citationIndex = &citationIndex

	reconciled, err := execution.reconcileEvidenceAdvisoryOnlyVerdict(&verdict)
	if err != nil {
		t.Fatalf("reconcile evidence advisory verdict: %v", err)
	}
	if !reconciled {
		t.Fatalf("expected live-observed source-evidence-only verdict to reconcile")
	}
	if verdict.Verdict != "PASS" {
		t.Fatalf("expected reconciled verdict PASS, got %q", verdict.Verdict)
	}
	for _, issue := range verdict.Issues {
		if issue.Severity != "warning" {
			t.Fatalf("expected evidence issue %q to remain visible as warning, got %#v", issue.Code, issue)
		}
	}
	if !strings.Contains(verdict.Summary, "source-evidence observations remain advisory") {
		t.Fatalf("expected reconciliation note, got %q", verdict.Summary)
	}

	if _, err := os.Stat(filepath.Join(root, runtimeValidatorVerdictPath(verdict.RunID))); !os.IsNotExist(err) {
		t.Fatalf("provider verdict must remain untouched, stat err=%v", err)
	}
}

func TestEvidenceAdvisoryIssuesFailClosedForStagedTargetOrCitationMismatch(t *testing.T) {
	t.Parallel()

	citationIndex, verdict := loadEvidenceAdvisoryFixture(t)

	stagedIssue := verdict.Issues[0]
	stagedIssue.Path = "reports/taskruns/run_bank_refresh/staging/final/final-run-index.json"
	if _, ok := evidenceAdvisoryIssues([]contracts.ValidatorIssue{stagedIssue}, citationIndex.Citations); ok {
		t.Fatalf("expected staged artifact issue to remain blocking")
	}

	mismatchedCitation := verdict.Issues[0]
	mismatchedCitation.CitationID = "cite.iac.acm.jwt"
	if _, ok := evidenceAdvisoryIssues([]contracts.ValidatorIssue{mismatchedCitation}, citationIndex.Citations); ok {
		t.Fatalf("expected citation/path mismatch to remain blocking")
	}

	pathlessIssue := verdict.Issues[0]
	pathlessIssue.Path = ""
	pathlessIssue.CitationID = ""
	if _, ok := evidenceAdvisoryIssues([]contracts.ValidatorIssue{pathlessIssue}, citationIndex.Citations); ok {
		t.Fatalf("expected pathless error issue to remain blocking")
	}

	pathOnlyIssue := verdict.Issues[0]
	pathOnlyIssue.CitationID = ""
	if _, ok := evidenceAdvisoryIssues([]contracts.ValidatorIssue{pathOnlyIssue}, citationIndex.Citations); ok {
		t.Fatalf("expected error issue without exact citation identity to remain blocking")
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

func loadDuplicateClaimFixture(t *testing.T) (contracts.FinalRunIndex, contracts.CitationIndex) {
	t.Helper()

	root := filepath.Join("..", "..", "fixtures", "scenarios", "validator-duplicate-claim", "bank-balance-reader")
	finalRaw, err := os.ReadFile(filepath.Join(root, "final-run-index.json"))
	if err != nil {
		t.Fatalf("read final-run-index fixture: %v", err)
	}
	finalIndex, err := contracts.ParseFinalRunIndex(finalRaw)
	if err != nil {
		t.Fatalf("parse final-run-index fixture: %v", err)
	}

	citationRaw, err := os.ReadFile(filepath.Join(root, "citation-index.json"))
	if err != nil {
		t.Fatalf("read citation-index fixture: %v", err)
	}
	citationIndex, err := contracts.ParseCitationIndex(citationRaw)
	if err != nil {
		t.Fatalf("parse citation-index fixture: %v", err)
	}

	return finalIndex, citationIndex
}

func containsStringValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
