package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/api"
	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	"github.com/GrinRus/ProvenArch/internal/qa"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/providers"
	"github.com/GrinRus/ProvenArch/internal/workspace"
	"gopkg.in/yaml.v3"
)

const (
	exitCodeOK             = 0
	exitCodeInvalidCommand = 2
	exitCodeValidation     = 3
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		printRootUsage(stdout)
		return exitCodeOK
	}

	switch args[0] {
	case "init-workspace":
		return runInitWorkspace(args[1:], stdout, stderr)
	case "serve":
		return runServe(args[1:], stdout, stderr)
	case "run":
		return runPipeline(args[1:], stdout, stderr)
	case "qa":
		return runQA(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printRootUsage(stderr)
		return exitCodeInvalidCommand
	}
}

func runServe(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)

	workspacePath := fs.String("workspace", "", "absolute path to arch-workspace")
	listenAddress := fs.String("listen", "127.0.0.1:8080", "listen address for local API server")
	runtimeMode := fs.String("runtime", "fake", "runtime mode: fake or headless")
	runtimeProvider := fs.String("runtime-provider", "", "runtime provider for headless mode: claude-code or qwen-code (fallback: ACP_RUNTIME_PROVIDER)")
	executionStrategy := fs.String("execution-strategy", "", "execution strategy override: sequential or parallel")
	maxParallelTasks := fs.Int("max-parallel-tasks", 0, "execution max parallel tasks override (>0)")
	failurePolicy := fs.String("failure-policy", "", "execution failure policy override: fail_fast or best_effort")
	runLogsTTLHrs := fs.Int("run-logs-ttl-hours", envInt("ACP_RUN_LOGS_TTL_HOURS", 168), "run logs retention TTL in hours")
	runLogsMaxRuns := fs.Int("run-logs-max-runs", envInt("ACP_RUN_LOGS_MAX_RUNS", 200), "maximum number of run log files to retain")
	dryRun := fs.Bool("dry-run", false, "validate workspace and server wiring without starting listener")
	autoInit := fs.Bool("auto-init", false, "bootstrap workspace manifest/layout when workspace.yaml is missing")
	repoName := fs.String("repo-name", "", "repo scope name for --auto-init")
	repoPath := fs.String("repo-path", "", "local repo checkout path (absolute or relative) for --auto-init")
	repoGitURL := fs.String("repo-git-url", "", "git repository URL source for --auto-init")
	repoRef := fs.String("repo-ref", "", "optional repo ref (branch/tag/commit) for --auto-init")
	reposFile := fs.String("repos-file", "", "YAML file with repos[] entries for --auto-init")
	docsImportsPath := fs.String("docs-imports-path", "./docs/imports", "docs imports path in workspace.yaml for --auto-init")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: acp serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--listen 127.0.0.1:8080] [--dry-run] [--auto-init ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path ./docs/imports]]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitCodeOK
		}
		return exitCodeInvalidCommand
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(stderr, "unexpected positional arguments: %s\n", strings.Join(fs.Args(), " "))
		fs.Usage()
		return exitCodeInvalidCommand
	}
	if *runLogsTTLHrs <= 0 {
		fmt.Fprintln(stderr, "run logs validation failed: --run-logs-ttl-hours must be > 0")
		return exitCodeValidation
	}
	if *runLogsMaxRuns <= 0 {
		fmt.Fprintln(stderr, "run logs validation failed: --run-logs-max-runs must be > 0")
		return exitCodeValidation
	}
	executionOverrides, err := executionOverridesFromCLI(*executionStrategy, *maxParallelTasks, *failurePolicy)
	if err != nil {
		fmt.Fprintf(stderr, "execution validation failed: %v\n", err)
		return exitCodeValidation
	}

	ws, err := openOrAutoInitWorkspace(workspaceInitConfig{
		WorkspacePath:   *workspacePath,
		RepoName:        *repoName,
		RepoPath:        *repoPath,
		RepoGitURL:      *repoGitURL,
		RepoRef:         *repoRef,
		ReposFile:       *reposFile,
		DocsImportsPath: *docsImportsPath,
		Force:           false,
		RequireRepo:     true,
	}, *autoInit)
	if err != nil {
		fmt.Fprintf(stderr, "workspace validation failed: %v\n", err)
		return exitCodeValidation
	}
	if err := ws.EnsureLayout(); err != nil {
		fmt.Fprintf(stderr, "workspace validation failed: ensure workspace layout: %v\n", err)
		return exitCodeValidation
	}

	mode, err := acpruntime.NormalizeMode(*runtimeMode)
	if err != nil {
		fmt.Fprintf(stderr, "runtime validation failed: %v\n", err)
		return exitCodeValidation
	}
	provider, err := acpruntime.ResolveProvider(*runtimeProvider)
	if err != nil {
		fmt.Fprintf(stderr, "runtime validation failed: %v\n", err)
		return exitCodeValidation
	}

	runner, err := buildRunner(mode, provider)
	if err != nil {
		fmt.Fprintf(stderr, "runtime validation failed: %v\n", err)
		return exitCodeValidation
	}
	service := orchestrator.NewService(
		orchestrator.WithRunner(runner),
		orchestrator.WithHistoryWorkspace(ws),
		orchestrator.WithRunLogsRetention(time.Duration(*runLogsTTLHrs)*time.Hour, *runLogsMaxRuns),
		orchestrator.WithExecutionOverrides(executionOverrides),
	)
	if err := service.ValidateRuntime(context.Background()); err != nil {
		printRunnerError(stderr, err)
		return exitCodeValidation
	}
	server := api.NewServer(ws, service)
	if *dryRun {
		fmt.Fprintf(stdout, "workspace ready at %s\n", ws.Path)
		fmt.Fprintf(stdout, "server configured for %s\n", *listenAddress)
		fmt.Fprintf(stdout, "runtime mode: %s\n", mode)
		fmt.Fprintf(stdout, "runtime provider: %s\n", provider)
		executionResolved := service.ResolveExecutionProfile(ws.Manifest)
		fmt.Fprintf(stdout, "execution strategy: %s\n", executionResolved.Effective.Strategy)
		fmt.Fprintf(stdout, "execution max_parallel_tasks: %d\n", executionResolved.Effective.MaxParallel)
		fmt.Fprintf(stdout, "execution failure_policy: %s\n", executionResolved.Effective.FailurePolicy)
		if mode == acpruntime.RuntimeModeFake {
			fmt.Fprintln(stdout, "runtime provider note: ignored in fake mode")
		}
		return exitCodeOK
	}

	fmt.Fprintf(stdout, "starting ACP API server on %s for workspace %s\n", *listenAddress, ws.Path)
	if err := server.Serve(context.Background(), *listenAddress); err != nil {
		fmt.Fprintf(stderr, "serve failed: %v\n", err)
		return exitCodeValidation
	}

	return exitCodeOK
}

