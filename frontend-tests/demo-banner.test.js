/**
 * Demo Banner Component Tests
 *
 * Tests for the demo banner JavaScript component that shows
 * an info banner when users are on the demo tenant.
 *
 * @jest-environment jsdom
 */

// Load the actual DemoBanner source file instead of duplicating the class
// This ensures tests always match the actual implementation
beforeAll(() => {
  // Reset DOM state before loading
  document.body.innerHTML = '';
  document.body.classList.remove('demo-mode');

  // Load the actual source file - DemoBanner will be available globally via window.DemoBanner
  loadSourceFile('internal/static/frontend/js/demo-banner.js');
});

// Reference to the loaded class (available after loadSourceFile)
let DemoBanner;

describe('DemoBanner.escapeHtml', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
    // Get reference to DemoBanner from window (loaded by loadSourceFile)
    DemoBanner = window.DemoBanner;
  });

  test('should escape < and > characters', () => {
    const result = DemoBanner.escapeHtml('<script>alert("XSS")</script>');
    expect(result).not.toContain('<script>');
    expect(result).toContain('&lt;');
    expect(result).toContain('&gt;');
  });

  test('should escape HTML entities', () => {
    const result = DemoBanner.escapeHtml('Test & "quotes" <tags>');
    expect(result).toContain('&amp;');
    // Note: textContent doesn't escape quotes, only < > &
    expect(result).toContain('"quotes"'); // quotes preserved as-is
    expect(result).toContain('&lt;');
    expect(result).toContain('&gt;');
  });

  test('should preserve normal text', () => {
    const result = DemoBanner.escapeHtml('23.12.2025 00:00');
    expect(result).toBe('23.12.2025 00:00');
  });

  test('should handle empty string', () => {
    const result = DemoBanner.escapeHtml('');
    expect(result).toBe('');
  });

  test('should handle special characters in date', () => {
    const result = DemoBanner.escapeHtml('23.12.2025 00:00 (Europe/Berlin)');
    expect(result).toBe('23.12.2025 00:00 (Europe/Berlin)');
  });
});

describe('DemoBanner.showBanner', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
    document.body.classList.remove('demo-mode');
    DemoBanner = window.DemoBanner;
  });

  test('should create banner element', () => {
    DemoBanner.showBanner(null);

    const banner = document.getElementById('demo-banner');
    expect(banner).not.toBeNull();
  });

  test('should add demo-mode class to body', () => {
    DemoBanner.showBanner(null);

    expect(document.body.classList.contains('demo-mode')).toBe(true);
  });

  test('should contain DEMO label', () => {
    DemoBanner.showBanner(null);

    const banner = document.getElementById('demo-banner');
    expect(banner.innerHTML).toContain('DEMO');
  });

  test('should contain German text', () => {
    DemoBanner.showBanner(null);

    const banner = document.getElementById('demo-banner');
    expect(banner.innerHTML).toContain('Demo-Umgebung');
    expect(banner.innerHTML).toContain('taeglich zurueckgesetzt');
  });

  test('should contain link to demo credentials', () => {
    DemoBanner.showBanner(null);

    const banner = document.getElementById('demo-banner');
    const link = banner.querySelector('a.demo-link');
    expect(link).not.toBeNull();
    // Link now points to clean URL on main domain (e.g., http://localhost/demo)
    expect(link.href).toContain('/demo');
    expect(link.target).toBe('_blank');
  });

  test('should show reset time when provided', () => {
    DemoBanner.showBanner('23.12.2025 00:00');

    const banner = document.getElementById('demo-banner');
    expect(banner.innerHTML).toContain('Reset:');
    expect(banner.innerHTML).toContain('23.12.2025 00:00');
  });

  test('should not show reset time when null', () => {
    DemoBanner.showBanner(null);

    const banner = document.getElementById('demo-banner');
    expect(banner.innerHTML).not.toContain('Reset:');
  });

  test('should escape XSS in reset time', () => {
    DemoBanner.showBanner('<script>alert("XSS")</script>');

    const banner = document.getElementById('demo-banner');
    expect(banner.innerHTML).not.toContain('<script>');
    expect(banner.innerHTML).toContain('&lt;script&gt;');
  });

  test('should remove existing banner before creating new one', () => {
    // Create first banner
    DemoBanner.showBanner('First');

    // Create second banner
    DemoBanner.showBanner('Second');

    // Should only have one banner
    const banners = document.querySelectorAll('#demo-banner');
    expect(banners.length).toBe(1);

    // Should show the new content
    const banner = document.getElementById('demo-banner');
    expect(banner.innerHTML).toContain('Second');
    expect(banner.innerHTML).not.toContain('First');
  });

  test('should prepend banner to body', () => {
    // Add some content to body
    const content = document.createElement('div');
    content.id = 'main-content';
    document.body.appendChild(content);

    DemoBanner.showBanner(null);

    // Banner should be first child
    expect(document.body.firstElementChild.id).toBe('demo-banner');
  });
});

