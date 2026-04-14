package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/model"
	"github.com/GrinRus/ProvenArch/internal/reports"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestDiscoverHeuristicShardPathsDetectsAllServiceMarkers(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	writeFile(t, filepath.Join(repoPath, "svc-go", "go.mod"), "module example.com/svc-go\n")
	writeFile(t, filepath.Join(repoPath, "svc-node", "package.json"), "{\n  \"name\": \"svc-node\"\n}\n")
	writeFile(t, filepath.Join(repoPath, "svc-py", "pyproject.toml"), "[project]\nname = \"svc-py\"\n")
	writeFile(t, filepath.Join(repoPath, "svc-rust", "Cargo.toml"), "[package]\nname = \"svc-rust\"\n")
	writeFile(t, filepath.Join(repoPath, "svc-maven", "pom.xml"), "<project/>\n")
	writeFile(t, filepath.Join(repoPath, "svc-gradle-build", "build.gradle"), "plugins {}\n")
	writeFile(t, filepath.Join(repoPath, "svc-gradle-settings", "settings.gradle"), "rootProject.name = 'svc-gradle-settings'\n")
	writeFile(t, filepath.Join(repoPath, "svc-bazel", "module.bazel"), "module(name = \"svc-bazel\")\n")
	writeFile(t, filepath.Join(repoPath, "svc-workspace-upper", "WORKSPACE"), "workspace(name = \"svc-workspace-upper\")\n")
	writeFile(t, filepath.Join(repoPath, "svc-workspace-lower", "workspace"), "workspace(name = \"svc-workspace-lower\")\n")

	paths, err := discoverHeuristicShardPaths(repoPath)
	if err != nil {
		t.Fatalf("discover heuristic shard paths: %v", err)
	}

	expected := []string{
		"svc-bazel",
		"svc-go",
		"svc-gradle-build",
		"svc-gradle-settings",
		"svc-maven",
		"svc-node",
		"svc-py",
		"svc-rust",
		"svc-workspace-lower",
		"svc-workspace-upper",
	}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("unexpected detected marker roots: got=%v want=%v", paths, expected)
	}
}

func TestDiscoverHeuristicShardPathsFallbacksToRootWhenNoMarkers(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	writeFile(t, filepath.Join(repoPath, "README.md"), "# no markers\n")

	paths, err := discoverHeuristicShardPaths(repoPath)
	if err != nil {
		t.Fatalf("discover heuristic shard paths: %v", err)
	}
	if !reflect.DeepEqual(paths, []string{"."}) {
		t.Fatalf("expected root fallback, got %v", paths)
	}
}

func TestChunkServiceDeterministicAndRespectsThresholds(t *testing.T) {
	t.Parallel()

	files := make([]serviceSourceFile, 0, 501)
	for idx := 0; idx < 501; idx++ {
		files = append(files, serviceSourceFile{
			Path:  fmt.Sprintf("services/payments/file-%04d.go", idx),
			Bytes: 16,
		})
	}

	first := chunkService("payments-repo", "svc.payments-services-payments", "services/payments", files, sumServiceSourceBytes(files))
	second := chunkService("payments-repo", "svc.payments-services-payments", "services/payments", files, sumServiceSourceBytes(files))
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("chunking must be deterministic:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if len(first) <= 1 {
		t.Fatalf("expected multi-chunk service for 501 source files, got %d chunks", len(first))
	}
	if len(first) > serviceChunkMaxCount {
		t.Fatalf("chunk count exceeds cap: got=%d cap=%d", len(first), serviceChunkMaxCount)
	}
	for idx, shard := range first {
		expectedID := fmt.Sprintf("svc.payments-services-payments-s%d", idx+1)
		if shard.ShardID != expectedID {
			t.Fatalf("unexpected shard id at index %d: got=%q want=%q", idx, shard.ShardID, expectedID)
		}
		if idx < len(first)-1 {
			if shard.FileCount > serviceChunkMaxFiles {
				t.Fatalf("non-last shard file cap exceeded: shard=%+v cap=%d", shard, serviceChunkMaxFiles)
			}
			if shard.SourceBytes > serviceChunkMaxBytes {
				t.Fatalf("non-last shard size cap exceeded: shard=%+v cap=%d", shard, serviceChunkMaxBytes)
			}
		}
	}
}

