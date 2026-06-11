package reports

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestCompileC4DiagramsDeterministicArtifactsAndIndex(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)

	entities := []contracts.Entity{
		{
			ID:   "api.orders.http.public",
			Type: "api.http",
			Name: "Public Orders API",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.9,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "cmd/server/main.go"},
				},
			},
		},
		{
			ID:   "svc.orders",
			Type: "service",
			Name: "orders",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/internal/handler.go"},
				},
			},
		},
		{
			ID:   "db.orders",
			Type: "datastore",
			Name: "orders-db",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "db/migrations/001_init.sql"},
				},
			},
		},
		{
			ID:   "team.platform",
			Type: "team",
			Name: "Platform Team",
			Provenance: contracts.Provenance{
				Kind:       "assertion",
				Confidence: 0.8,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "CODEOWNERS"},
				},
			},
		},
	}
	edges := []contracts.Edge{
		{
			ID:   "edge.orders.db",
			Type: "depends_on",
			From: "svc.orders",
			To:   "db.orders",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.9,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/internal/repository.go"},
				},
			},
		},
		{
			ID:   "edge.orders.api",
			Type: "implements",
			From: "svc.orders",
			To:   "api.orders.http.public",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.9,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/internal/handler.go"},
				},
			},
		},
	}

	firstArtifacts, err := compiler.CompileC4Diagrams(entities, edges)
	if err != nil {
		t.Fatalf("compile c4 diagrams (first): %v", err)
	}
	secondArtifacts, err := compiler.CompileC4Diagrams(entities, edges)
	if err != nil {
		t.Fatalf("compile c4 diagrams (second): %v", err)
	}
	if !reflect.DeepEqual(firstArtifacts, secondArtifacts) {
		t.Fatalf("expected deterministic artifact ordering, got first=%+v second=%+v", firstArtifacts, secondArtifacts)
	}

	requiredPaths := []string{
		"reports/diagrams/c4-context.mmd",
		"reports/diagrams/c4-container.mmd",
		"reports/diagrams/components/svc-orders.mmd",
		"reports/diagrams/code/svc-orders.mmd",
		"reports/diagrams/index.md",
	}
	for _, rel := range requiredPaths {
		if _, err := os.Stat(filepath.Join(ws.Path, rel)); err != nil {
			t.Fatalf("expected diagram artifact %q: %v", rel, err)
		}
	}

	indexContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/index.md"))
	if err != nil {
		t.Fatalf("read diagrams index: %v", err)
	}
	indexText := string(indexContent)
	if !strings.Contains(indexText, "reports/diagrams/components/svc-orders.mmd") {
		t.Fatalf("expected component diagram path in index, got:\n%s", indexText)
	}
	if !strings.Contains(indexText, "reports/diagrams/code/svc-orders.mmd") {
		t.Fatalf("expected code diagram path in index, got:\n%s", indexText)
	}

	componentContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/components/svc-orders.mmd"))
	if err != nil {
		t.Fatalf("read component diagram: %v", err)
	}
	componentText := string(componentContent)
	if !strings.Contains(componentText, "Public Orders API") {
		t.Fatalf("expected API component node in component diagram, got:\n%s", componentText)
	}
	if !strings.Contains(componentText, "|implements|") {
		t.Fatalf("expected evidence-backed component edge in component diagram, got:\n%s", componentText)
	}

	codeContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/code/svc-orders.mmd"))
	if err != nil {
		t.Fatalf("read code diagram: %v", err)
	}
	codeText := string(codeContent)
	for _, expectedPath := range []string{"cmd/", "db/", "services/"} {
		if !strings.Contains(codeText, expectedPath) {
			t.Fatalf("expected code diagram path %q, got:\n%s", expectedPath, codeText)
		}
	}
}

