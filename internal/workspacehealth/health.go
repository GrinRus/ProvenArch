package workspacehealth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

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
		ctx:    ctx,
		ws:     ws,
		report: &report,
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
	ctx    context.Context
	ws     workspace.Root
	report *Report
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
	return s.walkWorkspaceFiles("model/edges", ".yaml", func(rel string, raw []byte) {
		var edge contracts.Edge
		if err := yaml.Unmarshal(raw, &edge); err != nil {
			s.add(SeverityError, "model.edge.invalid_yaml", fmt.Sprintf("Cannot parse model edge YAML: %v", err), rel, nil)
			return
		}
		if isObservationWithoutEvidence(edge.Provenance) {
			title := "Observation edge has no evidence"
			if strings.TrimSpace(edge.ID) != "" {
				title = fmt.Sprintf("Observation edge %q has no evidence", edge.ID)
			}
			s.add(SeverityWarning, "model.observation.missing_evidence", title, rel, nil)
		}
	})
}

func (s scanner) scanDomainOutputs() error {
	return s.walkWorkspaceFiles("reports/agent-outputs/domains", ".md", func(rel string, _ []byte) {
		domainID := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
		cardRel := filepath.ToSlash(filepath.Join("charter/cards/domains", domainID+".md"))
		cardAbs, err := s.ws.Resolve(cardRel)
		if err != nil {
			s.add(SeverityError, "workspace.path.invalid", fmt.Sprintf("Cannot resolve domain card path for %q: %v", domainID, err), rel, nil)
			return
		}
		if _, statErr := os.Stat(cardAbs); errors.Is(statErr, os.ErrNotExist) {
			s.add(SeverityWarning, "domain.output.orphan", fmt.Sprintf("Domain output %q has no matching canonical domain card", domainID), rel, []string{cardRel})
			return
		} else if statErr != nil {
			s.add(SeverityError, "domain.card.unreadable", fmt.Sprintf("Cannot read matching domain card %q: %v", cardRel, statErr), rel, []string{cardRel})
		}
	})
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
