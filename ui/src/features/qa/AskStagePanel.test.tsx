import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("../../lib/qaApi", () => ({
  createQAProposalDraft: vi.fn(),
  getQARun: vi.fn(),
  listQARuns: vi.fn().mockResolvedValue([]),
  startQAQuestion: vi.fn(),
}));

import { AskStagePanel } from "./AskStagePanel";

describe("AskStagePanel", () => {
  it("keeps the Ask surface available when history is empty", async () => {
    render(<AskStagePanel onOpenArtifact={vi.fn()} />);
    expect(screen.getByTestId("qa-panel")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Ask" })).toBeInTheDocument();
    expect(screen.getByTestId("qa-authority-boundary")).toHaveTextContent("Current workspace · read-only");
    expect(screen.getByTestId("qa-authority-boundary")).toHaveTextContent("Provider outage blocks a new Ask run");
    expect(await screen.findByText("Ask the workspace to create the first read-only Q&A run.")).toBeInTheDocument();
  });
});
