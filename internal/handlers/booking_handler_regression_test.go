package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// Phase 8: Regression Testing
// Purpose: Ensure existing booking features still work after time restrictions added

// setupRegressionTest creates a test database with necessary tables and seed data
func setupRegressionTest(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Enable foreign keys for SQLite
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Run migrations with dialect
	dialect := database.NewSQLiteDialect()
	if err := database.RunMigrationsWithDialect(db, dialect); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Seed test users and dogs with timestamps
	now := time.Now()

	result, err := db.Exec(`INSERT INTO users (id, name, email, password_hash, experience_level, is_active, is_verified, created_at, last_activity_at, terms_accepted_at)
		VALUES
		(1, 'Green User', 'green@test.com', 'hash', 'green', 1, 1, ?, ?, ?),
		(2, 'Blue User', 'blue@test.com', 'hash', 'blue', 1, 1, ?, ?, ?),
		(3, 'Orange User', 'orange@test.com', 'hash', 'orange', 1, 1, ?, ?, ?)`,
		now, now, now, now, now, now, now, now, now)
	if err != nil {
		t.Fatalf("Failed to seed users: %v", err)
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected != 3 {
		t.Fatalf("Expected 3 users inserted, got %d", rowsAffected)
	}

	// Seed user_colors for the new color system
	// Color IDs: 1=gruen, 2=gelb, 3=orange, 4=hellblau, 5=dunkelblau
	// Green user (ID 1) gets only color 1 (gruen)
	db.Exec(`INSERT INTO user_colors (user_id, color_id) VALUES (1, 1)`)
	// Blue user (ID 2) gets all main colors (1-5)
	db.Exec(`INSERT INTO user_colors (user_id, color_id) VALUES (2, 1), (2, 2), (2, 3), (2, 4), (2, 5)`)
	// Orange user (ID 3) gets colors 1, 2, 3
	db.Exec(`INSERT INTO user_colors (user_id, color_id) VALUES (3, 1), (3, 2), (3, 3)`)

	// Seed dogs with color_id for the new color system
	result, err = db.Exec(`INSERT INTO dogs (id, name, breed, size, age, category, color_id, is_available, created_at, updated_at)
		VALUES
		(1, 'Green Dog', 'Labrador', 'medium', 5, 'green', 1, 1, ?, ?),
		(2, 'Blue Dog', 'German Shepherd', 'large', 6, 'blue', 5, 1, ?, ?),
		(3, 'Orange Dog', 'Husky', 'large', 7, 'orange', 3, 1, ?, ?),
		(4, 'Unavailable Dog', 'Beagle', 'small', 4, 'green', 1, 0, ?, ?)`,
		now, now, now, now, now, now, now, now)
	if err != nil {
		t.Fatalf("Failed to seed dogs: %v", err)
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected != 4 {
		t.Fatalf("Expected 4 dogs inserted, got %d", rowsAffected)
	}

	// Seed system settings
	db.Exec(`INSERT OR REPLACE INTO system_settings (key, value) VALUES
		('booking_advance_days', '14'),
		('cancellation_notice_hours', '12'),
		('auto_deactivation_days', '365')`)

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

// createTestContext creates a context with user authentication
func createTestContext(userID int, isAdmin bool) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, isAdmin)
	ctx = context.WithValue(ctx, middleware.EmailKey, fmt.Sprintf("user%d@test.com", userID))
	return ctx
}

// Test 8.1.1: Basic Booking Creation
func TestRegression_BasicBookingCreation(t *testing.T) {
	db, cleanup := setupRegressionTest(t)
	defer cleanup()

	cfg := &config.Config{}
	handler := NewBookingHandler(db, cfg)

	futureDate := time.Now().AddDate(0, 0, 5).Format("2006-01-02")

	testCases := []struct {
		name         string
		userID       int
		dogID        int
		date         string
		time         string
		expectStatus int
		description  string
	}{
		{
			name:         "TC-8.1.1-A: Create booking (within allowed times)",
			userID:       1,
			dogID:        1,
			date:         futureDate,
			time:         "15:00",
			expectStatus: http.StatusCreated,
			description:  "Should succeed if time is within allowed windows",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create booking
			bookingData := map[string]interface{}{
				"dog_id":         tc.dogID,
				"date":           tc.date,
				"scheduled_time": tc.time,
			}
			body, _ := json.Marshal(bookingData)

			req := httptest.NewRequest("POST", "/api/bookings", bytes.NewBuffer(body))
			req = req.WithContext(createTestContext(tc.userID, false))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.CreateBooking(w, req)

			if w.Code != tc.expectStatus {
				t.Errorf("%s: Expected status %d, got %d. Body: %s",
					tc.description, tc.expectStatus, w.Code, w.Body.String())
			}

			// Verify booking was created
			if w.Code == http.StatusCreated {
				var booking models.Booking
				if err := json.Unmarshal(w.Body.Bytes(), &booking); err != nil {
					t.Errorf("Failed to unmarshal response: %v. Body: %s", err, w.Body.String())
					return
				}

				// Verify booking exists in database
				dbBooking, err := handler.bookingRepo.FindByID(booking.ID)
				if err != nil {
					t.Errorf("Booking should exist in database: %v", err)
				}
				if dbBooking == nil {
					t.Error("Booking should not be nil")
				}
			}
		})
	}
}

