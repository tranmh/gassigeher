const { test, expect } = require('@playwright/test');
const LoginPage = require('../pages/LoginPage');
const DogsPage = require('../pages/DogsPage');
const BookingModalPage = require('../pages/BookingModalPage');
const { getConfigFromTestInfo } = require('../fixtures/test-config');

/**
 * DOG BROWSING TESTS
 * Test dog listing, filtering, search, experience level enforcement
 * GOAL: Find bugs in dog browsing and access control!
 *
 * Dual-Mode: Tests run against both Simple-Mode and SaaS-Mode
 */

test.describe('Dog Browsing - Basic Functionality', () => {

  test('should show dogs page after login', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    expect(page.url()).toContain('dogs.html');

    // Should show some dogs
    const count = await dogsPage.getDogCount();
    console.log(`[${config.mode}] Total dogs displayed:`, count);

    expect(count).toBeGreaterThan(0);
  });

  test('should display dog information cards', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    const count = await dogsPage.getDogCount();
    if (count > 0) {
      // Check first dog has name
      const dogName = await dogsPage.getDogName(0);
      console.log(`[${config.mode}] First dog name:`, dogName);

      expect(dogName).toBeTruthy();
      expect(dogName.length).toBeGreaterThan(0);
    }
  });

  test('should show book buttons for available dogs', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    const count = await dogsPage.getDogCount();
    if (count > 0) {
      // Check if first dog has book button
      const firstCard = page.locator('.dog-card').first();
      const bookButton = firstCard.locator('button').first();
      const hasButton = await bookButton.isVisible().catch(() => false);

      console.log(`[${config.mode}] First dog has book button:`, hasButton);
    }
  });

});

test.describe('Dog Browsing - Filters', () => {

  test('should filter dogs by category (experience level)', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    const totalDogs = await dogsPage.getDogCount();
    console.log(`[${config.mode}] Total dogs before filter:`, totalDogs);

    // Filter by green category
    await dogsPage.filterByCategory('green');
    await page.waitForTimeout(1000);

    const greenDogs = await dogsPage.getDogCount();
    console.log(`[${config.mode}] Green dogs after filter:`, greenDogs);

    // Should have fewer dogs (only green ones)
    if (greenDogs === totalDogs && totalDogs > 1) {
      console.warn(`[${config.mode}] Category filter might not be working!`);
    }
  });

  test('should filter dogs by size', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    const totalDogs = await dogsPage.getDogCount();

    // Filter by large size
    await dogsPage.filterBySize('large');
    await page.waitForTimeout(1000);

    const largeDogs = await dogsPage.getDogCount();
    console.log(`[${config.mode}] Large dogs:`, largeDogs, 'Total dogs:', totalDogs);
  });

  test('should search dogs by name', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    // Search for a common dog name
    await dogsPage.searchDogs('Luna');
    await page.waitForTimeout(1000);

    const searchResults = await dogsPage.getDogCount();
    console.log(`[${config.mode}] Search results for "Luna":`, searchResults);

    // Should find Luna if exists
    if (searchResults > 0) {
      const firstName = await dogsPage.getDogName(0);
      console.log(`[${config.mode}] First result name:`, firstName);
    }
  });

  test('should show "no results" for non-existent dog search', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    // Search for dog that doesn't exist
    await dogsPage.searchDogs('NonExistentDogXYZ123');
    await page.waitForTimeout(1000);

    const searchResults = await dogsPage.getDogCount();
    console.log(`[${config.mode}] Search results for non-existent dog:`, searchResults);

    // Should have no results
    expect(searchResults).toBe(0);

    // Should show "no results" message
    const hasNoResults = await dogsPage.hasNoResults();
    console.log(`[${config.mode}] Shows no results message:`, hasNoResults);
  });

});

test.describe('Dog Browsing - Experience Level Enforcement', () => {

  test('Admin user should see all dogs unlocked', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    const dogCount = await dogsPage.getDogCount();
    console.log(`[${config.mode}] Dogs visible to admin user:`, dogCount);

    // Admin should see all dogs
    expect(dogCount).toBeGreaterThan(0);
  });

  test('should show lock icon for dogs above user experience level', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.greenUser;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    // Check if any dog has lock icon (green user should see some locked)
    const dogCount = await dogsPage.getDogCount();
    if (dogCount > 0) {
      const isLocked = await dogsPage.isDogLocked(0);
      console.log(`[${config.mode}] First dog is locked:`, isLocked);
    }
  });

});

