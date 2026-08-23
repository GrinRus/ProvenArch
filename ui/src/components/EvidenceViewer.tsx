import { useEffect, useState, type ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { MermaidPreview } from "./MermaidPreview";
import { TabNav, tabPanelProps } from "./TabNav";

type EvidenceViewerProps = {
  path: string;
  content: string;
  runId?: string | null;
  sourceMode: "run_snapshot" | "promoted_current";
  demo?: boolean;
  provenance?: "demo" | "live" | "unknown";
  status?: "available" | "partial" | "unavailable" | "error";
  issues?: Array<{ code: string; message: string; path?: string }>;
  diff?: ReactNode;
  diffIdentity?: { left: string; right: string };
  onOpenArtifact?: (path: string) => void;
  mode?: "rendered" | "raw" | "diff";
  onModeChange?: (mode: "rendered" | "raw" | "diff") => void;
};

export function EvidenceViewer({ path, content, runId, sourceMode, demo, provenance, status = "available", issues = [], diff, diffIdentity, onOpenArtifact, mode, onModeChange }: EvidenceViewerProps) {
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
  const evidenceProvenance = provenance ?? (demo === true ? "demo" : demo === false ? "live" : "unknown");
  const provenanceLabel = evidenceProvenance === "demo" ? "Demo" : evidenceProvenance === "live" ? "Live" : "Unknown";
  const localLinkIssue = (target: string) => {
    const resolved = resolveLocalEvidenceLink(path, target);
    if (!resolved) return null;
    if ((sourceMode === "promoted_current" || sourceMode === "run_snapshot") &&
        (resolved === "reports/taskruns" || resolved.startsWith("reports/taskruns/"))) return null;
    return resolved;
  };
  const exceedsRenderBudget = content.length > 512 * 1024;

  return (
    <section className="evidence-viewer" data-testid="evidence-viewer">
      <header className="evidence-source-header">
        <div><strong>{path || "Evidence"}</strong><span>{sourceMode === "run_snapshot" ? `Run snapshot · ${runId ?? "unknown run"}` : "Current workspace"}</span></div>
        <span className={evidenceProvenance === "demo" ? "status warn" : evidenceProvenance === "live" ? "status ok" : "status"}>{provenanceLabel} evidence</span>
      </header>
      {status !== "available" ? <p className={status === "partial" ? "status warn" : "status err"} role="status">Evidence state: {status}.</p> : null}
      {issues.length > 0 ? (
        <ul className="evidence-issues" aria-label="Evidence issues">
          {issues.map((issue) => <li key={`${issue.code}:${issue.path ?? ""}:${issue.message}`}><code>{issue.code}</code>: {issue.message}</li>)}
        </ul>
      ) : null}
      {diff && diffIdentity ? <p className="hint" data-testid="evidence-diff-identity">Diff: {diffIdentity.left} → {diffIdentity.right}</p> : null}
      <TabNav ariaLabel="Evidence views" idBase="evidence-viewer" value={view} onChange={setView} options={options} />
      <div {...tabPanelProps("evidence-viewer", view)}>
        {view === "rendered" && exceedsRenderBudget ? <p className="status warn">Rendered preview is disabled above 512 KiB. Use Raw for the bounded text response.</p> : null}
        {view === "rendered" && !exceedsRenderBudget && path.endsWith(".mmd") ? <MermaidPreview source={content} title={path || "Mermaid diagram"} /> : null}
        {view === "rendered" && !exceedsRenderBudget && !path.endsWith(".mmd") ? (
          <div className="markdown-evidence">
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                h1: ({ children }) => <h2>{children}</h2>,
                a: ({ href, children }) => {
                  const target = href ?? "";
                  if (isLocalEvidenceLink(target)) {
                    const resolved = localLinkIssue(target);
                    if (!resolved) {
                      return <span className="status err" title="Link escapes or is invalid for the selected evidence authority">{children}</span>;
                    }
                    return <button type="button" className="evidence-link" onClick={() => onOpenArtifact?.(resolved)}>{children}</button>;
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

export function resolveLocalEvidenceLink(basePath: string, href: string): string | null {
  const raw = href.split(/[?#]/, 1)[0].trim().replace(/\\/g, "/");
  if (!raw || raw.startsWith("/") || raw.includes("\0")) return null;
  const baseParts = basePath.replace(/\\/g, "/").split("/").slice(0, -1);
  const stack = raw.startsWith("./") || !raw.includes(":") ? baseParts : [];
  for (const part of raw.split("/")) {
    if (!part || part === ".") continue;
    if (part === "..") {
      if (stack.length === 0) return null;
      stack.pop();
      continue;
    }
    stack.push(part);
  }
  return stack.length > 0 ? stack.join("/") : null;
}
