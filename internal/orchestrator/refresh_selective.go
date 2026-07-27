package orchestrator

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/refreshplan"
)

func (e *pipelineExecution) prepareSelectiveCollectBaseline() error {
	if e.refreshExecution == nil || e.refreshExecution.Mode != "affected_only" || e.refreshExecution.BaselineRunID == nil {
		return nil
	}
	affected := map[string]struct{}{}
	for _, id := range e.refreshExecution.AffectedShards {
		affected[strings.TrimSpace(id)] = struct{}{}
	}
	domains, err := loadCanonicalDomainIDs(e.workspace)
	if err != nil {
		return err
	}
	preserved := []string{}
	seen := map[string]string{}
	for _, domainID := range domains {
		prepared, err := e.prepareDomainCollect("refresh.step1.collect", domainID)
		if err != nil {
			return err
		}
		plans, _, _ := e.planRuntimeShards(prepared.DomainScopes)
		for _, plan := range plans {
			identity := shardPlanIdentity(domainID, plan)
			if previous, duplicate := seen[plan.ShardID]; duplicate {
				if previous != identity {
					return fmt.Errorf("shard id %q maps to conflicting identities", plan.ShardID)
				}
				continue
			}
			seen[plan.ShardID] = identity
			if _, changed := affected[plan.ShardID]; changed {
				continue
			}
			if err := e.copyBaselineShard(*e.refreshExecution.BaselineRunID, "refresh.step1.collect", domainID, plan); err != nil {
				return fmt.Errorf("preserve shard %q: %w", plan.ShardID, err)
			}
			preserved = append(preserved, plan.ShardID)
		}
	}
	for id := range affected {
		if _, ok := seen[id]; !ok {
			return fmt.Errorf("affected shard %q is absent from current shard plan", id)
		}
	}
	sort.Strings(preserved)
	e.refreshExecution.PreservedShards = preserved
	return nil
}

func (e *pipelineExecution) copyBaselineShard(
	baselineRunID string,
	stepID string,
	domainID string,
	plan runtimeShardPlan,
) error {
	integrity, files, err := e.validateBaselineShard(baselineRunID, stepID, domainID, plan)
	if err != nil {
		return err
	}
	targetRel := runtimeShardArtifactRoot(e.runID, plan.ShardID)
	for _, file := range files {
		content := file.Content
		if file.Path == shardPackManifestFile {
			content, err = rewritePreservedShardManifest(content, baselineRunID, e.runID, targetRel)
			if err != nil {
				return err
			}
		}
		if file.Path == runtimeExecutionFile {
			content, err = rewritePreservedRuntimeExecution(content, baselineRunID, e.runID, targetRel)
			if err != nil {
				return err
			}
		}
		if err := e.workspace.WriteFileAtomic(path.Join(targetRel, file.Path), content); err != nil {
			return err
		}
	}
	return e.writeShardBaselineIntegrity(stepID, domainID, plan, &integrity)
}

func shardPlanIdentity(domainID string, plan runtimeShardPlan) string {
	return strings.Join([]string{
		strings.TrimSpace(domainID),
		strings.Join(normalizedIdentityValues(plan.RepoScopes), "\x00"),
		strings.Join(normalizedIdentityValues(plan.PathScopes), "\x00"),
	}, "\x01")
}

func persistRefreshExecutionAudit(wsPath func(string, []byte) error, audit *refreshplan.RefreshExecution) error {
	raw, err := refreshplan.MarshalRefreshExecution(*audit)
	if err != nil {
		return err
	}
	if _, err := refreshplan.ParseRefreshExecution(raw); err != nil {
		return err
	}
	return wsPath(audit.ArtifactPath, append(raw, '\n'))
}
