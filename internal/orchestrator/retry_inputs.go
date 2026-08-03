package orchestrator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func copyRetryStaging(ws workspace.Root, parentRunID string, childRunID string, resumeStep string, requestedScopes []string) error {
	parentRunID = strings.TrimSpace(parentRunID)
	childRunID = strings.TrimSpace(childRunID)
	if parentRunID == "" || childRunID == "" || parentRunID == childRunID {
		return fmt.Errorf("retry parent and child run ids must be distinct and non-empty")
	}
	sourceRel := filepath.ToSlash(filepath.Join("reports", "taskruns", parentRunID, "staging"))
	sourceAbs, err := ws.Resolve(sourceRel)
	if err != nil {
		return err
	}
	info, err := os.Stat(sourceAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("parent staging is unavailable")
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("parent staging is not a directory")
	}
	return filepath.WalkDir(sourceAbs, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("retry staging contains unsupported symlink %q", entry.Name())
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(sourceAbs, current)
		if err != nil {
			return err
		}
		if !retryStagingPathReusable(filepath.ToSlash(rel), resumeStep, requestedScopes) {
			return nil
		}
		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		targetRel := filepath.ToSlash(filepath.Join("reports", "taskruns", childRunID, "staging", rel))
		if err := ws.WriteFile(targetRel, content); err != nil {
			return err
		}
		return nil
	})
}

func retryStagingPathReusable(relPath, resumeStep string, requestedScopes []string) bool {
	relPath = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(relPath)), "./")
	resumeStep = strings.ToLower(strings.TrimSpace(resumeStep))
	if relPath == "" || relPath == "." || strings.HasPrefix(relPath, "../") {
		return false
	}
	switch {
	case strings.Contains(resumeStep, "constitution"):
		return false
	case strings.Contains(resumeStep, "collect"):
		if len(requestedScopes) == 0 || !strings.HasPrefix(relPath, "shards/") {
			return false
		}
		for _, scope := range requestedScopes {
			scope = strings.ToLower(strings.TrimSpace(scope))
			if scope != "" && strings.Contains(strings.ToLower(relPath), scope) {
				return false
			}
		}
		return true
	case strings.Contains(resumeStep, "asis"):
		return strings.HasPrefix(relPath, "shards/")
	case strings.Contains(resumeStep, "findings"):
		return !strings.Contains(relPath, "reports/findings/") && !strings.Contains(relPath, "reports/coverage/") && !strings.Contains(relPath, "proposals/") && !strings.Contains(relPath, "reports/changelog/")
	case strings.Contains(resumeStep, "proposals"):
		return !strings.Contains(relPath, "proposals/") && !strings.Contains(relPath, "reports/changelog/")
	default:
		return false
	}
}
