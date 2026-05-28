import { Suspense, lazy, useEffect, useState, type ComponentProps, type ReactNode } from "react";

import { BaselineEditorsPanel } from "./BaselineEditorsPanel";
import { BaselineGitPanel } from "./BaselineGitPanel";
import { RuntimeProfileSettingsPanel } from "./RuntimeProfileSettingsPanel";
import { RunStatusPanel } from "./RunStatusPanel";
import { ArtifactPathButton, StatusBadge } from "./ConsolePrimitives";
import { getQARun, startQAQuestion, type QARunResponse } from "../lib/qaApi";
import { formatTimestamp } from "../lib/runState";
import type {
  Artifact,
  Diagnostic,
  DoctorResponse,
  EditableArtifactOption,
  GuidedRepo,
  RepoSourceMode,
  RuntimeExecutionValues,
  RuntimePermissionValues,
  RuntimePermissionRequest,
  RuntimeStepProviderValues,
  RuntimeTimeoutValues,
  RunListItem,
  RunLogEntry,
  RunStatusResponse,
  ValidateResponse,
} from "../lib/appContracts";

const MermaidPreview = lazy(async () => {
  const module = await import("./MermaidPreview");
  return { default: module.MermaidPreview };
});

export type SourceStageProps = {
  busy: boolean;
  guidedRepos: GuidedRepo[];
  guidedDocsImportsPath: string;
  manifestContent: string;
  validateResult: ValidateResponse | null;
  validationDiagnosticsByRepo: Array<[string, Diagnostic[]]>;
  doctorResult: DoctorResponse | null;
  doctorStatus: string;
  setupRuntime: string;
  setupRuntimeProvider: string;
  onRepoChange: (id: string, patch: Partial<GuidedRepo>) => void;
  onAddRepo: () => void;
  onRemoveRepo: (id: string) => void;
  onDocsImportsPathChange: (value: string) => void;
  onApplyGuidedWorkspaceSetup: () => void;
  onManifestChange: (value: string) => void;
  onSaveManifest: () => void;
};

export function SourceStagePanel({
  busy,
  guidedRepos,
  guidedDocsImportsPath,
  manifestContent,
  validateResult,
  validationDiagnosticsByRepo,
  doctorResult,
  doctorStatus,
  setupRuntime,
  setupRuntimeProvider,
  onRepoChange,
  onAddRepo,
  onRemoveRepo,
  onDocsImportsPathChange,
  onApplyGuidedWorkspaceSetup,
  onManifestChange,
  onSaveManifest,
}: SourceStageProps) {
  const firstRepo = guidedRepos[0];
  const suggestedWorkspace = `~/arch-workspaces/${slugify(firstRepo?.name || "my-service")}`;
  return (
    <section className="panel stage-panel" data-testid="workspace-panel">
      <div className="stage-header">
        <div>
          <h1>Source</h1>
          <p className="hint">Connect repository sources and the architecture workspace before running analysis.</p>
        </div>
        <StatusBadge tone={validateResult?.ok ? "ok" : "info"}>{validateResult?.ok ? "validated" : "draft"}</StatusBadge>
      </div>

      <div className="metric-grid source-snapshot" aria-label="source setup summary">
        <div className="metric-tile">
          <span className="metric-label">Repo sources</span>
          <strong>{guidedRepos.length}</strong>
        </div>
        <div className="metric-tile">
          <span className="metric-label">Docs imports</span>
          <strong>{guidedDocsImportsPath ? "set" : "default"}</strong>
        </div>
        <div className="metric-tile">
          <span className="metric-label">Runtime</span>
          <strong>{setupRuntime === "headless" ? setupRuntimeProvider : setupRuntime}</strong>
        </div>
      </div>

      <SourceRepoTable guidedRepos={guidedRepos} validateResult={validateResult} validationDiagnosticsByRepo={validationDiagnosticsByRepo} />

      <div className="form-section">
        <h2>Repository sources</h2>
        <p className="hint">Use GitHub/GitLab URL by default. Private repos use local git authentication.</p>
        {guidedRepos.map((repo, index) => (
          <div className="repo-card" key={repo.id}>
            <div className="repo-card-head">
              <h3>Repo {index + 1}</h3>
              <button type="button" className="inline-danger" onClick={() => onRemoveRepo(repo.id)} disabled={busy || guidedRepos.length <= 1}>
                Remove
              </button>
            </div>

            <div className="repo-field-grid">
              <div className="field">
                <label htmlFor={`guidedRepoName-${repo.id}`}>Repo name</label>
                <input id={`guidedRepoName-${repo.id}`} value={repo.name} onChange={(event) => onRepoChange(repo.id, { name: event.target.value })} />
              </div>

              <div className="field">
                <label htmlFor={`guidedRepoMode-${repo.id}`}>Repo source type</label>
                <select
                  id={`guidedRepoMode-${repo.id}`}
                  value={repo.mode}
                  onChange={(event) => onRepoChange(repo.id, { mode: event.target.value as RepoSourceMode })}
                >
                  <option value="git_url">GitHub/GitLab URL</option>
                  <option value="path">Local folder</option>
                </select>
              </div>

              {repo.mode === "path" ? (
                <div className="field is-wide">
                  <label htmlFor={`guidedRepoPath-${repo.id}`}>Local checkout path</label>
                  <input id={`guidedRepoPath-${repo.id}`} value={repo.path} onChange={(event) => onRepoChange(repo.id, { path: event.target.value })} />
                </div>
              ) : (
                <div className="field is-wide">
                  <label htmlFor={`guidedRepoGitURL-${repo.id}`}>Repository URL</label>
                  <input
                    id={`guidedRepoGitURL-${repo.id}`}
                    value={repo.git_url}
                    onChange={(event) => onRepoChange(repo.id, { git_url: event.target.value })}
                  />
                </div>
              )}

              <div className="field is-wide">
                <label htmlFor={`guidedRepoRef-${repo.id}`}>ref (optional)</label>
                <input
                  id={`guidedRepoRef-${repo.id}`}
                  value={repo.ref}
                  onChange={(event) => onRepoChange(repo.id, { ref: event.target.value })}
                  placeholder="Leave empty to use current checkout"
                />
              </div>
            </div>
          </div>
        ))}
      </div>

      <div className="form-section workspace-source-section">
        <div>
          <h2>Workspace</h2>
          <p className="hint">
            The workspace path is selected when starting `acp serve`. Recommended default: <code>{suggestedWorkspace}</code>.
          </p>
        </div>
        <div className="field">
          <label htmlFor="guidedDocsImportsPath">docs.imports_path</label>
          <input id="guidedDocsImportsPath" value={guidedDocsImportsPath} onChange={(event) => onDocsImportsPathChange(event.target.value)} />
        </div>
      </div>

      <div className="actions">
        <button type="button" onClick={onAddRepo} disabled={busy}>
          Add repo
        </button>
        <button type="button" onClick={onApplyGuidedWorkspaceSetup} disabled={busy}>
          Apply guided workspace form
        </button>
        <button type="button" onClick={onSaveManifest} disabled={busy} data-testid="workspace-save-btn">
          Save and validate workspace.yaml
        </button>
      </div>

      <details className="advanced-block">
        <summary>Advanced workspace.yaml editor</summary>
        <textarea
          id="workspaceManifestEditor"
          name="workspaceManifestEditor"
          aria-label="workspace.yaml content"
          value={manifestContent}
          onChange={(event) => onManifestChange(event.target.value)}
          rows={12}
        />
      </details>
      <WorkspaceValidationResult validateResult={validateResult} validationDiagnosticsByRepo={validationDiagnosticsByRepo} />
      {doctorStatus ? <p className="status">{doctorStatus}</p> : null}
      {doctorResult ? <DoctorChecklist doctorResult={doctorResult} /> : null}
    </section>
  );
}

export type ReadinessStageProps = {
  busy: boolean;
  validateResult: ValidateResponse | null;
  validationDiagnosticsByRepo: Array<[string, Diagnostic[]]>;
  doctorResult: DoctorResponse | null;
  doctorStatus: string;
  firstRunStatus: string;
  setupRuntime: string;
  setupRuntimeProvider: string;
  onSetupRuntimeChange: (value: string) => void;
  onSetupRuntimeProviderChange: (value: string) => void;
  onValidateWorkspace: () => void;
  onCheckDoctor: () => void;
  onRunFirstAnalysis: () => void;
  runtimeSettingsPanel: ReactNode;
  artifactCount: number;
  runtimeTimeoutEffective: RuntimeTimeoutValues;
  runtimeExecutionEffective: RuntimeExecutionValues;
  runtimePermissionEffective: RuntimePermissionValues;
  runtimeStepProviderEffective: Partial<RuntimeStepProviderValues>;
};

