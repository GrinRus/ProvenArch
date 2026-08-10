import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { GitDiffView } from "./GitDiffView";

describe("GitDiffView", () => {
  it("keeps unloaded diff state explicit", () => {
    render(<GitDiffView gitDiff={null} status="Loading full workspace diff" onSelectFile={() => undefined} />);
    expect(screen.getByText("Loading full workspace diff")).toBeInTheDocument();
  });

  it("renders selected file and hunk details", () => {
    render(<GitDiffView gitDiff={{ empty: false, files: [{ path: "docs/a.md", status: "modified", additions: 1, deletions: 0, binary: false }], folders: [], hunks: [{ header: "@@", lines: [{ kind: "added", old_line: null, new_line: 1, content: "+a" }] }], selected_file: { path: "docs/a.md", status: "modified", additions: 1, deletions: 0, binary: false }, selected_path: "docs/a.md", message: "" } as never} status="" onSelectFile={() => undefined} />);
    expect(screen.getByTestId("git-diff-view")).toHaveTextContent("docs/a.md");
    expect(screen.getByTestId("git-diff-hunks")).toHaveTextContent("+a");
  });
});
