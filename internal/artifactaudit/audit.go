package artifactaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/evidence"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const (
	Version          = 1
	MaxIssues        = 200
	MaxArtifacts     = 2_000
	MaxArtifactBytes = 1 << 20
	MaxMessageBytes  = 320
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Issue struct {
	Code         string   `json:"code"`
	Severity     string   `json:"severity"`
	Path         string   `json:"path,omitempty"`
	RelatedPaths []string `json:"related_paths"`
	Message      string   `json:"message"`
}

type Artifact struct {
	Path   string `json:"path"`
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
}

type Summary struct {
	Error    int `json:"error"`
	Warning  int `json:"warning"`
	Artifact int `json:"artifact"`
}

type Report struct {
	Version              int        `json:"version"`
	RunID                string     `json:"run_id"`
	Scope                string     `json:"scope"`
	Status               Status     `json:"status"`
	ProviderVerdictPath  string     `json:"provider_verdict_path,omitempty"`
	EffectiveVerdictPath string     `json:"effective_verdict_path,omitempty"`
	EffectiveAuthority   string     `json:"effective_authority"`
	Summary              Summary    `json:"summary"`
	Issues               []Issue    `json:"issues"`
	Artifacts            []Artifact `json:"artifacts"`
	Truncated            bool       `json:"truncated"`
}

func ScanSelectedRun(ws workspace.Root, runID string) Report {
	return scan(ws, runID, "selected_run", nil)
}

