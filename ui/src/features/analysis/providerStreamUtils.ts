import type { RunLogEntry } from "../../lib/appContracts";
import { firstNonEmpty } from "./analysisUtils";

export type ProviderStreamSummary = {
  chunks: number;
  streamEvents: number;
  stdout: number;
  stderr: number;
  characters: number;
  signalTypes: string[];
};

export function summarizeProviderStream(runLogs: RunLogEntry[]): ProviderStreamSummary {
  const signalTypes = new Set<string>();
  let chunks = 0;
  let streamEvents = 0;
  let stdout = 0;
  let stderr = 0;
  let characters = 0;

  for (const entry of runLogs) {
    if (entry.kind !== "runtime_output") {
      continue;
    }
    chunks += 1;
    characters += entry.message.length;
    if (entry.stream === "stderr") {
      stderr += 1;
    } else {
      stdout += 1;
    }
    const parsed = parseRuntimeOutputJSON(entry.message);
    if (!parsed) {
      continue;
    }
    const topType = objectString(parsed, "type");
    const event = objectField(parsed, "event");
    const eventType = objectString(event, "type");
    const delta = objectField(event, "delta") ?? objectField(parsed, "delta");
    const deltaType = objectString(delta, "type");
    if (topType === "stream_event" || eventType || deltaType) {
      streamEvents += 1;
    }
    const signal = firstNonEmpty([deltaType, eventType, topType]);
    if (signal) {
      signalTypes.add(signal);
    }
  }

  return {
    chunks,
    streamEvents,
    stdout,
    stderr,
    characters,
    signalTypes: Array.from(signalTypes).slice(0, 3),
  };
}

export function artifactHandoffStalled(normalizedMessage: string, normalizedValidationError: string, repairStage: string): boolean {
  const normalizedStage = repairStage.toLowerCase();
  const hasStallWord = /\bstall(?:ed|s|ing)?\b/.test(normalizedMessage);
  return (
    normalizedValidationError.includes("runtime_stalled_before_artifacts") ||
    normalizedMessage.includes("stalled before valid artifacts") ||
    normalizedMessage.includes("before valid artifacts were available") ||
    (normalizedStage.includes("collect_pair_repair") && hasStallWord)
  );
}

export function stageFromMessage(message: string): string {
  const match = message.match(/\bstage=([^\s)]+)/i);
  return match?.[1] ?? "";
}

export function formatShardMetric(plannedShards: number | undefined, observedCount: number, succeededCount: number, failedCount: number): string {
  if (plannedShards !== undefined && plannedShards > 0) {
    return `${succeededCount}/${plannedShards} ok · ${failedCount} failed`;
  }
  if (observedCount > 0) {
    return `${failedCount} failed / ${observedCount} observed`;
  }
  return "no shard counters";
}

export function parseShardCounters(message: string): { planned: number; succeeded: number; failed: number } | null {
  const match = message.match(/shards_total=(\d+)\s+succeeded=(\d+)\s+failed=(\d+)/i) ?? message.match(/planned=(\d+)\s+succeeded=(\d+)\s+failed=(\d+)/i);
  if (!match) {
    return null;
  }
  return {
    planned: Number(match[1]),
    succeeded: Number(match[2]),
    failed: Number(match[3]),
  };
}

function parseRuntimeOutputJSON(message: string): Record<string, unknown> | null {
  const trimmed = message.trim();
  if (!trimmed.startsWith("{") || !trimmed.endsWith("}")) {
    return null;
  }
  try {
    const parsed: unknown = JSON.parse(trimmed);
    return isRecord(parsed) ? parsed : null;
  } catch {
    return null;
  }
}

function objectField(record: Record<string, unknown> | null, key: string): Record<string, unknown> | null {
  const value = record?.[key];
  return isRecord(value) ? value : null;
}

function objectString(record: Record<string, unknown> | null, key: string): string {
  const value = record?.[key];
  return typeof value === "string" ? value.trim() : "";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
