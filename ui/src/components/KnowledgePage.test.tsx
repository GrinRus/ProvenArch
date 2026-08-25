import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useState } from "react";

import type { KnowledgeResponse } from "../lib/appContracts";
import { architectureFromKnowledge } from "../lib/workspaceApi";
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
    expect(screen.getByRole("button", { name: /reports\/as-is\/overview\.md/ })).toBeInTheDocument();
    expect(onDocumentChange).toHaveBeenCalledWith("reports/as-is/overview.md");
    expect(await screen.findByTestId("evidence-viewer")).toBeInTheDocument();
  });

  it("prefers the authoritative Architecture Home over an alphabetically first proposal", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response("# Architecture Home", { status: 200 })));
    const onDocumentChange = vi.fn();
    const knowledge = { ...partialKnowledge, artifacts: [
      { path: "proposals/adr-001.md", kind: "proposal", name: "ADR 001" },
      { path: "reports/as-is/overview.md", kind: "report", name: "Architecture Home" },
    ] };
    render(<KnowledgePage knowledge={knowledge} loading={false} error="" view="documents" onViewChange={vi.fn()} onEntityChange={vi.fn()} onDocumentChange={onDocumentChange} onOpenArtifact={vi.fn()} />);
    expect(await screen.findByTestId("evidence-viewer")).toBeInTheDocument();
    expect(onDocumentChange).toHaveBeenCalledWith("reports/as-is/overview.md");
    expect(screen.getByRole("button", { name: /Architecture Home/ })).toHaveAttribute("aria-current", "page");
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

  it("edits only allowlisted workspace Markdown and keeps promoted reports read-only", async () => {
    const editableKnowledge: KnowledgeResponse = { ...partialKnowledge, artifacts: [...partialKnowledge.artifacts, { path: "charter/readme.md", kind: "document", name: "readme.md" }] };
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => init?.method === "POST"
      ? new Response(JSON.stringify({ ok: true }), { status: 200, headers: { "Content-Type": "application/json" } })
      : new Response("# Editable charter\n\nKeep this exact text", { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    function Harness() {
      const [selectedPath, setSelectedPath] = useState<string>();
      return <KnowledgePage knowledge={editableKnowledge} loading={false} error="" view="documents" selectedArtifactPath={selectedPath} onViewChange={vi.fn()} onEntityChange={vi.fn()} onDocumentChange={setSelectedPath} onOpenArtifact={vi.fn()} />;
    }
    render(<Harness />);
    await screen.findByTestId("evidence-viewer");
    fireEvent.click(screen.getByRole("button", { name: /charter\/readme\.md/ }));
    await screen.findByText("Editable charter");
    fireEvent.click(screen.getByRole("button", { name: "Edit Markdown" }));
    const editor = await screen.findByTestId("markdown-editor");
    fireEvent.change(editor, { target: { value: "# Updated charter" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByText("Saved to the editable workspace surface.")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith("/api/artifacts/write", expect.objectContaining({ method: "POST" }));
    fireEvent.click(screen.getByRole("button", { name: /reports\/as-is\/overview\.md/ }));
    expect(await screen.findByText(/Promoted Architecture documents are read-only evidence/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit Markdown" })).not.toBeInTheDocument();
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

  it("keeps the structured model inspector read-only and exposes source diagnostics", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response("id: svc.payments\ntype: service\n", { status: 200 })));
    const onOpenArtifact = vi.fn();
    function Harness() {
      const [selectedID, setSelectedID] = useState<string>();
      return <KnowledgePage knowledge={partialKnowledge} loading={false} error="" view="map" selectedEntityID={selectedID} onViewChange={vi.fn()} onEntityChange={setSelectedID} onOpenArtifact={onOpenArtifact} />;
    }
    render(<Harness />);
    fireEvent.click(within(screen.getByLabelText("Architecture elements")).getByRole("button", { name: /Payments/ }));
    const inspector = await screen.findByTestId("structured-model-inspector");
    expect(inspector).toHaveTextContent("architecture.entity v1");
    expect(inspector).toHaveTextContent("Structured editing is unavailable until comments");
    expect(inspector).not.toHaveTextContent("Save");
    fireEvent.click(within(inspector).getByRole("button", { name: "Source (Advanced)" }));
    expect(await within(inspector).findByText(/0001 \| id: svc\.payments/)).toBeInTheDocument();
    fireEvent.click(within(inspector).getByRole("button", { name: "Open source artifact" }));
    expect(onOpenArtifact).toHaveBeenCalledWith("model/entities/svc.payments.yaml");
  });

  it("keeps Mermaid layout non-authoritative and exposes raw source plus relation navigation", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response("flowchart TD\n  Payments --> Users\n", { status: 200 })));
    const onOpenArtifact = vi.fn();
    const diagramKnowledge: KnowledgeResponse = { ...partialKnowledge, artifacts: [...partialKnowledge.artifacts, { path: "reports/diagrams/context.mmd", kind: "report", name: "context.mmd" }] };
    render(<KnowledgePage knowledge={diagramKnowledge} loading={false} error="" view="diagrams" onViewChange={vi.fn()} onEntityChange={vi.fn()} onOpenArtifact={onOpenArtifact} />);
    const studio = await screen.findByTestId("mermaid-evidence-context");
    expect(studio).toHaveTextContent("Mermaid layout and arrows are visual aids only");
    expect(studio).toHaveTextContent("Open relation evidence");
    fireEvent.click(screen.getByRole("tab", { name: "Raw" }));
    expect(await screen.findByTestId("evidence-raw")).toHaveTextContent("flowchart TD");
    fireEvent.click(within(studio).getByRole("button", { name: "Open relation evidence" }));
    expect(onOpenArtifact).toHaveBeenCalledWith("model/edges/edge.payments.calls.users.yaml");
  }, 15000);

  it("keeps findings and questions actionable without inventing approval state", () => {
    const reviewKnowledge = {
      ...partialKnowledge,
      review: {
        findings: [{ id: "finding-1", severity: "high", title: "Missing owner", description: "The service has no canonical owner.", related_ids: ["svc.payments"] }, { id: "finding-2", severity: "low", title: "Stale tag", description: "Tag needs review." }],
        questions: [{ id: "question-1", text: "Which team owns this boundary?", priority: "high", related_ids: ["finding-1"] }],
      },
      coverage: { missing: ["service ownership"] },
    };
    const architecture = { ...architectureFromKnowledge(reviewKnowledge), review: reviewKnowledge.review, coverage: reviewKnowledge.coverage };
    render(<KnowledgePage architecture={architecture} knowledge={reviewKnowledge} loading={false} error="" view="findings" onViewChange={vi.fn()} onEntityChange={vi.fn()} onOpenArtifact={vi.fn()} />);
    expect(screen.getByTestId("proposal-decision-boundary")).toHaveTextContent(/Approved status.*unavailable/);
    fireEvent.change(screen.getByRole("searchbox", { name: "Search" }), { target: { value: "owner" } });
    expect(screen.getByRole("button", { name: /Missing owner/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Stale tag/ })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Missing owner/ }));
    expect(screen.getByText("The service has no canonical owner.")).toBeInTheDocument();
    expect(screen.getByText(/0 evidence refs/)).toBeInTheDocument();
  });

  it("opens repository evidence through the source authority instead of workspace artifacts", async () => {
    const reviewKnowledge = {
      ...partialKnowledge,
      entities: [{ ...partialKnowledge.entities[0], provenance: { kind: "observation" as const, confidence: 0.9, evidence: [{ repo: "payments-service", path: "README.md" }] } }, partialKnowledge.entities[1]],
      review: { findings: [{ id: "finding-1", severity: "high", title: "Missing owner", description: "The service has no canonical owner.", related_ids: ["svc.payments"] }], questions: [] },
      coverage: { missing: [] },
    };
    const architecture = { ...architectureFromKnowledge(reviewKnowledge), review: reviewKnowledge.review, coverage: reviewKnowledge.coverage };
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => String(input).includes("/api/repository-evidence")
      ? new Response(JSON.stringify({ repo: "payments-service", path: "README.md", content: "# Payments\n" }), { status: 200, headers: { "Content-Type": "application/json" } })
      : new Response("", { status: 200 })));
    render(<KnowledgePage architecture={architecture} knowledge={reviewKnowledge} loading={false} error="" view="findings" onViewChange={vi.fn()} onEntityChange={vi.fn()} onOpenArtifact={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: /Missing owner/ }));
    fireEvent.click(screen.getByRole("button", { name: "Open source" }));
    expect(await screen.findByTestId("repository-evidence-viewer")).toHaveTextContent("payments-service");
    expect(screen.getByTestId("repository-evidence-viewer")).toHaveTextContent("# Payments");
  });
});
