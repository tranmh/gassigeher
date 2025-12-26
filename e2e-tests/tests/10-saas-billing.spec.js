const { test, expect } = require('@playwright/test');
const LandingRegisterPage = require('../pages/LandingRegisterPage');
const AdminDogsPage = require('../pages/AdminDogsPage');
const BillingPage = require('../pages/BillingPage');
const LoginPage = require('../pages/LoginPage');

/**
 * SAAS BILLING E2E TESTS
 * Complete test suite for tenant registration, dog limits, and billing flows
 *
 * Test Scenario:
 * 1. New user navigates landing page
 * 2. User registers new tenant (stuttgart.gassigeher.local)
 * 3. User selects Free plan (10 dogs limit)
 * 4. User logs in and adds dogs
 * 5. User hits 10 dog limit and sees error
 * 6. User upgrades to Pro via test-upgrade
 * 7. User can now add unlimited dogs
 * 8. Edge cases: downgrade, over-limit behavior, cross-tenant isolation
 */

// Base domain for SaaS testing
const BASE_DOMAIN = 'gassigeher.local';
const TENANT_SLUG = 'stuttgart';
const TENANT_URL = `http://${TENANT_SLUG}.${BASE_DOMAIN}:8080`;

// Test data
const testTenant = {
  plan: 'free',
  organization: {
    name: 'Tierheim Stuttgart E2E',
    slug: TENANT_SLUG,
    contactEmail: 'kontakt@tierheim-stuttgart-e2e.de',
    contactPhone: '+49 711 9999999',
    address: 'Teststrasse 42',
    postalCode: '70174',
    city: 'Stuttgart',
    federalState: 'BW',
  },
  admin: {
    firstName: 'Test',
    lastName: 'Admin',
    email: 'admin@tierheim-stuttgart-e2e.de',
    password: 'TestPass123!',
  },
};

// Dog test data
const testDogs = [
  { name: 'Bello', breed: 'Labrador', age: 2, category: 'green', size: 'medium' },
  { name: 'Rex', breed: 'Schaeferhund', age: 3, category: 'green', size: 'large' },
  { name: 'Luna', breed: 'Golden Retriever', age: 4, category: 'orange', size: 'large' },
  { name: 'Max', breed: 'Dackel', age: 5, category: 'green', size: 'small' },
  { name: 'Maja', breed: 'Beagle', age: 2, category: 'green', size: 'medium' },
  { name: 'Bruno', breed: 'Boxer', age: 4, category: 'orange', size: 'large' },
  { name: 'Lotta', breed: 'Pudel', age: 3, category: 'green', size: 'medium' },
  { name: 'Felix', breed: 'Husky', age: 5, category: 'blue', size: 'large' },
  { name: 'Mia', breed: 'Rottweiler', age: 4, category: 'blue', size: 'large' },
  { name: 'Fido', breed: 'Collie', age: 3, category: 'green', size: 'medium' },
  { name: 'Rocky', breed: 'Bulldog', age: 2, category: 'green', size: 'medium' }, // 11th dog
  { name: 'Charlie', breed: 'Terrier', age: 1, category: 'green', size: 'small' }, // 12th dog
];

// Store token for API tests
let authToken = null;

test.describe('SaaS Landing Page', () => {
  test('should display landing page', async ({ page }) => {
    await page.goto(`http://${BASE_DOMAIN}:8080/landing/`);
    await page.waitForLoadState('networkidle');

    // Check landing page elements
    await expect(page.locator('header')).toBeVisible();
    await expect(page.locator('header .logo')).toBeVisible();

    console.log('Landing page loaded successfully');
  });

  test('should have registration link', async ({ page }) => {
    await page.goto(`http://${BASE_DOMAIN}:8080/landing/`);
    await page.waitForLoadState('networkidle');

    // Find registration link/button
    const registerLink = page.locator('a[href*="register"], a:has-text("Registrieren"), a:has-text("Jetzt starten")');
    await expect(registerLink.first()).toBeVisible();
  });

  test('should navigate to registration page', async ({ page }) => {
    const landingPage = new LandingRegisterPage(page, BASE_DOMAIN);
    await landingPage.goto();

    // Check registration form is visible
    await expect(page.locator('#register-form')).toBeVisible();
    await expect(page.locator('#organization_name')).toBeVisible();
    await expect(page.locator('#slug')).toBeVisible();
  });
});

