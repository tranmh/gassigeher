# Demo Tenant Implementation Plan

## Overview

Create a demo tenant that auto-creates on startup at `demo.gassigeher.org` with sample data, public credentials, and daily midnight reset. The demo serves as a "try before you buy" experience for potential customers.

## Requirements Summary

| Requirement | Decision |
|-------------|----------|
| Multi-tenancy | Subdomain + shared DB (already implemented) |
| Demo data | 5 dogs, 3 users (green/orange/blue), sample bookings |
| Credentials | Random password, displayed on landing/demo.html |
| Reset | Daily at midnight Europe/Berlin, hard reset |
| Emails | Disabled for demo tenant |
| Uploads | Allowed, cleaned up at reset |
| Banner | Persistent orange banner in app |

---

## Implementation Steps

### Phase 1: Database Schema

**1.1 Create migration `internal/database/022_demo_tenant_state.go`**
- New table `demo_tenant_state` with columns:
  - `tenant_id` (FK to tenants)
  - `admin_password` (plain text - intentionally for demo display)
  - `last_reset_at`, `next_reset_at` timestamps

**1.2 Create migration `internal/database/023_add_tenant_is_demo.go`**
- Add `is_demo` boolean column to `tenants` table

---

### Phase 2: Models

**2.1 Create `internal/models/demo_tenant_state.go`**
```go
type DemoTenantState struct {
    ID, TenantID      int
    AdminPassword     string
    LastResetAt       *time.Time
    NextResetAt       *time.Time
}

type DemoCredentials struct {
    AdminEmail, AdminPassword string
    NextResetAt, LastResetAt  string  // Formatted for display
}
```

**2.2 Modify `internal/models/tenant.go`**
- Add `IsDemo bool` field to Tenant struct

---

### Phase 3: Repository Layer

**3.1 Create `internal/repository/demo_tenant_repository.go`**
- `GetState(tenantID)` - Get demo state
- `CreateState(state)` - Create initial state
- `UpdateState(tenantID, password, lastReset, nextReset)` - Update after reset
- `GetCredentials(tenantID)` - Get formatted credentials for display

**3.2 Modify `internal/repository/tenant_repository.go`**
- Add `GetBySlug(slug)` if not exists
- Add `CreateDemoTenant()` - Creates tenant with `is_demo=true`
- Add `IsDemoTenant(tenantID)` - Check if tenant is demo

---

### Phase 4: Demo Seed Service

**4.1 Create `internal/services/demo_seed_service.go`**

Constants:
- `DemoTenantSlug = "demo"`
- `DemoAdminEmail = "admin@demo.gassigeher.org"`

Methods:
- `EnsureDemoTenant()` - Create demo tenant if not exists, seed data
- `SeedDemoData(tenantID)` - Populate demo data
- `ResetDemoTenant()` - Delete all data, regenerate password, re-seed
- `seedDemoUsers(tenantID, password)` - Create 4 users:
  - Demo Admin (super admin, random password)
  - Anna Gruen (green level, password: demo1234)
  - Bernd Orange (orange level, password: demo1234)
  - Clara Blau (blue level, password: demo1234)
- `seedDemoDogs(tenantID)` - Create 5 dogs:
  - Bella (Labrador, green, featured)
  - Max (Golden Retriever, green, featured)
  - Luna (Border Collie, orange)
  - Rocky (Schaeferhund, orange)
  - Duke (Rottweiler, blue)
- `seedDemoBookings(tenantID)` - Create sample bookings
- `generateRandomPassword()` - 12-char hex using crypto/rand
- `calculateNextResetTime()` - Next midnight Europe/Berlin

---

### Phase 5: Seed Integration

**5.1 Modify `internal/database/seed.go`**
- Add `seedDemoTenant(db)` function
- Call from `SeedDatabase()` after Central Admin creation
- Non-fatal: log warning if fails, don't block startup

---

### Phase 6: Cron Service

**6.1 Modify `internal/cron/cron.go`**

Add to CronService:
- `demoSeedService *services.DemoSeedService`

