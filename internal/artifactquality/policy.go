package artifactquality

import "fmt"

func CollectManifestContractLines(artifactRoot string) []string {
	return []string{
		fmt.Sprintf(`- shard-pack-manifest.json MUST validate against the ACP shard-pack-manifest contract and use artifact_root = %q.`, artifactRoot),
		`- version MUST be integer 1 (not "1.0.0" or any other string form).`,
		`- Each documents[] item MUST include: id, kind, title, path, canonical_path, topics, citation_ids.`,
		`- Do NOT use documents[].citations; only documents[].citation_ids is allowed.`,
		`- Each citations[] item MUST include: id, repo, path, claim_ids, document_ids.`,
		`- citations[].id values MUST be unique, and every documents[].citation_ids entry MUST reference an existing citations[].id.`,
		`- citations[].document_ids MUST point back to ids that exist in documents[].id.`,
		`- canonical_path MUST be a stable promoted path under reports/as-is, reports/findings, reports/coverage, reports/agent-outputs, reports/diagrams, or proposals.`,
		`- Do NOT use reports/taskruns/... staging paths as canonical_path.`,
	}
}
