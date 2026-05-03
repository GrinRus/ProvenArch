package providercommon

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

type artifactSnapshot struct {
	ArtifactObserved bool
	Valid            bool
	State            string
	AuthoredFiles    int
	LastMutation     time.Time
}

func (s artifactSnapshot) diagnosticFields() map[string]any {
	fields := map[string]any{
		"manifest_state":      strings.TrimSpace(s.State),
		"artifact_observed":   s.ArtifactObserved,
		"artifact_valid":      s.Valid,
		"authored_file_count": s.AuthoredFiles,
	}
	if !s.LastMutation.IsZero() {
		fields["last_write_root_mutation_at"] = s.LastMutation.UTC().Format(time.RFC3339)
	}
	return fields
}

func (s artifactSnapshot) stallDiagnostic() StallDiagnostic {
	return StallDiagnostic{
		StallPhase:            StallPhasePostArtifact,
		ArtifactState:         s.State,
		AuthoredFileCount:     s.AuthoredFiles,
		LastWriteRootMutation: s.LastMutation,
	}
}

func monitorArtifactStall(ctx context.Context, task acpruntime.Task, tracker *commandActivityTracker, policy ActivityPolicy) (StallError, bool) {
	for {
		select {
		case <-ctx.Done():
			return StallError{}, false
		case <-time.After(policy.PollInterval):
		}
		snapshot := runtimeArtifactSnapshot(task)
		lastPipe := tracker.LastRead()
		lastMutation := snapshot.LastMutation
		if snapshot.ArtifactObserved {
			stallWindow := policy.PostArtifactStallWindow
			if !snapshot.Valid {
				stallWindow = policy.PartialArtifactStallWindow
			}
			if !lastPipe.IsZero() && time.Since(lastPipe) < stallWindow {
				continue
			}
			if !lastMutation.IsZero() && time.Since(lastMutation) < stallWindow {
				continue
			}
			return StallError{
				Sentinel: ErrStalledAfterArtifacts,
				Diagnostic: StallDiagnostic{
					StallPhase:            StallPhasePostArtifact,
					ArtifactState:         snapshot.State,
					AuthoredFileCount:     snapshot.AuthoredFiles,
					LastPipeActivity:      lastPipe,
					LastWriteRootMutation: lastMutation,
				},
			}, true
		}
		if !policy.MonitorPreArtifact {
			continue
		}
		if !lastMutation.IsZero() && time.Since(lastMutation) < policy.PreArtifactStallWindow {
			continue
		}
		if !lastPipe.IsZero() && time.Since(lastPipe) < policy.PreArtifactStallWindow {
			continue
		}
		return StallError{
			Sentinel: ErrStalledBeforeArtifacts,
			Diagnostic: StallDiagnostic{
				StallPhase:            StallPhasePreArtifact,
				ArtifactState:         snapshot.State,
				AuthoredFileCount:     snapshot.AuthoredFiles,
				LastPipeActivity:      lastPipe,
				LastWriteRootMutation: lastMutation,
			},
		}, true
	}
}

func runtimeArtifactSnapshot(task acpruntime.Task) artifactSnapshot {
	switch acpruntime.StepProviderKeyForStepID(task.StepID) {
	case acpruntime.StepProviderStep1Collect:
		return collectArtifactSnapshot(task.WriteRoot)
	case acpruntime.StepProviderStep3Findings:
		return validatorArtifactSnapshot(task.WriteRoot)
	default:
		if runtimedrafts.IsDraftStep(task.StepID) {
			return draftArtifactSnapshot(task)
		}
		return artifactSnapshot{}
	}
}

func collectArtifactSnapshot(writeRoot string) artifactSnapshot {
	snapshot := artifactSnapshot{}
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return snapshot
	}
	manifestPath := filepath.Join(filepath.Clean(writeRoot), ShardPackManifestFileName)
	if info, err := os.Stat(manifestPath); err == nil && !info.IsDir() {
		snapshot.ArtifactObserved = true
		snapshot.LastMutation = info.ModTime().UTC()
		snapshot.State = "present"
		if raw, readErr := os.ReadFile(manifestPath); readErr == nil {
			if _, parseErr := contracts.ParseShardPackManifest(raw); parseErr == nil {
				snapshot.Valid = true
				snapshot.State = "valid"
			} else {
				snapshot.State = "invalid"
			}
		}
	}
	cleanRoot := filepath.Clean(writeRoot)
	if _, err := os.Stat(cleanRoot); err != nil {
		return snapshot
	}
	_ = filepath.WalkDir(cleanRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil {
			return nil
		}
		if path == cleanRoot || entry.IsDir() || isCollectNonAuthoredFile(entry.Name()) {
			return nil
		}
		rel, relErr := filepath.Rel(cleanRoot, path)
		if relErr != nil || strings.TrimSpace(rel) == "" || rel == "." {
			return nil
		}
		snapshot.AuthoredFiles++
		snapshot.ArtifactObserved = true
		if info, statErr := os.Stat(path); statErr == nil {
			modTime := info.ModTime().UTC()
			if modTime.After(snapshot.LastMutation) {
				snapshot.LastMutation = modTime
			}
		}
		return nil
	})
	if snapshot.State == "" && snapshot.ArtifactObserved {
		snapshot.State = "partial"
	}
	return snapshot
}

