import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  use: { baseURL: "http://localhost:4173", headless: true },
  // WebKit runs alongside Chromium: its animation/CSP behavior diverges enough
  // to have shipped iOS-only breakage (stuck wordmark rects, no SVG favicon).
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "mobile-safari", use: { ...devices["iPhone 15 Pro"] } },
  ],
  webServer: {
    command: process.env.SMOKE_DIST ? "PORT=4173 bun tests/serve-dist.ts" : "PORT=4173 bun index.html",
    url: "http://localhost:4173",
    reuseExistingServer: !process.env.CI,
  },
});
