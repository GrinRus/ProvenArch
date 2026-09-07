import { lazy, Suspense, type ComponentProps } from "react";

import { ChangesWorkspace } from "../features/changes/ChangesWorkspace";
import { AnalysisStagePanel } from "./StagePanels";
import { LegacyRunPage } from "./LegacyRunPage";
import { ProductShell } from "./ProductShell";
import { SettingsPage } from "./SettingsPage";
import { SetupRoute, type SetupRouteProps } from "./SetupRoute";
import { TaskComposer } from "./TaskComposer";
import { TaskRouteContainer } from "./TaskRouteContainer";
import type { SystemVersionResponse } from "../lib/appContracts";
import type { AppRoute } from "../lib/appRoutes";
import type { StageId } from "../lib/consoleTypes";
import type { WorkflowState, WorkflowDestination } from "../lib/workflowState";

const KnowledgePage = lazy(() => import("./KnowledgePage").then((module) => ({ default: module.KnowledgePage })));

export type AppConsoleRouteData = {
  taskComposer: ComponentProps<typeof TaskComposer>;
  taskRoute: ComponentProps<typeof TaskRouteContainer>;
  changes: ComponentProps<typeof ChangesWorkspace>;
  knowledge: ComponentProps<typeof KnowledgePage>;
  settings: ComponentProps<typeof SettingsPage>;
  setup: SetupRouteProps;
  analysis: ComponentProps<typeof AnalysisStagePanel>;
};

export type ConsoleRouteKind =
  | "task-composer"
  | "tasks"
  | "changes"
  | "knowledge"
  | "settings"
  | "setup"
  | "legacy-analysis"
  | "empty";

export function getConsoleRouteKind(route: AppRoute, activeStage: StageId): ConsoleRouteKind {
  if (route.destination === "tasks") {
    if (route.taskView === "new") return "task-composer";
    if (route.taskView === "legacy") return activeStage === "analysis" ? "legacy-analysis" : "empty";
    return "tasks";
  }
  return route.destination;
}

export type AppConsoleViewProps = {
  route: AppRoute;
  activeStage: StageId;
  workflow: WorkflowState;
  destination: WorkflowDestination;
  workspacePath: string;
  runtimeLabel: string;
  systemVersion: SystemVersionResponse;
  workspaceValid: boolean;
  error: string | null;
  routeNotice: string;
  routeData: AppConsoleRouteData;
  onDestinationChange: (destination: WorkflowDestination) => void;
  onAsk: () => void;
  onDiagnostics: () => void;
  onRefresh: () => void;
};

export function AppConsoleView({
  route,
  activeStage,
  workflow,
  destination,
  workspacePath,
  runtimeLabel,
  systemVersion,
  workspaceValid,
  error,
  routeNotice,
  routeData,
  onDestinationChange,
  onAsk,
  onDiagnostics,
  onRefresh,
}: AppConsoleViewProps) {
  const routeKind = getConsoleRouteKind(route, activeStage);
  return (
    <ProductShell
      destination={destination}
      workflow={workflow}
      workspacePath={workspacePath}
      runtimeLabel={runtimeLabel}
      buildLabel={`${systemVersion.version} · ${systemVersion.commit}`}
      buildTitle={`version=${systemVersion.version}; commit=${systemVersion.commit}; built=${systemVersion.built}`}
      workspaceValid={workspaceValid}
      onDestinationChange={onDestinationChange}
      onAsk={onAsk}
      onDiagnostics={onDiagnostics}
      onRefresh={onRefresh}
    >
      {routeKind === "task-composer" ? <TaskComposer {...routeData.taskComposer} /> : null}
      {routeKind === "tasks" ? <TaskRouteContainer {...routeData.taskRoute} /> : null}
      {routeKind === "changes" ? <ChangesWorkspace {...routeData.changes} /> : null}
      {routeKind === "knowledge" ? (
        <Suspense fallback={<section className="panel stage-panel"><p className="status info">Loading Architecture Explorer…</p></section>}>
          <KnowledgePage {...routeData.knowledge} />
        </Suspense>
      ) : null}
      {routeKind === "settings" ? <SettingsPage {...routeData.settings} /> : null}
      {routeKind === "setup" ? <SetupRoute {...routeData.setup} /> : null}
      {routeKind === "legacy-analysis" ? (
        <LegacyRunPage coordination={routeData.analysis.coordination} selectedRunID={route.runId}>
          <AnalysisStagePanel {...routeData.analysis} />
        </LegacyRunPage>
      ) : null}
      {error ? <p className="status err">Error: {error}</p> : null}
      {routeNotice ? <p className="status warn" role="status" data-testid="route-notice">{routeNotice}</p> : null}
    </ProductShell>
  );
}
