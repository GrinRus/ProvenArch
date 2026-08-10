import { describe, expect, it } from "vitest";

import { deriveAppWorkflowState, derivePublicationState } from "./appDerived";

describe("app-derived selectors", () => {
  it("keeps publication unknown until an authoritative diff exists", () => {
    expect(derivePublicationState({ gitDiffStatus: "idle" })).toBe("unknown");
    expect(derivePublicationState({ gitDiffStatus: "loading" })).toBe("loading");
    expect(derivePublicationState({ gitDiff: { state: "clean" } })).toBe("clean");
  });

  it("maps active and queued runs to the run progress action", () => {
    const workflow = deriveAppWorkflowState({
      workspace: "ready",
      runStatuses: ["succeeded", "running"],
      selectedRunStatus: "succeeded",
      evidenceStatus: "available",
      artifactCount: 2,
      publication: "unknown",
      openQuestions: "",
    });
    expect(workflow.nextAction.destination).toBe("runs");
    expect(workflow.attention).toBe("Analysis is running.");
  });

  it("counts only bullet questions for review state", () => {
    const workflow = deriveAppWorkflowState({
      workspace: "ready",
      runStatuses: ["succeeded"],
      selectedRunStatus: "succeeded",
      evidenceStatus: "available",
      artifactCount: 1,
      publication: "clean",
      openQuestions: "Context\n- Confirm owner\nnotes\n* Confirm SLO",
    });
    expect(workflow.status).toBe("needs_review");
    expect(workflow.attention).toBe("2 open question(s) require review.");
  });
});
