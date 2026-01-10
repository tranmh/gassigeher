package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// Test 1.1.1: ValidateBookingTime - Weekday Allowed Times
func TestValidateBookingTime_WeekdayAllowed(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	testCases := []struct {
		name    string
		date    string
		time    string
		wantErr bool
	}{
		{"Morning window", "2025-01-27", "09:30", false},
		{"Morning window end", "2025-01-27", "11:45", false},
		{"Afternoon window", "2025-01-27", "14:45", false},
		{"Evening window", "2025-01-27", "18:30", false},
		{"Tuesday morning", "2025-01-28", "10:00", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ValidateBookingTime(context.Background(), 0, tc.date, tc.time) // tenantID = 0
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateBookingTime() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// Test 1.1.2: ValidateBookingTime - Weekday Blocked Times
func TestValidateBookingTime_WeekdayBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	testCases := []struct {
		name            string
		date            string
		time            string
		wantErrContains string
	}{
		{"Lunch block start", "2025-01-27", "13:00", "Mittagspause"},
		{"Lunch block middle", "2025-01-27", "13:45", "Mittagspause"},
		{"Feeding block start", "2025-01-27", "17:00", "Fütterungszeit"},
		{"Feeding block middle", "2025-01-27", "17:30", "Fütterungszeit"},
		{"Before opening", "2025-01-27", "08:00", "außerhalb"},
		{"After closing", "2025-01-27", "20:00", "außerhalb"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ValidateBookingTime(context.Background(), 0, tc.date, tc.time) // tenantID = 0
			if err == nil {
				t.Error("Expected error, got nil")
				return
			}
			if !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Errorf("Error %v should contain %q", err, tc.wantErrContains)
			}
		})
	}
}

