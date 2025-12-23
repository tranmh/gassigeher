# Missing SaaS Features for Going Live

> Technical roadmap for developers - organized by implementation phase

**Current State:** ~90% SaaS-ready
**Last Updated:** 2025-12-23

---

## What's Already Complete

Before diving into what's missing, here's what's already implemented:

| Feature | Status | Key Files |
|---------|--------|-----------|
| Multi-tenancy architecture | Complete | `internal/middleware/tenant.go`, `cmd/server/main.go:117` |
| Stripe billing (Free/Pro tiers) | Complete | `internal/services/stripe_service.go`, `internal/handlers/billing_handler.go` |
| 3-layer rate limiting | Complete | `internal/middleware/ratelimit_*.go` |
| JWT auth with RBAC | Complete | `internal/services/auth_service.go`, `internal/middleware/middleware.go` |
| Email infrastructure (18 templates) | Complete | `internal/services/email_service.go`, `internal/services/email_account.go` |
| Multi-database support | Complete | `internal/database/dialect*.go`, 23 migrations |
| S3 file storage | Complete | `internal/services/s3_service.go`, handlers connected |
| Central Admin UI | Complete | `internal/static/central/*.html` (3 pages) |
| Tenant provisioning | Complete | `internal/services/provisioning_service.go` |
| 305+ tests | Complete | `*_test.go` files throughout |

---

## Phase 1: Critical for Launch

### 1.1 User Data Export (GDPR Portability)

**What's Missing:**
Individual users cannot export their own data. Only central admins can export tenant-wide data via `ExportTenantData`.

**Why It Matters:**
GDPR Article 20 requires data portability - users must be able to download their personal data in a machine-readable format.

**Current State:**
- `ExportTenantData` exists at `internal/handlers/central_admin_handler.go:531`
- Central admin can export tenant data (users, dogs, booking count)
- No endpoint for individual users to export their own data

**Implementation Hints:**

```go
// internal/handlers/user_handler.go

// ExportMyData exports all personal data for the authenticated user
// GET /api/users/me/export
func (h *UserHandler) ExportMyData(w http.ResponseWriter, r *http.Request) {
    userID, _ := r.Context().Value(middleware.UserIDKey).(int)
    tenantID := middleware.GetTenantID(r)

    user, err := h.userRepo.FindByID(userID, tenantID)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Benutzerdaten")
        return
    }

    // Sanitize sensitive fields
    user.PasswordHash = nil
    user.VerificationToken = nil
    user.PasswordResetToken = nil

    export := map[string]interface{}{
        "user":        user,
        "exported_at": time.Now(),
    }

    // Get user's bookings
    bookings, _ := h.bookingRepo.FindByUserID(userID, tenantID)
    export["bookings"] = bookings

    // Get user's walk reports
    reports, _ := h.walkReportRepo.FindByUserID(userID)
    export["walk_reports"] = reports

    // Get user's experience requests
    requests, _ := h.experienceRepo.FindByUserID(userID)
    export["experience_requests"] = requests

    w.Header().Set("Content-Disposition", "attachment; filename=meine-daten.json")
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(export)
}
```

**Affected Files:**
- `internal/handlers/user_handler.go` - Add ExportMyData method
- `cmd/server/main.go` - Add route to protected router
- `frontend/js/api.js` - Add exportMyData() method
- `frontend/profile.html` - Add "Meine Daten exportieren" button

**Dependencies:** None

**Definition of Done:**
- [ ] User can click "Meine Daten exportieren" in profile
- [ ] JSON file downloads with all personal data
- [ ] Booking history included
- [ ] Walk reports included
- [ ] Sensitive fields (password, tokens) excluded
- [ ] German filename (meine-daten.json)

**Tests Required:**
```go
func TestUserHandler_ExportMyData(t *testing.T)
func TestUserHandler_ExportMyData_ExcludesSensitiveFields(t *testing.T)
func TestUserHandler_ExportMyData_IncludesBookings(t *testing.T)
func TestUserHandler_ExportMyData_TenantIsolation(t *testing.T)
```

