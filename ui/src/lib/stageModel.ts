import type { StageId, StageOption, StageStatus } from "./consoleTypes";

export const stageDefinitions: Record<StageId, { label: string; description: string }> = {
  source: { label: "Source", description: "Repos & imports" },
  readiness: { label: "Readiness", description: "Validate & doctor" },
  charter: { label: "Charter", description: "Scope & rules" },
  analysis: { label: "Analysis", description: "Run pipeline" },
  review: { label: "Review", description: "Evidence & findings" },
  proposals: { label: "Proposals", description: "ADR/RFC drafts" },
  ask: { label: "Ask", description: "Read-only workspace Q&A" },
  publish: { label: "Publish", description: "Git workflow" },
};

export const stageOrder: StageId[] = ["source", "readiness", "charter", "analysis", "review", "proposals", "ask", "publish"];

export type StageStatusInput = {
  activeStage: StageId;
  hasManifest: boolean;
  readinessBlocked: boolean;
  readinessDone: boolean;
  charterStarted: boolean;
  analysisBlocked: boolean;
  selectedRunIsActive: boolean;
  runSucceeded: boolean;
  artifactCount: number;
  proposalArtifactCount: number;
  runningRunCount: number;
  hasGitStatus: boolean;
};

export function deriveStageStatuses(input: StageStatusInput): Record<StageId, StageStatus> {
  return {
    source: input.hasManifest ? "done" : "active",
    readiness: input.readinessBlocked ? "blocked" : input.readinessDone ? "done" : activeOrPending(input.activeStage, "readiness"),
    charter: input.charterStarted ? "done" : activeOrPending(input.activeStage, "charter"),
    analysis: input.analysisBlocked
      ? "blocked"
      : input.selectedRunIsActive
        ? "active"
        : input.runSucceeded
          ? "done"
          : activeOrPending(input.activeStage, "analysis"),
    review: input.artifactCount > 0 ? "done" : activeOrPending(input.activeStage, "review"),
    proposals: input.proposalArtifactCount > 0 ? "done" : activeOrPending(input.activeStage, "proposals"),
    ask: activeOrPending(input.activeStage, "ask"),
    publish: input.hasGitStatus ? "done" : activeOrPending(input.activeStage, "publish"),
  };
}

export function buildStageOptions(input: StageStatusInput): StageOption[] {
  const statuses = deriveStageStatuses(input);
  return stageOrder.map((id) => ({
    id,
    label: stageDefinitions[id].label,
    description: stageDefinitions[id].description,
    status: id === input.activeStage && statuses[id] !== "blocked" ? "active" : statuses[id],
    count: id === "review" && input.artifactCount > 0 ? input.artifactCount : id === "analysis" && input.runningRunCount > 0 ? input.runningRunCount : undefined,
    testId: `stage-${id}`,
  }));
}

function activeOrPending(activeStage: StageId, stage: StageId): StageStatus {
  return activeStage === stage ? "active" : "pending";
}
