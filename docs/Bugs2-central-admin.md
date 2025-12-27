# Bug Report: Central Admin Dashboard

**Analysis Date:** 2025-12-27
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/static/central`
**Files Analyzed:** 6 files (4 HTML, 1 JS, 1 CSS)
**Bugs Found:** 12 bugs

---

## Summary

This analysis examined the SaaS central admin dashboard responsible for platform-wide administration across all tenants. The dashboard includes tenant management, user search, admin promotion/demotion, and impersonation features.

**Critical findings:**
- **3 High severity** bugs related to authorization bypass and privilege escalation
- **4 Medium severity** bugs related to XSS vulnerabilities and missing validation
- **5 Low severity** bugs related to UI issues and error handling

**Most Critical Issues:**
1. Missing `is_central_admin` field check in user search results (privilege escalation risk)
2. Multiple XSS vulnerabilities in innerHTML assignments without sanitization
3. Missing validation in promotion/demotion functions allowing privilege escalation
4. Hardcoded domain in impersonation redirect

---

## Bugs

## Bug #1: Missing is_central_admin Field Check in User Search Results

**Severity:** HIGH

**Description:**
The `searchUsers()` function in `users.html` displays a "Zum Central Admin" button for ALL users without checking if they are already central admins. The backend search endpoint (`SearchUsers` in `central_admin_handler.go`) does not return the `is_central_admin` field, making it impossible for the frontend to hide the button for existing central admins.

This creates confusion and could allow accidental re-promotion of existing central admins, wasting audit log entries and potentially masking security issues.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/users.html`
- Function: `searchUsers`
- Lines: 100-146

