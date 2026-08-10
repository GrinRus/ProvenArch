import { type ReactNode } from "react";

import { RepoAnalysisScopeFields } from "../../components/RepoAnalysisScopeFields";
import { StatusBadge } from "../../components/ConsolePrimitives";
import { providerCommandEnv, providerCommandHint, providerReadinessGuidance } from "../../lib/providerGuidance";
import { analysisScopeSummary } from "../../lib/analysisScope";
import { runtimeDisplayLabel } from "../../lib/runtimeDisplay";
import { isRunnerUnavailable } from "../../lib/runState";
import { slugify } from "../publish/publishUtils";
import {
  buildSourceValidationRecovery,
  type SourceRecoveryIssue,
} from "./sourceUtils";
import { workspaceHealthLabel, workspaceHealthTone } from "./readinessUtils";
import type {
  Diagnostic,
  DoctorResponse,
  GuidedRepo,
  RepoSourceMode,
  RuntimeExecutionValues,
  RuntimePermissionValues,
  RuntimeStepProviderValues,
  RuntimeTimeoutValues,
  ValidateResponse,
  WorkspaceHealthResponse,
} from "../../lib/appContracts";

export type SourceStageProps = {
  setupView?: "workspace" | "sources";
  busy: boolean;
  guidedRepos: GuidedRepo[];
  guidedDocsImportsPath: string;
  manifestContent: string;
  manifestStatus: string;
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
  onSaveGuidedWorkspaceSetup: () => void;
  onManifestChange: (value: string) => void;
  onSaveManifest: () => void;
};

export function SourceStagePanel({
  setupView = "sources",
  busy,
  guidedRepos,
  guidedDocsImportsPath,
  manifestContent,
  manifestStatus,
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
  onSaveGuidedWorkspaceSetup,
  onManifestChange,
  onSaveManifest,
}: SourceStageProps) {
  const firstRepo = guidedRepos[0];
  const suggestedWorkspace = `~/arch-workspaces/${slugify(firstRepo?.name || "my-service")}`;
  const sourceRecovery = buildSourceValidationRecovery(guidedRepos, validateResult, validationDiagnosticsByRepo);
  return (
    <section className={`panel stage-panel source-setup ${setupView === "workspace" ? "is-workspace-overview" : "is-source-editor"}`} data-testid="workspace-panel">
      <div className="stage-header">
        <div>
          <h1>{setupView === "workspace" ? "Workspace" : "Repositories"}</h1>
          <p className="hint">{setupView === "workspace" ? "Review the selected architecture workspace and its validation state." : "Connect read-only repository sources before running analysis."}</p>
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
          <strong>{runtimeDisplayLabel(setupRuntime, setupRuntimeProvider, { compact: true })}</strong>
        </div>
      </div>

      <div className="stage-local-next-action" data-testid="source-next-action">
        <strong>{setupView === "workspace" ? "Workspace status" : "Next in Repositories"}</strong>
        <span>{setupView === "workspace" ? "This is the separate, Git-versioned destination where ProvenArch writes architecture knowledge." : "Add read-only inputs, choose their scope, then save and validate before continuing."}</span>
      </div>

      {setupView === "workspace" ? (
        <section className="workspace-purpose" aria-labelledby="workspace-purpose-title">
          <div>
            <span className="eyebrow">Output boundary</span>
            <h2 id="workspace-purpose-title">Your source code stays untouched</h2>
            <p>Repositories are evidence inputs. Reports, models, findings and proposals are written only to this architecture workspace.</p>
          </div>
          <dl className="compact-defs">
            <div><dt>Versioning</dt><dd>Ordinary Git-reviewable files</dd></div>
            <div><dt>Repository access</dt><dd>Read-only during analysis</dd></div>
          </dl>
        </section>
      ) : null}

      {sourceRecovery ? <SourceValidationRecovery issue={sourceRecovery} /> : null}

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
            <RepoAnalysisScopeFields
              repoId={`guided-${repo.id}`}
              include={repo.analysis_include}
              exclude={repo.analysis_exclude}
              onIncludeChange={(value) => onRepoChange(repo.id, { analysis_include: value })}
              onExcludeChange={(value) => onRepoChange(repo.id, { analysis_exclude: value })}
            />
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
          Preview workspace.yaml draft
        </button>
        <button type="button" onClick={onSaveGuidedWorkspaceSetup} disabled={busy} data-testid="workspace-save-btn">
          Save and validate sources
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
        <button type="button" onClick={onSaveManifest} disabled={busy} data-testid="workspace-raw-save-btn">
          Save raw workspace.yaml
        </button>
      </details>
      {manifestStatus ? <p className={manifestStatus.includes("unsaved") ? "status warn" : "status ok"}>{manifestStatus}</p> : null}
      <WorkspaceValidationResult validateResult={validateResult} validationDiagnosticsByRepo={validationDiagnosticsByRepo} />
      {doctorStatus ? <p className="status">{doctorStatus}</p> : null}
      {doctorResult ? <DoctorChecklist doctorResult={doctorResult} /> : null}
    </section>
  );
}

