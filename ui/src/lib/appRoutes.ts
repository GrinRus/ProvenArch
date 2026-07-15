import type { StageId } from "./consoleTypes";
import type { WorkflowDestination } from "./workflowState";

export type SetupStep = "workspace" | "sources" | "brief" | "runner" | "review";
export type KnowledgeView = "overview" | "atlas" | "entities" | "artifacts";
export type ChangesView = "overview" | "evidence" | "findings" | "proposals" | "diff" | "publish";
export type RouteSource = "snapshot" | "current";
export type ViewerMode = "rendered" | "raw" | "diff";

export type AppRoute = {
  destination: WorkflowDestination;
  setupStep?: SetupStep;
  runId?: string;
  runRequested?: boolean;
  knowledgeView?: KnowledgeView;
  changesView?: ChangesView;
  source?: RouteSource;
  artifact?: string;
  entity?: string;
  mode?: ViewerMode;
  invalid: string[];
};

export const destinationPaths: Record<WorkflowDestination, string> = {
  setup: "/setup", home: "/home", runs: "/runs", knowledge: "/knowledge", changes: "/changes",
};

const setupSteps = new Set<SetupStep>(["workspace", "sources", "brief", "runner", "review"]);
const knowledgeViews = new Set<KnowledgeView>(["overview", "atlas", "entities", "artifacts"]);
const changesViews = new Set<ChangesView>(["overview", "evidence", "findings", "proposals", "diff", "publish"]);
const sources = new Set<RouteSource>(["snapshot", "current"]);
const viewerModes = new Set<ViewerMode>(["rendered", "raw", "diff"]);

export function parseAppRoute(location: Pick<Location, "pathname" | "search">, consoleReady: boolean): AppRoute {
  if (!consoleReady) return { destination: "setup", setupStep: "workspace", invalid: [] };
  const params = new URLSearchParams(location.search);
  const segments = location.pathname.split("/").filter(Boolean).map(decodeURIComponent);
  const destination = segments[0] && Object.prototype.hasOwnProperty.call(destinationPaths, segments[0])
    ? segments[0] as WorkflowDestination
    : "home";
  const invalid: string[] = [];
  const route: AppRoute = { destination, invalid };

  if (destination === "setup") route.setupStep = enumParam(params, "step", setupSteps, "workspace", invalid);
  if (destination === "runs") {
    if (segments.length === 2 && segments[1].trim()) {
      route.runId = segments[1];
      route.runRequested = true;
    }
    else if (segments.length > 1) invalid.push("run");
  }
  if (destination === "knowledge") {
    route.knowledgeView = enumParam(params, "view", knowledgeViews, "overview", invalid);
    route.source = enumParam(params, "source", new Set<RouteSource>(["current"]), "current", invalid);
    route.entity = textParam(params, "entity");
  }
  if (destination === "changes") {
    route.changesView = enumParam(params, "view", changesViews, "overview", invalid);
    route.source = enumParam(params, "source", sources, "snapshot", invalid);
    route.runId = textParam(params, "run");
    route.runRequested = params.has("run");
    route.artifact = textParam(params, "artifact");
    route.mode = enumParam(params, "mode", viewerModes, "rendered", invalid);
  }
  return route;
}

export function formatAppRoute(route: AppRoute): string {
  if (route.destination === "home") return "/home";
  if (route.destination === "runs") return route.runId ? `/runs/${encodeURIComponent(route.runId)}` : "/runs";
  const params = new URLSearchParams();
  if (route.destination === "setup") params.set("step", route.setupStep ?? "workspace");
  if (route.destination === "knowledge") {
    params.set("view", route.knowledgeView ?? "overview");
    params.set("source", "current");
    if (route.entity) params.set("entity", route.entity);
  }
  if (route.destination === "changes") {
    if (route.runId) params.set("run", route.runId);
    params.set("view", route.changesView ?? "overview");
    params.set("source", route.source ?? "snapshot");
    if (route.artifact) params.set("artifact", route.artifact);
    params.set("mode", route.mode ?? "rendered");
  }
  const query = params.toString();
  return `${destinationPaths[route.destination]}${query ? `?${query}` : ""}`;
}

export function destinationFromPath(pathname: string, consoleReady: boolean): WorkflowDestination {
  return parseAppRoute({ pathname, search: "" } as Location, consoleReady).destination;
}

export function destinationForStage(stage: StageId): WorkflowDestination {
  if (stage === "source" || stage === "readiness" || stage === "charter") return "setup";
  if (stage === "analysis") return "runs";
  if (stage === "ask") return "home";
  return "changes";
}

export function defaultStageForDestination(destination: WorkflowDestination): StageId {
  if (destination === "setup") return "source";
  if (destination === "runs") return "analysis";
  if (destination === "changes" || destination === "knowledge") return "review";
  return "source";
}

export function stageForRoute(route: AppRoute): StageId {
  if (route.destination === "setup") return route.setupStep === "runner" ? "readiness" : route.setupStep === "brief" || route.setupStep === "review" ? "charter" : "source";
  if (route.destination === "runs") return "analysis";
  if (route.destination === "changes") return route.changesView === "publish" ? "publish" : route.changesView === "proposals" ? "proposals" : "review";
  return defaultStageForDestination(route.destination);
}

function enumParam<T extends string>(params: URLSearchParams, key: string, allowed: Set<T>, fallback: T, invalid: string[]): T {
  const value = params.get(key);
  if (!value) return fallback;
  if (allowed.has(value as T)) return value as T;
  invalid.push(key);
  return fallback;
}

function textParam(params: URLSearchParams, key: string): string | undefined {
  return params.get(key)?.trim() || undefined;
}
