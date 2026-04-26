package fakeruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const (
	domainReportFileName      = "domain-report.md"
	shardPackManifestFileName = "shard-pack-manifest.json"
	validatorVerdictFileName  = "validator-verdict.json"
	finalRunIndexFileName     = "final-run-index.json"
	citationIndexFileName     = "citation-index.json"
	step0WizardContractPath   = "charter/wizard/step0-contract.json"
)

type runtimeDraftManifest = runtimedrafts.Manifest
type runtimeDraftOutput = runtimedrafts.Output

type runtimeStep0WizardContract struct {
	Version       int      `json:"version"`
	ProjectName   string   `json:"project_name"`
	Scope         string   `json:"scope"`
	NFRPriorities []string `json:"nfr_priorities"`
	Rules         []string `json:"rules"`
}

func persistRuntimeArtifacts(task acpruntime.Task, summary string, semantic contracts.SemanticSnapshot, verdict *contracts.ValidatorVerdict) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return nil
	}
	switch task.StepID {
	case "init.step0.constitution":
		return writeConstitutionDraftManifest(writeRoot, task, summary)
	case "init.step1.collect", "refresh.step1.collect":
		return writeShardPackManifest(writeRoot, task, summary, semantic)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return writeAsIsDraftManifest(writeRoot, task, summary, semantic)
	case "init.step3.findings", "refresh.step3.findings":
		return writeValidatorVerdict(writeRoot, task, summary, semantic, verdict)
	case "init.step4.proposals", "refresh.step4.proposals":
		return writeProposalsDraftManifest(writeRoot, task, summary)
	default:
		return nil
	}
}

func writeShardPackManifest(writeRoot string, task acpruntime.Task, summary string, semantic contracts.SemanticSnapshot) error {
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		return fmt.Errorf("create shard write root: %w", err)
	}

	documentID := runtimeDomainDocumentID(task)
	citations := deriveRuntimeCitations(task, semantic, documentID)
	document := contracts.AuthoredDocument{
		ID:            documentID,
		Kind:          "agent-output",
		Title:         runtimeDomainDocumentTitle(task),
		Path:          domainReportFileName,
		CanonicalPath: runtimeDomainCanonicalPath(task),
		Topics:        runtimeDomainTopics(task),
		CitationIDs:   collectCitationIDs(citations),
		Status:        "staged",
	}

	reportContent := renderRuntimeDomainReport(task, summary, semantic, citations)
	if err := writeRuntimeArtifactFile(writeRoot, document.Path, []byte(reportContent)); err != nil {
		return err
	}

	manifest := contracts.ShardPackManifest{
		Version:      1,
		RunID:        strings.TrimSpace(task.RunID),
		StepID:       strings.TrimSpace(task.StepID),
		ShardID:      strings.TrimSpace(task.ShardID),
		DomainID:     strings.TrimSpace(task.DomainID),
		AgentRole:    runtimeAgentRoleForTask(task),
		ArtifactRoot: strings.TrimSpace(task.ArtifactRoot),
		RepoScopes:   append([]string(nil), task.RepoScopes...),
		PathScopes:   append([]string(nil), task.PathScopes...),
		Summary:      strings.TrimSpace(summary),
		Documents:    []contracts.AuthoredDocument{document},
		Citations:    citations,
		Semantic:     semantic,
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal shard pack manifest: %w", err)
	}
	encoded = append(encoded, '\n')
	return writeRuntimeArtifactFile(writeRoot, shardPackManifestFileName, encoded)
}

func writeValidatorVerdict(
	writeRoot string,
	task acpruntime.Task,
	summary string,
	semantic contracts.SemanticSnapshot,
	verdict *contracts.ValidatorVerdict,
) error {
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		return fmt.Errorf("create validator write root: %w", err)
	}
	effective := contracts.ValidatorVerdict{
		Version:      1,
		RunID:        strings.TrimSpace(task.RunID),
		GeneratedAt:  task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
		Verdict:      "PASS",
		Summary:      strings.TrimSpace(summary),
		CheckedPaths: collectValidatorCheckedPaths(task),
		Findings:     append([]contracts.Finding(nil), semantic.Findings...),
		Questions:    append([]contracts.Question(nil), semantic.Questions...),
	}
	if verdict != nil {
		effective = *verdict
		if effective.Version == 0 {
			effective.Version = 1
		}
		if strings.TrimSpace(effective.RunID) == "" {
			effective.RunID = task.RunID
		}
		if strings.TrimSpace(effective.GeneratedAt) == "" {
			effective.GeneratedAt = task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339)
		}
		if strings.TrimSpace(effective.Summary) == "" {
			effective.Summary = strings.TrimSpace(summary)
		}
		if len(effective.CheckedPaths) == 0 {
			effective.CheckedPaths = collectValidatorCheckedPaths(task)
		}
	}
	encoded, err := json.MarshalIndent(effective, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal validator verdict: %w", err)
	}
	encoded = append(encoded, '\n')
	return writeRuntimeArtifactFile(writeRoot, validatorVerdictFileName, encoded)
}

