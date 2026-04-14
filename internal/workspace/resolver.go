package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ResolvedRepo struct {
	Name            string `json:"name"`
	Source          string `json:"source"`
	Path            string `json:"path"`
	Ref             string `json:"ref,omitempty"`
	DeclaredRole    string `json:"declared_role,omitempty"`
	EffectiveRole   string `json:"effective_role,omitempty"`
	Included        *bool  `json:"included,omitempty"`
	SelectionReason string `json:"selection_reason,omitempty"`
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
			entry, repoDiagnostics := r.resolvePathRepo(ctx, gitExec, repo, options, gitAvailable)
			if entry.Name != "" {
				resolved = append(resolved, entry)
			}
			diagnostics = append(diagnostics, repoDiagnostics...)
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

func (r Root) resolvePathRepo(ctx context.Context, gitExec GitExecutor, repo RepoSource, options ResolveOptions, gitAvailable bool) (ResolvedRepo, []Diagnostic) {
	repoPath := strings.TrimSpace(repo.Path)
	if repoPath == "" {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.path.empty",
			Repo:       repo.Name,
			Message:    "repo path is empty",
			Suggestion: "Set a valid path for the repository",
		}}
	}
	if !filepath.IsAbs(repoPath) {
		repoPath = filepath.Join(r.Path, repoPath)
	}
	repoPath = filepath.Clean(repoPath)
	info, err := os.Stat(repoPath)
	if err != nil {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.path.unreachable",
			Repo:       repo.Name,
			Path:       repoPath,
			Message:    fmt.Sprintf("repo path is not accessible: %v", err),
			Suggestion: "Ensure path exists and is readable by ACP",
		}}
	}
	if !info.IsDir() {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.path.not_dir",
			Repo:       repo.Name,
			Path:       repoPath,
			Message:    "repo path must point to a directory",
			Suggestion: "Fix repo path in workspace.yaml",
		}}
	}

	repoDiagnostics := []Diagnostic{}
	if options.VerifyRefs && strings.TrimSpace(repo.Ref) != "" {
		if !gitAvailable {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.ref.verify.git_required",
				Repo:       repo.Name,
				Path:       repoPath,
				Message:    "cannot verify repo ref because git is unavailable",
				Suggestion: "Install git or remove ref from workspace manifest",
			}}
		}
		verifiedRef, warnings, err := resolvePathRepoRef(ctx, gitExec, repoPath, strings.TrimSpace(repo.Ref))
		for idx := range warnings {
			warnings[idx].Repo = repo.Name
			warnings[idx].Path = repoPath
		}
		repoDiagnostics = append(repoDiagnostics, warnings...)
		if err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.ref.invalid",
				Repo:       repo.Name,
				Path:       repoPath,
				Message:    err.Error(),
				Suggestion: "Use an existing local or origin-tracked branch/tag/commit in workspace.yaml",
			}}
		}
		headSHA, headErr := gitExec.Run(ctx, repoPath, "rev-parse", "--verify", "HEAD^{commit}")
		resolvedSHA, refErr := gitExec.Run(ctx, repoPath, "rev-parse", "--verify", verifiedRef+"^{commit}")
		if headErr == nil && refErr == nil && strings.TrimSpace(headSHA) != strings.TrimSpace(resolvedSHA) {
			repoDiagnostics = append(repoDiagnostics, Diagnostic{
				Level:      DiagnosticWarning,
				Code:       "workspace.repo.ref.head_mismatch",
				Repo:       repo.Name,
				Path:       repoPath,
				Message:    fmt.Sprintf("configured ref %q resolves to %s, but current HEAD is %s", repo.Ref, strings.TrimSpace(resolvedSHA), strings.TrimSpace(headSHA)),
				Suggestion: "Switch local checkout to the configured ref for deterministic local runs",
			})
		}
	}

	return ResolvedRepo{Name: repo.Name, Source: "path", Path: repoPath, Ref: repo.Ref}, repoDiagnostics
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

	cacheDir, legacyCacheDir, err := r.resolveRepoCacheDir(repo.Name, repo.GitURL)
	if err != nil {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.cache.invalid",
			Repo:       repo.Name,
			Message:    err.Error(),
			Suggestion: "Ensure workspace path is valid and writable",
		}}
	}
	effectiveCacheDir := cacheDir
	repoDiagnostics := []Diagnostic{}
	if fallbackPath, ok := resolveLegacyRepoCacheFallback(cacheDir, legacyCacheDir); ok {
		effectiveCacheDir = fallbackPath
		repoDiagnostics = append(repoDiagnostics, Diagnostic{
			Level:      DiagnosticWarning,
			Code:       "workspace.repo.cache.legacy_fallback",
			Repo:       repo.Name,
			Path:       fallbackPath,
			Message:    fmt.Sprintf("using legacy git_url cache path %q because new cache key path %q is not materialized", fallbackPath, cacheDir),
			Suggestion: "Run one fetch cycle to migrate this cache entry to the hashed cache key",
		})
	}

	if !options.FetchGit {
		repoDiagnostics = append(repoDiagnostics, Diagnostic{
			Level:      DiagnosticWarning,
			Code:       "workspace.repo.git_url.dry_unresolved",
			Repo:       repo.Name,
			Path:       effectiveCacheDir,
			Message:    "git_url source was not fetched in dry validation mode",
			Suggestion: "Run pipeline execution to materialize and verify git_url sources",
		})
		return ResolvedRepo{Name: repo.Name, Source: "git_url", Path: effectiveCacheDir, Ref: repo.Ref}, repoDiagnostics
	}

	if err := os.MkdirAll(filepath.Dir(effectiveCacheDir), 0o755); err != nil {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.cache.mkdir_failed",
			Repo:       repo.Name,
			Path:       effectiveCacheDir,
			Message:    fmt.Sprintf("cannot create repo cache directory: %v", err),
			Suggestion: "Fix filesystem permissions for workspace cache",
		}}
	}

	repoExists := false
	if info, statErr := os.Stat(filepath.Join(effectiveCacheDir, ".git")); statErr == nil && info.IsDir() {
		repoExists = true
	}
	if repoExists {
		if _, err := gitExec.Run(ctx, effectiveCacheDir, "remote", "set-url", "origin", strings.TrimSpace(repo.GitURL)); err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.remote_failed",
				Repo:       repo.Name,
				Path:       effectiveCacheDir,
				Message:    err.Error(),
				Suggestion: "Verify git_url and repository permissions",
			}}
		}
		if _, err := gitExec.Run(ctx, effectiveCacheDir, "fetch", "--prune", "origin"); err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.fetch_failed",
				Repo:       repo.Name,
				Path:       effectiveCacheDir,
				Message:    err.Error(),
				Suggestion: "Check network access and git auth for this source",
			}}
		}
	} else {
		if _, err := gitExec.Run(ctx, "", "clone", strings.TrimSpace(repo.GitURL), effectiveCacheDir); err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.clone_failed",
				Repo:       repo.Name,
				Path:       effectiveCacheDir,
				Message:    err.Error(),
				Suggestion: "Check git_url and git auth context for this runner",
			}}
		}
	}

	if options.VerifyRefs && strings.TrimSpace(repo.Ref) != "" {
		if _, err := gitExec.Run(ctx, effectiveCacheDir, "rev-parse", "--verify", strings.TrimSpace(repo.Ref)+"^{commit}"); err != nil {
			if _, checkoutErr := gitExec.Run(ctx, effectiveCacheDir, "checkout", "--force", strings.TrimSpace(repo.Ref)); checkoutErr != nil {
				return ResolvedRepo{}, []Diagnostic{{
					Level:      DiagnosticError,
					Code:       "workspace.repo.git_url.ref_invalid",
					Repo:       repo.Name,
					Path:       effectiveCacheDir,
					Message:    fmt.Sprintf("cannot checkout ref %q: %v", repo.Ref, checkoutErr),
					Suggestion: "Use an existing branch/tag/commit in workspace.yaml",
				}}
			}
		}
	}

	if strings.TrimSpace(repo.Ref) != "" {
		if _, err := gitExec.Run(ctx, effectiveCacheDir, "checkout", "--force", strings.TrimSpace(repo.Ref)); err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.checkout_failed",
				Repo:       repo.Name,
				Path:       effectiveCacheDir,
				Message:    err.Error(),
				Suggestion: "Ensure the requested ref exists and can be checked out",
			}}
		}
	}

	if _, err := gitExec.Run(ctx, effectiveCacheDir, "rev-parse", "--verify", "HEAD"); err != nil {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.git_url.invalid_head",
			Repo:       repo.Name,
			Path:       effectiveCacheDir,
			Message:    err.Error(),
			Suggestion: "Ensure repository is cloned and has a valid HEAD",
		}}
	}

	return ResolvedRepo{Name: repo.Name, Source: "git_url", Path: effectiveCacheDir, Ref: repo.Ref}, repoDiagnostics
}

