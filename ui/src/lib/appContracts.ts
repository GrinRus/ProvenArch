export type Diagnostic = {
  level: "error" | "warning";
  code: string;
  message: string;
  suggestion?: string;
  path?: string;
  repo?: string;
};

export type ValidateResponse = {
  ok: boolean;
  workspace: string;
  warnings?: Diagnostic[];
  errors?: Diagnostic[];
  resolved_repos?: Array<{
    name: string;
    source: string;
    path: string;
    ref?: string;
  }>;
};

export type DoctorCheck = {
  id: string;
  label: string;
  status: "pass" | "warn" | "fail";
  message: string;
  suggestion?: string;
};

export type DoctorResponse = {
  ok: boolean;
  summary: string;
  checks: DoctorCheck[];
};

export type BaselineBundleEditableArtifact = {
  path: string;
  label: string;
  category: string;
  prompt_usage?: string;
};

export type BaselineBundleManifest = {
  schema_version: number;
  bundle_version: number;
  prompt_surface_policy?: {
    live_headless_source?: string;
    reference_only_pattern?: string;
  };
  editable_artifacts?: BaselineBundleEditableArtifact[];
};

export type BaselineBundleResponse = {
  ok: boolean;
  workspace: string;
  warnings?: Diagnostic[];
  manifest?: BaselineBundleManifest;
};

export type RunStartResponse = {
  run_id: string;
  status: string;
};

export type RunStatusResponse = {
  run_id: string;
  pipeline: string;
  status: "queued" | "running" | "succeeded" | "failed";
  started_at: string;
  finished_at?: string | null;
  current_step?: string;
  warnings?: string[];
  pending_permissions?: RuntimePermissionRequest[];
  error_code?: string | null;
  error?: string | null;
};

export type RunListItem = {
  run_id: string;
  pipeline: string;
  status: "queued" | "running" | "succeeded" | "failed";
  started_at: string;
  finished_at?: string | null;
  current_step?: string;
  warnings?: string[];
  pending_permissions?: RuntimePermissionRequest[];
  error_code?: string | null;
  error?: string | null;
};

export type RuntimePermissionRequest = {
  request_id: string;
  run_id: string;
  step_id: string;
  provider: string;
  action: string;
  path_or_command: string;
  reason?: string;
  decision?: RuntimePermissionDecision;
};

export type RuntimePermissionDecision = {
  request_id: string;
  decision: string;
  rule_id: string;
  message?: string;
};

export type RunListResponse = {
  items: RunListItem[];
};

export type Artifact = {
  path: string;
  kind: string;
  label: string;
};

export type ArtifactsResponse = {
  run_id: string;
  artifacts: Artifact[];
};

export type FinalRunIndexDocument = {
  canonical_path: string;
  kind?: string;
  title?: string;
};

export type FinalRunIndex = {
  citation_index_path?: string;
  canonical_documents?: FinalRunIndexDocument[];
};

export type RunLogEntry = {
  cursor: number;
  timestamp: string;
  level: "info" | "warning" | "error";
  kind?: "event" | "runtime_output";
  stream?: "stdout" | "stderr";
  step_id?: string;
  domain_id?: string;
  message: string;
  taskrun_path?: string;
  fields?: Record<string, unknown>;
};

export type RunLogsResponse = {
  run_id: string;
  items: RunLogEntry[];
  next_cursor: number;
  eof: boolean;
};

export type RunReviewStep = {
  step_id: string;
  key: string;
  label: string;
  state: "done" | "active" | "failed" | "pending";
  provider?: string;
  artifact_count: number;
  artifact_paths: string[];
  taskrun_paths: string[];
  warnings_count: number;
  errors_count: number;
  last_message?: string;
};

export type RunReviewSummaryResponse = {
  run_id: string;
  pipeline: string;
  status: "queued" | "running" | "succeeded" | "failed";
  started_at: string;
  finished_at?: string | null;
  current_step?: string;
  warnings?: string[];
  error_code?: string | null;
  error?: string | null;
  steps: RunReviewStep[];
};

export type GitDiffFile = {
  path: string;
  folder: string;
  status: "new" | "modified" | "deleted" | "untracked" | "renamed" | "copied" | "changed" | "unchanged";
  additions: number;
  deletions: number;
  binary: boolean;
};

export type GitDiffFolder = {
  folder: string;
  files: number;
  additions: number;
  deletions: number;
};

export type GitDiffLine = {
  kind: "context" | "add" | "delete" | "meta";
  old_line?: number;
  new_line?: number;
  content: string;
};

export type GitDiffHunk = {
  header: string;
  lines: GitDiffLine[];
};

