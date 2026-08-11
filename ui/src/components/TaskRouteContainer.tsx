import { useEffect, useMemo, useState, type KeyboardEvent } from "react";

import type { TaskFilters, TaskRouteView } from "../lib/appRoutes";
import type { RunReviewSummaryResponse } from "../lib/appContracts";
import { getPipelineRunReviewSummary } from "../lib/runApi";
import {
  getTask,
  getTaskAttempt,
  listTaskAttempts,
  listTasks,
  setTaskArchive,
  type ProductTask,
  type TaskAttempt,
} from "../lib/taskApi";
import { Button, PageHeader } from "./SemanticPrimitives";

type TaskRouteContainerProps = {
  view: TaskRouteView;
  taskId?: string;
  attemptId?: string;
  invalid?: string[];
  filters?: TaskFilters;
  onFiltersChange?: (filters: TaskFilters) => void;
  onSelectTask?: (taskId: string, filters: TaskFilters) => void;
  onSelectAttempt?: (taskId: string, attemptId: string, filters: TaskFilters) => void;
  onOpenStudio?: (taskId: string, attemptId: string, filters: TaskFilters) => void;
  onBackToAttempt?: (taskId: string, attemptId: string, filters: TaskFilters) => void;
  onNewTask?: () => void;
  onOpenArchitecture?: (taskId: string) => void;
};

const groups = [
  { id: "needs_attention", label: "Needs attention" },
  { id: "running", label: "Running" },
  { id: "ready", label: "Ready" },
  { id: "completed", label: "Completed" },
  { id: "archived", label: "Archived" },
] as const;
type TaskGroup = typeof groups[number]["id"];

export function TaskRouteContainer(props: TaskRouteContainerProps) {
  if (props.invalid && props.invalid.length > 0) return <InvalidTaskRoute invalid={props.invalid} />;
  if (props.view === "inbox") {
    return <TaskInbox filters={props.filters ?? {}} onFiltersChange={props.onFiltersChange} onSelectTask={props.onSelectTask} onNewTask={props.onNewTask} />;
  }
  if (props.view === "detail" && props.taskId) {
    return <TaskDetail taskId={props.taskId} filters={props.filters ?? {}} onSelectAttempt={props.onSelectAttempt} onBack={() => props.onFiltersChange?.(props.filters ?? {})} onOpenArchitecture={props.onOpenArchitecture} />;
  }
  if (props.view === "attempt" && props.taskId && props.attemptId) {
    return <AttemptDetail taskId={props.taskId} attemptId={props.attemptId} filters={props.filters ?? {}} onSelectTask={props.onSelectTask} onOpenStudio={(taskId, attemptId) => props.onOpenStudio?.(taskId, attemptId, props.filters ?? {})} />;
  }
  if (props.view === "studio" && props.taskId && props.attemptId) {
    return <PipelineStudio taskId={props.taskId} attemptId={props.attemptId} onBack={() => props.onBackToAttempt?.(props.taskId!, props.attemptId!, props.filters ?? {})} />;
  }
  return <InvalidTaskRoute invalid={["task"]} />;
}

function InvalidTaskRoute({ invalid }: { invalid: string[] }) {
  return (
    <section className="panel stage-panel task-route-container" data-testid="task-route-container">
      <PageHeader title="Task Inbox" purpose="Task identity is explicit and never inferred from a legacy run." state={<span className="status warn">Unavailable</span>} />
      <p className="status warn" role="status" data-testid="task-route-invalid">This Task URL is not valid ({invalid.join(", ")}). No Task or Attempt was selected.</p>
      <div className="task-route-target" data-testid="task-route-inbox"><p className="hint">No legacy run or latest result was substituted.</p></div>
    </section>
  );
}

