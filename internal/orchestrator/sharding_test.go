package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/runtime/codexcode"
	"github.com/GrinRus/ProvenArch/internal/runtime/fakeruntime"
	"github.com/GrinRus/ProvenArch/internal/runtime/qwencode"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestRuntimeMetaForRunnerCoversReleaseProviders(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		runner      acpruntime.Runner
		wantName    string
		wantVersion string
	}{
		{name: "fake", runner: fakeruntime.Runner{}, wantName: "fake", wantVersion: "fake"},
		{name: "claude", runner: claudecode.HeadlessRunner{}, wantName: "claude-code", wantVersion: "headless"},
		{name: "qwen", runner: qwencode.HeadlessRunner{}, wantName: "qwen-code", wantVersion: "headless"},
		{name: "codex", runner: codexcode.HeadlessRunner{}, wantName: "codex-code", wantVersion: "headless"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			meta := runtimeMetaForRunner(tc.runner)
			if meta.Name != tc.wantName || meta.Version != tc.wantVersion {
				t.Fatalf("unexpected runtime meta: got=%+v want name=%q version=%q", meta, tc.wantName, tc.wantVersion)
			}
		})
	}
}

type customMetadataRunner struct{}

func (customMetadataRunner) Run(context.Context, acpruntime.Task) (acpruntime.Result, error) {
	return acpruntime.Result{}, nil
}

func (customMetadataRunner) RuntimeMeta() contracts.RuntimeMeta {
	return contracts.RuntimeMeta{Name: "custom-runtime", Version: "v1"}
}

func TestRuntimeMetaForRunnerUsesMetadataInterface(t *testing.T) {
	t.Parallel()

	meta := runtimeMetaForRunner(customMetadataRunner{})
	if meta.Name != "custom-runtime" || meta.Version != "v1" {
		t.Fatalf("unexpected runtime meta from interface: %+v", meta)
	}
}

func TestPipelineExecutionRepoScopesUsesAllWorkspaceRepos(t *testing.T) {
	t.Parallel()

	execution := pipelineExecution{
		workspace: workspace.Root{
			Manifest: workspace.Manifest{
				Repos: []workspace.RepoSource{
					{Name: "zeta"},
					{Name: "alpha"},
					{Name: "alpha"},
					{Name: " "},
				},
			},
		},
	}

	got := execution.repoScopes()
	want := []string{"alpha", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected repo scopes: got=%v want=%v", got, want)
	}
}

func TestScheduleRuntimeShardRunsReturnsPlanOrder(t *testing.T) {
	t.Parallel()

	plans := []runtimeShardPlan{
		{ShardID: "shard-b"},
		{ShardID: "shard-a"},
		{ShardID: "shard-c"},
	}
	summaryState := &runtimeShardSummaryState{
		singleShard: false,
		entries: []runtimeShardSummaryEntry{
			{ShardID: "shard-b", Status: "failed", Error: "b failed"},
			{ShardID: "shard-a", Status: "failed", Error: "a failed"},
			{ShardID: "shard-c", Status: "failed", Error: "c failed"},
		},
		index: map[string]int{
			"shard-b": 0,
			"shard-a": 1,
			"shard-c": 2,
		},
	}
	execution := pipelineExecution{}

	results, terminalErr := execution.scheduleRuntimeShardRuns(
		context.Background(),
		"refresh.step1.collect",
		"domain-test",
		plans,
		summaryState,
		runtimeShardExecutionOptions{
			Strategy:      "parallel",
			MaxParallel:   2,
			FailurePolicy: "best_effort",
			BestEffort:    true,
		},
		"domain-test",
	)
	if terminalErr != nil {
		t.Fatalf("unexpected terminal error: %v", terminalErr)
	}
	if len(results) != len(plans) {
		t.Fatalf("unexpected result count: got=%d want=%d", len(results), len(plans))
	}
	for idx, result := range results {
		if result.Plan.ShardID != plans[idx].ShardID {
			t.Fatalf("result %d out of plan order: got=%q want=%q", idx, result.Plan.ShardID, plans[idx].ShardID)
		}
		if result.Err == nil {
			t.Fatalf("result %d unexpectedly succeeded", idx)
		}
	}
}

