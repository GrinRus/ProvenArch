import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ModalDialog } from "./ModalDialog";

describe("ModalDialog", () => {
  it("sets dialog semantics, initial focus, traps focus, and closes on Escape", () => {
    const onCancel = vi.fn();
    render(<ModalDialog open title="Confirm" description="Review inventory" confirmLabel="Commit" onConfirm={vi.fn()} onCancel={onCancel} />);

    const dialog = screen.getByRole("dialog");
    const cancel = screen.getByRole("button", { name: "Cancel" });
    const confirm = screen.getByRole("button", { name: "Commit" });
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(cancel).toHaveFocus();

    confirm.focus();
    fireEvent.keyDown(dialog, { key: "Tab" });
    expect(cancel).toHaveFocus();
    fireEvent.keyDown(dialog, { key: "Escape" });
    expect(onCancel).toHaveBeenCalledOnce();
  });
});
