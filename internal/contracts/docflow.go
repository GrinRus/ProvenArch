package contracts

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/validation"
)

type AuthoredDocument struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Title         string   `json:"title"`
	Path          string   `json:"path"`
	CanonicalPath string   `json:"canonical_path"`
	Topics        []string `json:"topics"`
	CitationIDs   []string `json:"citation_ids"`
	Status        string   `json:"status,omitempty"`
}

type DocumentCitation struct {
	ID          string     `json:"id"`
	Repo        string     `json:"repo"`
	Ref         string     `json:"ref,omitempty"`
	Path        string     `json:"path"`
	Lines       *LineRange `json:"lines,omitempty"`
	ExcerptHash string     `json:"excerpt_hash,omitempty"`
	Excerpt     string     `json:"excerpt,omitempty"`
	ClaimIDs    []string   `json:"claim_ids"`
	DocumentIDs []string   `json:"document_ids"`
}

type ShardPackManifest struct {
	Version      int                `json:"version"`
	RunID        string             `json:"run_id"`
	StepID       string             `json:"step_id"`
	ShardID      string             `json:"shard_id"`
	DomainID     string             `json:"domain_id,omitempty"`
	AgentRole    string             `json:"agent_role"`
	ArtifactRoot string             `json:"artifact_root"`
	RepoScopes   []string           `json:"repo_scopes,omitempty"`
	PathScopes   []string           `json:"path_scopes,omitempty"`
	Summary      string             `json:"summary,omitempty"`
	Documents    []AuthoredDocument `json:"documents"`
	Citations    []DocumentCitation `json:"citations"`
	Semantic     SemanticSnapshot   `json:"semantic"`
}

type FinalRunDocument struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Title         string   `json:"title"`
	CanonicalPath string   `json:"canonical_path"`
	StagedPath    string   `json:"staged_path"`
	Topics        []string `json:"topics"`
	CitationIDs   []string `json:"citation_ids"`
	SourceShards  []string `json:"source_shards"`
	Status        string   `json:"status"`
}

type TopicIndexEntry struct {
	ID          string   `json:"id"`
	DocumentIDs []string `json:"document_ids"`
}

type FinalRunIndex struct {
	Version            int                `json:"version"`
	RunID              string             `json:"run_id"`
	Pipeline           string             `json:"pipeline"`
	GeneratedAt        string             `json:"generated_at"`
	CitationIndexPath  string             `json:"citation_index_path"`
	CanonicalDocuments []FinalRunDocument `json:"canonical_documents"`
	Topics             []TopicIndexEntry  `json:"topics"`
	Semantic           SemanticSnapshot   `json:"semantic"`
}

type CitationIndex struct {
	Version     int                `json:"version"`
	RunID       string             `json:"run_id"`
	GeneratedAt string             `json:"generated_at"`
	Citations   []DocumentCitation `json:"citations"`
}

