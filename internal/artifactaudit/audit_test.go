package artifactaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestScanSelectedRunPassesDeterministicallyWithoutMutation(t *testing.T) {
	ws, runID := writeAuditFixture(t, false)
	before := snapshotFixtureFiles(t, ws.Path)

	first := ScanSelectedRun(ws, runID)
	second := ScanSelectedRun(ws, runID)
	if first.Status != StatusPass {
		t.Fatalf("expected clean audit PASS, got %+v", first)
	}
	firstRaw, err := Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondRaw, err := Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstRaw) != string(secondRaw) {
		t.Fatal("repeated audit output is not byte-identical")
	}
	after := snapshotFixtureFiles(t, ws.Path)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("read-only audit mutated workspace or source repository")
	}
	if len(first.Artifacts) != 4 {
		t.Fatalf("expected final index, citation index, validator verdict and overview digests, got %+v", first.Artifacts)
	}
}

func TestAuditTruncationIsFailClosed(t *testing.T) {
	audit := auditor{report: Report{Version: Version, Status: StatusPass, Issues: []Issue{}, Artifacts: []Artifact{}}}
	for index := 0; index < MaxIssues+10; index++ {
		audit.add("audit.synthetic.warning", "warning", "reports/doc.md", nil, "synthetic warning")
	}
	audit.finish()
	if !audit.report.Truncated || audit.report.Status != StatusFail {
		t.Fatalf("truncated audit must fail closed: %+v", audit.report)
	}
	if !hasIssue(audit.report, "audit.scan.truncated") {
		t.Fatalf("truncated audit must expose a typed issue: %+v", audit.report.Issues)
	}
}

