package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/orchestrator"
)

type architectureResponse struct {
	Version     int                         `json:"version"`
	GeneratedAt string                      `json:"generated_at"`
	Authority   architectureAuthority       `json:"authority"`
	Status      string                      `json:"status"`
	Counts      architectureCounts          `json:"counts"`
	Views       map[string]architectureView `json:"views"`
	Artifacts   []knowledgeArtifact         `json:"artifacts"`
	Exports     architectureExports         `json:"exports"`
	Issues      []knowledgeIssue            `json:"issues"`
	Comparison  architectureComparison      `json:"comparison"`
}

type architectureComparison struct {
	Available     bool                             `json:"available"`
	BaselineRunID string                           `json:"baseline_run_id,omitempty"`
	CurrentRunID  string                           `json:"current_run_id,omitempty"`
	Reason        string                           `json:"reason,omitempty"`
	Categories    map[string]architectureChangeSet `json:"categories"`
}

type architectureChangeSet struct {
	Added   []architectureChangeItem `json:"added"`
	Changed []architectureChangeItem `json:"changed"`
	Removed []architectureChangeItem `json:"removed"`
}

type architectureChangeItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

type architectureExports struct {
	HomePath       string   `json:"home_path,omitempty"`
	C4MermaidPaths []string `json:"c4_mermaid_paths"`
}

type architectureAuthority struct {
	Mode        string `json:"mode"`
	SourceRunID string `json:"source_run_id,omitempty"`
	PromotedAt  string `json:"promoted_at,omitempty"`
	Freshness   string `json:"freshness"`
}

type architectureCounts struct {
	Entities int `json:"entities"`
	Edges    int `json:"edges"`
	Evidence int `json:"evidence"`
	Issues   int `json:"issues"`
}

type architectureView struct {
	Level             string             `json:"level"`
	Available         bool               `json:"available"`
	UnavailableReason string             `json:"unavailable_reason,omitempty"`
	Nodes             []architectureNode `json:"nodes"`
	Edges             []architectureEdge `json:"edges"`
}

type architectureNode struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	Type             string               `json:"type"`
	OwnerTeamID      string               `json:"owner_team_id,omitempty"`
	Tags             []string             `json:"tags,omitempty"`
	Confidence       float64              `json:"confidence"`
	ProvenanceKind   string               `json:"provenance_kind"`
	Evidence         []contracts.Evidence `json:"evidence,omitempty"`
	Path             string               `json:"path"`
	AvailableLevels  []string             `json:"available_levels,omitempty"`
	Repositories     []string             `json:"repositories,omitempty"`
	RelatedFindings  []string             `json:"related_findings,omitempty"`
	RelatedQuestions []string             `json:"related_questions,omitempty"`
}

type architectureEdge struct {
	ID               string               `json:"id"`
	From             string               `json:"from"`
	To               string               `json:"to"`
	Type             string               `json:"type"`
	Name             string               `json:"name,omitempty"`
	Confidence       float64              `json:"confidence"`
	ProvenanceKind   string               `json:"provenance_kind"`
	Evidence         []contracts.Evidence `json:"evidence,omitempty"`
	Path             string               `json:"path"`
	Repositories     []string             `json:"repositories,omitempty"`
	RelatedFindings  []string             `json:"related_findings,omitempty"`
	RelatedQuestions []string             `json:"related_questions,omitempty"`
}

func (s *Server) handleArchitecture(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	snapshot := s.sessionSnapshot()
	knowledge := collectKnowledge(snapshot.Workspace.Path)
	response := buildArchitectureResponse(knowledge)
	if snapshot.Service != nil {
		promotedRuns := []orchestrator.RunInfo{}
		for _, run := range snapshot.Service.ListRuns(0) {
			if run.Status != orchestrator.RunStatusSucceeded || (run.Pipeline != string(orchestrator.PipelineInit) && run.Pipeline != string(orchestrator.PipelineRefresh)) {
				continue
			}
			artifacts, _ := snapshot.Service.GetRunArtifacts(run.RunID)
			if hasArchitectureSnapshot(artifacts) {
				promotedRuns = append(promotedRuns, run)
			}
		}
		if len(promotedRuns) > 0 {
			current := promotedRuns[0]
			response.Authority.SourceRunID = current.RunID
			if current.FinishedAt != nil {
				response.Authority.PromotedAt = current.FinishedAt.UTC().Format(time.RFC3339)
				response.Authority.Freshness = architectureFreshness(*current.FinishedAt)
			}
			if len(promotedRuns) > 1 {
				response.Comparison = comparePromotedArchitectures(snapshot.Workspace.Path, promotedRuns[0].RunID, promotedRuns[1].RunID)
			} else {
				response.Comparison = emptyArchitectureComparison("No previous promoted architecture is available for comparison.")
				response.Comparison.CurrentRunID = current.RunID
			}
		}
	}
	writeJSON(writer, http.StatusOK, response)
}

