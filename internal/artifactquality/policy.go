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
		`- citations[].claim_ids MUST be globally unique across the assembled staged final set; do NOT reuse the same claim id across different shard/citation surfaces.`,
		`- Build claim_ids as a semantic stem plus shard slug (example: claim.owner.<shard-slug>), and add a deterministic numeric suffix when a shard-local collision remains.`,
		`- citations[].document_ids MUST point back to ids that exist in documents[].id.`,
		`- compatibility MUST include coverage, questions, entities, edges, and findings; questions/entities/edges/findings MUST be arrays even when empty.`,
		`- canonical_path MUST be a stable promoted path under reports/as-is, reports/findings, reports/coverage, reports/agent-outputs, reports/diagrams, or proposals.`,
		`- Do NOT use reports/taskruns/... staging paths as canonical_path.`,
	}
}

func ClaimIDContractLines() []string {
	return []string{
		`- claim_ids are a global staged-final-set namespace, not a per-shard local namespace.`,
		`- Do NOT reuse the same claim_id across different citations or shards, even when they describe related evidence.`,
		`- When deriving claim_ids, append the shard slug and then a deterministic numeric suffix if another collision remains.`,
	}
}
