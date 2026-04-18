package orchestrator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

func (e *pipelineExecution) repairValidatorScopedArtifacts(verdict *contracts.ValidatorVerdict) error {
	if verdict == nil || e.finalRunIndex == nil || e.citationIndex == nil {
		return nil
	}
	changed := repairDuplicateClaimIDsInCitationIndex(e.finalRunIndex, e.citationIndex)
	if !changed {
		return nil
	}

	citationRaw, err := json.MarshalIndent(e.citationIndex, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal repaired citation index: %w", err)
	}
	citationRaw = append(citationRaw, '\n')
	repairedCitationIndex, err := contracts.ParseCitationIndex(citationRaw)
	if err != nil {
		return fmt.Errorf("parse repaired citation index: %w", err)
	}
	if err := e.workspace.WriteFile(runtimeCitationIndexPath(e.runID), citationRaw); err != nil {
		return fmt.Errorf("write repaired citation index: %w", err)
	}
	e.citationIndex = &repairedCitationIndex

	remainingIssues, resolvedCount := filterResolvedValidatorIssues(verdict.Issues)
	verdict.Issues = remainingIssues
	if resolvedCount > 0 && len(remainingIssues) == 0 && strings.EqualFold(strings.TrimSpace(verdict.Verdict), "FAIL") {
		verdict.Verdict = "PASS"
	}
	verdict.FixedPaths = appendUniqueValidatorPaths(verdict.FixedPaths, runtimeCitationIndexPath(e.runID))
	verdict.Summary = appendValidatorRepairNote(verdict.Summary, "deterministically repaired duplicate claim_ids in citation-index.json")
	verdictRaw, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal repaired validator verdict: %w", err)
	}
	verdictRaw = append(verdictRaw, '\n')
	repairedVerdict, err := contracts.ParseValidatorVerdict(verdictRaw)
	if err != nil {
		return fmt.Errorf("parse repaired validator verdict: %w", err)
	}
	if err := e.workspace.WriteFile(runtimeValidatorVerdictPath(e.runID), verdictRaw); err != nil {
		return fmt.Errorf("write repaired validator verdict: %w", err)
	}
	*verdict = repairedVerdict
	return nil
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
