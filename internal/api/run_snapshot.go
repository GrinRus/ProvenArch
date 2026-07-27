package api

import (
	"fmt"
	"path"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type runSnapshotStatus string

const (
	runSnapshotAvailable   runSnapshotStatus = "available"
	runSnapshotPartial     runSnapshotStatus = "partial"
	runSnapshotNotProduced runSnapshotStatus = "not_produced"
	runSnapshotUnavailable runSnapshotStatus = "unavailable"
	runSnapshotError       runSnapshotStatus = "error"
)

type runSnapshotIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

type runSnapshotArtifact struct {
	ID            string `json:"id,omitempty"`
	Path          string `json:"path"`
	ReadPath      string `json:"read_path"`
	CanonicalPath string `json:"canonical_path,omitempty"`
	Kind          string `json:"kind"`
	Label         string `json:"label"`
	SourceRunID   string `json:"source_run_id"`
	SourceMode    string `json:"source_mode"`
}

type runSnapshotResponse struct {
	RunID     string                `json:"run_id"`
	Status    runSnapshotStatus     `json:"status"`
	Artifacts []runSnapshotArtifact `json:"artifacts"`
	Issues    []runSnapshotIssue    `json:"issues"`
}

func resolveRunSnapshot(ws workspace.Root, runID string, runArtifacts []orchestrator.Artifact) runSnapshotResponse {
	response := runSnapshotResponse{
		RunID:     runID,
		Status:    runSnapshotNotProduced,
		Artifacts: []runSnapshotArtifact{},
		Issues:    []runSnapshotIssue{},
	}
	indexPath := runFinalIndexPath(runID)
	inventory := make(map[string]struct{}, len(runArtifacts))
	for _, artifact := range runArtifacts {
		normalized, ok := normalizeSnapshotPath(artifact.Path)
		if ok {
			inventory[normalized] = struct{}{}
		}
	}
	if _, ok := inventory[indexPath]; !ok {
		response.Issues = append(response.Issues, runSnapshotIssue{
			Code:    "snapshot_not_produced",
			Message: fmt.Sprintf("run %s has no authoritative final snapshot index", runID),
			Path:    indexPath,
		})
		return response
	}

	raw, err := ws.ReadFile(indexPath)
	if err != nil {
		response.Status = runSnapshotUnavailable
		response.Issues = append(response.Issues, runSnapshotIssue{
			Code:    "snapshot_index_unavailable",
			Message: err.Error(),
			Path:    indexPath,
		})
		return response
	}
	index, err := contracts.ParseFinalRunIndex(raw)
	if err != nil {
		response.Status = runSnapshotError
		response.Issues = append(response.Issues, runSnapshotIssue{
			Code:    "snapshot_index_invalid",
			Message: err.Error(),
			Path:    indexPath,
		})
		return response
	}
	if strings.TrimSpace(index.RunID) != runID {
		response.Status = runSnapshotError
		response.Issues = append(response.Issues, runSnapshotIssue{
			Code:    "snapshot_run_mismatch",
			Message: fmt.Sprintf("final snapshot run_id %q does not match selected run %q", index.RunID, runID),
			Path:    indexPath,
		})
		return response
	}

	response.Artifacts = append(response.Artifacts, runSnapshotArtifact{
		Path:        indexPath,
		ReadPath:    indexPath,
		Kind:        "taskrun",
		Label:       "Final run index",
		SourceRunID: runID,
		SourceMode:  "run_snapshot",
	})
	canonicalMappings := map[string]string{}
	for _, document := range index.CanonicalDocuments {
		canonicalPath, canonicalOK := normalizeSnapshotPath(document.CanonicalPath)
		stagedPath, stagedOK := normalizeSnapshotPath(document.StagedPath)
		if !canonicalOK || !stagedOK || !isRunFinalPath(runID, stagedPath) {
			response.Status = runSnapshotError
			response.Issues = append(response.Issues, runSnapshotIssue{
				Code:    "snapshot_document_path_invalid",
				Message: "final snapshot document has a non-normalized or foreign path",
				Path:    strings.TrimSpace(document.StagedPath),
			})
			return response
		}
		if previous, exists := canonicalMappings[canonicalPath]; exists {
			response.Status = runSnapshotError
			response.Issues = append(response.Issues, runSnapshotIssue{
				Code:    "snapshot_canonical_mapping_ambiguous",
				Message: fmt.Sprintf("canonical path %q maps more than once (%q and %q)", canonicalPath, previous, stagedPath),
				Path:    canonicalPath,
			})
			return response
		}
		canonicalMappings[canonicalPath] = stagedPath
		if _, ok := inventory[stagedPath]; !ok {
			response.Issues = append(response.Issues, runSnapshotIssue{
				Code:    "snapshot_document_not_in_inventory",
				Message: "indexed document is absent from the selected run artifact inventory",
				Path:    stagedPath,
			})
			continue
		}
		kind := strings.TrimSpace(document.Kind)
		if kind == "" {
			kind = "report"
		}
		label := strings.TrimSpace(document.Title)
		if label == "" {
			label = canonicalPath
		}
		response.Artifacts = append(response.Artifacts, runSnapshotArtifact{
			ID:            strings.TrimSpace(document.ID),
			Path:          canonicalPath,
			ReadPath:      stagedPath,
			CanonicalPath: canonicalPath,
			Kind:          kind,
			Label:         label,
			SourceRunID:   runID,
			SourceMode:    "run_snapshot",
		})
		if _, err := ws.ReadFile(stagedPath); err != nil {
			response.Issues = append(response.Issues, runSnapshotIssue{
				Code:    "snapshot_document_unavailable",
				Message: err.Error(),
				Path:    stagedPath,
			})
		}
	}

	if citationPath := strings.TrimSpace(index.CitationIndexPath); citationPath != "" {
		normalized, ok := normalizeSnapshotPath(citationPath)
		if !ok || !isRunFinalPath(runID, normalized) {
			response.Status = runSnapshotError
			response.Issues = append(response.Issues, runSnapshotIssue{
				Code:    "snapshot_citation_index_path_invalid",
				Message: "citation index path is non-normalized or belongs to another run",
				Path:    citationPath,
			})
			return response
		}
		if _, exists := inventory[normalized]; !exists {
			response.Issues = append(response.Issues, runSnapshotIssue{
				Code:    "snapshot_citation_index_not_in_inventory",
				Message: "citation index is absent from the selected run artifact inventory",
				Path:    normalized,
			})
		} else if _, err := ws.ReadFile(normalized); err != nil {
			response.Issues = append(response.Issues, runSnapshotIssue{
				Code:    "snapshot_citation_index_unavailable",
				Message: err.Error(),
				Path:    normalized,
			})
		} else {
			response.Artifacts = append(response.Artifacts, runSnapshotArtifact{
				Path:        normalized,
				ReadPath:    normalized,
				Kind:        "taskrun",
				Label:       "Citation index",
				SourceRunID: runID,
				SourceMode:  "run_snapshot",
			})
		}
	}

	if len(response.Issues) > 0 {
		response.Status = runSnapshotPartial
	} else {
		response.Status = runSnapshotAvailable
	}
	return response
}

func runFinalIndexPath(runID string) string {
	return path.Join("reports", "taskruns", runID, "staging", "final", "final-run-index.json")
}

func isRunFinalPath(runID string, value string) bool {
	prefix := path.Join("reports", "taskruns", runID, "staging", "final") + "/"
	return strings.HasPrefix(value, prefix)
}

func normalizeSnapshotPath(value string) (string, bool) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if trimmed == "" || strings.HasPrefix(trimmed, "/") {
		return "", false
	}
	clean := path.Clean(trimmed)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != trimmed {
		return "", false
	}
	return clean, true
}
