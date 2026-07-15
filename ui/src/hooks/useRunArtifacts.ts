import { useCallback, useEffect, useMemo, useRef, useState } from "react";

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
  const [evidenceSnapshot, setEvidenceSnapshot] = useState<RunEvidenceSnapshot>(emptyEvidenceSnapshot());
  const artifactsRequest = useRequestGate("run-evidence-snapshot");
  const previewRequest = useRequestGate("run-artifact-preview");
  const contentByReadPath = useRef(new Map<string, string>());

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

  const loadTextArtifact = useCallback(async (path: string, signal?: AbortSignal): Promise<string> => {
    if (!path) {
      throw new Error("artifact path is required");
    }
    const cached = contentByReadPath.current.get(path);
    if (cached !== undefined) {
      return cached;
    }
    const response = await fetch(`/api/artifacts?path=${encodeURIComponent(path)}`, { signal });
    if (!response.ok) {
      throw new Error(`artifact ${path} is unavailable (${response.status})`);
    }
    const content = await response.text();
    contentByReadPath.current.set(path, content);
    return content;
  }, []);

  async function fetchArtifacts(id: string) {
    const token = artifactsRequest.begin(id);
    setEvidenceSnapshot({ ...emptyEvidenceSnapshot(), runId: id, status: "loading" });
    contentByReadPath.current = new Map();
    try {
      const snapshot = await fetchRunSnapshotIndex(id, token.signal);
      if (!artifactsRequest.isCurrent(token)) {
        return;
      }
      if (!snapshot) {
        applyEvidenceSnapshot({
          runId: id,
          sourceMode: "run_snapshot",
          status: "not_produced",
          artifacts: [],
          coverageSummary: "",
          openQuestions: "",
          issues: [{ code: "snapshot_not_produced", message: `Run ${id} has no final snapshot index.` }],
        });
        return;
      }
      const nextArtifacts = dedupeArtifactsByPath([...snapshot.canonicalArtifacts, ...snapshot.indexArtifacts]);
      const issues: EvidenceIssue[] = [];
      await Promise.all(
        nextArtifacts.map(async (artifact) => {
          const readPath = artifact.read_path ?? artifact.path;
          try {
            await loadTextArtifact(readPath, token.signal);
          } catch (error) {
            if (isAbortError(error)) {
              throw error;
            }
            issues.push({ code: "indexed_artifact_unavailable", path: artifact.path, message: error instanceof Error ? error.message : String(error) });
          }
        }),
      );
      if (!artifactsRequest.isCurrent(token)) {
        return;
      }
      const coveragePath = snapshot.readPathByCanonicalPath.get("reports/coverage/summary.md");
      const questionsPath = snapshot.readPathByCanonicalPath.get("reports/coverage/open-questions.md");
      applyEvidenceSnapshot({
        runId: id,
        sourceMode: "run_snapshot",
        status: issues.length > 0 ? "partial" : "available",
        artifacts: nextArtifacts,
        coverageSummary: coveragePath ? contentByReadPath.current.get(coveragePath) ?? "" : "",
        openQuestions: questionsPath ? contentByReadPath.current.get(questionsPath) ?? "" : "",
        issues,
      });
    } catch (error) {
      if (isAbortError(error) || !artifactsRequest.isCurrent(token)) {
        return;
      }
      applyEvidenceSnapshot({
        runId: id,
        sourceMode: "run_snapshot",
        status: "error",
        artifacts: [],
        coverageSummary: "",
        openQuestions: "",
        issues: [{ code: "snapshot_load_failed", message: error instanceof Error ? error.message : String(error) }],
      });
    } finally {
      artifactsRequest.finish(token);
    }
  }

  const handleOpenArtifact = useCallback(async (path: string) => {
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
      setSelectedArtifactContent(error instanceof Error ? `Artifact unavailable: ${error.message}` : "Artifact unavailable.");
    } finally {
      previewRequest.finish(token);
    }
  }, [loadTextArtifact, previewRequest]);

  function clearArtifacts() {
    artifactsRequest.abort();
    previewRequest.abort();
    contentByReadPath.current = new Map();
    setArtifacts([]);
    setSelectedArtifact("");
    setSelectedArtifactContent("");
    setCoverageSummary("");
    setOpenQuestions("");
    setEvidenceSnapshot(emptyEvidenceSnapshot());
  }

  function applyEvidenceSnapshot(snapshot: RunEvidenceSnapshot) {
    setEvidenceSnapshot(snapshot);
    setCoverageSummary(snapshot.coverageSummary);
    setOpenQuestions(snapshot.openQuestions);
    applyArtifacts(snapshot.artifacts);
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
    evidenceSnapshot,
    diagramArtifacts,
    nonDiagramArtifacts,
    selectedArtifactIsMermaid,
    fetchArtifacts,
    handleOpenArtifact,
    clearArtifacts,
  };
}

export type EvidenceSourceMode = "run_snapshot" | "current_workspace";
export type EvidenceSnapshotStatus = "idle" | "loading" | "available" | "partial" | "not_produced" | "unavailable" | "error";
export type EvidenceIssue = { code: string; message: string; path?: string };
export type RunEvidenceSnapshot = {
  runId: string | null;
  sourceMode: EvidenceSourceMode;
  status: EvidenceSnapshotStatus;
  artifacts: Artifact[];
  coverageSummary: string;
  openQuestions: string;
  issues: EvidenceIssue[];
};

function emptyEvidenceSnapshot(): RunEvidenceSnapshot {
  return {
    runId: null,
    sourceMode: "current_workspace",
    status: "idle",
    artifacts: [],
    coverageSummary: "",
    openQuestions: "",
    issues: [],
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

  const finalRunIndex = await fetchJSON<FinalRunIndex>(`/api/artifacts?path=${encodeURIComponent(finalRunIndexPath)}`, { signal });
  return buildRunSnapshotIndex(id, finalRunIndexPath, finalRunIndex);
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
        id: String(document.id ?? "").trim() || undefined,
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
