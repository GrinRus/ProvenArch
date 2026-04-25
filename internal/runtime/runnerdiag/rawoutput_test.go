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

func TestWriteFailureArtifactsWithMetadataPersistsDiagnostics(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	task := acpruntime.Task{
		TaskID:       "task-diag",
		RunID:        "run-diag",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"repo-a"},
		StartedAtUTC: time.Date(2026, 4, 11, 10, 15, 0, 0, time.UTC),
	}

	artifacts, err := WriteFailureArtifactsWithMetadata(task, acpruntime.ProviderQwenCode, "stdout text", "stderr text", map[string]any{
		"stall_phase":         "post_write",
		"authored_file_count": 3,
	})
	if err != nil {
		t.Fatalf("write failure artifacts: %v", err)
	}

	rawMeta, err := os.ReadFile(artifacts.MetadataPath)
	if err != nil {
		t.Fatalf("read metadata file: %v", err)
	}
	meta := map[string]any{}
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		t.Fatalf("parse metadata json: %v", err)
	}
	if got := strings.TrimSpace(meta["command_family"].(string)); got != string(acpruntime.ProviderQwenCode) {
		t.Fatalf("unexpected command_family %q", got)
	}
	diagnostics, ok := meta["diagnostics"].(map[string]any)
	if !ok {
		t.Fatalf("expected diagnostics block in metadata: %#v", meta)
	}
	if got := strings.TrimSpace(diagnostics["stall_phase"].(string)); got != "post_write" {
		t.Fatalf("unexpected stall_phase %q", got)
	}
	if got := int(diagnostics["authored_file_count"].(float64)); got != 3 {
		t.Fatalf("unexpected authored_file_count %d", got)
	}
}

func TestWriteFailureArtifactsDoesNotOverwriteRapidFailures(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	task := acpruntime.Task{
		TaskID:       "task-rapid",
		RunID:        "run-rapid",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"repo-a"},
		StartedAtUTC: time.Date(2026, 4, 11, 10, 20, 0, 0, time.UTC),
	}

	first, err := WriteFailureArtifacts(task, acpruntime.ProviderQwenCode, "first stdout", "first stderr")
	if err != nil {
		t.Fatalf("write first failure artifacts: %v", err)
	}
	second, err := WriteFailureArtifacts(task, acpruntime.ProviderQwenCode, "second stdout", "second stderr")
	if err != nil {
		t.Fatalf("write second failure artifacts: %v", err)
	}

	if first.MetadataPath == second.MetadataPath {
		t.Fatalf("rapid failure metadata paths must be unique, got %q", first.MetadataPath)
	}
	if first.Stdout.Path == second.Stdout.Path {
		t.Fatalf("rapid failure stdout paths must be unique, got %q", first.Stdout.Path)
	}
	if got := mustReadRawOutput(t, first.Stdout.Path); got != "first stdout" {
		t.Fatalf("first stdout artifact was overwritten: %q", got)
	}
	if got := mustReadRawOutput(t, second.Stdout.Path); got != "second stdout" {
		t.Fatalf("second stdout artifact mismatch: %q", got)
	}
}

func mustReadRawOutput(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read raw output %s: %v", path, err)
	}
	return string(raw)
}
