package reports

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

const (
	maxContextInternalNodes = 14
	maxContextInternalEdges = 24
)

func (c Compiler) CompileC4Diagrams(entities []contracts.Entity, edges []contracts.Edge) ([]Artifact, error) {
	entities = uniqueEntitiesByID(entities)
	entityByID := make(map[string]contracts.Entity, len(entities))
	for _, entity := range entities {
		entityByID[entity.ID] = entity
	}

	services := filterEntitiesByType(entities, "service")
	externalSystems := filterEntitiesByType(entities, "external.system")
	datastores := filterEntitiesByType(entities, "datastore")
	teams := filterEntitiesByType(entities, "team")

	artifacts := []Artifact{}

	contextPath := "reports/diagrams/c4-context.mmd"
	contextDiagram := buildC4ContextDiagram(entities, services, externalSystems, teams, edges)
	if err := c.workspace.WriteFile(contextPath, []byte(contextDiagram)); err != nil {
		return nil, err
	}
	artifacts = append(artifacts, Artifact{
		Path:  contextPath,
		Kind:  "diagram",
		Label: "C4 Context",
	})

	containerPath := "reports/diagrams/c4-container.mmd"
	containerDiagram := buildC4ContainerDiagram(services, datastores, externalSystems, edges)
	if err := c.workspace.WriteFile(containerPath, []byte(containerDiagram)); err != nil {
		return nil, err
	}
	artifacts = append(artifacts, Artifact{
		Path:  containerPath,
		Kind:  "diagram",
		Label: "C4 Container",
	})

	componentPaths := []string{}
	codePaths := []string{}
	for _, service := range services {
		componentPath := fmt.Sprintf("reports/diagrams/components/%s.mmd", sanitizeProposalSlug(service.ID))
		componentDiagram := buildC4ComponentDiagram(service, entities, edges)
		if err := c.workspace.WriteFile(componentPath, []byte(componentDiagram)); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, Artifact{
			Path:  componentPath,
			Kind:  "diagram",
			Label: fmt.Sprintf("C4 Component (%s)", service.ID),
		})
		componentPaths = append(componentPaths, componentPath)

		codePath := fmt.Sprintf("reports/diagrams/code/%s.mmd", sanitizeProposalSlug(service.ID))
		codeDiagram := buildC4CodeDiagram(service, entityByID, edges)
		if err := c.workspace.WriteFile(codePath, []byte(codeDiagram)); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, Artifact{
			Path:  codePath,
			Kind:  "diagram",
			Label: fmt.Sprintf("C4 Code (%s)", service.ID),
		})
		codePaths = append(codePaths, codePath)
	}

	indexPath := "reports/diagrams/index.md"
	indexContent := buildDiagramIndex(contextPath, containerPath, componentPaths, codePaths)
	if err := c.workspace.WriteFile(indexPath, []byte(indexContent)); err != nil {
		return nil, err
	}
	artifacts = append(artifacts, Artifact{
		Path:  indexPath,
		Kind:  "diagram-index",
		Label: "Diagrams Index",
	})

	sortArtifacts(artifacts)
	return artifacts, nil
}

