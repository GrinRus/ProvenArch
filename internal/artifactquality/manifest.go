package artifactquality

import (
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

const shardPackManifestFile = "shard-pack-manifest.json"

type ManifestAssessment struct {
	ManifestPresent           bool
	DocumentCount             int
	LinkedDocumentCount       int
	CitationCount             int
	RepoSpecificCitationCount int
	GenericRuntimeSummaryOnly bool
	Rich                      bool
}

func AssessManifest(manifest contracts.ShardPackManifest) ManifestAssessment {
	assessment := ManifestAssessment{ManifestPresent: true}
	referencedCitationIDs := map[string]struct{}{}
	for _, document := range manifest.Documents {
		pathValue := strings.TrimSpace(document.Path)
		canonicalPathValue := strings.TrimSpace(document.CanonicalPath)
		if pathValue == "" && canonicalPathValue == "" {
			continue
		}
		assessment.DocumentCount++
		linkCount := 0
		for _, citationID := range document.CitationIDs {
			trimmedID := strings.TrimSpace(citationID)
			if trimmedID == "" {
				continue
			}
			referencedCitationIDs[trimmedID] = struct{}{}
			linkCount++
		}
		if linkCount > 0 {
			assessment.LinkedDocumentCount++
		}
	}

	uniqueLinkedCitationIDs := map[string]struct{}{}
	genericRuntimeSummaryOnly := true
	for _, citation := range manifest.Citations {
		citationID := strings.TrimSpace(citation.ID)
		if citationID == "" {
			continue
		}
		if _, linked := referencedCitationIDs[citationID]; !linked {
			continue
		}
		uniqueLinkedCitationIDs[citationID] = struct{}{}
		if !IsGenericRuntimeSummaryCitation(citationID) {
			genericRuntimeSummaryOnly = false
		}
		if strings.TrimSpace(citation.Repo) != "" &&
			strings.TrimSpace(citation.Path) != "" &&
			!IsGenericRuntimeSummaryCitation(citationID) {
			assessment.RepoSpecificCitationCount++
		}
	}

	assessment.CitationCount = len(uniqueLinkedCitationIDs)
	assessment.GenericRuntimeSummaryOnly = assessment.CitationCount > 0 && genericRuntimeSummaryOnly
	assessment.Rich = assessment.DocumentCount > 0 &&
		assessment.LinkedDocumentCount > 0 &&
		assessment.CitationCount > 0 &&
		assessment.RepoSpecificCitationCount > 0 &&
		!assessment.GenericRuntimeSummaryOnly
	return assessment
}

func HasRepoSpecificCitationSurface(manifest contracts.ShardPackManifest) bool {
	return AssessManifest(manifest).RepoSpecificCitationCount > 0
}

func IsGenericRuntimeSummaryCitation(id string) bool {
	normalized := strings.ToLower(strings.TrimSpace(id))
	return normalized == "cite.runtime-summary" || strings.HasPrefix(normalized, "cite.runtime-summary.")
}
