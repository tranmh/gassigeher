# Bug Report: frontend/js

**Analysis Date:** 2025-12-01
**Directory Analyzed:** `/home/jaco/Git-clones/gassigeher/frontend/js`
**Files Analyzed:** 7 files
**Bugs Found:** 18 bugs

**VERIFICATION DATE:** 2025-12-01
**Actual Location:** `/home/jaco/Git-clones/gassigeher/internal/static/frontend/js`

---

## Summary

Analysis of the frontend JavaScript directory revealed **18 functional bugs** across multiple categories. The most critical issues include:

- **7 Critical XSS vulnerabilities** from unsanitized user data in innerHTML assignments
- **3 High severity** authentication and error handling bugs
- **5 Medium severity** race conditions and logic errors
- **3 Low severity** edge case handling issues

The most concerning pattern is widespread use of template literals with `innerHTML` for rendering user-controlled data (names, emails, notes, reasons) without sanitization. This creates multiple XSS attack vectors throughout the application.

---

## Bugs

## Bug #1: XSS Vulnerability in User Name Rendering

**STATUS: CODE LOCATION VERIFIED - HTML FILES, NOT JS FILES**

**Note:** This bug is in HTML files (admin-users.html, admin-reactivation-requests.html, admin-experience-requests.html, dashboard.html), not in the frontend/js directory. The bug report category is "frontend/js" but this vulnerability exists in the HTML templates.

**Severity:** Critical

**Description:**
Multiple HTML files render user names directly into innerHTML without sanitization. An attacker can create an account with a malicious name containing JavaScript code (e.g., `<img src=x onerror=alert(document.cookie)>`) which will execute when admins view the user list or when the name appears in bookings/requests.

**Location:**
- Files: Multiple HTML files (admin-users.html, admin-reactivation-requests.html, admin-experience-requests.html, dashboard.html)
- Pattern: `${user.name}`, `${booking.dog.name}`, `${request.user.name}`
- **Verified Lines:** admin-users.html:155, 161, 174 (verified 2025-12-01)
- Original reported lines: admin-users.html:143, admin-reactivation-requests.html:100, admin-experience-requests.html:105, dashboard.html:175, 228

**Steps to Reproduce:**
1. Register new account with name: `<img src=x onerror=alert('XSS')>`
2. Create a booking or experience request
3. Admin opens admin-users.html or admin-experience-requests.html
4. Expected: Name displayed safely as text
5. Actual: JavaScript executes in admin's browser

**Impact:**
- Session hijacking (steal admin JWT token from localStorage)
- Account takeover
- Malicious actions performed as admin
- Data exfiltration

**Fix:**
Create a sanitization helper function and use it for all user-controlled content:

```diff
+// Add to a new file: /frontend/js/sanitize.js
+function sanitizeHTML(str) {
+    if (!str) return '';
+    const div = document.createElement('div');
+    div.textContent = str;
+    return div.innerHTML;
+}
+window.sanitizeHTML = sanitizeHTML;

// In admin-users.html and other files:
-<h4 style="margin: 0;">${user.name}</h4>
+<h4 style="margin: 0;">${sanitizeHTML(user.name)}</h4>

-<strong>E-Mail:</strong> ${user.email || 'N/A'}
+<strong>E-Mail:</strong> ${sanitizeHTML(user.email) || 'N/A'}
```

Alternatively, use `textContent` instead of `innerHTML` where possible, or create elements programmatically:

```javascript
const h4 = document.createElement('h4');
h4.textContent = user.name;  // Safe - treats as text
container.appendChild(h4);
```

---

## Bug #2: XSS Vulnerability in Dog Names and Breeds

**STATUS: CODE LOCATION VERIFIED - HTML FILES, NOT JS FILES**

**Note:** This bug is in HTML files (dogs.html, admin-dogs.html, dashboard.html, calendar.html), not in the frontend/js directory.

**Severity:** Critical

**Description:**
Dog names and breeds are rendered directly into innerHTML without sanitization. Admin can create dogs with malicious names/breeds that execute JavaScript when users browse the dog listing or view bookings.

**Location:**
- Files: dogs.html, admin-dogs.html, dashboard.html, calendar.html
- **Verified Lines:** dogs.html:352-353 (verified 2025-12-01)
- Original reported lines: dogs.html:353-354, admin-dogs.html:202-203, dashboard.html:175, 228
- Functions: Rendering dog cards, booking details, calendar cells

**Steps to Reproduce:**
1. Admin creates dog with name: `Fluffy<script>alert('XSS')</script>`
2. User visits dogs.html page
3. Expected: Dog name displayed as text
4. Actual: JavaScript executes in user's browser

**Impact:**
- All users viewing dogs page affected
- Can steal user tokens and perform actions as that user
- Can redirect to phishing pages
- Can modify page content to trick users

**Fix:**
Same as Bug #1 - sanitize all dog.name, dog.breed values:

```diff
-<h3 class="dog-card-title">${dog.name}</h3>
+<h3 class="dog-card-title">${sanitizeHTML(dog.name)}</h3>

-<p class="dog-card-info">${dog.breed} • ${getSizeLabel(dog.size)} • ${dog.age} Jahre</p>
+<p class="dog-card-info">${sanitizeHTML(dog.breed)} • ${getSizeLabel(dog.size)} • ${dog.age} Jahre</p>
```

---

## Bug #3: XSS Vulnerability in Error Messages

