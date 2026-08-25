import type { KnowledgeResponse, RunListItem } from "../../lib/appContracts";
import type { ProductTask } from "../../lib/taskApi";

export type ChangeReviewItem = RunListItem & {
  action: "review" | "run_studio";
  publication: "unknown";
  task_id: string;
  task_title: string;
  task_goal: string;
  attempt_id: string;
};

export function buildChangeReviewModel(tasks: ProductTask[], runs: RunListItem[]): ChangeReviewItem[] {
  return tasks
    .filter((task) => task.outcome.state === "available" && Boolean(task.outcome.run_id && task.outcome.attempt_id))
    .map((task) => {
      const runID = task.outcome.run_id as string;
      const attemptID = task.outcome.attempt_id as string;
      const run = runs.find((item) => item.run_id === runID);
      return {
        ...(run ?? {
          run_id: runID,
          pipeline: "init",
          status: "succeeded" as const,
          started_at: task.updated_at,
          authoritative_index: true,
        }),
        task_id: task.task_id,
        task_title: task.title,
        task_goal: task.goal,
        attempt_id: attemptID,
        action: run && (run.status !== "succeeded" || run.authoritative_index !== true)
          ? "run_studio" as const
          : "review" as const,
        publication: "unknown" as const,
      };
    });
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
