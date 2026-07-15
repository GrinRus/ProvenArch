package refreshplan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/validation"
)

const (
	SourceRevisionsVersion        = 1
	ImpactPlanVersion             = 1
	RefreshExecutionVersion       = 1
	RefreshMaterializationVersion = 1
	MaxChangedPaths               = 10_000
)

type SourceRevisions struct {
	Version                  int            `json:"version"`
	RunID                    string         `json:"run_id"`
	Pipeline                 string         `json:"pipeline"`
	CapturedAt               string         `json:"captured_at"`
	BaselineRunID            *string        `json:"baseline_run_id"`
	AnalysisInputFingerprint string         `json:"analysis_input_fingerprint"`
	AnalysisInputsState      string         `json:"analysis_inputs_state"`
	InputIssues              []string       `json:"input_issues"`
	Repos                    []RepoRevision `json:"repos"`
}

type RepoRevision struct {
	Name             string   `json:"name"`
	SourceKind       string   `json:"source_kind"`
	Path             string   `json:"path,omitempty"`
	GitURL           string   `json:"git_url,omitempty"`
	ConfiguredRef    string   `json:"configured_ref"`
	CurrentRevision  *string  `json:"current_revision"`
	BaselineRevision *string  `json:"baseline_revision"`
	WorktreeState    string   `json:"worktree_state"`
	EffectiveInclude []string `json:"effective_include"`
	EffectiveExclude []string `json:"effective_exclude"`
	Comparison       string   `json:"comparison"`
	FallbackReasons  []string `json:"fallback_reasons"`
}

type ImpactPlan struct {
	Version                     int         `json:"version"`
	RunID                       string      `json:"run_id"`
	Pipeline                    string      `json:"pipeline"`
	GeneratedAt                 string      `json:"generated_at"`
	BaselineRunID               *string     `json:"baseline_run_id"`
	Enforcement                 string      `json:"enforcement"`
	Decision                    string      `json:"decision"`
	FallbackReasons             []string    `json:"fallback_reasons"`
	RepoDeltas                  []RepoDelta `json:"repo_deltas"`
	AffectedShards              []string    `json:"affected_shards"`
	AffectedDomains             []string    `json:"affected_domains"`
	UnmappedPaths               []string    `json:"unmapped_paths"`
	StaleArtifactCandidates     []string    `json:"stale_artifact_candidates"`
	PreservedArtifactCandidates []string    `json:"preserved_artifact_candidates"`
	PlannedActions              []string    `json:"planned_actions"`
}

type RefreshExecution struct {
	Version              int           `json:"version"`
	RunID                string        `json:"run_id"`
	GeneratedAt          string        `json:"generated_at"`
	BaselineRunID        *string       `json:"baseline_run_id"`
	PlanDecision         string        `json:"plan_decision"`
	Mode                 string        `json:"mode"`
	ReasonCodes          []string      `json:"reason_codes"`
	SourceRanges         []SourceRange `json:"source_ranges"`
	AffectedShards       []string      `json:"affected_shards"`
	PreservedShards      []string      `json:"preserved_shards"`
	ProviderStepsSkipped bool          `json:"provider_steps_skipped"`
	ArtifactPath         string        `json:"artifact_path"`
}

type SourceRange struct {
	Repo             string  `json:"repo"`
	BaselineRevision *string `json:"baseline_revision"`
	CurrentRevision  *string `json:"current_revision"`
}

type RefreshMaterialization struct {
	Version       int                `json:"version"`
	RunID         string             `json:"run_id"`
	GeneratedAt   string             `json:"generated_at"`
	BaselineRunID *string            `json:"baseline_run_id"`
	Mode          string             `json:"mode"`
	Decisions     []ArtifactDecision `json:"decisions"`
}

type ArtifactDecision struct {
	Path          string   `json:"path"`
	Action        string   `json:"action"`
	ReasonCodes   []string `json:"reason_codes"`
	SourceRunID   *string  `json:"source_run_id"`
	ContentSHA256 *string  `json:"content_sha256"`
}

