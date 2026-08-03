import type { ArchitectureEdge, ArchitectureNode } from "../../lib/appContracts";

export function boundedLargeArchitecture(size = 80): { nodes: ArchitectureNode[]; edges: ArchitectureEdge[] } {
  const bounded = Math.max(2, Math.min(size, 120));
  const nodes = Array.from({ length: bounded }, (_, index): ArchitectureNode => ({ id: `svc.fixture-${String(index).padStart(3, "0")}`, name: `Fixture service ${index}`, type: "service", confidence: 0.8 + (index % 20) / 100, provenance_kind: "inference", path: `model/entities/svc.fixture-${String(index).padStart(3, "0")}.yaml`, repositories: [`repo-${index % 8}`], available_levels: ["context", "container"] }));
  const edges = nodes.slice(1).map((node, index): ArchitectureEdge => ({ id: `edge.fixture-${String(index).padStart(3, "0")}`, from: nodes[index].id, to: node.id, type: "calls", confidence: 0.8, provenance_kind: "inference", path: `model/edges/edge.fixture-${String(index).padStart(3, "0")}.yaml` }));
  return { nodes, edges };
}