func TestChunkServiceAppliesMaxChunkCap(t *testing.T) {
	t.Parallel()

	files := make([]serviceSourceFile, 0, 3000)
	for idx := 0; idx < 3000; idx++ {
		files = append(files, serviceSourceFile{
			Path:  fmt.Sprintf("services/huge/file-%04d.go", idx),
			Bytes: 80 * 1024,
		})
	}
	chunks := chunkService("huge-repo", "svc.huge-services-huge", "services/huge", files, sumServiceSourceBytes(files))
	if len(chunks) != serviceChunkMaxCount {
		t.Fatalf("expected chunk cap %d, got %d", serviceChunkMaxCount, len(chunks))
	}
	for idx, shard := range chunks {
		if idx < len(chunks)-1 {
			if shard.FileCount > serviceChunkMaxFiles {
				t.Fatalf("chunk file cap exceeded before final shard: idx=%d files=%d", idx, shard.FileCount)
			}
			if shard.SourceBytes > serviceChunkMaxBytes {
				t.Fatalf("chunk byte cap exceeded before final shard: idx=%d bytes=%d", idx, shard.SourceBytes)
			}
		}
	}
}

func TestSelectIncrementalShardsChoosesChangedNewRemovedAndRepoFallback(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for incremental shard selector test")
	}

	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	initGitRepoForServiceFirstTest(t, repoA)
	initGitRepoForServiceFirstTest(t, repoB)

	writeFile(t, filepath.Join(repoA, "services", "changed", "main.go"), "package changed\n")
	writeFile(t, filepath.Join(repoA, "services", "unchanged", "main.go"), "package unchanged\n")
	writeFile(t, filepath.Join(repoA, "services", "removed", "main.go"), "package removed\n")
	runGitForServiceFirstTest(t, repoA, "add", ".")
	runGitForServiceFirstTest(t, repoA, "commit", "-m", "initial")
	prevHeadA := runGitForServiceFirstTest(t, repoA, "rev-parse", "HEAD")

	writeFile(t, filepath.Join(repoA, "services", "changed", "main.go"), "package changed\n// modified\n")
	writeFile(t, filepath.Join(repoA, "services", "new", "main.go"), "package new\n")

	writeFile(t, filepath.Join(repoB, "services", "stable", "main.go"), "package stable\n")
	runGitForServiceFirstTest(t, repoB, "add", ".")
	runGitForServiceFirstTest(t, repoB, "commit", "-m", "initial")
	prevHeadB := runGitForServiceFirstTest(t, repoB, "rev-parse", "HEAD")

	execution := pipelineExecution{
		workspace: workspace.Root{
			Path: root,
			Manifest: workspace.Manifest{
				Version: 1,
				Repos: []workspace.RepoSource{
					{Name: "repo-a", Path: repoA},
					{Name: "repo-b", Path: repoB},
				},
			},
		},
		resolvedRepoPaths: map[string]string{
			"repo-a": repoA,
			"repo-b": filepath.Join(root, "repo-b-missing"),
		},
	}

	services := []serviceInventoryService{
		newInventoryServiceForTest("repo-a", "svc.repo-a-services-changed", "services/changed"),
		newInventoryServiceForTest("repo-a", "svc.repo-a-services-unchanged", "services/unchanged"),
		newInventoryServiceForTest("repo-a", "svc.repo-a-services-new", "services/new"),
		newInventoryServiceForTest("repo-b", "svc.repo-b-services-stable", "services/stable"),
	}
	previous := serviceInventorySnapshot{
		Version: serviceInventoryArtifactVersion,
		RepoHeads: []serviceInventoryRepoHead{
			{RepoScope: "repo-a", Head: prevHeadA},
			{RepoScope: "repo-b", Head: prevHeadB},
		},
		Services: []serviceInventorySnapshotService{
			newSnapshotServiceForTest("repo-a", "svc.repo-a-services-changed", "services/changed"),
			newSnapshotServiceForTest("repo-a", "svc.repo-a-services-unchanged", "services/unchanged"),
			newSnapshotServiceForTest("repo-a", "svc.repo-a-services-removed", "services/removed"),
			newSnapshotServiceForTest("repo-b", "svc.repo-b-services-stable", "services/stable"),
		},
	}

	selected, removed, warnings := execution.selectIncrementalShards(context.Background(), services, previous)

	selectedServices := make([]string, 0, len(selected))
	for _, shard := range selected {
		selectedServices = append(selectedServices, shard.ServiceID)
	}
	sort.Strings(selectedServices)
	expectedSelected := []string{
		"svc.repo-a-services-changed",
		"svc.repo-a-services-new",
		"svc.repo-b-services-stable",
	}
	if !reflect.DeepEqual(selectedServices, expectedSelected) {
		t.Fatalf("unexpected selected services: got=%v want=%v", selectedServices, expectedSelected)
	}

	if len(removed) != 1 || removed[0].ServiceID != "svc.repo-a-services-removed" {
		t.Fatalf("unexpected removed services: %+v", removed)
	}
	if !containsWarning(warnings, `incremental repo "repo-b" fallback to full`) {
		t.Fatalf("expected fallback-to-full warning for repo-b, got %v", warnings)
	}
}

