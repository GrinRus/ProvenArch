package artifactquality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func TestValidateCollectManifestRejectsContractInvalidCompatibilityPayload(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeFixtureManifest(t, writeRoot, "bank_extras_invalid_manifest.json")
	writeDoc(t, writeRoot, "extras-overview.md", "# Extras\n")

	if err := ValidateCollectManifestInRoot(writeRoot); err == nil {
		t.Fatalf("expected validation to fail for contract-invalid compatibility payload")
	} else if !strings.Contains(err.Error(), "shard pack manifest is invalid") {
		t.Fatalf("expected schema/contract rejection error, got %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read original manifest: %v", err)
	}
	if _, err := contracts.ParseShardPackManifest(raw); err == nil {
		t.Fatalf("expected original manifest to remain invalid")
	}
}

func TestValidateCollectManifestRejectsProviderToolDocumentPath(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	payload := validCollectManifestPayload()
	payload["documents"].([]any)[0].(map[string]any)["path"] = ".qwen/skills/acp-collect-shard-execution/SKILL.md"
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	writeDoc(t, writeRoot, ".qwen/skills/acp-collect-shard-execution/SKILL.md", "# Tool side effect\n")

	err = ValidateCollectManifestInRoot(writeRoot)
	if err == nil {
		t.Fatalf("expected provider tool document path to fail validation")
	}
	if !strings.Contains(err.Error(), ".qwen") {
		t.Fatalf("expected validation error to mention provider tool component, got %v", err)
	}
}

func TestValidateCollectManifestRejectsBootstrapOnlyDocument(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	manifest := bootstrapCollectManifest()
	writeManifest(t, writeRoot, manifest)
	writeDoc(t, writeRoot, "payments-overview.md", "# Payments Overview\n\n<!-- "+CollectBootstrapReplaceMarker+" -->\n\n## Observations\n- `src/payment_handler.go` defines the payment service entrypoint.\n")

	err := ValidateCollectManifestInRoot(writeRoot)
	if err == nil {
		t.Fatalf("expected bootstrap-only collect document to fail validation")
	}
	if !strings.Contains(err.Error(), "bootstrap-only collect document") {
		t.Fatalf("expected bootstrap-only document validation error, got %v", err)
	}
}

func TestValidateCollectManifestRejectsLowSignalRecoveryDocument(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	manifest := bootstrapCollectManifest()
	writeManifest(t, writeRoot, manifest)
	writeDoc(t, writeRoot, "payments-overview.md", "# Payments Overview\n\n## Recovery Summary\n- Repository: payments\n- Assigned scope: src\n- Evidence candidate used for the recovery manifest: `src`.\n\n## Evidence Candidates\n- `src` is the first scoped repository path encoded in the recovery manifest.\n- Additional repository-specific details should be enriched by the provider when available within the repair window.\n\n## Remaining Questions\n- Confirm concrete ownership, runtime responsibilities, and operational escalation evidence for this shard.\n")

	err := ValidateCollectManifestInRoot(writeRoot)
	if err == nil {
		t.Fatalf("expected low-signal recovery collect document to fail validation")
	}
	if !strings.Contains(err.Error(), "bootstrap-only collect document") {
		t.Fatalf("expected low-signal recovery document validation error, got %v", err)
	}
}

func TestValidateCollectManifestRejectsMarkerFreeInitialSeedDocument(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	manifest := bootstrapCollectManifest()
	writeManifest(t, writeRoot, manifest)
	writeDoc(t, writeRoot, "payments-overview.md", "# Payments Overview\n\n## Scope\n- Repository scope: payments.\n- Assigned scope summary: `src`.\n\n## Evidence Summary\n- Primary scoped evidence path: `src`.\n- This initial collect pair is a seed-only scoped evidence surface for the assigned shard.\n\n## Evidence Surface\n- `src`: scoped repository evidence available to this collect shard.\n\n## Initial Findings\n- The assigned evidence surface is traceable, but ownership, runtime responsibility, and escalation details need confirmation from richer repository evidence.\n\n## Coverage Gaps\n- Confirm concrete owners, runtime responsibilities, dependencies, and operational escalation paths for this shard.\n")

	err := ValidateCollectManifestInRoot(writeRoot)
	if err == nil {
		t.Fatalf("expected marker-free initial seed collect document to fail validation")
	}
	if !strings.Contains(err.Error(), "bootstrap-only collect document") {
		t.Fatalf("expected seed document validation error, got %v", err)
	}
}

