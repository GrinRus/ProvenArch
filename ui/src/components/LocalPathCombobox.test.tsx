import { useState } from "react";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LocalPathCombobox } from "./LocalPathCombobox";
import { loadOnboardingPathSuggestions } from "../lib/onboardingApi";
import type { OnboardingPathSuggestion } from "../lib/appContracts";

vi.mock("../lib/onboardingApi", () => ({
  loadOnboardingPathSuggestions: vi.fn(),
}));

const suggestions: OnboardingPathSuggestion[] = [
  {
    path: "/work/acp",
    label: "ACP workspace",
    exists: true,
    kind: "directory",
    source: "recent",
  },
  {
    path: "/work/provenarch",
    label: "ProvenArch",
    exists: true,
    kind: "git_repo",
    source: "parent",
  },
];

function PathComboboxHarness({ onSelect = vi.fn() }: { onSelect?: (suggestion: OnboardingPathSuggestion) => void }) {
  const [value, setValue] = useState("");
  return (
    <LocalPathCombobox
      id="workspace-path"
      kind="workspace"
      label="Workspace path"
      value={value}
      onChange={setValue}
      onSelect={onSelect}
      testID="workspace-path-combobox"
    />
  );
}

async function openSuggestions() {
  const input = screen.getByRole("combobox", { name: "Workspace path" });
  fireEvent.focus(input);
  await screen.findByRole("option", { name: /ACP workspace/ });
  return input;
}

describe("LocalPathCombobox", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadOnboardingPathSuggestions).mockResolvedValue({
      ok: true,
      kind: "workspace",
      query: "",
      items: suggestions,
    });
  });

  it("supports keyboard-only suggestion selection with aria-activedescendant", async () => {
    const onSelect = vi.fn();
    render(<PathComboboxHarness onSelect={onSelect} />);

    const input = await openSuggestions();
    fireEvent.keyDown(input, { key: "ArrowDown" });
    expect(input).toHaveAttribute("aria-activedescendant", "workspace-path-suggestions-option-0");
    expect(screen.getByRole("option", { name: /ACP workspace/ })).toHaveAttribute("aria-selected", "true");

    fireEvent.keyDown(input, { key: "Enter" });
    expect(input).toHaveValue("/work/acp");
    expect(onSelect).toHaveBeenCalledWith(suggestions[0]);
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    expect(input).not.toHaveAttribute("aria-activedescendant");
  });

  it("wraps active descendant with ArrowUp and ArrowDown", async () => {
    render(<PathComboboxHarness />);

    const input = await openSuggestions();
    fireEvent.keyDown(input, { key: "ArrowUp" });
    expect(input).toHaveAttribute("aria-activedescendant", "workspace-path-suggestions-option-1");
    expect(screen.getByRole("option", { name: /ProvenArch/ })).toHaveAttribute("aria-selected", "true");

    fireEvent.keyDown(input, { key: "ArrowDown" });
    expect(input).toHaveAttribute("aria-activedescendant", "workspace-path-suggestions-option-0");
  });

  it("closes on Escape without selecting", async () => {
    const onSelect = vi.fn();
    render(<PathComboboxHarness onSelect={onSelect} />);

    const input = await openSuggestions();
    fireEvent.keyDown(input, { key: "ArrowDown" });
    fireEvent.keyDown(input, { key: "Escape" });

    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    expect(input).toHaveValue("");
    expect(input).not.toHaveAttribute("aria-activedescendant");
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("keeps pointer selection parity with keyboard selection", async () => {
    const onSelect = vi.fn();
    render(<PathComboboxHarness onSelect={onSelect} />);

    await openSuggestions();
    const listbox = screen.getByRole("listbox", { name: "Workspace path suggestions" });
    fireEvent.click(within(listbox).getByRole("option", { name: /ProvenArch/ }));

    expect(screen.getByRole("combobox", { name: "Workspace path" })).toHaveValue("/work/provenarch");
    expect(onSelect).toHaveBeenCalledWith(suggestions[1]);
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  it("announces suggestion failures and links helper text to the input", async () => {
    vi.mocked(loadOnboardingPathSuggestions).mockRejectedValueOnce(new Error("unavailable"));
    render(<PathComboboxHarness />);

    const input = screen.getByRole("combobox", { name: "Workspace path" });
    fireEvent.focus(input);

    await screen.findByText("Suggestions unavailable. Typed path still works.");
    const alert = screen.getByRole("alert");
    expect(alert).toHaveAttribute("id", "workspace-path-helper");
    expect(input).toHaveAttribute("aria-describedby", "workspace-path-helper");
  });
});
