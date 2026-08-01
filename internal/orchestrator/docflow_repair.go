package orchestrator

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

type validatorRepairStageResult struct {
	Changed        bool
	ResolvedIssues int
}

func (e *pipelineExecution) applyValidatorRepairStage(stepID string, domainID string, taskID string, verdict *contracts.ValidatorVerdict) (validatorRepairStageResult, error) {
	result, err := e.repairValidatorScopedArtifacts(verdict)
	if err != nil {
		e.logError(stepID, domainID, "validator repair stage failed", map[string]any{
			"task_id": taskID,
			"error":   strings.TrimSpace(err.Error()),
		})
		return validatorRepairStageResult{}, err
	}
	if !result.Changed {
		e.logInfo(stepID, domainID, "validator repair stage skipped", map[string]any{
			"task_id": taskID,
		})
		return result, nil
	}
	e.logInfo(stepID, domainID, "validator repair stage applied", map[string]any{
		"task_id":         taskID,
		"resolved_issues": result.ResolvedIssues,
	})
	return result, nil
}

func (e *pipelineExecution) repairValidatorScopedArtifacts(verdict *contracts.ValidatorVerdict) (validatorRepairStageResult, error) {
	if verdict == nil || e.finalRunIndex == nil || e.citationIndex == nil {
		return validatorRepairStageResult{}, nil
	}
	citationCandidate, err := cloneCitationIndex(*e.citationIndex)
	if err != nil {
		return validatorRepairStageResult{}, fmt.Errorf("clone citation index for repair: %w", err)
	}
	changed := repairDuplicateClaimIDsInCitationIndex(e.finalRunIndex, &citationCandidate)
	if !changed {
		return validatorRepairStageResult{}, nil
	}

	verdictCandidate, err := cloneValidatorVerdict(*verdict)
	if err != nil {
		return validatorRepairStageResult{}, fmt.Errorf("clone validator verdict for repair: %w", err)
	}

	citationRaw, err := json.MarshalIndent(&citationCandidate, "", "  ")
	if err != nil {
		return validatorRepairStageResult{}, fmt.Errorf("marshal repaired citation index: %w", err)
	}
	citationRaw = append(citationRaw, '\n')
	repairedCitationIndex, err := contracts.ParseCitationIndex(citationRaw)
	if err != nil {
		return validatorRepairStageResult{}, fmt.Errorf("parse repaired citation index: %w", err)
	}
	if err := e.workspace.WriteFile(runtimeCitationIndexPath(e.runID), citationRaw); err != nil {
		return validatorRepairStageResult{}, fmt.Errorf("write repaired citation index: %w", err)
	}

	remainingIssues, resolvedCount := filterResolvedValidatorIssues(verdictCandidate.Issues)
	verdictCandidate.Issues = remainingIssues
	if resolvedCount > 0 && len(remainingIssues) == 0 && strings.EqualFold(strings.TrimSpace(verdict.Verdict), "FAIL") {
		verdictCandidate.Verdict = "PASS"
	}
	verdictCandidate.FixedPaths = appendUniqueValidatorPaths(verdictCandidate.FixedPaths, runtimeCitationIndexPath(e.runID))
	verdictCandidate.Summary = appendValidatorRepairNote(verdictCandidate.Summary, "deterministically repaired duplicate claim_ids in citation-index.json")
	verdictRaw, err := json.MarshalIndent(&verdictCandidate, "", "  ")
	if err != nil {
		return validatorRepairStageResult{}, fmt.Errorf("marshal repaired validator verdict: %w", err)
	}
	verdictRaw = append(verdictRaw, '\n')
	repairedVerdict, err := contracts.ParseValidatorVerdict(verdictRaw)
	if err != nil {
		return validatorRepairStageResult{}, fmt.Errorf("parse repaired validator verdict: %w", err)
	}
	if err := e.workspace.WriteFile(runtimeValidatorVerdictPath(e.runID), verdictRaw); err != nil {
		return validatorRepairStageResult{}, fmt.Errorf("write repaired validator verdict: %w", err)
	}
	e.citationIndex = &repairedCitationIndex
	*verdict = repairedVerdict
	return validatorRepairStageResult{
		Changed:        true,
		ResolvedIssues: resolvedCount,
	}, nil
}

func cloneCitationIndex(index contracts.CitationIndex) (contracts.CitationIndex, error) {
	raw, err := json.Marshal(index)
	if err != nil {
		return contracts.CitationIndex{}, err
	}
	return contracts.ParseCitationIndex(raw)
}

func cloneValidatorVerdict(verdict contracts.ValidatorVerdict) (contracts.ValidatorVerdict, error) {
	raw, err := json.Marshal(verdict)
	if err != nil {
		return contracts.ValidatorVerdict{}, err
	}
	return contracts.ParseValidatorVerdict(raw)
}