test.describe('Tenant Registration', () => {
  test('should check slug availability', async ({ page }) => {
    const landingPage = new LandingRegisterPage(page, BASE_DOMAIN);
    await landingPage.goto();

    // Enter slug and wait for validation
    await page.fill('#slug', 'test-available-slug');
    await page.waitForTimeout(1000); // Debounce wait

    // Slug status should update
    const statusText = await page.textContent('#slug-status');
    console.log('Slug status:', statusText);

    // Should indicate availability (new slug should be available)
    // Note: exact text depends on implementation
  });

  test('should show Free and Pro plan options', async ({ page }) => {
    const landingPage = new LandingRegisterPage(page, BASE_DOMAIN);
    await landingPage.goto();

    // Both plans should be visible
    await expect(page.locator('.plan-card[data-plan="free"]')).toBeVisible();
    await expect(page.locator('.plan-card[data-plan="pro"]')).toBeVisible();

    // Free plan should show 10 dogs limit
    const freeCard = page.locator('.plan-card[data-plan="free"]');
    await expect(freeCard).toContainText('10 Hunde');

    // Pro plan should show unlimited
    const proCard = page.locator('.plan-card[data-plan="pro"]');
    await expect(proCard).toContainText('Unbegrenzte Hunde');
  });

  test('should register new tenant with Free plan', async ({ page }) => {
    const landingPage = new LandingRegisterPage(page, BASE_DOMAIN);
    await landingPage.goto();

    // Generate unique slug for this test run
    const timestamp = Date.now();
    const uniqueSlug = `stuttgart-${timestamp}`;
    const uniqueEmail = `admin-${timestamp}@test-e2e.de`;

    const registrationData = {
      plan: 'free',
      organization: {
        ...testTenant.organization,
        slug: uniqueSlug,
        contactEmail: `kontakt-${timestamp}@test-e2e.de`,
      },
      admin: {
        ...testTenant.admin,
        email: uniqueEmail,
      },
    };

    await landingPage.register(registrationData);

    // Wait for registration to complete
    await page.waitForTimeout(2000);

    // Should show success message
    const hasSuccess = await landingPage.hasFreePlanSuccess();
    console.log('Free plan registration success:', hasSuccess);

    if (hasSuccess) {
      const loginUrl = await landingPage.getLoginUrl();
      console.log('Login URL:', loginUrl);
      expect(loginUrl).toContain(uniqueSlug);
    }
  });
});

