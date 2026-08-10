import type { Artifact } from "../../lib/appContracts";
import type { ReviewDomainMapEdge, ReviewDomainMapModel, ReviewDomainMapNode } from "../../components/ReviewDomainMap";

const MODEL_EDGE_TYPES = ["publishes", "subscribes", "calls", "reads", "writes", "exposes"] as const;

const DOMAIN_MAP_GROUPS: Array<{ key: string; label: string }> = [
  { key: "domains", label: "Domains" },
  { key: "services", label: "Services" },
  { key: "interfaces", label: "Interfaces / topics" },
  { key: "data", label: "Data stores" },
  { key: "external", label: "External systems" },
  { key: "ownership", label: "Ownership / repos" },
  { key: "other", label: "Other model artifacts" },
];

export function deriveReviewDomainMap({
  artifacts,
  coverageSummary,
  openQuestionCount,
}: {
  artifacts: Artifact[];
  coverageSummary: string;
  openQuestionCount: number;
}): ReviewDomainMapModel {
  const domainOutputs = artifacts
    .filter((artifact) => artifact.path.startsWith("reports/agent-outputs/domains/") && artifact.path.endsWith(".md"))
    .sort((left, right) => left.path.localeCompare(right.path));
  const entityArtifacts = artifacts
    .filter((artifact) => artifact.path.startsWith("model/entities/") && (artifact.path.endsWith(".yaml") || artifact.path.endsWith(".yml")))
    .sort((left, right) => left.path.localeCompare(right.path));
  const edgeArtifacts = artifacts
    .filter((artifact) => artifact.path.startsWith("model/edges/") && (artifact.path.endsWith(".yaml") || artifact.path.endsWith(".yml")))
    .sort((left, right) => left.path.localeCompare(right.path));
  const proposalArtifacts = artifacts
    .filter((artifact) => artifact.path.startsWith("proposals/") || artifact.path.startsWith("reports/changelog/"))
    .sort((left, right) => left.path.localeCompare(right.path));

  const domainNodes: ReviewDomainMapNode[] = domainOutputs.map((artifact) => {
    const id = stripArtifactSuffix(artifact.path.split("/").pop() ?? artifact.path);
    return { id, label: artifact.label?.trim() || humanizeModelID(id), typeLabel: "domain output", group: "domains", kind: "domain", artifact };
  });
  const entityNodes = entityArtifacts.map((artifact) => {
    const id = stripArtifactSuffix(artifact.path.replace(/^model\/entities\//, ""));
    const meta = deriveEntityMapMeta(id);
    return { id, label: artifact.label?.trim() || humanizeModelID(id), typeLabel: meta.typeLabel, group: meta.group, kind: "entity" as const, artifact };
  });
  const nodes = [...domainNodes, ...entityNodes];
  const edges = edgeArtifacts.map(parseModelEdgeArtifact);
  const groups = DOMAIN_MAP_GROUPS.map((group) => ({ ...group, nodes: nodes.filter((node) => node.group === group.key) })).filter((group) => group.nodes.length > 0);
  const teamCount = entityNodes.filter((node) => node.id.startsWith("team.")).length;
  const serviceCount = entityNodes.filter((node) => node.id.startsWith("svc.")).length;
  const repoCount = entityNodes.filter((node) => node.id.startsWith("repo.")).length;
  const coverageAvailable = Boolean(coverageSummary.trim()) || artifacts.some((artifact) => artifact.path === "reports/coverage/summary.md");
  const blockers: string[] = [];
  if (entityNodes.length === 0) blockers.push("Derived model entities are missing; map is limited to domain agent outputs.");
  if (entityNodes.length > 1 && edges.length === 0) blockers.push("Model entities exist, but no model edge artifacts are available.");
  if (serviceCount > 0 && teamCount === 0) blockers.push("Service nodes are present, but team ownership entities are missing from the artifact list.");
  if (openQuestionCount > 0) blockers.push(`${openQuestionCount} open questions remain linked to evidence review.`);

  return {
    nodes,
    groups,
    edges,
    domainOutputs,
    proposalArtifacts,
    navigationArtifacts: dedupeArtifactNavigation([...domainOutputs, ...entityArtifacts.slice(0, 4), ...edgeArtifacts.slice(0, 3), ...proposalArtifacts.slice(0, 2)]),
    entityCount: entityNodes.length,
    repoCount,
    ownershipStatus: teamCount > 0 ? `${teamCount} team node${teamCount === 1 ? "" : "s"} visible` : serviceCount > 0 ? "partial: service ownership requires team entities or entity content review" : "partial: no service ownership data",
    coverageStatus: coverageAvailable ? "coverage summary linked" : "partial: coverage summary missing",
    crossRepoStatus: repoCount > 1 ? `${repoCount} repo scopes visible` : repoCount === 1 ? "single repo scope visible" : domainOutputs.length > 1 ? "partial: multiple domain outputs, no repo entity artifacts" : "partial: repo scope not visible in model artifacts",
    blockers,
  };
}

function deriveEntityMapMeta(id: string): { typeLabel: string; group: string } {
  if (id.startsWith("svc.")) return { typeLabel: "service", group: "services" };
  if (id.startsWith("api.") || id.startsWith("topic.")) return { typeLabel: id.startsWith("topic.") ? "event topic" : "api", group: "interfaces" };
  if (id.startsWith("db.")) return { typeLabel: "datastore", group: "data" };
  if (id.startsWith("ext.")) return { typeLabel: "external system", group: "external" };
  if (id.startsWith("team.")) return { typeLabel: "team", group: "ownership" };
  if (id.startsWith("repo.")) return { typeLabel: "repo", group: "ownership" };
  return { typeLabel: "entity", group: "other" };
}

function parseModelEdgeArtifact(artifact: Artifact): ReviewDomainMapEdge {
  const id = stripArtifactSuffix(artifact.path.replace(/^model\/edges\//, ""));
  const edgeBody = id.startsWith("edge.") ? id.slice("edge.".length) : id;
  for (const type of MODEL_EDGE_TYPES) {
    const marker = `.${type}.`;
    const index = edgeBody.indexOf(marker);
    if (index > 0) return { id, type, from: edgeBody.slice(0, index), to: edgeBody.slice(index + marker.length), artifact };
  }
  return { id, type: "related", from: "unknown", to: edgeBody || "unknown", artifact };
}

function stripArtifactSuffix(value: string): string {
  return value.replace(/\.(yaml|yml|md|json)$/i, "");
}

function humanizeModelID(id: string): string {
  const normalized = id.replace(/^(svc|team|repo|ext|db|topic|api\.http|api\.grpc)\./, "").replace(/\./g, " ").replace(/-/g, " ").trim();
  return normalized ? normalized.replace(/\b[a-z]/g, (match) => match.toUpperCase()) : id;
}

function dedupeArtifactNavigation(artifacts: Artifact[]): Artifact[] {
  const seen = new Set<string>();
  return artifacts.filter((artifact) => {
    if (seen.has(artifact.path)) return false;
    seen.add(artifact.path);
    return true;
  });
}
