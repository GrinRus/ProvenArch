package runnerdiag

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

const maxStoredOutputBytes = 256 * 1024

var invalidPathChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type ArtifactFile struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
	Bytes        int    `json:"bytes"`
	StoredBytes  int    `json:"stored_bytes"`
	SHA256       string `json:"sha256"`
	Truncated    bool   `json:"truncated"`
}

type ParseFailureArtifacts struct {
	Directory            string       `json:"directory"`
	RelativeDirectory    string       `json:"relative_directory"`
	Stdout               ArtifactFile `json:"stdout"`
	Stderr               ArtifactFile `json:"stderr"`
	MetadataPath         string       `json:"metadata_path"`
	RelativeMetadataPath string       `json:"relative_metadata_path"`
}

func WriteParseFailureArtifacts(task acpruntime.Task, provider acpruntime.Provider, stdout string, stderr string) (ParseFailureArtifacts, error) {
	workspace := strings.TrimSpace(task.Workspace)
	if workspace == "" {
		return ParseFailureArtifacts{}, fmt.Errorf("workspace is empty")
	}
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return ParseFailureArtifacts{}, fmt.Errorf("resolve workspace path: %w", err)
	}
	rawDir := filepath.Join(absWorkspace, "reports", "taskruns", "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return ParseFailureArtifacts{}, fmt.Errorf("mkdir raw output dir: %w", err)
	}

	stamp := time.Now().UTC().Format("20060102T150405Z")
	base := safePathPart(task.RunID) + "-" + safePathPart(task.StepID) + "-" + safePathPart(task.TaskID) + "-" + safePathPart(string(provider)) + "-" + stamp

	stdoutFile := filepath.Join(rawDir, base+"-stdout.log")
	stderrFile := filepath.Join(rawDir, base+"-stderr.log")
	metaFile := filepath.Join(rawDir, base+"-meta.json")

	stdoutArtifact, err := writeBoundedArtifactFile(stdoutFile, []byte(stdout))
	if err != nil {
		return ParseFailureArtifacts{}, err
	}
	stderrArtifact, err := writeBoundedArtifactFile(stderrFile, []byte(stderr))
	if err != nil {
		return ParseFailureArtifacts{}, err
	}

	stdoutArtifact.Path = stdoutFile
	stderrArtifact.Path = stderrFile
	stdoutArtifact.RelativePath = toRelativePath(absWorkspace, stdoutFile)
	stderrArtifact.RelativePath = toRelativePath(absWorkspace, stderrFile)

	artifacts := ParseFailureArtifacts{
		Directory:            rawDir,
		RelativeDirectory:    toRelativePath(absWorkspace, rawDir),
		Stdout:               stdoutArtifact,
		Stderr:               stderrArtifact,
		MetadataPath:         metaFile,
		RelativeMetadataPath: toRelativePath(absWorkspace, metaFile),
	}

	metaPayload := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"provider":     provider,
		"task": map[string]any{
			"task_id":     task.TaskID,
			"run_id":      task.RunID,
			"step_id":     task.StepID,
			"workspace":   absWorkspace,
			"repo_scopes": append([]string(nil), task.RepoScopes...),
		},
		"stdout": map[string]any{
			"path":          stdoutArtifact.Path,
			"relative_path": stdoutArtifact.RelativePath,
			"bytes":         stdoutArtifact.Bytes,
			"stored_bytes":  stdoutArtifact.StoredBytes,
			"sha256":        stdoutArtifact.SHA256,
			"truncated":     stdoutArtifact.Truncated,
		},
		"stderr": map[string]any{
			"path":          stderrArtifact.Path,
			"relative_path": stderrArtifact.RelativePath,
			"bytes":         stderrArtifact.Bytes,
			"stored_bytes":  stderrArtifact.StoredBytes,
			"sha256":        stderrArtifact.SHA256,
			"truncated":     stderrArtifact.Truncated,
		},
	}
	rawMeta, err := json.MarshalIndent(metaPayload, "", "  ")
	if err != nil {
		return ParseFailureArtifacts{}, fmt.Errorf("marshal parse-failure metadata: %w", err)
	}
	if err := os.WriteFile(metaFile, append(rawMeta, '\n'), 0o644); err != nil {
		return ParseFailureArtifacts{}, fmt.Errorf("write parse-failure metadata: %w", err)
	}

	return artifacts, nil
}

func writeBoundedArtifactFile(path string, data []byte) (ArtifactFile, error) {
	summary := ArtifactFile{
		Bytes: len(data),
	}
	hash := sha256.Sum256(data)
	summary.SHA256 = hex.EncodeToString(hash[:])

	stored := data
	if len(stored) > maxStoredOutputBytes {
		trailer := fmt.Sprintf("\n...[truncated %d bytes]\n", len(stored)-maxStoredOutputBytes)
		stored = append(append([]byte{}, stored[:maxStoredOutputBytes]...), []byte(trailer)...)
		summary.Truncated = true
	}
	summary.StoredBytes = len(stored)
	if err := os.WriteFile(path, stored, 0o644); err != nil {
		return ArtifactFile{}, fmt.Errorf("write raw output artifact %s: %w", path, err)
	}
	return summary, nil
}

func toRelativePath(workspace string, absPath string) string {
	relative, err := filepath.Rel(workspace, absPath)
	if err != nil {
		return absPath
	}
	return filepath.ToSlash(relative)
}

func safePathPart(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return "unknown"
	}
	cleaned = strings.ReplaceAll(cleaned, string(filepath.Separator), "-")
	cleaned = invalidPathChars.ReplaceAllString(cleaned, "-")
	cleaned = strings.Trim(cleaned, "-")
	if cleaned == "" {
		return "unknown"
	}
	if len(cleaned) > 96 {
		cleaned = cleaned[:96]
	}
	return cleaned
}