function SourceValidationRecovery({ issue }: { issue: SourceRecoveryIssue }) {
  return (
    <section className="source-recovery-panel" data-testid="source-validation-recovery">
      <div className="section-heading-row">
        <div>
          <h2>Source validation recovery</h2>
          <p className="hint">Resolve the blocking repository/source diagnostic, save the workspace setup, then validate again before Readiness.</p>
        </div>
        <StatusBadge tone={issue.level === "warning" ? "warn" : "error"}>{issue.level === "warning" ? "source warning" : "source blocked"}</StatusBadge>
      </div>
      <div className="source-recovery-grid">
        <div>
          <span className="metric-label">Affected scope</span>
          <strong>{issue.repoKey}</strong>
        </div>
        <div>
          <span className="metric-label">Diagnostic</span>
          <strong>{issue.diagnosticLabel}</strong>
        </div>
        <div>
          <span className="metric-label">Source type</span>
          <strong>{issue.sourceType}</strong>
        </div>
        <div>
          <span className="metric-label">Ref</span>
          <strong>{issue.refValue}</strong>
        </div>
      </div>
      <dl className="compact-defs source-recovery-detail">
        <div>
          <dt>Message</dt>
          <dd>{issue.message}</dd>
        </div>
        <div>
          <dt>Suggested fix</dt>
          <dd>{issue.suggestion}</dd>
        </div>
        <div>
          <dt>Current source</dt>
          <dd>{issue.sourceValue}</dd>
        </div>
      </dl>
      <ul className="analysis-next-actions">
        <li>Fix the highlighted repository name, source URL/path, ref or local authentication.</li>
        <li>Use Save and validate sources after the edit so `workspace.yaml` and resolved repos update together.</li>
        <li>Move to Readiness only after Source shows the repository as resolved.</li>
      </ul>
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
  selectedRunErrorCode?: string | null;
  selectedRunError?: string | null;
  onSetupRuntimeChange: (value: string) => void;
  onSetupRuntimeProviderChange: (value: string) => void;
  onValidateWorkspace: () => void;
  onCheckDoctor: () => void;
  onRunFirstAnalysis: () => void;
  runtimeSettingsPanel: ReactNode;
  artifactCount: number;
  workspaceHealthReport: WorkspaceHealthResponse | null;
  workspaceHealthStatus: "idle" | "loading" | "loaded" | "error";
  workspaceHealthError: string;
  onRefreshWorkspaceHealth: () => void;
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
  selectedRunErrorCode,
  selectedRunError,
  onSetupRuntimeChange,
  onSetupRuntimeProviderChange,
  onValidateWorkspace,
  onCheckDoctor,
  onRunFirstAnalysis,
  runtimeSettingsPanel,
  artifactCount,
  workspaceHealthReport,
  workspaceHealthStatus,
  workspaceHealthError,
  onRefreshWorkspaceHealth,
  runtimeTimeoutEffective,
  runtimeExecutionEffective,
  runtimePermissionEffective,
  runtimeStepProviderEffective,
}: ReadinessStageProps) {
  const validated = validateResult?.ok === true;
  const localReady = doctorResult?.ok === true;
  const runtimeCheck = doctorResult?.checks.find((check) => check.id === "runtime_provider");
  const showProviderRecovery = isRunnerUnavailable(selectedRunErrorCode) || setupRuntime === "headless" || runtimeCheck?.status === "fail";
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

      <WorkspaceHealthSummary
        busy={busy}
        report={workspaceHealthReport}
        status={workspaceHealthStatus}
        error={workspaceHealthError}
        onRefresh={onRefreshWorkspaceHealth}
      />

      <div className="stage-local-next-action" data-testid="readiness-next-action">
        <strong>Next in Readiness</strong>
        <span>
          {!validated
            ? "Validate workspace before checking local runtime readiness."
            : !localReady
              ? "Check local readiness before first analysis."
              : "Readiness gates are clear; run first analysis when you are ready to generate evidence."}
        </span>
      </div>

      <RuntimeProfileSummary
        setupRuntime={setupRuntime}
        runtimeTimeoutEffective={runtimeTimeoutEffective}
        runtimeExecutionEffective={runtimeExecutionEffective}
        runtimePermissionEffective={runtimePermissionEffective}
        runtimeStepProviderEffective={runtimeStepProviderEffective}
      />

      {showProviderRecovery ? (
        <ProviderReadinessRecovery
          setupRuntime={setupRuntime}
          setupRuntimeProvider={setupRuntimeProvider}
          runtimeCheck={runtimeCheck}
          selectedRunErrorCode={selectedRunErrorCode}
          selectedRunError={selectedRunError}
        />
      ) : null}

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
        <button
          type="button"
          onClick={onRunFirstAnalysis}
          disabled={busy || !validated || !localReady}
          title={validated && !localReady ? "Check local readiness before first analysis." : undefined}
          data-testid="setup-run-first-btn"
        >
          Run first analysis
        </button>
      </div>
      {validated && !localReady ? <p className="status warn">Check local readiness before first analysis.</p> : null}

      <WorkspaceValidationResult validateResult={validateResult} validationDiagnosticsByRepo={validationDiagnosticsByRepo} />
      {doctorStatus ? <p className="status">{doctorStatus}</p> : null}
      {doctorResult ? <DoctorChecklist doctorResult={doctorResult} /> : null}
      {firstRunStatus ? <p className="status ok">{firstRunStatus}</p> : null}

      <details className="advanced-block readiness-advanced-settings" data-testid="readiness-advanced-settings">
        <summary>
          <span className="advanced-summary-copy">
            <strong>Advanced runtime settings</strong>
            <span>Timeouts, execution policy, permissions, and per-step provider overrides.</span>
          </span>
          <StatusBadge tone="info">operator tools</StatusBadge>
        </summary>
        {runtimeSettingsPanel}
      </details>
    </section>
  );
}

