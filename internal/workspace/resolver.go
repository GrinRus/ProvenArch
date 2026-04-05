package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ResolvedRepo struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Path   string `json:"path"`
	Ref    string `json:"ref,omitempty"`
}

type ResolveOptions struct {
	FetchGit   bool
	VerifyRefs bool
	Git        GitExecutor
}

type GitExecutor interface {
	EnsureAvailable() error
	Run(ctx context.Context, dir string, args ...string) (string, error)
}

type localGitExecutor struct{}

func (localGitExecutor) EnsureAvailable() error {
	_, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git binary is unavailable: %w", err)
	}
	return nil
}

func (localGitExecutor) Run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func (r Root) ResolveRepoSources(ctx context.Context, options ResolveOptions) ([]ResolvedRepo, []Diagnostic) {
	gitExec := options.Git
	if gitExec == nil {
		gitExec = localGitExecutor{}
	}

	resolved := make([]ResolvedRepo, 0, len(r.Manifest.Repos))
	diagnostics := []Diagnostic{}

	gitAvailable := true
	if err := gitExec.EnsureAvailable(); err != nil {
		gitAvailable = false
		diagnostics = append(diagnostics, Diagnostic{
			Level:      DiagnosticError,
			Code:       "workspace.git.unavailable",
			Message:    err.Error(),
			Suggestion: "Install git and ensure it is available in PATH",
		})
	}

	for _, repo := range r.Manifest.Repos {
		name := strings.TrimSpace(repo.Name)
		if strings.TrimSpace(repo.Path) != "" {
			entry, diag := r.resolvePathRepo(ctx, gitExec, repo, options, gitAvailable)
			if diag != nil {
				diagnostics = append(diagnostics, *diag)
				continue
			}
			resolved = append(resolved, entry)
			continue
		}
		if strings.TrimSpace(repo.GitURL) != "" {
			entry, repoDiagnostics := r.resolveGitURLRepo(ctx, gitExec, repo, options, gitAvailable)
			if entry.Name != "" {
				resolved = append(resolved, entry)
			}
			diagnostics = append(diagnostics, repoDiagnostics...)
			continue
		}

		diagnostics = append(diagnostics, Diagnostic{
			Level:      DiagnosticError,
			Code:       "workspace.repo.source.invalid",
			Repo:       name,
			Message:    "repo source is invalid: expected path or git_url",
			Suggestion: "Set exactly one of path or git_url in workspace.yaml",
		})
	}

	return resolved, diagnostics
}

func (r Root) resolvePathRepo(ctx context.Context, gitExec GitExecutor, repo RepoSource, options ResolveOptions, gitAvailable bool) (ResolvedRepo, *Diagnostic) {
	repoPath := strings.TrimSpace(repo.Path)
	if repoPath == "" {
		return ResolvedRepo{}, &Diagnostic{
			Level:      DiagnosticError,
			Code:       "workspace.repo.path.empty",
			Repo:       repo.Name,
			Message:    "repo path is empty",
			Suggestion: "Set a valid path for the repository",
		}
	}
	if !filepath.IsAbs(repoPath) {
		repoPath = filepath.Join(r.Path, repoPath)
	}
	repoPath = filepath.Clean(repoPath)
	info, err := os.Stat(repoPath)
	if err != nil {
		return ResolvedRepo{}, &Diagnostic{
			Level:      DiagnosticError,
			Code:       "workspace.repo.path.unreachable",
			Repo:       repo.Name,
			Path:       repoPath,
			Message:    fmt.Sprintf("repo path is not accessible: %v", err),
			Suggestion: "Ensure path exists and is readable by ACP",
		}
	}
	if !info.IsDir() {
		return ResolvedRepo{}, &Diagnostic{
			Level:      DiagnosticError,
			Code:       "workspace.repo.path.not_dir",
			Repo:       repo.Name,
			Path:       repoPath,
			Message:    "repo path must point to a directory",
			Suggestion: "Fix repo path in workspace.yaml",
		}
	}

	if options.VerifyRefs && strings.TrimSpace(repo.Ref) != "" {
		if !gitAvailable {
			return ResolvedRepo{}, &Diagnostic{
				Level:      DiagnosticError,
				Code:       "workspace.repo.ref.verify.git_required",
				Repo:       repo.Name,
				Path:       repoPath,
				Message:    "cannot verify repo ref because git is unavailable",
				Suggestion: "Install git or remove ref from workspace manifest",
			}
		}
		if _, err := gitExec.Run(ctx, repoPath, "rev-parse", "--verify", strings.TrimSpace(repo.Ref)+"^{commit}"); err != nil {
			return ResolvedRepo{}, &Diagnostic{
				Level:      DiagnosticError,
				Code:       "workspace.repo.ref.invalid",
				Repo:       repo.Name,
				Path:       repoPath,
				Message:    fmt.Sprintf("cannot resolve ref %q: %v", repo.Ref, err),
				Suggestion: "Use an existing branch/tag/commit in workspace.yaml",
			}
		}
	}

	return ResolvedRepo{Name: repo.Name, Source: "path", Path: repoPath, Ref: repo.Ref}, nil
}

