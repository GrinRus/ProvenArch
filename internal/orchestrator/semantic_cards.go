package orchestrator

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"

	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func extractQuestionIDs(questions []contracts.Question) []string {
	ids := make([]string, 0, len(questions))
	for _, question := range questions {
		if strings.TrimSpace(question.ID) == "" {
			continue
		}
		ids = append(ids, question.ID)
	}
	sort.Strings(ids)
	return uniqueSorted(ids)
}

func extractFindingIDs(findings []contracts.Finding) []string {
	ids := make([]string, 0, len(findings))
	for _, finding := range findings {
		if strings.TrimSpace(finding.ID) == "" {
			continue
		}
		ids = append(ids, finding.ID)
	}
	sort.Strings(ids)
	return uniqueSorted(ids)
}

func loadCanonicalDomainIDs(ws workspace.Root) ([]string, error) {
	domainsDir, err := ws.Resolve("charter/cards/domains")
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(domainsDir); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	domainSet := map[string]struct{}{}
	if err := filepath.WalkDir(domainsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(entry.Name()) != ".md" {
			return nil
		}
		base := strings.TrimSuffix(entry.Name(), ".md")
		base = strings.TrimSpace(base)
		if base == "" {
			return nil
		}
		domainSet[base] = struct{}{}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("scan canonical domain cards: %w", err)
	}

	domains := make([]string, 0, len(domainSet))
	for domainID := range domainSet {
		domains = append(domains, domainID)
	}
	sort.Strings(domains)
	return domains, nil
}

type domainRepoScopeResolution struct {
	DomainFileID           string
	DeclaredDomainID       string
	HasDeclaredDomainID    bool
	DomainIDMismatch       bool
	RepoScope              string
	DeclaredRepoScope      string
	HasDeclaredRepoScope   bool
	DeclaredRepoScopeKnown bool
}

func resolveRepoScopeForDomainCard(ws workspace.Root, domainID string, repos []workspace.RepoSource) (domainRepoScopeResolution, error) {
	cardPath := fmt.Sprintf("charter/cards/domains/%s.md", domainID)
	contentBytes, err := ws.ReadFile(cardPath)
	if err != nil {
		return domainRepoScopeResolution{}, err
	}
	return resolveRepoScopeForDomainCardContent(domainID, normalizeLineEndings(string(contentBytes)), repos), nil
}

func resolveRepoScopeForDomainCardContent(domainID string, content string, repos []workspace.RepoSource) domainRepoScopeResolution {
	resolution := domainRepoScopeResolution{
		DomainFileID: strings.TrimSpace(domainID),
	}
	declaredDomainID := strings.TrimSpace(extractCardField(content, "id"))
	if declaredDomainID != "" {
		resolution.DeclaredDomainID = declaredDomainID
		resolution.HasDeclaredDomainID = true
		if slugutil.Slugify(declaredDomainID) != slugutil.Slugify(domainID) {
			resolution.DomainIDMismatch = true
		}
	}

	declaredRepoScope := strings.TrimSpace(extractCardField(content, "repo_scope"))
	if declaredRepoScope != "" {
		resolution.DeclaredRepoScope = declaredRepoScope
		resolution.HasDeclaredRepoScope = true
		if repoScopeExists(declaredRepoScope, repos) {
			resolution.DeclaredRepoScopeKnown = true
			resolution.RepoScope = declaredRepoScope
			return resolution
		}
	}
	if strings.TrimSpace(resolution.RepoScope) == "" {
		resolution.RepoScope = strings.TrimSpace(repoScopeForDomain(domainID, repos))
	}
	return resolution
}

func repoScopeExists(scope string, repos []workspace.RepoSource) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false
	}
	for _, repo := range repos {
		if strings.TrimSpace(repo.Name) == scope {
			return true
		}
	}
	return false
}

func repoScopeOrUnknown(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "unknown"
	}
	return scope
}

func sanitizeDomainArtifactSlug(domainID string) string {
	return slugutil.Slugify(domainID)
}

type canonicalTeamCard struct {
	Slug   string
	TeamID string
}

