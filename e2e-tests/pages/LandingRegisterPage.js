const BasePage = require('./BasePage');

/**
 * Landing Registration Page Object
 * For SaaS tenant registration flow
 */
class LandingRegisterPage extends BasePage {
  constructor(page, baseDomain = 'gassigeher.local') {
    super(page);
    this.baseDomain = baseDomain;
    this.baseURL = `http://${baseDomain}:8080`;

    // Plan Selection
    this.freePlanCard = '.plan-card[data-plan="free"]';
    this.proPlanCard = '.plan-card[data-plan="pro"]';
    this.billingToggleContainer = '#billing-toggle-container';
    this.monthlyButton = '.billing-toggle button[data-cycle="monthly"]';
    this.yearlyButton = '.billing-toggle button[data-cycle="yearly"]';

    // Organization Info
    this.organizationNameInput = '#organization_name';
    this.slugInput = '#slug';
    this.slugStatus = '#slug-status';
    this.contactEmailInput = '#contact_email';
    this.contactPhoneInput = '#contact_phone';
    this.addressInput = '#address';
    this.postalCodeInput = '#postal_code';
    this.cityInput = '#city';
    this.federalStateSelect = '#federal_state';

    // Admin Account
    this.adminFirstNameInput = '#admin_first_name';
    this.adminLastNameInput = '#admin_last_name';
    this.adminEmailInput = '#admin_email';
    this.adminPasswordInput = '#admin_password';

    // Terms and Submit
    this.termsCheckbox = 'input[name="terms"]';
    this.submitButton = '#submit-btn';

    // Success Messages
    this.successMessageFree = '#success-message-free';
    this.successMessagePro = '#success-message-pro';
    this.loginLinkFree = '#login-link-free';
    this.checkoutButton = '#checkout-btn';
    this.skipPaymentLink = '#skip-payment-link';

    // Form
    this.registerForm = '#register-form';
  }

  /**
   * Navigate to landing registration page
   */
  async goto() {
    await this.page.goto(`${this.baseURL}/landing/register.html`);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Navigate to landing homepage
   */
  async gotoLanding() {
    await this.page.goto(`${this.baseURL}/landing/`);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Select Free plan
   */
  async selectFreePlan() {
    await this.page.click(this.freePlanCard);
    await this.page.waitForTimeout(200); // Wait for JS to update
  }

  /**
   * Select Pro plan
   */
  async selectProPlan() {
    await this.page.click(this.proPlanCard);
    await this.page.waitForTimeout(200);
  }

  /**
   * Select billing cycle (monthly or yearly)
   */
  async selectBillingCycle(cycle = 'monthly') {
    if (cycle === 'monthly') {
      await this.page.click(this.monthlyButton);
    } else {
      await this.page.click(this.yearlyButton);
    }
  }

  /**
   * Fill organization information
   */
  async fillOrganizationInfo(orgInfo) {
    await this.page.fill(this.organizationNameInput, orgInfo.name);
    await this.page.fill(this.slugInput, orgInfo.slug);

    // Wait for slug validation
    await this.page.waitForTimeout(500);

    if (orgInfo.contactEmail) {
      await this.page.fill(this.contactEmailInput, orgInfo.contactEmail);
    }
    if (orgInfo.contactPhone) {
      await this.page.fill(this.contactPhoneInput, orgInfo.contactPhone);
    }
    if (orgInfo.address) {
      await this.page.fill(this.addressInput, orgInfo.address);
    }
    await this.page.fill(this.postalCodeInput, orgInfo.postalCode);
    await this.page.fill(this.cityInput, orgInfo.city);
    await this.page.selectOption(this.federalStateSelect, orgInfo.federalState);
  }

  /**
   * Fill admin account information
   */
  async fillAdminInfo(adminInfo) {
    await this.page.fill(this.adminFirstNameInput, adminInfo.firstName);
    await this.page.fill(this.adminLastNameInput, adminInfo.lastName);
    await this.page.fill(this.adminEmailInput, adminInfo.email);
    await this.page.fill(this.adminPasswordInput, adminInfo.password);
  }

  /**
   * Accept terms and conditions
   */
  async acceptTerms() {
    await this.page.check(this.termsCheckbox);
  }

  /**
   * Submit registration form
   */
  async submit() {
    await this.page.click(this.submitButton);
  }

  /**
   * Complete registration with all info
   */
  async register(data) {
    if (data.plan === 'pro') {
      await this.selectProPlan();
      if (data.billingCycle) {
        await this.selectBillingCycle(data.billingCycle);
      }
    } else {
      await this.selectFreePlan();
    }

    await this.fillOrganizationInfo(data.organization);
    await this.fillAdminInfo(data.admin);
    await this.acceptTerms();
    await this.submit();
  }

  /**
   * Check if slug is available
   */
  async isSlugAvailable() {
    const statusText = await this.page.textContent(this.slugStatus);
    return statusText.toLowerCase().includes('verfügbar') ||
           statusText.toLowerCase().includes('available');
  }

  /**
   * Check if slug is unavailable
   */
  async isSlugUnavailable() {
    const statusText = await this.page.textContent(this.slugStatus);
    return statusText.toLowerCase().includes('vergeben') ||
           statusText.toLowerCase().includes('taken');
  }

  /**
   * Wait for slug validation
   */
  async waitForSlugValidation() {
    await this.page.waitForSelector(`${this.slugStatus}:not(:empty)`, { timeout: 3000 });
  }

  /**
   * Check if free plan success message is shown
   */
  async hasFreePlanSuccess() {
    return await this.page.locator(this.successMessageFree).isVisible().catch(() => false);
  }

  /**
   * Check if pro plan success message is shown
   */
  async hasProPlanSuccess() {
    return await this.page.locator(this.successMessagePro).isVisible().catch(() => false);
  }

  /**
   * Get login URL from success message
   */
  async getLoginUrl() {
    const href = await this.page.getAttribute(this.loginLinkFree, 'href');
    return href;
  }

  /**
   * Click login button after successful registration
   */
  async clickLoginButton() {
    await this.page.click(this.loginLinkFree);
  }

  /**
   * Skip payment for Pro plan
   */
  async skipPayment() {
    await this.page.click(this.skipPaymentLink);
  }

  /**
   * Check if registration form is visible
   */
  async isFormVisible() {
    return await this.page.locator(this.registerForm).isVisible().catch(() => false);
  }

  /**
   * Check for error message
   */
  async hasError() {
    const errorAlert = this.page.locator('.alert-error, .alert-danger, .error');
    return await errorAlert.isVisible().catch(() => false);
  }

  /**
   * Get error message text
   */
  async getErrorMessage() {
    const errorAlert = this.page.locator('.alert-error, .alert-danger, .error');
    return await errorAlert.textContent().catch(() => '');
  }
}

module.exports = LandingRegisterPage;
