import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { AppShell } from "./components/AppShell";
import { BaselineGitPanel } from "./components/BaselineGitPanel";
import { OnboardingShell } from "./components/OnboardingShell";
import { RuntimeProfileSettingsPanel } from "./components/RuntimeProfileSettingsPanel";
import {
  AnalysisStagePanel,
  AskStagePanel,
  CharterStagePanel,
  ProposalsStagePanel,
  PublishStagePanel,
  ReadinessStagePanel,
  ReviewStagePanel,
  SourceStagePanel,
} from "./components/StagePanels";
import { WizardContractPanel } from "./components/WizardContractPanel";
import {
  runtimeExecutionLabels,
  runtimePermissionLabels,
  runtimeStepProviderLabels,
  runtimeStepProviderOrder,
  runtimeTimeoutKeys,
  runtimeTimeoutLabels,
  type GuidedRepo,
  type OnboardingStatusResponse,
  type RuntimeExecutionKey,
  type RuntimePermissionKey,
  type RuntimePermissionRequest,
  type RuntimeTimeoutKey,
  type SystemVersionResponse,
  type WorkspaceHealthResponse,
} from "./lib/appContracts";
import type { InspectorItem, NextAction, StageId } from "./lib/consoleTypes";
import type { LoadGitDiffOptions } from "./lib/gitDiffApi";
import { buildStageOptions } from "./lib/stageModel";
import { runtimeDisplayLabel } from "./lib/runtimeDisplay";
import { isRunCanceled, isRunReconciledAfterRestart, isRunnerUnavailable } from "./lib/runState";
import { useRunExplorer } from "./hooks/useRunExplorer";
import { useRuntimeSettings } from "./hooks/useRuntimeSettings";
import { useWorkspaceSetup } from "./hooks/useWorkspaceSetup";
import { forgetOnboardingRecentWorkspace, loadOnboardingStatus, selectOnboardingRuntime, selectOnboardingWorkspace } from "./lib/onboardingApi";
import { loadSystemDoctor, loadSystemVersion } from "./lib/systemApi";
import { loadWorkspaceHealthAPI } from "./lib/workspaceApi";

