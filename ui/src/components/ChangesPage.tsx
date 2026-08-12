import type { ReactNode } from "react";

import type { ArchitectureComparison, RunListItem, RunReviewContract } from "../lib/appContracts";
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
  architectureComparisonMismatch = false,
  runReview,
  taskId,
  attemptId,
  onOpenTask,
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
  architectureComparisonMismatch?: boolean;
  runReview?: RunReviewContract;
  taskId?: string;
  attemptId?: string;
  onOpenTask?: (taskId: string) => void;
  children: ReactNode;
}) {
  const reviewCandidates = buildChangeReviewModel(runs, selectedRunID, selectedEvidenceStatus);
  const readOnlyWorkspace = sourceMode === "current";
  const changeViews = (readOnlyWorkspace ? ["evidence"] : ["overview", "evidence", "findings", "proposals", "diff"]) as ChangesView[];
  const selectedRun = selectedRunID ? runs.find((run) => run.run_id === selectedRunID) : undefined;
  const selectedRunReviewable = Boolean(selectedRun && selectedRun.status === "succeeded" && selectedRun.authoritative_index === true);
  const selectedRunBlocked = Boolean(selectedRunID && selectedRun && !selectedRunReviewable);
  const effectiveRunReview = selectedRunReviewable ? runReview : undefined;
  const canPublish = !readOnlyWorkspace && !selectedRunID ? true : !readOnlyWorkspace && selectedRunReviewable;
  return (
    <section className="changes-page" data-testid="changes-page">
      <PageHeader
        title={readOnlyWorkspace ? "Current workspace evidence" : !selectedRunReviewable && selectedRunID ? "Run needs recovery" : effectiveRunReview?.review_kind === "initial" ? "Review initial architecture" : "Review architecture update"}
        purpose={readOnlyWorkspace ? "Inspect the current workspace without mixing it with a historical run snapshot." : "Decide whether the new knowledge should replace the current snapshot. Review semantic changes first; inspect Git files only when needed."}
        source={sourceMode === "current" ? "Current workspace · read-only" : selectedRunID ? `Run snapshot · ${selectedRunID}` : "Choose a review package"}
        action={readOnlyWorkspace ? <span className="status info" data-testid="changes-read-only-badge">Read-only workspace</span> : !canPublish && selectedRunID ? <Button data-testid="changes-open-run-studio" onClick={() => onOpenRunStudio(selectedRunID)}>Open Run Studio</Button> : <Button tone="primary" data-testid="stage-publish" aria-current={view === "publish" ? "page" : undefined} onClick={() => onViewChange("publish")}>{view === "publish" ? "Publication review" : "Continue to publish"}</Button>}
      />
      {taskId ? <aside className="task-changes-context" data-testid="task-changes-context" aria-label="Task Changes context"><div><p className="eyebrow">Task context</p><strong>Changes for the selected Task</strong><p><code>{taskId}</code> · Attempt <code>{attemptId || "unavailable"}</code> · exact selected run identity only</p><span className="hint">No latest-run fallback; Current workspace evidence and historical snapshot publication stay distinct.</span></div>{onOpenTask ? <button type="button" className="ui-button tone-neutral" onClick={() => onOpenTask(taskId)}>Back to Task</button> : null}</aside> : null}
      <RouteTabs
        label="Change Review views"
        value={view}
        items={changeViews.map((id) => ({ id, label: label(id), testId: `stage-${id === "overview" ? "review" : id}` }))}
        onChange={onViewChange}
      />
      {!readOnlyWorkspace && selectedRunBlocked && (view === "overview" || view === "findings") ? <UnavailableRunReview /> : null}
      {!readOnlyWorkspace && !selectedRunBlocked && (view === "overview" || view === "findings") ? <SemanticArchitectureChanges comparison={architectureComparison} comparisonMismatch={architectureComparisonMismatch} runReview={effectiveRunReview} selectedRunID={selectedRunID} focus={view === "findings" ? "review" : "all"} /> : null}
      {!readOnlyWorkspace && view === "overview" ? (
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

function UnavailableRunReview() {
  return <section className="panel semantic-changes" data-testid="semantic-changes"><div><p className="eyebrow">Selected run</p><h2>Change Review is unavailable</h2><p>This run did not produce a successful authoritative snapshot. Open Run Studio to inspect recovery details or choose a successful indexed run.</p></div><span className="status blocked">Run Studio</span></section>;
}

function SemanticArchitectureChanges({ comparison, comparisonMismatch, runReview, selectedRunID, focus }: { comparison?: ArchitectureComparison; comparisonMismatch: boolean; runReview?: RunReviewContract; selectedRunID: string | null; focus: "all" | "review" }) {
  if (runReview && (!selectedRunID || runReview.source_run_id !== selectedRunID)) return <section className="panel semantic-changes" data-testid="semantic-changes"><div><p className="eyebrow">Selected run review</p><h2>Review unavailable for selected run</h2><p>This review package has no matching selected run, so no semantic or document delta is shown here.</p></div><span className="status blocked">Review snapshot</span></section>;
  if (runReview) return <RunPinnedReview review={runReview} focus={focus} />;
  if (comparisonMismatch) return <section className="panel semantic-changes" data-testid="semantic-changes"><div><p className="eyebrow">Selected run comparison</p><h2>Comparison unavailable for selected run</h2><p>The promoted architecture comparison belongs to another run, so no current delta is shown here.</p></div><span className="status blocked">Review snapshot</span></section>;
  if (!comparison?.available) return <section className="panel semantic-changes" data-testid="semantic-changes"><div><p className="eyebrow">Promoted baseline comparison</p><h2>Semantic comparison is not available yet</h2><p>{comparison?.reason || "Run and promote a second architecture generation to establish a baseline."}</p></div></section>;
  const categories = focus === "review" ? (["findings", "gaps"] as const) : (["entities", "edges", "findings", "gaps"] as const);
  return <section className="panel semantic-changes" data-testid="semantic-changes"><header><div><p className="eyebrow">Promoted baseline comparison</p><h2>What changed in the architecture</h2><p><code>{comparison.baseline_run_id}</code> → <code>{comparison.current_run_id}</code></p></div><span className="status ok">Semantic model</span></header><div className="semantic-change-grid">{categories.map((category) => { const changes = comparison.categories[category]; return <article key={category}><h3>{category}</h3><dl><div><dt>Added</dt><dd>{changes.added.length}</dd></div><div><dt>Changed</dt><dd>{changes.changed.length}</dd></div><div><dt>Removed</dt><dd>{changes.removed.length}</dd></div></dl>{[...changes.added.map((item) => ({ ...item, state: "added" })), ...changes.changed.map((item) => ({ ...item, state: "changed" })), ...changes.removed.map((item) => ({ ...item, state: "removed" }))].length > 0 ? <ul>{[...changes.added.map((item) => ({ ...item, state: "added" })), ...changes.changed.map((item) => ({ ...item, state: "changed" })), ...changes.removed.map((item) => ({ ...item, state: "removed" }))].slice(0, 6).map((item) => <li key={`${item.state}:${item.id}`}><span className={`change-state ${item.state}`}>{item.state}</span><strong>{item.name || item.id}</strong></li>)}</ul> : <p className="empty-state">No semantic changes.</p>}</article>; })}</div></section>;
}

function RunPinnedReview({ review, focus }: { review: RunReviewContract; focus: "all" | "review" }) {
  const comparison = review.semantic_changes;
  const categories = focus === "review" ? (["findings", "gaps"] as const) : (["entities", "edges", "findings", "gaps"] as const);
  const title = review.review_kind === "initial" ? "Initial architecture summary" : comparison.available ? "What changed in the architecture" : "Architecture update is ready for review";
  const documentItems = [...review.document_changes.added.map((item) => ({ ...item, state: "added" })), ...review.document_changes.changed.map((item) => ({ ...item, state: "changed" })), ...review.document_changes.removed.map((item) => ({ ...item, state: "removed" }))];
  return (
    <section className="panel semantic-changes" data-testid="semantic-changes">
      <header>
        <div><p className="eyebrow">{review.review_kind === "initial" ? "Run-pinned initial review" : "Run-pinned architecture update"}</p><h2>{title}</h2><p><code>{review.source_run_id}</code>{review.baseline_run_id ? <> · baseline <code>{review.baseline_run_id}</code></> : null}</p></div>
        <span className={`status ${comparison.available ? "ok" : "info"}`}>{comparison.available ? "Semantic delta" : "Snapshot summary"}</span>
      </header>
      {comparison.available ? <ReviewDeltaStrip comparison={comparison} review={review} /> : <p className="empty-state">This is the first architecture generation. Review the current documents, findings and coverage gaps as the baseline for future refreshes.</p>}
      {comparison.available ? <div className="semantic-change-grid">{categories.map((category) => <ChangeCategory key={category} category={category} changes={comparison.categories[category]} />)}</div> : null}
      <div className="review-summary-grid" data-testid="run-pinned-review-summary">
        <div><strong>{review.summary.documents_added + review.summary.documents_changed + review.summary.documents_removed}</strong><span>document changes</span></div>
        <div><strong>{review.summary.findings}</strong><span>findings</span></div>
        <div><strong>{review.summary.questions}</strong><span>questions</span></div>
        <div><strong>{review.summary.gaps}</strong><span>coverage gaps</span></div>
        <div><strong>{review.runtime.mode || "Unknown"}</strong><span>runtime</span></div>
        <div><strong>{review.authority.mode}</strong><span>authority</span></div>
      </div>
      <section className="run-review-detail-grid" aria-label="Run review details">
        <article><h3>Document changes <span>{documentItems.length}</span></h3>{!review.document_changes.available ? <p className="empty-state">{review.document_changes.reason || "Document inventory is unavailable."}</p> : documentItems.length === 0 ? <p className="empty-state">No promoted document changes.</p> : <ul className="compact-list">{documentItems.slice(0, 8).map((item) => <li key={`${item.state}:${item.id}`}><span className={`change-state ${item.state}`}>{item.state}</span><code>{item.path || item.id}</code></li>)}</ul>}</article>
        <article className="review-decision"><h3>Review decision <span>{review.findings.length + review.questions.length ? `${review.findings.length + review.questions.length} open` : "ready"}</span></h3><p className="hint">Accept knowledge, not individual files.</p>{review.findings.length === 0 && review.questions.length === 0 ? <p className="status ok">Model integrity and evidence quality passed.</p> : <ul className="compact-list">{review.findings.slice(0, 4).map((finding) => <li key={finding.id}><strong>{finding.title}</strong><span>{finding.severity}</span></li>)}{review.questions.slice(0, 4).map((question) => <li key={question.id}><strong>{question.text}</strong><span>{question.priority || "open"}</span></li>)}</ul>}<div className="review-decision-notice"><strong>Current snapshot remains unchanged</strong><span>Publishing promotes this reviewed update and commits the complete workspace.</span></div></article>
      </section>
    </section>
  );
}

function ReviewDeltaStrip({ comparison, review }: { comparison: NonNullable<RunReviewContract["semantic_changes"]>; review: RunReviewContract }) {
  const entities = comparison.categories.entities;
  const edges = comparison.categories.edges;
  const findings = comparison.categories.findings;
  const gaps = comparison.categories.gaps;
  const counts = [
    ["Added", entities.added.length + edges.added.length, "integration", "added"],
    ["Changed", entities.changed.length + edges.changed.length, "documents", "changed"],
    ["Removed", entities.removed.length + edges.removed.length, "entities", "removed"],
    ["Needs decision", review.summary.questions + findings.added.length + gaps.added.length, "open gaps", "warn"],
  ] as const;
  return <div className="review-delta-strip" data-testid="review-delta-strip">{counts.map(([label, value, detail, tone]) => <div className={`review-delta-card is-${tone}`} key={label}><span>{label}</span><strong>{value}</strong><small>{detail}</small></div>)}</div>;
}

function ChangeCategory({ category, changes }: { category: string; changes: { added: Array<{ id: string; name: string }>; changed: Array<{ id: string; name: string }>; removed: Array<{ id: string; name: string }> } }) {
  const items = [...changes.added.map((item) => ({ ...item, state: "added" })), ...changes.changed.map((item) => ({ ...item, state: "changed" })), ...changes.removed.map((item) => ({ ...item, state: "removed" }))];
  return <article><h3>{category}</h3><dl><div><dt>Added</dt><dd>{changes.added.length}</dd></div><div><dt>Changed</dt><dd>{changes.changed.length}</dd></div><div><dt>Removed</dt><dd>{changes.removed.length}</dd></div></dl>{items.length > 0 ? <ul>{items.slice(0, 6).map((item) => <li key={`${item.state}:${item.id}`}><span className={`change-state ${item.state}`}>{item.state}</span><strong>{item.name || item.id}</strong></li>)}</ul> : <p className="empty-state">No semantic changes.</p>}</article>;
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
