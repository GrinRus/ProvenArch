import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";

import { ProductShell } from "./components/ProductShell";
import { TaskRouteContainer } from "./components/TaskRouteContainer";
import { AppOverlays } from "./components/AppOverlays";
import { ChangesWorkspace } from "./features/changes/ChangesWorkspace";
import { HomePage, RunsPage } from "./components/ProductPages";
import { OnboardingShell } from "./components/OnboardingShell";
import { RuntimeProfileSettingsPanel } from "./components/RuntimeProfileSettingsPanel";
import { SettingsPage } from "./components/SettingsPage";
import { SetupRoute } from "./components/SetupRoute";
import { WizardContractPanel } from "./components/WizardContractPanel";
import {
  AnalysisStagePanel,
} from "./components/StagePanels";

const KnowledgePage = lazy(() => import("./components/KnowledgePage").then((module) => ({ default: module.KnowledgePage })));
import {
  runtimeExecutionLabels,
  runtimePermissionLabels,
  runtimeStepProviderLabels,
  runtimeStepProviderOrder,
  runtimeTimeoutKeys,
  runtimeTimeoutLabels,
  type GuidedRepo,
	type ArchitectureResponse,
  type KnowledgeResponse,
  type OnboardingStatusResponse,
  type RuntimeExecutionKey,
  type RuntimePermissionKey,
  type RuntimeTimeoutKey,
  type SystemVersionResponse,
  type WorkspaceHealthResponse,
} from "./lib/appContracts";
import type { StageId } from "./lib/consoleTypes";
import { destinationForStage, formatAppRoute, parseAppRoute, stageForRoute, type AppRoute, type ChangesView, type KnowledgeView, type SetupStep, type ViewerMode } from "./lib/appRoutes";
import type { LoadGitDiffOptions } from "./lib/gitDiffApi";
import { runtimeDisplayLabel } from "./lib/runtimeDisplay";
import { deriveAppWorkflowState, derivePublicationState, selectedRunIssueCopy } from "./lib/appDerived";
import { type WorkflowDestination } from "./lib/workflowState";
import { useRunExplorer } from "./hooks/useRunExplorer";
import { useRuntimeSettings } from "./hooks/useRuntimeSettings";
import { useWorkspaceSetup } from "./hooks/useWorkspaceSetup";
import { enterOnboardingConsole, forgetOnboardingRecentWorkspace, loadOnboardingStatus, selectOnboardingRuntime, selectOnboardingWorkspace } from "./lib/onboardingApi";
import { loadSystemDoctor, loadSystemVersion } from "./lib/systemApi";
import { architectureFromKnowledge, loadArchitectureAPI, loadArtifactText, loadKnowledgeAPI, loadWorkspaceHealthAPI } from "./lib/workspaceApi";
import type { QAProposalDraftResponse } from "./lib/qaApi";

