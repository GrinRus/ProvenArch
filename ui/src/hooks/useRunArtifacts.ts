import { useMemo, useState } from "react";

import { fetchJSON } from "../lib/api";
import { dedupeArtifactsByPath, indexArtifactPath } from "../lib/runState";
import type { Artifact, ArtifactsResponse, FinalRunIndex } from "../lib/appContracts";

export function useRunArtifacts() {
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [selectedArtifact, setSelectedArtifact] = useState("");
  const [selectedArtifactContent, setSelectedArtifactContent] = useState("");
  const [coverageSummary, setCoverageSummary] = useState("");
  const [openQuestions, setOpenQuestions] = useState("");

  const diagramArtifacts = useMemo(() => {
    return artifacts
      .filter((artifact) => artifact.kind === "diagram" || artifact.kind === "diagram-index" || artifact.path.startsWith("reports/diagrams/"))
      .sort((left, right) => left.path.localeCompare(right.path));
  }, [artifacts]);

  const nonDiagramArtifacts = useMemo(() => {
    return artifacts
      .filter((artifact) => !(artifact.kind === "diagram" || artifact.kind === "diagram-index" || artifact.path.startsWith("reports/diagrams/")))
      .sort((left, right) => left.path.localeCompare(right.path));
  }, [artifacts]);

  const selectedArtifactIsMermaid = useMemo(() => {
    if (!selectedArtifact) {
      return false;
    }
    if (selectedArtifact.endsWith(".mmd")) {
      return true;
    }
    const text = selectedArtifactContent.trim();
    if (text.startsWith("flowchart") || text.startsWith("graph") || text.startsWith("sequenceDiagram") || text.startsWith("classDiagram")) {
      return true;
    }
    return text.includes("```mermaid");
  }, [selectedArtifact, selectedArtifactContent]);

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

  async function loadCoverageArtifacts() {
    await loadTextArtifact("reports/coverage/summary.md", setCoverageSummary);
    await loadTextArtifact("reports/coverage/open-questions.md", setOpenQuestions);
  }

  async function fetchArtifacts(id: string) {
    const payload = await fetchJSON<ArtifactsResponse>(`/api/pipeline/runs/${id}/artifacts`);
    const runArtifacts = payload.artifacts ?? [];
    const finalRunIndexPath = indexArtifactPath(runArtifacts, "/staging/final/final-run-index.json");
    if (!finalRunIndexPath) {
      applyArtifacts(runArtifacts);
      return;
    }

    try {
      const finalRunIndex = await fetchJSON<FinalRunIndex>(`/api/artifacts?path=${encodeURIComponent(finalRunIndexPath)}`);
      const canonicalArtifacts: Artifact[] = (finalRunIndex.canonical_documents ?? [])
        .map((document) => {
          const canonicalPath = String(document.canonical_path ?? "").trim();
          if (!canonicalPath) {
            return null;
          }
          return {
            path: canonicalPath,
            kind: String(document.kind ?? "report").trim() || "report",
            label: String(document.title ?? canonicalPath).trim() || canonicalPath,
          } satisfies Artifact;
        })
        .filter((artifact): artifact is Artifact => artifact !== null);

      const indexArtifacts: Artifact[] = [
        {
          path: finalRunIndexPath,
          kind: "taskrun",
          label: "Final Run Index",
        },
      ];
      const citationIndexPath = String(finalRunIndex.citation_index_path ?? "").trim();
      if (citationIndexPath.length > 0) {
        indexArtifacts.push({
          path: citationIndexPath,
          kind: "taskrun",
          label: "Citation Index",
        });
      }
      applyArtifacts(dedupeArtifactsByPath([...canonicalArtifacts, ...indexArtifacts]));
    } catch {
      applyArtifacts(runArtifacts);
    }
  }

  async function handleOpenArtifact(path: string) {
    setSelectedArtifact(path);
    setSelectedArtifactContent("Loading...");
    await loadTextArtifact(path, setSelectedArtifactContent);
  }

  function clearArtifacts() {
    setArtifacts([]);
    setSelectedArtifact("");
    setSelectedArtifactContent("");
    setCoverageSummary("");
    setOpenQuestions("");
  }

  function applyArtifacts(nextArtifacts: Artifact[]) {
    setArtifacts(nextArtifacts);

    if (!selectedArtifact) {
      if (nextArtifacts.length === 0) {
        setSelectedArtifactContent("");
      }
      return;
    }

    const selectedStillExists = nextArtifacts.some((artifact) => artifact.path === selectedArtifact);
    if (!selectedStillExists) {
      setSelectedArtifact("");
      setSelectedArtifactContent("");
    }
  }

  return {
    artifacts,
    selectedArtifact,
    selectedArtifactContent,
    coverageSummary,
    openQuestions,
    diagramArtifacts,
    nonDiagramArtifacts,
    selectedArtifactIsMermaid,
    fetchArtifacts,
    loadCoverageArtifacts,
    handleOpenArtifact,
    clearArtifacts,
  };
}
