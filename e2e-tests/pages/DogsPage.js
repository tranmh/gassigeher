const BasePage = require('./BasePage');

/**
 * Dogs Page Object
 * For browsing and filtering dogs
 * Mode-aware: Uses baseURL from test configuration
 */
class DogsPage extends BasePage {
  constructor(page, testInfo = null) {
    super(page, testInfo);

    // Selectors (from actual rendered HTML)
    this.dogCards = '.dog-card';
    this.dogName = '.dog-card-title';
    this.dogCardBody = '.dog-card-body';
    this.lockedBanner = '.dog-locked-banner';
    this.unavailableBanner = '.dog-unavailable-banner';
    this.categoryBadge = '.dog-category-badge';

    // Filters (from actual HTML)
    this.breedFilter = '#filter-breed';
    this.colorFilter = '#filter-color';  // Experience level/category is called "color" in the UI
    this.sizeFilter = '#filter-size';
    this.searchInput = '#filter-search';
    this.applyFiltersButton = 'button[data-action="apply-filters"]';
    this.resetFiltersButton = 'button[data-action="reset-filters"]';

    // No results - container shows "Keine Hunde gefunden" text
    this.noResultsMessage = '#dogs-list p[data-i18n="dogs.no_dogs"]';
    this.dogsContainer = '#dogs-list';
  }

  /**
   * Navigate to dogs page
   */
  async goto() {
    await super.goto('/dogs.html');
  }

  /**
   * Get number of dog cards displayed
   */
  async getDogCount() {
    await this.page.waitForLoadState('networkidle');
    const count = await this.page.locator(this.dogCards).count();
    return count;
  }

  /**
   * Check if "no results" message is shown
   */
  async hasNoResults() {
    // Check for the "Keine Hunde gefunden" message or similar
    const noDogsMsg = await this.page.locator(this.noResultsMessage).isVisible().catch(() => false);
    if (noDogsMsg) return true;

    // Also check container text content for "Keine Hunde" phrase
    const containerText = await this.page.locator(this.dogsContainer).textContent().catch(() => '');
    return containerText.includes('Keine Hunde');
  }

