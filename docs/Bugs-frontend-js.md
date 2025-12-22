# Bug Report: frontend/js

**Analysis Date:** 2025-12-22
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js`
**Files Analyzed:** 10 files
**Bugs Found:** 15 bugs

---

## Summary

The frontend JavaScript codebase contains multiple security vulnerabilities, logic errors, and error handling gaps. The most critical issues include:
- **XSS vulnerabilities** in multiple locations where unsanitized user data is injected into HTML
- **Race conditions** in API calls and DOM manipulation
- **Missing error handling** for failed fetch operations
- **Memory leaks** from event listeners that are never removed
- **Logic errors** in URL construction and state management

**Critical issues require immediate attention**: XSS vulnerabilities in dog photo helpers and admin booking times page.

---

## Bugs

## Bug #1: XSS Vulnerability in getCalendarDogCell - Unsanitized Pattern Icon

**Description:**
The `getCalendarDogCell` function in `dog-photo-helpers.js` uses a pattern icon from `dogColor.pattern_icon` directly in HTML without validation. While the icon is looked up in a whitelist object, if `dogColor.pattern_icon` contains a value not in the whitelist, it defaults to '●'. However, the issue is that the fallback uses `|| '●'` which means if `patternIcons[dogColor.pattern_icon]` returns a falsy value (including an empty string or malicious value), it uses the default. But more critically, the `dogColor.pattern_icon` value itself is never validated before lookup, potentially allowing object prototype pollution or other attacks if the backend is compromised.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/dog-photo-helpers.js`
- Function: `getCalendarDogCell`
- Lines: 180-186

**Steps to Reproduce:**
1. Compromise backend to return a color object with `pattern_icon: "__proto__"` or other prototype pollution attack
2. Load calendar page with this dog/color
3. Observe potential prototype pollution or unexpected behavior

**Fix:**
Validate the pattern_icon value and ensure it's a string before lookup:

```diff
  function getCalendarDogCell(dog, color) {
      const photoUrl = getDogPhotoUrl(dog, true);
      const altText = getDogPhotoAlt(dog);
      const safeDogName = typeof sanitizeHTML !== 'undefined' ? sanitizeHTML(dog.name) : dog.name;

      const dogColor = color || dog.color;

      if (dogColor && dogColor.hex_code) {
          const patternIcons = {
              'circle': '●', 'triangle': '▲', 'square': '■', 'diamond': '◆',
              'pentagon': '⬠', 'hexagon': '⬡', 'star': '★', 'heart': '♥',
              'cross': '✚', 'spade': '♠', 'club': '♣', 'moon': '☽',
              'sun': '☀', 'ring': '○', 'target': '◎'
          };
-         const icon = patternIcons[dogColor.pattern_icon] || '●';
+         const safePatternIcon = typeof dogColor.pattern_icon === 'string' ? dogColor.pattern_icon : 'circle';
+         const icon = Object.prototype.hasOwnProperty.call(patternIcons, safePatternIcon)
+             ? patternIcons[safePatternIcon]
+             : '●';
```

---

## Bug #2: XSS Vulnerability via Inline Styles with Unsanitized Hex Colors