export default function App() {
  const [activeStage, setActiveStage] = useState<StageId>("source");
  const userSelectedStageRef = useRef(false);
  const autoOpenedStageRef = useRef(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [setupRuntime, setSetupRuntime] = useState("fake");
  const [setupRuntimeProvider, setSetupRuntimeProvider] = useState("claude-code");
  const [setupDoctorResult, setSetupDoctorResult] = useState<Awaited<ReturnType<typeof loadSystemDoctor>> | null>(null);
  const [systemVersion, setSystemVersion] = useState<SystemVersionResponse>({
    version: "dev",
    commit: "none",
    built: "unknown",
    ui_bundle: "embedded",
  });
  const [setupDoctorStatus, setSetupDoctorStatus] = useState("");
  const [firstRunStatus, setFirstRunStatus] = useState("");
  const [onboardingStatus, setOnboardingStatus] = useState<OnboardingStatusResponse | null>(null);
  const [onboardingWorkspacePath, setOnboardingWorkspacePath] = useState("");
  const [onboardingCreateWorkspace, setOnboardingCreateWorkspace] = useState(true);
  const [consoleReady, setConsoleReady] = useState(false);
  const [analysisFocusSignal, setAnalysisFocusSignal] = useState(0);
  const [askPrimaryActionSignal, setAskPrimaryActionSignal] = useState(0);
  const [workspaceHealthReport, setWorkspaceHealthReport] = useState<WorkspaceHealthResponse | null>(null);
  const [workspaceHealthStatus, setWorkspaceHealthStatus] = useState<"idle" | "loading" | "loaded" | "error">("idle");
  const [workspaceHealthError, setWorkspaceHealthError] = useState("");

  const runtimeSettings = useRuntimeSettings({
    setBusy,
    setError,
  });
  const runExplorer = useRunExplorer({
    setBusy,
    setError,
  });
  const workspaceSetup = useWorkspaceSetup({
    setBusy,
    setError,
  });

  const {
    runtimeTimeoutPersisted,
    runtimeTimeoutEffective,
    runtimeTimeoutSource,
    runtimeTimeoutDraft,
    runtimeTimeoutStatus,
    runtimeExecutionPersisted,
    runtimeExecutionEffective,
    runtimeExecutionSource,
    runtimeExecutionDraft,
    runtimeExecutionStatus,
    runtimePermissionPersisted,
    runtimePermissionEffective,
    runtimePermissionSource,
    runtimePermissionDraft,
    runtimePermissionStatus,
    runtimeStepProviderPersisted,
    runtimeStepProviderEffective,
    runtimeStepProviderSource,
    loadRuntimeTimeouts,
    loadRuntimeExecution,
    loadRuntimePermissions,
    loadRuntimeProfile,
    updateRuntimeTimeoutDraft,
    updateRuntimeExecutionDraft,
    updateRuntimePermissionDraft,
    handleSaveRuntimeTimeouts,
    handleResetRuntimeTimeouts,
    handleSaveRuntimeExecution,
    handleResetRuntimeExecution,
    handleSaveRuntimePermissions,
    handleResetRuntimePermissions,
  } = runtimeSettings;

  const {
    runId,
    runStatus,
    runList,
    selectedArtifact,
    selectedArtifactContent,
    runLogsStatus,
    runLogsViewMode,
    setRunLogsViewMode,
    runLogsMode,
    setRunLogsMode,
    runActionStatus,
    cancelBusy,
    coverageSummary,
    openQuestions,
    runCounters,
    runLogTaskrunPaths,
    runLogs,
    filteredRunLogs,
    diagramArtifacts,
    nonDiagramArtifacts,
    selectedArtifactIsMermaid,
    selectedRunWarnings,
    selectedRunIsActive,
    runLogsRendered,
    runReviewSummary,
    runReviewStatus,
    gitDiff,
    gitDiffStatus,
    bootstrapRuns,
    loadGitDiff,
    handleRunPipeline,
    handleSelectRun,
    handleCancelSelectedRun,
    handleOpenArtifact,
    handleCopyRunLogs,
    handleDownloadRunLogs,
  } = runExplorer;

  const {
    validateResult,
    validationDiagnosticsByRepo,
    manifestContent,
    manifestStatus,
    baselineEditorArtifacts,
    baselineBundleWarnings,
    workspaceRootPath,
    selectedEditorPath,
    selectedEditorContent,
    selectedEditorLoadedPath,
    editorStatus,
    guidedRepos,
    guidedDocsImportsPath,
    wizardProjectName,
    wizardScope,
    wizardNfr,
    wizardRules,
    wizardStatus,
    wizardContractLoaded,
    gitMessage,
    proposalBranch,
    gitStatus,
    gitError,
    bootstrapWorkspaceSetup,
    setManifestContent,
    setGuidedDocsImportsPath,
    setWizardProjectName,
    setWizardScope,
    setWizardNfr,
    setWizardRules,
    setSelectedEditorContent,
    setGitMessage,
    setProposalBranch,
    updateGuidedRepo,
    loadSelectedEditorContent,
    loadWizardContract,
    handleAddGuidedRepo,
    handleRemoveGuidedRepo,
    handleApplyGuidedWorkspaceSetup,
    handleSaveGuidedWorkspaceSetup,
    handleSaveManifest,
    handleValidateWorkspace,
    handleSaveStep0WizardContract,
    handleEditorSelectionChange,
    handleSaveSelectedEditorArtifact,
    handleGitCommit,
    handleCreateProposalBranch,
  } = workspaceSetup;

  useEffect(() => {
    void bootstrapApp();
  }, []);

  useEffect(() => {
    if (activeStage !== "charter") {
      return;
    }
    const wizardContractPathAvailable = baselineEditorArtifacts.some((artifact) => artifact.path === "charter/wizard/step0-contract.json");
    if (!wizardContractLoaded && wizardContractPathAvailable) {
      void loadWizardContract();
    }
    if (selectedEditorPath && selectedEditorLoadedPath !== selectedEditorPath) {
      void loadSelectedEditorContent(selectedEditorPath);
    }
  }, [activeStage, baselineEditorArtifacts, loadSelectedEditorContent, loadWizardContract, selectedEditorLoadedPath, selectedEditorPath, wizardContractLoaded]);

  async function bootstrapApp() {
    setError(null);
    try {
      await bootstrapSystemVersion();
      const status = await loadOnboardingStatus();
      syncOnboardingStatus(status);
      if (!status.can_enter_console) {
        setConsoleReady(false);
        return;
      }
      await bootstrapConsoleData({ validateWorkspace: true });
      setConsoleReady(true);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "console data refresh failed");
    }
  }

  async function bootstrapSystemVersion() {
    try {
      setSystemVersion(await loadSystemVersion());
    } catch {
      setSystemVersion({
        version: "dev",
        commit: "none",
        built: "unknown",
        ui_bundle: "embedded",
      });
    }
  }

  async function bootstrapConsoleData(options: { validateWorkspace?: boolean } = {}) {
    await bootstrapRuns();
    await bootstrapWorkspaceSetup();
    await loadRuntimeTimeouts();
    await loadRuntimeExecution();
    await loadRuntimePermissions();
    await loadRuntimeProfile();
    if (options.validateWorkspace) {
      await handleValidateWorkspace();
    }
    void refreshWorkspaceHealth();
  }

  function syncOnboardingStatus(status: OnboardingStatusResponse) {
    setOnboardingStatus(status);
    if (status.workspace && !onboardingWorkspacePath.trim()) {
      setOnboardingWorkspacePath(status.workspace);
    }
    if (status.runtime.runtime) {
      setSetupRuntime(status.runtime.runtime);
    }
    if (status.runtime.runtime_provider) {
      setSetupRuntimeProvider(status.runtime.runtime_provider);
    }
  }

  async function refreshOnboardingStatus() {
    const status = await loadOnboardingStatus();
    syncOnboardingStatus(status);
    return status;
  }

  async function handleOnboardingWorkspaceSelect(path = onboardingWorkspacePath, create = onboardingCreateWorkspace) {
    setBusy(true);
    setError(null);
    try {
      const status = await selectOnboardingWorkspace(path, create);
      setOnboardingWorkspacePath(status.workspace || path);
      setOnboardingCreateWorkspace(create);
      syncOnboardingStatus(status);
      if (status.workspace_ready && status.manifest_present) {
        await bootstrapWorkspaceSetup();
        await handleValidateWorkspace();
      }
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "workspace selection failed");
    } finally {
      setBusy(false);
    }
  }

  async function handleOpenRecentWorkspace(path: string) {
    setOnboardingWorkspacePath(path);
    setOnboardingCreateWorkspace(false);
    await handleOnboardingWorkspaceSelect(path, false);
  }

  async function handleForgetRecentWorkspace(path: string) {
    setBusy(true);
    setError(null);
    try {
      const status = await forgetOnboardingRecentWorkspace(path);
      syncOnboardingStatus(status);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to forget recent workspace");
    } finally {
      setBusy(false);
    }
  }

  async function handleOnboardingSaveSources() {
    await handleSetupSaveGuidedWorkspaceSetup();
    const status = await refreshOnboardingStatus();
    if (status.can_enter_console) {
      await bootstrapConsoleData({ validateWorkspace: false });
    }
  }

  async function handleOnboardingSaveRuntime() {
    setBusy(true);
    setError(null);
    try {
      const status = await selectOnboardingRuntime(setupRuntime, setupRuntimeProvider);
      syncOnboardingStatus(status);
      const validation = status.can_enter_console && validateResult?.ok !== true ? await handleValidateWorkspace() : validateResult;
      if (status.can_enter_console) {
        await bootstrapConsoleData({ validateWorkspace: validation?.ok !== true });
      }
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "runner selection failed");
    } finally {
      setBusy(false);
    }
  }

  async function handleOnboardingEnterConsole(): Promise<boolean> {
    const status = await refreshOnboardingStatus();
    const validation = validateResult?.ok === true ? validateResult : await handleValidateWorkspace();
    if (!status.can_enter_console || validation?.ok !== true) {
      setError("Validate sources and select a runner before opening the console.");
      return false;
    }
    await bootstrapConsoleData({ validateWorkspace: false });
    setConsoleReady(true);
    return true;
  }

  async function handleOnboardingRunFirstAnalysis() {
    const entered = await handleOnboardingEnterConsole();
    if (entered) {
      await handleSetupFirstRun("analysis");
    }
  }

  async function handleSetupDoctorCheck() {
    setBusy(true);
    setError(null);
    setSetupDoctorStatus("");
    try {
      const firstRepo = guidedRepos[0];
      const repoPayload =
        firstRepo?.mode === "path"
          ? { repo_path: firstRepo.path }
          : firstRepo?.mode === "git_url"
            ? { repo_git_url: firstRepo.git_url }
            : {};
      const result = await loadSystemDoctor({
        runtime: setupRuntime,
        runtime_provider: setupRuntimeProvider,
        ...repoPayload,
      });
      setSetupDoctorResult(result);
      setSetupDoctorStatus(result.ok ? "Local readiness passed." : "Local readiness needs attention.");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "local readiness check failed");
    } finally {
      setBusy(false);
    }
  }

  async function refreshWorkspaceHealth() {
    setWorkspaceHealthStatus("loading");
    setWorkspaceHealthError("");
    try {
      const report = await loadWorkspaceHealthAPI();
      setWorkspaceHealthReport(report);
      setWorkspaceHealthStatus("loaded");
      return report;
    } catch (requestError) {
      setWorkspaceHealthReport(null);
      setWorkspaceHealthStatus("error");
      setWorkspaceHealthError(requestError instanceof Error ? requestError.message : "workspace health scan failed");
      return null;
    }
  }

  async function handleValidateWorkspaceWithHealth() {
    const validation = await handleValidateWorkspace();
    await refreshWorkspaceHealth();
    return validation;
  }

  function clearFirstRunReadiness() {
    setSetupDoctorResult(null);
    setSetupDoctorStatus("");
    setFirstRunStatus("");
  }

  function handleSetupRepoChange(id: string, patch: Partial<GuidedRepo>) {
    updateGuidedRepo(id, patch);
    clearFirstRunReadiness();
  }

  function handleSetupAddRepo() {
    handleAddGuidedRepo();
    clearFirstRunReadiness();
  }

  function handleSetupRemoveRepo(id: string) {
    handleRemoveGuidedRepo(id);
    clearFirstRunReadiness();
  }

  function handleSetupDocsImportsPathChange(value: string) {
    setGuidedDocsImportsPath(value);
    clearFirstRunReadiness();
  }

  function handleSetupManifestChange(value: string) {
    setManifestContent(value);
    clearFirstRunReadiness();
  }

  function handleSetupApplyGuidedWorkspaceSetup() {
    handleApplyGuidedWorkspaceSetup();
    clearFirstRunReadiness();
  }

  async function handleSetupSaveGuidedWorkspaceSetup() {
    clearFirstRunReadiness();
    await handleSaveGuidedWorkspaceSetup();
  }

  function handleSetupRuntimeChange(value: string) {
    setSetupRuntime(value);
    clearFirstRunReadiness();
  }

  function handleSetupRuntimeProviderChange(value: string) {
    setSetupRuntimeProvider(value);
    clearFirstRunReadiness();
  }

  async function handleSetupFirstRun(nextStage: StageId = "analysis") {
    setFirstRunStatus("");
    const started = await handleRunPipeline("init");
    if (started) {
      setFirstRunStatus("First analysis started. Results will update as the run finishes.");
      setActiveStage(nextStage);
    }
  }

  async function handleOpenArtifactAndReview(path: string) {
    await handleOpenArtifact(path);
    if (path.startsWith("proposals/") || path.startsWith("reports/changelog/")) {
      setActiveStage("proposals");
      return;
    }
    setActiveStage("review");
  }

  function handleStageChange(stage: StageId) {
    userSelectedStageRef.current = true;
    if (stage === "review") {
      enterReviewStage();
      return;
    }
    setActiveStage(stage);
  }

  const diagnostics = useMemo(() => [...(validateResult?.errors ?? []), ...(validateResult?.warnings ?? [])], [validateResult]);
  const validationErrors = useMemo(() => diagnostics.filter((diagnostic) => diagnostic.level === "error"), [diagnostics]);
  const doctorFailures = useMemo(() => setupDoctorResult?.checks.filter((check) => check.status === "fail") ?? [], [setupDoctorResult]);
  const artifactCount = nonDiagramArtifacts.length + diagramArtifacts.length;
  const proposalArtifacts = useMemo(
    () => nonDiagramArtifacts.filter((artifact) => artifact.path.startsWith("proposals/") || artifact.path.startsWith("reports/changelog/")),
    [nonDiagramArtifacts],
  );
  const preferredReviewArtifactPath = useMemo(() => {
    const preferredArtifact =
      nonDiagramArtifacts.find((artifact) => artifact.path === "reports/as-is/overview.md") ??
      nonDiagramArtifacts.find((artifact) => artifact.path.startsWith("reports/") && !artifact.path.startsWith("reports/changelog/")) ??
      nonDiagramArtifacts.find((artifact) => artifact.path.startsWith("model/")) ??
      diagramArtifacts[0];
    return preferredArtifact?.path ?? "";
  }, [diagramArtifacts, nonDiagramArtifacts]);

  function enterReviewStage() {
    setActiveStage("review");
    if (preferredReviewArtifactPath && isProposalReviewArtifact(selectedArtifact) && selectedArtifact !== preferredReviewArtifactPath) {
      void handleOpenArtifact(preferredReviewArtifactPath);
    }
  }

  useEffect(() => {
    if (userSelectedStageRef.current || autoOpenedStageRef.current) {
      return;
    }
    if (selectedRunIsActive) {
      autoOpenedStageRef.current = true;
      setActiveStage("analysis");
      return;
    }
    if (artifactCount > 0) {
      autoOpenedStageRef.current = true;
      setActiveStage("review");
    }
  }, [artifactCount, selectedRunIsActive]);

  useEffect(() => {
    if (!consoleReady || onboardingStatus?.can_enter_console !== true) {
      return;
    }
    const path = activeStage === "publish" ? undefined : selectedArtifact || undefined;
    void loadGitDiff({ runId, path });
  }, [activeStage, consoleReady, loadGitDiff, onboardingStatus?.can_enter_console, runId, selectedArtifact]);

  const handleLoadGitDiff = useCallback(
    (options: LoadGitDiffOptions) => {
      void loadGitDiff({ runId, ...options });
    },
    [loadGitDiff, runId],
  );

  const stages = useMemo(
    () =>
      buildStageOptions({
        activeStage,
        hasManifest: Boolean(manifestContent.trim()),
        readinessBlocked: validationErrors.length > 0 || doctorFailures.length > 0,
        readinessDone: Boolean(validateResult?.ok && setupDoctorResult?.ok),
        charterStarted: Boolean(wizardProjectName.trim() || selectedEditorPath),
        analysisBlocked: runStatus?.status === "failed" || (runStatus?.pending_permissions?.length ?? 0) > 0,
        selectedRunIsActive,
        runSucceeded: runStatus?.status === "succeeded",
        artifactCount,
        proposalArtifactCount: proposalArtifacts.length,
        runningRunCount: runCounters.running,
        hasGitStatus: Boolean(gitStatus),
      }),
    [
      activeStage,
      artifactCount,
      doctorFailures.length,
      gitStatus,
      manifestContent,
      proposalArtifacts.length,
      runCounters.running,
      runStatus,
      selectedEditorPath,
      selectedRunIsActive,
      setupDoctorResult,
      validateResult,
      validationErrors.length,
      wizardProjectName,
    ],
  );

  const blockers = useMemo<InspectorItem[]>(() => {
    const items: InspectorItem[] = [];
    for (const diagnostic of validationErrors) {
      items.push({
        severity: "error",
        label: diagnostic.code,
        detail: diagnostic.suggestion ? `${diagnostic.message} Suggested fix available.` : diagnostic.message,
        path: diagnostic.path,
      });
    }
    for (const check of doctorFailures) {
      items.push({
        severity: "error",
        label: check.label,
        detail: check.suggestion ? `${check.message} Suggested fix available.` : check.message,
      });
    }
    for (const request of runStatus?.pending_permissions ?? []) {
      items.push({
        severity: "error",
        label: request.action ? `Permission: ${request.action}` : "Runtime permission",
        detail: formatPermissionBlockerDetail(request),
      });
    }
    if (runStatus?.error_code) {
      const issue = selectedRunIssueCopy(runStatus.error_code, runStatus.error, "inspector");
      items.push({
        severity: "error",
        label: issue.label,
        detail: issue.detail,
      });
    }
    if (openQuestions.trim()) {
      items.push({
        severity: "warn",
        label: "Open questions",
        detail: "Review coverage questions before publishing.",
        path: "reports/coverage/open-questions.md",
      });
    }
    if (activeStage === "publish" && artifactCount === 0) {
      items.push({
        severity: "error",
        label: "No publishable artifacts",
        detail: "Run Analysis before committing workspace artifacts.",
      });
    }
    return items;
  }, [activeStage, artifactCount, doctorFailures, openQuestions, runStatus, validationErrors]);

  const evidenceRefs = useMemo<InspectorItem[]>(() => {
    const refs: InspectorItem[] = [];
    const addIfPresent = (path: string, label: string) => {
      const exists = [...nonDiagramArtifacts, ...diagramArtifacts].some((artifact) => artifact.path === path) || Boolean(selectedArtifact === path);
      if (exists) {
        refs.push({ severity: "info", label, detail: path, path });
      }
    };
    addIfPresent("reports/as-is/overview.md", "As-is overview");
    addIfPresent("reports/coverage/summary.md", "Coverage summary");
    addIfPresent("reports/findings/findings.md", "Findings");
    if (diagramArtifacts[0]) {
      refs.push({ severity: "info", label: "Diagram", detail: diagramArtifacts[0].path, path: diagramArtifacts[0].path });
    }
    if (selectedArtifact && !refs.some((ref) => ref.path === selectedArtifact)) {
      refs.push({ severity: "info", label: "Selected artifact", detail: selectedArtifact, path: selectedArtifact });
    }
    return refs;
  }, [diagramArtifacts, nonDiagramArtifacts, selectedArtifact]);

  const workspaceHealth = useMemo<InspectorItem[]>(() => {
    if (workspaceHealthStatus === "loading") {
      return [{ severity: "info", label: "Workspace health", detail: "scan running" }];
    }
    if (workspaceHealthStatus === "error") {
      return [{ severity: "error", label: "Workspace health scan failed", detail: workspaceHealthError || "scan failed" }];
    }
    if (!workspaceHealthReport) {
      return [];
    }
    if (workspaceHealthReport.items.length === 0) {
      return [{ severity: "ok", label: "No health findings", detail: "Workspace health scan found no advisory issues." }];
    }
    return workspaceHealthReport.items.map((item) => ({
      severity: workspaceHealthSeverity(item.severity),
      label: item.id,
      detail: item.title,
      path: item.path,
    }));
  }, [workspaceHealthError, workspaceHealthReport, workspaceHealthStatus]);

  const runtimeLabel = runtimeDisplayLabel(setupRuntime, setupRuntimeProvider, { compact: true });

  const runtimeSafety = useMemo<InspectorItem[]>(
    () => [
      {
        severity: setupRuntime === "fake" ? "ok" : "warn",
        label: "Runtime mode",
        detail: setupRuntime === "fake" ? "fake baseline; no live provider command required" : `headless via ${setupRuntimeProvider}`,
      },
      {
        severity: runtimePermissionEffective.mode === "trusted_full_access" ? "warn" : "ok",
        label: "Permission mode",
        detail: String(runtimePermissionEffective.mode ?? "trusted_full_access"),
      },
      {
        severity: "info",
        label: "Approval channel",
        detail: String(runtimePermissionEffective.approval_channel ?? "fail_fast"),
      },
    ],
    [runtimePermissionEffective.approval_channel, runtimePermissionEffective.mode, setupRuntime, setupRuntimeProvider],
  );

  const gitPublication = useMemo<InspectorItem[]>(
    () => [
      {
        severity: gitError ? "error" : gitStatus ? "ok" : proposalArtifacts.length > 0 || artifactCount > 0 ? "info" : "warn",
        label: gitError ? "Git action failed" : gitStatus ? "Last Git action" : "Publication state",
        detail: gitError || gitStatus || (artifactCount > 0 ? "Workspace artifacts are available for review before commit." : "No generated artifacts are ready to publish yet."),
      },
      {
        severity: "info",
        label: "Commit message",
        detail: gitMessage || "not prepared",
      },
      {
        severity: "info",
        label: "Proposal branch",
        detail: proposalBranch || "not prepared",
      },
    ],
    [artifactCount, gitError, gitMessage, gitStatus, proposalArtifacts.length, proposalBranch],
  );

  const publishExternalGateItems = useMemo(
    () => [
      ...validationErrors.map((diagnostic) => ({
        label: diagnostic.code,
        detail: diagnostic.suggestion ? `${diagnostic.message} Suggested fix available.` : diagnostic.message,
        tone: "error" as const,
      })),
      ...doctorFailures.map((check) => ({
        label: check.label,
        detail: check.suggestion ? `${check.message} Suggested fix available.` : check.message,
        tone: "error" as const,
      })),
      ...(runStatus?.pending_permissions ?? []).map((request) => ({
        label: request.action || "Runtime permission",
        detail: "Runtime permission request is pending before publication.",
        tone: "error" as const,
      })),
      ...(runStatus?.error_code
        ? (() => {
            const issue = selectedRunIssueCopy(runStatus.error_code, runStatus.error, "publish");
            return [
              {
                label: issue.label,
                detail: issue.detail,
                tone: "error" as const,
              },
            ];
          })()
        : []),
    ],
    [doctorFailures, runStatus, validationErrors],
  );

  const nextAction = useMemo<NextAction>(() => deriveNextAction(activeStage, {
    validateOK: Boolean(validateResult?.ok),
    doctorOK: Boolean(setupDoctorResult?.ok),
    hasArtifacts: artifactCount > 0,
    hasProposals: proposalArtifacts.length > 0,
    hasRun: Boolean(runStatus),
    blockersCount: blockers.length,
    hardBlockersCount:
      validationErrors.length +
      doctorFailures.length +
      (runStatus?.pending_permissions?.length ?? 0) +
      (runStatus?.error_code ? 1 : 0) +
      (activeStage === "publish" && artifactCount === 0 ? 1 : 0),
    runBlockersCount: (runStatus?.pending_permissions?.length ?? 0) + (runStatus?.error_code ? 1 : 0) + (runStatus?.status === "failed" ? 1 : 0),
    runErrorCode: runStatus?.error_code ?? undefined,
    reviewFindingsCount: openQuestions.trim() ? 1 : 0,
    releaseBlockersCount: runStatus?.error_code === "release_verdict_FAIL" ? 1 : 0,
    gitActionFailed: Boolean(gitError),
  }), [activeStage, artifactCount, blockers.length, doctorFailures.length, gitError, openQuestions, proposalArtifacts.length, runStatus, setupDoctorResult, validateResult, validationErrors.length]);

  const runtimeSettingsPanel = (
    <RuntimeProfileSettingsPanel
      busy={busy}
      runtimeTimeoutKeys={[...runtimeTimeoutKeys]}
      runtimeTimeoutLabels={runtimeTimeoutLabels}
      runtimeTimeoutDraft={runtimeTimeoutDraft}
      runtimeTimeoutPersisted={runtimeTimeoutPersisted}
      runtimeTimeoutEffective={runtimeTimeoutEffective}
      runtimeTimeoutSource={runtimeTimeoutSource}
      runtimeTimeoutStatus={runtimeTimeoutStatus}
      onReloadTimeouts={() => void loadRuntimeTimeouts()}
      onSaveTimeouts={() => void handleSaveRuntimeTimeouts()}
      onResetTimeouts={() => void handleResetRuntimeTimeouts()}
      onTimeoutChange={(key, value) => updateRuntimeTimeoutDraft(key as RuntimeTimeoutKey, value)}
      runtimeExecutionLabels={runtimeExecutionLabels}
      runtimeExecutionDraft={runtimeExecutionDraft}
      runtimeExecutionPersisted={runtimeExecutionPersisted}
      runtimeExecutionEffective={runtimeExecutionEffective}
      runtimeExecutionSource={runtimeExecutionSource}
      runtimeExecutionStatus={runtimeExecutionStatus}
      onReloadExecution={() => void loadRuntimeExecution()}
      onSaveExecution={() => void handleSaveRuntimeExecution()}
      onResetExecution={() => void handleResetRuntimeExecution()}
      onExecutionChange={(key, value) => updateRuntimeExecutionDraft(key as RuntimeExecutionKey, value)}
      runtimePermissionLabels={runtimePermissionLabels}
      runtimePermissionDraft={runtimePermissionDraft}
      runtimePermissionPersisted={runtimePermissionPersisted}
      runtimePermissionEffective={runtimePermissionEffective}
      runtimePermissionSource={runtimePermissionSource}
      runtimePermissionStatus={runtimePermissionStatus}
      onReloadPermissions={() => void loadRuntimePermissions()}
      onSavePermissions={() => void handleSaveRuntimePermissions()}
      onResetPermissions={() => void handleResetRuntimePermissions()}
      onPermissionChange={(key, value) => updateRuntimePermissionDraft(key as RuntimePermissionKey, value)}
      stepProviderLabels={runtimeStepProviderLabels}
      stepProviderOrder={[...runtimeStepProviderOrder]}
      stepProviderPersisted={runtimeStepProviderPersisted}
      stepProviderEffective={runtimeStepProviderEffective}
      stepProviderSource={runtimeStepProviderSource}
      onReloadProfile={() => void loadRuntimeProfile()}
    />
  );

  function handleInspectorPrimaryAction() {
    switch (nextAction.primaryActionId) {
      case "source":
        void handleSetupSaveGuidedWorkspaceSetup();
        break;
      case "readiness":
        if (nextAction.intent === "open-readiness") {
          setActiveStage("readiness");
          break;
        }
        if (!validateResult?.ok) {
          void handleValidateWorkspace();
        } else if (!setupDoctorResult?.ok) {
          void handleSetupDoctorCheck();
        } else {
          void handleSetupFirstRun();
        }
        break;
      case "charter":
        void handleSaveStep0WizardContract();
        break;
      case "analysis":
        if (nextAction.intent === "focus-analysis-blocker") {
          setActiveStage("analysis");
          setAnalysisFocusSignal((value) => value + 1);
        } else {
          setActiveStage("analysis");
          void handleRunPipeline(runStatus ? "refresh" : "init");
        }
        break;
      case "review":
        enterReviewStage();
        break;
      case "proposals":
        setActiveStage("proposals");
        break;
      case "ask":
        if (activeStage === "ask") {
          setAskPrimaryActionSignal((value) => value + 1);
        } else {
          setActiveStage("ask");
          window.requestAnimationFrame(() => document.getElementById("qaQuestion")?.focus());
        }
        break;
      case "publish":
        void handleGitCommit();
        break;
    }
  }

  if (!consoleReady || onboardingStatus?.can_enter_console !== true) {
    return (
      <OnboardingShell
        busy={busy}
        error={error}
        status={onboardingStatus}
        workspacePath={onboardingWorkspacePath}
        createWorkspace={onboardingCreateWorkspace}
        guidedRepos={guidedRepos}
        guidedDocsImportsPath={guidedDocsImportsPath}
        validateResult={validateResult}
        doctorResult={setupDoctorResult}
        setupRuntime={setupRuntime}
        setupRuntimeProvider={setupRuntimeProvider}
        firstRunStatus={firstRunStatus}
        onWorkspacePathChange={setOnboardingWorkspacePath}
        onCreateWorkspaceChange={setOnboardingCreateWorkspace}
        onSelectWorkspace={() => void handleOnboardingWorkspaceSelect()}
        onOpenRecentWorkspace={(path) => void handleOpenRecentWorkspace(path)}
        onForgetRecentWorkspace={(path) => void handleForgetRecentWorkspace(path)}
        onRepoChange={handleSetupRepoChange}
        onAddRepo={handleSetupAddRepo}
        onRemoveRepo={handleSetupRemoveRepo}
        onDocsImportsPathChange={handleSetupDocsImportsPathChange}
        onSaveSources={() => void handleOnboardingSaveSources()}
        onRuntimeChange={handleSetupRuntimeChange}
        onRuntimeProviderChange={handleSetupRuntimeProviderChange}
        onSaveRuntime={() => void handleOnboardingSaveRuntime()}
        onCheckDoctor={() => void handleSetupDoctorCheck()}
        onEnterConsole={() => void handleOnboardingEnterConsole()}
        onRunFirstAnalysis={() => void handleOnboardingRunFirstAnalysis()}
      />
    );
  }

  return (
    <AppShell
      buildVersion={systemVersion.version}
      buildCommit={systemVersion.commit}
      buildBuilt={systemVersion.built}
      uiBundle={systemVersion.ui_bundle}
      workspacePath={validateResult?.workspace ?? workspaceRootPath ?? "bound workspace"}
      repoCount={validateResult?.resolved_repos?.length ?? guidedRepos.length}
      runtimeMode={setupRuntime}
      runtimeProvider={setupRuntimeProvider}
      permissionMode={String(runtimePermissionEffective.mode ?? "trusted_full_access")}
      gitStatus={gitStatus}
      healthLabel={setupDoctorResult?.ok ? "local ready" : validateResult?.ok ? "workspace valid" : "local connected"}
      stages={stages}
      activeStage={activeStage}
      nextAction={nextAction}
      blockers={blockers}
      evidenceRefs={evidenceRefs}
      workspaceHealth={workspaceHealth}
      runtimeSafety={runtimeSafety}
      gitPublication={gitPublication}
      runStatus={runStatus}
      runReviewSummary={runReviewSummary}
      runtimeLabel={runtimeLabel}
      cancelBusy={cancelBusy}
      selectedRunIsActive={selectedRunIsActive}
      selectedRunId={runStatus?.run_id}
      selectedRunStatus={runStatus?.status}
      selectedRunErrorCode={runStatus?.error_code ?? undefined}
      selectedRunError={runStatus?.error ?? undefined}
      logs={filteredRunLogs}
      renderedLogs={runLogsRendered}
      runLogsStatus={runLogsStatus}
      runLogsMode={runLogsMode}
      runLogsViewMode={runLogsViewMode}
      canExportLogs={filteredRunLogs.length > 0}
      taskrunPaths={runLogTaskrunPaths}
      onRefresh={() => void bootstrapApp()}
      onStageChange={handleStageChange}
      onPrimaryAction={handleInspectorPrimaryAction}
      onCancelRun={() => void handleCancelSelectedRun()}
      onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)}
      onRunLogsModeChange={setRunLogsMode}
      onRunLogsViewModeChange={setRunLogsViewMode}
      onCopyRunLogs={() => void handleCopyRunLogs()}
      onDownloadRunLogs={handleDownloadRunLogs}
    >
      {activeStage === "source" ? (
        <SourceStagePanel
          busy={busy}
          guidedRepos={guidedRepos}
          guidedDocsImportsPath={guidedDocsImportsPath}
          manifestContent={manifestContent}
          manifestStatus={manifestStatus}
          validateResult={validateResult}
          validationDiagnosticsByRepo={validationDiagnosticsByRepo}
          doctorResult={setupDoctorResult}
          doctorStatus={setupDoctorStatus}
          setupRuntime={setupRuntime}
          setupRuntimeProvider={setupRuntimeProvider}
          onRepoChange={handleSetupRepoChange}
          onAddRepo={handleSetupAddRepo}
          onRemoveRepo={handleSetupRemoveRepo}
          onDocsImportsPathChange={handleSetupDocsImportsPathChange}
          onApplyGuidedWorkspaceSetup={handleSetupApplyGuidedWorkspaceSetup}
          onSaveGuidedWorkspaceSetup={() => void handleSetupSaveGuidedWorkspaceSetup()}
          onManifestChange={handleSetupManifestChange}
          onSaveManifest={() => void handleSaveManifest()}
        />
      ) : null}

      {activeStage === "readiness" ? (
        <ReadinessStagePanel
          busy={busy}
          validateResult={validateResult}
          validationDiagnosticsByRepo={validationDiagnosticsByRepo}
          doctorResult={setupDoctorResult}
          doctorStatus={setupDoctorStatus}
          firstRunStatus={firstRunStatus}
          setupRuntime={setupRuntime}
          setupRuntimeProvider={setupRuntimeProvider}
          selectedRunErrorCode={runStatus?.error_code}
          selectedRunError={runStatus?.error}
          onSetupRuntimeChange={handleSetupRuntimeChange}
          onSetupRuntimeProviderChange={handleSetupRuntimeProviderChange}
          onValidateWorkspace={() => void handleValidateWorkspaceWithHealth()}
          onCheckDoctor={() => void handleSetupDoctorCheck()}
          onRunFirstAnalysis={() => void handleSetupFirstRun("analysis")}
          runtimeSettingsPanel={runtimeSettingsPanel}
          artifactCount={artifactCount}
          workspaceHealthReport={workspaceHealthReport}
          workspaceHealthStatus={workspaceHealthStatus}
          workspaceHealthError={workspaceHealthError}
          onRefreshWorkspaceHealth={() => void refreshWorkspaceHealth()}
          runtimeTimeoutEffective={runtimeTimeoutEffective}
          runtimeExecutionEffective={runtimeExecutionEffective}
          runtimePermissionEffective={runtimePermissionEffective}
          runtimeStepProviderEffective={runtimeStepProviderEffective}
        />
      ) : null}

      {activeStage === "charter" ? (
        <CharterStagePanel
          wizardProjectName={wizardProjectName}
          wizardScope={wizardScope}
          wizardNfr={wizardNfr}
          wizardRules={wizardRules}
          gitStatus={gitStatus}
          proposalBranch={proposalBranch}
          wizardPanel={
            <WizardContractPanel
              busy={busy}
              wizardProjectName={wizardProjectName}
              wizardScope={wizardScope}
              wizardNfr={wizardNfr}
              wizardRules={wizardRules}
              wizardStatus={wizardStatus}
              onProjectNameChange={setWizardProjectName}
              onScopeChange={setWizardScope}
              onNfrChange={setWizardNfr}
              onRulesChange={setWizardRules}
              onSave={() => void handleSaveStep0WizardContract()}
            />
          }
          gitPanel={
            <BaselineGitPanel
              busy={busy}
              gitMessage={gitMessage}
              proposalBranch={proposalBranch}
              gitStatus={gitStatus}
              gitError={gitError}
              onGitMessageChange={setGitMessage}
              onProposalBranchChange={setProposalBranch}
              onCommit={() => void handleGitCommit()}
              onCreateProposalBranch={() => void handleCreateProposalBranch()}
            />
          }
          busy={busy}
          baselineBundleWarnings={baselineBundleWarnings}
          baselineEditorArtifacts={baselineEditorArtifacts}
          selectedEditorPath={selectedEditorPath}
          selectedEditorContent={selectedEditorContent}
          editorStatus={editorStatus}
          onEditorSelectionChange={(path) => void handleEditorSelectionChange(path)}
          onEditorContentChange={setSelectedEditorContent}
          onSave={() => void handleSaveSelectedEditorArtifact()}
        />
      ) : null}

      {activeStage === "analysis" ? (
        <AnalysisStagePanel
          busy={busy}
          cancelBusy={cancelBusy}
          runId={runId}
          runStatus={runStatus}
          runList={runList}
          runActionStatus={runActionStatus}
          selectedRunWarnings={selectedRunWarnings}
          selectedRunIsActive={selectedRunIsActive}
          runCounters={runCounters}
          pendingPermissions={runStatus?.pending_permissions ?? []}
          runLogs={runLogs}
          artifacts={[...nonDiagramArtifacts, ...diagramArtifacts]}
          setupRuntime={setupRuntime}
          setupRuntimeProvider={setupRuntimeProvider}
          runReviewSummary={runReviewSummary}
          runReviewStatus={runReviewStatus}
          gitDiff={gitDiff}
          gitDiffStatus={gitDiffStatus}
          onLoadGitDiff={handleLoadGitDiff}
          focusBlockerSignal={analysisFocusSignal}
          onRunPipeline={(pipeline) => void handleRunPipeline(pipeline)}
          onCancelSelectedRun={() => void handleCancelSelectedRun()}
          onSelectRun={(id) => void handleSelectRun(id)}
          onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)}
        />
      ) : null}

      {activeStage === "review" ? (
        <ReviewStagePanel
          runId={runId}
          runStatus={runStatus}
          runList={runList}
          coverageSummary={coverageSummary}
          openQuestions={openQuestions}
          nonDiagramArtifacts={nonDiagramArtifacts}
          diagramArtifacts={diagramArtifacts}
          selectedArtifact={selectedArtifact}
          selectedArtifactContent={selectedArtifactContent}
          selectedArtifactIsMermaid={selectedArtifactIsMermaid}
          runLogs={runLogs}
          reviewSummary={runReviewSummary}
          gitDiff={gitDiff}
          gitDiffStatus={gitDiffStatus}
          onLoadGitDiff={handleLoadGitDiff}
          onSelectRun={(id) => void handleSelectRun(id)}
          onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)}
        />
      ) : null}

      {activeStage === "proposals" ? (
        <ProposalsStagePanel
          artifacts={[...nonDiagramArtifacts, ...diagramArtifacts]}
          selectedArtifact={selectedArtifact}
          selectedArtifactContent={selectedArtifactContent}
          openQuestions={openQuestions}
          proposalBranch={proposalBranch}
          gitStatus={gitStatus}
          runLogs={runLogs}
          gitDiff={gitDiff}
          gitDiffStatus={gitDiffStatus}
          onLoadGitDiff={handleLoadGitDiff}
          onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)}
          onGoPublish={() => setActiveStage("publish")}
        />
      ) : null}

      {activeStage === "ask" ? <AskStagePanel primaryActionSignal={askPrimaryActionSignal} onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)} /> : null}

      {activeStage === "publish" ? (
        <PublishStagePanel
          busy={busy}
          gitMessage={gitMessage}
          proposalBranch={proposalBranch}
          gitStatus={gitStatus}
          gitError={gitError}
          artifacts={[...nonDiagramArtifacts, ...diagramArtifacts]}
          selectedArtifact={selectedArtifact}
          selectedArtifactContent={selectedArtifactContent}
          openQuestions={openQuestions}
          externalGateItems={publishExternalGateItems}
          gitDiff={gitDiff}
          gitDiffStatus={gitDiffStatus}
          onLoadGitDiff={handleLoadGitDiff}
          onGitMessageChange={setGitMessage}
          onProposalBranchChange={setProposalBranch}
          onCommit={() => void handleGitCommit()}
          onCreateProposalBranch={() => void handleCreateProposalBranch()}
          onPreviewArtifact={(path) => void handleOpenArtifact(path)}
        />
      ) : null}

      {error ? <p className="status err">Error: {error}</p> : null}
    </AppShell>
  );
}

