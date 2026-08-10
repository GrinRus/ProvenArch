package orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

const (
	runtimeWriteAuditUnexpectedMutation = "runtime_write_audit_unexpected_mutation"
	runtimeWriteAuditRepoSkipped        = "runtime_write_audit_repo_skipped"
	runtimeWriteAuditRestoredMutation   = "runtime_write_audit_restored_mutation"
	runtimeWriteAuditRestoreConflict    = "runtime_write_audit_restore_conflict"
	runtimeWriteAuditMaxPaths           = 20
)

type runtimeProtectedFileSnapshot struct {
	digest  string
	content []byte
	mode    os.FileMode
}

type runtimeWriteAuditSnapshot struct {
	protectedFiles map[string]runtimeProtectedFileSnapshot
	repoStatuses   map[string][]string
	skippedRepos   []runtimeWriteAuditSkippedRepo
}

type runtimeWriteAuditSkippedRepo struct {
	Root   string
	Reason string
}

func beginRuntimeWriteAudit(task acpruntime.Task) runtimeWriteAuditSnapshot {
	return runtimeWriteAuditSnapshot{
		protectedFiles: snapshotProtectedWorkspaceFiles(task.Workspace),
		repoStatuses:   snapshotAuditedRepoStatuses(task),
		skippedRepos:   skippedRuntimeWriteAuditRepos(task),
	}
}

func (e *pipelineExecution) completeRuntimeWriteAudit(stepID string, domainID string, provider acpruntime.Provider, task acpruntime.Task, before runtimeWriteAuditSnapshot) error {
	violations := []string{}
	afterProtected := snapshotProtectedWorkspaceFiles(task.Workspace)
	if changed := changedProtectedSnapshotPaths(before.protectedFiles, afterProtected); len(changed) > 0 {
		violations = append(violations, e.reportRuntimeWriteAuditWarning(stepID, domainID, task, "workspace", strings.TrimSpace(task.Workspace), changed))
		e.restoreRuntimeWriteAuditMutations(stepID, domainID, task, before.protectedFiles, afterProtected, changed)
	}

	afterStatuses := snapshotAuditedRepoStatuses(task)
	for root, beforeStatus := range before.repoStatuses {
		afterStatus, ok := afterStatuses[root]
		if !ok {
			e.reportRuntimeWriteAuditRepoSkipped(stepID, domainID, task, root, "status_unavailable_after_runtime")
			violations = append(violations, fmt.Sprintf("%s: repo status unavailable after runtime under %s", runtimeWriteAuditUnexpectedMutation, root))
			continue
		}
		if !sameStringSlice(beforeStatus, afterStatus) {
			violations = append(violations, e.reportRuntimeWriteAuditWarning(stepID, domainID, task, "repo", root, changedRepoStatusPaths(beforeStatus, afterStatus)))
		}
	}

	for _, skipped := range before.skippedRepos {
		e.reportRuntimeWriteAuditRepoSkipped(stepID, domainID, task, skipped.Root, skipped.Reason)
	}
	violations = normalizeAuditPaths(violations)
	if len(violations) == 0 {
		return nil
	}
	if strings.TrimSpace(string(provider)) == "" {
		provider = acpruntime.ProviderClaudeCode
	}
	message := strings.Join(violations, "; ")
	return acpruntime.WrapRunnerError(provider, acpruntime.ErrorCodeRuntimeContract, message, nil)
}

func (e *pipelineExecution) reportRuntimeWriteAuditRepoSkipped(stepID string, domainID string, task acpruntime.Task, root string, reason string) {
	e.addWarning(fmt.Sprintf("%s: repo root skipped (%s): %s", runtimeWriteAuditRepoSkipped, reason, root))
	e.logWarn(stepID, domainID, runtimeWriteAuditRepoSkipped, map[string]any{
		"audit_code": runtimeWriteAuditRepoSkipped,
		"task_id":    task.TaskID,
		"root":       root,
		"reason":     reason,
	})
}

