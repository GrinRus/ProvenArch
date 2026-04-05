package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/api"
	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	"github.com/GrinRus/ProvenArch/internal/qa"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/workspace"
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
	dryRun := fs.Bool("dry-run", false, "validate workspace and server wiring without starting listener")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: acp serve --workspace <abs-path> [--runtime fake|headless] [--listen 127.0.0.1:8080] [--dry-run]")
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
	report := ws.Validate(context.Background(), workspace.ValidateOptions{
		ResolveRepos: true,
		FetchGit:     false,
		VerifyRefs:   true,
	})
	if !report.OK {
		printValidationReport(stderr, report)
		return exitCodeValidation
	}

	runner, err := buildRunner(*runtimeMode)
	if err != nil {
		fmt.Fprintf(stderr, "runtime validation failed: %v\n", err)
		return exitCodeValidation
	}
	service := orchestrator.NewService(orchestrator.WithRunner(runner))
	if err := service.ValidateRuntime(context.Background()); err != nil {
		printRunnerError(stderr, err)
		return exitCodeValidation
	}
	server := api.NewServer(ws, service)
	if *dryRun {
		fmt.Fprintf(stdout, "workspace validated at %s\n", ws.Path)
		fmt.Fprintf(stdout, "server configured for %s\n", *listenAddress)
		fmt.Fprintf(stdout, "runtime mode: %s\n", strings.TrimSpace(strings.ToLower(*runtimeMode)))
		return exitCodeOK
	}

	fmt.Fprintf(stdout, "starting ACP API server on %s for workspace %s\n", *listenAddress, ws.Path)
	if err := server.Serve(context.Background(), *listenAddress); err != nil {
		fmt.Fprintf(stderr, "serve failed: %v\n", err)
		return exitCodeValidation
	}

	return exitCodeOK
}

func runPipeline(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	workspacePath := fs.String("workspace", "", "absolute path to arch-workspace")
	pipelineName := fs.String("pipeline", "", "pipeline to run: init or refresh")
	runtimeMode := fs.String("runtime", "fake", "runtime mode: fake or headless")
	nonInteractive := fs.Bool("non-interactive", false, "disable interactive prompts")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--non-interactive]")
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

	runner, err := buildRunner(*runtimeMode)
	if err != nil {
		fmt.Fprintf(stderr, "runtime validation failed: %v\n", err)
		return exitCodeValidation
	}
	service := orchestrator.NewService(orchestrator.WithRunner(runner))
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
	fmt.Fprintln(w, "  acp serve --workspace <abs-path> [--runtime fake|headless]")
	fmt.Fprintln(w, "  acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--non-interactive]")
	fmt.Fprintln(w, "  acp qa --workspace <abs-path> --question \"<text>\"")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  serve   validate workspace path and start local API+UI service")
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

func buildRunner(runtimeMode string) (claudecode.Runner, error) {
	mode := strings.TrimSpace(strings.ToLower(runtimeMode))
	if mode == "" {
		mode = "fake"
	}
	switch mode {
	case "fake":
		return claudecode.FakeRunner{}, nil
	case "headless":
		return claudecode.HeadlessRunner{}, nil
	default:
		return nil, fmt.Errorf("unsupported runtime %q (allowed: fake, headless)", runtimeMode)
	}
}

func printRunnerError(w io.Writer, err error) {
	if code, message, ok := claudecode.ClassifyError(err); ok {
		fmt.Fprintf(w, "%s: %s\n", code, message)
		return
	}
	fmt.Fprintf(w, "runner validation failed: %v\n", err)
}
