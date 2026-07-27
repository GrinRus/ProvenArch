package orchestrator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/refreshplan"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const (
	shardBaselineIntegrityFile    = "baseline-integrity.json"
	shardBaselineIntegrityVersion = 1
)

type shardBaselineIntegrity struct {
	Version      int                       `json:"version"`
	RunID        string                    `json:"run_id"`
	SourceRunID  string                    `json:"source_run_id"`
	StepContract string                    `json:"step_contract"`
	ShardID      string                    `json:"shard_id"`
	DomainID     string                    `json:"domain_id"`
	RepoScopes   []string                  `json:"repo_scopes"`
	PathScopes   []string                  `json:"path_scopes"`
	SourceRanges []refreshplan.SourceRange `json:"source_ranges"`
	Files        []shardBaselineFile       `json:"files"`
}

type shardBaselineFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int    `json:"size"`
}

func (e *pipelineExecution) writeShardBaselineIntegrity(
	stepID string,
	domainID string,
	plan runtimeShardPlan,
	inherited *shardBaselineIntegrity,
) error {
	root := runtimeShardArtifactRoot(e.runID, plan.ShardID)
	files, err := e.workspace.ReadRegularTree(root)
	if err != nil {
		return err
	}
	files = withoutTreeFile(files, shardBaselineIntegrityFile)
	if err := validateShardIdentity(e.runID, stepID, domainID, plan, root, files); err != nil {
		return err
	}

	sourceRunID := e.runID
	sourceRanges, err := e.sourceRangesForRun(e.runID)
	if err != nil {
		return err
	}
	if inherited != nil {
		sourceRunID = inherited.SourceRunID
	}
	integrity := shardBaselineIntegrity{
		Version:      shardBaselineIntegrityVersion,
		RunID:        e.runID,
		SourceRunID:  sourceRunID,
		StepContract: runtimeStepContract(stepID),
		ShardID:      plan.ShardID,
		DomainID:     domainID,
		RepoScopes:   normalizedIdentityValues(plan.RepoScopes),
		PathScopes:   normalizedIdentityValues(plan.PathScopes),
		SourceRanges: sourceRanges,
		Files:        digestTreeFiles(files),
	}
	raw, err := json.MarshalIndent(integrity, "", "  ")
	if err != nil {
		return err
	}
	return e.workspace.WriteFileAtomic(path.Join(root, shardBaselineIntegrityFile), append(raw, '\n'))
}

func (e *pipelineExecution) validateBaselineShard(
	baselineRunID string,
	stepID string,
	domainID string,
	plan runtimeShardPlan,
) (shardBaselineIntegrity, []workspace.TreeFile, error) {
	root := runtimeShardArtifactRoot(baselineRunID, plan.ShardID)
	files, err := e.workspace.ReadRegularTree(root)
	if err != nil {
		return shardBaselineIntegrity{}, nil, err
	}
	sidecarRaw, ok := treeFileContent(files, shardBaselineIntegrityFile)
	if !ok {
		return shardBaselineIntegrity{}, nil, fmt.Errorf("baseline integrity sidecar is unavailable")
	}
	var integrity shardBaselineIntegrity
	if err := json.Unmarshal(sidecarRaw, &integrity); err != nil {
		return shardBaselineIntegrity{}, nil, fmt.Errorf("decode baseline integrity: %w", err)
	}
	if integrity.Version != shardBaselineIntegrityVersion ||
		integrity.RunID != baselineRunID ||
		integrity.StepContract != runtimeStepContract(stepID) ||
		integrity.ShardID != plan.ShardID ||
		integrity.DomainID != domainID ||
		!equalIdentityValues(integrity.RepoScopes, plan.RepoScopes) ||
		!equalIdentityValues(integrity.PathScopes, plan.PathScopes) {
		return shardBaselineIntegrity{}, nil, fmt.Errorf("baseline shard identity does not match current plan")
	}
	if err := e.validateBaselineSourceRanges(plan, integrity.SourceRanges); err != nil {
		return shardBaselineIntegrity{}, nil, err
	}
	contentFiles := withoutTreeFile(files, shardBaselineIntegrityFile)
	if err := validateShardIdentity(baselineRunID, stepID, domainID, plan, root, contentFiles); err != nil {
		return shardBaselineIntegrity{}, nil, err
	}
	if !equalDigests(integrity.Files, digestTreeFiles(contentFiles)) {
		return shardBaselineIntegrity{}, nil, fmt.Errorf("baseline shard digest inventory does not match stored files")
	}
	return integrity, contentFiles, nil
}

func (e *pipelineExecution) validateBaselineSourceRanges(plan runtimeShardPlan, stored []refreshplan.SourceRange) error {
	if e.refreshExecution == nil {
		return fmt.Errorf("refresh execution is unavailable")
	}
	storedByRepo := map[string]refreshplan.SourceRange{}
	for _, item := range stored {
		storedByRepo[item.Repo] = item
	}
	currentByRepo := map[string]refreshplan.SourceRange{}
	for _, item := range e.refreshExecution.SourceRanges {
		currentByRepo[item.Repo] = item
	}
	for _, repo := range normalizedIdentityValues(plan.RepoScopes) {
		baseline, ok := storedByRepo[repo]
		if !ok || baseline.CurrentRevision == nil {
			return fmt.Errorf("baseline source revision for repo %q is unavailable", repo)
		}
		current, ok := currentByRepo[repo]
		if !ok || current.BaselineRevision == nil || *current.BaselineRevision != *baseline.CurrentRevision {
			return fmt.Errorf("baseline source revision for repo %q does not match refresh range", repo)
		}
	}
	return nil
}

