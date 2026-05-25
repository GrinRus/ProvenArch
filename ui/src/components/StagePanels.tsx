import { Suspense, lazy, useState, type ComponentProps, type ReactNode } from "react";

import { BaselineEditorsPanel } from "./BaselineEditorsPanel";
import { BaselineGitPanel } from "./BaselineGitPanel";
import { RuntimeProfileSettingsPanel } from "./RuntimeProfileSettingsPanel";
import { RunStatusPanel } from "./RunStatusPanel";
import { StatusBadge } from "./ConsolePrimitives";
import { askArchitectureQuestion, type QAAskResponse } from "../lib/qaApi";
import { formatTimestamp } from "../lib/runState";
import type {
  Artifact,
  Diagnostic,
  DoctorResponse,
  EditableArtifactOption,
  GuidedRepo,
  RepoSourceMode,
  RuntimePermissionRequest,
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
  onValidateWorkspace: () => void;
  onCheckDoctor: () => void;
  onRunFirstAnalysis: () => void;
  onSetupRuntimeChange: (value: string) => void;
  onSetupRuntimeProviderChange: (value: string) => void;
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
  onValidateWorkspace,
  onCheckDoctor,
  onRunFirstAnalysis,
  onSetupRuntimeChange,
  onSetupRuntimeProviderChange,
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
      <div className="compat-controls" aria-hidden="true">
        <label htmlFor="setupRuntimeCompat">Runtime mode</label>
        <select id="setupRuntimeCompat" tabIndex={-1} value={setupRuntime} onChange={(event) => onSetupRuntimeChange(event.target.value)}>
          <option value="fake">fake</option>
          <option value="headless">headless</option>
        </select>
        <label htmlFor="setupRuntimeProviderCompat">Headless provider</label>
        <select
          id="setupRuntimeProviderCompat"
          tabIndex={-1}
          value={setupRuntimeProvider}
          onChange={(event) => onSetupRuntimeProviderChange(event.target.value)}
          disabled={setupRuntime !== "headless"}
        >
          <option value="claude-code">claude-code</option>
          <option value="qwen-code">qwen-code</option>
          <option value="codex-code">codex-code</option>
        </select>
        <button type="button" tabIndex={-1} onClick={onValidateWorkspace} disabled={busy} data-testid="workspace-validate-btn">
          Validate workspace
        </button>
        <button type="button" tabIndex={-1} onClick={onCheckDoctor} disabled={busy} data-testid="setup-doctor-btn">
          Check local readiness
        </button>
        <button
          type="button"
          tabIndex={-1}
          onClick={onRunFirstAnalysis}
          disabled={busy || validateResult?.ok !== true}
          data-testid="setup-run-first-btn"
        >
          Run first analysis
        </button>
      </div>
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
                  <button type="button" className="link-button" onClick={() => onOpenArtifact(artifact.path)}>
                    {artifact.path}
                  </button>{" "}
                  ({artifact.kind})
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
                  <button type="button" className="link-button" onClick={() => onOpenArtifact(artifact.path)}>
                    {artifact.path}
                  </button>{" "}
                  ({artifact.kind})
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
              <button type="button" className="link-button" onClick={() => onOpenArtifact(artifact.path)}>
                {artifact.path}
              </button>
              <span>{artifact.label || artifact.kind}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export function AskStagePanel({ onOpenArtifact }: { onOpenArtifact: (path: string) => void }) {
  const [question, setQuestion] = useState("Who owns payments-service?");
  const [answer, setAnswer] = useState<QAAskResponse | null>(null);
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);

  async function handleAsk() {
    const trimmed = question.trim();
    if (!trimmed) {
      setStatus("Question is required.");
      return;
    }
    setBusy(true);
    setStatus("");
    try {
      setAnswer(await askArchitectureQuestion(trimmed));
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
          <p className="hint">Ask read-only questions over charter cards, model, reports, and imported docs.</p>
        </div>
        <StatusBadge tone="ok">read-only</StatusBadge>
      </div>
      <label htmlFor="qaQuestion">Architecture question</label>
      <textarea id="qaQuestion" value={question} onChange={(event) => setQuestion(event.target.value)} rows={3} data-testid="qa-question-input" />
      <button type="button" onClick={handleAsk} disabled={busy} data-testid="qa-ask-btn">
        Ask workspace
      </button>
      {status ? <p className="status warn">{status}</p> : null}
      {answer ? (
        <div className="qa-answer" data-testid="qa-answer">
          <h2>Answer</h2>
          <p>{answer.answer}</p>
          <p className="hint">Confidence: {Math.round(answer.confidence * 100)}%</p>
          {answer.unresolved.length > 0 ? <p className="status warn">Unresolved: {answer.unresolved.join(", ")}</p> : null}
          <h3>Citations</h3>
          {answer.citations.length === 0 ? (
            <p>No citations returned.</p>
          ) : (
            <ul>
              {answer.citations.map((citation) => (
                <li key={`${citation.path}-${citation.reason}`}>
                  <button type="button" className="link-button" onClick={() => onOpenArtifact(citation.path)}>
                    {citation.path}
                  </button>{" "}
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
