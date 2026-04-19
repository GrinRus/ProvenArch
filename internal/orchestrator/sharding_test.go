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
	"github.com/GrinRus/ProvenArch/internal/model"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestDiscoverHeuristicShardPathsCoversResidualFilesAndDirsWithoutOverlap(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	writeFile(t, filepath.Join(repoPath, "go.mod"), "module example.com/root\n")
	writeFile(t, filepath.Join(repoPath, "README.md"), "# root\n")
	writeFile(t, filepath.Join(repoPath, "docs", "architecture.md"), "# docs\n")
	writeFile(t, filepath.Join(repoPath, "services", "api", "package.json"), "{\n  \"name\": \"api\"\n}\n")
	writeFile(t, filepath.Join(repoPath, "services", "web", "pyproject.toml"), "[project]\nname = \"web\"\n")
	writeFile(t, filepath.Join(repoPath, "services", "web", "sub", "Cargo.toml"), "[package]\nname = \"sub\"\n")

	paths, err := discoverHeuristicShardPaths(repoPath)
	if err != nil {
		t.Fatalf("discover heuristic shard paths: %v", err)
	}

	expected := []string{"README.md", "docs", "go.mod", "services/api", "services/web/pyproject.toml", "services/web/sub"}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected shard paths: got=%v want=%v", paths, expected)
	}
	for i, candidate := range paths {
		for j, other := range paths {
			if i == j {
				continue
			}
			if candidate == "." || other == "." {
				t.Fatalf("unexpected root shard in heuristic plan: %v", paths)
			}
			if strings.HasPrefix(other, candidate+"/") {
				t.Fatalf("overlapping shard paths detected: %q and %q", candidate, other)
			}
		}
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
	if strings.TrimSpace(step1Summary.Meta.Runtime.Name) == "" {
		t.Fatalf("expected non-empty shard summary runtime name")
	}
	if len(step1Summary.Items) < 2 {
		t.Fatalf("expected at least two step1 shard summary items, got %d", len(step1Summary.Items))
	}
	summaryScopes := make([]string, 0, len(step1Summary.Items))
	for _, item := range step1Summary.Items {
		if len(item.PathScopes) == 0 {
			continue
		}
		summaryScopes = append(summaryScopes, item.PathScopes[0])
	}
	if !containsString(summaryScopes, "services/api") || !containsString(summaryScopes, "services/web") {
		t.Fatalf("expected summary shards to include services/api and services/web, got %v", summaryScopes)
	}

	step1Plan := readSingleShardPlan(t, filepath.Join(ws.Path, "reports", "taskruns", "*-init-step1-collect-shard-plan-*.json"))
	if strings.TrimSpace(step1Plan.Meta.Runtime.Name) == "" {
		t.Fatalf("expected non-empty shard plan runtime name")
	}
	if len(step1Plan.Items) < 2 {
		t.Fatalf("expected at least two step1 shard-plan items, got %d", len(step1Plan.Items))
	}
	planScopes := make([]string, 0, len(step1Plan.Items))
	for _, item := range step1Plan.Items {
		if len(item.PathScopes) == 0 {
			continue
		}
		planScopes = append(planScopes, item.PathScopes[0])
	}
	if !containsString(planScopes, "services/api") || !containsString(planScopes, "services/web") {
		t.Fatalf("expected shard-plan to include services/api and services/web, got %v", planScopes)
	}

	step1Taskruns, err := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "*-init-step1-collect-domain-*-shard-*.json"))
	if err != nil {
		t.Fatalf("glob step1 shard taskruns: %v", err)
	}
	if len(step1Taskruns) < 2 {
		t.Fatalf("expected at least two step1 shard taskruns, got %d", len(step1Taskruns))
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

func TestRunInitAllStep1ShardsFailedMarksEvidenceUnusableAndSkipsFindingsRuntime(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	service := NewService(WithRunner(allStep1ShardFailureRunner{}))

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected run error due to unusable collect evidence")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if info.ErrorCode != runErrorCodePartialFailed {
		t.Fatalf("expected %q error code, got %q", runErrorCodePartialFailed, info.ErrorCode)
	}
	if info.CurrentStep != "init.step4.proposals" {
		t.Fatalf("expected pipeline to continue through step4, got %q", info.CurrentStep)
	}

	quality := readRunQualitySummary(t, ws, info.RunID)
	if quality.EvidenceState.Collect.Status != "unusable" {
		t.Fatalf("expected collect evidence unusable, got %+v", quality.EvidenceState.Collect)
	}
	if quality.EvidenceState.Findings.Status != "skipped" {
		t.Fatalf("expected findings evidence skipped, got %+v", quality.EvidenceState.Findings)
	}
	if quality.EvidenceState.ReportMode != "incomplete" {
		t.Fatalf("expected incomplete report mode, got %+v", quality.EvidenceState)
	}

	step3Summaries, globErr := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "*-init-step3-findings-shard-summary*.json"))
	if globErr != nil {
		t.Fatalf("glob step3 shard summaries: %v", globErr)
	}
	if len(step3Summaries) != 0 {
		t.Fatalf("expected no step3 shard summary artifacts when findings runtime is skipped, got %v", step3Summaries)
	}

	serviceCatalog := mustReadFile(t, filepath.Join(ws.Path, "reports/as-is/service-catalog.md"))
	if !strings.Contains(serviceCatalog, "Analysis incomplete.") {
		t.Fatalf("expected incomplete-analysis banner in service catalog, got:\n%s", serviceCatalog)
	}
	if !strings.Contains(serviceCatalog, "No evidence-backed services were materialized because analysis did not complete.") {
		t.Fatalf("expected unusable collect empty-state, got:\n%s", serviceCatalog)
	}
	if strings.Contains(serviceCatalog, "No services found.") {
		t.Fatalf("did not expect misleading service empty-state, got:\n%s", serviceCatalog)
	}

	coverageSummary := mustReadFile(t, filepath.Join(ws.Path, "reports/coverage/summary.md"))
	if !strings.Contains(coverageSummary, "Unknown due to incomplete analysis.") {
		t.Fatalf("expected incomplete coverage wording, got:\n%s", coverageSummary)
	}
	openQuestions := mustReadFile(t, filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if !strings.Contains(openQuestions, "Open questions unavailable due to incomplete analysis.") {
		t.Fatalf("expected incomplete questions wording, got:\n%s", openQuestions)
	}

	findingsReport := mustReadFile(t, filepath.Join(ws.Path, "reports/findings/findings.md"))
	if !strings.Contains(findingsReport, "Findings unavailable because analysis did not complete.") {
		t.Fatalf("expected incomplete findings wording, got:\n%s", findingsReport)
	}

	proposal := mustReadFile(t, filepath.Join(ws.Path, "proposals/proposal-baseline/proposal.md"))
	if !strings.Contains(proposal, "Proposal generation incomplete because no reliable findings set was produced.") {
		t.Fatalf("expected incomplete proposal wording, got:\n%s", proposal)
	}

	architectSummary := mustReadFile(t, filepath.Join(ws.Path, "reports/agent-outputs/architect/summary.md"))
	if !strings.Contains(architectSummary, "Analysis incomplete.") {
		t.Fatalf("expected incomplete-analysis banner in architect summary, got:\n%s", architectSummary)
	}
}

