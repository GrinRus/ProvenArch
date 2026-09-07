package reports

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type Artifact struct {
	Path  string
	Kind  string
	Label string
}

type DomainTaskEnvelope struct {
	ContractVersion int              `json:"contract_version"`
	AgentID         string           `json:"agent_id"`
	DomainID        string           `json:"domain_id"`
	RepoScope       string           `json:"repo_scope,omitempty"`
	Unresolved      []string         `json:"unresolved,omitempty"`
	Inputs          DomainTaskInputs `json:"inputs"`
	OutputPath      string           `json:"output_path"`
}

type DomainTaskInputs struct {
	DomainCardPath      string `json:"domain_card_path"`
	CoverageSummaryPath string `json:"coverage_summary_path"`
	QuestionsPath       string `json:"questions_path"`
	ModelEntitiesGlob   string `json:"model_entities_glob"`
	FindingsPath        string `json:"findings_path"`
}

type Compiler struct {
	workspace workspace.Root
}

func NewCompiler(ws workspace.Root) Compiler {
	return Compiler{workspace: ws}
}

func (c Compiler) WriteCoverage(coverage *contracts.Coverage, questions []contracts.Question, renderCtx ReportRenderContext) ([]Artifact, error) {
	renderCtx = NormalizeReportRenderContext(renderCtx)
	if coverage == nil {
		coverage = &contracts.Coverage{}
	}

	summary := strings.Builder{}
	summary.WriteString("# Coverage Summary\n\n")
	writeAnalysisBanner(&summary, renderCtx)
	writeStringListWithFallback(&summary, "Observed", coverage.Observed, coverageFallback("observed", renderCtx))
	writeStringListWithFallback(&summary, "Missing", coverage.Missing, coverageFallback("missing", renderCtx))
	writeStringListWithFallback(&summary, "Notes", coverage.Notes, coverageFallback("notes", renderCtx))
	if err := c.workspace.WriteFile("reports/coverage/summary.md", []byte(summary.String())); err != nil {
		return nil, err
	}

	questionsReport := strings.Builder{}
	questionsReport.WriteString("# Open Questions\n\n")
	writeAnalysisBanner(&questionsReport, renderCtx)
	if len(questions) == 0 {
		questionsReport.WriteString(openQuestionsFallback(renderCtx))
		questionsReport.WriteString("\n")
	} else {
		sort.Slice(questions, func(i, j int) bool { return questions[i].ID < questions[j].ID })
		for _, question := range questions {
			questionsReport.WriteString(fmt.Sprintf("- `%s` %s\n", question.ID, question.Text))
		}
	}
	if err := c.workspace.WriteFile("reports/coverage/open-questions.md", []byte(questionsReport.String())); err != nil {
		return nil, err
	}

	artifacts := []Artifact{
		{Path: "reports/coverage/summary.md", Kind: "report", Label: "Coverage Summary"},
		{Path: "reports/coverage/open-questions.md", Kind: "report", Label: "Open Questions"},
	}
	return artifacts, nil
}

func (c Compiler) WriteFindings(findings []contracts.Finding, renderCtx ReportRenderContext) ([]Artifact, error) {
	renderCtx = NormalizeReportRenderContext(renderCtx)
	sort.Slice(findings, func(i, j int) bool { return findings[i].ID < findings[j].ID })

	content := strings.Builder{}
	content.WriteString("# Findings\n\n")
	writeAnalysisBanner(&content, renderCtx)
	if len(findings) == 0 {
		content.WriteString(findingsFallback(renderCtx))
		content.WriteString("\n")
	} else {
		for _, finding := range findings {
			content.WriteString(fmt.Sprintf("## %s\n\n", finding.Title))
			content.WriteString(fmt.Sprintf("- ID: `%s`\n", finding.ID))
			content.WriteString(fmt.Sprintf("- Severity: `%s`\n", finding.Severity))
			if strings.TrimSpace(finding.RuleID) != "" {
				content.WriteString(fmt.Sprintf("- Rule ID: `%s`\n", finding.RuleID))
			}
			if len(finding.RelatedIDs) > 0 {
				content.WriteString(fmt.Sprintf("- Related IDs: %s\n", renderBacktickList(uniqueSorted(append([]string(nil), finding.RelatedIDs...)))))
			}
			if finding.Description != "" {
				content.WriteString(fmt.Sprintf("- Description: %s\n", finding.Description))
			}
			if len(finding.Provenance.Evidence) > 0 {
				refs := make([]string, 0, len(finding.Provenance.Evidence))
				for _, evidence := range finding.Provenance.Evidence {
					repo := strings.TrimSpace(evidence.Repo)
					path := strings.TrimSpace(evidence.Path)
					if repo == "" && path == "" {
						continue
					}
					if repo == "" {
						refs = append(refs, path)
						continue
					}
					if path == "" {
						refs = append(refs, repo)
						continue
					}
					refs = append(refs, repo+":"+path)
				}
				if len(refs) > 0 {
					content.WriteString(fmt.Sprintf("- Evidence: %s\n", renderBacktickList(uniqueSorted(refs))))
				}
			}
			content.WriteString("\n")
		}
	}
	if err := c.workspace.WriteFile("reports/findings/findings.md", []byte(content.String())); err != nil {
		return nil, err
	}
	return []Artifact{
		{Path: "reports/findings/findings.md", Kind: "report", Label: "Findings"},
	}, nil
}

