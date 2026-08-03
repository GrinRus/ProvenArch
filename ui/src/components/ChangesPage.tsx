import type { ReactNode } from "react";

import type { RunListItem } from "../lib/appContracts";
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
