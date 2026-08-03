import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
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
  qaProposalResponse?: { status: number; body: MockJSON };
  onboardingStatus?: MockJSON;
  onboardingWorkspaceSelectionStatus?: MockJSON;
  systemVersion?: MockJSON;
  gitCommitResponse?: { status: number; body: MockJSON };
  proposalBranchResponse?: { status: number; body: MockJSON };
  workspaceHealthResponse?: MockJSON;
  workspaceHealthStatus?: number;
  knowledgeResponse?: MockJSON;
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

function deferredResponse() {
  let resolve!: (response: Response) => void;
  const promise = new Promise<Response>((nextResolve) => {
    resolve = nextResolve;
  });
  return { promise, resolve };
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
    state: "dirty",
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
    console_entered: true,
    can_switch_runtime: false,
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
  for (const [fixtureRunID, payload] of Object.entries(runArtifacts)) {
    const artifacts = Array.isArray(payload.artifacts) ? (payload.artifacts as MockJSON[]) : [];
    const indexPath = `reports/taskruns/${fixtureRunID}/staging/final/final-run-index.json`;
    if (artifacts.length === 0 || artifacts.some((artifact) => artifact.path === indexPath)) {
      continue;
    }
    const canonicalDocuments = artifacts
      .filter((artifact) => typeof artifact.path === "string" && !String(artifact.path).startsWith("reports/taskruns/"))
      .map((artifact) => {
        const canonicalPath = String(artifact.path);
        const stagedPath = `reports/taskruns/${fixtureRunID}/staging/final/${canonicalPath}`;
        artifactText[stagedPath] = artifactText[canonicalPath] ?? `# ${String(artifact.label ?? canonicalPath)}\n`;
        return {
          canonical_path: canonicalPath,
          staged_path: stagedPath,
          kind: artifact.kind ?? "report",
          title: artifact.label ?? canonicalPath,
        };
      });
    artifacts.push({ path: indexPath, kind: "taskrun", label: "Final run index" });
    artifactText[indexPath] = JSON.stringify({ version: 1, run_id: fixtureRunID, generated_at: "2026-04-03T12:00:02Z", canonical_documents: canonicalDocuments });
  }
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

    if (method === "POST" && url === "/api/onboarding/enter-console") {
      return jsonResponse({ ...onboardingStatus, console_entered: true, can_switch_runtime: false });
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

    if (method === "GET" && url === "/api/workspace/health") {
      return jsonResponse(
        state.workspaceHealthResponse ?? {
          version: 1,
          generated_at: "2026-07-10T00:00:00Z",
          status: "pass",
          summary: { info: 0, warning: 0, error: 0 },
          items: [],
        },
        state.workspaceHealthStatus ?? 200,
      );
    }

    if (method === "GET" && url === "/api/knowledge") {
      return jsonResponse(state.knowledgeResponse ?? {
        version: 1,
        generated_at: "2026-07-15T00:00:00Z",
        source_mode: "promoted_current",
        status: "unavailable",
        entities: [],
        edges: [],
        artifacts: [],
        issues: [],
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

    const runSnapshotMatch = url.match(/^\/api\/pipeline\/runs\/([^/]+)\/snapshot$/);
    if (method === "GET" && runSnapshotMatch) {
      const requestedRunID = decodeURIComponent(runSnapshotMatch[1]);
      const indexPath = `reports/taskruns/${requestedRunID}/staging/final/final-run-index.json`;
      const rawIndex = artifactText[indexPath];
      if (rawIndex === undefined) {
        return jsonResponse({
          run_id: requestedRunID,
          status: "not_produced",
          artifacts: [],
          issues: [{ code: "snapshot_not_produced", message: `Run ${requestedRunID} has no final snapshot index.` }],
        });
      }
      const index = JSON.parse(rawIndex) as {
        citation_index_path?: string;
        canonical_documents?: Array<{ id?: string; canonical_path: string; staged_path: string; kind?: string; title?: string }>;
      };
      const snapshotArtifacts: MockJSON[] = (index.canonical_documents ?? []).map((document) => ({
        id: document.id,
        path: document.canonical_path,
        read_path: document.staged_path,
        canonical_path: document.canonical_path,
        kind: document.kind ?? "report",
        label: document.title ?? document.canonical_path,
        source_run_id: requestedRunID,
        source_mode: "run_snapshot",
      }));
      snapshotArtifacts.push({
        path: indexPath,
        read_path: indexPath,
        kind: "taskrun",
        label: "Final run index",
        source_run_id: requestedRunID,
        source_mode: "run_snapshot",
      });
      if (index.citation_index_path) {
        snapshotArtifacts.push({
          path: index.citation_index_path,
          read_path: index.citation_index_path,
          kind: "taskrun",
          label: "Citation index",
          source_run_id: requestedRunID,
          source_mode: "run_snapshot",
        });
      }
      return jsonResponse({ run_id: requestedRunID, status: "available", artifacts: snapshotArtifacts, issues: [] });
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

    if (method === "POST" && /^\/api\/qa\/runs\/[^/]+\/proposal-draft$/.test(url)) {
      return jsonResponse(
        state.qaProposalResponse?.body ?? {
          path: "proposals/qa-synthesis-qa-run-1-who-owns-payments",
          proposal_path: "proposals/qa-synthesis-qa-run-1-who-owns-payments/proposal.md",
          evidence_path: "proposals/qa-synthesis-qa-run-1-who-owns-payments/evidence.md",
          source_path: "proposals/qa-synthesis-qa-run-1-who-owns-payments/source-qa-answer.json",
          answer_digest: "a".repeat(64),
        },
        state.qaProposalResponse?.status ?? 201,
      );
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
          answer_digest: "a".repeat(64),
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
      if (state.gitCommitResponse) {
        return jsonResponse(state.gitCommitResponse.body, state.gitCommitResponse.status);
      }
      const payload = JSON.parse(String(init?.body ?? "{}")) as { message?: string };
      return jsonResponse({
        status: "ok",
        output: `committed: ${payload.message ?? ""}`.trim(),
      });
    }

    if (method === "POST" && url === "/api/git/proposal-branch") {
      if (state.proposalBranchResponse) {
        return jsonResponse(state.proposalBranchResponse.body, state.proposalBranchResponse.status);
      }
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

async function renderConsoleApp(path = "/setup") {
  window.history.replaceState({}, "", path);
  const view = render(<App />);
  await screen.findByTestId("top-status-bar");
  return view;
}

function navigateToStage(stage: "source" | "readiness" | "charter" | "analysis" | "review" | "proposals" | "ask" | "publish") {
  if (stage === "ask") {
    fireEvent.click(screen.getByTestId("stage-ask"));
    return;
  }
  const destination = stage === "analysis" ? "Runs" : stage === "source" || stage === "readiness" || stage === "charter" ? "Setup" : "Changes";
  const destinationLink = screen.getByRole("link", { name: destination });
  if (destinationLink.getAttribute("aria-current") !== "page") fireEvent.click(destinationLink);
  if (stage === "analysis") {
    const selectedRun = screen.queryByTestId("runs-history-table")?.querySelector<HTMLButtonElement>("tbody button");
    if (selectedRun) fireEvent.click(selectedRun);
  }
  if (stage !== "analysis") {
    fireEvent.click(screen.getByTestId(`stage-${stage}`));
  }
}

describe("App", () => {
  afterEach(() => {
    cleanup();
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

    fireEvent.click(screen.getByRole("button", { name: "Details" }));
    await waitFor(() => expect(screen.getByTestId("brand-version")).toHaveTextContent("v0.1.2"));
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

    await renderConsoleApp("/home");
	fireEvent.click(screen.getByRole("link", { name: "Setup" }));

    expect(screen.getByTestId("workspace-panel")).toBeInTheDocument();
    expect(screen.queryByTestId("runtime-timeouts-panel")).not.toBeInTheDocument();
    expect(screen.queryByTestId(`tab-${"settings"}`)).not.toBeInTheDocument();
    expect(screen.queryByTestId(`setup-${"stepper"}`)).not.toBeInTheDocument();

    navigateToStage("readiness");

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
    fireEvent.click(screen.getByRole("button", { name: "Continue to repositories" }));
    fireEvent.click(screen.getByTestId("onboarding-sources-save"));

    expect(await screen.findByText("Sources validated.")).toBeInTheDocument();
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Select the runner");
    fireEvent.click(screen.getByRole("button", { name: "Continue to analysis brief" }));
    fireEvent.click(screen.getByRole("button", { name: "Continue without brief" }));
    fireEvent.click(screen.getByTestId("onboarding-runtime-save"));

    await waitFor(() => expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Check local readiness"));

    fireEvent.click(screen.getByRole("button", { name: "Check readiness" }));
    await screen.findByTestId("onboarding-doctor-result");
    expect(screen.getByTestId("onboarding-progress-summary")).toHaveTextContent("Run first analysis");
    fireEvent.click(screen.getByRole("button", { name: "Review setup" }));
    await waitFor(() => expect(screen.getByTestId("onboarding-enter-console")).not.toBeDisabled());
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
    fireEvent.click(screen.getByRole("button", { name: "Continue to repositories" }));

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

    await renderConsoleApp("/home");

    await waitFor(() => expect(fetchMock.mock.calls.some((call) => call[0] === "/api/workspace/validate")).toBe(true));
    fireEvent.click(screen.getByRole("link", { name: "Setup" }));
    expect(await screen.findByTestId("source-repo-table")).toHaveTextContent("resolved");
    expect(screen.getByTestId("top-status-bar")).toHaveTextContent("Workspace ready");
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
    fireEvent.click(screen.getByRole("button", { name: "Details" }));

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

  it("renders primary destinations and reaches contextual product flows", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    for (const destination of ["Home", "Runs", "Architecture", "Changes", "Setup"]) {
      expect(screen.getByRole("link", { name: destination })).toBeInTheDocument();
    }
    expect(screen.queryByTestId("stage-rail")).not.toBeInTheDocument();

    navigateToStage("ask");
    expect(await screen.findByTestId("qa-panel")).toBeInTheDocument();

    navigateToStage("analysis");
    expect(await screen.findByTestId("runs-control-panel")).toBeInTheDocument();
  });

  it("renders the path-based shell without hidden legacy shell surfaces", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp("/home");

    expect(await screen.findByTestId("home-panel")).toBeInTheDocument();
    expect(screen.getByTestId("product-shell")).toBeInTheDocument();
    expect(screen.queryByTestId("stage-rail")).not.toBeInTheDocument();
    expect(screen.queryByTestId("right-inspector")).not.toBeInTheDocument();
    expect(screen.queryByTestId("activity-drawer")).not.toBeInTheDocument();
  });

  it("renders workspace health warnings in contextual Readiness", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        workspaceHealthResponse: {
          version: 1,
          generated_at: "2026-07-10T00:00:00Z",
          status: "warn",
          summary: { info: 1, warning: 1, error: 0 },
          items: [
            {
              id: "model.observation.missing_evidence",
              severity: "warning",
              title: "Observation entity \"svc.payments\" has no evidence",
              path: "model/entities/svc.payments.yaml",
              related_paths: [],
            },
            {
              id: "coverage.open_questions.count",
              severity: "info",
              title: "1 unresolved coverage question(s) are open",
              path: "reports/coverage/open-questions.md",
              related_paths: [],
            },
          ],
        },
      }),
    );

    await renderConsoleApp();

    navigateToStage("readiness");
    expect(await screen.findByTestId("workspace-health-summary")).toHaveTextContent("warn");
    expect(screen.getByTestId("workspace-health-items")).toHaveTextContent("Observation entity");
    expect(screen.getByTestId("workspace-health-items")).toHaveTextContent("model/entities/svc.payments.yaml");
  });

  it("renders workspace health scan failures without blocking the console", async () => {
    vi.stubGlobal(
      "fetch",
      createFetchMock({
        workspaceHealthStatus: 500,
        workspaceHealthResponse: {
          error: {
            code: "workspace_health_failed",
            message: "scan failed",
          },
        },
      }),
    );

    await renderConsoleApp();

    navigateToStage("readiness");
    expect(await screen.findByTestId("workspace-health-summary")).toHaveTextContent("scan failed");
    expect(screen.getByTestId("setup-run-first-btn")).toBeDisabled();
  });

  it("routes Home empty-evidence next action to Runs", async () => {
    const fetchMock = createFetchMock({ runID: "run-review-recovery" });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp("/home");

    const action = screen.getByRole("button", { name: "Open Runs" });
    expect(action).not.toBeDisabled();
    fireEvent.click(action);
    expect(await screen.findByTestId("analysis-run-progress")).toBeInTheDocument();
    expect(screen.getByTestId("destination-runs")).toHaveAttribute("aria-current", "page");
    expect(fetchMock.mock.calls.some((call) => call[0] === "/api/pipeline/init")).toBe(false);
  });

  it("keeps a direct Home route stable when the latest run has reviewable evidence", async () => {
    const runID = "run-home-evidence";
    vi.stubGlobal("fetch", createFetchMock({
      runID,
      runList: [{
        run_id: runID,
        pipeline: "init",
        status: "succeeded",
        started_at: "2026-04-03T12:00:00Z",
        finished_at: "2026-04-03T12:00:02Z",
        warnings: [],
        error_code: null,
        error: null,
      }],
      runArtifacts: {
        [runID]: {
          run_id: runID,
          artifacts: [{ path: "reports/as-is/overview.md", kind: "report", label: "Architecture Home" }],
        },
      },
      artifactText: { "reports/as-is/overview.md": "# Architecture Home\n" },
    }));

    await renderConsoleApp("/home");

    expect(await screen.findByTestId("home-panel")).toBeInTheDocument();
    await waitFor(() => expect(window.location.pathname).toBe("/home"));
    expect(screen.getByTestId("destination-home")).toHaveAttribute("aria-current", "page");
  });

  it("renders the Source V2 repo table with guided analysis scope summary", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    fireEvent.click(screen.getByTestId("setup-step-sources"));
    const sourceTable = await screen.findByTestId("source-repo-table");
    expect(screen.getByTestId("source-next-action")).toHaveTextContent("Next in Repositories");
    expect(screen.getByTestId("source-next-action")).toHaveTextContent("save and validate");
    expect(sourceTable).toHaveTextContent("Name");
    expect(sourceTable).toHaveTextContent("Source");
    expect(sourceTable).toHaveTextContent("Ref");
    expect(sourceTable).toHaveTextContent("Analysis include/exclude");
    expect(sourceTable).toHaveTextContent("all files");
    expect(sourceTable).toHaveTextContent("Git URL");
    expect(sourceTable).toHaveTextContent("https://github.com/org/my-service.git");
  });

  it("keeps review attention contextual while navigating destinations", async () => {
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

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");

    await screen.findByTestId("review-panel");
    await waitFor(() => expect(screen.getByText(/open question.*require review/i)).toBeInTheDocument());
    navigateToStage("proposals");
    expect(await screen.findByTestId("proposals-panel")).toBeInTheDocument();
    expect(screen.queryByTestId("next-action-panel")).not.toBeInTheDocument();
  });

  it("renders distinct Changes route models with server-authored Git truth", async () => {
    vi.stubGlobal("fetch", createFetchMock());
    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");

    for (const view of ["overview", "evidence", "findings", "diff", "proposals", "publish"] as const) {
      window.history.pushState({}, "", `/changes?run=run-1&view=${view}&source=snapshot&mode=rendered`);
      window.dispatchEvent(new PopStateEvent("popstate"));
      expect(await screen.findByTestId(`changes-route-${view}`)).toBeInTheDocument();
      expect(screen.getByTestId("changes-git-state")).toHaveTextContent("Git: dirty");
    }
  });

  it("canonicalizes invalid explicit Changes identity during PopState with replace semantics", async () => {
    vi.stubGlobal("fetch", createFetchMock());
    await renderConsoleApp("/changes?run=run-1&view=evidence&source=snapshot&mode=rendered");

    window.history.pushState({}, "", "/changes?run=run-1&view=invalid&source=foreign&mode=unsafe");
    window.dispatchEvent(new PopStateEvent("popstate"));

    expect(await screen.findByTestId("route-notice")).toHaveTextContent("view, source, mode");
    await waitFor(() => {
      expect(`${window.location.pathname}${window.location.search}`).toBe(
        "/changes?run=run-1&view=overview&source=snapshot&mode=rendered",
      );
    });
  });

  it("renders Readiness V2 cards and compact runtime profile summary", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    navigateToStage("readiness");

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

    navigateToStage("charter");

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

    navigateToStage("charter");

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

  it("supports browser history navigation across destination paths", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    fireEvent.click(screen.getByRole("link", { name: "Runs" }));
    expect(window.location.pathname).toBe("/runs");
    expect(await screen.findByTestId("runs-control-panel")).toBeInTheDocument();
    window.history.pushState({}, "", "/setup");
    fireEvent.popState(window);
    expect(await screen.findByTestId("workspace-panel")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Setup" })).toHaveAttribute("aria-current", "page");
  });

  it("sanitizes an explicitly missing run without falling back to another snapshot", async () => {
    vi.stubGlobal("fetch", createFetchMock({ runStarted: true }));

    await renderConsoleApp("/changes?run=missing-run&view=evidence&source=snapshot&mode=raw");

    expect(await screen.findByTestId("route-notice")).toHaveTextContent("missing-run is unavailable");
    await waitFor(() => expect(window.location.search).not.toContain("run="));
    expect(window.location.search).toContain("source=snapshot");
    expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("No evidence content is available");
    expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("unknown run");
  });

  it("sanitizes a stale current-workspace entity without inventing another selection", async () => {
    vi.stubGlobal("fetch", createFetchMock({
      knowledgeResponse: {
        version: 1, generated_at: "2026-07-15T00:00:00Z", source_mode: "promoted_current", status: "available",
        entities: [{ id: "svc.payments", type: "service", name: "Payments", path: "model/entities/svc.payments.yaml", provenance: { kind: "inference", confidence: 0.9 } }],
        edges: [], artifacts: [{ path: "model/entities/svc.payments.yaml", kind: "entity", name: "svc.payments.yaml" }], issues: [],
      },
    }));
    await renderConsoleApp("/knowledge?view=entities&entity=svc.missing&source=current");
    expect(await screen.findByTestId("route-notice")).toHaveTextContent("Entity svc.missing is unavailable");
    await waitFor(() => expect(window.location.search).not.toContain("entity="));
    expect(await screen.findByTestId("knowledge-panel")).toHaveTextContent("Current workspace");
    expect(screen.queryByTestId("knowledge-entity-detail")).not.toBeInTheDocument();
  });

  it("sanitizes a stale current artifact without falling back to selected-run evidence", async () => {
    vi.stubGlobal("fetch", createFetchMock({
      runStarted: true,
      knowledgeResponse: { version: 1, generated_at: "2026-07-15T00:00:00Z", source_mode: "promoted_current", status: "unavailable", entities: [], edges: [], artifacts: [], issues: [] },
    }));
    await renderConsoleApp("/changes?view=evidence&source=current&artifact=reports%2Fmissing.md&mode=raw");
    expect(await screen.findByTestId("route-notice")).toHaveTextContent("reports/missing.md is unavailable in the current workspace");
    await waitFor(() => expect(window.location.search).not.toContain("artifact="));
    expect(screen.getByTestId("current-workspace-evidence")).toHaveTextContent("No historical run snapshot will be substituted");
    expect(screen.queryByText(/Run snapshot/)).not.toBeInTheDocument();
  });

  it("restores a legacy artifact path and raw viewer mode from a direct Changes URL", async () => {
    vi.stubGlobal("fetch", createFetchMock({
      runStarted: true,
      runArtifacts: {
        "run-1": {
          run_id: "run-1",
          artifacts: [{ path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" }],
        },
      },
      artifactText: { "reports/as-is/overview.md": "# Direct evidence\n" },
    }));

    await renderConsoleApp("/changes?run=run-1&view=evidence&source=snapshot&artifact=reports%2Fas-is%2Foverview.md&mode=raw");

    expect(await screen.findByTestId("evidence-raw")).toHaveTextContent("Direct evidence");
    expect(window.location.search).toContain("mode=raw");
    expect(window.location.search).toContain("artifact=reports%2Fas-is%2Foverview.md");
  });

  it("warns before leaving Setup with an unsaved workspace draft", async () => {
    vi.stubGlobal("fetch", createFetchMock());
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
    await renderConsoleApp("/setup?step=sources");

    fireEvent.change(await screen.findByLabelText("workspace.yaml content"), { target: { value: "version: 1\nrepos: []\n" } });
    fireEvent.click(screen.getByRole("link", { name: "Runs" }));

    expect(confirm).toHaveBeenCalledWith(expect.stringContaining("Unsaved workspace or editor changes"));
    expect(window.location.pathname).toBe("/setup");
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

    await renderConsoleApp("/changes");

    expect(await screen.findByTestId("review-panel")).toBeInTheDocument();
    expect(window.location.search).toContain("run=run-1");
    expect(screen.getByTestId("stage-review")).toHaveAttribute("aria-current", "page");
    expect(screen.queryByTestId("workspace-panel")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Changes" })).toHaveAttribute("aria-current", "page");
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

    await renderConsoleApp(`/changes?run=${failedRunID}&view=overview&source=snapshot&mode=rendered`);

    const recovery = await screen.findByTestId("review-run-recovery");
    expect(recovery).toHaveTextContent(successfulRunID);
    expect(screen.getByTestId("review-artifact-explorer")).not.toHaveTextContent("reports/diagrams");

    fireEvent.click(within(recovery).getByRole("button", { name: /open last successful artifacts/i }));

    await waitFor(() => expect(screen.queryByTestId("review-run-recovery")).not.toBeInTheDocument());
    const explorer = screen.getByTestId("review-artifact-explorer");
    expect(explorer).toHaveTextContent("reports/diagrams");
    await waitFor(() => expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("Complete overview"));
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

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");

    const explorer = await screen.findByTestId("review-artifact-explorer");
    expect(screen.getByTestId("review-evidence-preview")).toHaveTextContent("Evidence preview");
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
    expect(citationCoverage).toHaveTextContent("0");
    expect(screen.getByTestId("review-decision-summary")).toHaveTextContent("after human confirmation");

    await waitFor(() => expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("System overview"));
    await waitFor(() => expect(within(explorer).getByRole("button", { name: /reports\/as-is\/overview\.md/i })).toHaveClass("is-selected"));

    fireEvent.click(within(reviewQueue).getByRole("button", { name: /review queue item: review coverage summary/i }));
    await waitFor(() => expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("reports/coverage/summary.md"));

    fireEvent.click(within(explorer).getByRole("button", { name: /reports\/as-is\/overview\.md/i }));
    await waitFor(() => expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("System overview"));

    fireEvent.click(screen.getByTestId("review-domain-map-toggle"));
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
    await waitFor(() => expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("id: svc.payments"));

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
          "reports/taskruns/run-1/staging/final/reports/as-is/overview.md": "# As-is overview\n",
          "reports/taskruns/run-1/staging/final/reports/diagrams/c4-context.mmd": "flowchart LR\n  A --> B\n",
          "reports/taskruns/run-1/staging/final/proposals/proposal-payments/proposal.md": "# Payments proposal\n",
          "reports/taskruns/run-1/staging/final/reports/changelog/2026-04-03.md": "# Iteration changelog\n",
          "reports/taskruns/run-1/staging/final/final-run-index.json": JSON.stringify({
            version: 1,
            run_id: "run-1",
            pipeline: "init",
            generated_at: "2026-04-03T12:00:00Z",
            canonical_documents: [
              {
                id: "doc.overview",
                kind: "report",
                title: "As-is overview",
                canonical_path: "reports/as-is/overview.md",
                staged_path: "reports/taskruns/run-1/staging/final/reports/as-is/overview.md",
              },
              {
                id: "doc.diagram",
                kind: "diagram",
                title: "C4 context",
                canonical_path: "reports/diagrams/c4-context.mmd",
                staged_path: "reports/taskruns/run-1/staging/final/reports/diagrams/c4-context.mmd",
              },
              {
                id: "doc.proposal",
                kind: "proposal",
                title: "Payments proposal",
                canonical_path: "proposals/proposal-payments/proposal.md",
                staged_path: "reports/taskruns/run-1/staging/final/proposals/proposal-payments/proposal.md",
              },
              {
                id: "doc.changelog",
                kind: "changelog",
                title: "Iteration changelog",
                canonical_path: "reports/changelog/2026-04-03.md",
                staged_path: "reports/taskruns/run-1/staging/final/reports/changelog/2026-04-03.md",
              },
            ],
          }),
        },
      }),
    );

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");

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

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");

    await screen.findByTestId("review-panel");
    fireEvent.click(screen.getByTestId("review-domain-map-toggle"));

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

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");

    await screen.findByTestId("review-panel");
    fireEvent.click(screen.getByTestId("review-domain-map-toggle"));

    const edgeList = screen.getByTestId("review-domain-map-edge-list");
    const inspector = screen.getByTestId("review-domain-map-inspector");
    expect(edgeList).toHaveTextContent("svc.payments");
    expect(edgeList).toHaveTextContent("svc.users");
    expect(inspector).toHaveTextContent("Proposal artifacts ready");

    fireEvent.click(within(edgeList).getByRole("button", { name: /model\/edges\/edge\.svc\.payments\.calls\.svc\.users\.yaml/i }));
    await waitFor(() => expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("type: calls"));

    fireEvent.click(screen.getByTestId("review-domain-map-toggle"));
    const refreshedInspector = screen.getByTestId("review-domain-map-inspector");
    fireEvent.click(within(refreshedInspector).getByRole("button", { name: /proposals\/proposal-payments\/proposal\.md/i }));
    await waitFor(() => expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("Proposal"));
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

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");

    await screen.findByTestId("review-panel");
    navigateToStage("proposals");

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
    await waitFor(() => expect(screen.getByTestId("proposal-preview-panel")).toHaveTextContent("Workspace Git diff loaded."));
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

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");

    await screen.findByTestId("review-panel");
    navigateToStage("proposals");

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

  it("preserves the selected snapshot artifact when returning from Proposals", async () => {
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

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");

    await screen.findByTestId("review-panel");
    await waitFor(() => expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("reports/as-is/overview.md"));

    navigateToStage("proposals");
    await waitFor(() => expect(screen.getByTestId("proposal-preview-panel")).toHaveTextContent("# Proposal"));

    navigateToStage("review");
    await waitFor(() => expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("proposals/proposal-payments/proposal.md"));
    expect(screen.getByTestId("evidence-viewer")).toHaveTextContent("Proposal body");
    fireEvent.click(screen.getByTestId("review-artifact-explorer-toggle"));
    expect(within(screen.getByTestId("review-artifact-explorer")).getByRole("button", { name: /proposals\/proposal-payments\/proposal\.md/i })).toHaveClass("is-selected");
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

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");
    await screen.findByTestId("review-panel");

    navigateToStage("publish");

    await waitFor(() => expect(screen.getByTestId("stage-publish")).toHaveAttribute("aria-current", "page"));
    await waitFor(() => expect(screen.getByTestId("publish-panel")).toBeInTheDocument());
    await waitFor(() => expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("reports/coverage"));
    expect(screen.getByTestId("publish-section-jumps")).toHaveTextContent("Diff");
    expect(screen.getByTestId("publish-section-jumps")).toHaveTextContent("Preview");
    expect(screen.getByTestId("publish-section-jumps")).toHaveTextContent("Gate");
    expect(screen.getByTestId("publish-section-jumps")).toHaveTextContent("Commit");
    expect(screen.getByTestId("publish-section-jumps").querySelector('a[href="#publish-gate-panel"]')).toBeInTheDocument();
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("Publication set");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("5 refs");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("review");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("No loaded open questions");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("Git action");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("ready");
    expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("Selected run Git diff");
    expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("model");
    expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("proposals");
    expect(screen.getByTestId("publish-gate-panel")).toHaveTextContent("No hard blockers");
    expect(screen.getByTestId("publish-hard-blockers")).toHaveTextContent("No hard blockers");
    expect(screen.getByTestId("publish-open-questions")).toHaveTextContent("No open questions");
    expect(screen.getByTestId("publish-ready-checks")).toHaveTextContent("Artifacts");
    expect(screen.getByTestId("publish-commit-plan")).toHaveTextContent("proposal/beta-refresh");
    expect(screen.getByTestId("publish-commit-selected-btn")).toBeInTheDocument();
    const proposalBranchButton = screen.getByTestId("git-proposal-branch-btn").closest("button") as HTMLButtonElement;
    expect(proposalBranchButton).not.toBeDisabled();

    fireEvent.click(screen.getByRole("button", { name: /Coverage summary.*reports\/coverage\/summary\.md/i }));
    await waitFor(() => expect(screen.getByTestId("publish-panel")).toHaveTextContent("Coverage ready for publication."));

    fireEvent.click(screen.getByRole("tab", { name: "Diff" }));
    await waitFor(() => expect(screen.getByTestId("publish-panel")).toHaveTextContent("Selected run Git diff"));
    await waitFor(() => expect(screen.getByTestId("git-diff-view")).toHaveTextContent("Workspace Git diff loaded."));
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
    fireEvent.click(await screen.findByRole("button", { name: "Commit all workspace changes" }));
    await waitFor(() => expect(screen.getByTestId("publish-commit-plan")).toHaveTextContent("committed: docs: publish architecture workspace"));

    fireEvent.click(proposalBranchButton);
    fireEvent.click(await screen.findByRole("button", { name: "Create proposal branch" }));
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

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");
    await screen.findByTestId("review-panel");

    navigateToStage("publish");
    await screen.findByTestId("publish-panel");
    await waitFor(() => expect(screen.getByTestId("stage-publish")).toHaveAttribute("aria-current", "page"));
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
    await waitFor(() => expect(publishArtifactList()).toHaveTextContent("reports/taskruns/run-1/staging/final/final-run-index.json"));
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
          "reports/taskruns/run-1/staging/final/reports/as-is/overview.md": "# As-is overview\n",
          "reports/taskruns/run-1/staging/final/proposals/proposal-baseline/proposal.md": "# Proposal\n",
          "reports/taskruns/run-1/staging/final/reports/changelog/2026-05-31-run-1.md": "# Run changelog\n- Proposal package compiled.\n",
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
              {
                id: "doc.proposal-baseline",
                kind: "proposal",
                title: "Baseline proposal",
                canonical_path: "proposals/proposal-baseline/proposal.md",
                staged_path: "reports/taskruns/run-1/staging/final/proposals/proposal-baseline/proposal.md",
              },
              {
                id: "doc.run-changelog",
                kind: "changelog",
                title: "Run changelog",
                canonical_path: "reports/changelog/2026-05-31-run-1.md",
                staged_path: "reports/taskruns/run-1/staging/final/reports/changelog/2026-05-31-run-1.md",
              },
            ],
          }),
        },
      }),
    );

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");

    await screen.findByTestId("review-panel");
    navigateToStage("publish");
    expect(await screen.findByTestId("publish-panel")).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByTestId("publish-preview-panel")).toHaveTextContent("reports/as-is/overview.md");
      expect(screen.getByTestId("publish-selected-preview-content")).toHaveTextContent("# As-is overview");
    });

    navigateToStage("proposals");
    const proposalList = await screen.findByTestId("proposals-artifact-list");
    expect(proposalList).toHaveTextContent("proposals/proposal-baseline");
    expect(proposalList).toHaveTextContent("reports/changelog");

    navigateToStage("publish");
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

    navigateToStage("publish");

    expect(await screen.findByTestId("publish-panel")).toBeInTheDocument();
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("0 refs");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("blocked");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("Run analysis before publishing workspace artifacts.");
    expect(screen.getByTestId("publish-gate-panel")).toHaveTextContent("Run analysis before publishing workspace artifacts.");
    expect(screen.getByTestId("publish-gate-panel")).toHaveTextContent("Run analysis before publishing workspace artifacts");
    expect(screen.getByTestId("publish-commit-selected-btn")).toBeDisabled();
    const proposalBranchButton = screen.getByTestId("git-proposal-branch-btn").closest("button") as HTMLButtonElement;
    expect(proposalBranchButton).toBeDisabled();

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

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");
    await screen.findByTestId("review-panel");

    navigateToStage("publish");

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

  it("surfaces failed Publish Git actions in the commit plan and inspector", async () => {
    const fetchMock = createFetchMock({
      runStarted: true,
      runArtifacts: {
        "run-1": {
          run_id: "run-1",
          artifacts: [{ path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" }],
        },
      },
      artifactText: {
        "reports/coverage/summary.md": "Coverage ready for publication.\n",
        "reports/coverage/open-questions.md": "",
      },
      gitCommitResponse: {
        status: 409,
        body: { error: { code: "git_commit_failed", message: "workspace has unresolved merge conflicts" } },
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");
    await screen.findByTestId("review-panel");

    navigateToStage("publish");
    expect(await screen.findByTestId("publish-panel")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("publish-diff-summary")).toHaveTextContent("reports/coverage"));

    fireEvent.click(screen.getByTestId("publish-commit-selected-btn"));
    fireEvent.click(await screen.findByRole("button", { name: "Commit all workspace changes" }));

    const recovery = await screen.findByTestId("publish-git-action-recovery");
    expect(recovery).toHaveTextContent("Git action failed");
    expect(recovery).toHaveTextContent("Git mutation failed: workspace has unresolved merge conflicts");
    expect(recovery).toHaveTextContent("Workspace Git state was not changed");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("failed");
    expect(screen.getByTestId("publish-readiness-summary")).toHaveTextContent("Git mutation failed");
    expect(screen.getByTestId("publish-commit-plan")).toHaveTextContent("failed");
    expect(screen.getByTestId("publish-commit-selected-btn")).not.toBeDisabled();
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

    await renderConsoleApp("/changes?run=run-1&view=overview&source=snapshot&mode=rendered");

    expect(await screen.findByTestId("review-panel")).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole("button", { name: /reports\/diagrams\/c4-context\.mmd/i })[0]);

    await waitFor(() => expect(mermaid.default.render).toHaveBeenCalledWith(expect.stringMatching(/^diagram-/), "flowchart LR\n  A --> B"));
    expect(mermaid.default.render).not.toHaveBeenCalledWith(expect.any(String), "Loading...");
  });

  it("starts agent-backed architecture Q&A and renders answer, citations, unresolved, and confidence", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    navigateToStage("ask");
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
    expect(fetchMock).toHaveBeenCalledWith("/api/qa/runs/qa-run-1", expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it("creates an explicit Ask proposal draft and routes to current Changes proposals with return context", async () => {
    const proposalPath = "proposals/qa-synthesis-qa-run-1-who-owns-payments/proposal.md";
    const fetchMock = createFetchMock({
      artifactText: {
        [proposalPath]: "# Who owns payments?\n\n## Evidence\n",
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("ask");
    fireEvent.change(await screen.findByTestId("qa-question-input"), { target: { value: "Who owns payments?" } });
    fireEvent.click(screen.getByTestId("qa-ask-btn"));
    expect(await screen.findByTestId("qa-create-proposal-btn")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("qa-create-proposal-btn"));
    const dialog = await screen.findByRole("dialog", { name: "Create proposal draft" });
    expect(dialog).toHaveTextContent("Ask remains read-only");
    expect(dialog).toHaveTextContent("Citations: 1");
    fireEvent.click(within(dialog).getByRole("button", { name: "Create proposal draft" }));

    expect(await screen.findByTestId("changes-route-proposals")).toBeInTheDocument();
    expect(screen.getByText("Return to Ask")).toBeInTheDocument();
    expect(window.location.search).toContain("view=proposals");
    expect(window.location.search).toContain("source=current");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/qa/runs/qa-run-1/proposal-draft",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining(`"expected_answer_digest":"${"a".repeat(64)}"`),
      }),
    );

    fireEvent.click(screen.getByText("Return to Ask"));
    expect(await screen.findByTestId("qa-panel")).toBeInTheDocument();
  });

  it("keeps Ask confirmation open on a stale proposal digest and offers answer reload", async () => {
    vi.stubGlobal("fetch", createFetchMock({
      qaProposalResponse: {
        status: 409,
        body: { error: { code: "qa_answer_stale", message: "qa answer digest is stale" } },
      },
    }));

    await renderConsoleApp();
    navigateToStage("ask");
    fireEvent.change(await screen.findByTestId("qa-question-input"), { target: { value: "Who owns payments?" } });
    fireEvent.click(screen.getByTestId("qa-ask-btn"));
    fireEvent.click(await screen.findByTestId("qa-create-proposal-btn"));
    const dialog = await screen.findByRole("dialog", { name: "Create proposal draft" });
    fireEvent.click(within(dialog).getByRole("button", { name: "Create proposal draft" }));

    expect(await screen.findByText("qa answer digest is stale")).toBeInTheDocument();
    expect(screen.getByRole("dialog", { name: "Create proposal draft" })).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Reload selected answer" })).toBeInTheDocument();
  });

  it("keeps an accepted Q&A run selected when the first detail GET fails and later recovers", async () => {
    const runID = "qa-start-accepted";
    let startCalls = 0;
    let detailCalls = 0;
    const baseFetch = createFetchMock();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/qa/runs?limit=20") {
        return jsonResponse({ items: [] });
      }
      if (method === "POST" && url === "/api/qa/runs") {
        startCalls += 1;
        return jsonResponse({ run_id: runID, status: "queued" }, 202);
      }
      if (method === "GET" && url === `/api/qa/runs/${runID}`) {
        detailCalls += 1;
        if (detailCalls === 1) {
          return jsonResponse({ error: { code: "temporary", message: "qa detail temporarily unavailable" } }, 503);
        }
        return jsonResponse({
          run_id: runID,
          pipeline: "qa",
          status: "succeeded",
          started_at: "2026-04-03T12:00:03Z",
          finished_at: "2026-04-03T12:00:04Z",
          question: "Who owns payments?",
          current_step: "qa.ask",
          runtime_provider: "claude-code",
          provider: "fake",
          answer: "Recovered Q&A answer for payments ownership.",
          citations: [{ path: "reports/as-is/overview.md", reason: "ownership evidence" }],
          unresolved: [],
          confidence: 0.87,
          generated_at: "2026-04-03T12:00:04Z",
        });
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("ask");
    fireEvent.change(await screen.findByTestId("qa-question-input"), { target: { value: "Who owns payments?" } });
    fireEvent.click(screen.getByTestId("qa-ask-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("qa-run-status")).toHaveTextContent(runID);
      expect(screen.getByTestId("qa-run-status")).toHaveTextContent("queued");
    });
    expect(screen.getByTestId("qa-ask-btn")).toBeDisabled();
    expect(await screen.findByText(`Q&A run ${runID} accepted; reconciling details failed: qa detail temporarily unavailable`)).toBeInTheDocument();
    expect(startCalls).toBe(1);

    await waitFor(() => expect(screen.getByTestId("qa-answer")).toHaveTextContent("Recovered Q&A answer for payments ownership."), { timeout: 4000 });
    expect(screen.getByTestId("qa-run-status")).toHaveTextContent("succeeded");
    expect(screen.getByTestId("qa-answer-panel")).toHaveTextContent("Confidence: 87%");
    expect(startCalls).toBe(1);
  }, 10_000);

  it("keeps the accepted Q&A run selected when an older history response resolves late", async () => {
    const oldRun = {
      run_id: "qa-old-history",
      pipeline: "qa",
      status: "succeeded",
      started_at: "2026-04-03T11:00:00Z",
      finished_at: "2026-04-03T11:00:01Z",
      question: "Old history question",
      current_step: "qa.ask",
      provider: "fake",
      answer: "Old history answer SHOULD NOT SELECT",
      citations: [],
      unresolved: [],
      confidence: 0.5,
      generated_at: "2026-04-03T11:00:01Z",
    };
    const newRunID = "qa-new-history";
    const lateHistory = deferredResponse();
    let historyCalls = 0;
    const baseFetch = createFetchMock();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/qa/runs?limit=20") {
        historyCalls += 1;
        if (historyCalls === 1) {
          return lateHistory.promise;
        }
        return jsonResponse({ items: [oldRun] });
      }
      if (method === "POST" && url === "/api/qa/runs") {
        return jsonResponse({ run_id: newRunID, status: "queued" }, 202);
      }
      if (method === "GET" && url === `/api/qa/runs/${newRunID}`) {
        return jsonResponse({
          run_id: newRunID,
          pipeline: "qa",
          status: "succeeded",
          started_at: "2026-04-03T12:00:03Z",
          finished_at: "2026-04-03T12:00:04Z",
          question: "New history question",
          current_step: "qa.ask",
          provider: "fake",
          answer: "New accepted answer remains selected.",
          citations: [],
          unresolved: [],
          confidence: 0.93,
          generated_at: "2026-04-03T12:00:04Z",
        });
      }
      if (method === "GET" && url === `/api/qa/runs/${oldRun.run_id}`) {
        return jsonResponse(oldRun);
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("ask");
    fireEvent.change(await screen.findByTestId("qa-question-input"), { target: { value: "New history question" } });
    fireEvent.click(screen.getByTestId("qa-ask-btn"));

    expect(await screen.findByTestId("qa-answer")).toHaveTextContent("New accepted answer remains selected.");
    lateHistory.resolve(jsonResponse({ items: [oldRun] }));

    await waitFor(() => {
      expect(screen.getByTestId("qa-run-status")).toHaveTextContent(newRunID);
      expect(screen.getByTestId("qa-answer")).toHaveTextContent("New accepted answer remains selected.");
      expect(screen.getByTestId("qa-answer")).not.toHaveTextContent("SHOULD NOT SELECT");
      expect(screen.getByTestId("qa-run-history")).toHaveTextContent("New history question");
    });
  }, 10_000);

  it("ignores a late Q&A detail response after a newer history run is selected", async () => {
    const lateOldDetail = deferredResponse();
    const oldRun = {
      run_id: "qa-old-detail",
      pipeline: "qa",
      status: "succeeded",
      started_at: "2026-04-03T11:00:00Z",
      finished_at: "2026-04-03T11:00:01Z",
      question: "Old detail question",
      current_step: "qa.ask",
      provider: "fake",
      answer: null,
      citations: [],
      unresolved: [],
      confidence: 0.1,
      generated_at: "2026-04-03T11:00:01Z",
    };
    const newRun = {
      run_id: "qa-new-detail",
      pipeline: "qa",
      status: "succeeded",
      started_at: "2026-04-03T12:00:00Z",
      finished_at: "2026-04-03T12:00:01Z",
      question: "New detail question",
      current_step: "qa.ask",
      provider: "fake",
      answer: "New selected detail remains visible.",
      citations: [],
      unresolved: [],
      confidence: 0.9,
      generated_at: "2026-04-03T12:00:01Z",
    };
    let oldDetailRequested = false;
    const baseFetch = createFetchMock();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/qa/runs?limit=20") {
        return jsonResponse({ items: [oldRun, newRun] });
      }
      if (method === "GET" && url === `/api/qa/runs/${oldRun.run_id}`) {
        oldDetailRequested = true;
        return lateOldDetail.promise;
      }
      if (method === "GET" && url === `/api/qa/runs/${newRun.run_id}`) {
        return jsonResponse(newRun);
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("ask");
    await waitFor(() => expect(oldDetailRequested).toBe(true));

    fireEvent.click(await screen.findByRole("button", { name: /New detail question/i }));

    await waitFor(() => {
      expect(screen.getByTestId("qa-run-status")).toHaveTextContent(newRun.run_id);
      expect(screen.getByTestId("qa-answer")).toHaveTextContent("New selected detail remains visible.");
    });

    lateOldDetail.resolve(
      jsonResponse({
        ...oldRun,
        answer: "Old delayed detail SHOULD NOT SHOW",
        confidence: 0.99,
      }),
    );

    await waitFor(() => {
      expect(screen.getByTestId("qa-run-status")).toHaveTextContent(newRun.run_id);
      expect(screen.getByTestId("qa-answer")).toHaveTextContent("New selected detail remains visible.");
      expect(screen.getByTestId("qa-answer")).not.toHaveTextContent("SHOULD NOT SHOW");
    });
  }, 10_000);

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

    navigateToStage("ask");
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

    navigateToStage("ask");
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

  it("submits Ask from the global utility flow", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    navigateToStage("ask");
    fireEvent.change(await screen.findByTestId("qa-question-input"), { target: { value: "Who owns payments?" } });
    fireEvent.click(screen.getByTestId("qa-ask-btn"));

    expect(await screen.findByTestId("qa-answer")).toHaveTextContent("payments-service is owned by Platform Architecture.");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/qa/runs",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ question: "Who owns payments?" }),
      }),
    );
  });

  it("keeps global Ask in an accessible read-only modal and returns focus on Escape", async () => {
    vi.stubGlobal("fetch", createFetchMock());
    await renderConsoleApp("/home");
    const askButton = screen.getByTestId("stage-ask");
    askButton.focus();
    fireEvent.click(askButton);
    const dialog = await screen.findByRole("dialog", { name: "Ask current workspace" });
    expect(dialog).toHaveTextContent("Current workspace · read-only");
    fireEvent.keyDown(dialog, { key: "Escape" });
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Ask current workspace" })).not.toBeInTheDocument());
    expect(askButton).toHaveFocus();
  });

  it("opens an Ask citation as current workspace evidence and restores Ask context", async () => {
    const historyRun = {
      run_id: "qa-citation-1", pipeline: "qa", status: "succeeded", started_at: "2026-07-15T00:00:00Z", finished_at: "2026-07-15T00:00:01Z",
      question: "Who owns payments?", answer: "Platform owns payments.", citations: [{ path: "model/entities/svc.payments.yaml", reason: "owner record" }], unresolved: [], confidence: 0.9,
    };
    vi.stubGlobal("fetch", createFetchMock({
      qaRuns: [historyRun],
      qaRunResponses: { "qa-citation-1": historyRun },
      knowledgeResponse: {
        version: 1, generated_at: "2026-07-15T00:00:00Z", source_mode: "promoted_current", status: "available",
        entities: [{ id: "svc.payments", type: "service", name: "Payments", path: "model/entities/svc.payments.yaml", provenance: { kind: "inference", confidence: 0.9 } }],
        edges: [], artifacts: [{ path: "model/entities/svc.payments.yaml", kind: "entity", name: "svc.payments.yaml" }], issues: [],
      },
      artifactText: { "model/entities/svc.payments.yaml": "# Payments\nOwner: Platform" },
    }));
    await renderConsoleApp("/home");
    navigateToStage("ask");
    fireEvent.click(await screen.findByRole("button", { name: /model\/entities\/svc\.payments\.yaml/i }));
    expect(await screen.findByTestId("current-workspace-evidence")).toHaveTextContent("Current workspace");
    expect(window.location.search).toContain("source=current");
    fireEvent.click(screen.getByRole("button", { name: "Return to Ask" }));
    expect(await screen.findByRole("dialog", { name: "Ask current workspace" })).toHaveTextContent("Platform owns payments.");
    expect(window.location.pathname).toBe("/home");
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

    navigateToStage("ask");

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

    navigateToStage("ask");
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

    expect(screen.getByTestId("stage-source")).toHaveAttribute("aria-current", "page");
    expect(screen.getByDisplayValue("https://github.com/org/my-service.git")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("workspace-save-btn"));

    await screen.findByTestId("workspace-validate-result");
    navigateToStage("readiness");
    expect(screen.getByTestId("setup-run-first-btn")).toBeDisabled();
    expect(screen.getByTestId("readiness-panel")).toHaveTextContent("Check local readiness");

    fireEvent.click(screen.getByTestId("setup-doctor-btn"));
    await screen.findByTestId("setup-doctor-result");
    expect(screen.getByText("Local readiness passed.")).toBeInTheDocument();
    expect(screen.getByTestId("setup-run-first-btn")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("setup-run-first-btn"));
    expect(await screen.findByRole("dialog", { name: "Start without a saved analysis brief?" })).toHaveTextContent("reduces evidence quality");
    fireEvent.click(screen.getByRole("button", { name: "Start with quality warning" }));
    await screen.findByTestId("analysis-run-progress");
    expect(screen.getByTestId("destination-runs")).toHaveAttribute("aria-current", "page");
    expect(fetchMock).toHaveBeenCalledWith("/api/pipeline/init", expect.anything());
  });

  it("starts from Guided Setup review without a warning after saving the analysis brief", async () => {
    const fetchMock = createFetchMock({ runID: "run-with-brief" });
    vi.stubGlobal("fetch", fetchMock);
    await renderConsoleApp("/setup?step=brief");

    fireEvent.change(await screen.findByLabelText("Project name"), { target: { value: "Payments architecture" } });
    fireEvent.change(screen.getByLabelText("Scope"), { target: { value: "payments and ledger boundaries" } });
    fireEvent.click(screen.getByRole("button", { name: "Save Step 0 wizard contract" }));
    expect(await screen.findByText("Saved charter/wizard/step0-contract.json")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("setup-step-review"));
    expect(await screen.findByTestId("guided-setup-review")).toHaveTextContent("Analysis briefSaved");
    fireEvent.click(screen.getByTestId("guided-start-analysis"));

    expect(await screen.findByTestId("analysis-run-progress")).toBeInTheDocument();
    expect(screen.queryByRole("dialog", { name: "Start without a saved analysis brief?" })).not.toBeInTheDocument();
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

    navigateToStage("readiness");
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
    navigateToStage("readiness");
    expect(screen.getByTestId("setup-run-first-btn")).toBeDisabled();
    fireEvent.click(screen.getByTestId("setup-doctor-btn"));
    await screen.findByTestId("setup-doctor-result");
    expect(screen.getByTestId("setup-run-first-btn")).not.toBeDisabled();

    navigateToStage("source");
    fireEvent.change(screen.getByLabelText("Repository URL"), { target: { value: "https://github.com/org/changed.git" } });

    expect(screen.queryByTestId("workspace-validate-result")).not.toBeInTheDocument();
    navigateToStage("readiness");
    expect(screen.getByTestId("setup-run-first-btn")).toBeDisabled();
  });

  it("clears readiness checklist after runtime selection changes", async () => {
    vi.stubGlobal("fetch", createFetchMock());

    await renderConsoleApp();

    navigateToStage("readiness");
    fireEvent.click(screen.getByTestId("setup-doctor-btn"));
    await screen.findByTestId("setup-doctor-result");
    expect(screen.getByText("Local readiness passed.")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Runtime mode"), { target: { value: "headless" } });

    expect(screen.queryByTestId("setup-doctor-result")).not.toBeInTheDocument();
    expect(screen.queryByText("Local readiness passed.")).not.toBeInTheDocument();
  });

  it("runs init from Runs and exposes progress plus step diff", async () => {
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

    navigateToStage("analysis");
    fireEvent.click(screen.getByTestId("run-init-btn"));

    const activeRunProgress = await screen.findByTestId("analysis-run-progress");
    expect(activeRunProgress).toHaveTextContent(runID);

    await screen.findByTestId("run-status-panel");
    expect(await screen.findAllByTestId("analysis-step-review-card")).toHaveLength(5);
    fireEvent.click(screen.getByTestId("analysis-step-tab-diff"));
    await waitFor(() => expect(screen.getByTestId("git-diff-view")).toHaveTextContent("Workspace Git diff loaded."));

  });

  it("keeps an accepted start run selected when the first detail GET fails and later recovers", async () => {
    const runID = "run-start-accepted";
    let startCalls = 0;
    let statusCalls = 0;
    const runListItem = {
      run_id: runID,
      pipeline: "init",
      status: "running",
      started_at: "2026-04-03T12:00:00Z",
      finished_at: null,
      warnings: [],
      error_code: null,
      error: null,
    };
    const recoveredStatus = {
      ...runListItem,
      current_step: "init.step1.collect",
    };
    const baseFetch = createFetchMock();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({ items: startCalls > 0 ? [runListItem] : [] });
      }
      if (method === "POST" && url === "/api/pipeline/init") {
        startCalls += 1;
        return jsonResponse({ run_id: runID, status: "queued" });
      }
      if (method === "GET" && url === `/api/pipeline/runs/${runID}`) {
        statusCalls += 1;
        if (statusCalls === 1) {
          return jsonResponse({ error: { code: "temporary", message: "status temporarily unavailable" } }, 503);
        }
        return jsonResponse(recoveredStatus);
      }
      if (method === "GET" && url === `/api/pipeline/runs/${runID}/artifacts`) {
        return jsonResponse({ run_id: runID, artifacts: [] });
      }
      if (method === "GET" && url.startsWith(`/api/pipeline/runs/${runID}/logs?`)) {
        return jsonResponse({ run_id: runID, items: [], next_cursor: 0, eof: true });
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("analysis");
    fireEvent.click(screen.getByTestId("run-init-btn"));

    await waitFor(() => expect(screen.getByTestId("run-status-run-id").textContent).toBe(runID));
    expect(screen.getByTestId("run-status-value")).toHaveTextContent("queued");
    expect(await screen.findByText(`Run ${runID} accepted; reconciling details failed: status temporarily unavailable`)).toBeInTheDocument();
    expect(startCalls).toBe(1);

    await waitFor(() => expect(screen.getByTestId("run-status-value")).toHaveTextContent("running"), { timeout: 4000 });
    expect(screen.getByTestId("run-status-panel")).toHaveTextContent("Current step: init.step1.collect");
    expect(startCalls).toBe(1);
  }, 10_000);

  it("disables ordinary starts and confirms explicit refresh queue replacement", async () => {
    const activeID = "run-active";
    const pendingID = "run-pending";
    const replacementID = "run-replacement";
    const active = {
      run_id: activeID,
      pipeline: "init",
      status: "running",
      started_at: "2026-07-15T10:00:00Z",
      warnings: [],
    };
    const pending = {
      run_id: pendingID,
      pipeline: "refresh",
      status: "queued",
      started_at: "2026-07-15T10:01:00Z",
      warnings: [],
    };
    const baseFetch = createFetchMock({ runID: activeID, runStarted: true, runStatus: { [activeID]: active } });
    let queued = false;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      if (method === "GET" && url === "/api/pipeline/runs?limit=100") {
        return jsonResponse({
          coordination: { active_run_id: activeID, pending: { run_id: queued ? replacementID : pendingID, pipeline: "refresh" } },
          items: queued
            ? [active, { ...pending, run_id: replacementID }, { ...pending, status: "canceled", error_code: "run_superseded", superseded_by_run_id: replacementID }]
            : [active, pending],
        });
      }
      if (method === "POST" && url === "/api/pipeline/refresh") {
        expect(JSON.parse(String(init?.body))).toMatchObject({ intent: "queue" });
        queued = true;
        return jsonResponse({ run_id: replacementID, status: "started" }, 202);
      }
      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("analysis");
    expect(screen.getByTestId("run-init-btn")).toBeDisabled();
    expect(screen.getByTestId("run-refresh-btn")).toBeDisabled();
    expect(screen.getByTestId("pending-run-summary")).toHaveTextContent(pendingID);

    fireEvent.click(screen.getByTestId("run-queue-refresh-btn"));
    expect(await screen.findByRole("dialog")).toHaveTextContent(`pending ${pendingID} will be canceled as run_superseded`);
    fireEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Replace pending refresh" }));

    expect(await screen.findByText(`Refresh ${replacementID} queued; the selected evidence remains unchanged.`)).toBeInTheDocument();
    expect(screen.getByTestId("pending-run-summary")).toHaveTextContent(replacementID);
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
    navigateToStage("analysis");

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
    expect(permissionRecovery).toHaveTextContent("npm install");
    expect(permissionRecovery).toHaveTextContent("package install requires review");
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

    await renderConsoleApp(`/runs/${runID}`);
    await waitFor(() => expect(screen.getByTestId("destination-runs")).toHaveAttribute("aria-current", "page"));

    await waitFor(
      () => expect(screen.getByTestId("analysis-run-progress")).toHaveTextContent(runID),
      { timeout: 5_000 },
    );
    const progress = screen.getByTestId("analysis-run-progress");
    expect(progress).toHaveTextContent("failed");
    expect(progress).toHaveTextContent("init.step1.collect");
    expect(progress).toHaveTextContent("4/5");
    expect(within(progress).getByTestId("analysis-review-blocker-btn")).not.toBeDisabled();

    const timeline = screen.getByTestId("analysis-run-timeline");
    expect(timeline).toHaveTextContent("init.step0.constitution");
    expect(timeline).toHaveTextContent("init.step1.collect");
    expect(timeline).toHaveTextContent("blocked");

    const shardTable = await screen.findByTestId("analysis-shard-table");
    expect(shardTable).toHaveTextContent("payments-root-shard");
    expect(shardTable).toHaveTextContent("invoices-module-shard");
    expect(shardTable).toHaveTextContent("qwen-code");
    expect(shardTable).toHaveTextContent("failed");
    expect(shardTable).toHaveTextContent("runtime-execution.json");
    expect(shardTable).toHaveTextContent("Runtime only");
    expect(shardTable).toHaveTextContent("authored markdown and shard-pack-manifest are missing");
    expect(shardTable).toHaveTextContent("No shard artifacts");
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
  }, 15_000);

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
    await screen.findByTestId("product-shell");

    navigateToStage("analysis");
    await waitFor(() => expect(screen.getByTestId("destination-runs")).toHaveAttribute("aria-current", "page"));

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
    navigateToStage("analysis");
    await screen.findByTestId("run-status-panel");
    fireEvent.click(screen.getByTestId("run-cancel-btn"));
    expect(await screen.findByText(`Cancel requested for ${acceptedRunID}`)).toBeInTheDocument();
  });

  it("keeps an accepted cancel acknowledgement when follow-up reconciliation fails", async () => {
    const runID = "run-cancel-reconcile";
    let cancelCalls = 0;
    let failedListAfterCancel = false;
    const baseFetch = createFetchMock({
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
    });
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "POST" && url === `/api/pipeline/runs/${runID}/cancel`) {
        cancelCalls += 1;
        return jsonResponse({ status: "cancel_requested" }, 202);
      }
      if (cancelCalls > 0 && !failedListAfterCancel && method === "GET" && url === "/api/pipeline/runs?limit=100") {
        failedListAfterCancel = true;
        return jsonResponse({ error: { code: "temporary", message: "run list temporarily unavailable" } }, 503);
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("analysis");
    await screen.findByTestId("run-status-panel");
    fireEvent.click(screen.getByTestId("run-cancel-btn"));

    expect(await screen.findByText(`Cancel requested for ${runID}; reconciling details failed: run list temporarily unavailable`)).toBeInTheDocument();
    expect(cancelCalls).toBe(1);
    expect(screen.queryByText(/^Error:/)).not.toBeInTheDocument();
  }, 10_000);

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
    navigateToStage("analysis");

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
    navigateToStage("analysis");
    await screen.findByTestId("run-status-panel");
    fireEvent.click(screen.getByTestId("run-cancel-btn"));
    expect(await screen.findByText("Selected run is already terminal.")).toBeInTheDocument();
  });

  it("saves and resets runtime timeout, execution, and permission settings", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("readiness");

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

  it("reads historical Review artifacts from selected-run staged paths", async () => {
    const baseFetch = createFetchMock();
    const finalIndexPath = (runID: string) => `reports/taskruns/${runID}/staging/final/final-run-index.json`;
    const stagedOverviewPath = (runID: string) => `reports/taskruns/${runID}/staging/final/reports/as-is/overview.md`;
    const stagedCoveragePath = (runID: string) => `reports/taskruns/${runID}/staging/final/reports/coverage/summary.md`;
    const stagedQuestionsPath = (runID: string) => `reports/taskruns/${runID}/staging/final/reports/coverage/open-questions.md`;
    const finalIndex = (runID: string) => ({
      version: 1,
      run_id: runID,
      pipeline: "refresh",
      generated_at: "2026-04-03T12:02:30Z",
      citation_index_path: `reports/taskruns/${runID}/staging/final/citation-index.json`,
      canonical_documents: [
        {
          id: "doc.overview",
          kind: "report",
          title: `${runID} overview`,
          canonical_path: "reports/as-is/overview.md",
          staged_path: stagedOverviewPath(runID),
        },
        {
          id: "doc.coverage",
          kind: "report",
          title: `${runID} coverage`,
          canonical_path: "reports/coverage/summary.md",
          staged_path: stagedCoveragePath(runID),
        },
        {
          id: "doc.questions",
          kind: "report",
          title: `${runID} questions`,
          canonical_path: "reports/coverage/open-questions.md",
          staged_path: stagedQuestionsPath(runID),
        },
      ],
    });
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

      for (const runID of ["run-old", "run-new"]) {
        if (method === "GET" && url === `/api/pipeline/runs/${runID}`) {
          return jsonResponse({
            run_id: runID,
            pipeline: "refresh",
            status: "succeeded",
            started_at: "2026-04-03T12:02:00Z",
            finished_at: "2026-04-03T12:02:30Z",
            warnings: [],
            error_code: null,
            error: null,
          });
        }
        if (method === "GET" && url === `/api/pipeline/runs/${runID}/artifacts`) {
          const foreignRunID = runID === "run-old" ? "run-new" : "run-old";
          return jsonResponse({
            run_id: runID,
            artifacts: [
              { path: finalIndexPath(foreignRunID), kind: "taskrun", label: "Foreign run index" },
              { path: finalIndexPath(runID), kind: "taskrun", label: "Final run index" },
            ],
          });
        }
        if (method === "GET" && url === `/api/pipeline/runs/${runID}/snapshot`) {
          const index = finalIndex(runID);
          return jsonResponse({
            run_id: runID,
            status: "available",
            issues: [],
            artifacts: [
              ...index.canonical_documents.map((document) => ({
                id: document.id,
                path: document.canonical_path,
                read_path: document.staged_path,
                canonical_path: document.canonical_path,
                kind: document.kind,
                label: document.title,
                source_run_id: runID,
                source_mode: "run_snapshot",
              })),
              {
                path: finalIndexPath(runID),
                read_path: finalIndexPath(runID),
                kind: "taskrun",
                label: "Final run index",
                source_run_id: runID,
                source_mode: "run_snapshot",
              },
            ],
          });
        }
        if (method === "GET" && url.startsWith(`/api/pipeline/runs/${runID}/logs?`)) {
          return jsonResponse({ run_id: runID, items: [], next_cursor: 0, eof: true });
        }
        if (method === "GET" && url === `/api/artifacts?path=${encodeURIComponent(finalIndexPath(runID))}`) {
          return jsonResponse(finalIndex(runID));
        }
        if (method === "GET" && url === `/api/artifacts?path=${encodeURIComponent(stagedOverviewPath(runID))}`) {
          return textResponse(runID === "run-old" ? "# Old snapshot overview\n" : "# New snapshot overview\n");
        }
        if (method === "GET" && url === `/api/artifacts?path=${encodeURIComponent(stagedCoveragePath(runID))}`) {
          return textResponse(runID === "run-old" ? "Old snapshot coverage\n" : "New snapshot coverage\n");
        }
        if (method === "GET" && url === `/api/artifacts?path=${encodeURIComponent(stagedQuestionsPath(runID))}`) {
          return textResponse(runID === "run-old" ? "- Old snapshot question\n" : "- New snapshot question\n");
        }
      }

      if (method === "GET" && url === "/api/artifacts?path=reports%2Fas-is%2Foverview.md") {
        return textResponse("# Current canonical overview SHOULD NOT SHOW\n");
      }
      if (method === "GET" && url === "/api/artifacts?path=reports%2Fcoverage%2Fsummary.md") {
        return textResponse("Current canonical coverage SHOULD NOT SHOW\n");
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    navigateToStage("review");
    await waitFor(() => {
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").toContain("reports/as-is/overview.md");
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").toContain("Old snapshot overview");
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").not.toContain("Current canonical");
      expect(screen.getByTestId("coverage-summary-content").textContent ?? "").toContain("Old snapshot coverage");
    });

    navigateToStage("analysis");
    fireEvent.click(screen.getByTestId("destination-runs"));
    fireEvent.click(screen.getByRole("button", { name: "run-new" }));

    await waitFor(() => {
      expect(screen.getByTestId("run-status-run-id").textContent).toBe("run-new");
      expect(fetchMock).toHaveBeenCalledWith("/api/pipeline/runs/run-new/snapshot", expect.anything());
      expect(fetchMock).toHaveBeenCalledWith(
        `/api/artifacts?path=${encodeURIComponent(stagedOverviewPath("run-new"))}`,
        expect.anything(),
      );
    });

    navigateToStage("review");
    await waitFor(() => {
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").toContain("reports/as-is/overview.md");
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").toContain("New snapshot overview");
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").not.toContain("Old snapshot overview");
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").not.toContain("Current canonical");
      expect(screen.getByTestId("coverage-summary-content").textContent ?? "").toContain("New snapshot coverage");
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
            { path: "reports/taskruns/run-old/staging/final/final-run-index.json", kind: "taskrun", label: "Final run index" },
          ],
        });
      }

      if (method === "GET" && url === "/api/pipeline/runs/run-new/artifacts") {
        return jsonResponse({
          run_id: "run-new",
          artifacts: [
            { path: "reports/as-is/new.md", kind: "report", label: "New artifact" },
            { path: "reports/taskruns/run-new/staging/final/final-run-index.json", kind: "taskrun", label: "Final run index" },
          ],
        });
      }

      for (const [runID, canonicalPath] of [
        ["run-old", "reports/as-is/old.md"],
        ["run-new", "reports/as-is/new.md"],
      ] as const) {
        if (method === "GET" && url === `/api/pipeline/runs/${runID}/snapshot`) {
          const indexPath = `reports/taskruns/${runID}/staging/final/final-run-index.json`;
          const stagedPath = `reports/taskruns/${runID}/staging/final/${canonicalPath}`;
          return jsonResponse({
            run_id: runID,
            status: "available",
            issues: [],
            artifacts: [
              {
                id: `doc.${runID}`,
                path: canonicalPath,
                read_path: stagedPath,
                canonical_path: canonicalPath,
                kind: "report",
                label: canonicalPath,
                source_run_id: runID,
                source_mode: "run_snapshot",
              },
              {
                path: indexPath,
                read_path: indexPath,
                kind: "taskrun",
                label: "Final run index",
                source_run_id: runID,
                source_mode: "run_snapshot",
              },
            ],
          });
        }
      }

      const snapshotArtifact = (runID: string, canonicalPath: string, body: string) => {
        const indexPath = `reports/taskruns/${runID}/staging/final/final-run-index.json`;
        const stagedPath = `reports/taskruns/${runID}/staging/final/${canonicalPath}`;
        if (url === `/api/artifacts?path=${encodeURIComponent(indexPath)}`) {
          return textResponse(JSON.stringify({ version: 1, run_id: runID, pipeline: "refresh", generated_at: "2026-04-03T12:03:00Z", canonical_documents: [{ id: `doc.${runID}`, kind: "report", title: canonicalPath, canonical_path: canonicalPath, staged_path: stagedPath }] }));
        }
        if (url === `/api/artifacts?path=${encodeURIComponent(stagedPath)}`) return textResponse(body);
        return null;
      };
      const oldSnapshot = snapshotArtifact("run-old", "reports/as-is/old.md", "# Old artifact\n");
      if (oldSnapshot) return oldSnapshot;
      const newSnapshot = snapshotArtifact("run-new", "reports/as-is/new.md", "# New artifact\n");
      if (newSnapshot) return newSnapshot;

      if (method === "GET" && url.startsWith("/api/pipeline/runs/run-old/logs?")) {
        return jsonResponse({ run_id: "run-old", items: [], next_cursor: 0, eof: true });
      }

      if (method === "GET" && url.startsWith("/api/pipeline/runs/run-new/logs?")) {
        return jsonResponse({ run_id: "run-new", items: [], next_cursor: 0, eof: true });
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    navigateToStage("review");
    navigateToStage("review");
    fireEvent.click(await screen.findByRole("button", { name: /reports\/as-is\/old\.md/i }));

    await waitFor(() => {
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").toContain("reports/as-is/old.md");
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").toContain("Old artifact");
    });

    navigateToStage("analysis");
    fireEvent.click(screen.getByTestId("destination-runs"));
    fireEvent.click(screen.getByRole("button", { name: "run-new" }));

    await waitFor(() => {
      expect(screen.getByTestId("run-status-run-id").textContent).toBe("run-new");
    });

    navigateToStage("review");
    navigateToStage("review");

    await waitFor(() => {
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").toContain("reports/as-is/new.md");
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").toContain("New artifact");
      expect(screen.getByTestId("evidence-viewer").textContent ?? "").not.toContain("Old artifact");
    });
  });

  it("ignores a late Git diff response after a newer artifact path is selected", async () => {
    const runID = "run-diff-race";
    const lateOldDiff = deferredResponse();
    let oldDiffRequested = false;
    const diffPayload = (path: string, marker: string) => ({
      ok: true,
      workspace: "/tmp/workspace",
      run_id: runID,
      step_id: null,
      selected_path: path,
      selected_file: {
        path,
        folder: "reports/as-is",
        status: "modified",
        additions: 1,
        deletions: 0,
        binary: false,
      },
      files: [
        {
          path,
          folder: "reports/as-is",
          status: "modified",
          additions: 1,
          deletions: 0,
          binary: false,
        },
      ],
      folders: [{ folder: "reports/as-is", files: 1, additions: 1, deletions: 0 }],
      hunks: [
        {
          header: "@@ -1 +1 @@",
          lines: [{ kind: "add", new_line: 1, content: marker }],
        },
      ],
      message: marker,
      empty: false,
    });
    const baseFetch = createFetchMock({
      runID,
      runStarted: true,
      runArtifacts: {
        [runID]: {
          run_id: runID,
          artifacts: [
            { path: "reports/as-is/old.md", kind: "report", label: "Old diff artifact" },
            { path: "reports/as-is/new.md", kind: "report", label: "New diff artifact" },
          ],
        },
      },
      artifactText: {
        "reports/as-is/old.md": "# Old artifact\n",
        "reports/as-is/new.md": "# New artifact\n",
      },
    });
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      if (method === "GET" && url.startsWith("/api/git/diff")) {
        const parsed = new URL(url, "http://localhost");
        const selectedPath = parsed.searchParams.get("path") ?? "";
        if (selectedPath === "reports/as-is/old.md") {
          oldDiffRequested = true;
          return lateOldDiff.promise;
        }
        if (selectedPath === "reports/as-is/new.md") {
          return jsonResponse(diffPayload(selectedPath, "New diff content"));
        }
        return jsonResponse(diffPayload(selectedPath || "reports/as-is/default.md", "Default diff content"));
      }
      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("review");
    fireEvent.click(await screen.findByRole("button", { name: /reports\/as-is\/old\.md/i }));
    await waitFor(() => expect(oldDiffRequested).toBe(true));

    fireEvent.click(await screen.findByRole("button", { name: /reports\/as-is\/new\.md/i }));
    fireEvent.click(screen.getByTestId("stage-diff"));

    await waitFor(() => {
      expect(screen.getByTestId("git-diff-view").textContent ?? "").toContain("reports/as-is/new.md");
      expect(screen.getByTestId("git-diff-view").textContent ?? "").toContain("New diff content");
    });

    lateOldDiff.resolve(jsonResponse(diffPayload("reports/as-is/old.md", "Old diff content SHOULD NOT SHOW")));

    await waitFor(() => {
      expect(screen.getByTestId("git-diff-view").textContent ?? "").toContain("reports/as-is/new.md");
      expect(screen.getByTestId("git-diff-view").textContent ?? "").toContain("New diff content");
      expect(screen.getByTestId("git-diff-view").textContent ?? "").not.toContain("SHOULD NOT SHOW");
    });
  }, 10_000);

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

    navigateToStage("analysis");
    await waitFor(() => expect(screen.getByTestId("destination-runs")).toHaveAttribute("aria-current", "page"));

    await screen.findByTestId("run-status-panel");
    expect(screen.getByTestId("run-status-value").textContent).toBe("failed");
    expect(screen.getByTestId("run-lifecycle-value").textContent).toBe("terminal");
    expect(screen.getByText("Stopped at: refresh.step3.findings")).toBeInTheDocument();
    expect(screen.getByText("Error code: run_partial_failed")).toBeInTheDocument();
    expect(screen.getByText("Error: runtime draft manifest invalid")).toBeInTheDocument();
    expect(screen.getByTestId("run-status-warnings").textContent ?? "").toContain("collect coverage incomplete");
    expect(screen.getByTestId("run-status-warnings").textContent ?? "").toContain("draft promotion skipped");

    const recovery = screen.getByTestId("analysis-failure-recovery");
    expect(recovery).toHaveTextContent("Recovery path");
    expect(recovery).toHaveTextContent("run_partial_failed");
    expect(recovery).toHaveTextContent("refresh.step3.findings");
    expect(recovery).toHaveTextContent("This attempt did not replace the last-good promoted architecture");
    expect(screen.getByTestId("analysis-retry-run-btn")).toHaveTextContent("Calculate retry plan");

    fireEvent.click(screen.getByTestId("analysis-review-blocker-btn"));
    expect(screen.getByTestId("destination-runs")).toHaveAttribute("aria-current", "page");
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

    await renderConsoleApp(`/runs/${runID}`);
    await screen.findByTestId("run-status-panel");

    const recovery = screen.getByTestId("analysis-failure-recovery");
    expect(recovery).toHaveTextContent("provider");
    expect(recovery).toHaveTextContent("provider quota exhausted");
    expect(recovery).toHaveTextContent("refresh.step1.collect");
    const liveDiagnostics = screen.getByTestId("analysis-live-diagnostics");
    expect(liveDiagnostics).toHaveTextContent("provider check");
    expect(liveDiagnostics).toHaveTextContent("provider unavailable before shard ids were emitted");
    expect(liveDiagnostics).toHaveTextContent("Check Readiness provider setup, binary/auth/quota before retrying the same pipeline");

    fireEvent.click(screen.getByTestId("destination-changes"));
    await waitFor(() => expect(screen.getByTestId("destination-changes")).toHaveAttribute("aria-current", "page"));
    fireEvent.click(screen.getByTestId("stage-publish"));
    expect(await screen.findByTestId("publish-panel")).toBeInTheDocument();
    expect(screen.getByTestId("publish-gate-panel")).toHaveTextContent("Provider unavailable");
    expect(screen.getByTestId("publish-gate-panel")).toHaveTextContent("run a successful analysis before publishing");

    navigateToStage("analysis");
    await screen.findByTestId("run-status-panel");
    const startCallsBefore = fetchMock.mock.calls.filter((call) => call[0] === "/api/pipeline/init" || call[0] === "/api/pipeline/refresh").length;
    fireEvent.click(screen.getByTestId("setup-utility"));
    await screen.findByTestId("guided-setup-page");
    fireEvent.click(screen.getByTestId("stage-readiness"));
    expect(screen.getByTestId("stage-readiness")).toHaveAttribute("aria-current", "page");
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

    navigateToStage("readiness");
    await waitFor(() => expect(screen.getByTestId("stage-readiness")).toHaveAttribute("aria-current", "page"));
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

    navigateToStage("analysis");

    await screen.findByTestId("run-status-panel");
    expect(screen.getByTestId("run-status-value").textContent).toBe("canceled");
    expect(screen.getByTestId("analysis-run-progress")).toHaveTextContent("canceled");
    const recovery = screen.getByTestId("analysis-failure-recovery");
    expect(recovery).toHaveTextContent("Canceled run");
    expect(recovery).toHaveTextContent("run_canceled");
    expect(recovery).toHaveTextContent("Failed step");
    expect(recovery).toHaveTextContent("refresh.step2.asis_docs");
    expect(recovery).toHaveTextContent("The run stopped by request");
    expect(recovery).toHaveTextContent("Validated taskrun evidence remains attached to this immutable run");
    expect(screen.getByTestId("analysis-retry-run-btn")).toHaveTextContent("Calculate retry plan");
    expect(screen.getByTestId("analysis-review-recovery-btn")).toHaveTextContent("Open technical details");
    fireEvent.click(screen.getByTestId("destination-runs"));
    expect(screen.getByTestId("runs-history-panel")).toHaveTextContent("Failed: 0");
    expect(screen.getByTestId("runs-history-panel")).toHaveTextContent("Canceled: 1");
    expect(screen.getByTestId("runs-history-table")).toHaveTextContent("canceled");
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

    navigateToStage("analysis");

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

    navigateToStage("analysis");

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

    navigateToStage("analysis");

    await screen.findByTestId("run-status-panel");
    expect(screen.getByTestId("run-status-value").textContent).toBe("recovered");
    expect(screen.getByTestId("run-lifecycle-value").textContent).toBe("recovered");
    expect(screen.getByTestId("analysis-run-progress")).toHaveTextContent("recovered");
    const recovery = screen.getByTestId("analysis-failure-recovery");
    expect(recovery).toHaveTextContent("Recovered after restart");
    expect(recovery).toHaveTextContent("ACP reconciled a stale run after restart");
    expect(recovery).toHaveTextContent("Validated taskrun evidence remains attached to this immutable run");
    expect(screen.getByTestId("analysis-retry-run-btn")).toHaveTextContent("Calculate retry plan");
    expect(screen.getByTestId("analysis-review-recovery-btn")).toHaveTextContent("Open technical details");
    fireEvent.click(screen.getByTestId("destination-runs"));
    expect(screen.getByTestId("runs-history-panel")).toHaveTextContent("Failed: 0");
    expect(screen.getByTestId("runs-history-panel")).toHaveTextContent("Recovered: 1");
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
        if (staleRunStatusCalls >= 3) {
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
    navigateToStage("analysis");

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
    navigateToStage("analysis");

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

    navigateToStage("review");
    navigateToStage("review");

    const diagramButton = await screen.findByRole("button", { name: /reports\/diagrams\/c4-context\.mmd/i });
    fireEvent.click(diagramButton);

    await waitFor(() => {
      const preview = screen.getByTestId("evidence-viewer").innerHTML;
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

    navigateToStage("charter");

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

  it("keeps raw workspace.yaml edits dirty when save resolves after newer text", async () => {
    const saveManifest = deferredResponse();
    const savedManifests: string[] = [];
    const baseFetch = createFetchMock();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();

      if (method === "PUT" && url === "/api/workspace/manifest") {
        const body = JSON.parse(String(init?.body ?? "{}")) as { content?: string };
        savedManifests.push(body.content ?? "");
        return saveManifest.promise;
      }

      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    fireEvent.click(screen.getByText("Advanced workspace.yaml editor"));
    const editor = await screen.findByLabelText("workspace.yaml content");
    const savedDraft = 'version: 1\nrepos:\n  - name: "saved-draft"\n    path: "/tmp/saved"\n';
    const newerDraft = 'version: 1\nrepos:\n  - name: "newer-unsaved-draft"\n    path: "/tmp/newer"\n';
    fireEvent.change(editor, { target: { value: savedDraft } });
    fireEvent.click(screen.getByTestId("workspace-raw-save-btn"));
    fireEvent.change(editor, { target: { value: newerDraft } });

    saveManifest.resolve(jsonResponse({ ok: true }));

    expect(await screen.findByText("Saved workspace.yaml; newer unsaved edits remain.")).toBeInTheDocument();
    expect(screen.getByLabelText("workspace.yaml content")).toHaveValue(newerDraft);
    expect(savedManifests).toEqual([savedDraft]);
    expect(screen.queryByText("Saved workspace.yaml", { exact: true })).not.toBeInTheDocument();
  });

  it("ignores a late baseline artifact load after another path is selected", async () => {
    const lateOverviewLoad = deferredResponse();
    let overviewRequested = false;
    const baseFetch = createFetchMock({
      artifactText: {
        "skills/prompt-packs/qa.md": "qa prompt current\n",
      },
    });
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      if (method === "GET" && url === "/api/artifacts?path=charter%2Foverview.md") {
        overviewRequested = true;
        return lateOverviewLoad.promise;
      }
      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("charter");

    const select = await screen.findByLabelText(/select artifact/i);
    fireEvent.change(select, { target: { value: "charter/overview.md" } });
    await waitFor(() => expect(overviewRequested).toBe(true));
    fireEvent.change(select, { target: { value: "skills/prompt-packs/qa.md" } });

    const editor = await screen.findByLabelText("skills/prompt-packs/qa.md");
    await waitFor(() => expect(editor).toHaveValue("qa prompt current\n"));

    lateOverviewLoad.resolve(textResponse("late overview SHOULD NOT OVERWRITE\n"));

    await waitFor(() => {
      expect(screen.getByLabelText("skills/prompt-packs/qa.md")).toHaveValue("qa prompt current\n");
      expect(screen.getByLabelText("skills/prompt-packs/qa.md")).not.toHaveValue(expect.stringContaining("SHOULD NOT OVERWRITE"));
    });
  }, 10_000);

  it("preserves dirty baseline drafts independently per selected path", async () => {
    const fetchMock = createFetchMock({
      artifactText: {
        "charter/overview.md": "overview baseline\n",
        "skills/prompt-packs/qa.md": "qa prompt baseline\n",
      },
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("charter");

    const select = await screen.findByLabelText(/select artifact/i);
    fireEvent.change(select, { target: { value: "charter/overview.md" } });
    const overviewEditor = await screen.findByLabelText("charter/overview.md");
    await waitFor(() => expect(overviewEditor).toHaveValue("overview baseline\n"));
    fireEvent.change(overviewEditor, { target: { value: "overview dirty draft\n" } });

    fireEvent.change(select, { target: { value: "skills/prompt-packs/qa.md" } });
    const qaEditor = await screen.findByLabelText("skills/prompt-packs/qa.md");
    await waitFor(() => expect(qaEditor).toHaveValue("qa prompt baseline\n"));
    fireEvent.change(qaEditor, { target: { value: "qa dirty draft\n" } });

    fireEvent.change(select, { target: { value: "charter/overview.md" } });
    expect(await screen.findByLabelText("charter/overview.md")).toHaveValue("overview dirty draft\n");

    fireEvent.change(select, { target: { value: "skills/prompt-packs/qa.md" } });
    expect(await screen.findByLabelText("skills/prompt-packs/qa.md")).toHaveValue("qa dirty draft\n");
  });

  it("keeps baseline edits dirty when save resolves after newer text", async () => {
    const saveArtifact = deferredResponse();
    const savedArtifacts: Array<{ path: string; content: string }> = [];
    const baseFetch = createFetchMock({
      artifactText: {
        "skills/prompt-packs/qa.md": "qa prompt baseline\n",
      },
    });
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = (init?.method ?? "GET").toUpperCase();
      if (method === "POST" && url === "/api/artifacts/write") {
        const body = JSON.parse(String(init?.body ?? "{}")) as { path?: string; content?: string };
        savedArtifacts.push({ path: body.path ?? "", content: body.content ?? "" });
        return saveArtifact.promise;
      }
      return baseFetch(input, init);
    });
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();
    navigateToStage("charter");

    const select = await screen.findByLabelText(/select artifact/i);
    fireEvent.change(select, { target: { value: "skills/prompt-packs/qa.md" } });
    const editor = await screen.findByLabelText("skills/prompt-packs/qa.md");
    await waitFor(() => expect(editor).toHaveValue("qa prompt baseline\n"));
    fireEvent.change(editor, { target: { value: "qa prompt save snapshot\n" } });
    fireEvent.click(screen.getByRole("button", { name: /save selected baseline artifact/i }));
    fireEvent.change(editor, { target: { value: "qa prompt newer unsaved draft\n" } });

    saveArtifact.resolve(jsonResponse({ ok: true }));

    expect(await screen.findByText("Saved skills/prompt-packs/qa.md; newer unsaved edits remain.")).toBeInTheDocument();
    expect(screen.getByLabelText("skills/prompt-packs/qa.md")).toHaveValue("qa prompt newer unsaved draft\n");
    expect(savedArtifacts).toEqual([{ path: "skills/prompt-packs/qa.md", content: "qa prompt save snapshot\n" }]);
  }, 10_000);

  it("keeps Git mutations out of Charter and exposes them only in Publish", async () => {
    const fetchMock = createFetchMock();
    vi.stubGlobal("fetch", fetchMock);

    await renderConsoleApp();

    navigateToStage("charter");
    expect(screen.queryByLabelText("Commit message")).not.toBeInTheDocument();
    expect(screen.queryByTestId("git-proposal-branch-btn")).not.toBeInTheDocument();

    navigateToStage("publish");
    expect(await screen.findByLabelText("Commit message")).toBeInTheDocument();
    expect(screen.getByTestId("git-proposal-branch-btn")).toBeInTheDocument();
    expect(fetchMock.mock.calls.filter((call) => call[0] === "/api/git/commit")).toHaveLength(0);
    expect(fetchMock.mock.calls.filter((call) => call[0] === "/api/git/proposal-branch")).toHaveLength(0);
  });
});