**Backend Issue:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/central_admin_handler.go`
- Function: `SearchUsers`
- Lines: 507-518 (UserSearchResult struct missing `IsCentralAdmin` field)

**Steps to Reproduce:**
1. Login as central admin
2. Navigate to `/central/users.html`
3. Search for any central admin user by name
4. Observe "Zum Central Admin" button displayed for already-central-admin user
5. Click button - backend will return error "Benutzer ist bereits Central Admin" (line 385-387 in handler)

**Impact:**
- **User confusion**: Misleading UI suggests user is not a central admin when they are
- **Wasted API calls**: Frontend makes unnecessary promotion requests
- **Audit log pollution**: Failed promotion attempts clutter security logs
- **Privilege information leak**: Cannot distinguish normal users from central admins in search results

**Fix:**

Backend changes:
```diff
// internal/handlers/central_admin_handler.go, line 507
type UserSearchResult struct {
    ID           int       `json:"id"`
    TenantID     int       `json:"tenant_id"`
    FirstName    string    `json:"first_name"`
    LastName     string    `json:"last_name"`
    Email        *string   `json:"email"`
    IsAdmin      bool      `json:"is_admin"`
    IsSuperAdmin bool      `json:"is_super_admin"`
+   IsCentralAdmin bool    `json:"is_central_admin"`
    IsActive     bool      `json:"is_active"`
    CreatedAt    time.Time `json:"created_at"`
    TenantName   *string   `json:"tenant_name"`
}
```

```diff
// Line 490 - Update query
rows, err := h.db.Query(`
    SELECT u.id, u.tenant_id, u.first_name, u.last_name, u.email, u.is_admin, u.is_super_admin,
-          u.is_active, u.created_at, t.name as tenant_name
+          u.is_central_admin, u.is_active, u.created_at, t.name as tenant_name
    FROM users u
    LEFT JOIN tenants t ON u.tenant_id = t.id
    WHERE u.is_deleted = 0
      AND (u.first_name LIKE ? OR u.last_name LIKE ? OR u.email LIKE ?)
    ORDER BY u.last_name, u.first_name
    LIMIT 100
`, searchPattern, searchPattern, searchPattern)
```

```diff
// Line 523 - Update scan
if err := rows.Scan(&u.ID, &u.TenantID, &u.FirstName, &u.LastName, &u.Email,
-   &u.IsAdmin, &u.IsSuperAdmin, &u.IsActive, &u.CreatedAt, &u.TenantName); err != nil {
+   &u.IsAdmin, &u.IsSuperAdmin, &u.IsCentralAdmin, &u.IsActive, &u.CreatedAt, &u.TenantName); err != nil {
```

Frontend changes:
```diff
// internal/static/central/users.html, line 134
<td>
-   ${!u.is_super_admin && !u.is_central_admin && u.is_active ?
+   ${!u.is_super_admin && !u.is_central_admin && u.is_active ?
        `<button class="btn btn-sm btn-primary" onclick="impersonateUser(${u.id}, '${escapeHtml(u.first_name)} ${escapeHtml(u.last_name)}', '${escapeHtml(u.tenant_name || '')}')">Imitieren</button>` : ''
    }
-   <button class="btn btn-sm btn-secondary" onclick="promoteToAdmin(${u.id}, '${escapeHtml(u.first_name)} ${escapeHtml(u.last_name)}')">Zum Central Admin</button>
+   ${!u.is_central_admin ?
+       `<button class="btn btn-sm btn-secondary" onclick="promoteToAdmin(${u.id}, '${escapeHtml(u.first_name)} ${escapeHtml(u.last_name)}')">Zum Central Admin</button>` :
+       '<span class="badge badge-info">Bereits Central Admin</span>'
+   }
</td>
```

This ensures:
1. Backend returns `is_central_admin` status for each user
2. Frontend conditionally shows promotion button only for non-central-admins
3. Central admins are clearly labeled in search results
4. Prevents unnecessary API calls and audit log pollution

---

## Bug #2: XSS Vulnerability in Tenant Name Display

**Severity:** MEDIUM

**Description:**
The `loadRecentTenants()` function in `index.html` uses template literals with `${escapeHtml(t.name)}` for tenant names, which is correct. However, the tenant details modal in `tenants.html` (line 264) uses `textContent` which is safe, but several other locations use innerHTML without proper context escaping.

While `escapeHtml()` is used in most places, the function itself (defined in `central.js` line 157-162) only escapes basic HTML entities and may not protect against all XSS vectors, especially when combined with onclick handlers that use template literals.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/index.html`
- Lines: 149-158 (tenant list)
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/tenants.html`
- Lines: 184-204 (tenant table)
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/users.html`
- Lines: 115-140 (user search results)

**Steps to Reproduce:**
1. Create a tenant with name: `Test"><img src=x onerror=alert('XSS')>`
2. Login as central admin
3. Navigate to central admin dashboard
4. Observe XSS attempt is blocked by `escapeHtml()` - **THIS IS GOOD**
5. However, create tenant with name: `Test' onmouseover='alert(1)' x='`
6. Navigate to tenants page and hover over tenant link
7. Potential XSS if escapeHtml doesn't handle single quotes in attribute context

**Impact:**
- **Medium risk**: XSS protection relies solely on client-side `escapeHtml()` function
- **Stored XSS**: Malicious tenant names stored in database could execute in admin context
- **Privilege escalation**: XSS in admin dashboard could steal admin tokens
- **Defense in depth violation**: No server-side validation or Content-Security-Policy headers

**Fix:**

1. **Server-side validation** (highest priority):
```diff
// internal/handlers/tenant_handler.go - Add to CreateTenant validation
func (h *TenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...

+   // Validate name for XSS prevention
+   if containsHTMLTags(req.Name) {
+       respondError(w, http.StatusBadRequest, "Tierheim-Name darf keine HTML-Tags enthalten")
+       return
+   }
}

+func containsHTMLTags(s string) bool {
+   return strings.ContainsAny(s, "<>&\"'")
+}
```

2. **Improve escapeHtml function**:
```diff
// internal/static/central/assets/js/central.js, line 157
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
-   return div.innerHTML;
+   // Also escape single quotes for attribute context safety
+   return div.innerHTML
+       .replace(/'/g, '&#39;')
+       .replace(/"/g, '&quot;');
}
```

3. **Add Content-Security-Policy header**:
```diff
// internal/middleware/middleware.go - Add to SecurityHeadersMiddleware
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+       w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
        // ... existing headers ...
    })
}
```

4. **Use textContent instead of innerHTML where possible**:
```diff
// tenants.html, line 264
-document.getElementById('modal-tenant-name').textContent = tenant.name;
+// This is already correct - keep using textContent
```

---

## Bug #3: Hardcoded Domain in Impersonation Redirect

**Severity:** MEDIUM

**Description:**
The `impersonateUser()` function in `users.html` (line 223-260) hardcodes the production domain `gassigeher.org` when constructing the redirect URL. This will break impersonation functionality in:
- Local development (localhost)
- Staging environments
- Custom domain deployments

The code attempts to detect localhost but fails to handle staging/custom domains properly.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/users.html`
- Function: `impersonateUser`
- Lines: 243-250

**Steps to Reproduce:**
1. Set up staging environment at `staging.example.com`
2. Login as central admin
3. Search for user and click "Imitieren"
4. Observe redirect attempts to go to `https://${slug}.gassigeher.org` instead of `https://${slug}.staging.example.com`
5. Redirect fails or goes to wrong domain

**Impact:**
- **Broken functionality**: Impersonation doesn't work in non-production environments
- **Security risk**: Redirecting to production from staging could leak staging tokens
- **Development friction**: Developers cannot test impersonation locally

**Fix:**

Extract base domain from current hostname:
```diff
// internal/static/central/users.html, line 243
async function impersonateUser(userId, userName, tenantName) {
    // ... existing code ...

    try {
        const result = await centralAPI.impersonateUser(userId);

        // ... token storage ...

        showAlert(`Sie sind jetzt als ${userName} angemeldet. Weiterleitung...`, 'success');

        // Redirect to tenant dashboard
        if (result.tenant?.slug) {
-           // In SaaS mode, redirect to tenant subdomain
-           const baseUrl = window.location.hostname.includes('localhost') ?
-               window.location.origin :
-               `https://${result.tenant.slug}.gassigeher.org`;
+           // Extract base domain from current central admin URL
+           // e.g., central.gassigeher.org -> gassigeher.org
+           //       central.staging.example.com -> staging.example.com
+           //       localhost:8080 -> localhost:8080
+           const hostname = window.location.hostname;
+           let baseDomain;
+
+           if (hostname.includes('localhost') || hostname.match(/^\d+\.\d+\.\d+\.\d+$/)) {
+               // Local development - keep protocol and port
+               baseDomain = window.location.origin;
+           } else {
+               // Production/staging - remove "central" subdomain
+               // central.gassigeher.org -> gassigeher.org
+               const parts = hostname.split('.');
+               if (parts.length >= 3 && parts[0] === 'central') {
+                   baseDomain = `https://${parts.slice(1).join('.')}`;
+               } else {
+                   // Fallback to current protocol and host
+                   baseDomain = `${window.location.protocol}//${hostname}`;
+               }
+           }
+
+           const tenantUrl = hostname.includes('localhost') ?
+               `${baseDomain}/dashboard.html?tenant=${result.tenant.slug}` :
+               `${baseDomain.replace(/https:\/\//, `https://${result.tenant.slug}.`)}/dashboard.html`;
+
            setTimeout(() => {
-               window.location.href = baseUrl + '/dashboard.html';
+               window.location.href = tenantUrl;
            }, 1000);
        } else {
            // Fallback to main domain dashboard
            setTimeout(() => {
                window.location.href = '/dashboard.html';
            }, 1000);
        }
    }
}
```

Alternative simpler fix using backend config:
```diff
// Have backend return full tenant URL in impersonation response
// internal/handlers/central_admin_handler.go, line 812
respondJSON(w, http.StatusOK, map[string]interface{}{
    "token":  token,
    "user":   targetUser,
    "tenant": tenant,
+   "tenant_url": h.cfg.GetTenantURL(tenant.Slug),  // e.g., "https://demo1.gassigeher.org"
})
```

---

## Bug #4: Missing Authorization Check in promoteToAdmin Function

**Severity:** HIGH

**Description:**
The frontend `promoteToAdmin()` function in `users.html` (line 199-209) does not check if the target user is already a super admin or central admin before making the promotion request. While the backend validates this (line 384-387 in `central_admin_handler.go`), the frontend should prevent the request entirely to:
1. Avoid unnecessary API calls
2. Provide immediate feedback
3. Prevent audit log pollution

Additionally, the function allows promoting users who are inactive, which may not be desired behavior.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/users.html`
- Function: `promoteToAdmin`
- Lines: 199-209