---

### 1.2 Enhanced Health Check

**What's Missing:**
Current health check only returns `{"status": "ok"}`. No database connectivity check, no dependency status.

**Why It Matters:**
Kubernetes/load balancers need accurate health information. A service can return "ok" while the database is down.

**Current State:**
```go
// internal/handlers/health_handler.go - CURRENT (15 lines total)
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
    data := map[string]string{"status": "ok"}
    respondJSON(w, http.StatusOK, data)
}
```

**Implementation Hints:**

```go
// internal/handlers/health_handler.go

type HealthHandler struct {
    db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
    return &HealthHandler{db: db}
}

type HealthResponse struct {
    Status    string            `json:"status"`
    Checks    map[string]Check  `json:"checks"`
    Timestamp time.Time         `json:"timestamp"`
}

type Check struct {
    Status  string `json:"status"`
    Message string `json:"message,omitempty"`
    Latency string `json:"latency,omitempty"`
}

// Health returns liveness status (is the process running?)
// GET /api/health
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
    respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready returns readiness status (can we serve traffic?)
// GET /api/ready
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
    response := HealthResponse{
        Status:    "ok",
        Checks:    make(map[string]Check),
        Timestamp: time.Now(),
    }

    // Check database
    start := time.Now()
    err := h.db.Ping()
    latency := time.Since(start)

    if err != nil {
        response.Status = "degraded"
        response.Checks["database"] = Check{
            Status:  "fail",
            Message: err.Error(),
        }
    } else {
        response.Checks["database"] = Check{
            Status:  "ok",
            Latency: latency.String(),
        }
    }

    // Set appropriate status code
    statusCode := http.StatusOK
    if response.Status != "ok" {
        statusCode = http.StatusServiceUnavailable
    }

    respondJSON(w, statusCode, response)
}
```

**Affected Files:**
- `internal/handlers/health_handler.go` - Rewrite with DB check
- `cmd/server/main.go` - Pass db to NewHealthHandler, add /api/ready route

**Dependencies:** None

**Definition of Done:**
- [ ] `/api/health` returns liveness (always ok if process running)
- [ ] `/api/ready` checks database connectivity
- [ ] `/api/ready` returns 503 if database unreachable
- [ ] Response includes latency metrics
- [ ] Kubernetes/nginx can use /api/ready for health checks

**Tests Required:**
```go
func TestHealthHandler_Health_ReturnsOK(t *testing.T)
func TestHealthHandler_Ready_DatabaseUp(t *testing.T)
func TestHealthHandler_Ready_DatabaseDown(t *testing.T)
func TestHealthHandler_Ready_ReturnsLatency(t *testing.T)
```

---

### 1.3 MySQL/PostgreSQL Backup Scripts

**What's Missing:**
`deploy/backup.sh` only supports SQLite. No backup scripts for MySQL or PostgreSQL.

**Why It Matters:**
Production SaaS deployments typically use MySQL or PostgreSQL. Without backup scripts, data is at risk.

**Current State:**
```bash
# deploy/backup.sh - Line 22 (SQLite only)
if sqlite3 "$DB_PATH" ".backup '$BACKUP_FILE'"; then
```

**Implementation Hints:**