// Test 1.1.3: ValidateBookingTime - Weekend Times
func TestValidateBookingTime_WeekendTimes(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	testCases := []struct {
		name    string
		date    string
		time    string
		wantErr bool
	}{
		{"Saturday morning", "2025-01-25", "10:00", false},
		{"Saturday afternoon", "2025-01-25", "15:00", false},
		{"Sunday morning", "2025-01-26", "11:30", false},
		{"Sunday afternoon", "2025-01-26", "16:30", false},
		{"Saturday feeding block", "2025-01-25", "12:30", true},
		{"Saturday lunch block", "2025-01-25", "13:30", true},
		{"Saturday outside window", "2025-01-25", "17:30", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ValidateBookingTime(context.Background(), 0, tc.date, tc.time) // tenantID = 0
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateBookingTime() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// Test 1.1.4: ValidateBookingTime - Holiday Times
func TestValidateBookingTime_HolidayTimes(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	// Seed holiday: 2025-01-01 (Neujahrstag)
	holiday := &models.CustomHoliday{
		Date:     "2025-01-01",
		Name:     "Neujahrstag",
		IsActive: true,
		Source:   "test",
	}
	err := holidayRepo.CreateHoliday(0, holiday) // tenantID = 0
	if err != nil {
		t.Fatalf("Failed to seed holiday: %v", err)
	}

	testCases := []struct {
		name    string
		date    string
		time    string
		wantErr bool
	}{
		{"Holiday morning (weekend rules)", "2025-01-01", "10:00", false},
		{"Holiday afternoon (weekend rules)", "2025-01-01", "15:00", false},
		{"Holiday feeding block (weekend)", "2025-01-01", "12:30", true},
		{"Holiday lunch block (weekend)", "2025-01-01", "13:30", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ValidateBookingTime(context.Background(), 0, tc.date, tc.time) // tenantID = 0
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateBookingTime() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// Test 1.1.5: GetAvailableTimeSlots - Granularity
func TestGetAvailableTimeSlots_Granularity(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	// Test weekday
	slots, err := service.GetAvailableTimeSlots(context.Background(), 0, "2025-01-27") // tenantID = 0, Monday
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify 15-minute intervals present in morning window
	// Note: 11:45 excluded due to 30-minute buffer before period end (12:00)
	expectedSlots := []string{
		"09:00", "09:15", "09:30", "09:45",
		"10:00", "10:15", "10:30", "10:45",
		"11:00", "11:15", "11:30",
	}

	for _, expected := range expectedSlots {
		if !containsTimeSlot(slots, expected) {
			t.Errorf("Expected slot %s not found in results", expected)
		}
	}

	// Verify blocked times NOT present
	blockedSlots := []string{"13:00", "13:15", "13:30", "13:45", "17:00", "17:15"}
	for _, blocked := range blockedSlots {
		if containsTimeSlot(slots, blocked) {
			t.Errorf("Blocked slot %s should not be in results", blocked)
		}
	}

	// Verify slots are in correct format (HH:MM)
	for _, slot := range slots {
		if len(slot) != 5 || slot[2] != ':' {
			t.Errorf("Slot %s has invalid format, expected HH:MM", slot)
		}
	}
}

// Test 1.1.6: RequiresApproval - Morning Walk Detection
func TestRequiresApproval(t *testing.T) {
	db := testutil.SetupTestDB(t)

	settingsRepo := repository.NewSettingsRepository(db)

	// Enable morning approval setting (should already exist from migration)
	err := settingsRepo.Update(0, "morning_walk_requires_approval", "true") // tenantID = 0
	if err != nil {
		t.Fatalf("Failed to update test setting: %v", err)
	}

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	testCases := []struct {
		time string
		want bool
	}{
		{"09:00", true},
		{"10:30", true},
		{"11:45", true},
		{"12:00", false}, // Boundary
		{"14:00", false},
		{"18:00", false},
	}

	for _, tc := range testCases {
		t.Run(tc.time, func(t *testing.T) {
			requires, err := service.RequiresApproval(0, tc.time) // tenantID = 0
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if requires != tc.want {
				t.Errorf("RequiresApproval(%s) = %v, want %v", tc.time, requires, tc.want)
			}
		})
	}
}

// Test 1.1.7: GetDayType - Day Type Classification
func TestGetDayType(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	// Seed holidays
	holidays := []models.CustomHoliday{
		{Date: "2025-01-01", Name: "Neujahrstag", IsActive: true, Source: "test"},
		{Date: "2025-01-06", Name: "Heilige Drei Könige", IsActive: true, Source: "test"},
	}
	for _, h := range holidays {
		holiday := h
		_ = holidayRepo.CreateHoliday(0, &holiday) // tenantID = 0
	}

	testCases := []struct {
		name string
		date string
		want string
	}{
		{"Monday weekday", "2025-01-27", "weekday"},
		{"Tuesday weekday", "2025-01-28", "weekday"},
		{"Saturday weekend", "2025-01-25", "weekend"},
		{"Sunday weekend", "2025-01-26", "weekend"},
		{"Wednesday holiday (Neujahr)", "2025-01-01", "weekend"},
		{"Monday holiday (Heilige 3 Könige)", "2025-01-06", "weekend"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use GetRulesForDate to test day type indirectly
			rules, err := service.GetRulesForDate(context.Background(), 0, tc.date) // tenantID = 0
			if err != nil {
				t.Fatalf("GetRulesForDate() error = %v", err)
			}

			// Verify rules are for correct day type
			for _, rule := range rules {
				if rule.DayType != tc.want {
					t.Errorf("Expected rules for %s, got rules for %s", tc.want, rule.DayType)
					break
				}
			}
		})
	}
}

// Helper function
func containsTimeSlot(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ========================================
// Phase 7: Performance Testing
// ========================================

// Test 7.1.2: Available Slots Generation Performance
// Purpose: Verify time slot generation is fast
func BenchmarkGetAvailableTimeSlots(b *testing.B) {
	db := testutil.SetupBenchmarkDB(b)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetAvailableTimeSlots(context.Background(), 0, "2025-01-27") // tenantID = 0
	}
}

// Test 7.1.2: Available Slots Generation for Weekend
func BenchmarkGetAvailableTimeSlots_Weekend(b *testing.B) {
	db := testutil.SetupBenchmarkDB(b)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetAvailableTimeSlots(context.Background(), 0, "2025-01-25") // tenantID = 0, Saturday
	}
}

// Test 7.1.3: Booking Validation Performance
// Purpose: Verify booking validation completes quickly
func BenchmarkValidateBookingTime(b *testing.B) {
	db := testutil.SetupBenchmarkDB(b)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.ValidateBookingTime(context.Background(), 0, "2025-01-27", "15:00") // tenantID = 0
	}
}

// Test 7.1.3: Booking Validation with Holiday Check
func BenchmarkValidateBookingTime_WithHolidayCheck(b *testing.B) {
	db := testutil.SetupBenchmarkDB(b)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	// Add some holidays to test holiday check performance
	for i := 1; i <= 50; i++ {
		holiday := &models.CustomHoliday{
			Date:     fmt.Sprintf("2025-%02d-%02d", (i%12)+1, (i%28)+1),
			Name:     fmt.Sprintf("Holiday %d", i),
			IsActive: true,
			Source:   "test",
		}
		_ = holidayRepo.CreateHoliday(0, holiday) // tenantID = 0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.ValidateBookingTime(context.Background(), 0, "2025-01-27", "15:00") // tenantID = 0
	}
}

// Test 7.1.3: Multiple Booking Validations
func TestValidateBookingTime_Performance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	// Validate 100 bookings and measure time
	start := time.Now()
	for i := 0; i < 100; i++ {
		_ = service.ValidateBookingTime(context.Background(), 0, "2025-01-27", "15:00") // tenantID = 0
	}
	elapsed := time.Since(start)

	// Target: < 500ms for 100 validations (5ms per validation)
	if elapsed > 500*time.Millisecond {
		t.Errorf("100 validations took %v, expected < 500ms", elapsed)
	}

	t.Logf("100 booking validations completed in %v (avg: %v per validation)",
		elapsed, elapsed/100)
}

// Test 7.1.2: Available Slots Generation Performance Test
func TestGetAvailableTimeSlots_Performance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, nil)

	// Generate slots 100 times and measure time
	start := time.Now()
	for i := 0; i < 100; i++ {
		_, _ = service.GetAvailableTimeSlots(context.Background(), 0, "2025-01-27") // tenantID = 0
	}
	elapsed := time.Since(start)

	// Target: < 1000ms for 100 generations (10ms per generation)
	if elapsed > 1000*time.Millisecond {
		t.Errorf("100 slot generations took %v, expected < 1000ms", elapsed)
	}

	t.Logf("100 time slot generations completed in %v (avg: %v per generation)",
		elapsed, elapsed/100)
}

