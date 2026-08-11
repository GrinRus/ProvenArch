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
	Version   int        `json:"version"`
	RunID     string     `json:"run_id"`
	Scope     string     `json:"scope"`
	Status    Status     `json:"status"`
	Summary   Summary    `json:"summary"`
	Issues    []Issue    `json:"issues"`
	Artifacts []Artifact `json:"artifacts"`
	Truncated bool       `json:"truncated"`
}

func ScanSelectedRun(ws workspace.Root, runID string) Report {
	return scan(ws, runID, "selected_run")
}

func ScanPromotedRun(ws workspace.Root, runID string) Report {
	return scan(ws, runID, "promoted_current")
}

func scan(ws workspace.Root, runID string, scope string) Report {
	runID = strings.TrimSpace(runID)
	audit := auditor{
		ws:          ws,
		runID:       runID,
		report:      Report{Version: Version, RunID: runID, Scope: scope, Status: StatusPass, Issues: []Issue{}, Artifacts: []Artifact{}},
		documents:   map[string]contracts.FinalRunDocument{},
		citations:   map[string]contracts.DocumentCitation{},
		repoSources: map[string]workspace.RepoSource{},
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
	verdictRaw, err := a.ws.ReadFileLimit(verdictPath, MaxArtifactBytes)
	if err != nil {
		a.add("audit.validator.unavailable", "error", verdictPath, nil, "validator verdict is unavailable or oversized")
		return
	}
	verdict, err := contracts.ParseValidatorVerdict(verdictRaw)
	if err != nil || verdict.RunID != a.runID || strings.ToUpper(verdict.Verdict) != "PASS" {
		a.add("audit.validator.not_promoted", "error", verdictPath, nil, "validator verdict is invalid, foreign or not PASS")
		return
	}
	a.record(verdictPath, verdictRaw)
	a.scanValidatorPaths(verdict, finalRoot, finalPath, citationPath)

	for _, document := range index.CanonicalDocuments {
		a.documents[document.ID] = document
		a.scanDocument(finalRoot, document)
	}
	a.scanReciprocalReferences()
	for _, citation := range citationIndex.Citations {
		a.scanEvidence(citation)
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
	source, ok := a.repoSources[citation.Repo]
	if !ok {
		a.add("audit.evidence.repo_unknown", "error", citation.Path, nil, "citation references an unknown repository")
		return
	}
	repoRoot := strings.TrimSpace(source.Path)
	if repoRoot == "" {
		a.add("audit.evidence.repo_unavailable", "warning", citation.Path, nil, "citation repository is not available as a local path")
		return
	}
	if !filepath.IsAbs(repoRoot) {
		repoRoot = filepath.Join(a.ws.Path, repoRoot)
	}
	rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(citation.Path)))
	if rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		a.add("audit.evidence.path_escape", "error", citation.Path, nil, "citation evidence path escapes its repository")
		return
	}
	resolvedRoot, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		a.add("audit.evidence.repo_unavailable", "warning", citation.Path, nil, "citation repository cannot be resolved")
		return
	}
	resolvedEvidence, err := filepath.EvalSymlinks(filepath.Join(resolvedRoot, rel))
	if err != nil {
		a.add("audit.evidence.file_missing", "error", citation.Path, nil, "citation evidence file does not exist")
		return
	}
	contained, err := filepath.Rel(resolvedRoot, resolvedEvidence)
	if err != nil || contained == ".." || strings.HasPrefix(contained, ".."+string(filepath.Separator)) || filepath.IsAbs(contained) {
		a.add("audit.evidence.path_escape", "error", citation.Path, nil, "citation evidence resolves outside its repository")
		return
	}
	info, err := os.Stat(resolvedEvidence)
	if err != nil || !info.Mode().IsRegular() {
		a.add("audit.evidence.not_regular", "error", citation.Path, nil, "citation evidence is not a regular file")
	}
}

func (a *auditor) scanValidatorPaths(verdict contracts.ValidatorVerdict, finalRoot, finalPath, citationPath string) {
	seen := map[string]struct{}{}
	hasFinal, hasCitation := false, false
	taskRoot := path.Join("reports", "taskruns", a.runID) + "/"
	for _, raw := range verdict.CheckedPaths {
		clean := filepath.ToSlash(path.Clean(strings.TrimSpace(raw)))
		if clean == "." || filepath.IsAbs(clean) || strings.Contains(clean, "\\") || !strings.HasPrefix(clean, taskRoot) {
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
		clean := filepath.ToSlash(path.Clean(strings.TrimSpace(raw)))
		if clean == "." || filepath.IsAbs(clean) || strings.Contains(clean, "\\") || !strings.HasPrefix(clean, taskRoot) {
			a.add("audit.validator.fixed_path_foreign", "error", clean, []string{finalPath}, "provider fixed_paths must remain inside the selected run")
		}
	}
}

func (a *auditor) record(filePath string, raw []byte) {
	if len(a.report.Artifacts) >= MaxArtifacts {
		a.report.Truncated = true
		return
	}
	sum := sha256.Sum256(raw)
	a.report.Artifacts = append(a.report.Artifacts, Artifact{
		Path: filePath, Size: len(raw), SHA256: hex.EncodeToString(sum[:]),
	})
}

func (a *auditor) add(code, severity, filePath string, related []string, message string) {
	if len(a.report.Issues) >= MaxIssues {
		a.report.Truncated = true
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

func (a *auditor) finish() {
	sort.Slice(a.report.Issues, func(i, j int) bool {
		left, right := a.report.Issues[i], a.report.Issues[j]
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		return left.Message < right.Message
	})
	sort.Slice(a.report.Artifacts, func(i, j int) bool { return a.report.Artifacts[i].Path < a.report.Artifacts[j].Path })
	a.report.Summary.Artifact = len(a.report.Artifacts)
	for _, issue := range a.report.Issues {
		if issue.Severity == "error" {
			a.report.Summary.Error++
		} else {
			a.report.Summary.Warning++
		}
	}
	switch {
	case a.report.Summary.Error > 0:
		a.report.Status = StatusFail
	case a.report.Summary.Warning > 0:
		a.report.Status = StatusWarn
	default:
		a.report.Status = StatusPass
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
