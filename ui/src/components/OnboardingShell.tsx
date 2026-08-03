import { useState, type ReactNode } from "react";

import type { Diagnostic, DoctorResponse, GuidedRepo, OnboardingRecentWorkspace, OnboardingStatusResponse, RepoSourceMode, ValidateResponse } from "../lib/appContracts";
import { providerCommandEnv, providerCommandHint, providerReadinessGuidance } from "../lib/providerGuidance";
import { AsyncStatusMessage } from "./AccessibleStatus";
import { StatusBadge } from "./ConsolePrimitives";
import { LocalPathCombobox } from "./LocalPathCombobox";
import { RepoAnalysisScopeFields } from "./RepoAnalysisScopeFields";

type OnboardingStep = "workspace" | "sources" | "brief" | "runner" | "review";

type OnboardingShellProps = {
  busy: boolean;
  error: string | null;
  status: OnboardingStatusResponse | null;
  workspacePath: string;
  createWorkspace: boolean;
  guidedRepos: GuidedRepo[];
  guidedDocsImportsPath: string;
  validateResult: ValidateResponse | null;
  doctorResult: DoctorResponse | null;
  setupRuntime: string;
  setupRuntimeProvider: string;
  briefReady?: boolean;
  briefPanel?: ReactNode;
  firstRunStatus: string;
  onWorkspacePathChange: (value: string) => void;
  onCreateWorkspaceChange: (value: boolean) => void;
  onSelectWorkspace: () => void;
  onOpenRecentWorkspace: (path: string) => void;
  onForgetRecentWorkspace: (path: string) => void;
  onRepoChange: (id: string, patch: Partial<GuidedRepo>) => void;
  onAddRepo: () => void;
  onRemoveRepo: (id: string) => void;
  onDocsImportsPathChange: (value: string) => void;
  onSaveSources: () => void;
  onRuntimeChange: (value: string) => void;
  onRuntimeProviderChange: (value: string) => void;
  onSaveRuntime: () => void;
  onCheckDoctor: () => void;
  onEnterConsole: () => void;
  onRunFirstAnalysis: () => void;
};