func TestRunInitPartialCollectMarksArtifactsIncompleteAndKeepsMaterializedData(t *testing.T) {
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
	if info.ErrorCode != runErrorCodePartialFailed {
		t.Fatalf("expected %q error code, got %q", runErrorCodePartialFailed, info.ErrorCode)
	}

	quality := readRunQualitySummary(t, ws, info.RunID)
	if quality.EvidenceState.Collect.Status != "partial" {
		t.Fatalf("expected collect evidence partial, got %+v", quality.EvidenceState.Collect)
	}
	if quality.EvidenceState.ReportMode != "incomplete" {
		t.Fatalf("expected incomplete report mode, got %+v", quality.EvidenceState)
	}

	serviceCatalog := mustReadFile(t, filepath.Join(ws.Path, "reports/as-is/service-catalog.md"))
	if !strings.Contains(serviceCatalog, "Partial analysis. Some shards failed; downstream content may be incomplete.") {
		t.Fatalf("expected partial-analysis banner in service catalog, got:\n%s", serviceCatalog)
	}
	if !strings.Contains(serviceCatalog, "svc.orders-monolith") {
		t.Fatalf("expected materialized service data to remain in partial analysis, got:\n%s", serviceCatalog)
	}
	if strings.Contains(serviceCatalog, "No evidence-backed services were materialized") {
		t.Fatalf("did not expect unusable collect empty-state for partial analysis, got:\n%s", serviceCatalog)
	}

	findingsReport := mustReadFile(t, filepath.Join(ws.Path, "reports/findings/findings.md"))
	if !strings.Contains(findingsReport, "Partial analysis. Some shards failed; downstream content may be incomplete.") {
		t.Fatalf("expected partial-analysis banner in findings report, got:\n%s", findingsReport)
	}

	serviceDetails := mustReadFile(t, filepath.Join(ws.Path, "reports/as-is/services/svc.orders-monolith.md"))
	if !strings.Contains(serviceDetails, "Partial analysis. Some shards failed; downstream content may be incomplete.") {
		t.Fatalf("expected partial-analysis banner in per-service report, got:\n%s", serviceDetails)
	}
}