func (r Root) resolveRepoCacheDir(repoName string, source string) (string, string, error) {
	slug := repoCacheSlug(repoName)
	sourceHash := repoCacheSourceHash(source)
	cacheSlug := slug
	if sourceHash != "" {
		cacheSlug = cacheSlug + "-" + sourceHash
	}
	absPath, err := r.Resolve(filepath.Join(".acp", "repos", cacheSlug))
	if err != nil {
		return "", "", err
	}
	legacyAbsPath, err := r.Resolve(filepath.Join(".acp", "repos", slug))
	if err != nil {
		return "", "", err
	}
	if legacyAbsPath == absPath {
		legacyAbsPath = ""
	}
	return absPath, legacyAbsPath, nil
}

func resolveLegacyRepoCacheFallback(primaryPath string, legacyPath string) (string, bool) {
	primaryPath = strings.TrimSpace(primaryPath)
	legacyPath = strings.TrimSpace(legacyPath)
	if primaryPath == "" || legacyPath == "" || primaryPath == legacyPath {
		return "", false
	}
	if hasGitRepoMetadata(primaryPath) {
		return "", false
	}
	if !hasGitRepoMetadata(legacyPath) {
		return "", false
	}
	return legacyPath, true
}

func hasGitRepoMetadata(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir()
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

func repoCacheSourceHash(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	encoded := hex.EncodeToString(sum[:])
	if len(encoded) <= 12 {
		return encoded
	}
	return encoded[:12]
}

func resolvePathRepoRef(ctx context.Context, gitExec GitExecutor, repoPath string, requestedRef string) (string, []Diagnostic, error) {
	ref := strings.TrimSpace(requestedRef)
	candidates := []string{ref}
	if strings.HasPrefix(ref, "origin/") {
		suffix := strings.TrimPrefix(ref, "origin/")
		if suffix != "" {
			candidates = append(candidates, "refs/remotes/origin/"+suffix)
		}
	} else if !strings.HasPrefix(ref, "refs/remotes/") {
		candidates = append(candidates, "origin/"+ref)
		candidates = append(candidates, "refs/remotes/origin/"+ref)
	}

	tried := []string{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if containsString(tried, candidate) {
			continue
		}
		tried = append(tried, candidate)
		if _, err := gitExec.Run(ctx, repoPath, "rev-parse", "--verify", candidate+"^{commit}"); err == nil {
			warnings := []Diagnostic{}
			if candidate != ref {
				warnings = append(warnings, Diagnostic{
					Level:      DiagnosticWarning,
					Code:       "workspace.repo.ref.resolved_via_remote",
					Path:       repoPath,
					Message:    fmt.Sprintf("ref %q was resolved via %q", ref, candidate),
					Suggestion: "Set ref to a local branch or keep it empty for current checkout",
				})
			}
			return candidate, warnings, nil
		}
	}

	return "", nil, fmt.Errorf("cannot resolve ref %q (tried: %s)", requestedRef, strings.Join(tried, ", "))
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
