package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/model"
	"github.com/GrinRus/ProvenArch/internal/reports"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestReadShardDocumentRejectsPathTraversalOutsideArtifactRoot(t *testing.T) {
	t.Parallel()

	artifactRoot := t.TempDir()
	manifest := contracts.ShardPackManifest{
		ArtifactRoot: artifactRoot,
	}
	_, err := readShardDocument(manifest, contracts.AuthoredDocument{
		ID:   "doc.escape",
		Path: "../outside.md",
	}, "")
	if err == nil {
		t.Fatalf("expected path traversal error")
	}
	if !strings.Contains(err.Error(), "escapes artifact_root") {
		t.Fatalf("expected escapes artifact_root error, got %v", err)
	}
}

func TestReadShardDocumentReadsRelativePathWithinArtifactRoot(t *testing.T) {
	t.Parallel()

	artifactRoot := t.TempDir()
	relPath := filepath.Join("nested", "report.md")
	absPath := filepath.Join(artifactRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("mkdir artifact root: %v", err)
	}
	if err := os.WriteFile(absPath, []byte("report-content"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	manifest := contracts.ShardPackManifest{
		ArtifactRoot: artifactRoot,
	}
	content, err := readShardDocument(manifest, contracts.AuthoredDocument{
		ID:   "doc.safe",
		Path: filepath.ToSlash(relPath),
	}, "")
	if err != nil {
		t.Fatalf("read shard document: %v", err)
	}
	if content != "report-content" {
		t.Fatalf("unexpected content %q", content)
	}
}

func TestReadShardDocumentResolvesWorkspaceRelativeArtifactRoot(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	artifactRel := filepath.Join("reports", "taskruns", "run-1", "staging", "shards", "sample")
	artifactRoot := filepath.Join(workspaceRoot, artifactRel)
	relPath := filepath.Join("nested", "report.md")
	absPath := filepath.Join(artifactRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("mkdir artifact root: %v", err)
	}
	if err := os.WriteFile(absPath, []byte("relative-report"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	manifest := contracts.ShardPackManifest{
		ArtifactRoot: filepath.ToSlash(artifactRel),
	}
	content, err := readShardDocument(manifest, contracts.AuthoredDocument{
		ID:   "doc.relative",
		Path: filepath.ToSlash(relPath),
	}, workspaceRoot)
	if err != nil {
		t.Fatalf("read shard document with relative artifact_root: %v", err)
	}
	if content != "relative-report" {
		t.Fatalf("unexpected content %q", content)
	}
}

func TestLoadShardPackManifestFromRootRejectsWorkspaceRelativeDocumentPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	raw := []byte(`{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-iac",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac",
  "documents": [
    {
      "id": "doc.iac",
      "kind": "analysis",
      "title": "IAC Overview",
      "path": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac/iac-overview.md",
      "canonical_path": "reports/as-is/bank-of-anthos/iac-overview.md",
      "topics": ["iac"],
      "citation_ids": ["cite.1"]
    }
  ],
  "citations": [
    {
      "id": "cite.1",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.1"],
      "document_ids": ["doc.iac"]
    }
  ],
  "semantic": {
    "coverage": {"observed": [], "missing": [], "notes": []},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`)
	if err := os.WriteFile(filepath.Join(root, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write shard-pack manifest: %v", err)
	}

	_, _, err := loadShardPackManifestFromRoot(root)
	if err == nil {
		t.Fatalf("expected compile-time manifest guard failure")
	}
	if !strings.Contains(err.Error(), "artifact_root-relative") {
		t.Fatalf("expected artifact_root-relative error, got %v", err)
	}
}

func TestCollectAuthoredStageDocumentsMergesByCanonicalPath(t *testing.T) {
	t.Parallel()

	firstRoot := t.TempDir()
	secondRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(firstRoot, "domain.md"), []byte("# Domain A\n"), 0o644); err != nil {
		t.Fatalf("write first shard doc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secondRoot, "domain.md"), []byte("## Addendum B\n"), 0o644); err != nil {
		t.Fatalf("write second shard doc: %v", err)
	}

	manifests := []contracts.ShardPackManifest{
		{
			ArtifactRoot: firstRoot,
			Documents: []contracts.AuthoredDocument{
				{
					ID:            "doc.domain.a",
					Kind:          "agent-output",
					Title:         "Domain Report",
					Path:          "domain.md",
					CanonicalPath: "reports/agent-outputs/domains/payments.md",
				},
			},
		},
		{
			ArtifactRoot: secondRoot,
			Documents: []contracts.AuthoredDocument{
				{
					ID:            "doc.domain.b",
					Kind:          "agent-output",
					Title:         "Domain Report",
					Path:          "domain.md",
					CanonicalPath: "reports/agent-outputs/domains/payments.md",
				},
			},
		},
	}

	documents, err := collectAuthoredStageDocuments(manifests, "")
	if err != nil {
		t.Fatalf("collect authored docs: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected one merged document, got %d", len(documents))
	}
	if documents[0].CanonicalPath != "reports/agent-outputs/domains/payments.md" {
		t.Fatalf("unexpected canonical path %q", documents[0].CanonicalPath)
	}
	if !strings.Contains(documents[0].Content, "# Domain A") || !strings.Contains(documents[0].Content, "## Addendum B") {
		t.Fatalf("expected merged content from both shard documents, got %q", documents[0].Content)
	}
	if !strings.Contains(documents[0].Content, "\n\n---\n\n") {
		t.Fatalf("expected shard merge separator in merged content")
	}
}

func TestPromoteValidatedArtifactsRejectsFailVerdict(t *testing.T) {
	t.Parallel()

	execution := &pipelineExecution{
		pipelineSemanticDocflowState: pipelineSemanticDocflowState{
			finalRunIndex:    &contracts.FinalRunIndex{},
			validatorVerdict: &contracts.ValidatorVerdict{Verdict: "FAIL"},
		},
	}
	err := execution.promoteValidatedArtifacts()
	if err == nil {
		t.Fatalf("expected promotion failure when validator verdict is FAIL")
	}
	if !strings.Contains(err.Error(), "validator verdict is FAIL") {
		t.Fatalf("expected validator verdict failure, got %v", err)
	}
}

func TestPromoteValidatedArtifactsRemovesStaleManagedCanonicalFiles(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	ws := workspace.Root{Path: workspaceRoot}

	stagedPath := "reports/taskruns/run-1/staging/final/reports/as-is/overview.md"
	absStagedPath := filepath.Join(workspaceRoot, filepath.FromSlash(stagedPath))
	if err := os.MkdirAll(filepath.Dir(absStagedPath), 0o755); err != nil {
		t.Fatalf("mkdir staged path: %v", err)
	}
	if err := os.WriteFile(absStagedPath, []byte("# Current Overview\n"), 0o644); err != nil {
		t.Fatalf("write staged overview: %v", err)
	}

	staleReportPath := filepath.Join(workspaceRoot, "reports", "as-is", "services", "legacy.md")
	if err := os.MkdirAll(filepath.Dir(staleReportPath), 0o755); err != nil {
		t.Fatalf("mkdir stale report dir: %v", err)
	}
	if err := os.WriteFile(staleReportPath, []byte("# Legacy\n"), 0o644); err != nil {
		t.Fatalf("write stale report: %v", err)
	}

	staleProposalPath := filepath.Join(workspaceRoot, "proposals", "proposal-legacy", "proposal.md")
	if err := os.MkdirAll(filepath.Dir(staleProposalPath), 0o755); err != nil {
		t.Fatalf("mkdir stale proposal dir: %v", err)
	}
	if err := os.WriteFile(staleProposalPath, []byte("# Legacy Proposal\n"), 0o644); err != nil {
		t.Fatalf("write stale proposal: %v", err)
	}

	execution := &pipelineExecution{
		workspace: ws,
		store:     model.NewStore(ws),
		compiler:  reports.NewCompiler(ws),
		pipelineRunProgressState: pipelineRunProgressState{
			stepStatus: RunInfo{CurrentStep: "init.step4.proposals"},
		},
		pipelineSemanticDocflowState: pipelineSemanticDocflowState{
			finalRunIndex: &contracts.FinalRunIndex{
				RunID: "run-1",
				CanonicalDocuments: []contracts.FinalRunDocument{
					{
						ID:            "doc.overview",
						Kind:          "report",
						Title:         "System Overview",
						CanonicalPath: "reports/as-is/overview.md",
						StagedPath:    stagedPath,
					},
				},
				Semantic: contracts.SemanticSnapshot{},
			},
			validatorVerdict: &contracts.ValidatorVerdict{Verdict: "PASS"},
		},
	}

	if err := execution.promoteValidatedArtifacts(); err != nil {
		t.Fatalf("promote validated artifacts: %v", err)
	}
	if _, err := os.Stat(staleReportPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale report removed, stat err=%v", err)
	}
	if _, err := os.Stat(staleProposalPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale proposal removed, stat err=%v", err)
	}
	content, err := os.ReadFile(filepath.Join(workspaceRoot, "reports", "as-is", "overview.md"))
	if err != nil {
		t.Fatalf("read promoted overview: %v", err)
	}
	if !strings.Contains(string(content), "Current Overview") {
		t.Fatalf("expected promoted overview content, got %q", string(content))
	}
}

func TestStageProposalDraftOutputsUpdatesFinalRunIndex(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	ws := workspace.Root{Path: workspaceRoot}
	runID := "run-1"
	stagedOverview := filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "final", "reports", "as-is", "overview.md"))
	if err := ws.WriteFile(stagedOverview, []byte("# Overview\n")); err != nil {
		t.Fatalf("write staged overview: %v", err)
	}
	draftRoot := filepath.Join(workspaceRoot, "reports", "taskruns", runID, "staging", "drafts", "step4_proposals")
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "proposal.md"), []byte("# Proposal\n\nEvidence-backed recommendation.\n"), 0o644); err != nil {
		t.Fatalf("write proposal draft: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "changelog.md"), []byte("# Changelog\n\n- Added evidence-backed proposal.\n"), 0o644); err != nil {
		t.Fatalf("write changelog draft: %v", err)
	}

	execution := &pipelineExecution{
		runID:     runID,
		pipeline:  PipelineInit,
		workspace: ws,
		pipelineRunProgressState: pipelineRunProgressState{
			stepStatus: RunInfo{CurrentStep: "init.step4.proposals"},
		},
		pipelineSemanticDocflowState: pipelineSemanticDocflowState{
			shardPacks: []contracts.ShardPackManifest{
				{
					ShardID: "payments",
					Documents: []contracts.AuthoredDocument{
						{
							ID:            "doc.overview",
							Kind:          "report",
							Title:         "Overview",
							Path:          "overview.md",
							CanonicalPath: "reports/as-is/overview.md",
							CitationIDs:   []string{"cite.overview"},
						},
					},
					Citations: []contracts.DocumentCitation{
						{
							ID:          "cite.overview",
							Repo:        "payments",
							Path:        "README.md",
							DocumentIDs: []string{"doc.overview"},
						},
					},
				},
			},
			citationIndex: &contracts.CitationIndex{
				Version:     1,
				RunID:       runID,
				GeneratedAt: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
				Citations: []contracts.DocumentCitation{
					{
						ID:          "cite.overview",
						Repo:        "payments",
						Path:        "README.md",
						DocumentIDs: []string{"doc.overview"},
					},
				},
			},
			finalRunIndex: &contracts.FinalRunIndex{
				Version:           1,
				RunID:             runID,
				Pipeline:          string(PipelineInit),
				GeneratedAt:       time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
				CitationIndexPath: runtimeCitationIndexPath(runID),
				CanonicalDocuments: []contracts.FinalRunDocument{
					{
						ID:            "doc.overview",
						Kind:          "report",
						Title:         "Overview",
						CanonicalPath: "reports/as-is/overview.md",
						StagedPath:    stagedOverview,
						CitationIDs:   []string{"cite.overview"},
						SourceShards:  []string{"payments"},
						Status:        "staged",
					},
				},
				Semantic: contracts.SemanticSnapshot{
					Coverage:  contracts.Coverage{Observed: []string{}, Missing: []string{}, Notes: []string{}},
					Questions: []contracts.Question{},
					Entities:  []contracts.Entity{},
					Edges:     []contracts.Edge{},
					Findings:  []contracts.Finding{},
				},
			},
		},
		pipelineDraftState: pipelineDraftState{
			proposalsDraftRoot: draftRoot,
			proposalsDraftManifest: &runtimeDraftManifest{
				Version:      1,
				RunID:        runID,
				StepID:       "init.step4.proposals",
				StepContract: "proposals",
				AgentRole:    "architect",
				Outputs: []runtimeDraftOutput{
					{Path: "proposal.md", CanonicalPath: "proposals/runtime-recommendations.md", Kind: "proposal", Title: "Runtime Recommendations"},
					{Path: "changelog.md", CanonicalPath: "reports/changelog/runtime-proposals.md", Kind: "changelog", Title: "Runtime Proposal Changelog"},
				},
			},
		},
	}

	if err := execution.stageProposalDraftOutputsForFinalIndex(); err != nil {
		t.Fatalf("stage proposal drafts: %v", err)
	}
	raw, err := ws.ReadFile(runtimeFinalRunIndexPath(runID))
	if err != nil {
		t.Fatalf("read final run index: %v", err)
	}
	finalIndex, err := contracts.ParseFinalRunIndex(raw)
	if err != nil {
		t.Fatalf("parse final run index: %v", err)
	}
	if !finalIndexHasCanonicalPath(finalIndex, "proposals/runtime-recommendations.md") {
		t.Fatalf("expected proposal in final index: %#v", finalIndex.CanonicalDocuments)
	}
	if !finalIndexHasCanonicalPath(finalIndex, "reports/changelog/runtime-proposals.md") {
		t.Fatalf("expected changelog in final index: %#v", finalIndex.CanonicalDocuments)
	}
	if _, err := ws.ReadFile(filepath.ToSlash(filepath.Join(runtimeFinalArtifactRoot(runID), "reports", "changelog", "runtime-proposals.md"))); err != nil {
		t.Fatalf("expected staged changelog draft: %v", err)
	}
}

func TestValidateStagedArtifactsReportsMissingStagedDocument(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	execution := &pipelineExecution{
		workspace: workspace.Root{Path: workspaceRoot},
		pipelineSemanticDocflowState: pipelineSemanticDocflowState{
			finalRunIndex: &contracts.FinalRunIndex{
				RunID: "run-1",
				CanonicalDocuments: []contracts.FinalRunDocument{
					{
						ID:            "doc.report",
						Kind:          "report",
						CanonicalPath: "reports/as-is/overview.md",
						StagedPath:    "reports/taskruns/run-1/staging/final/reports/as-is/overview.md",
					},
				},
				Topics: []contracts.TopicIndexEntry{
					{ID: "as-is", DocumentIDs: []string{"doc.report"}},
				},
			},
			citationIndex: &contracts.CitationIndex{
				RunID:     "run-1",
				Citations: []contracts.DocumentCitation{},
			},
			shardPacks: []contracts.ShardPackManifest{
				{
					ShardID:      "payments",
					Documents:    []contracts.AuthoredDocument{{ID: "doc.report", CanonicalPath: "reports/as-is/overview.md", CitationIDs: []string{"cite.1"}}},
					Citations:    []contracts.DocumentCitation{{ID: "cite.1", Repo: "payments-service", Path: "README.md", ClaimIDs: []string{"claim.1"}, DocumentIDs: []string{"doc.report"}}},
					ArtifactRoot: workspaceRoot,
				},
			},
		},
	}

	issues := execution.validateStagedArtifacts()
	if len(issues) == 0 {
		t.Fatalf("expected missing staged document issue")
	}
	found := false
	for _, issue := range issues {
		if issue.Code == "missing_staged_document" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing_staged_document issue, got %#v", issues)
	}
}

func finalIndexHasCanonicalPath(index contracts.FinalRunIndex, canonicalPath string) bool {
	for _, document := range index.CanonicalDocuments {
		if document.CanonicalPath == canonicalPath {
			return true
		}
	}
	return false
}

func TestValidateStagedArtifactsDetectsCitationAndTopicIssues(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	stagedPath := "reports/taskruns/run-1/staging/final/reports/findings/findings.md"
	absStagedPath := filepath.Join(workspaceRoot, filepath.FromSlash(stagedPath))
	if err := os.MkdirAll(filepath.Dir(absStagedPath), 0o755); err != nil {
		t.Fatalf("mkdir staged path: %v", err)
	}
	if err := os.WriteFile(absStagedPath, []byte("# Findings\n"), 0o644); err != nil {
		t.Fatalf("write staged document: %v", err)
	}

	execution := &pipelineExecution{
		workspace: workspace.Root{Path: workspaceRoot},
		pipelineSemanticDocflowState: pipelineSemanticDocflowState{
			finalRunIndex: &contracts.FinalRunIndex{
				RunID: "run-1",
				CanonicalDocuments: []contracts.FinalRunDocument{
					{
						ID:            "doc.findings",
						Kind:          "report",
						CanonicalPath: "reports/findings/findings.md",
						StagedPath:    stagedPath,
						CitationIDs:   []string{},
					},
				},
				Topics: []contracts.TopicIndexEntry{
					{ID: "topic.dup", DocumentIDs: []string{"doc.findings"}},
					{ID: "topic.dup", DocumentIDs: []string{"doc.findings"}},
					{ID: "topic.broken", DocumentIDs: []string{"doc.unknown"}},
				},
			},
			citationIndex: &contracts.CitationIndex{
				RunID: "run-1",
				Citations: []contracts.DocumentCitation{
					{
						ID:       "cite.1",
						Repo:     "payments-service",
						Path:     "README.md",
						ClaimIDs: []string{"claim.same"},
					},
					{
						ID:       "cite.2",
						Repo:     "payments-service",
						Path:     "service.yaml",
						ClaimIDs: []string{"claim.same"},
					},
				},
			},
			shardPacks: []contracts.ShardPackManifest{
				{
					ShardID:      "payments",
					ArtifactRoot: workspaceRoot,
					Documents: []contracts.AuthoredDocument{
						{ID: "doc.findings", CanonicalPath: "reports/findings/findings.md"},
					},
					Citations: []contracts.DocumentCitation{
						{ID: "cite.1", Repo: "payments-service", Path: "README.md", ClaimIDs: []string{"claim.same"}, DocumentIDs: []string{"doc.findings"}},
						{ID: "cite.2", Repo: "payments-service", Path: "service.yaml", ClaimIDs: []string{"claim.same"}, DocumentIDs: []string{"doc.findings"}},
					},
				},
			},
		},
	}

	issues := execution.validateStagedArtifacts()
	if len(issues) == 0 {
		t.Fatalf("expected validation issues")
	}
	seen := map[string]bool{}
	for _, issue := range issues {
		seen[issue.Code] = true
	}
	for _, expected := range []string{"missing_document_citations", "duplicate_claim_id", "duplicate_topic_id", "broken_topic_reference"} {
		if !seen[expected] {
			t.Fatalf("expected issue code %q, got %#v", expected, issues)
		}
	}
}

func TestDocflowIndexesUseConsistentManifestDocumentIDs(t *testing.T) {
	t.Parallel()

	manifests := []contracts.ShardPackManifest{
		{
			ShardID:      "payments",
			ArtifactRoot: "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments",
			Documents: []contracts.AuthoredDocument{
				{
					ID:            "doc.payments.overview",
					Kind:          "report",
					Title:         "Payments Overview",
					Path:          "overview.md",
					CanonicalPath: "reports/as-is/overview.md",
					CitationIDs:   []string{"cite.payments.readme"},
				},
			},
			Citations: []contracts.DocumentCitation{
				{
					ID:          "cite.payments.readme",
					Repo:        "payments-service",
					Path:        "README.md",
					ClaimIDs:    []string{"claim.payments.readme"},
					DocumentIDs: []string{"doc.payments.overview"},
				},
			},
			Semantic: contracts.SemanticSnapshot{
				Coverage:  contracts.Coverage{Observed: []string{"services"}},
				Questions: []contracts.Question{},
				Entities:  []contracts.Entity{},
				Edges:     []contracts.Edge{},
				Findings:  []contracts.Finding{},
			},
		},
	}
	documentInfos := aggregateDocumentInfos(manifests)
	citationIndex := aggregateCitationIndex("run-1", time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC), manifests, documentInfos)
	finalIndex, err := buildFinalRunIndex(
		"run-1",
		string(PipelineInit),
		time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC),
		[]Artifact{{
			Path:  "reports/taskruns/run-1/staging/final/reports/as-is/overview.md",
			Kind:  "report",
			Label: "Payments Overview",
		}},
		manifests,
		documentInfos,
		citationIndex,
		contracts.SemanticSnapshot{Coverage: contracts.Coverage{}, Questions: []contracts.Question{}, Entities: []contracts.Entity{}, Edges: []contracts.Edge{}, Findings: []contracts.Finding{}},
	)
	if err != nil {
		t.Fatalf("build final run index: %v", err)
	}
	if got, want := finalIndex.CanonicalDocuments[0].ID, "doc.payments.overview"; got != want {
		t.Fatalf("unexpected final run document id: got=%q want=%q", got, want)
	}
	if got, want := citationIndex.Citations[0].DocumentIDs, []string{"doc.payments.overview"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected citation document ids: got=%v want=%v", got, want)
	}
}

