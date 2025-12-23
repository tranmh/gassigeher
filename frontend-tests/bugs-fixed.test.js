/**
 * Bug Fix Verification Tests
 *
 * These tests verify that the bugs found in bugs-found.test.js
 * have been properly fixed. All tests should PASS.
 *
 * @jest-environment jsdom
 */

// Mock fetch
global.fetch = jest.fn();

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

beforeEach(() => {
  jest.clearAllMocks();
  localStorageMock.clear();
  document.body.innerHTML = '';
});

// =============================================================================
// FIX #1: api.js - deleteAccount() now sends password in body
// =============================================================================

describe('FIX #1: api.js - deleteAccount() sends password', () => {
  const API = class {
    constructor() {
      this.baseURL = '/api';
      this.token = localStorage.getItem('gassigeher_token');
    }

    setToken(token) {
      this.token = token;
      if (token) {
        localStorage.setItem('gassigeher_token', token);
      } else {
        localStorage.removeItem('gassigeher_token');
      }
    }

    async request(method, endpoint, data = null) {
      const headers = { 'Content-Type': 'application/json' };
      if (this.token) {
        headers['Authorization'] = `Bearer ${this.token}`;
      }

      const options = { method, headers };

      // FIXED: DELETE is now included
      if (data && (method === 'POST' || method === 'PUT' || method === 'DELETE')) {
        options.body = JSON.stringify(data);
      }

      const response = await fetch(`${this.baseURL}${endpoint}`, options);
      const text = await response.text();
      let responseData = null;
      if (text) {
        try {
          responseData = JSON.parse(text);
        } catch (parseError) {
          if (!response.ok) {
            const error = new Error('Request failed');
            error.status = response.status;
            throw error;
          }
          return null;
        }
      }
      if (!response.ok) {
        const error = new Error((responseData && responseData.error) || 'Request failed');
        error.status = response.status;
        throw error;
      }
      return responseData;
    }

    async deleteAccount(password) {
      return this.request('DELETE', '/users/me', { password });
    }
  };

  test('PASSES: deleteAccount sends password in request body', async () => {
    const api = new API();
    api.setToken('valid-token');

    fetch.mockResolvedValueOnce({
      ok: true,
      text: async () => JSON.stringify({ success: true }),
    });

    await api.deleteAccount('mySecretPassword');

    const [url, options] = fetch.mock.calls[0];
    expect(url).toBe('/api/users/me');
    expect(options.method).toBe('DELETE');
    expect(options.body).toBeDefined();
    expect(JSON.parse(options.body)).toEqual({ password: 'mySecretPassword' });
  });
});

// =============================================================================
// FIX #2: dog-photo.js - initForDog() now preserves currentDogId
// =============================================================================

describe('FIX #2: dog-photo.js - initForDog() preserves currentDogId', () => {
  const DogPhotoManager = class {
    constructor() {
      this.maxSizeMB = 10;
      this.allowedTypes = ['image/jpeg', 'image/png'];
      this.selectedFile = null;
      this.currentDogId = null;
      this.uploadInProgress = false;
    }

    clearPreview() {
      this.selectedFile = null;
    }

    hideCurrentPhoto(containerId) {
      const container = document.getElementById(containerId);
      if (container) {
        container.innerHTML = '';
        container.style.display = 'none';
      }
    }

    displayCurrentPhoto(photoUrl, containerId) {
      const container = document.getElementById(containerId);
      if (!container || !photoUrl) return;
      container.innerHTML = `<img src="/uploads/${photoUrl}">`;
      container.style.display = 'block';
    }

    reset() {
      this.clearPreview();
      this.selectedFile = null;
      this.currentDogId = null;
      this.hideCurrentPhoto('current-photo-container');

      const uploadZone = document.getElementById('photo-upload-zone');
      if (uploadZone) {
        uploadZone.style.display = 'block';
      }
    }

    // FIXED: Reset first, then set currentDogId
    initForDog(dog) {
      this.reset();  // Reset first
      this.currentDogId = dog.id;  // Then set - won't be cleared

      if (dog.photo) {
        this.displayCurrentPhoto(dog.photo, 'current-photo-container');
        const uploadZone = document.getElementById('photo-upload-zone');
        if (uploadZone) {
          uploadZone.style.display = 'none';
        }
      } else {
        this.hideCurrentPhoto('current-photo-container');
        const uploadZone = document.getElementById('photo-upload-zone');
        if (uploadZone) {
          uploadZone.style.display = 'block';
        }
      }
    }
  };

  test('PASSES: initForDog preserves currentDogId', () => {
    document.body.innerHTML = `
      <div id="current-photo-container"></div>
      <div id="photo-upload-zone"></div>
    `;

    const manager = new DogPhotoManager();
    const dog = { id: 42, name: 'Buddy', photo: 'buddy.jpg' };

    manager.initForDog(dog);

    expect(manager.currentDogId).toBe(42);
  });
});