func writeConstitutionDraftManifest(writeRoot string, task acpruntime.Task, summary string) error {
	outputs := []runtimeDraftOutput{
		{Path: "charter-overview.md", CanonicalPath: "charter/overview.md", Kind: "charter", Title: "Constitution"},
		{Path: "baseline-subagents.yaml", CanonicalPath: "skills/subagents.yaml", Kind: "bundle", Title: "Baseline Subagents"},
	}
	if err := writeRuntimeDraftFinals(task, map[string]string{
		"charter-overview.md":     renderConstitutionDraftOverview(task, summary),
		"baseline-subagents.yaml": string(workspace.BaselineSubagentsContent()),
	}); err != nil {
		return err
	}
	return writeRuntimeDraftManifest(writeRoot, runtimedrafts.ConstitutionManifestFile, task, summary, outputs)
}

func renderConstitutionDraftOverview(task acpruntime.Task, summary string) string {
	contract, ok := loadRuntimeStep0WizardContract(task.Workspace)
	if !ok {
		return "# Project Constitution\n\nGenerated baseline charter for ACP MVP.\n"
	}
	body := strings.TrimSpace(fmt.Sprintf(
		"# Project Constitution\n\n- project_name: `%s`\n- scope: `%s`\n\nGenerated from `%s`.\n",
		contract.ProjectName,
		contract.Scope,
		step0WizardContractPath,
	))
	if strings.TrimSpace(summary) != "" {
		body += "\n- summary: " + strings.TrimSpace(summary) + "\n"
	}
	return body + "\n"
}

func loadRuntimeStep0WizardContract(workspaceRoot string) (runtimeStep0WizardContract, bool) {
	if strings.TrimSpace(workspaceRoot) == "" {
		return runtimeStep0WizardContract{}, false
	}
	content, err := os.ReadFile(filepath.Join(filepath.Clean(workspaceRoot), filepath.FromSlash(step0WizardContractPath)))
	if err != nil {
		return runtimeStep0WizardContract{}, false
	}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	var contract runtimeStep0WizardContract
	if err := decoder.Decode(&contract); err != nil {
		return runtimeStep0WizardContract{}, false
	}
	if contract.Version != 1 || strings.TrimSpace(contract.ProjectName) == "" || strings.TrimSpace(contract.Scope) == "" {
		return runtimeStep0WizardContract{}, false
	}
	return contract, true
}

func writeAsIsDraftManifest(writeRoot string, task acpruntime.Task, summary string, semantic contracts.SemanticSnapshot) error {
	outputs := []runtimeDraftOutput{
		{Path: "overview.md", CanonicalPath: "reports/as-is/overview.md", Kind: "report", Title: "System Overview"},
		{Path: "summary.md", CanonicalPath: "reports/coverage/summary.md", Kind: "report", Title: "Coverage Summary"},
		{Path: "architect-summary.md", CanonicalPath: "reports/agent-outputs/architect/summary.md", Kind: "agent-output", Title: "Architect Summary"},
	}
	overview := strings.Builder{}
	overview.WriteString("# As-Is Overview (Runtime Draft)\n\n")
	overview.WriteString("Prepared by the step-scoped runtime synthesizer.\n\n")
	if strings.TrimSpace(summary) != "" {
		overview.WriteString("- Runtime summary: " + strings.TrimSpace(summary) + "\n")
	}
	if len(task.RepoScopes) > 0 {
		overview.WriteString("- Repo scopes: " + strings.Join(task.RepoScopes, ", ") + "\n")
	}

	coverage := strings.Builder{}
	coverage.WriteString("# Coverage Summary (Runtime Draft)\n\n")
	if len(semantic.Coverage.Observed) > 0 {
		coverage.WriteString("## Observed\n\n")
		for _, item := range semantic.Coverage.Observed {
			coverage.WriteString("- " + item + "\n")
		}
		coverage.WriteString("\n")
	}
	if len(semantic.Coverage.Missing) > 0 {
		coverage.WriteString("## Missing\n\n")
		for _, item := range semantic.Coverage.Missing {
			coverage.WriteString("- " + item + "\n")
		}
		coverage.WriteString("\n")
	}

	architect := "# Architect Summary\n\n" + strings.TrimSpace(summary) + "\n"
	if err := writeRuntimeDraftFinals(task, map[string]string{
		"overview.md":          overview.String(),
		"summary.md":           coverage.String(),
		"architect-summary.md": architect,
	}); err != nil {
		return err
	}
	return writeRuntimeDraftManifest(writeRoot, runtimedrafts.AsIsManifestFile, task, summary, outputs)
}

