import type { Artifact } from "./appContracts";

export type ReviewArtifactFilter = "all" | "reports" | "diagrams" | "proposals" | "runtime";
export type PublishArtifactFilter = "all" | "changed" | "reports" | "proposals" | "diagrams" | "taskruns";

export const REVIEW_ARTIFACT_FILTERS: Array<{ id: ReviewArtifactFilter; label: string }> = [
  { id: "all", label: "All" },
  { id: "reports", label: "Reports" },
  { id: "diagrams", label: "Diagrams" },
  { id: "proposals", label: "Proposals" },
  { id: "runtime", label: "Runtime" },
];

export const PUBLISH_ARTIFACT_FILTERS: Array<{ id: PublishArtifactFilter; label: string }> = [
  { id: "all", label: "All" },
  { id: "changed", label: "Changed" },
  { id: "reports", label: "Reports" },
  { id: "proposals", label: "Proposals" },
  { id: "diagrams", label: "Diagrams" },
  { id: "taskruns", label: "Taskruns" },
];

export type ArtifactGroup = {
  name: string;
  artifacts: Artifact[];
};

export function groupArtifactsByFolder(artifacts: Artifact[]): ArtifactGroup[] {
  const groups = new Map<string, Artifact[]>();
  for (const artifact of artifacts) {
    const name = reviewArtifactGroupName(artifact.path);
    groups.set(name, [...(groups.get(name) ?? []), artifact]);
  }
  return Array.from(groups.entries())
    .sort(([left], [right]) => reviewArtifactGroupPriority(left) - reviewArtifactGroupPriority(right) || left.localeCompare(right))
    .map(([name, groupArtifacts]) => ({ name, artifacts: groupArtifacts.sort((left, right) => left.path.localeCompare(right.path)) }));
}

export function filterReviewArtifactGroups(groups: ArtifactGroup[], filter: ReviewArtifactFilter): ArtifactGroup[] {
  if (filter === "all") {
    return groups;
  }
  return groups
    .map((group) => ({
      ...group,
      artifacts: group.artifacts.filter((artifact) => reviewArtifactMatchesFilter(artifact, filter)),
    }))
    .filter((group) => group.artifacts.length > 0);
}

export function reviewArtifactFilterLabel(filter: ReviewArtifactFilter): string {
  return REVIEW_ARTIFACT_FILTERS.find((option) => option.id === filter)?.label ?? "Selected";
}

function reviewArtifactMatchesFilter(artifact: Artifact, filter: ReviewArtifactFilter): boolean {
  const path = artifact.path;
  if (filter === "diagrams") {
    return artifact.kind === "diagram" || path.startsWith("reports/diagrams/");
  }
  if (filter === "proposals") {
    return artifact.kind === "proposal" || artifact.kind === "changelog" || path.startsWith("proposals/") || path.startsWith("reports/changelog/");
  }
  if (filter === "runtime") {
    return artifact.kind === "taskrun" || path.startsWith("reports/taskruns/");
  }
  if (filter === "reports") {
    return (
      (artifact.kind === "report" || artifact.kind === "agent-output" || path.startsWith("reports/")) &&
      !path.startsWith("reports/diagrams/") &&
      !path.startsWith("reports/changelog/") &&
      !path.startsWith("reports/taskruns/")
    );
  }
  return true;
}

function reviewArtifactGroupName(path: string): string {
  const parts = path.split("/").filter(Boolean);
  if (parts.length === 0) {
    return "root";
  }
  if (parts[0] === "reports" && parts[1]) {
    return `${parts[0]}/${parts[1]}`;
  }
  if (parts[0] === "model" && parts[1]) {
    return `${parts[0]}/${parts[1]}`;
  }
  return parts[0];
}

function reviewArtifactGroupPriority(name: string): number {
  if (name === "reports/as-is") {
    return 0;
  }
  if (name === "reports/coverage") {
    return 1;
  }
  if (name === "reports/findings") {
    return 2;
  }
  if (name === "reports/diagrams") {
    return 3;
  }
  if (name.startsWith("model/")) {
    return 4;
  }
  if (name === "reports/agent-outputs") {
    return 5;
  }
  if (name === "proposals") {
    return 6;
  }
  if (name === "reports/changelog") {
    return 7;
  }
  if (name === "reports/taskruns") {
    return 8;
  }
  return 20;
}

export function reviewArtifactGroupCategory(name: string): string {
  if (name.startsWith("reports/diagrams")) {
    return "is-diagram-group";
  }
  if (name.startsWith("reports/")) {
    return "is-report-group";
  }
  if (name.startsWith("model/")) {
    return "is-model-group";
  }
  if (name === "proposals" || name === "reports/changelog") {
    return "is-proposal-group";
  }
  return "is-supporting-group";
}

export function reviewArtifactGroupCategoryLabel(name: string): string {
  if (name.startsWith("reports/diagrams")) {
    return "diagram";
  }
  if (name.startsWith("reports/")) {
    return "report";
  }
  if (name.startsWith("model/")) {
    return "model";
  }
  if (name === "proposals" || name === "reports/changelog") {
    return "proposal";
  }
  return "support";
}

export function publishArtifactMatchesFilter(artifact: Artifact, filter: PublishArtifactFilter, changedPathSet: Set<string>): boolean {
  const path = artifact.path;
  if (filter === "all") {
    return true;
  }
  if (filter === "changed") {
    return changedPathSet.has(path);
  }
  if (filter === "reports") {
    return (
      (artifact.kind === "report" || artifact.kind === "agent-output" || path.startsWith("reports/")) &&
      !path.startsWith("reports/diagrams/") &&
      !path.startsWith("reports/changelog/") &&
      !path.startsWith("reports/taskruns/")
    );
  }
  if (filter === "proposals") {
    return artifact.kind === "proposal" || artifact.kind === "changelog" || path.startsWith("proposals/") || path.startsWith("reports/changelog/");
  }
  if (filter === "diagrams") {
    return artifact.kind === "diagram" || path.startsWith("reports/diagrams/");
  }
  if (filter === "taskruns") {
    return artifact.kind === "taskrun" || path.startsWith("reports/taskruns/");
  }
  return true;
}

export function publishArtifactFilterLabel(filter: PublishArtifactFilter): string {
  return PUBLISH_ARTIFACT_FILTERS.find((option) => option.id === filter)?.label ?? "Selected";
}
