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
		token, _ := authService.GenerateJWT(1, "test@example.com", false, false)

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
		token, _ := expiredService.GenerateJWT(1, "test@example.com", false, false)

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
		token, _ := authService.GenerateJWT(1, "admin@example.com", true, false)

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