func writeProposalsDraftManifest(writeRoot string, task acpruntime.Task, summary string) error {
	outputs := []runtimeDraftOutput{
		{Path: "proposal.md", CanonicalPath: "proposals/proposal-baseline/proposal.md", Kind: "proposal", Title: "proposal.md"},
		{Path: "ADR.md", CanonicalPath: "proposals/proposal-baseline/ADR.md", Kind: "proposal", Title: "ADR.md"},
		{Path: "RFC.md", CanonicalPath: "proposals/proposal-baseline/RFC.md", Kind: "proposal", Title: "RFC.md"},
		{Path: "migration-checklist.md", CanonicalPath: "proposals/proposal-baseline/migration-checklist.md", Kind: "proposal", Title: "migration-checklist.md"},
	}
	if err := writeRuntimeDraftFinals(task, map[string]string{
		"proposal.md":            "# Improvement Proposal (Runtime Draft)\n\n" + strings.TrimSpace(summary) + "\n",
		"ADR.md":                 "# ADR Draft (Runtime Draft)\n\nPromote validated staged findings into a reviewable remediation slice.\n",
		"RFC.md":                 "# RFC Draft (Runtime Draft)\n\nDocument rollout and validation gates for the proposed remediation.\n",
		"migration-checklist.md": "# Migration Checklist (Runtime Draft)\n\n- [ ] Confirm owners\n- [ ] Confirm rollout plan\n- [ ] Re-run validation gates\n",
	}); err != nil {
		return err
	}
	return writeRuntimeDraftManifest(writeRoot, runtimedrafts.ProposalsManifestFile, task, summary, outputs)
}

func writeRuntimeDraftManifest(writeRoot string, filename string, task acpruntime.Task, summary string, outputs []runtimeDraftOutput) error {
	manifest := runtimeDraftManifest{
		Version:      1,
		RunID:        strings.TrimSpace(task.RunID),
		StepID:       strings.TrimSpace(task.StepID),
		StepContract: strings.TrimSpace(task.StepContract),
		AgentRole:    runtimeAgentRoleForTask(task),
		Summary:      strings.TrimSpace(summary),
		Outputs:      outputs,
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal runtime draft manifest: %w", err)
	}
	encoded = append(encoded, '\n')
	return writeRuntimeArtifactFile(writeRoot, filename, encoded)
}