func runInitWorkspace(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init-workspace", flag.ContinueOnError)
	fs.SetOutput(stderr)

	workspacePath := fs.String("workspace", "", "absolute path to arch-workspace")
	repoName := fs.String("repo-name", "", "repo scope name")
	repoPath := fs.String("repo-path", "", "local repo checkout path (absolute or relative)")
	repoGitURL := fs.String("repo-git-url", "", "git repository URL source")
	repoRef := fs.String("repo-ref", "", "optional repo ref (branch/tag/commit)")
	reposFile := fs.String("repos-file", "", "YAML file with repos[] entries")
	docsImportsPath := fs.String("docs-imports-path", "./docs/imports", "docs imports path in workspace.yaml")
	force := fs.Bool("force", false, "overwrite existing workspace.yaml")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: acp init-workspace --workspace <abs-path> ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path ./docs/imports] [--force]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitCodeOK
		}
		return exitCodeInvalidCommand
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(stderr, "unexpected positional arguments: %s\n", strings.Join(fs.Args(), " "))
		fs.Usage()
		return exitCodeInvalidCommand
	}

	ws, err := createWorkspaceFromConfig(workspaceInitConfig{
		WorkspacePath:   *workspacePath,
		RepoName:        *repoName,
		RepoPath:        *repoPath,
		RepoGitURL:      *repoGitURL,
		RepoRef:         *repoRef,
		ReposFile:       *reposFile,
		DocsImportsPath: *docsImportsPath,
		Force:           *force,
		RequireRepo:     true,
	})
	if err != nil {
		fmt.Fprintf(stderr, "workspace init failed: %v\n", err)
		return exitCodeValidation
	}

	report := ws.Validate(context.Background(), workspace.ValidateOptions{
		ResolveRepos: true,
		FetchGit:     false,
		VerifyRefs:   true,
	})
	if !report.OK {
		printValidationReport(stderr, report)
		return exitCodeValidation
	}

	fmt.Fprintf(stdout, "workspace initialized at %s\n", ws.Path)
	fmt.Fprintf(stdout, "manifest: %s\n", ws.ManifestPath)
	for _, warning := range report.Warnings {
		fmt.Fprintf(stdout, "- [warning] %s: %s\n", warning.Code, warning.Message)
	}
	fmt.Fprintln(stdout, "next commands:")
	fmt.Fprintf(stdout, "  acp run --workspace %s --pipeline init --runtime fake --non-interactive\n", ws.Path)
	fmt.Fprintf(stdout, "  acp serve --workspace %s --runtime fake\n", ws.Path)
	return exitCodeOK
}