func TestRunRefreshIncrementalFallsBackToFullWhenSnapshotMissing(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   3,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
	})
	service := NewService()

	info, artifacts, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		RefreshMode:    RefreshModeIncremental,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run refresh pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	planPath := ""
	for _, artifact := range artifacts {
		if strings.HasSuffix(artifact.Path, "-service-inventory-plan.json") {
			planPath = artifact.Path
			break
		}
	}
	if strings.TrimSpace(planPath) == "" {
		t.Fatalf("service inventory plan artifact not found: %+v", artifacts)
	}

	var plan struct {
		Mode     string `json:"mode"`
		Services []struct {
			Shards []any `json:"shards"`
		} `json:"services"`
		SelectedShards []any    `json:"selected_shards"`
		Warnings       []string `json:"warnings"`
	}
	readJSONFile(t, filepath.Join(ws.Path, planPath), &plan)
	if plan.Mode != string(RefreshModeIncremental) {
		t.Fatalf("expected incremental mode, got %q", plan.Mode)
	}
	totalShards := 0
	for _, service := range plan.Services {
		totalShards += len(service.Shards)
	}
	if len(plan.SelectedShards) != totalShards {
		t.Fatalf("expected full fallback to select all shards: selected=%d total=%d", len(plan.SelectedShards), totalShards)
	}
	if !containsWarning(plan.Warnings, "incremental refresh fallback to full") {
		t.Fatalf("expected snapshot-missing fallback warning, got %v", plan.Warnings)
	}
}

