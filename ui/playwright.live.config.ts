import { defineConfig } from "@playwright/test";

const baseURL = process.env.UI_E2E_BASE_URL ?? "http://127.0.0.1:18080";
const outputDir = process.env.UI_E2E_OUTPUT_DIR ?? "/tmp/provenarch-ui-e2e/test-results";

export default defineConfig({
  testDir: "./e2e",
  outputDir,
  timeout: 6 * 60 * 1000,
  expect: {
    timeout: 60 * 1000
  },
  retries: 0,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure"
  }
});
