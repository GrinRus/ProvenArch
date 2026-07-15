import { useState, type ReactNode } from "react";
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
};

export function EvidenceViewer({ path, content, runId, sourceMode, demo, diff, onOpenArtifact }: EvidenceViewerProps) {
  const [view, setView] = useState<"rendered" | "raw" | "diff">("rendered");
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

function isLocalEvidenceLink(href: string): boolean {
  const normalized = href.trim();
  return normalized !== "" && !/^[a-z][a-z0-9+.-]*:/i.test(normalized) && !normalized.startsWith("#") && !normalized.startsWith("//");
}

function normalizeLocalEvidenceLink(href: string): string {
  return href.split("#", 1)[0].replace(/^\.\//, "");
}
