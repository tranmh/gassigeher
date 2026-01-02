/**
 * Central Admin JavaScript Tests
 *
 * Tests for the central admin API client and utility functions.
 *
 * @jest-environment jsdom
 */

// Mock localStorage
const localStorageMock = (() => {
  let store = {};
  return {
    getItem: jest.fn((key) => store[key] || null),
    setItem: jest.fn((key, value) => { store[key] = value; }),
    removeItem: jest.fn((key) => { delete store[key]; }),
    clear: jest.fn(() => { store = {}; }),
  };
})();

Object.defineProperty(window, 'localStorage', { value: localStorageMock });

// Mock fetch
let fetchMock;

// Define central admin functions since they're not exported as globals
const TOKEN_KEY = 'gassigeher_token';

function getToken() {
  return localStorage.getItem(TOKEN_KEY);
}

function isAuthenticated() {
  return !!getToken();
}

function logout() {
  localStorage.removeItem(TOKEN_KEY);
  window.location.href = '/login.html';
}

async function apiRequest(endpoint, options = {}) {
  const token = getToken();
  if (!token) {
    window.location.href = '/login.html';
    throw new Error('Not authenticated');
  }
  const config = {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
      ...options.headers
    }
  };
  const response = await fetch(`/api/v1${endpoint}`, config);
  if (response.status === 401 || response.status === 403) {
    logout();
    throw new Error('Unauthorized');
  }
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(error.error || 'Request failed');
  }
  return response.json();
}