export function TaskInbox({ filters, onFiltersChange, onSelectTask, onNewTask }: { filters: TaskFilters; onFiltersChange?: (filters: TaskFilters) => void; onSelectTask?: (taskId: string, filters: TaskFilters) => void; onNewTask?: () => void }) {
  const [tasks, setTasks] = useState<ProductTask[]>([]);
  const [nextCursor, setNextCursor] = useState("");
  const [hasMore, setHasMore] = useState(false);
  const [status, setStatus] = useState<"loading" | "loaded" | "error">("loading");
  const [error, setError] = useState("");
  const [loadMoreBusy, setLoadMoreBusy] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    setStatus("loading");
    setError("");
    void listTasks(filters, "", controller.signal).then((response) => {
      if (controller.signal.aborted) return;
      setTasks(Array.isArray(response.items) ? response.items : []);
      setNextCursor(typeof response.next_cursor === "string" ? response.next_cursor : "");
      setHasMore(response.has_more === true);
      setStatus("loaded");
    }).catch((requestError) => {
      if (controller.signal.aborted) return;
      setError(requestError instanceof Error ? requestError.message : "Task list could not be loaded");
      setStatus("error");
    });
    return () => controller.abort();
  }, [filters.lifecycle, filters.runner, filters.repository, filters.from, filters.to]);

  const visibleTasks = useMemo(() => {
    const needle = filters.search?.toLocaleLowerCase().trim();
    if (!needle) return tasks;
    return tasks.filter((task) => [task.title, task.goal, task.context ?? ""].some((value) => value.toLocaleLowerCase().includes(needle)));
  }, [filters.search, tasks]);
  const grouped = useMemo(() => groups.reduce<Record<TaskGroup, ProductTask[]>>((result, group) => {
    result[group.id] = visibleTasks.filter((task) => taskGroup(task) === group.id);
    return result;
  }, { needs_attention: [], running: [], ready: [], completed: [], archived: [] }), [visibleTasks]);

  async function loadMore() {
    if (!nextCursor || loadMoreBusy) return;
    setLoadMoreBusy(true);
    try {
      const response = await listTasks(filters, nextCursor);
      setTasks((current) => [...current, ...response.items]);
      setNextCursor(response.next_cursor);
      setHasMore(response.has_more);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "More Tasks could not be loaded");
    } finally {
      setLoadMoreBusy(false);
    }
  }

  return (
    <section className="panel stage-panel task-inbox" data-testid="task-route-inbox">
      <PageHeader title="Task Inbox" purpose="Scan durable Tasks by lifecycle and open an exact Task or Attempt without consulting legacy run recency." state={<span className="status info">Authoritative Task API</span>} action={<Button tone="primary" onClick={onNewTask} data-testid="task-inbox-new">New Task</Button>} />
      <TaskFiltersBar filters={filters} onChange={(next) => onFiltersChange?.(next)} />
      {status === "loading" ? <p className="status info" role="status" data-testid="task-inbox-loading">Loading Tasks…</p> : null}
      {status === "error" ? <div className="task-inbox-recovery"><p className="status err" role="alert" data-testid="task-inbox-error">{error}</p><Button onClick={() => onFiltersChange?.({ ...filters })}>Retry</Button></div> : null}
      {status === "loaded" && visibleTasks.length === 0 ? <p className="status info" role="status" data-testid="task-inbox-empty">No Tasks match these filters. Clear a filter or create a new Task.</p> : null}
      <div className="task-inbox-groups" data-testid="task-inbox-groups">
        {groups.map((group) => <section className="task-group" key={group.id} data-testid={`task-group-${group.id}`} aria-labelledby={`task-group-${group.id}-title`}>
          <header className="task-group-heading"><h2 id={`task-group-${group.id}-title`}>{group.label}</h2><span className="status info">{grouped[group.id].length}</span></header>
          {grouped[group.id].length === 0 ? <p className="hint task-group-empty">No Tasks</p> : <div className="task-row-list">{grouped[group.id].map((task) => <TaskRow key={task.task_id} task={task} group={group.id} onSelect={() => onSelectTask?.(task.task_id, filters)} />)}</div>}
        </section>)}
      </div>
      {hasMore ? <div className="actions"><Button onClick={() => void loadMore()} disabled={loadMoreBusy}>{loadMoreBusy ? "Loading…" : "Load more Tasks"}</Button></div> : null}
    </section>
  );
}