func TestNormalizeSemanticSnapshotDedupesRepoAliasEntitiesAndRewritesReferences(t *testing.T) {
	t.Parallel()

	resolver := newSemanticRepoAliasResolver(
		map[string]string{"bank-of-anthos": "/tmp/repos/bank-of-anthos-7fb01b96709b"},
		nil,
	)
	snapshot := normalizeSemanticSnapshot(contracts.SemanticSnapshot{
		Coverage: contracts.Coverage{Missing: []string{"owner mappings"}},
		Questions: []contracts.Question{
			{ID: "q.bank.owner", Text: "Who owns the bank service?", RelatedIDs: []string{"svc.bank_of_anthos"}},
		},
		Entities: []contracts.Entity{
			{
				ID:   "svc.bank-of-anthos",
				Type: "service",
				Name: "bank-of-anthos",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.7,
					Evidence:   []contracts.Evidence{{Repo: "bank-of-anthos", Path: "src/main.go"}},
				},
			},
			{
				ID:   "svc.bank_of_anthos",
				Type: "service",
				Name: "bank-of-anthos",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.8,
					Evidence:   []contracts.Evidence{{Repo: "bank-of-anthos-7fb01b96709b", Path: "src/main.go"}},
				},
			},
		},
		Edges: []contracts.Edge{
			{
				ID:   "edge.dep",
				Type: "depends_on",
				From: "svc.bank_of_anthos",
				To:   "svc.ledger",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.6,
					Evidence:   []contracts.Evidence{{Repo: "bank-of-anthos-7fb01b96709b", Path: "src/main.go"}},
				},
			},
		},
		Findings: []contracts.Finding{
			{
				ID:         "finding.bank.owner",
				Severity:   "medium",
				Title:      "Missing owner mapping",
				RelatedIDs: []string{"svc.bank_of_anthos"},
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.6,
					Evidence:   []contracts.Evidence{{Repo: "bank-of-anthos-7fb01b96709b", Path: "src/main.go"}},
				},
			},
		},
	}, resolver)

	if got, want := len(snapshot.Entities), 1; got != want {
		t.Fatalf("expected deduped entities, got=%d want=%d", got, want)
	}
	winnerID := snapshot.Entities[0].ID
	if winnerID != "svc.bank-of-anthos" {
		t.Fatalf("expected stable winning entity id, got %q", winnerID)
	}
	if got := snapshot.Findings[0].RelatedIDs; len(got) != 1 || got[0] != winnerID {
		t.Fatalf("expected finding related_ids to be rewritten to %q, got %v", winnerID, got)
	}
	if got := snapshot.Questions[0].RelatedIDs; len(got) != 1 || got[0] != winnerID {
		t.Fatalf("expected question related_ids to be rewritten to %q, got %v", winnerID, got)
	}
	if got := snapshot.Edges[0].From; got != winnerID {
		t.Fatalf("expected edge.from to be rewritten to %q, got %q", winnerID, got)
	}
	if got := snapshot.Entities[0].Provenance.Evidence[0].Repo; got != "bank-of-anthos" {
		t.Fatalf("expected repo alias normalization to logical repo scope, got %q", got)
	}
}

