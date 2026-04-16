import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";

import App from "./App";

type MockJSON = Record<string, unknown>;

type FetchMockState = {
  runID?: string;
  runStarted?: boolean;
  runLogs?: Record<string, MockJSON>;
  runStatus?: Record<string, MockJSON>;
  runArtifacts?: Record<string, MockJSON>;
  artifactText?: Record<string, string>;
};

function jsonResponse(body: MockJSON, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function textResponse(body: string, status = 200): Response {
  return new Response(body, {
    status,
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
}

function createFetchMock(state: FetchMockState = {}) {
  const runID = state.runID ?? "run-1";
  const runLogs = state.runLogs ?? {
    [runID]: {
      run_id: runID,
      items: [],
      next_cursor: 0,
      eof: true,
    },
  };
  const runStatus = state.runStatus ?? {
    [runID]: {
      run_id: runID,
      pipeline: "init",
      status: "succeeded",
      started_at: "2026-04-03T12:00:00Z",
      finished_at: "2026-04-03T12:00:02Z",
      warnings: [],
      error_code: null,
      error: null,
    },
  };
  const runArtifacts = state.runArtifacts ?? {
    [runID]: {
      run_id: runID,
      artifacts: [],
    },
  };

  const artifactText: Record<string, string> = {
    "charter/overview.md": "# Charter\n",
    "reports/coverage/summary.md": "Coverage: 84%\n",
    "reports/coverage/open-questions.md": "- Clarify owners\n",
    ...(state.artifactText ?? {}),
  };

  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === "string" ? input : input.toString();
    const method = (init?.method ?? "GET").toUpperCase();

    if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
      if (state.runStarted) {
        return jsonResponse({
          items: [
            {
              run_id: runID,
              pipeline: "init",
              status: "succeeded",
              started_at: "2026-04-03T12:00:00Z",
              finished_at: "2026-04-03T12:00:02Z",
              warnings: [],
              error_code: null,
              error: null,
            },
          ],
        });
      }
      return jsonResponse({ items: [] });
    }

    if (method === "GET" && url === "/api/workspace/manifest") {
      return jsonResponse({
        content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n",
      });
    }

    if (method === "GET" && url === "/api/runtime/timeouts") {
      return jsonResponse({
        ok: true,
        persisted: { step_timeout_sec: 1800 },
        effective: {
          step_timeout_sec: 1800,
          heartbeat_sec: 30,
          pipeline_timeout_sec: 2400,
          pipeline_kill_grace_sec: 30,
          api_ready_timeout_sec: 60,
          api_init_timeout_sec: 120,
          ui_init_poll_timeout_sec: 900,
          ui_cancel_poll_timeout_sec: 420,
        },
        source: { step_timeout_sec: "workspace" },
      });
    }

    if (method === "GET" && url === "/api/runtime/execution") {
      return jsonResponse({
        ok: true,
        persisted: { strategy: "sequential", max_parallel_tasks: 1 },
        effective: {
          strategy: "sequential",
          max_parallel_tasks: 1,
          failure_policy: "best_effort",
          shard_discovery_mode: "heuristics",
        },
        source: { strategy: "workspace" },
      });
    }

    if (method === "GET" && url.startsWith("/api/artifacts?path=")) {
      const encodedPath = url.slice("/api/artifacts?path=".length);
      const decodedPath = decodeURIComponent(encodedPath);
      if (decodedPath === "charter/wizard/step0-contract.json") {
        return textResponse("not found", 404);
      }
      if (artifactText[decodedPath] !== undefined) {
        return textResponse(artifactText[decodedPath]);
      }
      return textResponse("", 404);
    }

    if (method === "POST" && url === "/api/workspace/validate") {
      return jsonResponse({
        ok: true,
        workspace: "/tmp/workspace",
        warnings: [],
        errors: [],
      });
    }

    if (method === "POST" && url === "/api/pipeline/init") {
      state.runStarted = true;
      return jsonResponse({ run_id: runID, status: "queued" });
    }

    if (method === "GET" && url === `/api/pipeline/runs/${runID}`) {
      return jsonResponse((runStatus[runID] ?? {}) as MockJSON);
    }

    if (method === "GET" && url === `/api/pipeline/runs/${runID}/artifacts`) {
      return jsonResponse((runArtifacts[runID] ?? { run_id: runID, artifacts: [] }) as MockJSON);
    }

    if (method === "GET" && url.startsWith(`/api/pipeline/runs/${runID}/logs?`)) {
      return jsonResponse((runLogs[runID] ?? { run_id: runID, items: [], next_cursor: 0, eof: true }) as MockJSON);
    }

    if (method === "POST" && url === "/api/artifacts/write") {
      return jsonResponse({ ok: true });
    }

    return jsonResponse(
      {
        error: {
          code: "not_found",
          message: `unhandled request: ${method} ${url}`,
        },
      },
      404,
    );
  });

  return fetchMock;
}

