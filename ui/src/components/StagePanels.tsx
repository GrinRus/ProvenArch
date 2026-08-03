import { useCallback, useEffect, useRef, useState, type ComponentProps, type ReactNode, type RefObject } from "react";

import { BaselineEditorsPanel } from "./BaselineEditorsPanel";
import { BaselineGitPanel } from "./BaselineGitPanel";
import { EvidenceViewer } from "./EvidenceViewer";
import { ModalDialog } from "./ModalDialog";
import { RepoAnalysisScopeFields } from "./RepoAnalysisScopeFields";
import { RuntimeProfileSettingsPanel } from "./RuntimeProfileSettingsPanel";
import { RunStatusPanel } from "./RunStatusPanel";
import { RecoveryPanel, RunResultPanel, StructuredRunProgress } from "./RunOutcome";
import { TabNav, tabPanelProps } from "./TabNav";
import { ArtifactPathButton, StatusBadge } from "./ConsolePrimitives";
import { analysisScopeSummary } from "../lib/analysisScope";
import { isAbortError, useRequestGate } from "../hooks/useRequestGate";
import { providerCommandEnv, providerCommandHint, providerReadinessGuidance } from "../lib/providerGuidance";
import {
  createQAProposalDraft,
  getQARun,
  listQARuns,
  startQAQuestion,
  type QAProposalDraftResponse,
  type QARunResponse,
} from "../lib/qaApi";
import { providerDisplayLabel, runtimeDisplayLabel } from "../lib/runtimeDisplay";
import { formatTimestamp, isRunCanceled, isRunReconciledAfterRestart, isRunnerUnavailable, parseTimeOrMin, runOutcomeLabel, runOutcomeTone } from "../lib/runState";
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
  RunCoordination,
  RunLogEntry,
  RunReviewStep,
  RunReviewSummaryResponse,
  RunStatusResponse,
  ValidateResponse,
  WorkspaceHealthResponse,
} from "../lib/appContracts";
import type { LoadGitDiffOptions } from "../lib/gitDiffApi";

type ReviewArtifactFilter = "all" | "reports" | "diagrams" | "proposals" | "runtime";
type PublishArtifactFilter = "all" | "changed" | "reports" | "proposals" | "diagrams" | "taskruns";

const REVIEW_ARTIFACT_FILTERS: Array<{ id: ReviewArtifactFilter; label: string }> = [
  { id: "all", label: "All" },
  { id: "reports", label: "Reports" },
  { id: "diagrams", label: "Diagrams" },
  { id: "proposals", label: "Proposals" },
  { id: "runtime", label: "Runtime" },
];

const PUBLISH_ARTIFACT_FILTERS: Array<{ id: PublishArtifactFilter; label: string }> = [
  { id: "all", label: "All" },
  { id: "changed", label: "Changed" },
  { id: "reports", label: "Reports" },
  { id: "proposals", label: "Proposals" },
  { id: "diagrams", label: "Diagrams" },
  { id: "taskruns", label: "Taskruns" },
];

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

type SourceRecoveryIssue = {
  repoKey: string;
  diagnosticLabel: string;
  level: Diagnostic["level"] | "draft";
  message: string;
  suggestion: string;
  sourceType: string;
  sourceValue: string;
  refValue: string;
};

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

function buildSourceValidationRecovery(
  guidedRepos: GuidedRepo[],
  validateResult: ValidateResponse | null,
  validationDiagnosticsByRepo: Array<[string, Diagnostic[]]>,
): SourceRecoveryIssue | null {
  const serverIssue = firstSourceDiagnostic(guidedRepos, validateResult, validationDiagnosticsByRepo);
  if (serverIssue) {
    return serverIssue;
  }

  return firstDraftSourceIssue(guidedRepos);
}

function firstSourceDiagnostic(
  guidedRepos: GuidedRepo[],
  validateResult: ValidateResponse | null,
  validationDiagnosticsByRepo: Array<[string, Diagnostic[]]>,
): SourceRecoveryIssue | null {
  if (!validateResult) {
    return null;
  }

  const diagnosticEntry =
    validationDiagnosticsByRepo
      .flatMap(([repoKey, diagnostics]) => diagnostics.map((diagnostic) => ({ repoKey, diagnostic })))
      .find(({ diagnostic }) => diagnostic.level === "error") ??
    (!validateResult.ok
      ? validationDiagnosticsByRepo.flatMap(([repoKey, diagnostics]) => diagnostics.map((diagnostic) => ({ repoKey, diagnostic })))[0]
      : undefined);

  if (!diagnosticEntry) {
    return null;
  }

  const repo = guidedRepos.find((candidate) => candidate.name === diagnosticEntry.repoKey || candidate.name === diagnosticEntry.diagnostic.repo);
  return {
    repoKey: diagnosticEntry.repoKey === "__workspace__" ? "workspace.yaml" : diagnosticEntry.repoKey,
    diagnosticLabel: diagnosticEntry.diagnostic.code,
    level: diagnosticEntry.diagnostic.level,
    message: diagnosticEntry.diagnostic.message,
    suggestion: diagnosticEntry.diagnostic.suggestion || defaultSourceSuggestion(repo),
    sourceType: sourceTypeLabel(repo),
    sourceValue: sourceValueLabel(repo, validateResult.workspace),
    refValue: repo?.ref || "current/default",
  };
}

function firstDraftSourceIssue(guidedRepos: GuidedRepo[]): SourceRecoveryIssue | null {
  const nameCounts = new Map<string, number>();
  guidedRepos.forEach((repo) => {
    const name = repo.name.trim().toLowerCase();
    if (name) {
      nameCounts.set(name, (nameCounts.get(name) ?? 0) + 1);
    }
  });

  for (const [index, repo] of guidedRepos.entries()) {
    const repoKey = repo.name.trim() || `Repo ${index + 1}`;
    const duplicateName = repo.name.trim() && (nameCounts.get(repo.name.trim().toLowerCase()) ?? 0) > 1;
    const sourceMissing = repo.mode === "path" ? repo.path.trim() === "" : repo.git_url.trim() === "";
    const nameMissing = repo.name.trim() === "";

    if (!nameMissing && !duplicateName && !sourceMissing) {
      continue;
    }

    const diagnosticLabel = nameMissing ? "Repo name is missing" : duplicateName ? "Repo name is duplicated" : "Repository source is missing";
    const message = nameMissing
      ? "This repository needs a stable name before `workspace.yaml` can be saved and validated."
      : duplicateName
        ? "Repository names must be unique before Source can resolve each repo."
        : `${sourceTypeLabel(repo)} is empty, so Source cannot resolve this repository.`;
    const suggestion = nameMissing
      ? "Enter a short unique repository name, then save and validate sources."
      : duplicateName
        ? "Rename one of the duplicate repositories, then save and validate sources."
        : repo.mode === "path"
          ? "Enter the local checkout path, then save and validate sources."
          : "Enter the GitHub/GitLab URL and make sure local git authentication can reach it.";

    return {
      repoKey,
      diagnosticLabel,
      level: "draft",
      message,
      suggestion,
      sourceType: sourceTypeLabel(repo),
      sourceValue: sourceValueLabel(repo),
      refValue: repo.ref || "current/default",
    };
  }

  return null;
}

function sourceTypeLabel(repo?: GuidedRepo) {
  if (!repo) {
    return "Workspace manifest";
  }
  return repo.mode === "path" ? "Local folder" : "Git URL";
}

function sourceValueLabel(repo?: GuidedRepo, workspace?: string) {
  if (!repo) {
    return workspace || "workspace.yaml";
  }
  const sourceValue = repo.mode === "path" ? repo.path : repo.git_url;
  return sourceValue.trim() || `${sourceTypeLabel(repo)} missing`;
}

