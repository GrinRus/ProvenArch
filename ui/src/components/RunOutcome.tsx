import { useEffect, useState } from "react";

import type { RetryPlanResponse, RunReviewSummaryResponse, RunStatusResponse } from "../lib/appContracts";
import { calculateRetryPlan, startTargetedRetry } from "../lib/runApi";
import { StatusBadge } from "./ConsolePrimitives";

export function StructuredRunProgress({ runStatus, review, onReviewDetails }: { runStatus: RunStatusResponse | null; review: RunReviewSummaryResponse | null; onReviewDetails?: () => void }) {
  const progress = runStatus?.progress ?? review?.progress;
  if (!runStatus) return <section className="run-progress-empty" data-testid="analysis-run-progress"><h2>Runtime progress</h2><p>No run is selected.</p></section>;
  const total = progress?.total_steps || review?.steps?.length || 0;
  const reportedCompleted = progress?.completed_steps ?? review?.steps?.filter((step) => step.state === "done").length ?? 0;
  const completed = !progress && runStatus.status === "failed" && total > 0 ? Math.min(reportedCompleted, total - 1) : reportedCompleted;
  const phase = progress?.phase || legacyPresentationPhase(runStatus) || "queued";
  return <section className="structured-run-progress" data-testid="analysis-run-progress">
    <div className="section-heading-row"><div><p className="eyebrow">Runtime progress · {runStatus.run_id || "selected run"}</p><h2>{progress?.expected_result || stepPurpose(runStatus.current_step)}</h2></div><StatusBadge tone={phaseTone(phase)}>{phase.replace(/_/g, " ")}</StatusBadge></div>
    <div className="step-progress-track" role="progressbar" aria-valuemin={0} aria-valuemax={Math.max(total, 1)} aria-valuenow={completed} aria-label="Completed pipeline steps" style={{ gridTemplateColumns: `repeat(${Math.max(total, 1)}, minmax(0, 1fr))` }}>{total > 0 ? Array.from({ length: total }, (_, index) => <span key={index} className={index < completed ? "is-complete" : index === completed && !["succeeded", "failed", "canceled"].includes(runStatus.status) ? "is-current" : "is-pending"} aria-hidden="true" />) : <span className="is-pending" aria-hidden="true" />}</div>
    <div className="run-progress-facts"><div><span>Pipeline steps</span><strong>{completed}/{total || "—"}</strong></div><div><span>Current step</span><strong>{progress?.current_step || runStatus.current_step || terminalLabel(runStatus.status)}</strong></div><div><span>Scope</span><strong>{progress?.current_scopes?.join(", ") || "Whole configured workspace"}</strong></div><div><span>Useful artifact state</span><strong>{progress?.artifact_state || "Waiting for structured telemetry"}</strong></div></div>
    <div className="runtime-activity-row"><span>Elapsed: {typeof progress?.elapsed_ms === "number" ? formatDuration(progress.elapsed_ms) : formatElapsed(progress?.started_at || runStatus.started_at, runStatus.finished_at)}</span><span>Provider activity: {formatTime(progress?.last_activity_at)}</span><span>Last useful progress: {formatTime(progress?.last_progress_at)}</span>{progress?.planned_units ? <span>Units: {progress.succeeded_units ?? 0} complete · {progress.running_units ?? 0} running · {Math.max(0, progress.planned_units - (progress.succeeded_units ?? 0) - (progress.running_units ?? 0) - (progress.failed_units ?? 0))} pending · {progress.failed_units ?? 0} failed</span> : <span>Opaque provider work has no artificial percentage.</span>}{progress?.repair_attempt ? <span>Repair attempt: {progress.repair_attempt}/{progress.repair_limit || "—"}</span> : null}{progress?.stall_deadline_at ? <span>Stall deadline: {formatTime(progress.stall_deadline_at)}</span> : null}</div>
    {onReviewDetails && (runStatus.status === "failed" || Boolean(runStatus.error_code)) ? <button type="button" data-testid="analysis-review-blocker-btn" onClick={onReviewDetails}>Review error details</button> : null}
  </section>;
}

