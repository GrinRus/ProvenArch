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
	Name        string `json:"name"`
	Source      string `json:"source"`
	Path        string `json:"path"`
	Ref         string `json:"ref,omitempty"`
	ResolvedSHA string `json:"resolved_sha,omitempty"`
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

	cacheDir, err := r.resolveRepoCacheDir(repo.Name, repo.GitURL)
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

	resolvedSHA := ""
	if strings.TrimSpace(repo.Ref) != "" {
		resolvedSHA, err = resolveGitURLRef(ctx, gitExec, effectiveCacheDir, strings.TrimSpace(repo.Ref))
		if err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.ref_invalid",
				Repo:       repo.Name,
				Path:       effectiveCacheDir,
				Message:    fmt.Sprintf("cannot resolve ref %q: %v", repo.Ref, err),
				Suggestion: "Ensure the requested remote branch/tag/commit exists and can be fetched",
			}}
		}
		if err := checkoutExactGitCommit(ctx, gitExec, effectiveCacheDir, resolvedSHA); err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.invalid_head",
				Repo:       repo.Name,
				Path:       effectiveCacheDir,
				Message:    fmt.Sprintf("cannot checkout resolved ref %q at %s: %v", repo.Ref, resolvedSHA, err),
				Suggestion: "Ensure the requested ref resolves to a valid commit",
			}}
		}
	} else {
		remoteRef, sha, err := resolveGitURLRemoteDefaultHead(ctx, gitExec, effectiveCacheDir)
		if err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.default_head_failed",
				Repo:       repo.Name,
				Path:       effectiveCacheDir,
				Message:    err.Error(),
				Suggestion: "Ensure the remote has a default branch HEAD and can be fetched by git",
			}}
		}
		if err := checkoutExactGitCommit(ctx, gitExec, effectiveCacheDir, sha); err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.reset_failed",
				Repo:       repo.Name,
				Path:       effectiveCacheDir,
				Message:    fmt.Sprintf("cannot reset git_url cache to %s (%s): %v", sha, remoteRef, err),
				Suggestion: "Remove the ACP-owned .acp/repos cache entry and retry, or verify repository permissions",
			}}
		}
		resolvedSHA = strings.TrimSpace(sha)
	}

	if strings.TrimSpace(resolvedSHA) == "" {
		head, err := gitExec.Run(ctx, effectiveCacheDir, "rev-parse", "--verify", "HEAD^{commit}")
		if err != nil {
			return ResolvedRepo{}, []Diagnostic{{
				Level:      DiagnosticError,
				Code:       "workspace.repo.git_url.invalid_head",
				Repo:       repo.Name,
				Path:       effectiveCacheDir,
				Message:    err.Error(),
				Suggestion: "Ensure repository is cloned and has a valid HEAD",
			}}
		}
		resolvedSHA = strings.TrimSpace(head)
	}

	head, err := gitExec.Run(ctx, effectiveCacheDir, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.git_url.invalid_head",
			Repo:       repo.Name,
			Path:       effectiveCacheDir,
			Message:    err.Error(),
			Suggestion: "Ensure repository is cloned and has a valid HEAD",
		}}
	}
	if resolvedSHA != "" && strings.TrimSpace(head) != resolvedSHA {
		return ResolvedRepo{}, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.repo.git_url.identity_mismatch",
			Repo:       repo.Name,
			Path:       effectiveCacheDir,
			Message:    fmt.Sprintf("resolved ref %q is %s, but cache HEAD is %s", repo.Ref, resolvedSHA, strings.TrimSpace(head)),
			Suggestion: "Retry source resolution so the ACP-owned cache is checked out at the resolved commit",
		}}
	}

	return ResolvedRepo{Name: repo.Name, Source: "git_url", Path: effectiveCacheDir, Ref: repo.Ref, ResolvedSHA: resolvedSHA}, repoDiagnostics
}

func resolveGitURLRef(ctx context.Context, gitExec GitExecutor, repoPath string, requestedRef string) (string, error) {
	ref := strings.TrimSpace(requestedRef)
	if ref == "" {
		return "", fmt.Errorf("requested ref is empty")
	}

	candidates := gitURLRefCandidates(ref)
	tried := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || containsString(tried, candidate) {
			continue
		}
		tried = append(tried, candidate)
		resolved, err := gitExec.Run(ctx, repoPath, "rev-parse", "--verify", candidate+"^{commit}")
		if err != nil {
			continue
		}
		resolved = strings.TrimSpace(resolved)
		if resolved != "" {
			return resolved, nil
		}
	}

	return "", fmt.Errorf("cannot resolve ref %q (tried: %s)", requestedRef, strings.Join(tried, ", "))
}

