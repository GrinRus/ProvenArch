package orchestrator

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func countCoverageObserved(coverage *contracts.Coverage) int {
	if coverage == nil {
		return 0
	}
	return len(coverage.Observed)
}

func countCoverageMissing(coverage *contracts.Coverage) int {
	if coverage == nil {
		return 0
	}
	return len(coverage.Missing)
}

func mergeQuestions(existing []contracts.Question, incoming []contracts.Question) []contracts.Question {
	byID := map[string]contracts.Question{}
	byText := map[string]string{}
	order := make([]string, 0, len(existing)+len(incoming))

	appendQuestion := func(question contracts.Question) {
		id := canonicalizeQuestionID(question.ID)
		if id == "" {
			return
		}
		question.ID = id
		textKey := normalizeQuestionText(question.Text)
		if textKey != "" {
			if existingID, exists := byText[textKey]; exists {
				if existingID != id {
					return
				}
			}
		}
		if _, exists := byID[id]; exists {
			return
		}
		byID[id] = question
		order = append(order, id)
		if textKey != "" {
			byText[textKey] = id
		}
	}
	for _, question := range existing {
		appendQuestion(question)
	}
	for _, question := range incoming {
		appendQuestion(question)
	}

	merged := make([]contracts.Question, 0, len(order))
	for _, id := range order {
		merged = append(merged, byID[id])
	}
	return merged
}

func mergeCoverage(existing *contracts.Coverage, incoming *contracts.Coverage) *contracts.Coverage {
	if existing == nil && incoming == nil {
		return nil
	}
	if existing == nil {
		copyCoverage := *incoming
		copyCoverage.Observed = dedupeSemanticStrings(copyCoverage.Observed)
		copyCoverage.Missing = dedupeSemanticStrings(canonicalizeCoverageMissing(copyCoverage.Missing))
		copyCoverage.Notes = dedupeSemanticStrings(copyCoverage.Notes)
		return &copyCoverage
	}
	if incoming == nil {
		copyCoverage := *existing
		copyCoverage.Observed = dedupeSemanticStrings(copyCoverage.Observed)
		copyCoverage.Missing = dedupeSemanticStrings(canonicalizeCoverageMissing(copyCoverage.Missing))
		copyCoverage.Notes = dedupeSemanticStrings(copyCoverage.Notes)
		return &copyCoverage
	}

	merged := &contracts.Coverage{
		Observed: dedupeSemanticStrings(append(existing.Observed, incoming.Observed...)),
		Missing:  dedupeSemanticStrings(canonicalizeCoverageMissing(append(existing.Missing, incoming.Missing...))),
		Notes:    dedupeSemanticStrings(append(existing.Notes, incoming.Notes...)),
	}
	return merged
}

func canonicalizeCoverageMissing(values []string) []string {
	canonical := make([]string, 0, len(values))
	for _, value := range values {
		canonical = append(canonical, canonicalizeCoverageMissingValue(value))
	}
	return canonical
}

func canonicalizeCoverageMissingValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ToLower(trimmed)
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")

	switch normalized {
	case "owner mappings", "owner mapping", "owner team mapping", "owner team mappings":
		return "owner mappings"
	case "ci cd evidence", "ci cd pipelines", "ci cd pipeline evidence", "cicd evidence", "ci pipelines":
		return "ci-cd evidence"
	case "delta validation":
		return "delta validation"
	case "dependency graph":
		return "dependency graph"
	case "runtime metrics":
		return "runtime metrics"
	case "api contracts", "api contract", "api contracts drift", "api specification drift", "api specs":
		return "api contracts"
	case "deployment configs", "deployment config", "deployment configuration", "deployment configurations":
		return "deployment configs"
	case "integration edges", "service integrations":
		return "integration edges"
	case "datastore bindings", "database bindings":
		return "datastore bindings"
	case "dependencies", "dependency drift":
		return "dependencies"
	default:
		return trimmed
	}
}

