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
	runtimeWriteAuditMaxPaths           = 20
)

type runtimeWriteAuditSnapshot struct {
	protectedFiles map[string]string
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

func (e *pipelineExecution) completeRuntimeWriteAudit(stepID string, domainID string, task acpruntime.Task, before runtimeWriteAuditSnapshot) {
	afterProtected := snapshotProtectedWorkspaceFiles(task.Workspace)
	if changed := changedSnapshotPaths(before.protectedFiles, afterProtected); len(changed) > 0 {
		e.reportRuntimeWriteAuditWarning(stepID, domainID, task, "workspace", strings.TrimSpace(task.Workspace), changed)
	}

	afterStatuses := snapshotAuditedRepoStatuses(task)
	for root, beforeStatus := range before.repoStatuses {
		afterStatus, ok := afterStatuses[root]
		if !ok {
			afterStatus = []string{"<repo status unavailable after runtime>"}
		}
		if !sameStringSlice(beforeStatus, afterStatus) {
			e.reportRuntimeWriteAuditWarning(stepID, domainID, task, "repo", root, changedRepoStatusPaths(beforeStatus, afterStatus))
		}
	}

	for _, skipped := range before.skippedRepos {
		e.addWarning(fmt.Sprintf("%s: repo root skipped (%s): %s", runtimeWriteAuditRepoSkipped, skipped.Reason, skipped.Root))
		e.logWarn(stepID, domainID, runtimeWriteAuditRepoSkipped, map[string]any{
			"audit_code": runtimeWriteAuditRepoSkipped,
			"task_id":    task.TaskID,
			"root":       skipped.Root,
			"reason":     skipped.Reason,
		})
	}
}

func (e *pipelineExecution) reportRuntimeWriteAuditWarning(stepID string, domainID string, task acpruntime.Task, category string, root string, changed []string) {
	changed = normalizeAuditPaths(changed)
	if len(changed) == 0 {
		return
	}
	limited := changed
	if len(limited) > runtimeWriteAuditMaxPaths {
		limited = limited[:runtimeWriteAuditMaxPaths]
	}
	e.addWarning(fmt.Sprintf("%s: %s changed %d path(s) under %s", runtimeWriteAuditUnexpectedMutation, category, len(changed), root))
	e.logWarn(stepID, domainID, runtimeWriteAuditUnexpectedMutation, map[string]any{
		"audit_code":    runtimeWriteAuditUnexpectedMutation,
		"task_id":       task.TaskID,
		"category":      category,
		"root":          root,
		"changed_count": len(changed),
		"changed_paths": limited,
	})
}

func snapshotProtectedWorkspaceFiles(workspaceRoot string) map[string]string {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return map[string]string{}
	}
	absWorkspace, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return map[string]string{}
	}
	roots := []string{
		"workspace.yaml",
		filepath.Join("schemas"),
		filepath.Join("docs", "spec"),
		"charter",
	}
	out := map[string]string{}
	for _, relRoot := range roots {
		absRoot := filepath.Join(absWorkspace, relRoot)
		info, err := os.Lstat(absRoot)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			if digest, ok := fileDigest(absRoot); ok {
				out[filepath.ToSlash(relRoot)] = digest
			}
			continue
		}
		_ = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry == nil || entry.IsDir() {
				return nil
			}
			digest, ok := fileDigest(path)
			if !ok {
				return nil
			}
			rel, relErr := filepath.Rel(absWorkspace, path)
			if relErr != nil {
				return nil
			}
			out[filepath.ToSlash(rel)] = digest
			return nil
		})
	}
	return out
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
	output, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--show-toplevel").Output()
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain", "--untracked-files=all").Output()
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

func changedRepoStatusPaths(before []string, after []string) []string {
	seen := map[string]struct{}{}
	for _, line := range append(append([]string(nil), before...), after...) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 3 {
			line = strings.TrimSpace(line[3:])
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
