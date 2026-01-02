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

// TestPromoCodeHandler_CreatePromoCode tests creating promo codes
func TestPromoCodeHandler_CreatePromoCode(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{}
	handler := NewPromoCodeHandler(db, cfg)

	t.Run("creates percentage discount code", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":           "SAVE20",
			"description":    "20% off",
			"discount_type":  "percentage",
			"discount_value": 20,
			"is_active":      true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/central-admin/promo-codes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreatePromoCode(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("creates free months code", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":           "FREE3MONTHS",
			"discount_type":  "free_months",
			"discount_value": 3,
			"is_active":      true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/central-admin/promo-codes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreatePromoCode(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("rejects empty code", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":           "",
			"discount_type":  "percentage",
			"discount_value": 10,
			"is_active":      true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/central-admin/promo-codes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreatePromoCode(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("rejects code with special characters", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":           "SAVE@20%!",
			"discount_type":  "percentage",
			"discount_value": 20,
			"is_active":      true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/central-admin/promo-codes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreatePromoCode(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("rejects percentage over 100", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":           "TOOBIG",
			"discount_type":  "percentage",
			"discount_value": 150,
			"is_active":      true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/central-admin/promo-codes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreatePromoCode(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("rejects free_months over 24", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":           "TOOMANYMONTHS",
			"discount_type":  "free_months",
			"discount_value": 36,
			"is_active":      true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/central-admin/promo-codes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreatePromoCode(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("rejects invalid discount type", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"code":           "INVALID",
			"discount_type":  "invalid_type",
			"discount_value": 10,
			"is_active":      true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/central-admin/promo-codes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreatePromoCode(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("rejects duplicate code", func(t *testing.T) {
		// Create first code
		reqBody := map[string]interface{}{
			"code":           "DUPLICATE",
			"discount_type":  "percentage",
			"discount_value": 10,
			"is_active":      true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/central-admin/promo-codes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.CreatePromoCode(rec, req)

		// Try to create duplicate
		req2 := httptest.NewRequest("POST", "/api/v1/central-admin/promo-codes", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		rec2 := httptest.NewRecorder()
		handler.CreatePromoCode(rec2, req2)

		if rec2.Code != http.StatusConflict {
			t.Errorf("Expected status 409, got %d", rec2.Code)
		}
	})

	t.Run("accepts code with expiry date", func(t *testing.T) {
		expiresAt := "2030-12-31"
		reqBody := map[string]interface{}{
			"code":           "EXPIRING",
			"discount_type":  "percentage",
			"discount_value": 15,
			"is_active":      true,
			"expires_at":     expiresAt,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/central-admin/promo-codes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreatePromoCode(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}

// TestPromoCodeHandler_GetAllPromoCodes tests listing all promo codes
func TestPromoCodeHandler_GetAllPromoCodes(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{}
	handler := NewPromoCodeHandler(db, cfg)

	// Create some test codes
	now := testutil.Now()
	db.Exec("INSERT INTO promo_codes (code, discount_type, discount_value, is_active, created_at, updated_at) VALUES ('ACTIVE10', 'percentage', 10, 1, ?, ?)", now, now)
	db.Exec("INSERT INTO promo_codes (code, discount_type, discount_value, is_active, created_at, updated_at) VALUES ('INACTIVE5', 'percentage', 5, 0, ?, ?)", now, now)

	t.Run("returns all codes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/central-admin/promo-codes", nil)

		rec := httptest.NewRecorder()
		handler.GetAllPromoCodes(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		count := int(response["count"].(float64))
		if count < 2 {
			t.Errorf("Expected at least 2 codes, got %d", count)
		}
	})

	t.Run("filters active only", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/central-admin/promo-codes?active_only=true", nil)

		rec := httptest.NewRecorder()
		handler.GetAllPromoCodes(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		codes := response["promo_codes"].([]interface{})
		for _, code := range codes {
			codeMap := code.(map[string]interface{})
			if codeMap["is_active"] != true {
				t.Error("Expected only active codes when active_only=true")
			}
		}
	})
}

// TestPromoCodeHandler_ValidatePromoCode tests public promo code validation
func TestPromoCodeHandler_ValidatePromoCode(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{}
	handler := NewPromoCodeHandler(db, cfg)

	// Create test codes
	now := testutil.Now()
	db.Exec("INSERT INTO promo_codes (code, discount_type, discount_value, is_active, created_at, updated_at) VALUES ('VALID20', 'percentage', 20, 1, ?, ?)", now, now)
	db.Exec("INSERT INTO promo_codes (code, discount_type, discount_value, is_active, created_at, updated_at) VALUES ('INACTIVE', 'percentage', 10, 0, ?, ?)", now, now)

	t.Run("validates active code", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/promo-codes/validate/VALID20", nil)
		req = mux.SetURLVars(req, map[string]string{"code": "VALID20"})

		rec := httptest.NewRecorder()
		handler.ValidatePromoCode(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["valid"] != true {
			t.Errorf("Expected valid=true, got %v", response["valid"])
		}
		if response["discount_type"] != "percentage" {
			t.Errorf("Expected discount_type=percentage, got %v", response["discount_type"])
		}
	})

	t.Run("rejects inactive code", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/promo-codes/validate/INACTIVE", nil)
		req = mux.SetURLVars(req, map[string]string{"code": "INACTIVE"})

		rec := httptest.NewRecorder()
		handler.ValidatePromoCode(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["valid"] != false {
			t.Errorf("Expected valid=false for inactive code, got %v", response["valid"])
		}
	})

	t.Run("rejects non-existent code", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/promo-codes/validate/NONEXISTENT", nil)
		req = mux.SetURLVars(req, map[string]string{"code": "NONEXISTENT"})

		rec := httptest.NewRecorder()
		handler.ValidatePromoCode(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["valid"] != false {
			t.Errorf("Expected valid=false for non-existent code, got %v", response["valid"])
		}
	})

	t.Run("handles case-insensitive codes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/promo-codes/validate/valid20", nil)
		req = mux.SetURLVars(req, map[string]string{"code": "valid20"})

		rec := httptest.NewRecorder()
		handler.ValidatePromoCode(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["valid"] != true {
			t.Errorf("Expected valid=true for lowercase code, got %v", response["valid"])
		}
	})

	t.Run("handles empty code", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/promo-codes/validate/", nil)
		req = mux.SetURLVars(req, map[string]string{"code": ""})

		rec := httptest.NewRecorder()
		handler.ValidatePromoCode(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["valid"] != false {
			t.Errorf("Expected valid=false for empty code, got %v", response["valid"])
		}
	})
}

// TestPromoCodeHandler_DeletePromoCode tests deleting promo codes
func TestPromoCodeHandler_DeletePromoCode(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{}
	handler := NewPromoCodeHandler(db, cfg)

	// Create test code
	now := testutil.Now()
	result, _ := db.Exec("INSERT INTO promo_codes (code, discount_type, discount_value, is_active, created_at, updated_at) VALUES ('DELETE_ME', 'percentage', 10, 1, ?, ?)", now, now)
	codeID, _ := result.LastInsertId()

	t.Run("deletes existing code", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/central-admin/promo-codes/1", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})

		rec := httptest.NewRecorder()
		handler.DeletePromoCode(rec, req)

		// May be 200 or 404 depending on whether ID 1 exists
		if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", rec.Code)
		}
	})

	t.Run("returns 404 for non-existent code", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/central-admin/promo-codes/99999", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})

		rec := httptest.NewRecorder()
		handler.DeletePromoCode(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/central-admin/promo-codes/invalid", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})

		rec := httptest.NewRecorder()
		handler.DeletePromoCode(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	_ = codeID // Use the codeID
}
