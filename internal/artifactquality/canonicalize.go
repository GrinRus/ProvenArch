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
	if violations := detectLegacyCollectManifestViolations(raw); len(violations) > 0 {
		return contracts.ShardPackManifest{}, RepairReport{}, fmt.Errorf(
			"legacy collect manifest fields are forbidden: %s",
			strings.Join(violations, "; "),
		)
	}

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

func detectLegacyCollectManifestViolations(raw []byte) []string {
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}

	root, ok := payload.(map[string]any)
	if !ok {
		return nil
	}

	violations := make([]string, 0, 8)
	appendViolation := func(path string, detail string) {
		path = strings.TrimSpace(path)
		detail = strings.TrimSpace(detail)
		if path == "" || detail == "" {
			return
		}
		violation := fmt.Sprintf("%s (%s)", path, detail)
		for _, existing := range violations {
			if existing == violation {
				return
			}
		}
		violations = append(violations, violation)
	}

	if _, exists := root["step_contract"]; exists {
		appendViolation("step_contract", "top-level step_contract is forbidden")
	}
	if _, exists := root["compatibility"]; exists {
		appendViolation("compatibility", "legacy compatibility block is forbidden")
	}

	semantic, _ := root["semantic"].(map[string]any)
	if coverage, ok := semantic["coverage"].(map[string]any); ok {
		if _, exists := coverage["covered_topics"]; exists {
			appendViolation("semantic.coverage.covered_topics", "use semantic.coverage.observed")
		}
	}
	if questions, ok := semantic["questions"].([]any); ok {
		for index, question := range questions {
			questionMap, ok := question.(map[string]any)
			if !ok {
				continue
			}
			if _, exists := questionMap["question"]; exists {
				appendViolation(fmt.Sprintf("semantic.questions[%d].question", index), "use semantic.questions[*].text")
			}
			if confidence, exists := questionMap["confidence"]; exists {
				appendStringConfidenceViolation(appendViolation, fmt.Sprintf("semantic.questions[%d].confidence", index), confidence)
			}
		}
	}
	if entities, ok := semantic["entities"].([]any); ok {
		for index, entity := range entities {
			entityMap, ok := entity.(map[string]any)
			if !ok {
				continue
			}
			detectLegacyProvenanceShape(appendViolation, fmt.Sprintf("semantic.entities[%d].provenance", index), entityMap["provenance"])
		}
	}
	if edges, ok := semantic["edges"].([]any); ok {
		for index, edge := range edges {
			edgeMap, ok := edge.(map[string]any)
			if !ok {
				continue
			}
			if _, exists := edgeMap["relation"]; exists {
				appendViolation(fmt.Sprintf("semantic.edges[%d].relation", index), "use semantic.edges[*].type")
			}
			detectLegacyProvenanceShape(appendViolation, fmt.Sprintf("semantic.edges[%d].provenance", index), edgeMap["provenance"])
		}
	}
	if findings, ok := semantic["findings"].([]any); ok {
		for index, finding := range findings {
			findingMap, ok := finding.(map[string]any)
			if !ok {
				continue
			}
			if _, exists := findingMap["evidence_citation_ids"]; exists {
				appendViolation(fmt.Sprintf("semantic.findings[%d].evidence_citation_ids", index), "use semantic.findings[*].provenance.evidence")
			}
			if _, exists := findingMap["inference"]; exists {
				appendViolation(fmt.Sprintf("semantic.findings[%d].inference", index), "legacy inference compatibility fields are forbidden")
			}
			if _, exists := findingMap["summary"]; exists {
				appendViolation(fmt.Sprintf("semantic.findings[%d].summary", index), "legacy summary compatibility fields are forbidden")
			}
			if confidence, exists := findingMap["confidence"]; exists {
				appendStringConfidenceViolation(appendViolation, fmt.Sprintf("semantic.findings[%d].confidence", index), confidence)
			}
			detectLegacyProvenanceShape(appendViolation, fmt.Sprintf("semantic.findings[%d].provenance", index), findingMap["provenance"])
		}
	}

	if len(violations) > 8 {
		return append(violations[:8], "additional legacy collect violations omitted")
	}
	return violations
}

func appendStringConfidenceViolation(appendViolation func(string, string), path string, value any) {
	if _, ok := value.(string); ok {
		appendViolation(path, "confidence must be numeric")
	}
}

func detectLegacyProvenanceShape(appendViolation func(string, string), path string, value any) {
	switch typed := value.(type) {
	case []any:
		appendViolation(path, "provenance must be an object, not an array")
	case map[string]any:
		if confidence, exists := typed["confidence"]; exists {
			appendStringConfidenceViolation(appendViolation, path+".confidence", confidence)
		}
		if evidence, exists := typed["evidence"].([]any); exists {
			for index, item := range evidence {
				detectSemanticEvidenceViolations(appendViolation, fmt.Sprintf("%s.evidence[%d]", path, index), item)
			}
		}
	}
}

func detectSemanticEvidenceViolations(appendViolation func(string, string), path string, value any) {
	item, ok := value.(map[string]any)
	if !ok {
		return
	}
	repo := strings.TrimSpace(stringValue(item["repo"]))
	evidencePath := strings.TrimSpace(stringValue(item["path"]))
	if repo == "" || evidencePath == "" {
		appendViolation(path, "semantic provenance evidence requires non-empty repo/path; citation-only evidence objects are forbidden")
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
