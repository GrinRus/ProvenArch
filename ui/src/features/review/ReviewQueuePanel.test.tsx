import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ReviewQueuePanel } from "./ReviewQueuePanel";

describe("ReviewQueuePanel", () => {
  it("renders the empty review queue state", () => {
    render(<ReviewQueuePanel queue={[]} selectedArtifact="" onOpenArtifact={() => undefined} />);
    expect(screen.getByTestId("review-queue")).toHaveTextContent("No generated review items");
  });

  it("renders selected queue item and keeps its action", () => {
    const onOpenArtifact = () => undefined;
    render(<ReviewQueuePanel queue={[{ id: "question:one", kind: "question", severity: "warn", title: "Open question", path: "reports/questions.md" }]} selectedArtifact="reports/questions.md" onOpenArtifact={onOpenArtifact} />);
    expect(screen.getByRole("button", { name: "Review queue item: Open question" })).toHaveAttribute("aria-current", "true");
  });
});
