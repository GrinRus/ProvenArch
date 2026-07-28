package workspacehealth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
	"gopkg.in/yaml.v3"
)

const Version = 1

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Summary struct {
	Info    int `json:"info"`
	Warning int `json:"warning"`
	Error   int `json:"error"`
}

type Item struct {
	ID           string   `json:"id"`
	Severity     Severity `json:"severity"`
	Title        string   `json:"title"`
	Path         string   `json:"path,omitempty"`
	RelatedPaths []string `json:"related_paths"`
}

type Report struct {
	Version     int       `json:"version"`
	GeneratedAt time.Time `json:"generated_at"`
	Status      Status    `json:"status"`
	Summary     Summary   `json:"summary"`
	Items       []Item    `json:"items"`
}

type Options struct {
	Now func() time.Time
}

func Scan(ctx context.Context, ws workspace.Root, options Options) (Report, error) {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	report := Report{
		Version:     Version,
		GeneratedAt: now().UTC(),
		Status:      StatusPass,
		Items:       []Item{},
	}

	scanner := scanner{
		ctx:       ctx,
		ws:        ws,
		report:    &report,
		entityIDs: map[string]string{},
		aliases:   map[string][]string{},
	}
	if err := scanner.scanModelObservations(); err != nil {
		return Report{}, err
	}
	if err := scanner.scanDomainOutputs(); err != nil {
		return Report{}, err
	}
	if err := scanner.scanProposals(); err != nil {
		return Report{}, err
	}
	if err := scanner.scanOpenQuestions(); err != nil {
		return Report{}, err
	}
	if err := scanner.scanMarkdownIntegrity(); err != nil {
		return Report{}, err
	}
	if err := scanner.scanFindingLinks(); err != nil {
		return Report{}, err
	}
	if err := scanner.scanCitationCoverage(); err != nil {
		return Report{}, err
	}

	sort.SliceStable(report.Items, func(i, j int) bool {
		left := report.Items[i]
		right := report.Items[j]
		if left.Severity != right.Severity {
			return severityRank(left.Severity) > severityRank(right.Severity)
		}
		if left.ID != right.ID {
			return left.ID < right.ID
		}
		return left.Path < right.Path
	})
	for _, item := range report.Items {
		switch item.Severity {
		case SeverityError:
			report.Summary.Error++
		case SeverityWarning:
			report.Summary.Warning++
		default:
			report.Summary.Info++
		}
	}
	switch {
	case report.Summary.Error > 0:
		report.Status = StatusFail
	case report.Summary.Warning > 0:
		report.Status = StatusWarn
	default:
		report.Status = StatusPass
	}
	return report, nil
}

type scanner struct {
	ctx       context.Context
	ws        workspace.Root
	report    *Report
	entityIDs map[string]string
	aliases   map[string][]string
}

func (s scanner) scanModelObservations() error {
	if err := s.scanModelEntityObservations(); err != nil {
		return err
	}
	return s.scanModelEdgeObservations()
}

func (s scanner) scanModelEntityObservations() error {
	return s.walkWorkspaceFiles("model/entities", ".yaml", func(rel string, raw []byte) {
		var entity contracts.Entity
		if err := yaml.Unmarshal(raw, &entity); err != nil {
			s.add(SeverityError, "model.entity.invalid_yaml", fmt.Sprintf("Cannot parse model entity YAML: %v", err), rel, nil)
			return
		}
		entityID := strings.TrimSpace(entity.ID)
		if entityID != "" {
			s.entityIDs[entityID] = rel
		}
		for _, alias := range entity.Aliases {
			key := strings.ToLower(strings.TrimSpace(alias))
			if key != "" {
				s.aliases[key] = append(s.aliases[key], rel)
			}
		}
		if teamID := strings.TrimSpace(entity.OwnerTeamID); teamID != "" {
			teamCard := filepath.ToSlash(filepath.Join("charter/cards/teams", strings.TrimPrefix(teamID, "team.")+".md"))
			if _, err := s.ws.ReadFile(teamCard); isNotExistError(err) {
				s.add(SeverityWarning, "model.owner_team.missing", fmt.Sprintf("Entity %q references missing owner team %q", entityID, teamID), rel, []string{teamCard})
			}
		}
		if isObservationWithoutEvidence(entity.Provenance) {
			title := "Observation entity has no evidence"
			if strings.TrimSpace(entity.ID) != "" {
				title = fmt.Sprintf("Observation entity %q has no evidence", entity.ID)
			}
			s.add(SeverityWarning, "model.observation.missing_evidence", title, rel, nil)
		}
	})
}