function TaskFiltersBar({ filters, onChange }: { filters: TaskFilters; onChange: (filters: TaskFilters) => void }) {
  const update = (key: keyof TaskFilters, value: string) => onChange({ ...filters, [key]: value || undefined });
  return <form className="task-filters" aria-label="Task filters" onSubmit={(event) => event.preventDefault()}>
    <div className="field"><label htmlFor="task-filter-search">Search</label><input id="task-filter-search" data-testid="task-filter-search" value={filters.search ?? ""} onChange={(event) => update("search", event.target.value)} placeholder="Title, goal or context" /></div>
    <div className="field"><label htmlFor="task-filter-lifecycle">Lifecycle</label><select id="task-filter-lifecycle" data-testid="task-filter-lifecycle" value={filters.lifecycle ?? ""} onChange={(event) => update("lifecycle", event.target.value)}><option value="">Open and archived</option><option value="open">Open</option><option value="archived">Archived</option></select></div>
    <div className="field"><label htmlFor="task-filter-runner">Runner</label><input id="task-filter-runner" data-testid="task-filter-runner" value={filters.runner ?? ""} onChange={(event) => update("runner", event.target.value)} placeholder="Provider or preset" /></div>
    <div className="field"><label htmlFor="task-filter-repository">Repository</label><input id="task-filter-repository" data-testid="task-filter-repository" value={filters.repository ?? ""} onChange={(event) => update("repository", event.target.value)} placeholder="Repository name" /></div>
    <div className="field"><label htmlFor="task-filter-from">Activity from</label><input id="task-filter-from" data-testid="task-filter-from" type="date" value={dateInput(filters.from)} onChange={(event) => update("from", event.target.value ? `${event.target.value}T00:00:00Z` : "")} /></div>
    <div className="field"><label htmlFor="task-filter-to">Activity to</label><input id="task-filter-to" data-testid="task-filter-to" type="date" value={dateInput(filters.to)} onChange={(event) => update("to", event.target.value ? `${event.target.value}T23:59:59Z` : "")} /></div>
  </form>;
}

function TaskRow({ task, group, onSelect }: { task: ProductTask; group: TaskGroup; onSelect: () => void }) {
  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); onSelect(); } };
  return <article className="task-row" data-testid={`task-row-${task.task_id}`}>
    <div className="task-row-main" role="button" tabIndex={0} onClick={onSelect} onKeyDown={onKeyDown} aria-label={`Open Task ${task.title}`}>
      <div className="task-row-heading"><strong>{task.title || "Untitled Task"}</strong><span className={`status task-status-${group}`}>{groupLabel(group)}</span></div>
      <p>{task.goal}</p>
      <div className="task-row-meta"><span>{runnerLabel(task)}</span><span>{repositoryLabel(task)}</span><time dateTime={task.last_activity_at}>{formatDate(task.last_activity_at)}</time></div>
    </div>
  </article>;
}