```bash
#!/bin/bash
# deploy/backup.sh - Multi-database backup script

set -e

# Load environment
source /var/gassigeher/.env

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/gassigeher/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
DATE=$(date +%Y%m%d_%H%M%S)
LOG_FILE="/var/gassigeher/logs/backup.log"

mkdir -p "$BACKUP_DIR"

log() {
    echo "[$(date)] $1" >> "$LOG_FILE"
}

backup_sqlite() {
    BACKUP_FILE="${BACKUP_DIR}/gassigeher_${DATE}.db"
    log "Starting SQLite backup..."
    sqlite3 "$DATABASE_PATH" ".backup '$BACKUP_FILE'"
    gzip "$BACKUP_FILE"
    log "SQLite backup completed: ${BACKUP_FILE}.gz"
}

backup_mysql() {
    BACKUP_FILE="${BACKUP_DIR}/gassigeher_${DATE}.sql"
    log "Starting MySQL backup..."
    mysqldump -h "$DB_HOST" -P "$DB_PORT" -u "$DB_USER" -p"$DB_PASSWORD" \
        --single-transaction --routines --triggers "$DB_NAME" > "$BACKUP_FILE"
    gzip "$BACKUP_FILE"
    log "MySQL backup completed: ${BACKUP_FILE}.gz"
}

backup_postgres() {
    BACKUP_FILE="${BACKUP_DIR}/gassigeher_${DATE}.sql"
    log "Starting PostgreSQL backup..."
    PGPASSWORD="$DB_PASSWORD" pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
        -Fc "$DB_NAME" > "${BACKUP_FILE%.sql}.dump"
    gzip "${BACKUP_FILE%.sql}.dump"
    log "PostgreSQL backup completed: ${BACKUP_FILE%.sql}.dump.gz"
}

# Determine database type and run appropriate backup
case "${DB_TYPE:-sqlite}" in
    sqlite)
        backup_sqlite
        ;;
    mysql)
        backup_mysql
        ;;
    postgres|postgresql)
        backup_postgres
        ;;
    *)
        log "ERROR: Unknown database type: $DB_TYPE"
        exit 1
        ;;
esac

# Cleanup old backups
find "$BACKUP_DIR" -name "gassigeher_*" -type f -mtime +$RETENTION_DAYS -delete
log "Cleanup completed. Retention: $RETENTION_DAYS days"

# Calculate and log backup size
SIZE=$(du -sh "$BACKUP_DIR" | cut -f1)
log "Total backup size: $SIZE"

exit 0
```

**Affected Files:**
- `deploy/backup.sh` - Rewrite with multi-database support
- `deploy/restore.sh` - Create new restore script

**Dependencies:**
- `mysqldump` for MySQL
- `pg_dump` for PostgreSQL

**Definition of Done:**
- [ ] backup.sh detects DB_TYPE from environment
- [ ] SQLite backup works (existing functionality)
- [ ] MySQL backup uses mysqldump with --single-transaction
- [ ] PostgreSQL backup uses pg_dump with custom format
- [ ] Backups are compressed with gzip
- [ ] Old backups cleaned up after RETENTION_DAYS
- [ ] restore.sh can restore from backup files

**Tests Required:**
```bash
# Manual testing with each database type
./backup.sh  # Test with DB_TYPE=sqlite
./backup.sh  # Test with DB_TYPE=mysql
./backup.sh  # Test with DB_TYPE=postgres
./restore.sh <backup_file>  # Test restore
```

---

## Phase 2: Business Audit Logging

### 2.1 Audit Log Service

**What's Missing:**
No dedicated audit logging for business events. Currently only HTTP request logging exists in `internal/logging/logger.go`.

**Why It Matters:**
Enterprise customers require audit trails. Security incidents need forensic data. Compliance (SOC2, GDPR) requires knowing who did what.

**Current State:**
- HTTP request logging: `internal/logging/logger.go`
- Occasional `log.Printf("AUDIT: ...")` statements scattered in handlers
- No structured audit table or service

**Implementation Hints:**

```go
// internal/models/audit_log.go

package models

import "time"

type AuditLog struct {
    ID         int       `json:"id"`
    TenantID   int       `json:"tenant_id"`
    UserID     *int      `json:"user_id,omitempty"`
    Action     string    `json:"action"`      // e.g., "booking.created", "user.promoted"
    EntityType string    `json:"entity_type"` // e.g., "booking", "user", "dog"
    EntityID   *int      `json:"entity_id,omitempty"`
    OldValue   *string   `json:"old_value,omitempty"`  // JSON
    NewValue   *string   `json:"new_value,omitempty"`  // JSON
    IPAddress  string    `json:"ip_address"`
    UserAgent  string    `json:"user_agent"`
    CreatedAt  time.Time `json:"created_at"`
}

// AuditAction constants
const (
    AuditActionBookingCreated   = "booking.created"
    AuditActionBookingCancelled = "booking.cancelled"
    AuditActionBookingApproved  = "booking.approved"
    AuditActionUserCreated      = "user.created"
    AuditActionUserDeleted      = "user.deleted"
    AuditActionUserPromoted     = "user.promoted"
    AuditActionUserDemoted      = "user.demoted"
    AuditActionDogCreated       = "dog.created"
    AuditActionDogUpdated       = "dog.updated"
    AuditActionSettingsChanged  = "settings.changed"
    AuditActionDataExported     = "data.exported"
)
```