func (e *pipelineExecution) enrichCanonicalCards(domainIDs []string, teamCards []canonicalTeamCard) error {
	entities, err := e.store.ListEntities()
	if err != nil {
		return err
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].ID < entities[j].ID })

	if err := e.enrichDomainCards(domainIDs, entities); err != nil {
		return err
	}
	if err := e.enrichTeamCards(teamCards, entities); err != nil {
		return err
	}

	teamIDSet := map[string]struct{}{}
	for _, teamCard := range teamCards {
		teamIDSet[normalizeID(teamCard.TeamID)] = struct{}{}
	}
	missingTeamQuestions := []contracts.Question{}
	for _, entity := range entities {
		ownerTeamID := normalizeID(entity.OwnerTeamID)
		if ownerTeamID == "" {
			continue
		}
		if _, ok := teamIDSet[ownerTeamID]; ok {
			continue
		}
		slug := slugutil.Slugify(ownerTeamID)
		missingTeamQuestions = append(missingTeamQuestions, contracts.Question{
			ID:       fmt.Sprintf("q.team.%s.missing-canonical-card", slug),
			Text:     fmt.Sprintf("Owner team %q for entity %q has no canonical card in charter/cards/teams", ownerTeamID, entity.ID),
			Priority: "high",
		})
	}
	if len(missingTeamQuestions) > 0 {
		e.questions = mergeQuestions(e.questions, missingTeamQuestions)
	}
	return nil
}

func (e *pipelineExecution) enrichDomainCards(domainIDs []string, entities []contracts.Entity) error {
	for _, domainID := range domainIDs {
		cardPath := fmt.Sprintf("charter/cards/domains/%s.md", domainID)
		contentBytes, err := e.workspace.ReadFile(cardPath)
		if err != nil {
			return err
		}
		content := normalizeLineEndings(string(contentBytes))

		scopeResolution := resolveRepoScopeForDomainCardContent(domainID, content, e.workspace.Manifest.Repos)
		repoScope := strings.TrimSpace(scopeResolution.RepoScope)

		relatedEntities := collectDomainEntities(domainID, repoScope, entities)
		relatedEntitySet := map[string]struct{}{}
		for entityID := range relatedEntities {
			relatedEntitySet[entityID] = struct{}{}
		}

		relatedFindings := collectRelatedFindings(e.findings, relatedEntitySet)
		relatedQuestions := collectRelatedQuestions(e.questions, relatedEntitySet)
		evidenceRefs := collectEvidenceRefs(relatedEntities)
		coverageMissing := []string{}
		if e.coverage != nil {
			coverageMissing = append([]string(nil), e.coverage.Missing...)
		}

		derived := renderDomainDerivedSection(domainID, repoScope, sortedKeys(relatedEntitySet), relatedFindings, relatedQuestions, coverageMissing, evidenceRefs)
		updated := mergeDerivedSection(content, derived)
		if err := e.workspace.WriteFile(cardPath, []byte(updated)); err != nil {
			return err
		}
	}
	return nil
}

func (e *pipelineExecution) enrichTeamCards(teamCards []canonicalTeamCard, entities []contracts.Entity) error {
	for _, teamCard := range teamCards {
		cardPath := fmt.Sprintf("charter/cards/teams/%s.md", teamCard.Slug)
		contentBytes, err := e.workspace.ReadFile(cardPath)
		if err != nil {
			return err
		}
		content := normalizeLineEndings(string(contentBytes))

		teamID := normalizeID(teamCard.TeamID)
		if teamID == "" {
			teamID = "team." + slugutil.Slugify(teamCard.Slug)
		}

		relatedServices := collectTeamServices(teamID, entities)
		relatedServiceSet := map[string]struct{}{}
		for serviceID := range relatedServices {
			relatedServiceSet[serviceID] = struct{}{}
		}
		relatedFindings := collectRelatedFindings(e.findings, relatedServiceSet)
		relatedQuestions := collectRelatedQuestions(e.questions, relatedServiceSet)
		evidenceRefs := collectEvidenceRefs(relatedServices)
		coverageMissing := []string{}
		if e.coverage != nil {
			coverageMissing = append([]string(nil), e.coverage.Missing...)
		}

		derived := renderTeamDerivedSection(teamID, sortedKeys(relatedServiceSet), relatedFindings, relatedQuestions, coverageMissing, evidenceRefs)
		updated := mergeDerivedSection(content, derived)
		if err := e.workspace.WriteFile(cardPath, []byte(updated)); err != nil {
			return err
		}
	}
	return nil
}

