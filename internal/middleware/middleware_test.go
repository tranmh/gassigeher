package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/services"
)

// DONE: TestAuthMiddleware tests JWT authentication middleware
func TestAuthMiddleware(t *testing.T) {
	jwtSecret := "test-secret"
	authService := services.NewAuthService(jwtSecret, 24)
	middleware := AuthMiddleware(jwtSecret)

	// Create a test handler that checks context values
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(UserIDKey)
		email := r.Context().Value(EmailKey)
		isAdmin := r.Context().Value(IsAdminKey)

		if userID == nil {
			t.Error("UserID should be set in context")
		}
		if email == nil {
			t.Error("Email should be set in context")
		}
		if isAdmin == nil {
			t.Error("IsAdmin should be set in context")
		}

		w.WriteHeader(http.StatusOK)
	})

	t.Run("valid token", func(t *testing.T) {
		// Generate valid token
		token, _ := authService.GenerateJWT(1, "test@example.com", false, false, false, 0)

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})

	t.Run("invalid authorization format - no Bearer", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "some-token")

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		// Create service with 0 expiration
		expiredService := &services.AuthService{}
		expiredService = services.NewAuthService(jwtSecret, 0)
		token, _ := expiredService.GenerateJWT(1, "test@example.com", false, false, false, 0)

		// Wait for expiration
		time.Sleep(1 * time.Second)

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for expired token, got %d", rec.Code)
		}
	})

	t.Run("admin user context", func(t *testing.T) {
		token, _ := authService.GenerateJWT(1, "admin@example.com", true, false, false, 0)

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rec := httptest.NewRecorder()

		// Handler that checks admin flag
		adminCheckHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isAdmin := r.Context().Value(IsAdminKey)
			if isAdmin != true {
				t.Error("IsAdmin should be true for admin user")
			}
			w.WriteHeader(http.StatusOK)
		})

		middleware(adminCheckHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})
}

// DONE: TestRequireAdmin tests admin authorization middleware
func TestRequireAdmin(t *testing.T) {
	middleware := RequireAdmin

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("admin user allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin/test", nil)
		ctx := context.WithValue(req.Context(), IsAdminKey, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for admin, got %d", rec.Code)
		}
	})

	t.Run("non-admin user forbidden", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin/test", nil)
		ctx := context.WithValue(req.Context(), IsAdminKey, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for non-admin, got %d", rec.Code)
		}
	})

	t.Run("missing admin flag in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin/test", nil)
		rec := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 when admin flag missing, got %d", rec.Code)
		}
	})
}

// DONE: TestCORSMiddleware tests CORS headers middleware
func TestCORSMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORSMiddleware("http://localhost:8080")(testHandler)

	t.Run("adds CORS headers to GET request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		// BUG FIX #1: Test updated for restricted CORS policy
		req.Header.Set("Origin", "http://localhost:8080")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		headers := rec.Header()

		// After BUG #1 fix: CORS returns requesting origin, not *
		if headers.Get("Access-Control-Allow-Origin") != "http://localhost:8080" {
			t.Errorf("Expected Access-Control-Allow-Origin to be http://localhost:8080, got %s", headers.Get("Access-Control-Allow-Origin"))
		}

		if headers.Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected Access-Control-Allow-Methods to be set")
		}

		if headers.Get("Access-Control-Allow-Headers") == "" {
			t.Error("Expected Access-Control-Allow-Headers to be set")
		}
	})

	t.Run("handles OPTIONS preflight request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:8080") // BUG FIX #1
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", rec.Code)
		}

		// Verify CORS headers are present
		if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:8080" {
			t.Error("Expected CORS headers on OPTIONS request")
		}
	})
}

// DONE: TestSecurityHeadersMiddleware tests security headers middleware
func TestSecurityHeadersMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeadersMiddleware(testHandler)

	t.Run("adds security headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		headers := rec.Header()

		// Check all security headers
		if headers.Get("X-Content-Type-Options") != "nosniff" {
			t.Error("Expected X-Content-Type-Options: nosniff")
		}

		if headers.Get("X-Frame-Options") != "DENY" {
			t.Error("Expected X-Frame-Options: DENY")
		}

		if headers.Get("X-XSS-Protection") == "" {
			t.Error("Expected X-XSS-Protection to be set")
		}

		if headers.Get("Strict-Transport-Security") == "" {
			t.Error("Expected HSTS header to be set")
		}

		if headers.Get("Content-Security-Policy") == "" {
			t.Error("Expected CSP header to be set")
		}
	})
}