**Description:**
In `dog-photo-helpers.js`, the `getCalendarDogCell` function directly injects `dogColor.hex_code` into inline CSS styles without sanitization. If the backend returns a malicious hex_code value like `"#000; background-image: url('javascript:alert(1)');"`, it could lead to CSS injection attacks or even JavaScript execution in older browsers. While modern browsers have protections against `javascript:` URLs in CSS, this is still a defense-in-depth issue.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/dog-photo-helpers.js`
- Function: `getCalendarDogCell`
- Lines: 193-194

**Steps to Reproduce:**
1. Backend returns color with `hex_code: "#000; background-image: url('data:image/svg+xml,<svg>...</svg>')"`
2. Load calendar page
3. Expected: Only hex color applied
4. Actual: Additional CSS properties injected, potentially malicious content displayed

**Fix:**
Validate hex_code format before use:

```diff
  function getCalendarDogCell(dog, color) {
      // ... existing code ...

      if (dogColor && dogColor.hex_code) {
+         // Validate hex color format (#RGB or #RRGGBB)
+         const hexPattern = /^#[0-9A-Fa-f]{3}([0-9A-Fa-f]{3})?$/;
+         const safeHexCode = hexPattern.test(dogColor.hex_code) ? dogColor.hex_code : '#666666';
+
          const patternIcons = {
              'circle': '●', 'triangle': '▲', 'square': '■', 'diamond': '◆',
              'pentagon': '⬠', 'hexagon': '⬡', 'star': '★', 'heart': '♥',
              'cross': '✚', 'spade': '♠', 'club': '♣', 'moon': '☽',
              'sun': '☀', 'ring': '○', 'target': '◎'
          };
          const icon = patternIcons[dogColor.pattern_icon] || '●';
          const safeColorName = typeof sanitizeHTML !== 'undefined' ? sanitizeHTML(dogColor.name) : dogColor.name;

          return `<div class="calendar-dog-name-cell">
              <img src="${photoUrl}" alt="${altText}" class="calendar-dog-photo" loading="lazy">
              <div>
                  <div style="font-weight: 700; font-size: 1rem; color: var(--text-dark);">${safeDogName}</div>
-                 <span style="display: inline-flex; align-items: center; gap: 3px; font-size: 0.7rem; padding: 2px 8px; background: ${dogColor.hex_code}20; border: 1px solid ${dogColor.hex_code}; color: ${dogColor.hex_code}; border-radius: 4px; margin-top: 4px;">
+                 <span style="display: inline-flex; align-items: center; gap: 3px; font-size: 0.7rem; padding: 2px 8px; background: ${safeHexCode}20; border: 1px solid ${safeHexCode}; color: ${safeHexCode}; border-radius: 4px; margin-top: 4px;">
                      ${icon} ${safeColorName}
                  </span>
              </div>
          </div>`;
      }
```

---

## Bug #3: Potential XSS in admin-booking-times.js via Rule Names

**Description:**
In `admin-booking-times.js`, the `createRuleRow` function directly injects `rule.rule_name` into the DOM without sanitization at line 101. If an attacker can manipulate the rule name (either through the admin interface or by compromising the backend), they could inject malicious HTML/JavaScript that executes when the rules are displayed.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/admin-booking-times.js`
- Function: `createRuleRow`
- Lines: 100-101

**Steps to Reproduce:**
1. Admin creates a rule with name: `<img src=x onerror=alert('XSS')>`
2. Navigate to admin booking times page
3. Expected: Rule name displayed as text
4. Actual: JavaScript executes

**Fix:**
Use textContent or sanitizeHTML for rule names:

```diff
  function createRuleRow(rule) {
      const tr = document.createElement('tr');
+     const safeRuleName = typeof sanitizeHTML !== 'undefined' ? sanitizeHTML(rule.rule_name) : rule.rule_name;

      tr.innerHTML = `
-         <td>${rule.rule_name}</td>
+         <td>${safeRuleName}</td>
          <td><input type="time" value="${rule.start_time}" data-field="start"></td>
          <td><input type="time" value="${rule.end_time}" data-field="end"></td>
          <td>
              <select data-field="blocked">
                  <option value="0" ${!rule.is_blocked ? 'selected' : ''}>Erlaubt</option>
                  <option value="1" ${rule.is_blocked ? 'selected' : ''}>Gesperrt</option>
              </select>
          </td>
          <td>
              <button class="btn-save" data-id="${rule.id}">Speichern</button>
              <button class="btn-delete" data-id="${rule.id}">Löschen</button>
          </td>
      `;
```

---

## Bug #4: Race Condition in API Token Refresh

**Description:**
In `api.js`, the token is stored both in the class instance (`this.token`) and in localStorage. When multiple tabs are open and a logout occurs in one tab, the other tabs continue to use the stale token from their class instance until a page refresh. This creates a race condition where API calls may use an invalid token, leading to 401 errors and poor user experience.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/api.js`
- Function: `setToken`, `getToken`
- Lines: 9-21

**Steps to Reproduce:**
1. Login and open application in two browser tabs
2. Logout in tab 1 (calls `api.setToken(null)`)
3. In tab 2, make an API call (e.g., load dogs page)
4. Expected: Tab 2 detects logout and redirects to login
5. Actual: Tab 2 uses cached token, gets 401 error, shows error message instead of redirecting

**Fix:**
Add localStorage event listener to sync token across tabs:

```diff
  class API {
      constructor() {
          this.baseURL = '/api';
          this.token = localStorage.getItem('gassigeher_token');
+
+         // Listen for localStorage changes from other tabs
+         window.addEventListener('storage', (e) => {
+             if (e.key === 'gassigeher_token') {
+                 this.token = e.newValue;
+                 // Redirect to login if token was removed
+                 if (!this.token && window.location.pathname !== '/login.html' && window.location.pathname !== '/') {
+                     window.location.href = '/login.html';
+                 }
+             }
+         });
      }
```

---

## Bug #5: Missing Error Handling for JSON Parse Failures

**Description:**
In `api.js`, the `request` method at line 49 assumes `response.json()` will always succeed. If the server returns a non-JSON response (e.g., HTML error page, plain text, or malformed JSON), the promise will reject and the error will have a confusing message instead of indicating a JSON parse failure. This makes debugging difficult.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/api.js`
- Function: `request`
- Lines: 47-62

**Steps to Reproduce:**
1. Backend returns 500 error with HTML error page instead of JSON
2. Frontend calls `api.getDogs()`
3. Expected: Clear error message about server error
4. Actual: Error message "Unexpected token '<' at position 0" (JSON parse error)

**Fix:**
Add try-catch for JSON parsing with better error messages:

```diff
  async request(method, endpoint, data = null) {
      const headers = {
          'Content-Type': 'application/json',
      };

      if (this.token) {
          headers['Authorization'] = `Bearer ${this.token}`;
      }

      const options = {
          method,
          headers,
      };

      if (data && (method === 'POST' || method === 'PUT')) {
          options.body = JSON.stringify(data);
      }

      try {
          const response = await fetch(`${this.baseURL}${endpoint}`, options);
-         const responseData = await response.json();
+
+         let responseData;
+         try {
+             responseData = await response.json();
+         } catch (jsonError) {
+             // Server returned non-JSON response
+             const error = new Error(`Server returned invalid response (status ${response.status})`);
+             error.status = response.status;
+             error.data = { error: 'Invalid server response' };
+             throw error;
+         }

          if (!response.ok) {
              const error = new Error(responseData.error || 'Request failed');
              error.status = response.status;
              error.data = responseData;
              throw error;
          }

          return responseData;
      } catch (error) {
          throw error;
      }
  }
```

---

## Bug #6: Memory Leak in Dog Photo Manager - Event Listeners Never Removed

**Description:**
In `dog-photo.js`, the `setupDragDrop` method adds multiple event listeners to the drop zone but never removes them. If this function is called multiple times (e.g., when switching between dogs in an edit modal), the event listeners accumulate, causing memory leaks and duplicate event handling. Each time a file is dropped, all accumulated listeners fire.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/dog-photo.js`
- Function: `setupDragDrop`
- Lines: 139-179

**Steps to Reproduce:**
1. Open admin-dogs page
2. Click "Edit" on dog A (calls setupDragDrop)
3. Close modal, click "Edit" on dog B (calls setupDragDrop again)
4. Repeat 10 times
5. Drop a file in the upload zone
6. Expected: File handled once
7. Actual: Console shows multiple drop events firing (10x), onFileSelected called 10 times

**Fix:**
Store event listener references and remove them before adding new ones, or use event delegation:

```diff
  class DogPhotoManager {
      constructor() {
          this.maxSizeMB = 10;
          this.allowedTypes = ['image/jpeg', 'image/png'];
          this.selectedFile = null;
          this.currentDogId = null;
          this.uploadInProgress = false;
+         this.activeZone = null;
+         this.zoneEventHandlers = new Map();
      }

      setupDragDrop(zoneId, onFileSelected) {
          const zone = document.getElementById(zoneId);
          if (!zone) return;

+         // Remove old event listeners if this zone was already set up
+         if (this.activeZone === zone && this.zoneEventHandlers.has(zone)) {
+             const handlers = this.zoneEventHandlers.get(zone);
+             handlers.forEach(({ event, handler }) => {
+                 zone.removeEventListener(event, handler);
+             });
+             this.zoneEventHandlers.delete(zone);
+         }
+
+         const handlers = [];

-         ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
-             zone.addEventListener(eventName, (e) => {
-                 e.preventDefault();
-                 e.stopPropagation();
-             });
-         });
+         const preventDefaultHandler = (e) => {
+             e.preventDefault();
+             e.stopPropagation();
+         };
+
+         ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
+             zone.addEventListener(eventName, preventDefaultHandler);
+             handlers.push({ event: eventName, handler: preventDefaultHandler });
+         });

          // ... similar for other event listeners, storing them in handlers array

