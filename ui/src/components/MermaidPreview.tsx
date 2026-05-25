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
        setSVG(rendered.svg);
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
  return <div className="diagram-svg" dangerouslySetInnerHTML={{ __html: svg }} />;
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
