// @ts-check
const { defineConfig, devices } = require('@playwright/test');

/**
 * Playwright configuration for SaaS billing E2E tests
 * This config doesn't require database access - uses API calls only
 */
module.exports = defineConfig({
  testDir: './tests',
  testMatch: '**/10-saas-billing.spec.js',

  // Test execution settings
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 60 * 1000,

  // Reporting
  reporter: [
    ['html', { outputFolder: 'playwright-report-saas' }],
    ['list'],
  ],

  use: {
    // Base URL - uses gassigeher.local for SaaS mode
    baseURL: 'http://gassigeher.local:8080',

    // Browser options
    headless: process.env.CI ? true : false,
    viewport: { width: 1920, height: 1080 },

    // Screenshots and videos
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    trace: 'retain-on-failure',

    // Timeouts
    actionTimeout: 15 * 1000,
    navigationTimeout: 20 * 1000,

    // Ignore HTTPS errors
    ignoreHTTPSErrors: true,
  },

  // Only test on Chromium for faster execution
  projects: [
    {
      name: 'chromium-desktop',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1920, height: 1080 },
      },
    },
  ],

  // No global setup/teardown - these tests use API calls
  // globalSetup: undefined,
  // globalTeardown: undefined,
});
