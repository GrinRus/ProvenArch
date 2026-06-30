import { defineConfig } from "@playwright/test";

const baseURL = process.env.UI_E2E_BASE_URL ?? "http://127.0.0.1:18080";
const outputDir = process.env.UI_E2E_OUTPUT_DIR ?? "/tmp/provenarch-ui-e2e/test-results";
const initTimeoutSecRaw = Number.parseInt(process.env.ACP_UI_INIT_POLL_TIMEOUT_SEC ?? "900", 10);
const initTimeoutSec = Number.isFinite(initTimeoutSecRaw) && initTimeoutSecRaw > 0 ? initTimeoutSecRaw : 900;
const testTimeoutSec = Math.max(360, initTimeoutSec + 120);

export default defineConfig({
  testDir: "./e2e",
  outputDir,
  timeout: testTimeoutSec * 1000,
  expect: {
    timeout: 60 * 1000
  },
  retries: 0,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL,
    actionTimeout: 60 * 1000,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure"
  }
});
