import { useCallback, useEffect, useRef, useState, type ComponentProps, type ReactNode, type RefObject } from "react";

import { BaselineEditorsPanel } from "./BaselineEditorsPanel";
import { AnalysisLiveDiagnosticsPanel, type AnalysisLiveDiagnostics, type AnalysisLiveMetric, type AnalysisLiveTrace } from "./AnalysisDiagnosticsPanel";
import { ModalDialog } from "./ModalDialog";
import { RuntimeProfileSettingsPanel } from "./RuntimeProfileSettingsPanel";
import { RunStatusPanel } from "./RunStatusPanel";
import { RecoveryPanel, RunResultPanel, StructuredRunProgress, TargetedRerunPanel } from "./RunOutcome";
import { TabNav, tabPanelProps } from "./TabNav";
import { RunHistoryTable } from "./RunHistoryTable";
import { PendingPermissionsTable } from "./PendingPermissionsTable";
import { ReviewDomainMap } from "./ReviewDomainMap";
import { ArtifactPathButton, StatusBadge } from "./ConsolePrimitives";
import {
  boolField,
  fieldString,
  firstNonEmpty,
  firstNumericField,
  formatCompactCount,
  lastString,
  maxDefined,
  numericField,
  rawOutputRefsFromEntry,
} from "../features/analysis/analysisUtils";
import { isRunCanceled, isRunReconciledAfterRestart, isRunnerUnavailable, runOutcomeLabel, runOutcomeTone } from "../lib/runState";
import {
  groupArtifactsByFolder,
} from "../lib/artifactFilters";
import {
  deriveProposalArtifactType,
  deriveProposalReviewModel,
  proposalTabLabel,
} from "../features/proposals/proposalUtils";
import { ProposalPackageRecoveryPanel } from "../features/proposals/ProposalPackageRecoveryPanel";
import {
  buildReviewQueue,
  countMarkdownItems,
  deriveReviewTrustStatus,
  findLastSuccessfulRun,
  reviewRouteDescription,
} from "../features/review/reviewUtils";
import { deriveReviewDomainMap } from "../features/review/reviewDomainMapUtils";
import { buildCharterBaselineRecovery, splitSummaryList, type CharterRecoveryIssue } from "../features/setup/charterUtils";
import { failureEvidenceSummary, failureRecoveryGuidance } from "../features/analysis/analysisRecoveryUtils";
export { ReadinessStagePanel, SourceStagePanel } from "../features/setup/SetupStagePanels";
export { PublishStagePanel } from "../features/publish/PublishStagePanel";
import { GitDiffView } from "./GitDiffView";
export { AskStagePanel } from "../features/qa/AskStagePanel";
import { ReviewEvidenceWorkbench } from "../features/review/ReviewEvidenceWorkbench";
import {
  buildAnalysisShardRows,
  buildAnalysisStepTimeline,
  formatArtifactPairRefs,
  type AnalysisShardRow,
  type AnalysisStep,
} from "../features/analysis/analysisViewModels";
import { AnalysisStepReview } from "../features/analysis/AnalysisStepReview";
import {
  artifactHandoffStalled,
  formatShardMetric,
  parseShardCounters,
  stageFromMessage,
  summarizeProviderStream,
} from "../features/analysis/providerStreamUtils";
import type {
  Artifact,
  Diagnostic,
  EditableArtifactOption,
  GitDiffResponse,
  RuntimePermissionRequest,
  RunListItem,
  RunCoordination,
  RunLogEntry,
  RunReviewSummaryResponse,
  RunStatusResponse,
} from "../lib/appContracts";
import type { LoadGitDiffOptions } from "../lib/gitDiffApi";


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

