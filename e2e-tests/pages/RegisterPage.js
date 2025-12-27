const BasePage = require('./BasePage');

/**
 * Register Page Object
 * Mode-aware: Handles both Simple-Mode and SaaS-Mode registration forms
 */
class RegisterPage extends BasePage {
  constructor(page, testInfo = null) {
    super(page, testInfo);

    // Selectors - support both Simple and SaaS form field IDs
    // Note: HTML uses hyphens (first-name), not underscores (first_name)
    this.emailInput = '#email';
    this.nameInput = '#name, #first-name';  // SaaS uses first-name, Simple uses name
    this.firstNameInput = '#first-name';
    this.lastNameInput = '#last-name';
    this.phoneInput = '#phone';
    this.passwordInput = '#password';
    this.confirmPasswordInput = '#confirm-password';
    this.termsCheckbox = '#accept-terms';
    this.submitButton = 'button[type="submit"]';
    this.errorAlert = '.alert-error';
    this.successAlert = '.alert-success';
    this.loginLink = 'a[href="/login.html"], a[href="login.html"]';
  }

  /**
   * Navigate to register page
   */
  async goto() {
    await super.goto('/register.html');
  }

  /**
   * Fill registration form (mode-aware)
   */
  async fillForm({ email, name, firstName, lastName, phone, password, acceptTerms = true }) {
    if (email) await this.page.fill(this.emailInput, email);

    // Handle name fields based on what's available on the page
    // Note: HTML uses hyphens (#first-name), not underscores
    const hasFirstName = await this.page.locator('#first-name').isVisible().catch(() => false);
    if (hasFirstName) {
      // SaaS mode: separate first/last name
      if (firstName) await this.page.fill('#first-name', firstName);
      if (lastName) await this.page.fill('#last-name', lastName);
    } else {
      // Simple mode: combined name field
      const hasName = await this.page.locator('#name').isVisible().catch(() => false);
      if (hasName && name) await this.page.fill('#name', name);
    }

    if (phone) {
      const hasPhone = await this.page.locator(this.phoneInput).isVisible().catch(() => false);
      if (hasPhone) await this.page.fill(this.phoneInput, phone);
    }

    if (password) await this.page.fill(this.passwordInput, password);

    // Handle confirm password if present
    const hasConfirmPassword = await this.page.locator('#confirm-password').isVisible().catch(() => false);
    if (hasConfirmPassword && password) {
      await this.page.fill('#confirm-password', password);
    }

    if (acceptTerms) {
      const checkbox = this.page.locator(this.termsCheckbox);
      if (await checkbox.isVisible()) {
        await checkbox.check();
      }
    }
  }

  /**
   * Submit registration form
   */
  async submit() {
    await this.page.click(this.submitButton);
  }

  /**
   * Register user with all fields
   */
  async register({ email, name, firstName, lastName, phone, password, acceptTerms = true }) {
    await this.fillForm({ email, name, firstName, lastName, phone, password, acceptTerms });
    await this.submit();
  }

  /**
   * Get error message
   */
  async getErrorMessage() {
    await this.page.waitForSelector(this.errorAlert, { timeout: 5000 });
    return await this.page.textContent(this.errorAlert);
  }

  /**
   * Get success message
   */
  async getSuccessMessage() {
    await this.page.waitForSelector(this.successAlert, { timeout: 5000 });
    return await this.page.textContent(this.successAlert);
  }

  /**
   * Check if error is visible
   */
  async hasError() {
    return await this.page.locator(this.errorAlert).isVisible().catch(() => false);
  }

  /**
   * Check if success is visible
   */
  async hasSuccess() {
    return await this.page.locator(this.successAlert).isVisible().catch(() => false);
  }

  /**
   * Go to login page
   */
  async goToLogin() {
    await this.page.locator(this.loginLink).first().click();
    await this.page.waitForURL('**/login.html');
  }
}

module.exports = RegisterPage;