**STATUS: CODE LOCATION VERIFIED - HTML FILES, NOT JS FILES**

**Note:** This bug is in HTML files with showAlert() function definitions, not in the frontend/js directory.

**Severity:** Critical

**Description:**
Error messages from API responses are displayed directly in alert containers using innerHTML. If the backend returns user input in error messages (e.g., "User 'X' not found"), attackers can inject XSS through error messages.

**Location:**
- Files: Multiple HTML files
- Function: `showAlert(type, message)`
- Pattern: `container.innerHTML = \`<div class="alert alert-${type}">${message}</div>\``
- Lines: dashboard.html:271, admin-users.html:259, admin-dogs.html:436, profile.html:412, calendar.html:673

**Steps to Reproduce:**
1. Submit form with malicious input that triggers server error containing that input
2. Server returns: `{error: "Invalid name: <img src=x onerror=alert('XSS')>"}`
3. Frontend displays error via showAlert()
4. Expected: Error message shown safely
5. Actual: JavaScript executes

**Impact:**
- XSS on error conditions
- Can be triggered via API requests
- Affects all pages using showAlert()

**Fix:**
Sanitize error messages in showAlert function:

```diff
function showAlert(type, message) {
    const container = document.getElementById('alert-container');
-   container.innerHTML = `<div class="alert alert-${type}">${message}</div>`;
+   const div = document.createElement('div');
+   div.className = `alert alert-${type}`;
+   div.textContent = message;  // Safe - treats as text
+   container.innerHTML = '';
+   container.appendChild(div);
    setTimeout(() => container.innerHTML = '', 5000);
}
```

---

## Bug #4: XSS Vulnerability in Deactivation/Cancellation Reasons

**STATUS: CODE LOCATION VERIFIED - HTML FILES, NOT JS FILES**

**Note:** This bug is in HTML files (admin-users.html), not in the frontend/js directory.

**Severity:** Critical

**Description:**
User-provided deactivation reasons and booking cancellation reasons are rendered directly into innerHTML without sanitization.

**Location:**
- File: admin-users.html
- **Verified Lines:** 174 (verified 2025-12-01)
- Original reported lines: 160-163
- Code: `<strong>Deaktivierungsgrund:</strong> ${user.deactivation_reason}`

**Steps to Reproduce:**
1. Admin deactivates user with reason: `<img src=x onerror=alert('XSS')>`
2. Admin views user list later
3. Expected: Reason displayed safely
4. Actual: JavaScript executes

**Impact:**
- Persistent XSS (stored in database)
- Affects admin interface
- Can persist across sessions

**Fix:**
Sanitize all user-provided reason fields:

```diff
-<strong>Deaktivierungsgrund:</strong> ${user.deactivation_reason}
+<strong>Deaktivierungsgrund:</strong> ${sanitizeHTML(user.deactivation_reason)}
```

---

## Bug #5: XSS Vulnerability in Profile Photo Path

**STATUS: CODE LOCATION VERIFIED - HTML FILES, NOT JS FILES**

**Note:** This bug is in HTML files (dashboard.html, profile.html, calendar.html, dogs.html, index.html), not in the frontend/js directory.

**Severity:** Critical

**Description:**
Profile photo paths from the database are inserted directly into innerHTML without validation. While paths should be controlled by the server, if there's a bug in the upload handler, an attacker could inject a malicious "path" that contains JavaScript.

**Location:**
- Files: dashboard.html, profile.html, calendar.html, dogs.html, index.html
- Lines: dashboard.html:125, profile.html:173, 186, calendar.html:662, dogs.html:224
- Pattern: `headerPhoto.innerHTML = \`<img src="/uploads/${currentUser.profile_photo}" ...\``

**Steps to Reproduce:**
1. Exploit photo upload vulnerability to set profile_photo field to: `x" onerror="alert('XSS')"`
2. Login and navigate to any page showing profile photo
3. Expected: Safe image rendering or error
4. Actual: JavaScript executes via onerror event

**Impact:**
- XSS if upload validation is bypassed
- Defense-in-depth issue

**Fix:**
Use proper DOM manipulation instead of innerHTML:

```diff
if (currentUser && currentUser.profile_photo) {
    const headerPhoto = document.getElementById('header-photo');
-   headerPhoto.innerHTML = `<img src="/uploads/${currentUser.profile_photo}" style="width: 100%; height: 100%; object-fit: cover;" alt="Profile">`;
+   const img = document.createElement('img');
+   img.src = `/uploads/${sanitizeHTML(currentUser.profile_photo)}`;
+   img.style.cssText = 'width: 100%; height: 100%; object-fit: cover;';
+   img.alt = 'Profile';
+   headerPhoto.innerHTML = '';
+   headerPhoto.appendChild(img);
}
```

---

## Bug #6: SQL Injection-like Vulnerability in onclick Attributes

**STATUS: CODE LOCATION VERIFIED - HTML FILES, NOT JS FILES**

**Note:** This bug is in HTML files (admin-users.html), not in the frontend/js directory.

**Severity:** High

**Description:**
User names are embedded directly into onclick attributes with only basic escaping (`replace(/'/g, "\\'")`). This escaping is insufficient and can be bypassed with various encoding techniques or quote characters.

**Location:**
- File: admin-users.html
- Lines: 177, 179
- Code: `onclick="promoteToAdmin(${user.id}, '${user.name.replace(/'/g, "\\'")}')"`

