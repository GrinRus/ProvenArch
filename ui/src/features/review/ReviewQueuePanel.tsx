import { StatusBadge } from "../../components/ConsolePrimitives";
import type { ReviewQueueItem } from "../../lib/appContracts";

export function ReviewQueuePanel({
  queue,
  selectedArtifact,
  onOpenArtifact,
}: {
  queue: ReviewQueueItem[];
  selectedArtifact: string;
  onOpenArtifact: (path: string) => void;
}) {
  return (
    <section className="review-queue" id="review-queue" data-testid="review-queue">
      <div className="section-heading-row">
        <h2>Review Queue</h2>
        <StatusBadge tone={queue.length > 0 ? "warn" : "ok"}>{queue.length}</StatusBadge>
      </div>
      {queue.length === 0 ? (
        <p className="hint">No generated review items are waiting. Run Analysis or select a completed run.</p>
      ) : (
        <ul>
          {queue.slice(0, 10).map((item) => (
            <li key={item.id}>
              <button type="button" className={`review-queue-item${item.path === selectedArtifact ? " is-selected" : ""}`} aria-current={item.path === selectedArtifact ? "true" : undefined} aria-label={`Review queue item: ${item.title}`} onClick={() => onOpenArtifact(item.path)}>
                <StatusBadge tone={item.severity === "error" ? "error" : item.severity === "warn" ? "warn" : "info"}>{item.kind}</StatusBadge>
                <span>{item.title}</span>
                <code>{item.path}</code>
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
