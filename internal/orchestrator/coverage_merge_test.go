package orchestrator

import (
	"reflect"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func TestMergeCoverageCanonicalizesMissingTermsAndDedupes(t *testing.T) {
	t.Parallel()

	existing := &contracts.Coverage{
		Missing: []string{"owner_mappings", "ci_cd_evidence", "custom gap"},
	}
	incoming := &contracts.Coverage{
		Missing: []string{"owner mappings", "ci-cd evidence", "delta_validation", "dependency_graph", "runtime_metrics", "custom gap"},
	}

	merged := mergeCoverage(existing, incoming)
	if merged == nil {
		t.Fatalf("expected merged coverage")
	}
	expected := []string{
		"owner mappings",
		"ci-cd evidence",
		"custom gap",
		"delta validation",
		"dependency graph",
		"runtime metrics",
	}
	if !reflect.DeepEqual(merged.Missing, expected) {
		t.Fatalf("unexpected merged missing coverage:\nwant: %#v\ngot:  %#v", expected, merged.Missing)
	}
}

func TestCanonicalizeCoverageMissingValueKeepsUnknownTerms(t *testing.T) {
	t.Parallel()

	value := canonicalizeCoverageMissingValue("deployment configs")
	if value != "deployment configs" {
		t.Fatalf("expected canonical deployment term, got %q", value)
	}
}

func TestMergeCoverageCanonicalizesApiContractsAndDedupesNotesSemantically(t *testing.T) {
	t.Parallel()

	existing := &contracts.Coverage{
		Missing: []string{"api_contracts", "ci_cd_pipelines"},
		Notes:   []string{"Evidence gaps are captured explicitly"},
	}
	incoming := &contracts.Coverage{
		Missing: []string{"api contracts", "ci-cd evidence"},
		Notes:   []string{"evidence-gaps are captured explicitly"},
	}

	merged := mergeCoverage(existing, incoming)
	if merged == nil {
		t.Fatalf("expected merged coverage")
	}
	expectedMissing := []string{"api contracts", "ci-cd evidence"}
	if !reflect.DeepEqual(merged.Missing, expectedMissing) {
		t.Fatalf("unexpected merged missing terms:\nwant: %#v\ngot:  %#v", expectedMissing, merged.Missing)
	}
	if len(merged.Notes) != 1 {
		t.Fatalf("expected semantic note dedupe, got %#v", merged.Notes)
	}
}

func TestMergeQuestionsDedupesByCanonicalIDAndText(t *testing.T) {
	t.Parallel()

	existing := []contracts.Question{
		{ID: "q.refresh.delta", Text: "What changed since previous run that affects ownership or dependencies?"},
	}
	incoming := []contracts.Question{
		{ID: "q.refresh.delta.1", Text: "What changed since previous run that affects ownership or dependencies?"},
		{ID: "q.refresh.scope.1", Text: "Are there additional services in the current scenario not yet cataloged?"},
		{ID: "q.refresh.scope.2", Text: "Are there additional services in the current scenario not yet cataloged?"},
	}

	merged := mergeQuestions(existing, incoming)
	if len(merged) != 2 {
		t.Fatalf("expected 2 deduped questions, got %d: %#v", len(merged), merged)
	}
	if merged[0].ID != "q.refresh.delta" {
		t.Fatalf("unexpected first question id %q", merged[0].ID)
	}
	if merged[1].ID != "q.refresh.scope" {
		t.Fatalf("expected canonicalized scope question id, got %q", merged[1].ID)
	}
}
