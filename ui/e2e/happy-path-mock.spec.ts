import { expect, test, type Page } from "@playwright/test";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { expectNoCriticalAxeViolations } from "./axe";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const screenshotOutputDir = (process.env.UI_E2E_OUTPUT_DIR ?? "").trim();
const initRunID = "run-happy-init";
const refreshRunID = "run-happy-refresh";
const taskID = "task-happy-checkout";
const attemptID = "attempt-happy-init";

type FulfillBody = { status: number; contentType: string; body: string };

function json(body: unknown, status = 200): FulfillBody {
  return { status, contentType: "application/json", body: JSON.stringify(body) };
}

function text(body: string, status = 200): FulfillBody {
  return { status, contentType: "text/plain; charset=utf-8", body };
}

async function captureEvidenceScreenshot(page: Page, name: string): Promise<void> {
  if (!screenshotOutputDir) return;
  mkdirSync(screenshotOutputDir, { recursive: true });
  const screenshotPath = path.join(screenshotOutputDir, name);
  await page.screenshot({ path: screenshotPath, fullPage: true });
  await test.info().attach(name, { path: screenshotPath, contentType: "image/png" });
}

function runStatus(runID: string, pipeline: "init" | "refresh") {
  return {
    run_id: runID,
    pipeline,
    status: "succeeded",
    started_at: "2026-08-05T12:00:00Z",
    finished_at: "2026-08-05T12:08:00Z",
    current_step: `${pipeline}.step${pipeline === "init" ? "4" : "3"}.findings`,
    runtime_mode: "fake",
    step_providers: { [`${pipeline}.step1.collect`]: "fake" },
    warnings: [],
    error_code: null,
    error: null,
    refresh_summary:
      pipeline === "refresh"
        ? {
            mode: "affected_only",
            decision: "selective_candidate",
            baseline_run_id: initRunID,
            reason_codes: ["source_revisions_changed"],
            artifact_path: `reports/taskruns/${runID}/refresh-execution.json`,
            updated: 2,
            preserved: 3,
            removed: 0,
            uncertain: 1,
          }
        : null,
  };
}

function runListItem(runID: string, pipeline: "init" | "refresh") {
  return { ...runStatus(runID, pipeline), authoritative_index: true };
}

function taskPayload(attemptStarted = false) {
  return {
    version: 1,
    task_id: taskID,
    title: "Map checkout architecture",
    goal: "Describe the checkout service and its order persistence boundary.",
    context: "Deterministic Task-first mock flow.",
    scope: { repositories: [{ name: "checkout", paths: [] }] },
    desired_runner: { preset: "deterministic-demo", mode: "fake", provider: "claude-code" },
    lifecycle: "open",
    revision: 1,
    created_at: "2026-08-05T11:55:00Z",
    updated_at: "2026-08-05T12:08:00Z",
    last_activity_at: "2026-08-05T12:08:00Z",
    attempts: attemptStarted ? [{ attempt_id: attemptID, run_id: initRunID, status: "succeeded", updated_at: "2026-08-05T12:08:00Z" }] : [],
    outcome: attemptStarted ? { state: "available", attempt_id: attemptID, run_id: initRunID, snapshot_path: `reports/taskruns/${initRunID}/snapshot.json` } : { state: "unavailable", unavailable_reason: "no terminal Attempt" },
    publication: { state: "unavailable", unavailable_reason: "No publication has been recorded for this Task." },
  };
}

function attemptPayload() {
  return {
    version: 1,
    attempt_id: attemptID,
    task_id: taskID,
    run_id: initRunID,
    pipeline: "init",
    status: "succeeded",
    task_revision: 1,
    goal_snapshot: "Describe the checkout service and its order persistence boundary.",
    context_snapshot: "Deterministic Task-first mock flow.",
    scope_snapshot: { repositories: [{ name: "checkout", paths: [] }] },
    desired_runner: { preset: "deterministic-demo", mode: "fake", provider: "claude-code" },
    effective_runtime: { mode: "fake", provider: "claude-code", permissions: "trusted_full_access" },
    admitted_at: "2026-08-05T11:56:00Z",
    started_at: "2026-08-05T12:00:00Z",
    finished_at: "2026-08-05T12:08:00Z",
    outcome: { state: "available", attempt_id: attemptID, run_id: initRunID },
    retained_evidence: "The immutable run snapshot is retained.",
  };
}

