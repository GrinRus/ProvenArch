import { useEffect, useState } from "react";

import { BaselineEditorsPanel } from "./components/BaselineEditorsPanel";
import { BaselineGitPanel } from "./components/BaselineGitPanel";
import { ResultsPanels } from "./components/ResultsPanels";
import { RunPanels } from "./components/RunPanels";
import { RuntimeProfileSettingsPanel } from "./components/RuntimeProfileSettingsPanel";
import { SetupWorkspacePanel } from "./components/SetupWorkspacePanel";
import { TabNav, type TabOption } from "./components/TabNav";
import { WizardContractPanel } from "./components/WizardContractPanel";
import {
  runtimeExecutionLabels,
  runtimePermissionLabels,
  runtimeStepProviderLabels,
  runtimeStepProviderOrder,
  runtimeTimeoutKeys,
  runtimeTimeoutLabels,
  type GuidedRepo,
  type RuntimeExecutionKey,
  type RuntimePermissionKey,
  type RuntimeTimeoutKey,
} from "./lib/appContracts";
import { useRunExplorer } from "./hooks/useRunExplorer";
import { useRuntimeSettings } from "./hooks/useRuntimeSettings";
import { useWorkspaceSetup } from "./hooks/useWorkspaceSetup";
import { loadSystemDoctor } from "./lib/systemApi";

type TopTab = "setup" | "baseline" | "runs" | "results" | "settings";
type ResultsTab = "coverage" | "artifacts" | "diagrams";

const topTabOptions: Array<TabOption<TopTab>> = [
  { id: "setup", label: "Setup", testId: "tab-setup" },
  { id: "baseline", label: "Baseline", testId: "tab-baseline" },
  { id: "runs", label: "Runs", testId: "tab-runs" },
  { id: "results", label: "Results", testId: "tab-results" },
  { id: "settings", label: "Settings", testId: "tab-settings" },
];

const resultsTabOptions: Array<TabOption<ResultsTab>> = [
  { id: "coverage", label: "Coverage", testId: "results-tab-coverage" },
  { id: "artifacts", label: "Artifacts", testId: "results-tab-artifacts" },
  { id: "diagrams", label: "Diagrams", testId: "results-tab-diagrams" },
];

export default function App() {
  const [activeTab, setActiveTab] = useState<TopTab>("setup");
  const [resultsTab, setResultsTab] = useState<ResultsTab>("coverage");
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
    selectedEditorPath,
    selectedEditorContent,
    editorStatus,
    guidedRepos,
    guidedDocsImportsPath,
    wizardProjectName,
    wizardScope,
    wizardNfr,
    wizardRules,
    wizardStatus,
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

  async function handleSetupFirstRun() {
    setFirstRunStatus("");
    const started = await handleRunPipeline("init");
    if (started) {
      setFirstRunStatus("First analysis started. Results will update as the run finishes.");
      setActiveTab("results");
    }
  }

  return (
    <main className="shell">
      <section className="hero">
        <p className="eyebrow">ACP Beta Surface</p>
        <h1>Local-first architecture control plane</h1>
        <p className="lead">
          Validate workspace, tune settings, edit baseline prompts, run init/refresh pipelines, inspect logs, diagrams,
          and artifacts, then commit workspace updates.
        </p>
      </section>

      <TabNav value={activeTab} onChange={setActiveTab} options={topTabOptions} testId="top-tabs" />

      {activeTab === "setup" ? (
        <>
          <SetupWorkspacePanel
            busy={busy}
            guidedRepos={guidedRepos}
            guidedDocsImportsPath={guidedDocsImportsPath}
            manifestContent={manifestContent}
            validateResult={validateResult}
            validationDiagnosticsByRepo={validationDiagnosticsByRepo}
            doctorResult={setupDoctorResult}
            doctorStatus={setupDoctorStatus}
            firstRunStatus={firstRunStatus}
            setupRuntime={setupRuntime}
            setupRuntimeProvider={setupRuntimeProvider}
            onRepoChange={handleSetupRepoChange}
            onAddRepo={handleSetupAddRepo}
            onRemoveRepo={handleSetupRemoveRepo}
            onDocsImportsPathChange={handleSetupDocsImportsPathChange}
            onApplyGuidedWorkspaceSetup={handleSetupApplyGuidedWorkspaceSetup}
            onManifestChange={handleSetupManifestChange}
            onSaveManifest={() => void handleSaveManifest()}
            onValidateWorkspace={() => void handleValidateWorkspace()}
            onSetupRuntimeChange={handleSetupRuntimeChange}
            onSetupRuntimeProviderChange={handleSetupRuntimeProviderChange}
            onCheckDoctor={() => void handleSetupDoctorCheck()}
            onRunFirstAnalysis={() => void handleSetupFirstRun()}
          />

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
        </>
      ) : null}

      {activeTab === "settings" ? (
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
      ) : null}

      {activeTab === "baseline" ? (
        <>
          <BaselineEditorsPanel
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

          <BaselineGitPanel
            busy={busy}
            gitMessage={gitMessage}
            proposalBranch={proposalBranch}
            gitStatus={gitStatus}
            onGitMessageChange={setGitMessage}
            onProposalBranchChange={setProposalBranch}
            onCommit={() => void handleGitCommit()}
            onCreateProposalBranch={() => void handleCreateProposalBranch()}
          />
        </>
      ) : null}

      {activeTab === "runs" ? (
        <RunPanels
          model={{
            busy,
            cancelBusy,
            runId,
            runStatus,
            runList,
            runActionStatus,
            selectedRunWarnings,
            selectedRunIsActive,
            runCounters,
            runLogsMode,
            runLogsViewMode,
            filteredRunLogs,
            runLogsStatus,
            runLogTaskrunPaths,
            runLogsRendered,
            pendingPermissions: runStatus?.pending_permissions ?? [],
          }}
          actions={{
            onRunLogsModeChange: setRunLogsMode,
            onRunLogsViewModeChange: setRunLogsViewMode,
            onRunPipeline: (pipeline) => void handleRunPipeline(pipeline),
            onCancelSelectedRun: () => void handleCancelSelectedRun(),
            onSelectRun: (id) => void handleSelectRun(id),
            onCopyRunLogs: () => void handleCopyRunLogs(),
            onDownloadRunLogs: handleDownloadRunLogs,
            onOpenArtifact: (path) => void handleOpenArtifact(path),
          }}
        />
      ) : null}

      {activeTab === "results" ? (
        <ResultsPanels
          resultsTab={resultsTab}
          resultsTabOptions={resultsTabOptions}
          coverageSummary={coverageSummary}
          openQuestions={openQuestions}
          nonDiagramArtifacts={nonDiagramArtifacts}
          diagramArtifacts={diagramArtifacts}
          selectedArtifact={selectedArtifact}
          selectedArtifactContent={selectedArtifactContent}
          selectedArtifactIsMermaid={selectedArtifactIsMermaid}
          onResultsTabChange={setResultsTab}
          onOpenArtifact={(path) => void handleOpenArtifact(path)}
        />
      ) : null}

      {error ? <p className="status err">Error: {error}</p> : null}
    </main>
  );
}
