package doctor

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestRunFakeRuntimePassesWithoutProviderCommand(t *testing.T) {
	t.Parallel()

	report, err := Run(context.Background(), Options{
		WorkspacePath:       t.TempDir(),
		RuntimeMode:         acpruntime.RuntimeModeFake,
		EmbeddedUIAvailable: true,
		LookPath: func(name string) (string, error) {
			if name == "git" {
				return "/usr/bin/git", nil
			}
			return "", errors.New("unexpected lookup")
		},
	})
	if err != nil {
		t.Fatalf("doctor run: %v", err)
	}
	if !report.OK {
		t.Fatalf("expected report OK, got %+v", report)
	}
	check := findCheck(t, report, "runtime_provider")
	if check.Status != StatusPass || check.Message == "" {
		t.Fatalf("expected fake runtime provider pass, got %+v", check)
	}
	if repoCheck := findCheck(t, report, "repo_source"); repoCheck.Status != StatusWarn {
		t.Fatalf("missing repo source should be warning-only, got %+v", repoCheck)
	}
}

func TestRunHeadlessMissingProviderFails(t *testing.T) {
	t.Parallel()

	report, err := Run(context.Background(), Options{
		WorkspacePath:       t.TempDir(),
		RuntimeMode:         acpruntime.RuntimeModeHeadless,
		RuntimeProvider:     string(acpruntime.ProviderCodexCode),
		EmbeddedUIAvailable: true,
		LookPath: func(name string) (string, error) {
			if name == "git" {
				return "/usr/bin/git", nil
			}
			return "", errors.New("missing")
		},
	})
	if err != nil {
		t.Fatalf("doctor run: %v", err)
	}
	if report.OK {
		t.Fatalf("expected report to fail")
	}
	check := findCheck(t, report, "runtime_provider")
	if check.Status != StatusFail {
		t.Fatalf("expected runtime provider failure, got %+v", check)
	}
}

func TestRunChecksGitURLWithLsRemote(t *testing.T) {
	t.Parallel()

	var commands []GitCommand
	report, err := Run(context.Background(), Options{
		WorkspacePath:       t.TempDir(),
		RepoGitURL:          "https://github.com/org/repo.git",
		RuntimeMode:         acpruntime.RuntimeModeFake,
		EmbeddedUIAvailable: true,
		LookPath: func(name string) (string, error) {
			if name == "git" {
				return "/usr/bin/git", nil
			}
			return "", errors.New("missing")
		},
		RunGit: func(_ context.Context, command GitCommand) (GitCommandResult, error) {
			commands = append(commands, command)
			return GitCommandResult{Output: "refs/heads/main"}, nil
		},
	})
	if err != nil {
		t.Fatalf("doctor run: %v", err)
	}
	if !report.OK {
		t.Fatalf("expected report OK, got %+v", report)
	}
	if len(commands) != 1 {
		t.Fatalf("expected one git command, got %d", len(commands))
	}
	wantArgs := []string{"ls-remote", "--heads", "https://github.com/org/repo.git"}
	if !reflect.DeepEqual(commands[0].Args, wantArgs) {
		t.Fatalf("expected args %#v, got %#v", wantArgs, commands[0].Args)
	}
}

func TestRunLocalRepoPathRequiresGitCheckout(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	report, err := Run(context.Background(), Options{
		WorkspacePath:       t.TempDir(),
		RepoPath:            repoDir,
		RuntimeMode:         acpruntime.RuntimeModeFake,
		EmbeddedUIAvailable: true,
		LookPath: func(name string) (string, error) {
			if name == "git" {
				return "/usr/bin/git", nil
			}
			return "", errors.New("missing")
		},
		RunGit: func(_ context.Context, command GitCommand) (GitCommandResult, error) {
			if command.Dir != repoDir {
				t.Fatalf("expected git dir %q, got %q", repoDir, command.Dir)
			}
			return GitCommandResult{}, errors.New("not a git checkout")
		},
	})
	if err != nil {
		t.Fatalf("doctor run: %v", err)
	}
	if report.OK {
		t.Fatalf("expected report to fail")
	}
	check := findCheck(t, report, "repo_source")
	if check.Status != StatusFail {
		t.Fatalf("expected repo source failure, got %+v", check)
	}
}

func TestRunRejectsAmbiguousRepoSources(t *testing.T) {
	t.Parallel()

	_, err := Run(context.Background(), Options{
		WorkspacePath:       t.TempDir(),
		RepoPath:            filepath.Join(t.TempDir(), "repo"),
		RepoGitURL:          "https://github.com/org/repo.git",
		RuntimeMode:         acpruntime.RuntimeModeFake,
		EmbeddedUIAvailable: true,
	})
	if err == nil {
		t.Fatalf("expected ambiguous repo source error")
	}
}

func findCheck(t *testing.T, report Report, id string) Check {
	t.Helper()
	for _, check := range report.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("missing check %q in %+v", id, report.Checks)
	return Check{}
}