type workspaceInitConfig struct {
	WorkspacePath   string
	RepoName        string
	RepoPath        string
	RepoGitURL      string
	RepoRef         string
	ReposFile       string
	Repos           []workspace.RepoSource
	Runtime         *workspace.RuntimeConfig
	DocsImportsPath string
	Force           bool
	RequireRepo     bool
}

func createWorkspaceFromConfig(config workspaceInitConfig) (workspace.Root, error) {
	normalized, err := normalizeWorkspaceInitConfig(config)
	if err != nil {
		return workspace.Root{}, err
	}
	if err := os.MkdirAll(normalized.WorkspacePath, 0o755); err != nil {
		return workspace.Root{}, fmt.Errorf("create workspace directory: %w", err)
	}
	if err := ensureWorkspaceGitRepository(normalized.WorkspacePath); err != nil {
		return workspace.Root{}, err
	}

	manifestPath := filepath.Join(normalized.WorkspacePath, workspace.ManifestFileName)
	if _, err := os.Stat(manifestPath); err == nil && !normalized.Force {
		return workspace.Root{}, fmt.Errorf("%s already exists (use --force to overwrite)", manifestPath)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return workspace.Root{}, fmt.Errorf("stat workspace manifest: %w", err)
	}

	repos := append([]workspace.RepoSource(nil), normalized.Repos...)
	if len(repos) == 0 {
		repos = []workspace.RepoSource{{
			Name:   normalized.RepoName,
			Path:   normalized.RepoPath,
			GitURL: normalized.RepoGitURL,
			Ref:    normalized.RepoRef,
		}}
	}
	manifest := workspace.Manifest{
		Version: 1,
		Repos:   repos,
		Docs:    workspace.DocsConfig{ImportsPath: normalized.DocsImportsPath},
		Runtime: cloneRuntimeConfig(normalized.Runtime),
	}

	manifestContent, err := yaml.Marshal(manifest)
	if err != nil {
		return workspace.Root{}, fmt.Errorf("encode workspace manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestContent, 0o644); err != nil {
		return workspace.Root{}, fmt.Errorf("write workspace manifest: %w", err)
	}

	ws, err := workspace.Open(normalized.WorkspacePath)
	if err != nil {
		return workspace.Root{}, fmt.Errorf("reopen workspace: %w", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		return workspace.Root{}, fmt.Errorf("ensure workspace layout: %w", err)
	}
	if err := ws.EnsureBaselineBundle(); err != nil {
		return workspace.Root{}, fmt.Errorf("ensure baseline bundle: %w", err)
	}
	return ws, nil
}

func openOrAutoInitWorkspace(config workspaceInitConfig, autoInit bool) (workspace.Root, error) {
	workspacePath := strings.TrimSpace(config.WorkspacePath)
	if workspacePath == "" {
		return workspace.Root{}, errors.New("--workspace is required")
	}
	if !filepath.IsAbs(workspacePath) {
		return workspace.Root{}, errors.New("--workspace must be absolute")
	}

	ws, err := workspace.Open(workspacePath)
	if err == nil {
		return ws, nil
	}
	if !autoInit {
		return workspace.Root{}, err
	}

	if !errors.Is(err, workspace.ErrManifestMissing) && !errors.Is(err, os.ErrNotExist) {
		return workspace.Root{}, err
	}

	bootstrapConfig := config
	bootstrapConfig.WorkspacePath = workspacePath
	bootstrapConfig.RequireRepo = true
	bootstrapConfig.Force = false
	return createWorkspaceFromConfig(bootstrapConfig)
}

func normalizeWorkspaceInitConfig(config workspaceInitConfig) (workspaceInitConfig, error) {
	workspacePath := strings.TrimSpace(config.WorkspacePath)
	if workspacePath == "" {
		return workspaceInitConfig{}, errors.New("--workspace is required")
	}
	if !filepath.IsAbs(workspacePath) {
		return workspaceInitConfig{}, errors.New("--workspace must be absolute")
	}

	repoName := strings.TrimSpace(config.RepoName)
	repoPath := strings.TrimSpace(config.RepoPath)
	repoGitURL := strings.TrimSpace(config.RepoGitURL)
	repoRef := strings.TrimSpace(config.RepoRef)
	reposFile := strings.TrimSpace(config.ReposFile)
	importsPath := strings.TrimSpace(config.DocsImportsPath)
	if importsPath == "" {
		importsPath = "./docs/imports"
	}

	repos := []workspace.RepoSource{}
	var runtimeConfig *workspace.RuntimeConfig
	if reposFile != "" {
		if repoName != "" || repoPath != "" || repoGitURL != "" || repoRef != "" {
			return workspaceInitConfig{}, errors.New("set either --repos-file or single-repo flags (--repo-name + --repo-path|--repo-git-url)")
		}
		loadedRepos, loadedRuntime, err := loadRepoSourcesAndRuntimeFromFile(reposFile)
		if err != nil {
			return workspaceInitConfig{}, err
		}
		repos = loadedRepos
		runtimeConfig = loadedRuntime
	}

	if config.RequireRepo && len(repos) == 0 {
		if repoName == "" {
			return workspaceInitConfig{}, errors.New("--repo-name is required")
		}
		hasPath := repoPath != ""
		hasGitURL := repoGitURL != ""
		if hasPath == hasGitURL {
			return workspaceInitConfig{}, errors.New("set exactly one of --repo-path or --repo-git-url")
		}
	}
	if len(repos) == 0 && repoPath != "" {
		resolvedPath, err := filepath.Abs(repoPath)
		if err != nil {
			return workspaceInitConfig{}, fmt.Errorf("resolve --repo-path: %w", err)
		}
		repoPath = resolvedPath
	}

	return workspaceInitConfig{
		WorkspacePath:   workspacePath,
		RepoName:        repoName,
		RepoPath:        repoPath,
		RepoGitURL:      repoGitURL,
		RepoRef:         repoRef,
		ReposFile:       reposFile,
		Repos:           repos,
		Runtime:         runtimeConfig,
		DocsImportsPath: importsPath,
		Force:           config.Force,
		RequireRepo:     config.RequireRepo,
	}, nil
}

func runPipeline(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	workspacePath := fs.String("workspace", "", "absolute path to arch-workspace")
	pipelineName := fs.String("pipeline", "", "pipeline to run: init or refresh")
	runtimeMode := fs.String("runtime", "fake", "runtime mode: fake or headless")
	runtimeProvider := fs.String("runtime-provider", "", "runtime provider for headless mode: claude-code or qwen-code (fallback: ACP_RUNTIME_PROVIDER)")
	executionStrategy := fs.String("execution-strategy", "", "execution strategy override: sequential or parallel")
	maxParallelTasks := fs.Int("max-parallel-tasks", 0, "execution max parallel tasks override (>0)")
	failurePolicy := fs.String("failure-policy", "", "execution failure policy override: fail_fast or best_effort")
	runLogsTTLHrs := fs.Int("run-logs-ttl-hours", envInt("ACP_RUN_LOGS_TTL_HOURS", 168), "run logs retention TTL in hours")
	runLogsMaxRuns := fs.Int("run-logs-max-runs", envInt("ACP_RUN_LOGS_MAX_RUNS", 200), "maximum number of run log files to retain")
	nonInteractive := fs.Bool("non-interactive", false, "disable interactive prompts")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitCodeOK
		}
		return exitCodeInvalidCommand
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(stderr, "unexpected positional arguments: %s\n", strings.Join(fs.Args(), " "))
		fs.Usage()
		return exitCodeInvalidCommand
	}
	if *runLogsTTLHrs <= 0 {
		fmt.Fprintln(stderr, "run logs validation failed: --run-logs-ttl-hours must be > 0")
		return exitCodeValidation
	}
	if *runLogsMaxRuns <= 0 {
		fmt.Fprintln(stderr, "run logs validation failed: --run-logs-max-runs must be > 0")
		return exitCodeValidation
	}
	executionOverrides, err := executionOverridesFromCLI(*executionStrategy, *maxParallelTasks, *failurePolicy)
	if err != nil {
		fmt.Fprintf(stderr, "execution validation failed: %v\n", err)
		return exitCodeValidation
	}

	ws, err := workspace.Open(*workspacePath)
	if err != nil {
		fmt.Fprintf(stderr, "workspace validation failed: %v\n", err)
		return exitCodeValidation
	}

	pipeline, err := orchestrator.ParsePipeline(*pipelineName)
	if err != nil {
		fmt.Fprintf(stderr, "pipeline validation failed: %v\n", err)
		return exitCodeValidation
	}
	report := ws.Validate(context.Background(), workspace.ValidateOptions{
		ResolveRepos: true,
		FetchGit:     true,
		VerifyRefs:   true,
	})
	if !report.OK {
		printValidationReport(stderr, report)
		return exitCodeValidation
	}

	mode, err := acpruntime.NormalizeMode(*runtimeMode)
	if err != nil {
		fmt.Fprintf(stderr, "runtime validation failed: %v\n", err)
		return exitCodeValidation
	}
	provider, err := acpruntime.ResolveProvider(*runtimeProvider)
	if err != nil {
		fmt.Fprintf(stderr, "runtime validation failed: %v\n", err)
		return exitCodeValidation
	}

	runner, err := buildRunner(mode, provider)
	if err != nil {
		fmt.Fprintf(stderr, "runtime validation failed: %v\n", err)
		return exitCodeValidation
	}
	service := orchestrator.NewService(
		orchestrator.WithRunner(runner),
		orchestrator.WithHistoryWorkspace(ws),
		orchestrator.WithRunLogsRetention(time.Duration(*runLogsTTLHrs)*time.Hour, *runLogsMaxRuns),
		orchestrator.WithExecutionOverrides(executionOverrides),
	)
	if err := service.ValidateRuntime(context.Background()); err != nil {
		printRunnerError(stderr, err)
		return exitCodeValidation
	}
	runInfo, artifacts, err := service.Run(context.Background(), orchestrator.RunRequest{
		Workspace:      ws,
		Pipeline:       pipeline,
		NonInteractive: *nonInteractive,
	})
	if err != nil {
		if runInfo.ErrorCode != "" {
			fmt.Fprintf(stderr, "run failed (%s): %s\n", runInfo.ErrorCode, runInfo.Error)
		} else {
			fmt.Fprintf(stderr, "run failed: %v\n", err)
		}
		return exitCodeValidation
	}
	fmt.Fprintf(stdout, "workspace: %s\n", ws.Path)
	fmt.Fprintf(stdout, "run_id: %s\n", runInfo.RunID)
	fmt.Fprintf(stdout, "pipeline: %s\n", runInfo.Pipeline)
	fmt.Fprintf(stdout, "status: %s\n", runInfo.Status)
	fmt.Fprintf(stdout, "runtime mode: %s\n", mode)
	fmt.Fprintf(stdout, "runtime provider: %s\n", provider)
	executionResolved := service.ResolveExecutionProfile(ws.Manifest)
	fmt.Fprintf(stdout, "execution strategy: %s\n", executionResolved.Effective.Strategy)
	fmt.Fprintf(stdout, "execution max_parallel_tasks: %d\n", executionResolved.Effective.MaxParallel)
	fmt.Fprintf(stdout, "execution failure_policy: %s\n", executionResolved.Effective.FailurePolicy)
	if mode == acpruntime.RuntimeModeFake {
		fmt.Fprintln(stdout, "runtime provider note: ignored in fake mode")
	}
	fmt.Fprintf(stdout, "artifacts: %d\n", len(artifacts))

	return exitCodeOK
}

