import { useEffect, useMemo, useRef, useState } from "react";

import { AppShell } from "./components/AppShell";
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
  type Diagnostic,
  type GuidedRepo,
  type RuntimeExecutionKey,
  type RuntimePermissionKey,
  type RuntimeTimeoutKey,
} from "./lib/appContracts";
import type { InspectorItem, NextAction, StageId, StageOption, StageStatus } from "./lib/consoleTypes";
import { useRunExplorer } from "./hooks/useRunExplorer";
import { useRuntimeSettings } from "./hooks/useRuntimeSettings";
import { useWorkspaceSetup } from "./hooks/useWorkspaceSetup";
import { loadSystemDoctor } from "./lib/systemApi";

const stageLabels: Record<StageId, { label: string; description: string }> = {
  source: { label: "Source", description: "Repos & imports" },
  readiness: { label: "Readiness", description: "Validate & doctor" },
  charter: { label: "Charter", description: "Scope & rules" },
  analysis: { label: "Analysis", description: "Run pipeline" },
  review: { label: "Review", description: "Evidence & findings" },
  proposals: { label: "Proposals", description: "ADR/RFC drafts" },
  ask: { label: "Ask", description: "Agent-backed workspace Q&A" },
  publish: { label: "Publish", description: "Git workflow" },
};

const stageOrder: StageId[] = ["source", "readiness", "charter", "analysis", "review", "proposals", "ask", "publish"];

