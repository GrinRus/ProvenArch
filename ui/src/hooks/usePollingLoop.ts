import { useEffect, useRef } from "react";

type UsePollingLoopOptions = {
  enabled: boolean;
  poll: (signal: AbortSignal) => Promise<boolean | void>;
  intervalMs?: number;
  maxIntervalMs?: number;
};

function isPaused(): boolean {
  if (typeof document !== "undefined" && document.visibilityState === "hidden") {
    return true;
  }
  return typeof navigator !== "undefined" && navigator.onLine === false;
}

/**
 * Runs one bounded, sequential polling lifecycle. A false result or rejection
 * backs off exponentially; success returns to the base interval. Polling is
 * paused while the tab is hidden/offline and resumes with one fresh request.
 */
export function usePollingLoop({
  enabled,
  poll,
  intervalMs = 1000,
  maxIntervalMs = 8000,
}: UsePollingLoopOptions) {
  const pollRef = useRef(poll);
  pollRef.current = poll;

  useEffect(() => {
    if (!enabled) {
      return;
    }

    let disposed = false;
    let timer: number | null = null;
    let failures = 0;
    let inFlight = false;
    let controller: AbortController | null = null;
    let pauseGeneration = 0;

    const clearTimer = () => {
      if (timer !== null) {
        window.clearTimeout(timer);
        timer = null;
      }
    };

    const schedule = (delay: number) => {
      clearTimer();
      if (disposed || isPaused()) {
        return;
      }
      timer = window.setTimeout(() => {
        timer = null;
        void run();
      }, Math.max(0, delay));
    };

    const run = async () => {
      if (disposed || inFlight || isPaused()) {
        return;
      }
      inFlight = true;
      const generation = pauseGeneration;
      controller = new AbortController();
      let succeeded = true;
      try {
        succeeded = (await pollRef.current(controller.signal)) !== false;
      } catch {
        succeeded = false;
      } finally {
        controller = null;
        inFlight = false;
      }
      if (disposed) {
        return;
      }
      if (generation !== pauseGeneration) {
        if (!isPaused()) schedule(0);
        return;
      }
      if (isPaused()) return;
      failures = succeeded ? 0 : Math.min(failures + 1, 30);
      const backoff = intervalMs * 2 ** failures;
      schedule(Math.min(maxIntervalMs, succeeded ? intervalMs : backoff));
    };

    const resume = () => {
      if (!disposed && !isPaused() && !inFlight) {
        schedule(0);
      }
    };
    const pause = () => {
      pauseGeneration += 1;
      clearTimer();
      controller?.abort();
    };

    document.addEventListener("visibilitychange", resume);
    window.addEventListener("online", resume);
    window.addEventListener("offline", pause);
    schedule(intervalMs);

    return () => {
      disposed = true;
      clearTimer();
      controller?.abort();
      document.removeEventListener("visibilitychange", resume);
      window.removeEventListener("online", resume);
      window.removeEventListener("offline", pause);
    };
  }, [enabled, intervalMs, maxIntervalMs]);
}