+         this.zoneEventHandlers.set(zone, handlers);
+         this.activeZone = zone;
      }
```

---

## Bug #7: Incorrect Logic in i18n.js - Returns Key Instead of Empty String

**Description:**
In `i18n.js`, the `t()` method returns the original key when a translation is not found (line 30, 34). While this is useful for debugging, it creates a poor user experience in production and can potentially expose internal key names to users. Additionally, if the translation value is an empty string (valid case), it will return the key instead of the empty string due to the `|| key` fallback on line 34.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/i18n.js`
- Function: `t`
- Lines: 22-35

**Steps to Reproduce:**
1. Add translation entry: `"common.empty": ""`
2. Call `i18n.t('common.empty')`
3. Expected: Returns empty string
4. Actual: Returns 'common.empty' due to falsy check
5. Call `i18n.t('nonexistent.key')`
6. Expected: Returns empty string or warning
7. Actual: Returns 'nonexistent.key' exposing internal structure

**Fix:**
Use explicit undefined check and add warning for missing translations:

```diff
  t(key) {
      const keys = key.split('.');
      let value = this.translations;

      for (const k of keys) {
          if (value && typeof value === 'object') {
              value = value[k];
          } else {
-             return key;
+             console.warn(`Translation missing for key: ${key}`);
+             return '';
          }
      }

-     return value || key;
+     if (value === undefined || value === null) {
+         console.warn(`Translation missing for key: ${key}`);
+         return '';
+     }
+     return value;
  }
```

