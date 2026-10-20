import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  reporter: "list",
  use: {
    baseURL: "http://127.0.0.1:9250",
    channel: "chrome",
    headless: true,
    viewport: { width: 1280, height: 800 },
    trace: "retain-on-failure",
  },
  expect: {
    toHaveScreenshot: {
      maxDiffPixelRatio: 0.01,
      animations: "disabled",
    },
  },
  webServer: {
    command: "npm run dev -- --port 9250",
    url: "http://127.0.0.1:9250/e2e.html",
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
