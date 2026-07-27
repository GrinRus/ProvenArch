package api

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"gopkg.in/yaml.v3"
)

type knowledgeResponse struct {
	Version     int                 `json:"version"`
	GeneratedAt string              `json:"generated_at"`
	SourceMode  string              `json:"source_mode"`
	Status      string              `json:"status"`
	Entities    []knowledgeEntity   `json:"entities"`
	Edges       []knowledgeEdge     `json:"edges"`
	Artifacts   []knowledgeArtifact `json:"artifacts"`
	Issues      []knowledgeIssue    `json:"issues"`
}

type knowledgeEntity struct {
	contracts.Entity
	Path string `json:"path"`
}

type knowledgeEdge struct {
	contracts.Edge
	Path string `json:"path"`
}

type knowledgeArtifact struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type knowledgeIssue struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleKnowledge(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	payload := collectKnowledge(s.getWorkspace().Path)
	writeJSON(writer, http.StatusOK, payload)
}

func collectKnowledge(root string) knowledgeResponse {
	response := knowledgeResponse{
		Version:     1,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		SourceMode:  evidenceAuthorityPromotedCurrent,
		Entities:    []knowledgeEntity{},
		Edges:       []knowledgeEdge{},
		Artifacts:   []knowledgeArtifact{},
		Issues:      []knowledgeIssue{},
	}
	entityIDs := map[string]struct{}{}
	response.Entities = readKnowledgeEntities(root, &response.Issues, entityIDs)
	response.Edges = readKnowledgeEdges(root, &response.Issues, entityIDs)
	response.Artifacts = inventoryKnowledgeArtifacts(root, &response.Issues)
	sort.Slice(response.Issues, func(i, j int) bool {
		if response.Issues[i].Path == response.Issues[j].Path {
			return response.Issues[i].Code < response.Issues[j].Code
		}
		return response.Issues[i].Path < response.Issues[j].Path
	})
	switch {
	case len(response.Issues) > 0:
		response.Status = "partial"
	case len(response.Entities)+len(response.Edges)+len(response.Artifacts) > 0:
		response.Status = "available"
	default:
		response.Status = "unavailable"
	}
	return response
}

