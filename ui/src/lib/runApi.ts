import { fetchJSON, getErrorMessage } from "./api";
import type { RunListResponse, RunReviewSummaryResponse, RunSnapshotResponse, RunStatusResponse } from "./appContracts";

export async function listPipelineRuns(limit = 100, init?: RequestInit): Promise<RunListResponse> {
  return fetchJSON<RunListResponse>(`/api/pipeline/runs?limit=${limit}`, init);
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
