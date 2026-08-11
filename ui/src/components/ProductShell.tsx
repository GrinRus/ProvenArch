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

const workspaceDestinations: Array<{ id: WorkflowDestination; label: string; icon: string }> = [
  { id: "tasks", label: "Tasks", icon: "▣" },
  { id: "knowledge", label: "Architecture", icon: "◇" },
  { id: "changes", label: "Changes", icon: "Δ" },
];

export function ProductShell({ destination, workflow, workspacePath, runtimeLabel, buildLabel, buildTitle, workspaceValid, children, onDestinationChange, onAsk, onDiagnostics, onRefresh }: ProductShellProps) {
  const [navCollapsed, setNavCollapsed] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  return (
    <main className="product-shell" data-testid="product-shell">
      <header className="product-header" data-testid="top-status-bar">
        <div className="product-identity">
          <span className="product-logo" aria-hidden="true">PA</span>
          <strong>ProvenArch</strong>
          <span className="product-divider" aria-hidden="true" />
          <span className="product-workspace" title={workspacePath}>{workspaceLabel(workspacePath)}</span>
          <span className={workspaceValid ? "product-health is-ready" : "product-health is-warning"}>{workspaceValid ? "Workspace ready" : "Workspace needs attention"}</span>
          <span className="product-runtime">{runtimeLabel}</span>
        </div>
        <div className="product-utilities">
          <button className="product-ask" type="button" data-testid="stage-ask" onClick={onAsk}><span aria-hidden="true">⌕</span><span>Ask about this architecture</span></button>
          <button className="product-icon-button" type="button" data-testid="console-refresh-btn" aria-label="Refresh workspace data" onClick={onRefresh}>↻</button>
          <a className="product-icon-button" href={destinationPaths.settings} data-testid="settings-utility" aria-label="Workspace configuration" aria-current={destination === "settings" ? "page" : undefined} onClick={(event) => { event.preventDefault(); onDestinationChange("settings"); }}>⚙</a>
          <a className="product-icon-button" href={destinationPaths.setup} data-testid="setup-utility" aria-label="Lifecycle menu" aria-current={destination === "setup" ? "page" : undefined} onClick={(event) => { event.preventDefault(); onDestinationChange("setup"); }}>⋯</a>
          <button className="product-icon-button" type="button" aria-label="Details" aria-expanded={drawerOpen} onClick={() => setDrawerOpen(true)}>?</button>
        </div>
      </header>
      <div className={`product-layout ${navCollapsed ? "nav-collapsed" : ""}`}>
          <nav className="primary-nav" aria-label="Primary">
            <button className="nav-collapse" type="button" aria-label={navCollapsed ? "Expand navigation" : "Collapse navigation"} onClick={() => setNavCollapsed((value) => !value)}>{navCollapsed ? "›" : "‹"}</button>
          <p className="nav-section-label">Workspace</p>
          {workspaceDestinations.map((item) => <a key={item.id} href={destinationPaths[item.id]} data-testid={`destination-${item.id}`} aria-current={destination === item.id ? "page" : undefined} onClick={(event) => { event.preventDefault(); onDestinationChange(item.id); }}><span aria-hidden="true">{item.icon}</span><span className="nav-label">{item.label}</span></a>)}
          <p className="nav-section-label nav-section-label-utilities">Utilities</p>
          <a href={destinationPaths.settings} data-testid="settings-nav" aria-current={destination === "settings" ? "page" : undefined} onClick={(event) => { event.preventDefault(); onDestinationChange("settings"); }}><span aria-hidden="true">⚙</span><span className="nav-label">Settings</span></a>
          <a href={destinationPaths.setup} data-testid="setup-nav" aria-current={destination === "setup" ? "page" : undefined} onClick={(event) => { event.preventDefault(); onDestinationChange("setup"); }}><span aria-hidden="true">↗</span><span className="nav-label">Setup</span></a>
          <button type="button" data-testid="diagnostics-nav" onClick={onDiagnostics}><span aria-hidden="true">↗</span><span className="nav-label">Diagnostics</span></button>
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
          <button type="button" onClick={() => { setDrawerOpen(false); onDiagnostics(); }}>Open runtime diagnostics</button>
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
