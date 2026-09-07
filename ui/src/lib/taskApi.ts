import { fetchJSON } from "./api";
import type { TaskFilters } from "./appRoutes";

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
  outcome: { state: "available" | "unavailable"; unavailable_reason?: string; attempt_id?: string; run_id?: string; snapshot_path?: string };
  publication: { state: "linked" | "unavailable"; attempt_id?: string; run_id?: string; action?: "commit" | "branch" | "pull_request"; branch?: string; base_ref?: string; base_oid?: string; head_oid?: string; commit?: string; inventory_fingerprint?: string; unavailable_reason?: string };
};

export type TaskAttempt = {
  version: number;
  attempt_id: string;
  task_id: string;
  run_id: string;
  parent_attempt_id?: string;
  retry_reason?: string;
  pipeline: string;
  status: string;
  task_revision: number;
  intent_snapshot: {
    title: string;
    goal: string;
    context?: string;
    scope: TaskScope;
    desired_runner: TaskRunnerPreset;
  };
  effective_runtime: {
    mode?: string;
    provider?: string;
    model?: string;
    effort?: string;
    permissions?: string;
    timeouts?: Record<string, number>;
    scope?: TaskScope;
    execution?: {
      strategy?: string;
      max_parallel?: number;
      failure_policy?: string;
      shard_mode?: string;
    };
    step_overrides?: Record<string, TaskRunnerPreset>;
    resolution_sources?: Record<string, string>;
  };
  admitted_at: string;
  queued_at: string | null;
  started_at: string | null;
  finished_at: string | null;
  terminal_summary: { status: string; error_code?: string; error?: string; summary?: string; retained_evidence: string } | null;
  outcome: ProductTask["outcome"];
  retained_evidence: string;
  publication: ProductTask["publication"];
};

export type TaskListResponse = {
  items: ProductTask[];
  next_cursor: string;
  has_more: boolean;
  diagnostics?: { state?: string; message?: string };
};

export type TaskAttemptListResponse = {
  items: TaskAttempt[];
  diagnostics?: { state?: string; message?: string };
};

export type CreateTaskRequest = Pick<ProductTask, "title" | "goal" | "scope" | "desired_runner"> & { context?: string };
export type AdmitTaskAttemptRequest = { pipeline?: string; intent?: "start" | "queue" };
export type TaskAttemptActionRequest = { pipeline?: string; intent?: "start" | "queue"; reason?: string; idempotencyKey?: string };

export async function createTask(request: CreateTaskRequest): Promise<ProductTask> {
  const response = await fetchJSON<{ task: ProductTask }>("/api/tasks", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  return response.task;
}

export async function admitTaskAttempt(taskId: string, request: AdmitTaskAttemptRequest & { idempotencyKey?: string } = {}): Promise<TaskAttempt> {
  const response = await fetchJSON<{ attempt: TaskAttempt }>(`/api/tasks/${encodeURIComponent(taskId)}/attempts`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      idempotency_key: request.idempotencyKey ?? newIdempotencyKey(),
      pipeline: request.pipeline ?? "init",
      intent: request.intent ?? "start",
    }),
  });
  if (!response.attempt?.attempt_id) throw new Error("Attempt API returned no Attempt identity");
  return response.attempt;
}

export async function listTasks(filters: TaskFilters = {}, cursor = "", signal?: AbortSignal): Promise<TaskListResponse> {
  const params = new URLSearchParams({ limit: "100" });
  if (cursor) params.set("cursor", cursor);
  if (filters.lifecycle) params.set("lifecycle", filters.lifecycle);
  if (filters.runner) params.set("runner", filters.runner);
  if (filters.repository) params.set("repository", filters.repository);
  if (filters.from) params.set("from", filters.from);
  if (filters.to) params.set("to", filters.to);
  return fetchJSON<TaskListResponse>(`/api/tasks?${params.toString()}`, { signal });
}

export async function getTask(taskId: string, signal?: AbortSignal): Promise<ProductTask> {
  const response = await fetchJSON<{ task: ProductTask }>(`/api/tasks/${encodeURIComponent(taskId)}`, { signal });
  return response.task;
}

export async function listTaskAttempts(taskId: string, signal?: AbortSignal): Promise<TaskAttemptListResponse> {
  return fetchJSON<TaskAttemptListResponse>(`/api/tasks/${encodeURIComponent(taskId)}/attempts`, { signal });
}

async function admitChildAttempt(taskId: string, attemptId: string, action: "retry" | "rerun", request: TaskAttemptActionRequest = {}): Promise<TaskAttempt> {
  const response = await fetchJSON<{ attempt: TaskAttempt }>(`/api/tasks/${encodeURIComponent(taskId)}/attempts/${encodeURIComponent(attemptId)}/${action}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      idempotency_key: request.idempotencyKey ?? newIdempotencyKey(),
      pipeline: request.pipeline,
      intent: request.intent,
      reason: request.reason,
    }),
  });
  if (!response.attempt?.attempt_id) throw new Error(`${action} API returned no Attempt identity`);
  return response.attempt;
}

export function retryTaskAttempt(taskId: string, attemptId: string, request: TaskAttemptActionRequest = {}): Promise<TaskAttempt> {
  return admitChildAttempt(taskId, attemptId, "retry", request);
}

export function rerunTaskAttempt(taskId: string, attemptId: string, request: TaskAttemptActionRequest = {}): Promise<TaskAttempt> {
  return admitChildAttempt(taskId, attemptId, "rerun", request);
}

export async function getTaskAttempt(taskId: string, attemptId: string, signal?: AbortSignal): Promise<TaskAttempt> {
  const response = await fetchJSON<{ attempt: TaskAttempt }>(`/api/tasks/${encodeURIComponent(taskId)}/attempts/${encodeURIComponent(attemptId)}`, { signal });
  return response.attempt;
}

export async function setTaskArchive(taskId: string, expectedRevision: number, archived: boolean): Promise<ProductTask> {
  const response = await fetchJSON<{ task: ProductTask }>(`/api/tasks/${encodeURIComponent(taskId)}/${archived ? "archive" : "unarchive"}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ expected_revision: expectedRevision }),
  });
  return response.task;
}

export function newIdempotencyKey(): string {
  const cryptoAPI = globalThis.crypto as Crypto & { randomUUID?: () => string } | undefined;
  if (cryptoAPI?.randomUUID) return cryptoAPI.randomUUID();
  return `attempt-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