// DONE: TestLoggingMiddleware tests logging middleware
func TestLoggingMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(testHandler)

	t.Run("logs request without error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()

		// Should not panic or error
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("logs POST request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/users", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})
}

// TestCORSMiddleware_NoOriginHeader tests CORS does NOT set headers when no Origin is present
// BUG FIX: CORS bypass vulnerability - should not set headers for same-origin requests
func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORSMiddleware("http://localhost:8080")(testHandler)

	t.Run("no CORS headers when Origin header is missing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		// Intentionally NOT setting Origin header - same-origin request
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Should NOT set Access-Control-Allow-Origin for same-origin requests
		corsHeader := rec.Header().Get("Access-Control-Allow-Origin")
		if corsHeader != "" {
			t.Errorf("Expected no Access-Control-Allow-Origin header for same-origin request, got %s", corsHeader)
		}
	})

	t.Run("rejects unknown origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Origin", "http://evil-site.com")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Should NOT set Access-Control-Allow-Origin for unknown origins
		corsHeader := rec.Header().Get("Access-Control-Allow-Origin")
		if corsHeader != "" {
			t.Errorf("Expected no Access-Control-Allow-Origin for unknown origin, got %s", corsHeader)
		}
	})
}

// TestAuthMiddleware_TenantValidation tests tenant ID validation in JWT
// BUG FIX: Tenant isolation bypass - JWT with tenant_id=0 should be rejected when subdomain is set
func TestAuthMiddleware_TenantValidation(t *testing.T) {
	jwtSecret := "test-secret"
	authService := services.NewAuthService(jwtSecret, 24)
	middleware := AuthMiddleware(jwtSecret)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("rejects JWT with zero tenant_id when subdomain tenant is set", func(t *testing.T) {
		// Generate JWT with tenant_id=0 (no tenant)
		token, _ := authService.GenerateJWT(1, "test@example.com", false, false, false, 0)

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		// Set subdomain tenant in context (simulating TenantMiddleware ran first)
		ctx := context.WithValue(req.Context(), TenantIDKey, 1)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		// Should reject - JWT tenant_id=0 doesn't match subdomain tenant_id=1
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 for JWT with tenant_id=0 on tenant subdomain, got %d", rec.Code)
		}
	})

	t.Run("allows matching tenant IDs", func(t *testing.T) {
		// Generate JWT with tenant_id=1
		token, _ := authService.GenerateJWT(1, "test@example.com", false, false, false, 1)

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		// Set matching subdomain tenant in context
		ctx := context.WithValue(req.Context(), TenantIDKey, 1)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 for matching tenant IDs, got %d", rec.Code)
		}
	})

	t.Run("rejects mismatched tenant IDs", func(t *testing.T) {
		// Generate JWT with tenant_id=2
		token, _ := authService.GenerateJWT(1, "test@example.com", false, false, false, 2)

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		// Set different subdomain tenant in context
		ctx := context.WithValue(req.Context(), TenantIDKey, 1)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 for mismatched tenant IDs, got %d", rec.Code)
		}
	})
}

// TestRateLimitLogin_IPSpoofingPrevention tests that rate limiting cannot be bypassed
// by spoofing the X-Forwarded-For header when proxy trust is disabled
func TestRateLimitLogin_IPSpoofingPrevention(t *testing.T) {
	// Reset rate limiter state for clean test
	ResetRateLimiter()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitLogin(testHandler)

	t.Run("spoofed X-Forwarded-For does not bypass rate limit", func(t *testing.T) {
		// Make 5 requests (the limit) with the same RemoteAddr but different X-Forwarded-For
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("POST", "/api/auth/login", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			// Attacker tries to spoof different IPs to bypass rate limiting
			req.Header.Set("X-Forwarded-For", "10.0.0."+string(rune('1'+i)))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Request %d should succeed, got status %d", i+1, rec.Code)
			}
		}

		// 6th request should be rate limited (based on RemoteAddr, not spoofed header)
		req := httptest.NewRequest("POST", "/api/auth/login", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		req.Header.Set("X-Forwarded-For", "completely-different-ip")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("6th request should be rate limited, got status %d (expected 429)", rec.Code)
		}
	})
}

