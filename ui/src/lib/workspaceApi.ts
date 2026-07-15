import { fetchJSON } from "./api";
import type { BaselineBundleResponse, KnowledgeResponse, ValidateResponse, WorkspaceHealthResponse } from "./appContracts";

export async function loadWorkspaceManifest(): Promise<string> {
  const manifest = await fetchJSON<{ content: string }>("/api/workspace/manifest");
  return manifest.content ?? "";
}

export async function loadBaselineBundleAPI(): Promise<BaselineBundleResponse> {
  return fetchJSON<BaselineBundleResponse>("/api/workspace/bundle");
}

export async function loadWorkspaceHealthAPI(): Promise<WorkspaceHealthResponse> {
  const payload = await fetchJSON<Partial<WorkspaceHealthResponse>>("/api/workspace/health");
  return normalizeWorkspaceHealthResponse(payload);
}

export async function loadKnowledgeAPI(): Promise<KnowledgeResponse> {
  return fetchJSON<KnowledgeResponse>("/api/knowledge");
}

export async function loadArtifactText(path: string): Promise<string | null> {
  const response = await fetch(`/api/artifacts?path=${encodeURIComponent(path)}`);
  if (!response.ok) {
    return null;
  }
  return response.text();
}

export async function validateWorkspaceAPI(): Promise<ValidateResponse> {
  const response = await fetch("/api/workspace/validate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  });
  const payload = (await response.json()) as ValidateResponse;
  return payload;
}

export async function saveWorkspaceManifest(content: string): Promise<void> {
  await fetchJSON<{ ok: boolean }>("/api/workspace/manifest", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  });
}

export async function saveEditableArtifact(path: string, content: string): Promise<void> {
  await fetchJSON<{ ok: boolean }>("/api/artifacts/write", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, content }),
  });
}

export type GitConfirmationIdentity = {
  fingerprint: string;
  headOID: string;
  sourceBranch: string;
  baseRef: string;
  baseOID: string;
};

export async function commitWorkspaceArtifacts(message: string, confirmation: GitConfirmationIdentity): Promise<{ status: string; message?: string; output?: string }> {
  return fetchJSON<{ status: string; message?: string; output?: string }>("/api/git/commit", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message, expected_fingerprint: confirmation.fingerprint, expected_head_oid: confirmation.headOID }),
  });
}

export async function createProposalBranch(name: string, confirmation: GitConfirmationIdentity): Promise<{ branch: string }> {
  return fetchJSON<{ branch: string }>("/api/git/proposal-branch", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name,
      expected_fingerprint: confirmation.fingerprint,
      expected_source_branch: confirmation.sourceBranch,
      expected_base_ref: confirmation.baseRef,
      expected_base_oid: confirmation.baseOID,
      expected_head_oid: confirmation.headOID,
    }),
  });
}

function normalizeWorkspaceHealthResponse(payload: Partial<WorkspaceHealthResponse> | null | undefined): WorkspaceHealthResponse {
  const items = Array.isArray(payload?.items) ? payload.items : [];
  const summary = payload?.summary ?? { info: 0, warning: 0, error: 0 };
  const status = payload?.status === "warn" || payload?.status === "fail" ? payload.status : "pass";
  return {
    version: typeof payload?.version === "number" ? payload.version : 1,
    generated_at: typeof payload?.generated_at === "string" ? payload.generated_at : "",
    status,
    summary: {
      info: typeof summary.info === "number" ? summary.info : 0,
      warning: typeof summary.warning === "number" ? summary.warning : 0,
      error: typeof summary.error === "number" ? summary.error : 0,
    },
    items,
  };
}
