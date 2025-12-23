/**
 * Sanitize Module Tests
 *
 * Tests for the sanitization utilities that prevent XSS attacks.
 *
 * @jest-environment jsdom
 */

// Load the actual sanitize source file
beforeAll(() => {
  document.body.innerHTML = '';
  loadSourceFile('internal/static/frontend/js/sanitize.js');
});

describe('sanitizeHTML', () => {
  test('should escape script tags', () => {
    const result = window.sanitizeHTML('<script>alert("XSS")</script>');
    expect(result).not.toContain('<script>');
    expect(result).toContain('&lt;script&gt;');
    expect(result).toContain('&lt;/script&gt;');
  });

  test('should escape img tags with onerror', () => {
    const result = window.sanitizeHTML('<img src=x onerror=alert(1)>');
    expect(result).not.toContain('<img');
    expect(result).toContain('&lt;img');
  });

  test('should escape svg tags', () => {
    const result = window.sanitizeHTML('<svg onload=alert("XSS")>');
    expect(result).not.toContain('<svg');
    expect(result).toContain('&lt;svg');
  });

  test('should escape ampersands', () => {
    const result = window.sanitizeHTML('foo & bar');
    expect(result).toContain('&amp;');
  });

  test('should handle empty string', () => {
    expect(window.sanitizeHTML('')).toBe('');
  });

  test('should handle null', () => {
    expect(window.sanitizeHTML(null)).toBe('');
  });

  test('should handle undefined', () => {
    expect(window.sanitizeHTML(undefined)).toBe('');
  });

  test('should convert numbers to string', () => {
    expect(window.sanitizeHTML(123)).toBe('123');
  });

  test('should preserve normal text', () => {
    expect(window.sanitizeHTML('Hello World')).toBe('Hello World');
  });

  test('should preserve German umlauts', () => {
    expect(window.sanitizeHTML('Größe')).toBe('Größe');
    expect(window.sanitizeHTML('Über')).toBe('Über');
    expect(window.sanitizeHTML('Müller')).toBe('Müller');
  });

  test('should handle quotes (not escaped by textContent)', () => {
    const result = window.sanitizeHTML('Say "Hello"');
    expect(result).toBe('Say "Hello"');
  });

  test('should handle complex XSS payloads', () => {
    const payload = '"><img src=x onerror=alert(1)><"';
    const result = window.sanitizeHTML(payload);
    expect(result).not.toContain('<img');
    expect(result).toContain('&lt;img');
  });

  test('should handle nested script tags', () => {
    const payload = '<script><script>alert(1)</script></script>';
    const result = window.sanitizeHTML(payload);
    expect(result).not.toMatch(/<script>/);
  });

  test('should handle javascript protocol', () => {
    const result = window.sanitizeHTML('javascript:alert(1)');
    // textContent doesn't escape javascript: but it's safe in text context
    expect(result).toBe('javascript:alert(1)');
  });
});

describe('setTextContent', () => {
  test('should set text content of element', () => {
    const element = document.createElement('div');
    window.setTextContent(element, 'Hello World');
    expect(element.textContent).toBe('Hello World');
  });

  test('should escape HTML when setting text', () => {
    const element = document.createElement('div');
    window.setTextContent(element, '<script>alert(1)</script>');
    expect(element.textContent).toBe('<script>alert(1)</script>');
    expect(element.innerHTML).toContain('&lt;script&gt;');
  });

  test('should handle null element', () => {
    expect(() => window.setTextContent(null, 'test')).not.toThrow();
  });

  test('should handle null text', () => {
    const element = document.createElement('div');
    window.setTextContent(element, null);
    expect(element.textContent).toBe('');
  });

  test('should handle undefined text', () => {
    const element = document.createElement('div');
    window.setTextContent(element, undefined);
    expect(element.textContent).toBe('');
  });
});

describe('createSafeElement', () => {
  test('should create element with text content', () => {
    const element = window.createSafeElement('div', 'Hello');
    expect(element.tagName).toBe('DIV');
    expect(element.textContent).toBe('Hello');
  });

  test('should create element with class attribute', () => {
    const element = window.createSafeElement('span', 'Test', { class: 'my-class' });
    expect(element.className).toBe('my-class');
  });

  test('should create element with style object', () => {
    const element = window.createSafeElement('div', 'Test', {
      style: { color: 'red', fontSize: '14px' }
    });
    expect(element.style.color).toBe('red');
    expect(element.style.fontSize).toBe('14px');
  });

  test('should create element with data attributes', () => {
    const element = window.createSafeElement('button', 'Click', {
      'data-id': '123',
      'data-action': 'submit'
    });
    expect(element.getAttribute('data-id')).toBe('123');
    expect(element.getAttribute('data-action')).toBe('submit');
  });

  test('should escape XSS in text content', () => {
    const element = window.createSafeElement('p', '<script>alert(1)</script>');
    expect(element.textContent).toBe('<script>alert(1)</script>');
    expect(element.innerHTML).toContain('&lt;script&gt;');
  });

  test('should create different element types', () => {
    expect(window.createSafeElement('p', 'text').tagName).toBe('P');
    expect(window.createSafeElement('span', 'text').tagName).toBe('SPAN');
    expect(window.createSafeElement('button', 'text').tagName).toBe('BUTTON');
    expect(window.createSafeElement('a', 'text').tagName).toBe('A');
  });

  test('should handle empty attributes object', () => {
    const element = window.createSafeElement('div', 'Test', {});
    expect(element.textContent).toBe('Test');
  });

  test('should create element with id', () => {
    const element = window.createSafeElement('div', 'Test', { id: 'my-id' });
    expect(element.id).toBe('my-id');
  });

  test('should create element with href', () => {
    const element = window.createSafeElement('a', 'Link', { href: '/page' });
    expect(element.getAttribute('href')).toBe('/page');
  });
});
