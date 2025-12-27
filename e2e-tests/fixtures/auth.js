const { chromium, webkit, firefox } = require('@playwright/test');
const { getTestConfig, getConfigFromTestInfo, TEST_MODES } = require('./test-config');

/**
 * Authentication fixture
 * Mode-aware authentication helpers for dual-mode testing
 */

/**
 * Login helper for tests
 * @param {Page} page - Playwright page object
 * @param {string} email - User email
 * @param {string} password - User password
 * @param {object} testInfo - Playwright test info (optional, for mode detection)
 */
async function login(page, email, password, testInfo = null) {
  const config = testInfo ? getConfigFromTestInfo(testInfo) : getTestConfig();

  await page.goto(config.paths.login);
  await page.fill('#email', email);
  await page.fill('#password', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/dashboard.html', { timeout: 10000 });
}

/**
 * Login with default admin credentials for current mode
 * @param {Page} page - Playwright page object
 * @param {object} testInfo - Playwright test info (for mode detection)
 */
async function loginAsAdmin(page, testInfo) {
  const config = getConfigFromTestInfo(testInfo);
  const { email, password } = config.credentials.admin;

  await login(page, email, password, testInfo);
}

/**
 * Login with specific user type
 * @param {Page} page - Playwright page object
 * @param {string} userType - User type: 'admin', 'greenUser', 'orangeUser', 'blueUser'
 * @param {object} testInfo - Playwright test info
 */
async function loginAs(page, userType, testInfo) {
  const config = getConfigFromTestInfo(testInfo);
  const credentials = config.credentials[userType];

  if (!credentials) {
    throw new Error(`Unknown user type: ${userType}. Available: admin, greenUser, orangeUser, blueUser`);
  }

  await login(page, credentials.email, credentials.password, testInfo);
}

/**
 * Logout helper
 * @param {Page} page - Playwright page object
 */
async function logout(page) {
  // Find and click logout link (in navigation or dropdown)
  const logoutLink = page.locator('a:has-text("Abmelden")');
  if (await logoutLink.isVisible()) {
    await logoutLink.click();
    // Logout redirects to homepage (/), not login page
    await page.waitForLoadState('networkidle');
  }
}

/**
 * Setup authenticated admin session for a specific mode
 * Saves auth state to file for reuse
 * @param {string} mode - Test mode ('simple' or 'saas')
 */
async function setupAdminAuth(mode = TEST_MODES.SAAS) {
  const config = getTestConfig(mode);
  console.log(`🔐 Setting up ${mode} mode admin authentication...`);

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
    const loginURL = `${config.baseURL}${config.paths.login}`;
    await page.goto(loginURL);
    await page.fill('#email', config.credentials.admin.email);
    await page.fill('#password', config.credentials.admin.password);
    await page.click('button[type="submit"]');

    // Wait for redirect to dashboard
    await page.waitForURL('**/dashboard.html', { timeout: 10000 });
    console.log(`   ✅ ${mode} mode admin logged in successfully`);

    // Save authenticated state
    const stateFile = `${mode}-admin-storage-state.json`;
    await context.storageState({ path: stateFile });
    console.log(`   ✅ Admin auth state saved to ${stateFile}`);
  } catch (error) {
    console.error(`   ❌ Failed to authenticate ${mode} admin:`, error.message);
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

/**
 * Get credentials for current test mode
 * @param {object} testInfo - Playwright test info
 * @returns {object} credentials object
 */
function getCredentials(testInfo) {
  const config = getConfigFromTestInfo(testInfo);
  return config.credentials;
}

/**
 * Get base URL for current test mode
 * @param {object} testInfo - Playwright test info
 * @returns {string} base URL
 */
function getBaseURL(testInfo) {
  const config = getConfigFromTestInfo(testInfo);
  return config.baseURL;
}

module.exports = {
  login,
  loginAsAdmin,
  loginAs,
  logout,
  setupAdminAuth,
  isLoggedIn,
  getCredentials,
  getBaseURL,
};
