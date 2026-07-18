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
	architectureHomeRunIDPattern            = regexp.MustCompile(`(?i)\brun[_-]\d{8}(?:[_-]\d{6})?`)
	architectureHomeProcessIdentityPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bcurrent[- ]run\b`),
		regexp.MustCompile(`(?i)\btyped[- ]shard\b`),
		regexp.MustCompile(`(?i)\bshard[- ]packs?(?:[- ]manifests?)?\b`),
		regexp.MustCompile(`(?i)\bplanned\s*=\s*\d+\b.*\bsucceeded\s*=\s*\d+\b.*\bfailed\s*=\s*\d+\b.*\bincomplete\s*=\s*\d+\b`),
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
		if runtimeDraftTextHasEmptyEvidenceReferenceSlot(text) {
			problems = append(problems, fmt.Sprintf("outputs[%d].path %q contains empty evidence reference slots, likely from shell-expanded markdown paths", idx, output.Path))
		}
		if strings.TrimSpace(stepID) == "init.step2.asis_docs" || strings.TrimSpace(stepID) == "refresh.step2.asis_docs" {
			if filepath.ToSlash(path.Clean(strings.TrimSpace(output.CanonicalPath))) == "reports/as-is/overview.md" {
				if missing := runtimeDraftArchitectureHomeMissingSections(text); len(missing) > 0 {
					problems = append(problems, fmt.Sprintf("outputs[%d].path %q Architecture Home has missing or empty required sections: %s", idx, output.Path, strings.Join(missing, ", ")))
				}
				if runtimeDraftArchitectureHomeHasProcessNarration(text) {
					problems = append(problems, fmt.Sprintf("outputs[%d].path %q Architecture Home contains runtime/process narration, manifest recap, or unsupported confidence language", idx, output.Path))
				}
				if runtimeDraftArchitectureHomeHasTaskrunStagingReference(text) {
					problems = append(problems, fmt.Sprintf("outputs[%d].path %q Architecture Home references taskrun staging paths instead of canonical or repository evidence", idx, output.Path))
				}
				if runtimeDraftArchitectureHomeHasRuntimeCheckoutReference(text) {
					problems = append(problems, fmt.Sprintf("outputs[%d].path %q Architecture Home references runtime checkout paths instead of stable repository-relative evidence", idx, output.Path))
				}
			}
			if mismatch := runtimeDraftTextAsIsShardCompletenessMismatch(text, cleanDraftRoot, runID, output); mismatch != "" {
				problems = append(problems, fmt.Sprintf("outputs[%d].path %q %s", idx, output.Path, mismatch))
			}
		}
		if strings.TrimSpace(stepID) == "init.step4.proposals" || strings.TrimSpace(stepID) == "refresh.step4.proposals" {
			if mismatch := runtimeDraftTextProposalCompletenessMismatch(text, cleanDraftRoot, runID, output); mismatch != "" {
				problems = append(problems, fmt.Sprintf("outputs[%d].path %q %s", idx, output.Path, mismatch))
			}
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

func runtimeDraftArchitectureHomeHasProcessNarration(text string) bool {
	lower := strings.ToLower(text)
	markers := []string{
		"runtime provider",
		"provider execution",
		"provider generated",
		"manifest recap",
		"taskrun mechanics",
		"derived from the run",
		"derived from run",
		"current run collect",
		"current-run collect",
		"collect pass",
		"current run analyzed",
		"current analysis covers",
		"staged in the current run",
		"typed shard plan",
		"typed shard summary",
		"shard plan/summary",
		"shard-pack manifest",
		"shard completeness for this run",
		"based on the staged source",
		"confidence:",
		"confidence level",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	if architectureHomeRunIDPattern.MatchString(text) {
		return true
	}
	for _, pattern := range architectureHomeProcessIdentityPatterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

func runtimeDraftArchitectureHomeHasRuntimeCheckoutReference(text string) bool {
	normalized := strings.ReplaceAll(strings.ToLower(text), `\`, "/")
	return strings.Contains(normalized, "/.acp/repos/")
}

func runtimeDraftArchitectureHomeHasTaskrunStagingReference(text string) bool {
	for _, line := range strings.Split(strings.ToLower(text), "\n") {
		normalized := strings.ReplaceAll(line, `\`, "/")
		searchFrom := 0
		for {
			rel := strings.Index(normalized[searchFrom:], "reports/taskruns/")
			if rel < 0 {
				break
			}
			start := searchFrom + rel + len("reports/taskruns/")
			if strings.Contains(normalized[start:], "/staging") {
				return true
			}
			searchFrom = start
		}
	}
	return false
}

func runtimeDraftArchitectureHomeMissingSections(text string) []string {
	required := []string{
		"System at a glance",
		"Analyzed scope",
		"Domains and ownership",
		"Key flows",
		"Integrations and datastores",
		"Where to start",
		"Safe-change guidance",
		"Evidence gaps and open questions",
	}
	sections := map[string]bool{}
	current := ""
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			current = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
			continue
		}
		if current != "" && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			sections[current] = true
		}
	}
	missing := make([]string, 0)
	for _, heading := range required {
		if !sections[strings.ToLower(heading)] {
			missing = append(missing, heading)
		}
	}
	return missing
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
		"bounded evidence read",
		"bounded read root",
		"bounded read roots",
		"bounded read pass",
		"bounded read_context_roots",
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
		"replace placeholder content",
		"replace placeholder proposal content",
		"replaced placeholder content",
		"replaced placeholder proposal content",
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
		"current-run final-run-index.json or citation-index.json was not yet present",
		"current-run final-run-index.json or citation-index.json were not yet present",
		"current-run final-run-index.json or citation-index.json was not yet available",
		"current-run final-run-index.json or citation-index.json were not yet available",
		"current-run final-run-index.json and citation-index.json were not yet present",
		"current-run final-run-index.json and citation-index.json were not yet available",
		"final-run-index.json and citation-index.json were not present",
		"final-run-index.json and citation-index.json unavailable",
		"final-run-index.json and citation-index.json were not yet present",
		"final-run-index.json and citation-index.json were not yet available",
		"final-run-index.json and citation-index.json are not yet present",
		"final-run-index.json and citation-index.json are not yet available",
		"final-run-index and citation-index are not yet present",
		"final-run-index and citation-index are not yet available",
		"final-run-index and citation-index were not yet present",
		"final-run-index and citation-index were not yet available",
		"no current-run final-run-index document list was available",
		"no current-run final-run-index document list is available",
		"final-run-index document list was unavailable",
		"final-run-index document list is unavailable",
		"final-run-index document list is not yet available",
		"final-run-index document list was not yet available",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	for _, line := range strings.Split(lower, "\n") {
		if !runtimeDraftLineMentionsIndex(line) {
			continue
		}
		if runtimeDraftLineClaimsIndexUnavailable(line) {
			return true
		}
	}
	return false
}

func runtimeDraftLineMentionsIndex(line string) bool {
	return strings.Contains(line, "final-run-index") ||
		strings.Contains(line, "citation-index")
}

func runtimeDraftLineClaimsIndexUnavailable(line string) bool {
	for _, marker := range []string{
		"not observed",
		"not found",
		"not available",
		"not present",
		"not readable",
		"not yet observed",
		"not yet found",
		"not yet available",
		"not yet present",
		"was unavailable",
		"were unavailable",
		"is unavailable",
		"are unavailable",
	} {
		if strings.Contains(line, marker) {
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

func runtimeDraftTextHasEmptyEvidenceReferenceSlot(text string) bool {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:from|checked|read|use|using|under|for|at)(?::)?\s{2,}(?:and|under|for|to|\.|,|;|$)`),
		regexp.MustCompile(`(?i):\s{2,}and\s{2,}`),
		regexp.MustCompile(`(?i)\b(?:under|for|from)\s+\.\s*$`),
		regexp.MustCompile(`(?i)\buse\s{2,}and\s{2,}`),
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		for _, pattern := range patterns {
			if pattern.FindStringIndex(trimmed) != nil {
				return true
			}
		}
	}
	return false
}