func (c Compiler) WriteDomainOutputs(domainReports map[string]string) ([]Artifact, error) {
	var artifacts []Artifact
	domains := make([]string, 0, len(domainReports))
	for domain := range domainReports {
		domains = append(domains, domain)
	}
	sort.Strings(domains)

	for _, domain := range domains {
		path := fmt.Sprintf("reports/agent-outputs/domains/%s.md", domain)
		if err := c.workspace.WriteFile(path, []byte(domainReports[domain])); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, Artifact{
			Path:  path,
			Kind:  "agent-output",
			Label: domain,
		})
	}
	return artifacts, nil
}

func (c Compiler) WriteDomainTaskEnvelopes(envelopes []DomainTaskEnvelope) ([]Artifact, error) {
	if len(envelopes) == 0 {
		return nil, nil
	}
	sorted := append([]DomainTaskEnvelope(nil), envelopes...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].DomainID < sorted[j].DomainID
	})

	artifacts := make([]Artifact, 0, len(sorted))
	for _, envelope := range sorted {
		content, err := json.MarshalIndent(envelope, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal domain task envelope for %s: %w", envelope.DomainID, err)
		}
		path := fmt.Sprintf("reports/agent-outputs/domains/%s.task-envelope.json", sanitizeProposalSlug(envelope.DomainID))
		if err := c.workspace.WriteFile(path, append(content, '\n')); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, Artifact{
			Path:  path,
			Kind:  "agent-output",
			Label: envelope.DomainID + " task envelope",
		})
	}
	sortArtifacts(artifacts)
	return artifacts, nil
}

func (c Compiler) WriteIterationChangelog(runID string, pipeline string, artifacts []Artifact, startedAt time.Time, finishedAt time.Time) (Artifact, error) {
	datePart := finishedAt.UTC().Format("2006-01-02")
	path := fmt.Sprintf("reports/changelog/%s-%s.md", datePart, sanitizeProposalSlug(runID))

	content := strings.Builder{}
	content.WriteString(fmt.Sprintf("# Iteration %s\n\n", runID))
	content.WriteString(fmt.Sprintf("- Pipeline: `%s`\n", pipeline))
	content.WriteString(fmt.Sprintf("- Started: `%s`\n", startedAt.UTC().Format(time.RFC3339)))
	content.WriteString(fmt.Sprintf("- Finished: `%s`\n\n", finishedAt.UTC().Format(time.RFC3339)))
	content.WriteString("## Materialized artifacts\n\n")
	for _, artifact := range artifacts {
		content.WriteString(fmt.Sprintf("- `%s` (%s)\n", artifact.Path, artifact.Kind))
	}

	if err := c.workspace.WriteFile(path, []byte(content.String())); err != nil {
		return Artifact{}, err
	}
	return Artifact{Path: path, Kind: "changelog", Label: "Iteration Changelog"}, nil
}

func filterEntitiesByType(entities []contracts.Entity, entityType string) []contracts.Entity {
	var filtered []contracts.Entity
	for _, entity := range entities {
		if entity.Type == entityType {
			filtered = append(filtered, entity)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].ID < filtered[j].ID })
	return filtered
}

func writeStringListWithFallback(builder *strings.Builder, title string, values []string, fallback string) {
	builder.WriteString(fmt.Sprintf("## %s\n\n", title))
	if len(values) == 0 {
		builder.WriteString(strings.TrimSpace(fallback))
		builder.WriteString("\n\n")
		return
	}
	for _, value := range values {
		builder.WriteString(fmt.Sprintf("- %s\n", value))
	}
	builder.WriteString("\n")
}

