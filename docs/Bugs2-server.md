# Bug Report: cmd/server

**Analysis Date:** 2025-12-27
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/cmd/server`
**Files Analyzed:** 2 files (main.go, routes_test.go)
**Bugs Found:** 8 bugs

---

## Summary

The cmd/server directory is the main entry point for the application, responsible for initialization, routing setup, middleware configuration, and graceful shutdown. Analysis revealed **8 functional bugs** spanning critical initialization errors, configuration vulnerabilities, race conditions, and routing inconsistencies. Most critical issues include:

- **Missing Stripe service nil check** causing potential panic (Critical)
- **Insecure JWT secret default** in production environments (Critical)
- **Race condition** in cron service shutdown sequence (High)
- **Inconsistent tenant validation** across public routes (High)
- **Missing error handling** for handler initialization failures (Medium)

---

## Bugs

## Bug #1: Stripe Service Not Validated Before Use

**Severity:** Critical

**Description:**
The `stripeService` is initialized conditionally (only when `cfg.StripeSecretKey != ""`), but the `billingHandler` is always created with this potentially `nil` service. If Stripe is not configured but a user attempts to use billing endpoints, the handler will panic when trying to access `stripeService` methods.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/cmd/server/main.go`
- Lines: 221-236

**Steps to Reproduce:**
1. Start server without `STRIPE_SECRET_KEY` environment variable
2. Authenticate as a tenant admin
3. Make a request to `POST /api/v1/billing/checkout`
4. Expected: Error message stating Stripe is not configured
5. Actual: Server panic due to nil pointer dereference

**Code Analysis:**
```go
// Lines 221-236
var stripeService *services.StripeService
if cfg.StripeSecretKey != "" {
    stripeService = services.NewStripeService(...)
    log.Println("Stripe payment service initialized")
}
billingHandler := handlers.NewBillingHandler(db, cfg, stripeService)
```

The `billingHandler` receives a `nil` stripeService, but the handler doesn't defensively check for nil before calling methods like `stripeService.IsConfigured()`, `stripeService.CreateCheckoutSession()`, etc.

**Impact:**
- Server crash on any billing endpoint access when Stripe is not configured
- No graceful degradation for missing Stripe configuration
- Poor user experience with cryptic "500 Internal Server Error" instead of clear message

**Fix:**
Add validation in BillingHandler methods to check if Stripe service is configured:

```diff
  // In internal/handlers/billing_handler.go - CreateCheckout method
  func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
+     if h.stripeService == nil || !h.stripeService.IsConfigured() {
+         respondError(w, http.StatusServiceUnavailable, "Stripe payment service not configured")
+         return
+     }
      // ... existing code
  }
```

Apply this check to all billing handler methods that use `stripeService`. Alternatively, return an HTTP 503 (Service Unavailable) for all billing routes if Stripe is not configured:

```diff
  // In cmd/server/main.go
  var stripeService *services.StripeService
  if cfg.StripeSecretKey != "" {
      stripeService = services.NewStripeService(...)
      log.Println("Stripe payment service initialized")
+ } else {
+     log.Println("Warning: Stripe not configured - billing endpoints will return 503")
  }
  billingHandler := handlers.NewBillingHandler(db, cfg, stripeService)
```

---

## Bug #2: Insecure JWT Secret Default in Production

**Severity:** Critical

**Description:**
The JWT secret uses a default value "change-this-in-production-INSECURE" if the `JWT_SECRET` environment variable is not set. While the default includes "INSECURE" as a warning, the application still starts and uses this predictable secret, allowing attackers to forge valid JWT tokens and gain unauthorized access to any account.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Function: `Load()`
- Line: 137

**Steps to Reproduce:**
1. Start server without setting `JWT_SECRET` environment variable
2. Server starts successfully with default "change-this-in-production-INSECURE"
3. Attacker can generate valid JWT tokens using this known secret
4. Expected: Server refuses to start without secure JWT_SECRET
5. Actual: Server runs with insecure default, exposing all user accounts

**Code Analysis:**
```go
// Line 137 in config.go
JWTSecret: getEnvRequired("JWT_SECRET", "change-this-in-production-INSECURE"),
```

The `getEnvRequired` function returns the insecure default without failing:
```go
func getEnvRequired(key, insecureDefault string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return insecureDefault
}
```

