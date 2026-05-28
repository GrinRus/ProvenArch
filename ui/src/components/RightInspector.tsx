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
  const nextActionSeverity = blockers.some((item) => item.severity === "error") || nextAction.disabledReason ? "warn" : blockers.length > 0 ? "warn" : "ok";
  const nextActionStatus = nextAction.disabledReason ? "blocked" : blockers.length > 0 ? "attention" : "ready";
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

      <InspectorList testId="blockers-panel" title="Blockers" emptyLabel="No blockers detected." items={blockers} onOpenArtifact={onOpenArtifact} />
      <InspectorList testId="evidence-refs-panel" title="Evidence refs" emptyLabel="No evidence yet." items={evidenceRefs} onOpenArtifact={onOpenArtifact} />
      <InspectorList testId="workspace-health-panel" title="Workspace health" emptyLabel="Workspace status unavailable." items={workspaceHealth} onOpenArtifact={onOpenArtifact} />
      <InspectorList testId="runtime-safety-panel" title="Runtime safety" emptyLabel="Runtime profile unavailable." items={runtimeSafety} onOpenArtifact={onOpenArtifact} />
      <InspectorList testId="git-publication-panel" title="Git publication" emptyLabel="Git publication path unavailable." items={gitPublication} onOpenArtifact={onOpenArtifact} />
    </aside>
  );
}

type InspectorListProps = {
  testId: string;
  title: string;
  emptyLabel: string;
  items: InspectorItem[];
  onOpenArtifact: (path: string) => void;
};

function InspectorList({ testId, title, emptyLabel, items, onOpenArtifact }: InspectorListProps) {
  return (
    <section className="inspector-section" data-testid={testId}>
      <div className="section-heading-row">
        <h2>{title}</h2>
        <span className="count-pill">{items.length}</span>
      </div>
      {items.length === 0 ? (
        <p className="hint">{emptyLabel}</p>
      ) : (
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
      )}
    </section>
  );
}
