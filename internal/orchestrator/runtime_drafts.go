package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
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
	if manifest.Version != 1 {
		return fmt.Errorf("runtime draft manifest version must be 1")
	}
	if len(manifest.Outputs) == 0 {
		return fmt.Errorf("runtime draft manifest outputs must not be empty")
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

func validateRuntimeDraftOutputsExist(draftRoot string, manifest runtimeDraftManifest) error {
	draftRoot = strings.TrimSpace(draftRoot)
	if draftRoot == "" {
		return fmt.Errorf("runtime draft root is empty")
	}
	for idx, output := range manifest.Outputs {
		relPath := filepath.Clean(strings.TrimSpace(output.Path))
		if relPath == "" || relPath == "." {
			return fmt.Errorf("runtime draft manifest outputs[%d].path: must not be empty", idx)
		}
		absPath := filepath.Join(filepath.Clean(draftRoot), relPath)
		info, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("runtime draft manifest outputs[%d].path: referenced draft file %q is unavailable: %w", idx, output.Path, err)
		}
		if info.IsDir() {
			return fmt.Errorf("runtime draft manifest outputs[%d].path: %q must point to a file", idx, output.Path)
		}
	}
	return nil
}

func requiredRuntimeDraftManifestFile(stepID string) string {
	switch strings.TrimSpace(stepID) {
	case "init.step0.constitution":
		return constitutionDraftManifestFile
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return asisDraftManifestFile
	case "init.step4.proposals", "refresh.step4.proposals":
		return proposalsDraftManifestFile
	default:
		return ""
	}
}

func validateRequiredRuntimeDraftArtifacts(task acpruntime.Task) (runtimeDraftManifest, []byte, error) {
	filename := requiredRuntimeDraftManifestFile(task.StepID)
	if filename == "" {
		return runtimeDraftManifest{}, nil, nil
	}
	if len(task.ExpectedArtifacts) > 0 {
		found := false
		for _, artifact := range task.ExpectedArtifacts {
			if strings.TrimSpace(artifact) == filename {
				found = true
				break
			}
		}
		if !found {
			return runtimeDraftManifest{}, nil, fmt.Errorf("runtime draft manifest %q is not declared in expected_artifacts", filename)
		}
	}
	manifest, raw, err := loadRuntimeDraftManifest(task.WriteRoot, filename)
	if err != nil {
		return runtimeDraftManifest{}, nil, err
	}
	if err := validateRuntimeDraftManifestForTask(task, manifest); err != nil {
		return runtimeDraftManifest{}, nil, err
	}
	if err := validateRuntimeDraftOutputsExist(task.DraftFinalRoot, manifest); err != nil {
		return runtimeDraftManifest{}, nil, err
	}
	return manifest, raw, nil
}

func validateRuntimeDraftManifestForTask(task acpruntime.Task, manifest runtimeDraftManifest) error {
	expectedStepID := strings.TrimSpace(task.StepID)
	if expectedStepID != "" && strings.TrimSpace(manifest.StepID) != expectedStepID {
		return fmt.Errorf("runtime draft manifest step_id must equal %q", expectedStepID)
	}
	expectedContract := strings.TrimSpace(runtimeStepContract(task.StepID))
	if expectedContract != "" && strings.TrimSpace(manifest.StepContract) != expectedContract {
		return fmt.Errorf("runtime draft manifest step_contract must equal %q", expectedContract)
	}
	expectedRunID := strings.TrimSpace(task.RunID)
	if expectedRunID != "" && strings.TrimSpace(manifest.RunID) != expectedRunID {
		return fmt.Errorf("runtime draft manifest run_id must equal %q", expectedRunID)
	}
	if strings.TrimSpace(manifest.AgentRole) == "" {
		return fmt.Errorf("runtime draft manifest agent_role must not be empty")
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
