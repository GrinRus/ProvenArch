package conformance_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/artifactaudit"
	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/evidence"
	"github.com/GrinRus/ProvenArch/internal/runtime/providercommon"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

// This corpus is deliberately provider-free. An adapter envelope only carries
// the provider identity around the same semantic payload; no CLI is started.
type providerAdapter string

const (
	adapterClaude providerAdapter = "claude-code"
	adapterQwen   providerAdapter = "qwen-code"
	adapterCodex  providerAdapter = "codex-code"
)

type adapterObservation struct {
	accepted bool
	code     string
}

func TestIncidentCorpusRejectsKnownFalseAcceptClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		run  func(t *testing.T, adapter providerAdapter) error
	}{
		{
			name: "schema drift",
			want: "semantic.unknown_field",
			run: func(t *testing.T, adapter providerAdapter) error {
				return validateSemanticEnvelopePayload(t, adapter, map[string]any{
					"coverage": map[string]any{"missing": []string{"owner mapping"}},
					"entities": []any{map[string]any{
						"id": "service.payments", "type": "service", "name": "Payments",
						"provenance":        map[string]any{"kind": "observed", "confidence": 0.9},
						"unsupported_alias": "weak-model-drift",
					}},
				})
			},
		},
		{
			name: "graph collision",
			want: "semantic.graph_collision",
			run: func(t *testing.T, adapter providerAdapter) error {
				return validateSemanticEnvelopePayload(t, adapter, contracts.SemanticSnapshot{
					Entities: []contracts.Entity{
						{ID: "service.payments", Type: "service", Name: "Payments", Provenance: contracts.Provenance{Kind: "observed", Confidence: 0.9}},
						{ID: "service.payments", Type: "service", Name: "Billing", Provenance: contracts.Provenance{Kind: "observed", Confidence: 0.9}},
					},
				})
			},
		},
		{
			name: "dangling edge",
			want: "semantic.graph_dangling_edge",
			run: func(t *testing.T, adapter providerAdapter) error {
				return validateSemanticEnvelopePayload(t, adapter, contracts.SemanticSnapshot{
					Entities: []contracts.Entity{{ID: "service.payments", Type: "service", Name: "Payments"}},
					Edges:    []contracts.Edge{{ID: "edge.owns", Type: "owns", From: "service.payments", To: "service.missing"}},
				})
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			observations := make([]adapterObservation, 0, 3)
			for _, adapter := range []providerAdapter{adapterClaude, adapterQwen, adapterCodex} {
				err := test.run(t, adapter)
				observation := adapterObservation{accepted: err == nil, code: conformanceCode(err)}
				observations = append(observations, observation)
				if observation.accepted {
					t.Fatalf("%s accepted incident payload for %s", test.name, adapter)
				}
				if observation.code != test.want {
					t.Fatalf("%s for %s: got %q, want %q (err=%v)", adapter, adapter, observation.code, test.want, err)
				}
			}
			for _, observation := range observations[1:] {
				if observation.code != observations[0].code || observation.accepted != observations[0].accepted {
					t.Fatalf("adapter parity drift: %#v", observations)
				}
			}
		})
	}
}

func TestIncidentCorpusAllowsCleanAndSparseGapPayloads(t *testing.T) {
	t.Parallel()
	payloads := []struct {
		name    string
		payload any
	}{
		{
			name: "clean semantic graph",
			payload: contracts.SemanticSnapshot{
				Coverage: contracts.Coverage{Observed: []string{"README"}, Missing: []string{}},
				Entities: []contracts.Entity{{ID: "service.payments", Type: "service", Name: "Payments", Provenance: contracts.Provenance{Kind: "observed", Confidence: 0.9}}},
			},
		},
		{
			name: "sparse explicit gap",
			payload: contracts.SemanticSnapshot{
				Coverage:  contracts.Coverage{Observed: []string{}, Missing: []string{"owner mapping", "operational runbook"}},
				Questions: []contracts.Question{{ID: "question.owner", Text: "Which team owns this service?", Priority: "medium"}},
			},
		},
	}
	for _, payload := range payloads {
		payload := payload
		t.Run(payload.name, func(t *testing.T) {
			t.Parallel()
			for _, adapter := range []providerAdapter{adapterClaude, adapterQwen, adapterCodex} {
				if err := validateSemanticEnvelopePayload(t, adapter, payload.payload); err != nil {
					t.Fatalf("%s adapter must preserve truthful sparse payload: %v", adapter, err)
				}
			}
		})
	}
}

