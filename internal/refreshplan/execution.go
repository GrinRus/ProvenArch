package refreshplan

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func NewRefreshExecution(runID string, plan ImpactPlan, revisions SourceRevisions, now time.Time) RefreshExecution {
	mode := "full"
	reasons := append([]string(nil), plan.FallbackReasons...)
	if plan.Decision == "unchanged_candidate" && len(plan.FallbackReasons) == 0 {
		mode = "no_op"
		if allReposUnchanged(revisions) {
			reasons = append(reasons, "source_revisions_unchanged")
		} else {
			reasons = append(reasons, "changes_outside_analysis_scope")
		}
	}
	if plan.Decision == "selective_candidate" && len(plan.FallbackReasons) == 0 && plan.BaselineRunID != nil && len(plan.AffectedShards) > 0 {
		mode = "affected_only"
	}
	ranges := make([]SourceRange, 0, len(revisions.Repos))
	for _, repo := range revisions.Repos {
		ranges = append(ranges, SourceRange{Repo: repo.Name, BaselineRevision: repo.BaselineRevision, CurrentRevision: repo.CurrentRevision})
	}
	artifactPath := filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "refresh-execution.json"))
	return RefreshExecution{Version: RefreshExecutionVersion, RunID: runID, GeneratedAt: now.UTC().Format(time.RFC3339), BaselineRunID: plan.BaselineRunID, PlanDecision: plan.Decision, Mode: mode, ReasonCodes: uniqueSorted(reasons...), SourceRanges: ranges, AffectedShards: append([]string(nil), plan.AffectedShards...), PreservedShards: []string{}, ProviderStepsSkipped: mode == "no_op", ArtifactPath: artifactPath}
}

func allReposUnchanged(revisions SourceRevisions) bool {
	if len(revisions.Repos) == 0 {
		return false
	}
	for _, repo := range revisions.Repos {
		if strings.TrimSpace(repo.Comparison) != "unchanged" {
			return false
		}
	}
	return true
}

func SummaryReasonCodes(execution RefreshExecution) []string {
	out := append([]string(nil), execution.ReasonCodes...)
	sort.Strings(out)
	return out
}
