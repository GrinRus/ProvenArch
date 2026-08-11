import { useEffect, useMemo, useRef, useState } from "react";

import type { ArchitectureEdge, ArchitectureLevel, ArchitectureResponse, KnowledgeIssue, KnowledgeResponse, WorkspaceHealthResponse } from "../lib/appContracts";
import type { KnowledgeView } from "../lib/appRoutes";
import { architectureFromKnowledge, loadArtifactText, saveEditableArtifact } from "../lib/workspaceApi";
import { ArchitectureMap, levelLabel } from "./ArchitectureMap";
import { EvidenceViewer } from "./EvidenceViewer";

const views: Array<{ id: KnowledgeView; label: string }> = [
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
  const ownerOptions = useMemo(() => Array.from(new Set(activeView?.nodes.map((node) => node.owner_team_id).filter((owner): owner is string => Boolean(owner)) ?? [])).sort(), [activeView]);
  const tagOptions = useMemo(() => Array.from(new Set(activeView?.nodes.flatMap((node) => node.tags ?? []) ?? [])).sort(), [activeView]);
  const mobileNodes = (activeView?.nodes ?? []).filter((node) => (!typeFilter || node.type === typeFilter) && (!repositoryFilter || node.repositories?.includes(repositoryFilter)) && (!ownerFilter || node.owner_team_id === ownerFilter) && (!tagFilter || node.tags?.includes(tagFilter)) && [node.name, node.id, node.type, node.owner_team_id ?? "", ...(node.tags ?? [])].join(" ").toLowerCase().includes(query.trim().toLowerCase()));
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

      {architecture && architecture.status !== "unavailable" && activePageView === "documents" ? <DocumentsWorkspace architecture={architecture} selectedArtifactPath={selectedArtifactPath} onDocumentChange={onDocumentChange} onOpenArtifact={onOpenArtifact} mode="documents" /> : null}
      {architecture && architecture.status !== "unavailable" && activePageView === "diagrams" ? <DocumentsWorkspace architecture={architecture} selectedArtifactPath={selectedArtifactPath} onDocumentChange={onDocumentChange} onOpenArtifact={onOpenArtifact} mode="diagrams" /> : null}
      {architecture && architecture.status !== "unavailable" && activePageView === "findings" ? <FindingsView architecture={architecture} onOpenArtifact={onOpenArtifact} /> : null}

      {architecture && architecture.status !== "unavailable" && activeView && (activePageView === "map" || activePageView === "model") ? (
        <div className="architecture-map-workspace">
          <div className="architecture-map-toolbar">
            <div className="segmented-control" aria-label="C4 level">
			  {(["context", "container", "component"] as ArchitectureLevel[]).map((item) => <button key={item} type="button" aria-pressed={level === item} onClick={() => { advancedLevelRef.current?.removeAttribute("open"); setLevel(item); onEntityChange(undefined); }}>{levelLabel(item)}</button>)}
			  <details ref={advancedLevelRef}><summary>Advanced</summary><button type="button" aria-pressed={level === "code"} disabled={!selectedNode || selectedNode.type !== "service" || !(selectedNode.child_levels ?? []).includes("code")} title={!selectedNode || selectedNode.type !== "service" ? "Select a service to inspect code-level interfaces." : !(selectedNode.child_levels ?? []).includes("code") ? selectedNode.detail_unavailable_reason || "No validated code detail is available for this service." : undefined} onClick={() => { if (!selectedNode) return; advancedLevelRef.current?.removeAttribute("open"); setCodeScopeServiceID(selectedNode.id); setLevel("code"); }}>Code for selected service</button></details>
            </div>
            <div className="architecture-map-filters"><label className="architecture-search"><span className="sr-only">Search architecture</span><input type="search" value={query} placeholder="Search service, owner, type…" onChange={(event) => setQuery(event.target.value)} /></label><label><span className="sr-only">Filter by type</span><select value={typeFilter} onChange={(event) => { setTypeFilter(event.target.value); onEntityChange(undefined); }}><option value="">All types</option>{typeOptions.map((type) => <option key={type} value={type}>{type}</option>)}</select></label><label><span className="sr-only">Filter by repository</span><select value={repositoryFilter} onChange={(event) => { setRepositoryFilter(event.target.value); onEntityChange(undefined); }}><option value="">All repositories</option>{repositoryOptions.map((repository) => <option key={repository} value={repository}>{repository}</option>)}</select></label><label><span className="sr-only">Filter by owner</span><select value={ownerFilter} onChange={(event) => { setOwnerFilter(event.target.value); onEntityChange(undefined); }}><option value="">All owners</option>{ownerOptions.map((owner) => <option key={owner} value={owner}>{owner}</option>)}</select></label><label><span className="sr-only">Filter by domain or tag</span><select value={tagFilter} onChange={(event) => { setTagFilter(event.target.value); onEntityChange(undefined); }}><option value="">All domains/tags</option>{tagOptions.map((tag) => <option key={tag} value={tag}>{tag}</option>)}</select></label></div>
            <button type="button" aria-pressed={fullscreen} onClick={() => setFullscreen((value) => !value)}>{fullscreen ? "Exit fullscreen" : "Fullscreen"}</button>
          </div>
          <p className="architecture-breadcrumb">Architecture / {level === "code" && codeScopeService ? `${codeScopeService.name} / ` : ""}{levelLabel(level)}{selectedNode && selectedNode.id !== codeScopeService?.id ? ` / ${selectedNode.name}` : selectedEdge ? ` / ${selectedEdge.name || selectedEdge.type}` : ""}</p>
          <div className="architecture-map-layout">
            <ArchitectureMap view={activeView} level={level} selectedID={selectedEntityID} query={query} typeFilter={typeFilter} repositoryFilter={repositoryFilter} ownerFilter={ownerFilter} tagFilter={tagFilter} onSelect={onEntityChange} />
            <aside className="architecture-inspector" aria-label="Architecture inspector">
              {selectedNode ? <NodeInspector node={selectedNode} currentLevel={level} issues={architecture.issues} onOpenArtifact={onOpenArtifact} onDrillDown={(next) => setLevel(next)} /> : selectedEdge ? <EdgeInspector edge={selectedEdge} nodes={allNodes} issues={architecture.issues} onOpenArtifact={onOpenArtifact} /> : <div className="inspector-empty"><strong>Select a node or relationship</strong><p>Inspect ownership, confidence and exact repository evidence. Use Tab and Enter to navigate the map without a mouse.</p></div>}
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

function DocumentsWorkspace({ architecture, selectedArtifactPath, onDocumentChange, onOpenArtifact, mode }: { architecture: ArchitectureResponse; selectedArtifactPath?: string; onDocumentChange?: (path?: string) => void; onOpenArtifact: (path: string) => void; mode: "documents" | "diagrams" }) {
  const documents = architecture.artifacts.filter((artifact) => mode === "diagrams" ? artifact.path.endsWith(".mmd") : artifact.path.endsWith(".md")).sort((left, right) => left.path.localeCompare(right.path));
  const selectedPath = selectedArtifactPath && documents.some((artifact) => artifact.path === selectedArtifactPath) ? selectedArtifactPath : documents[0]?.path;
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
      {documents.length === 0 ? <p className="empty-state">No promoted {mode} are available yet.</p> : <ul>{documents.map((artifact) => <li key={artifact.path}><button type="button" aria-current={selectedPath === artifact.path ? "page" : undefined} onClick={() => onDocumentChange?.(artifact.path)}><strong>{artifact.name}</strong><code>{artifact.path}</code></button></li>)}</ul>}
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

function MermaidEvidenceContext({ architecture, onOpenArtifact }: { architecture: ArchitectureResponse; onOpenArtifact: (path: string) => void }) {
  const relations = uniqueEdges(Object.values(architecture.views).flatMap((view) => view.edges));
  return <section className="mermaid-evidence-context" data-testid="mermaid-evidence-context"><p className="eyebrow">Evidence Studio</p><h2>Validated relations</h2><p className="hint">This list is the accessible relation authority. Mermaid layout and arrows are visual aids only and never create semantic relations.</p><p className="status info mermaid-source-note">Promoted Mermaid source is read-only. Use Raw in the Source tab for bounded inspection; source diff remains the only supported comparison until deterministic visual diff exists.</p>{relations.length === 0 ? <p className="empty-state">No validated relations are available for this diagram.</p> : <ul className="mermaid-relation-list">{relations.map((relation) => <li key={relation.id}><span className="flow-type">{relation.type}</span><code className="model-logical-id">{relation.from} → {relation.to}</code><button type="button" onClick={() => onOpenArtifact(relation.path)}>Open relation evidence</button></li>)}</ul>}</section>;
}

function isEditableMarkdownPath(path?: string): boolean {
  if (!path || !path.endsWith(".md")) return false;
  return path === "charter" || path.startsWith("charter/") || path === "skills" || path.startsWith("skills/");
}

function FindingsView({ architecture, onOpenArtifact }: { architecture: ArchitectureResponse; onOpenArtifact: (path: string) => void }) {
  const findings = architecture.review?.findings ?? [];
  const questions = architecture.review?.questions ?? [];
  const gaps = architecture.coverage?.missing ?? [];
  return <section className="architecture-findings" data-testid="architecture-findings"><header><div><p className="eyebrow">Review queue</p><h2>Findings and open questions</h2><p className="hint">Only selected-run semantic findings and explicit coverage gaps are shown.</p></div><button type="button" onClick={() => onOpenArtifact("reports/findings/findings.md")}>Open findings document</button></header><div className="architecture-findings-grid"><article><h3>Findings <span>{findings.length}</span></h3>{findings.length === 0 ? <p className="empty-state">No findings in the promoted snapshot.</p> : <ul className="compact-list">{findings.map((finding) => <li key={finding.id}><strong>{finding.title}</strong><span>{finding.severity}</span><code>{finding.id}</code></li>)}</ul>}</article><article><h3>Questions <span>{questions.length}</span></h3>{questions.length === 0 ? <p className="empty-state">No open questions in the promoted snapshot.</p> : <ul className="compact-list">{questions.map((question) => <li key={question.id}><strong>{question.text}</strong><span>{question.priority || "open"}</span></li>)}</ul>}</article><article><h3>Coverage gaps <span>{gaps.length}</span></h3>{gaps.length === 0 ? <p className="empty-state">No named coverage gaps.</p> : <ul className="compact-list">{gaps.map((gap) => <li key={gap}><strong>{gap}</strong><span>coverage</span></li>)}</ul>}</article></div></section>;
}

function NodeInspector({ node, currentLevel, issues, onOpenArtifact, onDrillDown }: { node: ReturnType<typeof uniqueNodes>[number]; currentLevel: ArchitectureLevel; issues: KnowledgeIssue[]; onOpenArtifact: (path: string) => void; onDrillDown: (level: ArchitectureLevel) => void }) {
  const levels = node.child_levels ?? [];
  const order = ["context", "container", "component", "code"] as ArchitectureLevel[];
  const next = order.find((level) => order.indexOf(level) > order.indexOf(currentLevel) && levels.includes(level));
  return <div data-testid="knowledge-entity-detail"><p className="eyebrow">{node.type}</p><h2>{node.name}</h2><code className="model-logical-id">{node.id}</code><dl className="compact-defs"><div><dt>Confidence</dt><dd>{Math.round(node.confidence * 100)}%</dd></div><div><dt>Authority</dt><dd>{node.provenance_kind}</dd></div><div><dt>Owner</dt><dd>{node.owner_team_id || "Not established"}</dd></div><div><dt>Repositories</dt><dd>{node.repositories?.join(", ") || "Not established"}</dd></div><div><dt>Related review</dt><dd>{(node.related_findings?.length ?? 0)} findings · {(node.related_questions?.length ?? 0)} questions</dd></div></dl>{(node.related_findings?.length ?? 0) + (node.related_questions?.length ?? 0) > 0 ? <ul className="compact-list">{node.related_findings?.map((id) => <li key={id}><strong>Finding</strong><code>{id}</code></li>)}{node.related_questions?.map((id) => <li key={id}><strong>Question</strong><code>{id}</code></li>)}</ul> : null}<h3>Repository evidence</h3>{(node.evidence ?? []).length > 0 ? <ul className="evidence-ref-list">{node.evidence!.map((item, index) => <li key={`${item.repo}:${item.path}:${index}`}><strong>{item.repo}</strong><code>{item.path}</code>{item.lines ? <span>lines {item.lines.start}–{item.lines.end}</span> : null}</li>)}</ul> : <p className="status warn">No direct evidence reference is attached to this node.</p>}<StructuredArtifactInspector path={node.path} kind="entity" issues={issues} onOpenArtifact={onOpenArtifact} /><div className="inspector-actions"><button type="button" onClick={() => onOpenArtifact(node.path)}>Open model YAML</button>{next ? <button type="button" onClick={() => onDrillDown(next)}>Drill down to {levelLabel(next)}</button> : <span className="disabled-reason">{node.detail_unavailable_reason || "No validated lower-level detail is available."}</span>}</div></div>;
}

function EdgeInspector({ edge, nodes, issues, onOpenArtifact }: { edge: ArchitectureEdge; nodes: ReturnType<typeof uniqueNodes>; issues: KnowledgeIssue[]; onOpenArtifact: (path: string) => void }) {
  const from = nodes.find((node) => node.id === edge.from);
  const to = nodes.find((node) => node.id === edge.to);
  return <div data-testid="knowledge-edge-detail"><p className="eyebrow">{edge.type}</p><h2>{edge.name || `${from?.name ?? edge.from} → ${to?.name ?? edge.to}`}</h2><code className="model-logical-id">{edge.id}</code><dl className="compact-defs"><div><dt>From</dt><dd>{from?.name ?? edge.from}</dd></div><div><dt>To</dt><dd>{to?.name ?? edge.to}</dd></div><div><dt>Confidence</dt><dd>{Math.round(edge.confidence * 100)}%</dd></div><div><dt>Authority</dt><dd>{edge.provenance_kind}</dd></div><div><dt>Repositories</dt><dd>{edge.repositories?.join(", ") || "Not established"}</dd></div><div><dt>Related review</dt><dd>{(edge.related_findings?.length ?? 0)} findings · {(edge.related_questions?.length ?? 0)} questions</dd></div></dl><h3>Repository evidence</h3>{(edge.evidence ?? []).length > 0 ? <ul className="evidence-ref-list">{edge.evidence!.map((item, index) => <li key={`${item.repo}:${item.path}:${index}`}><strong>{item.repo}</strong><code>{item.path}</code>{item.lines ? <span>lines {item.lines.start}–{item.lines.end}</span> : null}</li>)}</ul> : <p className="status warn">No direct evidence reference is attached to this relationship.</p>}<StructuredArtifactInspector path={edge.path} kind="edge" issues={issues} onOpenArtifact={onOpenArtifact} /><div className="inspector-actions"><button type="button" onClick={() => onOpenArtifact(edge.path)}>Open relationship YAML</button></div></div>;
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