func normalizeSemanticKey(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

func dedupeSemanticStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := normalizeSemanticKey(trimmed)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func canonicalizeQuestionID(id string) string {
	canonical := strings.TrimSpace(id)
	for {
		dot := strings.LastIndex(canonical, ".")
		if dot <= 0 {
			break
		}
		suffix := canonical[dot+1:]
		if _, err := strconv.Atoi(suffix); err != nil {
			break
		}
		canonical = canonical[:dot]
	}
	return strings.TrimSpace(canonical)
}

func normalizeQuestionText(text string) string {
	return normalizeSemanticKey(text)
}

func isOwnerMappingsMissing(coverage *contracts.Coverage) bool {
	if coverage == nil {
		return false
	}
	for _, missing := range coverage.Missing {
		if canonicalizeCoverageMissingValue(missing) == "owner mappings" {
			return true
		}
	}
	return false
}

func guardRefreshCollectSemantic(stepID string, task acpruntime.Task, semantic contracts.SemanticSnapshot) (contracts.SemanticSnapshot, []string) {
	if strings.TrimSpace(stepID) != "refresh.step1.collect" {
		return semantic, nil
	}

	allowedRepos := semanticAllowedRepoKeys(task.RepoScopes)
	filteredEntityKinds := map[string]string{}
	diagnostics := []contracts.Finding{}
	warnings := []string{}

	guardedEntities := make([]contracts.Entity, 0, len(semantic.Entities))
	for _, entity := range semantic.Entities {
		if kind, reason, filtered := refreshEntityGuardReason(entity, allowedRepos); filtered {
			entityID := strings.TrimSpace(entity.ID)
			filteredEntityKinds[entityID] = kind
			diagnostics = append(diagnostics, semanticGuardDiagnosticFinding(kind, "entity", entityID, reason))
			warnings = append(warnings, semanticGuardWarning(kind, "entity", entityID))
			continue
		}
		guardedEntities = append(guardedEntities, entity)
	}
	semantic.Entities = guardedEntities

	guardedEdges := make([]contracts.Edge, 0, len(semantic.Edges))
	for _, edge := range semantic.Edges {
		if _, filtered := filteredEntityKinds[strings.TrimSpace(edge.From)]; filtered {
			continue
		}
		if _, filtered := filteredEntityKinds[strings.TrimSpace(edge.To)]; filtered {
			continue
		}
		if kind, reason, filtered := refreshEdgeGuardReason(edge, allowedRepos); filtered {
			edgeID := strings.TrimSpace(edge.ID)
			diagnostics = append(diagnostics, semanticGuardDiagnosticFinding(kind, "edge", edgeID, reason))
			warnings = append(warnings, semanticGuardWarning(kind, "edge", edgeID))
			continue
		}
		guardedEdges = append(guardedEdges, edge)
	}
	semantic.Edges = guardedEdges

	guardedFindings := make([]contracts.Finding, 0, len(semantic.Findings)+len(diagnostics))
	for _, finding := range semantic.Findings {
		related := filterGuardedRelatedIDs(finding.RelatedIDs, filteredEntityKinds)
		if len(finding.RelatedIDs) > 0 && len(related) == 0 {
			continue
		}
		if kind, reason, filtered := refreshFindingGuardReason(finding, allowedRepos); filtered {
			findingID := strings.TrimSpace(finding.ID)
			diagnostics = append(diagnostics, semanticGuardDiagnosticFinding(kind, "finding", findingID, reason))
			warnings = append(warnings, semanticGuardWarning(kind, "finding", findingID))
			continue
		}
		finding.RelatedIDs = related
		guardedFindings = append(guardedFindings, finding)
	}
	guardedFindings = append(guardedFindings, diagnostics...)
	semantic.Findings = guardedFindings

	guardedQuestions := make([]contracts.Question, 0, len(semantic.Questions))
	for _, question := range semantic.Questions {
		related := filterGuardedRelatedIDs(question.RelatedIDs, filteredEntityKinds)
		if len(question.RelatedIDs) > 0 && len(related) == 0 {
			continue
		}
		question.RelatedIDs = related
		guardedQuestions = append(guardedQuestions, question)
	}
	semantic.Questions = guardedQuestions

	return semantic, dedupeSemanticStrings(warnings)
}

func semanticAllowedRepoKeys(repoScopes []string) map[string]struct{} {
	allowed := map[string]struct{}{}
	for _, scope := range repoScopes {
		addSemanticRepoKey(allowed, scope)
	}
	return allowed
}

func addSemanticRepoKey(keys map[string]struct{}, repo string) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return
	}
	if normalized := normalizeSemanticKey(repo); normalized != "" {
		keys[normalized] = struct{}{}
	}
	if slug := slugutil.Slugify(repo); slug != "" {
		keys[slug] = struct{}{}
	}
}

