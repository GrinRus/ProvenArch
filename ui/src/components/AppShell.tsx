import type { ReactNode } from "react";

import { ActivityDrawer } from "./ActivityDrawer";
import { RightInspector } from "./RightInspector";
import { StageRail } from "./StageRail";
import { TopStatusBar } from "./TopStatusBar";
import type { InspectorItem, NextAction, StageId, StageOption } from "../lib/consoleTypes";
import type { RunLogEntry } from "../lib/appContracts";

type AppShellProps = {
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
  onOpenArtifact: (path: string) => void;
  onRunLogsModeChange: (value: "events" | "raw" | "all") => void;
  onRunLogsViewModeChange: (value: "line" | "line+fields") => void;
  onCopyRunLogs: () => void;
  onDownloadRunLogs: () => void;
};

export function AppShell({
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
  onOpenArtifact,
  onRunLogsModeChange,
  onRunLogsViewModeChange,
  onCopyRunLogs,
  onDownloadRunLogs,
}: AppShellProps) {
  return (
    <main className="console-shell" data-testid="console-shell">
      <TopStatusBar
        workspacePath={workspacePath}
        repoCount={repoCount}
        runtimeMode={runtimeMode}
        runtimeProvider={runtimeProvider}
        permissionMode={permissionMode}
        gitStatus={gitStatus}
        healthLabel={healthLabel}
        onRefresh={onRefresh}
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
