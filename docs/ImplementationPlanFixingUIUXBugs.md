# UI/UX Bug Fixes Implementation Plan

This document outlines the UI/UX bugs identified from user screenshots and codebase analysis, along with implementation plans to fix them.

---

## Bug Summary

| # | Issue | Severity | Affected Files |
|---|-------|----------|----------------|
| 1 | Admin navigation overflow on desktop | High | All admin-*.html, main.css |
| 2 | Inconsistent button border-radius | Medium | main.css, admin-users.html |
| 3 | Modal with undefined background color | High | dogs.html, main.css |
| 4 | Mobile calendar grid view shows empty content | High | calendar.html |
| 5 | Missing featured dogs on index.html | Feature | index.html, backend |

---

## Bug #1: Admin Navigation Overflow on Desktop

### Problem Description
The admin navigation has 12 menu items that overflow the header on desktop screens, causing links to be cut off. The mobile hamburger menu works correctly but desktop has no overflow handling.

**Current navigation items (12):**
1. Dashboard
2. Hunde
3. Buchungen
4. Gesperrte Tage
5. Level-Anfragen
6. Benutzer
7. Reaktivierungen
8. Buchungszeiten
9. Genehmigungen
10. Einstellungen
11. Benutzer-Bereich (area switcher)
12. Abmelden

### Root Cause
- Desktop navigation uses `display: flex` with `gap: var(--spacing-lg)` (1.76rem)
- No `flex-wrap` or overflow handling for smaller desktop screens
- Header is `position: sticky` with `overflow: visible` by default

### Solution Options

**Option A: Dropdown Menu (Recommended)**
Group related items into dropdown menus to reduce top-level items:
- Dashboard
- Hunde ▼ (Dogs submenu)
- Buchungen ▼ (Bookings, Genehmigungen, Buchungszeiten)
- Verwaltung ▼ (Benutzer, Level-Anfragen, Reaktivierungen)
- Einstellungen ▼ (Settings, Gesperrte Tage)
- Benutzer-Bereich | Abmelden

**Option B: Responsive Breakpoint**
Use hamburger menu on screens < 1200px instead of < 768px

**Option C: Horizontal Scroll with Icons**
Make nav scrollable horizontally with icon-only mode on medium screens

### Implementation (Option A - Dropdown)

```css
/* Add to main.css */
.admin-nav .nav-dropdown {
    position: relative;
}

.admin-nav .nav-dropdown-menu {
    display: none;
    position: absolute;
    top: 100%;
    left: 0;
    background: var(--header-bg);
    min-width: 200px;
    border-radius: var(--border-radius);
    box-shadow: 0 4px 12px rgba(0,0,0,0.2);
    z-index: 1001;
}

.admin-nav .nav-dropdown:hover .nav-dropdown-menu {
    display: block;
}

.admin-nav .nav-dropdown-menu a {
    display: block;
    padding: 12px 16px;
    border-bottom: 1px solid rgba(255,255,255,0.1);
}
```

### Files to Modify
- `frontend/assets/css/main.css` - Add dropdown styles
- All 10 `frontend/admin-*.html` files - Update navigation structure
- `frontend/js/nav-menu.js` - Add dropdown toggle for mobile

### Estimated Changes
- ~50 lines CSS
- ~30 lines JS
- ~20 lines per admin HTML file (10 files = 200 lines)

---

## Bug #2: Inconsistent Button Border-Radius

### Problem Description
Buttons have inconsistent border-radius values across the application:
- CSS variable: `--border-radius: 6px`
- `.btn` class: `border-radius: 8px` (hardcoded, ignores variable!)
- `.btn-promote`/`.btn-demote`: `border-radius: 4px` (inline in admin-users.html)
- `.badge-admin`/`.badge-super-admin`: `border-radius: 4px`

### Root Cause
The `.btn` class was updated to `8px` without using the CSS variable, and custom button styles in individual pages don't follow the design system.

### Solution
Standardize all border-radius values to use `var(--border-radius)`:

```css
/* main.css - Update .btn */
.btn {
    /* ... other styles ... */
    border-radius: var(--border-radius);  /* Was: 8px */
}

/* Add consistent button variants */
.btn-sm {
    padding: 8px 16px;
    font-size: 0.9rem;
    border-radius: var(--border-radius);
}
```

### Files to Modify
- `frontend/assets/css/main.css` - Fix `.btn` border-radius
- `frontend/admin-users.html` - Use `.btn` classes instead of custom styles

### Estimated Changes
- ~5 lines CSS
- ~10 lines HTML

---

## Bug #3: Modal with Undefined Background Color

### Problem Description
The booking modal in dogs.html has poor visibility because:
1. `.modal-content` uses `background-color: var(--dark-gray)`
2. `--dark-gray` is **NOT DEFINED** in the CSS variables (old dark theme was removed)
3. This causes the modal content to fall back to browser default or transparent

### Root Cause
CSS variable `--dark-gray` was removed when switching from dark theme to light theme, but references to it remain in inline styles.

### Current Modal CSS (dogs.html lines 126-166)
```css
.modal-content {
    background-color: var(--dark-gray);  /* UNDEFINED! */
    ...
}
.modal-close:hover {
    color: #fff;  /* White text assumes dark background */
}
```

