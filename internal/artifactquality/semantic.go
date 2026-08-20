package artifactquality

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

// ValidateSemanticEnvelope rejects semantic graph drift after the typed
// contract has been decoded. JSON Schema still owns shape validation; this
// layer owns identity and reference integrity.
func ValidateSemanticEnvelope(snapshot contracts.SemanticSnapshot) error {
	problems := make([]string, 0)
	problems = append(problems, validateSemanticIDCollisions(snapshot)...)
	entityIDs := map[string]struct{}{}
	for _, entity := range snapshot.Entities {
		id := strings.TrimSpace(entity.ID)
		if id != "" {
			entityIDs[id] = struct{}{}
		}
	}
	for _, edge := range snapshot.Edges {
		if from := strings.TrimSpace(edge.From); from == "" {
			problems = append(problems, "edge "+strings.TrimSpace(edge.ID)+" has an empty from endpoint")
		} else if _, ok := entityIDs[from]; !ok {
			problems = append(problems, fmt.Sprintf("edge %q references dangling from endpoint %q", edge.ID, edge.From))
		}
		if to := strings.TrimSpace(edge.To); to == "" {
			problems = append(problems, "edge "+strings.TrimSpace(edge.ID)+" has an empty to endpoint")
		} else if _, ok := entityIDs[to]; !ok {
			problems = append(problems, fmt.Sprintf("edge %q references dangling to endpoint %q", edge.ID, edge.To))
		}
	}
	for _, entity := range snapshot.Entities {
		if owner := strings.TrimSpace(entity.OwnerTeamID); owner != "" {
			if !strings.HasPrefix(owner, "team.") {
				problems = append(problems, fmt.Sprintf("entity %q owner team %q must use team.<slug> format", entity.ID, entity.OwnerTeamID))
				continue
			}
			ownerEntity, ok := findSemanticEntity(snapshot.Entities, owner)
			if !ok {
				problems = append(problems, fmt.Sprintf("entity %q references missing owner team %q", entity.ID, owner))
			} else if strings.TrimSpace(ownerEntity.Type) != "team" {
				problems = append(problems, fmt.Sprintf("entity %q owner team %q is not a team entity", entity.ID, owner))
			}
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return semanticProblems(problems)
}

func findSemanticEntity(entities []contracts.Entity, id string) (contracts.Entity, bool) {
	for _, entity := range entities {
		if strings.TrimSpace(entity.ID) == id {
			return entity, true
		}
	}
	return contracts.Entity{}, false
}

// ValidateSemanticIDCollisions checks IDs across a set of shards before
// aggregation can silently overwrite a sibling shard's object.
func ValidateSemanticIDCollisions(snapshots ...contracts.SemanticSnapshot) error {
	type registration struct {
		identity    string
		location    string
		fingerprint string
		value       any
	}
	seen := map[string]registration{}
	problems := []string{}
	for snapshotIdx, snapshot := range snapshots {
		register := func(kind, id string, value any) {
			id = strings.TrimSpace(id)
			if id == "" {
				problems = append(problems, fmt.Sprintf("%s[%d] id is required", kind, snapshotIdx))
				return
			}
			identity := kind + ":" + id
			location := fmt.Sprintf("%s (snapshot %d)", identity, snapshotIdx)
			fingerprintBytes, err := json.Marshal(semanticIdentityValue(kind, value))
			if err != nil {
				problems = append(problems, fmt.Sprintf("semantic id %q cannot be fingerprinted: %v", id, err))
				return
			}
			fingerprint := string(fingerprintBytes)
			if prior, exists := seen[id]; exists {
				if prior.identity != identity || prior.fingerprint != fingerprint {
					if priorEntity, ok := prior.value.(contracts.Entity); ok {
						if entity, ok := value.(contracts.Entity); ok && semanticEntitiesCanMerge(priorEntity, entity) {
							return
						}
					}
					if priorEdge, ok := prior.value.(contracts.Edge); ok {
						if edge, ok := value.(contracts.Edge); ok && semanticEdgesCanRekey(priorEdge, edge) {
							return
						}
					}
					problems = append(problems, fmt.Sprintf("semantic id %q collides between %s and %s", id, prior.location, location))
				}
				return
			}
			seen[id] = registration{identity: identity, location: location, fingerprint: fingerprint, value: value}
		}
		for _, entity := range snapshot.Entities {
			register("entity", entity.ID, entity)
		}
		for _, edge := range snapshot.Edges {
			register("edge", edge.ID, edge)
		}
		for _, finding := range snapshot.Findings {
			register("finding", finding.ID, finding)
		}
		for _, question := range snapshot.Questions {
			register("question", question.ID, question)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return semanticProblems(problems)
}

// semanticEntitiesCanMerge permits an exact entity ID to be observed by
// multiple shards when the observations are from the same logical repository,
// use compatible type vocabulary, and agree on the ID/name leaf. The exact ID
// remains authoritative; normalization later merges evidence and fields. A
// different repository, type family, or unrelated name is still a hard
// collision.
func semanticEntitiesCanMerge(left, right contracts.Entity) bool {
	if strings.TrimSpace(left.ID) == "" || strings.TrimSpace(left.ID) != strings.TrimSpace(right.ID) {
		return false
	}
	if normalizeSemanticType(left.Type) != normalizeSemanticType(right.Type) {
		return false
	}
	leftRepo := semanticLogicalRepo(left.Provenance.Evidence)
	rightRepo := semanticLogicalRepo(right.Provenance.Evidence)
	if leftRepo == "" || leftRepo != rightRepo {
		return false
	}
	return semanticNameAgreesWithID(left.ID, left.Name) && semanticNameAgreesWithID(left.ID, right.Name)
}

// semanticEdgesCanRekey permits a weak short edge ID to be reused by shards
// when the observations are from the same logical repository and share the
// same relation type. Normalization then derives a canonical ID from each
// endpoint pair, preserving both claims without overwriting either one.
func semanticEdgesCanRekey(left, right contracts.Edge) bool {
	if strings.TrimSpace(left.ID) == "" || strings.TrimSpace(left.ID) != strings.TrimSpace(right.ID) {
		return false
	}
	if normalizeSemanticType(left.Type) != normalizeSemanticType(right.Type) {
		return false
	}
	leftRepo := semanticLogicalRepo(left.Provenance.Evidence)
	rightRepo := semanticLogicalRepo(right.Provenance.Evidence)
	return leftRepo != "" && leftRepo == rightRepo
}

func normalizeSemanticType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "database", "data-store", "data store":
		return "datastore"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func semanticLogicalRepo(evidence []contracts.Evidence) string {
	if len(evidence) == 0 {
		return ""
	}
	repo := strings.TrimSpace(filepath.Base(strings.TrimSpace(evidence[0].Repo)))
	if repo == "" || repo == "." {
		return ""
	}
	if dash := strings.LastIndex(repo, "-"); dash > 0 && dash < len(repo)-1 {
		suffix := repo[dash+1:]
		if len(suffix) >= 7 && isHexToken(suffix) {
			repo = repo[:dash]
		}
	}
	return strings.ToLower(repo)
}

func semanticNameAgreesWithID(id, name string) bool {
	idLeaf := strings.TrimSpace(id)
	if splitAt := strings.LastIndexAny(idLeaf, ".:/\\"); splitAt >= 0 {
		idLeaf = idLeaf[splitAt+1:]
	}
	idToken := semanticNameToken(idLeaf)
	nameToken := semanticNameToken(name)
	if idToken == "" || nameToken == "" {
		return false
	}
	if idToken == nameToken || strings.Contains(nameToken, idToken) || strings.Contains(idToken, nameToken) {
		return true
	}
	for _, suffix := range []string{"database", "datastore", "db", "service"} {
		if strings.HasSuffix(idToken, suffix) && len(idToken) > len(suffix) {
			stem := strings.TrimSuffix(idToken, suffix)
			if strings.Contains(nameToken, stem) {
				return true
			}
		}
	}
	return false
}

func semanticNameToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isHexToken(value string) bool {
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// semanticIdentityValue intentionally excludes provenance. The same logical
// object may be observed by multiple shards with different evidence; those
// evidence records are merged during aggregation. Core identity fields must
// still agree, otherwise the duplicate ID is a conflicting graph claim.
func semanticIdentityValue(kind string, value any) any {
	switch kind {
	case "entity":
		entity, ok := value.(contracts.Entity)
		if ok {
			entity.Provenance = contracts.Provenance{}
			return entity
		}
	case "edge":
		edge, ok := value.(contracts.Edge)
		if ok {
			edge.Provenance = contracts.Provenance{}
			return edge
		}
	case "finding":
		finding, ok := value.(contracts.Finding)
		if ok {
			finding.Provenance = contracts.Provenance{}
			return finding
		}
	}
	return value
}

func validateSemanticIDCollisions(snapshot contracts.SemanticSnapshot) []string {
	if err := ValidateSemanticIDCollisions(snapshot); err != nil {
		return []string{err.Error()}
	}
	return nil
}

func semanticProblems(problems []string) error {
	sort.Strings(problems)
	return fmt.Errorf("semantic envelope is invalid: %s", strings.Join(problems, "; "))
}

// ValidateSemanticEnvelopeJSON applies the W24D unknown-field policy to the
// semantic fragment while preserving arbitrary keys inside `attributes`.
func ValidateSemanticEnvelopeJSON(raw []byte) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return fmt.Errorf("decode semantic envelope: %w", err)
	}
	semanticRaw, ok := root["semantic"]
	if !ok {
		return nil
	}
	var semantic map[string]json.RawMessage
	if err := json.Unmarshal(semanticRaw, &semantic); err != nil {
		return fmt.Errorf("decode semantic envelope: %w", err)
	}
	allowed := map[string]map[string]struct{}{
		"semantic":   {"coverage": {}, "questions": {}, "entities": {}, "edges": {}, "findings": {}},
		"coverage":   {"observed": {}, "missing": {}, "notes": {}, "covered_topics": {}},
		"question":   {"id": {}, "text": {}, "priority": {}, "related_ids": {}, "question": {}, "confidence": {}},
		"entity":     {"id": {}, "type": {}, "name": {}, "aliases": {}, "tags": {}, "attributes": {}, "owner_team_id": {}, "provenance": {}, "kind": {}, "repo": {}, "path": {}, "evidence": {}},
		"edge":       {"id": {}, "type": {}, "from": {}, "to": {}, "name": {}, "attributes": {}, "provenance": {}, "relation": {}, "source": {}, "target": {}},
		"finding":    {"id": {}, "severity": {}, "title": {}, "description": {}, "rule_id": {}, "related_ids": {}, "provenance": {}, "summary": {}, "inference": {}, "evidence_citation_ids": {}, "confidence": {}},
		"provenance": {"kind": {}, "confidence": {}, "evidence": {}},
		"evidence":   {"repo": {}, "ref": {}, "path": {}, "lines": {}, "excerpt_hash": {}, "excerpt": {}, "citation_id": {}},
		"line_range": {"start": {}, "end": {}},
	}
	problems := []string{}
	check := func(kind string, value map[string]json.RawMessage, label string) {
		for key := range value {
			if _, ok := allowed[kind][key]; !ok {
				problems = append(problems, fmt.Sprintf("%s contains unknown field %q", label, key))
			}
		}
	}
	check("semantic", semantic, "semantic")
	decodeObjects := func(key, kind string) []map[string]json.RawMessage {
		var items []json.RawMessage
		if err := json.Unmarshal(semantic[key], &items); err != nil {
			return nil
		}
		objects := make([]map[string]json.RawMessage, 0, len(items))
		for idx, item := range items {
			var object map[string]json.RawMessage
			if err := json.Unmarshal(item, &object); err != nil {
				continue
			}
			check(kind, object, fmt.Sprintf("semantic.%s[%d]", key, idx))
			objects = append(objects, object)
		}
		return objects
	}
	walkProvenance := func(object map[string]json.RawMessage, label string) {
		provenanceRaw, ok := object["provenance"]
		if !ok {
			return
		}
		var provenance map[string]json.RawMessage
		if json.Unmarshal(provenanceRaw, &provenance) != nil {
			return
		}
		check("provenance", provenance, label+".provenance")
		var evidenceItems []json.RawMessage
		if json.Unmarshal(provenance["evidence"], &evidenceItems) != nil {
			return
		}
		for idx, item := range evidenceItems {
			var evidence map[string]json.RawMessage
			if json.Unmarshal(item, &evidence) != nil {
				continue
			}
			check("evidence", evidence, fmt.Sprintf("%s.provenance.evidence[%d]", label, idx))
			if linesRaw, ok := evidence["lines"]; ok {
				var lines map[string]json.RawMessage
				if json.Unmarshal(linesRaw, &lines) == nil {
					check("line_range", lines, fmt.Sprintf("%s.provenance.evidence[%d].lines", label, idx))
				}
			}
		}
	}
	decodeObjects("questions", "question")
	for idx, object := range decodeObjects("entities", "entity") {
		walkProvenance(object, fmt.Sprintf("semantic.entities[%d]", idx))
	}
	for idx, object := range decodeObjects("edges", "edge") {
		walkProvenance(object, fmt.Sprintf("semantic.edges[%d]", idx))
	}
	for idx, object := range decodeObjects("findings", "finding") {
		walkProvenance(object, fmt.Sprintf("semantic.findings[%d]", idx))
	}
	if coverage := semantic["coverage"]; coverage != nil {
		var object map[string]json.RawMessage
		if json.Unmarshal(coverage, &object) == nil {
			check("coverage", object, "semantic.coverage")
		}
	}
	if len(problems) > 0 {
		return semanticProblems(problems)
	}
	return nil
}
