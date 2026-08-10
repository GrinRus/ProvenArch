import { describe, expect, it } from "vitest";

import { artifactHandoffStalled, formatShardMetric, parseShardCounters, summarizeProviderStream } from "./providerStreamUtils";

describe("provider stream selectors", () => {
  it("summarizes runtime output streams and JSON signals", () => {
    const summary = summarizeProviderStream([
      { cursor: 1, timestamp: "2026-08-10T10:00:00Z", level: "info", kind: "runtime_output", stream: "stdout", message: '{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta"}}}' },
      { cursor: 2, timestamp: "2026-08-10T10:00:01Z", level: "info", kind: "runtime_output", stream: "stderr", message: "provider note" },
      { cursor: 3, timestamp: "2026-08-10T10:00:02Z", level: "info", message: "not runtime output" },
    ]);
    expect(summary).toMatchObject({ chunks: 2, streamEvents: 1, stdout: 1, stderr: 1 });
    expect(summary.signalTypes).toEqual(["text_delta"]);
  });

  it("parses shard counters and keeps unknown text safe", () => {
    expect(parseShardCounters("shards_total=5 succeeded=4 failed=1")).toEqual({ planned: 5, succeeded: 4, failed: 1 });
    expect(parseShardCounters("no counters")).toBeNull();
    expect(formatShardMetric(5, 5, 4, 1)).toBe("4/5 ok · 1 failed");
    expect(formatShardMetric(undefined, 2, 2, 0)).toBe("0 failed / 2 observed");
  });

  it("recognizes artifact handoff stalls only for the known signals", () => {
    expect(artifactHandoffStalled("stalled before valid artifacts", "", "collect_pair_repair")).toBe(true);
    expect(artifactHandoffStalled("provider stalled", "", "collect")).toBe(false);
  });
});