func repairDuplicateClaimIDsInCitationIndex(finalRunIndex *contracts.FinalRunIndex, citationIndex *contracts.CitationIndex) bool {
	if finalRunIndex == nil || citationIndex == nil {
		return false
	}
	shardHints := citationShardHints(*finalRunIndex)
	seen := map[string]struct{}{}
	changed := false

	for idx := range citationIndex.Citations {
		citation := &citationIndex.Citations[idx]
		localSeen := map[string]struct{}{}
		for claimIdx, rawClaimID := range citation.ClaimIDs {
			claimID := strings.TrimSpace(rawClaimID)
			if claimID == "" {
				continue
			}
			if _, exists := seen[claimID]; !exists {
				if _, localExists := localSeen[claimID]; !localExists {
					seen[claimID] = struct{}{}
					localSeen[claimID] = struct{}{}
					continue
				}
			}

			hint := primaryCitationShardHint(shardHints[citation.ID], citation.Repo)
			repaired := nextUniqueClaimID(claimID, hint, seen, localSeen)
			if repaired != claimID {
				citation.ClaimIDs[claimIdx] = repaired
				changed = true
			}
			seen[repaired] = struct{}{}
			localSeen[repaired] = struct{}{}
		}
	}

	return changed
}

func citationShardHints(finalRunIndex contracts.FinalRunIndex) map[string][]string {
	byCitationID := map[string]map[string]struct{}{}
	for _, document := range finalRunIndex.CanonicalDocuments {
		shards := []string{}
		for _, shard := range document.SourceShards {
			if normalized := normalizeClaimIDShardSuffix(shard); normalized != "" {
				shards = append(shards, normalized)
			}
		}
		if len(shards) == 0 {
			continue
		}
		for _, citationID := range document.CitationIDs {
			citationID = strings.TrimSpace(citationID)
			if citationID == "" {
				continue
			}
			if _, ok := byCitationID[citationID]; !ok {
				byCitationID[citationID] = map[string]struct{}{}
			}
			for _, shard := range shards {
				byCitationID[citationID][shard] = struct{}{}
			}
		}
	}

	hints := map[string][]string{}
	for citationID, shardSet := range byCitationID {
		shards := make([]string, 0, len(shardSet))
		for shard := range shardSet {
			shards = append(shards, shard)
		}
		sort.Strings(shards)
		hints[citationID] = shards
	}
	return hints
}

func primaryCitationShardHint(shards []string, repo string) string {
	for _, shard := range shards {
		if shard = strings.TrimSpace(shard); shard != "" {
			return shard
		}
	}
	if repoHint := normalizeClaimIDShardSuffix(repo); repoHint != "" {
		return repoHint
	}
	return "staged"
}

func normalizeClaimIDShardSuffix(value string) string {
	normalized := slugutil.Slugify(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	return normalized
}

func nextUniqueClaimID(base string, shardHint string, seen map[string]struct{}, localSeen map[string]struct{}) string {
	base = strings.TrimSpace(base)
	shardHint = normalizeClaimIDShardSuffix(shardHint)
	if shardHint == "" {
		shardHint = "staged"
	}
	candidate := base + "." + shardHint
	if _, exists := seen[candidate]; !exists {
		if _, localExists := localSeen[candidate]; !localExists {
			return candidate
		}
	}
	for idx := 2; ; idx++ {
		candidate = fmt.Sprintf("%s.%s.%d", base, shardHint, idx)
		if _, exists := seen[candidate]; exists {
			continue
		}
		if _, localExists := localSeen[candidate]; localExists {
			continue
		}
		return candidate
	}
}

func appendUniqueValidatorPaths(existing []string, additions ...string) []string {
	seen := map[string]struct{}{}
	merged := make([]string, 0, len(existing)+len(additions))
	for _, value := range append(append([]string(nil), existing...), additions...) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		merged = append(merged, trimmed)
	}
	sort.Strings(merged)
	return merged
}

func appendValidatorRepairNote(summary string, note string) string {
	summary = strings.TrimSpace(summary)
	note = strings.TrimSpace(note)
	switch {
	case note == "":
		return summary
	case summary == "":
		return note
	case strings.Contains(summary, note):
		return summary
	default:
		return summary + "; " + note
	}
}

func filterResolvedValidatorIssues(issues []contracts.ValidatorIssue) ([]contracts.ValidatorIssue, int) {
	if len(issues) == 0 {
		return issues, 0
	}
	filtered := make([]contracts.ValidatorIssue, 0, len(issues))
	resolved := 0
	for _, issue := range issues {
		if strings.EqualFold(strings.TrimSpace(issue.Code), "duplicate_claim_id") {
			resolved++
			continue
		}
		filtered = append(filtered, issue)
	}
	return filtered, resolved
}

