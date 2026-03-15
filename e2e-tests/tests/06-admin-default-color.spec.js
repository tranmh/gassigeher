const { test, expect } = require('@playwright/test');
const LoginPage = require('../pages/LoginPage');
const { getConfigFromTestInfo } = require('../fixtures/test-config');

/**
 * ADMIN DEFAULT COLOR TESTS
 * Test that the default color for new users can be changed and persists
 * across page navigations on admin-colors.html
 *
 * Dual-Mode: Tests run against both Simple-Mode and SaaS-Mode
 */

test.describe('Admin Default Color Setting', () => {

  test('should persist default color selection after navigating away and back', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    // Login as admin
    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    // Navigate to admin-colors page
    await page.goto('/admin-colors.html');
    await page.waitForLoadState('networkidle');

    // Wait for the default color dropdown to be populated (not "Wird geladen...")
    const select = page.locator('#default-color-select');
    await expect(select).toBeVisible();
    await page.waitForFunction(() => {
      const sel = document.getElementById('default-color-select');
      return sel && sel.options.length > 0 && sel.options[0].value !== '';
    }, { timeout: 10000 });

    // Read current selection
    const initialValue = await select.inputValue();
    console.log(`[Default Color] Initial selected value: ${initialValue}`);

    // Get all available options
    const options = await select.locator('option').all();
    expect(options.length).toBeGreaterThanOrEqual(2);

    // Pick a different color than the current one
    const allValues = await select.locator('option').evaluateAll(opts => opts.map(o => o.value));
    const newValue = allValues.find(v => v !== initialValue);
    expect(newValue).toBeTruthy();
    console.log(`[Default Color] Changing from ${initialValue} to ${newValue}`);

    // Select the new color
    await select.selectOption(newValue);

    // Click save
    await page.locator('[data-action="save-default-color"]').click();

    // Wait for success alert
    await expect(page.locator('.alert-success')).toBeVisible({ timeout: 5000 });
    console.log('[Default Color] Save confirmed with success alert');

    // Navigate away to another admin page
    await page.goto('/admin-dashboard.html');
    await page.waitForLoadState('networkidle');
    console.log('[Default Color] Navigated away to admin-dashboard');

    // Navigate back to admin-colors
    await page.goto('/admin-colors.html');
    await page.waitForLoadState('networkidle');

    // Wait for dropdown to be populated again
    await page.waitForFunction(() => {
      const sel = document.getElementById('default-color-select');
      return sel && sel.options.length > 0 && sel.options[0].value !== '';
    }, { timeout: 10000 });

    // Verify the selection persisted
    const persistedValue = await select.inputValue();
    console.log(`[Default Color] After revisit, selected value: ${persistedValue}`);
    expect(persistedValue).toBe(newValue);

    // Restore original value
    await select.selectOption(initialValue);
    await page.locator('[data-action="save-default-color"]').click();
    await expect(page.locator('.alert-success')).toBeVisible({ timeout: 5000 });
    console.log(`[Default Color] Restored original value: ${initialValue}`);
  });

  test('should show correct color in dropdown matching saved setting', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    // Login as admin
    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    // Navigate to admin-colors page
    await page.goto('/admin-colors.html');
    await page.waitForLoadState('networkidle');

    // Wait for dropdown to be populated
    const select = page.locator('#default-color-select');
    await page.waitForFunction(() => {
      const sel = document.getElementById('default-color-select');
      return sel && sel.options.length > 0 && sel.options[0].value !== '';
    }, { timeout: 10000 });

    // Intercept the settings API to verify the value matches
    const selectedValue = await select.inputValue();
    const settingsResponse = await page.evaluate(async () => {
      const token = localStorage.getItem('gassigeher_token');
      const resp = await fetch('/api/settings', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      return resp.json();
    });

    const setting = settingsResponse.find(s => s.key === 'default_color_for_new_users');
    console.log(`[Default Color] API value: ${setting?.value}, Dropdown value: ${selectedValue}`);

    if (setting && setting.value) {
      expect(selectedValue).toBe(setting.value);
    }
  });
});
