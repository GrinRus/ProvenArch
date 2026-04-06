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

  it("does not start polling loop when there are no active runs", async () => {
    vi.useFakeTimers();

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

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

    expect(screen.getByText(/Запуски анализа/i)).toBeInTheDocument();

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

    await screen.findByText(/Запуски анализа/i);
    await screen.findByRole("button", { name: /run-queued/i });
    await screen.findByRole("button", { name: /run-ok/i });
    await screen.findByRole("button", { name: /run-failed/i });
    expect(screen.getByText(/Running:\s*1\s*\|\s*Succeeded:\s*1\s*\|\s*Failed:\s*1/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /run-failed/i }));
    await screen.findByText(/Error code:\s*runner_parse_failed/i);
    await screen.findByRole("button", { name: /reports\/taskruns\/run-failed.json/i });
  });
});