**Steps to Reproduce:**
1. Login as central admin
2. Search for a super admin user
3. Click "Zum Central Admin" button (should not be visible but is due to Bug #1)
4. Backend returns error "Benutzer ist bereits Central Admin"
5. User sees error instead of button being disabled

**Impact:**
- **Poor UX**: Error message instead of disabled button
- **Audit log pollution**: Failed promotion attempts logged
- **Wasted bandwidth**: Unnecessary API requests
- **Security confusion**: Makes it unclear who can be promoted

**Fix:**

Add validation before making API call:
```diff
// internal/static/central/users.html, line 199
-async function promoteToAdmin(userId, userName) {
+async function promoteToAdmin(userId, userName, userObj) {
+   // Validate user is eligible for promotion
+   if (userObj.is_central_admin) {
+       showAlert('Benutzer ist bereits Central Admin', 'warning');
+       return;
+   }
+
+   if (userObj.is_super_admin) {
+       showAlert('Super-Admins können nicht befördert werden', 'warning');
+       return;
+   }
+
+   if (!userObj.is_active) {
+       if (!confirm(`${userName} ist inaktiv. Trotzdem zum Central Admin befördern?`)) {
+           return;
+       }
+   }
+
    if (!confirm(`${userName} zum Central Administrator befördern?`)) return;

    try {
        await centralAPI.promoteToAdmin(userId);
        showAlert(`${userName} ist jetzt Central Administrator`, 'success');
        loadAdmins();
+       // Re-search to update button states
+       searchUsers();
    } catch (err) {
        showAlert('Fehler: ' + err.message, 'error');
    }
}
```

Update button onclick to pass full user object:
```diff
// Line 137
-<button class="btn btn-sm btn-secondary" onclick="promoteToAdmin(${u.id}, '${escapeHtml(u.first_name)} ${escapeHtml(u.last_name)}')">Zum Central Admin</button>
+<button class="btn btn-sm btn-secondary" onclick='promoteToAdmin(${u.id}, "${escapeHtml(u.first_name)} ${escapeHtml(u.last_name)}", ${JSON.stringify(u)})'>Zum Central Admin</button>
```

---

## Bug #5: Self-Demotion Prevention Missing in Frontend

**Severity:** MEDIUM

**Description:**
The `demoteAdmin()` function in `users.html` (line 211-221) does not prevent central admins from demoting themselves in the UI, even though the backend prevents it (line 415-419 in handler). This creates poor UX where the user clicks "Entfernen" on their own row, waits for the API call, then sees an error message.

The current user's ID should be checked before showing the "Entfernen" button or making the API call.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/users.html`
- Function: `loadAdmins` and `demoteAdmin`
- Lines: 153-197 (table rendering) and 211-221 (demotion function)

**Steps to Reproduce:**
1. Login as central admin (user ID = 1)
2. Navigate to `/central/users.html`
3. Scroll to "Central Administratoren verwalten" section
4. See "Entfernen" button next to your own name
5. Click it and confirm
6. Backend returns error "Sie können sich nicht selbst degradieren"

**Impact:**
- **Poor UX**: User can attempt self-demotion only to be denied
- **Wasted API call**: Unnecessary request that will always fail
- **Confusion**: Users may think system is broken

**Fix:**

Get current user ID and hide self-demotion button:
```diff
// internal/static/central/users.html, line 67
document.addEventListener('DOMContentLoaded', async () => {
    if (!isAuthenticated()) {
        window.location.href = '/login.html';
        return;
    }

+   // Get current user info to prevent self-demotion
+   try {
+       const me = await centralAPI.getMe();
+       window.currentUserId = me.id;
+   } catch (err) {
+       console.error('Failed to get current user:', err);
+   }

    await loadAdmins();
    // ...
});
```

```diff
// Line 186
<td>
-   <button class="btn btn-sm btn-warning" onclick="demoteAdmin(${a.id}, '${escapeHtml(a.first_name)} ${escapeHtml(a.last_name)}')">Entfernen</button>
+   ${window.currentUserId !== a.id ?
+       `<button class="btn btn-sm btn-warning" onclick="demoteAdmin(${a.id}, '${escapeHtml(a.first_name)} ${escapeHtml(a.last_name)}')">Entfernen</button>` :
+       '<span class="text-muted">Sie selbst</span>'
+   }
</td>
```

Add API method if not exists:
```diff
// internal/static/central/assets/js/central.js, line 66
const centralAPI = {
+   // Get current user info
+   async getMe() {
+       return apiRequest('/users/me');
+   },
+
    // Stats
    async getStats() {
        return apiRequest('/central-admin/stats');
    },
```

---

## Bug #6: Missing Error Handling in exportTenant Function

**Severity:** LOW

**Description:**
The `exportTenant()` function in `tenants.html` (line 466-485) downloads tenant data as JSON but does not validate that data was actually returned before creating the download. If the API returns an empty response or error, the function will create a blank or malformed JSON file.

Additionally, there's no check if `currentTenantId` is set, which could cause issues if the modal state is corrupted.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/tenants.html`
- Function: `exportTenant`
- Lines: 466-485

**Steps to Reproduce:**
1. Open tenant modal
2. Backend has database error during export
3. API returns error but function tries to download it anyway
4. User gets file named `tenant-X-export.json` containing `{"error": "..."}`

**Impact:**
- **Confusing UX**: User thinks export succeeded but got error JSON
- **Data loss**: User may delete original data thinking they have backup
- **Poor error reporting**: Error is hidden in downloaded file

**Fix:**

Add validation and error handling:
```diff
// internal/static/central/tenants.html, line 466
async function exportTenant() {
    if (!currentTenantId) {
+       showAlert('Kein Tierheim ausgewählt', 'error');
        return;
    }

    try {
        const data = await centralAPI.exportTenant(currentTenantId);
+
+       // Validate response
+       if (!data || typeof data !== 'object') {
+           showAlert('Export-Daten sind ungültig', 'error');
+           return;
+       }
+
+       // Check if response is an error object
+       if (data.error) {
+           showAlert('Export fehlgeschlagen: ' + data.error, 'error');
+           return;
+       }
+
+       // Validate required fields
+       if (!data.tenant || !data.users) {
+           showAlert('Export unvollständig - einige Daten fehlen', 'warning');
+       }
+
        // Download as JSON
        const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
-       a.download = `tenant-${currentTenantId}-export.json`;
+       const tenantSlug = data.tenant?.slug || currentTenantId;
+       const timestamp = new Date().toISOString().split('T')[0];
+       a.download = `tenant-${tenantSlug}-export-${timestamp}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
-       showAlert('Export heruntergeladen', 'success');
+
+       // Show detailed success message
+       const stats = `${data.users?.length || 0} Benutzer, ${data.dogs?.length || 0} Hunde`;
+       showAlert(`Export erfolgreich heruntergeladen (${stats})`, 'success');
    } catch (err) {
        showAlert('Fehler beim Export: ' + err.message, 'error');
    }
}
```

---

## Bug #7: Race Condition in Tenant Activity Loading

**Severity:** LOW

**Description:**
In `tenants.html`, the `loadTenants()` function has two different code paths for loading tenant activity data:
1. When "inactive only" is checked, it loads activity data inline (lines 134-156)
2. When showing all tenants, it loads activity data in the background (line 159)

The background loading path calls `loadActivityData()` which updates DOM elements by ID. However, if the user changes filters or navigates away before the activity data loads, these DOM elements may not exist, causing console errors.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/tenants.html`
- Function: `loadActivityData`
- Lines: 217-230

