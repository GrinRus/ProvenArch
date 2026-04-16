package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

func TestDocFirstStageThenPromoteFlow(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(docflowCustomProposalRunner{}),
		WithExecutionOverrides(acpruntime.ExecutionOverrides{
			FailurePolicy: strPtr(acpruntime.ExecutionFailurePolicyBestEffort),
		}),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
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

	finalIndex := readRunFinalRunIndex(t, ws.Path, info.RunID)
	customDocs := collectCustomProposalDocs(finalIndex)
	if len(customDocs) == 0 {
		t.Fatalf("expected custom docs-first proposal docs in final run index")
	}

	for _, document := range customDocs {
		stagedContent, stagedErr := os.ReadFile(filepath.Join(ws.Path, filepath.FromSlash(document.StagedPath)))
		if stagedErr != nil {
			t.Fatalf("read staged proposal %q: %v", document.StagedPath, stagedErr)
		}
		canonicalContent, canonicalErr := os.ReadFile(filepath.Join(ws.Path, filepath.FromSlash(document.CanonicalPath)))
		if canonicalErr != nil {
			t.Fatalf("read canonical proposal %q: %v", document.CanonicalPath, canonicalErr)
		}
		if string(stagedContent) != string(canonicalContent) {
			t.Fatalf("canonical proposal %q does not match staged content", document.CanonicalPath)
		}
	}

	manifestPaths, globErr := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", info.RunID, "staging", "shards", "*", "shard-pack-manifest.json"))
	if globErr != nil {
		t.Fatalf("glob shard manifests: %v", globErr)
	}
	if len(manifestPaths) == 0 {
		t.Fatalf("expected shard manifests in staged shard roots")
	}

	verdict := readRunValidatorVerdict(t, ws.Path, info.RunID)
	if verdict.Verdict != "PASS" {
		t.Fatalf("expected PASS verdict, got %q", verdict.Verdict)
	}

	if len(finalIndex.Compatibility.Entities) == 0 {
		t.Fatalf("expected compatibility entities in final run index")
	}
	entityFiles, entityGlobErr := filepath.Glob(filepath.Join(ws.Path, "model", "entities", "*.yaml"))
	if entityGlobErr != nil {
		t.Fatalf("glob model entities: %v", entityGlobErr)
	}
	if len(entityFiles) == 0 {
		t.Fatalf("expected derived model/entities files after promotion")
	}
}

func TestDocFirstValidatorFailBlocksPromotionInBestEffort(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(docflowFailingValidatorRunner{}),
		WithExecutionOverrides(acpruntime.ExecutionOverrides{
			FailurePolicy: strPtr(acpruntime.ExecutionFailurePolicyBestEffort),
		}),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected run error when validator verdict is FAIL")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if info.ErrorCode != runErrorCodePartialFailed {
		t.Fatalf("expected error_code %q, got %q", runErrorCodePartialFailed, info.ErrorCode)
	}

	finalIndex := readRunFinalRunIndex(t, ws.Path, info.RunID)
	customDocs := collectCustomProposalDocs(finalIndex)
	if len(customDocs) == 0 {
		t.Fatalf("expected custom docs-first proposal docs in final run index")
	}

	for _, document := range customDocs {
		if _, stagedErr := os.Stat(filepath.Join(ws.Path, filepath.FromSlash(document.StagedPath))); stagedErr != nil {
			t.Fatalf("expected staged proposal %q: %v", document.StagedPath, stagedErr)
		}
		_, canonicalErr := os.Stat(filepath.Join(ws.Path, filepath.FromSlash(document.CanonicalPath)))
		if canonicalErr == nil {
			t.Fatalf("expected promotion to be blocked for %q", document.CanonicalPath)
		}
		if !errors.Is(canonicalErr, os.ErrNotExist) {
			t.Fatalf("unexpected canonical proposal stat error for %q: %v", document.CanonicalPath, canonicalErr)
		}
	}

	verdict := readRunValidatorVerdict(t, ws.Path, info.RunID)
	if verdict.Verdict != "FAIL" {
		t.Fatalf("expected FAIL verdict, got %q", verdict.Verdict)
	}
}

type docflowCustomProposalRunner struct{}

func (docflowCustomProposalRunner) Preflight(context.Context) error { return nil }

func (docflowCustomProposalRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	result, err := claudecode.FakeRunner{}.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if strings.HasSuffix(task.StepID, "step1.collect") {
		if err := appendCustomProposalToManifest(task); err != nil {
			return acpruntime.Result{}, err
		}
	}
	return result, nil
}

type docflowFailingValidatorRunner struct{}

func (docflowFailingValidatorRunner) Preflight(context.Context) error { return nil }

func (docflowFailingValidatorRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	result, err := docflowCustomProposalRunner{}.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if strings.HasSuffix(task.StepID, "step3.findings") {
		if err := overwriteValidatorVerdict(task, "FAIL"); err != nil {
			return acpruntime.Result{}, err
		}
	}
	return result, nil
}