func writeAnalysisBanner(builder *strings.Builder, renderCtx ReportRenderContext) {
	renderCtx = NormalizeReportRenderContext(renderCtx)
	if !renderCtx.IsIncomplete() {
		return
	}
	builder.WriteString("> ")
	builder.WriteString(analysisBannerHeadline(renderCtx))
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("> Collect status: %s (planned=%d succeeded=%d failed=%d)\n",
		renderCtx.Collect.Status,
		renderCtx.Collect.PlannedShards,
		renderCtx.Collect.SucceededShards,
		renderCtx.Collect.FailedShards,
	))
	builder.WriteString(fmt.Sprintf("> Findings status: %s (planned=%d succeeded=%d failed=%d)\n",
		renderCtx.Findings.Status,
		renderCtx.Findings.PlannedShards,
		renderCtx.Findings.SucceededShards,
		renderCtx.Findings.FailedShards,
	))
	if len(renderCtx.Reasons) > 0 {
		builder.WriteString(fmt.Sprintf("> Reasons: %s\n", strings.Join(renderCtx.Reasons, ", ")))
	}
	builder.WriteString("\n")
}

func analysisBannerHeadline(renderCtx ReportRenderContext) string {
	renderCtx = NormalizeReportRenderContext(renderCtx)
	if renderCtx.Collect.Status == EvidenceStatusPartial || renderCtx.Findings.Status == EvidenceStatusPartial {
		if renderCtx.Collect.Status != EvidenceStatusUnusable && renderCtx.Findings.Status != EvidenceStatusUnusable && renderCtx.Findings.Status != EvidenceStatusSkipped {
			return "Partial analysis. Some shards failed; downstream content may be incomplete."
		}
		return "Analysis incomplete. Some shards failed; downstream content may be incomplete."
	}
	return "Analysis incomplete."
}

func coverageFallback(section string, renderCtx ReportRenderContext) string {
	switch section {
	case "observed":
		switch renderCtx.Collect.Status {
		case EvidenceStatusUnusable:
			return "Unavailable due to incomplete analysis."
		case EvidenceStatusPartial:
			return "May be incomplete because some shards failed."
		}
	case "missing":
		switch renderCtx.Collect.Status {
		case EvidenceStatusUnusable:
			return "Unknown due to incomplete analysis."
		case EvidenceStatusPartial:
			return "May be incomplete because some shards failed."
		}
	case "notes":
		if renderCtx.IsIncomplete() {
			return "Analysis incomplete. See banner above."
		}
	}
	return "None."
}

func openQuestionsFallback(renderCtx ReportRenderContext) string {
	switch renderCtx.Collect.Status {
	case EvidenceStatusUnusable:
		return "Open questions unavailable due to incomplete analysis."
	case EvidenceStatusPartial:
		return "Open questions may be incomplete because some shards failed."
	default:
		return "No open questions."
	}
}

func findingsFallback(renderCtx ReportRenderContext) string {
	switch renderCtx.Findings.Status {
	case EvidenceStatusUnusable, EvidenceStatusSkipped:
		return "Findings unavailable because analysis did not complete."
	case EvidenceStatusPartial:
		return "Findings may be incomplete because some shards failed."
	default:
		if renderCtx.IsIncomplete() {
			return "Findings may be incomplete because analysis did not complete."
		}
		return "No findings reported."
	}
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	uniq := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		uniq = append(uniq, trimmed)
	}
	sort.Strings(uniq)
	return uniq
}

func renderBacktickList(values []string) string {
	if len(values) == 0 {
		return ""
	}
	rendered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		rendered = append(rendered, fmt.Sprintf("`%s`", value))
	}
	return strings.Join(rendered, ", ")
}

func sortArtifacts(artifacts []Artifact) {
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].Kind == artifacts[j].Kind {
			return artifacts[i].Path < artifacts[j].Path
		}
		return artifacts[i].Kind < artifacts[j].Kind
	})
}

func sanitizeProposalSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "run"
	}

	var out []rune
	prevDash := false
	for _, r := range value {
		isAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isAlpha || isDigit {
			out = append(out, r)
			prevDash = false
			continue
		}
		if prevDash {
			continue
		}
		out = append(out, '-')
		prevDash = true
	}
	slug := strings.Trim(string(out), "-")
	if slug == "" {
		return "run"
	}
	return slug
}
