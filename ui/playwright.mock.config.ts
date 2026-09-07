import { defineConfig, devices } from "@playwright/test";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const baseURL = process.env.UI_E2E_BASE_URL;
if (!baseURL) {
  throw new Error("Use npm run e2e:mock to allocate an isolated server, or set UI_E2E_BASE_URL explicitly.");
}
const serverURL = new URL(baseURL);
const outputDir = process.env.UI_E2E_PLAYWRIGHT_OUTPUT_DIR ?? mkdtempSync(join(tmpdir(), "provenarch-ui-mock-results-"));
const shellQuote = (value: string) => `'${value.replace(/'/g, "'\\''")}'`;
const serverPort = serverURL.port || (serverURL.protocol === "https:" ? "443" : "80");
const webServerCommand = process.env.UI_E2E_WEB_SERVER_COMMAND ?? `npm run dev -- --host ${shellQuote(serverURL.hostname)} --port ${shellQuote(serverPort)} --strictPort`;

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
    reuseExistingServer: false,
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
