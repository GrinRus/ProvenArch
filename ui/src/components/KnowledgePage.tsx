import { useMemo, useRef, useState } from "react";

import type { ArchitectureEdge, ArchitectureLevel, ArchitectureResponse, KnowledgeResponse, WorkspaceHealthResponse } from "../lib/appContracts";
import type { KnowledgeView } from "../lib/appRoutes";
import { architectureFromKnowledge } from "../lib/workspaceApi";
import { ArchitectureMap, levelLabel } from "./ArchitectureMap";

const views: Array<{ id: KnowledgeView; label: string }> = [
  { id: "map", label: "Map" },
  { id: "overview", label: "Overview" },
  { id: "catalog", label: "Catalog" },
  { id: "flows", label: "Flows" },
  { id: "evidence", label: "Evidence" },
];

export function KnowledgePage({
  architecture: providedArchitecture,
  knowledge,
  loading,
  error,
  view,
  selectedEntityID,
  workspaceHealth,
  onViewChange,
  onEntityChange,
  onOpenArtifact,
  onOpenRuns,
}: {
  architecture?: ArchitectureResponse | null;
  knowledge: KnowledgeResponse | null;
  loading: boolean;
  error: string;
  view: KnowledgeView;
  selectedEntityID?: string;
  workspaceHealth?: WorkspaceHealthResponse | null;
  onViewChange: (view: KnowledgeView) => void;
  onEntityChange: (id?: string) => void;
  onOpenArtifact: (path: string) => void;
  onOpenRuns?: () => void;
}) {
	const architecture = providedArchitecture ?? (knowledge ? architectureFromKnowledge(knowledge) : null);
	const activePageView: KnowledgeView = view === "atlas" ? "map" : view === "entities" ? "catalog" : view === "artifacts" ? "evidence" : view;
  const [query, setQuery] = useState("");
  const [level, setLevel] = useState<ArchitectureLevel>("context");
  const [typeFilter, setTypeFilter] = useState("");
  const [repositoryFilter, setRepositoryFilter] = useState("");
  const [fullscreen, setFullscreen] = useState(false);
  const [codeScopeServiceID, setCodeScopeServiceID] = useState<string>();
	const advancedLevelRef = useRef<HTMLDetailsElement>(null);
  const allNodes = useMemo(() => architecture ? uniqueNodes(Object.values(architecture.views).flatMap((item) => item.nodes)) : [], [architecture]);
  const allEdges = useMemo(() => architecture ? uniqueEdges(Object.values(architecture.views).flatMap((item) => item.edges)) : [], [architecture]);
  const selectedNode = allNodes.find((node) => node.id === selectedEntityID);
  const selectedEdge = allEdges.find((edge) => edge.id === selectedEntityID);
  const codeScopeService = allNodes.find((node) => node.id === codeScopeServiceID && node.type === "service");
  const activeView = useMemo(() => {
    const view = architecture?.views[level];
    if (!view || level !== "code" || !codeScopeService) return view;
    const repositories = new Set(codeScopeService.repositories ?? []);
    const nodes = view.nodes.filter((node) => (node.repositories ?? []).some((repository) => repositories.has(repository)));
    const ids = new Set(nodes.map((node) => node.id));
    return { ...view, nodes, edges: view.edges.filter((edge) => ids.has(edge.from) && ids.has(edge.to)), available: nodes.length > 0, unavailable_reason: nodes.length > 0 ? undefined : "No validated code-level interfaces are linked to the selected service repositories." };
  }, [architecture, level, codeScopeService]);
  const typeOptions = useMemo(() => Array.from(new Set(activeView?.nodes.map((node) => node.type) ?? [])).sort(), [activeView]);
  const repositoryOptions = useMemo(() => Array.from(new Set(activeView?.nodes.flatMap((node) => node.repositories ?? []) ?? [])).sort(), [activeView]);
  const mobileNodes = (activeView?.nodes ?? []).filter((node) => (!typeFilter || node.type === typeFilter) && (!repositoryFilter || node.repositories?.includes(repositoryFilter)) && [node.name, node.id, node.type, node.owner_team_id ?? "", ...(node.tags ?? [])].join(" ").toLowerCase().includes(query.trim().toLowerCase()));
  const edges = architecture?.views.container.edges ?? [];
  const services = allNodes.filter((node) => node.type === "service");
  const external = allNodes.filter((node) => node.type === "external.system");
  const datastores = allNodes.filter((node) => node.type === "datastore");

  return (
    <section className={`panel stage-panel architecture-page ${fullscreen ? "is-map-fullscreen" : ""}`} data-testid="knowledge-panel">
      <header className="architecture-header">
        <div>
          <p className="eyebrow">Promoted architecture</p>
          <h1>Architecture</h1>
          <p className="page-purpose">Understand the system, inspect C4 levels and trace every visible fact to repository evidence.</p>
        </div>
        <div className="architecture-authority">
          <span className={`status ${architecture?.status === "available" ? "ok" : architecture?.status === "partial" ? "warn" : "err"}`}>{loading ? "Loading" : error ? "Unavailable" : architecture?.status ?? "Unavailable"}</span>
          <span>{architecture?.authority.freshness ?? "unknown"} · {architecture?.authority.source_run_id || "no promoted run"}</span>
          <span>Current workspace</span>
        </div>
      </header>

      <nav className="destination-tabs architecture-tabs" aria-label="Architecture views">
        {views.map((item) => <button key={item.id} type="button" aria-current={activePageView === item.id ? "page" : undefined} onClick={() => onViewChange(item.id)}>{item.label}</button>)}
      </nav>

      {error ? <p className="status err" role="status">{error}</p> : null}
      {!loading && !error && architecture?.status === "unavailable" ? <div className="empty-state recovery-empty"><strong>No promoted knowledge is available.</strong><span>Run an analysis to create validator-approved architecture knowledge and C4 views. Source repositories stay read-only.</span>{onOpenRuns ? <button type="button" onClick={onOpenRuns}>Start or inspect analysis</button> : null}</div> : null}
      {architecture?.status === "partial" ? <aside className="status warn" role="status"><strong>Architecture is usable with gaps.</strong> Valid facts remain visible; {architecture.counts.issues} model issue(s) were excluded instead of guessed.</aside> : null}

      {architecture && architecture.status !== "unavailable" && activeView && activePageView === "map" ? (
        <div className="architecture-map-workspace">
          <div className="architecture-map-toolbar">
            <div className="segmented-control" aria-label="C4 level">
			  {(["context", "container", "component"] as ArchitectureLevel[]).map((item) => <button key={item} type="button" aria-pressed={level === item} onClick={() => { advancedLevelRef.current?.removeAttribute("open"); setLevel(item); onEntityChange(undefined); }}>{levelLabel(item)}</button>)}
			  <details ref={advancedLevelRef}><summary>Advanced</summary><button type="button" aria-pressed={level === "code"} disabled={!selectedNode || selectedNode.type !== "service" || !(selectedNode.child_levels ?? []).includes("code")} title={!selectedNode || selectedNode.type !== "service" ? "Select a service to inspect code-level interfaces." : !(selectedNode.child_levels ?? []).includes("code") ? selectedNode.detail_unavailable_reason || "No validated code detail is available for this service." : undefined} onClick={() => { if (!selectedNode) return; advancedLevelRef.current?.removeAttribute("open"); setCodeScopeServiceID(selectedNode.id); setLevel("code"); }}>Code for selected service</button></details>
            </div>
            <div className="architecture-map-filters"><label className="architecture-search"><span className="sr-only">Search architecture</span><input type="search" value={query} placeholder="Search service, owner, type…" onChange={(event) => setQuery(event.target.value)} /></label><label><span className="sr-only">Filter by type</span><select value={typeFilter} onChange={(event) => { setTypeFilter(event.target.value); onEntityChange(undefined); }}><option value="">All types</option>{typeOptions.map((type) => <option key={type} value={type}>{type}</option>)}</select></label><label><span className="sr-only">Filter by repository</span><select value={repositoryFilter} onChange={(event) => { setRepositoryFilter(event.target.value); onEntityChange(undefined); }}><option value="">All repositories</option>{repositoryOptions.map((repository) => <option key={repository} value={repository}>{repository}</option>)}</select></label></div>
            <button type="button" aria-pressed={fullscreen} onClick={() => setFullscreen((value) => !value)}>{fullscreen ? "Exit fullscreen" : "Fullscreen"}</button>
          </div>
          <p className="architecture-breadcrumb">Architecture / {level === "code" && codeScopeService ? `${codeScopeService.name} / ` : ""}{levelLabel(level)}{selectedNode && selectedNode.id !== codeScopeService?.id ? ` / ${selectedNode.name}` : selectedEdge ? ` / ${selectedEdge.name || selectedEdge.type}` : ""}</p>
          <div className="architecture-map-layout">
            <ArchitectureMap view={activeView} level={level} selectedID={selectedEntityID} query={query} typeFilter={typeFilter} repositoryFilter={repositoryFilter} onSelect={onEntityChange} />
            <aside className="architecture-inspector" aria-label="Architecture inspector">
              {selectedNode ? <NodeInspector node={selectedNode} currentLevel={level} onOpenArtifact={onOpenArtifact} onDrillDown={(next) => setLevel(next)} /> : selectedEdge ? <EdgeInspector edge={selectedEdge} nodes={allNodes} onOpenArtifact={onOpenArtifact} /> : <div className="inspector-empty"><strong>Select a node or relationship</strong><p>Inspect ownership, confidence and exact repository evidence. Use Tab and Enter to navigate the map without a mouse.</p></div>}
            </aside>
          </div>
          <div className="architecture-mobile-list" aria-label="Architecture elements">
            {mobileNodes.map((node) => <button type="button" key={node.id} aria-pressed={selectedEntityID === node.id} onClick={() => onEntityChange(node.id)}><strong>{node.name}</strong><span>{node.type} · {Math.round(node.confidence * 100)}%</span></button>)}
            {mobileNodes.length === 0 ? <p className="empty-state">No architecture elements match these filters.</p> : null}
          </div>
        </div>
      ) : null}

      {architecture && architecture.status !== "unavailable" && activePageView === "overview" ? (
        <div className="architecture-overview">
          <section className="architecture-story"><p className="eyebrow">System at a glance</p><h2>{services.length > 0 ? `${services.length} validated service${services.length === 1 ? "" : "s"}` : "System boundary needs evidence"}</h2><p>{services.length > 0 ? `The promoted model connects ${services.length} services with ${datastores.length} datastores and ${external.length} external systems through ${architecture.counts.edges} validated relationships.` : "No service boundary has passed validation yet. ProvenArch does not invent topology from file names."}</p></section>
          <dl className="architecture-metrics"><div><dt>Entities</dt><dd>{architecture.counts.entities}</dd></div><div><dt>Relationships</dt><dd>{architecture.counts.edges}</dd></div><div><dt>Evidence refs</dt><dd>{architecture.counts.evidence}</dd></div><div><dt>Model gaps</dt><dd>{architecture.counts.issues}</dd></div></dl>
          <section><h2>Promoted outputs</h2><div className="architecture-output-links"><button type="button" onClick={() => onOpenArtifact("reports/as-is/overview.md")}>Open Architecture Home</button>{architecture.artifacts.filter((artifact) => artifact.path.includes("reports/diagrams/")).slice(0, 4).map((artifact) => <button type="button" key={artifact.path} onClick={() => onOpenArtifact(artifact.path)}>{artifact.name}</button>)}</div></section>
        </div>
      ) : null}

      {architecture && architecture.status !== "unavailable" && activePageView === "catalog" ? <Catalog nodes={allNodes} selectedID={selectedEntityID} query={query} onQuery={setQuery} onSelect={onEntityChange} onOpenArtifact={onOpenArtifact} /> : null}

      {architecture && architecture.status !== "unavailable" && activePageView === "flows" ? <section className="architecture-flows"><div><h2>Validated flows</h2><p className="hint">Only model edges with valid endpoints and provenance appear here.</p></div>{edges.length === 0 ? <p className="empty-state">No validated relationships are available yet.</p> : <ol>{edges.map((edge) => { const from = allNodes.find((node) => node.id === edge.from); const to = allNodes.find((node) => node.id === edge.to); return <li key={edge.id}><span className="flow-type">{edge.type}</span><strong>{from?.name ?? edge.from}</strong><span aria-hidden="true">→</span><strong>{to?.name ?? edge.to}</strong><button type="button" onClick={() => onOpenArtifact(edge.path)}>Evidence</button></li>; })}</ol>}</section> : null}

      {architecture && architecture.status !== "unavailable" && activePageView === "evidence" ? <section className="architecture-evidence"><div><h2>Evidence and generated outputs</h2><p className="hint">Current promoted authority. Historical taskrun files stay under Runs.</p></div>{architecture.issues.length > 0 ? <ul className="compact-list">{architecture.issues.map((issue) => <li key={`${issue.code}:${issue.path ?? issue.message}`}><strong>{issue.code}</strong>{issue.path ? <code>{issue.path}</code> : null}<span>{issue.message}</span></li>)}</ul> : <p className="status ok">All loaded model files passed the Architecture API checks.</p>}<ul className="knowledge-list">{(knowledge?.artifacts ?? architecture.artifacts).map((artifact) => <li key={artifact.path}><button type="button" onClick={() => onOpenArtifact(artifact.path)}><strong>{artifact.name}</strong><code>{artifact.path}</code><span>{artifact.kind}</span></button></li>)}</ul></section> : null}

      {workspaceHealth && activePageView === "evidence" ? <p className={`status ${workspaceHealth.status === "pass" ? "ok" : workspaceHealth.status === "warn" ? "warn" : "err"}`}>Workspace health: {workspaceHealth.status} · {workspaceHealth.summary.error} errors · {workspaceHealth.summary.warning} warnings</p> : null}
    </section>
  );
}