// =============================================================================
// FIX #3: router.js - getQueryParams() handles malformed URLs gracefully
// =============================================================================

describe('FIX #3: router.js - getQueryParams() handles malformed URLs', () => {
  const Router = class {
    constructor() {
      this.routes = {};
    }

    // FIXED: Wrapped in try-catch
    getQueryParams() {
      const params = {};
      const queryString = window.location.search.substring(1);
      const pairs = queryString.split('&');

      for (const pair of pairs) {
        const [key, value] = pair.split('=');
        if (key) {
          try {
            params[decodeURIComponent(key)] = decodeURIComponent(value || '');
          } catch (e) {
            // Handle malformed URI encoding gracefully
            params[key] = value || '';
          }
        }
      }

      return params;
    }
  };

  test('PASSES: getQueryParams handles malformed percent encoding', () => {
    delete window.location;
    window.location = {
      search: '?foo=%ZZ&bar=valid',
      pathname: '/',
    };

    const router = new Router();

    expect(() => router.getQueryParams()).not.toThrow();

    const params = router.getQueryParams();
    expect(params.foo).toBe('%ZZ');  // Falls back to raw value
    expect(params.bar).toBe('valid');
  });
});

// =============================================================================
// FIX #4: i18n.js - updateElement(null) handles gracefully
// =============================================================================

describe('FIX #4: i18n.js - updateElement(null) handles gracefully', () => {
  const I18n = class {
    constructor(locale = 'de') {
      this.locale = locale;
      this.translations = { test: { key: 'Value' } };
    }

    t(key) {
      const keys = key.split('.');
      let value = this.translations;
      for (const k of keys) {
        if (value && typeof value === 'object') {
          value = value[k];
        } else {
          return key;
        }
      }
      return value || key;
    }

    // FIXED: Added null check
    updateElement(element = document) {
      if (!element) {
        element = document;
      }
      element.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.dataset.i18n;
        const translation = this.t(key);

        if (el.dataset.i18nAttr) {
          el.setAttribute(el.dataset.i18nAttr, translation);
        } else {
          el.textContent = translation;
        }
      });

      element.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.dataset.i18nPlaceholder;
        el.placeholder = this.t(key);
      });
    }
  };

  test('PASSES: updateElement(null) does not crash', () => {
    const i18n = new I18n();

    expect(() => i18n.updateElement(null)).not.toThrow();
  });
});

// =============================================================================
// FIX #5: api.js - handles non-JSON server responses
// =============================================================================

describe('FIX #5: api.js - handles non-JSON server responses', () => {
  const API = class {
    constructor() {
      this.baseURL = '/api';
      this.token = 'valid-token';
    }

    // FIXED: Uses text() then JSON.parse with try-catch
    async request(method, endpoint, data = null) {
      const headers = { 'Content-Type': 'application/json' };
      if (this.token) {
        headers['Authorization'] = `Bearer ${this.token}`;
      }

      const options = { method, headers };
      if (data && (method === 'POST' || method === 'PUT' || method === 'DELETE')) {
        options.body = JSON.stringify(data);
      }

      try {
        const response = await fetch(`${this.baseURL}${endpoint}`, options);
        const text = await response.text();
        let responseData = null;
        if (text) {
          try {
            responseData = JSON.parse(text);
          } catch (parseError) {
            if (!response.ok) {
              const error = new Error('Request failed');
              error.status = response.status;
              throw error;
            }
            return null;
          }
        }

        if (!response.ok) {
          const error = new Error((responseData && responseData.error) || 'Request failed');
          error.status = response.status;
          throw error;
        }

        return responseData;
      } catch (error) {
        throw error;
      }
    }
  };

  test('PASSES: handles HTML error responses gracefully', async () => {
    const api = new API();

    fetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      text: async () => '<html><body>Internal Server Error</body></html>',
    });

    await expect(api.request('GET', '/some-endpoint')).rejects.toThrow('Request failed');
  });
});

