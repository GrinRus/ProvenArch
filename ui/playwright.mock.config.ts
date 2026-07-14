import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.UI_E2E_BASE_URL ?? "http://127.0.0.1:18180";
const outputDir = process.env.UI_E2E_PLAYWRIGHT_OUTPUT_DIR ?? "/tmp/provenarch-ui-mock-e2e/test-results";
const webServerCommand = process.env.UI_E2E_WEB_SERVER_COMMAND ?? "npm run dev -- --host 127.0.0.1 --port 18180";
const reuseExistingServer = process.env.CI ? false : true;

export default defineConfig({
  testDir: "./e2e",
  testMatch: "**/*mock.spec.ts",
  outputDir,
  timeout: 120 * 1000,
  expect: {
    timeout: 10 * 1000,
  },
  retries: 0,
  workers: 1,
  reporter: [["list"]],
  webServer: {
    command: webServerCommand,
    url: baseURL,
    reuseExistingServer,
    timeout: 120 * 1000,
    stdout: "pipe",
    stderr: "pipe",
  },
  use: {
    baseURL,
    actionTimeout: 15 * 1000,
    navigationTimeout: 20 * 1000,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "off",
  },
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        viewport: { width: 1440, height: 980 },
      },
    },
  ],
});