func (s scanner) scanModelEdgeObservations() error {
	err := s.walkWorkspaceFiles("model/edges", ".yaml", func(rel string, raw []byte) {
		var edge contracts.Edge
		if err := yaml.Unmarshal(raw, &edge); err != nil {
			s.add(SeverityError, "model.edge.invalid_yaml", fmt.Sprintf("Cannot parse model edge YAML: %v", err), rel, nil)
			return
		}
		for _, endpoint := range []struct {
			label string
			value string
		}{{"from", edge.From}, {"to", edge.To}} {
			if _, ok := s.entityIDs[strings.TrimSpace(endpoint.value)]; !ok {
				s.add(SeverityError, "model.edge.endpoint_missing", fmt.Sprintf("Edge %q %s endpoint %q does not resolve to a canonical entity", edge.ID, endpoint.label, endpoint.value), rel, nil)
			}
		}
		if isObservationWithoutEvidence(edge.Provenance) {
			title := "Observation edge has no evidence"
			if strings.TrimSpace(edge.ID) != "" {
				title = fmt.Sprintf("Observation edge %q has no evidence", edge.ID)
			}
			s.add(SeverityWarning, "model.observation.missing_evidence", title, rel, nil)
		}
	})
	if err != nil {
		return err
	}
	for alias, paths := range s.aliases {
		if len(paths) > 1 {
			s.add(SeverityWarning, "model.entity.alias_duplicate", fmt.Sprintf("Entity alias %q is used by %d entities", alias, len(paths)), paths[0], paths[1:])
		}
	}
	return nil
}

func (s scanner) scanDomainOutputs() error {
	if err := s.scanOwnedOutputs("domains", "domain"); err != nil {
		return err
	}
	return s.scanOwnedOutputs("teams", "team")
}

func (s scanner) scanOwnedOutputs(folder string, kind string) error {
	return s.walkWorkspaceFiles(filepath.ToSlash(filepath.Join("reports/agent-outputs", folder)), ".md", func(rel string, _ []byte) {
		domainID := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
		cardRel := filepath.ToSlash(filepath.Join("charter/cards", folder, domainID+".md"))
		cardAbs, err := s.ws.Resolve(cardRel)
		if err != nil {
			s.add(SeverityError, "workspace.path.invalid", fmt.Sprintf("Cannot resolve domain card path for %q: %v", domainID, err), rel, nil)
			return
		}
		if _, statErr := os.Stat(cardAbs); errors.Is(statErr, os.ErrNotExist) {
			label := strings.ToUpper(kind[:1]) + kind[1:]
			s.add(SeverityWarning, kind+".output.orphan", fmt.Sprintf("%s output %q has no matching canonical %s card", label, domainID, kind), rel, []string{cardRel})
			return
		} else if statErr != nil {
			s.add(SeverityError, "domain.card.unreadable", fmt.Sprintf("Cannot read matching domain card %q: %v", cardRel, statErr), rel, []string{cardRel})
		}
	})
}

var (
	markdownLinkPattern    = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	findingIDPattern       = regexp.MustCompile("`(finding\\.[A-Za-z0-9._-]+)`")
	citationIDPattern      = regexp.MustCompile(`(?i)\b(?:cite|citation)\.[A-Za-z0-9._-]+\b`)
	workspacePathCodeRegex = regexp.MustCompile("`((?:charter|model|reports|proposals|docs)/[^`#?]+)`")
	externalLinkPattern    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
)