function defaultSourceSuggestion(repo?: GuidedRepo) {
  if (!repo) {
    return "Fix the workspace manifest entry, then save and validate sources again.";
  }
  return repo.mode === "path"
    ? "Check the local checkout path and filesystem access, then save and validate sources again."
    : "Check the repository URL and your local git authentication, then save and validate sources again.";
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

function workspaceHealthTone(report: WorkspaceHealthResponse | null, status: "idle" | "loading" | "loaded" | "error") {
  if (status === "error") {
    return "error" as const;
  }
  if (!report || status === "loading") {
    return "info" as const;
  }
  if (report.status === "fail") {
    return "error" as const;
  }
  if (report.status === "warn") {
    return "warn" as const;
  }
  return "ok" as const;
}

function workspaceHealthLabel(report: WorkspaceHealthResponse | null, status: "idle" | "loading" | "loaded" | "error") {
  if (status === "loading") {
    return "scanning";
  }
  if (status === "error") {
    return "scan failed";
  }
  if (!report) {
    return "not available";
  }
  return report.status;
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

export type CharterStageProps = ComponentProps<typeof BaselineEditorsPanel> & {
  wizardPanel: ReactNode;
  wizardProjectName: string;
  wizardScope: string;
  wizardNfr: string;
  wizardRules: string;
  gitStatus: string;
  proposalBranch: string;
};

export function CharterStagePanel({
  wizardPanel,
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
  const charterRecovery = buildCharterBaselineRecovery(baselineProps.baselineBundleWarnings, baselineProps.baselineEditorArtifacts, baselineProps.selectedEditorPath);

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

      {charterRecovery ? <CharterBaselineRecovery issue={charterRecovery} /> : null}

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
    </div>
  );
}

type CharterRecoveryIssue = {
  artifactPath: string;
  artifactLabel: string;
  category: string;
  promptUsage: string;
  severity: Diagnostic["level"];
  diagnosticCode: string;
  message: string;
  suggestion: string;
};

function CharterBaselineRecovery({ issue }: { issue: CharterRecoveryIssue }) {
  const badgeTone = issue.severity === "error" ? "error" : "warn";
  return (
    <section className="charter-recovery-panel" data-testid="charter-baseline-recovery">
      <div className="section-heading-row">
        <div>
          <h2>Charter baseline recovery</h2>
          <p className="hint">Resolve prompt or charter bundle diagnostics before using the baseline as live analysis context.</p>
        </div>
        <StatusBadge tone={badgeTone}>{issue.severity === "error" ? "baseline blocked" : "baseline warning"}</StatusBadge>
      </div>
      <div className="charter-recovery-grid">
        <div>
          <span className="metric-label">Affected artifact</span>
          <strong>{issue.artifactLabel}</strong>
        </div>
        <div>
          <span className="metric-label">Category</span>
          <strong>{issue.category}</strong>
        </div>
        <div>
          <span className="metric-label">Runtime use</span>
          <strong>{issue.promptUsage}</strong>
        </div>
        <div>
          <span className="metric-label">Diagnostic</span>
          <strong>{issue.diagnosticCode}</strong>
        </div>
      </div>
      <dl className="compact-defs charter-recovery-detail">
        <div>
          <dt>Message</dt>
          <dd>{issue.message}</dd>
        </div>
        <div>
          <dt>Suggested fix</dt>
          <dd>{issue.suggestion}</dd>
        </div>
        <div>
          <dt>Artifact path</dt>
          <dd>{issue.artifactPath}</dd>
        </div>
      </dl>
      <ul className="analysis-next-actions">
        <li>Select the affected artifact in Baseline: Editors, update the charter or prompt content, then use Save selected baseline artifact.</li>
        <li>Keep live-consumed prompt packs aligned before running Analysis; reference-only prompts can be fixed after the primary charter path is clear.</li>
        <li>Use Git status below Charter to review the workspace diff before publication.</li>
      </ul>
    </section>
  );
}

function buildCharterBaselineRecovery(
  baselineBundleWarnings: Diagnostic[],
  baselineEditorArtifacts: EditableArtifactOption[],
  selectedEditorPath: string,
): CharterRecoveryIssue | null {
  const diagnostic = baselineBundleWarnings.find((warning) => warning.level === "error") ?? baselineBundleWarnings[0];
  if (!diagnostic) {
    return null;
  }

  const artifact = findCharterDiagnosticArtifact(diagnostic, baselineEditorArtifacts, selectedEditorPath);
  const artifactPath = artifact?.path ?? diagnostic.path ?? selectedEditorPath ?? "baseline bundle";
  return {
    artifactPath,
    artifactLabel: artifact?.label ?? artifactPath,
    category: artifact?.category ?? (artifactPath.startsWith("charter/") ? "charter" : artifactPath.startsWith("skills/") ? "skills" : "bundle"),
    promptUsage: promptUsageLabel(artifact),
    severity: diagnostic.level,
    diagnosticCode: diagnostic.code,
    message: diagnostic.message,
    suggestion: diagnostic.suggestion || defaultCharterSuggestion(artifactPath),
  };
}

function findCharterDiagnosticArtifact(
  diagnostic: Diagnostic,
  baselineEditorArtifacts: EditableArtifactOption[],
  selectedEditorPath: string,
): EditableArtifactOption | undefined {
  const directPath = diagnostic.path?.trim();
  if (directPath) {
    const direct = baselineEditorArtifacts.find((artifact) => artifact.path === directPath);
    if (direct) {
      return direct;
    }
    const suffix = baselineEditorArtifacts.find((artifact) => directPath.endsWith(artifact.path) || artifact.path.endsWith(directPath));
    if (suffix) {
      return suffix;
    }
  }

  const messageMatch = baselineEditorArtifacts.find((artifact) => diagnostic.message.includes(artifact.path) || diagnostic.message.includes(artifact.label));
  if (messageMatch) {
    return messageMatch;
  }

  if (selectedEditorPath) {
    return baselineEditorArtifacts.find((artifact) => artifact.path === selectedEditorPath);
  }

  return undefined;
}

function promptUsageLabel(artifact?: EditableArtifactOption) {
  if (!artifact) {
    return "bundle diagnostic";
  }
  if (artifact.prompt_usage === "live-consumed") {
    return "live consumed";
  }
  if (artifact.prompt_usage === "reference-only") {
    return "reference only";
  }
  return artifact.path.startsWith("charter/") ? "charter context" : "editable baseline";
}

function defaultCharterSuggestion(artifactPath: string) {
  if (artifactPath.startsWith("skills/prompt-packs/")) {
    return "Open the live-consumed prompt pack, fix the diagnostic, then save the selected baseline artifact.";
  }
  if (artifactPath.startsWith("charter/")) {
    return "Open the charter artifact, fix the project context, then save the selected baseline artifact.";
  }
  return "Open the affected baseline artifact, fix the diagnostic, then save it before running Analysis.";
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
  detailMode?: boolean;
  busy: boolean;
  cancelBusy: boolean;
  runId: string | null;
  runStatus: RunStatusResponse | null;
  runList: RunListItem[];
  coordination: RunCoordination;
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
  onRunPipeline: (pipeline: "init" | "refresh", intent?: "start" | "queue") => void;
  onCancelSelectedRun: () => void;
  onCancelRun: (runId: string) => void;
  onSelectRun: (runId: string) => void;
  onOpenArtifact: (path: string) => void;
	onOpenArchitecture: () => void;
};

export function AnalysisStagePanel({
  detailMode = false,
  busy,
  cancelBusy,
  runId,
  runStatus,
  runList,
  coordination,
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
  onCancelRun,
  onSelectRun,
  onOpenArtifact,
	onOpenArchitecture,
}: AnalysisStageProps) {
  const blockerDetailsRef = useRef<HTMLElement>(null);
  const [selectedStepID, setSelectedStepID] = useState("");
  const [queueConfirmationOpen, setQueueConfirmationOpen] = useState(false);
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
  const liveDiagnostics = buildAnalysisLiveDiagnostics(runStatus, runLogs, shardRows, artifacts, selectedRunWarnings);
  const showActiveLiveDiagnostics = selectedRunIsActive && runStatus?.status !== "failed" && liveDiagnostics.hasTelemetry;

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
    <section className={`panel stage-panel runs-surface ${detailMode ? "is-detail" : "is-index"}`} data-testid="runs-control-panel">
      <div className="stage-header">
        <div>
          <h1>{detailMode ? "Execution detail" : "Start analysis"}</h1>
          <p className="hint">{detailMode ? "Inspect the selected run, its retained evidence and recovery path." : "Create a new architecture snapshot or refresh the current one."}</p>
        </div>
        <StatusBadge tone={selectedRunIsActive ? "warn" : runOutcomeTone(runStatus)}>{runOutcomeLabel(runStatus)}</StatusBadge>
      </div>

      <div className="actions">
        <button type="button" onClick={() => onRunPipeline("init")} disabled={busy || Boolean(coordination.active_run_id)} data-testid="run-init-btn">
          Run init
        </button>
        <button type="button" onClick={() => onRunPipeline("refresh")} disabled={busy || Boolean(coordination.active_run_id)} data-testid="run-refresh-btn">
          Run refresh
        </button>
        {coordination.active_run_id ? (
          <button type="button" onClick={() => setQueueConfirmationOpen(true)} disabled={busy} data-testid="run-queue-refresh-btn">
            Queue refresh after current run
          </button>
        ) : null}
        <button type="button" onClick={onCancelSelectedRun} disabled={busy || cancelBusy || !runId || !selectedRunIsActive} data-testid="run-cancel-btn">
          Cancel selected run
        </button>
      </div>
      {coordination.active_run_id ? <p className="hint" data-testid="run-active-start-reason">Ordinary start is unavailable while <code>{coordination.active_run_id}</code> is active.</p> : null}
      {coordination.pending ? (
        <section className="subsection" data-testid="pending-run-summary">
          <h2>Pending refresh</h2>
          <p><code>{coordination.pending.run_id}</code> · {coordination.pending.pipeline}. A newly queued refresh replaces this pending run.</p>
          <button type="button" onClick={() => onCancelRun(coordination.pending!.run_id)} disabled={busy || cancelBusy}>Cancel pending refresh</button>
        </section>
      ) : null}
      {runActionStatus ? <p className="status warn">{runActionStatus}</p> : null}
      {detailMode && runStatus?.pipeline === "refresh" ? <RefreshExecutionSummary runStatus={runStatus} /> : null}

      <ModalDialog
        open={queueConfirmationOpen}
        title={coordination.pending ? "Replace pending refresh" : "Queue refresh"}
        description={coordination.pending
          ? `Active run ${coordination.active_run_id}; pending ${coordination.pending.run_id} will be canceled as run_superseded.`
          : `Refresh will start after active run ${coordination.active_run_id}.`}
        confirmLabel={coordination.pending ? "Replace pending refresh" : "Queue refresh after current run"}
        busy={busy}
        onCancel={() => setQueueConfirmationOpen(false)}
        onConfirm={() => { setQueueConfirmationOpen(false); onRunPipeline("refresh", "queue"); }}
      />

	  <StructuredRunProgress runStatus={runStatus} review={runReviewSummary} onReviewDetails={handleReviewBlocker} />
      {detailMode ? (
        <div className="run-studio-body">
		  <RunResultPanel review={runReviewSummary} onExploreArchitecture={onOpenArchitecture} />
		  <RecoveryPanel runStatus={runStatus} review={runReviewSummary} busy={busy} onRetryStarted={onSelectRun} onReviewDetails={handleReviewBlocker} />
          <AnalysisRunTimeline steps={stepTimeline} />
          <AnalysisStepReview
            steps={reviewSteps}
            selectedStep={selectedReviewStep}
            runtimeMode={setupRuntime}
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
          <AnalysisFailedShardDetails rows={issueRows} detailsRef={blockerDetailsRef} />
          <details className="advanced-block runs-diagnostics-drawer" data-testid="runs-diagnostics-drawer" open={showActiveLiveDiagnostics || pendingPermissions.length > 0}>
            <summary>Technical diagnostics</summary>
            <p className="hint">Shards, raw runtime output, permissions and provider telemetry for this run.</p>
            {liveDiagnostics.hasTelemetry ? <AnalysisLiveDiagnosticsPanel diagnostics={liveDiagnostics} /> : null}
            <AnalysisShardTable rows={shardRows} />
            <RunStatusPanel runStatus={runStatus} warnings={selectedRunWarnings} />
            <PendingPermissionsTable pendingPermissions={pendingPermissions} />
          </details>
        </div>
      ) : (
        <RunHistoryTable runId={runId} runList={runList} runCounters={runCounters} onSelectRun={onSelectRun} />
      )}
    </section>
  );
}

function RefreshExecutionSummary({ runStatus }: { runStatus: RunStatusResponse }) {
  const summary = runStatus.refresh_summary;
  if (!summary) {
    return <section className="subsection" data-testid="refresh-summary-unavailable"><h2>Refresh details</h2><p className="hint">Refresh details unavailable for this legacy run.</p></section>;
  }
  const title = summary.mode === "no_op" ? "No changes in analysis scope" : summary.mode === "affected_only" ? "Affected scope refresh" : "Full refresh";
  return (
    <section className="subsection" data-testid="refresh-execution-summary">
      <div className="section-heading-row"><h2>{title}</h2><StatusBadge tone={summary.mode === "full" ? "info" : "ok"}>{summary.mode.replace("_", " ")}</StatusBadge></div>
      <dl className="compact-defs">
        <div><dt>Planner decision</dt><dd>{summary.decision}</dd></div>
        <div><dt>Baseline run</dt><dd>{summary.baseline_run_id || "none"}</dd></div>
        <div><dt>Reason</dt><dd>{summary.reason_codes.length > 0 ? summary.reason_codes.join(", ") : "none"}</dd></div>
        <div><dt>Artifacts</dt><dd>{summary.updated} updated · {summary.preserved} preserved · {summary.removed} removed · {summary.uncertain} uncertain</dd></div>
      </dl>
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
  artifactPair: AnalysisArtifactPairState;
  duration: string;
  lastMessage: string;
};

type AnalysisArtifactPairState = {
  label: string;
  detail: string;
  tone: "info" | "ok" | "warn" | "error";
  runtimeRefs: string[];
  markdownRefs: string[];
  manifestRefs: string[];
};

type AnalysisLiveMetric = {
  label: string;
  value: string;
  detail: string;
};

type AnalysisLiveTrace = {
  label: string;
  value: string;
};

type AnalysisLiveDiagnostics = {
  status: string;
  tone: "info" | "ok" | "warn" | "error";
  summary: string;
  metrics: AnalysisLiveMetric[];
  traces: AnalysisLiveTrace[];
  actions: string[];
  hasTelemetry: boolean;
};

const canonicalAnalysisSteps = [
  { suffix: "step0.constitution", label: "Charter" },
  { suffix: "step1.collect", label: "Collect" },
  { suffix: "step2.asis_docs", label: "As-is docs" },
  { suffix: "step3.findings", label: "Findings" },
  { suffix: "step4.proposals", label: "Proposals" },
];

export function AnalysisRunProgress({
  runId,
  runStatus,
  runtimeLabel,
  warningCount,
  errorCount,
  stepTimeline,
  blockerCount,
  actionLabel,
  onReviewBlocker,
}: {
  runId: string | null;
  runStatus: RunStatusResponse | null;
  runtimeLabel: string;
  warningCount: number;
  errorCount: number;
  stepTimeline: AnalysisStep[];
  blockerCount: number;
  actionLabel: string;
  onReviewBlocker: () => void;
}) {
  const completedSteps = stepTimeline.filter((step) => step.state === "done").length;
  const activeOrFailed = stepTimeline.find((step) => step.state === "active" || step.state === "failed");
  const hasBlocker = blockerCount > 0 || runStatus?.status === "failed" || Boolean(runStatus?.error_code);
  const terminal = runStatus?.status === "succeeded" || runStatus?.status === "failed" || runStatus?.status === "canceled";
  const stepLabel = runStatus?.status === "failed" ? "Stopped at" : terminal ? "Outcome" : "Current step";
  const stepValue = runStatus?.status === "succeeded"
    ? "Completed"
    : runStatus?.status === "canceled"
      ? "Canceled"
      : runStatus?.current_step ?? activeOrFailed?.id ?? "Not running";
  return (
    <section className="analysis-progress" data-testid="analysis-run-progress">
      <div className="section-heading-row">
        <h2>Run status</h2>
        <StatusBadge tone={runOutcomeTone(runStatus)}>{runOutcomeLabel(runStatus)}</StatusBadge>
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
          <span className="metric-label">{stepLabel}</span>
          <strong>{stepValue}</strong>
        </div>
        <div>
          <span className="metric-label">Progress</span>
          <strong>
            {completedSteps}/{stepTimeline.length} steps
          </strong>
        </div>
        <div>
          <span className="metric-label">Warnings/errors</span>
          <strong>
            {warningCount} / {errorCount}
          </strong>
        </div>
      </div>
      {hasBlocker ? (
        <button type="button" data-testid="analysis-review-blocker-btn" onClick={onReviewBlocker}>
          {actionLabel}
        </button>
      ) : null}
    </section>
  );
}

export function AnalysisFailureRecovery({
  busy,
  runStatus,
  runtimeLabel,
  warningCount,
  issueCount,
  artifactCount,
  pendingPermissionCount,
  liveDiagnostics,
  onRetry,
  onReviewBlocker,
}: {
  busy: boolean;
  runStatus: RunStatusResponse | null;
  runtimeLabel: string;
  warningCount: number;
  issueCount: number;
  artifactCount: number;
  pendingPermissionCount: number;
  liveDiagnostics: AnalysisLiveDiagnostics;
  onRetry: (pipeline: "init" | "refresh") => void;
  onReviewBlocker: () => void;
}) {
  if (runStatus?.status !== "failed") {
    return null;
  }

  const retryPipeline = runStatus.pipeline === "refresh" ? "refresh" : "init";
  const errorCode = runStatus.error_code || "unclassified";
  const blockedStep = runStatus.current_step || `${retryPipeline}.unknown`;
  const evidenceSummary = failureEvidenceSummary(artifactCount, issueCount);
  const canceled = isRunCanceled(errorCode);
  const reconciled = isRunReconciledAfterRestart(errorCode);
  const title = canceled ? "Canceled run" : reconciled ? "Recovered after restart" : "Recovery path";
  const badgeLabel = canceled ? "canceled" : reconciled ? "recovered" : "failed";
  const badgeTone = canceled || reconciled ? "warn" : "error";
  const stepLabel = canceled ? "Stopped step" : reconciled ? "Recovered step" : "Blocked step";
  const retainedRun = canceled || reconciled;
  const retryLabel = retainedRun ? `Run ${retryPipeline} again` : `Retry ${retryPipeline}`;
  const reviewLabel = retainedRun ? "Review retained evidence" : "Review blocker details";
  const retentionHint = canceled
    ? "Starting again creates a new run; the canceled run and its taskrun evidence stay in History."
    : reconciled
      ? "Starting again creates a new run; the reconciled run and its taskrun evidence stay in History."
      : "Retry starts a new run; the failed run remains available in History for audit and comparison.";

  return (
    <section className="analysis-recovery-panel" data-testid="analysis-failure-recovery">
      <div className="section-heading-row">
        <div>
          <h2>{title}</h2>
          <p className="hint">{failureRecoveryGuidance(errorCode, pendingPermissionCount)}</p>
        </div>
        <StatusBadge tone={badgeTone}>{badgeLabel}</StatusBadge>
      </div>

      <div className="analysis-recovery-grid">
        <div>
          <span className="metric-label">Classification</span>
          <strong>{errorCode}</strong>
        </div>
        <div>
          <span className="metric-label">{stepLabel}</span>
          <strong>{blockedStep}</strong>
        </div>
        <div>
          <span className="metric-label">Evidence kept</span>
          <strong>{evidenceSummary}</strong>
        </div>
        <div>
          <span className="metric-label">Warnings</span>
          <strong>{warningCount}</strong>
        </div>
        <div>
          <span className="metric-label">Runtime/provider</span>
          <strong>{runtimeLabel}</strong>
        </div>
      </div>

      {runStatus.error ? <p className="status err">{runStatus.error}</p> : null}
      <AnalysisLiveDiagnosticsPanel diagnostics={liveDiagnostics} />

      <div className="actions analysis-recovery-actions">
        <button type="button" data-testid="analysis-retry-run-btn" onClick={() => onRetry(retryPipeline)} disabled={busy}>
          {retryLabel}
        </button>
        <button type="button" className="secondary" data-testid="analysis-review-recovery-btn" onClick={onReviewBlocker}>
          {reviewLabel}
        </button>
      </div>
      <p className="hint">{retentionHint}</p>
    </section>
  );
}

function AnalysisLiveDiagnosticsPanel({ diagnostics }: { diagnostics: AnalysisLiveDiagnostics }) {
  return (
    <section className="analysis-live-diagnostics" data-testid="analysis-live-diagnostics">
      <div className="section-heading-row">
        <div>
          <h3>Live diagnostics</h3>
          <p className="hint">{diagnostics.summary}</p>
        </div>
        <StatusBadge tone={diagnostics.tone}>{diagnostics.status}</StatusBadge>
      </div>
      <div className="analysis-live-grid">
        {diagnostics.metrics.map((metric) => (
          <div key={metric.label}>
            <span className="metric-label">{metric.label}</span>
            <strong>{metric.value}</strong>
            <span>{metric.detail}</span>
          </div>
        ))}
      </div>
      <dl className="compact-defs analysis-live-traces">
        {diagnostics.traces.map((trace) => (
          <div key={trace.label}>
            <dt>{trace.label}</dt>
            <dd>{trace.value}</dd>
          </div>
        ))}
      </dl>
      <ul className="analysis-next-actions">
        {diagnostics.actions.map((action) => (
          <li key={action}>{action}</li>
        ))}
      </ul>
    </section>
  );
}

function failureRecoveryGuidance(errorCode: string, pendingPermissionCount: number): string {
  if (isRunCanceled(errorCode)) {
    return "The run stopped by request. Review retained taskrun evidence, then start a new run when ready.";
  }
  if (isRunReconciledAfterRestart(errorCode)) {
    return "ACP reconciled a stale run after restart. Inspect retained evidence, then start a new run if the previous work should continue.";
  }
  if (pendingPermissionCount > 0 || errorCode === "runtime_permission_required") {
    return "Resolve the pending permission request, then retry the same pipeline.";
  }
  if (isRunnerUnavailable(errorCode)) {
    return "Provider/tool availability blocked artifact creation; check Readiness provider setup, binary/auth/quota, then retry the same pipeline.";
  }
  if (errorCode.includes("runtime_timeout")) {
    return "The run exhausted its time budget; inspect the last progress signal before retry.";
  }
  if (errorCode.includes("runtime_contract")) {
    return "Generated artifacts did not pass validation; inspect the rejected step evidence before retry.";
  }
  if (errorCode.includes("infra") || errorCode.includes("incomplete")) {
    return "The cycle ended incomplete; review the last durable evidence before starting another run.";
  }
  return "Review the blocker details, then retry the same pipeline when the cause is clear.";
}

function failureEvidenceSummary(artifactCount: number, issueCount: number): string {
  if (artifactCount > 0) {
    return `${artifactCount} artifact refs kept`;
  }
  if (issueCount > 0) {
    return `${issueCount} diagnostic rows`;
  }
  return "status and logs only";
}

function buildAnalysisLiveDiagnostics(
  runStatus: RunStatusResponse | null,
  runLogs: RunLogEntry[],
  shardRows: AnalysisShardRow[],
  artifacts: Artifact[],
  warnings: string[],
): AnalysisLiveDiagnostics {
  const observedShards = new Set<string>();
  const failedShards = new Set<string>();
  const recoveryModes = new Set<string>();
  const rawRefs = new Set<string>();
  const validationExcerpts: string[] = [];
  const providerRefs = new Set<string>();
  let plannedShards: number | undefined;
  let succeededShards: number | undefined;
  let failedShardCount: number | undefined;
  let partialFailureCount: number | undefined;
  let repairScheduled = 0;
  let repairCompleted = 0;
  let repairExhausted = 0;
  let stallCount = 0;
  let preArtifactStalls = 0;
  let validArtifactControlledStops = 0;
  let zeroOutputPreArtifactStalls = 0;
  let artifactHandoffStalls = 0;
  const providerStream = summarizeProviderStream(runLogs);

  for (const entry of runLogs) {
    const fields = entry.fields;
    const message = entry.message || "";
    const normalizedMessage = message.toLowerCase();
    const shardID = fieldString(fields, "shard_id");
    if (shardID) {
      observedShards.add(shardID);
      if (entry.level === "error" || normalizedMessage.includes("failed") || normalizedMessage.includes("exhausted")) {
        failedShards.add(shardID);
      }
    }

    plannedShards = maxDefined(plannedShards, firstNumericField(fields, ["shards_total", "planned_shards", "planned", "total_shards", "total"]));
    succeededShards = maxDefined(succeededShards, firstNumericField(fields, ["succeeded_shards", "succeeded", "completed_shards", "completed"]));
    failedShardCount = maxDefined(failedShardCount, firstNumericField(fields, ["failed_shards", "failed"]));
    partialFailureCount = maxDefined(partialFailureCount, numericField(fields, "partial_failure_count"));

    const parsedCounters = parseShardCounters(message);
    if (parsedCounters) {
      plannedShards = maxDefined(plannedShards, parsedCounters.planned);
      succeededShards = maxDefined(succeededShards, parsedCounters.succeeded);
      failedShardCount = maxDefined(failedShardCount, parsedCounters.failed);
    }

    const provider = fieldString(fields, "provider") || fieldString(fields, "selected_provider");
    if (provider) {
      providerRefs.add(provider);
    }
    const recoveryMode = fieldString(fields, "recovery_mode");
    if (recoveryMode) {
      recoveryModes.add(recoveryMode);
    }
    const repairStage = fieldString(fields, "stage") || stageFromMessage(message);
    if (repairStage) {
      recoveryModes.add(repairStage);
    }

    if (normalizedMessage.includes("focused artifact repair scheduled") || normalizedMessage.includes("focused artifact repair retry scheduled")) {
      repairScheduled += 1;
    }
    if (normalizedMessage.includes("focused artifact repair completed")) {
      repairCompleted += 1;
    }
    if (
      normalizedMessage.includes("focused artifact repair exhausted") ||
      normalizedMessage.includes("collect manifest repair exhausted") ||
      normalizedMessage.includes("repair exhausted")
    ) {
      repairExhausted += 1;
    }

    const validationError = fieldString(fields, "validation_error") || fieldString(fields, "error");
    if (validationError) {
      validationExcerpts.push(validationError);
    }
    if (artifactHandoffStalled(normalizedMessage, validationError.toLowerCase(), repairStage)) {
      artifactHandoffStalls += 1;
    }

    if (boolField(fields, "zero_output_pre_artifact_stall")) {
      zeroOutputPreArtifactStalls += 1;
    }
    const stallPhase = fieldString(fields, "stall_phase");
    const isStall =
      fieldString(fields, "exit_reason") === "stall" ||
      validationError.includes("runtime_stalled") ||
      normalizedMessage.includes("runtime_stalled") ||
      normalizedMessage.includes("stalled");
    const artifactIsValid = boolField(fields, "artifact_valid") || fieldString(fields, "artifact_state") === "valid";
    if (isStall && artifactIsValid && validationError === "") {
      validArtifactControlledStops += 1;
    } else if (isStall) {
      stallCount += 1;
      if (stallPhase.includes("pre") || validationError.includes("before_artifacts")) {
        preArtifactStalls += 1;
      }
    }

    for (const ref of rawOutputRefsFromEntry(entry)) {
      rawRefs.add(ref);
    }
  }

  for (const row of shardRows) {
    if (row.scope && row.scope !== "workspace") {
      observedShards.add(row.scope);
    }
    if (row.status === "failed" && row.scope && row.scope !== "workspace") {
      failedShards.add(row.scope);
    }
  }

  const failedRowsFallback = failedShards.size === 0 ? shardRows.filter((row) => row.status === "failed").length : 0;
  const failedCount = Math.max(failedShardCount ?? 0, partialFailureCount ?? 0, failedShards.size, failedRowsFallback);
  const observedCount = Math.max(plannedShards ?? 0, observedShards.size, shardRows.length > 1 ? shardRows.length : 0);
  const succeededCount =
    succeededShards ?? (plannedShards !== undefined ? Math.max(plannedShards - failedCount, 0) : Math.max(observedCount - failedCount, 0));
  const rawRefList = Array.from(rawRefs).slice(0, 3);
  const recoveryModeList = Array.from(recoveryModes).slice(0, 3);
  const providerList = Array.from(providerRefs).slice(0, 3);
  const terminalExcerpt = firstNonEmpty([lastString(validationExcerpts), runStatus?.error ?? "", warnings[0] ?? ""]);
  const hasTelemetry = runLogs.length > 0 || artifacts.length > 0 || warnings.length > 0;
  const providerUnavailable = isRunnerUnavailable(runStatus?.error_code);
  const artifactHandoffBlocked = !providerUnavailable && (artifactHandoffStalls > 0 || (preArtifactStalls > 0 && repairExhausted > 0));
  const authoredShardArtifactCount = shardRows.reduce(
    (count, row) => count + row.artifactPair.markdownRefs.length + row.artifactPair.manifestRefs.length,
    0,
  );
  const providerStreamAwaitingArtifacts =
    runStatus?.status === "running" && providerStream.chunks > 0 && authoredShardArtifactCount === 0 && !artifactHandoffBlocked && !providerUnavailable;
  const status = providerUnavailable
    ? "provider check"
    : artifactHandoffBlocked
      ? "artifact handoff"
      : providerStreamAwaitingArtifacts
        ? "provider stream"
        : failedCount > 0 || repairExhausted > 0
          ? "action needed"
          : hasTelemetry
            ? "review"
            : "logs missing";
  const tone = providerUnavailable || failedCount > 0 || repairExhausted > 0 ? "error" : hasTelemetry ? "warn" : "info";
  const summary = providerUnavailable
    ? "Provider/tool availability blocked execution; fix Readiness provider setup before retrying the same pipeline."
    : artifactHandoffBlocked
      ? "The provider reached collect repair, but valid shard artifacts were not written before the pre-artifact stall."
      : providerStreamAwaitingArtifacts
        ? "Provider output is streaming, but no authored shard artifact pair is visible yet; wait for markdown plus shard-pack-manifest before treating collect as complete."
        : hasTelemetry
          ? "Shard, repair, stall and raw-output signals from the selected run are summarized here before retry."
          : "No live log telemetry is loaded for this failed run; use persisted status and artifacts first.";

  const metrics: AnalysisLiveMetric[] = [
    {
      label: providerStreamAwaitingArtifacts ? "Run signal" : status === "review" ? "Diagnostic signal" : "Failure mode",
      value: artifactHandoffBlocked
        ? "Artifact handoff stalled"
        : providerUnavailable
          ? "Provider unavailable"
          : providerStreamAwaitingArtifacts
            ? "Artifact pair pending"
            : failedCount > 0
              ? "Shard failure"
              : "Telemetry review",
      detail: artifactHandoffBlocked
        ? "collect repair did not produce both markdown and shard-pack-manifest before stalling"
        : providerUnavailable
          ? "readiness blocked before durable shard artifacts"
          : providerStreamAwaitingArtifacts
            ? "runtime stream is active; authored markdown and shard-pack-manifest are not visible yet"
            : failedCount > 0
              ? "inspect failed shard evidence before retry"
              : "no terminal artifact handoff blocker detected",
    },
    ...(providerStream.chunks > 0
      ? [
          {
            label: "Provider stream",
            value: `${providerStream.chunks} ${providerStream.chunks === 1 ? "chunk" : "chunks"}`,
            detail:
              providerStream.streamEvents > 0
                ? `${providerStream.streamEvents} JSON stream ${providerStream.streamEvents === 1 ? "event" : "events"} · ${providerStream.stdout} stdout / ${providerStream.stderr} stderr`
                : `${providerStream.stdout} stdout / ${providerStream.stderr} stderr · ${formatCompactCount(providerStream.characters)} chars`,
          },
        ]
      : []),
    {
      label: "Shard state",
      value: formatShardMetric(plannedShards, observedCount, succeededCount, failedCount),
      detail:
        failedShards.size > 0
          ? `failed: ${Array.from(failedShards).slice(0, 3).join(", ")}`
          : providerUnavailable
            ? "provider unavailable before shard ids were emitted"
            : "no failed shard ids in loaded logs",
    },
    {
      label: "Focused repair",
      value: `${repairScheduled} scheduled / ${repairCompleted} completed / ${repairExhausted} exhausted`,
      detail: recoveryModeList.length > 0 ? recoveryModeList.join(", ") : "no recovery mode logged",
    },
    {
      label: "Stall pressure",
      value: `${stallCount} actual / ${validArtifactControlledStops} valid-stop`,
      detail: `${preArtifactStalls} pre-artifact · ${zeroOutputPreArtifactStalls} zero-output`,
    },
    {
      label: "Raw refs",
      value: rawRefs.size > 0 ? `${rawRefs.size} refs` : "none loaded",
      detail: rawRefList.length > 0 ? rawRefList.join(", ") : `${artifacts.length} selected-run artifacts`,
    },
  ];
  const traces: AnalysisLiveTrace[] = [
    { label: "Provider", value: providerList.length > 0 ? providerList.join(", ") : "not exposed in loaded logs" },
    ...(providerStream.chunks > 0
      ? [{ label: "Stream signal", value: providerStream.signalTypes.length > 0 ? providerStream.signalTypes.join(", ") : "plain runtime output" }]
      : []),
    { label: "Recovery stage", value: recoveryModeList.length > 0 ? recoveryModeList.join(", ") : "not logged" },
    { label: "Terminal excerpt", value: terminalExcerpt || "No terminal validation excerpt loaded." },
  ];

  return {
    status,
    tone,
    summary,
    metrics,
    traces,
    actions: buildAnalysisLiveActions(
      failedCount,
      repairExhausted,
      stallCount,
      rawRefs.size,
      hasTelemetry,
      providerUnavailable,
      artifactHandoffBlocked,
      providerStreamAwaitingArtifacts,
    ),
    hasTelemetry,
  };
}

function buildAnalysisLiveActions(
  failedCount: number,
  repairExhausted: number,
  stallCount: number,
  rawRefCount: number,
  hasTelemetry: boolean,
  providerUnavailable: boolean,
  artifactHandoffBlocked: boolean,
  providerStreamAwaitingArtifacts: boolean,
): string[] {
  if (providerUnavailable) {
    const actions = ["Check Readiness provider setup, binary/auth/quota before retrying the same pipeline."];
    if (hasTelemetry) {
      actions.push("Use terminal excerpt/provider rows only to confirm the outage, not as a shard-quality failure.");
    }
    return actions;
  }
  if (!hasTelemetry) {
    return ["Load run logs or open persisted artifacts before retrying the same pipeline."];
  }
  if (providerStreamAwaitingArtifacts) {
    return [
      "Watch for authored markdown plus shard-pack-manifest before treating provider output as collect progress.",
      "If collect stalls or repair starts, use raw-output metadata instead of reading the full provider stream.",
    ];
  }
  const actions = artifactHandoffBlocked
    ? ["Open the failed shard row and raw-output ref to confirm whether markdown and shard-pack-manifest were written.", "Retry after the provider artifact write path is fixed or a collect-capable provider is selected."]
    : ["Inspect failed shard rows and terminal excerpts before starting a retry."];
  if (failedCount > 0) {
    actions.push("Retry only after the failed shard/provider cause is understood.");
  }
  if (repairExhausted > 0 || stallCount > 0) {
    actions.push("Confirm the provider can write valid artifacts without relying on focused repair.");
  }
  if (rawRefCount > 0) {
    actions.push("Use raw-output metadata to compare stdout/stderr against the failed shard evidence.");
  }
  return actions;
}

type ProviderStreamSummary = {
  chunks: number;
  streamEvents: number;
  stdout: number;
  stderr: number;
  characters: number;
  signalTypes: string[];
};

function summarizeProviderStream(runLogs: RunLogEntry[]): ProviderStreamSummary {
  const signalTypes = new Set<string>();
  let chunks = 0;
  let streamEvents = 0;
  let stdout = 0;
  let stderr = 0;
  let characters = 0;

  for (const entry of runLogs) {
    if (entry.kind !== "runtime_output") {
      continue;
    }
    chunks += 1;
    characters += entry.message.length;
    if (entry.stream === "stderr") {
      stderr += 1;
    } else {
      stdout += 1;
    }
    const parsed = parseRuntimeOutputJSON(entry.message);
    if (!parsed) {
      continue;
    }
    const topType = objectString(parsed, "type");
    const event = objectField(parsed, "event");
    const eventType = objectString(event, "type");
    const delta = objectField(event, "delta") ?? objectField(parsed, "delta");
    const deltaType = objectString(delta, "type");
    if (topType === "stream_event" || eventType || deltaType) {
      streamEvents += 1;
    }
    const signal = firstNonEmpty([deltaType, eventType, topType]);
    if (signal) {
      signalTypes.add(signal);
    }
  }

  return {
    chunks,
    streamEvents,
    stdout,
    stderr,
    characters,
    signalTypes: Array.from(signalTypes).slice(0, 3),
  };
}

function parseRuntimeOutputJSON(message: string): Record<string, unknown> | null {
  const trimmed = message.trim();
  if (!trimmed.startsWith("{") || !trimmed.endsWith("}")) {
    return null;
  }
  try {
    const parsed: unknown = JSON.parse(trimmed);
    return isRecord(parsed) ? parsed : null;
  } catch {
    return null;
  }
}

function objectField(record: Record<string, unknown> | null, key: string): Record<string, unknown> | null {
  const value = record?.[key];
  return isRecord(value) ? value : null;
}

function objectString(record: Record<string, unknown> | null, key: string): string {
  const value = record?.[key];
  return typeof value === "string" ? value.trim() : "";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function artifactHandoffStalled(normalizedMessage: string, normalizedValidationError: string, repairStage: string): boolean {
  const normalizedStage = repairStage.toLowerCase();
  const hasStallWord = /\bstall(?:ed|s|ing)?\b/.test(normalizedMessage);
  return (
    normalizedValidationError.includes("runtime_stalled_before_artifacts") ||
    normalizedMessage.includes("stalled before valid artifacts") ||
    normalizedMessage.includes("before valid artifacts were available") ||
    (normalizedStage.includes("collect_pair_repair") && hasStallWord)
  );
}

function stageFromMessage(message: string): string {
  const match = message.match(/\bstage=([^\s)]+)/i);
  return match?.[1] ?? "";
}

function formatShardMetric(plannedShards: number | undefined, observedCount: number, succeededCount: number, failedCount: number): string {
  if (plannedShards !== undefined && plannedShards > 0) {
    return `${succeededCount}/${plannedShards} ok · ${failedCount} failed`;
  }
  if (observedCount > 0) {
    return `${failedCount} failed / ${observedCount} observed`;
  }
  return "no shard counters";
}

function parseShardCounters(message: string): { planned: number; succeeded: number; failed: number } | null {
  const match = message.match(/shards_total=(\d+)\s+succeeded=(\d+)\s+failed=(\d+)/i) ?? message.match(/planned=(\d+)\s+succeeded=(\d+)\s+failed=(\d+)/i);
  if (!match) {
    return null;
  }
  return {
    planned: Number(match[1]),
    succeeded: Number(match[2]),
    failed: Number(match[3]),
  };
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
  runtimeMode,
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
  runtimeMode: string;
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
                <span>{providerDisplayLabel(runtimeMode, step.provider)}</span>
                <span>
                  {step.artifact_count} artifacts · {step.warnings_count}/{step.errors_count} warn/error
                </span>
              </button>
            ))}
          </div>

          <TabNav
            ariaLabel="Step review tabs"
            className="step-review-tabs"
            idBase="analysis-step-tabs"
            testId="analysis-step-tabs"
            value={view}
            onChange={(tab) => {
              onViewChange(tab);
              if (tab === "diff" && selectedStep?.step_id) {
                onLoadGitDiff({ stepId: selectedStep.step_id });
              }
            }}
            options={(["artifacts", "logs", "evidence", "diff"] as const).map((tab) => ({ id: tab, label: capitalize(tab), testId: `analysis-step-tab-${tab}` }))}
          />

          <div className="step-review-body" {...tabPanelProps("analysis-step-tabs", view)}>
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
                <th>Artifact pair</th>
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
                  <td data-label="Artifact pair">
                    <span className={`artifact-pair-state ${row.artifactPair.tone}`}>
                      <strong>{row.artifactPair.label}</strong>
                      <span>{row.artifactPair.detail}</span>
                    </span>
                  </td>
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
  const shardScopedRows = rows.filter((row) => row.scope !== "workspace");
  const displayRows = shardScopedRows.length > 0 ? shardScopedRows : rows;
  return (
    <section className="subsection" data-testid="analysis-failed-shard-details" ref={detailsRef} tabIndex={-1}>
      <h2>Blocker drilldown</h2>
      {displayRows.length === 0 ? (
        <p className="hint">No failed shard or warning log entries for the selected run.</p>
      ) : (
        <ul className="compact-list">
          {displayRows.slice(0, 4).map((row) => (
            <li key={`${row.key}-detail`}>
              <span>
                {row.status.toUpperCase()} · {row.stepId} · {row.scope}
              </span>
              <span className={`artifact-pair-state ${row.artifactPair.tone}`}>
                <strong>{row.artifactPair.label}</strong>
                <span>{row.artifactPair.detail}</span>
              </span>
              <dl className="compact-defs artifact-pair-refs">
                <div>
                  <dt>Runtime record</dt>
                  <dd>{formatArtifactPairRefs(row.artifactPair.runtimeRefs)}</dd>
                </div>
                <div>
                  <dt>Authored markdown</dt>
                  <dd>{formatArtifactPairRefs(row.artifactPair.markdownRefs)}</dd>
                </div>
                <div>
                  <dt>Manifest</dt>
                  <dd>{formatArtifactPairRefs(row.artifactPair.manifestRefs)}</dd>
                </div>
              </dl>
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
    const shardScope = fieldString(entry.fields, "shard_id") || shardScopeFromPath(entry.taskrun_path ?? "");
    const key = shardScope ? `${entry.step_id || "run"}/shard/${shardScope}` : entry.taskrun_path || `${entry.step_id || "run"}/${entry.domain_id || "workspace"}`;
    grouped.set(key, [...(grouped.get(key) ?? []), entry]);
  }
  const provider = setupRuntime === "fake" ? "fake" : setupRuntimeProvider;
  const rows: AnalysisShardRow[] = [];
  for (const [key, entries] of grouped.entries()) {
    const last = entries[entries.length - 1];
    const stepId = last?.step_id || entries.find((entry) => entry.step_id)?.step_id || runStatus?.current_step || "-";
    const hasError = entries.some((entry) => entry.level === "error");
    const hasWarning = entries.some((entry) => entry.level === "warning");
    const shardScope = fieldString(last?.fields, "shard_id") || shardScopeFromPath(last?.taskrun_path ?? "");
    const scope = shardScope || last?.domain_id || fieldString(last?.fields, "domain_id") || fieldString(last?.fields, "repo") || "workspace";
    const artifactRef = [...entries].reverse().find((entry) => entry.taskrun_path)?.taskrun_path || (artifacts.length > 0 ? `${artifacts.length} selected-run artifacts` : "logs only");
    rows.push({
      key,
      stepId,
      scope,
      provider: setupRuntime === "fake" ? "fake" : fieldString(last?.fields, "provider") || provider,
      status: hasError ? "failed" : hasWarning ? "warning" : runStatus?.status === "succeeded" ? "succeeded" : runStatus?.current_step && stepMatches(runStatus.current_step, stepId) ? "active" : "observed",
      artifactRef,
      artifactPair: buildAnalysisArtifactPairState(scope, artifactRef, artifacts),
      duration: durationFromEntries(entries),
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
      artifactPair: buildAnalysisArtifactPairState("workspace", artifacts.length > 0 ? `${artifacts.length} selected-run artifacts` : "status only", artifacts),
      duration: "Duration unavailable",
      lastMessage: runStatus.error || runStatus.error_code || "No shard logs loaded yet.",
    });
  }
  return rows;
}

function buildAnalysisArtifactPairState(scope: string, artifactRef: string, artifacts: Artifact[]): AnalysisArtifactPairState {
  const shardScoped = scope !== "workspace" && scope.trim().length > 0;
  const normalizedScope = scope.trim();
  const selectedPaths = artifacts.map((artifact) => artifact.path).filter((path) => pathMatchesShardScope(path, normalizedScope));
  const refPaths = pathMatchesShardScope(artifactRef, normalizedScope) ? [artifactRef] : [];
  const paths = Array.from(new Set([...selectedPaths, ...refPaths]));
  const runtimeRefs = paths.filter((path) => path.endsWith("/runtime-execution.json"));
  const markdownRefs = paths.filter((path) => /\.(md|markdown)$/i.test(path) && !path.endsWith("/shard-pack-manifest.md"));
  const manifestRefs = paths.filter((path) => path.endsWith("/shard-pack-manifest.json"));

  if (!shardScoped) {
    return {
      label: "Run-level evidence",
      detail: artifacts.length > 0 ? "selected-run artifacts are available, but this row is not shard-scoped" : "artifact list not loaded for this run",
      tone: artifacts.length > 0 ? "info" : "warn",
      runtimeRefs,
      markdownRefs,
      manifestRefs,
    };
  }
  if (markdownRefs.length > 0 && manifestRefs.length > 0) {
    return {
      label: "Artifact pair present",
      detail: "authored markdown and shard-pack-manifest are both visible",
      tone: "ok",
      runtimeRefs,
      markdownRefs,
      manifestRefs,
    };
  }
  if (markdownRefs.length > 0) {
    return {
      label: "Markdown only",
      detail: "authored markdown is visible, but shard-pack-manifest is missing",
      tone: "warn",
      runtimeRefs,
      markdownRefs,
      manifestRefs,
    };
  }
  if (manifestRefs.length > 0) {
    return {
      label: "Manifest only",
      detail: "shard-pack-manifest is visible, but authored markdown is missing",
      tone: "warn",
      runtimeRefs,
      markdownRefs,
      manifestRefs,
    };
  }
  if (runtimeRefs.length > 0) {
    return {
      label: "Runtime only",
      detail: "runtime-execution exists; authored markdown and shard-pack-manifest are missing",
      tone: "error",
      runtimeRefs,
      markdownRefs,
      manifestRefs,
    };
  }
  return {
    label: artifacts.length > 0 ? "No shard artifacts" : "Artifact list not loaded",
    detail: artifacts.length > 0 ? "selected-run artifacts do not include this shard" : "load selected-run artifacts before retry triage",
    tone: artifacts.length > 0 ? "warn" : "info",
    runtimeRefs,
    markdownRefs,
    manifestRefs,
  };
}

function pathMatchesShardScope(path: string, scope: string): boolean {
  if (!scope) {
    return false;
  }
  const shardSegment = `staging/shards/${scope}/`;
  return path.includes(`/${shardSegment}`) || path.startsWith(shardSegment);
}

function shardScopeFromPath(path: string): string {
  const normalized = path.replace(/\\/g, "/");
  const match = normalized.match(/(?:^|\/)staging\/shards\/([^/]+)\//);
  return match?.[1] ?? "";
}

function formatArtifactPairRefs(paths: string[]): string {
  if (paths.length === 0) {
    return "missing";
  }
  const visible = paths.slice(0, 2).join(", ");
  return paths.length > 2 ? `${visible} +${paths.length - 2} more` : visible;
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

function durationFromEntries(entries: RunLogEntry[]): string {
  for (const entry of [...entries].reverse()) {
    const duration = durationFromLogFields(entry.fields);
    if (duration !== "Duration unavailable") {
      return duration;
    }
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

function firstNumericField(fields: Record<string, unknown> | undefined, keys: string[]): number | undefined {
  for (const key of keys) {
    const value = numericField(fields, key);
    if (value !== undefined) {
      return value;
    }
  }
  return undefined;
}

function maxDefined(current: number | undefined, next: number | undefined): number | undefined {
  if (next === undefined) {
    return current;
  }
  return current === undefined ? next : Math.max(current, next);
}

function boolField(fields: Record<string, unknown> | undefined, key: string): boolean {
  const value = fields?.[key];
  if (typeof value === "boolean") {
    return value;
  }
  return typeof value === "string" && value.trim().toLowerCase() === "true";
}

function rawOutputRefsFromEntry(entry: RunLogEntry): string[] {
  const refs = new Set<string>();
  const rawOutput = fieldString(entry.fields, "raw_output");
  if (rawOutput) {
    refs.add(rawOutput);
  }
  const rawOutputMetadata = fieldString(entry.fields, "raw_output_metadata");
  if (rawOutputMetadata) {
    refs.add(rawOutputMetadata);
  }
  for (const match of entry.message.matchAll(/raw_output=([^\s)]+)/gi)) {
    refs.add(match[1]);
  }
  return Array.from(refs);
}

function firstNonEmpty(values: string[]): string {
  return values.map((value) => value.trim()).find((value) => value.length > 0) ?? "";
}

function lastString(values: string[]): string {
  return values.length > 0 ? values[values.length - 1] : "";
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

function formatCompactCount(value: number): string {
  if (value < 1000) {
    return `${value}`;
  }
  if (value < 1_000_000) {
    return `${Math.round(value / 100) / 10}k`;
  }
  return `${Math.round(value / 100_000) / 10}m`;
}

function PendingPermissionsTable({ pendingPermissions }: { pendingPermissions: RuntimePermissionRequest[] }) {
  const primaryRequest = pendingPermissions[0] ?? null;
  const requestCountLabel = `${pendingPermissions.length} pending ${pendingPermissions.length === 1 ? "request" : "requests"}`;
  const blockedStep = compactUniqueValues(pendingPermissions.map((request) => request.step_id)).join(", ") || "-";
  const actions = compactUniqueValues(pendingPermissions.map((request) => request.action)).join(", ") || "-";
  const decisions = compactUniqueValues(pendingPermissions.map((request) => request.decision?.decision)).join(", ") || "-";
  const rules = compactUniqueValues(pendingPermissions.map((request) => request.decision?.rule_id)).join(", ") || "-";

  return (
    <section className="subsection" data-testid="runs-pending-permissions-panel">
      <h2>Pending permissions</h2>
      {pendingPermissions.length === 0 ? (
        <p>No pending runtime permission requests.</p>
      ) : (
        <>
          <section className="permission-recovery-panel" data-testid="runtime-permission-recovery">
            <div className="section-heading-row">
              <div>
                <h3>Permission triage</h3>
                <p className="hint">
                  Managed runtime paused before approving an operation outside the step envelope. Review the target and rule before retrying.
                </p>
              </div>
              <StatusBadge tone="warn">{requestCountLabel}</StatusBadge>
            </div>
            <div className="permission-recovery-grid">
              <div>
                <span className="metric-label">Blocked step</span>
                <strong>{blockedStep}</strong>
              </div>
              <div>
                <span className="metric-label">Operation</span>
                <strong>{actions}</strong>
              </div>
              <div>
                <span className="metric-label">Decision</span>
                <strong>{decisions}</strong>
              </div>
              <div>
                <span className="metric-label">Policy rule</span>
                <strong>{rules}</strong>
              </div>
            </div>
            {primaryRequest ? (
              <dl className="compact-defs permission-request-summary">
                <div>
                  <dt>Primary target</dt>
                  <dd>{primaryRequest.path_or_command || "-"}</dd>
                </div>
                <div>
                  <dt>Reason</dt>
                  <dd>{primaryRequest.reason || primaryRequest.decision?.message || "No reason recorded."}</dd>
                </div>
              </dl>
            ) : null}
            <ul className="analysis-next-actions">
              <li>Inspect the path or command and reason before rerun.</li>
              <li>Use Readiness - Advanced runtime settings - Runtime Permissions to choose the intended mode/channel.</li>
              <li>If the request is unexpected, adjust source scope or runtime profile before retrying the failed pipeline.</li>
            </ul>
          </section>
          <div className="permission-request-cards" data-testid="runs-pending-permissions-cards">
            {pendingPermissions.map((request) => (
              <article className="permission-request-card" key={request.request_id || `${request.step_id}-${request.action}-${request.path_or_command}-card`}>
                <div className="section-heading-row">
                  <div>
                    <span className="metric-label">Permission request</span>
                    <strong className="permission-request-card-title">{request.action || "runtime permission"}</strong>
                  </div>
                  <StatusBadge tone="warn">{request.decision?.decision || "pending"}</StatusBadge>
                </div>
                <dl className="compact-defs permission-request-summary">
                  <div>
                    <dt>Request ID</dt>
                    <dd>{request.request_id || "-"}</dd>
                  </div>
                  <div>
                    <dt>Provider</dt>
                    <dd>{request.provider || "-"}</dd>
                  </div>
                  <div>
                    <dt>Step</dt>
                    <dd>{request.step_id || "-"}</dd>
                  </div>
                  <div>
                    <dt>Rule</dt>
                    <dd>{request.decision?.rule_id || "-"}</dd>
                  </div>
                  <div>
                    <dt>Target</dt>
                    <dd>{request.path_or_command || "-"}</dd>
                  </div>
                  <div>
                    <dt>Reason</dt>
                    <dd>{request.reason || request.decision?.message || "-"}</dd>
                  </div>
                </dl>
              </article>
            ))}
          </div>
          <div className="run-table-wrap permission-request-table-wrap">
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
        </>
      )}
    </section>
  );
}

function compactUniqueValues(values: Array<string | null | undefined>): string[] {
  return Array.from(new Set(values.map((value) => String(value ?? "").trim()).filter(Boolean)));
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
  const terminalOutcomeCounts = runList.reduce(
    (counts, run) => {
      const label = runOutcomeLabel(run);
      if (label === "failed") {
        counts.failed += 1;
      } else if (label === "canceled") {
        counts.canceled += 1;
      } else if (label === "recovered") {
        counts.recovered += 1;
      }
      return counts;
    },
    { failed: 0, canceled: 0, recovered: 0 },
  );

  return (
    <section className="subsection" data-testid="runs-history-panel">
      <h2>History</h2>
      <p className="hint">
        Running: {runCounters.running} | Succeeded: {runCounters.succeeded} | Failed: {terminalOutcomeCounts.failed} | Canceled: {terminalOutcomeCounts.canceled} |
        Recovered: {terminalOutcomeCounts.recovered}
      </p>
      {runList.length === 0 ? (
        <p>No runs yet.</p>
      ) : (
        <div className="run-table-wrap">
          <table className="run-table responsive-card-table" data-testid="runs-history-table">
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
                  <td data-label="Run ID">
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
                  <td data-label="Status">
                    <StatusBadge tone={runOutcomeTone(run)}>{runOutcomeLabel(run)}</StatusBadge>
                  </td>
                  <td data-label="Pipeline">{run.pipeline}</td>
                  <td data-label="Started">{formatTimestamp(run.started_at)}</td>
                  <td data-label="Finished">{formatTimestamp(run.finished_at)}</td>
                  <td data-label="Error code">{run.error_code || "-"}</td>
                  <td data-label="Warnings">{run.warnings?.length ?? 0}</td>
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
  routeView?: "overview" | "evidence" | "findings" | "diff";
  runId: string | null;
  runStatus: RunStatusResponse | null;
  runList: RunListItem[];
  coverageSummary: string;
  openQuestions: string;
  nonDiagramArtifacts: Artifact[];
  diagramArtifacts: Artifact[];
  selectedArtifact: string;
  selectedArtifactContent: string;
  evidenceStatus?: "idle" | "loading" | "available" | "partial" | "not_produced" | "unavailable" | "error";
  evidenceIssues?: Array<{ code: string; message: string; path?: string }>;
  runLogs: RunLogEntry[];
  reviewSummary: RunReviewSummaryResponse | null;
  demo: boolean;
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  onLoadGitDiff: (options: LoadGitDiffOptions) => void;
  onSelectRun: (id: string) => void;
  onOpenArtifact: (path: string) => void;
};

export function ReviewStagePanel({
  routeView = "overview",
  runId,
  runStatus,
  runList,
  coverageSummary,
  openQuestions,
  nonDiagramArtifacts,
  diagramArtifacts,
  selectedArtifact,
  selectedArtifactContent,
  evidenceStatus = "available",
  evidenceIssues = [],
  runLogs,
  reviewSummary,
  demo,
  gitDiff,
  gitDiffStatus,
  onLoadGitDiff,
  onSelectRun,
  onOpenArtifact,
}: ReviewStageProps) {
  const openReviewArtifactRef = useRef(onOpenArtifact);
  openReviewArtifactRef.current = onOpenArtifact;
  const overviewArtifact = nonDiagramArtifacts.find((artifact) => artifact.path === "reports/as-is/overview.md");
  const coverageArtifact = nonDiagramArtifacts.find((artifact) => artifact.path === "reports/coverage/summary.md");
  const findingsArtifact = nonDiagramArtifacts.find((artifact) => artifact.path.startsWith("reports/findings/"));
  const allArtifacts = [...nonDiagramArtifacts, ...diagramArtifacts];
  const preferredReviewArtifact = routeView === "findings"
    ? findingsArtifact ?? overviewArtifact ?? coverageArtifact ?? allArtifacts[0]
    : routeView === "evidence"
      ? coverageArtifact ?? overviewArtifact ?? allArtifacts[0]
      : overviewArtifact ?? coverageArtifact ?? allArtifacts[0];
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
  }

  useEffect(() => {
    const routeRequiresPreferredArtifact = routeView === "findings" && selectedArtifact !== preferredReviewArtifact?.path;
    if (preferredReviewArtifact && (!selectedArtifact || routeRequiresPreferredArtifact)) {
      openReviewArtifactRef.current(preferredReviewArtifact.path);
    }
  }, [preferredReviewArtifact, routeView, selectedArtifact]);

  return (
    <div className="stage-stack" data-testid="review-panel">
      <section className="panel stage-panel" data-testid="results-coverage-panel">
        <div className="stage-header">
          <div>
            <h1>{routeView === "overview" ? "Review" : capitalize(routeView)}</h1>
            <p className="hint">{reviewRouteDescription(routeView)}</p>
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
        <ReviewEvidenceWorkbench
          routeView={routeView}
          coverageSummary={coverageSummary}
          openQuestions={openQuestions}
          openQuestionCount={openQuestionCount}
          trustStatus={trustStatus}
          overviewArtifact={overviewArtifact}
          findingsArtifact={findingsArtifact}
          artifactGroups={artifactGroups}
          diagramArtifacts={diagramArtifacts}
          selectedArtifact={selectedArtifact}
          selectedArtifactContent={selectedArtifactContent}
          evidenceStatus={evidenceStatus}
          evidenceIssues={evidenceIssues}
          selectedArtifactIsLoading={selectedArtifactIsLoading}
          runLogs={runLogs}
          reviewSummary={reviewSummary}
          demo={demo}
          reviewQueue={reviewQueue}
          gitDiff={gitDiff}
          gitDiffStatus={gitDiffStatus}
          onLoadGitDiff={onLoadGitDiff}
          onOpenArtifact={onOpenArtifact}
        />
        <details className="review-domain-map-disclosure">
          <summary data-testid="review-domain-map-toggle">Domain map preview</summary>
          <ReviewDomainMap domainMap={domainMap} onOpenArtifact={handleOpenDomainMapArtifact} />
        </details>
      </section>
    </div>
  );
}

function reviewRouteDescription(view: "overview" | "evidence" | "findings" | "diff"): string {
  switch (view) {
    case "overview": return "Validate run identity, coverage and review readiness before publication.";
    case "evidence": return "Inspect immutable selected-run evidence and its exact artifact identity.";
    case "findings": return "Resolve findings, coverage questions and decision blockers.";
    case "diff": return "Inspect the server-authoritative current workspace Git diff.";
  }
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
  routeView,
  coverageSummary,
  openQuestions,
  openQuestionCount,
  trustStatus,
  overviewArtifact,
  findingsArtifact,
  artifactGroups,
  diagramArtifacts,
  selectedArtifact,
  selectedArtifactContent,
  evidenceStatus,
  evidenceIssues,
  selectedArtifactIsLoading,
  reviewSummary,
  demo,
  reviewQueue,
  gitDiff,
  gitDiffStatus,
  onLoadGitDiff,
  onOpenArtifact,
}: {
  routeView: "overview" | "evidence" | "findings" | "diff";
  coverageSummary: string;
  openQuestions: string;
  openQuestionCount: number;
  trustStatus: ReviewTrustStatus;
  overviewArtifact?: Artifact;
  findingsArtifact?: Artifact;
  artifactGroups: ArtifactGroup[];
  diagramArtifacts: Artifact[];
  selectedArtifact: string;
  selectedArtifactContent: string;
  evidenceStatus: "idle" | "loading" | "available" | "partial" | "not_produced" | "unavailable" | "error";
  evidenceIssues: Array<{ code: string; message: string; path?: string }>;
  selectedArtifactIsLoading: boolean;
  runLogs: RunLogEntry[];
  reviewSummary: RunReviewSummaryResponse | null;
  demo: boolean;
  reviewQueue: ReviewQueueItem[];
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  onLoadGitDiff: (options: LoadGitDiffOptions) => void;
  onOpenArtifact: (path: string) => void;
}) {
  const evidenceView: "preview" | "diff" = routeView === "diff" ? "diff" : "preview";
  const [artifactFilter, setArtifactFilter] = useState<ReviewArtifactFilter>("all");
  const [artifactExplorerOpen, setArtifactExplorerOpen] = useState(reviewQueue.length === 0);
  const visibleArtifactGroups = filterReviewArtifactGroups(artifactGroups, artifactFilter);
  const visibleArtifactCount = visibleArtifactGroups.reduce((sum, group) => sum + group.artifacts.length, 0);

  useEffect(() => {
    if (evidenceView === "diff" && selectedArtifact) {
      onLoadGitDiff({ path: selectedArtifact });
    }
  }, [evidenceView, onLoadGitDiff, selectedArtifact]);

  useEffect(() => {
    if (reviewQueue.length === 0) {
      setArtifactExplorerOpen(true);
    }
  }, [reviewQueue.length]);

  return (
    <div className="review-workbench">
      <aside className="review-task-lane" id="review-task-lane" aria-label="Review tasks and supporting artifacts">
        <ReviewQueuePanel queue={reviewQueue} selectedArtifact={selectedArtifact} onOpenArtifact={onOpenArtifact} />
        <details
          className="review-artifact-explorer"
          data-testid="review-artifact-explorer"
          id="review-artifacts"
          open={artifactExplorerOpen}
          onToggle={(event) => setArtifactExplorerOpen(event.currentTarget.open)}
        >
          <summary className="review-artifact-explorer-summary" data-testid="review-artifact-explorer-toggle">
            <span className="review-artifact-summary-copy">
              <strong>Artifact explorer</strong>
              <span>Secondary browser for all generated files.</span>
            </span>
            <StatusBadge tone={visibleArtifactGroups.length > 0 ? "ok" : "info"}>
              {artifactFilter === "all" ? `${artifactGroups.length} groups` : `${visibleArtifactCount} refs`}
            </StatusBadge>
          </summary>
          <div className="review-artifact-explorer-body">
            <TabNav
              ariaLabel="Review artifact filters"
              className="artifact-filter-tabs"
              idBase="review-artifact-filters"
              testId="review-artifact-filters"
              value={artifactFilter}
              onChange={(filter) => {
                setArtifactFilter(filter);
                setArtifactExplorerOpen(true);
              }}
              options={REVIEW_ARTIFACT_FILTERS}
            />
            <div {...tabPanelProps("review-artifact-filters", artifactFilter)}>
              {visibleArtifactGroups.length === 0 ? (
                <p className="hint">
                  {artifactGroups.length === 0
                    ? "No selected-run artifacts yet. Run Analysis before evidence review."
                    : `No ${reviewArtifactFilterLabel(artifactFilter).toLowerCase()} artifacts are available in this run.`}
                </p>
              ) : (
                <div className="artifact-group-list" data-testid="results-artifacts-panel">
                  {visibleArtifactGroups.map((group) => (
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
            </div>
          </div>
        </details>
      </aside>

      <section className="review-evidence-preview" id="review-evidence-preview" data-testid="review-evidence-preview">
        <div className="section-heading-row">
          <div>
            <h2>Evidence preview</h2>
            <p className="hint">Select an artifact to inspect the reviewable evidence body.</p>
          </div>
          <span className="status">Validator-approved snapshot · human review is recorded through publication</span>
        </div>
        <div className="review-mode-content">
          {evidenceView === "preview" ? (
            selectedArtifactIsLoading ? <p className="hint">Loading evidence...</p> : (
              <EvidenceViewer
                path={selectedArtifact}
                content={selectedArtifactContent}
                runId={reviewSummary?.run_id}
                sourceMode="run_snapshot"
                provenance={demo ? "demo" : reviewSummary ? "live" : "unknown"}
                status={evidenceStatus === "partial" ? "partial" : evidenceStatus === "error" ? "error" : evidenceStatus === "unavailable" || evidenceStatus === "not_produced" ? "unavailable" : "available"}
                issues={evidenceIssues}
                onOpenArtifact={onOpenArtifact}
              />
            )
          ) : null}
          {evidenceView === "diff" ? <GitDiffView gitDiff={gitDiff} status={gitDiffStatus} onSelectFile={(path) => onLoadGitDiff({ path })} /> : null}
        </div>
      </section>

      <aside className="review-intel" id="review-trust" data-testid="review-citation-coverage">
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
          <details>
            <summary>Coverage summary</summary>
            <pre data-testid="coverage-summary-content">{coverageSummary || "No coverage summary yet."}</pre>
          </details>
          <details>
            <summary>Open questions · {openQuestionCount}</summary>
            <pre data-testid="open-questions-content">{openQuestions || "No open questions yet."}</pre>
          </details>
        </div>
      </aside>

    </div>
  );
}

function ReviewQueuePanel({
  queue,
  selectedArtifact,
  onOpenArtifact,
}: {
  queue: ReviewQueueItem[];
  selectedArtifact: string;
  onOpenArtifact: (path: string) => void;
}) {
  return (
    <section className="review-queue" id="review-queue" data-testid="review-queue">
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
                className={`review-queue-item${item.path === selectedArtifact ? " is-selected" : ""}`}
                aria-current={item.path === selectedArtifact ? "true" : undefined}
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

function filterReviewArtifactGroups(groups: ArtifactGroup[], filter: ReviewArtifactFilter): ArtifactGroup[] {
  if (filter === "all") {
    return groups;
  }
  return groups
    .map((group) => ({
      ...group,
      artifacts: group.artifacts.filter((artifact) => reviewArtifactMatchesFilter(artifact, filter)),
    }))
    .filter((group) => group.artifacts.length > 0);
}

function reviewArtifactMatchesFilter(artifact: Artifact, filter: ReviewArtifactFilter): boolean {
  const path = artifact.path;
  if (filter === "diagrams") {
    return artifact.kind === "diagram" || path.startsWith("reports/diagrams/");
  }
  if (filter === "proposals") {
    return artifact.kind === "proposal" || artifact.kind === "changelog" || path.startsWith("proposals/") || path.startsWith("reports/changelog/");
  }
  if (filter === "runtime") {
    return artifact.kind === "taskrun" || path.startsWith("reports/taskruns/");
  }
  if (filter === "reports") {
    return (
      (artifact.kind === "report" || artifact.kind === "agent-output" || path.startsWith("reports/")) &&
      !path.startsWith("reports/diagrams/") &&
      !path.startsWith("reports/changelog/") &&
      !path.startsWith("reports/taskruns/")
    );
  }
  return true;
}

function reviewArtifactFilterLabel(filter: ReviewArtifactFilter): string {
  return REVIEW_ARTIFACT_FILTERS.find((option) => option.id === filter)?.label ?? "Selected";
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
  const openProposalArtifactRef = useRef(onOpenArtifact);
  openProposalArtifactRef.current = onOpenArtifact;
  const proposalReview = deriveProposalReviewModel({ artifacts, openQuestions });
  const selectedProposalArtifact = proposalReview.proposalArtifacts.find((artifact) => artifact.path === selectedArtifact);
  const preferredProposalArtifact =
    proposalReview.proposalArtifacts.find((artifact) => artifact.path.startsWith("proposals/") && /(^|\/)proposal\.md$/i.test(artifact.path)) ??
    proposalReview.proposalArtifacts.find((artifact) => artifact.path.startsWith("proposals/")) ??
    proposalReview.changelogArtifacts[0];
  const selectedProposalIsLoading = selectedArtifactContent === "Loading...";

  useEffect(() => {
    if (!selectedProposalArtifact && preferredProposalArtifact && selectedArtifact !== preferredProposalArtifact.path) {
      openProposalArtifactRef.current(preferredProposalArtifact.path);
    }
  }, [preferredProposalArtifact, selectedArtifact, selectedProposalArtifact]);

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
      {proposalReview.blockers.length > 0 ? (
        <ProposalPackageRecoveryPanel
          proposalReview={proposalReview}
          preferredArtifact={preferredProposalArtifact}
          proposalBranch={proposalBranch}
          gitStatus={gitStatus}
          onOpenArtifact={onOpenArtifact}
          onGoPublish={onGoPublish}
        />
      ) : null}
      <div className="proposals-review-room" data-testid="proposals-review-room">
        <aside className="proposals-artifact-list" data-testid="proposals-artifact-list">
          <div className="section-heading-row">
            <h2>Proposal packages</h2>
            <StatusBadge tone={proposalReview.proposalArtifacts.length > 0 ? "ok" : "info"}>{proposalReview.packages.length} groups</StatusBadge>
          </div>
          {proposalReview.packages.length === 0 ? (
            <p className="hint">No proposal or changelog artifacts yet. Run Analysis until proposals are generated before publication review.</p>
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
          <TabNav
            ariaLabel="Proposal review tabs"
            className="proposal-preview-tabs"
            idBase="proposal-preview-tabs"
            testId="proposal-preview-tabs"
            value={proposalView}
            onChange={setProposalView}
            options={(["preview", "evidence", "changelog", "diff", "logs"] as const).map((view) => ({ id: view, label: proposalTabLabel(view) }))}
          />

          {proposalView === "preview" ? (
            <div className="proposal-tab-panel" {...tabPanelProps("proposal-preview-tabs", proposalView)}>
              <h3>{selectedProposalArtifact?.path || "Select a proposal artifact"}</h3>
              <pre>{selectedProposalArtifact ? (selectedProposalIsLoading ? "Loading proposal..." : selectedArtifactContent || "No preview content returned.") : "Select a proposal, ADR, RFC or checklist artifact from the package list."}</pre>
            </div>
          ) : null}

          {proposalView === "evidence" ? (
            <div className="proposal-tab-panel" {...tabPanelProps("proposal-preview-tabs", proposalView)}>
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
            <div className="proposal-tab-panel" {...tabPanelProps("proposal-preview-tabs", proposalView)}>
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
            <div className="proposal-tab-panel" {...tabPanelProps("proposal-preview-tabs", proposalView)}>
              <h3>Diff preview</h3>
              <GitDiffView gitDiff={gitDiff} status={gitDiffStatus} onSelectFile={(path) => onLoadGitDiff({ path })} />
            </div>
          ) : null}

          {proposalView === "logs" ? (
            <div className="proposal-tab-panel" {...tabPanelProps("proposal-preview-tabs", proposalView)}>
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

function ProposalPackageRecoveryPanel({
  proposalReview,
  preferredArtifact,
  proposalBranch,
  gitStatus,
  onOpenArtifact,
  onGoPublish,
}: {
  proposalReview: ProposalReviewModel;
  preferredArtifact: Artifact | undefined;
  proposalBranch: string;
  gitStatus: string;
  onOpenArtifact: (path: string) => void;
  onGoPublish: () => void;
}) {
  const primaryBlocker = proposalReview.blockers[0] ?? "No proposal package blocker detected.";
  const suggestedFix = proposalPackageSuggestedFix(primaryBlocker);
  const packageState = proposalReview.proposalDocumentArtifacts.length > 0 ? `${proposalReview.packages.length} artifact groups` : "proposal missing";
  const publicationPath =
    proposalReview.blockers.length > 0
      ? "Keep Publish as review-only until proposal, changelog and evidence blockers are resolved."
      : proposalBranch
        ? `Ready for Publish review on ${proposalBranch}.`
        : "Ready for Publish review; prepare a proposal branch before handoff.";

  return (
    <section className="proposal-recovery-panel" data-testid="proposal-package-recovery">
      <div className="section-heading-row">
        <div>
          <h2>Proposal package recovery</h2>
          <p className="hint">Resolve proposal/changelog gaps before treating this run as publication-ready.</p>
        </div>
        <StatusBadge tone="warn">proposal blocked</StatusBadge>
      </div>
      <div className="proposal-recovery-grid">
        <div>
          <span className="meta-label">Package state</span>
          <strong>{packageState}</strong>
        </div>
        <div>
          <span className="meta-label">Proposal docs</span>
          <strong>{proposalReview.proposalDocumentCount}</strong>
        </div>
        <div>
          <span className="meta-label">ADR/RFC</span>
          <strong>{proposalReview.adrRfcCount}</strong>
        </div>
        <div>
          <span className="meta-label">Changelog</span>
          <strong>{proposalReview.changelogArtifacts.length}</strong>
        </div>
        <div>
          <span className="meta-label">Evidence refs</span>
          <strong>{proposalReview.evidenceArtifacts.length}</strong>
        </div>
      </div>
      <dl className="compact-defs proposal-recovery-detail">
        <div>
          <dt>Primary blocker</dt>
          <dd>{primaryBlocker}</dd>
        </div>
        <div>
          <dt>Suggested fix</dt>
          <dd>{suggestedFix}</dd>
        </div>
        <div>
          <dt>Publication path</dt>
          <dd>
            {publicationPath}
            {gitStatus ? ` ${gitStatus}` : ""}
          </dd>
        </div>
      </dl>
      <div className="actions proposal-recovery-actions">
        {preferredArtifact ? (
          <button type="button" className="secondary" onClick={() => onOpenArtifact(preferredArtifact.path)}>
            Open available artifact
          </button>
        ) : null}
        <button type="button" className="secondary" onClick={onGoPublish}>
          Check Publish gate
        </button>
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
  proposalDocumentArtifacts: Artifact[];
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
    proposalDocumentArtifacts,
    changelogArtifacts,
    evidenceArtifacts,
    packages,
    proposalDocumentCount,
    adrRfcCount,
    blockers,
  };
}

function proposalPackageSuggestedFix(blocker: string): string {
  if (blocker.includes("No proposal package")) {
    return "Retry or rerun Analysis step4.proposals, then confirm a generated proposals/* artifact appears before Publish.";
  }
  if (blocker.includes("ADR or RFC")) {
    return "Generate or add an ADR/RFC draft under proposals/* so reviewers can see the decision record or implementation plan.";
  }
  if (blocker.includes("No changelog")) {
    return "Regenerate proposals so reports/changelog/* records the iteration changes linked to the package.";
  }
  if (blocker.includes("open questions")) {
    return "Resolve the Review open questions or record an explicit accepted gap before publication handoff.";
  }
  return "Inspect the proposal package artifacts, resolve blockers, and use Publish only after the package is complete.";
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
  onProposalCreated,
}: {
  primaryActionSignal?: number;
  onOpenArtifact: (path: string) => void;
  onProposalCreated?: (proposal: QAProposalDraftResponse) => void;
}) {
  const [question, setQuestion] = useState("");
  const [qaRun, setQARun] = useState<QARunResponse | null>(null);
  const [runHistory, setRunHistory] = useState<QARunResponse[]>([]);
  const [selectedRunID, setSelectedRunID] = useState<string | null>(null);
  const [historyStatus, setHistoryStatus] = useState("Loading Q&A history.");
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  const [selectedLoading, setSelectedLoading] = useState(false);
  const [proposalConfirmationOpen, setProposalConfirmationOpen] = useState(false);
  const [proposalTitle, setProposalTitle] = useState("");
  const [proposalOperatorNote, setProposalOperatorNote] = useState("");
  const [proposalBusy, setProposalBusy] = useState(false);
  const historyRequest = useRequestGate("qa-history");
  const detailRequest = useRequestGate("qa-detail");
  const pollRequest = useRequestGate("qa-poll");
  const selectedRunIDRef = useRef<string | null>(null);
  const qaRunRef = useRef<QARunResponse | null>(null);
  const selectionSequenceRef = useRef(0);
  const qaRunActive = qaRun?.status === "queued" || qaRun?.status === "running";
  const citations = qaRun?.citations ?? [];
  const unresolved = qaRun?.unresolved ?? [];
  const confidence = typeof qaRun?.confidence === "number" ? Math.round(qaRun.confidence * 100) : 0;

  function claimQASelection(runID: string): number {
    selectionSequenceRef.current += 1;
    selectedRunIDRef.current = runID;
    setSelectedRunID(runID);
    return selectionSequenceRef.current;
  }

  function isCurrentQASelection(selectionVersion: number, runID: string): boolean {
    return selectionSequenceRef.current === selectionVersion && selectedRunIDRef.current === runID;
  }

  useEffect(() => {
    selectedRunIDRef.current = selectedRunID;
  }, [selectedRunID]);

  useEffect(() => {
    qaRunRef.current = qaRun;
  }, [qaRun]);

  useEffect(() => {
    const selectionVersion = selectionSequenceRef.current;
    const historyToken = historyRequest.begin("initial");
    async function loadHistory() {
      try {
        const history = await listQARuns(20, historyToken.signal);
        if (!historyRequest.isCurrent(historyToken)) {
          return;
        }
        const items = history.items ?? [];
        const currentRun = qaRunRef.current;
        const visibleItems = currentRun ? mergeQARunHistory(currentRun, items, "preserve") : items;
        setRunHistory(visibleItems);
        setHistoryStatus(visibleItems.length > 0 ? "" : "No Q&A runs yet.");
        if (items[0]?.run_id && selectedRunIDRef.current === null && selectionSequenceRef.current === selectionVersion) {
          const selectedVersion = claimQASelection(items[0].run_id);
          setQARun(items[0]);
          const detailToken = detailRequest.begin(items[0].run_id);
          try {
            const detail = await getQARun(items[0].run_id, detailToken.signal);
            if (detailRequest.isCurrent(detailToken) && isCurrentQASelection(selectedVersion, items[0].run_id)) {
              setQARun(detail);
              setRunHistory((current) => mergeQARunHistory(detail, current, "preserve"));
              setHistoryStatus("");
            }
          } catch (error) {
            if (!isAbortError(error) && detailRequest.isCurrent(detailToken) && isCurrentQASelection(selectedVersion, items[0].run_id)) {
              setStatus(error instanceof Error ? error.message : "Q&A run detail failed");
            }
          } finally {
            detailRequest.finish(detailToken);
          }
        }
      } catch (error) {
        if (!isAbortError(error) && historyRequest.isCurrent(historyToken)) {
          setHistoryStatus(error instanceof Error ? error.message : "Q&A history failed to load");
        }
      } finally {
        historyRequest.finish(historyToken);
      }
    }
    void loadHistory();
    return () => {
      historyRequest.abort();
      detailRequest.abort();
    };
  }, []);

  useEffect(() => {
    if (!qaRun?.run_id || !qaRunActive) {
      return;
    }
    const runID = qaRun.run_id;
    let canceled = false;
    const refresh = async () => {
      const token = pollRequest.begin(runID);
      try {
        const next = await getQARun(runID, token.signal);
        if (!canceled && pollRequest.isCurrent(token) && selectedRunIDRef.current === runID) {
          setQARun(next);
          setSelectedRunID(next.run_id);
          setRunHistory((current) => mergeQARunHistory(next, current, "preserve"));
          setHistoryStatus("");
          setStatus(next.status === "succeeded" ? "Q&A run completed." : next.status === "failed" ? "Q&A run failed." : "Q&A run is running.");
        }
      } catch (error) {
        if (!isAbortError(error) && !canceled && pollRequest.isCurrent(token) && selectedRunIDRef.current === runID) {
          setStatus(error instanceof Error ? error.message : "Q&A run polling failed");
        }
      } finally {
        pollRequest.finish(token);
      }
    };
    const interval = window.setInterval(() => void refresh(), 1000);
    return () => {
      canceled = true;
      pollRequest.abort();
      window.clearInterval(interval);
    };
  }, [qaRun?.run_id, qaRunActive]);

  async function refreshHistory() {
    const token = historyRequest.begin("manual");
    setHistoryStatus("Refreshing Q&A history.");
    try {
      const history = await listQARuns(20, token.signal);
      if (!historyRequest.isCurrent(token)) {
        return;
      }
      const items = history.items ?? [];
      const currentRun = qaRunRef.current;
      const mergedItems = currentRun ? mergeQARunHistory(currentRun, items, "preserve") : items;
      setRunHistory(mergedItems);
      setHistoryStatus(mergedItems.length > 0 ? "" : "No Q&A runs yet.");
    } catch (error) {
      if (!isAbortError(error) && historyRequest.isCurrent(token)) {
        setHistoryStatus(error instanceof Error ? error.message : "Q&A history failed to load");
      }
    } finally {
      historyRequest.finish(token);
    }
  }

  async function handleSelectRun(run: QARunResponse) {
    const selectionVersion = claimQASelection(run.run_id);
    const token = detailRequest.begin(run.run_id);
    setQARun(run);
    setSelectedLoading(true);
    setStatus("");
    try {
      const detail = await getQARun(run.run_id, token.signal);
      if (detailRequest.isCurrent(token) && isCurrentQASelection(selectionVersion, run.run_id)) {
        setQARun(detail);
        setRunHistory((current) => mergeQARunHistory(detail, current, "preserve"));
        setHistoryStatus("");
        setStatus(detail.status === "succeeded" ? "Q&A run loaded." : detail.status === "failed" ? "Q&A run failed." : "Q&A run is running.");
      }
    } catch (error) {
      if (!isAbortError(error) && detailRequest.isCurrent(token) && isCurrentQASelection(selectionVersion, run.run_id)) {
        setStatus(error instanceof Error ? error.message : "Q&A run detail failed");
      }
    } finally {
      if (detailRequest.isCurrent(token) && isCurrentQASelection(selectionVersion, run.run_id)) {
        setSelectedLoading(false);
      }
      detailRequest.finish(token);
    }
  }

  async function reloadSelectedAnswer() {
    if (!qaRun?.run_id) return;
    await handleSelectRun(qaRun);
  }

  function openProposalConfirmation() {
    if (!qaRun?.answer_digest || qaRun.status !== "succeeded") return;
    setProposalTitle((qaRun.question || "Ask synthesis").trim());
    setProposalOperatorNote("");
    setStatus("");
    setProposalConfirmationOpen(true);
  }

  async function handleCreateProposalDraft() {
    if (!qaRun?.run_id || !qaRun.answer_digest) return;
    const title = proposalTitle.trim();
    if (!title) {
      setStatus("Proposal title is required.");
      return;
    }
    setProposalBusy(true);
    setStatus("Creating proposal draft from the immutable Ask answer.");
    try {
      const proposal = await createQAProposalDraft(qaRun.run_id, {
        title,
        expected_answer_digest: qaRun.answer_digest,
        operator_note: proposalOperatorNote.trim() || undefined,
      });
      setProposalConfirmationOpen(false);
      setStatus(`Proposal draft created at ${proposal.path}.`);
      onProposalCreated?.(proposal);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Proposal draft creation failed");
    } finally {
      setProposalBusy(false);
    }
  }

  async function handleAsk() {
    const trimmed = question.trim();
    if (!trimmed) {
      setStatus("Question is required.");
      return;
    }
    await startQARun(trimmed);
  }

  async function handleRetryQA() {
    const retryQuestion = (qaRun?.question || question).trim();
    if (!retryQuestion) {
      setStatus("Original question is unavailable.");
      return;
    }
    setQuestion(retryQuestion);
    await startQARun(retryQuestion);
  }

  async function startQARun(trimmed: string) {
    setBusy(true);
    setStatus("Submitting Q&A run.");
    try {
      const started = await startQAQuestion(trimmed);
      const provisionalRun = buildProvisionalQARun(started, trimmed);
      const selectionVersion = claimQASelection(started.run_id);
      const token = detailRequest.begin(started.run_id);
      setQARun(provisionalRun);
      setRunHistory((current) => mergeQARunHistory(provisionalRun, current));
      setHistoryStatus("");
      setStatus(`Q&A run ${started.run_id} accepted; reconciling details.`);
      try {
        const detail = await getQARun(started.run_id, token.signal);
        if (detailRequest.isCurrent(token) && isCurrentQASelection(selectionVersion, started.run_id)) {
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
        }
      } catch (error) {
        if (!isAbortError(error) && detailRequest.isCurrent(token) && isCurrentQASelection(selectionVersion, started.run_id)) {
          setStatus(`Q&A run ${started.run_id} accepted; reconciling details failed: ${qaErrorMessage(error, "Q&A run detail temporarily unavailable")}`);
        }
      } finally {
        detailRequest.finish(token);
      }
    } catch (error) {
      if (!isAbortError(error)) {
        setStatus(error instanceof Error ? error.message : "Q&A request failed");
      }
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
        <StatusBadge tone={runOutcomeTone(qaRun)}>{qaRunProviderLabel(qaRun)}</StatusBadge>
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
                      <StatusBadge tone={runOutcomeTone(run)}>{runOutcomeLabel(run, "unknown")}</StatusBadge>
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
                  Run <code>{qaRun.run_id}</code> status: <strong>{runOutcomeLabel(qaRun, "unknown")}</strong>
                </p>
                <p>Runtime provider: {qaRunProviderLabel(qaRun)}</p>
              </div>
              <a className="link-button" href={`/api/pipeline/runs/${encodeURIComponent(qaRun.run_id)}/logs`} target="_blank" rel="noreferrer">
                Open run logs
              </a>
              {qaRun.error ? <p className="status err">{qaRun.error}</p> : null}
              {(qaRun.warnings ?? []).length > 0 ? <p className="status warn">Warnings: {(qaRun.warnings ?? []).join(", ")}</p> : null}
            </div>
          ) : null}

          <QAFailureRecovery qaRun={qaRun} busy={busy || qaRunActive} onRetry={() => void handleRetryQA()} />

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
              {qaRun.status === "succeeded" && qaRun.answer_digest ? (
                <button type="button" onClick={openProposalConfirmation} data-testid="qa-create-proposal-btn">
                  Create proposal draft
                </button>
              ) : null}
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
      <ModalDialog
        open={proposalConfirmationOpen}
        title="Create proposal draft"
        description="This explicit mutation creates a new current-workspace proposal package. The selected Ask taskrun and source repositories remain read-only."
        confirmLabel="Create proposal draft"
        busy={proposalBusy}
        onCancel={() => setProposalConfirmationOpen(false)}
        onConfirm={() => void handleCreateProposalDraft()}
      >
        <label htmlFor="qaProposalTitle">Proposal title</label>
        <input id="qaProposalTitle" value={proposalTitle} onChange={(event) => setProposalTitle(event.target.value)} autoFocus />
        <label htmlFor="qaProposalNote">Operator note (optional)</label>
        <textarea id="qaProposalNote" value={proposalOperatorNote} onChange={(event) => setProposalOperatorNote(event.target.value)} rows={3} />
        <p><strong>Target:</strong> `proposals/qa-synthesis-{qaRun?.run_id ?? "run"}-&lt;slug&gt;/`</p>
        <p><strong>Citations:</strong> {citations.length}</p>
        <p><strong>Unresolved assumptions:</strong> {unresolved.length > 0 ? unresolved.join(", ") : "none"}</p>
        <p className="hint">Ask remains read-only; only the new proposal package is written.</p>
        <button type="button" className="secondary" onClick={() => void reloadSelectedAnswer()} disabled={proposalBusy}>
          Reload selected answer
        </button>
      </ModalDialog>
    </section>
  );
}

function buildProvisionalQARun(started: { run_id: string; status: string }, question: string): QARunResponse {
  return {
    run_id: started.run_id,
    pipeline: "qa",
    status: normalizeQARunStatus(started.status),
    started_at: new Date().toISOString(),
    finished_at: null,
    question,
    current_step: "qa.ask",
    answer: null,
    citations: [],
    unresolved: [],
    confidence: null,
    generated_at: null,
    warnings: [],
    error_code: null,
    error: null,
  };
}

function normalizeQARunStatus(status: string): QARunResponse["status"] {
  if (status === "queued" || status === "running" || status === "succeeded" || status === "failed") {
    return status;
  }
  return "queued";
}

function qaErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

function QAFailureRecovery({
  qaRun,
  busy,
  onRetry,
}: {
  qaRun: QARunResponse | null;
  busy: boolean;
  onRetry: () => void;
}) {
  if (qaRun?.status !== "failed") {
    return null;
  }

  const errorCode = qaRun.error_code || "unclassified";
  const blockedStep = qaRun.current_step || "qa.ask";
  const warningCount = qaRun.warnings?.length ?? 0;
  const auditRefs = `reports/taskruns/${qaRun.run_id}/qa/`;
  const canRetry = Boolean((qaRun.question || "").trim());
  const canceled = isRunCanceled(errorCode);
  const reconciled = isRunReconciledAfterRestart(errorCode);
  const title = canceled ? "Canceled answer run" : reconciled ? "Recovered answer run" : "Recovery path";
  const badgeLabel = canceled ? "canceled" : reconciled ? "recovered" : "failed";
  const badgeTone = canceled || reconciled ? "warn" : "error";
  const stepLabel = canceled ? "Stopped step" : reconciled ? "Recovered step" : "Blocked step";
  const retryLabel = canceled || reconciled ? "Ask again" : "Retry question";
  const retentionHint = canceled
    ? "Asking again creates a new Q&A run; the canceled attempt and QA audit artifacts stay in history."
    : reconciled
      ? "Asking again creates a new Q&A run; the reconciled attempt and QA audit artifacts stay in history."
      : "Retry starts a new Q&A run; the failed answer attempt stays in history for audit.";

  return (
    <section className="qa-recovery-panel" data-testid="qa-failure-recovery">
      <div className="section-heading-row">
        <div>
          <h2>{title}</h2>
          <p className="hint">{qaFailureGuidance(errorCode, warningCount)}</p>
        </div>
        <StatusBadge tone={badgeTone}>{badgeLabel}</StatusBadge>
      </div>
      <div className="qa-recovery-grid">
        <div>
          <span className="metric-label">Classification</span>
          <strong>{errorCode}</strong>
        </div>
        <div>
          <span className="metric-label">{stepLabel}</span>
          <strong>{blockedStep}</strong>
        </div>
        <div>
          <span className="metric-label">Audit evidence</span>
          <strong>{auditRefs}</strong>
        </div>
        <div>
          <span className="metric-label">Warnings</span>
          <strong>{warningCount}</strong>
        </div>
      </div>
      {qaRun.error ? <p className="status err">{qaRun.error}</p> : null}
      {warningCount > 0 ? <p className="status warn">Warnings: {(qaRun.warnings ?? []).join(", ")}</p> : null}
      <div className="actions qa-recovery-actions">
        <button type="button" data-testid="qa-retry-run-btn" onClick={onRetry} disabled={busy || !canRetry}>
          {retryLabel}
        </button>
        <a className="link-button" href={`/api/pipeline/runs/${encodeURIComponent(qaRun.run_id)}/logs`} target="_blank" rel="noreferrer">
          Open run logs
        </a>
      </div>
      <p className="hint">{retentionHint}</p>
    </section>
  );
}

function qaFailureGuidance(errorCode: string, warningCount: number): string {
  if (isRunCanceled(errorCode)) {
    return "The answer run stopped by request. Review QA audit artifacts, then ask again when ready.";
  }
  if (isRunReconciledAfterRestart(errorCode)) {
    return "ACP reconciled a stale answer run after restart. Review QA audit artifacts, then ask again if the question still matters.";
  }
  if (errorCode === "runtime_permission_required") {
    return "Resolve the runtime permission request, then retry the question.";
  }
  if (isRunnerUnavailable(errorCode)) {
    return "Provider/tool availability blocked the answer; check Readiness provider setup, binary/auth/quota, then ask again.";
  }
  if (errorCode.includes("runtime_timeout")) {
    return "The answer run exhausted its time budget; inspect the last progress signal before retry.";
  }
  if (errorCode.includes("runtime_contract")) {
    return "The answer artifact did not pass validation; inspect audit artifacts before retry.";
  }
  if (warningCount > 0) {
    return "Review warnings and audit artifacts, then retry when the issue is understood.";
  }
  return "Review logs and audit artifacts, then retry the same question when the cause is clear.";
}

function qaRunProviderLabel(run: QARunResponse | null): string {
  if (!run) {
    return "agent-backed";
  }
  if (run.provider === "fake") {
    return "fake";
  }
  return run.provider || run.runtime_provider || "agent-backed";
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
  gitError,
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
  const [artifactFilter, setArtifactFilter] = useState<PublishArtifactFilter>("all");
  const publishArtifacts = artifacts
    .filter((artifact) => artifact.path.trim().length > 0)
    .sort(comparePublishArtifactPriority);
  const changedPathSet = new Set(gitDiff?.files.map((file) => file.path) ?? []);
  const filteredPublishArtifacts = publishArtifacts.filter((artifact) => publishArtifactMatchesFilter(artifact, artifactFilter, changedPathSet));
  const visibleChangedFiles = artifactFilter === "all" || artifactFilter === "changed" ? (gitDiff?.files ?? []) : [];
  const selectedDiffFilePath = gitDiff?.selected_file?.path;
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
  const openQuestionCount = countMarkdownItems(openQuestions);
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
  const diffScopeTitle = gitDiffScopeTitle(gitDiff);
  const diffScopeHint = gitDiffScopeHint(gitDiff);
  const primaryPublishGateItem = blockingGateItems[0] ?? openQuestionGateItems[0] ?? warningGateItems[0];
  const publishGateTone = blockingGateItems.length > 0 ? "error" : warningGateItems.length > 0 || openQuestionGateItems.length > 0 ? "warn" : "ok";
  const publishGateLabel = blockingGateItems.length > 0 ? "blocked" : warningGateItems.length > 0 || openQuestionGateItems.length > 0 ? "review" : "ready";
  const publishGateDetail = primaryPublishGateItem ? `${primaryPublishGateItem.label}: ${primaryPublishGateItem.detail}` : "Git actions are allowed after final review.";
  const gitActionLabel = gitError ? "failed" : blockingGateItems.length > 0 ? "blocked" : gitMessage.trim() ? "ready" : "needs message";
  const gitActionDetail =
    gitError
      ? gitError
      : blockingGateItems.length > 0
      ? `${blockingGateItems[0].label}: ${blockingGateItems[0].detail}`
      : gitMessage.trim()
        ? `Commit message prepared: ${gitMessage}`
        : "Commit message is empty.";

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
        <div className="publish-readiness-summary" data-testid="publish-readiness-summary" aria-label="Publish readiness summary">
          <div>
            <span className="metric-label">Publication set</span>
            <strong>{publishArtifacts.length} refs</strong>
            <span>{visibleFolderSummaries.length} folders in scope</span>
          </div>
          <div>
            <span className="metric-label">Gate</span>
            <strong>{publishGateLabel}</strong>
            <span>{publishGateDetail}</span>
          </div>
          <div>
            <span className="metric-label">Open questions</span>
            <strong>{openQuestionCount}</strong>
            <span>{openQuestionCount > 0 ? "Review before commit." : "No loaded open questions."}</span>
          </div>
          <div>
            <span className="metric-label">Git action</span>
            <strong>{gitActionLabel}</strong>
            <span>{gitActionDetail}</span>
          </div>
        </div>
      </section>

      <nav className="publish-section-jumps" aria-label="Publish sections" data-testid="publish-section-jumps">
        <a href="#publish-diff-summary">Diff</a>
        <a href="#publish-preview-panel">Preview</a>
        <a href="#publish-gate-panel">Gate</a>
        <a href="#publish-commit-plan">Commit</a>
      </nav>

      <div className="publish-review-room">
        <section className="publish-diff-summary" id="publish-diff-summary" data-testid="publish-diff-summary">
          <div className="panel-subheader">
            <div>
              <h2>{diffScopeTitle}</h2>
              <p className="hint">{diffScopeHint}</p>
            </div>
            <StatusBadge tone={gitDiff && !gitDiff.empty ? "ok" : publishArtifacts.length > 0 ? "info" : "warn"}>
              {gitDiff ? `${gitDiff.files.length} changed` : `${publishArtifacts.length} refs`}
            </StatusBadge>
          </div>
          <div className="actions compact-actions">
            <button type="button" className="secondary-action" onClick={() => onLoadGitDiff({})}>
              Load selected run diff
            </button>
            <button type="button" className="secondary-action" onClick={() => onLoadGitDiff({ runId: null })}>
              Load full workspace diff
            </button>
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
          <TabNav
            ariaLabel="Publish artifact filters"
            className="artifact-filter-tabs"
            idBase="publish-artifact-filters"
            testId="publish-artifact-filters"
            value={artifactFilter}
            onChange={setArtifactFilter}
            options={PUBLISH_ARTIFACT_FILTERS}
          />
          <div {...tabPanelProps("publish-artifact-filters", artifactFilter)}>
            {visibleChangedFiles.length ? (
              <div className="publish-artifact-list compact" role="list" aria-label="changed workspace files">
                {visibleChangedFiles.slice(0, 16).map((file) => (
                  <div key={file.path} role="listitem">
                    <button
                      type="button"
                      className={`publish-artifact-row${selectedDiffFilePath === file.path ? " is-selected" : ""}`}
                      onClick={() => {
                        setPublishView("diff");
                        onLoadGitDiff({ path: file.path });
                      }}
                      aria-pressed={selectedDiffFilePath === file.path}
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
            {filteredPublishArtifacts.length === 0 ? (
              <p className="empty-state" data-testid="publish-artifact-filter-empty">
                {publishArtifacts.length === 0
                  ? "No selected-run artifacts are available for publication preview."
                  : `No ${publishArtifactFilterLabel(artifactFilter).toLowerCase()} artifact refs are available in this publication view.`}
              </p>
            ) : (
              <div className="publish-artifact-list" role="list" aria-label="publish artifact preview list">
                {filteredPublishArtifacts.slice(0, 12).map((artifact) => (
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
            )}
          </div>
        </section>

        <section className="publish-preview-panel" id="publish-preview-panel" data-testid="publish-preview-panel">
          <TabNav
            ariaLabel="Publish preview tabs"
            className="publish-preview-tabs"
            idBase="publish-preview-tabs"
            testId="publish-preview-tabs"
            value={publishView}
            onChange={setPublishView}
            options={(["preview", "diff", "evidence", "changelog"] as const).map((view) => ({ id: view, label: publishTabLabel(view) }))}
          />
          <div className="publish-tab-panel" data-testid="publish-tab-panel" {...tabPanelProps("publish-preview-tabs", publishView)}>
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
                <h2>{diffScopeTitle}</h2>
                <p className="hint">{diffScopeHint}</p>
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
          <section className="publish-gate-panel" id="publish-gate-panel" data-testid="publish-gate-panel">
            <div className="panel-subheader">
              <div>
                <h2>Publish gate</h2>
                <p className="hint">Checks gate Git commit and proposal branch actions; Git commands stay explicit operator actions.</p>
              </div>
              <StatusBadge tone={publishGateTone}>{publishGateLabel}</StatusBadge>
            </div>
            <PublishGateSection testId="publish-hard-blockers" title="Hard blockers" emptyLabel="No hard blockers. Git actions are allowed." items={blockingGateItems} />
            <PublishGateSection testId="publish-review-warnings" title="Review warnings" emptyLabel="No review warnings." items={warningGateItems} />
            <PublishGateSection testId="publish-open-questions" title="Open questions" emptyLabel="No open questions loaded." items={openQuestionGateItems} />
            <PublishGateSection testId="publish-ready-checks" title="Ready checks" emptyLabel="No ready checks yet." items={readyGateItems} />
          </section>

          <section className="publish-commit-plan" id="publish-commit-plan" data-testid="publish-commit-plan">
            <div className="panel-subheader">
              <div>
                <h2>Commit plan</h2>
                <p className="hint">Prepared commit/proposal branch actions use the existing Git API.</p>
              </div>
              <StatusBadge tone={gitError ? "error" : gitStatus ? "ok" : "info"}>{gitError ? "failed" : gitStatus ? "updated" : "pending"}</StatusBadge>
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
            {gitError ? (
              <div className="publish-git-recovery" data-testid="publish-git-action-recovery" role="alert">
                <strong>Git action failed</strong>
                <span>{gitError}</span>
                <p>Workspace Git state was not changed by this action. Review the message or branch name, check local Git permissions/status, then retry.</p>
              </div>
            ) : null}
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

function publishArtifactMatchesFilter(artifact: Artifact, filter: PublishArtifactFilter, changedPathSet: Set<string>): boolean {
  const path = artifact.path;
  if (filter === "all") {
    return true;
  }
  if (filter === "changed") {
    return changedPathSet.has(path);
  }
  if (filter === "reports") {
    return (
      (artifact.kind === "report" || artifact.kind === "agent-output" || path.startsWith("reports/")) &&
      !path.startsWith("reports/diagrams/") &&
      !path.startsWith("reports/changelog/") &&
      !path.startsWith("reports/taskruns/")
    );
  }
  if (filter === "proposals") {
    return artifact.kind === "proposal" || artifact.kind === "changelog" || path.startsWith("proposals/") || path.startsWith("reports/changelog/");
  }
  if (filter === "diagrams") {
    return artifact.kind === "diagram" || path.startsWith("reports/diagrams/");
  }
  if (filter === "taskruns") {
    return artifact.kind === "taskrun" || path.startsWith("reports/taskruns/");
  }
  return true;
}

function comparePublishArtifactPriority(left: Artifact, right: Artifact): number {
  const priority = (artifact: Artifact): number => {
    switch (artifact.path) {
      case "reports/as-is/overview.md":
        return 0;
      case "reports/findings/findings.md":
        return 1;
      case "reports/coverage/summary.md":
        return 2;
      case "reports/coverage/open-questions.md":
        return 3;
      default:
        return artifact.path.startsWith("proposals/") ? 4 : 5;
    }
  };
  const priorityDelta = priority(left) - priority(right);
  return priorityDelta !== 0 ? priorityDelta : left.path.localeCompare(right.path);
}

function publishArtifactFilterLabel(filter: PublishArtifactFilter): string {
  return PUBLISH_ARTIFACT_FILTERS.find((option) => option.id === filter)?.label ?? "Selected";
}

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

function gitDiffScopeTitle(gitDiff: GitDiffResponse | null): string {
  return gitDiff?.run_id ? "Selected run Git diff" : "Full workspace Git diff";
}

function gitDiffScopeHint(gitDiff: GitDiffResponse | null): string {
  if (gitDiff?.run_id) {
    return "Run-scoped view shows changed workspace artifacts linked to the selected run. Load the full workspace diff to inspect all uncommitted files.";
  }
  return "Full workspace view shows all uncommitted files in the local architecture workspace repository.";
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