type runtimeDraftShardCompleteness struct {
	planned    int
	succeeded  int
	failed     int
	incomplete int
}

func runtimeDraftTextAsIsShardCompletenessMismatch(text string, draftRoot string, runID string, output Output) string {
	counts, ok := readRuntimeDraftShardCompleteness(draftRoot, runID)
	if !ok || counts.planned == 0 {
		return ""
	}
	lower := strings.ToLower(text)
	if runtimeDraftTextClaimsShardEvidenceAbsent(lower) {
		return fmt.Sprintf("claims staging shard evidence is empty but current-run typed shard summary contains %d planned shard(s)", counts.planned)
	}

	canonicalPath := filepath.ToSlash(path.Clean(strings.TrimSpace(output.CanonicalPath)))
	if canonicalPath == "reports/as-is/overview.md" && !runtimeDraftTextHasConcreteEvidenceRef(text) {
		return "does not include concrete repo/path, citation, or staged artifact evidence references"
	}
	if canonicalPath == "reports/agent-outputs/architect/summary.md" && !runtimeDraftTextHasOperatorDecisionCue(text) {
		return "does not include a decision-ready operator summary with next inspection or decision cues"
	}
	if canonicalPath != "reports/coverage/summary.md" {
		return ""
	}
	summaryProblems := []string{}
	if runtimeDraftTextHasShardSummaryMetadataDump(text) {
		summaryProblems = append(summaryProblems, "includes metadata-only shard-summary keys instead of architecture coverage evidence")
	}
	if !runtimeDraftTextClaimsShardCompleteness(text, counts) {
		summaryProblems = append(summaryProblems, fmt.Sprintf("does not report exact current-run shard completeness from typed shard summary: planned=%d succeeded=%d failed=%d incomplete=%d", counts.planned, counts.succeeded, counts.failed, counts.incomplete))
	}
	if len(summaryProblems) > 0 {
		return strings.Join(summaryProblems, " and ")
	}
	return ""
}

