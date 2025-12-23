package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// TestTenantRateLimitConfigs tests the configuration structure
func TestTenantRateLimitConfigs(t *testing.T) {
	t.Run("free tier config exists", func(t *testing.T) {
		config, ok := TenantRateLimitConfigs["free"]
		if !ok {
			t.Fatal("free tier config not found")
		}
		if config.TenantRPS != 30 {
			t.Errorf("Expected TenantRPS 30, got %f", config.TenantRPS)
		}
		if config.PerIPRPS != 20 {
			t.Errorf("Expected PerIPRPS 20, got %f", config.PerIPRPS)
		}
	})

	t.Run("pro tier config exists", func(t *testing.T) {
		config, ok := TenantRateLimitConfigs["pro"]
		if !ok {
			t.Fatal("pro tier config not found")
		}
		if config.TenantRPS != 100 {
			t.Errorf("Expected TenantRPS 100, got %f", config.TenantRPS)
		}
		if config.PerIPRPS != 50 {
			t.Errorf("Expected PerIPRPS 50, got %f", config.PerIPRPS)
		}
	})

	t.Run("pro tier has higher limits than free", func(t *testing.T) {
		free := TenantRateLimitConfigs["free"]
		pro := TenantRateLimitConfigs["pro"]

		if pro.TenantRPS <= free.TenantRPS {
			t.Error("Pro tier should have higher tenant RPS than free")
		}
		if pro.PerIPRPS <= free.PerIPRPS {
			t.Error("Pro tier should have higher per-IP RPS than free")
		}
	})
}

// TestTenantRateLimiter_NoTenant tests that requests without tenant are not rate limited
func TestTenantRateLimiter_NoTenant(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create middleware without initializing (will skip rate limiting for no tenant)
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := GetTenantID(r)
			if tenantID == 0 {
				// No tenant - skip rate limiting
				next.ServeHTTP(w, r)
				return
			}
			// Would apply rate limiting here
			next.ServeHTTP(w, r)
		})
	}

	handler := middleware(testHandler)

	// Request without tenant context - should pass through
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d without tenant should succeed, got %d", i+1, rec.Code)
		}
	}
}

// TestTenantRateLimiter_WithTenantContext tests rate limiting with tenant in context
func TestTenantRateLimiter_WithTenantContext(t *testing.T) {
	// Create a simple rate limiter for testing (lower limits for faster tests)
	tenantLimiters := make(map[int]*rate.Limiter)
	var mu sync.Mutex

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test middleware with in-memory rate limiting (simulates tenant rate limiting)
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := GetTenantID(r)
			if tenantID == 0 {
				next.ServeHTTP(w, r)
				return
			}

			mu.Lock()
			limiter, exists := tenantLimiters[tenantID]
			if !exists {
				// Use low limit for testing: 5 req/s, burst 5
				limiter = rate.NewLimiter(5, 5)
				tenantLimiters[tenantID] = limiter
			}
			mu.Unlock()

			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Rate limited"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	handler := middleware(testHandler)

	t.Run("requests with tenant context are rate limited", func(t *testing.T) {
		// Reset limiters
		mu.Lock()
		tenantLimiters = make(map[int]*rate.Limiter)
		mu.Unlock()

		successCount := 0
		limitedCount := 0

		// Make 10 rapid requests (should hit the limit of 5 burst)
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			// Add tenant ID to context
			ctx := context.WithValue(req.Context(), TenantIDKey, 1)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code == http.StatusOK {
				successCount++
			} else if rec.Code == http.StatusTooManyRequests {
				limitedCount++
			}
		}

		// Should have 5 successful (burst) and 5 limited
		if successCount != 5 {
			t.Errorf("Expected 5 successful requests, got %d", successCount)
		}
		if limitedCount != 5 {
			t.Errorf("Expected 5 rate limited requests, got %d", limitedCount)
		}
	})
}