func readKnowledgeEntities(root string, issues *[]knowledgeIssue, ids map[string]struct{}) []knowledgeEntity {
	items := []knowledgeEntity{}
	for _, relative := range yamlFiles(root, "model/entities", issues) {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.file_unreadable", Path: relative, Message: err.Error()})
			continue
		}
		var entity contracts.Entity
		if err := yaml.Unmarshal(content, &entity); err != nil {
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.entity_malformed", Path: relative, Message: err.Error()})
			continue
		}
		if strings.TrimSpace(entity.ID) == "" || strings.TrimSpace(entity.Type) == "" || strings.TrimSpace(entity.Name) == "" {
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.entity_malformed", Path: relative, Message: "entity requires non-empty id, type and name"})
			continue
		}
		if err := validateKnowledgeProvenance(entity.Provenance); err != nil {
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.entity_malformed", Path: relative, Message: err.Error()})
			continue
		}
		if _, duplicate := ids[entity.ID]; duplicate {
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.entity_duplicate", Path: relative, Message: fmt.Sprintf("duplicate entity id %q", entity.ID)})
			continue
		}
		ids[entity.ID] = struct{}{}
		items = append(items, knowledgeEntity{Entity: entity, Path: relative})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func readKnowledgeEdges(root string, issues *[]knowledgeIssue, entityIDs map[string]struct{}) []knowledgeEdge {
	items := []knowledgeEdge{}
	for _, relative := range yamlFiles(root, "model/edges", issues) {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.file_unreadable", Path: relative, Message: err.Error()})
			continue
		}
		var edge contracts.Edge
		if err := yaml.Unmarshal(content, &edge); err != nil {
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.edge_malformed", Path: relative, Message: err.Error()})
			continue
		}
		if strings.TrimSpace(edge.ID) == "" || strings.TrimSpace(edge.Type) == "" || strings.TrimSpace(edge.From) == "" || strings.TrimSpace(edge.To) == "" {
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.edge_malformed", Path: relative, Message: "edge requires non-empty id, type, from and to"})
			continue
		}
		if err := validateKnowledgeProvenance(edge.Provenance); err != nil {
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.edge_malformed", Path: relative, Message: err.Error()})
			continue
		}
		_, fromOK := entityIDs[edge.From]
		_, toOK := entityIDs[edge.To]
		if !fromOK || !toOK {
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.edge_reference_missing", Path: relative, Message: fmt.Sprintf("edge references missing entities: from=%q to=%q", edge.From, edge.To)})
			continue
		}
		items = append(items, knowledgeEdge{Edge: edge, Path: relative})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func validateKnowledgeProvenance(provenance contracts.Provenance) error {
	switch provenance.Kind {
	case "observation", "inference", "assertion":
	default:
		return fmt.Errorf("provenance.kind must be observation, inference or assertion")
	}
	if provenance.Confidence < 0 || provenance.Confidence > 1 {
		return fmt.Errorf("provenance.confidence must be between 0 and 1")
	}
	if provenance.Kind == "observation" && len(provenance.Evidence) == 0 {
		return fmt.Errorf("observation provenance requires evidence")
	}
	for _, evidence := range provenance.Evidence {
		if strings.TrimSpace(evidence.Repo) == "" || strings.TrimSpace(evidence.Path) == "" {
			return fmt.Errorf("provenance evidence requires non-empty repo and path")
		}
	}
	return nil
}

func yamlFiles(root string, relativeRoot string, issues *[]knowledgeIssue) []string {
	paths := []string{}
	absRoot := filepath.Join(root, filepath.FromSlash(relativeRoot))
	if _, err := os.Stat(absRoot); os.IsNotExist(err) {
		return paths
	}
	err := filepath.WalkDir(absRoot, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			relative := relativeRoot
			if rel, err := filepath.Rel(root, current); err == nil {
				relative = filepath.ToSlash(rel)
			}
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.file_unreadable", Path: relative, Message: walkErr.Error()})
			return nil
		}
		if entry.Type().IsRegular() && (strings.HasSuffix(strings.ToLower(entry.Name()), ".yaml") || strings.HasSuffix(strings.ToLower(entry.Name()), ".yml")) {
			rel, err := filepath.Rel(root, current)
			if err == nil {
				paths = append(paths, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		*issues = append(*issues, knowledgeIssue{Code: "knowledge.inventory_unreadable", Path: relativeRoot, Message: err.Error()})
	}
	sort.Strings(paths)
	return paths
}

func inventoryKnowledgeArtifacts(root string, issues *[]knowledgeIssue) []knowledgeArtifact {
	items := []knowledgeArtifact{}
	for _, relativeRoot := range []string{"model", "proposals", "reports"} {
		absRoot := filepath.Join(root, relativeRoot)
		if _, err := os.Stat(absRoot); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(absRoot, func(current string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				rel, _ := filepath.Rel(root, current)
				*issues = append(*issues, knowledgeIssue{Code: "knowledge.file_unreadable", Path: filepath.ToSlash(rel), Message: walkErr.Error()})
				return nil
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(root, current)
			if err != nil {
				return nil
			}
			canonical := filepath.ToSlash(rel)
			if canonical == "reports/taskruns" || strings.HasPrefix(canonical, "reports/taskruns/") {
				return nil
			}
			if file, err := os.Open(current); err != nil {
				*issues = append(*issues, knowledgeIssue{Code: "knowledge.file_unreadable", Path: filepath.ToSlash(rel), Message: err.Error()})
				return nil
			} else {
				_ = file.Close()
			}
			items = append(items, knowledgeArtifact{Path: canonical, Kind: knowledgeArtifactKind(canonical), Name: entry.Name()})
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			*issues = append(*issues, knowledgeIssue{Code: "knowledge.inventory_unreadable", Path: relativeRoot, Message: err.Error()})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return items
}

func knowledgeArtifactKind(path string) string {
	switch {
	case strings.HasPrefix(path, "model/entities/"):
		return "entity"
	case strings.HasPrefix(path, "model/edges/"):
		return "edge"
	case strings.HasPrefix(path, "proposals/"):
		return "proposal"
	case strings.HasPrefix(path, "reports/"):
		return "report"
	default:
		return "model"
	}
}
