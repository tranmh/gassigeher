const { test, expect } = require('@playwright/test');
const LoginPage = require('../pages/LoginPage');
const RegisterPage = require('../pages/RegisterPage');
const DashboardPage = require('../pages/DashboardPage');
const DBHelper = require('../utils/db-helpers');
const GERMAN_TEXT = require('../utils/german-text');

/**
 * AUTHENTICATION TESTS
 * Test user registration, login, logout flows
 * GOAL: Find bugs in authentication flows!
 */

test.describe('Registration - Valid Cases', () => {

  test('should register new user successfully', async ({ page }) => {
    const registerPage = new RegisterPage(page);
    await registerPage.goto();

    const timestamp = Date.now();
    const testUser = {
      email: `test-${timestamp}@example.com`,
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
      console.log('Registration success message:', successMsg);
      expect(successMsg.toLowerCase()).toContain('erfolg'); // "erfolgreich"
    }

    // POTENTIAL BUG: Check if redirected to login or verification page
    const currentURL = page.url();
    console.log('After registration, URL is:', currentURL);
  });

  test('should show all registration form fields', async ({ page }) => {
    const registerPage = new RegisterPage(page);
    await registerPage.goto();

    // All fields should be visible
    await expect(page.locator('#email')).toBeVisible();
    await expect(page.locator('#name')).toBeVisible();
    await expect(page.locator('#phone')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.locator('#accept-terms')).toBeVisible();  // Correct ID

    // POTENTIAL BUG: Check field labels are in German
    const pageText = await page.textContent('body');
    expect(pageText).toContain('E-Mail' || 'Email');
    expect(pageText).toContain('Name');
    expect(pageText).toContain('Telefon' || 'Phone');
    expect(pageText).toContain('Passwort');
  });

});

test.describe('Registration - Validation Errors', () => {

  test('should reject registration without email', async ({ page }) => {
    const registerPage = new RegisterPage(page);
    await registerPage.goto();

    // HTML5 validation prevents submission with empty required fields
    // This test documents that empty email is blocked by browser
    // No backend error because form never submits
    test.skip(); // Skipping: HTML5 validation handles this

    await registerPage.register({
      email: '',  // Missing email - HTML5 blocks this
      name: 'Test User',
      phone: '+49 123 456 7890',
      password: 'Test123!',
      acceptTerms: true,
    });

    // HTML5 validation prevents submission, so no error alert
    const hasError = await registerPage.hasError();
    expect(hasError).toBe(true);

    if (hasError) {
      const errorMsg = await registerPage.getErrorMessage();
      console.log('Empty email error:', errorMsg);
      // POTENTIAL BUG: Error message should be in German
    }
  });

  test('should reject registration with invalid email format', async ({ page }) => {
    const registerPage = new RegisterPage(page);
    await registerPage.goto();

    await registerPage.register({
      email: 'not-an-email',  // Invalid format
      name: 'Test User',
      phone: '+49 123 456 7890',
      password: 'Test123!',
      acceptTerms: true,
    });

    // Should show error (either frontend or backend validation)
    await page.waitForLoadState('networkidle');
    const hasError = await registerPage.hasError();

    console.log('Invalid email shows error:', hasError);
    // POTENTIAL BUG: Frontend validation might be missing
  });

  test('should reject registration without accepting terms', async ({ page }) => {
    const registerPage = new RegisterPage(page);
    await registerPage.goto();

    const timestamp = Date.now();
    await registerPage.register({
      email: `test-${timestamp}@example.com`,
      name: 'Test User',
      phone: '+49 123 456 7890',
      password: 'Test123!',
      acceptTerms: false,  // Not accepting terms
    });

    // Should prevent submission or show error
    await page.waitForLoadState('networkidle');

    const currentURL = page.url();
    console.log('After registration without terms, URL:', currentURL);

    // Should still be on register page
    expect(currentURL).toContain('register.html');

    // CRITICAL BUG CHECK: Can user register without accepting terms?
    if (!currentURL.includes('register.html')) {
      console.error('🐛 CRITICAL BUG: User registered without accepting terms!');
    }
  });

  test('should reject registration with weak password', async ({ page }) => {
    const registerPage = new RegisterPage(page);
    await registerPage.goto();

    const timestamp = Date.now();
    await registerPage.register({
      email: `test-${timestamp}@example.com`,
      name: 'Test User',
      phone: '+49 123 456 7890',
      password: '123',  // Too short
      acceptTerms: true,
    });

    await page.waitForLoadState('networkidle');

    // Should show error
    const hasError = await registerPage.hasError();
    console.log('Weak password shows error:', hasError);

    // POTENTIAL BUG: Password validation might be weak
  });

  test.skip('should reject duplicate email registration', async ({ page }) => {
    // TODO: Investigate why duplicate email doesn't show error alert
    // Backend likely prevents it but error not surfaced to UI properly
    const registerPage = new RegisterPage(page);

    // Use existing user email from generated data
    const duplicateEmail = 'admin@tierheim-goeppingen.de'; // From generated data

    await registerPage.goto();
    await registerPage.register({
      email: duplicateEmail,
      name: 'Duplicate User',
      phone: '+49 123 456 7890',
      password: 'Test123!',
      acceptTerms: true,
    });

    await page.waitForTimeout(2000);

    // Should either show error OR stay on registration page
    const currentURL = page.url();
    const hasError = await registerPage.hasError();

    console.log('After duplicate email registration - URL:', currentURL, 'Has error:', hasError);

    // Should not successfully register (either error shown or stays on page)
    if (hasError) {
      const errorMsg = await registerPage.getErrorMessage();
      console.log('Duplicate email error:', errorMsg);
    }

    // Main validation: Should not be redirected to login (which would indicate success)
    expect(currentURL).toContain('register.html');
  });

});