func runQA(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("qa", flag.ContinueOnError)
	fs.SetOutput(stderr)

	workspacePath := fs.String("workspace", "", "absolute path to arch-workspace")
	question := fs.String("question", "", "architecture question to ask against workspace artifacts")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: acp qa --workspace <abs-path> --question \"<text>\"")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitCodeOK
		}
		return exitCodeInvalidCommand
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(stderr, "unexpected positional arguments: %s\n", strings.Join(fs.Args(), " "))
		fs.Usage()
		return exitCodeInvalidCommand
	}

	ws, err := workspace.Open(*workspacePath)
	if err != nil {
		fmt.Fprintf(stderr, "workspace validation failed: %v\n", err)
		return exitCodeValidation
	}

	trimmedQuestion := strings.TrimSpace(*question)
	if trimmedQuestion == "" {
		fmt.Fprintln(stderr, "qa validation failed: --question is required")
		return exitCodeValidation
	}

	service := qa.NewService()
	response, err := service.Ask(context.Background(), ws, trimmedQuestion)
	if err != nil {
		fmt.Fprintf(stderr, "qa failed: %v\n", err)
		return exitCodeValidation
	}

	fmt.Fprintf(stdout, "question: %s\n", trimmedQuestion)
	fmt.Fprintf(stdout, "answer: %s\n", strings.TrimSpace(response.Answer))
	fmt.Fprintf(stdout, "confidence: %.2f\n", response.Confidence)
	if len(response.Citations) == 0 {
		fmt.Fprintln(stdout, "citations: none")
	} else {
		fmt.Fprintln(stdout, "citations:")
		for _, citation := range response.Citations {
			fmt.Fprintf(stdout, "- %s (%s)\n", citation.Path, citation.Reason)
		}
	}
	if len(response.Unresolved) == 0 {
		fmt.Fprintln(stdout, "unresolved: none")
	} else {
		fmt.Fprintln(stdout, "unresolved:")
		for _, reason := range response.Unresolved {
			fmt.Fprintf(stdout, "- %s\n", reason)
		}
	}
	return exitCodeOK
}

