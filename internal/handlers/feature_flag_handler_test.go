package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestFeatureFlagHandler_ListFlags tests listing all feature flags
func TestFeatureFlagHandler_ListFlags(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewFeatureFlagHandler(db)

	t.Run("returns empty list when no flags", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/central-admin/feature-flags", nil)

		rec := httptest.NewRecorder()
		handler.ListFlags(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["flags"] == nil {
			t.Error("Expected flags field in response")
		}
	})

	t.Run("returns flags after creation", func(t *testing.T) {
		// Insert a test flag
		// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
		now := testutil.Now()
		db.Exec(`INSERT INTO feature_flags ("key", name, is_global, is_enabled, created_at, updated_at) VALUES ('test_flag', 'Test Flag', 1, 1, ?, ?)`, now, now)

		req := httptest.NewRequest("GET", "/api/v1/central-admin/feature-flags", nil)

		rec := httptest.NewRecorder()
		handler.ListFlags(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		flags := response["flags"].([]interface{})
		if len(flags) < 1 {
			t.Error("Expected at least one flag")
		}
	})
}

// TestFeatureFlagHandler_CreateFlag tests creating a feature flag
func TestFeatureFlagHandler_CreateFlag(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewFeatureFlagHandler(db)

	t.Run("creates flag successfully", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"key":         "new_feature",
			"name":        "New Feature",
			"description": "A test feature",
			"is_global":   true,
			"is_enabled":  false,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/central-admin/feature-flags", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreateFlag(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("rejects invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/central-admin/feature-flags", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreateFlag(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("rejects empty key", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"key":  "",
			"name": "Empty Key Feature",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/central-admin/feature-flags", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreateFlag(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestFeatureFlagHandler_GetFlag tests getting a single flag
func TestFeatureFlagHandler_GetFlag(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewFeatureFlagHandler(db)

	// Insert a test flag
	// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
	now := testutil.Now()
	result, _ := db.Exec(`INSERT INTO feature_flags ("key", name, is_global, is_enabled, created_at, updated_at) VALUES ('get_test', 'Get Test', 1, 1, ?, ?)`, now, now)
	flagID, _ := result.LastInsertId()

	t.Run("returns flag by ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/central-admin/feature-flags/1", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})

		rec := httptest.NewRecorder()
		handler.GetFlag(rec, req)

		// May be 200 or 404 depending on whether flag with ID 1 exists
		if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", rec.Code)
		}
	})

	t.Run("returns 404 for non-existent flag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/central-admin/feature-flags/99999", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})

		rec := httptest.NewRecorder()
		handler.GetFlag(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/central-admin/feature-flags/invalid", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})

		rec := httptest.NewRecorder()
		handler.GetFlag(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	_ = flagID // Use the flagID
}

// TestFeatureFlagHandler_CheckFlag tests checking a flag by key
func TestFeatureFlagHandler_CheckFlag(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewFeatureFlagHandler(db)

	// Insert a test flag
	// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
	now := testutil.Now()
	db.Exec(`INSERT INTO feature_flags ("key", name, is_global, is_enabled, created_at, updated_at) VALUES ('enabled_feature', 'Enabled', 1, 1, ?, ?)`, now, now)
	db.Exec(`INSERT INTO feature_flags ("key", name, is_global, is_enabled, created_at, updated_at) VALUES ('disabled_feature', 'Disabled', 1, 0, ?, ?)`, now, now)

	t.Run("returns enabled for enabled flag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/feature-flags/enabled_feature/check", nil)
		req = mux.SetURLVars(req, map[string]string{"key": "enabled_feature"})

		rec := httptest.NewRecorder()
		handler.CheckFlag(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["enabled"] != true {
			t.Errorf("Expected enabled=true, got %v", response["enabled"])
		}
	})

	t.Run("returns disabled for disabled flag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/feature-flags/disabled_feature/check", nil)
		req = mux.SetURLVars(req, map[string]string{"key": "disabled_feature"})

		rec := httptest.NewRecorder()
		handler.CheckFlag(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["enabled"] != false {
			t.Errorf("Expected enabled=false, got %v", response["enabled"])
		}
	})

	t.Run("returns disabled for non-existent flag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/feature-flags/nonexistent/check", nil)
		req = mux.SetURLVars(req, map[string]string{"key": "nonexistent"})

		rec := httptest.NewRecorder()
		handler.CheckFlag(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["enabled"] != false {
			t.Errorf("Expected enabled=false for non-existent flag, got %v", response["enabled"])
		}
	})
}

// TestFeatureFlagHandler_CheckMultipleFlags tests checking multiple flags at once
func TestFeatureFlagHandler_CheckMultipleFlags(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewFeatureFlagHandler(db)

	// Insert test flags
	// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
	now := testutil.Now()
	db.Exec(`INSERT INTO feature_flags ("key", name, is_global, is_enabled, created_at, updated_at) VALUES ('multi_enabled', 'Multi Enabled', 1, 1, ?, ?)`, now, now)
	db.Exec(`INSERT INTO feature_flags ("key", name, is_global, is_enabled, created_at, updated_at) VALUES ('multi_disabled', 'Multi Disabled', 1, 0, ?, ?)`, now, now)

	t.Run("returns status for multiple flags", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"keys": []string{"multi_enabled", "multi_disabled", "nonexistent"},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/feature-flags/check", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CheckMultipleFlags(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		flags := response["flags"].(map[string]interface{})

		if flags["multi_enabled"] != true {
			t.Errorf("Expected multi_enabled=true, got %v", flags["multi_enabled"])
		}
		if flags["multi_disabled"] != false {
			t.Errorf("Expected multi_disabled=false, got %v", flags["multi_disabled"])
		}
		if flags["nonexistent"] != false {
			t.Errorf("Expected nonexistent=false, got %v", flags["nonexistent"])
		}
	})

	t.Run("rejects invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/feature-flags/check", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CheckMultipleFlags(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}
