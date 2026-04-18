package artifactquality

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func EnsureCanonicalCollectManifest(task acpruntime.Task, result contracts.TaskResult) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return nil
	}

	manifestPath := filepath.Join(writeRoot, shardPackManifestFile)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	manifest, changed, err := canonicalizeCollectManifest(raw, task, result)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal canonical shard pack manifest: %w", err)
	}
	encoded = append(encoded, '\n')
	if _, err := contracts.ParseShardPackManifest(encoded); err != nil {
		return err
	}
	return os.WriteFile(manifestPath, encoded, 0o644)
}

func canonicalizeCollectManifest(raw []byte, task acpruntime.Task, result contracts.TaskResult) (contracts.ShardPackManifest, bool, error) {
	var manifest contracts.ShardPackManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return contracts.ShardPackManifest{}, false, fmt.Errorf("decode shard pack manifest for canonicalization: %w", err)
	}

	changed := false
	if manifest.Version != 1 {
		manifest.Version = 1
		changed = true
	}
	if runID := strings.TrimSpace(task.RunID); runID != "" && strings.TrimSpace(manifest.RunID) == "" {
		manifest.RunID = runID
		changed = true
	}
	if stepID := strings.TrimSpace(task.StepID); stepID != "" && strings.TrimSpace(manifest.StepID) == "" {
		manifest.StepID = stepID
		changed = true
	}
	if shardID := strings.TrimSpace(task.ShardID); shardID != "" && strings.TrimSpace(manifest.ShardID) == "" {
		manifest.ShardID = shardID
		changed = true
	}
	if domainID := strings.TrimSpace(task.DomainID); domainID != "" && strings.TrimSpace(manifest.DomainID) == "" {
		manifest.DomainID = domainID
		changed = true
	}
	if agentRole := canonicalAgentRole(task); agentRole != "" && strings.TrimSpace(manifest.AgentRole) == "" {
		manifest.AgentRole = agentRole
		changed = true
	}
	if artifactRoot := strings.TrimSpace(task.ArtifactRoot); artifactRoot != "" && strings.TrimSpace(manifest.ArtifactRoot) != artifactRoot {
		manifest.ArtifactRoot = artifactRoot
		changed = true
	}
	if len(manifest.RepoScopes) == 0 && len(task.RepoScopes) > 0 {
		manifest.RepoScopes = append([]string(nil), task.RepoScopes...)
		changed = true
	}
	if len(manifest.PathScopes) == 0 && len(task.PathScopes) > 0 {
		manifest.PathScopes = append([]string(nil), task.PathScopes...)
		changed = true
	}
	if summary := strings.TrimSpace(result.Summary); summary != "" && strings.TrimSpace(manifest.Summary) == "" {
		manifest.Summary = summary
		changed = true
	}

	compatibility := CompatibilitySnapshotFromTaskResult(result)
	if !reflect.DeepEqual(manifest.Compatibility, compatibility) {
		manifest.Compatibility = compatibility
		changed = true
	}

	encoded, err := json.Marshal(manifest)
	if err != nil {
		return contracts.ShardPackManifest{}, false, fmt.Errorf("marshal canonical shard pack manifest candidate: %w", err)
	}
	if _, err := contracts.ParseShardPackManifest(encoded); err != nil {
		return contracts.ShardPackManifest{}, false, err
	}
	return manifest, changed, nil
}

func CompatibilitySnapshotFromTaskResult(result contracts.TaskResult) contracts.CompatibilitySnapshot {
	snapshot := contracts.CompatibilitySnapshot{
		Coverage:  contracts.Coverage{},
		Questions: []contracts.Question{},
		Entities:  []contracts.Entity{},
		Edges:     []contracts.Edge{},
		Findings:  []contracts.Finding{},
	}
	if result.Coverage != nil {
		snapshot.Coverage = *result.Coverage
	}
	snapshot.Questions = append([]contracts.Question{}, result.Questions...)
	for _, op := range result.Changeset {
		switch op.Op {
		case "upsert_entity":
			if op.Entity != nil {
				snapshot.Entities = append(snapshot.Entities, *op.Entity)
			}
		case "upsert_edge":
			if op.Edge != nil {
				snapshot.Edges = append(snapshot.Edges, *op.Edge)
			}
		case "add_finding":
			if op.Finding != nil {
				snapshot.Findings = append(snapshot.Findings, *op.Finding)
			}
		}
	}
	sort.Slice(snapshot.Entities, func(i, j int) bool { return snapshot.Entities[i].ID < snapshot.Entities[j].ID })
	sort.Slice(snapshot.Edges, func(i, j int) bool { return snapshot.Edges[i].ID < snapshot.Edges[j].ID })
	sort.Slice(snapshot.Findings, func(i, j int) bool { return snapshot.Findings[i].ID < snapshot.Findings[j].ID })
	sort.Slice(snapshot.Questions, func(i, j int) bool { return snapshot.Questions[i].ID < snapshot.Questions[j].ID })
	return snapshot
}

func canonicalAgentRole(task acpruntime.Task) string {
	if role := strings.TrimSpace(task.AgentRole); role != "" {
		return role
	}
	switch {
	case strings.HasSuffix(task.StepID, "step1.collect"):
		return "shard-analyst"
	case strings.HasSuffix(task.StepID, "step3.findings"):
		return "validator"
	default:
		return "runtime"
	}
}
