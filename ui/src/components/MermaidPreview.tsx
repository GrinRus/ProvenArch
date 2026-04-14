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
        const rendered = await mermaid.default.render(`diagram-${Date.now()}-${Math.random()}`, graph);
        if (disposed || requestRef.current !== currentRequest) {
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

function extractMermaidGraph(content: string): string {
  const trimmed = content.trim();
  const blockMatch = trimmed.match(/```mermaid\s*([\s\S]*?)```/i);
  if (blockMatch && typeof blockMatch[1] === "string") {
    return blockMatch[1].trim();
  }
  return trimmed;
}