func TestCompileC4DiagramsStrictEvidenceIncludesGaps(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)

	entities := []contracts.Entity{
		{
			ID:   "svc.ghost",
			Type: "service",
			Name: "ghost",
			Provenance: contracts.Provenance{
				Kind:       "inference",
				Confidence: 0.2,
				Evidence:   nil,
			},
		},
	}

	if _, err := compiler.CompileC4Diagrams(entities, nil); err != nil {
		t.Fatalf("compile c4 diagrams: %v", err)
	}

	contextContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/c4-context.mmd"))
	if err != nil {
		t.Fatalf("read context diagram: %v", err)
	}
	contextText := string(contextContent)
	if !strings.Contains(contextText, "Gap: no evidence-backed external systems") {
		t.Fatalf("expected external systems gap in context diagram, got:\n%s", contextText)
	}
	if !strings.Contains(contextText, "Gap: no evidence-backed actors") {
		t.Fatalf("expected actors gap in context diagram, got:\n%s", contextText)
	}

	containerContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/c4-container.mmd"))
	if err != nil {
		t.Fatalf("read container diagram: %v", err)
	}
	containerText := string(containerContent)
	if !strings.Contains(containerText, "Gap: no evidence-backed containers") {
		t.Fatalf("expected container gap in container diagram, got:\n%s", containerText)
	}
	if !strings.Contains(containerText, "Gap: no evidence-backed container relations") {
		t.Fatalf("expected relation gap in container diagram, got:\n%s", containerText)
	}
}

func TestCompileC4DiagramsIgnoresEdgesWithoutEvidence(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)

	entities := []contracts.Entity{
		{
			ID:   "svc.orders",
			Type: "service",
			Name: "orders",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/main.go"},
				},
			},
		},
		{
			ID:   "ext.payments",
			Type: "external.system",
			Name: "payments",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.8,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "docs/integrations/payments.md"},
				},
			},
		},
		{
			ID:   "db.orders",
			Type: "datastore",
			Name: "orders-db",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "db/orders.sql"},
				},
			},
		},
	}
	edges := []contracts.Edge{
		{
			ID:   "edge.orders.external",
			Type: "calls",
			From: "svc.orders",
			To:   "ext.payments",
			Provenance: contracts.Provenance{
				Kind:       "inference",
				Confidence: 0.5,
				Evidence:   nil,
			},
		},
		{
			ID:   "edge.orders.db",
			Type: "depends_on",
			From: "svc.orders",
			To:   "db.orders",
			Provenance: contracts.Provenance{
				Kind:       "inference",
				Confidence: 0.5,
				Evidence:   nil,
			},
		},
	}

	if _, err := compiler.CompileC4Diagrams(entities, edges); err != nil {
		t.Fatalf("compile c4 diagrams: %v", err)
	}

	contextContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/c4-context.mmd"))
	if err != nil {
		t.Fatalf("read context diagram: %v", err)
	}
	contextText := string(contextContent)
	if !strings.Contains(contextText, "External: payments") {
		t.Fatalf("expected external node in context diagram, got:\n%s", contextText)
	}
	if !strings.Contains(contextText, "Gap: no evidence-backed relationships") {
		t.Fatalf("expected relations gap when edges have no evidence, got:\n%s", contextText)
	}
	if strings.Contains(contextText, "System --> ext_ext_payments") || strings.Contains(contextText, "ext_ext_payments --> System") {
		t.Fatalf("did not expect relation arrow to external node without edge evidence, got:\n%s", contextText)
	}

	containerContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/c4-container.mmd"))
	if err != nil {
		t.Fatalf("read container diagram: %v", err)
	}
	containerText := string(containerContent)
	if !strings.Contains(containerText, "Gap: no evidence-backed container relations") {
		t.Fatalf("expected container relations gap when edges have no evidence, got:\n%s", containerText)
	}

	codeContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/code/svc-orders.mmd"))
	if err != nil {
		t.Fatalf("read code diagram: %v", err)
	}
	codeText := string(codeContent)
	if !strings.Contains(codeText, "services/") {
		t.Fatalf("expected service evidence path in code diagram, got:\n%s", codeText)
	}
	if strings.Contains(codeText, "docs/") {
		t.Fatalf("did not expect related path from edge without evidence in code diagram, got:\n%s", codeText)
	}
}

