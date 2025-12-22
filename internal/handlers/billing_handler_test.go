package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// contextWithTenantID is a local helper that uses middleware's TenantIDKey
// to avoid import cycles when testutil imports middleware
func contextWithTenantID(ctx context.Context, tenantID int) context.Context {
	return context.WithValue(ctx, middleware.TenantIDKey, tenantID)
}

// contextWithTenantAndAdmin creates a context with tenant ID and admin flag
func contextWithTenantAndAdmin(ctx context.Context, tenantID int, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, isAdmin)
	return ctx
}

// TestBillingHandler_GetSubscription tests GET /api/billing/subscription (TDD RED Phase)
func TestBillingHandler_GetSubscription(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil)

	t.Run("returns subscription for authenticated user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/billing/subscription", nil)
		req = req.WithContext(contextWithTenantID(req.Context(), 1))
		w := httptest.NewRecorder()

		handler.GetSubscription(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["subscription"] == nil {
			t.Error("Expected subscription in response")
		}
		if response["plan"] == nil {
			t.Error("Expected plan in response")
		}
	})

	t.Run("returns error when tenant not in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/billing/subscription", nil)
		w := httptest.NewRecorder()

		handler.GetSubscription(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

// TestBillingHandler_GetPlans tests GET /api/billing/plans
func TestBillingHandler_GetPlans(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil)

	t.Run("returns all active plans", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/billing/plans", nil)
		w := httptest.NewRecorder()

		handler.GetPlans(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		plans, ok := response["plans"].([]interface{})
		if !ok {
			t.Fatal("Expected plans array in response")
		}
		if len(plans) < 2 {
			t.Errorf("Expected at least 2 plans, got %d", len(plans))
		}
	})
}

// TestBillingHandler_GetUsage tests GET /api/billing/usage
func TestBillingHandler_GetUsage(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil)

	// Seed some dogs for tenant 1
	testutil.SeedTestDog(t, db, "Max", "Labrador", "green")
	testutil.SeedTestDog(t, db, "Bella", "Poodle", "green")

	t.Run("returns usage for tenant", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/billing/usage", nil)
		req = req.WithContext(contextWithTenantID(req.Context(), 1))
		w := httptest.NewRecorder()

		handler.GetUsage(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		dogsUsed, ok := response["dogs_used"].(float64)
		if !ok {
			t.Fatal("Expected dogs_used in response")
		}
		if int(dogsUsed) != 2 {
			t.Errorf("Expected dogs_used = 2, got %d", int(dogsUsed))
		}

		dogsLimit, ok := response["dogs_limit"].(float64)
		if !ok {
			t.Fatal("Expected dogs_limit in response")
		}
		if int(dogsLimit) != 10 {
			t.Errorf("Expected dogs_limit = 10 for free tier, got %d", int(dogsLimit))
		}
	})

	t.Run("returns error when tenant not in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/billing/usage", nil)
		w := httptest.NewRecorder()

		handler.GetUsage(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

// TestBillingHandler_CreateCheckout tests POST /api/billing/checkout
func TestBillingHandler_CreateCheckout(t *testing.T) {
	db := testutil.SetupTestDB(t)
	// Create handler without Stripe (will return error)
	handler := NewBillingHandler(db, nil)

	t.Run("returns error when Stripe not configured", func(t *testing.T) {
		// Send valid request body to pass validation
		body := `{"plan_slug": "pro", "billing_cycle": "monthly"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// Must be admin to test Stripe check (admin auth happens first)
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 1, true))
		w := httptest.NewRecorder()

		handler.CreateCheckout(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("returns error when tenant not in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/checkout", nil)
		w := httptest.NewRecorder()

		handler.CreateCheckout(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

// TestBillingHandler_CancelSubscription tests POST /api/billing/cancel
func TestBillingHandler_CancelSubscription(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil)

	t.Run("cancels subscription when admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/cancel", nil)
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 1, true))
		w := httptest.NewRecorder()

		handler.CancelSubscription(w, req)

		// Should succeed for admin
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("returns error when tenant not in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/cancel", nil)
		w := httptest.NewRecorder()

		handler.CancelSubscription(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	// BUG 2: Non-admin users should NOT be able to cancel subscriptions
	t.Run("SECURITY: non-admin cannot cancel subscription", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/cancel", nil)
		// User has tenant access but is NOT an admin
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 1, false))
		w := httptest.NewRecorder()

		handler.CancelSubscription(w, req)

		// Should be FORBIDDEN (403) - only admins can cancel
		if w.Code != http.StatusForbidden {
			t.Errorf("SECURITY BUG: Non-admin was able to cancel subscription! Expected 403, got %d: %s",
				w.Code, w.Body.String())
		}
	})
}

// TestBillingHandler_WebhookBodyLimit tests that webhook body size is limited (DoS protection)
func TestBillingHandler_WebhookBodyLimit(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil)

	t.Run("SECURITY: rejects oversized webhook body", func(t *testing.T) {
		// Create a 2MB body (should exceed reasonable limit)
		largeBody := strings.Repeat("x", 2*1024*1024)
		req := httptest.NewRequest(http.MethodPost, "/api/billing/webhook", strings.NewReader(largeBody))
		req.Header.Set("Stripe-Signature", "test_signature")
		w := httptest.NewRecorder()

		handler.HandleWebhook(w, req)

		// Should be rejected with 413 Payload Too Large (not 503 from no Stripe)
		// If it returns 503, the body was read fully (no limit) - BUG!
		if w.Code == http.StatusServiceUnavailable {
			t.Error("SECURITY BUG: Oversized body was fully read (DoS vulnerability)")
		}
		// Should get 413 or 400 (bad request) before reaching Stripe check
		if w.Code != http.StatusRequestEntityTooLarge && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 413 or 400, got %d", w.Code)
		}
	})
}

// TestBillingHandler_CreateCheckout_AdminRequired tests admin authorization for checkout
func TestBillingHandler_CreateCheckout_AdminRequired(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil)

	// BUG 2: Non-admin users should NOT be able to initiate checkout (financial decision)
	t.Run("SECURITY: non-admin cannot create checkout", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/checkout", nil)
		// User has tenant access but is NOT an admin
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 1, false))
		w := httptest.NewRecorder()

		handler.CreateCheckout(w, req)

		// Should be FORBIDDEN (403) - only admins can initiate payments
		// Note: It might also return 503 (no Stripe), but security check should come FIRST
		if w.Code != http.StatusForbidden {
			t.Errorf("SECURITY BUG: Non-admin was able to initiate checkout! Expected 403, got %d: %s",
				w.Code, w.Body.String())
		}
	})
}

// TestBillingHandler_CreateCheckout_BillingCycleValidation tests billing_cycle validation
func TestBillingHandler_CreateCheckout_BillingCycleValidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil)

	// BUG 7: Invalid billing_cycle values should be rejected
	t.Run("rejects invalid billing_cycle", func(t *testing.T) {
		body := `{"plan_slug": "pro", "billing_cycle": "invalid_cycle"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 1, true))
		w := httptest.NewRecorder()

		handler.CreateCheckout(w, req)

		// Should be 400 Bad Request for invalid billing_cycle
		// If it returns 503 (no Stripe), validation isn't happening - BUG!
		if w.Code == http.StatusServiceUnavailable {
			t.Error("BUG: Invalid billing_cycle accepted (no validation)")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid billing_cycle, got %d: %s", w.Code, w.Body.String())
		}
	})
}
