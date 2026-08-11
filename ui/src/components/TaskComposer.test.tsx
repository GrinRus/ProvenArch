import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { GuidedRepo } from "../lib/appContracts";
import { TaskComposer } from "./TaskComposer";

const repos: GuidedRepo[] = [{ id: "repo-1", name: "payments", mode: "path", path: "/work/payments", git_url: "", ref: "", analysis_include: "src, docs", analysis_exclude: "" }];

afterEach(() => vi.unstubAllGlobals());

describe("TaskComposer", () => {
  it("shows inline readiness and creates a Task with immutable runner intent", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url === "/api/tasks") {
        expect(JSON.parse(String(init?.body))).toMatchObject({
          title: "Map payments",
          goal: "Explain authorization",
          scope: { repositories: [{ name: "payments", paths: ["src", "docs"] }] },
          desired_runner: { preset: "deterministic-demo", mode: "fake", provider: "claude-code" },
        });
        return new Response(JSON.stringify({ task: { task_id: "task-opaque" } }), { status: 201, headers: { "Content-Type": "application/json" } });
      }
      expect(url).toBe("/api/tasks/task-opaque/attempts");
      expect(JSON.parse(String(init?.body))).toMatchObject({ pipeline: "init", intent: "start" });
      return new Response(JSON.stringify({ attempt: { attempt_id: "attempt-opaque" } }), { status: 202, headers: { "Content-Type": "application/json" } });
    });
    vi.stubGlobal("fetch", fetchMock);
    const onCreated = vi.fn();
    render(<TaskComposer workspaceReady repos={repos} runtimeMode="fake" runtimeProvider="claude-code" onCreated={onCreated} />);
    expect(screen.getByTestId("task-runner-readiness")).toHaveTextContent("will snapshot fake / claude-code");
    fireEvent.change(screen.getByLabelText("Task title"), { target: { value: "Map payments" } });
    fireEvent.change(screen.getByLabelText("Goal"), { target: { value: "Explain authorization" } });
    fireEvent.click(screen.getByRole("button", { name: "Create Task" }));
    await waitFor(() => expect(onCreated).toHaveBeenCalledWith("task-opaque"));
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("blocks create when the selected runner differs from the admitted session", () => {
    render(<TaskComposer workspaceReady repos={repos} runtimeMode="fake" runtimeProvider="claude-code" onCreated={vi.fn()} />);
    fireEvent.change(screen.getByLabelText("Task title"), { target: { value: "Task" } });
    fireEvent.change(screen.getByLabelText("Goal"), { target: { value: "Goal" } });
    fireEvent.change(screen.getByLabelText("Runtime mode"), { target: { value: "headless" } });
    expect(screen.getByTestId("task-runner-readiness")).toHaveTextContent("Current session is fake");
    expect(screen.getByTestId("task-create-submit")).toBeDisabled();
  });

  it("keeps a created Task recoverable when Attempt admission fails", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      if (String(input) === "/api/tasks") return new Response(JSON.stringify({ task: { task_id: "task-created" } }), { status: 201 });
      return new Response(JSON.stringify({ error: { message: "another Attempt is active" } }), { status: 409, headers: { "Content-Type": "application/json" } });
    });
    vi.stubGlobal("fetch", fetchMock);
    const onCreated = vi.fn();
    render(<TaskComposer workspaceReady repos={repos} runtimeMode="fake" runtimeProvider="claude-code" onCreated={onCreated} />);
    fireEvent.change(screen.getByLabelText("Task title"), { target: { value: "Task" } });
    fireEvent.change(screen.getByLabelText("Goal"), { target: { value: "Goal" } });
    fireEvent.click(screen.getByRole("button", { name: "Create Task" }));
    expect(await screen.findByTestId("task-composer-error")).toHaveTextContent("Task created, but Attempt admission failed");
    fireEvent.click(screen.getByTestId("task-open-created"));
    expect(onCreated).toHaveBeenCalledWith("task-created");
  });

  it("fails closed when workspace scope is absent", () => {
    render(<TaskComposer workspaceReady runtimeMode="fake" runtimeProvider="claude-code" repos={[]} onCreated={vi.fn()} />);
    expect(screen.getByTestId("task-scope-empty")).toBeInTheDocument();
    expect(screen.getByTestId("task-create-submit")).toBeDisabled();
  });
});
