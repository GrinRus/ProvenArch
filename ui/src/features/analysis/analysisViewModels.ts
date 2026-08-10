import {
  fieldString,
  formatDurationMillis,
  numericField,
} from "./analysisUtils";
import type { Artifact, RunLogEntry, RunReviewStep, RunStatusResponse } from "../../lib/appContracts";

export type AnalysisStepState = "done" | "active" | "failed" | "pending";

export type AnalysisStep = {
  id: string;
  label: string;
  state: AnalysisStepState;
  detail: string;
};

export type AnalysisArtifactPairState = {
  label: string;
  detail: string;
  tone: "info" | "ok" | "warn" | "error";
  runtimeRefs: string[];
  markdownRefs: string[];
  manifestRefs: string[];
};

export type AnalysisShardRow = {
  key: string;
  stepId: string;
  scope: string;
  provider: string;
  status: "succeeded" | "active" | "failed" | "warning" | "observed";
  artifactRef: string;
  artifactPair: AnalysisArtifactPairState;
  duration: string;
  lastMessage: string;
};

const canonicalAnalysisSteps = [
  { suffix: "step0.constitution", label: "Charter" },
  { suffix: "step1.collect", label: "Collect" },
  { suffix: "step2.asis_docs", label: "As-is docs" },
  { suffix: "step3.findings", label: "Findings" },
  { suffix: "step4.proposals", label: "Proposals" },
];

export function buildAnalysisStepTimeline(runStatus: RunStatusResponse | null, runLogs: RunLogEntry[]): AnalysisStep[] {
  const pipeline = runStatus?.pipeline || "init";
  const currentIndex = findStepIndex(runStatus?.current_step);
  const loggedIndex = runLogs.reduce((maxIndex, entry) => Math.max(maxIndex, findStepIndex(entry.step_id)), -1);
  const activeIndex = currentIndex >= 0 ? currentIndex : loggedIndex >= 0 ? loggedIndex : 0;
  return canonicalAnalysisSteps.map((step, index) => {
    const id = `${pipeline}.${step.suffix}`;
    let state: AnalysisStepState = "pending";
    if (runStatus?.status === "succeeded") {
      state = "done";
    } else if (runStatus?.status === "failed") {
      state = index < activeIndex ? "done" : index === activeIndex ? "failed" : "pending";
    } else if (runStatus?.status === "running" || runStatus?.status === "queued") {
      state = index < activeIndex ? "done" : index === activeIndex ? "active" : "pending";
    } else if (loggedIndex >= index && loggedIndex >= 0) {
      state = "done";
    }
    return { id, label: step.label, state, detail: stepTimelineDetail(state) };
  });
}

export function buildAnalysisShardRows(
  runStatus: RunStatusResponse | null,
  runLogs: RunLogEntry[],
  artifacts: Artifact[],
  setupRuntime: string,
  setupRuntimeProvider: string,
): AnalysisShardRow[] {
  const grouped = new Map<string, RunLogEntry[]>();
  for (const entry of runLogs) {
    const shardScope = fieldString(entry.fields, "shard_id") || shardScopeFromPath(entry.taskrun_path ?? "");
    const key = shardScope ? `${entry.step_id || "run"}/shard/${shardScope}` : entry.taskrun_path || `${entry.step_id || "run"}/${entry.domain_id || "workspace"}`;
    grouped.set(key, [...(grouped.get(key) ?? []), entry]);
  }
  const provider = setupRuntime === "fake" ? "fake" : setupRuntimeProvider;
  const rows: AnalysisShardRow[] = [];
  for (const [key, entries] of grouped.entries()) {
    const last = entries[entries.length - 1];
    const stepId = last?.step_id || entries.find((entry) => entry.step_id)?.step_id || runStatus?.current_step || "-";
    const hasError = entries.some((entry) => entry.level === "error");
    const hasWarning = entries.some((entry) => entry.level === "warning");
    const shardScope = fieldString(last?.fields, "shard_id") || shardScopeFromPath(last?.taskrun_path ?? "");
    const scope = shardScope || last?.domain_id || fieldString(last?.fields, "domain_id") || fieldString(last?.fields, "repo") || "workspace";
    const artifactRef = [...entries].reverse().find((entry) => entry.taskrun_path)?.taskrun_path || (artifacts.length > 0 ? `${artifacts.length} selected-run artifacts` : "logs only");
    rows.push({
      key,
      stepId,
      scope,
      provider: setupRuntime === "fake" ? "fake" : fieldString(last?.fields, "provider") || provider,
      status: hasError ? "failed" : hasWarning ? "warning" : runStatus?.status === "succeeded" ? "succeeded" : runStatus?.current_step && stepMatches(runStatus.current_step, stepId) ? "active" : "observed",
      artifactRef,
      artifactPair: buildAnalysisArtifactPairState(scope, artifactRef, artifacts),
      duration: durationFromEntries(entries),
      lastMessage: last?.message || "-",
    });
  }
  if (rows.length === 0 && runStatus) {
    rows.push({
      key: runStatus.run_id,
      stepId: runStatus.current_step || `${runStatus.pipeline}.pending`,
      scope: "workspace",
      provider,
      status: runStatus.status === "failed" ? "failed" : runStatus.status === "succeeded" ? "succeeded" : runStatus.status === "running" ? "active" : "observed",
      artifactRef: artifacts.length > 0 ? `${artifacts.length} selected-run artifacts` : "status only",
      artifactPair: buildAnalysisArtifactPairState("workspace", artifacts.length > 0 ? `${artifacts.length} selected-run artifacts` : "status only", artifacts),
      duration: "Duration unavailable",
      lastMessage: runStatus.error || runStatus.error_code || "No shard logs loaded yet.",
    });
  }
  return rows;
}

