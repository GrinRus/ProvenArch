import { isRunCanceled, isRunReconciledAfterRestart, isRunnerUnavailable } from "../../lib/runState";
import type { QARunResponse } from "../../lib/qaApi";

export function buildProvisionalQARun(started: { run_id: string; status: string }, question: string): QARunResponse {
  return {
    run_id: started.run_id,
    pipeline: "qa",
    status: normalizeQARunStatus(started.status),
    started_at: new Date().toISOString(),
    finished_at: null,
    question,
    current_step: "qa.ask",
    answer: null,
    citations: [],
    unresolved: [],
    confidence: null,
    generated_at: null,
    warnings: [],
    error_code: null,
    error: null,
  };
}

export function normalizeQARunStatus(status: string): QARunResponse["status"] {
  if (status === "queued" || status === "running" || status === "succeeded" || status === "failed") {
    return status;
  }
  return "queued";
}

export function qaErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

export function qaFailureGuidance(errorCode: string, warningCount: number): string {
  if (isRunCanceled(errorCode)) {
    return "The answer run stopped by request. Review QA audit artifacts, then ask again when ready.";
  }
  if (isRunReconciledAfterRestart(errorCode)) {
    return "ACP reconciled a stale answer run after restart. Review QA audit artifacts, then ask again if the question still matters.";
  }
  if (errorCode === "runtime_permission_required") {
    return "Resolve the runtime permission request, then retry the question.";
  }
  if (isRunnerUnavailable(errorCode)) {
    return "Provider/tool availability blocked the answer; check Readiness provider setup, binary/auth/quota, then ask again.";
  }
  if (errorCode.includes("runtime_timeout")) {
    return "The answer run exhausted its time budget; inspect the last progress signal before retry.";
  }
  if (errorCode.includes("runtime_contract")) {
    return "The answer artifact did not pass validation; inspect audit artifacts before retry.";
  }
  if (warningCount > 0) {
    return "Review warnings and audit artifacts, then retry when the issue is understood.";
  }
  return "Review logs and audit artifacts, then retry the same question when the cause is clear.";
}

export function qaRunProviderLabel(run: QARunResponse | null): string {
  if (!run) {
    return "agent-backed";
  }
  if (run.provider === "fake") {
    return "fake";
  }
  return run.provider || run.runtime_provider || "agent-backed";
}

export function mergeQARunHistory(run: QARunResponse, history: QARunResponse[], mode: "prepend" | "preserve" = "prepend"): QARunResponse[] {
  if (history.some((item) => item.run_id === run.run_id)) {
    return history.map((item) => (item.run_id === run.run_id ? { ...item, ...run } : item)).slice(0, 20);
  }
  if (mode === "preserve") {
    return [...history, run].slice(0, 20);
  }
  return [run, ...history].slice(0, 20);
}
