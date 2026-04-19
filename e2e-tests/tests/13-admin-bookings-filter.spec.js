const { test, expect } = require('@playwright/test');
const LoginPage = require('../pages/LoginPage');
const { getConfigFromTestInfo } = require('../fixtures/test-config');

/**
 * ADMIN BOOKINGS - DOG FILTER
 * Locks in the /admin-bookings.html "filter by dog" dropdown:
 *  - Dropdown exists and is populated from api.getDogs()
 *  - Selecting a dog and clicking Anwenden sends ?dog_id=<id> to /api/v1/bookings
 *  - Zurücksetzen clears the selection and re-fires a request without dog_id
 *
 * Dual-Mode: runs against the demo tenant in both Simple-Mode and SaaS-Mode projects.
 */

test.describe('Admin Bookings - Dog Filter', () => {

  test('dropdown is populated and narrows bookings request by dog_id', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const admin = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(admin.email, admin.password);

    await page.goto('/admin-bookings.html');
    await page.waitForLoadState('networkidle');

    const dogSelect = page.locator('#filter-dog-id');
    await expect(dogSelect).toBeVisible();

    // Wait until the dropdown has been populated beyond the "Alle Hunde" placeholder.
    await page.waitForFunction(() => {
      const sel = document.getElementById('filter-dog-id');
      return sel && sel.options.length > 1;
    }, { timeout: 10000 });

    const optionCount = await dogSelect.locator('option').count();
    expect(optionCount).toBeGreaterThan(1);

    // Pick the first real dog option.
    const firstDogValue = await dogSelect.locator('option').nth(1).getAttribute('value');
    expect(firstDogValue).toBeTruthy();

    // Applying the filter should trigger a bookings request with dog_id in the URL.
    const filteredReqPromise = page.waitForRequest(
      (req) => req.url().includes('/api/v1/bookings')
            && req.url().includes(`dog_id=${firstDogValue}`),
      { timeout: 10000 }
    );
    await dogSelect.selectOption(firstDogValue);
    await page.click('[data-action="apply-filters"]');
    await filteredReqPromise;

    // Resetting should clear the select and re-fire a request without dog_id.
    const resetReqPromise = page.waitForRequest(
      (req) => req.url().includes('/api/v1/bookings')
            && !req.url().includes('dog_id='),
      { timeout: 10000 }
    );
    await page.click('[data-action="reset-filters"]');
    await resetReqPromise;
    await expect(dogSelect).toHaveValue('');
  });

});