func writeRuntimeDraftFinals(task acpruntime.Task, files map[string]string) error {
	draftRoot := strings.TrimSpace(task.DraftFinalRoot)
	if draftRoot == "" {
		draftRoot = strings.TrimSpace(task.WriteRoot)
	}
	for relativePath, content := range files {
		if err := writeRuntimeArtifactFile(draftRoot, relativePath, []byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func writeRuntimeArtifactFile(root string, relativePath string, content []byte) error {
	cleanRoot := filepath.Clean(root)
	rel := filepath.Clean(relativePath)
	target := filepath.Join(cleanRoot, rel)
	if !strings.HasPrefix(target, cleanRoot+string(os.PathSeparator)) && target != cleanRoot {
		return fmt.Errorf("artifact path %q escapes write root %q", relativePath, root)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create artifact parent for %q: %w", relativePath, err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		return fmt.Errorf("write artifact %q: %w", relativePath, err)
	}
	return nil
}

func deriveRuntimeCitations(task acpruntime.Task, semantic contracts.SemanticSnapshot, documentID string) []contracts.DocumentCitation {
	sources := []contracts.DocumentCitation{}
	appendEvidence := func(prefix string, objectID string, provenance contracts.Provenance) {
		claimKey := strings.TrimSpace(prefix + "." + objectID)
		if claimKey == "." || claimKey == "" {
			return
		}
		if len(provenance.Evidence) == 0 {
			sources = append(sources, fallbackCitation(task, claimKey, documentID))
			return
		}
		for idx, evidence := range provenance.Evidence {
			repo := strings.TrimSpace(evidence.Repo)
			if repo == "" {
				repo = primaryTaskRepoScope(task.RepoScope, task.RepoScopes)
			}
			path := strings.TrimSpace(evidence.Path)
			if path == "" {
				path = fallbackEvidencePath(task)
			}
			claimID := "claim." + slugutil.Slugify(fmt.Sprintf("%s.%d", claimKey, idx+1))
			citationID := "cite." + slugutil.Slugify(fmt.Sprintf("%s.%d", claimKey, idx+1))
			sources = append(sources, contracts.DocumentCitation{
				ID:          citationID,
				Repo:        repo,
				Ref:         strings.TrimSpace(evidence.Ref),
				Path:        path,
				Lines:       copyLineRange(evidence.Lines),
				ExcerptHash: strings.TrimSpace(evidence.ExcerptHash),
				Excerpt:     strings.TrimSpace(evidence.Excerpt),
				ClaimIDs:    []string{claimID},
				DocumentIDs: []string{documentID},
			})
		}
	}

	for _, entity := range semantic.Entities {
		appendEvidence("entity", entity.ID, entity.Provenance)
	}
	for _, edge := range semantic.Edges {
		appendEvidence("edge", edge.ID, edge.Provenance)
	}
	for _, finding := range semantic.Findings {
		appendEvidence("finding", finding.ID, finding.Provenance)
	}
	if len(sources) == 0 {
		sources = append(sources, fallbackCitation(task, "runtime.summary", documentID))
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].ID < sources[j].ID })
	return sources
}

func fallbackCitation(task acpruntime.Task, claimKey string, documentID string) contracts.DocumentCitation {
	repo := primaryTaskRepoScope(task.RepoScope, task.RepoScopes)
	if repo == "" {
		repo = "workspace"
	}
	claimID := "claim." + slugutil.Slugify(claimKey)
	citationID := "cite." + slugutil.Slugify(claimKey)
	return contracts.DocumentCitation{
		ID:          citationID,
		Repo:        repo,
		Path:        fallbackEvidencePath(task),
		ClaimIDs:    []string{claimID},
		DocumentIDs: []string{documentID},
	}
}

func fallbackEvidencePath(task acpruntime.Task) string {
	for _, candidate := range task.PathScopes {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return filepath.ToSlash(trimmed)
		}
	}
	return "README.md"
}

func copyLineRange(input *contracts.LineRange) *contracts.LineRange {
	if input == nil {
		return nil
	}
	copy := *input
	return &copy
}

