import { isRunCanceled, isRunReconciledAfterRestart, isRunnerUnavailable } from "../../lib/runState";

export function failureRecoveryGuidance(errorCode: string, pendingPermissionCount: number): string {
  if (isRunCanceled(errorCode)) {
    return "The run stopped by request. Review retained taskrun evidence, then start a new run when ready.";
  }
  if (isRunReconciledAfterRestart(errorCode)) {
    return "ACP reconciled a stale run after restart. Inspect retained evidence, then start a new run if the previous work should continue.";
  }
  if (pendingPermissionCount > 0 || errorCode === "runtime_permission_required") {
    return "Resolve the pending permission request, then retry the same pipeline.";
  }
  if (isRunnerUnavailable(errorCode)) {
    return "Provider/tool availability blocked artifact creation; check Readiness provider setup, binary/auth/quota, then retry the same pipeline.";
  }
  if (errorCode.includes("runtime_timeout")) {
    return "The run exhausted its time budget; inspect the last progress signal before retry.";
  }
  if (errorCode.includes("runtime_contract")) {
    return "Generated artifacts did not pass validation; inspect the rejected step evidence before retry.";
  }
  if (errorCode.includes("infra") || errorCode.includes("incomplete")) {
    return "The cycle ended incomplete; review the last durable evidence before starting another run.";
  }
  return "Review the blocker details, then retry the same pipeline when the cause is clear.";
}

export function failureEvidenceSummary(artifactCount: number, issueCount: number): string {
  if (artifactCount > 0) {
    return `${artifactCount} artifact refs kept`;
  }
  if (issueCount > 0) {
    return `${issueCount} diagnostic rows`;
  }
  return "status and logs only";
}
