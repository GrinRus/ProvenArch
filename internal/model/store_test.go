package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
	"gopkg.in/yaml.v3"
)

func TestApplySemanticSnapshotWritesEntityAndEdgeFiles(t *testing.T) {
	t.Parallel()

	ws := writeWorkspaceRoot(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}

	store := NewStore(ws)
	report, err := store.ApplySemanticSnapshot(contracts.SemanticSnapshot{
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
		Edges: []contracts.Edge{
			{
				ID:   "edge.svc.payments.calls.svc.users",
				Type: "calls",
				From: "svc.payments",
				To:   "svc.users",
				Provenance: contracts.Provenance{
					Kind:       "inference",
					Confidence: 0.6,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("apply semantic snapshot: %v", err)
	}
	if report.UpsertedEntities != 2 {
		t.Fatalf("expected 2 upserted entities, got %d", report.UpsertedEntities)
	}
	if report.UpsertedEdges != 1 {
		t.Fatalf("expected 1 upserted edge, got %d", report.UpsertedEdges)
	}

	entityPath := filepath.Join(ws.Path, "model/entities/svc.payments.yaml")
	if _, err := os.Stat(entityPath); err != nil {
		t.Fatalf("expected entity file at %q: %v", entityPath, err)
	}
}

func TestApplySemanticSnapshotRejectsUnknownOwnerTeam(t *testing.T) {
	t.Parallel()

	ws := writeWorkspaceRoot(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}

	store := NewStore(ws)
	_, err := store.ApplySemanticSnapshot(contracts.SemanticSnapshot{
		Entities: []contracts.Entity{
			{
				ID:          "svc.payments",
				Type:        "service",
				Name:        "Payments Service",
				OwnerTeamID: "team.unknown",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.9,
					Evidence: []contracts.Evidence{
						{Repo: "payments-service", Path: "README.md"},
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected owner_team_id validation error")
	}
	if !strings.Contains(err.Error(), `owner_team_id "team.unknown"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplySemanticSnapshotRemapsCollidingEntityIDs(t *testing.T) {
	t.Parallel()

	ws := writeWorkspaceRoot(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	store := NewStore(ws)

	_, err := store.ApplySemanticSnapshot(contracts.SemanticSnapshot{
		Entities: []contracts.Entity{
			{
				ID:   "svc.orders",
				Type: "service",
				Name: "Orders",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.9,
					Evidence: []contracts.Evidence{
						{Repo: "orders-api", Path: "README.md"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("seed entity: %v", err)
	}

	report, err := store.ApplySemanticSnapshot(contracts.SemanticSnapshot{
		Entities: []contracts.Entity{
			{
				ID:   "svc.orders",
				Type: "service",
				Name: "Orders V2",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.9,
					Evidence: []contracts.Evidence{
						{Repo: "orders-monolith", Path: "README.md"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("apply colliding entity: %v", err)
	}

	remappedID, ok := report.RemappedIDs["svc.orders"]
	if !ok {
		t.Fatalf("expected remapped id for collision")
	}
	if !strings.HasPrefix(remappedID, "svc.orders.repo-") {
		t.Fatalf("unexpected remapped id: %q", remappedID)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "model/entities", remappedID+".yaml"))
	if err != nil {
		t.Fatalf("read remapped entity file: %v", err)
	}
	var entity contracts.Entity
	if err := yaml.Unmarshal(content, &entity); err != nil {
		t.Fatalf("unmarshal remapped entity: %v", err)
	}
	if !contains(entity.Aliases, "svc.orders") {
		t.Fatalf("expected aliases to include original id, got %+v", entity.Aliases)
	}
}

func TestApplySemanticSnapshotKeepsCanonicalIDOnRenameInSameRepo(t *testing.T) {
	t.Parallel()

	ws := writeWorkspaceRoot(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	store := NewStore(ws)

	_, err := store.ApplySemanticSnapshot(contracts.SemanticSnapshot{
		Entities: []contracts.Entity{
			{
				ID:   "svc.payments",
				Type: "service",
				Name: "Payments Service",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.9,
					Evidence: []contracts.Evidence{
						{Repo: "payments-service", Path: "README.md"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("seed entity: %v", err)
	}

	report, err := store.ApplySemanticSnapshot(contracts.SemanticSnapshot{
		Entities: []contracts.Entity{
			{
				ID:   "svc.payments",
				Type: "service",
				Name: "Payments API",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.9,
					Evidence: []contracts.Evidence{
						{Repo: "payments-service", Path: "internal/api/server.go"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("update entity: %v", err)
	}
	if len(report.RemappedIDs) != 0 {
		t.Fatalf("expected no remaps for same repo rename, got %+v", report.RemappedIDs)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "model/entities/svc.payments.yaml"))
	if err != nil {
		t.Fatalf("read canonical entity: %v", err)
	}
	var entity contracts.Entity
	if err := yaml.Unmarshal(content, &entity); err != nil {
		t.Fatalf("unmarshal canonical entity: %v", err)
	}
	if entity.Name != "Payments API" {
		t.Fatalf("expected entity rename update in-place, got %q", entity.Name)
	}
}

func writeWorkspaceRoot(t *testing.T) workspace.Root {
	t.Helper()

	root := t.TempDir()
	manifest := "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
