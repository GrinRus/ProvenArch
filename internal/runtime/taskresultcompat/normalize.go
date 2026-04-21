package taskresultcompat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
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
	collectManifestReady := hasValidCollectManifestAtWriteRoot(task)
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
		if collectManifestReady && (targetsShardManifest(opMap["artifact"], task) || targetsShardManifest(opMap["doc_artifact"], task)) {
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

func NormalizeResult(task acpruntime.Task, result acpruntime.Result) (acpruntime.Result, bool, error) {
	raw := result.RawJSON
	if len(raw) == 0 && strings.TrimSpace(result.TaskResult.Meta.TaskID) != "" {
		marshaled, err := json.Marshal(result.TaskResult)
		if err != nil {
			return result, false, err
		}
		raw = marshaled
	}
	if len(raw) == 0 {
		return result, false, nil
	}
	normalizedRaw, changed, err := NormalizeRawTaskResult(task, raw)
	if err != nil || !changed {
		return result, changed, err
	}
	parsed, err := contracts.ParseTaskResult(normalizedRaw)
	if err != nil {
		return result, false, err
	}
	result.RawJSON = normalizedRaw
	result.TaskResult = parsed
	return result, true, nil
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
	return runtimedrafts.IsDraftStep(stepID)
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

func hasValidCollectManifestAtWriteRoot(task acpruntime.Task) bool {
	if !isCollectStep(task.StepID) {
		return false
	}
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return false
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(writeRoot), "shard-pack-manifest.json"))
	if err != nil {
		return false
	}
	_, err = contracts.ParseShardPackManifest(raw)
	return err == nil
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
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	if manifestFile == "" {
		return nil
	}
	manifestPath := filepath.Join(writeRoot, manifestFile)
	manifest, _, err := runtimedrafts.ValidateRequiredManifest(
		writeRoot,
		draftRoot,
		task.RunID,
		task.StepID,
		task.StepContract,
		nil,
	)
	if err != nil {
		return nil
	}

	targets := map[string]struct{}{
		manifestFile:                   {},
		filepath.ToSlash(manifestPath): {},
	}
	if artifactRoot != "" {
		targets[filepath.ToSlash(filepath.Join(artifactRoot, manifestFile))] = struct{}{}
	}
	for _, output := range manifest.Outputs {
		relPath := normalizeRelativePath(output.Path)
		canonicalPath := normalizeCanonicalPath(output.CanonicalPath)
		if relPath == "" || canonicalPath == "" {
			return nil
		}
		absDraftPath := filepath.Join(draftRoot, filepath.FromSlash(relPath))
		targets[relPath] = struct{}{}
		targets[canonicalPath] = struct{}{}
		targets[filepath.ToSlash(absDraftPath)] = struct{}{}
	}
	return targets
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

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
