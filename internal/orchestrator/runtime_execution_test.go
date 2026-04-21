package orchestrator

import (
	"errors"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestRuntimeExecutionFromFailurePreservesRawOutputRefsAndTimeoutStatus(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:            "task-1",
		RunID:             "run-1",
		StepID:            "refresh.step1.collect",
		ShardID:           "payments",
		ArtifactRoot:      "reports/taskruns/run-1/staging/shards/payments",
		WriteRoot:         "/tmp/write-root",
		DraftFinalRoot:    "/tmp/draft-root",
		RepoScopes:        []string{"payments-service"},
		PathScopes:        []string{"src"},
		ExpectedArtifacts: []string{"shard-pack-manifest.json"},
		StartedAtUTC:      time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC),
	}
	rawRefs := contracts.RuntimeOutputRefs{
		Stdout:   "reports/taskruns/raw/stdout.log",
		Stderr:   "reports/taskruns/raw/stderr.log",
		Metadata: "reports/taskruns/raw/meta.json",
	}
	err := acpruntime.WrapRunnerErrorWithDiagnostics(
		acpruntime.ProviderQwenCode,
		acpruntime.ErrorCodeRuntimeTimeout,
		"runtime timeout",
		"",
		"",
		rawRefs,
		errors.New("deadline exceeded"),
	)

	execution, ok := runtimeExecutionFromFailure(task, acpruntime.ProviderClaudeCode, err, time.Date(2026, 4, 21, 12, 0, 5, 0, time.UTC))
	if !ok {
		t.Fatalf("expected runtime execution from failure")
	}
	if execution.Status != "timeout" {
		t.Fatalf("expected timeout status, got %q", execution.Status)
	}
	if execution.Provider != string(acpruntime.ProviderQwenCode) {
		t.Fatalf("expected provider %q, got %q", acpruntime.ProviderQwenCode, execution.Provider)
	}
	if execution.RawOutputRefs != rawRefs {
		t.Fatalf("expected raw output refs %#v, got %#v", rawRefs, execution.RawOutputRefs)
	}
	if len(execution.RequiredArtifacts) != 1 || execution.RequiredArtifacts[0] != "shard-pack-manifest.json" {
		t.Fatalf("expected required artifacts to be preserved, got %#v", execution.RequiredArtifacts)
	}
}

func TestRuntimeExecutionFromFailureReturnsFalseForGenericError(t *testing.T) {
	t.Parallel()

	if _, ok := runtimeExecutionFromFailure(acpruntime.Task{}, acpruntime.ProviderClaudeCode, errors.New("boom"), time.Now().UTC()); ok {
		t.Fatalf("expected generic error to be ignored")
	}
}
