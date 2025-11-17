package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranm/gassigeher/internal/config"
	"github.com/tranm/gassigeher/internal/services"
	"github.com/tranm/gassigeher/internal/testutil"
)

// DONE: TestBookingHandler_CreateBooking tests booking creation endpoint
func TestBookingHandler_CreateBooking(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create test user and dog
	authService := services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours)
	hash, _ := authService.HashPassword("Test1234")

	email := "booking@example.com"
	userID := testutil.SeedTestUser(t, db, email, "Booking User", "green")
	dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

	// Update user to verified and active
	db.Exec("UPDATE users SET is_verified = 1, is_active = 1, password_hash = ? WHERE id = ?", hash, userID)

	// Create admin for blocked dates
	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "orange")

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	t.Run("successful booking creation", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"dog_id":         dogID,
			"date":           tomorrow,
			"walk_type":      "morning",
			"scheduled_time": "09:00",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/bookings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), userID, email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBooking(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["id"] == nil {
			t.Error("Expected booking ID in response")
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"date": tomorrow,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/bookings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), userID, email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBooking(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("past date booking", func(t *testing.T) {
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

		reqBody := map[string]interface{}{
			"dog_id":         dogID,
			"date":           yesterday,
			"walk_type":      "morning",
			"scheduled_time": "09:00",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/bookings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), userID, email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBooking(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for past date, got %d", rec.Code)
		}
	})

	t.Run("blocked date", func(t *testing.T) {
		// Create blocked date
		blockedDate := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
		testutil.SeedTestBlockedDate(t, db, blockedDate, "Holiday", adminID)

		reqBody := map[string]interface{}{
			"dog_id":         dogID,
			"date":           blockedDate,
			"walk_type":      "morning",
			"scheduled_time": "09:00",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/bookings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), userID, email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBooking(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for blocked date, got %d", rec.Code)
		}
	})

	t.Run("double booking same dog", func(t *testing.T) {
		// Create first booking
		date := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
		testutil.SeedTestBooking(t, db, userID, dogID, date, "morning", "09:00", "scheduled")

		// Try to create duplicate
		reqBody := map[string]interface{}{
			"dog_id":         dogID,
			"date":           date,
			"walk_type":      "morning",
			"scheduled_time": "09:30",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/bookings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), userID, email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBooking(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("Expected status 409 for double booking, got %d", rec.Code)
		}
	})

	t.Run("insufficient experience level", func(t *testing.T) {
		// Create orange dog (requires orange level)
		orangeDogID := testutil.SeedTestDog(t, db, "Rocky", "Rottweiler", "orange")

		// Green user tries to book orange dog
		date := time.Now().AddDate(0, 0, 2).Format("2006-01-02")

		reqBody := map[string]interface{}{
			"dog_id":         orangeDogID,
			"date":           date,
			"walk_type":      "morning",
			"scheduled_time": "09:00",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/bookings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), userID, email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBooking(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for insufficient level, got %d", rec.Code)
		}
	})

	t.Run("inactive user cannot book", func(t *testing.T) {
		// Create inactive user
		inactiveEmail := "inactive@example.com"
		inactiveID := testutil.SeedTestUser(t, db, inactiveEmail, "Inactive", "green")
		db.Exec("UPDATE users SET is_active = 0 WHERE id = ?", inactiveID)

		date := time.Now().AddDate(0, 0, 2).Format("2006-01-02")

		reqBody := map[string]interface{}{
			"dog_id":         dogID,
			"date":           date,
			"walk_type":      "evening",
			"scheduled_time": "15:00",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/bookings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), inactiveID, inactiveEmail, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateBooking(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for inactive user, got %d", rec.Code)
		}
	})
}

// DONE: TestBookingHandler_ListBookings tests listing user's bookings
func TestBookingHandler_ListBookings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create test data
	user1ID := testutil.SeedTestUser(t, db, "user1@example.com", "User 1", "green")
	user2ID := testutil.SeedTestUser(t, db, "user2@example.com", "User 2", "green")
	dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

	// Create bookings for user1
	date1 := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	date2 := time.Now().AddDate(0, 0, 2).Format("2006-01-02")
	testutil.SeedTestBooking(t, db, user1ID, dogID, date1, "morning", "09:00", "scheduled")
	testutil.SeedTestBooking(t, db, user1ID, dogID, date2, "evening", "15:00", "scheduled")

	// Create booking for user2
	testutil.SeedTestBooking(t, db, user2ID, dogID, date1, "evening", "16:00", "scheduled")

	t.Run("list user's own bookings", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/bookings", nil)
		ctx := contextWithUser(req.Context(), user1ID, "user1@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListBookings(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var bookings []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &bookings)

		if len(bookings) != 2 {
			t.Errorf("Expected 2 bookings for user1, got %d", len(bookings))
		}
	})

	t.Run("user cannot see other user's bookings", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/bookings", nil)
		ctx := contextWithUser(req.Context(), user2ID, "user2@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListBookings(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var bookings []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &bookings)

		// User2 should only see their own booking
		if len(bookings) != 1 {
			t.Errorf("Expected 1 booking for user2, got %d", len(bookings))
		}
	})
}

// DONE: TestBookingHandler_CancelBooking tests booking cancellation
func TestBookingHandler_CancelBooking(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "cancel@example.com", "Cancel User", "green")
	dogID := testutil.SeedTestDog(t, db, "Max", "Beagle", "green")

	// Create booking 2 days in future (beyond 12 hour notice period)
	twoDaysLater := time.Now().AddDate(0, 0, 2).Format("2006-01-02")
	bookingID := testutil.SeedTestBooking(t, db, userID, dogID, twoDaysLater, "morning", "09:00", "scheduled")

	t.Run("successful cancellation - admin override", func(t *testing.T) {
		// Admin can cancel without notice period restrictions
		req := httptest.NewRequest("PUT", "/api/bookings/"+fmt.Sprintf("%d", bookingID)+"/cancel", nil)

		// Set up router to handle path variables
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", bookingID)})

		ctx := contextWithUser(req.Context(), userID, "cancel@example.com", true) // isAdmin = true
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CancelBooking(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify booking is cancelled
		var status string
		db.QueryRow("SELECT status FROM bookings WHERE id = ?", bookingID).Scan(&status)

		if status != "cancelled" {
			t.Errorf("Expected status 'cancelled', got %s", status)
		}
	})

	t.Run("cancel booking of another user", func(t *testing.T) {
		// Create another user
		otherUserID := testutil.SeedTestUser(t, db, "other@example.com", "Other User", "green")

		// Create booking for user1
		date := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
		user1Booking := testutil.SeedTestBooking(t, db, userID, dogID, date, "evening", "15:00", "scheduled")

		// Try to cancel with otherUser context
		req := httptest.NewRequest("PUT", "/api/bookings/"+fmt.Sprintf("%d", user1Booking)+"/cancel", nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", user1Booking)})

		ctx := contextWithUser(req.Context(), otherUserID, "other@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CancelBooking(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})

	t.Run("cancel non-existent booking", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/bookings/99999/cancel", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})

		ctx := contextWithUser(req.Context(), userID, "cancel@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CancelBooking(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