func TestScheduleRuntimeShardRunsFailFastDoesNotStartNextSequentialShard(t *testing.T) {
	t.Parallel()

	plans := []runtimeShardPlan{
		{ShardID: "shard-failed"},
		{ShardID: "shard-pending"},
	}
	summaryState := &runtimeShardSummaryState{
		singleShard: false,
		entries: []runtimeShardSummaryEntry{
			{ShardID: "shard-failed", Status: "failed", Error: "first shard failed"},
			{ShardID: "shard-pending", Status: "pending"},
		},
		index: map[string]int{
			"shard-failed":  0,
			"shard-pending": 1,
		},
	}
	execution := pipelineExecution{}

	results, terminalErr := execution.scheduleRuntimeShardRuns(
		context.Background(),
		"refresh.step1.collect",
		"domain-test",
		plans,
		summaryState,
		runtimeShardExecutionOptions{
			Strategy:      "sequential",
			MaxParallel:   1,
			FailurePolicy: "fail_fast",
			BestEffort:    false,
		},
		"domain-test",
	)
	if terminalErr == nil {
		t.Fatalf("expected terminal error from failed first shard")
	}
	if len(results) != len(plans) {
		t.Fatalf("unexpected result count: got=%d want=%d", len(results), len(plans))
	}
	if results[0].Err == nil {
		t.Fatalf("expected first shard to fail")
	}
	if results[1].Prepared.Task.TaskID != "" || results[1].Err != nil {
		t.Fatalf("expected second shard to remain undispatched, got prepared=%+v err=%v", results[1].Prepared.Task, results[1].Err)
	}
}

func TestStructuralShardCoalescingPreservesMarkerLeafGroupsWithinCap(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	coverageRoots := []string{
		".gitignore",
		".pylintrc",
		"LICENSE",
		"Makefile",
		"README.md",
		"docs",
		"extras",
		"iac",
		"kubernetes-manifests",
		"mvnw",
		"mvnw.cmd",
		"pom.xml",
		"skaffold-e2e.yaml",
		"skaffold.yaml",
		"src/accounts",
		"src/components",
		"src/frontend",
		"src/ledger/balancereader",
		"src/ledger/cloudbuild.yaml",
		"src/ledger/components",
		"src/ledger/ledger-db",
		"src/ledger/ledgerwriter",
		"src/ledger/skaffold.yaml",
		"src/ledger/transactionhistory",
		"src/ledgermonolith",
		"src/loadgenerator",
	}
	for _, rel := range coverageRoots {
		writeShardFixturePath(t, repoPath, rel)
	}
	for _, rel := range []string{
		"src/ledger/balancereader/pom.xml",
		"src/ledger/ledgerwriter/pom.xml",
		"src/ledger/transactionhistory/pom.xml",
		"src/ledgermonolith/pom.xml",
	} {
		writeShardFixturePath(t, repoPath, rel)
	}

	groups, warnings := buildStructuralShardGroups(repoPath, coverageRoots)
	if got := len(groups); got > maxAutoShardsPerRepo {
		t.Fatalf("groups = %d, want <= %d: %#v", got, maxAutoShardsPerRepo, groups)
	}
	if hasSinglePathGroup(groups, "src") {
		t.Fatalf("expected src to be split into marker-aware groups, got %#v", groups)
	}
	for _, rel := range []string{
		"src/ledger/balancereader",
		"src/ledger/ledgerwriter",
		"src/ledger/transactionhistory",
		"src/ledgermonolith",
	} {
		if !hasSinglePathGroup(groups, rel) {
			t.Fatalf("expected marker leaf group %q in %#v", rel, groups)
		}
	}
	if !hasWarningContaining(warnings, "preserved 4 module marker leaf shard groups") {
		t.Fatalf("expected marker preservation warning, got %#v", warnings)
	}
}

