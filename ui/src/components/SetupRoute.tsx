import type { ReactNode } from "react";

import { CharterStagePanel, ReadinessStagePanel, SourceStagePanel } from "./StagePanels";
import { GuidedSetupPage, GuidedSetupReview } from "./ProductPages";
import { WizardContractPanel } from "./WizardContractPanel";
import type {
  Diagnostic,
  DoctorResponse,
  EditableArtifactOption,
  GuidedRepo,
  RuntimeExecutionValues,
  RuntimePermissionValues,
  RuntimeStepProviderValues,
  RuntimeTimeoutValues,
  ValidateResponse,
  WorkspaceHealthResponse,
} from "../lib/appContracts";
import type { SetupStep } from "../lib/appRoutes";

export type SetupRouteProps = {
  step: SetupStep;
  onStepChange: (step: SetupStep) => void;
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
  sourceRuntime: string;
  sourceRuntimeProvider: string;
  onRepoChange: (id: string, patch: Partial<GuidedRepo>) => void;
  onAddRepo: () => void;
  onRemoveRepo: (id: string) => void;
  onDocsImportsPathChange: (value: string) => void;
  onApplyGuidedWorkspaceSetup: () => void;
  onSaveGuidedWorkspaceSetup: () => void;
  onManifestChange: (value: string) => void;
  onSaveManifest: () => void;
  firstRunStatus: string;
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
  wizardProjectName: string;
  wizardScope: string;
  wizardNfr: string;
  wizardRules: string;
  gitStatus: string;
  proposalBranch: string;
  wizardStatus: string;
  onProjectNameChange: (value: string) => void;
  onScopeChange: (value: string) => void;
  onNfrChange: (value: string) => void;
  onRulesChange: (value: string) => void;
  onSaveWizardContract: () => void;
  baselineBundleWarnings: Diagnostic[];
  baselineEditorArtifacts: EditableArtifactOption[];
  selectedEditorPath: string;
  selectedEditorContent: string;
  editorStatus: string;
  onEditorSelectionChange: (path: string) => void;
  onEditorContentChange: (value: string) => void;
  onSaveEditor: () => void;
  wizardContractReady: boolean;
  onCreateTask: () => void;
  onStart: () => void;
};

export function SetupRoute({
  step,
  onStepChange,
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
  sourceRuntime,
  sourceRuntimeProvider,
  onRepoChange,
  onAddRepo,
  onRemoveRepo,
  onDocsImportsPathChange,
  onApplyGuidedWorkspaceSetup,
  onSaveGuidedWorkspaceSetup,
  onManifestChange,
  onSaveManifest,
  firstRunStatus,
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
  wizardProjectName,
  wizardScope,
  wizardNfr,
  wizardRules,
  gitStatus,
  proposalBranch,
  wizardStatus,
  onProjectNameChange,
  onScopeChange,
  onNfrChange,
  onRulesChange,
  onSaveWizardContract,
  baselineBundleWarnings,
  baselineEditorArtifacts,
  selectedEditorPath,
  selectedEditorContent,
  editorStatus,
  onEditorSelectionChange,
  onEditorContentChange,
  onSaveEditor,
  wizardContractReady,
  onCreateTask,
  onStart,
}: SetupRouteProps) {
  return (
    <GuidedSetupPage step={step} onStepChange={onStepChange}>
      {step === "workspace" || step === "sources" ? (
        <SourceStagePanel
          setupView={step}
          busy={busy}
          guidedRepos={guidedRepos}
          guidedDocsImportsPath={guidedDocsImportsPath}
          manifestContent={manifestContent}
          manifestStatus={manifestStatus}
          validateResult={validateResult}
          validationDiagnosticsByRepo={validationDiagnosticsByRepo}
          doctorResult={doctorResult}
          doctorStatus={doctorStatus}
          setupRuntime={sourceRuntime}
          setupRuntimeProvider={sourceRuntimeProvider}
          onRepoChange={onRepoChange}
          onAddRepo={onAddRepo}
          onRemoveRepo={onRemoveRepo}
          onDocsImportsPathChange={onDocsImportsPathChange}
          onApplyGuidedWorkspaceSetup={onApplyGuidedWorkspaceSetup}
          onSaveGuidedWorkspaceSetup={onSaveGuidedWorkspaceSetup}
          onManifestChange={onManifestChange}
          onSaveManifest={onSaveManifest}
        />
      ) : null}

      {step === "runner" ? (
        <ReadinessStagePanel
          busy={busy}
          validateResult={validateResult}
          validationDiagnosticsByRepo={validationDiagnosticsByRepo}
          doctorResult={doctorResult}
          doctorStatus={doctorStatus}
          firstRunStatus={firstRunStatus}
          setupRuntime={setupRuntime}
          setupRuntimeProvider={setupRuntimeProvider}
          selectedRunErrorCode={selectedRunErrorCode}
          selectedRunError={selectedRunError}
          onSetupRuntimeChange={onSetupRuntimeChange}
          onSetupRuntimeProviderChange={onSetupRuntimeProviderChange}
          onValidateWorkspace={onValidateWorkspace}
          onCheckDoctor={onCheckDoctor}
          onRunFirstAnalysis={onRunFirstAnalysis}
          runtimeSettingsPanel={runtimeSettingsPanel}
          artifactCount={artifactCount}
          workspaceHealthReport={workspaceHealthReport}
          workspaceHealthStatus={workspaceHealthStatus}
          workspaceHealthError={workspaceHealthError}
          onRefreshWorkspaceHealth={onRefreshWorkspaceHealth}
          runtimeTimeoutEffective={runtimeTimeoutEffective}
          runtimeExecutionEffective={runtimeExecutionEffective}
          runtimePermissionEffective={runtimePermissionEffective}
          runtimeStepProviderEffective={runtimeStepProviderEffective}
        />
      ) : null}

      {step === "brief" ? (
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
              onProjectNameChange={onProjectNameChange}
              onScopeChange={onScopeChange}
              onNfrChange={onNfrChange}
              onRulesChange={onRulesChange}
              onSave={onSaveWizardContract}
            />
          }
          busy={busy}
          baselineBundleWarnings={baselineBundleWarnings}
          baselineEditorArtifacts={baselineEditorArtifacts}
          selectedEditorPath={selectedEditorPath}
          selectedEditorContent={selectedEditorContent}
          editorStatus={editorStatus}
          onEditorSelectionChange={onEditorSelectionChange}
          onEditorContentChange={onEditorContentChange}
          onSave={onSaveEditor}
        />
      ) : null}

      {step === "review" ? <GuidedSetupReview briefReady={wizardContractReady} workspaceReady={validateResult?.ok === true} busy={busy} repoCount={guidedRepos.length} docsImportsReady={Boolean(guidedDocsImportsPath)} runtimeLabel={setupRuntimeProvider ? `${setupRuntime} · ${setupRuntimeProvider}` : setupRuntime} readinessLabel={doctorResult?.ok === true ? "Ready" : doctorResult ? "Needs attention" : "Not checked"} onBack={() => onStepChange("runner")} onCreateTask={onCreateTask} onStart={onStart} /> : null}
    </GuidedSetupPage>
  );
}