// TestRateLimitLogin_TrustedProxy tests rate limiting with trusted proxy configuration
func TestRateLimitLogin_TrustedProxy(t *testing.T) {
	// Reset rate limiter state
	ResetRateLimiter()

	// Configure trusted proxy
	SetTrustedProxies([]string{"127.0.0.1"})
	defer SetTrustedProxies(nil) // Reset after test

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitLogin(testHandler)

	t.Run("uses X-Forwarded-For when from trusted proxy", func(t *testing.T) {
		// Make 5 requests from trusted proxy with same client IP
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("POST", "/api/auth/login", nil)
			req.RemoteAddr = "127.0.0.1:12345" // Trusted proxy
			req.Header.Set("X-Forwarded-For", "203.0.113.50") // Real client IP

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Request %d should succeed, got status %d", i+1, rec.Code)
			}
		}

		// 6th request should be rate limited based on client IP from header
		req := httptest.NewRequest("POST", "/api/auth/login", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.50")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("6th request should be rate limited, got status %d", rec.Code)
		}
	})
}

// TestRateLimitLogin_NoConcurrentBlocking tests that rate limiter doesn't serialize requests
// from different IPs (i.e., doesn't hold mutex during handler execution)
func TestRateLimitLogin_NoConcurrentBlocking(t *testing.T) {
	ResetRateLimiter()

	// Handler that simulates slow processing
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitLogin(slowHandler)

	// Send 3 concurrent requests from different IPs
	numRequests := 3
	done := make(chan time.Duration, numRequests)

	start := time.Now()
	for i := 0; i < numRequests; i++ {
		go func(ip int) {
			reqStart := time.Now()
			req := httptest.NewRequest("POST", "/api/auth/login", nil)
			req.RemoteAddr = "192.168.1." + string(rune('0'+ip)) + ":12345"

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			done <- time.Since(reqStart)
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
	totalTime := time.Since(start)

	// If requests are serialized, total time would be ~300ms
	// If concurrent, total time should be ~100ms (plus some overhead)
	// We use 200ms as threshold to be safe
	if totalTime > 200*time.Millisecond {
		t.Errorf("Requests appear to be serialized (took %v). "+
			"Rate limiter should not hold mutex during handler execution.", totalTime)
	}
}

// TestRequireSuperAdmin tests super admin authorization middleware
func TestRequireSuperAdmin(t *testing.T) {
	middleware := RequireSuperAdmin

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("super admin allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/superadmin/test", nil)
		ctx := context.WithValue(req.Context(), IsSuperAdminKey, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for super admin, got %d", rec.Code)
		}
	})

	t.Run("non-super admin forbidden", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/superadmin/test", nil)
		ctx := context.WithValue(req.Context(), IsSuperAdminKey, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for non-super admin, got %d", rec.Code)
		}
	})

	t.Run("missing super admin flag in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/superadmin/test", nil)
		rec := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 when super admin flag missing, got %d", rec.Code)
		}
	})

	t.Run("regular admin is not super admin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/superadmin/test", nil)
		ctx := context.WithValue(req.Context(), IsAdminKey, true)
		ctx = context.WithValue(ctx, IsSuperAdminKey, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for regular admin, got %d", rec.Code)
		}
	})
}

// TestRequireCentralAdmin tests central admin authorization middleware
func TestRequireCentralAdmin(t *testing.T) {
	middleware := RequireCentralAdmin

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("central admin allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/central/test", nil)
		ctx := context.WithValue(req.Context(), IsCentralAdminKey, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for central admin, got %d", rec.Code)
		}
	})

	t.Run("non-central admin forbidden", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/central/test", nil)
		ctx := context.WithValue(req.Context(), IsCentralAdminKey, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for non-central admin, got %d", rec.Code)
		}
	})

	t.Run("missing central admin flag in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/central/test", nil)
		rec := httptest.NewRecorder()

		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 when central admin flag missing, got %d", rec.Code)
		}
	})

	t.Run("super admin is not automatically central admin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/central/test", nil)
		ctx := context.WithValue(req.Context(), IsSuperAdminKey, true)
		ctx = context.WithValue(ctx, IsCentralAdminKey, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for super admin without central admin flag, got %d", rec.Code)
		}
	})
}

