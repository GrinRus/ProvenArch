import { isRunCanceled, isRunReconciledAfterRestart, isRunnerUnavailable } from "./runState";
import { deriveWorkflowState, type PublicationState, type WorkflowState } from "./workflowState";
import type { GitDiffResponse } from "./appContracts";

export type AppEvidenceStatus = "idle" | "loading" | "available" | "partial" | "not_produced" | "unavailable" | "error";

export function derivePublicationState(input: {
  gitError?: string | null;
  gitDiffStatus?: string | null;
  gitDiff?: Pick<GitDiffResponse, "state"> | null;
}): PublicationState {
  if ((input.gitError ?? "").toLowerCase().includes("stale_git_confirmation")) return "stale";
  if (input.gitError) return "blocked";
  if ((input.gitDiffStatus ?? "").toLowerCase().startsWith("loading")) return "loading";
  if (!input.gitDiff) return "unknown";
  return input.gitDiff.state;
}

export function deriveAppWorkflowState(input: {
  workspace: "unconfigured" | "invalid" | "ready";
  runStatuses: readonly string[];
  selectedRunStatus?: string | null;
  evidenceStatus: AppEvidenceStatus;
  artifactCount: number;
  publication: PublicationState;
  openQuestions: string;
  demo?: boolean;
}): WorkflowState {
  const hasRunning = input.runStatuses.includes("running");
  const hasQueued = input.runStatuses.includes("queued");
  const evidence = input.evidenceStatus === "available"
    ? "snapshot"
    : input.evidenceStatus === "partial"
      ? "partial"
      : input.evidenceStatus === "not_produced" || input.evidenceStatus === "unavailable" || input.evidenceStatus === "error"
        ? "unavailable"
        : input.artifactCount > 0 ? "current" : "none";
  const execution = hasRunning
    ? "active"
    : hasQueued
      ? "pending"
      : input.selectedRunStatus === "failed"
        ? "failed"
        : input.selectedRunStatus === "succeeded"
          ? "succeeded"
          : "idle";
  return deriveWorkflowState({
    workspace: input.workspace,
    execution,
    evidence,
    publication: input.publication,
    openQuestions: input.openQuestions.split("\n").filter((line) => /^\s*[-*]\s+/.test(line)).length,
    demo: input.demo,
  });
}

export function selectedRunIssueCopy(
  errorCode: string,
  error: string | null | undefined,
  surface: "inspector" | "publish",
): { label: string; detail: string } {
  if (isRunCanceled(errorCode)) {
    return {
      label: "Canceled run",
      detail:
        surface === "publish"
          ? "run_canceled: select a successful run or start a new analysis before publishing."
          : "run_canceled: selected run was stopped by request; taskrun evidence remains in History.",
    };
  }
  if (isRunReconciledAfterRestart(errorCode)) {
    return {
      label: "Run reconciled after restart",
      detail:
        surface === "publish"
          ? "run_reconciled_after_restart: select a completed artifact run or start a new analysis before publishing."
          : "run_reconciled_after_restart: ACP preserved the stale run evidence in History after service restart.",
    };
  }
  if (isRunnerUnavailable(errorCode)) {
    return {
      label: "Provider unavailable",
      detail:
        surface === "publish"
          ? "runner_unavailable: check Readiness provider setup, binary/auth/quota, then run a successful analysis before publishing."
          : "runner_unavailable: check Readiness provider setup, binary/auth/quota, then retry the same analysis pipeline.",
    };
  }
  return {
    label: errorCode,
    detail: error || (surface === "publish" ? "Selected run failed before publication." : "Selected run failed."),
  };
}
