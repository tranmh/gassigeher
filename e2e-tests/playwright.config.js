// @ts-check
const { defineConfig, devices } = require('@playwright/test');

/**
 * Playwright configuration for Gassigeher E2E tests
 * See: https://playwright.dev/docs/test-configuration
 */
module.exports = defineConfig({
  testDir: './tests',

  // Test execution settings
  fullyParallel: false,  // Run sequentially for easier debugging
  workers: 1,            // One worker = sequential execution
  retries: 0,            // No retries locally (fast feedback)
  timeout: 30 * 1000,    // 30s per test

  // Reporting
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],  // Console output
    ['json', { outputFile: 'test-results.json' }],
  ],

  use: {
    // Base URL for all tests
    baseURL: process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:8080',

    // Browser options
    headless: process.env.CI ? true : false,  // See browser during local dev
    viewport: { width: 1920, height: 1080 },

    // Screenshots and videos
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    trace: 'retain-on-failure',

    // Timeouts
    actionTimeout: 10 * 1000,
    navigationTimeout: 15 * 1000,

    // Ignore HTTPS errors (for local dev)
    ignoreHTTPSErrors: true,
  },

  // Test projects (browsers/devices)
  projects: [
    {
      name: 'chromium-desktop',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1920, height: 1080 },
      },
    },
    {
      name: 'mobile-iphone',
      use: {
        ...devices['iPhone 13'],
      },
    },
    {
      name: 'mobile-android',
      use: {
        ...devices['Pixel 5'],
      },
    },
  ],

  // Note: Start Go server manually before running tests
  // Set these environment variables:
  // DATABASE_PATH=./e2e-tests/test.db
  // PORT=8080
  // JWT_SECRET=test-jwt-secret-for-e2e-only-do-not-use-in-production
  // SUPER_ADMIN_EMAIL=admin@test.com
  // Run: ./gassigeher.exe in separate terminal

  // webServer disabled for local testing - start server manually
  /* webServer: {
    command: 'cd .. && start /B .\\gassigeher.exe',
    url: 'http://localhost:8080',
    reuseExistingServer: true,
    timeout: 30 * 1000,
    env: {
      DATABASE_PATH: './e2e-tests/test.db',
      PORT: '8080',
      JWT_SECRET: 'test-jwt-secret-for-e2e-only-do-not-use-in-production',
      SUPER_ADMIN_EMAIL: 'admin@test.com',
      UPLOAD_DIR: './e2e-tests/test-uploads',
      GMAIL_CLIENT_ID: '',
      GMAIL_CLIENT_SECRET: '',
      GMAIL_REFRESH_TOKEN: '',
      GMAIL_FROM_EMAIL: '',
    },
  }, */

  // Global setup/teardown - DISABLED FOR NOW
  // Run tests against existing server with existing database
  // Start server manually: go run cmd/server/main.go OR ./gassigeher.exe
  // globalSetup: require.resolve('./global-setup.js'),
  // globalTeardown: require.resolve('./global-teardown.js'),
});

// DONE: Playwright configuration created with desktop + mobile projects
