package orchestrator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
	seen := map[string]struct{}{}
	for _, domainID := range domains {
		prepared, err := e.prepareDomainCollect("refresh.step1.collect", domainID)
		if err != nil {
			return err
		}
		plans, _, _ := e.planRuntimeShards(prepared.DomainScopes)
		for _, plan := range plans {
			if _, duplicate := seen[plan.ShardID]; duplicate {
				continue
			}
			seen[plan.ShardID] = struct{}{}
			if _, changed := affected[plan.ShardID]; changed {
				continue
			}
			if err := e.copyBaselineShard(*e.refreshExecution.BaselineRunID, plan.ShardID); err != nil {
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

func (e *pipelineExecution) copyBaselineShard(baselineRunID, shardID string) error {
	sourceRel := runtimeShardArtifactRoot(baselineRunID, shardID)
	targetRel := runtimeShardArtifactRoot(e.runID, shardID)
	source, err := e.workspace.Resolve(sourceRel)
	if err != nil {
		return err
	}
	target, err := e.workspace.Resolve(targetRel)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(source, shardPackManifestFile)); err != nil {
		return fmt.Errorf("baseline manifest unavailable: %w", err)
	}
	if _, err := os.Stat(filepath.Join(source, runtimeExecutionFile)); err != nil {
		return fmt.Errorf("baseline runtime execution unavailable: %w", err)
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		return os.WriteFile(destination, raw, 0o644)
	})
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
