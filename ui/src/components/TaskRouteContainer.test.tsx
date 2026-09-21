import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TaskRouteContainer } from "./TaskRouteContainer";
import { admitTaskAttempt, getTask, getTaskAttempt, listTaskAttempts, listTasks, type ProductTask, type TaskAttempt } from "../lib/taskApi";

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
  admitTaskAttempt: vi.fn(async () => ({ attempt_id: "attempt-admitted", task_id: "task-1", run_id: "run-admitted", status: "queued", pipeline: "init", admitted_at: "2026-08-11T10:02:00Z", task_revision: 1 })),
  listTasks: vi.fn(async () => ({ items: [task], next_cursor: "", has_more: false })),
  getTask: vi.fn(async () => task),
  listTaskAttempts: vi.fn(async () => ({ items: [] })),
  getTaskAttempt: vi.fn(async () => ({ attempt_id: "attempt-2", task_id: "task-1", run_id: "run-1", status: "failed", pipeline: "init", admitted_at: "2026-08-11T10:00:00Z", task_revision: 1 })),
  newIdempotencyKey: vi.fn(() => "attempt-idempotency-key"),
  setTaskArchive: vi.fn(async () => task),
}));
vi.mock("../lib/runApi", () => ({
  getPipelineRunReviewSummary: vi.fn(async () => null),
}));

import { getPipelineRunReviewSummary } from "../lib/runApi";

const taskAttempt = {
  version: 1,
  attempt_id: "attempt-admitted",
  task_id: "task-1",
  run_id: "run-admitted",
  status: "queued",
  pipeline: "init",
  admitted_at: "2026-08-11T10:02:00Z",
  queued_at: "2026-08-11T10:02:00Z",
  started_at: null,
  finished_at: null,
  task_revision: 1,
  intent_snapshot: task,
  effective_runtime: task.desired_runner,
  terminal_summary: null,
  outcome: task.outcome,
  retained_evidence: "none yet",
  publication: task.publication,
} as unknown as TaskAttempt;

function deferredResponse<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