// Test 8.1.2: Experience Level Validation
func TestRegression_ExperienceLevelValidation(t *testing.T) {
	db, cleanup := setupRegressionTest(t)
	defer cleanup()

	cfg := &config.Config{}
	handler := NewBookingHandler(db, cfg)

	futureDate := time.Now().AddDate(0, 0, 5).Format("2006-01-02")

	testCases := []struct {
		name         string
		userID       int
		dogID        int
		time         string
		expectStatus int
		description  string
	}{
		{
			name:         "TC-8.1.2-A: Green user books green dog",
			userID:       1, // Green user
			dogID:        1, // Green dog
			time:         "09:00",
			expectStatus: http.StatusCreated,
			description:  "Should succeed with matching level",
		},
		{
			name:         "TC-8.1.2-B: Green user books blue dog",
			userID:       1, // Green user
			dogID:        2, // Blue dog
			time:         "15:00",
			expectStatus: http.StatusForbidden,
			description:  "Should fail - insufficient experience level",
		},
		// New level order: green < orange < blue (blue is highest)
		{
			name:         "TC-8.1.2-C: Blue user books orange dog",
			userID:       2, // Blue user
			dogID:        3, // Orange dog
			time:         "09:00",
			expectStatus: http.StatusCreated,
			description:  "Should succeed - blue is highest level, can access all dogs",
		},
		{
			name:         "TC-8.1.2-D: Blue user books green dog",
			userID:       2, // Blue user
			dogID:        1, // Green dog
			time:         "15:00",
			expectStatus: http.StatusCreated,
			description:  "Should succeed - blue can access green",
		},
		{
			name:         "TC-8.1.2-E: Orange user books orange dog",
			userID:       3, // Orange user
			dogID:        3, // Orange dog
			time:         "15:00",
			expectStatus: http.StatusCreated,
			description:  "Should succeed - orange can access orange",
		},
		{
			name:         "TC-8.1.2-F: Orange user cannot book blue dog",
			userID:       3, // Orange user
			dogID:        2, // Blue dog
			time:         "15:30",
			expectStatus: http.StatusForbidden,
			description:  "Should fail - orange cannot access blue (blue is highest)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bookingData := map[string]interface{}{
				"dog_id":         tc.dogID,
				"date":           futureDate,
				"scheduled_time": tc.time,
			}
			body, _ := json.Marshal(bookingData)

			req := httptest.NewRequest("POST", "/api/bookings", bytes.NewBuffer(body))
			req = req.WithContext(createTestContext(tc.userID, false))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.CreateBooking(w, req)

			if w.Code != tc.expectStatus {
				t.Errorf("%s: Expected status %d, got %d. Body: %s",
					tc.description, tc.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

// Test 8.1.3: Date Restrictions
func TestRegression_DateRestrictions(t *testing.T) {
	db, cleanup := setupRegressionTest(t)
	defer cleanup()

	cfg := &config.Config{}
	handler := NewBookingHandler(db, cfg)

	// Create a blocked date
	blockedDate := time.Now().AddDate(0, 0, 10).Format("2006-01-02")
	if err := handler.blockedDateRepo.Create(&models.BlockedDate{
		Date:      blockedDate,
		Reason:    "Test block",
		CreatedBy: 1, // Admin user
	}); err != nil {
		t.Fatalf("Failed to create blocked date: %v", err)
	}

	// Create an existing booking for double-booking test
	existingBookingDate := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	if err := handler.bookingRepo.Create(&models.Booking{
		UserID:        2,
		DogID:         1,
		Date:          existingBookingDate,
		ScheduledTime: "15:00",
		Status:        "scheduled",
	}); err != nil {
		t.Fatalf("Failed to create existing booking: %v", err)
	}

	testCases := []struct {
		name         string
		userID       int
		dogID        int
		date         string
		time         string
		expectStatus int
		description  string
	}{
		{
			name:         "TC-8.1.3-A: Book date in past",
			userID:       1,
			dogID:        1,
			date:         "2020-01-01",
			time:         "15:00",
			expectStatus: http.StatusBadRequest,
			description:  "Should fail - date in past",
		},
		{
			name:         "TC-8.1.3-B: Book beyond advance limit",
			userID:       1,
			dogID:        1,
			date:         time.Now().AddDate(0, 0, 20).Format("2006-01-02"), // 20 days (limit is 14)
			time:         "15:00",
			expectStatus: http.StatusBadRequest,
			description:  "Should fail - beyond advance booking limit",
		},
		{
			name:         "TC-8.1.3-C: Book on blocked date",
			userID:       1,
			dogID:        1,
			date:         blockedDate,
			time:         "15:00",
			expectStatus: http.StatusBadRequest,
			description:  "Should fail - date is blocked",
		},
		{
			name:         "TC-8.1.3-D: Double booking same dog/date/time",
			userID:       1,
			dogID:        1,
			date:         existingBookingDate,
			time:         "15:00",
			expectStatus: http.StatusConflict,
			description:  "Should fail - dog already booked",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bookingData := map[string]interface{}{
				"dog_id":         tc.dogID,
				"date":           tc.date,
				"scheduled_time": tc.time,
			}
			body, _ := json.Marshal(bookingData)

			req := httptest.NewRequest("POST", "/api/bookings", bytes.NewBuffer(body))
			req = req.WithContext(createTestContext(tc.userID, false))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.CreateBooking(w, req)

			if w.Code != tc.expectStatus {
				t.Errorf("%s: Expected status %d, got %d. Body: %s",
					tc.description, tc.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}

// Test 8.2.2: Blocked Dates Functionality
func TestRegression_BlockedDates(t *testing.T) {
	db, cleanup := setupRegressionTest(t)
	defer cleanup()

	blockedDateRepo := repository.NewBlockedDateRepository(db)

	testCases := []struct {
		name        string
		testFunc    func(*testing.T)
		description string
	}{
		{
			name: "TC-8.2.2-A: Add blocked date",
			testFunc: func(t *testing.T) {
				date := time.Now().AddDate(0, 0, 15).Format("2006-01-02")
				err := blockedDateRepo.Create(&models.BlockedDate{
					Date:      date,
					Reason:    "Staff training",
					CreatedBy: 1, // Admin user
				})
				if err != nil {
					t.Errorf("Should be able to create blocked date: %v", err)
				}

				// Verify it was created
				isBlocked, _ := blockedDateRepo.IsBlocked(date)
				if !isBlocked {
					t.Error("Date should be blocked")
				}
			},
			description: "Should be able to create blocked dates",
		},
		{
			name: "TC-8.2.2-B: Remove blocked date",
			testFunc: func(t *testing.T) {
				date := time.Now().AddDate(0, 0, 16).Format("2006-01-02")
				blockedDateRepo.Create(&models.BlockedDate{
					Date:      date,
					Reason:    "Temporary",
					CreatedBy: 1, // Admin user
				})

				// Find it to get ID
				blockedDateObj, err := blockedDateRepo.FindByDate(date)
				if err != nil {
					t.Fatalf("Could not find blocked date: %v", err)
				}

				// Delete it
				err = blockedDateRepo.Delete(blockedDateObj.ID)
				if err != nil {
					t.Errorf("Should be able to delete blocked date: %v", err)
				}

				// Verify it was deleted
				isBlocked, _ := blockedDateRepo.IsBlocked(date)
				if isBlocked {
					t.Error("Date should not be blocked after deletion")
				}
			},
			description: "Should be able to remove blocked dates",
		},
		{
			name: "TC-8.2.2-C: View blocked dates",
			testFunc: func(t *testing.T) {
				// Create some blocked dates
				date1 := time.Now().AddDate(0, 0, 17).Format("2006-01-02")
				date2 := time.Now().AddDate(0, 0, 18).Format("2006-01-02")

				blockedDateRepo.Create(&models.BlockedDate{Date: date1, Reason: "Reason 1", CreatedBy: 1})
				blockedDateRepo.Create(&models.BlockedDate{Date: date2, Reason: "Reason 2", CreatedBy: 1})

				// Get all blocked dates
				blockedDates, err := blockedDateRepo.FindAll()
				if err != nil {
					t.Errorf("Should be able to retrieve blocked dates: %v", err)
				}

				if len(blockedDates) < 2 {
					t.Errorf("Expected at least 2 blocked dates, got %d", len(blockedDates))
				}
			},
			description: "Should be able to view all blocked dates",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.testFunc(t)
		})
	}
}

// Test 8.3.1: User Dashboard Functionality
func TestRegression_UserDashboard(t *testing.T) {
	db, cleanup := setupRegressionTest(t)
	defer cleanup()

	bookingRepo := repository.NewBookingRepository(db)

	// Create test bookings for user
	futureDate := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	pastDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02")

	// Upcoming booking
	bookingRepo.Create(&models.Booking{
		UserID:         1,
		DogID:          1,
		Date:           futureDate,
		ScheduledTime:  "15:00",
		Status:         "scheduled",
		ApprovalStatus: "approved",
	})

	// Completed booking
	bookingRepo.Create(&models.Booking{
		UserID:         1,
		DogID:          1,
		Date:           pastDate,
		ScheduledTime:  "15:00",
		Status:         "completed",
		ApprovalStatus: "approved",
	})

	testCases := []struct {
		name        string
		testFunc    func(*testing.T)
		description string
	}{
		{
			name: "TC-8.3.1-A: View upcoming bookings",
			testFunc: func(t *testing.T) {
				bookings, err := bookingRepo.GetUpcoming(1, 10)
				if err != nil {
					t.Errorf("Should be able to get upcoming bookings: %v", err)
				}
				if len(bookings) < 1 {
					t.Error("Should have at least 1 upcoming booking")
				}
			},
			description: "User should see upcoming bookings with approval status",
		},
		{
			name: "TC-8.3.1-B: Cancel booking",
			testFunc: func(t *testing.T) {
				// Get the booking
				bookings, _ := bookingRepo.GetUpcoming(1, 10)
				if len(bookings) == 0 {
					t.Skip("No bookings to cancel")
				}

				bookingID := bookings[0].ID
				err := bookingRepo.Cancel(bookingID, nil)
				if err != nil {
					t.Errorf("Should be able to cancel booking: %v", err)
				}

				// Verify cancellation
				booking, _ := bookingRepo.FindByID(bookingID)
				if booking.Status != "cancelled" {
					t.Error("Booking should be cancelled")
				}
			},
			description: "User should be able to cancel their bookings",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.testFunc(t)
		})
	}
}

// Test 8.3.2: Dog Browsing Functionality
func TestRegression_DogBrowsing(t *testing.T) {
	db, cleanup := setupRegressionTest(t)
	defer cleanup()

	dogRepo := repository.NewDogRepository(db)

	testCases := []struct {
		name        string
		testFunc    func(*testing.T)
		description string
	}{
		{
			name: "TC-8.3.2-A: View all dogs",
			testFunc: func(t *testing.T) {
				dogs, err := dogRepo.FindAll(&models.DogFilterRequest{})
				if err != nil {
					t.Errorf("Should be able to get all dogs: %v", err)
				}
				if len(dogs) < 4 {
					t.Errorf("Expected at least 4 dogs, got %d", len(dogs))
				}
			},
			description: "User should be able to view all dogs",
		},
		{
			name: "TC-8.3.2-B: Filter by experience level",
			testFunc: func(t *testing.T) {
				// Test can access dog based on level
				userLevel := "green"
				dogCategory := "green"
				canAccess := repository.CanUserAccessDog(userLevel, dogCategory)
				if !canAccess {
					t.Error("Green user should be able to access green dog")
				}

				// Test cannot access higher level
				dogCategory = "blue"
				canAccess = repository.CanUserAccessDog(userLevel, dogCategory)
				if canAccess {
					t.Error("Green user should NOT be able to access blue dog")
				}
			},
			description: "Experience level filtering should still work",
		},
		{
			name: "TC-8.3.2-C: View dog details",
			testFunc: func(t *testing.T) {
				dog, err := dogRepo.FindByID(1)
				if err != nil {
					t.Errorf("Should be able to get dog details: %v", err)
				}
				if dog == nil {
					t.Error("Dog should not be nil")
				}
				if dog.Name != "Green Dog" {
					t.Errorf("Expected 'Green Dog', got '%s'", dog.Name)
				}
			},
			description: "User should be able to view dog details",
		},
		{
			name: "TC-8.3.2-D: Check dog availability",
			testFunc: func(t *testing.T) {
				// Available dog
				dog, _ := dogRepo.FindByID(1)
				if !dog.IsAvailable {
					t.Error("Dog 1 should be available")
				}

				// Unavailable dog
				dog, _ = dogRepo.FindByID(4)
				if dog.IsAvailable {
					t.Error("Dog 4 should be unavailable")
				}
			},
			description: "Dog availability should be correctly reported",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.testFunc(t)
		})
	}
}