**Steps to Reproduce:**
1. Register user with name: `test'); alert('XSS'); //`
2. Admin views user list
3. Click "Zu Admin ernennen" button
4. Expected: Function called with name as parameter
5. Actual: Additional JavaScript executes

**Impact:**
- Code injection via onclick handlers
- Can execute arbitrary JavaScript
- Affects admin actions

**Fix:**
Use data attributes and event listeners instead of inline onclick:

```diff
-<button class="btn-promote" onclick="promoteToAdmin(${user.id}, '${user.name.replace(/'/g, "\\'")}')">Zu Admin ernennen</button>
+<button class="btn-promote" data-user-id="${user.id}" data-user-name="${sanitizeHTML(user.name)}">Zu Admin ernennen</button>

// Add event delegation:
document.querySelector('#users-list').addEventListener('click', (e) => {
    if (e.target.classList.contains('btn-promote')) {
        const userId = parseInt(e.target.dataset.userId);
        const userName = e.target.dataset.userName;
        promoteToAdmin(userId, userName);
    }
});
```

---

## Bug #7: XSS in Calendar onclick Attributes

**STATUS: CODE LOCATION VERIFIED - HTML FILES, NOT JS FILES**

**Note:** This bug is in HTML files (calendar.html), not in the frontend/js directory.

**Severity:** Critical

**Description:**
Dog names and dates are inserted into onclick attributes without proper escaping. An attacker with admin access can create dogs with malicious names that execute JavaScript when users click calendar cells.

**Location:**
- File: calendar.html
- Line: 591
- Code: `onclick="quickBook(${dog.id}, '${date}', ${!morningBooked}, ${!eveningBooked})" title="Klicken zum Buchen: ${dog.name} am ${formatDateGerman(date)}"`

**Steps to Reproduce:**
1. Admin creates dog with name: `", true, true); alert('XSS'); //`
2. User views calendar
3. User clicks on booking cell for that dog
4. Expected: Booking modal opens
5. Actual: JavaScript executes before modal opens

**Impact:**
- XSS via onclick attributes
- Affects all calendar users
- Can hijack booking actions

**Fix:**
Use data attributes instead of inline onclick:

```diff
-onclick="quickBook(${dog.id}, '${date}', ${!morningBooked}, ${!eveningBooked})"
+data-dog-id="${dog.id}" data-date="${date}" data-morning="${!morningBooked}" data-evening="${!eveningBooked}"

// Add event delegation:
document.querySelector('#calendar-grid').addEventListener('click', (e) => {
    const cell = e.target.closest('[data-dog-id]');
    if (cell) {
        const dogId = parseInt(cell.dataset.dogId);
        const date = cell.dataset.date;
        const morningAvailable = cell.dataset.morning === 'true';
        const eveningAvailable = cell.dataset.evening === 'true';
        quickBook(dogId, date, morningAvailable, eveningAvailable);
    }
});
```

---

## Bug #8: Race Condition in Token Refresh

**Severity:** High

**Description:**
The API class stores token in both instance variable and localStorage, but doesn't handle concurrent requests properly. If two API calls happen simultaneously after token refresh, one might use the old token.

**Location:**
- File: frontend/js/api.js (verified: `/home/jaco/Git-clones/gassigeher/internal/static/frontend/js/api.js`)
- **Verified Lines:** 5, 10-15, 34-35 (verified 2025-12-01)
- Function: `setToken()`, `request()`

**Verification Status:** ✅ **UNCHANGED - Bug still present**
- Line 5: `this.token = localStorage.getItem('gassigeher_token');`
- Lines 10-15: `setToken()` method updates both `this.token` and localStorage
- Lines 34-35: `request()` uses `this.token` instead of reading from localStorage

**Steps to Reproduce:**
1. Login to get token
2. Make two simultaneous API calls (e.g., getDogs() and getBookings())
3. If token expires between these calls, one might fail
4. Race condition: token updated in localStorage but not in `this.token` before second request uses it

**Impact:**
- Intermittent authentication failures
- Poor user experience (random logouts)
- Data inconsistency

**Fix:**
Always read token from localStorage in request() method:

```diff
async request(method, endpoint, data = null) {
    const headers = {
        'Content-Type': 'application/json',
    };

-   if (this.token) {
-       headers['Authorization'] = `Bearer ${this.token}`;
+   const token = localStorage.getItem('gassigeher_token');
+   if (token) {
+       headers['Authorization'] = `Bearer ${token}`;
    }

    // ... rest of method
}
```

Or use a mutex for token operations:

```javascript
constructor() {
    this.baseURL = '/api';
    this.tokenLock = Promise.resolve();
}

async getToken() {
    return localStorage.getItem('gassigeher_token');
}

async request(method, endpoint, data = null) {
    await this.tokenLock;
    const token = await this.getToken();
    // ... rest of method
}
```

---

## Bug #9: Missing Error Handling for JSON Parse Failures

**Severity:** High

**Description:**
In the `request()` method, `await response.json()` is called without try-catch for the JSON parsing. If the server returns non-JSON response (e.g., HTML error page from nginx), the promise rejects and the error is not handled properly.

**Location:**
- File: frontend/js/api.js (verified: `/home/jaco/Git-clones/gassigeher/internal/static/frontend/js/api.js`)
- **Verified Lines:** 47-62 (verified 2025-12-01), specifically line 49
- Function: `request()`

