type RunStatus = {
  run_id: string;
  pipeline: string;
  status: string;
  current_step?: string;
  error_code?: string | null;
  error?: string | null;
};

type RunStatusPanelProps = {
  runStatus: RunStatus | null;
  warnings: string[];
};

export function RunStatusPanel({ runStatus, warnings }: RunStatusPanelProps) {
  if (!runStatus) {
    return null;
  }

  return (
    <div className="status-block" data-testid="run-status-panel">
      <p>
        Run <code data-testid="run-status-run-id">{runStatus.run_id}</code> status: <strong data-testid="run-status-value">{runStatus.status}</strong>
      </p>
      <p>Pipeline: {runStatus.pipeline}</p>
      {runStatus.current_step ? <p>Current step: {runStatus.current_step}</p> : null}
      {runStatus.error_code ? <p className="status warn">Error code: {runStatus.error_code}</p> : null}
      {runStatus.error ? <p className="status err">Error: {runStatus.error}</p> : null}
      {warnings.length > 0 ? (
        <div data-testid="run-status-warnings">
          <p className="hint">Warnings ({warnings.length})</p>
          <ul>
            {warnings.map((warning, index) => (
              <li key={`run-warning-${index}-${warning}`}>{warning}</li>
            ))}
          </ul>
        </div>
      ) : (
        <p data-testid="run-status-warnings-empty">Warnings: none</p>
      )}
    </div>
  );
}
