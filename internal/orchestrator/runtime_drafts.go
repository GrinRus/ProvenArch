package orchestrator

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func applyRuntimeDraftOutputs(
	target workspace.Root,
	draftRoot string,
	manifest runtimedrafts.Manifest,
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

func draftManifestHasPrefix(manifest *runtimedrafts.Manifest, prefix string) bool {
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
