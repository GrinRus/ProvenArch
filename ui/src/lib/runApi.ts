import { fetchJSON, getErrorMessage } from "./api";
import type { RunListResponse, RunReviewSummaryResponse, RunSnapshotResponse, RunStartResponse, RunStatusResponse } from "./appContracts";

export async function listPipelineRuns(limit = 100, init?: RequestInit): Promise<RunListResponse> {
  return fetchJSON<RunListResponse>(`/api/pipeline/runs?limit=${limit}`, init);
}

export async function startPipelineRun(pipeline: "init" | "refresh", intent: "start" | "queue" = "start"): Promise<RunStartResponse> {
  return fetchJSON<RunStartResponse>(`/api/pipeline/${pipeline}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ trigger: "ui", intent, commit: false, create_proposal_branch: false }),
  });
}

export async function getPipelineRunStatus(id: string, allowMissing = false, init?: RequestInit): Promise<RunStatusResponse | null> {
  const response = await fetch(`/api/pipeline/runs/${id}`, init);
  const payload = await response.json();
  if (response.status === 404 && allowMissing) {
    return null;
  }
  if (!response.ok) {
    throw new Error(getErrorMessage(payload, `request failed: /api/pipeline/runs/${id}`));
  }
  return payload as RunStatusResponse;
}

export async function getPipelineRunReviewSummary(
  id: string,
  allowMissing = false,
  init?: RequestInit,
): Promise<RunReviewSummaryResponse | null> {
  const response = await fetch(`/api/pipeline/runs/${id}/review-summary`, init);
  const payload = await response.json();
  if (response.status === 404 && allowMissing) {
    return null;
  }
  if (!response.ok) {
    throw new Error(getErrorMessage(payload, `request failed: /api/pipeline/runs/${id}/review-summary`));
  }
  return payload as RunReviewSummaryResponse;
}

export async function getPipelineRunSnapshot(id: string, init?: RequestInit): Promise<RunSnapshotResponse> {
  return fetchJSON<RunSnapshotResponse>(`/api/pipeline/runs/${id}/snapshot`, init);
}

export type CancelRunResponse =
  | { status: 202; payload: unknown }
  | { status: 404; payload: unknown }
  | { status: 409; payload: unknown };

export async function requestRunCancel(runId: string): Promise<CancelRunResponse> {
  const response = await fetch(`/api/pipeline/runs/${runId}/cancel`, {
    method: "POST",
  });
  const payload = await response.json();
  if (response.status === 202 || response.status === 404 || response.status === 409) {
    return { status: response.status, payload } as CancelRunResponse;
  }
  throw new Error(getErrorMessage(payload, "failed to cancel selected run"));
}
