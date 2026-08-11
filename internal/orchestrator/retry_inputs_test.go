package orchestrator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	parentBefore, err := os.ReadFile(filepath.Join(ws.Path, filepath.FromSlash(parentUsers)))
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil || string(parentUsersBytes) != string(parentBefore) {
		t.Fatalf("immutable parent changed: %q, %v", parentUsersBytes, err)
	}
	childManifest, err := contracts.ParseShardPackManifest(childUsers)
	if err != nil || childManifest.RunID != "child" || childManifest.ArtifactRoot != "reports/taskruns/child/staging/shards/users" {
		t.Fatalf("child manifest was not rebound: %#v, %v", childManifest, err)
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

func TestCopyRetryStagingRejectsInvalidAggregatedInputs(t *testing.T) {
	ws := workspace.Root{Path: t.TempDir()}
	writeRetryShardFixture(t, ws, "parent", "users")
	if err := ws.WriteFile("reports/taskruns/parent/staging/final/final-run-index.json", []byte(`{"invalid":true}`)); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("reports/taskruns/parent/staging/final/citation-index.json", []byte(`{"invalid":true}`)); err != nil {
		t.Fatal(err)
	}
	if err := copyRetryStaging(ws, "parent", "child", "init.step3.findings", nil); err == nil {
		t.Fatal("expected invalid aggregated parent inputs to be rejected")
	}
}

func TestRetryHydratesReboundAggregateAndValidatorInputs(t *testing.T) {
	ws := workspace.Root{Path: t.TempDir()}
	writeRetryShardFixture(t, ws, "parent", "users")
	writeRetryAggregateFixture(t, ws, "parent")
	verdict := contracts.ValidatorVerdict{Version: 1, RunID: "parent", GeneratedAt: "2026-08-03T10:00:00Z", Verdict: "PASS", CheckedPaths: []string{"reports/taskruns/parent/staging/final/final-run-index.json"}, Findings: []contracts.Finding{}, Questions: []contracts.Question{}, Issues: []contracts.ValidatorIssue{}}
	verdictRaw, err := json.Marshal(verdict)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("reports/taskruns/parent/validator/validator-verdict.json", verdictRaw); err != nil {
		t.Fatal(err)
	}
	effective := contracts.EffectiveVerdict{
		Version: 1, Kind: "effective", Authority: "orchestrator", RunID: "parent", GeneratedAt: verdict.GeneratedAt,
		ProviderVerdictPath:   "reports/taskruns/parent/validator/validator-verdict.json",
		ProviderVerdictSHA256: strings.Repeat("0", 64), Verdict: "PASS", CheckedPaths: verdict.CheckedPaths,
		FixedPaths: []string{}, TechnicalIssues: []contracts.ValidatorIssue{}, AdvisoryIssues: []contracts.AdvisoryValidatorIssue{},
		Audit: contracts.EffectiveAuditSummary{Status: "pass", IssueCodes: []string{}},
	}
	effectiveRaw, err := json.Marshal(effective)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("reports/taskruns/parent/validator/effective-verdict.json", effectiveRaw); err != nil {
		t.Fatal(err)
	}
	if err := copyRetryStaging(ws, "parent", "child", "init.step4.proposals", nil); err != nil {
		t.Fatal(err)
	}
	execution := pipelineExecution{workspace: ws, runID: "child", pipelineSemanticDocflowState: pipelineSemanticDocflowState{findings: []contracts.Finding{}, questions: []contracts.Question{}}, pipelineRunProgressState: pipelineRunProgressState{resumeFromStep: "init.step4.proposals"}}
	if err := execution.hydrateRetryInputs(); err != nil {
		t.Fatal(err)
	}
	if execution.finalRunIndex == nil || execution.finalRunIndex.RunID != "child" || execution.citationIndex == nil || execution.citationIndex.RunID != "child" || execution.validatorVerdict == nil || execution.validatorVerdict.RunID != "child" {
		t.Fatalf("retry inputs were not hydrated: final=%#v citation=%#v verdict=%#v", execution.finalRunIndex, execution.citationIndex, execution.validatorVerdict)
	}
	if len(execution.finalRunIndex.CanonicalDocuments) != 2 {
		t.Fatalf("validated findings were not retained for proposals retry: %#v", execution.finalRunIndex.CanonicalDocuments)
	}
	if got := execution.finalRunIndex.CanonicalDocuments[0].StagedPath; !strings.Contains(got, "/child/") {
		t.Fatalf("staged path was not rebound: %q", got)
	}
	if err := copyRetryStaging(ws, "parent", "child-findings", "init.step3.findings", nil); err != nil {
		t.Fatal(err)
	}
	findingsExecution := pipelineExecution{workspace: ws, runID: "child-findings", pipelineSemanticDocflowState: pipelineSemanticDocflowState{findings: []contracts.Finding{}, questions: []contracts.Question{}}, pipelineRunProgressState: pipelineRunProgressState{resumeFromStep: "init.step3.findings"}}
	if err := findingsExecution.hydrateRetryInputs(); err != nil {
		t.Fatal(err)
	}
	if len(findingsExecution.finalRunIndex.CanonicalDocuments) != 1 || findingsExecution.finalRunIndex.CanonicalDocuments[0].CanonicalPath != "reports/as-is/overview.md" {
		t.Fatalf("findings retry retained invalidated downstream documents: %#v", findingsExecution.finalRunIndex.CanonicalDocuments)
	}
}

func writeRetryAggregateFixture(t *testing.T, ws workspace.Root, runID string) {
	t.Helper()
	docPath := filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "final", "reports", "as-is", "overview.md"))
	findingPath := filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "final", "reports", "findings", "findings.md"))
	if err := ws.WriteFile(docPath, []byte("# Overview\n")); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile(findingPath, []byte("# Findings\n")); err != nil {
		t.Fatal(err)
	}
	semantic := contracts.SemanticSnapshot{Coverage: contracts.Coverage{Observed: []string{"users"}, Missing: []string{}, Notes: []string{}}, Questions: []contracts.Question{}, Entities: []contracts.Entity{}, Edges: []contracts.Edge{}, Findings: []contracts.Finding{}}
	final := contracts.FinalRunIndex{Version: 1, RunID: runID, Pipeline: "init", GeneratedAt: "2026-08-03T10:00:00Z", CitationIndexPath: filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "final", "citation-index.json")), CanonicalDocuments: []contracts.FinalRunDocument{{ID: "doc.overview", Kind: "report", Title: "Overview", CanonicalPath: "reports/as-is/overview.md", StagedPath: docPath, Topics: []string{"overview"}, CitationIDs: []string{"cite.users"}, SourceShards: []string{"users"}, Status: "staged"}, {ID: "doc.findings", Kind: "findings", Title: "Findings", CanonicalPath: "reports/findings/findings.md", StagedPath: findingPath, Topics: []string{"findings"}, CitationIDs: []string{"cite.findings"}, SourceShards: []string{"users"}, Status: "staged"}}, Topics: []contracts.TopicIndexEntry{{ID: "overview", DocumentIDs: []string{"doc.overview"}}, {ID: "findings", DocumentIDs: []string{"doc.findings"}}}, Semantic: semantic}
	citations := contracts.CitationIndex{Version: 1, RunID: runID, GeneratedAt: "2026-08-03T10:00:00Z", Citations: []contracts.DocumentCitation{{ID: "cite.users", Repo: "users", Path: "README.md", ClaimIDs: []string{"claim.users"}, DocumentIDs: []string{"doc.overview"}}, {ID: "cite.findings", Repo: "users", Path: "README.md", ClaimIDs: []string{"claim.findings"}, DocumentIDs: []string{"doc.findings"}}}}
	for path, value := range map[string]any{"final-run-index.json": final, "citation-index.json": citations} {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := ws.WriteFile(filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "final", path)), raw); err != nil {
			t.Fatal(err)
		}
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
