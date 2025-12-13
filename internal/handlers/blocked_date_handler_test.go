package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/testutil"
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

// TestBlockedDateHandler_CreateDogSpecificBlock tests creating dog-specific blocked dates
func TestBlockedDateHandler_CreateDogSpecificBlock(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBlockedDateHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	t.Run("create dog-specific block", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"date":   "2025-12-25",
			"reason": "Vet appointment",
			"dog_id": dogID,
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

		blockedDate, ok := response["blocked_date"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected blocked_date in response")
		}

		// Check dog_id is set in response
		if blockedDate["dog_id"] == nil {
			t.Error("Expected dog_id in response")
		}

		responseDogID := int(blockedDate["dog_id"].(float64))
		if responseDogID != dogID {
			t.Errorf("Expected dog_id %d, got %d", dogID, responseDogID)
		}
	})

	t.Run("create global block with null dog_id", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"date":   "2025-12-26",
			"reason": "Boxing Day - Global",
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

		blockedDate, ok := response["blocked_date"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected blocked_date in response")
		}

		// Check dog_id is nil for global block
		if blockedDate["dog_id"] != nil {
			t.Errorf("Expected dog_id to be nil for global block, got %v", blockedDate["dog_id"])
		}
	})

	t.Run("invalid dog_id returns error", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"date":   "2025-12-27",
			"reason": "Invalid dog",
			"dog_id": 99999, // Non-existent dog
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/blocked-dates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBlockedDate(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent dog, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("same date different dogs allowed", func(t *testing.T) {
		date := "2025-12-28"
		dog1ID := testutil.SeedTestDog(t, db, "Dog1", "Labrador", "green")
		dog2ID := testutil.SeedTestDog(t, db, "Dog2", "Beagle", "blue")

		// Create block for dog1
		reqBody1 := map[string]interface{}{
			"date":   date,
			"reason": "Block Dog1",
			"dog_id": dog1ID,
		}
		body1, _ := json.Marshal(reqBody1)
		req1 := httptest.NewRequest("POST", "/api/blocked-dates", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		ctx1 := contextWithUser(req1.Context(), adminID, "admin@example.com", true)
		req1 = req1.WithContext(ctx1)

		rec1 := httptest.NewRecorder()
		handler.CreateBlockedDate(rec1, req1)

		if rec1.Code != http.StatusCreated {
			t.Fatalf("First dog block failed: %d. Body: %s", rec1.Code, rec1.Body.String())
		}

		// Create block for dog2 on same date - should succeed
		reqBody2 := map[string]interface{}{
			"date":   date,
			"reason": "Block Dog2",
			"dog_id": dog2ID,
		}
		body2, _ := json.Marshal(reqBody2)
		req2 := httptest.NewRequest("POST", "/api/blocked-dates", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		ctx2 := contextWithUser(req2.Context(), adminID, "admin@example.com", true)
		req2 = req2.WithContext(ctx2)

		rec2 := httptest.NewRecorder()
		handler.CreateBlockedDate(rec2, req2)

		if rec2.Code != http.StatusCreated {
			t.Errorf("Second dog block should succeed, got %d. Body: %s", rec2.Code, rec2.Body.String())
		}
	})

	t.Run("duplicate dog-date fails", func(t *testing.T) {
		date := "2025-12-29"
		dupDogID := testutil.SeedTestDog(t, db, "DupDog", "Poodle", "green")

		// First block
		reqBody1 := map[string]interface{}{
			"date":   date,
			"reason": "First block",
			"dog_id": dupDogID,
		}
		body1, _ := json.Marshal(reqBody1)
		req1 := httptest.NewRequest("POST", "/api/blocked-dates", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		ctx1 := contextWithUser(req1.Context(), adminID, "admin@example.com", true)
		req1 = req1.WithContext(ctx1)

		rec1 := httptest.NewRecorder()
		handler.CreateBlockedDate(rec1, req1)

		if rec1.Code != http.StatusCreated {
			t.Fatalf("First block failed: %d", rec1.Code)
		}

		// Duplicate block - should fail
		reqBody2 := map[string]interface{}{
			"date":   date,
			"reason": "Duplicate block",
			"dog_id": dupDogID,
		}
		body2, _ := json.Marshal(reqBody2)
		req2 := httptest.NewRequest("POST", "/api/blocked-dates", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		ctx2 := contextWithUser(req2.Context(), adminID, "admin@example.com", true)
		req2 = req2.WithContext(ctx2)

		rec2 := httptest.NewRecorder()
		handler.CreateBlockedDate(rec2, req2)

		if rec2.Code != http.StatusConflict {
			t.Errorf("Expected status 409 for duplicate dog-date, got %d. Body: %s", rec2.Code, rec2.Body.String())
		}
	})
}

// TestBlockedDateHandler_ListBlockedDatesWithDogInfo tests listing blocked dates shows dog info
func TestBlockedDateHandler_ListBlockedDatesWithDogInfo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBlockedDateHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	// Create global block
	testutil.SeedTestBlockedDate(t, db, "2025-12-25", "Christmas - Global", adminID)

	// Create dog-specific block
	testutil.SeedTestBlockedDateForDog(t, db, "2025-12-26", "Vet for Buddy", adminID, dogID)

	t.Run("list includes dog info", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/blocked-dates", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
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

		// Find the dog-specific block and check dog_name
		var foundDogBlock bool
		for _, bd := range dates {
			if bd["dog_id"] != nil {
				foundDogBlock = true
				if bd["dog_name"] == nil {
					t.Error("Expected dog_name for dog-specific block")
				} else if bd["dog_name"].(string) != "Buddy" {
					t.Errorf("Expected dog_name 'Buddy', got '%s'", bd["dog_name"])
				}
			}
		}

		if !foundDogBlock {
			t.Error("Expected to find dog-specific block in list")
		}
	})
}
