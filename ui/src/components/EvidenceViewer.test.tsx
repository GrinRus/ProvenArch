import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { EvidenceViewer } from "./EvidenceViewer";

vi.mock("./MermaidPreview", () => ({ MermaidPreview: ({ source }: { source: string }) => <div data-testid="mermaid">{source}</div> }));

describe("EvidenceViewer", () => {
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
    expect(onOpenArtifact).toHaveBeenCalledWith("detail.md");
    fireEvent.click(screen.getByRole("tab", { name: "Raw" }));
    expect(screen.getByTestId("evidence-raw")).toHaveTextContent("<script>alert(1)</script>");
  });
});
