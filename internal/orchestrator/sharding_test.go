package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestDiscoverHeuristicShardPathsPrunesParentRoots(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	writeFile(t, filepath.Join(repoPath, "go.mod"), "module example.com/root\n")
	writeFile(t, filepath.Join(repoPath, "services", "api", "package.json"), "{\n  \"name\": \"api\"\n}\n")
	writeFile(t, filepath.Join(repoPath, "services", "web", "pyproject.toml"), "[project]\nname = \"web\"\n")
	writeFile(t, filepath.Join(repoPath, "services", "web", "sub", "Cargo.toml"), "[package]\nname = \"sub\"\n")

	paths, err := discoverHeuristicShardPaths(repoPath)
	if err != nil {
		t.Fatalf("discover heuristic shard paths: %v", err)
	}

	expected := []string{"services/api", "services/web/sub"}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected shard paths: got=%v want=%v", paths, expected)
	}
}

func TestApplyRepoAnalysisFiltersIncludeExclude(t *testing.T) {
	t.Parallel()

	filtered := applyRepoAnalysisFilters(
		[]string{"services/api", "services/web", "tools/dev"},
		&workspace.RepoAnalysisConfig{
			Include: []string{"services/**"},
			Exclude: []string{"services/web/**"},
		},
	)
	expected := []string{"services/api"}
	if !reflect.DeepEqual(filtered, expected) {
		t.Fatalf("unexpected filtered shards: got=%v want=%v", filtered, expected)
	}
}

func TestPlanScopePathsFallsBackToRootWhenFiltersProduceNoShards(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Include: []string{"docs/**"},
	})
	execution := pipelineExecution{
		workspace:         ws,
		resolvedRepoPaths: map[string]string{"orders-monolith": filepath.Join(ws.Path, "repos", "orders-monolith")},
	}

	paths, warnings := execution.planScopePaths("orders-monolith")
	if !reflect.DeepEqual(paths, []string{"."}) {
		t.Fatalf("expected root fallback path, got %v", paths)
	}
	if !containsWarning(warnings, "analysis filters produced zero shards") {
		t.Fatalf("expected analysis-filter warning, got %v", warnings)
	}
}

func TestRunInitParallelShardsUseDeterministicApplyOrderAndPersistShardPlan(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	service := NewService(WithRunner(deterministicApplyOrderRunner{}))

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	sharedEntityRaw, err := os.ReadFile(filepath.Join(ws.Path, "model/entities/svc.shared.yaml"))
	if err != nil {
		t.Fatalf("read shared entity: %v", err)
	}
	if !strings.Contains(string(sharedEntityRaw), "name: Shared services/web") {
		t.Fatalf("expected deterministic apply order to keep services/web value, got:\n%s", string(sharedEntityRaw))
	}

	step1Summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", "*-init-step1-collect-shard-summary-*.json"))
	if len(step1Summary.Items) != 2 {
		t.Fatalf("expected two step1 shard summary items, got %d", len(step1Summary.Items))
	}
	if len(step1Summary.Items[0].PathScopes) == 0 || step1Summary.Items[0].PathScopes[0] != "services/api" {
		t.Fatalf("expected first summary shard path to be services/api, got %+v", step1Summary.Items[0].PathScopes)
	}
	if len(step1Summary.Items[1].PathScopes) == 0 || step1Summary.Items[1].PathScopes[0] != "services/web" {
		t.Fatalf("expected second summary shard path to be services/web, got %+v", step1Summary.Items[1].PathScopes)
	}

	step1Plan := readSingleShardPlan(t, filepath.Join(ws.Path, "reports", "taskruns", "*-init-step1-collect-shard-plan-*.json"))
	if len(step1Plan.Items) != 2 {
		t.Fatalf("expected two step1 shard-plan items, got %d", len(step1Plan.Items))
	}
	if len(step1Plan.Items[0].PathScopes) == 0 || step1Plan.Items[0].PathScopes[0] != "services/api" {
		t.Fatalf("expected first shard-plan path to be services/api, got %+v", step1Plan.Items[0].PathScopes)
	}
	if len(step1Plan.Items[1].PathScopes) == 0 || step1Plan.Items[1].PathScopes[0] != "services/web" {
		t.Fatalf("expected second shard-plan path to be services/web, got %+v", step1Plan.Items[1].PathScopes)
	}

	step1Taskruns, err := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "*-init-step1-collect-domain-*-shard-*.json"))
	if err != nil {
		t.Fatalf("glob step1 shard taskruns: %v", err)
	}
	if len(step1Taskruns) != 2 {
		t.Fatalf("expected two step1 shard taskruns, got %d", len(step1Taskruns))
	}
	for _, candidate := range step1Taskruns {
		var payload contracts.TaskResult
		readJSONFile(t, candidate, &payload)
		if strings.TrimSpace(payload.Meta.ShardID) == "" {
			t.Fatalf("expected taskrun meta.shard_id in %s", candidate)
		}
		if strings.TrimSpace(payload.Meta.RepoScope) == "" {
			t.Fatalf("expected taskrun meta.repo_scope in %s", candidate)
		}
		if len(payload.Meta.PathScopes) == 0 {
			t.Fatalf("expected taskrun meta.path_scopes in %s", candidate)
		}
	}
}

