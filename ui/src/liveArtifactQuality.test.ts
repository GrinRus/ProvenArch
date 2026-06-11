import { describe, expect, it } from "vitest";

import { evaluateDiagramArtifactReadability } from "./liveArtifactQuality";

describe("evaluateDiagramArtifactReadability", () => {
  it("accepts concrete internal C4 content that also names external evidence gaps", () => {
    const diagram = `flowchart LR
  System["Workspace System"]
  GapExternal["Gap: no evidence-backed external systems"]
  System -.-> GapExternal
  GapActors["Gap: no evidence-backed actors"]
  GapActors -.-> System
  subgraph InternalContext["Evidence-backed workspace internals"]
    ctx_svc_posthog["Service: PostHog"]
    ctx_svc_posthog_backend["Service: PostHog Django Backend"]
  end
  ctx_svc_posthog -->|contains| ctx_svc_posthog_backend`;

    expect(evaluateDiagramArtifactReadability(diagram)).toEqual({
      hasMermaidSyntax: true,
      hasConcreteEvidence: true,
      hasGapNote: true,
    });
  });

  it("rejects gap-only C4 content with no concrete evidence lines", () => {
    const diagram = `flowchart LR
  System["Workspace System"]
  GapExternal["Gap: no evidence-backed external systems"]
  System -.-> GapExternal
  GapActors["Gap: no evidence-backed actors"]
  GapActors -.-> System`;

    expect(evaluateDiagramArtifactReadability(diagram)).toEqual({
      hasMermaidSyntax: true,
      hasConcreteEvidence: false,
      hasGapNote: true,
    });
  });
});