Add methods:
- `runDemoReset()` - Scheduled at midnight Europe/Berlin
  - Calculate sleep until next midnight
  - Call `demoSeedService.ResetDemoTenant()`
  - Call `cleanDemoUploads()`
  - Retry up to 3 times on failure
- `cleanDemoUploads()` - Remove `uploads/tenants/{demoTenantID}/`

Start in `Start()`:
```go
go c.runDemoReset()
```

---

### Phase 7: Email Service

**7.1 Modify `internal/services/email_service.go`**

Add to EmailService:
- `tenantRepo *repository.TenantRepository`

Modify `SendEmail()`:
```go
if tenantID > 0 {
    if isDemo, _ := s.tenantRepo.IsDemoTenant(tenantID); isDemo {
        log.Printf("[Email] Skipping for demo tenant: %s", subject)
        return nil
    }
}
```

**7.2 Update handlers to pass tenantID to email methods**

---

### Phase 8: Demo API Handler

**8.1 Create `internal/handlers/demo_handler.go`**

Endpoints:
- `GET /api/demo/credentials` - Returns admin email, password, reset times
- `GET /api/demo/status` - Returns `{is_demo: bool, next_reset: string}`

**8.2 Register routes in `cmd/server/main.go`**
```go
router.HandleFunc("/api/demo/credentials", demoHandler.GetDemoCredentials).Methods("GET")
router.HandleFunc("/api/demo/status", demoHandler.GetDemoStatus).Methods("GET")
```

---

### Phase 9: Landing Page

**9.1 Create `internal/static/landing/demo.html`**

Content:
- Hero section with Gassigeher Demo title
- Credentials card with admin email/password (loaded from API)
- Copy buttons for credentials
- Warning box: "Daten werden taeglich um Mitternacht zurueckgesetzt"
- Reset time display
- Demo users table (Anna/Bernd/Clara with demo1234 password)
- Features overview
- "Zur Demo-Anwendung" button linking to demo.gassigeher.org/login.html

---

### Phase 10: Frontend Demo Banner

**10.1 Create `internal/static/frontend/js/demo-banner.js`**

Features:
- Auto-detect demo tenant by subdomain (`hostname.startsWith('demo.')`)
- Inject fixed orange banner at top:
  - "DEMO-MODUS - Daten werden taeglich um Mitternacht zurueckgesetzt"
  - Show next reset time from `/api/demo/status`
- Add `padding-top` to body for banner height (48px)

**10.2 Include in frontend pages**

Add to `<head>` of all frontend HTML files:
```html
<script src="/js/demo-banner.js"></script>
```

---

### Phase 11: Middleware Enhancement

**11.1 Modify `internal/middleware/tenant.go`**

Add context key:
```go
const IsDemoKey contextKey = "isDemo"
```

In TenantMiddleware, add to context:
```go
ctx = context.WithValue(ctx, IsDemoKey, tenant.IsDemo)
```

Add helper:
```go
func IsDemoTenant(ctx context.Context) bool
```

---

### Phase 12: Landing Page Integration (Try Before You Buy)

**Goal:** Position demo prominently across landing pages to convince hesitant prospects to try before committing.

#### 12.1 Navigation Update

**Modify all landing pages** - Add "Demo testen" to header navigation:

Current: `[Startseite] [Preise] [FAQ] [Jetzt starten]`

New: `[Startseite] [Preise] [FAQ] [Demo testen] [Jetzt starten]`

**Files to modify:**
- `internal/static/landing/index.html`
- `internal/static/landing/pricing.html`
- `internal/static/landing/faq.html`
- `internal/static/landing/register.html`
- `internal/static/landing/about.html`
- `internal/static/landing/contact.html`
- All legal pages (imprint, privacy, agb, sla, widerrufsbelehrung)

#### 12.2 Hero Section Enhancement (index.html)

Add demo CTA below main buttons:

```html
<div class="hero-buttons">
    <a href="/landing/register.html" class="btn btn-primary">Kostenlos registrieren</a>
    <a href="#features" class="btn btn-secondary">Mehr erfahren</a>
</div>
<!-- NEW: Demo teaser -->
<div class="demo-teaser">
    <span class="demo-teaser-text">Noch unsicher?</span>
    <a href="/landing/demo.html" class="demo-link">
        Erst unverbindlich testen
    </a>
</div>
```

#### 12.3 Pricing Page Demo Banner (pricing.html)

Add prominent banner above pricing cards:

```html
<section class="demo-banner">
    <div class="container">
        <div class="demo-banner-content">
            <div class="demo-banner-icon">Demo</div>
            <div class="demo-banner-text">
                <h3>Noch nicht sicher, welcher Plan passt?</h3>
                <p>Testen Sie Gassigeher unverbindlich in unserer Live-Demo.
                   Alle Funktionen, keine Registrierung, keine Verpflichtung.</p>
            </div>
            <a href="/landing/demo.html" class="btn btn-primary btn-large">
                Demo starten
            </a>
        </div>
    </div>
</section>
```

#### 12.4 How It Works Enhancement (index.html)

Add demo mention after 3-step process:

```html
<div class="how-it-works-demo">
    <p>Moechten Sie diese Schritte erst einmal ausprobieren?</p>
    <a href="/landing/demo.html" class="btn btn-secondary">
        In der Demo durchspielen
    </a>
</div>
```

#### 12.5 FAQ Page Addition (faq.html)

Add new FAQ entry about demo:

```html
<div class="faq-item">
    <button class="faq-question">
        Kann ich Gassigeher vor der Registrierung testen?
    </button>
    <div class="faq-answer">
        <p>Ja! Wir bieten eine vollstaendige <a href="/landing/demo.html">Live-Demo</a>
           an. Sie koennen alle Funktionen ausprobieren - als Administrator,
           als Gassigeher mit verschiedenen Erfahrungsstufen. Die Demo wird
           taeglich zurueckgesetzt, so dass Sie bedenkenlos alles testen koennen.</p>
    </div>
</div>
```

#### 12.6 Demo Landing Page Design (landing/demo.html)

**Layout:**

1. **Hero Section** - "Gassigeher Live erleben"
   - Headline: "Testen Sie Gassigeher - ohne Registrierung, ohne Verpflichtung"
   - Subheadline: "Erleben Sie alle Funktionen unserer Tierheim-Software in einer vollstaendigen Demo-Umgebung"

2. **Credentials Card** (prominently displayed)
   - Admin credentials (loaded from API)
   - Demo user credentials (fixed: demo1234)
   - Copy buttons
   - Warning: "Daten werden taeglich um Mitternacht zurueckgesetzt"

3. **What You Can Test Section**
   - Admin features: Hundeverwaltung, Buchungsuebersicht, Benutzer verwalten
   - User features: Hunde durchstoebern, Buchungen erstellen, Profil verwalten
   - Different experience levels: Gruen, Orange, Blau

4. **Demo Users Overview Table**
   - Anna Gruen (Anfaenger) - password: demo1234
   - Bernd Orange (Fortgeschritten) - password: demo1234
   - Clara Blau (Experte) - password: demo1234

5. **Demo Dogs Preview**
   - Show 5 sample dogs with categories

6. **CTA Section**
   - Primary: "Zur Demo-Anwendung" -> demo.gassigeher.org/login.html
   - Secondary: "Bereit fuer Ihr eigenes Tierheim?" -> /landing/register.html

7. **Trust Indicators**
   - "Keine Kreditkarte erforderlich"
   - "Keine Installation"
   - "Sofort loslegen"

#### 12.7 CSS Additions (landing.css)

