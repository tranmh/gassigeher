const BasePage = require('./BasePage');

// Global rate limit tracker - 5 logins per minute max (server-side limit for login endpoint)
// Match server: track per 60-second window
const loginTimestamps = [];
const RATE_LIMIT_WINDOW = 60000; // 60 seconds = 1 minute
const MAX_LOGINS_PER_WINDOW = 5; // Server allows 5, we use full allowance

/**
 * Login Page Object
 * Mode-aware: Uses baseURL from test configuration
 * Rate-limit aware: Adds delays to avoid server rate limiting
 */
class LoginPage extends BasePage {
  constructor(page, testInfo = null) {
    super(page, testInfo);

    // Selectors
    this.emailInput = '#email';
    this.passwordInput = '#password';
    this.submitButton = 'button[type="submit"]';
    this.errorAlert = '.alert-error';
    this.successAlert = '.alert-success';
    this.registerLink = 'a[href="/register.html"], a[href="register.html"]';
    this.forgotPasswordLink = 'a[href="/forgot-password.html"], a[href="forgot-password.html"]';
  }

  /**
   * Wait for rate limit window if needed
   */
  async waitForRateLimit() {
    const now = Date.now();

    // Remove old timestamps outside the window
    while (loginTimestamps.length > 0 && loginTimestamps[0] < now - RATE_LIMIT_WINDOW) {
      loginTimestamps.shift();
    }

    console.log(`[Rate Limit] Current logins in window: ${loginTimestamps.length}/${MAX_LOGINS_PER_WINDOW}`);

    // If we're at the limit, wait until the oldest login expires
    if (loginTimestamps.length >= MAX_LOGINS_PER_WINDOW) {
      const waitTime = loginTimestamps[0] + RATE_LIMIT_WINDOW - now + 2000; // Add 2s buffer
      if (waitTime > 0) {
        console.log(`[Rate Limit] Waiting ${Math.ceil(waitTime/1000)}s before next login...`);
        await this.page.waitForTimeout(waitTime);
        // Clean up timestamps after waiting
        while (loginTimestamps.length > 0 && loginTimestamps[0] < Date.now() - RATE_LIMIT_WINDOW) {
          loginTimestamps.shift();
        }
      }
    }

    // Record this login attempt
    loginTimestamps.push(Date.now());
    console.log(`[Rate Limit] Login recorded. Total in window: ${loginTimestamps.length}`);
  }

  /**
   * Navigate to login page
   */
  async goto() {
    await super.goto('/login.html');
  }

  /**
   * Login with credentials (rate-limit aware)
   */
  async login(email, password) {
    // Wait for rate limit window if needed
    await this.waitForRateLimit();

    await this.page.fill(this.emailInput, email);
    await this.page.fill(this.passwordInput, password);
    await this.page.click(this.submitButton);
  }

  /**
   * Login and wait for redirect to dashboard
   * If using storageState (pre-authenticated), navigates directly to dashboard
   */
  async loginAndWait(email, password) {
    // First, ensure we're on a page in our domain (needed to check localStorage)
    const currentURL = this.page.url();
    if (!currentURL.includes('gassigeher') && !currentURL.includes('localhost')) {
      await this.goto();
    }

    // Check if already on dashboard
    if (currentURL.includes('dashboard.html')) {
      return;
    }

    // Check if we have a valid token in localStorage (from storageState)
    const hasToken = await this.page.evaluate(() => {
      return !!localStorage.getItem('gassigeher_token');
    }).catch(() => false);

    if (hasToken) {
      // Already authenticated via storageState, go directly to dashboard
      await this.page.goto('/dashboard.html');
      await this.page.waitForLoadState('networkidle');
      // Verify we're on dashboard (not redirected to login)
      const finalURL = this.page.url();
      if (finalURL.includes('dashboard.html')) {
        return;
      }
      // Token might be expired, fall through to login
    }

    // Not authenticated or token expired, perform login
    await this.goto(); // Go to login page
    await this.login(email, password);
    // Login has 1-second delay before redirect, so wait longer
    await this.page.waitForURL('**/dashboard.html', { timeout: 20000 });
  }

  /**
   * Get error message
   */
  async getErrorMessage() {
    await this.page.waitForSelector(this.errorAlert, { timeout: 5000 });
    return await this.page.textContent(this.errorAlert);
  }

  /**
   * Check if error is visible
   */
  async hasError() {
    return await this.page.locator(this.errorAlert).isVisible().catch(() => false);
  }

  /**
   * Click register link
   */
  async goToRegister() {
    await this.page.locator(this.registerLink).first().click();
    await this.page.waitForURL('**/register.html');
  }

  /**
   * Click forgot password link
   */
  async goToForgotPassword() {
    await this.page.click(this.forgotPasswordLink);
    await this.page.waitForURL('**/forgot-password.html');
  }
}

module.exports = LoginPage;
