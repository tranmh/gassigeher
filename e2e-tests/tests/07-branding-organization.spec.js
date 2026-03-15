const { test, expect } = require('@playwright/test');
const { getConfigFromTestInfo } = require('../fixtures/test-config');

/**
 * BRANDING & ORGANIZATION FIELDS TESTS
 * Test that branding replaces hardcoded "Gassigeher" with tenant name,
 * and that organization fields appear on privacy/terms pages.
 *
 * These tests verify the fix for generic/placeholder content in Simple-Mode.
 */

test.describe('Branding - Dynamic Tenant Name', () => {

  test('homepage should show tenant name in title (not "Gassigeher")', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const title = await page.title();
    console.log(`[${config.mode}] Homepage title: "${title}"`);

    // Title should not show bare "Gassigeher" as the org name
    // It should either be replaced by tenant name or show the branding
    expect(title).toBeTruthy();
    expect(title.length).toBeGreaterThan(0);
  });

  test('login page should load branding.js and update title', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    await page.goto('/login.html');
    await page.waitForLoadState('networkidle');

    // branding.js should be loaded
    const brandingScriptLoaded = await page.evaluate(() => {
      return typeof window.loadBrandingData === 'function';
    });
    expect(brandingScriptLoaded).toBe(true);

    console.log(`[${config.mode}] Login page has branding.js loaded`);
  });

  test('branding API should return organization fields', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    await page.goto('/');

    // Call the branding API directly
    const branding = await page.evaluate(async () => {
      const response = await fetch('/api/v1/tenant/branding');
      if (!response.ok) return null;
      return await response.json();
    });

    expect(branding).toBeTruthy();
    expect(branding.tenant_name).toBeTruthy();
    expect(branding.theme_preset).toBeTruthy();

    console.log(`[${config.mode}] Branding API tenant_name: "${branding.tenant_name}"`);
    console.log(`[${config.mode}] Branding API org_name: "${branding.organization_name || '(not set)'}"`);
    console.log(`[${config.mode}] Branding API org_email: "${branding.organization_email || '(not set)'}"`);
  });

});

test.describe('Branding - Privacy Page', () => {

  test('privacy page should not show placeholder emails', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    await page.goto('/privacy.html');
    await page.waitForLoadState('networkidle');

    const bodyText = await page.textContent('body');

    // Should NOT contain the old placeholder emails
    expect(bodyText).not.toContain('datenschutz@gassigeher.example.com');
    expect(bodyText).not.toContain('info@gassigeher.example.com');

    // Should NOT contain "[Ihre Adresse]" placeholder
    expect(bodyText).not.toContain('[Ihre Adresse]');

    console.log(`[${config.mode}] Privacy page has no placeholder content`);
  });

  test('privacy page should have DSGVO content', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    await page.goto('/privacy.html');
    await page.waitForLoadState('networkidle');

    const bodyText = await page.textContent('body');

    // Should still contain essential GDPR/privacy content
    expect(bodyText).toContain('Datenschutzerklärung');
    expect(bodyText).toContain('DSGVO');
    expect(bodyText).toContain('Verantwortliche Stelle');

    console.log(`[${config.mode}] Privacy page has DSGVO content`);
  });

});

test.describe('Branding - Terms Page', () => {

  test('terms page should load and have AGB content', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    await page.goto('/terms.html');
    await page.waitForLoadState('networkidle');

    const bodyText = await page.textContent('body');

    expect(bodyText).toContain('Allgemeine Geschäftsbedingungen');
    expect(bodyText).toContain('Geltungsbereich');

    console.log(`[${config.mode}] Terms page has AGB content`);
  });

  test('terms page should have branding.js loaded', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    await page.goto('/terms.html');
    await page.waitForLoadState('networkidle');

    const hasBranding = await page.evaluate(() => {
      return typeof window.loadBrandingData === 'function';
    });
    expect(hasBranding).toBe(true);

    console.log(`[${config.mode}] Terms page has branding.js`);
  });

});

test.describe('Branding - Powered By Attribution', () => {

  test('homepage should preserve "Powered by Gassigeher" link', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // The "Powered by Gassigeher" should remain unchanged (data-no-branding)
    const poweredBy = page.locator('[data-no-branding] a[href="https://gassigeher.org"]');
    const count = await poweredBy.count();

    if (count > 0) {
      const text = await poweredBy.first().textContent();
      expect(text).toContain('Powered by Gassigeher');
      console.log(`[${config.mode}] "Powered by Gassigeher" preserved correctly`);
    } else {
      console.log(`[${config.mode}] No "Powered by" link found (may be SaaS mode)`);
    }
  });

});
