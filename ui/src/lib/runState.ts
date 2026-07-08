export const finalStatuses = new Set(["succeeded", "failed"]);
export const activeStatuses = new Set(["queued", "running"]);
export const runLogsPageLimit = 200;

export type RunStatusLike = {
  status: string;
  error_code?: string | null;
  warnings?: string[] | null;
};

export type RunOutcomeTone = "info" | "ok" | "warn" | "error";

export function dedupeArtifactsByPath<T extends { path: string }>(items: T[]): T[] {
  const deduped = new Map<string, T>();
  for (const artifact of items) {
    const key = artifact.path.trim();
    if (!key) {
      continue;
    }
    deduped.set(key, { ...artifact, path: key });
  }
  return Array.from(deduped.values()).sort((left, right) => left.path.localeCompare(right.path));
}

export function indexArtifactPath<T extends { path: string }>(artifacts: T[], suffix: string): string | null {
  const match = artifacts.find((artifact) => artifact.path.endsWith(suffix));
  return match ? match.path : null;
}

export function splitListInput(input: string): string[] {
  return input
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}

export function formatTimestamp(value?: string | null): string {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toISOString().replace("T", " ").replace(".000Z", " UTC");
}

export function parseTimeOrMin(value?: string | null): number {
  if (!value) {
    return Number.NEGATIVE_INFINITY;
  }
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return Number.NEGATIVE_INFINITY;
  }
  return parsed;
}

export function pickBootstrapRun<T extends { status: string; started_at: string }>(items: T[]): T | null {
  if (!Array.isArray(items) || items.length === 0) {
    return null;
  }
  let newestActive: T | null = null;
  let newestActiveStartedAt = Number.NEGATIVE_INFINITY;
  for (const item of items) {
    if (!activeStatuses.has(item.status)) {
      continue;
    }
    const startedAt = parseTimeOrMin(item.started_at);
    if (newestActive === null || startedAt > newestActiveStartedAt) {
      newestActive = item;
      newestActiveStartedAt = startedAt;
    }
  }
  if (newestActive) {
    return newestActive;
  }
  let newestTerminal: T | null = null;
  let newestTerminalStartedAt = Number.NEGATIVE_INFINITY;
  for (const item of items) {
    const startedAt = parseTimeOrMin(item.started_at);
    if (newestTerminal === null || startedAt > newestTerminalStartedAt) {
      newestTerminal = item;
      newestTerminalStartedAt = startedAt;
    }
  }
  return newestTerminal;
}

export function reconcileSelectedRunID<T extends { run_id: string; status: string; started_at: string }>(
  selectedRunID: string | null,
  items: T[],
): string | null {
  if (selectedRunID) {
    const selectedRunStillExists = items.some((item) => item.run_id === selectedRunID);
    if (selectedRunStillExists) {
      return selectedRunID;
    }
  }
  return pickBootstrapRun(items)?.run_id ?? null;
}

export function deriveRunLifecycleState(runStatus: RunStatusLike | null): "active" | "incomplete" | "recovered" | "terminal" | null {
  if (!runStatus) {
    return null;
  }
  if (activeStatuses.has(runStatus.status)) {
    return "active";
  }
  const errorCode = normalizeRunErrorCode(runStatus.error_code);
  const warnings = (runStatus.warnings ?? []).map((warning) => warning.toLowerCase());
  if (isRunReconciledAfterRestart(errorCode) || warnings.some((warning) => warning.includes("run_reconciled_after_restart"))) {
    return "recovered";
  }
  if (errorCode.includes("incomplete") || warnings.some((warning) => warning.includes("incomplete_cycle"))) {
    return "incomplete";
  }
  return "terminal";
}

export function normalizeRunErrorCode(errorCode?: string | null): string {
  return String(errorCode ?? "").trim().toLowerCase();
}

export function runOutcomeLabel(runStatus: Pick<RunStatusLike, "status" | "error_code"> | null | undefined, fallback = "idle"): string {
  if (!runStatus?.status) {
    return fallback;
  }
  const errorCode = normalizeRunErrorCode(runStatus.error_code);
  if (runStatus.status === "failed" && isRunCanceled(errorCode)) {
    return "canceled";
  }
  if (runStatus.status === "failed" && isRunReconciledAfterRestart(errorCode)) {
    return "recovered";
  }
  return runStatus.status;
}

export function runOutcomeTone(runStatus: Pick<RunStatusLike, "status" | "error_code"> | null | undefined): RunOutcomeTone {
  const outcome = runOutcomeLabel(runStatus);
  if (outcome === "succeeded") {
    return "ok";
  }
  if (outcome === "failed") {
    return "error";
  }
  if (outcome === "queued" || outcome === "running" || outcome === "canceled" || outcome === "recovered") {
    return "warn";
  }
  return "info";
}

export function isRunCanceled(errorCode?: string | null): boolean {
  return normalizeRunErrorCode(errorCode) === "run_canceled";
}

export function isRunReconciledAfterRestart(errorCode?: string | null): boolean {
  return normalizeRunErrorCode(errorCode).includes("run_reconciled_after_restart");
}
