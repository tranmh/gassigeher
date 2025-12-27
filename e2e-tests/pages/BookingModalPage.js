const BasePage = require('./BasePage');

/**
 * Booking Modal Page Object
 * For creating bookings via the modal dialog
 */
class BookingModalPage extends BasePage {
  constructor(page) {
    super(page);

    // Modal selectors
    this.modal = '#booking-modal';
    this.modalTitle = '#modal-title';  // The booking modal title element
    this.closeButton = '.modal-close, button[data-action="close-booking-modal"], button:has-text("Abbrechen")';

    // Form fields (booking form only has date and time, no walk type)
    this.dateInput = '#booking-date';
    this.timeSelect = '#booking-time';
    this.submitButton = '#booking-form button[type="submit"]';

    // Validation
    this.errorMessage = '.error, .alert-error, .form-error';
    this.successMessage = '.success, .alert-success';
  }

  /**
   * Check if modal is visible
   */
  async isVisible() {
    return await this.page.locator(this.modal).isVisible().catch(() => false);
  }

  /**
   * Wait for modal to appear
   */
  async waitForModal(timeout = 5000) {
    await this.page.waitForSelector(this.modal, { state: 'visible', timeout });
  }

  /**
   * Get modal title
   */
  async getTitle() {
    await this.waitForModal();
    const titleElement = this.page.locator(this.modalTitle).first();
    return await titleElement.textContent();
  }

  /**
   * Fill booking form
   * Note: Booking form only has date and time fields (no walk type)
   * @param {Object} options - { date, time }
   */
  async fillBookingForm({ date, time }) {
    await this.waitForModal();

    if (date) {
      const dateField = this.page.locator(this.dateInput);
      await dateField.fill(date);
      // Trigger change event to load time slots
      await this.page.waitForTimeout(500);
    }

    if (time) {
      // Wait for time options to load (populated dynamically based on date)
      await this.page.waitForTimeout(1000);
      const timeField = this.page.locator(this.timeSelect);
      await timeField.selectOption(time);
    }
  }

  /**
   * Create booking (fill form and submit)
   * Note: walkType parameter is ignored (not used in booking form)
   */
  async createBooking({ date, time, walkType }) {
    // walkType is ignored - booking form only has date and time
    await this.fillBookingForm({ date, time });
    await this.submit();
  }

  /**
   * Submit booking form
   */
  async submit() {
    const submitBtn = this.page.locator(this.submitButton).first();
    await submitBtn.click();
    // Wait for modal to close or error to appear
    await this.page.waitForTimeout(1000);
  }

  /**
   * Close modal
   */
  async close() {
    // Try multiple close methods in order of preference
    // 1. Try the Abbrechen button (most reliable)
    const cancelBtn = this.page.locator('button:has-text("Abbrechen")').first();
    const cancelExists = await cancelBtn.isVisible().catch(() => false);
    if (cancelExists) {
      await cancelBtn.click();
      await this.page.waitForTimeout(500);
      return;
    }

    // 2. Try the × close button
    const closeBtn = this.page.locator('.modal-close, [data-action="close-booking-modal"]').first();
    const closeExists = await closeBtn.isVisible().catch(() => false);
    if (closeExists) {
      await closeBtn.click();
      await this.page.waitForTimeout(500);
      return;
    }

    // 3. Try Escape key
    await this.page.keyboard.press('Escape');
    await this.page.waitForTimeout(500);
  }

  /**
   * Check if error message is shown
   */
  async hasError() {
    return await this.page.locator(this.errorMessage).isVisible().catch(() => false);
  }

  /**
   * Get error message text
   */
  async getErrorMessage() {
    await this.page.waitForSelector(this.errorMessage, { timeout: 3000 });
    return await this.page.locator(this.errorMessage).first().textContent();
  }

  /**
   * Check if success message is shown
   */
  async hasSuccess() {
    return await this.page.locator(this.successMessage).isVisible().catch(() => false);
  }
}

module.exports = BookingModalPage;

// DONE: Booking modal page object for creating bookings
