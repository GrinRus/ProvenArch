package doctor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

const (
	StatusPass = "pass"
	StatusWarn = "warn"
	StatusFail = "fail"
)

type Check struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

type Report struct {
	OK      bool    `json:"ok"`
	Summary string  `json:"summary"`
	Checks  []Check `json:"checks"`
}

type Options struct {
	WorkspacePath       string
	RepoPath            string
	RepoGitURL          string
	RuntimeMode         string
	RuntimeProvider     string
	ListenAddress       string
	CheckPort           bool
	EmbeddedUIAvailable bool

	LookPath func(string) (string, error)
	RunGit   func(context.Context, GitCommand) (GitCommandResult, error)
}

type GitCommand struct {
	Args    []string
	Dir     string
	Timeout time.Duration
}

type GitCommandResult struct {
	Output string
}

func Run(ctx context.Context, opts Options) (Report, error) {
	if strings.TrimSpace(opts.RepoPath) != "" && strings.TrimSpace(opts.RepoGitURL) != "" {
		return Report{}, errors.New("set at most one of repo_path or repo_git_url")
	}

	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	runGit := opts.RunGit
	if runGit == nil {
		runGit = runGitCommand
	}

	mode, err := acpruntime.NormalizeMode(opts.RuntimeMode)
	if err != nil {
		return Report{}, err
	}
	provider, _, err := acpruntime.ResolveProviderWithSource(opts.RuntimeProvider)
	if err != nil {
		return Report{}, err
	}

	checks := []Check{
		checkGit(lookPath),
		checkWorkspaceWritable(strings.TrimSpace(opts.WorkspacePath)),
		checkEmbeddedUI(opts.EmbeddedUIAvailable),
	}
	if opts.CheckPort {
		checks = append(checks, checkPort(strings.TrimSpace(opts.ListenAddress)))
	}
	checks = append(checks, checkRepoSource(ctx, opts, lookPath, runGit))
	checks = append(checks, checkRuntimeProvider(mode, provider, lookPath))

	ok := true
	for _, check := range checks {
		if check.Status == StatusFail {
			ok = false
			break
		}
	}
	summary := "ready"
	if !ok {
		summary = "needs_attention"
	}
	return Report{
		OK:      ok,
		Summary: summary,
		Checks:  checks,
	}, nil
}

func checkGit(lookPath func(string) (string, error)) Check {
	path, err := lookPath("git")
	if err != nil {
		return fail("git", "Git", "git is not available in PATH", "Install git and ensure the git command is available in PATH.")
	}
	return pass("git", "Git", fmt.Sprintf("git found at %s", path))
}

func checkWorkspaceWritable(workspacePath string) Check {
	if workspacePath == "" {
		return warn("workspace", "Workspace", "workspace was not checked", "Pass --workspace to verify the ACP workspace path.")
	}
	if !filepath.IsAbs(workspacePath) {
		return fail("workspace", "Workspace", "workspace path must be absolute", "Use an absolute path such as ~/acp-workspaces/my-service expanded by your shell.")
	}

	info, err := os.Stat(workspacePath)
	if err == nil {
		if !info.IsDir() {
			return fail("workspace", "Workspace", "workspace path exists but is not a directory", "Choose a directory path for the ACP workspace.")
		}
		if err := writeTempProbe(workspacePath); err != nil {
			return fail("workspace", "Workspace", fmt.Sprintf("workspace is not writable: %v", err), "Fix directory permissions or choose another workspace path.")
		}
		return pass("workspace", "Workspace", fmt.Sprintf("workspace is writable: %s", workspacePath))
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fail("workspace", "Workspace", fmt.Sprintf("workspace cannot be inspected: %v", err), "Fix filesystem permissions or choose another workspace path.")
	}

	parent := filepath.Dir(workspacePath)
	if parent == "." || parent == "" {
		return fail("workspace", "Workspace", "workspace parent directory is invalid", "Use an absolute workspace path.")
	}
	parentInfo, parentErr := os.Stat(parent)
	if parentErr != nil {
		return fail("workspace", "Workspace", fmt.Sprintf("workspace parent does not exist: %s", parent), "Create the parent directory or choose an existing parent path.")
	}
	if !parentInfo.IsDir() {
		return fail("workspace", "Workspace", "workspace parent is not a directory", "Choose a workspace path under an existing directory.")
	}
	if err := writeTempProbe(parent); err != nil {
		return fail("workspace", "Workspace", fmt.Sprintf("workspace parent is not writable: %v", err), "Fix parent directory permissions or choose another workspace path.")
	}
	return pass("workspace", "Workspace", fmt.Sprintf("workspace can be created under %s", parent))
}

func writeTempProbe(dir string) error {
	file, err := os.CreateTemp(dir, ".acp-doctor-*")
	if err != nil {
		return err
	}
	name := file.Name()
	closeErr := file.Close()
	removeErr := os.Remove(name)
	if closeErr != nil {
		return closeErr
	}
	return removeErr
}

func checkEmbeddedUI(available bool) Check {
	if !available {
		return fail("embedded_ui", "Embedded UI", "embedded UI assets are missing", "Build the release binary with embedded UI assets.")
	}
	return pass("embedded_ui", "Embedded UI", "embedded UI assets are present")
}

func checkPort(listenAddress string) Check {
	if listenAddress == "" {
		listenAddress = "127.0.0.1:8080"
	}
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return fail("port", "Listen port", fmt.Sprintf("cannot listen on %s: %v", listenAddress, err), "Stop the process using this port or start ACP with --listen on another address.")
	}
	_ = listener.Close()
	return pass("port", "Listen port", fmt.Sprintf("%s is available", listenAddress))
}