export type GitDiffResponse = {
  ok: boolean;
  workspace: string;
  run_id?: string | null;
  step_id?: string | null;
  selected_path?: string | null;
  selected_file?: GitDiffFile | null;
  files: GitDiffFile[];
  folders: GitDiffFolder[];
  hunks: GitDiffHunk[];
  message?: string;
  empty: boolean;
};

export type ReviewQueueItem = {
  id: string;
  kind: "report" | "coverage" | "finding" | "question" | "proposal" | "model" | "diagram" | "artifact";
  title: string;
  path: string;
  severity: "info" | "warn" | "error";
};

export type RepoSourceMode = "path" | "git_url";

export type GuidedRepo = {
  id: string;
  name: string;
  mode: RepoSourceMode;
  path: string;
  git_url: string;
  ref: string;
};

export type WizardContract = {
  version: number;
  project_name: string;
  scope: string;
  nfr_priorities: string[];
  rules: string[];
};

export type RuntimeTimeoutKey =
  | "step_timeout_sec"
  | "heartbeat_sec"
  | "pipeline_timeout_sec"
  | "pipeline_kill_grace_sec"
  | "api_ready_timeout_sec"
  | "api_init_timeout_sec"
  | "ui_init_poll_timeout_sec"
  | "ui_cancel_poll_timeout_sec";

export type RuntimeTimeoutValues = Record<RuntimeTimeoutKey, number>;
export type RuntimeTimeoutSources = Record<RuntimeTimeoutKey, string>;

export type RuntimeTimeoutsResponse = {
  ok: boolean;
  persisted?: Partial<RuntimeTimeoutValues>;
  effective?: Partial<RuntimeTimeoutValues>;
  source?: Partial<RuntimeTimeoutSources>;
};

export type RuntimeExecutionKey = "strategy" | "max_parallel_tasks" | "failure_policy" | "shard_discovery_mode";

export type RuntimeExecutionValues = Record<RuntimeExecutionKey, string | number>;
export type RuntimeExecutionSources = Record<RuntimeExecutionKey, string>;

export type RuntimeExecutionResponse = {
  ok: boolean;
  persisted?: Partial<RuntimeExecutionValues>;
  effective?: Partial<RuntimeExecutionValues>;
  source?: Partial<RuntimeExecutionSources>;
};

export type RuntimeExecutionDraft = Record<RuntimeExecutionKey, string>;

export type RuntimeStepProviderValues = Record<string, string>;

export type RuntimePermissionKey = "mode" | "approval_channel";
export type RuntimePermissionValues = Record<RuntimePermissionKey, string>;
export type RuntimePermissionSources = Record<RuntimePermissionKey, string>;
export type RuntimePermissionsResponse = {
  ok: boolean;
  persisted?: Partial<RuntimePermissionValues>;
  effective?: Partial<RuntimePermissionValues>;
  source?: Partial<RuntimePermissionSources>;
};
export type RuntimePermissionDraft = Record<RuntimePermissionKey, string>;

export type OnboardingRuntimeStatus = {
  selected: boolean;
  runtime: string;
  runtime_provider: string;
  provider_source?: string;
};

export type OnboardingStatusResponse = {
  ok: boolean;
  launcher_mode: boolean;
  workspace_selected: boolean;
  workspace_ready: boolean;
  workspace: string;
  manifest_present: boolean;
  runtime: OnboardingRuntimeStatus;
  can_enter_console: boolean;
};

export type RuntimeProfileResponse = {
  ok: boolean;
  permissions?: {
    persisted?: Partial<RuntimePermissionValues>;
    effective?: Partial<RuntimePermissionValues>;
    source?: Partial<RuntimePermissionSources>;
  };
  step_providers?: {
    persisted?: Partial<RuntimeStepProviderValues>;
    effective?: Partial<RuntimeStepProviderValues>;
    source?: Partial<RuntimeStepProviderValues>;
  };
};

export type EditableArtifactOption = {
  path: string;
  label: string;
  category?: string;
  prompt_usage?: string;
};

export const runtimeTimeoutKeys: RuntimeTimeoutKey[] = [
  "step_timeout_sec",
  "heartbeat_sec",
  "pipeline_timeout_sec",
  "pipeline_kill_grace_sec",
  "api_ready_timeout_sec",
  "api_init_timeout_sec",
  "ui_init_poll_timeout_sec",
  "ui_cancel_poll_timeout_sec",
];

export const defaultRuntimeTimeoutValues: RuntimeTimeoutValues = {
  step_timeout_sec: 1800,
  heartbeat_sec: 30,
  pipeline_timeout_sec: 2400,
  pipeline_kill_grace_sec: 30,
  api_ready_timeout_sec: 60,
  api_init_timeout_sec: 120,
  ui_init_poll_timeout_sec: 900,
  ui_cancel_poll_timeout_sec: 420,
};

