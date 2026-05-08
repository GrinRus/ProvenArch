package orchestrator

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func (e *pipelineExecution) promoteValidatedArtifacts() error {
	if e.finalRunIndex == nil {
		return fmt.Errorf("promote validated artifacts: final run index is missing")
	}
	if e.validatorVerdict == nil {
		return fmt.Errorf("promote validated artifacts: validator verdict is missing")
	}
	if e.validatorVerdict.Verdict != "PASS" {
		return fmt.Errorf("promote validated artifacts: validator verdict is %s", e.validatorVerdict.Verdict)
	}

	expectedCanonicalPaths := map[string]struct{}{}
	for _, document := range e.finalRunIndex.CanonicalDocuments {
		expectedCanonicalPaths[document.CanonicalPath] = struct{}{}
		content, err := e.workspace.ReadFile(document.StagedPath)
		if err != nil {
			return fmt.Errorf("read staged artifact %q: %w", document.StagedPath, err)
		}
		if err := e.workspace.WriteFile(document.CanonicalPath, content); err != nil {
			return fmt.Errorf("promote artifact %q: %w", document.CanonicalPath, err)
		}
		e.addArtifacts(Artifact{
			Path:  document.CanonicalPath,
			Kind:  document.Kind,
			Label: document.Title,
		})
	}
	if err := e.removeStaleManagedCanonicalArtifacts(expectedCanonicalPaths); err != nil {
		return err
	}

	if err := e.rebuildDerivedModel(); err != nil {
		return err
	}

	entities, err := e.store.ListEntities()
	if err != nil {
		return err
	}
	edges, err := e.store.ListEdges()
	if err != nil {
		return err
	}
	diagramArtifacts, err := e.compiler.CompileC4Diagrams(entities, edges)
	if err != nil {
		return err
	}
	e.addArtifacts(toOrchestratorArtifacts(diagramArtifacts)...)
	e.logInfo(e.stepStatus.CurrentStep, "", "validated artifacts promoted", map[string]any{
		"canonical_docs": len(e.finalRunIndex.CanonicalDocuments),
	})
	return nil
}

func (e *pipelineExecution) removeStaleManagedCanonicalArtifacts(expected map[string]struct{}) error {
	stale := map[string]struct{}{}
	for _, prefix := range managedCanonicalArtifactPrefixes() {
		absRoot, err := e.workspace.Resolve(prefix)
		if err != nil {
			return err
		}
		if _, err := os.Stat(absRoot); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("inspect managed canonical surface %q: %w", prefix, err)
		}
		if err := filepath.WalkDir(absRoot, func(item string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			relPath, err := filepath.Rel(e.workspace.Path, item)
			if err != nil {
				return err
			}
			canonicalPath := filepath.ToSlash(relPath)
			if _, ok := expected[canonicalPath]; !ok {
				stale[canonicalPath] = struct{}{}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("walk managed canonical surface %q: %w", prefix, err)
		}
	}
	if len(stale) == 0 {
		return nil
	}
	paths := make([]string, 0, len(stale))
	for canonicalPath := range stale {
		paths = append(paths, canonicalPath)
	}
	sort.Strings(paths)
	for _, canonicalPath := range paths {
		target, err := e.workspace.Resolve(canonicalPath)
		if err != nil {
			return err
		}
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove stale canonical artifact %q: %w", canonicalPath, err)
		}
	}
	e.removeArtifactsByPath(paths...)
	return nil
}

func managedCanonicalArtifactPrefixes() []string {
	return []string{
		"reports/as-is",
		"reports/coverage",
		"reports/findings",
		"reports/agent-outputs",
		"reports/diagrams",
		"proposals",
	}
}

func (e *pipelineExecution) rebuildDerivedModel() error {
	if e.finalRunIndex == nil {
		return fmt.Errorf("rebuild derived model: final run index is missing")
	}
	for _, rel := range []string{"model/entities", "model/edges"} {
		abs, err := e.workspace.Resolve(rel)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(abs); err != nil {
			return fmt.Errorf("clear derived model dir %q: %w", rel, err)
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return fmt.Errorf("recreate derived model dir %q: %w", rel, err)
		}
	}
	_, err := e.store.ApplySemanticSnapshot(contracts.SemanticSnapshot{
		Entities: e.finalRunIndex.Semantic.Entities,
		Edges:    e.finalRunIndex.Semantic.Edges,
	})
	return err
}
