import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { SourceStagePanel } from "./SetupStagePanels";

describe("SetupStagePanels", () => {
  it("keeps the source setup boundary explicit", () => {
    render(
      <SourceStagePanel
        busy={false}
        guidedRepos={[]}
        guidedDocsImportsPath=""
        manifestContent=""
        manifestStatus=""
        validateResult={null}
        validationDiagnosticsByRepo={[]}
        doctorResult={null}
        doctorStatus=""
        setupRuntime="fake"
        setupRuntimeProvider="claude-code"
        onRepoChange={vi.fn()}
        onAddRepo={vi.fn()}
        onRemoveRepo={vi.fn()}
        onDocsImportsPathChange={vi.fn()}
        onApplyGuidedWorkspaceSetup={vi.fn()}
        onSaveGuidedWorkspaceSetup={vi.fn()}
        onManifestChange={vi.fn()}
        onSaveManifest={vi.fn()}
      />,
    );
    expect(screen.getByTestId("workspace-panel")).toBeInTheDocument();
    expect(screen.getByText("Repository sources")).toBeInTheDocument();
  });
});