function NodeInspector({ node, currentLevel, onOpenArtifact, onDrillDown }: { node: ReturnType<typeof uniqueNodes>[number]; currentLevel: ArchitectureLevel; onOpenArtifact: (path: string) => void; onDrillDown: (level: ArchitectureLevel) => void }) {
  const levels = node.child_levels ?? [];
  const order = ["context", "container", "component", "code"] as ArchitectureLevel[];
  const next = order.find((level) => order.indexOf(level) > order.indexOf(currentLevel) && levels.includes(level));
  return <div data-testid="knowledge-entity-detail"><p className="eyebrow">{node.type}</p><h2>{node.name}</h2><code>{node.id}</code><dl className="compact-defs"><div><dt>Confidence</dt><dd>{Math.round(node.confidence * 100)}%</dd></div><div><dt>Authority</dt><dd>{node.provenance_kind}</dd></div><div><dt>Owner</dt><dd>{node.owner_team_id || "Not established"}</dd></div><div><dt>Repositories</dt><dd>{node.repositories?.join(", ") || "Not established"}</dd></div><div><dt>Related review</dt><dd>{(node.related_findings?.length ?? 0)} findings · {(node.related_questions?.length ?? 0)} questions</dd></div></dl>{(node.related_findings?.length ?? 0) + (node.related_questions?.length ?? 0) > 0 ? <ul className="compact-list">{node.related_findings?.map((id) => <li key={id}><strong>Finding</strong><code>{id}</code></li>)}{node.related_questions?.map((id) => <li key={id}><strong>Question</strong><code>{id}</code></li>)}</ul> : null}<h3>Repository evidence</h3>{(node.evidence ?? []).length > 0 ? <ul className="evidence-ref-list">{node.evidence!.map((item, index) => <li key={`${item.repo}:${item.path}:${index}`}><strong>{item.repo}</strong><code>{item.path}</code>{item.lines ? <span>lines {item.lines.start}–{item.lines.end}</span> : null}</li>)}</ul> : <p className="status warn">No direct evidence reference is attached to this node.</p>}<div className="inspector-actions"><button type="button" onClick={() => onOpenArtifact(node.path)}>Open model YAML</button>{next ? <button type="button" onClick={() => onDrillDown(next)}>Drill down to {levelLabel(next)}</button> : <span className="disabled-reason">{node.detail_unavailable_reason || "No validated lower-level detail is available."}</span>}</div></div>;
}