export const runtimeTimeoutLabels: Record<RuntimeTimeoutKey, string> = {
  step_timeout_sec: "runtime.profile.timeouts.step_timeout_sec",
  heartbeat_sec: "runtime.profile.timeouts.heartbeat_sec",
  pipeline_timeout_sec: "runtime.profile.timeouts.pipeline_timeout_sec",
  pipeline_kill_grace_sec: "runtime.profile.timeouts.pipeline_kill_grace_sec",
  api_ready_timeout_sec: "runtime.profile.timeouts.api_ready_timeout_sec",
  api_init_timeout_sec: "runtime.profile.timeouts.api_init_timeout_sec",
  ui_init_poll_timeout_sec: "runtime.profile.timeouts.ui_init_poll_timeout_sec",
  ui_cancel_poll_timeout_sec: "runtime.profile.timeouts.ui_cancel_poll_timeout_sec",
};

export const runtimeExecutionKeys: RuntimeExecutionKey[] = ["strategy", "max_parallel_tasks", "failure_policy", "shard_discovery_mode"];

export const defaultRuntimeExecutionValues: RuntimeExecutionValues = {
  strategy: "sequential",
  max_parallel_tasks: 1,
  failure_policy: "best_effort",
  shard_discovery_mode: "heuristics",
};

export const runtimeExecutionLabels: Record<RuntimeExecutionKey, string> = {
  strategy: "runtime.profile.execution.strategy",
  max_parallel_tasks: "runtime.profile.execution.max_parallel_tasks",
  failure_policy: "runtime.profile.execution.failure_policy",
  shard_discovery_mode: "runtime.profile.execution.shard_discovery.mode",
};

export const runtimeStepProviderOrder = [
  "step0_constitution",
  "step1_collect",
  "step2_as_is",
  "step3_findings",
  "step4_proposals",
  "qa",
] as const;

export const runtimeStepProviderLabels: Record<(typeof runtimeStepProviderOrder)[number], string> = {
  step0_constitution: "runtime.profile.steps.step0_constitution.provider",
  step1_collect: "runtime.profile.steps.step1_collect.provider",
  step2_as_is: "runtime.profile.steps.step2_as_is.provider",
  step3_findings: "runtime.profile.steps.step3_findings.provider",
  step4_proposals: "runtime.profile.steps.step4_proposals.provider",
  qa: "runtime.profile.steps.qa.provider",
};

export const runtimePermissionKeys: RuntimePermissionKey[] = ["mode", "approval_channel"];

export const defaultRuntimePermissionValues: RuntimePermissionValues = {
  mode: "trusted_full_access",
  approval_channel: "fail_fast",
};

export const runtimePermissionLabels: Record<RuntimePermissionKey, string> = {
  mode: "runtime.profile.permissions.mode",
  approval_channel: "runtime.profile.permissions.approval_channel",
};

let guidedRepoSeed = 0;

export function makeGuidedRepo(partial?: Partial<GuidedRepo>): GuidedRepo {
  guidedRepoSeed += 1;
  return {
    id: partial?.id ?? `repo-${guidedRepoSeed}`,
    name: partial?.name ?? `repo-${guidedRepoSeed}`,
    mode: partial?.mode ?? "git_url",
    path: partial?.path ?? "/absolute/path/to/repository",
    git_url: partial?.git_url ?? "https://github.com/org/repository.git",
    ref: partial?.ref ?? "",
  };
}

export function normalizeRuntimeTimeoutValues(
  partial: Partial<RuntimeTimeoutValues> | undefined,
  fallback: RuntimeTimeoutValues
): RuntimeTimeoutValues {
  const next = { ...fallback };
  for (const key of runtimeTimeoutKeys) {
    const raw = partial?.[key];
    if (typeof raw === "number" && Number.isFinite(raw) && raw > 0) {
      next[key] = Math.floor(raw);
    }
  }
  return next;
}

export function runtimeTimeoutDraftFromValues(values: RuntimeTimeoutValues): Record<RuntimeTimeoutKey, string> {
  const draft = {} as Record<RuntimeTimeoutKey, string>;
  for (const key of runtimeTimeoutKeys) {
    draft[key] = String(values[key]);
  }
  return draft;
}

