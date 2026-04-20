package runtimedrafts

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	ConstitutionManifestFile = "constitution-draft.json"
	AsIsManifestFile         = "asis-draft-manifest.json"
	ProposalsManifestFile    = "proposals-draft-manifest.json"
)

type Manifest struct {
	Version      int      `json:"version"`
	RunID        string   `json:"run_id"`
	StepID       string   `json:"step_id"`
	StepContract string   `json:"step_contract"`
	AgentRole    string   `json:"agent_role"`
	Summary      string   `json:"summary,omitempty"`
	Outputs      []Output `json:"outputs"`
}

type Output struct {
	Path          string `json:"path"`
	CanonicalPath string `json:"canonical_path"`
	Kind          string `json:"kind,omitempty"`
	Title         string `json:"title,omitempty"`
}

func IsDraftStep(stepID string) bool {
	return strings.TrimSpace(ManifestFileForStep(stepID)) != ""
}

func ManifestFileForStep(stepID string) string {
	switch strings.TrimSpace(stepID) {
	case "init.step0.constitution":
		return ConstitutionManifestFile
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return AsIsManifestFile
	case "init.step4.proposals", "refresh.step4.proposals":
		return ProposalsManifestFile
	default:
		return ""
	}
}

func StepContractForStep(stepID string) string {
	switch strings.TrimSpace(stepID) {
	case "init.step0.constitution":
		return "constitution"
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return "as_is"
	case "init.step4.proposals", "refresh.step4.proposals":
		return "proposals"
	default:
		return ""
	}
}

func Load(root string, filename string) (Manifest, []byte, error) {
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(root), filename))
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("read runtime draft manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, nil, fmt.Errorf("parse runtime draft manifest: %w", err)
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, nil, err
	}
	return manifest, raw, nil
}

func ValidateRequiredManifest(
	root string,
	draftRoot string,
	runID string,
	stepID string,
	stepContract string,
	expectedArtifacts []string,
) (Manifest, []byte, error) {
	filename := ManifestFileForStep(stepID)
	if filename == "" {
		return Manifest{}, nil, nil
	}
	if len(expectedArtifacts) > 0 {
		found := false
		for _, artifact := range expectedArtifacts {
			if strings.TrimSpace(artifact) == filename {
				found = true
				break
			}
		}
		if !found {
			return Manifest{}, nil, fmt.Errorf("runtime draft manifest %q is not declared in expected_artifacts", filename)
		}
	}
	manifest, raw, err := Load(root, filename)
	if err != nil {
		return Manifest{}, nil, err
	}
	if err := ValidateManifestForTask(manifest, runID, stepID, stepContract); err != nil {
		return Manifest{}, nil, err
	}
	if err := ValidateOutputsExist(draftRoot, manifest); err != nil {
		return Manifest{}, nil, err
	}
	return manifest, raw, nil
}

func ValidateManifest(manifest Manifest) error {
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

func ValidateManifestForTask(manifest Manifest, runID string, stepID string, stepContract string) error {
	expectedStepID := strings.TrimSpace(stepID)
	if expectedStepID != "" && strings.TrimSpace(manifest.StepID) != expectedStepID {
		return fmt.Errorf("runtime draft manifest step_id must equal %q", expectedStepID)
	}
	expectedContract := strings.TrimSpace(stepContract)
	if expectedContract == "" {
		expectedContract = strings.TrimSpace(StepContractForStep(stepID))
	}
	if expectedContract != "" && strings.TrimSpace(manifest.StepContract) != expectedContract {
		return fmt.Errorf("runtime draft manifest step_contract must equal %q", expectedContract)
	}
	expectedRunID := strings.TrimSpace(runID)
	if expectedRunID != "" && strings.TrimSpace(manifest.RunID) != expectedRunID {
		return fmt.Errorf("runtime draft manifest run_id must equal %q", expectedRunID)
	}
	if strings.TrimSpace(manifest.AgentRole) == "" {
		return fmt.Errorf("runtime draft manifest agent_role must not be empty")
	}
	return nil
}

func ValidateOutputsExist(draftRoot string, manifest Manifest) error {
	draftRoot = strings.TrimSpace(draftRoot)
	if draftRoot == "" {
		return fmt.Errorf("runtime draft root is empty")
	}
	cleanDraftRoot := filepath.Clean(draftRoot)
	for idx, output := range manifest.Outputs {
		relPath := filepath.Clean(strings.TrimSpace(output.Path))
		if relPath == "" || relPath == "." {
			return fmt.Errorf("runtime draft manifest outputs[%d].path: must not be empty", idx)
		}
		absPath := filepath.Join(cleanDraftRoot, relPath)
		if err := reconcileCanonicalDraftOutput(cleanDraftRoot, absPath, output); err != nil {
			return fmt.Errorf("runtime draft manifest outputs[%d].path: %w", idx, err)
		}
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

func reconcileCanonicalDraftOutput(draftRoot string, expectedPath string, output Output) error {
	if _, err := os.Stat(expectedPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	canonicalRel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(output.CanonicalPath)))
	if canonicalRel == "" || canonicalRel == "." {
		return nil
	}
	canonicalPath := filepath.Join(filepath.Clean(draftRoot), canonicalRel)
	if filepath.Clean(canonicalPath) == filepath.Clean(expectedPath) {
		return nil
	}
	info, err := os.Stat(canonicalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("canonical draft fallback %q must point to a file", output.CanonicalPath)
	}
	if err := os.MkdirAll(filepath.Dir(expectedPath), 0o755); err != nil {
		return err
	}
	source, err := os.Open(canonicalPath)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(expectedPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer target.Close()
	if _, err := io.Copy(target, source); err != nil {
		return err
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