function EdgeInspector({ edge, nodes, onOpenArtifact }: { edge: ArchitectureEdge; nodes: ReturnType<typeof uniqueNodes>; onOpenArtifact: (path: string) => void }) {
  const from = nodes.find((node) => node.id === edge.from);
  const to = nodes.find((node) => node.id === edge.to);
  return <div data-testid="knowledge-edge-detail"><p className="eyebrow">{edge.type}</p><h2>{edge.name || `${from?.name ?? edge.from} → ${to?.name ?? edge.to}`}</h2><code>{edge.id}</code><dl className="compact-defs"><div><dt>From</dt><dd>{from?.name ?? edge.from}</dd></div><div><dt>To</dt><dd>{to?.name ?? edge.to}</dd></div><div><dt>Confidence</dt><dd>{Math.round(edge.confidence * 100)}%</dd></div><div><dt>Authority</dt><dd>{edge.provenance_kind}</dd></div><div><dt>Repositories</dt><dd>{edge.repositories?.join(", ") || "Not established"}</dd></div><div><dt>Related review</dt><dd>{(edge.related_findings?.length ?? 0)} findings · {(edge.related_questions?.length ?? 0)} questions</dd></div></dl><h3>Repository evidence</h3>{(edge.evidence ?? []).length > 0 ? <ul className="evidence-ref-list">{edge.evidence!.map((item, index) => <li key={`${item.repo}:${item.path}:${index}`}><strong>{item.repo}</strong><code>{item.path}</code>{item.lines ? <span>lines {item.lines.start}–{item.lines.end}</span> : null}</li>)}</ul> : <p className="status warn">No direct evidence reference is attached to this relationship.</p>}<div className="inspector-actions"><button type="button" onClick={() => onOpenArtifact(edge.path)}>Open relationship YAML</button></div></div>;
}

