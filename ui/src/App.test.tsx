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

    if (method === "GET" && url === "/api/runtime/profile") {
      return jsonResponse({
        ok: true,
        step_providers: {
          persisted: { step2_as_is: "qwen-code" },
          effective: {
            step0_constitution: "claude-code",
            step1_collect: "claude-code",
            step2_as_is: "qwen-code",
            step3_findings: "claude-code",
            step4_proposals: "claude-code",
          },
          source: {
            step0_constitution: "default",
            step1_collect: "default",
            step2_as_is: "workspace",
            step3_findings: "default",
            step4_proposals: "default",
          },
        },
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

    if (method === "POST" && url === "/api/git/commit") {
      const payload = JSON.parse(String(init?.body ?? "{}")) as { message?: string };
      return jsonResponse({
        status: "ok",
        output: `committed: ${payload.message ?? ""}`.trim(),
      });
    }

    if (method === "POST" && url === "/api/git/proposal-branch") {
      const payload = JSON.parse(String(init?.body ?? "{}")) as { name?: string };
      return jsonResponse({
        branch: payload.name ?? "proposal/beta-refresh",
      });
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
    vi.useRealTimers();
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
    expect(screen.getByTestId("runtime-step-providers-panel")).toBeInTheDocument();
    expect(screen.getByText("runtime.profile.steps.step2_as_is.provider")).toBeInTheDocument();
    expect(screen.getAllByText("qwen-code").length).toBeGreaterThanOrEqual(1);
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

  it("renders failed run status with warnings and error details for partial live state", async () => {
    const runID = "run-partial-failed";
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID,
        runStarted: true,
        runStatus: {
          [runID]: {
            run_id: runID,
            pipeline: "refresh",
            status: "failed",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: "2026-04-03T12:05:00Z",
            current_step: "refresh.step3.findings",
            warnings: ["collect coverage incomplete", "draft promotion skipped"],
            error_code: "run_partial_failed",
            error: "runtime draft manifest invalid",
          },
        },
      }),
    );

    render(<App />);

    fireEvent.click(screen.getByTestId("tab-runs"));

    await screen.findByTestId("run-status-panel");
    expect(screen.getByTestId("run-status-value").textContent).toBe("failed");
    expect(screen.getByTestId("run-lifecycle-value").textContent).toBe("terminal");
    expect(screen.getByText("Current step: refresh.step3.findings")).toBeInTheDocument();
    expect(screen.getByText("Error code: run_partial_failed")).toBeInTheDocument();
    expect(screen.getByText("Error: runtime draft manifest invalid")).toBeInTheDocument();
    expect(screen.getByTestId("run-status-warnings").textContent ?? "").toContain("collect coverage incomplete");
    expect(screen.getByTestId("run-status-warnings").textContent ?? "").toContain("draft promotion skipped");
  });

  it("renders running run status for active live progress state", async () => {
    const runID = "run-live-running";
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID,
        runStarted: true,
        runStatus: {
          [runID]: {
            run_id: runID,
            pipeline: "refresh",
            status: "running",
            started_at: "2026-04-03T12:00:00Z",
            current_step: "refresh.step2.asis_docs",
            warnings: [],
            error_code: null,
            error: null,
          },
        },
      }),
    );

    render(<App />);

    fireEvent.click(screen.getByTestId("tab-runs"));

    await screen.findByTestId("run-status-panel");
    expect(screen.getByTestId("run-status-value").textContent).toBe("running");
    expect(screen.getByTestId("run-lifecycle-value").textContent).toBe("active");
    expect(screen.getByText("Current step: refresh.step2.asis_docs")).toBeInTheDocument();
    expect(screen.getByTestId("run-status-warnings-empty").textContent).toContain("Warnings: none");
  });

  it("renders incomplete lifecycle status when run error code marks incomplete cycle", async () => {
    const runID = "run-incomplete";
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID,
        runStarted: true,
        runStatus: {
          [runID]: {
            run_id: runID,
            pipeline: "refresh",
            status: "failed",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: "2026-04-03T12:05:00Z",
            current_step: "refresh.step4.proposals",
            warnings: ["infra_incomplete_cycle breadcrumb"],
            error_code: "infra_incomplete_cycle",
            error: "profile exited before completion",
          },
        },
      }),
    );

    render(<App />);

    fireEvent.click(screen.getByTestId("tab-runs"));

    await screen.findByTestId("run-status-panel");
    expect(screen.getByTestId("run-lifecycle-value").textContent).toBe("incomplete");
  });

  it("renders recovered lifecycle status for reconciled runs", async () => {
    const runID = "run-recovered";
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID,
        runStarted: true,
        runStatus: {
          [runID]: {
            run_id: runID,
            pipeline: "refresh",
            status: "failed",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: "2026-04-03T12:05:00Z",
            current_step: "refresh.step1.collect",
            warnings: [],
            error_code: "run_reconciled_after_restart",
            error: "orphaned run reconciled after restart",
          },
        },
      }),
    );

    render(<App />);

    fireEvent.click(screen.getByTestId("tab-runs"));

    await screen.findByTestId("run-status-panel");
    expect(screen.getByTestId("run-lifecycle-value").textContent).toBe("recovered");
  });

  it("switches to the next available run when the selected run disappears during refresh", async () => {
    const baseFetch = createFetchMock();
    let runListCalls = 0;
    let staleRunStatusCalls = 0;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        runListCalls += 1;
        if (runListCalls === 1) {
          return jsonResponse({
            items: [
              {
                run_id: "run-stale",
                pipeline: "refresh",
                status: "running",
                started_at: "2026-04-03T12:00:00Z",
                finished_at: null,
                warnings: [],
                error_code: null,
                error: null,
              },
            ],
          });
        }
        return jsonResponse({
          items: [
            {
              run_id: "run-fresh",
              pipeline: "refresh",
              status: "succeeded",
              started_at: "2026-04-03T12:01:00Z",
              finished_at: "2026-04-03T12:01:15Z",
              warnings: [],
              error_code: null,
              error: null,
            },
          ],
        });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-stale") {
        staleRunStatusCalls += 1;
        if (staleRunStatusCalls >= 2) {
          return jsonResponse({
            error: {
              code: "not_found",
              message: "run-stale no longer exists",
            },
          }, 404);
        }
        return jsonResponse({
          run_id: "run-stale",
          pipeline: "refresh",
          status: "running",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: null,
          current_step: "refresh.step2.asis_docs",
          warnings: [],
          error_code: null,
          error: null,
        });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-fresh") {
        return jsonResponse({
          run_id: "run-fresh",
          pipeline: "refresh",
          status: "succeeded",
          started_at: "2026-04-03T12:01:00Z",
          finished_at: "2026-04-03T12:01:15Z",
          warnings: [],
          error_code: null,
          error: null,
        });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-stale/artifacts") {
        return jsonResponse({ run_id: "run-stale", artifacts: [] });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-fresh/artifacts") {
        return jsonResponse({ run_id: "run-fresh", artifacts: [] });
      }

      if (method === "GET" && url.startsWith("/api/pipeline/runs/run-stale/logs?")) {
        return jsonResponse({ run_id: "run-stale", items: [], next_cursor: 0, eof: true });
      }

      if (method === "GET" && url.startsWith("/api/pipeline/runs/run-fresh/logs?")) {
        return jsonResponse({ run_id: "run-fresh", items: [], next_cursor: 0, eof: true });
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);
    fireEvent.click(screen.getByTestId("tab-runs"));

    await waitFor(() => {
      expect(screen.getByTestId("run-status-run-id").textContent).toBe("run-stale");
    });

    await waitFor(() => {
      expect(screen.getByTestId("run-status-run-id").textContent).toBe("run-fresh");
    }, { timeout: 4000 });
    expect(screen.getByText("Selected run no longer exists; switched to run-fresh.")).toBeInTheDocument();
  }, 10000);

  it("keeps the selected run when history refresh is temporarily empty but run status still exists", async () => {
    const baseFetch = createFetchMock();
    let runListCalls = 0;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        runListCalls += 1;
        if (runListCalls === 1) {
          return jsonResponse({
            items: [
              {
                run_id: "run-stale",
                pipeline: "refresh",
                status: "running",
                started_at: "2026-04-03T12:00:00Z",
                finished_at: null,
                warnings: [],
                error_code: null,
                error: null,
              },
            ],
          });
        }
        return jsonResponse({ items: [] });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-stale") {
        return jsonResponse({
          run_id: "run-stale",
          pipeline: "refresh",
          status: "running",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: null,
          current_step: "refresh.step2.asis_docs",
          warnings: [],
          error_code: null,
          error: null,
        });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-stale/artifacts") {
        return jsonResponse({ run_id: "run-stale", artifacts: [] });
      }

      if (method === "GET" && url.startsWith("/api/pipeline/runs/run-stale/logs?")) {
        return jsonResponse({ run_id: "run-stale", items: [], next_cursor: 0, eof: true });
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);
    fireEvent.click(screen.getByTestId("tab-runs"));

    await waitFor(() => {
      expect(screen.getByTestId("run-status-run-id").textContent).toBe("run-stale");
    });

    await waitFor(() => {
      expect(runListCalls).toBeGreaterThanOrEqual(2);
    }, { timeout: 4000 });

    expect(screen.getByTestId("run-status-run-id").textContent).toBe("run-stale");
    expect(screen.queryByText(/Selected run no longer exists;/)).not.toBeInTheDocument();
  }, 10000);

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

  it("executes git-helper commit and proposal-branch actions from Baseline tab", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    fireEvent.click(screen.getByTestId("tab-baseline"));

    const commitInput = await screen.findByLabelText("Commit message");
    fireEvent.change(commitInput, { target: { value: "feat: tighten prompt policy" } });
    fireEvent.click(screen.getByTestId("git-commit-btn"));

    await screen.findByText("committed: feat: tighten prompt policy");

    const branchInput = screen.getByLabelText("Proposal branch");
    fireEvent.change(branchInput, { target: { value: "proposal/prompt-policy" } });
    fireEvent.click(screen.getByTestId("git-proposal-branch-btn"));

    await screen.findByText("checked out proposal/prompt-policy");

    const commitCalls = fetchMock.mock.calls.filter((call) => call[0] === "/api/git/commit");
    expect(commitCalls).toHaveLength(1);
    const branchCalls = fetchMock.mock.calls.filter((call) => call[0] === "/api/git/proposal-branch");
    expect(branchCalls).toHaveLength(1);
  });
});