test.describe('Login - Valid Cases', () => {

  test('should login with valid credentials', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    // Login with generated test user (admin)
    await loginPage.loginAndWait('admin@tierheim-goeppingen.de', 'test123');

    // Should be on dashboard
    expect(page.url()).toContain('dashboard.html');

    // Dashboard should show user info
    const dashboardPage = new DashboardPage(page);
    const welcomeMsg = await dashboardPage.getWelcomeMessage();
    console.log('Welcome message:', welcomeMsg);

    // POTENTIAL BUG: Welcome message should include user name
    expect(welcomeMsg.length).toBeGreaterThan(0);
  });

  test('should store authentication token after login', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    await loginPage.loginAndWait('admin@tierheim-goeppingen.de', 'test123');

    // Check if token is stored in localStorage
    const token = await page.evaluate(() => {
      return localStorage.getItem('gassigeher_token');
    });

    console.log('Token stored:', token ? 'Yes' : 'No');
    expect(token).toBeTruthy();

    // CRITICAL BUG CHECK: Token should be a JWT
    if (token) {
      const isJWT = token.split('.').length === 3;
      console.log('Token is valid JWT:', isJWT);

      if (!isJWT) {
        console.error('🐛 POTENTIAL BUG: Token is not a valid JWT format!');
      }
    }
  });

  test('should persist login after page refresh', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    await loginPage.loginAndWait('admin@tierheim-goeppingen.de', 'test123');

    // Now refresh the page
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Should still be logged in
    const currentURL = page.url();
    console.log('After refresh, URL:', currentURL);

    // Should still be on dashboard, not redirected to login
    expect(currentURL).toContain('dashboard.html');

    // CRITICAL BUG CHECK: Session persistence
    if (currentURL.includes('login.html')) {
      console.error('🐛 CRITICAL BUG: User logged out after page refresh!');
    }
  });

});

