import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AsyncState, Button, DefinitionList, MetricGrid, PageHeader, RecoveryPanel, RouteTabs } from "./SemanticPrimitives";

describe("semantic primitives", () => {
  it("renders page hierarchy and component states", () => {
    render(<><PageHeader title="Changes" purpose="Review evidence" source="Run snapshot" state="Needs review" action={<Button tone="primary">Continue</Button>} /><RecoveryPanel title="Partial" tone="warning">Keep valid evidence visible.</RecoveryPanel><MetricGrid items={[{ label: "Files", value: 2 }]} /><DefinitionList density="compact" items={[{ label: "Run", value: "run-1" }]} /><AsyncState state="error">Unavailable</AsyncState></>);
    expect(screen.getByRole("heading", { name: "Changes" })).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("Unavailable");
    expect(screen.getByRole("button", { name: "Continue" })).toHaveClass("tone-primary");
  });

  it("uses route semantics without pretending navigation is an ARIA tablist", () => {
    const onChange = vi.fn();
    render(<RouteTabs label="Views" value="one" items={[{ id: "one", label: "One" }, { id: "two", label: "Two" }]} onChange={onChange} />);
    expect(screen.getByRole("button", { name: "One" })).toHaveAttribute("aria-current", "page");
    fireEvent.click(screen.getByRole("button", { name: "Two" }));
    expect(onChange).toHaveBeenCalledWith("two");
  });
});