func isHelp(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}

func printRootUsage(w io.Writer) {
	fmt.Fprintln(w, "ACP CLI")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  acp init-workspace --workspace <abs-path> ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path ./docs/imports]")
	fmt.Fprintln(w, "  acp serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--auto-init ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path ./docs/imports]]")
	fmt.Fprintln(w, "  acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]")
	fmt.Fprintln(w, "  acp qa --workspace <abs-path> --question \"<text>\"")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init-workspace create/update workspace.yaml and bootstrap workspace layout")
	fmt.Fprintln(w, "  serve   load workspace and start local API+UI service")
	fmt.Fprintln(w, "  run     validate workspace path and execute init/refresh pipeline")
	fmt.Fprintln(w, "  qa      ask read-only questions over workspace artifacts")
}

func printValidationReport(w io.Writer, report workspace.ValidationReport) {
	fmt.Fprintf(w, "workspace validation failed for %s\n", report.Workspace)
	for _, diagnostic := range report.Errors {
		fmt.Fprintf(w, "- [error] %s: %s\n", diagnostic.Code, diagnostic.Message)
		if strings.TrimSpace(diagnostic.Suggestion) != "" {
			fmt.Fprintf(w, "  suggestion: %s\n", diagnostic.Suggestion)
		}
	}
	for _, diagnostic := range report.Warnings {
		fmt.Fprintf(w, "- [warning] %s: %s\n", diagnostic.Code, diagnostic.Message)
	}
}

