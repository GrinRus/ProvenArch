import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { useManifestEditor } from "./useManifestEditor";

vi.mock("../lib/workspaceApi", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/workspaceApi")>();
  return {
    ...actual,
    loadWorkspaceManifest: vi.fn(async () => "version: 1\nrepos: []\n"),
    saveWorkspaceManifest: vi.fn(async () => undefined),
    validateWorkspaceAPI: vi.fn(async () => ({ version: 1, ok: true, workspace: "/workspace-a", errors: [], warnings: [] })),
  };
});

afterEach(() => {
  window.localStorage.clear();
  vi.clearAllMocks();
});

describe("useManifestEditor draft recovery", () => {
  it("restores a workspace setup draft after remount", async () => {
    const options = { setBusy: vi.fn(), setError: vi.fn(), workspaceKey: "/workspace-a" };
    const first = renderHook(() => useManifestEditor(options));
    act(() => first.result.current.setManifestContent("version: 1\nrepos:\n  - name: payments\n"));
    await waitFor(() => expect(first.result.current.hasUnsavedManifestDraft).toBe(true));
    first.unmount();

    const recovered = renderHook(() => useManifestEditor(options));
    await waitFor(() => expect(recovered.result.current.manifestContent).toContain("payments"));
    expect(recovered.result.current.manifestStatus).toContain("Recovered an unsaved workspace draft");
    expect(recovered.result.current.hasUnsavedManifestDraft).toBe(true);
    recovered.unmount();

    const otherWorkspace = renderHook(() => useManifestEditor({ ...options, workspaceKey: "/workspace-b" }));
    await waitFor(() => expect(otherWorkspace.result.current.manifestContent).toBe(""));
    expect(otherWorkspace.result.current.hasUnsavedManifestDraft).toBe(false);
  });
});
