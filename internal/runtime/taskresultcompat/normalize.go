package taskresultcompat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

// NormalizeRawTaskResult repairs known collect-step compatibility drift in raw
// TaskResult JSON before schema validation. The public schema stays unchanged;
// this only strips legacy manifest-repair operations that the runtime can infer
// from write_root state and that would otherwise fail strict validation.
func NormalizeRawTaskResult(task acpruntime.Task, raw []byte) ([]byte, bool, error) {
	if len(raw) == 0 {
		return raw, false, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return raw, false, nil
	}

	changeset, ok := payload["changeset"].([]any)
	if !ok {
		return raw, false, nil
	}

	normalized := make([]any, 0, len(changeset))
	changed := false
	draftTargets := resolveRuntimeDraftArtifactTargets(task)
	for _, item := range changeset {
		opMap, ok := item.(map[string]any)
		if !ok {
			normalized = append(normalized, item)
			continue
		}
		if strings.TrimSpace(stringValue(opMap["op"])) != "add_doc_artifact" {
			normalized = append(normalized, item)
			continue
		}
		if hasValidDocArtifact(opMap["doc_artifact"]) {
			normalized = append(normalized, item)
			continue
		}
		if isCollectStep(task.StepID) && (targetsShardManifest(opMap["artifact"], task) || targetsShardManifest(opMap["doc_artifact"], task)) {
			changed = true
			continue
		}
		if len(draftTargets) > 0 && (targetsKnownRuntimeDraftArtifact(opMap["artifact"], draftTargets) || targetsKnownRuntimeDraftArtifact(opMap["doc_artifact"], draftTargets)) {
			changed = true
			continue
		}
		normalized = append(normalized, item)
	}
	if !changed {
		return raw, false, nil
	}

	payload["changeset"] = normalized
	normalizedRaw, err := json.Marshal(payload)
	if err != nil {
		return raw, false, err
	}
	return normalizedRaw, true, nil
}

func isCollectStep(stepID string) bool {
	switch strings.TrimSpace(stepID) {
	case "init.step1.collect", "refresh.step1.collect":
		return true
	default:
		return false
	}
}

func isDraftManifestCompatibilityStep(stepID string) bool {
	switch strings.TrimSpace(stepID) {
	case "init.step2.asis_docs", "refresh.step2.asis_docs", "init.step4.proposals", "refresh.step4.proposals":
		return true
	default:
		return false
	}
}

func hasValidDocArtifact(value any) bool {
	artifact, ok := value.(map[string]any)
	if !ok {
		return false
	}
	for _, key := range []string{"id", "kind", "title", "path"} {
		if strings.TrimSpace(stringValue(artifact[key])) == "" {
			return false
		}
	}
	return true
}

func targetsShardManifest(value any, task acpruntime.Task) bool {
	artifact, ok := value.(map[string]any)
	if !ok {
		return false
	}
	candidates := []string{
		stringValue(artifact["id"]),
		stringValue(artifact["path"]),
		stringValue(artifact["target_path"]),
	}
	writeRoot := filepath.ToSlash(strings.TrimSpace(task.WriteRoot))
	for _, candidate := range candidates {
		candidate = filepath.ToSlash(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		if filepath.Base(candidate) == "shard-pack-manifest.json" {
			return true
		}
		if writeRoot != "" && strings.HasPrefix(candidate, writeRoot) && strings.HasSuffix(candidate, "/shard-pack-manifest.json") {
			return true
		}
	}
	return false
}

func resolveRuntimeDraftArtifactTargets(task acpruntime.Task) map[string]struct{} {
	if !isDraftManifestCompatibilityStep(task.StepID) {
		return nil
	}
	writeRoot := strings.TrimSpace(task.WriteRoot)
	draftRoot := strings.TrimSpace(task.DraftFinalRoot)
	artifactRoot := filepath.ToSlash(strings.TrimSpace(task.ArtifactRoot))
	if writeRoot == "" || draftRoot == "" {
		return nil
	}
	manifestFile := draftManifestFileForStep(task.StepID)
	if manifestFile == "" {
		return nil
	}
	manifestPath := filepath.Join(writeRoot, manifestFile)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil
	}
	var manifest runtimeDraftManifestCompat
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil
	}
	if manifest.Version != 1 || len(manifest.Outputs) == 0 {
		return nil
	}

	targets := map[string]struct{}{
		manifestFile:                   {},
		filepath.ToSlash(manifestPath): {},
		filepath.ToSlash(filepath.Join(artifactRoot, manifestFile)): {},
	}
	for _, output := range manifest.Outputs {
		relPath := normalizeRelativePath(stringValue(output["path"]))
		canonicalPath := normalizeCanonicalPath(stringValue(output["canonical_path"]))
		if relPath == "" || canonicalPath == "" {
			return nil
		}
		absDraftPath := filepath.Join(draftRoot, filepath.FromSlash(relPath))
		info, err := os.Stat(absDraftPath)
		if err != nil || info.IsDir() {
			return nil
		}
		targets[relPath] = struct{}{}
		targets[canonicalPath] = struct{}{}
		targets[filepath.ToSlash(absDraftPath)] = struct{}{}
	}
	return targets
}

func draftManifestFileForStep(stepID string) string {
	switch strings.TrimSpace(stepID) {
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return "asis-draft-manifest.json"
	case "init.step4.proposals", "refresh.step4.proposals":
		return "proposals-draft-manifest.json"
	default:
		return ""
	}
}

func targetsKnownRuntimeDraftArtifact(value any, targets map[string]struct{}) bool {
	if len(targets) == 0 {
		return false
	}
	artifact, ok := value.(map[string]any)
	if !ok {
		return false
	}
	candidates := []string{
		stringValue(artifact["id"]),
		stringValue(artifact["path"]),
		stringValue(artifact["target_path"]),
	}
	for _, candidate := range candidates {
		candidate = filepath.ToSlash(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		if _, exists := targets[candidate]; exists {
			return true
		}
	}
	return false
}

func normalizeRelativePath(value string) string {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
	if clean == "." || clean == "" || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(clean) {
		return ""
	}
	return clean
}

func normalizeCanonicalPath(value string) string {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
	if clean == "." || clean == "" || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(clean) {
		return ""
	}
	return clean
}

type runtimeDraftManifestCompat struct {
	Version int              `json:"version"`
	Outputs []map[string]any `json:"outputs"`
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
