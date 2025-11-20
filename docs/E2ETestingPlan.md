# E2E Testing Plan - Gassigeher
## Playwright End-to-End Test Strategy

**Goal**: Comprehensive E2E test coverage of all Gassigeher features using Playwright
**Status**: 📋 Planning Complete | 🚧 Implementation Pending
**Target**: 100+ E2E tests covering all user journeys and admin workflows

> **Configuration Based on Requirements**:
> - ✅ Comprehensive coverage of all features
> - ✅ Local development environment
> - ✅ Mock/skip email verification (faster tests)
> - ✅ Chrome desktop + Mobile viewports (dog walkers use phones!)

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture Decisions](#architecture-decisions)
3. [Test Environment Setup](#test-environment-setup)
4. [Test Organization](#test-organization)
5. [Test Data Strategy](#test-data-strategy)
6. [Page Object Model](#page-object-model)
7. [Mobile Testing Strategy](#mobile-testing-strategy)
8. [Test Coverage Matrix](#test-coverage-matrix)
9. [Implementation Phases](#implementation-phases)
10. [Running Tests](#running-tests)
11. [Writing New Tests](#writing-new-tests)
12. [Troubleshooting](#troubleshooting)
13. [Future Enhancements](#future-enhancements)

---

## Overview

### Why E2E Testing?

Gassigeher has:
- **23 HTML pages** (12 public, 3 protected, 8 admin)
- **50+ API endpoints** (already tested at backend level - 62.4% coverage)
- **Complex user journeys** (booking lifecycle, experience levels, admin workflows)
- **Critical business logic** (double-booking prevention, GDPR deletion, access control)

E2E tests ensure the **entire stack works together**: Frontend ↔ API ↔ Database ↔ Business Logic

### Testing Philosophy

```
Backend Unit/Integration Tests (62.4% coverage)
    ↓ Tests business logic, repositories, services

E2E Tests (This Plan)
    ↓ Tests complete user workflows through browser

= Confidence that features work end-to-end
```

**E2E tests catch**:
- UI bugs (broken forms, missing validation messages)
- Integration issues (frontend calling wrong API endpoint)
- UX problems (confusing flows, missing feedback)
- German translation errors (all UI text is in German)
- Mobile usability issues (responsive design problems)

---

## Architecture Decisions

### Decision 1: Playwright for Node.js (Not Go Playwright)

**Chosen**: Node.js Playwright
**Reasoning**:
- ✅ More mature ecosystem (better docs, active development)
- ✅ Superior debugging tools (UI mode, trace viewer, inspector)
- ✅ Easier test writing (JavaScript for UI tests is simpler)
- ✅ Better mobile device emulation
- ✅ Faster test development iteration

Go excels at backend testing (already at 62.4%). JavaScript excels at browser automation.

### Decision 2: Page Object Model (POM)

**Chosen**: Implement POM pattern
**Reasoning**:
- ✅ 100+ tests planned - code duplication would be nightmare
- ✅ UI changes affect multiple tests - POM centralizes selectors
- ✅ Easier for team members to write tests (reusable page methods)
- ✅ Better maintainability (change selector once, not 20 times)

**Example**:
```javascript
// Without POM (BAD)
await page.fill('#email', 'test@example.com');
await page.fill('#password', 'test123');
await page.click('button[type="submit"]');

// With POM (GOOD)
const loginPage = new LoginPage(page);
await loginPage.login('test@example.com', 'test123');
```

### Decision 3: Mock Email Verification

**Chosen**: Bypass email verification via direct database updates
**Reasoning**:
- ✅ Faster test execution (no waiting for emails)
- ✅ No external dependencies (Gmail API, test email service)
- ✅ More reliable (no flaky network/API issues)
- ✅ Backend already tested - just need to test UI flows

**Implementation**:
```javascript
// After user registration via UI
await dbHelper.verifyUser(email); // Sets is_verified=1 directly
```

Email templates are tested separately (backend has email service tests).

### Decision 4: Chrome + Mobile Viewports

**Chosen**: Desktop Chrome + Mobile emulation
**Reasoning**:
- ✅ Dog walkers use phones frequently (mobile critical)
- ✅ Chrome desktop covers majority of desktop users
- ✅ Mobile emulation faster than real devices
- ✅ Can expand to Firefox/Safari later if needed

**Viewports**:
- Desktop: 1920x1080 (standard)
- Mobile: iPhone 13 (390x844), Pixel 5 (393x851)

### Decision 5: Fresh Database Per Test

**Chosen**: Reset SQLite database before each test file
**Reasoning**:
- ✅ Test isolation (no flaky tests from shared state)
- ✅ Predictable starting point (known seed data)
- ✅ SQLite is fast (setup takes ~100ms)
- ✅ No complex cleanup logic needed

---

## Test Environment Setup

### Prerequisites

- Node.js 18+ installed
- Go application (gassigeher.exe) built
- SQLite database

### Installation Steps

#### 1. Create E2E Test Directory

```bash
# From project root
mkdir e2e-tests
cd e2e-tests
```

#### 2. Initialize Node.js Project

```bash
npm init -y
npm install -D @playwright/test
npm install -D sqlite3  # For database seeding/cleanup
npx playwright install chromium
```

#### 3. Directory Structure

```
e2e-tests/
├── tests/                          # Test specs
│   ├── 01-public-pages.spec.js
│   ├── 02-authentication.spec.js
│   ├── 03-user-profile.spec.js
│   ├── 04-dog-browsing.spec.js
│   ├── 05-booking-user.spec.js
│   ├── 06-calendar.spec.js
│   ├── 07-experience-requests.spec.js
│   ├── 08-admin-dogs.spec.js
│   ├── 09-admin-users.spec.js
│   ├── 10-admin-bookings.spec.js
│   ├── 11-admin-experience.spec.js
│   ├── 12-admin-reactivation.spec.js
│   ├── 13-admin-settings.spec.js
│   ├── 14-admin-blocked-dates.spec.js
│   └── 15-edge-cases.spec.js
├── pages/                          # Page Object Model
│   ├── BasePage.js
│   ├── LoginPage.js
│   ├── RegisterPage.js
│   ├── DashboardPage.js
│   ├── DogsPage.js
│   ├── ProfilePage.js
│   ├── BookingModalPage.js
│   ├── CalendarPage.js
│   ├── AdminDogsPage.js
│   ├── AdminUsersPage.js
│   ├── AdminBookingsPage.js
│   └── AdminSettingsPage.js
├── fixtures/                       # Test fixtures
│   ├── auth.js                     # Login helpers, auth state
│   ├── database.js                 # DB setup, seed, cleanup
│   └── test-data.js                # Sample data constants
├── utils/                          # Utilities
│   ├── db-helpers.js               # Direct DB manipulation
│   ├── german-text.js              # German translations for assertions
│   └── date-helpers.js             # Date formatting utilities
├── playwright.config.js            # Playwright configuration
├── global-setup.js                 # One-time setup (build app, etc)
├── global-teardown.js              # One-time cleanup
├── package.json
└── README.md                       # Quick start guide
```

#### 4. Playwright Configuration

**File**: `e2e-tests/playwright.config.js`

```javascript
// @ts-check
const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',

  // Test execution
  fullyParallel: false,  // Run sequentially for easier debugging locally
  workers: 1,            // One worker = sequential execution
  retries: 0,            // No retries locally (fast feedback)
  timeout: 30 * 1000,    // 30s per test

  // Reporting
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],  // Console output
  ],

  use: {
    // Base URL for all tests
    baseURL: 'http://localhost:8080',

    // Browser options
    headless: false,  // See browser during local dev
    viewport: { width: 1920, height: 1080 },

    // Screenshots and videos
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    trace: 'retain-on-failure',

    // Timeouts
    actionTimeout: 10 * 1000,
    navigationTimeout: 15 * 1000,
  },

  projects: [
    {
      name: 'chromium-desktop',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1920, height: 1080 },
      },
    },
    {
      name: 'mobile-iphone',
      use: {
        ...devices['iPhone 13'],
      },
    },
    {
      name: 'mobile-android',
      use: {
        ...devices['Pixel 5'],
      },
    },
  ],

  // Start Go server automatically
  webServer: {
    command: 'cd .. && .\\gassigeher.exe',  // Windows
    // command: 'cd .. && ./gassigeher',     // Linux/Mac
    url: 'http://localhost:8080',
    reuseExistingServer: true,  // Don't restart if already running
    timeout: 30 * 1000,
    env: {
      DATABASE_PATH: './e2e-tests/test.db',  // Separate test database
      PORT: '8080',
      JWT_SECRET: 'test-jwt-secret-for-e2e-only',
      ADMIN_EMAILS: 'admin@test.com',
    },
  },

  globalSetup: require.resolve('./global-setup.js'),
  globalTeardown: require.resolve('./global-teardown.js'),
});
```

#### 5. Global Setup/Teardown

**File**: `e2e-tests/global-setup.js`

```javascript
const { chromium } = require('@playwright/test');
const { setupDatabase, seedInitialData } = require('./fixtures/database');

module.exports = async config => {
  console.log('🔧 Global setup: Preparing test environment...');

  // Setup test database
  await setupDatabase();
  await seedInitialData();

  // Pre-authenticate admin user (speeds up admin tests)
  const browser = await chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();

  await page.goto('http://localhost:8080/login.html');
  await page.fill('#email', 'admin@test.com');
  await page.fill('#password', 'admin123');
  await page.click('button[type="submit"]');
  await page.waitForURL('**/dashboard.html');

  // Save admin auth state
  await context.storageState({ path: 'admin-storage-state.json' });
  await browser.close();

  console.log('✅ Global setup complete');
};
```

**File**: `e2e-tests/global-teardown.js`

```javascript
const { cleanupDatabase } = require('./fixtures/database');

module.exports = async config => {
  console.log('🧹 Global teardown: Cleaning up...');
  await cleanupDatabase();
  console.log('✅ Global teardown complete');
};
```

---

## Test Organization

### Test File Naming Convention

Pattern: `XX-feature-name.spec.js` where XX is execution order.

**Why numbered?**
- Sequential execution ensures test isolation
- Dependencies clear (e.g., profile tests need login working)
- Easier to understand test flow

### Test Categories (15 Files, ~100-150 Tests)

#### 1. Public Pages (01-public-pages.spec.js)
**Tests**: ~10 tests
- Homepage loads correctly
- Terms & Conditions page accessible
- Privacy Policy page accessible
- German translations present
- Navigation links work
- Footer links work

#### 2. Authentication (02-authentication.spec.js)
**Tests**: ~15 tests
- **Registration**:
  - Valid registration creates user
  - Validation errors shown (invalid email, weak password, etc)
  - Terms acceptance required
  - Duplicate email prevented
- **Login**:
  - Valid credentials redirect to dashboard
  - Invalid credentials show error
  - Unverified user shows warning
  - Remember me checkbox works
- **Logout**:
  - Logout clears session
  - Redirects to login
- **Password Reset**:
  - Forgot password form works
  - (Mock) Reset link sent
  - Password reset with valid token works
  - Invalid token shows error

#### 3. User Profile (03-user-profile.spec.js)
**Tests**: ~12 tests
- View profile shows correct data
- Update name works
- Update phone works
- Update email triggers re-verification
- Upload profile photo works (JPEG, PNG)
- Delete profile photo works
- Invalid file types rejected (PDF, etc)
- File size limit enforced
- GDPR account deletion flow:
  - Confirmation modal shown
  - Deletion succeeds
  - User anonymized (can still see past walk history)
  - Cannot login after deletion

#### 4. Dog Browsing (04-dog-browsing.spec.js)
**Tests**: ~15 tests
- View all dogs
- Filter by breed
- Filter by size (small, medium, large)
- Filter by age range (young, adult, senior)
- Filter by experience level (green, blue, orange)
- Search by name
- Multiple filters combined
- Available dogs only filter
- Experience level locking (🔒 icon shown for higher levels)
- Pagination works (if >20 dogs)
- Dog detail card shows all info (photo, description, category)
- Unavailable dogs show reason
- Clear filters button works

#### 5. Booking - User Flows (05-booking-user.spec.js)
**Tests**: ~20 tests
- **Create Booking**:
  - Click dog → modal opens
  - Select date, time, walk type
  - Create booking succeeds
  - Confirmation shown
  - Booking appears in dashboard
- **Validation**:
  - Cannot book past dates
  - Cannot book beyond advance limit (14 days)
  - Cannot book blocked dates
  - Cannot double-book same dog/time
  - Cannot book dog above experience level
  - Cannot book unavailable dogs
- **View Bookings**:
  - Dashboard shows all bookings
  - Filter by status (scheduled, completed, cancelled)
  - Filter by date range
- **Cancel Booking**:
  - Cancel button works
  - Cancellation reason required
  - Cannot cancel within notice period (12h)
  - Cancelled booking shows in history
- **Add Walk Notes** (completed bookings):
  - Add notes button shown
  - Notes saved successfully
  - Notes visible in booking details

#### 6. Calendar View (06-calendar.spec.js)
**Tests**: ~10 tests
- Calendar displays current month
- Navigate to next/previous month
- Today highlighted
- Blocked dates shown in red
- User's bookings shown
- Click date shows availability
- Quick booking from calendar works
- Month with 28, 30, 31 days rendered correctly
- Calendar on mobile (swipe gestures)

#### 7. Experience Level Requests (07-experience-requests.spec.js)
**Tests**: ~8 tests
- **User Actions**:
  - Request upgrade to Blue (from Green)
  - Cannot request Orange directly (must have Blue first)
  - Cannot request same level twice
  - Pending request shows status
  - View request history
- **Validation**:
  - Cannot request downgrade
  - Cannot have multiple pending requests

#### 8. Admin - Dogs Management (08-admin-dogs.spec.js)
**Tests**: ~15 tests
- List all dogs
- Create new dog:
  - All fields filled correctly
  - Upload photo
  - Validation errors shown
- Edit dog:
  - Update name, breed, description
  - Update category (green/blue/orange)
  - Change photo
- Delete dog:
  - Cannot delete with future bookings
  - Can delete with only past bookings
  - Confirmation modal shown
- Toggle availability:
  - Mark unavailable with reason
  - Mark available again (reason cleared)
  - Reason shown to users

#### 9. Admin - Users Management (09-admin-users.spec.js)
**Tests**: ~12 tests
- List all users
- Filter active/inactive users
- View user details
- Deactivate user:
  - Reason required
  - User cannot login after
  - Past bookings preserved
- Activate user:
  - Reactivation succeeds
  - User can login again
  - Optional welcome message
- User search functionality
- Export users list (if implemented)

#### 10. Admin - Bookings Management (10-admin-bookings.spec.js)
**Tests**: ~12 tests
- List all bookings
- Filter by status (all, scheduled, completed, cancelled)
- Filter by date range
- Filter by dog
- Filter by user
- View booking details (user + dog info)
- Move booking:
  - Change date
  - Cannot move to blocked date
  - Cannot move to double-booked slot
  - Cannot move completed booking
  - Reason required
- Search bookings

#### 11. Admin - Experience Requests (11-admin-experience.spec.js)
**Tests**: ~8 tests
- List all experience requests
- Filter pending/approved/denied
- View request details (user info, current level, requested level)
- Approve request:
  - User level updated immediately
  - Request marked approved
  - User can now book higher-level dogs
- Deny request:
  - Reason required
  - Request marked denied
  - User level unchanged

#### 12. Admin - Reactivation Requests (12-admin-reactivation.spec.js)
**Tests**: ~8 tests
- List all reactivation requests
- Filter pending/approved/denied
- View request details (user info, deactivation reason)
- Approve request:
  - User reactivated immediately
  - User can login
- Deny request:
  - Reason required
  - User remains deactivated

#### 13. Admin - Settings (13-admin-settings.spec.js)
**Tests**: ~6 tests
- View all system settings
- Update booking advance days (default 14)
- Update cancellation notice hours (default 12)
- Update auto-deactivation days (default 365)
- Validation (positive numbers only)
- Settings persist after save

#### 14. Admin - Blocked Dates (14-admin-blocked-dates.spec.js)
**Tests**: ~8 tests
- List all blocked dates
- Add blocked date:
  - Select date
  - Reason required
  - Cannot add duplicate date
- Delete blocked date:
  - Confirmation shown
  - Date becomes available for booking
- Users cannot book blocked dates
- Calendar shows blocked dates

#### 15. Edge Cases & Business Rules (15-edge-cases.spec.js)
**Tests**: ~15 tests
- **Double Booking Prevention**:
  - Two users cannot book same dog/time
  - Racing condition handled
- **Experience Level Enforcement**:
  - Green user cannot book Orange dog (even via direct URL)
  - Blue user can book Green and Blue dogs
  - Orange user can book any dog
- **Booking Time Windows**:
  - Cannot book beyond advance limit
  - Cannot cancel within notice period
  - Cannot add notes to non-completed bookings
- **GDPR Compliance**:
  - Deleted user data anonymized
  - Walk history preserved
  - Anonymous ID generated
- **Auto-Completion** (via cron):
  - Past bookings auto-completed
  - Status changes correctly
- **Token Expiration**:
  - Expired JWT redirects to login
  - Expired password reset token shows error
- **Admin Authorization**:
  - Non-admin cannot access admin pages (404 or redirect)
  - Direct URL access blocked

---

## Test Data Strategy

### Seed Data (Created in global-setup.js)

#### Users (6 users)
```javascript
const USERS = {
  GREEN_USER: {
    email: 'green@test.com',
    password: 'test123',
    name: 'Green User',
    experience_level: 'green',
    is_verified: 1,
  },
  BLUE_USER: {
    email: 'blue@test.com',
    password: 'test123',
    name: 'Blue User',
    experience_level: 'blue',
    is_verified: 1,
  },
  ORANGE_USER: {
    email: 'orange@test.com',
    password: 'test123',
    name: 'Orange User',
    experience_level: 'orange',
    is_verified: 1,
  },
  ADMIN_USER: {
    email: 'admin@test.com',
    password: 'admin123',
    name: 'Admin User',
    experience_level: 'orange',
    is_verified: 1,
  },
  UNVERIFIED_USER: {
    email: 'unverified@test.com',
    password: 'test123',
    name: 'Unverified User',
    is_verified: 0,
  },
  INACTIVE_USER: {
    email: 'inactive@test.com',
    password: 'test123',
    name: 'Inactive User',
    is_active: 0,
    deactivation_reason: 'Test deactivation',
  },
};
```

#### Dogs (9 dogs - 3 per category)
```javascript
const DOGS = {
  GREEN_DOG_1: { name: 'Luna', category: 'green', breed: 'Golden Retriever', is_available: 1 },
  GREEN_DOG_2: { name: 'Max', category: 'green', breed: 'Labrador', is_available: 1 },
  GREEN_DOG_3: { name: 'Bella', category: 'green', breed: 'Beagle', is_available: 0, unavailable_reason: 'In training' },

  BLUE_DOG_1: { name: 'Rocky', category: 'blue', breed: 'German Shepherd', is_available: 1 },
  BLUE_DOG_2: { name: 'Daisy', category: 'blue', breed: 'Border Collie', is_available: 1 },
  BLUE_DOG_3: { name: 'Charlie', category: 'blue', breed: 'Husky', is_available: 1 },

  ORANGE_DOG_1: { name: 'Rex', category: 'orange', breed: 'Rottweiler', is_available: 1 },
  ORANGE_DOG_2: { name: 'Zeus', category: 'orange', breed: 'Doberman', is_available: 1 },
  ORANGE_DOG_3: { name: 'Thor', category: 'orange', breed: 'Pitbull', is_available: 0, unavailable_reason: 'Veterinary care' },
};
```

#### System Settings
```javascript
const SETTINGS = {
  BOOKING_ADVANCE_DAYS: 14,
  CANCELLATION_NOTICE_HOURS: 12,
  AUTO_DEACTIVATION_DAYS: 365,
};
```

### Database Helpers

**File**: `e2e-tests/utils/db-helpers.js`

```javascript
const sqlite3 = require('sqlite3').verbose();

class DBHelper {
  constructor(dbPath) {
    this.db = new sqlite3.Database(dbPath);
  }

  async createUser(userData) {
    return new Promise((resolve, reject) => {
      const sql = `INSERT INTO users (email, name, password_hash, experience_level, is_verified, terms_accepted_at)
                   VALUES (?, ?, ?, ?, ?, datetime('now'))`;
      this.db.run(sql, [
        userData.email,
        userData.name,
        '$2a$10$hashedpassword', // Bcrypt hash for "test123"
        userData.experience_level,
        userData.is_verified,
      ], function(err) {
        if (err) reject(err);
        resolve(this.lastID);
      });
    });
  }

  async verifyUser(email) {
    return new Promise((resolve, reject) => {
      const sql = `UPDATE users SET is_verified = 1 WHERE email = ?`;
      this.db.run(sql, [email], err => {
        if (err) reject(err);
        resolve();
      });
    });
  }

  async createDog(dogData) {
    return new Promise((resolve, reject) => {
      const sql = `INSERT INTO dogs (name, breed, category, is_available, unavailable_reason, created_at)
                   VALUES (?, ?, ?, ?, ?, datetime('now'))`;
      this.db.run(sql, [
        dogData.name,
        dogData.breed,
        dogData.category,
        dogData.is_available,
        dogData.unavailable_reason || null,
      ], function(err) {
        if (err) reject(err);
        resolve(this.lastID);
      });
    });
  }

  async createBooking(bookingData) {
    return new Promise((resolve, reject) => {
      const sql = `INSERT INTO bookings (user_id, dog_id, date, walk_type, scheduled_time, status, created_at)
                   VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`;
      this.db.run(sql, [
        bookingData.user_id,
        bookingData.dog_id,
        bookingData.date,
        bookingData.walk_type,
        bookingData.scheduled_time,
        bookingData.status || 'scheduled',
      ], function(err) {
        if (err) reject(err);
        resolve(this.lastID);
      });
    });
  }

  async blockDate(date, reason) {
    return new Promise((resolve, reject) => {
      const sql = `INSERT INTO blocked_dates (date, reason, created_at)
                   VALUES (?, ?, datetime('now'))`;
      this.db.run(sql, [date, reason], function(err) {
        if (err) reject(err);
        resolve(this.lastID);
      });
    });
  }

  async resetDatabase() {
    // Delete all data but keep schema
    const tables = ['bookings', 'experience_requests', 'reactivation_requests', 'blocked_dates', 'dogs', 'users'];
    for (const table of tables) {
      await new Promise((resolve, reject) => {
        this.db.run(`DELETE FROM ${table}`, err => {
          if (err) reject(err);
          resolve();
        });
      });
    }
  }

  close() {
    this.db.close();
  }
}

module.exports = DBHelper;
```

---

## Page Object Model

### Base Page Class

**File**: `e2e-tests/pages/BasePage.js`

```javascript
class BasePage {
  constructor(page) {
    this.page = page;
  }

  async goto(path) {
    await this.page.goto(path);
  }

  async waitForNavigation() {
    await this.page.waitForLoadState('networkidle');
  }

  async getAlertText(type = 'success') {
    const selector = `.alert-${type}`;
    await this.page.waitForSelector(selector, { timeout: 5000 });
    return await this.page.textContent(selector);
  }

  async clickNavLink(text) {
    await this.page.click(`nav a:has-text("${text}")`);
  }

  async isLoggedIn() {
    // Check if dashboard/logout link visible
    return await this.page.locator('a[href="/dashboard.html"]').isVisible();
  }
}

module.exports = BasePage;
```

### Login Page

**File**: `e2e-tests/pages/LoginPage.js`

```javascript
const BasePage = require('./BasePage');

class LoginPage extends BasePage {
  constructor(page) {
    super(page);
    this.emailInput = '#email';
    this.passwordInput = '#password';
    this.submitButton = 'button[type="submit"]';
    this.errorAlert = '.alert-danger';
  }

  async goto() {
    await this.page.goto('/login.html');
  }

  async login(email, password) {
    await this.page.fill(this.emailInput, email);
    await this.page.fill(this.passwordInput, password);
    await this.page.click(this.submitButton);
  }

  async loginAndWait(email, password) {
    await this.login(email, password);
    await this.page.waitForURL('**/dashboard.html');
  }

  async getErrorMessage() {
    return await this.page.textContent(this.errorAlert);
  }
}

module.exports = LoginPage;
```

### Dashboard Page

**File**: `e2e-tests/pages/DashboardPage.js`

```javascript
const BasePage = require('./BasePage');

class DashboardPage extends BasePage {
  constructor(page) {
    super(page);
    this.upcomingBookings = '.booking-card';
    this.cancelButton = 'button.cancel-booking';
    this.addNotesButton = 'button.add-notes';
  }

  async goto() {
    await this.page.goto('/dashboard.html');
  }

  async getBookingCount() {
    return await this.page.locator(this.upcomingBookings).count();
  }

  async cancelBooking(index = 0) {
    const bookingCards = this.page.locator(this.upcomingBookings);
    const card = bookingCards.nth(index);
    await card.locator(this.cancelButton).click();

    // Fill cancellation reason
    await this.page.fill('#cancellation-reason', 'Test cancellation');
    await this.page.click('button:has-text("Bestätigen")');
  }

  async addNotesToBooking(index, notes) {
    const bookingCards = this.page.locator(this.upcomingBookings);
    const card = bookingCards.nth(index);
    await card.locator(this.addNotesButton).click();

    // Fill notes modal
    await this.page.fill('#booking-notes', notes);
    await this.page.click('button:has-text("Speichern")');
  }
}

module.exports = DashboardPage;
```

### Dogs Page

**File**: `e2e-tests/pages/DogsPage.js`

```javascript
const BasePage = require('./BasePage');

class DogsPage extends BasePage {
  constructor(page) {
    super(page);
    this.dogCards = '.dog-card';
    this.breedFilter = '#filter-breed';
    this.categoryFilter = '#filter-category';
    this.availableOnlyCheckbox = '#filter-available';
    this.searchInput = '#search-dogs';
    this.bookButton = 'button.btn-book';
  }

  async goto() {
    await this.page.goto('/dogs.html');
  }

  async getDogCount() {
    await this.page.waitForLoadState('networkidle');
    return await this.page.locator(this.dogCards).count();
  }

  async filterByBreed(breed) {
    await this.page.selectOption(this.breedFilter, breed);
    await this.page.waitForLoadState('networkidle');
  }

  async filterByCategory(category) {
    await this.page.selectOption(this.categoryFilter, category);
    await this.page.waitForLoadState('networkidle');
  }

  async filterAvailableOnly() {
    await this.page.check(this.availableOnlyCheckbox);
    await this.page.waitForLoadState('networkidle');
  }

  async searchDogs(query) {
    await this.page.fill(this.searchInput, query);
    await this.page.waitForLoadState('networkidle');
  }

  async clickBookButtonForDog(index = 0) {
    const dogCards = this.page.locator(this.dogCards);
    const card = dogCards.nth(index);
    await card.locator(this.bookButton).click();

    // Wait for booking modal to open
    await this.page.waitForSelector('#booking-modal', { state: 'visible' });
  }

  async isDogLocked(index) {
    const dogCards = this.page.locator(this.dogCards);
    const card = dogCards.nth(index);
    return await card.locator('.lock-icon').isVisible();
  }
}

module.exports = DogsPage;
```

### Booking Modal Page

**File**: `e2e-tests/pages/BookingModalPage.js`

```javascript
class BookingModalPage {
  constructor(page) {
    this.page = page;
    this.modal = '#booking-modal';
    this.dateInput = '#booking-date';
    this.walkTypeSelect = '#booking-walk-type';
    this.timeSelect = '#booking-time';
    this.submitButton = '#booking-form button[type="submit"]';
    this.closeButton = '.modal-close';
  }

  async isVisible() {
    return await this.page.locator(this.modal).isVisible();
  }

  async createBooking(date, walkType, time) {
    await this.page.fill(this.dateInput, date);
    await this.page.selectOption(this.walkTypeSelect, walkType);
    await this.page.selectOption(this.timeSelect, time);
    await this.page.click(this.submitButton);
  }

  async close() {
    await this.page.click(this.closeButton);
  }
}

module.exports = BookingModalPage;
```

**Note**: Create similar page objects for admin pages (AdminDogsPage, AdminUsersPage, etc.)

---

## Mobile Testing Strategy

### Why Mobile Testing Matters

Dog walkers are often **on-site at the shelter** or **outdoors**. Mobile usage is critical:
- Check dog availability on the go
- Book walks while at shelter
- Add walk notes after walks
- View calendar on mobile

### Mobile Test Approach

#### 1. Device Emulation (Not Real Devices)

**Chosen devices**:
- iPhone 13 (390x844) - iOS users
- Pixel 5 (393x851) - Android users

**Why emulation?**
- ✅ Faster execution
- ✅ Consistent results (no real device flakiness)
- ✅ Good enough for responsive design testing
- ❌ Cannot test real Safari quirks (but Chrome covers 90%)

#### 2. Test Subset (Not All 150 Tests)

Run critical flows on mobile, not everything:
- ✅ Login
- ✅ Browse dogs
- ✅ Create booking
- ✅ View dashboard
- ✅ View calendar
- ❌ Skip admin flows (admins use desktop)

**Estimated**: 20-30 mobile-specific tests

#### 3. Mobile-Specific Assertions

```javascript
test.describe('Mobile - Dog Browsing', () => {
  test.use({ ...devices['iPhone 13'] });

  test('should display dogs in single column on mobile', async ({ page }) => {
    const dogsPage = new DogsPage(page);
    await dogsPage.goto();

    // Check layout
    const dogCards = page.locator('.dog-card');
    const firstCard = dogCards.first();
    const boundingBox = await firstCard.boundingBox();

    // Card should be nearly full width on mobile
    expect(boundingBox.width).toBeGreaterThan(350); // ~90% of 390px viewport
  });

  test('should support touch interactions for booking', async ({ page }) => {
    const dogsPage = new DogsPage(page);
    await dogsPage.goto();

    // Tap (not click) to open booking modal
    await page.tap('.dog-card:first-child .btn-book');

    // Modal should open
    const modal = page.locator('#booking-modal');
    await expect(modal).toBeVisible();
  });
});
```

#### 4. Viewport-Specific CSS Checks

```javascript
test('calendar should stack vertically on mobile', async ({ page }) => {
  test.use({ ...devices['iPhone 13'] });

  const calendarPage = new CalendarPage(page);
  await calendarPage.goto();

  // Desktop: 7 columns (week days)
  // Mobile: Should show fewer or scrollable
  const calendarGrid = page.locator('.calendar-grid');
  const styles = await calendarGrid.evaluate(el => {
    return window.getComputedStyle(el).gridTemplateColumns;
  });

  // Mobile should not try to fit 7 columns
  expect(styles).not.toContain('repeat(7');
});
```

---

## Test Coverage Matrix

Comprehensive view of what's tested:

### Features Covered (All 50+ Features)

| Feature Category | Desktop | Mobile | Notes |
|-----------------|---------|--------|-------|
| **Authentication** |
| Registration | ✅ | ✅ | Form validation, duplicate check |
| Login | ✅ | ✅ | Valid/invalid credentials |
| Logout | ✅ | ✅ | Session cleared |
| Password Reset | ✅ | ❌ | Desktop only (complex flow) |
| Email Verification | ✅ | ❌ | Mocked via DB |
| **User Profile** |
| View Profile | ✅ | ✅ | All fields displayed |
| Update Info | ✅ | ✅ | Name, email, phone |
| Upload Photo | ✅ | ✅ | JPEG/PNG, size limits |
| Delete Account (GDPR) | ✅ | ❌ | Desktop only (critical action) |
| **Dogs** |
| Browse All Dogs | ✅ | ✅ | List view |
| Filter by Breed | ✅ | ✅ | Dropdown filter |
| Filter by Category | ✅ | ✅ | Green/Blue/Orange |
| Search by Name | ✅ | ✅ | Case-insensitive |
| View Dog Details | ✅ | ✅ | Photo, description, availability |
| Experience Level Locking | ✅ | ✅ | 🔒 icon shown |
| **Bookings** |
| Create Booking | ✅ | ✅ | Date, time, walk type |
| View Bookings | ✅ | ✅ | Dashboard list |
| Cancel Booking | ✅ | ✅ | With reason |
| Add Walk Notes | ✅ | ✅ | Completed bookings |
| Booking Validation | ✅ | ✅ | All rules enforced |
| **Calendar** |
| View Calendar | ✅ | ✅ | Month view |
| Navigate Months | ✅ | ✅ | Prev/Next |
| Blocked Dates Display | ✅ | ✅ | Red highlight |
| Quick Booking | ✅ | ✅ | Click date → book |
| **Experience Requests** |
| Request Upgrade | ✅ | ❌ | Desktop only |
| View Request Status | ✅ | ❌ | Desktop only |
| **Admin - Dogs** |
| List Dogs | ✅ | ❌ | Admin desktop only |
| Create Dog | ✅ | ❌ | Admin desktop only |
| Edit Dog | ✅ | ❌ | Admin desktop only |
| Delete Dog | ✅ | ❌ | Admin desktop only |
| Toggle Availability | ✅ | ❌ | Admin desktop only |
| **Admin - Users** |
| List Users | ✅ | ❌ | Admin desktop only |
| Deactivate User | ✅ | ❌ | Admin desktop only |
| Activate User | ✅ | ❌ | Admin desktop only |
| **Admin - Bookings** |
| List All Bookings | ✅ | ❌ | Admin desktop only |
| Move Booking | ✅ | ❌ | Admin desktop only |
| **Admin - Experience Requests** |
| List Requests | ✅ | ❌ | Admin desktop only |
| Approve Request | ✅ | ❌ | Admin desktop only |
| Deny Request | ✅ | ❌ | Admin desktop only |
| **Admin - Reactivation** |
| List Requests | ✅ | ❌ | Admin desktop only |
| Approve Request | ✅ | ❌ | Admin desktop only |
| Deny Request | ✅ | ❌ | Admin desktop only |
| **Admin - Settings** |
| View Settings | ✅ | ❌ | Admin desktop only |
| Update Settings | ✅ | ❌ | Admin desktop only |
| **Admin - Blocked Dates** |
| List Blocked Dates | ✅ | ❌ | Admin desktop only |
| Add Blocked Date | ✅ | ❌ | Admin desktop only |
| Delete Blocked Date | ✅ | ❌ | Admin desktop only |
| **Edge Cases** |
| Double Booking Prevention | ✅ | ❌ | Backend + frontend |
| Experience Level Enforcement | ✅ | ✅ | Tested on both |
| Token Expiration | ✅ | ❌ | Desktop only |
| GDPR Anonymization | ✅ | ❌ | Desktop only |

**Coverage Summary**:
- Desktop: 50+ features (100%)
- Mobile: ~25 critical user features (50%)
- Total Tests: ~130-150 tests

---

## Implementation Phases

### Phase 1: Foundation (Week 1) - 20% Complete

**Goal**: Setup infrastructure, basic tests working

**Tasks**:
- [ ] Create e2e-tests directory structure
- [ ] Install Playwright and dependencies
- [ ] Create playwright.config.js
- [ ] Implement global-setup.js and global-teardown.js
- [ ] Create BasePage class
- [ ] Create DBHelper class
- [ ] Write first 3 test files:
  - [ ] 01-public-pages.spec.js (5 tests)
  - [ ] 02-authentication.spec.js (10 tests)
  - [ ] 03-user-profile.spec.js (5 basic tests)
- [ ] Verify tests run successfully
- [ ] Document setup in e2e-tests/README.md

**Deliverable**: 20 passing tests, foundation ready

---

### Phase 2: User Flows (Week 2-3) - 50% Complete

**Goal**: Complete all user-facing features

**Tasks**:
- [ ] Create Page Objects:
  - [ ] LoginPage, RegisterPage, DashboardPage
  - [ ] DogsPage, BookingModalPage, CalendarPage
  - [ ] ProfilePage
- [ ] Write test files:
  - [ ] 04-dog-browsing.spec.js (15 tests)
  - [ ] 05-booking-user.spec.js (20 tests)
  - [ ] 06-calendar.spec.js (10 tests)
  - [ ] 07-experience-requests.spec.js (8 tests)
- [ ] Implement test data fixtures
- [ ] Add German text helpers

**Deliverable**: 70+ passing tests, all user flows covered

---

### Phase 3: Admin Flows (Week 4-5) - 80% Complete

**Goal**: Complete all admin features

**Tasks**:
- [ ] Create Admin Page Objects:
  - [ ] AdminDogsPage, AdminUsersPage, AdminBookingsPage
  - [ ] AdminExperiencePage, AdminReactivationPage
  - [ ] AdminSettingsPage, AdminBlockedDatesPage
- [ ] Write admin test files:
  - [ ] 08-admin-dogs.spec.js (15 tests)
  - [ ] 09-admin-users.spec.js (12 tests)
  - [ ] 10-admin-bookings.spec.js (12 tests)
  - [ ] 11-admin-experience.spec.js (8 tests)
  - [ ] 12-admin-reactivation.spec.js (8 tests)
  - [ ] 13-admin-settings.spec.js (6 tests)
  - [ ] 14-admin-blocked-dates.spec.js (8 tests)

**Deliverable**: 135+ passing tests, all admin flows covered

---

### Phase 4: Edge Cases & Mobile (Week 6) - 100% Complete

**Goal**: Edge cases, mobile tests, polish

**Tasks**:
- [ ] Write 15-edge-cases.spec.js (15 tests)
- [ ] Add mobile-specific tests (20-30 tests)
- [ ] Screenshot comparison tests (optional)
- [ ] Performance checks (load times)
- [ ] Accessibility audit (optional)
- [ ] Polish flaky tests
- [ ] Optimize test execution speed

**Deliverable**: 150+ passing tests, mobile coverage, production-ready

---

### Phase 5: Documentation & Maintenance (Week 7) - Ongoing

**Goal**: Documentation, CI/CD (future), team training

**Tasks**:
- [ ] Complete e2e-tests/README.md
- [ ] Add inline comments to complex tests
- [ ] Create troubleshooting guide
- [ ] Document common patterns
- [ ] (Future) CI/CD integration guide
- [ ] (Future) GitHub Actions workflow

**Deliverable**: Fully documented, maintainable test suite

---

## Running Tests

### Run All Tests

```bash
cd e2e-tests
npm test
```

### Run Specific Test File

```bash
# Run only authentication tests
npx playwright test tests/02-authentication.spec.js

# Run only admin tests
npx playwright test tests/08-admin-dogs.spec.js
```

### Run by Tag/Grep

```bash
# Run tests matching pattern
npx playwright test -g "login"

# Run tests in specific describe block
npx playwright test -g "User Profile"
```

### Run on Specific Browser/Device

```bash
# Desktop Chrome only
npx playwright test --project=chromium-desktop

# Mobile iPhone only
npx playwright test --project=mobile-iphone

# All mobile
npx playwright test --project=mobile-*
```

### Debug Mode

```bash
# Run with browser visible (headed mode)
npx playwright test --headed

# Run in debug mode (step through)
npx playwright test --debug

# Run specific test in debug mode
npx playwright test tests/05-booking-user.spec.js:42 --debug
```

### Interactive UI Mode (Best for Development)

```bash
# Opens interactive UI
npx playwright test --ui
```

Features:
- See all tests in tree view
- Run individual tests
- Step through test execution
- View screenshots/videos
- Inspect DOM at any point

### View Test Report

```bash
# After test run, open HTML report
npx playwright show-report
```

### Run with Tracing

```bash
# Generate trace files for failed tests
npx playwright test --trace on

# View trace
npx playwright show-trace trace.zip
```

Trace shows:
- Screenshots at each step
- Network requests
- Console logs
- DOM snapshots

---

## Writing New Tests

### Test Template

```javascript
const { test, expect } = require('@playwright/test');
const LoginPage = require('../pages/LoginPage');
const DogsPage = require('../pages/DogsPage');

test.describe('Feature Name', () => {
  test.beforeEach(async ({ page }) => {
    // Setup: Login or navigate to starting point
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.loginAndWait('test@example.com', 'test123');
  });

  test('should do something successfully', async ({ page }) => {
    // Arrange
    const dogsPage = new DogsPage(page);
    await dogsPage.goto();

    // Act
    await dogsPage.searchDogs('Luna');

    // Assert
    const count = await dogsPage.getDogCount();
    expect(count).toBe(1);
  });

  test('should show error for invalid action', async ({ page }) => {
    // Test error case
    const dogsPage = new DogsPage(page);
    await dogsPage.goto();

    await dogsPage.searchDogs('NonExistentDog');

    const count = await dogsPage.getDogCount();
    expect(count).toBe(0);
    await expect(page.locator('.no-results')).toBeVisible();
  });
});
```

### Best Practices

#### 1. Use Page Objects (Don't Repeat Selectors)

```javascript
// ❌ BAD: Selectors scattered in tests
test('login', async ({ page }) => {
  await page.fill('#email', 'test@example.com');
  await page.fill('#password', 'test123');
  await page.click('button[type="submit"]');
});

// ✅ GOOD: Selectors in Page Object
test('login', async ({ page }) => {
  const loginPage = new LoginPage(page);
  await loginPage.login('test@example.com', 'test123');
});
```

#### 2. Use German Text Constants

```javascript
// Create utils/german-text.js
const GERMAN_TEXT = {
  LOGIN_SUCCESS: 'Erfolgreich angemeldet',
  BOOKING_CREATED: 'Buchung erfolgreich erstellt',
  BOOKING_CANCELLED: 'Buchung storniert',
  INVALID_EMAIL: 'Ungültige E-Mail-Adresse',
  // ... more
};

// Use in tests
const { GERMAN_TEXT } = require('../utils/german-text');

test('should show success message', async ({ page }) => {
  // ... create booking
  const alert = await page.textContent('.alert-success');
  expect(alert).toContain(GERMAN_TEXT.BOOKING_CREATED);
});
```

#### 3. Wait for Network Idle (AJAX Calls)

```javascript
// ❌ BAD: Race condition
await page.click('.filter-breed');
const count = await page.locator('.dog-card').count(); // Might be old data

// ✅ GOOD: Wait for network
await page.click('.filter-breed');
await page.waitForLoadState('networkidle');
const count = await page.locator('.dog-card').count();
```

#### 4. Use Fixtures for Authentication

```javascript
// Create fixtures/auth.js
const { test as base } = require('@playwright/test');
const LoginPage = require('../pages/LoginPage');

const test = base.extend({
  authenticatedPage: async ({ page }, use) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.loginAndWait('test@example.com', 'test123');
    await use(page);
  },
});

// Use in tests
test('should see dashboard after login', async ({ authenticatedPage }) => {
  // Already logged in!
  await expect(authenticatedPage).toHaveURL(/dashboard\.html/);
});
```

#### 5. Reset Data Between Tests

```javascript
const DBHelper = require('../utils/db-helpers');

test.beforeEach(async () => {
  const db = new DBHelper('../test.db');
  await db.resetDatabase();
  await db.seedBasicData();
  db.close();
});
```

---

## Troubleshooting

### Common Issues

#### 1. Test Timeouts

**Symptom**: Test hangs and fails after 30s

**Causes**:
- Waiting for element that never appears
- Network request never completes
- Modal not opening

**Solutions**:
```javascript
// Increase timeout for specific action
await page.click('button', { timeout: 60000 }); // 60s

// Or for whole test
test('slow test', async ({ page }) => {
  test.setTimeout(120000); // 2 minutes
  // ... test code
});

// Debug: See what's on page
await page.screenshot({ path: 'debug.png' });
console.log(await page.content()); // Full HTML
```

#### 2. Element Not Found

**Symptom**: `Error: Selector "#my-button" not found`

**Causes**:
- Wrong selector
- Element not rendered yet
- Element in iframe

**Solutions**:
```javascript
// Wait for element to appear
await page.waitForSelector('#my-button', { state: 'visible' });

// Check if element exists
const exists = await page.locator('#my-button').count() > 0;

// Use more robust selector
await page.click('button:has-text("Speichern")'); // Text-based

// If in iframe
const frame = page.frameLocator('iframe#myframe');
await frame.locator('#my-button').click();
```

#### 3. Flaky Tests (Randomly Fail)

**Symptoms**: Test passes sometimes, fails sometimes

**Causes**:
- Race conditions (DOM updates mid-test)
- Network timing
- Animations not complete

**Solutions**:
```javascript
// Bad: Fixed wait (brittle)
await page.waitForTimeout(1000); // ❌

// Good: Wait for condition
await page.waitForLoadState('networkidle'); // ✅
await page.waitForSelector('.dog-card'); // ✅

// Wait for specific state
await expect(page.locator('.dog-card')).toHaveCount(5);

// Retry assertions
await expect(async () => {
  const count = await page.locator('.dog-card').count();
  expect(count).toBe(5);
}).toPass({ timeout: 5000 });
```

#### 4. Database Locked

**Symptom**: `Error: database is locked`

**Cause**: SQLite file in use by Go server and tests simultaneously

**Solution**:
```javascript
// Use separate test database
// In playwright.config.js
webServer: {
  env: {
    DATABASE_PATH: './e2e-tests/test.db',  // Separate DB
  },
}
```

#### 5. German Text Not Matching

**Symptom**: Assertion fails on German text

**Cause**: Special characters, whitespace, translations changed

**Solution**:
```javascript
// Use partial match
expect(text).toContain('erfolgreich'); // ✅

// Case-insensitive
expect(text.toLowerCase()).toContain('erfolgreich');

// Normalize whitespace
expect(text.trim()).toBe('Buchung erstellt');

// Regular expression
expect(text).toMatch(/Buchung.*erstellt/);
```

### Debugging Workflow

1. **Run test in headed mode**: See what's happening
   ```bash
   npx playwright test --headed tests/05-booking-user.spec.js
   ```

2. **Add screenshots at failure point**:
   ```javascript
   await page.screenshot({ path: 'before-click.png' });
   await page.click('button');
   await page.screenshot({ path: 'after-click.png' });
   ```

3. **Use debug mode**: Step through test
   ```bash
   npx playwright test --debug tests/05-booking-user.spec.js
   ```

4. **Inspect page state**:
   ```javascript
   console.log(await page.locator('.dog-card').count());
   console.log(await page.textContent('.alert'));
   console.log(await page.url());
   ```

5. **Check network requests**:
   ```javascript
   page.on('request', req => console.log('→', req.method(), req.url()));
   page.on('response', res => console.log('←', res.status(), res.url()));
   ```

6. **Use trace viewer**:
   ```bash
   npx playwright test --trace on
   npx playwright show-trace trace.zip
   ```

---

## Future Enhancements

### Phase 6+: CI/CD Integration (When Ready)

**GitHub Actions Workflow**:

**File**: `.github/workflows/e2e-tests.yml`

```yaml
name: E2E Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v3

    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'

    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Build Go application
      run: go build -o gassigeher ./cmd/server

    - name: Install Playwright
      working-directory: ./e2e-tests
      run: |
        npm ci
        npx playwright install --with-deps chromium

    - name: Run E2E tests
      working-directory: ./e2e-tests
      run: npm test
      env:
        CI: true

    - name: Upload test results
      if: always()
      uses: actions/upload-artifact@v3
      with:
        name: playwright-report
        path: e2e-tests/playwright-report/

    - name: Upload videos on failure
      if: failure()
      uses: actions/upload-artifact@v3
      with:
        name: test-videos
        path: e2e-tests/test-results/
```

### Visual Regression Testing

**Tool**: Playwright's `toHaveScreenshot()`

```javascript
test('homepage looks correct', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveScreenshot('homepage.png');
});

// First run: Creates baseline
// Future runs: Compares against baseline
```

### Accessibility Testing

**Tool**: `@axe-core/playwright`

```bash
npm install -D @axe-core/playwright
```

```javascript
const { injectAxe, checkA11y } = require('axe-playwright');

test('homepage should be accessible', async ({ page }) => {
  await page.goto('/');
  await injectAxe(page);
  await checkA11y(page);
});
```

### Performance Testing

```javascript
test('dogs page should load quickly', async ({ page }) => {
  const start = Date.now();
  await page.goto('/dogs.html');
  await page.waitForLoadState('networkidle');
  const loadTime = Date.now() - start;

  expect(loadTime).toBeLessThan(3000); // < 3 seconds
});
```

### Cross-Browser Testing (Firefox, Safari)

Update playwright.config.js:

```javascript
projects: [
  { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  { name: 'firefox', use: { ...devices['Desktop Firefox'] } },
  { name: 'webkit', use: { ...devices['Desktop Safari'] } },  // WebKit
],
```

### Test Data Generators

Use libraries like Faker.js for dynamic test data:

```bash
npm install -D @faker-js/faker
```

```javascript
const { faker } = require('@faker-js/faker');

test('register with random user', async ({ page }) => {
  const email = faker.internet.email();
  const name = faker.person.fullName();
  const phone = faker.phone.number('+49 ### #######');

  // ... use in registration
});
```

---

## Success Metrics

### Coverage Goals

- ✅ **100% of user-facing features tested** (50+ features)
- ✅ **100% of admin features tested** (15+ admin workflows)
- ✅ **Critical mobile flows tested** (booking, browsing)
- ✅ **All business rules validated** (experience levels, double-booking, GDPR)

### Quality Metrics

- ⏱️ **Test suite runs in < 30 minutes** (local dev)
- 🎯 **0% flaky tests** (all tests deterministic)
- 📊 **150+ passing tests**
- 🐛 **Catches regressions before deployment**

### Maintenance

- 📝 **Page Objects for all pages** (easy updates)
- 📖 **Comprehensive documentation**
- 🔄 **Easy for team to add new tests**

---

## Summary

This E2E testing plan provides:

1. ✅ **Comprehensive coverage** - All 50+ features tested
2. ✅ **Mobile testing** - Critical flows on iPhone/Android
3. ✅ **Maintainable** - Page Object Model pattern
4. ✅ **Fast feedback** - Local dev environment
5. ✅ **Mock email** - No external dependencies
6. ✅ **German language** - All assertions in German
7. ✅ **Phase approach** - Implement gradually over 6 weeks

**Next Steps**:
1. Review this plan
2. Start Phase 1 implementation
3. Setup e2e-tests directory
4. Write first 20 tests
5. Iterate and expand

---

**Status**: 📋 Plan Complete | Ready for implementation
**Estimated Effort**: 6 weeks part-time (or 3 weeks full-time)
**Expected Outcome**: 150+ passing E2E tests, production-ready test suite

🎯 **Goal**: Zero regressions reach production. Every feature validated end-to-end.
