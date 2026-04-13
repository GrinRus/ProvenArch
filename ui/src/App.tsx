import { useEffect, useMemo, useState } from "react";

type Diagnostic = {
  level: "error" | "warning";
  code: string;
  message: string;
  suggestion?: string;
  path?: string;
  repo?: string;
};

type ValidateResponse = {
  ok: boolean;
  workspace: string;
  warnings?: Diagnostic[];
  errors?: Diagnostic[];
  resolved_repos?: Array<{ name: string; source: string; path: string; ref?: string }>;
};

type RunStartResponse = {
  run_id: string;
  status: string;
};

type RunCancelResponse = {
  run_id: string;
  status: "cancel_requested";
};

type RunStatusResponse = {
  run_id: string;
  pipeline: string;
  status: "queued" | "running" | "succeeded" | "failed";
  started_at: string;
  finished_at?: string | null;
  current_step?: string;
  warnings?: string[];
  error_code?: string | null;
  error?: string | null;
};

type RunListItem = {
  run_id: string;
  pipeline: string;
  status: "queued" | "running" | "succeeded" | "failed";
  started_at: string;
  finished_at?: string | null;
  current_step?: string;
  warnings?: string[];
  error_code?: string | null;
  error?: string | null;
};

type RunListResponse = {
  items: RunListItem[];
};

type Artifact = {
  path: string;
  kind: string;
  label: string;
};

type ArtifactsResponse = {
  run_id: string;
  artifacts: Artifact[];
};

type RunLogEntry = {
  cursor: number;
  timestamp: string;
  level: "info" | "warning" | "error";
  step_id?: string;
  domain_id?: string;
  message: string;
  taskrun_path?: string;
  fields?: Record<string, unknown>;
};

type RunLogsResponse = {
  run_id: string;
  items: RunLogEntry[];
  next_cursor: number;
  eof: boolean;
};

type APIErrorPayload = {
  error?:
    | string
    | {
        code?: string;
        message?: string;
      };
};

type RepoSourceMode = "path" | "git_url";

type GuidedRepo = {
  id: string;
  name: string;
  mode: RepoSourceMode;
  path: string;
  git_url: string;
  ref: string;
};

type WizardContract = {
  version: number;
  project_name: string;
  scope: string;
  nfr_priorities: string[];
  rules: string[];
};

type RuntimeTimeoutKey =
  | "step_timeout_sec"
  | "heartbeat_sec"
  | "pipeline_timeout_sec"
  | "pipeline_kill_grace_sec"
  | "api_ready_timeout_sec"
  | "api_init_timeout_sec"
  | "ui_init_poll_timeout_sec"
  | "ui_cancel_poll_timeout_sec";

type RuntimeTimeoutValues = Record<RuntimeTimeoutKey, number>;
type RuntimeTimeoutSources = Record<RuntimeTimeoutKey, string>;

type RuntimeTimeoutsResponse = {
  ok: boolean;
  persisted?: Partial<RuntimeTimeoutValues>;
  effective?: Partial<RuntimeTimeoutValues>;
  source?: Partial<RuntimeTimeoutSources>;
};

type EditableArtifactOption = {
  path: string;
  label: string;
};

const baselineEditorArtifacts: EditableArtifactOption[] = [
  { path: "charter/overview.md", label: "charter/overview.md" },
  { path: "charter/rules.yaml", label: "charter/rules.yaml" },
  { path: "charter/nfr.yaml", label: "charter/nfr.yaml" },
  { path: "charter/glossary.yaml", label: "charter/glossary.yaml" },
  { path: "skills/subagents.yaml", label: "skills/subagents.yaml" },
  { path: "skills/prompt-packs/constitution.md", label: "skills/prompt-packs/constitution.md" },
  { path: "skills/prompt-packs/collect-context.md", label: "skills/prompt-packs/collect-context.md" },
  { path: "skills/prompt-packs/findings.md", label: "skills/prompt-packs/findings.md" },
  { path: "skills/prompt-packs/proposals.md", label: "skills/prompt-packs/proposals.md" },
  { path: "skills/prompt-packs/qa.md", label: "skills/prompt-packs/qa.md" },
  { path: "skills/service-inventory/prompts/system.md", label: "skills/service-inventory/prompts/system.md" },
  { path: "skills/service-inventory/prompts/task.md", label: "skills/service-inventory/prompts/task.md" },
  { path: "skills/interface-extraction/prompts/system.md", label: "skills/interface-extraction/prompts/system.md" },
  { path: "skills/interface-extraction/prompts/task.md", label: "skills/interface-extraction/prompts/task.md" },
  { path: "skills/integration-mapping/prompts/system.md", label: "skills/integration-mapping/prompts/system.md" },
  { path: "skills/integration-mapping/prompts/task.md", label: "skills/integration-mapping/prompts/task.md" },
  { path: "skills/datastore-mapping/prompts/system.md", label: "skills/datastore-mapping/prompts/system.md" },
  { path: "skills/datastore-mapping/prompts/task.md", label: "skills/datastore-mapping/prompts/task.md" },
  { path: "skills/cicd-mapping/prompts/system.md", label: "skills/cicd-mapping/prompts/system.md" },
  { path: "skills/cicd-mapping/prompts/task.md", label: "skills/cicd-mapping/prompts/task.md" },
  { path: "skills/ownership-coverage/prompts/system.md", label: "skills/ownership-coverage/prompts/system.md" },
  { path: "skills/ownership-coverage/prompts/task.md", label: "skills/ownership-coverage/prompts/task.md" },
  { path: "skills/findings/prompts/system.md", label: "skills/findings/prompts/system.md" },
  { path: "skills/findings/prompts/task.md", label: "skills/findings/prompts/task.md" },
  { path: "skills/proposals/prompts/system.md", label: "skills/proposals/prompts/system.md" },
  { path: "skills/proposals/prompts/task.md", label: "skills/proposals/prompts/task.md" },
  { path: "skills/qa/prompts/system.md", label: "skills/qa/prompts/system.md" },
  { path: "skills/qa/prompts/task.md", label: "skills/qa/prompts/task.md" }
];

const runtimeTimeoutKeys: RuntimeTimeoutKey[] = [
  "step_timeout_sec",
  "heartbeat_sec",
  "pipeline_timeout_sec",
  "pipeline_kill_grace_sec",
  "api_ready_timeout_sec",
  "api_init_timeout_sec",
  "ui_init_poll_timeout_sec",
  "ui_cancel_poll_timeout_sec",
];