func TestBuildDomainOutputsFromServiceRunsMappingPolicies(t *testing.T) {
	t.Parallel()

	t.Run("single-domain-per-repo direct map", func(t *testing.T) {
		t.Parallel()

		execution := newDomainMappingExecutionForTest(t, []string{"orders"})
		runs := []serviceRuntimeRun{{
			RepoScope:      "mono",
			ServiceID:      "svc.mono-orders",
			ServiceRoot:    "services/orders",
			ShardID:        "svc.mono-orders-s1",
			PathScopes:     []string{"services/orders"},
			RuntimeSummary: "orders summary",
		}}

		if err := execution.buildDomainOutputsFromServiceRuns("init.step2.service_collect", runs); err != nil {
			t.Fatalf("build domain outputs: %v", err)
		}
		orders := execution.domainRuns["orders"]
		if !strings.Contains(orders.RuntimeSummary, "orders summary") {
			t.Fatalf("expected mapped runtime summary, got %q", orders.RuntimeSummary)
		}
		if hasQuestionIDForServiceFirstTest(execution.questions, "q.domain.global.unresolved-service-mappings") {
			t.Fatalf("did not expect unresolved global mapping question, got %+v", execution.questions)
		}
	})

	t.Run("multi-domain-per-repo token match", func(t *testing.T) {
		t.Parallel()

		execution := newDomainMappingExecutionForTest(t, []string{"orders", "billing"})
		runs := []serviceRuntimeRun{{
			RepoScope:      "mono",
			ServiceID:      "svc.orders-api",
			ServiceRoot:    "services/orders/api",
			ShardID:        "svc.orders-api-s1",
			PathScopes:     []string{"services/orders/api"},
			RuntimeSummary: "orders api summary",
		}}

		if err := execution.buildDomainOutputsFromServiceRuns("init.step2.service_collect", runs); err != nil {
			t.Fatalf("build domain outputs: %v", err)
		}
		if !strings.Contains(execution.domainRuns["orders"].RuntimeSummary, "orders api summary") {
			t.Fatalf("expected token-matched summary in orders domain, got %q", execution.domainRuns["orders"].RuntimeSummary)
		}
		if execution.domainRuns["billing"].RuntimeSummary != "none" {
			t.Fatalf("expected unmatched billing runtime summary=none, got %q", execution.domainRuns["billing"].RuntimeSummary)
		}
		if hasQuestionIDForServiceFirstTest(execution.questions, "q.domain.global.unresolved-service-mappings") {
			t.Fatalf("did not expect unresolved global mapping question, got %+v", execution.questions)
		}
	})

	t.Run("multi-domain-per-repo ambiguous mapping", func(t *testing.T) {
		t.Parallel()

		execution := newDomainMappingExecutionForTest(t, []string{"orders", "billing"})
		runs := []serviceRuntimeRun{{
			RepoScope:      "mono",
			ServiceID:      "svc.core-platform",
			ServiceRoot:    "services/core",
			ShardID:        "svc.core-platform-s1",
			PathScopes:     []string{"services/core"},
			RuntimeSummary: "core summary",
		}}

		if err := execution.buildDomainOutputsFromServiceRuns("init.step2.service_collect", runs); err != nil {
			t.Fatalf("build domain outputs: %v", err)
		}
		if !hasQuestionIDForServiceFirstTest(execution.questions, "q.domain.svc-core-platform.service-mapping-ambiguous") {
			t.Fatalf("expected ambiguous service mapping question, got %+v", execution.questions)
		}
		if !hasQuestionIDForServiceFirstTest(execution.questions, "q.domain.global.unresolved-service-mappings") {
			t.Fatalf("expected unresolved global mapping question, got %+v", execution.questions)
		}
		for _, domainID := range []string{"orders", "billing"} {
			if !containsString(execution.domainRuns[domainID].Unresolved, "service_mapping_ambiguous") {
				t.Fatalf("expected unresolved service_mapping_ambiguous for domain %q, got %+v", domainID, execution.domainRuns[domainID].Unresolved)
			}
		}
	})
}

func TestRunInitExecutesGlobalReviewExactlyOnce(t *testing.T) {
	t.Parallel()

	ws := createShardingWorkspace(t, shardingWorkspaceOptions{
		Strategy:      acpruntime.ExecutionStrategyParallel,
		MaxParallel:   3,
		FailurePolicy: acpruntime.ExecutionFailurePolicyBestEffort,
		ShardMode:     acpruntime.ExecutionShardDiscoveryHeuristics,
	})
	runner := &trackingRunner{}
	service := NewService(WithRunner(runner))

	info, artifacts, err := service.Run(context.Background(), RunRequest{
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

	globalTasks := runner.tasksForStep("init.step5.global_review")
	if len(globalTasks) != 1 {
		t.Fatalf("expected exactly one global-review runtime task, got %d", len(globalTasks))
	}

	globalInputCount := 0
	for _, artifact := range artifacts {
		if strings.HasSuffix(artifact.Path, "-global-review-input.json") {
			globalInputCount++
		}
	}
	if globalInputCount != 1 {
		t.Fatalf("expected exactly one global review input artifact, got %d", globalInputCount)
	}

	summaryPath := filepath.Join(ws.Path, "reports", "agent-outputs", "architect", "summary.md")
	summaryContent, readErr := os.ReadFile(summaryPath)
	if readErr != nil {
		t.Fatalf("read architect summary: %v", readErr)
	}
	if !strings.Contains(string(summaryContent), "Architect Aggregation Summary") {
		t.Fatalf("expected global review summary header, got:\n%s", string(summaryContent))
	}
}

func initGitRepoForServiceFirstTest(t *testing.T, repoPath string) {
	t.Helper()
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo %s: %v", repoPath, err)
	}
	runGitForServiceFirstTest(t, repoPath, "init")
	runGitForServiceFirstTest(t, repoPath, "config", "user.email", "tests@example.com")
	runGitForServiceFirstTest(t, repoPath, "config", "user.name", "ACP Tests")
}

func runGitForServiceFirstTest(t *testing.T, repoPath string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output))
}