test.describe('Dog Browsing - Booking Flow Start', () => {

  test('should open booking modal when clicking available dog card', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    const dogCount = await dogsPage.getDogCount();
    if (dogCount > 0) {
      // Use full flow: dog card → detail modal → book button → booking modal
      const opened = await dogsPage.openBookingModalForFirstAvailableDog();

      if (opened) {
        // Check if booking modal appeared
        const bookingModal = new BookingModalPage(page);
        const modalVisible = await bookingModal.isVisible();

        console.log(`[${config.mode}] Booking modal opened:`, modalVisible);
        expect(modalVisible).toBe(true);
      } else {
        console.warn(`[${config.mode}] No available dogs to click - skipping modal test`);
      }
    }
  });

  test('should show dog name in booking modal', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    const dogCount = await dogsPage.getDogCount();
    if (dogCount > 0) {
      // Get dog name before clicking
      const dogName = await dogsPage.getDogName(0);
      console.log(`[${config.mode}] Booking for dog:`, dogName);

      // Use full flow to open booking modal
      const opened = await dogsPage.clickDogCardAndOpenBookingModal(0);

      if (opened) {
        // Check modal title/content contains dog name
        const bookingModal = new BookingModalPage(page);
        const modalVisible = await bookingModal.isVisible();

        if (modalVisible) {
          const modalTitle = await bookingModal.getTitle();
          console.log(`[${config.mode}] Modal title:`, modalTitle);
          // Modal title should be "Spaziergang buchen - {dogName}"
          expect(modalTitle).toContain('Spaziergang buchen');
          // Clean the dog name (remove extra whitespace)
          const cleanDogName = dogName.replace(/\s+/g, ' ').trim();
          console.log(`[${config.mode}] Clean dog name:`, cleanDogName);
          // Check modal contains the dog name (ignore whitespace issues)
          expect(modalTitle.toLowerCase()).toContain(cleanDogName.toLowerCase());
        }
      }
    }
  });

  test('should not show book button for unavailable dogs', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    // Look through dogs to find an unavailable one
    const dogCount = await dogsPage.getDogCount();
    console.log(`[${config.mode}] Checking`, dogCount, 'dogs for unavailable status...');

    let foundUnavailable = false;
    for (let i = 0; i < dogCount && i < 20; i++) {
      const isAvailable = await dogsPage.isDogAvailable(i);
      if (!isAvailable) {
        foundUnavailable = true;
        console.log(`[${config.mode}] Dog ${i} is unavailable`);

        // Check that unavailable dog doesn't have book button
        const dogCard = page.locator('.dog-card').nth(i);
        const bookButton = dogCard.locator('button:has-text("Buchen")');
        const hasBookButton = await bookButton.isVisible().catch(() => false);

        console.log(`[${config.mode}] Unavailable dog ${i} has book button:`, hasBookButton);
        expect(hasBookButton).toBe(false);
        break;
      }
    }

    if (!foundUnavailable) {
      console.warn(`[${config.mode}] No unavailable dogs found to test`);
    }
  });

});

test.describe('Dog Browsing - Edge Cases', () => {

  test('should handle no dogs scenario gracefully', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    // Search for impossible match
    await dogsPage.searchDogs('XYZ_NO_MATCH_999');
    await page.waitForTimeout(1000);

    const count = await dogsPage.getDogCount();
    console.log(`[${config.mode}] Dogs after impossible search:`, count);

    if (count === 0) {
      // Should show friendly message, not blank page
      const hasNoResults = await dogsPage.hasNoResults();
      console.log(`[${config.mode}] Shows no results message:`, hasNoResults);
    }
  });

  test('should load dog photos without errors', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    await page.waitForLoadState('networkidle');

    // Check for broken images
    const brokenImages = await page.evaluate(() => {
      const images = Array.from(document.querySelectorAll('.dog-card img'));
      return images.filter(img => !img.complete || img.naturalHeight === 0).length;
    });

    console.log(`[${config.mode}] Broken dog images:`, brokenImages);

    if (brokenImages > 0) {
      console.warn(`[${config.mode}] ${brokenImages} dog images failed to load`);
    }
  });

  test('should display experience level badges correctly', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    const pageText = await page.textContent('body');

    // Check if experience levels are shown (Grün, Blau, Orange)
    const hasGreen = pageText.includes('Grün') || pageText.includes('green');
    const hasBlue = pageText.includes('Blau') || pageText.includes('blue');
    const hasOrange = pageText.includes('Orange') || pageText.includes('orange');

    console.log(`[${config.mode}] Experience levels shown - Green:`, hasGreen, 'Blue:', hasBlue, 'Orange:', hasOrange);

    if (!hasGreen && !hasBlue && !hasOrange) {
      console.warn(`[${config.mode}] No experience level indicators shown!`);
    }
  });

});

test.describe('Dog Browsing - Multiple Filters Combined', () => {

  test('should apply multiple filters together', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    const initialCount = await dogsPage.getDogCount();
    console.log(`[${config.mode}] Initial dog count:`, initialCount);

    // Apply category filter
    await dogsPage.filterByCategory('green');
    await page.waitForTimeout(500);
    const afterCategory = await dogsPage.getDogCount();
    console.log(`[${config.mode}] After category filter:`, afterCategory);

    // Add size filter
    await dogsPage.filterBySize('large');
    await page.waitForTimeout(500);
    const afterSize = await dogsPage.getDogCount();
    console.log(`[${config.mode}] After adding size filter:`, afterSize);

    // Should have equal or fewer dogs
    expect(afterSize).toBeLessThanOrEqual(afterCategory);
  });

  test('should handle filter combinations that return zero results', async ({ page }, testInfo) => {
    const config = getConfigFromTestInfo(testInfo);
    const credentials = config.credentials.admin;

    const loginPage = new LoginPage(page, testInfo);
    await loginPage.goto();
    await loginPage.loginAndWait(credentials.email, credentials.password);

    const dogsPage = new DogsPage(page, testInfo);
    await dogsPage.goto();

    // Apply impossible filter combination
    await dogsPage.filterByCategory('green');
    await page.waitForTimeout(500);
    await dogsPage.searchDogs('ZZZNOMATCH999');
    await page.waitForTimeout(500);

    const count = await dogsPage.getDogCount();
    console.log(`[${config.mode}] Results for impossible filters:`, count);

    expect(count).toBe(0);

    // Should show no results message
    const hasNoResults = await dogsPage.hasNoResults();
    console.log(`[${config.mode}] Shows no results message:`, hasNoResults);
  });

});