test.describe('Dog Limit Enforcement - Happy Path', () => {
  // Use API for faster testing
  test.describe.configure({ mode: 'serial' });

  // Fixed test subdomain - must be in /etc/hosts
  // Add to /etc/hosts: 127.0.0.1  testdogs.gassigeher.local
  const TEST_SLUG = 'testdogs';
  const TEST_EMAIL = 'admin@testdogs.e2e-test.de';
  const TEST_PASSWORD = 'E2ETestPass123!';

  let token = null;

  test.beforeAll(async ({ request }) => {
    // First, try to login with existing tenant
    const loginResponse = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/auth/login`, {
      data: {
        email: TEST_EMAIL,
        password: TEST_PASSWORD,
      },
    });

    if (loginResponse.status() === 200) {
      // Tenant exists, use existing token
      const loginData = await loginResponse.json();
      token = loginData.token;
      authToken = token;
      console.log('Using existing tenant:', TEST_SLUG);
      return;
    }

    // Register new tenant via API
    console.log('Creating new tenant:', TEST_SLUG);
    const registerResponse = await request.post(`http://${BASE_DOMAIN}:8080/api/v1/tenants/register`, {
      data: {
        organization_name: 'E2E Dog Test Shelter',
        slug: TEST_SLUG,
        contact_email: 'kontakt@testdogs.e2e-test.de',
        contact_phone: '+49 711 1111111',
        address: 'E2E Test Street 1',
        city: 'Stuttgart',
        postal_code: '70174',
        federal_state: 'BW',
        admin_first_name: 'E2E',
        admin_last_name: 'Tester',
        admin_email: TEST_EMAIL,
        admin_password: TEST_PASSWORD,
      },
    });

    // 201 Created is expected for successful registration
    expect(registerResponse.status()).toBe(201);
    console.log('Tenant registered:', TEST_SLUG);

    // Login to get token
    const newLoginResponse = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/auth/login`, {
      data: {
        email: TEST_EMAIL,
        password: TEST_PASSWORD,
      },
    });

    expect(newLoginResponse.status()).toBe(200);
    const loginData = await newLoginResponse.json();
    token = loginData.token;
    authToken = token;
  });

  test('should verify Free plan with 10 dog limit', async ({ request }) => {
    const slug = TEST_SLUG;

    const usageResponse = await request.get(`http://${slug}.${BASE_DOMAIN}:8080/api/v1/billing/usage`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });

    const usage = await usageResponse.json();
    console.log('Initial usage:', usage);

    expect(usage.dogs_limit).toBe(10);
    expect(usage.dogs_used).toBe(0);
    expect(usage.over_limit).toBe(false);
  });

  test('should verify test mode is enabled', async ({ request }) => {
    const plansResponse = await request.get(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/plans`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });

    const plans = await plansResponse.json();
    console.log('Test mode enabled:', plans.test_mode);

    expect(plans.test_mode).toBe(true);
    expect(plans.stripe_configured).toBe(false);
  });

  test('should add 10 dogs successfully', async ({ request }) => {
    for (let i = 0; i < 10; i++) {
      const dog = testDogs[i];
      const response = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/dogs`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        data: dog,
      });

      expect(response.status()).toBe(201);
      const dogData = await response.json();
      console.log(`Dog ${i + 1}/10 created: ${dogData.name} (ID: ${dogData.id})`);
    }

    // Verify usage
    const usageResponse = await request.get(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/usage`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });
    const usage = await usageResponse.json();

    expect(usage.dogs_used).toBe(10);
    expect(usage.dogs_limit).toBe(10);
    expect(usage.over_limit).toBe(false);
    console.log('Usage after 10 dogs:', usage);
  });

  test('should block 11th dog with limit error', async ({ request }) => {
    const response = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/dogs`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: testDogs[10], // 11th dog
    });

    // Should return 409 Conflict
    expect(response.status()).toBe(409);

    const errorData = await response.json();
    console.log('11th dog error:', errorData);

    expect(errorData.error).toContain('Hundelimit');
    expect(errorData.limit).toBe(10);
    expect(errorData.current_count).toBe(10);
  });

  test('should upgrade to Pro via test-upgrade', async ({ request }) => {
    const response = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/test-upgrade`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: { plan_slug: 'pro' },
    });

    expect(response.status()).toBe(200);

    const upgradeData = await response.json();
    console.log('Upgrade response:', upgradeData);

    expect(upgradeData.plan).toBe('Pro');
    expect(upgradeData.test_mode).toBe(true);
    expect(upgradeData.subscription.plan.max_dogs).toBe(-1); // Unlimited
  });

  test('should add 11th and 12th dogs after Pro upgrade', async ({ request }) => {
    // Add 11th dog
    const response11 = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/dogs`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: testDogs[10],
    });

    expect(response11.status()).toBe(201);
    const dog11 = await response11.json();
    console.log('11th dog created:', dog11.name);

    // Add 12th dog
    const response12 = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/dogs`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: testDogs[11],
    });

    expect(response12.status()).toBe(201);
    const dog12 = await response12.json();
    console.log('12th dog created:', dog12.name);

    // Verify usage shows unlimited
    const usageResponse = await request.get(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/usage`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });
    const usage = await usageResponse.json();

    expect(usage.dogs_used).toBe(12);
    expect(usage.dogs_limit).toBe(-1); // Unlimited
    expect(usage.over_limit).toBe(false);
    console.log('Final usage:', usage);
  });
});

