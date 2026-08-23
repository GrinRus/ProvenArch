import type { StageId } from "./consoleTypes";
import type { WorkflowDestination } from "./workflowState";

export type SetupStep = "workspace" | "sources" | "brief" | "runner" | "review";
export type SettingsSection = "workspace" | "sources" | "runners" | "runtime" | "git" | "diagnostics";
export type KnowledgeView = "documents" | "diagrams" | "model" | "findings" | "map" | "overview" | "catalog" | "flows" | "evidence" | "atlas" | "entities" | "artifacts";
export type ChangesView = "overview" | "evidence" | "findings" | "proposals" | "diff" | "publish";
export type TaskRouteView = "inbox" | "new" | "detail" | "attempt" | "studio" | "legacy";
export type TaskLifecycleFilter = "open" | "archived";
export type TaskFilters = {
  search?: string;
  lifecycle?: TaskLifecycleFilter;
  runner?: string;
  repository?: string;
  from?: string;
  to?: string;
};
export type RouteSource = "snapshot" | "current";
export type ViewerMode = "rendered" | "raw" | "diff";

export type AppRoute = {
  destination: WorkflowDestination;
  setupStep?: SetupStep;
  runId?: string;
  runRequested?: boolean;
  taskView?: TaskRouteView;
  taskId?: string;
  attemptId?: string;
  taskFilters?: TaskFilters;
  knowledgeView?: KnowledgeView;
  settingsSection?: SettingsSection;
  changesView?: ChangesView;
  source?: RouteSource;
  artifact?: string;
  entity?: string;
  mode?: ViewerMode;
  invalid: string[];
};

export const destinationPaths: Record<WorkflowDestination, string> = {
  setup: "/setup", tasks: "/tasks", knowledge: "/architecture", changes: "/changes", settings: "/settings",
};

const setupSteps = new Set<SetupStep>(["workspace", "sources", "brief", "runner", "review"]);
const settingsSections = new Set<SettingsSection>(["workspace", "sources", "runners", "runtime", "git", "diagnostics"]);
const knowledgeViews = new Set<KnowledgeView>(["documents", "diagrams", "model", "findings", "map", "overview", "catalog", "flows", "evidence"]);
const changesViews = new Set<ChangesView>(["overview", "evidence", "findings", "proposals", "diff", "publish"]);
const sources = new Set<RouteSource>(["snapshot", "current"]);
const viewerModes = new Set<ViewerMode>(["rendered", "raw", "diff"]);

export function parseAppRoute(location: Pick<Location, "pathname" | "search">, consoleReady: boolean): AppRoute {
  if (!consoleReady) return { destination: "setup", setupStep: "workspace", invalid: [] };
  const params = new URLSearchParams(location.search);
  const invalid: string[] = [];
  const segments = location.pathname.split("/").filter(Boolean).map((segment) => {
    try {
      return decodeURIComponent(segment);
    } catch {
      invalid.push("path");
      return segment;
    }
  });
  const legacyRunPath = segments[0] === "runs" && segments.length === 2 && !invalid.includes("path") && validTaskRouteId(segments[1]) ? segments[1] : undefined;
  const destination = segments[0] === "architecture" || segments[0] === "knowledge" ? "knowledge" : segments[0] && Object.prototype.hasOwnProperty.call(destinationPaths, segments[0])
    ? segments[0] as WorkflowDestination
    : "tasks";
  const route: AppRoute = { destination, invalid };

  if (legacyRunPath) {
    route.taskView = "legacy";
    route.runId = legacyRunPath;
    route.runRequested = true;
  } else if (segments[0] === "runs") {
    if (segments.length > 2 || (segments.length === 2 && !validTaskRouteId(segments[1]))) invalid.push("run");
    route.taskView = segments.length === 1 ? "legacy" : "inbox";
  }

  if (destination === "setup") route.setupStep = enumParam(params, "step", setupSteps, "workspace", invalid);
  if (destination === "tasks") {
    route.taskFilters = parseTaskFilters(params, invalid);
    if (legacyRunPath || segments[0] === "runs") return route;
    parseTaskRoute(segments, route, invalid);
  }
  if (destination === "knowledge") {
	const legacy = params.get("view") ?? segments[1];
	const migrated = legacy === "atlas" ? "map" : legacy === "entities" ? "catalog" : legacy === "artifacts" ? "evidence" : legacy;
	if (migrated && knowledgeViews.has(migrated as KnowledgeView)) route.knowledgeView = migrated as KnowledgeView;
	else { route.knowledgeView = "map"; if (legacy) invalid.push("view"); }
	route.source = enumParam(params, "source", new Set<RouteSource>(["current"]), "current", invalid);
	route.taskId = taskParam(params, invalid);
	route.entity = textParam(params, "entity");
	route.artifact = textParam(params, "artifact");
  }
  if (destination === "changes") {
    route.changesView = enumParam(params, "view", changesViews, "overview", invalid);
    route.source = enumParam(params, "source", sources, "snapshot", invalid);
    route.taskId = taskParam(params, invalid);
    route.attemptId = attemptParam(params, invalid);
    route.runId = textParam(params, "run");
    route.runRequested = params.has("run");
    route.artifact = textParam(params, "artifact");
    route.mode = enumParam(params, "mode", viewerModes, "rendered", invalid);
  }
  if (destination === "settings") route.settingsSection = enumParam(params, "section", settingsSections, "workspace", invalid);
  return route;
}

