import { fetchJSON } from "./api";
import type { ArchitectureResponse, BaselineBundleResponse, KnowledgeResponse, ValidateResponse, WorkspaceHealthResponse } from "./appContracts";

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

export async function loadArchitectureAPI(): Promise<ArchitectureResponse> {
  const payload = await fetchJSON<Partial<ArchitectureResponse>>("/api/architecture");
  const levels = ["context", "container", "component", "code"] as const;
  if (!payload.views || !levels.every((level) => {
    const view = payload.views?.[level];
    return view && Array.isArray(view.nodes) && Array.isArray(view.edges);
  })) {
    throw new Error("Architecture response is incomplete");
  }
  return payload as ArchitectureResponse;
}

export function architectureFromKnowledge(knowledge: KnowledgeResponse): ArchitectureResponse {
  const levels: ArchitectureResponse["views"] = {} as ArchitectureResponse["views"];
  const visible = (level: keyof ArchitectureResponse["views"], type: string) => level === "context" ? ["service", "external.system", "team"].includes(type) : level === "container" ? ["service", "datastore", "external.system", "repo", "event.topic"].includes(type) : level === "component" ? ["api.http", "api.grpc", "event.topic"].includes(type) : ["api.http", "api.grpc"].includes(type);
  for (const level of ["context", "container", "component", "code"] as const) {
    const included = new Set(knowledge.entities.filter((entity) => visible(level, entity.type)).map((entity) => entity.id));
    const nodes = knowledge.entities.filter((entity) => included.has(entity.id)).map((entity) => ({ id: entity.id, name: entity.name, type: entity.type, owner_team_id: entity.owner_team_id, tags: entity.tags, confidence: entity.provenance.confidence, provenance_kind: entity.provenance.kind, evidence: Array.isArray(entity.provenance.evidence) ? entity.provenance.evidence as never : [], path: entity.path, available_levels: (["context", "container", "component", "code"] as const).filter((candidate) => visible(candidate, entity.type)), repositories: evidenceRepositories(entity.provenance.evidence), related_findings: [], related_questions: [] }));
    const edges = knowledge.edges.filter((edge) => included.has(edge.from) && included.has(edge.to)).map((edge) => ({ id: edge.id, from: edge.from, to: edge.to, type: edge.type, name: edge.name, confidence: edge.provenance.confidence, provenance_kind: edge.provenance.kind, evidence: Array.isArray(edge.provenance.evidence) ? edge.provenance.evidence as never : [], path: edge.path, repositories: evidenceRepositories(edge.provenance.evidence), related_findings: [], related_questions: [] }));
    levels[level] = { level, available: nodes.length > 0, unavailable_reason: nodes.length > 0 ? undefined : "No validated entities are available for this C4 level.", nodes, edges };
  }
  const evidence = knowledge.entities.reduce((sum, entity) => sum + (Array.isArray(entity.provenance.evidence) ? entity.provenance.evidence.length : 0), 0) + knowledge.edges.reduce((sum, edge) => sum + (Array.isArray(edge.provenance.evidence) ? edge.provenance.evidence.length : 0), 0);
  const empty = () => ({ added: [], changed: [], removed: [] });
  return { version: 1, generated_at: knowledge.generated_at, authority: { mode: "promoted_current", freshness: "unknown" }, status: knowledge.status, counts: { entities: knowledge.entities.length, edges: knowledge.edges.length, evidence, issues: knowledge.issues.length }, views: levels, exports: { home_path: knowledge.artifacts.some((item) => item.path === "reports/as-is/overview.md") ? "reports/as-is/overview.md" : undefined, c4_mermaid_paths: knowledge.artifacts.filter((item) => item.path.startsWith("reports/diagrams/") && item.path.endsWith(".mmd")).map((item) => item.path).sort() }, comparison: { available: false, reason: "A comparison will be available after two promoted architecture generations.", categories: { entities: empty(), edges: empty(), findings: empty(), gaps: empty() } }, review: { findings: [], questions: [] }, coverage: { observed: [], missing: [], notes: [] }, artifacts: knowledge.artifacts, issues: knowledge.issues };
}

function evidenceRepositories(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  const repos = value.map((item) => typeof item === "object" && item !== null && "repo" in item ? String((item as { repo?: unknown }).repo ?? "").trim() : "").filter(Boolean);
  return Array.from(new Set(repos)).sort();
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

export type GitPublicationContext = {
  taskId?: string;
  attemptId?: string;
  runId?: string;
};

function publicationContextPayload(context?: GitPublicationContext): Record<string, string> {
  return context?.taskId && context.attemptId && context.runId
    ? { task_id: context.taskId, attempt_id: context.attemptId, run_id: context.runId }
    : {};
}

export async function commitWorkspaceArtifacts(message: string, confirmation: GitConfirmationIdentity, context?: GitPublicationContext): Promise<{ status: string; message?: string; output?: string; publication?: { state: "linked" | "unavailable"; attempt_id?: string; unavailable_reason?: string } }> {
  return fetchJSON<{ status: string; message?: string; output?: string; publication?: { state: "linked" | "unavailable"; attempt_id?: string; unavailable_reason?: string } }>("/api/git/commit", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message, expected_fingerprint: confirmation.fingerprint, expected_head_oid: confirmation.headOID, ...publicationContextPayload(context) }),
  });
}

export async function createProposalBranch(name: string, confirmation: GitConfirmationIdentity, context?: GitPublicationContext): Promise<{ branch: string; publication?: { state: "linked" | "unavailable"; attempt_id?: string; unavailable_reason?: string } }> {
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
      ...publicationContextPayload(context),
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