func TestNormalizeSemanticSnapshotResolvesUniqueExtensionlessEvidencePaths(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeRepoFile := func(rel string) {
		t.Helper()
		abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("create repo fixture dir: %v", err)
		}
		if err := os.WriteFile(abs, []byte("fixture"), 0o644); err != nil {
			t.Fatalf("write repo fixture file: %v", err)
		}
	}
	writeRepoFile("src/utils/db/primary-store.ts")
	writeRepoFile("src/utils/db/cache-store.ts")
	writeRepoFile("src/config.ts")
	writeRepoFile("src/config.go")

	resolvedRepos := map[string]string{"sample-repo": repoRoot}
	repoAliases := newSemanticRepoAliasResolver(resolvedRepos, nil)
	evidencePaths := newSemanticEvidencePathResolver(resolvedRepos, repoAliases)
	snapshot := normalizeSemanticSnapshot(contracts.SemanticSnapshot{
		Entities: []contracts.Entity{
			{
				ID:   "db.primary",
				Type: "datastore",
				Name: "Primary Store",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.8,
					Evidence:   []contracts.Evidence{{Repo: "sample-repo", Path: "src/utils/db/primary-store"}},
				},
			},
			{
				ID:   "svc.config",
				Type: "service",
				Name: "Config",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.7,
					Evidence:   []contracts.Evidence{{Repo: "sample-repo", Path: "src/config"}},
				},
			},
		},
		Edges: []contracts.Edge{
			{
				ID:   "edge.cache",
				Type: "reads",
				From: "svc.config",
				To:   "db.cache",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.7,
					Evidence:   []contracts.Evidence{{Repo: "sample-repo", Path: "src/utils/db/cache-store"}},
				},
			},
		},
		Findings: []contracts.Finding{
			{
				ID:       "finding.cache",
				Severity: "medium",
				Title:    "Cache store dependency",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.7,
					Evidence:   []contracts.Evidence{{Repo: "sample-repo", Path: "src/utils/db/cache-store"}},
				},
			},
		},
	}, repoAliases, evidencePaths)

	entitiesByID := map[string]contracts.Entity{}
	for _, entity := range snapshot.Entities {
		entitiesByID[entity.ID] = entity
	}
	if got, want := entitiesByID["db.primary"].Provenance.Evidence[0].Path, "src/utils/db/primary-store.ts"; got != want {
		t.Fatalf("expected unique extensionless entity evidence path to resolve, got=%q want=%q", got, want)
	}
	if got, want := entitiesByID["svc.config"].Provenance.Evidence[0].Path, "src/config"; got != want {
		t.Fatalf("expected ambiguous extensionless entity evidence path to stay unchanged, got=%q want=%q", got, want)
	}
	if got, want := snapshot.Edges[0].Provenance.Evidence[0].Path, "src/utils/db/cache-store.ts"; got != want {
		t.Fatalf("expected unique extensionless edge evidence path to resolve, got=%q want=%q", got, want)
	}
	if got, want := snapshot.Findings[0].Provenance.Evidence[0].Path, "src/utils/db/cache-store.ts"; got != want {
		t.Fatalf("expected unique extensionless finding evidence path to resolve, got=%q want=%q", got, want)
	}
}