const defaultRuntimeTimeoutValues: RuntimeTimeoutValues = {
  step_timeout_sec: 1800,
  heartbeat_sec: 30,
  pipeline_timeout_sec: 2400,
  pipeline_kill_grace_sec: 30,
  api_ready_timeout_sec: 60,
  api_init_timeout_sec: 120,
  ui_init_poll_timeout_sec: 900,
  ui_cancel_poll_timeout_sec: 420,
};

const runtimeTimeoutLabels: Record<RuntimeTimeoutKey, string> = {
  step_timeout_sec: "runtime.timeouts.step_timeout_sec",
  heartbeat_sec: "runtime.timeouts.heartbeat_sec",
  pipeline_timeout_sec: "runtime.timeouts.pipeline_timeout_sec",
  pipeline_kill_grace_sec: "runtime.timeouts.pipeline_kill_grace_sec",
  api_ready_timeout_sec: "runtime.timeouts.api_ready_timeout_sec",
  api_init_timeout_sec: "runtime.timeouts.api_init_timeout_sec",
  ui_init_poll_timeout_sec: "runtime.timeouts.ui_init_poll_timeout_sec",
  ui_cancel_poll_timeout_sec: "runtime.timeouts.ui_cancel_poll_timeout_sec",
};

const finalStatuses = new Set(["succeeded", "failed"]);
const activeStatuses = new Set(["queued", "running"]);
const runLogsPageLimit = 200;
let guidedRepoSeed = 0;

function makeGuidedRepo(partial?: Partial<GuidedRepo>): GuidedRepo {
  guidedRepoSeed += 1;
  return {
    id: partial?.id ?? `repo-${guidedRepoSeed}`,
    name: partial?.name ?? `repo-${guidedRepoSeed}`,
    mode: partial?.mode ?? "path",
    path: partial?.path ?? "/absolute/path/to/repository",
    git_url: partial?.git_url ?? "https://gitlab.example.com/group/repository.git",
    ref: partial?.ref ?? "",
  };
}

function getErrorMessage(payload: unknown, fallback: string): string {
  const typed = payload as APIErrorPayload;
  if (!typed || typeof typed !== "object") {
    return fallback;
  }
  if (typeof typed.error === "string") {
    return typed.error;
  }
  if (typed.error && typeof typed.error.message === "string") {
    return typed.error.message;
  }
  return fallback;
}

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init);
  const payload = (await response.json()) as T;
  if (!response.ok) {
    throw new Error(getErrorMessage(payload, `request failed: ${url}`));
  }
  return payload;
}

function splitListInput(input: string): string[] {
  return input
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}

function formatTimestamp(value?: string | null): string {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toISOString().replace("T", " ").replace(".000Z", " UTC");
}