const artifacts = [
  { path: "reports/as-is/overview.md", kind: "report", label: "Architecture overview" },
  { path: "reports/diagrams/c4-context.mmd", kind: "diagram", label: "C4 context" },
  { path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" },
  { path: "reports/findings/findings.md", kind: "report", label: "Findings" },
];

const artifactBodies: Record<string, string> = {
  "reports/as-is/overview.md":
    "# Architecture overview\n\nThe checkout service receives orders and persists them in the order store.\n\n## Evidence\n\nRepository evidence confirms the service boundary and its persistence dependency.\n",
  "reports/diagrams/c4-context.mmd": "flowchart LR\n  customer[Customer] --> checkout[Checkout service]\n  checkout --> orders[(Order store)]\n",
  "reports/coverage/summary.md":
    "# Coverage summary\n\nObserved repository evidence covers the checkout service, order store and customer boundary.\n",
  "reports/findings/findings.md": "# Findings\n\nNo blocking findings were produced for this deterministic flow.\n",
};

function snapshotArtifacts(runID: string) {
  return artifacts.map((artifact) => ({
    ...artifact,
    canonical_path: artifact.path,
    read_path: artifact.path,
    source_run_id: runID,
    source_mode: "run_snapshot",
  }));
}

function emptyChangeSet() {
  return { added: [], changed: [], removed: [] };
}

function reviewContract(runID: string, pipeline: "init" | "refresh") {
  const refresh = pipeline === "refresh";
  const semanticChanges = refresh
    ? {
        available: true,
        baseline_run_id: initRunID,
        current_run_id: refreshRunID,
        categories: {
          entities: { added: [{ id: "entity.checkout", name: "Checkout service" }], changed: [], removed: [] },
          edges: { added: [], changed: [{ id: "edge.checkout.orders", name: "Checkout writes orders" }], removed: [] },
          findings: emptyChangeSet(),
          gaps: { added: [{ id: "gap.owner", name: "Ownership evidence" }], changed: [], removed: [] },
        },
      }
    : {
        available: false,
        reason: "The initial architecture is the baseline for future refreshes.",
        categories: { entities: emptyChangeSet(), edges: emptyChangeSet(), findings: emptyChangeSet(), gaps: emptyChangeSet() },
      };
  return {
    review_kind: refresh ? "refresh" : "initial",
    source_run_id: runID,
    ...(refresh ? { baseline_run_id: initRunID } : {}),
    semantic_changes: semanticChanges,
    document_changes: {
      available: true,
      added: refresh ? [{ id: "doc.coverage", name: "Coverage summary", path: "reports/coverage/summary.md" }] : [],
      changed: refresh ? [{ id: "doc.overview", name: "Architecture overview", path: "reports/as-is/overview.md" }] : [],
      removed: [],
    },
    findings: [],
    questions: refresh ? [{ id: "question.owner", text: "Confirm service ownership evidence", priority: "normal" }] : [],
    gaps: refresh ? ["Ownership evidence"] : [],
    summary: {
      entities_added: refresh ? 1 : 0,
      entities_changed: 0,
      entities_removed: 0,
      edges_added: 0,
      edges_changed: refresh ? 1 : 0,
      edges_removed: 0,
      documents_added: refresh ? 1 : 0,
      documents_changed: refresh ? 1 : 0,
      documents_removed: 0,
      findings: 0,
      questions: refresh ? 1 : 0,
      gaps: refresh ? 1 : 0,
    },
    runtime: { mode: "fake", providers: ["fake"], step_providers: { [`${pipeline}.step1.collect`]: "fake" } },
    authority: { mode: "promoted_run_snapshot", source_run_id: runID, ...(refresh ? { baseline_run_id: initRunID } : {}) },
    generated_at: "2026-08-05T12:08:00Z",
  };
}

function reviewSummary(runID: string, pipeline: "init" | "refresh") {
  return {
    ...runStatus(runID, pipeline),
    steps: ["constitution", "collect", "as_is_docs", "findings", "proposals"].map((key, index) => ({
      step_id: `${pipeline}.step${index}`,
      key,
      label: key,
      state: "done",
      provider: "fake",
      artifact_count: index === 1 ? 2 : 1,
      artifact_paths: artifacts.slice(0, index === 1 ? 2 : 1).map((artifact) => artifact.path),
      taskrun_paths: [],
      warnings_count: 0,
      errors_count: 0,
      last_message: "completed",
    })),
    result: {
      state: "completed",
      summary: pipeline === "init" ? "Initial architecture baseline is ready for review." : "Architecture refresh is ready for review.",
      produced: { documents: artifacts.length, diagrams: 1 },
      partial_scopes: 0,
      failed_scopes: 0,
      promotion: { changed: true, current_usable: true, ...(pipeline === "refresh" ? { baseline_run_id: initRunID } : {}) },
      recommended_action: "review_changes",
      coverage: { observed: 3, missing: pipeline === "refresh" ? 1 : 0, status: pipeline === "refresh" ? "partial" : "available" },
    },
    review: reviewContract(runID, pipeline),
  };
}

function architecturePayload(sourceRunID: string | null) {
  const node = {
    id: "entity.checkout",
    name: "Checkout service",
    type: "service",
    confidence: 0.94,
    provenance_kind: "repository_evidence",
    evidence: [{ repo: "checkout", path: "src/checkout/service.ts", lines: { start: 10, end: 30 } }],
    path: "model/entities/checkout-service.yaml",
    available_levels: ["context", "container"],
    repositories: ["checkout"],
    related_findings: [],
    related_questions: [],
  };
  const store = {
    id: "entity.orders",
    name: "Order store",
    type: "datastore",
    confidence: 0.9,
    provenance_kind: "repository_evidence",
    evidence: [{ repo: "checkout", path: "src/orders/store.ts", lines: { start: 1, end: 20 } }],
    path: "model/entities/order-store.yaml",
    available_levels: ["container"],
    repositories: ["checkout"],
    related_findings: [],
    related_questions: [],
  };
  const edge = {
    id: "edge.checkout.orders",
    from: node.id,
    to: store.id,
    type: "writes",
    name: "Checkout writes orders",
    confidence: 0.88,
    provenance_kind: "repository_evidence",
    evidence: [{ repo: "checkout", path: "src/orders/store.ts", lines: { start: 8, end: 16 } }],
    path: "model/edges/checkout-writes-orders.yaml",
    repositories: ["checkout"],
    related_findings: [],
    related_questions: [],
  };
  const view = (level: string, nodes: unknown[], edges: unknown[]) => ({ level, available: true, nodes, edges });
  return {
    version: 1,
    generated_at: "2026-08-05T12:08:00Z",
    authority: { mode: "promoted_current", source_run_id: sourceRunID ?? undefined, freshness: sourceRunID ? "current" : "unknown" },
    status: sourceRunID ? "available" : "unavailable",
    counts: { entities: sourceRunID ? 2 : 0, edges: sourceRunID ? 1 : 0, evidence: sourceRunID ? 3 : 0, issues: 0 },
    views: {
      context: view("context", sourceRunID ? [node] : [], []),
      container: view("container", sourceRunID ? [node, store] : [], sourceRunID ? [edge] : []),
      component: view("component", [], []),
      code: view("code", [], []),
    },
    exports: { home_path: "reports/as-is/overview.md", c4_mermaid_paths: ["reports/diagrams/c4-context.mmd"] },
    comparison: {
      available: sourceRunID === refreshRunID,
      ...(sourceRunID === refreshRunID ? { baseline_run_id: initRunID, current_run_id: refreshRunID } : {}),
      categories: sourceRunID === refreshRunID ? reviewContract(refreshRunID, "refresh").semantic_changes.categories : { entities: emptyChangeSet(), edges: emptyChangeSet(), findings: emptyChangeSet(), gaps: emptyChangeSet() },
    },
    review: { findings: [], questions: sourceRunID === refreshRunID ? [{ id: "question.owner", text: "Confirm service ownership evidence", priority: "normal" }] : [] },
    coverage: { observed: ["checkout"], missing: sourceRunID === refreshRunID ? ["ownership"] : [], notes: [] },
    artifacts: sourceRunID ? artifacts.map(({ path, kind, label }) => ({ path, kind, name: label })) : [],
    issues: [],
  };
}

async function installHappyPathMock(page: Page): Promise<{ commitMessages: string[] }> {
  const commitMessages: string[] = [];
  let promotedRunID: string | null = null;
  let taskCreated = false;
  let attemptAdmitted = false;

  const fullDiff = {
    ok: true,
    state: "dirty",
    workspace: "/tmp/happy-path-workspace",
    scope: "full_workspace",
    branch: "main",
    head_oid: "head-happy",
    base_ref: "HEAD",
    base_oid: "head-happy",
    fingerprint: "happy-path-fingerprint",
    run_id: null,
    step_id: null,
    selected_path: null,
    selected_file: null,
    files: [
      { path: "reports/as-is/overview.md", folder: "reports/as-is", status: "modified", additions: 8, deletions: 1, binary: false, index_status: "", worktree_status: "M", unavailable: false },
      { path: "reports/diagrams/c4-context.mmd", folder: "reports/diagrams", status: "new", additions: 4, deletions: 0, binary: false, index_status: "", worktree_status: "?", unavailable: false },
    ],
    folders: [
      { folder: "reports/as-is", files: 1, additions: 8, deletions: 1 },
      { folder: "reports/diagrams", files: 1, additions: 4, deletions: 0 },
    ],
    hunks: [{ header: "@@ -1,2 +1,4 @@", lines: [{ kind: "context", old_line: 1, new_line: 1, content: "# Architecture overview" }, { kind: "add", new_line: 2, content: "Validated checkout boundary." }] }],
    message: "Full workspace Git diff loaded.",
    empty: false,
  };

  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const method = request.method().toUpperCase();
    const apiPath = `${url.pathname}${url.search}`;
    const runIDs = promotedRunID ? [initRunID, refreshRunID] : [];

    if (method === "GET" && url.pathname === "/api/system/version") return route.fulfill({ ...json({ version: "dev", commit: "mock", built: "mock", ui_bundle: "vite" }) });
    if (method === "GET" && url.pathname === "/api/onboarding/status") return route.fulfill({ ...json({ ok: true, launcher_mode: false, workspace_selected: true, workspace_ready: true, workspace: "/tmp/happy-path-workspace", manifest_present: true, runtime: { selected: true, runtime: "fake", runtime_provider: "fake", provider_source: "workspace" }, can_enter_console: true, recent_workspaces: [] }) });
    if (method === "GET" && url.pathname === "/api/tasks") return route.fulfill({ ...json({ items: taskCreated ? [taskPayload(attemptAdmitted)] : [], next_cursor: "", has_more: false }) });
    if (method === "POST" && url.pathname === "/api/tasks") {
      taskCreated = true;
      return route.fulfill({ ...json({ task: taskPayload(false) }, 201) });
    }
    if (method === "GET" && url.pathname === `/api/tasks/${taskID}`) return route.fulfill({ ...json({ task: taskPayload(attemptAdmitted) }) });
    if (method === "GET" && url.pathname === `/api/tasks/${taskID}/attempts`) return route.fulfill({ ...json({ items: attemptAdmitted ? [attemptPayload()] : [] }) });
    if (method === "GET" && url.pathname === `/api/tasks/${taskID}/attempts/${attemptID}`) return route.fulfill({ ...json({ attempt: attemptPayload() }) });
    if (method === "POST" && url.pathname === `/api/tasks/${taskID}/attempts`) {
      taskCreated = true;
      attemptAdmitted = true;
      promotedRunID = initRunID;
      return route.fulfill({ ...json({ attempt: attemptPayload() }, 202) });
    }
    if (method === "GET" && url.pathname === "/api/pipeline/runs") return route.fulfill({ ...json({ items: runIDs.map((id) => runListItem(id, id === initRunID ? "init" : "refresh")), coordination: {} }) });
    if (method === "POST" && url.pathname === "/api/pipeline/init") {
      promotedRunID = initRunID;
      return route.fulfill({ ...json({ run_id: initRunID, status: "succeeded" }) });
    }
    if (method === "POST" && url.pathname === "/api/pipeline/refresh") {
      promotedRunID = refreshRunID;
      return route.fulfill({ ...json({ run_id: refreshRunID, status: "succeeded" }) });
    }
    const runMatch = url.pathname.match(/^\/api\/pipeline\/runs\/([^/]+)(?:\/(.*))?$/);
    if (runMatch) {
      const id = decodeURIComponent(runMatch[1]);
      const pipeline = id === refreshRunID ? "refresh" : "init";
      if (id !== initRunID && id !== refreshRunID) return route.fulfill({ ...json({ error: { code: "not_found", message: "run not found" } }, 404) });
      if (runMatch[2] === "snapshot") return route.fulfill({ ...json({ run_id: id, status: "available", artifacts: snapshotArtifacts(id), issues: [] }) });
      if (runMatch[2] === "artifacts") return route.fulfill({ ...json({ run_id: id, artifacts: snapshotArtifacts(id) }) });
      if (runMatch[2] === "logs") return route.fulfill({ ...json({ run_id: id, items: [], next_cursor: 0, eof: true }) });
      if (runMatch[2] === "review-summary") return route.fulfill({ ...json(reviewSummary(id, pipeline)) });
      return route.fulfill({ ...json(runStatus(id, pipeline)) });
    }
    if (method === "GET" && url.pathname === "/api/workspace/manifest") return route.fulfill({ ...json({ content: 'version: 1\nrepos:\n  - name: checkout\n    path: /tmp/checkout\ndocs:\n  imports_path: ./docs/imports\n' }) });
    if (method === "GET" && url.pathname === "/api/workspace/bundle") return route.fulfill({ ...json({ ok: true, workspace: "/tmp/happy-path-workspace", manifest: { schema_version: 1, bundle_version: 1, editable_artifacts: [{ path: "charter/overview.md", label: "charter/overview.md", category: "charter" }] }, warnings: [] }) });
    if (method === "POST" && url.pathname === "/api/workspace/validate") return route.fulfill({ ...json({ ok: true, workspace: "/tmp/happy-path-workspace", warnings: [], errors: [], resolved_repos: [{ name: "checkout", source: "/tmp/checkout", path: "/tmp/checkout", ref: "main", status: "ok" }] }) });
    if (method === "GET" && url.pathname === "/api/workspace/health") return route.fulfill({ ...json({ version: 1, generated_at: "2026-08-05T12:00:00Z", status: "pass", summary: { info: 0, warning: 0, error: 0 }, items: [] }) });
    if (method === "GET" && url.pathname === "/api/system/doctor") return route.fulfill({ ...json({ ok: true, summary: "ready", checks: [{ id: "git", label: "Git", status: "pass", message: "git found" }, { id: "runtime_provider", label: "Runtime provider", status: "pass", message: "fake runtime ready" }] }) });
    if (method === "GET" && url.pathname === "/api/runtime/timeouts") return route.fulfill({ ...json({ ok: true, persisted: { step_timeout_sec: 5400 }, effective: { step_timeout_sec: 5400, heartbeat_sec: 30, pipeline_timeout_sec: 14400, pipeline_kill_grace_sec: 30, api_ready_timeout_sec: 60, api_init_timeout_sec: 120, ui_init_poll_timeout_sec: 1500, ui_cancel_poll_timeout_sec: 420 }, source: { step_timeout_sec: "workspace" } }) });
    if (method === "GET" && url.pathname === "/api/runtime/execution") return route.fulfill({ ...json({ ok: true, persisted: { strategy: "sequential", max_parallel_tasks: 1 }, effective: { strategy: "sequential", max_parallel_tasks: 1, failure_policy: "best_effort", shard_discovery_mode: "heuristics" }, source: { strategy: "workspace" } }) });
    if (method === "GET" && url.pathname === "/api/runtime/permissions") return route.fulfill({ ...json({ ok: true, persisted: { mode: "trusted_full_access", approval_channel: "fail_fast" }, effective: { mode: "trusted_full_access", approval_channel: "fail_fast" }, source: { mode: "workspace", approval_channel: "default" } }) });
    if (method === "GET" && url.pathname === "/api/runtime/profile") return route.fulfill({ ...json({ ok: true, runtime_mode: "fake", runtime_provider: "claude-code", provider_source: "workspace", permissions: { persisted: { mode: "trusted_full_access", approval_channel: "fail_fast" }, effective: { mode: "trusted_full_access", approval_channel: "fail_fast" }, source: { mode: "workspace", approval_channel: "default" } }, step_providers: { persisted: {}, effective: {}, source: {} } }) });
    if (method === "GET" && url.pathname === "/api/architecture") return route.fulfill({ ...json(architecturePayload(promotedRunID)) });
    if (method === "GET" && url.pathname === "/api/knowledge") return route.fulfill({ ...json({ version: 1, generated_at: "2026-08-05T12:08:00Z", source_mode: "promoted_current", status: promotedRunID ? "available" : "unavailable", entities: [], edges: [], artifacts: promotedRunID ? artifacts.map(({ path, kind, label }) => ({ path, kind, name: label })) : [], issues: [] }) });
    if (method === "GET" && url.pathname === "/api/artifacts") {
      const requestedPath = url.searchParams.get("path") ?? "";
      return route.fulfill({ ...(requestedPath in artifactBodies ? text(artifactBodies[requestedPath]) : text("", 404)) });
    }
    if (method === "GET" && url.pathname === "/api/git/diff") return route.fulfill({ ...json({ ...fullDiff, run_id: url.searchParams.get("run_id") }) });
    if (method === "POST" && url.pathname === "/api/git/commit") {
      const payload = JSON.parse(request.postData() || "{}") as { message?: string };
      commitMessages.push(payload.message ?? "");
      return route.fulfill({ ...json({ status: "committed", output: `committed: ${payload.message ?? ""}` }) });
    }
    return route.fulfill({ ...json({ ok: true, ignored: apiPath }) });
  });

  return { commitMessages };
}

