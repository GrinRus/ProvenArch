import type { ReactNode } from "react";

import type { SetupStep } from "../lib/appRoutes";

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
      <header className="setup-page-heading">
        <div><p className="eyebrow">Workspace setup</p><h1>Prepare your first architecture task</h1><p className="hint">Connect sources, define the question and confirm local readiness before analysis writes to the workspace.</p></div>
        <span className="setup-progress">{guidedSetupSteps.findIndex((item) => item.id === step) + 1} of {guidedSetupSteps.length}</span>
      </header>
      <div className="setup-workbench">
        <nav className="setup-stepper" aria-label="Guided setup steps">
          {guidedSetupSteps.map((item, index) => <button key={item.id} type="button" className={item.id === step ? "is-active" : index < guidedSetupSteps.findIndex((current) => current.id === step) ? "is-complete" : ""} data-testid={item.id === "workspace" ? "stage-source" : item.id === "brief" ? "stage-charter" : item.id === "runner" ? "stage-readiness" : `setup-step-${item.id}`} aria-current={item.id === step ? "page" : undefined} onClick={() => onStepChange(item.id)}><span className="setup-step-marker" aria-hidden="true">{index + 1}</span><span><strong>{item.label}</strong><small>{setupStepHint(item.id)}</small></span></button>)}
        </nav>
        <main className="setup-step-content">{children}</main>
      </div>
    </section>
  );
}

function setupStepHint(step: SetupStep): string {
  if (step === "workspace") return "Attach a local workspace";
  if (step === "sources") return "Choose repositories and scope";
  if (step === "brief") return "Set the architecture question";
  if (step === "runner") return "Check provider and permissions";
  return "Review the plan and start";
}

export function GuidedSetupReview({ briefReady, workspaceReady, busy, onStart }: { briefReady: boolean; workspaceReady: boolean; busy: boolean; onStart: () => void }) {
  return (
    <section className="panel stage-panel" data-testid="guided-setup-review">
      <div className="stage-header"><div><h1>Review & start</h1><p className="hint">Confirm the persisted workspace, analysis brief and runtime readiness before the first Task Attempt.</p></div></div>
      <dl className="compact-defs">
        <div><dt>Workspace</dt><dd>{workspaceReady ? "Ready" : "Needs validation"}</dd></div>
        <div><dt>Analysis brief</dt><dd>{briefReady ? "Saved" : "Not saved — starting requires a quality warning confirmation"}</dd></div>
      </dl>
      <button type="button" data-testid="guided-start-analysis" disabled={busy || !workspaceReady} onClick={onStart}>Start Task</button>
    </section>
  );
}
