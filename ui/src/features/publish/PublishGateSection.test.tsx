import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PublishGateSection } from "./PublishGateSection";

describe("PublishGateSection", () => {
  it("renders the empty gate state", () => {
    render(<PublishGateSection testId="publish-gate" title="Hard blockers" emptyLabel="No blockers" items={[]} />);
    expect(screen.getByTestId("publish-gate")).toHaveTextContent("No blockers");
  });

  it("renders gate item tone and detail", () => {
    render(<PublishGateSection testId="publish-gate" title="Ready checks" emptyLabel="none" items={[{ label: "Diff loaded", detail: "Full workspace scope", tone: "ok" }]} />);
    expect(screen.getByTestId("publish-gate")).toHaveTextContent("Diff loaded");
    expect(screen.getByTestId("publish-gate")).toHaveTextContent("Full workspace scope");
  });
});
