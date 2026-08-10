import type { Artifact, GitDiffResponse } from "../../lib/appContracts";

export type PublishGateItem = {
  label: string;
  detail: string;
  tone: "info" | "ok" | "warn" | "error";
};

export function comparePublishArtifactPriority(left: Artifact, right: Artifact): number {
  const priority = (artifact: Artifact): number => {
    switch (artifact.path) {
      case "reports/as-is/overview.md":
        return 0;
      case "reports/findings/findings.md":
        return 1;
      case "reports/coverage/summary.md":
        return 2;
      case "reports/coverage/open-questions.md":
        return 3;
      default:
        return artifact.path.startsWith("proposals/") ? 4 : 5;
    }
  };
  const priorityDelta = priority(left) - priority(right);
  return priorityDelta !== 0 ? priorityDelta : left.path.localeCompare(right.path);
}

export function buildPublishFolderSummaries(artifacts: Artifact[]): Array<{ folder: string; count: number; sample: string }> {
  const grouped = new Map<string, Artifact[]>();
  for (const artifact of artifacts) {
    const folder = publishFolderLabel(artifact.path);
    grouped.set(folder, [...(grouped.get(folder) ?? []), artifact]);
  }
  return Array.from(grouped.entries())
    .map(([folder, items]) => ({
      folder,
      count: items.length,
      sample: items.slice(0, 2).map((item) => item.label || item.path).join(", "),
    }))
    .sort((left, right) => left.folder.localeCompare(right.folder));
}

function publishFolderLabel(path: string): string {
  const parts = path.split("/").filter(Boolean);
  if (parts[0] === "reports" && parts[1]) {
    return `reports/${parts[1]}`;
  }
  return parts[0] || "workspace";
}

export function gitDiffScopeTitle(gitDiff: GitDiffResponse | null): string {
  return gitDiff?.run_id ? "Selected run Git diff" : "Full workspace Git diff";
}

export function gitDiffScopeHint(gitDiff: GitDiffResponse | null): string {
  if (!gitDiff) {
    return "The full workspace Git inventory is required before publication can be declared clean or mutated.";
  }
  return "Full workspace view shows all uncommitted files in the local architecture workspace repository.";
}

export function buildPublishGateItems({
  previewArtifactCount,
  previewFolderCount,
  gitDiff,
  gitDiffStatus,
  gitMessage,
  proposalBranch,
  openQuestions,
}: {
  previewArtifactCount: number;
  previewFolderCount: number;
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  gitMessage: string;
  proposalBranch: string;
  openQuestions: string;
}): PublishGateItem[] {
  const openQuestionLines = openQuestions
    .split(/\r?\n/)
    .map((line) => line.trim().replace(/^[-*]\s*/, ""))
    .filter((line) => line.length > 0 && !line.startsWith("#"));
  const openQuestionCount = openQuestionLines.length;
  const firstOpenQuestion = openQuestionLines[0];
  const hasFullWorkspaceDiff = gitDiff?.scope === "full_workspace" && gitDiff.run_id == null;
  const gitInventoryReady = hasFullWorkspaceDiff && gitDiff.state !== "unknown" && gitDiff.state !== "blocked" && gitDiff.state !== "stale";
  const gitInventoryItem: PublishGateItem = gitInventoryReady
    ? {
        label: "Workspace Git inventory",
        detail: `${gitDiff.files.length} changed file${gitDiff.files.length === 1 ? "" : "s"} across ${gitDiff.folders.length} folders; this is the authoritative publication scope.`,
        tone: "ok",
      }
    : {
        label: "Workspace Git inventory",
        detail: gitDiffStatus || "Load the full workspace Git inventory before publication.",
        tone: "error",
      };
  return [
    gitInventoryItem,
    {
      label: previewArtifactCount > 0 ? "Preview artifacts" : "No preview artifacts",
      detail: previewArtifactCount > 0 ? `${previewArtifactCount} generated refs across ${previewFolderCount} preview folders. Preview does not limit the Git commit scope.` : "Run analysis before publishing workspace artifacts.",
      tone: previewArtifactCount > 0 ? "info" : "error",
    },
    {
      label: "Open questions",
      detail: openQuestionCount > 0 ? `${openQuestionCount} open question lines should be reviewed before commit. First: ${firstOpenQuestion}` : "No loaded open questions.",
      tone: openQuestionCount > 0 ? "warn" : "ok",
    },
    {
      label: "Message",
      detail: gitMessage.trim() ? gitMessage : "Commit message is empty.",
      tone: gitMessage.trim() ? "ok" : "warn",
    },
    {
      label: "Branch",
      detail: proposalBranch.trim() ? proposalBranch : "Proposal branch is optional but recommended for review.",
      tone: proposalBranch.trim() ? "ok" : "info",
    },
  ];
}

export function publishTabLabel(view: "preview" | "diff" | "evidence" | "changelog"): string {
  if (view === "preview") {
    return "Preview";
  }
  if (view === "diff") {
    return "Diff";
  }
  if (view === "evidence") {
    return "Evidence";
  }
  return "Changelog";
}

export function slugify(value: string): string {
  const normalized = value.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
  return normalized || "my-service";
}