export type AnalysisStageProps = {
  detailMode?: boolean;
  readOnly?: boolean;
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
  readOnly = false,
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
  const hasPromotedArchitecture = runList.some((run) => (run.pipeline === "init" || run.pipeline === "refresh") && run.status === "succeeded" && run.authoritative_index === true);

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
          <h1>{detailMode ? "Analysis" : hasPromotedArchitecture ? "Refresh architecture knowledge" : "Create your architecture workspace"}</h1>
          <p className="hint">{detailMode ? "Review the outcome, supporting evidence and recovery path for the selected run." : hasPromotedArchitecture ? "Refresh the current architecture or selectively rerun a weak stage when sources change." : "The first run turns connected repositories into a validated set of documents, diagrams, entities and evidence."}</p>
        </div>
        <StatusBadge tone={selectedRunIsActive ? "warn" : runOutcomeTone(runStatus)}>{runOutcomeLabel(runStatus)}</StatusBadge>
      </div>

      {!detailMode && !readOnly ? <section className="analysis-launcher-hero" aria-labelledby="analysis-launcher-title">
        <div className="analysis-launcher-copy"><p className="eyebrow">{hasPromotedArchitecture ? "Current snapshot · refresh available" : "Setup complete · ready to run"}</p><h2 id="analysis-launcher-title">{hasPromotedArchitecture ? "Keep the shared architecture current" : "Build the first trustworthy architecture snapshot"}</h2><p>{hasPromotedArchitecture ? "A refresh compares new evidence with the promoted baseline and stops for review before anything is published." : "ProvenArch inspects repository evidence, generates the model and documentation, validates citations, then stops for review before anything is published."}</p></div>
        <div className="analysis-launcher-actions"><button type="button" className="ui-button tone-primary" onClick={() => onRunPipeline(hasPromotedArchitecture ? "refresh" : "init")} disabled={busy || Boolean(coordination.active_run_id)}>{hasPromotedArchitecture ? "Refresh analysis" : "Run initial analysis"}</button><button type="button" className="ui-button" onClick={() => onOpenArchitecture()}>Review current architecture</button></div>
      </section> : null}

      {readOnly ? <p className="status info" data-testid="legacy-read-only-notice">Legacy run evidence is read-only. Starting, retrying, queueing and canceling runs are unavailable.</p> : <div className="actions">
        <button type="button" onClick={() => onRunPipeline("init")} disabled={busy || Boolean(coordination.active_run_id)} data-testid="run-init-btn">
          Run initial analysis
        </button>
        <button type="button" onClick={() => onRunPipeline("refresh")} disabled={busy || Boolean(coordination.active_run_id)} data-testid="run-refresh-btn">
          Refresh architecture
        </button>
        {coordination.active_run_id ? (
          <button type="button" onClick={() => setQueueConfirmationOpen(true)} disabled={busy} data-testid="run-queue-refresh-btn">
            Queue refresh after current run
          </button>
        ) : null}
        <button type="button" onClick={onCancelSelectedRun} disabled={busy || cancelBusy || !runId || !selectedRunIsActive} data-testid="run-cancel-btn">
          Cancel selected run
        </button>
      </div>}
      {coordination.active_run_id ? <p className="hint" data-testid="run-active-start-reason">Ordinary start is unavailable while <code>{coordination.active_run_id}</code> is active.</p> : null}
      {coordination.pending ? (
        <section className="subsection" data-testid="pending-run-summary">
          <h2>Pending refresh</h2>
          <p><code>{coordination.pending.run_id}</code> · {coordination.pending.pipeline}. A newly queued refresh replaces this pending run.</p>
          {!readOnly ? <button type="button" onClick={() => onCancelRun(coordination.pending!.run_id)} disabled={busy || cancelBusy}>Cancel pending refresh</button> : null}
        </section>
      ) : null}
      {runActionStatus ? <p className="status warn">{runActionStatus}</p> : null}
      {detailMode && runStatus?.pipeline === "refresh" ? <RefreshExecutionSummary runStatus={runStatus} /> : null}

      {!readOnly ? <ModalDialog
        open={queueConfirmationOpen}
        title={coordination.pending ? "Replace pending refresh" : "Queue refresh"}
        description={coordination.pending
          ? `Active run ${coordination.active_run_id}; pending ${coordination.pending.run_id} will be canceled as run_superseded.`
          : `Refresh will start after active run ${coordination.active_run_id}.`}
        confirmLabel={coordination.pending ? "Replace pending refresh" : "Queue refresh after current run"}
        busy={busy}
        onCancel={() => setQueueConfirmationOpen(false)}
        onConfirm={() => { setQueueConfirmationOpen(false); onRunPipeline("refresh", "queue"); }}
      /> : null}

	  {detailMode && runStatus?.status === "succeeded" && !runReviewSummary?.result ? <AnalysisOutcomeFallback runStatus={runStatus} artifacts={artifacts} onOpenArchitecture={onOpenArchitecture} /> : null}
	  <StructuredRunProgress runStatus={runStatus} review={runReviewSummary} onReviewDetails={handleReviewBlocker} />
      {detailMode ? (
        <div className="run-studio-body">
		  <RunResultPanel review={runReviewSummary} onExploreArchitecture={onOpenArchitecture} />
		  <TargetedRerunPanel runStatus={runStatus} review={runReviewSummary} busy={busy} readOnly={readOnly} onRetryStarted={onSelectRun} />
		  <RecoveryPanel runStatus={runStatus} review={runReviewSummary} busy={busy} readOnly={readOnly} onRetryStarted={onSelectRun} onReviewDetails={handleReviewBlocker} />
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

function AnalysisOutcomeFallback({ runStatus, artifacts, onOpenArchitecture }: { runStatus: RunStatusResponse; artifacts: Artifact[]; onOpenArchitecture: () => void }) {
  const documents = artifacts.filter((artifact) => artifact.path.endsWith(".md")).length;
  const diagrams = artifacts.filter((artifact) => artifact.path.endsWith(".mmd")).length;
  return <section className="analysis-outcome-fallback" data-testid="analysis-outcome-fallback"><div className="analysis-outcome-copy"><p className="eyebrow">{runStatus.pipeline === "refresh" ? "Refresh analysis outcome" : "Initial analysis outcome"}</p><div className="analysis-outcome-badges"><StatusBadge tone="ok">succeeded</StatusBadge><StatusBadge tone="warn">review required</StatusBadge></div><h2>{runStatus.pipeline === "refresh" ? "Architecture updated and ready for review" : "Architecture baseline ready for review"}</h2><p>All pipeline stages completed. Review the promoted documents and evidence before publication.</p></div><dl className="analysis-outcome-metrics"><div><dt>Documents</dt><dd>{documents}</dd></div><div><dt>Diagrams</dt><dd>{diagrams}</dd></div><div><dt>Run</dt><dd>{runStatus.run_id}</dd></div><div><dt>Pipeline</dt><dd>{runStatus.pipeline}</dd></div></dl><div className="analysis-outcome-action"><strong>Recommended next action</strong><span>Review architecture update</span><button type="button" onClick={onOpenArchitecture}>Explore architecture</button></div></section>;
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
    openQuestionCount,
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
            <p className="eyebrow">Review workbench</p>
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
        {routeView === "overview" && reviewSummary?.review ? <ChangesReviewBrief review={reviewSummary.review} /> : null}
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

function ChangesReviewBrief({ review }: { review: NonNullable<RunReviewSummaryResponse["review"]> }) {
  const comparison = review.semantic_changes;
  const changeCount = Object.values(comparison.categories).reduce((total, category) => total + category.added.length + category.changed.length + category.removed.length, 0);
  const documentCount = review.summary.documents_added + review.summary.documents_changed + review.summary.documents_removed;
  return <section className="changes-review-brief" data-testid="semantic-changes" aria-label="Selected change set summary">
    <div className="changes-review-brief-heading"><div><p className="eyebrow">{review.review_kind === "initial" ? "Run-pinned initial review" : "Run-pinned architecture update"}</p><h2>{review.review_kind === "initial" ? "Initial architecture review" : "Architecture update review"}</h2><p className="hint">Validator-approved promotion is complete. This workbench is the human Git review surface.</p></div><span className={`status ${comparison.available ? "ok" : "info"}`}>{comparison.available ? "Semantic delta" : "Initial snapshot"}</span></div>
    <div className="changes-review-brief-metrics" data-testid="run-pinned-review-summary"><div><strong>{changeCount}</strong><span>semantic changes</span></div><div><strong>{documentCount}</strong><span>document changes</span></div><div><strong>{review.summary.findings}</strong><span>findings</span></div><div><strong>{review.summary.gaps}</strong><span>coverage gaps</span></div><div><strong>{review.runtime.mode || "Unknown"}</strong><span>runner mode</span></div></div>
    <div className="changes-review-brief-grid"><section><h3>Change sets</h3>{comparison.available ? <ul>{(["entities", "edges", "findings", "gaps"] as const).map((category) => <li key={category}><strong>{category}</strong><span>{comparison.categories[category].added.length} added · {comparison.categories[category].changed.length} changed · {comparison.categories[category].removed.length} removed</span></li>)}</ul> : <p className="hint">This first snapshot establishes the baseline for future refresh comparisons.</p>}</section><section><h3>Review notes</h3>{review.findings.length + review.questions.length + review.gaps.length === 0 ? <p className="status ok">No open review notes.</p> : <ul>{review.findings.slice(0, 3).map((finding) => <li key={finding.id}><strong>{finding.title}</strong><span>{finding.severity}</span></li>)}{review.questions.slice(0, 3).map((question) => <li key={question.id}><strong>{question.text}</strong><span>{question.priority || "open"}</span></li>)}{review.gaps.slice(0, 3).map((gap) => <li key={gap}><strong>{gap}</strong><span>coverage gap</span></li>)}</ul>}</section></div>
  </section>;
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
  const proposalReview = deriveProposalReviewModel({ artifacts, openQuestionCount: countMarkdownItems(openQuestions) });
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

export function RuntimeSettingsStagePanel(props: ComponentProps<typeof RuntimeProfileSettingsPanel>) {
  return <RuntimeProfileSettingsPanel {...props} />;
}