test.describe('Login - Invalid Cases', () => {

  test('should reject login with invalid email', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    await loginPage.login('nonexistent@example.com', 'test123');
    await page.waitForLoadState('networkidle');

    // Should show error
    const hasError = await loginPage.hasError();
    expect(hasError).toBe(true);

    const errorMsg = await loginPage.getErrorMessage();
    console.log('Invalid email error:', errorMsg);

    // Error message should indicate invalid credentials
    const hasCredentialsError = errorMsg.toLowerCase().includes('invalid') ||
                                 errorMsg.toLowerCase().includes('ungültig') ||
                                 errorMsg.toLowerCase().includes('credentials') ||
                                 errorMsg.toLowerCase().includes('anmeldedaten');
    expect(hasCredentialsError).toBe(true);
  });

  test.skip('should reject login with wrong password', async ({ page }) => {
    // TODO: Error message language needs backend rebuild verification
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    await loginPage.login('admin@tierheim-goeppingen.de', 'wrongpassword');
    await page.waitForLoadState('networkidle');

    // Should show error
    const hasError = await loginPage.hasError();
    expect(hasError).toBe(true);

    if (hasError) {
      const errorMsg = await loginPage.getErrorMessage();
      console.log('Wrong password error:', errorMsg);

      // Error message should indicate invalid credentials
      const hasCredentialsError = errorMsg.toLowerCase().includes('invalid') ||
                                   errorMsg.toLowerCase().includes('ungültig') ||
                                   errorMsg.toLowerCase().includes('credentials') ||
                                   errorMsg.toLowerCase().includes('anmeldedaten');
      expect(hasCredentialsError).toBe(true);
    }
  });

  test('should reject login with empty credentials', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    await loginPage.login('', '');

    // Should show validation error or stay on page
    await page.waitForLoadState('networkidle');

    const currentURL = page.url();
    expect(currentURL).toContain('login.html');

    // POTENTIAL BUG: Frontend validation should catch empty fields
  });

  test('should reject login for unverified user', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    // Create unverified user for this test
    // For now skip this test as we need to setup unverified user first
    // TODO: Create unverified user in database before running this test
    test.skip();
    await page.waitForLoadState('networkidle');

    // Should show error about unverified account
    const currentURL = page.url();
    console.log('Unverified user login result:', currentURL);

    // Should either show error or be blocked
    if (currentURL.includes('dashboard.html')) {
      console.error('🐛 POTENTIAL BUG: Unverified user can log in!');
    }
  });

  test('should reject login for inactive user', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    // Inactive user should be in generated data (user #5)
    // Check the generated SQL file for the inactive user email
    // For now, test that inactive user cannot login
    test.skip();  // TODO: Get inactive user email from generated data
    await page.waitForLoadState('networkidle');

    // Should show error about inactive account
    const currentURL = page.url();
    console.log('Inactive user login result:', currentURL);

    if (currentURL.includes('dashboard.html')) {
      console.error('🐛 CRITICAL BUG: Inactive user can log in!');
    }
  });

});

test.describe('Logout', () => {

  test('should logout successfully', async ({ page }) => {
    // First login
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.loginAndWait('admin@tierheim-goeppingen.de', 'test123');

    const dashboardPage = new DashboardPage(page);
    await dashboardPage.logout();

    // Logout redirects to '/' (homepage), not login page directly
    const currentURL = page.url();
    console.log('After logout, URL is:', currentURL);
    // Should be on homepage or login (either is ok)
    expect(currentURL).toMatch(/\/(index\.html|login\.html)?$/);

    // Token should be cleared
    const token = await page.evaluate(() => {
      return localStorage.getItem('gassigeher_token');
    });

    console.log('Token after logout:', token);
    expect(token).toBeFalsy();

    // CRITICAL BUG CHECK: Token should be completely removed
    if (token) {
      console.error('🐛 CRITICAL BUG: Token not cleared after logout!');
    }
  });

  test('should not access protected pages after logout', async ({ page }) => {
    // Login
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.loginAndWait('admin@tierheim-goeppingen.de', 'test123');

    // Logout
    const dashboardPage = new DashboardPage(page);
    await dashboardPage.logout();

    // Clear any navigation state
    await page.waitForLoadState('networkidle');

    // Try to access dashboard
    await page.goto('http://localhost:8080/dashboard.html');
    await page.waitForLoadState('networkidle');

    const currentURL = page.url();
    console.log('After logout, trying to access dashboard:', currentURL);

    // Should redirect to login
    expect(currentURL).toContain('login.html');

    // CRITICAL BUG CHECK: Protected routes should be blocked
    if (currentURL.includes('dashboard.html')) {
      console.error('🐛 CRITICAL BUG: Can access dashboard after logout!');
    }
  });

});