func buildRunner(runtimeMode string, provider acpruntime.Provider) (acpruntime.Runner, error) {
	return providers.BuildRunner(runtimeMode, provider)
}

func printRunnerError(w io.Writer, err error) {
	if code, message, ok := acpruntime.ClassifyError(err); ok {
		fmt.Fprintf(w, "%s: %s\n", code, message)
		return
	}
	fmt.Fprintf(w, "runner validation failed: %v\n", err)
}

func envInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func executionOverridesFromCLI(strategy string, maxParallel int, failurePolicy string) (acpruntime.ExecutionOverrides, error) {
	overrides := acpruntime.ExecutionOverrides{}
	if trimmed := strings.TrimSpace(strategy); trimmed != "" {
		normalized := strings.ToLower(trimmed)
		if normalized != acpruntime.ExecutionStrategySequential && normalized != acpruntime.ExecutionStrategyParallel {
			return acpruntime.ExecutionOverrides{}, fmt.Errorf("--execution-strategy must be one of: %s, %s", acpruntime.ExecutionStrategySequential, acpruntime.ExecutionStrategyParallel)
		}
		overrides.Strategy = &normalized
	}
	if maxParallel < 0 {
		return acpruntime.ExecutionOverrides{}, errors.New("--max-parallel-tasks must be > 0 when set")
	}
	if maxParallel > 0 {
		overrides.MaxParallel = &maxParallel
	}
	if trimmed := strings.TrimSpace(failurePolicy); trimmed != "" {
		normalized := strings.ToLower(trimmed)
		if normalized != acpruntime.ExecutionFailurePolicyFailFast && normalized != acpruntime.ExecutionFailurePolicyBestEffort {
			return acpruntime.ExecutionOverrides{}, fmt.Errorf("--failure-policy must be one of: %s, %s", acpruntime.ExecutionFailurePolicyFailFast, acpruntime.ExecutionFailurePolicyBestEffort)
		}
		overrides.FailurePolicy = &normalized
	}
	return overrides, nil
}

