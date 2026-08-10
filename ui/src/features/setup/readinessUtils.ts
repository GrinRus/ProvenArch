import type { WorkspaceHealthResponse } from "../../lib/appContracts";

export type WorkspaceHealthLoadStatus = "idle" | "loading" | "loaded" | "error";
export type WorkspaceHealthTone = "info" | "ok" | "warn" | "error";

export function workspaceHealthTone(report: WorkspaceHealthResponse | null, status: WorkspaceHealthLoadStatus): WorkspaceHealthTone {
  if (status === "error") {
    return "error";
  }
  if (!report || status === "loading") {
    return "info";
  }
  if (report.status === "fail") {
    return "error";
  }
  if (report.status === "warn") {
    return "warn";
  }
  return "ok";
}

export function workspaceHealthLabel(report: WorkspaceHealthResponse | null, status: WorkspaceHealthLoadStatus): string {
  if (status === "loading") {
    return "scanning";
  }
  if (status === "error") {
    return "scan failed";
  }
  if (!report) {
    return "not available";
  }
  return report.status;
}
