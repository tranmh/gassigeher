# Missing Features for SaaS Implementation Plan

**Document Created:** 2025-12-22
**Target Domain:** gassigeher.org
**Current Status:** ~90% Complete
**Estimated Completion:** 3-4 days

---

## Executive Summary

The Gassigeher SaaS implementation is nearly complete with all major components implemented:
- Multi-tenant database schema (14 tables with `tenant_id`)
- JWT authentication with tenant claims
- All repositories and handlers multi-tenant aware
- Stripe billing integration
- Landing pages (10 pages)
- Docker + Caddy configuration with wildcard SSL
- Central Admin API (10+ endpoints)
- Migration tool for single→multi tenant

However, several **critical gaps** prevent production deployment. This document outlines what's missing and provides an implementation plan.

---

## Phase 1: Critical Blocking Issues

### 1.1 TenantMiddleware Not Registered

**Priority:** CRITICAL
**Effort:** 30 minutes
**File:** `cmd/server/main.go`

**Issue:** The `TenantMiddleware` exists in `internal/middleware/tenant.go` but is NOT added to the router middleware chain. Without this, subdomain-based tenant resolution doesn't work.

**Current State:**
```go
// middleware/tenant.go exists with:
func TenantMiddleware(tenantRepo *repository.TenantRepository, baseDomain string) func(http.Handler) http.Handler
```

**Required Change in main.go:**
```go
// After creating tenantRepo, before protected routes:
tenantRepo := repository.NewTenantRepository(db)

// Add tenant middleware to resolve subdomain to tenant
router.Use(middleware.TenantMiddleware(tenantRepo, cfg.BaseDomain))
```

**Testing:**
- Request to `tenant1.gassigeher.org` should set `TenantIDKey` in context
- Request to `gassigeher.org` (no subdomain) should pass through for landing pages

---

### 1.2 TenantRepository Missing Methods

**Priority:** CRITICAL
**Effort:** 2-3 hours
**File:** `internal/repository/tenant_repository.go`

**Missing Methods:**

| Method | Used By | Purpose |
|--------|---------|---------|
| `FindBySlug(slug string)` | TenantMiddleware | Resolve subdomain to tenant |
| `CreateTx(tx *sql.Tx, tenant *models.Tenant)` | TenantHandler | Transaction-safe tenant creation |
| `GetSettings(tenantID int)` | SettingsHandler | Fetch tenant_settings |
| `CreateSettingsTx(tx *sql.Tx, settings)` | Provisioning | Create default settings |
| `UpdateStatus(tenantID int, status string)` | CentralAdmin | Activate/suspend tenant |
| `GetStats(tenantID int)` | CentralAdmin | Tenant statistics |
| `FindAllWithStats()` | CentralAdmin | List tenants with counts |

**Implementation Template:**
```go
// FindBySlug returns a tenant by its subdomain slug
func (r *TenantRepository) FindBySlug(slug string) (*models.Tenant, error) {
    query := `SELECT id, slug, name, status, contact_email, contact_name,
              phone, address, created_at, updated_at
              FROM tenants WHERE slug = ? AND status = 'active'`

    var tenant models.Tenant
    err := r.db.QueryRow(query, slug).Scan(
        &tenant.ID, &tenant.Slug, &tenant.Name, &tenant.Status,
        &tenant.ContactEmail, &tenant.ContactName, &tenant.Phone,
        &tenant.Address, &tenant.CreatedAt, &tenant.UpdatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &tenant, nil
}
```

---

### 1.3 ProvisioningService Incomplete

**Priority:** CRITICAL
**Effort:** 2-3 hours
**File:** `internal/services/provisioning_service.go`

**Missing Methods:**

| Method | Purpose |
|--------|---------|
| `CreateDefaultColors(tx *sql.Tx, tenantID int)` | Create green/blue/orange dog categories |
| `CreateDefaultBookingRules(tx *sql.Tx, tenantID int)` | Create weekday/weekend time slots |
| `CreateDefaultSettings(tx *sql.Tx, tenantID int)` | Create system_settings entries |
| `ProvisionTenant(tenant *models.Tenant, adminUser *models.User)` | Full tenant setup |

**Implementation Requirements:**
```go
func (s *ProvisioningService) CreateDefaultColors(tx *sql.Tx, tenantID int) error {
    colors := []struct {
        Name  string
        Color string
        Level int
    }{
        {"Grün - Anfänger", "#82b965", 1},
        {"Orange - Fortgeschritten", "#f5a623", 2},
        {"Blau - Erfahren", "#4a90d9", 3},
    }

    for _, c := range colors {
        _, err := tx.Exec(`
            INSERT INTO dog_categories (tenant_id, name, color, experience_level)
            VALUES (?, ?, ?, ?)
        `, tenantID, c.Name, c.Color, c.Level)
        if err != nil {
            return err
        }
    }
    return nil
}
```