func TestRunInitSemanticShardPlanIncludesGraphEdges(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoverySemantic,
	})
	repoRoot := filepath.Join(ws.Path, "repos", "orders-monolith")
	writeFile(t, filepath.Join(repoRoot, "services", "api", "main.go"), "package api\n\nimport _ \"orders-monolith/services/web\"\n")
	writeFile(t, filepath.Join(repoRoot, "services", "web", "main.go"), "package web\n")

	service := NewService()
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline in semantic mode: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	step1Plan := readSingleShardPlan(t, filepath.Join(ws.Path, "reports", "taskruns", "*-init-step1-collect-shard-plan-*.json"))
	if step1Plan.ShardMode != acpruntime.ExecutionShardDiscoverySemantic {
		t.Fatalf("expected semantic shard mode in plan, got %q", step1Plan.ShardMode)
	}
	if len(step1Plan.SemanticGraph) == 0 {
		t.Fatalf("expected non-empty semantic graph in shard plan artifact")
	}
	if step1Plan.SemanticGraph[0].RepoScope != "orders-monolith" {
		t.Fatalf("expected semantic graph edge repo_scope orders-monolith, got %+v", step1Plan.SemanticGraph[0])
	}
}

func TestRunInitBestEffortShardFailureContinuesAndFailsFinal(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	service := NewService(WithRunner(step1ShardFailureRunner{}))

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected run error due to partial shard failure")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if info.ErrorCode != runErrorCodePartialFailed {
		t.Fatalf("expected %q error code, got %q", runErrorCodePartialFailed, info.ErrorCode)
	}
	if info.CurrentStep != "init.step4.proposals" {
		t.Fatalf("expected pipeline to continue through step4, got current_step=%q", info.CurrentStep)
	}

	step1Summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", "*-init-step1-collect-shard-summary-*.json"))
	failedCount := 0
	succeededCount := 0
	for _, item := range step1Summary.Items {
		switch item.Status {
		case "failed":
			failedCount++
		case "succeeded":
			succeededCount++
		}
	}
	if failedCount == 0 || succeededCount == 0 {
		t.Fatalf("expected mixed shard statuses in step1 summary, got %+v", step1Summary.Items)
	}

	step3Summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", "*-init-step3-findings-shard-summary.json"))
	if len(step3Summary.Items) == 0 {
		t.Fatalf("expected step3 shard summary items")
	}
}

func TestRunInitFailFastStopsOnShardFailure(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyFailFast,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	service := NewService(WithRunner(step1ShardFailureRunner{}))

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected run error in fail_fast mode")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if info.ErrorCode == runErrorCodePartialFailed {
		t.Fatalf("did not expect %q in fail_fast mode", runErrorCodePartialFailed)
	}
	if info.CurrentStep != "init.step1.collect" {
		t.Fatalf("expected fail_fast stop at step1, got %q", info.CurrentStep)
	}

	step3Summaries, globErr := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "*-init-step3-findings-shard-summary*.json"))
	if globErr != nil {
		t.Fatalf("glob step3 shard summaries: %v", globErr)
	}
	if len(step3Summaries) != 0 {
		t.Fatalf("expected no step3 shard summaries in fail_fast mode, got %v", step3Summaries)
	}
}

