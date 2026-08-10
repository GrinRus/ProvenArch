import { describe, expect, it } from "vitest";

import { buildAnalysisArtifactPairState, buildAnalysisShardRows, buildAnalysisStepTimeline, diffFileTone } from "./analysisViewModels";

const artifact = (path: string) => ({ path, kind: "markdown", label: path });

describe("analysis view-model selectors", () => {
  it("derives a completed five-step timeline", () => {
    const steps = buildAnalysisStepTimeline({ run_id: "run-1", pipeline: "init", status: "succeeded", started_at: "2026-08-10T10:00:00Z" }, []);
    expect(steps).toHaveLength(5);
    expect(steps.every((step) => step.state === "done")).toBe(true);
  });

  it("classifies complete and partial shard artifact pairs", () => {
    const complete = buildAnalysisArtifactPairState("repo-a", "staging/shards/repo-a/taskrun", [
      artifact("staging/shards/repo-a/overview.md"),
      artifact("staging/shards/repo-a/shard-pack-manifest.json"),
    ]);
    const partial = buildAnalysisArtifactPairState("repo-b", "staging/shards/repo-b/taskrun", [artifact("staging/shards/repo-b/runtime-execution.json")]);
    expect(complete.tone).toBe("ok");
    expect(partial.tone).toBe("error");
  });

  it("groups shard logs and preserves provider/status semantics", () => {
    const rows = buildAnalysisShardRows(
      { run_id: "run-1", pipeline: "init", status: "running", started_at: "2026-08-10T10:00:00Z", current_step: "init.step1.collect" },
      [{ cursor: 1, timestamp: "2026-08-10T10:00:01Z", level: "warning", step_id: "init.step1.collect", message: "partial", fields: { shard_id: "repo-a", provider: "codex-code", duration_ms: 1200 } }],
      [],
      "headless",
      "codex-code",
    );
    expect(rows).toHaveLength(1);
    expect(rows[0].status).toBe("warning");
    expect(rows[0].provider).toBe("codex-code");
    expect(rows[0].duration).toContain("1s");
  });

  it("maps diff statuses to stable visual tones", () => {
    expect(diffFileTone("deleted")).toBe("error");
    expect(diffFileTone("modified")).toBe("warn");
    expect(diffFileTone("new")).toBe("ok");
  });
});
