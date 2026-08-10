import { ArtifactPathButton, StatusBadge } from "./ConsolePrimitives";
import type { Artifact } from "../lib/appContracts";

export type ReviewDomainMapNode = {
  id: string;
  label: string;
  typeLabel: string;
  group: string;
  kind: "domain" | "entity";
  artifact: Artifact;
};

export type ReviewDomainMapEdge = {
  id: string;
  type: string;
  from: string;
  to: string;
  artifact: Artifact;
};

export type ReviewDomainMapGroup = {
  key: string;
  label: string;
  nodes: ReviewDomainMapNode[];
};

export type ReviewDomainMapModel = {
  nodes: ReviewDomainMapNode[];
  groups: ReviewDomainMapGroup[];
  edges: ReviewDomainMapEdge[];
  domainOutputs: Artifact[];
  proposalArtifacts: Artifact[];
  navigationArtifacts: Artifact[];
  entityCount: number;
  repoCount: number;
  ownershipStatus: string;
  coverageStatus: string;
  crossRepoStatus: string;
  blockers: string[];
};

export function ReviewDomainMap({ domainMap, onOpenArtifact }: { domainMap: ReviewDomainMapModel; onOpenArtifact: (path: string) => void }) {
  const hasMapData = domainMap.nodes.length > 0 || domainMap.edges.length > 0 || domainMap.domainOutputs.length > 0;
  return (
    <div className="review-domain-map" data-testid="review-domain-map">
      <section className="domain-map-canvas" data-testid="review-domain-map-canvas">
        <div className="section-heading-row">
          <div><h2>Domain/service map</h2><p className="hint">Derived from selected-run model artifacts and domain agent outputs.</p></div>
          <StatusBadge tone={hasMapData ? "ok" : "info"}>{hasMapData ? "derived model" : "partial"}</StatusBadge>
        </div>
        <div className="domain-map-summary-grid">
          <div className="metric-tile"><span className="metric-label">Entities</span><strong>{domainMap.entityCount}</strong></div>
          <div className="metric-tile"><span className="metric-label">Edges</span><strong>{domainMap.edges.length}</strong></div>
          <div className="metric-tile"><span className="metric-label">Domain outputs</span><strong>{domainMap.domainOutputs.length}</strong></div>
          <div className="metric-tile"><span className="metric-label">Repo scopes</span><strong>{domainMap.repoCount > 0 ? domainMap.repoCount : "partial"}</strong></div>
        </div>
        {!hasMapData ? <div className="domain-map-empty" data-testid="review-domain-map-empty"><strong>No derived model artifacts yet.</strong><span>Run Analysis and load a completed run with `model/entities/*` or `reports/agent-outputs/domains/*` artifacts to populate the map.</span></div> : (
          <>
            <div className="domain-map-lanes" aria-label="Domain map nodes">
              {domainMap.groups.map((group) => <section className="domain-map-lane" key={group.key}><div className="domain-map-lane-head"><h3>{group.label}</h3><span>{group.nodes.length}</span></div><div className="domain-map-node-grid">{group.nodes.map((node) => <article className={`domain-map-node ${node.group}`} data-testid="review-domain-map-node" key={`${node.kind}-${node.id}`}><div><span className="metric-label">{node.typeLabel}</span><strong>{node.label}</strong><code>{node.id}</code></div><ArtifactPathButton path={node.artifact.path} label={node.artifact.label || node.artifact.path} kind={node.artifact.kind} actionLabel="Open map entity" onOpenArtifact={onOpenArtifact} /></article>)}</div></section>)}
            </div>
            <section className="domain-map-edge-list" data-testid="review-domain-map-edge-list">
              <div className="section-heading-row"><h2>Relationships</h2><StatusBadge tone={domainMap.edges.length > 0 ? "ok" : "info"}>{domainMap.edges.length} edges</StatusBadge></div>
              {domainMap.edges.length === 0 ? <p className="hint">No model edge artifacts are available yet. Entity nodes can still be reviewed through their YAML artifacts.</p> : <ul>{domainMap.edges.map((edge) => <li className="domain-map-edge" key={edge.id}><span><code>{edge.from}</code><strong>{edge.type}</strong><code>{edge.to}</code></span><ArtifactPathButton path={edge.artifact.path} label={edge.artifact.label || edge.id} kind={edge.artifact.kind} actionLabel="Open map edge" onOpenArtifact={onOpenArtifact} /></li>)}</ul>}
            </section>
          </>
        )}
      </section>
      <aside className="domain-map-inspector" data-testid="review-domain-map-inspector">
        <div className="section-heading-row"><h2>Map inspector</h2><StatusBadge tone={domainMap.blockers.length > 0 ? "warn" : hasMapData ? "ok" : "info"}>{domainMap.blockers.length > 0 ? "review" : hasMapData ? "ready" : "partial"}</StatusBadge></div>
        <dl className="compact-defs"><dt>Ownership</dt><dd>{domainMap.ownershipStatus}</dd><dt>Coverage</dt><dd>{domainMap.coverageStatus}</dd><dt>Cross-repo signal</dt><dd>{domainMap.crossRepoStatus}</dd><dt>Publication path</dt><dd>{domainMap.proposalArtifacts.length > 0 ? "Proposal artifacts ready for Publish review" : "Use Publish gate after proposals are generated"}</dd></dl>
        <section className="domain-map-blockers"><h3>Blockers / partial state</h3>{domainMap.blockers.length === 0 ? <p className="hint">No map-specific blockers detected from the available artifact list.</p> : <ul>{domainMap.blockers.map((blocker) => <li key={blocker}>{blocker}</li>)}</ul>}</section>
        <section className="domain-map-navigation"><h3>Evidence navigation</h3>{domainMap.navigationArtifacts.length === 0 ? <p className="hint">No model, domain or proposal artifacts are available for map navigation yet.</p> : <ul>{domainMap.navigationArtifacts.slice(0, 8).map((artifact) => <li key={`${artifact.kind}-${artifact.path}`}><ArtifactPathButton path={artifact.path} label={artifact.label} kind={artifact.kind} onOpenArtifact={onOpenArtifact} /></li>)}</ul>}</section>
      </aside>
    </div>
  );
}
