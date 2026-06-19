package artifactquality

import (
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

const CollectBootstrapReplaceMarker = "ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT"

func CollectManifestBootstrapOnly(manifest contracts.ShardPackManifest, documentTextByPath map[string]string) bool {
	if len(manifest.Documents) == 0 || len(documentTextByPath) == 0 {
		return false
	}
	hasBootstrapDocument := false
	for _, document := range manifest.Documents {
		text := documentTextByPath[strings.TrimSpace(document.Path)]
		if collectDocumentBootstrapOnly(text) {
			if strings.Contains(text, CollectBootstrapReplaceMarker) {
				return true
			}
			hasBootstrapDocument = true
		}
	}
	if !hasBootstrapDocument {
		return false
	}
	return collectManifestSemanticBootstrapOnly(manifest.Semantic)
}

func CollectManifestSemanticBootstrapOnly(manifest contracts.ShardPackManifest) bool {
	return collectManifestSemanticBootstrapOnly(manifest.Semantic)
}

func CollectDocumentBootstrapOnly(text string) bool {
	return collectDocumentBootstrapOnly(text)
}

func collectDocumentBootstrapOnly(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	if strings.Contains(text, CollectBootstrapReplaceMarker) {
		return true
	}
	if collectDocumentInitialSeedOnly(lower) {
		return true
	}
	if collectDocumentRecoveryScaffoldOnly(lower) {
		return true
	}
	if collectDocumentTemporaryRecoveryOnly(lower) {
		return true
	}
	required := []string{
		"## observations",
		"repository scope:",
		"primary scoped evidence path:",
		"## evidence",
		"primary evidence path:",
		"## follow-up",
		"owner mapping evidence not confirmed from the initial scoped evidence path",
	}
	for _, needle := range required {
		if !strings.Contains(lower, needle) {
			return false
		}
	}
	return true
}

func collectDocumentInitialSeedOnly(lower string) bool {
	if strings.Contains(lower, "## evidence summary") &&
		strings.Contains(lower, "## evidence surface") &&
		strings.Contains(lower, "## initial findings") &&
		strings.Contains(lower, "## coverage gaps") &&
		strings.Contains(lower, "scoped repository evidence available to this collect shard") &&
		strings.Contains(lower, "ownership, runtime responsibility, and escalation details need confirmation") &&
		strings.Contains(lower, "confirm concrete owners, runtime responsibilities, dependencies, and operational escalation paths") {
		return true
	}
	if strings.Contains(lower, "this initial collect pair is a seed-only scaffold") &&
		strings.Contains(lower, "overwrite it with repository-specific evidence before final exit") {
		return true
	}
	return false
}

func collectDocumentRecoveryScaffoldOnly(lower string) bool {
	if strings.Contains(lower, "no repository evidence was emitted") &&
		(strings.Contains(lower, "bounded collection failure") ||
			strings.Contains(lower, "current evidence result: no file content observed") ||
			strings.Contains(lower, "first bounded evidence read") ||
			strings.Contains(lower, "bounded read failed")) {
		return true
	}
	if strings.Contains(lower, "## recovery bootstrap") &&
		strings.Contains(lower, "evidence candidate used for the recovery manifest:") &&
		strings.Contains(lower, "replace this recovery bootstrap with concrete repository evidence") {
		return true
	}
	if strings.Contains(lower, "## recovery summary") &&
		strings.Contains(lower, "evidence candidate used for the recovery manifest:") &&
		strings.Contains(lower, "## evidence candidates") &&
		strings.Contains(lower, "additional repository-specific details should be enriched by the provider") &&
		strings.Contains(lower, "## remaining questions") &&
		strings.Contains(lower, "confirm concrete ownership, runtime responsibilities, and operational escalation evidence") {
		return true
	}
	if strings.Contains(lower, "## recovery evidence summary") &&
		(strings.Contains(lower, "validation-ready collect recovery fallback") ||
			strings.Contains(lower, "seed-only collect recovery fallback") ||
			strings.Contains(lower, "collect recovery fallback")) &&
		strings.Contains(lower, "## recovery notes") &&
		strings.Contains(lower, "downstream compilation can preserve traceability instead of accepting an empty or marker-only shard") {
		return true
	}
	return false
}

func collectDocumentTemporaryRecoveryOnly(lower string) bool {
	if strings.Contains(lower, "first bounded evidence read was attempted") &&
		strings.Contains(lower, "initial artifact records only") &&
		strings.Contains(lower, "will be repaired with concrete") {
		return true
	}
	if strings.Contains(lower, "first bounded read was attempted") &&
		strings.Contains(lower, "requires concrete evidence repair") {
		return true
	}
	if strings.Contains(lower, "not yet confirmed after interrupted first read") &&
		strings.Contains(lower, "concrete evidence repair") {
		return true
	}
	if strings.Contains(lower, "shell glob handling interrupted") &&
		strings.Contains(lower, "will be repaired with concrete file-level evidence") {
		return true
	}
	return false
}

func collectManifestSemanticBootstrapOnly(semantic contracts.SemanticSnapshot) bool {
	if len(semantic.Entities) < 2 || len(semantic.Edges) == 0 || len(semantic.Findings) == 0 {
		return false
	}
	for _, edge := range semantic.Edges {
		edgeType := strings.ToLower(strings.TrimSpace(edge.Type))
		edgeName := strings.ToLower(strings.TrimSpace(edge.Name))
		if edgeType != "contains" && !strings.Contains(edgeName, "contains scoped surface") {
			return false
		}
	}
	for _, finding := range semantic.Findings {
		title := strings.ToLower(strings.TrimSpace(finding.Title))
		description := strings.ToLower(strings.TrimSpace(finding.Description))
		ruleID := strings.ToLower(strings.TrimSpace(finding.RuleID))
		if title != "owner mapping not confirmed" ||
			!strings.Contains(ruleID, "owner.mapping") ||
			!strings.Contains(description, "scoped evidence identifies") ||
			!strings.Contains(description, "does not confirm an owning team") {
			return false
		}
	}
	if collectCoverageHasBootstrapNote(semantic.Coverage.Notes) {
		return true
	}
	for _, missing := range semantic.Coverage.Missing {
		normalized := strings.ToLower(strings.TrimSpace(missing))
		if strings.Contains(normalized, "owner mapping evidence not confirmed from scoped repository files") {
			return true
		}
	}
	return false
}

func collectCoverageHasBootstrapNote(notes []string) bool {
	for _, note := range notes {
		normalized := strings.ToLower(strings.TrimSpace(note))
		if strings.Contains(normalized, "collect manifest covers the assigned shard scope") &&
			strings.Contains(normalized, "evidence paths listed in citations") {
			return true
		}
	}
	return false
}
