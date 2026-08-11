export type WorkflowStatus = "available" | "needs_review" | "blocked" | "complete";
export type WorkflowDestination = "setup" | "tasks" | "knowledge" | "changes" | "settings";
export type PublicationState = "clean" | "dirty" | "stale" | "blocked" | "loading" | "unknown";

export type WorkflowStateInput = {
  workspace: "unconfigured" | "invalid" | "ready";
  execution: "idle" | "active" | "pending" | "failed" | "succeeded";
  evidence: "none" | "snapshot" | "current" | "partial" | "unavailable";
  publication: PublicationState;
  openQuestions?: number;
  demo?: boolean;
};

export type WorkflowState = {
  status: WorkflowStatus;
  publication: PublicationState;
  attention: string;
  nextAction: { destination: WorkflowDestination; label: string };
};

export function deriveWorkflowState(input: WorkflowStateInput): WorkflowState {
  if (input.workspace !== "ready") {
    return {
      status: "blocked",
      publication: input.publication,
      attention: input.workspace === "unconfigured" ? "Select and configure a workspace." : "Resolve workspace validation blockers.",
      nextAction: { destination: "setup", label: input.workspace === "unconfigured" ? "Configure workspace" : "Fix workspace" },
    };
  }
  if (input.execution === "active" || input.execution === "pending") {
    return {
      status: "available",
      publication: input.publication,
      attention: input.execution === "pending" ? "A refresh is waiting behind the active run." : "Analysis is running.",
      nextAction: { destination: "tasks", label: "Open Task progress" },
    };
  }
  if (input.execution === "failed") {
    return { status: "blocked", publication: input.publication, attention: "The selected Attempt needs recovery.", nextAction: { destination: "tasks", label: "Review Attempt" } };
  }
  if (input.evidence === "none" || input.evidence === "unavailable") {
    return { status: "blocked", publication: input.publication, attention: "No trustworthy evidence snapshot is available.", nextAction: { destination: "tasks", label: "Open diagnostics" } };
  }
  if (input.evidence === "partial") {
    return { status: "needs_review", publication: input.publication, attention: "The selected evidence snapshot is partial.", nextAction: { destination: "knowledge", label: "Review partial evidence" } };
  }
  if ((input.openQuestions ?? 0) > 0) {
    return { status: "needs_review", publication: input.publication, attention: `${input.openQuestions} open question(s) require review.`, nextAction: { destination: "knowledge", label: "Review open questions" } };
  }
  if (input.publication === "loading") {
    return { status: "available", publication: input.publication, attention: "Checking the workspace Git publication state.", nextAction: { destination: "changes", label: "Check publication state" } };
  }
  if (input.publication === "unknown") {
    return { status: "blocked", publication: input.publication, attention: "Workspace Git publication state is unavailable.", nextAction: { destination: "changes", label: "Check publication state" } };
  }
  if (input.publication === "stale" || input.publication === "blocked") {
    return { status: "blocked", publication: input.publication, attention: input.publication === "stale" ? "Git confirmation is stale." : "Publication has blockers.", nextAction: { destination: "changes", label: "Resolve publication blocker" } };
  }
  if (input.publication === "dirty") {
    return { status: "needs_review", publication: input.publication, attention: input.demo ? "Demo evidence is ready for an explicit publication review." : "Workspace changes are ready for review.", nextAction: { destination: "changes", label: "Review workspace changes" } };
  }
  return { status: "complete", publication: input.publication, attention: "Workspace evidence and publication state are current.", nextAction: { destination: "tasks", label: "View Task Inbox" } };
}