func isCollectNonAuthoredFile(name string) bool {
	switch strings.TrimSpace(name) {
	case "", ShardPackManifestFileName, "runtime-execution.json":
		return true
	default:
		return false
	}
}

type writeRootFileSnapshot map[string]writeRootFileState

type writeRootFileState struct {
	IsDir   bool
	Mode    fs.FileMode
	Size    int64
	ModTime time.Time
	SHA256  [sha256.Size]byte
}

func snapshotWriteRootFiles(root string) (writeRootFileSnapshot, error) {
	snapshot := writeRootFileSnapshot{}
	root = strings.TrimSpace(root)
	if root == "" {
		return snapshot, nil
	}
	cleanRoot := filepath.Clean(root)
	if _, err := os.Stat(cleanRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return snapshot, nil
		}
		return nil, fmt.Errorf("stat write_root %q: %w", root, err)
	}
	if err := filepath.WalkDir(cleanRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry == nil {
			return nil
		}
		rel, err := filepath.Rel(cleanRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		state := writeRootFileState{
			IsDir: info.IsDir(),
			Mode:  info.Mode(),
		}
		if info.IsDir() {
			snapshot[filepath.ToSlash(rel)] = state
			return nil
		}
		state.Size = info.Size()
		state.ModTime = info.ModTime().UTC()
		if info.Mode().IsRegular() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			state.SHA256 = sha256.Sum256(content)
		}
		snapshot[filepath.ToSlash(rel)] = state
		return nil
	}); err != nil {
		return nil, fmt.Errorf("snapshot write_root %q: %w", root, err)
	}
	return snapshot, nil
}

func validateCollectManifestRepairWriteSet(task acpruntime.Task, before writeRootFileSnapshot) error {
	after, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return err
	}
	changes := unexpectedCollectRepairMutations(before, after)
	if len(changes) == 0 {
		return nil
	}
	return fmt.Errorf("manifest-only collect repair wrote forbidden files: %s", strings.Join(changes, "; "))
}

func validateCollectArtifactPairRepairWriteSet(task acpruntime.Task, before writeRootFileSnapshot) error {
	after, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return err
	}
	allowedDocPath := filepath.ToSlash(steppolicy.SuggestedCollectDocumentPath(task))
	changes := unexpectedRepairMutations(before, after, func(path string, _ writeRootFileState) bool {
		return path == ShardPackManifestFileName || path == allowedDocPath
	})
	if len(changes) == 0 {
		return nil
	}
	return fmt.Errorf("collect pair recovery wrote forbidden files: %s", strings.Join(changes, "; "))
}

func unexpectedCollectRepairMutations(before writeRootFileSnapshot, after writeRootFileSnapshot) []string {
	return unexpectedRepairMutations(before, after, func(path string, _ writeRootFileState) bool {
		return path == ShardPackManifestFileName
	})
}

func validateValidatorVerdictRepairWriteSet(task acpruntime.Task, before writeRootFileSnapshot) error {
	after, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return err
	}
	changes := unexpectedRepairMutations(before, after, func(path string, _ writeRootFileState) bool {
		return path == ValidatorVerdictFileName
	})
	if len(changes) == 0 {
		return nil
	}
	return fmt.Errorf("verdict-only validator repair wrote forbidden files: %s", strings.Join(changes, "; "))
}

func validateDraftArtifactRepairWriteSet(task acpruntime.Task, beforeWriteRoot writeRootFileSnapshot, beforeDraftRoot writeRootFileSnapshot) error {
	afterWriteRoot, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return err
	}
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	writeRootChanges := unexpectedRepairMutations(beforeWriteRoot, afterWriteRoot, func(path string, _ writeRootFileState) bool {
		return strings.TrimSpace(manifestFile) != "" && path == manifestFile
	})
	if len(writeRootChanges) > 0 {
		return fmt.Errorf("draft repair wrote forbidden write_root files: %s", strings.Join(writeRootChanges, "; "))
	}
	if strings.TrimSpace(task.DraftFinalRoot) == "" {
		return nil
	}
	afterDraftRoot, err := snapshotWriteRootFiles(task.DraftFinalRoot)
	if err != nil {
		return err
	}
	draftRootChanges := unexpectedRepairMutations(beforeDraftRoot, afterDraftRoot, func(_ string, _ writeRootFileState) bool {
		return true
	})
	if len(draftRootChanges) > 0 {
		return fmt.Errorf("draft repair wrote forbidden draft_final_root files: %s", strings.Join(draftRootChanges, "; "))
	}
	return nil
}