func gitURLRefCandidates(ref string) []string {
	if strings.HasPrefix(ref, "refs/remotes/") {
		return []string{ref}
	}
	if strings.HasPrefix(ref, "origin/") {
		return []string{"refs/remotes/" + ref, ref}
	}
	if strings.HasPrefix(ref, "refs/heads/") {
		return []string{"refs/remotes/origin/" + strings.TrimPrefix(ref, "refs/heads/")}
	}
	if strings.HasPrefix(ref, "refs/tags/") || strings.HasPrefix(ref, "refs/") {
		return []string{ref}
	}
	if isFullCommitSHA(ref) {
		return []string{ref}
	}
	// A plain ref in a remote source is a branch name first. Resolving the
	// freshly fetched remote-tracking ref avoids reusing a stale local branch.
	// The explicit tag candidate preserves the documented branch/tag behavior
	// without allowing a deleted remote branch to fall back to cache state.
	return []string{"refs/remotes/origin/" + ref, "refs/tags/" + ref}
}

func isFullCommitSHA(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') && !(r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}

func resolveGitURLRemoteDefaultHead(ctx context.Context, gitExec GitExecutor, repoPath string) (string, string, error) {
	if output, err := gitExec.Run(ctx, repoPath, "ls-remote", "--symref", "origin", "HEAD"); err == nil {
		remoteRef, sha := parseRemoteHeadSymref(output)
		if remoteRef != "" {
			resolvedSHA, revErr := gitExec.Run(ctx, repoPath, "rev-parse", "--verify", remoteRef+"^{commit}")
			if revErr == nil {
				return remoteRef, strings.TrimSpace(resolvedSHA), nil
			}
		}
		if sha != "" {
			resolvedSHA, revErr := gitExec.Run(ctx, repoPath, "rev-parse", "--verify", sha+"^{commit}")
			if revErr == nil {
				return "HEAD", strings.TrimSpace(resolvedSHA), nil
			}
		}
	}

	_, _ = gitExec.Run(ctx, repoPath, "remote", "set-head", "origin", "--auto")
	remoteRef, err := gitExec.Run(ctx, repoPath, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", "", fmt.Errorf("cannot resolve remote default HEAD: %w", err)
	}
	remoteRef = strings.TrimSpace(remoteRef)
	if remoteRef == "" {
		return "", "", fmt.Errorf("cannot resolve remote default HEAD: empty origin/HEAD")
	}
	resolvedSHA, err := gitExec.Run(ctx, repoPath, "rev-parse", "--verify", remoteRef+"^{commit}")
	if err != nil {
		return "", "", fmt.Errorf("cannot resolve remote default ref %q: %w", remoteRef, err)
	}
	return remoteRef, strings.TrimSpace(resolvedSHA), nil
}

func parseRemoteHeadSymref(output string) (string, string) {
	remoteRef := ""
	sha := ""
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "ref:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[2] == "HEAD" && strings.HasPrefix(fields[1], "refs/heads/") {
				branch := strings.TrimPrefix(fields[1], "refs/heads/")
				if branch != "" {
					remoteRef = "refs/remotes/origin/" + branch
				}
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "HEAD" {
			sha = fields[0]
		}
	}
	return remoteRef, sha
}

func checkoutExactGitCommit(ctx context.Context, gitExec GitExecutor, repoPath string, sha string) error {
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return fmt.Errorf("resolved commit SHA is empty")
	}
	if _, err := gitExec.Run(ctx, repoPath, "checkout", "--force", "--detach", sha); err != nil {
		return err
	}
	if _, err := gitExec.Run(ctx, repoPath, "reset", "--hard", sha); err != nil {
		return err
	}
	if _, err := gitExec.Run(ctx, repoPath, "clean", "-ffdx"); err != nil {
		return err
	}
	return nil
}

func (r Root) resolveRepoCacheDir(repoName string, source string) (string, error) {
	slug := repoCacheSlug(repoName)
	sourceHash := repoCacheSourceHash(source)
	cacheSlug := slug
	if sourceHash != "" {
		cacheSlug = cacheSlug + "-" + sourceHash
	}
	absPath, err := r.Resolve(filepath.Join(".acp", "repos", cacheSlug))
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
