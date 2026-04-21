package runnerdiag

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestWriteFailureArtifactsWritesFilesAndMetadata(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	task := acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"repo-a"},
		StartedAtUTC: time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC),
	}

	artifacts, err := WriteFailureArtifacts(task, acpruntime.ProviderQwenCode, "{\"bad\":true}", "stderr-text")
	if err != nil {
		t.Fatalf("write parse-failure artifacts: %v", err)
	}
	if artifacts.RelativeMetadataPath == "" || !strings.HasPrefix(artifacts.RelativeMetadataPath, "reports/taskruns/raw/") {
		t.Fatalf("unexpected relative metadata path: %q", artifacts.RelativeMetadataPath)
	}
	if _, err := os.Stat(artifacts.MetadataPath); err != nil {
		t.Fatalf("stat metadata file: %v", err)
	}
	if _, err := os.Stat(artifacts.Stdout.Path); err != nil {
		t.Fatalf("stat stdout artifact: %v", err)
	}
	if _, err := os.Stat(artifacts.Stderr.Path); err != nil {
		t.Fatalf("stat stderr artifact: %v", err)
	}

	rawMeta, err := os.ReadFile(artifacts.MetadataPath)
	if err != nil {
		t.Fatalf("read metadata file: %v", err)
	}
	meta := map[string]any{}
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		t.Fatalf("parse metadata json: %v", err)
	}
	stdoutMeta, ok := meta["stdout"].(map[string]any)
	if !ok {
		t.Fatalf("metadata stdout block is missing")
	}
	if strings.TrimSpace(stdoutMeta["sha256"].(string)) == "" {
		t.Fatalf("metadata stdout sha256 is empty")
	}
}

func TestWriteFailureArtifactsTruncatesLargeOutput(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	task := acpruntime.Task{
		TaskID:       "task-2",
		RunID:        "run-2",
		StepID:       "refresh.step3.findings",
		Workspace:    workspace,
		RepoScopes:   []string{"repo-a"},
		StartedAtUTC: time.Date(2026, 4, 11, 10, 5, 0, 0, time.UTC),
	}

	largeStdout := strings.Repeat("x", maxStoredOutputBytes+2048)
	artifacts, err := WriteFailureArtifacts(task, acpruntime.ProviderClaudeCode, largeStdout, "")
	if err != nil {
		t.Fatalf("write parse-failure artifacts: %v", err)
	}
	if !artifacts.Stdout.Truncated {
		t.Fatalf("expected stdout artifact to be marked truncated")
	}
	if artifacts.Stdout.Bytes <= maxStoredOutputBytes {
		t.Fatalf("expected original stdout bytes to exceed truncation threshold, got %d", artifacts.Stdout.Bytes)
	}

	storedBytes, err := os.ReadFile(filepath.Clean(artifacts.Stdout.Path))
	if err != nil {
		t.Fatalf("read stdout artifact: %v", err)
	}
	if len(storedBytes) > maxStoredOutputBytes+128 {
		t.Fatalf("stored stdout artifact is unexpectedly large: %d bytes", len(storedBytes))
	}
	if !strings.Contains(string(storedBytes), "...[truncated ") {
		t.Fatalf("stored stdout artifact must include truncation marker")
	}
}

func TestWriteFailureArtifactsMetadataIncludesTaskScopes(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	task := acpruntime.Task{
		TaskID:       "task-scopes",
		RunID:        "run-3",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"repo-a"},
		PathScopes:   []string{"services"},
		StartedAtUTC: time.Date(2026, 4, 11, 10, 10, 0, 0, time.UTC),
	}

	artifacts, err := WriteFailureArtifacts(task, acpruntime.ProviderQwenCode, "stdout text", "stderr text")
	if err != nil {
		t.Fatalf("write failure artifacts: %v", err)
	}
	if _, err := os.Stat(artifacts.MetadataPath); err != nil {
		t.Fatalf("stat metadata file: %v", err)
	}

	rawMeta, err := os.ReadFile(artifacts.MetadataPath)
	if err != nil {
		t.Fatalf("read metadata file: %v", err)
	}
	meta := map[string]any{}
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		t.Fatalf("parse metadata json: %v", err)
	}
	taskMeta, ok := meta["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected task block in metadata")
	}
	repoScopes, ok := taskMeta["repo_scopes"].([]any)
	if !ok || len(repoScopes) != 1 || strings.TrimSpace(repoScopes[0].(string)) != "repo-a" {
		t.Fatalf("unexpected repo_scopes payload %#v", taskMeta["repo_scopes"])
	}
	pathScopes, ok := taskMeta["path_scopes"].([]any)
	if !ok || len(pathScopes) != 1 || strings.TrimSpace(pathScopes[0].(string)) != "services" {
		t.Fatalf("unexpected path_scopes payload %#v", taskMeta["path_scopes"])
	}
}