**Verification Status:** ✅ **UNCHANGED - Bug still present**
- Line 47: `try {` - outer try block exists
- Line 49: `const responseData = await response.json();` - no inner try-catch for JSON parsing
- Lines 60-62: catch block only handles outer errors

**Steps to Reproduce:**
1. Server returns 500 error with HTML error page
2. Frontend calls `await response.json()`
3. Expected: Proper error handling with user-friendly message
4. Actual: Unhandled promise rejection, console error

**Impact:**
- Application crashes on certain server errors
- Poor error messages to users
- Difficult debugging

**Fix:**
Add proper JSON parsing error handling:

```diff
async request(method, endpoint, data = null) {
    // ... setup code ...

    try {
        const response = await fetch(`${this.baseURL}${endpoint}`, options);
-       const responseData = await response.json();
+
+       let responseData;
+       try {
+           responseData = await response.json();
+       } catch (jsonError) {
+           // Server returned non-JSON response (HTML error page, etc)
+           const errorText = await response.text();
+           throw new Error(`Server error (${response.status}): Unable to parse response`);
+       }

        if (!response.ok) {
            const error = new Error(responseData.error || 'Request failed');
            error.status = response.status;
            error.data = responseData;
            throw error;
        }

        return responseData;
    } catch (error) {
+       // Add more context to network errors
+       if (error.name === 'TypeError' && error.message === 'Failed to fetch') {
+           error.message = 'Netzwerkfehler: Server nicht erreichbar';
+       }
        throw error;
    }
}
```

---

## Bug #10: Unclosed Event Listeners in Dog Photo Manager

**Severity:** Medium

**Description:**
The `setupDragDrop()` method adds event listeners to the drop zone but never removes them. If the function is called multiple times (e.g., when reopening modal), duplicate listeners accumulate, causing multiple callbacks on each drag/drop event.

**Location:**
- File: frontend/js/dog-photo.js (verified: `/home/jaco/Git-clones/gassigeher/internal/static/frontend/js/dog-photo.js`)
- **Verified Lines:** 139-179 (verified 2025-12-01)
- Function: `setupDragDrop()`

**Verification Status:** ✅ **UNCHANGED - Bug still present**
- Lines 144-149: Event listeners added without cleanup
- Lines 152-162: More event listeners added
- Lines 165-178: Drop and click handlers added
- No AbortController or cleanup mechanism present

**Steps to Reproduce:**
1. Open dog creation modal
2. Close modal
3. Open modal again
4. Repeat 3-4 times
5. Try dragging a file
6. Expected: Preview appears once
7. Actual: Preview callback fires multiple times (memory leak)

**Impact:**
- Memory leak from unclosed listeners
- Multiple file validations on single drop
- Performance degradation
- Potential file upload duplicates

**Fix:**
Remove old listeners before adding new ones, or use AbortController:

```diff
class DogPhotoManager {
    constructor() {
        // ... existing properties ...
+       this.dragDropController = null;
    }

    setupDragDrop(zoneId, onFileSelected) {
        const zone = document.getElementById(zoneId);
        if (!zone) return;

+       // Abort previous listeners
+       if (this.dragDropController) {
+           this.dragDropController.abort();
+       }
+       this.dragDropController = new AbortController();
+       const signal = this.dragDropController.signal;

        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
-           zone.addEventListener(eventName, (e) => {
+           zone.addEventListener(eventName, (e) => {
                e.preventDefault();
                e.stopPropagation();
-           });
+           }, { signal });
        });

        // ... rest of listeners with { signal } option
    }

+   cleanup() {
+       if (this.dragDropController) {
+           this.dragDropController.abort();
+       }
+   }
}
```

---

## Bug #11: Race Condition in Upload Progress Display

**Severity:** Medium

**Description:**
The `uploadInProgress` flag in DogPhotoManager is not atomic. If two uploads are triggered rapidly (e.g., double-click), both can pass the initial check and start uploading, causing duplicate uploads and progress indicator confusion.

**Location:**
- File: frontend/js/dog-photo.js (verified: `/home/jaco/Git-clones/gassigeher/internal/static/frontend/js/dog-photo.js`)
- **Verified Lines:** 100-123 (verified 2025-12-01)
- Function: `uploadPhoto()`

**Verification Status:** ✅ **UNCHANGED - Bug still present**
- Line 101: Check `if (this.uploadInProgress)` happens first
- Line 106: `this.uploadInProgress = true` set AFTER async operations begin
- Lines 101-103: Early return doesn't prevent race condition
- Issue: Between line 101 check and line 106 assignment, another call can pass the check

**Steps to Reproduce:**
1. Select a photo
2. Double-click upload button rapidly
3. Expected: Second click ignored, single upload
4. Actual: Two uploads start simultaneously

**Impact:**
- Duplicate uploads to server
- Progress indicator doesn't hide properly
- Confusing user experience
- Wasted bandwidth

**Fix:**
Set flag before async operation and use Promise to prevent re-entry:

```diff
async uploadPhoto(dogId, file) {
+   if (this.uploadInProgress) {
+       throw new Error('Upload läuft bereits');
+   }
+
+   this.uploadInProgress = true;
+
-   if (this.uploadInProgress) {
-       throw new Error('Upload läuft bereits');
-   }

    try {
-       this.uploadInProgress = true;
        this.validateFile(file);

        this.showProgress();

        const response = await api.uploadDogPhoto(dogId, file);

        this.hideProgress();
        this.uploadInProgress = false;

        return response;
    } catch (error) {
        this.hideProgress();
        this.uploadInProgress = false;
        throw error;
    }
}
```