func checkRepoSource(ctx context.Context, opts Options, lookPath func(string) (string, error), runGit func(context.Context, GitCommand) (GitCommandResult, error)) Check {
	repoPath := strings.TrimSpace(opts.RepoPath)
	repoGitURL := strings.TrimSpace(opts.RepoGitURL)
	if repoPath == "" && repoGitURL == "" {
		return warn("repo_source", "Repository source", "repository source was not checked", "Pass --repo-path or --repo-git-url to verify source repository access.")
	}
	if _, err := lookPath("git"); err != nil {
		return fail("repo_source", "Repository source", "repository source requires git", "Install git before checking repository access.")
	}
	if repoPath != "" {
		return checkRepoPath(ctx, repoPath, runGit)
	}
	return checkRepoGitURL(ctx, repoGitURL, runGit)
}

func checkRepoPath(ctx context.Context, repoPath string, runGit func(context.Context, GitCommand) (GitCommandResult, error)) Check {
	if !filepath.IsAbs(repoPath) {
		return fail("repo_source", "Repository source", "repo path must be absolute", "Use an absolute local checkout path.")
	}
	info, err := os.Stat(repoPath)
	if err != nil {
		return fail("repo_source", "Repository source", fmt.Sprintf("repo path cannot be inspected: %v", err), "Clone the repository locally or use --repo-git-url.")
	}
	if !info.IsDir() {
		return fail("repo_source", "Repository source", "repo path is not a directory", "Use the repository checkout directory.")
	}
	if _, err := runGit(ctx, GitCommand{Args: []string{"rev-parse", "--is-inside-work-tree"}, Dir: repoPath, Timeout: 10 * time.Second}); err != nil {
		return fail("repo_source", "Repository source", fmt.Sprintf("repo path is not a git checkout: %v", err), "Use a git checkout directory or use --repo-git-url.")
	}
	return pass("repo_source", "Repository source", fmt.Sprintf("local git checkout is readable: %s", repoPath))
}

func checkRepoGitURL(ctx context.Context, repoGitURL string, runGit func(context.Context, GitCommand) (GitCommandResult, error)) Check {
	if repoGitURL == "" {
		return warn("repo_source", "Repository source", "repository URL was not checked", "Pass --repo-git-url to verify remote access.")
	}
	if _, err := runGit(ctx, GitCommand{Args: []string{"ls-remote", "--heads", repoGitURL}, Timeout: 20 * time.Second}); err != nil {
		return fail("repo_source", "Repository source", fmt.Sprintf("git cannot access %s: %v", repoGitURL, err), "Check the repository URL and your local git authentication.")
	}
	return pass("repo_source", "Repository source", fmt.Sprintf("git can access %s", repoGitURL))
}

func checkRuntimeProvider(mode string, provider acpruntime.Provider, lookPath func(string) (string, error)) Check {
	if mode == acpruntime.RuntimeModeFake {
		return pass("runtime_provider", "Runtime provider", "fake runtime selected; no headless provider command required")
	}
	commands := providerCommands(provider)
	for _, command := range commands {
		path, err := lookPath(command)
		if err == nil {
			return pass("runtime_provider", "Runtime provider", fmt.Sprintf("Provider ID: %s; executable: %s found at %s", provider, command, path))
		}
	}
	return fail("runtime_provider", "Runtime provider", fmt.Sprintf("Provider ID: %s; executable not found; checked: %s", provider, strings.Join(commands, ", ")), providerSuggestion(provider))
}

func providerCommands(provider acpruntime.Provider) []string {
	switch provider {
	case acpruntime.ProviderQwenCode:
		if value := strings.TrimSpace(os.Getenv("ACP_QWEN_CMD")); value != "" {
			return []string{value}
		}
		return []string{"qwen"}
	case acpruntime.ProviderCodexCode:
		if value := strings.TrimSpace(os.Getenv("ACP_CODEX_CMD")); value != "" {
			return []string{value}
		}
		return []string{"codex"}
	default:
		if value := strings.TrimSpace(os.Getenv("ACP_CLAUDE_CMD")); value != "" {
			return []string{value}
		}
		return []string{"claude", "claude-code"}
	}
}

func providerSuggestion(provider acpruntime.Provider) string {
	switch provider {
	case acpruntime.ProviderQwenCode:
		return "Install qwen or set ACP_QWEN_CMD to the provider command."
	case acpruntime.ProviderCodexCode:
		return "Install codex or set ACP_CODEX_CMD to the provider command."
	default:
		return "Install claude or set ACP_CLAUDE_CMD to the provider command. Legacy command claude-code is also supported."
	}
}

func runGitCommand(ctx context.Context, command GitCommand) (GitCommandResult, error) {
	timeout := command.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "git", command.Args...)
	cmd.Dir = strings.TrimSpace(command.Dir)
	output, err := cmd.CombinedOutput()
	if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
		return GitCommandResult{}, fmt.Errorf("git command timed out after %s", timeout)
	}
	if err != nil {
		return GitCommandResult{}, fmt.Errorf("git %s failed: %w: %s", strings.Join(command.Args, " "), err, strings.TrimSpace(string(output)))
	}
	return GitCommandResult{Output: strings.TrimSpace(string(output))}, nil
}

func pass(id string, label string, message string) Check {
	return Check{ID: id, Label: label, Status: StatusPass, Message: message}
}

func warn(id string, label string, message string, suggestion string) Check {
	return Check{ID: id, Label: label, Status: StatusWarn, Message: message, Suggestion: suggestion}
}

func fail(id string, label string, message string, suggestion string) Check {
	return Check{ID: id, Label: label, Status: StatusFail, Message: message, Suggestion: suggestion}
}