function selectedRunIssueCopy(errorCode: string, error: string | null | undefined, surface: "inspector" | "publish"): { label: string; detail: string } {
  if (isRunCanceled(errorCode)) {
    return {
      label: "Canceled run",
      detail:
        surface === "publish"
          ? "run_canceled: select a successful run or start a new analysis before publishing."
          : "run_canceled: selected run was stopped by request; taskrun evidence remains in History.",
    };
  }
  if (isRunReconciledAfterRestart(errorCode)) {
    return {
      label: "Run reconciled after restart",
      detail:
        surface === "publish"
          ? "run_reconciled_after_restart: select a completed artifact run or start a new analysis before publishing."
          : "run_reconciled_after_restart: ACP preserved the stale run evidence in History after service restart.",
    };
  }
  if (isRunnerUnavailable(errorCode)) {
    return {
      label: "Provider unavailable",
      detail:
        surface === "publish"
          ? "runner_unavailable: check Readiness provider setup, binary/auth/quota, then run a successful analysis before publishing."
          : "runner_unavailable: check Readiness provider setup, binary/auth/quota, then retry the same analysis pipeline.",
    };
  }
  return {
    label: errorCode,
    detail: error || (surface === "publish" ? "Selected run failed before publication." : "Selected run failed."),
  };
}

