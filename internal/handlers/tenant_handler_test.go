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
