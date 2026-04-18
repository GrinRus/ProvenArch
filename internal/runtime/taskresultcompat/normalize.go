package taskresultcompat

import (
	"encoding/json"
	"path/filepath"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

// NormalizeRawTaskResult repairs known collect-step compatibility drift in raw
// TaskResult JSON before schema validation. The public schema stays unchanged;
// this only strips legacy manifest-repair operations that the runtime can infer
// from write_root state and that would otherwise fail strict validation.
func NormalizeRawTaskResult(task acpruntime.Task, raw []byte) ([]byte, bool, error) {
	if !isCollectStep(task.StepID) || len(raw) == 0 {
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
		if targetsShardManifest(opMap["artifact"], task) || targetsShardManifest(opMap["doc_artifact"], task) {
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

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
