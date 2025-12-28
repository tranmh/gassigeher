# Bugs Found in Zero-Touch User Support Feature

**Date:** 2025-12-28
**Feature:** Zero-Touch User Support (Phases 1, 3, 7)
**Tested Files:**
- `internal/static/frontend/js/booking-errors.js`
- `internal/static/frontend/js/help-tooltips.js`
- `internal/static/shared/js/faq-data.js`
- `internal/static/frontend/account-status.html`

---

## CRITICAL: XSS Vulnerabilities

### Bug #1: XSS in booking-errors.js - renderError()

**Location:** `booking-errors.js:263-302`

**Description:** The `renderError()` function uses template literals with direct string interpolation for `errorInfo.title`, `errorInfo.message`, `errorInfo.solution`, and `errorInfo.action.text`. These values are inserted directly into HTML without escaping.

**Severity:** HIGH - If an attacker can control error messages from the server, they can execute arbitrary JavaScript.

**Proof of Concept:**
```javascript
const maliciousError = {
    title: '<script>alert("XSS")</script>',
    message: 'Test',
    solution: 'Test',
    icon: '⚠️'
};
const html = BookingErrors.renderError(maliciousError);
// html contains raw <script> tag
```

**Fix Required:**
```javascript
// Use textContent or escape HTML before interpolation
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Then use: ${escapeHtml(errorInfo.title)}
```

---

### Bug #2: XSS in booking-errors.js - javascript: protocol in href

**Location:** `booking-errors.js:267-269`

**Description:** The action button's `href` attribute is not validated. A `javascript:` URL can execute arbitrary code.

**Proof of Concept:**
```javascript
const maliciousError = {
    action: {
        text: 'Click me',
        href: 'javascript:alert("XSS")'
    }
};
```

**Fix Required:** Validate that href starts with `/`, `http://`, or `https://`:
```javascript
if (errorInfo.action.href && !errorInfo.action.href.match(/^(\/|https?:\/\/)/)) {
    errorInfo.action.href = '#'; // Fallback to safe value
}
```

---

### Bug #3: XSS in help-tooltips.js - showTooltip()

**Location:** `help-tooltips.js:257-259`

**Description:** Tooltip content (`content.title` and `content.text`) is inserted directly into innerHTML without escaping.

**Severity:** MEDIUM - The content comes from hardcoded values or i18n, but if i18n is compromised or user-controlled, XSS is possible.

**Proof of Concept:**
```javascript
HelpTooltips.content['xss_test'] = {
    title: '<svg onload=alert(1)>',
    text: 'Test'
};
// Tooltip will render raw SVG with onload handler
```

**Fix Required:** Use textContent for title/text elements:
```javascript
const titleEl = tooltip.querySelector('.help-tooltip-title');
titleEl.textContent = content.title; // Safe
```

---

## MEDIUM: Logic Errors

### Bug #4: German "gesperrt" incorrectly triggers date_blocked

**Location:** `booking-errors.js:193`

**Description:** The parseError function checks for "gesperrt" (German for "blocked") and always maps it to `date_blocked`, even if the message is about time being blocked ("Zeit gesperrt").

**Code:**
```javascript
} else if (serverMessage.includes('blocked') && serverMessage.includes('date') || serverMessage.includes('gesperrt')) {
    errorCode = 'date_blocked';
```

**Issue:** Operator precedence - the `||` binds looser than `&&`, so "gesperrt" alone triggers date_blocked.

**Expected Behavior:** "Zeit gesperrt" should map to `time_blocked`, "Datum gesperrt" to `date_blocked`.

**Fix Required:**
```javascript
} else if ((serverMessage.includes('blocked') && serverMessage.includes('date')) ||
           (serverMessage.includes('gesperrt') && serverMessage.includes('datum'))) {
    errorCode = 'date_blocked';
} else if ((serverMessage.includes('blocked') && serverMessage.includes('time')) ||
           (serverMessage.includes('gesperrt') && serverMessage.includes('zeit'))) {
    errorCode = 'time_blocked';
```

---

### Bug #5: Multiple init() calls add duplicate event listeners

**Location:** `help-tooltips.js:145-156, 201-225`

**Description:** Calling `HelpTooltips.init()` multiple times adds duplicate event listeners to document and elements. This causes tooltips to open/close multiple times per click.

**Proof of Concept:**
```javascript
HelpTooltips.init();
HelpTooltips.init();
HelpTooltips.init();
// Now clicking a help icon triggers 3 handlers
```

**Fix Required:** Track initialization state or remove listeners before adding:
```javascript
init(options = {}) {
    if (this.initialized) {
        this.cleanup(); // Remove old listeners
    }
    this.initialized = true;
    // ... rest of init
}
```

