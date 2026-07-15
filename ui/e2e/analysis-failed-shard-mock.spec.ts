import { expect, test, type Page } from "@playwright/test";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { expectNoCriticalAxeViolations } from "./axe";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const screenshotOutputDir = (process.env.UI_E2E_OUTPUT_DIR ?? "").trim();
const runID = "run-analysis-v2";

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

async function installFailedShardMock(page: Page): Promise<void> {
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const method = request.method().toUpperCase();
    const apiPath = `${url.pathname}${url.search}`;

    if (method === "GET" && url.pathname === "/api/system/version") {
      await route.fulfill({
        ...json({ version: "dev", commit: "mock", built: "mock", ui_bundle: "vite" }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/onboarding/status") {
      await route.fulfill({
        ...json({
          ok: true,
          launcher_mode: false,
          workspace_selected: true,
          workspace_ready: true,
          workspace: "/tmp/failed-shard-workspace",
          manifest_present: true,
          runtime: {
            selected: true,
            runtime: "headless",
            runtime_provider: "qwen-code",
            provider_source: "override",
          },
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
              status: "failed",
              current_step: "init.step1.collect",
              started_at: "2026-04-03T12:00:00Z",
              finished_at: "2026-04-03T12:00:06Z",
              warnings: ["collect warning"],
              error_code: "runtime_contract_failed",
              error: "collect pair recovery stalled before valid artifacts were available",
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
          status: "failed",
          current_step: "init.step1.collect",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: "2026-04-03T12:00:06Z",
          warnings: ["collect warning"],
          error_code: "runtime_contract_failed",
          error: "collect pair recovery stalled before valid artifacts were available",
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}/review-summary`) {
      await route.fulfill({
        ...json({
          run_id: runID,
          pipeline: "init",
          status: "failed",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: "2026-04-03T12:00:06Z",
          current_step: "init.step1.collect",
          warnings: ["collect warning"],
          error_code: "runtime_contract_failed",
          error: "collect pair recovery stalled before valid artifacts were available",
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
              state: "failed",
              provider: "qwen-code",
              artifact_count: 4,
              artifact_paths: [
                "reports/taskruns/run-analysis-v2/staging/shards/payments-root-shard/runtime-execution.json",
                "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/runtime-execution.json",
                "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/invoices-overview.md",
                "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/shard-pack-manifest.json",
              ],
              taskrun_paths: [
                "reports/taskruns/run-analysis-v2/staging/shards/payments-root-shard/runtime-execution.json",
                "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/runtime-execution.json",
              ],
              warnings_count: 1,
              errors_count: 1,
              last_message: "artifact handoff stalled",
            },
          ],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}/artifacts`) {
      await route.fulfill({
        ...json({
          run_id: runID,
          artifacts: [
            {
              path: "reports/taskruns/run-analysis-v2/staging/shards/payments-root-shard/runtime-execution.json",
              kind: "runtime",
              label: "payments runtime execution",
            },
            {
              path: "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/runtime-execution.json",
              kind: "runtime",
              label: "invoices runtime execution",
            },
            {
              path: "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/invoices-overview.md",
              kind: "report",
              label: "invoices overview",
            },
            {
              path: "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/shard-pack-manifest.json",
              kind: "manifest",
              label: "invoices shard manifest",
            },
          ],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}/logs`) {
      await route.fulfill({
        ...json({
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
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/manifest") {
      await route.fulfill({
        ...json({
          content:
            'version: 1\nrepos:\n  - name: "ftgo-application"\n    git_url: "https://github.com/microservices-patterns/ftgo-application.git"\n    ref: "558dfc53b11d30a5f1d995c0c6d58d5106c28189"\ndocs:\n  imports_path: "./docs/imports"\nruntime:\n  selected: headless\n  provider: qwen-code\n',
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/bundle") {
      await route.fulfill({
        ...json({
          ok: true,
          workspace: "/tmp/failed-shard-workspace",
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

    if (method === "GET" && url.pathname === "/api/workspace/health") {
      await route.fulfill({
        ...json({
          version: 1,
          generated_at: "2026-04-03T12:00:07Z",
          status: "pass",
          summary: { info: 0, warning: 0, error: 0 },
          items: [],
        }),
      });
      return;
    }

    if (method === "POST" && url.pathname === "/api/workspace/validate") {
      await route.fulfill({
        ...json({
          ok: true,
          workspace: "/tmp/failed-shard-workspace",
          warnings: [],
          errors: [],
          resolved_repos: [
            {
              name: "ftgo-application",
              source: "https://github.com/microservices-patterns/ftgo-application.git",
              ref: "558dfc5",
              status: "ok",
            },
          ],
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
          effective: {
            strategy: "sequential",
            max_parallel_tasks: 1,
            failure_policy: "best_effort",
            shard_discovery_mode: "heuristics",
          },
          source: { strategy: "workspace" },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/permissions") {
      await route.fulfill({
        ...json({
          ok: true,
          persisted: { mode: "trusted_full_access" },
          effective: { mode: "trusted_full_access", approval_channel: "fail_fast" },
          source: { mode: "workspace" },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/profile") {
      await route.fulfill({
        ...json({
          ok: true,
          permissions: {
            persisted: { mode: "trusted_full_access" },
            effective: { mode: "trusted_full_access", approval_channel: "fail_fast" },
            source: { mode: "workspace" },
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

    if (method === "GET" && url.pathname === "/api/artifacts") {
      const requestedPath = url.searchParams.get("path") ?? "";
      const bodies: Record<string, string> = {
        "charter/overview.md": "# Charter\n\nFTGO failed-shard fixture.\n",
        "reports/taskruns/run-analysis-v2/staging/shards/payments-root-shard/runtime-execution.json":
          '{"task_id":"payments-root-shard","status":"failed","provider":"qwen-code"}',
        "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/runtime-execution.json":
          '{"task_id":"invoices-module-shard","status":"succeeded","provider":"qwen-code"}',
        "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/invoices-overview.md":
          "# Invoices overview\n\nAuthored markdown exists with a matching shard manifest.\n",
        "reports/taskruns/run-analysis-v2/staging/shards/invoices-module-shard/shard-pack-manifest.json":
          '{"version":1,"documents":[{"id":"invoices","path":"invoices-overview.md"}]}',
        "reports/taskruns/run-analysis-v2/raw/payments/collect.json":
          '{"stdout_bytes":0,"stderr_bytes":0,"exit_reason":"stall","validation_error":"runtime_stalled_before_artifacts"}',
      };
      await route.fulfill({
        ...(requestedPath in bodies ? text(bodies[requestedPath]) : text(`# Fixture artifact\n\n${requestedPath || "unknown"}\n`)),
      });
      return;
    }

    await route.fulfill({
      ...json({ ok: true, ignored: apiPath }),
    });
  });
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

test("analysis failed shard mock: artifact handoff recovery remains readable", async ({ page }) => {
  test.skip(scenario !== "analysis-failed-shard-mock", `scenario ${scenario} skips failed shard mock`);

  const consoleErrors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") {
      consoleErrors.push(message.text());
    }
  });
  await installFailedShardMock(page);

  await page.setViewportSize({ width: 1440, height: 980 });
  await page.goto("/");
  await expect(page.getByTestId("product-shell")).toBeVisible();
  await page.getByTestId("stage-analysis").click();
  await expect(page.getByTestId("stage-analysis")).toHaveAttribute("aria-current", "page");

  const recovery = page.getByTestId("analysis-failure-recovery");
  await expect(recovery).toBeVisible();
  await expect(recovery).toContainText("Recovery path");
  await expect(recovery).toContainText("runtime_contract_failed");
  await expect(recovery).toContainText("init.step1.collect");
  await expect(recovery).toContainText("Generated artifacts did not pass validation");

  const liveDiagnostics = page.getByTestId("analysis-live-diagnostics");
  await expect(liveDiagnostics).toContainText("artifact handoff");
  await expect(liveDiagnostics).toContainText("Artifact handoff stalled");
  await expect(liveDiagnostics).toContainText("1/2 ok");
  await expect(liveDiagnostics).toContainText("collect_pair_repair");
  await expect(liveDiagnostics).toContainText("runtime_stalled_before_artifacts");
  await expect(liveDiagnostics).toContainText("Open the failed shard row and raw-output ref");

  const shardTable = page.getByTestId("analysis-shard-table");
  const diagnosticsDrawer = page.getByTestId("runs-diagnostics-drawer");
  await diagnosticsDrawer.locator("summary").click();
  await expect(diagnosticsDrawer).toHaveAttribute("open", "");
  await expect(shardTable).toContainText("payments-root-shard");
  await expect(shardTable).toContainText("Runtime only");
  await expect(shardTable).toContainText("authored markdown and shard-pack-manifest are missing");
  await expect(shardTable).toContainText("invoices-module-shard");
  await expect(shardTable).toContainText("artifact list not loaded for this run");
  await expect(shardTable).not.toContainText("ftgo-applicationRuntime only");

  const drilldown = page.getByTestId("analysis-failed-shard-details");
  await expect(drilldown).toContainText("payments-root-shard");
  await expect(drilldown).toContainText("Runtime record");
  await expect(drilldown).toContainText("Authored markdown");
  await expect(drilldown).toContainText("Manifest");
  await expect(drilldown).toContainText("reports/taskruns/run-analysis-v2/staging/shards/payments-root-shard/runtime-execution.json");
  await expect(drilldown).toContainText("missing");
  await expect(drilldown).not.toContainText("· workspace");
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "analysis-failed-shard-desktop.png");
  await drilldown.scrollIntoViewIfNeeded();
  await expect(drilldown).toBeInViewport();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "analysis-failed-shard-detail-desktop.png");

  await page.setViewportSize({ width: 390, height: 900 });
  await expect(recovery).toBeVisible();
  await expect(liveDiagnostics).toBeVisible();
  await expect(shardTable).toBeVisible();
  await expect(drilldown).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "analysis-failed-shard-mobile.png");

  await expectNoCriticalAxeViolations(page);
  expect(consoleErrors).toEqual([]);
});
