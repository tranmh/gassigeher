package repository

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tranm/gassigeher/internal/models"
	"github.com/tranm/gassigeher/internal/testutil"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables
	schema := `
	CREATE TABLE bookings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		dog_id INTEGER NOT NULL,
		date TEXT NOT NULL,
		walk_type TEXT CHECK(walk_type IN ('morning', 'evening')),
		scheduled_time TEXT NOT NULL,
		status TEXT DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
		completed_at TIMESTAMP,
		user_notes TEXT,
		admin_cancellation_reason TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(dog_id, date, walk_type)
	);
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
		WalkType:      "morning",
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
		WalkType:      "morning",
		ScheduledTime: "09:00",
	}
	repo.Create(booking)

	// Check for double booking
	isBooked, err := repo.CheckDoubleBooking(1, "2025-12-01", "morning")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !isBooked {
		t.Error("Expected dog to be marked as booked")
	}

	// Check different walk type
	isBooked, err = repo.CheckDoubleBooking(1, "2025-12-01", "evening")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if isBooked {
		t.Error("Expected evening slot to be available")
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
		WalkType:      "morning",
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

	booking := &models.Booking{
		UserID:        1,
		DogID:         1,
		Date:          "2025-12-01",
		WalkType:      "morning",
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
			WalkType:      "morning",
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
		WalkType:      "morning",
		ScheduledTime: "09:00",
		Status:        "scheduled",
	}
	repo.Create(booking1)

	booking2 := &models.Booking{
		UserID:        2,
		DogID:         2,
		Date:          "2025-12-02",
		WalkType:      "evening",
		ScheduledTime: "15:00",
		Status:        "scheduled",
	}
	repo.Create(booking2)

	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	booking3 := &models.Booking{
		UserID:        1,
		DogID:         1,
		Date:          yesterday,
		WalkType:      "morning",
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
		WalkType:      "morning",
		ScheduledTime: "09:00",
		Status:        "scheduled",
	}
	repo.Create(booking)

	// Update to completed status
	db.Exec("UPDATE bookings SET status = 'completed', completed_at = ? WHERE id = ?", time.Now(), booking.ID)

	t.Run("add notes to booking", func(t *testing.T) {
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
		WalkType:      "morning",
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
		WalkType:      "morning",
		ScheduledTime: "09:00",
		Status:        "scheduled",
	}
	repo.Create(futureBooking1)

	futureBooking2 := &models.Booking{
		UserID:        userID,
		DogID:         2,
		Date:          nextWeek,
		WalkType:      "evening",
		ScheduledTime: "15:00",
		Status:        "scheduled",
	}
	repo.Create(futureBooking2)

	// Create booking for different user (should not be included)
	otherUserBooking := &models.Booking{
		UserID:        2,
		DogID:         1,
		Date:          tomorrow,
		WalkType:      "evening",
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
		WalkType:      "morning",
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
			WalkType:      "morning",
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
			WalkType:      "evening",
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
			WalkType:      "morning",
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
			WalkType:      "evening",
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
		bookingID := testutil.SeedTestBooking(t, db, userID, dogID, bookingDate, "morning", "09:00", "scheduled")

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
		bookingID := testutil.SeedTestBooking(t, db, userID, dogID, bookingDate, "evening", "16:00", "scheduled")

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