export function RunResultPanel({ review, onExploreArchitecture }: { review: RunReviewSummaryResponse | null; onExploreArchitecture?: () => void }) {
  const result = review?.result;
  if (!result || !["succeeded", "failed", "canceled"].includes(review.status)) return null;
  const produced = Object.entries(result.produced).filter(([, value]) => value > 0);
  return <section className={`run-result outcome-${result.state}`} data-testid="run-result-panel">
    <div className="run-result-copy"><p className="eyebrow">Run result</p><h2>{outcomeTitle(result.state)}</h2><p>{result.summary}</p><p className="promotion-statement">{result.promotion.changed ? "The validator-approved current architecture was updated." : result.promotion.current_usable ? "The existing validated architecture remains current." : "No validated current architecture is available yet."}</p></div>
    <dl className="run-result-counts">{produced.map(([label, value]) => <div key={label}><dt>{label.replace(/_/g, " ")}</dt><dd>{value}</dd></div>)}{result.coverage ? <><div><dt>observed scopes</dt><dd>{result.coverage.observed}</dd></div><div><dt>coverage gaps</dt><dd>{result.coverage.missing}</dd></div></> : null}<div><dt>partial scopes</dt><dd>{result.partial_scopes}</dd></div><div><dt>failed scopes</dt><dd>{result.failed_scopes}</dd></div>{result.promotion.baseline_run_id ? <div><dt>previous baseline</dt><dd className="run-id-value">{result.promotion.baseline_run_id}</dd></div> : null}</dl>
    <div className="run-result-action"><strong>Recommended next action</strong><span>{actionLabel(result.recommended_action)}</span>{onExploreArchitecture && result.promotion.current_usable ? <button type="button" onClick={onExploreArchitecture}>Explore architecture</button> : null}</div>
  </section>;
}

export function TargetedRerunPanel({ runStatus, review, busy, onRetryStarted }: { runStatus: RunStatusResponse | null; review: RunReviewSummaryResponse | null; busy: boolean; onRetryStarted: (runID: string) => void }) {
  const steps = terminalRerunSteps(review?.pipeline || runStatus?.pipeline);
  const [step, setStep] = useState(steps[steps.length - 1] ?? "");
  const [plan, setPlan] = useState<RetryPlanResponse | null>(null);
  const [status, setStatus] = useState("");
  const [working, setWorking] = useState(false);
	useEffect(() => { setStep(steps[steps.length - 1] ?? ""); setPlan(null); setStatus(""); }, [review?.pipeline, runStatus?.run_id]);
  if (!runStatus || runStatus.status !== "succeeded" || steps.length === 0) return null;
  async function calculate() { setWorking(true); setStatus(""); try { setPlan(await calculateRetryPlan(runStatus!.run_id, step)); } catch (error) { setStatus(error instanceof Error ? error.message : "Rerun planning failed"); } finally { setWorking(false); } }
  async function start() { if (!plan) return; setWorking(true); setStatus(""); try { const response = await startTargetedRetry(runStatus!.run_id, plan); onRetryStarted(response.run_id); } catch (error) { const message = error instanceof Error ? error.message : "Targeted rerun failed to start"; if (message.toLowerCase().includes("retry inputs changed")) { setPlan(null); setStatus("The rerun plan is stale because sources or parent artifacts changed. Calculate it again, or start a full run if you intended to change the source baseline."); } else { setStatus(message); } } finally { setWorking(false); } }
  return <section className="targeted-rerun" data-testid="targeted-rerun-panel">
    <div><p className="eyebrow">Selective rerun</p><h2>Repeat only the work you need</h2><p>Choose a completed pipeline step. ProvenArch will create an auditable child run and automatically include every invalidated downstream step.</p></div>
    <label>Start from step<select value={step} onChange={(event) => { setStep(event.target.value); setPlan(null); setStatus(""); }}>{steps.map((item) => <option key={item} value={item}>{stepPurpose(item)}</option>)}</select></label>
    {plan ? <RetryPlanPreview plan={plan} /> : null}
    {status ? <p className="status err" role="status">{status}</p> : null}
    <div className="actions"><button type="button" onClick={() => void (plan ? start() : calculate())} disabled={busy || working}>{working ? "Working…" : plan ? "Start targeted rerun" : "Review rerun plan"}</button></div>
  </section>;
}

