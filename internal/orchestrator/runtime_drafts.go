package orchestrator

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type runtimeDraftManifest = runtimedrafts.Manifest
type runtimeDraftOutput = runtimedrafts.Output

func loadRuntimeDraftManifest(root string, filename string) (runtimeDraftManifest, []byte, error) {
	return runtimedrafts.Load(root, filename)
}

func loadValidatedRuntimeDraftManifest(root string, draftRoot string, filename string) (runtimeDraftManifest, []byte, error) {
	manifest, raw, err := loadRuntimeDraftManifest(root, filename)
	if err != nil {
		return runtimeDraftManifest{}, nil, err
	}
	if err := validateRuntimeDraftOutputsExist(draftRoot, manifest); err != nil {
		return runtimeDraftManifest{}, nil, err
	}
	return manifest, raw, nil
}

func validateRuntimeDraftManifest(manifest runtimeDraftManifest) error {
	return runtimedrafts.ValidateManifest(manifest)
}

func validateRuntimeDraftOutputsExist(draftRoot string, manifest runtimeDraftManifest) error {
	return runtimedrafts.ValidateOutputsExist(draftRoot, manifest)
}

func requiredRuntimeDraftManifestFile(stepID string) string {
	return runtimedrafts.ManifestFileForStep(stepID)
}

func validateRequiredRuntimeDraftArtifacts(task acpruntime.Task) (runtimeDraftManifest, []byte, error) {
	return runtimedrafts.ValidateRequiredManifest(
		task.WriteRoot,
		task.DraftFinalRoot,
		task.RunID,
		task.StepID,
		task.StepContract,
		task.ExpectedArtifacts,
	)
}

func validateRuntimeDraftManifestForTask(task acpruntime.Task, manifest runtimeDraftManifest) error {
	return runtimedrafts.ValidateManifestForTask(
		manifest,
		task.RunID,
		task.StepID,
		task.StepContract,
	)
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
