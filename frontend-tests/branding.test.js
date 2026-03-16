/**
 * Branding Module Tests
 *
 * Tests for the shared branding.js script that fetches tenant branding
 * and replaces hardcoded "Gassigeher" with the actual tenant name
 * across page titles, logo text, and footer copyright.
 *
 * @jest-environment jsdom
 */

// Mock fetch globally before loading branding.js
let mockBrandingResponse;

beforeEach(() => {
  // Reset DOM
  document.body.innerHTML = '';
  document.title = '';

  // Reset globals
  delete window.branding;
  delete window.loadBrandingData;
  delete window.invalidateBrandingCache;

  // Clear sessionStorage
  sessionStorage.clear();

  // Default mock branding response
  mockBrandingResponse = {
    tenant_name: 'Tierheim Göppingen',
    tenant_slug: 'tierheim-goeppingen',
    theme_preset: 'classic',
    welcome_message: null,
    organization_name: 'Tierschutzverein Göppingen e.V.',
    organization_email: 'info@tierheim-goeppingen.de',
    organization_address: 'Tierheimstr. 1, 73033 Göppingen',
    privacy_officer_email: 'datenschutz@tierheim-goeppingen.de',
  };

  // Mock fetch
  global.fetch = jest.fn(() =>
    Promise.resolve({
      ok: true,
      json: () => Promise.resolve(mockBrandingResponse),
    })
  );
});

afterEach(() => {
  jest.restoreAllMocks();
});

/**
 * Helper: Load branding.js fresh (re-executes the IIFE)
 * Since branding.js auto-executes on DOMContentLoaded, we need to
 * trigger it manually in tests.
 */
function loadBrandingScript() {
  // Set readyState so the IIFE calls loadBranding immediately
  Object.defineProperty(document, 'readyState', {
    value: 'complete',
    writable: true,
    configurable: true,
  });
  loadSourceFile('internal/static/frontend/js/branding.js');
}

describe('branding.js - Global API', () => {
  test('exposes window.loadBrandingData function', () => {
    loadBrandingScript();
    expect(typeof window.loadBrandingData).toBe('function');
  });

  test('exposes window.invalidateBrandingCache function', () => {
    loadBrandingScript();
    expect(typeof window.invalidateBrandingCache).toBe('function');
  });

  test('loadBrandingData fetches from /api/v1/tenant/branding', async () => {
    loadBrandingScript();
    await window.loadBrandingData();

    expect(global.fetch).toHaveBeenCalledWith('/api/v1/tenant/branding');
  });

  test('loadBrandingData returns branding data', async () => {
    loadBrandingScript();
    const result = await window.loadBrandingData();

    expect(result).toBeTruthy();
    expect(result.tenant_name).toBe('Tierheim Göppingen');
  });

  test('sets window.branding after loading', async () => {
    loadBrandingScript();
    await window.loadBrandingData();

    expect(window.branding).toBeTruthy();
    expect(window.branding.tenant_name).toBe('Tierheim Göppingen');
  });
});

describe('branding.js - Title Replacement', () => {
  test('replaces "Gassigeher" in page title with tenant name', async () => {
    document.title = 'Dashboard - Gassigeher';
    loadBrandingScript();
    await window.loadBrandingData();

    expect(document.title).toBe('Dashboard - Tierheim Göppingen');
  });

  test('replaces "Gassigeher Admin" in page title', async () => {
    document.title = 'Branding - Gassigeher Admin';
    loadBrandingScript();
    await window.loadBrandingData();

    expect(document.title).toBe('Branding - Tierheim Göppingen Admin');
  });

  test('does not modify title when no "Gassigeher" present', async () => {
    document.title = 'Some Other Page';
    loadBrandingScript();
    await window.loadBrandingData();

    expect(document.title).toBe('Some Other Page');
  });
});

describe('branding.js - Logo Replacement', () => {
  test('replaces "Gassigeher" in logo text', async () => {
    document.body.innerHTML = '<a href="/" class="logo">🐕 Gassigeher</a>';
    loadBrandingScript();
    await window.loadBrandingData();

    const logo = document.querySelector('a.logo');
    expect(logo.textContent).toBe('🐕 Tierheim Göppingen');
  });

  test('replaces "Gassigeher Admin" in admin logo', async () => {
    document.body.innerHTML = '<a href="/" class="logo">🐕 Gassigeher Admin</a>';
    loadBrandingScript();
    await window.loadBrandingData();

    const logo = document.querySelector('a.logo');
    expect(logo.textContent).toBe('🐕 Tierheim Göppingen Admin');
  });

  test('skips logo with id="site-logo" (index.html manages its own)', async () => {
    document.body.innerHTML = '<a href="/" class="logo" id="site-logo">🐕 Gassigeher</a>';
    loadBrandingScript();
    await window.loadBrandingData();

    const logo = document.querySelector('#site-logo');
    expect(logo.textContent).toBe('🐕 Gassigeher'); // unchanged
  });
});