func ensureWorkspaceGitRepository(workspacePath string) error {
	gitDir := filepath.Join(workspacePath, ".git")
	_, err := os.Stat(gitDir)
	if err == nil {
		// Accept both ".git" directory and ".git" file (git worktree marker).
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("workspace.git.init.stat_failed: %w", err)
	}

	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("workspace.git.init.git_required: install git and ensure it is available in PATH: %w", err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = workspacePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("workspace.git.init.failed: git init failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func loadRepoSourcesFromFile(rawPath string) ([]workspace.RepoSource, error) {
	repos, _, err := loadRepoSourcesAndRuntimeFromFile(rawPath)
	return repos, err
}

func loadRepoSourcesAndRuntimeFromFile(rawPath string) ([]workspace.RepoSource, *workspace.RuntimeConfig, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return nil, nil, errors.New("repos file path is required")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve --repos-file: %w", err)
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read --repos-file %q: %w", absPath, err)
	}

	var envelope struct {
		Repos   []workspace.RepoSource   `yaml:"repos"`
		Runtime *workspace.RuntimeConfig `yaml:"runtime"`
	}
	if err := yaml.Unmarshal(content, &envelope); err != nil {
		return nil, nil, fmt.Errorf("parse --repos-file %q: %w", absPath, err)
	}
	repos := envelope.Repos
	runtimeConfig := cloneRuntimeConfig(envelope.Runtime)
	if len(repos) == 0 {
		var list []workspace.RepoSource
		if err := yaml.Unmarshal(content, &list); err != nil {
			return nil, nil, fmt.Errorf("parse --repos-file %q: expected YAML with repos[] or array of repo entries", absPath)
		}
		repos = list
		runtimeConfig = nil
	}
	if len(repos) == 0 {
		return nil, nil, fmt.Errorf("--repos-file %q contains no repos", absPath)
	}

	baseDir := filepath.Dir(absPath)
	normalized := make([]workspace.RepoSource, 0, len(repos))
	seenNames := map[string]struct{}{}
	for idx, repo := range repos {
		item, normalizeErr := normalizeRepoSource(repo, baseDir, idx)
		if normalizeErr != nil {
			return nil, nil, normalizeErr
		}
		if _, exists := seenNames[item.Name]; exists {
			return nil, nil, fmt.Errorf("--repos-file %q contains duplicate repo.name %q", absPath, item.Name)
		}
		seenNames[item.Name] = struct{}{}
		normalized = append(normalized, item)
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Name < normalized[j].Name
	})
	return normalized, runtimeConfig, nil
}