export function RecoveryPanel({ runStatus, review, busy, onRetryStarted, onReviewDetails }: { runStatus: RunStatusResponse | null; review: RunReviewSummaryResponse | null; busy: boolean; onRetryStarted: (runID: string) => void; onReviewDetails: () => void }) {
  const [plan, setPlan] = useState<RetryPlanResponse | null>(null);
  const [status, setStatus] = useState("");
  const [working, setWorking] = useState(false);
  const recovery = review?.recovery ?? legacyRecovery(runStatus);
  if (!runStatus || !recovery || !["failed", "canceled"].includes(runStatus.status)) return null;
  async function calculate() { setWorking(true); setStatus(""); try { setPlan(await calculateRetryPlan(runStatus!.run_id, recovery?.failed_step, recovery?.failed_scopes)); } catch (error) { setStatus(error instanceof Error ? error.message : "Retry planning failed"); } finally { setWorking(false); } }
  async function start() { if (!plan) return; setWorking(true); setStatus(""); try { const response = await startTargetedRetry(runStatus!.run_id, plan); onRetryStarted(response.run_id); } catch (error) { const message = error instanceof Error ? error.message : "Targeted retry failed to start"; if (message.toLowerCase().includes("retry inputs changed")) { setPlan(null); setStatus("The retry plan is stale because sources or parent artifacts changed. Calculate it again, or start a full run if you intended to change the source baseline."); } else { setStatus(message); } } finally { setWorking(false); } }
  return <section className="structured-recovery" data-testid="analysis-failure-recovery">
    <div className="section-heading-row"><div><p className="eyebrow">Recovery</p><h2>{recovery.title}</h2></div><StatusBadge tone="error">{recovery.category}</StatusBadge></div>
    <p className="recovery-explanation">{recovery.explanation || "No additional provider explanation was recorded."}</p>
    <dl className="recovery-impact"><div><dt>Impact</dt><dd>{recovery.impact}</dd></div><div><dt>Evidence retained</dt><dd>{recovery.retained_evidence}</dd></div><div><dt>Recommended fix</dt><dd>{recovery.recommended_fix}</dd></div><div><dt>Failed step</dt><dd>{recovery.failed_step || "Not identified"}</dd></div></dl>
    {plan ? <RetryPlanPreview plan={plan} /> : null}
    {status ? <p className="status err" role="status">{status}</p> : null}
    <div className="actions"><button type="button" data-testid="analysis-retry-run-btn" onClick={() => void (plan ? start() : calculate())} disabled={busy || working || !recovery.can_retry}>{working ? "Working…" : plan ? "Start targeted retry" : "Calculate retry plan"}</button><button type="button" className="secondary" data-testid="analysis-review-recovery-btn" onClick={onReviewDetails}>Open technical details</button></div>
    {!recovery.can_retry ? <p className="disabled-reason">Retry is unavailable because the backend could not establish a safe dependency closure.</p> : null}
    <details><summary>Technical error</summary><code>{recovery.technical_code || runStatus.error_code || "unclassified"}</code></details>
  </section>;
}

function RetryPlanPreview({ plan }: { plan: RetryPlanResponse }) {
  return <div className="retry-plan" data-testid="retry-plan"><h3>Safe retry plan</h3>{plan.widened ? <p className="status warn">{plan.widen_reason}</p> : null}<div className="retry-plan-columns"><div><span>Reuse</span><strong>{plan.reused_inputs.length ? plan.reused_inputs.join(" → ") : "Nothing from this attempt"}</strong></div><div><span>Execute</span><strong>{plan.execute_steps.join(" → ")}</strong></div><div><span>Effective scope</span><strong>{plan.effective_scopes?.length ? plan.effective_scopes.join(", ") : "Whole configured workspace"}</strong></div></div><p>{plan.estimated_units} pipeline step(s) will run in a new auditable child run.</p></div>;
}

