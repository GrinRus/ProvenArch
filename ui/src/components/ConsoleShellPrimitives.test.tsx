import { useState } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ActivityDrawer } from "./ActivityDrawer";
import { RightInspector } from "./RightInspector";
import { StageRail } from "./StageRail";
import type { RunLogEntry } from "../lib/appContracts";
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

  it("prioritizes inspector blockers while keeping empty sections and artifact links accessible", () => {
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
    expect(screen.getByTestId("review-warnings-panel")).toHaveTextContent("No review warnings.");
    expect(screen.getByTestId("open-questions-panel")).toHaveTextContent("No open questions loaded.");
    expect(screen.getByTestId("evidence-refs-panel")).toHaveTextContent("No evidence yet.");
    expect(screen.getByTestId("workspace-health-panel")).toHaveTextContent("Workspace status unavailable.");
    expect(screen.getByTestId("runtime-safety-panel")).toHaveTextContent("Runtime profile unavailable.");
    expect(screen.getByTestId("git-publication-panel")).toHaveTextContent("Git publication path unavailable.");

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
    expect(screen.getByTestId("blockers-panel")).toHaveTextContent("No hard blockers detected.");
    expect(screen.getByTestId("open-questions-panel")).toHaveTextContent("Open questions");

    fireEvent.click(screen.getByTestId("inspector-primary-action"));

    expect(onPrimaryAction).toHaveBeenCalledTimes(1);
  });

  it("renders activity drawer empty/export-disabled state without hiding log controls", () => {
    const handlers = activityHandlers();

    render(<ActivityDrawer {...handlers} selectedRunId="run-empty" selectedRunStatus="running" logs={[]} renderedLogs="" runLogsStatus="" canExport={false} taskrunPaths={[]} />);

    expect(screen.getByTestId("activity-drawer")).toHaveAccessibleName("Selected run activity drawer");
    expect(screen.getByTestId("activity-drawer")).not.toHaveAttribute("open");
    expect(screen.getByTestId("activity-drawer-toggle")).toHaveTextContent("Activity / Events");
    expect(screen.getByText("0 log entries for run-empty")).toBeInTheDocument();
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
        selectedRunError="runtime_contract_failed"
        logs={[]}
        renderedLogs=""
        runLogsStatus=""
        canExport={false}
        taskrunPaths={[]}
      />,
    );

    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("Run failed before log entries were captured: runtime_contract_failed");
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

    expect(screen.getByText("2 log entries for run-logs")).toBeInTheDocument();
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
