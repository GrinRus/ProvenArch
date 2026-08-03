import type { ReactNode } from "react";

import type { RunCoordination, RunStatusResponse } from "../lib/appContracts";
import type { SetupStep } from "../lib/appRoutes";
import type { WorkflowState } from "../lib/workflowState";
import { Button, PageHeader, RouteTabs } from "./SemanticPrimitives";

export function HomePage({
  workflow,
  workspaceReady,
  coordination,
  runStatus,
  evidenceStatus,
  gitChanges,
  onPrimaryAction,
}: {
  workflow: WorkflowState;
  workspaceReady: boolean;
  coordination: RunCoordination;
  runStatus: RunStatusResponse | null;
  evidenceStatus: string;
  gitChanges: number;
  onPrimaryAction: () => void;
}) {
  const execution = coordination.active_run_id
    ? `Active · ${coordination.active_run_id}`
    : coordination.pending
      ? `Pending · ${coordination.pending.run_id}`
      : runStatus ? `${runStatus.pipeline} · ${runStatus.status}` : "Idle";
  const publication = gitChanges > 0 ? `${gitChanges} workspace changes` : "Clean";
  const evidence = evidenceStatus.replace(/_/g, " ");
  const attentionItems: Array<{ label: string; detail: string; tone: string }> = [
    !workspaceReady ? { label: "Finish workspace setup", detail: "Validate the workspace and at least one repository source.", tone: "blocked" } : null,
    coordination.active_run_id || coordination.pending ? { label: "Analysis is in progress", detail: "Open Runs to follow execution and review any intervention request.", tone: "active" } : null,
    evidenceStatus !== "available" ? { label: workflow.nextAction.label, detail: workflow.attention, tone: workflow.status } : null,
    gitChanges > 0 ? { label: "Review unpublished workspace changes", detail: `${gitChanges} change${gitChanges === 1 ? "" : "s"} can be inspected before an explicit Git action.`, tone: "review" } : null,
  ].filter((item): item is { label: string; detail: string; tone: string } => item !== null);
  const visibleAttention = attentionItems.length > 0 ? attentionItems.slice(0, 3) : [{ label: "Workspace is up to date", detail: "No immediate action is required. Explore current architecture or start a refresh when sources change.", tone: "complete" }];
  return (
    <section className="home-panel density-comfortable" data-testid="home-panel">
      <PageHeader
        title="Architecture workspace"
        purpose="Current knowledge, analysis activity and publication state for this workspace."
        state={<span className={`status ${workflow.status === "blocked" ? "err" : workflow.status === "complete" ? "ok" : "warn"}`}>{workflow.status.replace("_", " ")}</span>}
        action={<Button tone="primary" data-testid="home-primary-action" onClick={onPrimaryAction}>{workflow.nextAction.label}</Button>}
      />
      <dl className="home-axis-strip" aria-label="Workspace state">
        <div><dt>Workspace</dt><dd>{workspaceReady ? "Ready" : "Needs setup"}</dd></div>
        <div><dt>Execution</dt><dd>{execution}</dd></div>
        <div><dt>Evidence</dt><dd>{evidence}</dd></div>
        <div className={gitChanges > 0 ? "needs-attention" : ""}><dt>Publication</dt><dd>{publication}</dd></div>
      </dl>
      <div className="home-dashboard-grid">
        <section className="home-attention" aria-labelledby="home-attention-title">
          <div className="section-heading-row">
            <div><h2 id="home-attention-title">Needs attention</h2><p className="hint">The highest-priority next step for this workspace.</p></div>
          </div>
          <ol className="home-attention-list">
            {visibleAttention.map((item, index) => (
              <li className={`home-attention-item is-${item.tone}`} key={`${item.label}-${index}`}>
                <span className="attention-index" aria-hidden="true">{index + 1}</span>
                <div><strong>{item.label}</strong><p data-testid={index === 0 ? "home-attention-reason" : undefined}>{item.detail}</p></div>
              </li>
            ))}
          </ol>
          <section className="home-current-knowledge">
            <h2>Current architecture</h2>
            <p>{evidenceStatus === "available" ? "A validated evidence snapshot is available for review." : evidenceStatus === "partial" ? "Architecture knowledge is available with explicit evidence gaps." : "Run an analysis to create evidence-backed architecture knowledge."}</p>
            <p className="hint">Source repositories remain read-only. ProvenArch writes only to the architecture workspace.</p>
          </section>
        </section>
        <aside className="home-latest-run" aria-labelledby="home-latest-run-title">
          <h2 id="home-latest-run-title">Latest analysis</h2>
          {runStatus ? (
            <dl className="compact-defs">
              <div><dt>Pipeline</dt><dd>{runStatus.pipeline}</dd></div>
              <div><dt>Outcome</dt><dd>{runStatus.status}</dd></div>
              <div><dt>Runtime</dt><dd>{runStatus.runtime_mode || "unknown"}</dd></div>
              <div><dt>Publication</dt><dd>{publication}</dd></div>
            </dl>
          ) : <p className="empty-state">No analysis has been started yet.</p>}
        </aside>
      </div>
    </section>
  );
}