// ScanSelectedRunPublic is the public authority surface. New runs must carry
// an effective verdict; provider-only historical runs are explicit legacy /
// unavailable states and never get an inferred PASS.
func ScanSelectedRunPublic(ws workspace.Root, runID string) Report {
	runID = strings.TrimSpace(runID)
	effectivePath := path.Join("reports", "taskruns", runID, "validator", "effective-verdict.json")
	raw, err := ws.ReadFileLimit(effectivePath, MaxArtifactBytes)
	if err != nil {
		report := ScanSelectedRun(ws, runID)
		report.EffectiveVerdictPath = effectivePath
		report.EffectiveAuthority = "legacy_unavailable"
		report.Issues = append(report.Issues, Issue{Code: "audit.effective_verdict.unavailable", Severity: "error", Path: effectivePath, RelatedPaths: []string{}, Message: "effective technical verdict is unavailable; provider-only history is legacy evidence"})
		finalizeReport(&report)
		return report
	}
	effective, err := contracts.ParseEffectiveVerdict(raw)
	providerPath := path.Join("reports", "taskruns", runID, "validator", "validator-verdict.json")
	providerRaw, providerErr := ws.ReadFileLimit(providerPath, MaxArtifactBytes)
	providerDigestMatches := providerErr == nil && sha256Hex(providerRaw) == effective.ProviderVerdictSHA256
	if err != nil || effective.RunID != runID || effective.ProviderVerdictPath != providerPath || !providerDigestMatches {
		report := ScanSelectedRun(ws, runID)
		report.EffectiveVerdictPath = effectivePath
		report.EffectiveAuthority = "invalid"
		report.Issues = append(report.Issues, Issue{Code: "audit.effective_verdict.invalid", Severity: "error", Path: effectivePath, RelatedPaths: []string{}, Message: "effective technical verdict is invalid or belongs to another run"})
		finalizeReport(&report)
		return report
	}
	candidate := contracts.ValidatorVerdict{
		Version: 1, RunID: effective.RunID, GeneratedAt: effective.GeneratedAt, Verdict: effective.Verdict,
		Summary: effective.Summary, CheckedPaths: append([]string(nil), effective.CheckedPaths...),
		FixedPaths: append([]string(nil), effective.FixedPaths...), Findings: append([]contracts.Finding(nil), effective.Findings...),
		Questions: append([]contracts.Question(nil), effective.Questions...), Issues: append([]contracts.ValidatorIssue(nil), effective.TechnicalIssues...),
	}
	report := scan(ws, runID, "selected_run", &candidate)
	report.EffectiveVerdictPath = effectivePath
	report.EffectiveAuthority = "effective"
	if len(report.Artifacts) < MaxArtifacts {
		sum := sha256.Sum256(raw)
		report.Artifacts = append(report.Artifacts, Artifact{Path: effectivePath, Size: len(raw), SHA256: hex.EncodeToString(sum[:])})
	}
	finalizeReport(&report)
	return report
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// ScanSelectedRunWithCandidate uses the same provider-free scanner against an
// in-memory orchestrator technical candidate before effective-verdict
// persistence. This keeps audit authority acyclic.
func ScanSelectedRunWithCandidate(ws workspace.Root, runID string, candidate contracts.ValidatorVerdict) Report {
	return scan(ws, runID, "selected_run", &candidate)
}

func ScanPromotedRun(ws workspace.Root, runID string) Report {
	return scan(ws, runID, "promoted_current", nil)
}

func scan(ws workspace.Root, runID string, scope string, candidate *contracts.ValidatorVerdict) Report {
	runID = strings.TrimSpace(runID)
	audit := auditor{
		ws:          ws,
		runID:       runID,
		report:      Report{Version: Version, RunID: runID, Scope: scope, Status: StatusPass, EffectiveAuthority: "not_requested", Issues: []Issue{}, Artifacts: []Artifact{}},
		documents:   map[string]contracts.FinalRunDocument{},
		citations:   map[string]contracts.DocumentCitation{},
		repoSources: map[string]workspace.RepoSource{},
		candidate:   candidate,
	}
	for _, source := range ws.Manifest.Repos {
		audit.repoSources[source.Name] = source
	}
	audit.scan()
	if scope == "promoted_current" {
		audit.scanPromoted()
	}
	audit.finish()
	return audit.report
}

type auditor struct {
	ws          workspace.Root
	runID       string
	report      Report
	documents   map[string]contracts.FinalRunDocument
	citations   map[string]contracts.DocumentCitation
	repoSources map[string]workspace.RepoSource
	candidate   *contracts.ValidatorVerdict
}

func (a *auditor) scan() {
	if a.runID == "" || strings.Contains(a.runID, "/") || strings.Contains(a.runID, `\`) || a.runID == "." || a.runID == ".." {
		a.add("audit.run_id.invalid", "error", "", nil, "run identity is invalid")
		return
	}
	finalRoot := path.Join("reports", "taskruns", a.runID, "staging", "final")
	finalPath := path.Join(finalRoot, "final-run-index.json")
	finalRaw, err := a.ws.ReadFileLimit(finalPath, MaxArtifactBytes)
	if err != nil {
		a.add("audit.final_index.unavailable", "error", finalPath, nil, "selected-run final index is unavailable or oversized")
		return
	}
	index, err := contracts.ParseFinalRunIndex(finalRaw)
	if err != nil {
		a.add("audit.final_index.invalid", "error", finalPath, nil, "selected-run final index is invalid")
		return
	}
	if index.RunID != a.runID {
		a.add("audit.run_identity.foreign", "error", finalPath, nil, "final index belongs to a different run")
		return
	}
	a.record(finalPath, finalRaw)

	citationPath := path.Clean(index.CitationIndexPath)
	if citationPath != path.Join(finalRoot, "citation-index.json") {
		a.add("audit.citation_index.foreign", "error", citationPath, []string{finalPath}, "citation index path is outside the selected final snapshot")
		return
	}
	citationRaw, err := a.ws.ReadFileLimit(citationPath, MaxArtifactBytes)
	if err != nil {
		a.add("audit.citation_index.unavailable", "error", citationPath, nil, "citation index is unavailable or oversized")
		return
	}
	citationIndex, err := contracts.ParseCitationIndex(citationRaw)
	if err != nil || citationIndex.RunID != a.runID {
		a.add("audit.citation_index.invalid", "error", citationPath, nil, "citation index is invalid or belongs to another run")
		return
	}
	a.record(citationPath, citationRaw)
	for _, citation := range citationIndex.Citations {
		a.citations[citation.ID] = citation
	}
	verdictPath := path.Join("reports", "taskruns", a.runID, "validator", "validator-verdict.json")
	a.report.ProviderVerdictPath = verdictPath
	verdictRaw, err := a.ws.ReadFileLimit(verdictPath, MaxArtifactBytes)
	var verdict contracts.ValidatorVerdict
	if a.candidate != nil {
		verdict = *a.candidate
		if err == nil {
			a.record(verdictPath, verdictRaw)
		}
	} else {
		if err != nil {
			a.add("audit.validator.unavailable", "error", verdictPath, nil, "validator verdict is unavailable or oversized")
			return
		}
		verdict, err = contracts.ParseValidatorVerdict(verdictRaw)
		if err != nil || verdict.RunID != a.runID || strings.ToUpper(verdict.Verdict) != "PASS" {
			a.add("audit.validator.not_promoted", "error", verdictPath, nil, "validator verdict is invalid, foreign or not PASS")
			return
		}
		a.record(verdictPath, verdictRaw)
	}
	if verdict.RunID != a.runID {
		a.add("audit.validator.run_identity_foreign", "error", verdictPath, nil, "technical candidate belongs to a different run")
		return
	}
	a.scanValidatorPaths(verdict, finalRoot, finalPath, citationPath)

	for _, document := range index.CanonicalDocuments {
		a.documents[document.ID] = document
		a.scanDocument(finalRoot, document)
	}
	a.scanReciprocalReferences()
	for _, citation := range citationIndex.Citations {
		a.scanEvidence(citation)
	}
	for _, entity := range index.Semantic.Entities {
		for _, source := range entity.Provenance.Evidence {
			if evidence.HasBoundedEvidence(source.Lines, source.Excerpt, source.ExcerptHash) {
				a.scanSourceEvidence(source.Repo, source.Path, source.Lines, source.Excerpt, source.ExcerptHash)
			}
		}
	}
	for _, edge := range index.Semantic.Edges {
		for _, source := range edge.Provenance.Evidence {
			if evidence.HasBoundedEvidence(source.Lines, source.Excerpt, source.ExcerptHash) {
				a.scanSourceEvidence(source.Repo, source.Path, source.Lines, source.Excerpt, source.ExcerptHash)
			}
		}
	}
	for _, finding := range index.Semantic.Findings {
		for _, source := range finding.Provenance.Evidence {
			if evidence.HasBoundedEvidence(source.Lines, source.Excerpt, source.ExcerptHash) {
				a.scanSourceEvidence(source.Repo, source.Path, source.Lines, source.Excerpt, source.ExcerptHash)
			}
		}
	}
	if _, ok := findCanonicalDocument(index.CanonicalDocuments, "reports/as-is/overview.md"); !ok {
		a.add("audit.architecture_home.missing", "error", "reports/as-is/overview.md", nil, "Architecture Home is missing from the selected final index")
	}
}

func (a *auditor) scanPromoted() {
	for _, document := range a.documents {
		staged, err := a.ws.ReadFileLimit(document.StagedPath, MaxArtifactBytes)
		if err != nil {
			continue
		}
		current, err := a.ws.ReadFileLimit(document.CanonicalPath, MaxArtifactBytes)
		if err != nil {
			a.add("audit.promoted.document_missing", "error", document.CanonicalPath, []string{document.StagedPath}, "promoted canonical document is unavailable")
			continue
		}
		stagedDigest := sha256.Sum256(staged)
		currentDigest := sha256.Sum256(current)
		if stagedDigest != currentDigest {
			a.add("audit.promoted.digest_mismatch", "error", document.CanonicalPath, []string{document.StagedPath}, "promoted canonical bytes differ from selected-run staging")
		}
	}
}

func (a *auditor) scanDocument(finalRoot string, document contracts.FinalRunDocument) {
	staged := path.Clean(document.StagedPath)
	if staged == "." || !strings.HasPrefix(staged, finalRoot+"/") {
		a.add("audit.document.staged_path_escape", "error", staged, []string{document.CanonicalPath}, "document staged path escapes the selected final snapshot")
		return
	}
	raw, err := a.ws.ReadFileLimit(staged, MaxArtifactBytes)
	if err != nil {
		a.add("audit.document.unavailable", "error", staged, []string{document.CanonicalPath}, "indexed document is unavailable or oversized")
		return
	}
	a.record(staged, raw)
	text := strings.ToLower(string(raw))
	if containsAny(text, "reports/taskruns/", ".acp/repos/", "draft_artifact_enrichment", "current run shard", "runtime recovery") {
		a.add("audit.document.execution_contamination", "error", staged, nil, "document contains internal execution or taskrun narration")
	}
	if containsAny(text, "drafted required runtime artifacts", "recovery bootstrap", "replace placeholder content") {
		a.add("audit.document.scaffold", "error", staged, nil, "document contains scaffold or recovery placeholder text")
	}
	if document.CanonicalPath == "reports/as-is/overview.md" {
		a.scanArchitectureHome(staged, string(raw))
	}
	if strings.HasPrefix(document.CanonicalPath, "proposals/") && strings.HasSuffix(document.CanonicalPath, "proposal.md") {
		if !containsAny(text, "## decision", "## recommended", "## proposed changes", "## follow-up plan") {
			a.add("audit.proposal.not_actionable", "warning", staged, nil, "proposal lacks a decision or concrete follow-up section")
		}
	}
}

func (a *auditor) scanArchitectureHome(staged string, text string) {
	lower := strings.ToLower(text)
	for _, heading := range runtimedrafts.ArchitectureHomeRequiredSections() {
		if !strings.Contains(lower, strings.ToLower("## "+heading)) {
			a.add("audit.architecture_home.section_missing", "error", staged, nil, "Architecture Home is missing required section: "+heading)
		}
	}
}

func (a *auditor) scanReciprocalReferences() {
	for _, document := range a.documents {
		for _, citationID := range document.CitationIDs {
			citation, ok := a.citations[citationID]
			if !ok {
				a.add("audit.reference.citation_missing", "error", document.StagedPath, nil, "document references an absent citation")
				continue
			}
			if !contains(citation.DocumentIDs, document.ID) {
				a.add("audit.reference.not_reciprocal", "error", document.StagedPath, nil, "document/citation relationship is not reciprocal")
			}
		}
	}
	for _, citation := range a.citations {
		for _, documentID := range citation.DocumentIDs {
			document, ok := a.documents[documentID]
			if !ok {
				a.add("audit.reference.document_missing", "error", "", nil, "citation references an absent document")
				continue
			}
			if !contains(document.CitationIDs, citation.ID) {
				a.add("audit.reference.not_reciprocal", "error", document.StagedPath, nil, "citation/document relationship is not reciprocal")
			}
		}
	}
}

func (a *auditor) scanEvidence(citation contracts.DocumentCitation) {
	a.scanSourceEvidence(citation.Repo, citation.Path, citation.Lines, citation.Excerpt, citation.ExcerptHash)
}

func (a *auditor) scanSourceEvidence(repoName, evidencePath string, lines *contracts.LineRange, excerpt, excerptHash string) {
	source, ok := a.repoSources[repoName]
	if !ok {
		a.add("audit.evidence.repo_unknown", "error", evidencePath, nil, "evidence references an unknown repository")
		return
	}
	repoRoot := strings.TrimSpace(source.Path)
	if repoRoot == "" {
		a.add("audit.evidence.repo_unavailable", "warning", evidencePath, nil, "evidence repository is not available as a local path")
		return
	}
	if !filepath.IsAbs(repoRoot) {
		repoRoot = filepath.Join(a.ws.Path, repoRoot)
	}
	rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(evidencePath)))
	if rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		a.add("audit.evidence.path_escape", "error", evidencePath, nil, "evidence path escapes its repository")
		return
	}
	resolvedRoot, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		a.add("audit.evidence.repo_unavailable", "warning", evidencePath, nil, "evidence repository cannot be resolved")
		return
	}
	resolvedEvidence, err := filepath.EvalSymlinks(filepath.Join(resolvedRoot, rel))
	if err != nil {
		a.add("audit.evidence.file_missing", "error", evidencePath, nil, "evidence file does not exist")
		return
	}
	contained, err := filepath.Rel(resolvedRoot, resolvedEvidence)
	if err != nil || contained == ".." || strings.HasPrefix(contained, ".."+string(filepath.Separator)) || filepath.IsAbs(contained) {
		a.add("audit.evidence.path_escape", "error", evidencePath, nil, "evidence resolves outside its repository")
		return
	}
	info, err := os.Stat(resolvedEvidence)
	if err != nil {
		a.add("audit.evidence.not_regular", "error", evidencePath, nil, "evidence cannot be stat'ed")
		return
	}
	if !info.Mode().IsRegular() {
		a.add("audit.evidence.not_regular", "error", evidencePath, nil, "evidence is not a regular file")
		return
	}
	if lines == nil && excerpt == "" && strings.TrimSpace(excerptHash) == "" {
		return
	}
	if err := evidence.ValidateFile(resolvedEvidence, lines, excerpt, excerptHash); err != nil {
		a.add("audit."+evidence.Code(err), "error", evidencePath, nil, "evidence bytes failed bounded validation: "+err.Error())
	}
}

func (a *auditor) scanValidatorPaths(verdict contracts.ValidatorVerdict, finalRoot, finalPath, citationPath string) {
	seen := map[string]struct{}{}
	hasFinal, hasCitation := false, false
	taskRoot := path.Join("reports", "taskruns", a.runID) + "/"
	for _, raw := range verdict.CheckedPaths {
		clean, ok := a.normalizeValidatorPath(raw, taskRoot)
		if !ok {
			clean = filepath.ToSlash(path.Clean(strings.TrimSpace(raw)))
			a.add("audit.validator.checked_path_foreign", "error", clean, nil, "validator checked path is outside the selected run")
			continue
		}
		if _, exists := seen[clean]; exists {
			a.add("audit.validator.checked_path_duplicate", "error", clean, nil, "validator checked paths must be unique")
		}
		seen[clean] = struct{}{}
		hasFinal = hasFinal || clean == finalPath
		hasCitation = hasCitation || clean == citationPath
	}
	if !hasFinal {
		a.add("audit.validator.checked_path_missing_final", "error", finalPath, nil, "validator checked_paths must include the selected final index")
	}
	if !hasCitation {
		a.add("audit.validator.checked_path_missing_citation", "error", citationPath, nil, "validator checked_paths must include the selected citation index")
	}
	for _, raw := range verdict.FixedPaths {
		clean, ok := a.normalizeValidatorPath(raw, taskRoot)
		if !ok {
			clean = filepath.ToSlash(path.Clean(strings.TrimSpace(raw)))
			a.add("audit.validator.fixed_path_foreign", "error", clean, []string{finalPath}, "provider fixed_paths must remain inside the selected run")
		}
	}
}

// normalizeValidatorPath accepts the logical run-relative paths used by the
// contract and absolute paths emitted by some provider CLIs. Absolute paths
// are accepted only when they resolve inside this workspace's selected run;
// the returned value is always the canonical logical path used by the audit.
// Resolving both sides handles macOS aliases such as /tmp -> /private/tmp
// without weakening containment checks for symlink escapes.
func (a *auditor) normalizeValidatorPath(raw, taskRoot string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.Contains(trimmed, `\`) {
		return "", false
	}
	clean := filepath.Clean(filepath.FromSlash(trimmed))
	if clean == "." {
		return "", false
	}
	if !filepath.IsAbs(clean) {
		logical := filepath.ToSlash(path.Clean(trimmed))
		if logical == "." || logical == ".." || strings.HasPrefix(logical, "../") || !strings.HasPrefix(logical, taskRoot) {
			return "", false
		}
		return logical, true
	}

	workspaceRoot := filepath.Clean(a.ws.Path)
	resolvedWorkspaceRoot, err := resolvePathForContainment(workspaceRoot)
	if err != nil {
		return "", false
	}
	resolvedCandidate, err := resolvePathForContainment(clean)
	if err != nil {
		return "", false
	}
	runRel := filepath.FromSlash(strings.TrimSuffix(taskRoot, "/"))
	runRoot := filepath.Join(resolvedWorkspaceRoot, runRel)
	rel, err := filepath.Rel(runRoot, resolvedCandidate)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return path.Join(taskRoot, filepath.ToSlash(rel)), true
}

// resolvePathForContainment resolves an absolute path even when its leaf is
// not present yet by resolving the nearest existing parent and appending the
// missing suffix. This prevents a symlinked directory inside the selected run
// from making a foreign absolute path look lexically contained.
func resolvePathForContainment(value string) (string, error) {
	clean := filepath.Clean(value)
	current := clean
	suffix := []string{}
	for {
		if _, err := os.Lstat(current); err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for index := len(suffix) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, suffix[index])
			}
			return filepath.Clean(resolved), nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", os.ErrNotExist
		}
		suffix = append(suffix, filepath.Base(current))
		current = parent
	}
}

func (a *auditor) record(filePath string, raw []byte) {
	if len(a.report.Artifacts) >= MaxArtifacts {
		a.markTruncated()
		return
	}
	sum := sha256.Sum256(raw)
	a.report.Artifacts = append(a.report.Artifacts, Artifact{
		Path: filePath, Size: len(raw), SHA256: hex.EncodeToString(sum[:]),
	})
}

func (a *auditor) add(code, severity, filePath string, related []string, message string) {
	if len(a.report.Issues) >= MaxIssues-1 {
		a.markTruncated()
		return
	}
	if len(message) > MaxMessageBytes {
		message = message[:MaxMessageBytes]
	}
	a.report.Issues = append(a.report.Issues, Issue{
		Code: code, Severity: severity, Path: filepath.ToSlash(filePath),
		RelatedPaths: sortedUnique(related), Message: message,
	})
}

func (a *auditor) markTruncated() {
	a.report.Truncated = true
	for _, issue := range a.report.Issues {
		if issue.Code == "audit.scan.truncated" {
			return
		}
	}
	issue := Issue{
		Code:         "audit.scan.truncated",
		Severity:     "error",
		RelatedPaths: []string{},
		Message:      "audit output was truncated; the incomplete scan is not eligible for promotion",
	}
	if len(a.report.Issues) < MaxIssues {
		a.report.Issues = append(a.report.Issues, issue)
		return
	}
	a.report.Issues[MaxIssues-1] = issue
}

func (a *auditor) finish() {
	finalizeReport(&a.report)
}

func finalizeReport(report *Report) {
	sort.Slice(report.Issues, func(i, j int) bool {
		left, right := report.Issues[i], report.Issues[j]
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		return left.Message < right.Message
	})
	sort.Slice(report.Artifacts, func(i, j int) bool { return report.Artifacts[i].Path < report.Artifacts[j].Path })
	report.Summary.Artifact = len(report.Artifacts)
	report.Summary.Error = 0
	report.Summary.Warning = 0
	for _, issue := range report.Issues {
		if issue.Severity == "error" {
			report.Summary.Error++
		} else {
			report.Summary.Warning++
		}
	}
	switch {
	case report.Truncated, report.Summary.Error > 0:
		report.Status = StatusFail
	case report.Summary.Warning > 0:
		report.Status = StatusWarn
	default:
		report.Status = StatusPass
	}
}

func findCanonicalDocument(documents []contracts.FinalRunDocument, canonicalPath string) (contracts.FinalRunDocument, bool) {
	for _, document := range documents {
		if document.CanonicalPath == canonicalPath {
			return document, true
		}
	}
	return contracts.FinalRunDocument{}, false
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func containsAny(text string, markers ...string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func sortedUnique(values []string) []string {
	result := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = filepath.ToSlash(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func Marshal(report Report) ([]byte, error) {
	// Kept here so callers use the deterministic report value and do not add
	// timestamps or raw provider diagnostics.
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}