```css
/* Demo teaser in hero */
.demo-teaser {
    margin-top: 1.5rem;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    font-size: 0.95rem;
}

.demo-teaser-text {
    color: var(--color-text-light);
}

.demo-link {
    color: var(--color-primary);
    font-weight: 600;
    text-decoration: none;
    transition: color 0.2s;
}

.demo-link:hover {
    color: var(--color-primary-dark);
    text-decoration: underline;
}

/* Demo banner on pricing page */
.demo-banner {
    background: linear-gradient(135deg, #f0f4e8 0%, #e8f0dc 100%);
    padding: 2rem;
    border-radius: 12px;
    margin-bottom: 3rem;
}

.demo-banner-content {
    display: flex;
    align-items: center;
    gap: 2rem;
}

.demo-banner-icon {
    font-size: 3rem;
}

.demo-banner-text h3 {
    margin-bottom: 0.5rem;
    color: var(--color-secondary);
}

.demo-banner-text p {
    color: var(--color-text-light);
    margin: 0;
}

/* Nav highlight for demo link */
.nav-demo {
    background: var(--color-accent);
    padding: 0.5rem 1rem;
    border-radius: 6px;
    color: var(--color-primary) !important;
    font-weight: 600;
}
```

---

## Files Summary

### New Files (9 total)

| File | Purpose |
|------|---------|
| `internal/database/022_demo_tenant_state.go` | Migration for demo_tenant_state table |
| `internal/database/023_add_tenant_is_demo.go` | Migration for is_demo column |
| `internal/models/demo_tenant_state.go` | DemoTenantState and DemoCredentials models |
| `internal/repository/demo_tenant_repository.go` | Demo state persistence |
| `internal/services/demo_seed_service.go` | Core demo logic |
| `internal/handlers/demo_handler.go` | API endpoints for credentials |
| `internal/static/landing/demo.html` | Demo landing page (try before buy) |
| `internal/static/frontend/js/demo-banner.js` | Frontend banner component |

### Modified Files (17 total)

| File | Changes |
|------|---------|
| `internal/models/tenant.go` | Add IsDemo field |
| `internal/repository/tenant_repository.go` | Add demo tenant methods |
| `internal/database/seed.go` | Call seedDemoTenant() |
| `internal/cron/cron.go` | Add runDemoReset() job |
| `internal/services/email_service.go` | Skip emails for demo |
| `cmd/server/main.go` | Register demo routes |
| `internal/middleware/tenant.go` | Add IsDemoKey to context |
| `internal/static/landing/index.html` | Add demo teaser in hero + how-it-works |
| `internal/static/landing/pricing.html` | Add demo banner above pricing |
| `internal/static/landing/faq.html` | Add demo FAQ entry |
| `internal/static/landing/assets/css/landing.css` | Add demo-related styles |
| `internal/static/landing/register.html` | Add demo nav link |
| `internal/static/landing/about.html` | Add demo nav link |
| `internal/static/landing/contact.html` | Add demo nav link |
| `internal/static/landing/imprint.html` | Add demo nav link |
| `internal/static/landing/privacy.html` | Add demo nav link |
| + other legal pages | Add demo nav link |

---

## Testing Checklist

### Backend & Demo Tenant
- [ ] Demo tenant created on fresh startup
- [ ] Credentials displayed on landing/demo.html
- [ ] Admin can login with displayed password
- [ ] Demo users can login with demo1234
- [ ] 5 dogs visible with correct categories
- [ ] Sample bookings created
- [ ] Demo banner visible on all app pages (demo.gassigeher.org)
- [ ] Emails NOT sent for demo tenant
- [ ] Reset at midnight recreates all data
- [ ] New password generated after reset
- [ ] Uploads cleaned after reset

### Landing Page Integration
- [ ] "Demo testen" link visible in all landing page navigation
- [ ] Hero section shows "Noch unsicher? Erst unverbindlich testen"
- [ ] Pricing page shows demo banner above pricing cards
- [ ] How It Works section has "In der Demo durchspielen" button
- [ ] FAQ has entry about demo availability
- [ ] Demo landing page loads credentials from API
- [ ] Copy buttons work for credentials
- [ ] "Zur Demo-Anwendung" button links to demo.gassigeher.org
- [ ] "Bereit fuer Ihr eigenes Tierheim?" links to register page

---

## DNS Requirements

**Production:**
```
demo.gassigeher.org -> same server as main app
```

**Development:**
```
# /etc/hosts
127.0.0.1 demo.gassigeher.local
```