---

### 1.4 BaseDomain Configuration

**Priority:** CRITICAL
**Effort:** 30 minutes
**File:** `internal/config/config.go`

**Issue:** `BaseDomain` field may not be loaded from environment.

**Required:**
```go
type Config struct {
    // ... existing fields
    BaseDomain string `env:"BASE_DOMAIN" default:"localhost"`
}

// In LoadConfig():
cfg.BaseDomain = getEnv("BASE_DOMAIN", "localhost")
```

**Environment Variable:**
```bash
BASE_DOMAIN=gassigeher.org
```

---

## Phase 2: High Priority Features

### 2.1 Central Admin UI Pages

**Priority:** HIGH
**Effort:** 4-6 hours
**Location:** `internal/static/central/`

**Missing Pages:**

| File | Purpose |
|------|---------|
| `index.html` | Dashboard with platform stats |
| `tenants.html` | Tenant list, activate/suspend |
| `users.html` | User search across tenants |
| `assets/css/central.css` | Styles |
| `assets/js/central.js` | API interactions |

**API Endpoints Available:**
- `GET /api/central-admin/stats` - Platform statistics
- `GET /api/central-admin/tenants` - List all tenants
- `GET /api/central-admin/tenants/:id` - Tenant details
- `PUT /api/central-admin/tenants/:id` - Update tenant
- `POST /api/central-admin/tenants/:id/activate` - Activate
- `POST /api/central-admin/tenants/:id/suspend` - Suspend
- `GET /api/central-admin/users` - Search users

---

### 2.2 S3 Integration in Handlers

**Priority:** HIGH
**Effort:** 3-4 hours
**Files:** Multiple handlers

**Current State:** `S3Service` is fully implemented but not used by handlers.

**Required Changes:**

| Handler | Method | Change Needed |
|---------|--------|---------------|
| `dog_handler.go` | `UploadPhoto` | Use S3Service when `USE_S3=true` |
| `user_handler.go` | `UploadProfilePhoto` | Use S3Service when `USE_S3=true` |
| `settings_handler.go` | `UploadLogo` | Use S3Service when `USE_S3=true` |

**Implementation Pattern:**
```go
func (h *DogHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
    // ... existing validation ...

    if h.s3Service != nil && h.cfg.UseS3 {
        // Upload to S3
        tenantSlug := r.Context().Value(middleware.TenantSlugKey).(string)
        path := fmt.Sprintf("dogs/%d_%s", dogID, filename)
        url, err := h.s3Service.Upload(r.Context(), tenantSlug, path, data, contentType)
        if err != nil {
            respondError(w, http.StatusInternalServerError, "Failed to upload")
            return
        }
        dog.Photo = url
    } else {
        // Local storage (existing code)
    }
}
```

---

### 2.3 ThemeHandler Completion

**Priority:** HIGH
**Effort:** 2 hours
**File:** `internal/handlers/theme_handler.go`

**Missing Methods:**

```go
// GetCurrentTheme returns the current theme for a tenant
func (h *ThemeHandler) GetCurrentTheme(w http.ResponseWriter, r *http.Request) {
    tenantID := r.Context().Value(middleware.TenantIDKey).(int)
    settings, err := h.settingsRepo.GetTenantSettings(tenantID)
    // Return theme colors and preset name
}

// UpdateTheme updates the theme for a tenant (admin only)
func (h *ThemeHandler) UpdateTheme(w http.ResponseWriter, r *http.Request) {
    // Parse preset name or custom colors
    // Validate colors are valid hex
    // Update tenant_settings
}

// GetPresets returns available theme presets
func (h *ThemeHandler) GetPresets(w http.ResponseWriter, r *http.Request) {
    presets := []ThemePreset{
        {Name: "classic", Primary: "#82b965", ...},
        {Name: "ocean", Primary: "#0077b6", ...},
        // ... 10 presets
    }
    respondJSON(w, http.StatusOK, presets)
}
```

---

### 2.4 Stripe Webhook Verification

**Priority:** HIGH
**Effort:** 2-3 hours
**File:** `internal/handlers/billing_handler.go`

**Missing Implementation:**

