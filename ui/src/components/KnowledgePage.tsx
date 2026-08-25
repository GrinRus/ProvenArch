import { lazy, Suspense, useEffect, useMemo, useRef, useState } from "react";

import type { ArchitectureEdge, ArchitectureFinding, ArchitectureLevel, ArchitectureResponse, KnowledgeArtifact, KnowledgeIssue, KnowledgeResponse, WorkspaceHealthResponse } from "../lib/appContracts";
import type { KnowledgeView } from "../lib/appRoutes";
import { architectureFromKnowledge, loadArtifactText, loadRepositoryEvidenceAPI, saveEditableArtifact, type RepositoryEvidence } from "../lib/workspaceApi";
import { levelLabel } from "../lib/architectureLabels";
import { EvidenceViewer } from "./EvidenceViewer";

const ArchitectureMap = lazy(() => import("./ArchitectureMap").then((module) => ({ default: module.ArchitectureMap })));

const views: Array<{ id: KnowledgeView; label: string }> = [
  { id: "map", label: "Map" },
  { id: "documents", label: "Documents" },
  { id: "diagrams", label: "Diagrams" },
  { id: "model", label: "Model" },
  { id: "findings", label: "Findings" },
];

export function KnowledgePage({
  architecture: providedArchitecture,
  knowledge,
  loading,
  error,
  view,
  selectedEntityID,
  selectedArtifactPath,
  workspaceHealth,
  onViewChange,
  onEntityChange,
  onDocumentChange,
  onOpenArtifact,
  onOpenRuns,
  taskId,
  onOpenTask,
}: {
  architecture?: ArchitectureResponse | null;
  knowledge: KnowledgeResponse | null;
  loading: boolean;
  error: string;
  view: KnowledgeView;
  selectedEntityID?: string;
  selectedArtifactPath?: string;
  workspaceHealth?: WorkspaceHealthResponse | null;
  onViewChange: (view: KnowledgeView) => void;
  onEntityChange: (id?: string) => void;
  onDocumentChange?: (path?: string) => void;
  onOpenArtifact: (path: string) => void;
  onOpenRuns?: () => void;
  taskId?: string;
  onOpenTask?: (taskId: string) => void;
}) {
	const architecture = providedArchitecture ?? (knowledge ? architectureFromKnowledge(knowledge) : null);
	const activePageView: KnowledgeView = view === "atlas" ? "map" : view === "entities" ? "catalog" : view === "artifacts" ? "evidence" : view;
  const [query, setQuery] = useState("");
  const [level, setLevel] = useState<ArchitectureLevel>("context");
  const [typeFilter, setTypeFilter] = useState("");
  const [repositoryFilter, setRepositoryFilter] = useState("");
  const [ownerFilter, setOwnerFilter] = useState("");
  const [tagFilter, setTagFilter] = useState("");
  const [fullscreen, setFullscreen] = useState(false);
  const [filtersOpen, setFiltersOpen] = useState(() => window.matchMedia?.("(min-width: 681px)").matches ?? true);
	const [codeScopeServiceID, setCodeScopeServiceID] = useState<string>();
	const [repositoryEvidence, setRepositoryEvidence] = useState<RepositoryEvidence | null>(null);
	const [repositoryEvidenceStatus, setRepositoryEvidenceStatus] = useState<"idle" | "loading" | "unavailable">("idle");
	const advancedLevelRef = useRef<HTMLDetailsElement>(null);
	const openRepositoryEvidence = async (repo: string, path: string) => {
	  setRepositoryEvidenceStatus("loading");
	  const next = await loadRepositoryEvidenceAPI(repo, path);
	  if (!next) {
	    setRepositoryEvidence(null);
	    setRepositoryEvidenceStatus("unavailable");
	    return;
	  }
	  setRepositoryEvidence(next);
	  setRepositoryEvidenceStatus("idle");
	};
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
  const ownerOptions = useMemo(() => Array.from(new Set(activeView?.nodes.map((node) => node.owner_team_id).filter((owner): owner is string => Boolean(owner)) ?? [])).sort(), [activeView]);
  const tagOptions = useMemo(() => Array.from(new Set(activeView?.nodes.flatMap((node) => node.tags ?? []) ?? [])).sort(), [activeView]);
  const mobileNodes = (activeView?.nodes ?? []).filter((node) => (!typeFilter || node.type === typeFilter) && (!repositoryFilter || node.repositories?.includes(repositoryFilter)) && (!ownerFilter || node.owner_team_id === ownerFilter) && (!tagFilter || node.tags?.includes(tagFilter)) && [node.name, node.id, node.type, node.owner_team_id ?? "", ...(node.tags ?? [])].join(" ").toLowerCase().includes(query.trim().toLowerCase()));
  const edges = architecture?.views.container.edges ?? [];
  const services = allNodes.filter((node) => node.type === "service");
  const external = allNodes.filter((node) => node.type === "external.system");
  const datastores = allNodes.filter((node) => node.type === "datastore");
  const activeFilterCount = [query, typeFilter, repositoryFilter, ownerFilter, tagFilter].filter((value) => value.trim().length > 0).length;

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
          <span>{architectureAuthorityLabel(architecture)}</span>
          <span>Current workspace · read-only</span>
        </div>
      </header>

      {taskId ? <aside className="task-architecture-context" data-testid="task-architecture-context" aria-label="Task Architecture context">
        <div>
          <p className="eyebrow">Task context</p>
          <strong>Architecture for the selected Task</strong>
          <p><code>{taskId}</code> · current promoted workspace authority</p>
          <span className="hint">This surface is read-only and does not infer state from the latest run.</span>
        </div>
        {onOpenTask ? <button type="button" className="ui-button tone-neutral" onClick={() => onOpenTask(taskId)}>Back to Task</button> : null}
      </aside> : null}

      <nav className="destination-tabs architecture-tabs" aria-label="Architecture views">
        {views.map((item) => <button key={item.id} type="button" aria-current={activePageView === item.id ? "page" : undefined} onClick={() => onViewChange(item.id)}>{item.label}</button>)}
      </nav>

      {error ? <p className="status err" role="status">{error}</p> : null}
      {!loading && !error && architecture?.status === "unavailable" ? <div className="empty-state recovery-empty"><strong>No promoted knowledge is available.</strong><span>Run an analysis to create validator-approved architecture knowledge and C4 views. Source repositories stay read-only.</span>{onOpenRuns ? <button type="button" onClick={onOpenRuns}>Start or inspect analysis</button> : null}</div> : null}
      {architecture?.status === "partial" ? <aside className="status warn" role="status"><strong>Architecture is usable with gaps.</strong> Valid facts remain visible; {architecture.counts.issues} model issue(s) were excluded instead of guessed.</aside> : null}

	  {repositoryEvidenceStatus === "loading" ? <p className="status info" role="status">Loading repository evidence…</p> : null}
	  {repositoryEvidenceStatus === "unavailable" ? <p className="status warn" role="status">Repository evidence is unavailable in the configured checkout.</p> : null}
	  {repositoryEvidence ? <section className="repository-evidence-viewer panel" data-testid="repository-evidence-viewer"><header><div><p className="eyebrow">Repository source · read-only</p><h2>{repositoryEvidence.repo}</h2><code>{repositoryEvidence.path}</code></div><button type="button" onClick={() => setRepositoryEvidence(null)}>Close source</button></header><EvidenceViewer path={`${repositoryEvidence.repo}:${repositoryEvidence.path}`} content={repositoryEvidence.content} sourceMode="repository" provenance="live" /></section> : null}

      {architecture && architecture.status !== "unavailable" && activePageView === "documents" ? <DocumentsWorkspace architecture={architecture} selectedArtifactPath={selectedArtifactPath} onDocumentChange={onDocumentChange} onOpenArtifact={onOpenArtifact} mode="documents" /> : null}
      {architecture && architecture.status !== "unavailable" && activePageView === "diagrams" ? <DocumentsWorkspace architecture={architecture} selectedArtifactPath={selectedArtifactPath} onDocumentChange={onDocumentChange} onOpenArtifact={onOpenArtifact} mode="diagrams" /> : null}
      {architecture && architecture.status !== "unavailable" && activePageView === "findings" ? <FindingsView architecture={architecture} onOpenArtifact={onOpenArtifact} onOpenRepositoryEvidence={openRepositoryEvidence} /> : null}

      {architecture && architecture.status !== "unavailable" && activePageView === "model" ? <ModelWorkbench architecture={architecture} selectedEntityID={selectedEntityID} onEntityChange={onEntityChange} onOpenRepositoryEvidence={openRepositoryEvidence} /> : null}

      {architecture && architecture.status !== "unavailable" && activeView && activePageView === "map" ? (
        <div className="architecture-map-workspace">
          <div className="architecture-map-toolbar">
            <div className="segmented-control" aria-label="C4 level">
			  {(["context", "container", "component"] as ArchitectureLevel[]).map((item) => <button key={item} type="button" aria-pressed={level === item} onClick={() => { advancedLevelRef.current?.removeAttribute("open"); setLevel(item); onEntityChange(undefined); }}>{levelLabel(item)}</button>)}
			  <details ref={advancedLevelRef}><summary>Advanced</summary><button type="button" aria-pressed={level === "code"} disabled={!selectedNode || selectedNode.type !== "service" || !(selectedNode.child_levels ?? []).includes("code")} title={!selectedNode || selectedNode.type !== "service" ? "Select a service to inspect code-level interfaces." : !(selectedNode.child_levels ?? []).includes("code") ? selectedNode.detail_unavailable_reason || "No validated code detail is available for this service." : undefined} onClick={() => { if (!selectedNode) return; advancedLevelRef.current?.removeAttribute("open"); setCodeScopeServiceID(selectedNode.id); setLevel("code"); }}>Code for selected service</button></details>
            </div>
            <details className="architecture-filter-disclosure" open={filtersOpen} onToggle={(event) => setFiltersOpen(event.currentTarget.open)}>
              <summary>Filters <span>{activeFilterCount > 0 ? `${activeFilterCount} active` : "All filters"}</span></summary>
              <div className="architecture-map-filters"><label className="architecture-search"><span className="sr-only">Search architecture</span><input id="architecture-search" name="architecture-search" type="search" value={query} placeholder="Search service, owner, type…" onChange={(event) => setQuery(event.target.value)} /></label><label><span className="sr-only">Filter by type</span><select id="architecture-type-filter" name="architecture-type-filter" value={typeFilter} onChange={(event) => { setTypeFilter(event.target.value); onEntityChange(undefined); }}><option value="">All types</option>{typeOptions.map((type) => <option key={type} value={type}>{type}</option>)}</select></label><label><span className="sr-only">Filter by repository</span><select id="architecture-repository-filter" name="architecture-repository-filter" value={repositoryFilter} onChange={(event) => { setRepositoryFilter(event.target.value); onEntityChange(undefined); }}><option value="">All repositories</option>{repositoryOptions.map((repository) => <option key={repository} value={repository}>{repository}</option>)}</select></label><label><span className="sr-only">Filter by owner</span><select id="architecture-owner-filter" name="architecture-owner-filter" value={ownerFilter} onChange={(event) => { setOwnerFilter(event.target.value); onEntityChange(undefined); }}><option value="">All owners</option>{ownerOptions.map((owner) => <option key={owner} value={owner}>{owner}</option>)}</select></label><label><span className="sr-only">Filter by domain or tag</span><select id="architecture-tag-filter" name="architecture-tag-filter" value={tagFilter} onChange={(event) => { setTagFilter(event.target.value); onEntityChange(undefined); }}><option value="">All domains/tags</option>{tagOptions.map((tag) => <option key={tag} value={tag}>{tag}</option>)}</select></label></div>
            </details>
            <button type="button" aria-pressed={fullscreen} onClick={() => setFullscreen((value) => !value)}>{fullscreen ? "Exit fullscreen" : "Fullscreen"}</button>
          </div>
          <p className="architecture-breadcrumb">Architecture / {level === "code" && codeScopeService ? `${codeScopeService.name} / ` : ""}{levelLabel(level)}{selectedNode && selectedNode.id !== codeScopeService?.id ? ` / ${selectedNode.name}` : selectedEdge ? ` / ${selectedEdge.name || selectedEdge.type}` : ""}</p>
          <div className="architecture-map-layout">
            <aside className="architecture-element-list" aria-label="Architecture element list"><div className="panel-subheader"><div><p className="eyebrow">Elements</p><h2>{mobileNodes.length} visible</h2></div></div>{mobileNodes.length === 0 ? <p className="empty-state">No elements match these filters.</p> : <ul>{mobileNodes.map((node) => <li key={node.id}><button type="button" aria-pressed={selectedEntityID === node.id} onClick={() => onEntityChange(node.id)}><strong>{node.name}</strong><span>{node.type} · {Math.round(node.confidence * 100)}%</span></button></li>)}</ul>}</aside>
            <Suspense fallback={<div className="architecture-canvas architecture-canvas-loading" role="status">Loading architecture map…</div>}><ArchitectureMap view={activeView} level={level} selectedID={selectedEntityID} query={query} typeFilter={typeFilter} repositoryFilter={repositoryFilter} ownerFilter={ownerFilter} tagFilter={tagFilter} onSelect={onEntityChange} /></Suspense>
            <aside className="architecture-inspector" aria-label="Architecture inspector">
              {selectedNode ? <NodeInspector node={selectedNode} currentLevel={level} issues={architecture.issues} onOpenArtifact={onOpenArtifact} onOpenRepositoryEvidence={openRepositoryEvidence} onDrillDown={(next) => setLevel(next)} /> : selectedEdge ? <EdgeInspector edge={selectedEdge} nodes={allNodes} issues={architecture.issues} onOpenArtifact={onOpenArtifact} onOpenRepositoryEvidence={openRepositoryEvidence} /> : <div className="inspector-empty"><strong>Select an element to inspect</strong><p>Choose an architecture element below, or use the map in fullscreen mode to inspect ownership, confidence and exact repository evidence.</p></div>}
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

function ModelWorkbench({ architecture, selectedEntityID, onEntityChange, onOpenRepositoryEvidence }: { architecture: ArchitectureResponse; selectedEntityID?: string; onEntityChange: (id?: string) => void; onOpenRepositoryEvidence: (repo: string, path: string) => void }) {
  const entities = uniqueNodes(Object.values(architecture.views).flatMap((view) => view.nodes));
  const selected = entities.find((node) => node.id === selectedEntityID) ?? entities[0];
  const relationships = selected ? uniqueEdges(Object.values(architecture.views).flatMap((view) => view.edges)).filter((edge) => edge.from === selected.id || edge.to === selected.id) : [];
  return <section className="architecture-model-workbench" data-testid="architecture-model-workbench">
    <header className="architecture-model-header"><div><p className="eyebrow">Model &amp; schema</p><h2>Validated domain model</h2><p className="hint">Inspect entity contracts, ownership and relationships independently from the visual C4 map.</p></div><span className="status info">Promoted current workspace</span></header>
    <div className="architecture-model-grid">
      <aside className="architecture-model-entity-list" aria-label="Model entities"><div className="panel-subheader"><div><h3>Entities</h3><p className="hint">{entities.length} validated</p></div></div>{entities.length === 0 ? <p className="empty-state">No validated model entities are available.</p> : <ul>{entities.map((node) => <li key={node.id}><button type="button" aria-pressed={selected?.id === node.id} onClick={() => onEntityChange(node.id)}><strong>{node.name}</strong><span>{node.type} · {Math.round(node.confidence * 100)}%</span></button></li>)}</ul>}</aside>
      <section className="architecture-model-schema" aria-label="Selected model schema">{selected ? <><div className="panel-subheader"><div><p className="eyebrow">Selected entity</p><h3>{selected.name}</h3><p className="hint"><code>{selected.id}</code></p></div><span className="status ok">{Math.round(selected.confidence * 100)}% confidence</span></div><dl className="architecture-schema-fields"><div><dt>Type</dt><dd>{selected.type}</dd></div><div><dt>Owner</dt><dd>{selected.owner_team_id || "Unassigned"}</dd></div><div><dt>Repositories</dt><dd>{selected.repositories?.join(", ") || "Not linked"}</dd></div><div><dt>Tags</dt><dd>{selected.tags?.join(", ") || "None"}</dd></div><div><dt>Available levels</dt><dd>{(selected.available_levels ?? []).map(levelLabel).join(", ") || "Context only"}</dd></div></dl><section className="architecture-model-relations"><h4>Relationships <span>{relationships.length}</span></h4>{relationships.length === 0 ? <p className="empty-state">No validated relationships are linked to this entity.</p> : <ul>{relationships.map((edge) => <li key={edge.id}><span className="flow-type">{edge.type}</span><strong>{edge.from === selected.id ? edge.to : edge.from}</strong><span>{edge.name || "Validated relation"}</span></li>)}</ul>}</section><section className="architecture-model-evidence"><h4>Evidence</h4>{selected.evidence?.length ? <ul>{selected.evidence.map((item, index) => <li key={`${item.repo}:${item.path}:${index}`}><div><strong>{item.repo}</strong><code>{item.path}{item.lines ? `:${item.lines.start}-${item.lines.end}` : ""}</code></div><button type="button" onClick={() => onOpenRepositoryEvidence(item.repo, item.path)}>Open source</button></li>)}</ul> : <p className="empty-state">No bounded evidence is attached to this model entity.</p>}</section></> : <p className="empty-state">Select an entity to inspect its schema.</p>}</section>
      <aside className="architecture-model-authority"><p className="eyebrow">Model contract</p><h3>Evidence-backed only</h3><p className="hint">This workbench exposes validated fields and relations. It does not infer missing schema from source names.</p><dl className="compact-defs"><div><dt>Entities</dt><dd>{architecture.counts.entities}</dd></div><div><dt>Relations</dt><dd>{architecture.counts.edges}</dd></div><div><dt>Evidence refs</dt><dd>{architecture.counts.evidence}</dd></div><div><dt>Model issues</dt><dd>{architecture.counts.issues}</dd></div></dl></aside>
    </div>
  </section>;
}

function DocumentsWorkspace({ architecture, selectedArtifactPath, onDocumentChange, onOpenArtifact, mode }: { architecture: ArchitectureResponse; selectedArtifactPath?: string; onDocumentChange?: (path?: string) => void; onOpenArtifact: (path: string) => void; mode: "documents" | "diagrams" }) {
  const documents = architecture.artifacts.filter((artifact) => mode === "diagrams" ? artifact.path.endsWith(".mmd") : artifact.path.endsWith(".md")).sort((left, right) => left.path.localeCompare(right.path));
  const preferredPath = mode === "documents" ? architectureHomePath(architecture, documents) : undefined;
  const selectedPath = selectedArtifactPath && documents.some((artifact) => artifact.path === selectedArtifactPath) ? selectedArtifactPath : preferredPath ?? documents[0]?.path;
  const [content, setContent] = useState("");
  const [draft, setDraft] = useState("");
  const [status, setStatus] = useState<"idle" | "loading" | "loaded" | "error">("idle");
  const [editMode, setEditMode] = useState(false);
  const [saveStatus, setSaveStatus] = useState("");
  const editable = mode === "documents" && isEditableMarkdownPath(selectedPath);
  useEffect(() => {
    if (selectedPath && selectedPath !== selectedArtifactPath) onDocumentChange?.(selectedPath);
  }, [onDocumentChange, selectedArtifactPath, selectedPath]);
  useEffect(() => {
    if (!selectedPath) {
      setContent("");
      setDraft("");
      setStatus("idle");
      setEditMode(false);
      return;
    }
    let active = true;
    setStatus("loading");
    void loadArtifactText(selectedPath).then((value) => {
      if (!active) return;
      setContent(value ?? "");
      setDraft(value ?? "");
      setStatus(value === null ? "error" : "loaded");
      setEditMode(false);
      setSaveStatus("");
    });
    return () => { active = false; };
  }, [selectedPath]);
  return <div className="architecture-documents" data-testid={`architecture-${mode}`}>
    <aside className="architecture-document-tree" aria-label={`${mode === "documents" ? "Document" : "Diagram"} tree`}>
      <div><p className="eyebrow">{mode === "documents" ? "Documents" : "Diagrams"}</p><h2>{documents.length} available</h2><p className="hint">Selected promoted workspace authority.</p></div>
      {documents.length === 0 ? <p className="empty-state">No promoted {mode} are available yet.</p> : <div className="semantic-document-groups">{documentGroups(documents, architecture, mode).map((group) => <section key={group.label} className="semantic-document-group"><h3>{group.label}</h3><ul>{group.artifacts.map((artifact) => <li key={artifact.path}><button type="button" aria-current={selectedPath === artifact.path ? "page" : undefined} onClick={() => onDocumentChange?.(artifact.path)}><strong>{documentLabel(artifact, architecture)}</strong><code>{artifact.path}</code></button></li>)}</ul></section>)}</div>}
    </aside>
    <section className="architecture-document-reader" aria-label="Architecture document reader">
      {selectedPath && status === "loaded" ? <>
        {editable ? <div className="markdown-editor-toolbar" aria-label="Markdown editor controls">
          <span className="hint">Editable workspace Markdown · lossless text is preserved until save.</span>
          <div>
            <button type="button" className="ui-button tone-neutral" onClick={() => { if (editMode) setDraft(content); setEditMode((value) => !value); setSaveStatus(""); }}>{editMode ? "Cancel" : "Edit Markdown"}</button>
            {editMode ? <button type="button" className="ui-button tone-primary" disabled={draft === content || saveStatus === "Saving…"} onClick={() => void (async () => { setSaveStatus("Saving…"); try { await saveEditableArtifact(selectedPath, draft); setContent(draft); setEditMode(false); setSaveStatus("Saved to the editable workspace surface."); } catch (error) { setSaveStatus(error instanceof Error ? error.message : "Markdown save failed."); } })()}>Save</button> : null}
          </div>
        </div> : <p className="status info markdown-read-only-note">Promoted Architecture documents are read-only evidence. Edit is available only for editable workspace Markdown under <code>charter/</code> or <code>skills/</code>.</p>}
        {editMode ? <textarea className="markdown-editor" data-testid="markdown-editor" value={draft} onChange={(event) => setDraft(event.target.value)} aria-label={`Edit ${selectedPath}`} rows={24} /> : <EvidenceViewer key={selectedPath} path={selectedPath} content={content} sourceMode="promoted_current" onOpenArtifact={onOpenArtifact} />}
        {saveStatus ? <p className={saveStatus.startsWith("Saved") ? "status ok" : saveStatus === "Saving…" ? "status info" : "status err"} role="status">{saveStatus}</p> : null}
        {mode === "diagrams" ? <MermaidEvidenceContext architecture={architecture} onOpenArtifact={onOpenArtifact} /> : null}
      </> : status === "loading" ? <p className="status info">Loading document…</p> : status === "error" ? <p className="status err">The selected promoted document is unavailable.</p> : <p className="empty-state">Select a document to inspect its content.</p>}
    </section>
    <aside className="architecture-document-context" aria-label="Document context">
      <h2>Document context</h2>
      <div className="document-context-group"><span>Trust</span><strong className="status ok">validated</strong><p>Promoted workspace authority · current snapshot.</p></div>
      <div className="document-context-group"><span>Source</span><code>{selectedPath || "No document selected"}</code></div>
      <div className="document-context-group"><span>Related</span><p>{mode === "documents" ? "System Context diagram" : "Architecture Home"}</p><p>Evidence and findings</p></div>
      <button type="button" className="ui-button tone-primary" onClick={() => selectedPath && onOpenArtifact(selectedPath)} disabled={!selectedPath}>Open source artifact</button>
    </aside>
  </div>;
}

function architectureHomePath(architecture: ArchitectureResponse, documents: KnowledgeArtifact[]): string | undefined {
  const exportedHome = architecture.exports?.home_path;
  if (exportedHome && documents.some((artifact) => artifact.path === exportedHome)) return exportedHome;
  const namedHome = documents.find((artifact) => /architecture\s+home/i.test(artifact.name));
  if (namedHome) return namedHome.path;
  return documents.find((artifact) => artifact.path === "reports/as-is/overview.md")?.path;
}

function architectureAuthorityLabel(architecture: ArchitectureResponse | null): string {
  if (!architecture) return "Loading promoted current workspace";
  if (architecture.authority.freshness === "stale") return "Promoted current workspace · stale freshness";
  if (architecture.authority.freshness === "recent") return "Promoted current workspace · recently validated";
  return architecture.authority.source_run_id ? `Promoted current workspace · ${architecture.authority.source_run_id}` : "Promoted current workspace · validated evidence";
}

function documentLabel(artifact: KnowledgeArtifact, architecture: ArchitectureResponse): string {
  return architectureHomePath(architecture, architecture.artifacts.filter((item) => item.path.endsWith(".md"))) === artifact.path ? "Architecture Home" : artifact.name;
}

function documentGroups(documents: KnowledgeArtifact[], architecture: ArchitectureResponse, mode: "documents" | "diagrams"): Array<{ label: string; artifacts: KnowledgeArtifact[] }> {
  const labels = mode === "diagrams"
    ? ["Architecture diagrams"]
    : ["Architecture Home", "Services and domains", "Findings and questions", "Proposals", "Other documents"];
  const groups = new Map(labels.map((label) => [label, [] as KnowledgeArtifact[]]));
  for (const artifact of documents) {
    const home = architectureHomePath(architecture, documents);
    const label = mode === "diagrams" ? "Architecture diagrams"
      : artifact.path === home ? "Architecture Home"
      : artifact.path.startsWith("reports/findings/") || artifact.path.startsWith("reports/coverage/") ? "Findings and questions"
      : artifact.path.startsWith("proposals/") ? "Proposals"
      : artifact.path.startsWith("model/") || artifact.path.startsWith("reports/as-is/") ? "Services and domains"
      : "Other documents";
    groups.get(label)?.push(artifact);
  }
  return [...groups.entries()].filter(([, artifacts]) => artifacts.length > 0).map(([label, artifacts]) => ({ label, artifacts: artifacts.sort((left, right) => left.path.localeCompare(right.path)) }));
}

function MermaidEvidenceContext({ architecture, onOpenArtifact }: { architecture: ArchitectureResponse; onOpenArtifact: (path: string) => void }) {
  const relations = uniqueEdges(Object.values(architecture.views).flatMap((view) => view.edges));
  return <section className="mermaid-evidence-context" data-testid="mermaid-evidence-context"><p className="eyebrow">Evidence Studio</p><h2>Validated relations</h2><p className="hint">This list is the accessible relation authority. Mermaid layout and arrows are visual aids only and never create semantic relations.</p><p className="status info mermaid-source-note">Promoted Mermaid source is read-only. Use Raw in the Source tab for bounded inspection; source diff remains the only supported comparison until deterministic visual diff exists.</p>{relations.length === 0 ? <p className="empty-state">No validated relations are available for this diagram.</p> : <ul className="mermaid-relation-list">{relations.map((relation) => <li key={relation.id}><span className="flow-type">{relation.type}</span><code className="model-logical-id">{relation.from} → {relation.to}</code><button type="button" onClick={() => onOpenArtifact(relation.path)}>Open relation evidence</button></li>)}</ul>}</section>;
}

function isEditableMarkdownPath(path?: string): boolean {
  if (!path || !path.endsWith(".md")) return false;
  return path === "charter" || path.startsWith("charter/") || path === "skills" || path.startsWith("skills/");
}

function FindingsView({ architecture, onOpenArtifact, onOpenRepositoryEvidence }: { architecture: ArchitectureResponse; onOpenArtifact: (path: string) => void; onOpenRepositoryEvidence: (repo: string, path: string) => void }) {
  const findings = architecture.review?.findings ?? [];
  const questions = architecture.review?.questions ?? [];
  const gaps = architecture.coverage?.missing ?? [];
  const [query, setQuery] = useState("");
  const [severity, setSeverity] = useState("");
  const normalizedQuery = query.trim().toLowerCase();
  const visibleFindings = findings.filter((finding) => (!severity || finding.severity === severity) && [finding.id, finding.title, finding.description ?? ""].join(" ").toLowerCase().includes(normalizedQuery));
  const visibleQuestions = questions.filter((question) => [question.id, question.text, question.priority ?? ""].join(" ").toLowerCase().includes(normalizedQuery));
  const visibleGaps = gaps.filter((gap) => gap.toLowerCase().includes(normalizedQuery));
  const [selected, setSelected] = useState<{ kind: "finding" | "question" | "gap"; id: string }>(() => defaultFindingSelection(findings, questions, gaps));
  const selectedFinding = selected.kind === "finding" ? findings.find((finding) => finding.id === selected.id) : undefined;
  const selectedQuestion = selected.kind === "question" ? questions.find((question) => question.id === selected.id) : undefined;
  const selectedGap = selected.kind === "gap" ? gaps.find((gap) => gap === selected.id) : undefined;
  useEffect(() => {
    const visibleSelection = (selected.kind === "finding" && visibleFindings.some((item) => item.id === selected.id)) || (selected.kind === "question" && visibleQuestions.some((item) => item.id === selected.id)) || (selected.kind === "gap" && visibleGaps.includes(selected.id));
    if (visibleSelection) return;
    const next = defaultFindingSelection(visibleFindings, visibleQuestions, visibleGaps);
    setSelected(next);
  }, [selected, visibleFindings, visibleQuestions, visibleGaps]);
  const relatedNodes = (ids: string[] | undefined) => (ids ?? []).map((id) => allNodesForArchitecture(architecture).find((node) => node.id === id)).filter((node): node is ReturnType<typeof uniqueNodes>[number] => Boolean(node));
  return <section className="architecture-findings" data-testid="architecture-findings">
    <header><div><p className="eyebrow">Review queue</p><h2>Findings, questions and gaps</h2><p className="hint">Start with the highest-priority unresolved item. Every conclusion stays linked to promoted evidence.</p></div><button type="button" onClick={() => onOpenArtifact("reports/findings/findings.md")}>Open findings document</button></header>
    <div className="findings-toolbar"><label htmlFor="findings-search">Search<input id="findings-search" name="findings-search" type="search" value={query} onChange={(event) => setQuery(event.target.value)} /></label><label htmlFor="findings-severity">Severity<select id="findings-severity" name="findings-severity" value={severity} onChange={(event) => setSeverity(event.target.value)}><option value="">All severities</option>{Array.from(new Set(findings.map((finding) => finding.severity))).sort().map((item) => <option key={item} value={item}>{item}</option>)}</select></label><span className="status info">Unresolved · human decision required</span></div>
    <div className="architecture-findings-grid">
      <article><h3>Findings <span>{visibleFindings.length}/{findings.length}</span></h3>{visibleFindings.length === 0 ? <p className="empty-state">No findings match this filter.</p> : <ul className="compact-list">{visibleFindings.map((finding) => <li key={finding.id}><button type="button" title={`Finding ${finding.id}`} aria-pressed={selected.kind === "finding" && selected.id === finding.id} onClick={() => setSelected({ kind: "finding", id: finding.id })}><strong>{finding.title}</strong><span>{finding.severity}{finding.rule_id ? ` · ${finding.rule_id}` : ""}</span></button></li>)}</ul>}</article>
      <article><h3>Questions <span>{visibleQuestions.length}/{questions.length}</span></h3>{visibleQuestions.length === 0 ? <p className="empty-state">No open questions match this filter.</p> : <ul className="compact-list">{visibleQuestions.map((question) => <li key={question.id}><button type="button" title={`Question ${question.id}`} aria-pressed={selected.kind === "question" && selected.id === question.id} onClick={() => setSelected({ kind: "question", id: question.id })}><strong>{question.text}</strong><span>{question.priority || "open"}</span></button></li>)}</ul>}</article>
      <article><h3>Coverage gaps <span>{visibleGaps.length}/{gaps.length}</span></h3>{visibleGaps.length === 0 ? <p className="empty-state">No named coverage gaps match this filter.</p> : <ul className="compact-list">{visibleGaps.map((gap) => <li key={gap}><button type="button" aria-pressed={selected.kind === "gap" && selected.id === gap} onClick={() => setSelected({ kind: "gap", id: gap })}><strong>{gap}</strong><span>coverage</span></button></li>)}</ul>}</article>
    </div>
    <section className="finding-detail" aria-label="Selected review item">
      {selectedFinding ? <FindingDetail finding={selectedFinding} relatedNodes={relatedNodes(selectedFinding.related_ids)} onOpenArtifact={onOpenArtifact} onOpenRepositoryEvidence={onOpenRepositoryEvidence} /> : selectedQuestion ? <ReviewQuestionDetail question={selectedQuestion} relatedNodes={relatedNodes(selectedQuestion.related_ids)} onOpenArtifact={onOpenArtifact} onOpenRepositoryEvidence={onOpenRepositoryEvidence} /> : selectedGap ? <><p className="eyebrow">Coverage gap</p><h3>{selectedGap}</h3><div className="finding-detail-section"><h4>Why it matters</h4><p>This gap is unresolved and limits the confidence of the promoted architecture.</p></div><div className="finding-detail-section"><h4>Suggested architecture</h4><p className="status info">No persisted proposal is attached yet. Use the evidence chain to draft one in Proposals.</p></div></> : <p className="empty-state">No review item is available for this filter.</p>}
    </section>
    <aside className="proposal-decision-boundary" data-testid="proposal-decision-boundary"><strong>Proposal decision boundary</strong><span>Findings are evidence-backed review items. Approved status and proposal mutation are unavailable until a persisted human decision exists.</span></aside>
  </section>;
}

function defaultFindingSelection(findings: ArchitectureFinding[], questions: Array<{ id: string; text: string; priority?: string }>, gaps: string[]): { kind: "finding" | "question" | "gap"; id: string } {
  const severityRank: Record<string, number> = { critical: 0, high: 1, medium: 2, low: 3, info: 4 };
  const finding = [...findings].sort((left, right) => (severityRank[left.severity.toLowerCase()] ?? 5) - (severityRank[right.severity.toLowerCase()] ?? 5))[0];
  if (finding) return { kind: "finding", id: finding.id };
  const question = questions[0];
  if (question) return { kind: "question", id: question.id };
  return { kind: "gap", id: gaps[0] ?? "" };
}

function FindingDetail({ finding, relatedNodes, onOpenArtifact, onOpenRepositoryEvidence }: { finding: ArchitectureFinding; relatedNodes: ReturnType<typeof uniqueNodes>; onOpenArtifact: (path: string) => void; onOpenRepositoryEvidence: (repo: string, path: string) => void }) {
  const evidence = findingEvidence(finding, relatedNodes);
  return <><p className="eyebrow">Finding detail</p><h3>{finding.title}</h3><div className="finding-detail-meta"><span className={`status severity-${finding.severity.toLowerCase()}`}>{finding.severity}</span><span>{finding.provenance?.confidence !== undefined ? `${Math.round(finding.provenance.confidence * 100)}% confidence` : "Confidence unavailable"}</span><span>{evidence.length} evidence refs</span></div><div className="finding-detail-section"><h4>Observed</h4><p>{finding.description || "No narrative observation was recorded in the promoted snapshot."}</p></div><div className="finding-detail-section"><h4>Why it matters</h4><p>{finding.rule_id ? `This finding is raised by ${finding.rule_id} and remains unresolved.` : "The promoted architecture cannot treat this review item as resolved without a human decision."}</p></div><div className="finding-detail-section"><h4>Suggested architecture</h4><p className="status info">No persisted proposal is attached yet. Draft the change only after reviewing the cited evidence.</p></div><details className="finding-identity"><summary>Technical identity</summary><dl className="compact-defs"><div><dt>Finding ID</dt><dd><code>{finding.id}</code></dd></div><div><dt>Authority</dt><dd>{finding.provenance?.kind || "promoted semantic snapshot"}</dd></div></dl></details><FindingEvidenceChain evidence={evidence} onOpenArtifact={onOpenArtifact} onOpenRepositoryEvidence={onOpenRepositoryEvidence} /></>;
}

function ReviewQuestionDetail({ question, relatedNodes, onOpenArtifact, onOpenRepositoryEvidence }: { question: { id: string; text: string; priority?: string; related_ids?: string[] }; relatedNodes: ReturnType<typeof uniqueNodes>; onOpenArtifact: (path: string) => void; onOpenRepositoryEvidence: (repo: string, path: string) => void }) {
  return <><p className="eyebrow">Open question</p><h3>{question.text}</h3><div className="finding-detail-meta"><span className="status warn">{question.priority || "open"}</span><span>{relatedNodes.length} linked evidence refs</span></div><div className="finding-detail-section"><h4>Observed</h4><p>The promoted snapshot leaves this question unresolved.</p></div><div className="finding-detail-section"><h4>Suggested architecture</h4><p className="status info">No proposal is inferred. Resolve the question against the linked evidence before drafting a change.</p></div><details className="finding-identity"><summary>Technical identity</summary><code>{question.id}</code></details><FindingEvidenceChain evidence={findingEvidence(undefined, relatedNodes)} onOpenArtifact={onOpenArtifact} onOpenRepositoryEvidence={onOpenRepositoryEvidence} /></>;
}

function allNodesForArchitecture(architecture: ArchitectureResponse) {
  return uniqueNodes(Object.values(architecture.views).flatMap((view) => view.nodes));
}

function findingEvidence(finding: ArchitectureFinding | undefined, nodes: ReturnType<typeof uniqueNodes>): Array<{ id: string; label: string; repo?: string; path: string; lines?: { start: number; end: number }; excerpt?: string }> {
  const refs: Array<{ id: string; label: string; repo?: string; path: string; lines?: { start: number; end: number }; excerpt?: string }> = [];
  for (const [index, item] of (finding?.provenance?.evidence ?? []).entries()) refs.push({ id: `finding:${index}:${item.repo}:${item.path}`, label: "Finding citation", repo: item.repo, path: item.path, lines: item.lines, excerpt: item.excerpt });
  for (const node of nodes) for (const [index, item] of (node.evidence ?? []).entries()) refs.push({ id: `node:${node.id}:${index}:${item.path}`, label: node.name, repo: item.repo, path: item.path, lines: item.lines });
  return Array.from(new Map(refs.map((item) => [`${item.repo ?? ""}:${item.path}:${item.lines?.start ?? ""}`, item])).values());
}

function FindingEvidenceChain({ evidence, onOpenArtifact, onOpenRepositoryEvidence }: { evidence: Array<{ id: string; label: string; repo?: string; path: string; lines?: { start: number; end: number }; excerpt?: string }>; onOpenArtifact: (path: string) => void; onOpenRepositoryEvidence: (repo: string, path: string) => void }) {
  return <section className="finding-evidence-chain" aria-label="Evidence chain"><p className="eyebrow">Evidence chain</p>{evidence.length === 0 ? <p className="hint">No bounded citation is attached to this review item.</p> : <ol>{evidence.map((item, index) => <li key={item.id}><div><span className="finding-evidence-order">{index + 1}</span><strong>{item.label}</strong><span>{item.repo || "repository unavailable"}{item.lines ? ` · lines ${item.lines.start}–${item.lines.end}` : ""}</span><code>{item.path}</code>{item.excerpt ? <q>{item.excerpt}</q> : null}</div>{item.repo ? <button type="button" onClick={() => onOpenRepositoryEvidence(item.repo!, item.path)}>Open source</button> : <button type="button" onClick={() => onOpenArtifact(item.path)}>Open evidence</button>}</li>)}</ol>}</section>;
}

function NodeInspector({ node, currentLevel, issues, onOpenArtifact, onOpenRepositoryEvidence, onDrillDown }: { node: ReturnType<typeof uniqueNodes>[number]; currentLevel: ArchitectureLevel; issues: KnowledgeIssue[]; onOpenArtifact: (path: string) => void; onOpenRepositoryEvidence: (repo: string, path: string) => void; onDrillDown: (level: ArchitectureLevel) => void }) {
  const levels = node.child_levels ?? [];
  const order = ["context", "container", "component", "code"] as ArchitectureLevel[];
  const next = order.find((level) => order.indexOf(level) > order.indexOf(currentLevel) && levels.includes(level));
  return <div data-testid="knowledge-entity-detail"><p className="eyebrow">{node.type}</p><h2>{node.name}</h2><code className="model-logical-id">{node.id}</code><dl className="compact-defs"><div><dt>Confidence</dt><dd>{Math.round(node.confidence * 100)}%</dd></div><div><dt>Authority</dt><dd>{node.provenance_kind}</dd></div><div><dt>Owner</dt><dd>{node.owner_team_id || "Not established"}</dd></div><div><dt>Repositories</dt><dd>{node.repositories?.join(", ") || "Not established"}</dd></div><div><dt>Related review</dt><dd>{(node.related_findings?.length ?? 0)} findings · {(node.related_questions?.length ?? 0)} questions</dd></div></dl>{(node.related_findings?.length ?? 0) + (node.related_questions?.length ?? 0) > 0 ? <ul className="compact-list">{node.related_findings?.map((id) => <li key={id}><strong>Finding</strong><code>{id}</code></li>)}{node.related_questions?.map((id) => <li key={id}><strong>Question</strong><code>{id}</code></li>)}</ul> : null}<h3>Repository evidence</h3>{(node.evidence ?? []).length > 0 ? <ul className="evidence-ref-list">{node.evidence!.map((item, index) => <li key={`${item.repo}:${item.path}:${index}`}><strong>{item.repo}</strong><code>{item.path}</code>{item.lines ? <span>lines {item.lines.start}–{item.lines.end}</span> : null}<button type="button" onClick={() => onOpenRepositoryEvidence(item.repo, item.path)}>Open source</button></li>)}</ul> : <p className="status warn">No direct evidence reference is attached to this node.</p>}<StructuredArtifactInspector path={node.path} kind="entity" issues={issues} onOpenArtifact={onOpenArtifact} /><div className="inspector-actions"><button type="button" onClick={() => onOpenArtifact(node.path)}>Open model YAML</button>{next ? <button type="button" onClick={() => onDrillDown(next)}>Drill down to {levelLabel(next)}</button> : <span className="disabled-reason">{node.detail_unavailable_reason || "No validated lower-level detail is available."}</span>}</div></div>;
}

function EdgeInspector({ edge, nodes, issues, onOpenArtifact, onOpenRepositoryEvidence }: { edge: ArchitectureEdge; nodes: ReturnType<typeof uniqueNodes>; issues: KnowledgeIssue[]; onOpenArtifact: (path: string) => void; onOpenRepositoryEvidence: (repo: string, path: string) => void }) {
  const from = nodes.find((node) => node.id === edge.from);
  const to = nodes.find((node) => node.id === edge.to);
  return <div data-testid="knowledge-edge-detail"><p className="eyebrow">{edge.type}</p><h2>{edge.name || `${from?.name ?? edge.from} → ${to?.name ?? edge.to}`}</h2><code className="model-logical-id">{edge.id}</code><dl className="compact-defs"><div><dt>From</dt><dd>{from?.name ?? edge.from}</dd></div><div><dt>To</dt><dd>{to?.name ?? edge.to}</dd></div><div><dt>Confidence</dt><dd>{Math.round(edge.confidence * 100)}%</dd></div><div><dt>Authority</dt><dd>{edge.provenance_kind}</dd></div><div><dt>Repositories</dt><dd>{edge.repositories?.join(", ") || "Not established"}</dd></div><div><dt>Related review</dt><dd>{(edge.related_findings?.length ?? 0)} findings · {(edge.related_questions?.length ?? 0)} questions</dd></div></dl><h3>Repository evidence</h3>{(edge.evidence ?? []).length > 0 ? <ul className="evidence-ref-list">{edge.evidence!.map((item, index) => <li key={`${item.repo}:${item.path}:${index}`}><strong>{item.repo}</strong><code>{item.path}</code>{item.lines ? <span>lines {item.lines.start}–{item.lines.end}</span> : null}<button type="button" onClick={() => onOpenRepositoryEvidence(item.repo, item.path)}>Open source</button></li>)}</ul> : <p className="status warn">No direct evidence reference is attached to this relationship.</p>}<StructuredArtifactInspector path={edge.path} kind="edge" issues={issues} onOpenArtifact={onOpenArtifact} /><div className="inspector-actions"><button type="button" onClick={() => onOpenArtifact(edge.path)}>Open relationship YAML</button></div></div>;
}

function StructuredArtifactInspector({ path, kind, issues, onOpenArtifact }: { path: string; kind: "entity" | "edge"; issues: KnowledgeIssue[]; onOpenArtifact: (path: string) => void }) {
  const [sourceOpen, setSourceOpen] = useState(false);
  const [source, setSource] = useState<string>();
  const [sourceStatus, setSourceStatus] = useState<"idle" | "loading" | "loaded" | "unavailable">("idle");
  const artifactIssues = issues.filter((issue) => issue.path === path || issue.path?.startsWith(`${path}#`));
  useEffect(() => {
    if (!sourceOpen) return;
    let active = true;
    setSourceStatus("loading");
    void loadArtifactText(path).then((value) => {
      if (!active) return;
      if (value === null) setSourceStatus("unavailable");
      else { setSource(value); setSourceStatus("loaded"); }
    });
    return () => { active = false; };
  }, [path, sourceOpen]);
  const schema = kind === "entity" ? "architecture.entity" : "architecture.edge";
  return <section className="structured-model-inspector" data-testid="structured-model-inspector"><header><div><p className="eyebrow">Structured inspector</p><strong>{schema} v1</strong></div><span className={`status ${artifactIssues.length > 0 ? "warn" : "ok"}`}>{artifactIssues.length > 0 ? `${artifactIssues.length} validation issue${artifactIssues.length === 1 ? "" : "s"}` : "Validated structure"}</span></header><p className="hint">Canonical identity: <code className="model-logical-id">{path}</code></p>{artifactIssues.length > 0 ? <ul className="compact-list">{artifactIssues.map((issue) => <li key={`${issue.code}:${issue.path ?? issue.message}`}><strong>{issue.code}</strong><span>{issue.message}</span></li>)}</ul> : <p className="status ok">Schema and semantic checks passed for the promoted snapshot.</p>}<p className="markdown-read-only-note">Structured editing is unavailable until comments, unknown keys, ordering and multiline scalars have a proven lossless round trip. This inspector is read-only.</p><div className="inspector-actions"><button type="button" onClick={() => setSourceOpen((value) => !value)}>{sourceOpen ? "Hide source" : "Source (Advanced)"}</button><button type="button" onClick={() => onOpenArtifact(path)}>Open source artifact</button></div>{sourceOpen ? <div className="structured-source"><p className="eyebrow">Exact source bytes, line numbered for diagnostics</p>{sourceStatus === "loading" ? <p className="status info">Loading source…</p> : sourceStatus === "unavailable" ? <p className="status warn">Source is unavailable; the last valid promoted structure remains visible.</p> : <pre data-testid="structured-source">{(source ?? "").split("\n").map((line, index) => `${String(index + 1).padStart(4, "0")} | ${line}`).join("\n")}</pre>}</div> : null}</section>;
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