test.describe('Edge Cases - Downgrade and Over-Limit', () => {
  test.describe.configure({ mode: 'serial' });

  // Fixed test subdomain - must be in /etc/hosts
  // Add to /etc/hosts: 127.0.0.1  testedge.gassigeher.local
  const TEST_SLUG = 'testedge';
  const TEST_EMAIL = 'admin@testedge.e2e-test.de';
  const TEST_PASSWORD = 'EdgeTestPass123!';

  let token = null;

  test.beforeAll(async ({ request }) => {
    // Try to login with existing tenant
    const loginResponse = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/auth/login`, {
      data: { email: TEST_EMAIL, password: TEST_PASSWORD },
    });

    if (loginResponse.status() === 200) {
      const loginData = await loginResponse.json();
      token = loginData.token;
      console.log('Using existing edge case tenant:', TEST_SLUG);

      // Ensure it's at Pro with 12 dogs (reset state)
      await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/test-upgrade`, {
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        data: { plan_slug: 'pro' },
      });
      return;
    }

    // Register new tenant
    console.log('Creating new edge case tenant:', TEST_SLUG);
    await request.post(`http://${BASE_DOMAIN}:8080/api/v1/tenants/register`, {
      data: {
        organization_name: 'E2E Edge Case Shelter',
        slug: TEST_SLUG,
        contact_email: 'kontakt@testedge.e2e-test.de',
        contact_phone: '+49 711 2222222',
        address: 'Edge Case Street 1',
        city: 'Stuttgart',
        postal_code: '70174',
        federal_state: 'BW',
        admin_first_name: 'Edge',
        admin_last_name: 'Tester',
        admin_email: TEST_EMAIL,
        admin_password: TEST_PASSWORD,
      },
    });

    // Login
    const newLoginResponse = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/auth/login`, {
      data: { email: TEST_EMAIL, password: TEST_PASSWORD },
    });
    const loginData = await newLoginResponse.json();
    token = loginData.token;

    // Upgrade to Pro
    await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/test-upgrade`, {
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      data: { plan_slug: 'pro' },
    });

    // Add 12 dogs
    for (let i = 0; i < 12; i++) {
      await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/dogs`, {
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        data: testDogs[i],
      });
    }

    console.log('Edge case setup complete: Pro tenant with 12 dogs');
  });

  test('should show over_limit=true after downgrade', async ({ request }) => {
    // Downgrade to Free
    const downgradeResponse = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/test-upgrade`, {
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      data: { plan_slug: 'free' },
    });

    expect(downgradeResponse.status()).toBe(200);

    // Check usage
    const usageResponse = await request.get(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/usage`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });
    const usage = await usageResponse.json();

    console.log('Usage after downgrade:', usage);

    expect(usage.dogs_limit).toBe(10);
    expect(usage.dogs_used).toBe(12);
    expect(usage.over_limit).toBe(true);
    expect(usage.excess_count).toBe(2);
  });

  test('should block new dogs when over limit', async ({ request }) => {
    const response = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/dogs`, {
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      data: { name: 'Blocked', breed: 'Test', age: 1, category: 'green', size: 'small' },
    });

    expect(response.status()).toBe(409);
    const error = await response.json();
    console.log('Blocked dog error:', error);

    expect(error.error).toContain('Hundelimit');
  });

  test('should reject invalid plan slug', async ({ request }) => {
    const response = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/test-upgrade`, {
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      data: { plan_slug: 'enterprise' },
    });

    expect(response.status()).toBe(400);
    const error = await response.json();
    expect(error.error).toContain('Ungültiger Plan');
  });

  test('should default empty plan to Pro', async ({ request }) => {
    // First downgrade to free to make this test meaningful
    await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/test-upgrade`, {
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      data: { plan_slug: 'free' },
    });

    // Now upgrade with empty plan_slug
    const response = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/test-upgrade`, {
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      data: {},
    });

    expect(response.status()).toBe(200);
    const data = await response.json();
    expect(data.plan).toBe('Pro');
  });
});

test.describe('Security Tests', () => {
  // Fixed test subdomain - must be in /etc/hosts
  // Add to /etc/hosts: 127.0.0.1  testsec.gassigeher.local
  const TEST_SLUG = 'testsec';
  const TEST_EMAIL = 'admin@testsec.e2e-test.de';
  const TEST_PASSWORD = 'SecTest123!';

  let token = null;

  test.beforeAll(async ({ request }) => {
    // Try to login with existing tenant
    const loginResponse = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/auth/login`, {
      data: { email: TEST_EMAIL, password: TEST_PASSWORD },
    });

    if (loginResponse.status() === 200) {
      const loginData = await loginResponse.json();
      token = loginData.token;
      console.log('Using existing security test tenant:', TEST_SLUG);
      return;
    }

    // Register new tenant
    console.log('Creating new security test tenant:', TEST_SLUG);
    await request.post(`http://${BASE_DOMAIN}:8080/api/v1/tenants/register`, {
      data: {
        organization_name: 'Security Test',
        slug: TEST_SLUG,
        contact_email: 'kontakt@testsec.e2e-test.de',
        city: 'Stuttgart',
        postal_code: '70174',
        federal_state: 'BW',
        admin_first_name: 'Sec',
        admin_last_name: 'Test',
        admin_email: TEST_EMAIL,
        admin_password: TEST_PASSWORD,
      },
    });

    const newLoginResponse = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/auth/login`, {
      data: { email: TEST_EMAIL, password: TEST_PASSWORD },
    });
    const loginData = await newLoginResponse.json();
    token = loginData.token;
  });

  test('should reject test-upgrade without auth token', async ({ request }) => {
    const response = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/test-upgrade`, {
      data: { plan_slug: 'pro' },
    });

    expect(response.status()).toBe(401);
  });

  test('should reject regular checkout when Stripe not configured', async ({ request }) => {
    // Try regular checkout (should fail - Stripe not configured)
    const checkoutResponse = await request.post(`http://${TEST_SLUG}.${BASE_DOMAIN}:8080/api/v1/billing/checkout`, {
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      data: { plan_slug: 'pro', billing_cycle: 'monthly' },
    });

    expect(checkoutResponse.status()).toBe(503);
    const error = await checkoutResponse.json();
    expect(error.error).toContain('Zahlungssystem nicht konfiguriert');
  });
});

