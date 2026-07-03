import { defineConfig, devices } from '@playwright/test';

// End-to-end UI smoke suite for CivicOS. Assumes the backend stack is
// already running (identity :3001, community :3002, org :3003, gateway
// :3000, Postgres :5433, Redis :6379). Only the Vite dev server on :5173
// is started by Playwright.
export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  fullyParallel: false, // shared DB — one test at a time
  workers: 1,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [['github'], ['list']] : [['list']],
  globalSetup: './e2e/global-setup.ts',
  globalTeardown: './e2e/global-teardown.ts',
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
    video: 'retain-on-failure',
    screenshot: 'only-on-failure',
    // Slower actions are more forgiving of Vite's dev-mode HMR delays.
    actionTimeout: 8_000,
    navigationTimeout: 15_000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'pnpm dev',
    url: 'http://localhost:5173',
    reuseExistingServer: true,
    timeout: 30_000,
    stdout: 'ignore',
    stderr: 'pipe',
  },
});
