import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TaskRouteContainer } from "./TaskRouteContainer";

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
});
