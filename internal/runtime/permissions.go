package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const (
	PermissionModeTrustedFullAccess = "trusted_full_access"
	PermissionModeManaged           = "managed"

	PermissionApprovalFailFast = "fail_fast"
	PermissionApprovalUI       = "ui"
)

type PermissionSource string

const (
	PermissionSourceDefault   PermissionSource = "default"
	PermissionSourceWorkspace PermissionSource = "workspace"
)

type PermissionValues struct {
	Mode            string `json:"mode"`
	ApprovalChannel string `json:"approval_channel"`
}

type PermissionSources struct {
	Mode            PermissionSource `json:"mode"`
	ApprovalChannel PermissionSource `json:"approval_channel"`
}

type PermissionResolution struct {
	Persisted workspace.RuntimePermissionsConfig `json:"persisted"`
	Effective PermissionValues                   `json:"effective"`
	Source    PermissionSources                  `json:"source"`
}

type PermissionRequest struct {
	RequestID     string              `json:"request_id"`
	RunID         string              `json:"run_id"`
	StepID        string              `json:"step_id"`
	Provider      Provider            `json:"provider"`
	Action        string              `json:"action"`
	PathOrCommand string              `json:"path_or_command"`
	Reason        string              `json:"reason,omitempty"`
	Decision      *PermissionDecision `json:"decision,omitempty"`
}

type PermissionDecision struct {
	RequestID string `json:"request_id"`
	Decision  string `json:"decision"`
	RuleID    string `json:"rule_id"`
	Message   string `json:"message,omitempty"`
}

const (
	PermissionDecisionAutoApproved = "auto_approved"
	PermissionDecisionDenied       = "denied"
	PermissionDecisionNeedsUser    = "needs_user"
)

func ResolvePermissions(manifest workspace.Manifest) PermissionResolution {
	persisted := workspace.RuntimePermissionsConfig{}
	if manifest.Runtime != nil && manifest.Runtime.Profile != nil && manifest.Runtime.Profile.Permissions != nil {
		persisted = *manifest.Runtime.Profile.Permissions
	}

	effective := DefaultPermissions()
	source := PermissionSources{
		Mode:            PermissionSourceDefault,
		ApprovalChannel: PermissionSourceDefault,
	}
	if mode := normalizePermissionMode(persisted.Mode); mode != "" {
		effective.Mode = mode
		source.Mode = PermissionSourceWorkspace
	}
	if channel := normalizePermissionApprovalChannel(persisted.ApprovalChannel); channel != "" {
		effective.ApprovalChannel = channel
		source.ApprovalChannel = PermissionSourceWorkspace
	}
	return PermissionResolution{
		Persisted: persisted,
		Effective: effective,
		Source:    source,
	}
}

func DefaultPermissions() PermissionValues {
	return PermissionValues{
		Mode:            PermissionModeTrustedFullAccess,
		ApprovalChannel: PermissionApprovalFailFast,
	}
}

func normalizePermissionMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case PermissionModeTrustedFullAccess:
		return PermissionModeTrustedFullAccess
	case PermissionModeManaged:
		return PermissionModeManaged
	default:
		return ""
	}
}

func normalizePermissionApprovalChannel(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case PermissionApprovalFailFast:
		return PermissionApprovalFailFast
	case PermissionApprovalUI:
		return PermissionApprovalUI
	default:
		return ""
	}
}

func (decision PermissionDecision) Approved() bool {
	return strings.TrimSpace(decision.Decision) == PermissionDecisionAutoApproved
}

