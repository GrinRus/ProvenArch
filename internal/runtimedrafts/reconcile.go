package runtimedrafts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/runtime/compatibilityregistry"
)

func ReconcileOutputsAtDraftRoot(draftRoot string, manifest Manifest) (bool, error) {
	draftRoot = strings.TrimSpace(draftRoot)
	if draftRoot == "" {
		return false, fmt.Errorf("runtime draft root is empty")
	}
	cleanDraftRoot := filepath.Clean(draftRoot)
	changed := false
	for idx, output := range manifest.Outputs {
		reconciled, err := reconcileOutputAtDraftRoot(cleanDraftRoot, output)
		if err != nil {
			return changed, fmt.Errorf("%s: runtime draft manifest outputs[%d].path: %w", compatibilityregistry.RuleDraftRootReconcileExistingOutputs, idx, err)
		}
		if reconciled {
			changed = true
		}
	}
	return changed, nil
}

func reconcileOutputAtDraftRoot(draftRoot string, output Output) (bool, error) {
	relPath := filepath.Clean(strings.TrimSpace(output.Path))
	if relPath == "" || relPath == "." {
		return false, fmt.Errorf("must not be empty")
	}
	expectedPath := filepath.Join(filepath.Clean(draftRoot), relPath)
	if _, err := os.Stat(expectedPath); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	canonicalRel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(output.CanonicalPath)))
	if canonicalRel == "" || canonicalRel == "." {
		return false, nil
	}
	canonicalPath := filepath.Join(filepath.Clean(draftRoot), canonicalRel)
	if filepath.Clean(canonicalPath) == filepath.Clean(expectedPath) {
		return false, nil
	}

	content, mode, err := readCanonicalDraftFallback(canonicalPath, output.CanonicalPath)
	if err != nil {
		return false, err
	}
	if len(content) == 0 && mode == 0 {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(expectedPath), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(expectedPath, content, mode); err != nil {
		return false, err
	}
	return true, nil
}

func readCanonicalDraftFallback(canonicalPath string, originalPath string) ([]byte, os.FileMode, error) {
	info, err := os.Stat(canonicalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	if info.IsDir() {
		return nil, 0, fmt.Errorf("canonical draft fallback %q must point to a file", originalPath)
	}
	content, err := os.ReadFile(canonicalPath)
	if err != nil {
		return nil, 0, err
	}
	return content, info.Mode().Perm(), nil
}
