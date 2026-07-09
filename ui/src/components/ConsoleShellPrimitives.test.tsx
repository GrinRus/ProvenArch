import { useState } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ActiveRunStrip } from "./ActiveRunStrip";
import { ActivityDrawer } from "./ActivityDrawer";
import { RightInspector } from "./RightInspector";
import { StageRail } from "./StageRail";
import type { RunLogEntry, RunReviewSummaryResponse, RunStatusResponse } from "../lib/appContracts";
import type { StageId, StageOption } from "../lib/consoleTypes";

const stages: StageOption[] = [
  { id: "source", label: "Source", description: "Repos & imports", status: "done" },
  { id: "readiness", label: "Readiness", description: "Validate setup", status: "active" },
  { id: "charter", label: "Charter", description: "Scope & rules", status: "pending" },
  { id: "analysis", label: "Analysis", description: "Run pipeline", status: "blocked", count: 2 },
  { id: "review", label: "Review", description: "Evidence & findings", status: "pending" },
  { id: "proposals", label: "Proposals", description: "ADR/RFC drafts", status: "pending" },
  { id: "ask", label: "Ask", description: "Read-only workspace Q&A", status: "pending" },
  { id: "publish", label: "Publish", description: "Git workflow", status: "pending" },
];

function StageRailHarness() {
  const [activeStage, setActiveStage] = useState<StageId>("source");
  return <StageRail stages={stages} activeStage={activeStage} onStageChange={setActiveStage} />;
}

