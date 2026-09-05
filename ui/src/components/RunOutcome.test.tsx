import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { RunReviewSummaryResponse, RunStatusResponse } from "../lib/appContracts";
import { RecoveryPanel, RunResultPanel, StructuredRunProgress, TargetedRerunPanel } from "./RunOutcome";

const baseRun: RunStatusResponse = { run_id: "run-parent", pipeline: "refresh", status: "failed", started_at: "2026-08-03T10:00:00Z", current_step: "refresh.step1.collect", error_code: "provider_failed", error: "provider stopped" };

afterEach(() => vi.unstubAllGlobals());

describe("Run outcome surfaces", () => {
  it.each([
    ["completed", "Architecture updated"],
    ["completed_with_gaps", "Architecture updated with gaps"],
    ["failed", "Analysis needs recovery"],
    ["canceled", "Analysis canceled"],
  ] as const)("renders the %s terminal outcome", (state, title) => {
    const review = reviewWithResult(state);
    render(<RunResultPanel review={review} />);
    expect(screen.getByRole("heading", { name: title })).toBeInTheDocument();
    expect(screen.getByTestId("run-result-panel")).toHaveTextContent("Recommended next action");
  });

  it("shows structured stalled progress without counting provider activity as a percentage", () => {
    render(<StructuredRunProgress runStatus={{ ...baseRun, status: "running", progress: { phase: "stalled", completed_steps: 1, total_steps: 5, current_step: "refresh.step1.collect", expected_result: "Evidence-backed entities and relationships", planned_units: 5, running_units: 1, succeeded_units: 2, failed_units: 1, started_at: baseRun.started_at, last_activity_at: "2026-08-03T10:04:00Z", last_progress_at: "2026-08-03T10:01:00Z", artifact_state: "observed", repair_attempt: 1, repair_limit: 2, stall_deadline_at: "2026-08-03T10:06:00Z" } }} review={null} />);
    const progress = screen.getByTestId("analysis-run-progress");
    expect(progress).toHaveTextContent("stalled");
    expect(progress).toHaveTextContent("1/5");
    expect(progress).toHaveTextContent("2 complete · 1 running · 1 pending · 1 failed");
    expect(progress).toHaveTextContent("Repair attempt: 1/2");
    expect(progress).toHaveTextContent("Stall deadline:");
    expect(progress.querySelectorAll(".step-progress-track > span")).toHaveLength(5);
    expect(progress.querySelectorAll(".step-progress-track > .is-complete")).toHaveLength(1);
  });

  it("keeps failed legacy evidence readable without exposing retry mutations", () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    render(<RecoveryPanel runStatus={baseRun} review={{ ...reviewWithResult("failed"), recovery: { category: "provider", title: "Provider failed", explanation: "provider stopped", impact: "No promotion", retained_evidence: "Sibling shard retained", recommended_fix: "Restore provider", can_retry: true, failed_step: "refresh.step1.collect", failed_scopes: ["payments-a"], technical_code: "provider_failed" } }} />);
    expect(screen.getByRole("heading", { name: "Provider failed" })).toBeInTheDocument();
    expect(screen.getByTestId("analysis-failure-recovery")).toHaveTextContent("Sibling shard retained");
    expect(screen.getByTestId("legacy-read-only-recovery")).toHaveTextContent("Technical details remain readable");
    expect(screen.getByText("Technical error")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("keeps completed legacy runs read-only without planning a child run", () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    render(<TargetedRerunPanel runStatus={{ ...baseRun, status: "succeeded", error_code: null, error: null }} review={reviewWithResult("completed")} />);
    expect(screen.getByTestId("legacy-read-only-rerun")).toHaveTextContent("Selective reruns are unavailable for legacy evidence");
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();
  });
});

function reviewWithResult(state: "completed" | "completed_with_gaps" | "failed" | "canceled"): RunReviewSummaryResponse {
  return { run_id: "run-parent", pipeline: "refresh", status: state === "completed" || state === "completed_with_gaps" ? "succeeded" : state, started_at: "2026-08-03T10:00:00Z", steps: [], result: { state, summary: "Outcome summary", produced: { entities: 2 }, partial_scopes: state === "completed_with_gaps" ? 1 : 0, failed_scopes: state === "failed" ? 1 : 0, promotion: { changed: state === "completed" || state === "completed_with_gaps", current_usable: true, baseline_run_id: "run-baseline" }, recommended_action: "explore_architecture", coverage: { observed: 3, missing: 1, status: "partial" } } };
}
