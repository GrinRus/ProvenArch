package refreshplan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type gitFunc func(dir string, args ...string) ([]byte, error)

func (f gitFunc) Run(_ context.Context, dir string, args ...string) ([]byte, error) {
	return f(dir, args...)
}

func TestAnalysisInputFingerprintIsDeterministicAndTracksOnlyAnalysisInputs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for path, content := range map[string]string{"docs/imports/notes.md": "notes", "charter/brief.md": "brief", "skills/a.md": "skill", "reports/as-is/overview.md": "generated"} {
		abs := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ws := workspace.Root{Path: root, Manifest: workspace.Manifest{Repos: []workspace.RepoSource{{Name: "repo", Path: "/private/repo", Analysis: &workspace.RepoAnalysisConfig{Include: []string{"src/**"}}}}, Docs: workspace.DocsConfig{ImportsPath: "./docs/imports"}}}
	first, state, _ := analysisInputFingerprint(ws)
	second, _, _ := analysisInputFingerprint(ws)
	if first != second || state != "complete" {
		t.Fatalf("expected deterministic complete fingerprint, first=%s second=%s state=%s", first, second, state)
	}
	if err := os.WriteFile(filepath.Join(root, "reports/as-is/overview.md"), []byte("changed generated report"), 0o644); err != nil {
		t.Fatal(err)
	}
	generatedChanged, _, _ := analysisInputFingerprint(ws)
	if generatedChanged != first {
		t.Fatalf("generated reports must not affect analysis fingerprint")
	}
	if err := os.WriteFile(filepath.Join(root, "charter/brief.md"), []byte("changed brief"), 0o644); err != nil {
		t.Fatal(err)
	}
	inputChanged, _, _ := analysisInputFingerprint(ws)
	if inputChanged == first {
		t.Fatalf("charter change must affect analysis fingerprint")
	}
}

func TestCaptureSourceRevisionsComparisonAndPrivacy(t *testing.T) {
	t.Parallel()
	sha1 := strings.Repeat("1", 40)
	sha2 := strings.Repeat("2", 40)
	ws := workspace.Root{Path: t.TempDir(), Manifest: workspace.Manifest{Repos: []workspace.RepoSource{{Name: "repo", Path: "/Users/example/private/repo", Ref: "main"}}, Docs: workspace.DocsConfig{ImportsPath: "docs/imports"}}}
	baseline := SourceRevisions{RunID: "old", Repos: []RepoRevision{{Name: "repo", CurrentRevision: &sha1}}}
	git := gitFunc(func(_ string, args ...string) ([]byte, error) {
		switch args[0] {
		case "rev-parse":
			return []byte(sha2), nil
		case "status":
			return []byte("?? new.txt\n"), nil
		case "merge-base":
			return []byte{}, nil
		}
		return nil, fmt.Errorf("unexpected")
	})
	got := CaptureSourceRevisions(context.Background(), ws, []workspace.ResolvedRepo{{Name: "repo", Path: "/resolved/private"}}, "refresh", "new", time.Unix(1, 0), &baseline, git)
	if got.Repos[0].Comparison != "ahead" || got.Repos[0].WorktreeState != "dirty" || !contains(got.Repos[0].FallbackReasons, "dirty_worktree") {
		t.Fatalf("unexpected capture: %+v", got.Repos[0])
	}
	if strings.HasPrefix(got.Repos[0].Path, "/") || strings.Contains(got.Repos[0].Path, "Users/example") {
		t.Fatalf("absolute configured path leaked: %q", got.Repos[0].Path)
	}
}

func TestCaptureSourceRevisionsHandlesMultiRepoRewriteAndUnavailableCommit(t *testing.T) {
	t.Parallel()
	oldSHA := strings.Repeat("a", 40)
	newSHA := strings.Repeat("b", 40)
	ws := workspace.Root{Path: t.TempDir(), Manifest: workspace.Manifest{Repos: []workspace.RepoSource{{Name: "rewritten", Path: "repos/rewritten"}, {Name: "unavailable", GitURL: "https://example.test/unavailable.git"}}, Docs: workspace.DocsConfig{ImportsPath: "docs/imports"}}}
	baseline := SourceRevisions{RunID: "base", Repos: []RepoRevision{{Name: "rewritten", CurrentRevision: &oldSHA}, {Name: "unavailable", CurrentRevision: &oldSHA}}}
	git := gitFunc(func(dir string, args ...string) ([]byte, error) {
		if strings.Contains(dir, "unavailable") && args[0] == "rev-parse" {
			return nil, fmt.Errorf("missing commit")
		}
		switch args[0] {
		case "rev-parse":
			return []byte(newSHA), nil
		case "status":
			return nil, nil
		case "merge-base":
			return nil, fmt.Errorf("not ancestor")
		}
		return nil, fmt.Errorf("unexpected")
	})
	got := CaptureSourceRevisions(context.Background(), ws, []workspace.ResolvedRepo{{Name: "rewritten", Path: "/checkout/rewritten"}, {Name: "unavailable", Path: "/checkout/unavailable"}}, "refresh", "current", time.Unix(1, 0), &baseline, git)
	if len(got.Repos) != 2 {
		t.Fatalf("expected two repo revisions, got %+v", got.Repos)
	}
	if got.Repos[0].Comparison != "diverged" || !contains(got.Repos[0].FallbackReasons, "history_rewritten") {
		t.Fatalf("expected rewritten history fallback, got %+v", got.Repos[0])
	}
	if got.Repos[1].CurrentRevision != nil || !contains(got.Repos[1].FallbackReasons, "current_revision_unavailable") {
		t.Fatalf("expected unavailable commit fallback, got %+v", got.Repos[1])
	}
}