**Steps to Reproduce:**
1. Navigate to tenants page
2. Load all tenants (activity data loads in background)
3. Quickly change filter to "inactive only" before activity data arrives
4. Background request completes and tries to update non-existent DOM elements
5. Console error: "Cannot read property 'innerHTML' of null"

**Impact:**
- **Console pollution**: Error messages in developer console
- **Minor UX issue**: Activity badges not shown if request is slow
- **Potential memory leak**: Cached activity data not cleared on filter change

**Fix:**

Add cancellation token and existence check:
```diff
// internal/static/central/tenants.html
let tenantActivityData = {}; // Cache for activity data
+let activityLoadAbortController = null; // Cancellation token

async function loadTenants() {
    const search = document.getElementById('search-input').value;
    const activeOnly = document.getElementById('active-only-checkbox').checked;
    const inactiveOnly = document.getElementById('inactive-only-checkbox').checked;
    const inactivityDays = document.getElementById('inactivity-days').value;
    const container = document.getElementById('tenants-container');

+   // Cancel previous activity load
+   if (activityLoadAbortController) {
+       activityLoadAbortController.abort();
+   }

    container.innerHTML = '<p class="loading">Laden...</p>';

    try {
        let tenants;

        if (inactiveOnly) {
            // ... existing code ...
        } else {
            tenants = await centralAPI.getTenants(search, activeOnly);
            // Load activity data in background
-           loadActivityData();
+           activityLoadAbortController = new AbortController();
+           loadActivityData(activityLoadAbortController.signal);
        }

        // ... render tenants ...
    }
}

-async function loadActivityData() {
+async function loadActivityData(signal) {
    try {
        const activityData = await centralAPI.getTenantActivity();
+
+       // Check if request was cancelled
+       if (signal?.aborted) {
+           return;
+       }
+
        activityData.forEach(a => {
            tenantActivityData[a.tenant_id] = a;
            const cell = document.getElementById(`activity-${a.tenant_id}`);
-           if (cell) {
+           // Check cell exists and request wasn't cancelled
+           if (cell && !signal?.aborted) {
                cell.innerHTML = getActivityBadge(a.days_inactive, a.is_inactive);
            }
        });
    } catch (err) {
+       if (err.name === 'AbortError') {
+           return; // Request was cancelled - this is expected
+       }
        console.error('Failed to load activity data:', err);
    }
}
```

