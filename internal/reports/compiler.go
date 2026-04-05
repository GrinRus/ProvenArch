package reports

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
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

func (c Compiler) CompileAsIs(entities []contracts.Entity, edges []contracts.Edge) ([]Artifact, error) {
	var artifacts []Artifact

	serviceEntities := filterEntitiesByType(entities, "service")
	externalEntities := filterEntitiesByType(entities, "external.system")
	datastoreEntities := filterEntitiesByType(entities, "datastore")

	overview := strings.Builder{}
	overview.WriteString("# As-Is Overview\n\n")
	overview.WriteString(fmt.Sprintf("- Services: %d\n", len(serviceEntities)))
	overview.WriteString(fmt.Sprintf("- Dependencies (edges): %d\n", len(edges)))
	overview.WriteString(fmt.Sprintf("- External systems: %d\n", len(externalEntities)))
	overview.WriteString(fmt.Sprintf("- Datastores: %d\n", len(datastoreEntities)))
	if err := c.workspace.WriteFile("reports/as-is/overview.md", []byte(overview.String())); err != nil {
		return nil, err
	}
	artifacts = append(artifacts, Artifact{
		Path:  "reports/as-is/overview.md",
		Kind:  "report",
		Label: "System Overview",
	})

	serviceCatalog := strings.Builder{}
	serviceCatalog.WriteString("# Service Catalog\n\n")
	if len(serviceEntities) == 0 {
		serviceCatalog.WriteString("No services found.\n")
	} else {
		serviceCatalog.WriteString("| ID | Name |\n|---|---|\n")
		for _, entity := range serviceEntities {
			serviceCatalog.WriteString(fmt.Sprintf("| %s | %s |\n", entity.ID, entity.Name))
		}
	}
	if err := c.workspace.WriteFile("reports/as-is/service-catalog.md", []byte(serviceCatalog.String())); err != nil {
		return nil, err
	}
	artifacts = append(artifacts, Artifact{
		Path:  "reports/as-is/service-catalog.md",
		Kind:  "report",
		Label: "Service Catalog",
	})

	for _, service := range serviceEntities {
		content := strings.Builder{}
		content.WriteString(fmt.Sprintf("# %s\n\n", service.Name))
		content.WriteString(fmt.Sprintf("- ID: `%s`\n", service.ID))
		content.WriteString(fmt.Sprintf("- Type: `%s`\n", service.Type))
		if service.OwnerTeamID != "" {
			content.WriteString(fmt.Sprintf("- Owner team: `%s`\n", service.OwnerTeamID))
		}
		related := relatedEdges(service.ID, edges)
		content.WriteString(fmt.Sprintf("- Related edges: %d\n\n", len(related)))
		for _, edge := range related {
			content.WriteString(fmt.Sprintf("- `%s`: `%s` -> `%s`\n", edge.Type, edge.From, edge.To))
		}

		filePath := fmt.Sprintf("reports/as-is/services/%s.md", service.ID)
		if err := c.workspace.WriteFile(filePath, []byte(content.String())); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, Artifact{
			Path:  filePath,
			Kind:  "report",
			Label: service.Name,
		})
	}

	integrations := strings.Builder{}
	integrations.WriteString("# Integrations\n\n")
	if len(externalEntities) == 0 {
		integrations.WriteString("No external systems found.\n")
	} else {
		for _, ext := range externalEntities {
			integrations.WriteString(fmt.Sprintf("- `%s` %s\n", ext.ID, ext.Name))
		}
	}
	if err := c.workspace.WriteFile("reports/as-is/integrations.md", []byte(integrations.String())); err != nil {
		return nil, err
	}
	artifacts = append(artifacts, Artifact{
		Path:  "reports/as-is/integrations.md",
		Kind:  "report",
		Label: "Integrations",
	})

	datastores := strings.Builder{}
	datastores.WriteString("# Datastores\n\n")
	if len(datastoreEntities) == 0 {
		datastores.WriteString("No datastores found.\n")
	} else {
		for _, db := range datastoreEntities {
			datastores.WriteString(fmt.Sprintf("- `%s` %s\n", db.ID, db.Name))
		}
	}
	if err := c.workspace.WriteFile("reports/as-is/datastores.md", []byte(datastores.String())); err != nil {
		return nil, err
	}
	artifacts = append(artifacts, Artifact{
		Path:  "reports/as-is/datastores.md",
		Kind:  "report",
		Label: "Datastores",
	})

	ciCD := "# CI/CD\n\nCI/CD evidence is surfaced through coverage and findings artifacts.\n"
	if err := c.workspace.WriteFile("reports/as-is/ci-cd.md", []byte(ciCD)); err != nil {
		return nil, err
	}
	artifacts = append(artifacts, Artifact{
		Path:  "reports/as-is/ci-cd.md",
		Kind:  "report",
		Label: "CI/CD",
	})

	sortArtifacts(artifacts)
	return artifacts, nil
}

