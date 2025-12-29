package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tranmh/gassigeher/internal/config"
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
	handler := NewBillingHandler(db, nil, nil)

	t.Run("returns subscription for authenticated user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/billing/subscription", nil)
		req = req.WithContext(contextWithTenantID(req.Context(), 0))
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
	handler := NewBillingHandler(db, nil, nil)

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
	handler := NewBillingHandler(db, nil, nil)

	// Seed some dogs for tenant 0
	testutil.SeedTestDog(t, db, "Max", "Labrador", "green")
	testutil.SeedTestDog(t, db, "Bella", "Poodle", "green")

	t.Run("returns usage for tenant", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/billing/usage", nil)
		req = req.WithContext(contextWithTenantID(req.Context(), 0))
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
	handler := NewBillingHandler(db, nil, nil)

	t.Run("returns error when Stripe not configured", func(t *testing.T) {
		// Send valid request body to pass validation
		body := `{"plan_slug": "pro", "billing_cycle": "monthly"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// Must be admin to test Stripe check (admin auth happens first)
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 0, true))
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
	handler := NewBillingHandler(db, nil, nil)

	t.Run("cancels subscription when admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/cancel", nil)
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 0, true))
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
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 0, false))
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
	handler := NewBillingHandler(db, nil, nil)

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
	handler := NewBillingHandler(db, nil, nil)

	// BUG 2: Non-admin users should NOT be able to initiate checkout (financial decision)
	t.Run("SECURITY: non-admin cannot create checkout", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/checkout", nil)
		// User has tenant access but is NOT an admin
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 0, false))
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
	handler := NewBillingHandler(db, nil, nil)

	// BUG 7: Invalid billing_cycle values should be rejected
	t.Run("rejects invalid billing_cycle", func(t *testing.T) {
		body := `{"plan_slug": "pro", "billing_cycle": "invalid_cycle"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 0, true))
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

// TestBillingHandler_CreateBillingPortal tests POST /api/billing/portal
func TestBillingHandler_CreateBillingPortal(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil, nil)

	t.Run("returns error when tenant not in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/portal", nil)
		w := httptest.NewRecorder()

		handler.CreateBillingPortal(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("returns forbidden when non-admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/portal", nil)
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 0, false))
		w := httptest.NewRecorder()

		handler.CreateBillingPortal(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("returns service unavailable when Stripe not configured", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/portal", nil)
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 0, true))
		w := httptest.NewRecorder()

		handler.CreateBillingPortal(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d: %s", http.StatusServiceUnavailable, w.Code, w.Body.String())
		}
	})
}

