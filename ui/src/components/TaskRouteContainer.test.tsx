import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TaskRouteContainer } from "./TaskRouteContainer";
import { getTask, listTaskAttempts, type ProductTask, type TaskAttempt } from "../lib/taskApi";

const task = {
  version: 1,
  task_id: "task-1",
  title: "Payments",
  goal: "Map payment authorization",
  scope: { repositories: [{ name: "payments", paths: ["."] }] },
  desired_runner: { preset: "deterministic-demo", mode: "fake", provider: "claude-code" },
  lifecycle: "open",
  revision: 1,
  created_at: "2026-08-11T10:00:00Z",
  updated_at: "2026-08-11T10:00:00Z",
  last_activity_at: "2026-08-11T10:00:00Z",
  attempts: [],
  outcome: { state: "unavailable", unavailable_reason: "no attempt has completed" },
  publication: { state: "unavailable", unavailable_reason: "not published" },
};

vi.mock("../lib/taskApi", () => ({
  listTasks: vi.fn(async () => ({ items: [task], next_cursor: "", has_more: false })),
  getTask: vi.fn(async () => task),
  listTaskAttempts: vi.fn(async () => ({ items: [] })),
  getTaskAttempt: vi.fn(async () => ({ attempt_id: "attempt-2", task_id: "task-1", run_id: "run-1", status: "failed", pipeline: "init", admitted_at: "2026-08-11T10:00:00Z", task_revision: 1 })),
  setTaskArchive: vi.fn(async () => task),
}));
vi.mock("../lib/runApi", () => ({
  getPipelineRunReviewSummary: vi.fn(async () => null),
}));

import { getPipelineRunReviewSummary } from "../lib/runApi";

describe("TaskRouteContainer", () => {
  beforeEach(() => vi.clearAllMocks());

  it("loads an authoritative Task Inbox with derived groups", async () => {
    render(<TaskRouteContainer view="inbox" filters={{}} />);
    expect(screen.getByRole("heading", { name: "Task Inbox" })).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("task-row-task-1")).toHaveTextContent("Map payment authorization"));
    expect(screen.getByTestId("task-group-ready")).toHaveTextContent("Payments");
    expect(screen.getByTestId("task-route-inbox")).not.toHaveTextContent("latest run");
  });

  it("keeps exact Task and Attempt identities visible while loading", () => {
    render(<TaskRouteContainer view="attempt" taskId="task-1" attemptId="attempt-2" />);
    expect(screen.getByTestId("task-route-identities")).toHaveTextContent("task-1");
    expect(screen.getByTestId("task-route-identities")).toHaveTextContent("attempt-2");
  });

  it("fails closed for an invalid deep link", () => {
    render(<TaskRouteContainer view="inbox" invalid={["task"]} />);
    expect(screen.getByTestId("task-route-invalid")).toHaveTextContent("No Task or Attempt was selected");
  });

  it("shows semantic outcome counts only when the exact run review is available", async () => {
    vi.mocked(getTask).mockResolvedValue({ ...task, outcome: { state: "available", attempt_id: "attempt-1", run_id: "run-1" } } as ProductTask);
    vi.mocked(listTaskAttempts).mockResolvedValue({ items: [{ version: 1, attempt_id: "attempt-1", task_id: "task-1", run_id: "run-1", status: "succeeded", pipeline: "init", admitted_at: "2026-08-11T10:00:00Z", task_revision: 1, finished_at: "2026-08-11T10:01:00Z" } as TaskAttempt] });
    vi.mocked(getPipelineRunReviewSummary).mockResolvedValue({ run_id: "run-1", pipeline: "init", status: "succeeded", started_at: "2026-08-11T10:00:00Z", result: { state: "completed", summary: "Validated architecture result", produced: {}, partial_scopes: 0, failed_scopes: 0, promotion: { changed: true, current_usable: true }, recommended_action: "review_architecture" }, steps: [], review: { review_kind: "initial", source_run_id: "run-1", semantic_changes: { available: true, categories: { entities: { added: [], changed: [], removed: [] }, edges: { added: [], changed: [], removed: [] }, findings: { added: [], changed: [], removed: [] }, gaps: { added: [], changed: [], removed: [] } } }, document_changes: { available: true, added: [], changed: [], removed: [] }, findings: [], questions: [], gaps: [], summary: { entities_added: 2, entities_changed: 1, entities_removed: 0, edges_added: 3, edges_changed: 0, edges_removed: 1, documents_added: 0, documents_changed: 0, documents_removed: 0, findings: 1, questions: 2, gaps: 1 }, runtime: { providers: [], step_providers: {} }, authority: { mode: "promoted_current", source_run_id: "run-1" }, generated_at: "2026-08-11T10:01:00Z" } });
    render(<TaskRouteContainer view="detail" taskId="task-1" filters={{}} />);
    await waitFor(() => expect(screen.getByTestId("task-outcome")).toHaveTextContent("Validated architecture result"));
    expect(screen.getByTestId("task-outcome")).toHaveTextContent("2 added · 1 changed · 0 removed");
    expect(screen.getByTestId("task-outcome")).toHaveTextContent("Current validator-approved Architecture remains available");
  });
});
