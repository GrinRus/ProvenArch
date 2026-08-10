import { StatusBadge } from "../../components/ConsolePrimitives";
import { qaFailureGuidance } from "./qaUtils";
import { isRunCanceled, isRunReconciledAfterRestart } from "../../lib/runState";
import type { QARunResponse } from "../../lib/qaApi";

export function QAFailureRecovery({
  qaRun,
  busy,
  onRetry,
}: {
  qaRun: QARunResponse | null;
  busy: boolean;
  onRetry: () => void;
}) {
  if (qaRun?.status !== "failed") {
    return null;
  }
  const errorCode = qaRun.error_code || "unclassified";
  const blockedStep = qaRun.current_step || "qa.ask";
  const warningCount = qaRun.warnings?.length ?? 0;
  const auditRefs = `reports/taskruns/${qaRun.run_id}/qa/`;
  const canRetry = Boolean((qaRun.question || "").trim());
  const canceled = isRunCanceled(errorCode);
  const reconciled = isRunReconciledAfterRestart(errorCode);
  const title = canceled ? "Canceled answer run" : reconciled ? "Recovered answer run" : "Recovery path";
  const badgeLabel = canceled ? "canceled" : reconciled ? "recovered" : "failed";
  const badgeTone = canceled || reconciled ? "warn" : "error";
  const stepLabel = canceled ? "Stopped step" : reconciled ? "Recovered step" : "Blocked step";
  const retryLabel = canceled || reconciled ? "Ask again" : "Retry question";
  const retentionHint = canceled
    ? "Asking again creates a new Q&A run; the canceled attempt and QA audit artifacts stay in history."
    : reconciled
      ? "Asking again creates a new Q&A run; the reconciled attempt and QA audit artifacts stay in history."
      : "Retry starts a new Q&A run; the failed answer attempt stays in history for audit.";

  return (
    <section className="qa-recovery-panel" data-testid="qa-failure-recovery">
      <div className="section-heading-row">
        <div>
          <h2>{title}</h2>
          <p className="hint">{qaFailureGuidance(errorCode, warningCount)}</p>
        </div>
        <StatusBadge tone={badgeTone}>{badgeLabel}</StatusBadge>
      </div>
      <div className="qa-recovery-grid">
        <div><span className="metric-label">Classification</span><strong>{errorCode}</strong></div>
        <div><span className="metric-label">{stepLabel}</span><strong>{blockedStep}</strong></div>
        <div><span className="metric-label">Audit evidence</span><strong>{auditRefs}</strong></div>
        <div><span className="metric-label">Warnings</span><strong>{warningCount}</strong></div>
      </div>
      {qaRun.error ? <p className="status err">{qaRun.error}</p> : null}
      {warningCount > 0 ? <p className="status warn">Warnings: {(qaRun.warnings ?? []).join(", ")}</p> : null}
      <div className="actions qa-recovery-actions">
        <button type="button" data-testid="qa-retry-run-btn" onClick={onRetry} disabled={busy || !canRetry}>{retryLabel}</button>
        <a className="link-button" href={`/api/pipeline/runs/${encodeURIComponent(qaRun.run_id)}/logs`} target="_blank" rel="noreferrer">Open run logs</a>
      </div>
      <p className="hint">{retentionHint}</p>
    </section>
  );
}