// ============ Period-Based Booking Blocking Tests ============

// Test GetPeriodForTime
func TestBookingTimeService_GetPeriodForTime(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, bookingRepo)

	t.Run("returns morning rule for 09:00 on weekday", func(t *testing.T) {
		// Monday 2025-01-27
		rule, err := service.GetPeriodForTime(context.Background(), 0, "2025-01-27", "09:00")
		if err != nil {
			t.Fatalf("GetPeriodForTime() failed: %v", err)
		}
		if rule == nil {
			t.Fatal("Expected to find a rule for 09:00 on weekday")
		}
		// Rule name is in German: "Vormittag" for morning
		if rule.RuleName != "Vormittag" && rule.RuleName != "morning" {
			t.Errorf("Expected rule name 'Vormittag' or 'morning', got '%s'", rule.RuleName)
		}
	})

	t.Run("returns afternoon rule for 14:00", func(t *testing.T) {
		rule, err := service.GetPeriodForTime(context.Background(), 0, "2025-01-27", "14:00")
		if err != nil {
			t.Fatalf("GetPeriodForTime() failed: %v", err)
		}
		if rule == nil {
			t.Fatal("Expected to find a rule for 14:00")
		}
		// Rule name is in German: "Nachmittag" for afternoon
		if rule.RuleName != "Nachmittag" && rule.RuleName != "afternoon" {
			t.Errorf("Expected rule name 'Nachmittag' or 'afternoon', got '%s'", rule.RuleName)
		}
	})

	t.Run("returns nil for time in blocked period (lunch)", func(t *testing.T) {
		// 12:30 is in the blocked lunch period (12:00-14:00)
		rule, err := service.GetPeriodForTime(context.Background(), 0, "2025-01-27", "12:30")
		if err != nil {
			t.Fatalf("GetPeriodForTime() failed: %v", err)
		}
		if rule != nil {
			t.Errorf("Expected nil for blocked period, got rule '%s'", rule.RuleName)
		}
	})

	t.Run("returns nil for time outside any period", func(t *testing.T) {
		// 20:00 is outside all periods
		rule, err := service.GetPeriodForTime(context.Background(), 0, "2025-01-27", "20:00")
		if err != nil {
			t.Fatalf("GetPeriodForTime() failed: %v", err)
		}
		if rule != nil {
			t.Errorf("Expected nil for time outside periods, got rule '%s'", rule.RuleName)
		}
	})

	t.Run("uses weekend rules for Saturday", func(t *testing.T) {
		// Saturday 2025-01-25
		rule, err := service.GetPeriodForTime(context.Background(), 0, "2025-01-25", "09:30")
		if err != nil {
			t.Fatalf("GetPeriodForTime() failed: %v", err)
		}
		if rule == nil {
			t.Fatal("Expected to find a rule for Saturday")
		}
		// Weekend morning should work
	})
}

