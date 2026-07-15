import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { ProductShell } from "./components/ProductShell";
import { ModalDialog } from "./components/ModalDialog";
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
  type Artifact,
  type GuidedRepo,
  type OnboardingStatusResponse,
  type RuntimeExecutionKey,
  type RuntimePermissionKey,
  type RuntimePermissionRequest,
  type RuntimeTimeoutKey,
  type RunStatusResponse,
  type SystemVersionResponse,
  type WorkspaceHealthResponse,
} from "./lib/appContracts";
import type { InspectorItem, NextAction, StageId } from "./lib/consoleTypes";
import { destinationForStage, formatAppRoute, parseAppRoute, stageForRoute, type AppRoute } from "./lib/appRoutes";
import type { LoadGitDiffOptions } from "./lib/gitDiffApi";
import { runtimeDisplayLabel } from "./lib/runtimeDisplay";
import { isRunCanceled, isRunReconciledAfterRestart, isRunnerUnavailable } from "./lib/runState";
import { deriveWorkflowState, type WorkflowDestination, type WorkflowState } from "./lib/workflowState";
import { useRunExplorer } from "./hooks/useRunExplorer";
import { useRuntimeSettings } from "./hooks/useRuntimeSettings";
import { useWorkspaceSetup } from "./hooks/useWorkspaceSetup";
import { enterOnboardingConsole, forgetOnboardingRecentWorkspace, loadOnboardingStatus, selectOnboardingRuntime, selectOnboardingWorkspace } from "./lib/onboardingApi";
import { loadSystemDoctor, loadSystemVersion } from "./lib/systemApi";
import { loadWorkspaceHealthAPI } from "./lib/workspaceApi";

