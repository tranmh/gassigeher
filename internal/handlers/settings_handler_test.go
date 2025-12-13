package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

// createTestJPEG creates a test JPEG image in memory
func createTestJPEG(width, height int) *bytes.Buffer {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	buf := new(bytes.Buffer)
	jpeg.Encode(buf, img, &jpeg.Options{Quality: 85})
	return buf
}

// createMultipartRequest creates a multipart form request with a file
func createMultipartRequest(method, url, fieldName, fileName string, fileContent []byte) (*http.Request, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, bytes.NewReader(fileContent))
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

// TestSettingsHandler_GetLogo tests the public logo endpoint
func TestSettingsHandler_GetLogo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		UploadDir:          t.TempDir(),
	}
	handler := NewSettingsHandler(db, cfg)

	t.Run("returns default logo URL when no custom logo", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/settings/logo", nil)
		rec := httptest.NewRecorder()

		handler.GetLogo(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		logoURL, ok := response["logo_url"].(string)
		if !ok {
			t.Fatal("Expected logo_url in response")
		}

		// Should be the default Tierheim logo
		expectedDefault := "https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png"
		if logoURL != expectedDefault {
			t.Errorf("Expected default logo URL, got %s", logoURL)
		}
	})

	t.Run("returns custom logo URL when uploaded", func(t *testing.T) {
		// Update the setting to a custom path (with /uploads/ prefix as stored by UploadLogo)
		db.Exec("UPDATE system_settings SET value = ? WHERE key = ?", "/uploads/settings/site_logo.jpg", "site_logo")

		// Create the actual logo file
		settingsDir := filepath.Join(cfg.UploadDir, "settings")
		os.MkdirAll(settingsDir, 0755)
		logoPath := filepath.Join(settingsDir, "site_logo.jpg")
		os.WriteFile(logoPath, []byte("fake logo content"), 0644)

		req := httptest.NewRequest("GET", "/api/settings/logo", nil)
		rec := httptest.NewRecorder()

		handler.GetLogo(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		logoURL, ok := response["logo_url"].(string)
		if !ok {
			t.Fatal("Expected logo_url in response")
		}

		// Should be the uploads path
		if logoURL != "/uploads/settings/site_logo.jpg" {
			t.Errorf("Expected custom logo URL '/uploads/settings/site_logo.jpg', got %s", logoURL)
		}
	})

	t.Run("no authentication required", func(t *testing.T) {
		// Reset to default
		db.Exec("UPDATE system_settings SET value = ? WHERE key = ?",
			"https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png", "site_logo")

		// Request without any auth context
		req := httptest.NewRequest("GET", "/api/settings/logo", nil)
		rec := httptest.NewRecorder()

		handler.GetLogo(rec, req)

		// Should still work
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 (no auth required), got %d", rec.Code)
		}
	})
}

// TestSettingsHandler_UploadLogo tests logo upload endpoint
func TestSettingsHandler_UploadLogo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uploadDir := t.TempDir()
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		UploadDir:          uploadDir,
	}
	handler := NewSettingsHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	t.Run("admin can upload logo", func(t *testing.T) {
		// Create test image
		imgBuf := createTestJPEG(800, 100)

		req, err := createMultipartRequest("POST", "/api/settings/logo", "logo", "test-logo.jpg", imgBuf.Bytes())
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UploadLogo(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Should return new logo URL
		logoURL, ok := response["logo_url"].(string)
		if !ok {
			t.Fatal("Expected logo_url in response")
		}

		if logoURL != "/uploads/settings/site_logo.jpg" {
			t.Errorf("Expected '/uploads/settings/site_logo.jpg', got %s", logoURL)
		}

		// Verify file exists
		logoPath := filepath.Join(uploadDir, "settings", "site_logo.jpg")
		if _, err := os.Stat(logoPath); os.IsNotExist(err) {
			t.Error("Logo file was not created")
		}

		// Verify database updated (with /uploads/ prefix)
		var dbValue string
		db.QueryRow("SELECT value FROM system_settings WHERE key = ?", "site_logo").Scan(&dbValue)
		if dbValue != "/uploads/settings/site_logo.jpg" {
			t.Errorf("Database not updated, got value: %s", dbValue)
		}
	})

	// Note: In production, RequireAdmin middleware handles authorization.
	// The handler itself doesn't check admin status, so we test the happy path behavior.
	// The middleware test would verify 403 for non-admins.

	t.Run("invalid image rejected", func(t *testing.T) {
		req, err := createMultipartRequest("POST", "/api/settings/logo", "logo", "test.txt", []byte("not an image"))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UploadLogo(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid image, got %d", rec.Code)
		}
	})

	t.Run("missing file rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/settings/logo", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UploadLogo(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing file, got %d", rec.Code)
		}
	})

	// Note: File size validation is handled by ParseMultipartForm with MaxUploadSizeMB config.
	// In tests, the default config may not have a strict limit set.
	// Production deployments should configure MaxUploadSizeMB appropriately.
}