function TaskDetail({ taskId, filters, onSelectAttempt, onBack, onOpenArchitecture }: { taskId: string; filters: TaskFilters; onSelectAttempt?: (taskId: string, attemptId: string, filters: TaskFilters) => void; onBack: () => void; onOpenArchitecture?: (taskId: string) => void }) {
  const [task, setTask] = useState<ProductTask | null>(null);
  const [attempts, setAttempts] = useState<TaskAttempt[]>([]);
  const [review, setReview] = useState<RunReviewSummaryResponse | null>(null);
  const [state, setState] = useState<"loading" | "loaded" | "error">("loading");
  const [error, setError] = useState("");
  const [archiveStatus, setArchiveStatus] = useState("");
  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    void Promise.all([getTask(taskId, controller.signal), listTaskAttempts(taskId, controller.signal)]).then(async ([nextTask, nextAttempts]) => {
      if (controller.signal.aborted) return;
      setTask(nextTask);
      setAttempts(nextAttempts.items);
      const latest = nextAttempts.items[nextAttempts.items.length - 1];
      if (latest?.run_id && latest.status !== "queued" && latest.status !== "running") {
        setReview(await getPipelineRunReviewSummary(latest.run_id, true, { signal: controller.signal }));
      } else {
        setReview(null);
      }
      setState("loaded");
    }).catch((requestError) => {
      if (controller.signal.aborted) return;
      setError(requestError instanceof Error ? requestError.message : "Task could not be loaded");
      setState("error");
    });
    return () => controller.abort();
  }, [taskId]);

  async function archive(archived: boolean) {
    if (!task) return;
    if (!window.confirm(`${archived ? "Archive" : "Unarchive"} this Task? Run evidence is retained.`)) return;
    try {
      const updated = await setTaskArchive(task.task_id, task.revision, archived);
      setTask(updated);
      setArchiveStatus(archived ? "Task archived. Run evidence remains read-only and retained." : "Task unarchived.");
    } catch (requestError) {
      setArchiveStatus(requestError instanceof Error ? requestError.message : "Task lifecycle update failed");
    }
  }

  return <section className="panel stage-panel task-detail" data-testid="task-route-detail">
    <PageHeader title={task?.title || "Task detail"} purpose="Exact Task identity, durable outcome state and immutable Attempt history." state={<span className="status info">Task-first</span>} action={<div className="actions"><Button density="compact" onClick={onBack} data-testid="task-detail-back">Back to Inbox</Button>{task ? <Button density="compact" onClick={() => void archive(task.lifecycle === "open")} data-testid="task-archive">{task.lifecycle === "open" ? "Archive" : "Unarchive"}</Button> : null}</div>} />
    {state === "loading" ? <p className="status info" role="status">Loading exact Task identity…</p> : null}
    {state === "error" ? <p className="status err" role="alert" data-testid="task-detail-error">{error}</p> : null}
    {archiveStatus ? <p className="status info" role="status" data-testid="task-archive-status">{archiveStatus}</p> : null}
    {task ? <>
      <div className="task-detail-summary"><p className="eyebrow">Task ID <code>{task.task_id}</code></p><p>{task.goal}</p><dl className="compact-defs"><div><dt>Lifecycle</dt><dd>{task.lifecycle}</dd></div><div><dt>Outcome</dt><dd>{task.outcome.state === "available" ? "Available" : task.outcome.unavailable_reason || "Unavailable"}</dd></div><div><dt>Runner</dt><dd>{runnerLabel(task)}</dd></div><div><dt>Scope</dt><dd>{repositoryLabel(task)}</dd></div></dl></div>
      <TaskOutcome task={task} latestAttempt={attempts[attempts.length - 1]} review={review} onOpenArchitecture={onOpenArchitecture} />
      <section className="task-attempt-history" aria-labelledby="task-attempt-history-title"><div className="task-group-heading"><h2 id="task-attempt-history-title">Attempt history</h2><span className="status info">{attempts.length}</span></div>{attempts.length === 0 ? <p className="hint">No Attempt has been admitted for this Task yet.</p> : <div className="task-attempt-list">{attempts.map((attempt) => <AttemptRow key={attempt.attempt_id} attempt={attempt} onSelect={() => onSelectAttempt?.(task.task_id, attempt.attempt_id, filters)} />)}</div>}</section>
    </> : null}
  </section>;
}

