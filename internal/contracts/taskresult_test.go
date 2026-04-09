package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTaskResultFromFixture(t *testing.T) {
	t.Parallel()

	raw := readFixtureFile(t, "../../fixtures/taskresult/normalized-top-level.json")
	result, err := ParseTaskResult(raw)
	if err != nil {
		t.Fatalf("parse taskresult: %v", err)
	}
	if strings.TrimSpace(result.Meta.Runtime.Name) == "" {
		t.Fatalf("expected non-empty runtime.name")
	}
	if len(result.Changeset) == 0 {
		t.Fatalf("expected non-empty changeset")
	}
}

func TestNormalizeTaskResultMergesLegacyQuestionCoverageForms(t *testing.T) {
	t.Parallel()

	raw := readFixtureFile(t, "../../fixtures/taskresult/mixed-legacy.json")
	result, err := ParseTaskResult(raw)
	if err != nil {
		t.Fatalf("parse taskresult: %v", err)
	}

	normalized := NormalizeTaskResult(result)
	for _, op := range normalized.Changeset {
		if op.Op == "add_question" || op.Op == "set_coverage" {
			t.Fatalf("expected legacy ops to be removed from changeset, got %q", op.Op)
		}
	}
	if len(normalized.Questions) != 2 {
		t.Fatalf("expected 2 deduplicated questions, got %d", len(normalized.Questions))
	}
	if normalized.Coverage == nil {
		t.Fatalf("expected merged coverage")
	}
	if !contains(normalized.Coverage.Missing, "owners") || !contains(normalized.Coverage.Missing, "ci-cd") {
		t.Fatalf("expected merged coverage.missing, got %+v", normalized.Coverage.Missing)
	}
}

func TestParseTaskResultRejectsObservationWithoutEvidence(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"meta":{"task_id":"t1","step_id":"init.step1.collect","runtime":{"name":"claude-code"},"started_at":"2026-04-03T10:00:00Z"},
		"summary":"bad",
		"changeset":[
			{"op":"upsert_entity","entity":{"id":"svc.a","type":"service","name":"A","provenance":{"kind":"observation","confidence":0.8,"evidence":[]}}}
		]
	}`)

	_, err := ParseTaskResult(raw)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "observation requires non-empty evidence") {
		t.Fatalf("expected observation/evidence error, got %v", err)
	}
}

func TestParseTaskResultAllowsNestedAdditionalProperties(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"meta":{"task_id":"t1","step_id":"init.step1.collect","runtime":{"name":"claude-code","extra":"ok"},"started_at":"2026-04-03T10:00:00Z","unexpected_meta":"allowed"},
		"summary":"ok",
		"changeset":[
			{"op":"upsert_entity","entity":{"id":"svc.a","type":"service","name":"A","provenance":{"kind":"observation","confidence":0.8,"evidence":[{"repo":"r","path":"p"}],"extra_nested":"allowed"}}}
		]
	}`)

	parsed, err := ParseTaskResult(raw)
	if err != nil {
		t.Fatalf("expected nested additional properties to be allowed, got %v", err)
	}
	if strings.TrimSpace(parsed.Meta.Runtime.Name) == "" {
		t.Fatalf("expected non-empty runtime.name")
	}
}

func TestParseTaskResultRejectsUnknownTopLevelField(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"meta":{"task_id":"t1","step_id":"init.step1.collect","runtime":{"name":"claude-code"},"started_at":"2026-04-03T10:00:00Z"},
		"summary":"ok",
		"changeset":[],
		"unknown_top":"nope"
	}`)

	_, err := ParseTaskResult(raw)
	if err == nil {
		t.Fatalf("expected unknown top-level field error")
	}
	if !strings.Contains(err.Error(), "additionalProperties 'unknown_top' not allowed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func readFixtureFile(t *testing.T, rel string) []byte {
	t.Helper()

	abs := filepath.Clean(rel)
	content, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read fixture %q: %v", rel, err)
	}
	return content
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
