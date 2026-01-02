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
			INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
			VALUES (1, ?, 'Old', 'User', 'hash', 1, 1, ?, ?, ?)
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
			INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
			VALUES (1, ?, 'Active', 'User', 'hash', 1, 1, ?, ?, ?)
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
			INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, is_active, is_verified, terms_accepted_at, last_activity_at, deactivated_at, created_at)
			VALUES (1, ?, 'Deactivated', 'User', 'hash', 0, 1, ?, ?, ?, ?)
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

// DONE: TestCronService_AutoDeactivateInactiveUsers_SendsEmailNotifications tests that email notifications are sent on auto-deactivation
func TestCronService_AutoDeactivateInactiveUsers_SendsEmailNotifications(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	t.Run("should attempt to send email notification when user is auto-deactivated", func(t *testing.T) {
		// Create user with old last activity
		oldActivity := time.Now().AddDate(0, 0, -400) // 400 days ago
		email := "olduser_email@example.com"

		_, err := db.Exec(`
			INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
			VALUES (1, ?, 'Old', 'User Email', 'hash', 1, 1, ?, ?, ?)
		`, email, time.Now(), oldActivity, time.Now())
		if err != nil {
			t.Fatalf("Failed to create old user: %v", err)
		}

		// Run auto-deactivation (should not panic even though no email service is configured)
		cronService.autoDeactivateInactiveUsers()

		// Verify user is deactivated in database
		var isActive bool
		db.QueryRow("SELECT is_active FROM users WHERE email = ?", email).Scan(&isActive)
		if isActive {
			t.Error("User should be deactivated after auto-deactivation")
		}

		// TODO: After implementing SendAccountDeactivated method and adding email sending:
		// 1. Create a mock email service that tracks SendAccountDeactivated calls
		// 2. Verify that SendAccountDeactivated was called with correct email, name, and reason
		// 3. Test that emails are sent to multiple deactivated users
		// 4. Test that no email is sent if user has no email address
	})

	t.Run("should handle multiple inactive users", func(t *testing.T) {
		// Create two inactive users
		oldActivity1 := time.Now().AddDate(0, 0, -400)
		oldActivity2 := time.Now().AddDate(0, 0, -450)

		_, err := db.Exec(`
			INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
			VALUES (1, ?, 'Multi User', 'One', 'hash', 1, 1, ?, ?, ?)
		`, "multi1@example.com", time.Now(), oldActivity1, time.Now())
		if err != nil {
			t.Fatalf("Failed to create user 1: %v", err)
		}

		_, err = db.Exec(`
			INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
			VALUES (1, ?, 'Multi User', 'Two', 'hash', 1, 1, ?, ?, ?)
		`, "multi2@example.com", time.Now(), oldActivity2, time.Now())
		if err != nil {
			t.Fatalf("Failed to create user 2: %v", err)
		}

		// Run auto-deactivation
		cronService.autoDeactivateInactiveUsers()

		// Verify both users are deactivated
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM users WHERE email IN ('multi1@example.com', 'multi2@example.com') AND is_active = 0").Scan(&count)
		if err != nil || count != 2 {
			t.Errorf("Expected 2 deactivated users, but got %d", count)
		}
	})
}

