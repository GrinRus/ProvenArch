package artifactquality

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

// ValidateValidatorVerdict checks the provider draft/effective verdict invariants
// that are not expressible in validator-verdict.schema.json. Provider drafts may
// remain FAIL while advisory reconciliation is pending; effective verdicts must
// carry a technical error for FAIL. fixed_paths are orchestrator-owned after a
// deterministic repair and are therefore rejected in draft mode.
func ValidateValidatorVerdict(
	verdict contracts.ValidatorVerdict,
	finalIndex *contracts.FinalRunIndex,
	citationIndex *contracts.CitationIndex,
	requireTechnicalFailure bool,
	allowFixedPaths bool,
) error {
	problems := make([]string, 0)
	if !allowFixedPaths && len(verdict.FixedPaths) > 0 {
		problems = append(problems, "fixed_paths must be empty in provider-authored verdicts")
	}
	if strings.EqualFold(strings.TrimSpace(verdict.Verdict), "PASS") {
		for _, issue := range verdict.Issues {
			if strings.EqualFold(strings.TrimSpace(issue.Severity), "error") {
				problems = append(problems, "PASS verdict cannot contain error issues")
				break
			}
		}
	}
	if requireTechnicalFailure && strings.EqualFold(strings.TrimSpace(verdict.Verdict), "FAIL") && !hasTechnicalError(verdict.Issues) {
		problems = append(problems, "FAIL verdict must contain at least one error issue")
	}

	seen := map[string]string{}
	previous := ""
	for idx, issue := range verdict.Issues {
		identity := issueIdentity(issue)
		full := identity + "\x00" + strings.TrimSpace(issue.Severity) + "\x00" + strings.TrimSpace(issue.Message)
		if prior, exists := seen[identity]; exists {
			if prior == full {
				problems = append(problems, fmt.Sprintf("issues[%d] duplicates issue identity %q", idx, identity))
			} else {
				problems = append(problems, fmt.Sprintf("issues[%d] conflicts on issue identity %q", idx, identity))
			}
		} else {
			seen[identity] = full
		}
		canonical := issueSortKey(issue)
		if previous != "" && canonical < previous {
			problems = append(problems, "issues must be in deterministic code/path/document/citation order")
		}
		previous = canonical
	}

	if finalIndex != nil || citationIndex != nil {
		documentIDs, citationIDs, paths := verdictInventory(finalIndex, citationIndex)
		for idx, issue := range verdict.Issues {
			if documentID := strings.TrimSpace(issue.DocumentID); documentID != "" {
				if _, ok := documentIDs[documentID]; !ok {
					problems = append(problems, fmt.Sprintf("issues[%d].document_id %q is not in the selected-run inventory", idx, documentID))
				}
			}
			if citationID := strings.TrimSpace(issue.CitationID); citationID != "" {
				if _, ok := citationIDs[citationID]; !ok {
					problems = append(problems, fmt.Sprintf("issues[%d].citation_id %q is not in the selected-run inventory", idx, citationID))
				}
			}
			if rawPath := strings.TrimSpace(issue.Path); rawPath != "" {
				issuePath := normalizeVerdictPath(rawPath)
				if issuePath == "" {
					problems = append(problems, fmt.Sprintf("issues[%d].path %q is not a valid relative inventory path", idx, issue.Path))
					continue
				}
				if _, ok := paths[issuePath]; !ok {
					problems = append(problems, fmt.Sprintf("issues[%d].path %q is not in the selected-run inventory", idx, issue.Path))
				}
			}
		}
	}

	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("validator verdict consistency failed: %s", strings.Join(problems, "; "))
}

