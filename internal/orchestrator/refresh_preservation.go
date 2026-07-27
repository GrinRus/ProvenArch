package orchestrator

import (
	"fmt"
	"path"
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
		if err := validatePreservedDocumentReferences(current, e.finalRunIndex.CanonicalDocuments, e.citationIndex); err != nil {
			return err
		}
		content, err := e.readBaselineStagedDocument(baselineRunID, previous)
		if err != nil {
			return err
		}
		if err := e.workspace.WriteFileAtomic(current.StagedPath, content); err != nil {
			return err
		}
		e.preservedCanonicalPaths = append(e.preservedCanonicalPaths, current.CanonicalPath)
	}
	sort.Strings(e.preservedCanonicalPaths)
	return nil
}

func validatePreservedDocumentReferences(
	document contracts.FinalRunDocument,
	documents []contracts.FinalRunDocument,
	citationIndex *contracts.CitationIndex,
) error {
	if citationIndex == nil {
		return fmt.Errorf("cannot preserve %q without current citation index", document.CanonicalPath)
	}
	documentIDs := map[string]struct{}{}
	for _, candidate := range documents {
		documentIDs[strings.TrimSpace(candidate.ID)] = struct{}{}
	}
	citations := map[string]contracts.DocumentCitation{}
	for _, citation := range citationIndex.Citations {
		citations[strings.TrimSpace(citation.ID)] = citation
		for _, referencedDocumentID := range citation.DocumentIDs {
			if _, ok := documentIDs[strings.TrimSpace(referencedDocumentID)]; !ok {
				return fmt.Errorf("cannot preserve %q: citation %q references removed document %q", document.CanonicalPath, citation.ID, referencedDocumentID)
			}
		}
	}
	for _, citationID := range document.CitationIDs {
		citation, ok := citations[strings.TrimSpace(citationID)]
		if !ok {
			return fmt.Errorf("cannot preserve %q: citation %q was removed", document.CanonicalPath, citationID)
		}
		if !containsTrimmedString(citation.DocumentIDs, document.ID) {
			return fmt.Errorf("cannot preserve %q: citation %q no longer references document %q", document.CanonicalPath, citationID, document.ID)
		}
	}
	return nil
}

func (e *pipelineExecution) readBaselineStagedDocument(baselineRunID string, document contracts.FinalRunDocument) ([]byte, error) {
	root := runtimeFinalArtifactRoot(baselineRunID)
	staged := path.Clean(strings.TrimSpace(document.StagedPath))
	if staged == "." || (staged != root && !strings.HasPrefix(staged, root+"/")) {
		return nil, fmt.Errorf("baseline staged path %q is outside immutable final snapshot", document.StagedPath)
	}
	raw, err := e.workspace.ReadFile(staged)
	if err != nil {
		return nil, fmt.Errorf("read baseline staged document %q: %w", staged, err)
	}
	return raw, nil
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
