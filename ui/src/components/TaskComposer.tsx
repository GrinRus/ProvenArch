import { useMemo, useState, type FormEvent } from "react";

import type { GuidedRepo } from "../lib/appContracts";
import { createTask, type TaskScope } from "../lib/taskApi";
import { Button, PageHeader } from "./SemanticPrimitives";

type TaskComposerProps = {
  workspaceReady: boolean;
  repos: GuidedRepo[];
  runtimeMode: string;
  runtimeProvider: string;
  onCreated: (taskId: string) => void;
};

type RunnerMode = "fake" | "headless";
type RunnerProvider = "claude-code" | "qwen-code" | "codex-code";

const providers: Array<{ id: RunnerProvider; label: string }> = [
  { id: "claude-code", label: "Claude Code" },
  { id: "qwen-code", label: "Qwen Code" },
  { id: "codex-code", label: "Codex" },
];

export function TaskComposer({ workspaceReady, repos, runtimeMode, runtimeProvider, onCreated }: TaskComposerProps) {
  const normalizedMode = runtimeMode === "fake" || runtimeMode === "headless" ? runtimeMode : "";
  const normalizedProvider = providers.some((item) => item.id === runtimeProvider) ? runtimeProvider as RunnerProvider : "claude-code";
  const [title, setTitle] = useState("");
  const [goal, setGoal] = useState("");
  const [context, setContext] = useState("");
  const [mode, setMode] = useState<RunnerMode>((normalizedMode || "fake") as RunnerMode);
  const [provider, setProvider] = useState<RunnerProvider>(normalizedProvider);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const scope = useMemo(() => taskScope(repos), [repos]);
  const readiness = runnerReadiness({ workspaceReady, scope, mode, provider, runtimeMode: normalizedMode, runtimeProvider: normalizedProvider });
  const canSubmit = title.trim().length > 0 && goal.trim().length > 0 && readiness.ok && !busy;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canSubmit) return;
    setBusy(true);
    setError("");
    try {
      const task = await createTask({
        title: title.trim(),
        goal: goal.trim(),
        context: context.trim() || undefined,
        scope,
        desired_runner: {
          preset: mode === "fake" ? "deterministic-demo" : `${provider}-default`,
          mode,
          provider,
        },
      });
      if (!task.task_id) throw new Error("Task API returned no Task identity");
      onCreated(task.task_id);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Task creation failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel stage-panel task-composer" data-testid="task-composer">
      <PageHeader title="New Task" purpose="Describe the analysis intent, confirm its repository scope and admit a runner preset before any Attempt starts." state={<span className={`status ${readiness.ok ? "ok" : "warn"}`}>{readiness.label}</span>} />
      {!workspaceReady ? <p className="status warn" role="status" data-testid="task-composer-workspace-blocked">Workspace and runtime readiness are unavailable. No Task will be created.</p> : null}
      {error ? <p className="status err" role="alert" data-testid="task-composer-error">{error}</p> : null}
      <form onSubmit={handleSubmit}>
        <div className="settings-form-grid">
          <div className="field">
            <label htmlFor="task-title">Task title</label>
            <input id="task-title" data-testid="task-title" value={title} onChange={(event) => setTitle(event.target.value)} placeholder="e.g. Map payment authorization" required />
          </div>
          <div className="field">
            <label htmlFor="task-goal">Goal</label>
            <input id="task-goal" data-testid="task-goal" value={goal} onChange={(event) => setGoal(event.target.value)} placeholder="What should the validated result explain?" required />
          </div>
        </div>
        <div className="field">
          <label htmlFor="task-context">Context (optional)</label>
          <textarea id="task-context" data-testid="task-context" value={context} onChange={(event) => setContext(event.target.value)} placeholder="Constraints, questions or acceptance notes" />
        </div>
        <section className="subsection" aria-labelledby="task-scope-title">
          <h2 id="task-scope-title">Repository scope</h2>
          <p className="hint">The Task records the selected workspace repositories. Source repositories remain read-only.</p>
          {scope.repositories.length > 0 ? <ul className="compact-list" data-testid="task-scope-list">{scope.repositories.map((repository) => <li key={repository.name}><strong>{repository.name}</strong><span>{repository.paths.length > 0 ? repository.paths.join(", ") : "workspace root"}</span></li>)}</ul> : <p className="status warn" data-testid="task-scope-empty">No configured repository scope is available.</p>}
        </section>
        <section className="subsection" aria-labelledby="task-runner-title">
          <h2 id="task-runner-title">Runner readiness</h2>
          <div className="settings-form-grid">
            <div className="field">
              <label htmlFor="task-runner-mode">Runtime mode</label>
              <select id="task-runner-mode" data-testid="task-runner-mode" value={mode} onChange={(event) => setMode(event.target.value as RunnerMode)}>
                <option value="fake">Deterministic demo</option>
                <option value="headless">Headless provider</option>
              </select>
            </div>
            <div className="field">
              <label htmlFor="task-runner-provider">Provider</label>
              <select id="task-runner-provider" data-testid="task-runner-provider" value={provider} onChange={(event) => setProvider(event.target.value as RunnerProvider)}>{providers.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}</select>
            </div>
          </div>
          <p className={`status ${readiness.ok ? "ok" : "warn"}`} role="status" data-testid="task-runner-readiness">{readiness.detail}</p>
        </section>
        <div className="actions">
          <Button tone="primary" type="submit" data-testid="task-create-submit" disabled={!canSubmit}>{busy ? "Creating Task…" : "Create Task"}</Button>
        </div>
      </form>
    </section>
  );
}

function taskScope(repos: GuidedRepo[]): TaskScope {
  return {
    repositories: repos
      .map((repo) => ({ name: repo.name.trim(), paths: repo.analysis_include.split(/[\n,]/u).map((path) => path.trim()).filter(Boolean) }))
      .filter((repo) => repo.name.length > 0),
  };
}

function runnerReadiness(input: { workspaceReady: boolean; scope: TaskScope; mode: RunnerMode; provider: RunnerProvider; runtimeMode: string; runtimeProvider: RunnerProvider }): { ok: boolean; label: string; detail: string } {
  if (!input.workspaceReady) return { ok: false, label: "Blocked", detail: "Select and validate a workspace before creating a Task." };
  if (input.scope.repositories.length === 0) return { ok: false, label: "Scope needed", detail: "Add at least one configured repository before creating a Task." };
  if (!input.runtimeMode) return { ok: false, label: "Runner unavailable", detail: "Select a runtime in Setup before creating a Task." };
  if (input.mode !== input.runtimeMode) return { ok: false, label: "Mode mismatch", detail: `Current session is ${input.runtimeMode}; select the same mode for this Task.` };
  if (input.provider !== input.runtimeProvider) return { ok: false, label: "Provider mismatch", detail: `Current session is bound to ${input.runtimeProvider}; select that provider for this Task.` };
  return { ok: true, label: "Ready", detail: `Task will snapshot ${input.mode} / ${input.provider} at admission.` };
}
