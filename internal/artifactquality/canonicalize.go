package artifactquality

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/compatibilityregistry"
)

type RepairReport struct {
	Changed        bool     `json:"changed"`
	AppliedRuleIDs []string `json:"applied_rule_ids,omitempty"`
}

func (r *RepairReport) markChanged() {
	r.Changed = true
}

func (r *RepairReport) addAppliedRuleID(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	for _, existing := range r.AppliedRuleIDs {
		if existing == id {
			r.Changed = true
			return
		}
	}
	r.AppliedRuleIDs = append(r.AppliedRuleIDs, id)
	r.Changed = true
}

// RepairCollectManifest applies the explicit collect repair rules allowed by the
// docs-first runtime contract. It never invents semantic payloads; it only
// normalizes write-root metadata and unambiguous document paths.
func RepairCollectManifest(task acpruntime.Task) (RepairReport, error) {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return RepairReport{}, nil
	}

	manifestPath := filepath.Join(writeRoot, shardPackManifestFile)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return RepairReport{}, nil
		}
		return RepairReport{}, err
	}

	manifest, report, err := canonicalizeCollectManifest(raw, task)
	if err != nil {
		return RepairReport{}, err
	}
	if !report.Changed {
		return report, nil
	}

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return RepairReport{}, fmt.Errorf("marshal canonical shard pack manifest: %w", err)
	}
	encoded = append(encoded, '\n')
	if _, err := contracts.ParseShardPackManifest(encoded); err != nil {
		return RepairReport{}, err
	}
	if err := os.WriteFile(manifestPath, encoded, 0o644); err != nil {
		return RepairReport{}, err
	}
	return report, nil
}

func canonicalizeCollectManifest(raw []byte, task acpruntime.Task) (contracts.ShardPackManifest, RepairReport, error) {
	var manifest contracts.ShardPackManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return contracts.ShardPackManifest{}, RepairReport{}, fmt.Errorf("decode shard pack manifest for canonicalization: %w", err)
	}

	var report RepairReport
	if manifest.Version != 1 {
		manifest.Version = 1
		report.markChanged()
	}
	if runID := strings.TrimSpace(task.RunID); runID != "" && strings.TrimSpace(manifest.RunID) == "" {
		manifest.RunID = runID
		report.markChanged()
	}
	if stepID := strings.TrimSpace(task.StepID); stepID != "" && strings.TrimSpace(manifest.StepID) == "" {
		manifest.StepID = stepID
		report.markChanged()
	}
	if shardID := strings.TrimSpace(task.ShardID); shardID != "" && strings.TrimSpace(manifest.ShardID) == "" {
		manifest.ShardID = shardID
		report.markChanged()
	}
	if domainID := strings.TrimSpace(task.DomainID); domainID != "" && strings.TrimSpace(manifest.DomainID) == "" {
		manifest.DomainID = domainID
		report.markChanged()
	}
	if agentRole := canonicalAgentRole(task); agentRole != "" && strings.TrimSpace(manifest.AgentRole) == "" {
		manifest.AgentRole = agentRole
		report.markChanged()
	}
	if artifactRoot := strings.TrimSpace(task.ArtifactRoot); artifactRoot != "" && strings.TrimSpace(manifest.ArtifactRoot) != artifactRoot {
		manifest.ArtifactRoot = artifactRoot
		report.markChanged()
	}
	if normalizedDocuments, documentReport := normalizeCollectDocumentPaths(manifest.Documents, manifest.ArtifactRoot, task.WriteRoot); documentReport.Changed {
		manifest.Documents = normalizedDocuments
		report.markChanged()
		for _, ruleID := range documentReport.AppliedRuleIDs {
			report.addAppliedRuleID(ruleID)
		}
	}
	if len(manifest.RepoScopes) == 0 && len(task.RepoScopes) > 0 {
		manifest.RepoScopes = append([]string(nil), task.RepoScopes...)
		report.markChanged()
	}
	if len(manifest.PathScopes) == 0 && len(task.PathScopes) > 0 {
		manifest.PathScopes = append([]string(nil), task.PathScopes...)
		report.markChanged()
	}

	encoded, err := json.Marshal(manifest)
	if err != nil {
		return contracts.ShardPackManifest{}, RepairReport{}, fmt.Errorf("marshal canonical shard pack manifest candidate: %w", err)
	}
	if _, err := contracts.ParseShardPackManifest(encoded); err != nil {
		return contracts.ShardPackManifest{}, RepairReport{}, err
	}
	return manifest, report, nil
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

func normalizeCollectDocumentPaths(documents []contracts.AuthoredDocument, artifactRoot string, writeRoot string) ([]contracts.AuthoredDocument, RepairReport) {
	if len(documents) == 0 {
		return documents, RepairReport{}
	}

	normalized := append([]contracts.AuthoredDocument(nil), documents...)
	var report RepairReport
	for idx, document := range normalized {
		if relPath, ruleID, ok := normalizeCollectDocumentPath(document.Path, artifactRoot, writeRoot); ok && relPath != document.Path {
			normalized[idx].Path = relPath
			report.markChanged()
			report.addAppliedRuleID(ruleID)
		}
	}
	return normalized, report
}

func normalizeCollectDocumentPath(rawPath string, artifactRoot string, writeRoot string) (string, string, bool) {
	cleanedPath := filepath.Clean(filepath.FromSlash(strings.TrimSpace(rawPath)))
	if cleanedPath == "." || cleanedPath == "" {
		return "", "", false
	}

	cleanedWriteRoot := strings.TrimSpace(writeRoot)
	if filepath.IsAbs(cleanedPath) && cleanedWriteRoot != "" {
		if relPath, ok := normalizeAbsoluteCollectDocumentPath(cleanedPath, cleanedWriteRoot); ok {
			return relPath, compatibilityregistry.RuleSafeCollectDocumentPathNormalization, true
		}
	}

	normalizedArtifactRoot := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(artifactRoot))))
	if normalizedArtifactRoot == "." || normalizedArtifactRoot == "" {
		return "", "", false
	}

	normalizedPath := filepath.ToSlash(cleanedPath)
	if normalizedPath == normalizedArtifactRoot {
		return "", "", false
	}
	if !strings.HasPrefix(normalizedPath, normalizedArtifactRoot+"/") {
		return "", "", false
	}
	relPath := strings.TrimPrefix(normalizedPath, normalizedArtifactRoot+"/")
	if relPath == "" {
		return "", "", false
	}
	if !collectDocumentExistsAtWriteRoot(cleanedWriteRoot, relPath) {
		return "", "", false
	}
	return relPath, compatibilityregistry.RuleSafeCollectDocumentPathNormalization, true
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
