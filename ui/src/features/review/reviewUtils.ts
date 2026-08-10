import { parseTimeOrMin } from "../../lib/runState";
import type { Artifact, ReviewQueueItem, RunListItem } from "../../lib/appContracts";

export type ReviewTrustStatus = {
  label: string;
  title: string;
  detail: string;
  tone: "ok" | "warn" | "info";
};

export function reviewRouteDescription(view: "overview" | "evidence" | "findings" | "diff"): string {
  switch (view) {
    case "overview": return "Validate run identity, coverage and review readiness before publication.";
    case "evidence": return "Inspect immutable selected-run evidence and its exact artifact identity.";
    case "findings": return "Resolve findings, coverage questions and decision blockers.";
    case "diff": return "Inspect the server-authoritative current workspace Git diff.";
  }
}

export function findLastSuccessfulRun(runList: RunListItem[], currentRunID: string | null): RunListItem | null {
  const successfulRuns = runList
    .filter((run) => run.status === "succeeded" && run.run_id !== currentRunID)
    .sort((left, right) => parseTimeOrMin(right.started_at) - parseTimeOrMin(left.started_at));
  return successfulRuns[0] ?? null;
}

export function buildReviewQueue({
  artifacts,
  openQuestions,
  coverageSummary,
}: {
  artifacts: Artifact[];
  openQuestions: string;
  coverageSummary: string;
}): ReviewQueueItem[] {
  const queue: ReviewQueueItem[] = [];
  const addArtifact = (artifact: Artifact, kind: ReviewQueueItem["kind"], severity: ReviewQueueItem["severity"], title?: string) => {
    queue.push({
      id: `${kind}:${artifact.path}`,
      kind,
      severity,
      title: title || artifact.label || artifact.path,
      path: artifact.path,
    });
  };
  for (const artifact of artifacts) {
    if (artifact.path === "reports/as-is/overview.md") {
      addArtifact(artifact, "report", "info", "Review as-is overview");
    } else if (artifact.path === "reports/coverage/summary.md") {
      addArtifact(artifact, "coverage", coverageSummary.trim() ? "info" : "warn", "Review coverage summary");
    } else if (artifact.path === "reports/coverage/open-questions.md") {
      addArtifact(artifact, "question", openQuestions.trim() ? "warn" : "info", "Review open questions");
    } else if (artifact.path.startsWith("reports/findings/")) {
      addArtifact(artifact, "finding", "warn", "Review findings");
    } else if (artifact.path.startsWith("proposals/")) {
      addArtifact(artifact, "proposal", "info", "Review proposal");
    } else if (artifact.path.startsWith("model/")) {
      addArtifact(artifact, "model", "info", "Inspect derived model");
    } else if (artifact.path.startsWith("reports/diagrams/")) {
      addArtifact(artifact, "diagram", "info", "Inspect diagram");
    }
  }
  return queue
    .sort((left, right) => reviewQueuePriority(left) - reviewQueuePriority(right) || left.path.localeCompare(right.path))
    .slice(0, 16);
}

function reviewQueuePriority(item: ReviewQueueItem): number {
  if (item.kind === "question") {
    return 0;
  }
  if (item.kind === "finding") {
    return 1;
  }
  if (item.kind === "report" || item.kind === "coverage") {
    return 2;
  }
  if (item.kind === "proposal") {
    return 3;
  }
  return 5;
}

export function deriveReviewTrustStatus({
  artifactCount,
  hasCoverage,
  findingsCount,
  openQuestionCount,
}: {
  artifactCount: number;
  hasCoverage: boolean;
  findingsCount: number;
  openQuestionCount: number;
}): ReviewTrustStatus {
  if (artifactCount === 0) {
    return { label: "partial", title: "No evidence selected", detail: "Run Analysis to generate reviewable artifacts.", tone: "info" };
  }
  if (openQuestionCount > 0) {
    return { label: "review", title: "Review required", detail: "Open questions are present and should be resolved or accepted before publication.", tone: "warn" };
  }
  if (hasCoverage && findingsCount > 0) {
    return { label: "ready", title: "Evidence ready", detail: "Coverage and findings artifacts are available for human review.", tone: "ok" };
  }
  return { label: "partial", title: "Partial evidence", detail: "Some review artifacts are missing; inspect generated outputs before publication.", tone: "info" };
}

export function reviewDecisionSummary(trustStatus: ReviewTrustStatus, openQuestionCount: number): string {
  if (openQuestionCount > 0) {
    return `${openQuestionCount} open question${openQuestionCount === 1 ? "" : "s"} require review, but they are not a hard publish blocker.`;
  }
  if (trustStatus.tone === "ok") {
    return "Evidence can move to proposal or publish review after human confirmation.";
  }
  return "Treat this as partial evidence and inspect generated artifacts before publication.";
}

export function countMarkdownItems(content: string): number {
  return content
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.startsWith("- ") || /^\d+\./.test(line)).length;
}