func (s scanner) scanCitationCoverage() error {
	for _, root := range []string{"reports/as-is", "reports/findings", "proposals"} {
		if err := s.walkWorkspaceFiles(root, ".md", func(rel string, raw []byte) {
			if strings.HasPrefix(rel, "proposals/") && filepath.Base(rel) != "proposal.md" {
				return
			}
			content := strings.TrimSpace(string(raw))
			if content == "" || citationIDPattern.MatchString(content) {
				return
			}
			s.add(
				SeverityWarning,
				"citation.coverage.low",
				"Key architecture document has no explicit citation identifiers",
				rel,
				nil,
			)
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s scanner) scanMarkdownIntegrity() error {
	for _, root := range []string{"charter", "reports", "proposals"} {
		if err := s.walkWorkspaceFiles(root, ".md", func(rel string, raw []byte) {
			if rel == "reports/taskruns" || strings.HasPrefix(rel, "reports/taskruns/") {
				return
			}
			if !utf8.Valid(raw) || strings.IndexByte(string(raw), 0) >= 0 {
				s.add(SeverityError, "workspace.canonical.invalid_text", "Canonical Markdown is not valid UTF-8 text", rel, nil)
				return
			}
			for _, match := range markdownLinkPattern.FindAllStringSubmatch(string(raw), -1) {
				target := strings.TrimSpace(match[1])
				if target == "" || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "//") || externalLinkPattern.MatchString(target) {
					continue
				}
				target = strings.Split(strings.Split(target, "#")[0], "?")[0]
				resolved, ok := resolveHealthLink(rel, target)
				if !ok {
					s.add(SeverityWarning, "artifact.link.invalid", "Local Markdown link escapes the workspace or is invalid", rel, []string{target})
					continue
				}
				if _, err := s.ws.ReadFile(resolved); err != nil {
					s.add(SeverityWarning, "artifact.link.broken", fmt.Sprintf("Local Markdown link target %q does not exist", resolved), rel, []string{resolved})
				}
			}
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s scanner) scanFindingLinks() error {
	findings := map[string]string{}
	if err := s.walkWorkspaceFiles("reports/findings", ".md", func(rel string, raw []byte) {
		for _, match := range findingIDPattern.FindAllStringSubmatch(string(raw), -1) {
			findings[match[1]] = rel
		}
	}); err != nil {
		return err
	}
	linked := map[string]bool{}
	if err := s.walkWorkspaceFiles("proposals", ".md", func(rel string, raw []byte) {
		content := string(raw)
		for _, match := range findingIDPattern.FindAllStringSubmatch(content, -1) {
			linked[match[1]] = true
		}
		for _, match := range workspacePathCodeRegex.FindAllStringSubmatch(content, -1) {
			target := strings.TrimSpace(match[1])
			if _, err := s.ws.ReadFile(target); err != nil {
				s.add(SeverityWarning, "proposal.evidence.missing", fmt.Sprintf("Proposal evidence path %q does not exist", target), rel, []string{target})
			}
		}
	}); err != nil {
		return err
	}
	for id, rel := range findings {
		if !linked[id] {
			s.add(SeverityWarning, "finding.unlinked", fmt.Sprintf("Finding %q is not linked from any proposal", id), rel, nil)
		}
	}
	return nil
}

func resolveHealthLink(baseRel string, target string) (string, bool) {
	normalized := strings.ReplaceAll(strings.TrimSpace(target), "\\", "/")
	if normalized == "" || strings.HasPrefix(normalized, "/") {
		return "", false
	}
	resolved := path.Clean(path.Join(path.Dir(baseRel), normalized))
	if resolved == "." || resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", false
	}
	return resolved, true
}

func (s scanner) scanProposals() error {
	return s.walkWorkspaceFiles("proposals", "proposal.md", func(rel string, raw []byte) {
		missing := missingProposalSections(string(raw))
		if len(missing) == 0 {
			return
		}
		s.add(SeverityWarning, "proposal.missing_review_sections", fmt.Sprintf("Proposal is missing review section(s): %s", strings.Join(missing, ", ")), rel, nil)
	})
}

func (s scanner) scanOpenQuestions() error {
	rel := "reports/coverage/open-questions.md"
	raw, err := s.ws.ReadFile(rel)
	if err != nil {
		if isNotExistError(err) {
			return nil
		}
		return fmt.Errorf("read open questions: %w", err)
	}
	count := countMarkdownListItems(string(raw))
	if count > 0 {
		s.add(SeverityInfo, "coverage.open_questions.count", fmt.Sprintf("%d unresolved coverage question(s) are open", count), rel, nil)
	}
	return nil
}

func (s scanner) walkWorkspaceFiles(relDir string, suffix string, visitor func(rel string, raw []byte)) error {
	absDir, err := s.ws.Resolve(relDir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", relDir, err)
	}
	entries := []string{}
	if err := filepath.WalkDir(absDir, func(abs string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if s.ctx != nil {
			if err := s.ctx.Err(); err != nil {
				return err
			}
		}
		if entry.IsDir() {
			return nil
		}
		name := strings.ToLower(entry.Name())
		if suffix == "proposal.md" {
			if name != "proposal.md" {
				return nil
			}
		} else if !strings.HasSuffix(name, suffix) {
			return nil
		}
		rel, err := filepath.Rel(s.ws.Path, abs)
		if err != nil {
			return err
		}
		entries = append(entries, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("walk %s: %w", relDir, err)
	}
	sort.Strings(entries)
	for _, rel := range entries {
		raw, err := s.ws.ReadFile(rel)
		if err != nil {
			s.add(SeverityError, "workspace.file.unreadable", fmt.Sprintf("Cannot read workspace artifact: %v", err), rel, nil)
			continue
		}
		visitor(rel, raw)
	}
	return nil
}

func (s scanner) add(severity Severity, id string, title string, rel string, related []string) {
	item := Item{
		ID:           id,
		Severity:     severity,
		Title:        title,
		Path:         filepath.ToSlash(strings.TrimSpace(rel)),
		RelatedPaths: normalizeRelatedPaths(related),
	}
	s.report.Items = append(s.report.Items, item)
}

func isObservationWithoutEvidence(provenance contracts.Provenance) bool {
	return strings.EqualFold(strings.TrimSpace(provenance.Kind), "observation") && len(provenance.Evidence) == 0
}

var proposalHeadingPattern = regexp.MustCompile(`(?im)^#{1,6}\s+(evidence|citations?|unresolved|open questions?)\b`)

func missingProposalSections(content string) []string {
	found := map[string]bool{}
	for _, match := range proposalHeadingPattern.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		heading := strings.ToLower(strings.TrimSpace(match[1]))
		switch {
		case heading == "evidence":
			found["evidence"] = true
		case strings.HasPrefix(heading, "citation"):
			found["citations"] = true
		case heading == "unresolved" || heading == "open questions":
			found["unresolved"] = true
		}
	}
	missing := []string{}
	for _, required := range []string{"evidence", "citations", "unresolved"} {
		if !found[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func countMarkdownListItems(content string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			count++
		}
	}
	return count
}

func normalizeRelatedPaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, item := range paths {
		clean := filepath.ToSlash(strings.TrimSpace(item))
		if clean != "" {
			normalized = append(normalized, clean)
		}
	}
	sort.Strings(normalized)
	return normalized
}

func isNotExistError(err error) bool {
	return errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such file or directory")
}

func severityRank(severity Severity) int {
	switch severity {
	case SeverityError:
		return 3
	case SeverityWarning:
		return 2
	default:
		return 1
	}
}
