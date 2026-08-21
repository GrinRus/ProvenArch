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

export function GuidedSetupReview({ briefReady, workspaceReady, busy, repoCount, docsImportsReady, runtimeLabel, readinessLabel, onBack, onCreateTask, onStart }: { briefReady: boolean; workspaceReady: boolean; busy: boolean; repoCount: number; docsImportsReady: boolean; runtimeLabel: string; readinessLabel: string; onBack: () => void; onCreateTask: () => void; onStart: () => void }) {
  return (
    <section className="panel stage-panel" data-testid="guided-setup-review">
      <div className="stage-header"><div><h1>Review & start</h1><p className="hint">Confirm the persisted workspace, analysis brief and runtime readiness before the first Task Attempt.</p></div></div>
      <div className="setup-review-summary" aria-label="Setup review summary">
        <div><span>Workspace</span><strong>{workspaceReady ? "Ready" : "Needs validation"}</strong></div>
        <div><span>Sources</span><strong>{repoCount} {repoCount === 1 ? "repository" : "repositories"}</strong></div>
        <div><span>Docs imports</span><strong>{docsImportsReady ? "Configured" : "Default"}</strong></div>
        <div><span>Runner</span><strong>{runtimeLabel || "Not configured"}</strong></div>
        <div><span>Readiness</span><strong>{readinessLabel}</strong></div>
        <div><span>Analysis brief</span><strong>{briefReady ? "Saved" : "Needs review"}</strong></div>
      </div>
      {!briefReady ? <p className="status warn">Save the analysis brief before starting to keep the result focused and evidence-backed.</p> : null}
      <div className="setup-review-next" data-testid="setup-review-next">
        <div><strong>What happens next</strong><span>Create a task, run evidence-backed analysis, then review the generated architecture workspace.</span></div>
        <button type="button" className="ui-button tone-primary" data-testid="guided-create-task" disabled={busy || !workspaceReady} onClick={onCreateTask}>Create first task</button>
      </div>
      <div className="actions setup-review-actions">
        <button type="button" className="ui-button tone-neutral" onClick={onBack}>Back to readiness</button>
        <button type="button" className="ui-button tone-neutral" data-testid="guided-start-analysis" disabled={busy || !workspaceReady} onClick={onStart}>Start analysis</button>
      </div>
    </section>
  );
}
