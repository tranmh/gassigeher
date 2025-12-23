/**
 * Impersonation Banner Component Tests
 *
 * Tests for the banner that shows when super-admin is impersonating a user.
 *
 * @jest-environment jsdom
 */

// Mock window.api before loading
window.api = {
  getMe: jest.fn(),
  endImpersonation: jest.fn(),
  setToken: jest.fn(),
};

beforeAll(() => {
  document.body.innerHTML = '';
  loadSourceFile('internal/static/frontend/js/impersonation-banner.js');
});

beforeEach(() => {
  document.body.innerHTML = '';
  document.body.classList.remove('impersonating');
  jest.clearAllMocks();
});

describe('ImpersonationBanner.escapeHtml', () => {
  test('should escape < and > characters', () => {
    const result = window.ImpersonationBanner.escapeHtml('<script>alert("XSS")</script>');
    expect(result).not.toContain('<script>');
    expect(result).toContain('&lt;');
    expect(result).toContain('&gt;');
  });

  test('should escape ampersands', () => {
    const result = window.ImpersonationBanner.escapeHtml('Tom & Jerry');
    expect(result).toContain('&amp;');
  });

  test('should preserve normal text', () => {
    const result = window.ImpersonationBanner.escapeHtml('John Doe');
    expect(result).toBe('John Doe');
  });

  test('should handle empty string', () => {
    const result = window.ImpersonationBanner.escapeHtml('');
    expect(result).toBe('');
  });

  test('should handle German characters', () => {
    const result = window.ImpersonationBanner.escapeHtml('Müller Größe');
    expect(result).toBe('Müller Größe');
  });
});

describe('ImpersonationBanner.showBanner', () => {
  test('should create banner element', () => {
    window.ImpersonationBanner.showBanner('John Doe');

    const banner = document.getElementById('impersonation-banner');
    expect(banner).not.toBeNull();
  });

  test('should add impersonating class to body', () => {
    window.ImpersonationBanner.showBanner('John Doe');

    expect(document.body.classList.contains('impersonating')).toBe(true);
  });

  test('should display user name', () => {
    window.ImpersonationBanner.showBanner('John Doe');

    const banner = document.getElementById('impersonation-banner');
    expect(banner.innerHTML).toContain('John Doe');
  });

  test('should contain German text', () => {
    window.ImpersonationBanner.showBanner('Test User');

    const banner = document.getElementById('impersonation-banner');
    expect(banner.innerHTML).toContain('Impersonation aktiv');
    expect(banner.innerHTML).toContain('Sie sind als');
    expect(banner.innerHTML).toContain('angemeldet');
  });

  test('should contain back button', () => {
    window.ImpersonationBanner.showBanner('Test User');

    const banner = document.getElementById('impersonation-banner');
    const button = banner.querySelector('button');
    expect(button).not.toBeNull();
    expect(button.innerHTML).toContain('Zurück zum Admin');
  });

  test('should escape XSS in user name', () => {
    window.ImpersonationBanner.showBanner('<script>alert("XSS")</script>');

    const banner = document.getElementById('impersonation-banner');
    expect(banner.innerHTML).not.toContain('<script>');
    expect(banner.innerHTML).toContain('&lt;script&gt;');
  });

  test('should remove existing banner before creating new one', () => {
    window.ImpersonationBanner.showBanner('First User');
    window.ImpersonationBanner.showBanner('Second User');

    const banners = document.querySelectorAll('#impersonation-banner');
    expect(banners.length).toBe(1);

    const banner = document.getElementById('impersonation-banner');
    expect(banner.innerHTML).toContain('Second User');
  });

  test('should prepend banner to body', () => {
    const content = document.createElement('div');
    content.id = 'main-content';
    document.body.appendChild(content);

    window.ImpersonationBanner.showBanner('Test User');

    expect(document.body.firstElementChild.id).toBe('impersonation-banner');
  });
});

