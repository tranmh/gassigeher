# Bug Report: middleware

**Analysis Date:** 2025-12-22
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/middleware`
**Files Analyzed:** 5 files (middleware.go, tenant.go, ratelimit.go, ratelimit_global.go, middleware_test.go)
**Bugs Found:** 9 bugs

---

## Summary

The middleware directory contains critical security vulnerabilities and logic errors that could lead to authentication bypass, tenant isolation violations, and denial of service attacks. The most severe issues include:

- **Critical**: CORS origin bypass vulnerability allowing unauthorized cross-origin requests
- **Critical**: Missing tenant validation for JWT claims enabling cross-tenant data access
- **Critical**: Race condition in rate limiter allowing bypass of login protection
- **High**: Missing database error handling in tenant middleware causing information disclosure
- **High**: Memory leak in global rate limiter goroutine cleanup
- **High**: Super admin privilege escalation via JWT caching
- **Medium**: Inconsistent tenant context precedence logic
- **Medium**: IP spoofing vulnerability in global rate limiter
- **Medium**: Missing subdomain validation allowing injection attacks

These vulnerabilities could allow attackers to bypass multi-tenant isolation, perform brute force attacks, and access data from other tenants.

---

## Bugs

## Bug #1: CORS Origin Bypass via Empty Origin Header

**Description:**
The CORS middleware (lines 103-114 in `middleware.go`) has a logic flaw that allows unauthorized cross-origin requests when no `Origin` header is present. When `origin == ""`, the middleware sets `Access-Control-Allow-Origin` to the `baseURL`, which may not match the actual request origin. This bypasses the CORS security check and allows any origin to make credentialed requests by omitting the `Origin` header.

**Impact:** Attackers can bypass CORS restrictions and make authenticated requests from malicious websites, potentially leading to CSRF attacks and unauthorized data access.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/middleware/middleware.go`
- Function: `CORSMiddleware`
- Lines: 103-114

**Steps to Reproduce:**
1. Make an authenticated API request without setting the `Origin` header
2. Observe that the response includes `Access-Control-Allow-Origin: <baseURL>`
3. The request succeeds even though the origin wasn't validated
4. Expected: Request should fail or not set CORS headers for same-origin requests
5. Actual: CORS headers are set incorrectly, potentially allowing unauthorized access

**Fix:**
Remove the fallback logic that sets CORS headers for requests without an `Origin` header. Same-origin requests don't need CORS headers at all:

```diff
		origin := r.Header.Get("Origin")
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

-		// If no origin header or not in allowed list, allow same-origin requests
-		if origin == "" {
-			w.Header().Set("Access-Control-Allow-Origin", baseURL)
-		}
+		// Don't set CORS headers for same-origin requests (no Origin header)
+		// Browsers only send Origin header for cross-origin requests
```

This prevents the incorrect CORS header from being set when no `Origin` header is present.

---

## Bug #2: Tenant ID Mismatch Validation Allows Zero Value Bypass

**Description:**
The tenant validation logic in `AuthMiddleware` (lines 206-210) checks if `subdomainTenantID != jwtTenantID` but allows the case where both are 0 (zero values). This means that if a JWT has `tenant_id: 0` and no subdomain is set, the validation passes. An attacker could create a JWT with `tenant_id: 0` and access resources without proper tenant isolation.

**Impact:** Authentication bypass allowing access to resources without tenant validation. This could enable cross-tenant data access or access to platform-level resources that should be restricted.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/middleware/middleware.go`
- Function: `AuthMiddleware`
- Lines: 206-210

**Steps to Reproduce:**
1. Create a valid JWT with `tenant_id: 0`
2. Make a request to a tenant-specific endpoint without a subdomain
3. The validation at line 207 passes because both values are 0
4. Expected: Request should fail due to missing tenant
5. Actual: Request bypasses tenant validation

**Fix:**
Add explicit validation to reject zero tenant IDs when tenant validation is required:

```diff
		// SaaS: Validate JWT tenant_id matches subdomain tenant (if subdomain tenant is set)
		subdomainTenantID, _ := r.Context().Value(TenantIDKey).(int)