describe('branding.js - Footer Replacement', () => {
  test('replaces "Gassigeher" in footer copyright', async () => {
    document.body.innerHTML = `
      <footer>
        <p>&copy; 2025 Gassigeher. Alle Rechte vorbehalten.</p>
      </footer>
    `;
    loadBrandingScript();
    await window.loadBrandingData();

    const footerP = document.querySelector('footer p');
    expect(footerP.innerHTML).toContain('Tierheim Göppingen');
    expect(footerP.innerHTML).not.toContain('>Gassigeher<');
  });

  test('skips elements with data-no-branding attribute', async () => {
    document.body.innerHTML = `
      <footer>
        <p data-no-branding>Powered by Gassigeher</p>
      </footer>
    `;
    loadBrandingScript();
    await window.loadBrandingData();

    const footerP = document.querySelector('footer p');
    expect(footerP.textContent).toBe('Powered by Gassigeher'); // unchanged
  });

  test('skips footer-copyright element (index.html manages its own)', async () => {
    document.body.innerHTML = `
      <footer>
        <p id="footer-copyright">&copy; 2025 Gassigeher</p>
      </footer>
    `;
    loadBrandingScript();
    await window.loadBrandingData();

    const footerP = document.querySelector('#footer-copyright');
    expect(footerP.innerHTML).toContain('Gassigeher'); // unchanged
  });
});

describe('branding.js - SessionStorage Caching', () => {
  test('caches branding data in sessionStorage', async () => {
    loadBrandingScript();
    await window.loadBrandingData();

    const cached = sessionStorage.getItem('gassigeher_branding');
    expect(cached).toBeTruthy();

    const parsed = JSON.parse(cached);
    expect(parsed.data.tenant_name).toBe('Tierheim Göppingen');
    expect(parsed.timestamp).toBeTruthy();
  });

  test('uses cached data on second call (no extra fetch)', async () => {
    loadBrandingScript();
    // IIFE auto-fetches once on load, then loadBrandingData fetches again
    await window.loadBrandingData();
    const callsAfterFirstLoad = global.fetch.mock.calls.length;

    // Second call should use cache - no additional fetch
    await window.loadBrandingData();
    expect(global.fetch).toHaveBeenCalledTimes(callsAfterFirstLoad);
  });

  test('invalidateBrandingCache clears sessionStorage', async () => {
    loadBrandingScript();
    await window.loadBrandingData();

    expect(sessionStorage.getItem('gassigeher_branding')).toBeTruthy();

    window.invalidateBrandingCache();

    expect(sessionStorage.getItem('gassigeher_branding')).toBeNull();
  });

  test('fetches again after cache invalidation', async () => {
    loadBrandingScript();
    await window.loadBrandingData();
    const callsBeforeInvalidation = global.fetch.mock.calls.length;

    window.invalidateBrandingCache();
    await window.loadBrandingData();
    expect(global.fetch).toHaveBeenCalledTimes(callsBeforeInvalidation + 1);
  });
});

describe('branding.js - Error Handling', () => {
  let warnSpy;

  beforeEach(() => {
    warnSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
  });

  afterEach(() => {
    warnSpy.mockRestore();
  });

  test('returns null when API returns error', async () => {
    global.fetch = jest.fn(() =>
      Promise.resolve({ ok: false, status: 500 })
    );

    loadBrandingScript();
    const result = await window.loadBrandingData();
    expect(result).toBeNull();
  });

  test('returns null when fetch throws', async () => {
    global.fetch = jest.fn(() => Promise.reject(new Error('Network error')));

    loadBrandingScript();
    const result = await window.loadBrandingData();
    expect(result).toBeNull();
  });

  test('does not modify DOM when tenant_name is missing', async () => {
    mockBrandingResponse = { theme_preset: 'classic' };
    global.fetch = jest.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockBrandingResponse),
      })
    );

    document.title = 'Dashboard - Gassigeher';
    document.body.innerHTML = '<a href="/" class="logo">🐕 Gassigeher</a>';

    loadBrandingScript();
    await window.loadBrandingData();

    expect(document.title).toBe('Dashboard - Gassigeher'); // unchanged
    expect(document.querySelector('a.logo').textContent).toBe('🐕 Gassigeher'); // unchanged
  });
});

describe('branding.js - Organization Fields', () => {
  test('exposes organization fields via window.branding', async () => {
    loadBrandingScript();
    await window.loadBrandingData();

    expect(window.branding.organization_name).toBe('Tierschutzverein Göppingen e.V.');
    expect(window.branding.organization_email).toBe('info@tierheim-goeppingen.de');
    expect(window.branding.organization_address).toBe('Tierheimstr. 1, 73033 Göppingen');
    expect(window.branding.privacy_officer_email).toBe('datenschutz@tierheim-goeppingen.de');
  });

  test('handles missing organization fields gracefully', async () => {
    mockBrandingResponse = {
      tenant_name: 'Mein Tierheim',
      theme_preset: 'classic',
    };
    global.fetch = jest.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockBrandingResponse),
      })
    );

    loadBrandingScript();
    await window.loadBrandingData();

    expect(window.branding.tenant_name).toBe('Mein Tierheim');
    expect(window.branding.organization_name).toBeUndefined();
    expect(window.branding.organization_email).toBeUndefined();
  });
});

describe('branding.js - XSS Safety', () => {
  test('escapes tenant_name in footer innerHTML replacement', async () => {
    mockBrandingResponse = {
      tenant_name: '<script>alert("xss")</script>',
      theme_preset: 'classic',
    };
    global.fetch = jest.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockBrandingResponse),
      })
    );

    document.body.innerHTML = `
      <footer>
        <p>&copy; 2025 Gassigeher. Alle Rechte vorbehalten.</p>
      </footer>
    `;

    loadBrandingScript();
    await window.loadBrandingData();

    const footerP = document.querySelector('footer p');
    // Should be escaped, not executed
    expect(footerP.innerHTML).not.toContain('<script>');
    expect(footerP.innerHTML).toContain('&lt;script&gt;');
  });
});