Better solution: Disable upload button during upload:

```javascript
async uploadPhoto(dogId, file) {
    const uploadBtn = document.getElementById('upload-btn');
    if (uploadBtn) uploadBtn.disabled = true;

    try {
        // ... upload code ...
    } finally {
        if (uploadBtn) uploadBtn.disabled = false;
    }
}
```

---

## Bug #12: Router 404 Handler Uses innerHTML

**Severity:** Medium

**Description:**
The default 404 handler in router.js uses `document.body.innerHTML` to display error, which clears all page content and removes all event listeners. This creates a poor user experience and breaks the application state.

**Location:**
- File: frontend/js/router.js (verified: `/home/jaco/Git-clones/gassigeher/internal/static/frontend/js/router.js`)
- **Verified Lines:** 54-56 (verified 2025-12-01)
- Code: `document.body.innerHTML = '<h1>404 - Page Not Found</h1>';`

**Verification Status:** ✅ **UNCHANGED - Bug still present**
- Line 54: `handler = this.routes['/404'] || (() => {`
- Line 55: `document.body.innerHTML = '<h1>404 - Page Not Found</h1>';`
- Line 56: `});`

**Steps to Reproduce:**
1. Navigate to non-existent route
2. Expected: User-friendly 404 page with navigation
3. Actual: Blank page with only text, no way to navigate back

**Impact:**
- Poor user experience
- User stuck on error page
- All scripts and event listeners removed
- Must manually edit URL or refresh

**Fix:**
Create proper 404 page or redirect to home:

```diff
// Default to 404 if no match
if (!handler) {
-   handler = this.routes['/404'] || (() => {
-       document.body.innerHTML = '<h1>404 - Page Not Found</h1>';
-   });
+   handler = this.routes['/404'] || (() => {
+       console.warn('404 - Route not found:', path);
+       window.location.href = '/index.html';
+   });
}
```

Or create a proper modal error:

```javascript
handler = this.routes['/404'] || (() => {
    const modal = document.createElement('div');
    modal.className = 'modal';
    modal.innerHTML = `
        <div class="card">
            <h2>Seite nicht gefunden</h2>
            <p>Die angeforderte Seite existiert nicht.</p>
            <a href="/" class="btn">Zurück zur Startseite</a>
        </div>
    `;
    document.body.appendChild(modal);
    modal.style.display = 'flex';
});
```

---

## Bug #13: Missing Input Validation in Time Rules Creation

**Severity:** Medium

**Description:**
When adding new time rules via `prompt()`, there's no validation of time format. Invalid times like "99:99" or "abc" are sent to the API, causing server errors instead of client-side validation.

**Location:**
- File: frontend/js/admin-booking-times.js (verified: `/home/jaco/Git-clones/gassigeher/internal/static/frontend/js/admin-booking-times.js`)
- **Verified Lines:** 148-173, 175-200 (verified 2025-12-01)
- Functions: Event handlers for add-weekday-rule-btn and add-weekend-rule-btn

**Verification Status:** ✅ **UNCHANGED - Bug still present**
- Line 152: `const startTime = prompt('Startzeit (HH:MM):', '09:00');`
- Line 153: `if (!startTime) return;` - only checks for empty, not format
- Line 155: `const endTime = prompt('Endzeit (HH:MM):', '12:00');`
- Line 156: `if (!endTime) return;` - only checks for empty, not format
- No time format validation present
- Same pattern in weekend rule handler (lines 179-183)

**Steps to Reproduce:**
1. Go to booking times admin page
2. Click "Zeitfenster hinzufügen"
3. Enter invalid time: "25:70"
4. Expected: Client-side validation error
5. Actual: Request sent to server, server error returned

**Impact:**
- Poor user experience
- Unnecessary API calls
- Server error logs polluted
- Confusing error messages

**Fix:**
Add time format validation before API call:

```diff
document.getElementById('add-weekday-rule-btn').addEventListener('click', async () => {
    const ruleName = prompt('Name des Zeitfensters:');
    if (!ruleName) return;

    const startTime = prompt('Startzeit (HH:MM):', '09:00');
    if (!startTime) return;
+
+   // Validate time format
+   const timeRegex = /^([01]\d|2[0-3]):([0-5]\d)$/;
+   if (!timeRegex.test(startTime)) {
+       showAlert('error', 'Ungültige Startzeit. Format: HH:MM (00:00 - 23:59)');
+       return;
+   }

    const endTime = prompt('Endzeit (HH:MM):', '12:00');
    if (!endTime) return;
+
+   if (!timeRegex.test(endTime)) {
+       showAlert('error', 'Ungültige Endzeit. Format: HH:MM (00:00 - 23:59)');
+       return;
+   }
+
+   // Validate that end > start
+   if (endTime <= startTime) {
+       showAlert('error', 'Endzeit muss nach Startzeit liegen');
+       return;
+   }

    // ... rest of code
});
```

---

## Bug #14: Alert Timeout Not Cleared on Multiple Alerts

**STATUS: CODE LOCATION VERIFIED - HTML FILES, NOT JS FILES**

