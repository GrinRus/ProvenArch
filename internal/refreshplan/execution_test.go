package refreshplan

import (
	"testing"
	"time"
)

func TestNewRefreshExecutionChoosesSafeNoOp(t *testing.T) {
	baseline := "run-base"
	revision := "0123456789012345678901234567890123456789"
	value := NewRefreshExecution("run-next", ImpactPlan{Version: 1, RunID: "run-next", Pipeline: "refresh", BaselineRunID: &baseline, Decision: "unchanged_candidate"}, SourceRevisions{Repos: []RepoRevision{{Name: "repo", Comparison: "unchanged", BaselineRevision: &revision, CurrentRevision: &revision}}}, time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))
	if value.Mode != "no_op" || !value.ProviderStepsSkipped {
		t.Fatalf("expected safe no-op, got %#v", value)
	}
	raw, err := MarshalRefreshExecution(value)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseRefreshExecution(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.ArtifactPath != "reports/taskruns/run-next/refresh-execution.json" {
		t.Fatalf("unexpected path %q", parsed.ArtifactPath)
	}
}

func TestNewRefreshExecutionFailsClosed(t *testing.T) {
	value := NewRefreshExecution("run-next", ImpactPlan{Version: 1, RunID: "run-next", Pipeline: "refresh", Decision: "full_refresh_required", FallbackReasons: []string{"dirty_worktree"}}, SourceRevisions{}, time.Now())
	if value.Mode != "full" || value.ProviderStepsSkipped {
		t.Fatalf("expected full execution, got %#v", value)
	}
}

func TestRefreshMaterializationContract(t *testing.T) {
	digest := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	value := RefreshMaterialization{Version: 1, RunID: "run-1", GeneratedAt: time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC).Format(time.RFC3339), Mode: "affected_only", Decisions: []ArtifactDecision{{Path: "reports/as-is/overview.md", Action: "updated", ReasonCodes: []string{"global_dependency_changed"}, ContentSHA256: &digest}}}
	raw, err := MarshalRefreshMaterialization(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseRefreshMaterialization(raw); err != nil {
		t.Fatal(err)
	}
}