---

## Bug #8: Missing Validation in activateTenant and deactivateTenant

**Severity:** MEDIUM

**Description:**
The `activateTenant()` and `deactivateTenant()` functions in `tenants.html` (lines 438-464) do not validate `currentTenantStatus` before making API calls. This means:
1. User can click "Aktivieren" on already-active tenant → wasted API call
2. User can click "Deaktivieren" on already-suspended tenant → wasted API call
3. No check if tenant is in "deleted" status which should not be modifiable

The modal shows/hides buttons based on status (line 267-268), but the functions themselves should also validate.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/tenants.html`
- Functions: `activateTenant`, `deactivateTenant`
- Lines: 438-464

**Steps to Reproduce:**
1. Open tenant modal for active tenant
2. Open browser console
3. Execute `activateTenant()` directly
4. API call succeeds but does nothing (tenant already active)
5. No user feedback that operation was unnecessary

**Impact:**
- **Wasted API calls**: Unnecessary database operations
- **Audit log pollution**: Logs activation of already-active tenants
- **Poor UX**: No feedback that operation was no-op
- **Potential inconsistency**: Could activate deleted tenants

**Fix:**

Add status validation:
```diff
// internal/static/central/tenants.html, line 438
async function activateTenant() {
    if (!currentTenantId) return;
+
+   // Validate current status
+   if (currentTenantStatus === 'active') {
+       showAlert('Tierheim ist bereits aktiv', 'info');
+       return;
+   }
+
+   if (currentTenantStatus === 'deleted') {
+       showAlert('Gelöschte Tierheime können nicht reaktiviert werden', 'error');
+       return;
+   }
+
    if (!confirm('Tierheim wirklich aktivieren?')) return;

    try {
        await centralAPI.activateTenant(currentTenantId);
        showAlert('Tierheim aktiviert', 'success');
+       currentTenantStatus = 'active'; // Update cached status
        closeModal();
        loadTenants();
    } catch (err) {
        showAlert('Fehler: ' + err.message, 'error');
    }
}

