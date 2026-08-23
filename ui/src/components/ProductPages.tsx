import type { ReactNode } from "react";

import type { SetupStep } from "../lib/appRoutes";

export const guidedSetupSteps: Array<{ id: SetupStep; label: string }> = [
  { id: "workspace", label: "Workspace" },
  { id: "sources", label: "Repositories" },
  { id: "runner", label: "Provider & readiness" },
  { id: "review", label: "Review & start" },
];

export function GuidedSetupPage({ step, onStepChange, children }: { step: SetupStep; onStepChange: (step: SetupStep) => void; children: ReactNode }) {
  return (
    <section className="guided-setup" data-testid="guided-setup-page">
      <header className="setup-page-heading">
        <div><p className="eyebrow">Workspace setup</p><h1>Prepare your first architecture task</h1><p className="hint">Connect sources, define the question and confirm local readiness before analysis writes to the workspace.</p></div>
        <span className="setup-progress">{guidedSetupSteps.findIndex((item) => item.id === (step === "brief" ? "review" : step)) + 1} of {guidedSetupSteps.length}</span>
      </header>
      <div className="setup-workbench">
        <nav className="setup-stepper" aria-label="Guided setup steps">
          {guidedSetupSteps.map((item, index) => { const displayStep = step === "brief" ? "review" : step; const activeIndex = guidedSetupSteps.findIndex((current) => current.id === displayStep); return <button key={item.id} type="button" className={item.id === displayStep ? "is-active" : index < activeIndex ? "is-complete" : ""} data-testid={item.id === "workspace" ? "stage-source" : item.id === "runner" ? "stage-readiness" : `setup-step-${item.id}`} aria-current={item.id === displayStep ? "page" : undefined} onClick={() => onStepChange(item.id)}><span className="setup-step-marker" aria-hidden="true">{index + 1}</span><span><strong>{item.label}</strong><small>{setupStepHint(item.id)}</small></span></button>; })}
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

export function GuidedSetupReview({ workspaceReady, busy, repoCount, docsImportsReady, runtimeLabel, readinessLabel, onBack, onCreateTask }: { workspaceReady: boolean; busy: boolean; repoCount: number; docsImportsReady: boolean; runtimeLabel: string; readinessLabel: string; onBack: () => void; onCreateTask: () => void }) {
  return (
    <section className="panel stage-panel" data-testid="guided-setup-review">
      <div className="stage-header"><div><p className="eyebrow">Final setup check</p><h2>Ready to create your first Task?</h2><p className="hint">Confirm the persisted workspace, source boundary and runtime readiness. Analysis starts only after you define the Task goal.</p></div></div>
      <div className="setup-review-summary" aria-label="Setup review summary">
        <div><span>Workspace</span><strong>{workspaceReady ? "Ready" : "Needs validation"}</strong></div>
        <div><span>Sources</span><strong>{repoCount} {repoCount === 1 ? "repository" : "repositories"}</strong></div>
        <div><span>Docs imports</span><strong>{docsImportsReady ? "Configured" : "Default"}</strong></div>
        <div><span>Runner</span><strong>{runtimeLabel || "Not configured"}</strong></div>
        <div><span>Readiness</span><strong>{readinessLabel}</strong></div>
      </div>
      <div className="setup-review-next" data-testid="setup-review-next">
        <div><strong>What happens next</strong><span>Create a Task, run evidence-backed analysis, then review the generated architecture workspace.</span></div>
        <button type="button" className="ui-button tone-primary" data-testid="guided-create-task" disabled={busy || !workspaceReady} onClick={onCreateTask}>Create first Task</button>
      </div>
      <div className="actions setup-review-actions">
        <button type="button" className="ui-button tone-neutral" onClick={onBack}>Back to readiness</button>
      </div>
    </section>
  );
}
