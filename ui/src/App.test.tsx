import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { vi } from "vitest";

import App from "./App";

type MockJSON = Record<string, unknown>;

type FetchMockState = {
  runID?: string;
  runStarted?: boolean;
  runList?: MockJSON[];
  runLogs?: Record<string, MockJSON>;
  runStatus?: Record<string, MockJSON>;
  runArtifacts?: Record<string, MockJSON>;
  runReviewSummary?: Record<string, MockJSON>;
  gitDiff?: MockJSON;
  artifactText?: Record<string, string>;
  baselineBundleWarnings?: MockJSON[];
  cancelResponses?: Record<string, { status: number; body?: MockJSON }>;
  doctorResponse?: MockJSON;
  pathSuggestions?: MockJSON[];
  validateResponse?: MockJSON;
  validateStatus?: number;
  manifestContent?: string;
  qaResponse?: MockJSON;
  qaRunID?: string;
  qaRuns?: MockJSON[];
  qaRunResponses?: Record<string, MockJSON>;
  onboardingStatus?: MockJSON;
  onboardingWorkspaceSelectionStatus?: MockJSON;
  systemVersion?: MockJSON;
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
  const runReviewSummary = state.runReviewSummary ?? {
    [runID]: {
      run_id: runID,
      pipeline: "init",
      status: "succeeded",
      started_at: "2026-04-03T12:00:00Z",
      finished_at: "2026-04-03T12:00:02Z",
      current_step: "init.step4.proposals",
      warnings: [],
      error_code: null,
      error: null,
      steps: [
        {
          step_id: "step0_constitution",
          label: "Charter",
          state: "done",
          provider: "fake",
          artifact_count: 0,
          artifact_paths: [],
          taskrun_paths: [],
          warnings_count: 0,
          errors_count: 0,
          last_message: "charter ready",
        },
        {
          step_id: "step1_collect",
          label: "Collect",
          state: "done",
          provider: "fake",
          artifact_count: 0,
          artifact_paths: [],
          taskrun_paths: [],
          warnings_count: 0,
          errors_count: 0,
          last_message: "sources collected",
        },
        {
          step_id: "step2_as_is",
          label: "As-is docs",
          state: "done",
          provider: "fake",
          artifact_count: 1,
          artifact_paths: ["reports/as-is/overview.md"],
          taskrun_paths: [],
          warnings_count: 0,
          errors_count: 0,
          last_message: "overview generated",
        },
        {
          step_id: "step3_findings",
          label: "Findings",
          state: "done",
          provider: "fake",
          artifact_count: 1,
          artifact_paths: ["reports/findings/findings.md"],
          taskrun_paths: [],
          warnings_count: 0,
          errors_count: 0,
          last_message: "findings generated",
        },
        {
          step_id: "step4_proposals",
          label: "Proposals",
          state: "done",
          provider: "fake",
          artifact_count: 1,
          artifact_paths: ["proposals/proposal-payments/proposal.md"],
          taskrun_paths: [],
          warnings_count: 0,
          errors_count: 0,
          last_message: "proposals generated",
        },
      ],
    },
  };
  const defaultGitDiff = state.gitDiff ?? {
    ok: true,
    workspace: "/tmp/workspace",
    run_id: runID,
    step_id: null,
    selected_path: "reports/coverage/summary.md",
    selected_file: {
      path: "reports/coverage/summary.md",
      folder: "reports/coverage",
      status: "modified",
      additions: 1,
      deletions: 0,
      binary: false,
    },
    files: [
      {
        path: "reports/coverage/summary.md",
        folder: "reports/coverage",
        status: "modified",
        additions: 1,
        deletions: 0,
        binary: false,
      },
      {
        path: "model/entities/payments-service.yaml",
        folder: "model/entities",
        status: "new",
        additions: 1,
        deletions: 0,
        binary: false,
      },
      {
        path: "proposals/adr-001.md",
        folder: "proposals",
        status: "new",
        additions: 1,
        deletions: 0,
        binary: false,
      },
    ],
    folders: [
      { folder: "reports/coverage", files: 1, additions: 1, deletions: 0 },
      { folder: "model/entities", files: 1, additions: 1, deletions: 0 },
      { folder: "proposals", files: 1, additions: 1, deletions: 0 },
    ],
    hunks: [
      {
        header: "@@ -1 +1,2 @@",
        lines: [
          { kind: "context", old_line: 1, new_line: 1, content: "Coverage: 84%" },
          { kind: "add", new_line: 2, content: "Workspace Git diff is reviewable." },
        ],
      },
    ],
    message: "Workspace Git diff loaded.",
    empty: false,
  };
  const qaRunID = state.qaRunID ?? "qa-run-1";
  let onboardingStatus: MockJSON = state.onboardingStatus ?? {
    ok: true,
    launcher_mode: false,
    workspace_selected: true,
    workspace_ready: true,
    workspace: "/tmp/workspace",
    manifest_present: true,
    runtime: {
      selected: true,
      runtime: "fake",
      runtime_provider: "claude-code",
      provider_source: "default",
    },
    can_enter_console: true,
    recent_workspaces: [],
  };

  const artifactText: Record<string, string> = {
    "charter/overview.md": "# Charter\n",
    "reports/coverage/summary.md": "Coverage: 84%\n",
    "reports/coverage/open-questions.md": "- Clarify owners\n",
    ...(state.artifactText ?? {}),
  };
  const runtimeTimeoutEffective = {
    step_timeout_sec: 1800,
    heartbeat_sec: 30,
    pipeline_timeout_sec: 2400,
    pipeline_kill_grace_sec: 30,
    api_ready_timeout_sec: 60,
    api_init_timeout_sec: 120,
    ui_init_poll_timeout_sec: 900,
    ui_cancel_poll_timeout_sec: 420,
  };
  let runtimeTimeoutPersisted: Record<string, number> = { step_timeout_sec: 1800 };
  let runtimeTimeoutSource: Record<string, string> = { step_timeout_sec: "workspace" };
  let runtimeExecutionPersisted: Record<string, string | number> = { strategy: "sequential", max_parallel_tasks: 1 };
  let runtimeExecutionEffective: Record<string, string | number> = {
    strategy: "sequential",
    max_parallel_tasks: 1,
    failure_policy: "best_effort",
    shard_discovery_mode: "heuristics",
  };
  let runtimeExecutionSource: Record<string, string> = { strategy: "workspace" };
  let runtimePermissionPersisted: Record<string, string> = { mode: "trusted_full_access" };
  let runtimePermissionEffective: Record<string, string> = {
    mode: "trusted_full_access",
    approval_channel: "fail_fast",
  };
  let runtimePermissionSource: Record<string, string> = { mode: "workspace" };
  const baselineBundleManifest = {
    schema_version: 1,
    bundle_version: 1,
    prompt_surface_policy: {
      live_headless_source: "skills/prompt-packs/*.md",
      reference_only_pattern: "skills/*/prompts/*.md",
    },
    editable_artifacts: [
      { path: "charter/overview.md", label: "charter/overview.md", category: "charter" },
      { path: "charter/cards/domains/payments.md", label: "payments domain", category: "domain-card" },
      { path: "charter/cards/teams/platform.md", label: "platform team", category: "team-card" },
      {
        path: "skills/prompt-packs/findings.md",
        label: "skills/prompt-packs/findings.md",
        category: "prompt-pack",
        prompt_usage: "live-consumed",
      },
      {
        path: "skills/prompt-packs/qa.md",
        label: "skills/prompt-packs/qa.md",
        category: "prompt-pack",
        prompt_usage: "live-consumed",
      },
      {
        path: "skills/findings/prompts/system.md",
        label: "skills/findings/prompts/system.md (reference-only)",
        category: "skill-prompt",
        prompt_usage: "reference-only",
      },
    ],
  };

  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === "string" ? input : input.toString();
    const method = (init?.method ?? "GET").toUpperCase();

    if (method === "GET" && url === "/api/system/version") {
      return jsonResponse(
        state.systemVersion ?? {
          version: "dev",
          commit: "none",
          built: "unknown",
          ui_bundle: "embedded",
        },
      );
    }

    if (method === "GET" && url === "/api/onboarding/status") {
      return jsonResponse(onboardingStatus);
    }

    if (method === "GET" && url.startsWith("/api/onboarding/path-suggestions")) {
      const parsed = new URL(url, "http://localhost");
      const kind = parsed.searchParams.get("kind") ?? "workspace";
      const query = parsed.searchParams.get("query") ?? "";
      return jsonResponse({
        ok: true,
        kind,
        query,
        items:
          state.pathSuggestions ??
          [
            {
              path: kind === "repo" ? "/tmp/my-service" : "/tmp/onboarding-workspace",
              label: kind === "repo" ? "my-service" : "onboarding-workspace",
              exists: true,
              kind: kind === "repo" ? "git_repo" : "workspace",
              source: "query",
            },
          ],
      });
    }

    if (method === "POST" && url === "/api/onboarding/workspace") {
      const payload = JSON.parse(String(init?.body ?? "{}")) as { path?: string; create?: boolean };
      onboardingStatus =
        state.onboardingWorkspaceSelectionStatus ?? {
          ...onboardingStatus,
          workspace_selected: true,
          workspace_ready: false,
          manifest_present: false,
          workspace: payload.path ?? "/tmp/onboarding-workspace",
          can_enter_console: false,
          recent_workspaces: [
            {
              path: payload.path ?? "/tmp/onboarding-workspace",
              last_opened_at: "2026-06-04T10:00:00Z",
              exists: true,
            },
            ...(((onboardingStatus.recent_workspaces as MockJSON[] | undefined) ?? []).filter((item) => item.path !== payload.path)),
          ].slice(0, 10),
        };
      return jsonResponse(onboardingStatus);
    }

    if (method === "POST" && url === "/api/onboarding/recent-workspaces/forget") {
      const payload = JSON.parse(String(init?.body ?? "{}")) as { path?: string };
      onboardingStatus = {
        ...onboardingStatus,
        recent_workspaces: ((onboardingStatus.recent_workspaces as MockJSON[] | undefined) ?? []).filter((workspace) => workspace.path !== payload.path),
      };
      return jsonResponse(onboardingStatus);
    }

    if (method === "POST" && url === "/api/onboarding/runtime") {
      onboardingStatus = {
        ...onboardingStatus,
        runtime: {
          selected: true,
          runtime: "fake",
          runtime_provider: "claude-code",
          provider_source: "override",
        },
        can_enter_console: onboardingStatus.workspace_ready === true,
      };
      return jsonResponse(onboardingStatus);
    }

    if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
      if (state.runList) {
        return jsonResponse({ items: state.runList });
      }
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
        content:
          state.manifestContent ??
          'version: 1\nrepos:\n  - name: "my-service"\n    git_url: "https://github.com/org/my-service.git"\ndocs:\n  imports_path: "./docs/imports"\n',
      });
    }

    if (method === "PUT" && url === "/api/workspace/manifest") {
      onboardingStatus = {
        ...onboardingStatus,
        workspace_ready: true,
        manifest_present: true,
        can_enter_console: ((onboardingStatus.runtime as MockJSON | undefined)?.selected as boolean | undefined) === true,
      };
      return jsonResponse({ ok: true });
    }

    if (method === "GET" && url === "/api/workspace/bundle") {
      return jsonResponse({
        ok: true,
        workspace: "/tmp/workspace",
        manifest: baselineBundleManifest,
        warnings: state.baselineBundleWarnings ?? [],
      });
    }

    if (method === "GET" && url === "/api/runtime/timeouts") {
      return jsonResponse({
        ok: true,
        persisted: runtimeTimeoutPersisted,
        effective: runtimeTimeoutEffective,
        source: runtimeTimeoutSource,
      });
    }

    if (method === "GET" && url === "/api/runtime/execution") {
      return jsonResponse({
        ok: true,
        persisted: runtimeExecutionPersisted,
        effective: runtimeExecutionEffective,
        source: runtimeExecutionSource,
      });
    }

    if (method === "GET" && url === "/api/runtime/permissions") {
      return jsonResponse({
        ok: true,
        persisted: runtimePermissionPersisted,
        effective: runtimePermissionEffective,
        source: runtimePermissionSource,
      });
    }

    if (method === "GET" && url === "/api/runtime/profile") {
      return jsonResponse({
        ok: true,
        permissions: {
          persisted: runtimePermissionPersisted,
          effective: runtimePermissionEffective,
          source: runtimePermissionSource,
        },
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

    if (method === "GET" && url.startsWith("/api/system/doctor")) {
      return jsonResponse(
        state.doctorResponse ?? {
          ok: true,
          summary: "ready",
          checks: [
            { id: "git", label: "Git", status: "pass", message: "git found" },
            { id: "workspace", label: "Workspace", status: "pass", message: "workspace is writable" },
            { id: "embedded_ui", label: "Embedded UI", status: "pass", message: "embedded UI assets are present" },
            { id: "runtime_provider", label: "Runtime provider", status: "pass", message: "fake runtime selected" },
          ],
        },
      );
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
      return jsonResponse(
        state.validateResponse ?? {
          ok: true,
          workspace: "/tmp/workspace",
          warnings: [],
          errors: [],
        },
        state.validateStatus ?? 200,
      );
    }

    if (method === "POST" && (url === "/api/pipeline/init" || url === "/api/pipeline/refresh")) {
      state.runStarted = true;
      return jsonResponse({ run_id: runID, status: "queued" });
    }

    const cancelMatch = url.match(/^\/api\/pipeline\/runs\/([^/]+)\/cancel$/);
    if (method === "POST" && cancelMatch) {
      const requestedRunID = decodeURIComponent(cancelMatch[1]);
      const configured = state.cancelResponses?.[requestedRunID];
      return jsonResponse(configured?.body ?? { ok: true }, configured?.status ?? 202);
    }

    const runStatusMatch = url.match(/^\/api\/pipeline\/runs\/([^/]+)$/);
    if (method === "GET" && runStatusMatch) {
      const requestedRunID = decodeURIComponent(runStatusMatch[1]);
      return jsonResponse((runStatus[requestedRunID] ?? {}) as MockJSON);
    }

    const runReviewMatch = url.match(/^\/api\/pipeline\/runs\/([^/]+)\/review-summary$/);
    if (method === "GET" && runReviewMatch) {
      const requestedRunID = decodeURIComponent(runReviewMatch[1]);
      return jsonResponse((runReviewSummary[requestedRunID] ?? {}) as MockJSON);
    }

    const runArtifactsMatch = url.match(/^\/api\/pipeline\/runs\/([^/]+)\/artifacts$/);
    if (method === "GET" && runArtifactsMatch) {
      const requestedRunID = decodeURIComponent(runArtifactsMatch[1]);
      return jsonResponse((runArtifacts[requestedRunID] ?? { run_id: requestedRunID, artifacts: [] }) as MockJSON);
    }

    const runLogsMatch = url.match(/^\/api\/pipeline\/runs\/([^/]+)\/logs\?/);
    if (method === "GET" && runLogsMatch) {
      const requestedRunID = decodeURIComponent(runLogsMatch[1]);
      return jsonResponse((runLogs[requestedRunID] ?? { run_id: requestedRunID, items: [], next_cursor: 0, eof: true }) as MockJSON);
    }

    if (method === "POST" && url === "/api/artifacts/write") {
      return jsonResponse({ ok: true });
    }

    if (method === "POST" && url === "/api/qa/runs") {
      return jsonResponse({ run_id: qaRunID, status: "queued" }, 202);
    }

    if (method === "GET" && url.startsWith("/api/qa/runs/")) {
      const requestedQARunID = decodeURIComponent(url.slice("/api/qa/runs/".length));
      const qaPayload = state.qaResponse ?? {
        answer: "payments-service is owned by Platform Architecture.",
        citations: [{ path: "reports/as-is/overview.md", reason: "ownership evidence" }],
        unresolved: ["confirm escalation owner"],
        confidence: 0.82,
      };
      const configuredPayload =
        state.qaRunResponses?.[requestedQARunID] ??
        (requestedQARunID === qaRunID ? qaPayload : undefined) ??
        state.qaRuns?.find((item) => item.run_id === requestedQARunID) ??
        qaPayload;
      return jsonResponse(
        {
          run_id: requestedQARunID,
          pipeline: "qa",
          status: "succeeded",
          started_at: "2026-04-03T12:00:03Z",
          finished_at: "2026-04-03T12:00:04Z",
          question: "Who owns payments?",
          current_step: "qa.ask",
          runtime_provider: "claude-code",
          provider: "fake",
          generated_at: "2026-04-03T12:00:04Z",
          ...configuredPayload,
        },
      );
    }

    if (method === "GET" && url.startsWith("/api/qa/runs?")) {
      return jsonResponse({ items: state.qaRuns ?? [] });
    }

    if (method === "POST" && url === "/api/qa/ask") {
      return jsonResponse(
        state.qaResponse ?? {
          answer: "payments-service is owned by Platform Architecture.",
          citations: [{ path: "reports/as-is/overview.md", reason: "ownership evidence" }],
          unresolved: ["confirm escalation owner"],
          confidence: 0.82,
        },
      );
    }

    if (method === "PUT" && url === "/api/runtime/timeouts") {
      const payload = JSON.parse(String(init?.body ?? "{}")) as { timeouts?: Record<string, number> };
      runtimeTimeoutPersisted = { ...(payload.timeouts ?? {}) };
      runtimeTimeoutSource = {};
      for (const key of Object.keys(runtimeTimeoutPersisted)) {
        runtimeTimeoutSource[key] = "workspace";
      }
      Object.assign(runtimeTimeoutEffective, runtimeTimeoutPersisted);
      return jsonResponse({
        ok: true,
        persisted: runtimeTimeoutPersisted,
        effective: runtimeTimeoutEffective,
        source: runtimeTimeoutSource,
      });
    }

    if (method === "PUT" && url === "/api/runtime/execution") {
      const payload = JSON.parse(String(init?.body ?? "{}")) as { execution?: Record<string, string | number> };
      runtimeExecutionPersisted = { ...(payload.execution ?? {}) };
      runtimeExecutionSource = {};
      for (const key of Object.keys(runtimeExecutionPersisted)) {
        runtimeExecutionSource[key] = "workspace";
      }
      runtimeExecutionEffective = {
        ...runtimeExecutionEffective,
        ...runtimeExecutionPersisted,
      };
      return jsonResponse({
        ok: true,
        persisted: runtimeExecutionPersisted,
        effective: runtimeExecutionEffective,
        source: runtimeExecutionSource,
      });
    }

    if (method === "PUT" && url === "/api/runtime/permissions") {
      const payload = JSON.parse(String(init?.body ?? "{}")) as { permissions?: Record<string, string> };
      runtimePermissionPersisted = { ...(payload.permissions ?? {}) };
      runtimePermissionSource = {};
      for (const key of Object.keys(runtimePermissionPersisted)) {
        runtimePermissionSource[key] = "workspace";
      }
      runtimePermissionEffective = {
        ...runtimePermissionEffective,
        ...runtimePermissionPersisted,
      };
      return jsonResponse({
        ok: true,
        persisted: runtimePermissionPersisted,
        effective: runtimePermissionEffective,
        source: runtimePermissionSource,
      });
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

    if (method === "GET" && url.startsWith("/api/git/diff")) {
      const parsed = new URL(url, "http://localhost");
      const selectedPath = parsed.searchParams.get("path");
      const requestedRunID = parsed.searchParams.get("run_id");
      const scopedGitDiff = { ...defaultGitDiff, run_id: requestedRunID } as MockJSON;
      if (!selectedPath) {
        return jsonResponse(scopedGitDiff);
      }
      const files = (scopedGitDiff.files as MockJSON[] | undefined) ?? [];
      const selectedFile = files.find((file) => file.path === selectedPath) ?? {
        path: selectedPath,
        folder: selectedPath.includes("/") ? selectedPath.slice(0, selectedPath.lastIndexOf("/")) : ".",
        status: "unchanged",
        additions: 0,
        deletions: 0,
        binary: false,
      };
      return jsonResponse({
        ...scopedGitDiff,
        selected_path: selectedPath,
        selected_file: selectedFile,
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
      render: vi.fn(async (id: string, graph: string) => {
        if (!/^diagram-[A-Za-z0-9_-]+$/.test(id)) {
          throw new Error(`invalid Mermaid render id: ${id}`);
        }
        return {
          svg: `<svg data-graph="${graph.replace(/"/g, "&quot;")}"></svg>`,
        };
      }),
    },
  };
});

async function renderConsoleApp() {
  const view = render(<App />);
  await screen.findByTestId("top-status-bar");
  return view;
}

describe("App", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("renders release build metadata from the system version endpoint", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        systemVersion: {
          version: "v0.1.2",
          commit: "fa3c633",
          built: "2026-06-02T13:20:26Z",
          ui_bundle: "embedded",
        },
      }),
    );

    await renderConsoleApp();

    await waitFor(() => expect(screen.getByTestId("top-status-bar")).toHaveTextContent("v0.1.2"));
  }, 15_000);

  it("keeps the console recoverable when refresh API bootstrap fails and then recovers", async () => {
    const baseFetch = createFetchMock();
    let failNextStatus = false;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (failNextStatus && method === "GET" && url === "/api/onboarding/status") {
        failNextStatus = false;
        throw new Error("server unavailable");
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    failNextStatus = true;
    fireEvent.click(screen.getByTestId("console-refresh-btn"));

    expect(await screen.findByText("Error: server unavailable")).toBeInTheDocument();
    expect(screen.getByTestId("top-status-bar")).toBeInTheDocument();
    expect(screen.getByTestId("console-refresh-btn")).toBeEnabled();
    expect(screen.getByTestId("workspace-panel")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("console-refresh-btn"));

    await waitFor(() => expect(screen.queryByText("Error: server unavailable")).not.toBeInTheDocument());
    expect(screen.getByTestId("top-status-bar")).toBeInTheDocument();
    expect(screen.getByTestId("console-refresh-btn")).toBeEnabled();
  });

  it("keeps the console recoverable when workspace manifest reload fails", async () => {
    const baseFetch = createFetchMock();
    let failNextManifest = false;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (failNextManifest && method === "GET" && url === "/api/workspace/manifest") {
        failNextManifest = false;
        throw new Error("workspace manifest unavailable");
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    failNextManifest = true;
    fireEvent.click(screen.getByTestId("console-refresh-btn"));

    expect(await screen.findByText("Error: workspace manifest unavailable")).toBeInTheDocument();
    expect(screen.getByTestId("top-status-bar")).toBeInTheDocument();
    expect(screen.getByTestId("console-refresh-btn")).toBeEnabled();
    expect(screen.getByTestId("workspace-panel")).toBeInTheDocument();
  });

  it("supports stage navigation and settings relocation without compatibility controls", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    expect(screen.getByTestId("workspace-panel")).toBeInTheDocument();
    expect(screen.queryByTestId("runtime-timeouts-panel")).not.toBeInTheDocument();
    expect(screen.queryByTestId(`tab-${"settings"}`)).not.toBeInTheDocument();
    expect(screen.queryByTestId(`setup-${"stepper"}`)).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("stage-readiness"));

    expect(await screen.findByTestId("runtime-timeouts-panel")).toBeInTheDocument();
    expect(screen.getByTestId("runtime-execution-panel")).toBeInTheDocument();
    expect(screen.getByTestId("runtime-permissions-panel")).toBeInTheDocument();
    expect(screen.getByTestId("runtime-step-providers-panel")).toBeInTheDocument();
    expect(screen.getByText("runtime.profile.permissions.mode")).toBeInTheDocument();
    expect(screen.getByText("runtime.profile.steps.step2_as_is.provider")).toBeInTheDocument();
    expect(screen.getAllByText("qwen-code").length).toBeGreaterThanOrEqual(1);
  }, 15_000);

  it("guides launcher onboarding through workspace, sources, runner, and console entry", async () => {
    const fetchMock = createFetchMock({
      onboardingStatus: {
        ok: true,
        launcher_mode: true,
        workspace_selected: false,
        workspace_ready: false,
        workspace: "",
        manifest_present: false,
        runtime: {
          selected: false,
          runtime: "fake",
          runtime_provider: "claude-code",
          provider_source: "default",
        },
        can_enter_console: false,
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    expect(await screen.findByTestId("onboarding-shell")).toBeInTheDocument();
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Create or open a workspace");
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Workspace path is not selected.");
    expect(screen.getByTestId("onboarding-ready-action-hint")).toHaveTextContent("select or create a workspace");
    fireEvent.change(screen.getByLabelText("Architecture workspace path"), {
      target: { value: "/tmp/onboarding-workspace" },
    });
    fireEvent.click(screen.getByTestId("onboarding-workspace-save"));

    await screen.findByText("Selected: /tmp/onboarding-workspace");
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Save and validate sources");
    expect(fetchMock.mock.calls.some((call) => call[0] === "/api/workspace/bundle")).toBe(false);
    fireEvent.click(screen.getByTestId("onboarding-sources-save"));

    expect(await screen.findByText("Sources validated.")).toBeInTheDocument();
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Select the runner");
    fireEvent.click(screen.getByTestId("onboarding-runtime-save"));

    await waitFor(() => expect(screen.getByTestId("onboarding-enter-console")).not.toBeDisabled());
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Check local readiness");
    expect(screen.getByTestId("onboarding-run-first-analysis")).toBeDisabled();
    expect(screen.getByTestId("onboarding-ready-step")).toHaveTextContent("Local readiness checked");
    expect(screen.getByTestId("onboarding-ready-step")).toHaveTextContent("Check local readiness before first analysis.");
    expect(screen.getByTestId("onboarding-ready-action-hint")).toHaveTextContent("First analysis waits for: run local readiness check");

    fireEvent.click(screen.getByRole("button", { name: "Check readiness" }));
    await screen.findByTestId("onboarding-doctor-result");
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Run first analysis");
    expect(screen.getByTestId("onboarding-ready-action-hint")).toHaveTextContent("Ready to run the first analysis.");
    expect(screen.getByTestId("onboarding-run-first-analysis")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("onboarding-enter-console"));

    expect(await screen.findByTestId("top-status-bar")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith("/api/onboarding/workspace", expect.objectContaining({ method: "POST" }));
    expect(fetchMock).toHaveBeenCalledWith("/api/workspace/manifest", expect.objectContaining({ method: "PUT" }));
    expect(fetchMock).toHaveBeenCalledWith("/api/onboarding/runtime", expect.objectContaining({ method: "POST" }));
  });

  it("fills onboarding workspace and local repo paths from suggestions", async () => {
    const fetchMock = createFetchMock({
      onboardingStatus: {
        ok: true,
        launcher_mode: true,
        workspace_selected: false,
        workspace_ready: false,
        workspace: "",
        manifest_present: false,
        runtime: {
          selected: false,
          runtime: "fake",
          runtime_provider: "claude-code",
          provider_source: "default",
        },
        can_enter_console: false,
      },
      pathSuggestions: [
        {
          path: "/tmp/suggested-workspace",
          label: "suggested-workspace",
          exists: true,
          kind: "workspace",
          source: "query",
        },
        {
          path: "/tmp/my-service",
          label: "my-service",
          exists: true,
          kind: "git_repo",
          source: "query",
        },
      ],
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    const workspaceInput = await screen.findByLabelText("Architecture workspace path");
    fireEvent.focus(workspaceInput);
    fireEvent.click(await screen.findByRole("option", { name: /suggested-workspace/i }));
    expect(workspaceInput).toHaveValue("/tmp/suggested-workspace");

    fireEvent.click(screen.getByTestId("onboarding-workspace-save"));
    await screen.findByText("Selected: /tmp/suggested-workspace");

    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "" } });
    fireEvent.change(screen.getByLabelText("Source type"), { target: { value: "path" } });
    const repoPathInput = await screen.findByLabelText("Local checkout path");
    fireEvent.focus(repoPathInput);
    fireEvent.click(await screen.findByRole("option", { name: /my-service/i }));

    expect(repoPathInput).toHaveValue("/tmp/my-service");
    expect(screen.getByLabelText("Name")).toHaveValue("my-service");
  });

  it("shows provider ID and executable guidance for missing onboarding runner commands", async () => {
    const fetchMock = createFetchMock({
      onboardingStatus: {
        ok: true,
        launcher_mode: true,
        workspace_selected: false,
        workspace_ready: false,
        workspace: "",
        manifest_present: false,
        runtime: {
          selected: false,
          runtime: "fake",
          runtime_provider: "claude-code",
          provider_source: "default",
        },
        can_enter_console: false,
      },
      doctorResponse: {
        ok: false,
        summary: "needs attention",
        checks: [
          { id: "git", label: "Git", status: "pass", message: "git found" },
          {
            id: "runtime_provider",
            label: "Runtime provider",
            status: "fail",
            message: "Provider ID: claude-code; executable not found; checked: claude, claude-code",
            suggestion: "Install claude or set ACP_CLAUDE_CMD to the provider command.",
          },
        ],
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    fireEvent.change(await screen.findByLabelText("Architecture workspace path"), {
      target: { value: "/tmp/onboarding-workspace" },
    });
    fireEvent.click(screen.getByTestId("onboarding-workspace-save"));
    await screen.findByText("Selected: /tmp/onboarding-workspace");

    fireEvent.change(screen.getByLabelText("Runtime"), { target: { value: "headless" } });
    fireEvent.click(screen.getByTestId("onboarding-sources-save"));
    expect(await screen.findByText("Sources validated.")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("onboarding-runtime-save"));
    await waitFor(() => expect(screen.getByTestId("onboarding-enter-console")).not.toBeDisabled());
    fireEvent.click(screen.getByText("Check readiness"));

    const doctorPanel = await screen.findByTestId("onboarding-doctor-result");
    expect(doctorPanel).toHaveTextContent("Provider ID: claude-code");
    expect(doctorPanel).toHaveTextContent("checked: claude, claude-code");
    expect(doctorPanel).toHaveTextContent("ACP_CLAUDE_CMD");
    const runnerRecovery = screen.getByTestId("onboarding-runner-recovery");
    expect(runnerRecovery).toHaveTextContent("Provider setup for first analysis");
    expect(runnerRecovery).toHaveTextContent("claude-code");
    expect(runnerRecovery).toHaveTextContent("claude or claude-code");
    expect(runnerRecovery).toHaveTextContent("ACP_CLAUDE_CMD");
    expect(runnerRecovery).toHaveTextContent("Runtime provider: fail");
    expect(runnerRecovery).toHaveTextContent("Command unavailable");
    expect(runnerRecovery).toHaveTextContent("Binary discovery");
    expect(runnerRecovery).toHaveTextContent("Use fake baseline for a deterministic first walkthrough");
    expect(runnerRecovery).toHaveTextContent("Run Check local readiness before starting the first live analysis.");
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Fix local readiness blockers");
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Provider ID: claude-code");
  });

  it("reopens an existing workspace and enters console after validation without rewriting sources", async () => {
    const fetchMock = createFetchMock({
      onboardingStatus: {
        ok: true,
        launcher_mode: true,
        workspace_selected: false,
        workspace_ready: false,
        workspace: "",
        manifest_present: false,
        runtime: {
          selected: false,
          runtime: "fake",
          runtime_provider: "claude-code",
          provider_source: "default",
        },
        can_enter_console: false,
      },
      onboardingWorkspaceSelectionStatus: {
        ok: true,
        launcher_mode: true,
        workspace_selected: true,
        workspace_ready: true,
        workspace: "/tmp/existing-workspace",
        manifest_present: true,
        runtime: {
          selected: false,
          runtime: "fake",
          runtime_provider: "claude-code",
          provider_source: "default",
        },
        can_enter_console: false,
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    expect(await screen.findByTestId("onboarding-shell")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Architecture workspace path"), {
      target: { value: "/tmp/existing-workspace" },
    });
    fireEvent.click(screen.getByTestId("onboarding-workspace-save"));

    await screen.findByText("Selected: /tmp/existing-workspace");
    await waitFor(() => expect(fetchMock.mock.calls.some((call) => call[0] === "/api/workspace/validate")).toBe(true));

    fireEvent.click(screen.getByTestId("onboarding-runtime-save"));

    await waitFor(() => expect(screen.getByTestId("onboarding-enter-console")).not.toBeDisabled());
    expect(fetchMock.mock.calls.filter((call) => call[0] === "/api/workspace/manifest" && (call[1] as RequestInit | undefined)?.method === "PUT")).toHaveLength(0);

    fireEvent.click(screen.getByTestId("onboarding-enter-console"));
    expect(await screen.findByTestId("top-status-bar")).toBeInTheDocument();
  });

  it("opens an available recent workspace from launcher recents", async () => {
    const fetchMock = createFetchMock({
      onboardingStatus: {
        ok: true,
        launcher_mode: true,
        workspace_selected: false,
        workspace_ready: false,
        workspace: "",
        manifest_present: false,
        runtime: {
          selected: false,
          runtime: "fake",
          runtime_provider: "claude-code",
          provider_source: "default",
        },
        can_enter_console: false,
        recent_workspaces: [
          {
            path: "/tmp/recent-workspace",
            last_opened_at: "2026-06-04T10:00:00Z",
            exists: true,
          },
        ],
      },
      onboardingWorkspaceSelectionStatus: {
        ok: true,
        launcher_mode: true,
        workspace_selected: true,
        workspace_ready: true,
        workspace: "/tmp/recent-workspace",
        manifest_present: true,
        runtime: {
          selected: false,
          runtime: "fake",
          runtime_provider: "claude-code",
          provider_source: "default",
        },
        can_enter_console: false,
        recent_workspaces: [
          {
            path: "/tmp/recent-workspace",
            last_opened_at: "2026-06-04T10:00:00Z",
            exists: true,
          },
        ],
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    const recents = await screen.findByTestId("onboarding-recent-workspaces");
    expect(recents).toHaveTextContent("/tmp/recent-workspace");
    fireEvent.click(within(recents).getByRole("button", { name: "Open" }));

    await screen.findByText("Selected: /tmp/recent-workspace");
    const workspaceCall = fetchMock.mock.calls.find((call) => call[0] === "/api/onboarding/workspace" && (call[1] as RequestInit | undefined)?.method === "POST");
    expect(JSON.parse(String((workspaceCall?.[1] as RequestInit | undefined)?.body ?? "{}"))).toMatchObject({
      path: "/tmp/recent-workspace",
      create: false,
    });
  });

  it("collapses launcher recent workspaces to three and reveals the rest on demand", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        onboardingStatus: {
          ok: true,
          launcher_mode: true,
          workspace_selected: false,
          workspace_ready: false,
          workspace: "",
          manifest_present: false,
          runtime: {
            selected: false,
            runtime: "fake",
            runtime_provider: "claude-code",
            provider_source: "default",
          },
          can_enter_console: false,
          recent_workspaces: [
            { path: "/tmp/recent-workspace-1", last_opened_at: "2026-06-04T10:00:00Z", exists: true },
            { path: "/tmp/recent-workspace-2", last_opened_at: "2026-06-04T09:00:00Z", exists: true },
            { path: "/tmp/recent-workspace-3", last_opened_at: "2026-06-04T08:00:00Z", exists: true },
            { path: "/tmp/recent-workspace-4", last_opened_at: "2026-06-04T07:00:00Z", exists: true },
            { path: "/tmp/recent-workspace-5", last_opened_at: "2026-06-04T06:00:00Z", exists: false },
          ],
        },
      }),
    );

    render(<App />);

    const recents = await screen.findByTestId("onboarding-recent-workspaces");
    expect(recents).toHaveTextContent("/tmp/recent-workspace-1");
    expect(recents).toHaveTextContent("/tmp/recent-workspace-2");
    expect(recents).toHaveTextContent("/tmp/recent-workspace-3");
    expect(recents).not.toHaveTextContent("/tmp/recent-workspace-4");
    expect(recents).not.toHaveTextContent("/tmp/recent-workspace-5");

    fireEvent.click(within(recents).getByRole("button", { name: "Show 2 more workspaces" }));

    expect(recents).toHaveTextContent("/tmp/recent-workspace-4");
    expect(recents).toHaveTextContent("/tmp/recent-workspace-5");
    expect(within(recents).getAllByRole("button", { name: "Open" })[4]).toBeDisabled();

    fireEvent.click(within(recents).getByRole("button", { name: "Show fewer workspaces" }));

    expect(recents).not.toHaveTextContent("/tmp/recent-workspace-4");
    expect(recents).not.toHaveTextContent("/tmp/recent-workspace-5");
  });

  it("lets the operator forget a missing recent workspace", async () => {
    const fetchMock = createFetchMock({
      onboardingStatus: {
        ok: true,
        launcher_mode: true,
        workspace_selected: false,
        workspace_ready: false,
        workspace: "",
        manifest_present: false,
        runtime: {
          selected: false,
          runtime: "fake",
          runtime_provider: "claude-code",
          provider_source: "default",
        },
        can_enter_console: false,
        recent_workspaces: [
          {
            path: "/tmp/missing-workspace",
            last_opened_at: "2026-06-04T09:00:00Z",
            exists: false,
          },
        ],
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    const recents = await screen.findByTestId("onboarding-recent-workspaces");
    expect(within(recents).getByRole("button", { name: "Open" })).toBeDisabled();
    fireEvent.click(within(recents).getByRole("button", { name: "Forget" }));

    await screen.findByText("No recent workspaces yet.");
    expect(fetchMock).toHaveBeenCalledWith("/api/onboarding/recent-workspaces/forget", expect.objectContaining({ method: "POST" }));
  });

  it("restores workspace validation state when booting an already ready console", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    await waitFor(() => expect(fetchMock.mock.calls.some((call) => call[0] === "/api/workspace/validate")).toBe(true));
    expect(screen.getByTestId("source-repo-table")).toHaveTextContent("resolved");
    expect(screen.getByTestId("top-status-bar")).toHaveTextContent("workspace valid");
  });

  it("shows running build metadata instead of the latest public release label", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        systemVersion: {
          version: "dev-local",
          commit: "abc123",
          built: "2026-06-08T10:00:00Z",
          ui_bundle: "embedded",
        },
      }),
    );

    await renderConsoleApp();

    expect(screen.getByTestId("brand-version")).toHaveTextContent("dev-local");
    expect(screen.getByTestId("brand-version")).not.toHaveTextContent("v0.1.1 beta");
    expect(screen.getByTestId("brand-version")).toHaveAttribute("title", expect.stringContaining("commit=abc123"));
  });

  it("shows recoverable duplicate repo-name errors during onboarding source setup", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        onboardingStatus: {
          ok: true,
          launcher_mode: true,
          workspace_selected: false,
          workspace_ready: false,
          workspace: "",
          manifest_present: false,
          runtime: {
            selected: false,
            runtime: "fake",
            runtime_provider: "claude-code",
            provider_source: "default",
          },
          can_enter_console: false,
        },
      }),
    );

    render(<App />);

    await screen.findByTestId("onboarding-shell");
    fireEvent.change(screen.getByLabelText("Architecture workspace path"), {
      target: { value: "/tmp/onboarding-workspace" },
    });
    fireEvent.click(screen.getByTestId("onboarding-workspace-save"));

    await screen.findByText("Selected: /tmp/onboarding-workspace");
    fireEvent.click(screen.getByText("Add repo"));
    const nameInputs = screen.getAllByLabelText("Name");
    fireEvent.change(nameInputs[1], { target: { value: "my-service" } });

    await waitFor(() => expect(screen.getByTestId("onboarding-sources-save")).toBeDisabled());
    expect(screen.getAllByText(/repo_name_duplicate/i).length).toBeGreaterThanOrEqual(2);
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Fix source fields");
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("repo_name_duplicate");
    expect(screen.getByTestId("onboarding-ready-action-hint")).toHaveTextContent("fix source diagnostic repo_name_duplicate");
    expect(screen.getByTestId("onboarding-sources-save")).toBeDisabled();
  });

  it("renders the stage rail and switches product-flow stages", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    for (const stage of ["source", "readiness", "charter", "analysis", "review", "proposals", "ask", "publish"]) {
      expect(screen.getByTestId(`stage-${stage}`)).toBeInTheDocument();
    }

    fireEvent.click(screen.getByTestId("stage-ask"));
    expect(await screen.findByTestId("qa-panel")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("stage-analysis"));
    expect(await screen.findByTestId("runs-control-panel")).toBeInTheDocument();
  });

  it("renders V2 shell shared surfaces without hidden compatibility controls", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    expect(await screen.findByTestId("workspace-panel")).toBeInTheDocument();
    expect(screen.getByTestId("top-status-bar")).toHaveTextContent(/Permissions trusted_full_access/i);
    expect(screen.getByTestId("top-status-bar")).toHaveTextContent(/Git review pending/i);
    expect(screen.getByTestId("next-action-panel")).toBeInTheDocument();
    expect(screen.getByTestId("blockers-panel")).toBeInTheDocument();
    expect(screen.getByTestId("evidence-refs-panel")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-health-panel")).toBeInTheDocument();
    expect(screen.getByTestId("runtime-safety-panel")).toBeInTheDocument();
    expect(screen.getByTestId("git-publication-panel")).toHaveTextContent("proposal/beta-refresh");
    expect(screen.getByTestId("activity-drawer")).toHaveAccessibleName("Selected run activity drawer");
    expect(screen.queryByTestId(`setup-${"stepper"}`)).not.toBeInTheDocument();
  });

  it("routes Review empty-state recovery to Analysis instead of disabling the primary action", async () => {
    const fetchMock = createFetchMock({ runID: "run-review-recovery" });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-review"));

    expect(await screen.findByTestId("review-panel")).toBeInTheDocument();
    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("Run analysis first");
    expect(screen.getByTestId("inspector-primary-action")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("inspector-primary-action"));

    expect(await screen.findByTestId("analysis-run-progress")).toBeInTheDocument();
    expect(screen.getByTestId("stage-analysis")).toHaveClass("is-selected");
    expect(fetchMock).toHaveBeenCalledWith("/api/pipeline/init", expect.anything());
  });

  it("renders the Source V2 repo table with guided analysis scope summary", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    const sourceTable = await screen.findByTestId("source-repo-table");
    expect(screen.getByTestId("source-next-action")).toHaveTextContent("repository inventory");
    expect(screen.getByTestId("source-next-action")).toHaveTextContent("save and validate");
    expect(sourceTable).toHaveTextContent("Name");
    expect(sourceTable).toHaveTextContent("Source");
    expect(sourceTable).toHaveTextContent("Ref");
    expect(sourceTable).toHaveTextContent("Analysis include/exclude");
    expect(sourceTable).toHaveTextContent("all files");
    expect(sourceTable).toHaveTextContent("Git URL");
    expect(sourceTable).toHaveTextContent("https://github.com/org/my-service.git");
  });

  it("keeps the inspector primary action contextual for each V2 stage when review blockers exist", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [
              { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
              { path: "reports/coverage/open-questions.md", kind: "report", label: "Open questions" },
              { path: "proposals/proposal-payments/proposal.md", kind: "proposal", label: "Payments proposal" },
            ],
          },
        },
        artifactText: {
          "reports/as-is/overview.md": "# System overview\n",
          "reports/coverage/open-questions.md": "- Clarify owners\n",
          "proposals/proposal-payments/proposal.md": "# Proposal\n",
        },
      }),
    );

    await renderConsoleApp();

    await screen.findByTestId("review-panel");

    fireEvent.click(screen.getByTestId("stage-source"));
    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("Save and validate sources");
    expect(screen.getByTestId("blockers-panel")).toHaveTextContent("No hard blockers detected.");
    expect(screen.getByTestId("open-questions-panel")).toHaveTextContent("Open questions");

    fireEvent.click(screen.getByTestId("stage-charter"));
    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("Save charter contract");
    expect(screen.getByTestId("open-questions-panel")).toHaveTextContent("Open questions");

    fireEvent.click(screen.getByTestId("stage-proposals"));
    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("Review proposal");
    expect(screen.getByTestId("open-questions-panel")).toHaveTextContent("Open questions");

    fireEvent.click(screen.getByTestId("stage-ask"));
    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("Ask workspace");
    expect(screen.getByTestId("open-questions-panel")).toHaveTextContent("Open questions");
  });

  it("renders Readiness V2 cards and compact runtime profile summary", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-readiness"));

    const readinessCards = await screen.findByTestId("readiness-summary-cards");
    expect(screen.getByTestId("readiness-next-action")).toHaveTextContent("Check local readiness before first analysis.");
    expect(readinessCards).toHaveTextContent("Workspace");
    expect(readinessCards).toHaveTextContent("Repositories");
    expect(readinessCards).toHaveTextContent("Runtime provider");
    expect(readinessCards).toHaveTextContent("Permissions");
    expect(readinessCards).toHaveTextContent("Artifacts");

    const runtimeSummary = screen.getByTestId("readiness-runtime-summary");
    expect(runtimeSummary).toHaveTextContent("step 1800s / pipeline 2400s");
    expect(runtimeSummary).toHaveTextContent("sequential / max 1");
    expect(runtimeSummary).toHaveTextContent("best_effort");
    expect(runtimeSummary).toHaveTextContent("fake");
    expect(runtimeSummary).toHaveTextContent("Advanced runtime settings remain available below");
    const advancedSettings = screen.getByTestId("readiness-advanced-settings");
    expect(advancedSettings).toHaveTextContent("Timeouts, execution policy, permissions, and per-step provider overrides.");
    expect(advancedSettings).toHaveTextContent("operator tools");
    expect(advancedSettings).not.toHaveAttribute("open");
  });

  it("renders Charter V2 workbench summary, card overview, and prompt bundle status", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-charter"));

    const wizardSummary = await screen.findByTestId("charter-wizard-summary");
    expect(wizardSummary).toHaveTextContent("ProvenArch MVP");
    expect(wizardSummary).toHaveTextContent("payments, users, ci-cd");
    expect(wizardSummary).toHaveTextContent("2 listed");

    const cardOverview = screen.getByTestId("charter-card-overview");
    expect(cardOverview).toHaveTextContent("Domain cards");
    expect(cardOverview).toHaveTextContent("1");
    expect(cardOverview).toHaveTextContent("payments domain");
    expect(cardOverview).toHaveTextContent("platform team");

    const promptStatus = screen.getByTestId("charter-prompt-bundle-status");
    expect(promptStatus).toHaveTextContent("Baseline prompt bundle");
    expect(promptStatus).toHaveTextContent("Prompt packs");
    expect(promptStatus).toHaveTextContent("Live consumed");
    expect(promptStatus).toHaveTextContent("Reference-only");
    expect(promptStatus).toHaveTextContent("proposal/beta-refresh");

    expect(screen.getByTestId("charter-artifact-editor")).toHaveTextContent("Baseline: Editors");
  });

  it("shows Charter baseline recovery for prompt bundle diagnostics", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        baselineBundleWarnings: [
          {
            level: "warning",
            code: "baseline.prompt_pack.missing_policy",
            message: "skills/prompt-packs/qa.md is missing the artifact-only policy reminder.",
            suggestion: "Add the artifact-only policy reminder before live Q&A runs.",
            path: "skills/prompt-packs/qa.md",
          },
        ],
      }),
    );

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-charter"));

    const recovery = await screen.findByTestId("charter-baseline-recovery");
    expect(recovery).toHaveTextContent("Charter baseline recovery");
    expect(recovery).toHaveTextContent("skills/prompt-packs/qa.md");
    expect(recovery).toHaveTextContent("prompt-pack");
    expect(recovery).toHaveTextContent("live consumed");
    expect(recovery).toHaveTextContent("baseline.prompt_pack.missing_policy");
    expect(recovery).toHaveTextContent("Add the artifact-only policy reminder before live Q&A runs.");
    expect(recovery).toHaveTextContent("Save selected baseline artifact");
    expect(screen.getByTestId("charter-prompt-bundle-status")).toHaveTextContent("1 warnings");
  });

  it("supports keyboard navigation across the V2 stage rail", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    const sourceStage = screen.getByTestId("stage-source");
    sourceStage.focus();
    fireEvent.keyDown(sourceStage, { key: "End" });

    expect(await screen.findByTestId("publish-panel")).toBeInTheDocument();
    expect(screen.getByTestId("stage-publish")).toHaveAttribute("aria-current", "step");

    fireEvent.keyDown(screen.getByTestId("stage-publish"), { key: "Home" });
    expect(await screen.findByTestId("workspace-panel")).toBeInTheDocument();
    expect(screen.getByTestId("stage-source")).toHaveAttribute("aria-current", "step");

    fireEvent.keyDown(screen.getByTestId("stage-source"), { key: "ArrowRight" });
    expect(await screen.findByTestId("readiness-panel")).toBeInTheDocument();
    expect(screen.getByTestId("stage-readiness")).toHaveAttribute("aria-current", "step");
  });

  it("opens Review by default when a completed run already has artifacts", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [
              { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
              { path: "reports/diagrams/c4-context.mmd", kind: "diagram", label: "C4 context" },
            ],
          },
        },
      }),
    );

    await renderConsoleApp();

    expect(await screen.findByTestId("review-panel")).toBeInTheDocument();
    expect(screen.getByTestId("stage-review")).toHaveClass("is-selected");
    expect(screen.queryByTestId("workspace-panel")).not.toBeInTheDocument();
    expect(screen.getByTestId("right-inspector")).toHaveTextContent(/review/i);
  });

  it("offers last successful artifacts when Review opens on a failed partial run", async () => {
    const failedRunID = "run-refresh-failed";
    const successfulRunID = "run-init-success";
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID: failedRunID,
        runStarted: true,
        runList: [
          {
            run_id: failedRunID,
            pipeline: "refresh",
            status: "failed",
            started_at: "2026-04-03T12:10:00Z",
            finished_at: "2026-04-03T12:12:00Z",
            warnings: [],
            error_code: "runner_unavailable",
            error: "runtime unavailable",
          },
          {
            run_id: successfulRunID,
            pipeline: "init",
            status: "succeeded",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: "2026-04-03T12:08:00Z",
            warnings: [],
            error_code: null,
            error: null,
          },
        ],
        runStatus: {
          [failedRunID]: {
            run_id: failedRunID,
            pipeline: "refresh",
            status: "failed",
            started_at: "2026-04-03T12:10:00Z",
            finished_at: "2026-04-03T12:12:00Z",
            warnings: [],
            error_code: "runner_unavailable",
            error: "runtime unavailable",
          },
          [successfulRunID]: {
            run_id: successfulRunID,
            pipeline: "init",
            status: "succeeded",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: "2026-04-03T12:08:00Z",
            warnings: [],
            error_code: null,
            error: null,
          },
        },
        runArtifacts: {
          [failedRunID]: {
            run_id: failedRunID,
            artifacts: [{ path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" }],
          },
          [successfulRunID]: {
            run_id: successfulRunID,
            artifacts: [
              { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
              { path: "reports/diagrams/c4-context.mmd", kind: "diagram", label: "C4 context" },
            ],
          },
        },
        runReviewSummary: {
          [failedRunID]: {
            run_id: failedRunID,
            pipeline: "refresh",
            status: "failed",
            started_at: "2026-04-03T12:10:00Z",
            finished_at: "2026-04-03T12:12:00Z",
            current_step: "refresh.step2.asis_docs",
            warnings: [],
            error_code: "runner_unavailable",
            error: "runtime unavailable",
            steps: [],
          },
          [successfulRunID]: {
            run_id: successfulRunID,
            pipeline: "init",
            status: "succeeded",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: "2026-04-03T12:08:00Z",
            current_step: "init.step4.proposals",
            warnings: [],
            error_code: null,
            error: null,
            steps: [],
          },
        },
        artifactText: {
          "reports/coverage/summary.md": "# Coverage Summary\n\nAnalysis incomplete.\n",
          "reports/as-is/overview.md": "# Complete overview\n\nEvidence-backed architecture output.\n",
          "reports/diagrams/c4-context.mmd": "flowchart LR\n  A[Workspace System] --> B[Backend]\n",
        },
      }),
    );

    await renderConsoleApp();

    const recovery = await screen.findByTestId("review-run-recovery");
    expect(recovery).toHaveTextContent(successfulRunID);
    expect(screen.getByTestId("review-artifact-explorer")).not.toHaveTextContent("reports/diagrams");

    fireEvent.click(within(recovery).getByRole("button", { name: /open last successful artifacts/i }));

    await waitFor(() => expect(screen.queryByTestId("review-run-recovery")).not.toBeInTheDocument());
    const explorer = screen.getByTestId("review-artifact-explorer");
    expect(explorer).toHaveTextContent("reports/diagrams");
    await waitFor(() => expect(screen.getByTestId("run-artifact-content")).toHaveTextContent("# Complete overview"));
  });

  it("renders Review V2 evidence workbench and domain-map partial state", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [
              { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
              { path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" },
              { path: "reports/findings/findings.md", kind: "report", label: "Findings" },
              { path: "reports/diagrams/c4-context.mmd", kind: "diagram", label: "C4 context" },
              { path: "reports/agent-outputs/domains/payments-service.md", kind: "domain-report", label: "Payments domain" },
              { path: "model/entities/repo.payments-service.yaml", kind: "model-entity", label: "Payments repo" },
              { path: "model/entities/svc.payments.yaml", kind: "model-entity", label: "Payments Service" },
              { path: "model/entities/team.platform.yaml", kind: "model-entity", label: "Platform Team" },
              { path: "model/edges/edge.svc.payments.calls.svc.users.yaml", kind: "model-edge", label: "Payments calls Users" },
              { path: "proposals/proposal-payments/proposal.md", kind: "proposal", label: "Payments proposal" },
            ],
          },
        },
        artifactText: {
          "reports/as-is/overview.md": "# System overview\nPayments owns checkout.\n",
          "reports/coverage/summary.md": "Coverage: 84%\n",
          "reports/coverage/open-questions.md": "- Clarify owners\n",
          "reports/findings/findings.md": "# Findings\n- Owner gap\n",
          "reports/diagrams/c4-context.mmd": "flowchart LR\n  A --> B\n",
          "model/entities/svc.payments.yaml": "id: svc.payments\ntype: service\nname: Payments Service\n",
        },
      }),
    );

    await renderConsoleApp();

    const explorer = await screen.findByTestId("review-artifact-explorer");
    expect(screen.getByTestId("review-section-jumps")).toHaveTextContent("Preview");
    expect(screen.getByTestId("review-artifact-explorer-toggle")).toHaveTextContent("Secondary browser");
    expect(explorer).not.toHaveAttribute("open");
    fireEvent.click(screen.getByTestId("review-artifact-explorer-toggle"));
    await waitFor(() => expect(explorer).toHaveAttribute("open"));
    expect(explorer).toHaveTextContent("reports/as-is");
    expect(explorer).toHaveTextContent("report");
    expect(explorer).toHaveTextContent("diagram");
    expect(explorer).toHaveTextContent("model");
    expect(explorer).toHaveTextContent("proposal");
    expect(explorer).toHaveTextContent("reports/coverage");
    expect(explorer).toHaveTextContent("reports/diagrams");
    expect((explorer.textContent ?? "").indexOf("reports/as-is")).toBeLessThan((explorer.textContent ?? "").indexOf("proposals"));
    const reviewQueue = screen.getByTestId("review-queue");
    expect(reviewQueue).toHaveTextContent("Review Queue");
    expect(reviewQueue).toHaveTextContent("reports/coverage/summary.md");

    const citationCoverage = screen.getByTestId("review-citation-coverage");
    expect(citationCoverage).toHaveTextContent("Coverage summary");
    expect(citationCoverage).toHaveTextContent("ready");
    expect(citationCoverage).toHaveTextContent("Open questions");
    expect(citationCoverage).toHaveTextContent("Review required");
    expect(screen.getByTestId("review-decision-summary")).toHaveTextContent("not a hard publish blocker");

    await waitFor(() => expect(screen.getByTestId("run-artifact-content")).toHaveTextContent("# System overview"));
    await waitFor(() => expect(within(explorer).getByRole("button", { name: /reports\/as-is\/overview\.md/i })).toHaveClass("is-selected"));

    fireEvent.click(within(reviewQueue).getByRole("button", { name: /review queue item: review coverage summary/i }));
    await waitFor(() => expect(screen.getByTestId("run-artifact-selected-path")).toHaveTextContent("reports/coverage/summary.md"));

    fireEvent.click(within(explorer).getByRole("button", { name: /reports\/as-is\/overview\.md/i }));
    await waitFor(() => expect(screen.getByTestId("run-artifact-content")).toHaveTextContent("# System overview"));

    fireEvent.click(screen.getByTestId("review-view-domain-map-tab"));
    const domainMap = screen.getByTestId("review-domain-map");
    expect(screen.getByTestId("review-domain-map-canvas")).toHaveTextContent("Domain/service map");
    expect(domainMap).toHaveTextContent("Payments domain");
    expect(domainMap).toHaveTextContent("Payments Service");
    expect(domainMap).toHaveTextContent("Platform Team");
    expect(screen.getByTestId("review-domain-map-edge-list")).toHaveTextContent("calls");
    expect(screen.getByTestId("review-domain-map-edge-list")).toHaveTextContent("svc.payments");
    expect(screen.getByTestId("review-domain-map-edge-list")).toHaveTextContent("svc.users");
    expect(screen.getByTestId("review-domain-map-inspector")).toHaveTextContent("coverage summary linked");
    expect(screen.getByTestId("review-domain-map-inspector")).toHaveTextContent("Proposal artifacts ready");
    expect(screen.getAllByTestId("review-domain-map-node").length).toBeGreaterThanOrEqual(4);

    fireEvent.click(within(domainMap).getByRole("button", { name: /open map entity: model\/entities\/svc\.payments\.yaml/i }));
    await waitFor(() => expect(screen.getByTestId("run-artifact-content")).toHaveTextContent("id: svc.payments"));

    fireEvent.click(screen.getByTestId("review-view-evidence-tab"));
    expect(screen.getByTestId("review-evidence-preview")).toHaveTextContent("Evidence preview");
  });

  it("filters Review artifacts by report, diagram, proposal, and runtime groups", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [
              { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
              { path: "reports/diagrams/c4-context.mmd", kind: "diagram", label: "C4 context" },
              { path: "proposals/proposal-payments/proposal.md", kind: "proposal", label: "Payments proposal" },
              { path: "reports/changelog/2026-04-03.md", kind: "changelog", label: "Iteration changelog" },
              { path: "reports/taskruns/run-1/staging/final/final-run-index.json", kind: "taskrun", label: "Final run index" },
            ],
          },
        },
        artifactText: {
          "reports/coverage/open-questions.md": "",
        },
      }),
    );

    await renderConsoleApp();

    const explorer = await screen.findByTestId("review-artifact-explorer");
    fireEvent.click(await screen.findByTestId("review-artifact-explorer-toggle"));
    await waitFor(() => expect(explorer).toHaveAttribute("open"));
    const filters = await screen.findByTestId("review-artifact-filters");
    const artifactPanel = screen.getByTestId("results-artifacts-panel");
    expect(filters).toHaveAttribute("role", "tablist");
    expect(within(filters).getByRole("tab", { name: "All" })).toHaveAttribute("aria-selected", "true");

    fireEvent.click(within(filters).getByRole("tab", { name: "Diagrams" }));
    await waitFor(() => expect(explorer).toHaveAttribute("open"));
    expect(screen.getByTestId("run-diagrams-list")).toBeVisible();
    await waitFor(() => expect(artifactPanel).toHaveTextContent("C4 context"));
    expect(artifactPanel).not.toHaveTextContent("As-is overview");
    expect(artifactPanel).not.toHaveTextContent("Payments proposal");

    fireEvent.click(within(filters).getByRole("tab", { name: "Proposals" }));
    await waitFor(() => expect(artifactPanel).toHaveTextContent("Payments proposal"));
    expect(artifactPanel).toHaveTextContent("Iteration changelog");
    expect(artifactPanel).not.toHaveTextContent("C4 context");

    fireEvent.click(within(filters).getByRole("tab", { name: "Runtime" }));
    await waitFor(() => expect(artifactPanel).toHaveTextContent("Final run index"));
    expect(artifactPanel).not.toHaveTextContent("As-is overview");

    fireEvent.click(within(filters).getByRole("tab", { name: "Reports" }));
    await waitFor(() => expect(artifactPanel).toHaveTextContent("As-is overview"));
    expect(artifactPanel).not.toHaveTextContent("Final run index");
  });

  it("renders an explicit sparse state for Review domain map without model artifacts", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [
              { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
              { path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" },
            ],
          },
        },
        artifactText: {
          "reports/coverage/open-questions.md": "",
        },
      }),
    );

    await renderConsoleApp();

    await screen.findByTestId("review-panel");
    fireEvent.click(screen.getByTestId("review-view-domain-map-tab"));

    expect(screen.getByTestId("review-domain-map-empty")).toHaveTextContent("No derived model artifacts yet.");
    expect(screen.getByTestId("review-domain-map-inspector")).toHaveTextContent("partial");
    expect(screen.getByTestId("review-domain-map-inspector")).toHaveTextContent("Derived model entities are missing");
  });

  it("keeps Review domain-map diagnostic navigation for edges and proposal artifacts", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [
              { path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" },
              { path: "reports/agent-outputs/domains/payments-service.md", kind: "domain-report", label: "Payments domain" },
              { path: "model/entities/svc.payments.yaml", kind: "model-entity", label: "Payments Service" },
              { path: "model/entities/svc.users.yaml", kind: "model-entity", label: "Users Service" },
              { path: "model/edges/edge.svc.payments.calls.svc.users.yaml", kind: "model-edge", label: "Payments calls Users" },
              { path: "proposals/proposal-payments/proposal.md", kind: "proposal", label: "Payments proposal" },
            ],
          },
        },
        artifactText: {
          "reports/coverage/open-questions.md": "",
          "model/edges/edge.svc.payments.calls.svc.users.yaml": "from: svc.payments\ntype: calls\nto: svc.users\n",
          "proposals/proposal-payments/proposal.md": "# Proposal\nReview payments/users integration ownership.\n",
        },
      }),
    );

    await renderConsoleApp();

    await screen.findByTestId("review-panel");
    fireEvent.click(screen.getByTestId("review-view-domain-map-tab"));

    const edgeList = screen.getByTestId("review-domain-map-edge-list");
    const inspector = screen.getByTestId("review-domain-map-inspector");
    expect(edgeList).toHaveTextContent("svc.payments");
    expect(edgeList).toHaveTextContent("svc.users");
    expect(inspector).toHaveTextContent("Proposal artifacts ready");

    fireEvent.click(within(edgeList).getByRole("button", { name: /model\/edges\/edge\.svc\.payments\.calls\.svc\.users\.yaml/i }));
    await waitFor(() => expect(screen.getByTestId("run-artifact-content")).toHaveTextContent("type: calls"));

    fireEvent.click(screen.getByTestId("review-view-domain-map-tab"));
    const refreshedInspector = screen.getByTestId("review-domain-map-inspector");
    fireEvent.click(within(refreshedInspector).getByRole("button", { name: /proposals\/proposal-payments\/proposal\.md/i }));
    await waitFor(() => expect(screen.getByTestId("run-artifact-content")).toHaveTextContent("# Proposal"));
  });

  it("renders Proposals V2 review room with preview tabs and publication path", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [
              { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
              { path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" },
              { path: "reports/findings/findings.md", kind: "report", label: "Findings" },
              { path: "proposals/proposal-payments/proposal.md", kind: "proposal", label: "Payments proposal" },
              { path: "proposals/proposal-payments/ADR.md", kind: "proposal", label: "Payments ADR" },
              { path: "proposals/proposal-payments/RFC.md", kind: "proposal", label: "Payments RFC" },
              { path: "proposals/proposal-payments/migration-checklist.md", kind: "proposal", label: "Migration checklist" },
              { path: "reports/changelog/2026-05-28-payments.md", kind: "changelog", label: "Payments changelog" },
            ],
          },
        },
        artifactText: {
          "reports/coverage/open-questions.md": "",
          "proposals/proposal-payments/proposal.md": "# Proposal\nTighten payment ownership review.\n",
        },
      }),
    );

    await renderConsoleApp();

    await screen.findByTestId("review-panel");
    fireEvent.click(screen.getByTestId("stage-proposals"));

    expect(await screen.findByTestId("proposals-review-room")).toBeInTheDocument();
    expect(screen.queryByTestId("proposal-package-recovery")).not.toBeInTheDocument();
    const artifactList = screen.getByTestId("proposals-artifact-list");
    expect(artifactList).toHaveTextContent("proposals/proposal-payments");
    expect(artifactList).toHaveTextContent("reports/changelog");
    expect(screen.getByTestId("proposal-quality-panel")).toHaveTextContent("Proposal docs");
    expect(screen.getByTestId("proposal-quality-panel")).toHaveTextContent("4");
    expect(screen.getByTestId("proposal-publication-path")).toHaveTextContent("proposal/beta-refresh");
    await waitFor(() => expect(screen.getByTestId("proposal-preview-panel")).toHaveTextContent("# Proposal"));

    fireEvent.click(within(artifactList).getByRole("button", { name: /open proposal artifact: proposals\/proposal-payments\/proposal\.md/i }));
    await waitFor(() => expect(screen.getByTestId("proposal-preview-panel")).toHaveTextContent("# Proposal"));

    const tabs = screen.getByTestId("proposal-preview-tabs");
    fireEvent.click(within(tabs).getByRole("tab", { name: "Evidence" }));
    expect(screen.getByTestId("proposal-preview-panel")).toHaveTextContent("reports/findings");

    fireEvent.click(within(tabs).getByRole("tab", { name: "Changelog" }));
    expect(screen.getByTestId("proposal-preview-panel")).toHaveTextContent("Payments changelog");

    fireEvent.click(within(tabs).getByRole("tab", { name: "Diff" }));
    expect(screen.getByTestId("proposal-preview-panel")).toHaveTextContent("Workspace Git diff loaded.");
    expect(screen.getByTestId("git-diff-view")).toHaveTextContent("reports/coverage/summary.md");

    fireEvent.click(screen.getByRole("button", { name: "Review in Publish" }));
    expect(await screen.findByTestId("publish-panel")).toBeInTheDocument();
  });

  it("opens a changelog-only proposal run without an empty Proposals preview", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [
              { path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" },
              { path: "reports/changelog/2026-05-31-run-1.md", kind: "changelog", label: "Run changelog" },
            ],
          },
        },
        artifactText: {
          "reports/coverage/open-questions.md": "",
          "reports/changelog/2026-05-31-run-1.md": "# Run changelog\n- Proposal package pending.\n",
        },
      }),
    );

    await renderConsoleApp();

    await screen.findByTestId("review-panel");
    fireEvent.click(screen.getByTestId("stage-proposals"));

    const artifactList = await screen.findByTestId("proposals-artifact-list");
    const recovery = screen.getByTestId("proposal-package-recovery");
    expect(recovery).toHaveTextContent("Proposal package recovery");
    expect(recovery).toHaveTextContent("No proposal package artifacts are available.");
    expect(recovery).toHaveTextContent("Proposal docs");
    expect(recovery).toHaveTextContent("0");
    expect(recovery).toHaveTextContent("Retry or rerun Analysis step4.proposals");
    expect(recovery).toHaveTextContent("Keep Publish as review-only");
    await waitFor(() => expect(screen.getByTestId("proposal-preview-panel")).toHaveTextContent("# Run changelog"));
    expect(within(artifactList).getByRole("button", { name: /reports\/changelog\/2026-05-31-run-1\.md/i })).toHaveClass("is-selected");
  });

  it("restores the Review evidence artifact when returning from Proposals", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [
              { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
              { path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" },
              { path: "proposals/proposal-payments/proposal.md", kind: "proposal", label: "Payments proposal" },
            ],
          },
        },
        artifactText: {
          "reports/as-is/overview.md": "# System overview\nReviewable evidence body.\n",
          "reports/coverage/open-questions.md": "",
          "reports/coverage/summary.md": "# Coverage\n",
          "proposals/proposal-payments/proposal.md": "# Proposal\nProposal body.\n",
        },
      }),
    );

    await renderConsoleApp();

    await screen.findByTestId("review-panel");
    await waitFor(() => expect(screen.getByTestId("run-artifact-selected-path")).toHaveTextContent("reports/as-is/overview.md"));

    fireEvent.click(screen.getByTestId("stage-proposals"));
    await waitFor(() => expect(screen.getByTestId("proposal-preview-panel")).toHaveTextContent("# Proposal"));

    fireEvent.click(screen.getByTestId("stage-review"));
    await waitFor(() => expect(screen.getByTestId("run-artifact-selected-path")).toHaveTextContent("reports/as-is/overview.md"));
    expect(screen.getByTestId("run-artifact-content")).toHaveTextContent("# System overview");
    fireEvent.click(screen.getByTestId("review-artifact-explorer-toggle"));
    expect(within(screen.getByTestId("review-artifact-explorer")).getByRole("button", { name: /reports\/as-is\/overview\.md/i })).toHaveClass("is-selected");
  });

  it("renders the Publish gate with folder summary, preview tabs, commit plan, and Git actions", async () => {
    const fetchMock = createFetchMock({
      runStarted: true,
      runArtifacts: {
        "run-1": {
          run_id: "run-1",
          artifacts: [
            { path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" },
            { path: "model/entities/payments-service.yaml", kind: "model", label: "payments-service" },
            { path: "proposals/adr-001.md", kind: "proposal", label: "ADR 001" },
            { path: "reports/changelog/2026-04-03.md", kind: "changelog", label: "Iteration changelog" },
          ],
        },
      },
      artifactText: {
        "reports/coverage/summary.md": "Coverage ready for publication.\n",
        "reports/coverage/open-questions.md": "- Confirm owner sign-off\n",
        "model/entities/payments-service.yaml": "id: payments-service\n",
        "proposals/adr-001.md": "# ADR 001\n",
        "reports/changelog/2026-04-03.md": "# Iteration changelog\n- Published architecture workspace artifacts.\n",
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    await screen.findByTestId("review-panel");

    fireEvent.click(screen.getByTestId("stage-publish"));

    await waitFor(() => expect(screen.getByTestId("stage-publish")).toHaveAttribute("aria-current", "step"));
    await waitFor(() => expect(screen.getByTestId("publish-panel")).toBeInTheDocument());
    await waitFor(() => expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("reports/coverage"));
    expect(screen.getByTestId("publish-section-jumps")).toHaveTextContent("Diff");
    expect(screen.getByTestId("publish-section-jumps")).toHaveTextContent("Preview");
    expect(screen.getByTestId("publish-section-jumps")).toHaveTextContent("Gate");
    expect(screen.getByTestId("publish-section-jumps")).toHaveTextContent("Commit");
    expect(screen.getByTestId("publish-section-jumps").querySelector('a[href="#publish-gate-panel"]')).toBeInTheDocument();
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("Publication set");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("4 refs");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("review");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("Confirm owner sign-off");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("Git action");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("ready");
    expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("Selected run Git diff");
    expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("model");
    expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("proposals");
    expect(screen.getByTestId("publish-gate-panel")).toHaveTextContent("Confirm owner sign-off");
    expect(screen.getByTestId("publish-hard-blockers")).toHaveTextContent("No hard blockers");
    expect(screen.getByTestId("publish-open-questions")).toHaveTextContent("Confirm owner sign-off");
    expect(screen.getByTestId("publish-ready-checks")).toHaveTextContent("Artifacts");
    expect(screen.getByTestId("publish-commit-plan")).toHaveTextContent("proposal/beta-refresh");
    expect(screen.getByTestId("publish-commit-selected-btn")).toBeInTheDocument();
    const proposalBranchButton = screen.getByTestId("git-proposal-branch-btn").closest("button") as HTMLButtonElement;
    expect(proposalBranchButton).not.toBeDisabled();

    fireEvent.click(screen.getByRole("button", { name: /Coverage summary.*reports\/coverage\/summary\.md/i }));
    await waitFor(() => expect(screen.getByTestId("publish-panel")).toHaveTextContent("Coverage ready for publication."));

    fireEvent.click(screen.getByRole("tab", { name: "Diff" }));
    expect(screen.getByTestId("publish-panel")).toHaveTextContent("Selected run Git diff");
    expect(screen.getByTestId("git-diff-view")).toHaveTextContent("Workspace Git diff loaded.");
    expect(screen.getByTestId("git-diff-hunks")).toHaveTextContent("Workspace Git diff is reviewable.");

    fireEvent.click(screen.getByRole("button", { name: "Load full workspace diff" }));
    await waitFor(() => expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("Full workspace Git diff"));

    expect(screen.getByTestId("publish-preview-tabs")).toHaveTextContent("Changelog");
    expect(screen.getByTestId("publish-preview-tabs")).not.toHaveTextContent("Checklist");
    fireEvent.click(screen.getByRole("tab", { name: "Changelog" }));
    expect(screen.getByTestId("publish-panel")).toHaveTextContent("Iteration changelog");

    const commitInput = screen.getByLabelText("Commit message");
    fireEvent.change(commitInput, { target: { value: "docs: publish architecture workspace" } });
    fireEvent.click(screen.getByTestId("publish-commit-selected-btn"));
    await waitFor(() => expect(screen.getByTestId("publish-commit-plan")).toHaveTextContent("committed: docs: publish architecture workspace"));

    fireEvent.click(proposalBranchButton);
    await waitFor(() => expect(screen.getByTestId("publish-commit-plan")).toHaveTextContent("checked out proposal/beta-refresh"));

    expect(fetchMock.mock.calls.filter((call) => call[0] === "/api/git/commit")).toHaveLength(1);
    expect(fetchMock.mock.calls.filter((call) => call[0] === "/api/git/proposal-branch")).toHaveLength(1);
    expect(fetchMock.mock.calls.some((call) => String(call[0]).startsWith("/api/git/diff") && !String(call[0]).includes("run_id="))).toBe(true);
  });

  it("filters Publish artifact refs while preserving the selected artifact preview", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [
              { path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" },
              { path: "reports/diagrams/c4-context.mmd", kind: "diagram", label: "C4 context" },
              { path: "proposals/adr-001.md", kind: "proposal", label: "ADR 001" },
              { path: "reports/changelog/2026-04-03.md", kind: "changelog", label: "Iteration changelog" },
              { path: "reports/taskruns/run-1/staging/shards/payments/runtime-execution.json", kind: "taskrun", label: "Runtime execution" },
            ],
          },
        },
        artifactText: {
          "reports/coverage/summary.md": "Coverage ready for publication.\n",
          "reports/coverage/open-questions.md": "",
          "reports/diagrams/c4-context.mmd": "flowchart LR\n  A --> B\n",
          "proposals/adr-001.md": "# ADR 001\n",
          "reports/changelog/2026-04-03.md": "# Iteration changelog\n",
          "reports/taskruns/run-1/staging/shards/payments/runtime-execution.json": "{\"ok\":true}\n",
        },
      }),
    );

    await renderConsoleApp();
    await screen.findByTestId("review-panel");

    fireEvent.click(screen.getByTestId("stage-publish"));
    await screen.findByTestId("publish-panel");
    await waitFor(() => expect(screen.getByTestId("stage-publish")).toHaveAttribute("aria-current", "step"));
    await waitFor(() => expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("reports/coverage"));

    const filters = await screen.findByTestId("publish-artifact-filters");
    expect(filters).toHaveAttribute("role", "tablist");
    expect(within(filters).getByRole("tab", { name: "All" })).toHaveAttribute("aria-selected", "true");
    const publishArtifactList = () => screen.getByRole("list", { name: "publish artifact preview list" });

    fireEvent.click(within(publishArtifactList()).getByRole("button", { name: /ADR 001.*proposals\/adr-001\.md/i }));
    await waitFor(() => expect(screen.getByTestId("publish-selected-preview-content")).toHaveTextContent("# ADR 001"));

    fireEvent.click(within(filters).getByRole("tab", { name: "Diagrams" }));
    await waitFor(() => expect(publishArtifactList()).toHaveTextContent("reports/diagrams/c4-context.mmd"));
    expect(publishArtifactList()).not.toHaveTextContent("proposals/adr-001.md");
    expect(screen.getByTestId("publish-selected-preview-content")).toHaveTextContent("# ADR 001");

    fireEvent.click(within(filters).getByRole("tab", { name: "Taskruns" }));
    await waitFor(() => expect(publishArtifactList()).toHaveTextContent("reports/taskruns/run-1/staging/shards/payments/runtime-execution.json"));
    expect(publishArtifactList()).not.toHaveTextContent("reports/coverage/summary.md");

    fireEvent.click(within(filters).getByRole("tab", { name: "Changed" }));
    await waitFor(() => expect(publishArtifactList()).toHaveTextContent("reports/coverage/summary.md"));
    expect(publishArtifactList()).toHaveTextContent("proposals/adr-001.md");
    expect(publishArtifactList()).not.toHaveTextContent("reports/taskruns/run-1/staging/shards/payments/runtime-execution.json");
    expect(screen.getByRole("list", { name: "changed workspace files" })).toHaveTextContent("reports/coverage/summary.md");
  });

  it("keeps raw proposal and changelog artifacts when a final run index is available", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [
              { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
              { path: "proposals/proposal-baseline/proposal.md", kind: "proposal", label: "Baseline proposal" },
              { path: "reports/changelog/2026-05-31-run-1.md", kind: "changelog", label: "Run changelog" },
              { path: "reports/taskruns/run-1/staging/final/final-run-index.json", kind: "taskrun", label: "Final Run Index" },
            ],
          },
        },
        artifactText: {
          "reports/as-is/overview.md": "# As-is overview\n",
          "proposals/proposal-baseline/proposal.md": "# Proposal\n",
          "reports/changelog/2026-05-31-run-1.md": "# Run changelog\n- Proposal package compiled.\n",
          "reports/coverage/open-questions.md": "",
          "reports/taskruns/run-1/staging/final/final-run-index.json": JSON.stringify({
            version: 1,
            run_id: "run-1",
            pipeline: "init",
            generated_at: "2026-05-31T12:00:00Z",
            canonical_documents: [
              {
                id: "doc.reports-as-is-overview-md",
                kind: "report",
                title: "As-is overview",
                canonical_path: "reports/as-is/overview.md",
                staged_path: "reports/taskruns/run-1/staging/final/reports/as-is/overview.md",
              },
            ],
          }),
        },
      }),
    );

    await renderConsoleApp();

    await screen.findByTestId("review-panel");
    fireEvent.click(screen.getByTestId("stage-proposals"));
    const proposalList = await screen.findByTestId("proposals-artifact-list");
    expect(proposalList).toHaveTextContent("proposals/proposal-baseline");
    expect(proposalList).toHaveTextContent("reports/changelog");

    fireEvent.click(screen.getByTestId("stage-publish"));
    expect(await screen.findByTestId("publish-panel")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Changelog" }));

    await waitFor(() => {
      expect(screen.getByTestId("publish-panel")).toHaveTextContent("reports/changelog/2026-05-31-run-1.md");
      expect(screen.getByTestId("publish-panel")).toHaveTextContent("Proposal package compiled.");
    });
  });

  it("blocks Publish Git actions until generated artifacts exist", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-publish"));

    expect(await screen.findByTestId("publish-panel")).toBeInTheDocument();
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("0 refs");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("blocked");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("Run analysis before publishing workspace artifacts.");
    expect(screen.getByTestId("publish-gate-panel")).toHaveTextContent("Run analysis before publishing workspace artifacts.");
    expect(screen.getByTestId("blockers-panel")).toHaveTextContent("No publishable artifacts");
    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("No generated workspace artifacts are ready to publish.");
    expect(screen.getByTestId("publish-commit-selected-btn")).toBeDisabled();
    const proposalBranchButton = screen.getByTestId("git-proposal-branch-btn").closest("button") as HTMLButtonElement;
    expect(proposalBranchButton).toBeDisabled();

    fireEvent.click(screen.getByTestId("inspector-primary-action"));
    fireEvent.click(proposalBranchButton);
    expect(fetchMock.mock.calls.filter((call) => call[0] === "/api/git/commit")).toHaveLength(0);
    expect(fetchMock.mock.calls.filter((call) => call[0] === "/api/git/proposal-branch")).toHaveLength(0);
  });

  it("blocks Publish Git actions when generated artifacts still have runtime blockers", async () => {
    const fetchMock = createFetchMock({
      runStarted: true,
      runStatus: {
        "run-1": {
          run_id: "run-1",
          pipeline: "init",
          status: "failed",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: "2026-04-03T12:00:02Z",
          warnings: [],
          error_code: "runtime_contract_failed",
          error: "artifact validation failed",
        },
      },
      runArtifacts: {
        "run-1": {
          run_id: "run-1",
          artifacts: [{ path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" }],
        },
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    await screen.findByTestId("review-panel");

    fireEvent.click(screen.getByTestId("stage-publish"));

    expect(await screen.findByTestId("publish-panel")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("reports/coverage"));
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("blocked");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("runtime_contract_failed");
    expect(screen.getByTestId("publish-gate-panel")).toHaveTextContent("runtime_contract_failed");
    expect(screen.getByTestId("publish-commit-selected-btn")).toBeDisabled();
    const proposalBranchButton = screen.getByTestId("git-proposal-branch-btn").closest("button") as HTMLButtonElement;
    expect(proposalBranchButton).toBeDisabled();

    fireEvent.click(screen.getByTestId("publish-commit-selected-btn"));
    fireEvent.click(proposalBranchButton);
    expect(fetchMock.mock.calls.filter((call) => call[0] === "/api/git/commit")).toHaveLength(0);
    expect(fetchMock.mock.calls.filter((call) => call[0] === "/api/git/proposal-branch")).toHaveLength(0);
  });

  it("renders diagram artifacts without sending loading placeholder text to Mermaid", async () => {
    const mermaid = await import("mermaid");
    vi.mocked(mermaid.default.render).mockClear();
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runStarted: true,
        runArtifacts: {
          "run-1": {
            run_id: "run-1",
            artifacts: [{ path: "reports/diagrams/c4-context.mmd", kind: "diagram", label: "C4 context" }],
          },
        },
        artifactText: {
          "reports/coverage/open-questions.md": "",
          "reports/diagrams/c4-context.mmd": "flowchart LR\n  A --> B\n",
        },
      }),
    );

    await renderConsoleApp();

    expect(await screen.findByTestId("review-panel")).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole("button", { name: /reports\/diagrams\/c4-context\.mmd/i })[0]);

    await waitFor(() => expect(mermaid.default.render).toHaveBeenCalledWith(expect.stringMatching(/^diagram-/), "flowchart LR\n  A --> B"));
    expect(mermaid.default.render).not.toHaveBeenCalledWith(expect.any(String), "Loading...");
  });

  it("starts agent-backed architecture Q&A and renders answer, citations, unresolved, and confidence", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-ask"));
    fireEvent.change(await screen.findByTestId("qa-question-input"), { target: { value: "Who owns payments?" } });
    fireEvent.click(screen.getByTestId("qa-ask-btn"));

    expect(await screen.findByTestId("qa-answer")).toHaveTextContent("payments-service is owned by Platform Architecture.");
    expect(screen.getByTestId("qa-run-history")).toHaveTextContent("Who owns payments?");
    expect(screen.getByTestId("qa-answer-panel")).toHaveTextContent("Confidence: 82%");
    expect(screen.getByTestId("qa-readonly-safety-panel")).toHaveTextContent("no canonical writes");
    expect(screen.getByTestId("qa-citations-panel")).toHaveTextContent("ownership evidence");
    expect(screen.getByRole("button", { name: /reports\/as-is\/overview\.md/i })).toBeInTheDocument();
    expect(screen.getAllByText(/Unresolved: confirm escalation owner/).length).toBeGreaterThan(0);
    expect(screen.getByText("Confidence: 82%")).toBeInTheDocument();
    expect(screen.getByTestId("qa-run-status")).toHaveTextContent("Runtime provider: fake");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/qa/runs",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ question: "Who owns payments?" }),
      }),
    );
    expect(fetchMock).toHaveBeenCalledWith("/api/qa/runs/qa-run-1", undefined);
  });

  it("renders Q&A failure recovery and retries the original question", async () => {
    const fetchMock = createFetchMock({
      qaRunID: "qa-failed",
      qaResponse: {
        status: "failed",
        question: "Who owns payments?",
        current_step: "qa.ask",
        error_code: "runtime_contract_failed",
        error: "qa-answer.json failed validation",
        warnings: ["answer artifact missing citations"],
        answer: null,
        citations: null,
        unresolved: null,
        confidence: null,
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-ask"));
    fireEvent.change(await screen.findByTestId("qa-question-input"), { target: { value: "Who owns payments?" } });
    fireEvent.click(screen.getByTestId("qa-ask-btn"));

    const recovery = await screen.findByTestId("qa-failure-recovery");
    expect(recovery).toHaveTextContent("Recovery path");
    expect(recovery).toHaveTextContent("runtime_contract_failed");
    expect(recovery).toHaveTextContent("qa.ask");
    expect(recovery).toHaveTextContent("reports/taskruns/qa-failed/qa/");
    expect(recovery).toHaveTextContent("answer artifact missing citations");
    expect(screen.getByTestId("qa-answer")).toHaveTextContent("No answer returned yet");

    fireEvent.click(screen.getByTestId("qa-retry-run-btn"));

    await waitFor(() => {
      const starts = fetchMock.mock.calls.filter((call) => call[0] === "/api/qa/runs");
      expect(starts).toHaveLength(2);
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/qa/runs",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ question: "Who owns payments?" }),
      }),
    );
  });

  it("explains canceled Q&A runs without presenting them as answer validation failures", async () => {
    const fetchMock = createFetchMock({
      qaRunID: "qa-canceled",
      qaResponse: {
        status: "failed",
        question: "Who owns payments?",
        current_step: "qa.ask",
        error_code: "run_canceled",
        error: "canceled by request",
        warnings: [],
        answer: null,
        citations: null,
        unresolved: null,
        confidence: null,
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-ask"));
    fireEvent.change(await screen.findByTestId("qa-question-input"), { target: { value: "Who owns payments?" } });
    fireEvent.click(screen.getByTestId("qa-ask-btn"));

    const recovery = await screen.findByTestId("qa-failure-recovery");
    expect(screen.getByTestId("qa-run-status")).toHaveTextContent("status: canceled");
    expect(recovery).toHaveTextContent("Canceled answer run");
    expect(recovery).toHaveTextContent("run_canceled");
    expect(recovery).toHaveTextContent("Stopped step");
    expect(recovery).toHaveTextContent("qa.ask");
    expect(recovery).toHaveTextContent("The answer run stopped by request");
    expect(recovery).toHaveTextContent("the canceled attempt and QA audit artifacts stay in history");
    expect(screen.getByTestId("qa-retry-run-btn")).toHaveTextContent("Ask again");
    expect(recovery).not.toHaveTextContent("The answer artifact did not pass validation");
  });

  it("routes the inspector Ask primary action to the visible Q&A submit flow", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-ask"));
    fireEvent.change(await screen.findByTestId("qa-question-input"), { target: { value: "Who owns payments?" } });
    fireEvent.click(screen.getByTestId("inspector-primary-action"));

    expect(await screen.findByTestId("qa-answer")).toHaveTextContent("payments-service is owned by Platform Architecture.");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/qa/runs",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ question: "Who owns payments?" }),
      }),
    );
  });

  it("renders the Ask workbench with history, selected answer, audit safety, and citation drilldown", async () => {
    const historyRun = {
      run_id: "qa-history-1",
      pipeline: "qa",
      status: "succeeded",
      started_at: "2026-04-03T12:00:03Z",
      finished_at: "2026-04-03T12:00:04Z",
      question: "Which service owns checkout?",
      current_step: "qa.ask",
      runtime_provider: "claude-code",
      provider: "fake",
      answer: "checkout-service is owned by Payments Platform.",
      citations: [{ path: "model/entities/checkout-service.yaml", reason: "entity owner record" }],
      unresolved: ["confirm production support rotation"],
      confidence: 0.91,
      generated_at: "2026-04-03T12:00:04Z",
    };
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        qaRuns: [historyRun],
        qaRunResponses: { "qa-history-1": historyRun },
      }),
    );

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-ask"));

    expect(await screen.findByTestId("qa-run-history")).toHaveTextContent("Which service owns checkout?");
    expect(await screen.findByTestId("qa-answer-panel")).toHaveTextContent("checkout-service is owned by Payments Platform.");
    expect(screen.getByTestId("qa-answer-panel")).toHaveTextContent("Confidence: 91%");
    expect(screen.getByTestId("qa-answer-panel")).toHaveTextContent("Related entities and edges");
    expect(screen.getByTestId("qa-readonly-safety-panel")).toHaveTextContent("reports/taskruns/<run_id>/qa/");
    expect(screen.getByRole("button", { name: /context-pack\.json/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /qa-answer\.json/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /runtime-execution\.json/i })).toBeInTheDocument();
    expect(screen.getByTestId("qa-citations-panel")).toHaveTextContent("entity owner record");
    expect(screen.getByRole("button", { name: /model\/entities\/checkout-service\.yaml/i })).toBeInTheDocument();
    expect(screen.getAllByText(/Unresolved: confirm production support rotation/).length).toBeGreaterThan(0);
  });

  it("renders read-only Q&A when the API returns nullable evidence arrays", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        qaResponse: {
          answer: "Not enough indexed workspace evidence to answer confidently yet.",
          citations: null,
          unresolved: null,
          confidence: null,
        },
      }),
    );

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-ask"));
    fireEvent.change(await screen.findByTestId("qa-question-input"), { target: { value: "What is known?" } });
    fireEvent.click(await screen.findByTestId("qa-ask-btn"));

    expect(await screen.findByTestId("qa-answer")).toHaveTextContent("Not enough indexed workspace evidence");
    expect(screen.getByText("No citations returned.")).toBeInTheDocument();
    expect(screen.getByText("Confidence: 0%")).toBeInTheDocument();
  });

  it("guides first-run setup through validate, doctor, and run", async () => {
    const fetchMock = createFetchMock({ runID: "run-first" });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    expect(screen.getByTestId("stage-source")).toHaveClass("is-selected");
    expect(screen.getByDisplayValue("https://github.com/org/my-service.git")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("workspace-save-btn"));

    await screen.findByTestId("workspace-validate-result");
    fireEvent.click(screen.getByTestId("stage-readiness"));
    expect(screen.getByTestId("setup-run-first-btn")).toBeDisabled();
    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("Check local readiness");

    fireEvent.click(screen.getByTestId("setup-doctor-btn"));
    await screen.findByTestId("setup-doctor-result");
    expect(screen.getByText("Local readiness passed.")).toBeInTheDocument();
    expect(screen.getByTestId("setup-run-first-btn")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("setup-run-first-btn"));
    await screen.findByTestId("analysis-run-progress");
    expect(screen.getByTestId("stage-analysis")).toHaveClass("is-selected");
    expect(fetchMock).toHaveBeenCalledWith("/api/pipeline/init", expect.anything());
  });

  it("supports local-folder source mode in first-run setup", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    fireEvent.change(screen.getByLabelText("Repo source type"), { target: { value: "path" } });
    fireEvent.change(await screen.findByLabelText("Local checkout path"), { target: { value: "/tmp/my-service" } });
    fireEvent.click(screen.getByText("Analysis scope"));
    fireEvent.change(screen.getByLabelText("Include globs"), { target: { value: "services/**\ncmd/**" } });
    fireEvent.change(screen.getByLabelText("Exclude globs"), { target: { value: "**/vendor/**" } });
    fireEvent.click(screen.getByTestId("workspace-save-btn"));
    await screen.findByTestId("workspace-validate-result");
    fireEvent.click(screen.getByText("Advanced workspace.yaml editor"));

    const manifest = screen.getByDisplayValue((content) => content.includes('path: "/tmp/my-service"'));
    expect(manifest).toBeInTheDocument();
    expect((manifest as HTMLTextAreaElement).value).not.toContain("git_url:");
    const putCall = fetchMock.mock.calls.find(([url, init]) => url === "/api/workspace/manifest" && (init as RequestInit | undefined)?.method === "PUT");
    const savedManifest = JSON.parse(String((putCall?.[1] as RequestInit | undefined)?.body ?? "{}")) as { content?: string };
    expect(savedManifest.content).toContain('path: "/tmp/my-service"');
    expect(savedManifest.content).toContain("analysis:");
    expect(savedManifest.content).toContain("include:");
    expect(savedManifest.content).toContain('        - "services/**"');
    expect(savedManifest.content).toContain('        - "cmd/**"');
    expect(savedManifest.content).toContain("exclude:");
    expect(savedManifest.content).toContain('        - "**/vendor/**"');
  });

  it("hydrates guided first-run form from loaded workspace manifest", async () => {
    const goStyleManifest =
      'version: 1\nrepos:\n    - name: "loaded-service"\n      path: "/tmp/loaded-service"\n      ref: "main"\n      analysis:\n        include:\n          - "services/**"\n        exclude:\n          - "**/vendor/**"\ndocs:\n    imports_path: "./docs/imported"\n';
    const fetchMock = createFetchMock({
      manifestContent: goStyleManifest,
    });
    vi.stubGlobal(
      "fetch",
      fetchMock,
    );

    await renderConsoleApp();

    expect(await screen.findByDisplayValue("loaded-service")).toBeInTheDocument();
    expect(screen.getByLabelText("Repo source type")).toHaveValue("path");
    expect(screen.getByDisplayValue("/tmp/loaded-service")).toBeInTheDocument();
    expect(screen.getByDisplayValue("main")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Analysis scope"));
    expect(screen.getByDisplayValue("services/**")).toBeInTheDocument();
    expect(screen.getByDisplayValue("**/vendor/**")).toBeInTheDocument();
    expect(screen.getByDisplayValue("./docs/imported")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("workspace-save-btn"));
    await screen.findByTestId("workspace-validate-result");

    const putCall = fetchMock.mock.calls.find(([url, init]) => url === "/api/workspace/manifest" && (init as RequestInit | undefined)?.method === "PUT");
    const savedManifest = JSON.parse(String((putCall?.[1] as RequestInit | undefined)?.body ?? "{}")) as { content?: string };
    expect(savedManifest.content).toContain('name: "loaded-service"');
    expect(savedManifest.content).toContain('path: "/tmp/loaded-service"');
    expect(savedManifest.content).toContain('imports_path: "./docs/imported"');
    expect(savedManifest.content).not.toContain("my-service");
    expect(savedManifest.content).not.toContain("https://github.com/org/my-service.git");
  });

  it("shows validation suggestions as next actions", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        validateStatus: 400,
        validateResponse: {
          ok: false,
          workspace: "/tmp/workspace",
          warnings: [],
          errors: [
            {
              level: "error",
              code: "workspace.repo.git_url.fetch_failed",
              message: "git cannot clone this repo",
              suggestion: "Check the repository URL and your local git authentication.",
              repo: "my-service",
            },
          ],
          resolved_repos: [],
        },
      }),
    );

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-readiness"));
    fireEvent.click(screen.getByTestId("workspace-validate-btn"));

    await screen.findByTestId("workspace-validate-result");
    expect(screen.getByText(/Next: Check the repository URL and your local git authentication./)).toBeInTheDocument();
    expect(screen.getByTestId("setup-run-first-btn")).toBeDisabled();
  });

  it("shows Source validation recovery above raw diagnostics", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        validateStatus: 400,
        validateResponse: {
          ok: false,
          workspace: "/tmp/workspace",
          warnings: [],
          errors: [
            {
              level: "error",
              code: "workspace.repo.git_url.fetch_failed",
              message: "git cannot clone this repo",
              suggestion: "Check the repository URL and your local git authentication.",
              repo: "my-service",
            },
          ],
          resolved_repos: [],
        },
      }),
    );

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("workspace-save-btn"));

    const recovery = await screen.findByTestId("source-validation-recovery");
    expect(recovery).toHaveTextContent("Source validation recovery");
    expect(recovery).toHaveTextContent("my-service");
    expect(recovery).toHaveTextContent("workspace.repo.git_url.fetch_failed");
    expect(recovery).toHaveTextContent("Git URL");
    expect(recovery).toHaveTextContent("https://github.com/org/my-service.git");
    expect(recovery).toHaveTextContent("Check the repository URL and your local git authentication.");
    expect(recovery).toHaveTextContent("Save and validate sources");
    expect(screen.getByTestId("source-repo-table")).toHaveTextContent("blocked");
  });

  it("requires revalidation after first-run setup changes", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    fireEvent.click(screen.getByText("Preview workspace.yaml draft"));
    fireEvent.click(screen.getByTestId("workspace-save-btn"));
    await screen.findByTestId("workspace-validate-result");
    fireEvent.click(screen.getByTestId("stage-readiness"));
    expect(screen.getByTestId("setup-run-first-btn")).toBeDisabled();
    fireEvent.click(screen.getByTestId("setup-doctor-btn"));
    await screen.findByTestId("setup-doctor-result");
    expect(screen.getByTestId("setup-run-first-btn")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("stage-source"));
    fireEvent.change(screen.getByLabelText("Repository URL"), { target: { value: "https://github.com/org/changed.git" } });

    expect(screen.queryByTestId("workspace-validate-result")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("stage-readiness"));
    expect(screen.getByTestId("setup-run-first-btn")).toBeDisabled();
  });

  it("clears readiness checklist after runtime selection changes", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-readiness"));
    fireEvent.click(screen.getByTestId("setup-doctor-btn"));
    await screen.findByTestId("setup-doctor-result");
    expect(screen.getByText("Local readiness passed.")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Runtime mode"), { target: { value: "headless" } });

    expect(screen.queryByTestId("setup-doctor-result")).not.toBeInTheDocument();
    expect(screen.queryByText("Local readiness passed.")).not.toBeInTheDocument();
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

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-analysis"));
    fireEvent.click(screen.getByTestId("run-init-btn"));

    const activeRunStrip = await screen.findByTestId("active-run-strip");
    expect(activeRunStrip).toHaveTextContent(runID);
    expect(activeRunStrip).toHaveTextContent("5/5 steps");

    await screen.findByTestId("run-status-panel");
    expect(await screen.findAllByTestId("analysis-step-review-card")).toHaveLength(5);
    fireEvent.click(screen.getByTestId("analysis-step-tab-diff"));
    await waitFor(() => expect(screen.getByTestId("git-diff-view")).toHaveTextContent("Workspace Git diff loaded."));

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

  it("renders pending runtime permission requests for the selected run", async () => {
    const runID = "run-permissions";
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID,
        runStarted: true,
        runStatus: {
          [runID]: {
            run_id: runID,
            pipeline: "init",
            status: "failed",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: "2026-04-03T12:00:02Z",
            warnings: [],
            pending_permissions: [
              {
                request_id: "perm-1",
                run_id: runID,
                step_id: "init.step1.collect",
                provider: "fake",
                action: "shell",
                path_or_command: "npm install",
                reason: "package install requires review",
                decision: {
                  request_id: "perm-1",
                  decision: "needs_user",
                  rule_id: "ask_unsafe_operation",
                  message: "operation requires explicit user approval",
                },
              },
            ],
            error_code: "runtime_permission_required",
            error: "runtime permission required",
          },
        },
      }),
    );

    await renderConsoleApp();
    fireEvent.click(screen.getByTestId("stage-analysis"));

    const permissionRecovery = await screen.findByTestId("runtime-permission-recovery");
    expect(permissionRecovery).toBeInTheDocument();
    expect(within(permissionRecovery).getByText("Permission triage")).toBeInTheDocument();
    expect(within(permissionRecovery).getByText("1 pending request")).toBeInTheDocument();
    expect(within(permissionRecovery).getByText("Blocked step")).toBeInTheDocument();
    expect(within(permissionRecovery).getByText("init.step1.collect")).toBeInTheDocument();
    expect(within(permissionRecovery).getByText("Policy rule")).toBeInTheDocument();
    expect(
      within(permissionRecovery).getByText("Use Readiness - Advanced runtime settings - Runtime Permissions to choose the intended mode/channel."),
    ).toBeInTheDocument();
    const permissionCards = screen.getByTestId("runs-pending-permissions-cards");
    expect(permissionCards).toHaveTextContent("perm-1");
    expect(permissionCards).toHaveTextContent("npm install");
    expect(await screen.findByTestId("runs-pending-permissions-table")).toBeInTheDocument();
    expect(screen.getAllByText("perm-1").length).toBeGreaterThan(0);
    expect(screen.getAllByText("needs_user").length).toBeGreaterThan(0);
    expect(screen.getAllByText("ask_unsafe_operation").length).toBeGreaterThan(0);
    expect(screen.getAllByText("npm install").length).toBeGreaterThan(0);
    expect(screen.getAllByText("package install requires review").length).toBeGreaterThan(0);
    const blockersPanel = screen.getByTestId("blockers-panel");
    expect(blockersPanel).toHaveTextContent("Permission: shell");
    expect(blockersPanel).toHaveTextContent("init.step1.collect paused for needs_user via ask_unsafe_operation");
    expect(blockersPanel).toHaveTextContent("Target: npm install.");
    expect(blockersPanel).toHaveTextContent("Reason: package install requires review");
  });

  it("renders Analysis V2 run progress, timeline, and shard drilldown", async () => {
    const runID = "run-analysis-v2";
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID,
        runStarted: true,
        runStatus: {
          [runID]: {
            run_id: runID,
            pipeline: "init",
            status: "failed",
            current_step: "init.step1.collect",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: "2026-04-03T12:00:02Z",
            warnings: ["collect warning"],
            error_code: "runtime_contract_failed",
            error: "collect manifest missing",
          },
        },
        runLogs: {
          [runID]: {
            run_id: runID,
            items: [
              {
                cursor: 1,
                timestamp: "2026-04-03T12:00:01Z",
                level: "error",
                kind: "event",
                step_id: "init.step1.collect",
                domain_id: "ftgo-application",
                message: "collect manifest missing",
                taskrun_path: "reports/taskruns/run-analysis-v2/staging/shards/payments-root-shard/runtime-execution.json",
                fields: { provider: "qwen-code", shard_id: "payments-root-shard", duration_ms: 2140 },
              },
              {
                cursor: 2,
                timestamp: "2026-04-03T12:00:02Z",
                level: "info",
                kind: "event",
                step_id: "init.step1.collect",
                domain_id: "ftgo-application",
                message: "runtime execution persisted",
                fields: { provider: "qwen-code", shard_id: "invoices-module-shard", shards_total: 2, succeeded: 1, failed: 1 },
              },
              {
                cursor: 3,
                timestamp: "2026-04-03T12:00:03Z",
                level: "warning",
                kind: "event",
                step_id: "init.step1.collect",
                domain_id: "ftgo-application",
                message: "focused artifact repair scheduled",
                fields: {
                  provider: "qwen-code",
                  shard_id: "payments-root-shard",
                  recovery_mode: "collect_pair_repair",
                  validation_error: 'documents[0].path references process-contaminated collect document file "root-overview.md"',
                },
              },
              {
                cursor: 4,
                timestamp: "2026-04-03T12:00:04Z",
                level: "error",
                kind: "event",
                step_id: "init.step1.collect",
                domain_id: "ftgo-application",
                message:
                  "focused artifact repair exhausted stage=collect_pair_repair (raw_output=reports/taskruns/run-analysis-v2/raw/payments/collect.json stdout_bytes=0 stdout_sha256=abc stderr_bytes=0 stderr_sha256=def)",
                fields: {
                  provider: "qwen-code",
                  shard_id: "payments-root-shard",
                  recovery_mode: "collect_pair_repair",
                  stall_phase: "pre_artifact",
                  exit_reason: "stall",
                  artifact_valid: false,
                  validation_error: "runtime_stalled_before_artifacts",
                },
              },
              {
                cursor: 5,
                timestamp: "2026-04-03T12:00:05Z",
                level: "error",
                kind: "event",
                step_id: "init.step1.collect",
                message: "run failed: partial shard failures detected",
                fields: { error_code: "run_partial_failed", partial_failure_count: 1 },
              },
            ],
            next_cursor: 5,
            eof: true,
          },
        },
        runArtifacts: {
          [runID]: {
            run_id: runID,
            artifacts: [
              { path: "reports/taskruns/run-analysis-v2/staging/shards/payments-root-shard/runtime-execution.json", kind: "runtime", label: "runtime execution" },
              { path: "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/runtime-execution.json", kind: "runtime", label: "runtime execution" },
              { path: "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/invoices-overview.md", kind: "report", label: "invoices overview" },
              { path: "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/shard-pack-manifest.json", kind: "manifest", label: "shard manifest" },
            ],
          },
        },
      }),
    );

    await renderConsoleApp();
    await screen.findByTestId("review-panel");

    fireEvent.click(screen.getByTestId("stage-analysis"));
    await waitFor(() => expect(screen.getByTestId("stage-analysis")).toHaveClass("is-selected"));

    const progress = await screen.findByTestId("analysis-run-progress");
    expect(progress).toHaveTextContent(runID);
    expect(progress).toHaveTextContent("fake");
    expect(progress).toHaveTextContent("init.step1.collect");
    expect(progress).toHaveTextContent("1 / 1");
    expect(within(progress).getByTestId("analysis-review-blocker-btn")).not.toBeDisabled();

    const timeline = screen.getByTestId("analysis-run-timeline");
    expect(timeline).toHaveTextContent("init.step0.constitution");
    expect(timeline).toHaveTextContent("init.step1.collect");
    expect(timeline).toHaveTextContent("blocked");

    const shardTable = await screen.findByTestId("analysis-shard-table");
    expect(shardTable).toHaveTextContent("payments-root-shard");
    expect(shardTable).toHaveTextContent("invoices-module-shard");
    expect(shardTable).toHaveTextContent("fake");
    expect(shardTable).toHaveTextContent("failed");
    expect(shardTable).toHaveTextContent("runtime-execution.json");
    expect(shardTable).toHaveTextContent("Runtime only");
    expect(shardTable).toHaveTextContent("authored markdown and shard-pack-manifest are missing");
    expect(shardTable).toHaveTextContent("Artifact pair present");
    expect(shardTable).toHaveTextContent("2s");
    expect(shardTable).toHaveTextContent("Duration unavailable");
    expect(shardTable).not.toHaveTextContent("not exposed");

    const drilldown = screen.getByTestId("analysis-failed-shard-details");
    expect(drilldown).toHaveTextContent("focused artifact repair exhausted");
    expect(drilldown).toHaveTextContent("Runtime record");
    expect(drilldown).toHaveTextContent("Authored markdown");
    expect(drilldown).toHaveTextContent("Manifest");
    expect(drilldown).toHaveTextContent("reports/taskruns/run-analysis-v2/staging/shards/payments-root-shard/runtime-execution.json");
    expect(drilldown).toHaveTextContent("missing");

    const liveDiagnostics = screen.getByTestId("analysis-live-diagnostics");
    expect(liveDiagnostics).toHaveTextContent("Live diagnostics");
    expect(liveDiagnostics).toHaveTextContent("artifact handoff");
    expect(liveDiagnostics).toHaveTextContent("Artifact handoff stalled");
    expect(liveDiagnostics).toHaveTextContent("1/2 ok");
    expect(liveDiagnostics).toHaveTextContent("1 failed");
    expect(liveDiagnostics).toHaveTextContent("1 scheduled / 0 completed / 1 exhausted");
    expect(liveDiagnostics).toHaveTextContent("collect_pair_repair");
    expect(liveDiagnostics).toHaveTextContent("1 actual / 0 valid-stop");
    expect(liveDiagnostics).toHaveTextContent("1 pre-artifact");
    expect(liveDiagnostics).toHaveTextContent("reports/taskruns/run-analysis-v2/raw/payments/collect.json");
    expect(liveDiagnostics).toHaveTextContent("runtime_stalled_before_artifacts");
    expect(liveDiagnostics).toHaveTextContent("Open the failed shard row and raw-output ref");
    expect(liveDiagnostics).toHaveTextContent("Retry after the provider artifact write path is fixed");
  });

  it("surfaces active provider stream when collect has no authored shard artifacts yet", async () => {
    const runID = "run-provider-stream";
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID,
        runStarted: true,
        onboardingStatus: {
          ok: true,
          launcher_mode: false,
          workspace_selected: true,
          workspace_ready: true,
          workspace: "/tmp/workspace",
          manifest_present: true,
          runtime: {
            selected: true,
            runtime: "headless",
            runtime_provider: "qwen-code",
            provider_source: "override",
          },
          can_enter_console: true,
          recent_workspaces: [],
        },
        runStatus: {
          [runID]: {
            run_id: runID,
            pipeline: "init",
            status: "running",
            current_step: "init.step1.collect",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: null,
            warnings: [],
            error_code: null,
            error: null,
          },
        },
        runLogs: {
          [runID]: {
            run_id: runID,
            items: [
              {
                cursor: 1,
                timestamp: "2026-04-03T12:00:01Z",
                level: "info",
                kind: "event",
                step_id: "init.step1.collect",
                domain_id: "ftgo-application",
                message: "runtime task started",
                fields: { provider: "qwen-code", shard_id: "ftgo-root-shard", shards_total: 16 },
              },
              {
                cursor: 2,
                timestamp: "2026-04-03T12:00:02Z",
                level: "info",
                kind: "runtime_output",
                stream: "stdout",
                step_id: "init.step1.collect",
                message: JSON.stringify({
                  type: "stream_event",
                  event: {
                    type: "content_block_delta",
                    delta: { type: "thinking_delta", thinking: "checking repository structure" },
                  },
                }),
              },
              {
                cursor: 3,
                timestamp: "2026-04-03T12:00:03Z",
                level: "info",
                kind: "runtime_output",
                stream: "stdout",
                step_id: "init.step1.collect",
                message: JSON.stringify({
                  type: "stream_event",
                  event: {
                    type: "content_block_delta",
                    delta: { type: "text_delta", text: "drafting collect artifacts" },
                  },
                }),
              },
            ],
            next_cursor: 3,
            eof: false,
          },
        },
        runArtifacts: {
          [runID]: {
            run_id: runID,
            artifacts: [],
          },
        },
        runReviewSummary: {
          [runID]: {
            run_id: runID,
            pipeline: "init",
            status: "running",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: null,
            current_step: "init.step1.collect",
            warnings: [],
            error_code: null,
            error: null,
            steps: [
              {
                step_id: "step0_constitution",
                label: "Charter",
                state: "done",
                provider: "qwen-code",
                artifact_count: 1,
                artifact_paths: ["charter/overview.md"],
                taskrun_paths: [],
                warnings_count: 0,
                errors_count: 0,
                last_message: "charter ready",
              },
              {
                step_id: "step1_collect",
                label: "Collect",
                state: "active",
                provider: "qwen-code",
                artifact_count: 0,
                artifact_paths: [],
                taskrun_paths: [],
                warnings_count: 0,
                errors_count: 0,
                last_message: "runtime stream active",
              },
            ],
          },
        },
      }),
    );

    await renderConsoleApp();
    await screen.findByTestId("console-shell");

    fireEvent.click(screen.getByTestId("stage-analysis"));
    await waitFor(() => expect(screen.getByTestId("stage-analysis")).toHaveClass("is-selected"));

    expect(screen.queryByTestId("analysis-failure-recovery")).not.toBeInTheDocument();
    const liveDiagnostics = await screen.findByTestId("analysis-live-diagnostics");
    expect(liveDiagnostics).toHaveTextContent("provider stream");
    expect(liveDiagnostics).toHaveTextContent("Provider output is streaming, but no authored shard artifact pair is visible yet");
    expect(liveDiagnostics).toHaveTextContent("Run signal");
    expect(liveDiagnostics).toHaveTextContent("Artifact pair pending");
    expect(liveDiagnostics).toHaveTextContent("runtime stream is active; authored markdown and shard-pack-manifest are not visible yet");
    expect(liveDiagnostics).toHaveTextContent("Provider stream");
    expect(liveDiagnostics).toHaveTextContent("2 chunks");
    expect(liveDiagnostics).toHaveTextContent("2 JSON stream events");
    expect(liveDiagnostics).toHaveTextContent("thinking_delta, text_delta");
    expect(liveDiagnostics).toHaveTextContent("Watch for authored markdown plus shard-pack-manifest before treating provider output as collect progress.");
    expect(liveDiagnostics).toHaveTextContent("If collect stalls or repair starts, use raw-output metadata instead of reading the full provider stream.");
  });

  it("copies run logs using the active line+fields view", async () => {
    const runID = "run-copy-fields";
    const writeText = vi.fn(async (_text: string) => undefined);
    vi.stubGlobal("navigator", {
      clipboard: {
        writeText,
      },
    });
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID,
        runStarted: true,
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
                fields: {
                  task_id: "task-1",
                  artifact_count: 2,
                },
              },
            ],
            next_cursor: 1,
            eof: true,
          },
        },
      }),
    );

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-analysis"));
    await screen.findByTestId("run-logs-content");

    fireEvent.change(screen.getByTestId("run-logs-view-select"), { target: { value: "line+fields" } });
    fireEvent.click(screen.getByTestId("run-logs-copy-btn"));

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledTimes(1);
    });
    const copiedText = writeText.mock.calls[0][0];
    expect(copiedText).toContain('"task_id": "task-1"');
    expect(copiedText).toContain('"artifact_count": 2');
  });

  it("handles cancel requests with accepted, missing, and terminal responses", async () => {
    const acceptedRunID = "run-cancel-accepted";
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID: acceptedRunID,
        runStarted: true,
        runStatus: {
          [acceptedRunID]: {
            run_id: acceptedRunID,
            pipeline: "refresh",
            status: "running",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: null,
            current_step: "refresh.step2.asis_docs",
            warnings: [],
            error_code: null,
            error: null,
          },
        },
        cancelResponses: {
          [acceptedRunID]: {
            status: 202,
            body: { status: "cancel_requested" },
          },
        },
      }),
    );

    await renderConsoleApp();
    fireEvent.click(screen.getByTestId("stage-analysis"));
    await screen.findByTestId("run-status-panel");
    fireEvent.click(screen.getByTestId("run-cancel-btn"));
    expect(await screen.findByText(`Cancel requested for ${acceptedRunID}`)).toBeInTheDocument();
  });

  it("switches away from a missing run when cancel returns 404", async () => {
    const baseFetch = createFetchMock();
    let canceled = false;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        if (!canceled) {
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

      if (method === "POST" && url === "/api/pipeline/runs/run-stale/cancel") {
        canceled = true;
        return jsonResponse(
          {
            error: {
              code: "not_found",
              message: "run-stale no longer exists",
            },
          },
          404,
        );
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

    await renderConsoleApp();
    fireEvent.click(screen.getByTestId("stage-analysis"));

    await waitFor(() => {
      expect(screen.getByTestId("run-status-run-id").textContent).toBe("run-stale");
    });

    fireEvent.click(screen.getByTestId("run-cancel-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("run-status-run-id").textContent).toBe("run-fresh");
    });
    expect(screen.getByText("Selected run no longer exists; switched to run-fresh.")).toBeInTheDocument();
  });

  it("reports when cancel hits an already-terminal run", async () => {
    const runID = "run-cancel-terminal";
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
            finished_at: null,
            current_step: "refresh.step2.asis_docs",
            warnings: [],
            error_code: null,
            error: null,
          },
        },
        cancelResponses: {
          [runID]: {
            status: 409,
            body: { error: { code: "run_not_cancelable", message: "already terminal" } },
          },
        },
      }),
    );

    await renderConsoleApp();
    fireEvent.click(screen.getByTestId("stage-analysis"));
    await screen.findByTestId("run-status-panel");
    fireEvent.click(screen.getByTestId("run-cancel-btn"));
    expect(await screen.findByText("Selected run is already terminal.")).toBeInTheDocument();
  });

  it("saves and resets runtime timeout, execution, and permission settings", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    fireEvent.click(screen.getByTestId("stage-readiness"));

    const timeoutInput = await screen.findByTestId("runtime-timeout-input-step_timeout_sec");
    fireEvent.change(timeoutInput, { target: { value: "2400" } });
    fireEvent.click(screen.getByTestId("runtime-timeouts-save-btn"));
    expect(await screen.findByText("Runtime timeouts saved")).toBeInTheDocument();

    const timeoutPutCalls = fetchMock.mock.calls.filter((call) => call[0] === "/api/runtime/timeouts" && call[1]?.method === "PUT");
    expect(timeoutPutCalls).toHaveLength(1);
    expect(JSON.parse(String(timeoutPutCalls[0][1]?.body ?? "{}"))).toMatchObject({
      timeouts: { step_timeout_sec: 2400 },
    });

    fireEvent.click(screen.getByRole("button", { name: "Reset balanced defaults" }));
    expect(await screen.findByText("Runtime timeouts reset to balanced defaults")).toBeInTheDocument();

    const executionStrategy = screen.getByTestId("runtime-execution-strategy-select");
    fireEvent.change(executionStrategy, { target: { value: "parallel" } });
    fireEvent.change(screen.getByTestId("runtime-execution-max-parallel-input"), { target: { value: "3" } });
    fireEvent.click(screen.getByTestId("runtime-execution-save-btn"));
    expect(await screen.findByText("Runtime execution profile saved")).toBeInTheDocument();

    const executionPutCalls = fetchMock.mock.calls.filter((call) => call[0] === "/api/runtime/execution" && call[1]?.method === "PUT");
    expect(executionPutCalls).toHaveLength(1);
    expect(JSON.parse(String(executionPutCalls[0][1]?.body ?? "{}"))).toMatchObject({
      execution: {
        strategy: "parallel",
        max_parallel_tasks: 3,
      },
    });

    fireEvent.click(screen.getByRole("button", { name: "Reset execution defaults" }));
    expect(await screen.findByText("Runtime execution profile reset to defaults")).toBeInTheDocument();

    const permissionMode = screen.getByTestId("runtime-permission-mode-select");
    fireEvent.change(permissionMode, { target: { value: "managed" } });
    fireEvent.change(screen.getByTestId("runtime-permission-approval-channel-select"), { target: { value: "ui" } });
    fireEvent.click(screen.getByTestId("runtime-permissions-save-btn"));
    expect(await screen.findByText("Runtime permissions saved")).toBeInTheDocument();

    const permissionsPutCalls = fetchMock.mock.calls.filter((call) => call[0] === "/api/runtime/permissions" && call[1]?.method === "PUT");
    expect(permissionsPutCalls).toHaveLength(1);
    expect(JSON.parse(String(permissionsPutCalls[0][1]?.body ?? "{}"))).toMatchObject({
      permissions: {
        mode: "managed",
        approval_channel: "ui",
      },
    });

    fireEvent.click(screen.getByRole("button", { name: "Reset permission defaults" }));
    expect(await screen.findByText("Runtime permissions reset to defaults")).toBeInTheDocument();
  }, 10_000);

  it("opens runtime execution artifacts from run logs quick action", async () => {
    const runID = "run-runtime-artifact";
    const taskrunPath = `reports/taskruns/${runID}/refresh-step1.collect/runtime-execution.json`;
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID,
        runStarted: true,
        runLogs: {
          [runID]: {
            run_id: runID,
            items: [
              {
                cursor: 0,
                timestamp: "2026-04-03T12:00:00Z",
                level: "info",
                kind: "event",
                step_id: "refresh.step1.collect",
                message: "runtime execution persisted",
                taskrun_path: taskrunPath,
              },
            ],
            next_cursor: 1,
            eof: true,
          },
        },
        runArtifacts: {
          [runID]: {
            run_id: runID,
            artifacts: [
              { path: taskrunPath, kind: "taskrun", label: "Runtime Execution" },
            ],
          },
        },
        artifactText: {
          [taskrunPath]: "{\"task\":\"ok\"}\n",
        },
      }),
    );

    await renderConsoleApp();
    fireEvent.click(screen.getByTestId("stage-analysis"));

    fireEvent.click(await screen.findByText("Runtime execution artifacts (1)"));
    const quickAction = await screen.findByRole("button", { name: /open runtime execution artifact:/i });
    fireEvent.click(quickAction);

    fireEvent.click(screen.getByTestId("stage-review"));

    await waitFor(() => {
      expect(screen.getByTestId("run-artifact-selected-path").textContent).toBe(taskrunPath);
      expect(screen.getByTestId("run-artifact-content").textContent ?? "").toContain("\"task\":\"ok\"");
    });
  });

  it("clears a stale selected artifact when switching to a different run", async () => {
    const baseFetch = createFetchMock();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({
          items: [
            {
              run_id: "run-old",
              pipeline: "refresh",
              status: "succeeded",
              started_at: "2026-04-03T12:02:00Z",
              finished_at: "2026-04-03T12:02:30Z",
              warnings: [],
              error_code: null,
              error: null,
            },
            {
              run_id: "run-new",
              pipeline: "refresh",
              status: "succeeded",
              started_at: "2026-04-03T12:01:00Z",
              finished_at: "2026-04-03T12:01:30Z",
              warnings: [],
              error_code: null,
              error: null,
            },
          ],
        });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-old") {
        return jsonResponse({
          run_id: "run-old",
          pipeline: "refresh",
          status: "succeeded",
          started_at: "2026-04-03T12:02:00Z",
          finished_at: "2026-04-03T12:02:30Z",
          warnings: [],
          error_code: null,
          error: null,
        });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-new") {
        return jsonResponse({
          run_id: "run-new",
          pipeline: "refresh",
          status: "succeeded",
          started_at: "2026-04-03T12:01:00Z",
          finished_at: "2026-04-03T12:01:30Z",
          warnings: [],
          error_code: null,
          error: null,
        });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-old/artifacts") {
        return jsonResponse({
          run_id: "run-old",
          artifacts: [
            { path: "reports/as-is/old.md", kind: "report", label: "Old artifact" },
          ],
        });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-new/artifacts") {
        return jsonResponse({
          run_id: "run-new",
          artifacts: [
            { path: "reports/as-is/new.md", kind: "report", label: "New artifact" },
          ],
        });
      }

      if (method === "GET" && url.startsWith("/api/pipeline/runs/run-old/logs?")) {
        return jsonResponse({ run_id: "run-old", items: [], next_cursor: 0, eof: true });
      }

      if (method === "GET" && url.startsWith("/api/pipeline/runs/run-new/logs?")) {
        return jsonResponse({ run_id: "run-new", items: [], next_cursor: 0, eof: true });
      }

      if (method === "GET" && url === "/api/artifacts?path=reports%2Fas-is%2Fold.md") {
        return textResponse("# Old artifact\n");
      }

      if (method === "GET" && url === "/api/artifacts?path=reports%2Fas-is%2Fnew.md") {
        return textResponse("# New artifact\n");
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-review"));
    fireEvent.click(screen.getByTestId("stage-review"));
    fireEvent.click(await screen.findByRole("button", { name: /reports\/as-is\/old\.md/i }));

    await waitFor(() => {
      expect(screen.getByTestId("run-artifact-selected-path").textContent).toBe("reports/as-is/old.md");
      expect(screen.getByTestId("run-artifact-content").textContent ?? "").toContain("# Old artifact");
    });

    fireEvent.click(screen.getByTestId("stage-analysis"));
    fireEvent.click(screen.getByRole("button", { name: "run-new" }));

    await waitFor(() => {
      expect(screen.getByTestId("run-status-run-id").textContent).toBe("run-new");
    });

    fireEvent.click(screen.getByTestId("stage-review"));
    fireEvent.click(screen.getByTestId("stage-review"));

    await waitFor(() => {
      expect(screen.getByTestId("run-artifact-selected-path").textContent).toBe("reports/as-is/new.md");
      expect(screen.getByTestId("run-artifact-content").textContent ?? "").toContain("# New artifact");
      expect(screen.getByTestId("run-artifact-content").textContent ?? "").not.toContain("# Old artifact");
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

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-analysis"));
    await waitFor(() => expect(screen.getByTestId("stage-analysis")).toHaveClass("is-selected"));

    await screen.findByTestId("run-status-panel");
    expect(screen.getByTestId("run-status-value").textContent).toBe("failed");
    expect(screen.getByTestId("run-lifecycle-value").textContent).toBe("terminal");
    expect(screen.getByText("Current step: refresh.step3.findings")).toBeInTheDocument();
    expect(screen.getByText("Error code: run_partial_failed")).toBeInTheDocument();
    expect(screen.getByText("Error: runtime draft manifest invalid")).toBeInTheDocument();
    expect(screen.getByTestId("run-status-warnings").textContent ?? "").toContain("collect coverage incomplete");
    expect(screen.getByTestId("run-status-warnings").textContent ?? "").toContain("draft promotion skipped");
    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("Review blocker");

    const recovery = screen.getByTestId("analysis-failure-recovery");
    expect(recovery).toHaveTextContent("Recovery path");
    expect(recovery).toHaveTextContent("run_partial_failed");
    expect(recovery).toHaveTextContent("refresh.step3.findings");
    expect(recovery).toHaveTextContent("2");
    expect(screen.getByTestId("analysis-retry-run-btn")).toHaveTextContent("Retry refresh");

    fireEvent.click(screen.getByTestId("inspector-primary-action"));
    expect(screen.getByTestId("stage-analysis")).toHaveClass("is-selected");
    await waitFor(() => expect(document.activeElement).toBe(screen.getByTestId("analysis-failed-shard-details")));
  });

  it("routes runner unavailable analysis blockers to provider readiness before retry", async () => {
    const runID = "run-provider-unavailable";
    const fetchMock = createFetchMock({
      runID,
      runStarted: true,
      runStatus: {
        [runID]: {
          run_id: runID,
          pipeline: "refresh",
          status: "failed",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: "2026-04-03T12:01:00Z",
          current_step: "refresh.step1.collect",
          warnings: [],
          error_code: "runner_unavailable",
          error: "provider quota exhausted",
        },
      },
      runArtifacts: {
        [runID]: {
          run_id: runID,
          artifacts: [{ path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" }],
        },
      },
      onboardingStatus: {
        ok: true,
        launcher_mode: false,
        workspace_selected: true,
        workspace_ready: true,
        workspace: "/tmp/workspace",
        manifest_present: true,
        runtime: {
          selected: true,
          runtime: "headless",
          runtime_provider: "codex-code",
          provider_source: "override",
        },
        can_enter_console: true,
        recent_workspaces: [],
      },
      doctorResponse: {
        ok: false,
        summary: "needs attention",
        checks: [
          { id: "git", label: "Git", status: "pass", message: "git found" },
          {
            id: "runtime_provider",
            label: "Runtime provider",
            status: "fail",
            message: "Provider ID: codex-code; usage limit reached",
            suggestion: "Run codex login, confirm quota, or set ACP_CODEX_CMD to a working provider command.",
          },
        ],
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    await screen.findByTestId("review-panel");

    fireEvent.click(screen.getByTestId("stage-analysis"));
    await waitFor(() => expect(screen.getByTestId("stage-analysis")).toHaveClass("is-selected"));

    await screen.findByTestId("run-status-panel");
    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("Check provider readiness");
    expect(screen.getByTestId("next-action-panel")).toHaveTextContent("verify binary/auth/quota in Readiness before retrying the same pipeline");
    expect(screen.getByTestId("blockers-panel")).toHaveTextContent("Provider unavailable");
    expect(screen.getByTestId("blockers-panel")).toHaveTextContent("check Readiness provider setup, binary/auth/quota");

    const recovery = screen.getByTestId("analysis-failure-recovery");
    expect(recovery).toHaveTextContent("Provider/tool availability blocked artifact creation");
    expect(recovery).toHaveTextContent("refresh.step1.collect");
    const liveDiagnostics = screen.getByTestId("analysis-live-diagnostics");
    expect(liveDiagnostics).toHaveTextContent("provider check");
    expect(liveDiagnostics).toHaveTextContent("provider unavailable before shard ids were emitted");
    expect(liveDiagnostics).toHaveTextContent("Check Readiness provider setup, binary/auth/quota before retrying the same pipeline");

    fireEvent.click(screen.getByTestId("stage-publish"));
    expect(await screen.findByTestId("publish-panel")).toBeInTheDocument();
    expect(screen.getByTestId("publish-gate-panel")).toHaveTextContent("Provider unavailable");
    expect(screen.getByTestId("publish-gate-panel")).toHaveTextContent("run a successful analysis before publishing");

    fireEvent.click(screen.getByTestId("stage-analysis"));
    const startCallsBefore = fetchMock.mock.calls.filter((call) => call[0] === "/api/pipeline/init" || call[0] === "/api/pipeline/refresh").length;
    fireEvent.click(screen.getByTestId("inspector-primary-action"));
    expect(screen.getByTestId("stage-readiness")).toHaveClass("is-selected");
    const providerRecovery = screen.getByTestId("provider-readiness-recovery");
    expect(providerRecovery).toHaveTextContent("Provider readiness recovery");
    expect(providerRecovery).toHaveTextContent("codex-code");
    expect(providerRecovery).toHaveTextContent("ACP_CODEX_CMD");
    expect(providerRecovery).toHaveTextContent("runner_unavailable");
    expect(providerRecovery).toHaveTextContent("provider quota exhausted");

    fireEvent.click(screen.getByTestId("setup-doctor-btn"));
    await screen.findByTestId("setup-doctor-result");
    expect(screen.getByTestId("provider-readiness-recovery")).toHaveTextContent("Runtime provider: fail");
    expect(screen.getByTestId("provider-readiness-recovery")).toHaveTextContent("usage limit reached");
    expect(screen.getByTestId("provider-readiness-recovery")).toHaveTextContent("confirm quota");
    expect(fetchMock.mock.calls.filter((call) => call[0] === "/api/pipeline/init" || call[0] === "/api/pipeline/refresh")).toHaveLength(startCallsBefore);
  });

  it("explains headless provider probe timeouts before retrying analysis", async () => {
    const runID = "run-qwen-probe-timeout";
    const timeoutMessage = "qwen: headless_probe_timeout: qwen headless probe timed out after 30s";
    const fetchMock = createFetchMock({
      runID,
      runStatus: {
        [runID]: {
          run_id: runID,
          pipeline: "init",
          status: "failed",
          started_at: "2026-07-09T06:56:00Z",
          finished_at: "2026-07-09T06:57:00Z",
          current_step: "init.step1.collect",
          warnings: [],
          error_code: "runner_unavailable",
          error: timeoutMessage,
        },
      },
      onboardingStatus: {
        ok: true,
        launcher_mode: false,
        workspace_selected: true,
        workspace_ready: true,
        workspace: "/tmp/workspace",
        manifest_present: true,
        runtime: {
          selected: true,
          runtime: "headless",
          runtime_provider: "qwen-code",
          provider_source: "override",
        },
        can_enter_console: true,
        recent_workspaces: [],
      },
      doctorResponse: {
        ok: false,
        summary: "needs attention",
        checks: [
          { id: "git", label: "Git", status: "pass", message: "git found" },
          {
            id: "runtime_provider",
            label: "Runtime provider",
            status: "fail",
            message: timeoutMessage,
            suggestion: "Confirm qwen login/quota and rerun readiness.",
          },
        ],
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    await screen.findByTestId("stage-readiness");

    fireEvent.click(screen.getByTestId("stage-readiness"));
    await waitFor(() => expect(screen.getByTestId("stage-readiness")).toHaveClass("is-selected"));
    fireEvent.click(screen.getByTestId("setup-doctor-btn"));
    await screen.findByTestId("setup-doctor-result");

    const providerRecovery = await screen.findByTestId("provider-readiness-recovery");
    expect(providerRecovery).toHaveTextContent("qwen-code");
    expect(providerRecovery).toHaveTextContent("ACP_QWEN_CMD");
    expect(providerRecovery).toHaveTextContent("Headless probe timeout");
    expect(providerRecovery).toHaveTextContent("Text readiness probe");
    expect(providerRecovery).toHaveTextContent("qwen did not return the readiness response");
    expect(providerRecovery).toHaveTextContent("short headless prompt outside ACP");
    expect(providerRecovery).toHaveTextContent("retry Analysis only after Runtime provider passes");
  });

  it("explains terminal canceled runs without presenting them as runtime failures", async () => {
    const runID = "run-canceled-terminal";
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        runID,
        runStarted: true,
        runList: [
          {
            run_id: runID,
            pipeline: "refresh",
            status: "failed",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: "2026-04-03T12:02:00Z",
            current_step: "refresh.step2.asis_docs",
            warnings: [],
            error_code: "run_canceled",
            error: "canceled by request",
          },
        ],
        runStatus: {
          [runID]: {
            run_id: runID,
            pipeline: "refresh",
            status: "failed",
            started_at: "2026-04-03T12:00:00Z",
            finished_at: "2026-04-03T12:02:00Z",
            current_step: "refresh.step2.asis_docs",
            warnings: [],
            error_code: "run_canceled",
            error: "canceled by request",
          },
        },
      }),
    );

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-analysis"));

    await screen.findByTestId("run-status-panel");
    expect(screen.getByTestId("run-status-value").textContent).toBe("canceled");
    expect(screen.getByTestId("analysis-run-progress")).toHaveTextContent("canceled");
    expect(screen.getByTestId("runs-history-panel")).toHaveTextContent("Failed: 0");
    expect(screen.getByTestId("runs-history-panel")).toHaveTextContent("Canceled: 1");
    expect(screen.getByTestId("runs-history-table")).toHaveTextContent("canceled");
    const activeRunStrip = await screen.findByTestId("active-run-strip");
    expect(activeRunStrip).toHaveTextContent("canceled");
    expect(activeRunStrip).toHaveTextContent("Stopped step");
    expect(screen.getByTestId("activity-drawer-toggle")).toHaveTextContent("canceled run");
    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("Run was canceled before log entries were captured: run_canceled");
    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("Taskrun evidence remains in History.");
    const recovery = screen.getByTestId("analysis-failure-recovery");
    expect(recovery).toHaveTextContent("Canceled run");
    expect(recovery).toHaveTextContent("run_canceled");
    expect(recovery).toHaveTextContent("Stopped step");
    expect(recovery).toHaveTextContent("refresh.step2.asis_docs");
    expect(recovery).toHaveTextContent("The run stopped by request");
    expect(recovery).toHaveTextContent("the canceled run and its taskrun evidence stay in History");
    expect(screen.getByTestId("analysis-retry-run-btn")).toHaveTextContent("Run refresh again");
    expect(screen.getByTestId("analysis-review-recovery-btn")).toHaveTextContent("Review retained evidence");
    expect(screen.getByTestId("right-inspector")).toHaveTextContent("Review retained run evidence");
    expect(screen.getByTestId("right-inspector")).toHaveTextContent("The selected run was canceled; inspect retained History evidence");
    expect(screen.getByTestId("blockers-panel")).toHaveTextContent("Canceled run");
    expect(screen.getByTestId("blockers-panel")).toHaveTextContent("taskrun evidence remains in History");
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

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-analysis"));

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

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-analysis"));

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
        runList: [
          {
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
        ],
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

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-analysis"));

    await screen.findByTestId("run-status-panel");
    expect(screen.getByTestId("run-status-value").textContent).toBe("recovered");
    expect(screen.getByTestId("run-lifecycle-value").textContent).toBe("recovered");
    expect(screen.getByTestId("analysis-run-progress")).toHaveTextContent("recovered");
    expect(screen.getByTestId("runs-history-panel")).toHaveTextContent("Failed: 0");
    expect(screen.getByTestId("runs-history-panel")).toHaveTextContent("Recovered: 1");
    const activeRunStrip = await screen.findByTestId("active-run-strip");
    expect(activeRunStrip).toHaveTextContent("recovered");
    expect(activeRunStrip).toHaveTextContent("Recovered step");
    expect(screen.getByTestId("activity-drawer-toggle")).toHaveTextContent("recovered run");
    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("Run was reconciled after restart before log entries were captured: run_reconciled_after_restart");
    expect(screen.getByTestId("activity-empty-state")).toHaveTextContent("History retains the run; start a new run if analysis still matters.");
    const recovery = screen.getByTestId("analysis-failure-recovery");
    expect(recovery).toHaveTextContent("Recovered after restart");
    expect(recovery).toHaveTextContent("ACP reconciled a stale run after restart");
    expect(recovery).toHaveTextContent("the reconciled run and its taskrun evidence stay in History");
    expect(screen.getByTestId("analysis-retry-run-btn")).toHaveTextContent("Run refresh again");
    expect(screen.getByTestId("analysis-review-recovery-btn")).toHaveTextContent("Review retained evidence");
    expect(screen.getByTestId("right-inspector")).toHaveTextContent("Review retained run evidence");
    expect(screen.getByTestId("right-inspector")).toHaveTextContent("The selected run was recovered after restart; inspect retained History evidence");
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

    await renderConsoleApp();
    fireEvent.click(screen.getByTestId("stage-analysis"));

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

    await renderConsoleApp();
    fireEvent.click(screen.getByTestId("stage-analysis"));

    await waitFor(() => {
      expect(screen.getByTestId("run-status-run-id").textContent).toBe("run-stale");
    });

    await waitFor(() => {
      expect(runListCalls).toBeGreaterThanOrEqual(2);
    }, { timeout: 4000 });

    expect(screen.getByTestId("run-status-run-id").textContent).toBe("run-stale");
    expect(screen.queryByText(/Selected run no longer exists;/)).not.toBeInTheDocument();
  }, 10000);

  it("shows diagrams surface in Review stage and renders Mermaid preview", async () => {
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

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-review"));
    fireEvent.click(screen.getByTestId("stage-review"));

    const diagramButton = await screen.findByRole("button", { name: /reports\/diagrams\/c4-context\.mmd/i });
    fireEvent.click(diagramButton);

    await waitFor(() => {
      const preview = screen.getByTestId("run-diagram-content-panel").innerHTML;
      expect(preview).toContain("<svg");
    });
  });

  it("edits baseline prompt file in Charter stage and saves it", async () => {
    const fetchMock = createFetchMock({
      artifactText: {
        "skills/prompt-packs/qa.md": "qa prompt baseline\n",
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-charter"));

    expect(await screen.findByText(/step0\/step1\/step3\/step4/i)).toBeInTheDocument();
    expect(screen.queryByText(/collect`\/`findings/i)).not.toBeInTheDocument();

    const select = await screen.findByLabelText(/select artifact/i);
    fireEvent.change(select, { target: { value: "skills/prompt-packs/qa.md" } });

    const editor = await screen.findByLabelText("skills/prompt-packs/qa.md");
    fireEvent.change(editor, { target: { value: "qa prompt baseline\nupdated line\n" } });
    fireEvent.click(screen.getByRole("button", { name: /save selected baseline artifact/i }));

    await screen.findByText("Saved skills/prompt-packs/qa.md");

    const saveCalls = fetchMock.mock.calls.filter((call) => call[0] === "/api/artifacts/write");
    expect(saveCalls.length).toBe(1);
  });

  it("executes git-helper commit and proposal-branch actions from Charter stage", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("stage-charter"));

    const commitInput = await screen.findByLabelText("Commit message");
    fireEvent.change(commitInput, { target: { value: "feat: tighten prompt policy" } });
    fireEvent.click(screen.getByTestId("git-commit-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("baseline-git-helper-panel")).toHaveTextContent("committed: feat: tighten prompt policy");
      expect(screen.getByTestId("git-publication-panel")).toHaveTextContent("committed: feat: tighten prompt policy");
    });

    const branchInput = screen.getByLabelText("Proposal branch");
    fireEvent.change(branchInput, { target: { value: "proposal/prompt-policy" } });
    fireEvent.click(screen.getByTestId("git-proposal-branch-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("baseline-git-helper-panel")).toHaveTextContent("checked out proposal/prompt-policy");
      expect(screen.getByTestId("git-publication-panel")).toHaveTextContent("checked out proposal/prompt-policy");
    });

    const commitCalls = fetchMock.mock.calls.filter((call) => call[0] === "/api/git/commit");
    expect(commitCalls).toHaveLength(1);
    const branchCalls = fetchMock.mock.calls.filter((call) => call[0] === "/api/git/proposal-branch");
    expect(branchCalls).toHaveLength(1);
  });
});
