package qa

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestAskReturnsCitationsFromWorkspaceArtifacts(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	response, err := service.Ask(context.Background(), ws, "What does coverage say about owner mappings and architecture notes?")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if response.Confidence <= 0 {
		t.Fatalf("expected positive confidence")
	}
	if len(response.Citations) == 0 {
		t.Fatalf("expected citations")
	}
	if !strings.Contains(strings.ToLower(response.Answer), "artifact") {
		t.Fatalf("expected evidence-based answer, got %q", response.Answer)
	}
	if !hasCitationPathPrefix(response.Citations, "reports/coverage/") {
		t.Fatalf("expected coverage citation, got %+v", response.Citations)
	}
	if !hasCitationPathPrefix(response.Citations, "docs/imports/") {
		t.Fatalf("expected docs/imports citation, got %+v", response.Citations)
	}
}

func TestAskHandlesMissingEvidence(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	response, err := service.Ask(context.Background(), ws, "completely unrelated quantum topic")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if len(response.Citations) != 0 {
		t.Fatalf("expected no citations, got %+v", response.Citations)
	}
	if len(response.Unresolved) == 0 {
		t.Fatalf("expected unresolved notes")
	}
}

func TestAskIncludesModelArtifactsInIndex(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	response, err := service.Ask(context.Background(), ws, "Which service has owner team platform?")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if len(response.Citations) == 0 {
		t.Fatalf("expected citations")
	}
	if !hasCitationPathPrefix(response.Citations, "model/entities/") {
		t.Fatalf("expected model citation, got %+v", response.Citations)
	}
}

func TestAskCitationsAreDeduplicatedAndConfidenceBounded(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	response, err := service.Ask(context.Background(), ws, "owner owner owner payments payments architecture notes missing-token-alpha missing-token-beta missing-token-gamma missing-token-delta")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if hasDuplicateCitationPaths(response.Citations) {
		t.Fatalf("expected deduplicated citations, got %+v", response.Citations)
	}
	if len(response.Unresolved) > 3 {
		t.Fatalf("expected unresolved reasons capped at 3, got %d (%+v)", len(response.Unresolved), response.Unresolved)
	}
	if response.Confidence < 0 || response.Confidence > 1 {
		t.Fatalf("expected confidence in [0,1], got %f", response.Confidence)
	}
}

func TestAskConfidenceIsPenalizedWhenUnresolvedReasonsIncrease(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	baseline, err := service.Ask(context.Background(), ws, "owner payments architecture notes")
	if err != nil {
		t.Fatalf("ask baseline: %v", err)
	}
	withUnknowns, err := service.Ask(context.Background(), ws, "owner payments architecture notes missing-alpha missing-beta missing-gamma")
	if err != nil {
		t.Fatalf("ask with unresolved tokens: %v", err)
	}

	if len(withUnknowns.Unresolved) == 0 {
		t.Fatalf("expected unresolved reasons for unknown tokens")
	}
	if withUnknowns.Confidence >= baseline.Confidence {
		t.Fatalf("expected confidence penalty for unresolved reasons: baseline=%f with_unresolved=%f", baseline.Confidence, withUnknowns.Confidence)
	}
}

func TestAskRanksFindingsBeforeImportedDocsWhenKeywordsOverlap(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.WriteFile("reports/findings/findings.md", []byte("Critical integration risk for payments owner mapping.")); err != nil {
		t.Fatalf("write findings fixture: %v", err)
	}
	if err := ws.WriteFile("docs/imports/extra-notes.md", []byte("Critical integration risk was discussed in imported notes.")); err != nil {
		t.Fatalf("write docs fixture: %v", err)
	}

	service := NewService()
	response, err := service.Ask(context.Background(), ws, "critical integration risk payments")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if len(response.Citations) == 0 {
		t.Fatalf("expected citations")
	}
	if !strings.HasPrefix(response.Citations[0].Path, "reports/findings/") {
		t.Fatalf("expected highest-ranked citation from findings report, got %+v", response.Citations)
	}
}

func TestAskUsesConfiguredDocsImportsPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "payments-service")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	manifest := "version: 1\nrepos:\n  - name: payments-service\n    path: " + repoPath + "\ndocs:\n  imports_path: ./external/imports\n"
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.WriteFile("external/imports/index.yaml", []byte("- id: external-architecture-notes\n  path: external/imports/architecture-notes.md\n")); err != nil {
		t.Fatalf("write custom imports index: %v", err)
	}
	if err := ws.WriteFile("external/imports/architecture-notes.md", []byte("External architecture notes mention routing ownership.")); err != nil {
		t.Fatalf("write custom import: %v", err)
	}
	if err := ws.WriteFile("docs/imports/legacy-notes.md", []byte("Legacy notes mention routing ownership but should not be indexed for custom imports path.")); err != nil {
		t.Fatalf("write legacy import: %v", err)
	}

	service := NewService()
	response, err := service.Ask(context.Background(), ws, "routing ownership architecture")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if !hasCitationPathPrefix(response.Citations, "external/imports/") {
		t.Fatalf("expected custom imports citation, got %+v", response.Citations)
	}
	if hasCitationPathPrefix(response.Citations, "docs/imports/") {
		t.Fatalf("expected hardcoded docs/imports not to be indexed when custom imports path is configured, got %+v", response.Citations)
	}
}

func TestAskIncludesExplainableReasons(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	response, err := service.Ask(context.Background(), ws, "payments owner architecture missing-token")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if len(response.Citations) == 0 {
		t.Fatalf("expected citations")
	}
	for _, citation := range response.Citations {
		if !strings.Contains(citation.Reason, "matched") {
			t.Fatalf("expected explainable citation reason, got %+v", citation)
		}
	}
	if len(response.Unresolved) == 0 {
		t.Fatalf("expected unresolved reasons")
	}
	if !strings.Contains(response.Unresolved[0], "no supporting evidence for keyword") {
		t.Fatalf("expected unresolved reason format, got %+v", response.Unresolved)
	}
}

func TestAskIsDeterministicForSameQuestion(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	first, err := service.Ask(context.Background(), ws, "owner mappings architecture notes")
	if err != nil {
		t.Fatalf("first ask: %v", err)
	}
	second, err := service.Ask(context.Background(), ws, "owner mappings architecture notes")
	if err != nil {
		t.Fatalf("second ask: %v", err)
	}
	if first.Answer != second.Answer {
		t.Fatalf("expected deterministic answer, got %q vs %q", first.Answer, second.Answer)
	}
	if first.Confidence != second.Confidence {
		t.Fatalf("expected deterministic confidence, got %f vs %f", first.Confidence, second.Confidence)
	}
	if len(first.Citations) != len(second.Citations) {
		t.Fatalf("expected deterministic citation count, got %d vs %d", len(first.Citations), len(second.Citations))
	}
	for idx := range first.Citations {
		if first.Citations[idx] != second.Citations[idx] {
			t.Fatalf("expected deterministic citations, got %+v vs %+v", first.Citations, second.Citations)
		}
	}
}

func createWorkspace(t *testing.T) workspace.Root {
	t.Helper()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "payments-service")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	manifest := "version: 1\nrepos:\n  - name: payments-service\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}

	fixtures := map[string]string{
		"charter/overview.md":                "Payments domain overview.",
		"reports/coverage/summary.md":        "Missing: owner mappings",
		"reports/coverage/open-questions.md": "Who owns payments deployment?",
		"reports/as-is/overview.md":          "Services: payments-service",
		"docs/imports/index.yaml":            "- id: architecture-notes\n  path: docs/imports/architecture-notes.md\n",
		"docs/imports/architecture-notes.md": "Architecture notes mention owner mappings and deployment concerns.",
		"model/entities/svc.payments.yaml": `id: svc.payments
type: service
name: Payments Service
owner_team_id: team.platform
provenance:
  kind: assertion
  confidence: 1`,
	}
	for path, content := range fixtures {
		if err := ws.WriteFile(path, []byte(content)); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}

	return ws
}

func hasCitationPathPrefix(citations []Citation, prefix string) bool {
	for _, citation := range citations {
		if strings.HasPrefix(citation.Path, prefix) {
			return true
		}
	}
	return false
}

func hasDuplicateCitationPaths(citations []Citation) bool {
	seen := map[string]struct{}{}
	for _, citation := range citations {
		if _, exists := seen[citation.Path]; exists {
			return true
		}
		seen[citation.Path] = struct{}{}
	}
	return false
}
