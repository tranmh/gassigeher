/**
 * XSS File Tests - TDD Tests that Check Actual HTML Files
 *
 * These tests read the actual HTML files and verify that:
 * 1. sanitize.js is included
 * 2. User data is passed through sanitizeHTML()
 *
 * TDD Workflow:
 * 1. RED: Tests fail because HTML files have vulnerabilities
 * 2. GREEN: Fix HTML files to include sanitize.js and use sanitizeHTML()
 * 3. REFACTOR: Clean up code while keeping tests green
 */

const fs = require('fs');
const path = require('path');

// Path to frontend HTML files
const FRONTEND_PATH = path.join(__dirname, '../internal/static/frontend');

// Helper to read HTML file
function readHtmlFile(filename) {
  return fs.readFileSync(path.join(FRONTEND_PATH, filename), 'utf8');
}

describe('TDD: admin-blocked-dates.html XSS fixes', () => {
  let html;

  beforeAll(() => {
    html = readHtmlFile('admin-blocked-dates.html');
  });

  test('should include sanitize.js', () => {
    expect(html).toContain('src="/js/sanitize.js"');
  });

  test('should sanitize dog.name in loadDogs()', () => {
    // The file should use sanitizeHTML(dog.name) instead of just dog.name
    // Pattern: ${sanitizeHTML(dog.name)}
    expect(html).toMatch(/\$\{sanitizeHTML\(dog\.name\)\}/);
  });

  test('should sanitize blocked.dog_name in renderBlockedDates()', () => {
    // The file should use sanitizeHTML(blocked.dog_name)
    expect(html).toMatch(/sanitizeHTML\(blocked\.dog_name/);
  });

  test('should sanitize blocked.reason in renderBlockedDates()', () => {
    // The file should use sanitizeHTML(blocked.reason)
    expect(html).toMatch(/\$\{sanitizeHTML\(blocked\.reason\)\}/);
  });
});

describe('TDD: dashboard.html XSS fixes', () => {
  let html;

  beforeAll(() => {
    html = readHtmlFile('dashboard.html');
  });

  test('should include sanitize.js', () => {
    expect(html).toContain('src="/js/sanitize.js"');
  });

  test('should sanitize error.message in loadUpcomingBookings()', () => {
    // Should use sanitizeHTML(error.message)
    expect(html).toMatch(/sanitizeHTML\(error\.message\)/);
  });
});

describe('TDD: dogs.html XSS fixes', () => {
  let html;

  beforeAll(() => {
    html = readHtmlFile('dogs.html');
  });

  test('should include sanitize.js', () => {
    expect(html).toContain('src="/js/sanitize.js"');
  });

  test('should sanitize error.message in viewDog()', () => {
    // Should use sanitizeHTML(error.message)
    expect(html).toMatch(/sanitizeHTML\(error\.message\)/);
  });
});

describe('TDD: profile.html XSS fixes', () => {
  let html;

  beforeAll(() => {
    html = readHtmlFile('profile.html');
  });

  test('should include sanitize.js', () => {
    expect(html).toContain('src="/js/sanitize.js"');
  });

  test('should sanitize request.admin_message', () => {
    // Should use sanitizeHTML(request.admin_message)
    expect(html).toMatch(/sanitizeHTML\(request\.admin_message\)/);
  });
});

describe('TDD: admin-users.html onclick XSS fixes', () => {
  let html;

  beforeAll(() => {
    html = readHtmlFile('admin-users.html');
  });

  test('should NOT pass userName through onclick for demoteAdmin', () => {
    // The vulnerable pattern: onclick="demoteAdmin(${user.id}, '${safeName...}')"
    // The safe pattern: onclick="demoteAdmin(${user.id})"
    // Check that we DON'T have the vulnerable pattern
    expect(html).not.toMatch(/onclick="demoteAdmin\(\$\{user\.id\},\s*'\$\{safeName/);
  });

  test('should NOT pass userName through onclick for promoteToAdmin', () => {
    expect(html).not.toMatch(/onclick="promoteToAdmin\(\$\{user\.id\},\s*'\$\{safeName/);
  });

  test('should NOT pass userName through onclick for impersonateUser', () => {
    expect(html).not.toMatch(/onclick="impersonateUser\(\$\{user\.id\},\s*'\$\{safeName/);
  });

  test('should NOT pass userName through onclick for showDeleteModal', () => {
    expect(html).not.toMatch(/onclick="showDeleteModal\(\$\{user\.id\},\s*'\$\{safeName/);
  });
});
