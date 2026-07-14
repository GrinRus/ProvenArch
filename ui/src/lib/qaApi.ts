import { fetchJSON } from "./api";

export type QAAskCitation = {
  path: string;
  reason: string;
};

export type QARunStartResponse = {
  run_id: string;
  status: string;
};

export type QARunResponse = {
  run_id: string;
  pipeline: "qa";
  status: "queued" | "running" | "succeeded" | "failed";
  started_at: string;
  finished_at?: string | null;
  question?: string | null;
  current_step?: string;
  runtime_provider?: string | null;
  provider?: string | null;
  answer?: string | null;
  citations?: QAAskCitation[] | null;
  unresolved?: string[] | null;
  confidence?: number | null;
  generated_at?: string | null;
  warnings?: string[];
  error_code?: string | null;
  error?: string | null;
};

export type QARunListResponse = {
  items: QARunResponse[];
};

export async function startQAQuestion(question: string): Promise<QARunStartResponse> {
  return fetchJSON<QARunStartResponse>("/api/qa/runs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ question }),
  });
}

export async function getQARun(runId: string, signal?: AbortSignal): Promise<QARunResponse> {
  return fetchJSON<QARunResponse>(`/api/qa/runs/${encodeURIComponent(runId)}`, signal ? { signal } : undefined);
}

export async function listQARuns(limit = 20, signal?: AbortSignal): Promise<QARunListResponse> {
  return fetchJSON<QARunListResponse>(`/api/qa/runs?limit=${limit}`, signal ? { signal } : undefined);
}
