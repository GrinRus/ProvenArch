import { useState } from "react";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TabNav, tabPanelProps } from "./TabNav";

function TabHarness() {
  const [value, setValue] = useState<"one" | "two" | "three">("one");
  return (
    <div>
      <TabNav
        ariaLabel="Example tabs"
        idBase="example-tabs"
        testId="example-tabs"
        value={value}
        onChange={setValue}
        options={[
          { id: "one", label: "One" },
          { id: "two", label: "Two" },
          { id: "three", label: "Three" },
        ]}
      />
      <section {...tabPanelProps("example-tabs", value)}>Panel {value}</section>
    </div>
  );
}

describe("TabNav", () => {
  it("uses roving tabindex and linked tabpanel relationships", () => {
    render(<TabHarness />);

    const tablist = screen.getByTestId("example-tabs");
    const one = within(tablist).getByRole("tab", { name: "One" });
    const two = within(tablist).getByRole("tab", { name: "Two" });
    const three = within(tablist).getByRole("tab", { name: "Three" });
    const panel = screen.getByRole("tabpanel");

    expect(one).toHaveAttribute("id", "example-tabs-tab-one");
    expect(one).toHaveAttribute("aria-selected", "true");
    expect(one).toHaveAttribute("aria-controls", "example-tabs-tabpanel-one");
    expect(one).toHaveAttribute("tabindex", "0");
    expect(two).toHaveAttribute("tabindex", "-1");
    expect(three).toHaveAttribute("tabindex", "-1");
    expect(panel).toHaveAttribute("id", "example-tabs-tabpanel-one");
    expect(panel).toHaveAttribute("aria-labelledby", "example-tabs-tab-one");
  });

  it("supports Arrow, Home and End keyboard selection", () => {
    render(<TabHarness />);

    const tablist = screen.getByTestId("example-tabs");
    const one = within(tablist).getByRole("tab", { name: "One" });
    const two = within(tablist).getByRole("tab", { name: "Two" });
    const three = within(tablist).getByRole("tab", { name: "Three" });

    one.focus();
    fireEvent.keyDown(one, { key: "ArrowRight" });
    expect(two).toHaveFocus();
    expect(two).toHaveAttribute("aria-selected", "true");
    expect(one).toHaveAttribute("tabindex", "-1");
    expect(two).toHaveAttribute("tabindex", "0");
    expect(screen.getByRole("tabpanel")).toHaveAttribute("id", "example-tabs-tabpanel-two");

    fireEvent.keyDown(two, { key: "End" });
    expect(three).toHaveFocus();
    expect(three).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tabpanel")).toHaveAttribute("aria-labelledby", "example-tabs-tab-three");

    fireEvent.keyDown(three, { key: "Home" });
    expect(one).toHaveFocus();
    expect(one).toHaveAttribute("aria-selected", "true");

    fireEvent.keyDown(one, { key: "ArrowLeft" });
    expect(three).toHaveFocus();
    expect(three).toHaveAttribute("aria-selected", "true");
  });
});
