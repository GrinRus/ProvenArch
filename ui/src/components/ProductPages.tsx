import type { ReactNode } from "react";

import type { RunCoordination, RunStatusResponse } from "../lib/appContracts";
import type { SetupStep } from "../lib/appRoutes";
import type { WorkflowState } from "../lib/workflowState";
import { Button, MetricGrid, PageHeader, RouteTabs } from "./SemanticPrimitives";

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
  return (
    <section className="panel stage-panel home-panel density-comfortable" data-testid="home-panel">
      <PageHeader title="Architecture workspace" purpose="One authoritative view of readiness, execution, evidence and publication." state={<span className={`status ${workflow.status === "blocked" ? "err" : workflow.status === "complete" ? "ok" : "warn"}`}>{workflow.status.replace("_", " ")}</span>} action={<Button tone="primary" data-testid="home-primary-action" onClick={onPrimaryAction}>{workflow.nextAction.label}</Button>} />
      <MetricGrid items={[{ label: "Workspace", value: workspaceReady ? "Ready" : "Needs setup" }, { label: "Execution", value: execution }, { label: "Latest accepted evidence", value: evidenceStatus.replace(/_/g, " ") }, { label: "Publication", value: gitChanges > 0 ? `${gitChanges} workspace changes` : "Clean" }]} />
      <p data-testid="home-attention-reason">{workflow.attention}</p>
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

export function RunsPage({ coordination, children }: { coordination: RunCoordination; children: ReactNode }) {
  return (
    <section className="runs-page" data-testid="runs-page">
      <header className="page-context-header">
        <div><h1>Runs</h1><p className="hint">History and Run Studio share the same server-authored coordination state.</p></div>
        <p>{coordination.active_run_id ? `Active ${coordination.active_run_id}` : "No active run"}{coordination.pending ? ` · Pending ${coordination.pending.run_id}` : ""}</p>
      </header>
      {children}
    </section>
  );
}
