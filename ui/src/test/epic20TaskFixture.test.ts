import { describe, expect, it } from "vitest";
import { epic20TaskFixture } from "./epic20TaskFixture";

describe("Epic 20 deterministic task fixture", () => {
  it("keeps snapshot, runtime, queue, partial knowledge and full Git scope risks together", () => {
    expect(new Set(epic20TaskFixture.snapshots.map((item) => item.canonical_path)).size).toBe(1);
    expect(new Set(epic20TaskFixture.snapshots.map((item) => item.content)).size).toBe(2);
    expect(epic20TaskFixture.runtime.desired).not.toBe(epic20TaskFixture.runtime.effective);
    expect(epic20TaskFixture.coordination.pending.pipeline).toBe("refresh");
    expect(epic20TaskFixture.knowledge.status).toBe("partial");
    expect(epic20TaskFixture.git.files.map((item) => item.status)).toEqual(["modified", "untracked", "renamed"]);
  });
});