func unexpectedRepairMutations(before writeRootFileSnapshot, after writeRootFileSnapshot, allowed func(string, writeRootFileState) bool) []string {
	paths := map[string]struct{}{}
	for path := range before {
		paths[path] = struct{}{}
	}
	for path := range after {
		paths[path] = struct{}{}
	}
	changes := make([]string, 0)
	for path := range paths {
		beforeState, beforeExists := before[path]
		afterState, afterExists := after[path]
		switch {
		case !beforeExists && afterExists:
			if allowed != nil && allowed(path, afterState) {
				continue
			}
			changes = append(changes, "created "+describeWriteRootPath(path, afterState))
		case beforeExists && !afterExists:
			if allowed != nil && allowed(path, beforeState) {
				continue
			}
			changes = append(changes, "deleted "+describeWriteRootPath(path, beforeState))
		case beforeExists && afterExists && beforeState != afterState:
			if allowed != nil && allowed(path, afterState) {
				continue
			}
			changes = append(changes, "modified "+describeWriteRootPath(path, afterState))
		}
	}
	sort.Strings(changes)
	return changes
}

func describeWriteRootPath(path string, state writeRootFileState) string {
	if state.IsDir {
		return "directory " + path
	}
	return path
}

func draftArtifactSnapshot(task acpruntime.Task) artifactSnapshot {
	snapshot := artifactSnapshot{}
	if strings.TrimSpace(task.WriteRoot) == "" {
		return snapshot
	}
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	if manifestFile != "" {
		manifestPath := filepath.Join(filepath.Clean(task.WriteRoot), manifestFile)
		if info, err := os.Stat(manifestPath); err == nil && !info.IsDir() {
			snapshot.ArtifactObserved = true
			snapshot.LastMutation = info.ModTime().UTC()
			snapshot.State = "present"
			if _, _, validateErr := ValidateRequiredRuntimeDraftArtifacts(task); validateErr == nil {
				snapshot.Valid = true
				snapshot.State = "valid"
			} else {
				snapshot.State = "invalid"
			}
		}
	}
	snapshot.AuthoredFiles += countFiles(task.WriteRoot, manifestFile)
	snapshot.AuthoredFiles += countFilesRecursive(task.DraftFinalRoot, "")
	snapshot.LastMutation = latestMutation(snapshot.LastMutation, latestFileMutation(task.WriteRoot))
	snapshot.LastMutation = latestMutation(snapshot.LastMutation, latestFileMutationRecursive(task.DraftFinalRoot))
	if snapshot.AuthoredFiles > 0 {
		snapshot.ArtifactObserved = true
		if snapshot.State == "" {
			snapshot.State = "partial"
		}
	}
	return snapshot
}

func validatorArtifactSnapshot(writeRoot string) artifactSnapshot {
	snapshot := artifactSnapshot{}
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return snapshot
	}
	verdictPath := filepath.Join(filepath.Clean(writeRoot), ValidatorVerdictFileName)
	if info, err := os.Stat(verdictPath); err == nil && !info.IsDir() {
		snapshot.ArtifactObserved = true
		snapshot.LastMutation = info.ModTime().UTC()
		snapshot.State = "present"
		if raw, readErr := os.ReadFile(verdictPath); readErr == nil {
			if _, parseErr := contracts.ParseValidatorVerdict(raw); parseErr == nil {
				snapshot.Valid = true
				snapshot.State = "valid"
			} else {
				snapshot.State = "invalid"
			}
		}
	}
	return snapshot
}

func countFiles(root string, except string) int {
	root = strings.TrimSpace(root)
	if root == "" {
		return 0
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || (except != "" && entry.Name() == except) {
			continue
		}
		count++
	}
	return count
}

func countFilesRecursive(root string, except string) int {
	root = strings.TrimSpace(root)
	if root == "" {
		return 0
	}
	cleanRoot := filepath.Clean(root)
	count := 0
	_ = filepath.WalkDir(cleanRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(cleanRoot, path)
		if relErr != nil {
			return nil
		}
		if except != "" && filepath.ToSlash(rel) == filepath.ToSlash(except) {
			return nil
		}
		count++
		return nil
	})
	return count
}

func latestFileMutation(root string) time.Time {
	root = strings.TrimSpace(root)
	if root == "" {
		return time.Time{}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return time.Time{}
	}
	latest := time.Time{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}
		latest = latestMutation(latest, info.ModTime().UTC())
	}
	return latest
}

func latestFileMutationRecursive(root string) time.Time {
	root = strings.TrimSpace(root)
	if root == "" {
		return time.Time{}
	}
	var latest time.Time
	_ = filepath.WalkDir(filepath.Clean(root), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() {
			return nil
		}
		info, statErr := entry.Info()
		if statErr != nil {
			return nil
		}
		latest = latestMutation(latest, info.ModTime().UTC())
		_ = path
		return nil
	})
	return latest
}

func latestMutation(current time.Time, candidate time.Time) time.Time {
	if candidate.After(current) {
		return candidate
	}
	return current
}