describe("TaskRouteContainer", () => {
  beforeEach(() => vi.clearAllMocks());

  it("loads an authoritative Task Inbox with derived groups", async () => {
    render(<TaskRouteContainer view="inbox" filters={{}} />);
    expect(screen.getByRole("heading", { name: "Task Inbox" })).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("task-row-task-1")).toHaveTextContent("Map payment authorization"));
    expect(screen.getByTestId("task-group-ready")).toHaveTextContent("Payments");
    expect(screen.getByText("Advanced filters").closest("details")).not.toHaveAttribute("open");
    expect(screen.getByText("Empty lifecycle groups").closest("details")).not.toHaveAttribute("open");
    expect(screen.getByTestId("task-route-inbox")).not.toHaveTextContent("latest run");
  });

  it("keeps empty and error states explicit with one recovery action", async () => {
    vi.mocked(listTasks).mockResolvedValueOnce({ items: [], next_cursor: "", has_more: false });
    const { unmount } = render(<TaskRouteContainer view="inbox" filters={{}} />);
    expect(await screen.findByTestId("task-inbox-empty")).toHaveTextContent("No Tasks match these filters");
    expect(screen.getByTestId("task-inbox-empty")).toHaveAttribute("role", "status");
    unmount();

    vi.mocked(listTasks).mockRejectedValueOnce(new Error("Task service unavailable"));
    render(<TaskRouteContainer view="inbox" filters={{}} />);
    expect(await screen.findByTestId("task-inbox-error")).toHaveTextContent("Task service unavailable");
    expect(screen.getByTestId("task-inbox-error")).toHaveAttribute("role", "alert");
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("retries the Task Inbox request without relying on a URL change", async () => {
    vi.mocked(listTasks)
      .mockRejectedValueOnce(new Error("Task service unavailable"))
      .mockResolvedValueOnce({ items: [task as ProductTask], next_cursor: "", has_more: false });
    render(<TaskRouteContainer view="inbox" filters={{}} />);

    await screen.findByTestId("task-inbox-error");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await waitFor(() => expect(screen.getByTestId("task-row-task-1")).toHaveTextContent("Payments"));
    expect(listTasks).toHaveBeenCalledTimes(2);
  });

  it("keeps Load more failures visible and retryable", async () => {
    const nextTask = { ...task, task_id: "task-next", title: "Next Payments" } as ProductTask;
    vi.mocked(listTasks)
      .mockResolvedValueOnce({ items: [task as ProductTask], next_cursor: "cursor-1", has_more: true })
      .mockRejectedValueOnce(new Error("next Task page unavailable"))
      .mockResolvedValueOnce({ items: [nextTask], next_cursor: "", has_more: false });
    render(<TaskRouteContainer view="inbox" filters={{}} />);

    fireEvent.click(await screen.findByRole("button", { name: "Load more Tasks" }));
    await screen.findByTestId("task-inbox-load-more-error");
    fireEvent.click(screen.getByTestId("task-inbox-load-more-error").parentElement!.querySelector("button")!);

    await waitFor(() => expect(screen.getByTestId("task-row-task-next")).toHaveTextContent("Next Payments"));
    expect(screen.queryByTestId("task-inbox-load-more-error")).not.toBeInTheDocument();
    expect(listTasks).toHaveBeenCalledTimes(3);
  });

  it("opens a Task row from the keyboard without changing its identity", async () => {
    const onSelectTask = vi.fn();
    render(<TaskRouteContainer view="inbox" filters={{}} onSelectTask={onSelectTask} />);
    const row = await screen.findByLabelText("Open Task Payments");
    fireEvent.keyDown(row, { key: "Enter" });
    expect(onSelectTask).toHaveBeenCalledWith("task-1", {});
  });

  it("recovers a Task with no admitted Attempt from its detail page", async () => {
    const onSelectAttempt = vi.fn();
    render(<TaskRouteContainer view="detail" taskId="task-1" filters={{}} onSelectAttempt={onSelectAttempt} />);

    await screen.findByTestId("task-start-attempt");
    fireEvent.click(screen.getByTestId("task-start-attempt"));

    await waitFor(() => expect(admitTaskAttempt).toHaveBeenCalledWith("task-1", { pipeline: "init", intent: "start", idempotencyKey: "attempt-idempotency-key" }));
    expect(onSelectAttempt).toHaveBeenCalledWith("task-1", "attempt-admitted", {});
  });

  it("keeps first Attempt admission retryable after a transient failure", async () => {
    vi.mocked(admitTaskAttempt)
      .mockRejectedValueOnce(new Error("runner is warming up"))
      .mockResolvedValueOnce(taskAttempt);
    render(<TaskRouteContainer view="detail" taskId="task-1" filters={{}} />);

    const start = await screen.findByTestId("task-start-attempt");
    fireEvent.click(start);
    expect(await screen.findByTestId("task-attempt-admission-error")).toHaveTextContent("runner is warming up");
    fireEvent.click(screen.getByTestId("task-start-attempt"));

    await waitFor(() => expect(admitTaskAttempt).toHaveBeenCalledTimes(2));
    expect(vi.mocked(admitTaskAttempt).mock.calls[0]).toEqual(["task-1", { pipeline: "init", intent: "start", idempotencyKey: "attempt-idempotency-key" }]);
    expect(vi.mocked(admitTaskAttempt).mock.calls[1]).toEqual(["task-1", { pipeline: "init", intent: "start", idempotencyKey: "attempt-idempotency-key" }]);
  });

  it("drops a late Task page after the Inbox filter changes", async () => {
    const latePage = deferredResponse<{ items: ProductTask[]; next_cursor: string; has_more: boolean }>();
    const filteredTask = { ...task, task_id: "task-filtered", title: "Filtered Payments" } as ProductTask;
    const lateTask = { ...task, task_id: "task-late-page", title: "Late page" } as ProductTask;
    vi.mocked(listTasks)
      .mockImplementationOnce(async () => ({ items: [task as ProductTask], next_cursor: "cursor-old", has_more: true }))
      .mockImplementationOnce(async () => latePage.promise)
      .mockImplementationOnce(async () => ({ items: [filteredTask], next_cursor: "", has_more: false }));

    const { rerender } = render(<TaskRouteContainer view="inbox" filters={{}} />);
    fireEvent.click(await screen.findByRole("button", { name: "Load more Tasks" }));
    rerender(<TaskRouteContainer view="inbox" filters={{ runner: "qwen-code" }} />);
    await screen.findByTestId("task-row-task-filtered");

    latePage.resolve({ items: [lateTask], next_cursor: "", has_more: false });
    await waitFor(() => expect(screen.queryByTestId("task-row-task-late-page")).not.toBeInTheDocument());
    expect(screen.getByTestId("task-route-inbox")).toHaveTextContent("Filtered Payments");
  });

  it("ignores a late Task page error after the Inbox filter changes", async () => {
    const latePage = deferredResponse<{ items: ProductTask[]; next_cursor: string; has_more: boolean }>();
    const filteredTask = { ...task, task_id: "task-filtered-error", title: "Filtered after error" } as ProductTask;
    vi.mocked(listTasks)
      .mockImplementationOnce(async () => ({ items: [task as ProductTask], next_cursor: "cursor-old", has_more: true }))
      .mockImplementationOnce(async () => latePage.promise)
      .mockImplementationOnce(async () => ({ items: [filteredTask], next_cursor: "", has_more: false }));

    const { rerender } = render(<TaskRouteContainer view="inbox" filters={{}} />);
    fireEvent.click(await screen.findByRole("button", { name: "Load more Tasks" }));
    rerender(<TaskRouteContainer view="inbox" filters={{ runner: "qwen-code" }} />);
    await screen.findByTestId("task-row-task-filtered-error");

    latePage.reject(new Error("late page failed"));
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(screen.queryByTestId("task-inbox-error")).not.toBeInTheDocument();
    expect(screen.getByTestId("task-route-inbox")).toHaveTextContent("Filtered after error");
  });

  it("keeps a newer Task route when an older review response resolves late", async () => {
    const lateOldReview = deferredResponse<Awaited<ReturnType<typeof getPipelineRunReviewSummary>>>();
    const taskFor = (taskId: string, title: string, runId: string) => ({
      ...task,
      task_id: taskId,
      title,
      attempts: [{ attempt_id: `${taskId}-attempt`, run_id: runId, status: "succeeded", updated_at: "2026-08-11T10:01:00Z" }],
      outcome: { state: "available", attempt_id: `${taskId}-attempt`, run_id: runId },
    } as ProductTask);
    const attemptFor = (taskId: string, runId: string) => ({
      version: 1,
      attempt_id: `${taskId}-attempt`,
      task_id: taskId,
      run_id: runId,
      status: "succeeded",
      pipeline: "init",
      admitted_at: "2026-08-11T10:00:00Z",
      task_revision: 1,
    } as TaskAttempt);
    const reviewFor = (runId: string, summary: string) => ({
      run_id: runId,
      result: { state: "completed", summary, produced: {}, partial_scopes: 0, failed_scopes: 0, promotion: { changed: true, current_usable: true }, recommended_action: "review_architecture" },
    } as never);
    vi.mocked(getTask).mockImplementation(async (taskId) => taskFor(taskId, taskId === "task-old" ? "Old Task" : "New Task", taskId === "task-old" ? "run-old" : "run-new"));
    vi.mocked(listTaskAttempts).mockImplementation(async (taskId) => ({ items: [attemptFor(taskId, taskId === "task-old" ? "run-old" : "run-new")] }));
    vi.mocked(getPipelineRunReviewSummary).mockImplementation(async (runId) => runId === "run-old" ? lateOldReview.promise : reviewFor("run-new", "New route outcome"));

    const { rerender } = render(<TaskRouteContainer view="detail" taskId="task-old" filters={{}} />);
    await waitFor(() => expect(getPipelineRunReviewSummary).toHaveBeenCalledWith("run-old", true, expect.anything()));
    rerender(<TaskRouteContainer view="detail" taskId="task-new" filters={{}} />);
    await waitFor(() => expect(screen.getByTestId("task-outcome")).toHaveTextContent("New route outcome"));

    lateOldReview.resolve(reviewFor("run-old", "Old route outcome"));
    await waitFor(() => {
      expect(screen.getByTestId("task-route-detail")).toHaveTextContent("New Task");
      expect(screen.getByTestId("task-outcome")).toHaveTextContent("New route outcome");
      expect(screen.getByTestId("task-outcome")).not.toHaveTextContent("Old route outcome");
    });
  });

  it("keeps terminal Task identity visible when the outcome review needs retry", async () => {
    const terminalAttempt = { version: 1, attempt_id: "attempt-1", task_id: "task-1", run_id: "run-1", status: "succeeded", pipeline: "init", admitted_at: "2026-08-11T10:00:00Z", task_revision: 1 } as TaskAttempt;
    const successfulReview = {
      run_id: "run-1",
      result: { state: "completed", summary: "Review recovered", produced: {}, partial_scopes: 0, failed_scopes: 0, promotion: { changed: true, current_usable: true }, recommended_action: "review_architecture" },
    } as never;
    vi.mocked(getTask).mockResolvedValue({ ...task, outcome: { state: "available", attempt_id: "attempt-1", run_id: "run-1" } } as ProductTask);
    vi.mocked(listTaskAttempts).mockResolvedValue({ items: [terminalAttempt] });
    vi.mocked(getPipelineRunReviewSummary)
      .mockRejectedValueOnce(new Error("review service unavailable"))
      .mockResolvedValueOnce(successfulReview);
    render(<TaskRouteContainer view="detail" taskId="task-1" filters={{}} />);

    expect(await screen.findByTestId("task-review-error")).toHaveTextContent("review service unavailable");
    expect(screen.getByTestId("task-route-detail")).toHaveTextContent("Map payment authorization");
    fireEvent.click(screen.getByTestId("task-review-retry"));

    await waitFor(() => expect(screen.getByTestId("task-outcome")).toHaveTextContent("Review recovered"));
    expect(getPipelineRunReviewSummary).toHaveBeenCalledTimes(2);
  });

  it("keeps exact Task and Attempt identities visible while loading", () => {
    render(<TaskRouteContainer view="attempt" taskId="task-1" attemptId="attempt-2" />);
    expect(screen.getByTestId("task-route-identities")).toHaveTextContent("task-1");
    expect(screen.getByTestId("task-route-identities")).toHaveTextContent("attempt-2");
  });

  it("retries a failed Task detail load without changing the requested identity", async () => {
    vi.mocked(getTask).mockRejectedValueOnce(new Error("Task service unavailable")).mockResolvedValueOnce(task as ProductTask);
    render(<TaskRouteContainer view="detail" taskId="task-1" filters={{}} />);

    await screen.findByTestId("task-detail-error");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await waitFor(() => expect(screen.getByTestId("task-route-detail")).toHaveTextContent("Map payment authorization"));
    expect(getTask).toHaveBeenCalledTimes(2);
  });

  it("retries exact Attempt and Pipeline Studio loads in place", async () => {
    const attemptResponse = { attempt_id: "attempt-2", task_id: "task-1", run_id: "run-1", status: "failed", pipeline: "init", admitted_at: "2026-08-11T10:00:00Z", task_revision: 1 } as TaskAttempt;
    vi.mocked(getTaskAttempt).mockRejectedValueOnce(new Error("Attempt service unavailable")).mockResolvedValueOnce(attemptResponse);
    const { rerender } = render(<TaskRouteContainer view="attempt" taskId="task-1" attemptId="attempt-2" />);
    await screen.findByTestId("task-attempt-error");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() => expect(screen.getByTestId("task-route-attempt")).toHaveTextContent("run-1"));

    vi.mocked(getTaskAttempt).mockRejectedValueOnce(new Error("Studio service unavailable")).mockResolvedValueOnce(attemptResponse);
    rerender(<TaskRouteContainer view="studio" taskId="task-1" attemptId="attempt-2" />);
    await screen.findByTestId("pipeline-studio-error");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() => expect(screen.getByTestId("task-pipeline-studio")).toHaveTextContent("attempt-2"));
  });

  it("fails closed for an invalid deep link", () => {
    render(<TaskRouteContainer view="inbox" invalid={["task"]} />);
    expect(screen.getByTestId("task-route-invalid")).toHaveTextContent("No Task or Attempt was selected");
  });

  it("shows semantic outcome counts only when the exact run review is available", async () => {
    const onOutcomeSettled = vi.fn();
    vi.mocked(getTask).mockResolvedValue({ ...task, outcome: { state: "available", attempt_id: "attempt-1", run_id: "run-1" } } as ProductTask);
    vi.mocked(listTaskAttempts).mockResolvedValue({ items: [{ version: 1, attempt_id: "attempt-1", task_id: "task-1", run_id: "run-1", status: "succeeded", pipeline: "init", admitted_at: "2026-08-11T10:00:00Z", task_revision: 1, finished_at: "2026-08-11T10:01:00Z" } as TaskAttempt] });
    vi.mocked(getPipelineRunReviewSummary).mockResolvedValue({ run_id: "run-1", pipeline: "init", status: "succeeded", started_at: "2026-08-11T10:00:00Z", result: { state: "completed", summary: "Validated architecture result", produced: {}, partial_scopes: 0, failed_scopes: 0, promotion: { changed: true, current_usable: true }, recommended_action: "review_architecture" }, steps: [], review: { review_kind: "initial", source_run_id: "run-1", semantic_changes: { available: true, categories: { entities: { added: [], changed: [], removed: [] }, edges: { added: [], changed: [], removed: [] }, findings: { added: [], changed: [], removed: [] }, gaps: { added: [], changed: [], removed: [] } } }, document_changes: { available: true, added: [], changed: [], removed: [] }, findings: [], questions: [], gaps: [], summary: { entities_added: 2, entities_changed: 1, entities_removed: 0, edges_added: 3, edges_changed: 0, edges_removed: 1, documents_added: 0, documents_changed: 0, documents_removed: 0, findings: 1, questions: 2, gaps: 1 }, runtime: { providers: [], step_providers: {} }, authority: { mode: "promoted_current", source_run_id: "run-1" }, generated_at: "2026-08-11T10:01:00Z" } });
    render(<TaskRouteContainer view="detail" taskId="task-1" filters={{}} onOutcomeSettled={onOutcomeSettled} />);
    await waitFor(() => expect(screen.getByTestId("task-outcome")).toHaveTextContent("Validated architecture result"));
    expect(screen.getByTestId("task-outcome")).toHaveTextContent("2 added · 1 changed · 0 removed");
    expect(screen.getByTestId("task-outcome")).toHaveTextContent("Current validator-approved Architecture remains available");
    expect(onOutcomeSettled).toHaveBeenCalledWith("task-1", "attempt-1", "run-1");
  });

  it("does not fabricate zero semantic delta when comparison is unavailable", async () => {
    vi.mocked(getTask).mockResolvedValue({ ...task, outcome: { state: "available", attempt_id: "attempt-1", run_id: "run-1" } } as ProductTask);
    vi.mocked(listTaskAttempts).mockResolvedValue({ items: [{ version: 1, attempt_id: "attempt-1", task_id: "task-1", run_id: "run-1", status: "succeeded", pipeline: "init", admitted_at: "2026-08-11T10:00:00Z", task_revision: 1 } as TaskAttempt] });
    vi.mocked(getPipelineRunReviewSummary).mockResolvedValue({
      run_id: "run-1", pipeline: "init", status: "succeeded", started_at: "2026-08-11T10:00:00Z",
      result: { state: "completed", summary: "Initial snapshot ready", produced: {}, partial_scopes: 0, failed_scopes: 0, promotion: { changed: true, current_usable: true }, recommended_action: "review_architecture" },
      steps: [],
      review: { review_kind: "initial", source_run_id: "run-1", semantic_changes: { available: false, reason: "No baseline", categories: { entities: { added: [], changed: [], removed: [] }, edges: { added: [], changed: [], removed: [] }, findings: { added: [], changed: [], removed: [] }, gaps: { added: [], changed: [], removed: [] } } }, document_changes: { available: true, added: [], changed: [], removed: [] }, findings: [], questions: [], gaps: [], summary: { entities_added: 0, entities_changed: 0, entities_removed: 0, edges_added: 0, edges_changed: 0, edges_removed: 0, documents_added: 0, documents_changed: 0, documents_removed: 0, findings: 0, questions: 0, gaps: 0 }, runtime: { providers: [], step_providers: {} }, authority: { mode: "promoted_current", source_run_id: "run-1" }, generated_at: "2026-08-11T10:01:00Z" },
    });
    render(<TaskRouteContainer view="detail" taskId="task-1" filters={{}} />);
    const outcome = await screen.findByTestId("task-outcome");
    expect(outcome).toHaveTextContent("Semantic comparison is unavailable");
    expect(outcome).toHaveTextContent("EntitiesUnavailable");
    expect(outcome).not.toHaveTextContent("0 added · 0 changed · 0 removed");
  });

  it("keeps a succeeded Attempt terminal in Pipeline Studio", async () => {
    vi.mocked(getTaskAttempt).mockResolvedValue({ version: 1, attempt_id: "attempt-1", task_id: "task-1", run_id: "run-1", status: "succeeded", pipeline: "init", admitted_at: "2026-08-11T10:00:00Z", task_revision: 1, retained_evidence: "retained" } as TaskAttempt);
    vi.mocked(getPipelineRunReviewSummary).mockResolvedValue({
      run_id: "run-1", pipeline: "init", status: "succeeded", started_at: "2026-08-11T10:00:00Z",
      steps: [{ step_id: "init.step1", key: "collect", label: "Collect", state: "active", artifact_count: 0, artifact_paths: [], taskrun_paths: [], warnings_count: 0, errors_count: 0 }],
      result: { state: "completed", summary: "Ready", produced: {}, partial_scopes: 0, failed_scopes: 0, promotion: { changed: true, current_usable: true }, recommended_action: "review_architecture" },
    } as never);
    render(<TaskRouteContainer view="studio" taskId="task-1" attemptId="attempt-1" filters={{}} />);
    const studio = await screen.findByTestId("task-pipeline-studio");
    expect(studio).toHaveTextContent("done");
    expect(studio).not.toHaveTextContent("active");
    expect(screen.queryByTestId("pipeline-blocker")).not.toBeInTheDocument();
  });

  it("opens Pipeline Studio only for the exact Attempt and keeps diagnostics bounded", async () => {
    vi.mocked(getTaskAttempt).mockResolvedValue({ version: 1, attempt_id: "attempt-2", task_id: "task-1", run_id: "run-1", status: "failed", pipeline: "init", admitted_at: "2026-08-11T10:00:00Z", task_revision: 1, terminal_summary: { status: "failed", error: "stopped", retained_evidence: "retained" }, retained_evidence: "retained" } as TaskAttempt);
    vi.mocked(getPipelineRunReviewSummary).mockResolvedValue(null);
    render(<TaskRouteContainer view="studio" taskId="task-1" attemptId="attempt-2" filters={{}} />);
    await waitFor(() => expect(screen.getByTestId("task-pipeline-studio")).toHaveTextContent("run-1"));
    expect(screen.getByTestId("task-pipeline-studio")).toHaveTextContent("init.step0.constitution");
    expect(screen.getByTestId("pipeline-blocker")).toHaveTextContent("Attempt stopped");
    expect(screen.getByTestId("task-pipeline-studio")).not.toHaveTextContent("latest run");
  });
});