func TestNormalizeSemanticSnapshotDedupesServiceTokenVariantsAndFindingSignatures(t *testing.T) {
	t.Parallel()

	resolver := newSemanticRepoAliasResolver(
		map[string]string{"bank-of-anthos": "/tmp/repos/bank-of-anthos"},
		nil,
	)
	snapshot := normalizeSemanticSnapshot(contracts.SemanticSnapshot{
		Coverage: contracts.Coverage{Missing: []string{"owner mappings"}},
		Entities: []contracts.Entity{
			{
				ID:   "svc.bank-of-anthos.user-service",
				Type: "service",
				Name: "user-service",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.8,
					Evidence:   []contracts.Evidence{{Repo: "bank-of-anthos", Path: "src/user-service/main.go"}},
				},
			},
			{
				ID:   "svc.bank-of-anthos.userservice",
				Type: "service",
				Name: "userservice",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.7,
					Evidence:   []contracts.Evidence{{Repo: "bank-of-anthos", Path: "src/user-service/main.go"}},
				},
			},
		},
		Findings: []contracts.Finding{
			{
				ID:         "finding.owner.gap.1",
				Severity:   "medium",
				Title:      "Missing owner mapping",
				RuleID:     "owner-gap",
				RelatedIDs: []string{"svc.bank-of-anthos.user-service"},
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.6,
					Evidence:   []contracts.Evidence{{Repo: "bank-of-anthos", Path: "src/user-service/main.go"}},
				},
			},
			{
				ID:          "finding.owner.gap.2",
				Severity:    "medium",
				Title:       "Missing owner mapping",
				RuleID:      "owner-gap",
				Description: "owner_team_id is unknown",
				RelatedIDs:  []string{"svc.bank-of-anthos.userservice"},
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.7,
					Evidence:   []contracts.Evidence{{Repo: "bank-of-anthos", Path: "src/user-service/main.go"}},
				},
			},
		},
		Questions: []contracts.Question{
			{ID: "q.owner", Text: "Who owns user-service?", RelatedIDs: []string{"svc.bank-of-anthos.userservice"}},
		},
	}, resolver)

	if got, want := len(snapshot.Entities), 1; got != want {
		t.Fatalf("expected service aliases to dedupe into one entity, got=%d want=%d", got, want)
	}
	winnerID := snapshot.Entities[0].ID
	if winnerID != "svc.bank-of-anthos.user-service" {
		t.Fatalf("expected stable canonical id winner, got %q", winnerID)
	}
	if got, want := len(snapshot.Findings), 1; got != want {
		t.Fatalf("expected duplicate owner-gap findings to dedupe, got=%d want=%d", got, want)
	}
	if got := snapshot.Findings[0].RelatedIDs; len(got) != 1 || got[0] != winnerID {
		t.Fatalf("expected finding related_ids to use deduped entity %q, got %v", winnerID, got)
	}
	if got := strings.TrimSpace(snapshot.Findings[0].Description); got != "owner_team_id is unknown" {
		t.Fatalf("expected merged finding description, got %q", got)
	}
	if got := snapshot.Questions[0].RelatedIDs; len(got) != 1 || got[0] != winnerID {
		t.Fatalf("expected question related_ids to be rewritten to %q, got %v", winnerID, got)
	}
}
