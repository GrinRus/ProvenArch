package artifactquality

import (
	"fmt"
	"strings"
)

func CollectManifestContractLines(artifactRoot string) []string {
	lines := []string{
		fmt.Sprintf(`- shard-pack-manifest.json MUST validate against the ACP shard-pack-manifest contract and use artifact_root = %q.`, artifactRoot),
		`- version MUST be integer 1 (not "1.0.0" or any other string form).`,
		`- Each documents[] item MUST include: id, kind, title, path, canonical_path, topics, citation_ids.`,
		`- documents[].path MUST be artifact_root-relative only; valid examples: "iac-overview.md", "docs/service-catalog.md". Invalid examples: "reports/taskruns/.../iac-overview.md", "charter/overview.md", "proposals/plan.md".`,
		`- Do NOT use documents[].citations; only documents[].citation_ids is allowed.`,
		`- Each citations[] item MUST include: id, repo, path, claim_ids, document_ids.`,
		`- citations[].repo MUST match one resolved repo scope, and citations[].path MUST point to an existing relative path under that repo root; never cite guessed files such as pom.xml in Gradle-only repos.`,
		`- citations[].id values MUST be unique, and every documents[].citation_ids entry MUST reference an existing citations[].id.`,
		`- citations[].claim_ids MUST be globally unique across the assembled staged final set; do NOT reuse the same claim id across different shard/citation surfaces.`,
		`- Build claim_ids as a semantic stem plus shard slug (example: claim.owner.<shard-slug>), and add a deterministic numeric suffix when a shard-local collision remains.`,
		`- citations[].document_ids MUST point back to ids that exist in documents[].id.`,
		`- semantic MUST include coverage, questions, entities, edges, and findings; questions/entities/edges/findings MUST be arrays even when empty.`,
		`- semantic.questions[*] MUST include id and text; do NOT omit stable question ids.`,
		`- semantic.entities[*] MUST remain full entity objects with provenance; do not drop provenance during repair or retry flows.`,
		`- semantic.findings[*] MUST remain structured finding objects; never collapse findings to strings.`,
		`- semantic.findings[*] MUST include id, severity, title, and provenance; severity must be one of the canonical schema values.`,
		`- Every semantic.entities[*].provenance.evidence[*], semantic.edges[*].provenance.evidence[*], and semantic.findings[*].provenance.evidence[*] item MUST include non-empty repo and path values that point to concrete repository evidence.`,
		`- Semantic provenance evidence paths MUST exist under the resolved repo root; if a planned claim is not supported by an existing file, remove the claim or record it as a coverage gap.`,
		`- Citation-only semantic evidence objects are forbidden; do NOT use {"citation_id":"..."} without repo/path inside semantic provenance.evidence[].`,
		`- semantic.findings[*] MUST use id + severity + title + description + provenance; do NOT use summary as a compatibility alias.`,
		`- canonical_path MUST be a stable promoted path under reports/as-is, reports/findings, reports/coverage, reports/agent-outputs, reports/diagrams, or proposals.`,
		`- Do NOT use reports/taskruns/... staging paths as canonical_path.`,
	}
	return append(lines, CollectManifestLegacyHygieneLines()...)
}

func ClaimIDContractLines() []string {
	return []string{
		`- claim_ids are a global staged-final-set namespace, not a per-shard local namespace.`,
		`- Do NOT reuse the same claim_id across different citations or shards, even when they describe related evidence.`,
		`- When deriving claim_ids, append the shard slug and then a deterministic numeric suffix if another collision remains.`,
	}
}

func CollectManifestLegacyHygieneLines() []string {
	return []string{
		`- Treat schemas/*, docs/spec/*, and the enforced collect prompt contract as the only schema source of truth; do NOT infer manifest shape from prior reports/taskruns artifacts, raw logs, or archived examples.`,
		`- semantic.coverage MUST use observed/missing/notes; do NOT use covered_topics or alternate coverage keys.`,
		`- semantic.questions[*] MUST use id + text; do NOT use question or other alias keys.`,
		`- semantic.edges[*] MUST use canonical keys id/type/from/to/provenance; do NOT use relation/source/target aliases.`,
		`- semantic.entities[*].provenance, semantic.edges[*].provenance, and semantic.findings[*].provenance MUST be objects; do NOT use arrays of legacy citation records.`,
		`- provenance.confidence values MUST stay numeric; string confidence values are invalid.`,
		`- semantic.findings[*] MUST keep id/severity/title/provenance and evidence inside provenance.evidence; do NOT use evidence_citation_ids or legacy inference/summary compatibility fields.`,
		`- semantic provenance.evidence[*] objects MUST carry repo/path; citation-only evidence objects and empty repo/path strings are invalid legacy drift.`,
		`- Do NOT add top-level step_contract, compatibility, or any alternate semantic wrapper fields to shard-pack-manifest.json.`,
	}
}