export function buildAnalysisArtifactPairState(scope: string, artifactRef: string, artifacts: Artifact[]): AnalysisArtifactPairState {
  const shardScoped = scope !== "workspace" && scope.trim().length > 0;
  const normalizedScope = scope.trim();
  const selectedPaths = artifacts.map((artifact) => artifact.path).filter((path) => pathMatchesShardScope(path, normalizedScope));
  const refPaths = pathMatchesShardScope(artifactRef, normalizedScope) ? [artifactRef] : [];
  const paths = Array.from(new Set([...selectedPaths, ...refPaths]));
  const runtimeRefs = paths.filter((path) => path.endsWith("/runtime-execution.json"));
  const markdownRefs = paths.filter((path) => /\.(md|markdown)$/i.test(path) && !path.endsWith("/shard-pack-manifest.md"));
  const manifestRefs = paths.filter((path) => path.endsWith("/shard-pack-manifest.json"));

  if (!shardScoped) {
    return {
      label: "Run-level evidence",
      detail: artifacts.length > 0 ? "selected-run artifacts are available, but this row is not shard-scoped" : "artifact list not loaded for this run",
      tone: artifacts.length > 0 ? "info" : "warn",
      runtimeRefs,
      markdownRefs,
      manifestRefs,
    };
  }
  if (markdownRefs.length > 0 && manifestRefs.length > 0) {
    return { label: "Artifact pair present", detail: "authored markdown and shard-pack-manifest are both visible", tone: "ok", runtimeRefs, markdownRefs, manifestRefs };
  }
  if (markdownRefs.length > 0) {
    return { label: "Markdown only", detail: "authored markdown is visible, but shard-pack-manifest is missing", tone: "warn", runtimeRefs, markdownRefs, manifestRefs };
  }
  if (manifestRefs.length > 0) {
    return { label: "Manifest only", detail: "shard-pack-manifest is visible, but authored markdown is missing", tone: "warn", runtimeRefs, markdownRefs, manifestRefs };
  }
  if (runtimeRefs.length > 0) {
    return { label: "Runtime only", detail: "runtime-execution exists; authored markdown and shard-pack-manifest are missing", tone: "error", runtimeRefs, markdownRefs, manifestRefs };
  }
  return {
    label: artifacts.length > 0 ? "No shard artifacts" : "Artifact list not loaded",
    detail: artifacts.length > 0 ? "selected-run artifacts do not include this shard" : "load selected-run artifacts before retry triage",
    tone: artifacts.length > 0 ? "warn" : "info",
    runtimeRefs,
    markdownRefs,
    manifestRefs,
  };
}

function pathMatchesShardScope(path: string, scope: string): boolean {
  if (!scope) return false;
  const shardSegment = `staging/shards/${scope}/`;
  return path.includes(`/${shardSegment}`) || path.startsWith(shardSegment);
}

function shardScopeFromPath(path: string): string {
  const normalized = path.replace(/\\/g, "/");
  const match = normalized.match(/(?:^|\/)staging\/shards\/([^/]+)\//);
  return match?.[1] ?? "";
}

export function formatArtifactPairRefs(paths: string[]): string {
  if (paths.length === 0) return "missing";
  const visible = paths.slice(0, 2).join(", ");
  return paths.length > 2 ? `${visible} +${paths.length - 2} more` : visible;
}

function findStepIndex(stepId?: string): number {
  if (!stepId) return -1;
  const normalized = stepId.replace(/_/g, ".").toLowerCase();
  return canonicalAnalysisSteps.findIndex((step) => normalized.includes(step.suffix.replace(/_/g, ".")));
}

export function stepMatches(left: string, right: string): boolean {
  return findStepIndex(left) >= 0 && findStepIndex(left) === findStepIndex(right);
}

function stepTimelineDetail(state: AnalysisStepState): string {
  if (state === "done") return "completed";
  if (state === "active") return "current";
  if (state === "failed") return "blocked";
  return "pending";
}

export function selectedStepTone(state?: RunReviewStep["state"]): "info" | "ok" | "warn" | "error" {
  if (state === "done") return "ok";
  if (state === "failed") return "error";
  if (state === "active") return "warn";
  return "info";
}

export function diffFileTone(status?: string): "info" | "ok" | "warn" | "error" {
  if (status === "deleted") return "error";
  if (status === "modified" || status === "renamed" || status === "changed") return "warn";
  if (status === "new" || status === "untracked" || status === "copied") return "ok";
  return "info";
}

export function capitalize(value: string): string {
  return value.slice(0, 1).toUpperCase() + value.slice(1);
}

function durationFromLogFields(fields: Record<string, unknown> | undefined): string {
  const direct = fieldString(fields, "duration") || fieldString(fields, "elapsed") || fieldString(fields, "runtime_duration");
  if (direct) return direct;
  const millis = numericField(fields, "duration_ms") ?? numericField(fields, "elapsed_ms") ?? numericField(fields, "runtime_duration_ms");
  if (millis !== undefined) return formatDurationMillis(millis);
  const seconds = numericField(fields, "duration_sec") ?? numericField(fields, "elapsed_sec") ?? numericField(fields, "runtime_duration_sec");
  if (seconds !== undefined) return formatDurationMillis(seconds * 1000);
  return "Duration unavailable";
}

function durationFromEntries(entries: RunLogEntry[]): string {
  for (const entry of [...entries].reverse()) {
    const duration = durationFromLogFields(entry.fields);
    if (duration !== "Duration unavailable") return duration;
  }
  return "Duration unavailable";
}
