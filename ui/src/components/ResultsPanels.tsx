import { Suspense, lazy } from "react";

import { TabNav, type TabOption } from "./TabNav";
import type { Artifact } from "../lib/appContracts";

const MermaidPreview = lazy(async () => {
  const module = await import("./MermaidPreview");
  return { default: module.MermaidPreview };
});

type ResultsPanelsProps = {
  resultsTab: "coverage" | "artifacts" | "diagrams";
  resultsTabOptions: Array<TabOption<"coverage" | "artifacts" | "diagrams">>;
  coverageSummary: string;
  openQuestions: string;
  nonDiagramArtifacts: Artifact[];
  diagramArtifacts: Artifact[];
  selectedArtifact: string;
  selectedArtifactContent: string;
  selectedArtifactIsMermaid: boolean;
  onResultsTabChange: (tab: "coverage" | "artifacts" | "diagrams") => void;
  onOpenArtifact: (path: string) => void;
};

export function ResultsPanels({
  resultsTab,
  resultsTabOptions,
  coverageSummary,
  openQuestions,
  nonDiagramArtifacts,
  diagramArtifacts,
  selectedArtifact,
  selectedArtifactContent,
  selectedArtifactIsMermaid,
  onResultsTabChange,
  onOpenArtifact,
}: ResultsPanelsProps) {
  const selectedArtifactIsLoading = selectedArtifactContent === "Loading...";
  return (
    <>
      <TabNav value={resultsTab} onChange={onResultsTabChange} options={resultsTabOptions} testId="results-tabs" />

      {resultsTab === "coverage" ? (
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
      ) : null}

      {resultsTab === "artifacts" ? (
        <section className="panel" data-testid="results-artifacts-panel">
          <h2>Results: Run Artifacts</h2>
          {nonDiagramArtifacts.length === 0 ? (
            <p>No non-diagram artifacts yet.</p>
          ) : (
            <div className="columns">
              <ul data-testid="run-artifacts-list">
                {nonDiagramArtifacts.map((artifact) => (
                  <li key={`${artifact.kind}-${artifact.path}`}>
                    <button type="button" className="link-button" onClick={() => onOpenArtifact(artifact.path)}>
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
      ) : null}

      {resultsTab === "diagrams" ? (
        <section className="panel" data-testid="results-diagrams-panel">
          <h2>Results: Diagrams</h2>
          {diagramArtifacts.length === 0 ? (
            <p>No diagram artifacts yet.</p>
          ) : (
            <div className="columns">
              <ul data-testid="run-diagrams-list">
                {diagramArtifacts.map((artifact) => (
                  <li key={`${artifact.kind}-${artifact.path}`}>
                    <button type="button" className="link-button" onClick={() => onOpenArtifact(artifact.path)}>
                      {artifact.path}
                    </button>{" "}
                    ({artifact.kind})
                  </li>
                ))}
              </ul>
              <div data-testid="run-diagram-content-panel">
                <h3 data-testid="run-diagram-selected-path">{selectedArtifact || "Diagram Preview"}</h3>
                {selectedArtifactIsMermaid ? (
                  selectedArtifactIsLoading ? (
                    <p className="hint">Loading diagram...</p>
                  ) : (
                    <Suspense fallback={<p className="hint">Loading diagram renderer...</p>}>
                      <MermaidPreview source={selectedArtifactContent} title={selectedArtifact || "diagram"} />
                    </Suspense>
                  )
                ) : (
                  <pre data-testid="run-diagram-content">{selectedArtifactContent || "Select a `.mmd` diagram artifact to preview."}</pre>
                )}
              </div>
            </div>
          )}
        </section>
      ) : null}
    </>
  );
}
