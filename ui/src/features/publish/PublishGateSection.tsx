import { StatusBadge } from "../../components/ConsolePrimitives";
import type { PublishGateItem } from "./publishUtils";

export function PublishGateSection({
  testId,
  title,
  emptyLabel,
  items,
}: {
  testId: string;
  title: string;
  emptyLabel: string;
  items: PublishGateItem[];
}) {
  return (
    <section className="publish-gate-section" data-testid={testId}>
      <div className="publish-gate-section-head">
        <h3>{title}</h3>
        <span>{items.length}</span>
      </div>
      {items.length === 0 ? (
        <p className="hint">{emptyLabel}</p>
      ) : (
        <ul className="publish-checklist">
          {items.map((item) => (
            <li key={`${title}-${item.label}`}>
              <StatusBadge tone={item.tone}>{item.label}</StatusBadge>
              <span>{item.detail}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
