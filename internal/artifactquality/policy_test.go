package artifactquality

import (
	"strings"
	"testing"
)

func TestProposalsDraftManifestContractLinesDefineCanonicalSurface(t *testing.T) {
	t.Parallel()

	joined := strings.Join(ProposalsDraftManifestContractLines(), "\n")
	required := []string{
		`proposals-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="proposals", agent_role, outputs[], and optional summary/updated_at.`,
		`outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/*.`,
		`outputs[].canonical_path values MUST be unique.`,
		`pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps`,
	}
	for _, needle := range required {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected proposals contract lines to contain %q, got:\n%s", needle, joined)
		}
	}
}

func TestProposalsDraftManifestCanonicalExampleUsesStrictContract(t *testing.T) {
	t.Parallel()

	example := ProposalsDraftManifestCanonicalExample()
	required := []string{
		`"version": 1`,
		`"step_id": "init.step4.proposals"`,
		`"step_contract": "proposals"`,
		`"agent_role": "architect"`,
		`"outputs": [`,
		`"canonical_path": "proposals/proposal-baseline/proposal.md"`,
		`"canonical_path": "reports/changelog/run-1.md"`,
	}
	for _, needle := range required {
		if !strings.Contains(example, needle) {
			t.Fatalf("expected proposals canonical example to contain %q, got:\n%s", needle, example)
		}
	}
	for _, forbidden := range []string{`"pipeline"`, `"proposals": [`} {
		if strings.Contains(example, forbidden) {
			t.Fatalf("expected proposals canonical example to omit legacy token %q, got:\n%s", forbidden, example)
		}
	}
}

func TestCompactCollectManifestValidationChecklistIncludesCitationIDUniqueness(t *testing.T) {
	t.Parallel()

	joined := strings.Join(CompactCollectManifestValidationChecklist("reports/taskruns/run-1/staging/shards/root"), "\n")
	for _, needle := range []string{
		"citations[].id values must be unique",
		"derive each citations[].id from the shard/document stem plus the repo path slug",
		"every documents[].citation_ids value references a citations[].id",
	} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected compact collect checklist to contain %q, got:\n%s", needle, joined)
		}
	}
}

func TestValidatorVerdictContractLinesKeepSourceEvidenceAdvisory(t *testing.T) {
	t.Parallel()

	joined := strings.Join(ValidatorVerdictContractLines(), "\n")
	for _, needle := range []string{
		"questions[] items MUST include id and text",
		"only id, text, priority, and related_ids",
		"citation_ids and all other extra question fields are forbidden",
		"PASS/FAIL is limited to staged artifact integrity and contract correctness",
		"security, privacy, compliance, architecture-risk, or operational observations",
		"MUST NOT produce verdict=FAIL or error-severity issues[]",
		"not a repository evidence path",
	} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected validator contract lines to contain %q, got:\n%s", needle, joined)
		}
	}
}
