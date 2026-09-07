import "@testing-library/jest-dom/vitest";
import { afterEach } from "vitest";

class TestResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

if (typeof globalThis.ResizeObserver === "undefined") {
  globalThis.ResizeObserver = TestResizeObserver as typeof ResizeObserver;
}

try {
  if (typeof window !== "undefined" && typeof window.localStorage?.getItem !== "function") {
    const values = new Map<string, string>();
    Object.defineProperty(window, "localStorage", {
      configurable: true,
      value: {
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => { values.set(key, String(value)); },
        removeItem: (key: string) => { values.delete(key); },
        clear: () => { values.clear(); },
      },
    });
  }
} catch {
  // The browser owns storage in production; tests fall back to in-memory state when unavailable.
}

afterEach(() => {
  try {
    window.localStorage.clear();
  } catch {
    // Storage is optional in the test environment.
  }
});
