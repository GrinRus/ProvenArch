import { StatusBadge } from "./ConsolePrimitives";
import type { RunReviewSummaryResponse, RunStatusResponse } from "../lib/appContracts";
import { runReviewErrorCount, runReviewWarningCount } from "../lib/runReviewMetrics";

type ActiveRunStripProps = {
  runStatus: RunStatusResponse | null;
  reviewSummary: RunReviewSummaryResponse | null;
  runtimeLabel: string;
  cancelBusy: boolean;
  selectedRunIsActive: boolean;
  onCancel: () => void;
  onOpenAnalysis: () => void;
};

export function ActiveRunStrip({
  runStatus,
  reviewSummary,
  runtimeLabel,
  cancelBusy,
  selectedRunIsActive,
  onCancel,
  onOpenAnalysis,
}: ActiveRunStripProps) {
  const steps = reviewSummary?.steps ?? [];
  const activeStep = steps.find((step) => step.state === "active" || step.state === "failed");
  const doneCount = steps.filter((step) => step.state === "done").length;
  const warningCount = runReviewWarningCount(runStatus, reviewSummary);
  const errorCount = runReviewErrorCount(runStatus, reviewSummary);
  const status = runStatus?.status ?? reviewSummary?.status ?? "idle";
  const runID = runStatus?.run_id ?? reviewSummary?.run_id;
  const currentStep = runStatus?.current_step ?? reviewSummary?.current_step ?? activeStep?.step_id ?? "not running";
  const artifactCount = steps.reduce((sum, step) => sum + step.artifact_count, 0);
  const isSucceeded = status === "succeeded";
  const runContext = reviewSummary?.pipeline ?? runStatus?.pipeline ?? "pipeline";
  const primaryStatLabel = isSucceeded ? "Review state" : "Current step";
  const primaryStatValue = isSucceeded ? (artifactCount > 0 ? `${artifactCount} artifacts ready` : "review ready") : currentStep;
  const elapsed = elapsedLabel(runStatus?.started_at ?? reviewSummary?.started_at, runStatus?.finished_at ?? reviewSummary?.finished_at);
  const showCancelGuidance = selectedRunIsActive && (status === "queued" || status === "running");

  return (
    <section className="active-run-strip" data-testid="active-run-strip" aria-label="active run summary">
      <div className="active-run-main">
        <StatusBadge tone={status === "succeeded" ? "ok" : status === "failed" ? "error" : status === "running" || status === "queued" ? "warn" : "info"}>
          {status}
        </StatusBadge>
        <div>
          <strong>{runID || "No run selected"}</strong>
          <span>
            {runContext} · {runtimeLabel}
            {isSucceeded ? " · evidence ready for review" : ""}
          </span>
        </div>
      </div>
      <div className="active-run-stat">
        <span className="metric-label">{primaryStatLabel}</span>
        <strong>{primaryStatValue}</strong>
      </div>
      <div className="active-run-stat">
        <span className="metric-label">{isSucceeded ? "Steps complete" : "Progress"}</span>
        <strong>
          {steps.length > 0 ? `${doneCount}/${steps.length}` : "0/5"} steps
        </strong>
      </div>
      <div className="active-run-stat">
        <span className="metric-label">Elapsed</span>
        <strong>{elapsed}</strong>
      </div>
      <div className="active-run-stat">
        <span className="metric-label">Warnings / errors</span>
        <strong>
          {warningCount} / {errorCount}
        </strong>
      </div>
      <div className="active-run-actions">
        <div className="active-run-action-buttons">
          <button type="button" className="link-button" onClick={onOpenAnalysis}>
            Open Analysis
          </button>
          <button type="button" onClick={onCancel} disabled={!runID || !selectedRunIsActive || cancelBusy}>
            {cancelBusy ? "Canceling" : "Cancel"}
          </button>
        </div>
        {showCancelGuidance ? (
          <p className="active-run-cancel-note" data-testid="active-run-cancel-guidance">
            Cancel requests a cooperative stop; taskrun evidence stays in History.
          </p>
        ) : null}
      </div>
    </section>
  );
}

function elapsedLabel(startedAt?: string | null, finishedAt?: string | null): string {
  if (!startedAt) {
    return "not started";
  }
  const start = Date.parse(startedAt);
  const end = finishedAt ? Date.parse(finishedAt) : Date.now();
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) {
    return "unknown";
  }
  const totalSeconds = Math.max(0, Math.round((end - start) / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes === 0) {
    return `${seconds}s`;
  }
  return `${minutes}m ${seconds.toString().padStart(2, "0")}s`;
}