func TestValidateCollectManifestRejectsRecoveryEvidenceFallbackDocument(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	manifest := bootstrapCollectManifest()
	writeManifest(t, writeRoot, manifest)
	writeDoc(t, writeRoot, "payments-overview.md", "# Payments Overview\n\n## Recovery Evidence Summary\n- Repository scope: payments.\n- Assigned scope summary: `src`.\n- Primary scoped evidence path: `src`.\n- This document is a seed-only collect recovery fallback for a shard whose first collect attempt did not complete with enriched artifacts.\n\n## Evidence Surface\n- `src`: scoped repository evidence available to the collect shard.\n\n## Recovery Notes\n- The recovery pair records concrete scoped paths so downstream compilation can preserve traceability instead of accepting an empty or marker-only shard.\n\n## Remaining Questions\n- Confirm concrete ownership, runtime responsibilities, and operational escalation evidence for this shard.\n")

	err := ValidateCollectManifestInRoot(writeRoot)
	if err == nil {
		t.Fatalf("expected recovery evidence fallback collect document to fail validation")
	}
	if !strings.Contains(err.Error(), "bootstrap-only collect document") {
		t.Fatalf("expected recovery fallback validation error, got %v", err)
	}
}

func TestValidateCollectManifestRejectsForbiddenSemanticAliasesBySchema(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		mutate    func(map[string]any)
		wantToken string
	}{
		{
			name: "coverage covered_topics",
			mutate: func(payload map[string]any) {
				semanticMap(payload)["coverage"].(map[string]any)["covered_topics"] = []any{"services"}
			},
			wantToken: "covered_topics",
		},
		{
			name: "question text alias",
			mutate: func(payload map[string]any) {
				semanticSliceItem(payload, "questions", 0)["question"] = "Who owns payments?"
			},
			wantToken: "question",
		},
		{
			name: "question confidence alias",
			mutate: func(payload map[string]any) {
				semanticSliceItem(payload, "questions", 0)["confidence"] = "0.7"
			},
			wantToken: "confidence",
		},
		{
			name: "edge relation alias",
			mutate: func(payload map[string]any) {
				semanticSliceItem(payload, "edges", 0)["relation"] = "depends_on"
			},
			wantToken: "relation",
		},
		{
			name: "edge source alias",
			mutate: func(payload map[string]any) {
				semanticSliceItem(payload, "edges", 0)["source"] = "svc.payments"
			},
			wantToken: "source",
		},
		{
			name: "edge target alias",
			mutate: func(payload map[string]any) {
				semanticSliceItem(payload, "edges", 0)["target"] = "svc.ledger"
			},
			wantToken: "target",
		},
		{
			name: "entity provenance array",
			mutate: func(payload map[string]any) {
				semanticSliceItem(payload, "entities", 0)["provenance"] = []any{
					map[string]any{"citation_id": "cite.payments.readme"},
				}
			},
			wantToken: "provenance",
		},
		{
			name: "edge string confidence",
			mutate: func(payload map[string]any) {
				provenance := semanticSliceItem(payload, "edges", 0)["provenance"].(map[string]any)
				provenance["confidence"] = "0.7"
			},
			wantToken: "confidence",
		},
		{
			name: "finding summary alias",
			mutate: func(payload map[string]any) {
				semanticSliceItem(payload, "findings", 0)["summary"] = "legacy summary"
			},
			wantToken: "summary",
		},
		{
			name: "finding inference alias",
			mutate: func(payload map[string]any) {
				semanticSliceItem(payload, "findings", 0)["inference"] = "legacy inference"
			},
			wantToken: "inference",
		},
		{
			name: "finding citation alias",
			mutate: func(payload map[string]any) {
				semanticSliceItem(payload, "findings", 0)["evidence_citation_ids"] = []any{"cite.payments.readme"}
			},
			wantToken: "evidence_citation_ids",
		},
		{
			name: "finding confidence alias",
			mutate: func(payload map[string]any) {
				semanticSliceItem(payload, "findings", 0)["confidence"] = "0.7"
			},
			wantToken: "confidence",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload := validCollectManifestPayload()
			tc.mutate(payload)
			raw, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			err = ValidateCollectManifestBytes(raw)
			if err == nil {
				t.Fatalf("expected schema validation to fail for %s", tc.name)
			}
			if !strings.Contains(err.Error(), "shard pack manifest is invalid") {
				t.Fatalf("expected shard manifest schema error, got %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantToken) {
				t.Fatalf("expected validation error to mention %q, got %v", tc.wantToken, err)
			}
		})
	}
}