func runtimeDraftTextProposalCompletenessMismatch(text string, draftRoot string, runID string, output Output) string {
	canonicalPath := filepath.ToSlash(path.Clean(strings.TrimSpace(output.CanonicalPath)))
	if !strings.HasSuffix(strings.ToLower(filepath.ToSlash(filepath.Clean(strings.TrimSpace(output.Path)))), ".md") {
		return ""
	}
	isPrimaryProposal := canonicalPath == "proposals/runtime-recommendations.md"
	isChangelog := strings.HasPrefix(canonicalPath, "reports/changelog/")
	if !isPrimaryProposal && !isChangelog {
		return ""
	}

	if mismatch := runtimeDraftTextFindingsProposalDisconnect(text, draftRoot, runID, isPrimaryProposal); mismatch != "" {
		return mismatch
	}

	counts, ok := readRuntimeDraftShardCompleteness(draftRoot, runID)
	if ok && counts.planned > 0 && !runtimeDraftTextClaimsShardCompleteness(text, counts) {
		return fmt.Sprintf("does not report exact current-run proposal shard completeness from typed shard summary: planned=%d succeeded=%d failed=%d incomplete=%d", counts.planned, counts.succeeded, counts.failed, counts.incomplete)
	}
	if !runtimeDraftTextHasConcreteEvidenceRef(text) {
		return "does not include concrete repo/path, citation, or staged artifact evidence references"
	}
	if runtimeDraftTextHasDanglingProposalReference(text) {
		return "references findings/proposals above without including substantive findings/proposals"
	}

	if isPrimaryProposal {
		return runtimeDraftTextProposalBodyMismatch(text)
	}
	if isChangelog {
		return runtimeDraftTextProposalChangelogMismatch(text)
	}
	return ""
}

type runtimeDraftFindingSummary struct {
	ids            []string
	highOrMediumID bool
	highMediumIDs  []string
	severityByID   map[string]string
}

func runtimeDraftTextFindingsProposalDisconnect(text string, draftRoot string, runID string, requireActionability bool) string {
	findingsText, ok := readRuntimeDraftCurrentRunFindings(draftRoot, runID)
	if !ok {
		return ""
	}
	findings := runtimeDraftSummarizeMarkdownFindings(findingsText)
	if len(findings.ids) == 0 || strings.Contains(strings.ToLower(findingsText), "no findings reported.") {
		return ""
	}
	findingsHint := runtimeDraftFindingIDsHint(findings.ids)
	highMediumHint := runtimeDraftHighMediumFindingHint(findings)
	if runtimeDraftTextDeniesStructuredFindings(text) {
		return "claims no structured finding summary despite non-empty current-run findings" + findingsHint
	}
	if placeholder := runtimeDraftTextSyntheticFindingPlaceholder(text); placeholder != "" {
		return fmt.Sprintf("uses synthetic current-run finding placeholder %q despite non-empty current-run findings", placeholder) + findingsHint
	}
	if placeholder := runtimeDraftProposalTextHasPlaceholderFindingIDBullet(text); placeholder != "" {
		return fmt.Sprintf("uses placeholder Finding ID %q in actionable findings despite non-empty current-run findings", placeholder) + findingsHint
	}
	if !runtimeDraftTextReferencesAnyFindingID(text, findings.ids) {
		return "does not reference any current-run finding ID despite non-empty current-run findings" + findingsHint
	}
	if requireActionability && findings.highOrMediumID {
		if runtimeDraftProposalTextHasActionabilityTable(text, findings.highMediumIDs) {
			return "uses markdown table for medium/high actionable findings; expected a bullet-only Top Actionable Findings section with one bullet per finding and exact Finding ID, copied Severity value, Affected surface/path, Recommended operator action, and Residual gap all on the same bullet line" + highMediumHint
		}
		if !runtimeDraftProposalTextHasFindingActionability(text, findings.highMediumIDs) {
			return "does not link medium/high current-run findings to recommended operator action and affected surface/path; expected a bullet-only Top Actionable Findings section with one bullet per finding and exact Finding ID, copied Severity value from findings.md, Affected surface/path from Related IDs/Evidence, Recommended operator action, and Residual gap all on the same bullet line" + highMediumHint
		}
	}
	return ""
}

