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
