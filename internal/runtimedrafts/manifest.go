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
	"regexp"
	"sort"
	"strings"
)

const (
	ConstitutionManifestFile = "constitution-draft.json"
	AsIsManifestFile         = "asis-draft-manifest.json"
	ProposalsManifestFile    = "proposals-draft-manifest.json"
)

var (
	finalIndexDocumentCountPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(\d+)\s+(?:top-level\s+)?canonical document(?:s| entries)?\b`),
		regexp.MustCompile(`(?i)\b(\d+)\s+observed document entries\b`),
		regexp.MustCompile(`(?i)\b(\d+)\s+indexed document(?:s)?\b`),
	}
	citationIndexCountPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(\d+)\s+citation(?:s| entries)?\b`),
	}
)

type Manifest struct {
	Version      int      `json:"version"`
	RunID        string   `json:"run_id"`
	StepID       string   `json:"step_id"`
	StepContract string   `json:"step_contract"`
	AgentRole    string   `json:"agent_role"`
	Summary      string   `json:"summary,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
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
	if err := ValidateOutputContent(draftRoot, manifest, stepID, runID); err != nil {
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

func ValidateOutputContent(draftRoot string, manifest Manifest, stepID string, runID string) error {
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
		text := string(raw)
		if runtimeDraftTextBootstrapOnly(text) {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q references bootstrap-only placeholder draft content", idx, output.Path))
		}
		if strings.TrimSpace(stepID) == "init.step0.constitution" && runtimeDraftStep0TextHasDownstreamEvidenceLeak(text) {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q mentions downstream or runtime-only evidence in step0 constitution content", idx, output.Path))
		}
		if foreignRunID := runtimeDraftTextForeignRunID(text, runID); foreignRunID != "" {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q references foreign taskrun %q instead of current run_id %q", idx, output.Path, foreignRunID, strings.TrimSpace(runID)))
		}
		if runtimeDraftTextHasGenericShardGapWording(text) {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q uses generic conditional shard-gap wording instead of exact current-run shard status", idx, output.Path))
		}
		if runtimeDraftTextHasStaleIndexAvailabilityClaim(text) {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q claims current-run final/citation indexes are unavailable instead of omitting downstream index status", idx, output.Path))
		}
		if runtimeDraftTextHasStaleIndexZeroClaim(text) {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q claims current-run final-run-index has zero observed documents without validated zero-document evidence", idx, output.Path))
		}
		if runtimeDraftTextHasRawStructuredEvidenceDump(text) {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q includes raw structured evidence dumps instead of readable summaries", idx, output.Path))
		}
		if runtimeDraftTextHasMetadataOnlyEvidenceBullet(text) {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q includes metadata-only JSON keys as evidence instead of architecture or proposal signals", idx, output.Path))
		}
		if mismatch := runtimeDraftTextIndexCountMismatch(text, cleanDraftRoot); mismatch != "" {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q %s", idx, output.Path, mismatch))
		}
		if runtimeDraftTextHasMalformedMarkdown(text) {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q contains malformed markdown inline-code or code-fence syntax", idx, output.Path))
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
		"draft surface initialized",
		"draft surface initialized for the scoped repository analysis",
		"this draft is grounded in the current step manifest",
		"current draft manifest",
		"manifest target remains",
		"draft final root",
		"draft_final_root",
		"bounded staged evidence",
		"bounded read",
		"bounded evidence read",
		"recovery pass",
		"recovery action",
		"enrichment read",
		"enrichment pass",
		"final content must stay tied to collected shard evidence and validator output",
		"runtime proposal surface initialized for this analysis run",
		"runtime draft recovery initialized",
		"draft recovery initialized",
		"treat this as diagnostic evidence until",
		"bootstrap-only placeholder",
		"placeholder draft content",
		"placeholder draft text",
		"placeholder content",
		"placeholder proposal content",
		"replace placeholder",
		"replaced placeholder",
		"replacing placeholders",
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

var liveRunIDPattern = regexp.MustCompile(`\brun_[0-9]{8}_[0-9]{6}_[0-9]{3}\b`)

func runtimeDraftTextForeignRunID(text string, expectedRunID string) string {
	expected := strings.TrimSpace(expectedRunID)
	if expected == "" {
		return ""
	}
	for _, match := range liveRunIDPattern.FindAllString(text, -1) {
		if match != expected {
			return match
		}
	}
	return ""
}

func runtimeDraftTextHasGenericShardGapWording(text string) bool {
	lower := strings.ToLower(text)
	markers := []string{
		"if present above",
		"if failed or incomplete shard",
		"any failed or incomplete shard",
		"any failed, pending, checkpointed",
		"failed or incomplete shards remain",
		"failed or incomplete typed shard statuses",
		"re-run or repair any non-succeeded collection shards",
		"failed shards require rerun",
		"incomplete statuses require confirmation",
	}
	for _, marker := range markers {
		if runtimeDraftTextContainsGenericShardGapMarker(lower, marker) {
			return true
		}
	}
	return false
}

func runtimeDraftTextContainsGenericShardGapMarker(lower string, marker string) bool {
	searchFrom := 0
	for {
		rel := strings.Index(lower[searchFrom:], marker)
		if rel < 0 {
			return false
		}
		idx := searchFrom + rel
		if marker == "failed or incomplete shards remain" && runtimeDraftTextAllowedNoShardCoverageBlockerLine(lower, idx) {
			searchFrom = idx + len(marker)
			continue
		}
		return true
	}
}

func runtimeDraftTextAllowedNoShardCoverageBlockerLine(lower string, markerIdx int) bool {
	lineStart := strings.LastIndex(lower[:markerIdx], "\n")
	if lineStart < 0 {
		lineStart = 0
	} else {
		lineStart++
	}
	lineEndRel := strings.Index(lower[markerIdx:], "\n")
	lineEnd := len(lower)
	if lineEndRel >= 0 {
		lineEnd = markerIdx + lineEndRel
	}
	line := strings.TrimSpace(lower[lineStart:lineEnd])
	line = strings.TrimLeft(line, "-*0123456789. )\t")
	if !strings.HasPrefix(line, "no failed or incomplete shards remain") {
		return false
	}
	return strings.Contains(line, "current-run") && (strings.Contains(line, "typed shard summary") || strings.Contains(line, "typed shard status"))
}

func runtimeDraftTextHasStaleIndexAvailabilityClaim(text string) bool {
	lower := strings.ToLower(text)
	markers := []string{
		"no readable current-run final-run-index.json",
		"no current-run final-run-index.json",
		"current-run final index and citation index availability is: not present",
		"current-run final-run-index.json or citation-index.json was not present",
		"current-run final-run-index.json or citation-index.json was unavailable",
		"current-run final-run-index.json or citation-index.json were unavailable",
		"final-run-index.json and citation-index.json were not present",
		"final-run-index.json and citation-index.json unavailable",
		"no current-run final-run-index document list was available",
		"no current-run final-run-index document list is available",
		"final-run-index document list was unavailable",
		"final-run-index document list is unavailable",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func runtimeDraftTextHasStaleIndexZeroClaim(text string) bool {
	lower := strings.ToLower(text)
	markers := []string{
		"final-run-index.json contains 0 observed document entries",
		"final-run-index contains 0 observed document entries",
		"current-run final-run-index document entries: 0",
		"current-run final-run-index documents: 0",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	for _, line := range strings.Split(lower, "\n") {
		if !strings.Contains(line, "final-run-index") {
			continue
		}
		for _, zeroMarker := range []string{
			"0 observed document entries",
			"0 observed documents",
			"0 canonical document",
			"0 indexed document",
			"0 document entries",
		} {
			if strings.Contains(line, zeroMarker) {
				return true
			}
		}
	}
	return false
}

func runtimeDraftTextHasRawStructuredEvidenceDump(text string) bool {
	markers := []string{
		"{'id':",
		"{ 'id':",
		"documents=[{",
		"citations=[{",
		"claim_ids':",
		"document_ids':",
		"'repo':",
		"'path':",
	}
	hits := 0
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			hits++
		}
	}
	if hits >= 2 {
		return true
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- {'") || strings.HasPrefix(trimmed, "* {'") {
			return true
		}
		if strings.Contains(trimmed, "documents=[{") || strings.Contains(trimmed, "citations=[{") {
			return true
		}
	}
	return false
}

func runtimeDraftTextHasMetadataOnlyEvidenceBullet(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimLeft(trimmed, "-*0123456789. )\t")
		trimmed = strings.TrimSpace(trimmed)
		lower := strings.ToLower(trimmed)
		for _, marker := range []string{
			`"version":`,
			`"generated_at":`,
			`"run_id":`,
			`"pipeline":`,
			`"citation_index_path":`,
		} {
			if strings.HasPrefix(lower, marker) {
				return true
			}
		}
	}
	return false
}

func runtimeDraftTextIndexCountMismatch(text string, draftRoot string) string {
	finalDocumentCount, hasFinalDocumentCount := readRuntimeDraftFinalDocumentCount(draftRoot)
	citationCount, hasCitationCount := readRuntimeDraftCitationCount(draftRoot)
	for _, line := range strings.Split(text, "\n") {
		lower := strings.ToLower(line)
		if hasFinalDocumentCount && (strings.Contains(lower, "final-run-index") || strings.Contains(lower, "final run index")) {
			for _, claimed := range extractRuntimeDraftCounts(line, finalIndexDocumentCountPatterns) {
				if claimed != finalDocumentCount {
					return fmt.Sprintf("claims final-run-index canonical document count %d but current-run index contains %d", claimed, finalDocumentCount)
				}
			}
		}
		if hasCitationCount && (strings.Contains(lower, "citation-index") || strings.Contains(lower, "citation index")) {
			for _, claimed := range extractRuntimeDraftCounts(line, citationIndexCountPatterns) {
				if claimed != citationCount {
					return fmt.Sprintf("claims citation-index citation count %d but current-run index contains %d", claimed, citationCount)
				}
			}
		}
	}
	return ""
}

func extractRuntimeDraftCounts(line string, patterns []*regexp.Regexp) []int {
	counts := []int{}
	for _, pattern := range patterns {
		for _, match := range pattern.FindAllStringSubmatch(line, -1) {
			if len(match) < 2 {
				continue
			}
			var value int
			if _, err := fmt.Sscanf(match[1], "%d", &value); err == nil {
				counts = append(counts, value)
			}
		}
	}
	return counts
}

func readRuntimeDraftFinalDocumentCount(draftRoot string) (int, bool) {
	raw, err := os.ReadFile(filepath.Join(draftRoot, "final-run-index.json"))
	if err != nil {
		return 0, false
	}
	var index struct {
		CanonicalDocuments []json.RawMessage `json:"canonical_documents"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		return 0, false
	}
	return len(index.CanonicalDocuments), true
}

func readRuntimeDraftCitationCount(draftRoot string) (int, bool) {
	raw, err := os.ReadFile(filepath.Join(draftRoot, "citation-index.json"))
	if err != nil {
		return 0, false
	}
	var index struct {
		Citations []json.RawMessage `json:"citations"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		return 0, false
	}
	return len(index.Citations), true
}

func runtimeDraftStep0TextHasDownstreamEvidenceLeak(text string) bool {
	lower := strings.ToLower(text)
	markers := []string{
		"final-run-index.json",
		"citation-index.json",
		"validator-verdict.json",
		"reports/findings/",
		"reports/coverage/",
		"reports/changelog/",
		"reports/taskruns/",
		"staging/final",
		"staging/shards",
		"collected shard",
		"shard manifest",
		"validator output",
		"runtime provider",
		"produced by:",
		"draft manifest",
		"draft root",
		"manifest mutation",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	for _, line := range strings.Split(lower, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- generated:") || strings.HasPrefix(trimmed, "generated:") {
			return true
		}
	}
	return false
}

func runtimeDraftTextHasMalformedMarkdown(text string) bool {
	inFence := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.Count(line, "`")%2 != 0 {
			return true
		}
	}
	return inFence
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
