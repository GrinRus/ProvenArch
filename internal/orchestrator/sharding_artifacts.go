package orchestrator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (e *pipelineExecution) persistShardPlan(
	stepID string,
	domainID string,
	plans []runtimeShardPlan,
	plannerWarnings []string,
	semanticGraph []runtimeShardPlanGraphEdge,
	strategy string,
	maxParallel int,
	failurePolicy string,
) error {
	normalizedWarnings := make([]string, 0, len(plannerWarnings))
	for _, warning := range plannerWarnings {
		trimmed := strings.TrimSpace(warning)
		if trimmed == "" {
			continue
		}
		normalizedWarnings = append(normalizedWarnings, trimmed)
	}
	normalizedWarnings = normalizeOrderedUniqueStrings(normalizedWarnings)

	items := make([]runtimeShardPlanArtifactItem, 0, len(plans))
	for _, plan := range plans {
		items = append(items, runtimeShardPlanArtifactItem{
			SortKey:    plan.SortKey,
			ShardID:    plan.ShardID,
			RepoScopes: append([]string(nil), plan.RepoScopes...),
			PathScopes: append([]string(nil), plan.PathScopes...),
		})
	}
	semanticEdges := append([]runtimeShardPlanGraphEdge(nil), semanticGraph...)
	sort.Slice(semanticEdges, func(i, j int) bool {
		if semanticEdges[i].RepoScope != semanticEdges[j].RepoScope {
			return semanticEdges[i].RepoScope < semanticEdges[j].RepoScope
		}
		if semanticEdges[i].FromPath != semanticEdges[j].FromPath {
			return semanticEdges[i].FromPath < semanticEdges[j].FromPath
		}
		if semanticEdges[i].ToPath != semanticEdges[j].ToPath {
			return semanticEdges[i].ToPath < semanticEdges[j].ToPath
		}
		return semanticEdges[i].Reason < semanticEdges[j].Reason
	})

	payload := runtimeShardPlanArtifact{
		Version:       1,
		Meta:          runtimeArtifactMeta{Runtime: e.runtimeMetaForStep(stepID)},
		RunID:         e.runID,
		StepID:        stepID,
		DomainID:      strings.TrimSpace(domainID),
		Strategy:      strings.TrimSpace(strategy),
		MaxParallel:   maxParallel,
		FailurePolicy: strings.TrimSpace(failurePolicy),
		ShardMode:     strings.TrimSpace(e.executionProfile.ShardMode),
		PlannerNotes:  normalizedWarnings,
		SemanticGraph: semanticEdges,
		Items:         items,
	}
	if payload.Strategy == "" {
		payload.Strategy = "sequential"
	}
	if payload.MaxParallel <= 0 {
		payload.MaxParallel = 1
	}
	if payload.FailurePolicy == "" {
		payload.FailurePolicy = "best_effort"
	}
	if payload.ShardMode == "" {
		payload.ShardMode = "heuristics"
	}

	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	path := shardPlanPath(e.runID, stepID, domainID)
	if err := e.workspace.WriteFile(path, content); err != nil {
		return err
	}
	e.addArtifacts(Artifact{Path: path, Kind: "taskrun", Label: shardPlanLabel(stepID, domainID)})
	e.logInfo(stepID, domainID, "runtime shard plan persisted", map[string]any{
		"path":   path,
		"shards": len(plans),
	})
	return nil
}

