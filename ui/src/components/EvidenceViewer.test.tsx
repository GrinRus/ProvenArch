import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { EvidenceViewer } from "./EvidenceViewer";

vi.mock("./MermaidPreview", () => ({ MermaidPreview: ({ source }: { source: string }) => <div data-testid="mermaid">{source}</div> }));

describe("EvidenceViewer", () => {
  beforeEach(() => {
    window.history.replaceState({}, "", "/changes?mode=rendered");
  });

  it("renders GFM safely, routes local links, and exposes raw evidence", () => {
    const onOpenArtifact = vi.fn();
    render(
      <EvidenceViewer
        path="reports/overview.md"
        runId="run-1"
        sourceMode="run_snapshot"
        demo
        content={"# Overview\n\n|A|B|\n|-|-|\n|1|2|\n\n[local](./detail.md) [external](https://example.com)\n\n<script>alert(1)</script>\n\n```mermaid\ngraph TD; A-->B\n```"}
        onOpenArtifact={onOpenArtifact}
      />,
    );

    expect(screen.getByText("Demo evidence")).toBeInTheDocument();
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(document.querySelector("script")).toBeNull();
    expect(screen.getByTestId("mermaid")).toHaveTextContent("graph TD");
    fireEvent.click(screen.getByRole("button", { name: "local" }));
    expect(onOpenArtifact).toHaveBeenCalledWith("reports/detail.md");
    fireEvent.click(screen.getByRole("tab", { name: "Raw" }));
    expect(screen.getByTestId("evidence-raw")).toHaveTextContent("<script>alert(1)</script>");
  });

  it("blocks authority traversal, preserves source identity, and labels unknown provenance truthfully", () => {
    const onOpenArtifact = vi.fn();
    render(
      <EvidenceViewer
        path="reports/as-is/overview.md"
        runId="run-selected"
        sourceMode="run_snapshot"
        content={"[sibling](./details.md) [escape](../../../reports/taskruns/other/secret.md)"}
        status="partial"
        issues={[{ code: "snapshot_document_unavailable", message: "missing selected bytes" }]}
        onOpenArtifact={onOpenArtifact}
      />,
    );
    expect(screen.getByText("Run snapshot · run-selected")).toBeInTheDocument();
    expect(screen.getByText("Unknown evidence")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("partial");
    expect(screen.getByLabelText("Evidence issues")).toHaveTextContent("snapshot_document_unavailable");
    fireEvent.click(screen.getByRole("button", { name: "sibling" }));
    expect(onOpenArtifact).toHaveBeenCalledWith("reports/as-is/details.md");
    expect(screen.queryByRole("button", { name: "escape" })).not.toBeInTheDocument();
  });

  it("names both sources for an explicit diff", () => {
    render(
      <EvidenceViewer
        path="reports/overview.md"
        sourceMode="promoted_current"
        content="current"
        diff={<pre>changed</pre>}
        diffIdentity={{ left: "run_snapshot:run-1", right: "promoted_current" }}
      />,
    );
    expect(screen.getByTestId("evidence-diff-identity")).toHaveTextContent("run_snapshot:run-1 → promoted_current");
  });

  it("uses a readable raw fallback when rendered content exceeds its budget", () => {
    render(
      <EvidenceViewer
        path="reports/large.md"
        sourceMode="promoted_current"
        content={"x".repeat(512 * 1024 + 1)}
      />,
    );
    expect(screen.getByText(/Rendered preview is disabled/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Raw" }));
    expect(screen.getByTestId("evidence-raw").textContent).toHaveLength(512 * 1024 + 1);
  });
});
