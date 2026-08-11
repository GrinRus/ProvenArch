import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

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
  afterEach(() => vi.unstubAllGlobals());

  it("opens the documents reader as the default architecture mode", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response("# Architecture Home\n\nValidated services", { status: 200 })));
    const onDocumentChange = vi.fn();
    render(<KnowledgePage knowledge={partialKnowledge} loading={false} error="" view="documents" onViewChange={vi.fn()} onEntityChange={vi.fn()} onDocumentChange={onDocumentChange} onOpenArtifact={vi.fn()} />);
    expect(screen.getByTestId("architecture-documents")).toBeInTheDocument();
    expect(screen.getByText("overview.md")).toBeInTheDocument();
    expect(onDocumentChange).toHaveBeenCalledWith("reports/as-is/overview.md");
    expect(await screen.findByTestId("evidence-viewer")).toBeInTheDocument();
  });

  it("renders partial architecture flow without deriving topology from artifact names", () => {
    const onOpenArtifact = vi.fn();
    render(<KnowledgePage knowledge={partialKnowledge} loading={false} error="" view="flows" onViewChange={vi.fn()} onEntityChange={vi.fn()} onOpenArtifact={onOpenArtifact} />);
    expect(screen.getByText("Payments")).toBeInTheDocument();
    expect(screen.getByText("calls")).toBeInTheDocument();
    expect(screen.getByText(/Only model edges with valid endpoints/)).toBeInTheDocument();
    expect(screen.getByText(/Architecture is usable with gaps/)).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole("button", { name: "Evidence" })[0]);
    expect(onOpenArtifact).toHaveBeenCalledWith("model/edges/edge.payments.calls.users.yaml");
  });

  it("provides a searchable keyboard-accessible entity table", () => {
    const onEntityChange = vi.fn();
    render(<KnowledgePage knowledge={partialKnowledge} loading={false} error="" view="catalog" onViewChange={vi.fn()} onEntityChange={onEntityChange} onOpenArtifact={vi.fn()} />);
    fireEvent.change(screen.getByRole("searchbox", { name: "Search" }), { target: { value: "users" } });
    expect(screen.getByRole("button", { name: "Users" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Payments" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Users" }));
    expect(onEntityChange).toHaveBeenCalledWith("svc.users");
  });

  it("shows unavailable without inventing a current workspace graph", () => {
    render(<KnowledgePage knowledge={{ ...partialKnowledge, status: "unavailable", entities: [], edges: [], artifacts: [], issues: [] }} loading={false} error="" view="overview" onViewChange={vi.fn()} onEntityChange={vi.fn()} onOpenArtifact={vi.fn()} />);
    expect(screen.getByText(/No promoted knowledge is available/)).toBeInTheDocument();
  });

  it("shows advisory current-workspace health without presenting it as historical evidence", () => {
    render(
      <KnowledgePage
        knowledge={partialKnowledge}
        workspaceHealth={{ version: 1, generated_at: "2026-07-26T00:00:00Z", status: "warn", summary: { info: 1, warning: 2, error: 0 }, items: [] }}
        loading={false}
        error=""
        view="evidence"
        onViewChange={vi.fn()}
        onEntityChange={vi.fn()}
        onOpenArtifact={vi.fn()}
      />,
    );
    expect(screen.getByText(/Workspace health: warn/)).toHaveTextContent("2 warnings");
  });

  it("keeps Architecture explicitly scoped to the selected Task", () => {
    const onOpenTask = vi.fn();
    render(<KnowledgePage knowledge={partialKnowledge} loading={false} error="" view="overview" taskId="task-opaque-1" onOpenTask={onOpenTask} onViewChange={vi.fn()} onEntityChange={vi.fn()} onOpenArtifact={vi.fn()} />);
    const context = screen.getByTestId("task-architecture-context");
    expect(context).toHaveTextContent("task-opaque-1");
    expect(context).toHaveTextContent("current promoted workspace authority");
    expect(context).toHaveTextContent("does not infer state from the latest run");
    fireEvent.click(within(context).getByRole("button", { name: "Back to Task" }));
    expect(onOpenTask).toHaveBeenCalledWith("task-opaque-1");
  });

  it("filters the map and mobile fallback by canonical owner and domain tags", () => {
    const filteredKnowledge: KnowledgeResponse = {
      ...partialKnowledge,
      entities: [
        { ...partialKnowledge.entities[0], owner_team_id: "team-payments", tags: ["domain:commerce"] },
        { ...partialKnowledge.entities[1], owner_team_id: "team-identity", tags: ["domain:identity"] },
      ],
    };
    render(<KnowledgePage knowledge={filteredKnowledge} loading={false} error="" view="map" onViewChange={vi.fn()} onEntityChange={vi.fn()} onOpenArtifact={vi.fn()} />);
    const mobileList = screen.getByLabelText("Architecture elements");

    fireEvent.change(screen.getByLabelText("Filter by owner"), { target: { value: "team-payments" } });
    expect(within(mobileList).getByRole("button", { name: /Payments/ })).toBeInTheDocument();
    expect(within(mobileList).queryByRole("button", { name: /Users/ })).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Filter by owner"), { target: { value: "" } });
    fireEvent.change(screen.getByLabelText("Filter by domain or tag"), { target: { value: "domain:identity" } });
    expect(within(mobileList).getByRole("button", { name: /Users/ })).toBeInTheDocument();
    expect(within(mobileList).queryByRole("button", { name: /Payments/ })).not.toBeInTheDocument();
  });
});
