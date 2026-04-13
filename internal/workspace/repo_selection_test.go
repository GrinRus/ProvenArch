package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEvaluateRepoSelectionAllIncludesEveryRepo(t *testing.T) {
	t.Parallel()

	repos := []RepoSource{
		{Name: "payments", Analysis: &RepoAnalysisConfig{Role: RepoRoleBackend}},
		{Name: "web", Analysis: &RepoAnalysisConfig{Role: RepoRoleFrontend}},
		{Name: "gateway", Analysis: &RepoAnalysisConfig{Role: RepoRoleMixed}},
	}

	selectedScopes, decisions, warnings := EvaluateRepoSelection(repos, RepoSelectionAll)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings in all mode, got %+v", warnings)
	}
	if len(selectedScopes) != 3 {
		t.Fatalf("expected all repo scopes selected, got %+v", selectedScopes)
	}
	if len(decisions) != 3 {
		t.Fatalf("expected three decisions, got %d", len(decisions))
	}
	for _, decision := range decisions {
		if !decision.Included {
			t.Fatalf("expected repo %q to be included in all mode", decision.Name)
		}
	}
}

func TestEvaluateRepoSelectionBackendOnlyExcludesFrontendAndWarnsUnknown(t *testing.T) {
	t.Parallel()

	repos := []RepoSource{
		{Name: "payments", Analysis: &RepoAnalysisConfig{Role: RepoRoleBackend}},
		{Name: "web", Analysis: &RepoAnalysisConfig{Role: RepoRoleFrontend}},
		{Name: "unknown-role"},
	}

	selectedScopes, decisions, warnings := EvaluateRepoSelection(repos, RepoSelectionBackendOnly)
	if len(selectedScopes) != 2 {
		t.Fatalf("expected backend_only to keep backend+unknown repos, got %+v", selectedScopes)
	}
	if len(warnings) != 1 || warnings[0].Code != "workspace.repo.selection.role_unknown" {
		t.Fatalf("expected unknown-role warning, got %+v", warnings)
	}

	decisionsByRepo := map[string]RepoSelectionDecision{}
	for _, decision := range decisions {
		decisionsByRepo[decision.Name] = decision
	}
	if decisionsByRepo["payments"].Included != true {
		t.Fatalf("expected backend repo to stay included, got %+v", decisionsByRepo["payments"])
	}
	if decisionsByRepo["web"].Included != false {
		t.Fatalf("expected frontend repo to be excluded, got %+v", decisionsByRepo["web"])
	}
	if decisionsByRepo["unknown-role"].EffectiveRole != RepoRoleUnknown || decisionsByRepo["unknown-role"].Included != true {
		t.Fatalf("expected unknown role to be included with unknown effective role, got %+v", decisionsByRepo["unknown-role"])
	}
}

func TestValidateAnnotatesResolvedReposWithSelectionDecision(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for resolver validation")
	}

	root := t.TempDir()
	backendPath := filepath.Join(root, "repos", "payments")
	frontendPath := filepath.Join(root, "repos", "web")
	if err := os.MkdirAll(backendPath, 0o755); err != nil {
		t.Fatalf("create backend path: %v", err)
	}
	if err := os.MkdirAll(frontendPath, 0o755); err != nil {
		t.Fatalf("create frontend path: %v", err)
	}
	writeManifestFile(t, root, `
version: 1
repos:
  - name: payments
    path: `+backendPath+`
    analysis:
      role: backend
  - name: web
    path: `+frontendPath+`
    analysis:
      role: frontend
`)

	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	report := ws.Validate(context.Background(), ValidateOptions{
		ResolveRepos:      true,
		FetchGit:          false,
		VerifyRefs:        false,
		RepoSelectionMode: RepoSelectionBackendOnly,
	})
	if !report.OK {
		t.Fatalf("expected validation to succeed, got errors %+v", report.Errors)
	}
	if len(report.SelectedRepoScopes) != 1 || report.SelectedRepoScopes[0] != "payments" {
		t.Fatalf("expected only backend repo selected, got %+v", report.SelectedRepoScopes)
	}
	if len(report.ResolvedRepos) != 2 {
		t.Fatalf("expected two resolved repos, got %d", len(report.ResolvedRepos))
	}

	resolvedByName := map[string]ResolvedRepo{}
	for _, resolved := range report.ResolvedRepos {
		resolvedByName[resolved.Name] = resolved
	}
	if resolvedByName["payments"].EffectiveRole != RepoRoleBackend {
		t.Fatalf("expected backend effective role, got %+v", resolvedByName["payments"])
	}
	if resolvedByName["web"].EffectiveRole != RepoRoleFrontend {
		t.Fatalf("expected frontend effective role, got %+v", resolvedByName["web"])
	}
	if resolvedByName["payments"].Included == nil || *resolvedByName["payments"].Included != true {
		t.Fatalf("expected payments included=true, got %+v", resolvedByName["payments"])
	}
	if resolvedByName["web"].Included == nil || *resolvedByName["web"].Included != false {
		t.Fatalf("expected web included=false, got %+v", resolvedByName["web"])
	}
}