func runtimeDraftFindingIDsHint(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	limit := len(ids)
	if limit > 3 {
		limit = 3
	}
	return fmt.Sprintf("; current-run finding IDs include %s; copy IDs from backticked '- ID: `...`' lines in staging/final/reports/findings/findings.md, never from staging/final/reports/findings.md, and do not write synthetic placeholders such as no-current-run-finding-id", strings.Join(ids[:limit], ", "))
}

func runtimeDraftHighMediumFindingHint(findings runtimeDraftFindingSummary) string {
	if len(findings.highMediumIDs) == 0 {
		return runtimeDraftFindingIDsHint(findings.ids)
	}
	limit := len(findings.highMediumIDs)
	if limit > 3 {
		limit = 3
	}
	parts := make([]string, 0, limit)
	for _, id := range findings.highMediumIDs[:limit] {
		severity := strings.TrimSpace(findings.severityByID[id])
		if severity == "" {
			parts = append(parts, id)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s severity=%s", id, severity))
	}
	return fmt.Sprintf("; current-run high/medium findings include %s; copy IDs from backticked '- ID: `...`' lines in staging/final/reports/findings/findings.md, never from staging/final/reports/findings.md, and do not write synthetic placeholders such as no-current-run-finding-id", strings.Join(parts, ", "))
}

func readRuntimeDraftCurrentRunFindings(draftRoot string, runID string) (string, bool) {
	cleanDraftRoot := filepath.Clean(strings.TrimSpace(draftRoot))
	if cleanDraftRoot == "" || cleanDraftRoot == "." || strings.TrimSpace(runID) == "" {
		return "", false
	}
	candidates := []string{
		filepath.Join(cleanDraftRoot, "reports", "findings", "findings.md"),
		filepath.Join(cleanDraftRoot, "staging", "final", "reports", "findings", "findings.md"),
		filepath.Join(cleanDraftRoot, "..", "final", "reports", "findings", "findings.md"),
		filepath.Join(cleanDraftRoot, "..", "..", "final", "reports", "findings", "findings.md"),
		filepath.Join(cleanDraftRoot, "..", "..", "..", "staging", "final", "reports", "findings", "findings.md"),
	}
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		raw, err := os.ReadFile(candidate)
		if err == nil {
			return string(raw), true
		}
	}
	return "", false
}

func runtimeDraftSummarizeMarkdownFindings(text string) runtimeDraftFindingSummary {
	summary := runtimeDraftFindingSummary{
		severityByID: map[string]string{},
	}
	currentID := ""
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, "- id:"):
			currentID = runtimeDraftFirstMarkdownFieldValue(trimmed[len("- id:"):])
			if currentID != "" {
				summary.ids = append(summary.ids, currentID)
			}
		case strings.HasPrefix(lower, "- severity:"):
			severity := strings.ToLower(runtimeDraftFirstMarkdownFieldValue(trimmed[len("- severity:"):]))
			if currentID != "" && severity != "" {
				summary.severityByID[currentID] = severity
			}
			if currentID != "" && (severity == "high" || severity == "medium") {
				summary.highOrMediumID = true
				summary.highMediumIDs = append(summary.highMediumIDs, currentID)
			}
		}
	}
	summary.ids = runtimeDraftUniqueSortedStrings(summary.ids)
	summary.highMediumIDs = runtimeDraftUniqueSortedStrings(summary.highMediumIDs)
	return summary
}

func runtimeDraftFirstMarkdownFieldValue(value string) string {
	value = strings.TrimSpace(value)
	if start := strings.Index(value, "`"); start >= 0 {
		if end := strings.Index(value[start+1:], "`"); end >= 0 {
			return strings.TrimSpace(value[start+1 : start+1+end])
		}
	}
	value = strings.Trim(value, "` \t:;,")
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], "` \t:;,.")
}

func runtimeDraftUniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func runtimeDraftTextDeniesStructuredFindings(text string) bool {
	lower := strings.ToLower(text)
	for _, marker := range []string{
		"no structured finding summary was present",
		"no structured findings were present",
		"no structured finding summary",
		"no source-level architecture change is approved",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func runtimeDraftTextSyntheticFindingPlaceholder(text string) string {
	lower := strings.ToLower(text)
	for _, marker := range []string{
		"no-current-run-finding-id",
		"no structured current-run finding id",
		"no current-run finding id was available",
		"no current-run finding id",
		"finding unavailable",
	} {
		if strings.Contains(lower, marker) {
			return marker
		}
	}
	return ""
}

func runtimeDraftTextReferencesAnyFindingID(text string, findingIDs []string) bool {
	lower := strings.ToLower(text)
	for _, id := range findingIDs {
		id = strings.ToLower(strings.TrimSpace(id))
		if id != "" && strings.Contains(lower, id) {
			return true
		}
	}
	return false
}

func runtimeDraftProposalTextHasFindingActionability(text string, findingIDs []string) bool {
	body, ok := runtimeDraftMarkdownSectionBody(text, runtimeDraftMarkdownSectionSpec{
		name:         "Top Actionable Findings",
		alternatives: [][]string{{"top", "actionable", "finding"}},
	})
	if !ok {
		return false
	}
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !runtimeDraftProposalLineIsBullet(trimmed) {
			continue
		}
		lower := strings.ToLower(trimmed)
		if !runtimeDraftTextReferencesAnyFindingID(lower, findingIDs) {
			continue
		}
		if !strings.Contains(lower, "finding id") {
			continue
		}
		if !strings.Contains(lower, "severity") || !(strings.Contains(lower, "high") || strings.Contains(lower, "medium")) {
			continue
		}
		if !runtimeDraftTextContainsAny(lower, []string{"affected surface", "affected path", "related ids", "evidence"}) {
			continue
		}
		if !runtimeDraftTextContainsAny(lower, []string{"recommended operator action", "recommended action", "operator action"}) {
			continue
		}
		if !runtimeDraftTextContainsAny(lower, []string{"update ", "add ", "document ", "assign ", "remediate", "replace"}) {
			continue
		}
		if !strings.Contains(lower, "residual gap") {
			continue
		}
		return true
	}
	return false
}

func runtimeDraftProposalTextHasPlaceholderFindingIDBullet(text string) string {
	body, ok := runtimeDraftMarkdownSectionBody(text, runtimeDraftMarkdownSectionSpec{
		name:         "Top Actionable Findings",
		alternatives: [][]string{{"top", "actionable", "finding"}},
	})
	if !ok {
		return ""
	}
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !runtimeDraftProposalLineIsBullet(trimmed) {
			continue
		}
		value := runtimeDraftProposalFindingIDFieldValue(trimmed)
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "none", "n/a", "na", "not-applicable", "not_applicable", "unavailable", "finding-unavailable", "finding_unavailable", "no-current-run-finding-id":
			return value
		}
	}
	return ""
}

func runtimeDraftProposalFindingIDFieldValue(line string) string {
	lower := strings.ToLower(line)
	idx := strings.Index(lower, "finding id")
	if idx < 0 {
		return ""
	}
	value := strings.TrimSpace(line[idx+len("finding id"):])
	value = strings.TrimLeft(value, " \t:`")
	for end, r := range value {
		switch r {
		case ';', ',', '|', '`', '\t', '\n', '\r':
			return strings.TrimSpace(value[:end])
		}
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(fields[0])
}

func runtimeDraftProposalLineIsBullet(line string) bool {
	return strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")
}

func runtimeDraftProposalTextHasActionabilityTable(text string, findingIDs []string) bool {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		if !strings.Contains(lower, "|") {
			continue
		}
		if !(strings.Contains(lower, "finding id") ||
			strings.Contains(lower, "recommended operator action") ||
			strings.Contains(lower, "affected surface") ||
			strings.Contains(lower, "affected path") ||
			runtimeDraftTextReferencesAnyFindingID(lower, findingIDs)) {
			continue
		}
		for _, neighborIndex := range []int{i - 1, i + 1} {
			if neighborIndex < 0 || neighborIndex >= len(lines) {
				continue
			}
			if runtimeDraftLineIsMarkdownTableSeparator(lines[neighborIndex]) {
				return true
			}
		}
	}
	return false
}

func runtimeDraftLineIsMarkdownTableSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "|") || !strings.Contains(trimmed, "---") {
		return false
	}
	trimmed = strings.Trim(trimmed, "| ")
	for _, cell := range strings.Split(trimmed, "|") {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			continue
		}
		if !regexp.MustCompile(`^:?-{3,}:?$`).MatchString(cell) {
			return false
		}
	}
	return true
}

func runtimeDraftTextContainsAny(text string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func runtimeDraftTextProposalBodyMismatch(text string) string {
	required := []runtimeDraftMarkdownSectionSpec{
		{name: "Decision / recommended operator action", alternatives: [][]string{{"decision"}, {"recommended", "operator", "action"}}},
		{name: "Evidence used", alternatives: [][]string{{"evidence"}}},
		{name: "Proposed changes or follow-up plan", alternatives: [][]string{{"proposed", "changes"}, {"follow-up", "plan"}, {"follow up", "plan"}}},
		{name: "Risks, gaps, and out-of-scope notes", alternatives: [][]string{{"risk"}, {"gap"}, {"out-of-scope"}, {"out of scope"}}},
	}
	if missing := runtimeDraftMissingMarkdownSections(text, required); len(missing) > 0 {
		return "is missing substantive proposal section(s): " + strings.Join(missing, ", ")
	}
	body, ok := runtimeDraftMarkdownSectionBody(text, runtimeDraftMarkdownSectionSpec{
		name:         "Proposed changes or follow-up plan",
		alternatives: [][]string{{"proposed", "changes"}, {"follow-up", "plan"}, {"follow up", "plan"}},
	})
	if !ok {
		return "is missing substantive proposal section(s): Proposed changes or follow-up plan"
	}
	if runtimeDraftTextHasExplicitNoActionableProposalGap(body) {
		return ""
	}
	if !runtimeDraftProposalBodyHasActionableLine(body) {
		return "does not include an actionable proposal or explicit no-actionable-proposal gap"
	}
	return ""
}

func runtimeDraftTextProposalChangelogMismatch(text string) string {
	required := []runtimeDraftMarkdownSectionSpec{
		{name: "Updated architecture/proposal surfaces", alternatives: [][]string{{"updated", "surface"}, {"architecture", "proposal", "surface"}}},
		{name: "Findings/proposals summary", alternatives: [][]string{{"findings", "summary"}, {"proposals", "summary"}, {"findings/proposals"}}},
		{name: "Evidence index or citation references", alternatives: [][]string{{"evidence"}, {"citation"}}},
		{name: "Residual coverage gaps", alternatives: [][]string{{"residual", "gap"}, {"coverage", "gap"}}},
	}
	if missing := runtimeDraftMissingMarkdownSections(text, required); len(missing) > 0 {
		return "is missing substantive proposal changelog section(s): " + strings.Join(missing, ", ")
	}
	return ""
}

type runtimeDraftMarkdownSectionSpec struct {
	name         string
	alternatives [][]string
}

func runtimeDraftMissingMarkdownSections(text string, specs []runtimeDraftMarkdownSectionSpec) []string {
	missing := []string{}
	for _, spec := range specs {
		body, ok := runtimeDraftMarkdownSectionBody(text, spec)
		if !ok || !runtimeDraftMarkdownBodyHasSubstantiveContent(body) {
			missing = append(missing, spec.name)
		}
	}
	return missing
}

func runtimeDraftMarkdownSectionBody(text string, spec runtimeDraftMarkdownSectionSpec) (string, bool) {
	lines := strings.Split(text, "\n")
	start := -1
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		heading := strings.ToLower(strings.TrimSpace(strings.TrimLeft(trimmed, "#")))
		if runtimeDraftHeadingMatches(heading, spec.alternatives) {
			start = idx + 1
			break
		}
	}
	if start < 0 {
		return "", false
	}
	end := len(lines)
	for idx := start; idx < len(lines); idx++ {
		if strings.HasPrefix(strings.TrimSpace(lines[idx]), "#") {
			end = idx
			break
		}
	}
	return strings.Join(lines[start:end], "\n"), true
}

