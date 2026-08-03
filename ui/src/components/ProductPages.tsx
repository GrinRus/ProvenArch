import type { ReactNode } from "react";

import type { ArchitectureResponse, RunCoordination, RunStatusResponse } from "../lib/appContracts";
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
	architecture,
	onOpenArchitecture,
}: {
  workflow: WorkflowState;
  workspaceReady: boolean;
  coordination: RunCoordination;
  runStatus: RunStatusResponse | null;
  evidenceStatus: string;
  gitChanges: number;
  onPrimaryAction: () => void;
	architecture?: ArchitectureResponse | null;
	onOpenArchitecture?: () => void;
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
	const contextNodes = architecture?.views?.context?.nodes ?? [];
	const services = contextNodes.filter((node) => node.type === "service");
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
	  <section className="home-architecture-outcome" aria-labelledby="home-architecture-title">
		<div><p className="eyebrow">Current promoted result</p><h2 id="home-architecture-title">{services.length > 0 ? `${services.length} validated service${services.length === 1 ? "" : "s"} in the architecture model` : "Architecture boundary is not established yet"}</h2><p>{architecture?.status === "partial" ? "The result is usable with explicit model or evidence gaps." : architecture?.status === "available" ? `Explore ${architecture.counts.entities} entities, ${architecture.counts.edges} relationships and ${architecture.counts.evidence} repository evidence references.` : "Run analysis to produce validator-approved Architecture Home, model and C4 views."}</p>{contextNodes.length > 0 ? <div className="home-map-preview" aria-label="System context preview">{contextNodes.slice(0, 4).map((node, index) => <span key={node.id}><button type="button" onClick={onOpenArchitecture}>{node.name}</button>{index < Math.min(contextNodes.length, 4) - 1 ? <i aria-hidden="true">→</i> : null}</span>)}{contextNodes.length > 4 ? <small>+{contextNodes.length - 4} more</small> : null}</div> : null}{architecture?.review && architecture.review.findings.length + architecture.review.questions.length > 0 ? <div className="home-review-preview"><h3>Top review items</h3><ul className="compact-list">{architecture.review.findings.slice(0, 2).map((item) => <li key={item.id}><strong>{item.title}</strong><span>{item.severity} finding</span></li>)}{architecture.review.questions.slice(0, Math.max(0, 3 - Math.min(2, architecture.review.findings.length))).map((item) => <li key={item.id}><strong>{item.text}</strong><span>{item.priority || "open"} question</span></li>)}</ul></div> : null}</div>
		<dl><div><dt>Entities</dt><dd>{architecture?.counts.entities ?? 0}</dd></div><div><dt>Relationships</dt><dd>{architecture?.counts.edges ?? 0}</dd></div><div><dt>Evidence</dt><dd>{architecture?.counts.evidence ?? 0}</dd></div></dl>
		{onOpenArchitecture ? <div className="home-architecture-action"><button type="button" onClick={onOpenArchitecture} disabled={!architecture || architecture.status === "unavailable"} aria-describedby={!architecture || architecture.status === "unavailable" ? "architecture-action-reason" : undefined}>Explore architecture</button>{!architecture || architecture.status === "unavailable" ? <p id="architecture-action-reason" className="disabled-reason">Available after a validator-approved analysis result is promoted.</p> : null}</div> : null}
	  </section>
    </section>
  );
}

export const guidedSetupSteps: Array<{ id: SetupStep; label: string }> = [
  { id: "workspace", label: "Workspace" },
  { id: "sources", label: "Repositories" },
  { id: "brief", label: "Analysis brief" },
  { id: "runner", label: "Provider & readiness" },
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