func refreshEntityGuardReason(entity contracts.Entity, allowedRepos map[string]struct{}) (string, string, bool) {
	entityID := strings.TrimSpace(entity.ID)
	if semanticEntityLooksRuntimeMetadata(entity) {
		return "runtime_metadata", "runtime/provider/process metadata is not a model entity", true
	}
	if semanticEvidenceOffScope(entity.Provenance.Evidence, allowedRepos) {
		return "off_scope", "semantic entity evidence does not match the assigned refresh repo scopes", true
	}
	if entityID == "" {
		return "low_signal", "semantic entity has no stable id", true
	}
	return "", "", false
}

func refreshEdgeGuardReason(edge contracts.Edge, allowedRepos map[string]struct{}) (string, string, bool) {
	if semanticEvidenceHasRuntimeArtifact(edge.Provenance.Evidence) {
		return "runtime_metadata", "runtime/provider/process metadata is not a model edge", true
	}
	if semanticEvidenceOffScope(edge.Provenance.Evidence, allowedRepos) {
		return "off_scope", "semantic edge evidence does not match the assigned refresh repo scopes", true
	}
	return "", "", false
}

func refreshFindingGuardReason(finding contracts.Finding, allowedRepos map[string]struct{}) (string, string, bool) {
	if semanticEvidenceHasRuntimeArtifact(finding.Provenance.Evidence) {
		return "runtime_metadata", "runtime/provider/process metadata is not an architecture finding", true
	}
	if semanticEvidenceOffScope(finding.Provenance.Evidence, allowedRepos) {
		return "off_scope", "finding evidence does not match the assigned refresh repo scopes", true
	}
	return "", "", false
}