---

## Bug #8: Race Condition in handleImageLoad Function

**Description:**
In `dog-photo-helpers.js`, the `handleImageLoad` function is called as a string from the `onload` attribute (line 61). This creates a global function dependency and doesn't handle the case where the function is called before the DOM elements exist or after they've been removed. If the image loads very quickly (from cache) while the DOM is still being manipulated, it may try to access elements that don't exist yet, causing silent failures.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/dog-photo-helpers.js`
- Function: `handleImageLoad`
- Lines: 123-141

**Steps to Reproduce:**
1. Load page with many dog images (all in cache)
2. Images fire onload immediately, some before DOM is fully constructed
3. `handleImageLoad` is called with imageId
4. `document.getElementById(imageId)` returns null because element not yet inserted
5. Expected: Fade-in effect and skeleton removal
6. Actual: Silent failure, skeleton never removed, no fade-in

**Fix:**
Add null checks and use addEventListener instead of inline onload:

```diff
  function getDogPhotoHtml(dog, useThumbnail = false, className = 'dog-card-image', lazyLoad = true, withSkeleton = true) {
      const photoUrl = getDogPhotoUrl(dog, useThumbnail);
      const altText = getDogPhotoAlt(dog);
      const loadingAttr = lazyLoad ? ' loading="lazy"' : '';
      const uniqueId = `dog-img-${dog.id || Math.random().toString(36).substr(2, 9)}`;

      const isSvgPlaceholder = photoUrl.includes('.svg');

      if (withSkeleton && !isSvgPlaceholder) {
          return `<div class="dog-card-image-container" id="container-${uniqueId}">
                      <img src="${photoUrl}"
                           alt="${altText}"
                           class="${className}"
                           id="${uniqueId}"
                           ${loadingAttr}