async function deactivateTenant() {
    if (!currentTenantId) return;
+
+   // Validate current status
+   if (currentTenantStatus === 'suspended') {
+       showAlert('Tierheim ist bereits deaktiviert', 'info');
+       return;
+   }
+
+   if (currentTenantStatus === 'deleted') {
+       showAlert('Gelöschte Tierheime können nicht deaktiviert werden', 'error');
+       return;
+   }
+
    if (!confirm('Tierheim wirklich deaktivieren? Benutzer werden sich nicht mehr anmelden können.')) return;

    try {
        await centralAPI.deactivateTenant(currentTenantId);
        showAlert('Tierheim deaktiviert', 'success');
+       currentTenantStatus = 'suspended'; // Update cached status
        closeModal();
        loadTenants();
    } catch (err) {
        showAlert('Fehler: ' + err.message, 'error');
    }
}
```

---

## Bug #9: Missing Loading State in Marketing Tab Operations

**Severity:** LOW

**Description:**
The marketing management page (`marketing.html`) performs various async operations (creating campaigns, toggling referral codes, approving references) but does not show loading indicators during these operations. Users may click buttons multiple times thinking the first click didn't work, causing duplicate requests.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/marketing.html`
- Functions: `saveCampaign`, `saveReferralCode`, `toggleReferralCodeStatus`, `approveReference`
- Lines: 328-538

**Steps to Reproduce:**
1. Navigate to `/central/marketing.html`
2. Create new campaign with slow network connection
3. Click "Speichern" button
4. No visual feedback that request is processing
5. User clicks again → duplicate request sent

**Impact:**
- **Duplicate operations**: Users create duplicate campaigns/codes
- **Poor UX**: No feedback during slow operations
- **Data inconsistency**: Duplicate referral codes may cause issues

**Fix:**

Add loading states to all buttons:
```diff
// internal/static/central/marketing.html, line 328
async function saveCampaign() {
+   const saveBtn = document.querySelector('#campaign-modal .btn-primary');
+   saveBtn.disabled = true;
+   saveBtn.textContent = 'Speichern...';
+
    const id = document.getElementById('campaign-id').value;
    const data = {
        type: document.getElementById('campaign-type').value,
        name: document.getElementById('campaign-name').value,
        description: document.getElementById('campaign-description').value || null,
        config: document.getElementById('campaign-config').value || null,
        start_date: document.getElementById('campaign-start').value || null,
        end_date: document.getElementById('campaign-end').value || null,
        is_active: document.getElementById('campaign-active').checked
    };

    try {
        if (id) {
            await updateCampaign(id, data);
        } else {
            await createCampaign(data);
        }
        closeCampaignModal();
        showAlert('Kampagne gespeichert');
        await loadAll();
    } catch (err) {
        showAlert('Fehler: ' + err.message, 'error');
+   } finally {
+       saveBtn.disabled = false;
+       saveBtn.textContent = 'Speichern';
    }
}
```

Apply similar pattern to:
- `saveReferralCode()` (line 429)
- `toggleReferralCodeStatus()` (line 454)
- `approveReference()` (line 520)
- `deleteCampaignConfirm()` (line 354)
- `deleteReferralCodeConfirm()` (line 463)
- `deleteReferenceConfirm()` (line 530)

Generic helper function:
```diff
+// Add to central.js
+async function withLoadingState(button, asyncFunc) {
+   const originalText = button.textContent;
+   button.disabled = true;
+   button.textContent = originalText + '...';
+   try {
+       await asyncFunc();
+   } finally {
+       button.disabled = false;
+       button.textContent = originalText;
+   }
+}
```

---

## Bug #10: Inconsistent Date Formatting Across Pages

**Severity:** LOW

**Description:**
The central admin dashboard uses two different date formatting functions (`formatDate` and `formatDateTime`) but applies them inconsistently:
- `index.html` uses `formatDate` for tenant creation (line 156)
- `tenants.html` uses `formatDate` for creation, `formatDateTime` for update (lines 278-279)
- `users.html` uses `formatDate` for user creation (line 132)