---

### Bug #6: TypeError in countEligibleDogs when color_category.name is undefined

**Location:** `account-status.html:~373` (inline script)

**Description:** The function assumes `dog.color_category.name` exists. If a dog has `color_category: {}` (object with missing name), it throws TypeError.

**Error:** `Cannot read properties of undefined (reading 'toLowerCase')`

**Fix Required:**
```javascript
return dogs.filter(dog => {
    if (!dog.is_available) return false;
    if (!dog.color_category || !dog.color_category.name) return true;
    return userColorNames.includes(dog.color_category.name.toLowerCase());
});
```

---

### Bug #7: searchFAQs throws on non-string input

**Location:** `faq-data.js:367`

**Description:** `searchFAQs()` calls `query.toLowerCase()` without checking if query is a string. Passing null, undefined, or a number crashes.

**Error:** `TypeError: Cannot read properties of null (reading 'toLowerCase')`

**Fix Required:**
```javascript
function searchFAQs(query, landingOnly = false) {
    if (!query || typeof query !== 'string') {
        return landingOnly ? getFAQsForLanding() : getFAQsForApp();
    }
    const normalizedQuery = query.toLowerCase().trim();
    // ... rest of function
}
```

---

### Bug #8: Division by zero in calculateProgressPercent

**Location:** `account-status.html:~470` (inline script)

**Description:** `(daysUntilDeactivation / autoDeactivationDays) * 100` can result in Infinity if `autoDeactivationDays` is 0.

**Note:** Currently Math.min(100, ...) caps the result, but Infinity > 100 returns 100, which masks the bug. Better to explicitly handle.

**Fix Required:**
```javascript
const progressPercent = autoDeactivationDays > 0
    ? Math.min(100, (daysUntilDeactivation / autoDeactivationDays) * 100)
    : 0;
```

---

## LOW: Edge Cases

### Bug #9: Shallow clone in getErrorInfo doesn't deep clone action

**Location:** `booking-errors.js:224`

**Description:** `{ ...errorDef, code }` creates a shallow clone. The `action` object is shared with the original errorMap. Modifying `result.action.href` would mutate the global errorMap.

**Current Impact:** Low - no current code mutates the result.

**Fix Required:**
```javascript
const errorInfo = {
    ...errorDef,
    code,
    action: errorDef.action ? { ...errorDef.action } : undefined
};
```

---

### Bug #10: Negative progress percentage not clamped

**Location:** `account-status.html` (inline script)

**Description:** If `daysUntilDeactivation` is somehow negative (shouldn't happen but edge case), the progress bar would have negative width.

**Fix Required:**
```javascript
const progressPercent = Math.min(100, Math.max(0, (daysUntilDeactivation / autoDeactivationDays) * 100));
```

---

### Bug #11: Future dates give negative days in calculateDaysAgo

**Location:** `account-status.html` (inline script)

**Description:** If `lastActivity` is in the future (e.g., timezone issue), `daysAgo` becomes negative, displaying "-1 days ago".

**Fix Required:**
```javascript
const daysAgo = Math.max(0, Math.floor((Date.now() - lastActivity.getTime()) / (1000 * 60 * 60 * 24)));
```

---

## Summary

| # | Type | Severity | File | Status |
|---|------|----------|------|--------|
| 1 | XSS | HIGH | booking-errors.js | FIXED |
| 2 | XSS | HIGH | booking-errors.js | FIXED |
| 3 | XSS | MEDIUM | help-tooltips.js | FIXED |
| 4 | Logic | MEDIUM | booking-errors.js | FIXED |
| 5 | Logic | MEDIUM | help-tooltips.js | FIXED |
| 6 | TypeError | MEDIUM | account-status.html | FIXED |
| 7 | TypeError | MEDIUM | faq-data.js | FIXED |
| 8 | Edge Case | LOW | account-status.html | FIXED |
| 9 | Edge Case | LOW | booking-errors.js | FIXED |
| 10 | Edge Case | LOW | account-status.html | FIXED |
| 11 | Edge Case | LOW | account-status.html | FIXED |

**Total: 11 bugs found and FIXED**
- HIGH: 2 (FIXED)
- MEDIUM: 5 (FIXED)
- LOW: 4 (FIXED)

---

## Test Coverage

Tests written in:
- `frontend-tests/booking-errors.test.js` (182 assertions)
- `frontend-tests/help-tooltips.test.js` (85 assertions)
- `frontend-tests/faq-data.test.js` (63 assertions)
- `frontend-tests/account-status-logic.test.js` (50 assertions)

Run with: `npm test -- --testPathPattern="(booking-errors|help-tooltips|faq-data|account-status)"`
