import { StatusBadge } from "./ConsolePrimitives";
import { formatTimestamp, runOutcomeLabel, runOutcomeTone } from "../lib/runState";
import type { RunListItem } from "../lib/appContracts";

export function RunHistoryTable({
  runId,
  runList,
  runCounters,
  onSelectRun,
}: {
  runId: string | null;
  runList: RunListItem[];
  runCounters: { running: number; succeeded: number; failed: number };
  onSelectRun: (runId: string) => void;
}) {
  const terminalOutcomeCounts = runList.reduce(
    (counts, run) => {
      const label = runOutcomeLabel(run);
      if (label === "failed") {
        counts.failed += 1;
      } else if (label === "canceled") {
        counts.canceled += 1;
      } else if (label === "recovered") {
        counts.recovered += 1;
      }
      return counts;
    },
    { failed: 0, canceled: 0, recovered: 0 },
  );

  return (
    <section className="subsection" data-testid="runs-history-panel">
      <h2>History</h2>
      <p className="hint">
        Running: {runCounters.running} | Succeeded: {runCounters.succeeded} | Failed: {terminalOutcomeCounts.failed} | Canceled: {terminalOutcomeCounts.canceled} |
        Recovered: {terminalOutcomeCounts.recovered}
      </p>
      {runList.length === 0 ? (
        <p>No runs yet.</p>
      ) : (
        <div className="run-table-wrap">
          <table className="run-table responsive-card-table" data-testid="runs-history-table">
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
                  <td data-label="Run ID">
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
                  <td data-label="Status">
                    <StatusBadge tone={runOutcomeTone(run)}>{runOutcomeLabel(run)}</StatusBadge>
                  </td>
                  <td data-label="Pipeline">{run.pipeline}</td>
                  <td data-label="Started">{formatTimestamp(run.started_at)}</td>
                  <td data-label="Finished">{formatTimestamp(run.finished_at)}</td>
                  <td data-label="Error code">{run.error_code || "-"}</td>
                  <td data-label="Warnings">{run.warnings?.length ?? 0}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