The `last_activity_at` field in the admins list uses `formatDate` (line 193 in `index.html`) which loses time information, making it impossible to see when admins were last active during the same day.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/index.html`
- Line: 193
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/assets/js/central.js`
- Functions: `formatDate`, `formatDateTime`
- Lines: 164-177

**Steps to Reproduce:**
1. Navigate to central admin dashboard
2. Look at "Central Administratoren" section
3. "Letzte Aktivität" column shows only date: "27.12.2025"
4. Cannot tell if admin was active at 9am or 11pm today

**Impact:**
- **Lost information**: Time component discarded for activity timestamps
- **Poor monitoring**: Cannot track admin activity patterns
- **Inconsistency**: Different pages format same fields differently

**Fix:**

Use consistent formatting:
```diff
// internal/static/central/index.html, line 193
<td>
-   <td>${formatDate(a.last_activity_at)}</td>
+   <td>${formatDateTime(a.last_activity_at)}</td>
</td>
```

```diff
// internal/static/central/users.html, line 185
<td>
-   <td>${formatDate(a.last_activity_at)}</td>
+   <td>${formatDateTime(a.last_activity_at)}</td>
</td>
```

Improve date formatting with relative times:
```diff
// internal/static/central/assets/js/central.js
+function formatDateTimeRelative(dateStr) {
+   if (!dateStr) return '-';
+   const date = new Date(dateStr);
+   const now = new Date();
+   const diffMs = now - date;
+   const diffMins = Math.floor(diffMs / 60000);
+
+   // Less than 1 hour ago - show relative
+   if (diffMins < 60) {
+       return diffMins < 1 ? 'Gerade eben' : `Vor ${diffMins} Min.`;
+   }
+
+   // Less than 24 hours ago - show time today
+   if (diffMs < 86400000 && date.getDate() === now.getDate()) {
+       return 'Heute ' + date.toLocaleTimeString('de-DE', {
+           hour: '2-digit',
+           minute: '2-digit'
+       });
+   }
+
+   // Otherwise show full date and time
+   return formatDateTime(dateStr);
+}
```

---

## Bug #11: External Link Missing Target Security Attribute

**Severity:** MEDIUM

**Description:**
The tenants table in `tenants.html` (line 190) includes external links to tenant websites using `target="_blank"` but is missing the `rel="noopener noreferrer"` attribute. This creates a security vulnerability where the opened page gains access to the window.opener object and can potentially redirect the admin dashboard to a phishing site.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/tenants.html`
- Line: 190

**Steps to Reproduce:**
1. Create malicious tenant with website: `https://evil.com`
2. On evil.com, add code: `window.opener.location = 'https://fake-gassigeher.org/login'`
3. Central admin views tenant list and clicks external link
4. New tab opens evil.com
5. Original tab redirects to phishing site
6. Admin enters credentials on fake login page

**Impact:**
- **Phishing risk**: Admins can be redirected to fake login pages
- **Security**: Opened page can read/modify opener window
- **OWASP Top 10**: A05:2021 - Security Misconfiguration

**Fix:**

Add security attributes to all external links:
```diff
// internal/static/central/tenants.html, line 190
<td>
    <code>${escapeHtml(t.slug)}</code>
-   <a href="https://${escapeHtml(t.slug)}.gassigeher.org" target="_blank" class="external-link" title="Besuchen">↗</a>
+   <a href="https://${escapeHtml(t.slug)}.gassigeher.org" target="_blank" rel="noopener noreferrer" class="external-link" title="Besuchen">↗</a>
</td>
```

Also check other external links:
```bash
# Search for target="_blank" without rel attribute
grep -n 'target="_blank"' internal/static/central/*.html | grep -v 'rel='
```

Found in:
- `index.html` line 91: Landing page link (should add rel)
- `marketing.html` line 507: Website URL link (should add rel)

```diff
// index.html, line 91
-<a href="/landing/" class="btn btn-secondary" target="_blank">Landing Page</a>
+<a href="/landing/" class="btn btn-secondary" target="_blank" rel="noopener noreferrer">Landing Page</a>
```

```diff
// marketing.html, line 507 (in renderReferences)
<td>
-   ${r.website_url ? `<a href="${escapeHtml(r.website_url)}" target="_blank">Link</a>` : '-'}
+   ${r.website_url ? `<a href="${escapeHtml(r.website_url)}" target="_blank" rel="noopener noreferrer">Link</a>` : '-'}
</td>
```

---

## Bug #12: Logout Function Missing Token Cleanup

**Severity:** LOW

**Description:**
The `logout()` function in `central.js` (line 14-17) only removes the JWT token from localStorage but does not:
1. Clear cached data (tenantActivityData in tenants.html, campaigns/references in marketing.html)
2. Cancel in-flight API requests
3. Clear any impersonation state