func buildC4ContextDiagram(
	entities []contracts.Entity,
	services []contracts.Entity,
	externalSystems []contracts.Entity,
	teams []contracts.Entity,
	edges []contracts.Edge,
) string {
	builder := strings.Builder{}
	builder.WriteString("flowchart LR\n")
	builder.WriteString("  System[\"Workspace System\"]\n")

	workspaceEntityIDs := map[string]struct{}{}
	for _, entity := range entities {
		if !entityHasEvidence(entity) {
			continue
		}
		entityType := strings.TrimSpace(strings.ToLower(entity.Type))
		if entityType == "external.system" || entityType == "team" {
			continue
		}
		workspaceEntityIDs[entity.ID] = struct{}{}
	}

	serviceCount := 0
	for _, service := range services {
		if !entityHasEvidence(service) {
			continue
		}
		serviceCount++
	}

	if serviceCount > 0 {
		builder.WriteString(fmt.Sprintf("  SystemNote[\"Evidence-backed services: %d\"]\n", serviceCount))
		builder.WriteString("  System --- SystemNote\n")
	}

	externalNodeIDs := map[string]string{}
	externalIDs := []string{}
	externalCount := 0
	for _, ext := range externalSystems {
		if !entityHasEvidence(ext) {
			continue
		}
		externalCount++
		nodeID := mermaidNodeID("ext", ext.ID)
		externalNodeIDs[ext.ID] = nodeID
		externalIDs = append(externalIDs, ext.ID)
		builder.WriteString(fmt.Sprintf("  %s[\"External: %s\"]\n", nodeID, escapeMermaidLabel(ext.Name)))
	}
	if externalCount == 0 {
		builder.WriteString("  GapExternal[\"Gap: no evidence-backed external systems\"]\n")
		builder.WriteString("  System -.-> GapExternal\n")
	}

	teamNodeIDs := map[string]string{}
	teamIDs := []string{}
	teamCount := 0
	for _, team := range teams {
		if !entityHasEvidence(team) {
			continue
		}
		teamCount++
		nodeID := mermaidNodeID("team", team.ID)
		teamNodeIDs[team.ID] = nodeID
		teamIDs = append(teamIDs, team.ID)
		builder.WriteString(fmt.Sprintf("  %s[\"Actor: %s\"]\n", nodeID, escapeMermaidLabel(team.Name)))
	}
	if teamCount == 0 {
		builder.WriteString("  GapActors[\"Gap: no evidence-backed actors\"]\n")
		builder.WriteString("  GapActors -.-> System\n")
	}

	externalRelations := map[string]systemRelation{}
	teamRelations := map[string]systemRelation{}
	for _, edge := range edges {
		if !edgeHasEvidence(edge) {
			continue
		}
		from := strings.TrimSpace(edge.From)
		to := strings.TrimSpace(edge.To)
		if from == "" || to == "" {
			continue
		}
		if _, wsFrom := workspaceEntityIDs[from]; wsFrom {
			if _, isExternal := externalNodeIDs[to]; isExternal {
				rel := externalRelations[to]
				rel.systemToNode = true
				externalRelations[to] = rel
			}
			if _, isTeam := teamNodeIDs[to]; isTeam {
				rel := teamRelations[to]
				rel.systemToNode = true
				teamRelations[to] = rel
			}
		}
		if _, wsTo := workspaceEntityIDs[to]; wsTo {
			if _, isExternal := externalNodeIDs[from]; isExternal {
				rel := externalRelations[from]
				rel.nodeToSystem = true
				externalRelations[from] = rel
			}
			if _, isTeam := teamNodeIDs[from]; isTeam {
				rel := teamRelations[from]
				rel.nodeToSystem = true
				teamRelations[from] = rel
			}
		}
	}

	relationCount := 0
	for _, externalID := range externalIDs {
		if writeSystemRelation(&builder, "System", externalNodeIDs[externalID], externalRelations[externalID]) {
			relationCount++
		}
	}
	for _, teamID := range teamIDs {
		if writeSystemRelation(&builder, "System", teamNodeIDs[teamID], teamRelations[teamID]) {
			relationCount++
		}
	}
	if relationCount == 0 || (externalCount == 0 && serviceCount > 1) {
		relationCount += writeInternalContextFallback(&builder, services, filterEntitiesByType(entities, "datastore"), edges)
	}
	if relationCount == 0 {
		builder.WriteString("  GapRelations[\"Gap: no evidence-backed relationships\"]\n")
		builder.WriteString("  System -.-> GapRelations\n")
	}
	return builder.String()
}

func uniqueEntitiesByID(entities []contracts.Entity) []contracts.Entity {
	byID := make(map[string]contracts.Entity, len(entities))
	for _, entity := range entities {
		id := strings.TrimSpace(entity.ID)
		if id == "" {
			continue
		}
		entity.ID = id
		existing, ok := byID[id]
		if !ok || (!entityHasEvidence(existing) && entityHasEvidence(entity)) {
			byID[id] = entity
		}
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]contracts.Entity, 0, len(ids))
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out
}