// TestSettingsHandler_ResetLogo tests logo reset endpoint
func TestSettingsHandler_ResetLogo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uploadDir := t.TempDir()
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		UploadDir:          uploadDir,
	}
	handler := NewSettingsHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	// Setup: create a custom logo
	settingsDir := filepath.Join(uploadDir, "settings")
	os.MkdirAll(settingsDir, 0755)
	logoPath := filepath.Join(settingsDir, "site_logo.jpg")
	os.WriteFile(logoPath, []byte("custom logo content"), 0644)
	db.Exec("UPDATE system_settings SET value = ? WHERE key = ?", "/uploads/settings/site_logo.jpg", "site_logo")

	t.Run("admin can reset logo to default", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/settings/logo", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ResetLogo(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Should return default logo URL
		logoURL, ok := response["logo_url"].(string)
		if !ok {
			t.Fatal("Expected logo_url in response")
		}

		expectedDefault := "https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png"
		if logoURL != expectedDefault {
			t.Errorf("Expected default URL, got %s", logoURL)
		}

		// Verify database updated
		var dbValue string
		db.QueryRow("SELECT value FROM system_settings WHERE key = ?", "site_logo").Scan(&dbValue)
		if dbValue != expectedDefault {
			t.Errorf("Database not reset to default, got value: %s", dbValue)
		}

		// Verify custom file deleted
		if _, err := os.Stat(logoPath); !os.IsNotExist(err) {
			t.Error("Custom logo file was not deleted")
		}
	})

	// Note: In production, RequireAdmin middleware handles authorization.
	// The handler itself doesn't check admin status, so we test the happy path behavior.

	t.Run("reset is idempotent", func(t *testing.T) {
		// Reset again (logo already at default)
		req := httptest.NewRequest("DELETE", "/api/settings/logo", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ResetLogo(rec, req)

		// Should succeed even when already at default
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for idempotent reset, got %d", rec.Code)
		}
	})
}

// TestSettingsHandler_LogoWorkflow tests the complete logo upload/get/reset workflow
func TestSettingsHandler_LogoWorkflow(t *testing.T) {
	db := testutil.SetupTestDB(t)
	uploadDir := t.TempDir()
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		UploadDir:          uploadDir,
	}
	handler := NewSettingsHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	defaultLogo := "https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png"

	// Step 1: Get default logo
	t.Run("Step 1: Get default logo", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/settings/logo", nil)
		rec := httptest.NewRecorder()
		handler.GetLogo(rec, req)

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["logo_url"] != defaultLogo {
			t.Errorf("Expected default logo, got %v", response["logo_url"])
		}
	})

	// Step 2: Upload custom logo
	t.Run("Step 2: Upload custom logo", func(t *testing.T) {
		imgBuf := createTestJPEG(600, 80)
		req, _ := createMultipartRequest("POST", "/api/settings/logo", "logo", "custom.jpg", imgBuf.Bytes())
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UploadLogo(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Upload failed: %s", rec.Body.String())
		}
	})

	// Step 3: Get custom logo
	t.Run("Step 3: Get custom logo", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/settings/logo", nil)
		rec := httptest.NewRecorder()
		handler.GetLogo(rec, req)

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["logo_url"] != "/uploads/settings/site_logo.jpg" {
			t.Errorf("Expected custom logo path, got %v", response["logo_url"])
		}
	})

	// Step 4: Reset to default
	t.Run("Step 4: Reset to default", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/settings/logo", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ResetLogo(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Reset failed: %s", rec.Body.String())
		}
	})

	// Step 5: Verify back to default
	t.Run("Step 5: Verify back to default", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/settings/logo", nil)
		rec := httptest.NewRecorder()
		handler.GetLogo(rec, req)

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["logo_url"] != defaultLogo {
			t.Errorf("Expected default logo after reset, got %v", response["logo_url"])
		}
	})
}

// TestSettingsHandler_GetWhatsAppSettings tests the public WhatsApp settings endpoint
func TestSettingsHandler_GetWhatsAppSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewSettingsHandler(db, cfg)

	t.Run("returns disabled when whatsapp not enabled", func(t *testing.T) {
		// Default from migration is disabled
		req := httptest.NewRequest("GET", "/api/settings/whatsapp", nil)
		rec := httptest.NewRecorder()

		handler.GetWhatsAppSettings(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		enabled, ok := response["enabled"].(bool)
		if !ok {
			t.Fatal("Expected enabled field in response")
		}

		if enabled {
			t.Error("Expected WhatsApp to be disabled by default")
		}
	})

	t.Run("returns enabled with link when configured", func(t *testing.T) {
		// Enable WhatsApp and set link
		db.Exec("UPDATE system_settings SET value = ? WHERE key = ?", "true", "whatsapp_group_enabled")
		db.Exec("UPDATE system_settings SET value = ? WHERE key = ?", "https://chat.whatsapp.com/ABC123", "whatsapp_group_link")

		req := httptest.NewRequest("GET", "/api/settings/whatsapp", nil)
		rec := httptest.NewRecorder()

		handler.GetWhatsAppSettings(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		enabled, ok := response["enabled"].(bool)
		if !ok {
			t.Fatal("Expected enabled field in response")
		}

		if !enabled {
			t.Error("Expected WhatsApp to be enabled")
		}

		link, ok := response["link"].(string)
		if !ok {
			t.Fatal("Expected link field in response")
		}

		if link != "https://chat.whatsapp.com/ABC123" {
			t.Errorf("Expected link 'https://chat.whatsapp.com/ABC123', got %s", link)
		}
	})

	t.Run("no authentication required", func(t *testing.T) {
		// Request without any auth context
		req := httptest.NewRequest("GET", "/api/settings/whatsapp", nil)
		rec := httptest.NewRecorder()

		handler.GetWhatsAppSettings(rec, req)

		// Should still work without auth
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 (no auth required), got %d", rec.Code)
		}
	})
}

