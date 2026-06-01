import { formatTimestamp } from "../lib/runState";
import type { RunLogEntry } from "../lib/appContracts";
import { ArtifactPathButton } from "./ConsolePrimitives";

type ActivityDrawerProps = {
  selectedRunId?: string;
  selectedRunStatus?: string;
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
  const recentLogs = logs.slice(-6).reverse();
  const hasSelectedRun = Boolean(selectedRunId);
  const emptyLogMessage = !hasSelectedRun
    ? "No selected run. Start or select a run to stream activity."
    : selectedRunStatus === "failed"
      ? `Run failed before log entries were captured${selectedRunError ? `: ${selectedRunError}` : "."}`
      : "No run logs yet. Logs will appear when the selected run emits events or raw output.";
  const drawerSummary = hasSelectedRun ? `${logs.length} log entries for ${selectedRunId}` : "No selected run";
  return (
    <section className="activity-drawer" aria-label="Selected run activity drawer" data-testid="activity-drawer">
      <div className="activity-head">
        <div>
          <h2>Activity / Events</h2>
          <p className="hint">{drawerSummary}</p>
        </div>
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

      {recentLogs.length === 0 ? (
        <div className={`activity-empty-state${selectedRunStatus === "failed" ? " failed" : ""}`} data-testid="activity-empty-state">
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
                  <td>{entry.message}</td>
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
    </section>
  );
}
