import { useEffect, useMemo, useState, type ReactNode } from "react";
import ELK from "elkjs/lib/elk.bundled.js";
import {
  Background,
  Controls,
  Handle,
  MiniMap,
  Position,
  ReactFlow,
  type Edge as FlowEdge,
  type Node as FlowNode,
  type NodeProps,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import type { ArchitectureEdge, ArchitectureLevel, ArchitectureNode, ArchitectureView } from "../lib/appContracts";

const elk = new ELK();
const nodeTypes = { architecture: ArchitectureNodeControl };

function ArchitectureNodeControl({ id, data }: NodeProps) {
  const typed = data as { label: ReactNode; onSelect?: (id: string) => void };
  return <><Handle type="target" position={Position.Left} isConnectable={false} /><button type="button" className="architecture-node-button" onClick={() => typed.onSelect?.(id)}>{typed.label}</button><Handle type="source" position={Position.Right} isConnectable={false} /></>;
}

export function ArchitectureMap({
  view,
  level,
  selectedID,
  query,
  typeFilter,
  repositoryFilter,
  ownerFilter,
  tagFilter,
  onSelect,
}: {
  view: ArchitectureView;
  level: ArchitectureLevel;
  selectedID?: string;
  query: string;
  typeFilter?: string;
  repositoryFilter?: string;
  ownerFilter?: string;
  tagFilter?: string;
  onSelect: (id?: string) => void;
}) {
  const [layout, setLayout] = useState<{ nodes: FlowNode[]; edges: FlowEdge[] }>({ nodes: [], edges: [] });
  const [layoutError, setLayoutError] = useState("");
  const normalizedQuery = query.trim().toLowerCase();
  const visible = useMemo(() => {
    const candidates = view.nodes.filter((node) =>
      (!typeFilter || node.type === typeFilter) &&
      (!repositoryFilter || node.repositories?.includes(repositoryFilter)) &&
      (!ownerFilter || node.owner_team_id === ownerFilter) &&
      (!tagFilter || node.tags?.includes(tagFilter)),
    );
    if (!normalizedQuery) {
      const included = new Set(candidates.map((node) => node.id));
      return { ...view, nodes: candidates, edges: view.edges.filter((edge) => included.has(edge.from) && included.has(edge.to)) };
    }
    const matching = new Set(candidates.filter((node) => [node.name, node.id, node.type, node.owner_team_id ?? "", ...(node.tags ?? [])].join(" ").toLowerCase().includes(normalizedQuery)).map((node) => node.id));
    for (const edge of view.edges) if (matching.has(edge.from) || matching.has(edge.to)) { matching.add(edge.from); matching.add(edge.to); }
    return { ...view, nodes: view.nodes.filter((node) => matching.has(node.id)), edges: view.edges.filter((edge) => matching.has(edge.from) && matching.has(edge.to)) };
  }, [normalizedQuery, ownerFilter, repositoryFilter, tagFilter, typeFilter, view]);

  useEffect(() => {
    let active = true;
    setLayoutError("");
    void layoutArchitecture(visible.nodes, visible.edges).then((next) => { if (active) setLayout(next); }).catch((error: unknown) => { if (active) setLayoutError(error instanceof Error ? error.message : "Architecture layout failed"); });
    return () => { active = false; };
  }, [visible]);

  if (!view.available) return <div className="empty-state recovery-empty"><strong>{levelLabel(level)} is not available.</strong><span>{view.unavailable_reason || "The promoted model has no validated entities for this level."}</span></div>;
  if (visible.nodes.length === 0) return <div className="empty-state"><strong>No architecture elements match this search.</strong><span>Clear the search or choose another C4 level.</span></div>;
  if (layoutError) return <p className="status err" role="status">{layoutError}</p>;

  return (
    <div className="architecture-canvas" data-testid="architecture-canvas" aria-label={`${levelLabel(level)} architecture map`}>
      <ReactFlow
        nodes={layout.nodes.map((node) => ({ ...node, selected: node.id === selectedID, data: { ...node.data, onSelect } }))}
        edges={layout.edges.map((edge) => ({ ...edge, selected: edge.id === selectedID }))}
        fitView
        fitViewOptions={{ padding: 0.18, maxZoom: 1.15 }}
        minZoom={0.2}
        maxZoom={1.8}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable
        nodesFocusable={false}
        edgesFocusable
        autoPanOnNodeFocus={false}
        nodeTypes={nodeTypes}
        onNodeClick={(_, node) => onSelect(node.id)}
        onEdgeClick={(_, edge) => onSelect(edge.id)}
        onSelectionChange={({ nodes, edges }) => {
          const selected = nodes[nodes.length - 1] ?? edges[edges.length - 1];
          if (selected && selected.id !== selectedID) onSelect(selected.id);
        }}
        onPaneClick={() => onSelect(undefined)}
        proOptions={{ hideAttribution: true }}
      >
        <Background gap={24} size={1} color="var(--color-border-subtle)" />
        <MiniMap pannable zoomable ariaLabel="Architecture overview" nodeColor="var(--color-accent-muted)" />
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  );
}

export async function layoutArchitecture(nodes: ArchitectureNode[], edges: ArchitectureEdge[]): Promise<{ nodes: FlowNode[]; edges: FlowEdge[] }> {
  const graph = await elk.layout({
    id: "architecture",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": "RIGHT",
      "elk.spacing.nodeNode": "48",
      "elk.layered.spacing.nodeNodeBetweenLayers": "88",
      "elk.edgeRouting": "ORTHOGONAL",
    },
    children: nodes.map((node) => ({ id: node.id, width: 224, height: 84 })),
    edges: edges.map((edge) => ({ id: edge.id, sources: [edge.from], targets: [edge.to] })),
  });
  const byID = new Map(nodes.map((node) => [node.id, node]));
  return {
    nodes: (graph.children ?? []).map((item) => {
      const node = byID.get(item.id)!;
      return {
        id: item.id,
        position: { x: item.x ?? 0, y: item.y ?? 0 },
        data: { label: <div className="architecture-node-label"><span>{node.type}</span><strong>{node.name}</strong><small>{Math.round(node.confidence * 100)}% confidence{node.owner_team_id ? ` · ${node.owner_team_id}` : ""}</small></div> },
        className: `architecture-node type-${node.type.replace(/\./g, "-")}`,
        type: "architecture",
        ariaLabel: `${node.name}, ${node.type}, ${Math.round(node.confidence * 100)} percent confidence`,
        style: { width: 224, minHeight: 84 },
      } satisfies FlowNode;
    }),
    edges: edges.map((edge) => ({ id: edge.id, source: edge.from, target: edge.to, label: edge.name || edge.type, type: "smoothstep", animated: false, ariaLabel: `${edge.type} from ${edge.from} to ${edge.to}`, style: { strokeWidth: 1.5 } })),
  };
}

export function levelLabel(level: ArchitectureLevel): string {
  return level === "context" ? "System context" : level.charAt(0).toUpperCase() + level.slice(1);
}