func loadCanonicalTeamCards(ws workspace.Root) ([]canonicalTeamCard, error) {
	teamsDir, err := ws.Resolve("charter/cards/teams")
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(teamsDir); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	cards := []canonicalTeamCard{}
	if err := filepath.WalkDir(teamsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			return nil
		}
		slug := strings.TrimSpace(strings.TrimSuffix(entry.Name(), ".md"))
		if slug == "" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		teamID := strings.TrimSpace(extractCardField(normalizeLineEndings(string(content)), "id"))
		if teamID == "" {
			teamID = "team." + slugutil.Slugify(slug)
		}
		cards = append(cards, canonicalTeamCard{
			Slug:   slug,
			TeamID: teamID,
		})
		return nil
	}); err != nil {
		return nil, fmt.Errorf("scan canonical team cards: %w", err)
	}

	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Slug < cards[j].Slug
	})
	return cards, nil
}

func collectDomainEntities(domainID string, repoScope string, entities []contracts.Entity) map[string]contracts.Entity {
	domainSlug := slugutil.Slugify(domainID)
	related := map[string]contracts.Entity{}
	for _, entity := range entities {
		if strings.TrimSpace(repoScope) != "" && hasEvidenceRepo(entity, repoScope) {
			related[entity.ID] = entity
			continue
		}
		entitySlug := slugutil.Slugify(entity.ID)
		if strings.Contains(entitySlug, domainSlug) {
			related[entity.ID] = entity
		}
	}
	return related
}

func collectTeamServices(teamID string, entities []contracts.Entity) map[string]contracts.Entity {
	services := map[string]contracts.Entity{}
	for _, entity := range entities {
		if entity.Type != "service" {
			continue
		}
		if normalizeID(entity.OwnerTeamID) == teamID {
			services[entity.ID] = entity
		}
	}
	return services
}

func collectRelatedFindings(findings []contracts.Finding, relatedIDs map[string]struct{}) []string {
	ids := []string{}
	for _, finding := range findings {
		if !hasRelatedIntersection(finding.RelatedIDs, relatedIDs) {
			continue
		}
		if strings.TrimSpace(finding.ID) == "" {
			continue
		}
		ids = append(ids, finding.ID)
	}
	sort.Strings(ids)
	return uniqueSorted(ids)
}

func collectRelatedQuestions(questions []contracts.Question, relatedIDs map[string]struct{}) []string {
	ids := []string{}
	for _, question := range questions {
		if !hasRelatedIntersection(question.RelatedIDs, relatedIDs) {
			continue
		}
		if strings.TrimSpace(question.ID) == "" {
			continue
		}
		ids = append(ids, question.ID)
	}
	sort.Strings(ids)
	return uniqueSorted(ids)
}

func collectEvidenceRefs(entities map[string]contracts.Entity) []string {
	refs := []string{}
	for _, id := range sortedKeys(entities) {
		entity := entities[id]
		for _, evidence := range entity.Provenance.Evidence {
			repo := strings.TrimSpace(evidence.Repo)
			path := strings.TrimSpace(evidence.Path)
			if repo == "" || path == "" {
				continue
			}
			refs = append(refs, fmt.Sprintf("%s:%s", repo, path))
		}
	}
	sort.Strings(refs)
	return uniqueSorted(refs)
}