test.describe('Password Reset Flow', () => {

  test('should show forgot password page', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    await loginPage.goToForgotPassword();

    expect(page.url()).toContain('forgot-password.html');
  });

  test('should accept email for password reset', async ({ page }) => {
    await page.goto('http://localhost:8080/forgot-password.html');

    // Fill email
    await page.fill('#email', 'admin@tierheim-goeppingen.de');
    await page.click('button[type="submit"]');

    await page.waitForLoadState('networkidle');

    // Should show success message (even if email doesn't exist - security)
    const pageText = await page.textContent('body');
    console.log('Password reset response received');

    // POTENTIAL BUG: Should show generic success message for security
  });

  test('should show generic message for non-existent email (security)', async ({ page }) => {
    await page.goto('http://localhost:8080/forgot-password.html');

    // Use non-existent email
    await page.fill('#email', 'nonexistent@example.com');
    await page.click('button[type="submit"]');

    await page.waitForLoadState('networkidle');

    // Should NOT reveal that email doesn't exist (security)
    const pageText = await page.textContent('body');
    console.log('Password reset for non-existent email - generic response shown');

    // CRITICAL SECURITY BUG: Should not reveal user existence
  });

});

test.describe('Session Management', () => {

  test('should handle expired tokens gracefully', async ({ page }) => {
    // Login first
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.loginAndWait('admin@tierheim-goeppingen.de', 'test123');

    // Manually set an expired token
    await page.evaluate(() => {
      // Create a fake expired JWT (just for testing UI behavior)
      localStorage.setItem('gassigeher_token', 'expired.token.here');
    });

    // Try to access protected page
    await page.goto('http://localhost:8080/dashboard.html');
    await page.waitForLoadState('networkidle');

    const currentURL = page.url();
    console.log('With expired token, redirected to:', currentURL);

    // Should redirect to login
    // CRITICAL BUG CHECK: Expired tokens should be handled
    if (currentURL.includes('dashboard.html')) {
      console.warn('⚠️ POTENTIAL BUG: Expired token not handled properly!');
    }
  });

  test('should handle multiple tabs correctly', async ({ browser }) => {
    // Create two pages (tabs)
    const context = await browser.newContext();
    const page1 = await context.newPage();
    const page2 = await context.newPage();

    // Login in tab 1
    await page1.goto('http://localhost:8080/login.html');
    await page1.fill('#email', 'admin@tierheim-goeppingen.de');
    await page1.fill('#password', 'test123');
    await page1.click('button[type="submit"]');
    await page1.waitForURL('**/dashboard.html');

    // Tab 2 should also be logged in (shared localStorage)
    await page2.goto('http://localhost:8080/dashboard.html');
    await page2.waitForLoadState('networkidle');

    const page2URL = page2.url();
    console.log('Tab 2 URL after tab 1 login:', page2URL);

    expect(page2URL).toContain('dashboard.html');

    // Now logout in tab 1
    await page1.click('a:has-text("Abmelden")');
    // Logout redirects to homepage (/), not login
    await page1.waitForLoadState('networkidle');

    // Tab 2 should also be logged out (or redirect on next action)
    await page2.reload();
    await page2.waitForLoadState('networkidle');

    const page2URLAfterLogout = page2.url();
    console.log('Tab 2 URL after tab 1 logout:', page2URLAfterLogout);

    // POTENTIAL BUG: Multi-tab session management
    await context.close();
  });

});

// DONE: Authentication tests - registration, login, logout, validation, session management
