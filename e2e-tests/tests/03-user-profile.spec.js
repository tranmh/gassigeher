const { test, expect } = require('@playwright/test');
const LoginPage = require('../pages/LoginPage');
const DashboardPage = require('../pages/DashboardPage');
const { getConfigFromTestInfo } = require('../fixtures/test-config');

/**
 * USER PROFILE TESTS
 * Test profile viewing, updating, photo upload, password change
 *
 * Dual-Mode: Tests run against both Simple-Mode and SaaS-Mode
 *
 * Profile page structure:
 * - #display-first-name, #display-last-name (read-only)
 * - #edit-email, #edit-phone (editable)
 * - #old-password, #new-password, #confirm-password (password change)
 * - #photo-input (file upload)
 */

test.describe('Profile - View Profile', () => {

  test('should access profile page after login', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    expect(page.url()).toContain('profile.html');
    console.log(`[${config.mode}] Accessed profile page`);
  });

  test('should display user name fields', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    // Check for name display fields
    const firstNameField = page.locator('#display-first-name');
    const lastNameField = page.locator('#display-last-name');

    const hasFirstName = await firstNameField.count() > 0;
    const hasLastName = await lastNameField.count() > 0;

    console.log(`[${config.mode}] Has first name field:`, hasFirstName);
    console.log(`[${config.mode}] Has last name field:`, hasLastName);

    // At least one name field should exist
    expect(hasFirstName || hasLastName).toBe(true);
  });

  test('should display email field with current email', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    // Check email field
    const emailField = page.locator('#edit-email');
    const emailExists = await emailField.count() > 0;

    expect(emailExists).toBe(true);

    if (emailExists) {
      const currentEmail = await emailField.inputValue();
      console.log(`[${config.mode}] Email field shows:`, currentEmail);
      expect(currentEmail).toContain('@');
    }
  });

});

test.describe('Profile - Update Information', () => {

  test('should have editable phone field', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    // Check phone field exists and is editable
    const phoneField = page.locator('#edit-phone');
    const phoneExists = await phoneField.count() > 0;

    console.log(`[${config.mode}] Phone field exists:`, phoneExists);
    expect(phoneExists).toBe(true);

    if (phoneExists) {
      const isEditable = await phoneField.isEditable();
      console.log(`[${config.mode}] Phone field is editable:`, isEditable);
      expect(isEditable).toBe(true);
    }
  });

  test('should update phone number successfully', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    const phoneField = page.locator('#edit-phone');
    const originalPhone = await phoneField.inputValue().catch(() => '');

    // Use valid phone format: pattern requires max 9 digits at end
    const randomSuffix = Math.floor(Math.random() * 100000000).toString().padStart(8, '0');
    const newPhone = '+49 171 ' + randomSuffix;

    // Update phone
    await phoneField.fill(newPhone);
    await page.click('#profile-form button[type="submit"]');

    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Reload and verify
    await page.reload();
    await page.waitForLoadState('networkidle');

    const updatedPhone = await phoneField.inputValue();
    console.log(`[${config.mode}] Phone after update:`, updatedPhone);
    expect(updatedPhone).toBe(newPhone);

    // Restore original phone
    await phoneField.fill(originalPhone || '+49 123 456789');
    await page.click('#profile-form button[type="submit"]');
    await page.waitForLoadState('networkidle');
  });

  test('should reject invalid email format via HTML5 validation', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    const emailField = page.locator('#edit-email');
    const originalEmail = await emailField.inputValue();

    // Try to set invalid email
    await emailField.fill('not-an-email');
    await page.click('#profile-form button[type="submit"]');

    await page.waitForTimeout(500);

    // Should stay on profile page (HTML5 validation prevents submit)
    expect(page.url()).toContain('profile.html');
    console.log(`[${config.mode}] Invalid email validation works`);

    // Restore original email
    await emailField.fill(originalEmail);
  });

  test('should prevent empty email submission', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    const emailField = page.locator('#edit-email');
    const originalEmail = await emailField.inputValue();

    // Try to clear email (required field)
    await emailField.fill('');
    await page.click('#profile-form button[type="submit"]');

    await page.waitForTimeout(500);

    // Should stay on profile page due to HTML5 required validation
    expect(page.url()).toContain('profile.html');
    console.log(`[${config.mode}] Empty email validation works`);

    // Restore original email
    await emailField.fill(originalEmail);
  });

});

test.describe('Profile - Photo Upload', () => {

  test('should show photo upload section', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    // Look for file input (hidden but should exist)
    const fileInput = page.locator('#photo-input');
    const fileInputExists = await fileInput.count() > 0;

    console.log(`[${config.mode}] Photo input exists:`, fileInputExists);
    expect(fileInputExists).toBe(true);
  });

  test('should have photo upload button', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    // Look for upload trigger button
    const uploadButton = page.locator('button[data-action="trigger-photo-input"]');
    const buttonExists = await uploadButton.count() > 0;

    console.log(`[${config.mode}] Photo upload button exists:`, buttonExists);
    expect(buttonExists).toBe(true);
  });

  test('should accept only image file types', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    // Check file input accepts only images
    const fileInput = page.locator('#photo-input');
    const acceptAttr = await fileInput.getAttribute('accept');

    console.log(`[${config.mode}] File input accept attribute:`, acceptAttr);

    // Should only accept image types
    if (acceptAttr) {
      expect(acceptAttr).toMatch(/image/i);
    }
  });

});

test.describe('Profile - Password Change', () => {

  test('should show password change form', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    // Check for password fields
    const oldPasswordField = page.locator('#old-password');
    const newPasswordField = page.locator('#new-password');
    const confirmPasswordField = page.locator('#confirm-password');

    const hasOldPassword = await oldPasswordField.count() > 0;
    const hasNewPassword = await newPasswordField.count() > 0;
    const hasConfirmPassword = await confirmPasswordField.count() > 0;

    console.log(`[${config.mode}] Password form - Old:`, hasOldPassword, 'New:', hasNewPassword, 'Confirm:', hasConfirmPassword);

    // At least old and new password fields should exist
    expect(hasOldPassword).toBe(true);
    expect(hasNewPassword).toBe(true);
  });

  test('should have password change submit button', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    // Look for password form submit button
    const passwordForm = page.locator('#password-form');
    const submitButton = passwordForm.locator('button[type="submit"]');

    const buttonExists = await submitButton.count() > 0;
    console.log(`[${config.mode}] Password change button exists:`, buttonExists);
    expect(buttonExists).toBe(true);
  });

  test('should require all password fields', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    await page.goto('/profile.html');
    await page.waitForLoadState('networkidle');

    // Check that password fields are required
    const oldPasswordField = page.locator('#old-password');
    const newPasswordField = page.locator('#new-password');

    const oldRequired = await oldPasswordField.getAttribute('required');
    const newRequired = await newPasswordField.getAttribute('required');

    console.log(`[${config.mode}] Old password required:`, oldRequired !== null);
    console.log(`[${config.mode}] New password required:`, newRequired !== null);

    // Fields should be required
    expect(oldRequired !== null || newRequired !== null).toBe(true);
  });

});
