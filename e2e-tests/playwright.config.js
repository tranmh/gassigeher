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
  timeout: 90 * 1000,    // 90s per test (allows for rate limit waits)

  // Reporting
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],  // Console output
    ['json', { outputFile: 'test-results.json' }],
  ],

  use: {
    // Base URL for all tests
    baseURL: process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:8080',

    // Browser options - always headless for stability
    headless: true,
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

  // Test projects - All tests run against demo tenant in SaaS-Mode
  // Server runs in SaaS mode with demo tenant at demo.gassigeher.local:8080
  projects: [
    // Auth setup - runs first to create authenticated sessions
    {
      name: 'auth-setup',
      testDir: '.', // Look in root, not tests/
      testMatch: /auth\.setup\.js/,
      // Longer timeout for auth setup due to potential rate limit waits
      timeout: 240 * 1000, // 4 minutes per test (allows for rate limit retries)
      use: {
        ...devices['Desktop Chrome'],
        baseURL: process.env.SAAS_MODE_URL || 'http://demo.gassigeher.local:8080',
      },
    },
    // Desktop Chrome - primary testing (uses admin auth)
    {
      name: 'simple-chromium',
      dependencies: ['auth-setup'],
      // Exclude marketing tests - they have their own config (playwright.marketing.config.js)
      testIgnore: '**/11-marketing.spec.js',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1920, height: 1080 },
        baseURL: process.env.SAAS_MODE_URL || 'http://demo.gassigeher.local:8080',
        // Use pre-authenticated admin session (saas mode)
        storageState: './playwright/.auth/admin-saas.json',
      },
    },
    // SaaS-Mode: Same as simple-chromium but explicit
    {
      name: 'saas-chromium',
      dependencies: ['auth-setup'],
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1920, height: 1080 },
        baseURL: process.env.SAAS_MODE_URL || 'http://demo.gassigeher.local:8080',
        // Use pre-authenticated admin session
        storageState: './playwright/.auth/admin-saas.json',
      },
    },
    // Mobile projects (also use demo tenant, with admin auth)
    {
      name: 'simple-mobile-iphone',
      dependencies: ['auth-setup'],
      use: {
        ...devices['iPhone 13'],
        baseURL: process.env.SAAS_MODE_URL || 'http://demo.gassigeher.local:8080',
        storageState: './playwright/.auth/admin-saas.json',
      },
    },
    {
      name: 'simple-mobile-android',
      dependencies: ['auth-setup'],
      use: {
        ...devices['Pixel 5'],
        baseURL: process.env.SAAS_MODE_URL || 'http://demo.gassigeher.local:8080',
        storageState: './playwright/.auth/admin-saas.json',
      },
    },
    // Billing tests - no auth-setup dependency, uses API authentication
    {
      name: 'billing-tests',
      testMatch: /10-saas-billing\.spec\.js/,
      use: {
        ...devices['Desktop Chrome'],
        baseURL: 'http://gassigeher.local:8080',
        // No storageState - billing tests manage their own auth
      },
    },
  ],

  // Note: Start Go server manually before running tests
  // Set these environment variables:
  // DATABASE_PATH=./e2e-tests/test.db
  // PORT=8080
  // JWT_SECRET=test-jwt-secret-for-e2e-only-do-not-use-in-production
  // SUPER_ADMIN_EMAIL=admin@tierheim-goeppingen.de
  // SKIP_SEED=true (required! global-setup.js manages test data)
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
  globalSetup: require.resolve('./global-setup.js'),
  globalTeardown: require.resolve('./global-teardown.js'),
});

// DONE: Playwright configuration created with desktop + mobile projects