function outcomeTitle(state: NonNullable<RunReviewSummaryResponse["result"]>["state"]) { return state === "completed" ? "Architecture updated" : state === "completed_with_gaps" ? "Architecture updated with gaps" : state === "canceled" ? "Analysis canceled" : "Analysis needs recovery"; }
function actionLabel(value: string) { return value.replace(/_/g, " ").replace(/^./, (letter: string) => letter.toUpperCase()); }
function terminalLabel(status: string) { return status === "succeeded" ? "Completed" : status === "failed" ? "Stopped" : status === "canceled" ? "Canceled" : "Not started"; }
function phaseTone(phase: string): "ok" | "warn" | "error" | "info" { return phase === "completed" || phase === "succeeded" ? "ok" : phase === "failed" ? "error" : phase === "stalled" || phase === "repairing" ? "warn" : "info"; }
function formatTime(value?: string) { if (!value) return "not reported"; const date = new Date(value); return Number.isNaN(date.getTime()) ? value : date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" }); }
function formatElapsed(start?: string, finish?: string | null) { const started = start ? new Date(start).getTime() : Number.NaN; const finished = finish ? new Date(finish).getTime() : Date.now(); if (!Number.isFinite(started) || !Number.isFinite(finished) || finished < started) return "not available"; const seconds = Math.floor((finished - started) / 1000); const hours = Math.floor(seconds / 3600); const minutes = Math.floor((seconds % 3600) / 60); const remainder = seconds % 60; return hours > 0 ? `${hours}h ${minutes}m` : minutes > 0 ? `${minutes}m ${remainder}s` : `${remainder}s`; }
function formatDuration(milliseconds: number) { const seconds = Math.max(0, Math.floor(milliseconds / 1000)); const hours = Math.floor(seconds / 3600); const minutes = Math.floor((seconds % 3600) / 60); const remainder = seconds % 60; return hours > 0 ? `${hours}h ${minutes}m` : minutes > 0 ? `${minutes}m ${remainder}s` : `${remainder}s`; }
function stepPurpose(step?: string) { if (!step) return "Waiting to start"; if (step.includes("collect")) return "Collecting repository evidence"; if (step.includes("asis")) return "Building architecture model and diagrams"; if (step.includes("findings")) return "Validating findings and coverage"; if (step.includes("proposals")) return "Preparing proposals and promotion"; return "Establishing architecture scope"; }
function terminalRerunSteps(pipeline?: string) { return pipeline === "refresh" ? ["refresh.step1.collect", "refresh.step2.asis_docs", "refresh.step3.findings", "refresh.step4.proposals"] : pipeline === "init" ? ["init.step0.constitution", "init.step1.collect", "init.step2.asis_docs", "init.step3.findings", "init.step4.proposals"] : []; }

function legacyPresentationPhase(runStatus: RunStatusResponse) {
  if (runStatus.error_code === "run_canceled") return "canceled";
  if (runStatus.error_code === "run_reconciled_after_restart") return "recovered";
  return runStatus.status;
}

function legacyRecovery(runStatus: RunStatusResponse | null): NonNullable<RunReviewSummaryResponse["recovery"]> | null {
  if (!runStatus || !["failed", "canceled"].includes(runStatus.status)) return null;
  const code = runStatus.error_code || "unclassified";
  const canceled = runStatus.status === "canceled" || code === "run_canceled";
  const reconciled = code === "run_reconciled_after_restart";
  const contract = code.includes("contract") || code.includes("validation");
  const permission = code.includes("permission");
  return {
    category: canceled ? "canceled" : code.includes("permission") ? "permission" : code.includes("runner") || code.includes("provider") ? "provider" : code.includes("timeout") ? "timeout" : "infrastructure",
    title: canceled ? "Canceled run" : reconciled ? "Recovered after restart" : "Recovery path",
    explanation: canceled ? "The run stopped by request. The last-good architecture was not replaced." : reconciled ? "ACP reconciled a stale run after restart. The last-good architecture was not replaced." : contract ? `Generated artifacts did not pass validation. ${runStatus.error || "The run stopped before a validator-approved result could be promoted."}` : runStatus.error || "The run stopped before a validator-approved result could be promoted.",
    impact: "This attempt did not replace the last-good promoted architecture.",
    retained_evidence: "Validated taskrun evidence remains attached to this immutable run and may be reused by a safe retry plan.",
    recommended_fix: canceled ? "Confirm the intended scope, then calculate a targeted retry plan." : permission ? "Resolve the pending permission request, then calculate a targeted retry plan." : contract ? "Inspect the invalid artifact and provider output, correct the contract cause, then calculate a targeted retry plan." : "Resolve the reported cause, then calculate a targeted retry plan.",
    can_retry: code !== "unclassified",
    failed_step: runStatus.current_step,
    technical_code: code,
  };
}
