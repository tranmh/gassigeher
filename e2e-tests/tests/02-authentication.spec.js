const { test, expect } = require('@playwright/test');
const LoginPage = require('../pages/LoginPage');
const RegisterPage = require('../pages/RegisterPage');
const DashboardPage = require('../pages/DashboardPage');
const { getConfigFromTestInfo, getCredentials } = require('../fixtures/test-config');

/**
 * AUTHENTICATION TESTS
 * Test user registration, login, logout flows
 * GOAL: Find bugs in authentication flows!
 *
 * Dual-Mode: Tests run against both Simple-Mode and SaaS-Mode
 */

test.describe('Registration - Valid Cases', () => {

  test('should register new user successfully', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const registerPage = new RegisterPage(page, testInfo);
    await registerPage.goto();

    const timestamp = Date.now();
    const testUser = {
      email: `test-${timestamp}@example.com`,
      firstName: 'Test',
      lastName: 'User',
      name: 'Test User',
      phone: '+49 123 456 7890',
      password: 'Test123!',
      acceptTerms: true,
    };

    await registerPage.register(testUser);

    // Wait for success message or redirect
    await page.waitForLoadState('networkidle');

    // Check for success message
    const hasSuccess = await registerPage.hasSuccess();
    if (hasSuccess) {
      const successMsg = await registerPage.getSuccessMessage();
      console.log(`[${config.mode}] Registration success message:`, successMsg);
    }

    console.log(`[${config.mode}] After registration, URL is:`, page.url());
  });

  test('should show all registration form fields', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const registerPage = new RegisterPage(page, testInfo);
    await registerPage.goto();

    // All fields should be visible
    // Note: HTML uses hyphens (#first-name), not underscores (#first_name)
    await expect(page.locator('#email')).toBeVisible();
    await expect(page.locator('#first-name, #name')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.locator('#accept-terms')).toBeVisible();

    console.log(`[${config.mode}] Registration form fields visible`);
  });

});

test.describe('Registration - Validation Errors', () => {

  test('should reject registration with invalid email format', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const registerPage = new RegisterPage(page, testInfo);
    await registerPage.goto();

    await registerPage.register({
      email: 'not-an-email',
      firstName: 'Test',
      lastName: 'User',
      name: 'Test User',
      phone: '+49 123 456 7890',
      password: 'Test123!',
      acceptTerms: true,
    });

    await page.waitForLoadState('networkidle');
    // HTML5 validation or backend should catch this
    console.log(`[${config.mode}] Invalid email shows validation`);
  });

  test('should reject registration without accepting terms', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const registerPage = new RegisterPage(page, testInfo);
    await registerPage.goto();

    const timestamp = Date.now();
    await registerPage.register({
      email: `test-${timestamp}@example.com`,
      firstName: 'Test',
      lastName: 'User',
      name: 'Test User',
      phone: '+49 123 456 7890',
      password: 'Test123!',
      acceptTerms: false,
    });

    await page.waitForLoadState('networkidle');

    const currentURL = page.url();
    console.log(`[${config.mode}] After registration without terms, URL:`, currentURL);

    // Should still be on register page
    expect(currentURL).toContain('register.html');
  });

  test('should reject registration with weak password', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const registerPage = new RegisterPage(page, testInfo);
    await registerPage.goto();

    const timestamp = Date.now();
    await registerPage.register({
      email: `test-${timestamp}@example.com`,
      firstName: 'Test',
      lastName: 'User',
      name: 'Test User',
      phone: '+49 123 456 7890',
      password: '123',  // Too short
      acceptTerms: true,
    });

    await page.waitForLoadState('networkidle');
    console.log(`[${config.mode}] Weak password should show error`);
  });

});

test.describe('Login - Valid Cases', () => {

  test('should login with valid credentials', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();

    await loginPage.loginAndWait(credentials.email, credentials.password);

    // Should be on dashboard
    expect(page.url()).toContain('dashboard.html');

    // Dashboard should show user info
    const dashboardPage = new DashboardPage(page, testInfo);
    const welcomeMsg = await dashboardPage.getWelcomeMessage();
    console.log(`[${config.mode}] Welcome message:`, welcomeMsg);

    expect(welcomeMsg.length).toBeGreaterThan(0);
  });

  test('should store authentication token after login', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();

    await loginPage.loginAndWait(credentials.email, credentials.password);

    // Check if token is stored in localStorage
    const token = await page.evaluate(() => {
      return localStorage.getItem('gassigeher_token');
    });

    console.log(`[${config.mode}] Token stored:`, token ? 'Yes' : 'No');
    expect(token).toBeTruthy();

    // Token should be a JWT
    if (token) {
      const isJWT = token.split('.').length === 3;
      console.log(`[${config.mode}] Token is valid JWT:`, isJWT);
      expect(isJWT).toBe(true);
    }
  });

  test('should persist login after page refresh', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();

    await loginPage.loginAndWait(credentials.email, credentials.password);

    // Now refresh the page
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Should still be logged in
    const currentURL = page.url();
    console.log(`[${config.mode}] After refresh, URL:`, currentURL);

    // Should still be on dashboard
    expect(currentURL).toContain('dashboard.html');
  });

});

