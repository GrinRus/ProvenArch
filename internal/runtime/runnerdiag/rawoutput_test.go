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

func TestWriteParseFailureArtifactsWritesFilesAndMetadata(t *testing.T) {
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

	artifacts, err := WriteParseFailureArtifacts(task, acpruntime.ProviderQwenCode, "{\"bad\":true}", "stderr-text")
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

func TestWriteParseFailureArtifactsTruncatesLargeOutput(t *testing.T) {
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
	artifacts, err := WriteParseFailureArtifacts(task, acpruntime.ProviderClaudeCode, largeStdout, "")
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

func TestWritePromptArtifactsWritesPromptTaskAndMetadata(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	task := acpruntime.Task{
		TaskID:       "task-prompt",
		RunID:        "run-3",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"repo-a"},
		PathScopes:   []string{"services"},
		StartedAtUTC: time.Date(2026, 4, 11, 10, 10, 0, 0, time.UTC),
	}

	artifacts, err := WritePromptArtifacts(task, acpruntime.ProviderQwenCode, "prompt body", []byte(`{"task":"payload"}`), PromptArtifactsMetadata{
		Attempt:            "parse-retry",
		IncludeDirectories: []string{workspace, "/tmp/repo-a"},
		PromptPack: PromptPackMetadata{
			Name:         "collect-context",
			RelativePath: "skills/prompt-packs/collect-context.md",
			Source:       "workspace",
		},
	})
	if err != nil {
		t.Fatalf("write prompt artifacts: %v", err)
	}
	if _, err := os.Stat(artifacts.MetadataPath); err != nil {
		t.Fatalf("stat prompt metadata file: %v", err)
	}
	if _, err := os.Stat(artifacts.Prompt.Path); err != nil {
		t.Fatalf("stat prompt file: %v", err)
	}
	if _, err := os.Stat(artifacts.TaskPayload.Path); err != nil {
		t.Fatalf("stat task payload file: %v", err)
	}

	rawMeta, err := os.ReadFile(artifacts.MetadataPath)
	if err != nil {
		t.Fatalf("read metadata file: %v", err)
	}
	meta := map[string]any{}
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		t.Fatalf("parse metadata json: %v", err)
	}
	if got := strings.TrimSpace(meta["attempt"].(string)); got != "parse-retry" {
		t.Fatalf("unexpected attempt %q", got)
	}
	promptPack, ok := meta["prompt_pack"].(map[string]any)
	if !ok {
		t.Fatalf("expected prompt_pack block in metadata")
	}
	if got := strings.TrimSpace(promptPack["relative_path"].(string)); got != "skills/prompt-packs/collect-context.md" {
		t.Fatalf("unexpected prompt pack relative path %q", got)
	}
}

func TestWriteRuntimeFailureArtifactWritesStructuredFailureFile(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	task := acpruntime.Task{
		TaskID:       "task-failure",
		RunID:        "run-4",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"repo-a", "repo-b"},
		PathScopes:   []string{"services/api"},
		StartedAtUTC: time.Date(2026, 4, 21, 8, 0, 0, 0, time.UTC),
	}

	artifact, err := WriteRuntimeFailureArtifact(task, acpruntime.ProviderQwenCode, FailureArtifactInput{
		ErrorCode:       acpruntime.ErrorCodeRunnerParseFailed,
		FailureClass:    "runtime_artifact_contract",
		FailureSubclass: "manifest_invalid",
		ParseStage:      "artifact_repair.schema",
		ShortCause:      "collect artifacts remained invalid after one repair attempt",
	}, "stdout text", "stderr text")
	if err != nil {
		t.Fatalf("write runtime failure artifact: %v", err)
	}
	if artifact.RelativePath == "" || !strings.HasPrefix(artifact.RelativePath, "reports/taskruns/raw/") {
		t.Fatalf("unexpected failure artifact path: %q", artifact.RelativePath)
	}
	if _, err := os.Stat(artifact.Path); err != nil {
		t.Fatalf("stat runtime failure artifact: %v", err)
	}

	rawFailure, err := os.ReadFile(artifact.Path)
	if err != nil {
		t.Fatalf("read runtime failure artifact: %v", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(rawFailure, &payload); err != nil {
		t.Fatalf("parse runtime failure artifact: %v", err)
	}
	if got := strings.TrimSpace(payload["failure_class"].(string)); got != "runtime_artifact_contract" {
		t.Fatalf("unexpected failure class %q", got)
	}
	if got := strings.TrimSpace(payload["failure_subclass"].(string)); got != "manifest_invalid" {
		t.Fatalf("unexpected failure subclass %q", got)
	}
	if got := strings.TrimSpace(payload["parse_stage"].(string)); got != "artifact_repair.schema" {
		t.Fatalf("unexpected parse stage %q", got)
	}
	rawOutput, ok := payload["raw_output"].(map[string]any)
	if !ok {
		t.Fatalf("expected raw_output block")
	}
	relativeMeta := strings.TrimSpace(rawOutput["relative_metadata_path"].(string))
	if relativeMeta == "" || !strings.HasPrefix(relativeMeta, "reports/taskruns/raw/") {
		t.Fatalf("unexpected raw_output relative metadata path %q", relativeMeta)
	}
}
