# Gassigeher SaaS Transformation Plan

> **Transform Gassigeher from single-tenant to multi-tenant SaaS for 500 Tierheime**

**Version:** 1.0
**Date:** December 2025
**Status:** Ready for Implementation

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Overview](#2-architecture-overview)
3. [Phase 1: Database Foundation](#phase-1-database-foundation)
4. [Phase 2: Tenant Middleware & JWT](#phase-2-tenant-middleware--jwt)
5. [Phase 3: Repository Layer Updates](#phase-3-repository-layer-updates)
6. [Phase 4: Handler Layer Updates](#phase-4-handler-layer-updates)
7. [Phase 5: Enhanced Security](#phase-5-enhanced-security)
8. [Phase 6: Hetzner S3 Storage](#phase-6-hetzner-s3-storage)
9. [Phase 7: Theming System](#phase-7-theming-system)
10. [Phase 8: Tenant Registration](#phase-8-tenant-registration)
11. [Phase 9: Landing Page](#phase-9-landing-page)
12. [Phase 10: Central Admin Dashboard](#phase-10-central-admin-dashboard)
13. [Phase 11: Docker Infrastructure](#phase-11-docker-infrastructure)
14. [Phase 12: Testing & Migration](#phase-12-testing--migration)
15. [File Change Summary](#file-change-summary)
16. [Configuration Reference](#configuration-reference)

---

## 1. Executive Summary

### Goal
Transform Gassigeher into a multi-tenant SaaS platform serving 500+ German animal shelters (Tierheime) via `*.gassigeher.org` subdomains.

### Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Multi-tenancy** | Row-level isolation (tenant_id) | Single database, simple migrations, industry standard |
| **Database** | PostgreSQL with RLS | Row Level Security for database-enforced isolation |
| **Container strategy** | Single instance (start), pooled (scale) | Low maintenance, simple deployment |
| **File storage** | Hetzner S3 Object Storage | Scalable, cost-effective, GDPR-compliant |
| **Email** | Global SMTP (Strato/Hetzner) | Single server for all tenants |
| **SSL** | Caddy with wildcard certificate | Automatic HTTPS, DNS challenge |
| **Billing** | None (donation button only) | Community-driven model |

### Architecture Diagram

```
                                 ┌─────────────────────────┐
                                 │     gassigeher.org      │
                                 │    (Landing Page)       │
                                 └───────────┬─────────────┘
                                             │
┌────────────────────────────────────────────┼────────────────────────────────────────────┐
│                                            │                                             │
│   ┌─────────────────┐              ┌───────▼───────┐              ┌─────────────────┐   │
│   │  tierheim-a     │              │               │              │  tierheim-b     │   │
│   │  .gassigeher    │──────────────│    Caddy      │──────────────│  .gassigeher    │   │
│   │  .org           │              │  (Wildcard    │              │  .org           │   │
│   └─────────────────┘              │   SSL/TLS)    │              └─────────────────┘   │
│                                    └───────┬───────┘                                    │
│                                            │                                             │
│                                    ┌───────▼───────┐                                    │
│                                    │               │                                    │
│                                    │  Gassigeher   │                                    │
│                                    │  Go App       │                                    │
│                                    │  (Single)     │                                    │
│                                    └───────┬───────┘                                    │
│                                            │                                             │
│                         ┌──────────────────┼──────────────────┐                         │
│                         │                  │                  │                         │
│                 ┌───────▼───────┐  ┌───────▼───────┐  ┌───────▼───────┐                │
│                 │  PostgreSQL   │  │  Hetzner S3   │  │  Strato SMTP  │                │
│                 │  (Row-Level   │  │  (Photos/     │  │  (Global      │                │
│                 │   Security)   │  │   Files)      │  │   Email)      │                │
│                 └───────────────┘  └───────────────┘  └───────────────┘                │
│                                                                                         │
│                              Hetzner Cloud (Germany)                                    │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Architecture Overview

### Current State (Single-Tenant)

- **15 database tables** without tenant isolation
- **13 repositories** with direct SQL queries
- **14 handlers** serving HTTP requests
- **JWT auth** with user_id, email, is_admin, is_super_admin
- **Local file storage** in `/uploads/` directory
- **Rate limiting** on login only (5/min/IP)

### Target State (Multi-Tenant SaaS)

- **17 database tables** with tenant_id + PostgreSQL RLS
- **13 repositories** with automatic tenant filtering
- **16 handlers** (+ tenant, theme, central admin)
- **JWT auth** with tenant_id claim added
- **S3 file storage** organized by tenant
- **Comprehensive security** (rate limiting, brute force, lockout)

---

## Phase 1: Database Foundation

**Objective:** Create tenant infrastructure without breaking existing functionality.

### 1.1 New Table: `tenants`

```sql
-- internal/database/003_add_tenants.go

CREATE TABLE tenants (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(100) NOT NULL UNIQUE,      -- "tierheim-goeppingen"
    name VARCHAR(255) NOT NULL,              -- "Tierheim Göppingen"
    status VARCHAR(20) DEFAULT 'active',     -- active, suspended, deleted

    -- Contact
    contact_email VARCHAR(255) NOT NULL,
    contact_phone VARCHAR(50),
    address TEXT,
    city VARCHAR(100),
    postal_code VARCHAR(20),
    federal_state VARCHAR(50) DEFAULT 'BW',  -- For holiday API

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    suspended_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status);
```

### 1.2 New Table: `tenant_settings`

```sql
CREATE TABLE tenant_settings (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Branding (10 presets + custom)
    theme_preset VARCHAR(50) DEFAULT 'classic',
    color_primary VARCHAR(7),       -- #82b965
    color_secondary VARCHAR(7),     -- #26272b
    color_accent VARCHAR(7),        -- #4a90e2
    color_background VARCHAR(7),    -- #fef9f3
    color_text VARCHAR(7),          -- #2c3e34

    -- Logo
    logo_url VARCHAR(500),          -- S3 URL
    favicon_url VARCHAR(500),

    -- Content
    welcome_message TEXT,           -- Custom HTML welcome
    footer_text TEXT,

    -- External Links
    website_url VARCHAR(500),
    donation_url VARCHAR(500),      -- Buy me a coffee link

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id)
);
```

### 1.3 Add `tenant_id` to All 15 Tables

```sql
-- internal/database/004_add_tenant_ids.go

-- Core tables
ALTER TABLE users ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
ALTER TABLE dogs ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
ALTER TABLE bookings ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
ALTER TABLE blocked_dates ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
ALTER TABLE color_categories ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
ALTER TABLE user_colors ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);

-- Request tables
ALTER TABLE color_requests ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
ALTER TABLE experience_requests ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
ALTER TABLE reactivation_requests ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);

-- Config tables
ALTER TABLE system_settings ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
ALTER TABLE booking_time_rules ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
ALTER TABLE custom_holidays ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);

-- Report tables
ALTER TABLE walk_reports ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
ALTER TABLE walk_report_photos ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);

-- Cache (global, no tenant_id needed)
-- feiertage_cache stays global

-- Indexes
CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_dogs_tenant ON dogs(tenant_id);
CREATE INDEX idx_bookings_tenant ON bookings(tenant_id);
-- ... repeat for all tables
```

### 1.4 PostgreSQL Row Level Security (RLS)

```sql
-- internal/database/005_add_rls.go

-- Enable RLS on all tenant tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE dogs ENABLE ROW LEVEL SECURITY;
ALTER TABLE bookings ENABLE ROW LEVEL SECURITY;
-- ... all 14 tables

-- Create application role
CREATE ROLE gassigeher_app;

-- RLS Policy: Users can only see their tenant's data
CREATE POLICY tenant_isolation_users ON users
    FOR ALL
    TO gassigeher_app
    USING (tenant_id = current_setting('app.tenant_id', true)::INTEGER);

CREATE POLICY tenant_isolation_dogs ON dogs
    FOR ALL
    TO gassigeher_app
    USING (tenant_id = current_setting('app.tenant_id', true)::INTEGER);

-- ... repeat for all 14 tables

-- Grant permissions
GRANT ALL ON ALL TABLES IN SCHEMA public TO gassigeher_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO gassigeher_app;
```

### 1.5 Update Unique Constraints

```sql
-- internal/database/006_update_constraints.go

-- User email unique per tenant (not globally)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
CREATE UNIQUE INDEX idx_users_tenant_email ON users(tenant_id, email) WHERE email IS NOT NULL;

-- System settings key unique per tenant
ALTER TABLE system_settings DROP CONSTRAINT IF EXISTS system_settings_pkey;
ALTER TABLE system_settings ADD PRIMARY KEY (tenant_id, key);

-- Booking time rules unique per tenant
ALTER TABLE booking_time_rules DROP CONSTRAINT IF EXISTS booking_time_rules_day_type_rule_name_key;
CREATE UNIQUE INDEX idx_booking_rules_tenant ON booking_time_rules(tenant_id, day_type, rule_name);
```

### Files to Create

| File | Purpose |
|------|---------|
| `internal/database/003_add_tenants.go` | Create tenants + tenant_settings tables |
| `internal/database/004_add_tenant_ids.go` | Add tenant_id to all 14 tables |
| `internal/database/005_add_rls.go` | PostgreSQL Row Level Security policies |
| `internal/database/006_update_constraints.go` | Update unique constraints for multi-tenant |
| `internal/models/tenant.go` | Tenant and TenantSettings models |
| `internal/repository/tenant_repository.go` | Tenant CRUD operations |

---

## Phase 2: Tenant Middleware & JWT

**Objective:** Resolve tenant from subdomain and include in JWT tokens.

### 2.1 Tenant Middleware

```go
// internal/middleware/tenant.go

package middleware

import (
    "context"
    "net/http"
    "strings"

    "github.com/tranmh/gassigeher/internal/repository"
)

const TenantIDKey contextKey = "tenantID"
const TenantSlugKey contextKey = "tenantSlug"

func TenantMiddleware(tenantRepo *repository.TenantRepository, baseDomain string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            host := r.Host

            // Extract subdomain from host
            // e.g., "tierheim-goeppingen.gassigeher.org" → "tierheim-goeppingen"
            slug := extractSubdomain(host, baseDomain)

            if slug == "" || slug == "www" {
                // Main domain - serve landing page or redirect
                next.ServeHTTP(w, r)
                return
            }

            // Lookup tenant by slug
            tenant, err := tenantRepo.FindBySlug(slug)
            if err != nil || tenant == nil {
                http.Error(w, `{"error":"Tierheim nicht gefunden"}`, http.StatusNotFound)
                return
            }

            if tenant.Status != "active" {
                http.Error(w, `{"error":"Tierheim ist deaktiviert"}`, http.StatusForbidden)
                return
            }

            // Inject tenant into context
            ctx := context.WithValue(r.Context(), TenantIDKey, tenant.ID)
            ctx = context.WithValue(ctx, TenantSlugKey, tenant.Slug)

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func extractSubdomain(host, baseDomain string) string {
    // Remove port if present
    if idx := strings.Index(host, ":"); idx != -1 {
        host = host[:idx]
    }

    // Check if it's a subdomain of baseDomain
    if !strings.HasSuffix(host, "."+baseDomain) {
        return ""
    }

    // Extract subdomain
    subdomain := strings.TrimSuffix(host, "."+baseDomain)
    return subdomain
}
```

### 2.2 JWT Extension

```go
// internal/services/auth_service.go - Update GenerateJWT

func (s *AuthService) GenerateJWT(userID int, email string, isAdmin bool, isSuperAdmin bool, tenantID int) (string, error) {
    claims := jwt.MapClaims{
        "user_id":        userID,
        "email":          email,
        "is_admin":       isAdmin,
        "is_super_admin": isSuperAdmin,
        "tenant_id":      tenantID,  // NEW: Add tenant to token
        "exp":            time.Now().Add(time.Hour * time.Duration(s.jwtExpirationHours)).Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(s.jwtSecret))
}
```

### 2.3 Auth Middleware Update

```go
// internal/middleware/middleware.go - Update AuthMiddleware

func AuthMiddleware(authService *services.AuthService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // ... existing token extraction ...

            // Extract claims including tenant_id
            tenantID, _ := (*claims)["tenant_id"].(float64)

            // Validate tenant_id matches subdomain
            ctxTenantID, _ := r.Context().Value(TenantIDKey).(int)
            if ctxTenantID != 0 && ctxTenantID != int(tenantID) {
                respondError(w, http.StatusUnauthorized, "Token für anderes Tierheim")
                return
            }

            // Add to context
            ctx := context.WithValue(r.Context(), TenantIDKey, int(tenantID))
            ctx = context.WithValue(ctx, UserIDKey, int(userID))
            // ... rest of claims ...

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Files to Modify

| File | Changes |
|------|---------|
| `internal/middleware/middleware.go` | Add TenantIDKey, TenantSlugKey; update AuthMiddleware |
| `internal/services/auth_service.go` | Add tenantID parameter to GenerateJWT |
| `cmd/server/main.go` | Add TenantMiddleware to chain before AuthMiddleware |

### Files to Create

| File | Purpose |
|------|---------|
| `internal/middleware/tenant.go` | Subdomain → tenant resolution |

---

## Phase 3: Repository Layer Updates

**Objective:** Modify all repositories to automatically filter by tenant_id.

### 3.1 Base Repository Pattern

```go
// internal/repository/base_repository.go

package repository

import (
    "context"
    "database/sql"
)

type TenantAware interface {
    SetTenantContext(ctx context.Context, tenantID int) context.Context
}

// SetTenantSession sets PostgreSQL session variable for RLS
func SetTenantSession(ctx context.Context, db *sql.DB, tenantID int) error {
    _, err := db.ExecContext(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
    return err
}
```

### 3.2 Repository Update Pattern

Each repository needs these changes:

```go
// Example: internal/repository/dog_repository.go

// Before
func (r *DogRepository) FindAll(filter DogFilter) ([]*models.Dog, error) {
    query := `SELECT ... FROM dogs WHERE is_available = ?`
    // ...
}

// After
func (r *DogRepository) FindAll(ctx context.Context, tenantID int, filter DogFilter) ([]*models.Dog, error) {
    // Set RLS context
    if err := SetTenantSession(ctx, r.db, tenantID); err != nil {
        return nil, err
    }

    query := `SELECT ... FROM dogs WHERE tenant_id = $1 AND is_available = $2`
    rows, err := r.db.QueryContext(ctx, query, tenantID, filter.Available)
    // ...
}

// Create must include tenant_id
func (r *DogRepository) Create(ctx context.Context, tenantID int, dog *models.Dog) error {
    if err := SetTenantSession(ctx, r.db, tenantID); err != nil {
        return err
    }

    query := `INSERT INTO dogs (tenant_id, name, breed, ...) VALUES ($1, $2, $3, ...)`
    _, err := r.db.ExecContext(ctx, query, tenantID, dog.Name, dog.Breed, ...)
    return err
}
```

### 3.3 Repositories to Update (13 files)

| Repository | Methods to Update |
|------------|-------------------|
| `user_repository.go` | Create, FindByID, FindByEmail, FindAll, Update, Delete, etc. (12+ methods) |
| `dog_repository.go` | Create, FindByID, FindAll, Update, Delete, etc. (10+ methods) |
| `booking_repository.go` | Create, FindByID, FindByUser, FindByDog, Update, etc. (15+ methods) |
| `blocked_date_repository.go` | Create, FindByDate, FindAll, Delete (5 methods) |
| `color_category_repository.go` | Create, FindAll, FindByID, Update, Delete (5 methods) |
| `color_request_repository.go` | Create, FindByUser, FindAll, Update (5 methods) |
| `user_color_repository.go` | Create, FindByUser, SetUserColors, Delete (4 methods) |
| `experience_request_repository.go` | Create, FindByUser, FindAll, Update (5 methods) |
| `reactivation_request_repository.go` | Create, FindByUser, FindAll, Update (5 methods) |
| `settings_repository.go` | Get, Set, GetAll (3 methods) |
| `booking_time_repository.go` | GetRules, SaveRules, GetAvailableSlots (4 methods) |
| `holiday_repository.go` | Create, FindByDate, FindAll, Delete (5 methods) |
| `walk_report_repository.go` | Create, FindByBooking, Update, AddPhoto (6 methods) |

---

## Phase 4: Handler Layer Updates

**Objective:** Extract tenant_id from context and pass to repositories.

### 4.1 Handler Update Pattern

```go
// Example: internal/handlers/dog_handler.go

func (h *DogHandler) ListDogs(w http.ResponseWriter, r *http.Request) {
    // Extract tenant from context (set by TenantMiddleware)
    tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
    if !ok || tenantID == 0 {
        respondError(w, http.StatusBadRequest, "Tenant nicht gefunden")
        return
    }

    // Pass context and tenantID to repository
    dogs, err := h.dogRepo.FindAll(r.Context(), tenantID, filter)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Hunde")
        return
    }

    respondJSON(w, http.StatusOK, dogs)
}
```

### 4.2 Handlers to Update (14 files)

| Handler | Methods to Update |
|---------|-------------------|
| `auth_handler.go` | Register, Login, VerifyEmail, ResetPassword |
| `user_handler.go` | GetMe, UpdateMe, GetUsers, DeleteAccount, etc. |
| `dog_handler.go` | ListDogs, GetDog, CreateDog, UpdateDog, etc. |
| `booking_handler.go` | CreateBooking, GetBookings, CancelBooking, etc. |
| `blocked_date_handler.go` | Create, List, Delete |
| `settings_handler.go` | GetSettings, UpdateSettings |
| `experience_request_handler.go` | Create, List, Approve, Deny |
| `reactivation_request_handler.go` | Create, List, Approve, Deny |
| `dashboard_handler.go` | GetStats, GetAdminStats |
| `walk_report_handler.go` | Create, Get, AddPhoto |
| `color_category_handler.go` | List, Create, Update, Delete |
| `color_request_handler.go` | Create, List, Approve, Deny |
| `user_color_handler.go` | GetUserColors, SetUserColors |
| `booking_time_handler.go` | GetRules, SaveRules, GetSlots |
| `holiday_handler.go` | List, Create, Delete |

---

## Phase 5: Enhanced Security

**Objective:** Implement comprehensive rate limiting and brute force protection.

### 5.1 Global Rate Limiter

```go
// internal/middleware/ratelimit_global.go

package middleware

import (
    "net/http"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

type GlobalRateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rps      rate.Limit  // requests per second
    burst    int
}

func NewGlobalRateLimiter(rps float64, burst int) *GlobalRateLimiter {
    return &GlobalRateLimiter{
        limiters: make(map[string]*rate.Limiter),
        rps:      rate.Limit(rps),
        burst:    burst,
    }
}

func (g *GlobalRateLimiter) GetLimiter(ip string) *rate.Limiter {
    g.mu.Lock()
    defer g.mu.Unlock()

    limiter, exists := g.limiters[ip]
    if !exists {
        limiter = rate.NewLimiter(g.rps, g.burst)
        g.limiters[ip] = limiter
    }
    return limiter
}

func GlobalRateLimit(rps float64, burst int) func(http.Handler) http.Handler {
    limiter := NewGlobalRateLimiter(rps, burst)

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := getClientIP(r)

            if !limiter.GetLimiter(ip).Allow() {
                w.Header().Set("Retry-After", "60")
                http.Error(w, `{"error":"Zu viele Anfragen"}`, http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### 5.2 Brute Force Protection

```go
// internal/services/brute_force_service.go

package services

import (
    "sync"
    "time"
)

type BruteForceService struct {
    failures    map[string]*FailureRecord
    mu          sync.RWMutex
    maxAttempts int           // 3
    lockoutBase time.Duration // 30 seconds
    maxLockout  time.Duration // 30 minutes
}

type FailureRecord struct {
    Count       int
    LastFailed  time.Time
    LockedUntil time.Time
}

func NewBruteForceService() *BruteForceService {
    return &BruteForceService{
        failures:    make(map[string]*FailureRecord),
        maxAttempts: 3,
        lockoutBase: 30 * time.Second,
        maxLockout:  30 * time.Minute,
    }
}

func (s *BruteForceService) RecordFailure(key string) time.Duration {
    s.mu.Lock()
    defer s.mu.Unlock()

    record, exists := s.failures[key]
    if !exists {
        record = &FailureRecord{}
        s.failures[key] = record
    }

    record.Count++
    record.LastFailed = time.Now()

    if record.Count >= s.maxAttempts {
        // Exponential backoff: 30s, 60s, 120s, 240s... max 30min
        multiplier := 1 << (record.Count - s.maxAttempts)
        delay := s.lockoutBase * time.Duration(multiplier)
        if delay > s.maxLockout {
            delay = s.maxLockout
        }
        record.LockedUntil = time.Now().Add(delay)
        return delay
    }

    return 0
}

func (s *BruteForceService) IsLocked(key string) (bool, time.Duration) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    record, exists := s.failures[key]
    if !exists {
        return false, 0
    }

    if time.Now().Before(record.LockedUntil) {
        return true, time.Until(record.LockedUntil)
    }

    return false, 0
}

func (s *BruteForceService) ClearFailures(key string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.failures, key)
}
```

### 5.3 Integration with Login Handler

```go
// internal/handlers/auth_handler.go - Update Login

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
        return
    }

    // Check brute force lockout (by email + IP)
    lockKey := req.Email + ":" + getClientIP(r)
    if locked, remaining := h.bruteForce.IsLocked(lockKey); locked {
        respondError(w, http.StatusTooManyRequests,
            fmt.Sprintf("Konto gesperrt. Versuchen Sie es in %d Sekunden erneut.", int(remaining.Seconds())))
        return
    }

    // Attempt login
    user, err := h.userRepo.FindByEmail(r.Context(), tenantID, req.Email)
    if err != nil || !h.authService.CheckPassword(req.Password, user.PasswordHash) {
        // Record failure
        delay := h.bruteForce.RecordFailure(lockKey)
        if delay > 0 {
            respondError(w, http.StatusTooManyRequests,
                fmt.Sprintf("Zu viele Fehlversuche. Konto für %d Sekunden gesperrt.", int(delay.Seconds())))
        } else {
            respondError(w, http.StatusUnauthorized, "Ungültige Anmeldedaten")
        }
        return
    }

    // Success - clear failures
    h.bruteForce.ClearFailures(lockKey)

    // Generate token with tenant_id
    token, err := h.authService.GenerateJWT(user.ID, user.Email, user.IsAdmin, user.IsSuperAdmin, tenantID)
    // ...
}
```

### 5.4 Database Changes for Account Lockout

```sql
-- internal/database/007_add_lockout_fields.go

ALTER TABLE users ADD COLUMN failed_login_attempts INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN locked_until TIMESTAMP;
ALTER TABLE users ADD COLUMN last_failed_login TIMESTAMP;
```

### Files to Create

| File | Purpose |
|------|---------|
| `internal/middleware/ratelimit_global.go` | Global rate limiting for all endpoints |
| `internal/services/brute_force_service.go` | Brute force detection and lockout |
| `internal/database/007_add_lockout_fields.go` | Add lockout fields to users table |

### Files to Modify

| File | Changes |
|------|---------|
| `internal/handlers/auth_handler.go` | Integrate brute force service |
| `internal/repository/user_repository.go` | Add lockout methods |
| `cmd/server/main.go` | Add global rate limiter to middleware chain |

---

## Phase 6: Hetzner S3 Storage

**Objective:** Replace local filesystem with S3-compatible object storage.

### 6.1 S3 Service

```go
// internal/services/s3_service.go

package services

import (
    "bytes"
    "context"
    "fmt"
    "io"

    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Service struct {
    client     *minio.Client
    bucketName string
    publicURL  string
}

type S3Config struct {
    Endpoint   string // e.g., "fsn1.your-objectstorage.com"
    AccessKey  string
    SecretKey  string
    BucketName string
    Region     string
    PublicURL  string // e.g., "https://gassigeher-uploads.fsn1.your-objectstorage.com"
    UseSSL     bool
}

func NewS3Service(cfg *S3Config) (*S3Service, error) {
    client, err := minio.New(cfg.Endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
        Secure: cfg.UseSSL,
        Region: cfg.Region,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create S3 client: %w", err)
    }

    return &S3Service{
        client:     client,
        bucketName: cfg.BucketName,
        publicURL:  cfg.PublicURL,
    }, nil
}

func (s *S3Service) Upload(ctx context.Context, tenantSlug, path string, data []byte, contentType string) (string, error) {
    // Organize by tenant: tenants/{slug}/{path}
    objectKey := fmt.Sprintf("tenants/%s/%s", tenantSlug, path)

    _, err := s.client.PutObject(ctx, s.bucketName, objectKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
        ContentType: contentType,
    })
    if err != nil {
        return "", fmt.Errorf("failed to upload to S3: %w", err)
    }

    // Return public URL
    return fmt.Sprintf("%s/%s", s.publicURL, objectKey), nil
}

func (s *S3Service) Delete(ctx context.Context, objectKey string) error {
    return s.client.RemoveObject(ctx, s.bucketName, objectKey, minio.RemoveObjectOptions{})
}

func (s *S3Service) GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
    return s.client.PresignedGetObject(ctx, s.bucketName, objectKey, expiry, nil)
}
```

### 6.2 Update Image Service

```go
// internal/services/image_service.go - Update to use S3

type ImageService struct {
    s3Service  *S3Service
    tenantSlug string
    localDir   string // Fallback for development
    useS3      bool
}

func (s *ImageService) ProcessDogPhoto(ctx context.Context, file multipart.File, dogID int) (fullURL, thumbURL string, err error) {
    // Decode and process image
    img, err := imaging.Decode(file)
    if err != nil {
        return "", "", err
    }

    // Resize full image
    fullImg := imaging.Fit(img, MaxImageWidth, MaxImageHeight, imaging.Lanczos)
    var fullBuf bytes.Buffer
    if err := imaging.Encode(&fullBuf, fullImg, imaging.JPEG, imaging.JPEGQuality(JPEGQuality)); err != nil {
        return "", "", err
    }

    // Create thumbnail
    thumbImg := imaging.Fill(img, ThumbnailSize, ThumbnailSize, imaging.Center, imaging.Lanczos)
    var thumbBuf bytes.Buffer
    if err := imaging.Encode(&thumbBuf, thumbImg, imaging.JPEG, imaging.JPEGQuality(JPEGQuality)); err != nil {
        return "", "", err
    }

    if s.useS3 {
        // Upload to S3
        fullURL, err = s.s3Service.Upload(ctx, s.tenantSlug,
            fmt.Sprintf("dogs/dog_%d_full.jpg", dogID), fullBuf.Bytes(), "image/jpeg")
        if err != nil {
            return "", "", err
        }

        thumbURL, err = s.s3Service.Upload(ctx, s.tenantSlug,
            fmt.Sprintf("dogs/dog_%d_thumb.jpg", dogID), thumbBuf.Bytes(), "image/jpeg")
        if err != nil {
            return "", "", err
        }
    } else {
        // Local storage fallback
        // ... existing local storage code ...
    }

    return fullURL, thumbURL, nil
}
```

### 6.3 Configuration

```go
// internal/config/config.go - Add S3 config

type Config struct {
    // ... existing fields ...

    // S3 Storage (Hetzner Object Storage)
    UseS3         bool
    S3Endpoint    string
    S3AccessKey   string
    S3SecretKey   string
    S3BucketName  string
    S3Region      string
    S3PublicURL   string
}
```

### Environment Variables

```env
# Hetzner S3 Object Storage
USE_S3=true
S3_ENDPOINT=fsn1.your-objectstorage.com
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
S3_BUCKET_NAME=gassigeher-uploads
S3_REGION=fsn1
S3_PUBLIC_URL=https://gassigeher-uploads.fsn1.your-objectstorage.com
```

### Files to Create

| File | Purpose |
|------|---------|
| `internal/services/s3_service.go` | S3 upload/delete operations |

### Files to Modify

| File | Changes |
|------|---------|
| `internal/services/image_service.go` | Use S3 for storage |
| `internal/config/config.go` | Add S3 configuration |
| `internal/handlers/dog_handler.go` | Pass tenant slug to image service |
| `internal/handlers/settings_handler.go` | Logo upload to S3 |
| `internal/handlers/walk_report_handler.go` | Report photos to S3 |

---

## Phase 7: Theming System

**Objective:** Implement 10 color presets + custom 5 colors.

### 7.1 Theme Presets

```go
// internal/models/theme.go

package models

type ThemeColors struct {
    Primary    string `json:"primary"`    // Main brand color
    Secondary  string `json:"secondary"`  // Dark/contrast color
    Accent     string `json:"accent"`     // Highlight color
    Background string `json:"background"` // Page background
    Text       string `json:"text"`       // Main text color
}

var ThemePresets = map[string]ThemeColors{
    "classic": {
        Primary:    "#82b965",
        Secondary:  "#26272b",
        Accent:     "#4a90e2",
        Background: "#fef9f3",
        Text:       "#2c3e34",
    },
    "ocean": {
        Primary:    "#0077b6",
        Secondary:  "#023e8a",
        Accent:     "#48cae4",
        Background: "#f0f9ff",
        Text:       "#1a365d",
    },
    "forest": {
        Primary:    "#2d6a4f",
        Secondary:  "#1b4332",
        Accent:     "#52b788",
        Background: "#f0fdf4",
        Text:       "#14532d",
    },
    "sunset": {
        Primary:    "#f97316",
        Secondary:  "#7c2d12",
        Accent:     "#fb923c",
        Background: "#fff7ed",
        Text:       "#431407",
    },
    "lavender": {
        Primary:    "#7c3aed",
        Secondary:  "#4c1d95",
        Accent:     "#a78bfa",
        Background: "#faf5ff",
        Text:       "#3b0764",
    },
    "coral": {
        Primary:    "#f43f5e",
        Secondary:  "#881337",
        Accent:     "#fb7185",
        Background: "#fff1f2",
        Text:       "#4c0519",
    },
    "midnight": {
        Primary:    "#3b82f6",
        Secondary:  "#1e3a5f",
        Accent:     "#60a5fa",
        Background: "#f8fafc",
        Text:       "#0f172a",
    },
    "emerald": {
        Primary:    "#10b981",
        Secondary:  "#064e3b",
        Accent:     "#34d399",
        Background: "#ecfdf5",
        Text:       "#022c22",
    },
    "rose": {
        Primary:    "#ec4899",
        Secondary:  "#831843",
        Accent:     "#f472b6",
        Background: "#fdf2f8",
        Text:       "#500724",
    },
    "slate": {
        Primary:    "#64748b",
        Secondary:  "#334155",
        Accent:     "#94a3b8",
        Background: "#f8fafc",
        Text:       "#1e293b",
    },
}
```

### 7.2 Dynamic CSS Endpoint

```go
// internal/handlers/theme_handler.go

package handlers

import (
    "fmt"
    "net/http"

    "github.com/tranmh/gassigeher/internal/middleware"
    "github.com/tranmh/gassigeher/internal/models"
    "github.com/tranmh/gassigeher/internal/repository"
)

type ThemeHandler struct {
    tenantRepo *repository.TenantRepository
}

func (h *ThemeHandler) GetCSS(w http.ResponseWriter, r *http.Request) {
    tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

    settings, err := h.tenantRepo.GetSettings(r.Context(), tenantID)
    if err != nil {
        http.Error(w, "Theme not found", http.StatusInternalServerError)
        return
    }

    // Get colors (custom or preset)
    var colors models.ThemeColors
    if settings.ColorPrimary != "" {
        colors = models.ThemeColors{
            Primary:    settings.ColorPrimary,
            Secondary:  settings.ColorSecondary,
            Accent:     settings.ColorAccent,
            Background: settings.ColorBackground,
            Text:       settings.ColorText,
        }
    } else {
        colors = models.ThemePresets[settings.ThemePreset]
    }

    css := fmt.Sprintf(`:root {
    --color-primary: %s;
    --color-secondary: %s;
    --color-accent: %s;
    --color-background: %s;
    --color-text: %s;
}`, colors.Primary, colors.Secondary, colors.Accent, colors.Background, colors.Text)

    w.Header().Set("Content-Type", "text/css")
    w.Header().Set("Cache-Control", "public, max-age=3600")
    w.Write([]byte(css))
}

func (h *ThemeHandler) GetPresets(w http.ResponseWriter, r *http.Request) {
    respondJSON(w, http.StatusOK, models.ThemePresets)
}
```

### 7.3 Frontend Integration

```html
<!-- In all HTML pages, add before main.css -->
<link rel="stylesheet" href="/api/theme/css">
<link rel="stylesheet" href="/assets/css/main.css">
```

### 7.4 New API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `GET /api/theme/css` | GET | Dynamic CSS variables |
| `GET /api/theme/presets` | GET | List all 10 presets |
| `PUT /api/admin/theme` | PUT | Update tenant theme (admin) |

### Files to Create

| File | Purpose |
|------|---------|
| `internal/models/theme.go` | Theme presets and colors |
| `internal/handlers/theme_handler.go` | CSS generation, preset listing |

### Files to Modify

| File | Changes |
|------|---------|
| `cmd/server/main.go` | Add theme routes |
| `frontend/*.html` | Add theme CSS link |

---

## Phase 8: Tenant Registration

**Objective:** Build self-service tenant registration flow.

### 8.1 Registration Handler

```go
// internal/handlers/tenant_handler.go

package handlers

type TenantRegistrationRequest struct {
    // Organization
    OrganizationName string `json:"organization_name" validate:"required,min=3,max=255"`
    Slug             string `json:"slug" validate:"required,min=3,max=100,slug"`
    ContactEmail     string `json:"contact_email" validate:"required,email"`
    ContactPhone     string `json:"contact_phone"`
    Address          string `json:"address"`
    City             string `json:"city" validate:"required"`
    PostalCode       string `json:"postal_code" validate:"required"`
    FederalState     string `json:"federal_state" validate:"required"`

    // Admin User
    AdminFirstName string `json:"admin_first_name" validate:"required"`
    AdminLastName  string `json:"admin_last_name" validate:"required"`
    AdminEmail     string `json:"admin_email" validate:"required,email"`
    AdminPassword  string `json:"admin_password" validate:"required,min=8"`
}

func (h *TenantHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req TenantRegistrationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
        return
    }

    // Validate slug format (lowercase, alphanumeric, hyphens)
    if !isValidSlug(req.Slug) {
        respondError(w, http.StatusBadRequest, "Ungültiger Subdomain-Name")
        return
    }

    // Check slug availability
    existing, _ := h.tenantRepo.FindBySlug(req.Slug)
    if existing != nil {
        respondError(w, http.StatusConflict, "Diese Subdomain ist bereits vergeben")
        return
    }

    // Start transaction
    tx, err := h.db.Begin()
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Datenbankfehler")
        return
    }
    defer tx.Rollback()

    // 1. Create tenant
    tenant := &models.Tenant{
        Slug:         req.Slug,
        Name:         req.OrganizationName,
        Status:       "active",
        ContactEmail: req.ContactEmail,
        ContactPhone: req.ContactPhone,
        Address:      req.Address,
        City:         req.City,
        PostalCode:   req.PostalCode,
        FederalState: req.FederalState,
    }
    tenantID, err := h.tenantRepo.CreateTx(tx, tenant)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen")
        return
    }

    // 2. Create default settings
    settings := &models.TenantSettings{
        TenantID:    tenantID,
        ThemePreset: "classic",
    }
    if err := h.tenantRepo.CreateSettingsTx(tx, settings); err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen der Einstellungen")
        return
    }

    // 3. Create super admin user
    hashedPassword, _ := h.authService.HashPassword(req.AdminPassword)
    admin := &models.User{
        TenantID:     tenantID,
        FirstName:    req.AdminFirstName,
        LastName:     req.AdminLastName,
        Email:        req.AdminEmail,
        PasswordHash: hashedPassword,
        IsAdmin:      true,
        IsSuperAdmin: true,
        IsVerified:   true, // Skip verification for initial admin
        IsActive:     true,
    }
    if err := h.userRepo.CreateTx(tx, admin); err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen des Admins")
        return
    }

    // 4. Create default color categories
    if err := h.provisioningService.CreateDefaultColors(tx, tenantID); err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler bei der Einrichtung")
        return
    }

    // 5. Create default booking time rules
    if err := h.provisioningService.CreateDefaultBookingRules(tx, tenantID); err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler bei der Einrichtung")
        return
    }

    // 6. Create default system settings
    if err := h.provisioningService.CreateDefaultSettings(tx, tenantID); err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler bei der Einrichtung")
        return
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Speichern")
        return
    }

    // 7. Send welcome email
    go h.emailService.SendTenantWelcomeEmail(req.ContactEmail, req.OrganizationName, req.Slug, req.AdminEmail)

    respondJSON(w, http.StatusCreated, map[string]interface{}{
        "message":   "Tierheim erfolgreich registriert",
        "slug":      req.Slug,
        "login_url": fmt.Sprintf("https://%s.%s/login", req.Slug, h.baseDomain),
    })
}

func (h *TenantHandler) CheckSlug(w http.ResponseWriter, r *http.Request) {
    slug := r.URL.Query().Get("slug")
    if slug == "" {
        respondError(w, http.StatusBadRequest, "Slug erforderlich")
        return
    }

    existing, _ := h.tenantRepo.FindBySlug(slug)
    available := existing == nil && isValidSlug(slug)

    respondJSON(w, http.StatusOK, map[string]bool{"available": available})
}
```

### 8.2 Provisioning Service

```go
// internal/services/provisioning_service.go

package services

type ProvisioningService struct {
    db *sql.DB
}

func (s *ProvisioningService) CreateDefaultColors(tx *sql.Tx, tenantID int) error {
    colors := []struct {
        Name      string
        HexCode   string
        SortOrder int
    }{
        {"Grün", "#22c55e", 1},
        {"Gelb", "#eab308", 2},
        {"Orange", "#f97316", 3},
        {"Rot", "#ef4444", 4},
        {"Blau", "#3b82f6", 5},
    }

    for _, c := range colors {
        _, err := tx.Exec(`INSERT INTO color_categories (tenant_id, name, hex_code, sort_order) VALUES ($1, $2, $3, $4)`,
            tenantID, c.Name, c.HexCode, c.SortOrder)
        if err != nil {
            return err
        }
    }
    return nil
}

func (s *ProvisioningService) CreateDefaultBookingRules(tx *sql.Tx, tenantID int) error {
    rules := []struct {
        DayType   string
        RuleName  string
        StartTime string
        EndTime   string
        IsBlocked bool
    }{
        {"weekday", "morning", "08:00", "12:00", false},
        {"weekday", "lunch", "12:00", "14:00", true},
        {"weekday", "afternoon", "14:00", "18:00", false},
        {"weekend", "morning", "09:00", "12:00", false},
        {"weekend", "afternoon", "14:00", "17:00", false},
        {"holiday", "morning", "10:00", "12:00", false},
        {"holiday", "afternoon", "14:00", "16:00", false},
    }

    for _, r := range rules {
        _, err := tx.Exec(`INSERT INTO booking_time_rules (tenant_id, day_type, rule_name, start_time, end_time, is_blocked) VALUES ($1, $2, $3, $4, $5, $6)`,
            tenantID, r.DayType, r.RuleName, r.StartTime, r.EndTime, r.IsBlocked)
        if err != nil {
            return err
        }
    }
    return nil
}

func (s *ProvisioningService) CreateDefaultSettings(tx *sql.Tx, tenantID int) error {
    settings := map[string]string{
        "booking_advance_days":      "14",
        "cancellation_notice_hours": "12",
        "auto_deactivation_days":    "365",
    }

    for key, value := range settings {
        _, err := tx.Exec(`INSERT INTO system_settings (tenant_id, key, value) VALUES ($1, $2, $3)`,
            tenantID, key, value)
        if err != nil {
            return err
        }
    }
    return nil
}
```

### 8.3 New API Endpoints

| Endpoint | Method | Auth | Purpose |
|----------|--------|------|---------|
| `POST /api/tenants/register` | POST | Public | Register new tenant |
| `GET /api/tenants/check-slug` | GET | Public | Check slug availability |
| `GET /api/tenants/me` | GET | Auth | Get current tenant info |
| `PUT /api/tenants/me` | PUT | Admin | Update tenant info |

### Files to Create

| File | Purpose |
|------|---------|
| `internal/handlers/tenant_handler.go` | Registration, slug check |
| `internal/services/provisioning_service.go` | Default data setup |

---

## Phase 9: Landing Page

**Objective:** Build public landing page at gassigeher.org.

### 9.1 Landing Page Structure

```
internal/static/landing/
├── index.html           # Main landing page
├── register.html        # Registration form
├── pricing.html         # Donation info
├── features.html        # Feature showcase
├── faq.html            # FAQ
├── imprint.html        # Impressum
├── privacy.html        # Datenschutz
├── assets/
│   ├── css/
│   │   └── landing.css
│   ├── js/
│   │   └── landing.js
│   └── images/
│       ├── hero.jpg
│       ├── screenshots/
│       └── logos/
└── i18n/
    └── de.json
```

### 9.2 Registration Form (register.html)

```html
<!DOCTYPE html>
<html lang="de">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Tierheim registrieren - Gassigeher</title>
    <link rel="stylesheet" href="/landing/assets/css/landing.css">
</head>
<body>
    <main class="register-page">
        <h1>Tierheim registrieren</h1>
        <p>Starten Sie in wenigen Minuten mit Gassigeher.</p>

        <form id="register-form">
            <section>
                <h2>Tierheim-Informationen</h2>

                <div class="form-group">
                    <label for="organization_name">Name des Tierheims *</label>
                    <input type="text" id="organization_name" name="organization_name" required>
                </div>

                <div class="form-group">
                    <label for="slug">Subdomain *</label>
                    <div class="slug-input">
                        <input type="text" id="slug" name="slug" required
                               pattern="[a-z0-9-]+"
                               placeholder="tierheim-musterstadt">
                        <span>.gassigeher.org</span>
                    </div>
                    <span id="slug-status"></span>
                </div>

                <div class="form-row">
                    <div class="form-group">
                        <label for="contact_email">E-Mail *</label>
                        <input type="email" id="contact_email" name="contact_email" required>
                    </div>
                    <div class="form-group">
                        <label for="contact_phone">Telefon</label>
                        <input type="tel" id="contact_phone" name="contact_phone">
                    </div>
                </div>

                <div class="form-group">
                    <label for="address">Adresse</label>
                    <input type="text" id="address" name="address">
                </div>

                <div class="form-row">
                    <div class="form-group">
                        <label for="postal_code">PLZ *</label>
                        <input type="text" id="postal_code" name="postal_code" required>
                    </div>
                    <div class="form-group">
                        <label for="city">Stadt *</label>
                        <input type="text" id="city" name="city" required>
                    </div>
                </div>

                <div class="form-group">
                    <label for="federal_state">Bundesland *</label>
                    <select id="federal_state" name="federal_state" required>
                        <option value="BW">Baden-Württemberg</option>
                        <option value="BY">Bayern</option>
                        <!-- ... all 16 states ... -->
                    </select>
                </div>
            </section>

            <section>
                <h2>Administrator-Konto</h2>

                <div class="form-row">
                    <div class="form-group">
                        <label for="admin_first_name">Vorname *</label>
                        <input type="text" id="admin_first_name" name="admin_first_name" required>
                    </div>
                    <div class="form-group">
                        <label for="admin_last_name">Nachname *</label>
                        <input type="text" id="admin_last_name" name="admin_last_name" required>
                    </div>
                </div>

                <div class="form-group">
                    <label for="admin_email">Admin E-Mail *</label>
                    <input type="email" id="admin_email" name="admin_email" required>
                </div>

                <div class="form-group">
                    <label for="admin_password">Passwort *</label>
                    <input type="password" id="admin_password" name="admin_password"
                           required minlength="8">
                    <small>Mindestens 8 Zeichen</small>
                </div>
            </section>

            <div class="form-group">
                <label>
                    <input type="checkbox" required>
                    Ich akzeptiere die <a href="/privacy.html">Datenschutzerklärung</a>
                    und <a href="/terms.html">Nutzungsbedingungen</a>
                </label>
            </div>

            <button type="submit" class="btn-primary">Tierheim registrieren</button>
        </form>

        <div id="success-message" style="display: none;">
            <h2>Registrierung erfolgreich!</h2>
            <p>Ihr Tierheim wurde erfolgreich eingerichtet.</p>
            <p>Sie können sich jetzt anmelden unter:</p>
            <a id="login-link" class="btn-primary" href="">Zum Login</a>
        </div>
    </main>

    <script src="/landing/assets/js/landing.js"></script>
</body>
</html>
```

### 9.3 Donation Section

```html
<!-- In index.html or pricing.html -->
<section class="donation">
    <h2>Unterstützen Sie Gassigeher</h2>
    <p>Gassigeher ist kostenlos für alle Tierheime.
       Wenn Sie unsere Arbeit unterstützen möchten:</p>

    <a href="https://buymeacoffee.com/gassigeher" class="btn-donation">
        <img src="/landing/assets/images/bmc-logo.svg" alt="">
        Buy me a coffee
    </a>

    <p class="donation-note">
        100% der Spenden fließen in Hosting und Weiterentwicklung.
    </p>
</section>
```

### Files to Create

| File | Purpose |
|------|---------|
| `internal/static/landing/index.html` | Main landing page |
| `internal/static/landing/register.html` | Registration form |
| `internal/static/landing/features.html` | Feature showcase |
| `internal/static/landing/faq.html` | FAQ |
| `internal/static/landing/imprint.html` | Impressum (legal) |
| `internal/static/landing/privacy.html` | Datenschutz |
| `internal/static/landing/assets/css/landing.css` | Landing styles |
| `internal/static/landing/assets/js/landing.js` | Registration logic |

---

## Phase 10: Central Admin Dashboard

**Objective:** Build dashboard for Tierschutzbund to manage all tenants.

### 10.1 Central Admin Role

```sql
-- internal/database/008_add_central_admin.go

-- Central admin is a special user not tied to any tenant
ALTER TABLE users ADD COLUMN is_central_admin BOOLEAN DEFAULT FALSE;

-- Create central admin user (run once)
INSERT INTO users (tenant_id, first_name, last_name, email, password_hash,
                   is_admin, is_super_admin, is_central_admin, is_verified, is_active)
VALUES (NULL, 'Central', 'Admin', 'admin@gassigeher.org', '$2a$12$...',
        true, true, true, true, true);
```

### 10.2 Central Admin Handler

```go
// internal/handlers/central_admin_handler.go

package handlers

type CentralAdminHandler struct {
    tenantRepo *repository.TenantRepository
    userRepo   *repository.UserRepository
    db         *sql.DB
}

func (h *CentralAdminHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
    // Verify central admin
    isCentralAdmin, _ := r.Context().Value(middleware.IsCentralAdminKey).(bool)
    if !isCentralAdmin {
        respondError(w, http.StatusForbidden, "Zugriff verweigert")
        return
    }

    // Get query params
    status := r.URL.Query().Get("status") // active, suspended, all
    search := r.URL.Query().Get("search")

    tenants, err := h.tenantRepo.FindAllWithStats(r.Context(), status, search)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Laden")
        return
    }

    respondJSON(w, http.StatusOK, tenants)
}

func (h *CentralAdminHandler) GetTenantStats(w http.ResponseWriter, r *http.Request) {
    tenantID, _ := strconv.Atoi(mux.Vars(r)["id"])

    stats, err := h.tenantRepo.GetStats(r.Context(), tenantID)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Laden")
        return
    }

    respondJSON(w, http.StatusOK, stats)
}

func (h *CentralAdminHandler) SuspendTenant(w http.ResponseWriter, r *http.Request) {
    tenantID, _ := strconv.Atoi(mux.Vars(r)["id"])

    var req struct {
        Reason string `json:"reason"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    if err := h.tenantRepo.Suspend(r.Context(), tenantID, req.Reason); err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Sperren")
        return
    }

    respondJSON(w, http.StatusOK, map[string]string{"message": "Tierheim gesperrt"})
}

func (h *CentralAdminHandler) ActivateTenant(w http.ResponseWriter, r *http.Request) {
    tenantID, _ := strconv.Atoi(mux.Vars(r)["id"])

    if err := h.tenantRepo.Activate(r.Context(), tenantID); err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Aktivieren")
        return
    }

    respondJSON(w, http.StatusOK, map[string]string{"message": "Tierheim aktiviert"})
}

func (h *CentralAdminHandler) GetPlatformStats(w http.ResponseWriter, r *http.Request) {
    stats := &PlatformStats{}

    // Total tenants
    h.db.QueryRow(`SELECT COUNT(*) FROM tenants WHERE status = 'active'`).Scan(&stats.ActiveTenants)
    h.db.QueryRow(`SELECT COUNT(*) FROM tenants`).Scan(&stats.TotalTenants)

    // Total users across all tenants
    h.db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_deleted = false`).Scan(&stats.TotalUsers)

    // Total dogs
    h.db.QueryRow(`SELECT COUNT(*) FROM dogs`).Scan(&stats.TotalDogs)

    // Bookings this month
    h.db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE date >= $1`,
        time.Now().Format("2006-01-01")).Scan(&stats.BookingsThisMonth)

    respondJSON(w, http.StatusOK, stats)
}
```

### 10.3 Central Admin Middleware

```go
// internal/middleware/central_admin.go

const IsCentralAdminKey contextKey = "isCentralAdmin"

func RequireCentralAdmin(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        isCentralAdmin, _ := r.Context().Value(IsCentralAdminKey).(bool)
        if !isCentralAdmin {
            http.Error(w, `{"error":"Central Admin access required"}`, http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### 10.4 Central Admin Pages

```
internal/static/central/
├── index.html           # Dashboard overview
├── tenants.html         # Tenant list
├── tenant-detail.html   # Individual tenant view
├── assets/
│   ├── css/central.css
│   └── js/central.js
└── i18n/de.json
```

### 10.5 New API Endpoints

| Endpoint | Method | Auth | Purpose |
|----------|--------|------|---------|
| `GET /api/central/stats` | GET | Central Admin | Platform statistics |
| `GET /api/central/tenants` | GET | Central Admin | List all tenants |
| `GET /api/central/tenants/:id` | GET | Central Admin | Tenant details |
| `GET /api/central/tenants/:id/stats` | GET | Central Admin | Tenant statistics |
| `PUT /api/central/tenants/:id/suspend` | PUT | Central Admin | Suspend tenant |
| `PUT /api/central/tenants/:id/activate` | PUT | Central Admin | Activate tenant |

### Files to Create

| File | Purpose |
|------|---------|
| `internal/handlers/central_admin_handler.go` | Central admin operations |
| `internal/middleware/central_admin.go` | Central admin authorization |
| `internal/static/central/index.html` | Dashboard |
| `internal/static/central/tenants.html` | Tenant management |
| `internal/database/008_add_central_admin.go` | Central admin role migration |

---

## Phase 11: Docker Infrastructure

**Objective:** Create production-ready Docker deployment.

### 11.1 Dockerfile

```dockerfile
# Dockerfile
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X github.com/tranmh/gassigeher/internal/version.Version=$(git describe --tags --always)" \
    -o gassigeher ./cmd/server

# Runtime image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary
COPY --from=builder /app/gassigeher .

# Copy embedded frontend (already in binary via go:embed)
# Copy landing page (not embedded)
COPY --from=builder /app/internal/static/landing ./internal/static/landing
COPY --from=builder /app/internal/static/central ./internal/static/central

# Create non-root user
RUN addgroup -g 1000 gassigeher && \
    adduser -u 1000 -G gassigeher -s /bin/sh -D gassigeher && \
    chown -R gassigeher:gassigeher /app

USER gassigeher

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

CMD ["./gassigeher"]
```

### 11.2 Docker Compose (Production)

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  app:
    image: gassigeher:latest
    build: .
    restart: unless-stopped
    environment:
      - DB_TYPE=postgres
      - DB_HOST=db
      - DB_PORT=5432
      - DB_NAME=${DB_NAME}
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - JWT_SECRET=${JWT_SECRET}
      - BASE_URL=https://gassigeher.org
      - BASE_DOMAIN=gassigeher.org
      # S3
      - USE_S3=true
      - S3_ENDPOINT=${S3_ENDPOINT}
      - S3_ACCESS_KEY=${S3_ACCESS_KEY}
      - S3_SECRET_KEY=${S3_SECRET_KEY}
      - S3_BUCKET_NAME=${S3_BUCKET_NAME}
      - S3_REGION=${S3_REGION}
      - S3_PUBLIC_URL=${S3_PUBLIC_URL}
      # SMTP
      - EMAIL_PROVIDER=smtp
      - SMTP_HOST=${SMTP_HOST}
      - SMTP_PORT=${SMTP_PORT}
      - SMTP_USERNAME=${SMTP_USERNAME}
      - SMTP_PASSWORD=${SMTP_PASSWORD}
      - SMTP_FROM_EMAIL=${SMTP_FROM_EMAIL}
      - SMTP_USE_SSL=true
    depends_on:
      db:
        condition: service_healthy
    networks:
      - internal
    labels:
      - "traefik.enable=false"

  db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      - POSTGRES_DB=${DB_NAME}
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER} -d ${DB_NAME}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - internal

  caddy:
    image: caddy:2-alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    environment:
      - HETZNER_DNS_TOKEN=${HETZNER_DNS_TOKEN}
    depends_on:
      - app
    networks:
      - internal
      - external

networks:
  internal:
    driver: bridge
  external:
    driver: bridge

volumes:
  postgres_data:
  caddy_data:
  caddy_config:
```

### 11.3 Caddyfile (Wildcard SSL)

```caddyfile
# Caddyfile

# Main domain - Landing page
gassigeher.org {
    tls {
        dns hetzner {env.HETZNER_DNS_TOKEN}
    }

    # Landing page routes
    handle /api/tenants/* {
        reverse_proxy app:8080
    }

    handle /landing/* {
        root * /srv/landing
        file_server
    }

    handle {
        root * /srv/landing
        try_files {path} /index.html
        file_server
    }

    header {
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
    }
}

# Central admin
admin.gassigeher.org {
    tls {
        dns hetzner {env.HETZNER_DNS_TOKEN}
    }

    reverse_proxy app:8080

    header {
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
    }
}

# Wildcard for all tenant subdomains
*.gassigeher.org {
    tls {
        dns hetzner {env.HETZNER_DNS_TOKEN}
    }

    reverse_proxy app:8080

    header {
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
    }

    # Rate limiting
    rate_limit {
        zone dynamic {
            key {remote_host}
            events 100
            window 1m
        }
    }
}
```

### 11.4 Environment Template

```env
# .env.production

# Database
DB_TYPE=postgres
DB_NAME=gassigeher
DB_USER=gassigeher
DB_PASSWORD=your-secure-password-here

# JWT
JWT_SECRET=your-256-bit-secret-here
JWT_EXPIRATION_HOURS=24

# Domain
BASE_URL=https://gassigeher.org
BASE_DOMAIN=gassigeher.org

# Hetzner S3
USE_S3=true
S3_ENDPOINT=fsn1.your-objectstorage.com
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
S3_BUCKET_NAME=gassigeher-uploads
S3_REGION=fsn1
S3_PUBLIC_URL=https://gassigeher-uploads.fsn1.your-objectstorage.com

# SMTP (Strato)
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.strato.de
SMTP_PORT=465
SMTP_USERNAME=noreply@gassigeher.org
SMTP_PASSWORD=your-email-password
SMTP_FROM_EMAIL=noreply@gassigeher.org
SMTP_USE_SSL=true

# Hetzner DNS (for Caddy wildcard SSL)
HETZNER_DNS_TOKEN=your-dns-api-token
```

### Files to Create

| File | Purpose |
|------|---------|
| `Dockerfile` | Multi-stage build |
| `docker-compose.yml` | Development stack |
| `docker-compose.prod.yml` | Production stack |
| `Caddyfile` | Reverse proxy with wildcard SSL |
| `.env.production.example` | Production environment template |
| `scripts/deploy.sh` | Deployment script |
| `scripts/backup.sh` | Backup script |

---

## Phase 12: Testing & Migration

**Objective:** Comprehensive testing and data migration for existing tenants.

### 12.1 Test Plan

```
tests/
├── unit/
│   ├── tenant_repository_test.go
│   ├── brute_force_service_test.go
│   ├── s3_service_test.go
│   └── provisioning_service_test.go
├── integration/
│   ├── tenant_isolation_test.go    # RLS verification
│   ├── registration_flow_test.go
│   └── multi_tenant_booking_test.go
├── e2e/
│   ├── registration_e2e_test.go
│   ├── booking_e2e_test.go
│   └── admin_e2e_test.go
└── security/
    ├── rls_bypass_test.go
    ├── rate_limit_test.go
    └── brute_force_test.go
```

### 12.2 Tenant Isolation Test

```go
// tests/integration/tenant_isolation_test.go

func TestTenantIsolation(t *testing.T) {
    // Setup: Create two tenants
    tenant1 := createTestTenant(t, "tenant1")
    tenant2 := createTestTenant(t, "tenant2")

    // Create a dog in tenant1
    dog1 := createTestDog(t, tenant1.ID, "Max")

    // Try to access dog1 from tenant2 context
    ctx := context.WithValue(context.Background(), middleware.TenantIDKey, tenant2.ID)

    dogs, err := dogRepo.FindAll(ctx, tenant2.ID, DogFilter{})
    assert.NoError(t, err)
    assert.Empty(t, dogs, "Tenant2 should not see Tenant1's dogs")

    // Verify tenant1 can see their dog
    ctx = context.WithValue(context.Background(), middleware.TenantIDKey, tenant1.ID)
    dogs, err = dogRepo.FindAll(ctx, tenant1.ID, DogFilter{})
    assert.NoError(t, err)
    assert.Len(t, dogs, 1)
    assert.Equal(t, "Max", dogs[0].Name)
}
```

### 12.3 Migration Script (Existing Single Tenant)

```go
// cmd/migrate-to-saas/main.go

package main

import (
    "database/sql"
    "flag"
    "log"
)

func main() {
    tenantName := flag.String("name", "", "Tenant name")
    tenantSlug := flag.String("slug", "", "Tenant slug")
    contactEmail := flag.String("email", "", "Contact email")
    flag.Parse()

    if *tenantName == "" || *tenantSlug == "" || *contactEmail == "" {
        log.Fatal("Missing required flags: -name, -slug, -email")
    }

    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    tx, err := db.Begin()
    if err != nil {
        log.Fatal(err)
    }
    defer tx.Rollback()

    // 1. Create tenant
    var tenantID int
    err = tx.QueryRow(`
        INSERT INTO tenants (slug, name, status, contact_email)
        VALUES ($1, $2, 'active', $3)
        RETURNING id
    `, *tenantSlug, *tenantName, *contactEmail).Scan(&tenantID)
    if err != nil {
        log.Fatal("Failed to create tenant:", err)
    }

    log.Printf("Created tenant ID: %d", tenantID)

    // 2. Update all existing records with tenant_id
    tables := []string{
        "users", "dogs", "bookings", "blocked_dates",
        "color_categories", "user_colors", "color_requests",
        "experience_requests", "reactivation_requests",
        "system_settings", "booking_time_rules", "custom_holidays",
        "walk_reports", "walk_report_photos",
    }

    for _, table := range tables {
        result, err := tx.Exec(fmt.Sprintf(
            "UPDATE %s SET tenant_id = $1 WHERE tenant_id IS NULL", table), tenantID)
        if err != nil {
            log.Fatalf("Failed to update %s: %v", table, err)
        }
        rows, _ := result.RowsAffected()
        log.Printf("Updated %d rows in %s", rows, table)
    }

    // 3. Create default tenant settings
    _, err = tx.Exec(`
        INSERT INTO tenant_settings (tenant_id, theme_preset)
        VALUES ($1, 'classic')
    `, tenantID)
    if err != nil {
        log.Fatal("Failed to create tenant settings:", err)
    }

    // 4. Commit
    if err := tx.Commit(); err != nil {
        log.Fatal("Failed to commit:", err)
    }

    log.Println("Migration completed successfully!")
    log.Printf("Tenant URL: https://%s.gassigeher.org", *tenantSlug)
}
```

### 12.4 Rollback Script

```go
// cmd/rollback-tenant/main.go

func main() {
    tenantID := flag.Int("id", 0, "Tenant ID to rollback")
    flag.Parse()

    // Remove tenant_id from all records
    // Delete tenant and settings
    // Restore to single-tenant mode
}
```

---

## File Change Summary

### New Files to Create (32 files)

| File | Phase | Purpose |
|------|-------|---------|
| `internal/database/003_add_tenants.go` | 1 | Tenants + settings tables |
| `internal/database/004_add_tenant_ids.go` | 1 | Add tenant_id columns |
| `internal/database/005_add_rls.go` | 1 | PostgreSQL RLS policies |
| `internal/database/006_update_constraints.go` | 1 | Update unique constraints |
| `internal/database/007_add_lockout_fields.go` | 5 | Account lockout fields |
| `internal/database/008_add_central_admin.go` | 10 | Central admin role |
| `internal/models/tenant.go` | 1 | Tenant models |
| `internal/models/theme.go` | 7 | Theme presets |
| `internal/repository/tenant_repository.go` | 1 | Tenant CRUD |
| `internal/repository/base_repository.go` | 3 | Base tenant-aware repo |
| `internal/middleware/tenant.go` | 2 | Subdomain resolution |
| `internal/middleware/ratelimit_global.go` | 5 | Global rate limiter |
| `internal/middleware/central_admin.go` | 10 | Central admin auth |
| `internal/services/brute_force_service.go` | 5 | Brute force protection |
| `internal/services/s3_service.go` | 6 | S3 operations |
| `internal/services/provisioning_service.go` | 8 | Tenant setup |
| `internal/handlers/tenant_handler.go` | 8 | Registration |
| `internal/handlers/theme_handler.go` | 7 | Dynamic CSS |
| `internal/handlers/central_admin_handler.go` | 10 | Central admin |
| `internal/static/landing/*.html` | 9 | Landing pages (6 files) |
| `internal/static/central/*.html` | 10 | Central admin pages (3 files) |
| `Dockerfile` | 11 | Docker build |
| `docker-compose.yml` | 11 | Development |
| `docker-compose.prod.yml` | 11 | Production |
| `Caddyfile` | 11 | Reverse proxy |
| `.env.production.example` | 11 | Production config |
| `cmd/migrate-to-saas/main.go` | 12 | Migration tool |
| `scripts/deploy.sh` | 11 | Deployment |
| `scripts/backup.sh` | 11 | Backup |

### Files to Modify (30+ files)

| File | Phases | Changes |
|------|--------|---------|
| `internal/config/config.go` | 1,6,11 | Add tenant, S3, domain config |
| `internal/middleware/middleware.go` | 2 | Add TenantIDKey, update AuthMiddleware |
| `internal/services/auth_service.go` | 2 | Add tenantID to GenerateJWT |
| `internal/handlers/auth_handler.go` | 2,5 | Integrate brute force, tenant |
| `internal/repository/user_repository.go` | 3 | Add tenant filtering |
| `internal/repository/dog_repository.go` | 3 | Add tenant filtering |
| `internal/repository/booking_repository.go` | 3 | Add tenant filtering |
| `internal/repository/blocked_date_repository.go` | 3 | Add tenant filtering |
| `internal/repository/color_category_repository.go` | 3 | Add tenant filtering |
| `internal/repository/color_request_repository.go` | 3 | Add tenant filtering |
| `internal/repository/user_color_repository.go` | 3 | Add tenant filtering |
| `internal/repository/experience_request_repository.go` | 3 | Add tenant filtering |
| `internal/repository/reactivation_request_repository.go` | 3 | Add tenant filtering |
| `internal/repository/settings_repository.go` | 3 | Add tenant filtering |
| `internal/repository/booking_time_repository.go` | 3 | Add tenant filtering |
| `internal/repository/holiday_repository.go` | 3 | Add tenant filtering |
| `internal/repository/walk_report_repository.go` | 3 | Add tenant filtering |
| `internal/handlers/user_handler.go` | 4 | Extract tenant from context |
| `internal/handlers/dog_handler.go` | 4,6 | Extract tenant, S3 upload |
| `internal/handlers/booking_handler.go` | 4 | Extract tenant from context |
| `internal/handlers/settings_handler.go` | 4,6 | Extract tenant, S3 logo |
| `internal/handlers/walk_report_handler.go` | 4,6 | Extract tenant, S3 photos |
| `internal/services/image_service.go` | 6 | Use S3 service |
| `internal/services/email_service.go` | 8 | Add tenant branding to emails |
| `cmd/server/main.go` | 2,5,7,8,10 | Add new middleware, routes, handlers |
| `frontend/*.html` | 7 | Add theme CSS link |

---

## Configuration Reference

### Environment Variables (Complete)

```env
# === Server ===
PORT=8080
BASE_URL=https://gassigeher.org
BASE_DOMAIN=gassigeher.org

# === Database ===
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=gassigeher
DB_USER=gassigeher
DB_PASSWORD=secure-password
DB_SSLMODE=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# === JWT ===
JWT_SECRET=your-256-bit-random-secret
JWT_EXPIRATION_HOURS=24

# === S3 Storage (Hetzner) ===
USE_S3=true
S3_ENDPOINT=fsn1.your-objectstorage.com
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
S3_BUCKET_NAME=gassigeher-uploads
S3_REGION=fsn1
S3_PUBLIC_URL=https://gassigeher-uploads.fsn1.your-objectstorage.com

# === Email (SMTP) ===
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.strato.de
SMTP_PORT=465
SMTP_USERNAME=noreply@gassigeher.org
SMTP_PASSWORD=your-email-password
SMTP_FROM_EMAIL=noreply@gassigeher.org
SMTP_FROM_NAME=Gassigeher
SMTP_USE_SSL=true

# === Security ===
RATE_LIMIT_RPS=10
RATE_LIMIT_BURST=20
BRUTE_FORCE_MAX_ATTEMPTS=3
BRUTE_FORCE_LOCKOUT_BASE=30
BRUTE_FORCE_LOCKOUT_MAX=1800

# === Logging ===
LOG_DIR=./logs
LOG_MAX_AGE_DAYS=30
LOG_CONSOLE_OUTPUT=true

# === DNS (for Caddy wildcard SSL) ===
HETZNER_DNS_TOKEN=your-dns-api-token
```

---

## Estimated Timeline

| Phase | Description | Duration |
|-------|-------------|----------|
| 1 | Database Foundation | 3-4 days |
| 2 | Tenant Middleware & JWT | 2-3 days |
| 3 | Repository Layer Updates | 4-5 days |
| 4 | Handler Layer Updates | 3-4 days |
| 5 | Enhanced Security | 2-3 days |
| 6 | Hetzner S3 Storage | 3-4 days |
| 7 | Theming System | 2-3 days |
| 8 | Tenant Registration | 3-4 days |
| 9 | Landing Page | 3-4 days |
| 10 | Central Admin Dashboard | 3-4 days |
| 11 | Docker Infrastructure | 2-3 days |
| 12 | Testing & Migration | 5-7 days |
| **Total** | | **36-48 days** |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Data loss during migration | Full backup before migration, rollback script ready |
| RLS bypass vulnerabilities | Security audit, penetration testing, double validation |
| Performance degradation | Load testing with 500 simulated tenants |
| Tenant isolation breach | JWT + RLS double validation, comprehensive tests |
| S3 dependency failure | Local storage fallback, retry logic with exponential backoff |
| SMTP deliverability | Use established provider (Strato), SPF/DKIM setup |

---

## Success Criteria

1. **Functional:** 2 test tenants can operate independently
2. **Security:** No cross-tenant data leakage in security tests
3. **Performance:** <200ms response time under load
4. **Registration:** New tenant provisioned in <30 seconds
5. **Reliability:** 99.9% uptime target

---

## Next Steps

1. **Review this plan** with stakeholders
2. **Purchase domain** gassigeher.org
3. **Set up Hetzner infrastructure** (server, S3, DNS)
4. **Begin Phase 1** implementation
5. **Test with 2 pilot Tierheime** before wider rollout

---

*Document created: December 2025*
*Last updated: December 2025*