```go
// internal/services/audit_service.go

package services

import (
    "database/sql"
    "encoding/json"
    "net/http"

    "github.com/tranmh/gassigeher/internal/models"
)

type AuditService struct {
    db *sql.DB
}

func NewAuditService(db *sql.DB) *AuditService {
    return &AuditService{db: db}
}

func (s *AuditService) Log(r *http.Request, tenantID int, userID *int, action, entityType string, entityID *int, oldValue, newValue interface{}) error {
    var oldJSON, newJSON *string

    if oldValue != nil {
        b, _ := json.Marshal(oldValue)
        str := string(b)
        oldJSON = &str
    }
    if newValue != nil {
        b, _ := json.Marshal(newValue)
        str := string(b)
        newJSON = &str
    }

    _, err := s.db.Exec(`
        INSERT INTO audit_logs (tenant_id, user_id, action, entity_type, entity_id,
                                old_value, new_value, ip_address, user_agent, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
        tenantID, userID, action, entityType, entityID,
        oldJSON, newJSON, getClientIP(r), r.UserAgent(),
    )
    return err
}

func getClientIP(r *http.Request) string {
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        return strings.Split(xff, ",")[0]
    }
    return strings.Split(r.RemoteAddr, ":")[0]
}
```

```sql
-- Migration: 024_create_audit_logs.go

CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id INTEGER NOT NULL,
    user_id INTEGER,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id INTEGER,
    old_value TEXT,
    new_value TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