export default function App() {
  const [route, setRoute] = useState<AppRoute>(() => parseAppRoute(window.location, true));
  const destination = route.destination;
  const setupStep = route.setupStep ?? "workspace";
  const [activeStage, setActiveStageState] = useState<StageId>(() => stageForRoute(parseAppRoute(window.location, true)));
  const [routeNotice, setRouteNotice] = useState("");
  const [askOpen, setAskOpen] = useState(false);
  const [askReturnRoute, setAskReturnRoute] = useState<AppRoute | null>(null);
  const [createdQAProposal, setCreatedQAProposal] = useState<QAProposalDraftResponse | null>(null);
  const [knowledge, setKnowledge] = useState<KnowledgeResponse | null>(null);
	const [architecture, setArchitecture] = useState<ArchitectureResponse | null>(null);
  const [knowledgeStatus, setKnowledgeStatus] = useState<"idle" | "loading" | "loaded" | "error">("idle");
  const [knowledgeError, setKnowledgeError] = useState("");
  const [currentArtifactPath, setCurrentArtifactPath] = useState("");
  const [currentArtifactContent, setCurrentArtifactContent] = useState("");
  const [briefSkipConfirmationOpen, setBriefSkipConfirmationOpen] = useState(false);
  const unsavedDraftRef = useRef(false);
  const restoredRouteRunRef = useRef<string | null>(null);
  const restoredArtifactRef = useRef<string | null>(null);
  const defaultChangesRunRef = useRef<string | null>(null);
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

  function handleDestinationChange(nextDestination: WorkflowDestination) {
    if (nextDestination === "changes" && runId) {
      navigateRoute({ destination: "changes", runId, runRequested: true, changesView: "overview", source: "snapshot", mode: "rendered", invalid: [] });
      return;
    }
    navigateDestination(nextDestination);
  }

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
    runtimeProviderModels,
    runtimeProviderModelDraft,
    runtimeProviderModelStatus,
    loadRuntimeTimeouts,
    loadRuntimeExecution,
    loadRuntimePermissions,
    loadRuntimeProfile,
    updateRuntimeTimeoutDraft,
    updateRuntimeExecutionDraft,
    updateRuntimePermissionDraft,
    updateRuntimeProviderModelDraft,
    handleSaveRuntimeTimeouts,
    handleResetRuntimeTimeouts,
    handleSaveRuntimeExecution,
    handleResetRuntimeExecution,
    handleSaveRuntimePermissions,
    handleResetRuntimePermissions,
    handleSaveRuntimeProviderModels,
    handleResetRuntimeProviderModels,
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
    wizardContractReady,
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
      if (nextRoute.invalid.length > 0) {
        const canonicalPath = formatAppRoute(nextRoute);
        window.history.replaceState({}, "", canonicalPath);
      }
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
		// bootstrapRuns selects the newest run for the initial console state. When
		// a deep link pins a historical run, restore that selection explicitly
		// after bootstrap so the default selection cannot win the race with the
		// route-driven effect.
		if (restoredRoute.runId) {
		  await handleSelectRun(restoredRoute.runId, { silentErrors: true });
		}
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
    void refreshKnowledge();
  }

  async function refreshKnowledge() {
    setKnowledgeStatus("loading");
    setKnowledgeError("");
	let architectureError: unknown;
    try {
	  const response = await loadArchitectureAPI();
	  setArchitecture(response);
	  try { setKnowledge(await loadKnowledgeAPI()); } catch { setKnowledge(null); }
      setKnowledgeStatus("loaded");
	  return null;
	} catch (requestError) {
	  architectureError = requestError;
	}
	try {
	  const response = await loadKnowledgeAPI();
	  setKnowledge(response);
	  setArchitecture(architectureFromKnowledge(response));
	  setKnowledgeStatus("loaded");
	  return response;
	} catch (requestError) {
      setKnowledge(null);
	  setArchitecture(null);
      setKnowledgeStatus("error");
	  const failure = architectureError ?? requestError;
	  setKnowledgeError(failure instanceof Error ? failure.message : "architecture failed to load");
      return null;
    }
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
      await startFirstRun("analysis");
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
    if (!wizardContractReady) {
      setBriefSkipConfirmationOpen(true);
      return;
    }
    await startFirstRun(nextStage);
  }

  async function startFirstRun(nextStage: StageId = "analysis") {
    setFirstRunStatus("");
    const startedRunID = await handleRunPipeline("init");
    if (startedRunID) {
      setFirstRunStatus("First analysis started. Results will update as the run finishes.");
      setActiveStage(nextStage);
      navigateRoute({ destination: "runs", runId: startedRunID, runRequested: true, invalid: [] });
    }
  }

  async function handleRunPipelineFromRuns(pipeline: "init" | "refresh", intent: "start" | "queue" = "start") {
    const startedRunID = await handleRunPipeline(pipeline, intent);
    if (startedRunID && intent !== "queue") {
      navigateRoute({ destination: "runs", runId: startedRunID, runRequested: true, invalid: [] });
    }
  }

  const handleSetupStepChange = useCallback((step: SetupStep) => {
    navigateRoute({ destination: "setup", setupStep: step, invalid: [] });
  }, [navigateRoute]);

  const handleOpenArtifactAndReview = useCallback(async (path: string) => {
    const openArtifact = handleOpenArtifact(path, route.mode ?? "rendered");
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
    await openArtifact;
  }, [diagramArtifacts, handleOpenArtifact, navigateRoute, nonDiagramArtifacts, route.changesView, route.destination, route.mode, runId]);

  const handleOpenCurrentArtifact = useCallback(async (path: string) => {
    const content = await loadArtifactText(path);
    if (content === null) {
      setRouteNotice(`Artifact ${path} is unavailable in the current workspace.`);
      return false;
    }
    setCurrentArtifactPath(path);
    setCurrentArtifactContent(content);
    navigateRoute({ destination: "changes", changesView: "evidence", source: "current", artifact: path, mode: route.mode ?? "rendered", invalid: [] });
    return true;
  }, [navigateRoute, route.mode]);

  const handleAskCitation = useCallback(async (path: string) => {
    setAskReturnRoute(route);
    setAskOpen(false);
    await handleOpenCurrentArtifact(path);
  }, [handleOpenCurrentArtifact, route]);

  const handleQAProposalCreated = useCallback(async (proposal: QAProposalDraftResponse) => {
    setCreatedQAProposal(proposal);
    setAskReturnRoute(route);
    setAskOpen(false);
    await loadGitDiff();
    const content = await loadArtifactText(proposal.proposal_path);
    if (content !== null) {
      setCurrentArtifactPath(proposal.proposal_path);
      setCurrentArtifactContent(content);
    }
    navigateRoute({
      destination: "changes",
      changesView: "proposals",
      source: "current",
      artifact: proposal.proposal_path,
      mode: "rendered",
      invalid: [],
    });
  }, [loadGitDiff, navigateRoute, route]);

  const handleOpenCreatedProposalArtifact = useCallback(async (path: string) => {
    const content = await loadArtifactText(path);
    if (content === null) {
      setRouteNotice(`Proposal artifact ${path} is unavailable in the current workspace.`);
      return;
    }
    setCurrentArtifactPath(path);
    setCurrentArtifactContent(content);
  }, []);

  async function handleSelectRunAndRoute(id: string) {
    restoredRouteRunRef.current = id;
    if (destination === "runs") navigateRoute({ destination: "runs", runId: id, runRequested: true, invalid: [] });
    else navigateRoute({ ...route, destination: "changes", runId: id, runRequested: true, source: "snapshot", artifact: undefined, invalid: [] });
    await handleSelectRun(id);
  }

  async function handleSelectRunInRuns(id: string) {
    restoredRouteRunRef.current = id;
    navigateRoute({ destination: "runs", runId: id, runRequested: true, invalid: [] });
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
    if (route.destination === "changes" && route.source !== "current" && !route.runRequested) {
      const latest = runList.find((item) => item.status === "succeeded" && (item.pipeline === "init" || item.pipeline === "refresh"));
      if (latest && defaultChangesRunRef.current !== latest.run_id) {
        defaultChangesRunRef.current = latest.run_id;
        navigateRoute({ ...route, runId: latest.run_id, runRequested: true, source: "snapshot", invalid: [] }, true);
        if (runId !== latest.run_id) void handleSelectRun(latest.run_id);
      }
    }
  }, [clearRunSelection, consoleReady, handleSelectRun, navigateRoute, route, runId, runList]);

  useEffect(() => {
    if (route.source === "current" || !route.artifact || evidenceSnapshot.status === "idle" || evidenceSnapshot.status === "loading") return;
    const match = [...nonDiagramArtifacts, ...diagramArtifacts].find((artifact) => artifact.id === route.artifact || artifact.path === route.artifact);
    if (!match) {
      setRouteNotice(`Artifact ${route.artifact} is unavailable in the selected source.`);
      navigateRoute({ ...route, artifact: undefined, invalid: [] }, true);
      return;
    }
    if (selectedArtifact !== match.path && restoredArtifactRef.current !== route.artifact) {
      restoredArtifactRef.current = route.artifact;
      void handleOpenArtifact(match.path, route.mode ?? "rendered");
    }
  }, [diagramArtifacts, evidenceSnapshot.status, handleOpenArtifact, navigateRoute, nonDiagramArtifacts, route, selectedArtifact]);

  useEffect(() => {
    if (!consoleReady || route.destination !== "knowledge" || knowledgeStatus === "loading") return;
    if (knowledgeStatus === "idle") {
      void refreshKnowledge();
      return;
    }
	const architectureHasEntity = Boolean(architecture && Object.values(architecture.views).some((view) => view.nodes.some((entity) => entity.id === route.entity) || view.edges.some((edge) => edge.id === route.entity)));
    if (route.entity && knowledgeStatus === "loaded" && !architectureHasEntity && !knowledge?.entities.some((entity) => entity.id === route.entity) && !knowledge?.edges.some((edge) => edge.id === route.entity)) {
      setRouteNotice(`Entity ${route.entity} is unavailable in the current workspace.`);
      navigateRoute({ ...route, entity: undefined, invalid: [] }, true);
    }
  }, [architecture, consoleReady, knowledge, knowledgeStatus, navigateRoute, route]);

  useEffect(() => {
    if (!consoleReady || route.destination !== "changes" || route.source !== "current" || !route.artifact || knowledgeStatus === "loading") return;
    if (knowledgeStatus === "idle") {
      void refreshKnowledge();
      return;
    }
	if (!architecture?.artifacts.some((artifact) => artifact.path === route.artifact) && !knowledge?.artifacts.some((artifact) => artifact.path === route.artifact)) {
      setRouteNotice(`Artifact ${route.artifact} is unavailable in the current workspace.`);
      setCurrentArtifactPath("");
      setCurrentArtifactContent("");
      navigateRoute({ ...route, artifact: undefined, invalid: [] }, true);
      return;
    }
    if (currentArtifactPath !== route.artifact) {
      void loadArtifactText(route.artifact).then((content) => {
        if (content === null) {
          setRouteNotice(`Artifact ${route.artifact} is unreadable in the current workspace.`);
          return;
        }
        setCurrentArtifactPath(route.artifact ?? "");
        setCurrentArtifactContent(content);
      });
    }
  }, [architecture, consoleReady, currentArtifactPath, knowledge, knowledgeStatus, navigateRoute, route]);

  useEffect(() => {
    if (!route.artifact) restoredArtifactRef.current = null;
  }, [route.artifact]);

  const diagnostics = useMemo(() => [...(validateResult?.errors ?? []), ...(validateResult?.warnings ?? [])], [validateResult]);
  const validationErrors = useMemo(() => diagnostics.filter((diagnostic) => diagnostic.level === "error"), [diagnostics]);
  const doctorFailures = useMemo(() => setupDoctorResult?.checks.filter((check) => check.status === "fail") ?? [], [setupDoctorResult]);
  const artifactCount = nonDiagramArtifacts.length + diagramArtifacts.length;
  useEffect(() => {
    if (!consoleReady || onboardingStatus?.can_enter_console !== true || destination !== "changes" || route.changesView !== "publish") {
      return;
    }
    void loadGitDiff({ runId: null });
  }, [consoleReady, destination, loadGitDiff, onboardingStatus?.can_enter_console, route.changesView]);

  useEffect(() => {
    if (!consoleReady || onboardingStatus?.can_enter_console !== true || (destination === "changes" && route.changesView === "publish")) {
      return;
    }
    void loadGitDiff({ runId, path: selectedArtifact || undefined });
  }, [consoleReady, destination, loadGitDiff, onboardingStatus?.can_enter_console, route.changesView, runId, selectedArtifact]);

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

  const publication = useMemo(() => derivePublicationState({ gitError, gitDiffStatus, gitDiff }), [gitDiff, gitDiffStatus, gitError]);
  const workflow = useMemo(() => deriveAppWorkflowState({
    workspace: validateResult?.ok ? "ready" : validateResult ? "invalid" : "unconfigured",
    runStatuses: runList.map((run) => run.status),
    selectedRunStatus: runStatus?.status,
    evidenceStatus: evidenceSnapshot.status,
    artifactCount,
    publication,
    openQuestions,
    demo: runStatus?.runtime_mode === "fake",
  }), [artifactCount, evidenceSnapshot.status, openQuestions, publication, runList, runStatus?.runtime_mode, runStatus?.status, validateResult]);

  const architectureComparisonMismatch = Boolean(
    runId
      && architecture?.comparison?.available
      && (!architecture.comparison.current_run_id || architecture.comparison.current_run_id !== runId),
  );
  const selectedArchitectureComparison = architectureComparisonMismatch ? undefined : architecture?.comparison;

  const handleHomePrimaryAction = useCallback(() => {
    const hasPromotedArchitecture = architecture?.status === "available" || architecture?.status === "partial";
    const promotedRunID = architecture?.authority.source_run_id?.trim();
    if (hasPromotedArchitecture && promotedRunID) {
      navigateRoute({
        destination: "changes",
        runId: promotedRunID,
        runRequested: true,
        changesView: "overview",
        source: "snapshot",
        mode: "rendered",
        invalid: [],
      });
      return;
    }
    if (hasPromotedArchitecture) {
      navigateRoute({ destination: "knowledge", knowledgeView: "documents", source: "current", invalid: [] });
      return;
    }
    handleDestinationChange(workflow.nextAction.destination);
  }, [architecture, handleDestinationChange, navigateRoute, workflow.nextAction.destination]);

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
      runtimeProviderModels={runtimeProviderModels}
      runtimeProviderModelDraft={runtimeProviderModelDraft}
      runtimeProviderModelStatus={runtimeProviderModelStatus}
      onSaveProviderModels={() => void handleSaveRuntimeProviderModels()}
      onResetProviderModels={() => void handleResetRuntimeProviderModels()}
      onProviderModelChange={(provider, field, value) => updateRuntimeProviderModelDraft(provider, field, value)}
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
        briefReady={wizardContractReady}
        briefPanel={
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
        onAsk={() => setAskOpen(true)}
        onDiagnostics={() => { handleDestinationChange("runs"); setActiveStageState("analysis"); }}
        onRefresh={() => void handleConsoleRefresh()}
      >
	  {destination === "tasks" ? <TaskRouteContainer view={route.taskView ?? "inbox"} taskId={route.taskId} attemptId={route.attemptId} invalid={route.invalid} /> : null}
	  {destination === "changes" ? (
		<ChangesWorkspace
		  view={route.changesView ?? "overview"}
		  source={route.source ?? "snapshot"}
		  page={{
			runs: runList,
			selectedRunID: runId,
			selectedEvidenceStatus: evidenceSnapshot.status,
			onViewChange: (view: ChangesView) => navigateRoute({ ...route, destination: "changes", changesView: view, invalid: [] }),
			onSelectChangeReview: (id: string) => { navigateRoute({ destination: "changes", runId: id, runRequested: true, changesView: "overview", source: "snapshot", mode: "rendered", invalid: [] }); void handleSelectRun(id); },
			onOpenRunStudio: (id: string) => { navigateRoute({ destination: "runs", runId: id, runRequested: true, invalid: [] }); void handleSelectRun(id); },
			architectureComparison: selectedArchitectureComparison,
			architectureComparisonMismatch,
			runReview: runReviewSummary?.review,
		  }}
		  review={{ runId, runStatus, runList, coverageSummary, openQuestions, nonDiagramArtifacts, diagramArtifacts, selectedArtifact, selectedArtifactContent, evidenceStatus: evidenceSnapshot.status, evidenceIssues: evidenceSnapshot.issues, reviewSummary: runReviewSummary, demo: runStatus?.runtime_mode === "fake", gitDiff, gitDiffStatus, onLoadGitDiff: handleLoadGitDiff, onSelectRun: (id) => void handleSelectRunAndRoute(id), onOpenArtifact: (path) => void handleOpenArtifactAndReview(path) }}
		  proposals={{
		    artifacts: [
		      ...nonDiagramArtifacts,
		      ...diagramArtifacts,
		      ...(createdQAProposal ? [
		        { id: createdQAProposal.proposal_path, path: createdQAProposal.proposal_path, kind: "proposal", label: "Ask proposal draft" },
		        { id: createdQAProposal.evidence_path, path: createdQAProposal.evidence_path, kind: "proposal-evidence", label: "Ask proposal evidence" },
		        { id: createdQAProposal.source_path, path: createdQAProposal.source_path, kind: "proposal-source", label: "Ask proposal source" },
		      ] : []),
		    ],
		    selectedArtifact: createdQAProposal && currentArtifactPath.startsWith(`${createdQAProposal.path}/`) ? currentArtifactPath : selectedArtifact,
		    selectedArtifactContent: createdQAProposal && currentArtifactPath.startsWith(`${createdQAProposal.path}/`) ? currentArtifactContent : selectedArtifactContent,
		    openQuestions,
		    proposalBranch,
		    gitStatus,
		    runLogs,
		    gitDiff,
		    gitDiffStatus,
		    onLoadGitDiff: handleLoadGitDiff,
		    onOpenArtifact: (path) => void (createdQAProposal && path.startsWith(`${createdQAProposal.path}/`) ? handleOpenCreatedProposalArtifact(path) : handleOpenArtifactAndReview(path)),
		    onGoPublish: () => navigateRoute({ ...route, destination: "changes", changesView: "publish", invalid: [] }),
		  }}
		  publish={{ busy, gitMessage, proposalBranch, gitStatus, gitError, artifacts: [...nonDiagramArtifacts, ...diagramArtifacts], selectedArtifact, selectedArtifactContent, openQuestions, externalGateItems: publishExternalGateItems, gitDiff, gitDiffStatus, onLoadGitDiff: handleLoadGitDiff, onGitMessageChange: setGitMessage, onProposalBranchChange: setProposalBranch, onCommit: () => void handleGitCommit(), onCreateProposalBranch: () => void handleCreateProposalBranch(), onPreviewArtifact: (path) => void handleOpenArtifact(path, route.mode ?? "rendered") }}
		  currentArtifact={currentArtifactPath ? { path: currentArtifactPath, content: currentArtifactContent } : null}
		  viewerMode={route.mode ?? "rendered"}
		  askReturnAvailable={askReturnRoute !== null}
		  onViewerModeChange={(mode: ViewerMode) => navigateRoute({ ...route, mode, invalid: [] })}
		  onOpenCurrentArtifact={(path) => void handleOpenCurrentArtifact(path)}
		  onReturnToAsk={() => { if (askReturnRoute) navigateRoute(askReturnRoute); setAskOpen(true); setAskReturnRoute(null); }}
		/>
	  ) : null}
	  {destination === "home" ? (
		<HomePage workflow={workflow} workspaceReady={validateResult?.ok === true} coordination={coordination} runStatus={runStatus} evidenceStatus={evidenceSnapshot.status} gitChanges={gitDiff?.files?.length ?? 0} architecture={architecture} onPrimaryAction={handleHomePrimaryAction} onOpenArchitecture={() => navigateRoute({ destination: "knowledge", knowledgeView: "documents", source: "current", invalid: [] })} />
	  ) : null}
	  {destination === "knowledge" ? (
		<Suspense fallback={<section className="panel stage-panel"><p className="status info">Loading Architecture Explorer…</p></section>}><KnowledgePage
		  architecture={architecture}
		  knowledge={knowledge}
		  workspaceHealth={workspaceHealthReport}
		  loading={knowledgeStatus === "loading" || knowledgeStatus === "idle"}
		  error={knowledgeError}
		  view={route.knowledgeView ?? "documents"}
		  selectedEntityID={route.entity}
		  selectedArtifactPath={route.artifact}
		  onViewChange={(view: KnowledgeView) => navigateRoute({ ...route, destination: "knowledge", knowledgeView: view, source: "current", invalid: [] })}
		  onEntityChange={(entity) => navigateRoute({ ...route, destination: "knowledge", knowledgeView: route.knowledgeView ?? "model", source: "current", entity, invalid: [] })}
		  onDocumentChange={(artifact) => navigateRoute({ ...route, destination: "knowledge", knowledgeView: route.knowledgeView ?? "documents", source: "current", artifact, invalid: [] })}
		  onOpenArtifact={(path) => void handleOpenCurrentArtifact(path)}
		  onOpenRuns={() => handleDestinationChange("runs")}
		/></Suspense>
	  ) : null}
	  {destination === "settings" ? (
	    <SettingsPage
	      workspacePath={validateResult?.workspace ?? workspaceRootPath ?? "bound workspace"}
	      workspaceValid={validateResult?.ok === true}
	      runtimeLabel={runtimeLabel}
	      runtimeSettingsPanel={runtimeSettingsPanel}
	      onOpenSetup={() => handleDestinationChange("setup")}
	      onOpenChanges={() => handleDestinationChange("changes")}
	      onOpenRuns={() => handleDestinationChange("runs")}
	    />
	  ) : null}

      {destination === "setup" ? (
        <SetupRoute
          step={setupStep}
          onStepChange={handleSetupStepChange}
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
          sourceRuntime={effectiveRuntimeMode}
          sourceRuntimeProvider={effectiveRuntimeProvider}
          onRepoChange={handleSetupRepoChange}
          onAddRepo={handleSetupAddRepo}
          onRemoveRepo={handleSetupRemoveRepo}
          onDocsImportsPathChange={handleSetupDocsImportsPathChange}
          onApplyGuidedWorkspaceSetup={handleSetupApplyGuidedWorkspaceSetup}
          onSaveGuidedWorkspaceSetup={() => void handleSetupSaveGuidedWorkspaceSetup()}
          onManifestChange={handleSetupManifestChange}
          onSaveManifest={() => void handleSaveManifest()}
          firstRunStatus={firstRunStatus}
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
          wizardProjectName={wizardProjectName}
          wizardScope={wizardScope}
          wizardNfr={wizardNfr}
          wizardRules={wizardRules}
          gitStatus={gitStatus}
          proposalBranch={proposalBranch}
          wizardStatus={wizardStatus}
          onProjectNameChange={setWizardProjectName}
          onScopeChange={setWizardScope}
          onNfrChange={setWizardNfr}
          onRulesChange={setWizardRules}
          onSaveWizardContract={() => void handleSaveStep0WizardContract()}
          baselineBundleWarnings={baselineBundleWarnings}
          baselineEditorArtifacts={baselineEditorArtifacts}
          selectedEditorPath={selectedEditorPath}
          selectedEditorContent={selectedEditorContent}
          editorStatus={editorStatus}
          onEditorSelectionChange={(path) => void handleEditorSelectionChange(path)}
          onEditorContentChange={setSelectedEditorContent}
          onSaveEditor={() => void handleSaveSelectedEditorArtifact()}
          wizardContractReady={wizardContractReady}
          onStart={() => void handleSetupFirstRun("analysis")}
        />
      ) : null}

      {destination === "runs" && activeStage === "analysis" ? (
        <RunsPage coordination={coordination} selectedRunID={route.runId}>
        <AnalysisStagePanel
          detailMode={Boolean(route.runId)}
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
          onRunPipeline={(pipeline, intent) => void handleRunPipelineFromRuns(pipeline, intent)}
          onCancelSelectedRun={() => void handleCancelSelectedRun()}
          onCancelRun={(id) => void handleCancelRun(id)}
          onSelectRun={(id) => void handleSelectRunInRuns(id)}
          onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)}
		  onOpenArchitecture={() => navigateRoute({ destination: "knowledge", knowledgeView: "documents", source: "current", invalid: [] })}
        />
        </RunsPage>
      ) : null}

      {error ? <p className="status err">Error: {error}</p> : null}
      {routeNotice ? <p className="status warn" role="status" data-testid="route-notice">{routeNotice}</p> : null}
      </ProductShell>
      <AppOverlays
        askOpen={askOpen}
        gitConfirmation={gitConfirmation}
        briefSkipConfirmationOpen={briefSkipConfirmationOpen}
        busy={busy}
        onCloseAsk={() => setAskOpen(false)}
        onOpenAskCitation={(path) => void handleAskCitation(path)}
        onProposalCreated={(proposal) => void handleQAProposalCreated(proposal)}
        onCancelGitAction={cancelGitAction}
        onConfirmGitAction={() => void confirmGitAction()}
        onCancelBriefSkip={() => setBriefSkipConfirmationOpen(false)}
        onConfirmBriefSkip={() => { setBriefSkipConfirmationOpen(false); void startFirstRun("analysis"); }}
      />
    </>
  );
}