func TestValidateCollectManifestRejectsCitationOnlySemanticEvidenceObjects(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeDoc(t, writeRoot, "overview.md", "# Overview\n")

	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "shard_id": "payments",
  "domain_id": "payments",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/payments",
  "repo_scopes": ["payments-service"],
  "path_scopes": ["src"],
  "documents": [
    {
      "id": "doc.payments.overview",
      "kind": "report",
      "title": "Payments Overview",
      "path": "overview.md",
      "canonical_path": "reports/as-is/payments/overview.md",
      "topics": ["payments"],
      "citation_ids": ["cite.payments.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.payments.readme",
      "repo": "payments-service",
      "path": "README.md",
      "claim_ids": ["claim.payments.readme"],
      "document_ids": ["doc.payments.overview"]
    }
  ],
  "semantic": {
    "coverage": {
      "observed": ["services"],
      "missing": ["owner mappings"],
      "notes": ["citation-only drift"]
    },
    "questions": [],
    "entities": [
      {
        "id": "svc.payments",
        "name": "payments",
        "type": "service",
        "provenance": {
          "kind": "observation",
          "confidence": 0.8,
          "evidence": [
            {
              "citation_id": "cite.payments.readme"
            }
          ]
        }
      }
    ],
    "edges": [],
    "findings": []
  }
}`)
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := ValidateCollectManifestInRoot(writeRoot)
	if err == nil {
		t.Fatalf("expected validation to fail for citation-only semantic evidence")
	}
	if !strings.Contains(err.Error(), "/semantic/entities/0/provenance/evidence/0") {
		t.Fatalf("expected explicit semantic evidence path in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "'repo', 'path'") {
		t.Fatalf("expected repo/path requirement in error, got %v", err)
	}
}

func TestValidateCollectManifestRejectsArtifactRootPrefixedDocumentPathWithoutRewrite(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeFixtureManifest(t, writeRoot, "bank_of_anthos_doubled_path_manifest.json")
	writeDoc(t, writeRoot, "iac-overview.md", "# IAC Overview\n")

	manifestPath := filepath.Join(writeRoot, shardPackManifestFile)
	before, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read original manifest: %v", err)
	}

	if err := ValidateCollectManifestInRoot(writeRoot); err == nil {
		t.Fatalf("expected artifact-root-prefixed document path to fail strict validation")
	} else if !strings.Contains(err.Error(), "must be artifact_root-relative") {
		t.Fatalf("expected strict document path error, got %v", err)
	}

	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest after validation: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("expected strict validation not to rewrite manifest")
	}
}

func TestValidateCollectManifestRejectsAbsoluteDocumentPathWithoutRewrite(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeDoc(t, writeRoot, "service-inventory.md", "# Service Inventory\n")

	manifest := contracts.ShardPackManifest{
		Version:      1,
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		ShardID:      "openedx-platform",
		AgentRole:    "shard-analyst",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/openedx-platform",
		RepoScopes:   []string{"openedx-platform"},
		PathScopes:   []string{"."},
		Documents: []contracts.AuthoredDocument{
			{
				ID:            "doc.service-inventory",
				Kind:          "report",
				Title:         "Service Inventory",
				Path:          filepath.Join(writeRoot, "service-inventory.md"),
				CanonicalPath: "reports/as-is/service-inventory/openedx-platform.md",
				Topics:        []string{"openedx"},
				CitationIDs:   []string{"cite.openedx.readme"},
			},
		},
		Citations: []contracts.DocumentCitation{
			{
				ID:          "cite.openedx.readme",
				Repo:        "openedx-platform",
				Path:        "README.md",
				ClaimIDs:    []string{"claim.openedx.readme"},
				DocumentIDs: []string{"doc.service-inventory"},
			},
		},
		Semantic: contracts.SemanticSnapshot{
			Coverage: contracts.Coverage{
				Observed: []string{"services"},
				Missing:  []string{"owner mappings"},
				Notes:    []string{"absolute path drift"},
			},
			Questions: []contracts.Question{},
			Entities:  []contracts.Entity{},
			Edges:     []contracts.Edge{},
			Findings:  []contracts.Finding{},
		},
	}
	writeManifest(t, writeRoot, manifest)

	manifestPath := filepath.Join(writeRoot, shardPackManifestFile)
	before, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read original manifest: %v", err)
	}

	if err := ValidateCollectManifestInRoot(writeRoot); err == nil {
		t.Fatalf("expected absolute document path to fail strict validation")
	} else if !strings.Contains(err.Error(), "must be relative") {
		t.Fatalf("expected strict document path error, got %v", err)
	}

	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest after validation: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("expected strict validation not to rewrite manifest")
	}
}

func TestValidateCollectManifestRejectsMissingReferencedDocumentPath(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	payload := validCollectManifestPayload()
	documents := payload["documents"].([]any)
	documents[0].(map[string]any)["path"] = "src-ledger-balereader-overview.md"
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	writeDoc(t, writeRoot, "src-ledger-balancereader-overview.md", "# Balance Reader\n")

	if err := ValidateCollectManifestInRoot(writeRoot); err == nil {
		t.Fatalf("expected missing referenced document path to fail strict validation")
	} else if !strings.Contains(err.Error(), `documents[0].path references missing document file "src-ledger-balereader-overview.md"`) {
		t.Fatalf("expected missing document path error, got %v", err)
	}
}

func TestValidateCollectManifestRejectsDirectoryDocumentPath(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	payload := validCollectManifestPayload()
	if err := os.Mkdir(filepath.Join(writeRoot, "overview.md"), 0o755); err != nil {
		t.Fatalf("mkdir document directory: %v", err)
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if err := ValidateCollectManifestInRoot(writeRoot); err == nil {
		t.Fatalf("expected directory referenced as document path to fail strict validation")
	} else if !strings.Contains(err.Error(), `documents[0].path references a directory`) {
		t.Fatalf("expected directory document path error, got %v", err)
	}
}

func TestValidateCollectManifestRejectsMissingRequiredMetadataWithoutAutofill(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "shard_id": "payments",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/payments",
  "documents": [],
  "citations": [],
  "semantic": {
    "coverage": {"observed": [], "missing": [], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`)
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if err := ValidateCollectManifestInRoot(writeRoot); err != nil {
		t.Fatalf("expected valid manifest without optional repo/path/domain metadata to pass: %v", err)
	}

	missingRunID := strings.Replace(string(raw), `"run_id": "run-1",`, `"run_id": "",`, 1)
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), []byte(missingRunID), 0o644); err != nil {
		t.Fatalf("write missing run_id manifest: %v", err)
	}
	if err := ValidateCollectManifestInRoot(writeRoot); err == nil {
		t.Fatalf("expected missing run_id to fail strict validation")
	} else if !strings.Contains(err.Error(), "run_id") {
		t.Fatalf("expected run_id validation error, got %v", err)
	}
}

