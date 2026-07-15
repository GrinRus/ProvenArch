import type { ReactNode } from "react";

import type { RunListItem } from "../lib/appContracts";
import type { ChangesView } from "../lib/appRoutes";

export function ChangesPage({
  runs,
  selectedRunID,
  selectedEvidenceStatus,
  view,
  onViewChange,
  onSelectChangeReview,
  onOpenRunStudio,
  children,
}: {
  runs: RunListItem[];
  selectedRunID: string | null;
  selectedEvidenceStatus: string;
  view: ChangesView;
  onViewChange: (view: ChangesView) => void;
  onSelectChangeReview: (id: string) => void;
  onOpenRunStudio: (id: string) => void;
  children: ReactNode;
}) {
  const reviewCandidates = runs.filter((run) => run.pipeline !== "qa" && (run.pipeline === "init" || run.pipeline === "refresh"));
  return (
    <section className="changes-page" data-testid="changes-page">
      <header className="page-context-header"><div><h1>Changes</h1><p className="hint">Historical review packages stay bound to their run snapshot; publication state is evaluated from the current workspace.</p></div></header>
      <nav className="destination-tabs" aria-label="Change Review views">
        {(["overview", "evidence", "findings", "proposals", "diff", "publish"] as ChangesView[]).map((item) => <button key={item} type="button" data-testid={`stage-${item === "overview" ? "review" : item}`} aria-current={view === item ? "page" : undefined} onClick={() => onViewChange(item)}>{label(item)}</button>)}
      </nav>
      {view === "overview" ? (
        <aside className="panel review-packages" data-testid="review-packages">
          <h2>Review packages</h2>
          {reviewCandidates.length === 0 ? <p className="empty-state">No analysis history is available.</p> : <ul className="compact-list">{reviewCandidates.map((run) => {
            const selected = run.run_id === selectedRunID;
            const reviewable = run.status === "succeeded" && run.authoritative_index === true && (!selected || selectedEvidenceStatus === "available" || selectedEvidenceStatus === "partial");
            return <li key={run.run_id}><div><strong>{run.pipeline} · {run.run_id}</strong><span>{run.status}</span><span>Publication: Unknown</span></div>{reviewable ? <button type="button" onClick={() => onSelectChangeReview(run.run_id)}>Change Review</button> : <button type="button" onClick={() => onOpenRunStudio(run.run_id)}>Open Run Studio</button>}</li>;
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