function Catalog({ nodes, selectedID, query, onQuery, onSelect, onOpenArtifact }: { nodes: ReturnType<typeof uniqueNodes>; selectedID?: string; query: string; onQuery: (value: string) => void; onSelect: (id?: string) => void; onOpenArtifact: (path: string) => void }) {
  const filtered = nodes.filter((node) => [node.name, node.type, node.id, node.owner_team_id ?? ""].join(" ").toLowerCase().includes(query.trim().toLowerCase()));
  return <section className="architecture-catalog"><div className="catalog-toolbar"><div><h2>Architecture catalog</h2><p className="hint">Search validated entities without using the graph.</p></div><label>Search<input type="search" value={query} onChange={(event) => onQuery(event.target.value)} /></label></div><table className="responsive-card-table"><thead><tr><th>Name</th><th>Type</th><th>Owner</th><th>Confidence</th><th>Source</th></tr></thead><tbody>{filtered.map((node) => <tr key={node.id} aria-selected={selectedID === node.id}><td data-label="Name"><button type="button" onClick={() => onSelect(node.id)}>{node.name}</button></td><td data-label="Type">{node.type}</td><td data-label="Owner">{node.owner_team_id || "—"}</td><td data-label="Confidence">{Math.round(node.confidence * 100)}%</td><td data-label="Source"><button type="button" onClick={() => onOpenArtifact(node.path)}>Open YAML</button></td></tr>)}</tbody></table>{filtered.length === 0 ? <p className="empty-state">No validated entities match this search.</p> : null}</section>;
}

function uniqueNodes(nodes: ArchitectureResponse["views"][ArchitectureLevel]["nodes"]) {
  return Array.from(new Map(nodes.map((node) => [node.id, node])).values()).sort((left, right) => left.name.localeCompare(right.name));
}

function uniqueEdges(edges: ArchitectureResponse["views"][ArchitectureLevel]["edges"]) {
  return Array.from(new Map(edges.map((edge) => [edge.id, edge])).values()).sort((left, right) => left.id.localeCompare(right.id));
}