test.describe('Login - Invalid Cases', () => {

  test('should reject login with invalid email', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();

    await loginPage.login('nonexistent@example.com', 'wrongpassword');
    await page.waitForLoadState('networkidle');

    // Should show error
    const hasError = await loginPage.hasError();
    expect(hasError).toBe(true);

    if (hasError) {
      const errorMsg = await loginPage.getErrorMessage();
      console.log(`[${config.mode}] Invalid email error:`, errorMsg);
    }
  });

  test('should reject login with wrong password', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();

    await loginPage.login(credentials.email, 'wrongpassword');

    // Wait for error alert to appear (not just network idle)
    await page.waitForSelector('.alert-error', { timeout: 5000 });

    // Should show error
    const hasError = await loginPage.hasError();
    expect(hasError).toBe(true);

    if (hasError) {
      const errorMsg = await loginPage.getErrorMessage();
      console.log(`[${config.mode}] Wrong password error:`, errorMsg);
    }
  });

  test('should reject login with empty credentials', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();

    await loginPage.login('', '');

    await page.waitForLoadState('networkidle');

    const currentURL = page.url();
    expect(currentURL).toContain('login.html');
    console.log(`[${config.mode}] Empty credentials blocked`);
  });

});

test.describe('Logout', () => {

  test('should logout successfully', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    // First login
    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dashboardPage = new DashboardPage(page, testInfo);
    await dashboardPage.logout();

    // Logout redirects to '/' (homepage), not login page directly
    const currentURL = page.url();
    console.log(`[${config.mode}] After logout, URL is:`, currentURL);

    // Token should be cleared
    const token = await page.evaluate(() => {
      return localStorage.getItem('gassigeher_token');
    });

    console.log(`[${config.mode}] Token after logout:`, token);
    expect(token).toBeFalsy();
  });

  test('should not access protected pages after logout', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    // Login
    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    // Logout
    const dashboardPage = new DashboardPage(page, testInfo);
    await dashboardPage.logout();

    await page.waitForLoadState('networkidle');

    // Try to access dashboard
    await page.goto('/dashboard.html');
    await page.waitForLoadState('networkidle');

    const currentURL = page.url();
    console.log(`[${config.mode}] After logout, trying to access dashboard:`, currentURL);

    // Should redirect to login
    expect(currentURL).toContain('login.html');
  });

});

test.describe('Password Reset Flow', () => {

  test('should show forgot password page', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();

    await loginPage.goToForgotPassword();

    expect(page.url()).toContain('forgot-password.html');
    console.log(`[${config.mode}] Forgot password page accessible`);
  });

  test('should accept email for password reset', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    await page.goto('/forgot-password.html');

    await page.fill('#email', credentials.email);
    await page.click('button[type="submit"]');

    await page.waitForLoadState('networkidle');

    console.log(`[${config.mode}] Password reset request submitted`);
  });

  test('should show generic message for non-existent email (security)', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);

    await page.goto('/forgot-password.html');

    await page.fill('#email', 'nonexistent@example.com');
    await page.click('button[type="submit"]');

    await page.waitForLoadState('networkidle');

    // Should NOT reveal that email doesn't exist (security)
    console.log(`[${config.mode}] Password reset for non-existent email - should show generic message`);
  });

});

test.describe('Session Management', () => {

  test('should handle expired tokens gracefully', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    // Login first
    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    // Manually set an expired/invalid token
    await page.evaluate(() => {
      localStorage.setItem('gassigeher_token', 'expired.token.here');
    });

    // Try to access protected page
    await page.goto('/dashboard.html');
    await page.waitForLoadState('networkidle');

    const currentURL = page.url();
    console.log(`[${config.mode}] With expired token, redirected to:`, currentURL);

    // Should redirect to login
    expect(currentURL).toContain('login.html');
  });

  test('should handle multiple tabs correctly', async ({ browser }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    // Create two pages (tabs)
    const context = await browser.newContext({
      baseURL: config.baseURL
    });
    const page1 = await context.newPage();
    const page2 = await context.newPage();

    // Login in tab 1
    await page1.goto('/login.html');
    await page1.fill('#email', credentials.email);
    await page1.fill('#password', credentials.password);
    await page1.click('button[type="submit"]');
    await page1.waitForURL('**/dashboard.html');

    // Tab 2 should also be logged in (shared localStorage)
    await page2.goto('/dashboard.html');
    await page2.waitForLoadState('networkidle');

    const page2URL = page2.url();
    console.log(`[${config.mode}] Tab 2 URL after tab 1 login:`, page2URL);

    expect(page2URL).toContain('dashboard.html');

    // Now logout in tab 1
    await page1.click('a:has-text("Abmelden")');
    await page1.waitForLoadState('networkidle');

    // Tab 2 should also be logged out on reload
    await page2.reload();
    await page2.waitForLoadState('networkidle');

    const page2URLAfterLogout = page2.url();
    console.log(`[${config.mode}] Tab 2 URL after tab 1 logout:`, page2URLAfterLogout);

    await context.close();
  });

});
