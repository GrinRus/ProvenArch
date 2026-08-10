import type { Artifact } from "../../lib/appContracts";

export type ProposalReviewPackage = {
  name: string;
  artifacts: Artifact[];
};

export type ProposalReviewModel = {
  proposalArtifacts: Artifact[];
  proposalDocumentArtifacts: Artifact[];
  changelogArtifacts: Artifact[];
  evidenceArtifacts: Artifact[];
  packages: ProposalReviewPackage[];
  proposalDocumentCount: number;
  adrRfcCount: number;
  blockers: string[];
};

export function deriveProposalReviewModel({
  artifacts,
  openQuestionCount,
}: {
  artifacts: Artifact[];
  openQuestionCount: number;
}): ProposalReviewModel {
  const proposalArtifacts = artifacts
    .filter((artifact) => artifact.path.startsWith("proposals/") || artifact.path.startsWith("reports/changelog/"))
    .sort((left, right) => left.path.localeCompare(right.path));
  const changelogArtifacts = proposalArtifacts.filter((artifact) => artifact.path.startsWith("reports/changelog/"));
  const proposalDocumentArtifacts = proposalArtifacts.filter((artifact) => artifact.path.startsWith("proposals/"));
  const evidenceArtifacts = artifacts
    .filter((artifact) => artifact.path.startsWith("reports/findings/") || artifact.path.startsWith("reports/coverage/") || artifact.path.startsWith("reports/as-is/"))
    .sort((left, right) => left.path.localeCompare(right.path));
  const packages = groupProposalArtifacts(proposalArtifacts);
  const adrRfcCount = proposalDocumentArtifacts.filter((artifact) => /(^|\/)(ADR|RFC)\.md$/i.test(artifact.path)).length;
  const proposalDocumentCount = proposalDocumentArtifacts.filter((artifact) => artifact.path.endsWith(".md")).length;
  const blockers: string[] = [];
  if (proposalDocumentArtifacts.length === 0) {
    blockers.push("No proposal package artifacts are available.");
  }
  if (proposalDocumentArtifacts.length > 0 && adrRfcCount === 0) {
    blockers.push("Proposal package has no ADR or RFC draft artifact.");
  }
  if (proposalDocumentArtifacts.length > 0 && changelogArtifacts.length === 0) {
    blockers.push("No changelog artifact is linked to this proposal run.");
  }
  if (openQuestionCount > 0) {
    blockers.push(`${openQuestionCount} open questions remain from evidence review.`);
  }
  return {
    proposalArtifacts,
    proposalDocumentArtifacts,
    changelogArtifacts,
    evidenceArtifacts,
    packages,
    proposalDocumentCount,
    adrRfcCount,
    blockers,
  };
}

export function proposalPackageSuggestedFix(blocker: string): string {
  if (blocker.includes("No proposal package")) {
    return "Retry or rerun Analysis step4.proposals, then confirm a generated proposals/* artifact appears before Publish.";
  }
  if (blocker.includes("ADR or RFC")) {
    return "Generate or add an ADR/RFC draft under proposals/* so reviewers can see the decision record or implementation plan.";
  }
  if (blocker.includes("No changelog")) {
    return "Regenerate proposals so reports/changelog/* records the iteration changes linked to the package.";
  }
  if (blocker.includes("open questions")) {
    return "Resolve the Review open questions or record an explicit accepted gap before publication handoff.";
  }
  return "Inspect the proposal package artifacts, resolve blockers, and use Publish only after the package is complete.";
}

function groupProposalArtifacts(artifacts: Artifact[]): ProposalReviewPackage[] {
  const groups = new Map<string, Artifact[]>();
  for (const artifact of artifacts) {
    const name = proposalPackageName(artifact.path);
    groups.set(name, [...(groups.get(name) ?? []), artifact]);
  }
  return Array.from(groups.entries())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([name, groupArtifacts]) => ({ name, artifacts: groupArtifacts.sort((left, right) => left.path.localeCompare(right.path)) }));
}

function proposalPackageName(path: string): string {
  const parts = path.split("/").filter(Boolean);
  if (parts[0] === "proposals" && parts[1]) {
    return `${parts[0]}/${parts[1]}`;
  }
  if (parts[0] === "reports" && parts[1]) {
    return `${parts[0]}/${parts[1]}`;
  }
  return parts[0] || "root";
}

export function deriveProposalArtifactType(path: string): string {
  const basename = path.split("/").pop()?.toLowerCase() ?? "";
  if (path.startsWith("reports/changelog/")) {
    return "changelog";
  }
  if (basename === "adr.md") {
    return "ADR";
  }
  if (basename === "rfc.md") {
    return "RFC";
  }
  if (basename.includes("checklist")) {
    return "checklist";
  }
  if (basename.includes("proposal")) {
    return "proposal";
  }
  return "artifact";
}

export function proposalTabLabel(view: "preview" | "evidence" | "changelog" | "diff" | "logs"): string {
  if (view === "preview") {
    return "Preview";
  }
  if (view === "evidence") {
    return "Evidence";
  }
  if (view === "changelog") {
    return "Changelog";
  }
  if (view === "diff") {
    return "Diff";
  }
  return "Logs";
}