```go
func (h *BillingHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    // Read body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read body", http.StatusBadRequest)
        return
    }

    // Verify signature
    sigHeader := r.Header.Get("Stripe-Signature")
    event, err := webhook.ConstructEvent(body, sigHeader, h.webhookSecret)
    if err != nil {
        http.Error(w, "Invalid signature", http.StatusBadRequest)
        return
    }

    // Handle events
    switch event.Type {
    case "checkout.session.completed":
        // Activate subscription
    case "invoice.paid":
        // Record payment
    case "customer.subscription.updated":
        // Update plan limits
    case "customer.subscription.deleted":
        // Downgrade to free
    }

    w.WriteHeader(http.StatusOK)
}
```

---

### 2.5 Tenant Welcome Email

**Priority:** HIGH
**Effort:** 1-2 hours
**File:** `internal/services/email_service.go`

**Add Method:**
```go
func (s *EmailService) SendTenantWelcomeEmail(to, tenantName, adminName, loginURL string) error {
    subject := fmt.Sprintf("Willkommen bei Gassigeher - %s ist bereit!", tenantName)

    tmpl := `
    <h2>Herzlich Willkommen bei Gassigeher!</h2>
    <p>Hallo {{.AdminName}},</p>
    <p>Ihr Tierheim <strong>{{.TenantName}}</strong> wurde erfolgreich eingerichtet.</p>
    <p>Sie können sich jetzt anmelden unter:</p>
    <p><a href="{{.LoginURL}}">{{.LoginURL}}</a></p>
    <h3>Nächste Schritte:</h3>
    <ol>
        <li>Loggen Sie sich ein</li>
        <li>Fügen Sie Ihre Hunde hinzu</li>
        <li>Laden Sie Freiwillige ein</li>
    </ol>
    `
    // ... template execution and send
}
```

---

## Phase 3: Medium Priority Features

### 3.1 Cron Jobs Tenant-Awareness

**Priority:** MEDIUM
**Effort:** 2-3 hours
**File:** `internal/cron/cron.go`

**Required Changes:**
- Auto-deactivation should run per tenant
- Booking reminders should use tenant email settings
- Holiday API should use tenant's federal state setting

```go
func (s *CronService) autoDeactivateInactiveUsers() {
    // Get all active tenants
    tenants, _ := s.tenantRepo.FindAllActive()

    for _, tenant := range tenants {
        // Get tenant-specific deactivation days
        days, _ := s.settingsRepo.Get(tenant.ID, "auto_deactivation_days")

        // Deactivate users for this tenant
        users, _ := s.userRepo.FindInactiveUsers(tenant.ID, days)
        for _, user := range users {
            s.userRepo.Deactivate(user.ID)
            // Send email using tenant branding
        }
    }
}
```

---

### 3.2 Tenant-Specific Rate Limiting

**Priority:** MEDIUM
**Effort:** 3-4 hours
**File:** `internal/middleware/ratelimit_tenant.go` (new)

**Implementation:**
```go
type TenantRateLimiter struct {
    limiters map[int]*rate.Limiter  // per tenant
    mu       sync.RWMutex
}

func TenantRateLimit(limiter *TenantRateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tenantID := r.Context().Value(TenantIDKey).(int)

            if !limiter.Allow(tenantID) {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

---

### 3.3 Subscription Sync from Webhooks

**Priority:** MEDIUM
**Effort:** 2-3 hours

**Required Logic:**
- When Pro subscription expires → set `max_dogs = 10`
- When payment fails → mark as `past_due`
- Grace period of 7 days before downgrade
- Email notifications for subscription changes

---

## Phase 4: Configuration & Documentation

### 4.1 Environment Variables

**File:** `.env.production.example`

```bash
# ===========================================
# Gassigeher SaaS Production Configuration
# ===========================================

# Application
PORT=8080
BASE_URL=https://gassigeher.org
BASE_DOMAIN=gassigeher.org
JWT_SECRET=<generate: openssl rand -base64 32>

# Database (PostgreSQL recommended for SaaS)
DB_TYPE=postgres
DB_HOST=postgres
DB_PORT=5432
DB_NAME=gassigeher
DB_USER=gassigeher
DB_PASSWORD=<secure-password>
DB_SSLMODE=require

# Super Admin (Central Admin)
SUPER_ADMIN_EMAIL=admin@gassigeher.org
CENTRAL_ADMIN_EMAIL=admin@gassigeher.org

# Email (Gmail API)
EMAIL_PROVIDER=gmail
GMAIL_CLIENT_ID=<from-google-cloud>
GMAIL_CLIENT_SECRET=<from-google-cloud>
GMAIL_REFRESH_TOKEN=<from-google-cloud>
GMAIL_FROM_EMAIL=noreply@gassigeher.org

# Contact Form
CONTACT_EMAIL=kontakt@gassigeher.org

