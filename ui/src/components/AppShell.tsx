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
  healthLabel: string;
  stages: StageOption[];
  activeStage: StageId;
  nextAction: NextAction;
  blockers: InspectorItem[];
  evidenceRefs: InspectorItem[];
  workspaceHealth: InspectorItem[];
  runtimeSafety: InspectorItem[];
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
  healthLabel,
  stages,
  activeStage,
  nextAction,
  blockers,
  evidenceRefs,
  workspaceHealth,
  runtimeSafety,
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
      <div className="compat-controls" aria-hidden="true">
        <button type="button" tabIndex={-1} data-testid="tab-setup" onClick={() => onStageChange("source")}>
          Setup
        </button>
        <button type="button" tabIndex={-1} data-testid="tab-baseline" onClick={() => onStageChange("charter")}>
          Baseline
        </button>
        <button type="button" tabIndex={-1} data-testid="tab-runs" onClick={() => onStageChange("analysis")}>
          Runs
        </button>
        <button type="button" tabIndex={-1} data-testid="tab-results" onClick={() => onStageChange("review")}>
          Results
        </button>
        <button type="button" tabIndex={-1} data-testid="tab-settings" onClick={() => onStageChange("readiness")}>
          Settings
        </button>
        <button type="button" tabIndex={-1} data-testid="results-tab-coverage" onClick={() => onStageChange("review")}>
          Coverage
        </button>
        <button type="button" tabIndex={-1} data-testid="results-tab-artifacts" onClick={() => onStageChange("review")}>
          Artifacts
        </button>
        <button type="button" tabIndex={-1} data-testid="results-tab-diagrams" onClick={() => onStageChange("review")}>
          Diagrams
        </button>
      </div>
      <TopStatusBar
        workspacePath={workspacePath}
        repoCount={repoCount}
        runtimeMode={runtimeMode}
        runtimeProvider={runtimeProvider}
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