// TestExtractSubdomain tests the subdomain extraction function
func TestExtractSubdomain(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		baseDomain string
		expected   string
	}{
		{"valid subdomain", "tierheim-goeppingen.gassigeher.org", "gassigeher.org", "tierheim-goeppingen"},
		{"no subdomain", "gassigeher.org", "gassigeher.org", ""},
		{"localhost", "localhost", "gassigeher.org", ""},
		{"localhost with port", "localhost:8080", "gassigeher.org", ""},
		{"127.0.0.1", "127.0.0.1", "gassigeher.org", ""},
		{"host with port", "tierheim.gassigeher.org:8080", "gassigeher.org", "tierheim"},
		{"empty base domain", "tierheim.gassigeher.org", "", ""},
		{"base domain with port", "tierheim.gassigeher.org", "gassigeher.org:8080", "tierheim"},
		{"multi-level subdomain rejected", "sub1.sub2.gassigeher.org", "gassigeher.org", ""},
		{"www subdomain", "www.gassigeher.org", "gassigeher.org", "www"},
		{"different domain", "tierheim.example.com", "gassigeher.org", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSubdomain(tt.host, tt.baseDomain)
			if result != tt.expected {
				t.Errorf("extractSubdomain(%q, %q) = %q, want %q", tt.host, tt.baseDomain, result, tt.expected)
			}
		})
	}
}

// TestRequireTenant tests the tenant requirement middleware
func TestRequireTenant(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireTenant(testHandler)

	t.Run("tenant present - allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		ctx := context.WithValue(req.Context(), TenantIDKey, 1)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 with tenant, got %d", rec.Code)
		}
	})

	t.Run("tenant missing - rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 without tenant, got %d", rec.Code)
		}
	})

	t.Run("tenant zero - rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		ctx := context.WithValue(req.Context(), TenantIDKey, 0)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 with tenant=0, got %d", rec.Code)
		}
	})
}

// TestOptionalTenant tests the optional tenant middleware
func TestOptionalTenant(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := OptionalTenant(testHandler)

	t.Run("passes through with tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		ctx := context.WithValue(req.Context(), TenantIDKey, 1)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("passes through without tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})
}

// TestGetTenantID tests the tenant ID helper function
func TestGetTenantID(t *testing.T) {
	t.Run("returns tenant ID from context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		ctx := context.WithValue(req.Context(), TenantIDKey, 42)
		req = req.WithContext(ctx)

		tenantID := GetTenantID(req)
		if tenantID != 42 {
			t.Errorf("Expected tenant ID 42, got %d", tenantID)
		}
	})

	t.Run("returns 0 when not set", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)

		tenantID := GetTenantID(req)
		if tenantID != 0 {
			t.Errorf("Expected tenant ID 0, got %d", tenantID)
		}
	})
}

// TestGetTenantSlug tests the tenant slug helper function
func TestGetTenantSlug(t *testing.T) {
	t.Run("returns tenant slug from context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		ctx := context.WithValue(req.Context(), TenantSlugKey, "tierheim-goeppingen")
		req = req.WithContext(ctx)

		slug := GetTenantSlug(req)
		if slug != "tierheim-goeppingen" {
			t.Errorf("Expected slug 'tierheim-goeppingen', got '%s'", slug)
		}
	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)

		slug := GetTenantSlug(req)
		if slug != "" {
			t.Errorf("Expected empty slug, got '%s'", slug)
		}
	})
}

// TestGlobalRateLimiter tests the global rate limiter
func TestGlobalRateLimiter(t *testing.T) {
	t.Run("creates new limiter for IP", func(t *testing.T) {
		limiter := NewGlobalRateLimiter(10, 5)
		defer limiter.Close()

		l1 := limiter.GetLimiter("192.168.1.1")
		if l1 == nil {
			t.Error("Expected limiter to be created")
		}
	})

	t.Run("reuses existing limiter for same IP", func(t *testing.T) {
		limiter := NewGlobalRateLimiter(10, 5)
		defer limiter.Close()

		l1 := limiter.GetLimiter("192.168.1.1")
		l2 := limiter.GetLimiter("192.168.1.1")

		if l1 != l2 {
			t.Error("Expected same limiter instance for same IP")
		}
	})

	t.Run("creates different limiters for different IPs", func(t *testing.T) {
		limiter := NewGlobalRateLimiter(10, 5)
		defer limiter.Close()

		l1 := limiter.GetLimiter("192.168.1.1")
		l2 := limiter.GetLimiter("192.168.1.2")

		if l1 == l2 {
			t.Error("Expected different limiter instances for different IPs")
		}
	})
}

