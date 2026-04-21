package contracts

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type RuntimeOutputRefs struct {
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

type RuntimeExecution struct {
	Version           int               `json:"version"`
	TaskID            string            `json:"task_id"`
	RunID             string            `json:"run_id"`
	StepID            string            `json:"step_id"`
	ShardID           string            `json:"shard_id,omitempty"`
	DomainID          string            `json:"domain_id,omitempty"`
	Provider          string            `json:"provider"`
	RuntimeVersion    string            `json:"runtime_version,omitempty"`
	StartedAt         string            `json:"started_at"`
	FinishedAt        string            `json:"finished_at"`
	RepoScope         string            `json:"repo_scope,omitempty"`
	RepoScopes        []string          `json:"repo_scopes,omitempty"`
	PathScopes        []string          `json:"path_scopes,omitempty"`
	ArtifactRoot      string            `json:"artifact_root,omitempty"`
	WriteRoot         string            `json:"write_root,omitempty"`
	DraftFinalRoot    string            `json:"draft_final_root,omitempty"`
	Status            string            `json:"status"`
	RequiredArtifacts []string          `json:"required_artifacts,omitempty"`
	Warnings          []string          `json:"warnings,omitempty"`
	RawOutputRefs     RuntimeOutputRefs `json:"raw_output_refs,omitempty"`
}

func ParseRuntimeExecution(raw []byte) (RuntimeExecution, error) {
	var execution RuntimeExecution
	if len(raw) == 0 {
		return execution, fmt.Errorf("runtime execution payload is required")
	}
	if err := json.Unmarshal(raw, &execution); err != nil {
		return execution, fmt.Errorf("decode runtime execution: %w", err)
	}
	if err := validateRuntimeExecution(execution); err != nil {
		return RuntimeExecution{}, err
	}
	return execution, nil
}

func NormalizeRuntimeExecution(execution RuntimeExecution) RuntimeExecution {
	normalized := execution
	if normalized.Version == 0 {
		normalized.Version = 1
	}
	normalized.RepoScopes = uniquePreserveOrderStrings(normalized.RepoScopes)
	normalized.PathScopes = uniquePreserveOrderStrings(normalized.PathScopes)
	normalized.RequiredArtifacts = uniquePreserveOrderStrings(normalized.RequiredArtifacts)
	normalized.Warnings = uniquePreserveOrderStrings(normalized.Warnings)
	return normalized
}

func validateRuntimeExecution(execution RuntimeExecution) error {
	problems := []string{}
	if execution.Version != 1 {
		problems = append(problems, "version must be 1")
	}
	if strings.TrimSpace(execution.TaskID) == "" {
		problems = append(problems, "task_id is required")
	}
	if strings.TrimSpace(execution.RunID) == "" {
		problems = append(problems, "run_id is required")
	}
	if strings.TrimSpace(execution.StepID) == "" {
		problems = append(problems, "step_id is required")
	}
	if strings.TrimSpace(execution.Provider) == "" {
		problems = append(problems, "provider is required")
	}
	if strings.TrimSpace(execution.StartedAt) == "" {
		problems = append(problems, "started_at is required")
	}
	if strings.TrimSpace(execution.FinishedAt) == "" {
		problems = append(problems, "finished_at is required")
	}
	switch strings.TrimSpace(execution.Status) {
	case "succeeded", "failed", "canceled", "timeout":
	default:
		problems = append(problems, "status must be one of succeeded, failed, canceled, timeout")
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("runtime execution is invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}

func uniquePreserveOrderStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
