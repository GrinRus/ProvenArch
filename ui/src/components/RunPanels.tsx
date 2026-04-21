import { RunStatusPanel } from "./RunStatusPanel";
import { formatTimestamp } from "../lib/runState";
import type { RunListItem, RunStatusResponse } from "../lib/appContracts";

type RunPanelsProps = {
  busy: boolean;
  cancelBusy: boolean;
  runId: string | null;
  runStatus: RunStatusResponse | null;
  runList: RunListItem[];
  runActionStatus: string;
  selectedRunWarnings: string[];
  selectedRunIsActive: boolean;
  runCounters: { running: number; succeeded: number; failed: number };
  runLogsMode: "events" | "raw" | "all";
  runLogsViewMode: "line" | "line+fields";
  filteredRunLogs: unknown[];
  runLogsStatus: string;
  runLogTaskrunPaths: string[];
  runLogsRendered: string;
  onRunLogsModeChange: (value: "events" | "raw" | "all") => void;
  onRunLogsViewModeChange: (value: "line" | "line+fields") => void;
  onRunPipeline: (pipeline: "init" | "refresh") => void;
  onCancelSelectedRun: () => void;
  onSelectRun: (runId: string) => void;
  onCopyRunLogs: () => void;
  onDownloadRunLogs: () => void;
  onOpenArtifact: (path: string) => void;
};

export function RunPanels({
  busy,
  cancelBusy,
  runId,
  runStatus,
  runList,
  runActionStatus,
  selectedRunWarnings,
  selectedRunIsActive,
  runCounters,
  runLogsMode,
  runLogsViewMode,
  filteredRunLogs,
  runLogsStatus,
  runLogTaskrunPaths,
  runLogsRendered,
  onRunLogsModeChange,
  onRunLogsViewModeChange,
  onRunPipeline,
  onCancelSelectedRun,
  onSelectRun,
  onCopyRunLogs,
  onDownloadRunLogs,
  onOpenArtifact,
}: RunPanelsProps) {
  return (
    <>
      <section className="panel" data-testid="runs-control-panel">
        <h2>Runs: Pipeline Control</h2>
        <div className="actions">
          <button type="button" onClick={() => onRunPipeline("init")} disabled={busy} data-testid="run-init-btn">
            Run init
          </button>
          <button type="button" onClick={() => onRunPipeline("refresh")} disabled={busy} data-testid="run-refresh-btn">
            Run refresh
          </button>
          <button
            type="button"
            onClick={onCancelSelectedRun}
            disabled={busy || cancelBusy || !runId || !selectedRunIsActive}
            data-testid="run-cancel-btn"
          >
            Cancel selected run
          </button>
        </div>
        {runActionStatus ? <p className="status warn">{runActionStatus}</p> : null}

        <RunStatusPanel runStatus={runStatus} warnings={selectedRunWarnings} />
      </section>

      <section className="panel" data-testid="runs-history-panel">
        <h2>Runs: History</h2>
        <p className="hint">
          Running: {runCounters.running} | Succeeded: {runCounters.succeeded} | Failed: {runCounters.failed}
        </p>
        {runList.length === 0 ? (
          <p>No runs yet.</p>
        ) : (
          <div className="run-table-wrap">
            <table className="run-table" data-testid="runs-history-table">
              <thead>
                <tr>
                  <th>Run ID</th>
                  <th>Status</th>
                  <th>Pipeline</th>
                  <th>Started</th>
                  <th>Finished</th>
                  <th>Error code</th>
                  <th>Warnings</th>
                </tr>
              </thead>
              <tbody>
                {runList.map((run) => (
                  <tr key={run.run_id} className={runId === run.run_id ? "selected" : ""} onClick={() => onSelectRun(run.run_id)}>
                    <td>
                      <button
                        type="button"
                        className="link-button"
                        onClick={(event) => {
                          event.stopPropagation();
                          onSelectRun(run.run_id);
                        }}
                      >
                        {run.run_id}
                      </button>
                    </td>
                    <td>{run.status}</td>
                    <td>{run.pipeline}</td>
                    <td>{formatTimestamp(run.started_at)}</td>
                    <td>{formatTimestamp(run.finished_at)}</td>
                    <td>{run.error_code || "-"}</td>
                    <td>{run.warnings?.length ?? 0}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="panel" data-testid="runs-logs-panel">
        <h2>Runs: Logs</h2>
        <div className="actions">
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
          <button type="button" onClick={onCopyRunLogs} disabled={filteredRunLogs.length === 0} data-testid="run-logs-copy-btn">
            Copy logs
          </button>
          <button
            type="button"
            onClick={onDownloadRunLogs}
            disabled={filteredRunLogs.length === 0 || !runId}
            data-testid="run-logs-download-btn"
          >
            Download logs
          </button>
        </div>
        {runLogsStatus ? <p className="status ok">{runLogsStatus}</p> : null}
        {runLogTaskrunPaths.length > 0 ? (
          <div className="actions">
            {runLogTaskrunPaths.map((path) => (
              <button key={`taskrun-log-open-${path}`} type="button" onClick={() => onOpenArtifact(path)}>
                Open runtime execution artifact: {path}
              </button>
            ))}
          </div>
        ) : null}
        {filteredRunLogs.length === 0 ? <p>No run logs yet.</p> : <pre data-testid="run-logs-content">{runLogsRendered}</pre>}
      </section>
    </>
  );
}