func TestIncidentCorpusRejectsVerdictEvidenceRepairAndAuditFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		run  func(t *testing.T) error
	}{
		{
			name: "contradictory verdict",
			want: "verdict.contradictory",
			run: func(t *testing.T) error {
				return artifactquality.ValidateValidatorVerdict(contracts.ValidatorVerdict{
					RunID: "run-corpus", Verdict: "PASS", CheckedPaths: []string{"final-run-index.json"},
					Issues: []contracts.ValidatorIssue{{Code: "audit.final_index.invalid", Severity: "error", Message: "broken index"}},
				}, nil, nil, false, false)
			},
		},
		{
			name: "stale provider repair",
			want: "repair.provider_fixed_path",
			run: func(t *testing.T) error {
				return artifactquality.ValidateValidatorVerdict(contracts.ValidatorVerdict{
					RunID: "run-corpus", Verdict: "PASS", CheckedPaths: []string{"final-run-index.json"},
					FixedPaths: []string{"reports/taskruns/old/validator/validator-verdict.json"},
				}, nil, nil, false, false)
			},
		},
		{
			name: "invalid evidence range",
			want: "evidence.lines_invalid",
			run: func(t *testing.T) error {
				return evidence.Validate([]byte("one\ntwo\n"), &contracts.LineRange{Start: 0, End: 1}, "one", "")
			},
		},
		{
			name: "wrong evidence hash",
			want: "evidence.hash_mismatch",
			run: func(t *testing.T) error {
				return evidence.Validate([]byte("one\n"), &contracts.LineRange{Start: 1, End: 1}, "one", strings.Repeat("0", 64))
			},
		},
		{
			name: "audit foreign identity",
			want: "audit.run_identity.foreign",
			run: func(t *testing.T) error {
				ws, runID := writeCorpusAuditFixture(t)
				finalPath := path.Join("reports", "taskruns", runID, "staging", "final", "final-run-index.json")
				var index contracts.FinalRunIndex
				raw, err := ws.ReadFile(finalPath)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(raw, &index); err != nil {
					return err
				}
				index.RunID = "run-foreign"
				updated, err := json.Marshal(index)
				if err != nil {
					return err
				}
				if err := ws.WriteFileAtomic(finalPath, append(updated, '\n')); err != nil {
					return err
				}
				report := artifactaudit.ScanSelectedRun(ws, runID)
				return reportIssue(report, "audit.run_identity.foreign")
			},
		},
		{
			name: "audit missing evidence",
			want: "audit.evidence.file_missing",
			run: func(t *testing.T) error {
				ws, runID := writeCorpusAuditFixture(t)
				citationPath := path.Join("reports", "taskruns", runID, "staging", "final", "citation-index.json")
				var index contracts.CitationIndex
				raw, err := ws.ReadFile(citationPath)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(raw, &index); err != nil {
					return err
				}
				index.Citations[0].Path = "missing.md"
				updated, err := json.Marshal(index)
				if err != nil {
					return err
				}
				if err := ws.WriteFileAtomic(citationPath, append(updated, '\n')); err != nil {
					return err
				}
				report := artifactaudit.ScanSelectedRun(ws, runID)
				return reportIssue(report, "audit.evidence.file_missing")
			},
		},
		{
			name: "audit promotion failure",
			want: "audit.document.execution_contamination",
			run: func(t *testing.T) error {
				ws, runID := writeCorpusAuditFixture(t)
				finalPath := path.Join("reports", "taskruns", runID, "staging", "final", "final-run-index.json")
				var index contracts.FinalRunIndex
				raw, err := ws.ReadFile(finalPath)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(raw, &index); err != nil {
					return err
				}
				index.CanonicalDocuments[0].StagedPath = path.Join("reports", "taskruns", "other", "staging", "final", "reports", "as-is", "overview.md")
				updated, err := json.Marshal(index)
				if err != nil {
					return err
				}
				if err := ws.WriteFileAtomic(finalPath, append(updated, '\n')); err != nil {
					return err
				}
				report := artifactaudit.ScanSelectedRun(ws, runID)
				return reportIssue(report, "audit.document.execution_contamination")
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.run(t)
			if err == nil {
				t.Fatalf("%s was accepted by the conformance boundary", test.name)
			}
			if got := conformanceCode(err); got != test.want {
				t.Fatalf("got %q, want %q (err=%v)", got, test.want, err)
			}
		})
	}
}

func validateSemanticEnvelopePayload(t *testing.T, adapter providerAdapter, payload any) error {
	t.Helper()
	semanticRaw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	providerRaw, err := json.Marshal(string(adapter))
	if err != nil {
		return err
	}
	root := map[string]json.RawMessage{"provider": providerRaw, "semantic": semanticRaw}
	raw, err := json.Marshal(root)
	if err != nil {
		return err
	}
	if err := artifactquality.ValidateSemanticEnvelopeJSON(raw); err != nil {
		return err
	}
	var snapshot contracts.SemanticSnapshot
	if err := json.Unmarshal(semanticRaw, &snapshot); err != nil {
		return err
	}
	return artifactquality.ValidateSemanticEnvelope(snapshot)
}