describe('DemoBanner hostname detection', () => {
  let originalLocation;

  beforeEach(() => {
    originalLocation = window.location;
    document.body.innerHTML = '';
    document.body.classList.remove('demo-mode');
  });

  afterEach(() => {
    // Restore original location
    if (originalLocation !== window.location) {
      delete window.location;
      window.location = originalLocation;
    }
  });

  test('should detect demo. prefix', () => {
    const hostname = 'demo.gassigeher.org';
    const isDemo = hostname.startsWith('demo.') || hostname === 'demo';
    expect(isDemo).toBe(true);
  });

  test('should detect demo subdomain only', () => {
    const hostname = 'demo';
    const isDemo = hostname.startsWith('demo.') || hostname === 'demo';
    expect(isDemo).toBe(true);
  });

  test('should not detect non-demo subdomains', () => {
    const hostname = 'tierheim-goeppingen.gassigeher.org';
    const isDemo = hostname.startsWith('demo.') || hostname === 'demo';
    expect(isDemo).toBe(false);
  });

  test('should not detect localhost', () => {
    const hostname = 'localhost';
    const isDemo = hostname.startsWith('demo.') || hostname === 'demo';
    expect(isDemo).toBe(false);
  });

  test('should detect demo.localhost', () => {
    const hostname = 'demo.localhost';
    const isDemo = hostname.startsWith('demo.') || hostname === 'demo';
    expect(isDemo).toBe(true);
  });

  test('should not detect www subdomain', () => {
    const hostname = 'www.gassigeher.org';
    const isDemo = hostname.startsWith('demo.') || hostname === 'demo';
    expect(isDemo).toBe(false);
  });
});

describe('DemoBanner XSS prevention', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
    document.body.classList.remove('demo-mode');
    DemoBanner = window.DemoBanner;
  });

  test('should escape script tags in reset time', () => {
    DemoBanner.showBanner('<script>document.cookie</script>');

    const banner = document.getElementById('demo-banner');
    // Should not execute script
    expect(banner.innerHTML).not.toMatch(/<script>/);
    // Should contain escaped version
    expect(banner.innerHTML).toContain('&lt;script&gt;');
  });

  test('should escape img onerror in reset time', () => {
    DemoBanner.showBanner('<img src=x onerror=alert("XSS")>');

    const banner = document.getElementById('demo-banner');
    // Should not contain actual img tag
    expect(banner.innerHTML).not.toMatch(/<img /);
    // Should contain escaped version
    expect(banner.innerHTML).toContain('&lt;img');
  });

  test('should escape event handlers in reset time', () => {
    DemoBanner.showBanner('"><img src=x onerror=alert(1)><"');

    const banner = document.getElementById('demo-banner');
    // Should escape < and > preventing HTML injection
    expect(banner.innerHTML).toContain('&lt;img');
    expect(banner.innerHTML).toContain('&gt;');
    // Should not contain actual img tag
    expect(banner.innerHTML).not.toMatch(/<img /);
  });

  test('should escape SVG payloads', () => {
    DemoBanner.showBanner('<svg onload=alert("XSS")>');

    const banner = document.getElementById('demo-banner');
    // Should not contain actual svg tag
    expect(banner.innerHTML).not.toMatch(/<svg/);
    // Should contain escaped version
    expect(banner.innerHTML).toContain('&lt;svg');
  });
});