export function formatAppRoute(route: AppRoute): string {
  if (route.destination === "tasks") {
    const path = route.taskView === "legacy" ? (validTaskRouteId(route.runId) ? `/tasks/legacy/${encodeURIComponent(route.runId)}` : "/tasks/legacy")
      : route.taskView === "new" ? "/tasks/new"
      : route.taskView === "detail" && validTaskRouteId(route.taskId) ? `/tasks/${encodeURIComponent(route.taskId)}`
      : (route.taskView === "attempt" || route.taskView === "studio") && validTaskRouteId(route.taskId) && validTaskRouteId(route.attemptId) ? `/tasks/${encodeURIComponent(route.taskId)}/attempts/${encodeURIComponent(route.attemptId)}${route.taskView === "studio" ? "/studio" : ""}`
      : "/tasks";
    const params = new URLSearchParams();
    const filters = route.taskFilters;
    if (filters?.search) params.set("search", filters.search);
    if (filters?.lifecycle) params.set("lifecycle", filters.lifecycle);
    if (filters?.runner) params.set("runner", filters.runner);
    if (filters?.repository) params.set("repository", filters.repository);
    if (filters?.from) params.set("from", filters.from);
    if (filters?.to) params.set("to", filters.to);
    const query = params.toString();
    return `${path}${query ? `?${query}` : ""}`;
  }
  const params = new URLSearchParams();
  if (route.destination === "setup") params.set("step", route.setupStep ?? "workspace");
  if (route.destination === "knowledge") {
    params.set("view", route.knowledgeView ?? "map");
    params.set("source", "current");
    if (route.taskId) params.set("task", route.taskId);
    if (route.entity) params.set("entity", route.entity);
    if (route.artifact) params.set("artifact", route.artifact);
  }
  if (route.destination === "changes") {
    if (route.taskId) params.set("task", route.taskId);
    if (route.attemptId) params.set("attempt", route.attemptId);
    if (route.runId) params.set("run", route.runId);
    params.set("view", route.changesView ?? "overview");
    params.set("source", route.source ?? "snapshot");
    if (route.artifact) params.set("artifact", route.artifact);
    params.set("mode", route.mode ?? "rendered");
  }
  if (route.destination === "settings" && route.settingsSection && route.settingsSection !== "workspace") params.set("section", route.settingsSection);
  const query = params.toString();
  return `${destinationPaths[route.destination]}${query ? `?${query}` : ""}`;
}

export function destinationFromPath(pathname: string, consoleReady: boolean): WorkflowDestination {
  return parseAppRoute({ pathname, search: "" } as Location, consoleReady).destination;
}

