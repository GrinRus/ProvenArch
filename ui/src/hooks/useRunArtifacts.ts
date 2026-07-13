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
    if (!path) {
      setter("");
      return;
    }
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

  async function loadCoverageArtifacts(id?: string) {
    const snapshot = id ? await fetchRunSnapshotIndex(id) : null;
    if (snapshot) {
      await loadTextArtifact(snapshot.readPathByCanonicalPath.get("reports/coverage/summary.md") ?? "", setCoverageSummary);
      await loadTextArtifact(snapshot.readPathByCanonicalPath.get("reports/coverage/open-questions.md") ?? "", setOpenQuestions);
      return;
    }
    await loadTextArtifact("reports/coverage/summary.md", setCoverageSummary);
    await loadTextArtifact("reports/coverage/open-questions.md", setOpenQuestions);
  }

  async function fetchArtifacts(id: string) {
    const snapshot = await fetchRunSnapshotIndex(id);
    if (!snapshot) {
      const payload = await fetchJSON<ArtifactsResponse>(`/api/pipeline/runs/${id}/artifacts`);
      applyArtifacts(payload.artifacts ?? []);
      return;
    }

    applyArtifacts(dedupeArtifactsByPath([...snapshot.canonicalArtifacts, ...snapshot.indexArtifacts]));
  }

  async function handleOpenArtifact(path: string) {
    setSelectedArtifact(path);
    setSelectedArtifactContent("Loading...");
    const artifact = artifacts.find((item) => item.path === path);
    await loadTextArtifact(artifact?.read_path ?? path, setSelectedArtifactContent);
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

type RunSnapshotIndex = {
  canonicalArtifacts: Artifact[];
  indexArtifacts: Artifact[];
  readPathByCanonicalPath: Map<string, string>;
};

async function fetchRunSnapshotIndex(id: string): Promise<RunSnapshotIndex | null> {
  const payload = await fetchJSON<ArtifactsResponse>(`/api/pipeline/runs/${id}/artifacts`);
  const runArtifacts = payload.artifacts ?? [];
  const finalRunIndexPath = indexArtifactPath(runArtifacts, "/staging/final/final-run-index.json");
  if (!finalRunIndexPath) {
    return null;
  }

  try {
    const finalRunIndex = await fetchJSON<FinalRunIndex>(`/api/artifacts?path=${encodeURIComponent(finalRunIndexPath)}`);
    return buildRunSnapshotIndex(id, finalRunIndexPath, finalRunIndex);
  } catch {
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
