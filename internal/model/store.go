package model

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
	"gopkg.in/yaml.v3"
)

type Store struct {
	workspace workspace.Root
}

type ApplyReport struct {
	UpsertedEntities int
	UpsertedEdges    int
	RemappedIDs      map[string]string
}

func NewStore(ws workspace.Root) Store {
	return Store{workspace: ws}
}

func (s Store) ApplySemanticSnapshot(snapshot contracts.SemanticSnapshot) (ApplyReport, error) {
	report := ApplyReport{RemappedIDs: map[string]string{}}

	existingEntities, err := s.loadEntityMap()
	if err != nil {
		return report, err
	}
	existingTeams := make(map[string]struct{})
	for id, entity := range existingEntities {
		if entity.Type == "team" {
			existingTeams[id] = struct{}{}
		}
	}
	incomingTeams := collectIncomingTeamIDs(snapshot.Entities)

	for _, entity := range snapshot.Entities {
		originalIncomingID := entity.ID
		entity.ID = normalizeCanonicalID(entity.ID)
		entity.OwnerTeamID = normalizeCanonicalID(entity.OwnerTeamID)
		if originalIncomingID != entity.ID {
			report.RemappedIDs[originalIncomingID] = entity.ID
			entity.Aliases = appendUnique(entity.Aliases, originalIncomingID)
		}
		if err := validateOwnerTeamLink(entity, existingTeams, incomingTeams); err != nil {
			return report, err
		}

		originalID := entity.ID
		if existing, exists := existingEntities[entity.ID]; exists {
			if shouldRemapCollision(existing, entity) {
				repoSlug := extractEvidenceRepoSlug(entity.Provenance.Evidence)
				entity.ID = fmt.Sprintf("%s.repo-%s", originalID, repoSlug)
				entity.Aliases = appendUnique(entity.Aliases, originalID)
				report.RemappedIDs[originalID] = entity.ID
			}
		}

		if err := s.writeYAML(filepath.Join("model/entities", safeFileName(entity.ID)+".yaml"), entity); err != nil {
			return report, fmt.Errorf("write entity %q: %w", entity.ID, err)
		}
		existingEntities[entity.ID] = entity
		if entity.Type == "team" {
			existingTeams[entity.ID] = struct{}{}
		}
		report.UpsertedEntities++
	}

	for _, edge := range snapshot.Edges {
		originalFrom := edge.From
		originalTo := edge.To
		edge.From = normalizeCanonicalID(edge.From)
		edge.To = normalizeCanonicalID(edge.To)
		if remapped, ok := report.RemappedIDs[originalFrom]; ok {
			edge.From = remapped
		}
		if remapped, ok := report.RemappedIDs[originalTo]; ok {
			edge.To = remapped
		}
		if remapped, ok := report.RemappedIDs[edge.From]; ok {
			edge.From = remapped
		}
		if remapped, ok := report.RemappedIDs[edge.To]; ok {
			edge.To = remapped
		}
		edge.ID = canonicalEdgeID(edge.From, edge.Type, edge.To)
		if err := s.writeYAML(filepath.Join("model/edges", safeFileName(edge.ID)+".yaml"), edge); err != nil {
			return report, fmt.Errorf("write edge %q: %w", edge.ID, err)
		}
		report.UpsertedEdges++
	}

	return report, nil
}

func (s Store) ListEntities() ([]contracts.Entity, error) {
	var entities []contracts.Entity
	if err := s.walkYAMLFiles("model/entities", func(content []byte) error {
		var entity contracts.Entity
		if err := yaml.Unmarshal(content, &entity); err != nil {
			return fmt.Errorf("parse entity yaml: %w", err)
		}
		entities = append(entities, entity)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].ID < entities[j].ID
	})
	return entities, nil
}

func (s Store) ListEdges() ([]contracts.Edge, error) {
	var edges []contracts.Edge
	if err := s.walkYAMLFiles("model/edges", func(content []byte) error {
		var edge contracts.Edge
		if err := yaml.Unmarshal(content, &edge); err != nil {
			return fmt.Errorf("parse edge yaml: %w", err)
		}
		edges = append(edges, edge)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].ID < edges[j].ID
	})
	return edges, nil
}

func (s Store) loadEntityMap() (map[string]contracts.Entity, error) {
	entities, err := s.ListEntities()
	if err != nil {
		return nil, err
	}
	m := make(map[string]contracts.Entity, len(entities))
	for _, entity := range entities {
		m[entity.ID] = entity
	}
	return m, nil
}

func (s Store) walkYAMLFiles(relDir string, visitor func(content []byte) error) error {
	dir, err := s.workspace.Resolve(relDir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".yaml") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %q: %w", path, err)
		}
		return visitor(content)
	})
}