function TaskOutcome({ task, latestAttempt, review, onOpenArchitecture }: { task: ProductTask; latestAttempt?: TaskAttempt; review: RunReviewSummaryResponse | null; onOpenArchitecture?: (taskId: string) => void }) {
  const result = review?.result;
  const semantic = review?.review?.summary;
  const terminalFailure = latestAttempt && ["failed", "canceled", "timeout"].includes(latestAttempt.status);
  if (!result && !terminalFailure) return <section className="task-outcome task-outcome-unavailable" data-testid="task-outcome"><h2>Outcome</h2><p className="hint">No terminal Attempt outcome is available yet. The Task intent is durable and can be admitted when the runner is ready.</p></section>;
  if (!result) return <section className="task-outcome task-outcome-failed" data-testid="task-outcome"><div><p className="eyebrow">Attempt outcome</p><h2>{latestAttempt?.status === "canceled" ? "Attempt canceled" : "Attempt needs recovery"}</h2><p>{latestAttempt?.terminal_summary?.message || "The last Attempt did not produce a promotable outcome."}</p></div><dl className="compact-defs"><div><dt>Evidence</dt><dd>{latestAttempt?.retained_evidence || "Retained evidence state is reported by the Attempt."}</dd></div><div><dt>Current Architecture</dt><dd>Not changed by this Attempt; last-good state remains independent.</dd></div></dl></section>;
  const partial = !review?.review?.semantic_changes.available;
  return <section className={`task-outcome task-outcome-${result.state}`} data-testid="task-outcome"><div className="task-outcome-copy"><p className="eyebrow">Outcome · exact run snapshot <code>{review.run_id}</code></p><h2>{result.state === "completed" ? "Architecture result ready" : result.state === "completed_with_gaps" ? "Result ready with gaps" : result.state === "canceled" ? "Attempt canceled" : "Attempt needs recovery"}</h2><p>{result.summary}</p><p className="hint">{result.promotion.current_usable ? "Current validator-approved Architecture remains available independently of this Task review." : "No current validator-approved Architecture is available."}</p></div><dl className="task-outcome-counts"><div><dt>Entities</dt><dd>{semantic ? `${semantic.entities_added} added · ${semantic.entities_changed} changed · ${semantic.entities_removed} removed` : partial ? "Unavailable" : result.produced.entities ?? 0}</dd></div><div><dt>Edges</dt><dd>{semantic ? `${semantic.edges_added} added · ${semantic.edges_changed} changed · ${semantic.edges_removed} removed` : partial ? "Unavailable" : result.produced.edges ?? 0}</dd></div><div><dt>Findings / questions</dt><dd>{semantic ? `${semantic.findings} / ${semantic.questions}` : partial ? "Unavailable" : "Unavailable"}</dd></div><div><dt>Gaps</dt><dd>{semantic ? semantic.gaps : result.coverage?.missing ?? "Unavailable"}</dd></div></dl><div className="task-outcome-action"><strong>Recommended next action</strong><span>{result.recommended_action.replace(/_/g, " ")}</span>{onOpenArchitecture && result.promotion.current_usable ? <Button density="compact" onClick={() => onOpenArchitecture(task.task_id)} data-testid="task-open-architecture">Open current Architecture</Button> : null}</div>{partial ? <p className="status warn">Semantic comparison is unavailable for this snapshot; no zero delta was fabricated.</p> : null}</section>;
}

function AttemptRow({ attempt, onSelect }: { attempt: TaskAttempt; onSelect: () => void }) {
  const duration = attempt.started_at && attempt.finished_at ? formatDuration(Date.parse(attempt.finished_at) - Date.parse(attempt.started_at)) : "—";
  return <button type="button" className="task-attempt-row" data-testid={`attempt-row-${attempt.attempt_id}`} onClick={onSelect}><span><strong>{attempt.status}</strong><small>{attempt.attempt_id}</small></span><span>{runnerLabelFromAttempt(attempt)}</span><span>{duration}</span><span>{attempt.parent_attempt_id ? `child of ${attempt.parent_attempt_id}` : "root Attempt"}</span></button>;
}

