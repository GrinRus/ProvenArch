import { act, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { usePollingLoop } from "./usePollingLoop";

function PollHarness({ enabled, poll, intervalMs = 1000, maxIntervalMs = 8000 }: { enabled: boolean; poll: (signal: AbortSignal) => Promise<boolean | void>; intervalMs?: number; maxIntervalMs?: number }) {
  usePollingLoop({ enabled, poll, intervalMs, maxIntervalMs });
  return null;
}

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("usePollingLoop", () => {
  it("runs sequentially and backs off failures with a bounded delay", async () => {
    vi.useFakeTimers();
    const poll = vi.fn<(signal: AbortSignal) => Promise<boolean>>();
    poll.mockResolvedValueOnce(false).mockResolvedValue(true);
    render(<PollHarness enabled poll={poll} intervalMs={1000} maxIntervalMs={2500} />);

    await act(() => vi.advanceTimersByTimeAsync(1000));
    expect(poll).toHaveBeenCalledTimes(1);
    await act(() => vi.advanceTimersByTimeAsync(1999));
    expect(poll).toHaveBeenCalledTimes(1);
    await act(() => vi.advanceTimersByTimeAsync(1));
    expect(poll).toHaveBeenCalledTimes(2);
    await act(() => vi.advanceTimersByTimeAsync(1000));
    expect(poll).toHaveBeenCalledTimes(3);
  });

  it("does not overlap a pending request", async () => {
    vi.useFakeTimers();
    let resolve!: (value: boolean) => void;
    const pending = new Promise<boolean>((nextResolve) => { resolve = nextResolve; });
    const poll = vi.fn<(signal: AbortSignal) => Promise<boolean>>().mockReturnValue(pending);
    render(<PollHarness enabled poll={poll} />);

    await act(() => vi.advanceTimersByTimeAsync(1000));
    await act(() => vi.advanceTimersByTimeAsync(10_000));
    expect(poll).toHaveBeenCalledTimes(1);
    resolve(true);
    await act(() => vi.advanceTimersByTimeAsync(999));
    expect(poll).toHaveBeenCalledTimes(1);
    await act(() => vi.advanceTimersByTimeAsync(1));
    expect(poll).toHaveBeenCalledTimes(2);
  });

  it("aborts a pending request when the lifecycle is disabled", async () => {
    vi.useFakeTimers();
    let activeSignal: AbortSignal | undefined;
    const poll = vi.fn<(signal: AbortSignal) => Promise<boolean>>((signal) => {
      activeSignal = signal;
      return new Promise(() => undefined);
    });
    const view = render(<PollHarness enabled poll={poll} />);
    await act(() => vi.advanceTimersByTimeAsync(1000));
    expect(activeSignal?.aborted).toBe(false);
    view.unmount();
    expect(activeSignal?.aborted).toBe(true);
  });

  it("pauses while hidden or offline and resumes on the relevant event", async () => {
    vi.useFakeTimers();
    Object.defineProperty(document, "visibilityState", { configurable: true, value: "hidden" });
    Object.defineProperty(navigator, "onLine", { configurable: true, value: true });
    const poll = vi.fn<(signal: AbortSignal) => Promise<boolean>>().mockResolvedValue(true);
    render(<PollHarness enabled poll={poll} />);
    await act(() => vi.advanceTimersByTimeAsync(2000));
    expect(poll).not.toHaveBeenCalled();

    Object.defineProperty(document, "visibilityState", { configurable: true, value: "visible" });
    document.dispatchEvent(new Event("visibilitychange"));
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(poll).toHaveBeenCalledTimes(1);

    Object.defineProperty(navigator, "onLine", { configurable: true, value: false });
    window.dispatchEvent(new Event("offline"));
    await act(() => vi.advanceTimersByTimeAsync(2000));
    expect(poll).toHaveBeenCalledTimes(1);
    Object.defineProperty(navigator, "onLine", { configurable: true, value: true });
    window.dispatchEvent(new Event("online"));
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(poll).toHaveBeenCalledTimes(2);
  });
});