func (c Compiler) WriteCoverage(coverage *contracts.Coverage, questions []contracts.Question) ([]Artifact, error) {
	if coverage == nil {
		coverage = &contracts.Coverage{}
	}

	summary := strings.Builder{}
	summary.WriteString("# Coverage Summary\n\n")
	writeStringList(&summary, "Observed", coverage.Observed)
	writeStringList(&summary, "Missing", coverage.Missing)
	writeStringList(&summary, "Notes", coverage.Notes)
	if err := c.workspace.WriteFile("reports/coverage/summary.md", []byte(summary.String())); err != nil {
		return nil, err
	}

	questionsReport := strings.Builder{}
	questionsReport.WriteString("# Open Questions\n\n")
	if len(questions) == 0 {
		questionsReport.WriteString("No open questions.\n")
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

func (c Compiler) WriteFindings(findings []contracts.Finding) ([]Artifact, error) {
	sort.Slice(findings, func(i, j int) bool { return findings[i].ID < findings[j].ID })

	content := strings.Builder{}
	content.WriteString("# Findings\n\n")
	if len(findings) == 0 {
		content.WriteString("No findings reported.\n")
	} else {
		for _, finding := range findings {
			content.WriteString(fmt.Sprintf("## %s\n\n", finding.Title))
			content.WriteString(fmt.Sprintf("- ID: `%s`\n", finding.ID))
			content.WriteString(fmt.Sprintf("- Severity: `%s`\n", finding.Severity))
			if finding.Description != "" {
				content.WriteString(fmt.Sprintf("- Description: %s\n", finding.Description))
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

func (c Compiler) WriteDocArtifacts(artifactsInput []contracts.DocArtifact) ([]Artifact, error) {
	if len(artifactsInput) == 0 {
		return nil, nil
	}
	artifacts := append([]contracts.DocArtifact(nil), artifactsInput...)
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].ID < artifacts[j].ID })

	content := strings.Builder{}
	content.WriteString("# Doc Artifact Metadata\n\n")
	for _, artifact := range artifacts {
		content.WriteString(fmt.Sprintf("## %s\n\n", artifact.Title))
		content.WriteString(fmt.Sprintf("- ID: `%s`\n", artifact.ID))
		content.WriteString(fmt.Sprintf("- Kind: `%s`\n", artifact.Kind))
		content.WriteString(fmt.Sprintf("- Path: `%s`\n", artifact.Path))
		if strings.TrimSpace(artifact.Format) != "" {
			content.WriteString(fmt.Sprintf("- Format: `%s`\n", artifact.Format))
		}
		content.WriteString("\n")
	}

	path := "reports/taskruns/doc-artifacts.md"
	if err := c.workspace.WriteFile(path, []byte(content.String())); err != nil {
		return nil, err
	}
	return []Artifact{
		{Path: path, Kind: "taskrun", Label: "Doc Artifact Metadata"},
	}, nil
}

func (c Compiler) CompileProposals(findings []contracts.Finding) ([]Artifact, error) {
	proposalID := fmt.Sprintf("proposal-%s", proposalSlugFromFindings(findings))
	baseDir := fmt.Sprintf("proposals/%s", proposalID)

	proposalBody := strings.Builder{}
	proposalBody.WriteString("# Improvement Proposal\n\n")
	proposalBody.WriteString("This proposal is generated from findings and baseline charter constraints.\n\n")
	if len(findings) == 0 {
		proposalBody.WriteString("No findings available. Keep monitoring coverage and refresh pipeline.\n")
	} else {
		proposalBody.WriteString("## Findings addressed\n\n")
		for _, finding := range findings {
			proposalBody.WriteString(fmt.Sprintf("- `%s` %s\n", finding.ID, finding.Title))
		}
	}

	files := map[string]string{
		baseDir + "/proposal.md": proposalBody.String(),
		baseDir + "/ADR.md": `# ADR Draft

## Context
Generated by ACP proposals compiler from current findings set.

## Decision
Approve the selected remediation scope from proposal.md during architecture review.
`,
		baseDir + "/RFC.md": `# RFC Draft

## Problem
Captured by ACP findings and coverage outputs.

## Proposal
Decompose approved findings into phased implementation tasks with owners and rollout windows.
`,
		baseDir + "/migration-checklist.md": `# Migration Checklist

- [ ] Confirm owners
- [ ] Confirm CI/CD impact
- [ ] Define rollout steps
- [ ] Validate regressions in synthetic scenarios
`,
	}

	var artifacts []Artifact
	for path, content := range files {
		if err := c.workspace.WriteFile(path, []byte(content)); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, Artifact{
			Path:  path,
			Kind:  "proposal",
			Label: filepathLabel(path),
		})
	}
	sortArtifacts(artifacts)
	return artifacts, nil
}

func proposalSlugFromFindings(findings []contracts.Finding) string {
	if len(findings) == 0 {
		return "baseline"
	}
	ids := make([]string, 0, len(findings))
	for _, finding := range findings {
		id := strings.TrimSpace(finding.ID)
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return "baseline"
	}
	sort.Strings(ids)
	hasher := fnv.New64a()
	for _, id := range ids {
		_, _ = hasher.Write([]byte(id))
		_, _ = hasher.Write([]byte{'\n'})
	}
	return fmt.Sprintf("findings-%x", hasher.Sum64())
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

func (c Compiler) WriteArchitectSummary(summary string) ([]Artifact, error) {
	path := "reports/agent-outputs/architect/summary.md"
	if err := c.workspace.WriteFile(path, []byte(summary)); err != nil {
		return nil, err
	}
	return []Artifact{
		{Path: path, Kind: "agent-output", Label: "Architect Summary"},
	}, nil
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

func relatedEdges(entityID string, edges []contracts.Edge) []contracts.Edge {
	var result []contracts.Edge
	for _, edge := range edges {
		if edge.From == entityID || edge.To == entityID {
			result = append(result, edge)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func writeStringList(builder *strings.Builder, title string, values []string) {
	builder.WriteString(fmt.Sprintf("## %s\n\n", title))
	if len(values) == 0 {
		builder.WriteString("None.\n\n")
		return
	}
	for _, value := range values {
		builder.WriteString(fmt.Sprintf("- %s\n", value))
	}
	builder.WriteString("\n")
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

func filepathLabel(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}
