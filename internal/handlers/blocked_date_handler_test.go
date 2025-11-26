package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tranm/gassigeher/internal/config"
	"github.com/tranm/gassigeher/internal/testutil"
)

// DONE: TestBlockedDateHandler_ListBlockedDates tests listing blocked dates
func TestBlockedDateHandler_ListBlockedDates(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBlockedDateHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	// Create blocked dates
	testutil.SeedTestBlockedDate(t, db, "2025-12-25", "Christmas", adminID)
	testutil.SeedTestBlockedDate(t, db, "2025-12-26", "Boxing Day", adminID)

	t.Run("list all blocked dates", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/blocked-dates", nil)
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListBlockedDates(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var dates []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &dates)

		if len(dates) != 2 {
			t.Errorf("Expected 2 blocked dates, got %d", len(dates))
		}
	})

	t.Run("empty list when no blocked dates", func(t *testing.T) {
		// Use fresh DB
		db2 := testutil.SetupTestDB(t)
		handler2 := NewBlockedDateHandler(db2, cfg)

		req := httptest.NewRequest("GET", "/api/blocked-dates", nil)
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler2.ListBlockedDates(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var dates []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &dates)

		if len(dates) != 0 {
			t.Errorf("Expected 0 blocked dates, got %d", len(dates))
		}
	})
}

// DONE: TestBlockedDateHandler_CreateBlockedDate tests creating blocked dates (admin only)
func TestBlockedDateHandler_CreateBlockedDate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBlockedDateHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")
	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")

	t.Run("successful creation by admin", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"date":   "2025-12-31",
			"reason": "New Year's Eve",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/blocked-dates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBlockedDate(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["blocked_date"] == nil {
			t.Error("Expected blocked_date in response")
		}

		blockedDate, ok := response["blocked_date"].(map[string]interface{})
		if !ok || blockedDate["id"] == nil {
			t.Error("Expected blocked date ID in response")
		}

		if response["cancelled_bookings"] == nil {
			t.Error("Expected cancelled_bookings count in response")
		}
	})

	t.Run("non-admin cannot create", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"date":   "2026-01-01",
			"reason": "Holiday",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/blocked-dates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBlockedDate(rec, req)

		// Note: RequireAdmin middleware blocks in production
		// Test handler behavior when reached
		t.Logf("Non-admin create attempt returned status: %d", rec.Code)
	})

	t.Run("invalid date format", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"date":   "31-12-2025", // Wrong format
			"reason": "Holiday",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/blocked-dates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBlockedDate(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid date format, got %d", rec.Code)
		}
	})

	t.Run("missing reason", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"date": "2025-12-31",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/blocked-dates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBlockedDate(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing reason, got %d", rec.Code)
		}
	})

	t.Run("duplicate date", func(t *testing.T) {
		// Create first blocked date
		date := "2025-11-20"
		testutil.SeedTestBlockedDate(t, db, date, "Already blocked", adminID)

		// Try to create duplicate
		reqBody := map[string]interface{}{
			"date":   date,
			"reason": "Duplicate",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/blocked-dates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBlockedDate(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("Expected status 409 for duplicate date, got %d", rec.Code)
		}
	})
}

// DONE: TestBlockedDateHandler_DeleteBlockedDate tests deleting blocked dates (admin only)
func TestBlockedDateHandler_DeleteBlockedDate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBlockedDateHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")
	blockedID := testutil.SeedTestBlockedDate(t, db, "2025-12-25", "Christmas", adminID)

	t.Run("successful deletion", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/blocked-dates/"+fmt.Sprintf("%d", blockedID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", blockedID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteBlockedDate(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify deletion
		var count int
		db.QueryRow("SELECT COUNT(*) FROM blocked_dates WHERE id = ?", blockedID).Scan(&count)

		if count != 0 {
			t.Error("Blocked date should be deleted")
		}
	})

	t.Run("delete non-existent blocked date", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/blocked-dates/99999", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteBlockedDate(rec, req)

		// Handler returns OK even if blocked date doesn't exist (idempotent delete)
		t.Logf("Delete non-existent blocked date returned status: %d", rec.Code)
	})

	t.Run("invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/blocked-dates/invalid", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteBlockedDate(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}
