import { useEffect, useState } from "react";

import { formatTimestamp, runOutcomeLabel } from "../lib/runState";
import type { RunLogEntry } from "../lib/appContracts";
import { ArtifactPathButton } from "./ConsolePrimitives";

type ActivityDrawerProps = {
  selectedRunId?: string;
  selectedRunStatus?: string;
  selectedRunErrorCode?: string;
  selectedRunError?: string;
  logs: RunLogEntry[];
  renderedLogs: string;
  runLogsStatus: string;
  runLogsMode: "events" | "raw" | "all";
  runLogsViewMode: "line" | "line+fields";
  onRunLogsModeChange: (value: "events" | "raw" | "all") => void;
  onRunLogsViewModeChange: (value: "line" | "line+fields") => void;
  onCopyRunLogs: () => void;
  onDownloadRunLogs: () => void;
  canExport: boolean;
  taskrunPaths: string[];
  onOpenArtifact: (path: string) => void;
};

export function ActivityDrawer({
  selectedRunId,
  selectedRunStatus,
  selectedRunErrorCode,
  selectedRunError,
  logs,
  renderedLogs,
  runLogsStatus,
  runLogsMode,
  runLogsViewMode,
  onRunLogsModeChange,
  onRunLogsViewModeChange,
  onCopyRunLogs,
  onDownloadRunLogs,
  canExport,
  taskrunPaths,
  onOpenArtifact,
}: ActivityDrawerProps) {
  const shouldOpenForRunState = selectedRunStatus === "queued" || selectedRunStatus === "running" || selectedRunStatus === "failed";
  const [isOpen, setIsOpen] = useState(shouldOpenForRunState);
  const recentLogs = logs.slice(-6).reverse();
  const lastLog = logs.length > 0 ? logs[logs.length - 1] : undefined;
  const streamSummary = summarizeActivityStream(logs);
  const hasSelectedRun = Boolean(selectedRunId);
  const selectedRunOutcome = selectedRunStatus ? runOutcomeLabel({ status: selectedRunStatus, error_code: selectedRunErrorCode }, selectedRunStatus) : "";
  const selectedRunDetail = selectedRunErrorCode || selectedRunError;
  const emptyLogMessage = !hasSelectedRun
    ? "No selected run. Start or select a run to stream activity."
    : selectedRunOutcome === "canceled"
      ? `Run was canceled before log entries were captured${selectedRunDetail ? `: ${selectedRunDetail}` : "."} Taskrun evidence remains in History.`
      : selectedRunOutcome === "recovered"
        ? `Run was reconciled after restart before log entries were captured${selectedRunDetail ? `: ${selectedRunDetail}` : "."} History retains the run; start a new run if analysis still matters.`
        : selectedRunStatus === "failed"
          ? `Run failed before log entries were captured${selectedRunDetail ? `: ${selectedRunDetail}` : "."}`
          : "No run logs yet. Logs will appear when the selected run emits events or raw output.";
  const drawerSummary = hasSelectedRun
    ? `${selectedRunOutcome ? `${selectedRunOutcome} run` : "selected run"} · ${logs.length} log entries${lastLog ? ` · last: ${activityLogSummary(lastLog)}` : ` for ${selectedRunId}`}`
    : "No selected run";
  const drawerStatus = runLogsStatus || (isOpen ? `${recentLogs.length} recent` : `${recentLogs.length} recent · open logs`);
  const emptyStateTone = selectedRunOutcome === "canceled" || selectedRunOutcome === "recovered" ? " warning" : selectedRunStatus === "failed" ? " failed" : "";

  useEffect(() => {
    setIsOpen(shouldOpenForRunState);
  }, [selectedRunId, selectedRunStatus, shouldOpenForRunState]);

  return (
    <details
      className={`activity-drawer${isOpen ? " is-open" : " is-collapsed"}`}
      aria-label="Selected run activity drawer"
      data-testid="activity-drawer"
      open={isOpen}
      onToggle={(event) => setIsOpen(event.currentTarget.open)}
    >
      <summary className="activity-summary" data-testid="activity-drawer-toggle">
        <span className="activity-summary-copy">
          <strong>Activity / Events</strong>
          <span className="hint">{drawerSummary}</span>
        </span>
        <span className="activity-summary-status">{drawerStatus}</span>
      </summary>
      <div className="activity-drawer-body">
        <div className="activity-head">
          <div className="activity-controls">
            <label htmlFor="runLogsMode">Mode</label>
            <select
              id="runLogsMode"
              value={runLogsMode}
              onChange={(event) => onRunLogsModeChange(event.target.value as "events" | "raw" | "all")}
              className="inline-select"
              data-testid="run-logs-mode-select"
            >
              <option value="all">all</option>
              <option value="events">event timeline</option>
              <option value="raw">raw agent stream</option>
            </select>
            <label htmlFor="runLogsViewMode">View</label>
            <select
              id="runLogsViewMode"
              value={runLogsViewMode}
              onChange={(event) => onRunLogsViewModeChange(event.target.value as "line" | "line+fields")}
              className="inline-select"
              data-testid="run-logs-view-select"
            >
              <option value="line">line</option>
              <option value="line+fields">line+fields</option>
            </select>
            <button type="button" className="secondary-action" onClick={onCopyRunLogs} disabled={!canExport} data-testid="run-logs-copy-btn">
              Copy logs
            </button>
            <button type="button" className="secondary-action" onClick={onDownloadRunLogs} disabled={!canExport} data-testid="run-logs-download-btn">
              Download logs
            </button>
          </div>
        </div>

        {runLogsStatus ? (
          <p className="status ok" aria-live="polite">
            {runLogsStatus}
          </p>
        ) : null}
        {taskrunPaths.length > 0 ? (
          <details className="taskrun-actions-details">
            <summary>Runtime execution artifacts ({taskrunPaths.length})</summary>
            <div className="taskrun-actions">
              {taskrunPaths.map((path) => (
                <ArtifactPathButton key={`taskrun-log-open-${path}`} path={path} label="runtime-execution.json" kind="taskrun" actionLabel="Open runtime execution artifact" onOpenArtifact={onOpenArtifact} />
              ))}
            </div>
          </details>
        ) : null}
        {streamSummary ? (
          <div className="activity-stream-summary" data-testid="activity-stream-summary">
            <strong>Provider stream summarized</strong>
            <span>{streamSummary}</span>
          </div>
        ) : null}

        {recentLogs.length === 0 ? (
          <div className={`activity-empty-state${emptyStateTone}`} data-testid="activity-empty-state">
            <strong>{hasSelectedRun ? "No log entries" : "No run selected"}</strong>
            <span>{emptyLogMessage}</span>
          </div>
        ) : (
          <div className="activity-table-wrap">
            <table className="activity-table" data-testid="activity-events-table">
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Level</th>
                  <th>Step</th>
                  <th>Event</th>
                  <th>Details</th>
                </tr>
              </thead>
              <tbody>
                {recentLogs.map((entry) => (
                  <tr key={`${entry.cursor}-${entry.timestamp}-${entry.message}`}>
                    <td>{formatTimestamp(entry.timestamp)}</td>
                    <td>{entry.level}</td>
                    <td>{entry.step_id || "-"}</td>
                    <td>{entry.kind === "runtime_output" ? "runtime output" : "event"}</td>
                    <td>{activityLogSummary(entry)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <details className="raw-log-details">
          <summary>Full selected log view</summary>
          {logs.length === 0 ? <p>{emptyLogMessage}</p> : <pre data-testid="run-logs-content">{renderedLogs}</pre>}
        </details>
      </div>
    </details>
  );
}

function summarizeActivityStream(logs: RunLogEntry[]): string | null {
  let rawOutputEntries = 0;
  let jsonStreamChunks = 0;
  const signalTypes = new Set<string>();
  for (const entry of logs) {
    if ((entry.kind ?? "event") !== "runtime_output") {
      continue;
    }
    rawOutputEntries += 1;
    const signal = providerStreamSignal(entry.message);
    if (signal) {
      jsonStreamChunks += 1;
      signalTypes.add(signal);
    }
  }
  if (rawOutputEntries === 0) {
    return null;
  }
  if (jsonStreamChunks === 0) {
    return `${rawOutputEntries} raw output ${rawOutputEntries === 1 ? "entry" : "entries"} loaded. Full payload remains available below and in exported logs.`;
  }
  const signalText = signalTypes.size > 0 ? ` (${Array.from(signalTypes).slice(0, 3).join(", ")})` : "";
  return `${jsonStreamChunks} JSON stream ${jsonStreamChunks === 1 ? "chunk" : "chunks"}${signalText} summarized from ${rawOutputEntries} raw output ${rawOutputEntries === 1 ? "entry" : "entries"}. Full payload remains available below and in exported logs.`;
}

function activityLogSummary(entry: RunLogEntry): string {
  const message = entry.message || "";
  if ((entry.kind ?? "event") === "runtime_output") {
    const signal = providerStreamSignal(message);
    if (signal) {
      return `Provider stream chunk: ${signal}. Full raw payload is in the selected log view.`;
    }
  }
  return truncateActivityMessage(message);
}

function providerStreamSignal(message: string): string | null {
  try {
    const parsed = JSON.parse(message) as unknown;
    if (!parsed || typeof parsed !== "object") {
      return null;
    }
    const record = parsed as Record<string, unknown>;
    const event = isRecord(record.event) ? record.event : null;
    const delta = isRecord(event?.delta) ? event.delta : isRecord(record.delta) ? record.delta : null;
    const signal = firstString([delta?.type, event?.type, record.type]);
    if (record.type === "stream_event" || event || delta) {
      return signal || "stream_event";
    }
  } catch {
    return null;
  }
  return null;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function firstString(values: unknown[]): string | null {
  for (const value of values) {
    if (typeof value === "string" && value.trim().length > 0) {
      return value.trim();
    }
  }
  return null;
}

function truncateActivityMessage(message: string): string {
  const normalized = message.replace(/\s+/g, " ").trim();
  if (normalized.length <= 220) {
    return normalized;
  }
  return `${normalized.slice(0, 217)}...`;
}
