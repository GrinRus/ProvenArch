package refreshplan

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/pathscope"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func BuildImpactPlan(ctx context.Context, current SourceRevisions, baseline *SourceRevisions, resolved []workspace.ResolvedRepo, evidence PriorEvidence, generatedAt time.Time, git GitRunner) ImpactPlan {
	if git == nil {
		git = LocalGitRunner{}
	}
	plan := ImpactPlan{Version: ImpactPlanVersion, RunID: current.RunID, Pipeline: "refresh", GeneratedAt: generatedAt.UTC().Format(time.RFC3339Nano), Enforcement: "advisory", Decision: "full_refresh_required", RepoDeltas: []RepoDelta{}, FallbackReasons: []string{}, AffectedShards: []string{}, AffectedDomains: []string{}, UnmappedPaths: []string{}, StaleArtifactCandidates: []string{}, PreservedArtifactCandidates: []string{}, PlannedActions: []string{"continue_full_refresh"}}
	if baseline == nil {
		plan.FallbackReasons = []string{"baseline_missing"}
		plan.PlannedActions = append(plan.PlannedActions, "conservative_full_refresh")
		normalizeImpactPlan(&plan)
		return plan
	}
	id := baseline.RunID
	plan.BaselineRunID = &id
	if current.AnalysisInputsState != "complete" {
		plan.FallbackReasons = append(plan.FallbackReasons, "analysis_inputs_unreadable")
	}
	if current.AnalysisInputFingerprint != baseline.AnalysisInputFingerprint {
		plan.FallbackReasons = append(plan.FallbackReasons, "analysis_inputs_changed")
	}
	if !evidence.Readable {
		plan.FallbackReasons = append(plan.FallbackReasons, "prior_evidence_unreadable")
	}
	resolvedByName := map[string]workspace.ResolvedRepo{}
	for _, repo := range resolved {
		resolvedByName[repo.Name] = repo
	}
	baselineByName := map[string]RepoRevision{}
	for _, repo := range baseline.Repos {
		baselineByName[repo.Name] = repo
	}
	allInScope := 0
	for _, repo := range current.Repos {
		prior, ok := baselineByName[repo.Name]
		delta := RepoDelta{Repo: repo.Name, BaselineRevision: repo.BaselineRevision, CurrentRevision: repo.CurrentRevision, Comparison: repo.Comparison, ChangesComplete: true, Changes: []PathChange{}}
		for _, reason := range repo.FallbackReasons {
			switch reason {
			case "dirty_worktree", "current_revision_unavailable", "baseline_revision_unavailable", "history_rewritten":
				plan.FallbackReasons = append(plan.FallbackReasons, reason)
			}
		}
		if !ok || repo.CurrentRevision == nil || prior.CurrentRevision == nil {
			delta.ChangesComplete = false
			plan.RepoDeltas = append(plan.RepoDeltas, delta)
			continue
		}
		if *repo.CurrentRevision == *prior.CurrentRevision {
			plan.RepoDeltas = append(plan.RepoDeltas, delta)
			continue
		}
		resolvedRepo, found := resolvedByName[repo.Name]
		if !found {
			delta.ChangesComplete = false
			plan.FallbackReasons = append(plan.FallbackReasons, "git_diff_unavailable")
			plan.RepoDeltas = append(plan.RepoDeltas, delta)
			continue
		}
		raw, err := git.Run(ctx, resolvedRepo.Path, "diff", "--name-status", "-z", "-M", "-C", *prior.CurrentRevision+".."+*repo.CurrentRevision)
		if err != nil {
			delta.ChangesComplete = false
			plan.FallbackReasons = append(plan.FallbackReasons, "git_diff_unavailable")
			plan.RepoDeltas = append(plan.RepoDeltas, delta)
			continue
		}
		changes, err := ParseNameStatusZ(raw)
		if err != nil {
			delta.ChangesComplete = false
			plan.FallbackReasons = append(plan.FallbackReasons, "git_diff_unavailable")
			plan.RepoDeltas = append(plan.RepoDeltas, delta)
			continue
		}
		delta.ChangedPathCount = len(changes)
		if len(changes) > MaxChangedPaths {
			delta.ChangesComplete = false
			delta.Changes = []PathChange{}
			plan.FallbackReasons = append(plan.FallbackReasons, "change_limit_exceeded")
			plan.RepoDeltas = append(plan.RepoDeltas, delta)
			continue
		}
		for i := range changes {
			changes[i].InScope = inAnalysisScope(changes[i].Path, repo.EffectiveInclude, repo.EffectiveExclude) || (changes[i].OriginalPath != "" && inAnalysisScope(changes[i].OriginalPath, repo.EffectiveInclude, repo.EffectiveExclude))
			if changes[i].InScope {
				allInScope++
				var mapped bool
				changes[i].MatchedShards, changes[i].MatchedDomains, mapped = matchEvidence(repo.Name, changes[i], evidence)
				if !mapped {
					key := repo.Name + ":" + changes[i].Path
					plan.UnmappedPaths = append(plan.UnmappedPaths, key)
					plan.FallbackReasons = append(plan.FallbackReasons, "unmapped_in_scope_path")
				}
			}
			plan.AffectedShards = append(plan.AffectedShards, changes[i].MatchedShards...)
			plan.AffectedDomains = append(plan.AffectedDomains, changes[i].MatchedDomains...)
		}
		delta.Changes = changes
		plan.RepoDeltas = append(plan.RepoDeltas, delta)
	}
	plan.FallbackReasons = uniqueSorted(plan.FallbackReasons...)
	plan.AffectedShards = uniqueSorted(plan.AffectedShards...)
	plan.AffectedDomains = uniqueSorted(plan.AffectedDomains...)
	plan.UnmappedPaths = uniqueSorted(plan.UnmappedPaths...)
	stale := map[string]struct{}{}
	for artifact, shards := range evidence.ArtifactShards {
		if intersects(shards, plan.AffectedShards) {
			stale[artifact] = struct{}{}
		}
	}
	for _, delta := range plan.RepoDeltas {
		for _, change := range delta.Changes {
			if !change.InScope {
				continue
			}
			for _, candidatePath := range []string{change.Path, change.OriginalPath} {
				for _, docID := range evidence.CitationDocuments[delta.Repo+"\x00"+candidatePath] {
					if artifact := evidence.DocumentPaths[docID]; artifact != "" {
						stale[artifact] = struct{}{}
					}
				}
				for _, artifact := range evidence.ProvenanceArtifacts[delta.Repo+"\x00"+candidatePath] {
					stale[artifact] = struct{}{}
				}
			}
		}
	}
	for _, artifact := range evidence.AllCanonicalPaths {
		if _, ok := stale[artifact]; ok {
			plan.StaleArtifactCandidates = append(plan.StaleArtifactCandidates, artifact)
		} else {
			plan.PreservedArtifactCandidates = append(plan.PreservedArtifactCandidates, artifact)
		}
	}
	if len(plan.FallbackReasons) > 0 {
		plan.Decision = "full_refresh_required"
		plan.PlannedActions = append(plan.PlannedActions, "conservative_full_refresh")
	} else if allInScope == 0 {
		plan.Decision = "unchanged_candidate"
		plan.PlannedActions = append(plan.PlannedActions, "candidate_no_op")
	} else {
		plan.Decision = "selective_candidate"
		plan.PlannedActions = append(plan.PlannedActions, "candidate_selective_execution")
	}
	normalizeImpactPlan(&plan)
	return plan
}

