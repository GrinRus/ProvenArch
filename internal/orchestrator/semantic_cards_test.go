package orchestrator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/model"
)

func TestEnrichCanonicalCardsAddsIdempotentDerivedSections(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/payments-service.md", []byte(strings.Join([]string{
		"# Payments Domain",
		"",
		"- id: payments-service",
		"- repo_scope: payments-service",
		"",
		"Human-authored domain notes stay intact.",
		"",
	}, "\n"))); err != nil {
		t.Fatalf("write domain card: %v", err)
	}
	if err := ws.WriteFile("charter/cards/teams/platform.md", []byte(strings.Join([]string{
		"# Platform Team",
		"",
		"- id: team.platform",
		"",
		"Human-authored team notes stay intact.",
		"",
	}, "\n"))); err != nil {
		t.Fatalf("write team card: %v", err)
	}

	store := model.NewStore(ws)
	if _, err := store.ApplySemanticSnapshot(contracts.SemanticSnapshot{
		Entities: []contracts.Entity{
			{
				ID:   "team.platform",
				Type: "team",
				Name: "Platform",
				Provenance: contracts.Provenance{
					Kind:       "assertion",
					Confidence: 1,
				},
			},
			{
				ID:          "svc.payments",
				Type:        "service",
				Name:        "Payments Service",
				OwnerTeamID: "team.platform",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.9,
					Evidence: []contracts.Evidence{
						{Repo: "payments-service", Path: "README.md"},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("apply semantic snapshot: %v", err)
	}

	coverage := contracts.Coverage{Missing: []string{"owner escalation path"}}
	execution := &pipelineExecution{
		workspace: ws,
		store:     store,
		pipelineSemanticDocflowState: pipelineSemanticDocflowState{
			findings: []contracts.Finding{
				{
					ID:         "f.payments.owner",
					Severity:   "medium",
					Title:      "Payments ownership needs review",
					RelatedIDs: []string{"svc.payments"},
				},
			},
			questions: []contracts.Question{
				{
					ID:         "q.payments.slo",
					Text:       "Which SLO owns payments?",
					Priority:   "medium",
					RelatedIDs: []string{"svc.payments"},
				},
			},
			coverage: &coverage,
		},
	}

	teamCards := []canonicalTeamCard{{Slug: "platform", TeamID: "team.platform"}}
	for i := 0; i < 2; i++ {
		if err := execution.enrichCanonicalCards([]string{"payments-service"}, teamCards); err != nil {
			t.Fatalf("enrich canonical cards pass %d: %v", i+1, err)
		}
	}

	domainCard := readWorkspaceFileForTest(t, ws.Path, "charter/cards/domains/payments-service.md")
	assertContainsText(t, domainCard, "Human-authored domain notes stay intact.")
	assertContainsText(t, domainCard, "## Derived (ACP Step1)")
	assertContainsText(t, domainCard, "- repo_scope: `payments-service`")
	assertContainsText(t, domainCard, "- related_entities: `svc.payments`")
	assertContainsText(t, domainCard, "- related_findings: `f.payments.owner`")
	assertContainsText(t, domainCard, "- open_questions: `q.payments.slo`")
	assertContainsText(t, domainCard, "- coverage_missing: owner escalation path")
	assertContainsText(t, domainCard, "  - `payments-service:README.md`")
	if got := strings.Count(domainCard, "## Derived (ACP Step1)"); got != 1 {
		t.Fatalf("expected one domain derived section, got %d:\n%s", got, domainCard)
	}

	teamCard := readWorkspaceFileForTest(t, ws.Path, "charter/cards/teams/platform.md")
	assertContainsText(t, teamCard, "Human-authored team notes stay intact.")
	assertContainsText(t, teamCard, "- team_id: `team.platform`")
	assertContainsText(t, teamCard, "- related_services: `svc.payments`")
	assertContainsText(t, teamCard, "- related_findings: `f.payments.owner`")
	assertContainsText(t, teamCard, "- open_questions: `q.payments.slo`")
	assertContainsText(t, teamCard, "  - `payments-service:README.md`")
	if got := strings.Count(teamCard, "## Derived (ACP Step1)"); got != 1 {
		t.Fatalf("expected one team derived section, got %d:\n%s", got, teamCard)
	}
}

func TestEnrichCanonicalCardsDoesNotCreateMissingTeamCards(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/payments-service.md", []byte(strings.Join([]string{
		"# Payments Domain",
		"",
		"- id: payments-service",
		"- repo_scope: payments-service",
		"",
	}, "\n"))); err != nil {
		t.Fatalf("write domain card: %v", err)
	}

	store := model.NewStore(ws)
	if _, err := store.ApplySemanticSnapshot(contracts.SemanticSnapshot{
		Entities: []contracts.Entity{
			{
				ID:   "team.shadow",
				Type: "team",
				Name: "Shadow Team",
				Provenance: contracts.Provenance{
					Kind:       "assertion",
					Confidence: 0.7,
				},
			},
			{
				ID:          "svc.shadow-payments",
				Type:        "service",
				Name:        "Shadow Payments",
				OwnerTeamID: "team.shadow",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.8,
					Evidence: []contracts.Evidence{
						{Repo: "payments-service", Path: "shadow/README.md"},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("apply semantic snapshot: %v", err)
	}

	execution := &pipelineExecution{
		workspace: ws,
		store:     store,
	}
	if err := execution.enrichCanonicalCards([]string{"payments-service"}, nil); err != nil {
		t.Fatalf("enrich canonical cards: %v", err)
	}

	missingTeamPath := filepath.Join(ws.Path, "charter", "cards", "teams", "shadow.md")
	if _, err := os.Stat(missingTeamPath); err == nil {
		t.Fatalf("missing canonical team card was auto-created at %s", missingTeamPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat missing team card: %v", err)
	}
	if len(execution.questions) != 1 {
		t.Fatalf("expected one missing-team question, got %+v", execution.questions)
	}
	if got := execution.questions[0].ID; got != "q.team.team-shadow.missing-canonical-card" {
		t.Fatalf("unexpected missing-team question id: %q", got)
	}
}

func TestInitPipelineEnrichesCanonicalCardsAfterStep1(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithClock(func() time.Time {
			return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	domainCards, err := filepath.Glob(filepath.Join(ws.Path, "charter", "cards", "domains", "*.md"))
	if err != nil {
		t.Fatalf("glob domain cards: %v", err)
	}
	if len(domainCards) == 0 {
		t.Fatalf("expected init pipeline to create canonical domain cards")
	}
	for _, path := range domainCards {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read domain card %s: %v", path, err)
		}
		if got := strings.Count(string(content), "## Derived (ACP Step1)"); got != 1 {
			t.Fatalf("expected one derived section in %s, got %d:\n%s", path, got, string(content))
		}
	}

	teamCards, err := filepath.Glob(filepath.Join(ws.Path, "charter", "cards", "teams", "*.md"))
	if err != nil {
		t.Fatalf("glob team cards: %v", err)
	}
	if len(teamCards) == 0 {
		t.Fatalf("expected init pipeline to create canonical team cards")
	}
	for _, path := range teamCards {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read team card %s: %v", path, err)
		}
		if got := strings.Count(string(content), "## Derived (ACP Step1)"); got != 1 {
			t.Fatalf("expected one derived section in %s, got %d:\n%s", path, got, string(content))
		}
	}
}

func readWorkspaceFileForTest(t *testing.T, root string, relPath string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	return string(content)
}

func assertContainsText(t *testing.T, content string, want string) {
	t.Helper()

	if !strings.Contains(content, want) {
		t.Fatalf("expected content to contain %q:\n%s", want, content)
	}
}
