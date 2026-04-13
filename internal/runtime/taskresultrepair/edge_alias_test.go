package taskresultrepair

import (
	"encoding/json"
	"testing"
)

func TestRepairEdgeAliasesRepairsLegacyKeys(t *testing.T) {
	t.Parallel()

	input := []byte(`{
		"meta":{"task_id":"task-1","step_id":"refresh.step3.findings","runtime":{"name":"qwen-code"},"started_at":"2026-04-13T10:00:00Z"},
		"summary":"ok",
		"changeset":[
			{"op":"upsert_edge","edge":{"id":"edge.1","kind":"depends_on","source":"svc.a","target":"svc.b","provenance":{"kind":"inference","confidence":0.5,"evidence":[{"repo":"a","path":"README.md"}]}}}
		]
	}`)

	repaired := RepairEdgeAliases(input)
	var payload map[string]any
	if err := json.Unmarshal(repaired, &payload); err != nil {
		t.Fatalf("unmarshal repaired payload: %v", err)
	}
	changeset, _ := payload["changeset"].([]any)
	if len(changeset) != 1 {
		t.Fatalf("expected one changeset op, got %d", len(changeset))
	}
	op, _ := changeset[0].(map[string]any)
	edge, _ := op["edge"].(map[string]any)
	if edge["type"] != "depends_on" {
		t.Fatalf("expected repaired type, got %#v", edge["type"])
	}
	if edge["from"] != "svc.a" {
		t.Fatalf("expected repaired from, got %#v", edge["from"])
	}
	if edge["to"] != "svc.b" {
		t.Fatalf("expected repaired to, got %#v", edge["to"])
	}
}

func TestRepairEdgeAliasesDoesNotOverrideCanonicalFields(t *testing.T) {
	t.Parallel()

	input := []byte(`{
		"meta":{"task_id":"task-1","step_id":"refresh.step3.findings","runtime":{"name":"qwen-code"},"started_at":"2026-04-13T10:00:00Z"},
		"summary":"ok",
		"changeset":[
			{"op":"upsert_edge","edge":{"id":"edge.1","type":"calls","from":"svc.c","to":"svc.d","kind":"depends_on","source":"svc.a","target":"svc.b","provenance":{"kind":"inference","confidence":0.5,"evidence":[{"repo":"a","path":"README.md"}]}}}
		]
	}`)

	repaired := RepairEdgeAliases(input)
	var payload map[string]any
	if err := json.Unmarshal(repaired, &payload); err != nil {
		t.Fatalf("unmarshal repaired payload: %v", err)
	}
	changeset, _ := payload["changeset"].([]any)
	op, _ := changeset[0].(map[string]any)
	edge, _ := op["edge"].(map[string]any)
	if edge["type"] != "calls" {
		t.Fatalf("expected canonical type to stay unchanged, got %#v", edge["type"])
	}
	if edge["from"] != "svc.c" {
		t.Fatalf("expected canonical from to stay unchanged, got %#v", edge["from"])
	}
	if edge["to"] != "svc.d" {
		t.Fatalf("expected canonical to to stay unchanged, got %#v", edge["to"])
	}
}
