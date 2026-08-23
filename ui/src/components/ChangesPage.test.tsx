import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { ArchitectureComparison, RunListItem, RunReviewContract } from "../lib/appContracts";
import { ChangesPage } from "./ChangesPage";

const runs: RunListItem[] = [
  { run_id: "run-good", pipeline: "init", status: "succeeded", started_at: "2026-07-15T00:00:00Z", authoritative_index: true },
  { run_id: "run-no-index", pipeline: "refresh", status: "succeeded", started_at: "2026-07-15T00:00:01Z", authoritative_index: false },
  { run_id: "run-failed", pipeline: "refresh", status: "failed", started_at: "2026-07-15T00:00:02Z", error_code: "runtime_failed" },
  { run_id: "qa-1", pipeline: "qa", status: "succeeded", started_at: "2026-07-15T00:00:03Z", authoritative_index: true },
];

describe("ChangesPage", () => {
  it("routes only successful indexed analysis runs to architecture review", () => {
    const onSelectChangeReview = vi.fn();
    const onOpenRunStudio = vi.fn();
    render(<ChangesPage runs={runs} selectedRunID={null} selectedEvidenceStatus="idle" view="overview" onViewChange={vi.fn()} onSelectChangeReview={onSelectChangeReview} onOpenRunStudio={onOpenRunStudio}>content</ChangesPage>);
    expect(screen.getByTestId("review-packages")).not.toHaveTextContent("qa-1");
    expect(screen.getAllByText("Publication: Unknown")).toHaveLength(3);
    fireEvent.click(screen.getByRole("button", { name: "Review architecture" }));
    expect(onSelectChangeReview).toHaveBeenCalledWith("run-good");
    const studioButtons = screen.getAllByRole("button", { name: "Open recovery" });
    fireEvent.click(studioButtons[0]);
    fireEvent.click(studioButtons[1]);
    expect(onOpenRunStudio.mock.calls.flat()).toEqual(expect.arrayContaining(["run-no-index", "run-failed"]));
  });
});

it("explains no-op refresh packages", () => {
  const noopRuns: RunListItem[] = [{ run_id: "run-noop", pipeline: "refresh", status: "succeeded", started_at: "2026-07-15T12:00:00Z", authoritative_index: true, refresh_summary: { mode: "no_op", decision: "unchanged_candidate", baseline_run_id: "run-base", reason_codes: ["source_revisions_unchanged"], artifact_path: "reports/taskruns/run-noop/refresh-execution.json", updated: 0, preserved: 3, removed: 0, uncertain: 0 } }];
  render(<ChangesPage runs={noopRuns} selectedRunID={null} selectedEvidenceStatus="idle" view="overview" onViewChange={vi.fn()} onSelectChangeReview={vi.fn()} onOpenRunStudio={vi.fn()}>content</ChangesPage>);
  expect(screen.getByText("No changes in analysis scope")).toBeInTheDocument();
});

it("fails closed when the promoted comparison belongs to another run", () => {
  const comparison: ArchitectureComparison = {
    available: true,
    baseline_run_id: "run-base",
    current_run_id: "run-current",
    categories: {
      entities: { added: [{ id: "entity-1", name: "Entity 1" }], changed: [], removed: [] },
      edges: { added: [], changed: [], removed: [] },
      findings: { added: [], changed: [], removed: [] },
      gaps: { added: [], changed: [], removed: [] },
    },
  };
  render(<ChangesPage runs={runs} selectedRunID="run-selected" selectedEvidenceStatus="idle" view="findings" onViewChange={vi.fn()} onSelectChangeReview={vi.fn()} onOpenRunStudio={vi.fn()} architectureComparison={comparison} architectureComparisonMismatch>content</ChangesPage>);
  expect(screen.getByText("Comparison unavailable for selected run")).toBeInTheDocument();
  expect(screen.queryByText("What changed in the architecture")).not.toBeInTheDocument();
  expect(screen.getByText("The promoted architecture comparison belongs to another run, so no current delta is shown here.")).toBeInTheDocument();
});

