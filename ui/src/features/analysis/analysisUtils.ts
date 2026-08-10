import type { RunLogEntry } from "../../lib/appContracts";

export function fieldString(fields: Record<string, unknown> | undefined, key: string): string {
  const value = fields?.[key];
  return typeof value === "string" && value.trim().length > 0 ? value.trim() : "";
}

export function numericField(fields: Record<string, unknown> | undefined, key: string): number | undefined {
  const value = fields?.[key];
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim()) {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
  }
  return undefined;
}

export function firstNumericField(fields: Record<string, unknown> | undefined, keys: string[]): number | undefined {
  for (const key of keys) {
    const value = numericField(fields, key);
    if (value !== undefined) {
      return value;
    }
  }
  return undefined;
}

export function maxDefined(current: number | undefined, next: number | undefined): number | undefined {
  if (next === undefined) {
    return current;
  }
  return current === undefined ? next : Math.max(current, next);
}

export function boolField(fields: Record<string, unknown> | undefined, key: string): boolean {
  const value = fields?.[key];
  if (typeof value === "boolean") {
    return value;
  }
  return typeof value === "string" && value.trim().toLowerCase() === "true";
}

export function rawOutputRefsFromEntry(entry: RunLogEntry): string[] {
  const refs = new Set<string>();
  const rawOutput = fieldString(entry.fields, "raw_output");
  if (rawOutput) {
    refs.add(rawOutput);
  }
  const rawOutputMetadata = fieldString(entry.fields, "raw_output_metadata");
  if (rawOutputMetadata) {
    refs.add(rawOutputMetadata);
  }
  for (const match of entry.message.matchAll(/raw_output=([^\s)]+)/gi)) {
    refs.add(match[1]);
  }
  return Array.from(refs);
}

export function firstNonEmpty(values: string[]): string {
  return values.map((value) => value.trim()).find((value) => value.length > 0) ?? "";
}

export function lastString(values: string[]): string {
  return values.length > 0 ? values[values.length - 1] : "";
}

export function formatDurationMillis(milliseconds: number): string {
  if (milliseconds < 1000) {
    return `${Math.round(milliseconds)}ms`;
  }
  const totalSeconds = Math.round(milliseconds / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes === 0) {
    return `${seconds}s`;
  }
  return `${minutes}m ${seconds.toString().padStart(2, "0")}s`;
}

export function formatCompactCount(value: number): string {
  if (value < 1000) {
    return `${value}`;
  }
  if (value < 1_000_000) {
    return `${Math.round(value / 100) / 10}k`;
  }
  return `${Math.round(value / 100_000) / 10}m`;
}