func TestCompileC4ContextDiagramRendersEvidenceBackedRelations(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)

	entities := []contracts.Entity{
		{
			ID:   "svc.orders",
			Type: "service",
			Name: "orders",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/main.go"},
				},
			},
		},
		{
			ID:   "ext.payments",
			Type: "external.system",
			Name: "payments",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "docs/integrations/payments.md"},
				},
			},
		},
		{
			ID:   "team.platform",
			Type: "team",
			Name: "platform",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "CODEOWNERS"},
				},
			},
		},
	}
	edges := []contracts.Edge{
		{
			ID:   "edge.orders.payments",
			Type: "calls",
			From: "svc.orders",
			To:   "ext.payments",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.9,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/client/payments.go"},
				},
			},
		},
		{
			ID:   "edge.platform.orders",
			Type: "owns",
			From: "team.platform",
			To:   "svc.orders",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.8,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "CODEOWNERS"},
				},
			},
		},
	}

	if _, err := compiler.CompileC4Diagrams(entities, edges); err != nil {
		t.Fatalf("compile c4 diagrams: %v", err)
	}

	contextContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/c4-context.mmd"))
	if err != nil {
		t.Fatalf("read context diagram: %v", err)
	}
	contextText := string(contextContent)

	if strings.Contains(contextText, "Gap: no evidence-backed relationships") {
		t.Fatalf("did not expect relationships gap with evidence-backed edges, got:\n%s", contextText)
	}
	if !strings.Contains(contextText, "System --> ext_ext_payments") {
		t.Fatalf("expected system-to-external relation in context diagram, got:\n%s", contextText)
	}
	if !strings.Contains(contextText, "team_team_platform --> System") {
		t.Fatalf("expected team-to-system relation in context diagram, got:\n%s", contextText)
	}
}

func TestCompileC4ContextDiagramKeepsInternalRelationsWhenOnlyActorRelationExists(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)

	entities := []contracts.Entity{
		{
			ID:   "team.security",
			Type: "team",
			Name: "Security Team",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "CODEOWNERS"},
				},
			},
		},
		{
			ID:   "svc.capture",
			Type: "service",
			Name: "Capture Service",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/capture/main.go"},
				},
			},
		},
		{
			ID:   "svc.ingestion",
			Type: "service",
			Name: "Ingestion Service",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/ingestion/main.go"},
				},
			},
		},
		{
			ID:   "db.events",
			Type: "datastore",
			Name: "Events Store",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "db/events.sql"},
				},
			},
		},
	}
	edges := []contracts.Edge{
		{
			ID:   "edge.security.capture",
			Type: "owns",
			From: "team.security",
			To:   "svc.capture",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.9,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "CODEOWNERS"},
				},
			},
		},
		{
			ID:   "edge.capture.ingestion",
			Type: "publishes_to",
			From: "svc.capture",
			To:   "svc.ingestion",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.9,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/capture/publisher.go"},
				},
			},
		},
		{
			ID:   "edge.ingestion.events",
			Type: "persists_to",
			From: "svc.ingestion",
			To:   "db.events",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.9,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/ingestion/store.go"},
				},
			},
		},
	}

	if _, err := compiler.CompileC4Diagrams(entities, edges); err != nil {
		t.Fatalf("compile c4 diagrams: %v", err)
	}

	contextContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/c4-context.mmd"))
	if err != nil {
		t.Fatalf("read context diagram: %v", err)
	}
	contextText := string(contextContent)

	for _, expected := range []string{
		"team_team_security --> System",
		"Evidence-backed workspace internals",
		"Service: Capture Service",
		"Service: Ingestion Service",
		"Datastore: Events Store",
		"ctx_svc_capture -->|publishes_to| ctx_svc_ingestion",
		"ctx_svc_ingestion -->|persists_to| ctx_db_events",
	} {
		if !strings.Contains(contextText, expected) {
			t.Fatalf("expected %q in context diagram, got:\n%s", expected, contextText)
		}
	}
}