func newInventoryServiceForTest(repoScope string, serviceID string, root string) serviceInventoryService {
	shardID := serviceID + "-s1"
	return serviceInventoryService{
		RepoScope:   repoScope,
		ServiceID:   serviceID,
		ServiceRoot: root,
		Shards: []serviceShardPlan{
			{
				RepoScope:   repoScope,
				ServiceID:   serviceID,
				ServiceRoot: root,
				ShardID:     shardID,
				SortKey:     fmt.Sprintf("%s:%s:001", repoScope, serviceID),
				PathScopes:  []string{root},
				FileCount:   1,
				SourceBytes: 1,
			},
		},
	}
}

func newSnapshotServiceForTest(repoScope string, serviceID string, root string) serviceInventorySnapshotService {
	return serviceInventorySnapshotService{
		RepoScope:   repoScope,
		ServiceID:   serviceID,
		ServiceRoot: root,
		Shards: []serviceInventorySnapshotShard{
			{
				ShardID:    serviceID + "-s1",
				PathScopes: []string{root},
				FileCount:  1,
			},
		},
	}
}

func newDomainMappingExecutionForTest(t *testing.T, domainIDs []string) *pipelineExecution {
	t.Helper()

	root := t.TempDir()
	repoPath := filepath.Join(root, "repos", "mono")
	writeFile(t, filepath.Join(repoPath, "services", "orders", "package.json"), "{\n  \"name\": \"orders\"\n}\n")
	writeFile(t, filepath.Join(repoPath, "services", "billing", "package.json"), "{\n  \"name\": \"billing\"\n}\n")

	manifest := workspace.Manifest{
		Version: 1,
		Repos: []workspace.RepoSource{
			{Name: "mono", Path: repoPath},
		},
	}
	manifestRaw, err := workspace.RenderManifest(manifest)
	if err != nil {
		t.Fatalf("render manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, workspace.ManifestFileName), manifestRaw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	for _, domainID := range domainIDs {
		card := fmt.Sprintf("# %s\n\n- id: %s\n- repo_scope: mono\n", domainID, domainID)
		if err := ws.WriteFile(fmt.Sprintf("charter/cards/domains/%s.md", domainID), []byte(card)); err != nil {
			t.Fatalf("write domain card %s: %v", domainID, err)
		}
	}
	teamCard := "# platform\n\n- id: platform\n"
	if err := ws.WriteFile("charter/cards/teams/platform.md", []byte(teamCard)); err != nil {
		t.Fatalf("write team card: %v", err)
	}

	execution := &pipelineExecution{
		workspace:          ws,
		store:              model.NewStore(ws),
		compiler:           reports.NewCompiler(ws),
		selectedRepoScopes: []string{"mono"},
		repoSelectionMode:  workspace.RepoSelectionAll,
		domainRuns:         map[string]domainRunSummary{},
	}
	return execution
}

func hasQuestionIDForServiceFirstTest(questions []contracts.Question, id string) bool {
	target := strings.TrimSpace(id)
	for _, question := range questions {
		if strings.TrimSpace(question.ID) == target {
			return true
		}
	}
	return false
}
