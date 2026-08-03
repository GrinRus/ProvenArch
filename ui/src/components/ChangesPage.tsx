import type { ReactNode } from "react";

import type { ArchitectureComparison, RunListItem } from "../lib/appContracts";
import type { ChangesView } from "../lib/appRoutes";
import { buildChangeReviewModel } from "../features/workbench/viewModels";
import { Button, PageHeader, RouteTabs } from "./SemanticPrimitives";

export function ChangesPage({
  runs,
  selectedRunID,
  selectedEvidenceStatus,
  sourceMode = "snapshot",
  view,
  onViewChange,
  onSelectChangeReview,
  onOpenRunStudio,
  architectureComparison,
  children,
}: {
  runs: RunListItem[];
  selectedRunID: string | null;
  selectedEvidenceStatus: string;
  sourceMode?: "snapshot" | "current";
  view: ChangesView;
  onViewChange: (view: ChangesView) => void;
  onSelectChangeReview: (id: string) => void;
  onOpenRunStudio: (id: string) => void;
  architectureComparison?: ArchitectureComparison;
  children: ReactNode;
}) {
  const reviewCandidates = buildChangeReviewModel(runs, selectedRunID, selectedEvidenceStatus);
  return (
    <section className="changes-page" data-testid="changes-page">
      <PageHeader
        title="Architecture changes"
        purpose="Understand what changed, inspect supporting evidence and prepare the workspace for publication."
        source={sourceMode === "current" ? "Current workspace · read-only" : selectedRunID ? `Run snapshot · ${selectedRunID}` : "Choose a review package"}
        action={<Button tone="primary" data-testid="stage-publish" aria-current={view === "publish" ? "page" : undefined} onClick={() => onViewChange("publish")}>{view === "publish" ? "Publication review" : "Continue to publish"}</Button>}
      />
      <RouteTabs
        label="Change Review views"
        value={view}
        items={(["overview", "findings", "proposals", "diff"] as ChangesView[]).map((id) => ({ id, label: label(id), testId: `stage-${id === "overview" ? "review" : id}` }))}
        onChange={onViewChange}
      />
      {(view === "overview" || view === "findings") ? <SemanticArchitectureChanges comparison={architectureComparison} focus={view === "findings" ? "review" : "all"} /> : null}
      {view === "overview" ? (
        <aside className="panel review-packages" data-testid="review-packages">
          <h2>Review packages</h2>
          {reviewCandidates.length === 0 ? <p className="empty-state">No analysis history is available.</p> : <ul className="compact-list">{reviewCandidates.map((run) => {
            return <li key={run.run_id}><div><strong>{run.pipeline} · {run.run_id}</strong><span>{run.status}</span><span>{refreshLabel(run)}</span><span>Publication: Unknown</span></div>{run.action === "review" ? <button type="button" onClick={() => onSelectChangeReview(run.run_id)}>Change Review</button> : <button type="button" onClick={() => onOpenRunStudio(run.run_id)}>Open Run Studio</button>}</li>;
          })}</ul>}
        </aside>
      ) : null}
      {children}
    </section>
  );
}

function SemanticArchitectureChanges({ comparison, focus }: { comparison?: ArchitectureComparison; focus: "all" | "review" }) {
  if (!comparison?.available) return <section className="panel semantic-changes" data-testid="semantic-changes"><div><p className="eyebrow">Promoted baseline comparison</p><h2>Semantic comparison is not available yet</h2><p>{comparison?.reason || "Run and promote a second architecture generation to establish a baseline."}</p></div></section>;
  const categories = focus === "review" ? (["findings", "gaps"] as const) : (["entities", "edges", "findings", "gaps"] as const);
  return <section className="panel semantic-changes" data-testid="semantic-changes"><header><div><p className="eyebrow">Promoted baseline comparison</p><h2>What changed in the architecture</h2><p><code>{comparison.baseline_run_id}</code> → <code>{comparison.current_run_id}</code></p></div><span className="status ok">Semantic model</span></header><div className="semantic-change-grid">{categories.map((category) => { const changes = comparison.categories[category]; return <article key={category}><h3>{category}</h3><dl><div><dt>Added</dt><dd>{changes.added.length}</dd></div><div><dt>Changed</dt><dd>{changes.changed.length}</dd></div><div><dt>Removed</dt><dd>{changes.removed.length}</dd></div></dl>{[...changes.added.map((item) => ({ ...item, state: "added" })), ...changes.changed.map((item) => ({ ...item, state: "changed" })), ...changes.removed.map((item) => ({ ...item, state: "removed" }))].length > 0 ? <ul>{[...changes.added.map((item) => ({ ...item, state: "added" })), ...changes.changed.map((item) => ({ ...item, state: "changed" })), ...changes.removed.map((item) => ({ ...item, state: "removed" }))].slice(0, 6).map((item) => <li key={`${item.state}:${item.id}`}><span className={`change-state ${item.state}`}>{item.state}</span><strong>{item.name || item.id}</strong></li>)}</ul> : <p className="empty-state">No semantic changes.</p>}</article>; })}</div></section>;
}

function refreshLabel(run: RunListItem): string {
  if (run.pipeline !== "refresh") return "Initial architecture build";
  if (!run.refresh_summary) return "Refresh details unavailable";
  if (run.refresh_summary.mode === "no_op") return "No changes in analysis scope";
  const title = run.refresh_summary.mode === "affected_only" ? "Affected scope refresh" : "Full refresh";
  return `${title} · ${run.refresh_summary.updated} updated · ${run.refresh_summary.preserved} preserved · ${run.refresh_summary.uncertain} uncertain`;
}

function label(view: ChangesView): string {
  return view.charAt(0).toUpperCase() + view.slice(1);
}
