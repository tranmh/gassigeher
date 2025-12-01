package cron

import (
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/testutil"
)

// DONE: TestCronService_AutoCompleteBookings tests automatic booking completion
func TestCronService_AutoCompleteBookings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	// Create test user and dog
	userID := testutil.SeedTestUser(t, db, "test@example.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

	t.Run("complete past bookings", func(t *testing.T) {
		// Create booking from yesterday
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		testutil.SeedTestBooking(t, db, userID, dogID, yesterday, "09:00", "scheduled")

		// Create booking from last week
		lastWeek := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
		testutil.SeedTestBooking(t, db, userID, dogID, lastWeek, "15:00", "scheduled")

		// Create future booking (should not be completed)
		tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		futureBookingID := testutil.SeedTestBooking(t, db, userID, dogID, tomorrow, "09:00", "scheduled")

		// Run auto-complete
		cronService.autoCompleteBookings()

		// Verify past bookings are completed
		var yesterdayStatus, lastWeekStatus, futureStatus string
		db.QueryRow("SELECT status FROM bookings WHERE date = ? AND scheduled_time = '09:00'", yesterday).Scan(&yesterdayStatus)
		db.QueryRow("SELECT status FROM bookings WHERE date = ? AND scheduled_time = '15:00'", lastWeek).Scan(&lastWeekStatus)
		db.QueryRow("SELECT status FROM bookings WHERE id = ?", futureBookingID).Scan(&futureStatus)

		if yesterdayStatus != "completed" {
			t.Errorf("Yesterday's booking should be completed, got status: %s", yesterdayStatus)
		}

		if lastWeekStatus != "completed" {
			t.Errorf("Last week's booking should be completed, got status: %s", lastWeekStatus)
		}

		if futureStatus != "scheduled" {
			t.Errorf("Future booking should remain scheduled, got status: %s", futureStatus)
		}
	})

	t.Run("skip already completed bookings", func(t *testing.T) {
		// Create already completed booking from past
		past := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
		bookingID := testutil.SeedTestBooking(t, db, userID, dogID, past, "10:00", "completed")

		// Set completed_at timestamp
		db.Exec("UPDATE bookings SET completed_at = ? WHERE id = ?", time.Now().AddDate(0, 0, -5), bookingID)

		// Run auto-complete
		cronService.autoCompleteBookings()

		// Verify completed_at wasn't overwritten
		var completedAt string
		db.QueryRow("SELECT completed_at FROM bookings WHERE id = ?", bookingID).Scan(&completedAt)

		if completedAt == "" {
			t.Error("completed_at should not be cleared")
		}
	})

	t.Run("skip cancelled bookings", func(t *testing.T) {
		// Create cancelled booking from past
		past := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
		testutil.SeedTestBooking(t, db, userID, dogID, past, "16:00", "cancelled")

		// Run auto-complete
		cronService.autoCompleteBookings()

		// Verify status remains cancelled
		var status string
		db.QueryRow("SELECT status FROM bookings WHERE date = ? AND scheduled_time = '16:00'", past).Scan(&status)

		if status != "cancelled" {
			t.Errorf("Cancelled booking should remain cancelled, got: %s", status)
		}
	})
}

// DONE: TestCronService_AutoDeactivateInactiveUsers tests automatic user deactivation
func TestCronService_AutoDeactivateInactiveUsers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	t.Run("deactivate users inactive for 365+ days", func(t *testing.T) {
		// Create user with old last activity
		oldActivity := time.Now().AddDate(0, 0, -400) // 400 days ago
		email := "old@example.com"

		_, err := db.Exec(`
			INSERT INTO users (email, name, password_hash, experience_level, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
			VALUES (?, 'Old User', 'hash', 'green', 1, 1, ?, ?, ?)
		`, email, time.Now(), oldActivity, time.Now())
		if err != nil {
			t.Fatalf("Failed to create old user: %v", err)
		}

		// Create recent user (should not be deactivated)
		testutil.SeedTestUser(t, db, "recent@example.com", "Recent User", "green")

		// Run auto-deactivation
		cronService.autoDeactivateInactiveUsers()

		// Verify old user is deactivated
		var isActive bool
		var deactivationReason *string
		err = db.QueryRow("SELECT is_active, deactivation_reason FROM users WHERE email = ?", email).Scan(&isActive, &deactivationReason)
		if err != nil {
			t.Fatalf("Failed to query old user: %v", err)
		}

		if isActive {
			t.Error("Old user should be deactivated")
		}

		if deactivationReason == nil || *deactivationReason == "" {
			t.Error("Deactivation reason should be set")
		}
	})

	t.Run("skip users with recent activity", func(t *testing.T) {
		// Create user with recent activity
		recentEmail := "active@example.com"
		recentActivity := time.Now().AddDate(0, 0, -30) // 30 days ago

		_, err := db.Exec(`
			INSERT INTO users (email, name, password_hash, experience_level, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
			VALUES (?, 'Active User', 'hash', 'green', 1, 1, ?, ?, ?)
		`, recentEmail, time.Now(), recentActivity, time.Now())
		if err != nil {
			t.Fatalf("Failed to create recent user: %v", err)
		}

		// Run auto-deactivation
		cronService.autoDeactivateInactiveUsers()

		// Verify recent user is still active
		var isActive bool
		db.QueryRow("SELECT is_active FROM users WHERE email = ?", recentEmail).Scan(&isActive)

		if !isActive {
			t.Error("Recent user should remain active")
		}
	})

	t.Run("skip already deactivated users", func(t *testing.T) {
		// Create already deactivated user
		email := "already_deactivated@example.com"
		oldActivity := time.Now().AddDate(0, 0, -500)

		_, err := db.Exec(`
			INSERT INTO users (email, name, password_hash, experience_level, is_active, is_verified, terms_accepted_at, last_activity_at, deactivated_at, created_at)
			VALUES (?, 'Deactivated User', 'hash', 'green', 0, 1, ?, ?, ?, ?)
		`, email, time.Now(), oldActivity, time.Now().AddDate(0, 0, -100), time.Now())
		if err != nil {
			t.Fatalf("Failed to create deactivated user: %v", err)
		}

		// Run auto-deactivation
		cronService.autoDeactivateInactiveUsers()

		// Verify user remains deactivated (no duplicate processing)
		var isActive bool
		db.QueryRow("SELECT is_active FROM users WHERE email = ?", email).Scan(&isActive)

		if isActive {
			t.Error("Already deactivated user should remain deactivated")
		}
	})
}

// DONE: TestCronService_NewCronService tests cron service initialization
func TestCronService_NewCronService(t *testing.T) {
	db := testutil.SetupTestDB(t)

	service := NewCronService(db, nil)

	if service == nil {
		t.Fatal("NewCronService should return non-nil service")
	}

	if service.db == nil {
		t.Error("Database should be set")
	}

	if service.bookingRepo == nil {
		t.Error("BookingRepository should be initialized")
	}

	if service.userRepo == nil {
		t.Error("UserRepository should be initialized")
	}

	if service.settingsRepo == nil {
		t.Error("SettingsRepository should be initialized")
	}

	if service.stopChan == nil {
		t.Error("Stop channel should be initialized")
	}
}