// NormalizeProviderValidatorPaths converts provider-emitted absolute paths to
// workspace-relative logical paths before consistency validation. Absolute
// paths are accepted only after symlink-resolved containment proves that they
// point inside the selected workspace; foreign or unresolved paths fail closed.
// Relative paths remain untouched so the normal verdict validator still owns
// their traversal and inventory checks.
func NormalizeProviderValidatorPaths(verdict *contracts.ValidatorVerdict, workspaceRoot string) error {
	if verdict == nil || strings.TrimSpace(workspaceRoot) == "" {
		return nil
	}

	root, err := filepath.Abs(filepath.Clean(workspaceRoot))
	if err != nil {
		return fmt.Errorf("resolve validator workspace root: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve validator workspace root symlinks: %w", err)
	}

	normalize := func(label, value string) (string, error) {
		raw := strings.TrimSpace(value)
		if !filepath.IsAbs(filepath.FromSlash(raw)) {
			return value, nil
		}
		resolvedPath, err := filepath.EvalSymlinks(filepath.Clean(filepath.FromSlash(raw)))
		if err != nil {
			return "", fmt.Errorf("%s %q cannot be resolved inside selected workspace: %w", label, value, err)
		}
		relative, err := filepath.Rel(resolvedRoot, resolvedPath)
		if err != nil {
			return "", fmt.Errorf("%s %q cannot be made relative to selected workspace: %w", label, value, err)
		}
		normalized := normalizeVerdictPath(filepath.ToSlash(relative))
		if normalized == "" {
			return "", fmt.Errorf("%s %q is outside the selected workspace", label, value)
		}
		return normalized, nil
	}

	for idx := range verdict.CheckedPaths {
		normalized, err := normalize("checked_paths", verdict.CheckedPaths[idx])
		if err != nil {
			return err
		}
		verdict.CheckedPaths[idx] = normalized
	}
	for idx := range verdict.FixedPaths {
		normalized, err := normalize("fixed_paths", verdict.FixedPaths[idx])
		if err != nil {
			return err
		}
		verdict.FixedPaths[idx] = normalized
	}
	for idx := range verdict.Issues {
		normalized, err := normalize(fmt.Sprintf("issues[%d].path", idx), verdict.Issues[idx].Path)
		if err != nil {
			return err
		}
		verdict.Issues[idx].Path = normalized
	}
	return nil
}

func hasTechnicalError(issues []contracts.ValidatorIssue) bool {
	for _, issue := range issues {
		if strings.EqualFold(strings.TrimSpace(issue.Severity), "error") {
			return true
		}
	}
	return false
}

func issueIdentity(issue contracts.ValidatorIssue) string {
	return strings.Join([]string{
		strings.TrimSpace(issue.Code),
		normalizeVerdictPath(issue.Path),
		strings.TrimSpace(issue.DocumentID),
		strings.TrimSpace(issue.CitationID),
	}, "\x00")
}

func issueSortKey(issue contracts.ValidatorIssue) string {
	return strings.Join([]string{
		strings.TrimSpace(issue.Code),
		normalizeVerdictPath(issue.Path),
		strings.TrimSpace(issue.DocumentID),
		strings.TrimSpace(issue.CitationID),
		strings.TrimSpace(issue.Severity),
		strings.TrimSpace(issue.Message),
	}, "\x00")
}

func verdictInventory(finalIndex *contracts.FinalRunIndex, citationIndex *contracts.CitationIndex) (map[string]struct{}, map[string]struct{}, map[string]struct{}) {
	documents := map[string]struct{}{}
	citations := map[string]struct{}{}
	paths := map[string]struct{}{}
	if finalIndex != nil {
		if runID := strings.TrimSpace(finalIndex.RunID); runID != "" {
			finalRoot := path.Join("reports", "taskruns", runID, "staging", "final")
			paths[path.Join(finalRoot, "final-run-index.json")] = struct{}{}
			paths[path.Join(finalRoot, "citation-index.json")] = struct{}{}
		}
		if normalized := normalizeVerdictPath(finalIndex.CitationIndexPath); normalized != "" {
			paths[normalized] = struct{}{}
		}
		for _, document := range finalIndex.CanonicalDocuments {
			if id := strings.TrimSpace(document.ID); id != "" {
				documents[id] = struct{}{}
			}
			for _, value := range []string{document.CanonicalPath, document.StagedPath} {
				if normalized := normalizeVerdictPath(value); normalized != "" {
					paths[normalized] = struct{}{}
				}
			}
		}
	}
	if citationIndex != nil {
		if runID := strings.TrimSpace(citationIndex.RunID); runID != "" {
			paths[path.Join("reports", "taskruns", runID, "staging", "final", "citation-index.json")] = struct{}{}
		}
		for _, citation := range citationIndex.Citations {
			if id := strings.TrimSpace(citation.ID); id != "" {
				citations[id] = struct{}{}
			}
			if normalized := normalizeVerdictPath(citation.Path); normalized != "" {
				paths[normalized] = struct{}{}
			}
		}
	}
	return documents, citations, paths
}

func normalizeVerdictPath(value string) string {
	clean := path.Clean(strings.TrimSpace(strings.ReplaceAll(value, "\\", "/")))
	if clean == "." || clean == "" || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return ""
	}
	return clean
}