**Note:** This bug is in HTML files with showAlert() function definitions, not in the frontend/js directory.

**Severity:** Medium

**Description:**
The `showAlert()` function uses `setTimeout()` to clear alerts after 5 seconds, but doesn't clear previous timeouts. If multiple alerts are shown rapidly, previous timeouts will clear newer alerts prematurely.

**Location:**
- Files: Multiple HTML files
- Function: `showAlert(type, message)`
- Pattern: `setTimeout(() => container.innerHTML = '', 5000);`
- **Verified Example:** admin-booking-times.js line 321 (verified 2025-12-01) - but this is in HTML script tags

**Steps to Reproduce:**
1. Trigger 3 alerts in quick succession (within 1 second)
2. Expected: Each alert visible for 5 seconds
3. Actual: First alert's timeout clears all alerts after 5 seconds from first alert

**Impact:**
- Alerts disappear too quickly
- Users miss important messages
- Confusing UX

**Fix:**
Clear previous timeout before setting new one:

```diff
+let alertTimeout = null;

function showAlert(type, message) {
    const container = document.getElementById('alert-container');
+
+   // Clear previous timeout
+   if (alertTimeout) {
+       clearTimeout(alertTimeout);
+   }
+
    const div = document.createElement('div');
    div.className = `alert alert-${type}`;
    div.textContent = message;
    container.innerHTML = '';
    container.appendChild(div);

-   setTimeout(() => container.innerHTML = '', 5000);
+   alertTimeout = setTimeout(() => {
+       container.innerHTML = '';
+       alertTimeout = null;
+   }, 5000);
}
```

---

## Bug #15: localStorage Data Persists After Logout

**Severity:** Medium

**Description:**
The `logout()` method only clears the token from localStorage but doesn't clear other data like `pendingBooking`. This leaves sensitive data on the device after logout, creating a privacy issue on shared computers.

**Location:**
- File: frontend/js/api.js (verified: `/home/jaco/Git-clones/gassigeher/internal/static/frontend/js/api.js`)
- **Verified Lines:** 110-113 (verified 2025-12-01)
- Function: `logout()`

**Verification Status:** ✅ **UNCHANGED - Bug still present**
- Line 110: `async logout() {`
- Line 111: `this.setToken(null);` - only clears token
- Line 112: `window.location.href = '/';`
- Line 113: `}`
- No cleanup of other localStorage keys (e.g., pendingBooking)

**Steps to Reproduce:**
1. Login and create a pending booking (stored in localStorage)
2. Logout
3. Check localStorage in browser DevTools
4. Expected: All app data cleared
5. Actual: pendingBooking and other data still present

**Impact:**
- Privacy leak on shared computers
- Next user sees previous user's pending bookings
- Potential data confusion

**Fix:**
Clear all app-specific localStorage on logout:

```diff
async logout() {
-   this.setToken(null);
+   // Clear all app data from localStorage
+   const keysToRemove = [];
+   for (let i = 0; i < localStorage.length; i++) {
+       const key = localStorage.key(i);
+       // Remove gassigeher-specific keys
+       if (key === 'gassigeher_token' || key === 'pendingBooking') {
+           keysToRemove.push(key);
+       }
+   }
+   keysToRemove.forEach(key => localStorage.removeItem(key));
+
+   this.token = null;
    window.location.href = '/';
}
```

Or use a namespace pattern:

```javascript
// Store all app data under one key
localStorage.setItem('gassigeher_data', JSON.stringify({
    token: '...',
    pendingBooking: {...}
}));

// Clear everything on logout
localStorage.removeItem('gassigeher_data');
```

---

## Bug #16: I18n Translation Key Returned Instead of Fallback Text

**Severity:** Low

**Description:**
When a translation key is not found, the i18n system returns the key itself (e.g., "dogs.invalid_key"). For deeply nested or misspelled keys, this displays cryptic dotted strings to users instead of meaningful fallback text.

**Location:**
- File: frontend/js/i18n.js (verified: `/home/jaco/Git-clones/gassigeher/internal/static/frontend/js/i18n.js`)
- **Verified Lines:** 22-35 (verified 2025-12-01)
- Function: `t(key)`

**Verification Status:** ✅ **UNCHANGED - Bug still present**
- Line 22: `t(key) {`
- Lines 23-31: Loop through keys
- Line 30: `return key;` - returns key if not found during traversal
- Line 34: `return value || key;` - returns key if value is undefined

**Steps to Reproduce:**
1. Use translation key that doesn't exist: `i18n.t('dogs.nonexistent_key')`
2. Expected: Fallback to English or meaningful default
3. Actual: Returns "dogs.nonexistent_key" as-is

**Impact:**
- Poor user experience when translations missing
- Debug artifacts visible to end users
- Difficult to identify missing translations in production

**Fix:**
Return last part of key or a clear fallback message:

```diff
t(key) {
    const keys = key.split('.');
    let value = this.translations;

    for (const k of keys) {
        if (value && typeof value === 'object') {
            value = value[k];
        } else {
-           return key; // Return key if translation not found
+           // Return last part of key as fallback (more user-friendly)
+           const lastPart = keys[keys.length - 1];
+           console.warn(`Translation missing for key: ${key}`);
+           return lastPart.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
        }
    }

-   return value || key;
+   if (value === undefined || value === null) {
+       const lastPart = keys[keys.length - 1];
+       console.warn(`Translation missing for key: ${key}`);
+       return lastPart.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
+   }
+   return value;
}
```

