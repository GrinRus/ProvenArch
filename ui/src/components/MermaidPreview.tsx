import { useEffect, useRef, useState } from "react";

type MermaidPreviewProps = {
  source: string;
  title: string;
};

export function MermaidPreview(props: MermaidPreviewProps) {
  const { source, title } = props;
  const [svg, setSVG] = useState<string>("");
  const [error, setError] = useState<string>("");
  const requestRef = useRef(0);

  useEffect(() => {
    const trimmed = source.trim();
    if (!trimmed) {
      setSVG("");
      setError("Diagram content is empty.");
      return;
    }

    const currentRequest = requestRef.current + 1;
    requestRef.current = currentRequest;
    let disposed = false;

    void (async () => {
      try {
        const mermaid = await import("mermaid");
        mermaid.default.initialize({
          startOnLoad: false,
          securityLevel: "strict",
          theme: "base",
        });
        const graph = extractMermaidGraph(trimmed);
        if (!isSupportedMermaidGraph(graph)) {
          setSVG("");
          setError("Diagram content is not a supported Mermaid graph.");
          return;
        }
        const renderID = createMermaidRenderID();
        const rendered = await renderMermaid(mermaid.default, renderID, graph);
        if (disposed || requestRef.current !== currentRequest) {
          return;
        }
        if (isMermaidErrorSVG(rendered.svg)) {
          setSVG("");
          setError("Mermaid reported a diagram syntax error.");
          return;
        }
        const safeSVG = sanitizeMermaidSVG(rendered.svg);
        if (!safeSVG) {
          setSVG("");
          setError("Mermaid returned an unsafe or invalid SVG.");
          return;
        }
        setSVG(safeSVG);
        setError("");
      } catch (renderErr) {
        if (disposed || requestRef.current !== currentRequest) {
          return;
        }
        setSVG("");
        setError(renderErr instanceof Error ? renderErr.message : "Mermaid rendering failed");
      }
    })();

    return () => {
      disposed = true;
    };
  }, [source]);

  if (error) {
    return <p className="status err">Diagram render error: {error}</p>;
  }
  if (!svg) {
    return <p className="hint">Rendering {title}...</p>;
  }
  return <div className="diagram-svg"><img src={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`} alt={title} /></div>;
}

const mermaidGraphStartPattern =
  /^(flowchart|graph|sequenceDiagram|classDiagram|stateDiagram|erDiagram|gantt|journey|pie|gitGraph|mindmap|timeline|quadrantChart|requirementDiagram|c4Context|c4Container|block-beta|architecture-beta)\b/i;

function extractMermaidGraph(content: string): string {
  const trimmed = content.trim();
  const blockMatch = trimmed.match(/```mermaid\s*([\s\S]*?)```/i);
  if (blockMatch && typeof blockMatch[1] === "string") {
    return blockMatch[1].trim();
  }
  return trimmed;
}

function isSupportedMermaidGraph(graph: string): boolean {
  return mermaidGraphStartPattern.test(graph.trim());
}

function isMermaidErrorSVG(svg: string): boolean {
  return /aria-roledescription=["']error["']/.test(svg) || svg.includes("Syntax error in text");
}

function sanitizeMermaidSVG(svg: string): string {
  if (typeof DOMParser === "undefined" || typeof XMLSerializer === "undefined") return "";
  const document = new DOMParser().parseFromString(svg, "image/svg+xml");
  const root = document.documentElement;
  if (!root || root.tagName.toLowerCase() !== "svg" || document.querySelector("parsererror")) return "";
  document.querySelectorAll("script, foreignObject").forEach((node) => node.remove());
  document.querySelectorAll("*").forEach((element) => {
    for (const attribute of Array.from(element.attributes)) {
      const name = attribute.name.toLowerCase();
      const value = attribute.value.trim().toLowerCase();
      if (name.startsWith("on") || ((name === "href" || name.endsWith(":href")) && value.startsWith("javascript:"))) {
        element.removeAttribute(attribute.name);
      }
    }
  });
  return new XMLSerializer().serializeToString(root);
}

function cleanupMermaidScratch(renderID: string) {
  document.getElementById(`d${renderID}`)?.remove();
}

async function renderMermaid(mermaid: { render: (id: string, graph: string) => Promise<{ svg: string }> }, renderID: string, graph: string) {
  try {
    return await mermaid.render(renderID, graph);
  } finally {
    cleanupMermaidScratch(renderID);
  }
}

function createMermaidRenderID(): string {
  return `diagram-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
