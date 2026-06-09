const MERMAID_DIAGRAM_RE = /^(flowchart|graph)\b|```mermaid/i;
const GAP_NOTE_RE = /Gap: no evidence-backed (services|containers|relationships|actors|external systems)/i;
const CONCRETE_NODE_RE = /\b(Service|Datastore|External|Actor):/i;
const CONCRETE_RELATION_RE = /(-->|---|\buses\b|\bcalls\b|\bcontains\b|\bdepends_on\b)/i;

export type DiagramArtifactReadability = {
  hasMermaidSyntax: boolean;
  hasConcreteEvidence: boolean;
  hasGapNote: boolean;
};

export function evaluateDiagramArtifactReadability(content: string): DiagramArtifactReadability {
  const text = content.trim();
  const nonGapLines = text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line !== "" && !GAP_NOTE_RE.test(line));
  return {
    hasMermaidSyntax: MERMAID_DIAGRAM_RE.test(text),
    hasConcreteEvidence: nonGapLines.some((line) => CONCRETE_NODE_RE.test(line) || CONCRETE_RELATION_RE.test(line)),
    hasGapNote: GAP_NOTE_RE.test(text),
  };
}