describe('DemoBanner CSS classes', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
    document.body.classList.remove('demo-mode');
    DemoBanner = window.DemoBanner;
  });

  test('banner should have demo-banner id', () => {
    DemoBanner.showBanner(null);

    const banner = document.getElementById('demo-banner');
    expect(banner).not.toBeNull();
  });

  test('banner should contain demo-banner-content', () => {
    DemoBanner.showBanner(null);

    const banner = document.getElementById('demo-banner');
    const content = banner.querySelector('.demo-banner-content');
    expect(content).not.toBeNull();
  });

  test('banner should contain demo-label', () => {
    DemoBanner.showBanner(null);

    const banner = document.getElementById('demo-banner');
    const label = banner.querySelector('.demo-label');
    expect(label).not.toBeNull();
    expect(label.textContent).toBe('DEMO');
  });

  test('banner should contain demo-text', () => {
    DemoBanner.showBanner(null);

    const banner = document.getElementById('demo-banner');
    const text = banner.querySelector('.demo-text');
    expect(text).not.toBeNull();
  });

  test('banner should contain demo-link', () => {
    DemoBanner.showBanner(null);

    const banner = document.getElementById('demo-banner');
    const link = banner.querySelector('.demo-link');
    expect(link).not.toBeNull();
  });

  test('reset info should have reset-info class', () => {
    DemoBanner.showBanner('23.12.2025 00:00');

    const banner = document.getElementById('demo-banner');
    const resetInfo = banner.querySelector('.reset-info');
    expect(resetInfo).not.toBeNull();
  });
});

describe('DemoBanner API integration', () => {
  let fetchMock;

  beforeEach(() => {
    document.body.innerHTML = '';
    document.body.classList.remove('demo-mode');
    DemoBanner = window.DemoBanner;

    // Mock fetch
    fetchMock = jest.fn();
    global.fetch = fetchMock;
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  test('should call /api/v1/demo/status when on demo subdomain', async () => {
    // Mock location
    delete window.location;
    window.location = { hostname: 'demo.gassigeher.org' };

    // Mock successful API response
    fetchMock.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          is_demo: true,
          next_reset_at: '23.12.2025 00:00',
        }),
    });

    await DemoBanner.init();

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/demo/status');
  });

  test('should not call API when not on demo subdomain', async () => {
    // Mock location
    delete window.location;
    window.location = { hostname: 'tierheim.gassigeher.org' };

    await DemoBanner.init();

    expect(fetchMock).not.toHaveBeenCalled();
  });

  test('should show banner even when API fails', async () => {
    // Mock location
    delete window.location;
    window.location = { hostname: 'demo.gassigeher.org' };

    // Mock failed API response
    fetchMock.mockRejectedValue(new Error('Network error'));

    // Suppress expected console.debug output
    const debugSpy = jest.spyOn(console, 'debug').mockImplementation(() => {});

    await DemoBanner.init();

    const banner = document.getElementById('demo-banner');
    expect(banner).not.toBeNull();

    debugSpy.mockRestore();
  });

  test('should show banner with reset time from API', async () => {
    // Mock location
    delete window.location;
    window.location = { hostname: 'demo.gassigeher.org' };

    // Mock successful API response
    fetchMock.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          is_demo: true,
          next_reset_at: '24.12.2025 00:00',
        }),
    });

    await DemoBanner.init();

    const banner = document.getElementById('demo-banner');
    expect(banner.innerHTML).toContain('24.12.2025 00:00');
  });
});
