import { useMemo, useState } from "react";

import { fetchJSON } from "../lib/api";
import { formatTimestamp, runLogsPageLimit } from "../lib/runState";
import type { RunLogEntry, RunLogsResponse } from "../lib/appContracts";

type UseRunLogsOptions = {
  runId: string | null;
};

export type RunLogsMode = "events" | "raw" | "all";
export type RunLogsViewMode = "line" | "line+fields";

export function useRunLogs({ runId }: UseRunLogsOptions) {
  const [runLogs, setRunLogs] = useState<RunLogEntry[]>([]);
  const [runLogsCursor, setRunLogsCursor] = useState(0);
  const [runLogsEOF, setRunLogsEOF] = useState(false);
  const [runLogsStatus, setRunLogsStatus] = useState("");
  const [runLogsViewMode, setRunLogsViewMode] = useState<RunLogsViewMode>("line");
  const [runLogsMode, setRunLogsMode] = useState<RunLogsMode>("all");

  const runLogTaskrunPaths = useMemo(() => {
    const paths = new Set<string>();
    for (const entry of runLogs) {
      if (entry.taskrun_path && entry.taskrun_path.trim().length > 0) {
        paths.add(entry.taskrun_path.trim());
      }
    }
    return Array.from(paths).sort((left, right) => left.localeCompare(right));
  }, [runLogs]);

  const filteredRunLogs = useMemo(() => {
    if (runLogsMode === "all") {
      return runLogs;
    }
    if (runLogsMode === "events") {
      return runLogs.filter((entry) => (entry.kind ?? "event") !== "runtime_output");
    }
    return runLogs.filter((entry) => (entry.kind ?? "event") === "runtime_output");
  }, [runLogs, runLogsMode]);

  const runLogsRendered = useMemo(() => {
    const includeFields = runLogsViewMode === "line+fields";
    return filteredRunLogs
      .map((entry) => {
        const line = formatRunLogLine(entry);
        if (!includeFields) {
          return line;
        }
        if (!entry.fields || Object.keys(entry.fields).length === 0) {
          return line;
        }
        const serialized = JSON.stringify(entry.fields, null, 2);
        if (!serialized) {
          return line;
        }
        return `${line}\n${serialized}`;
      })
      .join(includeFields ? "\n\n" : "\n");
  }, [filteredRunLogs, runLogsViewMode]);

  function resetRunLogs() {
    setRunLogs([]);
    setRunLogsCursor(0);
    setRunLogsEOF(false);
    setRunLogsStatus("");
  }

  function mergeRunLogsPayload(payload: RunLogsResponse, reset: boolean, fallbackCursor: number) {
    setRunLogs((previous) => {
      const seed = reset ? [] : previous;
      const merged = [...seed];
      const seen = new Set(merged.map((entry) => entry.cursor));
      for (const entry of payload.items ?? []) {
        if (seen.has(entry.cursor)) {
          continue;
        }
        merged.push(entry);
        seen.add(entry.cursor);
      }
      merged.sort((left, right) => left.cursor - right.cursor);
      return merged;
    });
    setRunLogsCursor(payload.next_cursor ?? fallbackCursor);
    setRunLogsEOF(Boolean(payload.eof));
  }

  async function fetchRunLogs(id: string, reset = false): Promise<RunLogsResponse | null> {
    if (!id) {
      return null;
    }
    const cursor = reset ? 0 : runLogsCursor;
    if (!reset && runLogsEOF) {
      return null;
    }
    const payload = await fetchJSON<RunLogsResponse>(`/api/pipeline/runs/${id}/logs?cursor=${cursor}&limit=${runLogsPageLimit}`);
    mergeRunLogsPayload(payload, reset, cursor);
    return payload;
  }

  async function fetchRunLogsUntilEOF(id: string) {
    if (!id) {
      return;
    }
    let cursor = 0;
    let reset = true;
    for (let page = 0; page < 25; page += 1) {
      const payload = await fetchJSON<RunLogsResponse>(`/api/pipeline/runs/${id}/logs?cursor=${cursor}&limit=${runLogsPageLimit}`);
      mergeRunLogsPayload(payload, reset, cursor);
      if (payload.eof) {
        return;
      }
      const nextCursor = payload.next_cursor ?? cursor;
      if (nextCursor <= cursor) {
        return;
      }
      cursor = nextCursor;
      reset = false;
    }
  }

  async function handleCopyRunLogs() {
    if (filteredRunLogs.length === 0 || runLogsRendered.trim().length === 0) {
      return;
    }
    if (!navigator.clipboard || !navigator.clipboard.writeText) {
      setRunLogsStatus("Clipboard API is not available in this browser context.");
      return;
    }

    try {
      await navigator.clipboard.writeText(runLogsRendered);
      setRunLogsStatus("Run logs copied to clipboard");
    } catch (requestError) {
      setRunLogsStatus(requestError instanceof Error ? requestError.message : "Run logs copy failed");
    }
  }

  function handleDownloadRunLogs() {
    if (filteredRunLogs.length === 0 || !runId || runLogsRendered.trim().length === 0) {
      return;
    }
    const blob = new Blob([runLogsRendered], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${runId}.logs.txt`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    setRunLogsStatus(`Downloaded ${runId}.logs.txt`);
  }

  return {
    runLogsCursor,
    runLogsEOF,
    runLogsStatus,
    runLogsViewMode,
    runLogsMode,
    runLogTaskrunPaths,
    filteredRunLogs,
    runLogsRendered,
    setRunLogsViewMode,
    setRunLogsMode,
    resetRunLogs,
    fetchRunLogs,
    fetchRunLogsUntilEOF,
    handleCopyRunLogs,
    handleDownloadRunLogs,
  };
}

function formatRunLogLine(entry: RunLogEntry): string {
  const kind = entry.kind ?? "event";
  const stream = entry.stream ? `[${entry.stream.toUpperCase()}]` : "";
  const parts = [
    formatTimestamp(entry.timestamp),
    entry.level.toUpperCase(),
    kind === "runtime_output" ? "[RAW]" : "[EVENT]",
    stream,
    entry.step_id ? `[${entry.step_id}]` : "",
    entry.domain_id ? `(${entry.domain_id})` : "",
    entry.message,
  ].filter((value) => value && value.trim().length > 0);
  return parts.join(" ");
}
