import { describe, expect, it } from "vitest";

import { deriveWorkflowState, type WorkflowStateInput } from "./workflowState";

const ready: WorkflowStateInput = {
  workspace: "ready",
  execution: "succeeded",
  evidence: "snapshot",
  publication: "clean",
};

describe("deriveWorkflowState", () => {
  it.each([
    [{ ...ready, workspace: "invalid" as const }, "setup", "blocked"],
    [{ ...ready, execution: "active" as const }, "runs", "available"],
    [{ ...ready, execution: "pending" as const }, "runs", "available"],
    [{ ...ready, evidence: "unavailable" as const }, "runs", "blocked"],
    [{ ...ready, evidence: "partial" as const }, "knowledge", "needs_review"],
    [{ ...ready, publication: "stale" as const }, "changes", "blocked"],
    [{ ...ready, publication: "blocked" as const }, "changes", "blocked"],
    [{ ...ready, publication: "dirty" as const }, "changes", "needs_review"],
  ])("maps independent workflow axes to one next action", (input, destination, status) => {
    const result = deriveWorkflowState(input);
    expect(result.nextAction.destination).toBe(destination);
    expect(result.status).toBe(status);
  });

  it("labels dirty fake evidence as an explicit demo publication review", () => {
    expect(deriveWorkflowState({ ...ready, publication: "dirty", demo: true }).attention).toContain("Demo evidence");
  });
});