---

## Bug #17: No Abort Signal for Pending Fetch Requests

**Severity:** Low

**Description:**
When users navigate away from a page quickly (e.g., click back button during API call), fetch requests continue in the background. This wastes bandwidth and can cause race conditions if responses arrive after page change.

**Location:**
- File: frontend/js/api.js (verified: `/home/jaco/Git-clones/gassigeher/internal/static/frontend/js/api.js`)
- **Verified Lines:** 38-45, 48 (verified 2025-12-01)
- Function: `request()`

**Verification Status:** ✅ **UNCHANGED - Bug still present**
- Lines 38-41: `options` object created
- Line 48: `fetch()` called with options
- No `signal` property in options
- No AbortController usage

**Steps to Reproduce:**
1. Navigate to dogs.html page
2. Immediately click back button while dogs are loading
3. Expected: API request cancelled
4. Actual: Request continues, response processed even though user left page

**Impact:**
- Wasted bandwidth
- Server resources used unnecessarily
- Potential race conditions with stale data
- Console errors when DOM elements don't exist

**Fix:**
Use AbortController for all fetch requests:

```diff
class API {
    constructor() {
        this.baseURL = '/api';
        this.token = localStorage.getItem('gassigeher_token');
+       this.pendingRequests = new Map();
    }

    async request(method, endpoint, data = null) {
+       // Create abort controller for this request
+       const controller = new AbortController();
+       const requestId = `${method}:${endpoint}`;
+
+       // Abort previous identical request if still pending
+       if (this.pendingRequests.has(requestId)) {
+           this.pendingRequests.get(requestId).abort();
+       }
+       this.pendingRequests.set(requestId, controller);

        const headers = {
            'Content-Type': 'application/json',
        };

        if (this.token) {
            headers['Authorization'] = `Bearer ${this.token}`;
        }

        const options = {
            method,
            headers,
+           signal: controller.signal,
        };

        if (data && (method === 'POST' || method === 'PUT')) {
            options.body = JSON.stringify(data);
        }

        try {
            const response = await fetch(`${this.baseURL}${endpoint}`, options);
+           this.pendingRequests.delete(requestId);

            const responseData = await response.json();

            if (!response.ok) {
                const error = new Error(responseData.error || 'Request failed');
                error.status = response.status;
                error.data = responseData;
                throw error;
            }

            return responseData;
        } catch (error) {
+           this.pendingRequests.delete(requestId);
+           // Don't throw error if request was aborted
+           if (error.name === 'AbortError') {
+               console.log('Request aborted:', requestId);
+               return null;
+           }
            throw error;
        }
    }
+
+   // Call this when navigating away from page
+   cancelAllRequests() {
+       this.pendingRequests.forEach(controller => controller.abort());
+       this.pendingRequests.clear();
+   }
}
```

Then add to page navigation:

```javascript
window.addEventListener('beforeunload', () => {
    window.api.cancelAllRequests();
});
```

---

## Bug #18: Unvalidated Date Format in Calendar localStorage

**STATUS: CODE LOCATION VERIFIED - HTML FILES, NOT JS FILES**

**Note:** This bug is in HTML files (dogs.html, calendar.html), not in the frontend/js directory.

**Severity:** Low

**Description:**
The calendar saves pendingBooking to localStorage with date strings, but doesn't validate the date format when reading it back. If the data is corrupted or manually edited, invalid dates can cause booking failures.

**Location:**
- Files: dogs.html, calendar.html
- Lines: dogs.html:289-306, calendar.html:646-652

**Steps to Reproduce:**
1. Create pending booking via calendar
2. Manually edit localStorage to set invalid date: `{"date": "invalid", "dogId": 1}`
3. Navigate to dogs.html
4. Expected: Validation error or data cleared
5. Actual: Code attempts to use invalid date, causes errors

**Impact:**
- Booking failures with unclear errors
- Poor user experience
- Data corruption if user edits localStorage

**Fix:**
Validate localStorage data before using:

```diff
// In dogs.html
const pendingBookingStr = localStorage.getItem('pendingBooking');
console.log('[DEBUG] localStorage pendingBooking:', pendingBookingStr);

if (pendingBookingStr) {
    try {
        const pendingBooking = JSON.parse(pendingBookingStr);
-       console.log('[DEBUG] Parsed pendingBooking:', pendingBooking);
+
+       // Validate required fields
+       if (!pendingBooking.dogId || !pendingBooking.date) {
+           console.warn('[DEBUG] Invalid pendingBooking structure, clearing');
+           localStorage.removeItem('pendingBooking');
+           return;
+       }
+
+       // Validate date format (YYYY-MM-DD)
+       const dateRegex = /^\d{4}-\d{2}-\d{2}$/;
+       if (!dateRegex.test(pendingBooking.date)) {
+           console.warn('[DEBUG] Invalid date format in pendingBooking, clearing');
+           localStorage.removeItem('pendingBooking');
+           return;
+       }
+
+       // Validate date is not in the past
+       const bookingDate = new Date(pendingBooking.date);
+       const today = new Date();
+       today.setHours(0, 0, 0, 0);
+       if (bookingDate < today) {
+           console.warn('[DEBUG] Pending booking date is in the past, clearing');
+           localStorage.removeItem('pendingBooking');
+           return;
+       }
+
+       console.log('[DEBUG] Valid pendingBooking:', pendingBooking);

        // Find the dog and open booking modal
        // ... rest of code
    } catch (e) {
        console.error('[DEBUG] Failed to parse pendingBooking:', e);
        localStorage.removeItem('pendingBooking');
    }
}
```

