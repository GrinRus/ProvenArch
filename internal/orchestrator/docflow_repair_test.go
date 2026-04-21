package orchestrator

import (
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
	if err := execution.repairValidatorScopedArtifacts(&verdict); err != nil {
		t.Fatalf("repair validator scoped artifacts: %v", err)
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

	verdictRaw, err := os.ReadFile(filepath.Join(root, runtimeValidatorVerdictPath("run_bank_duplicate_claim")))
	if err != nil {
		t.Fatalf("read repaired validator verdict: %v", err)
	}
	if _, err := contracts.ParseValidatorVerdict(verdictRaw); err != nil {
		t.Fatalf("repaired validator verdict must stay parseable: %v", err)
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

	if err := execution.repairValidatorScopedArtifacts(&verdict); err != nil {
		t.Fatalf("repair validator scoped artifacts: %v", err)
	}
	if verdict.Verdict != "PASS" {
		t.Fatalf("expected repaired duplicate-claim verdict to become PASS, got %q", verdict.Verdict)
	}
	if len(verdict.Issues) != 0 {
		t.Fatalf("expected repaired duplicate-claim issues to be removed, got %#v", verdict.Issues)
	}
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
