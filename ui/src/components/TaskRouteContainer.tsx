import type { TaskRouteView } from "../lib/appRoutes";
import { PageHeader } from "./SemanticPrimitives";

type TaskRouteContainerProps = {
  view: TaskRouteView;
  taskId?: string;
  attemptId?: string;
  invalid?: string[];
};

const viewCopy: Record<TaskRouteView, { title: string; purpose: string }> = {
  inbox: { title: "Task Inbox", purpose: "The additive Task-first route is reserved for the authoritative Task list." },
  new: { title: "New Task", purpose: "The additive route is ready for the Task composer." },
  detail: { title: "Task detail", purpose: "The additive route is ready for an authoritative Task and its Attempt history." },
  attempt: { title: "Attempt detail", purpose: "The additive route is ready for an immutable Attempt and exact run snapshot." },
};

export function TaskRouteContainer({ view, taskId, attemptId, invalid = [] }: TaskRouteContainerProps) {
  const copy = viewCopy[view];
  const hasSelection = view === "detail" || view === "attempt";
  return (
    <section className="panel stage-panel task-route-container" data-testid="task-route-container" aria-labelledby="task-route-title">
      <PageHeader title={copy.title} purpose={copy.purpose} state={<span className="status info">Route target</span>} />
      {invalid.length > 0 ? <p className="status warn" role="status" data-testid="task-route-invalid">This Task URL is not valid ({invalid.join(", ")}). No Task or Attempt was selected.</p> : null}
      <div className="task-route-target" data-testid={`task-route-${view}`}>
        <p className="eyebrow">W23B1 target container</p>
        <h2 id="task-route-title">Authoritative Task/Attempt surface pending</h2>
        <p className="hint">This additive route does not load or mutate Task data yet. It never substitutes a legacy run or the latest available result.</p>
        {hasSelection ? <dl className="compact-defs" data-testid="task-route-identities">
          <div><dt>Task ID</dt><dd>{taskId ?? "Unavailable"}</dd></div>
          {view === "attempt" ? <div><dt>Attempt ID</dt><dd>{attemptId ?? "Unavailable"}</dd></div> : null}
        </dl> : null}
      </div>
    </section>
  );
}
