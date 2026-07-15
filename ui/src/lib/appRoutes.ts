import type { StageId } from "./consoleTypes";
import type { WorkflowDestination } from "./workflowState";

export const destinationPaths: Record<WorkflowDestination, string> = {
  setup: "/setup",
  home: "/home",
  runs: "/runs",
  knowledge: "/knowledge",
  changes: "/changes",
};

export function destinationFromPath(pathname: string, consoleReady: boolean): WorkflowDestination {
  const entry = (Object.entries(destinationPaths) as Array<[WorkflowDestination, string]>).find(([, path]) => path === pathname);
  if (!consoleReady) {
    return "setup";
  }
  return entry?.[0] ?? "home";
}

export function destinationForStage(stage: StageId): WorkflowDestination {
  switch (stage) {
    case "source":
    case "readiness":
    case "charter":
      return "setup";
    case "analysis":
      return "runs";
    case "review":
    case "proposals":
    case "publish":
      return "changes";
    case "ask":
      return "home";
  }
}

export function defaultStageForDestination(destination: WorkflowDestination): StageId {
  switch (destination) {
    case "setup": return "source";
    case "runs": return "analysis";
    case "changes": return "review";
    case "knowledge": return "review";
    case "home": return "source";
  }
}
