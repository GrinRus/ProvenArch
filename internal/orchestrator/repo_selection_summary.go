package orchestrator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type repoSelectionSummaryArtifact struct {
	Version            int                               `json:"version"`
	RunID              string                            `json:"run_id"`
	Pipeline           string                            `json:"pipeline"`
	RepoSelectionMode  string                            `json:"repo_selection_mode"`
	GeneratedAt        string                            `json:"generated_at"`
	SelectedRepoScopes []string                          `json:"selected_repo_scopes"`
	Decisions          []workspace.RepoSelectionDecision `json:"decisions"`
}

func (e *pipelineExecution) persistRepoSelectionSummary() error {
	mode := strings.TrimSpace(e.repoSelectionMode)
	if mode == "" {
		mode = workspace.RepoSelectionAll
	}

	selectedScopes := normalizeOrderedUniqueStrings(e.selectedRepoScopes)
	sort.Strings(selectedScopes)

	decisions := append([]workspace.RepoSelectionDecision(nil), e.repoSelection...)
	sort.Slice(decisions, func(i, j int) bool {
		return strings.TrimSpace(decisions[i].Name) < strings.TrimSpace(decisions[j].Name)
	})

	payload := repoSelectionSummaryArtifact{
		Version:            1,
		RunID:              e.runID,
		Pipeline:           string(e.pipeline),
		RepoSelectionMode:  mode,
		GeneratedAt:        e.clock().UTC().Format(time.RFC3339),
		SelectedRepoScopes: append([]string(nil), selectedScopes...),
		Decisions:          decisions,
	}

	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal repo selection summary: %w", err)
	}
	content = append(content, '\n')

	path := fmt.Sprintf("reports/taskruns/%s-repo-selection-summary.json", e.runID)
	if err := e.workspace.WriteFile(path, content); err != nil {
		return err
	}
	e.addArtifacts(Artifact{Path: path, Kind: "taskrun", Label: "Repo Selection Summary"})
	e.logInfo("", "", "repo selection summary persisted", map[string]any{
		"path":                path,
		"repo_selection_mode": mode,
		"selected_count":      len(selectedScopes),
	})
	return nil
}