export function OnboardingShell({
  busy,
  error,
  status,
  workspacePath,
  createWorkspace,
  guidedRepos,
  guidedDocsImportsPath,
  validateResult,
  doctorResult,
  setupRuntime,
  setupRuntimeProvider,
  briefReady = false,
  briefPanel = null,
  firstRunStatus,
  onWorkspacePathChange,
  onCreateWorkspaceChange,
  onSelectWorkspace,
  onOpenRecentWorkspace,
  onForgetRecentWorkspace,
  onRepoChange,
  onAddRepo,
  onRemoveRepo,
  onDocsImportsPathChange,
  onSaveSources,
  onRuntimeChange,
  onRuntimeProviderChange,
  onSaveRuntime,
  onCheckDoctor,
  onEnterConsole,
  onRunFirstAnalysis,
}: OnboardingShellProps) {
  const [activeStep, setActiveStep] = useState<OnboardingStep>(() => initialOnboardingStep(status, validateResult, briefReady, doctorResult));
  const workspaceReady = status?.workspace_selected ?? false;
  const recentWorkspaces = status?.recent_workspaces ?? [];
  const repoDiagnosticsByID = buildRepoDiagnostics(guidedRepos, validateResult);
  const hasRepoDraftErrors = Array.from(repoDiagnosticsByID.values()).some((diagnostics) => diagnostics.some((diagnostic) => diagnostic.level === "error"));
  const sourcesReady = validateResult?.ok === true && !hasRepoDraftErrors;
  const runtimeReady = status?.runtime.selected === true;
  const localReady = doctorResult?.ok === true;
  const canEnterConsole = status?.can_enter_console === true && sourcesReady && runtimeReady;
  const canRunFirstAnalysis = canEnterConsole && localReady;
  const sourceBlockingDiagnostic = findSourceBlockingDiagnostic(repoDiagnosticsByID, validateResult);
  const doctorFailures = doctorResult?.checks.filter((check) => check.status === "fail") ?? [];
  const runtimeProviderCheck = doctorResult?.checks.find((check) => check.id === "runtime_provider");
  const showRunnerRecovery = setupRuntime === "headless" || runtimeProviderCheck?.status === "fail";
  const progressSummary = buildOnboardingProgressSummary({
    workspaceReady,
    sourcesReady,
    runtimeReady,
    localReady,
    briefReady,
    canEnterConsole,
    canRunFirstAnalysis,
    workspacePath: status?.workspace ?? "",
    guidedRepoCount: guidedRepos.length,
    setupRuntime,
    setupRuntimeProvider,
    sourceBlockingDiagnostic,
    doctorResult,
    firstRunStatus,
  });
  const openConsoleDisabledReason = canEnterConsole
    ? ""
    : getOpenConsoleDisabledReason({
        workspaceReady,
        sourcesReady,
        runtimeReady,
        sourceBlockingDiagnostic,
        validateResult,
      });
  const firstAnalysisDisabledReason = canRunFirstAnalysis
    ? ""
    : getFirstAnalysisDisabledReason({
        canEnterConsole,
        localReady,
        openConsoleDisabledReason,
        doctorFailures,
        doctorResult,
      });

  return (
    <main className="onboarding-shell" data-testid="onboarding-shell">
      <section className="onboarding-panel">
        <div className="onboarding-header">
          <div>
            <p className="eyebrow">ProvenArch local console</p>
            <h1>Set up your architecture workspace</h1>
            <p className="hint">Connect read-only repositories, describe the analysis, choose a runner and review everything before the first run.</p>
          </div>
          <div className="onboarding-status-stack" aria-label="onboarding status">
            <StatusPill label="Workspace" ready={workspaceReady} />
            <StatusPill label="Repositories" ready={sourcesReady} />
            <StatusPill label="Brief" ready={briefReady} />
            <StatusPill label="Runner" ready={runtimeReady} />
          </div>
        </div>

        {error ? (
          <AsyncStatusMessage tone="error" className="error-banner">
            {error}
          </AsyncStatusMessage>
        ) : null}

        <OnboardingProgressSummaryPanel summary={progressSummary} activeStep={activeStep} onStepChange={setActiveStep} />

        <div className="onboarding-grid">
          <section className="onboarding-card" data-testid="onboarding-workspace-step" hidden={activeStep !== "workspace"}>
            <div className="card-heading">
              <span className="step-index">1</span>
              <div>
                <h2>Workspace</h2>
                <p className="hint">This is where ACP writes architecture artifacts. Source repos stay read-only inputs.</p>
              </div>
            </div>
            <LocalPathCombobox
              id="onboardingWorkspacePath"
              label="Architecture workspace path"
              kind="workspace"
              value={workspacePath}
              placeholder="/Users/me/acp-workspaces/my-system"
              testID="onboarding-workspace-path-combobox"
              onChange={onWorkspacePathChange}
            />
            <label className="checkbox-row">
              <input type="checkbox" checked={createWorkspace} onChange={(event) => onCreateWorkspaceChange(event.target.checked)} />
              <span>Create path if missing</span>
            </label>
            <button type="button" onClick={onSelectWorkspace} disabled={busy || !workspacePath.trim()} data-testid="onboarding-workspace-save">
              {createWorkspace ? "Create workspace" : "Open workspace"}
            </button>
            {status?.workspace_selected ? (
              <AsyncStatusMessage tone="success" className="status">
                Selected: {status.workspace}
              </AsyncStatusMessage>
            ) : null}
            <RecentWorkspacesList recentWorkspaces={recentWorkspaces} busy={busy} onOpen={onOpenRecentWorkspace} onForget={onForgetRecentWorkspace} />
            {workspaceReady ? <button type="button" className="onboarding-continue" onClick={() => setActiveStep("sources")}>Continue to repositories</button> : null}
          </section>

          <section className="onboarding-card onboarding-card-sources" data-testid="onboarding-sources-step" hidden={activeStep !== "sources"}>
            <div className="card-heading">
              <span className="step-index">2</span>
              <div>
                <h2>Repositories</h2>
                <p className="hint">Add one or more read-only repositories. A local checkout is the fastest path; Git URLs use your local Git credentials.</p>
              </div>
            </div>
            {guidedRepos.map((repo, index) => {
              const diagnostics = repoDiagnosticsByID.get(repo.id) ?? [];
              const diagnosticsID = repoDiagnosticsID(repo.id);
              const nameInvalid = hasRepoNameError(diagnostics);
              const sourceInvalid = hasRepoSourceError(diagnostics);
              return (
                <div className="onboarding-repo-row" key={repo.id}>
                  <div className="repo-row-title">
                    <strong>Repo {index + 1}</strong>
                    <button type="button" className="inline-danger" onClick={() => onRemoveRepo(repo.id)} disabled={busy || guidedRepos.length <= 1}>
                      Remove
                    </button>
                  </div>
                  <div className="repo-field-grid">
                    <div className="field">
                      <label htmlFor={`onboardingRepoName-${repo.id}`}>Name</label>
                      <input
                        id={`onboardingRepoName-${repo.id}`}
                        value={repo.name}
                        aria-invalid={nameInvalid || undefined}
                        aria-describedby={nameInvalid ? diagnosticsID : undefined}
                        onChange={(event) => onRepoChange(repo.id, { name: event.target.value })}
                      />
                    </div>
                    <div className="field">
                      <label htmlFor={`onboardingRepoMode-${repo.id}`}>Source type</label>
                      <select
                        id={`onboardingRepoMode-${repo.id}`}
                        value={repo.mode}
                        onChange={(event) => onRepoChange(repo.id, { mode: event.target.value as RepoSourceMode })}
                      >
                        <option value="git_url">GitHub/GitLab URL</option>
                        <option value="path">Local folder</option>
                      </select>
                    </div>
                    {repo.mode === "path" ? (
                      <LocalPathCombobox
                        id={`onboardingRepoSource-${repo.id}`}
                        label="Local checkout path"
                        kind="repo"
                        value={repo.path}
                        invalid={sourceInvalid}
                        describedBy={sourceInvalid ? diagnosticsID : undefined}
                        testID={`onboarding-repo-path-combobox-${repo.id}`}
                        onChange={(value) => onRepoChange(repo.id, { path: value })}
                        onSelect={(suggestion) => {
                          const patch: Partial<GuidedRepo> = { path: suggestion.path };
                          if (!repo.name.trim()) {
                            patch.name = repoNameFromPath(suggestion.path);
                          }
                          onRepoChange(repo.id, patch);
                        }}
                      />
                    ) : (
                      <div className="field is-wide">
                        <label htmlFor={`onboardingRepoSource-${repo.id}`}>Repository URL</label>
                        <input
                          id={`onboardingRepoSource-${repo.id}`}
                          value={repo.git_url}
                          aria-invalid={sourceInvalid || undefined}
                          aria-describedby={sourceInvalid ? diagnosticsID : undefined}
                          onChange={(event) => onRepoChange(repo.id, { git_url: event.target.value })}
                        />
                      </div>
                    )}
                  </div>
                  <details className="onboarding-advanced" open={Boolean(repo.ref || repo.analysis_include || repo.analysis_exclude)}>
                    <summary>Advanced scope</summary>
                    <div className="field">
                      <label htmlFor={`onboardingRepoRef-${repo.id}`}>Git ref (optional)</label>
                      <input id={`onboardingRepoRef-${repo.id}`} value={repo.ref} onChange={(event) => onRepoChange(repo.id, { ref: event.target.value })} />
                    </div>
                    <RepoAnalysisScopeFields
                      repoId={`onboarding-${repo.id}`}
                      include={repo.analysis_include}
                      exclude={repo.analysis_exclude}
                      onIncludeChange={(value) => onRepoChange(repo.id, { analysis_include: value })}
                      onExcludeChange={(value) => onRepoChange(repo.id, { analysis_exclude: value })}
                    />
                  </details>
                  <RepoDiagnostics id={diagnosticsID} diagnostics={diagnostics} />
                </div>
              );
            })}
            <div className="field">
              <label htmlFor="onboardingDocsImportsPath">docs.imports_path</label>
              <input id="onboardingDocsImportsPath" value={guidedDocsImportsPath} onChange={(event) => onDocsImportsPathChange(event.target.value)} />
            </div>
            <div className="actions">
              <button type="button" onClick={onAddRepo} disabled={busy || !workspaceReady}>
                Add repo
              </button>
              <button type="button" onClick={onSaveSources} disabled={busy || !workspaceReady || hasRepoDraftErrors} data-testid="onboarding-sources-save">
                Save and validate sources
              </button>
            </div>
            {validateResult ? (
              <AsyncStatusMessage tone={validateResult.ok ? "success" : "error"} className={validateResult.ok ? "status" : "error-text"}>
                {validateResult.ok ? "Sources validated." : "Sources need fixes."}
              </AsyncStatusMessage>
            ) : null}
            {sourcesReady ? <button type="button" className="onboarding-continue" onClick={() => setActiveStep("brief")}>Continue to analysis brief</button> : null}
          </section>

          <section className="onboarding-card" data-testid="onboarding-brief-step" hidden={activeStep !== "brief"}>
            <div className="card-heading">
              <span className="step-index">3</span>
              <div>
                <h2>Analysis brief</h2>
                <p className="hint">Describe the system, scope and quality priorities so the result is useful for architectural decisions.</p>
              </div>
            </div>
            {briefPanel}
            {!briefReady ? <p className="status warn">You can continue without a brief, but the first result may be less decision-ready.</p> : <p className="status ok">Analysis brief saved.</p>}
            <button type="button" className="onboarding-continue" onClick={() => setActiveStep("runner")}>{briefReady ? "Continue to runner" : "Continue without brief"}</button>
          </section>

          <section className="onboarding-card" data-testid="onboarding-runner-step" hidden={activeStep !== "runner"}>
            <div className="card-heading">
              <span className="step-index">4</span>
              <div>
                <h2>Runner</h2>
                <p className="hint">Choose a deterministic walkthrough or explicitly opt in to live architecture analysis.</p>
              </div>
            </div>
            <div className="field">
              <label htmlFor="onboardingRuntime">Runtime</label>
              <select id="onboardingRuntime" value={setupRuntime} onChange={(event) => onRuntimeChange(event.target.value)}>
                <option value="fake">Deterministic walkthrough (recommended)</option>
                <option value="headless">Live architecture analysis</option>
              </select>
            </div>
            <div className="field">
              <label htmlFor="onboardingRuntimeProvider">Provider</label>
              <select id="onboardingRuntimeProvider" value={setupRuntimeProvider} disabled={setupRuntime !== "headless"} onChange={(event) => onRuntimeProviderChange(event.target.value)}>
                <option value="claude-code">Claude Code (claude-code)</option>
                <option value="qwen-code">Qwen Code (qwen-code)</option>
                <option value="codex-code">Codex (codex-code)</option>
              </select>
              {setupRuntime !== "headless" ? <p className="hint">No live provider is used during the deterministic walkthrough.</p> : null}
            </div>
            <div className="actions">
              <button type="button" onClick={onSaveRuntime} disabled={busy || !workspaceReady} data-testid="onboarding-runtime-save">
                Select runner
              </button>
              <button type="button" onClick={onCheckDoctor} disabled={busy || !workspaceReady}>
                Check readiness
              </button>
            </div>
            {doctorResult ? (
              <AsyncStatusMessage tone={doctorResult.ok ? "success" : "error"} className={doctorResult.ok ? "status" : "error-text"}>
                {doctorResult.ok ? "Runner and local readiness passed." : "Readiness has blockers."}
              </AsyncStatusMessage>
            ) : null}
            {showRunnerRecovery ? (
              <OnboardingRunnerRecovery
                setupRuntime={setupRuntime}
                setupRuntimeProvider={setupRuntimeProvider}
                runtimeProviderCheck={runtimeProviderCheck}
              />
            ) : null}
            {doctorResult ? <OnboardingDoctorChecklist doctorResult={doctorResult} /> : null}
            {runtimeReady ? <button type="button" className="onboarding-continue" onClick={() => setActiveStep("review")}>Review setup</button> : null}
          </section>

          <section className="onboarding-card onboarding-review-card" data-testid="onboarding-ready-step" hidden={activeStep !== "review"}>
            <div className="card-heading">
              <span className="step-index">5</span>
              <div>
                <h2>Review & start</h2>
                <p className="hint">Confirm what ProvenArch will read, where it will write and which runner is effective.</p>
              </div>
            </div>
            <ul className="checklist">
              <li className={workspaceReady ? "is-ready" : ""}>Workspace selected</li>
              <li className={sourcesReady ? "is-ready" : ""}>Read-only repositories validated</li>
              <li className={briefReady ? "is-ready" : ""}>{briefReady ? "Analysis brief saved" : "Analysis brief skipped — quality warning"}</li>
              <li className={runtimeReady ? "is-ready" : ""}>Runner selected</li>
              <li className={localReady ? "is-ready" : ""}>Local readiness checked</li>
            </ul>
            <div className="actions">
              <button
                type="button"
                onClick={onEnterConsole}
                disabled={busy || !canEnterConsole}
                title={!canEnterConsole ? openConsoleDisabledReason : undefined}
                data-testid="onboarding-enter-console"
              >
                Open console without running
              </button>
              <button
                type="button"
                onClick={onRunFirstAnalysis}
                disabled={busy || !canRunFirstAnalysis}
                title={!canRunFirstAnalysis ? firstAnalysisDisabledReason : undefined}
                data-testid="onboarding-run-first-analysis"
              >
                Run first analysis
              </button>
            </div>
            {!canEnterConsole || !canRunFirstAnalysis ? (
              <div className="ready-action-hint" data-testid="onboarding-ready-action-hint">
                {!canEnterConsole ? <p>Open console waits for: {openConsoleDisabledReason}</p> : null}
                {!canRunFirstAnalysis ? <p>First analysis waits for: {firstAnalysisDisabledReason}</p> : null}
              </div>
            ) : (
              <p className="status" data-testid="onboarding-ready-action-hint">
                Ready to run the first analysis.
              </p>
            )}
            {canEnterConsole && !localReady ? <p className="status warn">Check local readiness before first analysis.</p> : null}
            {firstRunStatus ? (
              <AsyncStatusMessage tone="progress" className="status">
                {firstRunStatus}
              </AsyncStatusMessage>
            ) : null}
            <div className="onboarding-boundary-note"><strong>Source boundary</strong><span>Repositories are read-only. ProvenArch writes generated knowledge only to the architecture workspace.</span></div>
          </section>
        </div>
      </section>
    </main>
  );
}