vi.mock("mermaid", () => {
  return {
    default: {
      initialize: vi.fn(),
      render: vi.fn(async (_id: string, graph: string) => ({
        svg: `<svg data-graph="${graph.replace(/"/g, "&quot;")}"></svg>`,
      })),
    },
  };
});

describe("App", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("supports top tab navigation and settings relocation", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    render(<App />);

    expect(screen.getByTestId("workspace-panel")).toBeInTheDocument();
    expect(screen.queryByTestId("runtime-timeouts-panel")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tab-settings"));

    expect(await screen.findByTestId("runtime-timeouts-panel")).toBeInTheDocument();
    expect(screen.getByTestId("runtime-execution-panel")).toBeInTheDocument();
  });

  it("runs init from Runs tab and supports event/raw log modes", async () => {
    const runID = "run-logs";
    const fetchMock = createFetchMock({
      runID,
      runLogs: {
        [runID]: {
          run_id: runID,
          items: [
            {
              cursor: 0,
              timestamp: "2026-04-03T12:00:00Z",
              level: "info",
              kind: "event",
              step_id: "init.step1.collect",
              message: "runtime task started",
            },
            {
              cursor: 1,
              timestamp: "2026-04-03T12:00:01Z",
              level: "info",
              kind: "runtime_output",
              stream: "stdout",
              step_id: "init.step1.collect",
              message: "agent stdout line",
            },
            {
              cursor: 2,
              timestamp: "2026-04-03T12:00:01Z",
              level: "info",
              kind: "runtime_output",
              stream: "stderr",
              step_id: "init.step1.collect",
              message: "agent stderr line",
            },
          ],
          next_cursor: 3,
          eof: true,
        },
      },
      runArtifacts: {
        [runID]: {
          run_id: runID,
          artifacts: [
            { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
          ],
        },
      },
      artifactText: {
        "reports/as-is/overview.md": "# As-is\n",
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    fireEvent.click(screen.getByTestId("tab-runs"));
    fireEvent.click(screen.getByTestId("run-init-btn"));

    await screen.findByTestId("run-status-panel");
    const logs = await screen.findByTestId("run-logs-content");
    expect(logs.textContent ?? "").toContain("[EVENT]");
    expect(logs.textContent ?? "").toContain("[RAW]");

    fireEvent.change(screen.getByTestId("run-logs-mode-select"), { target: { value: "raw" } });

    await waitFor(() => {
      const rawOnly = screen.getByTestId("run-logs-content").textContent ?? "";
      expect(rawOnly).toContain("[RAW]");
      expect(rawOnly).not.toContain("[EVENT]");
    });
  });

  it("shows diagrams surface in Results tab and renders Mermaid preview", async () => {
    const runID = "run-diagrams";
    const fetchMock = createFetchMock({
      runID,
      runStarted: true,
      runArtifacts: {
        [runID]: {
          run_id: runID,
          artifacts: [
            { path: "reports/diagrams/c4-context.mmd", kind: "diagram", label: "C4 Context" },
            { path: "reports/diagrams/index.md", kind: "diagram-index", label: "Diagrams Index" },
            { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
          ],
        },
      },
      artifactText: {
        "reports/diagrams/c4-context.mmd": "flowchart LR\nA-->B\n",
        "reports/diagrams/index.md": "# C4 Diagrams Index\n",
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    fireEvent.click(screen.getByTestId("tab-results"));
    fireEvent.click(screen.getByTestId("results-tab-diagrams"));

    const diagramButton = await screen.findByRole("button", { name: /reports\/diagrams\/c4-context\.mmd/i });
    fireEvent.click(diagramButton);

    await waitFor(() => {
      const preview = screen.getByTestId("run-diagram-content-panel").innerHTML;
      expect(preview).toContain("<svg");
    });
  });

  it("edits baseline prompt file in Baseline tab and saves it", async () => {
    const fetchMock = createFetchMock({
      artifactText: {
        "skills/prompt-packs/qa.md": "qa prompt baseline\n",
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    fireEvent.click(screen.getByTestId("tab-baseline"));

    const select = await screen.findByLabelText(/select artifact/i);
    fireEvent.change(select, { target: { value: "skills/prompt-packs/qa.md" } });

    const editor = await screen.findByLabelText("skills/prompt-packs/qa.md");
    fireEvent.change(editor, { target: { value: "qa prompt baseline\nupdated line\n" } });
    fireEvent.click(screen.getByRole("button", { name: /save selected baseline artifact/i }));

    await screen.findByText("Saved skills/prompt-packs/qa.md");

    const saveCalls = fetchMock.mock.calls.filter((call) => call[0] === "/api/artifacts/write");
    expect(saveCalls.length).toBe(1);
  });
});