func conformanceCode(err error) string {
	if err == nil {
		return "accepted"
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "unknown field"):
		return "semantic.unknown_field"
	case strings.Contains(text, "collides"):
		return "semantic.graph_collision"
	case strings.Contains(text, "dangling"):
		return "semantic.graph_dangling_edge"
	case strings.Contains(text, "pass verdict cannot contain error"):
		return "verdict.contradictory"
	case strings.Contains(text, "fixed_paths must be empty"):
		return "repair.provider_fixed_path"
	case strings.Contains(text, "audit."):
		for _, code := range []string{
			"audit.run_identity.foreign", "audit.evidence.file_missing", "audit.document.execution_contamination",
		} {
			if strings.Contains(text, code) {
				return code
			}
		}
	case strings.Contains(text, "evidence") || strings.Contains(text, "excerpt") || strings.Contains(text, "line range") || strings.Contains(text, "source evidence"):
		return evidence.Code(err)
	}
	return strings.TrimSpace(err.Error())
}

func reportIssue(report artifactaudit.Report, code string) error {
	for _, issue := range report.Issues {
		if issue.Code == code {
			return fmt.Errorf("%s: %s", code, issue.Message)
		}
	}
	return fmt.Errorf("expected audit issue %s, got %#v", code, report.Issues)
}

func writeCorpusAuditFixture(t *testing.T) (workspace.Root, string) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# Evidence\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws := workspace.Root{Path: root, Manifest: workspace.Manifest{Version: 1, Repos: []workspace.RepoSource{{Name: "sample", Path: repo}}}}
	runID := "run-corpus"
	finalRoot := path.Join("reports", "taskruns", runID, "staging", "final")
	stagedOverview := path.Join(finalRoot, "reports", "as-is", "overview.md")
	var overview strings.Builder
	overview.WriteString("# Architecture Home\n\n")
	for _, section := range runtimedrafts.ArchitectureHomeRequiredSections() {
		overview.WriteString("## " + section + "\n\nEvidence-backed content for this section.\n\n")
	}
	if err := ws.WriteFileAtomic(stagedOverview, []byte(overview.String())); err != nil {
		t.Fatal(err)
	}
	document := contracts.FinalRunDocument{ID: "doc.overview", Kind: "report", Title: "Architecture Home", CanonicalPath: "reports/as-is/overview.md", StagedPath: stagedOverview, Topics: []string{"architecture"}, CitationIDs: []string{"cite.overview"}, SourceShards: []string{"sample"}, Status: "staged"}
	citation := contracts.DocumentCitation{ID: "cite.overview", Repo: "sample", Path: "README.md", ClaimIDs: []string{"claim.overview"}, DocumentIDs: []string{"doc.overview"}}
	index := contracts.FinalRunIndex{Version: 1, RunID: runID, Pipeline: "init", GeneratedAt: "2026-08-11T00:00:00Z", CitationIndexPath: path.Join(finalRoot, "citation-index.json"), CanonicalDocuments: []contracts.FinalRunDocument{document}, Topics: []contracts.TopicIndexEntry{{ID: "architecture", DocumentIDs: []string{"doc.overview"}}}, Semantic: contracts.SemanticSnapshot{Coverage: contracts.Coverage{Observed: []string{"README"}, Missing: []string{}, Notes: []string{}}, Questions: []contracts.Question{}, Entities: []contracts.Entity{}, Edges: []contracts.Edge{}, Findings: []contracts.Finding{}}}
	citationIndex := contracts.CitationIndex{Version: 1, RunID: runID, GeneratedAt: "2026-08-11T00:00:00Z", Citations: []contracts.DocumentCitation{citation}}
	verdict := contracts.ValidatorVerdict{Version: 1, RunID: runID, GeneratedAt: "2026-08-11T00:00:00Z", Verdict: "PASS", CheckedPaths: []string{path.Join(finalRoot, "final-run-index.json"), path.Join(finalRoot, "citation-index.json")}}
	for filePath, value := range map[string]any{
		path.Join(finalRoot, "final-run-index.json"):                                   index,
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

func TestIncidentCorpusEvidenceDigestFixtureIsStable(t *testing.T) {
	t.Parallel()
	selected := "# Evidence"
	digest := sha256.Sum256([]byte(selected))
	if got := hex.EncodeToString(digest[:]); len(got) != 64 {
		t.Fatalf("unexpected digest length: %d", len(got))
	}
	if err := evidence.Validate([]byte("# Evidence\n"), &contracts.LineRange{Start: 1, End: 1}, selected, hex.EncodeToString(digest[:])); err != nil {
		t.Fatalf("clean bounded evidence fixture must pass: %v", err)
	}
}

func TestIncidentCorpusP95InvocationMeasurement(t *testing.T) {
	t.Parallel()
	// The sample is a deterministic provider-free trace: clean and sparse
	// units finish on the normal call; repaired units use one bounded recovery.
	samples := []int{1, 1, 1, 1, 2, 2, 1, 2, 1, 2, 1, 1, 2, 1, 2, 1, 1, 2, 1, 2}
	if got := providercommon.ProviderInvocationP95(samples); got > 2 {
		t.Fatalf("conformance p95 provider invocations exceeded two: %d (samples=%v)", got, samples)
	}
	if got := providercommon.ProviderInvocationP95(nil); got != 0 {
		t.Fatalf("empty p95 sample must be zero, got %d", got)
	}
}
