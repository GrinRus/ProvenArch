package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

const RuntimeModeFake = "fake"
const RuntimeModeHeadless = "headless"

// Provider identifies the live runtime implementation used for headless runs.
type Provider string

const (
	ProviderClaudeCode Provider = "claude-code"
	ProviderQwenCode   Provider = "qwen-code"
	ProviderCodexCode  Provider = "codex-code"
)

const RuntimeProviderEnv = "ACP_RUNTIME_PROVIDER"

type OutputStream string

const (
	OutputStreamStdout OutputStream = "stdout"
	OutputStreamStderr OutputStream = "stderr"
)

const (
	// RuntimeOutputStreamHardCapBytes is an internal safeguard against
	// unbounded live stream forwarding. It does not alter parser behavior.
	RuntimeOutputStreamHardCapBytes = 4 * 1024 * 1024
)

func ParseProvider(value string) (Provider, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case string(ProviderClaudeCode):
		return ProviderClaudeCode, nil
	case string(ProviderQwenCode):
		return ProviderQwenCode, nil
	case string(ProviderCodexCode):
		return ProviderCodexCode, nil
	default:
		return "", fmt.Errorf(
			"unsupported runtime provider %q (allowed: %s, %s, %s)",
			value,
			ProviderClaudeCode,
			ProviderQwenCode,
			ProviderCodexCode,
		)
	}
}

func ResolveProvider(cliValue string) (Provider, error) {
	provider, _, err := ResolveProviderWithSource(cliValue)
	return provider, err
}

func NormalizeMode(mode string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	if normalized == "" {
		normalized = RuntimeModeFake
	}
	switch normalized {
	case RuntimeModeFake, RuntimeModeHeadless:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported runtime %q (allowed: %s, %s)", mode, RuntimeModeFake, RuntimeModeHeadless)
	}
}

type ErrorCode string

const (
	ErrorCodeRunnerUnavailable ErrorCode = "runner_unavailable"
	ErrorCodeRuntimeContract   ErrorCode = "runtime_contract_failed"
	ErrorCodeRuntimeTimeout    ErrorCode = "runtime_timeout"
	ErrorCodeRunCanceled       ErrorCode = "run_canceled"
)

type RunnerError struct {
	Provider      Provider
	Code          ErrorCode
	Message       string
	Stdout        string
	Stderr        string
	RawOutputRefs contracts.RuntimeOutputRefs
	Cause         error
}

func (e RunnerError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return string(e.Code)
}

func (e RunnerError) Unwrap() error {
	return e.Cause
}

func WrapRunnerError(provider Provider, code ErrorCode, message string, cause error) error {
	return WrapRunnerErrorWithOutput(provider, code, message, "", "", cause)
}

func WrapRunnerErrorWithOutput(
	provider Provider,
	code ErrorCode,
	message string,
	stdout string,
	stderr string,
	cause error,
) error {
	return WrapRunnerErrorWithDiagnostics(provider, code, message, stdout, stderr, contracts.RuntimeOutputRefs{}, cause)
}

func WrapRunnerErrorWithDiagnostics(
	provider Provider,
	code ErrorCode,
	message string,
	stdout string,
	stderr string,
	rawOutputRefs contracts.RuntimeOutputRefs,
	cause error,
) error {
	return RunnerError{
		Provider:      provider,
		Code:          code,
		Message:       strings.TrimSpace(message),
		Stdout:        stdout,
		Stderr:        stderr,
		RawOutputRefs: rawOutputRefs,
		Cause:         cause,
	}
}

func ClassifyError(err error) (code string, message string, ok bool) {
	var runnerErr RunnerError
	if !errors.As(err, &runnerErr) {
		return "", "", false
	}
	return string(runnerErr.Code), runnerErr.Error(), true
}

type Task struct {
	TaskID            string
	RunID             string
	StepID            string
	ShardID           string
	DomainID          string
	Workspace         string
	ArtifactRoot      string
	WriteRoot         string
	DraftFinalRoot    string
	ReadContextRoots  []string
	AgentRole         string
	StepContract      string
	ExpectedArtifacts []string
	RepoScope         string
	RepoScopes        []string
	PathScopes        []string
	StartedAtUTC      time.Time
	OnOutput          func(OutputChunk)     `json:"-"`
	OnDiagnostic      func(DiagnosticEvent) `json:"-"`
}

type OutputChunk struct {
	Stream    OutputStream
	Text      string
	Truncated bool
}

type DiagnosticEvent struct {
	Message string
	Fields  map[string]any
}

type Result struct {
	Execution contracts.RuntimeExecution
	Stdout    string
	Stderr    string
}

type Runner interface {
	Run(context.Context, Task) (Result, error)
}

type PreflightRunner interface {
	Preflight(context.Context) error
}

func NewExecution(task Task, provider Provider, runtimeVersion string, status string, finishedAt time.Time, warnings []string) contracts.RuntimeExecution {
	startedAt := task.StartedAtUTC.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	doneAt := finishedAt.UTC()
	if doneAt.IsZero() {
		doneAt = time.Now().UTC()
	}
	return contracts.NormalizeRuntimeExecution(contracts.RuntimeExecution{
		Version:           1,
		TaskID:            strings.TrimSpace(task.TaskID),
		RunID:             strings.TrimSpace(task.RunID),
		StepID:            strings.TrimSpace(task.StepID),
		ShardID:           strings.TrimSpace(task.ShardID),
		DomainID:          strings.TrimSpace(task.DomainID),
		Provider:          string(provider),
		RuntimeVersion:    strings.TrimSpace(runtimeVersion),
		StartedAt:         startedAt.Format(time.RFC3339),
		FinishedAt:        doneAt.Format(time.RFC3339),
		RepoScope:         strings.TrimSpace(task.RepoScope),
		RepoScopes:        append([]string(nil), task.RepoScopes...),
		PathScopes:        append([]string(nil), task.PathScopes...),
		ArtifactRoot:      strings.TrimSpace(task.ArtifactRoot),
		WriteRoot:         strings.TrimSpace(task.WriteRoot),
		DraftFinalRoot:    strings.TrimSpace(task.DraftFinalRoot),
		Status:            strings.TrimSpace(status),
		RequiredArtifacts: append([]string(nil), task.ExpectedArtifacts...),
		Warnings:          append([]string(nil), warnings...),
	})
}