func TestStructuralShardCoalescingSkipsMarkerLeavesWhenCapWouldBeExceeded(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	coverageRoots := make([]string, 0, maxAutoShardsPerRepo+1)
	for idx := 0; idx < maxAutoShardsPerRepo-1; idx++ {
		rel := "area-" + string(rune('a'+idx))
		coverageRoots = append(coverageRoots, rel)
		if err := os.MkdirAll(filepath.Join(repoPath, rel), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
	}
	for _, rel := range []string{
		"src/service-a",
		"src/service-b",
	} {
		coverageRoots = append(coverageRoots, rel)
		if err := os.MkdirAll(filepath.Join(repoPath, filepath.FromSlash(rel)), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		writeShardFixturePath(t, repoPath, rel+"/pom.xml")
	}

	groups, warnings := buildStructuralShardGroups(repoPath, coverageRoots)
	if got := len(groups); got != maxAutoShardsPerRepo {
		t.Fatalf("groups = %d, want %d: %#v", got, maxAutoShardsPerRepo, groups)
	}
	if !hasSinglePathGroup(groups, "src") {
		t.Fatalf("expected src to remain coalesced when marker leaves exceed cap, got %#v", groups)
	}
	if hasSinglePathGroup(groups, "src/service-a") || hasSinglePathGroup(groups, "src/service-b") {
		t.Fatalf("did not expect marker leaves to be preserved beyond cap, got %#v", groups)
	}
	if !hasWarningContaining(warnings, `marker preservation skipped for "src"`) {
		t.Fatalf("expected cap warning for src marker preservation, got %#v", warnings)
	}
}

func TestRootMarkerOnlyLargeRepoDoesNotCollapseToRootShard(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	for _, rel := range []string{
		"pyproject.toml",
		"README.rst",
		"MAINTAINERS",
		"api-guide",
		"api-ref",
		"devstack",
		"doc",
		"etc",
		"gate",
		"nova",
		"playbooks",
		"releasenotes",
		"roles",
		"tools",
	} {
		writeShardFixturePath(t, repoPath, rel)
	}

	discovery, err := discoverHeuristicShardPathsWithMeta(repoPath)
	if err != nil {
		t.Fatalf("discover root-marker repo shards: %v", err)
	}
	if len(discovery.Paths) == 1 && discovery.Paths[0] == "." {
		t.Fatalf("root-marker-only repo collapsed to root shard: %#v", discovery.Paths)
	}
	groups, warnings := buildStructuralShardGroups(repoPath, discovery.Paths)
	if got := len(groups); got > maxAutoShardsPerRepo {
		t.Fatalf("groups = %d, want <= %d: %#v warnings=%#v", got, maxAutoShardsPerRepo, groups, warnings)
	}
	if hasSinglePathGroup(groups, ".") {
		t.Fatalf("did not expect root shard group for large root-marker repo: %#v", groups)
	}
	if !hasGroupWithAllPaths(groups, []string{"MAINTAINERS", "README.rst", "pyproject.toml"}) {
		t.Fatalf("expected root-file group to preserve root metadata files, got %#v", groups)
	}
	for _, rel := range []string{"nova", "doc", "tools"} {
		if !hasSinglePathGroup(groups, rel) {
			t.Fatalf("expected top-level shard %q in %#v", rel, groups)
		}
	}
}

func TestStructuralShardCoalescingEnforcesCapForManyTopLevelRoots(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	coverageRoots := []string{"README.rst", "pyproject.toml"}
	for idx := 0; idx < 40; idx++ {
		rel := "project-" + string(rune('a'+idx/26)) + string(rune('a'+idx%26))
		coverageRoots = append(coverageRoots, rel)
		if err := os.MkdirAll(filepath.Join(repoPath, rel), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
	}
	for _, rel := range []string{"README.rst", "pyproject.toml"} {
		writeShardFixturePath(t, repoPath, rel)
	}

	groups, warnings := buildStructuralShardGroups(repoPath, coverageRoots)
	if got := len(groups); got != maxAutoShardsPerRepo {
		t.Fatalf("groups = %d, want %d: %#v warnings=%#v", got, maxAutoShardsPerRepo, groups, warnings)
	}
	if hasSinglePathGroup(groups, ".") {
		t.Fatalf("did not expect root shard group for many-top-level repo: %#v", groups)
	}
	if !hasGroupWithAllPaths(groups, []string{"README.rst", "pyproject.toml"}) {
		t.Fatalf("expected root-file group to preserve root metadata files, got %#v", groups)
	}
	if !hasWarningContaining(warnings, "to enforce target cap") {
		t.Fatalf("expected cap enforcement warning, got %#v", warnings)
	}
	if !hasMultiPathNonRootGroup(groups) {
		t.Fatalf("expected at least one merged top-level group, got %#v", groups)
	}
	seen := map[string]int{}
	for _, group := range groups {
		for _, rel := range group {
			seen[rel]++
		}
	}
	for _, rel := range coverageRoots {
		if seen[rel] != 1 {
			t.Fatalf("coverage root %q seen %d times in groups %#v", rel, seen[rel], groups)
		}
	}
}

func TestBuildShardIDBoundsLongRootFileGroups(t *testing.T) {
	t.Parallel()

	paths := []string{
		".babelrc",
		".coveragerc",
		".dockerignore",
		".editorconfig",
		".gitattributes",
		".gitignore",
		".npmignore",
		".npmrc",
		".nvmrc",
		"README.rst",
		"catalog-info.yaml",
		"conftest.py",
		"manage.py",
		"package-lock.json",
		"package.json",
		"pyproject.toml",
		"tox.ini",
		"webpack-prod.config.js",
	}

	shardID := buildShardID("openedx-platform", paths)
	if len(shardID) > maxRuntimeShardIDLength {
		t.Fatalf("shard id length=%d want <= %d: %q", len(shardID), maxRuntimeShardIDLength, shardID)
	}
	if !strings.HasPrefix(shardID, "openedx-platform") {
		t.Fatalf("bounded shard id should keep a readable repo prefix, got %q", shardID)
	}
	sequenced := appendShardIDSequence(shardID, 23)
	if len(sequenced) > maxRuntimeShardIDLength {
		t.Fatalf("sequenced shard id length=%d want <= %d: %q", len(sequenced), maxRuntimeShardIDLength, sequenced)
	}
	if !strings.HasSuffix(sequenced, "-23") {
		t.Fatalf("sequenced shard id should keep sequence suffix, got %q", sequenced)
	}
}

func TestShardPlanItemsInvariantAcrossBaselineAndParallelDefault(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	repoPath := filepath.Join(workspaceRoot, "repos", "nova")
	for _, rel := range []string{
		"pyproject.toml",
		"README.rst",
		"api-guide",
		"api-ref",
		"devstack",
		"doc",
		"etc",
		"gate",
		"nova",
		"playbooks",
		"releasenotes",
		"roles",
		"tools",
	} {
		writeShardFixturePath(t, repoPath, rel)
	}
	manifest := "version: 1\nrepos:\n  - name: nova\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspaceRoot, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}
	ws, err := workspace.Open(workspaceRoot)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	planFor := func(profile acpruntime.ExecutionValues) []runtimeShardPlan {
		t.Helper()
		execution := &pipelineExecution{
			workspace: ws,
			pipelineRuntimeState: pipelineRuntimeState{
				executionProfile:  profile,
				resolvedRepoPaths: map[string]string{"nova": repoPath},
			},
		}
		plans, warnings, _ := execution.planRuntimeShards([]string{"nova"})
		if len(plans) == 0 {
			t.Fatalf("empty shard plan warnings=%#v", warnings)
		}
		if len(plans) == 1 && len(plans[0].PathScopes) == 1 && plans[0].PathScopes[0] == "." {
			t.Fatalf("root-marker-only repo collapsed to root shard under profile %+v: warnings=%#v", profile, warnings)
		}
		for _, plan := range plans {
			if len(plan.ShardID) > maxRuntimeShardIDLength {
				t.Fatalf("shard id length=%d want <= %d for %#v", len(plan.ShardID), maxRuntimeShardIDLength, plan)
			}
		}
		return plans
	}

	baseline := planFor(acpruntime.ExecutionValues{
		Strategy:      "sequential",
		MaxParallel:   1,
		FailurePolicy: "best_effort",
		ShardMode:     "heuristics",
	})
	parallelDefault := planFor(acpruntime.ExecutionValues{
		Strategy:      "parallel",
		MaxParallel:   4,
		FailurePolicy: "best_effort",
		ShardMode:     "heuristics",
	})

	if !reflect.DeepEqual(baseline, parallelDefault) {
		t.Fatalf("baseline and parallel-default shard plans differ:\nbaseline=%#v\nparallel=%#v", baseline, parallelDefault)
	}
}

func writeShardFixturePath(t *testing.T, root string, rel string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	fixtureDirs := map[string]struct{}{
		"docs":                          {},
		"extras":                        {},
		"iac":                           {},
		"kubernetes-manifests":          {},
		"src/accounts":                  {},
		"src/components":                {},
		"src/frontend":                  {},
		"src/ledger/balancereader":      {},
		"src/ledger/components":         {},
		"src/ledger/ledger-db":          {},
		"src/ledger/ledgerwriter":       {},
		"src/ledger/transactionhistory": {},
		"src/ledgermonolith":            {},
		"src/loadgenerator":             {},
		"api-guide":                     {},
		"api-ref":                       {},
		"devstack":                      {},
		"doc":                           {},
		"etc":                           {},
		"gate":                          {},
		"nova":                          {},
		"playbooks":                     {},
		"releasenotes":                  {},
		"roles":                         {},
		"tools":                         {},
	}
	if _, ok := fixtureDirs[rel]; ok {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent for %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte("fixture\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func hasSinglePathGroup(groups [][]string, rel string) bool {
	for _, group := range groups {
		if len(group) == 1 && group[0] == rel {
			return true
		}
	}
	return false
}

func hasGroupWithAllPaths(groups [][]string, paths []string) bool {
	for _, group := range groups {
		seen := map[string]struct{}{}
		for _, rel := range group {
			seen[rel] = struct{}{}
		}
		missing := false
		for _, rel := range paths {
			if _, ok := seen[rel]; !ok {
				missing = true
				break
			}
		}
		if !missing {
			return true
		}
	}
	return false
}

func hasMultiPathNonRootGroup(groups [][]string) bool {
	for _, group := range groups {
		if len(group) <= 1 {
			continue
		}
		if hasRootFileLikePath(group) {
			continue
		}
		return true
	}
	return false
}

func hasRootFileLikePath(group []string) bool {
	for _, rel := range group {
		if !strings.Contains(rel, "/") && strings.Contains(filepath.Base(rel), ".") {
			return true
		}
	}
	return false
}

func hasWarningContaining(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}