### Solution
Update modal to use the light theme correctly:

```css
.modal-content {
    background-color: var(--card-bg);  /* Use defined card background */
    padding: 30px;
    border-radius: var(--border-radius);
    max-width: 500px;
    width: 90%;
    position: relative;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.modal-close {
    position: absolute;
    right: 15px;
    top: 15px;
    font-size: 28px;
    font-weight: bold;
    color: var(--text-gray);  /* Use defined gray */
    cursor: pointer;
}

.modal-close:hover {
    color: var(--error-red);  /* Red on hover */
}
```

### Files to Modify
- `frontend/dogs.html` - Update modal styles
- Consider moving modal styles to `main.css` for reuse

### Estimated Changes
- ~15 lines CSS

---

## Bug #4: Mobile Calendar Grid View Shows Empty

### Problem Description
On mobile devices, when "Rasteransicht" (grid view) is selected, the calendar shows empty content instead of the dog availability grid.

### Root Cause Analysis
Looking at calendar.html:

1. CSS media query at line 218:
```css
@media (max-width: 768px) {
    .calendar-grid {
        display: none;  /* Hidden on mobile by default */
    }
    .calendar-mobile {
        display: block;
    }
}
```

2. `switchView()` function (line 653):
```javascript
if (view === 'grid') {
    calendarGrid.style.display = 'block';  /* Shows wrapper, but... */
}
```

3. The issue: On mobile, the grid is hidden via CSS, and the JavaScript only shows the wrapper `.calendar-wrapper`, not the inner `.calendar-grid`. Additionally, the 15-column grid layout doesn't work on mobile.

### Solution

**Option A: Hide Grid View Button on Mobile (Recommended)**
Don't show the grid/list toggle on mobile since the grid isn't designed for small screens.

**Option B: Make Grid Responsive**
Reduce columns and simplify the grid for mobile view.

### Implementation (Option A)
```css
@media (max-width: 768px) {
    .view-toggle {
        display: none;  /* Hide view toggle on mobile */
    }
    .calendar-grid {
        display: none;  /* Keep grid hidden */
    }
    .calendar-mobile {
        display: block !important;  /* Always show mobile view */
    }
}
```

### Implementation (Option B - Alternative)
```javascript
function switchView(view) {
    const isMobile = window.innerWidth <= 768;

    if (view === 'grid' && isMobile) {
        // Show simplified grid or default to list on mobile
        showAlert('info', 'Rasteransicht ist auf mobilen Geräten nicht verfügbar');
        switchView('list');
        return;
    }
    // ... rest of function
}
```

### Files to Modify
- `frontend/calendar.html` - Update CSS media query and/or JavaScript

### Estimated Changes
- ~10 lines CSS or JS

---

## Feature #5: Featured Dogs on Index Page

### Problem Description
The index.html landing page doesn't show any dogs. User requested adding 3 featured dogs with admin control.

### Requirements
1. Display 3 featured dogs on the index.html page
2. Admin can select which dogs appear as featured
3. Dogs should be visually appealing with photos
4. Non-authenticated users can see the dogs but must register to book

### Solution Design

#### Database Changes
Add `is_featured` boolean field to dogs table:
```sql
ALTER TABLE dogs ADD COLUMN is_featured INTEGER DEFAULT 0;
```

Or use system_settings table for featured dog IDs:
```sql
INSERT INTO system_settings (key, value) VALUES ('featured_dog_ids', '1,2,3');
```

#### Backend Changes
- Add `GET /api/dogs/featured` public endpoint (no auth required)
- Returns up to 3 dogs where `is_featured = true` or from settings
- Admin endpoint to set featured dogs

#### Frontend Changes

**index.html - Add featured dogs section:**
```html
<section id="featured-dogs" class="container" style="padding: 60px 0;">
    <h2 class="text-center">Unsere Hunde</h2>
    <p class="text-center" style="color: var(--text-gray); margin-bottom: 30px;">
        Lernen Sie einige unserer vierbeinigen Freunde kennen
    </p>
    <div id="featured-dogs-grid" class="dog-grid">
        <!-- Dogs loaded via JavaScript -->
    </div>
    <div class="text-center" style="margin-top: 30px;">
        <a href="/register.html" class="btn">Jetzt registrieren</a>
    </div>
</section>
```

**Admin control (admin-dogs.html):**
- Add toggle/checkbox "Auf Startseite anzeigen" for each dog
- Limit to 3 featured dogs maximum

### Files to Modify
- `internal/database/migrations.go` - Add migration for featured field
- `internal/models/dog.go` - Add IsFeatured field
- `internal/repository/dog_repository.go` - Add GetFeaturedDogs method
- `internal/handlers/dog_handler.go` - Add GetFeaturedDogs handler
- `cmd/server/main.go` - Register public route
- `frontend/index.html` - Add featured dogs section
- `frontend/admin-dogs.html` - Add featured toggle
- `frontend/js/api.js` - Add getFeaturedDogs method

### Estimated Changes
- ~30 lines Go (backend)
- ~50 lines HTML
- ~30 lines JavaScript