describe("console shell primitives", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("keeps the V2 stage rail keyboard navigation and collapse state accessible", async () => {
    render(<StageRailHarness />);

    expect(screen.getByTestId("stage-rail")).toHaveAccessibleName("Proven Arch workflow");
    expect(screen.getByTestId("stage-analysis")).toHaveAccessibleName("Analysis: Run pipeline; blocked");
    expect(screen.getByTestId("stage-source")).toHaveAttribute("aria-current", "step");

    screen.getByTestId("stage-source").focus();
    fireEvent.keyDown(screen.getByTestId("stage-source"), { key: "End" });
    await waitFor(() => expect(screen.getByTestId("stage-publish")).toHaveAttribute("aria-current", "step"));

    fireEvent.keyDown(screen.getByTestId("stage-publish"), { key: "ArrowLeft" });
    await waitFor(() => expect(screen.getByTestId("stage-ask")).toHaveAttribute("aria-current", "step"));

    fireEvent.keyDown(screen.getByTestId("stage-ask"), { key: "Home" });
    await waitFor(() => expect(screen.getByTestId("stage-source")).toHaveAttribute("aria-current", "step"));

    const collapseButton = screen.getByTestId("stage-rail-collapse-btn");
    expect(collapseButton).toHaveAccessibleName("Collapse workflow rail");
    expect(collapseButton).toHaveAttribute("aria-pressed", "false");

    fireEvent.click(collapseButton);

    expect(screen.getByTestId("stage-rail")).toHaveClass("is-collapsed");
    expect(collapseButton).toHaveAccessibleName("Expand workflow rail");
    expect(collapseButton).toHaveAttribute("aria-pressed", "true");
  });

  it("starts the stage rail collapsed in narrow inspection viewports", () => {
    const originalMatchMedia = window.matchMedia;
    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      writable: true,
      value: vi.fn().mockReturnValue({
        matches: true,
        media: "(max-width: 900px)",
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }),
    });

    render(<StageRailHarness />);

    expect(screen.getByTestId("stage-rail")).toHaveClass("is-collapsed");

    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      writable: true,
      value: originalMatchMedia,
    });
  });

  it("keeps the active stage visible in narrow inspection viewports", async () => {
    const originalMatchMedia = window.matchMedia;
    const originalScrollIntoView = HTMLElement.prototype.scrollIntoView;
    const scrollIntoView = vi.fn();
    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      writable: true,
      value: vi.fn().mockReturnValue({
        matches: true,
        media: "(max-width: 900px)",
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }),
    });
    Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {
      configurable: true,
      value: scrollIntoView,
    });

    render(<StageRailHarness />);
    fireEvent.click(screen.getByTestId("stage-publish"));

    await waitFor(() => expect(screen.getByTestId("stage-publish")).toHaveAttribute("aria-current", "step"));
    expect(scrollIntoView).toHaveBeenCalledWith({ block: "nearest", inline: "center" });

    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      writable: true,
      value: originalMatchMedia,
    });
    if (originalScrollIntoView) {
      Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {
        configurable: true,
        value: originalScrollIntoView,
      });
    } else {
      delete (HTMLElement.prototype as { scrollIntoView?: unknown }).scrollIntoView;
    }
  });

  it("prioritizes inspector blockers while keeping empty sections discoverable but collapsed", () => {
    const onPrimaryAction = vi.fn();
    const onOpenArtifact = vi.fn();

    render(
      <RightInspector
        nextAction={{
          label: "Run analysis",
          description: "Validate readiness before starting the pipeline.",
          disabledReason: "Workspace validation is blocked.",
        }}
        blockers={[{ severity: "error", label: "Workspace invalid", detail: "workspace.yaml failed validation", path: "reports/coverage/open-questions.md" }]}
        evidenceRefs={[]}
        workspaceHealth={[]}
        runtimeSafety={[]}
        gitPublication={[]}
        onPrimaryAction={onPrimaryAction}
        onOpenArtifact={onOpenArtifact}
      />,
    );

    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("blocked");
    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("Workspace validation is blocked.");
    expect(screen.getByTestId("inspector-primary-action")).toBeDisabled();
    expect(screen.getByTestId("blockers-panel")).toHaveTextContent("Hard blockers");
    expect(screen.getByTestId("blockers-panel")).toHaveTextContent("Workspace invalid");
    expect(screen.getByTestId("review-warnings-panel")).not.toHaveAttribute("open");
    expect(screen.getByTestId("open-questions-panel")).not.toHaveAttribute("open");
    expect(screen.getByTestId("evidence-refs-panel")).not.toHaveAttribute("open");
    expect(screen.getByTestId("workspace-health-panel")).not.toHaveAttribute("open");
    expect(screen.getByTestId("runtime-safety-panel")).not.toHaveAttribute("open");
    expect(screen.getByTestId("git-publication-panel")).not.toHaveAttribute("open");

    fireEvent.click(screen.getByRole("button", { name: "Open evidence reference" }));

    expect(onOpenArtifact).toHaveBeenCalledWith("reports/coverage/open-questions.md");
    expect(onPrimaryAction).not.toHaveBeenCalled();
  });

  it("keeps inspector primary action available when only warning blockers require attention", () => {
    const onPrimaryAction = vi.fn();

    render(
      <RightInspector
        nextAction={{ label: "Review findings", description: "Inspect warnings before publishing." }}
        blockers={[{ severity: "warn", label: "Open questions", detail: "Coverage questions need operator review." }]}
        evidenceRefs={[{ severity: "info", label: "Coverage summary", detail: "reports/coverage/summary.md", path: "reports/coverage/summary.md" }]}
        workspaceHealth={[{ severity: "ok", label: "Workspace", detail: "manifest loaded" }]}
        runtimeSafety={[{ severity: "ok", label: "Runtime mode", detail: "fake baseline" }]}
        gitPublication={[{ severity: "info", label: "Proposal branch", detail: "proposal/beta-refresh" }]}
        onPrimaryAction={onPrimaryAction}
        onOpenArtifact={vi.fn()}
      />,
    );

    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("review");
    expect(screen.getByTestId("inspector-primary-action")).toBeEnabled();
    expect(screen.getByTestId("blockers-panel")).not.toHaveAttribute("open");
    expect(screen.getByTestId("open-questions-panel")).toHaveTextContent("Open questions");

    fireEvent.click(screen.getByTestId("inspector-primary-action"));

    expect(onPrimaryAction).toHaveBeenCalledTimes(1);
  });

  it("opens activity drawer by default for active run diagnostics", () => {
    const handlers = activityHandlers();

    render(<ActivityDrawer {...handlers} selectedRunId="run-empty" selectedRunStatus="running" logs={[]} renderedLogs="" runLogsStatus="" canExport={false} taskrunPaths={[]} />);

    expect(screen.getByTestId("activity-drawer")).toHaveAccessibleName("Selected run activity drawer");
    expect(screen.getByTestId("activity-drawer")).toHaveAttribute("open");
    expect(screen.getByTestId("activity-drawer-toggle")).toHaveTextContent("Activity / Events");
    expect(screen.getByText(/running run · 0 log entries/)).toBeInTheDocument();
    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("Logs will appear when the selected run emits events or raw output.");
    expect(screen.getByTestId("run-logs-copy-btn")).toBeDisabled();
    expect(screen.getByTestId("run-logs-download-btn")).toBeDisabled();
    expect(screen.getByTestId("run-logs-mode-select")).toHaveValue("all");
    expect(screen.getByTestId("run-logs-view-select")).toHaveValue("line");
  });

  it("renders activity drawer selected-run recovery states for missing or failed logs", () => {
    const handlers = activityHandlers();

    const { rerender } = render(<ActivityDrawer {...handlers} logs={[]} renderedLogs="" runLogsStatus="" canExport={false} taskrunPaths={[]} />);

    expect(screen.getByTestId("activity-drawer")).not.toHaveAttribute("open");
    expect(screen.getByText("No selected run")).toBeInTheDocument();
    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("Start or select a run to stream activity.");

    rerender(
      <ActivityDrawer
        {...handlers}
        selectedRunId="run-failed"
        selectedRunStatus="failed"
        selectedRunErrorCode="runtime_contract_failed"
        selectedRunError="runtime draft invalid"
        logs={[]}
        renderedLogs=""
        runLogsStatus=""
        canExport={false}
        taskrunPaths={[]}
      />,
    );

    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("Run failed before log entries were captured: runtime_contract_failed");
    expect(screen.getByTestId("activity-drawer")).toHaveAttribute("open");
    expect(screen.getByTestId("activity-drawer-toggle")).toHaveTextContent("failed run");

    rerender(
      <ActivityDrawer
        {...handlers}
        selectedRunId="run-canceled"
        selectedRunStatus="failed"
        selectedRunErrorCode="run_canceled"
        selectedRunError="canceled by request"
        logs={[]}
        renderedLogs=""
        runLogsStatus=""
        canExport={false}
        taskrunPaths={[]}
      />,
    );

    expect(screen.getByTestId("activity-drawer-toggle")).toHaveTextContent("canceled run");
    expect(screen.getByTestId("activity-empty-state")).toHaveClass("warning");
    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("Run was canceled before log entries were captured: run_canceled");
    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("Taskrun evidence remains in History.");
    expect(screen.getByTestId("activity-empty-state")).not.toHaveTextContent("Run failed before log entries");

    rerender(
      <ActivityDrawer
        {...handlers}
        selectedRunId="run-recovered"
        selectedRunStatus="failed"
        selectedRunErrorCode="run_reconciled_after_restart"
        selectedRunError="orphaned run reconciled after restart"
        logs={[]}
        renderedLogs=""
        runLogsStatus=""
        canExport={false}
        taskrunPaths={[]}
      />,
    );

    expect(screen.getByTestId("activity-drawer-toggle")).toHaveTextContent("recovered run");
    expect(screen.getByTestId("activity-empty-state")).toHaveClass("warning");
    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("Run was reconciled after restart before log entries were captured: run_reconciled_after_restart");
    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("History retains the run; start a new run if analysis still matters.");
  });

  it("renders activity drawer logs, taskrun artifact links and control callbacks", () => {
    const handlers = {
      ...activityHandlers(),
      runLogsMode: "events" as const,
      runLogsViewMode: "line+fields" as const,
    };
    const logs: RunLogEntry[] = [
      {
        cursor: 1,
        timestamp: "2026-04-03T12:00:00Z",
        level: "info",
        kind: "event",
        step_id: "init.step1.collect",
        message: "collect completed",
      },
      {
        cursor: 2,
        timestamp: "2026-04-03T12:00:01Z",
        level: "warning",
        kind: "runtime_output",
        stream: "stderr",
        step_id: "init.step2.as_is",
        message: "provider emitted warning",
      },
    ];

    render(
      <ActivityDrawer
        {...handlers}
        selectedRunId="run-logs"
        selectedRunStatus="succeeded"
        logs={logs}
        renderedLogs={"line one\nline two"}
        runLogsStatus="Logs copied."
        canExport={true}
        taskrunPaths={["reports/taskruns/run-1/runtime/runtime-execution.json"]}
      />,
    );

    expect(screen.getByText(/succeeded run · 2 log entries · last: provider emitted warning/)).toBeInTheDocument();
    expect(screen.getByTestId("activity-drawer")).not.toHaveAttribute("open");
    expect(screen.getByTestId("activity-drawer-toggle")).toHaveTextContent("Logs copied.");
    expect(screen.getAllByText("Logs copied.").length).toBeGreaterThan(0);
    expect(screen.getByTestId("activity-events-table")).toHaveTextContent("runtime output");
    expect(screen.getByTestId("activity-events-table")).toHaveTextContent("provider emitted warning");
    expect(screen.getByText("Runtime execution artifacts (1)")).toBeInTheDocument();
    expect(screen.getByTestId("run-logs-copy-btn")).toBeEnabled();
    expect(screen.getByTestId("run-logs-download-btn")).toBeEnabled();

    fireEvent.change(screen.getByTestId("run-logs-mode-select"), { target: { value: "raw" } });
    fireEvent.change(screen.getByTestId("run-logs-view-select"), { target: { value: "line" } });
    fireEvent.click(screen.getByTestId("run-logs-copy-btn"));
    fireEvent.click(screen.getByTestId("run-logs-download-btn"));
    fireEvent.click(screen.getByRole("button", { name: /Open runtime execution artifact/i }));

    expect(handlers.onRunLogsModeChange).toHaveBeenCalledWith("raw");
    expect(handlers.onRunLogsViewModeChange).toHaveBeenCalledWith("line");
    expect(handlers.onCopyRunLogs).toHaveBeenCalledTimes(1);
    expect(handlers.onDownloadRunLogs).toHaveBeenCalledTimes(1);
    expect(handlers.onOpenArtifact).toHaveBeenCalledWith("reports/taskruns/run-1/runtime/runtime-execution.json");
  });

  it("summarizes provider stream chunks without hiding the raw log view", () => {
    const handlers = activityHandlers();
    const rawPayload = JSON.stringify({
      type: "stream_event",
      event: {
        type: "content_block_delta",
        delta: { type: "thinking_delta", thinking: "checking repository structure before collect artifact writes" },
      },
    });
    const logs: RunLogEntry[] = [
      {
        cursor: 1,
        timestamp: "2026-04-03T12:00:00Z",
        level: "info",
        kind: "event",
        step_id: "init.step1.collect",
        message: "runtime task started",
      },
      {
        cursor: 2,
        timestamp: "2026-04-03T12:00:01Z",
        level: "info",
        kind: "runtime_output",
        stream: "stdout",
        step_id: "init.step1.collect",
        message: rawPayload,
      },
    ];

    render(
      <ActivityDrawer
        {...handlers}
        selectedRunId="run-stream"
        selectedRunStatus="running"
        logs={logs}
        renderedLogs={`[RAW] ${rawPayload}`}
        runLogsStatus=""
        canExport={true}
        taskrunPaths={[]}
      />,
    );

    expect(screen.getByTestId("activity-stream-summary")).toHaveTextContent("1 JSON stream chunk");
    expect(screen.getByTestId("activity-stream-summary")).toHaveTextContent("thinking_delta");
    expect(screen.getByTestId("activity-stream-summary")).toHaveTextContent("Full payload remains available below and in exported logs.");
    expect(screen.getByTestId("activity-events-table")).toHaveTextContent("Provider stream chunk: thinking_delta.");
    expect(screen.getByTestId("activity-events-table")).not.toHaveTextContent("checking repository structure before collect artifact writes");

    fireEvent.click(screen.getByText("Full selected log view"));

    expect(screen.getByTestId("run-logs-content")).toHaveTextContent("checking repository structure before collect artifact writes");
  });

  it("summarizes succeeded runs around reviewable artifacts instead of stale current step", () => {
    const reviewSummary: RunReviewSummaryResponse = {
      run_id: "run-success",
      pipeline: "init",
      status: "succeeded",
      started_at: "2026-04-03T12:00:00Z",
      finished_at: "2026-04-03T12:00:04Z",
      current_step: "init.step4.proposals",
      warnings: [],
      error_code: null,
      error: null,
      steps: [
        {
          step_id: "init.step0.constitution",
          key: "step0_constitution",
          label: "Charter",
          state: "done",
          provider: "fake",
          artifact_count: 2,
          artifact_paths: [],
          taskrun_paths: [],
          warnings_count: 0,
          errors_count: 0,
          last_message: "charter ready",
        },
        {
          step_id: "init.step1.collect",
          key: "step1_collect",
          label: "Collect",
          state: "done",
          provider: "fake",
          artifact_count: 3,
          artifact_paths: [],
          taskrun_paths: [],
          warnings_count: 0,
          errors_count: 0,
          last_message: "collect ready",
        },
      ],
    };

    render(
      <ActiveRunStrip
        runStatus={null}
        reviewSummary={reviewSummary}
        runtimeLabel="fake"
        cancelBusy={false}
        selectedRunIsActive={false}
        onCancel={vi.fn()}
        onOpenAnalysis={vi.fn()}
      />,
    );

    expect(screen.getByTestId("active-run-strip")).toHaveTextContent("evidence ready for review");
    expect(screen.getByTestId("active-run-strip")).toHaveTextContent("Review state");
    expect(screen.getByTestId("active-run-strip")).toHaveTextContent("5 artifacts ready");
    expect(screen.getByTestId("active-run-strip")).not.toHaveTextContent("Current step");
    expect(screen.queryByTestId("active-run-cancel-guidance")).not.toBeInTheDocument();
  });

  it("explains cooperative cancellation for active selected runs", () => {
    const runStatus: RunStatusResponse = {
      run_id: "run-active-cancel",
      pipeline: "refresh",
      status: "running",
      started_at: "2026-04-03T12:00:00Z",
      finished_at: null,
      current_step: "refresh.step1.collect",
      warnings: [],
      error_code: null,
      error: null,
    };

    render(
      <ActiveRunStrip
        runStatus={runStatus}
        reviewSummary={null}
        runtimeLabel="qwen-code"
        cancelBusy={false}
        selectedRunIsActive
        onCancel={vi.fn()}
        onOpenAnalysis={vi.fn()}
      />,
    );

    expect(screen.getByTestId("active-run-strip")).toHaveTextContent("refresh.step1.collect");
    expect(screen.getByTestId("active-run-cancel-guidance")).toHaveTextContent("Cancel requests a cooperative stop; taskrun evidence stays in History.");
  });
});

function activityHandlers() {
  return {
    runLogsMode: "all" as const,
    runLogsViewMode: "line" as const,
    onRunLogsModeChange: vi.fn(),
    onRunLogsViewModeChange: vi.fn(),
    onCopyRunLogs: vi.fn(),
    onDownloadRunLogs: vi.fn(),
    onOpenArtifact: vi.fn(),
  };
}
