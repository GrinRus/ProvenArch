package taskresultrepair

import (
	"encoding/json"
	"strings"
)

// RepairEdgeAliases canonicalizes legacy edge keys inside upsert_edge operations.
// It maps edge.source->edge.from, edge.target->edge.to, and edge.kind->edge.type
// when canonical keys are missing/empty. The function is best-effort and returns
// the original payload when repair is not applicable.
func RepairEdgeAliases(raw []byte) []byte {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return raw
	}
	changeset, ok := payload["changeset"].([]any)
	if !ok || len(changeset) == 0 {
		return raw
	}

	changed := false
	for idx := range changeset {
		op, ok := changeset[idx].(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(asString(op["op"])) != "upsert_edge" {
			continue
		}
		edge, ok := op["edge"].(map[string]any)
		if !ok {
			continue
		}
		if repairEdgeAliasMap(edge) {
			changed = true
		}
	}
	if !changed {
		return raw
	}
	payload["changeset"] = changeset
	encoded, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return encoded
}

func repairEdgeAliasMap(edge map[string]any) bool {
	changed := false
	if strings.TrimSpace(asString(edge["from"])) == "" {
		if alias := strings.TrimSpace(asString(edge["source"])); alias != "" {
			edge["from"] = alias
			changed = true
		}
	}
	if strings.TrimSpace(asString(edge["to"])) == "" {
		if alias := strings.TrimSpace(asString(edge["target"])); alias != "" {
			edge["to"] = alias
			changed = true
		}
	}
	if strings.TrimSpace(asString(edge["type"])) == "" {
		if alias := strings.TrimSpace(asString(edge["kind"])); alias != "" {
			edge["type"] = alias
			changed = true
		}
	}
	return changed
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}
