package orchestrator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestRetryStagingReuseInvalidatesRequestedAndDownstreamScopes(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		step   string
		scopes []string
		want   bool
	}{
		{name: "charter reuses nothing", path: "shards/payments/manifest.json", step: "init.step0.constitution", want: false},
		{name: "full collect reuses nothing", path: "shards/users/manifest.json", step: "init.step1.collect", want: false},
		{name: "collect keeps validated sibling", path: "shards/users/manifest.json", step: "init.step1.collect", scopes: []string{"payments"}, want: true},
		{name: "collect invalidates failed scope", path: "shards/payments/manifest.json", step: "init.step1.collect", scopes: []string{"payments"}, want: false},
		{name: "as-is keeps collect", path: "shards/users/manifest.json", step: "init.step2.asis_docs", want: true},
		{name: "as-is invalidates final", path: "final/reports/as-is/overview.md", step: "init.step2.asis_docs", want: false},
		{name: "findings keeps as-is", path: "final/reports/as-is/overview.md", step: "init.step3.findings", want: true},
		{name: "findings invalidates gaps", path: "final/reports/coverage/summary.md", step: "init.step3.findings", want: false},
		{name: "proposals keeps findings", path: "final/reports/findings/findings.md", step: "init.step4.proposals", want: true},
		{name: "proposals invalidates proposal", path: "final/proposals/p-1/proposal.md", step: "init.step4.proposals", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := retryStagingPathReusable(test.path, test.step, test.scopes); got != test.want {
				t.Fatalf("retryStagingPathReusable(%q, %q) = %v, want %v", test.path, test.step, got, test.want)
			}
		})
	}
}

func TestCopyRetryStagingReusesSiblingWithoutMutatingParent(t *testing.T) {
	ws := workspace.Root{Path: t.TempDir()}
	writeRetryShardFixture(t, ws, "parent", "payments")
	writeRetryShardFixture(t, ws, "parent", "users")
	parentUsers := "reports/taskruns/parent/staging/shards/users/shard-pack-manifest.json"
	if err := copyRetryStaging(ws, "parent", "child", "init.step1.collect", []string{"payments"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ws.Path, "reports/taskruns/child/staging/shards/payments/shard-pack-manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("failed scope was reused: %v", err)
	}
	childUsers, err := os.ReadFile(filepath.Join(ws.Path, "reports/taskruns/child/staging/shards/users/shard-pack-manifest.json"))
	if err != nil || len(childUsers) == 0 {
		t.Fatalf("validated sibling was not copied: %q, %v", childUsers, err)
	}
	parentUsersBytes, err := os.ReadFile(filepath.Join(ws.Path, filepath.FromSlash(parentUsers)))
	if err != nil || string(parentUsersBytes) != string(childUsers) {
		t.Fatalf("immutable parent changed: %q, %v", parentUsersBytes, err)
	}
}

func TestCopyRetryStagingRejectsUnvalidatedSibling(t *testing.T) {
	ws := workspace.Root{Path: t.TempDir()}
	writeRetryShardFixture(t, ws, "parent", "users")
	if err := ws.WriteFile("reports/taskruns/parent/staging/shards/users/shard-pack-manifest.json", []byte(`{"not":"valid"}`)); err != nil {
		t.Fatal(err)
	}
	if err := copyRetryStaging(ws, "parent", "child", "init.step2.asis_docs", nil); err == nil {
		t.Fatal("expected invalid parent shard to be rejected")
	}
}

func writeRetryShardFixture(t *testing.T, ws workspace.Root, runID, shardID string) {
	t.Helper()
	root := filepath.Join(ws.Path, "reports", "taskruns", runID, "staging", "shards", shardID)
	manifest := contracts.ShardPackManifest{
		Version: 1, RunID: runID, StepID: "init.step1.collect", ShardID: shardID, DomainID: shardID,
		AgentRole: "shard-analyst", ArtifactRoot: root, RepoScopes: []string{shardID}, PathScopes: []string{"."},
		Documents: []contracts.AuthoredDocument{{ID: "doc." + shardID, Kind: "report", Title: shardID + " overview", Path: "overview.md", CanonicalPath: "reports/as-is/" + shardID + "/overview.md", Topics: []string{shardID}, CitationIDs: []string{"cite." + shardID}}},
		Citations: []contracts.DocumentCitation{{ID: "cite." + shardID, Repo: shardID, Path: "README.md", ClaimIDs: []string{"claim." + shardID}, DocumentIDs: []string{"doc." + shardID}}},
		Semantic:  contracts.SemanticSnapshot{Coverage: contracts.Coverage{Observed: []string{"entrypoints"}, Missing: []string{"owner"}}, Questions: []contracts.Question{{ID: "q." + shardID, Text: "Who owns " + shardID + "?"}}, Entities: []contracts.Entity{{ID: "svc." + shardID, Type: "service", Name: shardID, Provenance: contracts.Provenance{Kind: "observation", Confidence: .8, Evidence: []contracts.Evidence{{Repo: shardID, Path: "README.md"}}}}}, Edges: []contracts.Edge{}, Findings: []contracts.Finding{}},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile(filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "shards", shardID, "overview.md")), []byte("# Overview\n")); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile(filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "shards", shardID, "shard-pack-manifest.json")), raw); err != nil {
		t.Fatal(err)
	}
}