func TestValidateCollectManifestRejectsUnknownTopLevelFieldWithoutRewrite(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "shard_id": "payments",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/payments",
  "unknown_runtime_wrapper": true,
  "documents": [],
  "citations": [],
  "semantic": {
    "coverage": {"observed": [], "missing": [], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`)
	manifestPath := filepath.Join(writeRoot, shardPackManifestFile)
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if err := ValidateCollectManifestInRoot(writeRoot); err == nil {
		t.Fatalf("expected unknown top-level field to fail strict validation")
	} else if !strings.Contains(err.Error(), "unknown_runtime_wrapper") {
		t.Fatalf("expected unknown field name in validation error, got %v", err)
	}

	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest after validation: %v", err)
	}
	if string(after) != string(raw) {
		t.Fatalf("expected strict validation not to rewrite manifest")
	}
}

func validCollectManifestPayload() map[string]any {
	evidence := []any{
		map[string]any{
			"repo": "payments-service",
			"path": "README.md",
		},
	}
	provenance := func(confidence float64) map[string]any {
		return map[string]any{
			"kind":       "observation",
			"confidence": confidence,
			"evidence":   evidence,
		}
	}
	return map[string]any{
		"version":       float64(1),
		"run_id":        "run-1",
		"step_id":       "refresh.step1.collect",
		"shard_id":      "payments",
		"domain_id":     "payments",
		"agent_role":    "shard-analyst",
		"artifact_root": "reports/taskruns/run-1/staging/shards/payments",
		"repo_scopes":   []any{"payments-service"},
		"path_scopes":   []any{"src"},
		"documents": []any{
			map[string]any{
				"id":             "doc.payments.overview",
				"kind":           "report",
				"title":          "Payments Overview",
				"path":           "overview.md",
				"canonical_path": "reports/as-is/payments/overview.md",
				"topics":         []any{"payments"},
				"citation_ids":   []any{"cite.payments.readme"},
			},
		},
		"citations": []any{
			map[string]any{
				"id":           "cite.payments.readme",
				"repo":         "payments-service",
				"path":         "README.md",
				"claim_ids":    []any{"claim.payments.readme"},
				"document_ids": []any{"doc.payments.overview"},
			},
		},
		"semantic": map[string]any{
			"coverage": map[string]any{
				"observed": []any{"services"},
				"missing":  []any{"owner mappings"},
				"notes":    []any{"canonical payload"},
			},
			"questions": []any{
				map[string]any{
					"id":       "q.payments.owner",
					"text":     "Who owns payments?",
					"priority": "high",
				},
			},
			"entities": []any{
				map[string]any{
					"id":         "svc.payments",
					"name":       "payments",
					"type":       "service",
					"provenance": provenance(0.8),
				},
			},
			"edges": []any{
				map[string]any{
					"id":         "edge.payments.depends-on",
					"type":       "depends_on",
					"from":       "svc.payments",
					"to":         "svc.ledger",
					"provenance": provenance(0.7),
				},
			},
			"findings": []any{
				map[string]any{
					"id":          "finding.payments.owner",
					"severity":    "medium",
					"title":       "Missing owner",
					"description": "Owner mapping is absent.",
					"provenance":  provenance(0.6),
				},
			},
		},
	}
}

func semanticMap(payload map[string]any) map[string]any {
	return payload["semantic"].(map[string]any)
}

func semanticSliceItem(payload map[string]any, key string, index int) map[string]any {
	return semanticMap(payload)[key].([]any)[index].(map[string]any)
}

func writeFixtureManifest(t *testing.T, writeRoot string, fixtureName string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", fixtureName))
	if err != nil {
		t.Fatalf("read fixture %q: %v", fixtureName, err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}
}

func writeManifest(t *testing.T, writeRoot string, manifest contracts.ShardPackManifest) {
	t.Helper()
	raw, err := jsonMarshalManifest(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func jsonMarshalManifest(manifest contracts.ShardPackManifest) ([]byte, error) {
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func writeDoc(t *testing.T, writeRoot string, rel string, content string) {
	t.Helper()
	abs := filepath.Join(writeRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir for doc %q: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write doc %q: %v", rel, err)
	}
}