---

## Implementation Sprints

### Sprint 1: Critical Fixes (High Priority)
**Goal: Fix visibility and functionality issues**

| Task | Files | Est. Time |
|------|-------|-----------|
| Fix modal background color | dogs.html | 15 min |
| Fix mobile calendar grid view | calendar.html | 20 min |
| Test on multiple screen sizes | - | 15 min |

### Sprint 2: Consistency Fixes (Medium Priority)
**Goal: Standardize UI across application**

| Task | Files | Est. Time |
|------|-------|-----------|
| Standardize button border-radius | main.css | 10 min |
| Update admin-users.html buttons | admin-users.html | 15 min |
| Audit other pages for inconsistencies | Various | 30 min |

### Sprint 3: Navigation Improvement (High Priority)
**Goal: Fix admin navigation overflow**

| Task | Files | Est. Time |
|------|-------|-----------|
| Design dropdown navigation structure | - | 20 min |
| Add CSS for dropdown menus | main.css | 30 min |
| Update all admin pages (10 files) | admin-*.html | 60 min |
| Add dropdown JS for mobile | nav-menu.js | 20 min |
| Test navigation on all screen sizes | - | 20 min |

### Sprint 4: Featured Dogs Feature
**Goal: Add featured dogs to index page**

| Task | Files | Est. Time |
|------|-------|-----------|
| Add database migration | migrations.go | 15 min |
| Update dog model | dog.go | 5 min |
| Add repository method | dog_repository.go | 15 min |
| Add handler and route | dog_handler.go, main.go | 20 min |
| Add featured dogs section to index | index.html | 30 min |
| Add admin toggle control | admin-dogs.html | 30 min |
| Update API client | api.js | 10 min |
| Test complete feature | - | 20 min |

---

## Additional UI/UX Issues Found During Analysis

### Issue A: i18n Translation Not Applied Immediately
Some pages call `window.i18n.updateElement(document.body)` which may cause a flash of untranslated content.

**Recommendation:** Add CSS to hide content until translations load:
```css
[data-i18n]:empty { visibility: hidden; }
```

### Issue B: Alert Styles Redefined in Multiple Places
The `.alert-warning` class is defined differently in main.css (lines 331-337 vs lines 932-939).

**Recommendation:** Remove duplicate definition, keep consistent styling.

### Issue C: Form Actions Styling Not Global
`.form-actions` is only defined inline in dogs.html modal.

**Recommendation:** Move to main.css for reuse in other modals.

### Issue D: Dark Theme CSS Variables Still Referenced
Multiple references to `var(--dark-gray)` and `var(--light-gray)` which are undefined:
- dogs.html: `var(--dark-gray)`
- Other files may have similar issues

**Recommendation:** Search and replace all undefined CSS variables.

---

## Testing Checklist

### Desktop Testing (1920x1080, 1366x768, 1024x768)
- [ ] Admin navigation doesn't overflow
- [ ] Dropdowns open and close properly
- [ ] All buttons have consistent styling
- [ ] Modal is clearly visible with proper background
- [ ] Calendar grid view works correctly

### Mobile Testing (375x667, 414x896)
- [ ] Hamburger menu works
- [ ] Calendar defaults to list view
- [ ] Grid view toggle hidden or handled gracefully
- [ ] Featured dogs display correctly on index

### Cross-Browser Testing
- [ ] Chrome
- [ ] Firefox
- [ ] Safari
- [ ] Edge

---

## CSS Variable Reference

### Currently Defined (main.css lines 4-34)
```css
:root {
    --primary-green: #82b965;
    --secondary-green: #6fa050;
    --accent-orange: #ff8c42;
    --accent-blue: #4a90e2;
    --warm-cream: #fef9f3;
    --header-bg: #4a7c59;
    --text-white: #ffffff;
    --text-dark: #2c3e34;
    --text-gray: #5a6c57;
    --error-red: #e74c3c;
    --warning-orange: #f39c12;
    --info-blue: #3498db;
    --light-bg: #f8f9fa;
    --card-bg: #ffffff;
    --border-light: #e1e8e5;
    --border-radius: 6px;
    --font-family: Arial, sans-serif;
    --spacing-xs: 0.44rem;
    --spacing-sm: 0.88rem;
    --spacing-md: 1.32rem;
    --spacing-lg: 1.76rem;
    --spacing-xl: 2.64rem;
}
```

### Missing/Undefined (DO NOT USE)
- `--dark-gray` - NOT DEFINED
- `--light-gray` - NOT DEFINED (spinner uses it at line 424)

---

## Summary

| Sprint | Priority | Est. Time | Description |
|--------|----------|-----------|-------------|
| 1 | High | 50 min | Fix modal and mobile calendar |
| 2 | Medium | 55 min | Standardize button styles |
| 3 | High | 2.5 hours | Fix admin navigation |
| 4 | Feature | 2.5 hours | Add featured dogs |

**Total Estimated Time: ~6 hours**

---

## Approval

- [ ] Plan reviewed by stakeholder
- [ ] Priority order confirmed
- [ ] Ready to begin implementation