func (e *pipelineExecution) reconcileOwnerGapOnlyVerdict(verdict *contracts.ValidatorVerdict) (bool, error) {
	if verdict == nil || !strings.EqualFold(strings.TrimSpace(verdict.Verdict), "FAIL") {
		return false, nil
	}
	if !ownerGapOnlyResidual(*verdict, e.coverage) {
		return false, nil
	}

	verdictCandidate, err := cloneValidatorVerdict(*verdict)
	if err != nil {
		return false, fmt.Errorf("clone validator verdict for owner-gap reconciliation: %w", err)
	}
	verdictCandidate.Verdict = "PASS"
	verdictCandidate.Summary = appendValidatorRepairNote(
		verdictCandidate.Summary,
		"owner-gap remains visible in findings/questions but is non-blocking without technical validator issues",
	)

	verdictRaw, err := json.MarshalIndent(&verdictCandidate, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal owner-gap-reconciled validator verdict: %w", err)
	}
	verdictRaw = append(verdictRaw, '\n')
	repairedVerdict, err := contracts.ParseValidatorVerdict(verdictRaw)
	if err != nil {
		return false, fmt.Errorf("parse owner-gap-reconciled validator verdict: %w", err)
	}
	if err := e.workspace.WriteFile(runtimeValidatorVerdictPath(e.runID), verdictRaw); err != nil {
		return false, fmt.Errorf("write owner-gap-reconciled validator verdict: %w", err)
	}
	*verdict = repairedVerdict
	return true, nil
}

func (e *pipelineExecution) reconcileEvidenceAdvisoryOnlyVerdict(verdict *contracts.ValidatorVerdict) (bool, error) {
	if verdict == nil || e.citationIndex == nil || !strings.EqualFold(strings.TrimSpace(verdict.Verdict), "FAIL") {
		return false, nil
	}

	issues, ok := evidenceAdvisoryIssues(verdict.Issues, e.citationIndex.Citations)
	if !ok {
		return false, nil
	}
	if len(issues) == 0 && len(verdict.Findings) == 0 && len(verdict.Questions) == 0 {
		return false, nil
	}

	verdictCandidate, err := cloneValidatorVerdict(*verdict)
	if err != nil {
		return false, fmt.Errorf("clone evidence-advisory validator verdict: %w", err)
	}
	verdictCandidate.Verdict = "PASS"
	verdictCandidate.Issues = issues
	verdictCandidate.Summary = appendValidatorRepairNote(
		verdictCandidate.Summary,
		"source-evidence observations remain advisory after deterministic staged artifact validation",
	)

	verdictRaw, err := json.MarshalIndent(&verdictCandidate, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal evidence-advisory validator verdict: %w", err)
	}
	verdictRaw = append(verdictRaw, '\n')
	repairedVerdict, err := contracts.ParseValidatorVerdict(verdictRaw)
	if err != nil {
		return false, fmt.Errorf("parse evidence-advisory validator verdict: %w", err)
	}
	if err := e.workspace.WriteFile(runtimeValidatorVerdictPath(e.runID), verdictRaw); err != nil {
		return false, fmt.Errorf("write evidence-advisory validator verdict: %w", err)
	}
	*verdict = repairedVerdict
	return true, nil
}

func evidenceAdvisoryIssues(issues []contracts.ValidatorIssue, citations []contracts.DocumentCitation) ([]contracts.ValidatorIssue, bool) {
	citationsByID := make(map[string]contracts.DocumentCitation, len(citations))
	for _, citation := range citations {
		citationID := strings.TrimSpace(citation.ID)
		if citationID != "" {
			citationsByID[citationID] = citation
		}
	}

	reconciled := append([]contracts.ValidatorIssue(nil), issues...)
	for idx := range reconciled {
		issue := &reconciled[idx]
		switch strings.ToLower(strings.TrimSpace(issue.Severity)) {
		case "warning":
			continue
		case "error":
		default:
			return nil, false
		}
		if strings.TrimSpace(issue.DocumentID) != "" {
			return nil, false
		}

		issuePath := normalizedEvidencePath(issue.Path)
		if issuePath == "" {
			return nil, false
		}
		citationID := strings.TrimSpace(issue.CitationID)
		citation, exists := citationsByID[citationID]
		if citationID == "" || !exists || normalizedEvidencePath(citation.Path) != issuePath {
			return nil, false
		}
		issue.Severity = "warning"
	}
	return reconciled, true
}

func normalizedEvidencePath(value string) string {
	clean := path.Clean(strings.TrimSpace(strings.ReplaceAll(value, "\\", "/")))
	if clean == "." || clean == "" || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return ""
	}
	return strings.TrimPrefix(clean, "./")
}

func ownerGapOnlyResidual(verdict contracts.ValidatorVerdict, coverage *contracts.Coverage) bool {
	if len(verdict.Issues) > 0 || !isOwnerMappingsMissing(coverage) {
		return false
	}
	if len(verdict.Findings) == 0 && len(verdict.Questions) == 0 {
		return false
	}
	for _, finding := range verdict.Findings {
		if !isOwnerGapFinding(finding) {
			return false
		}
	}
	for _, question := range verdict.Questions {
		if !isOwnerGapQuestion(question) {
			return false
		}
	}
	return true
}

func isOwnerGapFinding(finding contracts.Finding) bool {
	if strings.EqualFold(strings.TrimSpace(finding.RuleID), "rule.owner.required") {
		return true
	}
	text := normalizeSemanticKey(strings.Join([]string{finding.Title, finding.Description}, " "))
	return strings.Contains(text, "owner") || strings.Contains(text, "ownership")
}

func isOwnerGapQuestion(question contracts.Question) bool {
	text := normalizeSemanticKey(question.Text)
	return strings.Contains(text, "owner") || strings.Contains(text, "owns") || strings.Contains(text, "ownership")
}
