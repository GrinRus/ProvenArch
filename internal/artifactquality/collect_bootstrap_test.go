package artifactquality

import (
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func TestCollectManifestBootstrapOnlyDetectsUnchangedFirstActionPair(t *testing.T) {
	t.Parallel()

	manifest := bootstrapCollectManifest()
	docs := map[string]string{
		"payments-overview.md": "# Payments Overview\n\n## Scope\n- Repository scope: payments.\n- Assigned scope summary: `src`.\n\n## Evidence Summary\n- Primary scoped evidence path: `src`.\n- This initial collect pair is a seed-only scoped evidence surface for the assigned shard.\n\n## Evidence Surface\n- `src`: scoped repository evidence available to this collect shard.\n\n## Initial Findings\n- The assigned evidence surface is traceable, but ownership, runtime responsibility, and escalation details need confirmation from richer repository evidence.\n\n## Coverage Gaps\n- Confirm concrete owners, runtime responsibilities, dependencies, and operational escalation paths for this shard.\n",
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

func TestCollectDocumentBootstrapOnlyRejectsRecoveryEvidenceSummary(t *testing.T) {
	t.Parallel()

	text := "# Payments Overview\n\n## Recovery Evidence Summary\n- Repository scope: payments.\n- Assigned scope summary: `src`, `README.md`.\n- Primary scoped evidence path: `README.md`.\n- This document is a seed-only collect recovery fallback for a shard whose first collect attempt did not complete with enriched artifacts.\n\n## Evidence Surface\n- `README.md`: scoped repository evidence available to the collect shard.\n- `src`: scoped repository evidence available to the collect shard.\n\n## Recovery Notes\n- The recovery pair records concrete scoped paths so downstream compilation can preserve traceability instead of accepting an empty or marker-only shard.\n\n## Remaining Questions\n- Confirm concrete ownership, runtime responsibilities, and operational escalation evidence for this shard.\n"
	if !CollectDocumentBootstrapOnly(text) {
		t.Fatalf("expected recovery evidence summary to be classified as low-signal bootstrap-only")
	}
}

func TestCollectDocumentBootstrapOnlyDetectsInterruptedTemporaryArtifact(t *testing.T) {
	t.Parallel()

	text := "# Bin Overview\n\nThis shard covers the PostHog repository `bin` path scope. The first bounded evidence read was attempted against the configured PostHog entrypoint hints and `bin` scope, but shell glob handling interrupted the listing before file contents were emitted. This initial artifact records only the assigned scoped surface and will be repaired with concrete file-level evidence after the artifact pair exists.\n\n## Observed Surface\n\n- Repository scope: `posthog`.\n- Assigned path scope: `bin`.\n- Required authored artifact: `bin-overview.md`.\n\n## Evidence Gaps\n\n- Concrete `bin` file roles still require file-level confirmation.\n- Owner mapping and escalation evidence are not confirmed.\n- Operational dependency evidence is not confirmed.\n"
	if !CollectDocumentBootstrapOnly(text) {
		t.Fatalf("expected interrupted temporary collect artifact to be classified as bootstrap-only")
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
