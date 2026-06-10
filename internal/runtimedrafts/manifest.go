package runtimedrafts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
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
	manifest, err := decodeManifest(raw)
	if err != nil {
		return Manifest{}, nil, err
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
	if err := ValidateOutputContent(draftRoot, manifest, stepID); err != nil {
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
	if err := validateStepSpecificOutputs(manifest, stepID); err != nil {
		return err
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

func ValidateOutputContent(draftRoot string, manifest Manifest, stepID string) error {
	switch strings.TrimSpace(stepID) {
	case "init.step0.constitution", "init.step2.asis_docs", "refresh.step2.asis_docs", "init.step4.proposals", "refresh.step4.proposals":
	default:
		return nil
	}

	cleanDraftRoot := filepath.Clean(strings.TrimSpace(draftRoot))
	problems := []string{}
	for idx, output := range manifest.Outputs {
		relPath := filepath.Clean(strings.TrimSpace(output.Path))
		if relPath == "" || relPath == "." {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(cleanDraftRoot, relPath))
		if err != nil {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q cannot be read for content validation: %v", idx, output.Path, err))
			continue
		}
		if runtimeDraftTextBootstrapOnly(string(raw)) {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q references bootstrap-only placeholder draft content", idx, output.Path))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("runtime draft manifest outputs are invalid: %s", strings.Join(problems, "; "))
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

func decodeManifest(raw []byte) (Manifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse runtime draft manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return Manifest{}, fmt.Errorf("parse runtime draft manifest: unexpected trailing JSON payload")
	} else if !errors.Is(err, io.EOF) {
		return Manifest{}, fmt.Errorf("parse runtime draft manifest: %w", err)
	}
	return manifest, nil
}

func validateStepSpecificOutputs(manifest Manifest, stepID string) error {
	switch strings.TrimSpace(stepID) {
	case "init.step0.constitution":
		return validateConstitutionDraftOutputs(manifest.Outputs)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return validateAsIsDraftOutputs(manifest.Outputs)
	case "init.step4.proposals", "refresh.step4.proposals":
		return validateProposalsDraftOutputs(manifest.Outputs)
	default:
		return nil
	}
}

func validateConstitutionDraftOutputs(outputs []Output) error {
	required := map[string]string{
		"charter/overview.md":   "charter-overview.md",
		"skills/subagents.yaml": "baseline-subagents.yaml",
	}
	byCanonicalPath := make(map[string]Output, len(outputs))
	problems := []string{}

	for idx, output := range outputs {
		canonicalPath := filepath.ToSlash(path.Clean(strings.TrimSpace(output.CanonicalPath)))
		if canonicalPath == "" || canonicalPath == "." {
			continue
		}
		if _, exists := byCanonicalPath[canonicalPath]; exists {
			problems = append(problems, fmt.Sprintf("outputs[%d].canonical_path %q must be unique", idx, canonicalPath))
			continue
		}
		byCanonicalPath[canonicalPath] = output
	}

	for canonicalPath, requiredPath := range required {
		output, ok := byCanonicalPath[canonicalPath]
		if !ok {
			problems = append(problems, fmt.Sprintf("outputs must include %q", canonicalPath))
			continue
		}
		actualPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(output.Path)))
		if actualPath != requiredPath {
			problems = append(problems, fmt.Sprintf("output %q must use path %q", canonicalPath, requiredPath))
		}
	}

	for canonicalPath := range byCanonicalPath {
		if _, ok := required[canonicalPath]; !ok {
			problems = append(problems, fmt.Sprintf("output %q is outside the allowed constitution publish surface", canonicalPath))
		}
	}

	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("runtime draft manifest outputs are invalid: %s", strings.Join(problems, "; "))
}

func validateAsIsDraftOutputs(outputs []Output) error {
	required := map[string]string{
		"reports/as-is/overview.md":                  "overview.md",
		"reports/coverage/summary.md":                "summary.md",
		"reports/agent-outputs/architect/summary.md": "architect-summary.md",
	}
	byCanonicalPath := make(map[string]Output, len(outputs))
	problems := []string{}

	for idx, output := range outputs {
		canonicalPath := filepath.ToSlash(path.Clean(strings.TrimSpace(output.CanonicalPath)))
		if canonicalPath == "" || canonicalPath == "." {
			continue
		}
		if _, exists := byCanonicalPath[canonicalPath]; exists {
			problems = append(problems, fmt.Sprintf("outputs[%d].canonical_path %q must be unique", idx, canonicalPath))
			continue
		}
		byCanonicalPath[canonicalPath] = output
	}

	for canonicalPath, requiredPath := range required {
		output, ok := byCanonicalPath[canonicalPath]
		if !ok {
			problems = append(problems, fmt.Sprintf("outputs must include %q", canonicalPath))
			continue
		}
		actualPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(output.Path)))
		if actualPath != requiredPath {
			problems = append(problems, fmt.Sprintf("output %q must use path %q", canonicalPath, requiredPath))
		}
	}

	for canonicalPath := range byCanonicalPath {
		if _, ok := required[canonicalPath]; ok {
			continue
		}
		if !isAllowedAdditionalAsIsCanonicalPath(canonicalPath) {
			problems = append(problems, fmt.Sprintf("output %q is outside the allowed as-is publish surface", canonicalPath))
		}
	}

	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("runtime draft manifest outputs are invalid: %s", strings.Join(problems, "; "))
}

func isAllowedAdditionalAsIsCanonicalPath(canonicalPath string) bool {
	parts := strings.Split(filepath.ToSlash(path.Clean(strings.TrimSpace(canonicalPath))), "/")
	return len(parts) == 4 &&
		parts[0] == "reports" &&
		parts[1] == "as-is" &&
		strings.TrimSpace(parts[2]) != "" &&
		parts[3] == "overview.md"
}

func validateProposalsDraftOutputs(outputs []Output) error {
	seen := make(map[string]struct{}, len(outputs))
	problems := []string{}

	for idx, output := range outputs {
		canonicalPath := filepath.ToSlash(path.Clean(strings.TrimSpace(output.CanonicalPath)))
		if canonicalPath == "" || canonicalPath == "." {
			continue
		}
		if _, exists := seen[canonicalPath]; exists {
			problems = append(problems, fmt.Sprintf("outputs[%d].canonical_path %q must be unique", idx, canonicalPath))
			continue
		}
		seen[canonicalPath] = struct{}{}
		if !isAllowedProposalsCanonicalPath(canonicalPath) {
			problems = append(problems, fmt.Sprintf("output %q is outside the allowed proposals publish surface", canonicalPath))
		}
	}

	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("runtime draft manifest outputs are invalid: %s", strings.Join(problems, "; "))
}

func isAllowedProposalsCanonicalPath(canonicalPath string) bool {
	clean := filepath.ToSlash(path.Clean(strings.TrimSpace(canonicalPath)))
	return strings.HasPrefix(clean, "proposals/") || strings.HasPrefix(clean, "reports/changelog/")
}

func runtimeDraftTextBootstrapOnly(text string) bool {
	lower := strings.ToLower(text)
	hardMarkers := []string{
		"provider wrote this draft artifact",
		"drafted required runtime artifacts",
		"draft surface initialized for the scoped repository analysis",
		"final content must stay tied to collected shard evidence and validator output",
		"runtime proposal surface initialized for this analysis run",
		"runtime draft recovery initialized",
		"draft recovery initialized",
		"treat this as diagnostic evidence until",
	}
	for _, marker := range hardMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}

	softMarkers := []string{
		"current run evidence should be reviewed before promotion",
		"owner mappings and unresolved coverage gaps remain the first follow-up surfaces",
		"promote only recommendations that cite collected shard manifests",
		"changes must remain traceable to collected evidence",
		"promote only after artifact validation succeeds",
	}
	if runtimeDraftTextHasEvidenceMarker(lower) {
		return false
	}
	for _, marker := range softMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func runtimeDraftTextHasEvidenceMarker(lower string) bool {
	markers := []string{
		"reports/findings/",
		"reports/coverage/",
		"reports/as-is/",
		"validator-verdict.json",
		"final-run-index.json",
		"finding.",
		"question.",
		"cite.",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