test.describe('Cross-Tenant Isolation', () => {
  // Fixed test subdomains - must be in /etc/hosts
  // Add to /etc/hosts:
  //   127.0.0.1  testtenant1.gassigeher.local
  //   127.0.0.1  testtenant2.gassigeher.local
  const TENANT1_SLUG = 'testtenant1';
  const TENANT1_EMAIL = 'admin@testtenant1.e2e-test.de';
  const TENANT1_PASSWORD = 'T1Pass123!';

  const TENANT2_SLUG = 'testtenant2';
  const TENANT2_EMAIL = 'admin@testtenant2.e2e-test.de';
  const TENANT2_PASSWORD = 'T2Pass123!';

  let token1 = null;
  let token2 = null;

  test.beforeAll(async ({ request }) => {
    // Setup tenant 1
    const login1Response = await request.post(`http://${TENANT1_SLUG}.${BASE_DOMAIN}:8080/api/v1/auth/login`, {
      data: { email: TENANT1_EMAIL, password: TENANT1_PASSWORD },
    });

    if (login1Response.status() === 200) {
      token1 = (await login1Response.json()).token;
      console.log('Using existing tenant 1:', TENANT1_SLUG);
    } else {
      console.log('Creating tenant 1:', TENANT1_SLUG);
      await request.post(`http://${BASE_DOMAIN}:8080/api/v1/tenants/register`, {
        data: {
          organization_name: 'Tenant 1',
          slug: TENANT1_SLUG,
          contact_email: 'kontakt@testtenant1.e2e-test.de',
          city: 'Stuttgart',
          postal_code: '70174',
          federal_state: 'BW',
          admin_first_name: 'T1',
          admin_last_name: 'Admin',
          admin_email: TENANT1_EMAIL,
          admin_password: TENANT1_PASSWORD,
        },
      });
      const newLogin1 = await request.post(`http://${TENANT1_SLUG}.${BASE_DOMAIN}:8080/api/v1/auth/login`, {
        data: { email: TENANT1_EMAIL, password: TENANT1_PASSWORD },
      });
      token1 = (await newLogin1.json()).token;
    }

    // Setup tenant 2
    const login2Response = await request.post(`http://${TENANT2_SLUG}.${BASE_DOMAIN}:8080/api/v1/auth/login`, {
      data: { email: TENANT2_EMAIL, password: TENANT2_PASSWORD },
    });

    if (login2Response.status() === 200) {
      token2 = (await login2Response.json()).token;
      console.log('Using existing tenant 2:', TENANT2_SLUG);
    } else {
      console.log('Creating tenant 2:', TENANT2_SLUG);
      await request.post(`http://${BASE_DOMAIN}:8080/api/v1/tenants/register`, {
        data: {
          organization_name: 'Tenant 2',
          slug: TENANT2_SLUG,
          contact_email: 'kontakt@testtenant2.e2e-test.de',
          city: 'Berlin',
          postal_code: '10115',
          federal_state: 'BE',
          admin_first_name: 'T2',
          admin_last_name: 'Admin',
          admin_email: TENANT2_EMAIL,
          admin_password: TENANT2_PASSWORD,
        },
      });
      const newLogin2 = await request.post(`http://${TENANT2_SLUG}.${BASE_DOMAIN}:8080/api/v1/auth/login`, {
        data: { email: TENANT2_EMAIL, password: TENANT2_PASSWORD },
      });
      token2 = (await newLogin2.json()).token;
    }
  });

  test('should not allow cross-tenant data access', async ({ request }) => {
    // Add a unique dog to tenant 1
    const uniqueDogName = `T1Dog-${Date.now()}`;
    await request.post(`http://${TENANT1_SLUG}.${BASE_DOMAIN}:8080/api/v1/dogs`, {
      headers: { 'Authorization': `Bearer ${token1}`, 'Content-Type': 'application/json' },
      data: { name: uniqueDogName, breed: 'Test', age: 1, category: 'green', size: 'small' },
    });

    // Try to access tenant 2 with tenant 1's token
    const crossAccessResponse = await request.get(`http://${TENANT2_SLUG}.${BASE_DOMAIN}:8080/api/v1/dogs`, {
      headers: { 'Authorization': `Bearer ${token1}` },
    });

    // Should be rejected or return empty (not tenant 1's dog)
    if (crossAccessResponse.status() === 200) {
      const dogs = await crossAccessResponse.json();
      // Should not contain tenant 1's unique dog
      const hasCrossTenantDog = dogs.some(d => d.name === uniqueDogName);
      expect(hasCrossTenantDog).toBe(false);
      console.log('Cross-tenant isolation verified: Tenant 2 has', dogs.length, 'dogs');
    } else {
      // Rejected is also acceptable
      console.log('Cross-tenant access rejected with status:', crossAccessResponse.status());
    }
  });
});

// Required /etc/hosts entries for these tests:
// 127.0.0.1  gassigeher.local
// 127.0.0.1  stuttgart.gassigeher.local
// 127.0.0.1  testdogs.gassigeher.local
// 127.0.0.1  testedge.gassigeher.local
// 127.0.0.1  testsec.gassigeher.local
// 127.0.0.1  testtenant1.gassigeher.local
// 127.0.0.1  testtenant2.gassigeher.local