func (e *pipelineExecution) reportRuntimeWriteAuditWarning(stepID string, domainID string, task acpruntime.Task, category string, root string, changed []string) string {
	changed = normalizeAuditPaths(changed)
	if len(changed) == 0 {
		return ""
	}
	limited := changed
	if len(limited) > runtimeWriteAuditMaxPaths {
		limited = limited[:runtimeWriteAuditMaxPaths]
	}
	message := fmt.Sprintf("%s: %s changed %d path(s) under %s", runtimeWriteAuditUnexpectedMutation, category, len(changed), root)
	e.addWarning(message)
	e.logWarn(stepID, domainID, runtimeWriteAuditUnexpectedMutation, map[string]any{
		"audit_code":    runtimeWriteAuditUnexpectedMutation,
		"task_id":       task.TaskID,
		"category":      category,
		"root":          root,
		"changed_count": len(changed),
		"changed_paths": limited,
	})
	return message
}

func snapshotProtectedWorkspaceFiles(workspaceRoot string) map[string]runtimeProtectedFileSnapshot {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return map[string]runtimeProtectedFileSnapshot{}
	}
	absWorkspace, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return map[string]runtimeProtectedFileSnapshot{}
	}
	roots := []string{
		"workspace.yaml",
		filepath.Join("schemas"),
		filepath.Join("docs", "spec"),
		"charter",
	}
	out := map[string]runtimeProtectedFileSnapshot{}
	for _, relRoot := range roots {
		absRoot := filepath.Join(absWorkspace, relRoot)
		info, err := os.Lstat(absRoot)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			if snapshot, ok := protectedFileSnapshot(absRoot); ok {
				out[filepath.ToSlash(relRoot)] = snapshot
			}
			continue
		}
		_ = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry == nil || entry.IsDir() {
				return nil
			}
			snapshot, ok := protectedFileSnapshot(path)
			if !ok {
				return nil
			}
			rel, relErr := filepath.Rel(absWorkspace, path)
			if relErr != nil {
				return nil
			}
			out[filepath.ToSlash(rel)] = snapshot
			return nil
		})
	}
	return out
}

func protectedFileSnapshot(path string) (runtimeProtectedFileSnapshot, bool) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return runtimeProtectedFileSnapshot{}, false
	}
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return runtimeProtectedFileSnapshot{}, false
	}
	sum := sha256.Sum256(content)
	return runtimeProtectedFileSnapshot{
		digest:  hex.EncodeToString(sum[:]),
		content: append([]byte(nil), content...),
		mode:    info.Mode().Perm(),
	}, true
}

func fileDigest(path string) (string, bool) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", false
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), true
}

func snapshotAuditedRepoStatuses(task acpruntime.Task) map[string][]string {
	statuses := map[string][]string{}
	for _, root := range auditedRuntimeRepoRoots(task) {
		status, err := gitStatusSnapshot(root)
		if err != nil {
			continue
		}
		statuses[root] = status
	}
	return statuses
}

func skippedRuntimeWriteAuditRepos(task acpruntime.Task) []runtimeWriteAuditSkippedRepo {
	skipped := []runtimeWriteAuditSkippedRepo{}
	for _, root := range candidateRuntimeRepoRoots(task) {
		absRoot := absClean(root)
		if absRoot == "" || runtimeAuditRootIsExcluded(absRoot, task) {
			continue
		}
		if _, err := gitTopLevel(absRoot); err != nil {
			skipped = append(skipped, runtimeWriteAuditSkippedRepo{Root: absRoot, Reason: "not_git_repo"})
		}
	}
	return skipped
}

func auditedRuntimeRepoRoots(task acpruntime.Task) []string {
	roots := []string{}
	seen := map[string]struct{}{}
	for _, root := range candidateRuntimeRepoRoots(task) {
		absRoot := absClean(root)
		if absRoot == "" || runtimeAuditRootIsExcluded(absRoot, task) {
			continue
		}
		topLevel, err := gitTopLevel(absRoot)
		if err != nil {
			continue
		}
		if runtimeAuditRootIsExcluded(topLevel, task) {
			continue
		}
		if _, ok := seen[topLevel]; ok {
			continue
		}
		seen[topLevel] = struct{}{}
		roots = append(roots, topLevel)
	}
	sort.Strings(roots)
	return roots
}

func candidateRuntimeRepoRoots(task acpruntime.Task) []string {
	roots := append([]string(nil), task.ReadContextRoots...)
	return normalizeAuditPaths(roots)
}

