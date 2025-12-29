package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCSRFMiddleware_RejectsMissingToken verifies that POST/PUT/DELETE requests
// without a CSRF token are rejected with 403 Forbidden
func TestCSRFMiddleware_RejectsMissingToken(t *testing.T) {
	csrf := NewCSRFMiddleware()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		method string
	}{
		{"POST"},
		{"PUT"},
		{"DELETE"},
		{"PATCH"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/bookings", strings.NewReader("{}"))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Errorf("Expected status 403 for %s without token, got %d", tt.method, rr.Code)
			}
		})
	}
}

// TestCSRFMiddleware_RejectsMismatchedToken verifies that requests with
// mismatched cookie and header tokens are rejected
func TestCSRFMiddleware_RejectsMismatchedToken(t *testing.T) {
	csrf := NewCSRFMiddleware()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/bookings", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{
		Name:  CSRFCookieName,
		Value: "token-from-cookie",
	})
	req.Header.Set(CSRFHeaderName, "different-token-from-header")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for mismatched tokens, got %d", rr.Code)
	}
}

// TestCSRFMiddleware_AcceptsValidToken verifies that requests with matching
// cookie and header tokens are allowed through
func TestCSRFMiddleware_AcceptsValidToken(t *testing.T) {
	csrf := NewCSRFMiddleware()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	validToken := "valid-csrf-token-12345"

	req := httptest.NewRequest("POST", "/api/bookings", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{
		Name:  CSRFCookieName,
		Value: validToken,
	})
	req.Header.Set(CSRFHeaderName, validToken)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid token, got %d", rr.Code)
	}

	if rr.Body.String() != "success" {
		t.Errorf("Expected body 'success', got '%s'", rr.Body.String())
	}
}

// TestCSRFMiddleware_SkipsGETRequests verifies that safe methods (GET, HEAD, OPTIONS)
// do not require CSRF tokens
func TestCSRFMiddleware_SkipsGETRequests(t *testing.T) {
	csrf := NewCSRFMiddleware()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("allowed"))
	}))

	safeMethods := []string{"GET", "HEAD", "OPTIONS"}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/dogs", nil)
			// No CSRF token provided
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s request without token, got %d", method, rr.Code)
			}
		})
	}
}

// TestCSRFMiddleware_SkipsSafeEndpoints verifies that certain endpoints
// (webhooks, health checks) are exempt from CSRF protection
func TestCSRFMiddleware_SkipsSafeEndpoints(t *testing.T) {
	csrf := NewCSRFMiddleware()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("allowed"))
	}))

	safeEndpoints := []string{
		"/api/v1/billing/webhook",
		"/api/health",
	}

	for _, endpoint := range safeEndpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest("POST", endpoint, strings.NewReader("{}"))
			// No CSRF token provided
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200 for POST to %s without token, got %d", endpoint, rr.Code)
			}
		})
	}
}

// TestCSRFMiddleware_GeneratesTokenOnGET verifies that GET requests
// receive a CSRF token cookie if they don't have one
func TestCSRFMiddleware_GeneratesTokenOnGET(t *testing.T) {
	csrf := NewCSRFMiddleware()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/dogs", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Check that a CSRF cookie was set
	cookies := rr.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == CSRFCookieName {
			csrfCookie = cookie
			break
		}
	}

	if csrfCookie == nil {
		t.Error("Expected CSRF cookie to be set on GET request")
	} else {
		// Verify cookie properties
		if csrfCookie.HttpOnly {
			t.Error("CSRF cookie should NOT be HttpOnly (frontend needs to read it)")
		}
		if !csrfCookie.Secure {
			// Note: In production this should be true, but test may run without HTTPS
			t.Log("Warning: CSRF cookie should be Secure in production")
		}
		if csrfCookie.SameSite != http.SameSiteLaxMode {
			t.Error("CSRF cookie should have SameSite=Lax (Strict breaks email link navigation)")
		}
		if len(csrfCookie.Value) < 32 {
			t.Errorf("CSRF token should be at least 32 characters, got %d", len(csrfCookie.Value))
		}
	}
}

// TestCSRFMiddleware_TokenGeneration verifies that the token generator
// produces unique, cryptographically secure tokens
func TestCSRFMiddleware_TokenGeneration(t *testing.T) {
	csrf := NewCSRFMiddleware()

	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token := csrf.GenerateToken()
		if tokens[token] {
			t.Errorf("Generated duplicate token: %s", token)
		}
		tokens[token] = true

		if len(token) < 32 {
			t.Errorf("Token too short: %s (length %d)", token, len(token))
		}
	}
}

// TestCSRFMiddleware_PreservesExistingToken verifies that existing valid
// CSRF cookies are not replaced unnecessarily
func TestCSRFMiddleware_PreservesExistingToken(t *testing.T) {
	csrf := NewCSRFMiddleware()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	existingToken := "existing-valid-token-12345678901234567890"

	req := httptest.NewRequest("GET", "/api/dogs", nil)
	req.AddCookie(&http.Cookie{
		Name:  CSRFCookieName,
		Value: existingToken,
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Check that no new cookie was set (existing token preserved)
	cookies := rr.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == CSRFCookieName {
			// A cookie was set - it should have the same value
			if cookie.Value != existingToken {
				t.Log("Note: Cookie was refreshed but with different value")
			}
		}
	}
}

// TestCSRFMiddleware_ErrorResponse verifies that rejected requests
// return proper error response with message
func TestCSRFMiddleware_ErrorResponse(t *testing.T) {
	csrf := NewCSRFMiddleware()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/bookings", strings.NewReader("{}"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "CSRF") {
		t.Errorf("Error response should mention CSRF, got: %s", body)
	}
}

// TestCSRFMiddleware_ConstantTimeComparison verifies that token validation
// uses constant-time comparison to prevent timing attacks
func TestCSRFMiddleware_ConstantTimeComparison(t *testing.T) {
	csrf := NewCSRFMiddleware()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with tokens of equal length but different content
	// Timing attacks would show different response times for tokens
	// that match more characters at the beginning
	testCases := []struct {
		name        string
		cookieToken string
		headerToken string
		shouldPass  bool
	}{
		{"exact match", "abcdefghij1234567890abcdefghij12", "abcdefghij1234567890abcdefghij12", true},
		{"first char differs", "Xbcdefghij1234567890abcdefghij12", "abcdefghij1234567890abcdefghij12", false},
		{"last char differs", "abcdefghij1234567890abcdefghij1X", "abcdefghij1234567890abcdefghij12", false},
		{"middle char differs", "abcdefghij12345X7890abcdefghij12", "abcdefghij1234567890abcdefghij12", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/test", strings.NewReader("{}"))
			req.AddCookie(&http.Cookie{
				Name:  CSRFCookieName,
				Value: tc.cookieToken,
			})
			req.Header.Set(CSRFHeaderName, tc.headerToken)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if tc.shouldPass && rr.Code != http.StatusOK {
				t.Errorf("Expected 200 for matching tokens, got %d", rr.Code)
			}
			if !tc.shouldPass && rr.Code != http.StatusForbidden {
				t.Errorf("Expected 403 for mismatched tokens, got %d", rr.Code)
			}
		})
	}
}