func appendCustomProposalToManifest(task acpruntime.Task) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return fmt.Errorf("write_root is required")
	}
	manifestPath := filepath.Join(writeRoot, "shard-pack-manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read shard manifest: %w", err)
	}
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		return fmt.Errorf("parse shard manifest: %w", err)
	}

	shardSlug := slugutil.Slugify(strings.TrimSpace(task.ShardID))
	if shardSlug == "" {
		shardSlug = "workspace"
	}
	docID := "doc.runtime-proposal." + shardSlug
	citationID := "cite.runtime-proposal." + shardSlug
	claimID := "claim.runtime-proposal." + shardSlug
	canonicalPath := path.Join("proposals", "runtime-docs-first", "proposal-"+shardSlug+".md")

	for _, existing := range manifest.Documents {
		if strings.TrimSpace(existing.ID) == docID {
			return nil
		}
	}

	content := fmt.Sprintf("# Runtime Proposal (%s)\n\nThis proposal is authored by shard `%s`.\n", shardSlug, shardSlug)
	if err := os.WriteFile(filepath.Join(writeRoot, "custom-proposal.md"), []byte(content), 0o644); err != nil {
		return fmt.Errorf("write custom proposal: %w", err)
	}

	manifest.Documents = append(manifest.Documents, contracts.AuthoredDocument{
		ID:            docID,
		Kind:          "proposal",
		Title:         "Runtime Proposal " + shardSlug,
		Path:          "custom-proposal.md",
		CanonicalPath: canonicalPath,
		Topics:        []string{"runtime-docs-first"},
		CitationIDs:   []string{citationID},
		Status:        "staged",
	})
	manifest.Citations = append(manifest.Citations, contracts.DocumentCitation{
		ID:          citationID,
		Repo:        primaryRepoForTask(task),
		Path:        "README.md",
		ClaimIDs:    []string{claimID},
		DocumentIDs: []string{docID},
	})

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal shard manifest: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(manifestPath, encoded, 0o644); err != nil {
		return fmt.Errorf("rewrite shard manifest: %w", err)
	}
	return nil
}

func overwriteValidatorVerdict(task acpruntime.Task, verdictValue string) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return fmt.Errorf("write_root is required")
	}
	checkedPaths := []string{
		path.Join("reports", "taskruns", task.RunID, "staging", "final", "final-run-index.json"),
		path.Join("reports", "taskruns", task.RunID, "staging", "final", "citation-index.json"),
	}
	verdict := contracts.ValidatorVerdict{
		Version:      1,
		RunID:        strings.TrimSpace(task.RunID),
		GeneratedAt:  task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
		Verdict:      verdictValue,
		Summary:      "Synthetic validator verdict for docs-first promotion test.",
		CheckedPaths: checkedPaths,
	}
	if verdictValue == "FAIL" {
		verdict.Issues = []contracts.ValidatorIssue{
			{
				Code:     "synthetic_validator_fail",
				Severity: "error",
				Message:  "validator forced fail for best-effort path test",
				Path:     checkedPaths[0],
			},
		}
	}
	encoded, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal validator verdict: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(writeRoot, "validator-verdict.json"), encoded, 0o644); err != nil {
		return fmt.Errorf("write validator verdict: %w", err)
	}
	return nil
}

func primaryRepoForTask(task acpruntime.Task) string {
	if value := strings.TrimSpace(task.RepoScope); value != "" {
		return value
	}
	for _, scope := range task.RepoScopes {
		if value := strings.TrimSpace(scope); value != "" {
			return value
		}
	}
	return "workspace"
}

func readRunFinalRunIndex(t *testing.T, workspacePath string, runID string) contracts.FinalRunIndex {
	t.Helper()

	finalIndexPath := filepath.Join(workspacePath, "reports", "taskruns", runID, "staging", "final", "final-run-index.json")
	raw, err := os.ReadFile(finalIndexPath)
	if err != nil {
		t.Fatalf("read final run index %q: %v", finalIndexPath, err)
	}
	index, err := contracts.ParseFinalRunIndex(raw)
	if err != nil {
		t.Fatalf("parse final run index %q: %v", finalIndexPath, err)
	}
	return index
}

func readRunValidatorVerdict(t *testing.T, workspacePath string, runID string) contracts.ValidatorVerdict {
	t.Helper()

	verdictPath := filepath.Join(workspacePath, "reports", "taskruns", runID, "validator", "validator-verdict.json")
	raw, err := os.ReadFile(verdictPath)
	if err != nil {
		t.Fatalf("read validator verdict %q: %v", verdictPath, err)
	}
	verdict, err := contracts.ParseValidatorVerdict(raw)
	if err != nil {
		t.Fatalf("parse validator verdict %q: %v", verdictPath, err)
	}
	return verdict
}

func collectCustomProposalDocs(index contracts.FinalRunIndex) []contracts.FinalRunDocument {
	documents := make([]contracts.FinalRunDocument, 0, len(index.CanonicalDocuments))
	for _, document := range index.CanonicalDocuments {
		if strings.HasPrefix(strings.TrimSpace(document.CanonicalPath), "proposals/runtime-docs-first/") {
			documents = append(documents, document)
		}
	}
	return documents
}

func strPtr(value string) *string {
	return &value
}
