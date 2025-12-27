// @ts-check
const { defineConfig, devices } = require('@playwright/test');

/**
 * Dual-Mode Playwright Configuration
 * Runs tests against both Simple-Mode and SaaS-Mode servers
 *
 * Usage:
 *   # Run all tests in both modes (requires separate servers)
 *   npx playwright test --config=playwright.dualmode.config.js
 *
 *   # Run only SaaS-Mode tests (against demo.gassigeher.local:8080)
 *   npx playwright test --config=playwright.dualmode.config.js --project=saas-chromium
 *
 *   # Run only Simple-Mode tests (against localhost:8080 without BASE_DOMAIN)
 *   npx playwright test --config=playwright.dualmode.config.js --project=simple-chromium
 */
module.exports = defineConfig({
  testDir: './tests',

  // Test execution settings
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 30 * 1000,

  // Reporting
  reporter: [
    ['html', { outputFolder: 'playwright-report-dualmode' }],
    ['list'],
    ['json', { outputFile: 'test-results-dualmode.json' }],
  ],

  use: {
    // Screenshots and videos for debugging
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    trace: 'retain-on-failure',

    // Timeouts
    actionTimeout: 10 * 1000,
    navigationTimeout: 15 * 1000,

    // Ignore HTTPS errors
    ignoreHTTPSErrors: true,
  },

  // Test projects for each mode
  projects: [
    // ========================================
    // SaaS-Mode Projects (Multi-Tenant)
    // ========================================
    {
      name: 'saas-chromium',
      testMatch: /0[1-5]-.*\.spec\.js$/,  // Tests 01-05
      use: {
        ...devices['Desktop Chrome'],
        baseURL: process.env.SAAS_MODE_URL || 'http://demo.gassigeher.local:8080',
        viewport: { width: 1920, height: 1080 },
      },
    },
    {
      name: 'saas-mobile',
      testMatch: /0[1-5]-.*\.spec\.js$/,
      use: {
        ...devices['iPhone 13'],
        baseURL: process.env.SAAS_MODE_URL || 'http://demo.gassigeher.local:8080',
      },
    },

    // ========================================
    // Simple-Mode Projects (Single-Tenant)
    // Requires server running WITHOUT BASE_DOMAIN
    // ========================================
    {
      name: 'simple-chromium',
      testMatch: /0[1-5]-.*\.spec\.js$/,
      use: {
        ...devices['Desktop Chrome'],
        baseURL: process.env.SIMPLE_MODE_URL || 'http://localhost:8080',
        viewport: { width: 1920, height: 1080 },
      },
    },
    {
      name: 'simple-mobile',
      testMatch: /0[1-5]-.*\.spec\.js$/,
      use: {
        ...devices['iPhone 13'],
        baseURL: process.env.SIMPLE_MODE_URL || 'http://localhost:8080',
      },
    },

    // ========================================
    // SaaS-Only Tests (Billing, Marketing, Central Admin)
    // These only run in SaaS mode
    // ========================================
    {
      name: 'saas-only-chromium',
      testMatch: /1[0-9]-.*\.spec\.js$/,  // Tests 10-19
      use: {
        ...devices['Desktop Chrome'],
        baseURL: process.env.SAAS_MODE_URL || 'http://demo.gassigeher.local:8080',
        viewport: { width: 1920, height: 1080 },
      },
    },
  ],

  // Global setup/teardown disabled for dual-mode
  // Each mode needs different setup
  // globalSetup: require.resolve('./global-setup.js'),
  // globalTeardown: require.resolve('./global-teardown.js'),
});