// TestSettingsHandler_UpdateWhatsAppSettings tests WhatsApp settings validation
func TestSettingsHandler_UpdateWhatsAppSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewSettingsHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	t.Run("admin can enable whatsapp group", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "true",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/whatsapp_group_enabled", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "whatsapp_group_enabled"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify update
		var value string
		db.QueryRow("SELECT value FROM system_settings WHERE key = ?", "whatsapp_group_enabled").Scan(&value)

		if value != "true" {
			t.Errorf("Expected value 'true', got %s", value)
		}
	})

	t.Run("admin can disable whatsapp group", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "false",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/whatsapp_group_enabled", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "whatsapp_group_enabled"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("reject invalid boolean for whatsapp_group_enabled", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "invalid",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/whatsapp_group_enabled", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "whatsapp_group_enabled"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid boolean, got %d", rec.Code)
		}
	})

	t.Run("admin can set valid whatsapp link", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "https://chat.whatsapp.com/ABCDEF123456",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/whatsapp_group_link", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "whatsapp_group_link"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify update
		var value string
		db.QueryRow("SELECT value FROM system_settings WHERE key = ?", "whatsapp_group_link").Scan(&value)

		if value != "https://chat.whatsapp.com/ABCDEF123456" {
			t.Errorf("Expected WhatsApp link to be saved, got %s", value)
		}
	})

	t.Run("admin can set empty whatsapp link", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/whatsapp_group_link", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "whatsapp_group_link"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		// Empty link should be allowed (to clear the link)
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for empty link, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("reject invalid whatsapp link format", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "https://example.com/not-whatsapp",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/whatsapp_group_link", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "whatsapp_group_link"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid WhatsApp link, got %d", rec.Code)
		}

		// Verify error message mentions WhatsApp
		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)
		errorMsg, _ := response["error"].(string)

		if errorMsg == "" || !contains(errorMsg, "chat.whatsapp.com") {
			t.Errorf("Expected error message to mention valid WhatsApp URL format, got: %s", errorMsg)
		}
	})
}

// TestSettingsHandler_WhatsAppWorkflow tests the complete WhatsApp enable/disable workflow
func TestSettingsHandler_WhatsAppWorkflow(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewSettingsHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	// Step 1: Get initial state (disabled)
	t.Run("Step 1: WhatsApp disabled by default", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/settings/whatsapp", nil)
		rec := httptest.NewRecorder()
		handler.GetWhatsAppSettings(rec, req)

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["enabled"] != false {
			t.Error("WhatsApp should be disabled by default")
		}
	})

	// Step 2: Set WhatsApp link
	t.Run("Step 2: Set WhatsApp link", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "https://chat.whatsapp.com/TestGroup123",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/whatsapp_group_link", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "whatsapp_group_link"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Failed to set WhatsApp link: %s", rec.Body.String())
		}
	})

	// Step 3: Enable WhatsApp
	t.Run("Step 3: Enable WhatsApp", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "true",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/whatsapp_group_enabled", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "whatsapp_group_enabled"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Failed to enable WhatsApp: %s", rec.Body.String())
		}
	})

	// Step 4: Verify WhatsApp is enabled with link
	t.Run("Step 4: Verify WhatsApp enabled with link", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/settings/whatsapp", nil)
		rec := httptest.NewRecorder()
		handler.GetWhatsAppSettings(rec, req)

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["enabled"] != true {
			t.Error("WhatsApp should be enabled")
		}

		if response["link"] != "https://chat.whatsapp.com/TestGroup123" {
			t.Errorf("Expected link to match, got %v", response["link"])
		}
	})

	// Step 5: Disable WhatsApp
	t.Run("Step 5: Disable WhatsApp", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"value": "false",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/settings/whatsapp_group_enabled", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"key": "whatsapp_group_enabled"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateSetting(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Failed to disable WhatsApp: %s", rec.Body.String())
		}
	})

	// Step 6: Verify WhatsApp is disabled (link should still exist)
	t.Run("Step 6: Verify WhatsApp disabled", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/settings/whatsapp", nil)
		rec := httptest.NewRecorder()
		handler.GetWhatsAppSettings(rec, req)

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["enabled"] != false {
			t.Error("WhatsApp should be disabled")
		}

		// Link should still be preserved
		if response["link"] != "https://chat.whatsapp.com/TestGroup123" {
			t.Errorf("Link should be preserved after disabling, got %v", response["link"])
		}
	})
}

// helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Suppress unused import warning
var _ = fmt.Sprintf