function AttemptDetail({ taskId, attemptId, filters, onSelectTask, onOpenStudio }: { taskId: string; attemptId: string; filters: TaskFilters; onSelectTask?: (taskId: string, filters: TaskFilters) => void; onOpenStudio?: (taskId: string, attemptId: string) => void }) {
  const [attempt, setAttempt] = useState<TaskAttempt | null>(null);
  const [state, setState] = useState<"loading" | "loaded" | "error">("loading");
  const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    void getTaskAttempt(taskId, attemptId, controller.signal).then((nextAttempt) => { if (!controller.signal.aborted) { setAttempt(nextAttempt); setState("loaded"); } }).catch((requestError) => { if (!controller.signal.aborted) { setError(requestError instanceof Error ? requestError.message : "Attempt could not be loaded"); setState("error"); } });
    return () => controller.abort();
  }, [taskId, attemptId]);
  return <section className="panel stage-panel task-detail" data-testid="task-route-attempt"><PageHeader title="Attempt detail" purpose="Immutable admitted snapshot linked to this exact Task and pipeline run." state={<span className="status info">Read-only snapshot</span>} action={<div className="actions"><Button density="compact" onClick={() => onSelectTask?.(taskId, filters)}>Back to Task</Button>{attempt ? <Button density="compact" onClick={() => onOpenStudio?.(taskId, attempt.attempt_id)} data-testid="attempt-open-studio">Open Pipeline Studio</Button> : null}</div>} />{state === "loading" ? <p className="status info" role="status">Loading exact Attempt identity…</p> : null}{state === "error" ? <p className="status err" role="alert">{error}</p> : null}<dl className="compact-defs" data-testid="task-route-identities"><div><dt>Task ID</dt><dd>{taskId}</dd></div><div><dt>Attempt ID</dt><dd>{attemptId}</dd></div></dl>{attempt ? <div className="task-attempt-detail"><p className="eyebrow">Attempt ID <code>{attempt.attempt_id}</code></p><dl className="compact-defs"><div><dt>Task ID</dt><dd>{attempt.task_id}</dd></div><div><dt>Run ID</dt><dd>{attempt.run_id}</dd></div><div><dt>Status</dt><dd>{attempt.status}</dd></div><div><dt>Runner</dt><dd>{runnerLabelFromAttempt(attempt)}</dd></div><div><dt>Pipeline</dt><dd>{attempt.pipeline}</dd></div><div><dt>Lineage</dt><dd>{attempt.parent_attempt_id ? `child of ${attempt.parent_attempt_id}` : "root Attempt"}</dd></div></dl><p className="hint">The admitted snapshot is immutable; later Settings or workspace changes cannot rewrite it.</p></div> : null}</section>;
}

function PipelineStudio({ taskId, attemptId, onBack }: { taskId: string; attemptId: string; onBack: () => void }) {
  const [attempt, setAttempt] = useState<TaskAttempt | null>(null);
  const [review, setReview] = useState<RunReviewSummaryResponse | null>(null);
  const [state, setState] = useState<"loading" | "loaded" | "error">("loading");
  const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    void getTaskAttempt(taskId, attemptId, controller.signal).then(async (nextAttempt) => {
      if (controller.signal.aborted) return;
      setAttempt(nextAttempt);
      setReview(await getPipelineRunReviewSummary(nextAttempt.run_id, true, { signal: controller.signal }));
      setState("loaded");
    }).catch((requestError) => { if (!controller.signal.aborted) { setError(requestError instanceof Error ? requestError.message : "Pipeline Studio could not be loaded"); setState("error"); } });
    return () => controller.abort();
  }, [taskId, attemptId]);
  const steps = review?.steps.length ? review.steps : fallbackStudioSteps(attempt?.pipeline);
  const active = attempt?.status === "queued" || attempt?.status === "running";
  const blocker = review?.recovery || (attempt && ["failed", "canceled", "timeout"].includes(attempt.status) ? { title: "Attempt stopped", explanation: attempt.terminal_summary?.message || "No promotable result was recorded.", retained_evidence: attempt.retained_evidence || "Retained evidence state is available on the Attempt." } : null);
  return <section className="panel stage-panel pipeline-studio" data-testid="task-pipeline-studio"><PageHeader title="Pipeline Studio" purpose="Focused diagnostics for this exact immutable Attempt; no global run or latest-result fallback." state={<span className="status info">Attempt-bound</span>} action={<Button density="compact" onClick={onBack} data-testid="pipeline-studio-back">Back to Attempt</Button>} />{state === "loading" ? <p className="status info" role="status">Loading exact Attempt and structured progress…</p> : null}{state === "error" ? <p className="status err" role="alert">{error}</p> : null}{attempt ? <><div className="pipeline-studio-identity"><p className="eyebrow">Task <code>{taskId}</code> · Attempt <code>{attempt.attempt_id}</code> · Run <code>{attempt.run_id}</code></p><dl className="compact-defs"><div><dt>Status</dt><dd>{attempt.status}</dd></div><div><dt>Runner snapshot</dt><dd>{runnerLabelFromAttempt(attempt)}</dd></div><div><dt>Pipeline</dt><dd>{attempt.pipeline}</dd></div></dl></div><section className="pipeline-track" aria-labelledby="pipeline-track-title"><div className="task-group-heading"><h2 id="pipeline-track-title">Canonical pipeline steps</h2>{active && review?.progress ? <span className="status info">{review.progress.completed_steps}/{review.progress.total_steps} complete</span> : null}</div><ol>{steps.map((step) => <li key={step.step_id} className={`pipeline-step pipeline-step-${step.state}`}><span className="pipeline-step-marker" aria-hidden="true" /> <div><strong>{step.label || step.key}</strong><span>{step.state}</span>{step.last_message ? <small>{step.last_message}</small> : null}</div></li>)}</ol>{!review?.progress ? <p className="hint">Structured progress is unavailable for this snapshot; no percentage is derived from provider output or heartbeats.</p> : null}</section>{blocker ? <section className="pipeline-blocker" data-testid="pipeline-blocker"><p className="eyebrow">Selected blocker</p><h2>{blocker.title}</h2><p>{blocker.explanation}</p><p className="hint">Retained data: {blocker.retained_evidence}</p></section> : null}<details className="pipeline-diagnostics"><summary>Diagnostics disclosure</summary><dl className="compact-defs"><div><dt>Error code</dt><dd>{review?.error_code || attempt.terminal_summary?.error_code || "none recorded"}</dd></div><div><dt>Warnings</dt><dd>{review?.warnings?.length ? review.warnings.join("; ") : "none recorded"}</dd></div></dl></details></> : null}</section>;
}