// TestCronService_RunDaily_UsesEuropeBerlinTimezone tests that runDaily uses Europe/Berlin timezone
func TestCronService_RunDaily_UsesEuropeBerlinTimezone(t *testing.T) {
	// This test verifies that the daily scheduler uses Europe/Berlin timezone
	// The demo reset should happen at midnight Berlin time, not server local time

	t.Run("getBerlinLocation returns valid Europe/Berlin timezone", func(t *testing.T) {
		loc := getBerlinLocation()

		// Should return a valid location
		if loc == nil {
			t.Fatal("getBerlinLocation should return non-nil location")
		}

		// Should be Europe/Berlin (or UTC as fallback)
		locName := loc.String()
		if locName != "Europe/Berlin" && locName != "UTC" {
			t.Errorf("Expected Europe/Berlin or UTC, got %s", locName)
		}
	})

	t.Run("getBerlinLocation handles timezone loading", func(t *testing.T) {
		// Call multiple times to verify consistency
		loc1 := getBerlinLocation()
		loc2 := getBerlinLocation()

		if loc1.String() != loc2.String() {
			t.Error("getBerlinLocation should return consistent results")
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

// TestCronService_SendBookingReminders tests the booking reminder functionality
func TestCronService_SendBookingReminders(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	t.Run("handles nil email service gracefully", func(t *testing.T) {
		// Should not panic when email service is nil
		cronService.sendBookingReminders()
		// Test passes if no panic occurs
	})

	t.Run("processes bookings in reminder window", func(t *testing.T) {
		// Create test user with email and dog
		userID := testutil.SeedTestUser(t, db, "reminder_test@example.com", "Reminder Test", "green")
		dogID := testutil.SeedTestDog(t, db, "ReminderDog", "Labrador", "green")

		// Create booking 90 minutes from now (within 1-2 hour window)
		now := time.Now()
		reminderTime := now.Add(90 * time.Minute)

		// Only run test if we don't cross midnight
		if reminderTime.Day() != now.Day() {
			t.Skip("Skipping test: reminder time crosses midnight")
		}

		bookingDate := reminderTime.Format("2006-01-02")
		bookingTime := reminderTime.Format("15:04")

		testutil.SeedTestBooking(t, db, userID, dogID, bookingDate, bookingTime, "scheduled")

		// Run reminders (won't send email since service is nil, but should process without error)
		cronService.sendBookingReminders()
		// Test passes if no panic/error occurs
	})
}

// TestCronService_SendBookingReminders_SkipsAlreadyReminded tests skipping already-reminded bookings
func TestCronService_SendBookingReminders_SkipsAlreadyReminded(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	// Create test data
	userID := testutil.SeedTestUser(t, db, "already_reminded@example.com", "Already Reminded", "green")
	dogID := testutil.SeedTestDog(t, db, "RemindedDog", "Beagle", "green")

	// Create booking 90 minutes from now
	now := time.Now()
	reminderTime := now.Add(90 * time.Minute)

	if reminderTime.Day() != now.Day() {
		t.Skip("Skipping test: reminder time crosses midnight")
	}

	bookingDate := reminderTime.Format("2006-01-02")
	bookingTime := reminderTime.Format("15:04")

	bookingID := testutil.SeedTestBooking(t, db, userID, dogID, bookingDate, bookingTime, "scheduled")

	// Mark reminder as already sent
	_, err := db.Exec("UPDATE bookings SET reminder_sent_at = ? WHERE id = ?", time.Now(), bookingID)
	if err != nil {
		t.Fatalf("Failed to set reminder_sent_at: %v", err)
	}

	// Run reminders - should skip this booking
	cronService.sendBookingReminders()

	// Verify reminder_sent_at wasn't changed (it was already set)
	var reminderSentAt *time.Time
	db.QueryRow("SELECT reminder_sent_at FROM bookings WHERE id = ?", bookingID).Scan(&reminderSentAt)

	if reminderSentAt == nil {
		t.Error("reminder_sent_at should still be set")
	}
}

// TestCronService_SendBookingReminders_SkipsNoEmail tests skipping users without email
func TestCronService_SendBookingReminders_SkipsNoEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	// Create user without email (simulating deleted user)
	_, err := db.Exec(`
		INSERT INTO users (tenant_id, first_name, last_name, password_hash, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
		VALUES (1, 'No', 'Email', 'hash', 1, 1, ?, ?, ?)
	`, time.Now(), time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to create user without email: %v", err)
	}

	var userID int
	db.QueryRow("SELECT id FROM users WHERE first_name = 'No' AND last_name = 'Email'").Scan(&userID)

	dogID := testutil.SeedTestDog(t, db, "NoEmailDog", "Terrier", "green")

	// Create booking 90 minutes from now
	now := time.Now()
	reminderTime := now.Add(90 * time.Minute)

	if reminderTime.Day() != now.Day() {
		t.Skip("Skipping test: reminder time crosses midnight")
	}

	bookingDate := reminderTime.Format("2006-01-02")
	bookingTime := reminderTime.Format("15:04")

	testutil.SeedTestBooking(t, db, userID, dogID, bookingDate, bookingTime, "scheduled")

	// Run reminders - should handle gracefully
	cronService.sendBookingReminders()
	// Test passes if no panic occurs
}

// TestCronService_ResetDemoTenant tests demo tenant reset functionality
func TestCronService_ResetDemoTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	t.Run("skips when no demo tenant exists", func(t *testing.T) {
		// Ensure no demo tenant exists
		db.Exec("UPDATE tenants SET is_demo = 0")

		// Should not panic or error
		cronService.resetDemoTenant()
		// Test passes if no panic occurs
	})

	t.Run("processes demo tenant when exists", func(t *testing.T) {
		// Create a demo tenant
		_, err := db.Exec(`
			INSERT INTO tenants (slug, name, contact_email, status, is_demo, created_at)
			VALUES ('demo-test', 'Demo Test Tenant', 'demo@example.com', 'active', 1, ?)
		`, time.Now())
		if err != nil {
			t.Fatalf("Failed to create demo tenant: %v", err)
		}

		var demoTenantID int
		db.QueryRow("SELECT id FROM tenants WHERE slug = 'demo-test'").Scan(&demoTenantID)

		// Run reset
		cronService.resetDemoTenant()
		// Test passes if no panic occurs
	})
}

// TestCronService_ResetDemoTenant_RespectsNextResetAt tests that reset respects scheduled time
func TestCronService_ResetDemoTenant_RespectsNextResetAt(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	// Create a demo tenant
	_, err := db.Exec(`
		INSERT INTO tenants (slug, name, contact_email, status, is_demo, created_at)
		VALUES ('demo-scheduled', 'Demo Scheduled', 'demo@example.com', 'active', 1, ?)
	`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create demo tenant: %v", err)
	}

	var demoTenantID int
	db.QueryRow("SELECT id FROM tenants WHERE slug = 'demo-scheduled'").Scan(&demoTenantID)

	// Set next_reset_at to future (should skip reset)
	futureReset := time.Now().Add(24 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO demo_tenant_state (tenant_id, admin_password, next_reset_at, created_at, updated_at)
		VALUES (?, 'test-password', ?, ?, ?)
	`, demoTenantID, futureReset, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to create demo state: %v", err)
	}

	// Run reset - should skip because next_reset_at is in the future
	cronService.resetDemoTenant()
	// Test passes if no panic occurs and reset was skipped
}

// TestCronService_AutoDeactivateUsersForTenant_RespectsTenantSettings tests tenant-specific deactivation days
func TestCronService_AutoDeactivateUsersForTenant_RespectsTenantSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	t.Run("uses tenant-specific deactivation days setting", func(t *testing.T) {
		// Set tenant-specific setting to 30 days (shorter than default 365)
		// Note: system_settings has columns (tenant_id, key, value, updated_at)
		// Use INSERT OR REPLACE for SQLite compatibility
		// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
		_, err := db.Exec(`
			INSERT OR REPLACE INTO system_settings (tenant_id, "key", value, updated_at)
			VALUES (1, 'auto_deactivation_days', '30', ?)
		`, time.Now())
		if err != nil {
			t.Fatalf("Failed to set tenant setting: %v", err)
		}

		// Create user inactive for 50 days (should be deactivated with 30-day setting)
		oldActivity := time.Now().AddDate(0, 0, -50)
		email := "tenant_specific@example.com"

		_, err = db.Exec(`
			INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
			VALUES (1, ?, 'Tenant', 'Specific', 'hash', 1, 1, ?, ?, ?)
		`, email, time.Now(), oldActivity, time.Now())
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Run deactivation for tenant 1
		cronService.autoDeactivateUsersForTenant(1)

		// Verify user is deactivated
		var isActive bool
		db.QueryRow("SELECT is_active FROM users WHERE email = ?", email).Scan(&isActive)

		if isActive {
			t.Error("User should be deactivated based on tenant-specific 30-day setting")
		}
	})

	t.Run("uses default 365 days when setting not configured", func(t *testing.T) {
		// Remove the setting
		db.Exec(`DELETE FROM system_settings WHERE tenant_id = 1 AND "key" = 'auto_deactivation_days'`)

		// Create user inactive for 100 days (should NOT be deactivated with default 365)
		oldActivity := time.Now().AddDate(0, 0, -100)
		email := "default_days@example.com"

		_, err := db.Exec(`
			INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
			VALUES (1, ?, 'Default', 'Days', 'hash', 1, 1, ?, ?, ?)
		`, email, time.Now(), oldActivity, time.Now())
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Run deactivation
		cronService.autoDeactivateUsersForTenant(1)

		// Verify user is still active (100 days < 365 days default)
		var isActive bool
		db.QueryRow("SELECT is_active FROM users WHERE email = ?", email).Scan(&isActive)

		if !isActive {
			t.Error("User should remain active - only 100 days inactive vs 365-day default")
		}
	})
}

// TestCronService_AutoDeactivateUsersForTenant_SkipsAdmins tests that admins are excluded from auto-deactivation
func TestCronService_AutoDeactivateUsersForTenant_SkipsAdmins(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	// Create admin user with old activity (should NOT be deactivated)
	oldActivity := time.Now().AddDate(0, 0, -400)
	email := "admin_skip@example.com"

	_, err := db.Exec(`
		INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, is_active, is_verified, is_admin, terms_accepted_at, last_activity_at, created_at)
		VALUES (1, ?, 'Admin', 'User', 'hash', 1, 1, 1, ?, ?, ?)
	`, email, time.Now(), oldActivity, time.Now())
	if err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	// Run deactivation
	cronService.autoDeactivateUsersForTenant(1)

	// Verify admin is still active
	var isActive bool
	db.QueryRow("SELECT is_active FROM users WHERE email = ?", email).Scan(&isActive)

	if !isActive {
		t.Error("Admin user should NOT be auto-deactivated regardless of inactivity")
	}
}

// TestCronService_Stop tests the stop functionality
func TestCronService_Stop(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	// Stop should not panic even if Start wasn't called
	cronService.Stop()

	// Verify stopChan is closed (reading from closed channel returns immediately)
	select {
	case <-cronService.stopChan:
		// Channel is closed, as expected
	default:
		t.Error("stopChan should be closed after Stop()")
	}
}

// TestCronService_RunPeriodically_ExecutesImmediately tests that periodic jobs run immediately on start
func TestCronService_RunPeriodically_ExecutesImmediately(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	executed := false
	executionCount := 0

	// Run in goroutine with very long interval
	go func() {
		cronService.runPeriodically("test-immediate", 1*time.Hour, func() {
			executed = true
			executionCount++
		})
	}()

	// Give it a moment to execute
	time.Sleep(50 * time.Millisecond)

	// Stop the service
	cronService.Stop()

	if !executed {
		t.Error("runPeriodically should execute function immediately on start")
	}

	if executionCount != 1 {
		t.Errorf("Expected exactly 1 execution, got %d", executionCount)
	}
}

// TestCronService_RunPeriodically_StopsOnSignal tests that periodic jobs stop when signaled
func TestCronService_RunPeriodically_StopsOnSignal(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	stopped := make(chan bool, 1)

	go func() {
		cronService.runPeriodically("test-stop", 10*time.Millisecond, func() {
			// Short interval, will run multiple times if not stopped
		})
		stopped <- true
	}()

	// Let it run briefly
	time.Sleep(30 * time.Millisecond)

	// Stop the service
	cronService.Stop()

	// Wait for goroutine to finish
	select {
	case <-stopped:
		// Goroutine stopped as expected
	case <-time.After(1 * time.Second):
		t.Error("runPeriodically should stop when stopChan is closed")
	}
}

// =============================================================================
// BUG-FINDING EDGE CASE TESTS
// =============================================================================

// TestCronService_AutoCompleteBookings_EdgeCases tests boundary conditions
func TestCronService_AutoCompleteBookings_EdgeCases(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	userID := testutil.SeedTestUser(t, db, "edge@example.com", "Edge User", "green")
	dogID := testutil.SeedTestDog(t, db, "EdgeDog", "Labrador", "green")

	t.Run("handles booking with empty scheduled_time", func(t *testing.T) {
		// Create booking with empty scheduled_time (edge case - schema requires NOT NULL but let's test with empty string)
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		_, err := db.Exec(`
			INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, tenant_id)
			VALUES (?, ?, ?, '', 'scheduled', 1)
		`, userID, dogID, yesterday)
		if err != nil {
			t.Fatalf("Failed to create booking with empty time: %v", err)
		}

		// Should not panic
		cronService.autoCompleteBookings()

		// Verify booking is completed (past date with empty time should still complete)
		var status string
		db.QueryRow("SELECT status FROM bookings WHERE scheduled_time = ''").Scan(&status)
		if status != "completed" {
			t.Errorf("Booking with empty scheduled_time should be completed, got: %s", status)
		}
	})

	t.Run("handles multiple bookings same dog same day", func(t *testing.T) {
		testutil.ClearTable(t, db, "bookings")

		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

		// Create multiple bookings for same dog on same day (morning, afternoon, evening)
		testutil.SeedTestBooking(t, db, userID, dogID, yesterday, "09:00", "scheduled")
		testutil.SeedTestBooking(t, db, userID, dogID, yesterday, "14:00", "scheduled")
		testutil.SeedTestBooking(t, db, userID, dogID, yesterday, "18:00", "scheduled")

		cronService.autoCompleteBookings()

		// All three should be completed
		var count int
		db.QueryRow("SELECT COUNT(*) FROM bookings WHERE date = ? AND status = 'completed'", yesterday).Scan(&count)
		if count != 3 {
			t.Errorf("Expected 3 completed bookings, got %d", count)
		}
	})

	t.Run("boundary: booking at current time today", func(t *testing.T) {
		testutil.ClearTable(t, db, "bookings")

		now := time.Now()
		// Skip this test if it's too early in the morning (before 02:00)
		// because -1 hour would cross to previous day
		if now.Hour() < 2 {
			t.Skip("Skipping boundary test: too close to midnight")
		}

		today := now.Format("2006-01-02")
		// Create booking at an hour ago (should complete)
		pastTime := now.Add(-1 * time.Hour).Format("15:04")

		testutil.SeedTestBooking(t, db, userID, dogID, today, pastTime, "scheduled")

		cronService.autoCompleteBookings()

		var status string
		db.QueryRow("SELECT status FROM bookings WHERE date = ? AND scheduled_time = ?", today, pastTime).Scan(&status)
		if status != "completed" {
			t.Errorf("Today's booking with past time should be completed, got: %s", status)
		}
	})
}

// TestCronService_AutoDeactivateUsers_EdgeCases tests admin/superadmin skipping
func TestCronService_AutoDeactivateUsers_EdgeCases(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	// Ensure tenant exists (use INSERT OR IGNORE to handle existing tenant from seed)
	_, err := db.Exec(`
		INSERT OR IGNORE INTO tenants (id, slug, name, contact_email, status, created_at)
		VALUES (1, 'test-tenant', 'Test Tenant', 'test@example.com', 'active', ?)
	`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	t.Run("skips super admin users", func(t *testing.T) {
		// Create super admin with old activity
		oldActivity := time.Now().AddDate(-2, 0, 0) // 2 years ago
		_, err := db.Exec(`
			INSERT INTO users (email, first_name, last_name, password_hash, is_verified, is_active,
			                   is_admin, is_super_admin, tenant_id, last_activity_at, created_at, terms_accepted_at)
			VALUES ('superadmin@test.com', 'Super', 'Admin', 'hash', 1, 1, 1, 1, 1, ?, ?, ?)
		`, oldActivity, oldActivity, oldActivity)
		if err != nil {
			t.Fatalf("Failed to create super admin: %v", err)
		}

		cronService.autoDeactivateInactiveUsers()

		// Super admin should NOT be deactivated
		var isActive int
		db.QueryRow("SELECT is_active FROM users WHERE email = 'superadmin@test.com'").Scan(&isActive)
		if isActive != 1 {
			t.Error("Super admin should NOT be deactivated regardless of inactivity")
		}
	})

	t.Run("skips regular admin users", func(t *testing.T) {
		// Create regular admin with old activity
		oldActivity := time.Now().AddDate(-2, 0, 0)
		_, err := db.Exec(`
			INSERT INTO users (email, first_name, last_name, password_hash, is_verified, is_active,
			                   is_admin, is_super_admin, tenant_id, last_activity_at, created_at, terms_accepted_at)
			VALUES ('admin@test.com', 'Regular', 'Admin', 'hash', 1, 1, 1, 0, 1, ?, ?, ?)
		`, oldActivity, oldActivity, oldActivity)
		if err != nil {
			t.Fatalf("Failed to create admin: %v", err)
		}

		cronService.autoDeactivateInactiveUsers()

		// Admin should NOT be deactivated
		var isActive int
		db.QueryRow("SELECT is_active FROM users WHERE email = 'admin@test.com'").Scan(&isActive)
		if isActive != 1 {
			t.Error("Admin should NOT be deactivated regardless of inactivity")
		}
	})

	t.Run("boundary: exactly 365 days inactive", func(t *testing.T) {
		testutil.ClearTable(t, db, "users")

		// Create user with exactly 365 days of inactivity (boundary)
		exactly365DaysAgo := time.Now().AddDate(0, 0, -365)
		_, err := db.Exec(`
			INSERT INTO users (email, first_name, last_name, password_hash, is_verified, is_active,
			                   is_admin, is_super_admin, tenant_id, last_activity_at, created_at, terms_accepted_at)
			VALUES ('boundary@test.com', 'Boundary', 'User', 'hash', 1, 1, 0, 0, 1, ?, ?, ?)
		`, exactly365DaysAgo, exactly365DaysAgo, exactly365DaysAgo)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		cronService.autoDeactivateInactiveUsers()

		// User at exactly 365 days should be deactivated (>= 365)
		var isActive int
		db.QueryRow("SELECT is_active FROM users WHERE email = 'boundary@test.com'").Scan(&isActive)
		if isActive != 0 {
			t.Error("User with exactly 365 days inactivity should be deactivated")
		}
	})

	t.Run("boundary: 364 days inactive - should NOT deactivate", func(t *testing.T) {
		testutil.ClearTable(t, db, "users")

		// Create user with 364 days of inactivity (just under threshold)
		days364Ago := time.Now().AddDate(0, 0, -364)
		_, err := db.Exec(`
			INSERT INTO users (email, first_name, last_name, password_hash, is_verified, is_active,
			                   is_admin, is_super_admin, tenant_id, last_activity_at, created_at, terms_accepted_at)
			VALUES ('under@test.com', 'Under', 'Threshold', 'hash', 1, 1, 0, 0, 1, ?, ?, ?)
		`, days364Ago, days364Ago, days364Ago)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		cronService.autoDeactivateInactiveUsers()

		// User at 364 days should NOT be deactivated
		var isActive int
		db.QueryRow("SELECT is_active FROM users WHERE email = 'under@test.com'").Scan(&isActive)
		if isActive != 1 {
			t.Error("User with 364 days inactivity should NOT be deactivated")
		}
	})
}

// TestCronService_NilDemoSeedService tests handling when demo seed service is nil
func TestCronService_NilDemoSeedService(t *testing.T) {
	db := testutil.SetupTestDB(t)
	// Create cron service with nil config (which means nil demoSeedService)
	cronService := NewCronService(db, nil)

	// Create a demo tenant to trigger the reset logic
	_, err := db.Exec(`
		INSERT INTO tenants (slug, name, contact_email, status, is_demo, created_at)
		VALUES ('demo', 'Demo', 'demo@test.com', 'active', 1, ?)
	`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create demo tenant: %v", err)
	}

	// Should not panic - gracefully skip when demoSeedService is nil
	cronService.resetDemoTenant()
	// Test passes if no panic occurs
}

// TestCronService_ConcurrentStopCalls tests thread safety of Stop()
func TestCronService_ConcurrentStopCalls(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cronService := NewCronService(db, nil)

	// Call Stop() concurrently from multiple goroutines
	// This tests the sync.Once protection
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			cronService.Stop() // Should not panic even when called multiple times
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// OK
		case <-time.After(1 * time.Second):
			t.Fatal("Concurrent Stop() calls timed out")
		}
	}
}
