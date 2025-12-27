const { test: setup } = require('@playwright/test');
const { getConfigFromTestInfo } = require('./fixtures/test-config');

/**
 * Auth Setup - Pre-authenticates users and saves sessions
 * This runs before all tests and saves auth state to files
 * Tests can then reuse these sessions without logging in again
 */

// Storage paths for different user types
const STORAGE_DIR = './playwright/.auth';

/**
 * Login with rate limit handling
 * Waits and retries if rate limited
 */
async function loginWithRateLimitHandling(page, email, password, maxRetries = 3) {
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    await page.goto('/login.html');
    await page.waitForLoadState('networkidle');

    // Check for existing rate limit message
    const rateLimitMsg = page.locator('text=Zu viele Anmeldeversuche');
    if (await rateLimitMsg.isVisible().catch(() => false)) {
      console.log(`Rate limit detected before login, waiting 60s (attempt ${attempt + 1}/${maxRetries})...`);
      await page.waitForTimeout(60000);
      continue;
    }

    // Fill and submit
    await page.fill('#email', email);
    await page.fill('#password', password);
    await page.click('button[type="submit"]');

    // Wait for either dashboard or error
    try {
      await page.waitForURL('**/dashboard.html', { timeout: 5000 });
      return true; // Success
    } catch {
      // Check if rate limited after submit
      if (await rateLimitMsg.isVisible().catch(() => false)) {
        console.log(`Rate limited after submit, waiting 60s (attempt ${attempt + 1}/${maxRetries})...`);
        await page.waitForTimeout(60000);
        continue;
      }

      // Some other error - check if we're on dashboard anyway
      if (page.url().includes('dashboard.html')) {
        return true;
      }

      // Login failed for other reason (wrong credentials?)
      const errorMsg = await page.locator('[class*="error"], [class*="alert"]').textContent().catch(() => '');
      console.log(`Login attempt ${attempt + 1} failed: ${errorMsg}`);

      // If it's not a rate limit, don't retry
      if (!errorMsg.includes('Anmeldeversuche')) {
        return false;
      }

      await page.waitForTimeout(60000);
    }
  }
  return false;
}

setup.describe('Authentication Setup', () => {

  setup('authenticate admin user', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const success = await loginWithRateLimitHandling(page, credentials.email, credentials.password);

    if (!success) {
      throw new Error('Admin login failed after retries - check credentials or rate limit');
    }

    // Save storage state
    await page.context().storageState({ path: `${STORAGE_DIR}/admin-${config.mode}.json` });
    console.log(`[Auth Setup] Saved admin session for ${config.mode}`);
  });

  setup('authenticate green user', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const success = await loginWithRateLimitHandling(page, credentials.email, credentials.password);

    if (!success) {
      throw new Error('Green user login failed after retries - check credentials or rate limit');
    }

    await page.context().storageState({ path: `${STORAGE_DIR}/green-${config.mode}.json` });
    console.log(`[Auth Setup] Saved green user session for ${config.mode}`);
  });

});
