import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ContextDrawer } from "./ContextDrawer";

afterEach(() => vi.unstubAllGlobals());

describe("ContextDrawer", () => {
  it("uses a focus-managed modal below the wide breakpoint", () => {
    vi.stubGlobal("matchMedia", () => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    const onClose = vi.fn();
    render(<><button>Return target</button><ContextDrawer open title="Context" description="Details" onClose={onClose}><button>Drawer action</button></ContextDrawer></>);
    expect(screen.getByRole("dialog", { name: "Context" })).toHaveAttribute("aria-modal", "true");
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("uses a non-modal complementary landmark on wide desktop", () => {
    vi.stubGlobal("matchMedia", () => ({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    render(<ContextDrawer open title="Context" description="Details" onClose={vi.fn()}>Body</ContextDrawer>);
    expect(screen.getByRole("complementary", { name: "Context" })).not.toHaveAttribute("aria-modal");
  });

  it("switches an open drawer to a focus-managed sheet after an orientation breakpoint change", () => {
    let listener: (() => void) | undefined;
    const media = {
      matches: true,
      addEventListener: vi.fn((_event: string, next: () => void) => { listener = next; }),
      removeEventListener: vi.fn(),
    };
    vi.stubGlobal("matchMedia", () => media);
    render(<ContextDrawer open title="Context" description="Details" onClose={vi.fn()}><button>Drawer action</button></ContextDrawer>);
    expect(screen.getByRole("complementary", { name: "Context" })).toBeInTheDocument();
    act(() => {
      media.matches = false;
      listener?.();
    });
    expect(screen.getByRole("dialog", { name: "Context" })).toHaveAttribute("aria-modal", "true");
    expect(screen.getByRole("button", { name: "Cancel" })).toHaveFocus();
  });
});
