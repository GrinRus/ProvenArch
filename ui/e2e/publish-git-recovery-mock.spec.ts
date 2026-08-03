import { expect, test, type Page } from "@playwright/test";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { expectNoCriticalAxeViolations } from "./axe";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const screenshotOutputDir = (process.env.UI_E2E_OUTPUT_DIR ?? "").trim();
const runID = "run-publish-git-recovery";

type FulfillBody = { status: number; contentType: string; body: string };

function json(body: unknown, status = 200): FulfillBody {
  return { status, contentType: "application/json", body: JSON.stringify(body) };
}

function text(body: string, status = 200): FulfillBody {
  return { status, contentType: "text/plain; charset=utf-8", body };
}

async function captureEvidenceScreenshot(page: Page, name: string): Promise<string | null> {
  if (screenshotOutputDir === "") {
    return null;
  }
  mkdirSync(screenshotOutputDir, { recursive: true });
  const screenshotPath = path.join(screenshotOutputDir, name);
  await page.screenshot({ path: screenshotPath, fullPage: true });
  await test.info().attach(name, {
    path: screenshotPath,
    contentType: "image/png",
  });
  return screenshotPath;
}

async function installPublishGitRecoveryMock(page: Page): Promise<{ commitMessages: string[]; branchNames: string[] }> {
  const commitMessages: string[] = [];
  const branchNames: string[] = [];
  const finalIndexPath = `reports/taskruns/${runID}/staging/final/final-run-index.json`;
  const artifacts = [
    { path: "reports/coverage/summary.md", kind: "report", label: "Coverage summary" },
    { path: "model/entities/checkout-service.yaml", kind: "model", label: "checkout-service" },
    { path: "proposals/checkout-ownership/proposal.md", kind: "proposal", label: "Checkout ownership proposal" },
    { path: "reports/changelog/2026-04-03.md", kind: "changelog", label: "Iteration changelog" },
    { path: finalIndexPath, kind: "taskrun", label: "Final run index" },
  ];
  const gitDiff = {
    ok: true,
    state: "dirty",
    workspace: "/tmp/publish-git-recovery-workspace",
    branch: "main",
    head_oid: "abc123",
    base_ref: "HEAD",
    base_oid: "abc123",
    fingerprint: "publish-git-recovery-fixture",
    run_id: runID,
    step_id: null,
    selected_path: "reports/coverage/summary.md",
    selected_file: {
      path: "reports/coverage/summary.md",
      folder: "reports/coverage",
      status: "modified",
      additions: 8,
      deletions: 1,
      binary: false,
    },
    files: [
      { path: "reports/coverage/summary.md", folder: "reports/coverage", status: "modified", additions: 8, deletions: 1, binary: false },
      { path: "model/entities/checkout-service.yaml", folder: "model/entities", status: "new", additions: 22, deletions: 0, binary: false },
      { path: "proposals/checkout-ownership/proposal.md", folder: "proposals", status: "new", additions: 18, deletions: 0, binary: false },
    ],
    folders: [
      { folder: "reports/coverage", files: 1, additions: 8, deletions: 1 },
      { folder: "model/entities", files: 1, additions: 22, deletions: 0 },
      { folder: "proposals", files: 1, additions: 18, deletions: 0 },
    ],
    hunks: [
      {
        header: "@@ -1,2 +1,4 @@",
        lines: [
          { kind: "context", old_line: 1, new_line: 1, content: "Coverage ready for publication." },
          { kind: "add", new_line: 2, content: "Open ownership question was resolved before handoff." },
        ],
      },
    ],
    message: "Selected run Git diff loaded.",
    empty: false,
  };

  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const method = request.method().toUpperCase();
    const apiPath = `${url.pathname}${url.search}`;

    if (method === "GET" && url.pathname === "/api/system/version") {
      await route.fulfill({ ...json({ version: "dev", commit: "mock", built: "mock", ui_bundle: "vite" }) });
      return;
    }

    if (method === "GET" && url.pathname === "/api/onboarding/status") {
      await route.fulfill({
        ...json({
          ok: true,
          launcher_mode: false,
          workspace_selected: true,
          workspace_ready: true,
          workspace: "/tmp/publish-git-recovery-workspace",
          manifest_present: true,
          runtime: { selected: true, runtime: "headless", runtime_provider: "qwen-code", provider_source: "override" },
          can_enter_console: true,
          recent_workspaces: [],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/pipeline/runs") {
      await route.fulfill({
        ...json({
          items: [
            {
              run_id: runID,
              pipeline: "init",
              status: "succeeded",
              current_step: "init.step4.proposals",
              started_at: "2026-04-03T12:00:00Z",
              finished_at: "2026-04-03T12:08:00Z",
              warnings: [],
              error_code: null,
              error: null,
            },
          ],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}`) {
      await route.fulfill({
        ...json({
          run_id: runID,
          pipeline: "init",
          status: "succeeded",
          current_step: "init.step4.proposals",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: "2026-04-03T12:08:00Z",
          warnings: [],
          error_code: null,
          error: null,
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}/review-summary`) {
      await route.fulfill({
        ...json({
          run_id: runID,
          pipeline: "init",
          status: "succeeded",
          current_step: "init.step4.proposals",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: "2026-04-03T12:08:00Z",
          warnings: [],
          error_code: null,
          error: null,
          steps: [
            { step_id: "step0_constitution", label: "Charter", state: "done", provider: "qwen-code", artifact_count: 1, warnings_count: 0, errors_count: 0 },
            { step_id: "step1_collect", label: "Collect", state: "done", provider: "qwen-code", artifact_count: 2, warnings_count: 0, errors_count: 0 },
            { step_id: "step2_as_is", label: "As-is", state: "done", provider: "qwen-code", artifact_count: 2, warnings_count: 0, errors_count: 0 },
            { step_id: "step3_findings", label: "Findings", state: "done", provider: "qwen-code", artifact_count: 1, warnings_count: 0, errors_count: 0 },
            { step_id: "step4_proposals", label: "Proposals", state: "done", provider: "qwen-code", artifact_count: 2, warnings_count: 0, errors_count: 0 },
          ],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}/artifacts`) {
      await route.fulfill({ ...json({ run_id: runID, artifacts }) });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}/snapshot`) {
      await route.fulfill({
        ...json({
          run_id: runID,
          status: "available",
          issues: [],
          artifacts: artifacts.map((artifact) => ({
            ...artifact,
            canonical_path: artifact.path,
            read_path: artifact.path === finalIndexPath ? finalIndexPath : `reports/taskruns/${runID}/staging/final/${artifact.path}`,
            source_run_id: runID,
            source_mode: "run_snapshot",
          })),
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}/logs`) {
      await route.fulfill({ ...json({ run_id: runID, items: [], next_cursor: 0, eof: true }) });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/manifest") {
      await route.fulfill({
        ...json({
          content:
            'version: 1\nrepos:\n  - name: "checkout"\n    path: "/tmp/checkout"\ndocs:\n  imports_path: "./docs/imports"\nruntime:\n  selected: headless\n  provider: qwen-code\n',
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/bundle") {
      await route.fulfill({
        ...json({
          ok: true,
          workspace: "/tmp/publish-git-recovery-workspace",
          manifest: {
            schema_version: 1,
            bundle_version: 1,
            editable_artifacts: [{ path: "charter/overview.md", label: "charter/overview.md", category: "charter" }],
          },
          warnings: [],
        }),
      });
      return;
    }

    if (method === "POST" && url.pathname === "/api/workspace/validate") {
      await route.fulfill({
        ...json({
          ok: true,
          workspace: "/tmp/publish-git-recovery-workspace",
          warnings: [],
          errors: [],
          resolved_repos: [{ name: "checkout", source: "/tmp/checkout", ref: "main", status: "ok" }],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/system/doctor") {
      await route.fulfill({
        ...json({
          ok: true,
          summary: "ready",
          checks: [
            { id: "git", label: "Git", status: "pass", message: "git found" },
            { id: "runtime_provider", label: "Runtime provider", status: "pass", message: "qwen found" },
          ],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/timeouts") {
      await route.fulfill({
        ...json({
          ok: true,
          persisted: { step_timeout_sec: 5400 },
          effective: {
            step_timeout_sec: 5400,
            heartbeat_sec: 30,
            pipeline_timeout_sec: 14400,
            pipeline_kill_grace_sec: 30,
            api_ready_timeout_sec: 60,
            api_init_timeout_sec: 120,
            ui_init_poll_timeout_sec: 1500,
            ui_cancel_poll_timeout_sec: 420,
          },
          source: { step_timeout_sec: "workspace" },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/execution") {
      await route.fulfill({
        ...json({
          ok: true,
          persisted: { strategy: "sequential", max_parallel_tasks: 1 },
          effective: { strategy: "sequential", max_parallel_tasks: 1, failure_policy: "best_effort", shard_discovery_mode: "heuristics" },
          source: { strategy: "workspace" },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/permissions") {
      await route.fulfill({
        ...json({
          ok: true,
          persisted: { mode: "trusted_full_access", approval_channel: "fail_fast" },
          effective: { mode: "trusted_full_access", approval_channel: "fail_fast" },
          source: { mode: "workspace", approval_channel: "default" },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/profile") {
      await route.fulfill({
        ...json({
          ok: true,
          permissions: {
            persisted: { mode: "trusted_full_access", approval_channel: "fail_fast" },
            effective: { mode: "trusted_full_access", approval_channel: "fail_fast" },
            source: { mode: "workspace", approval_channel: "default" },
          },
          step_providers: {
            persisted: {},
            effective: {
              step0_constitution: "qwen-code",
              step1_collect: "qwen-code",
              step2_as_is: "qwen-code",
              step3_findings: "qwen-code",
              step4_proposals: "qwen-code",
            },
            source: {
              step0_constitution: "default",
              step1_collect: "default",
              step2_as_is: "default",
              step3_findings: "default",
              step4_proposals: "default",
            },
          },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/git/diff") {
      await route.fulfill({ ...json(gitDiff) });
      return;
    }

    if (method === "POST" && url.pathname === "/api/git/commit") {
      const payload = JSON.parse(request.postData() || "{}") as { message?: string };
      commitMessages.push(payload.message ?? "");
      await route.fulfill({
        ...json({ error: { code: "git_commit_failed", message: "workspace has unresolved merge conflicts in reports/coverage/summary.md" } }, 409),
      });
      return;
    }

    if (method === "POST" && url.pathname === "/api/git/proposal-branch") {
      const payload = JSON.parse(request.postData() || "{}") as { name?: string };
      branchNames.push(payload.name ?? "");
      await route.fulfill({
        ...json({ error: { code: "git_branch_failed", message: "proposal/beta-refresh already has uncommitted local changes" } }, 409),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/artifacts") {
      const requestedPath = url.searchParams.get("path") ?? "";
      const bodies: Record<string, string> = {
        "charter/overview.md": "# Charter\n\nPublish Git recovery fixture.\n",
        "reports/coverage/summary.md": "# Coverage\n\nCoverage ready for publication.\n",
        "reports/coverage/open-questions.md": "",
        "model/entities/checkout-service.yaml": "id: checkout-service\nkind: service\n",
        "proposals/checkout-ownership/proposal.md": "# Proposal\n\nAssign checkout ownership before release handoff.\n",
        "reports/changelog/2026-04-03.md": "# Iteration changelog\n\n- Prepared checkout ownership proposal.\n",
      };
      const canonicalPaths = artifacts.filter((artifact) => artifact.path !== finalIndexPath).map((artifact) => artifact.path);
      for (const canonicalPath of canonicalPaths) {
        bodies[`reports/taskruns/${runID}/staging/final/${canonicalPath}`] = bodies[canonicalPath];
      }
      bodies[finalIndexPath] = JSON.stringify({
        version: 1,
        run_id: runID,
        pipeline: "init",
        generated_at: "2026-04-03T12:08:00Z",
        canonical_documents: canonicalPaths.map((canonicalPath) => ({
          id: `doc.${canonicalPath}`,
          kind: artifacts.find((artifact) => artifact.path === canonicalPath)?.kind ?? "report",
          title: canonicalPath,
          canonical_path: canonicalPath,
          staged_path: `reports/taskruns/${runID}/staging/final/${canonicalPath}`,
        })),
      });
      await route.fulfill({ ...(requestedPath in bodies ? text(bodies[requestedPath]) : text(`# Fixture artifact\n\n${requestedPath || "unknown"}\n`)) });
      return;
    }

    await route.fulfill({ ...json({ ok: true, ignored: apiPath }) });
  });

  return { commitMessages, branchNames };
}

async function expectNoHorizontalOverflow(page: Page): Promise<void> {
  await expect
    .poll(async () =>
      page.evaluate(() => {
        const root = document.documentElement;
        const body = document.body;
        return Math.max(root.scrollWidth - root.clientWidth, body.scrollWidth - body.clientWidth);
      }),
    )
    .toBeLessThanOrEqual(1);
}

test("publish git recovery mock: failed git mutations stay local and retryable", async ({ page }) => {
  test.skip(scenario !== "publish-git-recovery-mock", `scenario ${scenario} skips Publish Git recovery mock`);

  const consoleErrors: string[] = [];
  page.on("console", (message) => {
    const text = message.text();
    if (message.type() === "error" && !text.includes("409 (Conflict)")) {
      consoleErrors.push(text);
    }
  });
  const { commitMessages, branchNames } = await installPublishGitRecoveryMock(page);

  await page.setViewportSize({ width: 1440, height: 980 });
  await page.goto("/");
  await expect(page.getByTestId("product-shell")).toBeVisible();
  await page.getByRole("link", { name: "Changes" }).click();
  await expect(page.getByTestId("changes-route-overview")).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "changes-overview-desktop.png");

  await page.setViewportSize({ width: 1024, height: 768 });
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "changes-overview-tablet.png");

  await page.setViewportSize({ width: 390, height: 844 });
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "changes-overview-mobile.png");

  await page.setViewportSize({ width: 1440, height: 980 });
  await page.getByTestId("stage-publish").click();
  await expect(page.getByTestId("stage-publish")).toHaveAttribute("aria-current", "page");
  await expect(page.getByTestId("publish-panel")).toBeVisible();
  await expect(page.getByTestId("publish-readiness-summary")).toContainText("ready");
  await expect(page.getByTestId("publish-readiness-summary")).toContainText("Commit message prepared");
  await expect(page.getByTestId("publish-commit-selected-btn")).toBeEnabled();

  await page.getByLabel("Commit message").fill("docs: publish checkout architecture evidence");
  await page.getByTestId("publish-commit-selected-btn").click();
  await page.getByRole("button", { name: "Commit all workspace changes" }).click();
  const recovery = page.getByTestId("publish-git-action-recovery");
  await expect(recovery).toBeVisible();
  await expect(recovery).toContainText("Git action failed");
  await expect(recovery).toContainText("Git mutation failed: workspace has unresolved merge conflicts in reports/coverage/summary.md");
  await expect(recovery).toContainText("Workspace Git state was not changed");
  await expect(page.getByTestId("publish-readiness-summary")).toContainText("failed");
  await expect(page.getByTestId("publish-commit-selected-btn")).toBeEnabled();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "publish-git-recovery-desktop.png");

  await page.setViewportSize({ width: 1024, height: 768 });
  await expect(page.getByTestId("publish-panel")).toBeVisible();
  await expect(recovery).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "publish-git-recovery-tablet.png");

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByTestId("publish-panel")).toBeVisible();
  await expect(recovery).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "publish-git-recovery-mobile.png");

  await page.getByLabel("Proposal branch").fill("proposal/checkout-ownership-review");
  await page.getByTestId("git-proposal-branch-btn").click();
  await page.getByRole("button", { name: "Create proposal branch" }).click();
  await expect(recovery).toContainText("Git mutation failed: proposal/beta-refresh already has uncommitted local changes");
  expect(commitMessages).toEqual(["docs: publish checkout architecture evidence"]);
  expect(branchNames).toEqual(["proposal/checkout-ownership-review"]);
  await expectNoCriticalAxeViolations(page);
  expect(consoleErrors).toEqual([]);
});
