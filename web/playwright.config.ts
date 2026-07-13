import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  use: { baseURL: "http://localhost:4173", headless: true },
  webServer: {
    command: process.env.SMOKE_DIST ? "PORT=4173 bun tests/serve-dist.ts" : "PORT=4173 bun index.html",
    url: "http://localhost:4173",
    reuseExistingServer: !process.env.CI,
  },
});
