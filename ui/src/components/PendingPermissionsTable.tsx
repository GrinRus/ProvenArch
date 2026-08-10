import { StatusBadge } from "./ConsolePrimitives";
import type { RuntimePermissionRequest } from "../lib/appContracts";

function compactUniqueValues(values: Array<string | null | undefined>): string[] {
  return Array.from(new Set(values.map((value) => String(value ?? "").trim()).filter(Boolean)));
}

export function PendingPermissionsTable({ pendingPermissions }: { pendingPermissions: RuntimePermissionRequest[] }) {
  const primaryRequest = pendingPermissions[0] ?? null;
  const requestCountLabel = `${pendingPermissions.length} pending ${pendingPermissions.length === 1 ? "request" : "requests"}`;
  const blockedStep = compactUniqueValues(pendingPermissions.map((request) => request.step_id)).join(", ") || "-";
  const actions = compactUniqueValues(pendingPermissions.map((request) => request.action)).join(", ") || "-";
  const decisions = compactUniqueValues(pendingPermissions.map((request) => request.decision?.decision)).join(", ") || "-";
  const rules = compactUniqueValues(pendingPermissions.map((request) => request.decision?.rule_id)).join(", ") || "-";

  return (
    <section className="subsection" data-testid="runs-pending-permissions-panel">
      <h2>Pending permissions</h2>
      {pendingPermissions.length === 0 ? (
        <p>No pending runtime permission requests.</p>
      ) : (
        <>
          <section className="permission-recovery-panel" data-testid="runtime-permission-recovery">
            <div className="section-heading-row">
              <div>
                <h3>Permission triage</h3>
                <p className="hint">
                  Managed runtime paused before approving an operation outside the step envelope. Review the target and rule before retrying.
                </p>
              </div>
              <StatusBadge tone="warn">{requestCountLabel}</StatusBadge>
            </div>
            <div className="permission-recovery-grid">
              <div>
                <span className="metric-label">Blocked step</span>
                <strong>{blockedStep}</strong>
              </div>
              <div>
                <span className="metric-label">Operation</span>
                <strong>{actions}</strong>
              </div>
              <div>
                <span className="metric-label">Decision</span>
                <strong>{decisions}</strong>
              </div>
              <div>
                <span className="metric-label">Policy rule</span>
                <strong>{rules}</strong>
              </div>
            </div>
            {primaryRequest ? (
              <dl className="compact-defs permission-request-summary">
                <div>
                  <dt>Primary target</dt>
                  <dd>{primaryRequest.path_or_command || "-"}</dd>
                </div>
                <div>
                  <dt>Reason</dt>
                  <dd>{primaryRequest.reason || primaryRequest.decision?.message || "No reason recorded."}</dd>
                </div>
              </dl>
            ) : null}
            <ul className="analysis-next-actions">
              <li>Inspect the path or command and reason before rerun.</li>
              <li>Use Readiness - Advanced runtime settings - Runtime Permissions to choose the intended mode/channel.</li>
              <li>If the request is unexpected, adjust source scope or runtime profile before retrying the failed pipeline.</li>
            </ul>
          </section>
          <div className="permission-request-cards" data-testid="runs-pending-permissions-cards">
            {pendingPermissions.map((request) => (
              <article className="permission-request-card" key={request.request_id || `${request.step_id}-${request.action}-${request.path_or_command}-card`}>
                <div className="section-heading-row">
                  <div>
                    <span className="metric-label">Permission request</span>
                    <strong className="permission-request-card-title">{request.action || "runtime permission"}</strong>
                  </div>
                  <StatusBadge tone="warn">{request.decision?.decision || "pending"}</StatusBadge>
                </div>
                <dl className="compact-defs permission-request-summary">
                  <div><dt>Request ID</dt><dd>{request.request_id || "-"}</dd></div>
                  <div><dt>Provider</dt><dd>{request.provider || "-"}</dd></div>
                  <div><dt>Step</dt><dd>{request.step_id || "-"}</dd></div>
                  <div><dt>Rule</dt><dd>{request.decision?.rule_id || "-"}</dd></div>
                  <div><dt>Target</dt><dd>{request.path_or_command || "-"}</dd></div>
                  <div><dt>Reason</dt><dd>{request.reason || request.decision?.message || "-"}</dd></div>
                </dl>
              </article>
            ))}
          </div>
          <div className="run-table-wrap permission-request-table-wrap">
            <table className="run-table" data-testid="runs-pending-permissions-table">
              <thead>
                <tr>
                  <th>Request ID</th><th>Provider</th><th>Step</th><th>Action</th><th>Decision</th><th>Rule</th><th>Path or command</th><th>Reason</th>
                </tr>
              </thead>
              <tbody>
                {pendingPermissions.map((request) => (
                  <tr key={request.request_id || `${request.step_id}-${request.action}-${request.path_or_command}`}>
                    <td>{request.request_id || "-"}</td><td>{request.provider || "-"}</td><td>{request.step_id || "-"}</td>
                    <td>{request.action || "-"}</td><td>{request.decision?.decision || "-"}</td><td>{request.decision?.rule_id || "-"}</td>
                    <td>{request.path_or_command || "-"}</td><td>{request.reason || "-"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </section>
  );
}