func TestParseNameStatusZHandlesRenameCopyDelete(t *testing.T) {
	t.Parallel()
	raw := []byte("R100\x00old.go\x00new.go\x00C090\x00base.go\x00copy.go\x00D\x00gone.go\x00")
	changes, err := ParseNameStatusZ(raw)
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[string]string{}
	originals := map[string]string{}
	for _, change := range changes {
		statuses[change.Path] = change.Status
		originals[change.Path] = change.OriginalPath
	}
	if statuses["new.go"] != "renamed" || originals["new.go"] != "old.go" || statuses["copy.go"] != "copied" || statuses["gone.go"] != "deleted" {
		t.Fatalf("unexpected changes: %+v", changes)
	}
}

func TestBuildImpactPlanDecisionMatrix(t *testing.T) {
	t.Parallel()
	oldSHA := strings.Repeat("a", 40)
	newSHA := strings.Repeat("b", 40)
	baseline := SourceRevisions{RunID: "base", AnalysisInputFingerprint: strings.Repeat("1", 64), Repos: []RepoRevision{{Name: "repo", CurrentRevision: &oldSHA}}}
	evidence := PriorEvidence{Readable: true, Shards: []ShardScope{{ShardID: "payments-api", DomainID: "payments", RepoScopes: []string{"repo"}, PathScopes: []string{"src/payments/**"}}}, ArtifactShards: map[string][]string{"reports/as-is/payments.md": {"payments-api"}, "reports/as-is/other.md": {"other"}}, CitationDocuments: map[string][]string{}, DocumentPaths: map[string]string{}, AllCanonicalPaths: []string{"reports/as-is/other.md", "reports/as-is/payments.md"}}
	tests := []struct {
		name, diff, fingerprint, worktree, want string
		evidence                                PriorEvidence
	}{
		{name: "unchanged", fingerprint: baseline.AnalysisInputFingerprint, worktree: "clean", want: "unchanged_candidate", evidence: evidence},
		{name: "out of scope", diff: "M\x00docs/readme.md\x00", fingerprint: baseline.AnalysisInputFingerprint, worktree: "clean", want: "unchanged_candidate", evidence: evidence},
		{name: "one domain", diff: "M\x00src/payments/api.go\x00", fingerprint: baseline.AnalysisInputFingerprint, worktree: "clean", want: "selective_candidate", evidence: evidence},
		{name: "input changed", fingerprint: strings.Repeat("2", 64), worktree: "clean", want: "full_refresh_required", evidence: evidence},
		{name: "dirty", fingerprint: baseline.AnalysisInputFingerprint, worktree: "dirty", want: "full_refresh_required", evidence: evidence},
		{name: "unmapped", diff: "M\x00src/unknown.go\x00", fingerprint: baseline.AnalysisInputFingerprint, worktree: "clean", want: "full_refresh_required", evidence: evidence},
		{name: "missing prior evidence", diff: "M\x00src/payments/api.go\x00", fingerprint: baseline.AnalysisInputFingerprint, worktree: "clean", want: "full_refresh_required", evidence: PriorEvidence{Readable: false}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			currentSHA := &newSHA
			comparison := "ahead"
			if tc.diff == "" && tc.name == "unchanged" {
				currentSHA = &oldSHA
				comparison = "unchanged"
			}
			reasons := []string{}
			if tc.worktree == "dirty" {
				reasons = []string{"dirty_worktree"}
			}
			current := SourceRevisions{RunID: "current", Pipeline: "refresh", AnalysisInputFingerprint: tc.fingerprint, AnalysisInputsState: "complete", Repos: []RepoRevision{{Name: "repo", CurrentRevision: currentSHA, BaselineRevision: &oldSHA, WorktreeState: tc.worktree, EffectiveInclude: []string{"src/**"}, EffectiveExclude: []string{"src/generated/**"}, Comparison: comparison, FallbackReasons: reasons}}}
			git := gitFunc(func(_ string, args ...string) ([]byte, error) { return []byte(tc.diff), nil })
			plan := BuildImpactPlan(context.Background(), current, &baseline, []workspace.ResolvedRepo{{Name: "repo", Path: "/repo"}}, tc.evidence, time.Unix(2, 0), git)
			if plan.Decision != tc.want {
				t.Fatalf("decision=%s want=%s reasons=%v", plan.Decision, tc.want, plan.FallbackReasons)
			}
			if tc.name == "one domain" && (len(plan.AffectedDomains) != 1 || plan.AffectedDomains[0] != "payments" || len(plan.StaleArtifactCandidates) != 1) {
				t.Fatalf("unexpected mapped impact: %+v", plan)
			}
		})
	}
}