it("renders a run-pinned initial summary instead of a baseline delta", () => {
  const review: RunReviewContract = {
    review_kind: "initial",
    source_run_id: "run-good",
    semantic_changes: { available: false, current_run_id: "run-good", reason: "Initial architecture summary", categories: { entities: { added: [], changed: [], removed: [] }, edges: { added: [], changed: [], removed: [] }, findings: { added: [], changed: [], removed: [] }, gaps: { added: [], changed: [], removed: [] } } },
    document_changes: { available: true, added: [], changed: [], removed: [] },
    findings: [],
    questions: [],
    gaps: [],
    summary: { entities_added: 0, entities_changed: 0, entities_removed: 0, edges_added: 0, edges_changed: 0, edges_removed: 0, documents_added: 0, documents_changed: 0, documents_removed: 0, findings: 0, questions: 0, gaps: 0 },
    runtime: { mode: "fake", providers: ["fake"], step_providers: {} },
    authority: { mode: "promoted_run_snapshot", source_run_id: "run-good" },
    generated_at: "2026-08-04T20:00:00Z",
  };
  render(<ChangesPage runs={runs} selectedRunID="run-good" selectedEvidenceStatus="available" view="findings" onViewChange={vi.fn()} onSelectChangeReview={vi.fn()} onOpenRunStudio={vi.fn()} runReview={review}>content</ChangesPage>);
  expect(screen.getByText("Initial architecture summary")).toBeInTheDocument();
  expect(screen.getByTestId("run-pinned-review-summary")).toHaveTextContent("fake");
  expect(screen.queryByText("What changed in the architecture")).not.toBeInTheDocument();
});

it("keeps current workspace evidence read-only and does not reuse a run review", () => {
  const review: RunReviewContract = {
    review_kind: "initial",
    source_run_id: "run-good",
    semantic_changes: { available: false, current_run_id: "run-good", categories: { entities: { added: [], changed: [], removed: [] }, edges: { added: [], changed: [], removed: [] }, findings: { added: [], changed: [], removed: [] }, gaps: { added: [], changed: [], removed: [] } } },
    document_changes: { available: false, added: [], changed: [], removed: [] },
    findings: [], questions: [], gaps: [],
    summary: { entities_added: 0, entities_changed: 0, entities_removed: 0, edges_added: 0, edges_changed: 0, edges_removed: 0, documents_added: 0, documents_changed: 0, documents_removed: 0, findings: 0, questions: 0, gaps: 0 },
    runtime: { mode: "fake", providers: ["fake"], step_providers: {} },
    authority: { mode: "promoted_run_snapshot", source_run_id: "run-good" },
    generated_at: "2026-08-04T20:00:00Z",
  };
  render(<ChangesPage runs={runs} selectedRunID="run-good" selectedEvidenceStatus="available" sourceMode="current" view="evidence" onViewChange={vi.fn()} onSelectChangeReview={vi.fn()} onOpenRunStudio={vi.fn()} runReview={review}>content</ChangesPage>);
  expect(screen.getByTestId("changes-read-only-badge")).toHaveTextContent("Read-only workspace");
  expect(screen.getByTestId("stage-evidence")).toHaveAttribute("aria-current", "page");
  expect(screen.queryByTestId("stage-publish")).not.toBeInTheDocument();
  expect(screen.queryByText("Initial architecture summary")).not.toBeInTheDocument();
});

it("keeps failed runs in Run Studio instead of presenting a publishable review", () => {
  render(<ChangesPage runs={runs} selectedRunID="run-failed" selectedEvidenceStatus="available" view="overview" onViewChange={vi.fn()} onSelectChangeReview={vi.fn()} onOpenRunStudio={vi.fn()}>content</ChangesPage>);
  expect(screen.getByText("Recovery is required before review")).toBeInTheDocument();
  expect(screen.getByTestId("changes-open-run-studio")).toBeInTheDocument();
  expect(screen.queryByTestId("stage-publish")).not.toBeInTheDocument();
});

it("keeps Changes scoped to the exact Task identity without latest-run fallback", () => {
  const onOpenTask = vi.fn();
  render(<ChangesPage runs={runs} selectedRunID="run-good" selectedEvidenceStatus="available" view="overview" taskId="task-opaque-23" onOpenTask={onOpenTask} onViewChange={vi.fn()} onSelectChangeReview={vi.fn()} onOpenRunStudio={vi.fn()}>content</ChangesPage>);
  const context = screen.getByTestId("task-changes-context");
  expect(context).toHaveTextContent("task-opaque-23");
  expect(context).toHaveTextContent("No latest-run fallback");
  fireEvent.click(screen.getByRole("button", { name: "Back to Task" }));
  expect(onOpenTask).toHaveBeenCalledWith("task-opaque-23");
});
