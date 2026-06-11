import { Suspense, lazy, useCallback, useEffect, useRef, useState, type ComponentProps, type ReactNode, type RefObject } from "react";

import { BaselineEditorsPanel } from "./BaselineEditorsPanel";
import { BaselineGitPanel } from "./BaselineGitPanel";
import { RuntimeProfileSettingsPanel } from "./RuntimeProfileSettingsPanel";
import { RunStatusPanel } from "./RunStatusPanel";
import { ArtifactPathButton, StatusBadge } from "./ConsolePrimitives";
import { getQARun, listQARuns, startQAQuestion, type QARunResponse } from "../lib/qaApi";
import { formatTimestamp, parseTimeOrMin } from "../lib/runState";
import type {
  Artifact,
  Diagnostic,
  DoctorResponse,
  EditableArtifactOption,
  GitDiffResponse,
  GuidedRepo,
  RepoSourceMode,
  ReviewQueueItem,
  RuntimeExecutionValues,
  RuntimePermissionValues,
  RuntimePermissionRequest,
  RuntimeStepProviderValues,
  RuntimeTimeoutValues,
  RunListItem,
  RunLogEntry,
  RunReviewStep,
  RunReviewSummaryResponse,
  RunStatusResponse,
  ValidateResponse,
} from "../lib/appContracts";
import type { LoadGitDiffOptions } from "../lib/gitDiffApi";

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
  onSaveGuidedWorkspaceSetup: () => void;
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
  onSaveGuidedWorkspaceSetup,
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

      <div className="stage-local-next-action" data-testid="source-next-action">
        <strong>Next in Source</strong>
        <span>Keep this screen focused on repository inventory, refs and imports; then save and validate before readiness gates.</span>
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
  const localReady = doctorResult?.ok === true;
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

      <div className="stage-local-next-action" data-testid="readiness-next-action">
        <strong>Next in Readiness</strong>
        <span>
          {!validated
            ? "Validate workspace before checking local runtime readiness."
            : !localReady
              ? "Run local readiness checks before starting analysis."
              : "Readiness gates are clear; run first analysis when you are ready to generate evidence."}
        </span>
      </div>

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
        <button
          type="button"
          onClick={onRunFirstAnalysis}
          disabled={busy || !validated || !localReady}
          title={validated && !localReady ? "Run local readiness before starting analysis." : undefined}
          data-testid="setup-run-first-btn"
        >
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
  runReviewSummary: RunReviewSummaryResponse | null;
  runReviewStatus: string;
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  onLoadGitDiff: (options: LoadGitDiffOptions) => void;
  focusBlockerSignal: number;
  onRunPipeline: (pipeline: "init" | "refresh") => void;
  onCancelSelectedRun: () => void;
  onSelectRun: (runId: string) => void;
  onOpenArtifact: (path: string) => void;
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
  runReviewSummary,
  runReviewStatus,
  gitDiff,
  gitDiffStatus,
  onLoadGitDiff,
  focusBlockerSignal,
  onRunPipeline,
  onCancelSelectedRun,
  onSelectRun,
  onOpenArtifact,
}: AnalysisStageProps) {
  const blockerDetailsRef = useRef<HTMLElement>(null);
  const [selectedStepID, setSelectedStepID] = useState("");
  const [stepReviewView, setStepReviewView] = useState<"artifacts" | "logs" | "evidence" | "diff">("artifacts");
  const stepTimeline = buildAnalysisStepTimeline(runStatus, runLogs);
  const reviewSteps = runReviewSummary?.steps ?? [];
  const preferredReviewStep =
    reviewSteps.find((step) => step.state === "failed") ??
    reviewSteps.find((step) => step.state === "active") ??
    reviewSteps.find((step) => step.artifact_count > 0) ??
    reviewSteps[0];
  const selectedReviewStep =
    reviewSteps.find((step) => step.step_id === selectedStepID) ??
    preferredReviewStep ??
    null;
  const shardRows = buildAnalysisShardRows(runStatus, runLogs, artifacts, setupRuntime, setupRuntimeProvider);
  const issueRows = shardRows.filter((row) => row.status === "failed" || row.status === "warning");
  const blockerRows = shardRows.filter((row) => row.status === "failed");
  const runtimeLabel = setupRuntime === "fake" ? "fake" : `${setupRuntime}/${setupRuntimeProvider}`;

  const focusBlockerDetails = useCallback(() => {
    blockerDetailsRef.current?.scrollIntoView?.({ block: "center" });
    blockerDetailsRef.current?.focus();
  }, []);

  useEffect(() => {
    if (focusBlockerSignal <= 0) {
      return;
    }
    focusBlockerDetails();
  }, [focusBlockerDetails, focusBlockerSignal]);

  useEffect(() => {
    if (preferredReviewStep && !reviewSteps.some((step) => step.step_id === selectedStepID)) {
      setSelectedStepID(preferredReviewStep.step_id);
    }
  }, [preferredReviewStep, reviewSteps, selectedStepID]);

  useEffect(() => {
    if (selectedReviewStep?.step_id && stepReviewView === "diff") {
      onLoadGitDiff({ stepId: selectedReviewStep.step_id });
    }
  }, [onLoadGitDiff, selectedReviewStep?.step_id, stepReviewView]);

  function handleReviewBlocker() {
    focusBlockerDetails();
  }

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
        onReviewBlocker={handleReviewBlocker}
      />
      <AnalysisRunTimeline steps={stepTimeline} />
      <AnalysisStepReview
        steps={reviewSteps}
        selectedStep={selectedReviewStep}
        runReviewStatus={runReviewStatus}
        runLogs={runLogs}
        gitDiff={gitDiff}
        gitDiffStatus={gitDiffStatus}
        view={stepReviewView}
        onViewChange={setStepReviewView}
        onSelectStep={(stepID) => {
          setSelectedStepID(stepID);
          setStepReviewView("artifacts");
        }}
        onOpenArtifact={onOpenArtifact}
        onLoadGitDiff={onLoadGitDiff}
      />
      <AnalysisShardTable rows={shardRows} />
      <AnalysisFailedShardDetails rows={issueRows} detailsRef={blockerDetailsRef} />

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
  duration: string;
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

function AnalysisStepReview({
  steps,
  selectedStep,
  runReviewStatus,
  runLogs,
  gitDiff,
  gitDiffStatus,
  view,
  onViewChange,
  onSelectStep,
  onOpenArtifact,
  onLoadGitDiff,
}: {
  steps: RunReviewStep[];
  selectedStep: RunReviewStep | null;
  runReviewStatus: string;
  runLogs: RunLogEntry[];
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  view: "artifacts" | "logs" | "evidence" | "diff";
  onViewChange: (view: "artifacts" | "logs" | "evidence" | "diff") => void;
  onSelectStep: (stepID: string) => void;
  onOpenArtifact: (path: string) => void;
  onLoadGitDiff: (options: LoadGitDiffOptions) => void;
}) {
  const stepLogs = selectedStep ? runLogs.filter((entry) => stepMatches(entry.step_id || "", selectedStep.step_id) || stepMatches(entry.taskrun_path || "", selectedStep.key)) : [];
  return (
    <section className="analysis-step-review" data-testid="analysis-step-review-panel">
      <div className="section-heading-row">
        <div>
          <h2>Step review</h2>
          <p className="hint">Review step-level artifacts, logs, evidence and workspace diff without waiting for final publication.</p>
        </div>
        <StatusBadge tone={selectedStepTone(selectedStep?.state)}>{selectedStep?.state ?? "empty"}</StatusBadge>
      </div>
      {runReviewStatus ? <p className="status warn">{runReviewStatus}</p> : null}
      {steps.length === 0 ? (
        <p className="empty-state">No review summary is available for the selected run yet. Status and logs still appear below.</p>
      ) : (
        <>
          <div className="analysis-step-card-grid">
            {steps.map((step, index) => (
              <button
                type="button"
                key={step.step_id}
                className={`analysis-step-review-card ${step.state}${selectedStep?.step_id === step.step_id ? " is-selected" : ""}`}
                data-testid="analysis-step-review-card"
                onClick={() => onSelectStep(step.step_id)}
                aria-pressed={selectedStep?.step_id === step.step_id}
              >
                <span className="step-index">{index}</span>
                <strong>{step.label}</strong>
                <code>{step.step_id}</code>
                <span>{step.provider || "provider pending"}</span>
                <span>
                  {step.artifact_count} artifacts · {step.warnings_count}/{step.errors_count} warn/error
                </span>
              </button>
            ))}
          </div>

          <div className="step-review-tabs" role="tablist" aria-label="Step review tabs">
            {(["artifacts", "logs", "evidence", "diff"] as const).map((tab) => (
              <button
                type="button"
                key={tab}
                role="tab"
                aria-selected={view === tab}
                className={view === tab ? "is-active" : ""}
                data-testid={`analysis-step-tab-${tab}`}
                onClick={() => {
                  onViewChange(tab);
                  if (tab === "diff" && selectedStep?.step_id) {
                    onLoadGitDiff({ stepId: selectedStep.step_id });
                  }
                }}
              >
                {capitalize(tab)}
              </button>
            ))}
          </div>

          <div className="step-review-body">
            {view === "artifacts" ? (
              selectedStep && selectedStep.artifact_paths.length > 0 ? (
                <ul className="compact-list">
                  {selectedStep.artifact_paths.map((path) => (
                    <li key={path}>
                      <button type="button" className="link-button" onClick={() => onOpenArtifact(path)}>
                        {path}
                      </button>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="empty-state">No artifacts have been attached to this step yet. Logs may still be streaming.</p>
              )
            ) : null}

            {view === "logs" ? (
              stepLogs.length > 0 ? (
                <ul className="compact-list">
                  {stepLogs.slice(-8).map((entry) => (
                    <li key={entry.cursor}>
                      <span>
                        {entry.level.toUpperCase()} · {entry.step_id || selectedStep?.step_id}
                      </span>
                      <code>{entry.message}</code>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="empty-state">{selectedStep?.state === "active" ? "Provider is silent; still waiting for logs." : "No logs are available for this step."}</p>
              )
            ) : null}

            {view === "evidence" ? (
              <div className="step-evidence-summary">
                <dl className="compact-defs">
                  <div>
                    <dt>Last message</dt>
                    <dd>{selectedStep?.last_message || "No step message yet."}</dd>
                  </div>
                  <div>
                    <dt>Taskrun refs</dt>
                    <dd>{selectedStep?.taskrun_paths.join(", ") || "No taskrun refs."}</dd>
                  </div>
                </dl>
              </div>
            ) : null}

            {view === "diff" ? <GitDiffView gitDiff={gitDiff} status={gitDiffStatus} onSelectFile={(path) => onLoadGitDiff({ path, stepId: selectedStep?.step_id })} /> : null}
          </div>
        </>
      )}
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
                <th>Duration</th>
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
                  <td data-label="Duration">{row.duration}</td>
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

function GitDiffView({
  gitDiff,
  status,
  onSelectFile,
}: {
  gitDiff: GitDiffResponse | null;
  status: string;
  onSelectFile: (path: string) => void;
}) {
  if (!gitDiff) {
    return <p className="empty-state">{status || "Workspace Git diff is not loaded yet."}</p>;
  }
  const selected = gitDiff.selected_file;
  return (
    <div className="git-diff-view" data-testid="git-diff-view">
      {status ? <p className={gitDiff.empty ? "status ok" : "status warn"}>{status}</p> : null}
      {gitDiff.empty ? (
        <p className="empty-state">No workspace Git changes. Generated artifacts may already be committed or this run has not produced publishable file changes yet.</p>
      ) : null}
      <div className="git-diff-layout">
        <aside className="git-diff-file-list" aria-label="changed files">
          <div className="section-heading-row">
            <h3>Changed files</h3>
            <StatusBadge tone={gitDiff.files.length > 0 ? "ok" : "info"}>{gitDiff.files.length}</StatusBadge>
          </div>
          {gitDiff.folders.length > 0 ? (
            <div className="git-diff-folder-summary">
              {gitDiff.folders.map((folder) => (
                <span key={folder.folder}>
                  {folder.folder}: {folder.files} files, +{folder.additions}/-{folder.deletions}
                </span>
              ))}
            </div>
          ) : null}
          {gitDiff.files.length === 0 ? (
            <p className="hint">No changed files match the current filter.</p>
          ) : (
            <ul>
              {gitDiff.files.slice(0, 40).map((file) => (
                <li key={file.path}>
                  <button
                    type="button"
                    className={`git-diff-file${selected?.path === file.path ? " is-selected" : ""}`}
                    onClick={() => onSelectFile(file.path)}
                    aria-pressed={selected?.path === file.path}
                  >
                    <span>
                      <StatusBadge tone={diffFileTone(file.status)}>{file.status}</StatusBadge>
                      {file.path}
                    </span>
                    <code>
                      +{file.additions} / -{file.deletions}
                      {file.binary ? " / binary" : ""}
                    </code>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </aside>
        <section className="git-diff-hunks" data-testid="git-diff-hunks">
          <div className="section-heading-row">
            <h3>{selected?.path || gitDiff.selected_path || "No file selected"}</h3>
            <StatusBadge tone={selected ? diffFileTone(selected.status) : "info"}>{selected?.status ?? "none"}</StatusBadge>
          </div>
          {gitDiff.message ? <p className="hint">{gitDiff.message}</p> : null}
          {selected?.binary ? <p className="empty-state">Binary/non-text diff. Review the file path and status, then use Git tooling for binary inspection.</p> : null}
          {!selected?.binary && gitDiff.hunks.length === 0 ? <p className="empty-state">No line-level hunks for the selected file.</p> : null}
          {gitDiff.hunks.map((hunk, index) => (
            <div className="git-diff-hunk" key={`${hunk.header}-${index}`}>
              <div className="git-diff-hunk-header">{hunk.header}</div>
              {hunk.lines.map((line, lineIndex) => (
                <div className={`git-diff-line ${line.kind}`} key={`${hunk.header}-${lineIndex}`}>
                  <span className="git-diff-line-number">{line.old_line ?? ""}</span>
                  <span className="git-diff-line-number">{line.new_line ?? ""}</span>
                  <code>{line.content || " "}</code>
                </div>
              ))}
            </div>
          ))}
        </section>
      </div>
    </div>
  );
}

function AnalysisFailedShardDetails({ rows, detailsRef }: { rows: AnalysisShardRow[]; detailsRef: RefObject<HTMLElement> }) {
  return (
    <section className="subsection" data-testid="analysis-failed-shard-details" ref={detailsRef} tabIndex={-1}>
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
      duration: durationFromLogFields(last?.fields),
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
      duration: "Duration unavailable",
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

function selectedStepTone(state?: RunReviewStep["state"]): "info" | "ok" | "warn" | "error" {
  if (state === "done") {
    return "ok";
  }
  if (state === "failed") {
    return "error";
  }
  if (state === "active") {
    return "warn";
  }
  return "info";
}

function diffFileTone(status?: string): "info" | "ok" | "warn" | "error" {
  if (status === "deleted") {
    return "error";
  }
  if (status === "modified" || status === "renamed" || status === "changed") {
    return "warn";
  }
  if (status === "new" || status === "untracked" || status === "copied") {
    return "ok";
  }
  return "info";
}

function capitalize(value: string): string {
  return value.slice(0, 1).toUpperCase() + value.slice(1);
}

function fieldString(fields: Record<string, unknown> | undefined, key: string): string {
  const value = fields?.[key];
  return typeof value === "string" && value.trim().length > 0 ? value.trim() : "";
}

function durationFromLogFields(fields: Record<string, unknown> | undefined): string {
  const direct = fieldString(fields, "duration") || fieldString(fields, "elapsed") || fieldString(fields, "runtime_duration");
  if (direct) {
    return direct;
  }
  const millis = numericField(fields, "duration_ms") ?? numericField(fields, "elapsed_ms") ?? numericField(fields, "runtime_duration_ms");
  if (millis !== undefined) {
    return formatDurationMillis(millis);
  }
  const seconds = numericField(fields, "duration_sec") ?? numericField(fields, "elapsed_sec") ?? numericField(fields, "runtime_duration_sec");
  if (seconds !== undefined) {
    return formatDurationMillis(seconds * 1000);
  }
  return "Duration unavailable";
}

function numericField(fields: Record<string, unknown> | undefined, key: string): number | undefined {
  const value = fields?.[key];
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim()) {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
  }
  return undefined;
}

function formatDurationMillis(milliseconds: number): string {
  if (milliseconds < 1000) {
    return `${Math.round(milliseconds)}ms`;
  }
  const totalSeconds = Math.round(milliseconds / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes === 0) {
    return `${seconds}s`;
  }
  return `${minutes}m ${seconds.toString().padStart(2, "0")}s`;
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
  runId: string | null;
  runStatus: RunStatusResponse | null;
  runList: RunListItem[];
  coverageSummary: string;
  openQuestions: string;
  nonDiagramArtifacts: Artifact[];
  diagramArtifacts: Artifact[];
  selectedArtifact: string;
  selectedArtifactContent: string;
  selectedArtifactIsMermaid: boolean;
  runLogs: RunLogEntry[];
  reviewSummary: RunReviewSummaryResponse | null;
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  onLoadGitDiff: (options: LoadGitDiffOptions) => void;
  onSelectRun: (id: string) => void;
  onOpenArtifact: (path: string) => void;
};

export function ReviewStagePanel({
  runId,
  runStatus,
  runList,
  coverageSummary,
  openQuestions,
  nonDiagramArtifacts,
  diagramArtifacts,
  selectedArtifact,
  selectedArtifactContent,
  selectedArtifactIsMermaid,
  runLogs,
  reviewSummary,
  gitDiff,
  gitDiffStatus,
  onLoadGitDiff,
  onSelectRun,
  onOpenArtifact,
}: ReviewStageProps) {
  const [reviewView, setReviewView] = useState<"evidence" | "domain-map">("evidence");
  const overviewArtifact = nonDiagramArtifacts.find((artifact) => artifact.path === "reports/as-is/overview.md");
  const coverageArtifact = nonDiagramArtifacts.find((artifact) => artifact.path === "reports/coverage/summary.md");
  const findingsArtifact = nonDiagramArtifacts.find((artifact) => artifact.path.startsWith("reports/findings/"));
  const allArtifacts = [...nonDiagramArtifacts, ...diagramArtifacts];
  const preferredReviewArtifact =
    overviewArtifact ??
    coverageArtifact ??
    allArtifacts[0];
  const artifactGroups = groupArtifactsByFolder(allArtifacts);
  const reviewQueue = buildReviewQueue({
    artifacts: allArtifacts,
    openQuestions,
    coverageSummary,
  });
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
  const lastSuccessfulRun = findLastSuccessfulRun(runList, runId);
  function handleOpenDomainMapArtifact(path: string) {
    onOpenArtifact(path);
    setReviewView("evidence");
  }

  useEffect(() => {
    if (reviewView === "evidence" && !selectedArtifact && preferredReviewArtifact) {
      onOpenArtifact(preferredReviewArtifact.path);
    }
  }, [onOpenArtifact, preferredReviewArtifact, reviewView, selectedArtifact]);

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
        {runStatus?.status === "failed" && lastSuccessfulRun ? (
          <section className="review-run-recovery" data-testid="review-run-recovery">
            <div>
              <strong>Latest selected run failed before complete evidence was available.</strong>
              <span>
                Review is showing {allArtifacts.length} artifact refs from {runId ?? "the failed run"}. Last successful run{" "}
                <code>{lastSuccessfulRun.run_id}</code> is available for artifact review.
              </span>
            </div>
            <button type="button" onClick={() => onSelectRun(lastSuccessfulRun.run_id)}>
              Open last successful artifacts
            </button>
          </section>
        ) : null}
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
            runLogs={runLogs}
            reviewSummary={reviewSummary}
            reviewQueue={reviewQueue}
            gitDiff={gitDiff}
            gitDiffStatus={gitDiffStatus}
            onLoadGitDiff={onLoadGitDiff}
            onOpenArtifact={onOpenArtifact}
          />
        )}
      </section>
    </div>
  );
}

function findLastSuccessfulRun(runList: RunListItem[], currentRunID: string | null): RunListItem | null {
  const successfulRuns = runList
    .filter((run) => run.status === "succeeded" && run.run_id !== currentRunID)
    .sort((left, right) => parseTimeOrMin(right.started_at) - parseTimeOrMin(left.started_at));
  return successfulRuns[0] ?? null;
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
  runLogs,
  reviewSummary,
  reviewQueue,
  gitDiff,
  gitDiffStatus,
  onLoadGitDiff,
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
  runLogs: RunLogEntry[];
  reviewSummary: RunReviewSummaryResponse | null;
  reviewQueue: ReviewQueueItem[];
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  onLoadGitDiff: (options: LoadGitDiffOptions) => void;
  onOpenArtifact: (path: string) => void;
}) {
  const [evidenceView, setEvidenceView] = useState<"preview" | "diff" | "evidence" | "logs">("preview");

  useEffect(() => {
    if (evidenceView === "diff" && selectedArtifact) {
      onLoadGitDiff({ path: selectedArtifact });
    }
  }, [evidenceView, onLoadGitDiff, selectedArtifact]);

  return (
    <div className="review-workbench">
      <aside className="review-artifact-explorer" data-testid="review-artifact-explorer">
        <ReviewQueuePanel queue={reviewQueue} onOpenArtifact={onOpenArtifact} />
        <div className="section-heading-row">
          <h2>Artifact explorer</h2>
          <StatusBadge tone={artifactGroups.length > 0 ? "ok" : "info"}>{artifactGroups.length} groups</StatusBadge>
        </div>
        {artifactGroups.length === 0 ? (
          <p className="hint">No selected-run artifacts yet. Run Analysis before evidence review.</p>
        ) : (
          <div className="artifact-group-list" data-testid="results-artifacts-panel">
            {artifactGroups.map((group) => (
              <section key={group.name} className={`artifact-group ${reviewArtifactGroupCategory(group.name)}`}>
                <div className="artifact-group-heading">
                  <h3>{group.name}</h3>
                  <span>{reviewArtifactGroupCategoryLabel(group.name)}</span>
                </div>
                <ul data-testid={group.name === "reports/diagrams" ? "run-diagrams-list" : undefined}>
                  {group.artifacts.map((artifact) => (
                    <li key={`${artifact.kind}-${artifact.path}`}>
                      <ArtifactPathButton
                        path={artifact.path}
                        label={artifact.label}
                        kind={artifact.kind}
                        selected={artifact.path === selectedArtifact}
                        onOpenArtifact={onOpenArtifact}
                      />
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
        <div className="evidence-preview-tabs" role="tablist" aria-label="Artifact workbench tabs">
          {(["preview", "diff", "evidence", "logs"] as const).map((tab) => (
            <button
              type="button"
              key={tab}
              role="tab"
              aria-selected={evidenceView === tab}
              className={evidenceView === tab ? "is-active" : ""}
              onClick={() => setEvidenceView(tab)}
            >
              {capitalize(tab)}
            </button>
          ))}
        </div>
        {evidenceView === "preview" ? (
          selectedArtifactIsMermaid ? (
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
          )
        ) : null}
        {evidenceView === "diff" ? <GitDiffView gitDiff={gitDiff} status={gitDiffStatus} onSelectFile={(path) => onLoadGitDiff({ path })} /> : null}
        {evidenceView === "evidence" ? (
          <div className="review-tab-summary">
            <h3>Decision evidence</h3>
            <dl className="compact-defs">
              <div>
                <dt>Selected artifact</dt>
                <dd>{selectedArtifact || "none"}</dd>
              </div>
              <div>
                <dt>Current run</dt>
                <dd>{reviewSummary?.run_id || "none selected"}</dd>
              </div>
              <div>
                <dt>Review queue</dt>
                <dd>{reviewQueue.length} item(s)</dd>
              </div>
            </dl>
            <p className="hint">{reviewDecisionSummary(trustStatus, openQuestionCount)}</p>
          </div>
        ) : null}
        {evidenceView === "logs" ? (
          <div className="review-tab-summary">
            <h3>Related logs</h3>
            {runLogs.length === 0 ? (
              <p className="empty-state">No logs are loaded for the selected run.</p>
            ) : (
              <ul className="compact-list">
                {runLogs.slice(-8).map((entry) => (
                  <li key={`review-log-${entry.cursor}`}>
                    <span>
                      {entry.level.toUpperCase()} · {entry.step_id || "run"}
                    </span>
                    <code>{entry.message}</code>
                  </li>
                ))}
              </ul>
            )}
          </div>
        ) : null}
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
          <span className="review-decision-summary" data-testid="review-decision-summary">{reviewDecisionSummary(trustStatus, openQuestionCount)}</span>
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

function ReviewQueuePanel({ queue, onOpenArtifact }: { queue: ReviewQueueItem[]; onOpenArtifact: (path: string) => void }) {
  return (
    <section className="review-queue" data-testid="review-queue">
      <div className="section-heading-row">
        <h2>Review Queue</h2>
        <StatusBadge tone={queue.length > 0 ? "warn" : "ok"}>{queue.length}</StatusBadge>
      </div>
      {queue.length === 0 ? (
        <p className="hint">No generated review items are waiting. Run Analysis or select a completed run.</p>
      ) : (
        <ul>
          {queue.slice(0, 10).map((item) => (
            <li key={item.id}>
              <button
                type="button"
                className="review-queue-item"
                aria-label={`Review queue item: ${item.title}`}
                onClick={() => onOpenArtifact(item.path)}
              >
                <StatusBadge tone={item.severity === "error" ? "error" : item.severity === "warn" ? "warn" : "info"}>{item.kind}</StatusBadge>
                <span>{item.title}</span>
                <code>{item.path}</code>
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function buildReviewQueue({
  artifacts,
  openQuestions,
  coverageSummary,
}: {
  artifacts: Artifact[];
  openQuestions: string;
  coverageSummary: string;
}): ReviewQueueItem[] {
  const queue: ReviewQueueItem[] = [];
  const addArtifact = (artifact: Artifact, kind: ReviewQueueItem["kind"], severity: ReviewQueueItem["severity"], title?: string) => {
    queue.push({
      id: `${kind}:${artifact.path}`,
      kind,
      severity,
      title: title || artifact.label || artifact.path,
      path: artifact.path,
    });
  };
  for (const artifact of artifacts) {
    if (artifact.path === "reports/as-is/overview.md") {
      addArtifact(artifact, "report", "info", "Review as-is overview");
    } else if (artifact.path === "reports/coverage/summary.md") {
      addArtifact(artifact, "coverage", coverageSummary.trim() ? "info" : "warn", "Review coverage summary");
    } else if (artifact.path === "reports/coverage/open-questions.md") {
      addArtifact(artifact, "question", openQuestions.trim() ? "warn" : "info", "Review open questions");
    } else if (artifact.path.startsWith("reports/findings/")) {
      addArtifact(artifact, "finding", "warn", "Review findings");
    } else if (artifact.path.startsWith("proposals/")) {
      addArtifact(artifact, "proposal", "info", "Review proposal");
    } else if (artifact.path.startsWith("model/")) {
      addArtifact(artifact, "model", "info", "Inspect derived model");
    } else if (artifact.path.startsWith("reports/diagrams/")) {
      addArtifact(artifact, "diagram", "info", "Inspect diagram");
    }
  }
  return queue
    .sort((left, right) => reviewQueuePriority(left) - reviewQueuePriority(right) || left.path.localeCompare(right.path))
    .slice(0, 16);
}

function reviewQueuePriority(item: ReviewQueueItem): number {
  if (item.kind === "question") {
    return 0;
  }
  if (item.kind === "finding") {
    return 1;
  }
  if (item.kind === "report" || item.kind === "coverage") {
    return 2;
  }
  if (item.kind === "proposal") {
    return 3;
  }
  return 5;
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
    .sort(([left], [right]) => reviewArtifactGroupPriority(left) - reviewArtifactGroupPriority(right) || left.localeCompare(right))
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

function reviewArtifactGroupPriority(name: string): number {
  if (name === "reports/as-is") {
    return 0;
  }
  if (name === "reports/coverage") {
    return 1;
  }
  if (name === "reports/findings") {
    return 2;
  }
  if (name === "reports/diagrams") {
    return 3;
  }
  if (name.startsWith("model/")) {
    return 4;
  }
  if (name === "reports/agent-outputs") {
    return 5;
  }
  if (name === "proposals") {
    return 6;
  }
  if (name === "reports/changelog") {
    return 7;
  }
  if (name === "reports/taskruns") {
    return 8;
  }
  return 20;
}

function reviewArtifactGroupCategory(name: string): string {
  if (name.startsWith("reports/diagrams")) {
    return "is-diagram-group";
  }
  if (name.startsWith("reports/")) {
    return "is-report-group";
  }
  if (name.startsWith("model/")) {
    return "is-model-group";
  }
  if (name === "proposals" || name === "reports/changelog") {
    return "is-proposal-group";
  }
  return "is-supporting-group";
}

function reviewArtifactGroupCategoryLabel(name: string): string {
  if (name.startsWith("reports/diagrams")) {
    return "diagram";
  }
  if (name.startsWith("reports/")) {
    return "report";
  }
  if (name.startsWith("model/")) {
    return "model";
  }
  if (name === "proposals" || name === "reports/changelog") {
    return "proposal";
  }
  return "support";
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

function reviewDecisionSummary(trustStatus: ReviewTrustStatus, openQuestionCount: number): string {
  if (openQuestionCount > 0) {
    return `${openQuestionCount} open question${openQuestionCount === 1 ? "" : "s"} require review, but they are not a hard publish blocker.`;
  }
  if (trustStatus.tone === "ok") {
    return "Evidence can move to proposal or publish review after human confirmation.";
  }
  return "Treat this as partial evidence and inspect generated artifacts before publication.";
}

export function ProposalsStagePanel({
  artifacts,
  selectedArtifact,
  selectedArtifactContent,
  openQuestions,
  proposalBranch,
  gitStatus,
  runLogs,
  gitDiff,
  gitDiffStatus,
  onLoadGitDiff,
  onOpenArtifact,
  onGoPublish,
}: {
  artifacts: Artifact[];
  selectedArtifact: string;
  selectedArtifactContent: string;
  openQuestions: string;
  proposalBranch: string;
  gitStatus: string;
  runLogs: RunLogEntry[];
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  onLoadGitDiff: (options: LoadGitDiffOptions) => void;
  onOpenArtifact: (path: string) => void;
  onGoPublish: () => void;
}) {
  const [proposalView, setProposalView] = useState<"preview" | "evidence" | "changelog" | "diff" | "logs">("preview");
  const proposalReview = deriveProposalReviewModel({ artifacts, openQuestions });
  const selectedProposalArtifact = proposalReview.proposalArtifacts.find((artifact) => artifact.path === selectedArtifact);
  const preferredProposalArtifact =
    proposalReview.proposalArtifacts.find((artifact) => artifact.path.startsWith("proposals/") && /(^|\/)proposal\.md$/i.test(artifact.path)) ??
    proposalReview.proposalArtifacts.find((artifact) => artifact.path.startsWith("proposals/")) ??
    proposalReview.changelogArtifacts[0];
  const selectedProposalIsLoading = selectedArtifactContent === "Loading...";

  useEffect(() => {
    if (!selectedProposalArtifact && preferredProposalArtifact && selectedArtifact !== preferredProposalArtifact.path) {
      onOpenArtifact(preferredProposalArtifact.path);
    }
  }, [onOpenArtifact, preferredProposalArtifact, selectedArtifact, selectedProposalArtifact]);

  useEffect(() => {
    if (proposalView === "diff") {
      onLoadGitDiff({ path: selectedArtifact || preferredProposalArtifact?.path });
    }
  }, [onLoadGitDiff, preferredProposalArtifact?.path, proposalView, selectedArtifact]);

  return (
    <section className="panel stage-panel" data-testid="proposals-panel">
      <div className="stage-header">
        <div>
          <h1>Proposals</h1>
          <p className="hint">Review generated proposal packages, ADR/RFC drafts and iteration changelog.</p>
        </div>
        <StatusBadge tone={proposalReview.proposalArtifacts.length > 0 ? "ok" : "info"}>{proposalReview.proposalArtifacts.length} refs</StatusBadge>
      </div>
      <div className="proposals-review-room" data-testid="proposals-review-room">
        <aside className="proposals-artifact-list" data-testid="proposals-artifact-list">
          <div className="section-heading-row">
            <h2>Proposal packages</h2>
            <StatusBadge tone={proposalReview.proposalArtifacts.length > 0 ? "ok" : "info"}>{proposalReview.packages.length} groups</StatusBadge>
          </div>
          {proposalReview.packages.length === 0 ? (
            <p className="hint">No proposal or changelog artifacts yet. Run Analysis through `step4.proposals` before publication review.</p>
          ) : (
            <div className="proposal-package-list">
              {proposalReview.packages.map((group) => (
                <section className="proposal-package" key={group.name}>
                  <h3>{group.name}</h3>
                  <ul>
                    {group.artifacts.map((artifact) => (
                      <li key={`${artifact.kind}-${artifact.path}`}>
                        <ArtifactPathButton
                          path={artifact.path}
                          label={artifact.label}
                          kind={artifact.kind}
                          actionLabel="Open proposal artifact"
                          selected={artifact.path === selectedArtifact}
                          onOpenArtifact={onOpenArtifact}
                        />
                        <span>{deriveProposalArtifactType(artifact.path)}</span>
                      </li>
                    ))}
                  </ul>
                </section>
              ))}
            </div>
          )}
        </aside>

        <section className="proposal-preview-panel" data-testid="proposal-preview-panel">
          <div className="section-heading-row">
            <div>
              <h2>Proposal review</h2>
              <p className="hint">Inspect proposal text, linked evidence and publication readiness before moving to Publish.</p>
            </div>
            <button type="button" disabled title="Proposal approval persistence is planned for the Publish gate slice.">
              Approve proposal
            </button>
          </div>
          <div className="proposal-preview-tabs" role="tablist" aria-label="Proposal review tabs" data-testid="proposal-preview-tabs">
            {(["preview", "evidence", "changelog", "diff", "logs"] as const).map((view) => (
              <button
                type="button"
                role="tab"
                aria-selected={proposalView === view}
                className={proposalView === view ? "is-active" : ""}
                key={view}
                onClick={() => setProposalView(view)}
              >
                {proposalTabLabel(view)}
              </button>
            ))}
          </div>

          {proposalView === "preview" ? (
            <div className="proposal-tab-panel">
              <h3>{selectedProposalArtifact?.path || "Select a proposal artifact"}</h3>
              <pre>{selectedProposalArtifact ? (selectedProposalIsLoading ? "Loading proposal..." : selectedArtifactContent || "No preview content returned.") : "Select a proposal, ADR, RFC or checklist artifact from the package list."}</pre>
            </div>
          ) : null}

          {proposalView === "evidence" ? (
            <div className="proposal-tab-panel">
              <h3>Linked evidence</h3>
              {proposalReview.evidenceArtifacts.length === 0 ? (
                <p className="hint">No findings, coverage or as-is evidence artifacts are available in the selected run.</p>
              ) : (
                <ul className="proposal-evidence-list">
                  {proposalReview.evidenceArtifacts.map((artifact) => (
                    <li key={`${artifact.kind}-${artifact.path}`}>
                      <ArtifactPathButton path={artifact.path} label={artifact.label} kind={artifact.kind} onOpenArtifact={onOpenArtifact} />
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ) : null}

          {proposalView === "changelog" ? (
            <div className="proposal-tab-panel">
              <h3>Changelog</h3>
              {proposalReview.changelogArtifacts.length === 0 ? (
                <p className="hint">No changelog artifact is available yet. Keep this as a publication blocker or generate proposals again.</p>
              ) : (
                <ul className="proposal-evidence-list">
                  {proposalReview.changelogArtifacts.map((artifact) => (
                    <li key={`${artifact.kind}-${artifact.path}`}>
                      <ArtifactPathButton path={artifact.path} label={artifact.label} kind={artifact.kind} onOpenArtifact={onOpenArtifact} />
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ) : null}

          {proposalView === "diff" ? (
            <div className="proposal-tab-panel">
              <h3>Diff preview</h3>
              <GitDiffView gitDiff={gitDiff} status={gitDiffStatus} onSelectFile={(path) => onLoadGitDiff({ path })} />
            </div>
          ) : null}

          {proposalView === "logs" ? (
            <div className="proposal-tab-panel">
              <h3>Run logs</h3>
              {runLogs.length === 0 ? (
                <p className="empty-state">No logs are loaded for the selected run.</p>
              ) : (
                <ul className="compact-list">
                  {runLogs.slice(-8).map((entry) => (
                    <li key={`proposal-log-${entry.cursor}`}>
                      <span>
                        {entry.level.toUpperCase()} · {entry.step_id || "run"}
                      </span>
                      <code>{entry.message}</code>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ) : null}
        </section>

        <aside className="proposal-quality-panel" data-testid="proposal-quality-panel">
          <div className="section-heading-row">
            <h2>Quality / blockers</h2>
            <StatusBadge tone={proposalReview.blockers.length > 0 ? "warn" : proposalReview.proposalArtifacts.length > 0 ? "ok" : "info"}>
              {proposalReview.blockers.length > 0 ? "review" : proposalReview.proposalArtifacts.length > 0 ? "ready" : "partial"}
            </StatusBadge>
          </div>
          <div className="proposal-quality-grid">
            <div className="metric-tile">
              <span className="metric-label">Proposal docs</span>
              <strong>{proposalReview.proposalDocumentCount}</strong>
            </div>
            <div className="metric-tile">
              <span className="metric-label">ADR/RFC</span>
              <strong>{proposalReview.adrRfcCount}</strong>
            </div>
            <div className="metric-tile">
              <span className="metric-label">Changelog</span>
              <strong>{proposalReview.changelogArtifacts.length}</strong>
            </div>
            <div className="metric-tile">
              <span className="metric-label">Evidence refs</span>
              <strong>{proposalReview.evidenceArtifacts.length}</strong>
            </div>
          </div>
          <div className="proposal-publication-path" data-testid="proposal-publication-path">
            <strong>Publication path</strong>
            <span>{proposalBranch ? `Proposal branch: ${proposalBranch}` : "Proposal branch is not prepared."}</span>
            <span>{gitStatus || "Git action pending; commit and branch actions stay in Publish."}</span>
            <button type="button" onClick={onGoPublish}>
              Review in Publish
            </button>
          </div>
          <section className="proposal-blocker-list">
            <h3>Unresolved blockers</h3>
            {proposalReview.blockers.length === 0 ? (
              <p className="hint">No proposal-specific blockers detected from available artifact refs.</p>
            ) : (
              <ul>
                {proposalReview.blockers.map((blocker) => (
                  <li key={blocker}>{blocker}</li>
                ))}
              </ul>
            )}
          </section>
        </aside>
      </div>
    </section>
  );
}

type ProposalReviewPackage = {
  name: string;
  artifacts: Artifact[];
};

type ProposalReviewModel = {
  proposalArtifacts: Artifact[];
  changelogArtifacts: Artifact[];
  evidenceArtifacts: Artifact[];
  packages: ProposalReviewPackage[];
  proposalDocumentCount: number;
  adrRfcCount: number;
  blockers: string[];
};

function deriveProposalReviewModel({
  artifacts,
  openQuestions,
}: {
  artifacts: Artifact[];
  openQuestions: string;
}): ProposalReviewModel {
  const proposalArtifacts = artifacts
    .filter((artifact) => artifact.path.startsWith("proposals/") || artifact.path.startsWith("reports/changelog/"))
    .sort((left, right) => left.path.localeCompare(right.path));
  const changelogArtifacts = proposalArtifacts.filter((artifact) => artifact.path.startsWith("reports/changelog/"));
  const proposalDocumentArtifacts = proposalArtifacts.filter((artifact) => artifact.path.startsWith("proposals/"));
  const evidenceArtifacts = artifacts
    .filter((artifact) => artifact.path.startsWith("reports/findings/") || artifact.path.startsWith("reports/coverage/") || artifact.path.startsWith("reports/as-is/"))
    .sort((left, right) => left.path.localeCompare(right.path));
  const packages = groupProposalArtifacts(proposalArtifacts);
  const adrRfcCount = proposalDocumentArtifacts.filter((artifact) => /(^|\/)(ADR|RFC)\.md$/i.test(artifact.path)).length;
  const proposalDocumentCount = proposalDocumentArtifacts.filter((artifact) => artifact.path.endsWith(".md")).length;
  const blockers: string[] = [];
  if (proposalDocumentArtifacts.length === 0) {
    blockers.push("No proposal package artifacts are available.");
  }
  if (proposalDocumentArtifacts.length > 0 && adrRfcCount === 0) {
    blockers.push("Proposal package has no ADR or RFC draft artifact.");
  }
  if (proposalDocumentArtifacts.length > 0 && changelogArtifacts.length === 0) {
    blockers.push("No changelog artifact is linked to this proposal run.");
  }
  const openQuestionCount = countMarkdownItems(openQuestions);
  if (openQuestionCount > 0) {
    blockers.push(`${openQuestionCount} open questions remain from evidence review.`);
  }
  return {
    proposalArtifacts,
    changelogArtifacts,
    evidenceArtifacts,
    packages,
    proposalDocumentCount,
    adrRfcCount,
    blockers,
  };
}

function groupProposalArtifacts(artifacts: Artifact[]): ProposalReviewPackage[] {
  const groups = new Map<string, Artifact[]>();
  for (const artifact of artifacts) {
    const name = proposalPackageName(artifact.path);
    groups.set(name, [...(groups.get(name) ?? []), artifact]);
  }
  return Array.from(groups.entries())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([name, groupArtifacts]) => ({ name, artifacts: groupArtifacts.sort((left, right) => left.path.localeCompare(right.path)) }));
}

function proposalPackageName(path: string): string {
  const parts = path.split("/").filter(Boolean);
  if (parts[0] === "proposals" && parts[1]) {
    return `${parts[0]}/${parts[1]}`;
  }
  if (parts[0] === "reports" && parts[1]) {
    return `${parts[0]}/${parts[1]}`;
  }
  return parts[0] || "root";
}

function deriveProposalArtifactType(path: string): string {
  const basename = path.split("/").pop()?.toLowerCase() ?? "";
  if (path.startsWith("reports/changelog/")) {
    return "changelog";
  }
  if (basename === "adr.md") {
    return "ADR";
  }
  if (basename === "rfc.md") {
    return "RFC";
  }
  if (basename.includes("checklist")) {
    return "checklist";
  }
  if (basename.includes("proposal")) {
    return "proposal";
  }
  return "artifact";
}

function proposalTabLabel(view: "preview" | "evidence" | "changelog" | "diff" | "logs"): string {
  if (view === "preview") {
    return "Preview";
  }
  if (view === "evidence") {
    return "Evidence";
  }
  if (view === "changelog") {
    return "Changelog";
  }
  if (view === "diff") {
    return "Diff";
  }
  return "Logs";
}

export function AskStagePanel({
  primaryActionSignal = 0,
  onOpenArtifact,
}: {
  primaryActionSignal?: number;
  onOpenArtifact: (path: string) => void;
}) {
  const [question, setQuestion] = useState("");
  const [qaRun, setQARun] = useState<QARunResponse | null>(null);
  const [runHistory, setRunHistory] = useState<QARunResponse[]>([]);
  const [selectedRunID, setSelectedRunID] = useState<string | null>(null);
  const [historyStatus, setHistoryStatus] = useState("Loading Q&A history.");
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  const [selectedLoading, setSelectedLoading] = useState(false);
  const qaRunActive = qaRun?.status === "queued" || qaRun?.status === "running";
  const citations = qaRun?.citations ?? [];
  const unresolved = qaRun?.unresolved ?? [];
  const confidence = typeof qaRun?.confidence === "number" ? Math.round(qaRun.confidence * 100) : 0;

  useEffect(() => {
    let canceled = false;
    async function loadHistory() {
      try {
        const history = await listQARuns(20);
        if (canceled) {
          return;
        }
        const items = history.items ?? [];
        setRunHistory(items);
        setHistoryStatus(items.length > 0 ? "" : "No Q&A runs yet.");
        if (items[0]?.run_id) {
          setSelectedRunID(items[0].run_id);
          setQARun(items[0]);
          try {
            const detail = await getQARun(items[0].run_id);
            if (!canceled) {
              setQARun(detail);
              setRunHistory((current) => mergeQARunHistory(detail, current, "preserve"));
              setHistoryStatus("");
            }
          } catch (error) {
            if (!canceled) {
              setStatus(error instanceof Error ? error.message : "Q&A run detail failed");
            }
          }
        }
      } catch (error) {
        if (!canceled) {
          setHistoryStatus(error instanceof Error ? error.message : "Q&A history failed to load");
        }
      }
    }
    void loadHistory();
    return () => {
      canceled = true;
    };
  }, []);

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
          setSelectedRunID(next.run_id);
          setRunHistory((current) => mergeQARunHistory(next, current, "preserve"));
          setHistoryStatus("");
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

  async function refreshHistory() {
    setHistoryStatus("Refreshing Q&A history.");
    try {
      const history = await listQARuns(20);
      const items = history.items ?? [];
      const mergedItems = qaRun ? mergeQARunHistory(qaRun, items, "preserve") : items;
      setRunHistory(mergedItems);
      setHistoryStatus(mergedItems.length > 0 ? "" : "No Q&A runs yet.");
    } catch (error) {
      setHistoryStatus(error instanceof Error ? error.message : "Q&A history failed to load");
    }
  }

  async function handleSelectRun(run: QARunResponse) {
    setSelectedRunID(run.run_id);
    setQARun(run);
    setSelectedLoading(true);
    setStatus("");
    try {
      const detail = await getQARun(run.run_id);
      setQARun(detail);
      setRunHistory((current) => mergeQARunHistory(detail, current, "preserve"));
      setHistoryStatus("");
      setStatus(detail.status === "succeeded" ? "Q&A run loaded." : detail.status === "failed" ? "Q&A run failed." : "Q&A run is running.");
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Q&A run detail failed");
    } finally {
      setSelectedLoading(false);
    }
  }

  async function handleAsk() {
    const trimmed = question.trim();
    if (!trimmed) {
      setStatus("Question is required.");
      return;
    }
    setBusy(true);
    setQARun(null);
    setSelectedRunID(null);
    setStatus("");
    try {
      const started = await startQAQuestion(trimmed);
      setStatus("Q&A run queued.");
      setSelectedRunID(started.run_id);
      const detail = await getQARun(started.run_id);
      setQARun(detail);
      setRunHistory((current) => mergeQARunHistory(detail, current));
      setHistoryStatus("");
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

  useEffect(() => {
    if (primaryActionSignal <= 0) {
      return;
    }
    void handleAsk();
  }, [primaryActionSignal]);

  return (
    <section className="panel stage-panel" data-testid="qa-panel">
      <div className="stage-header">
        <div>
          <h1>Ask</h1>
          <p className="hint">Ask agent-backed questions over existing workspace artifacts. Source repos and canonical outputs stay unchanged.</p>
        </div>
        <StatusBadge tone={qaRunStatusTone(qaRun?.status)}>{qaRunProviderLabel(qaRun)}</StatusBadge>
      </div>

      <div className="qa-workbench">
        <aside className="qa-run-history" data-testid="qa-run-history">
          <div className="panel-subheader">
            <div>
              <h2>Run history</h2>
              <p className="hint">Async Q&A over existing workspace evidence.</p>
            </div>
            <button type="button" className="link-button" onClick={() => void refreshHistory()}>
              Refresh
            </button>
          </div>
          {historyStatus ? <p className="hint">{historyStatus}</p> : null}
          {runHistory.length === 0 ? (
            <p className="empty-state">Ask the workspace to create the first read-only Q&A run.</p>
          ) : (
            <div className="qa-history-list" role="list">
              {runHistory.map((run) => (
                <div key={run.run_id} role="listitem">
                  <button
                    type="button"
                    className={`qa-history-row${selectedRunID === run.run_id ? " is-selected" : ""}`}
                    onClick={() => void handleSelectRun(run)}
                    aria-pressed={selectedRunID === run.run_id}
                  >
                    <span className="qa-history-question">{run.question || run.run_id}</span>
                    <span className="qa-history-meta">
                      <StatusBadge tone={qaRunStatusTone(run.status)}>{run.status}</StatusBadge>
                      <span>{qaRunProviderLabel(run)}</span>
                    </span>
                    <span className="qa-history-time">{formatTimestamp(run.finished_at || run.started_at)}</span>
                  </button>
                </div>
              ))}
            </div>
          )}
        </aside>

        <div className="qa-answer-panel" data-testid="qa-answer-panel">
          <div className="qa-composer">
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
            {status ? <p className={qaRun?.status === "failed" ? "status err" : qaRun?.status === "succeeded" ? "status ok" : "status warn"}>{status}</p> : null}
          </div>

          {qaRun ? (
            <div className="run-summary qa-run-summary" data-testid="qa-run-status">
              <div>
                <p>
                  Run <code>{qaRun.run_id}</code> status: <strong>{qaRun.status}</strong>
                </p>
                <p>Runtime provider: {qaRun.provider || qaRun.runtime_provider || "pending"}</p>
              </div>
              <a href={`/api/pipeline/runs/${encodeURIComponent(qaRun.run_id)}/logs`} target="_blank" rel="noreferrer">
                Open run logs
              </a>
              {qaRun.error ? <p className="status err">{qaRun.error}</p> : null}
              {(qaRun.warnings ?? []).length > 0 ? <p className="status warn">Warnings: {(qaRun.warnings ?? []).join(", ")}</p> : null}
            </div>
          ) : null}

          {qaRun ? (
            <div className="qa-answer" data-testid="qa-answer">
              <div className="panel-subheader">
                <div>
                  <h2>Answer</h2>
                  <p className="hint">{selectedLoading ? "Loading selected Q&A run." : qaRun.generated_at ? `Generated ${formatTimestamp(qaRun.generated_at)}` : "Awaiting generated answer."}</p>
                </div>
                <StatusBadge tone={confidence >= 75 ? "ok" : confidence > 0 ? "warn" : "info"}>Confidence: {confidence}%</StatusBadge>
              </div>
              {qaRun.answer ? <p>{qaRun.answer}</p> : <p className="empty-state">No answer returned yet. Check run status and logs for details.</p>}
              {unresolved.length > 0 ? <p className="status warn">Unresolved: {unresolved.join(", ")}</p> : <p className="hint">No unresolved assumptions returned.</p>}
              <div className="qa-related-partial">
                <h3>Related entities and edges</h3>
                <p className="hint">Partial state: the current QA API returns citations, not a structured related-entity graph. Use the citation trail for drilldown.</p>
              </div>
            </div>
          ) : (
            <div className="qa-answer qa-empty-answer">
              <h2>Answer</h2>
              <p className="empty-state">Select a historical run or ask a new question to review the answer, confidence and assumptions.</p>
            </div>
          )}
        </div>

        <aside className="qa-side-column">
          <section className="qa-readonly-safety-panel" data-testid="qa-readonly-safety-panel">
            <div className="panel-subheader">
              <div>
                <h2>Read-only runtime safety</h2>
                <p className="hint">Ask runs are audit-scoped and do not publish changes.</p>
              </div>
              <StatusBadge tone="ok">no canonical writes</StatusBadge>
            </div>
            <ul className="compact-list">
              <li>Source repositories stay read-only inputs.</li>
              <li>Canonical workspace outputs are not mutated by Q&A.</li>
              <li>Writes are limited to `reports/taskruns/&lt;run_id&gt;/qa/` audit artifacts.</li>
            </ul>
            {qaRun ? (
              <div className="actions qa-audit-actions">
                <button type="button" className="link-button" onClick={() => onOpenArtifact(`reports/taskruns/${qaRun.run_id}/qa/context-pack.json`)}>
                  context-pack.json
                </button>
                <button type="button" className="link-button" onClick={() => onOpenArtifact(`reports/taskruns/${qaRun.run_id}/qa/qa-answer.json`)}>
                  qa-answer.json
                </button>
                <button type="button" className="link-button" onClick={() => onOpenArtifact(`reports/taskruns/${qaRun.run_id}/qa/runtime-execution.json`)}>
                  runtime-execution.json
                </button>
              </div>
            ) : (
              <p className="hint">Audit links appear after selecting or starting a Q&A run.</p>
            )}
          </section>

          <section className="qa-citations-panel" data-testid="qa-citations-panel">
            <div className="panel-subheader">
              <div>
                <h2>Citations</h2>
                <p className="hint">Evidence used by the answer.</p>
              </div>
              <StatusBadge tone={citations.length > 0 ? "ok" : qaRun ? "warn" : "info"}>{citations.length} refs</StatusBadge>
            </div>
            {qaRun && citations.length === 0 ? (
              <p>No citations returned.</p>
            ) : citations.length > 0 ? (
              <ul className="citation-list">
                {citations.map((citation) => (
                  <li key={`${citation.path}-${citation.reason}`}>
                    <ArtifactPathButton path={citation.path} onOpenArtifact={onOpenArtifact} /> <span>{citation.reason}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="empty-state">No selected Q&A run yet.</p>
            )}
            <div className="qa-unresolved-box">
              <h3>Unresolved assumptions</h3>
              {unresolved.length > 0 ? <p className="status warn">Unresolved: {unresolved.join(", ")}</p> : <p className="hint">No unresolved assumptions returned.</p>}
            </div>
          </section>
        </aside>
      </div>
    </section>
  );
}

function qaRunStatusTone(status?: QARunResponse["status"]): "info" | "ok" | "warn" | "error" {
  if (status === "succeeded") {
    return "ok";
  }
  if (status === "failed") {
    return "error";
  }
  if (status === "queued" || status === "running") {
    return "warn";
  }
  return "info";
}

function qaRunProviderLabel(run: QARunResponse | null): string {
  return run?.provider || run?.runtime_provider || "agent-backed";
}

function mergeQARunHistory(run: QARunResponse, history: QARunResponse[], mode: "prepend" | "preserve" = "prepend"): QARunResponse[] {
  if (history.some((item) => item.run_id === run.run_id)) {
    return history.map((item) => (item.run_id === run.run_id ? { ...item, ...run } : item)).slice(0, 20);
  }
  if (mode === "preserve") {
    return [...history, run].slice(0, 20);
  }
  return [run, ...history].slice(0, 20);
}

type PublishStageProps = ComponentProps<typeof BaselineGitPanel> & {
  artifacts: Artifact[];
  selectedArtifact: string;
  selectedArtifactContent: string;
  openQuestions: string;
  externalGateItems?: PublishGateItem[];
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  onLoadGitDiff: (options: LoadGitDiffOptions) => void;
  onPreviewArtifact: (path: string) => void;
};

export function PublishStagePanel({
  busy,
  gitMessage,
  proposalBranch,
  gitStatus,
  artifacts,
  selectedArtifact,
  selectedArtifactContent,
  openQuestions,
  externalGateItems = [],
  gitDiff,
  gitDiffStatus,
  onLoadGitDiff,
  onGitMessageChange,
  onProposalBranchChange,
  onCommit,
  onCreateProposalBranch,
  onPreviewArtifact,
}: PublishStageProps) {
  const [publishView, setPublishView] = useState<"preview" | "diff" | "evidence" | "changelog">("preview");
  const [localSelectedPath, setLocalSelectedPath] = useState("");
  const [copyStatus, setCopyStatus] = useState("");
  const publishArtifacts = artifacts.filter((artifact) => artifact.path.trim().length > 0);
  const effectiveSelectedPath =
    (localSelectedPath && publishArtifacts.some((artifact) => artifact.path === localSelectedPath) ? localSelectedPath : "") ||
    (selectedArtifact && publishArtifacts.some((artifact) => artifact.path === selectedArtifact) ? selectedArtifact : "") ||
    publishArtifacts[0]?.path ||
    "";
  const folderSummaries = buildPublishFolderSummaries(publishArtifacts);
  const changelogArtifacts = publishArtifacts.filter((artifact) => artifact.kind === "changelog" || artifact.path.startsWith("reports/changelog/"));
  const selectedChangelogPath =
    (localSelectedPath && changelogArtifacts.some((artifact) => artifact.path === localSelectedPath) ? localSelectedPath : "") ||
    (selectedArtifact && changelogArtifacts.some((artifact) => artifact.path === selectedArtifact) ? selectedArtifact : "") ||
    changelogArtifacts[0]?.path ||
    "";
  const activePreviewPath = publishView === "changelog" && selectedChangelogPath ? selectedChangelogPath : effectiveSelectedPath;
  const selectedPublishArtifact = publishArtifacts.find((artifact) => artifact.path === activePreviewPath);
  const selectedPublishContent = selectedArtifact === activePreviewPath ? selectedArtifactContent : "";
  const selectedChangelogArtifact = changelogArtifacts.find((artifact) => artifact.path === selectedChangelogPath);
  const selectedChangelogContent = selectedArtifact === selectedChangelogPath ? selectedArtifactContent : "";
  const gateItems = [
    ...externalGateItems,
    ...buildPublishGateItems({
      artifactCount: publishArtifacts.length,
      folderCount: folderSummaries.length,
      gitMessage,
      proposalBranch,
      openQuestions,
    }),
  ];
  const blockingGateItems = gateItems.filter((item) => item.tone === "error");
  const openQuestionGateItems = gateItems.filter((item) => item.tone === "warn" && item.label.toLowerCase().includes("open question"));
  const warningGateItems = gateItems.filter((item) => item.tone === "warn" && !openQuestionGateItems.includes(item));
  const readyGateItems = gateItems.filter((item) => item.tone === "ok" || item.tone === "info");
  const gitMutationDisabled = busy || blockingGateItems.length > 0;
  const gitMutationBlockedTitle =
    blockingGateItems.length > 0 ? "Resolve publish gate blockers before changing Git publication state." : undefined;
  const realFolderSummaries =
    gitDiff?.folders.map((summary) => ({
      folder: summary.folder,
      count: summary.files,
      sample: `+${summary.additions} / -${summary.deletions}`,
    })) ?? [];
  const visibleFolderSummaries = realFolderSummaries.length > 0 ? realFolderSummaries : folderSummaries;

  useEffect(() => {
    if (activePreviewPath && selectedArtifact !== activePreviewPath) {
      onPreviewArtifact(activePreviewPath);
    }
  }, [activePreviewPath, selectedArtifact, onPreviewArtifact]);

  useEffect(() => {
    if (publishView === "diff") {
      onLoadGitDiff({});
    }
  }, [onLoadGitDiff, publishView]);

  function handleSelectPublishArtifact(path: string) {
    setLocalSelectedPath(path);
    setPublishView("preview");
    onPreviewArtifact(path);
  }

  function handleSelectChangelogArtifact(path: string) {
    setLocalSelectedPath(path);
    onPreviewArtifact(path);
  }

  async function handleCopyCommitMessage() {
    if (!navigator.clipboard) {
      setCopyStatus("Clipboard unavailable in this browser.");
      return;
    }
    try {
      await navigator.clipboard.writeText(gitMessage);
      setCopyStatus("Commit message copied.");
    } catch (error) {
      setCopyStatus(error instanceof Error ? error.message : "Commit message copy failed.");
    }
  }

  return (
    <div className="stage-stack" data-testid="publish-panel">
      <section className="panel stage-panel">
        <div className="stage-header">
          <div>
            <h1>Publish</h1>
            <p className="hint">Review workspace artifacts, check publication blockers and prepare Git commit/proposal branch handoff.</p>
          </div>
          <StatusBadge tone={blockingGateItems.length > 0 ? "error" : warningGateItems.length > 0 ? "warn" : publishArtifacts.length > 0 ? "ok" : "info"}>
            {blockingGateItems.length > 0 ? "blocked" : warningGateItems.length > 0 ? "review" : publishArtifacts.length > 0 ? "ready" : "partial"}
          </StatusBadge>
        </div>
      </section>

      <div className="publish-review-room">
        <section className="publish-diff-summary" data-testid="publish-diff-summary">
          <div className="panel-subheader">
            <div>
              <h2>Folder diff summary</h2>
              <p className="hint">Workspace Git diff from the local architecture workspace repository.</p>
            </div>
            <StatusBadge tone={gitDiff && !gitDiff.empty ? "ok" : publishArtifacts.length > 0 ? "info" : "warn"}>
              {gitDiff ? `${gitDiff.files.length} changed` : `${publishArtifacts.length} refs`}
            </StatusBadge>
          </div>
          {gitDiffStatus ? <p className={gitDiff?.empty ? "status ok" : "status warn"}>{gitDiffStatus}</p> : null}
          {visibleFolderSummaries.length === 0 ? (
            <p className="empty-state">No workspace Git changes are available for publication yet.</p>
          ) : (
            <div className="publish-folder-list">
              {visibleFolderSummaries.map((summary) => (
                <div key={summary.folder} className="publish-folder-row">
                  <div>
                    <strong>{summary.folder}</strong>
                    <span>{summary.count} file refs</span>
                  </div>
                  <span>{summary.sample}</span>
                </div>
              ))}
            </div>
          )}
          {gitDiff?.files.length ? (
            <div className="publish-artifact-list compact" role="list" aria-label="changed workspace files">
              {gitDiff.files.slice(0, 16).map((file) => (
                <div key={file.path} role="listitem">
                  <button
                    type="button"
                    className={`publish-artifact-row${gitDiff.selected_file?.path === file.path ? " is-selected" : ""}`}
                    onClick={() => {
                      setPublishView("diff");
                      onLoadGitDiff({ path: file.path });
                    }}
                    aria-pressed={gitDiff.selected_file?.path === file.path}
                  >
                    <span>{file.path}</span>
                    <code>
                      {file.status} · +{file.additions}/-{file.deletions}
                    </code>
                  </button>
                </div>
              ))}
            </div>
          ) : null}
          <div className="publish-artifact-list" role="list" aria-label="publish artifact preview list">
            {publishArtifacts.slice(0, 12).map((artifact) => (
              <div key={artifact.path} role="listitem">
                <button
                  type="button"
                  className={`publish-artifact-row${activePreviewPath === artifact.path ? " is-selected" : ""}`}
                  onClick={() => handleSelectPublishArtifact(artifact.path)}
                  aria-pressed={activePreviewPath === artifact.path}
                >
                  <span>{artifact.label || artifact.path}</span>
                  <code>{artifact.path}</code>
                </button>
              </div>
            ))}
          </div>
        </section>

        <section className="publish-preview-panel" data-testid="publish-preview-panel">
          <div className="publish-preview-tabs" role="tablist" aria-label="Publish preview tabs" data-testid="publish-preview-tabs">
            {(["preview", "diff", "evidence", "changelog"] as const).map((view) => (
              <button
                key={view}
                type="button"
                className={publishView === view ? "is-active" : ""}
                onClick={() => setPublishView(view)}
                aria-selected={publishView === view}
                role="tab"
              >
                {publishTabLabel(view)}
              </button>
            ))}
          </div>
          <div className="publish-tab-panel" data-testid="publish-tab-panel">
            {publishView === "preview" ? (
              <div className="publish-selected-preview" data-testid="publish-selected-preview">
                <h2>Selected artifact preview</h2>
                {selectedPublishArtifact ? (
                  <>
                    <p className="hint">{selectedPublishArtifact.path}</p>
                    {selectedPublishContent ? (
                      <pre data-testid="publish-selected-preview-content">{selectedPublishContent}</pre>
                    ) : (
                      <p className="empty-state" data-testid="publish-selected-preview-empty">Select an artifact to load its preview in this Publish room.</p>
                    )}
                  </>
                ) : (
                  <p className="empty-state" data-testid="publish-selected-preview-empty">No artifact selected for publication preview.</p>
                )}
              </div>
            ) : null}
            {publishView === "diff" ? (
              <>
                <h2>Git diff</h2>
                <GitDiffView gitDiff={gitDiff} status={gitDiffStatus} onSelectFile={(path) => onLoadGitDiff({ path })} />
              </>
            ) : null}
            {publishView === "evidence" ? (
              <>
                <h2>Evidence before commit</h2>
                {openQuestions.trim() ? <pre>{openQuestions}</pre> : <p className="hint">No open-question artifact content is currently loaded.</p>}
                <p className="hint">{visibleFolderSummaries.length} workspace folders have Git diff or artifact refs in this publication view.</p>
              </>
            ) : null}
            {publishView === "changelog" ? (
              <>
                <h2>Changelog</h2>
                {changelogArtifacts.length === 0 ? (
                  <>
                    <p className="empty-state">No changelog artifact is available from the selected run.</p>
                    <p className="hint">Keep publication review focused on the selected artifact preview, evidence tab and Publish gate until a changelog artifact is generated.</p>
                  </>
                ) : (
                  <>
                    <div className="publish-artifact-list compact" role="list" aria-label="publish changelog artifacts">
                      {changelogArtifacts.map((artifact) => (
                        <div key={artifact.path} role="listitem">
                          <button
                            type="button"
                            className={`publish-artifact-row${selectedChangelogPath === artifact.path ? " is-selected" : ""}`}
                            onClick={() => handleSelectChangelogArtifact(artifact.path)}
                            aria-pressed={selectedChangelogPath === artifact.path}
                          >
                            <span>{artifact.label || artifact.path}</span>
                            <code>{artifact.path}</code>
                          </button>
                        </div>
                      ))}
                    </div>
                    <div className="publish-changelog-preview">
                      <h3>Selected changelog preview</h3>
                      {selectedChangelogArtifact ? (
                        <>
                          <p className="hint">{selectedChangelogArtifact.path}</p>
                          {selectedChangelogContent ? <pre>{selectedChangelogContent}</pre> : <p className="empty-state">Loading changelog preview...</p>}
                        </>
                      ) : (
                        <p className="empty-state">No changelog selected.</p>
                      )}
                    </div>
                  </>
                )}
              </>
            ) : null}
          </div>
        </section>

        <aside className="publish-side-column">
          <section className="publish-gate-panel" data-testid="publish-gate-panel">
            <div className="panel-subheader">
              <div>
                <h2>Publish gate</h2>
                <p className="hint">Checks gate Git commit and proposal branch actions; Git commands stay explicit operator actions.</p>
              </div>
              <StatusBadge tone={blockingGateItems.length > 0 ? "error" : warningGateItems.length > 0 || openQuestionGateItems.length > 0 ? "warn" : "ok"}>
                {blockingGateItems.length > 0 ? "blocked" : warningGateItems.length > 0 || openQuestionGateItems.length > 0 ? "review" : "ready"}
              </StatusBadge>
            </div>
            <PublishGateSection testId="publish-hard-blockers" title="Hard blockers" emptyLabel="No hard blockers. Git actions are allowed." items={blockingGateItems} />
            <PublishGateSection testId="publish-review-warnings" title="Review warnings" emptyLabel="No review warnings." items={warningGateItems} />
            <PublishGateSection testId="publish-open-questions" title="Open questions" emptyLabel="No open questions loaded." items={openQuestionGateItems} />
            <PublishGateSection testId="publish-ready-checks" title="Ready checks" emptyLabel="No ready checks yet." items={readyGateItems} />
          </section>

          <section className="publish-commit-plan" data-testid="publish-commit-plan">
            <div className="panel-subheader">
              <div>
                <h2>Commit plan</h2>
                <p className="hint">Prepared commit/proposal branch actions use the existing Git API.</p>
              </div>
              <StatusBadge tone={gitStatus ? "ok" : "info"}>{gitStatus ? "updated" : "pending"}</StatusBadge>
            </div>
            <dl className="compact-defs">
              <div>
                <dt>Folders</dt>
                <dd>{visibleFolderSummaries.map((summary) => summary.folder).join(", ") || "No changed folders yet"}</dd>
              </div>
              <div>
                <dt>Proposal branch</dt>
                <dd>{proposalBranch || "proposal branch not prepared"}</dd>
              </div>
            </dl>
            <label htmlFor="publishGitMessage">Commit message</label>
            <input id="publishGitMessage" value={gitMessage} onChange={(event) => onGitMessageChange(event.target.value)} />
            <div className="actions publish-actions">
              <button
                type="button"
                className="publish-primary-action"
                onClick={onCommit}
                disabled={gitMutationDisabled}
                title={gitMutationBlockedTitle ?? "Commit reviewed workspace artifacts."}
                data-testid="publish-commit-selected-btn"
              >
                <span data-testid="git-commit-btn">Commit selected artifacts</span>
              </button>
              <button type="button" className="link-button" onClick={() => void handleCopyCommitMessage()}>
                Copy commit message
              </button>
            </div>
            <label htmlFor="publishProposalBranch">Proposal branch</label>
            <input id="publishProposalBranch" value={proposalBranch} onChange={(event) => onProposalBranchChange(event.target.value)} />
            <button type="button" onClick={onCreateProposalBranch} disabled={gitMutationDisabled} title={gitMutationBlockedTitle}>
              <span data-testid="git-proposal-branch-btn">Create/Switch proposal branch</span>
            </button>
            {copyStatus ? <p className="status ok">{copyStatus}</p> : null}
            {gitStatus ? <p className="status ok">{gitStatus}</p> : null}
          </section>
        </aside>
      </div>
    </div>
  );
}

type PublishGateItem = {
  label: string;
  detail: string;
  tone: "info" | "ok" | "warn" | "error";
};

function buildPublishFolderSummaries(artifacts: Artifact[]): Array<{ folder: string; count: number; sample: string }> {
  const grouped = new Map<string, Artifact[]>();
  for (const artifact of artifacts) {
    const folder = publishFolderLabel(artifact.path);
    grouped.set(folder, [...(grouped.get(folder) ?? []), artifact]);
  }
  return Array.from(grouped.entries())
    .map(([folder, items]) => ({
      folder,
      count: items.length,
      sample: items.slice(0, 2).map((item) => item.label || item.path).join(", "),
    }))
    .sort((left, right) => left.folder.localeCompare(right.folder));
}

function publishFolderLabel(path: string): string {
  const parts = path.split("/").filter(Boolean);
  if (parts[0] === "reports" && parts[1]) {
    return `reports/${parts[1]}`;
  }
  return parts[0] || "workspace";
}

function buildPublishGateItems({
  artifactCount,
  folderCount,
  gitMessage,
  proposalBranch,
  openQuestions,
}: {
  artifactCount: number;
  folderCount: number;
  gitMessage: string;
  proposalBranch: string;
  openQuestions: string;
}): PublishGateItem[] {
  const openQuestionLines = openQuestions
    .split(/\r?\n/)
    .map((line) => line.trim().replace(/^[-*]\s*/, ""))
    .filter((line) => line.length > 0 && !line.startsWith("#"));
  const openQuestionCount = openQuestionLines.length;
  const firstOpenQuestion = openQuestionLines[0];
  return [
    {
      label: artifactCount > 0 ? "Artifacts" : "Blocked",
      detail: artifactCount > 0 ? `${artifactCount} artifact refs across ${folderCount} folders.` : "Run analysis before publishing workspace artifacts.",
      tone: artifactCount > 0 ? "ok" : "error",
    },
    {
      label: "Open questions",
      detail: openQuestionCount > 0 ? `${openQuestionCount} open question lines should be reviewed before commit. First: ${firstOpenQuestion}` : "No loaded open questions.",
      tone: openQuestionCount > 0 ? "warn" : "ok",
    },
    {
      label: gitMessage.trim() ? "Message" : "Message",
      detail: gitMessage.trim() ? gitMessage : "Commit message is empty.",
      tone: gitMessage.trim() ? "ok" : "warn",
    },
    {
      label: proposalBranch.trim() ? "Branch" : "Branch",
      detail: proposalBranch.trim() ? proposalBranch : "Proposal branch is optional but recommended for review.",
      tone: proposalBranch.trim() ? "ok" : "info",
    },
  ];
}

function PublishGateSection({
  testId,
  title,
  emptyLabel,
  items,
}: {
  testId: string;
  title: string;
  emptyLabel: string;
  items: PublishGateItem[];
}) {
  return (
    <section className="publish-gate-section" data-testid={testId}>
      <div className="publish-gate-section-head">
        <h3>{title}</h3>
        <span>{items.length}</span>
      </div>
      {items.length === 0 ? (
        <p className="hint">{emptyLabel}</p>
      ) : (
        <ul className="publish-checklist">
          {items.map((item) => (
            <li key={`${title}-${item.label}`}>
              <StatusBadge tone={item.tone}>{item.label}</StatusBadge>
              <span>{item.detail}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function publishTabLabel(view: "preview" | "diff" | "evidence" | "changelog"): string {
  if (view === "preview") {
    return "Preview";
  }
  if (view === "diff") {
    return "Diff";
  }
  if (view === "evidence") {
    return "Evidence";
  }
  return "Changelog";
}

export function RuntimeSettingsStagePanel(props: ComponentProps<typeof RuntimeProfileSettingsPanel>) {
  return <RuntimeProfileSettingsPanel {...props} />;
}

function slugify(value: string): string {
  const normalized = value.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
  return normalized || "my-service";
}
