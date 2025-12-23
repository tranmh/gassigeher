package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestWalkReportHandler_CreateReport tests creating walk reports
func TestWalkReportHandler_CreateReport(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		UploadDir:       "/tmp/test-uploads",
		MaxUploadSizeMB: 5,
	}
	handler := NewWalkReportHandler(db, cfg)

	// Create test user and dog
	userID := testutil.SeedTestUser(t, db, "walker@example.com", "Walker User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin User", "blue")
	db.Exec("UPDATE users SET is_admin = 1 WHERE id = ?", adminID)

	otherUserID := testutil.SeedTestUser(t, db, "other@example.com", "Other User", "green")

	// Create a dog
	dogResult, err := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available)
		VALUES (1, 'Test Dog', 'Mixed', 'medium', 3, 1, 1)
	`)
	if err != nil {
		t.Fatalf("Failed to create test dog: %v", err)
	}
	dogID, _ := dogResult.LastInsertId()

	// Create a completed booking for the user
	bookingResult, _ := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
		VALUES (1, ?, ?, ?, '10:00', 'completed')
	`, userID, int(dogID), time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	completedBookingID, _ := bookingResult.LastInsertId()

	// Create a scheduled (not completed) booking
	scheduledResult, _ := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
		VALUES (1, ?, ?, ?, '11:00', 'scheduled')
	`, userID, int(dogID), time.Now().AddDate(0, 0, 1).Format("2006-01-02"))
	scheduledBookingID, _ := scheduledResult.LastInsertId()

	// Create a completed booking for other user
	otherBookingResult, _ := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
		VALUES (1, ?, ?, ?, '14:00', 'completed')
	`, otherUserID, int(dogID), time.Now().AddDate(0, 0, -2).Format("2006-01-02"))
	otherUserBookingID, _ := otherBookingResult.LastInsertId()

	t.Run("creates report for own completed booking", func(t *testing.T) {
		notes := "Great walk!"
		reqBody := map[string]interface{}{
			"booking_id":      int(completedBookingID),
			"behavior_rating": 5,
			"energy_level":    "high",
			"notes":           notes,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/walk-reports", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateReport(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["behavior_rating"] != float64(5) {
			t.Errorf("Expected behavior_rating 5, got %v", response["behavior_rating"])
		}
	})

	t.Run("returns 409 if report already exists", func(t *testing.T) {
		// Try to create another report for the same booking
		reqBody := map[string]interface{}{
			"booking_id":      int(completedBookingID),
			"behavior_rating": 4,
			"energy_level":    "medium",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/walk-reports", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateReport(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("Expected status 409, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for booking not completed", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"booking_id":      int(scheduledBookingID),
			"behavior_rating": 4,
			"energy_level":    "medium",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/walk-reports", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateReport(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns 403 for booking owned by another user", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"booking_id":      int(otherUserBookingID),
			"behavior_rating": 4,
			"energy_level":    "medium",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/walk-reports", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 1, userID, false) // userID trying to create report for otherUser's booking
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateReport(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})

	t.Run("admin can create report for any booking", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"booking_id":      int(otherUserBookingID),
			"behavior_rating": 4,
			"energy_level":    "medium",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/walk-reports", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateReport(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201 for admin, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("returns 400 for invalid behavior rating", func(t *testing.T) {
		// Create another completed booking for validation test
		newBookingResult, _ := db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
			VALUES (1, ?, ?, ?, '15:00', 'completed')
		`, userID, int(dogID), time.Now().AddDate(0, 0, -3).Format("2006-01-02"))
		newBookingID, _ := newBookingResult.LastInsertId()

		reqBody := map[string]interface{}{
			"booking_id":      int(newBookingID),
			"behavior_rating": 10, // Invalid - must be 1-5
			"energy_level":    "medium",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/walk-reports", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateReport(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid rating, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid energy level", func(t *testing.T) {
		// Create another completed booking for validation test
		newBookingResult, _ := db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
			VALUES (1, ?, ?, ?, '16:00', 'completed')
		`, userID, int(dogID), time.Now().AddDate(0, 0, -4).Format("2006-01-02"))
		newBookingID, _ := newBookingResult.LastInsertId()

		reqBody := map[string]interface{}{
			"booking_id":      int(newBookingID),
			"behavior_rating": 4,
			"energy_level":    "super_high", // Invalid - must be low/medium/high
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/walk-reports", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateReport(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid energy level, got %d", rec.Code)
		}
	})

	t.Run("returns 404 for non-existent booking", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"booking_id":      99999,
			"behavior_rating": 4,
			"energy_level":    "medium",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/walk-reports", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateReport(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

// TestWalkReportHandler_GetReport tests getting a report by ID
func TestWalkReportHandler_GetReport(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		UploadDir:       "/tmp/test-uploads",
		MaxUploadSizeMB: 5,
	}
	handler := NewWalkReportHandler(db, cfg)

	// Create test user and dog
	userID := testutil.SeedTestUser(t, db, "walker@example.com", "Walker User", "green")

	dogResult, _ := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available)
		VALUES (1, 'Test Dog', 'Mixed', 'medium', 3, 1, 1)
	`)
	dogID, _ := dogResult.LastInsertId()

	bookingResult, _ := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
		VALUES (1, ?, ?, ?, '10:00', 'completed')
	`, userID, int(dogID), time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	bookingID, _ := bookingResult.LastInsertId()

	// Create a walk report
	reportResult, _ := db.Exec(`
		INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level, notes)
		VALUES (1, ?, 5, 'high', 'Great walk!')
	`, int(bookingID))
	reportID, _ := reportResult.LastInsertId()

	t.Run("returns report by ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/walk-reports/1", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetReport(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["id"] != float64(reportID) {
			t.Errorf("Expected report ID %d, got %v", reportID, response["id"])
		}
	})

	t.Run("returns 404 for non-existent report", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/walk-reports/99999", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetReport(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid report ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/walk-reports/invalid", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetReport(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestWalkReportHandler_GetReportByBooking tests getting a report by booking ID
func TestWalkReportHandler_GetReportByBooking(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		UploadDir:       "/tmp/test-uploads",
		MaxUploadSizeMB: 5,
	}
	handler := NewWalkReportHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "walker@example.com", "Walker User", "green")

	dogResult, _ := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available)
		VALUES (1, 'Test Dog', 'Mixed', 'medium', 3, 1, 1)
	`)
	dogID, _ := dogResult.LastInsertId()

	bookingResult, _ := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
		VALUES (1, ?, ?, ?, '10:00', 'completed')
	`, userID, int(dogID), time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	bookingID, _ := bookingResult.LastInsertId()

	// Create a walk report
	db.Exec(`
		INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level)
		VALUES (1, ?, 4, 'medium')
	`, int(bookingID))

	t.Run("returns report by booking ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/bookings/1/report", nil)
		req = mux.SetURLVars(req, map[string]string{"bookingId": "1"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetReportByBooking(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("returns 404 for booking without report", func(t *testing.T) {
		// Create another booking without a report
		noReportBooking, _ := db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
			VALUES (1, ?, ?, ?, '11:00', 'completed')
		`, userID, int(dogID), time.Now().AddDate(0, 0, -2).Format("2006-01-02"))
		noReportBookingID, _ := noReportBooking.LastInsertId()

		req := httptest.NewRequest("GET", "/api/bookings/2/report", nil)
		req = mux.SetURLVars(req, map[string]string{"bookingId": strconv.Itoa(int(noReportBookingID))})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetReportByBooking(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

// TestWalkReportHandler_GetDogWalkReports tests getting reports for a dog
func TestWalkReportHandler_GetDogWalkReports(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		UploadDir:       "/tmp/test-uploads",
		MaxUploadSizeMB: 5,
	}
	handler := NewWalkReportHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "walker@example.com", "Walker User", "green")

	dogResult, _ := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available)
		VALUES (1, 'Test Dog', 'Mixed', 'medium', 3, 1, 1)
	`)
	dogID, _ := dogResult.LastInsertId()

	// Create multiple completed bookings with reports
	for i := 1; i <= 5; i++ {
		bookingResult, _ := db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
			VALUES (1, ?, ?, ?, '10:00', 'completed')
		`, userID, int(dogID), time.Now().AddDate(0, 0, -i).Format("2006-01-02"))
		bookingID, _ := bookingResult.LastInsertId()

		db.Exec(`
			INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level)
			VALUES (1, ?, ?, 'medium')
		`, int(bookingID), i)
	}

	t.Run("returns reports with stats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs/1/walk-reports", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetDogWalkReports(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["dog"] == nil {
			t.Error("Expected 'dog' in response")
		}
		if response["stats"] == nil {
			t.Error("Expected 'stats' in response")
		}
		if response["reports"] == nil {
			t.Error("Expected 'reports' in response")
		}
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs/1/walk-reports?limit=2", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetDogWalkReports(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		reports := response["reports"].([]interface{})
		if len(reports) > 2 {
			t.Errorf("Expected max 2 reports, got %d", len(reports))
		}
	})

	t.Run("returns 404 for non-existent dog", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs/99999/walk-reports", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetDogWalkReports(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

// TestWalkReportHandler_UpdateReport tests updating reports
func TestWalkReportHandler_UpdateReport(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		UploadDir:       "/tmp/test-uploads",
		MaxUploadSizeMB: 5,
	}
	handler := NewWalkReportHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "walker@example.com", "Walker User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin User", "blue")
	db.Exec("UPDATE users SET is_admin = 1 WHERE id = ?", adminID)
	otherUserID := testutil.SeedTestUser(t, db, "other@example.com", "Other User", "green")

	dogResult, _ := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available)
		VALUES (1, 'Test Dog', 'Mixed', 'medium', 3, 1, 1)
	`)
	dogID, _ := dogResult.LastInsertId()

	// Create booking and report for userID
	bookingResult, _ := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
		VALUES (1, ?, ?, ?, '10:00', 'completed')
	`, userID, int(dogID), time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	bookingID, _ := bookingResult.LastInsertId()

	reportResult, _ := db.Exec(`
		INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level)
		VALUES (1, ?, 3, 'low')
	`, int(bookingID))
	reportID, _ := reportResult.LastInsertId()

	t.Run("updates own report", func(t *testing.T) {
		newNotes := "Updated notes"
		reqBody := map[string]interface{}{
			"behavior_rating": 5,
			"energy_level":    "high",
			"notes":           newNotes,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/walk-reports/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateReport(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["behavior_rating"] != float64(5) {
			t.Errorf("Expected behavior_rating 5, got %v", response["behavior_rating"])
		}
		if response["energy_level"] != "high" {
			t.Errorf("Expected energy_level 'high', got %v", response["energy_level"])
		}
	})

	t.Run("returns 403 for other user's report", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"behavior_rating": 4,
			"energy_level":    "medium",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/walk-reports/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		ctx := contextWithTenant(req.Context(), 1, otherUserID, false) // otherUser trying to update userID's report
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateReport(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})

	t.Run("admin can update any report", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"behavior_rating": 4,
			"energy_level":    "medium",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/walk-reports/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateReport(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for admin, got %d", rec.Code)
		}
	})

	t.Run("returns 404 for non-existent report", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"behavior_rating": 4,
			"energy_level":    "medium",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/walk-reports/99999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateReport(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid request", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"behavior_rating": 10, // Invalid
			"energy_level":    "medium",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/walk-reports/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(int(reportID))})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateReport(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestWalkReportHandler_DeleteReport tests deleting reports
func TestWalkReportHandler_DeleteReport(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		UploadDir:       "/tmp/test-uploads",
		MaxUploadSizeMB: 5,
	}
	handler := NewWalkReportHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "walker@example.com", "Walker User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin User", "blue")
	db.Exec("UPDATE users SET is_admin = 1 WHERE id = ?", adminID)
	otherUserID := testutil.SeedTestUser(t, db, "other@example.com", "Other User", "green")

	dogResult, _ := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available)
		VALUES (1, 'Test Dog', 'Mixed', 'medium', 3, 1, 1)
	`)
	dogID, _ := dogResult.LastInsertId()

	// Create booking and report for userID
	bookingResult, _ := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
		VALUES (1, ?, ?, ?, '10:00', 'completed')
	`, userID, int(dogID), time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	bookingID, _ := bookingResult.LastInsertId()

	t.Run("deletes own report", func(t *testing.T) {
		// Create a report to delete
		reportResult, _ := db.Exec(`
			INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level)
			VALUES (1, ?, 3, 'low')
		`, int(bookingID))
		reportID, _ := reportResult.LastInsertId()

		req := httptest.NewRequest("DELETE", "/api/walk-reports/1", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(int(reportID))})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteReport(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("returns 403 for other user's report", func(t *testing.T) {
		// Create another report
		bookingResult2, _ := db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
			VALUES (1, ?, ?, ?, '11:00', 'completed')
		`, userID, int(dogID), time.Now().AddDate(0, 0, -2).Format("2006-01-02"))
		bookingID2, _ := bookingResult2.LastInsertId()

		reportResult2, _ := db.Exec(`
			INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level)
			VALUES (1, ?, 4, 'medium')
		`, int(bookingID2))
		reportID2, _ := reportResult2.LastInsertId()

		req := httptest.NewRequest("DELETE", "/api/walk-reports/2", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(int(reportID2))})
		ctx := contextWithTenant(req.Context(), 1, otherUserID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteReport(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})

	t.Run("admin can delete any report", func(t *testing.T) {
		// Create another report
		bookingResult3, _ := db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
			VALUES (1, ?, ?, ?, '12:00', 'completed')
		`, userID, int(dogID), time.Now().AddDate(0, 0, -3).Format("2006-01-02"))
		bookingID3, _ := bookingResult3.LastInsertId()

		reportResult3, _ := db.Exec(`
			INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level)
			VALUES (1, ?, 5, 'high')
		`, int(bookingID3))
		reportID3, _ := reportResult3.LastInsertId()

		req := httptest.NewRequest("DELETE", "/api/walk-reports/3", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(int(reportID3))})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteReport(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for admin, got %d", rec.Code)
		}
	})

	t.Run("returns 404 for non-existent report", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/walk-reports/99999", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteReport(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

// TestWalkReportHandler_UploadPhoto tests photo upload authorization
func TestWalkReportHandler_UploadPhoto(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		UploadDir:       "/tmp/test-uploads",
		MaxUploadSizeMB: 5,
	}
	handler := NewWalkReportHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "walker@example.com", "Walker User", "green")
	otherUserID := testutil.SeedTestUser(t, db, "other@example.com", "Other User", "green")

	dogResult, _ := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available)
		VALUES (1, 'Test Dog', 'Mixed', 'medium', 3, 1, 1)
	`)
	dogID, _ := dogResult.LastInsertId()

	bookingResult, _ := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
		VALUES (1, ?, ?, ?, '10:00', 'completed')
	`, userID, int(dogID), time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	bookingID, _ := bookingResult.LastInsertId()

	reportResult, _ := db.Exec(`
		INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level)
		VALUES (1, ?, 4, 'medium')
	`, int(bookingID))
	reportID, _ := reportResult.LastInsertId()

	t.Run("returns 403 for other user", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/walk-reports/1/photos", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(int(reportID))})
		ctx := contextWithTenant(req.Context(), 1, otherUserID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UploadPhoto(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})

	t.Run("returns 404 for non-existent report", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/walk-reports/99999/photos", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UploadPhoto(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("returns 400 when max photos exceeded", func(t *testing.T) {
		// Add 3 photos to reach limit
		for i := 0; i < 3; i++ {
			db.Exec(`
				INSERT INTO walk_report_photos (tenant_id, walk_report_id, photo_path, photo_thumbnail, display_order)
				VALUES (1, ?, ?, ?, ?)
			`, int(reportID), "path.jpg", "thumb.jpg", i)
		}

		req := httptest.NewRequest("POST", "/api/walk-reports/1/photos", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(int(reportID))})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UploadPhoto(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for max photos, got %d", rec.Code)
		}
	})
}

// TestWalkReportHandler_DeletePhoto tests photo deletion
func TestWalkReportHandler_DeletePhoto(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		UploadDir:       "/tmp/test-uploads",
		MaxUploadSizeMB: 5,
	}
	handler := NewWalkReportHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "walker@example.com", "Walker User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin User", "blue")
	db.Exec("UPDATE users SET is_admin = 1 WHERE id = ?", adminID)
	otherUserID := testutil.SeedTestUser(t, db, "other@example.com", "Other User", "green")

	dogResult, _ := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available)
		VALUES (1, 'Test Dog', 'Mixed', 'medium', 3, 1, 1)
	`)
	dogID, _ := dogResult.LastInsertId()

	bookingResult, _ := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
		VALUES (1, ?, ?, ?, '10:00', 'completed')
	`, userID, int(dogID), time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	bookingID, _ := bookingResult.LastInsertId()

	reportResult, _ := db.Exec(`
		INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level)
		VALUES (1, ?, 4, 'medium')
	`, int(bookingID))
	reportID, _ := reportResult.LastInsertId()

	// Add a photo
	photoResult, _ := db.Exec(`
		INSERT INTO walk_report_photos (tenant_id, walk_report_id, photo_path, photo_thumbnail, display_order)
		VALUES (1, ?, 'test/path.jpg', 'test/thumb.jpg', 0)
	`, int(reportID))
	photoID, _ := photoResult.LastInsertId()

	t.Run("deletes own photo", func(t *testing.T) {
		// Add another photo to test deletion
		photoResult2, _ := db.Exec(`
			INSERT INTO walk_report_photos (tenant_id, walk_report_id, photo_path, photo_thumbnail, display_order)
			VALUES (1, ?, 'test/path2.jpg', 'test/thumb2.jpg', 1)
		`, int(reportID))
		photoID2, _ := photoResult2.LastInsertId()

		req := httptest.NewRequest("DELETE", "/api/walk-reports/1/photos/2", nil)
		req = mux.SetURLVars(req, map[string]string{
			"id":      strconv.Itoa(int(reportID)),
			"photoId": strconv.Itoa(int(photoID2)),
		})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeletePhoto(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("returns 403 for other user", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/walk-reports/1/photos/1", nil)
		req = mux.SetURLVars(req, map[string]string{
			"id":      strconv.Itoa(int(reportID)),
			"photoId": strconv.Itoa(int(photoID)),
		})
		ctx := contextWithTenant(req.Context(), 1, otherUserID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeletePhoto(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})

	t.Run("admin can delete any photo", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/walk-reports/1/photos/1", nil)
		req = mux.SetURLVars(req, map[string]string{
			"id":      strconv.Itoa(int(reportID)),
			"photoId": strconv.Itoa(int(photoID)),
		})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeletePhoto(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for admin, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("returns 404 for non-existent report", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/walk-reports/99999/photos/1", nil)
		req = mux.SetURLVars(req, map[string]string{
			"id":      "99999",
			"photoId": "1",
		})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeletePhoto(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}