**Impact:**
- Complete authentication bypass - attacker can forge tokens for any user
- No protection against unauthorized access in production
- Silent failure - no warning logs when using insecure default
- Violates security best practices for production deployments

**Fix:**
Fail server startup if JWT_SECRET is not explicitly set in production:

```diff
  // In cmd/server/main.go, after cfg := config.Load()
  cfg := config.Load()
+
+ // SECURITY: Refuse to start with insecure JWT secret
+ if strings.Contains(cfg.JWTSecret, "INSECURE") || cfg.JWTSecret == "change-this-in-production-INSECURE" {
+     log.Fatal("SECURITY ERROR: JWT_SECRET environment variable must be set to a secure random value. Generate one with: openssl rand -base64 32")
+ }
```

Or make the check environment-aware (only fail in production):
```diff
  cfg := config.Load()
+
+ // Check if running in production (non-local, non-test environments)
+ isProduction := !strings.HasSuffix(cfg.BaseDomain, ".local") &&
+                 !strings.HasSuffix(cfg.BaseDomain, ".localhost") &&
+                 os.Getenv("ENVIRONMENT") == "production"
+
+ if isProduction && (strings.Contains(cfg.JWTSecret, "INSECURE") || len(cfg.JWTSecret) < 32) {
+     log.Fatal("SECURITY ERROR: Production environments must set JWT_SECRET to a secure random value (min 32 chars). Generate: openssl rand -base64 32")
+ }
```

---

## Bug #3: Race Condition in Shutdown Sequence

**Severity:** High

**Description:**
The graceful shutdown sequence stops the cron service **before** waiting for the HTTP server to finish processing requests. If a cron job (auto-completion, reminders, auto-deactivation) is running when shutdown is triggered, and an HTTP handler is simultaneously accessing the database, this creates a race condition. The HTTP server continues serving requests after cron cleanup has started, potentially causing database connection issues or incomplete operations.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/cmd/server/main.go`
- Lines: 686-703

**Steps to Reproduce:**
1. Start server and let cron jobs run (auto-completion runs every 15 minutes)
2. Send a request to a database-heavy endpoint (e.g., `GET /api/v1/bookings`)
3. While request is processing, send SIGTERM signal to server
4. Cron service stops immediately (line 692)
5. HTTP server shutdown begins but request is still in-flight
6. Expected: HTTP requests complete before any cleanup
7. Actual: Cron cleanup happens while HTTP requests are still processing

**Code Analysis:**
```go
// Lines 686-703
select {
case err := <-serverErr:
    log.Printf("Server failed to start: %v", err)
case sig := <-quit:
    log.Printf("Received signal %v, initiating graceful shutdown...", sig)
}

// Stop cron service first (before HTTP server shutdown)
cronService.Stop()  // ← BUG: Stops cron BEFORE HTTP server
log.Println("Cron service stopped")

// Create context with timeout for shutdown
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Shutdown server gracefully
if err := srv.Shutdown(ctx); err != nil {  // ← HTTP server stops AFTER cron
    log.Printf("Server forced to shutdown: %v", err)
}
```

**Impact:**
- Database connection issues during shutdown
- Incomplete HTTP requests if cron jobs are cleaning up shared resources
- Potential data corruption if cron jobs modify data while HTTP handlers read it
- Violates graceful shutdown best practices

**Fix:**
Reverse the shutdown order - stop HTTP server first, then cron service:

```diff
  select {
  case err := <-serverErr:
      log.Printf("Server failed to start: %v", err)
  case sig := <-quit:
      log.Printf("Received signal %v, initiating graceful shutdown...", sig)
  }

- // Stop cron service first (before HTTP server shutdown)
- cronService.Stop()
- log.Println("Cron service stopped")
-
  // Create context with timeout for shutdown
  ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
  defer cancel()

  // Shutdown server gracefully
  if err := srv.Shutdown(ctx); err != nil {
      log.Printf("Server forced to shutdown: %v", err)
  }
+ log.Println("HTTP server stopped")
+
+ // Stop cron service after HTTP server (no more in-flight requests)
+ cronService.Stop()
+ log.Println("Cron service stopped")

  log.Println("Server stopped gracefully")
