import type { RunListItem } from "../../lib/appContracts";
import type { ProductTask } from "../../lib/taskApi";

export type ChangeReviewItem = RunListItem & {
  action: "review" | "run_studio";
  publication: "unknown";
  task_id: string;
  task_title: string;
  task_goal: string;
  attempt_id: string;
};

export function buildChangeReviewModel(tasks: ProductTask[], runs: RunListItem[]): ChangeReviewItem[] {
  return tasks
    .filter((task) => task.outcome.state === "available" && Boolean(task.outcome.run_id && task.outcome.attempt_id))
    .map((task) => {
      const runID = task.outcome.run_id as string;
      const attemptID = task.outcome.attempt_id as string;
      const run = runs.find((item) => item.run_id === runID);
      return {
        ...(run ?? {
          run_id: runID,
          pipeline: "init",
          status: "succeeded" as const,
          started_at: task.updated_at,
          authoritative_index: true,
        }),
        task_id: task.task_id,
        task_title: task.title,
        task_goal: task.goal,
        attempt_id: attemptID,
        action: run && (run.status !== "succeeded" || run.authoritative_index !== true)
          ? "run_studio" as const
          : "review" as const,
        publication: "unknown" as const,
      };
    });
}
