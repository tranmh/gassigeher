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
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestTenantHandler_GetBranding_OrganizationFields tests that GetBranding returns org fields
func TestTenantHandler_GetBranding_OrganizationFields(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		BaseDomain:         "gassigeher.org",
		Port:               "8080",
	}
	handler := NewTenantHandler(db, cfg)
	tenantRepo := repository.NewTenantRepository(db)

	// Create a test tenant with settings
	tenant := &models.Tenant{
		Slug:         "branding-org-test",
		Name:         "Branding Org Test",
		Status:       models.TenantStatusActive,
		ContactEmail: "branding@test.com",
		FederalState: "BW",
	}
	if err := tenantRepo.Create(tenant); err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	orgName := "Tierschutzverein Teststadt e.V."
	orgAddress := "Teststraße 1, 12345 Teststadt"
	orgEmail := "info@tierheim-teststadt.de"
	orgPhone := "01234 56789"
	privacyEmail := "datenschutz@tierheim-teststadt.de"

	settings := &models.TenantSettings{
		TenantID:            tenant.ID,
		ThemePreset:         "classic",
		OrganizationName:    &orgName,
		OrganizationAddress: &orgAddress,
		OrganizationEmail:   &orgEmail,
		OrganizationPhone:   &orgPhone,
		PrivacyOfficerEmail: &privacyEmail,
	}
	if err := tenantRepo.CreateSettings(settings); err != nil {
		t.Fatalf("Failed to create settings: %v", err)
	}

	t.Run("returns organization fields in branding response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tenant/branding", nil)
		ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenant.ID)
		ctx = context.WithValue(ctx, middleware.TenantSlugKey, "branding-org-test")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetBranding(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["tenant_name"] != "Branding Org Test" {
			t.Errorf("Expected tenant_name 'Branding Org Test', got %v", response["tenant_name"])
		}
		if response["organization_name"] != orgName {
			t.Errorf("Expected organization_name %q, got %v", orgName, response["organization_name"])
		}
		if response["organization_address"] != orgAddress {
			t.Errorf("Expected organization_address %q, got %v", orgAddress, response["organization_address"])
		}
		if response["organization_email"] != orgEmail {
			t.Errorf("Expected organization_email %q, got %v", orgEmail, response["organization_email"])
		}
		if response["organization_phone"] != orgPhone {
			t.Errorf("Expected organization_phone %q, got %v", orgPhone, response["organization_phone"])
		}
		if response["privacy_officer_email"] != privacyEmail {
			t.Errorf("Expected privacy_officer_email %q, got %v", privacyEmail, response["privacy_officer_email"])
		}
	})

	t.Run("omits null organization fields from response", func(t *testing.T) {
		// Create another tenant with no org fields
		tenant2 := &models.Tenant{
			Slug:         "branding-no-org",
			Name:         "No Org Fields",
			Status:       models.TenantStatusActive,
			ContactEmail: "noorg@test.com",
			FederalState: "BW",
		}
		if err := tenantRepo.Create(tenant2); err != nil {
			t.Fatalf("Failed to create tenant2: %v", err)
		}
		settings2 := &models.TenantSettings{
			TenantID:    tenant2.ID,
			ThemePreset: "classic",
		}
		if err := tenantRepo.CreateSettings(settings2); err != nil {
			t.Fatalf("Failed to create settings2: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/v1/tenant/branding", nil)
		ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenant2.ID)
		ctx = context.WithValue(ctx, middleware.TenantSlugKey, "branding-no-org")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetBranding(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Organization fields should be omitted (not present in JSON) when nil
		if _, ok := response["organization_name"]; ok {
			t.Error("Expected organization_name to be omitted when nil")
		}
		if _, ok := response["organization_address"]; ok {
			t.Error("Expected organization_address to be omitted when nil")
		}
		if _, ok := response["privacy_officer_email"]; ok {
			t.Error("Expected privacy_officer_email to be omitted when nil")
		}
	})
}