func TestBuildImpactPlanUsesExactCountAndRejectsPartialMappingOverLimit(t *testing.T) {
	t.Parallel()
	oldSHA := strings.Repeat("a", 40)
	newSHA := strings.Repeat("b", 40)
	records := strings.Builder{}
	for i := 0; i < MaxChangedPaths+1; i++ {
		fmt.Fprintf(&records, "M\x00src/file-%05d.go\x00", i)
	}
	baseline := SourceRevisions{RunID: "base", AnalysisInputFingerprint: strings.Repeat("1", 64), Repos: []RepoRevision{{Name: "repo", CurrentRevision: &oldSHA}}}
	current := SourceRevisions{RunID: "current", AnalysisInputFingerprint: baseline.AnalysisInputFingerprint, AnalysisInputsState: "complete", Repos: []RepoRevision{{Name: "repo", CurrentRevision: &newSHA, BaselineRevision: &oldSHA, WorktreeState: "clean", EffectiveInclude: []string{"src/**"}, Comparison: "ahead"}}}
	plan := BuildImpactPlan(context.Background(), current, &baseline, []workspace.ResolvedRepo{{Name: "repo", Path: "/repo"}}, PriorEvidence{Readable: true}, time.Unix(2, 0), gitFunc(func(_ string, _ ...string) ([]byte, error) { return []byte(records.String()), nil }))
	if plan.Decision != "full_refresh_required" || plan.RepoDeltas[0].ChangedPathCount != 10001 || plan.RepoDeltas[0].ChangesComplete || len(plan.RepoDeltas[0].Changes) != 0 || !contains(plan.FallbackReasons, "change_limit_exceeded") {
		t.Fatalf("unexpected limit plan: %+v", plan)
	}
}

func TestBuildImpactPlanMapsOneChangeToMultipleDomains(t *testing.T) {
	t.Parallel()
	oldSHA := strings.Repeat("a", 40)
	newSHA := strings.Repeat("b", 40)
	baseline := SourceRevisions{RunID: "base", AnalysisInputFingerprint: strings.Repeat("1", 64), Repos: []RepoRevision{{Name: "repo", CurrentRevision: &oldSHA}}}
	current := SourceRevisions{RunID: "current", AnalysisInputFingerprint: baseline.AnalysisInputFingerprint, AnalysisInputsState: "complete", Repos: []RepoRevision{{Name: "repo", CurrentRevision: &newSHA, BaselineRevision: &oldSHA, WorktreeState: "clean", EffectiveInclude: []string{"src/**"}, Comparison: "ahead"}}}
	evidence := PriorEvidence{Readable: true, Shards: []ShardScope{{ShardID: "payments-shared", DomainID: "payments", RepoScopes: []string{"repo"}, PathScopes: []string{"src/shared/**"}}, {ShardID: "orders-shared", DomainID: "orders", RepoScopes: []string{"repo"}, PathScopes: []string{"src/shared/**"}}}, ArtifactShards: map[string][]string{}, CitationDocuments: map[string][]string{}, DocumentPaths: map[string]string{}, ProvenanceDomains: map[string][]string{}, ProvenanceArtifacts: map[string][]string{}}
	plan := BuildImpactPlan(context.Background(), current, &baseline, []workspace.ResolvedRepo{{Name: "repo", Path: "/repo"}}, evidence, time.Unix(2, 0), gitFunc(func(_ string, _ ...string) ([]byte, error) { return []byte("M\x00src/shared/types.go\x00"), nil }))
	if plan.Decision != "selective_candidate" || strings.Join(plan.AffectedDomains, ",") != "orders,payments" {
		t.Fatalf("unexpected multi-domain plan: %+v", plan)
	}
}

