const BasePage = require('./BasePage');

/**
 * Dashboard Page Object
 * Mode-aware: Uses baseURL from test configuration
 */
class DashboardPage extends BasePage {
  constructor(page, testInfo = null) {
    super(page, testInfo);

    // Selectors
    this.welcomeMessage = 'h1, h2';
    this.upcomingBookingsContainer = '#upcoming-bookings';
    this.upcomingBookings = '#upcoming-bookings .card';  // Bookings are rendered as .card elements
    this.noBookingsMessage = '#upcoming-bookings p[data-i18n="dashboard.no_upcoming_walks"]';
    this.cancelButton = 'button[data-action="cancel-booking"], button:has-text("Stornieren")';
    this.addNotesButton = 'button[data-action="open-walk-report-modal"], button:has-text("Bericht")';
    this.bookingStatus = '[data-status]';

    // Navigation
    this.dogsLink = 'a[href="/dogs.html"]';
    this.profileLink = 'a[href="/profile.html"]';
    this.calendarLink = 'a[href="/calendar.html"]';
    this.logoutLink = 'a:has-text("Abmelden")';
  }

  /**
   * Navigate to dashboard
   */
  async goto() {
    await super.goto('/dashboard.html');
  }

  /**
   * Get number of bookings displayed
   */
  async getBookingCount() {
    await this.page.waitForLoadState('networkidle');
    const count = await this.page.locator(this.upcomingBookings).count();
    return count;
  }

  /**
   * Check if "no bookings" message is shown
   */
  async hasNoBookingsMessage() {
    return await this.page.locator(this.noBookingsMessage).isVisible().catch(() => false);
  }

  /**
   * Get welcome message text
   */
  async getWelcomeMessage() {
    await this.page.waitForLoadState('networkidle');
    const heading = this.page.locator(this.welcomeMessage).first();
    return await heading.textContent().catch(() => '');
  }

  /**
   * Cancel a booking by index
   */
  async cancelBooking(index = 0, reason = 'Test cancellation') {
    const bookingCards = this.page.locator(this.upcomingBookings);
    const card = bookingCards.nth(index);

    await card.locator(this.cancelButton).click();

    // Fill cancellation reason in modal
    const reasonInput = this.page.locator('#cancellation-reason, #cancel-reason, textarea');
    if (await reasonInput.isVisible({ timeout: 2000 }).catch(() => false)) {
      await reasonInput.fill(reason);
    }

    await this.page.click('button:has-text("Bestätigen"), button:has-text("Stornieren")');
    await this.waitForNavigation();
  }

  /**
   * Add notes to booking by index
   */
  async addNotesToBooking(index, notes) {
    const bookingCards = this.page.locator(this.upcomingBookings);
    const card = bookingCards.nth(index);

    await card.locator(this.addNotesButton).click();

    // Fill notes modal
    await this.page.waitForSelector('#booking-notes, #notes, textarea', { timeout: 2000 });
    await this.page.fill('#booking-notes, #notes, textarea', notes);
    await this.page.click('button:has-text("Speichern")');

    await this.waitForNavigation();
  }

  /**
   * Navigate to dogs page
   */
  async goToDogs() {
    await this.page.click(this.dogsLink);
    await this.page.waitForURL('**/dogs.html');
  }

  /**
   * Navigate to profile page
   */
  async goToProfile() {
    await this.page.click(this.profileLink);
    await this.page.waitForURL('**/profile.html');
  }

  /**
   * Navigate to calendar page
   */
  async goToCalendar() {
    await this.page.click(this.calendarLink);
    await this.page.waitForURL('**/calendar.html');
  }

  /**
   * Logout
   */
  async logout() {
    // Logout uses onclick="api.logout()" which redirects to '/' (homepage)
    await this.page.click(this.logoutLink);
    // Wait for redirect to complete
    await this.page.waitForLoadState('networkidle', { timeout: 15000 });
  }
}

module.exports = DashboardPage;