func TestRunInitStep3AllShardsFailedKeepsAsIsButMarksFindingsUnusable(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	service := NewService(WithRunner(step3ParseFailureRunner{}))

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected run error due to findings shard failures")
	}
	if info.ErrorCode != runErrorCodePartialFailed {
		t.Fatalf("expected %q error code, got %q", runErrorCodePartialFailed, info.ErrorCode)
	}

	quality := readRunQualitySummary(t, ws, info.RunID)
	if quality.EvidenceState.Collect.Status != "usable" {
		t.Fatalf("expected collect evidence usable, got %+v", quality.EvidenceState.Collect)
	}
	if quality.EvidenceState.Findings.Status != "unusable" {
		t.Fatalf("expected findings evidence unusable, got %+v", quality.EvidenceState.Findings)
	}
	if quality.EvidenceState.ReportMode != "incomplete" {
		t.Fatalf("expected incomplete report mode, got %+v", quality.EvidenceState)
	}

	step3Summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", "*-init-step3-findings-shard-summary.json"))
	if len(step3Summary.Items) == 0 {
		t.Fatalf("expected step3 shard summary items")
	}
	for _, item := range step3Summary.Items {
		if item.Status != "failed" {
			t.Fatalf("expected all step3 shard summary items failed, got %+v", step3Summary.Items)
		}
	}

	serviceCatalog := mustReadFile(t, filepath.Join(ws.Path, "reports/as-is/service-catalog.md"))
	if !strings.Contains(serviceCatalog, "svc.orders-monolith") {
		t.Fatalf("expected as-is artifacts from successful collect step, got:\n%s", serviceCatalog)
	}
	if strings.Contains(serviceCatalog, "No evidence-backed services were materialized") {
		t.Fatalf("did not expect unusable collect empty-state, got:\n%s", serviceCatalog)
	}

	findingsReport := mustReadFile(t, filepath.Join(ws.Path, "reports/findings/findings.md"))
	if !strings.Contains(findingsReport, "Findings unavailable because analysis did not complete.") {
		t.Fatalf("expected incomplete findings wording, got:\n%s", findingsReport)
	}

	proposal := mustReadFile(t, filepath.Join(ws.Path, "proposals/proposal-baseline/proposal.md"))
	if !strings.Contains(proposal, "Proposal generation incomplete because no reliable findings set was produced.") {
		t.Fatalf("expected incomplete proposal wording, got:\n%s", proposal)
	}

	adrDraft := mustReadFile(t, filepath.Join(ws.Path, "proposals/proposal-baseline/ADR.md"))
	if !strings.Contains(adrDraft, "This draft is triage-only because analysis did not complete.") {
		t.Fatalf("expected triage-only note in ADR draft, got:\n%s", adrDraft)
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

	quality := readRunQualitySummary(t, ws, info.RunID)
	if quality.EvidenceState.Collect.Status == "usable" {
		t.Fatalf("expected non-usable collect evidence after fail_fast step1 abort, got %+v", quality.EvidenceState.Collect)
	}
	if quality.EvidenceState.Collect.PlannedShards == 0 {
		t.Fatalf("expected collect shard outcome to be preserved after fail_fast abort, got %+v", quality.EvidenceState.Collect)
	}
	if quality.EvidenceState.Findings.Status != "skipped" {
		t.Fatalf("expected findings evidence skipped after pipeline abort, got %+v", quality.EvidenceState.Findings)
	}
	if quality.EvidenceState.ReportMode != "incomplete" {
		t.Fatalf("expected incomplete report mode after pipeline abort, got %+v", quality.EvidenceState)
	}

	step3Summaries, globErr := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "*-init-step3-findings-shard-summary*.json"))
	if globErr != nil {
		t.Fatalf("glob step3 shard summaries: %v", globErr)
	}
	if len(step3Summaries) != 0 {
		t.Fatalf("expected no step3 shard summaries in fail_fast mode, got %v", step3Summaries)
	}

	serviceCatalog := mustReadFile(t, filepath.Join(ws.Path, "reports/as-is/service-catalog.md"))
	if !strings.Contains(serviceCatalog, "Analysis incomplete.") && !strings.Contains(serviceCatalog, "Partial analysis. Some shards failed; downstream content may be incomplete.") {
		t.Fatalf("expected incomplete/partial banner in service catalog after fail_fast abort, got:\n%s", serviceCatalog)
	}

	coverageSummary := mustReadFile(t, filepath.Join(ws.Path, "reports/coverage/summary.md"))
	if !strings.Contains(coverageSummary, "Analysis incomplete.") && !strings.Contains(coverageSummary, "Partial analysis. Some shards failed; downstream content may be incomplete.") {
		t.Fatalf("expected incomplete/partial banner in coverage summary after fail_fast abort, got:\n%s", coverageSummary)
	}

	openQuestions := mustReadFile(t, filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if !strings.Contains(openQuestions, "Analysis incomplete.") && !strings.Contains(openQuestions, "Partial analysis. Some shards failed; downstream content may be incomplete.") {
		t.Fatalf("expected incomplete/partial banner in open-questions report after fail_fast abort, got:\n%s", openQuestions)
	}

	findingsReport := mustReadFile(t, filepath.Join(ws.Path, "reports/findings/findings.md"))
	if !strings.Contains(findingsReport, "Findings unavailable because analysis did not complete.") {
		t.Fatalf("expected incomplete findings wording after fail_fast abort, got:\n%s", findingsReport)
	}

	proposal := mustReadFile(t, filepath.Join(ws.Path, "proposals/proposal-baseline/proposal.md"))
	if !strings.Contains(proposal, "Proposal generation incomplete because no reliable findings set was produced.") {
		t.Fatalf("expected incomplete proposal wording after fail_fast abort, got:\n%s", proposal)
	}

	architectSummary := mustReadFile(t, filepath.Join(ws.Path, "reports/agent-outputs/architect/summary.md"))
	if !strings.Contains(architectSummary, "Analysis incomplete.") && !strings.Contains(architectSummary, "Partial analysis. Some shards failed; downstream content may be incomplete.") {
		t.Fatalf("expected incomplete/partial banner in architect summary after fail_fast abort, got:\n%s", architectSummary)
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

func TestRepoWithoutMarkerFilesFallsBackToSingleRootShard(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	writeFile(t, filepath.Join(repoPath, "README.md"), "# root\n")
	writeFile(t, filepath.Join(repoPath, "docs", "guide.md"), "# docs\n")

	discovery, err := discoverHeuristicShardPathsWithMeta(repoPath)
	if err != nil {
		t.Fatalf("discover heuristic shard paths: %v", err)
	}
	if !discovery.FallbackNoMarkers {
		t.Fatalf("expected no-marker fallback")
	}
	if !reflect.DeepEqual(discovery.Paths, []string{"."}) {
		t.Fatalf("expected single root shard, got %v", discovery.Paths)
	}
}

func TestSyntheticLargeMonorepoShardingCoalescesStructurallyByTopLevelSegment(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   4,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
	})
	repoRoot := filepath.Join(ws.Path, "repos", "orders-monolith")
	for idx := 0; idx < 20; idx++ {
		modulePath := filepath.Join(repoRoot, "services", "module-"+strconv.Itoa(idx))
		writeFile(t, filepath.Join(modulePath, "package.json"), "{\n  \"name\": \"module\"\n}\n")
		writeFile(t, filepath.Join(modulePath, "README.md"), "# module\n")
	}
	for idx := 0; idx < 6; idx++ {
		modulePath := filepath.Join(repoRoot, "libs", "lib-"+strconv.Itoa(idx))
		writeFile(t, filepath.Join(modulePath, "package.json"), "{\n  \"name\": \"lib\"\n}\n")
		writeFile(t, filepath.Join(modulePath, "README.md"), "# lib\n")
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
	if len(step1Summary.Items) != 3 {
		t.Fatalf("expected structural coalescing to reduce step1 shards to 3, got %d", len(step1Summary.Items))
	}
	got := [][]string{}
	for _, item := range step1Summary.Items {
		got = append(got, append([]string(nil), item.PathScopes...))
	}
	want := [][]string{{"README.md"}, {"libs"}, {"services"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected coalesced step1 path scopes: got=%v want=%v", got, want)
	}
}

func TestAutoResumeRefreshStep1ReplaysSucceededAndCheckpointedShards(t *testing.T) {
	t.Parallel()

	sourceWS := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	sourceService := NewService(WithRunner(deterministicApplyOrderRunner{}))
	if info, _, err := sourceService.Run(context.Background(), RunRequest{
		Workspace:      sourceWS,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil || info.Status != RunStatusSucceeded {
		t.Fatalf("prepare source init run failed: status=%s err=%v", info.Status, err)
	}
	sourceRefreshInfo, _, err := sourceService.Run(context.Background(), RunRequest{
		Workspace:      sourceWS,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("prepare source refresh run failed: %v", err)
	}
	if sourceRefreshInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected successful source refresh run, got %s (%s)", sourceRefreshInfo.Status, sourceRefreshInfo.Error)
	}

	targetWS := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	targetService := NewService(WithRunner(deterministicApplyOrderRunner{}))
	if info, _, err := targetService.Run(context.Background(), RunRequest{
		Workspace:      targetWS,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil || info.Status != RunStatusSucceeded {
		t.Fatalf("prepare target init run failed: status=%s err=%v", info.Status, err)
	}

	resumeRunID := "run_refresh_resume_step1"
	step1Summary := copyRefreshShardArtifacts(t, sourceWS, targetWS, sourceRefreshInfo.RunID, resumeRunID, "refresh.step1.collect", func(item runtimeShardSummaryEntry) string {
		if len(item.PathScopes) > 0 && item.PathScopes[0] == "services/web" {
			return "checkpointed"
		}
		return "succeeded"
	}, func(item runtimeShardSummaryEntry) bool {
		return len(item.PathScopes) > 0 && item.PathScopes[0] == "services/api"
	})
	if len(step1Summary.Items) < 2 {
		t.Fatalf("expected at least two step1 shards, got %d", len(step1Summary.Items))
	}

	historySeed := NewService(
		WithHistoryWorkspace(targetWS),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
		}),
	)
	historySeed.storeRun(runRecord{
		info: RunInfo{
			RunID:       resumeRunID,
			Pipeline:    string(PipelineRefresh),
			Status:      RunStatusRunning,
			StartedAt:   time.Date(2026, 4, 16, 9, 30, 0, 0, time.UTC),
			CurrentStep: "refresh.step1.collect",
		},
	})

	resumeRunner := &guardedResumeRunner{
		blockedSteps: map[string]struct{}{
			"refresh.step1.collect": {},
		},
	}
	resumedService := NewService(
		WithHistoryWorkspace(targetWS),
		WithResumeStaleAsyncRuns(),
		WithRunner(resumeRunner),
	)
	status := waitForTerminalRunStatus(t, targetWS, resumeRunID, 10*time.Second)
	if status != RunStatusSucceeded {
		info, _ := resumedService.GetRun(resumeRunID)
		t.Fatalf("expected resumed run to succeed, got status=%s error_code=%s error=%s", status, info.ErrorCode, info.Error)
	}
	if resumeRunner.blockedCallCount() != 0 {
		t.Fatalf("expected no step1 provider reruns during resume, got %d", resumeRunner.blockedCallCount())
	}

	info, ok := resumedService.GetRun(resumeRunID)
	if !ok {
		t.Fatalf("expected resumed run in registry")
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded resumed run, got %s (%s)", info.Status, info.Error)
	}
	if info.RunID != resumeRunID {
		t.Fatalf("expected same run_id after resume, got %s", info.RunID)
	}

	finalSummary := readSingleShardSummary(t, filepath.Join(targetWS.Path, "reports", "taskruns", resumeRunID+"-refresh-step1-collect-shard-summary-*.json"))
	for _, item := range finalSummary.Items {
		if item.Status != "succeeded" {
			t.Fatalf("expected all resumed step1 shard statuses succeeded, got %+v", finalSummary.Items)
		}
	}

	sharedEntity := mustReadFile(t, filepath.Join(targetWS.Path, "model/entities/svc.shared.yaml"))
	if !strings.Contains(sharedEntity, "name: Shared services/web") {
		t.Fatalf("expected checkpointed shard apply to land during resume, got:\n%s", sharedEntity)
	}

	page, found, err := resumedService.GetRunLogs(resumeRunID, 0, 500)
	if err != nil {
		t.Fatalf("read resumed run logs: %v", err)
	}
	if !found {
		t.Fatalf("expected resumed run logs")
	}
	assertRunLogMessage(t, page.Items, "run resumed after restart")
	assertRunLogMessage(t, page.Items, "shard loaded from persisted taskrun")
	assertRunLogMessage(t, page.Items, "shard replayed from checkpoint")
}

func TestAutoResumeRefreshStep3UsesPersistedTaskrunsWithoutProviderRerun(t *testing.T) {
	t.Parallel()

	sourceWS := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	sourceService := NewService(WithRunner(deterministicApplyOrderRunner{}))
	if info, _, err := sourceService.Run(context.Background(), RunRequest{
		Workspace:      sourceWS,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil || info.Status != RunStatusSucceeded {
		t.Fatalf("prepare source init run failed: status=%s err=%v", info.Status, err)
	}
	sourceRefreshInfo, _, err := sourceService.Run(context.Background(), RunRequest{
		Workspace:      sourceWS,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("prepare source refresh run failed: %v", err)
	}
	if sourceRefreshInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected successful source refresh run, got %s (%s)", sourceRefreshInfo.Status, sourceRefreshInfo.Error)
	}

	targetWS := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	targetService := NewService(WithRunner(deterministicApplyOrderRunner{}))
	if info, _, err := targetService.Run(context.Background(), RunRequest{
		Workspace:      targetWS,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil || info.Status != RunStatusSucceeded {
		t.Fatalf("prepare target init run failed: status=%s err=%v", info.Status, err)
	}

	resumeRunID := "run_refresh_resume_step3"
	step1Summary := copyRefreshShardArtifacts(t, sourceWS, targetWS, sourceRefreshInfo.RunID, resumeRunID, "refresh.step1.collect", func(runtimeShardSummaryEntry) string {
		return "succeeded"
	}, func(item runtimeShardSummaryEntry) bool {
		return true
	})
	if len(step1Summary.Items) == 0 {
		t.Fatalf("expected copied step1 summary items")
	}
	step3Summary := copyRefreshShardArtifacts(t, sourceWS, targetWS, sourceRefreshInfo.RunID, resumeRunID, "refresh.step3.findings", func(item runtimeShardSummaryEntry) string {
		if len(item.PathScopes) > 0 && item.PathScopes[0] == "services/web" {
			return "checkpointed"
		}
		return "succeeded"
	}, func(item runtimeShardSummaryEntry) bool {
		return false
	})
	if len(step3Summary.Items) == 0 {
		t.Fatalf("expected copied step3 summary items")
	}

	historySeed := NewService(
		WithHistoryWorkspace(targetWS),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC)
		}),
	)
	historySeed.storeRun(runRecord{
		info: RunInfo{
			RunID:       resumeRunID,
			Pipeline:    string(PipelineRefresh),
			Status:      RunStatusRunning,
			StartedAt:   time.Date(2026, 4, 16, 10, 15, 0, 0, time.UTC),
			CurrentStep: "refresh.step3.findings",
		},
	})

	resumeRunner := &guardedResumeRunner{
		blockedSteps: map[string]struct{}{
			"refresh.step1.collect":   {},
			"refresh.step2.asis_docs": {},
			"refresh.step3.findings":  {},
		},
	}
	resumedService := NewService(
		WithHistoryWorkspace(targetWS),
		WithResumeStaleAsyncRuns(),
		WithRunner(resumeRunner),
	)
	status := waitForTerminalRunStatus(t, targetWS, resumeRunID, 10*time.Second)
	if status != RunStatusSucceeded {
		info, _ := resumedService.GetRun(resumeRunID)
		t.Fatalf("expected resumed run to succeed, got status=%s error_code=%s error=%s", status, info.ErrorCode, info.Error)
	}
	if resumeRunner.blockedCallCount() != 0 {
		t.Fatalf("expected no step1/2/3 provider reruns during step3 resume, got %d", resumeRunner.blockedCallCount())
	}

	finalSummary := readSingleShardSummary(t, filepath.Join(targetWS.Path, "reports", "taskruns", resumeRunID+"-refresh-step3-findings-shard-summary.json"))
	for _, item := range finalSummary.Items {
		if item.Status != "succeeded" {
			t.Fatalf("expected all resumed step3 shard statuses succeeded, got %+v", finalSummary.Items)
		}
	}

	findingsReport := mustReadFile(t, filepath.Join(targetWS.Path, "reports/findings/findings.md"))
	if !strings.Contains(findingsReport, "# Findings") {
		t.Fatalf("expected findings report after resumed step3, got:\n%s", findingsReport)
	}

	page, found, err := resumedService.GetRunLogs(resumeRunID, 0, 500)
	if err != nil {
		t.Fatalf("read resumed run logs: %v", err)
	}
	if !found {
		t.Fatalf("expected resumed run logs")
	}
	assertRunLogMessage(t, page.Items, "run resumed after restart")
	assertRunLogMessage(t, page.Items, "shard loaded from persisted taskrun")
	assertRunLogMessage(t, page.Items, "shard replayed from checkpoint")
}

func TestShardSummaryStateMachineSupportsPendingCheckpointedSucceededFailed(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})

	execution := pipelineExecution{
		runID:         "run_summary_state_machine",
		workspace:     ws,
		clock:         func() time.Time { return time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC) },
		artifacts:     []Artifact{},
		artifactIndex: map[string]int{},
		executionProfile: acpruntime.ExecutionValues{
			Strategy:      acpruntime.ExecutionStrategyParallel,
			MaxParallel:   2,
			FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
			ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
		},
	}
	plans := []runtimeShardPlan{
		{SortKey: "1", ShardID: "shard-a", RepoScopes: []string{"orders-monolith"}, PathScopes: []string{"services/a"}, PrimaryRepo: "orders-monolith"},
		{SortKey: "2", ShardID: "shard-b", RepoScopes: []string{"orders-monolith"}, PathScopes: []string{"services/b"}, PrimaryRepo: "orders-monolith"},
		{SortKey: "3", ShardID: "shard-c", RepoScopes: []string{"orders-monolith"}, PathScopes: []string{"services/c"}, PrimaryRepo: "orders-monolith"},
		{SortKey: "4", ShardID: "shard-d", RepoScopes: []string{"orders-monolith"}, PathScopes: []string{"services/d"}, PrimaryRepo: "orders-monolith"},
	}
	state, err := execution.loadRuntimeShardSummaryState("refresh.step1.collect", "payments-service", plans, false)
	if err != nil {
		t.Fatalf("init shard summary state: %v", err)
	}

	rawB := mustBuildShardTaskrunRaw(t, execution.runID, "refresh.step1.collect", "shard-b", "task-b", []string{"orders-monolith"}, []string{"services/b"})
	pathB := shardTaskrunPath(execution.runID, "refresh.step1.collect", "payments-service", "shard-b", false)
	if err := state.markCheckpointed(plans[1], "task-b", pathB, shardTaskrunLabel("refresh.step1.collect", "payments-service", "shard-b", false), rawB); err != nil {
		t.Fatalf("mark shard-b checkpointed: %v", err)
	}

	rawC := mustBuildShardTaskrunRaw(t, execution.runID, "refresh.step1.collect", "shard-c", "task-c", []string{"orders-monolith"}, []string{"services/c"})
	pathC := shardTaskrunPath(execution.runID, "refresh.step1.collect", "payments-service", "shard-c", false)
	if err := state.markCheckpointed(plans[2], "task-c", pathC, shardTaskrunLabel("refresh.step1.collect", "payments-service", "shard-c", false), rawC); err != nil {
		t.Fatalf("mark shard-c checkpointed: %v", err)
	}
	if err := state.markSucceeded(plans[2], "task-c", pathC); err != nil {
		t.Fatalf("mark shard-c succeeded: %v", err)
	}
	if err := state.markFailed(plans[3], "task-d", "synthetic apply failure"); err != nil {
		t.Fatalf("mark shard-d failed: %v", err)
	}

	summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", execution.runID+"-refresh-step1-collect-shard-summary-payments-service.json"))
	statusByShard := map[string]string{}
	for _, item := range summary.Items {
		statusByShard[item.ShardID] = item.Status
		if (item.Status == "checkpointed" || item.Status == "succeeded") && strings.TrimSpace(item.TaskRun) == "" {
			t.Fatalf("expected taskrun_path for status=%s shard=%s", item.Status, item.ShardID)
		}
	}
	expected := map[string]string{
		"shard-a": "pending",
		"shard-b": "checkpointed",
		"shard-c": "succeeded",
		"shard-d": "failed",
	}
	if !reflect.DeepEqual(statusByShard, expected) {
		t.Fatalf("unexpected shard statuses: got=%v want=%v", statusByShard, expected)
	}

	var persisted contracts.TaskResult
	readJSONFile(t, filepath.Join(ws.Path, pathB), &persisted)
	if persisted.Meta.ShardID != "shard-b" {
		t.Fatalf("expected persisted shard id shard-b, got %q", persisted.Meta.ShardID)
	}
	if len(persisted.Meta.RepoScopes) != 1 || persisted.Meta.RepoScopes[0] != "orders-monolith" {
		t.Fatalf("expected persisted repo scopes in taskrun, got %+v", persisted.Meta.RepoScopes)
	}
	if len(persisted.Meta.PathScopes) != 1 || persisted.Meta.PathScopes[0] != "services/b" {
		t.Fatalf("expected persisted path scopes in taskrun, got %+v", persisted.Meta.PathScopes)
	}
}

func TestResumeDeterminismMatchesUninterruptedRun(t *testing.T) {
	t.Parallel()

	sourceWS := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	sourceService := NewService(WithRunner(deterministicApplyOrderRunner{}))
	if info, _, err := sourceService.Run(context.Background(), RunRequest{
		Workspace:      sourceWS,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil || info.Status != RunStatusSucceeded {
		t.Fatalf("source init run failed: status=%s err=%v", info.Status, err)
	}
	sourceRefreshInfo, _, err := sourceService.Run(context.Background(), RunRequest{
		Workspace:      sourceWS,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("source refresh run failed: %v", err)
	}
	if sourceRefreshInfo.Status != RunStatusSucceeded {
		t.Fatalf("expected source refresh success, got %s (%s)", sourceRefreshInfo.Status, sourceRefreshInfo.Error)
	}

	targetWS := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   2,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	targetService := NewService(WithRunner(deterministicApplyOrderRunner{}))
	if info, _, err := targetService.Run(context.Background(), RunRequest{
		Workspace:      targetWS,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil || info.Status != RunStatusSucceeded {
		t.Fatalf("target init run failed: status=%s err=%v", info.Status, err)
	}

	resumeRunID := "run_refresh_resume_deterministic"
	copyRefreshShardArtifacts(t, sourceWS, targetWS, sourceRefreshInfo.RunID, resumeRunID, "refresh.step1.collect", func(item runtimeShardSummaryEntry) string {
		if len(item.PathScopes) > 0 && item.PathScopes[0] == "services/web" {
			return "checkpointed"
		}
		return "succeeded"
	}, func(item runtimeShardSummaryEntry) bool {
		return len(item.PathScopes) > 0 && item.PathScopes[0] == "services/api"
	})

	historySeed := NewService(
		WithHistoryWorkspace(targetWS),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 16, 13, 0, 0, 0, time.UTC)
		}),
	)
	historySeed.storeRun(runRecord{
		info: RunInfo{
			RunID:       resumeRunID,
			Pipeline:    string(PipelineRefresh),
			Status:      RunStatusRunning,
			StartedAt:   time.Date(2026, 4, 16, 12, 45, 0, 0, time.UTC),
			CurrentStep: "refresh.step3.findings",
		},
	})

	resumeRunner := &guardedResumeRunner{
		blockedSteps: map[string]struct{}{
			"refresh.step1.collect": {},
		},
	}
	resumedService := NewService(
		WithHistoryWorkspace(targetWS),
		WithResumeStaleAsyncRuns(),
		WithRunner(resumeRunner),
	)
	status := waitForTerminalRunStatus(t, targetWS, resumeRunID, 10*time.Second)
	if status != RunStatusSucceeded {
		info, _ := resumedService.GetRun(resumeRunID)
		t.Fatalf("expected resumed deterministic run success, got status=%s error=%s", status, info.Error)
	}
	if resumeRunner.blockedCallCount() != 0 {
		t.Fatalf("expected no refresh.step1.collect provider reruns, got %d", resumeRunner.blockedCallCount())
	}

	assertGlobFilesEqual(t, sourceWS, targetWS, "model/entities/*.yaml")
	assertGlobFilesEqual(t, sourceWS, targetWS, "model/edges/*.yaml")
	assertGlobFilesEqual(t, sourceWS, targetWS, "reports/agent-outputs/domains/*.md")
	assertRelativeFileEqual(t, sourceWS, targetWS, "reports/agent-outputs/architect/summary.md")
	assertRelativeFileEqual(t, sourceWS, targetWS, "reports/coverage/summary.md")
	assertRelativeFileEqual(t, sourceWS, targetWS, "reports/coverage/open-questions.md")
	assertRelativeFileEqual(t, sourceWS, targetWS, "reports/findings/findings.md")
	assertRelativeFileEqual(t, sourceWS, targetWS, "reports/as-is/service-catalog.md")
	assertRunQualitySummaryEqual(t, sourceWS, sourceRefreshInfo.RunID, targetWS, resumeRunID)
}

func TestPersistShardSummaryRejectsCheckpointedWithoutTaskrunPath(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{})
	execution := pipelineExecution{
		runID:         "run_invalid_shard_summary",
		workspace:     ws,
		clock:         func() time.Time { return time.Date(2026, 4, 16, 15, 0, 0, 0, time.UTC) },
		artifacts:     []Artifact{},
		artifactIndex: map[string]int{},
		executionProfile: acpruntime.ExecutionValues{
			Strategy:      acpruntime.ExecutionStrategySequential,
			MaxParallel:   1,
			FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
			ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
		},
	}

	err := execution.persistShardSummary("refresh.step1.collect", "payments-service", []runtimeShardSummaryEntry{
		{
			ShardID: "shard-x",
			Status:  "checkpointed",
		},
	})
	if err == nil {
		t.Fatalf("expected validation error for checkpointed shard without taskrun_path")
	}
	if !strings.Contains(err.Error(), "requires taskrun_path") {
		t.Fatalf("expected taskrun_path validation error, got %v", err)
	}
}

func TestResumeReplayLoadFailureMarksShardFailed(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategySequential,
		MaxParallel:   1,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	runner := &failOnRunRunner{}
	execution := pipelineExecution{
		runID:          "run_resume_missing_taskrun",
		workspace:      ws,
		runnerResolver: newStepRunnerResolver(stepRunnerFactoryFunc(func(acpruntime.Provider) (acpruntime.Runner, error) { return runner, nil }), acpruntime.StepProviderValues{acpruntime.StepProviderStep1Collect: acpruntime.ProviderClaudeCode}),
		stepProviders:  acpruntime.StepProviderValues{acpruntime.StepProviderStep1Collect: acpruntime.ProviderClaudeCode},
		clock:          func() time.Time { return time.Date(2026, 4, 16, 16, 0, 0, 0, time.UTC) },
		artifacts:      []Artifact{},
		artifactIndex:  map[string]int{},
		executionProfile: acpruntime.ExecutionValues{
			Strategy:      acpruntime.ExecutionStrategySequential,
			MaxParallel:   1,
			FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
			ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
		},
	}

	stepID := "refresh.step1.collect"
	domainID := "payments-service"
	repoScopes := []string{"unknown-repo"}
	plans, _, _ := execution.planRuntimeShards(repoScopes)
	if len(plans) != 1 {
		t.Fatalf("expected single fallback shard plan, got %d", len(plans))
	}

	state, err := execution.loadRuntimeShardSummaryState(stepID, domainID, plans, true)
	if err != nil {
		t.Fatalf("init shard summary state: %v", err)
	}
	taskrunPath := shardTaskrunPath(execution.runID, stepID, domainID, plans[0].ShardID, true)
	raw := mustBuildShardTaskrunRaw(
		t,
		execution.runID,
		stepID,
		plans[0].ShardID,
		"task-missing-persisted",
		repoScopes,
		plans[0].PathScopes,
	)
	if err := state.markCheckpointed(
		plans[0],
		"task-missing-persisted",
		taskrunPath,
		shardTaskrunLabel(stepID, domainID, plans[0].ShardID, true),
		raw,
	); err != nil {
		t.Fatalf("mark checkpointed: %v", err)
	}
	if err := os.Remove(filepath.Join(ws.Path, taskrunPath)); err != nil {
		t.Fatalf("remove persisted taskrun %s: %v", taskrunPath, err)
	}

	executions, outcome, err := execution.executeRuntimeTasksSharded(
		context.Background(),
		stepID,
		domainID,
		repoScopes,
		"resume-missing-taskrun",
	)
	if err != nil {
		t.Fatalf("execute sharded resume with missing taskrun: %v", err)
	}
	if len(executions) != 0 {
		t.Fatalf("expected zero executions, got %d", len(executions))
	}
	if outcome.PlannedShards != 1 || outcome.FailedShards != 1 || outcome.SucceededShards != 0 {
		t.Fatalf("unexpected shard outcome: %+v", outcome)
	}
	if runner.callCount() != 0 {
		t.Fatalf("expected no runtime provider rerun, got %d calls", runner.callCount())
	}

	summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", execution.runID+"-refresh-step1-collect-shard-summary-*.json"))
	if len(summary.Items) != 1 {
		t.Fatalf("expected one shard summary item, got %d", len(summary.Items))
	}
	item := summary.Items[0]
	if item.Status != "failed" {
		t.Fatalf("expected failed shard status after replay load error, got %+v", item)
	}
	if item.TaskRun != taskrunPath {
		t.Fatalf("expected shard summary taskrun_path preserved as %q, got %q", taskrunPath, item.TaskRun)
	}
	if !strings.Contains(item.Error, "load persisted taskrun") {
		t.Fatalf("expected replay load error in shard summary, got %q", item.Error)
	}
}

func TestResumeFailFastStopsBeforePendingShardAfterPersistedFailure(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategySequential,
		MaxParallel:   1,
		FailurePolicy: acpruntime.ExecutionFailurePolicyFailFast,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	runner := &failOnRunRunner{}
	execution := pipelineExecution{
		runID:          "run_resume_failfast_stops_pending",
		workspace:      ws,
		runnerResolver: newStepRunnerResolver(stepRunnerFactoryFunc(func(acpruntime.Provider) (acpruntime.Runner, error) { return runner, nil }), acpruntime.StepProviderValues{acpruntime.StepProviderStep1Collect: acpruntime.ProviderClaudeCode}),
		stepProviders:  acpruntime.StepProviderValues{acpruntime.StepProviderStep1Collect: acpruntime.ProviderClaudeCode},
		clock:          func() time.Time { return time.Date(2026, 4, 16, 17, 0, 0, 0, time.UTC) },
		artifacts:      []Artifact{},
		artifactIndex:  map[string]int{},
		executionProfile: acpruntime.ExecutionValues{
			Strategy:      acpruntime.ExecutionStrategySequential,
			MaxParallel:   1,
			FailurePolicy: acpruntime.ExecutionFailurePolicyFailFast,
			ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
		},
	}

	stepID := "refresh.step1.collect"
	domainID := "payments-service"
	repoScopes := []string{"orders-monolith"}
	plans, _, _ := execution.planRuntimeShards(repoScopes)
	if len(plans) < 2 {
		t.Fatalf("expected at least two shard plans for fail-fast resume scenario, got %d", len(plans))
	}

	state, err := execution.loadRuntimeShardSummaryState(stepID, domainID, plans, false)
	if err != nil {
		t.Fatalf("init shard summary state: %v", err)
	}
	if err := state.markFailed(plans[0], "task-persisted-failed", "persisted shard failure"); err != nil {
		t.Fatalf("mark persisted failed shard: %v", err)
	}

	executions, outcome, err := execution.executeRuntimeTasksSharded(
		context.Background(),
		stepID,
		domainID,
		repoScopes,
		"resume-failfast-stops-pending",
	)
	if err == nil {
		t.Fatalf("expected fail-fast terminal error from persisted failed shard")
	}
	if len(executions) != 0 {
		t.Fatalf("expected zero successful executions, got %d", len(executions))
	}
	if runner.callCount() != 0 {
		t.Fatalf("expected no provider reruns after persisted fail-fast shard, got %d", runner.callCount())
	}
	if outcome.PlannedShards != len(plans) {
		t.Fatalf("unexpected planned shard count: got=%d want=%d", outcome.PlannedShards, len(plans))
	}
	if outcome.FailedShards != len(plans) {
		t.Fatalf("expected all shards reported failed/aborted under fail-fast, got %+v", outcome)
	}

	summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", execution.runID+"-refresh-step1-collect-shard-summary-*.json"))
	if len(summary.Items) != len(plans) {
		t.Fatalf("expected %d shard summary items, got %d", len(plans), len(summary.Items))
	}
	var sawPersistedFailure bool
	var sawFailFastAbort bool
	for _, item := range summary.Items {
		if item.Status != "failed" {
			t.Fatalf("expected failed status for all shards after fail-fast resume, got %+v", summary.Items)
		}
		if strings.Contains(item.Error, "persisted shard failure") {
			sawPersistedFailure = true
		}
		if strings.Contains(item.Error, "fail_fast aborted remaining work") {
			sawFailFastAbort = true
		}
	}
	if !sawPersistedFailure {
		t.Fatalf("expected persisted failure reason preserved in shard summary, got %+v", summary.Items)
	}
	if !sawFailFastAbort {
		t.Fatalf("expected fail-fast abort reason for pending shards, got %+v", summary.Items)
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
	expected := []int{3, 4}
	if !reflect.DeepEqual(summarySizes, expected) {
		t.Fatalf("unexpected per-domain shard counts: got=%v want=%v", summarySizes, expected)
	}

	step3Summary := readSingleShardSummary(t, filepath.Join(ws.Path, "reports", "taskruns", "*-init-step3-findings-shard-summary.json"))
	if len(step3Summary.Items) != 7 {
		t.Fatalf("expected seven step3 shards ((2+1)+(3+1)), got %d", len(step3Summary.Items))
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

type allStep1ShardFailureRunner struct{}

func (allStep1ShardFailureRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if strings.HasSuffix(task.StepID, "step1.collect") {
		return acpruntime.Result{}, acpruntime.WrapRunnerError(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			"synthetic shard parse failure on all collect shards",
			errors.New("synthetic shard parse failure"),
		)
	}
	return claudecode.FakeRunner{}.Run(ctx, task)
}

func (allStep1ShardFailureRunner) Preflight(context.Context) error {
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

type failOnRunRunner struct {
	mu    sync.Mutex
	calls int
}

func (r *failOnRunRunner) Run(context.Context, acpruntime.Task) (acpruntime.Result, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return acpruntime.Result{}, errors.New("unexpected provider invocation during resume")
}

func (r *failOnRunRunner) Preflight(context.Context) error {
	return nil
}

func (r *failOnRunRunner) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

type guardedResumeRunner struct {
	mu           sync.Mutex
	blockedSteps map[string]struct{}
	totalCalls   int
	blockedCalls int
}

func (r *guardedResumeRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	r.mu.Lock()
	r.totalCalls++
	_, blocked := r.blockedSteps[task.StepID]
	if blocked {
		r.blockedCalls++
	}
	r.mu.Unlock()
	if blocked {
		return acpruntime.Result{}, errors.New("unexpected provider invocation during resume")
	}
	return claudecode.FakeRunner{}.Run(ctx, task)
}

func (r *guardedResumeRunner) Preflight(context.Context) error {
	return nil
}

func (r *guardedResumeRunner) blockedCallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.blockedCalls
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

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func copyRefreshShardArtifacts(
	t *testing.T,
	sourceWS workspace.Root,
	targetWS workspace.Root,
	sourceRunID string,
	resumeRunID string,
	stepID string,
	statusForItem func(runtimeShardSummaryEntry) string,
	applyToTargetModel func(runtimeShardSummaryEntry) bool,
) runtimeShardSummary {
	t.Helper()

	stepSlug := strings.ReplaceAll(stepID, ".", "-")
	sourceSummaryPath := mustGlobSingle(t, filepath.Join(sourceWS.Path, "reports", "taskruns", sourceRunID+"-"+stepSlug+"-shard-summary*.json"))
	var summary runtimeShardSummary
	readJSONFile(t, sourceSummaryPath, &summary)
	summary.RunID = resumeRunID
	singleShard := len(summary.Items) == 1

	for idx, item := range summary.Items {
		raw := mustReadTaskrunFile(t, sourceWS, item.TaskRun)
		prepared, err := loadPreparedExecutionFromPersistedTaskRun(raw)
		if err != nil {
			t.Fatalf("load persisted taskrun %s: %v", item.TaskRun, err)
		}
		prepared.Task.RunID = resumeRunID
		prepared.Normalized.Meta.RunID = resumeRunID
		rewrittenRaw, err := json.MarshalIndent(prepared.Normalized, "", "  ")
		if err != nil {
			t.Fatalf("marshal rewritten taskrun %s: %v", item.TaskRun, err)
		}
		newTaskrunPath := shardTaskrunPath(resumeRunID, stepID, summary.DomainID, prepared.Task.ShardID, singleShard)
		writeFile(t, filepath.Join(targetWS.Path, newTaskrunPath), string(append(rewrittenRaw, '\n')))
		summary.Items[idx].TaskRun = newTaskrunPath
		summary.Items[idx].TaskID = prepared.Task.TaskID
		summary.Items[idx].Status = statusForItem(item)
		if applyToTargetModel(item) {
			result, parseErr := contracts.ParseTaskResult(rewrittenRaw)
			if parseErr != nil {
				t.Fatalf("parse rewritten taskrun %s: %v", newTaskrunPath, parseErr)
			}
			if _, applyErr := model.NewStore(targetWS).ApplyChangeset(result); applyErr != nil {
				t.Fatalf("apply rewritten taskrun %s to target model: %v", newTaskrunPath, applyErr)
			}
		}
	}

	sourcePlanPath := mustGlobSingle(t, filepath.Join(sourceWS.Path, "reports", "taskruns", sourceRunID+"-"+stepSlug+"-shard-plan*.json"))
	var plan runtimeShardPlanArtifact
	readJSONFile(t, sourcePlanPath, &plan)
	plan.RunID = resumeRunID
	planRaw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatalf("marshal rewritten shard plan: %v", err)
	}
	writeFile(t, filepath.Join(targetWS.Path, shardPlanPath(resumeRunID, stepID, summary.DomainID)), string(append(planRaw, '\n')))

	summaryRaw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatalf("marshal rewritten shard summary: %v", err)
	}
	writeFile(t, filepath.Join(targetWS.Path, shardSummaryPath(resumeRunID, stepID, summary.DomainID)), string(append(summaryRaw, '\n')))
	return summary
}

func mustReadTaskrunFile(t *testing.T, ws workspace.Root, relPath string) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(ws.Path, relPath))
	if err != nil {
		t.Fatalf("read taskrun %s: %v", relPath, err)
	}
	return content
}

func mustGlobSingle(t *testing.T, pattern string) string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob %q: %v", pattern, err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one match for %q, got %d (%v)", pattern, len(matches), matches)
	}
	return matches[0]
}