```

**Rationale:**
The HTTP server's `Shutdown()` method waits for all in-flight requests to complete (up to the context timeout). Only after all requests finish should background services like cron be stopped. This ensures no request is interrupted by resource cleanup.

---

## Bug #4: Missing Tenant Validation on Public Routes

**Severity:** High

**Description:**
Several public API routes (booking times, featured dogs, color categories, settings, theme CSS, tenant branding) are accessible without tenant validation in SaaS mode. While `TenantMiddleware` runs globally, these routes don't check if a tenant was successfully resolved. This allows requests to the base domain (no subdomain) to access these endpoints, potentially returning data from tenant_id=0 or causing database errors.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/cmd/server/main.go`
- Lines: 302-333 (public routes)

**Steps to Reproduce (SaaS Mode):**
1. Start server in SaaS mode with `BASE_DOMAIN=gassigeher.org`
2. Make request to `https://gassigeher.org/api/v1/dogs/featured` (no subdomain)
3. TenantMiddleware passes through (no tenant, no error)
4. Handler executes with `tenant_id = 0` from context
5. Database query: `SELECT * FROM dogs WHERE tenant_id = 0 AND is_featured = 1`
6. Expected: 400 Bad Request "Kein Tierheim ausgewählt"
7. Actual: Query executes, returns empty results or data for tenant_id=0 if exists

**Code Analysis:**
```go
// Lines 302-333 - Public routes without tenant validation
router.HandleFunc("/api/v1/booking-times/available", bookingTimeHandler.GetAvailableSlots).Methods("GET")
router.HandleFunc("/api/v1/dogs/featured", dogHandler.GetFeaturedDogs).Methods("GET")
router.HandleFunc("/api/v1/colors", colorCategoryHandler.ListColors).Methods("GET")
router.HandleFunc("/api/v1/settings/logo", settingsHandler.GetLogo).Methods("GET")
router.HandleFunc("/api/v1/theme/css", themeHandler.GetCSS).Methods("GET")
router.HandleFunc("/api/v1/tenant/branding", tenantHandler.GetBranding).Methods("GET")
```

The `TenantMiddleware` (line 179) only returns errors for **invalid** tenants (suspended, not found), but allows requests with **no tenant** (tenant_id=0) to pass through:

```go
// TenantMiddleware behavior
if slug == "" || slug == "www" || slug == "admin" {
    next.ServeHTTP(w, r)  // ← Passes through with tenant_id=0
    return
}
```

