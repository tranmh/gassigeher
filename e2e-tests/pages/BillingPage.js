const BasePage = require('./BasePage');

/**
 * Billing Page Object
 * For managing subscription and billing in tenant interface
 */
class BillingPage extends BasePage {
  constructor(page, subdomain = 'stuttgart', baseDomain = 'gassigeher.local') {
    super(page);
    this.subdomain = subdomain;
    this.baseDomain = baseDomain;
    this.baseURL = `http://${subdomain}.${baseDomain}:8080`;

    // Page Elements
    this.currentPlanName = '.current-plan-name, #current-plan, .plan-name';
    this.usageBar = '.usage-bar, #usage-bar';
    this.usageText = '.usage-text, #usage-text';
    this.dogsUsed = '#dogs-used, .dogs-used';
    this.dogsLimit = '#dogs-limit, .dogs-limit';

    // Upgrade/Downgrade
    this.upgradeButton = '#upgrade-btn, button:has-text("Upgrade"), a:has-text("Upgrade")';
    this.testUpgradeButton = '#test-upgrade-btn, button:has-text("Test Upgrade")';

    // Plan Cards
    this.freePlanCard = '.plan-card[data-plan="free"], .plan-free';
    this.proPlanCard = '.plan-card[data-plan="pro"], .plan-pro';

    // Status Messages
    this.overLimitWarning = '.over-limit-warning, .limit-warning';
    this.testModeIndicator = '.test-mode, #test-mode-indicator';

    // API Responses (for direct API testing)
    this.apiBase = `${this.baseURL}/api/v1`;
  }

  /**
   * Navigate to billing page
   */
  async goto() {
    await this.page.goto(`${this.baseURL}/billing.html`);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Update subdomain for this page
   */
  setSubdomain(subdomain) {
    this.subdomain = subdomain;
    this.baseURL = `http://${subdomain}.${this.baseDomain}:8080`;
    this.apiBase = `${this.baseURL}/api/v1`;
  }

  /**
   * Get current plan name
   */
  async getCurrentPlan() {
    return await this.page.textContent(this.currentPlanName).catch(() => '');
  }

  /**
   * Get usage information via API
   */
  async getUsageViaAPI(token) {
    const response = await this.page.request.get(`${this.apiBase}/billing/usage`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });
    return await response.json();
  }

  /**
   * Get plans via API
   */
  async getPlansViaAPI(token) {
    const response = await this.page.request.get(`${this.apiBase}/billing/plans`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });
    return await response.json();
  }

  /**
   * Test upgrade via API (only works in test mode)
   */
  async testUpgradeViaAPI(token, planSlug = 'pro') {
    const response = await this.page.request.post(`${this.apiBase}/billing/test-upgrade`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: { plan_slug: planSlug },
    });
    return {
      status: response.status(),
      data: await response.json().catch(() => ({})),
    };
  }

  /**
   * Test downgrade via API (only works in test mode)
   */
  async testDowngradeViaAPI(token) {
    return await this.testUpgradeViaAPI(token, 'free');
  }

  /**
   * Check if over limit warning is visible
   */
  async hasOverLimitWarning() {
    return await this.page.locator(this.overLimitWarning).isVisible().catch(() => false);
  }

  /**
   * Check if test mode is indicated
   */
  async isTestMode() {
    // Check via API
    const plansResponse = await this.page.request.get(`${this.apiBase}/billing/plans`);
    const data = await plansResponse.json();
    return data.test_mode === true;
  }

  /**
   * Click upgrade button
   */
  async clickUpgrade() {
    await this.page.click(this.upgradeButton);
  }

  /**
   * Get subscription info via API
   */
  async getSubscriptionViaAPI(token) {
    const response = await this.page.request.get(`${this.apiBase}/billing/subscription`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });
    return await response.json();
  }
}

module.exports = BillingPage;