export const guidedSetupSteps: Array<{ id: SetupStep; label: string }> = [
  { id: "workspace", label: "Workspace" },
  { id: "sources", label: "Sources" },
  { id: "brief", label: "Analysis brief" },
  { id: "runner", label: "Runner & readiness" },
  { id: "review", label: "Review & start" },
];

export function GuidedSetupPage({ step, onStepChange, children }: { step: SetupStep; onStepChange: (step: SetupStep) => void; children: ReactNode }) {
  return (
    <section className="guided-setup" data-testid="guided-setup-page">
      <RouteTabs label="Guided setup steps" value={step} items={guidedSetupSteps.map((item) => ({ ...item, testId: item.id === "workspace" ? "stage-source" : item.id === "brief" ? "stage-charter" : item.id === "runner" ? "stage-readiness" : `setup-step-${item.id}` }))} onChange={onStepChange} />
      {children}
    </section>
  );
}

export function GuidedSetupReview({ briefReady, workspaceReady, busy, onStart }: { briefReady: boolean; workspaceReady: boolean; busy: boolean; onStart: () => void }) {
  return (
    <section className="panel stage-panel" data-testid="guided-setup-review">
      <div className="stage-header"><div><h1>Review & start</h1><p className="hint">Confirm the persisted workspace, analysis brief and runtime readiness before the first run.</p></div></div>
      <dl className="compact-defs">
        <div><dt>Workspace</dt><dd>{workspaceReady ? "Ready" : "Needs validation"}</dd></div>
        <div><dt>Analysis brief</dt><dd>{briefReady ? "Saved" : "Not saved — starting requires a quality warning confirmation"}</dd></div>
      </dl>
      <button type="button" data-testid="guided-start-analysis" disabled={busy || !workspaceReady} onClick={onStart}>Start analysis</button>
    </section>
  );
}

export function RunsPage({ coordination, selectedRunID, children }: { coordination: RunCoordination; selectedRunID?: string; children: ReactNode }) {
  return (
    <section className="runs-page" data-testid="runs-page">
      <header className="page-context-header">
        <div>
          {selectedRunID ? <a className="context-back-link" href="/runs">← All runs</a> : null}
          <h1>{selectedRunID ? "Run Studio" : "Runs"}</h1>
          <p className="hint">{selectedRunID ? `Execution detail and recovery for ${selectedRunID}.` : "Start an analysis, follow active work or open a previous run."}</p>
        </div>
        <p>{coordination.active_run_id ? `Active ${coordination.active_run_id}` : "No active run"}{coordination.pending ? ` · Pending ${coordination.pending.run_id}` : ""}</p>
      </header>
      {children}
    </section>
  );
}
