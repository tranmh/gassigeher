package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// contextWithTenant creates a context with tenant authentication
func contextWithTenant(ctx context.Context, tenantID, userID int, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, isAdmin)
	return ctx
}

// TestTenantHandler_Register tests tenant registration
func TestTenantHandler_Register(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		BaseDomain:         "gassigeher.org",
		Port:               "8080",
	}
	handler := NewTenantHandler(db, cfg)

	t.Run("creates new tenant with super admin", func(t *testing.T) {
		reqBody := map[string]string{
			"organization_name": "Tierheim München",
			"slug":              "tierheim-muenchen",
			"contact_email":     "kontakt@tierheim-muenchen.de",
			"contact_phone":     "+49 89 123456",
			"address":           "Tierheimstr. 1",
			"city":              "München",
			"postal_code":       "80333",
			"federal_state":     "BY",
			"admin_first_name":  "Max",
			"admin_last_name":   "Mustermann",
			"admin_email":       "admin@tierheim-muenchen.de",
			"admin_password":    "securepassword123",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Register(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["slug"] != "tierheim-muenchen" {
			t.Errorf("Expected slug 'tierheim-muenchen', got %v", response["slug"])
		}

		// Verify tenant was created in database
		var tenantCount int
		db.QueryRow("SELECT COUNT(*) FROM tenants WHERE slug = 'tierheim-muenchen'").Scan(&tenantCount)
		if tenantCount != 1 {
			t.Errorf("Expected 1 tenant, got %d", tenantCount)
		}

		// Verify super admin was created
		var adminCount int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE email = 'admin@tierheim-muenchen.de' AND is_super_admin = 1").Scan(&adminCount)
		if adminCount != 1 {
			t.Errorf("Expected 1 super admin, got %d", adminCount)
		}
	})

	t.Run("returns 400 for missing organization name", func(t *testing.T) {
		reqBody := map[string]string{
			"slug":             "test-tenant-2",
			"contact_email":    "test@example.com",
			"city":             "Berlin",
			"postal_code":      "10115",
			"federal_state":    "BE",
			"admin_first_name": "Admin",
			"admin_last_name":  "Test",
			"admin_email":      "admin@test.com",
			"admin_password":   "password123",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Register(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for short password", func(t *testing.T) {
		reqBody := map[string]string{
			"organization_name": "Test Org",
			"slug":              "test-short-pw",
			"contact_email":     "test@example.com",
			"city":              "Berlin",
			"postal_code":       "10115",
			"federal_state":     "BE",
			"admin_first_name":  "Admin",
			"admin_last_name":   "Test",
			"admin_email":       "admin@test.com",
			"admin_password":    "short",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Register(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestTenantHandler_Register_RejectsReservedSlug tests that reserved slugs are rejected
func TestTenantHandler_Register_RejectsReservedSlug(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewTenantHandler(db, cfg)

	reservedSlugs := []string{"www", "api", "admin", "app", "mail", "support", "demo"}

	for _, slug := range reservedSlugs {
		t.Run("rejects reserved slug: "+slug, func(t *testing.T) {
			reqBody := map[string]string{
				"organization_name": "Test Org",
				"slug":              slug,
				"contact_email":     "test@example.com",
				"city":              "Berlin",
				"postal_code":       "10115",
				"federal_state":     "BE",
				"admin_first_name":  "Admin",
				"admin_last_name":   "Test",
				"admin_email":       "admin@test.com",
				"admin_password":    "password123",
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.Register(rec, req)

			if rec.Code != http.StatusConflict {
				t.Errorf("Expected status 409 for reserved slug '%s', got %d", slug, rec.Code)
			}
		})
	}
}

// TestTenantHandler_Register_ValidatesSlugFormat tests slug format validation
func TestTenantHandler_Register_ValidatesSlugFormat(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewTenantHandler(db, cfg)

	invalidSlugs := []struct {
		slug   string
		reason string
	}{
		{"a", "too short"},
		{"ab", "too short (2 chars)"},
		{"UPPERCASE", "contains uppercase"},
		{"with spaces", "contains spaces"},
		{"with_underscore", "contains underscore"},
		{"-starts-with-dash", "starts with dash"},
		{"ends-with-dash-", "ends with dash"},
		{"1starts-with-number", "starts with number"},
	}

	for _, tc := range invalidSlugs {
		t.Run("rejects: "+tc.reason, func(t *testing.T) {
			reqBody := map[string]string{
				"organization_name": "Test Org",
				"slug":              tc.slug,
				"contact_email":     "test@example.com",
				"city":              "Berlin",
				"postal_code":       "10115",
				"federal_state":     "BE",
				"admin_first_name":  "Admin",
				"admin_last_name":   "Test",
				"admin_email":       "admin@test.com",
				"admin_password":    "password123",
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.Register(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for invalid slug '%s' (%s), got %d", tc.slug, tc.reason, rec.Code)
			}
		})
	}
}

// TestTenantHandler_Register_RejectsDuplicateSlug tests duplicate slug detection
func TestTenantHandler_Register_RejectsDuplicateSlug(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewTenantHandler(db, cfg)

	// Note: test-tenant already exists from SetupTestDB
	reqBody := map[string]string{
		"organization_name": "Duplicate Org",
		"slug":              "test-tenant", // Already exists
		"contact_email":     "test@example.com",
		"city":              "Berlin",
		"postal_code":       "10115",
		"federal_state":     "BE",
		"admin_first_name":  "Admin",
		"admin_last_name":   "Test",
		"admin_email":       "admin@duplicate.com",
		"admin_password":    "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate slug, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// TestTenantHandler_CheckSlug tests slug availability checking
func TestTenantHandler_CheckSlug(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewTenantHandler(db, cfg)

	t.Run("returns available for valid unused slug", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tenants/check-slug?slug=new-shelter", nil)

		rec := httptest.NewRecorder()
		handler.CheckSlug(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["available"] != true {
			t.Errorf("Expected slug to be available")
		}
	})

	t.Run("returns unavailable for existing slug", func(t *testing.T) {
		// test-tenant exists from SetupTestDB
		req := httptest.NewRequest("GET", "/api/tenants/check-slug?slug=test-tenant", nil)

		rec := httptest.NewRecorder()
		handler.CheckSlug(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["available"] != false {
			t.Errorf("Expected slug to be unavailable")
		}
	})

	t.Run("returns unavailable for reserved slug", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tenants/check-slug?slug=admin", nil)

		rec := httptest.NewRecorder()
		handler.CheckSlug(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["available"] != false {
			t.Errorf("Expected reserved slug to be unavailable")
		}
		if response["reason"] != "Reserviert" {
			t.Errorf("Expected reason 'Reserviert', got %v", response["reason"])
		}
	})

	t.Run("returns unavailable for invalid format", func(t *testing.T) {
		// Using short slug (too short) - note: CheckSlug normalizes to lowercase first
		req := httptest.NewRequest("GET", "/api/tenants/check-slug?slug=ab", nil)

		rec := httptest.NewRecorder()
		handler.CheckSlug(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["available"] != false {
			t.Errorf("Expected invalid slug to be unavailable")
		}
		if response["reason"] != "Ungültiges Format" {
			t.Errorf("Expected reason 'Ungültiges Format', got %v", response["reason"])
		}
	})

	t.Run("returns 400 for missing slug", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tenants/check-slug", nil)

		rec := httptest.NewRecorder()
		handler.CheckSlug(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestTenantHandler_GetCurrentTenant tests getting current tenant info
func TestTenantHandler_GetCurrentTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewTenantHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "green")

	t.Run("returns current tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tenant", nil)
		ctx := contextWithTenant(req.Context(), 1, userID, false) // tenant_id=1 from SetupTestDB
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetCurrentTenant(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var tenant map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &tenant)

		if tenant["slug"] != "test-tenant" {
			t.Errorf("Expected slug 'test-tenant', got %v", tenant["slug"])
		}
	})

	t.Run("returns 400 when no tenant in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tenant", nil)
		// No tenant context

		rec := httptest.NewRecorder()
		handler.GetCurrentTenant(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestTenantHandler_UpdateTenant tests updating tenant information
func TestTenantHandler_UpdateTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewTenantHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin User", "blue")
	db.Exec("UPDATE users SET is_admin = 1 WHERE id = ?", adminID)

	t.Run("updates tenant name", func(t *testing.T) {
		reqBody := map[string]string{
			"name": "Updated Tenant Name",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/tenant", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateTenant(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify in database
		var name string
		db.QueryRow("SELECT name FROM tenants WHERE id = 1").Scan(&name)
		if name != "Updated Tenant Name" {
			t.Errorf("Expected name 'Updated Tenant Name', got %s", name)
		}
	})

	t.Run("updates multiple fields", func(t *testing.T) {
		reqBody := map[string]string{
			"contact_email": "new-contact@example.com",
			"city":          "Hamburg",
			"federal_state": "HH",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/tenant", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateTenant(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Verify in database
		var email, city, state string
		db.QueryRow("SELECT contact_email, city, federal_state FROM tenants WHERE id = 1").Scan(&email, &city, &state)
		if email != "new-contact@example.com" {
			t.Errorf("Expected email 'new-contact@example.com', got %s", email)
		}
		if city != "Hamburg" {
			t.Errorf("Expected city 'Hamburg', got %s", city)
		}
		if state != "HH" {
			t.Errorf("Expected state 'HH', got %s", state)
		}
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/tenant", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateTenant(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns 400 when no tenant in context", func(t *testing.T) {
		reqBody := map[string]string{"name": "Test"}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/tenant", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// No tenant context

		rec := httptest.NewRecorder()
		handler.UpdateTenant(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestTenantHandler_GetTenantStats tests getting tenant statistics
func TestTenantHandler_GetTenantStats(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewTenantHandler(db, cfg)

	// Create some test data
	userID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")
	testutil.SeedTestBooking(t, db, userID, dogID, testutil.GetFutureDate(1), "10:00", "scheduled")

	t.Run("returns tenant statistics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tenant/stats", nil)
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetTenantStats(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var stats map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &stats)

		// Verify stats structure exists (actual values may vary)
		if stats == nil {
			t.Error("Expected stats to be returned")
		}
	})

	t.Run("returns 400 when no tenant in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tenant/stats", nil)
		// No tenant context

		rec := httptest.NewRecorder()
		handler.GetTenantStats(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestIsValidSlug tests the slug validation function
func TestIsValidSlug(t *testing.T) {
	validSlugs := []string{
		"tierheim-berlin",
		"abc",
		"my-shelter-123",
		"test123",
	}

	for _, slug := range validSlugs {
		if !isValidSlug(slug) {
			t.Errorf("Expected '%s' to be valid", slug)
		}
	}

	invalidSlugs := []string{
		"ab",           // too short
		"",             // empty
		"UPPERCASE",    // uppercase
		"-starts-dash", // starts with dash
		"ends-dash-",   // ends with dash
		"1starts-num",  // starts with number
		"has space",    // has space
		"has_under",    // has underscore
	}

	for _, slug := range invalidSlugs {
		if isValidSlug(slug) {
			t.Errorf("Expected '%s' to be invalid", slug)
		}
	}
}

// TestIsReservedSlug tests the reserved slug checking function
func TestIsReservedSlug(t *testing.T) {
	reserved := []string{
		"www", "api", "admin", "app", "mail", "email", "smtp", "ftp",
		"support", "help", "billing", "status", "dev", "staging", "test",
		"demo", "blog", "news", "docs", "static", "assets", "cdn", "media",
	}

	for _, slug := range reserved {
		if !isReservedSlug(slug) {
			t.Errorf("Expected '%s' to be reserved", slug)
		}
	}

	notReserved := []string{
		"tierheim-berlin",
		"my-shelter",
		"animal-rescue",
	}

	for _, slug := range notReserved {
		if isReservedSlug(slug) {
			t.Errorf("Expected '%s' to not be reserved", slug)
		}
	}
}

// TestTenantHandler_Register_CreatesSubscription tests that registration creates a Free subscription (TDD RED Phase)
// BUG #2: New tenants should get a tenant_subscriptions record with Free plan
func TestTenantHandler_Register_CreatesSubscription(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		BaseDomain:         "gassigeher.local",
	}
	handler := NewTenantHandler(db, cfg)

	reqBody := map[string]string{
		"organization_name": "Subscription Test Org",
		"slug":              "subscription-test",
		"contact_email":     "test@subscriptiontest.com",
		"city":              "Berlin",
		"postal_code":       "10115",
		"federal_state":     "BE",
		"admin_first_name":  "Admin",
		"admin_last_name":   "Test",
		"admin_email":       "admin@subscriptiontest.com",
		"admin_password":    "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Get the created tenant ID from the response
	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	// Query database to check if subscription was created
	var subscriptionCount int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM tenant_subscriptions ts
		JOIN tenants t ON ts.tenant_id = t.id
		WHERE t.slug = ?
	`, "subscription-test").Scan(&subscriptionCount)

	if err != nil {
		t.Fatalf("Failed to query subscription: %v", err)
	}

	if subscriptionCount != 1 {
		t.Errorf("BUG #2: Expected 1 subscription for new tenant, got %d", subscriptionCount)
	}

	// Verify the subscription is for Free plan (plan_id = 1)
	var planID int
	err = db.QueryRow(`
		SELECT ts.plan_id FROM tenant_subscriptions ts
		JOIN tenants t ON ts.tenant_id = t.id
		WHERE t.slug = ?
	`, "subscription-test").Scan(&planID)

	if err != nil {
		t.Fatalf("Failed to query plan_id: %v", err)
	}

	if planID != 1 {
		t.Errorf("Expected Free plan (plan_id=1), got plan_id=%d", planID)
	}
}

// =============================================================================
// SECURITY TEST: XSS Sanitization in Tenant Registration (TDD RED Phase)
// =============================================================================
// BUG: Organization name with script tags is stored without sanitization
// This could lead to XSS attacks when the name is displayed in the UI
func TestTenantHandler_Register_XSSSanitization(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		BaseDomain:         "gassigeher.org",
	}
	handler := NewTenantHandler(db, cfg)

	t.Run("sanitizes script tags from organization_name", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"organization_name": "<script>alert('XSS')</script>Evil Shelter",
			"slug":              "xss-test-1",
			"contact_email":     "xss-test@example.com",
			"city":              "Test City",
			"postal_code":       "12345",
			"federal_state":     "BW",
			"admin_first_name":  "Admin",
			"admin_last_name":   "User",
			"admin_email":       "admin@xss-test.com",
			"admin_password":    "SecurePass123!",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Register(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Check that script tags are NOT stored in database
		var storedName string
		err := db.QueryRow("SELECT name FROM tenants WHERE slug = ?", "xss-test-1").Scan(&storedName)
		if err != nil {
			t.Fatalf("Failed to query tenant: %v", err)
		}

		// Name should NOT contain script tags
		if containsHTML(storedName) {
			t.Errorf("XSS VULNERABILITY: Script tags stored in organization_name! Got: %s", storedName)
		}

		// Verify the safe content is preserved
		if storedName != "Evil Shelter" && storedName != "alertXSSEvil Shelter" {
			t.Logf("Sanitized name: %s (original contained script tags)", storedName)
		}
	})

	t.Run("sanitizes img onerror from organization_name", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"organization_name": `<img src=x onerror="alert('XSS')">Malicious Shelter`,
			"slug":              "xss-test-2",
			"contact_email":     "xss-test2@example.com",
			"city":              "Test City",
			"postal_code":       "12345",
			"federal_state":     "BW",
			"admin_first_name":  "Admin",
			"admin_last_name":   "User",
			"admin_email":       "admin@xss-test2.com",
			"admin_password":    "SecurePass123!",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Register(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var storedName string
		err := db.QueryRow("SELECT name FROM tenants WHERE slug = ?", "xss-test-2").Scan(&storedName)
		if err != nil {
			t.Fatalf("Failed to query tenant: %v", err)
		}

		if containsHTML(storedName) {
			t.Errorf("XSS VULNERABILITY: HTML stored in organization_name! Got: %s", storedName)
		}
	})

	t.Run("sanitizes event handlers from admin names", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"organization_name": "Normal Shelter",
			"slug":              "xss-test-3",
			"contact_email":     "xss-test3@example.com",
			"city":              "Test City",
			"postal_code":       "12345",
			"federal_state":     "BW",
			"admin_first_name":  `<a href="javascript:alert('XSS')">Click</a>`,
			"admin_last_name":   `<div onmouseover="alert('XSS')">Hover</div>`,
			"admin_email":       "admin@xss-test3.com",
			"admin_password":    "SecurePass123!",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Register(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Check user names don't contain HTML
		var firstName, lastName string
		err := db.QueryRow(`
			SELECT u.first_name, u.last_name FROM users u
			JOIN tenants t ON u.tenant_id = t.id
			WHERE t.slug = ?
		`, "xss-test-3").Scan(&firstName, &lastName)
		if err != nil {
			t.Fatalf("Failed to query user: %v", err)
		}

		if containsHTML(firstName) {
			t.Errorf("XSS VULNERABILITY: HTML stored in admin first_name! Got: %s", firstName)
		}
		if containsHTML(lastName) {
			t.Errorf("XSS VULNERABILITY: HTML stored in admin last_name! Got: %s", lastName)
		}
	})
}

// containsHTML checks if a string contains HTML tags
func containsHTML(s string) bool {
	// Check for common XSS patterns
	patterns := []string{
		"<script", "</script>",
		"<img", "<a ",
		"<div", "</div>",
		"onerror=", "onmouseover=", "onclick=",
		"javascript:",
	}
	for _, p := range patterns {
		if len(s) >= len(p) {
			for i := 0; i <= len(s)-len(p); i++ {
				if s[i:i+len(p)] == p {
					return true
				}
			}
		}
	}
	return false
}

// =============================================================================
// BUG #1: GLOBAL EMAIL UNIQUENESS - Same email across tenants
// =============================================================================
// CRITICAL: The same email can be registered as admin in multiple tenants.
// This creates login ambiguity and potential security issues.
// =============================================================================

// TestTenantHandler_Register_RejectsDuplicateEmailAcrossTenants tests that the same
// email cannot be used to register as admin in multiple tenants
// TDD RED PHASE: This test should FAIL until we implement global email uniqueness
func TestTenantHandler_Register_RejectsDuplicateEmailAcrossTenants(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		BaseDomain:         "gassigeher.org",
	}
	handler := NewTenantHandler(db, cfg)

	// First registration - should succeed
	reqBody1 := map[string]string{
		"organization_name": "First Shelter",
		"slug":              "first-shelter",
		"contact_email":     "contact@first-shelter.de",
		"city":              "Berlin",
		"postal_code":       "10115",
		"federal_state":     "BE",
		"admin_first_name":  "Max",
		"admin_last_name":   "Mustermann",
		"admin_email":       "shared@example.com", // This email will be used twice
		"admin_password":    "SecurePass123!",
	}
	body1, _ := json.Marshal(reqBody1)

	req1 := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")

	rec1 := httptest.NewRecorder()
	handler.Register(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("First registration should succeed, got %d: %s", rec1.Code, rec1.Body.String())
	}

	// Second registration with SAME admin email - should FAIL
	reqBody2 := map[string]string{
		"organization_name": "Second Shelter",
		"slug":              "second-shelter",
		"contact_email":     "contact@second-shelter.de",
		"city":              "Munich",
		"postal_code":       "80331",
		"federal_state":     "BY",
		"admin_first_name":  "Anna",
		"admin_last_name":   "Schmidt",
		"admin_email":       "shared@example.com", // SAME email as first tenant!
		"admin_password":    "AnotherPass456!",
	}
	body2, _ := json.Marshal(reqBody2)

	req2 := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")

	rec2 := httptest.NewRecorder()
	handler.Register(rec2, req2)

	// BUG: Currently returns 201 (success), should return 409 (conflict)
	if rec2.Code == http.StatusCreated {
		t.Errorf("BUG: Same email 'shared@example.com' was allowed to register in TWO different tenants!")
		t.Errorf("This creates login ambiguity and potential security issues.")
	}

	// Expected behavior: reject with 409 Conflict
	if rec2.Code != http.StatusConflict {
		t.Errorf("Expected status 409 Conflict for duplicate email across tenants, got %d: %s",
			rec2.Code, rec2.Body.String())
	}

	// Verify there's only ONE user with this email
	var userCount int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "shared@example.com").Scan(&userCount)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}

	if userCount > 1 {
		t.Errorf("BUG: Found %d users with email 'shared@example.com' - should be only 1!", userCount)
	}
}

// TestTenantHandler_Register_AllowsDifferentEmails tests that different emails can register
func TestTenantHandler_Register_AllowsDifferentEmails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		BaseDomain:         "gassigeher.org",
	}
	handler := NewTenantHandler(db, cfg)

	// First registration
	reqBody1 := map[string]string{
		"organization_name": "Shelter A",
		"slug":              "shelter-a",
		"contact_email":     "contact@shelter-a.de",
		"city":              "Berlin",
		"postal_code":       "10115",
		"federal_state":     "BE",
		"admin_first_name":  "Admin",
		"admin_last_name":   "One",
		"admin_email":       "admin1@unique.com",
		"admin_password":    "SecurePass123!",
	}
	body1, _ := json.Marshal(reqBody1)

	req1 := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")

	rec1 := httptest.NewRecorder()
	handler.Register(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("First registration should succeed, got %d", rec1.Code)
	}

	// Second registration with DIFFERENT email - should succeed
	reqBody2 := map[string]string{
		"organization_name": "Shelter B",
		"slug":              "shelter-b",
		"contact_email":     "contact@shelter-b.de",
		"city":              "Munich",
		"postal_code":       "80331",
		"federal_state":     "BY",
		"admin_first_name":  "Admin",
		"admin_last_name":   "Two",
		"admin_email":       "admin2@unique.com", // Different email
		"admin_password":    "AnotherPass456!",
	}
	body2, _ := json.Marshal(reqBody2)

	req2 := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")

	rec2 := httptest.NewRecorder()
	handler.Register(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Errorf("Second registration with different email should succeed, got %d: %s",
			rec2.Code, rec2.Body.String())
	}
}

// =============================================================================
// BUG #5: Invalid Email Format Accepted in Registration (TDD RED Phase)
// =============================================================================
// The registration endpoint accepts invalid email formats like "notanemail"
// This should be rejected with 400 Bad Request
// =============================================================================

func TestTenantHandler_Register_InvalidEmailFormat_ReturnsError(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		BaseDomain:         "gassigeher.org",
	}
	handler := NewTenantHandler(db, cfg)

	testCases := []struct {
		name        string
		slug        string
		adminEmail  string
		shouldFail  bool
		description string
	}{
		{"missing-at", "email-test-missing-at", "notanemail", true, "Email without @ symbol"},
		{"missing-domain", "email-test-missing-domain", "test@", true, "Email without domain"},
		{"missing-local", "email-test-missing-local", "@example.com", true, "Email without local part"},
		{"double-at", "email-test-double-at", "test@@example.com", true, "Email with double @"},
		{"spaces", "email-test-spaces", "test @example.com", true, "Email with spaces"},
		{"valid-email", "email-test-valid-email", "valid@example.com", false, "Valid email should pass"},
		{"valid-plus", "email-test-valid-plus", "test+tag@example.com", false, "Valid email with plus addressing"},
		{"valid-subdomain", "email-test-valid-subdomain", "test@sub.example.com", false, "Valid email with subdomain"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := map[string]string{
				"organization_name": "Email Test Org " + tc.name,
				"slug":              tc.slug,
				"contact_email":     "contact@example.com",
				"city":              "Berlin",
				"postal_code":       "10115",
				"federal_state":     "BE",
				"admin_first_name":  "Admin",
				"admin_last_name":   "Test",
				"admin_email":       tc.adminEmail,
				"admin_password":    "SecurePass123!",
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.Register(rec, req)

			if tc.shouldFail {
				// BUG: Currently invalid emails return 201 (success)
				// Should return 400 Bad Request
				if rec.Code == http.StatusCreated {
					t.Errorf("BUG #5: Invalid email '%s' was accepted! %s", tc.adminEmail, tc.description)
				}
				if rec.Code != http.StatusBadRequest {
					t.Errorf("Expected status 400 for invalid email '%s', got %d. %s",
						tc.adminEmail, rec.Code, tc.description)
				}
			} else {
				if rec.Code != http.StatusCreated {
					t.Errorf("Valid email '%s' should be accepted, got %d: %s",
						tc.adminEmail, rec.Code, rec.Body.String())
				}
			}
		})
	}
}

// TestTenantHandler_Register_InvalidContactEmailFormat tests contact email validation
func TestTenantHandler_Register_InvalidContactEmailFormat_ReturnsError(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		BaseDomain:         "gassigeher.org",
	}
	handler := NewTenantHandler(db, cfg)

	// Test with invalid contact email
	reqBody := map[string]string{
		"organization_name": "Contact Email Test",
		"slug":              "contact-email-test",
		"contact_email":     "invalid-contact-email", // Invalid!
		"city":              "Berlin",
		"postal_code":       "10115",
		"federal_state":     "BE",
		"admin_first_name":  "Admin",
		"admin_last_name":   "Test",
		"admin_email":       "valid-admin@example.com",
		"admin_password":    "SecurePass123!",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/tenants/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.Register(rec, req)

	// BUG: Currently invalid contact emails return 201 (success)
	if rec.Code == http.StatusCreated {
		t.Errorf("BUG #5: Invalid contact email was accepted!")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid contact email, got %d", rec.Code)
	}
}

// TestTenantHandler_CheckSlug_RateLimiting tests rate limiting on slug enumeration
// TDD RED PHASE: This test should FAIL until we add rate limiting to CheckSlug
// BUG: Without rate limiting, attackers can enumerate all tenant slugs
func TestTenantHandler_CheckSlug_RateLimiting(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewTenantHandler(db, cfg)

	t.Run("SECURITY: rate limits excessive slug checks from same IP", func(t *testing.T) {
		// Simulate an attacker trying to enumerate tenant slugs
		// After 10 requests in quick succession, should get rate limited

		var rateLimitedCount int
		var successCount int

		// Make 15 requests - should get rate limited after ~10
		for i := 0; i < 15; i++ {
			slug := "test-slug-" + string(rune('a'+i))
			req := httptest.NewRequest("GET", "/api/tenants/check-slug?slug="+slug, nil)
			req.RemoteAddr = "192.168.1.100:12345" // Same IP for all requests

			rec := httptest.NewRecorder()
			handler.CheckSlug(rec, req)

			if rec.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			} else if rec.Code == http.StatusOK {
				successCount++
			}
		}

		// SECURITY FIX: Should rate limit after ~10 requests
		if rateLimitedCount == 0 {
			t.Errorf("SECURITY BUG: CheckSlug endpoint has no rate limiting! "+
				"Attackers can enumerate all tenant slugs. "+
				"Got %d successes, %d rate limited (expected some rate limiting)",
				successCount, rateLimitedCount)
		}

		t.Logf("Rate limiting test: %d successful, %d rate limited", successCount, rateLimitedCount)
	})

	t.Run("SECURITY: different IPs are not rate limited together", func(t *testing.T) {
		// Different IPs should have separate rate limit counters
		// Create a fresh handler for this test
		handler2 := NewTenantHandler(db, cfg)

		// Make 5 requests from IP A
		for i := 0; i < 5; i++ {
			slug := "test-a-" + string(rune('a'+i))
			req := httptest.NewRequest("GET", "/api/tenants/check-slug?slug="+slug, nil)
			req.RemoteAddr = "10.0.0.1:12345"

			rec := httptest.NewRecorder()
			handler2.CheckSlug(rec, req)
		}

		// Request from different IP should still work
		req := httptest.NewRequest("GET", "/api/tenants/check-slug?slug=different-ip-test", nil)
		req.RemoteAddr = "10.0.0.2:54321" // Different IP

		rec := httptest.NewRecorder()
		handler2.CheckSlug(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Different IP should not be rate limited, got status %d", rec.Code)
		}
	})
}