export function ReadinessStagePanel({
  busy,
  validateResult,
  validationDiagnosticsByRepo,
  doctorResult,
  doctorStatus,
  firstRunStatus,
  setupRuntime,
  setupRuntimeProvider,
  onSetupRuntimeChange,
  onSetupRuntimeProviderChange,
  onValidateWorkspace,
  onCheckDoctor,
  onRunFirstAnalysis,
  runtimeSettingsPanel,
  artifactCount,
  runtimeTimeoutEffective,
  runtimeExecutionEffective,
  runtimePermissionEffective,
  runtimeStepProviderEffective,
}: ReadinessStageProps) {
  const validated = validateResult?.ok === true;
  return (
    <section className="panel stage-panel" data-testid="readiness-panel">
      <div className="stage-header">
        <div>
          <h1>Readiness</h1>
          <p className="hint">Validate workspace layout, repo access, local prerequisites, and runtime profile.</p>
        </div>
        <StatusBadge tone={validated ? "ok" : validateResult ? "error" : "info"}>{validated ? "ready" : validateResult ? "blocked" : "unchecked"}</StatusBadge>
      </div>

      <ReadinessSummaryCards
        validateResult={validateResult}
        validationDiagnosticsByRepo={validationDiagnosticsByRepo}
        doctorResult={doctorResult}
        setupRuntime={setupRuntime}
        setupRuntimeProvider={setupRuntimeProvider}
        artifactCount={artifactCount}
        runtimePermissionEffective={runtimePermissionEffective}
      />

      <RuntimeProfileSummary
        runtimeTimeoutEffective={runtimeTimeoutEffective}
        runtimeExecutionEffective={runtimeExecutionEffective}
        runtimePermissionEffective={runtimePermissionEffective}
        runtimeStepProviderEffective={runtimeStepProviderEffective}
      />

      <div className="columns compact">
        <div>
          <label htmlFor="setupRuntime">Runtime mode</label>
          <select id="setupRuntime" value={setupRuntime} onChange={(event) => onSetupRuntimeChange(event.target.value)}>
            <option value="fake">fake</option>
            <option value="headless">headless</option>
          </select>
        </div>
        <div>
          <label htmlFor="setupRuntimeProvider">Headless provider</label>
          <select
            id="setupRuntimeProvider"
            value={setupRuntimeProvider}
            onChange={(event) => onSetupRuntimeProviderChange(event.target.value)}
            disabled={setupRuntime !== "headless"}
          >
            <option value="claude-code">claude-code</option>
            <option value="qwen-code">qwen-code</option>
            <option value="codex-code">codex-code</option>
          </select>
        </div>
      </div>
      {setupRuntime === "headless" ? (
        <p className="status warn">
          Headless mode is process-scoped. If this service was started with `--runtime fake`, restart it with `--runtime headless --runtime-provider {setupRuntimeProvider}`.
        </p>
      ) : null}

      <div className="actions">
        <button type="button" onClick={onValidateWorkspace} disabled={busy} data-testid="workspace-validate-btn">
          Validate workspace
        </button>
        <button type="button" onClick={onCheckDoctor} disabled={busy} data-testid="setup-doctor-btn">
          Check local readiness
        </button>
        <button type="button" onClick={onRunFirstAnalysis} disabled={busy || !validated} data-testid="setup-run-first-btn">
          Run first analysis
        </button>
      </div>

      <WorkspaceValidationResult validateResult={validateResult} validationDiagnosticsByRepo={validationDiagnosticsByRepo} />
      {doctorStatus ? <p className="status">{doctorStatus}</p> : null}
      {doctorResult ? <DoctorChecklist doctorResult={doctorResult} /> : null}
      {firstRunStatus ? <p className="status ok">{firstRunStatus}</p> : null}

      <details className="advanced-block">
        <summary>Advanced runtime settings</summary>
        {runtimeSettingsPanel}
      </details>
    </section>
  );
}