func buildArchitectureResponse(knowledge knowledgeResponse) architectureResponse {
	response := architectureResponse{Version: 1, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Authority: architectureAuthority{Mode: evidenceAuthorityPromotedCurrent, Freshness: "unknown"}, Status: knowledge.Status, Artifacts: knowledge.Artifacts, Issues: knowledge.Issues, Views: map[string]architectureView{}, Exports: architectureExports{C4MermaidPaths: []string{}}, Comparison: emptyArchitectureComparison("A comparison will be available after two promoted architecture generations.")}
	for _, artifact := range knowledge.Artifacts {
		if artifact.Path == "reports/as-is/overview.md" {
			response.Exports.HomePath = artifact.Path
		}
		if strings.HasPrefix(artifact.Path, "reports/diagrams/") && strings.HasSuffix(artifact.Path, ".mmd") {
			response.Exports.C4MermaidPaths = append(response.Exports.C4MermaidPaths, artifact.Path)
		}
	}
	sort.Strings(response.Exports.C4MermaidPaths)
	evidenceCount := 0
	for _, entity := range knowledge.Entities {
		evidenceCount += len(entity.Provenance.Evidence)
	}
	for _, edge := range knowledge.Edges {
		evidenceCount += len(edge.Provenance.Evidence)
	}
	response.Counts = architectureCounts{Entities: len(knowledge.Entities), Edges: len(knowledge.Edges), Evidence: evidenceCount, Issues: len(knowledge.Issues)}
	for _, level := range []string{"context", "container", "component", "code"} {
		response.Views[level] = architectureViewFor(level, knowledge.Entities, knowledge.Edges)
	}
	return response
}

func hasArchitectureSnapshot(artifacts []orchestrator.Artifact) bool {
	for _, artifact := range artifacts {
		if artifact.Kind == "architecture-snapshot" {
			return true
		}
	}
	return false
}

func emptyArchitectureComparison(reason string) architectureComparison {
	categories := map[string]architectureChangeSet{}
	for _, category := range []string{"entities", "edges", "findings", "gaps"} {
		categories[category] = architectureChangeSet{Added: []architectureChangeItem{}, Changed: []architectureChangeItem{}, Removed: []architectureChangeItem{}}
	}
	return architectureComparison{Available: false, Reason: reason, Categories: categories}
}

func comparePromotedArchitectures(workspaceRoot, currentRunID, baselineRunID string) architectureComparison {
	currentRoot := filepath.Join(workspaceRoot, "reports", "taskruns", currentRunID, "promoted-snapshot")
	baselineRoot := filepath.Join(workspaceRoot, "reports", "taskruns", baselineRunID, "promoted-snapshot")
	current := collectKnowledge(currentRoot)
	baseline := collectKnowledge(baselineRoot)
	comparison := emptyArchitectureComparison("")
	comparison.Available = true
	comparison.CurrentRunID = currentRunID
	comparison.BaselineRunID = baselineRunID
	comparison.Categories["entities"] = compareArchitectureValues(current.Entities, baseline.Entities, func(item knowledgeEntity) (string, string, string) { return item.ID, item.Name, item.Path })
	comparison.Categories["edges"] = compareArchitectureValues(current.Edges, baseline.Edges, func(item knowledgeEdge) (string, string, string) { return item.ID, item.Name, item.Path })
	currentFiles := readArchitectureSnapshotFiles(currentRoot)
	baselineFiles := readArchitectureSnapshotFiles(baselineRoot)
	comparison.Categories["findings"] = compareArchitectureFiles(currentFiles, baselineFiles, "reports/findings/")
	comparison.Categories["gaps"] = compareArchitectureFiles(currentFiles, baselineFiles, "reports/coverage/")
	return comparison
}

func compareArchitectureValues[T any](current, baseline []T, identity func(T) (string, string, string)) architectureChangeSet {
	type value struct{ name, path, digest string }
	index := func(items []T) map[string]value {
		result := map[string]value{}
		for _, item := range items {
			id, name, path := identity(item)
			raw, _ := json.Marshal(item)
			digest := sha256.Sum256(raw)
			result[id] = value{name: name, path: path, digest: hex.EncodeToString(digest[:])}
		}
		return result
	}
	currentIndex, baselineIndex := index(current), index(baseline)
	result := architectureChangeSet{Added: []architectureChangeItem{}, Changed: []architectureChangeItem{}, Removed: []architectureChangeItem{}}
	for id, item := range currentIndex {
		before, exists := baselineIndex[id]
		change := architectureChangeItem{ID: id, Name: item.name, Path: item.path}
		if !exists {
			result.Added = append(result.Added, change)
		} else if before.digest != item.digest {
			result.Changed = append(result.Changed, change)
		}
	}
	for id, item := range baselineIndex {
		if _, exists := currentIndex[id]; !exists {
			result.Removed = append(result.Removed, architectureChangeItem{ID: id, Name: item.name, Path: item.path})
		}
	}
	sortArchitectureChanges(&result)
	return result
}

