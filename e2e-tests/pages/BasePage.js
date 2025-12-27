const { getConfigFromTestInfo, getTestConfig } = require('../fixtures/test-config');

/**
 * Base Page Object
 * Contains common methods used across all pages
 * Mode-aware: Uses baseURL from test configuration
 */
class BasePage {
  constructor(page, testInfo = null) {
    this.page = page;
    this.testInfo = testInfo;

    // Get baseURL from test config or fall back to page's baseURL
    if (testInfo) {
      const config = getConfigFromTestInfo(testInfo);
      this.baseURL = config.baseURL;
      this.paths = config.paths;
    } else {
      // Fallback: try to get from page context or use default
      this.baseURL = page.context()?.constructor?.name === 'BrowserContext'
        ? 'http://localhost:8080'
        : 'http://localhost:8080';
      this.paths = getTestConfig().paths;
    }
  }

  /**
   * Navigate to a path
   */
  async goto(path) {
    // Use page.goto with relative path when baseURL is set via config
    await this.page.goto(path);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Navigate to a named path (e.g., 'login', 'dashboard')
   */
  async gotoPath(pathName) {
    const path = this.paths[pathName];
    if (!path) {
      throw new Error(`Unknown path: ${pathName}`);
    }
    await this.goto(path);
  }

  /**
   * Wait for navigation to complete
   */
  async waitForNavigation() {
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Get alert message text
   * @param {string} type - alert type (success, error, warning, info)
   */
  async getAlertText(type = 'success') {
    const selector = `.alert-${type}`;
    await this.page.waitForSelector(selector, { timeout: 5000 });
    return await this.page.textContent(selector);
  }

  /**
   * Check if alert exists
   */
  async hasAlert(type = 'success') {
    const selector = `.alert-${type}`;
    return await this.page.locator(selector).isVisible().catch(() => false);
  }

  /**
   * Wait for alert to appear
   */
  async waitForAlert(type = 'success', timeout = 5000) {
    const selector = `.alert-${type}`;
    await this.page.waitForSelector(selector, { timeout });
  }

  /**
   * Click navigation link by text
   */
  async clickNavLink(text) {
    await this.page.click(`nav a:has-text("${text}")`);
    await this.waitForNavigation();
  }

  /**
   * Check if user is logged in
   */
  async isLoggedIn() {
    // Check if dashboard link visible or logout button exists
    const dashboardLink = this.page.locator('a[href="/dashboard.html"]');
    const logoutLink = this.page.locator('a:has-text("Abmelden")');

    const hasDashboard = await dashboardLink.isVisible().catch(() => false);
    const hasLogout = await logoutLink.isVisible().catch(() => false);

    return hasDashboard || hasLogout;
  }

  /**
   * Get current URL
   */
  async getCurrentURL() {
    return this.page.url();
  }

  /**
   * Wait for URL to match pattern
   */
  async waitForURL(pattern, timeout = 5000) {
    await this.page.waitForURL(pattern, { timeout });
  }

  /**
   * Get page title
   */
  async getTitle() {
    return await this.page.title();
  }

  /**
   * Take screenshot (useful for debugging)
   */
  async screenshot(name) {
    await this.page.screenshot({ path: `screenshots/${name}.png` });
  }

  /**
   * Fill form field
   */
  async fill(selector, value) {
    await this.page.fill(selector, value);
  }

  /**
   * Click element
   */
  async click(selector) {
    await this.page.click(selector);
  }

  /**
   * Check if element is visible
   */
  async isVisible(selector) {
    return await this.page.locator(selector).isVisible().catch(() => false);
  }

  /**
   * Get element text content
   */
  async textContent(selector) {
    return await this.page.textContent(selector);
  }

  /**
   * Wait for element
   */
  async waitForSelector(selector, timeout = 10000) {
    await this.page.waitForSelector(selector, { timeout });
  }
}

module.exports = BasePage;