function ProviderReadinessRecovery({
  setupRuntime,
  setupRuntimeProvider,
  runtimeCheck,
  selectedRunErrorCode,
  selectedRunError,
}: {
  setupRuntime: string;
  setupRuntimeProvider: string;
  runtimeCheck?: DoctorResponse["checks"][number];
  selectedRunErrorCode?: string | null;
  selectedRunError?: string | null;
}) {
  const providerLabel = runtimeDisplayLabel(setupRuntime, setupRuntimeProvider, { compact: true });
  const providerCommand = providerCommandHint(setupRuntimeProvider);
  const envOverride = providerCommandEnv(setupRuntimeProvider);
  const doctorStatus = runtimeCheck ? `${runtimeCheck.label}: ${runtimeCheck.status}` : "not checked";
  const lastRunBlocker = isRunnerUnavailable(selectedRunErrorCode) ? "runner_unavailable" : "none selected";
  const readinessMessage = runtimeCheck?.message || selectedRunError || selectedRunErrorCode || "";
  const guidance = providerReadinessGuidance(setupRuntimeProvider, readinessMessage);
  const summary = isRunnerUnavailable(selectedRunErrorCode)
    ? "The selected run stopped because provider/tool availability failed. Confirm the provider command, auth/quota and runtime mode before retrying."
    : runtimeCheck?.status === "fail"
      ? "The runtime provider doctor check is failing. Fix the provider command, auth or quota before starting analysis."
      : "Headless provider mode is selected. Run local readiness after provider changes before starting analysis.";

  return (
    <section className="provider-recovery-panel" data-testid="provider-readiness-recovery">
      <div className="section-heading-row">
        <div>
          <h2>Provider readiness recovery</h2>
          <p className="hint">{summary}</p>
        </div>
        <StatusBadge tone={runtimeCheck?.status === "pass" ? "ok" : "warn"}>{runtimeCheck?.status === "pass" ? "provider ready" : "provider check"}</StatusBadge>
      </div>
      <div className="provider-recovery-grid">
        <div>
          <span className="metric-label">Selected provider</span>
          <strong>{providerLabel}</strong>
        </div>
        <div>
          <span className="metric-label">Doctor check</span>
          <strong>{doctorStatus}</strong>
        </div>
        <div>
          <span className="metric-label">Command override</span>
          <strong>{envOverride}</strong>
        </div>
        <div>
          <span className="metric-label">Failure mode</span>
          <strong>{guidance.failureMode}</strong>
        </div>
        <div>
          <span className="metric-label">Probe stage</span>
          <strong>{guidance.probeStage}</strong>
        </div>
        <div>
          <span className="metric-label">Last run blocker</span>
          <strong>{lastRunBlocker}</strong>
        </div>
      </div>
      <dl className="compact-defs provider-recovery-detail">
        <div>
          <dt>Expected command</dt>
          <dd>{providerCommand}</dd>
        </div>
        <div>
          <dt>Doctor message</dt>
          <dd>{readinessMessage || "Run local readiness to check provider availability."}</dd>
        </div>
        <div>
          <dt>Operator focus</dt>
          <dd>{guidance.operatorFocus}</dd>
        </div>
        {runtimeCheck?.suggestion ? (
          <div>
            <dt>Suggested fix</dt>
            <dd>{runtimeCheck.suggestion}</dd>
          </div>
        ) : null}
      </dl>
      <ul className="analysis-next-actions">
        {guidance.nextActions.map((action) => (
          <li key={action}>{action}</li>
        ))}
      </ul>
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
        <table className="run-table source-table responsive-card-table">
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
                  <td data-label="Name">
                    <strong>{repo.name || "unnamed repo"}</strong>
                  </td>
                  <td data-label="Source">
                    <span className="source-mode-label">{repo.mode === "path" ? "Local" : "Git URL"}</span>
                    <code>{sourceValue}</code>
                  </td>
                  <td data-label="Ref">{repo.ref || resolved?.ref || "current/default"}</td>
                  <td data-label="Scope">
                    <span className="status">{analysisScopeSummary(repo.analysis_include, repo.analysis_exclude)}</span>
                  </td>
                  <td data-label="Status">
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
        status={runtimeDisplayLabel(setupRuntime, setupRuntimeProvider, { compact: true })}
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

function WorkspaceHealthSummary({
  busy,
  report,
  status,
  error,
  onRefresh,
}: {
  busy: boolean;
  report: WorkspaceHealthResponse | null;
  status: "idle" | "loading" | "loaded" | "error";
  error: string;
  onRefresh: () => void;
}) {
  const tone = workspaceHealthTone(report, status);
  return (
    <section className="status-block" data-testid="workspace-health-summary">
      <div className="section-heading-row">
        <h2>Workspace health</h2>
        <StatusBadge tone={tone}>{workspaceHealthLabel(report, status)}</StatusBadge>
      </div>
      <p className="hint">Read-only snapshot over published workspace artifacts. It does not block run, review, publish, or Q&amp;A flows.</p>
      <div className="actions compact-actions">
        <button type="button" onClick={onRefresh} disabled={busy || status === "loading"} data-testid="workspace-health-refresh-btn">
          Refresh health
        </button>
      </div>
      {status === "loading" ? <p className="status">Workspace health scan running.</p> : null}
      {status === "error" ? <p className="status err">Workspace health scan failed: {error || "scan failed"}</p> : null}
      {status !== "loading" && status !== "error" && !report ? <p className="hint">Workspace health not available.</p> : null}
      {report ? (
        <>
          <p className={report.status === "fail" ? "status err" : report.status === "warn" ? "status warn" : "status ok"}>
            {report.items.length === 0
              ? "No health findings."
              : `${report.summary.error} error(s), ${report.summary.warning} warning(s), ${report.summary.info} info item(s).`}
          </p>
          {report.items.length > 0 ? (
            <ul className="compact-list" data-testid="workspace-health-items">
              {report.items.slice(0, 6).map((item) => (
                <li key={`${item.id}-${item.path}`}>
                  <span className={item.severity === "error" ? "status err" : item.severity === "warning" ? "status warn" : "status"}>
                    {item.id}
                  </span>{" "}
                  {item.title}
                  {item.path ? (
                    <>
                      {" "}
                      <code>{item.path}</code>
                    </>
                  ) : null}
                </li>
              ))}
            </ul>
          ) : null}
        </>
      ) : null}
    </section>
  );
}

function RuntimeProfileSummary({
  setupRuntime,
  runtimeTimeoutEffective,
  runtimeExecutionEffective,
  runtimePermissionEffective,
  runtimeStepProviderEffective,
}: {
  setupRuntime: string;
  runtimeTimeoutEffective: RuntimeTimeoutValues;
  runtimeExecutionEffective: RuntimeExecutionValues;
  runtimePermissionEffective: RuntimePermissionValues;
  runtimeStepProviderEffective: Partial<RuntimeStepProviderValues>;
}) {
  const providerValues = Object.values(runtimeStepProviderEffective).filter(Boolean);
  const uniqueProviders = [...new Set(providerValues)];
  const providerSummary = setupRuntime === "fake" ? "fake" : uniqueProviders.length > 0 ? uniqueProviders.join(", ") : "default provider";
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
          <strong>{providerSummary}</strong>
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

