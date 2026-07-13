import { useEffect, useMemo, useRef, useState } from "react";

import { fetchJSON } from "../lib/api";
import { dedupeArtifactsByPath, indexArtifactPath } from "../lib/runState";
import type { Artifact, ArtifactsResponse, FinalRunIndex } from "../lib/appContracts";
import { isAbortError, useRequestGate } from "./useRequestGate";

export function useRunArtifacts() {
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [selectedArtifact, setSelectedArtifact] = useState("");
  const [selectedArtifactContent, setSelectedArtifactContent] = useState("");
  const [coverageSummary, setCoverageSummary] = useState("");
  const [openQuestions, setOpenQuestions] = useState("");
  const artifactsRef = useRef(artifacts);
  const selectedArtifactRef = useRef(selectedArtifact);
  const artifactsRequest = useRequestGate("run-artifacts");
  const coverageRequest = useRequestGate("run-coverage");
  const previewRequest = useRequestGate("run-artifact-preview");

  useEffect(() => {
    artifactsRef.current = artifacts;
  }, [artifacts]);

  useEffect(() => {
    selectedArtifactRef.current = selectedArtifact;
  }, [selectedArtifact]);

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

  async function loadTextArtifact(path: string, signal?: AbortSignal): Promise<string> {
    if (!path) {
      return "";
    }
    try {
      const response = await fetch(`/api/artifacts?path=${encodeURIComponent(path)}`, { signal });
      if (!response.ok) {
        return "";
      }
      return response.text();
    } catch (error) {
      if (isAbortError(error)) {
        throw error;
      }
      return "";
    }
  }

  async function loadCoverageArtifacts(id?: string) {
    const token = coverageRequest.begin(id ?? "current-workspace");
    try {
      const snapshot = id ? await fetchRunSnapshotIndex(id, token.signal) : null;
      const summaryPath = snapshot ? snapshot.readPathByCanonicalPath.get("reports/coverage/summary.md") ?? "" : "reports/coverage/summary.md";
      const questionsPath = snapshot
        ? snapshot.readPathByCanonicalPath.get("reports/coverage/open-questions.md") ?? ""
        : "reports/coverage/open-questions.md";
      const [summary, questions] = await Promise.all([
        loadTextArtifact(summaryPath, token.signal),
        loadTextArtifact(questionsPath, token.signal),
      ]);
      if (!coverageRequest.isCurrent(token)) {
        return;
      }
      setCoverageSummary(summary);
      setOpenQuestions(questions);
    } catch (error) {
      if (isAbortError(error) || !coverageRequest.isCurrent(token)) {
        return;
      }
      throw error;
    } finally {
      coverageRequest.finish(token);
    }
  }

  async function fetchArtifacts(id: string) {
    const token = artifactsRequest.begin(id);
    try {
      const snapshot = await fetchRunSnapshotIndex(id, token.signal);
      if (!artifactsRequest.isCurrent(token)) {
        return;
      }
      if (!snapshot) {
        const payload = await fetchJSON<ArtifactsResponse>(`/api/pipeline/runs/${id}/artifacts`, { signal: token.signal });
        if (!artifactsRequest.isCurrent(token)) {
          return;
        }
        applyArtifacts(payload.artifacts ?? []);
        return;
      }

      applyArtifacts(dedupeArtifactsByPath([...snapshot.canonicalArtifacts, ...snapshot.indexArtifacts]));
    } catch (error) {
      if (isAbortError(error) || !artifactsRequest.isCurrent(token)) {
        return;
      }
      throw error;
    } finally {
      artifactsRequest.finish(token);
    }
  }

  async function handleOpenArtifact(path: string) {
    const artifact = artifactsRef.current.find((item) => item.path === path);
    const readPath = artifact?.read_path ?? path;
    const token = previewRequest.begin(`${path}|${readPath}`);
    setSelectedArtifact(path);
    setSelectedArtifactContent("Loading...");
    try {
      const content = await loadTextArtifact(readPath, token.signal);
      if (!previewRequest.isCurrent(token)) {
        return;
      }
      setSelectedArtifactContent(content);
    } catch (error) {
      if (isAbortError(error) || !previewRequest.isCurrent(token)) {
        return;
      }
      setSelectedArtifactContent("");
    } finally {
      previewRequest.finish(token);
    }
  }

  function clearArtifacts() {
    artifactsRequest.abort();
    coverageRequest.abort();
    previewRequest.abort();
    setArtifacts([]);
    setSelectedArtifact("");
    setSelectedArtifactContent("");
    setCoverageSummary("");
    setOpenQuestions("");
  }

  function applyArtifacts(nextArtifacts: Artifact[]) {
    setArtifacts(nextArtifacts);

    const currentSelectedArtifact = selectedArtifactRef.current;
    if (!currentSelectedArtifact) {
      if (nextArtifacts.length === 0) {
        setSelectedArtifactContent("");
      }
      return;
    }

    const selectedStillExists = nextArtifacts.some((artifact) => artifact.path === currentSelectedArtifact);
    if (!selectedStillExists) {
      previewRequest.abort();
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

type RunSnapshotIndex = {
  canonicalArtifacts: Artifact[];
  indexArtifacts: Artifact[];
  readPathByCanonicalPath: Map<string, string>;
};

async function fetchRunSnapshotIndex(id: string, signal?: AbortSignal): Promise<RunSnapshotIndex | null> {
  const payload = await fetchJSON<ArtifactsResponse>(`/api/pipeline/runs/${id}/artifacts`, { signal });
  const runArtifacts = payload.artifacts ?? [];
  const finalRunIndexPath = indexArtifactPath(runArtifacts, "/staging/final/final-run-index.json");
  if (!finalRunIndexPath) {
    return null;
  }

  try {
    const finalRunIndex = await fetchJSON<FinalRunIndex>(`/api/artifacts?path=${encodeURIComponent(finalRunIndexPath)}`, { signal });
    return buildRunSnapshotIndex(id, finalRunIndexPath, finalRunIndex);
  } catch (error) {
    if (isAbortError(error)) {
      throw error;
    }
    return {
      canonicalArtifacts: [],
      indexArtifacts: [
        {
          path: finalRunIndexPath,
          read_path: finalRunIndexPath,
          kind: "taskrun",
          label: "Final run index",
          source_run_id: id,
          source_mode: "run_snapshot",
        },
      ],
      readPathByCanonicalPath: new Map(),
    };
  }
}

function buildRunSnapshotIndex(id: string, finalRunIndexPath: string, finalRunIndex: FinalRunIndex): RunSnapshotIndex {
  const indexRunID = String(finalRunIndex.run_id ?? "").trim();
  if (indexRunID && indexRunID !== id) {
    throw new Error(`final run index run_id ${indexRunID} does not match selected run ${id}`);
  }

  const readPathByCanonicalPath = new Map<string, string>();
  const canonicalArtifacts: Artifact[] = (finalRunIndex.canonical_documents ?? [])
    .map((document) => {
      const canonicalPath = String(document.canonical_path ?? "").trim();
      const stagedPath = String(document.staged_path ?? "").trim();
      if (!canonicalPath || !stagedPath) {
        throw new Error("final run index document is missing canonical_path or staged_path");
      }
      if (!isRunFinalStagedPath(id, stagedPath)) {
        throw new Error(`final run index staged_path ${stagedPath} is outside selected run snapshot`);
      }
      readPathByCanonicalPath.set(canonicalPath, stagedPath);
      return {
        path: canonicalPath,
        read_path: stagedPath,
        canonical_path: canonicalPath,
        kind: String(document.kind ?? "report").trim() || "report",
        label: String(document.title ?? canonicalPath).trim() || canonicalPath,
        source_run_id: id,
        source_mode: "run_snapshot",
      } satisfies Artifact;
    });

  const indexArtifacts: Artifact[] = [
    {
      path: finalRunIndexPath,
      read_path: finalRunIndexPath,
      kind: "taskrun",
      label: "Final run index",
      source_run_id: id,
      source_mode: "run_snapshot",
    },
  ];
  const citationIndexPath = String(finalRunIndex.citation_index_path ?? "").trim();
  if (citationIndexPath.length > 0) {
    if (!isRunFinalStagedPath(id, citationIndexPath)) {
      throw new Error(`citation_index_path ${citationIndexPath} is outside selected run snapshot`);
    }
    indexArtifacts.push({
      path: citationIndexPath,
      read_path: citationIndexPath,
      kind: "taskrun",
      label: "Citation index",
      source_run_id: id,
      source_mode: "run_snapshot",
    });
  }

  return { canonicalArtifacts, indexArtifacts, readPathByCanonicalPath };
}

function isRunFinalStagedPath(runID: string, path: string): boolean {
  return path === `reports/taskruns/${runID}/staging/final/final-run-index.json` || path.startsWith(`reports/taskruns/${runID}/staging/final/`);
}