This could lead to:
- Memory leaks if user logs out and back in multiple times
- Stale data shown if localStorage persists across sessions
- Security concern if impersonation data remains in localStorage

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/static/central/assets/js/central.js`
- Function: `logout`
- Lines: 14-17

**Steps to Reproduce:**
1. Login as central admin
2. Navigate to marketing page (loads campaigns data)
3. Open browser localStorage panel
4. See `gassigeher_token` present
5. Click logout
6. Token removed but other data may remain in memory
7. Login again as different admin
8. Potentially see cached data from previous session

**Impact:**
- **Memory leaks**: Cached objects not garbage collected
- **Security**: Impersonation state may persist
- **Confusion**: Stale data shown after re-login
- **Privacy**: Previous admin's cached data visible to next admin

**Fix:**

Comprehensive logout with cleanup:
```diff
// internal/static/central/assets/js/central.js, line 14
function logout() {
+   // Clear all gassigeher-related items from localStorage
    localStorage.removeItem(TOKEN_KEY);
+   localStorage.removeItem('gassigeher_impersonating');
+   localStorage.removeItem('gassigeher_user');
+
+   // Clear any cached data in memory (if this script has references)
+   if (typeof tenantActivityData !== 'undefined') {
+       tenantActivityData = {};
+   }
+   if (typeof campaigns !== 'undefined') {
+       campaigns = [];
+   }
+   if (typeof referralCodes !== 'undefined') {
+       referralCodes = [];
+   }
+   if (typeof references !== 'undefined') {
+       references = [];
+   }
+
+   // Cancel any in-flight activity load
+   if (typeof activityLoadAbortController !== 'undefined' && activityLoadAbortController) {
+       activityLoadAbortController.abort();
+   }
+
    window.location.href = '/login.html';
}
```

Alternatively, add event listener for storage clear:
```diff
// On logout, dispatch custom event that pages can listen to
function logout() {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem('gassigeher_impersonating');
+
+   // Dispatch logout event for pages to clean up
+   window.dispatchEvent(new CustomEvent('gassigeher:logout'));
+
    window.location.href = '/login.html';
}
```

Then in each page:
```javascript
// Listen for logout event
window.addEventListener('gassigeher:logout', () => {
    // Clear page-specific data
    tenantActivityData = {};
    campaigns = [];
    // etc...
});
```

---

## Statistics

- **Critical:** 0 bugs
- **High:** 3 bugs (Authorization/privilege issues)
- **Medium:** 4 bugs (XSS, missing validation, security attributes)
- **Low:** 5 bugs (UI issues, error handling, consistency)

---

## Recommendations

### Immediate Actions (High Priority)

1. **Fix Bug #1**: Add `is_central_admin` field to user search results to prevent confusion and audit log pollution
2. **Fix Bug #2**: Implement server-side validation for tenant names to prevent XSS attacks
3. **Fix Bug #3**: Use dynamic domain extraction for impersonation redirects to support staging/development
4. **Fix Bug #4**: Add frontend validation before promotion requests to prevent unnecessary API calls
5. **Fix Bug #11**: Add `rel="noopener noreferrer"` to all external links to prevent phishing

### Short-term Improvements (Medium Priority)

6. **Fix Bug #5**: Prevent self-demotion UI by checking current user ID
7. **Fix Bug #8**: Validate tenant status before activation/deactivation
8. **Fix Bug #6**: Add comprehensive error handling to export function

### Long-term Enhancements (Low Priority)

9. **Fix Bug #7**: Implement request cancellation for activity data loading
10. **Fix Bug #9**: Add loading states to all async operations
11. **Fix Bug #10**: Standardize date formatting across all pages
12. **Fix Bug #12**: Implement comprehensive logout with memory cleanup

### General Security Improvements

1. **Add Content-Security-Policy headers** to prevent XSS attacks
2. **Implement rate limiting** on central admin endpoints (especially impersonation)
3. **Add audit logging UI** to view all central admin actions
4. **Implement session timeout** for idle central admin sessions
5. **Add 2FA requirement** for central admin accounts
6. **Implement IP whitelisting** for central admin access

### Code Quality Improvements

1. **Extract common validation logic** into reusable functions
2. **Create TypeScript definitions** for API responses
3. **Add unit tests** for frontend validation functions
4. **Implement E2E tests** for critical admin flows (impersonation, tenant activation)
5. **Add JSDoc comments** to all functions
6. **Use consistent error handling** pattern across all pages

### UX Improvements

1. **Add breadcrumbs** for navigation context
2. **Implement keyboard shortcuts** (e.g., ESC to close modals)
3. **Add bulk operations** (activate/deactivate multiple tenants)
4. **Implement data export** in multiple formats (JSON, CSV, Excel)
5. **Add advanced filtering** (by creation date, activity level, etc.)
6. **Implement pagination** for large tenant lists
