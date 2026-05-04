import { fetchJSON } from "./api";
import type { BaselineBundleResponse, ValidateResponse } from "./appContracts";

export async function loadWorkspaceManifest(): Promise<string> {
  const manifest = await fetchJSON<{ content: string }>("/api/workspace/manifest");
  return manifest.content ?? "";
}

export async function loadBaselineBundleAPI(): Promise<BaselineBundleResponse> {
  return fetchJSON<BaselineBundleResponse>("/api/workspace/bundle");
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

export async function commitWorkspaceArtifacts(message: string): Promise<{ status: string; message?: string; output?: string }> {
  return fetchJSON<{ status: string; message?: string; output?: string }>("/api/git/commit", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message }),
  });
}

export async function createProposalBranch(name: string): Promise<{ branch: string }> {
  return fetchJSON<{ branch: string }>("/api/git/proposal-branch", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
}
