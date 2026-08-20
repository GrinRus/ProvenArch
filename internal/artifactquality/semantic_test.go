package artifactquality

import (
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func semanticEntity(id string) contracts.Entity {
	return contracts.Entity{ID: id, Type: "service", Name: id, Provenance: contracts.Provenance{Kind: "observation", Confidence: 0.8}}
}

func TestValidateSemanticEnvelopeRejectsDanglingGraph(t *testing.T) {
	snapshot := contracts.SemanticSnapshot{
		Entities:  []contracts.Entity{semanticEntity("svc.api")},
		Edges:     []contracts.Edge{{ID: "edge.api", Type: "calls", From: "svc.api", To: "svc.missing"}},
		Findings:  []contracts.Finding{{ID: "finding.api", Title: "gap", RelatedIDs: []string{"finding.missing"}}},
		Questions: []contracts.Question{{ID: "q.api", Text: "question", RelatedIDs: []string{"q.missing"}}},
	}
	err := ValidateSemanticEnvelope(snapshot)
	if err == nil || !strings.Contains(err.Error(), "dangling to endpoint") {
		t.Fatalf("expected dangling edge semantic issue, got %v", err)
	}
}

func TestValidateSemanticEnvelopeRejectsInvalidOwnerTeam(t *testing.T) {
	for name, owner := range map[string]string{
		"wrong prefix": "org.platform",
		"missing team": "team.platform",
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := contracts.SemanticSnapshot{Entities: []contracts.Entity{{ID: "svc.api", Type: "service", OwnerTeamID: owner}, semanticEntity("team.other")}}
			if err := ValidateSemanticEnvelope(snapshot); err == nil {
				t.Fatalf("expected invalid owner team %q", owner)
			}
		})
	}
	valid := contracts.SemanticSnapshot{Entities: []contracts.Entity{{ID: "svc.api", Type: "service", OwnerTeamID: "team.platform"}, {ID: "team.platform", Type: "team", Name: "Platform"}}}
	if err := ValidateSemanticEnvelope(valid); err != nil {
		t.Fatalf("expected valid owner team, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsRejectsCrossShardIdentityCollision(t *testing.T) {
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{semanticEntity("svc.api")}},
		contracts.SemanticSnapshot{Findings: []contracts.Finding{{ID: "svc.api", Title: "collision"}}},
	); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("expected cross-shard collision, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsIdenticalShardReplayButRejectsConflictingPayload(t *testing.T) {
	first := contracts.SemanticSnapshot{Entities: []contracts.Entity{semanticEntity("svc.api")}}
	if err := ValidateSemanticIDCollisions(first, first); err != nil {
		t.Fatalf("identical shard replay should be idempotent, got %v", err)
	}
	conflicting := contracts.SemanticSnapshot{Entities: []contracts.Entity{{ID: "svc.api", Type: "service", Name: "different"}}}
	if err := ValidateSemanticIDCollisions(first, conflicting); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("expected conflicting same-kind payload rejection, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsSameRepoEntityObservations(t *testing.T) {
	left := contracts.Entity{
		ID:   "svc.bank.accounts-db",
		Type: "database",
		Name: "accounts-db",
		Provenance: contracts.Provenance{
			Kind:     "observation",
			Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: "README.md"}},
		},
	}
	right := contracts.Entity{
		ID:   "svc.bank.accounts-db",
		Type: "datastore",
		Name: "Accounts PostgreSQL database",
		Provenance: contracts.Provenance{
			Kind:     "observation",
			Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: "src/accounts/accounts-db/README.md"}},
		},
	}
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left}},
		contracts.SemanticSnapshot{Entities: []contracts.Entity{right}},
	); err != nil {
		t.Fatalf("same-repo exact entity observations should merge during normalization, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsRejectsSameRepoUnrelatedEntityObservation(t *testing.T) {
	left := contracts.Entity{
		ID:   "svc.bank.accounts-db",
		Type: "service",
		Name: "Accounts",
		Provenance: contracts.Provenance{
			Kind:     "observation",
			Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: "README.md"}},
		},
	}
	right := left
	right.Name = "Payments API"
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Entities: []contracts.Entity{left}},
		contracts.SemanticSnapshot{Entities: []contracts.Entity{right}},
	); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("unrelated same-repo exact ID should remain a collision, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsSameRepoWeakEdgeIDRekey(t *testing.T) {
	left := contracts.Edge{
		ID:   "edge.balance-reader.reads-ledger-db",
		Type: "reads-from",
		From: "service.balance-reader",
		To:   "service.ledger-db",
		Provenance: contracts.Provenance{
			Kind:     "observation",
			Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: "README.md"}},
		},
	}
	right := left
	right.From = "svc.bank.of.anthos.src.ledger.balancereader"
	right.To = "db.bank.of.anthos.ledger-db"
	right.Provenance.Evidence = []contracts.Evidence{{Repo: "bank-of-anthos", Path: "src/ledger/balancereader/README.md"}}
	if err := ValidateSemanticIDCollisions(
		contracts.SemanticSnapshot{Edges: []contracts.Edge{left}},
		contracts.SemanticSnapshot{Edges: []contracts.Edge{right}},
	); err != nil {
		t.Fatalf("same-repo weak edge IDs should be rekeyed during normalization, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsCanonicalIDTypeFamilies(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: path}}}
	}
	observations := []contracts.SemanticSnapshot{
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "service", Name: "Bank of Anthos", Provenance: evidence("README.md")}}},
		{Entities: []contracts.Entity{{ID: "svc.bank.of.anthos", Type: "application", Name: "Bank of Anthos", Provenance: evidence("README.md")}}},
		{Entities: []contracts.Entity{{ID: "db.bank.of.anthos.accounts", Type: "stateful-workload", Name: "accounts-db PostgreSQL StatefulSet", Provenance: evidence("kubernetes-manifests/accounts-db.yaml")}}},
		{Entities: []contracts.Entity{{ID: "db.bank.of.anthos.accounts", Type: "datastore", Name: "Accounts database", Provenance: evidence("README.md")}}},
		{Entities: []contracts.Entity{{ID: "team.bank.of.anthos.default-maintainers", Type: "team", Name: "maintainers", Provenance: evidence(".github/CODEOWNERS")}}},
		{Entities: []contracts.Entity{{ID: "team.bank.of.anthos.default-maintainers", Type: "owner-group", Name: "GoogleCloudPlatform maintainers", Provenance: evidence(".github/CODEOWNERS")}}},
	}
	if err := ValidateSemanticIDCollisions(observations...); err != nil {
		t.Fatalf("canonical ID type families should merge, got %v", err)
	}
}