---

## Statistics

- **Critical:** 7 bugs (XSS vulnerabilities - all in HTML files, not JS files)
- **High:** 3 bugs (Authentication, error handling - 2 in api.js still present)
- **Medium:** 5 bugs (Race conditions, memory leaks, UX issues - 4 in JS files still present, 1 in HTML)
- **Low:** 3 bugs (Edge cases, validation - 2 in api.js/i18n.js still present, 1 in HTML)

**JS Files Breakdown:**
- **api.js:** 4 bugs still present (Bugs #8, #9, #15, #17)
- **dog-photo.js:** 2 bugs still present (Bugs #10, #11)
- **router.js:** 1 bug still present (Bug #12)
- **i18n.js:** 1 bug still present (Bug #16)
- **admin-booking-times.js:** 1 bug still present (Bug #13)
- **HTML files (not in scope):** 9 bugs (Bugs #1-7, #14, #18)

---

## Recommendations

### Immediate Actions (Critical Priority)

1. **Implement XSS Protection** (Bugs #1-7 - HTML FILES)
   - Create centralized `sanitizeHTML()` utility function
   - Audit ALL innerHTML assignments (18+ locations across HTML files)
   - Replace innerHTML with textContent or DOM manipulation where possible
   - Add Content Security Policy headers on backend

2. **Fix onclick Injection Vectors** (Bugs #6-7 - HTML FILES)
   - Replace inline onclick handlers with event delegation
   - Use data attributes to pass parameters
   - Sanitize all user-controlled data in attributes

3. **Secure Error Display** (Bug #3 - HTML FILES)
   - Sanitize all error messages before displaying
   - Never trust API error messages to be safe HTML

### Short-term Improvements (High Priority)

4. **Improve Token Management** (Bug #8 - api.js)
   - Always read token from localStorage in request()
   - Consider using HttpOnly cookies instead of localStorage
   - Implement token refresh mechanism

5. **Add Request Cancellation** (Bug #17 - api.js)
   - Use AbortController for all fetch requests
   - Cancel pending requests on page navigation
   - Prevent duplicate requests

6. **Enhance Error Handling** (Bug #9 - api.js)
   - Add JSON parse error handling in API client
   - Provide user-friendly error messages
   - Log errors for debugging

### Long-term Enhancements (Medium Priority)

7. **Memory Leak Prevention** (Bug #10 - dog-photo.js)
   - Use AbortController for event listeners
   - Clean up listeners when components unmount
   - Implement proper lifecycle management

8. **Input Validation** (Bug #13 - admin-booking-times.js)
   - Add client-side validation for all user inputs
   - Validate date/time formats before API calls
   - Validate localStorage data before using

9. **Privacy Improvements** (Bug #15 - api.js)
   - Clear all localStorage on logout
   - Use session storage for temporary data
   - Implement data expiration for cached data

### Best Practices Going Forward

10. **Security-First Development**
    - Never use innerHTML with user data
    - Always sanitize before rendering
    - Use textContent or createElement() instead
    - Implement CSP headers

11. **Code Quality**
    - Add JSDoc comments for all functions
    - Use TypeScript for type safety
    - Implement unit tests for critical functions
    - Add linting rules to catch innerHTML usage

12. **User Experience**
    - Show loading states during API calls
    - Provide clear error messages
    - Implement retry logic for failed requests
    - Add offline support

### Tools and Libraries to Consider

- **DOMPurify**: Industry-standard HTML sanitization library
- **TypeScript**: Catch type-related bugs at compile time
- **ESLint**: Enforce security rules (no-innerHTML, etc.)
- **Jest**: Unit testing for JavaScript functions
- **CSP**: Content Security Policy to prevent XSS at browser level

---

## Conclusion

The frontend JavaScript codebase has **serious security vulnerabilities** due to widespread unsanitized innerHTML usage. However, **most XSS bugs (Bugs #1-7) are in HTML files, not the JavaScript modules themselves**.

**JavaScript Modules Status:**
- **9 out of 18 bugs verified in JS files** (Bugs #8-#13, #15-#17)
- **All 9 JS bugs are UNCHANGED and still present** in the codebase
- **Bugs #1-7, #14, #18 are in HTML files**, outside the scope of the frontend/js directory

Beyond security, the code has several architectural issues (race conditions, memory leaks, poor error handling) that impact reliability and user experience.

**Priority order:**
1. Fix all XSS vulnerabilities (Bugs #1-7) in HTML files - **URGENT**
2. Improve token management and error handling (Bugs #8-9 in api.js)
3. Address memory leaks and race conditions (Bugs #10-11 in dog-photo.js)
4. Enhance validation and UX (Bugs #12-13, #15-17 in various JS files)

**Estimated effort:**
- Critical fixes (HTML files): 2-3 days (sanitization, onclick removal)
- High priority (api.js): 1-2 days (token, error handling)
- Medium priority (dog-photo.js, others): 2-3 days (memory leaks, validation)
- **Total: 5-8 days** for a senior developer

The good news is that most bugs follow similar patterns, so fixes can be applied systematically across multiple files once the patterns are established.