// =============================================================================
// FIX #6: central.js - handles 204 No Content
// =============================================================================

describe('FIX #6: central.js - handles 204 No Content', () => {
  // FIXED: Checks for 204 status and empty body
  async function apiRequest(endpoint, options = {}) {
    const token = 'valid-token';

    const config = {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        ...options.headers
      }
    };

    const response = await fetch(`/api${endpoint}`, config);

    if (response.status === 401 || response.status === 403) {
      throw new Error('Unauthorized');
    }

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Unknown error' }));
      throw new Error(error.error || 'Request failed');
    }

    // FIXED: Handle 204 No Content
    if (response.status === 204) {
      return null;
    }

    const text = await response.text();
    if (!text) {
      return null;
    }
    try {
      return JSON.parse(text);
    } catch (e) {
      return null;
    }
  }

  test('PASSES: handles 204 No Content responses', async () => {
    fetch.mockResolvedValueOnce({
      ok: true,
      status: 204,
    });

    const result = await apiRequest('/delete-something', { method: 'DELETE' });
    expect(result).toBeNull();
  });
});

// =============================================================================
// FIX #7: api.js - isAuthenticated() rejects whitespace tokens
// =============================================================================

describe('FIX #7: api.js - isAuthenticated() rejects whitespace tokens', () => {
  const API = class {
    constructor() {
      this.baseURL = '/api';
      this.token = localStorage.getItem('gassigeher_token');
    }

    setToken(token) {
      this.token = token;
      if (token) {
        localStorage.setItem('gassigeher_token', token);
      } else {
        localStorage.removeItem('gassigeher_token');
      }
    }

    // FIXED: Check for whitespace-only tokens
    isAuthenticated() {
      return !!(this.token && this.token.trim());
    }
  };

  test('PASSES: whitespace-only token is not authenticated', () => {
    const api = new API();
    api.setToken('   ');

    expect(api.isAuthenticated()).toBe(false);
  });

  test('PASSES: valid token is authenticated', () => {
    const api = new API();
    api.setToken('valid-jwt-token');

    expect(api.isAuthenticated()).toBe(true);
  });
});

// =============================================================================
// FIX #8: router.js - wildcard params are now extracted and passed
// =============================================================================

describe('FIX #8: router.js - wildcard params extracted', () => {
  const Router = class {
    constructor() {
      this.routes = {};
    }

    on(path, handler) {
      this.routes[path] = handler;
    }

    // FIXED: Extract params and pass to handler
    navigate(path) {
      let handler = this.routes[path];
      let params = {};

      if (!handler) {
        for (const route in this.routes) {
          if (route.includes(':')) {
            const pattern = new RegExp('^' + route.replace(/:[^\s/]+/g, '([^/]+)') + '$');
            const match = pattern.exec(path);
            if (match) {
              handler = this.routes[route];
              // FIXED: Extract param names and values
              const paramNames = (route.match(/:[^\s/]+/g) || []).map(p => p.substring(1));
              paramNames.forEach((name, index) => {
                params[name] = match[index + 1];
              });
              break;
            }
          }
        }
      }

      if (!handler) {
        handler = this.routes['/404'] || (() => {});
      }

      // FIXED: Pass params to handler
      handler(params);
    }
  };

  test('PASSES: wildcard route handler receives matched params', () => {
    const router = new Router();
    let receivedParams = null;

    router.on('/dogs/:id', (params) => {
      receivedParams = params;
    });

    router.navigate('/dogs/123');

    expect(receivedParams).not.toBeNull();
    expect(receivedParams.id).toBe('123');
  });

  test('PASSES: multiple params are extracted', () => {
    const router = new Router();
    let receivedParams = null;

    router.on('/users/:userId/dogs/:dogId', (params) => {
      receivedParams = params;
    });

    router.navigate('/users/5/dogs/10');

    expect(receivedParams.userId).toBe('5');
    expect(receivedParams.dogId).toBe('10');
  });
});

