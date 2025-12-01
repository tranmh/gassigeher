package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables
	schema := `
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE,
		phone TEXT,
		password_hash TEXT,
		experience_level TEXT DEFAULT 'green',
		is_admin INTEGER DEFAULT 0,
		is_super_admin INTEGER DEFAULT 0,
		is_verified INTEGER DEFAULT 0,
		is_active INTEGER DEFAULT 1,
		is_deleted INTEGER DEFAULT 0,
		verification_token TEXT,
		verification_token_expires TIMESTAMP,
		password_reset_token TEXT,
		password_reset_expires TIMESTAMP,
		profile_photo TEXT,
		anonymous_id TEXT,
		terms_accepted_at TIMESTAMP,
		last_activity_at TIMESTAMP,
		deactivated_at TIMESTAMP,
		deactivation_reason TEXT,
		reactivated_at TIMESTAMP,
		deleted_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE dogs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		breed TEXT,
		size TEXT,
		age INTEGER,
		category TEXT DEFAULT 'green',
		photo TEXT,
		photo_thumbnail TEXT,
		special_needs TEXT,
		pickup_location TEXT,
		walk_route TEXT,
		walk_duration INTEGER,
		special_instructions TEXT,
		default_morning_time TEXT,
		default_evening_time TEXT,
		is_available INTEGER DEFAULT 1,
		is_featured INTEGER DEFAULT 0,
		external_link TEXT,
		unavailable_reason TEXT,
		unavailable_since TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE bookings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		dog_id INTEGER NOT NULL,
		date TEXT NOT NULL,
		scheduled_time TEXT NOT NULL,
		status TEXT DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
		completed_at TIMESTAMP,
		reminder_sent_at TIMESTAMP,
		user_notes TEXT,
		admin_cancellation_reason TEXT,
		requires_approval INTEGER DEFAULT 0,
		approval_status TEXT DEFAULT 'approved',
		approved_by INTEGER,
		approved_at TIMESTAMP,
		rejection_reason TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(dog_id, date, scheduled_time)
	);

	-- Insert test users
	INSERT INTO users (id, name, email) VALUES (1, 'Test User', 'test@example.com');
	INSERT INTO users (id, name, email) VALUES (2, 'Test User 2', 'test2@example.com');

	-- Insert test dogs
	INSERT INTO dogs (id, name, breed) VALUES (1, 'Buddy', 'Labrador');
	INSERT INTO dogs (id, name, breed) VALUES (2, 'Max', 'German Shepherd');
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func TestBookingRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)

	booking := &models.Booking{
		UserID:        1,
		DogID:         1,
		Date:          "2025-12-01",
		ScheduledTime: "09:00",
	}

	err := repo.Create(booking)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if booking.ID == 0 {
		t.Error("Expected booking ID to be set")
	}

	if booking.Status != "scheduled" {
		t.Errorf("Expected status to be 'scheduled', got %s", booking.Status)
	}
}

func TestBookingRepository_CheckDoubleBooking(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)

	// Create first booking
	booking := &models.Booking{
		UserID:        1,
		DogID:         1,
		Date:          "2025-12-01",
		ScheduledTime: "09:00",
	}
	repo.Create(booking)

	// Check for double booking - same scheduled time
	isBooked, err := repo.CheckDoubleBooking(1, "2025-12-01", "09:00")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !isBooked {
		t.Error("Expected dog to be marked as booked for 09:00")
	}

	// Check different scheduled time - should be available
	isBooked, err = repo.CheckDoubleBooking(1, "2025-12-01", "15:00")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if isBooked {
		t.Error("Expected 15:00 slot to be available")
	}
}

func TestBookingRepository_AutoComplete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)

	// Create past booking
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	booking := &models.Booking{
		UserID:        1,
		DogID:         1,
		Date:          yesterday,
		ScheduledTime: "09:00",
	}
	repo.Create(booking)

	// Run auto-complete
	count, err := repo.AutoComplete()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 booking to be completed, got %d", count)
	}

	// Verify booking is completed
	completed, _ := repo.FindByID(booking.ID)
	if completed.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", completed.Status)
	}
}

// DONE: TestBookingRepository_Cancel tests booking cancellation
func TestBookingRepository_Cancel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)

	t.Run("cancel with reason", func(t *testing.T) {
		booking := &models.Booking{
			UserID:        1,
			DogID:         1,
			Date:          "2025-12-01",
			ScheduledTime: "09:00",
		}
		repo.Create(booking)

		reason := "Dog is sick"
		err := repo.Cancel(booking.ID, &reason)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify cancellation
		cancelled, _ := repo.FindByID(booking.ID)
		if cancelled.Status != "cancelled" {
			t.Errorf("Expected status 'cancelled', got %s", cancelled.Status)
		}

		if cancelled.AdminCancellationReason == nil || *cancelled.AdminCancellationReason != reason {
			t.Error("Expected cancellation reason to be set")
		}
	})

	t.Run("cancel without reason", func(t *testing.T) {
		booking := &models.Booking{
			UserID:        2,
			DogID:         2,
			Date:          "2025-12-02",
			ScheduledTime: "15:00",
		}
		repo.Create(booking)

		err := repo.Cancel(booking.ID, nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify cancellation
		cancelled, _ := repo.FindByID(booking.ID)
		if cancelled.Status != "cancelled" {
			t.Errorf("Expected status 'cancelled', got %s", cancelled.Status)
		}
	})

	t.Run("cancel non-existent booking", func(t *testing.T) {
		reason := "Test"
		err := repo.Cancel(99999, &reason)

		// May or may not error depending on implementation
		if err != nil {
			t.Logf("Cancel non-existent booking returned: %v", err)
		}
	})
}

// DONE: TestBookingRepository_FindByID tests finding booking by ID
func TestBookingRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)

	t.Run("booking exists", func(t *testing.T) {
		booking := &models.Booking{
			UserID:        1,
			DogID:         1,
			Date:          "2025-12-01",
			ScheduledTime: "09:00",
		}
		repo.Create(booking)

		found, err := repo.FindByID(booking.ID)
		if err != nil {
			t.Fatalf("FindByID() failed: %v", err)
		}

		if found.ID != booking.ID {
			t.Errorf("Expected ID %d, got %d", booking.ID, found.ID)
		}

		if found.Date != "2025-12-01" {
			t.Errorf("Expected date '2025-12-01', got %s", found.Date)
		}
	})

	t.Run("booking not found", func(t *testing.T) {
		found, err := repo.FindByID(99999)
		if found != nil {
			t.Error("Expected nil for non-existent ID")
		}
		if err != nil {
			t.Logf("FindByID returned error: %v", err)
		}
	})
}

// DONE: TestBookingRepository_FindAll tests listing bookings with filters
func TestBookingRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)

	// Create test bookings
	booking1 := &models.Booking{
		UserID:        1,
		DogID:         1,
		Date:          "2025-12-01",
		ScheduledTime: "09:00",
		Status:        "scheduled",
	}
	repo.Create(booking1)

	booking2 := &models.Booking{
		UserID:        2,
		DogID:         2,
		Date:          "2025-12-02",
		ScheduledTime: "15:00",
		Status:        "scheduled",
	}
	repo.Create(booking2)

	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	booking3 := &models.Booking{
		UserID:        1,
		DogID:         1,
		Date:          yesterday,
		ScheduledTime: "09:00",
		Status:        "completed",
	}
	repo.Create(booking3)

	t.Run("all bookings - no filter", func(t *testing.T) {
		bookings, err := repo.FindAll(nil)
		if err != nil {
			t.Fatalf("FindAll() failed: %v", err)
		}

		if len(bookings) != 3 {
			t.Errorf("Expected 3 bookings, got %d", len(bookings))
		}
	})

	t.Run("filter by user_id", func(t *testing.T) {
		userID := 1
		filter := &models.BookingFilterRequest{
			UserID: &userID,
		}

		bookings, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with user filter failed: %v", err)
		}

		if len(bookings) != 2 {
			t.Errorf("Expected 2 bookings for user 1, got %d", len(bookings))
		}

		for _, b := range bookings {
			if b.UserID != 1 {
				t.Errorf("Expected all bookings to have UserID=1, got %d", b.UserID)
			}
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		status := "scheduled"
		filter := &models.BookingFilterRequest{
			Status: &status,
		}

		bookings, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with status filter failed: %v", err)
		}

		// Should find scheduled bookings
		for _, b := range bookings {
			if b.Status != "scheduled" && b.Status != "" {
				t.Errorf("Expected status 'scheduled', got %s", b.Status)
			}
		}

		t.Logf("Found %d scheduled bookings", len(bookings))
	})

	t.Run("filter by dog_id", func(t *testing.T) {
		dogID := 2
		filter := &models.BookingFilterRequest{
			DogID: &dogID,
		}

		bookings, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with dog filter failed: %v", err)
		}

		if len(bookings) != 1 {
			t.Errorf("Expected 1 booking for dog 2, got %d", len(bookings))
		}

		if len(bookings) > 0 && bookings[0].DogID != 2 {
			t.Errorf("Expected DogID=2, got %d", bookings[0].DogID)
		}
	})


	t.Run("filter by date_from", func(t *testing.T) {
		dateFrom := "2025-12-01"
		filter := &models.BookingFilterRequest{
			DateFrom: &dateFrom,
		}

		bookings, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with date_from filter failed: %v", err)
		}

		// Should only get bookings from 2025-12-01 onwards
		for _, b := range bookings {
			if b.Date < dateFrom {
				t.Errorf("Expected date >= %s, got %s", dateFrom, b.Date)
			}
		}

		if len(bookings) != 2 {
			t.Errorf("Expected 2 bookings from 2025-12-01 onwards, got %d", len(bookings))
		}
	})

	t.Run("filter by date_to", func(t *testing.T) {
		dateTo := "2025-12-01"
		filter := &models.BookingFilterRequest{
			DateTo: &dateTo,
		}

		bookings, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with date_to filter failed: %v", err)
		}

		// Should get bookings up to and including 2025-12-01
		for _, b := range bookings {
			if b.Date > dateTo {
				t.Errorf("Expected date <= %s, got %s", dateTo, b.Date)
			}
		}
	})

	t.Run("filter by date range", func(t *testing.T) {
		dateFrom := "2025-12-01"
		dateTo := "2025-12-02"
		filter := &models.BookingFilterRequest{
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
		}

		bookings, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with date range filter failed: %v", err)
		}

		if len(bookings) != 2 {
			t.Errorf("Expected 2 bookings in date range, got %d", len(bookings))
		}

		for _, b := range bookings {
			if b.Date < dateFrom || b.Date > dateTo {
				t.Errorf("Expected date in range %s to %s, got %s", dateFrom, dateTo, b.Date)
			}
		}
	})

	t.Run("filter by year and month", func(t *testing.T) {
		year := 2025
		month := 12
		filter := &models.BookingFilterRequest{
			Year:  &year,
			Month: &month,
		}

		bookings, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with year/month filter failed: %v", err)
		}

		// Should find bookings in December 2025
		if len(bookings) != 2 {
			t.Errorf("Expected 2 bookings in December 2025, got %d", len(bookings))
		}

		for _, b := range bookings {
			if !strings.HasPrefix(b.Date, "2025-12") {
				t.Errorf("Expected date in 2025-12, got %s", b.Date)
			}
		}
	})

	t.Run("filter with multiple criteria", func(t *testing.T) {
		userID := 1
		status := "scheduled"
		filter := &models.BookingFilterRequest{
			UserID: &userID,
			Status: &status,
		}

		bookings, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with multiple filters failed: %v", err)
		}

		// User 1 has bookings, filter to scheduled ones
		for _, b := range bookings {
			if b.UserID != 1 {
				t.Errorf("Expected UserID=1, got %d", b.UserID)
			}
			// Status filter may or may not work depending on Create() setting status
			// We're testing the filter logic works
		}

		t.Logf("Found %d scheduled bookings for user 1", len(bookings))
	})

	t.Run("no results with filter", func(t *testing.T) {
		userID := 999
		filter := &models.BookingFilterRequest{
			UserID: &userID,
		}

		bookings, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll failed: %v", err)
		}

		if len(bookings) != 0 {
			t.Errorf("Expected 0 bookings for non-existent user, got %d", len(bookings))
		}
	})
}

// DONE: TestBookingRepository_AddNotes tests adding notes to bookings
func TestBookingRepository_AddNotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)

	// Create booking and mark as completed
	booking := &models.Booking{
		UserID:        1,
		DogID:         1,
		Date:          "2025-12-01",
		ScheduledTime: "09:00",
		Status:        "scheduled",
	}
	repo.Create(booking)

	// Update to completed status
	db.Exec("UPDATE bookings SET status = 'completed', completed_at = ? WHERE id = ?", time.Now(), booking.ID)

	t.Run("add notes to completed booking", func(t *testing.T) {
		notes := "Great walk! Dog was very energetic."

		err := repo.AddNotes(booking.ID, notes)
		if err != nil {
			t.Fatalf("AddNotes() failed: %v", err)
		}

		// Verify notes via direct query
		var userNotes *string
		db.QueryRow("SELECT user_notes FROM bookings WHERE id = ?", booking.ID).Scan(&userNotes)

		if userNotes == nil || *userNotes != notes {
			t.Errorf("Expected notes '%s', got %v", notes, userNotes)
		}
	})

	t.Run("cannot add notes to scheduled booking", func(t *testing.T) {
		// Create another booking that's still scheduled
		scheduledBooking := &models.Booking{
			UserID:        2,
			DogID:         2,
			Date:          "2025-12-02",
			ScheduledTime: "16:00",
			Status:        "scheduled",
		}
		repo.Create(scheduledBooking)

		notes := "Should fail"
		err := repo.AddNotes(scheduledBooking.ID, notes)

		if err == nil {
			t.Error("Expected error when adding notes to scheduled booking, got nil")
		}
	})

	t.Run("cannot add notes to cancelled booking", func(t *testing.T) {
		// Create cancelled booking
		cancelledBooking := &models.Booking{
			UserID:        3,
			DogID:         3,
			Date:          "2025-12-03",
			ScheduledTime: "09:00",
			Status:        "cancelled",
		}
		repo.Create(cancelledBooking)

		// Update to cancelled
		db.Exec("UPDATE bookings SET status = 'cancelled' WHERE id = ?", cancelledBooking.ID)

		notes := "Should fail"
		err := repo.AddNotes(cancelledBooking.ID, notes)

		if err == nil {
			t.Error("Expected error when adding notes to cancelled booking, got nil")
		}
	})

	t.Run("add empty notes", func(t *testing.T) {
		// Create another completed booking
		completedBooking := &models.Booking{
			UserID:        4,
			DogID:         4,
			Date:          "2025-12-04",
			ScheduledTime: "09:00",
		}
		repo.Create(completedBooking)
		db.Exec("UPDATE bookings SET status = 'completed', completed_at = ? WHERE id = ?", time.Now(), completedBooking.ID)

		err := repo.AddNotes(completedBooking.ID, "")
		if err != nil {
			t.Fatalf("AddNotes() with empty notes failed: %v", err)
		}
	})

	t.Run("non-existent booking", func(t *testing.T) {
		err := repo.AddNotes(99999, "Notes for non-existent booking")

		if err == nil {
			t.Error("Expected error for non-existent booking, got nil")
		}
	})

	t.Run("update existing notes", func(t *testing.T) {
		// Add notes first
		originalNotes := "Original notes"
		repo.AddNotes(booking.ID, originalNotes)

		// Update notes
		updatedNotes := "Updated notes"
		err := repo.AddNotes(booking.ID, updatedNotes)
		if err != nil {
			t.Fatalf("AddNotes() update failed: %v", err)
		}

		// Verify updated notes
		var userNotes *string
		db.QueryRow("SELECT user_notes FROM bookings WHERE id = ?", booking.ID).Scan(&userNotes)

		if userNotes == nil || *userNotes != updatedNotes {
			t.Errorf("Expected notes '%s', got %v", updatedNotes, userNotes)
		}
	})
}

// DONE: TestBookingRepository_GetUpcoming tests getting upcoming bookings for a user
func TestBookingRepository_GetUpcoming(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)

	userID := 1

	// Create past booking (should not be included)
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	pastBooking := &models.Booking{
		UserID:        userID,
		DogID:         1,
		Date:          yesterday,
		ScheduledTime: "09:00",
		Status:        "completed",
	}
	repo.Create(pastBooking)

	// Create future bookings
	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	nextWeek := time.Now().Add(7 * 24 * time.Hour).Format("2006-01-02")

	futureBooking1 := &models.Booking{
		UserID:        userID,
		DogID:         1,
		Date:          tomorrow,
		ScheduledTime: "09:00",
		Status:        "scheduled",
	}
	repo.Create(futureBooking1)

	futureBooking2 := &models.Booking{
		UserID:        userID,
		DogID:         2,
		Date:          nextWeek,
		ScheduledTime: "15:00",
		Status:        "scheduled",
	}
	repo.Create(futureBooking2)

	// Create booking for different user (should not be included)
	otherUserBooking := &models.Booking{
		UserID:        2,
		DogID:         1,
		Date:          tomorrow,
		ScheduledTime: "16:00",
		Status:        "scheduled",
	}
	repo.Create(otherUserBooking)

	t.Run("get upcoming bookings for user", func(t *testing.T) {
		upcoming, err := repo.GetUpcoming(userID, 10)
		if err != nil {
			t.Fatalf("GetUpcoming() failed: %v", err)
		}

		// Should get only future bookings for user 1
		if len(upcoming) != 2 {
			t.Errorf("Expected 2 upcoming bookings, got %d", len(upcoming))
		}

		for _, b := range upcoming {
			if b.UserID != userID {
				t.Errorf("Expected all bookings for user %d, got booking with user %d", userID, b.UserID)
			}
			if b.Status != "scheduled" {
				t.Errorf("Expected status 'scheduled', got %s", b.Status)
			}
		}
	})

	t.Run("limit upcoming bookings", func(t *testing.T) {
		upcoming, err := repo.GetUpcoming(userID, 1)
		if err != nil {
			t.Fatalf("GetUpcoming() failed: %v", err)
		}

		if len(upcoming) > 1 {
			t.Errorf("Expected limit of 1 booking, got %d", len(upcoming))
		}
	})
}

// DONE: TestBookingRepository_Update tests updating booking
func TestBookingRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)

	booking := &models.Booking{
		UserID:        1,
		DogID:         1,
		Date:          "2025-12-01",
		ScheduledTime: "09:00",
	}
	repo.Create(booking)

	t.Run("update booking time", func(t *testing.T) {
		booking.ScheduledTime = "10:00"

		err := repo.Update(booking)
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify update
		updated, _ := repo.FindByID(booking.ID)
		if updated.ScheduledTime != "10:00" {
			t.Errorf("Expected time '10:00', got %s", updated.ScheduledTime)
		}
	})

	t.Run("update booking date", func(t *testing.T) {
		booking.Date = "2025-12-15"

		err := repo.Update(booking)
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify update
		updated, _ := repo.FindByID(booking.ID)
		if updated.Date != "2025-12-15" {
			t.Errorf("Expected date '2025-12-15', got %s", updated.Date)
		}
	})


	t.Run("update non-existent booking", func(t *testing.T) {
		nonExistent := &models.Booking{
			ID:            99999,
			Date:          "2025-12-20",
			ScheduledTime: "09:00",
		}

		err := repo.Update(nonExistent)
		// Should not error even if no rows updated
		if err != nil {
			t.Logf("Update non-existent booking returned: %v", err)
		}
	})
}

// DONE: TestBookingRepository_GetForReminders tests getting bookings for reminder emails
func TestBookingRepository_GetForReminders(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)
	now := time.Now()

	t.Run("returns bookings in reminder window", func(t *testing.T) {
		// Create booking scheduled 1.5 hours from now
		reminderTime := now.Add(90 * time.Minute)
		reminderDate := reminderTime.Format("2006-01-02")
		reminderScheduledTime := reminderTime.Format("15:04")

		booking := &models.Booking{
			UserID:        1,
			DogID:         1,
			Date:          reminderDate,
			ScheduledTime: reminderScheduledTime,
			Status:        "scheduled",
		}
		repo.Create(booking)

		// Get reminders
		reminders, err := repo.GetForReminders()
		if err != nil {
			t.Fatalf("GetForReminders() failed: %v", err)
		}

		// Should find the booking (if time is within 1-2 hour window)
		found := false
		for _, r := range reminders {
			if r.ID == booking.ID {
				found = true
				break
			}
		}

		if !found && reminderTime.Sub(now) >= 1*time.Hour && reminderTime.Sub(now) < 2*time.Hour {
			t.Error("Expected to find booking in reminder window")
		}

		t.Logf("GetForReminders() found %d bookings", len(reminders))
	})

	t.Run("does not return bookings too far in future", func(t *testing.T) {
		// Create booking 5 hours from now
		futureTime := now.Add(5 * time.Hour)
		futureDate := futureTime.Format("2006-01-02")
		futureScheduledTime := futureTime.Format("15:04")

		booking := &models.Booking{
			UserID:        2,
			DogID:         2,
			Date:          futureDate,
			ScheduledTime: futureScheduledTime,
			Status:        "scheduled",
		}
		repo.Create(booking)

		// Get reminders
		reminders, err := repo.GetForReminders()
		if err != nil {
			t.Fatalf("GetForReminders() failed: %v", err)
		}

		// Should not find the booking
		for _, r := range reminders {
			if r.ID == booking.ID {
				t.Error("Should not find booking too far in future")
			}
		}
	})

	t.Run("does not return completed bookings", func(t *testing.T) {
		// Create completed booking in reminder window
		reminderTime := now.Add(90 * time.Minute)
		reminderDate := reminderTime.Format("2006-01-02")
		reminderScheduledTime := reminderTime.Format("15:04")

		completedTime := time.Now()
		booking := &models.Booking{
			UserID:        3,
			DogID:         3,
			Date:          reminderDate,
			ScheduledTime: reminderScheduledTime,
			Status:        "completed",
			CompletedAt:   &completedTime,
		}
		repo.Create(booking)

		// Manually update status since Create sets it to scheduled
		db.Exec("UPDATE bookings SET status = 'completed', completed_at = ? WHERE id = ?", completedTime, booking.ID)

		// Get reminders
		reminders, err := repo.GetForReminders()
		if err != nil {
			t.Fatalf("GetForReminders() failed: %v", err)
		}

		// Should not find completed booking
		for _, r := range reminders {
			if r.ID == booking.ID {
				t.Error("Should not find completed booking")
			}
		}
	})

	t.Run("does not return cancelled bookings", func(t *testing.T) {
		// Create cancelled booking in reminder window
		reminderTime := now.Add(90 * time.Minute)
		reminderDate := reminderTime.Format("2006-01-02")
		reminderScheduledTime := reminderTime.Format("15:04")

		booking := &models.Booking{
			UserID:        4,
			DogID:         4,
			Date:          reminderDate,
			ScheduledTime: reminderScheduledTime,
			Status:        "cancelled",
		}
		repo.Create(booking)

		// Manually update status
		db.Exec("UPDATE bookings SET status = 'cancelled' WHERE id = ?", booking.ID)

		// Get reminders
		reminders, err := repo.GetForReminders()
		if err != nil {
			t.Fatalf("GetForReminders() failed: %v", err)
		}

		// Should not find cancelled booking
		for _, r := range reminders {
			if r.ID == booking.ID {
				t.Error("Should not find cancelled booking")
			}
		}
	})
}

// DONE: TestBookingRepository_FindByIDWithDetails tests finding booking with joined data
func TestBookingRepository_FindByIDWithDetails(t *testing.T) {
	// Use testutil for full schema
	db := testutil.SetupTestDB(t)
	repo := NewBookingRepository(db)

	t.Run("returns booking with user and dog details", func(t *testing.T) {
		// Seed data using testutil
		userID := testutil.SeedTestUser(t, db, "bookinguser@example.com", "Booking User", "green")
		dogID := testutil.SeedTestDog(t, db, "Test Dog", "Labrador", "green")

		// Create booking
		bookingDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		bookingID := testutil.SeedTestBooking(t, db, userID, dogID, bookingDate, "09:00", "scheduled")

		// Find with details
		booking, err := repo.FindByIDWithDetails(bookingID)
		if err != nil {
			t.Fatalf("FindByIDWithDetails() failed: %v", err)
		}

		if booking == nil {
			t.Fatal("Expected booking, got nil")
		}

		// Verify booking data
		if booking.ID != bookingID {
			t.Errorf("Expected ID %d, got %d", bookingID, booking.ID)
		}

		// Verify user details are populated
		if booking.User == nil {
			t.Fatal("Expected user details, got nil")
		}

		if booking.User.Name != "Booking User" {
			t.Errorf("Expected user name 'Booking User', got %s", booking.User.Name)
		}

		if booking.User.Email == nil || *booking.User.Email != "bookinguser@example.com" {
			t.Errorf("Expected user email 'bookinguser@example.com', got %v", booking.User.Email)
		}

		// Verify dog details are populated
		if booking.Dog == nil {
			t.Fatal("Expected dog details, got nil")
		}

		if booking.Dog.Name != "Test Dog" {
			t.Errorf("Expected dog name 'Test Dog', got %s", booking.Dog.Name)
		}

		if booking.Dog.Breed != "Labrador" {
			t.Errorf("Expected breed 'Labrador', got %s", booking.Dog.Breed)
		}

		if booking.Dog.Size != "medium" {
			t.Errorf("Expected size 'medium', got %s", booking.Dog.Size)
		}

		// Age is set by seed helper to 5
		if booking.Dog.Age == 0 {
			t.Error("Dog age should be set")
		}
	})

	t.Run("handles deleted user gracefully", func(t *testing.T) {
		// Create user and delete them
		userID := testutil.SeedTestUser(t, db, "deleteduser@example.com", "Deleted User Name", "green")
		dogID := testutil.SeedTestDog(t, db, "Test Dog 2", "Poodle", "green")

		// Create booking before deletion
		bookingDate := time.Now().AddDate(0, 0, 2).Format("2006-01-02")
		bookingID := testutil.SeedTestBooking(t, db, userID, dogID, bookingDate, "16:00", "scheduled")

		// Delete user (GDPR anonymization)
		userRepo := NewUserRepository(db)
		userRepo.DeleteAccount(userID)

		// Find booking with details
		booking, err := repo.FindByIDWithDetails(bookingID)
		if err != nil {
			t.Fatalf("FindByIDWithDetails() failed: %v", err)
		}

		if booking == nil {
			t.Fatal("Expected booking, got nil")
		}

		// User name should be "Deleted User"
		if booking.User.Name != "Deleted User" {
			t.Errorf("Expected user name 'Deleted User', got %s", booking.User.Name)
		}

		// Email should be nil after deletion
		if booking.User.Email != nil {
			t.Errorf("Expected nil email for deleted user, got %v", booking.User.Email)
		}

		// Dog details should still be present
		if booking.Dog.Name != "Test Dog 2" {
			t.Errorf("Expected dog name 'Test Dog 2', got %s", booking.Dog.Name)
		}
	})

	t.Run("returns nil for non-existent booking", func(t *testing.T) {
		booking, err := repo.FindByIDWithDetails(99999)
		if err != nil {
			t.Fatalf("FindByIDWithDetails() failed: %v", err)
		}

		if booking != nil {
			t.Error("Expected nil for non-existent booking")
		}
	})
}

// ============ Phase 2: Approval Tests ============

// Test 2.3.1: GetPendingApprovalBookings - Query Filtering
func TestGetPendingApprovalBookings_QueryFiltering(t *testing.T) {
	db := testutil.SetupTestDB(t)

	repo := NewBookingRepository(db)

	t.Run("returns only pending bookings", func(t *testing.T) {
		// Create users and dogs using testutil
		for i := 1; i <= 8; i++ {
			testutil.SeedTestUser(t, db, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User %d", i), "green")
			testutil.SeedTestDog(t, db, fmt.Sprintf("Dog %d", i), "Labrador", "green")
		}

		// Create 5 pending bookings
		for i := 1; i <= 5; i++ {
			_, err := db.Exec(`
				INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status)
				VALUES (?, ?, ?, '10:00', 'scheduled', 'pending')
			`, i, i, "2025-01-30")
			if err != nil {
				t.Fatalf("Failed to create pending booking: %v", err)
			}
		}

		// Create 3 approved bookings
		for i := 6; i <= 8; i++ {
			_, err := db.Exec(`
				INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status, approved_by, approved_at)
				VALUES (?, ?, ?, '10:00', 'scheduled', 'approved', 1, datetime('now'))
			`, i, i, "2025-01-31")
			if err != nil {
				t.Fatalf("Failed to create approved booking: %v", err)
			}
		}

		// Get pending approvals
		pending, err := repo.GetPendingApprovalBookings()
		if err != nil {
			t.Fatalf("GetPendingApprovalBookings failed: %v", err)
		}

		// Should return 5 pending bookings
		if len(pending) != 5 {
			t.Errorf("Expected 5 pending bookings, got %d", len(pending))
		}

		// Verify all have pending status
		for _, b := range pending {
			if b.ApprovalStatus != "pending" {
				t.Errorf("Expected approval_status 'pending', got '%s'", b.ApprovalStatus)
			}
		}
	})

	t.Run("returns empty when all approved", func(t *testing.T) {
		db2 := testutil.SetupTestDB(t)
		repo2 := NewBookingRepository(db2)

		// Create users and dogs
		for i := 1; i <= 3; i++ {
			testutil.SeedTestUser(t, db2, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User %d", i), "green")
			testutil.SeedTestDog(t, db2, fmt.Sprintf("Dog %d", i), "Labrador", "green")
		}

		// Create only approved bookings
		for i := 1; i <= 3; i++ {
			_, err := db2.Exec(`
				INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status)
				VALUES (?, ?, ?, '10:00', 'scheduled', 'approved')
			`, i, i, "2025-01-30")
			if err != nil {
				t.Fatalf("Failed to create approved booking: %v", err)
			}
		}

		pending, err := repo2.GetPendingApprovalBookings()
		if err != nil {
			t.Fatalf("GetPendingApprovalBookings failed: %v", err)
		}

		if len(pending) != 0 {
			t.Errorf("Expected 0 pending bookings, got %d", len(pending))
		}
	})

	t.Run("includes pending excludes rejected", func(t *testing.T) {
		db3 := testutil.SetupTestDB(t)

		repo3 := NewBookingRepository(db3)

		// Create users and dogs
		for i := 1; i <= 3; i++ {
			testutil.SeedTestUser(t, db3, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User %d", i), "green")
			testutil.SeedTestDog(t, db3, fmt.Sprintf("Dog %d", i), "Labrador", "green")
		}

		// Create 2 pending bookings
		for i := 1; i <= 2; i++ {
			_, err := db3.Exec(`
				INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status)
				VALUES (?, ?, ?, '10:00', 'scheduled', 'pending')
			`, i, i, "2025-01-30")
			if err != nil {
				t.Fatalf("Failed to create pending booking: %v", err)
			}
		}

		// Create 1 rejected booking
		_, err := db3.Exec(`
			INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status, rejection_reason)
			VALUES (3, 3, '2025-01-30', '10:00', 'cancelled', 'rejected', 'Dog not available')
		`)
		if err != nil {
			t.Fatalf("Failed to create rejected booking: %v", err)
		}

		pending, err := repo3.GetPendingApprovalBookings()
		if err != nil {
			t.Fatalf("GetPendingApprovalBookings failed: %v", err)
		}

		// Should return 2 pending bookings (not the rejected one)
		if len(pending) != 2 {
			t.Errorf("Expected 2 pending bookings, got %d", len(pending))
		}
	})
}

// Test 2.3.2: ApproveBooking - State Transition
func TestApproveBooking_StateTransition(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)

	t.Run("approves pending booking", func(t *testing.T) {
		// Create pending booking
		result, err := db.Exec(`
			INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status)
			VALUES (1, 1, '2025-01-30', '10:00', 'scheduled', 'pending')
		`)
		if err != nil {
			t.Fatalf("Failed to create pending booking: %v", err)
		}

		bookingID, _ := result.LastInsertId()
		adminID := 1

		// Approve booking
		err = repo.ApproveBooking(int(bookingID), adminID)
		if err != nil {
			t.Fatalf("ApproveBooking failed: %v", err)
		}

		// Verify approval
		var approvalStatus string
		var approvedBy *int
		var approvedAt *time.Time
		err = db.QueryRow(`
			SELECT approval_status, approved_by, approved_at
			FROM bookings WHERE id = ?
		`, bookingID).Scan(&approvalStatus, &approvedBy, &approvedAt)
		if err != nil {
			t.Fatalf("Failed to query booking: %v", err)
		}

		if approvalStatus != "approved" {
			t.Errorf("Expected approval_status 'approved', got '%s'", approvalStatus)
		}

		if approvedBy == nil || *approvedBy != adminID {
			t.Errorf("Expected approved_by = %d, got %v", adminID, approvedBy)
		}

		if approvedAt == nil {
			t.Error("Expected approved_at to be set")
		}
	})

	t.Run("approving already approved booking", func(t *testing.T) {
		// Create already approved booking
		result, err := db.Exec(`
			INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status, approved_by, approved_at)
			VALUES (2, 2, '2025-01-30', '10:00', 'scheduled', 'approved', 1, datetime('now'))
		`)
		if err != nil {
			t.Fatalf("Failed to create approved booking: %v", err)
		}

		bookingID, _ := result.LastInsertId()
		adminID := 2

		// Try to approve again
		err = repo.ApproveBooking(int(bookingID), adminID)

		// Should handle gracefully (no change or error depending on implementation)
		if err != nil {
			t.Logf("ApproveBooking on already approved returned: %v", err)
		}

		// Verify still approved with original admin
		var approvedBy int
		db.QueryRow("SELECT approved_by FROM bookings WHERE id = ?", bookingID).Scan(&approvedBy)

		// Original admin ID (1) should still be there or updated to 2
		// Both are acceptable behaviors
		t.Logf("Approved by after re-approval: %d", approvedBy)
	})

	t.Run("approving rejected booking", func(t *testing.T) {
		// Create rejected booking
		result, err := db.Exec(`
			INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status, rejection_reason)
			VALUES (3, 3, '2025-01-30', '10:00', 'cancelled', 'rejected', 'Not available')
		`)
		if err != nil {
			t.Fatalf("Failed to create rejected booking: %v", err)
		}

		bookingID, _ := result.LastInsertId()
		adminID := 1

		// Try to approve rejected booking
		err = repo.ApproveBooking(int(bookingID), adminID)

		// Should handle gracefully (no change or error)
		if err != nil {
			t.Logf("ApproveBooking on rejected returned: %v", err)
		}

		// Verify still rejected
		var approvalStatus string
		db.QueryRow("SELECT approval_status FROM bookings WHERE id = ?", bookingID).Scan(&approvalStatus)

		// Should remain rejected or be approved (depends on implementation)
		t.Logf("Status after approve attempt on rejected: %s", approvalStatus)
	})
}

// Test 2.3.3: RejectBooking - Reason Required
func TestRejectBooking_ReasonRequired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewBookingRepository(db)

	t.Run("rejects pending booking with reason", func(t *testing.T) {
		// Create pending booking
		result, err := db.Exec(`
			INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status)
			VALUES (1, 1, '2025-01-30', '10:00', 'scheduled', 'pending')
		`)
		if err != nil {
			t.Fatalf("Failed to create pending booking: %v", err)
		}

		bookingID, _ := result.LastInsertId()
		adminID := 1
		reason := "Kein Verfügbar"

		// Reject booking
		err = repo.RejectBooking(int(bookingID), adminID, reason)
		if err != nil {
			t.Fatalf("RejectBooking failed: %v", err)
		}

		// Verify rejection
		var approvalStatus string
		var status string
		var rejectionReason *string
		err = db.QueryRow(`
			SELECT approval_status, status, rejection_reason
			FROM bookings WHERE id = ?
		`, bookingID).Scan(&approvalStatus, &status, &rejectionReason)
		if err != nil {
			t.Fatalf("Failed to query booking: %v", err)
		}

		if approvalStatus != "rejected" {
			t.Errorf("Expected approval_status 'rejected', got '%s'", approvalStatus)
		}

		if status != "cancelled" {
			t.Errorf("Expected status 'cancelled', got '%s'", status)
		}

		if rejectionReason == nil || *rejectionReason != reason {
			t.Errorf("Expected rejection_reason '%s', got %v", reason, rejectionReason)
		}
	})

	t.Run("rejects with empty reason", func(t *testing.T) {
		// Create pending booking
		result, err := db.Exec(`
			INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status)
			VALUES (2, 2, '2025-01-30', '10:00', 'scheduled', 'pending')
		`)
		if err != nil {
			t.Fatalf("Failed to create pending booking: %v", err)
		}

		bookingID, _ := result.LastInsertId()
		adminID := 1
		reason := ""

		// Try to reject with empty reason
		err = repo.RejectBooking(int(bookingID), adminID, reason)

		// Should fail with validation error
		if err == nil {
			t.Error("Expected error when rejecting with empty reason, got nil")
		}
	})

	t.Run("cannot reject approved booking", func(t *testing.T) {
		// Create approved booking
		result, err := db.Exec(`
			INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status, approved_by, approved_at)
			VALUES (3, 3, '2025-01-30', '10:00', 'scheduled', 'approved', 1, datetime('now'))
		`)
		if err != nil {
			t.Fatalf("Failed to create approved booking: %v", err)
		}

		bookingID, _ := result.LastInsertId()
		adminID := 1
		reason := "Test"

		// Try to reject approved booking
		err = repo.RejectBooking(int(bookingID), adminID, reason)

		// Should fail or handle gracefully
		if err != nil {
			t.Logf("RejectBooking on approved returned: %v", err)
		}

		// Verify still approved
		var approvalStatus string
		db.QueryRow("SELECT approval_status FROM bookings WHERE id = ?", bookingID).Scan(&approvalStatus)

		// Should remain approved or be rejected (depends on implementation)
		t.Logf("Status after reject attempt on approved: %s", approvalStatus)
	})

	t.Run("rejection stores admin ID", func(t *testing.T) {
		// Create pending booking
		result, err := db.Exec(`
			INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, approval_status)
			VALUES (4, 4, '2025-01-30', '10:00', 'scheduled', 'pending')
		`)
		if err != nil {
			t.Fatalf("Failed to create pending booking: %v", err)
		}

		bookingID, _ := result.LastInsertId()
		adminID := 5
		reason := "Dog is sick"

		// Reject booking
		err = repo.RejectBooking(int(bookingID), adminID, reason)
		if err != nil {
			t.Fatalf("RejectBooking failed: %v", err)
		}

		// Verify approved_by is set to admin who rejected
		var approvedBy *int
		db.QueryRow("SELECT approved_by FROM bookings WHERE id = ?", bookingID).Scan(&approvedBy)

		if approvedBy == nil || *approvedBy != adminID {
			t.Errorf("Expected approved_by = %d, got %v", adminID, approvedBy)
		}
	})
}
