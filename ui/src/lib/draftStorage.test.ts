import { afterEach, describe, expect, it, vi } from "vitest";

import { clearDraft, draftStorageKey, readDraft, writeDraft } from "./draftStorage";

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

describe("draftStorage", () => {
  it("round-trips versioned drafts with a scoped key", () => {
    const key = draftStorageKey("task-composer", "/workspace with spaces");
    expect(writeDraft(key, { goal: "trace recovery" })).toBe(true);
    expect(readDraft<{ goal: string }>(key)?.value).toEqual({ goal: "trace recovery" });
    expect(readDraft<{ goal: string }>(key)?.version).toBe(1);
    clearDraft(key);
    expect(readDraft(key)).toBeNull();
  });

  it("ignores malformed or unsupported records", () => {
    const key = draftStorageKey("setup", "/workspace");
    window.localStorage.setItem(key, JSON.stringify({ version: 2, savedAt: new Date().toISOString(), value: {} }));
    expect(readDraft(key)).toBeNull();
    window.localStorage.setItem(key, "not-json");
    expect(readDraft(key)).toBeNull();
  });

  it("fails safely when browser storage is unavailable", () => {
    vi.spyOn(window.localStorage, "setItem").mockImplementation(() => { throw new Error("quota"); });
    expect(writeDraft("key", { value: true })).toBe(false);
    expect(() => clearDraft("key")).not.toThrow();
  });
});
