import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("runs mocked flow open -> validate -> run -> inspect", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

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
        return jsonResponse({ run_id: "run-1", status: "queued" });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-1") {
        return jsonResponse({
          run_id: "run-1",
          pipeline: "init",
          status: "succeeded",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: "2026-04-03T12:00:02Z"
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
});
