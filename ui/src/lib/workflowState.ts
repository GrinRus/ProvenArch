export type WorkflowStatus = "available" | "needs_review" | "blocked" | "complete";
export type WorkflowDestination = "setup" | "home" | "runs" | "knowledge" | "changes";

export type WorkflowStateInput = {
  workspace: "unconfigured" | "invalid" | "ready";
  execution: "idle" | "active" | "pending" | "failed" | "succeeded";
  evidence: "none" | "snapshot" | "current" | "partial" | "unavailable";
  publication: "clean" | "dirty" | "stale" | "blocked";
  openQuestions?: number;
  demo?: boolean;
};

export type WorkflowState = {
  status: WorkflowStatus;
  attention: string;
  nextAction: { destination: WorkflowDestination; label: string };
};

export function deriveWorkflowState(input: WorkflowStateInput): WorkflowState {
  if (input.workspace !== "ready") {
    return {
      status: "blocked",
      attention: input.workspace === "unconfigured" ? "Select and configure a workspace." : "Resolve workspace validation blockers.",
      nextAction: { destination: "setup", label: input.workspace === "unconfigured" ? "Configure workspace" : "Fix workspace" },
    };
  }
  if (input.execution === "active" || input.execution === "pending") {
    return {
      status: "available",
      attention: input.execution === "pending" ? "A refresh is waiting behind the active run." : "Analysis is running.",
      nextAction: { destination: "runs", label: "Open run progress" },
    };
  }
  if (input.execution === "failed") {
    return { status: "blocked", attention: "The selected run needs recovery.", nextAction: { destination: "runs", label: "Review run blocker" } };
  }
  if (input.evidence === "none" || input.evidence === "unavailable") {
    return { status: "blocked", attention: "No trustworthy evidence snapshot is available.", nextAction: { destination: "runs", label: "Open Runs" } };
  }
  if (input.evidence === "partial") {
    return { status: "needs_review", attention: "The selected evidence snapshot is partial.", nextAction: { destination: "knowledge", label: "Review partial evidence" } };
  }
  if ((input.openQuestions ?? 0) > 0) {
    return { status: "needs_review", attention: `${input.openQuestions} open question(s) require review.`, nextAction: { destination: "knowledge", label: "Review open questions" } };
  }
  if (input.publication === "stale" || input.publication === "blocked") {
    return { status: "blocked", attention: input.publication === "stale" ? "Git confirmation is stale." : "Publication has blockers.", nextAction: { destination: "changes", label: "Resolve publication blocker" } };
  }
  if (input.publication === "dirty") {
    return { status: "needs_review", attention: input.demo ? "Demo evidence is ready for an explicit publication review." : "Workspace changes are ready for review.", nextAction: { destination: "changes", label: "Review workspace changes" } };
  }
  return { status: "complete", attention: "Workspace evidence and publication state are current.", nextAction: { destination: "home", label: "View workspace summary" } };
}