func cloneRuntimeConfig(input *workspace.RuntimeConfig) *workspace.RuntimeConfig {
	if input == nil {
		return nil
	}
	var clonedTimeouts *workspace.RuntimeTimeoutsConfig
	if input.Profile != nil && input.Profile.Timeouts != nil {
		clonedTimeouts = &workspace.RuntimeTimeoutsConfig{
			StepTimeoutSec:         cloneIntPointer(input.Profile.Timeouts.StepTimeoutSec),
			HeartbeatSec:           cloneIntPointer(input.Profile.Timeouts.HeartbeatSec),
			PipelineTimeoutSec:     cloneIntPointer(input.Profile.Timeouts.PipelineTimeoutSec),
			PipelineKillGraceSec:   cloneIntPointer(input.Profile.Timeouts.PipelineKillGraceSec),
			APIReadyTimeoutSec:     cloneIntPointer(input.Profile.Timeouts.APIReadyTimeoutSec),
			APIInitTimeoutSec:      cloneIntPointer(input.Profile.Timeouts.APIInitTimeoutSec),
			UIInitPollTimeoutSec:   cloneIntPointer(input.Profile.Timeouts.UIInitPollTimeoutSec),
			UICancelPollTimeoutSec: cloneIntPointer(input.Profile.Timeouts.UICancelPollTimeoutSec),
		}
		if clonedTimeouts.IsZero() {
			clonedTimeouts = nil
		}
	}
	var clonedExecution *workspace.RuntimeExecutionConfig
	if input.Profile != nil && input.Profile.Execution != nil {
		clonedExecution = &workspace.RuntimeExecutionConfig{
			Strategy:      strings.TrimSpace(input.Profile.Execution.Strategy),
			MaxParallel:   cloneIntPointer(input.Profile.Execution.MaxParallel),
			FailurePolicy: strings.TrimSpace(input.Profile.Execution.FailurePolicy),
		}
		if input.Profile.Execution.ShardDiscovery != nil {
			clonedExecution.ShardDiscovery = &workspace.RuntimeShardDiscoveryConfig{
				Mode: strings.TrimSpace(input.Profile.Execution.ShardDiscovery.Mode),
			}
		}
		if clonedExecution.IsZero() {
			clonedExecution = nil
		}
	}
	cloned := &workspace.RuntimeConfig{
		Profile: &workspace.RuntimeProfileConfig{
			Timeouts:  clonedTimeouts,
			Execution: clonedExecution,
		},
	}
	if cloned.IsZero() {
		return nil
	}
	return cloned
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func normalizeRepoSource(repo workspace.RepoSource, baseDir string, index int) (workspace.RepoSource, error) {
	repo.Name = strings.TrimSpace(repo.Name)
	repo.Path = strings.TrimSpace(repo.Path)
	repo.GitURL = strings.TrimSpace(repo.GitURL)
	repo.Ref = strings.TrimSpace(repo.Ref)

	label := fmt.Sprintf("repos[%d]", index)
	if repo.Name == "" {
		return workspace.RepoSource{}, fmt.Errorf("%s.name is required", label)
	}
	hasPath := repo.Path != ""
	hasGitURL := repo.GitURL != ""
	if hasPath == hasGitURL {
		return workspace.RepoSource{}, fmt.Errorf("%s must contain exactly one of path or git_url", label)
	}
	if hasPath && !filepath.IsAbs(repo.Path) {
		repo.Path = filepath.Join(baseDir, repo.Path)
	}
	if hasPath {
		resolvedPath, err := filepath.Abs(repo.Path)
		if err != nil {
			return workspace.RepoSource{}, fmt.Errorf("resolve %s.path: %w", label, err)
		}
		repo.Path = resolvedPath
	}

	return repo, nil
}
