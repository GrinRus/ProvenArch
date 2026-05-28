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
};

export function CharterStagePanel({ wizardPanel, gitPanel, ...baselineProps }: CharterStageProps) {
  return (
    <div className="stage-stack" data-testid="charter-panel">
      <section className="panel stage-panel">
        <div className="stage-header">
          <div>
            <h1>Charter</h1>
            <p className="hint">Define project scope, rules, NFRs, domain cards and editable baseline prompts.</p>
          </div>
          <StatusBadge tone="info">human-owned</StatusBadge>
        </div>
      </section>
      {wizardPanel}
      <BaselineEditorsPanel {...baselineProps} />
      {gitPanel}
    </div>
  );
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
  onRunPipeline,
  onCancelSelectedRun,
  onSelectRun,
}: AnalysisStageProps) {
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

      <RunStatusPanel runStatus={runStatus} warnings={selectedRunWarnings} />
      <PendingPermissionsTable pendingPermissions={pendingPermissions} />
      <RunHistoryTable runId={runId} runList={runList} runCounters={runCounters} onSelectRun={onSelectRun} />
    </section>
  );
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
  const overviewArtifact = nonDiagramArtifacts.find((artifact) => artifact.path === "reports/as-is/overview.md");
  const findingsArtifact = nonDiagramArtifacts.find((artifact) => artifact.path.startsWith("reports/findings/"));
  const selectedArtifactIsLoading = selectedArtifactContent === "Loading...";
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
        <div className="metric-grid">
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
        </div>
        <div className="columns">
          <div>
            <h2>Coverage Summary</h2>
            <pre data-testid="coverage-summary-content">{coverageSummary || "No coverage summary yet."}</pre>
          </div>
          <div>
            <h2>Open Questions</h2>
            <pre data-testid="open-questions-content">{openQuestions || "No open questions yet."}</pre>
          </div>
        </div>
      </section>

      <section className="panel stage-panel" data-testid="results-artifacts-panel">
        <h2>Artifacts</h2>
        {nonDiagramArtifacts.length === 0 ? (
          <p>No non-diagram artifacts yet.</p>
        ) : (
          <div className="columns">
            <ul data-testid="run-artifacts-list">
              {nonDiagramArtifacts.map((artifact) => (
                <li key={`${artifact.kind}-${artifact.path}`}>
                  <ArtifactPathButton path={artifact.path} label={artifact.label} kind={artifact.kind} onOpenArtifact={onOpenArtifact} />
                </li>
              ))}
            </ul>
            <div data-testid="run-artifact-content-panel">
              <h3 data-testid="run-artifact-selected-path">{selectedArtifact || "Artifact Content"}</h3>
              <pre data-testid="run-artifact-content">{selectedArtifactContent || "Select artifact to inspect."}</pre>
            </div>
          </div>
        )}
      </section>

      <section className="panel stage-panel" data-testid="results-diagrams-panel">
        <h2>Generated diagrams</h2>
        {diagramArtifacts.length === 0 ? (
          <p>No diagram artifacts yet.</p>
        ) : (
          <div className="columns">
            <ul data-testid="run-diagrams-list">
              {diagramArtifacts.map((artifact) => (
                <li key={`${artifact.kind}-${artifact.path}`}>
                  <ArtifactPathButton path={artifact.path} label={artifact.label} kind={artifact.kind} onOpenArtifact={onOpenArtifact} />
                </li>
              ))}
            </ul>
            <div data-testid="run-diagram-content-panel">
              <h3 data-testid="run-diagram-selected-path">{selectedArtifact || "Diagram Preview"}</h3>
              {selectedArtifactIsMermaid ? (
                selectedArtifactIsLoading ? (
                  <p className="hint">Loading diagram...</p>
                ) : (
                  <Suspense fallback={<p className="hint">Loading diagram renderer...</p>}>
                    <MermaidPreview source={selectedArtifactContent} title={selectedArtifact || "diagram"} />
                  </Suspense>
                )
              ) : (
                <pre data-testid="run-diagram-content">{selectedArtifactContent || "Select a `.mmd` diagram artifact to preview."}</pre>
              )}
            </div>
          </div>
        )}
      </section>
    </div>
  );
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
