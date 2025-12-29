package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestThemeHandler_GetCSS tests getting CSS variables
func TestThemeHandler_GetCSS(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewThemeHandler(db)

	t.Run("returns default classic theme CSS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/theme/css", nil)
		ctx := contextWithTenant(req.Context(), 1, 0, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetCSS(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Check content type
		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/css") {
			t.Errorf("Expected Content-Type to contain 'text/css', got %s", contentType)
		}

		// Check CSS contains root variables
		body := rec.Body.String()
		if !strings.Contains(body, ":root") {
			t.Error("Expected CSS to contain :root")
		}
		if !strings.Contains(body, "--color-primary:") {
			t.Error("Expected CSS to contain --color-primary")
		}
	})

	t.Run("returns CSS without tenant context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/theme/css", nil)
		// No tenant context - should still return default

		rec := httptest.NewRecorder()
		handler.GetCSS(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 (default theme), got %d", rec.Code)
		}
	})
}

// TestThemeHandler_GetCSS_CustomColors tests getting CSS with custom colors
func TestThemeHandler_GetCSS_CustomColors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewThemeHandler(db)

	// First, ensure tenant_settings exists for tenant 1 (should be created by provisioning or manually)
	// Use INSERT OR REPLACE to ensure the row exists
	_, err := db.Exec(`
		INSERT OR REPLACE INTO tenant_settings (tenant_id, color_primary, color_secondary, color_accent, color_background, color_text, theme_preset)
		VALUES (1, '#ff0000', '#00ff00', '#0000ff', '#ffffff', '#000000', '')
	`)
	if err != nil {
		t.Fatalf("Failed to set custom colors: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/theme/css", nil)
	ctx := contextWithTenant(req.Context(), 1, 0, false)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.GetCSS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "#ff0000") {
		t.Errorf("Expected CSS to contain custom primary color #ff0000. Got: %s", body)
	}
}

// TestThemeHandler_GetPresets tests getting all available presets
func TestThemeHandler_GetPresets(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewThemeHandler(db)

	req := httptest.NewRequest("GET", "/api/theme/presets", nil)

	rec := httptest.NewRecorder()
	handler.GetPresets(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var presets []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &presets); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should have multiple presets
	if len(presets) < 3 {
		t.Errorf("Expected at least 3 presets, got %d", len(presets))
	}

	// Check that each preset has required fields
	for _, preset := range presets {
		if preset["name"] == nil {
			t.Error("Preset should have 'name' field")
		}
		if preset["colors"] == nil {
			t.Error("Preset should have 'colors' field")
		}
	}
}

// TestThemeHandler_GetCurrentTheme tests getting current tenant theme
func TestThemeHandler_GetCurrentTheme(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewThemeHandler(db)

	t.Run("returns default theme for new tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/theme", nil)
		ctx := contextWithTenant(req.Context(), 1, 0, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetCurrentTheme(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["preset"] == nil {
			t.Error("Expected 'preset' field in response")
		}
		if response["colors"] == nil {
			t.Error("Expected 'colors' field in response")
		}
	})

	t.Run("returns custom colors when set", func(t *testing.T) {
		// Set custom colors
		db.Exec(`
			INSERT OR REPLACE INTO tenant_settings (tenant_id, color_primary, color_secondary, color_accent, color_background, color_text, theme_preset)
			VALUES (1, '#custom1', '#custom2', '#custom3', '#custom4', '#custom5', '')
		`)

		req := httptest.NewRequest("GET", "/api/theme", nil)
		ctx := contextWithTenant(req.Context(), 1, 0, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetCurrentTheme(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["preset"] != "custom" {
			t.Errorf("Expected preset 'custom', got %v", response["preset"])
		}
	})
}

// TestThemeHandler_UpdateTheme tests updating tenant theme
func TestThemeHandler_UpdateTheme(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewThemeHandler(db)

	// Create admin user
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin User", "blue")
	db.Exec("UPDATE users SET is_admin = 1 WHERE id = ?", adminID)

	t.Run("updates to preset theme", func(t *testing.T) {
		reqBody := map[string]string{
			"preset": "forest", // Valid preset name
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/theme", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 0, adminID, true) // tenant_id=0 to match SeedTestUser and DB queries
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateTheme(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify in database
		var preset string
		db.QueryRow("SELECT theme_preset FROM tenant_settings WHERE tenant_id = 0").Scan(&preset)
		if preset != "forest" {
			t.Errorf("Expected theme_preset 'forest', got %s", preset)
		}
	})

	t.Run("updates to custom colors", func(t *testing.T) {
		reqBody := map[string]string{
			"preset":     "custom",
			"primary":    "#aabbcc",
			"secondary":  "#ddeeff",
			"accent":     "#112233",
			"background": "#ffffff",
			"text":       "#000000",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/theme", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 0, adminID, true) // tenant_id=0 to match SeedTestUser and DB queries
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateTheme(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify in database
		var primary string
		db.QueryRow("SELECT color_primary FROM tenant_settings WHERE tenant_id = 0").Scan(&primary)
		if primary != "#aabbcc" {
			t.Errorf("Expected color_primary '#aabbcc', got %s", primary)
		}
	})

	t.Run("returns 400 for invalid preset", func(t *testing.T) {
		reqBody := map[string]string{
			"preset": "nonexistent",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/theme", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 0, adminID, true) // tenant_id=0 to match SeedTestUser
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateTheme(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for custom without all colors", func(t *testing.T) {
		reqBody := map[string]string{
			"preset":  "custom",
			"primary": "#aabbcc",
			// Missing other colors
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/theme", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 0, adminID, true) // tenant_id=0 to match SeedTestUser
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateTheme(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/theme", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 0, adminID, true) // tenant_id=0 to match SeedTestUser
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateTheme(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestSafeDeref tests the safeDeref helper function
func TestSafeDeref(t *testing.T) {
	t.Run("returns string value when not nil", func(t *testing.T) {
		value := "test"
		result := safeDeref(&value)
		if result != "test" {
			t.Errorf("Expected 'test', got %s", result)
		}
	})

	t.Run("returns empty string when nil", func(t *testing.T) {
		var value *string = nil
		result := safeDeref(value)
		if result != "" {
			t.Errorf("Expected empty string, got %s", result)
		}
	})
}
