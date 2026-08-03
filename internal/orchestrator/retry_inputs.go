package orchestrator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func copyRetryStaging(ws workspace.Root, parentRunID string, childRunID string, resumeStep string, requestedScopes []string) error {
	parentRunID = strings.TrimSpace(parentRunID)
	childRunID = strings.TrimSpace(childRunID)
	if parentRunID == "" || childRunID == "" || parentRunID == childRunID {
		return fmt.Errorf("retry parent and child run ids must be distinct and non-empty")
	}
	sourceRel := filepath.ToSlash(filepath.Join("reports", "taskruns", parentRunID, "staging"))
	sourceAbs, err := ws.Resolve(sourceRel)
	if err != nil {
		return err
	}
	info, err := os.Stat(sourceAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("parent staging is unavailable")
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("parent staging is not a directory")
	}
	validatedShards, err := validatedReusableShardRoots(sourceAbs, parentRunID, resumeStep, requestedScopes)
	if err != nil {
		return err
	}
	return filepath.WalkDir(sourceAbs, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("retry staging contains unsupported symlink %q", entry.Name())
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(sourceAbs, current)
		if err != nil {
			return err
		}
		canonicalRel := filepath.ToSlash(rel)
		if strings.HasPrefix(canonicalRel, "shards/") {
			parts := strings.Split(canonicalRel, "/")
			if len(parts) < 3 || !validatedShards[parts[1]] {
				return nil
			}
		} else if !retryStagingPathReusable(canonicalRel, resumeStep, requestedScopes) {
			return nil
		}
		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		targetRel := filepath.ToSlash(filepath.Join("reports", "taskruns", childRunID, "staging", rel))
		if err := ws.WriteFile(targetRel, content); err != nil {
			return err
		}
		return nil
	})
}

// ValidateRetryStaging verifies every parent shard that the retry intends to reuse.
func ValidateRetryStaging(ws workspace.Root, parentRunID, resumeStep string, requestedScopes []string) error {
	sourceRel := filepath.ToSlash(filepath.Join("reports", "taskruns", strings.TrimSpace(parentRunID), "staging"))
	sourceAbs, err := ws.Resolve(sourceRel)
	if err != nil {
		return err
	}
	info, err := os.Stat(sourceAbs)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("parent staging is unavailable")
	}
	_, err = validatedReusableShardRoots(sourceAbs, strings.TrimSpace(parentRunID), resumeStep, requestedScopes)
	return err
}

func validatedReusableShardRoots(stagingRoot, parentRunID, resumeStep string, requestedScopes []string) (map[string]bool, error) {
	result := map[string]bool{}
	if strings.Contains(strings.ToLower(resumeStep), "constitution") || (strings.Contains(strings.ToLower(resumeStep), "collect") && len(requestedScopes) == 0) {
		return result, nil
	}
	shardsRoot := filepath.Join(stagingRoot, "shards")
	entries, err := os.ReadDir(shardsRoot)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.Contains(strings.ToLower(resumeStep), "collect") && retryScopeMatchesValues(requestedScopes, []string{entry.Name()}) {
			continue
		}
		root := filepath.Join(shardsRoot, entry.Name())
		if err := artifactquality.ValidateCollectManifestInRoot(root); err != nil {
			return nil, fmt.Errorf("parent retry input shard %q is not reusable: %w", entry.Name(), err)
		}
		raw, err := os.ReadFile(filepath.Join(root, "shard-pack-manifest.json"))
		if err != nil {
			return nil, err
		}
		manifest, err := contracts.ParseShardPackManifest(raw)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(manifest.RunID) != parentRunID || !strings.HasSuffix(strings.TrimSpace(manifest.StepID), ".step1.collect") || strings.TrimSpace(manifest.ShardID) != entry.Name() || filepath.Clean(manifest.ArtifactRoot) != filepath.Clean(root) {
			return nil, fmt.Errorf("parent retry input shard %q has mismatched task identity", entry.Name())
		}
		if strings.Contains(strings.ToLower(resumeStep), "collect") && retryScopeMatchesManifest(requestedScopes, manifest) {
			continue
		}
		result[entry.Name()] = true
	}
	return result, nil
}

func retryScopeMatchesManifest(scopes []string, manifest contracts.ShardPackManifest) bool {
	values := append([]string{manifest.ShardID, manifest.DomainID}, manifest.RepoScopes...)
	return retryScopeMatchesValues(scopes, values)
}

func retryScopeMatchesValues(scopes, values []string) bool {
	for _, scope := range scopes {
		scope = strings.ToLower(strings.TrimSpace(scope))
		for _, value := range values {
			if scope != "" && scope == strings.ToLower(strings.TrimSpace(value)) {
				return true
			}
		}
	}
	return false
}

func retryStagingPathReusable(relPath, resumeStep string, requestedScopes []string) bool {
	relPath = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(relPath)), "./")
	resumeStep = strings.ToLower(strings.TrimSpace(resumeStep))
	if relPath == "" || relPath == "." || strings.HasPrefix(relPath, "../") {
		return false
	}
	switch {
	case strings.Contains(resumeStep, "constitution"):
		return false
	case strings.Contains(resumeStep, "collect"):
		if len(requestedScopes) == 0 || !strings.HasPrefix(relPath, "shards/") {
			return false
		}
		parts := strings.Split(relPath, "/")
		if len(parts) > 1 && retryScopeMatchesValues(requestedScopes, []string{parts[1]}) {
			return false
		}
		return true
	case strings.Contains(resumeStep, "asis"):
		return strings.HasPrefix(relPath, "shards/")
	case strings.Contains(resumeStep, "findings"):
		return !strings.Contains(relPath, "reports/findings/") && !strings.Contains(relPath, "reports/coverage/") && !strings.Contains(relPath, "proposals/") && !strings.Contains(relPath, "reports/changelog/")
	case strings.Contains(resumeStep, "proposals"):
		return !strings.Contains(relPath, "proposals/") && !strings.Contains(relPath, "reports/changelog/")
	default:
		return false
	}
}
