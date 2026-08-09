import { defineConfig, devices } from '@playwright/test';

/**
 * E2E runs against the real dockerized stack (nginx on :8081, which proxies
 * /api and /ws to the backend). Override with E2E_BASE_URL for other targets.
 */
export default defineConfig({
  testDir: './e2e',
  timeout: 45_000,
  // Serial only: the backend enforces one active session per user (Redis SSO),
  // so concurrent tests would stomp each other's login sessions.
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['line'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:8081',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