func assertRunLogMessage(t *testing.T, entries []RunLogEntry, message string) {
	t.Helper()
	for _, entry := range entries {
		if entry.Message == message {
			return
		}
	}
	t.Fatalf("expected run log message %q, got %+v", message, entries)
}

func mustBuildShardTaskrunRaw(
	t *testing.T,
	runID string,
	stepID string,
	shardID string,
	taskID string,
	repoScopes []string,
	pathScopes []string,
) []byte {
	t.Helper()
	repoScope := ""
	if len(repoScopes) > 0 {
		repoScope = repoScopes[0]
	}
	result := contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:     taskID,
			StepID:     stepID,
			RunID:      runID,
			Runtime:    contracts.RuntimeMeta{Name: "synthetic-shard-runner", Version: "test"},
			StartedAt:  time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
			FinishedAt: time.Date(2026, 4, 16, 12, 0, 1, 0, time.UTC).Format(time.RFC3339),
			Workspace:  "/tmp/workspace",
			ShardID:    shardID,
			RepoScope:  repoScope,
			RepoScopes: append([]string(nil), repoScopes...),
			PathScopes: append([]string(nil), pathScopes...),
		},
		Summary: "synthetic checkpoint payload",
		Changeset: []contracts.Operation{
			{
				Op: "upsert_entity",
				Entity: &contracts.Entity{
					ID:   "svc." + shardID,
					Type: "service",
					Name: "Synthetic " + shardID,
					Provenance: contracts.Provenance{
						Kind:       "observation",
						Confidence: 0.9,
						Evidence: []contracts.Evidence{
							{
								Repo: repoScope,
								Path: "README.md",
							},
						},
					},
				},
			},
		},
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("marshal synthetic shard taskrun: %v", err)
	}
	return raw
}

