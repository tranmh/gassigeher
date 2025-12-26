// @ts-check
const { defineConfig, devices } = require('@playwright/test');

/**
 * Minimal Playwright config for marketing tests
 * Runs against existing server without global setup
 */
module.exports = defineConfig({
  testDir: './tests',
  testMatch: '11-marketing.spec.js',

  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 30 * 1000,

  reporter: [['list']],

  use: {
    baseURL: 'http://gassigeher.local:8080',
    headless: true,
    viewport: { width: 1920, height: 1080 },
    screenshot: 'only-on-failure',
    actionTimeout: 10 * 1000,
    navigationTimeout: 15 * 1000,
    ignoreHTTPSErrors: true,
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // No global setup/teardown - use existing server
});
