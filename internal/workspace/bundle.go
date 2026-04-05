package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type subagentsBundle struct {
	Agents []subagentDefinition `yaml:"agents"`
}

type subagentDefinition struct {
	ID     string   `yaml:"id"`
	Skills []string `yaml:"skills"`
}

func (r Root) validateSubagentsBundle() []Diagnostic {
	subagentsPath, err := r.Resolve("skills/subagents.yaml")
	if err != nil {
		return []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.skills.subagents.invalid_path",
			Message:    err.Error(),
			Suggestion: "Ensure skills/subagents.yaml stays within workspace root",
		}}
	}
	content, err := os.ReadFile(subagentsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Diagnostic{{
				Level:      DiagnosticWarning,
				Code:       "workspace.skills.subagents.missing",
				Path:       subagentsPath,
				Message:    "skills/subagents.yaml is missing",
				Suggestion: "Run init.step0.constitution to bootstrap baseline bundle",
			}}
		}
		return []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.skills.subagents.unreadable",
			Path:       subagentsPath,
			Message:    fmt.Sprintf("cannot read skills/subagents.yaml: %v", err),
			Suggestion: "Fix filesystem permissions for workspace skills directory",
		}}
	}

	var bundle subagentsBundle
	if err := yaml.Unmarshal(content, &bundle); err != nil {
		return []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.skills.subagents.invalid_yaml",
			Path:       subagentsPath,
			Message:    fmt.Sprintf("cannot parse skills/subagents.yaml: %v", err),
			Suggestion: "Fix YAML syntax in skills/subagents.yaml",
		}}
	}

	diagnostics := []Diagnostic{}
	for _, agent := range bundle.Agents {
		agentID := strings.TrimSpace(agent.ID)
		if agentID == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Level:      DiagnosticError,
				Code:       "workspace.skills.subagents.agent_id_missing",
				Path:       subagentsPath,
				Message:    "subagents bundle contains agent entry with empty id",
				Suggestion: "Set a non-empty agent id in skills/subagents.yaml",
			})
			continue
		}
		for _, skill := range agent.Skills {
			skill = strings.TrimSpace(skill)
			if skill == "" {
				continue
			}
			skillPath, resolveErr := r.Resolve(filepath.Join("skills", skill))
			if resolveErr != nil {
				diagnostics = append(diagnostics, Diagnostic{
					Level:      DiagnosticError,
					Code:       "workspace.skills.subagents.skill_invalid_path",
					Repo:       agentID,
					Message:    resolveErr.Error(),
					Suggestion: "Ensure skill references stay inside workspace skills directory",
				})
				continue
			}
			if info, statErr := os.Stat(skillPath); statErr != nil || !info.IsDir() {
				diagnostics = append(diagnostics, Diagnostic{
					Level:      DiagnosticError,
					Code:       "workspace.skills.subagents.skill_missing",
					Repo:       agentID,
					Path:       skillPath,
					Message:    fmt.Sprintf("referenced skill %q for agent %q does not exist", skill, agentID),
					Suggestion: "Create skill package directory or remove broken reference from subagents.yaml",
				})
			}
		}
	}

	return diagnostics
}
