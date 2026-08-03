import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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

  it("previews reuse and execution closure before starting a targeted child run", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      const body = JSON.parse(String(init?.body || "{}")) as { plan_hash?: string };
      if (body.plan_hash) return response({ run_id: "run-child", status: "started", parent_run_id: "run-parent" }, 202);
      return response({ parent_run_id: "run-parent", pipeline: "refresh", requested_step: "refresh.step1.collect", effective_start_step: "refresh.step1.collect", requested_scopes: ["payments-a"], reused_inputs: ["validated_sibling_collect_scopes"], execute_steps: ["refresh.step1.collect", "refresh.step2.asis_docs", "refresh.step3.findings", "refresh.step4.proposals"], invalidated_steps: ["refresh.step1.collect", "refresh.step2.asis_docs", "refresh.step3.findings", "refresh.step4.proposals"], estimated_units: 4, widened: false, plan_hash: "plan-1" });
    });
    vi.stubGlobal("fetch", fetchMock);
    const onRetryStarted = vi.fn();
    render(<RecoveryPanel runStatus={baseRun} review={{ ...reviewWithResult("failed"), recovery: { category: "provider", title: "Provider failed", explanation: "provider stopped", impact: "No promotion", retained_evidence: "Sibling shard retained", recommended_fix: "Restore provider", can_retry: true, failed_step: "refresh.step1.collect", failed_scopes: ["payments-a"], technical_code: "provider_failed" } }} busy={false} onRetryStarted={onRetryStarted} onReviewDetails={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: "Calculate retry plan" }));
    const plan = await screen.findByTestId("retry-plan");
    expect(plan).toHaveTextContent("validated_sibling_collect_scopes");
    expect(plan).toHaveTextContent("refresh.step4.proposals");
    fireEvent.click(screen.getByRole("button", { name: "Start targeted retry" }));
    await waitFor(() => expect(onRetryStarted).toHaveBeenCalledWith("run-child"));
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("lets a completed run rerun a selected step through a child-run plan", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      const body = JSON.parse(String(init?.body || "{}")) as { step_id?: string; plan_hash?: string };
      if (body.plan_hash) return response({ run_id: "run-child", status: "started", parent_run_id: "run-parent" }, 202);
      return response({ parent_run_id: "run-parent", pipeline: "refresh", requested_step: body.step_id, effective_start_step: body.step_id, requested_scopes: [], effective_scopes: [], reused_inputs: ["refresh.step1.collect", "refresh.step2.asis_docs"], execute_steps: ["refresh.step3.findings", "refresh.step4.proposals"], invalidated_steps: ["refresh.step3.findings", "refresh.step4.proposals"], estimated_units: 2, widened: false, plan_hash: "plan-success" });
    });
    vi.stubGlobal("fetch", fetchMock);
    const onRetryStarted = vi.fn();
    render(<TargetedRerunPanel runStatus={{ ...baseRun, status: "succeeded", error_code: null, error: null }} review={reviewWithResult("completed")} busy={false} onRetryStarted={onRetryStarted} />);
    fireEvent.change(screen.getByLabelText("Start from step"), { target: { value: "refresh.step3.findings" } });
    fireEvent.click(screen.getByRole("button", { name: "Review rerun plan" }));
    expect(await screen.findByTestId("retry-plan")).toHaveTextContent("refresh.step3.findings → refresh.step4.proposals");
    fireEvent.click(screen.getByRole("button", { name: "Start targeted rerun" }));
    await waitFor(() => expect(onRetryStarted).toHaveBeenCalledWith("run-child"));
  });
});

function reviewWithResult(state: "completed" | "completed_with_gaps" | "failed" | "canceled"): RunReviewSummaryResponse {
  return { run_id: "run-parent", pipeline: "refresh", status: state === "completed" || state === "completed_with_gaps" ? "succeeded" : state, started_at: "2026-08-03T10:00:00Z", steps: [], result: { state, summary: "Outcome summary", produced: { entities: 2 }, partial_scopes: state === "completed_with_gaps" ? 1 : 0, failed_scopes: state === "failed" ? 1 : 0, promotion: { changed: state === "completed" || state === "completed_with_gaps", current_usable: true, baseline_run_id: "run-baseline" }, recommended_action: "explore_architecture", coverage: { observed: 3, missing: 1, status: "partial" } } };
}

function response(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), { status, headers: { "Content-Type": "application/json" } });
}
