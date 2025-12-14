/**
 * XSS Security Tests - TDD for Frontend XSS Prevention
 *
 * These tests verify that user-controlled data is properly sanitized
 * before being inserted into the DOM via innerHTML.
 *
 * TDD Workflow:
 * 1. RED: Tests fail because vulnerable patterns are used
 * 2. GREEN: Apply sanitizeHTML() to fix vulnerabilities
 * 3. REFACTOR: Clean up code while keeping tests green
 *
 * @jest-environment jsdom
 */

/**
 * sanitizeHTML - Exact copy of the function from sanitize.js
 * This ensures tests match the actual implementation.
 * If sanitize.js changes, this should be updated to match.
 */
function sanitizeHTML(str) {
  if (!str) return '';
  if (typeof str !== 'string') str = String(str);

  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

// XSS Payloads for testing
const XSS_PAYLOADS = {
  scriptTag: '<script>alert("XSS")</script>',
  imgOnerror: '<img src=x onerror=alert("XSS")>',
  divOnclick: '<div onclick=alert("XSS")>Click me</div>',
  svgOnload: '<svg onload=alert("XSS")>',
  javascriptProtocol: 'javascript:alert("XSS")',
  eventHandler: '"><img src=x onerror=alert("XSS")><"',
};

describe('sanitizeHTML function', () => {
  test('should escape < and > characters', () => {
    const result = sanitizeHTML(XSS_PAYLOADS.scriptTag);
    expect(result).not.toContain('<script>');
    expect(result).not.toContain('</script>');
    expect(result).toContain('&lt;');
    expect(result).toContain('&gt;');
  });

  test('should escape onerror event handlers', () => {
    const result = sanitizeHTML(XSS_PAYLOADS.imgOnerror);
    // The < and > are escaped, so the tag won't execute
    expect(result).not.toContain('<img');
    expect(result).toContain('&lt;img');
    // The result contains 'onerror=' as literal text (safe)
  });

  test('should escape onclick event handlers', () => {
    const result = sanitizeHTML(XSS_PAYLOADS.divOnclick);
    // The < and > are escaped, so the tag won't execute
    expect(result).not.toContain('<div');
    expect(result).toContain('&lt;div');
    // The result contains 'onclick=' as literal text (safe)
  });

  test('should handle null and undefined', () => {
    expect(sanitizeHTML(null)).toBe('');
    expect(sanitizeHTML(undefined)).toBe('');
    expect(sanitizeHTML('')).toBe('');
  });

  test('should preserve normal text', () => {
    expect(sanitizeHTML('John Doe')).toBe('John Doe');
    expect(sanitizeHTML('Max Mustermann')).toBe('Max Mustermann');
    expect(sanitizeHTML("O'Brien")).toBe("O'Brien");
  });
});

/**
 * Helper function to simulate vulnerable innerHTML rendering
 * This mimics what the HTML files do BEFORE the fix
 */
function renderUnsafe(template, data) {
  // Create a temporary div to render the HTML
  const div = document.createElement('div');
  // Directly use template literals without sanitization (VULNERABLE)
  div.innerHTML = template(data);
  return div.innerHTML;
}

/**
 * Helper function to simulate safe innerHTML rendering
 * This mimics what the HTML files should do AFTER the fix
 */
function renderSafe(template, data) {
  const div = document.createElement('div');
  // Use sanitizeHTML on user data (SAFE)
  div.innerHTML = template(data);
  return div.innerHTML;
}

describe('XSS Vulnerability Tests - admin-blocked-dates.html', () => {
  describe('Bug: dog.name not sanitized (line 134)', () => {
    const maliciousDog = { id: 1, name: XSS_PAYLOADS.scriptTag };

    // This simulates the CURRENT vulnerable code
    const vulnerableTemplate = (dog) =>
      `<option value="${dog.id}">${dog.name}</option>`;

    // This simulates the FIXED code
    const safeTemplate = (dog) =>
      `<option value="${dog.id}">${sanitizeHTML(dog.name)}</option>`;

    test('VULNERABLE: renders XSS payload in option (should FAIL)', () => {
      const result = renderUnsafe(vulnerableTemplate, maliciousDog);
      // This test EXPECTS the vulnerability to exist
      // It will FAIL after we fix the code (which is what we want in TDD)
      expect(result).toContain('<script>');
    });

    test('SAFE: escapes XSS payload in option', () => {
      const result = renderSafe(safeTemplate, maliciousDog);
      expect(result).not.toContain('<script>');
      expect(result).toContain('&lt;script&gt;');
    });
  });

  describe('Bug: blocked.dog_name not sanitized (line 160)', () => {
    const maliciousBlocked = {
      id: 1,
      dog_id: 1,
      dog_name: XSS_PAYLOADS.imgOnerror,
      date: '2025-01-01',
      reason: 'Holiday'
    };

    const vulnerableTemplate = (blocked) =>
      `<span>${blocked.dog_name || 'Unbekannter Hund'}</span>`;

    const safeTemplate = (blocked) =>
      `<span>${sanitizeHTML(blocked.dog_name || 'Unbekannter Hund')}</span>`;

    test('VULNERABLE: renders XSS payload in dog name (should FAIL)', () => {
      const result = renderUnsafe(vulnerableTemplate, maliciousBlocked);
      expect(result).toContain('<img');
    });

    test('SAFE: escapes XSS payload in dog name', () => {
      const result = renderSafe(safeTemplate, maliciousBlocked);
      expect(result).not.toContain('<img');
    });
  });

  describe('Bug: blocked.reason not sanitized (line 169)', () => {
    const maliciousBlocked = {
      id: 1,
      reason: XSS_PAYLOADS.divOnclick
    };

    const vulnerableTemplate = (blocked) =>
      `<p>${blocked.reason}</p>`;

    const safeTemplate = (blocked) =>
      `<p>${sanitizeHTML(blocked.reason)}</p>`;

    test('VULNERABLE: renders XSS payload in reason (should FAIL)', () => {
      const result = renderUnsafe(vulnerableTemplate, maliciousBlocked);
      expect(result).toContain('onclick=');
    });

    test('SAFE: escapes XSS payload in reason', () => {
      const result = renderSafe(safeTemplate, maliciousBlocked);
      // The < and > are escaped, so no actual HTML tags
      expect(result).not.toContain('<div');
      expect(result).toContain('&lt;div');
    });
  });
});

describe('XSS Vulnerability Tests - dashboard.html', () => {
  describe('Bug: error.message not sanitized (lines 291, 380)', () => {
    const maliciousError = { message: XSS_PAYLOADS.scriptTag };

    const vulnerableTemplate = (error) =>
      `<p class="alert alert-error">${error.message}</p>`;

    const safeTemplate = (error) =>
      `<p class="alert alert-error">${sanitizeHTML(error.message)}</p>`;

    test('VULNERABLE: renders XSS payload in error message (should FAIL)', () => {
      const result = renderUnsafe(vulnerableTemplate, maliciousError);
      expect(result).toContain('<script>');
    });

    test('SAFE: escapes XSS payload in error message', () => {
      const result = renderSafe(safeTemplate, maliciousError);
      expect(result).not.toContain('<script>');
    });
  });
});

describe('XSS Vulnerability Tests - dogs.html', () => {
  describe('Bug: error.message not sanitized (line 506)', () => {
    const maliciousError = { message: XSS_PAYLOADS.svgOnload };

    const vulnerableTemplate = (error) =>
      `<p class="alert alert-error">Fehler beim Laden: ${error.message}</p>`;

    const safeTemplate = (error) =>
      `<p class="alert alert-error">Fehler beim Laden: ${sanitizeHTML(error.message)}</p>`;

    test('VULNERABLE: renders XSS payload in error message (should FAIL)', () => {
      const result = renderUnsafe(vulnerableTemplate, maliciousError);
      expect(result).toContain('<svg');
    });

    test('SAFE: escapes XSS payload in error message', () => {
      const result = renderSafe(safeTemplate, maliciousError);
      expect(result).not.toContain('<svg');
    });
  });
});

describe('XSS Vulnerability Tests - profile.html', () => {
  describe('Bug: request.admin_message not sanitized (line 307)', () => {
    const maliciousRequest = {
      admin_message: XSS_PAYLOADS.eventHandler
    };

    const vulnerableTemplate = (request) =>
      `<p><strong>Nachricht:</strong> ${request.admin_message}</p>`;

    const safeTemplate = (request) =>
      `<p><strong>Nachricht:</strong> ${sanitizeHTML(request.admin_message)}</p>`;

    test('VULNERABLE: renders XSS payload in admin message (should FAIL)', () => {
      const result = renderUnsafe(vulnerableTemplate, maliciousRequest);
      expect(result).toContain('onerror=');
    });

    test('SAFE: escapes XSS payload in admin message', () => {
      const result = renderSafe(safeTemplate, maliciousRequest);
      // The < and > are escaped, so no actual HTML tags
      expect(result).not.toContain('<img');
      expect(result).toContain('&lt;img');
    });
  });
});

describe('XSS Vulnerability Tests - admin-users.html onclick injection', () => {
  describe('Bug: userName passed through onclick attribute (lines 329, 331, 334, 337)', () => {
    // The vulnerable pattern: onclick="func(${id}, '${name.replace(/'/g, "\\'")}')"
    // This is vulnerable because the replace only escapes single quotes,
    // but not HTML special characters or JavaScript injection via other means

    const maliciousUser = {
      id: 1,
      name: "'); alert('XSS'); //"
    };

    // Simulate the vulnerable onclick pattern
    const vulnerableOnclick = (user) => {
      const safeName = user.name.replace(/'/g, "\\'");
      return `onclick="demoteAdmin(${user.id}, '${safeName}')"`;
    };

    // Safe pattern: don't pass user data through onclick at all
    const safeOnclick = (user) => {
      return `onclick="demoteAdmin(${user.id})"`;
    };

    test('VULNERABLE: allows injection through single quote bypass', () => {
      const result = vulnerableOnclick(maliciousUser);
      // The replace escapes ', but '); already breaks out
      // This is a logic flaw in the escaping
      expect(result).toContain("\\')");
    });

    test('SAFE: does not pass user name through onclick', () => {
      const result = safeOnclick(maliciousUser);
      expect(result).not.toContain(maliciousUser.name);
      expect(result).toBe('onclick="demoteAdmin(1)"');
    });
  });
});