func (e *pipelineExecution) persistShardSummary(stepID string, domainID string, items []runtimeShardSummaryEntry) error {
	normalizedItems, err := normalizeAndValidateShardSummaryItems(items)
	if err != nil {
		return err
	}
	summary := runtimeShardSummary{
		Version:       1,
		Meta:          runtimeArtifactMeta{Runtime: e.runtimeMetaForStep(stepID)},
		RunID:         e.runID,
		StepID:        stepID,
		DomainID:      strings.TrimSpace(domainID),
		Strategy:      strings.TrimSpace(e.executionProfile.Strategy),
		MaxParallel:   e.executionProfile.MaxParallel,
		FailurePolicy: strings.TrimSpace(e.executionProfile.FailurePolicy),
		ShardMode:     strings.TrimSpace(e.executionProfile.ShardMode),
		GeneratedAt:   e.clock().UTC().Format(time.RFC3339),
		Items:         normalizedItems,
	}
	if summary.Strategy == "" {
		summary.Strategy = "sequential"
	}
	if summary.MaxParallel <= 0 {
		summary.MaxParallel = 1
	}
	if summary.FailurePolicy == "" {
		summary.FailurePolicy = "best_effort"
	}
	if summary.ShardMode == "" {
		summary.ShardMode = "heuristics"
	}

	content, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	path := shardSummaryPath(e.runID, stepID, domainID)
	if err := e.workspace.WriteFile(path, content); err != nil {
		return err
	}
	e.addArtifacts(Artifact{Path: path, Kind: "taskrun", Label: shardSummaryLabel(stepID, domainID)})
	e.logInfo(stepID, domainID, "runtime shard summary persisted", map[string]any{"path": path, "items": len(items)})
	return nil
}

func normalizeAndValidateShardSummaryItems(items []runtimeShardSummaryEntry) ([]runtimeShardSummaryEntry, error) {
	normalized := make([]runtimeShardSummaryEntry, 0, len(items))
	for _, entry := range items {
		candidate := entry
		candidate.ShardID = strings.TrimSpace(candidate.ShardID)
		candidate.TaskID = strings.TrimSpace(candidate.TaskID)
		candidate.TaskRun = strings.TrimSpace(candidate.TaskRun)
		candidate.ErrorCode = strings.TrimSpace(candidate.ErrorCode)
		candidate.Error = strings.TrimSpace(candidate.Error)
		candidate.Status = normalizeShardSummaryStatus(candidate.Status)
		if (candidate.Status == "checkpointed" || candidate.Status == "succeeded") && candidate.TaskRun == "" {
			shardID := candidate.ShardID
			if shardID == "" {
				shardID = "<unknown>"
			}
			return nil, fmt.Errorf("invalid shard summary: shard %q status %q requires taskrun_path", shardID, candidate.Status)
		}
		normalized = append(normalized, candidate)
	}
	return normalized, nil
}

func shardTaskrunPath(runID string, stepID string, domainID string, shardID string, singleShard bool) string {
	_ = domainID
	_ = singleShard
	return runtimeExecutionMetadataPath(runID, stepID, shardID)
}

func shardTaskrunLabel(stepID string, domainID string, shardID string, singleShard bool) string {
	if singleShard {
		if strings.TrimSpace(domainID) != "" {
			return fmt.Sprintf("%s.%s", stepID, domainID)
		}
		return stepID
	}
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("%s.%s.%s", stepID, domainID, shardID)
	}
	return fmt.Sprintf("%s.%s", stepID, shardID)
}

func shardSummaryPath(runID string, stepID string, domainID string) string {
	stepSlug := strings.ReplaceAll(stepID, ".", "-")
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("reports/taskruns/%s-%s-shard-summary-%s.json", runID, stepSlug, sanitizeDomainArtifactSlug(domainID))
	}
	return fmt.Sprintf("reports/taskruns/%s-%s-shard-summary.json", runID, stepSlug)
}

func shardSummaryLabel(stepID string, domainID string) string {
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("%s.%s.shards", stepID, domainID)
	}
	return fmt.Sprintf("%s.shards", stepID)
}

func shardPlanPath(runID string, stepID string, domainID string) string {
	stepSlug := strings.ReplaceAll(stepID, ".", "-")
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("reports/taskruns/%s-%s-shard-plan-%s.json", runID, stepSlug, sanitizeDomainArtifactSlug(domainID))
	}
	return fmt.Sprintf("reports/taskruns/%s-%s-shard-plan.json", runID, stepSlug)
}

func shardPlanLabel(stepID string, domainID string) string {
	if strings.TrimSpace(domainID) != "" {
		return fmt.Sprintf("%s.%s.shard-plan", stepID, domainID)
	}
	return fmt.Sprintf("%s.shard-plan", stepID)
}

func buildShardTaskSuffix(prefix string, shardID string) string {
	if strings.TrimSpace(prefix) == "" {
		return "shard-" + sanitizeDomainArtifactSlug(shardID)
	}
	return strings.TrimSpace(prefix) + "-shard-" + sanitizeDomainArtifactSlug(shardID)
}