-		if subdomainTenantID != 0 && jwtTenantID != 0 && subdomainTenantID != jwtTenantID {
+		if subdomainTenantID != 0 && jwtTenantID == 0 {
+			http.Error(w, `{"error":"Token ohne Tierheim-ID ungültig"}`, http.StatusUnauthorized)
+			return
+		}
+		if subdomainTenantID != 0 && jwtTenantID != 0 && subdomainTenantID != jwtTenantID {
			http.Error(w, `{"error":"Token für anderes Tierheim ungültig"}`, http.StatusUnauthorized)
			return
		}
```

This ensures that when a subdomain tenant is set, the JWT must have a valid (non-zero) tenant ID that matches.

---

## Bug #3: Race Condition in Login Rate Limiter Map Access

**Description:**
The `RateLimitLogin` middleware (lines 77-109 in `ratelimit.go`) locks the entire rate limiter for the duration of the request processing via `defer loginLimiter.mu.Unlock()` at line 80. This means the mutex is held during the execution of `next.ServeHTTP(w, r)` at line 107, which could take seconds if the handler is slow. This causes all concurrent login requests to be serialized, creating a denial of service vulnerability where a slow login handler blocks all other login attempts.

Additionally, the cleanup logic (lines 88-96) runs on every request, which is inefficient and could cause performance degradation under load.

**Impact:** Denial of service where legitimate users cannot log in because the rate limiter mutex is held during slow handler execution. Performance degradation under concurrent load.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/middleware/ratelimit.go`
- Function: `RateLimitLogin`
- Lines: 77-109

**Steps to Reproduce:**
1. Send 5 concurrent login requests from different IPs
2. Make one login handler take 5 seconds (simulate slow database)
3. All other login requests are blocked waiting for the mutex
4. Expected: Each IP should be rate limited independently without blocking others
5. Actual: All requests are serialized due to mutex held during handler execution

**Fix:**
Release the mutex before calling the next handler and ensure atomic operations:

```diff
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginLimiter.mu.Lock()
-		defer loginLimiter.mu.Unlock()

		// Get client IP safely (prevents IP spoofing)
		ip := getClientIP(r, loginLimiter.trustedProxies)

		now := time.Now()

		// Clean old requests outside window
		if requests, exists := loginLimiter.requests[ip]; exists {
			validRequests := []time.Time{}
			for _, reqTime := range requests {
				if now.Sub(reqTime) < loginLimiter.window {
					validRequests = append(validRequests, reqTime)
				}
			}
			loginLimiter.requests[ip] = validRequests
		}

		// Check if limit exceeded
		if len(loginLimiter.requests[ip]) >= loginLimiter.limit {
+			loginLimiter.mu.Unlock()
			http.Error(w, `{"error":"Zu viele Anmeldeversuche. Bitte versuchen Sie es in einer Minute erneut."}`, http.StatusTooManyRequests)
			return
		}

		// Add current request
		loginLimiter.requests[ip] = append(loginLimiter.requests[ip], now)
+		loginLimiter.mu.Unlock()

		next.ServeHTTP(w, r)
	})
```

This ensures the mutex is only held during the rate limit check, not during the actual request processing.

---

## Bug #4: Missing Database Error Logging in Tenant Middleware

**Description:**
The `TenantMiddleware` function (lines 29-33 in `tenant.go`) returns a generic "Interner Serverfehler" (Internal Server Error) when `tenantRepo.FindBySlug()` fails, but doesn't log the actual error. This makes debugging production issues extremely difficult and could hide critical database connectivity problems. Additionally, the error message leaks internal system information to attackers.

**Impact:** Production issues become difficult to diagnose. Attackers can use timing analysis to determine if database queries are failing. Legitimate database errors (connection issues, timeouts) are silently converted to generic errors.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/middleware/tenant.go`
- Function: `TenantMiddleware`
- Lines: 29-33

**Steps to Reproduce:**
1. Cause a database connection error (e.g., disconnect database)
2. Make a request to a tenant subdomain
3. Observe that only a generic error is returned
4. Expected: Error should be logged for monitoring
5. Actual: No error logging occurs, making diagnosis impossible

**Fix:**
Add proper error logging and differentiate between database errors and not-found errors:

```diff
		// Lookup tenant by slug
		tenant, err := tenantRepo.FindBySlug(slug)
		if err != nil {
+			log.Printf("ERROR: Failed to lookup tenant '%s': %v", slug, err)
			http.Error(w, `{"error":"Interner Serverfehler"}`, http.StatusInternalServerError)
			return
		}

		if tenant == nil {
+			log.Printf("WARNING: Tenant not found for slug: %s", slug)
			http.Error(w, `{"error":"Tierheim nicht gefunden"}`, http.StatusNotFound)
			return
		}
```

This provides visibility into database errors while maintaining security by not exposing internal details to clients.

---

## Bug #5: Memory Leak in Global Rate Limiter Cleanup Goroutine

**Description:**
The `NewGlobalRateLimiter` function (lines 24-36 in `ratelimit_global.go`) starts a cleanup goroutine via `go grl.cleanupStaleEntries()` but never provides a way to stop it. If the rate limiter is recreated multiple times (e.g., during tests or hot reloads), each instance creates a new goroutine that runs forever. The goroutine only stops when the ticker is garbage collected, but the goroutine itself prevents garbage collection of the `GlobalRateLimiter` instance.

**Impact:** Memory leak where each rate limiter instance keeps a goroutine alive indefinitely. In long-running servers with configuration reloads, this accumulates goroutines and memory, eventually degrading performance or causing crashes.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/middleware/ratelimit_global.go`
- Function: `NewGlobalRateLimiter`
- Lines: 24-36

**Steps to Reproduce:**
1. Create multiple `GlobalRateLimiter` instances
2. Monitor goroutine count with `runtime.NumGoroutine()`
3. Observe that goroutines never decrease
4. Expected: Old rate limiters should be garbage collected
5. Actual: Goroutines leak indefinitely

**Fix:**
Add a context-based cancellation mechanism:

```diff
+import "context"

type GlobalRateLimiter struct {
	limiters map[string]*rateLimiterEntry
	mu       sync.RWMutex
	rps      rate.Limit
	burst    int
+	ctx      context.Context
+	cancel   context.CancelFunc
}

func NewGlobalRateLimiter(rps float64, burst int) *GlobalRateLimiter {
+	ctx, cancel := context.WithCancel(context.Background())
	grl := &GlobalRateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rps:      rate.Limit(rps),
		burst:    burst,
+		ctx:      ctx,
+		cancel:   cancel,
	}

	// Start cleanup goroutine to remove stale entries
	go grl.cleanupStaleEntries()

	return grl
}

+// Close stops the cleanup goroutine and releases resources
+func (g *GlobalRateLimiter) Close() {
+	g.cancel()
+}

func (g *GlobalRateLimiter) cleanupStaleEntries() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
+		select {
+		case <-g.ctx.Done():
+			return
+		case <-ticker.C:
-	for range ticker.C {
			g.mu.Lock()
			cutoff := time.Now().Add(-1 * time.Hour)
			for ip, entry := range g.limiters {
				if entry.lastSeen.Before(cutoff) {
					delete(g.limiters, ip)
				}
			}
			g.mu.Unlock()
+		}
	}
}
```

Callers should defer `grl.Close()` to ensure cleanup when the rate limiter is no longer needed.

---

## Bug #6: Inconsistent Tenant Context Precedence Logic

**Description:**
The `AuthMiddleware` (lines 220-225) sets tenant context with precedence logic: "prefer subdomain, fallback to JWT". However, this is inconsistent with the validation logic at lines 206-210 which requires a match when both are set. If `subdomainTenantID` is 0 but `jwtTenantID` is non-zero, the JWT tenant is used. This creates a scenario where a user with JWT tenant 1 could access tenant 2's subdomain endpoints if tenant 2 doesn't have tenant middleware applied.

**Impact:** Tenant isolation breach where users can access resources from other tenants by manipulating subdomain access and relying on JWT tenant ID.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/middleware/middleware.go`
- Function: `AuthMiddleware`
- Lines: 220-225

**Steps to Reproduce:**
1. Obtain a valid JWT with `tenant_id: 1`
2. Access an endpoint on tenant 2's subdomain that has `AuthMiddleware` but not `TenantMiddleware`
3. The request succeeds with `tenant_id: 1` in context
4. Expected: Request should only access tenant 2's data
5. Actual: Request can access tenant 1's data on tenant 2's subdomain

**Fix:**
Make subdomain tenant mandatory when present and don't allow JWT-only tenant context on tenant subdomains:

```diff
		// SaaS: Add tenant_id to context (prefer subdomain, fallback to JWT)
		if subdomainTenantID != 0 {
			ctx = context.WithValue(ctx, TenantIDKey, subdomainTenantID)
-		} else if jwtTenantID != 0 {
-			ctx = context.WithValue(ctx, TenantIDKey, jwtTenantID)
		}
+		// If no subdomain tenant but JWT has tenant, only allow on non-tenant routes
+		// Tenant routes must use TenantMiddleware to set subdomain tenant
```

This ensures that tenant subdomains always enforce the subdomain's tenant ID, not the JWT's tenant ID.

---

## Bug #7: Missing Validation for Super Admin Privilege Escalation

**Description:**
The `RequireSuperAdmin` middleware (lines 246-255) only checks if the `isSuperAdmin` flag in the JWT is true. However, there's no validation that this flag matches the actual database state. If an attacker obtains a JWT for a user and then that user's super admin privileges are revoked, the JWT remains valid until expiration (24 hours by default). This allows a former super admin to continue accessing super admin endpoints.

**Impact:** Privilege escalation where revoked super admins can continue accessing administrative functions for up to 24 hours after privilege revocation.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/middleware/middleware.go`
- Function: `RequireSuperAdmin`
- Lines: 246-255

**Steps to Reproduce:**
1. User logs in as super admin and receives JWT
2. Administrator revokes user's super admin privileges in database
3. User continues using old JWT to access super admin endpoints
4. Expected: Access should be denied after privilege revocation
5. Actual: Access continues until JWT expires (up to 24 hours)

**Fix:**
Add database validation for critical privilege checks:

```diff
+import "database/sql"

-func RequireSuperAdmin(next http.Handler) http.Handler {
+func RequireSuperAdmin(db *sql.DB) func(http.Handler) http.Handler {
+	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isSuperAdmin, ok := r.Context().Value(IsSuperAdminKey).(bool)
			if !ok || !isSuperAdmin {
				http.Error(w, `{"error":"Super Admin access required"}`, http.StatusForbidden)
				return
			}
+
+			// Verify super admin status in database for critical operations
+			userID, _ := r.Context().Value(UserIDKey).(int)
+			var isStillSuperAdmin bool
+			err := db.QueryRow("SELECT is_super_admin FROM users WHERE id = ?", userID).Scan(&isStillSuperAdmin)
+			if err != nil || !isStillSuperAdmin {
+				http.Error(w, `{"error":"Super Admin privileges have been revoked"}`, http.StatusForbidden)
+				return
+			}
+
			next.ServeHTTP(w, r)
		})
+	}
}
```

This adds a database check to validate that super admin privileges haven't been revoked since JWT issuance.

---

## Bug #8: IP Spoofing Vulnerability in Global Rate Limiter

**Description:**
The `GlobalRateLimit` middleware (line 81 in `ratelimit_global.go`) calls `getClientIP(r, nil)` with `nil` as the trusted proxies parameter. This means it will NEVER trust the `X-Forwarded-For` header, even when behind a legitimate reverse proxy. However, the function signature suggests it should accept trusted proxies. This creates an inconsistency where the global rate limiter uses only `RemoteAddr`, which will be the proxy's IP in production, causing all users to share the same rate limit.

**Impact:** In production behind a reverse proxy (nginx, load balancer), all users appear to have the same IP (the proxy's IP), causing legitimate users to be rate limited when they shouldn't be. This is a denial of service vulnerability.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/middleware/ratelimit_global.go`
- Function: `GlobalRateLimit`
- Lines: 74-94

**Steps to Reproduce:**
1. Deploy application behind nginx reverse proxy
2. Have multiple users make concurrent requests
3. All requests come from proxy IP (e.g., 127.0.0.1)
4. Expected: Each user rate limited independently
5. Actual: All users share the same rate limit, blocking legitimate traffic

**Fix:**
Accept trusted proxies as a parameter and pass them to `getClientIP`:

```diff
-func GlobalRateLimit(rps float64, burst int) func(http.Handler) http.Handler {
+func GlobalRateLimit(rps float64, burst int, trustedProxies []string) func(http.Handler) http.Handler {
	limiter := NewGlobalRateLimiter(rps, burst)
+
+	// Convert slice to map for efficient lookup
+	trustedProxyMap := make(map[string]bool)
+	for _, proxy := range trustedProxies {
+		trustedProxyMap[proxy] = true
+	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
-			ip := getClientIP(r, nil)
+			ip := getClientIP(r, trustedProxyMap)

			if !limiter.GetLimiter(ip).Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Zu viele Anfragen. Bitte warten Sie einen Moment."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

This allows proper IP extraction when behind reverse proxies in production.

---

## Bug #9: Missing Subdomain Validation in extractSubdomain Function

**Description:**
The `extractSubdomain` function (lines 63-102 in `tenant.go`) has incomplete validation for subdomain format. While it checks for dots at line 97 to reject multi-level subdomains, it doesn't validate other invalid characters or patterns. For example:
- Empty subdomain after trimming whitespace
- Subdomains with special characters (e.g., "tenant@123")
- Subdomains starting or ending with hyphens (invalid DNS)
- Subdomains longer than 63 characters (DNS limit)

**Impact:** Invalid subdomains could cause database lookup failures, expose internal errors, or be used for injection attacks if subdomain is used in other contexts (logging, analytics, etc.).

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/middleware/tenant.go`
- Function: `extractSubdomain`
- Lines: 63-102

**Steps to Reproduce:**
1. Make request to host: `-invalid-.example.com`
2. Subdomain `-invalid-` is extracted and passed to database
3. Expected: Subdomain should be rejected as invalid
4. Actual: Invalid subdomain passes validation

**Fix:**
Add comprehensive subdomain validation:

```diff
+import "regexp"
+
+// DNS-compliant subdomain regex: alphanumeric + hyphens, cannot start/end with hyphen
+var subdomainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func extractSubdomain(host, baseDomain string) string {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Handle localhost for development
	if host == "localhost" || host == "127.0.0.1" {
		return ""
	}

	// Check if baseDomain is set
	if baseDomain == "" {
		return ""
	}

	// Remove port from baseDomain if present
	if idx := strings.Index(baseDomain, ":"); idx != -1 {
		baseDomain = baseDomain[:idx]
	}

	// Check if host ends with baseDomain
	if !strings.HasSuffix(host, "."+baseDomain) {
		if host == baseDomain {
			return ""
		}
		return ""
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(host, "."+baseDomain)
+	subdomain = strings.ToLower(strings.TrimSpace(subdomain))

	// Validate subdomain (no dots allowed - only first-level subdomains)
	if strings.Contains(subdomain, ".") {
		return ""
	}
+
+	// Validate DNS-compliant format (alphanumeric + hyphens, max 63 chars)
+	if !subdomainRegex.MatchString(subdomain) {
+		return ""
+	}

	return subdomain
}
```

This ensures only valid DNS-compliant subdomains are processed, preventing injection and error cases.

---

## Statistics

- **Critical:** 3 bugs (CORS bypass, tenant isolation, race condition)
- **High:** 3 bugs (missing error logging, memory leak, privilege escalation)
- **Medium:** 3 bugs (tenant context precedence, IP spoofing, subdomain validation)
- **Low:** 0 bugs

---

## Recommendations

### Immediate Actions (Critical Priority)

1. **Fix CORS origin bypass (Bug #1)**: Remove the fallback logic that sets CORS headers for requests without Origin header. Deploy immediately to prevent CSRF attacks.

2. **Fix tenant isolation bypass (Bug #2)**: Add validation to reject JWTs with zero tenant IDs when tenant context is required. This prevents cross-tenant data access.

3. **Fix race condition in rate limiter (Bug #3)**: Release mutex before calling next handler to prevent race conditions and performance issues.

### High Priority Security Fixes

4. **Add database error logging (Bug #4)**: Implement proper error logging in tenant middleware for production debugging and monitoring.

5. **Fix memory leak (Bug #5)**: Add context-based cancellation to cleanup goroutine to prevent resource exhaustion.

6. **Add privilege escalation protection (Bug #7)**: Implement database validation for super admin checks to prevent revoked admins from retaining access.

### Medium Priority Improvements

7. **Fix tenant context logic (Bug #6)**: Enforce subdomain tenant precedence and don't allow JWT-only tenant on tenant subdomains.

8. **Fix global rate limiter for production (Bug #8)**: Add trusted proxy support to global rate limiter for proper operation behind reverse proxies.

9. **Validate subdomains properly (Bug #9)**: Add DNS-compliant subdomain validation to prevent invalid inputs and potential injection attacks.

### Security Best Practices

- Implement JWT token revocation mechanism (Redis blacklist) for immediate privilege revocation
- Add audit logging for all authentication and authorization decisions
- Implement rate limiting at multiple levels (per-endpoint, per-user, per-IP)
- Regular security audits of authentication and authorization code
- Add integration tests for multi-tenant isolation
- Monitor for unusual tenant access patterns
- Implement WAF rules to detect and block malicious patterns

### Code Quality Improvements

- Add comprehensive unit tests for all middleware edge cases
- Document security assumptions and threat model
- Implement middleware composition helpers to ensure correct ordering
- Add telemetry for authentication failures and rate limit hits
- Create security checklist for new middleware development
