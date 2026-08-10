import { describe, expect, it } from "vitest";

import { buildReviewQueue, deriveReviewTrustStatus, findLastSuccessfulRun, reviewDecisionSummary } from "./reviewUtils";

const artifact = (path: string) => ({ path, kind: "markdown", label: path });

describe("review selectors", () => {
  it("prioritizes questions and findings while preserving deterministic queue order", () => {
    const queue = buildReviewQueue({
      artifacts: [
        artifact("reports/findings/orders.md"),
        artifact("reports/coverage/open-questions.md"),
        artifact("reports/as-is/overview.md"),
      ],
      openQuestions: "- confirm owner",
      coverageSummary: "coverage",
    });
    expect(queue.map((item) => item.kind)).toEqual(["question", "finding", "report"]);
  });

  it("selects the latest successful run without replacing the current run", () => {
    const selected = findLastSuccessfulRun([
      { run_id: "failed", pipeline: "architecture", status: "failed", started_at: "2026-08-10T10:00:00Z" },
      { run_id: "old", pipeline: "architecture", status: "succeeded", started_at: "2026-08-09T10:00:00Z" },
      { run_id: "new", pipeline: "architecture", status: "succeeded", started_at: "2026-08-10T09:00:00Z" },
    ], "new");
    expect(selected?.run_id).toBe("old");
  });

  it("makes open questions visible in trust status and summary", () => {
    const trust = deriveReviewTrustStatus({ artifactCount: 3, hasCoverage: true, findingsCount: 1, openQuestionCount: 2 });
    expect(trust.tone).toBe("warn");
    expect(reviewDecisionSummary(trust, 2)).toContain("2 open questions");
  });
});
