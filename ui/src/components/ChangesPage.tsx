import type { ReactNode } from "react";

import type { RunListItem } from "../lib/appContracts";
import type { ChangesView } from "../lib/appRoutes";
import { buildChangeReviewModel } from "../features/workbench/viewModels";
import { PageHeader, RouteTabs } from "./SemanticPrimitives";

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
      <PageHeader title="Changes" purpose="Historical review packages stay bound to their run snapshot; publication state is evaluated from the current workspace." source={sourceMode === "current" ? "Current workspace · read-only" : selectedRunID ? `Run snapshot · ${selectedRunID}` : "Choose a review package"} />
      <RouteTabs label="Change Review views" value={view} items={(["overview", "evidence", "findings", "proposals", "diff", "publish"] as ChangesView[]).map((id) => ({ id, label: label(id), testId: `stage-${id === "overview" ? "review" : id}` }))} onChange={onViewChange} />
      {view === "overview" ? (
        <aside className="panel review-packages" data-testid="review-packages">
          <h2>Review packages</h2>
          {reviewCandidates.length === 0 ? <p className="empty-state">No analysis history is available.</p> : <ul className="compact-list">{reviewCandidates.map((run) => {
            return <li key={run.run_id}><div><strong>{run.pipeline} · {run.run_id}</strong><span>{run.status}</span><span>Publication: Unknown</span></div>{run.action === "review" ? <button type="button" onClick={() => onSelectChangeReview(run.run_id)}>Change Review</button> : <button type="button" onClick={() => onOpenRunStudio(run.run_id)}>Open Run Studio</button>}</li>;
          })}</ul>}
        </aside>
      ) : null}
      {children}
    </section>
  );
}

function label(view: ChangesView): string {
  return view.charAt(0).toUpperCase() + view.slice(1);
}
