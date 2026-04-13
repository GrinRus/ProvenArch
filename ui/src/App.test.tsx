import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import App from "./App";

type MockJSON = Record<string, unknown>;

function jsonResponse(body: MockJSON, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" }
  });
}

function textResponse(body: string, status = 200): Response {
  return new Response(body, {
    status,
    headers: { "Content-Type": "text/plain; charset=utf-8" }
  });
}

function maybeRunLogsResponse(method: string, url: string): Response | null {
  if (method !== "GET") {
    return null;
  }
  const match = url.match(/^\/api\/pipeline\/runs\/([^/]+)\/logs\?.*$/);
  if (!match) {
    return null;
  }
  return jsonResponse({
    run_id: match[1],
    items: [],
    next_cursor: 0,
    eof: true
  });
}

describe("App", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("runs mocked flow open -> validate -> run -> inspect", async () => {
    let runStarted = false;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        if (!runStarted) {
          return jsonResponse({ items: [] });
        }
        return jsonResponse({
          items: [
            {
              run_id: "run-1",
              pipeline: "init",
              status: "succeeded",
              started_at: "2026-04-03T12:00:00Z",
              finished_at: "2026-04-03T12:00:02Z",
              warnings: [],
              error_code: null,
              error: null
            }
          ]
        });
      }

      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }

      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }

      if (method === "GET" && url === "/api/artifacts?path=skills%2Fprompt-packs%2Fcollect-context.md") {
        return textResponse("collect context prompt\n");
      }

      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse(
          JSON.stringify(
            {
              version: 1,
              project_name: "ProvenArch MVP",
              scope: "payments, users",
              nfr_priorities: ["availability"],
              rules: ["evidence-first"]
            },
            null,
            2
          )
        );
      }

      if (method === "POST" && url === "/api/workspace/validate") {
        return jsonResponse({
          ok: true,
          workspace: "/tmp/workspace",
          warnings: [],
          errors: [],
          resolved_repos: [
            {
              name: "payments-service",
              source: "path",
              path: "/tmp/payments-service"
            }
          ]
        });
      }

      if (method === "POST" && url === "/api/pipeline/init") {
        runStarted = true;
        return jsonResponse({ run_id: "run-1", status: "queued" });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-1") {
        return jsonResponse({
          run_id: "run-1",
          pipeline: "init",
          status: "succeeded",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: "2026-04-03T12:00:02Z",
          error_code: null,
          warnings: []
        });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-1/artifacts") {
        return jsonResponse({
          run_id: "run-1",
          artifacts: [
            {
              path: "reports/as-is/overview.md",
              kind: "report",
              label: "As-is overview"
            }
          ]
        });
      }

      if (method === "GET" && url === "/api/artifacts?path=reports%2Fcoverage%2Fsummary.md") {
        return textResponse("Coverage: 84%\n");
      }

      if (method === "GET" && url === "/api/artifacts?path=reports%2Fcoverage%2Fopen-questions.md") {
        return textResponse("- Clarify owners for notifications-service\n");
      }

      if (method === "GET" && url === "/api/artifacts?path=reports%2Fas-is%2Foverview.md") {
        return textResponse("# As-is\nDeterministic snapshot\n");
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    expect(screen.getByText("ACP Beta Surface")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /validate workspace/i }));

    await screen.findByText("Status: valid");
    expect(screen.getByText(/Workspace:/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /^run init$/i }));

    await screen.findByText(/Pipeline:\s*init/i);
    await screen.findByRole("button", { name: /run-1/i });
    expect(screen.getByText(/Running:\s*0\s*\|\s*Succeeded:\s*1\s*\|\s*Failed:\s*0/i)).toBeInTheDocument();
    await screen.findByRole("button", { name: /reports\/as-is\/overview.md/i });

    fireEvent.click(screen.getByRole("button", { name: /reports\/as-is\/overview.md/i }));

    await screen.findByText(/Deterministic snapshot/i);
    expect(screen.getByText(/Coverage: 84%/i)).toBeInTheDocument();
    expect(screen.getByText(/Clarify owners for notifications-service/i)).toBeInTheDocument();

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith("/api/pipeline/init", expect.any(Object));
    });
  });

  it("edits selected baseline artifact and saves through write endpoint", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({ items: [] });
      }

      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }

      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }

      if (method === "GET" && url === "/api/artifacts?path=skills%2Fprompt-packs%2Fqa.md") {
        return textResponse("qa prompt baseline\n");
      }

      if (method === "POST" && url === "/api/artifacts/write") {
        return jsonResponse({ ok: true });
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    const select = await screen.findByLabelText(/select artifact/i);
    fireEvent.change(select, { target: { value: "skills/prompt-packs/qa.md" } });

    const editor = await screen.findByLabelText("skills/prompt-packs/qa.md");
    fireEvent.change(editor, { target: { value: "qa prompt baseline\nupdated line\n" } });
    fireEvent.click(screen.getByRole("button", { name: /save selected baseline artifact/i }));

    await screen.findByText("Saved skills/prompt-packs/qa.md");

    const saveCalls = fetchMock.mock.calls.filter((call) => call[0] === "/api/artifacts/write");
    expect(saveCalls.length).toBe(1);

    const [, saveInit] = saveCalls[0] as [RequestInfo | URL, RequestInit];
    expect(saveInit?.method).toBe("POST");
    expect(saveInit?.body).toBe(
      JSON.stringify({
        path: "skills/prompt-packs/qa.md",
        content: "qa prompt baseline\nupdated line\n"
      })
    );
  });

  it("loads and saves runtime timeouts via API", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({ items: [] });
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
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
            ui_cancel_poll_timeout_sec: 420
          },
          source: {
            step_timeout_sec: "workspace"
          }
        });
      }
      if (method === "PUT" && url === "/api/runtime/timeouts") {
        return jsonResponse({ ok: true });
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    const stepInput = await screen.findByTestId("runtime-timeout-input-step_timeout_sec");
    fireEvent.change(stepInput, { target: { value: "1500" } });
    fireEvent.click(screen.getByTestId("runtime-timeouts-save-btn"));
    await screen.findByText("Runtime timeouts saved");

    const saveCalls = fetchMock.mock.calls.filter(
      (call) => call[0] === "/api/runtime/timeouts" && ((call[1] as RequestInit | undefined)?.method ?? "").toUpperCase() === "PUT"
    );
    expect(saveCalls.length).toBe(1);
    const [, saveInit] = saveCalls[0] as [RequestInfo | URL, RequestInit];
    expect(saveInit.body).toContain('"step_timeout_sec":1500');
  });

  it("loads and saves runtime execution profile via API", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({ items: [] });
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }
      if (method === "GET" && url === "/api/runtime/execution") {
        return jsonResponse({
          ok: true,
          persisted: {
            strategy: "sequential",
            max_parallel_tasks: 1,
            failure_policy: "best_effort",
            shard_discovery_mode: "heuristics",
            repo_selection: "all"
          },
          effective: {
            strategy: "sequential",
            max_parallel_tasks: 1,
            failure_policy: "best_effort",
            shard_discovery_mode: "heuristics",
            repo_selection: "all"
          },
          source: {
            strategy: "workspace",
            max_parallel_tasks: "workspace",
            failure_policy: "workspace",
            shard_discovery_mode: "workspace",
            repo_selection: "workspace"
          }
        });
      }
      if (method === "PUT" && url === "/api/runtime/execution") {
        return jsonResponse({ ok: true });
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    const strategySelect = await screen.findByTestId("runtime-execution-strategy-select");
    fireEvent.change(strategySelect, { target: { value: "parallel" } });
    const maxParallelInput = screen.getByTestId("runtime-execution-max-parallel-input");
    fireEvent.change(maxParallelInput, { target: { value: "3" } });
    const failurePolicySelect = screen.getByTestId("runtime-execution-failure-policy-select");
    fireEvent.change(failurePolicySelect, { target: { value: "fail_fast" } });
    const shardModeSelect = screen.getByTestId("runtime-execution-shard-mode-select");
    fireEvent.change(shardModeSelect, { target: { value: "semantic" } });
    const repoSelectionSelect = screen.getByTestId("runtime-execution-repo-selection-select");
    fireEvent.change(repoSelectionSelect, { target: { value: "backend_only" } });

    fireEvent.click(screen.getByTestId("runtime-execution-save-btn"));
    await screen.findByText("Runtime execution profile saved");

    const saveCalls = fetchMock.mock.calls.filter(
      (call) => call[0] === "/api/runtime/execution" && ((call[1] as RequestInit | undefined)?.method ?? "").toUpperCase() === "PUT"
    );
    expect(saveCalls.length).toBe(1);
    const [, saveInit] = saveCalls[0] as [RequestInfo | URL, RequestInit];
    expect(saveInit.body).toContain('"strategy":"parallel"');
    expect(saveInit.body).toContain('"max_parallel_tasks":3');
    expect(saveInit.body).toContain('"failure_policy":"fail_fast"');
    expect(saveInit.body).toContain('"shard_discovery_mode":"semantic"');
    expect(saveInit.body).toContain('"repo_selection":"backend_only"');
  });

  it("does not start polling loop when there are no active runs", async () => {
    vi.useFakeTimers();

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({ items: [] });
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    expect(screen.getByText(/Runs:\s*History/i)).toBeInTheDocument();

    const listCallsBefore = fetchMock.mock.calls.filter((call) => call[0] === "/api/pipeline/runs?limit=100").length;
    expect(listCallsBefore).toBe(1);

    await act(async () => {
      vi.advanceTimersByTime(3000);
    });

    const listCallsAfter = fetchMock.mock.calls.filter((call) => call[0] === "/api/pipeline/runs?limit=100").length;
    expect(listCallsAfter).toBe(1);
  });

  it("renders queued/succeeded/failed runs and opens selected run details", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({
          items: [
            {
              run_id: "run-queued",
              pipeline: "refresh",
              status: "queued",
              started_at: "2026-04-03T12:01:00Z",
              finished_at: null,
              warnings: [],
              error_code: null,
              error: null
            },
            {
              run_id: "run-ok",
              pipeline: "init",
              status: "succeeded",
              started_at: "2026-04-03T12:00:00Z",
              finished_at: "2026-04-03T12:00:02Z",
              warnings: ["step0_wizard_contract_missing: fallback baseline used"],
              error_code: null,
              error: null
            },
            {
              run_id: "run-failed",
              pipeline: "refresh",
              status: "failed",
              started_at: "2026-04-03T11:59:00Z",
              finished_at: "2026-04-03T11:59:04Z",
              warnings: ["repo source warning"],
              error_code: "runner_parse_failed",
              error: "synthetic parse failure"
            }
          ]
        });
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-failed") {
        return jsonResponse({
          run_id: "run-failed",
          pipeline: "refresh",
          status: "failed",
          started_at: "2026-04-03T11:59:00Z",
          finished_at: "2026-04-03T11:59:04Z",
          current_step: "refresh.step3.findings",
          warnings: ["repo source warning"],
          error_code: "runner_parse_failed",
          error: "synthetic parse failure"
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-failed/artifacts") {
        return jsonResponse({
          run_id: "run-failed",
          artifacts: [
            {
              path: "reports/taskruns/run-failed.json",
              kind: "taskrun",
              label: "run-failed taskrun"
            }
          ]
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=reports%2Ftaskruns%2Frun-failed.json") {
        return textResponse("{\"status\":\"failed\"}\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=reports%2Fcoverage%2Fsummary.md") {
        return textResponse("Coverage: 61%\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=reports%2Fcoverage%2Fopen-questions.md") {
        return textResponse("- clarify ownership\n");
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    await screen.findByText(/Runs:\s*History/i);
    await screen.findByRole("button", { name: /run-queued/i });
    await screen.findByRole("button", { name: /run-ok/i });
    await screen.findByRole("button", { name: /run-failed/i });
    expect(screen.getByText(/Running:\s*1\s*\|\s*Succeeded:\s*1\s*\|\s*Failed:\s*1/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /run-failed/i }));
    await screen.findByText(/Error code:\s*runner_parse_failed/i);
    await screen.findByRole("button", { name: /reports\/taskruns\/run-failed.json/i });
  });

  it("builds multi-repo manifest in guided setup with empty refs by default", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({ items: [] });
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }
      if (method === "PUT" && url === "/api/workspace/manifest") {
        return jsonResponse({ ok: true });
      }
      if (method === "POST" && url === "/api/workspace/validate") {
        return jsonResponse({
          ok: true,
          workspace: "/tmp/workspace",
          warnings: [],
          errors: [],
          resolved_repos: []
        });
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: /add repo/i }));

    const repoNameInputs = screen.getAllByLabelText(/Repo name/i);
    fireEvent.change(repoNameInputs[1], { target: { value: "users-service" } });

    const repoModeSelects = screen.getAllByLabelText(/Repo source type/i);
    fireEvent.change(repoModeSelects[1], { target: { value: "git_url" } });

    const gitURLInput = await screen.findByLabelText(/^git_url$/i);
    fireEvent.change(gitURLInput, {
      target: { value: "https://gitlab.example.com/platform/users-service.git" }
    });

    fireEvent.click(screen.getByRole("button", { name: /apply guided workspace form/i }));
    fireEvent.click(screen.getByRole("button", { name: /save workspace\.yaml/i }));

    await screen.findByText("Status: valid");

    const saveCalls = fetchMock.mock.calls.filter(
      (call) => call[0] === "/api/workspace/manifest" && ((call[1] as RequestInit | undefined)?.method ?? "").toUpperCase() === "PUT"
    );
    expect(saveCalls.length).toBe(1);

    const [, saveInit] = saveCalls[0] as [RequestInfo | URL, RequestInit];
    const parsed = JSON.parse(String(saveInit?.body)) as { content: string };
    expect(parsed.content).toContain("name: payments-service");
    expect(parsed.content).toContain("name: users-service");
    expect(parsed.content).toContain("git_url: https://gitlab.example.com/platform/users-service.git");
    expect(parsed.content).not.toContain("\n    ref:");
  });

  it("shows resolved repos and diagnostics grouped by repo", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({ items: [] });
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }
      if (method === "POST" && url === "/api/workspace/validate") {
        return jsonResponse({
          ok: false,
          workspace: "/tmp/workspace",
          repo_selection_mode: "backend_only",
          selected_repo_scopes: ["payments-service"],
          repo_selection: [
            {
              name: "payments-service",
              declared_role: "backend",
              effective_role: "backend",
              included: true,
              reason: "included by repo_selection=backend_only (effective_role=backend)"
            },
            {
              name: "web-frontend",
              declared_role: "frontend",
              effective_role: "frontend",
              included: false,
              reason: "excluded by repo_selection=backend_only (effective_role=frontend)"
            }
          ],
          resolved_repos: [
            {
              name: "payments-service",
              source: "path",
              path: "/tmp/payments-service",
              ref: "main",
              effective_role: "backend",
              included: true,
              selection_reason: "included by repo_selection=backend_only (effective_role=backend)"
            }
          ],
          warnings: [
            {
              level: "warning",
              code: "workspace.repo.ref.resolved_via_remote",
              message: "ref \"main\" was resolved via \"origin/main\"",
              repo: "payments-service"
            }
          ],
          errors: [
            {
              level: "error",
              code: "workspace.layout.dir.unreadable",
              message: "workspace directory unreadable"
            }
          ]
        });
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: /validate workspace/i }));

    await screen.findByText("Status: invalid");
    expect(screen.getByText(/Resolved repos/i)).toBeInTheDocument();
    expect(screen.getByText(/Repo selection mode:/i)).toBeInTheDocument();
    expect(screen.getByText(/Repo selection decisions/i)).toBeInTheDocument();
    expect(screen.getByText(/Diagnostics for payments-service/i)).toBeInTheDocument();
    expect(screen.getByText(/Workspace diagnostics/i)).toBeInTheDocument();
    expect(screen.getByText(/workspace\.repo\.ref\.resolved_via_remote/i)).toBeInTheDocument();
    expect(screen.getByText(/workspace\.layout\.dir\.unreadable/i)).toBeInTheDocument();
  });

  it("renders run logs panel and opens taskrun artifact from log quick action", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({
          items: [
            {
              run_id: "run-log",
              pipeline: "init",
              status: "succeeded",
              started_at: "2026-04-03T12:00:00Z",
              finished_at: "2026-04-03T12:00:02Z",
              warnings: ["init.step1.collect: synthetic runtime warning"],
              error_code: null,
              error: null
            }
          ]
        });
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-log") {
        return jsonResponse({
          run_id: "run-log",
          pipeline: "init",
          status: "succeeded",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: "2026-04-03T12:00:02Z",
          current_step: null,
          warnings: ["init.step1.collect: synthetic runtime warning"],
          error_code: null,
          error: null
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-log/artifacts") {
        return jsonResponse({
          run_id: "run-log",
          artifacts: [
            {
              path: "reports/taskruns/run-log-step1.json",
              kind: "taskrun",
              label: "run-log step1 taskrun"
            }
          ]
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-log/logs?cursor=0&limit=200") {
        return jsonResponse({
          run_id: "run-log",
          items: [
            {
              cursor: 0,
              timestamp: "2026-04-03T12:00:00Z",
              level: "info",
              step_id: "init.step1.collect",
              domain_id: "payments-service",
              message: "runtime task started"
            },
            {
              cursor: 1,
              timestamp: "2026-04-03T12:00:01Z",
              level: "warning",
              step_id: "init.step1.collect",
              domain_id: "payments-service",
              message: "runtime warning",
              taskrun_path: "reports/taskruns/run-log-step1.json",
              fields: {
                task_id: "task-run-log-init-step1",
                stderr_snippet: "line-1\\nline-2"
              }
            }
          ],
          next_cursor: 2,
          eof: true
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=reports%2Ftaskruns%2Frun-log-step1.json") {
        return textResponse("{\"status\":\"ok\"}\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=reports%2Fcoverage%2Fsummary.md") {
        return textResponse("Coverage: 70%\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=reports%2Fcoverage%2Fopen-questions.md") {
        return textResponse("- verify ownership\n");
      }

      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    await screen.findByRole("button", { name: /^run-log$/i });
    fireEvent.click(screen.getByRole("button", { name: /^run-log$/i }));

    await screen.findByText(/runtime task started/i);
    await screen.findByRole("button", { name: /Open taskrun artifact: reports\/taskruns\/run-log-step1\.json/i });
    expect(screen.queryByText(/task-run-log-init-step1/i)).not.toBeInTheDocument();

    fireEvent.change(screen.getByTestId("run-logs-view-select"), { target: { value: "line+fields" } });
    await screen.findByText(/task-run-log-init-step1/i);

    fireEvent.click(screen.getByRole("button", { name: /Open taskrun artifact: reports\/taskruns\/run-log-step1\.json/i }));
    await screen.findByText(/\{\"status\":\"ok\"\}/i);
  });

  it("auto-selects newest active run on bootstrap even if list order differs", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({
          items: [
            {
              run_id: "run-active-old",
              pipeline: "refresh",
              status: "running",
              started_at: "2026-04-03T12:01:00Z",
              finished_at: null,
              warnings: ["older active run"],
              error_code: null,
              error: null
            },
            {
              run_id: "run-active-new",
              pipeline: "refresh",
              status: "queued",
              started_at: "2026-04-03T12:03:00Z",
              finished_at: null,
              warnings: ["newest active run"],
              error_code: null,
              error: null
            },
            {
              run_id: "run-old",
              pipeline: "init",
              status: "succeeded",
              started_at: "2026-04-03T11:58:00Z",
              finished_at: "2026-04-03T11:58:02Z",
              warnings: [],
              error_code: null,
              error: null
            }
          ]
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active-new") {
        return jsonResponse({
          run_id: "run-active-new",
          pipeline: "refresh",
          status: "queued",
          started_at: "2026-04-03T12:03:00Z",
          finished_at: null,
          current_step: "",
          warnings: ["newest active run"],
          error_code: null,
          error: null
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active-new/artifacts") {
        return jsonResponse({
          run_id: "run-active-new",
          artifacts: []
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active-new/logs?cursor=0&limit=200") {
        return jsonResponse({
          run_id: "run-active-new",
          items: [],
          next_cursor: 0,
          eof: true
        });
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }

      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    await screen.findByTestId("run-status-run-id");
    expect(screen.getByTestId("run-status-run-id")).toHaveTextContent("run-active-new");
    expect(screen.getByText(/Warnings \(1\)/i)).toBeInTheDocument();
    expect(screen.getByText(/newest active run/i)).toBeInTheDocument();
  });

  it("keeps manual run selection and does not auto-switch on polling updates", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({
          items: [
            {
              run_id: "run-active-new",
              pipeline: "refresh",
              status: "running",
              started_at: "2026-04-03T12:03:00Z",
              finished_at: null,
              warnings: ["newest active run"],
              error_code: null,
              error: null
            },
            {
              run_id: "run-active-old",
              pipeline: "refresh",
              status: "running",
              started_at: "2026-04-03T12:01:00Z",
              finished_at: null,
              warnings: ["older active run"],
              error_code: null,
              error: null
            }
          ]
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active-new") {
        return jsonResponse({
          run_id: "run-active-new",
          pipeline: "refresh",
          status: "running",
          started_at: "2026-04-03T12:03:00Z",
          finished_at: null,
          current_step: "refresh.step1.collect",
          warnings: ["newest active run"],
          error_code: null,
          error: null
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active-old") {
        return jsonResponse({
          run_id: "run-active-old",
          pipeline: "refresh",
          status: "running",
          started_at: "2026-04-03T12:01:00Z",
          finished_at: null,
          current_step: "refresh.step0.prepare",
          warnings: ["older active run"],
          error_code: null,
          error: null
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active-new/artifacts") {
        return jsonResponse({
          run_id: "run-active-new",
          artifacts: []
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active-old/artifacts") {
        return jsonResponse({
          run_id: "run-active-old",
          artifacts: []
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active-new/logs?cursor=0&limit=200") {
        return jsonResponse({
          run_id: "run-active-new",
          items: [],
          next_cursor: 0,
          eof: true
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active-old/logs?cursor=0&limit=200") {
        return jsonResponse({
          run_id: "run-active-old",
          items: [],
          next_cursor: 0,
          eof: true
        });
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }

      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    await screen.findByTestId("run-status-run-id");
    expect(screen.getByTestId("run-status-run-id")).toHaveTextContent("run-active-new");

    fireEvent.click(screen.getByRole("button", { name: /^run-active-old$/i }));
    await waitFor(() => {
      expect(screen.getByTestId("run-status-run-id")).toHaveTextContent("run-active-old");
    });

    await waitFor(() => {
      const polls = fetchMock.mock.calls.filter(
        (call) => typeof call[0] === "string" && call[0] === "/api/pipeline/runs?limit=100"
      );
      expect(polls.length).toBeGreaterThan(1);
    }, { timeout: 3500 });
    expect(screen.getByTestId("run-status-run-id")).toHaveTextContent("run-active-old");
  });

  it("cancels selected active run and updates status", async () => {
    let canceled = false;

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({
          items: [
            {
              run_id: "run-active",
              pipeline: "refresh",
              status: canceled ? "failed" : "running",
              started_at: "2026-04-03T12:01:00Z",
              finished_at: canceled ? "2026-04-03T12:01:09Z" : null,
              warnings: canceled ? ["user canceled run"] : [],
              error_code: canceled ? "run_canceled" : null,
              error: canceled ? "run canceled by request" : null
            }
          ]
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active") {
        return jsonResponse({
          run_id: "run-active",
          pipeline: "refresh",
          status: canceled ? "failed" : "running",
          started_at: "2026-04-03T12:01:00Z",
          finished_at: canceled ? "2026-04-03T12:01:09Z" : null,
          current_step: canceled ? "refresh.step1.collect" : "refresh.step1.collect",
          warnings: canceled ? ["user canceled run"] : [],
          error_code: canceled ? "run_canceled" : null,
          error: canceled ? "run canceled by request" : null
        });
      }
      if (method === "POST" && url === "/api/pipeline/runs/run-active/cancel") {
        canceled = true;
        return jsonResponse({
          run_id: "run-active",
          status: "cancel_requested"
        }, 202);
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active/artifacts") {
        return jsonResponse({
          run_id: "run-active",
          artifacts: []
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active/logs?cursor=0&limit=200") {
        if (canceled) {
          return jsonResponse({
            run_id: "run-active",
            items: [
              {
                cursor: 0,
                timestamp: "2026-04-03T12:01:01Z",
                level: "info",
                message: "run queued",
                fields: {
                  pipeline: "refresh"
                }
              },
              {
                cursor: 1,
                timestamp: "2026-04-03T12:01:09Z",
                level: "error",
                step_id: "refresh.step3.findings",
                message: "run failed",
                fields: {
                  error_code: "run_canceled",
                  error: "run canceled by request"
                }
              }
            ],
            next_cursor: 2,
            eof: true
          });
        }
        return jsonResponse({
          run_id: "run-active",
          items: [
            {
              cursor: 0,
              timestamp: "2026-04-03T12:01:01Z",
              level: "info",
              message: "run queued",
              fields: {
                pipeline: "refresh"
              }
            }
          ],
          next_cursor: 1,
          eof: true
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=reports%2Fcoverage%2Fsummary.md") {
        return textResponse("Coverage: 70%\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=reports%2Fcoverage%2Fopen-questions.md") {
        return textResponse("- verify ownership\n");
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }

      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    await screen.findByTestId("run-status-run-id");
    await waitFor(() => {
      expect(screen.getByTestId("run-cancel-btn")).toBeEnabled();
    });
    fireEvent.click(screen.getByTestId("run-cancel-btn"));

    await screen.findByText(/Cancel requested for run-active/i);
    await waitFor(() => {
      expect(screen.getByTestId("run-status-value")).toHaveTextContent("failed");
    });
    await screen.findByText(/Error code:\s*run_canceled/i);
    fireEvent.change(screen.getByTestId("run-logs-view-select"), { target: { value: "line+fields" } });
    await waitFor(() => {
      expect(screen.getByTestId("run-logs-content")).toHaveTextContent('"error_code": "run_canceled"');
    });
  });

  it("shows cancel endpoint conflict/not-found responses in UI status", async () => {
    let mode: "not_found" | "not_cancelable" = "not_found";

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({
          items: [
            {
              run_id: "run-active",
              pipeline: "refresh",
              status: "running",
              started_at: "2026-04-03T12:01:00Z",
              finished_at: null,
              warnings: [],
              error_code: null,
              error: null
            }
          ]
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active") {
        return jsonResponse({
          run_id: "run-active",
          pipeline: "refresh",
          status: "running",
          started_at: "2026-04-03T12:01:00Z",
          finished_at: null,
          current_step: "refresh.step1.collect",
          warnings: [],
          error_code: null,
          error: null
        });
      }
      if (method === "POST" && url === "/api/pipeline/runs/run-active/cancel") {
        if (mode === "not_found") {
          mode = "not_cancelable";
          return jsonResponse(
            {
              error: {
                code: "run_not_found",
                message: "run not found"
              }
            },
            404
          );
        }
        return jsonResponse(
          {
            error: {
              code: "run_not_cancelable",
              message: "run is already terminal"
            }
          },
          409
        );
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active/artifacts") {
        return jsonResponse({
          run_id: "run-active",
          artifacts: []
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active/logs?cursor=0&limit=200") {
        return jsonResponse({
          run_id: "run-active",
          items: [],
          next_cursor: 0,
          eof: true
        });
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }

      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    await screen.findByTestId("run-status-run-id");
    await waitFor(() => {
      expect(screen.getByTestId("run-cancel-btn")).toBeEnabled();
    });

    fireEvent.click(screen.getByTestId("run-cancel-btn"));
    await screen.findByText(/Selected run no longer exists\./i);

    fireEvent.click(screen.getByTestId("run-cancel-btn"));
    await screen.findByText(/Selected run is already terminal\./i);
  });

  it("clears stale selected run details when selected run disappears from list", async () => {
    let runsCalls = 0;

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        runsCalls += 1;
        if (runsCalls === 1) {
          return jsonResponse({
            items: [
              {
                run_id: "run-active",
                pipeline: "refresh",
                status: "running",
                started_at: "2026-04-03T12:01:00Z",
                finished_at: null,
                warnings: [],
                error_code: null,
                error: null
              }
            ]
          });
        }
        return jsonResponse({ items: [] });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active") {
        return jsonResponse({
          run_id: "run-active",
          pipeline: "refresh",
          status: "running",
          started_at: "2026-04-03T12:01:00Z",
          finished_at: null,
          current_step: "refresh.step1.collect",
          warnings: [],
          error_code: null,
          error: null
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active/artifacts") {
        return jsonResponse({
          run_id: "run-active",
          artifacts: []
        });
      }
      if (method === "GET" && url === "/api/pipeline/runs/run-active/logs?cursor=0&limit=200") {
        return jsonResponse({
          run_id: "run-active",
          items: [],
          next_cursor: 0,
          eof: true
        });
      }
      if (method === "GET" && url === "/api/workspace/manifest") {
        return jsonResponse({
          content: "version: 1\nrepos:\n  - name: payments-service\n    path: /tmp/payments-service\n"
        });
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        return textResponse("# Charter\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=charter%2Fwizard%2Fstep0-contract.json") {
        return textResponse("not found", 404);
      }

      const runLogsResponse = maybeRunLogsResponse(method, url);
      if (runLogsResponse) {
        return runLogsResponse;
      }

      return jsonResponse(
        {
          error: {
            code: "not_found",
            message: `unhandled request: ${method} ${url}`
          }
        },
        404
      );
    });

    vi.stubGlobal("fetch", fetchMock);
    render(<App />);

    await screen.findByTestId("run-status-run-id");
    expect(screen.getByTestId("run-status-run-id")).toHaveTextContent("run-active");
    expect(screen.getByTestId("run-cancel-btn")).toBeEnabled();

    await waitFor(
      () => {
        const polls = fetchMock.mock.calls.filter(
          (call) => typeof call[0] === "string" && call[0] === "/api/pipeline/runs?limit=100"
        );
        expect(polls.length).toBeGreaterThan(1);
      },
      { timeout: 3500 }
    );

    await waitFor(
      () => {
        expect(screen.queryByTestId("run-status-panel")).not.toBeInTheDocument();
      },
      { timeout: 3500 }
    );
    expect(screen.getByTestId("run-cancel-btn")).toBeDisabled();
  });
});
