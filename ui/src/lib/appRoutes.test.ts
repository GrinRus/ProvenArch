import { describe, expect, it } from "vitest";

import { destinationForStage, formatAppRoute, parseAppRoute, stageForRoute } from "./appRoutes";

const location = (value: string) => {
  const url = new URL(value, "http://localhost");
  return { pathname: url.pathname, search: url.search } as Location;
};

describe("application route codec", () => {
  it("canonicalizes every route to setup before Console entry", () => {
    expect(parseAppRoute(location("/changes?run=old"), false)).toMatchObject({ destination: "setup", setupStep: "workspace" });
  });

  it("defaults an unknown console path to the authoritative Task Inbox", () => {
    expect(parseAppRoute(location("/"), true)).toMatchObject({ destination: "tasks", taskView: "inbox", invalid: [] });
    expect(parseAppRoute(location("/unknown-path"), true)).toMatchObject({ destination: "tasks", taskView: "inbox" });
  });

  it("round-trips run and Changes context", () => {
    const changes = parseAppRoute(location("/changes?run=run-1&view=evidence&source=snapshot&artifact=doc.overview&mode=raw"), true);
    expect(changes).toMatchObject({ destination: "changes", runId: "run-1", changesView: "evidence", source: "snapshot", artifact: "doc.overview", mode: "raw" });
    expect(formatAppRoute(changes)).toBe("/changes?run=run-1&view=evidence&source=snapshot&artifact=doc.overview&mode=raw");
    expect(parseAppRoute(location("/runs/run-1"), true)).toMatchObject({ destination: "tasks", taskView: "legacy", runId: "run-1" });
    expect(formatAppRoute(parseAppRoute(location("/runs/run-1"), true))).toBe("/tasks/legacy/run-1");
    expect(parseAppRoute(location("/runs"), true)).toMatchObject({ destination: "tasks", taskView: "legacy" });
    expect(formatAppRoute(parseAppRoute(location("/runs"), true))).toBe("/tasks/legacy");
  });

  it("preserves optional Task context on Architecture and Changes targets", () => {
    const changes = parseAppRoute(location("/changes?task=task-1&attempt=attempt-1&run=run-1&view=evidence&source=snapshot"), true);
    expect(changes).toMatchObject({ destination: "changes", taskId: "task-1", attemptId: "attempt-1", runId: "run-1" });
    expect(formatAppRoute(changes)).toBe("/changes?task=task-1&attempt=attempt-1&run=run-1&view=evidence&source=snapshot&mode=rendered");
    const architecture = parseAppRoute(location("/architecture?task=task-1&view=map&source=current"), true);
    expect(architecture).toMatchObject({ destination: "knowledge", taskId: "task-1" });
    expect(formatAppRoute(architecture)).toBe("/architecture?view=map&source=current&task=task-1");
  });

  it("drops unsafe Task context instead of treating it as an authority", () => {
    const route = parseAppRoute(location("/changes?task=task%2Fforeign&view=overview"), true);
    expect(route.taskId).toBeUndefined();
    expect(route.invalid).toContain("task");
    expect(formatAppRoute(route)).toBe("/changes?view=overview&source=snapshot&mode=rendered");
  });

  it("round-trips typed Task and Attempt identities without run fallback", () => {
    expect(parseAppRoute(location("/tasks"), true)).toMatchObject({ destination: "tasks", taskView: "inbox" });
    expect(parseAppRoute(location("/tasks/new"), true)).toMatchObject({ destination: "tasks", taskView: "new" });
    const task = parseAppRoute(location("/tasks/task-opaque"), true);
    expect(task).toMatchObject({ destination: "tasks", taskView: "detail", taskId: "task-opaque" });
    expect(formatAppRoute(task)).toBe("/tasks/task-opaque");
    const attempt = parseAppRoute(location("/tasks/task-opaque/attempts/attempt-opaque"), true);
    expect(attempt).toMatchObject({ destination: "tasks", taskView: "attempt", taskId: "task-opaque", attemptId: "attempt-opaque" });
    expect(formatAppRoute(attempt)).toBe("/tasks/task-opaque/attempts/attempt-opaque");
    const studio = parseAppRoute(location("/tasks/task-opaque/attempts/attempt-opaque/studio"), true);
    expect(studio).toMatchObject({ destination: "tasks", taskView: "studio", taskId: "task-opaque", attemptId: "attempt-opaque" });
    expect(formatAppRoute(studio)).toBe("/tasks/task-opaque/attempts/attempt-opaque/studio");
    const legacy = parseAppRoute(location("/tasks/legacy/run-opaque"), true);
    expect(legacy).toMatchObject({ destination: "tasks", taskView: "legacy", runId: "run-opaque" });
    expect(formatAppRoute(legacy)).toBe("/tasks/legacy/run-opaque");
  });

  it("round-trips URL-restorable Task Inbox filters through detail", () => {
    const route = parseAppRoute(location("/tasks?search=payments&lifecycle=open&runner=claude-code&repository=payments&from=2026-08-01T00:00:00Z&to=2026-08-11T23:59:59Z"), true);
    expect(route.taskFilters).toEqual({ search: "payments", lifecycle: "open", runner: "claude-code", repository: "payments", from: "2026-08-01T00:00:00Z", to: "2026-08-11T23:59:59Z" });
    expect(formatAppRoute({ ...route, taskView: "detail", taskId: "task-1" })).toBe("/tasks/task-1?search=payments&lifecycle=open&runner=claude-code&repository=payments&from=2026-08-01T00%3A00%3A00Z&to=2026-08-11T23%3A59%3A59Z");
  });

  it("rejects an inverted Task activity range without selecting a different identity", () => {
    const route = parseAppRoute(location("/tasks?from=2026-08-11&to=2026-08-01"), true);
    expect(route.invalid).toContain("task_time_range");
    expect(route.taskFilters?.from).toBeUndefined();
    expect(route.taskFilters?.to).toBeUndefined();
  });

  it("rejects ambiguous Task identities and never chooses another item", () => {
    const malformed = parseAppRoute(location("/tasks/task%2Fwith-slash"), true);
    expect(malformed).toMatchObject({ destination: "tasks", taskView: "inbox" });
    expect(malformed.taskId).toBeUndefined();
    expect(malformed.invalid).toContain("task");
    expect(formatAppRoute(malformed)).toBe("/tasks");
    const malformedAttempt = parseAppRoute(location("/tasks/task-1/attempts/"), true);
    expect(malformedAttempt.taskId).toBeUndefined();
    expect(malformedAttempt.invalid).toContain("task_route");
    expect(formatAppRoute(malformedAttempt)).toBe("/tasks");
    const reservedAttempt = parseAppRoute(location("/tasks/new/attempts/attempt-1"), true);
    expect(reservedAttempt.taskId).toBeUndefined();
    expect(reservedAttempt.invalid).toContain("task");
  });

  it("sanitizes unsupported values without changing source", () => {
    const route = parseAppRoute(location("/changes?run=missing&view=magic&source=snapshot&mode=html"), true);
    expect(route.invalid).toEqual(["view", "mode"]);
    expect(route).toMatchObject({ runId: "missing", changesView: "overview", source: "snapshot", mode: "rendered" });
  });

  it("maps setup and stage ownership", () => {
    expect(stageForRoute(parseAppRoute(location("/setup?step=runner"), true))).toBe("readiness");
    expect(destinationForStage("publish")).toBe("changes");
  });

  it("opens Architecture in the semantic Map while preserving explicit document routes", () => {
    const map = parseAppRoute(location("/architecture"), true);
    expect(map).toMatchObject({ destination: "knowledge", knowledgeView: "map" });
    expect(formatAppRoute(map)).toBe("/architecture?view=map&source=current");
    const documents = parseAppRoute(location("/architecture?view=documents"), true);
    expect(documents).toMatchObject({ destination: "knowledge", knowledgeView: "documents" });
    expect(parseAppRoute(location("/architecture/map"), true)).toMatchObject({ destination: "knowledge", knowledgeView: "map" });
  });

  it("does not crash on malformed percent-encoded path segments", () => {
    const route = parseAppRoute({ pathname: "/runs/%E0%A4%A", search: "" } as Location, true);
    expect(route.destination).toBe("tasks");
    expect(route.taskView).toBe("inbox");
    expect(route.runId).toBeUndefined();
    expect(route.invalid).toContain("path");
  });
});