function SourceRepoTable({
  guidedRepos,
  validateResult,
  validationDiagnosticsByRepo,
}: {
  guidedRepos: GuidedRepo[];
  validateResult: ValidateResponse | null;
  validationDiagnosticsByRepo: Array<[string, Diagnostic[]]>;
}) {
  const diagnosticsByRepo = new Map(validationDiagnosticsByRepo);
  const resolvedByName = new Map((validateResult?.resolved_repos ?? []).map((repo) => [repo.name, repo]));

  return (
    <section className="subsection source-table-section" data-testid="source-repo-table">
      <div className="section-heading-row">
        <h2>Source repository table</h2>
        <StatusBadge tone={validateResult?.ok ? "ok" : validateResult ? "error" : "info"}>{validateResult?.ok ? "resolved" : validateResult ? "blocked" : "draft"}</StatusBadge>
      </div>
      <div className="run-table-wrap">
        <table className="run-table source-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Source</th>
              <th>Ref</th>
              <th>Analysis include/exclude</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {guidedRepos.map((repo) => {
              const diagnostics = diagnosticsByRepo.get(repo.name) ?? [];
              const hasErrors = diagnostics.some((diagnostic) => diagnostic.level === "error");
              const hasWarnings = diagnostics.some((diagnostic) => diagnostic.level === "warning");
              const resolved = resolvedByName.get(repo.name);
              const statusTone = hasErrors ? "error" : hasWarnings ? "warn" : resolved ? "ok" : validateResult ? "warn" : "info";
              const statusLabel = hasErrors ? "blocked" : hasWarnings ? "warning" : resolved ? "resolved" : validateResult ? "not resolved" : "draft";
              const sourceValue = repo.mode === "path" ? repo.path || "local path missing" : repo.git_url || "Git URL missing";
              return (
                <tr key={`source-row-${repo.id}`}>
                  <td>
                    <strong>{repo.name || "unnamed repo"}</strong>
                  </td>
                  <td>
                    <span className="source-mode-label">{repo.mode === "path" ? "Local" : "Git URL"}</span>
                    <code>{sourceValue}</code>
                  </td>
                  <td>{repo.ref || resolved?.ref || "current/default"}</td>
                  <td>
                    <span className="status warn">Advanced workspace.yaml only</span>
                  </td>
                  <td>
                    <StatusBadge tone={statusTone}>{statusLabel}</StatusBadge>
                    {diagnostics.length > 0 ? <p className="hint">{diagnostics.length} diagnostic(s)</p> : null}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}

export function WorkspaceValidationResult({
  validateResult,
  validationDiagnosticsByRepo,
}: {
  validateResult: ValidateResponse | null;
  validationDiagnosticsByRepo: Array<[string, Diagnostic[]]>;
}) {
  if (!validateResult) {
    return null;
  }
  return (
    <div className="status-block" data-testid="workspace-validate-result">
      <p>
        Workspace: <code>{validateResult.workspace}</code>
      </p>
      <p>Status: {validateResult.ok ? "valid" : "invalid"}</p>

      {(validateResult.resolved_repos ?? []).length > 0 ? (
        <div className="repo-summary" data-testid="workspace-validate-resolved-repos">
          <p className="hint">Resolved repos</p>
          <ul>
            {(validateResult.resolved_repos ?? []).map((repo) => (
              <li key={`resolved-${repo.name}-${repo.path}`}>
                <code>{repo.name}</code> ({repo.source}) {repo.path}
                {repo.ref ? ` @ ${repo.ref}` : ""}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {validationDiagnosticsByRepo.map(([repoKey, diagnostics]) => (
        <div key={`diag-group-${repoKey}`} className="repo-summary">
          <p className="hint">{repoKey === "__workspace__" ? "Workspace diagnostics" : `Diagnostics for ${repoKey}`}</p>
          {diagnostics.map((diagnostic, index) => (
            <p className={diagnostic.level === "error" ? "status err" : "status warn"} key={`${repoKey}-${diagnostic.code}-${diagnostic.message}-${index}`}>
              {diagnostic.level === "error" ? "Error" : "Warning"} [{diagnostic.code}]: {diagnostic.message}
              {diagnostic.suggestion ? <span> Next: {diagnostic.suggestion}</span> : null}
            </p>
          ))}
        </div>
      ))}
    </div>
  );
}

function ReadinessSummaryCards({
  validateResult,
  validationDiagnosticsByRepo,
  doctorResult,
  setupRuntime,
  setupRuntimeProvider,
  artifactCount,
  runtimePermissionEffective,
}: {
  validateResult: ValidateResponse | null;
  validationDiagnosticsByRepo: Array<[string, Diagnostic[]]>;
  doctorResult: DoctorResponse | null;
  setupRuntime: string;
  setupRuntimeProvider: string;
  artifactCount: number;
  runtimePermissionEffective: RuntimePermissionValues;
}) {
  const diagnostics = validationDiagnosticsByRepo.flatMap(([, items]) => items);
  const runtimeCheck = doctorResult?.checks.find((check) => check.id === "runtime_provider");
  const artifactCheck = doctorResult?.checks.find((check) => check.id === "embedded_ui");
  const permissionMode = String(runtimePermissionEffective.mode ?? "trusted_full_access");
  return (
    <section className="readiness-card-grid" aria-label="readiness summary" data-testid="readiness-summary-cards">
      <ReadinessCard
        title="Workspace"
        tone={validateResult?.ok ? "ok" : validateResult ? "error" : "info"}
        status={validateResult?.ok ? "valid" : validateResult ? "blocked" : "unchecked"}
        detail={validateResult?.workspace ?? "workspace manifest has not been validated yet"}
      />
      <ReadinessCard
        title="Repositories"
        tone={diagnostics.some((diagnostic) => diagnostic.level === "error") ? "error" : diagnostics.length > 0 ? "warn" : validateResult?.ok ? "ok" : "info"}
        status={`${validateResult?.resolved_repos?.length ?? 0} resolved`}
        detail={diagnostics.length > 0 ? `${diagnostics.length} diagnostic(s) across repo/workspace sources` : "repo source diagnostics clear or not checked yet"}
      />
      <ReadinessCard
        title="Runtime provider"
        tone={doctorTone(runtimeCheck?.status) ?? (setupRuntime === "fake" ? "ok" : "warn")}
        status={setupRuntime === "headless" ? setupRuntimeProvider : "fake"}
        detail={runtimeCheck?.message ?? "doctor check has not run in this session"}
      />
      <ReadinessCard
        title="Permissions"
        tone={permissionMode === "trusted_full_access" ? "warn" : "ok"}
        status={permissionMode}
        detail={`approval channel: ${String(runtimePermissionEffective.approval_channel ?? "fail_fast")}`}
      />
      <ReadinessCard
        title="Artifacts"
        tone={artifactCount > 0 ? "ok" : (doctorTone(artifactCheck?.status) ?? "info")}
        status={artifactCount > 0 ? `${artifactCount} available` : "none yet"}
        detail={artifactCount > 0 ? "selected run artifacts are ready for review" : artifactCheck?.message ?? "run analysis to produce review artifacts"}
      />
    </section>
  );
}

function RuntimeProfileSummary({
  runtimeTimeoutEffective,
  runtimeExecutionEffective,
  runtimePermissionEffective,
  runtimeStepProviderEffective,
}: {
  runtimeTimeoutEffective: RuntimeTimeoutValues;
  runtimeExecutionEffective: RuntimeExecutionValues;
  runtimePermissionEffective: RuntimePermissionValues;
  runtimeStepProviderEffective: Partial<RuntimeStepProviderValues>;
}) {
  const providerValues = Object.values(runtimeStepProviderEffective).filter(Boolean);
  const uniqueProviders = [...new Set(providerValues)];
  return (
    <section className="runtime-profile-summary" data-testid="readiness-runtime-summary">
      <div className="section-heading-row">
        <h2>Runtime profile summary</h2>
        <StatusBadge tone={String(runtimePermissionEffective.mode) === "trusted_full_access" ? "warn" : "ok"}>
          {String(runtimePermissionEffective.mode ?? "trusted_full_access")}
        </StatusBadge>
      </div>
      <div className="runtime-summary-grid">
        <div>
          <span className="metric-label">Timeouts</span>
          <strong>
            step {runtimeTimeoutEffective.step_timeout_sec}s / pipeline {runtimeTimeoutEffective.pipeline_timeout_sec}s
          </strong>
        </div>
        <div>
          <span className="metric-label">Execution</span>
          <strong>
            {String(runtimeExecutionEffective.strategy)} / max {String(runtimeExecutionEffective.max_parallel_tasks)}
          </strong>
        </div>
        <div>
          <span className="metric-label">Failure policy</span>
          <strong>{String(runtimeExecutionEffective.failure_policy)}</strong>
        </div>
        <div>
          <span className="metric-label">Step providers</span>
          <strong>{uniqueProviders.length > 0 ? uniqueProviders.join(", ") : "default provider"}</strong>
        </div>
      </div>
      <p className="hint">Advanced runtime settings remain available below for exact persisted/effective/source values.</p>
    </section>
  );
}

function ReadinessCard({ title, tone, status, detail }: { title: string; tone: "info" | "ok" | "warn" | "error"; status: string; detail: string }) {
  return (
    <article className={`readiness-card ${tone}`}>
      <div className="section-heading-row">
        <h3>{title}</h3>
        <StatusBadge tone={tone}>{status}</StatusBadge>
      </div>
      <p className="hint">{detail}</p>
    </article>
  );
}

function doctorTone(status?: DoctorResponse["checks"][number]["status"]): "ok" | "warn" | "error" | undefined {
  if (status === "pass") {
    return "ok";
  }
  if (status === "warn") {
    return "warn";
  }
  if (status === "fail") {
    return "error";
  }
  return undefined;
}

function DoctorChecklist({ doctorResult }: { doctorResult: DoctorResponse }) {
  return (
    <div className="status-block" data-testid="setup-doctor-result">
      <p>Status: {doctorResult.ok ? "ready" : "needs attention"}</p>
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

export type CharterStageProps = ComponentProps<typeof BaselineEditorsPanel> & {
  wizardPanel: ReactNode;
  gitPanel: ReactNode;
  wizardProjectName: string;
  wizardScope: string;
  wizardNfr: string;
  wizardRules: string;
  gitStatus: string;
  proposalBranch: string;
};

export function CharterStagePanel({
  wizardPanel,
  gitPanel,
  wizardProjectName,
  wizardScope,
  wizardNfr,
  wizardRules,
  gitStatus,
  proposalBranch,
  ...baselineProps
}: CharterStageProps) {
  const charterArtifacts = baselineProps.baselineEditorArtifacts.filter((artifact) => artifact.path.startsWith("charter/"));
  const domainCards = baselineProps.baselineEditorArtifacts.filter((artifact) => artifact.path.startsWith("charter/cards/domains/"));
  const teamCards = baselineProps.baselineEditorArtifacts.filter((artifact) => artifact.path.startsWith("charter/cards/teams/"));
  const promptPacks = baselineProps.baselineEditorArtifacts.filter((artifact) => artifact.path.startsWith("skills/prompt-packs/"));
  const livePromptPacks = promptPacks.filter((artifact) => artifact.prompt_usage === "live-consumed");
  const referenceOnlyPrompts = baselineProps.baselineEditorArtifacts.filter((artifact) => artifact.prompt_usage === "reference-only");
  const wizardReady = Boolean(wizardProjectName.trim() && wizardScope.trim());

  return (
    <div className="stage-stack" data-testid="charter-panel">
      <section className="panel stage-panel">
        <div className="stage-header">
          <div>
            <h1>Charter</h1>
            <p className="hint">Define project scope, rules, NFRs, domain cards and editable baseline prompts.</p>
          </div>
          <StatusBadge tone={wizardReady ? "ok" : "info"}>{wizardReady ? "ready draft" : "human-owned"}</StatusBadge>
        </div>
        <CharterWizardSummary wizardProjectName={wizardProjectName} wizardScope={wizardScope} wizardNfr={wizardNfr} wizardRules={wizardRules} />
      </section>

      <section className="charter-workbench-grid" data-testid="charter-workbench">
        <CharterCardOverview domainCards={domainCards} teamCards={teamCards} charterArtifacts={charterArtifacts} />
        <CharterPromptBundleStatus
          baselineBundleWarnings={baselineProps.baselineBundleWarnings}
          promptPacks={promptPacks}
          livePromptPacks={livePromptPacks}
          referenceOnlyPrompts={referenceOnlyPrompts}
          selectedEditorPath={baselineProps.selectedEditorPath}
          gitStatus={gitStatus}
          proposalBranch={proposalBranch}
        />
      </section>

      {wizardPanel}
      <BaselineEditorsPanel {...baselineProps} />
      {gitPanel}
    </div>
  );
}

function CharterWizardSummary({
  wizardProjectName,
  wizardScope,
  wizardNfr,
  wizardRules,
}: {
  wizardProjectName: string;
  wizardScope: string;
  wizardNfr: string;
  wizardRules: string;
}) {
  const nfrCount = splitSummaryList(wizardNfr).length;
  const ruleCount = splitSummaryList(wizardRules).length;
  return (
    <div className="charter-summary-grid" data-testid="charter-wizard-summary">
      <article className="charter-summary-card">
        <span className="metric-label">Project</span>
        <strong>{wizardProjectName.trim() || "unnamed project"}</strong>
      </article>
      <article className="charter-summary-card">
        <span className="metric-label">Scope</span>
        <strong>{wizardScope.trim() || "scope required"}</strong>
      </article>
      <article className="charter-summary-card">
        <span className="metric-label">NFR priorities</span>
        <strong>{nfrCount > 0 ? `${nfrCount} listed` : "none listed"}</strong>
      </article>
      <article className="charter-summary-card">
        <span className="metric-label">Rules</span>
        <strong>{ruleCount > 0 ? `${ruleCount} listed` : "none listed"}</strong>
      </article>
    </div>
  );
}

function CharterCardOverview({
  domainCards,
  teamCards,
  charterArtifacts,
}: {
  domainCards: EditableArtifactOption[];
  teamCards: EditableArtifactOption[];
  charterArtifacts: EditableArtifactOption[];
}) {
  return (
    <section className="charter-overview-panel" data-testid="charter-card-overview">
      <div className="section-heading-row">
        <h2>Domain and team cards</h2>
        <StatusBadge tone={domainCards.length + teamCards.length > 0 ? "ok" : "info"}>
          {domainCards.length + teamCards.length > 0 ? "available" : "partial"}
        </StatusBadge>
      </div>
      <div className="charter-card-stats">
        <div>
          <span className="metric-label">Domain cards</span>
          <strong>{domainCards.length}</strong>
        </div>
        <div>
          <span className="metric-label">Team cards</span>
          <strong>{teamCards.length}</strong>
        </div>
        <div>
          <span className="metric-label">Charter artifacts</span>
          <strong>{charterArtifacts.length}</strong>
        </div>
      </div>
      {domainCards.length + teamCards.length > 0 ? (
        <ul className="compact-list">
          {[...domainCards, ...teamCards].slice(0, 5).map((artifact) => (
            <li key={artifact.path}>
              <span>{artifact.label}</span>
              <code>{artifact.path}</code>
            </li>
          ))}
        </ul>
      ) : (
        <p className="hint">No domain/team card artifacts are exposed by the baseline bundle yet. Keep ownership updates in the existing charter files until a cards API exists.</p>
      )}
    </section>
  );
}

function CharterPromptBundleStatus({
  baselineBundleWarnings,
  promptPacks,
  livePromptPacks,
  referenceOnlyPrompts,
  selectedEditorPath,
  gitStatus,
  proposalBranch,
}: {
  baselineBundleWarnings: Diagnostic[];
  promptPacks: EditableArtifactOption[];
  livePromptPacks: EditableArtifactOption[];
  referenceOnlyPrompts: EditableArtifactOption[];
  selectedEditorPath: string;
  gitStatus: string;
  proposalBranch: string;
}) {
  return (
    <section className="charter-overview-panel" data-testid="charter-prompt-bundle-status">
      <div className="section-heading-row">
        <h2>Baseline prompt bundle</h2>
        <StatusBadge tone={baselineBundleWarnings.some((warning) => warning.level === "error") ? "error" : baselineBundleWarnings.length > 0 ? "warn" : "ok"}>
          {baselineBundleWarnings.length > 0 ? `${baselineBundleWarnings.length} warnings` : "ready"}
        </StatusBadge>
      </div>
      <div className="charter-card-stats">
        <div>
          <span className="metric-label">Prompt packs</span>
          <strong>{promptPacks.length}</strong>
        </div>
        <div>
          <span className="metric-label">Live consumed</span>
          <strong>{livePromptPacks.length}</strong>
        </div>
        <div>
          <span className="metric-label">Reference-only</span>
          <strong>{referenceOnlyPrompts.length}</strong>
        </div>
      </div>
      <dl className="compact-defs">
        <div>
          <dt>Selected artifact</dt>
          <dd>{selectedEditorPath || "none selected"}</dd>
        </div>
        <div>
          <dt>Git path</dt>
          <dd>{gitStatus || `proposal branch ${proposalBranch || "not prepared"}`}</dd>
        </div>
      </dl>
    </section>
  );
}

function splitSummaryList(value: string): string[] {
  return value
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

export type AnalysisStageProps = {
  busy: boolean;
  cancelBusy: boolean;
  runId: string | null;
  runStatus: RunStatusResponse | null;
  runList: RunListItem[];
  runActionStatus: string;
  selectedRunWarnings: string[];
  selectedRunIsActive: boolean;
  runCounters: { running: number; succeeded: number; failed: number };
  pendingPermissions: RuntimePermissionRequest[];
  runLogs: RunLogEntry[];
  artifacts: Artifact[];
  setupRuntime: string;
  setupRuntimeProvider: string;
  onReviewBlocker: () => void;
  onRunPipeline: (pipeline: "init" | "refresh") => void;
  onCancelSelectedRun: () => void;
  onSelectRun: (runId: string) => void;
};

export function AnalysisStagePanel({
  busy,
  cancelBusy,
  runId,
  runStatus,
  runList,
  runActionStatus,
  selectedRunWarnings,
  selectedRunIsActive,
  runCounters,
  pendingPermissions,
  runLogs,
  artifacts,
  setupRuntime,
  setupRuntimeProvider,
  onReviewBlocker,
  onRunPipeline,
  onCancelSelectedRun,
  onSelectRun,
}: AnalysisStageProps) {
  const stepTimeline = buildAnalysisStepTimeline(runStatus, runLogs);
  const shardRows = buildAnalysisShardRows(runStatus, runLogs, artifacts, setupRuntime, setupRuntimeProvider);
  const issueRows = shardRows.filter((row) => row.status === "failed" || row.status === "warning");
  const blockerRows = shardRows.filter((row) => row.status === "failed");
  const runtimeLabel = setupRuntime === "fake" ? "fake" : `${setupRuntime}/${setupRuntimeProvider}`;

  return (
    <section className="panel stage-panel" data-testid="runs-control-panel">
      <div className="stage-header">
        <div>
          <h1>Analysis</h1>
          <p className="hint">Run init or refresh, monitor active steps, inspect pending permissions, and select history.</p>
        </div>
        <StatusBadge tone={selectedRunIsActive ? "warn" : runStatus?.status === "succeeded" ? "ok" : runStatus?.status === "failed" ? "error" : "info"}>
          {runStatus?.status ?? "idle"}
        </StatusBadge>
      </div>

      <div className="actions">
        <button type="button" onClick={() => onRunPipeline("init")} disabled={busy} data-testid="run-init-btn">
          Run init
        </button>
        <button type="button" onClick={() => onRunPipeline("refresh")} disabled={busy} data-testid="run-refresh-btn">
          Run refresh
        </button>
        <button type="button" onClick={onCancelSelectedRun} disabled={busy || cancelBusy || !runId || !selectedRunIsActive} data-testid="run-cancel-btn">
          Cancel selected run
        </button>
      </div>
      {runActionStatus ? <p className="status warn">{runActionStatus}</p> : null}

      <AnalysisRunProgress
        runId={runId}
        runStatus={runStatus}
        runtimeLabel={runtimeLabel}
        selectedRunWarnings={selectedRunWarnings}
        stepTimeline={stepTimeline}
        issueCount={issueRows.length}
        blockerCount={blockerRows.length}
        onReviewBlocker={onReviewBlocker}
      />
      <AnalysisRunTimeline steps={stepTimeline} />
      <AnalysisShardTable rows={shardRows} />
      <AnalysisFailedShardDetails rows={issueRows} />

      <RunStatusPanel runStatus={runStatus} warnings={selectedRunWarnings} />
      <PendingPermissionsTable pendingPermissions={pendingPermissions} />
      <RunHistoryTable runId={runId} runList={runList} runCounters={runCounters} onSelectRun={onSelectRun} />
    </section>
  );
}

type AnalysisStepState = "done" | "active" | "failed" | "pending";

type AnalysisStep = {
  id: string;
  label: string;
  state: AnalysisStepState;
  detail: string;
};

type AnalysisShardRow = {
  key: string;
  stepId: string;
  scope: string;
  provider: string;
  status: "succeeded" | "active" | "failed" | "warning" | "observed";
  artifactRef: string;
  lastMessage: string;
};

const canonicalAnalysisSteps = [
  { suffix: "step0.constitution", label: "Charter" },
  { suffix: "step1.collect", label: "Collect" },
  { suffix: "step2.asis_docs", label: "As-is docs" },
  { suffix: "step3.findings", label: "Findings" },
  { suffix: "step4.proposals", label: "Proposals" },
];

function AnalysisRunProgress({
  runId,
  runStatus,
  runtimeLabel,
  selectedRunWarnings,
  stepTimeline,
  issueCount,
  blockerCount,
  onReviewBlocker,
}: {
  runId: string | null;
  runStatus: RunStatusResponse | null;
  runtimeLabel: string;
  selectedRunWarnings: string[];
  stepTimeline: AnalysisStep[];
  issueCount: number;
  blockerCount: number;
  onReviewBlocker: () => void;
}) {
  const completedSteps = stepTimeline.filter((step) => step.state === "done").length;
  const activeOrFailed = stepTimeline.find((step) => step.state === "active" || step.state === "failed");
  const hasBlocker = blockerCount > 0 || runStatus?.status === "failed" || Boolean(runStatus?.error_code);
  return (
    <section className="analysis-progress" data-testid="analysis-run-progress">
      <div className="section-heading-row">
        <h2>Run mission control</h2>
        <StatusBadge tone={runStatus?.status === "succeeded" ? "ok" : runStatus?.status === "failed" ? "error" : runStatus ? "warn" : "info"}>
          {runStatus?.status ?? "idle"}
        </StatusBadge>
      </div>
      <div className="analysis-progress-grid">
        <div>
          <span className="metric-label">Run ID</span>
          <strong>{runId ?? "none selected"}</strong>
        </div>
        <div>
          <span className="metric-label">Runtime/provider</span>
          <strong>{runtimeLabel}</strong>
        </div>
        <div>
          <span className="metric-label">Current step</span>
          <strong>{runStatus?.current_step ?? activeOrFailed?.id ?? "not running"}</strong>
        </div>
        <div>
          <span className="metric-label">Progress</span>
          <strong>
            {completedSteps}/{stepTimeline.length} steps
          </strong>
        </div>
        <div>
          <span className="metric-label">Warnings/errors</span>
          <strong>{selectedRunWarnings.length + issueCount + (runStatus?.error_code ? 1 : 0)}</strong>
        </div>
      </div>
      <button type="button" data-testid="analysis-review-blocker-btn" onClick={onReviewBlocker} disabled={!hasBlocker}>
        Review blocker
      </button>
    </section>
  );
}

function AnalysisRunTimeline({ steps }: { steps: AnalysisStep[] }) {
  return (
    <section className="subsection" data-testid="analysis-run-timeline">
      <h2>Run timeline</h2>
      <ol className="analysis-timeline">
        {steps.map((step, index) => (
          <li key={step.id} className={`analysis-step ${step.state}`}>
            <span className="step-index">{index}</span>
            <div>
              <strong>{step.label}</strong>
              <code>{step.id}</code>
              <span>{step.detail}</span>
            </div>
          </li>
        ))}
      </ol>
    </section>
  );
}

function AnalysisShardTable({ rows }: { rows: AnalysisShardRow[] }) {
  return (
    <section className="subsection" data-testid="analysis-shard-panel">
      <h2>Shard/log table</h2>
      {rows.length === 0 ? (
        <p className="hint">No shard or runtime log rows are available yet. Start analysis or load a run with persisted logs.</p>
      ) : (
        <div className="run-table-wrap analysis-shard-wrap">
          <table className="run-table analysis-shard-table" data-testid="analysis-shard-table">
            <thead>
              <tr>
                <th>Step</th>
                <th>Scope</th>
                <th>Provider</th>
                <th>Status</th>
                <th>Artifact/log ref</th>
                <th>Last message</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.key} className={row.status === "failed" ? "failed" : row.status === "warning" ? "warn" : ""}>
                  <td data-label="Step">{row.stepId}</td>
                  <td data-label="Scope">{row.scope}</td>
                  <td data-label="Provider">{row.provider}</td>
                  <td data-label="Status">{row.status}</td>
                  <td data-label="Artifact/log ref">{row.artifactRef}</td>
                  <td data-label="Last message">{row.lastMessage}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function AnalysisFailedShardDetails({ rows }: { rows: AnalysisShardRow[] }) {
  return (
    <section className="subsection" data-testid="analysis-failed-shard-details">
      <h2>Blocker drilldown</h2>
      {rows.length === 0 ? (
        <p className="hint">No failed shard or warning log entries for the selected run.</p>
      ) : (
        <ul className="compact-list">
          {rows.slice(0, 4).map((row) => (
            <li key={`${row.key}-detail`}>
              <span>
                {row.status.toUpperCase()} · {row.stepId} · {row.scope}
              </span>
              <code>{row.lastMessage}</code>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function buildAnalysisStepTimeline(runStatus: RunStatusResponse | null, runLogs: RunLogEntry[]): AnalysisStep[] {
  const pipeline = runStatus?.pipeline || "init";
  const currentIndex = findStepIndex(runStatus?.current_step);
  const loggedIndex = runLogs.reduce((maxIndex, entry) => Math.max(maxIndex, findStepIndex(entry.step_id)), -1);
  const activeIndex = currentIndex >= 0 ? currentIndex : loggedIndex >= 0 ? loggedIndex : 0;
  return canonicalAnalysisSteps.map((step, index) => {
    const id = `${pipeline}.${step.suffix}`;
    let state: AnalysisStepState = "pending";
    if (runStatus?.status === "succeeded") {
      state = "done";
    } else if (runStatus?.status === "failed") {
      state = index < activeIndex ? "done" : index === activeIndex ? "failed" : "pending";
    } else if (runStatus?.status === "running" || runStatus?.status === "queued") {
      state = index < activeIndex ? "done" : index === activeIndex ? "active" : "pending";
    } else if (loggedIndex >= index && loggedIndex >= 0) {
      state = "done";
    }
    return { id, label: step.label, state, detail: stepTimelineDetail(state) };
  });
}

function buildAnalysisShardRows(
  runStatus: RunStatusResponse | null,
  runLogs: RunLogEntry[],
  artifacts: Artifact[],
  setupRuntime: string,
  setupRuntimeProvider: string,
): AnalysisShardRow[] {
  const grouped = new Map<string, RunLogEntry[]>();
  for (const entry of runLogs) {
    const key = entry.taskrun_path || `${entry.step_id || "run"}/${entry.domain_id || "workspace"}`;
    grouped.set(key, [...(grouped.get(key) ?? []), entry]);
  }
  const provider = setupRuntime === "fake" ? "fake" : setupRuntimeProvider;
  const rows: AnalysisShardRow[] = [];
  for (const [key, entries] of grouped.entries()) {
    const last = entries[entries.length - 1];
    const stepId = last?.step_id || entries.find((entry) => entry.step_id)?.step_id || runStatus?.current_step || "-";
    const hasError = entries.some((entry) => entry.level === "error");
    const hasWarning = entries.some((entry) => entry.level === "warning");
    rows.push({
      key,
      stepId,
      scope: last?.domain_id || fieldString(last?.fields, "domain_id") || fieldString(last?.fields, "repo") || fieldString(last?.fields, "shard_id") || "workspace",
      provider: fieldString(last?.fields, "provider") || provider,
      status: hasError ? "failed" : hasWarning ? "warning" : runStatus?.status === "succeeded" ? "succeeded" : runStatus?.current_step && stepMatches(runStatus.current_step, stepId) ? "active" : "observed",
      artifactRef: last?.taskrun_path || (artifacts.length > 0 ? `${artifacts.length} selected-run artifacts` : "logs only"),
      lastMessage: last?.message || "-",
    });
  }
  if (rows.length === 0 && runStatus) {
    rows.push({
      key: runStatus.run_id,
      stepId: runStatus.current_step || `${runStatus.pipeline}.pending`,
      scope: "workspace",
      provider,
      status: runStatus.status === "failed" ? "failed" : runStatus.status === "succeeded" ? "succeeded" : runStatus.status === "running" ? "active" : "observed",
      artifactRef: artifacts.length > 0 ? `${artifacts.length} selected-run artifacts` : "status only",
      lastMessage: runStatus.error || runStatus.error_code || "No shard logs loaded yet.",
    });
  }
  return rows;
}

function findStepIndex(stepId?: string): number {
  if (!stepId) {
    return -1;
  }
  const normalized = stepId.replace(/_/g, ".").toLowerCase();
  return canonicalAnalysisSteps.findIndex((step) => normalized.includes(step.suffix.replace(/_/g, ".")));
}

function stepMatches(left: string, right: string): boolean {
  return findStepIndex(left) >= 0 && findStepIndex(left) === findStepIndex(right);
}

function stepTimelineDetail(state: AnalysisStepState): string {
  if (state === "done") {
    return "completed";
  }
  if (state === "active") {
    return "current";
  }
  if (state === "failed") {
    return "blocked";
  }
  return "pending";
}

function fieldString(fields: Record<string, unknown> | undefined, key: string): string {
  const value = fields?.[key];
  return typeof value === "string" && value.trim().length > 0 ? value.trim() : "";
}

function PendingPermissionsTable({ pendingPermissions }: { pendingPermissions: RuntimePermissionRequest[] }) {
  return (
    <section className="subsection" data-testid="runs-pending-permissions-panel">
      <h2>Pending permissions</h2>
      {pendingPermissions.length === 0 ? (
        <p>No pending runtime permission requests.</p>
      ) : (
        <div className="run-table-wrap">
          <table className="run-table" data-testid="runs-pending-permissions-table">
            <thead>
              <tr>
                <th>Request ID</th>
                <th>Provider</th>
                <th>Step</th>
                <th>Action</th>
                <th>Decision</th>
                <th>Rule</th>
                <th>Path or command</th>
                <th>Reason</th>
              </tr>
            </thead>
            <tbody>
              {pendingPermissions.map((request) => (
                <tr key={request.request_id || `${request.step_id}-${request.action}-${request.path_or_command}`}>
                  <td>{request.request_id || "-"}</td>
                  <td>{request.provider || "-"}</td>
                  <td>{request.step_id || "-"}</td>
                  <td>{request.action || "-"}</td>
                  <td>{request.decision?.decision || "-"}</td>
                  <td>{request.decision?.rule_id || "-"}</td>
                  <td>{request.path_or_command || "-"}</td>
                  <td>{request.reason || "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function RunHistoryTable({
  runId,
  runList,
  runCounters,
  onSelectRun,
}: {
  runId: string | null;
  runList: RunListItem[];
  runCounters: { running: number; succeeded: number; failed: number };
  onSelectRun: (runId: string) => void;
}) {
  return (
    <section className="subsection" data-testid="runs-history-panel">
      <h2>History</h2>
      <p className="hint">
        Running: {runCounters.running} | Succeeded: {runCounters.succeeded} | Failed: {runCounters.failed}
      </p>
      {runList.length === 0 ? (
        <p>No runs yet.</p>
      ) : (
        <div className="run-table-wrap">
          <table className="run-table" data-testid="runs-history-table">
            <thead>
              <tr>
                <th>Run ID</th>
                <th>Status</th>
                <th>Pipeline</th>
                <th>Started</th>
                <th>Finished</th>
                <th>Error code</th>
                <th>Warnings</th>
              </tr>
            </thead>
            <tbody>
              {runList.map((run) => (
                <tr key={run.run_id} className={runId === run.run_id ? "selected" : ""} onClick={() => onSelectRun(run.run_id)}>
                  <td>
                    <button
                      type="button"
                      className="link-button"
                      onClick={(event) => {
                        event.stopPropagation();
                        onSelectRun(run.run_id);
                      }}
                    >
                      {run.run_id}
                    </button>
                  </td>
                  <td>{run.status}</td>
                  <td>{run.pipeline}</td>
                  <td>{formatTimestamp(run.started_at)}</td>
                  <td>{formatTimestamp(run.finished_at)}</td>
                  <td>{run.error_code || "-"}</td>
                  <td>{run.warnings?.length ?? 0}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

export type ReviewStageProps = {
  coverageSummary: string;
  openQuestions: string;
  nonDiagramArtifacts: Artifact[];
  diagramArtifacts: Artifact[];
  selectedArtifact: string;
  selectedArtifactContent: string;
  selectedArtifactIsMermaid: boolean;
  onOpenArtifact: (path: string) => void;
};

export function ReviewStagePanel({
  coverageSummary,
  openQuestions,
  nonDiagramArtifacts,
  diagramArtifacts,
  selectedArtifact,
  selectedArtifactContent,
  selectedArtifactIsMermaid,
  onOpenArtifact,
}: ReviewStageProps) {
  const [reviewView, setReviewView] = useState<"evidence" | "domain-map">("evidence");
  const overviewArtifact = nonDiagramArtifacts.find((artifact) => artifact.path === "reports/as-is/overview.md");
  const findingsArtifact = nonDiagramArtifacts.find((artifact) => artifact.path.startsWith("reports/findings/"));
  const allArtifacts = [...nonDiagramArtifacts, ...diagramArtifacts];
  const artifactGroups = groupArtifactsByFolder(allArtifacts);
  const selectedArtifactIsLoading = selectedArtifactContent === "Loading...";
  const openQuestionCount = countMarkdownItems(openQuestions);
  const trustStatus = deriveReviewTrustStatus({
    artifactCount: allArtifacts.length,
    hasCoverage: Boolean(coverageSummary),
    findingsCount: findingsArtifact ? 1 : 0,
    openQuestionCount,
  });
  const domainMap = deriveReviewDomainMap({
    artifacts: nonDiagramArtifacts,
    coverageSummary,
    openQuestions,
  });
  function handleOpenDomainMapArtifact(path: string) {
    onOpenArtifact(path);
    setReviewView("evidence");
  }
  return (
    <div className="stage-stack" data-testid="review-panel">
      <section className="panel stage-panel" data-testid="results-coverage-panel">
        <div className="stage-header">
          <div>
            <h1>Review</h1>
            <p className="hint">Validate as-is evidence, coverage gaps, findings and diagrams before publishing workspace changes.</p>
          </div>
          <StatusBadge tone={nonDiagramArtifacts.length + diagramArtifacts.length > 0 ? "ok" : "info"}>
            {nonDiagramArtifacts.length + diagramArtifacts.length} artifacts
          </StatusBadge>
        </div>
        <div className="review-tabs" role="tablist" aria-label="Review views">
          <button
            type="button"
            role="tab"
            aria-selected={reviewView === "evidence"}
            className={reviewView === "evidence" ? "is-active" : ""}
            data-testid="review-view-evidence-tab"
            onClick={() => setReviewView("evidence")}
          >
            Evidence
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={reviewView === "domain-map"}
            className={reviewView === "domain-map" ? "is-active" : ""}
            data-testid="review-view-domain-map-tab"
            onClick={() => setReviewView("domain-map")}
          >
            Domain map
          </button>
        </div>
        {reviewView === "domain-map" ? (
          <ReviewDomainMap domainMap={domainMap} onOpenArtifact={handleOpenDomainMapArtifact} />
        ) : (
          <ReviewEvidenceWorkbench
            coverageSummary={coverageSummary}
            openQuestions={openQuestions}
            openQuestionCount={openQuestionCount}
            trustStatus={trustStatus}
            overviewArtifact={overviewArtifact}
            findingsArtifact={findingsArtifact}
            artifactGroups={artifactGroups}
            nonDiagramArtifacts={nonDiagramArtifacts}
            diagramArtifacts={diagramArtifacts}
            selectedArtifact={selectedArtifact}
            selectedArtifactContent={selectedArtifactContent}
            selectedArtifactIsMermaid={selectedArtifactIsMermaid}
            selectedArtifactIsLoading={selectedArtifactIsLoading}
            onOpenArtifact={onOpenArtifact}
          />
        )}
      </section>
    </div>
  );
}

function ReviewDomainMap({
  domainMap,
  onOpenArtifact,
}: {
  domainMap: ReviewDomainMapModel;
  onOpenArtifact: (path: string) => void;
}) {
  const hasMapData = domainMap.nodes.length > 0 || domainMap.edges.length > 0 || domainMap.domainOutputs.length > 0;
  return (
    <div className="review-domain-map" data-testid="review-domain-map">
      <section className="domain-map-canvas" data-testid="review-domain-map-canvas">
        <div className="section-heading-row">
          <div>
            <h2>Domain/service map</h2>
            <p className="hint">Derived from selected-run model artifacts and domain agent outputs.</p>
          </div>
          <StatusBadge tone={hasMapData ? "ok" : "info"}>{hasMapData ? "derived model" : "partial"}</StatusBadge>
        </div>

        <div className="domain-map-summary-grid">
          <div className="metric-tile">
            <span className="metric-label">Entities</span>
            <strong>{domainMap.entityCount}</strong>
          </div>
          <div className="metric-tile">
            <span className="metric-label">Edges</span>
            <strong>{domainMap.edges.length}</strong>
          </div>
          <div className="metric-tile">
            <span className="metric-label">Domain outputs</span>
            <strong>{domainMap.domainOutputs.length}</strong>
          </div>
          <div className="metric-tile">
            <span className="metric-label">Repo scopes</span>
            <strong>{domainMap.repoCount > 0 ? domainMap.repoCount : "partial"}</strong>
          </div>
        </div>

        {!hasMapData ? (
          <div className="domain-map-empty" data-testid="review-domain-map-empty">
            <strong>No derived model artifacts yet.</strong>
            <span>Run Analysis and load a completed run with `model/entities/*` or `reports/agent-outputs/domains/*` artifacts to populate the map.</span>
          </div>
        ) : (
          <>
            <div className="domain-map-lanes" aria-label="Domain map nodes">
              {domainMap.groups.map((group) => (
                <section className="domain-map-lane" key={group.key}>
                  <div className="domain-map-lane-head">
                    <h3>{group.label}</h3>
                    <span>{group.nodes.length}</span>
                  </div>
                  <div className="domain-map-node-grid">
                    {group.nodes.map((node) => (
                      <article className={`domain-map-node ${node.group}`} data-testid="review-domain-map-node" key={`${node.kind}-${node.id}`}>
                        <div>
                          <span className="metric-label">{node.typeLabel}</span>
                          <strong>{node.label}</strong>
                          <code>{node.id}</code>
                        </div>
                        <ArtifactPathButton
                          path={node.artifact.path}
                          label={node.artifact.label || node.artifact.path}
                          kind={node.artifact.kind}
                          actionLabel="Open map entity"
                          onOpenArtifact={onOpenArtifact}
                        />
                      </article>
                    ))}
                  </div>
                </section>
              ))}
            </div>

            <section className="domain-map-edge-list" data-testid="review-domain-map-edge-list">
              <div className="section-heading-row">
                <h2>Relationships</h2>
                <StatusBadge tone={domainMap.edges.length > 0 ? "ok" : "info"}>{domainMap.edges.length} edges</StatusBadge>
              </div>
              {domainMap.edges.length === 0 ? (
                <p className="hint">No model edge artifacts are available yet. Entity nodes can still be reviewed through their YAML artifacts.</p>
              ) : (
                <ul>
                  {domainMap.edges.map((edge) => (
                    <li className="domain-map-edge" key={edge.id}>
                      <span>
                        <code>{edge.from}</code>
                        <strong>{edge.type}</strong>
                        <code>{edge.to}</code>
                      </span>
                      <ArtifactPathButton
                        path={edge.artifact.path}
                        label={edge.artifact.label || edge.id}
                        kind={edge.artifact.kind}
                        actionLabel="Open map edge"
                        onOpenArtifact={onOpenArtifact}
                      />
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </>
        )}
      </section>

      <aside className="domain-map-inspector" data-testid="review-domain-map-inspector">
        <div className="section-heading-row">
          <h2>Map inspector</h2>
          <StatusBadge tone={domainMap.blockers.length > 0 ? "warn" : hasMapData ? "ok" : "info"}>
            {domainMap.blockers.length > 0 ? "review" : hasMapData ? "ready" : "partial"}
          </StatusBadge>
        </div>
        <dl className="compact-defs">
          <dt>Ownership</dt>
          <dd>{domainMap.ownershipStatus}</dd>
          <dt>Coverage</dt>
          <dd>{domainMap.coverageStatus}</dd>
          <dt>Cross-repo signal</dt>
          <dd>{domainMap.crossRepoStatus}</dd>
          <dt>Publication path</dt>
          <dd>{domainMap.proposalArtifacts.length > 0 ? "Proposal artifacts ready for Publish review" : "Use Publish gate after proposals are generated"}</dd>
        </dl>

        <section className="domain-map-blockers">
          <h3>Blockers / partial state</h3>
          {domainMap.blockers.length === 0 ? (
            <p className="hint">No map-specific blockers detected from the available artifact list.</p>
          ) : (
            <ul>
              {domainMap.blockers.map((blocker) => (
                <li key={blocker}>{blocker}</li>
              ))}
            </ul>
          )}
        </section>

        <section className="domain-map-navigation">
          <h3>Evidence navigation</h3>
          {domainMap.navigationArtifacts.length === 0 ? (
            <p className="hint">No model, domain or proposal artifacts are available for map navigation yet.</p>
          ) : (
            <ul>
              {domainMap.navigationArtifacts.slice(0, 8).map((artifact) => (
                <li key={`${artifact.kind}-${artifact.path}`}>
                  <ArtifactPathButton path={artifact.path} label={artifact.label} kind={artifact.kind} onOpenArtifact={onOpenArtifact} />
                </li>
              ))}
            </ul>
          )}
        </section>
      </aside>
    </div>
  );
}

function ReviewEvidenceWorkbench({
  coverageSummary,
  openQuestions,
  openQuestionCount,
  trustStatus,
  overviewArtifact,
  findingsArtifact,
  artifactGroups,
  nonDiagramArtifacts,
  diagramArtifacts,
  selectedArtifact,
  selectedArtifactContent,
  selectedArtifactIsMermaid,
  selectedArtifactIsLoading,
  onOpenArtifact,
}: {
  coverageSummary: string;
  openQuestions: string;
  openQuestionCount: number;
  trustStatus: ReviewTrustStatus;
  overviewArtifact?: Artifact;
  findingsArtifact?: Artifact;
  artifactGroups: ArtifactGroup[];
  nonDiagramArtifacts: Artifact[];
  diagramArtifacts: Artifact[];
  selectedArtifact: string;
  selectedArtifactContent: string;
  selectedArtifactIsMermaid: boolean;
  selectedArtifactIsLoading: boolean;
  onOpenArtifact: (path: string) => void;
}) {
  return (
    <div className="review-workbench">
      <aside className="review-artifact-explorer" data-testid="review-artifact-explorer">
        <div className="section-heading-row">
          <h2>Artifact explorer</h2>
          <StatusBadge tone={artifactGroups.length > 0 ? "ok" : "info"}>{artifactGroups.length} groups</StatusBadge>
        </div>
        {artifactGroups.length === 0 ? (
          <p className="hint">No selected-run artifacts yet. Run Analysis before evidence review.</p>
        ) : (
          <div className="artifact-group-list" data-testid="results-artifacts-panel">
            {artifactGroups.map((group) => (
              <section key={group.name} className="artifact-group">
                <h3>{group.name}</h3>
                <ul data-testid={group.name === "reports/diagrams" ? "run-diagrams-list" : undefined}>
                  {group.artifacts.map((artifact) => (
                    <li key={`${artifact.kind}-${artifact.path}`}>
                      <ArtifactPathButton path={artifact.path} label={artifact.label} kind={artifact.kind} onOpenArtifact={onOpenArtifact} />
                    </li>
                  ))}
                </ul>
              </section>
            ))}
          </div>
        )}
      </aside>

      <section className="review-evidence-preview" data-testid="review-evidence-preview">
        <div className="section-heading-row">
          <div>
            <h2>Evidence preview</h2>
            <p className="hint">Select an artifact to inspect the reviewable evidence body.</p>
          </div>
          <button type="button" disabled title="Evidence approval persistence is planned for a later publish gate slice.">
            Approve selected evidence
          </button>
        </div>
        {selectedArtifactIsMermaid ? (
          <div data-testid="run-diagram-content-panel">
            <h3 data-testid="run-diagram-selected-path">{selectedArtifact || "Diagram Preview"}</h3>
            {selectedArtifactIsLoading ? (
              <p className="hint">Loading diagram...</p>
            ) : (
              <Suspense fallback={<p className="hint">Loading diagram renderer...</p>}>
                <MermaidPreview source={selectedArtifactContent} title={selectedArtifact || "diagram"} />
              </Suspense>
            )}
          </div>
        ) : (
          <div data-testid="run-artifact-content-panel">
            <h3 data-testid="run-artifact-selected-path">{selectedArtifact || "Artifact Content"}</h3>
            <pre data-testid="run-artifact-content">{selectedArtifactContent || "Select artifact to inspect."}</pre>
          </div>
        )}
      </section>

      <aside className="review-intel" data-testid="review-citation-coverage">
        <div className="section-heading-row">
          <h2>Citations / coverage</h2>
          <StatusBadge tone={trustStatus.tone}>{trustStatus.label}</StatusBadge>
        </div>
        <div className="review-intel-grid">
          <div className="metric-tile">
            <span className="metric-label">Architecture overview</span>
            <strong>{overviewArtifact ? "ready" : "missing"}</strong>
          </div>
          <div className="metric-tile">
            <span className="metric-label">Coverage summary</span>
            <strong>{coverageSummary ? "ready" : "missing"}</strong>
          </div>
          <div className="metric-tile">
            <span className="metric-label">Findings</span>
            <strong>{findingsArtifact ? "ready" : "missing"}</strong>
          </div>
          <div className="metric-tile">
            <span className="metric-label">Open questions</span>
            <strong>{openQuestionCount}</strong>
          </div>
          <div className="metric-tile">
            <span className="metric-label">Diagrams</span>
            <strong>{diagramArtifacts.length}</strong>
          </div>
        </div>
        <div className="trust-panel">
          <strong>{trustStatus.title}</strong>
          <span>{trustStatus.detail}</span>
        </div>
        <div className="review-source-lists">
          <div>
            <h2>Coverage Summary</h2>
            <pre data-testid="coverage-summary-content">{coverageSummary || "No coverage summary yet."}</pre>
          </div>
          <div>
            <h2>Open Questions</h2>
            <pre data-testid="open-questions-content">{openQuestions || "No open questions yet."}</pre>
          </div>
        </div>
      </aside>

    </div>
  );
}

type ArtifactGroup = {
  name: string;
  artifacts: Artifact[];
};

type ReviewDomainMapNode = {
  id: string;
  label: string;
  typeLabel: string;
  group: string;
  kind: "domain" | "entity";
  artifact: Artifact;
};

type ReviewDomainMapEdge = {
  id: string;
  type: string;
  from: string;
  to: string;
  artifact: Artifact;
};

type ReviewDomainMapGroup = {
  key: string;
  label: string;
  nodes: ReviewDomainMapNode[];
};

type ReviewDomainMapModel = {
  nodes: ReviewDomainMapNode[];
  groups: ReviewDomainMapGroup[];
  edges: ReviewDomainMapEdge[];
  domainOutputs: Artifact[];
  proposalArtifacts: Artifact[];
  navigationArtifacts: Artifact[];
  entityCount: number;
  repoCount: number;
  ownershipStatus: string;
  coverageStatus: string;
  crossRepoStatus: string;
  blockers: string[];
};

type ReviewTrustStatus = {
  label: string;
  title: string;
  detail: string;
  tone: "ok" | "warn" | "info";
};

function groupArtifactsByFolder(artifacts: Artifact[]): ArtifactGroup[] {
  const groups = new Map<string, Artifact[]>();
  for (const artifact of artifacts) {
    const name = reviewArtifactGroupName(artifact.path);
    groups.set(name, [...(groups.get(name) ?? []), artifact]);
  }
  return Array.from(groups.entries())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([name, groupArtifacts]) => ({ name, artifacts: groupArtifacts.sort((left, right) => left.path.localeCompare(right.path)) }));
}

function reviewArtifactGroupName(path: string): string {
  const parts = path.split("/").filter(Boolean);
  if (parts.length === 0) {
    return "root";
  }
  if (parts[0] === "reports" && parts[1]) {
    return `${parts[0]}/${parts[1]}`;
  }
  if (parts[0] === "model" && parts[1]) {
    return `${parts[0]}/${parts[1]}`;
  }
  return parts[0];
}

const MODEL_EDGE_TYPES = ["publishes", "subscribes", "calls", "reads", "writes", "exposes"] as const;

const DOMAIN_MAP_GROUPS: Array<{ key: string; label: string }> = [
  { key: "domains", label: "Domains" },
  { key: "services", label: "Services" },
  { key: "interfaces", label: "Interfaces / topics" },
  { key: "data", label: "Data stores" },
  { key: "external", label: "External systems" },
  { key: "ownership", label: "Ownership / repos" },
  { key: "other", label: "Other model artifacts" },
];

function deriveReviewDomainMap({
  artifacts,
  coverageSummary,
  openQuestions,
}: {
  artifacts: Artifact[];
  coverageSummary: string;
  openQuestions: string;
}): ReviewDomainMapModel {
  const domainOutputs = artifacts
    .filter((artifact) => artifact.path.startsWith("reports/agent-outputs/domains/") && artifact.path.endsWith(".md"))
    .sort((left, right) => left.path.localeCompare(right.path));
  const entityArtifacts = artifacts
    .filter((artifact) => artifact.path.startsWith("model/entities/") && (artifact.path.endsWith(".yaml") || artifact.path.endsWith(".yml")))
    .sort((left, right) => left.path.localeCompare(right.path));
  const edgeArtifacts = artifacts
    .filter((artifact) => artifact.path.startsWith("model/edges/") && (artifact.path.endsWith(".yaml") || artifact.path.endsWith(".yml")))
    .sort((left, right) => left.path.localeCompare(right.path));
  const proposalArtifacts = artifacts
    .filter((artifact) => artifact.path.startsWith("proposals/") || artifact.path.startsWith("reports/changelog/"))
    .sort((left, right) => left.path.localeCompare(right.path));

  const domainNodes: ReviewDomainMapNode[] = domainOutputs.map((artifact) => {
    const id = stripArtifactSuffix(artifact.path.split("/").pop() ?? artifact.path);
    return {
      id,
      label: artifact.label?.trim() || humanizeModelID(id),
      typeLabel: "domain output",
      group: "domains",
      kind: "domain",
      artifact,
    };
  });
  const entityNodes = entityArtifacts.map((artifact) => {
    const id = stripArtifactSuffix(artifact.path.replace(/^model\/entities\//, ""));
    const meta = deriveEntityMapMeta(id);
    return {
      id,
      label: artifact.label?.trim() || humanizeModelID(id),
      typeLabel: meta.typeLabel,
      group: meta.group,
      kind: "entity" as const,
      artifact,
    };
  });
  const nodes = [...domainNodes, ...entityNodes];
  const edges = edgeArtifacts.map(parseModelEdgeArtifact);
  const groups = DOMAIN_MAP_GROUPS.map((group) => ({
    ...group,
    nodes: nodes.filter((node) => node.group === group.key),
  })).filter((group) => group.nodes.length > 0);
  const teamCount = entityNodes.filter((node) => node.id.startsWith("team.")).length;
  const serviceCount = entityNodes.filter((node) => node.id.startsWith("svc.")).length;
  const repoCount = entityNodes.filter((node) => node.id.startsWith("repo.")).length;
  const coverageAvailable = Boolean(coverageSummary.trim()) || artifacts.some((artifact) => artifact.path === "reports/coverage/summary.md");
  const openQuestionCount = countMarkdownItems(openQuestions);
  const blockers: string[] = [];
  if (entityNodes.length === 0) {
    blockers.push("Derived model entities are missing; map is limited to domain agent outputs.");
  }
  if (entityNodes.length > 1 && edges.length === 0) {
    blockers.push("Model entities exist, but no model edge artifacts are available.");
  }
  if (serviceCount > 0 && teamCount === 0) {
    blockers.push("Service nodes are present, but team ownership entities are missing from the artifact list.");
  }
  if (openQuestionCount > 0) {
    blockers.push(`${openQuestionCount} open questions remain linked to evidence review.`);
  }

  const navigationArtifacts = dedupeArtifactNavigation([...domainOutputs, ...entityArtifacts.slice(0, 4), ...edgeArtifacts.slice(0, 3), ...proposalArtifacts.slice(0, 2)]);
  return {
    nodes,
    groups,
    edges,
    domainOutputs,
    proposalArtifacts,
    navigationArtifacts,
    entityCount: entityNodes.length,
    repoCount,
    ownershipStatus:
      teamCount > 0
        ? `${teamCount} team node${teamCount === 1 ? "" : "s"} visible`
        : serviceCount > 0
          ? "partial: service ownership requires team entities or entity content review"
          : "partial: no service ownership data",
    coverageStatus: coverageAvailable ? "coverage summary linked" : "partial: coverage summary missing",
    crossRepoStatus:
      repoCount > 1
        ? `${repoCount} repo scopes visible`
        : repoCount === 1
          ? "single repo scope visible"
          : domainOutputs.length > 1
            ? "partial: multiple domain outputs, no repo entity artifacts"
            : "partial: repo scope not visible in model artifacts",
    blockers,
  };
}

function deriveEntityMapMeta(id: string): { typeLabel: string; group: string } {
  if (id.startsWith("svc.")) {
    return { typeLabel: "service", group: "services" };
  }
  if (id.startsWith("api.") || id.startsWith("topic.")) {
    return { typeLabel: id.startsWith("topic.") ? "event topic" : "api", group: "interfaces" };
  }
  if (id.startsWith("db.")) {
    return { typeLabel: "datastore", group: "data" };
  }
  if (id.startsWith("ext.")) {
    return { typeLabel: "external system", group: "external" };
  }
  if (id.startsWith("team.")) {
    return { typeLabel: "team", group: "ownership" };
  }
  if (id.startsWith("repo.")) {
    return { typeLabel: "repo", group: "ownership" };
  }
  return { typeLabel: "entity", group: "other" };
}

function parseModelEdgeArtifact(artifact: Artifact): ReviewDomainMapEdge {
  const id = stripArtifactSuffix(artifact.path.replace(/^model\/edges\//, ""));
  const edgeBody = id.startsWith("edge.") ? id.slice("edge.".length) : id;
  for (const type of MODEL_EDGE_TYPES) {
    const marker = `.${type}.`;
    const index = edgeBody.indexOf(marker);
    if (index > 0) {
      return {
        id,
        type,
        from: edgeBody.slice(0, index),
        to: edgeBody.slice(index + marker.length),
        artifact,
      };
    }
  }
  return { id, type: "related", from: "unknown", to: edgeBody || "unknown", artifact };
}

function stripArtifactSuffix(value: string): string {
  return value.replace(/\.(yaml|yml|md|json)$/i, "");
}

function humanizeModelID(id: string): string {
  const normalized = id
    .replace(/^(svc|team|repo|ext|db|topic|api\.http|api\.grpc)\./, "")
    .replace(/\./g, " ")
    .replace(/-/g, " ")
    .trim();
  if (!normalized) {
    return id;
  }
  return normalized.replace(/\b[a-z]/g, (match) => match.toUpperCase());
}

function dedupeArtifactNavigation(artifacts: Artifact[]): Artifact[] {
  const seen = new Set<string>();
  const deduped: Artifact[] = [];
  for (const artifact of artifacts) {
    if (seen.has(artifact.path)) {
      continue;
    }
    seen.add(artifact.path);
    deduped.push(artifact);
  }
  return deduped;
}

function countMarkdownItems(content: string): number {
  return content
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.startsWith("- ") || /^\d+\./.test(line)).length;
}

function deriveReviewTrustStatus({
  artifactCount,
  hasCoverage,
  findingsCount,
  openQuestionCount,
}: {
  artifactCount: number;
  hasCoverage: boolean;
  findingsCount: number;
  openQuestionCount: number;
}): ReviewTrustStatus {
  if (artifactCount === 0) {
    return { label: "partial", title: "No evidence selected", detail: "Run Analysis to generate reviewable artifacts.", tone: "info" };
  }
  if (openQuestionCount > 0) {
    return { label: "review", title: "Review required", detail: "Open questions are present and should be resolved or accepted before publication.", tone: "warn" };
  }
  if (hasCoverage && findingsCount > 0) {
    return { label: "ready", title: "Evidence ready", detail: "Coverage and findings artifacts are available for human review.", tone: "ok" };
  }
  return { label: "partial", title: "Partial evidence", detail: "Some review artifacts are missing; inspect generated outputs before publication.", tone: "info" };
}

export function ProposalsStagePanel({
  artifacts,
  onOpenArtifact,
}: {
  artifacts: Artifact[];
  onOpenArtifact: (path: string) => void;
}) {
  const proposalArtifacts = artifacts.filter((artifact) => artifact.path.startsWith("proposals/") || artifact.path.startsWith("reports/changelog/"));
  return (
    <section className="panel stage-panel" data-testid="proposals-panel">
      <div className="stage-header">
        <div>
          <h1>Proposals</h1>
          <p className="hint">Review generated proposal packages, ADR/RFC drafts and iteration changelog.</p>
        </div>
        <StatusBadge tone={proposalArtifacts.length > 0 ? "ok" : "info"}>{proposalArtifacts.length} refs</StatusBadge>
      </div>
      {proposalArtifacts.length === 0 ? (
        <p>No proposal or changelog artifacts yet.</p>
      ) : (
        <ul className="artifact-list">
          {proposalArtifacts.map((artifact) => (
            <li key={`${artifact.kind}-${artifact.path}`}>
              <ArtifactPathButton path={artifact.path} label={artifact.label} kind={artifact.kind} onOpenArtifact={onOpenArtifact} />
              <span>{artifact.kind}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export function AskStagePanel({ onOpenArtifact }: { onOpenArtifact: (path: string) => void }) {
  const [question, setQuestion] = useState("");
  const [qaRun, setQARun] = useState<QARunResponse | null>(null);
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  const qaRunActive = qaRun?.status === "queued" || qaRun?.status === "running";

  useEffect(() => {
    if (!qaRun?.run_id || !qaRunActive) {
      return;
    }
    let canceled = false;
    const refresh = async () => {
      try {
        const next = await getQARun(qaRun.run_id);
        if (!canceled) {
          setQARun(next);
          setStatus(next.status === "succeeded" ? "Q&A run completed." : next.status === "failed" ? "Q&A run failed." : "Q&A run is running.");
        }
      } catch (error) {
        if (!canceled) {
          setStatus(error instanceof Error ? error.message : "Q&A run polling failed");
        }
      }
    };
    const interval = window.setInterval(() => void refresh(), 1000);
    return () => {
      canceled = true;
      window.clearInterval(interval);
    };
  }, [qaRun?.run_id, qaRunActive]);

  async function handleAsk() {
    const trimmed = question.trim();
    if (!trimmed) {
      setStatus("Question is required.");
      return;
    }
    setBusy(true);
    setQARun(null);
    setStatus("");
    try {
      const started = await startQAQuestion(trimmed);
      setStatus("Q&A run queued.");
      const detail = await getQARun(started.run_id);
      setQARun(detail);
      if (detail.status === "succeeded") {
        setStatus("Q&A run completed.");
      } else if (detail.status === "failed") {
        setStatus("Q&A run failed.");
      } else {
        setStatus("Q&A run is running.");
      }
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Q&A request failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel stage-panel" data-testid="qa-panel">
      <div className="stage-header">
        <div>
          <h1>Ask</h1>
          <p className="hint">Ask agent-backed questions over existing workspace artifacts. Source repos and canonical outputs stay unchanged.</p>
        </div>
        <StatusBadge tone={qaRun?.status === "failed" ? "error" : qaRunActive ? "warn" : qaRun?.status === "succeeded" ? "ok" : "info"}>
          {qaRun?.provider || qaRun?.runtime_provider || "agent-backed"}
        </StatusBadge>
      </div>
      <label htmlFor="qaQuestion">Architecture question</label>
      <textarea
        id="qaQuestion"
        value={question}
        onChange={(event) => setQuestion(event.target.value)}
        rows={3}
        placeholder="Ask about ownership, dependencies, findings, proposals, or coverage in this workspace."
        data-testid="qa-question-input"
      />
      <button type="button" onClick={handleAsk} disabled={busy || qaRunActive} data-testid="qa-ask-btn">
        {qaRunActive ? "Agent is answering" : "Ask workspace"}
      </button>
      {status ? <p className="status warn">{status}</p> : null}
      {qaRun ? (
        <div className="run-summary" data-testid="qa-run-status">
          <p>
            Run <code>{qaRun.run_id}</code> status: <strong>{qaRun.status}</strong>
          </p>
          <p>Runtime provider: {qaRun.provider || qaRun.runtime_provider || "pending"}</p>
          <p>
            <a href={`/api/pipeline/runs/${encodeURIComponent(qaRun.run_id)}/logs`} target="_blank" rel="noreferrer">
              Open run logs
            </a>
          </p>
          {qaRun.error ? <p className="status err">{qaRun.error}</p> : null}
          <div className="actions">
            <button type="button" className="link-button" onClick={() => onOpenArtifact(`reports/taskruns/${qaRun.run_id}/qa/context-pack.json`)}>
              context-pack.json
            </button>
            <button type="button" className="link-button" onClick={() => onOpenArtifact(`reports/taskruns/${qaRun.run_id}/qa/runtime-execution.json`)}>
              runtime-execution.json
            </button>
          </div>
        </div>
      ) : null}
      {qaRun?.answer ? (
        <div className="qa-answer" data-testid="qa-answer">
          <h2>Answer</h2>
          <p>{qaRun.answer}</p>
          <p className="hint">Confidence: {typeof qaRun.confidence === "number" ? Math.round(qaRun.confidence * 100) : 0}%</p>
          {(qaRun.unresolved ?? []).length > 0 ? <p className="status warn">Unresolved: {(qaRun.unresolved ?? []).join(", ")}</p> : null}
          <h3>Citations</h3>
          {(qaRun.citations ?? []).length === 0 ? (
            <p>No citations returned.</p>
          ) : (
            <ul>
              {(qaRun.citations ?? []).map((citation) => (
                <li key={`${citation.path}-${citation.reason}`}>
                  <ArtifactPathButton path={citation.path} onOpenArtifact={onOpenArtifact} />{" "}
                  {citation.reason}
                </li>
              ))}
            </ul>
          )}
        </div>
      ) : null}
    </section>
  );
}

export function PublishStagePanel(props: ComponentProps<typeof BaselineGitPanel>) {
  return (
    <div className="stage-stack" data-testid="publish-panel">
      <section className="panel stage-panel">
        <div className="stage-header">
          <div>
            <h1>Publish</h1>
            <p className="hint">Commit architecture workspace changes or switch to a proposal branch for review.</p>
          </div>
          <StatusBadge tone="info">git-backed</StatusBadge>
        </div>
      </section>
      <BaselineGitPanel {...props} />
    </div>
  );
}

export function RuntimeSettingsStagePanel(props: ComponentProps<typeof RuntimeProfileSettingsPanel>) {
  return <RuntimeProfileSettingsPanel {...props} />;
}

function slugify(value: string): string {
  const normalized = value.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
  return normalized || "my-service";
}