func TestCompileC4ContextDiagramUsesInternalRelationsWhenNoExternalActors(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)

	entities := []contracts.Entity{
		{
			ID:   "svc.orders",
			Type: "service",
			Name: "orders",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/main.go"},
				},
			},
		},
		{
			ID:   "svc.kitchen",
			Type: "service",
			Name: "kitchen",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/kitchen/main.go"},
				},
			},
		},
		{
			ID:   "db.orders",
			Type: "datastore",
			Name: "orders-db",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "db/orders.sql"},
				},
			},
		},
	}
	edges := []contracts.Edge{
		{
			ID:   "edge.orders.kitchen",
			Type: "calls",
			From: "svc.orders",
			To:   "svc.kitchen",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.9,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/kitchen_client.go"},
				},
			},
		},
		{
			ID:   "edge.orders.db",
			Type: "uses",
			From: "svc.orders",
			To:   "db.orders",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.9,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/repository.go"},
				},
			},
		},
	}

	if _, err := compiler.CompileC4Diagrams(entities, edges); err != nil {
		t.Fatalf("compile c4 diagrams: %v", err)
	}

	contextContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/c4-context.mmd"))
	if err != nil {
		t.Fatalf("read context diagram: %v", err)
	}
	contextText := string(contextContent)

	if strings.Contains(contextText, "Gap: no evidence-backed relationships") {
		t.Fatalf("did not expect relationship gap with internal evidence-backed edges, got:\n%s", contextText)
	}
	for _, expected := range []string{
		"Evidence-backed workspace internals",
		"Service: orders",
		"Service: kitchen",
		"Datastore: orders-db",
		"ctx_svc_orders -->|calls| ctx_svc_kitchen",
		"ctx_svc_orders -->|uses| ctx_db_orders",
	} {
		if !strings.Contains(contextText, expected) {
			t.Fatalf("expected %q in context diagram, got:\n%s", expected, contextText)
		}
	}
}

func TestCompileC4DiagramsDeduplicatesEntityIDs(t *testing.T) {
	t.Parallel()

	ws := writeReportsWorkspace(t)
	compiler := NewCompiler(ws)

	entities := []contracts.Entity{
		{
			ID:   "svc.orders",
			Type: "service",
			Name: "orders",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/main.go"},
				},
			},
		},
		{
			ID:   "svc.orders",
			Type: "service",
			Name: "orders duplicate",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/handler.go"},
				},
			},
		},
		{
			ID:   "db.orders",
			Type: "datastore",
			Name: "orders-db",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 1,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "db/orders.sql"},
				},
			},
		},
	}
	edges := []contracts.Edge{
		{
			ID:   "edge.orders.db",
			Type: "uses",
			From: "svc.orders",
			To:   "db.orders",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.9,
				Evidence: []contracts.Evidence{
					{Repo: "sample", Path: "services/orders/repository.go"},
				},
			},
		},
	}

	if _, err := compiler.CompileC4Diagrams(entities, edges); err != nil {
		t.Fatalf("compile c4 diagrams: %v", err)
	}

	containerContent, err := os.ReadFile(filepath.Join(ws.Path, "reports/diagrams/c4-container.mmd"))
	if err != nil {
		t.Fatalf("read container diagram: %v", err)
	}
	containerText := string(containerContent)
	if got := strings.Count(containerText, `svc_svc_orders["Service:`); got != 1 {
		t.Fatalf("expected one service node for duplicate entity id, got %d:\n%s", got, containerText)
	}
}

func writeReportsWorkspace(t *testing.T) workspace.Root {
	t.Helper()

	root := t.TempDir()
	manifest := strings.TrimSpace(`
version: 1
repos:
  - name: sample
    path: /tmp/sample
`) + "\n"
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	return ws
}