func ParseNameStatusZ(raw []byte) ([]PathChange, error) {
	parts := strings.Split(string(raw), "\x00")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	changes := []PathChange{}
	for i := 0; i < len(parts); {
		statusToken := parts[i]
		i++
		if statusToken == "" || i >= len(parts) {
			return nil, fmt.Errorf("malformed name-status record")
		}
		kind := statusToken[:1]
		change := PathChange{MatchedShards: []string{}, MatchedDomains: []string{}}
		switch kind {
		case "A":
			change.Status = "added"
		case "M", "T":
			change.Status = "modified"
		case "D":
			change.Status = "deleted"
		case "R", "C":
			if i+1 >= len(parts) {
				return nil, fmt.Errorf("malformed rename/copy record")
			}
			change.OriginalPath = path.Clean(parts[i])
			change.Path = path.Clean(parts[i+1])
			i += 2
			if kind == "R" {
				change.Status = "renamed"
			} else {
				change.Status = "copied"
			}
			changes = append(changes, change)
			continue
		default:
			return nil, fmt.Errorf("unsupported git status %q", statusToken)
		}
		change.Path = path.Clean(parts[i])
		i++
		changes = append(changes, change)
	}
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Path+"\x00"+changes[i].OriginalPath < changes[j].Path+"\x00"+changes[j].OriginalPath
	})
	return changes, nil
}

func inAnalysisScope(candidate string, includes, excludes []string) bool {
	candidate = strings.TrimPrefix(path.Clean(candidate), "./")
	included := len(includes) == 0
	for _, pattern := range includes {
		if matchPathPattern(pattern, candidate) {
			included = true
			break
		}
	}
	if !included {
		return false
	}
	for _, pattern := range excludes {
		if matchPathPattern(pattern, candidate) {
			return false
		}
	}
	return true
}

func matchPathPattern(pattern, candidate string) bool {
	return pathscope.Match(pattern, candidate)
}

func matchEvidence(repo string, change PathChange, evidence PriorEvidence) ([]string, []string, bool) {
	shards, domains := []string{}, []string{}
	mapped := false
	for _, scope := range evidence.Shards {
		if len(scope.RepoScopes) > 0 && !contains(scope.RepoScopes, repo) {
			continue
		}
		matched := len(scope.PathScopes) == 0
		for _, p := range scope.PathScopes {
			if matchPathPattern(p, change.Path) || (change.OriginalPath != "" && matchPathPattern(p, change.OriginalPath)) {
				matched = true
				break
			}
		}
		if matched {
			mapped = true
			shards = append(shards, scope.ShardID)
			domains = append(domains, scope.DomainID)
		}
	}
	for _, candidatePath := range []string{change.Path, change.OriginalPath} {
		key := repo + "\x00" + candidatePath
		if len(evidence.ProvenanceArtifacts[key]) > 0 {
			mapped = true
		}
		domains = append(domains, evidence.ProvenanceDomains[key]...)
	}
	return uniqueSorted(shards...), uniqueSorted(domains...), mapped
}

func intersects(left, right []string) bool {
	set := map[string]struct{}{}
	for _, value := range left {
		set[value] = struct{}{}
	}
	for _, value := range right {
		if _, ok := set[value]; ok {
			return true
		}
	}
	return false
}