# Hetzner S3 Object Storage
USE_S3=true
S3_ENDPOINT=fsn1.your-objectstorage.com
S3_ACCESS_KEY=<hetzner-access-key>
S3_SECRET_KEY=<hetzner-secret-key>
S3_BUCKET_NAME=gassigeher-uploads
S3_REGION=fsn1
S3_PUBLIC_URL=https://gassigeher-uploads.fsn1.your-objectstorage.com
S3_USE_SSL=true

# Stripe Billing
STRIPE_SECRET_KEY=sk_live_xxx
STRIPE_PUBLISHABLE_KEY=pk_live_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx
STRIPE_PRICE_MONTHLY=price_xxx
STRIPE_PRICE_YEARLY=price_xxx

# Hetzner DNS (for Caddy wildcard SSL)
HETZNER_DNS_TOKEN=<hetzner-dns-api-token>

# Rate Limiting
RATE_LIMIT_RPS=100
RATE_LIMIT_BURST=200

# Brute Force Protection
BRUTE_FORCE_MAX_ATTEMPTS=3
BRUTE_FORCE_LOCKOUT_BASE=30s
BRUTE_FORCE_MAX_LOCKOUT=30m

# System Defaults
BOOKING_ADVANCE_DAYS=14
CANCELLATION_NOTICE_HOURS=12
AUTO_DEACTIVATION_DAYS=365
```

---

### 4.2 DNS Configuration

**Required DNS Records for gassigeher.org:**

| Type | Name | Value | Purpose |
|------|------|-------|---------|
| A | @ | `<server-ip>` | Main domain |
| A | www | `<server-ip>` | www subdomain |
| A | admin | `<server-ip>` | Central admin |
| A | * | `<server-ip>` | Wildcard for tenants |
| TXT | _acme-challenge | (managed by Caddy) | SSL verification |

---

### 4.3 Caddy Configuration

**File:** `deploy/Caddyfile` (already exists)

```caddyfile
{
    acme_dns hetzner {env.HETZNER_DNS_TOKEN}
}

# Main landing page
gassigeher.org, www.gassigeher.org {
    reverse_proxy app:8080
    encode gzip

    header {
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
    }
}

# Central admin
admin.gassigeher.org {
    reverse_proxy app:8080
}

# Tenant subdomains (wildcard)
*.gassigeher.org {
    reverse_proxy app:8080
}
```

---

## Implementation Checklist

### Day 1: Critical Path
- [ ] Add TenantMiddleware to main.go router
- [ ] Implement TenantRepository.FindBySlug()
- [ ] Implement TenantRepository.CreateTx()
- [ ] Add BaseDomain to config
- [ ] Test subdomain resolution

### Day 2: Provisioning & Services
- [ ] Complete ProvisioningService methods
- [ ] Implement S3 integration in handlers
- [ ] Add SendTenantWelcomeEmail()
- [ ] Test tenant registration flow

### Day 3: Admin UI & Webhooks
- [ ] Create central admin HTML pages
- [ ] Complete ThemeHandler methods
- [ ] Implement Stripe webhook verification
- [ ] Test billing flow

### Day 4: Testing & Deployment
- [ ] End-to-end tenant registration test
- [ ] Payment flow test (Stripe test mode)
- [ ] Multi-tenant isolation test
- [ ] Deploy to staging environment
- [ ] Final production deployment

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Tenant isolation failure | CRITICAL | Test with multiple tenants, verify all queries include tenant_id |
| Payment webhook failures | HIGH | Implement webhook retry, logging, alerting |
| S3 bucket misconfiguration | HIGH | Test uploads before launch, verify CORS |
| Wildcard SSL issues | MEDIUM | Test Caddy DNS challenge, have backup plan |
| Email delivery failures | MEDIUM | Use Gmail API, monitor bounce rates |

---

## Success Criteria

Before declaring production-ready:

1. **Tenant Registration:** New tenant can register, receives welcome email, can log in
2. **Multi-tenant Isolation:** Tenant A cannot see Tenant B's data
3. **Billing:** Stripe checkout works, webhook updates subscription
4. **Central Admin:** Can view all tenants, activate/suspend
5. **Theme:** Tenant can change colors, changes reflect in UI
6. **S3:** Photos upload to S3, URLs resolve correctly
7. **SSL:** Wildcard certificate works for `*.gassigeher.org`

---

## Related Documentation

- [DEPLOYMENT.md](DEPLOYMENT.md) - General deployment guide
- [API.md](API.md) - API endpoint reference
- [ImplementationPlan.md](ImplementationPlan.md) - Original architecture plan
- [DatabasesSupportPlan.md](DatabasesSupportPlan.md) - Multi-database support

---

## Contact

For questions about this implementation plan, refer to the codebase or create an issue.
