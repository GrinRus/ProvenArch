import { useEffect, useState, type ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { MermaidPreview } from "./MermaidPreview";
import { TabNav, tabPanelProps } from "./TabNav";

type EvidenceViewerProps = {
  path: string;
  content: string;
  runId?: string | null;
  sourceMode: "run_snapshot" | "current_workspace";
  demo?: boolean;
  diff?: ReactNode;
  onOpenArtifact?: (path: string) => void;
  mode?: "rendered" | "raw" | "diff";
  onModeChange?: (mode: "rendered" | "raw" | "diff") => void;
};

export function EvidenceViewer({ path, content, runId, sourceMode, demo, diff, onOpenArtifact, mode, onModeChange }: EvidenceViewerProps) {
  const [localView, setLocalView] = useState<"rendered" | "raw" | "diff">(() => modeFromLocation(diff));
  const view = mode ?? localView;
  const setView = (next: "rendered" | "raw" | "diff") => {
    setLocalView(next);
    onModeChange?.(next);
    if (!mode && typeof window !== "undefined") {
      const url = new URL(window.location.href);
      url.searchParams.set("mode", next);
      window.history.pushState({}, "", `${url.pathname}${url.search}`);
    }
  };
  useEffect(() => {
    if (mode) return;
    const restore = () => setLocalView(modeFromLocation(diff));
    window.addEventListener("popstate", restore);
    return () => window.removeEventListener("popstate", restore);
  }, [diff, mode]);
  const options = [
    { id: "rendered" as const, label: "Rendered" },
    { id: "raw" as const, label: "Raw" },
    ...(diff ? [{ id: "diff" as const, label: "Diff" }] : []),
  ];

  return (
    <section className="evidence-viewer" data-testid="evidence-viewer">
      <header className="evidence-source-header">
        <div><strong>{path || "Evidence"}</strong><span>{sourceMode === "run_snapshot" ? `Run snapshot · ${runId ?? "unknown run"}` : "Current workspace"}</span></div>
        <span className={demo ? "status warn" : "status ok"}>{demo ? "Demo evidence" : "Live evidence"}</span>
      </header>
      <TabNav ariaLabel="Evidence views" idBase="evidence-viewer" value={view} onChange={setView} options={options} />
      <div {...tabPanelProps("evidence-viewer", view)}>
        {view === "rendered" && path.endsWith(".mmd") ? <MermaidPreview source={content} title={path || "Mermaid diagram"} /> : null}
        {view === "rendered" && !path.endsWith(".mmd") ? (
          <div className="markdown-evidence">
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                a: ({ href, children }) => {
                  const target = href ?? "";
                  if (isLocalEvidenceLink(target)) {
                    return <button type="button" className="evidence-link" onClick={() => onOpenArtifact?.(normalizeLocalEvidenceLink(target))}>{children}</button>;
                  }
                  return <a href={target} target="_blank" rel="noreferrer">{children}<span className="sr-only"> (external link)</span></a>;
                },
                code: ({ className, children }) => {
                  const language = /language-([^\s]+)/.exec(className ?? "")?.[1];
                  const source = String(children).replace(/\n$/, "");
                  return language === "mermaid" ? <MermaidPreview source={source} title={path || "Mermaid diagram"} /> : <code className={className}>{children}</code>;
                },
              }}
            >
              {content || "No evidence content is available."}
            </ReactMarkdown>
          </div>
        ) : null}
        {view === "raw" ? <pre data-testid="evidence-raw">{content || "No evidence content is available."}</pre> : null}
        {view === "diff" ? diff : null}
      </div>
    </section>
  );
}

function modeFromLocation(diff: ReactNode): "rendered" | "raw" | "diff" {
  if (typeof window === "undefined") return "rendered";
  const value = new URLSearchParams(window.location.search).get("mode");
  if (value === "raw" || (value === "diff" && diff)) return value;
  return "rendered";
}

function isLocalEvidenceLink(href: string): boolean {
  const normalized = href.trim();
  return normalized !== "" && !/^[a-z][a-z0-9+.-]*:/i.test(normalized) && !normalized.startsWith("#") && !normalized.startsWith("//");
}

function normalizeLocalEvidenceLink(href: string): string {
  return href.split("#", 1)[0].replace(/^\.\//, "");
}