func renderDomainDerivedSection(domainID string, repoScope string, entityIDs []string, findingIDs []string, questionIDs []string, coverageMissing []string, evidenceRefs []string) string {
	coverageMissing = uniqueSorted(append([]string(nil), coverageMissing...))

	builder := strings.Builder{}
	builder.WriteString("## Derived (ACP Step1)\n\n")
	builder.WriteString(fmt.Sprintf("- domain_id: `%s`\n", domainID))
	builder.WriteString(fmt.Sprintf("- repo_scope: `%s`\n", repoScopeOrUnknown(repoScope)))
	builder.WriteString(fmt.Sprintf("- related_entities: %s\n", renderBacktickList(entityIDs)))
	builder.WriteString(fmt.Sprintf("- related_findings: %s\n", renderBacktickList(findingIDs)))
	builder.WriteString(fmt.Sprintf("- open_questions: %s\n", renderBacktickList(questionIDs)))
	builder.WriteString(fmt.Sprintf("- coverage_missing: %s\n", renderPlainList(coverageMissing)))
	if len(evidenceRefs) == 0 {
		builder.WriteString("- evidence_refs: none\n")
	} else {
		builder.WriteString("- evidence_refs:\n")
		for _, ref := range evidenceRefs {
			builder.WriteString(fmt.Sprintf("  - `%s`\n", ref))
		}
	}
	return strings.TrimRight(builder.String(), "\n")
}

func renderTeamDerivedSection(teamID string, serviceIDs []string, findingIDs []string, questionIDs []string, coverageMissing []string, evidenceRefs []string) string {
	coverageMissing = uniqueSorted(append([]string(nil), coverageMissing...))

	builder := strings.Builder{}
	builder.WriteString("## Derived (ACP Step1)\n\n")
	builder.WriteString(fmt.Sprintf("- team_id: `%s`\n", teamID))
	builder.WriteString(fmt.Sprintf("- related_services: %s\n", renderBacktickList(serviceIDs)))
	builder.WriteString(fmt.Sprintf("- related_findings: %s\n", renderBacktickList(findingIDs)))
	builder.WriteString(fmt.Sprintf("- open_questions: %s\n", renderBacktickList(questionIDs)))
	builder.WriteString(fmt.Sprintf("- coverage_missing: %s\n", renderPlainList(coverageMissing)))
	if len(evidenceRefs) == 0 {
		builder.WriteString("- evidence_refs: none\n")
	} else {
		builder.WriteString("- evidence_refs:\n")
		for _, ref := range evidenceRefs {
			builder.WriteString(fmt.Sprintf("  - `%s`\n", ref))
		}
	}
	return strings.TrimRight(builder.String(), "\n")
}

func mergeDerivedSection(content string, derivedSection string) string {
	const heading = "## Derived (ACP Step1)"
	content = normalizeLineEndings(content)
	trimmed := strings.TrimRight(content, "\n")
	if idx := strings.Index(trimmed, heading); idx >= 0 {
		trimmed = strings.TrimRight(trimmed[:idx], "\n")
	}
	if strings.TrimSpace(trimmed) == "" {
		return derivedSection + "\n"
	}
	return trimmed + "\n\n" + derivedSection + "\n"
}

func hasEvidenceRepo(entity contracts.Entity, repoScope string) bool {
	for _, evidence := range entity.Provenance.Evidence {
		if strings.TrimSpace(evidence.Repo) == strings.TrimSpace(repoScope) {
			return true
		}
	}
	return false
}

func hasRelatedIntersection(related []string, universe map[string]struct{}) bool {
	if len(universe) == 0 {
		return false
	}
	for _, id := range related {
		if _, ok := universe[id]; ok {
			return true
		}
	}
	return false
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func renderBacktickList(values []string) string {
	values = uniqueSorted(append([]string(nil), values...))
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("`%s`", value))
	}
	return strings.Join(parts, ", ")
}

func renderPlainList(values []string) string {
	values = uniqueSorted(append([]string(nil), values...))
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func normalizeLineEndings(input string) string {
	return strings.ReplaceAll(input, "\r\n", "\n")
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func extractCardField(content string, field string) string {
	prefix := "- " + strings.TrimSpace(field) + ":"
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		value = strings.Trim(value, "`")
		value = strings.Trim(value, `"'`)
		return strings.TrimSpace(value)
	}
	return ""
}
