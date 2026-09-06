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
    resolved_sha?: string;
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

export type SystemVersionResponse = {
  version: string;
  commit: string;
  built: string;
  ui_bundle: string;
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

export type WorkspaceHealthSeverity = "info" | "warning" | "error";

export type WorkspaceHealthStatus = "pass" | "warn" | "fail";

export type WorkspaceHealthItem = {
  id: string;
  severity: WorkspaceHealthSeverity;
  title: string;
  path?: string;
  related_paths: string[];
};

export type WorkspaceHealthResponse = {
  version: number;
  generated_at: string;
  status: WorkspaceHealthStatus;
  summary: {
    info: number;
    warning: number;
    error: number;
  };
  items: WorkspaceHealthItem[];
};

export type KnowledgeEntity = {
  id: string;
  type: string;
  name: string;
  aliases?: string[];
  tags?: string[];
  attributes?: unknown;
  owner_team_id?: string;
  provenance: { kind: string; confidence: number; evidence?: unknown[] };
  path: string;
};

export type KnowledgeEdge = {
  id: string;
  type: string;
  from: string;
  to: string;
  name?: string;
  attributes?: unknown;
  provenance: { kind: string; confidence: number; evidence?: unknown[] };
  path: string;
};

export type KnowledgeArtifact = {
  path: string;
  kind: "entity" | "edge" | "proposal" | "report" | "model" | string;
  name: string;
};

export type KnowledgeIssue = {
  code: string;
  path?: string;
  message: string;
};

export type KnowledgeResponse = {
  version: number;
  generated_at: string;
  source_mode: "promoted_current";
  status: "available" | "partial" | "unavailable";
  entities: KnowledgeEntity[];
  edges: KnowledgeEdge[];
  artifacts: KnowledgeArtifact[];
  issues: KnowledgeIssue[];
};

export type ArchitectureLevel = "context" | "container" | "component" | "code";

export type ArchitectureNode = {
  id: string;
  name: string;
  type: string;
  owner_team_id?: string;
  tags?: string[];
  confidence: number;
  provenance_kind: string;
  evidence?: Array<{ repo: string; path: string; ref?: string; lines?: { start: number; end: number } }>;
  path: string;
  available_levels?: ArchitectureLevel[];
  child_levels?: ArchitectureLevel[];
  detail_unavailable_reason?: string;
  repositories?: string[];
  related_findings?: string[];
  related_questions?: string[];
};

export type ArchitectureEdge = {
  id: string;
  from: string;
  to: string;
  type: string;
  name?: string;
  confidence: number;
  provenance_kind: string;
  evidence?: ArchitectureNode["evidence"];
  path: string;
  repositories?: string[];
  related_findings?: string[];
  related_questions?: string[];
};

export type ArchitectureFinding = {
  id: string;
  severity: string;
  title: string;
  description?: string;
  rule_id?: string;
  related_ids?: string[];
  provenance?: {
    kind?: string;
    confidence?: number;
    evidence?: Array<{ repo: string; path: string; ref?: string; lines?: { start: number; end: number }; excerpt?: string }>;
  };
};

export type ArchitectureView = {
  level: ArchitectureLevel;
  available: boolean;
  unavailable_reason?: string;
  nodes: ArchitectureNode[];
  edges: ArchitectureEdge[];
};

export type ArchitectureResponse = {
  version: number;
  generated_at: string;
  authority: { mode: "promoted_current"; source_run_id?: string; promoted_at?: string; freshness: "current" | "recent" | "stale" | "unknown" };
  status: "available" | "partial" | "unavailable";
  counts: { entities: number; edges: number; evidence: number; issues: number };
  views: Record<ArchitectureLevel, ArchitectureView>;
  exports?: { home_path?: string; c4_mermaid_paths: string[] };
  comparison?: ArchitectureComparison;
  review?: {
    findings: ArchitectureFinding[];
    questions: Array<{ id: string; text: string; priority?: string; related_ids?: string[] }>;
  };
  coverage?: { observed?: string[]; missing?: string[]; notes?: string[] };
  artifacts: KnowledgeArtifact[];
  issues: KnowledgeIssue[];
};

export type ArchitectureChangeItem = { id: string; name: string; path?: string };
export type ArchitectureChangeSet = { added: ArchitectureChangeItem[]; changed: ArchitectureChangeItem[]; removed: ArchitectureChangeItem[] };
export type ArchitectureComparison = { available: boolean; baseline_run_id?: string; current_run_id?: string; reason?: string; categories: Record<"entities" | "edges" | "findings" | "gaps", ArchitectureChangeSet> };
export type RunReviewContract = {
  review_kind: "initial" | "refresh";
  source_run_id: string;
  baseline_run_id?: string;
  semantic_changes: ArchitectureComparison;
  document_changes: ArchitectureChangeSet & { available: boolean; reason?: string };
  findings: ArchitectureFinding[];
  questions: Array<{ id: string; text: string; priority?: string; related_ids?: string[] }>;
  gaps: string[];
  summary: {
    entities_added: number;
    entities_changed: number;
    entities_removed: number;
    edges_added: number;
    edges_changed: number;
    edges_removed: number;
    documents_added: number;
    documents_changed: number;
    documents_removed: number;
    findings: number;
    questions: number;
    gaps: number;
  };
  runtime: {
    mode?: string;
    providers: string[];
    step_providers: Record<string, string>;
    provider_models?: Record<string, { model?: string; effort?: string }>;
  };
  authority: {
    mode: string;
    source_run_id: string;
    baseline_run_id?: string;
    snapshot_path?: string;
  };
  generated_at: string;
};

export type RunProgress = {
  phase: "queued" | "provider_working" | "artifact_observed" | "validating" | "repairing" | "stalled" | "completed" | "succeeded" | "failed" | "canceled";
  completed_steps: number;
  total_steps: number;
  current_step?: string;
  expected_result?: string;
  planned_units?: number;
  running_units?: number;
  succeeded_units?: number;
  failed_units?: number;
  current_scopes?: string[];
  started_at: string;
  elapsed_ms?: number;
  last_activity_at?: string;
  last_progress_at?: string;
  artifact_state?: string;
  repair_attempt?: number;
  repair_limit?: number;
  stall_deadline_at?: string;
};

export type RetryLineage = { parent_run_id: string; reason: string; requested_step: string; effective_start_step: string; requested_scopes?: string[]; effective_scopes?: string[]; reused_inputs?: string[] };

export type RunStatusResponse = {
  run_id: string;
  pipeline: string;
  status: "queued" | "running" | "succeeded" | "failed" | "canceled";
  started_at: string;
  finished_at?: string | null;
  current_step?: string;
  runtime_mode?: "fake" | "headless" | null;
  step_providers?: Record<string, string>;
  warnings?: string[];
  pending_permissions?: RuntimePermissionRequest[];
  error_code?: string | null;
  error?: string | null;
  superseded_by_run_id?: string | null;
  refresh_summary?: RefreshSummary | null;
  progress?: RunProgress | null;
  retry?: RetryLineage | null;
};

export type RefreshSummary = {
  mode: "no_op" | "affected_only" | "full";
  decision: "unchanged_candidate" | "selective_candidate" | "full_refresh_required";
  baseline_run_id?: string;
  reason_codes: string[];
  artifact_path: string;
  updated: number;
  preserved: number;
  removed: number;
  uncertain: number;
};

export type RunListItem = {
  run_id: string;
  pipeline: string;
  status: "queued" | "running" | "succeeded" | "failed" | "canceled";
  started_at: string;
  finished_at?: string | null;
  current_step?: string;
  runtime_mode?: "fake" | "headless" | null;
  step_providers?: Record<string, string>;
  warnings?: string[];
  pending_permissions?: RuntimePermissionRequest[];
  error_code?: string | null;
  error?: string | null;
  superseded_by_run_id?: string | null;
  refresh_summary?: RefreshSummary | null;
  authoritative_index?: boolean;
  progress?: RunProgress | null;
  retry?: RetryLineage | null;
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
  coordination?: RunCoordination;
  history_diagnostics?: string[];
};

export type RunCoordination = {
  active_run_id?: string;
  pending?: { run_id: string; pipeline: string } | null;
};

export type Artifact = {
  id?: string;
  path: string;
  kind: string;
  label: string;
  read_path?: string;
  canonical_path?: string;
  source_run_id?: string;
  source_mode?: "run_snapshot" | "promoted_current";
};

export type RunSnapshotIssue = {
  code: string;
  message: string;
  path?: string;
};

export type RunSnapshotResponse = {
  run_id: string;
  status: "available" | "partial" | "not_produced" | "unavailable" | "error";
  artifacts: Artifact[];
  issues: RunSnapshotIssue[];
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
  status: "queued" | "running" | "succeeded" | "failed" | "canceled";
  started_at: string;
  finished_at?: string | null;
  current_step?: string;
  warnings?: string[];
  error_code?: string | null;
  error?: string | null;
  steps: RunReviewStep[];
  progress?: RunProgress | null;
  retry?: RetryLineage | null;
  result?: { state: "completed" | "completed_with_gaps" | "failed" | "canceled"; summary: string; produced: Record<string, number>; partial_scopes: number; failed_scopes: number; promotion: { changed: boolean; current_usable: boolean; baseline_run_id?: string }; recommended_action: string; coverage?: { observed: number; missing: number; status: "available" | "partial" | "unavailable" } };
  recovery?: { category: string; title: string; explanation: string; impact: string; retained_evidence: string; recommended_fix: string; can_retry: boolean; failed_step?: string; failed_scopes?: string[]; technical_code?: string } | null;
  review?: RunReviewContract;
};


export type GitDiffFile = {
  path: string;
  original_path?: string | null;
  folder: string;
  status: "new" | "modified" | "deleted" | "untracked" | "renamed" | "copied" | "changed" | "unchanged";
  additions: number;
  deletions: number;
  binary: boolean;
  index_status: string;
  worktree_status: string;
  old_mode?: string;
  new_mode?: string;
  head_oid?: string;
  index_oid?: string;
  worktree_sha256?: string;
  unavailable: boolean;
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
  state: "clean" | "dirty" | "stale" | "blocked" | "unknown";
  workspace: string;
  scope: "full_workspace";
  branch: string;
  head_oid?: string | null;
  base_ref: string;
  base_oid?: string | null;
  fingerprint: string;
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
  analysis_include: string;
  analysis_exclude: string;
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

export type RuntimeProviderModelConfig = {
  model?: string;
  effort?: string;
};

export type RuntimeProviderModelDraft = {
  model: string;
  effort: string;
};

export type RuntimeProviderModelCapability = {
  model: boolean;
  efforts: string[];
};

export type RuntimeProviderModelEntry = {
  persisted?: RuntimeProviderModelConfig;
  effective?: RuntimeProviderModelConfig;
  source?: {
    model?: string;
    effort?: string;
  };
  capabilities?: RuntimeProviderModelCapability;
};

export type RuntimeProviderModels = Record<string, RuntimeProviderModelEntry>;

export type RuntimeProviderModelsResponse = {
  ok: boolean;
  providers?: RuntimeProviderModels;
};

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

export type OnboardingRecentWorkspace = {
  path: string;
  last_opened_at: string;
  exists: boolean;
};

export type OnboardingPathSuggestion = {
  path: string;
  label: string;
  exists: boolean;
  kind: string;
  source: string;
};

export type OnboardingPathSuggestionsResponse = {
  ok: boolean;
  kind: "workspace" | "repo";
  query: string;
  items: OnboardingPathSuggestion[];
};

export type OnboardingStatusResponse = {
  ok: boolean;
  launcher_mode: boolean;
  console_entered?: boolean;
  can_switch_runtime?: boolean;
  workspace_selected: boolean;
  workspace_ready: boolean;
  workspace: string;
  manifest_present: boolean;
  runtime: OnboardingRuntimeStatus;
  can_enter_console: boolean;
  recent_workspaces?: OnboardingRecentWorkspace[];
};

export type RuntimeProfileResponse = {
  ok: boolean;
  runtime_mode?: "fake" | "headless";
  runtime_provider?: string;
  provider_source?: string;
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
  provider_models?: RuntimeProviderModels;
};

export const runtimeModelProviderOrder = ["claude-code", "qwen-code", "codex-code"] as const;

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
    analysis_include: partial?.analysis_include ?? "",
    analysis_exclude: partial?.analysis_exclude ?? "",
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
