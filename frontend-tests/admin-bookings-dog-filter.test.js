/**
 * admin-bookings.html — Dog Filter regression tests
 *
 * Locks in:
 *  1. The filter label translates to a proper singular dog word in de/en
 *     (bug where it previously resolved to "Name").
 *  2. The inline loadBookings() does not re-fetch dogs that
 *     loadDogsForFilter() has already cached — N+1 request bug.
 *
 * @jest-environment jsdom
 */

const fs = require('fs');
const path = require('path');

const HTML_PATH = path.join(__dirname, '..', 'internal', 'static', 'frontend', 'admin-bookings.html');
const DE_PATH   = path.join(__dirname, '..', 'internal', 'static', 'frontend', 'i18n', 'de.json');
const EN_PATH   = path.join(__dirname, '..', 'internal', 'static', 'frontend', 'i18n', 'en.json');

const html = fs.readFileSync(HTML_PATH, 'utf8');
const de   = JSON.parse(fs.readFileSync(DE_PATH, 'utf8'));
const en   = JSON.parse(fs.readFileSync(EN_PATH, 'utf8'));

function resolveI18n(json, key) {
  return key.split('.').reduce(
    (o, k) => (o && typeof o === 'object') ? o[k] : undefined,
    json,
  );
}

describe('admin-bookings.html — dog filter label (bug 1)', () => {
  beforeAll(() => {
    document.documentElement.innerHTML = html;
  });

  test('#filter-dog-id select exists in the DOM', () => {
    expect(document.querySelector('#filter-dog-id')).not.toBeNull();
  });

  test('its label uses a data-i18n key resolving to "Hund" in de.json', () => {
    const select = document.querySelector('#filter-dog-id');
    const label  = select.closest('.form-group').querySelector('label');
    const key    = label.getAttribute('data-i18n');
    expect(key).toBeTruthy();
    expect(resolveI18n(de, key)).toBe('Hund');
  });

  test('its label uses a data-i18n key resolving to "Dog" in en.json', () => {
    const select = document.querySelector('#filter-dog-id');
    const label  = select.closest('.form-group').querySelector('label');
    const key    = label.getAttribute('data-i18n');
    expect(resolveI18n(en, key)).toBe('Dog');
  });

  test('"Alle Hunde" placeholder option uses dogs.all_dogs', () => {
    const opt = document.querySelector('#filter-dog-id option[value=""]');
    expect(opt.getAttribute('data-i18n')).toBe('dogs.all_dogs');
  });
});

/**
 * Extract a function body from the inline <script> block by name.
 * Returns the text between the opening brace and the matching closing brace.
 */
function extractFunctionBody(source, signatureRegex) {
  const match = source.match(signatureRegex);
  if (!match) return null;
  const start = match.index + match[0].length;
  let depth = 0;
  for (let i = start - 1; i < source.length; i++) {
    const c = source[i];
    if (c === '{') depth++;
    else if (c === '}') {
      depth--;
      if (depth === 0) return source.slice(start, i);
    }
  }
  return null;
}

describe('admin-bookings.html — loadBookings cache guard (bug 2)', () => {
  test('loadBookings guards api.getDog() behind a cache check', () => {
    const body = extractFunctionBody(html, /async\s+function\s+loadBookings\s*\(\s*\)\s*\{/);
    expect(body).not.toBeNull();

    // The body must contain a cache-hit short-circuit before any api.getDog(...) call
    // inside the Promise.all dogIds.map(...) loop. Concretely, we look for a branch
    // that bails out when dogs[id] already exists — patterns we accept:
    //   if (dogs[id]) return;
    //   if (!dogs[id]) { ... dogs[id] = await api.getDog(id); ... }
    const hasGuard =
      /if\s*\(\s*dogs\s*\[[^\]]+\]\s*\)/.test(body) ||
      /if\s*\(\s*!\s*dogs\s*\[[^\]]+\]/.test(body);

    expect(hasGuard).toBe(true);
  });
});