// TestGlobalRateLimit_Middleware tests the global rate limit middleware
func TestGlobalRateLimit_Middleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allows requests within limit", func(t *testing.T) {
		// High RPS limit for this test
		handler := GlobalRateLimit(100, 10)(testHandler)

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("rate limits excessive requests", func(t *testing.T) {
		// Very low limit - 1 request per second, burst of 1
		handler := GlobalRateLimit(1, 1)(testHandler)

		// First request should succeed
		req1 := httptest.NewRequest("GET", "/api/test", nil)
		req1.RemoteAddr = "10.0.0.50:12345"
		rec1 := httptest.NewRecorder()
		handler.ServeHTTP(rec1, req1)

		if rec1.Code != http.StatusOK {
			t.Errorf("First request: expected status 200, got %d", rec1.Code)
		}

		// Immediate second request should be rate limited
		req2 := httptest.NewRequest("GET", "/api/test", nil)
		req2.RemoteAddr = "10.0.0.50:12345"
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusTooManyRequests {
			t.Errorf("Second request: expected status 429, got %d", rec2.Code)
		}

		// Verify Retry-After header
		if rec2.Header().Get("Retry-After") == "" {
			t.Error("Expected Retry-After header on rate limited response")
		}
	})
}

// TestAuthMiddleware_ImpersonationClaims tests impersonation claims extraction
func TestAuthMiddleware_ImpersonationClaims(t *testing.T) {
	jwtSecret := "test-secret"
	authService := services.NewAuthService(jwtSecret, 24)
	middleware := AuthMiddleware(jwtSecret)

	t.Run("extracts impersonation claims when present", func(t *testing.T) {
		// Generate impersonation token
		// GenerateImpersonationJWT(targetUserID, targetEmail, targetIsAdmin, targetIsSuperAdmin, targetIsCentralAdmin, originalUserID, tenantID)
		impersonatedToken, err := authService.GenerateImpersonationJWT(2, "user@example.com", false, false, false, 1, 1)
		if err != nil {
			t.Fatalf("Failed to generate impersonation token: %v", err)
		}

		var capturedOriginalUserID, capturedIsImpersonating interface{}
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedOriginalUserID = r.Context().Value(OriginalUserIDKey)
			capturedIsImpersonating = r.Context().Value(IsImpersonatingKey)
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+impersonatedToken)
		ctx := context.WithValue(req.Context(), TenantIDKey, 1)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		if capturedIsImpersonating != true {
			t.Error("Expected isImpersonating to be true")
		}

		if capturedOriginalUserID != 1 {
			t.Errorf("Expected originalUserID 1, got %v", capturedOriginalUserID)
		}
	})
}

// TestAuthMiddleware_CentralAdminClaim tests central admin claim extraction
func TestAuthMiddleware_CentralAdminClaim(t *testing.T) {
	jwtSecret := "test-secret"
	authService := services.NewAuthService(jwtSecret, 24)
	middleware := AuthMiddleware(jwtSecret)

	t.Run("extracts central admin claim when true", func(t *testing.T) {
		// Generate token with central admin claim
		token, _ := authService.GenerateJWT(1, "central@example.com", false, false, true, 0)

		var capturedIsCentralAdmin interface{}
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedIsCentralAdmin = r.Context().Value(IsCentralAdminKey)
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rec := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		if capturedIsCentralAdmin != true {
			t.Error("Expected isCentralAdmin to be true")
		}
	})
}

// TestLoggingMiddleware_RequestID tests that request ID is generated and set
func TestLoggingMiddleware_RequestID(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request ID is in context
		requestID := r.Context().Value(RequestIDKey)
		if requestID == nil {
			t.Error("Expected request ID in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check X-Request-ID header is set
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID header to be set")
	}
}

// TestLoggingMiddleware_CapturesStatusCode tests status code capture
func TestLoggingMiddleware_CapturesStatusCode(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	handler := LoggingMiddleware(testHandler)

	req := httptest.NewRequest("POST", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}
}
