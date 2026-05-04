package orchestrator

import (
	"context"
	"strings"
	"sync"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type runtimeShardPlan struct {
	SortKey     string
	ShardID     string
	RepoScopes  []string
	PathScopes  []string
	PrimaryRepo string
}

type ShardPlanInput struct {
	Workspace         workspace.Root
	ResolvedRepoPaths map[string]string
	ExecutionProfile  acpruntime.ExecutionValues
	RepoScopes        []string
}

type ShardPlanResult struct {
	Plans         []runtimeShardPlan
	Warnings      []string
	SemanticGraph []runtimeShardPlanGraphEdge
}

type ShardPlanner interface {
	PlanRuntimeShards(input ShardPlanInput) ShardPlanResult
}

type defaultShardPlanner struct{}

type ShardScheduleRequest struct {
	StepID           string
	DomainID         string
	Plans            []runtimeShardPlan
	SummaryState     *runtimeShardSummaryState
	Options          runtimeShardExecutionOptions
	TaskSuffixPrefix string
}

type ShardScheduler interface {
	ScheduleRuntimeShardRuns(ctx context.Context, request ShardScheduleRequest) ([]runtimeShardRunResult, error)
}

type defaultShardScheduler struct {
	execution *pipelineExecution
}

type ShardSummaryStore interface {
	LoadSummary(stepID string, domainID string) ([]runtimeShardSummaryEntry, error)
	PersistSummary(stepID string, domainID string, items []runtimeShardSummaryEntry) error
	RuntimeExecutionExists(path string) bool
	PersistRuntimeExecutionArtifact(path string, label string, raw []byte) error
}

type defaultShardSummaryStore struct {
	execution *pipelineExecution
}

type runtimeShardRunResult struct {
	Plan           runtimeShardPlan
	Prepared       runtimePreparedExecution
	Err            error
	AlreadyApplied bool
	FromCheckpoint bool
}

type runtimeShardExecutionOptions struct {
	Strategy      string
	MaxParallel   int
	FailurePolicy string
	BestEffort    bool
}

type runtimeShardSummary struct {
	Version       int                        `json:"version"`
	Meta          runtimeArtifactMeta        `json:"meta,omitempty"`
	RunID         string                     `json:"run_id"`
	StepID        string                     `json:"step_id"`
	DomainID      string                     `json:"domain_id,omitempty"`
	Strategy      string                     `json:"strategy"`
	MaxParallel   int                        `json:"max_parallel_tasks"`
	FailurePolicy string                     `json:"failure_policy"`
	ShardMode     string                     `json:"shard_discovery_mode"`
	GeneratedAt   string                     `json:"generated_at"`
	Items         []runtimeShardSummaryEntry `json:"items"`
}

type runtimeShardSummaryEntry struct {
	ShardID    string   `json:"shard_id"`
	RepoScopes []string `json:"repo_scopes,omitempty"`
	PathScopes []string `json:"path_scopes,omitempty"`
	Status     string   `json:"status"`
	TaskID     string   `json:"task_id,omitempty"`
	TaskRun    string   `json:"taskrun_path,omitempty"`
	Error      string   `json:"error,omitempty"`
}

type runtimeShardSummaryState struct {
	store       ShardSummaryStore
	stepID      string
	domainID    string
	singleShard bool
	entries     []runtimeShardSummaryEntry
	index       map[string]int
	mu          sync.Mutex
}

type runtimeShardPlanArtifact struct {
	Version       int                            `json:"version"`
	Meta          runtimeArtifactMeta            `json:"meta,omitempty"`
	RunID         string                         `json:"run_id"`
	StepID        string                         `json:"step_id"`
	DomainID      string                         `json:"domain_id,omitempty"`
	Strategy      string                         `json:"strategy"`
	MaxParallel   int                            `json:"max_parallel_tasks"`
	FailurePolicy string                         `json:"failure_policy"`
	ShardMode     string                         `json:"shard_discovery_mode"`
	PlannerNotes  []string                       `json:"planner_notes,omitempty"`
	SemanticGraph []runtimeShardPlanGraphEdge    `json:"semantic_graph,omitempty"`
	Items         []runtimeShardPlanArtifactItem `json:"items"`
}

type runtimeShardPlanArtifactItem struct {
	SortKey    string   `json:"sort_key"`
	ShardID    string   `json:"shard_id"`
	RepoScopes []string `json:"repo_scopes,omitempty"`
	PathScopes []string `json:"path_scopes,omitempty"`
}

type runtimeShardPlanGraphEdge struct {
	RepoScope string `json:"repo_scope"`
	FromPath  string `json:"from_path"`
	ToPath    string `json:"to_path"`
	Reason    string `json:"reason"`
}

type runtimeArtifactMeta struct {
	Runtime contracts.RuntimeMeta `json:"runtime"`
}

type heuristicShardDiscoveryResult struct {
	Paths             []string
	FallbackNoMarkers bool
}

const (
	maxAutoShardsPerRepo     = 16
	maxRuntimeShardIDLength  = 96
	runtimeShardIDHashLength = 12
)

func runtimeMetaForRunner(runner acpruntime.Runner) contracts.RuntimeMeta {
	if metadataRunner, ok := runner.(acpruntime.MetadataRunner); ok {
		meta := metadataRunner.RuntimeMeta()
		if strings.TrimSpace(meta.Name) != "" {
			return meta
		}
	}
	return contracts.RuntimeMeta{Name: "unknown"}
}

func (e *pipelineExecution) runtimeMetaForStep(stepID string) contracts.RuntimeMeta {
	if e.runnerResolver != nil {
		provider, runner, err := e.runnerResolver.RunnerForStep(stepID)
		if err == nil {
			meta := runtimeMetaForRunner(runner)
			if strings.TrimSpace(meta.Name) == "" || meta.Name == "unknown" {
				if provider != "" {
					meta.Name = string(provider)
				}
			}
			return meta
		}
	}
	if provider := e.stepProviders.ProviderForStep(stepID); provider != "" {
		return contracts.RuntimeMeta{Name: string(provider)}
	}
	return contracts.RuntimeMeta{Name: "unknown"}
}

var shardModuleMarkerFiles = map[string]struct{}{
	"go.mod":          {},
	"package.json":    {},
	"pyproject.toml":  {},
	"cargo.toml":      {},
	"pom.xml":         {},
	"build.gradle":    {},
	"settings.gradle": {},
	"workspace":       {},
	"module.bazel":    {},
}

var shardSkippedDirs = map[string]struct{}{
	".git":         {},
	".hg":          {},
	".svn":         {},
	"node_modules": {},
	"vendor":       {},
	"dist":         {},
	"build":        {},
	"target":       {},
	"tmp":          {},
	".idea":        {},
	".vscode":      {},
}

var semanticSourceExtensions = map[string]struct{}{
	".go":   {},
	".ts":   {},
	".tsx":  {},
	".js":   {},
	".jsx":  {},
	".py":   {},
	".java": {},
	".kt":   {},
	".rs":   {},
}
