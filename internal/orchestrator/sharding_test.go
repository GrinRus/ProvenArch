package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/runtime/codexcode"
	"github.com/GrinRus/ProvenArch/internal/runtime/fakeruntime"
	"github.com/GrinRus/ProvenArch/internal/runtime/qwencode"
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

func hasWarningContaining(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}
