package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDocflowExamples(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		path  string
		parse func([]byte) error
	}{
		{
			name: "shard-pack-manifest",
			path: filepath.Join("..", "..", "examples", "shard-pack-manifest.example.json"),
			parse: func(raw []byte) error {
				_, err := ParseShardPackManifest(raw)
				return err
			},
		},
		{
			name: "citation-index",
			path: filepath.Join("..", "..", "examples", "citation-index.example.json"),
			parse: func(raw []byte) error {
				_, err := ParseCitationIndex(raw)
				return err
			},
		},
		{
			name: "final-run-index",
			path: filepath.Join("..", "..", "examples", "final-run-index.example.json"),
			parse: func(raw []byte) error {
				_, err := ParseFinalRunIndex(raw)
				return err
			},
		},
		{
			name: "validator-verdict",
			path: filepath.Join("..", "..", "examples", "validator-verdict.example.json"),
			parse: func(raw []byte) error {
				_, err := ParseValidatorVerdict(raw)
				return err
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile(filepath.Clean(tc.path))
			if err != nil {
				t.Fatalf("read example %q: %v", tc.path, err)
			}
			if err := tc.parse(raw); err != nil {
				t.Fatalf("parse example %q: %v", tc.path, err)
			}
		})
	}
}

func TestParseShardPackManifestRejectsUnknownCitationReference(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "payments-service",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/run-1/shard",
  "documents": [
    {
      "id": "doc.domain.payments",
      "kind": "agent-output",
      "title": "Payments",
      "path": "domain-report.md",
      "canonical_path": "reports/agent-outputs/domains/payments.md",
      "topics": ["domain.payments"],
      "citation_ids": ["cite.missing"],
      "status": "staged"
    }
  ],
  "citations": [],
  "semantic": {
    "coverage": {"observed": [], "missing": [], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`)
	_, err := ParseShardPackManifest(raw)
	if err == nil {
		t.Fatalf("expected shard manifest validation error")
	}
	if !strings.Contains(err.Error(), "unknown citation_id") {
		t.Fatalf("expected unknown citation_id error, got %v", err)
	}
}

func TestParseFinalRunIndexRejectsBrokenTopicReference(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "pipeline": "init",
  "generated_at": "2026-04-16T12:00:00Z",
  "citation_index_path": "reports/taskruns/run-1/staging/final/citation-index.json",
  "canonical_documents": [],
  "topics": [
    {"id": "domain.payments", "document_ids": ["doc.missing"]}
  ],
  "semantic": {
    "coverage": {"observed": [], "missing": [], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`)
	_, err := ParseFinalRunIndex(raw)
	if err == nil {
		t.Fatalf("expected final run index validation error")
	}
	if !strings.Contains(err.Error(), "unknown document_id") {
		t.Fatalf("expected unknown document_id error, got %v", err)
	}
}

func TestParseShardPackManifestRejectsForbiddenCanonicalPath(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "payments-service",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/run-1/shard",
  "documents": [
    {
      "id": "doc.forbidden",
      "kind": "agent-output",
      "title": "Forbidden Doc",
      "path": "forbidden.md",
      "canonical_path": "charter/overview.md",
      "topics": ["domain.payments"],
      "citation_ids": ["cite.1"]
    }
  ],
  "citations": [
    {
      "id": "cite.1",
      "repo": "payments-service",
      "path": "README.md",
      "claim_ids": ["claim.1"],
      "document_ids": ["doc.forbidden"]
    }
  ],
  "semantic": {
    "coverage": {"observed": [], "missing": [], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`)
	_, err := ParseShardPackManifest(raw)
	if err == nil {
		t.Fatalf("expected shard manifest canonical_path validation error")
	}
	if !strings.Contains(err.Error(), "canonical_path must be within reports/as-is") {
		t.Fatalf("expected canonical path surface error, got %v", err)
	}
}

func TestParseShardPackManifestRejectsEscapingDocumentPath(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "payments-service",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/run-1/shard",
  "documents": [
    {
      "id": "doc.escape",
      "kind": "agent-output",
      "title": "Escaping Doc",
      "path": "../escape.md",
      "canonical_path": "reports/agent-outputs/domains/payments.md",
      "topics": ["domain.payments"],
      "citation_ids": ["cite.1"]
    }
  ],
  "citations": [
    {
      "id": "cite.1",
      "repo": "payments-service",
      "path": "README.md",
      "claim_ids": ["claim.1"],
      "document_ids": ["doc.escape"]
    }
  ],
  "semantic": {
    "coverage": {"observed": [], "missing": [], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`)
	_, err := ParseShardPackManifest(raw)
	if err == nil {
		t.Fatalf("expected shard manifest document path validation error")
	}
	if !strings.Contains(err.Error(), "must not escape artifact_root") {
		t.Fatalf("expected path traversal error, got %v", err)
	}
}

func TestParseShardPackManifestRejectsWorkspaceRelativeDocumentPath(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-iac",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac",
  "documents": [
    {
      "id": "doc.iac",
      "kind": "analysis",
      "title": "IAC Overview",
      "path": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac/iac-overview.md",
      "canonical_path": "reports/as-is/bank-of-anthos/iac-overview.md",
      "topics": ["iac"],
      "citation_ids": ["cite.1"]
    }
  ],
  "citations": [
    {
      "id": "cite.1",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.1"],
      "document_ids": ["doc.iac"]
    }
  ],
  "semantic": {
    "coverage": {"observed": [], "missing": [], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`)

	_, err := ParseShardPackManifest(raw)
	if err == nil {
		t.Fatalf("expected workspace-relative document path validation error")
	}
	if !strings.Contains(err.Error(), "artifact_root-relative") {
		t.Fatalf("expected artifact_root-relative error, got %v", err)
	}
}

func TestParseShardPackManifestRejectsDuplicatedArtifactRootDocumentPath(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "payments-service",
  "agent_role": "shard-analyst",
  "artifact_root": "staging/shards/payments-service",
  "documents": [
    {
      "id": "doc.payments",
      "kind": "analysis",
      "title": "Payments Overview",
      "path": "staging/shards/payments-service/domain-report.md",
      "canonical_path": "reports/as-is/payments-service/domain-report.md",
      "topics": ["payments"],
      "citation_ids": ["cite.1"]
    }
  ],
  "citations": [
    {
      "id": "cite.1",
      "repo": "payments-service",
      "path": "README.md",
      "claim_ids": ["claim.1"],
      "document_ids": ["doc.payments"]
    }
  ],
  "semantic": {
    "coverage": {"observed": [], "missing": [], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`)

	_, err := ParseShardPackManifest(raw)
	if err == nil {
		t.Fatalf("expected duplicated artifact_root document path validation error")
	}
	if !strings.Contains(err.Error(), "must not repeat artifact_root") {
		t.Fatalf("expected repeated artifact_root error, got %v", err)
	}
}

func TestParseValidatorVerdictRejectsUnsupportedVerdict(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "generated_at": "2026-04-16T12:00:00Z",
  "verdict": "WARN",
  "checked_paths": ["reports/taskruns/run-1/staging/final/final-run-index.json"]
}`)
	_, err := ParseValidatorVerdict(raw)
	if err == nil {
		t.Fatalf("expected validator verdict validation error")
	}
	if !strings.Contains(err.Error(), `"PASS", "FAIL"`) {
		t.Fatalf("expected PASS or FAIL error, got %v", err)
	}
}