**Impact:**
- Data leakage if tenant_id=0 accidentally contains data
- Confusing empty responses for legitimate requests
- Inconsistent API behavior (some routes require tenant, others don't)
- Security risk if repositories don't properly filter by tenant_id

**Fix:**
Add `RequireTenant` middleware to public routes that depend on tenant context:

```diff
  // Booking time routes (public - for time slot availability)
+ tenantOnlyRoutes := router.PathPrefix("/api/v1").Subrouter()
+ if cfg.BaseDomain != "" { // Only in SaaS mode
+     tenantOnlyRoutes.Use(middleware.RequireTenant)
+ }
- router.HandleFunc("/api/v1/booking-times/available", bookingTimeHandler.GetAvailableSlots).Methods("GET")
+ tenantOnlyRoutes.HandleFunc("/booking-times/available", bookingTimeHandler.GetAvailableSlots).Methods("GET")
- router.HandleFunc("/api/v1/booking-times/rules-for-date", bookingTimeHandler.GetRulesForDate).Methods("GET")
+ tenantOnlyRoutes.HandleFunc("/booking-times/rules-for-date", bookingTimeHandler.GetRulesForDate).Methods("GET")
- router.HandleFunc("/api/v1/dogs/featured", dogHandler.GetFeaturedDogs).Methods("GET")
+ tenantOnlyRoutes.HandleFunc("/dogs/featured", dogHandler.GetFeaturedDogs).Methods("GET")
- router.HandleFunc("/api/v1/colors", colorCategoryHandler.ListColors).Methods("GET")
+ tenantOnlyRoutes.HandleFunc("/colors", colorCategoryHandler.ListColors).Methods("GET")
- router.HandleFunc("/api/v1/settings/logo", settingsHandler.GetLogo).Methods("GET")
+ tenantOnlyRoutes.HandleFunc("/settings/logo", settingsHandler.GetLogo).Methods("GET")
- router.HandleFunc("/api/v1/theme/css", themeHandler.GetCSS).Methods("GET")
+ tenantOnlyRoutes.HandleFunc("/theme/css", themeHandler.GetCSS).Methods("GET")
- router.HandleFunc("/api/v1/tenant/branding", tenantHandler.GetBranding).Methods("GET")
+ tenantOnlyRoutes.HandleFunc("/tenant/branding", tenantHandler.GetBranding).Methods("GET")
```

**Note:** Routes that should work without a tenant (e.g., tenant registration, contact form, FOMO widget) should remain on the main router.

---

## Bug #5: Handler Initialization Failures Not Logged

**Severity:** Medium

**Description:**
Multiple handlers are initialized in sequence (lines 190-214), but if any handler's `New*Handler()` function fails during initialization (e.g., email service initialization fails, repository initialization fails, service dependencies missing), the error is either silently ignored or causes a panic. The server continues running with potentially broken handlers, leading to runtime failures when those endpoints are accessed.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/cmd/server/main.go`
- Lines: 190-214

**Steps to Reproduce:**
1. Corrupt the email configuration (e.g., invalid SMTP host)
2. Start server
3. Handlers initialize - email service fails in `NewAuthHandler` (line 190)
4. Failure is logged as "Warning: Failed to initialize email service"
5. Server starts successfully
6. User registers → verification email fails silently
7. Expected: Clear startup warning listing which handlers have degraded functionality
8. Actual: Server appears healthy but silently fails at runtime

**Code Analysis:**
```go
// Lines 190-214 - No error checking on handler initialization
authHandler := handlers.NewAuthHandler(db, cfg)
userHandler := handlers.NewUserHandler(db, cfg)
dogHandler := handlers.NewDogHandler(db, cfg)
bookingHandler := handlers.NewBookingHandler(db, cfg)
// ... 15+ more handlers, no error checking
```

Each handler's `New*Handler()` function may encounter initialization errors:
- Email service fails (SMTP/Gmail API misconfigured)
- S3 service fails (invalid credentials)
- Stripe service fails (invalid API key)
- Repository initialization fails (database connection issues)

**Impact:**
- Silent failures at runtime instead of startup failures
- Hard to debug issues (no clear startup warnings)
- Inconsistent behavior (some features work, others fail mysteriously)
- Poor operational visibility

**Fix:**
Add initialization health checks and log warnings for degraded functionality:

```diff
  // Initialize handlers
  authHandler := handlers.NewAuthHandler(db, cfg)
+ if authHandler.EmailService == nil {
+     log.Println("⚠️  WARNING: Auth handler email service unavailable - verification emails disabled")
+ }
  userHandler := handlers.NewUserHandler(db, cfg)
+ if userHandler.EmailService == nil {
+     log.Println("⚠️  WARNING: User handler email service unavailable - account emails disabled")
+ }
  // ... similar checks for other handlers
```

Or implement a health check aggregator:

```go
// After all handlers initialized (line 218)
type HandlerHealthCheck struct {
    Handler string
    Healthy bool
    Reason  string
}

healthChecks := []HandlerHealthCheck{}
if authHandler.EmailService == nil {
    healthChecks = append(healthChecks, HandlerHealthCheck{
        Handler: "AuthHandler",
        Healthy: false,
        Reason:  "Email service unavailable",
    })
}
// ... check all handlers

if len(healthChecks) > 0 {
    log.Println("⚠️  Server starting with degraded functionality:")
    for _, check := range healthChecks {
        log.Printf("   - %s: %s", check.Handler, check.Reason)
    }
}
```

---

## Bug #6: Middleware Order Vulnerability

**Severity:** Medium

**Description:**
The `TenantMiddleware` runs **after** `CORSMiddleware` and `SecurityHeadersMiddleware` (lines 173, 179), but the CORS middleware uses `cfg.BaseURL` to set allowed origins. In SaaS mode, tenants have different subdomains (e.g., `tierheim-goeppingen.gassigeher.org`), but the CORS allowed origins only include the base URL. This means cross-origin requests from tenant subdomains to the API will be blocked by the browser.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/cmd/server/main.go`
- Lines: 168-186

**Steps to Reproduce (SaaS Mode):**
1. Start server with `BASE_URL=https://gassigeher.org` and `BASE_DOMAIN=gassigeher.org`
2. Frontend served at `https://tierheim-goeppingen.gassigeher.org/`
3. Frontend JavaScript makes API call to `/api/v1/dogs`
4. Browser sends preflight OPTIONS request with Origin header
5. CORSMiddleware checks if `Origin: https://tierheim-goeppingen.gassigeher.org` matches allowed origins
6. Allowed origins = `[https://gassigeher.org]` (from cfg.BaseURL)
7. Expected: CORS headers set to allow subdomain
8. Actual: No CORS headers set, browser blocks the request

**Code Analysis:**
```go
// Lines 168-186
router.Use(middleware.GlobalRateLimit(100, 200))
router.Use(middleware.MetricsMiddleware)
router.Use(middleware.LoggingMiddleware)
router.Use(middleware.BlockDangerousMethods)
router.Use(middleware.SecurityHeadersMiddleware)
router.Use(middleware.CORSMiddleware(cfg.BaseURL))  // ← Line 173: Uses cfg.BaseURL

// ... later
tenantRepo := repository.NewTenantRepository(db)
if cfg.BaseDomain != "" {
    router.Use(middleware.TenantMiddleware(tenantRepo, cfg.BaseDomain))  // ← Line 179: Tenant resolved here
    // ... but tenant info not available to CORSMiddleware above
}
```

The `CORSMiddleware` is initialized with `cfg.BaseURL` (e.g., `https://gassigeher.org`), which doesn't include tenant subdomains.

**Impact:**
- CORS errors on all API calls from tenant subdomains
- Frontend JavaScript cannot communicate with backend API
- Complete application failure in SaaS mode for tenant users
- Only works for the base domain (landing page), not tenant applications

**Fix:**
Update `CORSMiddleware` to dynamically allow origins based on tenant subdomain:

```diff
  router.Use(middleware.SecurityHeadersMiddleware)
- router.Use(middleware.CORSMiddleware(cfg.BaseURL))
+ // CORS middleware must run AFTER tenant middleware in SaaS mode
+ // to allow tenant subdomains

  tenantRepo := repository.NewTenantRepository(db)
  if cfg.BaseDomain != "" {
      router.Use(middleware.TenantMiddleware(tenantRepo, cfg.BaseDomain))
+     // Dynamic CORS for SaaS mode (allows all subdomains of BaseDomain)
+     router.Use(middleware.DynamicCORSMiddleware(cfg.BaseDomain))
  } else {
+     // Static CORS for Simple mode
+     router.Use(middleware.CORSMiddleware(cfg.BaseURL))
  }
```

Then implement `DynamicCORSMiddleware` in `internal/middleware/middleware.go`:

```go
// DynamicCORSMiddleware allows CORS for tenant subdomains in SaaS mode
func DynamicCORSMiddleware(baseDomain string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")

            // Check if origin is a valid subdomain of baseDomain
            if origin != "" && isValidTenantOrigin(origin, baseDomain) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
                w.Header().Set("Access-Control-Allow-Credentials", "true")
            }

            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

func isValidTenantOrigin(origin, baseDomain string) bool {
    // Parse origin URL
    u, err := url.Parse(origin)
    if err != nil {
        return false
    }

    host := u.Hostname()

    // Allow base domain itself (landing page)
    if host == baseDomain {
        return true
    }

    // Allow valid subdomains: <slug>.baseDomain
    if strings.HasSuffix(host, "."+baseDomain) {
        subdomain := strings.TrimSuffix(host, "."+baseDomain)
        // Validate subdomain format (no further dots)
        return !strings.Contains(subdomain, ".")
    }

    return false
}
```

---

## Bug #7: Unprotected Uploads Directory in Simple Mode

**Severity:** Medium

**Description:**
The uploads directory is served via `http.FileServer` without authentication checks (line 589). In Simple Mode (single-tenant), this means any user (including unauthenticated users) can access uploaded files by guessing filenames. This exposes user profile photos and dog photos to unauthorized viewing, potentially violating GDPR privacy requirements.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/cmd/server/main.go`
- Line: 589

**Steps to Reproduce:**
1. Start server in Simple Mode (no `BASE_DOMAIN` set)
2. Admin uploads dog photo via admin panel
3. Photo stored as `/uploads/dogs/dog_1_full.jpg`
4. Unauthenticated user makes GET request to `http://localhost:8080/uploads/dogs/dog_1_full.jpg`
5. Expected: 401 Unauthorized (photos should require authentication)
6. Actual: 200 OK - Photo is served to anyone

**Code Analysis:**
```go
// Line 589
router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", handlers.SafeFileServer(http.Dir("./uploads"))))
```

No authentication middleware is applied to the `/uploads/` route. The `SafeFileServer` only validates file paths (prevents directory traversal), but doesn't check user authentication.

**Impact:**
- Privacy violation - user photos accessible without authentication
- GDPR compliance risk - personal data (profile photos) exposed
- Dog photos leaked to competitors or unauthorized users
- No access control on sensitive files

**Fix:**
Wrap the uploads handler with authentication middleware:

```diff
- // Uploads directory (user photos, dog photos) - must remain on filesystem
- // BUG FIX #4: Use SafeFileServer to prevent null byte injection and path traversal
- router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", handlers.SafeFileServer(http.Dir("./uploads"))))
+ // Uploads directory (user photos, dog photos) - protected by authentication
+ // BUG FIX #4: Use SafeFileServer to prevent null byte injection and path traversal
+ uploadsRouter := router.PathPrefix("/uploads/").Subrouter()
+ uploadsRouter.Use(middleware.AuthMiddleware(cfg.JWTSecret))
+ uploadsRouter.PathPrefix("/").Handler(http.StripPrefix("/uploads/", handlers.SafeFileServer(http.Dir("./uploads"))))
```

**Alternative Fix (Less Restrictive):**
Allow public access to dog photos but protect user photos:

```diff
  // Dog photos - public access (for marketing, featured dogs on homepage)
  router.PathPrefix("/uploads/dogs/").Handler(http.StripPrefix("/uploads/dogs/", handlers.SafeFileServer(http.Dir("./uploads/dogs"))))

+ // User photos - authenticated access only
+ userUploadsRouter := router.PathPrefix("/uploads/users/").Subrouter()
+ userUploadsRouter.Use(middleware.AuthMiddleware(cfg.JWTSecret))
+ userUploadsRouter.PathPrefix("/").Handler(http.StripPrefix("/uploads/users/", handlers.SafeFileServer(http.Dir("./uploads/users"))))
```

**Considerations:**
- If dog photos should be public (for marketing), only protect user uploads
- If all uploads should be private, apply auth to entire `/uploads/` path
- Consider signed URLs with expiration for temporary public access (advanced solution)

---

## Bug #8: Missing Rate Limiting on Critical Public Endpoints

**Severity:** Medium

**Description:**
While auth endpoints have specific rate limiting (register: 3/min, login: 5/min, forgot-password: 3/min), other critical public endpoints lack rate limiting. Specifically, the tenant registration endpoint (`POST /api/v1/tenants/register`) and contact form (`POST /api/v1/contact`) have no rate limits, allowing abuse via spam attacks, database flooding, or email bombing.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/cmd/server/main.go`
- Lines: 326, 335

**Steps to Reproduce:**
1. Write script to repeatedly call `POST /api/v1/tenants/register` with random data
2. No rate limiting prevents rapid requests
3. Each request:
   - Creates database entries (tenants table)
   - Sends provisioning emails
   - Consumes database connections
   - Generates audit logs
4. Expected: Rate limit prevents abuse (e.g., max 3 registrations per 10 minutes per IP)
5. Actual: Unlimited registrations possible, server resources exhausted

**Code Analysis:**
```go
// Line 326 - No rate limiting
router.HandleFunc("/api/v1/tenants/register", tenantHandler.Register).Methods("POST")

// Line 335 - No rate limiting
router.HandleFunc("/api/v1/contact", contactHandler.Submit).Methods("POST")

// Compare to auth endpoints with rate limiting (lines 277-296):
registerRoute := router.PathPrefix("/api/v1/auth/register").Subrouter()
registerRoute.Use(middleware.RateLimitAuthEndpoint)  // ← Has rate limiting
registerRoute.HandleFunc("", authHandler.Register).Methods("POST")
```

**Impact:**
- Spam attacks: Thousands of fake tenant registrations
- Email bombing: Contact form abused to send spam emails
- Database pollution: Fake entries consume storage
- Resource exhaustion: Excessive database connections, email sends
- Cost implications: Email provider costs for spam emails

**Fix:**
Add rate limiting to tenant registration and contact form:

```diff
  // Tenant registration (public - for self-service signup)
- router.HandleFunc("/api/v1/tenants/register", tenantHandler.Register).Methods("POST")
+ tenantRegisterRoute := router.PathPrefix("/api/v1/tenants/register").Subrouter()
+ tenantRegisterRoute.Use(middleware.RateLimitAuthEndpoint) // 3 req/min per IP
+ tenantRegisterRoute.HandleFunc("", tenantHandler.Register).Methods("POST")

  // Contact form (public - for landing page inquiries)
- router.HandleFunc("/api/v1/contact", contactHandler.Submit).Methods("POST")
+ contactFormRoute := router.PathPrefix("/api/v1/contact").Subrouter()
+ contactFormRoute.Use(middleware.RateLimitAuthEndpoint) // 3 req/min per IP
+ contactFormRoute.HandleFunc("", contactHandler.Submit).Methods("POST")
```

**Alternative:** Create a dedicated rate limiter for public form submissions:

```go
// In internal/middleware/ratelimit.go
func RateLimitPublicForms(next http.Handler) http.Handler {
    limiter := NewIPRateLimiter(rate.Limit(5.0/60.0), 3) // 5 requests per minute, burst 3
    return limiter.Limit(next)
}
```

Then apply:
```diff
+ tenantRegisterRoute.Use(middleware.RateLimitPublicForms)
+ contactFormRoute.Use(middleware.RateLimitPublicForms)
```

---

## Statistics

- **Critical:** 2 bugs (Stripe nil check, JWT secret default)
- **High:** 2 bugs (Race condition in shutdown, tenant validation)
- **Medium:** 4 bugs (Handler init failures, CORS middleware order, unprotected uploads, rate limiting gaps)
- **Low:** 0 bugs

---

## Recommendations

### Immediate Actions (Critical/High)

1. **Add Stripe service validation** in all BillingHandler methods before use
2. **Enforce JWT secret validation** at startup - refuse to run with insecure defaults in production
3. **Fix shutdown sequence** - stop HTTP server before cron service to avoid race conditions
4. **Add tenant validation** to public routes that require tenant context in SaaS mode

### Short-Term Improvements (Medium)

5. **Implement handler health checks** - log warnings at startup for degraded functionality
6. **Fix CORS middleware** - implement dynamic CORS for SaaS mode to allow tenant subdomains
7. **Protect uploads directory** - add authentication to sensitive file routes
8. **Add rate limiting** to tenant registration and contact form endpoints

### Long-Term Enhancements

9. **Structured logging** - Replace `log.Printf` with structured logging (e.g., `logrus`, `zap`) for better observability
10. **Health check endpoint** - Expand `/api/health` to include handler status (email service, Stripe, S3)
11. **Configuration validation** - Implement comprehensive startup validation for all required environment variables
12. **Dependency injection** - Refactor handler initialization to use dependency injection framework for better error handling
13. **Integration tests** - Add tests for initialization order, middleware chain, and shutdown sequence

### Security Hardening

14. **Secrets management** - Consider using external secrets manager (e.g., Vault, AWS Secrets Manager) instead of environment variables
15. **API versioning** - Document API versioning strategy and deprecation policy for future breaking changes
16. **Audit logging** - Add audit logs for configuration changes, handler initialization failures, and shutdown events
17. **Monitoring integration** - Add metrics for handler initialization success/failure, middleware execution time, and resource usage

---

## Testing Notes

The following test scenarios should be added to `cmd/server/main_test.go`:

1. **Initialization tests:**
   - Test server startup with missing required configuration
   - Test handler initialization with broken dependencies
   - Test middleware chain order and execution

2. **Shutdown tests:**
   - Test graceful shutdown with in-flight HTTP requests
   - Test graceful shutdown with running cron jobs
   - Test shutdown timeout handling

3. **Route registration tests:**
   - Test all public routes respond without authentication
   - Test all protected routes reject unauthenticated requests
   - Test all admin routes reject non-admin users
   - Test all super-admin routes reject non-super-admin users

4. **SaaS mode tests:**
   - Test tenant resolution from subdomain
   - Test CORS headers for tenant subdomains
   - Test public routes require tenant context

5. **Rate limiting tests:**
   - Test rate limits on auth endpoints
   - Test rate limits on public form endpoints
   - Test global rate limiting

Current test file (`routes_test.go`) only tests HTML page routing - expand to cover the bugs identified above.