func readArchitectureSnapshotFiles(root string) map[string]string {
	raw, err := os.ReadFile(filepath.Join(root, "architecture-snapshot.json"))
	if err != nil {
		return map[string]string{}
	}
	var manifest struct {
		Files []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	if json.Unmarshal(raw, &manifest) != nil {
		return map[string]string{}
	}
	result := map[string]string{}
	for _, item := range manifest.Files {
		result[item.Path] = item.SHA256
	}
	return result
}

func compareArchitectureFiles(current, baseline map[string]string, prefix string) architectureChangeSet {
	result := architectureChangeSet{Added: []architectureChangeItem{}, Changed: []architectureChangeItem{}, Removed: []architectureChangeItem{}}
	for path, digest := range current {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		before, exists := baseline[path]
		item := architectureChangeItem{ID: path, Name: filepath.Base(path), Path: path}
		if !exists {
			result.Added = append(result.Added, item)
		} else if before != digest {
			result.Changed = append(result.Changed, item)
		}
	}
	for path := range baseline {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		if _, exists := current[path]; !exists {
			result.Removed = append(result.Removed, architectureChangeItem{ID: path, Name: filepath.Base(path), Path: path})
		}
	}
	sortArchitectureChanges(&result)
	return result
}

func sortArchitectureChanges(value *architectureChangeSet) {
	sort.Slice(value.Added, func(i, j int) bool { return value.Added[i].ID < value.Added[j].ID })
	sort.Slice(value.Changed, func(i, j int) bool { return value.Changed[i].ID < value.Changed[j].ID })
	sort.Slice(value.Removed, func(i, j int) bool { return value.Removed[i].ID < value.Removed[j].ID })
}

func architectureViewFor(level string, entities []knowledgeEntity, edges []knowledgeEdge) architectureView {
	allowed := map[string]bool{}
	for _, entity := range entities {
		if architectureTypeVisible(level, entity.Type) {
			allowed[entity.ID] = true
		}
	}
	nodes := []architectureNode{}
	for _, entity := range entities {
		if !allowed[entity.ID] {
			continue
		}
		nodes = append(nodes, architectureNode{ID: entity.ID, Name: entity.Name, Type: entity.Type, OwnerTeamID: entity.OwnerTeamID, Tags: append([]string(nil), entity.Tags...), Confidence: entity.Provenance.Confidence, ProvenanceKind: entity.Provenance.Kind, Evidence: append([]contracts.Evidence(nil), entity.Provenance.Evidence...), Path: entity.Path, AvailableLevels: architectureLevelsForType(entity.Type), Repositories: evidenceRepositories(entity.Provenance.Evidence), RelatedFindings: []string{}, RelatedQuestions: []string{}})
	}
	viewEdges := []architectureEdge{}
	for _, edge := range edges {
		if !allowed[edge.From] || !allowed[edge.To] {
			continue
		}
		viewEdges = append(viewEdges, architectureEdge{ID: edge.ID, From: edge.From, To: edge.To, Type: edge.Type, Name: edge.Name, Confidence: edge.Provenance.Confidence, ProvenanceKind: edge.Provenance.Kind, Evidence: append([]contracts.Evidence(nil), edge.Provenance.Evidence...), Path: edge.Path, Repositories: evidenceRepositories(edge.Provenance.Evidence), RelatedFindings: []string{}, RelatedQuestions: []string{}})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(viewEdges, func(i, j int) bool { return viewEdges[i].ID < viewEdges[j].ID })
	view := architectureView{Level: level, Available: len(nodes) > 0, Nodes: nodes, Edges: viewEdges}
	if !view.Available {
		view.UnavailableReason = "No validated entities are available for this C4 level."
	}
	return view
}

func evidenceRepositories(evidence []contracts.Evidence) []string {
	set := map[string]struct{}{}
	for _, item := range evidence {
		if repo := strings.TrimSpace(item.Repo); repo != "" {
			set[repo] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for repo := range set {
		result = append(result, repo)
	}
	sort.Strings(result)
	return result
}

func architectureTypeVisible(level, entityType string) bool {
	switch level {
	case "context":
		return entityType == "service" || entityType == "external.system" || entityType == "team"
	case "container":
		return entityType == "service" || entityType == "datastore" || entityType == "external.system" || entityType == "repo" || entityType == "event.topic"
	case "component":
		return entityType == "api.http" || entityType == "api.grpc" || entityType == "event.topic"
	case "code":
		return entityType == "api.http" || entityType == "api.grpc"
	default:
		return false
	}
}

func architectureLevelsForType(entityType string) []string {
	levels := []string{}
	for _, level := range []string{"context", "container", "component", "code"} {
		if architectureTypeVisible(level, entityType) {
			levels = append(levels, level)
		}
	}
	return levels
}

func architectureFreshness(promotedAt time.Time) string {
	age := time.Since(promotedAt)
	if age < 0 {
		return "current"
	}
	if age <= 24*time.Hour {
		return "current"
	}
	if age <= 7*24*time.Hour {
		return "recent"
	}
	return "stale"
}
