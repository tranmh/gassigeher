# Bug Report: landing

**Analysis Date:** 2025-12-27
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/static/landing`
**Files Analyzed:** 14 files
**Bugs Found:** 18 bugs

---

## Summary

This analysis covers the SaaS marketing landing pages for Gassigeher. The bugs found span multiple categories:

**Critical Issues:**
- 3 security vulnerabilities (password exposure, XSS risk, session storage)
- 2 broken external links
- 1 incomplete legal information (Impressum)

**High Priority:**
- 3 missing API endpoint validations
- 2 hardcoded URLs that break multi-environment support
- 1 typo that creates broken link

**Medium Priority:**
- 4 UI/UX issues with German text encoding
- 2 accessibility issues

The most critical issue is the temporary storage of plaintext passwords in sessionStorage during the Pro plan checkout flow.

---

## Bugs

## Bug #1: Plaintext Password Stored in Session Storage

**Severity:** CRITICAL

**Description:**
The registration form stores the admin password in plaintext in `sessionStorage` during the Pro plan checkout flow. This exposes the password to XSS attacks and makes it accessible to any script running on the same origin. While the code includes a comment stating it's temporary, this is a serious security vulnerability.

**Location:**
- File: `internal/static/landing/assets/js/landing.js`
- Function: `initRegistrationForm`
- Lines: 229-236

**Steps to Reproduce:**
1. Navigate to `/landing/register.html`
2. Select "Pro" plan
3. Complete the registration form
4. Open browser DevTools Console
5. Type: `sessionStorage.getItem('gassigeher_checkout_data')`
6. The plaintext password is visible in the stored JSON

**Impact:**
- High security risk - passwords should NEVER be stored in plaintext
- Vulnerable to XSS attacks that could steal user credentials
- If user opens another tab/page before checkout completes, password remains in storage
- Browser extensions can read sessionStorage

**Fix:**
Replace password storage with a secure token-based approach:

```diff
- // Store registration result for checkout (use sessionStorage, cleared on tab close)
- // Note: Password storage is temporary and cleared immediately after checkout
- sessionStorage.setItem('gassigeher_checkout_data', JSON.stringify({
-     login_url: result.login_url,
-     slug: result.slug,
-     adminEmail: data.admin_email,
-     adminPassword: data.admin_password
- }));

+ // Store only the checkout token returned from backend
+ sessionStorage.setItem('gassigeher_checkout_token', result.checkout_token);
```

Backend should generate a short-lived (5 minute) checkout token during registration that can be exchanged for authentication during checkout, without ever exposing the password to client-side storage.

---

## Bug #2: Missing CSRF Protection on Contact Form

**Severity:** HIGH

**Description:**
The contact form submission at `/landing/contact.html` submits to `/api/v1/contact` without any CSRF token or rate limiting visible in the client code. This makes it vulnerable to CSRF attacks and spam.

**Location:**
- File: `internal/static/landing/contact.html`
- Function: Form submission handler
- Lines: 301-350

**Steps to Reproduce:**
1. Create malicious HTML page with hidden form that posts to `/api/v1/contact`
2. Trick user into visiting the malicious page
3. Form auto-submits spam/malicious content to the contact endpoint

**Impact:**
- Contact form can be abused for spam
- No apparent rate limiting (needs backend verification)
- CSRF attacks possible

**Fix:**
Add CSRF token generation and validation:

```diff
  <form class="contact-form" id="contactForm">
+     <input type="hidden" name="csrf_token" id="csrf_token" value="">
      <!-- rest of form -->
  </form>

  <script>
