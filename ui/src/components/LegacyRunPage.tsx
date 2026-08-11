import type { ReactNode } from "react";

import type { RunCoordination } from "../lib/appContracts";

/**
 * Explicit read-only migration surface for pre-Task pipeline runs.
 *
 * A legacy run is never promoted into a synthetic Task. Keeping this wrapper
 * under the Tasks route makes the authority boundary visible while allowing
 * operators to inspect retained evidence until those runs age out.
 */
export function LegacyRunPage({ coordination, selectedRunID, children }: { coordination: RunCoordination; selectedRunID?: string; children: ReactNode }) {
  return (
    <section className="legacy-run-page" data-testid="legacy-run-page">
      <header className="page-context-header">
        <div>
          <p className="eyebrow">Legacy evidence · read-only</p>
          <h1>{selectedRunID ? "Legacy run diagnostics" : "Runtime diagnostics"}</h1>
          <p className="hint">This execution predates Task/Attempt identity. It remains inspectable evidence and is never converted into a synthetic Task.</p>
        </div>
        <p>{coordination.active_run_id ? `Active ${coordination.active_run_id}` : "No active run"}{coordination.pending ? ` · Pending ${coordination.pending.run_id}` : ""}</p>
      </header>
      {children}
    </section>
  );
}