func TestShardSchedulerRespectsSequentialAndParallelStrategies(t *testing.T) {
	t.Parallel()

	wsParallel := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
	})
	parallelRunner := &concurrencyProbeRunner{delay: 80 * time.Millisecond}
	parallelService := NewService(WithRunner(parallelRunner))
	parallelInfo, _, parallelErr := parallelService.Run(context.Background(), RunRequest{
		Workspace:      wsParallel,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if parallelErr != nil {
		t.Fatalf("run parallel init pipeline: %v", parallelErr)
	}
	if parallelInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded parallel run, got %s (%s)", parallelInfo.Status, parallelInfo.Error)
	}
	if parallelRunner.maxStep1Concurrency() < 2 {
		t.Fatalf("expected parallel step1 shard concurrency >= 2, got %d", parallelRunner.maxStep1Concurrency())
	}

	wsSequential := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategySequential,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
	})
	sequentialRunner := &concurrencyProbeRunner{delay: 80 * time.Millisecond}
	sequentialService := NewService(WithRunner(sequentialRunner))
	sequentialInfo, _, sequentialErr := sequentialService.Run(context.Background(), RunRequest{
		Workspace:      wsSequential,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if sequentialErr != nil {
		t.Fatalf("run sequential init pipeline: %v", sequentialErr)
	}
	if sequentialInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded sequential run, got %s (%s)", sequentialInfo.Status, sequentialInfo.Error)
	}
	if sequentialRunner.maxStep1Concurrency() != 1 {
		t.Fatalf("expected sequential step1 shard concurrency 1, got %d", sequentialRunner.maxStep1Concurrency())
	}
}

func TestRefreshPipelineUsesShardingForStep1AndStep3(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	service := NewService()

	initInfo, _, initErr := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if initErr != nil {
		t.Fatalf("prepare refresh with init pipeline: %v", initErr)
	}
	if initInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected init pre-run succeeded, got %s (%s)", initInfo.Status, initInfo.Error)
	}

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded refresh run, got %s (%s)", info.Status, info.Error)
	}

	step1Summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", "*-refresh-step1-collect-shard-summary-*.json"))
	if len(step1Summary.Items) == 0 {
		t.Fatalf("expected refresh step1 shard summary items")
	}
	step3Summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", "*-refresh-step3-findings-shard-summary.json"))
	if len(step3Summary.Items) == 0 {
		t.Fatalf("expected refresh step3 shard summary items")
	}
}

func TestSyntheticLargeMonorepoShardingProducesManyShardArtifacts(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   4,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
	})
	repoRoot := filepath.Join(ws.Path, "repos", "orders-monolith")
	for idx := 0; idx < 12; idx++ {
		modulePath := filepath.Join(repoRoot, "services", "module-"+strconv.Itoa(idx))
		writeFile(t, filepath.Join(modulePath, "package.json"), "{\n  \"name\": \"module\"\n}\n")
		writeFile(t, filepath.Join(modulePath, "README.md"), "# module\n")
	}

	service := NewService()
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline for synthetic monorepo: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	step1Summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", "*-init-step1-collect-shard-summary-*.json"))
	if len(step1Summary.Items) < 10 {
		t.Fatalf("expected at least 10 shards in synthetic monorepo, got %d", len(step1Summary.Items))
	}
	for _, item := range step1Summary.Items {
		if len(item.PathScopes) == 0 {
			t.Fatalf("expected non-empty path scope in summary item: %+v", item)
		}
	}
}

func TestMultiRepoShardingHandlesDifferentModuleCounts(t *testing.T) {
	t.Parallel()

	ws := createMultiRepoShardingWorkspace(t, map[string]int{
		"payments-service": 2,
		"users-service":    3,
	}, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   4,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
	})
	service := NewService()

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline for multi-repo workspace: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	step1Summaries, globErr := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "*-init-step1-collect-shard-summary-*.json"))
	if globErr != nil {
		t.Fatalf("glob step1 shard summaries: %v", globErr)
	}
	if len(step1Summaries) != 2 {
		t.Fatalf("expected two domain step1 shard summaries for two repos, got %d (%v)", len(step1Summaries), step1Summaries)
	}
	summarySizes := make([]int, 0, len(step1Summaries))
	for _, path := range step1Summaries {
		var summary runtimeShardSummary
		readJSONFile(t, path, &summary)
		summarySizes = append(summarySizes, len(summary.Items))
	}
	sort.Ints(summarySizes)
	expected := []int{2, 3}
	if !reflect.DeepEqual(summarySizes, expected) {
		t.Fatalf("unexpected per-domain shard counts: got=%v want=%v", summarySizes, expected)
	}

	step3Summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", "*-init-step3-findings-shard-summary.json"))
	if len(step3Summary.Items) != 5 {
		t.Fatalf("expected five step3 shards (2+3), got %d", len(step3Summary.Items))
	}
}

type shardingWorkspaceOptions struct {
	Include       []string
	Exclude       []string
	Strategy      string
	MaxParallel   int
	FailurePolicy string
	ShardMode     string
}