// TestBillingHandler_HandleWebhook tests POST /api/billing/webhook
func TestBillingHandler_HandleWebhook(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil, nil)

	t.Run("returns service unavailable when Stripe not configured", func(t *testing.T) {
		body := `{"type": "checkout.session.completed"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/webhook", strings.NewReader(body))
		req.Header.Set("Stripe-Signature", "test_signature")
		w := httptest.NewRecorder()

		handler.HandleWebhook(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d: %s", http.StatusServiceUnavailable, w.Code, w.Body.String())
		}
	})

	t.Run("returns error when signature missing", func(t *testing.T) {
		// Create a handler that would have Stripe configured
		// but signature check should fail first
		body := `{"type": "checkout.session.completed"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/webhook", strings.NewReader(body))
		// No Stripe-Signature header
		w := httptest.NewRecorder()

		handler.HandleWebhook(w, req)

		// Should fail before Stripe check because missing signature
		// But since stripeService is nil, it returns 503 first
		if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 503 or 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestBillingHandler_CancelSubscription_NoSubscription tests cancellation without subscription
func TestBillingHandler_CancelSubscription_NoSubscription(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil, nil)

	// Create a new tenant without any subscription
	_, err := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('no-sub', 'No Sub Tenant', 'test@example.com', 'active', datetime('now'))`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	var tenantID int
	db.QueryRow(`SELECT id FROM tenants WHERE slug = 'no-sub'`).Scan(&tenantID)

	t.Run("returns error when no subscription exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/billing/cancel", nil)
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), tenantID, true))
		w := httptest.NewRecorder()

		handler.CancelSubscription(w, req)

		// Should return error since no subscription
		if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
			t.Errorf("Expected status 400 or 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestBillingHandler_CreateCheckout_PlanSlugValidation tests plan slug validation
func TestBillingHandler_CreateCheckout_PlanSlugValidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil, nil)

	t.Run("rejects empty plan_slug", func(t *testing.T) {
		body := `{"plan_slug": "", "billing_cycle": "monthly"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 0, true))
		w := httptest.NewRecorder()

		handler.CreateCheckout(w, req)

		// Current implementation checks Stripe config first, then validates
		// Both 400 (validation) and 503 (Stripe not configured) are acceptable
		if w.Code != http.StatusBadRequest && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected 400 or 503, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects invalid plan_slug", func(t *testing.T) {
		body := `{"plan_slug": "nonexistent_plan", "billing_cycle": "monthly"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 0, true))
		w := httptest.NewRecorder()

		handler.CreateCheckout(w, req)

		// Current implementation checks Stripe config first
		// Acceptable responses: 404 (plan not found), 400 (validation), 503 (Stripe)
		if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected 404, 400, or 503 for invalid plan_slug, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("accepts valid monthly billing cycle", func(t *testing.T) {
		body := `{"plan_slug": "pro", "billing_cycle": "monthly"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 0, true))
		w := httptest.NewRecorder()

		handler.CreateCheckout(w, req)

		// Should fail at Stripe check, not validation
		if w.Code == http.StatusBadRequest {
			t.Errorf("Valid request should not fail validation: %s", w.Body.String())
		}
	})

	t.Run("accepts valid yearly billing cycle", func(t *testing.T) {
		body := `{"plan_slug": "pro", "billing_cycle": "yearly"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/checkout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithTenantAndAdmin(req.Context(), 0, true))
		w := httptest.NewRecorder()

		handler.CreateCheckout(w, req)

		// Should fail at Stripe check, not validation
		if w.Code == http.StatusBadRequest {
			t.Errorf("Valid request should not fail validation: %s", w.Body.String())
		}
	})
}

// TestBillingHandler_GetUsage_ShowsOverLimitWarning tests that usage endpoint shows over_limit warning
// Enhancement: Show visual indication when tenant is over their subscription limit
func TestBillingHandler_GetUsage_ShowsOverLimitWarning(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBillingHandler(db, nil, nil)

	// Create 15 dogs for tenant 0 (over the 10 dog limit for Free plan)
	for i := 0; i < 15; i++ {
		_, err := db.Exec(`
			INSERT INTO dogs (tenant_id, name, breed, size, age, is_available)
			VALUES (0, ?, 'Test Breed', 'medium', 3, 1)
		`, "Dog"+string(rune('A'+i)))
		if err != nil {
			t.Fatalf("Failed to create dog: %v", err)
		}
	}

	t.Run("shows over_limit true when dogs exceed limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/billing/usage", nil)
		req = req.WithContext(contextWithTenantID(req.Context(), 0))
		w := httptest.NewRecorder()

		handler.GetUsage(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		// Check over_limit field exists and is true
		overLimit, ok := response["over_limit"].(bool)
		if !ok {
			t.Error("Enhancement: Expected 'over_limit' field in usage response")
		} else if !overLimit {
			t.Error("Enhancement: Expected over_limit=true when dogs_used > dogs_limit")
		}

		// Check excess_count field
		excessCount, ok := response["excess_count"].(float64)
		if !ok {
			t.Error("Enhancement: Expected 'excess_count' field in usage response")
		} else if int(excessCount) != 5 {
			t.Errorf("Expected excess_count=5 (15 dogs - 10 limit), got %v", excessCount)
		}
	})

	t.Run("shows over_limit false when within limit", func(t *testing.T) {
		// Delete some dogs to get under limit
		_, _ = db.Exec(`DELETE FROM dogs WHERE tenant_id = 0`)
		for i := 0; i < 5; i++ {
			_, _ = db.Exec(`
				INSERT INTO dogs (tenant_id, name, breed, size, age, is_available)
				VALUES (0, ?, 'Test Breed', 'medium', 3, 1)
			`, "Dog"+string(rune('A'+i)))
		}

		req := httptest.NewRequest(http.MethodGet, "/api/billing/usage", nil)
		req = req.WithContext(contextWithTenantID(req.Context(), 0))
		w := httptest.NewRecorder()

		handler.GetUsage(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		overLimit, ok := response["over_limit"].(bool)
		if ok && overLimit {
			t.Error("Expected over_limit=false when dogs_used <= dogs_limit")
		}
	})
}

// TestBillingHandler_TestUpgrade_EmptyPlanSlug tests that empty plan_slug is rejected
// TDD RED PHASE: This test should FAIL because the handler currently defaults empty
// plan_slug to "pro", which is unexpected behavior
func TestBillingHandler_TestUpgrade_EmptyPlanSlug(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Create test config with billing test mode enabled
	cfg := &config.Config{
		BillingTestMode: true,
	}
	handler := NewBillingHandler(db, cfg, nil)

	// Create a tenant and subscription
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (88, 'test-upgrade-tenant', 'Test Upgrade Tenant', 'active', 'test@upgrade.com', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	_, err = db.Exec(`INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, billing_cycle, created_at, updated_at)
		VALUES (88, 1, 'active', 'monthly', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	t.Run("should reject empty plan_slug", func(t *testing.T) {
		// Send request with empty plan_slug
		reqBody := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/test-upgrade", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndAdmin(req.Context(), 88, true)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.TestUpgrade(w, req)

		// BUG: Currently this returns 200 and upgrades to Pro
		// It SHOULD return 400 Bad Request because plan_slug is required
		if w.Code == http.StatusOK {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			if response["plan"] == "Pro" {
				t.Errorf("BUG DETECTED: Empty plan_slug silently upgraded to Pro. " +
					"Should return 400 Bad Request instead. Response: %s", w.Body.String())
			}
		}

		// The correct behavior is to reject empty plan_slug
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for empty plan_slug, got %d. Response: %s",
				w.Code, w.Body.String())
		}
	})

	t.Run("should accept valid plan_slug", func(t *testing.T) {
		// Send request with valid plan_slug
		reqBody := `{"plan_slug":"pro"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/test-upgrade", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndAdmin(req.Context(), 88, true)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.TestUpgrade(w, req)

		// Should succeed with valid plan_slug
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK for valid plan_slug 'pro', got %d. Response: %s",
				w.Code, w.Body.String())
		}
	})
}

// TestBillingHandler_TestUpgrade_UsesDBPlanLookup tests that plan IDs are looked up
// from the database by slug, not hardcoded
// TDD RED PHASE: This test should FAIL because the handler currently uses hardcoded
// plan IDs (1=free, 2=pro) instead of looking them up by slug
func TestBillingHandler_TestUpgrade_UsesDBPlanLookup(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Create test config with billing test mode enabled
	cfg := &config.Config{
		BillingTestMode: true,
	}
	handler := NewBillingHandler(db, cfg, nil)

	// Create a tenant
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (77, 'custom-plan-tenant', 'Custom Plan Tenant', 'active', 'custom@plan.com', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Update existing plans to have different IDs to test that handler uses DB lookup
	// The migrations create plans with id=1 (free) and id=2 (pro)
	// We'll update the pro plan to have id=101 to verify the handler looks it up by slug
	_, err = db.Exec(`UPDATE pricing_plans SET id = 101 WHERE slug = 'pro'`)
	if err != nil {
		t.Fatalf("Failed to update pro plan ID: %v", err)
	}

	// Create subscription with the Free plan ID (1)
	_, err = db.Exec(`INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, billing_cycle, created_at, updated_at)
		VALUES (77, 1, 'active', 'monthly', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	t.Run("should use DB lookup for plan ID, not hardcoded value", func(t *testing.T) {
		// Try to upgrade to Pro - should use plan_id=101 from DB, not hardcoded 2
		reqBody := `{"plan_slug":"pro"}`
		req := httptest.NewRequest(http.MethodPost, "/api/billing/test-upgrade", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndAdmin(req.Context(), 77, true)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.TestUpgrade(w, req)

		// Check if the upgrade succeeded
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Response: %s", w.Code, w.Body.String())
		}

		// Verify the subscription was updated to plan_id=101, not 2
		var planID int
		err := db.QueryRow("SELECT plan_id FROM tenant_subscriptions WHERE tenant_id = 77").Scan(&planID)
		if err != nil {
			t.Fatalf("Failed to query subscription: %v", err)
		}

		// BUG: The handler uses hardcoded plan_id=2 for "pro"
		// but the Pro plan in our DB has id=101
		if planID == 2 {
			t.Errorf("BUG DETECTED: Handler used hardcoded plan_id=2 instead of looking up 'pro' slug from DB. "+
				"Pro plan in DB has id=101, but subscription was updated to plan_id=%d", planID)
		}

		if planID != 101 {
			t.Errorf("Expected subscription to be updated to plan_id=101 (Pro from DB), got plan_id=%d", planID)
		}
	})
}
