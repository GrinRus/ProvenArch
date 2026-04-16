package workspace

import (
	"fmt"
	"sort"
	"strings"
)

type RepoSelectionDecision struct {
	Name          string `json:"name"`
	DeclaredRole  string `json:"declared_role,omitempty"`
	EffectiveRole string `json:"effective_role"`
	Included      bool   `json:"included"`
	Reason        string `json:"reason"`
}

func NormalizeRepoRole(value string) (string, bool) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case RepoRoleBackend, RepoRoleFrontend, RepoRoleMixed, RepoRoleUnknown:
		return normalized, true
	default:
		return "", false
	}
}

func CanonicalRepoRole(value string) string {
	if normalized, ok := NormalizeRepoRole(value); ok {
		return normalized
	}
	return RepoRoleUnknown
}

func NormalizeRepoSelectionMode(value string) (string, bool) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case RepoSelectionAll, RepoSelectionBackendOnly:
		return normalized, true
	default:
		return "", false
	}
}

func CanonicalRepoSelectionMode(value string) string {
	if normalized, ok := NormalizeRepoSelectionMode(value); ok {
		return normalized
	}
	return RepoSelectionAll
}

func EvaluateRepoSelection(repos []RepoSource, mode string) ([]string, []RepoSelectionDecision, []Diagnostic) {
	selectionMode := CanonicalRepoSelectionMode(mode)
	selectedSet := map[string]struct{}{}
	selectedScopes := make([]string, 0, len(repos))
	decisions := make([]RepoSelectionDecision, 0, len(repos))
	warnings := []Diagnostic{}

	for _, repo := range repos {
		name := strings.TrimSpace(repo.Name)
		roleResolution := ResolveRepoRole(repo)
		declaredRole := roleResolution.DeclaredRole
		effectiveRole := roleResolution.EffectiveRole
		roleSource := roleResolution.Source

		included := true
		reason := fmt.Sprintf("included by repo_selection=%s", selectionMode)
		if selectionMode == RepoSelectionBackendOnly {
			switch effectiveRole {
			case RepoRoleFrontend:
				included = false
				if roleSource == "inferred" {
					reason = "excluded by repo_selection=backend_only (effective_role=frontend; inferred from repo source)"
				} else {
					reason = "excluded by repo_selection=backend_only (effective_role=frontend)"
				}
			case RepoRoleBackend, RepoRoleMixed:
				reason = fmt.Sprintf("included by repo_selection=backend_only (effective_role=%s)", effectiveRole)
				if roleSource == "inferred" {
					reason = fmt.Sprintf("%s; inferred from repo source", strings.TrimSuffix(reason, ")")) + ")"
				}
			default:
				if declaredRole == RepoRoleUnknown {
					reason = "included by repo_selection=backend_only (effective_role=unknown; declared analysis.role=unknown)"
				} else {
					reason = "included by repo_selection=backend_only (effective_role=unknown; no declared or inferred role)"
				}
				warningMessage := fmt.Sprintf("repo %q has no declared analysis.role and ACP could not infer frontend/backend role from repo source; backend_only keeps it included", name)
				if declaredRole == RepoRoleUnknown {
					warningMessage = fmt.Sprintf("repo %q declares analysis.role=unknown and remains included by backend_only policy", name)
				}
				warnings = append(warnings, Diagnostic{
					Level:   DiagnosticWarning,
					Code:    "workspace.repo.selection.role_unknown",
					Repo:    name,
					Message: warningMessage,
				})
			}
		}

		if included && name != "" {
			if _, exists := selectedSet[name]; !exists {
				selectedSet[name] = struct{}{}
				selectedScopes = append(selectedScopes, name)
			}
		}
		decisions = append(decisions, RepoSelectionDecision{
			Name:          name,
			DeclaredRole:  declaredRole,
			EffectiveRole: effectiveRole,
			Included:      included,
			Reason:        reason,
		})
	}

	sort.Strings(selectedScopes)
	sort.Slice(decisions, func(i, j int) bool {
		return decisions[i].Name < decisions[j].Name
	})
	return selectedScopes, decisions, warnings
}
