package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestResolvePermissionsDefaultsToTrustedFullAccess(t *testing.T) {
	t.Parallel()

	resolved := ResolvePermissions(workspace.Manifest{})
	if resolved.Effective.Mode != PermissionModeTrustedFullAccess {
		t.Fatalf("expected trusted default, got %q", resolved.Effective.Mode)
	}
	if resolved.Effective.ApprovalChannel != PermissionApprovalFailFast {
		t.Fatalf("expected fail_fast default, got %q", resolved.Effective.ApprovalChannel)
	}
	if resolved.Source.Mode != PermissionSourceDefault {
		t.Fatalf("expected default source, got %q", resolved.Source.Mode)
	}
}

func TestResolvePermissionsUsesWorkspaceValues(t *testing.T) {
	t.Parallel()

	resolved := ResolvePermissions(workspace.Manifest{
		Runtime: &workspace.RuntimeConfig{
			Profile: &workspace.RuntimeProfileConfig{
				Permissions: &workspace.RuntimePermissionsConfig{
					Mode:            PermissionModeManaged,
					ApprovalChannel: PermissionApprovalUI,
				},
			},
		},
	})
	if resolved.Effective.Mode != PermissionModeManaged || resolved.Effective.ApprovalChannel != PermissionApprovalUI {
		t.Fatalf("unexpected effective permissions: %+v", resolved.Effective)
	}
	if resolved.Source.Mode != PermissionSourceWorkspace || resolved.Source.ApprovalChannel != PermissionSourceWorkspace {
		t.Fatalf("unexpected sources: %+v", resolved.Source)
	}
}

func TestDecideRuntimePermissionAutoApprovesEnvelopeReadsAndWrites(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "arch")
	repoRoot := filepath.Join(root, "repo")
	writeRoot := filepath.Join(workspaceRoot, "reports", "taskruns", "run-1", "write")
	draftRoot := filepath.Join(workspaceRoot, "reports", "taskruns", "run-1", "final")
	for _, dir := range []string{workspaceRoot, repoRoot, writeRoot, draftRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	task := Task{
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        workspaceRoot,
		WriteRoot:        writeRoot,
		DraftFinalRoot:   draftRoot,
		ReadContextRoots: []string{workspaceRoot, repoRoot},
	}

	readDecision := DecideRuntimePermission(task, PermissionRequest{
		RequestID:     "read",
		Action:        "read",
		PathOrCommand: filepath.Join(repoRoot, "README.md"),
	})
	if readDecision.Decision != PermissionDecisionAutoApproved || readDecision.RuleID != "auto_read_context_root" {
		t.Fatalf("expected auto-approved read, got %+v", readDecision)
	}

	writeDecision := DecideRuntimePermission(task, PermissionRequest{
		RequestID:     "write",
		Action:        "write",
		PathOrCommand: filepath.Join(writeRoot, "shard-pack-manifest.json"),
	})
	if writeDecision.Decision != PermissionDecisionAutoApproved || writeDecision.RuleID != "auto_write_runtime_staging_root" {
		t.Fatalf("expected auto-approved write, got %+v", writeDecision)
	}
}

func TestDecideRuntimePermissionDeniesUnsafeWritesAndEscapes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "arch")
	repoRoot := filepath.Join(workspaceRoot, "repos", "payments-service")
	writeRoot := filepath.Join(workspaceRoot, "reports", "taskruns", "run-1", "write")
	for _, dir := range []string{workspaceRoot, repoRoot, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	task := Task{
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        workspaceRoot,
		WriteRoot:        writeRoot,
		ReadContextRoots: []string{workspaceRoot, repoRoot},
	}

	sourceDecision := DecideRuntimePermission(task, PermissionRequest{
		Action:        "write",
		PathOrCommand: filepath.Join(repoRoot, "main.go"),
	})
	if sourceDecision.Decision != PermissionDecisionDenied || sourceDecision.RuleID != "deny_source_repo_write" {
		t.Fatalf("expected source repo write denial, got %+v", sourceDecision)
	}

	protectedDecision := DecideRuntimePermission(task, PermissionRequest{
		Action:        "write",
		PathOrCommand: filepath.Join(workspaceRoot, "workspace.yaml"),
	})
	if protectedDecision.Decision != PermissionDecisionDenied || protectedDecision.RuleID != "deny_protected_workspace_write" {
		t.Fatalf("expected protected workspace denial, got %+v", protectedDecision)
	}

	traversalDecision := DecideRuntimePermission(task, PermissionRequest{
		Action:        "read",
		PathOrCommand: "../outside",
	})
	if traversalDecision.Decision != PermissionDecisionDenied || traversalDecision.RuleID != "deny_path_traversal" {
		t.Fatalf("expected traversal denial, got %+v", traversalDecision)
	}
}

func TestDecideRuntimePermissionDenyRulesWinOverBroadWriteRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "arch")
	repoRoot := filepath.Join(workspaceRoot, "repos", "payments-service")
	for _, dir := range []string{workspaceRoot, repoRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	task := Task{
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        workspaceRoot,
		WriteRoot:        workspaceRoot,
		ReadContextRoots: []string{workspaceRoot, repoRoot},
	}

	for _, tc := range []struct {
		name   string
		path   string
		ruleID string
	}{
		{
			name:   "protected workspace",
			path:   filepath.Join(workspaceRoot, "workspace.yaml"),
			ruleID: "deny_protected_workspace_write",
		},
		{
			name:   "source repo",
			path:   filepath.Join(repoRoot, "main.go"),
			ruleID: "deny_source_repo_write",
		},
	} {
		decision := DecideRuntimePermission(task, PermissionRequest{
			Action:        "write",
			PathOrCommand: tc.path,
		})
		if decision.Decision != PermissionDecisionDenied || decision.RuleID != tc.ruleID {
			t.Fatalf("%s: expected %s, got %+v", tc.name, tc.ruleID, decision)
		}
	}
}

func TestDecideRuntimePermissionDeniesSymlinkEscapes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "arch")
	writeRoot := filepath.Join(workspaceRoot, "reports", "taskruns", "run-1", "write")
	outsideRoot := filepath.Join(root, "outside")
	for _, dir := range []string{workspaceRoot, writeRoot, outsideRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	linkPath := filepath.Join(writeRoot, "escaped")
	if err := os.Symlink(outsideRoot, linkPath); err != nil {
		t.Fatalf("symlink %s -> %s: %v", linkPath, outsideRoot, err)
	}

	task := Task{
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        workspaceRoot,
		WriteRoot:        writeRoot,
		ReadContextRoots: []string{workspaceRoot},
	}
	decision := DecideRuntimePermission(task, PermissionRequest{
		Action:        "write",
		PathOrCommand: filepath.Join(linkPath, "outside.txt"),
	})
	if decision.Decision != PermissionDecisionDenied || decision.RuleID != "deny_outside_allowed_roots" {
		t.Fatalf("expected symlink escape denial, got %+v", decision)
	}
}

func TestDecideRuntimePermissionMarksShellAndUnknownForUserApproval(t *testing.T) {
	t.Parallel()

	task := Task{WriteRoot: t.TempDir()}
	for _, request := range []PermissionRequest{
		{Action: "shell", PathOrCommand: "rm -rf /"},
		{Action: "unexpected", PathOrCommand: "anything"},
	} {
		decision := DecideRuntimePermission(task, request)
		if decision.Decision != PermissionDecisionNeedsUser {
			t.Fatalf("expected needs_user for %+v, got %+v", request, decision)
		}
	}
}