func TestScanSelectedRunPublicRequiresEffectiveAuthority(t *testing.T) {
	ws, runID := writeAuditFixture(t, false)
	legacy := ScanSelectedRunPublic(ws, runID)
	if legacy.Status != StatusFail || !hasIssue(legacy, "audit.effective_verdict.unavailable") || legacy.EffectiveAuthority != "legacy_unavailable" {
		t.Fatalf("expected explicit legacy/unavailable authority, got %+v", legacy)
	}
	providerPath := path.Join("reports", "taskruns", runID, "validator", "validator-verdict.json")
	providerRaw, err := ws.ReadFile(providerPath)
	if err != nil {
		t.Fatal(err)
	}
	provider := contracts.ValidatorVerdict{}
	if err := json.Unmarshal(providerRaw, &provider); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(providerRaw)
	effective := contracts.EffectiveVerdict{
		Version: 1, Kind: "effective", Authority: "orchestrator", RunID: runID, GeneratedAt: provider.GeneratedAt,
		ProviderVerdictPath: providerPath, ProviderVerdictSHA256: hex.EncodeToString(sum[:]), Verdict: "PASS",
		CheckedPaths: provider.CheckedPaths, FixedPaths: []string{}, Findings: provider.Findings, Questions: provider.Questions,
		TechnicalIssues: []contracts.ValidatorIssue{}, AdvisoryIssues: []contracts.AdvisoryValidatorIssue{},
		Audit: contracts.EffectiveAuditSummary{Status: "pass", IssueCodes: []string{}},
	}
	effectiveRaw, err := json.MarshalIndent(effective, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFileAtomic(path.Join("reports", "taskruns", runID, "validator", "effective-verdict.json"), append(effectiveRaw, '\n')); err != nil {
		t.Fatal(err)
	}
	public := ScanSelectedRunPublic(ws, runID)
	if public.Status != StatusPass || public.EffectiveAuthority != "effective" {
		t.Fatalf("expected effective public PASS authority, got %+v", public)
	}
}

func TestScanPromotedRunDetectsCanonicalDigestMismatch(t *testing.T) {
	ws, runID := writeAuditFixture(t, false)
	stagedPath := path.Join("reports", "taskruns", runID, "staging", "final", "reports", "as-is", "overview.md")
	staged, err := ws.ReadFile(stagedPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFileAtomic("reports/as-is/overview.md", staged); err != nil {
		t.Fatal(err)
	}
	if report := ScanPromotedRun(ws, runID); report.Status != StatusPass {
		t.Fatalf("expected matching promoted bytes to pass, got %+v", report)
	}
	if err := ws.WriteFileAtomic("reports/as-is/overview.md", append(staged, []byte("\nuser edit\n")...)); err != nil {
		t.Fatal(err)
	}
	report := ScanPromotedRun(ws, runID)
	if !hasIssue(report, "audit.promoted.digest_mismatch") {
		t.Fatalf("expected promoted digest mismatch, got %+v", report.Issues)
	}
}

func TestScanSelectedRunDetectsHistoricalIncidentClasses(t *testing.T) {
	ws, runID := writeAuditFixture(t, true)
	report := ScanSelectedRun(ws, runID)
	for _, code := range []string{
		"audit.document.execution_contamination",
		"audit.document.scaffold",
		"audit.evidence.file_missing",
	} {
		if !hasIssue(report, code) {
			t.Fatalf("expected incident code %q in %+v", code, report.Issues)
		}
	}
	if report.Status != StatusFail {
		t.Fatalf("expected incident audit FAIL, got %q", report.Status)
	}
}

func TestScanSelectedRunRejectsForeignIdentityAndOversizedFiles(t *testing.T) {
	ws, runID := writeAuditFixture(t, false)
	finalPath := path.Join("reports", "taskruns", runID, "staging", "final", "final-run-index.json")
	raw, err := ws.ReadFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	payload["run_id"] = "run-foreign"
	raw, _ = json.Marshal(payload)
	if err := ws.WriteFileAtomic(finalPath, raw); err != nil {
		t.Fatal(err)
	}
	report := ScanSelectedRun(ws, runID)
	if !hasIssue(report, "audit.run_identity.foreign") {
		t.Fatalf("expected foreign run issue, got %+v", report.Issues)
	}

	if err := ws.WriteFileAtomic(finalPath, make([]byte, MaxArtifactBytes+1)); err != nil {
		t.Fatal(err)
	}
	report = ScanSelectedRun(ws, runID)
	if !hasIssue(report, "audit.final_index.unavailable") {
		t.Fatalf("expected bounded read issue, got %+v", report.Issues)
	}
}

func TestScanSelectedRunValidatesCitationEvidenceBytes(t *testing.T) {
	ws, runID := writeAuditFixture(t, false)
	citationPath := path.Join("reports", "taskruns", runID, "staging", "final", "citation-index.json")
	raw, err := ws.ReadFile(citationPath)
	if err != nil {
		t.Fatal(err)
	}
	var index contracts.CitationIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	selected := "# Evidence"
	digest := sha256.Sum256([]byte(selected))
	index.Citations[0].Lines = &contracts.LineRange{Start: 1, End: 1}
	index.Citations[0].Excerpt = selected
	index.Citations[0].ExcerptHash = hex.EncodeToString(digest[:])
	updated, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFileAtomic(citationPath, append(updated, '\n')); err != nil {
		t.Fatal(err)
	}
	if report := ScanSelectedRun(ws, runID); report.Status != StatusPass {
		t.Fatalf("expected exact line evidence to pass, got %+v", report)
	}

	index.Citations[0].Excerpt = "Evidence"
	updated, err = json.MarshalIndent(index, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFileAtomic(citationPath, append(updated, '\n')); err != nil {
		t.Fatal(err)
	}
	report := ScanSelectedRun(ws, runID)
	if !hasIssue(report, "audit.evidence.excerpt_mismatch") {
		t.Fatalf("expected exact excerpt issue, got %+v", report.Issues)
	}
}

func writeAuditFixture(t *testing.T, incident bool) (workspace.Root, string) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# Evidence\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws := workspace.Root{
		Path: root,
		Manifest: workspace.Manifest{Version: 1, Repos: []workspace.RepoSource{{
			Name: "sample", Path: repo,
		}}},
	}
	runID := "run-audit"
	finalRoot := path.Join("reports", "taskruns", runID, "staging", "final")
	stagedOverview := path.Join(finalRoot, "reports", "as-is", "overview.md")
	var overview strings.Builder
	overview.WriteString("# Architecture Home\n\n")
	for _, section := range runtimedrafts.ArchitectureHomeRequiredSections() {
		overview.WriteString("## " + section + "\n\nEvidence-backed content for this section.\n\n")
	}
	evidencePath := "README.md"
	if incident {
		overview.WriteString("reports/taskruns/run-other/staging/final and Recovery Bootstrap: replace placeholder content\n")
		evidencePath = "missing.md"
	}
	if err := ws.WriteFileAtomic(stagedOverview, []byte(overview.String())); err != nil {
		t.Fatal(err)
	}
	document := contracts.FinalRunDocument{
		ID: "doc.overview", Kind: "report", Title: "Architecture Home",
		CanonicalPath: "reports/as-is/overview.md", StagedPath: stagedOverview,
		Topics: []string{"architecture"}, CitationIDs: []string{"cite.overview"},
		SourceShards: []string{"sample"}, Status: "staged",
	}
	citation := contracts.DocumentCitation{
		ID: "cite.overview", Repo: "sample", Path: evidencePath,
		ClaimIDs: []string{"claim.overview"}, DocumentIDs: []string{"doc.overview"},
	}
	semantic := contracts.SemanticSnapshot{
		Coverage:  contracts.Coverage{Observed: []string{"README"}, Missing: []string{}, Notes: []string{}},
		Questions: []contracts.Question{}, Entities: []contracts.Entity{},
		Edges: []contracts.Edge{}, Findings: []contracts.Finding{},
	}
	finalIndex := contracts.FinalRunIndex{
		Version: 1, RunID: runID, Pipeline: "init", GeneratedAt: "2026-07-26T00:00:00Z",
		CitationIndexPath:  path.Join(finalRoot, "citation-index.json"),
		CanonicalDocuments: []contracts.FinalRunDocument{document},
		Topics:             []contracts.TopicIndexEntry{{ID: "architecture", DocumentIDs: []string{"doc.overview"}}},
		Semantic:           semantic,
	}
	citationIndex := contracts.CitationIndex{
		Version: 1, RunID: runID, GeneratedAt: "2026-07-26T00:00:00Z",
		Citations: []contracts.DocumentCitation{citation},
	}
	verdict := contracts.ValidatorVerdict{
		Version: 1, RunID: runID, GeneratedAt: "2026-07-26T00:00:00Z",
		Verdict: "PASS", Summary: "selected-run snapshot passed validation",
		CheckedPaths: []string{path.Join(finalRoot, "final-run-index.json"), path.Join(finalRoot, "citation-index.json")},
		FixedPaths:   []string{}, Findings: []contracts.Finding{},
		Questions: []contracts.Question{}, Issues: []contracts.ValidatorIssue{},
	}
	for filePath, value := range map[string]any{
		path.Join(finalRoot, "final-run-index.json"):                                   finalIndex,
		path.Join(finalRoot, "citation-index.json"):                                    citationIndex,
		path.Join("reports", "taskruns", runID, "validator", "validator-verdict.json"): verdict,
	} {
		raw, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := ws.WriteFileAtomic(filePath, append(raw, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	return ws, runID
}

func snapshotFixtureFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		raw, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, filePath)
		result[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func hasIssue(report Report, code string) bool {
	for _, issue := range report.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