describe('ImpersonationBanner.init', () => {
  test('should call api.getMe', async () => {
    window.api.getMe.mockResolvedValue({ id: 1, is_impersonating: false });

    await window.ImpersonationBanner.init();

    expect(window.api.getMe).toHaveBeenCalled();
  });

  test('should show banner when impersonating', async () => {
    window.api.getMe.mockResolvedValue({
      id: 1,
      first_name: 'John',
      last_name: 'Doe',
      is_impersonating: true,
    });

    await window.ImpersonationBanner.init();

    const banner = document.getElementById('impersonation-banner');
    expect(banner).not.toBeNull();
    expect(banner.innerHTML).toContain('John Doe');
  });

  test('should not show banner when not impersonating', async () => {
    window.api.getMe.mockResolvedValue({
      id: 1,
      first_name: 'Admin',
      last_name: 'User',
      is_impersonating: false,
    });

    await window.ImpersonationBanner.init();

    const banner = document.getElementById('impersonation-banner');
    expect(banner).toBeNull();
  });

  test('should handle API error gracefully', async () => {
    window.api.getMe.mockRejectedValue(new Error('Not logged in'));
    const consoleSpy = jest.spyOn(console, 'debug').mockImplementation(() => {});

    await window.ImpersonationBanner.init();

    const banner = document.getElementById('impersonation-banner');
    expect(banner).toBeNull();

    consoleSpy.mockRestore();
  });

  test('should handle null response', async () => {
    window.api.getMe.mockResolvedValue(null);

    await window.ImpersonationBanner.init();

    const banner = document.getElementById('impersonation-banner');
    expect(banner).toBeNull();
  });
});

describe('ImpersonationBanner.endImpersonation', () => {
  beforeEach(() => {
    delete window.location;
    window.location = { href: '' };
  });

  test('should call api.endImpersonation', async () => {
    window.api.endImpersonation.mockResolvedValue({ token: 'admin-token' });

    await window.ImpersonationBanner.endImpersonation();

    expect(window.api.endImpersonation).toHaveBeenCalled();
  });

  test('should set new token on success', async () => {
    window.api.endImpersonation.mockResolvedValue({ token: 'admin-token' });

    await window.ImpersonationBanner.endImpersonation();

    expect(window.api.setToken).toHaveBeenCalledWith('admin-token');
  });

  test('should redirect to admin dashboard on success', async () => {
    window.api.endImpersonation.mockResolvedValue({ token: 'admin-token' });

    await window.ImpersonationBanner.endImpersonation();

    expect(window.location.href).toBe('/admin-dashboard.html');
  });

  test('should handle error gracefully', async () => {
    window.api.endImpersonation.mockRejectedValue(new Error('Failed'));
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
    const alertSpy = jest.spyOn(window, 'alert').mockImplementation(() => {});

    await window.ImpersonationBanner.endImpersonation();

    expect(consoleSpy).toHaveBeenCalled();
    expect(alertSpy).toHaveBeenCalled();

    consoleSpy.mockRestore();
    alertSpy.mockRestore();
  });

  test('should not redirect when no token returned', async () => {
    window.api.endImpersonation.mockResolvedValue({});

    await window.ImpersonationBanner.endImpersonation();

    expect(window.api.setToken).not.toHaveBeenCalled();
    expect(window.location.href).not.toBe('/admin-dashboard.html');
  });
});

describe('ImpersonationBanner - Global availability', () => {
  test('should be available on window', () => {
    expect(window.ImpersonationBanner).toBeDefined();
  });

  test('should have static methods', () => {
    expect(typeof window.ImpersonationBanner.init).toBe('function');
    expect(typeof window.ImpersonationBanner.showBanner).toBe('function');
    expect(typeof window.ImpersonationBanner.endImpersonation).toBe('function');
    expect(typeof window.ImpersonationBanner.escapeHtml).toBe('function');
  });
});

describe('ImpersonationBanner - XSS Prevention', () => {
  test('should escape script tags in user name', () => {
    window.ImpersonationBanner.showBanner('<script>document.cookie</script>');

    const banner = document.getElementById('impersonation-banner');
    expect(banner.innerHTML).not.toMatch(/<script>/);
    expect(banner.innerHTML).toContain('&lt;script&gt;');
  });

  test('should escape img onerror in user name', () => {
    window.ImpersonationBanner.showBanner('<img src=x onerror=alert("XSS")>');

    const banner = document.getElementById('impersonation-banner');
    expect(banner.innerHTML).not.toMatch(/<img /);
    expect(banner.innerHTML).toContain('&lt;img');
  });

  test('should escape event handlers in user name', () => {
    window.ImpersonationBanner.showBanner('"><img src=x onerror=alert(1)><"');

    const banner = document.getElementById('impersonation-banner');
    expect(banner.innerHTML).toContain('&lt;img');
    expect(banner.innerHTML).not.toMatch(/<img /);
  });
});

describe('ImpersonationBanner - Button functionality', () => {
  test('button should call endImpersonation', async () => {
    window.api.endImpersonation.mockResolvedValue({ token: 'admin-token' });

    window.ImpersonationBanner.showBanner('Test User');

    const banner = document.getElementById('impersonation-banner');
    const button = banner.querySelector('button');

    // The onclick is set as string attribute, test that it exists
    expect(button.getAttribute('onclick')).toContain('endImpersonation');
  });
});
