import { StatusBadge } from "./ConsolePrimitives";

export type AnalysisLiveMetric = {
  label: string;
  value: string;
  detail: string;
};

export type AnalysisLiveTrace = {
  label: string;
  value: string;
};

export type AnalysisLiveDiagnostics = {
  status: string;
  tone: "info" | "ok" | "warn" | "error";
  summary: string;
  metrics: AnalysisLiveMetric[];
  traces: AnalysisLiveTrace[];
  actions: string[];
  hasTelemetry: boolean;
};

export function AnalysisLiveDiagnosticsPanel({ diagnostics }: { diagnostics: AnalysisLiveDiagnostics }) {
  return (
    <section className="analysis-live-diagnostics" data-testid="analysis-live-diagnostics">
      <div className="section-heading-row">
        <div>
          <h3>Live diagnostics</h3>
          <p className="hint">{diagnostics.summary}</p>
        </div>
        <StatusBadge tone={diagnostics.tone}>{diagnostics.status}</StatusBadge>
      </div>
      <div className="analysis-live-grid">
        {diagnostics.metrics.map((metric) => (
          <div key={metric.label}>
            <span className="metric-label">{metric.label}</span>
            <strong>{metric.value}</strong>
            <span>{metric.detail}</span>
          </div>
        ))}
      </div>
      <dl className="compact-defs analysis-live-traces">
        {diagnostics.traces.map((trace) => (
          <div key={trace.label}>
            <dt>{trace.label}</dt>
            <dd>{trace.value}</dd>
          </div>
        ))}
      </dl>
      <ul className="analysis-next-actions">
        {diagnostics.actions.map((action) => (
          <li key={action}>{action}</li>
        ))}
      </ul>
    </section>
  );
}