func buildC4ContainerDiagram(
	services []contracts.Entity,
	datastores []contracts.Entity,
	externalSystems []contracts.Entity,
	edges []contracts.Edge,
) string {
	builder := strings.Builder{}
	builder.WriteString("flowchart LR\n")
	builder.WriteString("  subgraph Workspace[\"Workspace Containers\"]\n")

	nodeKinds := map[string]string{}
	for _, service := range services {
		if !entityHasEvidence(service) {
			continue
		}
		nodeID := mermaidNodeID("svc", service.ID)
		builder.WriteString(fmt.Sprintf("    %s[\"Service: %s\"]\n", nodeID, escapeMermaidLabel(service.Name)))
		nodeKinds[service.ID] = nodeID
	}
	for _, datastore := range datastores {
		if !entityHasEvidence(datastore) {
			continue
		}
		nodeID := mermaidNodeID("db", datastore.ID)
		builder.WriteString(fmt.Sprintf("    %s[(\"Datastore: %s\")]\n", nodeID, escapeMermaidLabel(datastore.Name)))
		nodeKinds[datastore.ID] = nodeID
	}
	builder.WriteString("  end\n")

	for _, ext := range externalSystems {
		if !entityHasEvidence(ext) {
			continue
		}
		nodeID := mermaidNodeID("ext", ext.ID)
		builder.WriteString(fmt.Sprintf("  %s[\"External: %s\"]\n", nodeID, escapeMermaidLabel(ext.Name)))
		nodeKinds[ext.ID] = nodeID
	}

	edgeCount := 0
	for _, edge := range edges {
		if !edgeHasEvidence(edge) {
			continue
		}
		fromNode, okFrom := nodeKinds[edge.From]
		toNode, okTo := nodeKinds[edge.To]
		if !okFrom || !okTo {
			continue
		}
		edgeCount++
		builder.WriteString(fmt.Sprintf("  %s -->|%s| %s\n", fromNode, escapeMermaidLabel(edge.Type), toNode))
	}
	if edgeCount == 0 {
		builder.WriteString("  GapContainerEdges[\"Gap: no evidence-backed container relations\"]\n")
		builder.WriteString("  Workspace -.-> GapContainerEdges\n")
	}
	if len(nodeKinds) == 0 {
		builder.WriteString("  GapContainers[\"Gap: no evidence-backed containers\"]\n")
		builder.WriteString("  Workspace -.-> GapContainers\n")
	}
	return builder.String()
}

func buildC4ComponentDiagram(service contracts.Entity, entities []contracts.Entity, edges []contracts.Edge) string {
	serviceSlug := slugFromServiceID(service.ID)
	componentEntities := []contracts.Entity{}
	componentIDs := map[string]struct{}{
		service.ID: {},
	}
	nodeByEntityID := map[string]string{
		service.ID: mermaidNodeID("svc", service.ID),
	}
	for _, entity := range entities {
		if !entityHasEvidence(entity) {
			continue
		}
		if strings.HasPrefix(entity.Type, "api.") && strings.Contains(entity.ID, "."+serviceSlug+".") {
			componentEntities = append(componentEntities, entity)
			componentIDs[entity.ID] = struct{}{}
			nodeByEntityID[entity.ID] = mermaidNodeID("cmp", entity.ID)
		}
	}
	sort.Slice(componentEntities, func(i, j int) bool {
		return componentEntities[i].ID < componentEntities[j].ID
	})

	builder := strings.Builder{}
	builder.WriteString("flowchart TB\n")
	builder.WriteString(fmt.Sprintf("  subgraph Service[\"Components: %s\"]\n", escapeMermaidLabel(service.Name)))
	serviceNodeID := nodeByEntityID[service.ID]
	builder.WriteString(fmt.Sprintf("    %s[\"Service Core\"]\n", serviceNodeID))

	if len(componentEntities) == 0 {
		builder.WriteString("    GapComponents[\"Gap: no evidence-backed API/Event components\"]\n")
		builder.WriteString(fmt.Sprintf("    %s -.-> GapComponents\n", serviceNodeID))
	} else {
		for _, component := range componentEntities {
			componentNodeID := nodeByEntityID[component.ID]
			builder.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", componentNodeID, escapeMermaidLabel(component.Name)))
			builder.WriteString(fmt.Sprintf("    %s --> %s\n", serviceNodeID, componentNodeID))
		}
	}
	builder.WriteString("  end\n")

	componentEdgeCount := 0
	for _, edge := range edges {
		if !edgeHasEvidence(edge) {
			continue
		}
		_, fromIncluded := componentIDs[edge.From]
		_, toIncluded := componentIDs[edge.To]
		if !fromIncluded || !toIncluded {
			continue
		}
		componentEdgeCount++
		fromNode := nodeByEntityID[edge.From]
		toNode := nodeByEntityID[edge.To]
		if strings.TrimSpace(fromNode) == "" || strings.TrimSpace(toNode) == "" {
			continue
		}
		builder.WriteString(fmt.Sprintf("  %s -->|%s| %s\n", fromNode, escapeMermaidLabel(edge.Type), toNode))
	}
	if componentEdgeCount == 0 {
		builder.WriteString("  GapComponentEdges[\"Gap: no evidence-backed component relations\"]\n")
		builder.WriteString("  Service -.-> GapComponentEdges\n")
	}
	return builder.String()
}

