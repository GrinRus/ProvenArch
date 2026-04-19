package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type runtimeDraftManifest struct {
	Version      int                  `json:"version"`
	RunID        string               `json:"run_id"`
	StepID       string               `json:"step_id"`
	StepContract string               `json:"step_contract"`
	AgentRole    string               `json:"agent_role"`
	Summary      string               `json:"summary,omitempty"`
	Outputs      []runtimeDraftOutput `json:"outputs"`
}

type runtimeDraftOutput struct {
	Path          string `json:"path"`
	CanonicalPath string `json:"canonical_path"`
	Kind          string `json:"kind,omitempty"`
	Title         string `json:"title,omitempty"`
}

func loadRuntimeDraftManifest(root string, filename string) (runtimeDraftManifest, []byte, error) {
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(root), filename))
	if err != nil {
		return runtimeDraftManifest{}, nil, fmt.Errorf("read runtime draft manifest: %w", err)
	}
	var manifest runtimeDraftManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return runtimeDraftManifest{}, nil, fmt.Errorf("parse runtime draft manifest: %w", err)
	}
	if err := validateRuntimeDraftManifest(manifest); err != nil {
		return runtimeDraftManifest{}, nil, err
	}
	return manifest, raw, nil
}

func validateRuntimeDraftManifest(manifest runtimeDraftManifest) error {
	if manifest.Version != 1 {
		return fmt.Errorf("runtime draft manifest version must be 1")
	}
	for idx, output := range manifest.Outputs {
		if err := validateRelativeDraftPath(output.Path); err != nil {
			return fmt.Errorf("runtime draft manifest outputs[%d].path: %w", idx, err)
		}
		if err := validateCanonicalDraftPath(output.CanonicalPath); err != nil {
			return fmt.Errorf("runtime draft manifest outputs[%d].canonical_path: %w", idx, err)
		}
	}
	return nil
}

func validateRelativeDraftPath(value string) error {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
	if clean == "." || clean == "" {
		return fmt.Errorf("must not be empty")
	}
	if path.IsAbs(clean) || strings.HasPrefix(clean, "../") || clean == ".." {
		return fmt.Errorf("must stay inside draft root")
	}
	return nil
}

func validateCanonicalDraftPath(value string) error {
	clean := filepath.ToSlash(path.Clean(strings.TrimSpace(value)))
	if clean == "." || clean == "" {
		return fmt.Errorf("must not be empty")
	}
	if path.IsAbs(clean) || strings.HasPrefix(clean, "../") || clean == ".." {
		return fmt.Errorf("must stay inside workspace")
	}
	return nil
}

func applyRuntimeDraftOutputs(
	target workspace.Root,
	draftRoot string,
	manifest runtimeDraftManifest,
	stagePrefix string,
	allow func(string) bool,
) ([]Artifact, error) {
	artifacts := make([]Artifact, 0, len(manifest.Outputs))
	for _, output := range manifest.Outputs {
		canonicalPath := filepath.ToSlash(path.Clean(strings.TrimSpace(output.CanonicalPath)))
		if !allow(canonicalPath) {
			return nil, fmt.Errorf("runtime draft output %q is outside the allowed publish surface", canonicalPath)
		}
		sourcePath := filepath.Join(filepath.Clean(draftRoot), filepath.Clean(strings.TrimSpace(output.Path)))
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("read runtime draft output %q: %w", output.Path, err)
		}
		if len(content) > 0 && content[len(content)-1] != '\n' {
			content = append(content, '\n')
		}
		if err := target.WriteFile(canonicalPath, content); err != nil {
			return nil, err
		}
		artifactPath := canonicalPath
		if strings.TrimSpace(stagePrefix) != "" {
			artifactPath = path.Join(stagePrefix, canonicalPath)
		}
		kind := strings.TrimSpace(output.Kind)
		if kind == "" {
			kind = inferDocflowArtifactKind(canonicalPath)
		}
		label := strings.TrimSpace(output.Title)
		if label == "" {
			label = path.Base(canonicalPath)
		}
		artifacts = append(artifacts, Artifact{
			Path:  artifactPath,
			Kind:  kind,
			Label: label,
		})
	}
	return artifacts, nil
}

func draftManifestHasPrefix(manifest *runtimeDraftManifest, prefix string) bool {
	if manifest == nil {
		return false
	}
	for _, output := range manifest.Outputs {
		if strings.HasPrefix(filepath.ToSlash(path.Clean(strings.TrimSpace(output.CanonicalPath))), prefix) {
			return true
		}
	}
	return false
}

func draftManifestCanonicalPathsWithPrefix(manifest *runtimeDraftManifest, prefix string) []string {
	if manifest == nil {
		return nil
	}
	paths := make([]string, 0, len(manifest.Outputs))
	for _, output := range manifest.Outputs {
		canonicalPath := filepath.ToSlash(path.Clean(strings.TrimSpace(output.CanonicalPath)))
		if strings.HasPrefix(canonicalPath, prefix) {
			paths = append(paths, canonicalPath)
		}
	}
	return paths
}
