const { chromium, webkit, firefox } = require('@playwright/test');

/**
 * Authentication fixture
 * Pre-authenticate users to speed up tests
 */

/**
 * Login helper for tests
 * @param {Page} page - Playwright page object
 * @param {string} email - User email
 * @param {string} password - User password (default: test123)
 */
async function login(page, email, password = 'test123') {
  await page.goto('/login.html');
  await page.fill('#email', email);
  await page.fill('#password', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/dashboard.html', { timeout: 5000 });
}

/**
 * Logout helper
 * @param {Page} page - Playwright page object
 */
async function logout(page) {
  // Find and click logout link (in navigation or dropdown)
  await page.click('a[href*="logout"], button:has-text("Abmelden")');
  await page.waitForURL('**/login.html', { timeout: 5000 });
}

/**
 * Setup authenticated admin session
 * Saves auth state to file for reuse
 */
async function setupAdminAuth() {
  console.log('🔐 Setting up admin authentication...');

  let browser;
  try {
    browser = await chromium.launch();
  } catch {
    try {
      browser = await webkit.launch();
    } catch {
      browser = await firefox.launch();
    }
  }

  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    await page.goto('http://localhost:8080/login.html');
    await page.fill('#email', 'admin@tierheim-goeppingen.de');
    await page.fill('#password', 'test123');
    await page.click('button[type="submit"]');

    // Wait for redirect to dashboard
    await page.waitForURL('**/dashboard.html', { timeout: 5000 });
    console.log('   ✅ Admin logged in successfully');

    // Save authenticated state
    await context.storageState({ path: 'admin-storage-state.json' });
    console.log('   ✅ Admin auth state saved');
  } catch (error) {
    console.error('   ❌ Failed to authenticate admin:', error.message);
    throw error;
  } finally {
    await browser.close();
  }
}

/**
 * Check if user is logged in
 * @param {Page} page - Playwright page object
 * @returns {boolean}
 */
async function isLoggedIn(page) {
  // Check if logout link exists
  const logoutLink = page.locator('a:has-text("Abmelden")');
  return await logoutLink.isVisible().catch(() => false);
}

module.exports = {
  login,
  logout,
  setupAdminAuth,
  isLoggedIn,
};

// DONE: Authentication fixture for login helpers