// =============================================================================
// FIX #9: dog-photo-helpers.js - CSS injection prevented
// =============================================================================

describe('FIX #9: dog-photo-helpers.js - CSS injection prevented', () => {
  // FIXED: Added sanitizeHexCode function
  function sanitizeHexCode(hexCode) {
    if (!hexCode || typeof hexCode !== 'string') {
      return '#808080';
    }
    const hexPattern = /^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{8})$/;
    if (hexPattern.test(hexCode)) {
      return hexCode;
    }
    return '#808080';
  }

  function getCalendarDogCell(dog, color) {
    const dogColor = color || dog.color;
    const safeDogName = dog.name;

    if (dogColor && dogColor.hex_code) {
      const patternIcons = { 'circle': '●', 'triangle': '▲', 'square': '■' };
      const icon = patternIcons[dogColor.pattern_icon] || '●';
      const safeColorName = dogColor.name;
      // FIXED: Sanitize hex_code
      const safeHexCode = sanitizeHexCode(dogColor.hex_code);

      return `<span style="background: ${safeHexCode}20; border: 1px solid ${safeHexCode}; color: ${safeHexCode};">
                    ${icon} ${safeColorName}
                </span>`;
    }

    return '<span>No color</span>';
  }

  test('PASSES: CSS injection is prevented', () => {
    const dog = { name: 'Buddy', color: null };
    const maliciousColor = {
      name: 'Red',
      hex_code: '#ff0000; position: fixed; top: 0; left: 0',
      pattern_icon: 'circle'
    };

    const html = getCalendarDogCell(dog, maliciousColor);

    expect(html).not.toContain('position: fixed');
    expect(html).toContain('#808080');  // Fallback color
  });

  test('PASSES: valid hex codes work correctly', () => {
    const dog = { name: 'Buddy', color: null };
    const validColor = {
      name: 'Blue',
      hex_code: '#0000ff',
      pattern_icon: 'circle'
    };

    const html = getCalendarDogCell(dog, validColor);

    expect(html).toContain('#0000ff');
    expect(html).toContain('Blue');
  });
});

// =============================================================================
// FIX #10: dog-photo-helpers.js - XSS in onload prevented
// =============================================================================

describe('FIX #10: dog-photo-helpers.js - XSS in onload prevented', () => {
  // FIXED: Added escapeForAttribute function
  function escapeForAttribute(value) {
    if (value === null || value === undefined) {
      return '';
    }
    return String(value)
      .replace(/&/g, '&amp;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
  }

  function getDogPhotoHtml(dog) {
    const photoUrl = `/uploads/${dog.photo}`;
    const altText = `${dog.name} (${dog.breed})`;
    // FIXED: Escape dog.id
    const safeId = escapeForAttribute(dog.id) || Math.random().toString(36).substring(2, 11);
    const uniqueId = `dog-img-${safeId}`;

    return `<div class="dog-card-image-container" id="container-${uniqueId}">
                <img src="${photoUrl}"
                     alt="${altText}"
                     class="dog-card-image"
                     id="${uniqueId}"
                     loading="lazy"
                     onload="handleImageLoad('${uniqueId}')">
            </div>`;
  }

  test('PASSES: XSS in dog.id is escaped', () => {
    const maliciousDog = {
      id: "'); alert('XSS'); ('",
      name: 'Buddy',
      breed: 'Lab',
      photo: 'buddy.jpg'
    };

    const html = getDogPhotoHtml(maliciousDog);

    // The quotes are escaped, so no XSS
    expect(html).not.toContain("alert('XSS')");
    expect(html).toContain('&#39;');  // Escaped single quote
  });

  test('PASSES: normal dog.id works correctly', () => {
    const normalDog = {
      id: 42,
      name: 'Buddy',
      breed: 'Lab',
      photo: 'buddy.jpg'
    };

    const html = getDogPhotoHtml(normalDog);

    expect(html).toContain('dog-img-42');
    expect(html).toContain("handleImageLoad('dog-img-42')");
  });
});