func (r Root) resolveGitURLRepo(ctx context.Context, gitExec GitExecutor, repo RepoSource, options ResolveOptions, gitAvailable bool) (ResolvedRepo, []Diagnostic) {
	if !gitAvailable {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.git_url.git_required",
			Repo:       repo.Name,
			Message:    "cannot resolve git_url source because git is unavailable",
			Suggestion: "Install git and ensure repository access is configured on this machine",
		}}
	}

	cacheDir, err := r.resolveRepoCacheDir(repo.Name)
	if err != nil {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.cache.invalid",
			Repo:       repo.Name,
			Message:    err.Error(),
			Suggestion: "Ensure workspace path is valid and writable",
		}}
	}

	if !options.FetchGit {
		return ResolvedRepo{Name: repo.Name, Source: "git_url", Path: cacheDir, Ref: repo.Ref}, []Diagnostic{{
			Level:      DiagnosticWarning,
			Code:       "workspace.repo.git_url.dry_unresolved",
			Repo:       repo.Name,
			Path:       cacheDir,
			Message:    "git_url source was not fetched in dry validation mode",
			Suggestion: "Run pipeline execution to materialize and verify git_url sources",
		}}
	}

	if err := os.MkdirAll(filepath.Dir(cacheDir), 0o755); err != nil {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.cache.mkdir_failed",
			Repo:       repo.Name,
			Path:       cacheDir,
			Message:    fmt.Sprintf("cannot create repo cache directory: %v", err),
			Suggestion: "Fix filesystem permissions for workspace cache",
		}}
	}

	repoExists := false
	if info, statErr := os.Stat(filepath.Join(cacheDir, ".git")); statErr == nil && info.IsDir() {
		repoExists = true
	}
	if repoExists {
		if _, err := gitExec.Run(ctx, cacheDir, "remote", "set-url", "origin", strings.TrimSpace(repo.GitURL)); err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.remote_failed",
				Repo:       repo.Name,
				Path:       cacheDir,
				Message:    err.Error(),
				Suggestion: "Verify git_url and repository permissions",
			}}
		}
		if _, err := gitExec.Run(ctx, cacheDir, "fetch", "--prune", "origin"); err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.fetch_failed",
				Repo:       repo.Name,
				Path:       cacheDir,
				Message:    err.Error(),
				Suggestion: "Check network access and git auth for this source",
			}}
		}
	} else {
		if _, err := gitExec.Run(ctx, "", "clone", strings.TrimSpace(repo.GitURL), cacheDir); err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.clone_failed",
				Repo:       repo.Name,
				Path:       cacheDir,
				Message:    err.Error(),
				Suggestion: "Check git_url and git auth context for this runner",
			}}
		}
	}

	if options.VerifyRefs && strings.TrimSpace(repo.Ref) != "" {
		if _, err := gitExec.Run(ctx, cacheDir, "rev-parse", "--verify", strings.TrimSpace(repo.Ref)+"^{commit}"); err != nil {
			if _, checkoutErr := gitExec.Run(ctx, cacheDir, "checkout", "--force", strings.TrimSpace(repo.Ref)); checkoutErr != nil {
				return ResolvedRepo{}, []Diagnostic{{
					Level:      DiagnosticError,
					Code:       "workspace.repo.git_url.ref_invalid",
					Repo:       repo.Name,
					Path:       cacheDir,
					Message:    fmt.Sprintf("cannot checkout ref %q: %v", repo.Ref, checkoutErr),
					Suggestion: "Use an existing branch/tag/commit in workspace.yaml",
				}}
			}
		}
	}

	if strings.TrimSpace(repo.Ref) != "" {
		if _, err := gitExec.Run(ctx, cacheDir, "checkout", "--force", strings.TrimSpace(repo.Ref)); err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.checkout_failed",
				Repo:       repo.Name,
				Path:       cacheDir,
				Message:    err.Error(),
				Suggestion: "Ensure the requested ref exists and can be checked out",
			}}
		}
	}

	if _, err := gitExec.Run(ctx, cacheDir, "rev-parse", "--verify", "HEAD"); err != nil {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.git_url.invalid_head",
			Repo:       repo.Name,
			Path:       cacheDir,
			Message:    err.Error(),
			Suggestion: "Ensure repository is cloned and has a valid HEAD",
		}}
	}

	return ResolvedRepo{Name: repo.Name, Source: "git_url", Path: cacheDir, Ref: repo.Ref}, nil
}

func (r Root) resolveRepoCacheDir(repoName string) (string, error) {
	slug := repoCacheSlug(repoName)
	relPath := filepath.Join(".acp", "repos", slug)
	absPath, err := r.Resolve(relPath)
	if err != nil {
		return "", err
	}
	return absPath, nil
}

func repoCacheSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "unknown"
	}
	var out []rune
	prevDash := false
	for _, r := range name {
		isAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isAlpha || isDigit {
			out = append(out, r)
			prevDash = false
			continue
		}
		if prevDash {
			continue
		}
		out = append(out, '-')
		prevDash = true
	}
	slug := strings.Trim(string(out), "-")
	if slug == "" {
		return "unknown"
	}
	return slug
}