type RepoDelta struct {
	Repo             string       `json:"repo"`
	BaselineRevision *string      `json:"baseline_revision"`
	CurrentRevision  *string      `json:"current_revision"`
	Comparison       string       `json:"comparison"`
	ChangesComplete  bool         `json:"changes_complete"`
	ChangedPathCount int          `json:"changed_path_count"`
	Changes          []PathChange `json:"changes"`
}

type PathChange struct {
	Status         string   `json:"status"`
	Path           string   `json:"path"`
	OriginalPath   string   `json:"original_path,omitempty"`
	InScope        bool     `json:"in_scope"`
	MatchedShards  []string `json:"matched_shards"`
	MatchedDomains []string `json:"matched_domains"`
}

type ShardScope struct {
	ShardID    string
	DomainID   string
	RepoScopes []string
	PathScopes []string
}

type PriorEvidence struct {
	Shards              []ShardScope
	ArtifactShards      map[string][]string
	CitationDocuments   map[string][]string
	DocumentPaths       map[string]string
	ProvenanceDomains   map[string][]string
	ProvenanceArtifacts map[string][]string
	AllCanonicalPaths   []string
	Readable            bool
}

func ParseSourceRevisions(raw []byte) (SourceRevisions, error) {
	if err := validation.ValidateRawJSON(validation.SourceRevisionsSchema, raw); err != nil {
		return SourceRevisions{}, fmt.Errorf("source revisions are invalid: %w", err)
	}
	var value SourceRevisions
	if err := json.Unmarshal(raw, &value); err != nil {
		return SourceRevisions{}, fmt.Errorf("decode source revisions: %w", err)
	}
	normalizeSourceRevisions(&value)
	return value, nil
}

func ParseImpactPlan(raw []byte) (ImpactPlan, error) {
	if err := validation.ValidateRawJSON(validation.RefreshImpactPlanSchema, raw); err != nil {
		return ImpactPlan{}, fmt.Errorf("refresh impact plan is invalid: %w", err)
	}
	var value ImpactPlan
	if err := json.Unmarshal(raw, &value); err != nil {
		return ImpactPlan{}, fmt.Errorf("decode refresh impact plan: %w", err)
	}
	normalizeImpactPlan(&value)
	return value, nil
}

func ParseRefreshExecution(raw []byte) (RefreshExecution, error) {
	if err := validation.ValidateRawJSON(validation.RefreshExecutionSchema, raw); err != nil {
		return RefreshExecution{}, fmt.Errorf("refresh execution is invalid: %w", err)
	}
	var value RefreshExecution
	if err := json.Unmarshal(raw, &value); err != nil {
		return RefreshExecution{}, fmt.Errorf("decode refresh execution: %w", err)
	}
	normalizeRefreshExecution(&value)
	return value, nil
}

func MarshalRefreshExecution(value RefreshExecution) ([]byte, error) {
	normalizeRefreshExecution(&value)
	return json.MarshalIndent(value, "", "  ")
}

func ParseRefreshMaterialization(raw []byte) (RefreshMaterialization, error) {
	if err := validation.ValidateRawJSON(validation.RefreshMaterializationSchema, raw); err != nil {
		return RefreshMaterialization{}, fmt.Errorf("refresh materialization is invalid: %w", err)
	}
	var value RefreshMaterialization
	if err := json.Unmarshal(raw, &value); err != nil {
		return RefreshMaterialization{}, fmt.Errorf("decode refresh materialization: %w", err)
	}
	normalizeRefreshMaterialization(&value)
	return value, nil
}

func MarshalRefreshMaterialization(value RefreshMaterialization) ([]byte, error) {
	normalizeRefreshMaterialization(&value)
	return json.MarshalIndent(value, "", "  ")
}