func runtimeAuditRootIsExcluded(root string, task acpruntime.Task) bool {
	for _, excluded := range []string{task.Workspace, task.WriteRoot, task.DraftFinalRoot} {
		absExcluded := absClean(excluded)
		if absExcluded == "" {
			continue
		}
		if pathInsideOrEqual(root, absExcluded) {
			return true
		}
	}
	return false
}

func gitTopLevel(root string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := gitReadOnlyCommand(ctx, root, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	topLevel := strings.TrimSpace(string(output))
	if topLevel == "" {
		return "", errors.New("empty git top-level")
	}
	return absClean(topLevel), nil
}

func gitStatusSnapshot(root string) ([]string, error) {
	if repoRootAppearsReadOnly(root) {
		return gitReadOnlyRepoSnapshot(root)
	}
	return gitPorcelainStatusSnapshot(root)
}

func gitPorcelainStatusSnapshot(root string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := gitReadOnlyCommand(ctx, root, "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil {
		return nil, err
	}
	lines := []string{}
	for _, line := range strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	sort.Strings(lines)
	return lines, nil
}

func gitReadOnlyRepoSnapshot(root string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := gitReadOnlyCommand(ctx, root, "rev-parse", "HEAD").Output()
	if err != nil {
		return nil, err
	}
	head := strings.TrimSpace(string(output))
	if head == "" {
		return nil, errors.New("empty git HEAD")
	}

	lines := []string{"readonly:head:" + head}
	for _, rel := range []string{".", ".git", filepath.Join(".git", "HEAD"), filepath.Join(".git", "index"), filepath.Join(".git", "config")} {
		path := filepath.Join(root, rel)
		info, statErr := os.Lstat(path)
		if statErr != nil {
			lines = append(lines, "readonly:missing:"+filepath.ToSlash(rel))
			continue
		}
		lines = append(lines, fmt.Sprintf("readonly:mode:%s:%04o", filepath.ToSlash(rel), info.Mode().Perm()))
		if !info.IsDir() {
			if digest, ok := fileDigest(path); ok {
				lines = append(lines, "readonly:digest:"+filepath.ToSlash(rel)+":"+digest)
			}
		}
	}
	if _, err := os.Lstat(filepath.Join(root, ".git", "index.lock")); err == nil {
		lines = append(lines, "readonly:index-lock:present")
	} else {
		lines = append(lines, "readonly:index-lock:absent")
	}
	sort.Strings(lines)
	return lines, nil
}

func gitReadOnlyCommand(ctx context.Context, root string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	return cmd
}

func repoRootAppearsReadOnly(root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	checked := 0
	for _, rel := range []string{".", ".git", filepath.Join(".git", "index")} {
		info, err := os.Lstat(filepath.Join(root, rel))
		if err != nil {
			return false
		}
		checked++
		if info.Mode().Perm()&0o222 != 0 {
			return false
		}
	}
	return checked > 0
}

func changedProtectedSnapshotPaths(before map[string]runtimeProtectedFileSnapshot, after map[string]runtimeProtectedFileSnapshot) []string {
	seen := map[string]struct{}{}
	for path := range before {
		seen[path] = struct{}{}
	}
	for path := range after {
		seen[path] = struct{}{}
	}
	changed := []string{}
	for path := range seen {
		beforeSnapshot, beforeOK := before[path]
		afterSnapshot, afterOK := after[path]
		if !beforeOK || !afterOK || beforeSnapshot.digest != afterSnapshot.digest || beforeSnapshot.mode != afterSnapshot.mode {
			changed = append(changed, path)
		}
	}
	return normalizeAuditPaths(changed)
}

// changedSnapshotPaths retains the small digest-map helper used by older
// package tests and callers; protected workspace audits use the richer typed
// snapshot above so file modes can also be restored.
func changedSnapshotPaths(before map[string]string, after map[string]string) []string {
	seen := map[string]struct{}{}
	for path := range before {
		seen[path] = struct{}{}
	}
	for path := range after {
		seen[path] = struct{}{}
	}
	changed := []string{}
	for path := range seen {
		if before[path] != after[path] {
			changed = append(changed, path)
		}
	}
	return normalizeAuditPaths(changed)
}

// restoreRuntimeWriteAuditMutations removes provider writes from protected
// workspace surfaces after the audit has observed them. A restore is only
// attempted when the file still matches the post-run snapshot; if another
// actor changed it after the provider exited, we leave it untouched and make
// the conflict visible in the run log.
func (e *pipelineExecution) restoreRuntimeWriteAuditMutations(
	stepID string,
	domainID string,
	task acpruntime.Task,
	before map[string]runtimeProtectedFileSnapshot,
	after map[string]runtimeProtectedFileSnapshot,
	changed []string,
) {
	restored := []string{}
	conflicts := []string{}
	for _, rel := range normalizeAuditPaths(changed) {
		beforeSnapshot, hadBefore := before[rel]
		afterSnapshot, hadAfter := after[rel]
		path := filepath.Join(absClean(task.Workspace), filepath.FromSlash(rel))
		current, currentOK := protectedFileSnapshot(path)
		if hadAfter {
			if !currentOK || current.digest != afterSnapshot.digest || current.mode != afterSnapshot.mode {
				conflicts = append(conflicts, rel)
				continue
			}
		} else if currentOK {
			conflicts = append(conflicts, rel)
			continue
		}

		if hadBefore {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				conflicts = append(conflicts, rel)
				continue
			}
			if err := os.WriteFile(path, beforeSnapshot.content, beforeSnapshot.mode); err != nil {
				conflicts = append(conflicts, rel)
				continue
			}
			if err := os.Chmod(path, beforeSnapshot.mode); err != nil {
				conflicts = append(conflicts, rel)
				continue
			}
			restored = append(restored, rel)
			continue
		}

		if !hadAfter {
			if _, err := os.Lstat(path); err == nil {
				conflicts = append(conflicts, rel)
				continue
			} else if !errors.Is(err, os.ErrNotExist) {
				conflicts = append(conflicts, rel)
				continue
			}
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			conflicts = append(conflicts, rel)
			continue
		}
		restored = append(restored, rel)
	}

	if len(restored) > 0 {
		e.logInfo(stepID, domainID, runtimeWriteAuditRestoredMutation, map[string]any{
			"audit_code":     runtimeWriteAuditRestoredMutation,
			"task_id":        task.TaskID,
			"restored_count": len(restored),
			"restored_paths": limitAuditPaths(restored),
		})
	}
	if len(conflicts) > 0 {
		e.addWarning(fmt.Sprintf("%s: protected workspace paths changed after runtime and were left untouched", runtimeWriteAuditRestoreConflict))
		e.logWarn(stepID, domainID, runtimeWriteAuditRestoreConflict, map[string]any{
			"audit_code":     runtimeWriteAuditRestoreConflict,
			"task_id":        task.TaskID,
			"conflict_count": len(conflicts),
			"conflict_paths": limitAuditPaths(conflicts),
		})
	}
}