func TestValidateSemanticIDCollisionsAllowsExternalSystemProductAliases(t *testing.T) {
	evidence := func(path string) contracts.Provenance {
		return contracts.Provenance{Kind: "observation", Evidence: []contracts.Evidence{{Repo: "bank-of-anthos", Path: path}}}
	}
	observations := []contracts.SemanticSnapshot{
		{Entities: []contracts.Entity{{ID: "external.system.bank.of.anthos.gke", Type: "external.system", Name: "Google Kubernetes Engine", Provenance: evidence("docs/ci-cd-pipeline.md")}}},
		{Entities: []contracts.Entity{{ID: "external.system.bank.of.anthos.gke", Type: "external.system", Name: "Google Cloud GKE and Anthos platform", Provenance: evidence("iac/tf-anthos-gke/README.md")}}},
	}
	if err := ValidateSemanticIDCollisions(observations...); err != nil {
		t.Fatalf("external-system product name/acronym aliases should merge, got %v", err)
	}
	conflicting := observations[1]
	conflicting.Entities[0].Name = "Cloud SQL"
	if err := ValidateSemanticIDCollisions(observations[0], conflicting); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("unrelated external-system names should remain a collision, got %v", err)
	}
}

func TestValidateSemanticEnvelopeJSONRejectsUnknownNestedFields(t *testing.T) {
	raw := []byte(`{"semantic":{"coverage":{"observed":[],"missing":[],"notes":[]},"questions":[],"entities":[{"id":"svc.api","type":"service","name":"API","provenance":{"kind":"observation","confidence":0.8,"unexpected":true}}],"edges":[],"findings":[]}}`)
	if err := ValidateSemanticEnvelopeJSON(raw); err == nil || !strings.Contains(err.Error(), `unknown field "unexpected"`) {
		t.Fatalf("expected unknown semantic field rejection, got %v", err)
	}
}
