import { useState, type ReactNode } from "react";

import { Button, PageHeader } from "./SemanticPrimitives";

export function SettingsPage({ workspacePath, workspaceValid, runtimeLabel, runtimeSettingsPanel, onOpenSetup, onOpenChanges, onOpenRuns }: {
  workspacePath: string;
  workspaceValid: boolean;
  runtimeLabel: string;
  runtimeSettingsPanel: ReactNode;
  onOpenSetup: () => void;
  onOpenChanges: () => void;
  onOpenRuns: () => void;
}) {
  const [activeSection, setActiveSection] = useState("settings-workspace");
  const sections = [
    ["settings-workspace", "Workspace"],
    ["settings-repositories", "Repositories"],
    ["settings-runtime", "Analysis runtime"],
    ["settings-scope", "Scope & rules"],
    ["settings-git", "Git & publication"],
    ["settings-appearance", "Appearance"],
    ["settings-diagnostics", "Diagnostics"],
  ] as const;
  const focusSection = (id: string) => {
    setActiveSection(id);
    document.getElementById(id)?.scrollIntoView?.({ block: "start" });
  };
  return (
    <section className="settings-page" data-testid="settings-page">
      <PageHeader title="Settings" purpose="Review persisted workspace configuration. Edit workspace and analysis values in Setup before the next run." action={<Button tone="primary" onClick={onOpenSetup}>Edit in Setup</Button>} />
      <section className="settings-shell" aria-label="Workspace settings">
        <aside className="settings-nav-panel" aria-label="Settings sections">
          {sections.map(([id, label]) => <button type="button" className={`settings-nav-item ${activeSection === id ? 'is-active' : ''}`} key={id} aria-current={activeSection === id ? 'page' : undefined} aria-controls={id} onClick={() => focusSection(id)}>{label}</button>)}
        </aside>
        <div className="settings-main">
          <div className="settings-main-heading"><div><p className="eyebrow">Workspace</p><h2>Workspace settings</h2><p className="hint">Identity, storage and default behavior</p></div><span className={`status ${workspaceValid ? "ok" : "err"}`}>{workspaceValid ? "Valid" : "Needs validation"}</span></div>
          <div className="settings-form-grid">
            <section className="settings-card settings-card-primary" id="settings-workspace" data-testid="settings-workspace">
              <p className="field-label">Workspace directory</p>
              <p className="settings-value" title={workspacePath}>{workspacePath}</p>
              <p className="hint">Changing location requires an explicit migration; it never silently moves files.</p>
              <div className="settings-summary"><div><span>Repositories</span><strong>Connected sources</strong></div><div><span>Runtime</span><strong>{runtimeLabel || "Not configured"}</strong></div><div><span>Publication</span><strong>Manual review</strong></div></div>
              <button type="button" className="link-button" onClick={onOpenSetup}>Manage workspace and repositories</button>
            </section>
            <section className="settings-card" id="settings-repositories" data-testid="settings-repositories"><p className="eyebrow">Repositories</p><h3>Analysis sources</h3><p>Source repositories are read-only inputs. Include/exclude rules are applied from the workspace manifest.</p><button type="button" className="link-button" onClick={onOpenSetup}>Edit sources and scope</button></section>
            <section className="settings-card" id="settings-scope" data-testid="settings-scope"><p className="eyebrow">Scope and rules</p><h3>Analysis brief</h3><p>Project scope, non-functional requirements and operating rules are edited in Setup before a run.</p><button type="button" className="link-button" onClick={onOpenSetup}>Edit analysis brief</button></section>
            <section className="settings-card" id="settings-git" data-testid="settings-git"><p className="eyebrow">Git and publication</p><h3>Workspace-wide mutations</h3><p>Publish uses the authoritative full workspace Git inventory with branch, HEAD and fingerprint confirmation.</p><button type="button" className="link-button" onClick={onOpenChanges}>Open Changes and Publish</button></section>
          </div>
          <section className="settings-card settings-runtime-card" id="settings-runtime" data-testid="settings-runtime"><div className="settings-card-heading"><div><p className="eyebrow">Analysis runtime</p><h3>{runtimeLabel || "Runtime profile"}</h3><p className="hint">Choose a runner profile for the next admission. Existing Attempts keep their immutable snapshot.</p></div><span className="status info">Persisted profile</span></div><aside className="settings-runtime-boundary"><strong>Runner settings are draft configuration for the next admission.</strong><span>Provider readiness is checked in Setup. A provider outage blocks a new Ask or Attempt, while existing evidence remains available for review.</span></aside><div className="settings-runtime-actions"><Button tone="neutral" onClick={onOpenSetup}>Check readiness in Setup</Button><Button onClick={onOpenRuns}>Open runtime diagnostics</Button></div><div className="runner-presets" data-testid="runner-presets"><div className="runner-preset-card is-current"><span className="runner-preset-mark runner-preset-mark-demo" aria-hidden="true" /><div><strong>Deterministic demo</strong><span>Fast local baseline</span></div><em>{runtimeLabel?.toLocaleLowerCase().includes("demo") ? "Current" : "Available"}</em></div><div className="runner-preset-card"><span className="runner-preset-mark">CC</span><div><strong>Claude Code</strong><span>Balanced headless runner</span></div><em>Provider</em></div><div className="runner-preset-card"><span className="runner-preset-mark">Q</span><div><strong>Qwen Code</strong><span>Alternative headless runner</span></div><em>Provider</em></div><div className="runner-preset-card"><span className="runner-preset-mark">CX</span><div><strong>Codex</strong><span>Release peer</span></div><em>Provider</em></div></div><details className="settings-runtime-advanced" open><summary>Advanced runtime controls</summary><p className="hint">Timeouts, execution policy, permissions and provider models.</p>{runtimeSettingsPanel}</details></section>
          <div className="settings-secondary-grid"><section className="settings-card" id="settings-appearance" data-testid="settings-appearance"><p className="eyebrow">Appearance</p><h3>Product defaults</h3><p>Keyboard focus and reduced-motion preferences follow local console defaults.</p></section><section className="settings-card" id="settings-diagnostics" data-testid="settings-diagnostics"><p className="eyebrow">Diagnostics</p><h3>Readiness and recovery</h3><p>Provider checks, permission recovery and run logs remain available from runtime diagnostics.</p><button type="button" className="link-button" onClick={onOpenRuns}>Open runtime diagnostics</button></section></div>
        </div>
      </section>
      <section className="settings-danger-zone" aria-label="Workspace lifecycle"><div><p className="eyebrow">Workspace lifecycle</p><h2>Export, migrate or reinitialize</h2><p className="hint">These guarded actions stay separate from everyday configuration.</p></div><Button onClick={onOpenSetup}>Manage workspace</Button></section>
    </section>
  );
}