func limitAuditPaths(paths []string) []string {
	paths = normalizeAuditPaths(paths)
	if len(paths) > runtimeWriteAuditMaxPaths {
		return paths[:runtimeWriteAuditMaxPaths]
	}
	return paths
}

func changedRepoStatusPaths(before []string, after []string) []string {
	seen := map[string]struct{}{}
	for _, line := range append(append([]string(nil), before...), after...) {
		line = strings.TrimRight(line, "\r\n")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(line) > 3 && line[2] == ' ' {
			line = strings.TrimSpace(line[3:])
		} else {
			line = strings.TrimSpace(line)
		}
		seen[line] = struct{}{}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, filepath.ToSlash(path))
	}
	return normalizeAuditPaths(paths)
}

func normalizeAuditPaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, pathValue := range paths {
		pathValue = filepath.ToSlash(strings.TrimSpace(pathValue))
		if pathValue == "" {
			continue
		}
		if _, ok := seen[pathValue]; ok {
			continue
		}
		seen[pathValue] = struct{}{}
		out = append(out, pathValue)
	}
	sort.Strings(out)
	return out
}

func sameStringSlice(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func absClean(pathValue string) string {
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		return ""
	}
	abs, err := filepath.Abs(pathValue)
	if err != nil {
		return ""
	}
	return filepath.Clean(abs)
}

func pathInsideOrEqual(pathValue string, root string) bool {
	pathValue = absClean(pathValue)
	root = absClean(root)
	if pathValue == "" || root == "" {
		return false
	}
	if pathValue == root {
		return true
	}
	rel, err := filepath.Rel(root, pathValue)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}
