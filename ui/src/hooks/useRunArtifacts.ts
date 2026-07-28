import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { dedupeArtifactsByPath } from "../lib/runState";
import type { Artifact } from "../lib/appContracts";
import { getPipelineRunSnapshot } from "../lib/runApi";
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
  const evidenceSnapshotRef = useRef(evidenceSnapshot);
  const artifactsRequest = useRequestGate("run-evidence-snapshot");
  const previewRequest = useRequestGate("run-artifact-preview");
  const contentByReadPath = useRef(new Map<string, string>());

  useEffect(() => {
    artifactsRef.current = artifacts;
  }, [artifacts]);

  useEffect(() => {
    selectedArtifactRef.current = selectedArtifact;
  }, [selectedArtifact]);

  useEffect(() => {
    evidenceSnapshotRef.current = evidenceSnapshot;
  }, [evidenceSnapshot]);

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
      const snapshot = await getPipelineRunSnapshot(id, { signal: token.signal });
      if (!artifactsRequest.isCurrent(token)) {
        return;
      }
      if (snapshot.status === "not_produced" || snapshot.status === "unavailable" || snapshot.status === "error") {
        applyEvidenceSnapshot({
          runId: id,
          sourceMode: "run_snapshot",
          status: snapshot.status,
          artifacts: [],
          coverageSummary: "",
          openQuestions: "",
          issues: snapshot.issues,
        });
        return;
      }
      const nextArtifacts = dedupeArtifactsByPath(snapshot.artifacts);
      const issues: EvidenceIssue[] = [...snapshot.issues];
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
      const coveragePath = nextArtifacts.find((artifact) => artifact.canonical_path === "reports/coverage/summary.md")?.read_path;
      const questionsPath = nextArtifacts.find((artifact) => artifact.canonical_path === "reports/coverage/open-questions.md")?.read_path;
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

  const handleOpenArtifact = useCallback(async (path: string, viewerMode = "rendered") => {
    const artifact = artifactsRef.current.find((item) => item.path === path);
    if (!artifact) {
      previewRequest.abort();
      setSelectedArtifact(path);
      setSelectedArtifactContent("Artifact unavailable: the link is outside the selected run snapshot inventory.");
      return;
    }
    const readPath = artifact.read_path ?? artifact.path;
    const snapshot = evidenceSnapshotRef.current;
    const token = previewRequest.begin([
      snapshot.runId ?? "",
      snapshot.sourceMode,
      path,
      readPath,
      viewerMode,
    ].join("|"));
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

export type EvidenceSourceMode = "run_snapshot" | "promoted_current";
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
    sourceMode: "promoted_current",
    status: "idle",
    artifacts: [],
    coverageSummary: "",
    openQuestions: "",
    issues: [],
  };
}
