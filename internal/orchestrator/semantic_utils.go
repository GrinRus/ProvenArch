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

func shouldFilterRefreshCollectEntityType(entityType string) bool {
	switch normalizeSemanticKey(entityType) {
	case "runtime provider", "runtime", "runtime meta", "metadata":
		return true
	default:
		return false
	}
}

func offTopicCollectTerms() []string {
	return []string{
		"bidding",
		"tender",
		"chinabidding",
		"power system",
		"power enterprise",
		"relay protection",
		"load flow",
		"electric analysis",
		"继电",
		"潮流",
		"电力",
		"招标",
	}
}

func isLikelyPowerScope(value string) bool {
	normalized := normalizeSemanticKey(value)
	if normalized == "" {
		return false
	}
	for _, hint := range []string{"power", "energy", "electric", "grid", "utility"} {
		if strings.Contains(normalized, hint) {
			return true
		}
	}
	return false
}

func shouldApplyOffTopicGuard(task acpruntime.Task) bool {
	if isLikelyPowerScope(task.Workspace) {
		return false
	}
	for _, scope := range task.RepoScopes {
		if isLikelyPowerScope(scope) {
			return false
		}
	}
	return true
}

func detectOffTopicTerms(text string) []string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return nil
	}
	hits := []string{}
	for _, term := range offTopicCollectTerms() {
		if strings.Contains(normalized, strings.ToLower(term)) {
			hits = append(hits, term)
		}
	}
	return dedupeSemanticStrings(hits)
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