func createShardingWorkspace(t *testing.T, options shardingWorkspaceOptions) workspace.Root {
	t.Helper()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "orders-monolith")
	writeFile(t, filepath.Join(repoPath, "README.md"), "# orders-monolith\n")
	writeFile(t, filepath.Join(repoPath, "services", "api", "package.json"), "{\n  \"name\": \"api\"\n}\n")
	writeFile(t, filepath.Join(repoPath, "services", "api", "README.md"), "# api\n")
	writeFile(t, filepath.Join(repoPath, "services", "web", "package.json"), "{\n  \"name\": \"web\"\n}\n")
	writeFile(t, filepath.Join(repoPath, "services", "web", "README.md"), "# web\n")

	repo := workspace.RepoSource{
		Name: "orders-monolith",
		Path: repoPath,
	}
	if len(options.Include) > 0 || len(options.Exclude) > 0 {
		repo.Analysis = &workspace.RepoAnalysisConfig{
			Include: append([]string(nil), options.Include...),
			Exclude: append([]string(nil), options.Exclude...),
		}
	}

	manifest := workspace.Manifest{
		Version: 1,
		Repos:   []workspace.RepoSource{repo},
	}
	if runtimeConfig := runtimeConfigFromShardingOptions(options); runtimeConfig != nil {
		manifest.Runtime = runtimeConfig
	}

	manifestRaw, err := workspace.RenderManifest(manifest)
	if err != nil {
		t.Fatalf("render sharding manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, workspace.ManifestFileName), manifestRaw, 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}

func createMultiRepoShardingWorkspace(t *testing.T, moduleCounts map[string]int, options shardingWorkspaceOptions) workspace.Root {
	t.Helper()

	root := t.TempDir()
	repos := make([]workspace.RepoSource, 0, len(moduleCounts))
	names := make([]string, 0, len(moduleCounts))
	for name := range moduleCounts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		repoPath := filepath.Join(root, "repos", name)
		writeFile(t, filepath.Join(repoPath, "README.md"), "# "+name+"\n")
		moduleCount := moduleCounts[name]
		for idx := 0; idx < moduleCount; idx++ {
			modulePath := filepath.Join(repoPath, "services", "module-"+strconv.Itoa(idx))
			writeFile(t, filepath.Join(modulePath, "package.json"), "{\n  \"name\": \"module\"\n}\n")
			writeFile(t, filepath.Join(modulePath, "README.md"), "# module\n")
		}
		repos = append(repos, workspace.RepoSource{
			Name: name,
			Path: repoPath,
		})
	}

	manifest := workspace.Manifest{
		Version: 1,
		Repos:   repos,
		Runtime: runtimeConfigFromShardingOptions(options),
	}
	manifestRaw, err := workspace.RenderManifest(manifest)
	if err != nil {
		t.Fatalf("render multi-repo sharding manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, workspace.ManifestFileName), manifestRaw, 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}

func runtimeConfigFromShardingOptions(options shardingWorkspaceOptions) *workspace.RuntimeConfig {
	if strings.TrimSpace(options.Strategy) == "" &&
		options.MaxParallel <= 0 &&
		strings.TrimSpace(options.FailurePolicy) == "" &&
		strings.TrimSpace(options.ShardMode) == "" {
		return nil
	}
	execution := &workspace.RuntimeExecutionConfig{
		Strategy:      strings.TrimSpace(options.Strategy),
		FailurePolicy: strings.TrimSpace(options.FailurePolicy),
	}
	if options.MaxParallel > 0 {
		maxParallel := options.MaxParallel
		execution.MaxParallel = &maxParallel
	}
	if strings.TrimSpace(options.ShardMode) != "" {
		execution.ShardDiscovery = &workspace.RuntimeShardDiscoveryConfig{
			Mode: strings.TrimSpace(options.ShardMode),
		}
	}
	return &workspace.RuntimeConfig{
		Profile: &workspace.RuntimeProfileConfig{
			Execution: execution,
		},
	}
}

type deterministicApplyOrderRunner struct{}

func (deterministicApplyOrderRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if strings.HasSuffix(task.StepID, "step1.collect") {
		pathScope := "."
		if len(task.PathScopes) > 0 && strings.TrimSpace(task.PathScopes[0]) != "" {
			pathScope = strings.TrimSpace(task.PathScopes[0])
		}
		delay := 15 * time.Millisecond
		if pathScope == "services/api" {
			delay = 140 * time.Millisecond
		}
		select {
		case <-ctx.Done():
			return acpruntime.Result{}, ctx.Err()
		case <-time.After(delay):
		}

		repoScope := "orders-monolith"
		if len(task.RepoScopes) > 0 && strings.TrimSpace(task.RepoScopes[0]) != "" {
			repoScope = strings.TrimSpace(task.RepoScopes[0])
		}
		evidencePath := pathScope + "/README.md"
		if pathScope == "." {
			evidencePath = "README.md"
		}
		result := contracts.TaskResult{
			Meta: contracts.Meta{
				TaskID:     task.TaskID,
				StepID:     task.StepID,
				RunID:      task.RunID,
				Runtime:    contracts.RuntimeMeta{Name: "synthetic-shard-runner", Version: "test"},
				StartedAt:  task.StartedAtUTC.UTC().Format(time.RFC3339),
				FinishedAt: task.StartedAtUTC.UTC().Add(time.Second).Format(time.RFC3339),
				Workspace:  task.Workspace,
				ShardID:    task.ShardID,
				RepoScope:  task.RepoScope,
				RepoScopes: append([]string(nil), task.RepoScopes...),
				PathScopes: append([]string(nil), task.PathScopes...),
			},
			Summary: "synthetic shard collect for " + pathScope,
			Changeset: []contracts.Operation{
				{
					Op: "upsert_entity",
					Entity: &contracts.Entity{
						ID:   "svc.shared",
						Type: "service",
						Name: "Shared " + pathScope,
						Attributes: map[string]any{
							"repo_scope": repoScope,
						},
						Provenance: contracts.Provenance{
							Kind:       "observation",
							Confidence: 0.9,
							Evidence: []contracts.Evidence{
								{
									Repo: repoScope,
									Path: evidencePath,
								},
							},
						},
					},
				},
			},
		}
		return marshalSyntheticTaskResult(result)
	}
	return claudecode.FakeRunner{}.Run(ctx, task)
}

func (deterministicApplyOrderRunner) Preflight(context.Context) error {
	return nil
}

type step1ShardFailureRunner struct{}

func (step1ShardFailureRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if strings.HasSuffix(task.StepID, "step1.collect") && containsString(task.PathScopes, "services/api") {
		return acpruntime.Result{}, acpruntime.WrapRunnerError(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			"synthetic shard parse failure on services/api",
			errors.New("synthetic shard parse failure"),
		)
	}
	return claudecode.FakeRunner{}.Run(ctx, task)
}

func (step1ShardFailureRunner) Preflight(context.Context) error {
	return nil
}

type concurrencyProbeRunner struct {
	delay time.Duration

	mu      sync.Mutex
	current int
	max     int
}

func (r *concurrencyProbeRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if strings.HasSuffix(task.StepID, "step1.collect") {
		r.mu.Lock()
		r.current++
		if r.current > r.max {
			r.max = r.current
		}
		r.mu.Unlock()

		defer func() {
			r.mu.Lock()
			r.current--
			r.mu.Unlock()
		}()

		select {
		case <-ctx.Done():
			return acpruntime.Result{}, ctx.Err()
		case <-time.After(r.delay):
		}
	}
	return claudecode.FakeRunner{}.Run(ctx, task)
}

func (r *concurrencyProbeRunner) Preflight(context.Context) error {
	return nil
}

func (r *concurrencyProbeRunner) maxStep1Concurrency() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.max
}

func marshalSyntheticTaskResult(result contracts.TaskResult) (acpruntime.Result, error) {
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return acpruntime.Result{}, err
	}
	return acpruntime.Result{
		TaskResult: result,
		RawJSON:    raw,
	}, nil
}

func readSingleShardSummary(t *testing.T, pattern string) runtimeShardSummary {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob shard summary %q: %v", pattern, err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one shard summary for pattern %q, got %d (%v)", pattern, len(matches), matches)
	}
	var summary runtimeShardSummary
	readJSONFile(t, matches[0], &summary)
	return summary
}

func readSingleShardPlan(t *testing.T, pattern string) runtimeShardPlanArtifact {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob shard plan %q: %v", pattern, err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one shard plan for pattern %q, got %d (%v)", pattern, len(matches), matches)
	}
	var plan runtimeShardPlanArtifact
	readJSONFile(t, matches[0], &plan)
	return plan
}

func readJSONFile(t *testing.T, path string, dst any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(content, dst); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func containsWarning(warnings []string, needle string) bool {
	target := strings.TrimSpace(needle)
	for _, warning := range warnings {
		if strings.Contains(strings.TrimSpace(warning), target) {
			return true
		}
	}
	return false
}

func containsString(values []string, candidate string) bool {
	target := strings.TrimSpace(candidate)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