function RecentWorkspacesList({
  recentWorkspaces,
  busy,
  onOpen,
  onForget,
}: {
  recentWorkspaces: OnboardingRecentWorkspace[];
  busy: boolean;
  onOpen: (path: string) => void;
  onForget: (path: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const visibleWorkspaces = expanded ? recentWorkspaces : recentWorkspaces.slice(0, 3);
  const hiddenCount = Math.max(0, recentWorkspaces.length - visibleWorkspaces.length);

  if (recentWorkspaces.length === 0) {
    return <p className="recent-workspace-empty">No recent workspaces yet.</p>;
  }
  return (
    <div className="recent-workspace-list" data-testid="onboarding-recent-workspaces">
      <div className="recent-workspace-list-heading">
        <strong>Recent workspaces</strong>
        <span>{recentWorkspaces.length}/10</span>
      </div>
      {visibleWorkspaces.map((workspace) => (
        <div className={workspace.exists ? "recent-workspace-row" : "recent-workspace-row is-missing"} key={workspace.path}>
          <div>
            <code>{workspace.path}</code>
            <span>
              {workspace.exists ? "available" : "missing"} · {formatRecentWorkspaceTimestamp(workspace.last_opened_at)}
            </span>
          </div>
          <div className="recent-workspace-actions">
            <button type="button" className="link-button" onClick={() => onOpen(workspace.path)} disabled={busy || !workspace.exists}>
              Open
            </button>
            <button type="button" className="inline-danger" onClick={() => onForget(workspace.path)} disabled={busy}>
              Forget
            </button>
          </div>
        </div>
      ))}
      {recentWorkspaces.length > 3 ? (
        <button type="button" className="link-button recent-workspace-toggle" onClick={() => setExpanded((value) => !value)}>
          {expanded ? "Show fewer workspaces" : `Show ${hiddenCount} more workspaces`}
        </button>
      ) : null}
    </div>
  );
}

function formatRecentWorkspaceTimestamp(value: string): string {
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) {
    return "last opened recently";
  }
  return `last opened ${new Date(parsed).toISOString().replace("T", " ").replace(".000Z", " UTC")}`;
}

function initialOnboardingStep(status: OnboardingStatusResponse | null, validateResult: ValidateResponse | null, briefReady: boolean, doctorResult: DoctorResponse | null): OnboardingStep {
  if (!status?.workspace_selected) return "workspace";
  if (validateResult?.ok !== true) return "sources";
  if (!briefReady) return "brief";
  if (status.runtime.selected !== true || doctorResult?.ok !== true) return "runner";
  return "review";
}

function StatusPill({ label, ready }: { label: string; ready: boolean }) {
  return <span className={ready ? "status-pill is-ready" : "status-pill"}>{label}</span>;
}

type OnboardingProgressItem = {
  id: OnboardingStep;
  label: string;
  detail: string;
  ready: boolean;
  available: boolean;
};

type OnboardingProgressSummary = {
  action: string;
  blocker: string;
  detail: string;
  items: OnboardingProgressItem[];
  step: string;
  tone: "blocked" | "ready" | "waiting";
};

function OnboardingProgressSummaryPanel({ summary, activeStep, onStepChange }: { summary: OnboardingProgressSummary; activeStep: OnboardingStep; onStepChange: (step: OnboardingStep) => void }) {
  return (
    <section className={`onboarding-progress-summary is-${summary.tone}`} data-testid="onboarding-progress-summary" aria-label="Onboarding setup progress">
      <div className="onboarding-progress-primary">
        <p className="eyebrow">{summary.step}</p>
        <h2>{summary.action}</h2>
        <p>{summary.detail}</p>
      </div>
      <div className="onboarding-progress-blocker">
        <span>{summary.tone === "ready" ? "Ready state" : "Current blocker"}</span>
        <strong>{summary.blocker}</strong>
      </div>
      <ol className="onboarding-progress-steps">
        {summary.items.map((item, index) => (
          <li className={`${item.ready ? "is-ready" : ""} ${activeStep === item.id ? "is-active" : ""}`.trim()} key={item.label}>
            <button type="button" data-testid={`onboarding-progress-${item.id}`} aria-current={activeStep === item.id ? "step" : undefined} disabled={!item.available} onClick={() => onStepChange(item.id)}>
              <span>{index + 1}</span>
              <div>
                <strong>{item.label}</strong>
                <p>{item.detail}</p>
              </div>
            </button>
          </li>
        ))}
      </ol>
    </section>
  );
}

function repoNameFromPath(pathValue: string): string {
  const normalized = pathValue.trim().replace(/\\/g, "/").replace(/\/+$/, "");
  const parts = normalized.split("/");
  return parts[parts.length - 1] || "repo";
}

function RepoDiagnostics({ id, diagnostics }: { id: string; diagnostics: Diagnostic[] }) {
  if (diagnostics.length === 0) {
    return null;
  }
  return (
    <div className="diagnostic-list" id={id} data-testid="onboarding-repo-diagnostics">
      {diagnostics.map((diagnostic, index) => (
        <AsyncStatusMessage
          className={diagnostic.level === "error" ? "status err" : "status warn"}
          key={`${diagnostic.code}-${diagnostic.message}-${index}`}
          tone={diagnostic.level === "error" ? "error" : "warning"}
        >
          {diagnostic.level === "error" ? "Error" : "Warning"} [{diagnostic.code}]: {diagnostic.message}
          {diagnostic.suggestion ? <span> Next: {diagnostic.suggestion}</span> : null}
        </AsyncStatusMessage>
      ))}
    </div>
  );
}

function repoDiagnosticsID(repoID: string): string {
  return `onboardingRepoDiagnostics-${repoID}`;
}

function hasRepoNameError(diagnostics: Diagnostic[]): boolean {
  return diagnostics.some((diagnostic) => diagnostic.level === "error" && diagnostic.code.startsWith("repo_name"));
}

function hasRepoSourceError(diagnostics: Diagnostic[]): boolean {
  return diagnostics.some((diagnostic) => {
    if (diagnostic.level !== "error") {
      return false;
    }
    const code = diagnostic.code.toLowerCase();
    return code.includes("repo_path") || code.includes("repo_git_url") || code.includes("repo_source") || code.includes("source");
  });
}

function OnboardingDoctorChecklist({ doctorResult }: { doctorResult: DoctorResponse }) {
  return (
    <div className="status-block compact" data-testid="onboarding-doctor-result">
      <ul className="check-list">
        {doctorResult.checks.map((check) => (
          <li className={`check ${check.status}`} key={check.id}>
            <strong>{check.label}:</strong> {check.message}
            {check.suggestion ? <span> Next: {check.suggestion}</span> : null}
          </li>
        ))}
      </ul>
    </div>
  );
}

function OnboardingRunnerRecovery({
  setupRuntime,
  setupRuntimeProvider,
  runtimeProviderCheck,
}: {
  setupRuntime: string;
  setupRuntimeProvider: string;
  runtimeProviderCheck?: DoctorResponse["checks"][number];
}) {
  const providerCommand = providerCommandHint(setupRuntimeProvider);
  const envOverride = providerCommandEnv(setupRuntimeProvider);
  const doctorStatus = runtimeProviderCheck ? `${runtimeProviderCheck.label}: ${runtimeProviderCheck.status}` : "not checked";
  const isPassing = runtimeProviderCheck?.status === "pass";
  const isFailing = runtimeProviderCheck?.status === "fail";
  const guidance = providerReadinessGuidance(setupRuntimeProvider, runtimeProviderCheck?.message || runtimeProviderCheck?.suggestion);
  const summary = isPassing
    ? "Headless provider is ready for first analysis after the other setup gates pass."
    : isFailing
      ? "Fix the provider command, authentication or quota before running the first live analysis."
      : "Headless provider is selected. Check the command and auth/quota before the first live analysis.";

  return (
    <div className="onboarding-runner-recovery" data-testid="onboarding-runner-recovery">
      <div className="section-heading-row">
        <div>
          <strong>Provider setup for first analysis</strong>
          <p className="hint">{summary}</p>
        </div>
        <StatusBadge tone={isPassing ? "ok" : isFailing ? "error" : "warn"}>{isPassing ? "provider ready" : "provider check"}</StatusBadge>
      </div>
      <div className="onboarding-runner-recovery-grid">
        <div>
          <span className="metric-label">Selected runner</span>
          <strong>{setupRuntime === "headless" ? setupRuntimeProvider : `${setupRuntime} baseline`}</strong>
        </div>
        <div>
          <span className="metric-label">Expected command</span>
          <strong>{setupRuntime === "headless" ? providerCommand : "none required"}</strong>
        </div>
        <div>
          <span className="metric-label">Command override</span>
          <strong>{setupRuntime === "headless" ? envOverride : "not needed"}</strong>
        </div>
        <div>
          <span className="metric-label">Readiness check</span>
          <strong>{doctorStatus}</strong>
        </div>
        <div>
          <span className="metric-label">Failure mode</span>
          <strong>{isPassing ? "Provider ready" : guidance.failureMode}</strong>
        </div>
        <div>
          <span className="metric-label">Probe stage</span>
          <strong>{isPassing ? "Ready" : guidance.probeStage}</strong>
        </div>
      </div>
      <dl className="compact-defs onboarding-runner-recovery-detail">
        <div>
          <dt>Doctor message</dt>
          <dd>{runtimeProviderCheck?.message ?? "Run Check readiness after selecting the runner."}</dd>
        </div>
        <div>
          <dt>Operator focus</dt>
          <dd>{isPassing ? "Provider readiness passed; continue with workspace and source readiness gates." : guidance.operatorFocus}</dd>
        </div>
        {runtimeProviderCheck?.suggestion ? (
          <div>
            <dt>Suggested fix</dt>
            <dd>{runtimeProviderCheck.suggestion}</dd>
          </div>
        ) : null}
      </dl>
      <ul className="analysis-next-actions">
        <li>Use fake baseline for a deterministic first walkthrough when live provider setup is not ready.</li>
        {(isPassing ? [`For headless, keep ${providerCommand} available to the ACP service before live analysis.`, "Run first analysis after the remaining setup gates pass."] : guidance.nextActions).map((action) => (
          <li key={action}>{action}</li>
        ))}
      </ul>
    </div>
  );
}

function buildRepoDiagnostics(guidedRepos: GuidedRepo[], validateResult: ValidateResponse | null): Map<string, Diagnostic[]> {
  const diagnosticsByID = new Map<string, Diagnostic[]>();
  const normalizedNameCounts = new Map<string, number>();
  for (const repo of guidedRepos) {
    const normalizedName = repo.name.trim();
    if (normalizedName) {
      normalizedNameCounts.set(normalizedName, (normalizedNameCounts.get(normalizedName) ?? 0) + 1);
    }
  }

  const append = (repoID: string, diagnostic: Diagnostic) => {
    const diagnostics = diagnosticsByID.get(repoID) ?? [];
    diagnostics.push(diagnostic);
    diagnosticsByID.set(repoID, diagnostics);
  };

  for (const repo of guidedRepos) {
    const name = repo.name.trim();
    if (!name) {
      append(repo.id, {
        level: "error",
        code: "repo_name_required",
        message: "Repo name is required.",
        suggestion: "Add a unique workspace repo name.",
      });
    } else if ((normalizedNameCounts.get(name) ?? 0) > 1) {
      append(repo.id, {
        level: "error",
        code: "repo_name_duplicate",
        message: "Repo names must be unique inside workspace.yaml.",
        suggestion: "Rename one of the duplicate repo rows.",
      });
    }
    const sourceValue = repo.mode === "path" ? repo.path.trim() : repo.git_url.trim();
    if (!sourceValue) {
      append(repo.id, {
        level: "error",
        code: repo.mode === "path" ? "repo_path_required" : "repo_git_url_required",
        message: repo.mode === "path" ? "Local checkout path is required." : "Repository URL is required.",
        suggestion: repo.mode === "path" ? "Enter an existing checkout path." : "Enter a GitHub/GitLab URL.",
      });
    }
  }

  const serverDiagnostics = [...(validateResult?.errors ?? []), ...(validateResult?.warnings ?? [])];
  for (const diagnostic of serverDiagnostics) {
    const repoName = diagnostic.repo?.trim();
    if (!repoName) {
      continue;
    }
    for (const repo of guidedRepos) {
      if (repo.name.trim() === repoName) {
        append(repo.id, diagnostic);
      }
    }
  }

  return diagnosticsByID;
}

function findSourceBlockingDiagnostic(repoDiagnosticsByID: Map<string, Diagnostic[]>, validateResult: ValidateResponse | null): Diagnostic | null {
  const repoError = Array.from(repoDiagnosticsByID.values())
    .flat()
    .find((diagnostic) => diagnostic.level === "error");
  return repoError ?? validateResult?.errors?.[0] ?? null;
}

function buildOnboardingProgressSummary({
  workspaceReady,
  sourcesReady,
  runtimeReady,
  localReady,
  briefReady,
  canEnterConsole,
  canRunFirstAnalysis,
  workspacePath,
  guidedRepoCount,
  setupRuntime,
  setupRuntimeProvider,
  sourceBlockingDiagnostic,
  doctorResult,
  firstRunStatus,
}: {
  workspaceReady: boolean;
  sourcesReady: boolean;
  runtimeReady: boolean;
  localReady: boolean;
  briefReady: boolean;
  canEnterConsole: boolean;
  canRunFirstAnalysis: boolean;
  workspacePath: string;
  guidedRepoCount: number;
  setupRuntime: string;
  setupRuntimeProvider: string;
  sourceBlockingDiagnostic: Diagnostic | null;
  doctorResult: DoctorResponse | null;
  firstRunStatus: string;
}): OnboardingProgressSummary {
  const firstDoctorFailure = doctorResult?.checks.find((check) => check.status === "fail") ?? null;
  const items: OnboardingProgressItem[] = [
    {
      id: "workspace",
      label: "Workspace",
      ready: workspaceReady,
      available: true,
      detail: workspaceReady ? compactPathLabel(workspacePath) : "Choose or create the Git-tracked ACP workspace.",
    },
    {
      id: "sources",
      label: "Repositories",
      ready: sourcesReady,
      available: workspaceReady,
      detail: sourcesReady
        ? `${guidedRepoCount} ${guidedRepoCount === 1 ? "repo" : "repos"} validated`
        : sourceBlockingDiagnostic
          ? `${sourceBlockingDiagnostic.code}: ${sourceBlockingDiagnostic.message}`
          : workspaceReady
            ? "Save and validate the repo inventory."
            : "Available after workspace selection.",
    },
    {
      id: "brief",
      label: "Analysis brief",
      ready: briefReady,
      available: sourcesReady,
      detail: briefReady ? "Saved for the first analysis" : sourcesReady ? "Recommended; may be skipped with a quality warning." : "Available after repository validation.",
    },
    {
      id: "runner",
      label: "Runner",
      ready: runtimeReady,
      available: sourcesReady,
      detail: runtimeReady ? runnerLabel(setupRuntime, setupRuntimeProvider) : "Select fake baseline or opt in to a headless provider.",
    },
    {
      id: "review",
      label: "Review & start",
      ready: localReady,
      available: sourcesReady && runtimeReady,
      detail: localReady
        ? "Local readiness passed"
        : firstDoctorFailure
          ? `${firstDoctorFailure.label}: ${firstDoctorFailure.message}`
          : "Run local readiness before the first analysis.",
    },
  ];

  if (!workspaceReady) {
    return {
      step: "Step 1 of 5",
      action: "Create or open a workspace",
      blocker: "Workspace path is not selected.",
      detail: "ACP needs a Git-tracked architecture workspace before it can save sources or runner settings.",
      items,
      tone: "blocked",
    };
  }
  if (!sourcesReady) {
    return {
      step: "Step 2 of 5",
      action: sourceBlockingDiagnostic ? "Fix source fields" : "Save and validate sources",
      blocker: sourceBlockingDiagnostic ? `${sourceBlockingDiagnostic.code}: ${sourceBlockingDiagnostic.message}` : "Sources have not passed validation yet.",
      detail: "Connect at least one repository, then validate the manifest before selecting the final console action.",
      items,
      tone: "blocked",
    };
  }
  if (!runtimeReady) {
    return {
      step: "Step 4 of 5",
      action: "Select the runner",
      blocker: "Runner is not selected.",
      detail: "Use fake for the first deterministic walkthrough. Headless providers remain explicit opt-in.",
      items,
      tone: "blocked",
    };
  }
  if (doctorResult && !localReady) {
    return {
      step: "Step 5 of 5",
      action: "Fix local readiness blockers",
      blocker: firstDoctorFailure ? `${firstDoctorFailure.label}: ${firstDoctorFailure.message}` : "Readiness has warnings or failed checks.",
      detail: canEnterConsole ? "The console can open, but first analysis should wait until readiness passes." : "Readiness is blocking first analysis.",
      items,
      tone: "blocked",
    };
  }
  if (!localReady) {
    return {
      step: "Step 5 of 5",
      action: "Check local readiness",
      blocker: "First analysis needs a passing readiness check.",
      detail: canEnterConsole ? "You can open the console now; run the readiness check before starting analysis." : "Run the doctor check before the first analysis.",
      items,
      tone: "waiting",
    };
  }
  if (canRunFirstAnalysis) {
    return {
      step: "Step 5 of 5",
      action: firstRunStatus ? "First analysis is starting" : "Run first analysis",
      blocker: "No setup blockers.",
      detail: firstRunStatus || "Workspace, sources, runner and local readiness are ready.",
      items,
      tone: "ready",
    };
  }
  return {
    step: "Step 5 of 5",
    action: "Open the console",
    blocker: "Setup is ready for the product shell.",
    detail: "Console review can start from the selected workspace.",
    items,
    tone: "ready",
  };
}

function getOpenConsoleDisabledReason({
  workspaceReady,
  sourcesReady,
  runtimeReady,
  sourceBlockingDiagnostic,
  validateResult,
}: {
  workspaceReady: boolean;
  sourcesReady: boolean;
  runtimeReady: boolean;
  sourceBlockingDiagnostic: Diagnostic | null;
  validateResult: ValidateResponse | null;
}): string {
  if (!workspaceReady) {
    return "select or create a workspace";
  }
  if (!sourcesReady) {
    if (sourceBlockingDiagnostic) {
      return `fix source diagnostic ${sourceBlockingDiagnostic.code}`;
    }
    return validateResult ? "fix source validation errors" : "save and validate sources";
  }
  if (!runtimeReady) {
    return "select a runner";
  }
  return "wait for server setup confirmation";
}

function getFirstAnalysisDisabledReason({
  canEnterConsole,
  localReady,
  openConsoleDisabledReason,
  doctorFailures,
  doctorResult,
}: {
  canEnterConsole: boolean;
  localReady: boolean;
  openConsoleDisabledReason: string;
  doctorFailures: DoctorResponse["checks"];
  doctorResult: DoctorResponse | null;
}): string {
  if (!localReady) {
    if (doctorFailures.length > 0) {
      return `fix ${doctorFailures[0].label}: ${doctorFailures[0].message}`;
    }
    return doctorResult ? "fix readiness checks" : "run local readiness check";
  }
  if (!canEnterConsole) {
    return openConsoleDisabledReason;
  }
  return "ready";
}

function compactPathLabel(pathValue: string): string {
  const value = pathValue.trim();
  if (!value) {
    return "Workspace selected";
  }
  const parts = value.replace(/\\/g, "/").split("/").filter(Boolean);
  const tail = parts.slice(-2).join("/");
  if (parts.length > 2 && tail) {
    return `Selected .../${tail}`;
  }
  if (tail && value.startsWith("/")) {
    return `Selected /${tail}`;
  }
  return tail ? `Selected ${tail}` : `Selected ${value}`;
}

function runnerLabel(runtime: string, provider: string): string {
  if (runtime === "headless") {
    return `headless provider ${provider}`;
  }
  return `${runtime} baseline`;
}
