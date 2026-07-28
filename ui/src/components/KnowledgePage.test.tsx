import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { KnowledgeResponse } from "../lib/appContracts";
import { KnowledgePage } from "./KnowledgePage";

const partialKnowledge: KnowledgeResponse = {
  version: 1,
  generated_at: "2026-07-15T00:00:00Z",
  source_mode: "promoted_current",
  status: "partial",
  entities: [
    { id: "svc.payments", type: "service", name: "Payments", path: "model/entities/svc.payments.yaml", provenance: { kind: "inference", confidence: 0.9 } },
    { id: "svc.users", type: "service", name: "Users", path: "model/entities/svc.users.yaml", provenance: { kind: "inference", confidence: 0.8 } },
  ],
  edges: [
    { id: "edge.payments.calls.users", type: "calls", from: "svc.payments", to: "svc.users", path: "model/edges/edge.payments.calls.users.yaml", provenance: { kind: "inference", confidence: 0.7 } },
  ],
  artifacts: [{ path: "reports/as-is/overview.md", kind: "report", name: "overview.md" }],
  issues: [{ code: "knowledge.entity_malformed", path: "model/entities/broken.yaml", message: "invalid YAML" }],
};

describe("KnowledgePage", () => {
  it("renders partial validated atlas without deriving topology from artifact names", () => {
    const onOpenArtifact = vi.fn();
    render(<KnowledgePage knowledge={partialKnowledge} loading={false} error="" view="atlas" onViewChange={vi.fn()} onEntityChange={vi.fn()} onOpenArtifact={onOpenArtifact} />);
    expect(screen.getByText("Payments (svc.payments)")).toBeInTheDocument();
    expect(screen.getByText("calls")).toBeInTheDocument();
    expect(screen.getByText(/File names are never interpreted/)).toBeInTheDocument();
    expect(screen.getByText(/Atlas is incomplete/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "model/edges/edge.payments.calls.users.yaml" }));
    expect(onOpenArtifact).toHaveBeenCalledWith("model/edges/edge.payments.calls.users.yaml");
  });

  it("provides a searchable keyboard-accessible entity table", () => {
    const onEntityChange = vi.fn();
    render(<KnowledgePage knowledge={partialKnowledge} loading={false} error="" view="entities" onViewChange={vi.fn()} onEntityChange={onEntityChange} onOpenArtifact={vi.fn()} />);
    fireEvent.change(screen.getByRole("searchbox", { name: "Search entities" }), { target: { value: "users" } });
    expect(screen.getByRole("button", { name: "Users" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Payments" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Users" }));
    expect(onEntityChange).toHaveBeenCalledWith("svc.users");
  });

  it("shows unavailable without inventing a current workspace graph", () => {
    render(<KnowledgePage knowledge={{ ...partialKnowledge, status: "unavailable", entities: [], edges: [], artifacts: [], issues: [] }} loading={false} error="" view="overview" onViewChange={vi.fn()} onEntityChange={vi.fn()} onOpenArtifact={vi.fn()} />);
    expect(screen.getByText(/No promoted knowledge is available/)).toBeInTheDocument();
    expect(screen.getAllByText("0", { selector: "strong" })).toHaveLength(4);
  });

  it("shows advisory current-workspace health without presenting it as historical evidence", () => {
    render(
      <KnowledgePage
        knowledge={partialKnowledge}
        workspaceHealth={{ version: 1, generated_at: "2026-07-26T00:00:00Z", status: "warn", summary: { info: 1, warning: 2, error: 0 }, items: [] }}
        loading={false}
        error=""
        view="overview"
        onViewChange={vi.fn()}
        onEntityChange={vi.fn()}
        onOpenArtifact={vi.fn()}
      />,
    );
    expect(screen.getByTestId("knowledge-workspace-health")).toHaveTextContent("Workspace Health: warn");
    expect(screen.getByTestId("knowledge-workspace-health")).toHaveTextContent("Advisory only");
  });
});
