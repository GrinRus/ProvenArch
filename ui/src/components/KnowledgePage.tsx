import { useMemo, useState } from "react";

import type { KnowledgeResponse } from "../lib/appContracts";
import type { KnowledgeView } from "../lib/appRoutes";

export function KnowledgePage({
  knowledge,
  loading,
  error,
  view,
  selectedEntityID,
  onViewChange,
  onEntityChange,
  onOpenArtifact,
}: {
  knowledge: KnowledgeResponse | null;
  loading: boolean;
  error: string;
  view: KnowledgeView;
  selectedEntityID?: string;
  onViewChange: (view: KnowledgeView) => void;
  onEntityChange: (id?: string) => void;
  onOpenArtifact: (path: string) => void;
}) {
  const [query, setQuery] = useState("");
  const entities = knowledge?.entities ?? [];
  const edges = knowledge?.edges ?? [];
  const artifacts = knowledge?.artifacts ?? [];
  const selectedEntity = entities.find((entity) => entity.id === selectedEntityID);
  const filteredEntities = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return entities;
    return entities.filter((entity) => `${entity.id} ${entity.name} ${entity.type} ${(entity.tags ?? []).join(" ")}`.toLowerCase().includes(normalized));
  }, [entities, query]);

  return (
    <section className="panel stage-panel knowledge-page" data-testid="knowledge-panel">
      <div className="stage-header">
        <div><h1>Knowledge</h1><p className="hint">Validated entities, relationships and promoted artifacts from the current workspace only.</p></div>
        <span className={`status ${knowledge?.status === "available" ? "ok" : knowledge?.status === "partial" ? "warn" : "err"}`}>{loading ? "loading" : error ? "unavailable" : knowledge?.status ?? "unavailable"}</span>
      </div>
      <p className="source-identity"><strong>Current workspace</strong> · promoted, read-only knowledge</p>
      <nav className="destination-tabs" aria-label="Knowledge views">
        {(["overview", "atlas", "entities", "artifacts"] as KnowledgeView[]).map((item) => (
          <button key={item} type="button" aria-current={view === item ? "page" : undefined} onClick={() => onViewChange(item)}>{label(item)}</button>
        ))}
      </nav>
      {error ? <p className="status err" role="status">{error}</p> : null}
      {!loading && !error && knowledge?.status === "unavailable" ? <p className="empty-state">No promoted knowledge is available in the current workspace.</p> : null}
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
          {edges.length === 0 ? <p className="empty-state">No validated relationships are available.</p> : (
            <table><caption className="sr-only">Validated architecture relationships</caption><thead><tr><th>From</th><th>Relationship</th><th>To</th><th>Source</th></tr></thead><tbody>
              {edges.map((edge) => <tr key={edge.id}><td>{entityName(entities, edge.from)}</td><td>{edge.type}</td><td>{entityName(entities, edge.to)}</td><td><button type="button" onClick={() => onOpenArtifact(edge.path)}>{edge.path}</button></td></tr>)}
            </tbody></table>
          )}
          {knowledge.status === "partial" ? <p className="status warn">Atlas is incomplete because malformed or missing-reference files were excluded.</p> : null}
        </div>
      ) : null}

      {knowledge && view === "entities" ? (
        <div data-testid="knowledge-entities">
          <label>Search entities<input type="search" value={query} onChange={(event) => setQuery(event.target.value)} /></label>
          <table><caption className="sr-only">Searchable validated entities</caption><thead><tr><th>Name</th><th>Type</th><th>ID</th><th>Source</th></tr></thead><tbody>
            {filteredEntities.map((entity) => <tr key={entity.id} aria-selected={selectedEntityID === entity.id}><td><button type="button" onClick={() => onEntityChange(entity.id)}>{entity.name}</button></td><td>{entity.type}</td><td><code>{entity.id}</code></td><td><button type="button" onClick={() => onOpenArtifact(entity.path)}>{entity.path}</button></td></tr>)}
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
