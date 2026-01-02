// @ts-check
const { defineConfig, devices } = require('@playwright/test');

/**
 * Playwright configuration for Central Admin Impersonation E2E tests
 * Tests cross-domain impersonation flow between central admin and tenant subdomains
 */
module.exports = defineConfig({
  testDir: './tests',
  testMatch: '**/12-central-admin-impersonation.spec.js',

  // Test execution settings
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 60 * 1000,

  // Reporting
  reporter: [
    ['html', { outputFolder: 'playwright-report-impersonation' }],
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
});