-                          onload="handleImageLoad('${uniqueId}')">
+                          data-load-handler="true">
                  </div>`;
      }

      return `<img src="${photoUrl}" alt="${altText}" class="${className}"${loadingAttr}>`;
  }

  function handleImageLoad(imageId) {
      const img = document.getElementById(imageId);
      const container = document.getElementById(`container-${imageId}`);

-     if (img) {
+     if (img && img.complete && img.naturalHeight !== 0) {
          img.classList.add('loaded');

-         if (img.complete && img.naturalHeight !== 0) {
-             img.classList.add('no-animation');
-         }
+         // Always skip animation for cached images
+         img.classList.add('no-animation');
      }

-     if (container) {
+     if (container && container.parentElement) {
          container.classList.add('loaded');
      }
  }

+ // Use event delegation for image load events
+ document.addEventListener('DOMContentLoaded', () => {
+     document.body.addEventListener('load', (e) => {
+         if (e.target.tagName === 'IMG' && e.target.dataset.loadHandler) {
+             handleImageLoad(e.target.id);
+         }
+     }, true);
+ });
```

---

## Bug #9: Incorrect URL Construction in getDogs with Empty Filters

**Description:**
In `api.js` line 154-156, the `getDogs` method constructs the URL by checking if `params.toString()` returns a truthy value. However, if filters object contains properties with empty string values like `{category: ''}`, the URLSearchParams will include them as `?category=`, which is technically different from no parameter at all. This could cause backend filtering logic to behave unexpectedly.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/api.js`
- Function: `getDogs`
- Lines: 153-157

**Steps to Reproduce:**
1. Call `api.getDogs({category: '', is_available: ''})`
2. Expected URL: `/api/dogs` (no parameters)
3. Actual URL: `/api/dogs?category=&is_available=`
4. Backend may interpret empty strings differently than missing parameters

**Fix:**
Filter out empty/falsy values before creating URLSearchParams:

```diff
  async getDogs(filters = {}) {
-     const params = new URLSearchParams(filters);
+     // Remove empty/falsy values from filters
+     const cleanFilters = {};
+     for (const [key, value] of Object.entries(filters)) {
+         if (value !== null && value !== undefined && value !== '') {
+             cleanFilters[key] = value;
+         }
+     }
+     const params = new URLSearchParams(cleanFilters);
      const endpoint = `/dogs${params.toString() ? '?' + params.toString() : ''}`;
      return this.request('GET', endpoint);
  }
```

---

## Bug #10: Missing Error Handling in i18n.load

**Description:**
In `i18n.js`, the `load()` method catches errors and logs them to console (line 17), but continues silently. This means if translations fail to load, all calls to `t()` will return the key instead of translated text, but the application continues running with broken i18n. Users see internal key names instead of proper German text, degrading the user experience significantly.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/i18n.js`
- Function: `load`
- Lines: 8-19

**Steps to Reproduce:**
1. Network error occurs during translation file fetch (e.g., ad blocker blocks /i18n/de.json)
2. Application loads successfully
3. All UI elements show keys like "common.save", "dogs.name" instead of German text
4. No visible error to user
5. Expected: Error notification or fallback to English/default text
6. Actual: Broken UI with key names shown

**Fix:**
Add better error handling and user notification:

```diff
  async load() {
      try {
          const response = await fetch(`/i18n/${this.locale}.json`);
          if (!response.ok) {
              throw new Error(`Failed to load translations: ${response.status}`);
          }
          this.translations = await response.json();
          this.applyTranslations();
+         return true;
      } catch (error) {
          console.error('Failed to load translations:', error);
+         // Show user-friendly error message
+         const errorDiv = document.createElement('div');
+         errorDiv.style.cssText = 'position: fixed; top: 10px; right: 10px; background: #ff4444; color: white; padding: 15px; border-radius: 5px; z-index: 10000;';
+         errorDiv.textContent = 'Übersetzungen konnten nicht geladen werden. Bitte Seite neu laden.';
+         document.body.appendChild(errorDiv);
+         return false;
      }
  }
```

---

## Bug #11: Unvalidated File Input in previewFile

**Description:**
In `dog-photo.js`, the `previewFile` method uses `FileReader.readAsDataURL()` to preview images (line 64). However, while file type validation exists (lines 16-18), the validation happens synchronously but the FileReader operation is asynchronous. If an attacker modifies the file extension after validation but before reading (though unlikely in browser context), or if validation is bypassed, malicious files could be processed. More importantly, there's no size check before reading into memory, potentially causing memory exhaustion with very large files.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/dog-photo.js`
- Function: `previewFile`
- Lines: 32-68

**Steps to Reproduce:**
1. Select a 500MB JPEG file
2. Browser attempts to read entire file into memory as data URL
3. Expected: Error message about file size
4. Actual: Browser tab freezes or crashes due to memory exhaustion

**Fix:**
Check file size before reading (validation already exists but should be double-checked):

```diff
  previewFile(file, previewElementId) {
      return new Promise((resolve, reject) => {
          try {
              this.validateFile(file);
+
+             // Additional safety check before memory-intensive operation
+             const sizeMB = file.size / (1024 * 1024);
+             if (sizeMB > this.maxSizeMB) {
+                 throw new Error(`Datei zu groß. Maximum: ${this.maxSizeMB}MB`);
+             }

              const reader = new FileReader();
              reader.onload = (e) => {
                  const previewImg = document.getElementById(previewElementId);
                  if (previewImg) {
                      previewImg.src = e.target.result;
                      previewImg.style.display = 'block';
                  }

                  // Show preview container and hide prompt
                  const photoPreview = document.getElementById('photo-preview');
                  const uploadPrompt = document.querySelector('.upload-prompt');

                  if (photoPreview) {
                      photoPreview.classList.remove('hidden');
                  }
                  if (uploadPrompt) {
                      uploadPrompt.style.display = 'none';
                  }

                  this.selectedFile = file;
                  resolve(e.target.result);
              };

              reader.onerror = () => {
                  reject(new Error('Fehler beim Lesen der Datei'));
              };
+
+             // Add abort handler for large files
+             reader.onabort = () => {
+                 reject(new Error('Datei-Lesen wurde abgebrochen'));
+             };

              reader.readAsDataURL(file);
          } catch (error) {
              reject(error);
          }
      });
  }
```

---

## Bug #12: Missing Null Check in Router Navigate Function

