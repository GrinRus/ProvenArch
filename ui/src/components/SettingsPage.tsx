import { useEffect, type ReactNode } from "react";

import { Button, PageHeader } from "./SemanticPrimitives";
import type { SettingsSection } from "../lib/appRoutes";

export function SettingsPage({ activeSection, workspacePath, workspaceValid, runtimeLabel, setupRuntime, setupRuntimeProvider, runtimeSettingsPanel, onOpenSetup, onOpenChanges, onOpenRuns, onRuntimeChange, onRuntimeProviderChange, onSaveRuntime, busy, onSectionChange }: {
  activeSection: SettingsSection;
  workspacePath: string;
  workspaceValid: boolean;
  runtimeLabel: string;
  setupRuntime: string;
  setupRuntimeProvider: string;
  runtimeSettingsPanel: ReactNode;
  onOpenSetup: () => void;
  onOpenChanges: () => void;
  onOpenRuns: () => void;
  onRuntimeChange: (value: string) => void;
  onRuntimeProviderChange: (value: string) => void;
  onSaveRuntime: () => void;
  busy: boolean;
  onSectionChange: (section: SettingsSection) => void;
}) {
  const sections: Array<[SettingsSection, string, string]> = [
    ["workspace", "Workspace", "settings-workspace"],
    ["sources", "Sources", "settings-repositories"],
    ["runners", "Runners", "settings-runtime"],
    ["runtime", "Runtime", "settings-runtime"],
    ["git", "Git & publication", "settings-git"],
    ["diagnostics", "Diagnostics", "settings-diagnostics"],
  ] as const;
  const sectionTarget = sectionTargetID(activeSection);
  useEffect(() => {
    if (!sectionTarget) return;
    const timer = window.setTimeout(() => document.getElementById(sectionTarget)?.scrollIntoView?.({ block: "start" }), 0);
    return () => window.clearTimeout(timer);
  }, [sectionTarget]);
  const focusSection = (section: SettingsSection, id: string) => {
    onSectionChange(section);
    document.getElementById(id)?.scrollIntoView?.({ block: "start" });
  };
  return (
    <section className="settings-page" data-testid="settings-page">
      <PageHeader title="Settings" purpose="Review persisted workspace configuration. Edit sources and runner settings in Setup before creating the next Task." action={<Button tone="primary" onClick={onOpenSetup}>Edit in Setup</Button>} />
      <section className="settings-shell" aria-label="Workspace settings">
        <aside className="settings-nav-panel" aria-label="Settings sections">
          {sections.map(([section, label, id]) => <button type="button" className={`settings-nav-item ${activeSection === section ? 'is-active' : ''}`} key={section} aria-current={activeSection === section ? 'page' : undefined} aria-controls={id} onClick={() => focusSection(section, id)}>{label}</button>)}
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
            <section className="settings-card" id="settings-git" data-testid="settings-git"><p className="eyebrow">Git and publication</p><h3>Workspace-wide mutations</h3><p>Publish uses the authoritative full workspace Git inventory with branch, HEAD and fingerprint confirmation.</p><button type="button" className="link-button" onClick={onOpenChanges}>Open Changes and Publish</button></section>
          </div>
          <section className="settings-card settings-runtime-card" id="settings-runtime" data-testid="settings-runtime"><div className="settings-card-heading"><div><p className="eyebrow">Analysis runtime</p><h3>{runtimeLabel || "Runtime profile"}</h3><p className="hint">Choose a runner profile for the next admission. Existing Attempts keep their immutable snapshot.</p></div><span className="status info">Persisted profile</span></div><aside className="settings-runtime-boundary"><strong>Runner Settings are draft configuration for the next admission.</strong><span>Provider readiness is checked in Setup. A provider outage blocks a new Ask or Attempt, while existing evidence remains available for review.</span></aside><div className="settings-runtime-actions"><Button tone="neutral" onClick={onOpenSetup}>Check readiness in Setup</Button><Button onClick={onOpenRuns}>Open runtime diagnostics</Button></div><div className="runner-presets" data-testid="runner-presets" role="group" aria-label="Runner presets"><RunnerPresetButton selected={setupRuntime === "fake"} mark="demo" title="Deterministic demo" description="Fast local baseline" onClick={() => onRuntimeChange("fake")} /></div><div className="runner-presets" data-testid="runner-provider-presets" role="group" aria-label="Headless runner providers"><RunnerPresetButton selected={setupRuntime === "headless" && setupRuntimeProvider === "claude-code"} mark="CC" title="Claude Code" description="Balanced headless runner" onClick={() => { onRuntimeChange("headless"); onRuntimeProviderChange("claude-code"); }} /><RunnerPresetButton selected={setupRuntime === "headless" && setupRuntimeProvider === "qwen-code"} mark="Q" title="Qwen Code" description="Alternative headless runner" onClick={() => { onRuntimeChange("headless"); onRuntimeProviderChange("qwen-code"); }} /><RunnerPresetButton selected={setupRuntime === "headless" && setupRuntimeProvider === "codex-code"} mark="CX" title="Codex" description="Release peer" onClick={() => { onRuntimeChange("headless"); onRuntimeProviderChange("codex-code"); }} /></div><div className="settings-runtime-save"><span className="hint">Selected: {setupRuntime === "headless" ? setupRuntimeProvider : "deterministic demo"}. Save before creating the next Task.</span><Button tone="primary" onClick={onSaveRuntime} disabled={busy || !workspaceValid} data-testid="settings-runtime-save">Save runner preset</Button></div><details className="settings-runtime-advanced"><summary>Advanced runtime controls</summary><p className="hint">Timeouts, execution policy, permissions and provider models.</p>{runtimeSettingsPanel}</details></section>
          <div className="settings-secondary-grid"><section className="settings-card" id="settings-appearance" data-testid="settings-appearance"><p className="eyebrow">Appearance</p><h3>Product defaults</h3><p>Keyboard focus and reduced-motion preferences follow local console defaults.</p></section><section className="settings-card" id="settings-diagnostics" data-testid="settings-diagnostics"><p className="eyebrow">Diagnostics</p><h3>Readiness and recovery</h3><p>Provider checks, permission recovery and run logs remain available from runtime diagnostics.</p><button type="button" className="link-button" onClick={onOpenRuns}>Open runtime diagnostics</button></section></div>
        </div>
      </section>
      <section className="settings-danger-zone" aria-label="Workspace lifecycle"><div><p className="eyebrow">Workspace lifecycle</p><h2>Export, migrate or reinitialize</h2><p className="hint">These guarded actions stay separate from everyday configuration.</p></div><Button onClick={onOpenSetup}>Manage workspace</Button></section>
    </section>
  );
}

function sectionTargetID(section: SettingsSection): string {
  if (section === "sources") return "settings-repositories";
  if (section === "git") return "settings-git";
  if (section === "diagnostics") return "settings-diagnostics";
  return section === "workspace" ? "settings-workspace" : "settings-runtime";
}

function RunnerPresetButton({ selected, mark, title, description, onClick }: { selected: boolean; mark: string; title: string; description: string; onClick: () => void }) {
  return <button type="button" className={`runner-preset-card ${selected ? "is-current" : ""}`} aria-pressed={selected} onClick={onClick}>
    <span className={`runner-preset-mark ${mark === "demo" ? "runner-preset-mark-demo" : ""}`} aria-hidden="true">{mark === "demo" ? "" : mark}</span>
    <span><strong>{title}</strong><span>{description}</span></span>
    <em>{selected ? "Selected" : "Choose"}</em>
  </button>;
}
