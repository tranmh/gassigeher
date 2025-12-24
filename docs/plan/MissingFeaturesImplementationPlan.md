# Missing Features Implementation Plan

> **Document Version:** 1.0
> **Created:** 2025-12-24
> **Status:** Approved for Implementation

This document outlines the implementation plan for missing features required before the Gassigeher SaaS platform can go live.

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [User Requirements](#user-requirements)
3. [Current State Analysis](#current-state-analysis)
4. [Implementation Phases](#implementation-phases)
   - [Phase 1: P0 Critical Blockers](#phase-1-p0-critical-blockers)
   - [Phase 2: P1 High Priority Features](#phase-2-p1-high-priority-features)
   - [Phase 3: P2 Growth Features](#phase-3-p2-growth-features-post-launch)
5. [Critical Files Summary](#critical-files-summary)
6. [Execution Timeline](#execution-timeline)
7. [TEO Import Research](#teo-import-research)

---

## Executive Summary

The Gassigeher SaaS platform requires several features before public launch:

| Priority | Features | Status |
|----------|----------|--------|
| **P0 - Blockers** | Configurable Frontend, Upload Isolation, Self-Service Audit, Landing Backlink, Subscription Verification | Not started |
| **P1 - High** | Guided Tour, Unused Tenant Detection, Central Admin Impersonation, GDPR Export, Monitoring | Not started |
| **P2 - Growth** | Marketing Features, TEO Import, Translations | Post-launch |

**Key Finding:** Database schema is 100% ready for most features - implementation is primarily API exposure and frontend work.

---

## User Requirements

| Decision | User Choice |
|----------|-------------|
| Timeline | No fixed date - quality over speed |
| Frontend customization | Text + sections (welcome, about, features configurable) |
| Billing state | Stripe integrated, needs expiry enforcement verification |
| Legal | Templates exist in `internal/static/landing/*` |
| Guided tour | Library-based (Shepherd.js or Intro.js) |
| TEO import | Critical - most shelters use TEO |
| Unused tenant detection | Manual review flag in central admin dashboard |
| Impersonation | Exists for Super Admin, need for Central Admin too |
| Marketing features | All features, toggleable via central admin UI |
| Data export | JSON dump format |
| Monitoring | Both Sentry + Prometheus/Grafana |
| Payment failure | Verify existing implementation (should already work) |

---

## Current State Analysis

### 1. Configurable Frontend (#1)

**Current State:** Database schema 100% ready, but not exposed to frontend

**What exists:**
- `TenantSettings` model with: `welcome_message`, `footer_text`, `website_url`, `donation_url`, `logo_url`, `favicon_url`
- Theme system fully working (10 presets + custom colors)
- Repository methods: `GetSettings()`, `UpdateSettings()`

**What's hardcoded in `index.html`:**
| Location | Line | Content |
|----------|------|---------|
| Hero h1 | 44 | "Willkommen im Gassigeher-Team des Tierheim Göppingen" |
| Subtitle | 45-46 | Static subtitle and description |
| Footer | 92 | "Gassigeher. Alle Rechte vorbehalten." |
| Logo | 13 | "Gassigeher" text |

**What needs to be built:**
1. API endpoint `GET /api/tenant/branding` - returns tenant name + settings
2. API endpoint `PUT /api/tenant/branding` - admin updates branding
3. Frontend JS to load and inject tenant-specific text
4. Admin UI page `admin-branding.html` for editing

### 2. Subscription Expiry Handling

**Current State:** PARTIALLY implemented

**What exists:**
- `Subscription.IsExpired()` method in model
- Stripe webhooks update status to `past_due` on payment failure
- Billing handler processes all webhook events

**What's MISSING (needs verification):**
- No middleware enforcing subscription status on API calls
- No blocking of "add dog" when expired

**Action:** Verify if enforcement exists in code, add if missing

### 3. Impersonation

**Current State:** FULLY implemented for Super Admin

**What exists (`internal/handlers/user_handler.go` lines 769-928):**
- `POST /api/super-admin/users/:id/impersonate`
- `POST /api/super-admin/end-impersonation`
- JWT with `impersonating`, `original_user_id`, `tenant_id` claims
- Audit logging
- Frontend UI with impersonation banner

**What needs to be built:**
- Central Admin impersonation endpoint (similar to Super Admin)
- Central Admin UI for cross-tenant impersonation

### 4. Upload Tenant Isolation

**Current State:** Implemented via path prefix

**What exists:**
- S3 service uses `tenants/{slug}/` path prefix
- Path validation prevents traversal attacks

**What needs verification:**
- Automated tests confirming cross-tenant access is blocked
- File serving endpoint validates tenant context

---

## Implementation Phases

### Phase 1: P0 Critical Blockers

#### 1.1 Configurable Frontend (3-4 days)

**Backend Changes:**

```
internal/handlers/tenant_handler.go
  - Add GetBranding() → GET /api/tenant/branding
  - Add UpdateBranding() → PUT /api/tenant/branding (admin only)

cmd/server/main.go
  - Register new routes
```

**Frontend Changes:**

```
internal/static/frontend/index.html
  - Add JS to fetch /api/tenant/branding on load
  - Replace hardcoded text with dynamic content
  - Fallback to i18n if no custom text

internal/static/frontend/admin-branding.html (NEW)
  - Form fields: welcome_message, tagline, description, footer_text
  - Live preview of hero section
  - Theme/color picker integration
  - Logo upload (reuse existing)

internal/static/frontend/js/api.js
  - Add getBranding(), updateBranding() methods
```

**Database:** No changes needed - fields already exist in `tenant_settings` table

---

#### 1.2 Upload Tenant Isolation Test (0.5 days)

**Verification Tasks:**
- [x] S3 service uses `tenants/{slug}/` path prefix (confirmed in code)
- [ ] Write automated test to verify cross-tenant file access is blocked
- [ ] Test file serving endpoint validates tenant context

**Files to create/modify:**
```
internal/services/s3_service_test.go - Add isolation tests
internal/handlers/file_handler_test.go - Add cross-tenant tests
```

---

#### 1.3 Self-Service Completeness Audit (1 day)

**Verification Checklist:**
- [ ] Tenant admin can reset own password
- [ ] Tenant admin can add/edit/delete dogs
- [ ] Tenant admin can manage users (promote to admin, deactivate)
- [ ] Tenant admin can configure booking times
- [ ] Tenant admin can manage blocked dates
- [ ] Tenant admin can view/manage bookings
- [ ] Tenant admin can configure system settings
- [ ] Tenant admin can upload logo

**Deliverable:** Create checklist document and test each flow manually

---

#### 1.4 Landing Page Backlink (0.5 days)

**Requirement:** Organic growth via existing users - all tenant pages link back to gassigeher.org

**Implementation:**
- Add "Powered by Gassigeher" footer link on ALL tenant pages
- Links to `https://gassigeher.org`
- Non-removable (both free AND pro tiers)
- Subtle but visible - standard SaaS pattern

**Files to modify:**
```
internal/static/frontend/*.html (all pages)
  - Add footer: <a href="https://gassigeher.org" target="_blank">Powered by Gassigeher</a>
```

**Design:** Similar to Shopify/Wix free tier branding - visible but not intrusive

---

#### 1.5 Subscription Expiry Verification (0.5 days)

**Verification Tasks:**
- [ ] Check if `is_expired` flag exists and is enforced
- [ ] Check if "add dog" is blocked when expired
- [ ] If missing, add middleware check

**Files to check:**
```
internal/middleware/tenant.go - Should check subscription status
internal/handlers/dog_handler.go - Should block CreateDog when expired
```

---

### Phase 2: P1 High Priority Features

#### 2.1 Guided Tour (#4) (2-3 days)

**Library Choice:** Shepherd.js (MIT license, 12KB gzipped, excellent UX)

**Implementation:**

```
internal/static/frontend/js/shepherd-tour.js (NEW)
  - User tour: Dashboard → Dogs → Calendar → Booking flow
  - Admin tour: Dashboard → Dogs management → Users → Settings

internal/static/frontend/js/tour-steps.js (NEW)
  - Define tour steps with i18n support
  - Track completion in localStorage
  - Always-on for demo.gassigeher.org

Pages to modify:
  - dashboard.html, dogs.html, calendar.html (user tour)
  - admin-dashboard.html, admin-dogs.html, admin-users.html (admin tour)
```

**Features:**
- Skip after first completion (localStorage flag)
- "Replay tour" button in profile/settings
- Always active on demo tenant

---

#### 2.2 Unused Tenant Detection (#3) (1 day)

**Implementation:**

```
internal/cron/tenant_activity.go (NEW)
  - Daily job checking last_booking_date per tenant
  - Flag tenants with no bookings in X days (configurable)
  - Store flag in tenants table or new tenant_activity table

internal/handlers/central_admin_handler.go
  - Add GetInactiveTenants() endpoint
  - Returns list with days_inactive, last_activity_date

internal/static/central/tenants.html
  - Add "Inactive" filter/tab
  - Show inactivity warning badge
  - Manual review actions (email tenant, suspend, delete)
```

**Behavior:** No auto-action - just flagging for manual review by central admin

---

#### 2.3 Central Admin Impersonation (1 day)

**Implementation:**

```
internal/handlers/central_admin_handler.go
  - Add ImpersonateTenantUser() → POST /api/central/impersonate/:userId
  - Add EndCentralImpersonation() → POST /api/central/end-impersonation
  - Audit log all impersonation events

internal/static/central/users.html
  - Add "Impersonate" button per user
  - Show impersonation banner when active
```

**Architecture:** Reuse existing impersonation JWT logic from Super Admin implementation

---

#### 2.4 Tenant Data Export - GDPR (1 day)

**Implementation:**

```
internal/handlers/tenant_handler.go
  - Add ExportTenantData() → GET /api/tenant/export
  - Collects: users, dogs, bookings, settings
  - Returns JSON dump

internal/services/export_service.go (NEW)
  - Builds complete tenant data package
  - Includes all related records
  - Sanitizes sensitive data (password hashes)

internal/static/frontend/admin-settings.html
  - Add "Export all data" button
  - Download as .json file
```

---

#### 2.5 Monitoring Setup (1-2 days)

**Sentry Integration (Error Tracking):**

```
internal/services/sentry_service.go (NEW)
  - Initialize Sentry SDK
  - Capture panics, errors
  - Add tenant context to errors

cmd/server/main.go
  - Initialize Sentry on startup
  - Add recovery middleware
```

**Prometheus/Grafana (Metrics):**

```
internal/metrics/metrics.go (NEW)
  - HTTP request duration histogram
  - Active users gauge
  - Bookings counter
  - Database connection pool stats

cmd/server/main.go
  - Expose /metrics endpoint
```

**Environment Variables:**
- `SENTRY_DSN` - Sentry project DSN
- `PROMETHEUS_ENABLED` - Enable metrics endpoint

---

### Phase 3: P2 Growth Features (Post-Launch)

#### 3.1 Marketing Features (#2.x) (3-4 days)

**Database Schema:**

```sql
-- New table for marketing campaigns
CREATE TABLE marketing_campaigns (
  id SERIAL PRIMARY KEY,
  type VARCHAR(50),           -- 'fomo_countdown', 'referral', 'reference_page'
  name VARCHAR(255),
  config JSONB,               -- Campaign-specific settings
  is_active BOOLEAN DEFAULT false,
  start_date TIMESTAMP,
  end_date TIMESTAMP,
  created_at TIMESTAMP
);

-- Referral tracking
CREATE TABLE referral_codes (
  id SERIAL PRIMARY KEY,
  code VARCHAR(50) UNIQUE,
  referrer_tenant_id INT,
  discount_months_referrer INT DEFAULT 3,
  discount_months_referee INT DEFAULT 1,
  uses_count INT DEFAULT 0,
  max_uses INT,
  is_active BOOLEAN DEFAULT true
);

CREATE TABLE referral_uses (
  id SERIAL PRIMARY KEY,
  code_id INT REFERENCES referral_codes(id),
  referee_tenant_id INT,
  applied_at TIMESTAMP
);
```

**Features:**
1. **FOMO Countdown** (#2): "First 10 get Pro free" with countdown timer
2. **Referral Codes** (#2.3): Generate codes, track usage, apply discounts
3. **Reference Page** (#2.2, #2.4): Public list of participating shelters
4. **Marketing Dashboard**: Central admin UI to enable/disable campaigns

**Central Admin UI:**
```
internal/static/central/marketing.html (NEW)
  - Toggle campaigns on/off
  - Configure FOMO countdown (limit, end date)
  - Generate referral codes
  - View referral statistics
```

---

#### 3.2 TEO Import (#8) (2-3 days)

**Backend:**

```
internal/handlers/import_handler.go (NEW)
  - POST /api/admin/import/dogs/preview - Dry run
  - POST /api/admin/import/dogs - Execute import

internal/services/import_service.go (NEW)
  - Parse Excel/CSV files
  - Validate records
  - Field mapping logic
  - Batch insert with transaction

internal/services/excel_parser.go (NEW)
  - Parse .xlsx files using excelize library
  - Extract column headers
  - Read data rows
```

**Frontend:**

```
internal/static/frontend/admin-import.html (NEW)
  - File upload (drag & drop)
  - Column mapping UI (dropdown per field)
  - Preview table (first 10 rows)
  - Import progress bar
  - Results summary

internal/static/frontend/js/import-ui.js (NEW)
  - Handle file upload
  - Manage mapping state
  - Display preview/results
```

**Field Mapping TEO → Gassigeher:**

| TEO Field | Gassigeher Field | Notes |
|-----------|------------------|-------|
| Name | dog.name | Direct mapping |
| Rasse | dog.breed | Direct mapping |
| Größe | dog.size | Map to small/medium/large |
| Alter | dog.age | Direct mapping |
| Farbe | dog.color_id | Map to ColorCategory |
| Status | dog.is_available | Boolean conversion |
| Spezialhinweise | dog.special_instructions | Direct mapping |
| Abholort | dog.pickup_location | Direct mapping |

---

#### 3.3 Translations (#7) (Ongoing)

**Approach:** Add language files incrementally

```
internal/static/frontend/i18n/
  - de.json (existing - German)
  - en.json (NEW - English)
  - fr.json (NEW - French)
  - es.json (NEW - Spanish)
  - it.json (NEW - Italian)
  - nl.json (NEW - Dutch)
  - da.json (NEW - Danish)
  - fi.json (NEW - Finnish)
```

**Implementation:**
- Language selector in footer/header
- Store preference in localStorage
- Fallback to German if translation missing

---

## Critical Files Summary

### Must Modify

| File | Changes |
|------|---------|
| `internal/handlers/tenant_handler.go` | Add branding endpoints |
| `internal/static/frontend/index.html` | Dynamic content loading |
| `internal/static/frontend/js/api.js` | New API methods |
| `cmd/server/main.go` | New routes |
| `internal/handlers/central_admin_handler.go` | Impersonation, inactive tenants |
| `internal/static/frontend/*.html` | Landing page backlink |

### Must Create

| File | Purpose |
|------|---------|
| `internal/static/frontend/admin-branding.html` | Branding admin UI |
| `internal/static/frontend/js/shepherd-tour.js` | Guided tour |
| `internal/cron/tenant_activity.go` | Inactive tenant detection |
| `internal/handlers/import_handler.go` | TEO import |
| `internal/services/import_service.go` | Import logic |
| `internal/services/export_service.go` | GDPR export |
| `internal/services/sentry_service.go` | Error tracking |
| `internal/metrics/metrics.go` | Prometheus metrics |
| `internal/static/central/marketing.html` | Marketing dashboard |

---

## Execution Timeline

### Week 1: P0 Blockers
- [x] 1.1 Configurable Frontend (most impactful)
- [ ] 1.2 Upload Isolation Test
- [ ] 1.3 Self-Service Audit
- [ ] 1.4 Landing Page Backlink (organic growth)
- [ ] 1.5 Subscription Expiry Verification

### Week 2: P1 Features
- [ ] 2.1 Guided Tour
- [ ] 2.2 Unused Tenant Detection
- [ ] 2.3 Central Admin Impersonation
- [ ] 2.4 GDPR Export

### Week 3: Operations
- [ ] 2.5 Monitoring (Sentry + Prometheus)
- [ ] Testing and bug fixes

### Post-Launch: P2 Features
- [ ] 3.1 Marketing Features
- [ ] 3.2 TEO Import
- [ ] 3.3 Translations (ongoing)

---

## TEO Import Research

### What is TEO?

**TEO (Tierschutz Erfolgreich Organisieren)** - "Animal Protection Successfully Organized"

- Official animal shelter management software of the **Deutscher Tierschutzbund e.V.** (German Animal Welfare Federation)
- Built on the **SuccessControl CRM** platform (proprietary)
- Used by **500+ German shelters** and animal welfare organizations
- Comprehensive system for shelter operations, volunteer coordination, and animal records

### TEO Data Export Capabilities

| Format | Availability | Notes |
|--------|--------------|-------|
| Microsoft Access (.mdb/.accdb) | Native format | Database export |
| Microsoft Excel (.xlsx) | Supported | Structured tables |
| Public API | **Not available** | Proprietary system |

### Recommended Import Approach

1. **Excel file upload** with field mapping UI
2. **Preview/dry-run** before actual import
3. **Batch import** with transaction support
4. **Error handling** with skip invalid + log errors

### Sources

- [TEO Homepage](http://www.teo.successcontrol.de/)
- [TEO Handbuch](http://teo.successcontrol.de/TEO-Handbuch-V109.pdf)
- [Deutscher Tierschutzbund TEO Page](https://www.tierschutzbund.de/teo/)

---

## Open Questions Resolved

| Question | Resolution |
|----------|------------|
| Subscription expiry | Verify existing - user believes it's implemented |
| Impersonation scope | Central Admin needs it, Super Admin already has it |
| TEO format | Excel upload with mapping UI (no API available) |
| Marketing toggle | All features, central admin toggles via UI |
| Monitoring choice | Both Sentry (errors) + Prometheus (metrics) |
