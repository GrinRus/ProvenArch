import { describe, expect, it } from "vitest";

import type { RunReviewSummaryResponse, RunStatusResponse } from "./appContracts";
import { runReviewErrorCount, runReviewWarningCount } from "./runReviewMetrics";

describe("runReviewMetrics", () => {
  it("deduplicates run-level warnings and adds unique step warning counts", () => {
    const runStatus = {
      run_id: "run-1",
      pipeline: "init",
      status: "succeeded",
      started_at: "2026-06-03T10:00:00Z",
      warnings: ["duplicate warning", "status-only warning"],
      error_code: null,
      error: null,
    } satisfies RunStatusResponse;
    const reviewSummary = {
      run_id: "run-1",
      pipeline: "init",
      status: "succeeded",
      started_at: "2026-06-03T10:00:00Z",
      finished_at: "2026-06-03T10:00:01Z",
      warnings: ["duplicate   warning"],
      error_code: null,
      error: null,
      steps: [
        {
          step_id: "init.step1.collect",
          key: "step1_collect",
          label: "Collect",
          state: "done",
          provider: "fake",
          artifact_count: 0,
          artifact_paths: [],
          taskrun_paths: [],
          warnings_count: 1,
          errors_count: 0,
        },
      ],
    } satisfies RunReviewSummaryResponse;

    expect(runReviewWarningCount(runStatus, reviewSummary)).toBe(3);
    expect(runReviewErrorCount(runStatus, reviewSummary)).toBe(0);
  });
});