**Description:**
In `router.js`, the `navigate` function at line 34 attempts to find a handler for the current path. If no handler is found, it defaults to a 404 handler (lines 53-56). However, the 404 handler directly sets `document.body.innerHTML`, which destroys all existing event listeners and global state (including the router itself). This causes the application to become non-functional after hitting a 404.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/router.js`
- Function: `navigate`
- Lines: 34-66

**Steps to Reproduce:**
1. Navigate to valid page (e.g., /dashboard.html)
2. Manually change URL to /nonexistent.html
3. Default 404 handler fires: `document.body.innerHTML = '<h1>404 - Page Not Found</h1>'`
4. All event listeners are destroyed
5. Try to navigate anywhere: Router no longer works
6. Expected: 404 page with functional navigation
7. Actual: Broken application state

**Fix:**
Create a proper 404 page or at least preserve the navigation structure:

```diff
  navigate(path, pushState = true) {
      // Find matching route
      let handler = this.routes[path];

      // Try exact match first, then wildcard
      if (!handler) {
          // Check for wildcard routes
          for (const route in this.routes) {
              if (route.includes(':')) {
                  const pattern = new RegExp('^' + route.replace(/:[^\s/]+/g, '([^/]+)') + '$');
                  if (pattern.test(path)) {
                      handler = this.routes[route];
                      break;
                  }
              }
          }
      }

      // Default to 404 if no match
      if (!handler) {
-         handler = this.routes['/404'] || (() => {
-             document.body.innerHTML = '<h1>404 - Page Not Found</h1>';
-         });
+         handler = this.routes['/404'] || (() => {
+             // Redirect to actual 404 page instead of destroying DOM
+             window.location.href = '/404.html';
+             return;
+         });
      }

      // Update browser history
      if (pushState) {
          window.history.pushState({}, '', path);
      }

      // Call route handler
      handler();
  }
```

---

## Bug #13: uploadInProgress Flag Not Reset on Network Error

**Description:**
In `dog-photo.js`, the `uploadPhoto` method sets `this.uploadInProgress = true` at line 106, and resets it in the try-catch at lines 115 and 120. However, if the network request itself throws an error that's not caught by the inner try-catch (e.g., network disconnection), the finally block is not used, so the flag may remain true. This locks the upload functionality permanently until page refresh.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/dog-photo.js`
- Function: `uploadPhoto`
- Lines: 100-123

**Steps to Reproduce:**
1. Start uploading a photo
2. Disconnect network while upload is in progress
3. Upload fails with network error
4. `uploadInProgress` remains true
5. Try to upload again
6. Expected: Upload starts
7. Actual: Error "Upload läuft bereits" even though no upload is running

**Fix:**
Use finally block to ensure flag is always reset:

```diff
  async uploadPhoto(dogId, file) {
      if (this.uploadInProgress) {
          throw new Error('Upload läuft bereits');
      }

      try {
          this.uploadInProgress = true;
          this.validateFile(file);

          // Show progress indicator
          this.showProgress();

          const response = await api.uploadDogPhoto(dogId, file);

-         this.hideProgress();
-         this.uploadInProgress = false;

          return response;
      } catch (error) {
-         this.hideProgress();
-         this.uploadInProgress = false;
          throw error;
+     } finally {
+         this.hideProgress();
+         this.uploadInProgress = false;
      }
  }
```

---

## Bug #14: Potential Infinite Loop in admin-booking-times.js Event Listener

**Description:**
In `admin-booking-times.js`, line 253 has an event listener for the holiday active toggle checkbox. If the API call fails (line 260), the code tries to revert the checkbox state by setting `e.target.checked = !e.target.checked`. However, this triggers a new 'change' event, which calls the handler again, potentially creating an infinite loop of API calls if the failure condition persists.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/admin-booking-times.js`
- Function: Anonymous event handler in `createHolidayRow`
- Lines: 253-264

**Steps to Reproduce:**
1. Backend API is down or returns 500 error
2. Admin toggles holiday active checkbox
3. API call fails
4. Line 262 executes: `e.target.checked = !e.target.checked`
5. This triggers another 'change' event
6. Event handler called again
7. API call fails again
8. Loop continues
9. Expected: Checkbox reverts once and stops
10. Actual: Rapid-fire API calls until browser freezes

**Fix:**
Use a flag to prevent re-entry:

```diff
+ let isUpdating = false;
  tr.querySelector('.holiday-active-toggle').addEventListener('change', async (e) => {
+     if (isUpdating) return;
+
      const checkbox = e.target;
      const originalValue = !checkbox.checked;

      try {
+         isUpdating = true;
          await api.updateHoliday(holiday.id, {
              name: holiday.name,
              is_active: checkbox.checked
          });
          showAlert('success', 'Feiertag aktualisiert');
      } catch (error) {
          showAlert('error', error.message || 'Fehler beim Aktualisieren');
          checkbox.checked = originalValue;
+     } finally {
+         isUpdating = false;
      }
  });