func buildC4CodeDiagram(service contracts.Entity, entityByID map[string]contracts.Entity, edges []contracts.Edge) string {
	builder := strings.Builder{}
	builder.WriteString("flowchart TB\n")
	serviceNodeID := mermaidNodeID("svc", service.ID)
	builder.WriteString(fmt.Sprintf("  %s[\"Code: %s\"]\n", serviceNodeID, escapeMermaidLabel(service.Name)))

	paths := map[string]struct{}{}
	for _, evidence := range service.Provenance.Evidence {
		if top := topPathSegment(evidence.Path); top != "" {
			paths[top] = struct{}{}
		}
	}
	for _, edge := range edges {
		if !edgeHasEvidence(edge) {
			continue
		}
		if edge.From != service.ID && edge.To != service.ID {
			continue
		}
		relatedID := edge.From
		if relatedID == service.ID {
			relatedID = edge.To
		}
		related, ok := entityByID[relatedID]
		if !ok {
			continue
		}
		for _, evidence := range related.Provenance.Evidence {
			if top := topPathSegment(evidence.Path); top != "" {
				paths[top] = struct{}{}
			}
		}
	}

	pathList := make([]string, 0, len(paths))
	for value := range paths {
		pathList = append(pathList, value)
	}
	sort.Strings(pathList)

	if len(pathList) == 0 {
		builder.WriteString("  GapCode[\"Gap: no evidence-backed code paths\"]\n")
		builder.WriteString(fmt.Sprintf("  %s -.-> GapCode\n", serviceNodeID))
		return builder.String()
	}

	for _, path := range pathList {
		nodeID := mermaidNodeID("path", service.ID+"-"+path)
		builder.WriteString(fmt.Sprintf("  %s[\"%s/\"]\n", nodeID, escapeMermaidLabel(path)))
		builder.WriteString(fmt.Sprintf("  %s --> %s\n", serviceNodeID, nodeID))
	}

	return builder.String()
}

func buildDiagramIndex(contextPath string, containerPath string, componentPaths []string, codePaths []string) string {
	sort.Strings(componentPaths)
	sort.Strings(codePaths)

	builder := strings.Builder{}
	builder.WriteString("# C4 Diagrams Index\n\n")
	builder.WriteString("- Generated by ACP Step 2 (`as-is`) with strict evidence-first policy.\n")
	builder.WriteString("- Gaps are rendered explicitly inside diagrams where evidence is insufficient.\n\n")

	builder.WriteString("## Workspace\n\n")
	builder.WriteString(fmt.Sprintf("- Context: `%s`\n", contextPath))
	builder.WriteString(fmt.Sprintf("- Container: `%s`\n", containerPath))

	builder.WriteString("\n## Component\n\n")
	if len(componentPaths) == 0 {
		builder.WriteString("- none\n")
	} else {
		for _, path := range componentPaths {
			builder.WriteString(fmt.Sprintf("- `%s`\n", path))
		}
	}

	builder.WriteString("\n## Code\n\n")
	if len(codePaths) == 0 {
		builder.WriteString("- none\n")
	} else {
		for _, path := range codePaths {
			builder.WriteString(fmt.Sprintf("- `%s`\n", path))
		}
	}
	return builder.String()
}

func entityHasEvidence(entity contracts.Entity) bool {
	if len(entity.Provenance.Evidence) == 0 {
		return false
	}
	for _, evidence := range entity.Provenance.Evidence {
		if strings.TrimSpace(evidence.Repo) != "" || strings.TrimSpace(evidence.Path) != "" {
			return true
		}
	}
	return false
}

func edgeHasEvidence(edge contracts.Edge) bool {
	if len(edge.Provenance.Evidence) == 0 {
		return false
	}
	for _, evidence := range edge.Provenance.Evidence {
		if strings.TrimSpace(evidence.Repo) != "" || strings.TrimSpace(evidence.Path) != "" {
			return true
		}
	}
	return false
}

type systemRelation struct {
	systemToNode bool
	nodeToSystem bool
}

type internalContextNode struct {
	id     string
	nodeID string
	label  string
	shape  string
}

type internalContextRelation struct {
	id    string
	from  string
	to    string
	label string
}

