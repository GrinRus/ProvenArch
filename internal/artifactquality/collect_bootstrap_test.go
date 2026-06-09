package artifactquality

import (
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func TestCollectManifestBootstrapOnlyDetectsUnchangedFirstActionPair(t *testing.T) {
	t.Parallel()

	manifest := bootstrapCollectManifest()
	docs := map[string]string{
		"payments-overview.md": "# Payments Overview\n\n## Scope\n- Repository: payments\n\n## Observations\n- Repository scope: payments.\n- Primary scoped evidence path: `src`.\n\n## Evidence\n- Primary evidence path: `src`\n\n## Follow-up\n- Owner mapping evidence not confirmed from the initial scoped evidence path.\n",
	}

	if !CollectManifestBootstrapOnly(manifest, docs) {
		t.Fatalf("expected unchanged collect first-action pair to be classified as bootstrap-only")
	}
}

func TestCollectManifestBootstrapOnlyIgnoresEnrichedDocument(t *testing.T) {
	t.Parallel()

	manifest := bootstrapCollectManifest()
	docs := map[string]string{
		"payments-overview.md": "# Payments Overview\n\n## Observations\n- `src/payment_handler.go` defines the payment service entrypoint.\n- `src/ledger_client.go` calls the ledger write API when a payment is accepted.\n\n## Evidence\n- `src/payment_handler.go`\n- `src/ledger_client.go`\n",
	}

	if CollectManifestBootstrapOnly(manifest, docs) {
		t.Fatalf("did not expect an enriched document to be classified as bootstrap-only")
	}
}

func TestCollectManifestBootstrapOnlyDetectsMarkerEvenWithRichSemantic(t *testing.T) {
	t.Parallel()

	manifest := bootstrapCollectManifest()
	manifest.Semantic.Coverage.Missing = nil
	manifest.Semantic.Coverage.Notes = []string{"Observed payment API, ledger calls, and queue processing from concrete source files."}
	manifest.Semantic.Entities = append(manifest.Semantic.Entities, contracts.Entity{
		ID:   "svc.ledger",
		Type: "service",
		Name: "Ledger",
		Provenance: contracts.Provenance{
			Kind:       "observation",
			Confidence: 0.8,
			Evidence:   []contracts.Evidence{{Repo: "payments", Path: "src/ledger_client.go"}},
		},
	})
	manifest.Semantic.Edges = append(manifest.Semantic.Edges, contracts.Edge{
		ID:   "edge.payments.ledger.write",
		Type: "depends_on",
		From: "svc.payments.src",
		To:   "svc.ledger",
		Name: "writes accepted payments",
		Provenance: contracts.Provenance{
			Kind:       "observation",
			Confidence: 0.8,
			Evidence:   []contracts.Evidence{{Repo: "payments", Path: "src/ledger_client.go"}},
		},
	})
	manifest.Semantic.Findings = []contracts.Finding{{
		ID:          "finding.payments.retry.visibility",
		Severity:    "medium",
		Title:       "Payment retry visibility is split across handler and queue code",
		Description: "Operational retry behavior is split between the payment handler and queue worker.",
		RuleID:      "rule.operational.visibility",
		RelatedIDs:  []string{"svc.payments.src"},
		Provenance: contracts.Provenance{
			Kind:       "inference",
			Confidence: 0.7,
			Evidence:   []contracts.Evidence{{Repo: "payments", Path: "src/payment_handler.go"}},
		},
	}}
	docs := map[string]string{
		"payments-overview.md": "# Payments Overview\n\n<!-- " + CollectBootstrapReplaceMarker + " -->\n\n## Observations\n- `src/payment_handler.go` defines the payment service entrypoint.\n",
	}

	if !CollectManifestBootstrapOnly(manifest, docs) {
		t.Fatalf("expected marker-bearing collect document to be classified as bootstrap-only")
	}
}

func TestCollectDocumentBootstrapOnlyDetectsLowSignalRecoveryScaffold(t *testing.T) {
	t.Parallel()

	text := "# Payments Overview\n\n## Recovery Summary\n- Repository: payments\n- Assigned scope: src\n- Evidence candidate used for the recovery manifest: `src`.\n\n## Evidence Candidates\n- `src` is the first scoped repository path encoded in the recovery manifest.\n- Additional repository-specific details should be enriched by the provider when available within the repair window.\n\n## Remaining Questions\n- Confirm concrete ownership, runtime responsibilities, and operational escalation evidence for this shard.\n"
	if !CollectDocumentBootstrapOnly(text) {
		t.Fatalf("expected low-signal recovery scaffold to be classified as bootstrap-only")
	}
}

func TestCollectDocumentBootstrapOnlyDetectsMarkerFreeRecoveryBootstrap(t *testing.T) {
	t.Parallel()

	text := "# Payments Overview\n\n## Recovery Bootstrap\n- Repository: payments\n- Assigned scope: src\n- Evidence candidate used for the recovery manifest: `src`.\n\n## Required Enrichment\n- `src` is the first scoped repository path encoded in the recovery manifest.\n- Replace this recovery bootstrap with concrete repository evidence from the assigned path scopes before final exit.\n\n## Remaining Questions\n- Confirm concrete ownership, runtime responsibilities, and operational escalation evidence for this shard.\n"
	if !CollectDocumentBootstrapOnly(text) {
		t.Fatalf("expected marker-free recovery bootstrap to be classified as bootstrap-only")
	}
}

func bootstrapCollectManifest() contracts.ShardPackManifest {
	return contracts.ShardPackManifest{
		Version:      1,
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		ShardID:      "payments-src",
		AgentRole:    "shard-analyst",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/payments-src",
		Documents: []contracts.AuthoredDocument{{
			ID:            "doc.payments.src.overview",
			Kind:          "report",
			Title:         "Payments Overview",
			Path:          "payments-overview.md",
			CanonicalPath: "reports/as-is/payments-src/payments-overview.md",
			Topics:        []string{"payments"},
			CitationIDs:   []string{"cite.payments.src.overview"},
		}},
		Citations: []contracts.DocumentCitation{{
			ID:          "cite.payments.src.overview",
			Repo:        "payments",
			Path:        "src",
			ClaimIDs:    []string{"claim.payments.src.overview"},
			DocumentIDs: []string{"doc.payments.src.overview"},
		}},
		Semantic: contracts.SemanticSnapshot{
			Coverage: contracts.Coverage{
				Observed: []string{"payments"},
				Missing:  []string{"owner mapping evidence not confirmed from scoped repository files"},
				Notes:    []string{"Collect manifest covers the assigned shard scope with evidence paths listed in citations."},
			},
			Questions: []contracts.Question{{
				ID:         "question.payments.src.owner.mapping",
				Text:       "Which team owns the payments surface and its operational escalation path?",
				Priority:   "medium",
				RelatedIDs: []string{"svc.payments.src"},
			}},
			Entities: []contracts.Entity{
				{
					ID:   "svc.payments",
					Type: "service",
					Name: "Payments",
					Provenance: contracts.Provenance{
						Kind:       "observation",
						Confidence: 0.55,
						Evidence:   []contracts.Evidence{{Repo: "payments", Path: "src"}},
					},
				},
				{
					ID:   "svc.payments.src",
					Type: "service",
					Name: "Payments Src",
					Provenance: contracts.Provenance{
						Kind:       "observation",
						Confidence: 0.55,
						Evidence:   []contracts.Evidence{{Repo: "payments", Path: "src"}},
					},
				},
			},
			Edges: []contracts.Edge{{
				ID:   "edge.payments.src.contained-by-repo",
				Type: "contains",
				From: "svc.payments",
				To:   "svc.payments.src",
				Name: "contains scoped surface",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.45,
					Evidence:   []contracts.Evidence{{Repo: "payments", Path: "src"}},
				},
			}},
			Findings: []contracts.Finding{{
				ID:          "finding.payments.src.owner.mapping",
				Severity:    "medium",
				Title:       "Owner mapping not confirmed",
				Description: "Scoped evidence identifies the payments surface but does not confirm an owning team or escalation path.",
				RuleID:      "rule.owner.mapping.required",
				RelatedIDs:  []string{"svc.payments.src"},
				Provenance: contracts.Provenance{
					Kind:       "inference",
					Confidence: 0.45,
					Evidence:   []contracts.Evidence{{Repo: "payments", Path: "src"}},
				},
			}},
		},
	}
}