function normalizeRuntimeTimeoutValues(
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

function runtimeTimeoutDraftFromValues(values: RuntimeTimeoutValues): Record<RuntimeTimeoutKey, string> {
  const draft = {} as Record<RuntimeTimeoutKey, string>;
  for (const key of runtimeTimeoutKeys) {
    draft[key] = String(values[key]);
  }
  return draft;
}

function parseRuntimeTimeoutPatch(draft: Record<RuntimeTimeoutKey, string>): RuntimeTimeoutValues {
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

function parseTimeOrMin(value?: string | null): number {
  if (!value) {
    return Number.NEGATIVE_INFINITY;
  }
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return Number.NEGATIVE_INFINITY;
  }
  return parsed;
}

function pickBootstrapRun(items: RunListItem[]): RunListItem | null {
  if (!Array.isArray(items) || items.length === 0) {
    return null;
  }
  let newestActive: RunListItem | null = null;
  let newestActiveStartedAt = Number.NEGATIVE_INFINITY;
  for (const item of items) {
    if (!activeStatuses.has(item.status)) {
      continue;
    }
    const startedAt = parseTimeOrMin(item.started_at);
    if (newestActive === null || startedAt > newestActiveStartedAt) {
      newestActive = item;
      newestActiveStartedAt = startedAt;
    }
  }
  if (newestActive) {
    return newestActive;
  }
  return items[0];
}

export default function App() {
  const [validateResult, setValidateResult] = useState<ValidateResponse | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [manifestContent, setManifestContent] = useState("");
  const [selectedEditorPath, setSelectedEditorPath] = useState<string>(baselineEditorArtifacts[0].path);
  const [selectedEditorContent, setSelectedEditorContent] = useState("");
  const [editorStatus, setEditorStatus] = useState("");

  const [guidedRepos, setGuidedRepos] = useState<GuidedRepo[]>(() => [
    makeGuidedRepo({
      name: "payments-service",
      mode: "path",
      path: "/absolute/path/to/payments-service"
    })
  ]);
  const [guidedDocsImportsPath, setGuidedDocsImportsPath] = useState("./docs/imports");

  const [wizardProjectName, setWizardProjectName] = useState("ProvenArch MVP");
  const [wizardScope, setWizardScope] = useState("payments, users, ci-cd");
  const [wizardNfr, setWizardNfr] = useState("availability, traceability");
  const [wizardRules, setWizardRules] = useState("no silent re-key, evidence-first findings");
  const [wizardStatus, setWizardStatus] = useState("");

  const [runtimeTimeoutPersisted, setRuntimeTimeoutPersisted] = useState<Partial<RuntimeTimeoutValues>>({});
  const [runtimeTimeoutEffective, setRuntimeTimeoutEffective] = useState<RuntimeTimeoutValues>(defaultRuntimeTimeoutValues);
  const [runtimeTimeoutSource, setRuntimeTimeoutSource] = useState<Partial<RuntimeTimeoutSources>>({});
  const [runtimeTimeoutDraft, setRuntimeTimeoutDraft] = useState<Record<RuntimeTimeoutKey, string>>(
    runtimeTimeoutDraftFromValues(defaultRuntimeTimeoutValues)
  );
  const [runtimeTimeoutStatus, setRuntimeTimeoutStatus] = useState("");

  const [runId, setRunID] = useState<string | null>(null);
  const [runStatus, setRunStatus] = useState<RunStatusResponse | null>(null);
  const [runList, setRunList] = useState<RunListItem[]>([]);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [selectedArtifact, setSelectedArtifact] = useState<string>("");
  const [selectedArtifactContent, setSelectedArtifactContent] = useState<string>("");
  const [runLogs, setRunLogs] = useState<RunLogEntry[]>([]);
  const [runLogsCursor, setRunLogsCursor] = useState(0);
  const [runLogsEOF, setRunLogsEOF] = useState(false);
  const [runLogsStatus, setRunLogsStatus] = useState("");
  const [runLogsViewMode, setRunLogsViewMode] = useState<"line" | "line+fields">("line");
  const [runActionStatus, setRunActionStatus] = useState("");
  const [cancelBusy, setCancelBusy] = useState(false);

  const [coverageSummary, setCoverageSummary] = useState<string>("");
  const [openQuestions, setOpenQuestions] = useState<string>("");

  const [gitMessage, setGitMessage] = useState("chore: update ACP workspace artifacts");
  const [proposalBranch, setProposalBranch] = useState("proposal/beta-refresh");
  const [gitStatus, setGitStatus] = useState<string>("");

  const hasActiveRuns = useMemo(() => runList.some((run) => activeStatuses.has(run.status)), [runList]);
  const runCounters = useMemo(
    () =>
      runList.reduce(
        (acc, run) => {
          if (run.status === "running" || run.status === "queued") {
            acc.running += 1;
          } else if (run.status === "succeeded") {
            acc.succeeded += 1;
          } else if (run.status === "failed") {
            acc.failed += 1;
          }
          return acc;
        },
        { running: 0, succeeded: 0, failed: 0 }
      ),
    [runList]
  );
  const validationDiagnosticsByRepo = useMemo(() => {
    if (!validateResult) {
      return [];
    }
    const grouped = new Map<string, Diagnostic[]>();
    const diagnostics = [...(validateResult.warnings ?? []), ...(validateResult.errors ?? [])];
    for (const diagnostic of diagnostics) {
      const key = diagnostic.repo?.trim() ? diagnostic.repo : "__workspace__";
      const existing = grouped.get(key) ?? [];
      existing.push(diagnostic);
      grouped.set(key, existing);
    }
    return Array.from(grouped.entries()).sort((left, right) => left[0].localeCompare(right[0]));
  }, [validateResult]);
  const runLogTaskrunPaths = useMemo(() => {
    const paths = new Set<string>();
    for (const entry of runLogs) {
      if (entry.taskrun_path && entry.taskrun_path.trim().length > 0) {
        paths.add(entry.taskrun_path.trim());
      }
    }
    return Array.from(paths).sort((left, right) => left.localeCompare(right));
  }, [runLogs]);
  const selectedRunListItem = useMemo(() => {
    if (!runId) {
      return null;
    }
    return runList.find((item) => item.run_id === runId) ?? null;
  }, [runId, runList]);
  const selectedRunWarnings = useMemo(() => {
    if (runStatus && runId && runStatus.run_id === runId) {
      return runStatus.warnings ?? [];
    }
    return selectedRunListItem?.warnings ?? [];
  }, [runId, runStatus, selectedRunListItem]);
  const selectedRunIsActive = useMemo(() => {
    if (runStatus && runId && runStatus.run_id === runId) {
      return activeStatuses.has(runStatus.status);
    }
    if (selectedRunListItem) {
      return activeStatuses.has(selectedRunListItem.status);
    }
    return false;
  }, [runId, runStatus, selectedRunListItem]);
  const shouldPollRunDetails = useMemo(() => {
    return hasActiveRuns || selectedRunIsActive || (runId !== null && !runLogsEOF);
  }, [hasActiveRuns, selectedRunIsActive, runId, runLogsEOF]);
  const runLogsRendered = useMemo(() => {
    const includeFields = runLogsViewMode === "line+fields";
    return runLogs
      .map((entry) => {
        const line = formatRunLogLine(entry);
        if (!includeFields) {
          return line;
        }
        if (!entry.fields || Object.keys(entry.fields).length === 0) {
          return line;
        }
        const serialized = JSON.stringify(entry.fields, null, 2);
        if (!serialized) {
          return line;
        }
        return `${line}\n${serialized}`;
      })
      .join(includeFields ? "\n\n" : "\n");
  }, [runLogs, runLogsViewMode]);

  useEffect(() => {
    void bootstrapEditorData();
  }, []);

  useEffect(() => {
    if (!shouldPollRunDetails) {
      return;
    }

    const interval = setInterval(() => {
      void pollRunUpdates();
    }, 1000);

    return () => clearInterval(interval);
  }, [shouldPollRunDetails, runId, runLogsCursor, runLogsEOF]);

  async function bootstrapEditorData() {
    let initialRuns: RunListItem[] = [];
    try {
      initialRuns = await loadRunList(100);
    } catch {
      setRunList([]);
    }

    const bootstrapRun = pickBootstrapRun(initialRuns);
    if (bootstrapRun) {
      await handleSelectRun(bootstrapRun.run_id, { silentErrors: true });
    }

    try {
      const manifest = await fetchJSON<{ content: string }>("/api/workspace/manifest");
      setManifestContent(manifest.content ?? "");
    } catch {
      setManifestContent("");
    }

    await loadTextArtifact(selectedEditorPath, setSelectedEditorContent);
    await loadWizardContract();
    await loadRuntimeTimeouts();
  }

  async function loadRunList(limit = 100): Promise<RunListItem[]> {
    const payload = await fetchJSON<RunListResponse>(`/api/pipeline/runs?limit=${limit}`);
    const items = payload.items ?? [];
    setRunList(items);
    return items;
  }

  function resetRunLogs() {
    setRunLogs([]);
    setRunLogsCursor(0);
    setRunLogsEOF(false);
    setRunLogsStatus("");
  }

  function mergeRunLogsPayload(payload: RunLogsResponse, reset: boolean, fallbackCursor: number) {
    setRunLogs((previous) => {
      const seed = reset ? [] : previous;
      const merged = [...seed];
      const seen = new Set(merged.map((entry) => entry.cursor));
      for (const entry of payload.items ?? []) {
        if (seen.has(entry.cursor)) {
          continue;
        }
        merged.push(entry);
        seen.add(entry.cursor);
      }
      merged.sort((left, right) => left.cursor - right.cursor);
      return merged;
    });
    setRunLogsCursor(payload.next_cursor ?? fallbackCursor);
    setRunLogsEOF(Boolean(payload.eof));
  }

  async function fetchRunLogs(id: string, reset = false): Promise<RunLogsResponse | null> {
    if (!id) {
      return null;
    }
    const cursor = reset ? 0 : runLogsCursor;
    if (!reset && runLogsEOF) {
      return null;
    }
    const payload = await fetchJSON<RunLogsResponse>(
      `/api/pipeline/runs/${id}/logs?cursor=${cursor}&limit=${runLogsPageLimit}`
    );
    mergeRunLogsPayload(payload, reset, cursor);
    return payload;
  }

  async function fetchRunLogsUntilEOF(id: string) {
    if (!id) {
      return;
    }
    let cursor = 0;
    let reset = true;
    for (let page = 0; page < 25; page += 1) {
      const payload = await fetchJSON<RunLogsResponse>(
        `/api/pipeline/runs/${id}/logs?cursor=${cursor}&limit=${runLogsPageLimit}`
      );
      mergeRunLogsPayload(payload, reset, cursor);
      if (payload.eof) {
        return;
      }
      const nextCursor = payload.next_cursor ?? cursor;
      if (nextCursor <= cursor) {
        return;
      }
      cursor = nextCursor;
      reset = false;
    }
  }

  async function pollRunUpdates() {
    try {
      const latestRuns = await loadRunList(100);
      if (runId) {
        const selectedRunExists = latestRuns.some((item) => item.run_id === runId);
        if (!selectedRunExists) {
          setRunStatus((previous) => {
            if (previous && previous.run_id === runId) {
              return null;
            }
            return previous;
          });
          setArtifacts([]);
          resetRunLogs();
          return;
        }
        const previousStatus = runStatus?.status ?? null;
        const status = await fetchRunStatus(runId);
        if (activeStatuses.has(status.status)) {
          await fetchRunLogs(runId, false);
          return;
        }
        const statusChanged = previousStatus !== status.status;
        if (statusChanged || !runLogsEOF) {
          await fetchRunLogsUntilEOF(runId);
        }
      }
    } catch {
      // keep UI responsive even if polling fails temporarily
    }
  }

  async function loadWizardContract() {
    try {
      const response = await fetch("/api/artifacts?path=charter/wizard/step0-contract.json");
      if (!response.ok) {
        return;
      }
      const content = (await response.text()).trim();
      if (!content) {
        return;
      }
      const parsed = JSON.parse(content) as Partial<WizardContract>;
      if (typeof parsed.project_name === "string") {
        setWizardProjectName(parsed.project_name);
      }
      if (typeof parsed.scope === "string") {
        setWizardScope(parsed.scope);
      }
      if (Array.isArray(parsed.nfr_priorities)) {
        setWizardNfr(parsed.nfr_priorities.join(", "));
      }
      if (Array.isArray(parsed.rules)) {
        setWizardRules(parsed.rules.join(", "));
      }
    } catch {
      // no-op: wizard contract is optional during bootstrap
    }
  }

  async function loadRuntimeTimeouts() {
    try {
      const payload = await fetchJSON<RuntimeTimeoutsResponse>("/api/runtime/timeouts");
      const nextEffective = normalizeRuntimeTimeoutValues(payload.effective, defaultRuntimeTimeoutValues);
      const nextPersisted = payload.persisted ?? {};
      const nextSource = payload.source ?? {};
      setRuntimeTimeoutPersisted(nextPersisted);
      setRuntimeTimeoutEffective(nextEffective);
      setRuntimeTimeoutSource(nextSource);
      setRuntimeTimeoutDraft(runtimeTimeoutDraftFromValues(nextEffective));
    } catch {
      setRuntimeTimeoutPersisted({});
      setRuntimeTimeoutEffective(defaultRuntimeTimeoutValues);
      setRuntimeTimeoutSource({});
      setRuntimeTimeoutDraft(runtimeTimeoutDraftFromValues(defaultRuntimeTimeoutValues));
    }
  }

  function updateRuntimeTimeoutDraft(key: RuntimeTimeoutKey, value: string) {
    setRuntimeTimeoutDraft((previous) => ({ ...previous, [key]: value }));
  }

  async function handleSaveRuntimeTimeouts() {
    setBusy(true);
    setError(null);
    setRuntimeTimeoutStatus("");
    try {
      const patch = parseRuntimeTimeoutPatch(runtimeTimeoutDraft);
      await fetchJSON<RuntimeTimeoutsResponse>("/api/runtime/timeouts", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ timeouts: patch }),
      });
      await loadRuntimeTimeouts();
      setRuntimeTimeoutStatus("Runtime timeouts saved");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save runtime timeouts");
    } finally {
      setBusy(false);
    }
  }

  async function handleResetRuntimeTimeouts() {
    setBusy(true);
    setError(null);
    setRuntimeTimeoutStatus("");
    try {
      await fetchJSON<RuntimeTimeoutsResponse>("/api/runtime/timeouts", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ timeouts: defaultRuntimeTimeoutValues }),
      });
      await loadRuntimeTimeouts();
      setRuntimeTimeoutStatus("Runtime timeouts reset to balanced defaults");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to reset runtime timeouts");
    } finally {
      setBusy(false);
    }
  }

  async function loadTextArtifact(path: string, setter: (value: string) => void) {
    try {
      const response = await fetch(`/api/artifacts?path=${encodeURIComponent(path)}`);
      if (!response.ok) {
        setter("");
        return;
      }
      setter(await response.text());
    } catch {
      setter("");
    }
  }

  async function handleValidateWorkspace() {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch("/api/workspace/validate", {
        method: "POST",
        headers: { "Content-Type": "application/json" }
      });
      const payload = (await response.json()) as ValidateResponse & APIErrorPayload;
      setValidateResult(payload);
      if (!response.ok) {
        throw new Error(getErrorMessage(payload, "workspace validation failed"));
      }
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "workspace validation failed");
    } finally {
      setBusy(false);
    }
  }

  function updateGuidedRepo(id: string, patch: Partial<GuidedRepo>) {
    setGuidedRepos((previous) => previous.map((repo) => (repo.id === id ? { ...repo, ...patch } : repo)));
  }

  function handleAddGuidedRepo() {
    setGuidedRepos((previous) => [...previous, makeGuidedRepo()]);
  }

  function handleRemoveGuidedRepo(id: string) {
    setGuidedRepos((previous) => {
      if (previous.length <= 1) {
        return previous;
      }
      return previous.filter((repo) => repo.id !== id);
    });
  }

  function buildManifestFromGuidedForm(): string {
    const importsPath = guidedDocsImportsPath.trim() || "./docs/imports";
    const names = new Set<string>();
    const lines = ["version: 1", "repos:"];

    if (guidedRepos.length === 0) {
      throw new Error("at least one repo entry is required");
    }

    for (const repo of guidedRepos) {
      const name = repo.name.trim();
      const pathValue = repo.path.trim();
      const gitURLValue = repo.git_url.trim();
      const refValue = repo.ref.trim();

      if (!name) {
        throw new Error("repo name is required for every entry");
      }
      if (names.has(name)) {
        throw new Error(`duplicate repo name "${name}" in guided setup`);
      }
      names.add(name);

      if (repo.mode === "path" && !pathValue) {
        throw new Error(`repo "${name}" with path source requires non-empty path`);
      }
      if (repo.mode === "git_url" && !gitURLValue) {
        throw new Error(`repo "${name}" with git_url source requires repository URL`);
      }

      lines.push(`  - name: ${name}`);
      if (repo.mode === "path") {
        lines.push(`    path: ${pathValue}`);
      } else {
        lines.push(`    git_url: ${gitURLValue}`);
      }
      if (refValue) {
        lines.push(`    ref: ${refValue}`);
      }
    }

    lines.push("docs:");
    lines.push(`  imports_path: ${importsPath}`);
    return `${lines.join("\n")}\n`;
  }

  function handleApplyGuidedWorkspaceSetup() {
    setError(null);
    try {
      setManifestContent(buildManifestFromGuidedForm());
    } catch (buildError) {
      setError(buildError instanceof Error ? buildError.message : "failed to apply guided setup");
    }
  }

  async function handleSaveManifest() {
    setBusy(true);
    setError(null);
    try {
      await fetchJSON<{ ok: boolean }>("/api/workspace/manifest", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ content: manifestContent })
      });
      await handleValidateWorkspace();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save manifest");
    } finally {
      setBusy(false);
    }
  }

  async function saveEditableArtifact(path: string, content: string) {
    await fetchJSON<{ ok: boolean }>("/api/artifacts/write", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path, content })
    });
  }

  async function handleSaveStep0WizardContract() {
    setBusy(true);
    setError(null);
    setWizardStatus("");

    const projectName = wizardProjectName.trim();
    const scope = wizardScope.trim();
    if (!projectName || !scope) {
      setBusy(false);
      setError("step0 wizard contract requires project name and scope");
      return;
    }

    const payload: WizardContract = {
      version: 1,
      project_name: projectName,
      scope,
      nfr_priorities: splitListInput(wizardNfr),
      rules: splitListInput(wizardRules)
    };

    try {
      await saveEditableArtifact("charter/wizard/step0-contract.json", `${JSON.stringify(payload, null, 2)}\n`);
      setWizardStatus("Saved charter/wizard/step0-contract.json");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save step0 wizard contract");
    } finally {
      setBusy(false);
    }
  }

  async function handleEditorSelectionChange(path: string) {
    setSelectedEditorPath(path);
    await loadTextArtifact(path, setSelectedEditorContent);
  }

  async function handleSaveSelectedEditorArtifact() {
    setBusy(true);
    setError(null);
    setEditorStatus("");
    try {
      await saveEditableArtifact(selectedEditorPath, selectedEditorContent);
      setEditorStatus(`Saved ${selectedEditorPath}`);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save editor artifact");
    } finally {
      setBusy(false);
    }
  }

  async function handleRunPipeline(pipeline: "init" | "refresh") {
    setBusy(true);
    setError(null);
    setRunActionStatus("");
    setArtifacts([]);
    setSelectedArtifact("");
    setSelectedArtifactContent("");
    resetRunLogs();
    try {
      const payload = await fetchJSON<RunStartResponse>(`/api/pipeline/${pipeline}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ trigger: "ui", commit: false, create_proposal_branch: false })
      });
      setRunList((previous) => [
        {
          run_id: payload.run_id,
          pipeline,
          status: "queued",
          started_at: new Date().toISOString(),
          finished_at: null,
          warnings: [],
          error_code: null,
          error: null
        },
        ...previous.filter((run) => run.run_id !== payload.run_id)
      ]);
      setRunID(payload.run_id);
      const status = await fetchRunStatus(payload.run_id);
      await fetchRunLogs(payload.run_id, true);
      if (finalStatuses.has(status.status)) {
        await fetchRunLogsUntilEOF(payload.run_id);
      }
      await loadRunList(100);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to start pipeline");
    } finally {
      setBusy(false);
    }
  }

  async function fetchRunStatus(id: string): Promise<RunStatusResponse> {
    const payload = await fetchJSON<RunStatusResponse>(`/api/pipeline/runs/${id}`);
    setRunStatus(payload);
    if (finalStatuses.has(payload.status)) {
      await fetchArtifacts(id);
      await loadCoverageArtifacts();
    }
    return payload;
  }

  async function handleSelectRun(
    id: string,
    options?: {
      silentErrors?: boolean;
    }
  ) {
    try {
      setRunActionStatus("");
      setRunID(id);
      resetRunLogs();
      const status = await fetchRunStatus(id);
      await fetchArtifacts(id);
      await fetchRunLogs(id, true);
      if (finalStatuses.has(status.status)) {
        await fetchRunLogsUntilEOF(id);
      }
    } catch (requestError) {
      if (!options?.silentErrors) {
        setError(requestError instanceof Error ? requestError.message : "failed to load run details");
      }
    }
  }

  async function handleCancelSelectedRun() {
    if (!runId || !selectedRunIsActive) {
      return;
    }

    setCancelBusy(true);
    setError(null);
    setRunActionStatus("");
    try {
      const response = await fetch(`/api/pipeline/runs/${runId}/cancel`, {
        method: "POST"
      });
      const payload = (await response.json()) as RunCancelResponse & APIErrorPayload;

      if (response.status === 202) {
        setRunActionStatus(`Cancel requested for ${runId}`);
        await loadRunList(100);
        const status = await fetchRunStatus(runId);
        if (finalStatuses.has(status.status)) {
          await fetchRunLogsUntilEOF(runId);
        } else {
          await fetchRunLogs(runId, false);
        }
        return;
      }

      if (response.status === 404) {
        setRunActionStatus("Selected run no longer exists.");
        const latestRuns = await loadRunList(100);
        const selectedRunExists = latestRuns.some((item) => item.run_id === runId);
        if (!selectedRunExists) {
          setRunStatus((previous) => {
            if (previous && previous.run_id === runId) {
              return null;
            }
            return previous;
          });
          setArtifacts([]);
          resetRunLogs();
        }
        return;
      }

      if (response.status === 409) {
        setRunActionStatus("Selected run is already terminal.");
        await loadRunList(100);
        const status = await fetchRunStatus(runId);
        if (finalStatuses.has(status.status)) {
          await fetchRunLogsUntilEOF(runId);
        }
        return;
      }

      throw new Error(getErrorMessage(payload, "failed to cancel selected run"));
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to cancel selected run");
    } finally {
      setCancelBusy(false);
    }
  }

  async function fetchArtifacts(id: string) {
    const payload = await fetchJSON<ArtifactsResponse>(`/api/pipeline/runs/${id}/artifacts`);
    setArtifacts(payload.artifacts ?? []);
  }

  async function loadCoverageArtifacts() {
    await loadTextArtifact("reports/coverage/summary.md", setCoverageSummary);
    await loadTextArtifact("reports/coverage/open-questions.md", setOpenQuestions);
  }

  async function handleOpenArtifact(path: string) {
    setSelectedArtifact(path);
    setSelectedArtifactContent("Loading...");
    await loadTextArtifact(path, setSelectedArtifactContent);
  }

  function formatRunLogLine(entry: RunLogEntry): string {
    const parts = [
      entry.timestamp,
      entry.level.toUpperCase(),
      entry.step_id ? `[${entry.step_id}]` : "",
      entry.domain_id ? `(${entry.domain_id})` : "",
      entry.message
    ].filter((value) => value && value.trim().length > 0);
    return parts.join(" ");
  }

  async function handleCopyRunLogs() {
    if (runLogs.length === 0) {
      return;
    }
    const text = runLogs.map((entry) => formatRunLogLine(entry)).join("\n");
    if (!navigator.clipboard || !navigator.clipboard.writeText) {
      setRunLogsStatus("Clipboard API is not available in this browser context.");
      return;
    }

    try {
      await navigator.clipboard.writeText(text);
      setRunLogsStatus("Run logs copied to clipboard");
    } catch (requestError) {
      setRunLogsStatus(
        requestError instanceof Error ? requestError.message : "Run logs copy failed"
      );
    }
  }

  function handleDownloadRunLogs() {
    if (runLogs.length === 0 || !runId) {
      return;
    }
    const text = runLogs.map((entry) => formatRunLogLine(entry)).join("\n");
    const blob = new Blob([text], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${runId}.logs.txt`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    setRunLogsStatus(`Downloaded ${runId}.logs.txt`);
  }

  async function handleGitCommit() {
    setBusy(true);
    setError(null);
    setGitStatus("");
    try {
      const payload = await fetchJSON<{ status: string; message?: string; output?: string }>("/api/git/commit", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message: gitMessage })
      });
      setGitStatus(payload.output ?? payload.message ?? payload.status);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "git commit failed");
    } finally {
      setBusy(false);
    }
  }

  async function handleCreateProposalBranch() {
    setBusy(true);
    setError(null);
    setGitStatus("");
    try {
      const payload = await fetchJSON<{ branch: string }>("/api/git/proposal-branch", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: proposalBranch })
      });
      setGitStatus(`checked out ${payload.branch}`);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to create proposal branch");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="shell">
      <section className="hero">
        <p className="eyebrow">ACP Beta Surface</p>
        <h1>Local-first architecture control plane</h1>
        <p className="lead">
          Validate workspace, edit charter/skills baseline bundle, run init/refresh pipeline, inspect artifacts and
          coverage, then commit workspace updates.
        </p>
      </section>

      <section className="panel" data-testid="workspace-panel">
        <h2>Setup: Workspace</h2>
        <p className="hint">Guided setup writes a valid multi-repo `workspace.yaml` draft.</p>
        {guidedRepos.map((repo, index) => (
          <div className="repo-card" key={repo.id}>
            <div className="repo-card-head">
              <h3>Repo {index + 1}</h3>
              <button type="button" className="inline-danger" onClick={() => handleRemoveGuidedRepo(repo.id)} disabled={busy || guidedRepos.length <= 1}>
                Remove
              </button>
            </div>

            <label htmlFor={`guidedRepoName-${repo.id}`}>Repo name</label>
            <input
              id={`guidedRepoName-${repo.id}`}
              value={repo.name}
              onChange={(event) => updateGuidedRepo(repo.id, { name: event.target.value })}
            />

            <label htmlFor={`guidedRepoMode-${repo.id}`}>Repo source type</label>
            <select
              id={`guidedRepoMode-${repo.id}`}
              value={repo.mode}
              onChange={(event) => updateGuidedRepo(repo.id, { mode: event.target.value as RepoSourceMode })}
            >
              <option value="path">path</option>
              <option value="git_url">git_url</option>
            </select>

            {repo.mode === "path" ? (
              <>
                <label htmlFor={`guidedRepoPath-${repo.id}`}>path</label>
                <input
                  id={`guidedRepoPath-${repo.id}`}
                  value={repo.path}
                  onChange={(event) => updateGuidedRepo(repo.id, { path: event.target.value })}
                />
              </>
            ) : (
              <>
                <label htmlFor={`guidedRepoGitURL-${repo.id}`}>git_url</label>
                <input
                  id={`guidedRepoGitURL-${repo.id}`}
                  value={repo.git_url}
                  onChange={(event) => updateGuidedRepo(repo.id, { git_url: event.target.value })}
                />
              </>
            )}

            <label htmlFor={`guidedRepoRef-${repo.id}`}>ref (optional)</label>
            <input
              id={`guidedRepoRef-${repo.id}`}
              value={repo.ref}
              onChange={(event) => updateGuidedRepo(repo.id, { ref: event.target.value })}
              placeholder="Leave empty to use current checkout"
            />
          </div>
        ))}

        <label htmlFor="guidedDocsImportsPath">docs.imports_path</label>
        <input id="guidedDocsImportsPath" value={guidedDocsImportsPath} onChange={(event) => setGuidedDocsImportsPath(event.target.value)} />

        <div className="actions">
          <button type="button" onClick={handleAddGuidedRepo} disabled={busy}>
            Add repo
          </button>
          <button type="button" onClick={handleApplyGuidedWorkspaceSetup} disabled={busy}>
            Apply guided workspace form
          </button>
        </div>

        <p className="hint">`workspace.yaml` editor (path/git_url sources)</p>
        <textarea value={manifestContent} onChange={(event) => setManifestContent(event.target.value)} rows={12} />
        <div className="actions">
          <button type="button" onClick={() => void handleSaveManifest()} disabled={busy} data-testid="workspace-save-btn">
            Save workspace.yaml
          </button>
          <button
            type="button"
            onClick={() => void handleValidateWorkspace()}
            disabled={busy}
            data-testid="workspace-validate-btn"
          >
            Validate workspace
          </button>
        </div>
        {validateResult ? (
          <div className="status-block" data-testid="workspace-validate-result">
            <p>
              Workspace: <code>{validateResult.workspace}</code>
            </p>
            <p>Status: {validateResult.ok ? "valid" : "invalid"}</p>

            {(validateResult.resolved_repos ?? []).length > 0 ? (
              <div className="repo-summary">
                <p className="hint">Resolved repos</p>
                <ul>
                  {(validateResult.resolved_repos ?? []).map((repo) => (
                    <li key={`resolved-${repo.name}-${repo.path}`}>
                      <code>{repo.name}</code> ({repo.source}) {repo.path}
                      {repo.ref ? ` @ ${repo.ref}` : ""}
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}

            {validationDiagnosticsByRepo.map(([repoKey, diagnostics]) => (
              <div key={`diag-group-${repoKey}`} className="repo-summary">
                <p className="hint">{repoKey === "__workspace__" ? "Workspace diagnostics" : `Diagnostics for ${repoKey}`}</p>
                {diagnostics.map((diagnostic, index) => (
                  <p className={diagnostic.level === "error" ? "status err" : "status warn"} key={`${repoKey}-${diagnostic.code}-${diagnostic.message}-${index}`}>
                    {diagnostic.level === "error" ? "Error" : "Warning"} [{diagnostic.code}]: {diagnostic.message}
                  </p>
                ))}
              </div>
            ))}
          </div>
        ) : null}
      </section>

      <section className="panel" data-testid="runtime-timeouts-panel">
        <h2>Setup: Runtime Timeouts</h2>
        <p className="hint">Persisted in `workspace.yaml` (`runtime.timeouts`) with precedence `env &gt; workspace &gt; defaults`.</p>
        <div className="actions">
          <button type="button" onClick={() => void loadRuntimeTimeouts()} disabled={busy}>
            Reload runtime timeouts
          </button>
          <button type="button" onClick={() => void handleSaveRuntimeTimeouts()} disabled={busy} data-testid="runtime-timeouts-save-btn">
            Save runtime timeouts
          </button>
          <button type="button" onClick={() => void handleResetRuntimeTimeouts()} disabled={busy}>
            Reset balanced defaults
          </button>
        </div>
        {runtimeTimeoutKeys.map((key) => (
          <div key={`timeout-${key}`}>
            <label htmlFor={`runtime-timeout-${key}`}>{runtimeTimeoutLabels[key]}</label>
            <input
              id={`runtime-timeout-${key}`}
              data-testid={`runtime-timeout-input-${key}`}
              value={runtimeTimeoutDraft[key]}
              onChange={(event) => updateRuntimeTimeoutDraft(key, event.target.value)}
            />
            <p className="hint">
              persisted: {runtimeTimeoutPersisted[key] ?? "-"} | effective: {runtimeTimeoutEffective[key]} | source: {runtimeTimeoutSource[key] ?? "default"}
            </p>
          </div>
        ))}
        {runtimeTimeoutStatus ? <p className="status ok">{runtimeTimeoutStatus}</p> : null}
      </section>

      <section className="panel">
        <h2>Setup: Step 0 Wizard Contract</h2>
        <p className="hint">Structured contract persisted as `charter/wizard/step0-contract.json`.</p>

        <label htmlFor="wizardProjectName">Project name</label>
        <input id="wizardProjectName" value={wizardProjectName} onChange={(event) => setWizardProjectName(event.target.value)} />

        <label htmlFor="wizardScope">Scope</label>
        <textarea id="wizardScope" value={wizardScope} onChange={(event) => setWizardScope(event.target.value)} rows={3} />

        <label htmlFor="wizardNfr">NFR priorities (comma/newline)</label>
        <textarea id="wizardNfr" value={wizardNfr} onChange={(event) => setWizardNfr(event.target.value)} rows={3} />

        <label htmlFor="wizardRules">Rules (comma/newline)</label>
        <textarea id="wizardRules" value={wizardRules} onChange={(event) => setWizardRules(event.target.value)} rows={3} />

        <button type="button" onClick={() => void handleSaveStep0WizardContract()} disabled={busy}>
          Save Step 0 wizard contract
        </button>

        {wizardStatus ? <p className="status ok">{wizardStatus}</p> : null}
      </section>

      <section className="panel">
        <h2>Baseline: Editors</h2>
        <p className="hint">Editable baseline files from `charter/*` and `skills/*`.</p>
        <label htmlFor="baselineArtifactSelect">Select artifact</label>
        <select
          id="baselineArtifactSelect"
          value={selectedEditorPath}
          onChange={(event) => {
            void handleEditorSelectionChange(event.target.value);
          }}
        >
          {baselineEditorArtifacts.map((artifact) => (
            <option key={artifact.path} value={artifact.path}>
              {artifact.label}
            </option>
          ))}
        </select>
        <label htmlFor="baselineArtifactEditor">{selectedEditorPath}</label>
        <textarea
          id="baselineArtifactEditor"
          value={selectedEditorContent}
          onChange={(event) => setSelectedEditorContent(event.target.value)}
          rows={10}
        />
        <button type="button" onClick={() => void handleSaveSelectedEditorArtifact()} disabled={busy}>
          Save selected baseline artifact
        </button>
        {editorStatus ? <p className="status ok">{editorStatus}</p> : null}
      </section>

      <section className="panel" data-testid="runs-control-panel">
        <h2>Runs: Pipeline Control</h2>
        <div className="actions">
          <button type="button" onClick={() => void handleRunPipeline("init")} disabled={busy} data-testid="run-init-btn">
            Run init
          </button>
          <button
            type="button"
            onClick={() => void handleRunPipeline("refresh")}
            disabled={busy}
            data-testid="run-refresh-btn"
          >
            Run refresh
          </button>
          <button
            type="button"
            onClick={() => void handleCancelSelectedRun()}
            disabled={busy || cancelBusy || !runId || !selectedRunIsActive}
            data-testid="run-cancel-btn"
          >
            Cancel selected run
          </button>
        </div>
        {runActionStatus ? <p className="status warn">{runActionStatus}</p> : null}

        {runStatus ? (
          <div className="status-block" data-testid="run-status-panel">
            <p>
              Run <code data-testid="run-status-run-id">{runStatus.run_id}</code> status:{" "}
              <strong data-testid="run-status-value">{runStatus.status}</strong>
            </p>
            <p>Pipeline: {runStatus.pipeline}</p>
            {runStatus.current_step ? <p>Current step: {runStatus.current_step}</p> : null}
            {runStatus.error_code ? <p className="status warn">Error code: {runStatus.error_code}</p> : null}
            {runStatus.error ? <p className="status err">Error: {runStatus.error}</p> : null}
            {selectedRunWarnings.length > 0 ? (
              <div data-testid="run-status-warnings">
                <p className="hint">Warnings ({selectedRunWarnings.length})</p>
                <ul>
                  {selectedRunWarnings.map((warning, index) => (
                    <li key={`run-warning-${index}-${warning}`}>{warning}</li>
                  ))}
                </ul>
              </div>
            ) : (
              <p data-testid="run-status-warnings-empty">Warnings: none</p>
            )}
          </div>
        ) : null}
      </section>

      <section className="panel" data-testid="runs-history-panel">
        <h2>Runs: History</h2>
        <p className="hint">
          Running: {runCounters.running} | Succeeded: {runCounters.succeeded} | Failed: {runCounters.failed}
        </p>
        {runList.length === 0 ? (
          <p>No runs yet.</p>
        ) : (
          <div className="run-table-wrap">
            <table className="run-table" data-testid="runs-history-table">
              <thead>
                <tr>
                  <th>Run ID</th>
                  <th>Status</th>
                  <th>Pipeline</th>
                  <th>Started</th>
                  <th>Finished</th>
                  <th>Error code</th>
                  <th>Warnings</th>
                </tr>
              </thead>
              <tbody>
                {runList.map((run) => (
                  <tr
                    key={run.run_id}
                    className={runId === run.run_id ? "selected" : ""}
                    onClick={() => void handleSelectRun(run.run_id)}
                  >
                    <td>
                      <button
                        type="button"
                        className="link-button"
                        onClick={(event) => {
                          event.stopPropagation();
                          void handleSelectRun(run.run_id);
                        }}
                      >
                        {run.run_id}
                      </button>
                    </td>
                    <td>{run.status}</td>
                    <td>{run.pipeline}</td>
                    <td>{formatTimestamp(run.started_at)}</td>
                    <td>{formatTimestamp(run.finished_at)}</td>
                    <td>{run.error_code || "-"}</td>
                    <td>{run.warnings?.length ?? 0}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="panel" data-testid="runs-logs-panel">
        <h2>Runs: Logs</h2>
        <div className="actions">
          <label htmlFor="runLogsViewMode">View</label>
          <select
            id="runLogsViewMode"
            value={runLogsViewMode}
            onChange={(event) => setRunLogsViewMode(event.target.value as "line" | "line+fields")}
            className="inline-select"
            data-testid="run-logs-view-select"
          >
            <option value="line">line</option>
            <option value="line+fields">line+fields</option>
          </select>
          <button
            type="button"
            onClick={() => void handleCopyRunLogs()}
            disabled={runLogs.length === 0}
            data-testid="run-logs-copy-btn"
          >
            Copy logs
          </button>
          <button
            type="button"
            onClick={() => handleDownloadRunLogs()}
            disabled={runLogs.length === 0 || !runId}
            data-testid="run-logs-download-btn"
          >
            Download logs
          </button>
        </div>
        {runLogsStatus ? <p className="status ok">{runLogsStatus}</p> : null}
        {runLogTaskrunPaths.length > 0 ? (
          <div className="actions">
            {runLogTaskrunPaths.map((path) => (
              <button key={`taskrun-log-open-${path}`} type="button" onClick={() => void handleOpenArtifact(path)}>
                Open taskrun artifact: {path}
              </button>
            ))}
          </div>
        ) : null}
        {runLogs.length === 0 ? (
          <p>No run logs yet.</p>
        ) : (
          <pre data-testid="run-logs-content">{runLogsRendered}</pre>
        )}
      </section>

      <section className="panel" data-testid="results-coverage-panel">
        <h2>Results: Coverage & Questions</h2>
        <div className="columns">
          <div>
            <h3>Coverage Summary</h3>
            <pre data-testid="coverage-summary-content">{coverageSummary || "No coverage summary yet."}</pre>
          </div>
          <div>
            <h3>Open Questions</h3>
            <pre data-testid="open-questions-content">{openQuestions || "No open questions yet."}</pre>
          </div>
        </div>
      </section>

      <section className="panel" data-testid="results-artifacts-panel">
        <h2>Results: Run Artifacts</h2>
        {artifacts.length === 0 ? (
          <p>No artifacts yet.</p>
        ) : (
          <div className="columns">
            <ul data-testid="run-artifacts-list">
              {artifacts.map((artifact) => (
                <li key={`${artifact.kind}-${artifact.path}`}>
                  <button type="button" className="link-button" onClick={() => void handleOpenArtifact(artifact.path)}>
                    {artifact.path}
                  </button>{" "}
                  ({artifact.kind})
                </li>
              ))}
            </ul>
            <div data-testid="run-artifact-content-panel">
              <h3 data-testid="run-artifact-selected-path">{selectedArtifact || "Artifact Content"}</h3>
              <pre data-testid="run-artifact-content">{selectedArtifactContent || "Select artifact to inspect."}</pre>
            </div>
          </div>
        )}
      </section>

      <section className="panel">
        <h2>Baseline: Git Helper Actions</h2>
        <label htmlFor="gitMessage">Commit message</label>
        <input id="gitMessage" value={gitMessage} onChange={(event) => setGitMessage(event.target.value)} />
        <button type="button" onClick={() => void handleGitCommit()} disabled={busy}>
          Commit workspace changes
        </button>

        <label htmlFor="proposalBranch">Proposal branch</label>
        <input id="proposalBranch" value={proposalBranch} onChange={(event) => setProposalBranch(event.target.value)} />
        <button type="button" onClick={() => void handleCreateProposalBranch()} disabled={busy}>
          Create/Switch proposal branch
        </button>

        {gitStatus ? <p className="status ok">{gitStatus}</p> : null}
      </section>

      {error ? <p className="status err">Error: {error}</p> : null}
    </main>
  );
}
