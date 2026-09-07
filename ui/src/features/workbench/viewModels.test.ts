import { describe, expect, it } from "vitest";

import type { ProductTask } from "../../lib/taskApi";
import { buildChangeReviewModel } from "./viewModels";

describe("workbench view models", () => {
  it("routes only authoritative successful analysis runs to Change Review", () => {
    const runs = [
      { run_id: "ok", pipeline: "refresh", status: "succeeded", authoritative_index: true },
      { run_id: "failed", pipeline: "init", status: "failed", authoritative_index: false },
      { run_id: "qa", pipeline: "qa", status: "succeeded", authoritative_index: false },
    ] as never;
    const tasks = [
      { task_id: "task-ok", title: "Refresh payments", goal: "Refresh the architecture", updated_at: "2026-07-15T00:00:00Z", outcome: { state: "available", attempt_id: "attempt-ok", run_id: "ok" } },
      { task_id: "task-failed", title: "Recover payments", goal: "Recover the failed refresh", updated_at: "2026-07-15T00:00:00Z", outcome: { state: "available", attempt_id: "attempt-failed", run_id: "failed" } },
    ] as ProductTask[];
    expect(buildChangeReviewModel(tasks, runs).map((item) => [item.task_id, item.run_id, item.action])).toEqual([
      ["task-ok", "ok", "review"], ["task-failed", "failed", "run_studio"],
    ]);
  });
});