func (s Store) writeYAML(relPath string, payload any) error {
	content, err := yaml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	return s.workspace.WriteFile(relPath, content)
}

func (s Store) removeFile(relPath string) error {
	abs, err := s.workspace.Resolve(relPath)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %q: %w", relPath, err)
	}
	return nil
}

func collectIncomingTeamIDs(entities []contracts.Entity) map[string]struct{} {
	ids := map[string]struct{}{}
	for _, entity := range entities {
		if entity.Type == "team" && strings.TrimSpace(entity.ID) != "" {
			ids[normalizeCanonicalID(entity.ID)] = struct{}{}
		}
	}
	return ids
}

func validateOwnerTeamLink(entity contracts.Entity, existing, incoming map[string]struct{}) error {
	ownerID := normalizeCanonicalID(entity.OwnerTeamID)
	if ownerID == "" {
		return nil
	}
	if !strings.HasPrefix(ownerID, "team.") {
		return fmt.Errorf("owner_team_id %q for entity %q must use team.<slug> format", entity.OwnerTeamID, entity.ID)
	}
	if _, ok := existing[ownerID]; ok {
		return nil
	}
	if _, ok := incoming[ownerID]; ok {
		return nil
	}
	return fmt.Errorf("owner_team_id %q for entity %q does not reference existing team.<slug>", ownerID, entity.ID)
}

func extractEvidenceRepoSlug(evidence []contracts.Evidence) string {
	repo := normalizeEvidenceRepoIdentity(primaryEvidenceRepo(evidence))
	if repo == "" {
		return "unknown"
	}
	return slugutil.Slugify(repo)
}

func primaryEvidenceRepo(evidence []contracts.Evidence) string {
	if len(evidence) == 0 {
		return ""
	}
	return normalizeEvidenceRepoIdentity(evidence[0].Repo)
}

func shouldRemapCollision(existing contracts.Entity, incoming contracts.Entity) bool {
	existingRepo := primaryEvidenceRepo(existing.Provenance.Evidence)
	incomingRepo := primaryEvidenceRepo(incoming.Provenance.Evidence)
	if existingRepo == "" || incomingRepo == "" {
		return false
	}
	return existingRepo != incomingRepo
}

func normalizeEvidenceRepoIdentity(value string) string {
	value = strings.TrimSpace(filepath.Base(strings.TrimSpace(value)))
	if value == "" || value == "." {
		return ""
	}
	lastDash := strings.LastIndex(value, "-")
	if lastDash > 0 && lastDash < len(value)-1 {
		suffix := value[lastDash+1:]
		if len(suffix) >= 7 && isHexToken(suffix) {
			value = value[:lastDash]
		}
	}
	return strings.TrimSpace(value)
}

func isHexToken(value string) bool {
	for _, r := range value {
		isDigit := r >= '0' && r <= '9'
		isLowerHex := r >= 'a' && r <= 'f'
		isUpperHex := r >= 'A' && r <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return false
		}
	}
	return true
}

func appendUnique(values []string, candidate string) []string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return values
	}
	for _, value := range values {
		if value == candidate {
			return values
		}
	}
	return append(values, candidate)
}

func safeFileName(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "unknown"
	}
	var out []rune
	for _, r := range id {
		isAlpha := r >= 'a' && r <= 'z'
		isUpper := r >= 'A' && r <= 'Z'
		isDigit := r >= '0' && r <= '9'
		if isAlpha || isUpper || isDigit || r == '.' || r == '-' || r == '_' {
			out = append(out, r)
			continue
		}
		out = append(out, '-')
	}
	return strings.Trim(string(out), "-")
}

func normalizeCanonicalID(id string) string {
	id = strings.TrimSpace(strings.ToLower(id))
	if id == "" {
		return ""
	}
	var out []rune
	prevDash := false
	for _, r := range id {
		isAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isAlpha || isDigit || r == '.' || r == '_' {
			out = append(out, r)
			prevDash = false
			continue
		}
		if r == '-' {
			if prevDash {
				continue
			}
			out = append(out, r)
			prevDash = true
			continue
		}
		if prevDash {
			continue
		}
		out = append(out, '-')
		prevDash = true
	}
	normalized := strings.Trim(string(out), "-")
	normalized = strings.Trim(normalized, ".")
	return normalized
}

func canonicalEdgeID(from string, edgeType string, to string) string {
	from = normalizeCanonicalID(from)
	edgeType = normalizeCanonicalID(edgeType)
	to = normalizeCanonicalID(to)
	return fmt.Sprintf("edge.%s.%s.%s", from, edgeType, to)
}
