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
	if !collectManifestSemanticBootstrapOnly(manifest.Semantic) {
		return false
	}
	for _, document := range manifest.Documents {
		text := documentTextByPath[strings.TrimSpace(document.Path)]
		if collectDocumentBootstrapOnly(text) {
			return true
		}
	}
	return false
}

func collectDocumentBootstrapOnly(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	if strings.Contains(text, CollectBootstrapReplaceMarker) {
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
