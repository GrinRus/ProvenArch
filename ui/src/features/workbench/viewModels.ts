import type { KnowledgeResponse, RunListItem } from "../../lib/appContracts";

export type ChangeReviewItem = RunListItem & { action: "review" | "run_studio"; publication: "unknown" };

export function buildChangeReviewModel(runs: RunListItem[], selectedRunID: string | null, selectedEvidenceStatus: string): ChangeReviewItem[] {
  return runs
    .filter((run) => run.pipeline === "init" || run.pipeline === "refresh")
    .map((run) => ({
      ...run,
      action: run.status === "succeeded" && run.authoritative_index === true
        && (run.run_id !== selectedRunID || selectedEvidenceStatus === "available" || selectedEvidenceStatus === "partial")
        ? "review" as const
        : "run_studio" as const,
      publication: "unknown" as const,
    }));
}

export type KnowledgeViewModel = {
  status: "loading" | "error" | "available" | "partial" | "unavailable";
  entities: KnowledgeResponse["entities"];
  filteredEntities: KnowledgeResponse["entities"];
  edges: KnowledgeResponse["edges"];
  artifacts: KnowledgeResponse["artifacts"];
  issues: KnowledgeResponse["issues"];
  selectedEntity?: KnowledgeResponse["entities"][number];
};

export function buildKnowledgeViewModel(knowledge: KnowledgeResponse | null, loading: boolean, error: string, query: string, selectedEntityID?: string): KnowledgeViewModel {
  const entities = knowledge?.entities ?? [];
  const normalized = query.trim().toLowerCase();
  return {
    status: loading ? "loading" : error ? "error" : knowledge?.status ?? "unavailable",
    entities,
    filteredEntities: normalized
      ? entities.filter((entity) => `${entity.id} ${entity.name} ${entity.type} ${(entity.tags ?? []).join(" ")}`.toLowerCase().includes(normalized))
      : entities,
    edges: knowledge?.edges ?? [],
    artifacts: knowledge?.artifacts ?? [],
    issues: knowledge?.issues ?? [],
    selectedEntity: entities.find((entity) => entity.id === selectedEntityID),
  };
}

export type PublishViewModel = {
  changeCount: number;
  questionCount: number;
  blockerCount: number;
  evidenceIdentity: "demo" | "live";
  actionLabel: "Commit all demo workspace changes" | "Commit all workspace changes";
};

export function buildPublishViewModel(changeCount: number, questionCount: number, blockerCount: number, demo: boolean): PublishViewModel {
  return {
    changeCount,
    questionCount,
    blockerCount,
    evidenceIdentity: demo ? "demo" : "live",
    actionLabel: demo ? "Commit all demo workspace changes" : "Commit all workspace changes",
  };
}