async function expectNoHorizontalOverflow(page: Page): Promise<void> {
  await expect.poll(async () => page.evaluate(() => Math.max(document.documentElement.scrollWidth - document.documentElement.clientWidth, document.body.scrollWidth - document.body.clientWidth))).toBeLessThanOrEqual(1);
}

test("Task-first mock: create Task -> immutable Attempt -> architecture -> full workspace publish", async ({ page }) => {
  test.skip(scenario !== "happy-path-mock", `scenario ${scenario} skips happy path mock`);
  const consoleErrors: string[] = [];
  page.on("console", (message) => { if (message.type() === "error") consoleErrors.push(message.text()); });
  const { commitMessages } = await installHappyPathMock(page);

  await page.goto("/tasks");
  await expect(page.getByTestId("task-route-inbox")).toBeVisible();
  await page.getByTestId("task-inbox-new").click();
  await expect(page.getByTestId("task-composer")).toBeVisible();
  await page.getByTestId("task-title").fill("Map checkout architecture");
  await page.getByTestId("task-goal").fill("Describe the checkout service and its order persistence boundary.");
  await page.getByTestId("task-create-submit").click();
  await expect(page.getByTestId("task-route-detail")).toBeVisible();
  await expect(page.getByTestId("task-outcome")).toContainText("Initial architecture baseline is ready for review.");
  await expect(page.getByTestId("task-outcome")).toContainText(initRunID);
  await page.getByTestId("task-open-architecture").click();
  await expect(page.getByTestId("knowledge-panel")).toBeVisible();
  await expect(page.getByTestId("knowledge-panel")).toContainText(initRunID);
  await page.goBack();
  await expect(page.getByTestId("task-route-detail")).toBeVisible();
  await expect(page.getByTestId(`attempt-row-${attemptID}`)).toBeVisible();
  await page.getByTestId(`attempt-row-${attemptID}`).click();
  await expect(page.getByTestId("task-route-attempt")).toBeVisible();
  await page.getByTestId("attempt-open-studio").click();
  await expect(page.getByTestId("task-pipeline-studio")).toBeVisible();
  await expect(page.getByTestId("task-pipeline-studio")).toContainText(taskID);
  await expect(page.getByTestId("task-pipeline-studio")).toContainText(attemptID);
  await expect(page.getByTestId("task-pipeline-studio")).toContainText(initRunID);
  await captureEvidenceScreenshot(page, "happy-path-task-attempt.png");

  await page.goto("/knowledge?view=documents&source=current");
  await expect(page.getByTestId("architecture-documents")).toBeVisible();
  await expect(page.getByRole("region", { name: "Architecture document reader" })).toContainText("Architecture overview");
  await page.getByRole("button", { name: "Diagrams", exact: true }).click();
  await expect(page.getByTestId("architecture-diagrams")).toBeVisible();
  await captureEvidenceScreenshot(page, "happy-path-architecture.png");

  await page.goto(`/changes?task=${taskID}&attempt=${attemptID}&run=${initRunID}&view=overview&source=snapshot&mode=rendered`);
  await expect(page.getByTestId("semantic-changes")).toBeVisible();
  await expect(page.getByTestId("semantic-changes")).toContainText("Run-pinned initial review");
  await expect(page.getByTestId("run-pinned-review-summary")).toContainText("0");
  await captureEvidenceScreenshot(page, "happy-path-changes.png");

  await page.goto(`/changes?task=${taskID}&attempt=${attemptID}&run=${initRunID}&view=publish&source=snapshot&mode=rendered`);
  await expect(page.getByTestId("publish-panel")).toBeVisible();
  await expect(page.getByTestId("publish-readiness-summary")).toContainText("Workspace scope");
  await expect(page.getByTestId("publish-readiness-summary")).toContainText("2 changed");
  await expect(page.getByTestId("publish-commit-selected-btn")).toBeEnabled();
  await page.getByLabel("Commit message").fill("docs: publish Task architecture");
  await page.getByTestId("publish-commit-selected-btn").click();
  await expect(page.getByRole("dialog")).toContainText("Commit all workspace changes");
  await page.getByRole("dialog").getByRole("button", { name: "Commit all workspace changes" }).click();
  await expect(page.getByTestId("publish-commit-plan")).toContainText("committed: docs: publish Task architecture");
  expect(commitMessages).toEqual(["docs: publish Task architecture"]);
  await expectNoHorizontalOverflow(page);
  await expectNoCriticalAxeViolations(page);
  expect(consoleErrors).toEqual([]);
  await captureEvidenceScreenshot(page, "happy-path-publish.png");
});
