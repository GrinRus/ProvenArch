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
  status: "queued" | "running" | "succeeded" | "failed" | "canceled";
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
  answer_status?: "available" | "not_produced";
  answer_digest?: string | null;
  answer_authority?: EvidenceAuthority;
  audit_authority?: EvidenceAuthority;
  warnings?: string[];
  error_code?: string | null;
  error?: string | null;
};

export type QAProposalDraftRequest = {
  title: string;
  expected_answer_digest: string;
  slug?: string;
  operator_note?: string;
};

export type QAProposalDraftResponse = {
  path: string;
  proposal_path: string;
  evidence_path: string;
  source_path: string;
  answer_digest: string;
};

export type EvidenceAuthority = {
  mode: "promoted_current" | "run_snapshot" | "qa_snapshot" | "qa_audit";
  run_id?: string;
  root: string;
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

export async function createQAProposalDraft(runId: string, request: QAProposalDraftRequest): Promise<QAProposalDraftResponse> {
  return fetchJSON<QAProposalDraftResponse>(`/api/qa/runs/${encodeURIComponent(runId)}/proposal-draft`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}