func assertRelativeFileEqual(t *testing.T, left workspace.Root, right workspace.Root, relPath string) {
	t.Helper()
	leftContent := mustReadFile(t, filepath.Join(left.Path, relPath))
	rightContent := mustReadFile(t, filepath.Join(right.Path, relPath))
	if leftContent != rightContent {
		t.Fatalf("content mismatch for %s", relPath)
	}
}

func assertGlobFilesEqual(t *testing.T, left workspace.Root, right workspace.Root, relPattern string) {
	t.Helper()

	leftMatches := mustRelativeGlobMatches(t, left.Path, relPattern)
	rightMatches := mustRelativeGlobMatches(t, right.Path, relPattern)
	if !reflect.DeepEqual(leftMatches, rightMatches) {
		t.Fatalf("glob mismatch for pattern %q: left=%v right=%v", relPattern, leftMatches, rightMatches)
	}
	for _, relPath := range leftMatches {
		assertRelativeFileEqual(t, left, right, relPath)
	}
}

func mustRelativeGlobMatches(t *testing.T, root string, relPattern string) []string {
	t.Helper()
	absPattern := filepath.Join(root, relPattern)
	matches, err := filepath.Glob(absPattern)
	if err != nil {
		t.Fatalf("glob %q: %v", relPattern, err)
	}
	relMatches := make([]string, 0, len(matches))
	for _, absPath := range matches {
		relPath, relErr := filepath.Rel(root, absPath)
		if relErr != nil {
			t.Fatalf("relative path for %q: %v", absPath, relErr)
		}
		relMatches = append(relMatches, filepath.ToSlash(relPath))
	}
	sort.Strings(relMatches)
	return relMatches
}

