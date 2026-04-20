package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestRuntimeArtifactContextFindingsExcludesRepoRootsFromDefaultReadContext(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	repoRoot := filepath.Join(workspaceRoot, "..", "repo-a")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}

	execution := &pipelineExecution{
		runID:             "run-ctx",
		workspace:         workspace.Root{Path: workspaceRoot},
		resolvedRepoPaths: map[string]string{"repo-a": repoRoot},
	}

	_, writeRoot, readRoots, err := execution.runtimeArtifactContext("refresh.step3.findings", "validator", []string{"repo-a"})
	if err != nil {
		t.Fatalf("runtime artifact context: %v", err)
	}
	finalRoot, resolveErr := execution.workspace.Resolve(runtimeFinalArtifactRoot(execution.runID))
	if resolveErr != nil {
		t.Fatalf("resolve final root: %v", resolveErr)
	}
	if !containsPathValue(readRoots, workspaceRoot) {
		t.Fatalf("expected workspace root in read context, got %v", readRoots)
	}
	if !containsPathValue(readRoots, writeRoot) {
		t.Fatalf("expected validator write root in read context, got %v", readRoots)
	}
	if !containsPathValue(readRoots, finalRoot) {
		t.Fatalf("expected staged final root in read context, got %v", readRoots)
	}
	if containsPathValue(readRoots, repoRoot) {
		t.Fatalf("did not expect repo root in findings read context, got %v", readRoots)
	}
}

func TestRuntimeArtifactContextCollectKeepsRepoRootsInReadContext(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	repoRoot := filepath.Join(workspaceRoot, "..", "repo-b")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}

	execution := &pipelineExecution{
		runID:             "run-ctx",
		workspace:         workspace.Root{Path: workspaceRoot},
		resolvedRepoPaths: map[string]string{"repo-b": repoRoot},
	}

	_, writeRoot, readRoots, err := execution.runtimeArtifactContext("refresh.step1.collect", "repo-b", []string{"repo-b"})
	if err != nil {
		t.Fatalf("runtime artifact context: %v", err)
	}
	if !containsPathValue(readRoots, workspaceRoot) {
		t.Fatalf("expected workspace root in collect read context, got %v", readRoots)
	}
	if !containsPathValue(readRoots, repoRoot) {
		t.Fatalf("expected repo root in collect read context, got %v", readRoots)
	}
	if containsPathValue(readRoots, writeRoot) {
		t.Fatalf("did not expect collect write_root in read context by default, got %v", readRoots)
	}
}

func containsPathValue(values []string, target string) bool {
	target = filepath.Clean(strings.TrimSpace(target))
	for _, value := range values {
		if filepath.Clean(strings.TrimSpace(value)) == target {
			return true
		}
	}
	return false
}

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
		finalRunIndex:    &contracts.FinalRunIndex{},
		validatorVerdict: &contracts.ValidatorVerdict{Verdict: "FAIL"},
	}
	err := execution.promoteValidatedArtifacts()
	if err == nil {
		t.Fatalf("expected promotion failure when validator verdict is FAIL")
	}
	if !strings.Contains(err.Error(), "validator verdict is FAIL") {
		t.Fatalf("expected validator verdict failure, got %v", err)
	}
}

func TestValidateStagedArtifactsReportsMissingStagedDocument(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	execution := &pipelineExecution{
		workspace: workspace.Root{Path: workspaceRoot},
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

func TestValidateStagedArtifactsReportsMissingRequiredCanonicalLiveDocuments(t *testing.T) {
	t.Parallel()

	execution := &pipelineExecution{
		finalRunIndex: &contracts.FinalRunIndex{
			RunID: "run-1",
			CanonicalDocuments: []contracts.FinalRunDocument{
				{
					ID:            "doc.agent.domain",
					Kind:          "agent-output",
					CanonicalPath: "reports/agent-outputs/domains/payments.md",
					StagedPath:    "reports/taskruns/run-1/staging/final/reports/agent-outputs/domains/payments.md",
				},
			},
		},
		citationIndex: &contracts.CitationIndex{RunID: "run-1"},
	}

	issues := execution.validateStagedArtifacts()
	missingRequired := 0
	for _, issue := range issues {
		if issue.Code == "missing_required_canonical_document" {
			missingRequired++
		}
	}
	if missingRequired != len(requiredCanonicalLiveDocuments) {
		t.Fatalf("expected %d missing required canonical document issues, got %d (%#v)", len(requiredCanonicalLiveDocuments), missingRequired, issues)
	}
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
