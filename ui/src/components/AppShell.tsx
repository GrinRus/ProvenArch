import type { ReactNode } from "react";

import { ActivityDrawer } from "./ActivityDrawer";
import { ActiveRunStrip } from "./ActiveRunStrip";
import { RightInspector } from "./RightInspector";
import { StageRail } from "./StageRail";
import { TopStatusBar } from "./TopStatusBar";
import type { InspectorItem, NextAction, StageId, StageOption } from "../lib/consoleTypes";
import type { RunLogEntry, RunReviewSummaryResponse, RunStatusResponse } from "../lib/appContracts";
import type { SystemInfoResponse } from "../lib/systemApi";

type AppShellProps = {
  buildVersion: string;
  buildCommit: string;
  buildBuilt: string;
  uiBundle: string;
  workspacePath: string;
  repoCount: number;
  runtimeMode: string;
  runtimeProvider: string;
  permissionMode: string;
  gitStatus: string;
  healthLabel: string;
  stages: StageOption[];
  activeStage: StageId;
  nextAction: NextAction;
  blockers: InspectorItem[];
  evidenceRefs: InspectorItem[];
  workspaceHealth: InspectorItem[];
  runtimeSafety: InspectorItem[];
  gitPublication: InspectorItem[];
  runStatus: RunStatusResponse | null;
  runReviewSummary: RunReviewSummaryResponse | null;
  runtimeLabel: string;
  cancelBusy: boolean;
  selectedRunIsActive: boolean;
  selectedRunId?: string;
  selectedRunStatus?: string;
  selectedRunError?: string;
  logs: RunLogEntry[];
  renderedLogs: string;
  runLogsStatus: string;
  runLogsMode: "events" | "raw" | "all";
  runLogsViewMode: "line" | "line+fields";
  canExportLogs: boolean;
  taskrunPaths: string[];
  children: ReactNode;
  onRefresh: () => void;
  onStageChange: (stage: StageId) => void;
  onPrimaryAction: () => void;
  onCancelRun: () => void;
  onOpenArtifact: (path: string) => void;
  onRunLogsModeChange: (value: "events" | "raw" | "all") => void;
  onRunLogsViewModeChange: (value: "line" | "line+fields") => void;
  onCopyRunLogs: () => void;
  onDownloadRunLogs: () => void;
};

export function AppShell({
  buildVersion,
  buildCommit,
  buildBuilt,
  uiBundle,
  workspacePath,
  repoCount,
  runtimeMode,
  runtimeProvider,
  permissionMode,
  gitStatus,
  healthLabel,
  stages,
  activeStage,
  nextAction,
  blockers,
  evidenceRefs,
  workspaceHealth,
  runtimeSafety,
  gitPublication,
  runStatus,
  runReviewSummary,
  runtimeLabel,
  cancelBusy,
  selectedRunIsActive,
  selectedRunId,
  selectedRunStatus,
  selectedRunError,
  logs,
  renderedLogs,
  runLogsStatus,
  runLogsMode,
  runLogsViewMode,
  canExportLogs,
  taskrunPaths,
  children,
  onRefresh,
  onStageChange,
  onPrimaryAction,
  onCancelRun,
  onOpenArtifact,
  onRunLogsModeChange,
  onRunLogsViewModeChange,
  onCopyRunLogs,
  onDownloadRunLogs,
}: AppShellProps) {
  return (
    <main className="console-shell" data-testid="console-shell">
      <TopStatusBar
        buildVersion={buildVersion}
        buildCommit={buildCommit}
        buildBuilt={buildBuilt}
        uiBundle={uiBundle}
        workspacePath={workspacePath}
        repoCount={repoCount}
        runtimeMode={runtimeMode}
        runtimeProvider={runtimeProvider}
        permissionMode={permissionMode}
        gitStatus={gitStatus}
        healthLabel={healthLabel}
        onRefresh={onRefresh}
      />
      <ActiveRunStrip
        runStatus={runStatus}
        reviewSummary={runReviewSummary}
        runtimeLabel={runtimeLabel}
        cancelBusy={cancelBusy}
        selectedRunIsActive={selectedRunIsActive}
        onCancel={onCancelRun}
        onOpenAnalysis={() => onStageChange("analysis")}
      />
      <div className="console-grid">
        <StageRail stages={stages} activeStage={activeStage} onStageChange={onStageChange} />
        <section className="work-area" data-testid="stage-content">
          {children}
        </section>
        <RightInspector
          nextAction={nextAction}
          blockers={blockers}
          evidenceRefs={evidenceRefs}
          workspaceHealth={workspaceHealth}
          runtimeSafety={runtimeSafety}
          gitPublication={gitPublication}
          onPrimaryAction={onPrimaryAction}
          onOpenArtifact={onOpenArtifact}
        />
      </div>
      <ActivityDrawer
        selectedRunId={selectedRunId}
        selectedRunStatus={selectedRunStatus}
        selectedRunError={selectedRunError}
        logs={logs}
        renderedLogs={renderedLogs}
        runLogsStatus={runLogsStatus}
        runLogsMode={runLogsMode}
        runLogsViewMode={runLogsViewMode}
        onRunLogsModeChange={onRunLogsModeChange}
        onRunLogsViewModeChange={onRunLogsViewModeChange}
        onCopyRunLogs={onCopyRunLogs}
        onDownloadRunLogs={onDownloadRunLogs}
        canExport={canExportLogs}
        taskrunPaths={taskrunPaths}
        onOpenArtifact={onOpenArtifact}
      />
    </main>
  );
}