export function parseRuntimeTimeoutPatch(draft: Record<RuntimeTimeoutKey, string>): RuntimeTimeoutValues {
  const patch = {} as RuntimeTimeoutValues;
  for (const key of runtimeTimeoutKeys) {
    const value = Number.parseInt((draft[key] ?? "").trim(), 10);
    if (!Number.isFinite(value) || value <= 0) {
      throw new Error(`runtime timeout ${key} must be a positive integer`);
    }
    patch[key] = value;
  }
  return patch;
}

export function normalizeRuntimeExecutionValues(
  partial: Partial<RuntimeExecutionValues> | undefined,
  fallback: RuntimeExecutionValues
): RuntimeExecutionValues {
  const strategyRaw = String(partial?.strategy ?? "").trim().toLowerCase();
  const strategy = strategyRaw === "parallel" || strategyRaw === "sequential" ? strategyRaw : String(fallback.strategy);

  const failureRaw = String(partial?.failure_policy ?? "").trim().toLowerCase();
  const failurePolicy = failureRaw === "fail_fast" || failureRaw === "best_effort" ? failureRaw : String(fallback.failure_policy);

  const shardRaw = String(partial?.shard_discovery_mode ?? "").trim().toLowerCase();
  const shardMode = shardRaw === "semantic" || shardRaw === "heuristics" ? shardRaw : String(fallback.shard_discovery_mode);

  const maxRaw = Number(partial?.max_parallel_tasks);
  const maxParallel = Number.isFinite(maxRaw) && maxRaw > 0 ? Math.floor(maxRaw) : Number(fallback.max_parallel_tasks);

  return {
    strategy,
    max_parallel_tasks: maxParallel,
    failure_policy: failurePolicy,
    shard_discovery_mode: shardMode,
  };
}

export function runtimeExecutionDraftFromValues(values: RuntimeExecutionValues): RuntimeExecutionDraft {
  return {
    strategy: String(values.strategy),
    max_parallel_tasks: String(values.max_parallel_tasks),
    failure_policy: String(values.failure_policy),
    shard_discovery_mode: String(values.shard_discovery_mode),
  };
}

export function parseRuntimeExecutionPatch(draft: RuntimeExecutionDraft): RuntimeExecutionValues {
  const strategy = draft.strategy.trim().toLowerCase();
  if (strategy !== "sequential" && strategy !== "parallel") {
    throw new Error("runtime execution strategy must be sequential or parallel");
  }
  const failurePolicy = draft.failure_policy.trim().toLowerCase();
  if (failurePolicy !== "fail_fast" && failurePolicy !== "best_effort") {
    throw new Error("runtime execution failure_policy must be fail_fast or best_effort");
  }
  const shardMode = draft.shard_discovery_mode.trim().toLowerCase();
  if (shardMode !== "heuristics" && shardMode !== "semantic") {
    throw new Error("runtime execution shard_discovery_mode must be heuristics or semantic");
  }
  const maxParallel = Number.parseInt(draft.max_parallel_tasks.trim(), 10);
  if (!Number.isFinite(maxParallel) || maxParallel <= 0) {
    throw new Error("runtime execution max_parallel_tasks must be a positive integer");
  }

  return {
    strategy,
    max_parallel_tasks: maxParallel,
    failure_policy: failurePolicy,
    shard_discovery_mode: shardMode,
  };
}

export function normalizeRuntimePermissionValues(
  partial: Partial<RuntimePermissionValues> | undefined,
  fallback: RuntimePermissionValues
): RuntimePermissionValues {
  const modeRaw = String(partial?.mode ?? "").trim().toLowerCase();
  const mode = modeRaw === "managed" || modeRaw === "trusted_full_access" ? modeRaw : fallback.mode;

  const channelRaw = String(partial?.approval_channel ?? "").trim().toLowerCase();
  const approvalChannel = channelRaw === "ui" || channelRaw === "fail_fast" ? channelRaw : fallback.approval_channel;

  return {
    mode,
    approval_channel: approvalChannel,
  };
}

export function runtimePermissionDraftFromValues(values: RuntimePermissionValues): RuntimePermissionDraft {
  return {
    mode: String(values.mode),
    approval_channel: String(values.approval_channel),
  };
}

export function parseRuntimePermissionPatch(draft: RuntimePermissionDraft): RuntimePermissionValues {
  const mode = draft.mode.trim().toLowerCase();
  if (mode !== "trusted_full_access" && mode !== "managed") {
    throw new Error("runtime permissions mode must be trusted_full_access or managed");
  }
  const approvalChannel = draft.approval_channel.trim().toLowerCase();
  if (approvalChannel !== "fail_fast" && approvalChannel !== "ui") {
    throw new Error("runtime permissions approval_channel must be fail_fast or ui");
  }
  return {
    mode,
    approval_channel: approvalChannel,
  };
}