  /**
   * Filter by breed
   */
  async filterByBreed(breed) {
    await this.page.selectOption(this.breedFilter, breed);
    await this.page.click(this.applyFiltersButton);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Filter by category (experience level / color)
   * Maps category names to actual dropdown options
   * @param {string} category - 'green', 'yellow', 'orange', 'lightblue', 'darkblue' or German equivalents
   */
  async filterByCategory(category) {
    // Map English category names to German dropdown text (partial match)
    const categoryMap = {
      'green': 'Gruen',
      'gruen': 'Gruen',
      'yellow': 'Gelb',
      'gelb': 'Gelb',
      'orange': 'Orange',
      'lightblue': 'Hellblau',
      'hellblau': 'Hellblau',
      'darkblue': 'Dunkelblau',
      'dunkelblau': 'Dunkelblau',
      'blue': 'Dunkelblau'  // Default blue to darkblue
    };

    const searchText = categoryMap[category.toLowerCase()] || category;

    // Find and select the option by label text (contains search text)
    const selectElement = this.page.locator(this.colorFilter);
    const options = selectElement.locator('option');
    const count = await options.count();

    for (let i = 0; i < count; i++) {
      const optionText = await options.nth(i).textContent();
      if (optionText.includes(searchText)) {
        const value = await options.nth(i).getAttribute('value');
        await selectElement.selectOption(value);
        break;
      }
    }

    await this.page.click(this.applyFiltersButton);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Filter by size
   */
  async filterBySize(size) {
    await this.page.selectOption(this.sizeFilter, size);
    await this.page.click(this.applyFiltersButton);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Search dogs by name
   */
  async searchDogs(query) {
    await this.page.fill(this.searchInput, query);
    await this.page.click(this.applyFiltersButton);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Clear all filters
   */
  async resetFilters() {
    await this.page.click(this.resetFiltersButton);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Click dog card to open dog detail modal
   * Only works for accessible, available dogs!
   * Note: This opens the DETAIL modal, not the booking modal directly
   */
  async clickDogCard(index = 0) {
    const dogCards = this.page.locator(this.dogCards);
    const card = dogCards.nth(index);

    // Check if dog is clickable (not locked or unavailable)
    const classes = await card.getAttribute('class') || '';
    const isLocked = classes.includes('locked');
    const isUnavailable = classes.includes('unavailable');

    if (isLocked || isUnavailable) {
      console.warn(`⚠️ Dog at index ${index} is locked or unavailable, cannot click`);
      return false;
    }

    // Click the dog card to open detail modal
    await card.click();
    await this.page.waitForTimeout(1000);
    return true;
  }

  /**
   * Click dog card and then the book button in detail modal to open booking modal
   * This is the full flow: dog card → detail modal → booking modal
   */
  async clickDogCardAndOpenBookingModal(index = 0) {
    const clicked = await this.clickDogCard(index);
    if (!clicked) return false;

    // Wait for detail modal to appear
    await this.page.waitForSelector('#dog-detail-modal', { state: 'visible', timeout: 5000 }).catch(() => null);

    // Click the booking button inside detail modal
    const bookButton = this.page.locator('button[data-action="book-dog"]');
    const bookButtonVisible = await bookButton.isVisible().catch(() => false);

    if (bookButtonVisible) {
      await bookButton.click();
      await this.page.waitForTimeout(1000);
      return true;
    }

    return false;
  }

  /**
   * Find and click first AVAILABLE dog to open detail modal
   * Returns true if a dog was clicked
   */
  async clickFirstAvailableDog() {
    const dogCards = this.page.locator(this.dogCards);
    const count = await dogCards.count();

    for (let i = 0; i < count; i++) {
      const card = dogCards.nth(i);
      const classes = await card.getAttribute('class') || '';
      const isAvailable = !classes.includes('locked') && !classes.includes('unavailable');

      if (isAvailable) {
        console.log(`Found available dog at index ${i}`);
        await card.click();
        await this.page.waitForTimeout(1000);
        return true;
      }
    }

    console.error('❌ No available dogs found to click!');
    return false;
  }

  /**
   * Find first AVAILABLE dog and open the booking modal (full flow)
   * Dog card → Detail modal → Book button → Booking modal
   */
  async openBookingModalForFirstAvailableDog() {
    const dogCards = this.page.locator(this.dogCards);
    const count = await dogCards.count();

    for (let i = 0; i < count; i++) {
      const card = dogCards.nth(i);
      const classes = await card.getAttribute('class') || '';
      const isAvailable = !classes.includes('locked') && !classes.includes('unavailable');

      if (isAvailable) {
        console.log(`Found available dog at index ${i}, opening booking modal...`);
        return await this.clickDogCardAndOpenBookingModal(i);
      }
    }

    console.error('❌ No available dogs found to open booking modal!');
    return false;
  }

  /**
   * Alias for clickDogCard (for backwards compatibility)
   */
  async clickBookButton(index = 0) {
    await this.clickDogCard(index);
  }

  /**
   * Check if dog is locked (experience level too high)
   */
  async isDogLocked(index = 0) {
    const dogCards = this.page.locator(this.dogCards);
    const card = dogCards.nth(index);
    const classes = await card.getAttribute('class') || '';
    const hasLockedClass = classes.includes('locked');
    const hasLockedBanner = await card.locator(this.lockedBanner).isVisible().catch(() => false);
    return hasLockedClass || hasLockedBanner;
  }

  /**
   * Get dog name by index
   */
  async getDogName(index = 0) {
    const dogCards = this.page.locator(this.dogCards);
    const card = dogCards.nth(index);
    const nameElement = card.locator(this.dogName).first();
    return await nameElement.textContent();
  }

  /**
   * Check if dog is available (not unavailable)
   */
  async isDogAvailable(index = 0) {
    const dogCards = this.page.locator(this.dogCards);
    const card = dogCards.nth(index);
    const cardText = await card.textContent();
    return !cardText.toLowerCase().includes('nicht verfügbar');
  }
}

module.exports = DogsPage;