// TestTenantHandler_UpdateBranding_OrganizationFields tests updating org fields via branding API
func TestTenantHandler_UpdateBranding_OrganizationFields(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		BaseDomain:         "gassigeher.org",
		Port:               "8080",
	}
	handler := NewTenantHandler(db, cfg)
	tenantRepo := repository.NewTenantRepository(db)

	// Create a test tenant
	tenant := &models.Tenant{
		Slug:         "update-branding-org",
		Name:         "Update Branding Org",
		Status:       models.TenantStatusActive,
		ContactEmail: "update-org@test.com",
		FederalState: "BW",
	}
	if err := tenantRepo.Create(tenant); err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	settings := &models.TenantSettings{
		TenantID:    tenant.ID,
		ThemePreset: "classic",
	}
	if err := tenantRepo.CreateSettings(settings); err != nil {
		t.Fatalf("Failed to create settings: %v", err)
	}

	t.Run("saves organization fields via update branding", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"theme_preset":        "classic",
			"organization_name":    "Neuer Tierschutzverein e.V.",
			"organization_address": "Hauptstr. 10, 70000 Stuttgart",
			"organization_email":   "info@neuer-verein.de",
			"organization_phone":   "0711 11111",
			"privacy_officer_email": "dsgvo@neuer-verein.de",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/admin/tenant/branding", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), tenant.ID, 1, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateBranding(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify via GetBranding
		getReq := httptest.NewRequest("GET", "/api/v1/tenant/branding", nil)
		getCtx := context.WithValue(getReq.Context(), middleware.TenantIDKey, tenant.ID)
		getCtx = context.WithValue(getCtx, middleware.TenantSlugKey, "update-branding-org")
		getReq = getReq.WithContext(getCtx)

		getRec := httptest.NewRecorder()
		handler.GetBranding(getRec, getReq)

		var response map[string]interface{}
		json.Unmarshal(getRec.Body.Bytes(), &response)

		if response["organization_name"] != "Neuer Tierschutzverein e.V." {
			t.Errorf("Expected organization_name 'Neuer Tierschutzverein e.V.', got %v", response["organization_name"])
		}
		if response["organization_address"] != "Hauptstr. 10, 70000 Stuttgart" {
			t.Errorf("Expected organization_address, got %v", response["organization_address"])
		}
		if response["organization_email"] != "info@neuer-verein.de" {
			t.Errorf("Expected organization_email, got %v", response["organization_email"])
		}
		if response["organization_phone"] != "0711 11111" {
			t.Errorf("Expected organization_phone, got %v", response["organization_phone"])
		}
		if response["privacy_officer_email"] != "dsgvo@neuer-verein.de" {
			t.Errorf("Expected privacy_officer_email, got %v", response["privacy_officer_email"])
		}
	})

	t.Run("clears organization fields when set to null", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"theme_preset":        "classic",
			"organization_name":    nil,
			"organization_email":   nil,
			"privacy_officer_email": nil,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/admin/tenant/branding", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), tenant.ID, 1, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateBranding(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify fields were cleared
		getReq := httptest.NewRequest("GET", "/api/v1/tenant/branding", nil)
		getCtx := context.WithValue(getReq.Context(), middleware.TenantIDKey, tenant.ID)
		getCtx = context.WithValue(getCtx, middleware.TenantSlugKey, "update-branding-org")
		getReq = getReq.WithContext(getCtx)

		getRec := httptest.NewRecorder()
		handler.GetBranding(getRec, getReq)

		var response map[string]interface{}
		json.Unmarshal(getRec.Body.Bytes(), &response)

		if _, ok := response["organization_name"]; ok {
			t.Error("Expected organization_name to be omitted after clearing")
		}
		if _, ok := response["organization_email"]; ok {
			t.Error("Expected organization_email to be omitted after clearing")
		}
	})
}

// TestTenantHandler_GetBranding_SimpleModeDefaultTenant tests branding with tenant_id=0
func TestTenantHandler_GetBranding_SimpleModeDefaultTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		Port:               "8080",
		// No BaseDomain = Simple-Mode
	}
	handler := NewTenantHandler(db, cfg)

	t.Run("returns branding for default tenant (id=0) in Simple-Mode", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tenant/branding", nil)
		// Simulate SimpleModeMiddleware: tenant_id=0, slug="default"
		ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 0)
		ctx = context.WithValue(ctx, middleware.TenantSlugKey, "default")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetBranding(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Default tenant should have a name
		if response["tenant_name"] == nil || response["tenant_name"] == "" {
			t.Error("Expected tenant_name to be set for default tenant")
		}

		// theme_preset should default to classic
		if response["theme_preset"] != "classic" {
			t.Errorf("Expected theme_preset 'classic', got %v", response["theme_preset"])
		}
	})
}
