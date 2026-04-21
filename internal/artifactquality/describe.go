package artifactquality

import "strings"

func DescribeAssessmentProblem(assessment ManifestAssessment, err error) string {
	if err != nil {
		return "shard-pack-manifest.json is missing or invalid: " + strings.TrimSpace(err.Error())
	}
	if !assessment.ManifestPresent {
		return "shard-pack-manifest.json is missing"
	}
	if assessment.Rich {
		return "shard-pack-manifest.json is rich"
	}

	problems := []string{}
	if assessment.DocumentCount == 0 {
		problems = append(problems, "no documents")
	}
	if assessment.LinkedDocumentCount == 0 {
		problems = append(problems, "documents have no citation_ids")
	}
	if assessment.CitationCount == 0 {
		problems = append(problems, "no linked citations")
	}
	if assessment.RepoSpecificCitationCount == 0 {
		problems = append(problems, "no repo-specific citations")
	}
	if assessment.GenericRuntimeSummaryOnly {
		problems = append(problems, `citations collapse to generic "cite.runtime-summary"`)
	}
	if len(problems) == 0 {
		return "shard-pack-manifest.json is not rich enough"
	}
	return "shard-pack-manifest.json is not rich enough: " + strings.Join(problems, ", ")
}