func collectCitationIDs(citations []contracts.DocumentCitation) []string {
	ids := make([]string, 0, len(citations))
	for _, citation := range citations {
		if trimmed := strings.TrimSpace(citation.ID); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	sort.Strings(ids)
	return ids
}

func runtimeDomainDocumentID(task acpruntime.Task) string {
	return "doc.domain." + runtimeDomainSlug(task)
}

func runtimeDomainCanonicalPath(task acpruntime.Task) string {
	return fmt.Sprintf("reports/agent-outputs/domains/%s.md", runtimeDomainSlug(task))
}

func runtimeDomainDocumentTitle(task acpruntime.Task) string {
	if domainID := strings.TrimSpace(task.DomainID); domainID != "" {
		return fmt.Sprintf("Domain dossier: %s", domainID)
	}
	return fmt.Sprintf("Shard dossier: %s", runtimeDomainSlug(task))
}

func runtimeDomainTopics(task acpruntime.Task) []string {
	topics := []string{}
	if domainID := strings.TrimSpace(task.DomainID); domainID != "" {
		topics = append(topics, "domain."+slugutil.Slugify(domainID))
	}
	for _, scope := range task.RepoScopes {
		if trimmed := strings.TrimSpace(scope); trimmed != "" {
			topics = append(topics, "repo."+slugutil.Slugify(trimmed))
		}
	}
	sort.Strings(topics)
	return uniqueSortedStrings(topics)
}

func runtimeDomainSlug(task acpruntime.Task) string {
	candidates := []string{
		strings.TrimSpace(task.DomainID),
		strings.TrimSpace(task.ShardID),
		primaryTaskRepoScope(task.RepoScope, task.RepoScopes),
	}
	for _, candidate := range candidates {
		if slug := slugutil.Slugify(candidate); slug != "" {
			return slug
		}
	}
	return "domain"
}

func runtimeAgentRoleForTask(task acpruntime.Task) string {
	if role := strings.TrimSpace(task.AgentRole); role != "" {
		return role
	}
	if strings.HasSuffix(task.StepID, "step3.findings") {
		return "validator-findings"
	}
	if strings.HasSuffix(task.StepID, "step0.constitution") {
		return "constitution"
	}
	if strings.HasSuffix(task.StepID, "step2.asis_docs") {
		return "architect-aggregator"
	}
	if strings.HasSuffix(task.StepID, "step4.proposals") {
		return "proposals"
	}
	return "shard-analyst"
}

func renderRuntimeDomainReport(
	task acpruntime.Task,
	summary string,
	semantic contracts.SemanticSnapshot,
	citations []contracts.DocumentCitation,
) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("# Runtime Dossier: %s\n\n", runtimeDomainSlug(task)))
	builder.WriteString(fmt.Sprintf("- domain_id: `%s`\n", defaultBacktickValue(task.DomainID, "unknown")))
	builder.WriteString(fmt.Sprintf("- shard_id: `%s`\n", defaultBacktickValue(task.ShardID, "unknown")))
	builder.WriteString(fmt.Sprintf("- agent_role: `%s`\n", defaultBacktickValue(runtimeAgentRoleForTask(task), "runtime")))
	builder.WriteString(fmt.Sprintf("- repo_scopes: %s\n", renderBacktickListOrNone(task.RepoScopes)))
	builder.WriteString(fmt.Sprintf("- summary: %s\n", defaultPlainValue(strings.TrimSpace(summary), "none")))
	builder.WriteString(fmt.Sprintf("- citations: %d\n\n", len(citations)))

	builder.WriteString("## Coverage\n\n")
	builder.WriteString(fmt.Sprintf("- observed: %s\n", renderBacktickListOrNone(semantic.Coverage.Observed)))
	builder.WriteString(fmt.Sprintf("- missing: %s\n", renderBacktickListOrNone(semantic.Coverage.Missing)))
	builder.WriteString(fmt.Sprintf("- notes: %s\n\n", renderPlainListOrNone(semantic.Coverage.Notes)))

	builder.WriteString("## Entities\n\n")
	if len(semantic.Entities) == 0 {
		builder.WriteString("- none\n\n")
	} else {
		for _, entity := range semantic.Entities {
			builder.WriteString(fmt.Sprintf("- `%s` (%s)\n", entity.ID, entity.Type))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("## Findings\n\n")
	if len(semantic.Findings) == 0 {
		builder.WriteString("- none\n\n")
	} else {
		for _, finding := range semantic.Findings {
			builder.WriteString(fmt.Sprintf("- `%s`: %s\n", finding.ID, finding.Title))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("## Questions\n\n")
	if len(semantic.Questions) == 0 {
		builder.WriteString("- none\n\n")
	} else {
		for _, question := range semantic.Questions {
			builder.WriteString(fmt.Sprintf("- `%s`: %s\n", question.ID, question.Text))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("## Citation IDs\n\n")
	for _, citationID := range collectCitationIDs(citations) {
		builder.WriteString(fmt.Sprintf("- `%s`\n", citationID))
	}
	return builder.String()
}

func renderBacktickListOrNone(values []string) string {
	normalized := uniqueSortedStrings(values)
	if len(normalized) == 0 {
		return "`none`"
	}
	quoted := make([]string, 0, len(normalized))
	for _, value := range normalized {
		quoted = append(quoted, "`"+value+"`")
	}
	return strings.Join(quoted, ", ")
}

func renderPlainListOrNone(values []string) string {
	normalized := uniqueSortedStrings(values)
	if len(normalized) == 0 {
		return "none"
	}
	return strings.Join(normalized, ", ")
}

func defaultBacktickValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	return value
}

func defaultPlainValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func primaryTaskRepoScope(explicit string, scopes []string) string {
	if trimmed := strings.TrimSpace(explicit); trimmed != "" {
		return trimmed
	}
	for _, scope := range scopes {
		if trimmed := strings.TrimSpace(scope); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func collectValidatorCheckedPaths(task acpruntime.Task) []string {
	paths := []string{}
	stageRoot := filepath.Join(task.Workspace, "reports", "taskruns", task.RunID, "staging", "final")
	if err := filepath.Walk(stageRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(task.Workspace, path)
		if relErr != nil {
			return nil
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		paths = nil
	}
	if len(paths) == 0 {
		paths = append(paths,
			filepath.ToSlash(filepath.Join("reports", "taskruns", task.RunID, "staging", "final", finalRunIndexFileName)),
			filepath.ToSlash(filepath.Join("reports", "taskruns", task.RunID, "staging", "final", citationIndexFileName)),
		)
	}
	sort.Strings(paths)
	return uniqueSortedStrings(paths)
}
