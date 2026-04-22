package artifactquality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/compatibilityregistry"
)

func TestRepairCollectManifestRejectsLegacyCompatibilityPayload(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeFixtureManifest(t, writeRoot, "bank_extras_invalid_manifest.json")
	writeDoc(t, writeRoot, "extras-overview.md", "# Extras\n")

	task := testCollectTask(writeRoot, "bank-of-anthos-extras", "bank-of-anthos")
	if _, err := RepairCollectManifest(task); err == nil {
		t.Fatalf("expected repair to fail for legacy compatibility payload")
	} else if !strings.Contains(err.Error(), "legacy collect manifest fields are forbidden") {
		t.Fatalf("expected explicit legacy rejection error, got %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read original manifest: %v", err)
	}
	if _, err := contracts.ParseShardPackManifest(raw); err == nil {
		t.Fatalf("expected original manifest to remain invalid")
	}
}

func TestRepairCollectManifestRejectsLegacySemanticAliasesBeforeDecode(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeDoc(t, writeRoot, "overview.md", "# Overview\n")

	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "step_contract": "collect",
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
      "covered_topics": ["services"],
      "missing": ["owner mappings"],
      "notes": ["legacy payload"]
    },
    "questions": [
      {
        "id": "q.payments.owner",
        "question": "Who owns payments?"
      }
    ],
    "entities": [
      {
        "id": "svc.payments",
        "name": "payments",
        "type": "service",
        "provenance": [
          {
            "citation_id": "cite.payments.readme"
          }
        ]
      }
    ],
    "edges": [
      {
        "id": "edge.payments.depends-on",
        "relation": "depends_on",
        "from": "svc.payments",
        "to": "svc.ledger",
        "provenance": {
          "kind": "observation",
          "confidence": "0.7"
        }
      }
    ],
    "findings": [
      {
        "id": "finding.payments.owner",
        "severity": "medium",
        "title": "Missing owner",
        "description": "Owner mapping is absent.",
        "summary": "legacy summary",
        "evidence_citation_ids": ["cite.payments.readme"]
      }
    ]
  }
}`)
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	task := testCollectTask(writeRoot, "payments", "payments-service")
	_, err := RepairCollectManifest(task)
	if err == nil {
		t.Fatalf("expected repair to fail for legacy semantic aliases")
	}

	message := err.Error()
	for _, token := range []string{
		"step_contract",
		"semantic.coverage.covered_topics",
		"semantic.questions[0].question",
		"semantic.entities[0].provenance",
		"semantic.edges[0].relation",
		"semantic.edges[0].provenance.confidence",
		"semantic.findings[0].summary",
		"semantic.findings[0].evidence_citation_ids",
	} {
		if !strings.Contains(message, token) {
			t.Fatalf("expected legacy rejection message to mention %q, got %v", token, err)
		}
	}
}

func TestRepairCollectManifestRejectsCitationOnlySemanticEvidenceObjects(t *testing.T) {
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

	task := testCollectTask(writeRoot, "payments", "payments-service")
	_, err := RepairCollectManifest(task)
	if err == nil {
		t.Fatalf("expected repair to fail for citation-only semantic evidence")
	}
	if !strings.Contains(err.Error(), "semantic.entities[0].provenance.evidence[0]") {
		t.Fatalf("expected explicit semantic evidence path in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "repo/path") {
		t.Fatalf("expected repo/path requirement in error, got %v", err)
	}
}

func TestRepairCollectManifestNormalizesArtifactRootPrefixedDocumentPath(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeFixtureManifest(t, writeRoot, "bank_of_anthos_doubled_path_manifest.json")
	writeDoc(t, writeRoot, "iac-overview.md", "# IAC Overview\n")

	task := testCollectTask(writeRoot, "bank-of-anthos-iac", "bank-of-anthos")
	task.RunID = "run_20260420_054749_001"
	task.StepID = "init.step1.collect"
	task.ArtifactRoot = "reports/taskruns/run_20260420_054749_001/staging/shards/bank-of-anthos-iac"
	task.PathScopes = []string{"iac"}

	report, err := RepairCollectManifest(task)
	if err != nil {
		t.Fatalf("repair manifest: %v", err)
	}
	if !report.Changed {
		t.Fatalf("expected repair report to mark manifest as changed")
	}
	if len(report.AppliedRuleIDs) != 1 || report.AppliedRuleIDs[0] != compatibilityregistry.RuleSafeCollectDocumentPathNormalization {
		t.Fatalf("expected collect path normalization rule, got %#v", report.AppliedRuleIDs)
	}

	raw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read repaired manifest: %v", err)
	}
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		t.Fatalf("parse repaired manifest: %v", err)
	}
	if got := manifest.Documents[0].Path; got != "iac-overview.md" {
		t.Fatalf("expected normalized document path, got %q", got)
	}

	assessment, err := LoadManifestAssessment(writeRoot)
	if err != nil {
		t.Fatalf("load manifest assessment: %v", err)
	}
	if !assessment.Rich {
		t.Fatalf("expected repaired manifest to remain rich, got %#v", assessment)
	}
}

func TestRepairCollectManifestNormalizesAbsoluteDocumentPathUnderWriteRoot(t *testing.T) {
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

	task := testCollectTask(writeRoot, "openedx-platform", "openedx-platform")
	report, err := RepairCollectManifest(task)
	if err != nil {
		t.Fatalf("repair manifest: %v", err)
	}
	if !report.Changed {
		t.Fatalf("expected repair report to mark manifest as changed")
	}
	if len(report.AppliedRuleIDs) != 1 || report.AppliedRuleIDs[0] != compatibilityregistry.RuleSafeCollectDocumentPathNormalization {
		t.Fatalf("expected collect path normalization rule, got %#v", report.AppliedRuleIDs)
	}

	raw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read repaired manifest: %v", err)
	}
	repaired, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		t.Fatalf("parse repaired manifest: %v", err)
	}
	if got := repaired.Documents[0].Path; got != "service-inventory.md" {
		t.Fatalf("expected absolute path to normalize to write-root relative path, got %q", got)
	}
}

func TestCanonicalizeCollectManifestDoesNotReportRepairForAmbiguousDocumentPath(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()

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
				Path:          "reports/taskruns/run-1/staging/shards/openedx-platform/service-inventory.md",
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
				Notes:    []string{"artifact root path remains invalid when file is absent"},
			},
		},
	}
	raw, err := jsonMarshalManifest(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	task := testCollectTask(writeRoot, "openedx-platform", "openedx-platform")
	_, report, err := canonicalizeCollectManifest(raw, task)
	if err == nil {
		t.Fatalf("expected invalid duplicated artifact_root path to remain invalid without safe repair")
	}
	if report.Changed {
		t.Fatalf("expected no repair report for ambiguous document path, got %#v", report)
	}
	if len(report.AppliedRuleIDs) != 0 {
		t.Fatalf("expected no applied rule ids for ambiguous document path, got %#v", report.AppliedRuleIDs)
	}
}

func testCollectTask(writeRoot string, shardID string, repo string) acpruntime.Task {
	return acpruntime.Task{
		TaskID:       "task-" + shardID,
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    filepath.Clean(filepath.Join(writeRoot, "..", "workspace")),
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/" + shardID,
		ShardID:      shardID,
		DomainID:     shardID,
		AgentRole:    "shard-analyst",
		RepoScope:    repo,
		RepoScopes:   []string{repo},
		PathScopes:   []string{"."},
		StartedAtUTC: time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC),
	}
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