func normalizeRefreshExecution(value *RefreshExecution) {
	if value.ReasonCodes == nil {
		value.ReasonCodes = []string{}
	}
	if value.SourceRanges == nil {
		value.SourceRanges = []SourceRange{}
	}
	if value.AffectedShards == nil {
		value.AffectedShards = []string{}
	}
	if value.PreservedShards == nil {
		value.PreservedShards = []string{}
	}
	sort.Strings(value.ReasonCodes)
	sort.Strings(value.AffectedShards)
	sort.Strings(value.PreservedShards)
	sort.Slice(value.SourceRanges, func(i, j int) bool { return value.SourceRanges[i].Repo < value.SourceRanges[j].Repo })
}

func normalizeRefreshMaterialization(value *RefreshMaterialization) {
	if value.Decisions == nil {
		value.Decisions = []ArtifactDecision{}
	}
	for i := range value.Decisions {
		if value.Decisions[i].ReasonCodes == nil {
			value.Decisions[i].ReasonCodes = []string{}
		}
		sort.Strings(value.Decisions[i].ReasonCodes)
	}
	sort.Slice(value.Decisions, func(i, j int) bool { return value.Decisions[i].Path < value.Decisions[j].Path })
}

func MarshalSourceRevisions(value SourceRevisions) ([]byte, error) {
	normalizeSourceRevisions(&value)
	return json.MarshalIndent(value, "", "  ")
}

func MarshalImpactPlan(value ImpactPlan) ([]byte, error) {
	normalizeImpactPlan(&value)
	return json.MarshalIndent(value, "", "  ")
}

func normalizeSourceRevisions(value *SourceRevisions) {
	if value.InputIssues == nil {
		value.InputIssues = []string{}
	}
	if value.Repos == nil {
		value.Repos = []RepoRevision{}
	}
	sort.Strings(value.InputIssues)
	sort.Slice(value.Repos, func(i, j int) bool { return value.Repos[i].Name < value.Repos[j].Name })
	for i := range value.Repos {
		sort.Strings(value.Repos[i].EffectiveInclude)
		sort.Strings(value.Repos[i].EffectiveExclude)
		sort.Strings(value.Repos[i].FallbackReasons)
	}
}

func normalizeImpactPlan(value *ImpactPlan) {
	for _, values := range [][]string{value.FallbackReasons, value.AffectedShards, value.AffectedDomains, value.UnmappedPaths, value.StaleArtifactCandidates, value.PreservedArtifactCandidates, value.PlannedActions} {
		sort.Strings(values)
	}
	if value.FallbackReasons == nil {
		value.FallbackReasons = []string{}
	}
	if value.RepoDeltas == nil {
		value.RepoDeltas = []RepoDelta{}
	}
	if value.AffectedShards == nil {
		value.AffectedShards = []string{}
	}
	if value.AffectedDomains == nil {
		value.AffectedDomains = []string{}
	}
	if value.UnmappedPaths == nil {
		value.UnmappedPaths = []string{}
	}
	if value.StaleArtifactCandidates == nil {
		value.StaleArtifactCandidates = []string{}
	}
	if value.PreservedArtifactCandidates == nil {
		value.PreservedArtifactCandidates = []string{}
	}
	if value.PlannedActions == nil {
		value.PlannedActions = []string{}
	}
	sort.Slice(value.RepoDeltas, func(i, j int) bool { return value.RepoDeltas[i].Repo < value.RepoDeltas[j].Repo })
	for i := range value.RepoDeltas {
		sort.Slice(value.RepoDeltas[i].Changes, func(a, b int) bool {
			left := value.RepoDeltas[i].Changes[a].Path + "\x00" + value.RepoDeltas[i].Changes[a].OriginalPath
			right := value.RepoDeltas[i].Changes[b].Path + "\x00" + value.RepoDeltas[i].Changes[b].OriginalPath
			return left < right
		})
		for j := range value.RepoDeltas[i].Changes {
			sort.Strings(value.RepoDeltas[i].Changes[j].MatchedShards)
			sort.Strings(value.RepoDeltas[i].Changes[j].MatchedDomains)
		}
	}
}

func uniqueSorted(values ...string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