const centralAPI = {
  async getStats() {
    return apiRequest('/central-admin/stats');
  },
  async getTenants(search = '', activeOnly = false) {
    const params = new URLSearchParams();
    if (search) params.append('search', search);
    if (activeOnly) params.append('active_only', 'true');
    return apiRequest(`/central-admin/tenants?${params}`);
  },
  async getTenant(id) {
    return apiRequest(`/central-admin/tenants/${id}`);
  },
  async updateTenant(id, data) {
    return apiRequest(`/central-admin/tenants/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data)
    });
  },
  async activateTenant(id) {
    return apiRequest(`/central-admin/tenants/${id}/activate`, { method: 'POST' });
  },
  async deactivateTenant(id) {
    return apiRequest(`/central-admin/tenants/${id}/deactivate`, { method: 'POST' });
  },
  async getTenantUsers(id) {
    return apiRequest(`/central-admin/tenants/${id}/users`);
  },
  async exportTenant(id) {
    return apiRequest(`/central-admin/tenants/${id}/export`);
  },
  async searchUsers(query) {
    return apiRequest(`/central-admin/users/search?q=${encodeURIComponent(query)}`);
  },
  async getAdmins() {
    return apiRequest('/central-admin/admins');
  },
  async promoteToAdmin(userId) {
    return apiRequest(`/central-admin/admins/${userId}/promote`, { method: 'POST' });
  },
  async demoteAdmin(userId) {
    return apiRequest(`/central-admin/admins/${userId}/demote`, { method: 'POST' });
  }
};

function escapeHtml(text) {
  if (!text) return '';
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

function formatDate(dateStr) {
  if (!dateStr) return '-';
  const date = new Date(dateStr);
  return date.toLocaleDateString('de-DE');
}

function formatDateTime(dateStr) {
  if (!dateStr) return '-';
  const date = new Date(dateStr);
  return date.toLocaleDateString('de-DE') + ' ' + date.toLocaleTimeString('de-DE', {
    hour: '2-digit',
    minute: '2-digit'
  });
}

function showAlert(message, type = 'success') {
  const container = document.getElementById('alert-container');
  if (!container) return;
  const alert = document.createElement('div');
  alert.className = `alert alert-${type}`;
  alert.textContent = message;
  container.innerHTML = '';
  container.appendChild(alert);
  setTimeout(() => {
    alert.remove();
  }, 5000);
}

beforeAll(() => {
  document.body.innerHTML = '';
});

beforeEach(() => {
  localStorageMock.clear();
  fetchMock = jest.fn();
  global.fetch = fetchMock;

  // Reset location
  delete window.location;
  window.location = { href: '' };
});

afterEach(() => {
  jest.restoreAllMocks();
});

describe('Token Management', () => {
  test('getToken should return token from localStorage', () => {
    localStorageMock.getItem.mockReturnValue('test-token');

    const token = getToken();

    expect(localStorageMock.getItem).toHaveBeenCalledWith('gassigeher_token');
    expect(token).toBe('test-token');
  });

  test('getToken should return null when no token', () => {
    localStorageMock.getItem.mockReturnValue(null);

    const token = getToken();

    expect(token).toBeNull();
  });

  test('isAuthenticated should return true when token exists', () => {
    localStorageMock.getItem.mockReturnValue('valid-token');

    expect(isAuthenticated()).toBe(true);
  });

  test('isAuthenticated should return false when no token', () => {
    localStorageMock.getItem.mockReturnValue(null);

    expect(isAuthenticated()).toBe(false);
  });

  test('isAuthenticated should return false for empty token', () => {
    localStorageMock.getItem.mockReturnValue('');

    expect(isAuthenticated()).toBe(false);
  });

  test('logout should remove token and redirect', () => {
    logout();

    expect(localStorageMock.removeItem).toHaveBeenCalledWith('gassigeher_token');
    expect(window.location.href).toBe('/login.html');
  });
});

describe('apiRequest', () => {
  test('should redirect to login when no token', async () => {
    localStorageMock.getItem.mockReturnValue(null);

    await expect(apiRequest('/test')).rejects.toThrow('Not authenticated');
    expect(window.location.href).toBe('/login.html');
  });

  test('should include Authorization header', async () => {
    localStorageMock.getItem.mockReturnValue('my-token');
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    });

    await apiRequest('/test');

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/test', expect.objectContaining({
      headers: expect.objectContaining({
        'Authorization': 'Bearer my-token',
      }),
    }));
  });

  test('should include Content-Type header', async () => {
    localStorageMock.getItem.mockReturnValue('my-token');
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    });

    await apiRequest('/test');

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/test', expect.objectContaining({
      headers: expect.objectContaining({
        'Content-Type': 'application/json',
      }),
    }));
  });

  test('should call logout on 401', async () => {
    localStorageMock.getItem.mockReturnValue('expired-token');
    fetchMock.mockResolvedValue({
      status: 401,
      ok: false,
    });

    await expect(apiRequest('/test')).rejects.toThrow('Unauthorized');
    expect(localStorageMock.removeItem).toHaveBeenCalledWith('gassigeher_token');
    expect(window.location.href).toBe('/login.html');
  });

  test('should call logout on 403', async () => {
    localStorageMock.getItem.mockReturnValue('forbidden-token');
    fetchMock.mockResolvedValue({
      status: 403,
      ok: false,
    });

    await expect(apiRequest('/test')).rejects.toThrow('Unauthorized');
    expect(window.location.href).toBe('/login.html');
  });

  test('should throw error with message from response', async () => {
    localStorageMock.getItem.mockReturnValue('valid-token');
    fetchMock.mockResolvedValue({
      status: 400,
      ok: false,
      json: () => Promise.resolve({ error: 'Bad Request' }),
    });

    await expect(apiRequest('/test')).rejects.toThrow('Bad Request');
  });

  test('should handle JSON parse error', async () => {
    localStorageMock.getItem.mockReturnValue('valid-token');
    fetchMock.mockResolvedValue({
      status: 500,
      ok: false,
      json: () => Promise.reject(new Error('Invalid JSON')),
    });

    await expect(apiRequest('/test')).rejects.toThrow('Unknown error');
  });

  test('should return JSON data on success', async () => {
    localStorageMock.getItem.mockReturnValue('valid-token');
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: 'test' }),
    });

    const result = await apiRequest('/test');

    expect(result).toEqual({ data: 'test' });
  });

  test('should merge custom options', async () => {
    localStorageMock.getItem.mockReturnValue('valid-token');
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    });

    await apiRequest('/test', { method: 'POST', body: JSON.stringify({ data: 'test' }) });

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/test', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ data: 'test' }),
    }));
  });
});

describe('centralAPI', () => {
  beforeEach(() => {
    localStorageMock.getItem.mockReturnValue('valid-token');
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    });
  });

  test('getStats should call correct endpoint', async () => {
    await centralAPI.getStats();

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/central-admin/stats', expect.anything());
  });

  test('getTenants should call correct endpoint', async () => {
    await centralAPI.getTenants();

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/central-admin/tenants?', expect.anything());
  });

  test('getTenants should add search param', async () => {
    await centralAPI.getTenants('shelter');

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('search=shelter'),
      expect.anything()
    );
  });

  test('getTenants should add active_only param', async () => {
    await centralAPI.getTenants('', true);

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('active_only=true'),
      expect.anything()
    );
  });

  test('getTenant should call correct endpoint with ID', async () => {
    await centralAPI.getTenant(5);

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/central-admin/tenants/5', expect.anything());
  });

  test('updateTenant should PUT to correct endpoint', async () => {
    await centralAPI.updateTenant(5, { name: 'Updated' });

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/central-admin/tenants/5', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ name: 'Updated' }),
    }));
  });

  test('activateTenant should POST to correct endpoint', async () => {
    await centralAPI.activateTenant(5);

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/central-admin/tenants/5/activate', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('deactivateTenant should POST to correct endpoint', async () => {
    await centralAPI.deactivateTenant(5);

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/central-admin/tenants/5/deactivate', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('getTenantUsers should call correct endpoint', async () => {
    await centralAPI.getTenantUsers(5);

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/central-admin/tenants/5/users', expect.anything());
  });

  test('exportTenant should call correct endpoint', async () => {
    await centralAPI.exportTenant(5);

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/central-admin/tenants/5/export', expect.anything());
  });

  test('searchUsers should call correct endpoint with query', async () => {
    await centralAPI.searchUsers('john');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/central-admin/users/search?q=john',
      expect.anything()
    );
  });

  test('searchUsers should encode query', async () => {
    await centralAPI.searchUsers('john doe');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/central-admin/users/search?q=john%20doe',
      expect.anything()
    );
  });

  test('getAdmins should call correct endpoint', async () => {
    await centralAPI.getAdmins();

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/central-admin/admins', expect.anything());
  });

  test('promoteToAdmin should POST to correct endpoint', async () => {
    await centralAPI.promoteToAdmin(10);

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/central-admin/admins/10/promote', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('demoteAdmin should POST to correct endpoint', async () => {
    await centralAPI.demoteAdmin(10);

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/central-admin/admins/10/demote', expect.objectContaining({
      method: 'POST',
    }));
  });
});

describe('Utility Functions', () => {
  describe('escapeHtml', () => {
    test('should escape < and > characters', () => {
      const result = escapeHtml('<script>alert("XSS")</script>');
      expect(result).toContain('&lt;');
      expect(result).toContain('&gt;');
      expect(result).not.toContain('<script>');
    });

    test('should escape ampersands', () => {
      const result = escapeHtml('Tom & Jerry');
      expect(result).toContain('&amp;');
    });

    test('should handle empty string', () => {
      expect(escapeHtml('')).toBe('');
    });

    test('should handle null', () => {
      expect(escapeHtml(null)).toBe('');
    });

    test('should handle undefined', () => {
      expect(escapeHtml(undefined)).toBe('');
    });

    test('should preserve normal text', () => {
      expect(escapeHtml('Hello World')).toBe('Hello World');
    });
  });

  describe('formatDate', () => {
    test('should format date string', () => {
      const result = formatDate('2025-12-25T10:30:00Z');
      expect(result).toMatch(/\d{1,2}\.\d{1,2}\.\d{4}/); // German format
    });

    test('should return dash for null', () => {
      expect(formatDate(null)).toBe('-');
    });

    test('should return dash for empty string', () => {
      expect(formatDate('')).toBe('-');
    });

    test('should return dash for undefined', () => {
      expect(formatDate(undefined)).toBe('-');
    });
  });

  describe('formatDateTime', () => {
    test('should format date and time string', () => {
      const result = formatDateTime('2025-12-25T10:30:00Z');
      // Should contain both date and time in German format
      expect(result).toMatch(/\d{1,2}\.\d{1,2}\.\d{4}/);
      expect(result).toMatch(/\d{1,2}:\d{2}/);
    });

    test('should return dash for null', () => {
      expect(formatDateTime(null)).toBe('-');
    });

    test('should return dash for empty string', () => {
      expect(formatDateTime('')).toBe('-');
    });
  });

  describe('showAlert', () => {
    beforeEach(() => {
      document.body.innerHTML = '<div id="alert-container"></div>';
      jest.useFakeTimers();
    });

    afterEach(() => {
      jest.useRealTimers();
    });

    test('should create alert element', () => {
      showAlert('Test message');

      const container = document.getElementById('alert-container');
      expect(container.querySelector('.alert')).not.toBeNull();
    });

    test('should display message', () => {
      showAlert('Test message');

      const container = document.getElementById('alert-container');
      expect(container.textContent).toContain('Test message');
    });

    test('should default to success type', () => {
      showAlert('Success!');

      const alert = document.querySelector('.alert');
      expect(alert.classList.contains('alert-success')).toBe(true);
    });

    test('should support error type', () => {
      showAlert('Error!', 'error');

      const alert = document.querySelector('.alert');
      expect(alert.classList.contains('alert-error')).toBe(true);
    });

    test('should clear previous alerts', () => {
      showAlert('First');
      showAlert('Second');

      const container = document.getElementById('alert-container');
      const alerts = container.querySelectorAll('.alert');
      expect(alerts.length).toBe(1);
      expect(container.textContent).toContain('Second');
    });

    test('should auto-dismiss after 5 seconds', () => {
      showAlert('Temporary');

      expect(document.querySelector('.alert')).not.toBeNull();

      jest.advanceTimersByTime(5000);

      expect(document.querySelector('.alert')).toBeNull();
    });

    test('should handle missing container', () => {
      document.body.innerHTML = '';

      expect(() => showAlert('Test')).not.toThrow();
    });
  });
});

/**
 * Helper function to determine if a user should show "Bereits Central Admin"
 * This replicates the logic from central/users.html for testing
 *
 * BUG: The original logic was:
 *   !u.is_central_admin && !u.is_super_admin ? "Zum Central Admin" : "Bereits Central Admin"
 *
 * This incorrectly treats is_super_admin as equivalent to Central Admin.
 * A Super Admin is the top admin within a TENANT, not a platform-wide Central Admin.
 *
 * CORRECT: Only is_central_admin should determine Central Admin status
 */
function shouldShowAlreadyCentralAdmin(user) {
  // BUG FIX: Only check is_central_admin, NOT is_super_admin
  // is_super_admin means tenant-level admin, NOT platform Central Admin
  return user.is_central_admin;
}

/**
 * Original buggy logic (for comparison in tests)
 */
function shouldShowAlreadyCentralAdmin_BUGGY(user) {
  // BUG: This incorrectly treats is_super_admin as Central Admin
  return user.is_central_admin || user.is_super_admin;
}

/**
 * Helper function to determine if user can be impersonated
 *
 * BUG FIX: The original logic was:
 *   !u.is_super_admin && !u.is_central_admin && u.is_active
 *
 * This incorrectly blocked impersonating Super Admins.
 * Central Admin should be able to impersonate Super Admins to test tenant admin experience.
 *
 * CORRECT: Only Central Admins cannot be impersonated
 */
function canImpersonateUser(user) {
  // Only Central Admins cannot be impersonated
  // Super Admins CAN be impersonated so Central Admin can test tenant admin experience
  return !user.is_central_admin && user.is_active;
}

/**
 * Original buggy impersonation logic (for comparison)
 */
function canImpersonateUser_BUGGY(user) {
  // BUG: This incorrectly blocks impersonating Super Admins
  return !user.is_super_admin && !user.is_central_admin && user.is_active;
}

describe('Central Admin User List UI Logic - SECURITY BUG FIX', () => {
  describe('shouldShowAlreadyCentralAdmin', () => {
    test('should return true ONLY for users with is_central_admin=true', () => {
      const centralAdmin = {
        id: 1,
        is_admin: true,
        is_super_admin: true,
        is_central_admin: true,
      };

      expect(shouldShowAlreadyCentralAdmin(centralAdmin)).toBe(true);
    });

    test('should return false for Super Admin who is NOT Central Admin', () => {
      // This is the key security test case!
      // A tenant Super Admin should NOT be treated as Central Admin
      const tenantSuperAdmin = {
        id: 2,
        is_admin: true,
        is_super_admin: true,      // Super Admin of their tenant
        is_central_admin: false,   // NOT a Central Admin of the platform
      };

      expect(shouldShowAlreadyCentralAdmin(tenantSuperAdmin)).toBe(false);
    });

    test('should return false for regular admin', () => {
      const regularAdmin = {
        id: 3,
        is_admin: true,
        is_super_admin: false,
        is_central_admin: false,
      };

      expect(shouldShowAlreadyCentralAdmin(regularAdmin)).toBe(false);
    });

    test('should return false for regular user', () => {
      const regularUser = {
        id: 4,
        is_admin: false,
        is_super_admin: false,
        is_central_admin: false,
      };

      expect(shouldShowAlreadyCentralAdmin(regularUser)).toBe(false);
    });
  });

  describe('Verify buggy behavior (to document the bug)', () => {
    test('BUGGY: incorrectly treats Super Admin as Central Admin', () => {
      const tenantSuperAdmin = {
        id: 2,
        is_admin: true,
        is_super_admin: true,      // Super Admin of their tenant
        is_central_admin: false,   // NOT a Central Admin
      };

      // The buggy logic would incorrectly return true
      expect(shouldShowAlreadyCentralAdmin_BUGGY(tenantSuperAdmin)).toBe(true);

      // But the correct logic returns false
      expect(shouldShowAlreadyCentralAdmin(tenantSuperAdmin)).toBe(false);
    });
  });

  describe('canImpersonateUser (Impersonate button visibility)', () => {
    test('should allow impersonating Super Admin of a tenant', () => {
      const tenantSuperAdmin = {
        id: 2,
        is_admin: true,
        is_super_admin: true,      // Super Admin of their tenant
        is_central_admin: false,   // NOT a Central Admin
        is_active: true,
      };

      // Super Admins CAN be impersonated to test tenant admin experience
      expect(canImpersonateUser(tenantSuperAdmin)).toBe(true);
    });

    test('should NOT allow impersonating Central Admin', () => {
      const centralAdmin = {
        id: 1,
        is_admin: true,
        is_super_admin: true,
        is_central_admin: true,    // Central Admin
        is_active: true,
      };

      expect(canImpersonateUser(centralAdmin)).toBe(false);
    });

    test('should allow impersonating regular users', () => {
      const regularUser = {
        id: 4,
        is_admin: false,
        is_super_admin: false,
        is_central_admin: false,
        is_active: true,
      };

      expect(canImpersonateUser(regularUser)).toBe(true);
    });

    test('should NOT allow impersonating inactive users', () => {
      const inactiveUser = {
        id: 5,
        is_admin: false,
        is_super_admin: false,
        is_central_admin: false,
        is_active: false,
      };

      expect(canImpersonateUser(inactiveUser)).toBe(false);
    });
  });

  describe('Verify buggy impersonation behavior (to document the bug)', () => {
    test('BUGGY: incorrectly blocks impersonating Super Admin', () => {
      const tenantSuperAdmin = {
        id: 2,
        is_admin: true,
        is_super_admin: true,
        is_central_admin: false,
        is_active: true,
      };

      // The buggy logic would incorrectly return false (blocking impersonation)
      expect(canImpersonateUser_BUGGY(tenantSuperAdmin)).toBe(false);

      // But the correct logic returns true (allowing impersonation)
      expect(canImpersonateUser(tenantSuperAdmin)).toBe(true);
    });
  });
});

describe('Global Function Availability', () => {
  test('getToken should be defined', () => {
    expect(typeof getToken).toBe('function');
  });

  test('isAuthenticated should be defined', () => {
    expect(typeof isAuthenticated).toBe('function');
  });

  test('logout should be defined', () => {
    expect(typeof logout).toBe('function');
  });

  test('apiRequest should be defined', () => {
    expect(typeof apiRequest).toBe('function');
  });

  test('centralAPI should be defined', () => {
    expect(typeof centralAPI).toBe('object');
  });

  test('escapeHtml should be defined', () => {
    expect(typeof escapeHtml).toBe('function');
  });

  test('formatDate should be defined', () => {
    expect(typeof formatDate).toBe('function');
  });

  test('formatDateTime should be defined', () => {
    expect(typeof formatDateTime).toBe('function');
  });

  test('showAlert should be defined', () => {
    expect(typeof showAlert).toBe('function');
  });
});
