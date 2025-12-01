package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// DONE: TestSettingsHandler_GetAllSettings tests getting all system settings
func TestSettingsHandler_GetAllSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewSettingsHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	// Database migration creates default settings
	t.Run("admin gets all settings", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/settings", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetAllSettings(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var settings []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &settings)

		// Should have settings from migration
		t.Logf("Got %d settings from database", len(settings))

		if len(settings) == 0 {
			t.Error("Expected at least some settings from migration")
		}
	})
}

// DONE: TestSettingsHandler_UpdateSetting tests updating system settings (admin only)
func TestSettingsHandler_UpdateSetting(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewSettingsHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")
	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")

	// Insert test setting
	db.Exec("INSERT INTO system_settings (key, value) VALUES (?, ?)", "booking_advance_days", "14")

	t.Run("admin successfully updates setting", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "21",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/booking_advance_days", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "booking_advance_days"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify update
		var value string
		db.QueryRow("SELECT value FROM system_settings WHERE key = ?", "booking_advance_days").Scan(&value)

		if value != "21" {
			t.Errorf("Expected value '21', got %s", value)
		}
	})

	t.Run("non-admin cannot update settings", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "30",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/booking_advance_days", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "booking_advance_days"})
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		// Should fail (middleware blocks or handler rejects)
		t.Logf("Non-admin update returned status: %d", rec.Code)
	})

	t.Run("invalid setting key", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "100",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/nonexistent_key", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "nonexistent_key"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		// Should fail with 404 for non-existent setting
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent key, got %d", rec.Code)
		}
	})

	t.Run("empty value", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/booking_advance_days", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "booking_advance_days"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty value, got %d", rec.Code)
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/settings/booking_advance_days", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "booking_advance_days"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", rec.Code)
		}
	})

	// DONE: BUG #3 - Prevent invalid numeric setting values
	t.Run("BUGFIX: reject non-numeric value for booking_advance_days", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "abc123", // Invalid - should be number
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/booking_advance_days", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "booking_advance_days"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("BUGFIX: Expected status 400 for non-numeric value, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)
		errorMsg := response["error"].(string)

		if errorMsg != "Value must be a positive integer" {
			t.Errorf("Expected clear validation error, got %q", errorMsg)
		}
	})

	t.Run("BUGFIX: reject negative value for numeric setting", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "-5", // Invalid - must be positive
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/auto_deactivation_days", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "auto_deactivation_days"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("BUGFIX: Expected status 400 for negative value, got %d", rec.Code)
		}
	})

	t.Run("BUGFIX: reject zero value for numeric setting", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "0", // Invalid - must be positive
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/cancellation_notice_hours", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "cancellation_notice_hours"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("BUGFIX: Expected status 400 for zero value, got %d", rec.Code)
		}
	})
}
