package providercommon

import (
	"errors"
	"reflect"
	"sort"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestValidationIssueSetIsStableAcrossProvidersAndMessageOrder(t *testing.T) {
	problems := []string{
		`documents[0].citation_ids references missing citation`,
		`semantic/questions/0/text is required`,
		`citations[1].id must be unique`,
	}
	reversed := append([]string(nil), problems...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	expected := []string{
		issueCollectCitationReference,
		issueCollectDuplicateCitation,
		issueCollectQuestionText,
	}
	sort.Strings(expected)

	for _, provider := range []acpruntime.Provider{
		acpruntime.ProviderClaudeCode,
		acpruntime.ProviderQwenCode,
		acpruntime.ProviderCodexCode,
	} {
		for _, ordered := range [][]string{problems, reversed} {
			message := string(provider) + ": shard pack manifest is invalid: " +
				ordered[0] + "; " + ordered[1] + "; " + ordered[2]
			issues := classifyValidationIssues(errors.New(message))
			actual := []string{}
			for _, issue := range issues.Items {
				if issue.Code == issueCollectCitationReference ||
					issue.Code == issueCollectDuplicateCitation ||
					issue.Code == issueCollectQuestionText {
					actual = append(actual, issue.Code)
				}
			}
			sort.Strings(actual)
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("provider=%s order=%v issues=%v want=%v", provider, ordered, actual, expected)
			}
			if !shouldRetryCollectManifestShapeCleanup(errors.New(message)) {
				t.Fatalf("provider=%s order=%v did not select typed shape cleanup", provider, ordered)
			}
		}
	}
}

func TestTypedDraftRecoverySelectionIsBoundedByTransitionStage(t *testing.T) {
	err := errors.New("runtime draft manifest outputs are invalid: overview.md does not report exact current-run shard status")
	task := acpruntime.Task{StepID: "refresh.step2.asis_docs"}

	if !shouldRecoverDraftRepairValidationWithEnrichment(task, err) {
		t.Fatal("expected semantic draft issue to route to enrichment")
	}
	if !shouldRetryDraftShardStatusCleanupEnrichment("draft_artifact_enrichment", err) {
		t.Fatal("expected one shard-status cleanup transition")
	}
	if shouldRetryDraftShardStatusCleanupEnrichment("draft_artifact_enrichment_shard_status_cleanup", err) {
		t.Fatal("same recovery transition must have a one-attempt budget")
	}
}

func TestTypedRepoEvidenceIssuesRetainBoundedPaths(t *testing.T) {
	err := errors.New(`repo evidence path "src/b.go" is missing; repo evidence path "src/a.go" is missing; repo evidence path "src/a.go" is missing`)
	got := missingRepoEvidencePathsFromError(err)
	want := []string{"src/a.go", "src/b.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths=%v want=%v", got, want)
	}
}
