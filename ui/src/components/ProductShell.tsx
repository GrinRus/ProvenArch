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
        <div className="product-identity">
          <strong>ProvenArch</strong>
          <span className="product-workspace" title={workspacePath}>{workspaceLabel(workspacePath)}</span>
          <span className={workspaceValid ? "product-health is-ready" : "product-health is-warning"}>{workspaceValid ? "Workspace ready" : "Workspace needs attention"}</span>
          <span className="product-runtime">{runtimeLabel}</span>
        </div>
        <div className="product-utilities">
          <button type="button" data-testid="stage-ask" onClick={onAsk}>Ask workspace</button>
          <button type="button" data-testid="console-refresh-btn" onClick={onRefresh}>Refresh data</button>
          <a href={destinationPaths.setup} data-testid="setup-utility" aria-current={destination === "setup" ? "page" : undefined} onClick={(event) => { event.preventDefault(); onDestinationChange("setup"); }}>Setup</a>
          <button type="button" aria-expanded={drawerOpen} onClick={() => setDrawerOpen(true)}>Details</button>
        </div>
      </header>
      <div className={`product-layout ${navCollapsed ? "nav-collapsed" : ""}`}>
        <nav className="primary-nav" aria-label="Primary">
          <button className="nav-collapse" type="button" aria-label={navCollapsed ? "Expand navigation" : "Collapse navigation"} onClick={() => setNavCollapsed((value) => !value)}>{navCollapsed ? "›" : "‹"}</button>
          {destinations.map((item) => <a key={item.id} href={destinationPaths[item.id]} data-testid={`destination-${item.id}`} aria-current={destination === item.id ? "page" : undefined} onClick={(event) => { event.preventDefault(); onDestinationChange(item.id); }}><span aria-hidden="true">{item.label.slice(0, 1)}</span><span className="nav-label">{item.label}</span></a>)}
        </nav>
        <div className="product-content">{children}</div>
        <ContextDrawer open={drawerOpen} title="Workspace details" description="Runtime, build and diagnostic context for this local session." onClose={() => setDrawerOpen(false)}>
          <dl className="compact-defs">
            <div><dt>Workspace</dt><dd title={workspacePath}>{workspacePath}</dd></div>
            <div><dt>Runtime</dt><dd>{runtimeLabel}</dd></div>
            <div><dt>Build</dt><dd data-testid="brand-version" title={buildTitle}>{buildLabel}</dd></div>
          </dl>
          <p>{workflow.attention}</p>
          <div className="context-drawer-actions">
            <button type="button" onClick={() => { setDrawerOpen(false); onRefresh(); }}>Refresh workspace data</button>
            <button type="button" onClick={() => { setDrawerOpen(false); onDestinationChange("setup"); }}>Open Setup</button>
          </div>
          <button type="button" onClick={() => { setDrawerOpen(false); onDiagnostics(); }}>Open Runs diagnostics</button>
        </ContextDrawer>
      </div>
    </main>
  );
}

function workspaceLabel(path: string): string {
  const normalized = path.trim().replace(/\\/g, "/").replace(/\/+$/, "");
  const parts = normalized.split("/").filter(Boolean);
  return parts[parts.length - 1] || "Workspace";
}
