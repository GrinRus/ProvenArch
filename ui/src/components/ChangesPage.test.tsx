import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { RunListItem } from "../lib/appContracts";
import { ChangesPage } from "./ChangesPage";

const runs: RunListItem[] = [
  { run_id: "run-good", pipeline: "init", status: "succeeded", started_at: "2026-07-15T00:00:00Z", authoritative_index: true },
  { run_id: "run-no-index", pipeline: "refresh", status: "succeeded", started_at: "2026-07-15T00:00:01Z", authoritative_index: false },
  { run_id: "run-failed", pipeline: "refresh", status: "failed", started_at: "2026-07-15T00:00:02Z", error_code: "runtime_failed" },
  { run_id: "qa-1", pipeline: "qa", status: "succeeded", started_at: "2026-07-15T00:00:03Z", authoritative_index: true },
];

describe("ChangesPage", () => {
  it("routes only successful indexed analysis runs to Change Review", () => {
    const onSelectChangeReview = vi.fn();
    const onOpenRunStudio = vi.fn();
    render(<ChangesPage runs={runs} selectedRunID={null} selectedEvidenceStatus="idle" view="overview" onViewChange={vi.fn()} onSelectChangeReview={onSelectChangeReview} onOpenRunStudio={onOpenRunStudio}>content</ChangesPage>);
    expect(screen.getByTestId("review-packages")).not.toHaveTextContent("qa-1");
    expect(screen.getAllByText("Publication: Unknown")).toHaveLength(3);
    fireEvent.click(screen.getByRole("button", { name: "Change Review" }));
    expect(onSelectChangeReview).toHaveBeenCalledWith("run-good");
    const studioButtons = screen.getAllByRole("button", { name: "Open Run Studio" });
    fireEvent.click(studioButtons[0]);
    fireEvent.click(studioButtons[1]);
    expect(onOpenRunStudio.mock.calls.flat()).toEqual(expect.arrayContaining(["run-no-index", "run-failed"]));
  });
});
