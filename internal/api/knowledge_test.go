package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func TestKnowledgeUnavailableWhenPromotedWorkspaceIsEmpty(t *testing.T) {
	server := newTestServer(t)
	recorder := httptest.NewRecorder()
	server.handleKnowledge(recorder, httptest.NewRequest(http.MethodGet, "/api/knowledge", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response knowledgeResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "unavailable" || response.SourceMode != evidenceAuthorityPromotedCurrent {
		t.Fatalf("unexpected source/status: %#v", response)
	}
	if len(response.Entities)+len(response.Edges)+len(response.Artifacts)+len(response.Issues) != 0 {
		t.Fatalf("empty workspace must not invent knowledge: %#v", response)
	}
}

func TestArchitectureReviewLinksAndChildDetailAreExplicit(t *testing.T) {
	knowledge := knowledgeResponse{Status: "available", Entities: []knowledgeEntity{
		{Entity: contracts.Entity{ID: "svc.payments", Type: "service", Name: "Payments", Provenance: contracts.Provenance{Kind: "observation", Confidence: .9, Evidence: []contracts.Evidence{{Repo: "payments", Path: "README.md"}}}}, Path: "model/entities/svc.payments.yaml"},
		{Entity: contracts.Entity{ID: "api.payments", Type: "api.http", Name: "Payments API", Provenance: contracts.Provenance{Kind: "observation", Confidence: .8, Evidence: []contracts.Evidence{{Repo: "payments", Path: "api/openapi.yaml"}}}}, Path: "model/entities/api.payments.yaml"},
	}, Edges: []knowledgeEdge{}, Artifacts: []knowledgeArtifact{}, Issues: []knowledgeIssue{}}
	response := buildArchitectureResponse(knowledge)
	enrichArchitectureReview(&response, contracts.SemanticSnapshot{
		Coverage:  contracts.Coverage{Observed: []string{"http api"}, Missing: []string{"owner"}},
		Findings:  []contracts.Finding{{ID: "finding.owner", Severity: "medium", Title: "Owner missing", RelatedIDs: []string{"svc.payments"}}},
		Questions: []contracts.Question{{ID: "question.owner", Text: "Who owns Payments?", RelatedIDs: []string{"svc.payments"}}},
	})
	node := response.Views["context"].Nodes[0]
	if len(node.RelatedFindings) != 1 || len(node.RelatedQuestions) != 1 || !architectureContainsString(node.ChildLevels, "component") || !architectureContainsString(node.ChildLevels, "code") {
		t.Fatalf("architecture review/detail was not linked: %#v", node)
	}
	if len(response.Review.Findings) != 1 || len(response.Coverage.Missing) != 1 {
		t.Fatalf("top-level review/coverage missing: %#v", response)
	}
}

func TestKnowledgeReturnsValidatedEntitiesEdgesAndArtifacts(t *testing.T) {
	server := newTestServer(t)
	root := server.getWorkspace().Path
	writeKnowledgeTestFile(t, root, "model/entities/svc.payments.yaml", "id: svc.payments\ntype: service\nname: Payments\nprovenance:\n  kind: inference\n  confidence: 0.9\n")
	writeKnowledgeTestFile(t, root, "model/entities/svc.users.yaml", "id: svc.users\ntype: service\nname: Users\nprovenance:\n  kind: inference\n  confidence: 0.8\n")
	writeKnowledgeTestFile(t, root, "model/edges/edge.payments.calls.users.yaml", "id: edge.payments.calls.users\ntype: calls\nfrom: svc.payments\nto: svc.users\nprovenance:\n  kind: inference\n  confidence: 0.7\n")
	writeKnowledgeTestFile(t, root, "reports/as-is/overview.md", "# Overview\n")
	writeKnowledgeTestFile(t, root, "reports/taskruns/qa-old/qa/qa-answer.json", `{"answer":"must stay out of promoted Knowledge"}`)

	recorder := httptest.NewRecorder()
	server.handleKnowledge(recorder, httptest.NewRequest(http.MethodGet, "/api/knowledge", nil))
	var response knowledgeResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "available" || len(response.Entities) != 2 || len(response.Edges) != 1 {
		t.Fatalf("unexpected knowledge response: %#v", response)
	}
	if response.Entities[0].Path != "model/entities/svc.payments.yaml" || response.Edges[0].Path != "model/edges/edge.payments.calls.users.yaml" {
		t.Fatalf("paths are not canonical workspace paths: %#v %#v", response.Entities, response.Edges)
	}
	if len(response.Artifacts) != 4 {
		t.Fatalf("artifact inventory length = %d, want 4", len(response.Artifacts))
	}
	for _, artifact := range response.Artifacts {
		if strings.HasPrefix(artifact.Path, "reports/taskruns/") {
			t.Fatalf("taskrun artifact leaked into promoted Knowledge: %#v", artifact)
		}
	}
}

func TestKnowledgeKeepsValidDataWhenFilesAreMalformedOrReferencesBreak(t *testing.T) {
	server := newTestServer(t)
	root := server.getWorkspace().Path
	writeKnowledgeTestFile(t, root, "model/entities/svc.valid.yaml", "id: svc.valid\ntype: service\nname: Valid\nprovenance:\n  kind: inference\n  confidence: 1\n")
	writeKnowledgeTestFile(t, root, "model/entities/broken.yaml", "id: [not-valid\n")
	writeKnowledgeTestFile(t, root, "model/edges/broken-reference.yaml", "id: edge.valid.calls.missing\ntype: calls\nfrom: svc.valid\nto: svc.missing\nprovenance:\n  kind: inference\n  confidence: 0.5\n")

	recorder := httptest.NewRecorder()
	server.handleKnowledge(recorder, httptest.NewRequest(http.MethodGet, "/api/knowledge", nil))
	var response knowledgeResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "partial" || len(response.Entities) != 1 || len(response.Edges) != 0 {
		t.Fatalf("partial response did not retain valid knowledge: %#v", response)
	}
	codes := map[string]bool{}
	for _, issue := range response.Issues {
		codes[issue.Code] = true
	}
	if !codes["knowledge.entity_malformed"] || !codes["knowledge.edge_reference_missing"] {
		t.Fatalf("missing typed issues: %#v", response.Issues)
	}
}

func TestKnowledgeRejectsMutatingMethods(t *testing.T) {
	server := newTestServer(t)
	recorder := httptest.NewRecorder()
	server.handleKnowledge(recorder, httptest.NewRequest(http.MethodPost, "/api/knowledge", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestArchitectureReturnsLevelViewsAndEvidenceAuthority(t *testing.T) {
	server := newTestServer(t)
	root := server.getWorkspace().Path
	writeKnowledgeTestFile(t, root, "model/entities/svc.payments.yaml", "id: svc.payments\ntype: service\nname: Payments\nprovenance:\n  kind: inference\n  confidence: 0.9\n")
	writeKnowledgeTestFile(t, root, "model/entities/ext.bank.yaml", "id: ext.bank\ntype: external.system\nname: Bank\nprovenance:\n  kind: inference\n  confidence: 0.8\n")
	writeKnowledgeTestFile(t, root, "model/edges/edge.payments.calls.bank.yaml", "id: edge.payments.calls.bank\ntype: calls\nfrom: svc.payments\nto: ext.bank\nprovenance:\n  kind: inference\n  confidence: 0.7\n")
	recorder := httptest.NewRecorder()
	server.handleArchitecture(recorder, httptest.NewRequest(http.MethodGet, "/api/architecture", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response architectureResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode architecture: %v", err)
	}
	if response.Authority.Mode != evidenceAuthorityPromotedCurrent || response.Status != "available" {
		t.Fatalf("unexpected authority/status: %#v", response)
	}
	if !response.Views["context"].Available || len(response.Views["context"].Nodes) != 2 || len(response.Views["context"].Edges) != 1 {
		t.Fatalf("unexpected context view: %#v", response.Views["context"])
	}
	if response.Counts.Entities != 2 || response.Counts.Edges != 1 {
		t.Fatalf("unexpected counts: %#v", response.Counts)
	}
}

func TestArchitectureComparisonClassifiesSemanticAndReviewChanges(t *testing.T) {
	root := t.TempDir()
	currentRoot := filepath.Join(root, "reports", "taskruns", "run-current", "promoted-snapshot")
	baselineRoot := filepath.Join(root, "reports", "taskruns", "run-baseline", "promoted-snapshot")
	writeKnowledgeTestFile(t, currentRoot, "model/entities/svc.payments.yaml", "id: svc.payments\ntype: service\nname: Payments v2\nprovenance:\n  kind: inference\n  confidence: 0.9\n")
	writeKnowledgeTestFile(t, baselineRoot, "model/entities/svc.payments.yaml", "id: svc.payments\ntype: service\nname: Payments\nprovenance:\n  kind: inference\n  confidence: 0.9\n")
	writeKnowledgeTestFile(t, currentRoot, "model/entities/svc.users.yaml", "id: svc.users\ntype: service\nname: Users\nprovenance:\n  kind: inference\n  confidence: 0.8\n")
	writeKnowledgeTestFile(t, baselineRoot, "model/entities/svc.legacy.yaml", "id: svc.legacy\ntype: service\nname: Legacy\nprovenance:\n  kind: inference\n  confidence: 0.7\n")
	writeKnowledgeTestFile(t, currentRoot, "architecture-snapshot.json", `{"files":[{"path":"reports/findings/findings.md","sha256":"new"},{"path":"reports/coverage/summary.md","sha256":"same"}]}`)
	writeKnowledgeTestFile(t, baselineRoot, "architecture-snapshot.json", `{"files":[{"path":"reports/findings/findings.md","sha256":"old"},{"path":"reports/coverage/summary.md","sha256":"same"}]}`)
	comparison := comparePromotedArchitectures(root, "run-current", "run-baseline")
	entities := comparison.Categories["entities"]
	if !comparison.Available || len(entities.Added) != 1 || len(entities.Changed) != 1 || len(entities.Removed) != 1 {
		t.Fatalf("unexpected entity comparison: %#v", comparison)
	}
	if len(comparison.Categories["findings"].Changed) != 1 || len(comparison.Categories["gaps"].Changed) != 0 {
		t.Fatalf("unexpected review comparison: %#v", comparison.Categories)
	}
}

func TestKnowledgeFixtureMatchesWireContract(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "api", "knowledge-current-workspace.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var response knowledgeResponse
	if err := json.Unmarshal(content, &response); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if response.Version != 1 || response.SourceMode != evidenceAuthorityPromotedCurrent || response.Status != "partial" {
		t.Fatalf("fixture identity is invalid: %#v", response)
	}
	if len(response.Entities) != 1 || len(response.Artifacts) != 2 || len(response.Issues) != 1 {
		t.Fatalf("fixture coverage is incomplete: %#v", response)
	}
}

func writeKnowledgeTestFile(t *testing.T, root string, relative string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relative, err)
	}
}
