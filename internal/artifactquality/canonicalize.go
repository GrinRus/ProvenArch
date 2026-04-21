package artifactquality

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

// RepairCollectManifest applies the explicit collect repair rules allowed by the
// docs-first runtime contract. It never invents semantic payloads; it only
// normalizes write-root metadata and unambiguous document paths.
func RepairCollectManifest(task acpruntime.Task) error {
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

	manifest, changed, err := canonicalizeCollectManifest(raw, task)
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

func canonicalizeCollectManifest(raw []byte, task acpruntime.Task) (contracts.ShardPackManifest, bool, error) {
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
	if normalizedDocuments, documentChanged := normalizeCollectDocumentPaths(manifest.Documents, manifest.ArtifactRoot, task.WriteRoot); documentChanged {
		manifest.Documents = normalizedDocuments
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

	encoded, err := json.Marshal(manifest)
	if err != nil {
		return contracts.ShardPackManifest{}, false, fmt.Errorf("marshal canonical shard pack manifest candidate: %w", err)
	}
	if _, err := contracts.ParseShardPackManifest(encoded); err != nil {
		return contracts.ShardPackManifest{}, false, err
	}
	return manifest, changed, nil
}

func canonicalAgentRole(task acpruntime.Task) string {
	if role := strings.TrimSpace(task.AgentRole); role != "" {
		return role
	}
	switch {
	case strings.HasSuffix(task.StepID, "step1.collect"):
		return "shard-analyst"
	case strings.HasSuffix(task.StepID, "step3.findings"):
		return "validator-findings"
	default:
		return "runtime"
	}
}

func normalizeCollectDocumentPaths(documents []contracts.AuthoredDocument, artifactRoot string, writeRoot string) ([]contracts.AuthoredDocument, bool) {
	if len(documents) == 0 {
		return documents, false
	}

	normalized := append([]contracts.AuthoredDocument(nil), documents...)
	changed := false
	for idx, document := range normalized {
		if relPath, ok := normalizeCollectDocumentPath(document.Path, artifactRoot, writeRoot); ok && relPath != document.Path {
			normalized[idx].Path = relPath
			changed = true
		}
	}
	return normalized, changed
}

func normalizeCollectDocumentPath(rawPath string, artifactRoot string, writeRoot string) (string, bool) {
	cleanedPath := filepath.Clean(filepath.FromSlash(strings.TrimSpace(rawPath)))
	if cleanedPath == "." || cleanedPath == "" {
		return "", false
	}

	cleanedWriteRoot := strings.TrimSpace(writeRoot)
	if filepath.IsAbs(cleanedPath) && cleanedWriteRoot != "" {
		if relPath, ok := normalizeAbsoluteCollectDocumentPath(cleanedPath, cleanedWriteRoot); ok {
			return relPath, true
		}
	}

	normalizedArtifactRoot := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(artifactRoot))))
	if normalizedArtifactRoot == "." || normalizedArtifactRoot == "" {
		return "", false
	}

	normalizedPath := filepath.ToSlash(cleanedPath)
	if normalizedPath == normalizedArtifactRoot {
		return "", false
	}
	if !strings.HasPrefix(normalizedPath, normalizedArtifactRoot+"/") {
		return "", false
	}
	relPath := strings.TrimPrefix(normalizedPath, normalizedArtifactRoot+"/")
	if relPath == "" {
		return "", false
	}
	if !collectDocumentExistsAtWriteRoot(cleanedWriteRoot, relPath) {
		return "", false
	}
	return relPath, true
}

func normalizeAbsoluteCollectDocumentPath(absPath string, writeRoot string) (string, bool) {
	relativePath, err := filepath.Rel(filepath.Clean(writeRoot), filepath.Clean(absPath))
	if err != nil {
		return "", false
	}
	relativePath = filepath.Clean(relativePath)
	if relativePath == "." || relativePath == "" {
		return "", false
	}
	if strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || relativePath == ".." {
		return "", false
	}
	if !collectDocumentExistsAtWriteRoot(writeRoot, relativePath) {
		return "", false
	}
	return filepath.ToSlash(relativePath), true
}

func collectDocumentExistsAtWriteRoot(writeRoot string, relPath string) bool {
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return false
	}
	target := filepath.Join(filepath.Clean(writeRoot), filepath.Clean(relPath))
	info, err := os.Stat(target)
	return err == nil && !info.IsDir()
}
