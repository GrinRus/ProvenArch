import { EvidenceLink, PrimaryAction, StatusBadge } from "./ConsolePrimitives";
import type { InspectorItem, NextAction, Severity } from "../lib/consoleTypes";

type RightInspectorProps = {
  nextAction: NextAction;
  blockers: InspectorItem[];
  evidenceRefs: InspectorItem[];
  workspaceHealth: InspectorItem[];
  runtimeSafety: InspectorItem[];
  gitPublication: InspectorItem[];
  onPrimaryAction: () => void;
  onOpenArtifact: (path: string) => void;
};

const severityGlyphs: Record<Severity, string> = {
  info: "i",
  ok: "✓",
  warn: "!",
  error: "!",
};

export function RightInspector({
  nextAction,
  blockers,
  evidenceRefs,
  workspaceHealth,
  runtimeSafety,
  gitPublication,
  onPrimaryAction,
  onOpenArtifact,
}: RightInspectorProps) {
  const hardBlockers = blockers.filter((item) => item.severity === "error");
  const openQuestionItems = blockers.filter((item) => item.severity === "warn" && item.label.toLowerCase().includes("open question"));
  const reviewWarnings = blockers.filter((item) => item.severity === "warn" && !openQuestionItems.includes(item));
  const nextActionSeverity = nextAction.disabledReason || hardBlockers.length > 0 ? "error" : reviewWarnings.length > 0 || openQuestionItems.length > 0 ? "warn" : "ok";
  const nextActionStatus = nextAction.disabledReason ? "blocked" : hardBlockers.length > 0 ? "needs fix" : reviewWarnings.length > 0 || openQuestionItems.length > 0 ? "review" : "ready";
  return (
    <aside className="right-inspector" data-testid="right-inspector">
      <section className="inspector-section next-action-section" data-testid="next-action-panel">
        <div className="section-heading-row">
          <h2>Next action</h2>
          <StatusBadge tone={nextActionSeverity}>{nextActionStatus}</StatusBadge>
        </div>
        <p className="inspector-title">{nextAction.label}</p>
        <p className="hint">{nextAction.description}</p>
        {nextAction.disabledReason ? <p className="status warn">{nextAction.disabledReason}</p> : null}
        <PrimaryAction onClick={onPrimaryAction} disabled={Boolean(nextAction.disabledReason)} testId="inspector-primary-action">
          {nextAction.label}
        </PrimaryAction>
      </section>

      <InspectorList testId="blockers-panel" title="Hard blockers" emptyLabel="No hard blockers detected." items={hardBlockers} onOpenArtifact={onOpenArtifact} />
      <InspectorList testId="review-warnings-panel" title="Review warnings" emptyLabel="No review warnings." items={reviewWarnings} onOpenArtifact={onOpenArtifact} />
      <InspectorList testId="open-questions-panel" title="Open questions" emptyLabel="No open questions loaded." items={openQuestionItems} onOpenArtifact={onOpenArtifact} />
      <InspectorList testId="evidence-refs-panel" title="Evidence refs" emptyLabel="No evidence yet." items={evidenceRefs} onOpenArtifact={onOpenArtifact} />
      <InspectorList testId="workspace-health-panel" title="Workspace health" emptyLabel="Workspace status unavailable." items={workspaceHealth} onOpenArtifact={onOpenArtifact} secondary />
      <InspectorList testId="runtime-safety-panel" title="Runtime safety" emptyLabel="Runtime profile unavailable." items={runtimeSafety} onOpenArtifact={onOpenArtifact} secondary />
      <InspectorList testId="git-publication-panel" title="Git publication" emptyLabel="Git publication path unavailable." items={gitPublication} onOpenArtifact={onOpenArtifact} secondary />
    </aside>
  );
}

type InspectorListProps = {
  testId: string;
  title: string;
  emptyLabel: string;
  items: InspectorItem[];
  onOpenArtifact: (path: string) => void;
  secondary?: boolean;
};

function InspectorList({ testId, title, emptyLabel, items, onOpenArtifact, secondary = false }: InspectorListProps) {
  const hasItems = items.length > 0;
  const hasAttention = items.some((item) => item.severity === "error" || item.severity === "warn");
  const compact = !hasItems || (secondary && !hasAttention);
  const countLabel = hasItems ? `${items.length}` : "0";
  if (compact) {
    return (
      <details className={`inspector-section inspector-disclosure${hasItems ? " has-items" : " is-empty"}`} data-testid={testId}>
        <summary>
          <span>{title}</span>
          <span className="count-pill">{countLabel}</span>
        </summary>
        {hasItems ? <InspectorItems title={title} items={items} onOpenArtifact={onOpenArtifact} /> : <p className="hint">{emptyLabel}</p>}
      </details>
    );
  }
  return (
    <section className="inspector-section" data-testid={testId}>
      <div className="section-heading-row">
        <h2>{title}</h2>
        <span className="count-pill">{countLabel}</span>
      </div>
      <InspectorItems title={title} items={items} onOpenArtifact={onOpenArtifact} />
    </section>
  );
}

function InspectorItems({ title, items, onOpenArtifact }: { title: string; items: InspectorItem[]; onOpenArtifact: (path: string) => void }) {
  return (
    <ul className="inspector-list">
      {items.map((item, index) => (
        <li key={`${title}-${item.label}-${index}`} className={`inspector-item ${item.severity}`}>
          <span className={`inspector-item-icon ${item.severity}`} aria-label={item.severity}>
            {severityGlyphs[item.severity]}
          </span>
          <div>
            <p className="inspector-item-title">{item.label}</p>
            <p className="hint">{item.detail}</p>
            {item.path ? <EvidenceLink path={item.path} onOpenArtifact={onOpenArtifact} /> : null}
          </div>
        </li>
      ))}
    </ul>
  );
}
