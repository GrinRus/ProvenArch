import { useState, type ReactNode } from "react";

import { destinationPaths } from "../lib/appRoutes";
import type { WorkflowDestination, WorkflowState } from "../lib/workflowState";
import { ContextDrawer } from "./ContextDrawer";

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
  onDiagnostics: () => void;
  onRefresh: () => void;
};

const destinations: Array<{ id: WorkflowDestination; label: string }> = [
  { id: "home", label: "Home" },
  { id: "runs", label: "Runs" },
  { id: "knowledge", label: "Knowledge" },
  { id: "changes", label: "Changes" },
];

export function ProductShell({ destination, workflow, workspacePath, runtimeLabel, buildLabel, buildTitle, workspaceValid, children, onDestinationChange, onAsk, onDiagnostics, onRefresh }: ProductShellProps) {
  const [navCollapsed, setNavCollapsed] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  return (
    <main className="product-shell" data-testid="product-shell">
      <header className="product-header" data-testid="top-status-bar">
        <div><strong>ACP</strong><span title={workspacePath}>{workspacePath}</span><span>{workspaceValid ? "workspace valid" : "workspace needs attention"}</span><span>{runtimeLabel}</span><span data-testid="brand-version" title={buildTitle}>{buildLabel}</span></div>
        <div className="product-utilities">
          <button type="button" data-testid="stage-ask" onClick={onAsk}>Ask</button>
          <button type="button" data-testid="console-refresh-btn" onClick={onRefresh}>Refresh</button>
          <a href={destinationPaths.setup} data-testid="setup-utility" aria-current={destination === "setup" ? "page" : undefined} onClick={(event) => { event.preventDefault(); onDestinationChange("setup"); }}>Setup</a>
          <button type="button" aria-expanded={drawerOpen} onClick={() => setDrawerOpen(true)}>Context</button>
        </div>
      </header>
      <aside className={`workflow-banner is-${workflow.status}`} aria-live="polite">
        <strong>{workflow.status.replace("_", " ")}</strong><span>{workflow.attention}</span>
      </aside>
      <div className={`product-layout ${navCollapsed ? "nav-collapsed" : ""}`}>
        <nav className="primary-nav" aria-label="Primary">
          <button className="nav-collapse" type="button" aria-label={navCollapsed ? "Expand navigation" : "Collapse navigation"} onClick={() => setNavCollapsed((value) => !value)}>{navCollapsed ? "›" : "‹"}</button>
          {destinations.map((item) => <a key={item.id} href={destinationPaths[item.id]} data-testid={`destination-${item.id}`} aria-current={destination === item.id ? "page" : undefined} onClick={(event) => { event.preventDefault(); onDestinationChange(item.id); }}><span aria-hidden="true">{item.label.slice(0, 1)}</span><span className="nav-label">{item.label}</span></a>)}
        </nav>
        <div className="product-content">{children}</div>
        <ContextDrawer open={drawerOpen} title="Workspace context" description="Diagnostics are secondary to workflow acceptance." onClose={() => setDrawerOpen(false)}><p><strong>{runtimeLabel}</strong></p><p>{workflow.attention}</p><button type="button" onClick={() => { setDrawerOpen(false); onDiagnostics(); }}>Open Runs diagnostics</button></ContextDrawer>
      </div>
    </main>
  );
}