func waitForTerminalRunStatus(t *testing.T, ws workspace.Root, runID string, timeout time.Duration) RunStatus {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		snapshot, err := loadRunHistorySnapshot(ws)
		if err == nil {
			for _, item := range snapshot.Items {
				if item.RunID != runID {
					continue
				}
				if item.Status == RunStatusSucceeded || item.Status == RunStatusFailed {
					return item.Status
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("run %q did not reach terminal status before timeout %s", runID, timeout)
	return ""
}

func readRunQualitySummary(t *testing.T, ws workspace.Root, runID string) struct {
	EvidenceState struct {
		Collect struct {
			Status          string `json:"status"`
			PlannedShards   int    `json:"planned_shards"`
			SucceededShards int    `json:"succeeded_shards"`
			FailedShards    int    `json:"failed_shards"`
		} `json:"collect"`
		Findings struct {
			Status          string `json:"status"`
			PlannedShards   int    `json:"planned_shards"`
			SucceededShards int    `json:"succeeded_shards"`
			FailedShards    int    `json:"failed_shards"`
		} `json:"findings"`
		ReportMode string   `json:"report_mode"`
		Reasons    []string `json:"reasons"`
	} `json:"evidence_state"`
} {
	t.Helper()
	var summary struct {
		EvidenceState struct {
			Collect struct {
				Status          string `json:"status"`
				PlannedShards   int    `json:"planned_shards"`
				SucceededShards int    `json:"succeeded_shards"`
				FailedShards    int    `json:"failed_shards"`
			} `json:"collect"`
			Findings struct {
				Status          string `json:"status"`
				PlannedShards   int    `json:"planned_shards"`
				SucceededShards int    `json:"succeeded_shards"`
				FailedShards    int    `json:"failed_shards"`
			} `json:"findings"`
			ReportMode string   `json:"report_mode"`
			Reasons    []string `json:"reasons"`
		} `json:"evidence_state"`
	}
	readJSONFile(t, filepath.Join(ws.Path, "reports", "taskruns", runID+"-quality.json"), &summary)
	return summary
}

func assertRunQualitySummaryEqual(
	t *testing.T,
	left workspace.Root,
	leftRunID string,
	right workspace.Root,
	rightRunID string,
) {
	t.Helper()

	readSanitized := func(ws workspace.Root, runID string) map[string]any {
		var payload map[string]any
		readJSONFile(t, filepath.Join(ws.Path, "reports", "taskruns", runID+"-quality.json"), &payload)
		delete(payload, "run_id")
		delete(payload, "generated_at")
		delete(payload, "run_warnings")
		delete(payload, "totals")
		delete(payload, "steps")
		return payload
	}

	leftPayload := readSanitized(left, leftRunID)
	rightPayload := readSanitized(right, rightRunID)
	if !reflect.DeepEqual(leftPayload, rightPayload) {
		t.Fatalf("quality summary mismatch between uninterrupted and resumed runs")
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
