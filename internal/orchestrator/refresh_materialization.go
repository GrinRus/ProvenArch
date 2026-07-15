package orchestrator

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/refreshplan"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type materializationCounts struct{ Updated, Preserved, Removed, Uncertain int }

func writeRefreshMaterialization(ws workspace.Root, runID string, audit refreshplan.RefreshExecution, paths []string, defaultAction string, preservedPaths []string, removedPaths []string, now string) (Artifact, materializationCounts, error) {
	preserved := map[string]struct{}{}
	for _, path := range preservedPaths {
		preserved[path] = struct{}{}
	}
	decisions := make([]refreshplan.ArtifactDecision, 0, len(paths))
	seen := map[string]struct{}{}
	for _, rawPath := range paths {
		canonicalPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(rawPath)))
		if canonicalPath == "." || !isManagedRefreshArtifact(canonicalPath) {
			continue
		}
		if _, ok := seen[canonicalPath]; ok {
			continue
		}
		seen[canonicalPath] = struct{}{}
		var digest *string
		if raw, err := ws.ReadFile(canonicalPath); err == nil {
			value := fmt.Sprintf("%x", sha256.Sum256(raw))
			digest = &value
		}
		action := defaultAction
		if _, ok := preserved[canonicalPath]; ok {
			action = "preserved"
		}
		var sourceRunID *string
		if action == "preserved" && audit.BaselineRunID != nil {
			value := *audit.BaselineRunID
			sourceRunID = &value
		}
		decisions = append(decisions, refreshplan.ArtifactDecision{Path: canonicalPath, Action: action, ReasonCodes: []string{materializationReason(action, audit.Mode)}, SourceRunID: sourceRunID, ContentSHA256: digest})
	}
	for _, rawPath := range removedPaths {
		canonicalPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(rawPath)))
		if canonicalPath == "." || !isManagedRefreshArtifact(canonicalPath) {
			continue
		}
		if _, ok := seen[canonicalPath]; ok {
			continue
		}
		seen[canonicalPath] = struct{}{}
		decisions = append(decisions, refreshplan.ArtifactDecision{Path: canonicalPath, Action: "removed", ReasonCodes: []string{"affected_producer_removed"}, SourceRunID: audit.BaselineRunID, ContentSHA256: nil})
	}
	sort.Slice(decisions, func(i, j int) bool { return decisions[i].Path < decisions[j].Path })
	value := refreshplan.RefreshMaterialization{Version: refreshplan.RefreshMaterializationVersion, RunID: runID, GeneratedAt: now, BaselineRunID: audit.BaselineRunID, Mode: audit.Mode, Decisions: decisions}
	raw, err := refreshplan.MarshalRefreshMaterialization(value)
	if err != nil {
		return Artifact{}, materializationCounts{}, err
	}
	if _, err := refreshplan.ParseRefreshMaterialization(raw); err != nil {
		return Artifact{}, materializationCounts{}, err
	}
	artifactPath := filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "refresh-materialization.json"))
	if err := ws.WriteFile(artifactPath, append(raw, '\n')); err != nil {
		return Artifact{}, materializationCounts{}, err
	}
	counts := materializationCounts{}
	for _, decision := range decisions {
		switch decision.Action {
		case "updated":
			counts.Updated++
		case "preserved":
			counts.Preserved++
		case "removed":
			counts.Removed++
		case "uncertain":
			counts.Uncertain++
		}
	}
	return Artifact{Path: artifactPath, Kind: "refresh-materialization", Label: "Refresh materialization"}, counts, nil
}

func materializationReason(action, mode string) string {
	if action == "preserved" {
		if mode == "affected_only" {
			return "unaffected_dependency_preserved"
		}
		return "no_op_preserved_baseline"
	}
	return "full_refresh_rebuilt"
}

func isManagedRefreshArtifact(path string) bool {
	for _, prefix := range []string{"reports/as-is/", "reports/coverage/", "reports/findings/", "reports/agent-outputs/", "reports/changelog/", "proposals/", "model/entities/", "model/edges/", "model/views/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func managedContentDigests(ws workspace.Root, paths []string) map[string]string {
	out := map[string]string{}
	for _, path := range paths {
		if !isManagedRefreshArtifact(path) {
			continue
		}
		if raw, err := ws.ReadFile(path); err == nil {
			out[path] = fmt.Sprintf("%x", sha256.Sum256(raw))
		}
	}
	return out
}