export function destinationForStage(stage: StageId): WorkflowDestination {
  if (stage === "source" || stage === "readiness" || stage === "charter") return "setup";
  if (stage === "analysis" || stage === "ask") return "tasks";
  return "changes";
}

export function defaultStageForDestination(destination: WorkflowDestination): StageId {
  if (destination === "setup") return "source";
  if (destination === "tasks" || destination === "changes" || destination === "knowledge") return "review";
  return "source";
}

export function stageForRoute(route: AppRoute): StageId {
  if (route.destination === "setup") return route.setupStep === "runner" ? "readiness" : route.setupStep === "brief" || route.setupStep === "review" ? "charter" : "source";
  if (route.destination === "tasks" && route.taskView === "legacy") return "analysis";
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

function taskParam(params: URLSearchParams, invalid: string[]): string | undefined {
  const value = textParam(params, "task");
  if (!value || validTaskRouteId(value)) return value;
  invalid.push("task");
  return undefined;
}

function attemptParam(params: URLSearchParams, invalid: string[]): string | undefined {
  const value = textParam(params, "attempt");
  if (!value || validTaskRouteId(value)) return value;
  invalid.push("attempt");
  return undefined;
}

function parseTaskRoute(segments: string[], route: AppRoute, invalid: string[]): void {
  route.taskView = "inbox";
  // The service serves the console shell from `/`; treat that root as the
  // canonical Task Inbox instead of surfacing a spurious invalid-route state.
  if (segments.length === 0 || segments.length === 1) return;
  if (segments.length === 2 && segments[1] === "legacy") {
    route.taskView = "legacy";
    return;
  }
  if (segments.length === 3 && segments[1] === "legacy") {
    if (!validTaskRouteId(segments[2])) {
      invalid.push("run");
      return;
    }
    route.taskView = "legacy";
    route.runId = segments[2];
    route.runRequested = true;
    return;
  }
  if (segments.length === 2) {
    if (segments[1] === "new") {
      route.taskView = "new";
      return;
    }
    if (validTaskRouteId(segments[1])) {
      route.taskView = "detail";
      route.taskId = segments[1];
      return;
    }
    invalid.push("task");
    return;
  }
  if ((segments.length === 4 || segments.length === 5) && segments[2] === "attempts") {
    if (!validTaskRouteId(segments[1])) invalid.push("task");
    if (!validTaskRouteId(segments[3])) invalid.push("attempt");
    if (invalid.includes("task") || invalid.includes("attempt")) return;
    if (segments.length === 5 && segments[4] !== "studio") { invalid.push("task_route"); return; }
    route.taskView = segments.length === 5 ? "studio" : "attempt";
    route.taskId = segments[1];
    route.attemptId = segments[3];
    return;
  }
  invalid.push("task_route");
}

function parseTaskFilters(params: URLSearchParams, invalid: string[]): TaskFilters {
  const filters: TaskFilters = {};
  const search = params.get("search")?.trim();
  if (search) filters.search = search;
  const lifecycle = params.get("lifecycle")?.trim();
  if (lifecycle === "open" || lifecycle === "archived") filters.lifecycle = lifecycle;
  else if (lifecycle) invalid.push("lifecycle");
  for (const key of ["runner", "repository"] as const) {
    const value = params.get(key)?.trim();
    if (value) filters[key] = value;
  }
  for (const key of ["from", "to"] as const) {
    const value = params.get(key)?.trim();
    if (!value) continue;
    if (Number.isNaN(Date.parse(value))) invalid.push(key);
    else filters[key] = value;
  }
  if (filters.from && filters.to && Date.parse(filters.to) < Date.parse(filters.from)) {
    invalid.push("task_time_range");
    delete filters.from;
    delete filters.to;
  }
  return filters;
}

function validTaskRouteId(value: string | undefined): value is string {
  return Boolean(value && value === value.trim() && value !== "new" && value !== "." && value !== ".." && !/[\s/\\]/u.test(value));
}