func validateShardIdentity(
	runID string,
	stepID string,
	domainID string,
	plan runtimeShardPlan,
	root string,
	files []workspace.TreeFile,
) error {
	manifestRaw, ok := treeFileContent(files, shardPackManifestFile)
	if !ok {
		return fmt.Errorf("shard manifest is unavailable")
	}
	manifest, err := contracts.ParseShardPackManifest(manifestRaw)
	if err != nil {
		return fmt.Errorf("parse shard manifest: %w", err)
	}
	if manifest.RunID != runID ||
		manifest.ShardID != plan.ShardID ||
		(strings.TrimSpace(manifest.DomainID) != "" && manifest.DomainID != domainID) ||
		runtimeStepContract(manifest.StepID) != runtimeStepContract(stepID) ||
		manifest.ArtifactRoot != root {
		return fmt.Errorf("shard manifest identity does not match plan")
	}

	executionRaw, ok := treeFileContent(files, runtimeExecutionFile)
	if !ok {
		return fmt.Errorf("runtime execution is unavailable")
	}
	execution, err := contracts.ParseRuntimeExecution(executionRaw)
	if err != nil {
		return fmt.Errorf("parse runtime execution: %w", err)
	}
	if execution.RunID != runID ||
		execution.ShardID != plan.ShardID ||
		execution.DomainID != domainID ||
		execution.Status != "succeeded" ||
		runtimeStepContract(execution.StepID) != runtimeStepContract(stepID) ||
		execution.ArtifactRoot != root ||
		!equalIdentityValues(execution.RepoScopes, plan.RepoScopes) ||
		!equalIdentityValues(execution.PathScopes, plan.PathScopes) {
		return fmt.Errorf("runtime execution identity does not match plan")
	}
	return nil
}

func (e *pipelineExecution) sourceRangesForRun(runID string) ([]refreshplan.SourceRange, error) {
	raw, err := e.workspace.ReadFile(path.Join("reports", "taskruns", runID, "source-revisions.json"))
	if err != nil {
		return nil, fmt.Errorf("read source revisions: %w", err)
	}
	revisions, err := refreshplan.ParseSourceRevisions(raw)
	if err != nil || revisions.RunID != runID {
		if err == nil {
			err = fmt.Errorf("source revisions run_id mismatch")
		}
		return nil, err
	}
	ranges := make([]refreshplan.SourceRange, 0, len(revisions.Repos))
	for _, repo := range revisions.Repos {
		ranges = append(ranges, refreshplan.SourceRange{
			Repo:             repo.Name,
			BaselineRevision: cloneStringPointer(repo.BaselineRevision),
			CurrentRevision:  cloneStringPointer(repo.CurrentRevision),
		})
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].Repo < ranges[j].Repo })
	return ranges, nil
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func digestTreeFiles(files []workspace.TreeFile) []shardBaselineFile {
	result := make([]shardBaselineFile, 0, len(files))
	for _, file := range files {
		sum := sha256.Sum256(file.Content)
		result = append(result, shardBaselineFile{
			Path:   file.Path,
			SHA256: hex.EncodeToString(sum[:]),
			Size:   len(file.Content),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func treeFileContent(files []workspace.TreeFile, name string) ([]byte, bool) {
	for _, file := range files {
		if file.Path == name {
			return file.Content, true
		}
	}
	return nil, false
}

func withoutTreeFile(files []workspace.TreeFile, name string) []workspace.TreeFile {
	result := make([]workspace.TreeFile, 0, len(files))
	for _, file := range files {
		if file.Path != name {
			result = append(result, file)
		}
	}
	return result
}

func normalizedIdentityValues(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	sort.Strings(result)
	return result
}

func equalIdentityValues(left, right []string) bool {
	return strings.Join(normalizedIdentityValues(left), "\x00") == strings.Join(normalizedIdentityValues(right), "\x00")
}

func equalDigests(left, right []shardBaselineFile) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func rewritePreservedShardManifest(raw []byte, baselineRunID, runID, artifactRoot string) ([]byte, error) {
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		return nil, err
	}
	if manifest.RunID != baselineRunID {
		return nil, fmt.Errorf("baseline shard manifest run_id mismatch")
	}
	manifest.RunID = runID
	manifest.StepID = "refresh.step1.collect"
	manifest.ArtifactRoot = artifactRoot
	rewritten, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if _, err := contracts.ParseShardPackManifest(rewritten); err != nil {
		return nil, err
	}
	return append(rewritten, '\n'), nil
}

func rewritePreservedRuntimeExecution(raw []byte, baselineRunID, runID, artifactRoot string) ([]byte, error) {
	execution, err := contracts.ParseRuntimeExecution(raw)
	if err != nil {
		return nil, err
	}
	if execution.RunID != baselineRunID {
		return nil, fmt.Errorf("baseline runtime execution run_id mismatch")
	}
	execution.RunID = runID
	execution.StepID = "refresh.step1.collect"
	execution.ArtifactRoot = artifactRoot
	rewritten, err := json.MarshalIndent(execution, "", "  ")
	if err != nil {
		return nil, err
	}
	if _, err := contracts.ParseRuntimeExecution(rewritten); err != nil {
		return nil, err
	}
	return append(rewritten, '\n'), nil
}
