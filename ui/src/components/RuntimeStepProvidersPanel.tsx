type RuntimeStepProvidersPanelProps = {
  labels: Record<string, string>;
  persisted: Record<string, string | undefined>;
  effective: Record<string, string | undefined>;
  source: Record<string, string | undefined>;
  order: string[];
};

export function RuntimeStepProvidersPanel(props: RuntimeStepProvidersPanelProps) {
  const { labels, persisted, effective, source, order } = props;

  return (
    <div className="run-table-wrap" data-testid="runtime-step-providers-table-wrap">
      <table className="run-table" data-testid="runtime-step-providers-table">
        <thead>
          <tr>
            <th>Step</th>
            <th>Persisted</th>
            <th>Effective</th>
            <th>Source</th>
          </tr>
        </thead>
        <tbody>
          {order.map((stepKey) => (
            <tr key={stepKey}>
              <td>{labels[stepKey] ?? stepKey}</td>
              <td>{persisted[stepKey] ?? "-"}</td>
              <td>{effective[stepKey] ?? "-"}</td>
              <td>{source[stepKey] ?? "default"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