export default function App() {
  const [route, setRoute] = useState<AppRoute>(() => parseAppRoute(window.location, true));
  const destination = route.destination;
  const [activeStage, setActiveStageState] = useState<StageId>(() => stageForRoute(parseAppRoute(window.location, true)));
  const [routeNotice, setRouteNotice] = useState("");
  const unsavedDraftRef = useRef(false);
  const restoredRouteRunRef = useRef<string | null>(null);
  const restoredArtifactRef = useRef<string | null>(null);
  const defaultChangesRunRef = useRef<string | null>(null);
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
  const [analysisFocusSignal] = useState(0);
  const [askPrimaryActionSignal] = useState(0);
  const [workspaceHealthReport, setWorkspaceHealthReport] = useState<WorkspaceHealthResponse | null>(null);
  const [workspaceHealthStatus, setWorkspaceHealthStatus] = useState<"idle" | "loading" | "loaded" | "error">("idle");
  const [workspaceHealthError, setWorkspaceHealthError] = useState("");

  const navigateRoute = useCallback((nextRoute: AppRoute, replace = false) => {
    if (!replace && route.destination === "setup" && nextRoute.destination !== "setup" && unsavedDraftRef.current && !window.confirm("Leave Setup? Unsaved workspace or editor changes will be lost.")) return;
    const nextPath = formatAppRoute(nextRoute);
    if (`${window.location.pathname}${window.location.search}` !== nextPath) window.history[replace ? "replaceState" : "pushState"]({}, "", nextPath);
    setRoute(nextRoute);
    setActiveStageState(stageForRoute(nextRoute));
  }, [route.destination]);

  const navigateDestination = useCallback((nextDestination: WorkflowDestination, replace = false) => {
    navigateRoute({ destination: nextDestination, invalid: [] }, replace);
  }, [navigateRoute]);

  const setActiveStage = useCallback((stage: StageId) => {
    const nextDestination = destinationForStage(stage);
    const nextRoute: AppRoute = nextDestination === route.destination ? { ...route, invalid: [] } : { destination: nextDestination, invalid: [] };
    if (nextDestination === "setup") nextRoute.setupStep = stage === "readiness" ? "runner" : stage === "charter" ? "brief" : "sources";
    if (nextDestination === "changes") nextRoute.changesView = stage === "publish" ? "publish" : stage === "proposals" ? "proposals" : "overview";
    navigateRoute(nextRoute);
  }, [navigateRoute, route]);

  const handleDestinationChange = useCallback((nextDestination: WorkflowDestination) => {
    navigateDestination(nextDestination);
  }, [navigateDestination]);

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
    effectiveRuntimeMode,
    effectiveRuntimeProvider,
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
    coordination,
    selectedArtifact,
    selectedArtifactContent,
    runActionStatus,
    cancelBusy,
    coverageSummary,
    openQuestions,
    evidenceSnapshot,
    runCounters,
    runLogs,
    diagramArtifacts,
    nonDiagramArtifacts,
    selectedRunWarnings,
    selectedRunIsActive,
    runReviewSummary,
    runReviewStatus,
    gitDiff,
    gitDiffStatus,
    bootstrapRuns,
    clearRunSelection,
    loadGitDiff,
    handleRunPipeline,
    handleSelectRun,
    handleCancelSelectedRun,
    handleCancelRun,
    handleOpenArtifact,
  } = runExplorer;

  const {
    validateResult,
    validationDiagnosticsByRepo,
    manifestContent,
    hasUnsavedManifestDraft,
    manifestStatus,
    baselineEditorArtifacts,
    baselineBundleWarnings,
    workspaceRootPath,
    selectedEditorPath,
    selectedEditorContent,
    hasUnsavedEditorDraft,
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
    gitConfirmation,
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
    confirmGitAction,
    cancelGitAction,
  } = workspaceSetup;

  useEffect(() => {
    unsavedDraftRef.current = hasUnsavedManifestDraft || hasUnsavedEditorDraft;
  }, [hasUnsavedEditorDraft, hasUnsavedManifestDraft]);

  useEffect(() => {
    const warnBeforeUnload = (event: BeforeUnloadEvent) => {
      if (!unsavedDraftRef.current) return;
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", warnBeforeUnload);
    return () => window.removeEventListener("beforeunload", warnBeforeUnload);
  }, []);

  useEffect(() => {
    void bootstrapApp();
  }, []);

  useEffect(() => {
    const handlePopState = () => {
      const nextRoute = parseAppRoute(window.location, consoleReady);
      setRoute(nextRoute);
      setActiveStageState(stageForRoute(nextRoute));
      setRouteNotice(nextRoute.invalid.length ? `Unsupported URL context was removed: ${nextRoute.invalid.join(", ")}.` : "");
    };
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, [consoleReady]);

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
		navigateDestination("setup", true);
        return;
      }
      await bootstrapConsoleData({ validateWorkspace: true });
      setConsoleReady(true);
		const restoredRoute = parseAppRoute(window.location, true);
		setRouteNotice(restoredRoute.invalid.length ? `Unsupported URL context was removed: ${restoredRoute.invalid.join(", ")}.` : "");
		navigateRoute(restoredRoute, true);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "console data refresh failed");
    }
  }

  async function handleConsoleRefresh() {
    setBusy(true);
    setError(null);
    try {
      await bootstrapApp();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "console data refresh failed");
    } finally {
      setBusy(false);
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
	const enteredStatus = await enterOnboardingConsole();
	syncOnboardingStatus(enteredStatus);
	setConsoleReady(true);
	handleDestinationChange("home");
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

  const handleOpenArtifactAndReview = useCallback(async (path: string) => {
    await handleOpenArtifact(path);
    const artifactKey = [...nonDiagramArtifacts, ...diagramArtifacts].find((artifact) => artifact.path === path)?.id || path;
    navigateRoute({
      destination: "changes",
      runId: runId ?? undefined,
      runRequested: Boolean(runId),
      changesView: route.destination === "changes" ? route.changesView ?? "overview" : "evidence",
      source: "snapshot",
      artifact: artifactKey,
      mode: route.mode ?? "rendered",
      invalid: [],
    });
  }, [diagramArtifacts, handleOpenArtifact, navigateRoute, nonDiagramArtifacts, route.changesView, route.destination, route.mode, runId]);

  async function handleSelectRunAndRoute(id: string) {
    if (destination === "runs") navigateRoute({ destination: "runs", runId: id, runRequested: true, invalid: [] });
    else navigateRoute({ ...route, destination: "changes", runId: id, runRequested: true, source: "snapshot", invalid: [] });
    await handleSelectRun(id);
  }

  useEffect(() => {
    if (!consoleReady || runList.length === 0) return;
    if (route.runId) {
      if (!runList.some((item) => item.run_id === route.runId)) {
        if (restoredRouteRunRef.current !== route.runId) {
          setRouteNotice(`Run ${route.runId} is unavailable. Select another run; the source was not changed.`);
          restoredRouteRunRef.current = route.runId;
          clearRunSelection();
          navigateRoute({ ...route, runId: undefined, runRequested: true, artifact: undefined, invalid: [] }, true);
        }
      } else if (runId !== route.runId && restoredRouteRunRef.current !== route.runId) {
        restoredRouteRunRef.current = route.runId;
        void handleSelectRun(route.runId);
      }
      return;
    }
    restoredRouteRunRef.current = null;
    if (route.destination === "changes" && !route.runRequested) {
      const latest = runList.find((item) => item.status === "succeeded" && (item.pipeline === "init" || item.pipeline === "refresh"));
      if (latest && defaultChangesRunRef.current !== latest.run_id) {
        defaultChangesRunRef.current = latest.run_id;
        navigateRoute({ ...route, runId: latest.run_id, runRequested: true, source: "snapshot", invalid: [] }, true);
        if (runId !== latest.run_id) void handleSelectRun(latest.run_id);
      }
    }
  }, [clearRunSelection, consoleReady, handleSelectRun, navigateRoute, route, runId, runList]);

  useEffect(() => {
    if (!route.artifact || evidenceSnapshot.status === "idle" || evidenceSnapshot.status === "loading") return;
    const match = [...nonDiagramArtifacts, ...diagramArtifacts].find((artifact) => artifact.id === route.artifact || artifact.path === route.artifact);
    if (!match) {
      setRouteNotice(`Artifact ${route.artifact} is unavailable in the selected source.`);
      navigateRoute({ ...route, artifact: undefined, invalid: [] }, true);
      return;
    }
    if (selectedArtifact !== match.path && restoredArtifactRef.current !== route.artifact) {
      restoredArtifactRef.current = route.artifact;
      void handleOpenArtifact(match.path);
    }
  }, [diagramArtifacts, evidenceSnapshot.status, handleOpenArtifact, navigateRoute, nonDiagramArtifacts, route, selectedArtifact]);

  useEffect(() => {
    if (!route.artifact) restoredArtifactRef.current = null;
  }, [route.artifact]);

  const diagnostics = useMemo(() => [...(validateResult?.errors ?? []), ...(validateResult?.warnings ?? [])], [validateResult]);
  const validationErrors = useMemo(() => diagnostics.filter((diagnostic) => diagnostic.level === "error"), [diagnostics]);
  const doctorFailures = useMemo(() => setupDoctorResult?.checks.filter((check) => check.status === "fail") ?? [], [setupDoctorResult]);
  const artifactCount = nonDiagramArtifacts.length + diagramArtifacts.length;
  useEffect(() => {
    if (destination !== "home") return;
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
  }, [artifactCount, destination, selectedRunIsActive, setActiveStage]);

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

  const runtimeLabel = runtimeDisplayLabel(effectiveRuntimeMode === "unknown" ? "" : effectiveRuntimeMode, effectiveRuntimeProvider, { compact: true });
  const selectedRunProvider = useMemo(() => {
    if (!runStatus?.runtime_mode) {
      return "Unknown";
    }
    const providers = Array.from(new Set(Object.values(runStatus.step_providers ?? {}).map((value) => value.trim()).filter(Boolean)));
    return providers.length > 1 ? "Mixed" : providers[0] ?? (runStatus.runtime_mode === "fake" ? "fake" : "Unknown");
  }, [runStatus?.runtime_mode, runStatus?.step_providers]);

  useEffect(() => {
    if (runStatus?.runtime_mode === "fake" && gitMessage === "chore: update ACP workspace artifacts") {
      setGitMessage("chore: publish deterministic ACP demo evidence");
    }
  }, [gitMessage, runStatus?.runtime_mode, setGitMessage]);

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

  const workflow = useMemo(() => {
    const hasRunning = runList.some((run) => run.status === "running");
    const hasQueued = runList.some((run) => run.status === "queued");
    const evidence = evidenceSnapshot.status === "available"
      ? "snapshot"
      : evidenceSnapshot.status === "partial"
        ? "partial"
        : evidenceSnapshot.status === "not_produced" || evidenceSnapshot.status === "unavailable" || evidenceSnapshot.status === "error"
          ? "unavailable"
          : artifactCount > 0 ? "current" : "none";
    return deriveWorkflowState({
      workspace: validateResult?.ok ? "ready" : validateResult ? "invalid" : "unconfigured",
      execution: hasRunning ? "active" : hasQueued ? "pending" : runStatus?.status === "failed" ? "failed" : runStatus?.status === "succeeded" ? "succeeded" : "idle",
      evidence,
      publication: gitError.toLowerCase().includes("stale_git_confirmation") ? "stale" : gitError ? "blocked" : (gitDiff?.files?.length ?? 0) > 0 ? "dirty" : "clean",
      openQuestions: openQuestions.split("\n").filter((line) => /^\s*[-*]\s+/.test(line)).length,
      demo: runStatus?.runtime_mode === "fake",
    });
  }, [artifactCount, evidenceSnapshot.status, gitDiff?.files?.length, gitError, openQuestions, runList, runStatus?.runtime_mode, runStatus?.status, validateResult]);

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
    <>
      <ProductShell
        destination={destination}
        workflow={workflow}
        workspacePath={validateResult?.workspace ?? workspaceRootPath ?? "bound workspace"}
        runtimeLabel={runtimeLabel}
        buildLabel={`${systemVersion.version} · ${systemVersion.commit}`}
        buildTitle={`version=${systemVersion.version}; commit=${systemVersion.commit}; built=${systemVersion.built}`}
        workspaceValid={validateResult?.ok === true}
        onDestinationChange={handleDestinationChange}
        onAsk={() => setActiveStageState("ask")}
        onSettings={() => { handleDestinationChange("setup"); setActiveStageState("readiness"); }}
        onDiagnostics={() => { handleDestinationChange("runs"); setActiveStageState("analysis"); }}
        onRefresh={() => void handleConsoleRefresh()}
      >
	  {destination === "setup" ? (
		<nav className="destination-tabs" aria-label="Setup sections">
			  {(["source", "readiness", "charter"] as const).map((stage) => <button key={stage} type="button" data-testid={`stage-${stage}`} aria-current={activeStage === stage ? "page" : undefined} onClick={() => setActiveStage(stage)}>{stage === "source" ? "Workspace" : stage === "readiness" ? "Runtime & readiness" : "Charter"}</button>)}
		</nav>
	  ) : null}
	  {destination === "changes" ? (
		<nav className="destination-tabs" aria-label="Changes sections">
			  {(["review", "proposals", "publish"] as const).map((stage) => <button key={stage} type="button" data-testid={`stage-${stage}`} aria-current={activeStage === stage ? "page" : undefined} onClick={() => setActiveStage(stage)}>{stage === "review" ? "Review" : stage === "proposals" ? "Proposals" : "Publish"}</button>)}
		</nav>
	  ) : null}
	  {destination === "home" && activeStage !== "ask" ? (
		<HomePanel workflow={workflow} runStatus={runStatus} evidenceStatus={evidenceSnapshot.status} gitChanges={gitDiff?.files?.length ?? 0} onPrimaryAction={() => handleDestinationChange(workflow.nextAction.destination)} />
	  ) : null}
	  {destination === "knowledge" && activeStage !== "ask" ? (
		<KnowledgePanel artifacts={[...nonDiagramArtifacts, ...diagramArtifacts]} evidenceStatus={evidenceSnapshot.status} onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)} />
	  ) : null}

      {destination === "setup" && activeStage === "source" ? (
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
          setupRuntime={effectiveRuntimeMode}
          setupRuntimeProvider={effectiveRuntimeProvider}
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

      {destination === "setup" && activeStage === "readiness" ? (
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

      {destination === "setup" && activeStage === "charter" ? (
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

      {destination === "runs" && activeStage === "analysis" ? (
        <AnalysisStagePanel
          busy={busy}
          cancelBusy={cancelBusy}
          runId={runId}
          runStatus={runStatus}
          runList={runList}
          coordination={coordination}
          runActionStatus={runActionStatus}
          selectedRunWarnings={selectedRunWarnings}
          selectedRunIsActive={selectedRunIsActive}
          runCounters={runCounters}
          pendingPermissions={runStatus?.pending_permissions ?? []}
          runLogs={runLogs}
          artifacts={[...nonDiagramArtifacts, ...diagramArtifacts]}
          setupRuntime={runStatus?.runtime_mode ?? ""}
          setupRuntimeProvider={selectedRunProvider}
          runReviewSummary={runReviewSummary}
          runReviewStatus={runReviewStatus}
          gitDiff={gitDiff}
          gitDiffStatus={gitDiffStatus}
          onLoadGitDiff={handleLoadGitDiff}
          focusBlockerSignal={analysisFocusSignal}
          onRunPipeline={(pipeline, intent) => void handleRunPipeline(pipeline, intent)}
          onCancelSelectedRun={() => void handleCancelSelectedRun()}
          onCancelRun={(id) => void handleCancelRun(id)}
          onSelectRun={(id) => void handleSelectRunAndRoute(id)}
          onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)}
        />
      ) : null}

      {destination === "changes" && activeStage === "review" ? (
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
          runLogs={runLogs}
          reviewSummary={runReviewSummary}
          demo={runStatus?.runtime_mode === "fake"}
          gitDiff={gitDiff}
          gitDiffStatus={gitDiffStatus}
          onLoadGitDiff={handleLoadGitDiff}
          onSelectRun={(id) => void handleSelectRunAndRoute(id)}
          onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)}
        />
      ) : null}

      {destination === "changes" && activeStage === "proposals" ? (
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

      {destination === "changes" && activeStage === "publish" ? (
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
      {routeNotice ? <p className="status warn" role="status" data-testid="route-notice">{routeNotice}</p> : null}
      </ProductShell>
      <ModalDialog
        open={gitConfirmation !== null}
        title={gitConfirmation?.action === "branch" ? "Confirm proposal branch" : "Confirm workspace commit"}
        description="This action uses the complete workspace Git inventory shown below. If branch, HEAD, or any file changes before confirmation, ACP will reject it without a Git mutation."
        confirmLabel={gitConfirmation?.action === "branch" ? "Create proposal branch" : "Commit all workspace changes"}
        busy={busy}
        onCancel={cancelGitAction}
        onConfirm={() => void confirmGitAction()}
      >
        {gitConfirmation ? (
          <div className="git-confirmation" data-testid="git-confirmation-inventory">
            <dl className="compact-defs">
              <div><dt>Branch</dt><dd>{gitConfirmation.diff.branch}</dd></div>
              <div><dt>HEAD</dt><dd><code>{gitConfirmation.diff.head_oid ?? "unborn"}</code></dd></div>
              <div><dt>Base</dt><dd>{gitConfirmation.diff.base_ref} · <code>{gitConfirmation.diff.base_oid ?? "unborn"}</code></dd></div>
              <div><dt>Fingerprint</dt><dd><code>{gitConfirmation.diff.fingerprint}</code></dd></div>
            </dl>
            {gitConfirmation.diff.files.length === 0 ? <p>No workspace changes.</p> : (
              <ul>
                {gitConfirmation.diff.files.map((file) => (
                  <li key={`${file.status}:${file.original_path ?? ""}:${file.path}`}>
                    <strong>{file.status}</strong> <code>{file.path}</code>
                    {file.original_path ? <span> from <code>{file.original_path}</code></span> : null}
                  </li>
                ))}
              </ul>
            )}
          </div>
        ) : null}
      </ModalDialog>
    </>
  );
}

function HomePanel({ workflow, runStatus, evidenceStatus, gitChanges, onPrimaryAction }: {
  workflow: WorkflowState;
  runStatus: RunStatusResponse | null;
  evidenceStatus: string;
  gitChanges: number;
  onPrimaryAction: () => void;
}) {
  return (
    <section className="panel stage-panel home-panel" data-testid="home-panel">
      <div className="stage-header"><div><h1>Architecture workspace</h1><p className="hint">One current view of readiness, execution, evidence and publication.</p></div><span className={`status ${workflow.status === "blocked" ? "err" : workflow.status === "complete" ? "ok" : "warn"}`}>{workflow.status.replace("_", " ")}</span></div>
      <div className="home-summary-grid">
        <article><span className="metric-label">Latest run</span><strong>{runStatus ? `${runStatus.pipeline} · ${runStatus.status}` : "No runs"}</strong></article>
        <article><span className="metric-label">Evidence</span><strong>{evidenceStatus.replace("_", " ")}</strong></article>
        <article><span className="metric-label">Workspace changes</span><strong>{gitChanges}</strong></article>
      </div>
      <p>{workflow.attention}</p>
      <button type="button" onClick={onPrimaryAction}>{workflow.nextAction.label}</button>
    </section>
  );
}

function KnowledgePanel({ artifacts, evidenceStatus, onOpenArtifact }: { artifacts: Artifact[]; evidenceStatus: string; onOpenArtifact: (path: string) => void }) {
  return (
    <section className="panel stage-panel knowledge-panel" data-testid="knowledge-panel">
      <div className="stage-header"><div><h1>Knowledge</h1><p className="hint">Evidence-backed workspace documents and model artifacts from the selected snapshot.</p></div><span className="status">{evidenceStatus.replace("_", " ")}</span></div>
      {artifacts.length === 0 ? <p className="empty-state">No selected-run knowledge is available. Run Analysis or select a completed run.</p> : (
        <ul className="knowledge-list">
          {artifacts.map((artifact) => <li key={`${artifact.kind}:${artifact.path}`}><button type="button" onClick={() => onOpenArtifact(artifact.path)}><strong>{artifact.label || artifact.path}</strong><code>{artifact.path}</code><span>{artifact.kind}</span></button></li>)}
        </ul>
      )}
    </section>
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

export function formatPermissionBlockerDetail(request: RuntimePermissionRequest): string {
  const step = request.step_id || "runtime step";
  const decision = request.decision?.decision || "pending";
  const rule = request.decision?.rule_id ? ` via ${request.decision.rule_id}` : "";
  const target = request.path_or_command ? ` Target: ${request.path_or_command}.` : "";
  const reason = request.reason || request.decision?.message;
  const reasonDetail = reason ? ` Reason: ${reason}` : "";
  return `${step} paused for ${decision}${rule}.${target}${reasonDetail}`;
}

export function workspaceHealthSeverity(severity: string): InspectorItem["severity"] {
  switch (severity) {
    case "error":
      return "error";
    case "warning":
      return "warn";
    default:
      return "info";
  }
}

export function deriveNextAction(
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
