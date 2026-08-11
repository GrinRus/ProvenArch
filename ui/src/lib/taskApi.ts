import { fetchJSON } from "./api";

export type TaskRepositoryScope = { name: string; paths: string[] };
export type TaskScope = { repositories: TaskRepositoryScope[] };
export type TaskRunnerPreset = {
  preset: string;
  mode?: "fake" | "headless";
  provider?: "claude-code" | "qwen-code" | "codex-code";
  model?: string;
  effort?: string;
  permissions?: "trusted_full_access" | "managed";
};

export type ProductTask = {
  version: number;
  task_id: string;
  title: string;
  goal: string;
  context?: string;
  scope: TaskScope;
  desired_runner: TaskRunnerPreset;
  lifecycle: "open" | "archived";
  revision: number;
  created_at: string;
  updated_at: string;
  last_activity_at: string;
  attempts: Array<{ attempt_id: string; run_id: string; status: string; updated_at: string }>;
  outcome: { state: "available" | "unavailable"; unavailable_reason?: string };
  publication: { state: "linked" | "unavailable"; unavailable_reason?: string };
};

export type CreateTaskRequest = Pick<ProductTask, "title" | "goal" | "scope" | "desired_runner"> & { context?: string };

export async function createTask(request: CreateTaskRequest): Promise<ProductTask> {
  const response = await fetchJSON<{ task: ProductTask }>("/api/tasks", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  return response.task;
}
