import { useState } from "react";

import type { KnowledgeResponse, WorkspaceHealthResponse } from "../lib/appContracts";
import type { KnowledgeView } from "../lib/appRoutes";
import { buildKnowledgeViewModel } from "../features/workbench/viewModels";

export function KnowledgePage({
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
  const [query, setQuery] = useState("");
  const model = buildKnowledgeViewModel(knowledge, loading, error, query, selectedEntityID);
  const { entities, edges, artifacts, filteredEntities, selectedEntity } = model;

  return (
    <section className="panel stage-panel knowledge-page" data-testid="knowledge-panel">
      <div className="stage-header">
        <div><h1>Knowledge</h1><p className="hint">Validated entities, relationships and promoted artifacts from the current workspace only.</p></div>
        <span className={`status ${knowledge?.status === "available" ? "ok" : knowledge?.status === "partial" ? "warn" : "err"}`}>{loading ? "loading" : error ? "unavailable" : knowledge?.status ?? "unavailable"}</span>
      </div>
      <p className="source-identity"><strong>Current workspace</strong> · promoted, read-only knowledge</p>
      {workspaceHealth ? (
        <aside className={`status ${workspaceHealth.status === "pass" ? "ok" : workspaceHealth.status === "warn" ? "warn" : "err"}`} data-testid="knowledge-workspace-health">
          Workspace Health: <strong>{workspaceHealth.status}</strong> · {workspaceHealth.summary.error} errors, {workspaceHealth.summary.warning} warnings, {workspaceHealth.summary.info} info. Advisory only.
        </aside>
      ) : null}
      <nav className="destination-tabs" aria-label="Knowledge views">
        {(["overview", "atlas", "entities", "artifacts"] as KnowledgeView[]).map((item) => (
          <button key={item} type="button" aria-current={view === item ? "page" : undefined} onClick={() => onViewChange(item)}>{label(item)}</button>
        ))}
      </nav>
      {error ? <p className="status err" role="status">{error}</p> : null}
      {!loading && !error && knowledge?.status === "unavailable" ? <div className="empty-state recovery-empty"><strong>No promoted knowledge is available.</strong><span>Run an analysis to create validated architecture knowledge for this workspace.</span>{onOpenRuns ? <button type="button" onClick={onOpenRuns}>Open Runs</button> : null}</div> : null}
      {knowledge?.status === "partial" ? (
        <aside className="status warn" role="status"><strong>Partial knowledge.</strong> Valid data remains visible; {knowledge.issues.length} issue(s) need attention.</aside>
      ) : null}

      {knowledge && view === "overview" ? (
        <div data-testid="knowledge-overview">
          <div className="home-summary-grid">
            <article><span className="metric-label">Entities</span><strong>{entities.length}</strong></article>
            <article><span className="metric-label">Validated edges</span><strong>{edges.length}</strong></article>
            <article><span className="metric-label">Artifacts</span><strong>{artifacts.length}</strong></article>
            <article><span className="metric-label">Issues</span><strong>{knowledge.issues.length}</strong></article>
          </div>
          {knowledge.issues.length > 0 ? <IssueList knowledge={knowledge} /> : <p>No knowledge validation issues.</p>}
        </div>
      ) : null}

      {knowledge && view === "atlas" ? (
        <div data-testid="knowledge-atlas">
          <h2>Validated relationship atlas</h2>
          <p className="hint">Topology is derived only from parsed entity and edge contents. File names are never interpreted as relationships.</p>
          {edges.length === 0 ? <div className="empty-state recovery-empty"><strong>No validated relationships are available.</strong><span>Relationships appear only when entity and edge artifacts pass validation; ProvenArch does not guess missing topology.</span>{onOpenRuns ? <button type="button" onClick={onOpenRuns}>Run or inspect analysis</button> : null}</div> : (
            <table className="responsive-card-table"><caption className="sr-only">Validated architecture relationships</caption><thead><tr><th>From</th><th>Relationship</th><th>To</th><th>Source</th></tr></thead><tbody>
              {edges.map((edge) => <tr key={edge.id}><td data-label="From">{entityName(entities, edge.from)}</td><td data-label="Relationship">{edge.type}</td><td data-label="To">{entityName(entities, edge.to)}</td><td data-label="Source"><button type="button" onClick={() => onOpenArtifact(edge.path)}>{edge.path}</button></td></tr>)}
            </tbody></table>
          )}
          {knowledge.status === "partial" ? <p className="status warn">Atlas is incomplete because malformed or missing-reference files were excluded.</p> : null}
        </div>
      ) : null}

      {knowledge && view === "entities" ? (
        <div data-testid="knowledge-entities">
          <label>Search entities<input type="search" value={query} onChange={(event) => setQuery(event.target.value)} /></label>
          <table className="responsive-card-table"><caption className="sr-only">Searchable validated entities</caption><thead><tr><th>Name</th><th>Type</th><th>ID</th><th>Source</th></tr></thead><tbody>
            {filteredEntities.map((entity) => <tr key={entity.id} aria-selected={selectedEntityID === entity.id}><td data-label="Name"><button type="button" onClick={() => onEntityChange(entity.id)}>{entity.name}</button></td><td data-label="Type">{entity.type}</td><td data-label="ID"><code>{entity.id}</code></td><td data-label="Source"><button type="button" onClick={() => onOpenArtifact(entity.path)}>{entity.path}</button></td></tr>)}
          </tbody></table>
          {filteredEntities.length === 0 ? <p className="empty-state">No validated entities match this search.</p> : null}
          {selectedEntity ? <aside className="inspector-card" data-testid="knowledge-entity-detail"><h2>{selectedEntity.name}</h2><p>{selectedEntity.type} · <code>{selectedEntity.id}</code></p><button type="button" onClick={() => onOpenArtifact(selectedEntity.path)}>Open source artifact</button></aside> : null}
        </div>
      ) : null}

      {knowledge && view === "artifacts" ? (
        <div data-testid="knowledge-artifacts"><h2>Promoted artifact inventory</h2>{artifacts.length === 0 ? <p className="empty-state">No readable promoted artifacts.</p> : <ul className="knowledge-list">
          {artifacts.map((artifact) => <li key={artifact.path}><button type="button" onClick={() => onOpenArtifact(artifact.path)}><strong>{artifact.name}</strong><code>{artifact.path}</code><span>{artifact.kind}</span></button></li>)}
        </ul>}</div>
      ) : null}
    </section>
  );
}

function IssueList({ knowledge }: { knowledge: KnowledgeResponse }) {
  return <ul className="compact-list" data-testid="knowledge-issues">{knowledge.issues.map((issue) => <li key={`${issue.code}:${issue.path ?? issue.message}`}><strong>{issue.code}</strong>{issue.path ? <code>{issue.path}</code> : null}<span>{issue.message}</span></li>)}</ul>;
}

function entityName(entities: KnowledgeResponse["entities"], id: string): string {
  const entity = entities.find((candidate) => candidate.id === id);
  return entity ? `${entity.name} (${id})` : id;
}

function label(view: KnowledgeView): string {
  return view.charAt(0).toUpperCase() + view.slice(1);
}
