const BasePage = require('./BasePage');

/**
 * Admin Dogs Page Object
 * For managing dogs in tenant admin interface
 */
class AdminDogsPage extends BasePage {
  constructor(page, subdomain = 'stuttgart', baseDomain = 'gassigeher.local') {
    super(page);
    this.subdomain = subdomain;
    this.baseDomain = baseDomain;
    this.baseURL = `http://${subdomain}.${baseDomain}:8080`;

    // Page Elements
    this.addDogButton = '#add-dog-btn, button:has-text("Hund hinzufügen"), .btn-add-dog';
    this.dogCards = '.dog-card';
    this.dogLimitInfo = '#dog-limit-info';
    this.dogLimitText = '#dog-limit-text';
    this.dogLimitBar = '#dog-limit-bar';

    // Dog Form Modal
    this.dogModal = '#dog-modal, .modal';
    this.dogNameInput = '#dog-name, #name, input[name="name"]';
    this.dogBreedInput = '#dog-breed, #breed, input[name="breed"]';
    this.dogAgeInput = '#dog-age, #age, input[name="age"]';
    this.dogCategorySelect = '#dog-category, #category, select[name="category"]';
    this.dogSizeSelect = '#dog-size, #size, select[name="size"]';
    this.dogDescriptionInput = '#dog-description, #description, textarea[name="description"]';
    this.saveDogButton = '#save-dog-btn, button:has-text("Speichern")';
    this.cancelButton = '.modal button:has-text("Abbrechen"), .modal .btn-cancel';

    // Error/Success Messages
    this.errorAlert = '.alert-error, .alert-danger';
    this.successAlert = '.alert-success';

    // Dog List
    this.dogsContainer = '#dogs-container, .dogs-list, .dogs-grid';
  }

  /**
   * Navigate to admin dogs page
   */
  async goto() {
    await this.page.goto(`${this.baseURL}/admin-dogs.html`);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Update subdomain for this page
   */
  setSubdomain(subdomain) {
    this.subdomain = subdomain;
    this.baseURL = `http://${subdomain}.${this.baseDomain}:8080`;
  }

  /**
   * Click add dog button
   */
  async clickAddDog() {
    await this.page.click(this.addDogButton);
    await this.page.waitForTimeout(300); // Wait for modal animation
  }

  /**
   * Fill dog form
   */
  async fillDogForm(dogData) {
    await this.page.fill(this.dogNameInput, dogData.name);
    await this.page.fill(this.dogBreedInput, dogData.breed);
    await this.page.fill(this.dogAgeInput, String(dogData.age));

    if (dogData.category) {
      await this.page.selectOption(this.dogCategorySelect, dogData.category);
    }
    if (dogData.size) {
      await this.page.selectOption(this.dogSizeSelect, dogData.size);
    }
    if (dogData.description) {
      await this.page.fill(this.dogDescriptionInput, dogData.description);
    }
  }

  /**
   * Save dog form
   */
  async saveDog() {
    await this.page.click(this.saveDogButton);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Add a new dog
   */
  async addDog(dogData) {
    await this.clickAddDog();
    await this.fillDogForm(dogData);
    await this.saveDog();
  }

  /**
   * Get count of dogs displayed
   */
  async getDogCount() {
    const cards = await this.page.locator(this.dogCards).count();
    return cards;
  }

  /**
   * Check if dog limit info is visible
   */
  async isDogLimitVisible() {
    return await this.page.locator(this.dogLimitInfo).isVisible().catch(() => false);
  }

  /**
   * Get dog limit text (e.g., "5 / 10 Hunde")
   */
  async getDogLimitText() {
    return await this.page.textContent(this.dogLimitText).catch(() => '');
  }

  /**
   * Check for error message
   */
  async hasError() {
    return await this.page.locator(this.errorAlert).isVisible().catch(() => false);
  }

  /**
   * Get error message text
   */
  async getErrorMessage() {
    await this.page.waitForSelector(this.errorAlert, { timeout: 3000 }).catch(() => {});
    return await this.page.textContent(this.errorAlert).catch(() => '');
  }

  /**
   * Check for success message
   */
  async hasSuccess() {
    return await this.page.locator(this.successAlert).isVisible().catch(() => false);
  }

  /**
   * Get success message text
   */
  async getSuccessMessage() {
    return await this.page.textContent(this.successAlert).catch(() => '');
  }

  /**
   * Wait for dogs to load
   */
  async waitForDogsToLoad() {
    await this.page.waitForLoadState('networkidle');
    await this.page.waitForTimeout(500);
  }

  /**
   * Check if add dog button is enabled
   */
  async isAddDogEnabled() {
    const button = this.page.locator(this.addDogButton);
    const disabled = await button.getAttribute('disabled');
    return disabled === null;
  }

  /**
   * Close modal if open
   */
  async closeModal() {
    const modalVisible = await this.page.locator(this.dogModal).isVisible().catch(() => false);
    if (modalVisible) {
      await this.page.click(this.cancelButton).catch(() => {});
    }
  }
}

module.exports = AdminDogsPage;