func runtimeDraftHeadingMatches(heading string, alternatives [][]string) bool {
	for _, terms := range alternatives {
		matched := true
		for _, term := range terms {
			if !strings.Contains(heading, strings.ToLower(strings.TrimSpace(term))) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func runtimeDraftMarkdownBodyHasSubstantiveContent(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimLeft(trimmed, "-*0123456789. )\t")
		trimmed = strings.TrimSpace(trimmed)
		if len(trimmed) >= 12 {
			return true
		}
	}
	return false
}

func runtimeDraftTextHasDanglingProposalReference(text string) bool {
	lower := strings.ToLower(text)
	if runtimeDraftTextHasSubstantiveLinkedProposalContent(lower) {
		return false
	}
	markers := []string{
		"finding above",
		"findings above",
		"proposal above",
		"proposals above",
		"each finding above",
		"each proposal above",
		"findings listed above",
		"proposals listed above",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func runtimeDraftTextHasSubstantiveLinkedProposalContent(lower string) bool {
	if !strings.Contains(lower, "finding id:") {
		return false
	}
	if !runtimeDraftTextContainsAny(lower, []string{
		"recommended operator action",
		"recommended action",
		"operator action",
		"proposed changes",
		"follow-up plan",
	}) {
		return false
	}
	if !runtimeDraftTextContainsAny(lower, []string{
		"affected surface",
		"affected path",
		"related ids",
		"evidence",
	}) {
		return false
	}
	return strings.Contains(lower, "residual gap") ||
		strings.Contains(lower, "residual coverage gap") ||
		strings.Contains(lower, "proposal implementation remains unverified")
}

func runtimeDraftTextHasExplicitNoActionableProposalGap(text string) bool {
	lower := strings.ToLower(text)
	markers := []string{
		"no actionable proposal evidence",
		"no actionable proposals",
		"no actionable proposal",
		"no structured finding summary was present",
		"no structured findings were present",
		"no proposal candidates were present",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func runtimeDraftProposalBodyHasActionableLine(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimLeft(trimmed, "-*0123456789. )\t")
		trimmed = strings.TrimSpace(trimmed)
		if len(trimmed) < 20 {
			continue
		}
		if runtimeDraftProposalLineIsGeneric(trimmed) {
			continue
		}
		return true
	}
	return false
}

func runtimeDraftProposalLineIsGeneric(line string) bool {
	lower := strings.ToLower(line)
	markers := []string{
		"prioritize each finding",
		"prioritise each finding",
		"address the surfaced findings",
		"update the architecture model and source files",
		"re-run the collect/validate pipeline",
		"rerun the collect/validate pipeline",
		"use the cited documents as the source of truth",
		"use the cited documents as source of truth",
		"review current-run evidence",
		"current-run evidence should be reviewed",
		"current run evidence should be reviewed",
		"follow-up surfaces",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func runtimeDraftTextClaimsShardEvidenceAbsent(lower string) bool {
	markers := []string{
		"staging shard directory contains 0 file",
		"staging shards directory contains 0 file",
		"staging shard directory contains zero file",
		"staging shards directory contains zero file",
		"staging shard directory contains 0 shard",
		"staging shards directory contains 0 shard",
		"shard pack manifests: none observed",
		"shard-pack manifests: none observed",
		"shard manifests: none observed",
		"collected shard manifests: none observed",
		"shard-pack-manifest.json: none observed",
		"no shard pack manifests observed",
		"no shard-pack manifests observed",
		"no shard manifests observed",
		"no collected shard manifests observed",
		"no shard-pack-manifest.json",
		"0 shard-pack-manifest",
		"zero shard-pack-manifest",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func runtimeDraftTextHasConcreteEvidenceRef(text string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "cite.") ||
		strings.Contains(lower, "reports/as-is/") ||
		strings.Contains(lower, "reports/coverage/") ||
		strings.Contains(lower, "reports/findings/") ||
		strings.Contains(lower, "final-run-index.json") ||
		strings.Contains(lower, "citation-index.json") ||
		strings.Contains(lower, "shard-pack-manifest.json") {
		return true
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, " -> ") && strings.Contains(trimmed, "/") {
			return true
		}
		if strings.Contains(trimmed, ".") && strings.Contains(trimmed, "/") {
			return true
		}
	}
	return false
}

func runtimeDraftTextHasOperatorDecisionCue(text string) bool {
	lower := strings.ToLower(text)
	decisionMarkers := []string{
		"operator",
		"decision",
		"decide",
		"inspect next",
		"what to inspect next",
		"next inspection",
		"what is complete",
		"what is missing",
		"review next",
		"publish",
		"accept",
		"residual risk",
	}
	for _, marker := range decisionMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func runtimeDraftTextHasShardSummaryMetadataDump(text string) bool {
	hits := 0
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimLeft(trimmed, "-*0123456789. )\t")
		lower := strings.ToLower(strings.TrimSpace(trimmed))
		for _, marker := range []string{
			"meta:",
			"step_id:",
			"domain_id:",
			"strategy:",
			"max_parallel_tasks:",
			"failure_policy:",
			"shard_discovery_mode:",
		} {
			if strings.HasPrefix(lower, marker) {
				hits++
				break
			}
		}
	}
	return hits >= 2
}

func readRuntimeDraftShardCompleteness(draftRoot string, runID string) (runtimeDraftShardCompleteness, bool) {
	taskrunsRoot := filepath.Clean(filepath.Join(strings.TrimSpace(draftRoot), "..", "..", "..", ".."))
	if strings.TrimSpace(runID) == "" {
		return runtimeDraftShardCompleteness{}, false
	}
	pattern := filepath.Join(taskrunsRoot, strings.TrimSpace(runID)+"-*-step1-collect-shard-summary-*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return runtimeDraftShardCompleteness{}, false
	}
	sort.Strings(matches)
	for _, match := range matches {
		counts, ok := readRuntimeDraftShardCompletenessFile(match)
		if ok && counts.planned > 0 {
			return counts, true
		}
	}
	return runtimeDraftShardCompleteness{}, false
}

func readRuntimeDraftShardCompletenessFile(filename string) (runtimeDraftShardCompleteness, bool) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return runtimeDraftShardCompleteness{}, false
	}
	var summary struct {
		Items []struct {
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &summary); err != nil || len(summary.Items) == 0 {
		return runtimeDraftShardCompleteness{}, false
	}
	counts := runtimeDraftShardCompleteness{planned: len(summary.Items)}
	for _, item := range summary.Items {
		switch strings.ToLower(strings.TrimSpace(item.Status)) {
		case "succeeded":
			counts.succeeded++
		case "failed":
			counts.failed++
		default:
			counts.incomplete++
		}
	}
	return counts, true
}

func runtimeDraftTextClaimsShardCompleteness(text string, counts runtimeDraftShardCompleteness) bool {
	lower := strings.ToLower(text)
	exactSucceeded := fmt.Sprintf("%d/%d succeeded", counts.succeeded, counts.planned)
	hasExactSucceeded := strings.Contains(lower, exactSucceeded)
	hasPlanned := hasExactSucceeded || runtimeDraftTextHasShardCompletenessCount(lower, "planned", counts.planned)
	hasSucceeded := hasExactSucceeded || runtimeDraftTextHasShardCompletenessCount(lower, "succeeded", counts.succeeded)
	hasFailed := runtimeDraftTextHasShardCompletenessCount(lower, "failed", counts.failed)
	if counts.failed == 0 {
		hasFailed = hasFailed || runtimeDraftTextClaimsNoFailedShard(lower)
	}
	hasIncomplete := runtimeDraftTextHasShardCompletenessCount(lower, "incomplete", counts.incomplete) ||
		runtimeDraftTextHasShardCompletenessCount(lower, "pending", counts.incomplete)
	if counts.incomplete == 0 {
		hasIncomplete = hasIncomplete || runtimeDraftTextClaimsNoIncompleteShard(lower)
	}
	return hasPlanned && hasSucceeded && hasFailed && hasIncomplete
}

func runtimeDraftTextHasShardCompletenessCount(lower string, label string, count int) bool {
	quoted := regexp.QuoteMeta(strings.ToLower(strings.TrimSpace(label)))
	patterns := []string{
		fmt.Sprintf(`\b%d\s+%s\b`, count, quoted),
		fmt.Sprintf(`\b%d\s+(?:[a-z][a-z0-9_-]*\s+){1,3}%s\b`, count, quoted),
		fmt.Sprintf(`\b%s\s*(?:=|:)?\s*%d\b`, quoted, count),
	}
	for _, pattern := range patterns {
		if regexp.MustCompile(pattern).FindStringIndex(lower) != nil {
			return true
		}
	}
	return false
}

func runtimeDraftTextClaimsNoFailedShard(lower string) bool {
	return strings.Contains(lower, "no failed") || strings.Contains(lower, "failed=0")
}

func runtimeDraftTextClaimsNoIncompleteShard(lower string) bool {
	return strings.Contains(lower, "no failed, pending, or incomplete") ||
		strings.Contains(lower, "no failed or incomplete") ||
		strings.Contains(lower, "no incomplete") ||
		strings.Contains(lower, "incomplete=0") ||
		strings.Contains(lower, "pending=0")
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
