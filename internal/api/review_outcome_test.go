package api

import (
	"testing"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
)

func TestRecoveryTaxonomyKeepsUnknownRetryDisabled(t *testing.T) {
	category, _, _, canRetry := classifyRecovery("unexpected_failure")
	if category != "infrastructure" || canRetry {
		t.Fatalf("unknown recovery must be conservative: category=%s can_retry=%v", category, canRetry)
	}
	category, _, _, canRetry = classifyRecovery("artifact_quality.evidence_missing")
	if category != "evidence" || !canRetry {
		t.Fatalf("evidence recovery was not classified: category=%s can_retry=%v", category, canRetry)
	}
}

func TestRunScopeCountsUseStructuredDomainIDs(t *testing.T) {
	logs := []orchestrator.RunLogEntry{
		{Level: orchestrator.RunLogLevelWarning, DomainID: "payments", Message: "partial evidence"},
		{Level: orchestrator.RunLogLevelWarning, DomainID: "payments", Message: "same scope"},
		{Level: orchestrator.RunLogLevelError, DomainID: "identity", Message: "runtime task failed"},
	}
	partial, failed := runScopeCounts(logs)
	if partial != 1 || failed != 1 {
		t.Fatalf("scope counts = partial %d failed %d", partial, failed)
	}
	if scopes := failedScopesFromLogs(logs); len(scopes) != 1 || scopes[0] != "identity" {
		t.Fatalf("failed scopes = %#v", scopes)
	}
}