func writeInternalContextFallback(builder *strings.Builder, services []contracts.Entity, datastores []contracts.Entity, edges []contracts.Edge) int {
	if builder == nil {
		return 0
	}
	candidates := map[string]internalContextNode{}
	for _, service := range services {
		if !entityHasEvidence(service) {
			continue
		}
		candidates[service.ID] = internalContextNode{
			id:     service.ID,
			nodeID: mermaidNodeID("ctx", service.ID),
			label:  "Service: " + service.Name,
			shape:  "box",
		}
	}
	for _, datastore := range datastores {
		if !entityHasEvidence(datastore) {
			continue
		}
		candidates[datastore.ID] = internalContextNode{
			id:     datastore.ID,
			nodeID: mermaidNodeID("ctx", datastore.ID),
			label:  "Datastore: " + datastore.Name,
			shape:  "database",
		}
	}
	if len(candidates) == 0 {
		return 0
	}

	relations := make([]internalContextRelation, 0)
	for _, edge := range edges {
		if !edgeHasEvidence(edge) {
			continue
		}
		from := strings.TrimSpace(edge.From)
		to := strings.TrimSpace(edge.To)
		if from == "" || to == "" {
			continue
		}
		if _, ok := candidates[from]; !ok {
			continue
		}
		if _, ok := candidates[to]; !ok {
			continue
		}
		relations = append(relations, internalContextRelation{
			id:    edge.ID,
			from:  from,
			to:    to,
			label: edge.Type,
		})
	}
	if len(relations) == 0 {
		return 0
	}
	sort.Slice(relations, func(i, j int) bool {
		left := relations[i]
		right := relations[j]
		if left.from != right.from {
			return left.from < right.from
		}
		if left.to != right.to {
			return left.to < right.to
		}
		if left.label != right.label {
			return left.label < right.label
		}
		return left.id < right.id
	})

	included := map[string]struct{}{}
	selectedRelations := make([]internalContextRelation, 0, minInt(len(relations), maxContextInternalEdges))
	for _, relation := range relations {
		nextNodes := 0
		if _, ok := included[relation.from]; !ok {
			nextNodes++
		}
		if relation.to != relation.from {
			if _, ok := included[relation.to]; !ok {
				nextNodes++
			}
		}
		if len(included)+nextNodes > maxContextInternalNodes {
			continue
		}
		included[relation.from] = struct{}{}
		included[relation.to] = struct{}{}
		selectedRelations = append(selectedRelations, relation)
		if len(selectedRelations) >= maxContextInternalEdges {
			break
		}
	}
	if len(selectedRelations) == 0 {
		return 0
	}

	nodeIDs := make([]string, 0, len(included))
	for id := range included {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	builder.WriteString("  subgraph InternalContext[\"Evidence-backed workspace internals\"]\n")
	for _, id := range nodeIDs {
		node := candidates[id]
		switch node.shape {
		case "database":
			builder.WriteString(fmt.Sprintf("    %s[(\"%s\")]\n", node.nodeID, escapeMermaidLabel(node.label)))
		default:
			builder.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", node.nodeID, escapeMermaidLabel(node.label)))
		}
	}
	builder.WriteString("  end\n")

	for _, relation := range selectedRelations {
		fromNode := candidates[relation.from].nodeID
		toNode := candidates[relation.to].nodeID
		builder.WriteString(fmt.Sprintf("  %s -->|%s| %s\n", fromNode, escapeMermaidLabel(relation.label), toNode))
	}
	return len(selectedRelations)
}

func writeSystemRelation(builder *strings.Builder, systemNodeID string, targetNodeID string, relation systemRelation) bool {
	if builder == nil {
		return false
	}
	switch {
	case relation.systemToNode && relation.nodeToSystem:
		builder.WriteString(fmt.Sprintf("  %s --> %s\n", systemNodeID, targetNodeID))
		builder.WriteString(fmt.Sprintf("  %s --> %s\n", targetNodeID, systemNodeID))
		return true
	case relation.systemToNode:
		builder.WriteString(fmt.Sprintf("  %s --> %s\n", systemNodeID, targetNodeID))
		return true
	case relation.nodeToSystem:
		builder.WriteString(fmt.Sprintf("  %s --> %s\n", targetNodeID, systemNodeID))
		return true
	default:
		return false
	}
}

func mermaidNodeID(prefix string, value string) string {
	if strings.TrimSpace(value) == "" {
		return prefix + "_unknown"
	}
	return prefix + "_" + strings.ReplaceAll(slugutil.Slugify(value), "-", "_")
}

func escapeMermaidLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	value = strings.ReplaceAll(value, "\"", "'")
	return value
}

func slugFromServiceID(serviceID string) string {
	serviceID = strings.TrimSpace(strings.ToLower(serviceID))
	if strings.HasPrefix(serviceID, "svc.") {
		return strings.TrimPrefix(serviceID, "svc.")
	}
	return serviceID
}

func topPathSegment(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = filepath.ToSlash(path)
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