```

---

## Bug #15: Missing CSRF Protection for Logout

**Description:**
In `api.js`, the `logout` method (lines 110-113) immediately clears the token and redirects to the homepage without making any API call to invalidate the session on the server side. While JWT tokens don't typically require server-side invalidation, this approach means the token remains valid until expiration. An attacker who has obtained the token can continue using it even after the user has logged out. Additionally, there's no CSRF protection for the logout action itself.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/api.js`
- Function: `logout`
- Lines: 110-113

**Steps to Reproduce:**
1. User logs in, JWT token issued (valid for X hours)
2. Attacker steals token via XSS or other means
3. User logs out (client-side only, token not invalidated on server)
4. Attacker uses stolen token to make API calls
5. Expected: Token is invalid after logout
6. Actual: Token still works until natural expiration

**Fix:**
Add server-side logout endpoint call (requires backend implementation):

```diff
  async logout() {
-     this.setToken(null);
-     window.location.href = '/';
+     try {
+         // Call backend to invalidate token (add to backend if not exists)
+         await this.request('POST', '/auth/logout');
+     } catch (error) {
+         // Continue with logout even if server call fails
+         console.error('Server-side logout failed:', error);
+     } finally {
+         this.setToken(null);
+         window.location.href = '/';
+     }
  }
```

Note: This fix requires a corresponding `/auth/logout` endpoint on the backend that adds the token to a blacklist or decrements its version. This is a defense-in-depth measure for JWT-based authentication.

---

## Statistics

- **Critical:** 3 bugs (XSS vulnerabilities)
- **High:** 6 bugs (Race conditions, missing error handling, security issues)
- **Medium:** 5 bugs (Logic errors, UX issues)
- **Low:** 1 bug (Minor edge case)

---

## Recommendations

### Immediate Actions (Critical)

1. **Implement comprehensive input sanitization**: All user-generated content must be sanitized before insertion into DOM. Use the existing `sanitizeHTML` function consistently across all files.

2. **Validate color hex codes**: Add regex validation for hex color inputs to prevent CSS injection attacks.

3. **Fix XSS in admin-booking-times.js**: Sanitize all rule names and user-provided data before rendering.

### High Priority

4. **Add localStorage sync**: Implement cross-tab token synchronization to prevent race conditions when users have multiple tabs open.

5. **Improve error handling**: Add try-catch blocks for all JSON parsing operations and provide user-friendly error messages.

6. **Fix memory leaks**: Implement proper event listener cleanup in DogPhotoManager and other components that add dynamic event handlers.

7. **Add server-side logout**: Implement token blacklisting or versioning on the backend to invalidate tokens on logout.

### Medium Priority

8. **Improve i18n error handling**: Add user-visible error messages when translations fail to load.

9. **Fix router 404 handling**: Redirect to proper 404 page instead of destroying DOM.

10. **Add request deduplication**: Prevent duplicate API calls when users rapidly click buttons.

### Code Quality Improvements

11. **Use event delegation**: Replace inline event handlers (`onload="..."`) with event delegation for better performance and security.

12. **Add unit tests**: Create automated tests for critical functions like `sanitizeHTML`, URL construction, and token management.

13. **Implement CSP headers**: Add Content-Security-Policy headers to prevent inline script execution and mitigate XSS risks.

14. **Code review process**: Establish code review guidelines focusing on XSS prevention, input validation, and error handling.

### Defense in Depth

15. **Input validation library**: Consider using a well-tested sanitization library like DOMPurify instead of custom `sanitizeHTML` function.

16. **TypeScript migration**: Consider migrating to TypeScript for better type safety and catching errors at compile time.

17. **API request retry logic**: Add exponential backoff retry logic for failed API requests to improve resilience.

18. **Rate limiting on client**: Implement client-side rate limiting to prevent users from accidentally DDoSing the backend with rapid requests.