function isProposalReviewArtifact(path: string): boolean {
  return path.startsWith("proposals/") || path.startsWith("reports/changelog/");
}

function formatPermissionBlockerDetail(request: RuntimePermissionRequest): string {
  const step = request.step_id || "runtime step";
  const decision = request.decision?.decision || "pending";
  const rule = request.decision?.rule_id ? ` via ${request.decision.rule_id}` : "";
  const target = request.path_or_command ? ` Target: ${request.path_or_command}.` : "";
  const reason = request.reason || request.decision?.message;
  const reasonDetail = reason ? ` Reason: ${reason}` : "";
  return `${step} paused for ${decision}${rule}.${target}${reasonDetail}`;
}

function workspaceHealthSeverity(severity: string): InspectorItem["severity"] {
  switch (severity) {
    case "error":
      return "error";
    case "warning":
      return "warn";
    default:
      return "info";
  }
}

function deriveNextAction(
  activeStage: StageId,
  state: {
    validateOK: boolean;
    doctorOK: boolean;
    hasArtifacts: boolean;
    hasProposals: boolean;
    hasRun: boolean;
    blockersCount: number;
    hardBlockersCount: number;
    runBlockersCount: number;
    runErrorCode?: string | null;
    reviewFindingsCount: number;
    releaseBlockersCount: number;
    gitActionFailed: boolean;
  },
): NextAction {
  if (activeStage === "publish") {
    const disabledReason = !state.hasArtifacts
      ? "No generated workspace artifacts are ready to publish."
      : state.gitActionFailed
        ? "Review the Git action failure in Commit plan before retrying."
      : state.hardBlockersCount > 0
        ? "Resolve hard blockers before committing workspace artifacts."
        : undefined;
    return {
      label: state.gitActionFailed ? "Resolve Git action failure" : "Commit selected artifacts",
      description: state.gitActionFailed
        ? "Use the Commit plan recovery details, local Git status and prepared message before retrying the Git action."
        : disabledReason
          ? "Resolve publish blockers before creating a Git commit."
          : "Create a Git commit for reviewed architecture workspace updates.",
      primaryActionId: "publish",
      disabledReason,
    };
  }
  if (activeStage === "analysis" && state.runBlockersCount > 0) {
    if (isRunCanceled(state.runErrorCode)) {
      return {
        label: "Review retained run evidence",
        description: "The selected run was canceled; inspect retained History evidence or start a new analysis when ready.",
        primaryActionId: "analysis",
        intent: "focus-analysis-blocker",
      };
    }
    if (isRunReconciledAfterRestart(state.runErrorCode)) {
      return {
        label: "Review retained run evidence",
        description: "The selected run was recovered after restart; inspect retained History evidence or start a new analysis when ready.",
        primaryActionId: "analysis",
        intent: "focus-analysis-blocker",
      };
    }
    if (isRunnerUnavailable(state.runErrorCode)) {
      return {
        label: "Check provider readiness",
        description: "Provider/tool availability blocked the selected run; verify binary/auth/quota in Readiness before retrying the same pipeline.",
        primaryActionId: "readiness",
        intent: "open-readiness",
      };
    }
    return {
      label: "Review blocker",
      description: "Focus the failed shard, pending permission or runtime error before retrying analysis.",
      primaryActionId: "analysis",
      intent: "focus-analysis-blocker",
    };
  }
  switch (activeStage) {
    case "source":
      return {
        label: "Save and validate sources",
        description: "Persist source settings to workspace.yaml and run workspace validation.",
        primaryActionId: "source",
      };
    case "readiness":
      if (!state.validateOK) {
        return {
          label: "Validate workspace",
          description: "Check manifest, layout and repo source resolution.",
          primaryActionId: "readiness",
        };
      }
      if (!state.doctorOK) {
        return {
          label: "Check local readiness",
          description: "Verify local git, workspace write access, embedded UI and runtime provider readiness.",
          primaryActionId: "readiness",
        };
      }
      return {
        label: "Run first analysis",
        description: "Start init pipeline after validation and local readiness pass.",
        primaryActionId: "readiness",
      };
    case "charter":
      return {
        label: "Save charter contract",
        description: "Persist scope, NFR priorities and rules for step0 constitution.",
        primaryActionId: "charter",
      };
    case "analysis":
      return {
        label: state.hasRun ? "Run refresh" : "Run init",
        description: state.hasRun ? "Refresh architecture evidence from the current workspace." : "Run the initial staged architecture analysis.",
        primaryActionId: "analysis",
      };
    case "review":
      if (state.blockersCount > 0) {
        if (state.releaseBlockersCount > 0) {
          return {
            label: "Verify release blockers",
            description: "Inspect release verdict evidence before making a release decision.",
            primaryActionId: "review",
          };
        }
        if (state.hardBlockersCount === 0 && state.reviewFindingsCount > 0) {
          return {
            label: "Review findings",
            description: "Inspect coverage questions and findings before publishing.",
            primaryActionId: "review",
          };
        }
        return {
          label: "Resolve blockers",
          description: "Resolve workspace, doctor or runtime blockers before publishing.",
          primaryActionId: "readiness",
        };
      }
      return {
        label: state.hasArtifacts ? "Ask about evidence" : "Run analysis first",
        description: state.hasArtifacts ? "Use workspace-backed Q&A to inspect unresolved architecture context." : "Start analysis to generate reviewable evidence artifacts.",
        primaryActionId: state.hasArtifacts ? "ask" : "analysis",
      };
    case "proposals":
      return {
        label: state.hasProposals ? "Review proposal" : "Review results first",
        description: state.hasProposals ? "Inspect the generated proposal package, linked evidence and publication readiness." : "No proposal artifacts are available yet.",
        primaryActionId: state.hasProposals ? "proposals" : "review",
      };
    case "ask":
      return {
        label: "Ask workspace",
        description: "Submit an agent-backed question over existing workspace artifacts.",
        primaryActionId: "ask",
        intent: "submit-ask",
      };
  }
}