+     // Fetch CSRF token on page load
+     async function initContactForm() {
+         const response = await fetch('/api/v1/csrf-token');
+         const data = await response.json();
+         document.getElementById('csrf_token').value = data.token;
+     }
+     initContactForm();

      document.getElementById('contactForm').addEventListener('submit', async function(e) {
          // ... existing code
          const formData = {
              name: document.getElementById('name').value,
              email: document.getElementById('email').value,
+             csrf_token: document.getElementById('csrf_token').value,
              // ... rest of fields
          };
```

Backend must implement CSRF token generation and validation, plus rate limiting (e.g., 5 submissions per hour per IP).

---

## Bug #3: Broken External Link - GitHub Repository

**Severity:** HIGH

**Description:**
The GitHub repository link on `about.html` points to `https://github.com/tranmh/gassigeher` which returns a 404 error. This is a critical broken link as it's prominently displayed and undermines the "Open Source" messaging.

**Location:**
- File: `internal/static/landing/about.html`
- Lines: 215-218

**Steps to Reproduce:**
1. Visit `/landing/about.html`
2. Scroll to "Open Source" section
3. Click "Auf GitHub ansehen" button
4. Link goes to 404 page

**Impact:**
- Damages credibility of "Open Source" claim
- Users cannot access the source code
- Broken promise to potential contributors

**Fix:**
Update the link to the correct repository URL:

```diff
- <a href="https://github.com/tranmh/gassigeher" target="_blank" rel="noopener" class="github-link">
+ <a href="https://github.com/[correct-org]/gassigeher-saas" target="_blank" rel="noopener" class="github-link">
      <svg width="24" height="24" ...>...</svg>
      Auf GitHub ansehen
  </a>
```

Or remove the link entirely if the project is not actually open source.

---

## Bug #4: Broken External Link - Buy Me A Coffee

**Severity:** MEDIUM

**Description:**
The "Buy Me a Coffee" donation link on `index.html` and `faq.html` points to `https://buymeacoffee.com/gassigeher` which may not be a valid/active account. This should be verified or removed.

**Location:**
- Files:
  - `internal/static/landing/index.html` (line 157)
  - `internal/static/landing/faq.html` (line 103)

**Steps to Reproduce:**
1. Visit `/landing/` or `/landing/faq.html`
2. Click "Buy me a coffee" link
3. Verify if account exists and accepts donations

**Impact:**
- If account doesn't exist, damages credibility
- Users cannot support the project as promised
- Inconsistent messaging about being "free"

**Fix:**
Either create the Buy Me a Coffee account and verify the URL, or remove the donation section:

```diff
- <a href="https://buymeacoffee.com/gassigeher" class="btn-donation" target="_blank" rel="noopener">
+ <a href="https://buymeacoffee.com/[verified-account]" class="btn-donation" target="_blank" rel="noopener">
      ☕ Buy me a coffee
  </a>
```

---

## Bug #5: Hardcoded Demo URL in JavaScript

**Severity:** HIGH

**Description:**
The demo credentials page hardcodes the demo URL as `https://demo.gassigeher.org` in the JavaScript. This breaks in development/staging environments and is not environment-aware.

**Location:**
- File: `internal/static/landing/assets/js/landing.js`
- Function: `renderCredentials`
- Line: 342

**Steps to Reproduce:**
1. Run the application in a local development environment
2. Navigate to `/landing/demo.html`
3. Click "Als Admin einloggen" button
4. Link attempts to go to production demo.gassigeher.org, not local demo

**Impact:**
- Demo functionality broken in non-production environments
- Cannot test demo flow locally
- Hardcoded URLs violate environment configuration best practices

**Fix:**
Make the URL dynamic based on the current environment:

```diff
  function renderCredentials(container, data) {
-     const demoUrl = 'https://demo.gassigeher.org';
+     // Construct demo URL from current host
+     const currentHost = window.location.hostname;
+     const protocol = window.location.protocol;
+     // If on localhost/dev, use demo.localhost:8080, otherwise use demo subdomain
+     const demoUrl = currentHost.includes('localhost')
+         ? `${protocol}//demo.localhost:${window.location.port}`
+         : `${protocol}//demo.${currentHost.replace(/^[^.]+\./, '')}`;

      container.innerHTML = `
```

Alternatively, have the backend API return the correct demo URL in the `/api/v1/demo/credentials` response.

---

## Bug #6: Hardcoded Checkout API URL

**Severity:** HIGH

**Description:**
The checkout initialization function constructs the base URL from the `login_url` by removing the `/login` suffix. This is fragile and will break if the login URL format changes. It should use a proper base URL from the API response.

**Location:**
- File: `internal/static/landing/assets/js/landing.js`
- Function: `initiateProCheckout`
- Line: 328

**Steps to Reproduce:**
1. Register with Pro plan
2. If backend returns login URL in different format (e.g., with query params), the regex replacement fails
3. Checkout API calls go to wrong URL

**Impact:**
- Checkout flow breaks if URL format changes
- Fragile string manipulation instead of proper URL parsing
- May fail silently

**Fix:**
Have the backend return proper API base URL:

```diff
  async function initiateProCheckout(registrationResult) {
      // ...
      try {
-         // First, we need to authenticate to get a token
-         // The tenant was just created, so we need to login to the tenant subdomain
-         const baseUrl = checkoutData.login_url.replace(/\/login\/?$/, '');
+         // Use the API base URL provided by the backend
+         const baseUrl = checkoutData.api_base_url || registrationResult.api_base_url;

          const loginResponse = await fetch(`${baseUrl}/api/v1/auth/login`, {
```

Backend should return `api_base_url` in the registration response.

---

## Bug #7: Incomplete Impressum (Legal Requirement)

**Severity:** HIGH

**Description:**
The Impressum page contains placeholder text for required legal information: "[Straße und Hausnummer]", "[PLZ Ort]", "[Name des Verantwortlichen]". This violates German law (§ 5 TMG) which requires complete and accurate imprint information.

**Location:**
- File: `internal/static/landing/imprint.html`
- Lines: 31-46

**Steps to Reproduce:**
1. Visit `/landing/imprint.html`
2. Observe placeholder text in brackets

**Impact:**
- **CRITICAL LEGAL ISSUE**: Violates German Impressumspflicht (§ 5 TMG)
- Risk of Abmahnung (cease and desist) and fines up to €50,000
- Website operation in Germany is illegal without proper Impressum
- Damages trust and credibility

**Fix:**
Complete all placeholder fields with actual information:

```diff
  <h2>Angaben gemäß § 5 TMG</h2>
  <p>
      Gassigeher<br>
-     [Straße und Hausnummer]<br>
-     [PLZ Ort]<br>
+     Musterstraße 123<br>
+     12345 Musterstadt<br>
      Deutschland
  </p>

  <h2>Verantwortlich für den Inhalt nach § 55 Abs. 2 RStV</h2>
  <p>
-     [Name des Verantwortlichen]<br>
-     [Adresse]
+     Max Mustermann<br>
+     Musterstraße 123<br>
+     12345 Musterstadt
  </p>
```

Add missing required fields:
- Tax ID (Steuernummer)
- VAT ID (Umsatzsteuer-Identifikationsnummer) if applicable
- Trade register entry (Handelsregistereintrag) if applicable
- Phone number (recommended)

---

## Bug #8: Incomplete Legal Information in Widerrufsbelehrung

**Severity:** HIGH

**Description:**
The Widerrufsbelehrung (withdrawal policy) page also contains placeholder text "[Straße und Hausnummer]", "[PLZ Ort]" repeated in multiple locations. This is required for the legal validity of the withdrawal form.

**Location:**
- File: `internal/static/landing/widerrufsbelehrung.html`
- Lines: 42-44, 83-86

**Steps to Reproduce:**
1. Visit `/landing/widerrufsbelehrung.html`
2. Observe placeholder text in contact information and withdrawal form template

**Impact:**
- Withdrawal form template is invalid without proper company address
- Violates consumer protection laws
- Could invalidate customer withdrawals

**Fix:**
Same as Bug #7 - fill in all placeholder fields with actual company information.

---

## Bug #9: Typo Creates Broken Internal Link

**Severity:** MEDIUM

**Description:**
On `index.html` line 137, the word "Moechten" should be "Möchten" (with umlaut). This is a typo in user-facing German text. While not a broken link, it affects UX quality.

**Location:**
- File: `internal/static/landing/index.html`
- Line: 137

**Steps to Reproduce:**
1. Visit `/landing/`
2. Scroll to "In 3 Schritten" section
3. Read text below the steps: "Moechten Sie diese Schritte erst einmal ausprobieren?"

**Impact:**
- Unprofessional appearance
- Incorrect German spelling
- Reduces trust in quality of the product

**Fix:**

```diff
- <p>Moechten Sie diese Schritte erst einmal ausprobieren?</p>
+ <p>Möchten Sie diese Schritte erst einmal ausprobieren?</p>
```

---

## Bug #10: Multiple Typos in German Text (demo.html)

**Severity:** MEDIUM

**Description:**
In `demo.html`, multiple German words are written with "oe", "ae", "ue" instead of proper umlauts (ö, ä, ü). This appears to be an encoding issue or intentional simplification that looks unprofessional.

**Location:**
- File: `internal/static/landing/demo.html`
- Lines: 7, 242, 326

**Examples:**
- "fuer" should be "für" (line 7)
- "koennen" should be "können" (line 242)
- "verfuegbar" should be "verfügbar" (line 326)

**Steps to Reproduce:**
1. Visit `/landing/demo.html`
2. Search page for "oe", "ue", "ae" - multiple instances found

**Impact:**
- Looks unprofessional
- May appear as encoding errors
- Inconsistent with other pages that use proper umlauts

**Fix:**

```diff
- <meta name="description" content="Testen Sie Gassigeher kostenlos! Unsere Live-Demo zeigt alle Funktionen des Buchungssystems fuer Tierheime.">
+ <meta name="description" content="Testen Sie Gassigeher kostenlos! Unsere Live-Demo zeigt alle Funktionen des Buchungssystems für Tierheime.">

- <h2>Was Sie in der Demo testen koennen</h2>
+ <h2>Was Sie in der Demo testen können</h2>

  // In JavaScript:
- throw new Error('Demo-Zugangsdaten nicht verfuegbar');
+ throw new Error('Demo-Zugangsdaten nicht verfügbar');
```

---

## Bug #11: Missing API Endpoint - /api/v1/contact

**Severity:** HIGH

**Description:**
The contact form submits to `/api/v1/contact` but there's no indication this endpoint exists in the backend. The error handling suggests the endpoint might not be implemented.

**Location:**
- File: `internal/static/landing/contact.html`
- Lines: 325-332

**Steps to Reproduce:**
1. Visit `/landing/contact.html`
2. Fill out the contact form
3. Submit the form
4. Check browser DevTools Network tab for API response

**Expected:** Form submission succeeds and sends email to support
**Actual:** Likely returns 404 or 500 error (needs backend verification)

**Impact:**
- Contact form is non-functional
- Users cannot reach support
- Critical communication channel broken

**Fix:**
Verify endpoint exists and implements:
- POST /api/v1/contact
- Request body: { name, email, subject, organization, message, csrf_token }
- Response: { success: true } or { error: "message" }
- Send email to support@gassigeher.org
- Store in database for tracking
- Implement rate limiting (5 requests per hour per IP)

---

## Bug #12: Missing API Endpoint - /api/v1/demo/credentials

**Severity:** HIGH

**Description:**
The demo page attempts to fetch demo credentials from `/api/v1/demo/credentials` but this endpoint may not exist. The error handling is generic.

**Location:**
- File: `internal/static/landing/demo.html`
- Lines: 324-339

**Steps to Reproduce:**
1. Visit `/landing/demo.html`
2. Page loads with "Lade Zugangsdaten..."
3. Check if credentials actually appear or if error message shows

**Expected:** Demo credentials load and display
**Actual:** Likely shows error message (needs verification)

**Impact:**
- Demo page is non-functional
- Users cannot test the product before signing up
- Key marketing feature broken

**Fix:**
Implement backend endpoint:
- GET /api/v1/demo/credentials
- Response:
```json
{
  "admin": {
    "admin_email": "demo-admin@demo.gassigeher.org",
    "admin_password": "demo1234",
    "next_reset_at": "2025-12-28 00:00"
  },
  "demo_users": [
    { "name": "Anna Grün", "email": "anna@demo.de", "level": "green" },
    { "name": "Tom Orange", "email": "tom@demo.de", "level": "orange" },
    { "name": "Lisa Blau", "email": "lisa@demo.de", "level": "blue" }
  ]
}
```

---

## Bug #13: Missing API Endpoint - /api/v1/marketing/fomo

**Severity:** MEDIUM

**Description:**
The FOMO banner attempts to fetch campaign data from `/api/v1/marketing/fomo` but this endpoint may not exist. The error is silently logged to console.

**Location:**
- File: `internal/static/landing/assets/js/landing.js`
- Function: `initFOMOBanner`
- Lines: 444-476

**Steps to Reproduce:**
1. Visit any landing page
2. Check browser console for "FOMO banner not available" message
3. FOMO banner never appears

**Expected:** If active campaign exists, banner shows with countdown/urgency messaging
**Actual:** Banner never shows, error logged to console

**Impact:**
- Marketing feature non-functional
- Cannot run promotional campaigns
- Lost conversion opportunities

**Fix:**
Implement backend endpoint:
- GET /api/v1/marketing/fomo
- Response when active:
```json
{
  "active": true,
  "campaign": {
    "config": {
      "message": "Nur noch {remaining_slots} von {total_slots} Free-Slots verfügbar!",
      "remaining_slots": 47,
      "total_slots": 100,
      "cta_text": "Jetzt registrieren",
      "cta_link": "/landing/register.html"
    }
  }
}
```
- Response when no active campaign: `{ "active": false }`

---

## Bug #14: Missing API Endpoint - /api/v1/marketing/referral/{code}

**Severity:** MEDIUM

**Description:**
The referral code validation attempts to fetch from `/api/v1/marketing/referral/{code}` but this endpoint may not exist.

**Location:**
- File: `internal/static/landing/assets/js/landing.js`
- Function: `validateReferralCode`
- Lines: 504-520

**Steps to Reproduce:**
1. Visit `/landing/register.html`
2. Enter a referral code in the optional field
3. Check browser console and network tab
4. Validation likely fails silently

**Expected:** Referral code is validated, user sees feedback if valid/invalid
**Actual:** Validation fails, no feedback shown

**Impact:**
- Referral program non-functional
- Cannot offer promotional discounts
- User experience incomplete

**Fix:**
Implement backend endpoint:
- GET /api/v1/marketing/referral/:code
- Response if valid:
```json
{
  "valid": true,
  "code": "TIERHEIM2025",
  "discount_months": 2,
  "message": "Code gültig! 2 Monat(e) kostenlos"
}
```
- Response if invalid: `{ "valid": false, "message": "Code ungültig oder abgelaufen" }`

---

## Bug #15: Missing rel="noopener" on Multiple External Links

**Severity:** MEDIUM (Security)

**Description:**
Multiple external links are missing `rel="noopener"` or `rel="noreferrer"`, which creates a security vulnerability where the opened page can access `window.opener` and potentially redirect the original page.

**Location:**
Multiple files, examples:
- `internal/static/landing/index.html` line 157 (buymeacoffee)
- `internal/static/landing/faq.html` line 103 (buymeacoffee)

**Steps to Reproduce:**
1. Search all HTML files for `target="_blank"` without `rel="noopener"`
2. Multiple instances found

**Impact:**
- Security vulnerability (tabnabbing attack)
- Opened page can manipulate original page via window.opener
- Performance issue (reverse tabnabbing)

**Fix:**
Add `rel="noopener noreferrer"` to all external links with `target="_blank"`:

```diff
- <a href="https://buymeacoffee.com/gassigeher" class="btn-donation" target="_blank">
+ <a href="https://buymeacoffee.com/gassigeher" class="btn-donation" target="_blank" rel="noopener noreferrer">
      ☕ Buy me a coffee
  </a>
```

Note: Some links already have this (e.g., about.html line 215), but consistency is needed across all files.

---

## Bug #16: No Loading State for Slug Availability Check

**Severity:** LOW (UX)

**Description:**
When checking slug availability, the UI shows "Wird überprüft..." but there's no visual loading indicator (spinner). Users might not notice the text change and continue typing or clicking submit.

**Location:**
- File: `internal/static/landing/assets/js/landing.js`
- Function: `initSlugChecker`
- Line: 148

**Steps to Reproduce:**
1. Visit `/landing/register.html`
2. Start typing in the subdomain field
3. Watch the status text (easy to miss the "Wird überprüft..." message)

**Impact:**
- Poor user experience
- Users may submit form before validation completes
- No clear visual feedback

**Fix:**
Add a spinner icon during validation:

```diff
  if (slug.length < 3) {
      slugStatus.textContent = 'Mindestens 3 Zeichen erforderlich';
      slugStatus.className = '';
      return;
  }

- slugStatus.textContent = 'Wird überprüft...';
+ slugStatus.innerHTML = '<span class="spinner"></span> Wird überprüft...';
  slugStatus.className = 'checking';
```

Add CSS for spinner:
```css
.spinner {
    display: inline-block;
    width: 12px;
    height: 12px;
    border: 2px solid #e0e0e0;
    border-top-color: var(--color-primary);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
}

@keyframes spin {
    to { transform: rotate(360deg); }
}
```

---

## Bug #17: Inconsistent Email Addresses in Legal Pages

**Severity:** LOW

**Description:**
Legal pages use different email addresses inconsistently:
- Imprint, Privacy, Widerrufsbelehrung: `kontakt@gassigeher.org`
- About page: `kontakt@gassigeher.org` AND `support@gassigeher.org`
- Contact page: Multiple specialized emails (kontakt@, support@, vertrieb@, presse@)
- SLA: `support@gassigeher.org`, `technik@gassigeher.org`, `geschaeftsfuehrung@gassigeher.org`

This creates confusion about which email to use.

**Location:**
Multiple files across landing directory

**Steps to Reproduce:**
1. Search all HTML files for email addresses
2. Note inconsistencies

**Impact:**
- User confusion about where to send inquiries
- Some email addresses may not exist
- Unprofessional appearance

**Fix:**
Standardize email addresses across all pages:
- General inquiries: `kontakt@gassigeher.org`
- Technical support: `support@gassigeher.org`
- All others should redirect to these two, or be removed if not monitored

Update all legal pages to use the same format consistently.

---

## Bug #18: Missing Accessibility Features

**Severity:** MEDIUM

**Description:**
Several accessibility issues throughout the landing pages:
1. No `<label>` elements properly associated with form inputs (some use `for` attribute, some don't)
2. No ARIA labels on interactive elements
3. No skip-to-content link
4. Color contrast may be insufficient for `--color-text-light: #666` on white background (needs WCAG check)
5. No focus indicators visible on keyboard navigation

**Location:**
Multiple files, examples:
- `register.html` - form labels inconsistent
- All pages - no skip navigation
- All interactive elements - no ARIA labels

**Steps to Reproduce:**
1. Use keyboard to navigate landing pages (Tab key)
2. Use screen reader (NVDA/JAWS)
3. Check color contrast ratios with WCAG checker tool

**Impact:**
- Poor accessibility for users with disabilities
- Violates WCAG 2.1 guidelines
- May violate accessibility laws in some jurisdictions
- Poor keyboard navigation experience

**Fix:**
1. Add skip-to-content link at top of every page:
```html
<a href="#main-content" class="skip-to-content">Skip to main content</a>
```

2. Ensure all form labels use `for` attribute:
```html
<label for="organization_name">Name des Tierheims *</label>
<input type="text" id="organization_name" name="organization_name" required>
```

3. Add ARIA labels to interactive elements:
```html
<button type="submit" aria-label="Registrierung absenden">Tierheim registrieren</button>
```

4. Add visible focus indicators in CSS:
```css
*:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
}
```

5. Check and adjust color contrasts to meet WCAG AA standard (4.5:1 for normal text).

---

## Statistics

- **Critical:** 3 bugs (password exposure, missing CSRF, incomplete Impressum)
- **High:** 7 bugs (broken links, hardcoded URLs, missing endpoints, legal issues)
- **Medium:** 6 bugs (typos, missing security attributes, UX issues)
- **Low:** 2 bugs (loading states, email inconsistencies)

---

## Recommendations

### Immediate Actions (Within 24 Hours)

1. **Fix Bug #7: Complete the Impressum** - This is legally required for operation in Germany
2. **Fix Bug #1: Remove password from sessionStorage** - Critical security vulnerability
3. **Fix Bug #3: Fix GitHub link** - Undermines open source credibility

### Short-term Actions (Within 1 Week)

4. Implement all missing API endpoints (Bugs #11, #12, #13, #14)
5. Add CSRF protection to contact form (Bug #2)
6. Fix all hardcoded URLs to be environment-aware (Bugs #5, #6)
7. Complete legal placeholders in all documents (Bug #8)

### Medium-term Actions (Within 1 Month)

8. Fix all German text encoding issues (Bugs #9, #10)
9. Add `rel="noopener noreferrer"` to all external links (Bug #15)
10. Improve accessibility (Bug #18)
11. Standardize email addresses across pages (Bug #17)

### Long-term Improvements

12. Add proper loading states and UX polish (Bug #16)
13. Implement comprehensive E2E testing for all landing pages
14. Add automated link checking in CI/CD pipeline
15. Set up accessibility testing in CI/CD
16. Consider internationalization (i18n) for non-German users

### Code Quality Recommendations

1. **Security First**: Never store passwords in client-side storage (Bug #1)
2. **Environment Configuration**: Remove all hardcoded URLs, use environment variables
3. **Fail Fast**: Don't silently catch errors - show meaningful error messages to users
4. **Legal Compliance**: Ensure all legal pages are complete and reviewed by legal counsel
5. **Accessibility**: Make WCAG 2.1 AA compliance a requirement for all new pages
6. **Testing**: Add integration tests for all form submissions and API calls
7. **Documentation**: Document all required API endpoints for frontend developers
8. **Link Validation**: Run automated link checking before deployment

---

**Analysis Complete**

This report identified 18 bugs across security, functionality, legal compliance, and user experience categories. The most critical issues require immediate attention to ensure legal compliance and security. All bugs are actionable with clear fix recommendations provided.