type ValidatorIssue struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Path       string `json:"path,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	CitationID string `json:"citation_id,omitempty"`
}

type ValidatorVerdict struct {
	Version      int              `json:"version"`
	RunID        string           `json:"run_id"`
	GeneratedAt  string           `json:"generated_at"`
	Verdict      string           `json:"verdict"`
	Summary      string           `json:"summary,omitempty"`
	CheckedPaths []string         `json:"checked_paths"`
	FixedPaths   []string         `json:"fixed_paths,omitempty"`
	Findings     []Finding        `json:"findings,omitempty"`
	Questions    []Question       `json:"questions,omitempty"`
	Issues       []ValidatorIssue `json:"issues,omitempty"`
}

// EffectiveVerdict is the orchestrator-owned technical authority persisted
// after deterministic validation and the provider-free selected-run audit.
// The provider-authored ValidatorVerdict is deliberately kept separate and
// immutable; findings/questions are copied only as advisory semantic input.
type EffectiveVerdict struct {
	Version               int                      `json:"version"`
	Kind                  string                   `json:"kind"`
	Authority             string                   `json:"authority"`
	RunID                 string                   `json:"run_id"`
	GeneratedAt           string                   `json:"generated_at"`
	ProviderVerdictPath   string                   `json:"provider_verdict_path"`
	ProviderVerdictSHA256 string                   `json:"provider_verdict_sha256"`
	Verdict               string                   `json:"verdict"`
	Summary               string                   `json:"summary,omitempty"`
	CheckedPaths          []string                 `json:"checked_paths"`
	FixedPaths            []string                 `json:"fixed_paths"`
	Findings              []Finding                `json:"findings,omitempty"`
	Questions             []Question               `json:"questions,omitempty"`
	TechnicalIssues       []ValidatorIssue         `json:"technical_issues"`
	AdvisoryIssues        []AdvisoryValidatorIssue `json:"advisory_issues"`
	Audit                 EffectiveAuditSummary    `json:"audit"`
}

type AdvisoryValidatorIssue struct {
	Source     string `json:"source"`
	MatchKey   string `json:"match_key"`
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Path       string `json:"path,omitempty"`
	DocumentID string `json:"document_id,omitempty"`
	CitationID string `json:"citation_id,omitempty"`
}

type EffectiveAuditSummary struct {
	Status       string   `json:"status"`
	ErrorCount   int      `json:"error_count"`
	WarningCount int      `json:"warning_count"`
	IssueCodes   []string `json:"issue_codes"`
}

func ParseShardPackManifest(raw []byte) (ShardPackManifest, error) {
	if err := validation.ValidateRawJSON(validation.ShardPackManifestSchema, raw); err != nil {
		return ShardPackManifest{}, fmt.Errorf("shard pack manifest is invalid: %w", err)
	}
	var manifest ShardPackManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return ShardPackManifest{}, fmt.Errorf("decode shard pack manifest: %w", err)
	}
	if err := validateShardPackManifest(manifest); err != nil {
		return ShardPackManifest{}, err
	}
	return manifest, nil
}

func ParseFinalRunIndex(raw []byte) (FinalRunIndex, error) {
	if err := validation.ValidateRawJSON(validation.FinalRunIndexSchema, raw); err != nil {
		return FinalRunIndex{}, fmt.Errorf("final run index is invalid: %w", err)
	}
	var index FinalRunIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return FinalRunIndex{}, fmt.Errorf("decode final run index: %w", err)
	}
	if err := validateFinalRunIndex(index); err != nil {
		return FinalRunIndex{}, err
	}
	return index, nil
}

func ParseCitationIndex(raw []byte) (CitationIndex, error) {
	if err := validation.ValidateRawJSON(validation.CitationIndexSchema, raw); err != nil {
		return CitationIndex{}, fmt.Errorf("citation index is invalid: %w", err)
	}
	var index CitationIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return CitationIndex{}, fmt.Errorf("decode citation index: %w", err)
	}
	if err := validateCitationIndex(index); err != nil {
		return CitationIndex{}, err
	}
	return index, nil
}

func ParseValidatorVerdict(raw []byte) (ValidatorVerdict, error) {
	if err := validation.ValidateRawJSON(validation.ValidatorVerdictSchema, raw); err != nil {
		return ValidatorVerdict{}, fmt.Errorf("validator verdict is invalid: %w", err)
	}
	var verdict ValidatorVerdict
	if err := json.Unmarshal(raw, &verdict); err != nil {
		return ValidatorVerdict{}, fmt.Errorf("decode validator verdict: %w", err)
	}
	if err := validateValidatorVerdict(verdict); err != nil {
		return ValidatorVerdict{}, err
	}
	return verdict, nil
}

func ParseEffectiveVerdict(raw []byte) (EffectiveVerdict, error) {
	if err := validation.ValidateRawJSON(validation.EffectiveVerdictSchema, raw); err != nil {
		return EffectiveVerdict{}, fmt.Errorf("effective verdict is invalid: %w", err)
	}
	var verdict EffectiveVerdict
	if err := json.Unmarshal(raw, &verdict); err != nil {
		return EffectiveVerdict{}, fmt.Errorf("decode effective verdict: %w", err)
	}
	if err := validateEffectiveVerdict(verdict); err != nil {
		return EffectiveVerdict{}, err
	}
	return verdict, nil
}

func validateShardPackManifest(manifest ShardPackManifest) error {
	problems := []string{}
	if strings.TrimSpace(manifest.RunID) == "" {
		problems = append(problems, "run_id is required")
	}
	if strings.TrimSpace(manifest.StepID) == "" {
		problems = append(problems, "step_id is required")
	}
	if strings.TrimSpace(manifest.ShardID) == "" {
		problems = append(problems, "shard_id is required")
	}
	if strings.TrimSpace(manifest.ArtifactRoot) == "" {
		problems = append(problems, "artifact_root is required")
	}
	if len(manifest.Documents) == 0 {
		problems = append(problems, "documents must contain at least one authored document")
	}
	if len(manifest.Citations) == 0 {
		problems = append(problems, "citations must contain at least one citation")
	}
	problems = append(problems, validateDocumentSet(manifest.ArtifactRoot, manifest.Documents, manifest.Citations)...)
	problems = append(problems, validateCitationSet(manifest.Citations)...)
	problems = append(problems, validateDocumentCitationSymmetry(manifest.Documents, manifest.Citations)...)
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("shard pack manifest is invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}

func validateFinalRunIndex(index FinalRunIndex) error {
	problems := []string{}
	if strings.TrimSpace(index.RunID) == "" {
		problems = append(problems, "run_id is required")
	}
	if strings.TrimSpace(index.Pipeline) == "" {
		problems = append(problems, "pipeline is required")
	}
	if strings.TrimSpace(index.CitationIndexPath) == "" {
		problems = append(problems, "citation_index_path is required")
	}
	seenDocIDs := map[string]struct{}{}
	seenCanonicalPaths := map[string]struct{}{}
	for idx, doc := range index.CanonicalDocuments {
		label := fmt.Sprintf("canonical_documents[%d]", idx)
		if strings.TrimSpace(doc.ID) == "" {
			problems = append(problems, label+".id is required")
		}
		if strings.TrimSpace(doc.CanonicalPath) == "" {
			problems = append(problems, label+".canonical_path is required")
		}
		if strings.TrimSpace(doc.StagedPath) == "" {
			problems = append(problems, label+".staged_path is required")
		}
		if _, exists := seenDocIDs[doc.ID]; exists && strings.TrimSpace(doc.ID) != "" {
			problems = append(problems, label+".id must be unique")
		}
		seenDocIDs[doc.ID] = struct{}{}
		if _, exists := seenCanonicalPaths[doc.CanonicalPath]; exists && strings.TrimSpace(doc.CanonicalPath) != "" {
			problems = append(problems, label+".canonical_path must be unique")
		}
		seenCanonicalPaths[doc.CanonicalPath] = struct{}{}
	}
	for idx, topic := range index.Topics {
		label := fmt.Sprintf("topics[%d]", idx)
		if strings.TrimSpace(topic.ID) == "" {
			problems = append(problems, label+".id is required")
		}
		for _, documentID := range topic.DocumentIDs {
			if _, ok := seenDocIDs[documentID]; !ok {
				problems = append(problems, fmt.Sprintf("%s references unknown document_id %q", label, documentID))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("final run index is invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}

func validateCitationIndex(index CitationIndex) error {
	problems := validateCitationSet(index.Citations)
	if strings.TrimSpace(index.RunID) == "" {
		problems = append(problems, "run_id is required")
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("citation index is invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}

func validateValidatorVerdict(verdict ValidatorVerdict) error {
	problems := []string{}
	if strings.TrimSpace(verdict.RunID) == "" {
		problems = append(problems, "run_id is required")
	}
	switch strings.TrimSpace(verdict.Verdict) {
	case "PASS", "FAIL":
	default:
		problems = append(problems, "verdict must be PASS or FAIL")
	}
	if len(verdict.CheckedPaths) == 0 {
		problems = append(problems, "checked_paths is required")
	}
	for idx, issue := range verdict.Issues {
		label := fmt.Sprintf("issues[%d]", idx)
		if strings.TrimSpace(issue.Code) == "" {
			problems = append(problems, label+".code is required")
		}
		if strings.TrimSpace(issue.Message) == "" {
			problems = append(problems, label+".message is required")
		}
		switch strings.TrimSpace(issue.Severity) {
		case "error", "warning":
		default:
			problems = append(problems, label+".severity must be error or warning")
		}
	}
	for idx, finding := range verdict.Findings {
		label := fmt.Sprintf("findings[%d]", idx)
		if strings.TrimSpace(finding.ID) == "" {
			problems = append(problems, label+".id is required")
		}
		if strings.TrimSpace(finding.Title) == "" {
			problems = append(problems, label+".title is required")
		}
		if strings.TrimSpace(finding.Severity) == "" {
			problems = append(problems, label+".severity is required")
		}
	}
	for idx, question := range verdict.Questions {
		label := fmt.Sprintf("questions[%d]", idx)
		if strings.TrimSpace(question.ID) == "" {
			problems = append(problems, label+".id is required")
		}
		if strings.TrimSpace(question.Text) == "" {
			problems = append(problems, label+".text is required")
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("validator verdict is invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}

func validateEffectiveVerdict(verdict EffectiveVerdict) error {
	problems := []string{}
	if verdict.RunID == "" {
		problems = append(problems, "run_id is required")
	}
	if verdict.Kind != "effective" {
		problems = append(problems, "kind must be effective")
	}
	if verdict.Authority != "orchestrator" {
		problems = append(problems, "authority must be orchestrator")
	}
	if verdict.ProviderVerdictPath == "" {
		problems = append(problems, "provider_verdict_path is required")
	}
	if verdict.ProviderVerdictSHA256 == "" {
		problems = append(problems, "provider_verdict_sha256 is required")
	}
	if verdict.Verdict != "PASS" && verdict.Verdict != "FAIL" {
		problems = append(problems, "verdict must be PASS or FAIL")
	}
	if len(verdict.CheckedPaths) == 0 {
		problems = append(problems, "checked_paths is required")
	}
	if verdict.Audit.Status != "pass" && verdict.Audit.Status != "warn" && verdict.Audit.Status != "fail" {
		problems = append(problems, "audit.status must be pass, warn or fail")
	}
	for idx, issue := range verdict.TechnicalIssues {
		if issue.Code == "" || issue.Message == "" || (issue.Severity != "error" && issue.Severity != "warning") {
			problems = append(problems, fmt.Sprintf("technical_issues[%d] has invalid code, severity or message", idx))
		}
	}
	for idx, issue := range verdict.AdvisoryIssues {
		if issue.Source != "provider" || issue.MatchKey == "" || issue.Code == "" || issue.Message == "" || issue.Severity != "warning" {
			problems = append(problems, fmt.Sprintf("advisory_issues[%d] has invalid source, match_key, severity or message", idx))
		}
	}
	if verdict.Verdict == "PASS" {
		for _, issue := range verdict.TechnicalIssues {
			if issue.Severity == "error" {
				problems = append(problems, "PASS effective verdict cannot contain technical error issues")
				break
			}
		}
	}
	if verdict.Verdict == "FAIL" {
		hasError := false
		for _, issue := range verdict.TechnicalIssues {
			if issue.Severity == "error" {
				hasError = true
				break
			}
		}
		if !hasError {
			problems = append(problems, "FAIL effective verdict must contain a technical error")
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("effective verdict is invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}

func validateDocumentSet(artifactRoot string, documents []AuthoredDocument, citations []DocumentCitation) []string {
	problems := []string{}
	citationIDs := map[string]struct{}{}
	for _, citation := range citations {
		if strings.TrimSpace(citation.ID) == "" {
			continue
		}
		citationIDs[citation.ID] = struct{}{}
	}
	seen := map[string]struct{}{}
	for idx, doc := range documents {
		label := fmt.Sprintf("documents[%d]", idx)
		if strings.TrimSpace(doc.ID) == "" {
			problems = append(problems, label+".id is required")
		}
		if strings.TrimSpace(doc.Path) == "" {
			problems = append(problems, label+".path is required")
		}
		if strings.TrimSpace(doc.CanonicalPath) == "" {
			problems = append(problems, label+".canonical_path is required")
		} else if !isAllowedCanonicalRuntimeDocumentPath(doc.CanonicalPath) {
			problems = append(problems, label+".canonical_path must be within reports/as-is, reports/findings, reports/coverage, reports/agent-outputs, reports/diagrams, or proposals")
		}
		if pathErr := validateShardDocumentRelativePath(doc.Path, artifactRoot); pathErr != "" {
			problems = append(problems, label+".path "+pathErr)
		}
		if _, exists := seen[doc.ID]; exists && strings.TrimSpace(doc.ID) != "" {
			problems = append(problems, label+".id must be unique")
		}
		seen[doc.ID] = struct{}{}
		if len(doc.CitationIDs) == 0 {
			problems = append(problems, label+".citation_ids is required")
		}
		for _, citationID := range doc.CitationIDs {
			if _, ok := citationIDs[citationID]; !ok {
				problems = append(problems, fmt.Sprintf("%s references unknown citation_id %q", label, citationID))
			}
		}
	}
	return problems
}

func validateShardDocumentRelativePath(rawPath string, artifactRoot string) string {
	cleaned := filepath.Clean(filepath.FromSlash(strings.TrimSpace(rawPath)))
	if cleaned == "." || cleaned == "" {
		return "must not be empty"
	}
	if filepath.IsAbs(cleaned) {
		return "must be relative"
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "must not escape artifact_root"
	}
	if workspacePrefix := forbiddenWorkspaceLevelDocumentPrefix(cleaned); workspacePrefix != "" {
		return fmt.Sprintf("must be artifact_root-relative, not workspace-level path starting with %q", workspacePrefix)
	}
	if component := forbiddenProviderToolDocumentComponent(cleaned); component != "" {
		return fmt.Sprintf("must not reference provider/tool side-effect path component %q", component)
	}
	if duplicatedArtifactRootPrefix(cleaned, artifactRoot) {
		return "must be artifact_root-relative and must not repeat artifact_root"
	}
	return ""
}

func forbiddenWorkspaceLevelDocumentPrefix(cleanedPath string) string {
	normalized := strings.ToLower(filepath.ToSlash(strings.TrimSpace(cleanedPath)))
	for _, prefix := range []string{"reports/", "charter/", "proposals/"} {
		if strings.HasPrefix(normalized, prefix) {
			return prefix
		}
	}
	return ""
}

func forbiddenProviderToolDocumentComponent(cleanedPath string) string {
	normalized := filepath.ToSlash(strings.TrimSpace(cleanedPath))
	for _, component := range strings.Split(normalized, "/") {
		trimmed := strings.TrimSpace(strings.ToLower(component))
		switch trimmed {
		case ".qwen", ".claude", ".codex", ".git", ".hg", ".svn", "node_modules":
			return component
		}
	}
	return ""
}

func duplicatedArtifactRootPrefix(cleanedPath string, artifactRoot string) bool {
	normalizedArtifactRoot := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(artifactRoot))))
	if normalizedArtifactRoot == "." || normalizedArtifactRoot == "" {
		return false
	}
	normalizedPath := filepath.ToSlash(cleanedPath)
	return normalizedPath == normalizedArtifactRoot || strings.HasPrefix(normalizedPath, normalizedArtifactRoot+"/")
}

func isAllowedCanonicalRuntimeDocumentPath(rawPath string) bool {
	canonicalPath := filepath.ToSlash(strings.TrimSpace(rawPath))
	if canonicalPath == "" {
		return false
	}
	allowedPrefixes := []string{
		"reports/as-is/",
		"reports/findings/",
		"reports/coverage/",
		"reports/agent-outputs/",
		"reports/diagrams/",
		"proposals/",
	}
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(canonicalPath, prefix) {
			return true
		}
	}
	return false
}

func validateCitationSet(citations []DocumentCitation) []string {
	problems := []string{}
	seen := map[string]struct{}{}
	for idx, citation := range citations {
		label := fmt.Sprintf("citations[%d]", idx)
		if strings.TrimSpace(citation.ID) == "" {
			problems = append(problems, label+".id is required")
		}
		if strings.TrimSpace(citation.Repo) == "" {
			problems = append(problems, label+".repo is required")
		}
		if strings.TrimSpace(citation.Path) == "" {
			problems = append(problems, label+".path is required")
		}
		if len(citation.ClaimIDs) == 0 {
			problems = append(problems, label+".claim_ids is required")
		}
		if len(citation.DocumentIDs) == 0 {
			problems = append(problems, label+".document_ids is required")
		}
		if _, exists := seen[citation.ID]; exists && strings.TrimSpace(citation.ID) != "" {
			problems = append(problems, label+".id must be unique")
		}
		seen[citation.ID] = struct{}{}
	}
	return problems
}

func validateDocumentCitationSymmetry(documents []AuthoredDocument, citations []DocumentCitation) []string {
	problems := []string{}
	documentsByID := map[string]AuthoredDocument{}
	citationsByID := map[string]DocumentCitation{}
	for _, doc := range documents {
		docID := strings.TrimSpace(doc.ID)
		if docID == "" {
			continue
		}
		if _, exists := documentsByID[docID]; exists {
			continue
		}
		documentsByID[docID] = doc
	}
	for _, citation := range citations {
		citationID := strings.TrimSpace(citation.ID)
		if citationID == "" {
			continue
		}
		if _, exists := citationsByID[citationID]; exists {
			continue
		}
		citationsByID[citationID] = citation
	}

	for idx, doc := range documents {
		docID := strings.TrimSpace(doc.ID)
		if docID == "" {
			continue
		}
		label := fmt.Sprintf("documents[%d]", idx)
		for _, rawCitationID := range doc.CitationIDs {
			citationID := strings.TrimSpace(rawCitationID)
			if citationID == "" {
				continue
			}
			citation, ok := citationsByID[citationID]
			if !ok {
				continue
			}
			if !containsStringValue(citation.DocumentIDs, docID) {
				problems = append(problems, fmt.Sprintf("%s references citation_id %q but citation does not list document_id %q", label, citationID, docID))
			}
		}
	}

	for idx, citation := range citations {
		citationID := strings.TrimSpace(citation.ID)
		if citationID == "" {
			continue
		}
		label := fmt.Sprintf("citations[%d]", idx)
		for _, rawDocumentID := range citation.DocumentIDs {
			documentID := strings.TrimSpace(rawDocumentID)
			if documentID == "" {
				continue
			}
			doc, ok := documentsByID[documentID]
			if !ok {
				problems = append(problems, fmt.Sprintf("%s references unknown document_id %q", label, documentID))
				continue
			}
			if !containsStringValue(doc.CitationIDs, citationID) {
				problems = append(problems, fmt.Sprintf("%s references document_id %q but document does not list citation_id %q", label, documentID, citationID))
			}
		}
	}
	return problems
}

func containsStringValue(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
