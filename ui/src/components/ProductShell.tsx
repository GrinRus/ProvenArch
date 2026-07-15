import type { ReactNode } from "react";

import { destinationPaths } from "../lib/appRoutes";
import type { WorkflowDestination, WorkflowState } from "../lib/workflowState";

type ProductShellProps = {
  destination: WorkflowDestination;
  workflow: WorkflowState;
  workspacePath: string;
  runtimeLabel: string;
  buildLabel: string;
  buildTitle: string;
  workspaceValid: boolean;
  children: ReactNode;
  onDestinationChange: (destination: WorkflowDestination) => void;
  onAsk: () => void;
  onSettings: () => void;
  onDiagnostics: () => void;
  onRefresh: () => void;
};

const destinations: Array<{ id: WorkflowDestination; label: string }> = [
  { id: "home", label: "Home" },
  { id: "runs", label: "Runs" },
  { id: "knowledge", label: "Knowledge" },
  { id: "changes", label: "Changes" },
  { id: "setup", label: "Setup" },
];

const destinationTestIDs: Partial<Record<WorkflowDestination, string>> = {
  runs: "stage-analysis",
};

export function ProductShell({ destination, workflow, workspacePath, runtimeLabel, buildLabel, buildTitle, workspaceValid, children, onDestinationChange, onAsk, onSettings, onDiagnostics, onRefresh }: ProductShellProps) {
  return (
    <main className="product-shell" data-testid="product-shell">
      <header className="product-header" data-testid="top-status-bar">
        <div><strong>ACP</strong><span title={workspacePath}>{workspacePath}</span><span>{workspaceValid ? "workspace valid" : "workspace needs attention"}</span><span>{runtimeLabel}</span><span data-testid="brand-version" title={buildTitle}>{buildLabel}</span></div>
        <nav aria-label="Primary">
          {destinations.map((item) => (
            <a
              key={item.id}
              href={destinationPaths[item.id]}
              data-testid={destinationTestIDs[item.id]}
              aria-current={destination === item.id ? "page" : undefined}
              onClick={(event) => { event.preventDefault(); onDestinationChange(item.id); }}
            >{item.label}</a>
          ))}
        </nav>
        <div className="product-utilities">
          <button type="button" data-testid="stage-ask" onClick={onAsk}>Ask</button>
          <button type="button" data-testid="console-refresh-btn" onClick={onRefresh}>Refresh</button>
          <button type="button" onClick={onSettings}>Settings</button>
          <button type="button" onClick={onDiagnostics}>Diagnostics</button>
        </div>
      </header>
      <aside className={`workflow-banner is-${workflow.status}`} aria-live="polite">
        <strong>{workflow.status.replace("_", " ")}</strong><span>{workflow.attention}</span>
      </aside>
      <div className="product-content">{children}</div>
    </main>
  );
}