// Test CheckPeriodAvailability
func TestBookingTimeService_CheckPeriodAvailability(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, bookingRepo)

	// Create test user and dog
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO users (id, tenant_id, first_name, last_name, email, password_hash, is_verified, is_active, terms_accepted_at, last_activity_at, created_at)
		VALUES (1, 0, 'Test', 'User', 'test@example.com', 'hash', 1, 1, ?, ?, ?)
	`, now, now, now)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO dogs (id, tenant_id, name, breed, color_id, is_available, created_at)
		VALUES (1, 0, 'Buddy', 'Labrador', 1, 1, ?)
	`, now)
	if err != nil {
		t.Fatalf("Failed to create test dog: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO dogs (id, tenant_id, name, breed, color_id, is_available, created_at)
		VALUES (2, 0, 'Max', 'German Shepherd', 1, 1, ?)
	`, now)
	if err != nil {
		t.Fatalf("Failed to create test dog 2: %v", err)
	}

	testDate := "2025-01-27" // Monday

	t.Run("available when no existing booking", func(t *testing.T) {
		available, booking, period, err := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate, "09:00")
		if err != nil {
			t.Fatalf("CheckPeriodAvailability() failed: %v", err)
		}
		if !available {
			t.Error("Expected period to be available")
		}
		if booking != nil {
			t.Error("Expected no existing booking")
		}
		if period == nil {
			t.Fatal("Expected period to be returned")
		}
		// Rule name is in German: "Vormittag" for morning
		if period.RuleName != "Vormittag" && period.RuleName != "morning" {
			t.Errorf("Expected period 'Vormittag' or 'morning', got '%s'", period.RuleName)
		}
	})

	t.Run("not available when dog has booking in same period", func(t *testing.T) {
		// Create booking at 10:00 (morning period)
		_, err := db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status, created_at, updated_at)
			VALUES (0, 1, 1, ?, '10:00', 'scheduled', ?, ?)
		`, testDate, now, now)
		if err != nil {
			t.Fatalf("Failed to create booking: %v", err)
		}

		// Try to book at 09:00 (same morning period)
		available, booking, period, err := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate, "09:00")
		if err != nil {
			t.Fatalf("CheckPeriodAvailability() failed: %v", err)
		}
		if available {
			t.Error("Expected period to NOT be available")
		}
		if booking == nil {
			t.Fatal("Expected existing booking to be returned")
		}
		if booking.ScheduledTime != "10:00" {
			t.Errorf("Expected existing booking at 10:00, got %s", booking.ScheduledTime)
		}
		if period == nil {
			t.Fatal("Expected period to be returned")
		}
	})

	t.Run("available for different period", func(t *testing.T) {
		// Dog 1 has booking at 10:00 (morning), check 14:00 (afternoon)
		available, _, _, err := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate, "14:00")
		if err != nil {
			t.Fatalf("CheckPeriodAvailability() failed: %v", err)
		}
		if !available {
			t.Error("Expected afternoon period to be available")
		}
	})

	t.Run("available for different dog", func(t *testing.T) {
		// Dog 1 has booking, check for Dog 2 at same time
		available, _, _, err := service.CheckPeriodAvailability(context.Background(), 0, 2, testDate, "09:00")
		if err != nil {
			t.Fatalf("CheckPeriodAvailability() failed: %v", err)
		}
		if !available {
			t.Error("Expected period to be available for different dog")
		}
	})

	t.Run("tenant isolation - other tenant's booking doesn't affect availability", func(t *testing.T) {
		// Create tenant 2 first (foreign key requirement)
		_, err := db.Exec(`
			INSERT INTO tenants (id, slug, name, contact_email, status, created_at, updated_at)
			VALUES (2, 'tenant2', 'Tenant 2', 'tenant2@test.com', 'active', ?, ?)
		`, now, now)
		if err != nil {
			t.Fatalf("Failed to create tenant 2: %v", err)
		}

		// Create booking time rules for tenant 2 (needed for period lookup)
		_, err = db.Exec(`
			INSERT INTO booking_time_rules (tenant_id, day_type, rule_name, start_time, end_time, is_blocked, created_at, updated_at)
			VALUES (2, 'weekday', 'Vormittag', '08:30', '12:00', 0, ?, ?),
			       (2, 'weekday', 'Nachmittag', '14:00', '17:00', 0, ?, ?)
		`, now, now, now, now)
		if err != nil {
			t.Fatalf("Failed to create booking time rules for tenant 2: %v", err)
		}

		// Create booking for tenant 1
		_, err = db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status, created_at, updated_at)
			VALUES (1, 1, 2, '2025-01-28', '09:00', 'scheduled', ?, ?)
		`, now, now)
		if err != nil {
			t.Fatalf("Failed to create booking for tenant 1: %v", err)
		}

		// Check availability for tenant 2 - should be available (tenant 1's booking doesn't affect tenant 2)
		available, _, _, err := service.CheckPeriodAvailability(context.Background(), 2, 2, "2025-01-28", "10:00")
		if err != nil {
			t.Fatalf("CheckPeriodAvailability() failed: %v", err)
		}
		if !available {
			t.Error("Expected period to be available (tenant isolation)")
		}
	})
}

// Test FilterSlotsForDog
func TestBookingTimeService_FilterSlotsForDog(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, bookingRepo)

	// Create test user and dog
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO users (id, tenant_id, first_name, last_name, email, password_hash, is_verified, is_active, terms_accepted_at, last_activity_at, created_at)
		VALUES (1, 0, 'Test', 'User', 'test@example.com', 'hash', 1, 1, ?, ?, ?)
	`, now, now, now)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO dogs (id, tenant_id, name, breed, color_id, is_available, created_at)
		VALUES (1, 0, 'Buddy', 'Labrador', 1, 1, ?)
	`, now)
	if err != nil {
		t.Fatalf("Failed to create test dog: %v", err)
	}

	testDate := "2025-01-29" // Wednesday
	// Slots aligned with period times: Morning starts at 08:30 (per testutil seed data)
	allSlots := []string{"08:30", "08:45", "09:00", "09:30", "10:00", "11:00", "14:00", "15:00", "16:00"}

	t.Run("returns all slots when no bookings", func(t *testing.T) {
		booked, slots, err := service.FilterSlotsForDog(context.Background(), 0, 1, testDate, allSlots)
		if err != nil {
			t.Fatalf("FilterSlotsForDog() failed: %v", err)
		}
		if len(booked) != 0 {
			t.Errorf("Expected 0 booked periods, got %d", len(booked))
		}
		if len(slots) != len(allSlots) {
			t.Errorf("Expected %d slots, got %d", len(allSlots), len(slots))
		}
	})

	t.Run("filters morning slots when morning is booked", func(t *testing.T) {
		// Create booking in morning period
		_, err := db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status, created_at, updated_at)
			VALUES (0, 1, 1, ?, '09:00', 'scheduled', ?, ?)
		`, testDate, now, now)
		if err != nil {
			t.Fatalf("Failed to create booking: %v", err)
		}

		booked, slots, err := service.FilterSlotsForDog(context.Background(), 0, 1, testDate, allSlots)
		if err != nil {
			t.Fatalf("FilterSlotsForDog() failed: %v", err)
		}
		if len(booked) != 1 {
			t.Errorf("Expected 1 booked period, got %d", len(booked))
		}
		// Rule name is in German: "Vormittag" for morning
		if len(booked) > 0 && booked[0].RuleName != "Vormittag" && booked[0].RuleName != "morning" {
			t.Errorf("Expected booked period 'Vormittag' or 'morning', got '%s'", booked[0].RuleName)
		}

		// Only afternoon slots should remain
		for _, slot := range slots {
			slotTime, _ := time.Parse("15:04", slot)
			morningEnd, _ := time.Parse("15:04", "12:00")
			if slotTime.Before(morningEnd) {
				t.Errorf("Slot %s should have been filtered (morning booked)", slot)
			}
		}
	})

	t.Run("tenant isolation - other tenant bookings don't affect filtering", func(t *testing.T) {
		testDate2 := "2025-01-30"

		// Create tenant 2 first (foreign key requirement)
		_, err := db.Exec(`
			INSERT INTO tenants (id, slug, name, contact_email, status, created_at, updated_at)
			VALUES (2, 'tenant2', 'Tenant 2', 'tenant2@test.com', 'active', ?, ?)
		`, now, now)
		if err != nil {
			t.Fatalf("Failed to create tenant 2: %v", err)
		}

		// Create booking time rules for tenant 2 (needed for period lookup)
		_, err = db.Exec(`
			INSERT INTO booking_time_rules (tenant_id, day_type, rule_name, start_time, end_time, is_blocked, created_at, updated_at)
			VALUES (2, 'weekday', 'Vormittag', '08:30', '12:00', 0, ?, ?),
			       (2, 'weekday', 'Nachmittag', '14:00', '17:00', 0, ?, ?)
		`, now, now, now, now)
		if err != nil {
			t.Fatalf("Failed to create booking time rules for tenant 2: %v", err)
		}

		// Create booking for tenant 1
		_, err = db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status, created_at, updated_at)
			VALUES (1, 1, 1, ?, '09:00', 'scheduled', ?, ?)
		`, testDate2, now, now)
		if err != nil {
			t.Fatalf("Failed to create booking: %v", err)
		}

		// Filter for tenant 2 - should return all slots (tenant 1's booking doesn't affect tenant 2)
		booked, slots, err := service.FilterSlotsForDog(context.Background(), 2, 1, testDate2, allSlots)
		if err != nil {
			t.Fatalf("FilterSlotsForDog() failed: %v", err)
		}
		if len(booked) != 0 {
			t.Errorf("Expected 0 booked periods for tenant 2 (isolation), got %d", len(booked))
		}
		if len(slots) != len(allSlots) {
			t.Errorf("Expected all slots for tenant 2, got %d", len(slots))
		}
	})
}

// Test CheckPeriodAvailability with buffer time enforcement
func TestBookingTimeService_CheckPeriodAvailability_BufferTime(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, bookingRepo)

	// Create test user and dog
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO users (id, tenant_id, first_name, last_name, email, password_hash, is_verified, is_active, terms_accepted_at, last_activity_at, created_at)
		VALUES (1, 0, 'Test', 'User', 'test@example.com', 'hash', 1, 1, ?, ?, ?)
	`, now, now, now)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO dogs (id, tenant_id, name, breed, color_id, is_available, created_at)
		VALUES (1, 0, 'Buddy', 'Labrador', 1, 1, ?)
	`, now)
	if err != nil {
		t.Fatalf("Failed to create test dog: %v", err)
	}

	testDate := "2025-01-27" // Monday (weekday)

	t.Run("allows booking at exact period start time (08:30)", func(t *testing.T) {
		// Period starts at 08:30, booking at 08:30 should be allowed
		available, _, period, err := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate, "08:30")
		if err != nil {
			t.Fatalf("CheckPeriodAvailability() failed: %v", err)
		}
		if !available {
			t.Error("Expected booking at exact period start (08:30) to be available")
		}
		if period == nil {
			t.Error("Expected period to be returned")
		}
	})

	t.Run("allows booking well before period end", func(t *testing.T) {
		// Period ends at 12:00, booking at 10:00 should be allowed (2 hours before end)
		available, _, _, err := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate, "10:00")
		if err != nil {
			t.Fatalf("CheckPeriodAvailability() failed: %v", err)
		}
		if !available {
			t.Error("Expected booking at 10:00 to be available (well before period end)")
		}
	})

	t.Run("rejects booking too close to period end (buffer time)", func(t *testing.T) {
		// Period ends at 12:00, buffer is 30 minutes, so 11:45 should be rejected
		available, _, _, err := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate, "11:45")
		if err == nil {
			t.Error("Expected error for booking too close to period end")
		}
		if available {
			t.Error("Expected booking at 11:45 to be rejected (within 30-min buffer of 12:00)")
		}
		// Error should mention the buffer time
		if err != nil && !strings.Contains(err.Error(), "30 Minuten") {
			t.Errorf("Expected error to mention buffer time, got: %v", err)
		}
	})

	t.Run("allows booking exactly at buffer cutoff", func(t *testing.T) {
		// Period ends at 12:00, buffer is 30 minutes, so 11:30 should be allowed (exactly at cutoff)
		available, _, _, err := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate, "11:30")
		if err != nil {
			t.Fatalf("CheckPeriodAvailability() failed: %v", err)
		}
		if !available {
			t.Error("Expected booking at 11:30 to be available (exactly at buffer cutoff)")
		}
	})

	t.Run("afternoon period also has buffer time", func(t *testing.T) {
		// Afternoon period ends at 17:00, so 16:45 should be rejected
		available, _, _, err := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate, "16:45")
		if err == nil {
			t.Error("Expected error for afternoon booking too close to period end")
		}
		if available {
			t.Error("Expected booking at 16:45 to be rejected (within 30-min buffer of 17:00)")
		}
	})
}

// Test moving booking across periods on the same day
func TestBookingTimeService_CheckPeriodAvailability_CrossPeriod(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo, bookingRepo)

	// Create test user and dog
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO users (id, tenant_id, first_name, last_name, email, password_hash, is_verified, is_active, terms_accepted_at, last_activity_at, created_at)
		VALUES (1, 0, 'Test', 'User', 'test@example.com', 'hash', 1, 1, ?, ?, ?)
	`, now, now, now)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO dogs (id, tenant_id, name, breed, color_id, is_available, created_at)
		VALUES (1, 0, 'Buddy', 'Labrador', 1, 1, ?)
	`, now)
	if err != nil {
		t.Fatalf("Failed to create test dog: %v", err)
	}

	testDate := "2025-01-27" // Monday

	t.Run("moving from morning to afternoon is allowed", func(t *testing.T) {
		// Create a morning booking at 09:00
		_, err := db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status, created_at, updated_at)
			VALUES (0, 1, 1, ?, '09:00', 'scheduled', ?, ?)
		`, testDate, now, now)
		if err != nil {
			t.Fatalf("Failed to create booking: %v", err)
		}

		// Check if afternoon (14:00) is available for the same dog
		available, existingBooking, period, err := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate, "14:00")
		if err != nil {
			t.Fatalf("CheckPeriodAvailability() failed: %v", err)
		}
		if !available {
			t.Error("Expected afternoon to be available even though morning is booked")
		}
		if existingBooking != nil {
			t.Error("Expected no existing booking in afternoon period")
		}
		if period == nil || (period.RuleName != "Nachmittag" && period.RuleName != "afternoon") {
			t.Errorf("Expected afternoon period, got: %v", period)
		}
	})

	t.Run("morning blocked does not affect afternoon availability", func(t *testing.T) {
		testDate2 := "2025-01-28"
		// Create a morning booking
		_, err := db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status, created_at, updated_at)
			VALUES (0, 1, 1, ?, '10:00', 'scheduled', ?, ?)
		`, testDate2, now, now)
		if err != nil {
			t.Fatalf("Failed to create booking: %v", err)
		}

		// Morning should be blocked
		availableMorning, _, _, _ := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate2, "09:00")
		if availableMorning {
			t.Error("Expected morning to be blocked")
		}

		// Afternoon should still be available
		availableAfternoon, _, _, err := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate2, "15:00")
		if err != nil {
			t.Fatalf("CheckPeriodAvailability() failed: %v", err)
		}
		if !availableAfternoon {
			t.Error("Expected afternoon to be available")
		}
	})

	t.Run("both periods blocked when both have bookings", func(t *testing.T) {
		testDate3 := "2025-01-29"
		// Create morning booking
		_, err := db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status, created_at, updated_at)
			VALUES (0, 1, 1, ?, '09:00', 'scheduled', ?, ?)
		`, testDate3, now, now)
		if err != nil {
			t.Fatalf("Failed to create morning booking: %v", err)
		}

		// Create afternoon booking
		_, err = db.Exec(`
			INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status, created_at, updated_at)
			VALUES (0, 1, 1, ?, '14:00', 'scheduled', ?, ?)
		`, testDate3, now, now)
		if err != nil {
			t.Fatalf("Failed to create afternoon booking: %v", err)
		}

		// Morning should be blocked
		availableMorning, _, _, _ := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate3, "10:00")
		if availableMorning {
			t.Error("Expected morning to be blocked")
		}

		// Afternoon should also be blocked
		availableAfternoon, _, _, _ := service.CheckPeriodAvailability(context.Background(), 0, 1, testDate3, "15:00")
		if availableAfternoon {
			t.Error("Expected afternoon to be blocked")
		}
	})
}
