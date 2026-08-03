import { expect, test, type Page } from "@playwright/test";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { expectNoCriticalAxeViolations } from "./axe";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const screenshotOutputDir = (process.env.UI_E2E_OUTPUT_DIR ?? "").trim();
const runID = "run-provider-stream";

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

async function installProviderStreamMock(page: Page): Promise<void> {
  const streamPayload = JSON.stringify({
    type: "stream_event",
    event: {
      type: "content_block_delta",
      delta: {
        type: "thinking_delta",
        thinking: "checking repository structure before collect artifact writes and enumerating source files",
      },
    },
  });
  const textStreamPayload = JSON.stringify({
    type: "stream_event",
    event: {
      type: "content_block_delta",
      delta: {
        type: "text_delta",
        text: "drafting collect artifacts",
      },
    },
  });

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
          workspace: "/tmp/provider-stream-workspace",
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
              status: "running",
              current_step: "init.step1.collect",
              started_at: "2026-04-03T12:00:00Z",
              finished_at: null,
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
          status: "running",
          current_step: "init.step1.collect",
          started_at: "2026-04-03T12:00:00Z",
          finished_at: null,
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
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}/artifacts`) {
      await route.fulfill({ ...json({ run_id: runID, artifacts: [] }) });
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
              level: "info",
              kind: "event",
              step_id: "init.step1.collect",
              domain_id: "posthog",
              message: "runtime task started",
              fields: { provider: "qwen-code", shard_id: "posthog-root-shard", shards_total: 16 },
            },
            {
              cursor: 2,
              timestamp: "2026-04-03T12:00:02Z",
              level: "info",
              kind: "runtime_output",
              stream: "stdout",
              step_id: "init.step1.collect",
              message: streamPayload,
            },
            {
              cursor: 3,
              timestamp: "2026-04-03T12:00:03Z",
              level: "info",
              kind: "runtime_output",
              stream: "stdout",
              step_id: "init.step1.collect",
              message: textStreamPayload,
            },
          ],
          next_cursor: 3,
          eof: false,
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/manifest") {
      await route.fulfill({
        ...json({
          content:
            'version: 1\nrepos:\n  - name: "posthog"\n    path: "/tmp/provenarch-live-e2e/posthog/posthog"\ndocs:\n  imports_path: "./docs/imports"\nruntime:\n  selected: headless\n  provider: qwen-code\n',
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/bundle") {
      await route.fulfill({
        ...json({
          ok: true,
          workspace: "/tmp/provider-stream-workspace",
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
          workspace: "/tmp/provider-stream-workspace",
          warnings: [],
          errors: [],
          resolved_repos: [
            {
              name: "posthog",
              source: "/tmp/provenarch-live-e2e/posthog/posthog",
              ref: "14d29a5",
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
      await route.fulfill({
        ...(requestedPath === "charter/overview.md" ? text("# Charter\n\nProvider stream fixture.\n") : text("", 404)),
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

test("provider stream mock: Analysis diagnostics remain readable", async ({ page }) => {
  test.skip(scenario !== "provider-stream-mock", `scenario ${scenario} skips provider stream mock`);

  const consoleErrors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") {
      consoleErrors.push(message.text());
    }
  });
  await installProviderStreamMock(page);

  await page.setViewportSize({ width: 1440, height: 980 });
  await page.goto("/runs");
  await expect(page.getByTestId("product-shell")).toBeVisible();
  await expect(page.getByTestId("destination-runs")).toHaveAttribute("aria-current", "page");
  await page.getByRole("button", { name: runID }).click();
  await expect(page).toHaveURL(`/runs/${runID}`);

  const liveDiagnostics = page.getByTestId("analysis-live-diagnostics");
  await expect(liveDiagnostics).toBeVisible();
  await expect(liveDiagnostics).toContainText("provider stream");
  await expect(liveDiagnostics).toContainText("Run signal");
  await expect(liveDiagnostics).toContainText("Artifact pair pending");
  await expect(liveDiagnostics).toContainText("2 JSON stream events");

  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "provider-stream-desktop.png");

  await page.setViewportSize({ width: 390, height: 900 });
  await expect(liveDiagnostics).toBeVisible();
  await expect(liveDiagnostics).toContainText("thinking_delta, text_delta");
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "provider-stream-mobile.png");

  await expectNoCriticalAxeViolations(page);
  expect(consoleErrors).toEqual([]);
});