func TestImpactPlanMarshallingIsDeterministic(t *testing.T) {
	t.Parallel()
	plan := ImpactPlan{Version: 1, RunID: "run", Pipeline: "refresh", GeneratedAt: time.Unix(1, 0).UTC().Format(time.RFC3339), Enforcement: "advisory", Decision: "full_refresh_required", FallbackReasons: []string{"unmapped_in_scope_path", "dirty_worktree"}, RepoDeltas: []RepoDelta{}, AffectedShards: []string{"b", "a"}, AffectedDomains: []string{}, UnmappedPaths: []string{}, StaleArtifactCandidates: []string{}, PreservedArtifactCandidates: []string{}, PlannedActions: []string{"continue_full_refresh", "conservative_full_refresh"}}
	first, _ := MarshalImpactPlan(plan)
	second, _ := MarshalImpactPlan(plan)
	if string(first) != string(second) {
		t.Fatalf("non-deterministic marshal")
	}
}

func TestPublishedExamplesAndFixturesValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path  string
		parse func([]byte) error
	}{
		{"../../examples/source-revisions.example.json", func(raw []byte) error { _, err := ParseSourceRevisions(raw); return err }},
		{"../../examples/refresh-impact-plan.example.json", func(raw []byte) error { _, err := ParseImpactPlan(raw); return err }},
		{"../../fixtures/refresh-planning/unchanged/source-revisions.json", func(raw []byte) error { _, err := ParseSourceRevisions(raw); return err }},
		{"../../fixtures/refresh-planning/unchanged/refresh-impact-plan.json", func(raw []byte) error { _, err := ParseImpactPlan(raw); return err }},
		{"../../fixtures/refresh-planning/full-fallback/refresh-impact-plan.json", func(raw []byte) error { _, err := ParseImpactPlan(raw); return err }},
	}
	for _, tc := range tests {
		raw, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatalf("read %s: %v", tc.path, err)
		}
		if err := tc.parse(raw); err != nil {
			t.Fatalf("validate %s: %v", tc.path, err)
		}
	}
}

func TestLoadLatestValidBaselineSkipsLegacyRunWithoutRevisionArtifact(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ws := workspace.Root{Path: root}
	readExample := func(name string) []byte {
		raw, err := os.ReadFile(filepath.Join("../../examples", name))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	for _, runID := range []string{"legacy", "run-123"} {
		final := strings.ReplaceAll(string(readExample("final-run-index.example.json")), "run-123", runID)
		verdict := strings.ReplaceAll(string(readExample("validator-verdict.example.json")), "run-123", runID)
		effective := strings.ReplaceAll(string(readExample("effective-verdict.example.json")), "run-123", runID)
		citation := strings.ReplaceAll(string(readExample("citation-index.example.json")), "run-123", runID)
		for rel, raw := range map[string][]byte{
			filepath.Join("reports/taskruns", runID, "staging/final/final-run-index.json"): []byte(final),
			filepath.Join("reports/taskruns", runID, "staging/final/citation-index.json"):  []byte(citation),
			filepath.Join("reports/taskruns", runID, "validator/validator-verdict.json"):   []byte(verdict),
			filepath.Join("reports/taskruns", runID, "validator/effective-verdict.json"):   []byte(effective),
		} {
			if err := ws.WriteFile(filepath.ToSlash(rel), raw); err != nil {
				t.Fatal(err)
			}
		}
	}
	sha := strings.Repeat("a", 40)
	revisions := SourceRevisions{Version: 1, RunID: "run-123", Pipeline: "init", CapturedAt: time.Unix(1, 0).UTC().Format(time.RFC3339), AnalysisInputFingerprint: strings.Repeat("1", 64), AnalysisInputsState: "complete", InputIssues: []string{}, Repos: []RepoRevision{{Name: "repo", SourceKind: "path", Path: "repos/repo", ConfiguredRef: "", CurrentRevision: &sha, WorktreeState: "clean", EffectiveInclude: []string{"**"}, EffectiveExclude: []string{}, Comparison: "initial", FallbackReasons: []string{"initial_run"}}}}
	raw, err := MarshalSourceRevisions(revisions)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("reports/taskruns/run-123/source-revisions.json", raw); err != nil {
		t.Fatal(err)
	}
	plan := `{"version":1,"run_id":"run-123","step_id":"init.step1.collect","domain_id":"payments","strategy":"sequential","max_parallel_tasks":1,"failure_policy":"fail_fast","shard_discovery_mode":"static","generated_at":"2026-07-15T12:00:00Z","items":[{"sort_key":"1","shard_id":"payments-api","repo_scopes":["repo"],"path_scopes":["src/**"]}]}`
	if err := ws.WriteFile("reports/taskruns/run-123-init-step1-collect-shard-plan-payments.json", []byte(plan)); err != nil {
		t.Fatal(err)
	}
	baseline, evidence := LoadLatestValidBaseline(ws, []string{"legacy", "run-123"})
	if baseline == nil || baseline.RunID != "run-123" {
		t.Fatalf("expected valid non-legacy baseline, got %+v", baseline)
	}
	if !evidence.Readable || len(evidence.Shards) != 1 {
		t.Fatalf("expected readable prior evidence, got %+v", evidence)
	}
}