func DecideRuntimePermission(task Task, request PermissionRequest) PermissionDecision {
	requestID := strings.TrimSpace(request.RequestID)
	action := strings.TrimSpace(strings.ToLower(request.Action))
	target := strings.TrimSpace(request.PathOrCommand)
	base := ResolveHeadlessWorkingDirectory(task)
	decision := func(kind string, ruleID string, message string) PermissionDecision {
		return PermissionDecision{
			RequestID: requestID,
			Decision:  kind,
			RuleID:    ruleID,
			Message:   strings.TrimSpace(message),
		}
	}

	switch action {
	case "read", "list", "glob", "grep":
		path, err := policyPath(target, base)
		if err != nil {
			return decision(PermissionDecisionDenied, "deny_invalid_path", err.Error())
		}
		if containsPathTraversal(target) {
			return decision(PermissionDecisionDenied, "deny_path_traversal", "path traversal is not auto-approved")
		}
		if pathWithinAny(path, readRootsForPermissionPolicy(task)) {
			return decision(PermissionDecisionAutoApproved, "auto_read_context_root", "read is inside runtime read context")
		}
		return decision(PermissionDecisionDenied, "deny_outside_allowed_roots", "read target is outside runtime read context")
	case "create", "write", "overwrite", "mkdir":
		path, err := policyPath(target, base)
		if err != nil {
			return decision(PermissionDecisionDenied, "deny_invalid_path", err.Error())
		}
		if containsPathTraversal(target) {
			return decision(PermissionDecisionDenied, "deny_path_traversal", "path traversal is not auto-approved")
		}
		if pathWithinAny(path, sourceRepoRootsForPermissionPolicy(task)) {
			return decision(PermissionDecisionDenied, "deny_source_repo_write", "runtime writes to analyzed repositories are forbidden")
		}
		if isProtectedWorkspacePath(task.Workspace, path) {
			return decision(PermissionDecisionDenied, "deny_protected_workspace_write", "runtime writes to protected workspace paths are forbidden")
		}
		if pathWithinAny(path, writeRootsForPermissionPolicy(task)) {
			return decision(PermissionDecisionAutoApproved, "auto_write_runtime_staging_root", "write is inside runtime staging root")
		}
		return decision(PermissionDecisionDenied, "deny_outside_allowed_roots", "write target is outside runtime staging roots")
	case "network", "http", "fetch", "install", "package_install", "shell", "command", "exec":
		return decision(PermissionDecisionNeedsUser, "ask_unsafe_operation", "operation requires explicit user approval")
	default:
		return decision(PermissionDecisionNeedsUser, "ask_unknown_action", fmt.Sprintf("unknown permission action %q", action))
	}
}

func readRootsForPermissionPolicy(task Task) []string {
	roots := append([]string(nil), task.ReadContextRoots...)
	roots = append(roots, task.WriteRoot, task.DraftFinalRoot)
	return normalizePermissionRoots(roots)
}

func writeRootsForPermissionPolicy(task Task) []string {
	return normalizePermissionRoots([]string{task.WriteRoot, task.DraftFinalRoot})
}

func sourceRepoRootsForPermissionPolicy(task Task) []string {
	workspaceRoot, _ := canonicalPolicyPath(task.Workspace)
	out := []string{}
	for _, root := range task.ReadContextRoots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		canonical, err := canonicalPolicyPath(trimmed)
		if err != nil || canonical == "" {
			continue
		}
		if canonical == workspaceRoot || isRuntimeTaskrunRootForPermissionPolicy(canonical) {
			continue
		}
		out = append(out, canonical)
	}
	return normalizePermissionRoots(out)
}

func isRuntimeTaskrunRootForPermissionPolicy(path string) bool {
	slash := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	return strings.Contains(slash, "/reports/taskruns/")
}

func normalizePermissionRoots(roots []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, root := range roots {
		canonical, err := canonicalPolicyPath(strings.TrimSpace(root))
		if err != nil || canonical == "" {
			continue
		}
		if _, exists := seen[canonical]; exists {
			continue
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	return out
}

func policyPath(raw string, base string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("path is required")
	}
	path := strings.TrimSpace(raw)
	if !filepath.IsAbs(path) {
		if strings.TrimSpace(base) == "" {
			return "", fmt.Errorf("relative path %q has no runtime working directory", raw)
		}
		path = filepath.Join(base, path)
	}
	return canonicalPolicyPath(path)
}

func canonicalPolicyPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved), nil
	}

	parts := []string{}
	parent := abs
	for {
		if resolved, err := filepath.EvalSymlinks(parent); err == nil {
			for i := len(parts) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, parts[i])
			}
			return filepath.Clean(resolved), nil
		}
		next := filepath.Dir(parent)
		if next == parent {
			break
		}
		parts = append(parts, filepath.Base(parent))
		parent = next
	}
	return filepath.Clean(abs), nil
}

func containsPathTraversal(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(strings.TrimSpace(path)), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func pathWithinAny(path string, roots []string) bool {
	for _, root := range roots {
		if sameOrWithin(path, root) {
			return true
		}
	}
	return false
}

func sameOrWithin(path string, root string) bool {
	candidate, err := canonicalPolicyPath(path)
	if err != nil {
		return false
	}
	base, err := canonicalPolicyPath(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(base, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func isProtectedWorkspacePath(workspaceRoot string, path string) bool {
	root, err := canonicalPolicyPath(workspaceRoot)
	if err != nil || root == "" {
		return false
	}
	candidate, err := canonicalPolicyPath(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == "workspace.yaml" ||
		strings.HasPrefix(rel, "schemas/") ||
		strings.HasPrefix(rel, "docs/spec/") ||
		strings.HasPrefix(rel, "charter/")
}
