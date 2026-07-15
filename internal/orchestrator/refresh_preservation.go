package orchestrator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func (e *pipelineExecution) preserveUnaffectedStagedDocuments() error {
	if e.refreshExecution == nil || e.refreshExecution.Mode != "affected_only" || e.refreshExecution.BaselineRunID == nil || e.finalRunIndex == nil {
		return nil
	}
	baselineRunID := *e.refreshExecution.BaselineRunID
	finalRaw, err := e.workspace.ReadFile(runtimeFinalRunIndexPath(baselineRunID))
	if err != nil {
		return fmt.Errorf("read baseline final index: %w", err)
	}
	baseline, err := contracts.ParseFinalRunIndex(finalRaw)
	if err != nil {
		return err
	}
	baselineByPath := map[string]contracts.FinalRunDocument{}
	for _, doc := range baseline.CanonicalDocuments {
		baselineByPath[doc.CanonicalPath] = doc
	}
	candidates := map[string]struct{}{}
	for _, path := range e.preservedArtifactCandidates {
		candidates[path] = struct{}{}
	}
	for _, current := range e.finalRunIndex.CanonicalDocuments {
		if _, ok := candidates[current.CanonicalPath]; !ok {
			continue
		}
		previous, ok := baselineByPath[current.CanonicalPath]
		if !ok || !sameDocumentBinding(previous, current) {
			continue
		}
		content, err := e.workspace.ReadFile(current.CanonicalPath)
		if err != nil {
			continue
		}
		if err := e.workspace.WriteFile(current.StagedPath, content); err != nil {
			return err
		}
		e.preservedCanonicalPaths = append(e.preservedCanonicalPaths, current.CanonicalPath)
	}
	sort.Strings(e.preservedCanonicalPaths)
	return nil
}

func sameDocumentBinding(left, right contracts.FinalRunDocument) bool {
	if left.ID != right.ID || left.Kind != right.Kind {
		return false
	}
	return equalSorted(left.CitationIDs, right.CitationIDs) && equalSorted(left.SourceShards, right.SourceShards)
}

func equalSorted(left, right []string) bool {
	a := append([]string(nil), left...)
	b := append([]string(nil), right...)
	sort.Strings(a)
	sort.Strings(b)
	return strings.Join(a, "\x00") == strings.Join(b, "\x00")
}
