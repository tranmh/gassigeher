package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestImpersonationTokenCanEndImpersonation verifies that an impersonation token
// can successfully call the end-impersonation endpoint
// RED PHASE: This test will fail because RequireCentralAdmin blocks impersonation tokens
func TestImpersonationTokenCanEndImpersonation(t *testing.T) {
	// Create a request with impersonation context
	req := httptest.NewRequest("POST", "/api/v1/central-admin/end-impersonation", nil)

	// Set up context as if it came from an impersonation token
	ctx := req.Context()
	ctx = context.WithValue(ctx, UserIDKey, 3)                   // Impersonated user ID
	ctx = context.WithValue(ctx, IsAdminKey, false)              // Impersonated user is not admin
	ctx = context.WithValue(ctx, IsCentralAdminKey, false)       // Impersonated user is not central admin
	ctx = context.WithValue(ctx, IsImpersonatingKey, true)       // This IS an impersonation session
	ctx = context.WithValue(ctx, OriginalUserIDKey, 1)           // Original central admin user ID
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// The middleware that should allow impersonation tokens to end impersonation
	handler := AllowImpersonationEnd(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	handler.ServeHTTP(rr, req)

	// Should allow through, not block with 403
	if rr.Code == http.StatusForbidden {
		t.Errorf("Impersonation token was blocked from ending impersonation: status %d, body: %s",
			rr.Code, rr.Body.String())
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// TestNonImpersonatingUserCannotUseEndImpersonation verifies that regular users
// cannot access the end-impersonation endpoint
func TestNonImpersonatingUserCannotUseEndImpersonation(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/central-admin/end-impersonation", nil)

	// Set up context as a regular user (NOT impersonating)
	ctx := req.Context()
	ctx = context.WithValue(ctx, UserIDKey, 3)
	ctx = context.WithValue(ctx, IsAdminKey, false)
	ctx = context.WithValue(ctx, IsCentralAdminKey, false)
	ctx = context.WithValue(ctx, IsImpersonatingKey, false) // NOT impersonating
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler := AllowImpersonationEnd(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	// Should block non-impersonating users
	if rr.Code != http.StatusForbidden {
		t.Errorf("Non-impersonating user should be blocked, got status %d", rr.Code)
	}
}

// TestCentralAdminCanAccessEndImpersonation verifies that central admins
// can access the endpoint (even if not currently impersonating - for cleanup)
func TestCentralAdminCanAccessEndImpersonation(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/central-admin/end-impersonation", nil)

	// Set up context as central admin
	ctx := req.Context()
	ctx = context.WithValue(ctx, UserIDKey, 1)
	ctx = context.WithValue(ctx, IsAdminKey, true)
	ctx = context.WithValue(ctx, IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, IsImpersonatingKey, false)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler := AllowImpersonationEnd(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	// Central admin should be allowed through
	if rr.Code != http.StatusOK {
		t.Errorf("Central admin should be allowed, got status %d", rr.Code)
	}
}