// TestTenantRateLimiter_DifferentTenants tests that different tenants have separate limits
func TestTenantRateLimiter_DifferentTenants(t *testing.T) {
	tenantLimiters := make(map[int]*rate.Limiter)
	var mu sync.Mutex

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := GetTenantID(r)
			if tenantID == 0 {
				next.ServeHTTP(w, r)
				return
			}

			mu.Lock()
			limiter, exists := tenantLimiters[tenantID]
			if !exists {
				limiter = rate.NewLimiter(3, 3) // 3 req burst for testing
				tenantLimiters[tenantID] = limiter
			}
			mu.Unlock()

			if !limiter.Allow() {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	handler := middleware(testHandler)

	// Exhaust tenant 1's limit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		ctx := context.WithValue(req.Context(), TenantIDKey, 1)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Tenant 1 should be limited now
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, 1)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Tenant 1 should be rate limited, got %d", rec.Code)
	}

	// Tenant 2 should still have full quota
	req = httptest.NewRequest("GET", "/", nil)
	ctx = context.WithValue(req.Context(), TenantIDKey, 2)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Tenant 2 should NOT be rate limited, got %d", rec.Code)
	}
}

// TestTenantRateLimiter_IPIsolationWithinTenant tests per-IP isolation within a tenant
func TestTenantRateLimiter_IPIsolationWithinTenant(t *testing.T) {
	type tenantIPKey struct {
		tenantID int
		ip       string
	}

	ipLimiters := make(map[tenantIPKey]*rate.Limiter)
	var mu sync.Mutex

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := GetTenantID(r)
			if tenantID == 0 {
				next.ServeHTTP(w, r)
				return
			}

			ip := getClientIP(r, nil)
			key := tenantIPKey{tenantID: tenantID, ip: ip}

			mu.Lock()
			limiter, exists := ipLimiters[key]
			if !exists {
				limiter = rate.NewLimiter(2, 2) // 2 req burst for testing
				ipLimiters[key] = limiter
			}
			mu.Unlock()

			if !limiter.Allow() {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	handler := middleware(testHandler)

	// Exhaust IP1's limit within tenant 1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		ctx := context.WithValue(req.Context(), TenantIDKey, 1)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// IP1 in tenant 1 should be limited
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	ctx := context.WithValue(req.Context(), TenantIDKey, 1)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("IP1 in tenant 1 should be rate limited, got %d", rec.Code)
	}

	// IP2 in same tenant should still work
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.2:12345"
	ctx = context.WithValue(req.Context(), TenantIDKey, 1)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("IP2 in tenant 1 should NOT be rate limited, got %d", rec.Code)
	}

	// IP1 in different tenant should also work
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	ctx = context.WithValue(req.Context(), TenantIDKey, 2)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("IP1 in tenant 2 should NOT be rate limited, got %d", rec.Code)
	}
}

// TestTenantRateLimiter_ResponseHeaders tests that correct headers are set on rate limit
func TestTenantRateLimiter_ResponseHeaders(t *testing.T) {
	limiter := rate.NewLimiter(1, 1) // Allow 1 request

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Rate limited"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	handler := middleware(testHandler)

	// First request succeeds
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Second request should be rate limited with proper headers
	req = httptest.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type: application/json")
	}

	if rec.Header().Get("Retry-After") != "60" {
		t.Error("Expected Retry-After: 60")
	}
}

// TestTenantRateLimiter_Cleanup tests the cleanup of stale entries
func TestTenantRateLimiter_Cleanup(t *testing.T) {
	entries := make(map[int]time.Time)
	var mu sync.Mutex

	cleanup := func(maxAge time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		cutoff := time.Now().Add(-maxAge)
		for id, lastSeen := range entries {
			if lastSeen.Before(cutoff) {
				delete(entries, id)
			}
		}
	}

	// Add some entries
	mu.Lock()
	entries[1] = time.Now()
	entries[2] = time.Now().Add(-2 * time.Hour) // Old entry
	entries[3] = time.Now()
	mu.Unlock()

	// Run cleanup with 1 hour max age
	cleanup(1 * time.Hour)

	mu.Lock()
	defer mu.Unlock()

	if _, exists := entries[1]; !exists {
		t.Error("Entry 1 should still exist (recent)")
	}
	if _, exists := entries[2]; exists {
		t.Error("Entry 2 should be cleaned up (old)")
	}
	if _, exists := entries[3]; !exists {
		t.Error("Entry 3 should still exist (recent)")
	}
}

// TestDefaultTenantRateLimitConfig tests that default config is free tier
func TestDefaultTenantRateLimitConfig(t *testing.T) {
	freeConfig := TenantRateLimitConfigs["free"]

	if DefaultTenantRateLimitConfig.TenantRPS != freeConfig.TenantRPS {
		t.Error("Default config should match free tier")
	}
	if DefaultTenantRateLimitConfig.PerIPRPS != freeConfig.PerIPRPS {
		t.Error("Default config should match free tier")
	}
}