export default function App() {
  const [activeStage, setActiveStage] = useState<StageId>("source");
  const userSelectedStageRef = useRef(false);
  const autoOpenedStageRef = useRef(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [setupRuntime, setSetupRuntime] = useState("fake");
  const [setupRuntimeProvider, setSetupRuntimeProvider] = useState("claude-code");
  const [setupDoctorResult, setSetupDoctorResult] = useState<Awaited<ReturnType<typeof loadSystemDoctor>> | null>(null);
  const [setupDoctorStatus, setSetupDoctorStatus] = useState("");
  const [firstRunStatus, setFirstRunStatus] = useState("");

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
    filteredRunLogs,
    diagramArtifacts,
    nonDiagramArtifacts,
    selectedArtifactIsMermaid,
    selectedRunWarnings,
    selectedRunIsActive,
    runLogsRendered,
    bootstrapRuns,
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
    if (!wizardContractLoaded) {
      void loadWizardContract();
    }
    if (selectedEditorPath && selectedEditorLoadedPath !== selectedEditorPath) {
      void loadSelectedEditorContent(selectedEditorPath);
    }
  }, [activeStage, loadSelectedEditorContent, loadWizardContract, selectedEditorLoadedPath, selectedEditorPath, wizardContractLoaded]);

  async function bootstrapApp() {
    await bootstrapRuns();
    await bootstrapWorkspaceSetup();
    await loadRuntimeTimeouts();
    await loadRuntimeExecution();
    await loadRuntimePermissions();
    await loadRuntimeProfile();
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
    setActiveStage(stage);
  }

  const diagnostics = useMemo(() => [...(validateResult?.errors ?? []), ...(validateResult?.warnings ?? [])], [validateResult]);
  const validationErrors = useMemo(() => diagnostics.filter((diagnostic) => diagnostic.level === "error"), [diagnostics]);
  const doctorFailures = useMemo(() => setupDoctorResult?.checks.filter((check) => check.status === "fail") ?? [], [setupDoctorResult]);
  const doctorWarnings = useMemo(() => setupDoctorResult?.checks.filter((check) => check.status === "warn") ?? [], [setupDoctorResult]);
  const artifactCount = nonDiagramArtifacts.length + diagramArtifacts.length;
  const proposalArtifacts = useMemo(
    () => nonDiagramArtifacts.filter((artifact) => artifact.path.startsWith("proposals/") || artifact.path.startsWith("reports/changelog/")),
    [nonDiagramArtifacts],
  );

  useEffect(() => {
    if (userSelectedStageRef.current || autoOpenedStageRef.current) {
      return;
    }
    if (artifactCount > 0) {
      autoOpenedStageRef.current = true;
      setActiveStage("review");
      return;
    }
    if (selectedRunIsActive) {
      autoOpenedStageRef.current = true;
      setActiveStage("analysis");
    }
  }, [artifactCount, selectedRunIsActive]);

  const stageStatuses = useMemo<Record<StageId, StageStatus>>(() => {
    const readinessBlocked = validationErrors.length > 0 || doctorFailures.length > 0;
    const analysisBlocked = runStatus?.status === "failed" || (runStatus?.pending_permissions?.length ?? 0) > 0;
    return {
      source: manifestContent.trim() ? "done" : "active",
      readiness: readinessBlocked ? "blocked" : validateResult?.ok && setupDoctorResult?.ok ? "done" : activeStage === "readiness" ? "active" : "pending",
      charter: wizardProjectName.trim() || selectedEditorPath ? "done" : activeStage === "charter" ? "active" : "pending",
      analysis: analysisBlocked ? "blocked" : selectedRunIsActive ? "active" : runStatus?.status === "succeeded" ? "done" : activeStage === "analysis" ? "active" : "pending",
      review: artifactCount > 0 ? "done" : activeStage === "review" ? "active" : "pending",
      proposals: proposalArtifacts.length > 0 ? "done" : activeStage === "proposals" ? "active" : "pending",
      ask: activeStage === "ask" ? "active" : "pending",
      publish: gitStatus ? "done" : activeStage === "publish" ? "active" : "pending",
    };
  }, [
    activeStage,
    artifactCount,
    doctorFailures.length,
    gitStatus,
    manifestContent,
    proposalArtifacts.length,
    runStatus,
    selectedEditorPath,
    selectedRunIsActive,
    setupDoctorResult,
    validateResult,
    validationErrors.length,
    wizardProjectName,
  ]);

  const stages = useMemo<StageOption[]>(
    () =>
      stageOrder.map((id) => ({
        id,
        label: stageLabels[id].label,
        description: stageLabels[id].description,
        status: id === activeStage && stageStatuses[id] !== "blocked" ? "active" : stageStatuses[id],
        count: id === "review" && artifactCount > 0 ? artifactCount : id === "analysis" && runCounters.running > 0 ? runCounters.running : undefined,
        testId: `stage-${id}`,
      })),
    [activeStage, artifactCount, runCounters.running, stageStatuses],
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
        severity: "warn",
        label: request.action || "runtime permission",
        detail: "Runtime permission request is pending.",
      });
    }
    if (runStatus?.error_code) {
      items.push({
        severity: "error",
        label: runStatus.error_code,
        detail: runStatus.error || "Selected run failed.",
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
    return items;
  }, [doctorFailures, openQuestions, runStatus, validationErrors]);

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

  const workspaceHealth = useMemo<InspectorItem[]>(
    () => [
      {
        severity: validateResult?.ok ? "ok" : validateResult ? "error" : "info",
        label: "Workspace validation",
        detail: validateResult ? (validateResult.ok ? "workspace.yaml is valid" : "workspace.yaml has validation errors") : "not checked yet",
      },
      {
        severity: "info",
        label: "Repo sources",
        detail: `${validateResult?.resolved_repos?.length ?? guidedRepos.length} configured`,
      },
      {
        severity: doctorWarnings.length > 0 ? "warn" : setupDoctorResult?.ok ? "ok" : "info",
        label: "Local doctor",
        detail: setupDoctorResult ? setupDoctorResult.summary : "not checked yet",
      },
      {
        severity: "info",
        label: "Docs imports",
        detail: guidedDocsImportsPath || "./docs/imports",
      },
    ],
    [doctorWarnings.length, guidedDocsImportsPath, guidedRepos.length, setupDoctorResult, validateResult],
  );

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
      (runStatus?.error_code ? 1 : 0),
    reviewFindingsCount: openQuestions.trim() ? 1 : 0,
    releaseBlockersCount: runStatus?.error_code === "release_verdict_FAIL" ? 1 : 0,
  }), [activeStage, artifactCount, blockers.length, doctorFailures.length, openQuestions, proposalArtifacts.length, runStatus, setupDoctorResult, validateResult, validationErrors.length]);

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
        handleSetupApplyGuidedWorkspaceSetup();
        break;
      case "readiness":
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
        void handleRunPipeline(runStatus ? "refresh" : "init");
        break;
      case "review":
        setActiveStage("review");
        break;
      case "proposals":
        setActiveStage("proposals");
        break;
      case "ask":
        setActiveStage("ask");
        break;
      case "publish":
        void handleGitCommit();
        break;
    }
  }

  return (
    <AppShell
      workspacePath={validateResult?.workspace ?? workspaceRootPath ?? "bound workspace"}
      repoCount={validateResult?.resolved_repos?.length ?? guidedRepos.length}
      runtimeMode={setupRuntime}
      runtimeProvider={setupRuntimeProvider}
      healthLabel={setupDoctorResult?.ok ? "local ready" : validateResult?.ok ? "workspace valid" : "local connected"}
      stages={stages}
      activeStage={activeStage}
      nextAction={nextAction}
      blockers={blockers}
      evidenceRefs={evidenceRefs}
      workspaceHealth={workspaceHealth}
      runtimeSafety={runtimeSafety}
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
          onSetupRuntimeChange={handleSetupRuntimeChange}
          onSetupRuntimeProviderChange={handleSetupRuntimeProviderChange}
          onValidateWorkspace={() => void handleValidateWorkspace()}
          onCheckDoctor={() => void handleSetupDoctorCheck()}
          onRunFirstAnalysis={() => void handleSetupFirstRun("review")}
          runtimeSettingsPanel={runtimeSettingsPanel}
        />
      ) : null}

      {activeStage === "charter" ? (
        <CharterStagePanel
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
            <PublishStagePanel
              busy={busy}
              gitMessage={gitMessage}
              proposalBranch={proposalBranch}
              gitStatus={gitStatus}
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
          onRunPipeline={(pipeline) => void handleRunPipeline(pipeline)}
          onCancelSelectedRun={() => void handleCancelSelectedRun()}
          onSelectRun={(id) => void handleSelectRun(id)}
        />
      ) : null}

      {activeStage === "review" ? (
        <ReviewStagePanel
          coverageSummary={coverageSummary}
          openQuestions={openQuestions}
          nonDiagramArtifacts={nonDiagramArtifacts}
          diagramArtifacts={diagramArtifacts}
          selectedArtifact={selectedArtifact}
          selectedArtifactContent={selectedArtifactContent}
          selectedArtifactIsMermaid={selectedArtifactIsMermaid}
          onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)}
        />
      ) : null}

      {activeStage === "proposals" ? <ProposalsStagePanel artifacts={nonDiagramArtifacts} onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)} /> : null}

      {activeStage === "ask" ? <AskStagePanel onOpenArtifact={(path) => void handleOpenArtifactAndReview(path)} /> : null}

      {activeStage === "publish" ? (
        <PublishStagePanel
          busy={busy}
          gitMessage={gitMessage}
          proposalBranch={proposalBranch}
          gitStatus={gitStatus}
          onGitMessageChange={setGitMessage}
          onProposalBranchChange={setProposalBranch}
          onCommit={() => void handleGitCommit()}
          onCreateProposalBranch={() => void handleCreateProposalBranch()}
        />
      ) : null}

      {error ? <p className="status err">Error: {error}</p> : null}
    </AppShell>
  );
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
    reviewFindingsCount: number;
    releaseBlockersCount: number;
  },
): NextAction {
  if (state.blockersCount > 0 && activeStage !== "readiness") {
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
  switch (activeStage) {
    case "source":
      return {
        label: "Apply source form",
        description: "Render the guided source fields into workspace.yaml before validation.",
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
      return {
        label: state.hasArtifacts ? "Ask about evidence" : "Run analysis first",
        description: state.hasArtifacts ? "Use workspace-backed Q&A to inspect unresolved architecture context." : "No review artifacts are available yet.",
        primaryActionId: state.hasArtifacts ? "ask" : "analysis",
        disabledReason: state.hasArtifacts ? undefined : "No analysis artifacts yet.",
      };
    case "proposals":
      return {
        label: state.hasProposals ? "Publish changes" : "Review results first",
        description: state.hasProposals ? "Commit or branch the generated proposal package." : "No proposal artifacts are available yet.",
        primaryActionId: state.hasProposals ? "publish" : "review",
      };
    case "ask":
      return {
        label: "Ask workspace",
        description: "Submit an agent-backed question over existing workspace artifacts.",
        primaryActionId: "ask",
      };
    case "publish":
      return {
        label: "Commit workspace changes",
        description: "Create a Git commit for reviewed architecture workspace updates.",
        primaryActionId: "publish",
      };
  }
}