```

**Affected Files:**
- `internal/models/audit_log.go` - New model
- `internal/services/audit_service.go` - New service
- `internal/repository/audit_repository.go` - New repository
- `internal/database/024_create_audit_logs.go` - New migration
- `internal/handlers/*.go` - Add audit calls to existing handlers
- `cmd/server/main.go` - Initialize audit service

**Dependencies:** None

**Definition of Done:**
- [ ] audit_logs table created via migration
- [ ] AuditService can log events with old/new values
- [ ] Booking creation logged
- [ ] Booking cancellation logged
- [ ] User promotion/demotion logged
- [ ] Settings changes logged
- [ ] Data exports logged
- [ ] Admin can view audit logs via API
- [ ] Audit logs are tenant-scoped

**Tests Required:**
```go
func TestAuditService_LogBookingCreated(t *testing.T)
func TestAuditService_LogUserPromoted(t *testing.T)
func TestAuditService_TenantIsolation(t *testing.T)
func TestAuditService_CapturesIPAddress(t *testing.T)
func TestAuditService_CapturesOldAndNewValues(t *testing.T)
```

---

## Phase 3: Observability

### 3.1 Prometheus Metrics Endpoint

**What's Missing:**
No `/metrics` endpoint for Prometheus scraping. No request count, latency histograms, or error rate metrics.

**Why It Matters:**
Production monitoring requires metrics. Alerting on error rates, latency P99, and request volume is essential for reliability.

**Current State:**
No metrics infrastructure. Only HTTP request logging.

**Implementation Hints:**

```go
// internal/middleware/metrics.go

package middleware

import (
    "net/http"
    "strconv"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )

    activeConnections = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "http_active_connections",
            Help: "Number of active HTTP connections",
        },
    )
)

func MetricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        activeConnections.Inc()
        defer activeConnections.Dec()

        // Wrap response writer to capture status code
        wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

        next.ServeHTTP(wrapped, r)

        duration := time.Since(start).Seconds()
        path := normalizePath(r.URL.Path)

        httpRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(wrapped.statusCode)).Inc()
        httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

// normalizePath reduces cardinality by replacing IDs with placeholders
func normalizePath(path string) string {
    // /api/dogs/123 -> /api/dogs/:id
    // /api/bookings/456/cancel -> /api/bookings/:id/cancel
    // Implementation left as exercise
    return path
}
```

```go
// cmd/server/main.go additions

import "github.com/prometheus/client_golang/prometheus/promhttp"

// In main():
router.Handle("/metrics", promhttp.Handler())
router.Use(middleware.MetricsMiddleware)
```

**Affected Files:**
- `internal/middleware/metrics.go` - New middleware
- `cmd/server/main.go` - Add /metrics endpoint and middleware
- `go.mod` - Add prometheus dependency

**Dependencies:**
- `github.com/prometheus/client_golang`

**Definition of Done:**
- [ ] `/metrics` returns Prometheus format
- [ ] http_requests_total counter with method, path, status labels
- [ ] http_request_duration_seconds histogram
- [ ] http_active_connections gauge
- [ ] Path cardinality managed (IDs normalized)
- [ ] Grafana dashboard template provided

**Tests Required:**
```go
func TestMetricsMiddleware_IncrementsCounter(t *testing.T)
func TestMetricsMiddleware_RecordsDuration(t *testing.T)
func TestMetricsMiddleware_NormalizesPath(t *testing.T)
func TestMetricsEndpoint_ReturnsPrometheusFormat(t *testing.T)
```

---

## Phase 4: Compliance & Legal

### 4.1 Consent Management

**What's Missing:**
No tracking of when users accepted Terms of Service or Privacy Policy. No forced consent flow.

**Why It Matters:**
GDPR requires demonstrable consent. If audited, you must prove users agreed to data processing.

**Current State:**
- Terms page exists: `frontend/terms.html`
- Privacy page exists: `frontend/privacy.html`
- No consent tracking in database
- No forced acceptance on registration

**Implementation Hints:**

```go
// internal/models/consent.go

package models

import "time"

type Consent struct {
    ID          int       `json:"id"`
    UserID      int       `json:"user_id"`
    TenantID    int       `json:"tenant_id"`
    ConsentType string    `json:"consent_type"` // "terms", "privacy", "marketing"
    Version     string    `json:"version"`      // e.g., "2025-01-01"
    IPAddress   string    `json:"ip_address"`
    UserAgent   string    `json:"user_agent"`
    AcceptedAt  time.Time `json:"accepted_at"`
}
```

```go
// Modify internal/handlers/auth_handler.go Register()

type RegisterRequest struct {
    FirstName       string `json:"first_name"`
    LastName        string `json:"last_name"`
    Email           string `json:"email"`
    Password        string `json:"password"`
    TermsAccepted   bool   `json:"terms_accepted"`
    PrivacyAccepted bool   `json:"privacy_accepted"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req RegisterRequest
    // ... decode request ...

    if !req.TermsAccepted || !req.PrivacyAccepted {
        respondError(w, http.StatusBadRequest,
            "Sie müssen die Nutzungsbedingungen und Datenschutzerklärung akzeptieren")
        return
    }

    // ... create user ...

    // Record consent
    h.consentRepo.Create(&models.Consent{
        UserID:      user.ID,
        TenantID:    tenantID,
        ConsentType: "terms",
        Version:     "2025-01-01",
        IPAddress:   getClientIP(r),
        UserAgent:   r.UserAgent(),
        AcceptedAt:  time.Now(),
    })
    h.consentRepo.Create(&models.Consent{
        UserID:      user.ID,
        TenantID:    tenantID,
        ConsentType: "privacy",
        Version:     "2025-01-01",
        IPAddress:   getClientIP(r),
        UserAgent:   r.UserAgent(),
        AcceptedAt:  time.Now(),
    })
}
```

**Affected Files:**
- `internal/models/consent.go` - New model
- `internal/repository/consent_repository.go` - New repository
- `internal/database/025_create_consents.go` - New migration
- `internal/handlers/auth_handler.go` - Modify Register to require consent
- `frontend/register.html` - Add consent checkboxes
- `frontend/i18n/de.json` - Add translations

**Dependencies:** None

**Definition of Done:**
- [ ] consents table created via migration
- [ ] Registration requires checking both boxes
- [ ] Consent recorded with timestamp, IP, user agent
- [ ] Version tracked for when policies change
- [ ] Admin can view user's consent history
- [ ] User can view their own consent history

**Tests Required:**
```go
func TestAuthHandler_Register_RequiresConsent(t *testing.T)
func TestAuthHandler_Register_RecordsConsent(t *testing.T)
func TestConsentRepository_Create(t *testing.T)
func TestConsentRepository_FindByUserID(t *testing.T)
```

---

## Phase 5: Platform Maturity

### 5.1 API Versioning

**What's Missing:**
All endpoints at `/api/*` with no version prefix. Breaking changes affect all clients immediately.

**Why It Matters:**
Mobile apps, third-party integrations need stable APIs. Version prefix allows deprecation periods.

**Current State:**
All routes registered directly at `/api/*`:
```go
// cmd/server/main.go
protected := router.PathPrefix("/api").Subrouter()
```

**Implementation Hints:**

```go
// cmd/server/main.go

// Create versioned subrouters
apiV1 := router.PathPrefix("/api/v1").Subrouter()
apiV1.Use(middleware.AuthMiddleware(cfg.JWTSecret))

// Register all routes under /api/v1/
apiV1.HandleFunc("/users/me", userHandler.GetMe).Methods("GET")
apiV1.HandleFunc("/dogs", dogHandler.ListDogs).Methods("GET")
// ... all other routes ...

// Legacy redirect: /api/* -> /api/v1/*
router.PathPrefix("/api/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Rewrite path from /api/xxx to /api/v1/xxx
    newPath := strings.Replace(r.URL.Path, "/api/", "/api/v1/", 1)
    http.Redirect(w, r, newPath+r.URL.RawQuery, http.StatusTemporaryRedirect)
})
```

**Affected Files:**
- `cmd/server/main.go` - Change all `/api/` to `/api/v1/`
- `frontend/js/api.js` - Update BASE_URL to `/api/v1`

**Dependencies:** None

**Definition of Done:**
- [ ] All routes under `/api/v1/`
- [ ] Old `/api/*` redirects to `/api/v1/*` (307 Temporary Redirect)
- [ ] Frontend uses `/api/v1/` directly
- [ ] API version header: `X-API-Version: v1`
- [ ] Documentation updated

**Tests Required:**
```go
func TestAPIVersioning_V1Prefix(t *testing.T)
func TestAPIVersioning_LegacyRedirect(t *testing.T)
func TestAPIVersioning_VersionHeader(t *testing.T)
```

---

### 5.2 Feature Flags

**What's Missing:**
No feature flag system. All features enabled/disabled via environment variables or code.

**Why It Matters:**
Gradual rollout reduces risk. A/B testing requires feature flags. Per-tenant feature enablement for premium features.

**Current State:**
No feature flag infrastructure.

**Implementation Hints:**

```go
// internal/models/feature_flag.go

package models

type FeatureFlag struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`       // e.g., "walk_reports", "dark_mode"
    Description string  `json:"description"`
    Enabled     bool    `json:"enabled"`    // Global default
    TenantID    *int    `json:"tenant_id"`  // NULL = global, set = tenant-specific
    CreatedAt   string  `json:"created_at"`
    UpdatedAt   string  `json:"updated_at"`
}
```

```go
// internal/services/feature_flag_service.go

package services

type FeatureFlagService struct {
    db    *sql.DB
    cache map[string]map[int]bool // name -> tenantID -> enabled
    mu    sync.RWMutex
}

func (s *FeatureFlagService) IsEnabled(name string, tenantID int) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // Check tenant-specific override first
    if tenantFlags, ok := s.cache[name]; ok {
        if enabled, exists := tenantFlags[tenantID]; exists {
            return enabled
        }
        // Fall back to global (tenantID = 0)
        if enabled, exists := tenantFlags[0]; exists {
            return enabled
        }
    }
    return false
}

// Usage in handler:
if featureFlags.IsEnabled("walk_reports", tenantID) {
    // Show walk reports feature
}
```

**Affected Files:**
- `internal/models/feature_flag.go` - New model
- `internal/services/feature_flag_service.go` - New service
- `internal/repository/feature_flag_repository.go` - New repository
- `internal/database/026_create_feature_flags.go` - New migration
- `internal/handlers/feature_flag_handler.go` - Admin management endpoints

**Dependencies:** None

**Definition of Done:**
- [ ] feature_flags table created
- [ ] Global flags apply to all tenants
- [ ] Tenant-specific overrides possible
- [ ] In-memory cache with refresh
- [ ] Admin can toggle flags via API
- [ ] Frontend can check flags via API

**Tests Required:**
```go
func TestFeatureFlagService_IsEnabled_GlobalFlag(t *testing.T)
func TestFeatureFlagService_IsEnabled_TenantOverride(t *testing.T)
func TestFeatureFlagService_CacheRefresh(t *testing.T)
func TestFeatureFlagHandler_ToggleFlag(t *testing.T)
```

---

### 5.3 Caching Layer

**What's Missing:**
No general-purpose caching. All queries hit the database. Only holiday API results are cached.

**Why It Matters:**
Database load increases linearly with traffic. Caching reduces latency and database cost.

**Current State:**
- Holiday caching exists: `internal/repository/holiday_repository.go` uses `feiertage_cache` table
- No in-memory cache
- No Redis integration

**Implementation Hints:**

```go
// internal/services/cache_service.go

package services

import (
    "sync"
    "time"
)

type CacheItem struct {
    Value      interface{}
    ExpiresAt  time.Time
}

type CacheService struct {
    items map[string]CacheItem
    mu    sync.RWMutex
}

func NewCacheService() *CacheService {
    c := &CacheService{
        items: make(map[string]CacheItem),
    }
    go c.cleanupLoop()
    return c
}

func (c *CacheService) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    item, exists := c.items[key]
    if !exists || time.Now().After(item.ExpiresAt) {
        return nil, false
    }
    return item.Value, true
}

func (c *CacheService) Set(key string, value interface{}, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.items[key] = CacheItem{
        Value:     value,
        ExpiresAt: time.Now().Add(ttl),
    }
}

func (c *CacheService) Delete(key string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.items, key)
}

func (c *CacheService) InvalidatePrefix(prefix string) {
    c.mu.Lock()
    defer c.mu.Unlock()

    for key := range c.items {
        if strings.HasPrefix(key, prefix) {
            delete(c.items, key)
        }
    }
}

func (c *CacheService) cleanupLoop() {
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        c.mu.Lock()
        now := time.Now()
        for key, item := range c.items {
            if now.After(item.ExpiresAt) {
                delete(c.items, key)
            }
        }
        c.mu.Unlock()
    }
}
```

```go
// Usage in dog_handler.go:

func (h *DogHandler) ListDogs(w http.ResponseWriter, r *http.Request) {
    tenantID := middleware.GetTenantID(r)
    cacheKey := fmt.Sprintf("dogs:tenant:%d", tenantID)

    if cached, ok := h.cache.Get(cacheKey); ok {
        respondJSON(w, http.StatusOK, cached)
        return
    }

    dogs, err := h.dogRepo.FindAll(tenantID)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Hunde")
        return
    }

    h.cache.Set(cacheKey, dogs, 5*time.Minute)
    respondJSON(w, http.StatusOK, dogs)
}

// Invalidate on write:
func (h *DogHandler) CreateDog(w http.ResponseWriter, r *http.Request) {
    // ... create dog ...
    h.cache.InvalidatePrefix(fmt.Sprintf("dogs:tenant:%d", tenantID))
}
```

**Affected Files:**
- `internal/services/cache_service.go` - New service
- `internal/handlers/*.go` - Add caching to read endpoints
- `cmd/server/main.go` - Initialize cache service

**Dependencies:** None (in-memory first, Redis later)

**Definition of Done:**
- [ ] In-memory cache with TTL
- [ ] Automatic cleanup of expired entries
- [ ] Cache invalidation on writes
- [ ] Dog list cached (5 min TTL)
- [ ] Tenant settings cached (10 min TTL)
- [ ] Cache hit/miss metrics (optional)

**Tests Required:**
```go
func TestCacheService_SetGet(t *testing.T)
func TestCacheService_TTLExpiration(t *testing.T)
func TestCacheService_Delete(t *testing.T)
func TestCacheService_InvalidatePrefix(t *testing.T)
func TestDogHandler_ListDogs_UsesCacheOnHit(t *testing.T)
func TestDogHandler_CreateDog_InvalidatesCache(t *testing.T)
```

---

## Appendix: Files Reference

### New Files to Create

| File | Description |
|------|-------------|
| `internal/models/audit_log.go` | Audit log model |
| `internal/models/consent.go` | Consent tracking model |
| `internal/models/feature_flag.go` | Feature flag model |
| `internal/services/audit_service.go` | Audit logging service |
| `internal/services/cache_service.go` | In-memory cache service |
| `internal/services/feature_flag_service.go` | Feature flag service |
| `internal/repository/audit_repository.go` | Audit log repository |
| `internal/repository/consent_repository.go` | Consent repository |
| `internal/repository/feature_flag_repository.go` | Feature flag repository |
| `internal/middleware/metrics.go` | Prometheus metrics middleware |
| `internal/database/024_create_audit_logs.go` | Audit logs migration |
| `internal/database/025_create_consents.go` | Consents migration |
| `internal/database/026_create_feature_flags.go` | Feature flags migration |
| `deploy/restore.sh` | Database restore script |

### Files to Modify

| File | Changes |
|------|---------|
| `internal/handlers/health_handler.go` | Add /ready endpoint with DB check |
| `internal/handlers/user_handler.go` | Add ExportMyData method |
| `internal/handlers/auth_handler.go` | Add consent requirement to Register |
| `cmd/server/main.go` | Add metrics, versioning, pass db to health handler |
| `deploy/backup.sh` | Multi-database support |
| `frontend/js/api.js` | Update to /api/v1/, add exportMyData |
| `frontend/register.html` | Add consent checkboxes |
| `frontend/profile.html` | Add data export button |
| `go.mod` | Add prometheus dependency |

---

## Implementation Priority

| Priority | Feature | Effort | Impact |
|----------|---------|--------|--------|
| P0 | User Data Export | Low | High (GDPR compliance) |
| P0 | Enhanced Health Check | Low | High (operations) |
| P0 | MySQL/PostgreSQL Backup | Medium | High (data safety) |
| P1 | Audit Logging | Medium | High (compliance) |
| P1 | Prometheus Metrics | Medium | High (observability) |
| P1 | Consent Management | Medium | High (GDPR compliance) |
| P2 | API Versioning | Low | Medium (future-proofing) |
| P2 | Feature Flags | Medium | Medium (flexibility) |
| P2 | Caching Layer | Medium | Medium (performance) |

---

## Summary

**Total Missing Features:** 9
**Critical (P0):** 3
**Important (P1):** 3
**Nice-to-Have (P2):** 3

The codebase is approximately 90% SaaS-ready. The remaining features focus on:
1. **Compliance** - GDPR data export, consent tracking, audit logging
2. **Operations** - Health checks, metrics, backup scripts
3. **Platform** - API versioning, feature flags, caching

Most features are additive (new files) rather than modifications to existing code, reducing risk.