function fallbackStudioSteps(pipeline?: string): Array<{ step_id: string; key: string; label: string; state: "done" | "active" | "failed" | "pending"; artifact_count: number; artifact_paths: string[]; taskrun_paths: string[]; warnings_count: number; errors_count: number; last_message?: string }> {
  const keys = pipeline === "refresh" ? ["refresh.step1.collect", "refresh.step2.asis_docs", "refresh.step3.findings", "refresh.step4.proposals"] : ["init.step0.constitution", "init.step1.collect", "init.step2.asis_docs", "init.step3.findings", "init.step4.proposals"];
  return keys.map((key) => ({ step_id: key, key, label: key, state: "pending", artifact_count: 0, artifact_paths: [], taskrun_paths: [], warnings_count: 0, errors_count: 0 }));
}

function taskGroup(task: ProductTask): TaskGroup {
  if (task.lifecycle === "archived") return "archived";
  const latest = task.attempts[task.attempts.length - 1];
  if (latest && (latest.status === "queued" || latest.status === "running")) return "running";
  if (!latest) return "ready";
  if (latest.status === "succeeded" && task.outcome.state === "available") return "completed";
  return "needs_attention";
}

function groupLabel(group: TaskGroup): string { return groups.find((item) => item.id === group)?.label ?? group; }
function runnerLabel(task: ProductTask): string { return task.desired_runner.provider || task.desired_runner.preset || "runner unavailable"; }
function runnerLabelFromAttempt(attempt: TaskAttempt): string { return attempt.effective_runtime?.provider || attempt.desired_runner?.provider || attempt.desired_runner?.preset || "runner snapshot unavailable"; }
function repositoryLabel(task: ProductTask): string { return task.scope.repositories.map((repository) => repository.name).join(", ") || "scope unavailable"; }
function formatDate(value: string): string { const date = new Date(value); return Number.isNaN(date.getTime()) ? "activity unavailable" : date.toLocaleString(); }
function dateInput(value?: string): string { return value ? value.slice(0, 10) : ""; }
function formatDuration(milliseconds: number): string { if (!Number.isFinite(milliseconds) || milliseconds < 0) return "—"; const seconds = Math.round(milliseconds / 1000); if (seconds < 60) return `${seconds}s`; const minutes = Math.floor(seconds / 60); return `${minutes}m ${seconds % 60}s`; }