func semanticEntityLooksRuntimeMetadata(entity contracts.Entity) bool {
	switch normalizeSemanticKey(entity.Type) {
	case "runtime", "runtime provider", "runtime meta", "runtime metadata", "metadata", "taskrun", "task run", "shard metadata", "provider metadata":
		return true
	}
	if semanticEvidenceHasRuntimeArtifact(entity.Provenance.Evidence) {
		return true
	}
	text := normalizeSemanticKey(entitySemanticText(&entity))
	for _, marker := range runtimeMetadataSemanticMarkers() {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func runtimeMetadataSemanticMarkers() []string {
	return []string{
		"runtime provider",
		"runtime execution",
		"runtime metadata",
		"provider lifecycle",
		"taskrun",
		"task run",
		"shard metadata",
		"artifact root",
		"write root",
		"draft final root",
		"read context root",
		"raw output",
		"stdout",
		"stderr",
		"final run index",
		"citation index",
		"validator verdict",
		"acp runtime",
	}
}

func semanticEvidenceHasRuntimeArtifact(evidence []contracts.Evidence) bool {
	for _, item := range evidence {
		if semanticPathLooksRuntimeArtifact(item.Path) {
			return true
		}
	}
	return false
}

func semanticPathLooksRuntimeArtifact(path string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(path, "\\", "/")))
	if normalized == "" {
		return false
	}
	if strings.HasPrefix(normalized, "/") {
		return true
	}
	for _, marker := range []string{
		"reports/taskruns/",
		"runtime-execution.json",
		"shard-pack-manifest.json",
		"final-run-index.json",
		"citation-index.json",
		"validator-verdict.json",
		"raw/stdout",
		"raw/stderr",
		"raw/meta",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func semanticEvidenceOffScope(evidence []contracts.Evidence, allowedRepos map[string]struct{}) bool {
	if len(allowedRepos) == 0 {
		return false
	}
	hasRepoEvidence := false
	for _, item := range evidence {
		repo := strings.TrimSpace(item.Repo)
		if repo == "" {
			continue
		}
		hasRepoEvidence = true
		if semanticRepoAllowed(repo, allowedRepos) {
			return false
		}
	}
	return hasRepoEvidence
}

func semanticRepoAllowed(repo string, allowedRepos map[string]struct{}) bool {
	for _, key := range []string{normalizeSemanticKey(repo), slugutil.Slugify(repo)} {
		if key == "" {
			continue
		}
		if _, ok := allowedRepos[key]; ok {
			return true
		}
	}
	return false
}

func filterGuardedRelatedIDs(relatedIDs []string, filteredEntityKinds map[string]string) []string {
	filtered := make([]string, 0, len(relatedIDs))
	seen := map[string]struct{}{}
	for _, relatedID := range relatedIDs {
		trimmed := strings.TrimSpace(relatedID)
		if trimmed == "" {
			continue
		}
		if _, dropped := filteredEntityKinds[trimmed]; dropped {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		filtered = append(filtered, trimmed)
	}
	return filtered
}

func semanticGuardDiagnosticFinding(kind string, candidateType string, candidateID string, reason string) contracts.Finding {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "filtered"
	}
	candidateType = strings.TrimSpace(candidateType)
	if candidateType == "" {
		candidateType = "candidate"
	}
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		candidateID = "unknown"
	}
	slug := slugutil.Slugify(candidateType + "-" + candidateID)
	if slug == "" {
		slug = "unknown"
	}
	titleKind := strings.ReplaceAll(kind, "_", " ")
	return contracts.Finding{
		ID:          "semantic_guard.refresh." + strings.ReplaceAll(kind, "_", "-") + "." + slug,
		Severity:    "medium",
		Title:       "Refresh semantic guard filtered " + titleKind + " " + candidateType,
		Description: strings.TrimSpace(reason),
		RuleID:      "semantic_guard.refresh." + kind,
		Provenance: contracts.Provenance{
			Kind:       "assertion",
			Confidence: 1,
		},
	}
}

func semanticGuardWarning(kind string, candidateType string, candidateID string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "filtered"
	}
	candidateType = strings.TrimSpace(candidateType)
	if candidateType == "" {
		candidateType = "candidate"
	}
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		candidateID = "unknown"
	}
	return "semantic_guard: " + kind + "_filtered in refresh.step1.collect candidate_type=" + candidateType + " candidate_id=" + candidateID
}

func entitySemanticText(entity *contracts.Entity) string {
	if entity == nil {
		return ""
	}
	parts := []string{
		strings.TrimSpace(entity.ID),
		strings.TrimSpace(entity.Type),
		strings.TrimSpace(entity.Name),
	}
	if len(entity.Aliases) > 0 {
		parts = append(parts, strings.Join(entity.Aliases, " "))
	}
	if entity.Attributes != nil {
		if raw, err := json.Marshal(entity.Attributes); err == nil {
			parts = append(parts, string(raw))
		}
	}
	return strings.Join(parts, " ")
}

func repoScopeForDomain(domainID string, repos []workspace.RepoSource) string {
	domainSlug := slugutil.Slugify(domainID)
	for _, repo := range repos {
		repoSlug := slugutil.Slugify(repo.Name)
		if repoSlug == domainSlug {
			return repo.Name
		}
	}
	return ""
}