func CompactCollectManifestValidationChecklist(artifactRoot string) []string {
	return []string{
		fmt.Sprintf(`- artifact_root must remain exactly %q.`, artifactRoot),
		`- version must be integer 1.`,
		`- documents[] items use only id, kind, title, path, canonical_path, topics, citation_ids; documents[].path is artifact_root-relative.`,
		`- citations[] items use id, repo, path, claim_ids, document_ids; every documents[].citation_ids value references a citations[].id.`,
		`- citations[].path and semantic provenance evidence paths must be existing relative paths under the resolved repo root; do not cite guessed paths.`,
		`- semantic must include coverage, questions, entities, edges, findings; use coverage.observed/missing/notes and questions[].id + questions[].text.`,
		`- semantic.findings[] items, when present, must include id, severity, title, and provenance.`,
		`- semantic provenance/evidence objects, when present, must include concrete repo/path values; citation-only semantic evidence is forbidden.`,
		`- forbidden legacy aliases: step_contract, compatibility, covered_topics, question, relation, source, target, evidence_citation_ids, string confidence, summary as finding alias.`,
	}
}

func CollectManifestCanonicalExample() string {
	return strings.TrimSpace(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
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
      "claim_ids": ["claim.payments.readme.payments"],
      "document_ids": ["doc.payments.overview"]
    }
  ],
  "semantic": {
    "coverage": {
      "observed": ["services"],
      "missing": ["owner mappings"],
      "notes": ["Use canonical keys only."]
    },
    "questions": [
      {
        "id": "q.payments.owner",
        "text": "Who owns the payments service?"
      }
    ],
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
              "citation_id": "cite.payments.readme",
              "repo": "payments-service",
              "path": "README.md"
            }
          ]
        }
      }
    ],
    "edges": [
      {
        "id": "edge.payments.depends-on-ledger",
        "type": "depends_on",
        "from": "svc.payments",
        "to": "svc.ledger",
        "provenance": {
          "kind": "observation",
          "confidence": 0.7,
          "evidence": [
            {
              "citation_id": "cite.payments.readme",
              "repo": "payments-service",
              "path": "README.md"
            }
          ]
        }
      }
    ],
    "findings": [
      {
        "id": "finding.payments.owner",
        "severity": "medium",
        "title": "Missing owner mapping",
        "description": "Repository evidence names the payments service but does not identify an owning team.",
        "rule_id": "rule.owner.required",
        "related_ids": ["svc.payments"],
        "provenance": {
          "kind": "observation",
          "confidence": 0.6,
          "evidence": [
            {
              "citation_id": "cite.payments.readme",
              "repo": "payments-service",
              "path": "README.md"
            }
          ]
        }
      }
    ]
  }
}`)
}

func ValidatorVerdictContractLines() []string {
	return []string{
		`- validator-verdict.json MUST validate against the ACP validator-verdict contract.`,
		`- validator-verdict.json MUST include version=1, run_id, generated_at, verdict, and checked_paths.`,
		`- generated_at MUST be an RFC3339 UTC timestamp string (example: "2026-04-16T12:00:02Z").`,
		`- checked_paths MUST enumerate the staged final artifacts inspected by the validator.`,
		`- Optional fixed_paths/issues/findings/questions arrays are allowed, but they must keep canonical object shapes.`,
		`- issues[] items MUST use exactly the canonical validator issue shape: code, severity, message, and optional path/document_id/citation_id.`,
		`- issues[].severity MUST be "error" or "warning" only; do NOT use high/medium/low/critical in issues[].`,
		`- Do NOT put legacy finding-shaped fields inside issues[]: id, title, description, rule_id, related_paths, related_ids, provenance.`,
		`- findings[] items MUST use title + description + provenance; do NOT use summary as a finding alias.`,
		`- For observation provenance, findings[*].provenance.evidence[] MUST be non-empty and each evidence item MUST include non-empty repo and path values.`,
		`- owner-gap findings/questions remain visible, but owner-only residual evidence gaps may still return verdict=PASS when no technical validator issues remain.`,
	}
}

func ValidatorVerdictIssueCanonicalExample() string {
	return strings.TrimSpace(`{
  "code": "staged_index_missing",
  "severity": "error",
  "message": "final-run-index.json is missing from the staged final set.",
  "path": "reports/taskruns/run-1/staging/final/final-run-index.json"
}`)
}

func ValidatorVerdictCanonicalExample() string {
	return strings.TrimSpace(`{
  "version": 1,
  "run_id": "run-1",
  "generated_at": "2026-04-16T12:00:02Z",
  "verdict": "PASS",
  "summary": "Validator kept the owner-gap visible in findings/questions, but no blocking technical issues remain.",
  "checked_paths": [
    "reports/taskruns/run-1/staging/final/final-run-index.json",
    "reports/taskruns/run-1/staging/final/citation-index.json",
    "reports/taskruns/run-1/staging/final/reports/as-is/payments/overview.md"
  ],
  "fixed_paths": [],
  "findings": [
    {
      "id": "finding.payments.owner",
      "severity": "medium",
      "title": "Owner mapping remains unresolved",
      "description": "Validator could not confirm an owning team from the staged evidence set, but the staged docflow is otherwise technically valid.",
      "rule_id": "rule.owner.required",
      "related_ids": ["svc.payments"],
      "provenance": {
        "kind": "observation",
        "confidence": 0.7,
        "evidence": [
          {
            "citation_id": "cite.payments.readme",
            "repo": "payments-service",
            "path": "README.md"
          }
        ]
      }
    }
  ],
  "questions": [
    {
      "id": "q.payments.owner",
      "text": "Which team owns the payments service?"
    }
  ],
  "issues": []
}`)
}

func AsIsDraftManifestContractLines() []string {
	return []string{
		`- asis-draft-manifest.json MUST validate against the ACP runtime draft manifest contract.`,
		`- asis-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="as_is", agent_role, and outputs[]; optional top-level metadata is limited to summary and updated_at.`,
		`- Do NOT add legacy top-level fields such as repo_scopes, path_scopes, compatibility, generated_at, or alternate metadata envelopes.`,
		`- outputs[] MUST include exactly these required publish mappings: overview.md -> reports/as-is/overview.md, summary.md -> reports/coverage/summary.md, architect-summary.md -> reports/agent-outputs/architect/summary.md.`,
		`- Additional outputs are allowed only under reports/as-is/<domain>/overview.md.`,
		`- outputs[].path MUST stay relative to draft_final_root and outputs[].canonical_path MUST stay workspace-relative.`,
	}
}

func AsIsDraftManifestCanonicalExample() string {
	return strings.TrimSpace(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step2.asis_docs",
  "step_contract": "as_is",
  "agent_role": "architect",
  "summary": "Drafted canonical as-is reports from the staged final context.",
  "outputs": [
    {
      "path": "overview.md",
      "canonical_path": "reports/as-is/overview.md",
      "kind": "report",
      "title": "System Overview"
    },
    {
      "path": "summary.md",
      "canonical_path": "reports/coverage/summary.md",
      "kind": "report",
      "title": "Coverage Summary"
    },
    {
      "path": "architect-summary.md",
      "canonical_path": "reports/agent-outputs/architect/summary.md",
      "kind": "agent-output",
      "title": "Architect Summary"
    },
    {
      "path": "payments-overview.md",
      "canonical_path": "reports/as-is/payments/overview.md",
      "kind": "report",
      "title": "Payments Overview"
    }
  ]
}`)
}

func ProposalsDraftManifestContractLines() []string {
	return []string{
		`- proposals-draft-manifest.json MUST validate against the ACP runtime draft manifest contract.`,
		`- proposals-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="proposals", agent_role, outputs[], and optional summary/updated_at.`,
		`- outputs[].path MUST stay relative to draft_final_root and outputs[].canonical_path MUST stay workspace-relative.`,
		`- outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/*.`,
		`- outputs[].canonical_path values MUST be unique.`,
		`- Do NOT add legacy top-level fields such as pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps.`,
		`- Do NOT emit final-index-like proposal envelopes; proposals-draft-manifest.json is only the runtime draft publish map.`,
	}
}

func ProposalsDraftManifestCanonicalExample() string {
	return strings.TrimSpace(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "summary": "Drafted remediation proposals from validated findings.",
  "outputs": [
    {
      "path": "proposal.md",
      "canonical_path": "proposals/proposal-baseline/proposal.md",
      "kind": "proposal",
      "title": "Baseline Remediation Proposal"
    },
    {
      "path": "changelog.md",
      "canonical_path": "reports/changelog/run-1.md",
      "kind": "changelog",
      "title": "Proposal Changelog"
    }
  ]
}`)
}
